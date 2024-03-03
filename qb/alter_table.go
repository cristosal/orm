package qb

import "fmt"

type addColumnStmt struct {
	tableName string
	col       *ColumnDefinition
}

func (stmt addColumnStmt) String() string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", stmt.tableName, stmt.col)
}

func AddColumn(table string, column *ColumnDefinition) fmt.Stringer {
	return addColumnStmt{table, column}
}

type renameTableStmt struct{ tableName, newTableName string }

func (stmt renameTableStmt) String() string {
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", stmt.tableName, stmt.newTableName)
}

func RenameTable(tableName, newTableName string) fmt.Stringer {
	return renameTableStmt{tableName, newTableName}
}

type renameConstraintStmt struct{ tableName, constraintName, newConstraintName string }

func (stmt renameConstraintStmt) String() string {
	return fmt.Sprintf("ALTER TABLE %s RENAME CONSTRAINT %s TO %s",
		stmt.tableName, stmt.constraintName, stmt.newConstraintName)
}

func RenameConstraint(tableName, constraintName, newConstraintName string) fmt.Stringer {
	return renameConstraintStmt{tableName, constraintName, newConstraintName}
}

type renameColumnStmt struct{ tableName, columnName, newColumnName string }

func (stmt renameColumnStmt) String() string {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", stmt.tableName, stmt.columnName, stmt.newColumnName)
}

func RenameColumn(tableName, columnName, newColumnName string) fmt.Stringer {
	return renameColumnStmt{tableName, columnName, newColumnName}
}

type dropColumnStmt struct{ tableName, columnName string }

func (stmt dropColumnStmt) String() string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", stmt.tableName, stmt.columnName)
}

func DropColumn(tableName, columnName string) fmt.Stringer {
	return dropColumnStmt{
		tableName:  tableName,
		columnName: columnName,
	}
}
