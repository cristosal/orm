package orm_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/cristosal/orm"
	"github.com/cristosal/orm/schema"
)

func TestPaginate(t *testing.T) {
	db := mockDB{}
	type User struct {
		ID       int64
		Username string
		Password string
	}

	var users []User

	orm.Paginate(&db, &users, &orm.PaginationOptions{
		Page:     1,
		PageSize: 25,
	})

	db.ExpectSQL(t, "")
}

func TestUpdate(t *testing.T) {
	db := mockDB{}
	type A struct {
		ID       int64
		Username string
		Password string
	}
	var a A

	a.Username = "foo"
	a.Password = "bar"
	orm.Update(&db, &a, "WHERE username = $1", a.Username)
	db.ExpectSQL(t, "UPDATE a SET username = $2, password = $3 WHERE username = $1")
	db.ExpectValueAt(t, 0, a.Username)
	db.ExpectValueAt(t, 1, a.Username)
	db.ExpectValueAt(t, 2, a.Password)
}

func TestFieldsFindByColumn(t *testing.T) {
	type A struct {
		ID       int64
		Username string
		Password string
	}

	type B struct {
		A
		Role string
	}

	var b B
	schema := schema.MustGet(&b)
	_, indexes, err := schema.Fields.FindByColumn("username")
	if err != nil {
		t.Fatal(err)
	}

	// this is the tt
	expected := []int{0, 1}

	for i := range expected {
		if indexes[i] != expected[i] {
			t.Fatalf("expected index %d to be %d, got %d", i, expected[i], indexes[i])
		}
	}
}

func TestOneNoSQL(t *testing.T) {
	type TempTable struct{ V string }
	db := &mockDB{}
	var foo TempTable
	orm.Get(db, &foo, "")
	db.ExpectSQL(t, "SELECT v FROM temp_table")
}

func TestOneSQL(t *testing.T) {
	type TempTable struct{ V string }
	db := &mockDB{}
	var foo TempTable
	orm.Get(db, &foo, "WHERE v = $1", 1)
	db.ExpectSQL(t, "SELECT v FROM temp_table WHERE v = $1")
	db.ExpectValueAt(t, 0, 1)
}

func TestExec(t *testing.T) {
	db := &mockDB{}
	sql := "create table test_table (id serial primary key)"
	// orm.Exec(db, sql)
	orm.Exec(db, sql)
	db.ExpectSQL(t, sql)
}

type mockResult struct{}

func (mockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

type mockDB struct {
	SQL    string
	Values []any
}

func (db *mockDB) ExpectSQL(t *testing.T, sql string) {
	if sql != db.SQL {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", sql, db.SQL)
	}
}

func (db *mockDB) Begin() (*sql.Tx, error) {
	return nil, nil
}

func (db *mockDB) ExpectValueAt(t *testing.T, index int, value interface{}) {
	if db.Values[index] != value {
		t.Fatalf("expected value at index %d to be %v\ngot %v", index, db.Values[index], value)
	}
}

func (db *mockDB) Exec(s string, args ...any) (sql.Result, error) {
	db.SQL = s
	db.Values = args
	return mockResult{}, nil
}

func (db *mockDB) Query(s string, args ...any) (*sql.Rows, error) {
	db.SQL = s
	db.Values = args
	return nil, errors.New("test implementation")
}

func (db *mockDB) QueryRow(s string, args ...any) *sql.Row {
	db.SQL = s
	db.Values = args
	return nil
}
