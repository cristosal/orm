package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cristosal/orm/schema"
)

type (
	// DB interface allows for interoperability between sql.Tx and sql.DB types
	DB interface {
		Begin() (*sql.Tx, error)
		Exec(sql string, args ...any) (sql.Result, error)
		Query(sql string, args ...any) (*sql.Rows, error)
		QueryRow(sql string, args ...any) *sql.Row
	}

	Rows interface {
		Next() bool
		Close()
		Scan(...any) error
	}

	Row interface {
		Scan(...any) error
	}
)

var (
	// ErrNotFound is an alias for sql.ErrNoRows
	ErrNotFound = sql.ErrNoRows

	// ErrInvalidType is an alias for schema.ErrInvalidType.
	// The error occurs when the interface passed in as the v argument of an orm func is invalid.
	// Most orm funcs accept either a pointer to a struct, or pointer to a slice of structs.
	ErrInvalidType = schema.ErrInvalidType
)

// Exec executes the sql string returning any error encountered
func Exec(db DB, sql string, args ...any) error {
	_, err := db.Exec(sql, args...)
	return err
}

// Query executes an sql statement and scans the result set into v
func Query[T any](db DB, v *[]T, sql string, args ...any) error {
	var t T
	_, err := schema.Get(&t)
	if err != nil {
		return err
	}

	rows, err := db.Query(sql, args...)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var row T

		if err := scanRow(rows, &row); err != nil {
			return err
		}

		*v = append(*v, row)
	}

	return nil
}

// List is a select over columns defined in v
func List[T any](db DB, v *[]T, sql string, args ...any) error {
	var t T
	schema, err := schema.Get(&t)
	if err != nil {
		return err
	}

	var (
		cols = schema.Fields.Columns().List()
		sql2 = fmt.Sprintf("SELECT %s FROM %s %s", cols, schema.Table, sql)
	)

	rows, err := db.Query(strings.Trim(sql2, " "), args...)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		// generics are need for instantiation here
		var row T
		if err := scanRow(rows, &row); err != nil {
			return err
		}

		*v = append(*v, row)
	}

	return nil
}

// QueryRow executes a given sql query and scans the result into v
func QueryRow(db DB, v any, sql string, args ...any) error {
	_, err := schema.Get(v)
	if err != nil {
		return err
	}

	row := db.QueryRow(sql, args...)
	return scanRow(row, v)
}

// Get returns the first row encountered.
// The sql string is placed immediately after the SELECT statement.
func Get(db DB, v any, s string, args ...any) error {
	sch, err := schema.Get(v)
	if err != nil {
		return err
	}

	cols := sch.Fields.Columns().List()

	q := fmt.Sprintf("SELECT %s FROM %s", cols, sch.Table)

	// append sql argument if not empty
	if s != "" {
		q = fmt.Sprintf("%s %s", q, s)
	}

	row := db.QueryRow(q, args...)
	if row == nil {
		return ErrNotFound
	}

	if row.Err() != nil {
		return row.Err()
	}

	return scanRow(row, v)
}

func GetByID(db DB, v any) error {
	sch, err := schema.Get(v)
	if err != nil {
		return err
	}

	f, index, err := sch.Fields.FindPK()
	if err != nil {
		return err
	}

	val := getValueAtIndex(v, index)
	col := f.Column

	return Get(db, v, fmt.Sprintf("WHERE %s = $1", col), val)
}

// ListAll is the same as List with an empty sql string.
// It will return all rows from the table deduced by v and is equivalent to a select from table.
func ListAll[T any](db DB, v *[]T) error {
	return List(db, v, "")
}

// Add inserts v into designated table. ID is set on v if available
func Add(db DB, v any) error {
	sch, err := schema.Get(v)
	if err != nil {
		return err
	}

	var cols = sch.Fields.Writeable().Columns()
	sql := fmt.Sprintf("insert into %s (%s) values (%s)", sch.Table, cols.List(), cols.ValueList(1))
	vals, err := schema.GetWriteableValues(v)
	if err != nil {
		return err
	}

	id, index, err := sch.Fields.FindPK()
	if errors.Is(err, schema.ErrFieldNotFound) {
		return Exec(db, sql, vals...)
	}

	// other possible error
	if err != nil {
		return err
	}

	sql = fmt.Sprintf("%s returning %s", sql, id.Column)
	row := db.QueryRow(sql, vals...)
	addr := getAddrAtIndex(v, index)
	return row.Scan(addr)
}

func Update(db DB, v any, sql string, args ...any) error {
	start := len(args) + 1
	sch, err := schema.Get(v)
	if err != nil {
		return err
	}

	assignments := sch.Fields.Writeable().Columns().AssignmentList(start)
	s := fmt.Sprintf("UPDATE %s SET %s", sch.Table, assignments)
	if sql != "" {
		s = fmt.Sprintf("%s %s", s, sql)
	}

	values, err := schema.GetWriteableValues(v)
	if err != nil {
		return err
	}

	args = append(args, values...)
	return Exec(db, s, args...)
}

// UpdateByID sets values by the identity. If no id is found, UpdateByID return ErrNoIdentity
func UpdateByID(db DB, v any) error {
	sch, err := schema.Get(v)
	if err != nil {
		return err
	}

	idField, indexes, err := sch.Fields.FindPK()
	if err != nil {
		return err
	}

	cols := sch.Fields.Writeable().Columns()
	placeholders := cols.AssignmentList(1)
	sql := fmt.Sprintf("update %s set %s", sch.Table, placeholders)
	values, err := schema.GetWriteableValues(v)
	if err != nil {
		return err
	}

	sv := reflect.ValueOf(v).Elem()
	f := sv.FieldByIndex(indexes)
	id := f.Int()
	sql += fmt.Sprintf(" where %s = $%d", idField.Column, len(cols)+1)
	values = append(values, id)
	return Exec(db, sql, values...)
}

// CollectStrings scans rows for string values
func CollectStrings(rows Rows) ([]string, error) {
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
		return nil, sql.ErrNoRows
	}

	return strs, nil
}

// CollectRows scans a T from each row
func CollectRows[T any](rows Rows) (items []T, err error) {
	defer rows.Close()
	for rows.Next() {
		var t T
		if err := scanRow(rows, &t); err != nil {
			return nil, err
		}
		items = append(items, t)
	}

	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}

	return
}

// gets the address of the struct value at a given index
func getAddrAtIndex(v any, index []int) interface{} {
	return reflect.ValueOf(v).Elem().FieldByIndex(index).Addr().Interface()
}

// gets the concrete value at given index
func getValueAtIndex(v any, index []int) interface{} {
	return reflect.ValueOf(v).Elem().FieldByIndex(index).Interface()
}

func scanRow(row Row, v any) error {
	vals, err := schema.GetScanableValues(v)
	if err != nil {
		return err
	}

	return row.Scan(vals...)
}
