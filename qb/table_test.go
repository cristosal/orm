package qb_test

import (
	"testing"

	"github.com/cristosal/orm/qb"
)

func TestDropTableAction(t *testing.T) {
	got := qb.DropTable("users").String()
	expected := "DROP TABLE users"

	if expected != got {
		t.Fatalf("expected: %s\ngot: %s", expected, got)
	}
}

func TestTableDefinition(t *testing.T) {
	tt := [][]string{
		{qb.Serial("id").PrimaryKey().String(), "id SERIAL PRIMARY KEY"},
		{qb.Varchar("name", 255).NotNull().Default("John Doe").String(), "name VARCHAR(255) NOT NULL DEFAULT 'John Doe'"},
		{qb.Integer("age").NotNull().Default(1).String(), "age INTEGER NOT NULL DEFAULT 1"},
		{qb.Integer("profile_id").NotNull().String(), "profile_id INTEGER NOT NULL"},
		{qb.ForeignKey("profile_id", "profile", "id").OnDelete("CASCADE").String(), "FOREIGN KEY(profile_id) REFERENCES profile(id) ON DELETE CASCADE"},
	}

	for i, tc := range tt {
		if tc[0] != tc[1] {
			t.Fatalf("test case %d failed:\nexpected: %s\ngot: %s", i, tc[1], tc[0])
		}
	}

}
