package pgxx

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type (
	PaginationOptions struct {
		Record        Record
		Query         string
		QueryColumns  []string
		Page          int
		PageSize      int
		SortBy        string
		SortDirection SortDirection
	}

	PaginationResults[T any] struct {
		Total   int64
		Items   []T
		Page    int
		Start   int
		End     int
		HasNext bool
	}

	SortDirection = string
)

const (
	Ascending  = SortDirection("asc")
	Descending = SortDirection("desc")
)

func (opts *PaginationOptions) queryable() bool {
	return opts.Query != "" && opts.QueryColumns != nil && len(opts.QueryColumns) > 0
}

func (opts *PaginationOptions) sortable() bool {
	return opts.SortBy != "" && opts.SortDirection != ""
}

func (opts *PaginationOptions) sqlQueryParam() string {
	return "%" + opts.Query + "%"
}

func Paginate[T any](a Interface, opts *PaginationOptions) (*PaginationResults[T], error) {
	sql := ""

	if opts.queryable() {
		var parts []string
		for _, col := range opts.QueryColumns {
			parts = append(parts, fmt.Sprintf("%s like $1", col))
		}

		likeClause := strings.Join(parts, " or ")
		sql = fmt.Sprintf("where %s", likeClause)
	}

	// count
	countq := fmt.Sprintf("select count(*) from %s %s", opts.Record.TableName(), sql)
	var row pgx.Row

	if opts.queryable() {
		row = a.QueryRow(ctx, countq, opts.sqlQueryParam())
	} else {
		row = a.QueryRow(ctx, countq)
	}
	var count int64
	if err := row.Scan(&count); err != nil {
		return nil, err
	}

	if opts.sortable() {
		sql = fmt.Sprintf("%s order by %s %s", sql, opts.SortBy, opts.SortDirection)
	}

	offset := opts.Page * opts.PageSize
	sql = fmt.Sprintf("%s limit %d offset %d", sql, opts.PageSize, offset)

	var rows Rows
	var err error

	if opts.queryable() {
		rows, err = Select(a, opts.Record, sql, opts.sqlQueryParam())
	} else {
		rows, err = Select(a, opts.Record, sql)
	}

	if err != nil {
		return nil, err
	}

	items, err := CollectRows[T](rows)
	if err != nil {
		return nil, err
	}

	hasNext := (int64(opts.Page)*int64(opts.PageSize))+int64(opts.PageSize) < count

	results := &PaginationResults[T]{
		Start:   offset + 1,
		End:     min((opts.Page+1)*opts.PageSize, int(count)),
		Items:   items,
		Page:    opts.Page,
		Total:   count,
		HasNext: hasNext,
	}

	return results, nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
