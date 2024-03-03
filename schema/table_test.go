package schema_test

import (
	"testing"

	"github.com/cristosal/orm/schema"
)

func TestTables(t *testing.T) {
	td := schema.TableDefinition{}

	// schema action { action name CreateTable -> Schema.TableDefiniton }
	//well this will do shit
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

	schema.CreateTable("profile", func(t *schema.TableDefinition) {
		t.Timestamp("last_login")
		t.Text("color_theme").NotNull().Default("rgb(1, 2, 3)")
	})

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
