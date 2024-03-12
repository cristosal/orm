package qb

import (
	"strconv"
	"strings"
)

func CreateTable(table string, buildFunc func(TableBuilder)) *QueryBuilder {
	qb := New()
	qb.b.WriteString("CREATE TABLE IF NOT EXISTS ")
	qb.b.WriteString(table)
	qb.b.WriteString(" (")

	// this gets its own builder as its used to determine if a new line should be added
	t := TableBuilder{new(strings.Builder)}
	buildFunc(t)
	qb.b.WriteString(t.Builder.String())
	qb.b.WriteString(")")
	return qb
}

type TableBuilder struct{ *strings.Builder }

func (b *TableBuilder) write(sql string) {
	b.WriteString(sql)
}

type ColumnBuilder struct{ *strings.Builder }

func (b *TableBuilder) writeColumn(name, typ string) ColumnBuilder {
	b.maybeWriteComma()
	b.write(name)
	b.write(" ")
	b.write(typ)
	return ColumnBuilder{b.Builder}
}

// this should return foreign key builder

type ForeignKeyBuilder struct {
	t *TableBuilder
}

func (b *TableBuilder) ForeignKey(name, referenceTable, referenceColumn string) ForeignKeyBuilder {
	b.maybeWriteComma()
	b.write("FOREIGN KEY (")
	b.write(name)
	b.write(") REFERENCES ")
	b.write(referenceTable)
	b.write("(")
	b.write(referenceColumn)
	b.write(")")
	return ForeignKeyBuilder{b}
}

func (b ForeignKeyBuilder) OnDelete(action string) ForeignKeyBuilder {
	b.t.write(" ON DELETE ")
	b.t.write(action)
	return b
}

func (b ForeignKeyBuilder) OnUpdate(action string) ForeignKeyBuilder {
	b.t.write(" ON UPDATE ")
	b.t.write(action)
	return b
}

func (b *TableBuilder) SmallInt(name string) ColumnBuilder {
	return b.writeColumn(name, "SMALLINT")
}

func (b *TableBuilder) Integer(name string) ColumnBuilder {
	return b.writeColumn(name, "INTEGER")
}

func (b *TableBuilder) BigInt(name string) ColumnBuilder {
	return b.writeColumn(name, "BIGINT")
}

func (b *TableBuilder) JSON(name string) ColumnBuilder {
	return b.writeColumn(name, "JSON")
}

func (b *TableBuilder) JSONB(name string) ColumnBuilder {
	return b.writeColumn(name, "JSONB")
}

func (b *TableBuilder) Text(name string) ColumnBuilder {
	return b.writeColumn(name, "TEXT")
}

func (b *TableBuilder) maybeWriteComma() {
	if b.Len() > 0 {
		b.write(", ")
	}
}

func (b *TableBuilder) Char(name string, length int) ColumnBuilder {
	col := b.writeColumn(name, "CHAR")
	col.WriteString("(")
	col.WriteString(strconv.Itoa(length))
	col.WriteString(")")
	return col
}

func (b *TableBuilder) Varchar(name string, length int) ColumnBuilder {
	col := b.writeColumn(name, "VARCHAR")
	col.WriteString("(")
	col.WriteString(strconv.Itoa(length))
	col.WriteString(")")
	return col
}

func (b *TableBuilder) Boolean(name string) ColumnBuilder {
	return b.writeColumn(name, "BOOLEAN")
}

func (b *TableBuilder) Serial(name string) ColumnBuilder {
	return b.writeColumn(name, "SERIAL")
}

func (b *TableBuilder) Time(name string) ColumnBuilder {
	return b.writeColumn(name, "TIME")
}

func (b *TableBuilder) Interval(name string) ColumnBuilder {
	return b.writeColumn(name, "INTERVAL")
}

func (b *TableBuilder) Date(name string) ColumnBuilder {
	return b.writeColumn(name, "DATE")
}

func (b *TableBuilder) String(name string) ColumnBuilder {
	return b.Varchar(name, 255)
}

func (b *TableBuilder) Timestamp(name string) ColumnBuilder {
	return b.writeColumn(name, "TIMESTAMP")
}

func (b *TableBuilder) TimestampTZ(name string) ColumnBuilder {
	return b.writeColumn(name, "TIMESTAMPTZ")
}

func (b ColumnBuilder) NotNull() ColumnBuilder {
	b.WriteString(" NOT NULL")
	return b
}

func (b ColumnBuilder) PrimaryKey() ColumnBuilder {
	b.WriteString(" PRIMARY KEY")
	return b
}

func (b ColumnBuilder) Unique() ColumnBuilder {
	b.WriteString(" UNIQUE")
	return b
}

func (b ColumnBuilder) Default(str string) ColumnBuilder {
	b.WriteString(" DEFAULT ")
	b.WriteString(str)
	return b
}
