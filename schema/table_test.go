package schema_test

import (
	"testing"

	"github.com/cristosal/orm/schema"
)

type UsersTable struct {
	ID *schema.ColumnDefinition
}

func TestDropTableAction(t *testing.T) {
	got := schema.DropTable("users").String()
	expected := "DROP TABLE users"

	if expected != got {
		t.Fatalf("expected: %s\ngot: %s", expected, got)
	}
}

func TestCreateTableAction(t *testing.T) {
	createUsersTableAction := schema.CreateTable("users", func(t *schema.TableDefinition) {
		t.Serial("id").PrimaryKey()
		t.String("name").NotNull()
		t.Integer("age").Default(18)
		t.Integer("profile_id").NotNull()
		t.Foreign("profile_id", "profile", "id").OnDelete("CASCADE")
	})

	expected := "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL, age INTEGER DEFAULT 18, profile_id INTEGER NOT NULL, FOREIGN KEY(profile_id) REFERENCES profile(id) ON DELETE CASCADE)"

	if createUsersTableAction.String() != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, createUsersTableAction.String())

	}
}

func TestTableDefinition(t *testing.T) {
	td := schema.TableDefinition{}
	tt := [][]string{
		{td.Serial("id").PrimaryKey().String(), "id SERIAL PRIMARY KEY"},
		{td.Varchar("name", 255).NotNull().Default("John Doe").String(), "name VARCHAR(255) NOT NULL DEFAULT 'John Doe'"},
		{td.Integer("age").NotNull().Default(1).String(), "age INTEGER NOT NULL DEFAULT 1"},
		{td.Integer("profile_id").NotNull().String(), "profile_id INTEGER NOT NULL"},
		{td.Foreign("profile_id", "profile", "id").OnDelete("CASCADE").String(), "FOREIGN KEY(profile_id) REFERENCES profile(id) ON DELETE CASCADE"},
	}

	for i, tc := range tt {
		if tc[0] != tc[1] {
			t.Fatalf("test case %d failed:\nexpected: %s\ngot: %s", i, tc[1], tc[0])
		}
	}

}
