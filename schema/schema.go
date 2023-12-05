package schema

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

var (
	ErrInvalidType   = errors.New("invalid type")
	ErrFieldNotFound = errors.New("field not found")
	schemaCache      = make(map[string]*Schema)
	schemaCacheMtx   = new(sync.RWMutex)
)

type (
	// Schema contains the database mapping information for a given type
	Schema struct {
		Parent *Schema      // Not nil if schema represents an embeded type
		Table  string       // Table name in databse
		Type   reflect.Type // Underlying reflect type
		Fields Fields       // Field mappings
	}

	// Record is implemented by structs which wish to override the default table name
	Record interface {
		TableName() string
	}
)

// IsRoot is true when the schema is not embeded
func (s *Schema) IsRoot() bool { return s.Parent == nil }

// Get returns a schema representing the mapping between the go type and database row.
// Schemas are cached by table name so as not to repeat analisis unnecesarily.
func Get(v interface{}) (sch *Schema, err error) {
	var table string
	rec, isRecord := v.(Record)
	if isRecord {
		table = rec.TableName()
		sch := lookup(table)
		if sch != nil {
			return sch, nil
		}
	}

	typ, val, err := infer(v)
	if err != nil {
		return
	}

	if table == "" {
		table = snakecase(typ.Name())
	}

	if lookup(table) != nil {
		return lookup(table), nil
	}

	sch = new(Schema)
	sch.Table = table
	sch.Type = typ

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if field.Anonymous && field.IsExported() {
			embeded, _ := Get(val.Field(i).Interface())
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
				PK:       col == "id",
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
			PK:       column == "id",
			ReadOnly: column == "id",
		}

		for i, part := range parts {
			if i == 0 {
				if part == "id" || part == "pk" {
					info.PK = true
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
				// TODO: add other cases for db tags here
				case "ro", "readonly":
					info.ReadOnly = true
				}
			}
		}

		sch.Fields = append(sch.Fields, info)
	}

	save(table, sch)
	return
}

// MustGet panics if Get fails. See Get for further information
func MustGet(v interface{}) *Schema {
	sch, err := Get(v)
	if err != nil {
		panic(err)
	}

	return sch
}

// ClearCache clears the schema cache
func ClearCache() {
	schemaCache = make(map[string]*Schema)
}

// Addrs returns all scannable values from a given struct.
func Addrs(v interface{}) (values []any, err error) {
	sch, err := Get(v)
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
			recursive, err := Addrs(v.Addr().Interface())
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

// Values returns the values from struct fields not marked as readonly
func Values(v interface{}) (values []any, err error) {
	sch, err := Get(v)
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
			vals, _ := Values(v.Interface())
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

func lookup(key string) *Schema {
	schemaCacheMtx.RLock()
	defer schemaCacheMtx.RUnlock()
	return schemaCache[key]
}

func save(key string, s *Schema) {
	schemaCacheMtx.Lock()
	defer schemaCacheMtx.Unlock()
	schemaCache[key] = s
}
