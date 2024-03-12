package qb

func DropTable(tableName string) *QueryBuilder {
	qb := New()
	qb.write("DROP TABLE ")
	qb.write(tableName)
	return qb
}
