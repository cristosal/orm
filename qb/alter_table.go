package qb

func SetDataType(table, column, dataType string) *QueryBuilder {
	qb := New()
	qb.write("ALTER TABLE ")
	qb.write(table)
	qb.write(" ALTER COLUMN ")
	qb.write(column)
	qb.write(" SET DATA TYPE ")
	qb.write(dataType)
	return qb
}

// would need a standalone one as well
func AddColumn(table string, col string) *QueryBuilder {
	qb := New()
	qb.write("ALTER TABLE ")
	qb.write(table)
	qb.write(" ADD COLUMN ")
	qb.write(col)
	return qb
}

func RenameTable(tableName, newTableName string) *QueryBuilder {
	qb := New()
	qb.write("ALTER TABLE ")
	qb.write(tableName)
	qb.write(" RENAME TO ")
	qb.write(newTableName)
	return qb
}

func RenameConstraint(tableName, constraintName, newConstraintName string) *QueryBuilder {
	qb := New()
	qb.b.WriteString("ALTER TABLE " + tableName + " RENAME CONSTRAINT " + constraintName + " TO " + newConstraintName)
	return qb
}

func RenameColumn(tableName, columnName, newColumnName string) *QueryBuilder {
	qb := New()
	qb.b.WriteString("ALTER TABLE " + tableName + " RENAME COLUMN " + columnName + " TO " + newColumnName)
	return qb
}

func DropColumn(tableName, columnName, action string) *QueryBuilder {
	qb := New()
	qb.write("ALTER TABLE ")
	qb.write(tableName)
	qb.write(" DROP COLUMN ")
	qb.write(columnName)
	if action != "" {
		qb.write(" " + action)
	}

	return qb
}
