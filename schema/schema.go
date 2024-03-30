package schema

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

var (
	ErrInvalidType   = errors.New("invalid type")
	ErrFieldNotFound = errors.New("field not found")
	schemaCache      = make(map[string]*StructMapping)
	schemaCacheMtx   = new(sync.RWMutex)
)

// Record is implemented by structs which wish to override the default table name
type Record interface {
	TableName() string
}

// StructMapping contains the database mapping information for a given type
type StructMapping struct {
	Parent *StructMapping // Not nil if schema represents an embeded type
	Table  string         // Table name in databse
	Type   reflect.Type   // Underlying reflect type
	Fields FieldMappings  // Field mappings
}

// IsRoot is true when the schema is not embeded
func (s *StructMapping) IsRoot() bool {
	return s.Parent == nil
}

// GetMapping returns a mapping between the go type and database row.
// Results cached by table name so as not to repeat analysis unnecesarily.
func GetMapping(v any) (mapping *StructMapping, err error) {
	var table string

	typ, val, err := Reflect(v)
	if err != nil {
		return
	}

	// check if the value implements Record interface
	if record, ok := val.Interface().(Record); ok {
		if sch := Lookup(record.TableName()); sch != nil {
			return sch, nil
		}
	}

	if table == "" {
		table = snakecase(typ.Name())
	}

	if mapping := Lookup(table); mapping != nil {
		return mapping, nil
	}

	// parse the mapping
	mapping = &StructMapping{
		Table: table,
		Type:  typ,
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if field.Anonymous && field.IsExported() {
			embeded, _ := GetMapping(val.Field(i).Interface())
			embeded.Parent = mapping

			mapping.Fields = append(mapping.Fields, FieldMapping{
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
			finfo := FieldMapping{
				Name:         field.Name,
				Column:       col,
				Index:        i,
				IsPrimaryKey: col == "id",
				IsReadOnly:   col == "id",
			}

			mapping.Fields = append(mapping.Fields, finfo)
			continue
		}

		parts := strings.Split(dbTag, ",")
		column := strings.Trim(parts[0], " ")
		info := FieldMapping{
			Name:         field.Name,
			Index:        i,
			Column:       column,
			IsPrimaryKey: column == "id",
			IsReadOnly:   column == "id",
		}

		for i, part := range parts {
			if i == 0 {
				if part == "id" || part == "pk" {
					info.IsPrimaryKey = true
					info.IsReadOnly = true
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
				// TODO: add other cases for db tags here
				case "ro", "readonly":
					info.IsReadOnly = true
				}
			}
		}

		mapping.Fields = append(mapping.Fields, info)
	}

	SaveMapping(table, mapping)
	return
}

// MustGet panics if Get fails. See Get for further information
func MustGet(v any) *StructMapping {
	mapping, err := GetMapping(v)
	if err != nil {
		panic(err)
	}

	return mapping
}

// ClearCache clears the schema cache
func ClearCache() {
	schemaCache = make(map[string]*StructMapping)
}

// Addrs returns all scannable values from a given struct.
func Addrs(v interface{}) (values []any, err error) {
	mapping, err := GetMapping(v)
	if err != nil {
		return nil, err
	}

	sv, err := inferValue(v)
	if err != nil {
		return nil, err
	}

	for _, f := range mapping.Fields {
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
	sch, err := GetMapping(v)
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

// Reflect infers the underlying reflection struct type and reflection value.
// If the v is a slice the reflect value will be a new zerod value of the slice type
func Reflect(v any) (typ reflect.Type, val reflect.Value, err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER %v", err)
		}
	}()

	val = reflect.ValueOf(v)
	typ = val.Type()

	// Initialize error in order to return this error when function panics
	err = ErrInvalidType

	switch typ.Kind() {
	case reflect.Interface:
		return Reflect(val.Elem().Interface())
	case reflect.Slice:
		typ = typ.Elem()
		val = reflect.New(typ).Elem()
	case reflect.Pointer:
		typ = typ.Elem()
		val = val.Elem()

		// v is a pointer to interface
		if typ.Kind() == reflect.Interface {
			typ = typ.Elem()
			val = val.Elem()
			err = nil
		}

		// v is a pointer to slice
		if typ.Kind() == reflect.Slice {
			return Reflect(val.Interface())
		}
	}

	// assert that the underlying type is a struct
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

func Lookup(key string) *StructMapping {
	schemaCacheMtx.RLock()
	defer schemaCacheMtx.RUnlock()
	return schemaCache[key]
}

func SaveMapping(key string, s *StructMapping) {
	schemaCacheMtx.Lock()
	defer schemaCacheMtx.Unlock()
	schemaCache[key] = s
}
