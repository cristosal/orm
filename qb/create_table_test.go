package qb_test

import (
	"testing"

	"github.com/cristosal/orm/qb"
)

func TestCreateTable(t *testing.T) {
	qb.CreateTable(&qb.TableDefinition{
		Name: "users",
		Columns: qb.Columns{
			qb.Serial("id").PrimaryKey(),
			qb.String("name").NotNull(),
			qb.String("email").NotNull().Unique(),
		},
	})

	qb.DropTable("users")
}
