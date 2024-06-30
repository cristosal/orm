package orm

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/cristosal/orm/schema"
)

type (
	// PaginationOptions for configuring paginate query
	PaginationOptions struct {
		Query         string
		QueryColumns  []string
		Page          int
		PageSize      int
		SortBy        string
		SortDirection string
	}

	// PaginationResults contain results and stats from pagination query
	PaginationResults struct {
		Total   int64
		Page    int
		Start   int
		End     int
		HasNext bool
	}
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

// Paginate returns paginated data for T
func Paginate[T any](db DB, v *[]T, opts *PaginationOptions) (*PaginationResults, error) {
	if opts == nil {
		opts = &PaginationOptions{
			Page:     1,
			PageSize: 25,
		}
	}

	var t T
	sch, _, err := schema.GetMapping(&t)
	if err != nil {
		return nil, err
	}

	sqlstr := ""
	if opts.queryable() {
		var parts []string
		for _, col := range opts.QueryColumns {
			parts = append(parts, fmt.Sprintf("%s LIKE $1", col))
		}

		likeClause := strings.Join(parts, " OR ")
		sqlstr = fmt.Sprintf("WHERE %s", likeClause)
	}

	CountWhere(sch.Table, sqlstr)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", sch.Table, sqlstr)
	var row *sql.Row

	if opts.queryable() {
		row = db.QueryRow(countQuery, opts.sqlQueryParam())
	} else {
		row = db.QueryRow(countQuery)
	}

	if row == nil {
		return nil, sql.ErrNoRows
	}

	var count int64
	if err := row.Scan(&count); err != nil {
		return nil, err
	}

	if opts.sortable() {
		sqlstr = fmt.Sprintf("%s ORDER BY %s %s", sqlstr, opts.SortBy, opts.SortDirection)
	}

	offset := opts.Page * opts.PageSize
	sqlstr = fmt.Sprintf("%s LIMIT %d OFFSET %d", sqlstr, opts.PageSize, offset)

	if opts.queryable() {
		err = List(db, v, sqlstr, opts.sqlQueryParam())
	} else {
		err = List(db, v, sqlstr)
	}

	if err != nil {
		return nil, err
	}

	hasNext := (int64(opts.Page)*int64(opts.PageSize))+int64(opts.PageSize) < count

	results := &PaginationResults{
		Start:   offset + 1,
		End:     min((opts.Page+1)*opts.PageSize, int(count)),
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
