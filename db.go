package pgxx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	sqlfs                fs.FS
	ctx                  = context.Background()
	ErrNoForeignKeys     = errors.New("no foreign keys were found")
	ErrNoForeignKeyMatch = errors.New("no foreign key matched")
	ErrNotFound          = fmt.Errorf("not found: %w", pgx.ErrNoRows)
)

type (
	// Interface is depended on for services that need to use connection or pool methods
	Interface interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
		Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	}

	ID int64

	scanable interface{ Scaned() }
	scanner  interface{ Scan(dest ...any) error }

	Record interface{ TableName() string }

	Rows interface {
		Close()
		Next() bool
		scanner
	}
)

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// SetSQLDir root for running scripts
func SetSQLDir(dir fs.FS) {
	sqlfs = dir
}

func ParseID(str string) (ID, error) {
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return ID(id), nil
}

func InnerJoin[T any](r Interface, v1, v2 any, sql string, args ...any) ([]T, error) {
	r1, err := Analyze(v1)
	if err != nil {
		return nil, err
	}

	rep2, err := Analyze(v2)
	if err != nil {
		return nil, err
	}

	fks := rep2.Fields.ForeignKeys()

	if len(fks) == 0 {
		return nil, ErrNoForeignKeys
	}

	var match *Field
	for _, fk := range fks {
		if fk.ForeignKey.Table == r1.Table {
			match = &fk
		}
	}

	if match == nil {
		return nil, ErrNoForeignKeyMatch
	}

	cols := r1.Fields.Columns().PrefixedList("a")
	sql = fmt.Sprintf("select %s from %s a inner join %s b on a.%s = b.%s %s", cols, r1.Table, rep2.Table, match.ForeignKey.Column, match.Column, sql)
	sql = strings.Trim(sql, " ") // in case empty string was passed as argument
	rows, err := r.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return CollectRows[T](rows)
}

func Seed[T any](a Interface, items []T) error {
	var t T
	sch, err := Analyze(&t)
	if err != nil {
		return err
	}

	var (
		cols   = sch.Fields.Writeable().Columns()
		ncols  = len(cols)
		nitems = len(items)
		values []any
		list   []string
	)

	for i := 0; i < nitems; i++ {
		list = append(list, fmt.Sprintf("(%s)", cols.ValueList((i+1)*ncols)))
		vals, err := writeableValues(items[i])
		if err != nil {
			return err
		}
		values = append(values, vals...)
	}

	sql := fmt.Sprintf("insert into %s (%s) values %s on conflict do nothing", sch.Table, cols.List(), strings.Join(list, ", "))
	return Exec(a, sql, values...)
}

func Exec(a Interface, sql string, args ...any) error {
	_, err := a.Exec(context.Background(), sql, args...)
	return err
}

func SelectMany[T any](adpt Interface, v []T, sql string, args ...any) error {
	var (
		ctx    = context.Background()
		result = MustAnalyze(v)
		cols   = result.Columns().List()
		sql2   = fmt.Sprintf("select %s from %s %s", cols, result.Table, sql)
	)

	rows, err := adpt.Query(ctx, strings.Trim(sql2, " "), args...)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var r T
		if err := Scan(rows, &r); err != nil {
			return err
		}
		v = append(v, r)
	}
	return nil
}

func One(a Interface, v Record, sql string, args ...any) error {
	cols := MustAnalyze(v).Fields.Columns().List()
	table := v.TableName()
	q := fmt.Sprintf("select %s from %s %s", cols, table, sql)
	row := a.QueryRow(context.Background(), q, args...)
	return Scan(row, v)
}

func SelectOne(a Interface, v Record, sql string, args ...any) scanner {
	cols := MustAnalyze(v).Fields.Columns().List()
	table := v.TableName()
	q := fmt.Sprintf("select %s from %s %s", cols, table, sql)
	return a.QueryRow(context.Background(), q, args...)
}

func Select(a Interface, v Record, sql string, args ...any) (Rows, error) {
	rep := MustAnalyze(v)
	cols := rep.Fields.Columns().List()
	table := v.TableName()
	q := fmt.Sprintf("select %s from %s %s", cols, table, sql)
	return a.Query(context.Background(), q, args...)
}

func All(a Interface, v Record) (Rows, error) {
	cols := MustAnalyze(v).Fields.Columns().List()
	table := v.TableName()
	q := fmt.Sprintf("select %s from %s", cols, table)
	return a.Query(context.Background(), q)
}

