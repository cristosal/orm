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
}
