package qb

func DropTable(tableName string) *DropTableAction {
	return &DropTableAction{tableName: tableName}
}

type DropTableAction struct {
	tableName string
}

func (action *DropTableAction) String() string {
	return "DROP TABLE " + action.tableName
}
