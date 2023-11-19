package dbx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	ErrInvalidType = errors.New("invalid type")
	ErrNoIdentity  = errors.New("identity not found")
	cache          = make(map[string]*Schema)
	cacheMtx       = new(sync.RWMutex)
)

func getSchema(key string) *Schema {
	cacheMtx.RLock()
	defer cacheMtx.RUnlock()
	return cache[key]
}

func setSchema(key string, s *Schema) {
	cacheMtx.Lock()
	defer cacheMtx.Unlock()
	cache[key] = s
}

type (
	// Schema contains the database mapping information for a given type
	Schema struct {
		Parent *Schema      // Not nil if schema represents an embeded type
		Table  string       // Database table name
		Type   reflect.Type // Underlying reflect type
		Fields Fields       // Field maps
	}

	// Fields faciliatates collection methods over fields
	Fields []Field

	// Field contains mapping information between struct field and database column
	Field struct {
		Name     string  // Name of the field in the struct
		Column   string  // Name of the database column
		Index    int     // Index of the field within a struct
		Identity bool    // Is an ID field
		ReadOnly bool    // Is only for select queries
		FK       *FK     // Foreign key meta data
		Schema   *Schema // Embeded schema
	}

	// FK represents foreign key field metadata
	FK struct {
		Table  string // Foreign table name
		Column string // Foreign table column
	}
)

// IsRoot is true when the schema is not embeded
func (s *Schema) IsRoot() bool { return s.Parent == nil }

// HasSchema returns true when the field contains an embeded schema
func (f *Field) HasSchema() bool { return f.Schema != nil }

// IsWriteable is true when the fields value can be included in an insert or update statement
func (f *Field) IsWriteable() bool { return !f.ReadOnly && !f.Identity }

// ClearCache clears the schema cache
func ClearCache() {
	cache = make(map[string]*Schema)
}

// Analyze returns a schema representing the mapping between the go type and database row.
// Schemas are cached by table name so as not to repeat analisis unnecesarily.
func Analyze(v interface{}) (sch *Schema, err error) {
	var table string

	rec, isRecord := v.(Record)
	if isRecord {
		table = rec.TableName()
		if getSchema(table) != nil {
			return getSchema(table), nil
		}
	}

	typ, val, err := infer(v)
	if err != nil {
		return
	}

	if table == "" {
		table = snakecase(typ.Name())
	}

	if getSchema(table) != nil {
		return getSchema(table), nil
	}

	sch = new(Schema)
	sch.Table = table
	sch.Type = typ

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if field.Anonymous && field.IsExported() {
			embeded, _ := Analyze(val.Field(i).Interface())
			embeded.Parent = sch

			sch.Fields = append(sch.Fields, Field{
				Name:   field.Name,
				Index:  i,
				Schema: embeded,
			})
			continue
		}

		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		if dbTag == "" {
			col := snakecase(field.Name)
			finfo := Field{
				Name:     field.Name,
				Column:   col,
				Index:    i,
				Identity: col == "id",
				ReadOnly: col == "id",
			}

			sch.Fields = append(sch.Fields, finfo)
			continue
		}

		parts := strings.Split(dbTag, ",")
		column := strings.Trim(parts[0], " ")
		info := Field{
			Name:     field.Name,
			Index:    i,
			Column:   column,
			Identity: column == "id",
			ReadOnly: column == "id",
		}

		for i, part := range parts {
			if i == 0 {
				if part == "id" {
					info.Identity = true
					info.ReadOnly = true
				}
				continue
			}

			part = strings.Trim(part, " ")

			// check for foreign key
			if strings.HasPrefix(part, "fk=") {
				val := strings.Trim(strings.TrimPrefix(part, "fk="), " ")
				parts := strings.Split(val, ".")
				if len(parts) != 2 {
					continue
				}

				info.FK = &FK{
					Table:  parts[0],
					Column: parts[1],
				}
			} else {
				switch part {
				case "pk":
					info.Identity = true
				case "ro", "readonly":
					info.ReadOnly = true
				}
			}
		}

		sch.Fields = append(sch.Fields, info)
	}

	setSchema(table, sch)
	return
}

// MustAnalyze panics if schema analisis fails. See Analyze for further information
func MustAnalyze(v interface{}) *Schema {
	sch, err := Analyze(v)
	if err != nil {
		panic(err)
	}
	return sch
}

