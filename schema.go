// Package db contains functions that help with PGX
package pgxx

import (
	"errors"
	"reflect"
	"strings"
)

var (
	ErrInvalidType = errors.New("invalid type")
	ErrNoIdentity  = errors.New("identity not found")
	cache          = make(map[string]*Schema)
)

type (
	// Schema contains the database mapping information for a given type
	Schema struct {
		Parent *Schema      // Not nil if schema represents an embeded type
		Table  string       // Database table name
		Type   reflect.Type // Underlying reflect type
		Fields Fields       // Field maps
		Embeds []Child      // Embeded schemas
	}

	// Child represents the schema for an embeded struct
	Child struct {
		Index  int
		Schema Schema
	}

	// Fields faciliatates collection methods over fields
	Fields []Field

	// Field contains mapping information between struct field and database column
	Field struct {
		Name       string      // Name of the field in the struct
		Column     string      // Name of the database column
		Index      int         // Index of the field within a struct
		Identity   bool        // Is an ID field
		ReadOnly   bool        // Is only for select queries
		ForeignKey *ForeignKey // Foreign key meta data
	}

	// ForeignKey Field Metadata
	ForeignKey struct {
		Table  string // Foreign table name
		Column string // Foreign table column
	}
)

// IsRoot is true when the schema is not embeded
func (s *Schema) IsRoot() bool { return s.Parent == nil }

// Children returns embeded schemas
func (s *Schema) Children() []Schema {
	var children []Schema
	for _, e := range s.Embeds {
		children = append(children, e.Schema)
	}
	return children
}

// Analyze returns a schema representing the mapping between the go type and database row.
// Schemas are cached by table name so as not to repeat analisis unnecesarily.
func Analyze(v interface{}) (sch *Schema, err error) {
	var table string

	rec, isRecord := v.(Record)
	if isRecord {
		table = rec.TableName()
		if cache[table] != nil {
			return cache[table], nil
		}
	}

	typ, val, err := infer(v)
	if err != nil {
		return
	}

	if table == "" {
		table = snakecase(typ.Name())
	}

	if cache[table] != nil {
		return cache[table], nil
	}

	sch = new(Schema)
	sch.Table = table
	sch.Type = typ
	sch.Embeds = make([]Child, 0)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// handle embedded structs
		if field.Anonymous && field.IsExported() {
			embeded, err := Analyze(val.Field(i).Interface())
			embeded.Parent = sch
			if err == nil {
				sch.Embeds = append(sch.Embeds, Child{
					Index:  i,
					Schema: *embeded,
				})
			}

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

				info.ForeignKey = &ForeignKey{
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

	cache[table] = sch
	return
}

// MustAnalyze panics if schema analisis fails. See Analyze for further information
func MustAnalyze(v interface{}) *Schema {
	res, err := Analyze(v)
	if err != nil {
		panic(err)
	}
	return res
}

func (sch *Schema) Columns() Columns {
	var cols Columns
	cols = append(cols, sch.Fields.Columns()...)
	for i := range sch.Embeds {
		cols = append(cols, sch.Embeds[i].Schema.Columns()...)
	}

	return cols
}

func (fields Fields) Identity() (*Field, error) {
	for _, f := range fields {
		if f.Identity {
			return &f, nil
		}
	}
	return nil, ErrNoIdentity
}

func (fields Fields) ForeignKeys() Fields {
	info := []Field{}
	for _, f := range fields {
		if f.ForeignKey != nil {
			info = append(info, f)
		}
	}
	return info
}

func (fields Fields) Writeable() Fields {
	var f Fields
	for i := range fields {
		if fields[i].ReadOnly || fields[i].Identity {
			continue
		}

		fields = append(fields, fields[i])
	}
	return f
}

func (fields Fields) Columns() (cols Columns) {
	for _, f := range fields {
		cols = append(cols, f.Column)
	}
	return
}

// ScanValues returns all scannable values from a given struct.
func (sch *Schema) ScanValues(v interface{}) (values []any, err error) {
	if err != nil {
		return nil, err
	}

	sv, err := inferValue(v)
	if err != nil {
		return nil, err
	}

	// loop through our fields and get the ptr to value
	for _, f := range sch.Fields {
		v := sv.Field(f.Index)
		values = append(values, v.Addr().Interface())
	}

	// recursive over embeded schemas
	for _, e := range sch.Embeds {
		v := sv.Field(e.Index)
		vals, err := sch.ScanValues(v.Addr().Interface())
		if err != nil {
			return nil, err
		}

		values = append(values, vals...)
	}

	return values, nil
}

func writeableValues(v interface{}) (values []any, err error) {
	sch, err := Analyze(v)
	if err != nil {
		return nil, err
	}

	sv, err := inferValue(v)
	if err != nil {
		return nil, err
	}

	for _, field := range sch.Fields.Writeable() {
		f := sv.Field(field.Index)
		switch f.Kind() {
		case reflect.Pointer,
			reflect.Map,
			reflect.Interface,
			reflect.Slice,
			reflect.Func,
			reflect.Chan:
			if f.IsNil() {
				values = append(values, nil)
			} else {
				values = append(values, f.Interface())
			}
		default:
			values = append(values, f.Interface())
		}
	}

	for _, e := range sch.Embeds {
		vals, err := writeableValues(sv.Field(e.Index).Interface())
		if err != nil {
			return nil, err
		}

		values = append(values, vals...)
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
	case reflect.Slice, reflect.Pointer:
		typ = typ.Elem()
		val = val.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return
	}

	err = nil
	return
}
