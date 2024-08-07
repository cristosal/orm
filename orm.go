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
	ORM struct{ DB }

	// DB interface allows for interoperability between sql.Tx and sql.DB types
	DB interface {
		Beginner
		QuerierExecuter
	}

	Beginner interface {
		Begin() (*sql.Tx, error)
	}

	Executer interface {
		Exec(sql string, args ...any) (sql.Result, error)
	}

	Querier interface {
		Query(sql string, args ...any) (*sql.Rows, error)
		QueryRow(sql string, args ...any) *sql.Row
	}

	QuerierExecuter interface {
		Querier
		Executer
	}

	Rows interface {
		Next() bool
		Close() error
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

func Open(driverName, dataSourceName string) (*ORM, error) {
	sqlDB, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	return New(sqlDB), nil
}

func New(db DB) *ORM { return &ORM{db} }

// Exec executes the sql string returning any error encountered
func (o *ORM) Exec(sql string, args ...any) error { return Exec(o.DB, sql, args...) }

// Exec executes the sql string returning any error encountered
func Exec(db Executer, sql string, args ...any) error {
	_, err := db.Exec(sql, args...)
	return err
}

// Query executes an sql statement and scans the result set into v
func (o *ORM) Query(v any, sql string, args ...any) error {
	return Query(o.DB, v, sql, args...)
}

// Query executes an sql statement and scans the result set into v
func Query(db Querier, v any, sql string, args ...any) error {
	mapping, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	slice := reflect.ValueOf(v).Elem()

	rows, err := db.Query(sql, args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		row := reflect.New(mapping.Type)
		if err := Scan(rows, row.Interface()); err != nil {
			return err
		}

		slice.Set(reflect.Append(slice, row.Elem()))
	}

	return nil
}

// List is a select over columns defined in v
func (o *ORM) ListWhere(v any, sql string, args ...any) error {
	return ListWhere(o.DB, v, sql, args...)
}

// List is a select over columns defined in v
func ListWhere(db Querier, v any, sql string, args ...any) error {
	return List(db, v, "WHERE "+sql, args...)
}

// List is a select over columns defined in v
func (o *ORM) List(v any, sql string, args ...any) error {
	return List(o.DB, v, sql, args...)
}

// List is a select over columns defined in v
func List(db Querier, v any, sql string, args ...any) error {
	mapping, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	var (
		val  = reflect.ValueOf(v).Elem()
		cols = mapping.Fields.Columns().List()
		sql2 = fmt.Sprintf("SELECT %s FROM %s %s", cols, mapping.Table, sql)
	)

	rows, err := db.Query(strings.Trim(sql2, " "), args...)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		row := reflect.New(mapping.Type)

		if err := Scan(rows, row.Interface()); err != nil {
			return err
		}

		val.Set(reflect.Append(val, row.Elem()))
	}

	return nil
}

func (o *ORM) QueryRow(v any, sql string, args ...any) error {
	return QueryRow(o.DB, v, sql, args...)
}

// QueryRow executes a given sql query and scans the result into v
func QueryRow(db Querier, v any, sql string, args ...any) error {
	_, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	row := db.QueryRow(sql, args...)
	return Scan(row, v)
}

func (o *ORM) GetWhere(v any, sql string, args ...any) error {
	return Get(o.DB, v, "WHERE "+sql, args...)
}

func GetString(db Querier, sql string, args ...any) (string, error) {
	row := db.QueryRow(sql, args...)
	var str string
	err := row.Scan(&str)
	return str, err
}

func (o *ORM) Get(v any, sql string, args ...any) error {
	return Get(o.DB, v, sql, args...)
}

func GetWhere(db Querier, v any, s string, args ...any) error {
	return Get(db, v, "WHERE "+s, args...)
}

// Get returns the first row encountered.
// The sql string is placed immediately after the SELECT statement.
func Get(db Querier, v any, s string, args ...any) error {
	sch, _, err := schema.GetMapping(v)
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

	return Scan(row, v)
}

func (o *ORM) GetByID(v any) error {
	return GetByID(o.DB, v)
}

func GetByID(db Querier, v any) error {
	sch, _, err := schema.GetMapping(v)
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

func (o *ORM) ListAll(v any) error {
	return ListAll(o.DB, v)
}

// ListAll is the same as List with an empty sql string.
// It will return all rows from the table deduced by v and is equivalent to a select from table.
func ListAll(db Querier, v any) error {
	return List(db, v, "")
}

func (o *ORM) Add(v any) error {
	return Add(o.DB, v)
}

// Add inserts v into designated table. ID is set on v if available
func Add(db QuerierExecuter, v any) error {
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	// get the writeable columns
	cols := sch.Fields.Writeable().Columns()

	// if it was an array we would neet to iterate over it
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", sch.Table, cols.List(), cols.ValueList(1))

	vals, err := schema.Values(v)
	if err != nil {
		return err
	}

	// the below only happens in postgres (returning clause)
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

func (o *ORM) AddMany(v any) error {
	return AddMany(o.DB, v)
}

func AddMany(db Executer, v any) error {
	slice := reflect.ValueOf(v)

	if slice.Kind() != reflect.Slice {
		return ErrInvalidType
	}

	// ok now what so now we need to find out if we can loop it and get address
	mapping, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	var (
		columns    = mapping.Fields.Writeable().Columns()
		counter    = 1
		valueParts []string
		args       []any
	)

	for i := 0; i < slice.Len(); i++ {
		record := slice.Index(i)
		vals, err := schema.Values(record.Interface())
		if err != nil {
			return err
		}

		args = append(args, vals...)
		valueParts = append(valueParts, "("+columns.ValueList(counter)+")")
		counter += columns.Len()
	}

	// lets not do a returning
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		mapping.Table, columns.List(), strings.Join(valueParts, ", "))

	return Exec(db, sql, args...)
}

func (o *ORM) DropTable(v any) error {
	return DropTable(o.DB, v)
}

func DropTable(db Executer, v any) error {
	if str, ok := v.(string); ok {
		s := fmt.Sprintf("DROP TABLE %s", str)
		return Exec(db, s)
	}

	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	s := fmt.Sprintf("DROP TABLE %s", sch.Table)
	return Exec(db, s)
}

func (o *ORM) Remove(v any, sql string, args ...any) error {
	return Remove(o.DB, v, sql, args...)
}

func (o *ORM) RemoveWhere(v any, sql string, args ...any) error {
	return RemoveWhere(o.DB, v, sql, args...)
}

func RemoveWhere(db Executer, v any, s string, args ...any) error {
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}
	sqlstr := fmt.Sprintf("DELETE FROM %s WHERE %s", sch.Table, s)
	return Exec(db, sqlstr, args...)
}

