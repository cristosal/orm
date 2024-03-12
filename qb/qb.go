package qb

import (
	"strconv"
	"strings"

	"github.com/cristosal/orm"
)

type Dialect interface {
	Placeholder(n int) string
}

type PostgresDialect struct{}

func (PostgresDialect) Placeholder(n int) string {
	return "$" + strconv.Itoa(n)
}

var DefaultDialect Dialect = PostgresDialect{}

type QueryBuilder struct {
	dialect Dialect
	b       *strings.Builder
	v       []interface{}
}

func (qb *QueryBuilder) apply(v ...BuildFunc) {
	for i := range v {
		v[i](qb)
	}
}

func (qb *QueryBuilder) n() int {
	return len(qb.v)
}

func (qb *QueryBuilder) String() string {
	return qb.b.String()
}

func New() *QueryBuilder {
	return &QueryBuilder{
		b:       new(strings.Builder),
		v:       make([]interface{}, 0),
		dialect: DefaultDialect,
	}
}

func (qb *QueryBuilder) SetDialect(d Dialect) *QueryBuilder {
	qb.dialect = d
	return qb
}

func (qb *QueryBuilder) write(sql string) {
	qb.b.WriteString(sql)
}

func (qb *QueryBuilder) addValues(values ...interface{}) {
	qb.v = append(qb.v, values...)
}

func (qb *QueryBuilder) Exec(db orm.DB) error {
	return orm.Exec(db, qb.b.String(), qb.v...)
}
