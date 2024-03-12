package qb_test

import (
	"testing"

	"github.com/cristosal/orm/qb"
)

func TestCreateTable(t *testing.T) {
	stmt := qb.CreateTable("users", func(t qb.TableBuilder) {
		t.Serial("id").PrimaryKey()
		t.String("name").NotNull()
		t.String("email").NotNull().Unique()
	})

	expected := `CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) NOT NULL UNIQUE)`
	got := stmt.String()

	if got != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, got)
	}
}