func Query[T any](a Interface, sql string, args ...any) ([]T, error) {
	rows, err := a.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}

	return CollectRows[T](rows)
}

func Insert(db Interface, v any) error {
	sql, vals, err := insertQ(v)
	if err != nil {
		return err
	}

	sch := MustAnalyze(v)

	// we also should check if we have identity field within the struct and get the address to the interface
	id, err := sch.Fields.Identity()

	if errors.Is(err, ErrNoIdentity) {
		return Exec(db, sql, vals...)
	}

	if err != nil {
		return err
	}

	row := db.QueryRow(context.Background(), fmt.Sprintf("%s returning %s", sql, id.Column), vals...)
	addr := reflect.ValueOf(v).Elem().Field(id.Index).Addr().Interface()
	return row.Scan(addr)
}

func insertQ(v any) (sql string, values []any, err error) {
	sch, err := Analyze(v)
	if err != nil {
		return
	}

	var cols = sch.Fields.Writeable().Columns()
	for _, e := range sch.Embeds {
		cols = append(cols, e.Schema.Fields.Writeable().Columns()...)
	}

	var (
		list  = cols.List()
		vlist = cols.ValueList(1)
	)

	sql = fmt.Sprintf("insert into %s (%s) values (%s)", sch.Table, list, vlist)

	// need to handle write values...
	values, err = writeableValues(v)
	return
}

func Update(a Interface, r any) error {
	sql, vals, err := updateQ(r)
	if err != nil {
		return err
	}

	_, err = a.Exec(context.Background(), sql, vals...)
	return err
}

func updateQ(r any) (sql string, values []any, err error) {
	sch, err := Analyze(r)
	if err != nil {
		return
	}

	idField, err := sch.Fields.Identity()
	if err != nil {
		return
	}

	cols := sch.Fields.Writeable().Columns()

	placeholders := cols.AssignmentList(1)
	sql = fmt.Sprintf("update %s set %s", sch.Table, placeholders)
	values, err = writeableValues(r)
	sv := reflect.ValueOf(r).Elem()
	f := sv.Field(idField.Index)
	id := f.Int()
	sql += fmt.Sprintf(" where %s = $%d", idField.Column, len(cols)+1)
	values = append(values, id)
	return
}

func RunScript(conn Interface, script string, tdata any) error {
	var fpath = script
	var t *template.Template
	var err error

	if sqlfs != nil {
		t, err = template.ParseFS(sqlfs, script)
		if err != nil {
			return err
		}
	} else {
		t, err = template.ParseFiles(fpath)
		if err != nil {
			return err
		}

	}

	b := new(strings.Builder)
	if err := t.Execute(b, tdata); err != nil {
		return err
	}

	return Exec(conn, b.String())
}

func CollectStrings(rows Rows) ([]string, error) {
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	if ids == nil {
		return nil, ErrNotFound
	}

	return ids, nil

}

func CollectIDs(rows Rows) ([]ID, error) {
	defer rows.Close()
	var ids []ID
	for rows.Next() {
		var id ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	if ids == nil {
		return nil, ErrNotFound
	}

	return ids, nil
}

func CollectRows[T any](rows Rows) (items []T, err error) {
	defer rows.Close()
	for rows.Next() {
		var t T
		if err := Scan(rows, &t); err != nil {
			return nil, err
		}
		items = append(items, t)
	}

	if len(items) == 0 {
		return nil, ErrNotFound
	}

	return
}

func Scan(s scanner, v any) error {
	sch, err := Analyze(v)
	if err != nil {
		return err
	}

	vals, err := sch.ScanValues(v)
	if err != nil {
		return err
	}

	err = s.Scan(vals...)
	if err != nil {
		return err
	}

	if scannable, ok := v.(scanable); ok {
		scannable.Scaned()
	}

	return nil
}

func snakecase(input string) string {
	var (
		buf       bytes.Buffer
		prevUpper = false
		nextLower = false
	)

	for i, c := range input {
		// need to check if next character is lower case
		if i == len(input)-1 {
			nextLower = false
		} else {
			nextLower = unicode.IsLower(rune(input[i+1]))
		}

		if unicode.IsUpper(c) {
			if i > 0 && !prevUpper {
				buf.WriteRune('_')
			}

			if nextLower && prevUpper {
				buf.WriteRune('_')
			}

			buf.WriteRune(unicode.ToLower(c))
			prevUpper = true
		} else {
			prevUpper = false
			buf.WriteRune(c)
		}
	}

	return buf.String()
}
