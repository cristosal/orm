package schema

import (
	"fmt"
	"strings"
	"time"
)

func NowFunc() any {
	return time.Now()
}

type DropTableAction struct {
	tableName string
}

func (action *DropTableAction) String() string {
	return "DROP TABLE " + action.tableName
}

type CreateTableAction struct {
	ifNotExists     bool
	tableName       string
	tableDefinition TableDefinition
}

func (action *CreateTableAction) String() string {
	var lines []string
	for _, cd := range action.tableDefinition.columns {
		lines = append(lines, cd.String())
	}

	for _, fk := range action.tableDefinition.foreignKeys {
		lines = append(lines, fk.String())

	}

	return fmt.Sprintf("CREATE TABLE %s (%s)", action.tableName, strings.Join(lines, ", "))
}

type TableDefinition struct {
	tableName   string
	columns     []*ColumnDefinition
	foreignKeys []*ForeignKeyDefinition
}

func (t *TableDefinition) appendColumn(name, typ string) *ColumnDefinition {
	cd := &ColumnDefinition{
		name: name,
		typ:  typ,
	}

	t.columns = append(t.columns, cd)
	return cd
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

func DropTable(tableName string) *DropTableAction {
	return &DropTableAction{
		tableName: tableName,
	}
}

func CreateTableIfNotExists(tableName string, fn func(t *TableDefinition)) *CreateTableAction {
	action := CreateTableAction{
		tableName:   tableName,
		ifNotExists: true,
		tableDefinition: TableDefinition{
			tableName: tableName,
		},
	}

	fn(&action.tableDefinition)
	return &action
}

func CreateTable(tableName string, fn func(t *TableDefinition)) *CreateTableAction {
	action := CreateTableAction{
		tableName: tableName,
		tableDefinition: TableDefinition{
			tableName: tableName,
		},
	}

	fn(&action.tableDefinition)
	return &action
}

func (td *TableDefinition) Foreign(name, referenceTable, referenceColumn string) *ForeignKeyDefinition {
	fk := &ForeignKeyDefinition{
		column:          name,
		referenceColumn: referenceColumn,
		referenceTable:  referenceTable,
	}

	td.foreignKeys = append(td.foreignKeys, fk)
	return fk
}

func (fk *ForeignKeyDefinition) OnDelete(action string) *ForeignKeyDefinition {
	fk.onDeleteAction = action
	return fk
}

func (fk *ForeignKeyDefinition) OnUpdate(action string) *ForeignKeyDefinition {
	fk.onUpdateAction = action
	return fk
}

func (t *TableDefinition) SmallInt(name string) *ColumnDefinition {
	return t.appendColumn(name, "SMALLINT")
}

func (t *TableDefinition) Integer(name string) *ColumnDefinition {
	return t.appendColumn(name, "INTEGER")
}

func (t *TableDefinition) BigInt(name string) *ColumnDefinition {
	return t.appendColumn(name, "BIGINT")
}

func (t *TableDefinition) JSON(name string) *ColumnDefinition {
	return t.appendColumn(name, "JSON")
}

func (t *TableDefinition) JSONB(name string) *ColumnDefinition {
	return t.appendColumn(name, "JSONB")
}

func (t *TableDefinition) Text(name string) *ColumnDefinition {
	return t.appendColumn(name, "TEXT")
}

func (t *TableDefinition) Char(name string, length int) *ColumnDefinition {
	col := t.appendColumn(name, "CHAR")
	col.length = length
	return col
}

func (t *TableDefinition) Varchar(name string, length int) *ColumnDefinition {
	col := t.appendColumn(name, "VARCHAR")
	col.length = length
	return col
}

func (t *TableDefinition) Boolean(name string) *ColumnDefinition {
	return t.appendColumn(name, "BOOLEAN")
}

func (t *TableDefinition) Serial(name string) *ColumnDefinition {
	return t.appendColumn(name, "SERIAL")
}

func (t *TableDefinition) Time(name string) *ColumnDefinition {
	return t.appendColumn(name, "TIME")
}

func (t *TableDefinition) Interval(name string) *ColumnDefinition {
	return t.appendColumn(name, "INTERVAL")
}

func (t *TableDefinition) Date(name string) *ColumnDefinition {
	return t.appendColumn(name, "DATE")
}

func (t *TableDefinition) String(name string) *ColumnDefinition {
	return t.Varchar(name, 255)
}

func (t *TableDefinition) Timestamp(name string) *ColumnDefinition {
	return t.appendColumn(name, "TIMESTAMP")
}

func (t *TableDefinition) TimestampTZ(name string) *ColumnDefinition {
	return t.appendColumn(name, "TIMESTAMPTZ")
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
