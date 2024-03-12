package qb

import (
	"strconv"
	"strings"
)

type (
	SelectBuilder struct{ *QueryBuilder }
	BuildFunc     func(*QueryBuilder)
)

func Select(v ...string) SelectBuilder {
	qb := New()
	b := SelectBuilder{qb}
	b.write("SELECT ")
	if len(v) == 0 {
		b.write("*")
	} else {
		b.write(strings.Join(v, ", "))
	}
	return b
}

func (b SelectBuilder) From(table string) SelectBuilder {
	b.write(" FROM ")
	b.write(table)
	return b
}

func Or(v ...BuildFunc) BuildFunc {
	return func(qb *QueryBuilder) {
		for i := range v {
			if i > 0 {
				qb.write(" OR ")
			}
			v[i](qb)
		}
	}
}

func And(v ...BuildFunc) BuildFunc {
	return func(qb *QueryBuilder) {
		for i := range v {
			if i > 0 {
				qb.write(" AND ")
			}
			v[i](qb)
		}
	}
}

func IsNull(col string) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(col)
		qb.write(" IS NULL")
	}
}

func IsNotNull(col string) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(col)
		qb.write(" IS NOT NULL")
	}
}

func Gte(lhs string, rhs any) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(lhs)
		qb.write(" >= ")
		qb.addValues(rhs)
		qb.write(qb.dialect.Placeholder(qb.n()))
	}
}

func Gt(lhs string, rhs any) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(lhs)
		qb.write(" > ")
		qb.addValues(rhs)
		qb.write(qb.dialect.Placeholder(qb.n()))
	}
}

func Lte(lhs string, rhs any) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(lhs)
		qb.write(" <= ")
		qb.addValues(rhs)
		qb.write(qb.dialect.Placeholder(qb.n()))
	}
}

func Lt(lhs string, rhs any) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(lhs)
		qb.write(" < ")
		qb.addValues(rhs)
		qb.write(qb.dialect.Placeholder(qb.n()))
	}
}

func Neq(lhs string, rhs any) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(lhs)
		qb.write(" != ")
		qb.addValues(rhs)
		qb.write(qb.dialect.Placeholder(qb.n()))
	}
}

func Eq(lhs string, rhs any) BuildFunc {
	return func(qb *QueryBuilder) {
		qb.write(lhs)
		qb.write(" = ")
		qb.addValues(rhs)
		qb.write(qb.dialect.Placeholder(qb.n()))
	}
}

func (b SelectBuilder) Limit(n int) SelectBuilder {
	b.write(" LIMIT ")
	b.write(strconv.Itoa(n))
	return b
}

func (b SelectBuilder) Offset(n int) SelectBuilder {
	b.write(" OFFSET ")
	b.write(strconv.Itoa(n))
	return b
}

func (b SelectBuilder) Where(v ...BuildFunc) SelectBuilder {
	b.write(" WHERE ")
	b.apply(v...)
	return b
}
