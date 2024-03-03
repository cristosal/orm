package qb

import (
	"fmt"
	"strings"
	"time"
)

type Columns = []*ColumnDefinition

func Statements(stringers ...fmt.Stringer) string {
	var stmts []string
	for _, str := range stringers {
		stmts = append(stmts, str.String())
	}

	return strings.Join(stmts, ";\n")
}

func NowFunc() any { return time.Now() }

func defineColumn(name, typ string) *ColumnDefinition {
	return &ColumnDefinition{
		name: name,
		typ:  typ,
	}

}

type ColumnDefinition struct {
	name             string // name of given column
	typ              string // type of given column can be int or varchar etc..
	length           int    // length for defining the int
	primaryKey       bool   // is primary key
	unique           bool
	notNull          bool              // is not nullable
	references       *ColumnDefinition // foreign key, means that Column Definition
	defaultValue     any
	defaultValueFunc func() any
}

func (cd *ColumnDefinition) String() string {
	parts := []string{cd.name}

	switch cd.typ {
	case "VARCHAR", "CHAR":
		parts = append(parts, fmt.Sprintf("%s(%d)", cd.typ, cd.length))
	default:
		parts = append(parts, cd.typ)
	}

	if cd.primaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	if cd.notNull {
		parts = append(parts, "NOT NULL")
	}

	if cd.unique {
		parts = append(parts, "UNIQUE")
	}

	var v any

	if cd.defaultValueFunc != nil {
		v = cd.defaultValueFunc()
	} else if cd.defaultValue != nil {
		v = cd.defaultValue
	}

	// check if v is not nill
	if v != nil {
		switch v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("DEFAULT '%s'", v))
		default:
			parts = append(parts, fmt.Sprintf("DEFAULT %v", v))
		}
	}

	return strings.Join(parts, " ")
}

type ForeignKeyDefinition struct {
	column          string
	referenceColumn string
	referenceTable  string
	onDeleteAction  string
	onUpdateAction  string
}

func (fk *ForeignKeyDefinition) String() string {
	str := fmt.Sprintf("FOREIGN KEY(%s) REFERENCES %s(%s)", fk.column, fk.referenceTable, fk.referenceColumn)
	if fk.onDeleteAction != "" {
		str = str + " ON DELETE " + fk.onDeleteAction
	}
	if fk.onUpdateAction != "" {
		str = str + " ON UPDATE " + fk.onUpdateAction

	}

	return str
}

func ForeignKey(name, referenceTable, referenceColumn string) *ForeignKeyDefinition {
	return &ForeignKeyDefinition{
		column:          name,
		referenceColumn: referenceColumn,
		referenceTable:  referenceTable,
	}
}

func (fk *ForeignKeyDefinition) OnDelete(action string) *ForeignKeyDefinition {
	fk.onDeleteAction = action
	return fk
}

func (fk *ForeignKeyDefinition) OnUpdate(action string) *ForeignKeyDefinition {
	fk.onUpdateAction = action
	return fk
}

func SmallInt(name string) *ColumnDefinition {
	return defineColumn(name, "SMALLINT")
}

func Integer(name string) *ColumnDefinition {
	return defineColumn(name, "INTEGER")
}

func BigInt(name string) *ColumnDefinition {
	return defineColumn(name, "BIGINT")
}

func JSON(name string) *ColumnDefinition {
	return defineColumn(name, "JSON")
}

func JSONB(name string) *ColumnDefinition {
	return defineColumn(name, "JSONB")
}

func Text(name string) *ColumnDefinition {
	return defineColumn(name, "TEXT")
}

func Char(name string, length int) *ColumnDefinition {
	col := defineColumn(name, "CHAR")
	col.length = length
	return col
}

func Varchar(name string, length int) *ColumnDefinition {
	col := defineColumn(name, "VARCHAR")
	col.length = length
	return col
}

func Boolean(name string) *ColumnDefinition {
	return defineColumn(name, "BOOLEAN")
}

func Serial(name string) *ColumnDefinition {
	return defineColumn(name, "SERIAL")
}

func Time(name string) *ColumnDefinition {
	return defineColumn(name, "TIME")
}

func Interval(name string) *ColumnDefinition {
	return defineColumn(name, "INTERVAL")
}

func Date(name string) *ColumnDefinition {
	return defineColumn(name, "DATE")
}

func String(name string) *ColumnDefinition {
	return Varchar(name, 255)
}

func Timestamp(name string) *ColumnDefinition {
	return defineColumn(name, "TIMESTAMP")
}

func TimestampTZ(name string) *ColumnDefinition {
	return defineColumn(name, "TIMESTAMPTZ")
}

func (cd *ColumnDefinition) References(table string, col *ColumnDefinition) {
	cd.references = col
}

func (cd *ColumnDefinition) NotNull() *ColumnDefinition {
	cd.notNull = true
	return cd
}

func (cd *ColumnDefinition) PrimaryKey() *ColumnDefinition {
	cd.primaryKey = true
	return cd
}

func (cd *ColumnDefinition) Unique() *ColumnDefinition {
	cd.unique = true
	return cd
}

func (cd *ColumnDefinition) Default(v any) *ColumnDefinition {
	cd.defaultValue = v
	return cd
}

func (cd *ColumnDefinition) DefaultFunc(fn func() any) *ColumnDefinition {
	cd.defaultValueFunc = fn
	return cd
}
