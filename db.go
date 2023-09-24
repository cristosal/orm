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
	// DB represents the underlying pgx connection. It can be a single conn, pool or transaction
	DB interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
		Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	}

	// ID represents a serial id
	ID int64

	// Record is the interface implemented by structs that want to explicitly specify their table name
	Record interface{ TableName() string }

	scanable interface{ Scaned() }

	scanner interface{ Scan(dest ...any) error }
)

// String is the string representation of a serial id
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// SetScriptFS sets the underlying fs for reading sql scripts with RunScript
func SetScriptFS(dir fs.FS) {
	sqlfs = dir
}

// ParseID attempts to parse str into postgres serial id
func ParseID(str string) (ID, error) {
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return ID(id), nil
}

// Exec executes a given sql statement and returns any error encountered
func Exec(db DB, sql string, args ...any) error {
	_, err := db.Exec(context.Background(), sql, args...)
	return err
}

// Many returns all rows encountered that satisfy the sql condition.
// The sql string is placed immediately after the select statement
func Many[T any](db DB, v *[]T, sql string, args ...any) error {
	var t T
	schema, err := Analyze(&t)
	if err != nil {
		return err
	}

	var (
		ctx  = context.Background()
		cols = schema.Fields.Columns().List()
		sql2 = fmt.Sprintf("select %s from %s %s", cols, schema.Table, sql)
	)

	rows, err := db.Query(ctx, strings.Trim(sql2, " "), args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		// generics are need for instantiation here
		var row T
		if err := scan(rows, &row); err != nil {
			return err
		}
		*v = append(*v, row)
	}

	return nil
}

// One returns the first row encountered that satisfies the sql condition.
// The sql string is placed immediately after the select statement
func One(db DB, v any, sql string, args ...any) error {
	sch, err := Analyze(v)
	if err != nil {
		return err
	}

	cols := sch.Fields.Columns().List()
	q := fmt.Sprintf("select %s from %s %s", cols, sch.Table, sql)
	row := db.QueryRow(context.Background(), q, args...)
	return scan(row, v)
}

// First returns the first row encountered for a given table.
// It is equivalent to One with an empty sql string
func First(db DB, v any) error {
	return One(db, v, "")
}

// All is the same as Many with an empty sql string.
// It will return all rows from the table deduced by v and is equivalent to a select from table.
func All[T any](db DB, v *[]T) error {
	return Many(db, v, "")
}

// Insert inserts v into it's designated table. ID is set on v if available
func Insert(db DB, v any) error {
	sch, err := Analyze(v)
	if err != nil {
		return err
	}

	var cols = sch.Fields.Writeable().Columns()
	sql := fmt.Sprintf("insert into %s (%s) values (%s)", sch.Table, cols.List(), cols.ValueList(1))

	vals, err := WriteableValues(v)
	if err != nil {
		return err
	}

	id, index, err := sch.Fields.Identity()
	if errors.Is(err, ErrNoIdentity) {
		return Exec(db, sql, vals...)
	}

	if err != nil {
		return err
	}

	sql = fmt.Sprintf("%s returning %s", sql, id.Column)
	row := db.QueryRow(ctx, sql, vals...)
	addr := reflect.ValueOf(v).Elem().FieldByIndex(index).Addr().Interface()
	return row.Scan(addr)
}

// Update updates v by its identity (ID field). If no id is found, Update return ErrNoIdentity
func Update(db DB, v any) error {
	sch, err := Analyze(v)
	if err != nil {
		return err
	}

	idField, indexes, err := sch.Fields.Identity()
	if err != nil {
		return err
	}

	cols := sch.Fields.Writeable().Columns()

	placeholders := cols.AssignmentList(1)
	sql := fmt.Sprintf("update %s set %s", sch.Table, placeholders)
	values, err := WriteableValues(v)
	if err != nil {
		return err
	}

	sv := reflect.ValueOf(v).Elem()
	f := sv.FieldByIndex(indexes)
	id := f.Int()
	sql += fmt.Sprintf(" where %s = $%d", idField.Column, len(cols)+1)
	values = append(values, id)
	_, err = db.Exec(context.Background(), sql, values...)
	return err
}

// RunScript executes a script from the underlying fs set using SetScriptFS.
// scripts are run as template so it is possible to pass data onto the scripts
func RunScript(conn DB, script string, tdata any) error {
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

// CollectStrings scans rows for string values
func CollectStrings(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	var strs []string
	for rows.Next() {
		var str string
		if err := rows.Scan(&str); err != nil {
			return nil, err
		}

		strs = append(strs, str)
	}

	if strs == nil {
		return nil, ErrNotFound
	}

	return strs, nil

}

// CollectIDs scans each row for id value
func CollectIDs(rows pgx.Rows) ([]ID, error) {
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

// CollectRows scans a T from each row
func CollectRows[T any](rows pgx.Rows) (items []T, err error) {
	defer rows.Close()
	for rows.Next() {
		var t T
		if err := scan(rows, &t); err != nil {
			return nil, err
		}
		items = append(items, t)
	}

	if len(items) == 0 {
		return nil, ErrNotFound
	}

	return
}

func scan(s scanner, v any) error {
	vals, err := ScanableValues(v)
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
