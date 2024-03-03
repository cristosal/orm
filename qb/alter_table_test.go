package qb_test

import (
	"testing"

	"github.com/cristosal/orm/qb"
)

func TestAddColumn(t *testing.T) {
	got := qb.AddColumn("users", qb.Varchar("zip", 8)).String()
	expected := "ALTER TABLE users ADD COLUMN zip VARCHAR(8)"
	if got != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, got)
	}
}

func TestAlterTable(t *testing.T) {
	tt := [][]string{
		{qb.DropColumn("users", "email").String(), "ALTER TABLE users DROP COLUMN email"},
		{qb.RenameColumn("users", "email", "email_address").String(), "ALTER TABLE users RENAME COLUMN email TO email_address"},
		{qb.RenameConstraint("users", "fk_profile_id", "fk_user_profile_id").String(), "ALTER TABLE users RENAME CONSTRAINT fk_profile_id TO fk_user_profile_id"},
		{qb.RenameTable("users", "members").String(), "ALTER TABLE users RENAME TO members"},
	}

	for _, tc := range tt {
		if tc[0] != tc[1] {
			t.Fatalf("expected: %s\ngot: %s", tc[1], tc[0])
		}
	}

}