func Remove(db Executer, v any, s string, args ...any) error {
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}
	sqlstr := fmt.Sprintf("DELETE FROM %s %s", sch.Table, s)
	return Exec(db, sqlstr, args...)
}

func (o *ORM) RemoveByID(v any) error {
	return RemoveByID(o.DB, v)
}

func RemoveByID(db Executer, v any) error {
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	f, index, err := sch.Fields.FindPK()
	if err != nil {
		return err
	}

	var (
		val = getValueAtIndex(v, index)
		sql = fmt.Sprintf("DELETE FROM %s WHERE %s = $1", sch.Table, f.Column)
	)

	return Exec(db, sql, val)
}

func (o *ORM) UpdateWhere(v any, sql string, args ...any) error {
	return UpdateWhere(o.DB, v, sql, args...)
}

func UpdateWhere(db Executer, v any, sql string, args ...any) error {
	return Update(db, v, "WHERE "+sql, args...)
}

func (o *ORM) Update(v any, sql string, args ...any) error {
	return Update(o.DB, v, sql, args...)
}

func Update(db Executer, v any, sql string, args ...any) error {
	start := len(args) + 1
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return err
	}

	assignments := sch.Fields.Writeable().Columns().AssignmentList(start)
	s := fmt.Sprintf("UPDATE %s SET %s", sch.Table, assignments)
	if sql != "" {
		s = fmt.Sprintf("%s %s", s, sql)
	}

	values, err := schema.Values(v)
	if err != nil {
		return err
	}

	args = append(args, values...)
	return Exec(db, s, args...)
}

// UpdateByID sets values by the identity. If no id is found, UpdateByID return ErrNoIdentity
func (o *ORM) UpdateByID(v any) error {
	return UpdateByID(o.DB, v)
}

// UpdateByID sets values by the identity. If no id is found, UpdateByID return ErrNoIdentity
func UpdateByID(db Executer, v any) error {
	sch, _, err := schema.GetMapping(v)
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
	values, err := schema.Values(v)
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

func (o *ORM) CountAll(v any) (int64, error) {
	return CountAll(o.DB, v)
}

func CountAll(q Querier, v any) (int64, error) {
	return Count(q, v, "")
}

func (o *ORM) CountWhere(v any, sql string, args ...any) (count int64, err error) {
	return Count(o.DB, v, "WHERE "+sql, args...)
}

func (o *ORM) Count(v any, sql string, args ...any) (count int64, err error) {
	return Count(o.DB, v, sql, args...)
}

func CountWhere(q Querier, v any, sql string, args ...any) (count int64, err error) {
	return Count(q, v, "WHERE "+sql, args...)
}

func Count(q Querier, v any, sql string, args ...any) (count int64, err error) {
	var sqlstr string

	if tbl, ok := v.(string); ok {
		sqlstr = fmt.Sprintf("SELECT COUNT(*) FROM %s", tbl)
	} else {
		sch, _, err := schema.GetMapping(v)
		if err != nil {
			return 0, err
		}

		sqlstr = fmt.Sprintf("SELECT COUNT(*) FROM %s", sch.Table)
	}

	if sql != "" {
		sqlstr = sqlstr + " " + sql
	}

	row := q.QueryRow(sqlstr, args...)
	if row == nil {
		err = ErrNotFound
		return
	}

	if err = row.Err(); err != nil {
		return
	}

	err = row.Scan(&count)
	return
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
		if err := Scan(rows, &t); err != nil {
			return nil, err
		}
		items = append(items, t)
	}

	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}

	return
}

// TableName returns the deduced table name for v.
// Returns empty string if deduction fails
func TableName(v any) string {
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return ""
	}
	return sch.Table
}

// Columns allows for performing actions
func Columns(v any) schema.Columns {
	sch, _, err := schema.GetMapping(v)
	if err != nil {
		return []string{}
	}

	return sch.Fields.Columns()
}

// Scan scans the row to value
func Scan(row Row, v any) error {
	vals, err := schema.Addrs(v)
	if err != nil {
		return err
	}

	return row.Scan(vals...)
}

// gets the address of the struct value at a given index
func getAddrAtIndex(v any, index []int) interface{} {
	return reflect.ValueOf(v).Elem().FieldByIndex(index).Addr().Interface()
}

// gets the concrete value at given index
func getValueAtIndex(v any, index []int) interface{} {
	return reflect.ValueOf(v).Elem().FieldByIndex(index).Interface()
}
