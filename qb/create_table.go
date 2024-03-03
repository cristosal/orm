package qb

import (
	"fmt"
	"strings"
)

func CreateTable(b *TableDefinition) *TableDefinition {
	return b
}

type TableDefinition struct {
	Name        string // table name
	IfNotExists bool   // if not exists
	Columns     Columns
	ForeignKeys []*ForeignKeyDefinition
}

func (action *TableDefinition) String() string {
	var lines []string
	for _, cd := range action.Columns {
		lines = append(lines, cd.String())
	}

	for _, fk := range action.ForeignKeys {
		lines = append(lines, fk.String())

	}

	if action.IfNotExists {
		return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", action.Name, strings.Join(lines, ", "))
	}

	return fmt.Sprintf("CREATE TABLE %s (%s)", action.Name, strings.Join(lines, ", "))
}