// Identity returns the first identity field found
func (fields Fields) Identity() (*Field, []int, error) {
	var index []int

	for _, field := range fields {
		if field.Identity {
			index = append(index, field.Index)
			return &field, index, nil
		}

		// returns the first field with the identity
		if field.HasSchema() {
			f, indexes, _ := field.Schema.Fields.Identity()
			if f != nil {
				index = append(index, field.Index)
				index = append(index, indexes...)
				return f, index, nil
			}
		}
	}
	return nil, index, ErrNoIdentity
}

// ForeignKeys are fields representing foreign keys
func (fields Fields) ForeignKeys() Fields {
	info := []Field{}
	for _, f := range fields {
		if f.FK != nil {
			info = append(info, f)
		}
	}
	return info
}

// Writeable returns all writeable fields
// A field is writeable if it is not marked as readonly or is an identity field
func (fields Fields) Writeable() Fields {
	var ret Fields

	for _, field := range fields {
		if !field.IsWriteable() {
			continue
		}

		if field.HasSchema() {
			fs := field.Schema.Fields.Writeable()
			ret = append(ret, fs...)
			continue
		}

		ret = append(ret, field)
	}

	return ret
}

// Columns returns all database columns for the given fields
// it goes recursively through fields
func (fields Fields) Columns() (columns Columns) {
	for _, f := range fields {
		if f.HasSchema() {
			columns = append(columns, f.Schema.Fields.Columns()...)
			continue
		}

		columns = append(columns, f.Column)
	}
	return
}

// scanableValues returns all scannable values from a given struct.
func scanableValues(v interface{}) (values []any, err error) {
	sch, err := Analyze(v)
	if err != nil {
		return nil, err
	}

	sv, err := inferValue(v)
	if err != nil {
		return nil, err
	}

	for _, f := range sch.Fields {
		v := sv.Field(f.Index)

		if f.HasSchema() {
			recursive, err := scanableValues(v.Addr().Interface())
			if err != nil {
				return nil, err
			}

			values = append(values, recursive...)
			continue
		}

		values = append(values, v.Addr().Interface())
	}

	return values, nil
}

// writeableValues returns the value from struct fields not marked as readonly
func writeableValues(v interface{}) (values []any, err error) {
	sch, err := Analyze(v)
	if err != nil {
		return nil, err
	}

	sv, err := inferValue(v)
	if err != nil {
		return nil, err
	}

	for _, field := range sch.Fields {
		if !field.IsWriteable() {
			continue
		}

		v := sv.Field(field.Index)

		// recursively analyze the schema
		if field.HasSchema() {
			vals, _ := writeableValues(v.Interface())
			values = append(values, vals...)
			continue
		}

		switch v.Kind() {
		case reflect.Pointer,
			reflect.Map,
			reflect.Interface,
			reflect.Slice,
			reflect.Func,
			reflect.Chan:
			if v.IsNil() {
				values = append(values, nil)
			} else {
				values = append(values, v.Interface())
			}
		default:
			values = append(values, v.Interface())
		}
	}
	return
}

func inferValue(v interface{}) (val reflect.Value, err error) {
	defer func() {
		_ = recover()
	}()
	val = reflect.ValueOf(v)
	err = ErrInvalidType

	switch val.Kind() {
	case reflect.Slice, reflect.Pointer:
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	err = nil
	return
}

func infer(v interface{}) (typ reflect.Type, val reflect.Value, err error) {
	defer func() { _ = recover() }()
	val = reflect.ValueOf(v)
	typ = val.Type()
	err = ErrInvalidType

	switch typ.Kind() {
	case reflect.Interface:
		return infer(val.Elem().Interface())
	case reflect.Slice:
		typ = typ.Elem()
	case reflect.Pointer:
		typ = typ.Elem()
		val = val.Elem()

		// was pointer to interface
		if typ.Kind() == reflect.Interface {
			typ = typ.Elem()
			val = val.Elem()
		}

		// can be pointer to slice
		if typ.Kind() == reflect.Slice {
			return infer(val.Interface())
		}
	}

	if typ.Kind() != reflect.Struct {
		err = fmt.Errorf("%w: %s", ErrInvalidType, typ.Kind().String())
		return
	}

	err = nil
	return
}
