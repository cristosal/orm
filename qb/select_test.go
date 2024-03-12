package qb_test

import (
	"testing"

	"github.com/cristosal/orm/qb"
)

func TestSelect(t *testing.T) {
	/*
		var u User
		type TableDefinition = map[string]ColumnDefinition
		var usersTable = TableDefinition{
			"id": 			t.Serial().PrimaryKey(),
			"profile_id": 	t.Integer().References("profiles", "id").OnDelete("cascade"),
		}


		orm.SelectInto(db, &u, func (q SelectBuilder) {
			q.Where(
				q.Or(
					q.Eq("id", 32),
					q.Where("username", "bob")
				)
			)
		})
	*/

	tt := [][]string{
		{
			qb.Select().From("users").Where(qb.Eq("id", 123)).String(),
			"SELECT * FROM users WHERE id = $1",
		},
		{
			qb.Select("username").From("users").Where(qb.And(qb.Eq("username", "frank"))).String(),
			"SELECT username FROM users WHERE username = $1",
		},
		{
			qb.Select("username").From("users").Where(qb.And(qb.Eq("username", "frank"), qb.Gt("age", 18))).String(),
			"SELECT username FROM users WHERE username = $1 AND age > $2",
		},
	}

	for _, tc := range tt {
		got := tc[0]
		expected := tc[1]
		if got != expected {
			t.Fatalf("expected: %s\ngot: %s", expected, got)
		}
	}
}
