package pgxx

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

type TestStruct struct {
	ID              ID     `db:"id"`
	Name            string `db:"name"`
	VerifySSLExpiry bool   `db:"verify_ssl_expiry"`
	ReadOnly        string `db:"foo,readonly"`
}

func (ts *TestStruct) TableName() string {
	return "tablename"
}

var connString = os.Getenv("TEST_CONNECTION_STRING")

func TestInsertAddsIDInEmbededStructs(t *testing.T) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `create temporary table a (
		id serial primary key, 
		name varchar(255)
	)`)

	if err != nil {
		t.Fatal(err)
	}
}

type tstruct2 struct {
	ID  int64  `db:"id"`
	Foo string `db:"foo"`
	Bar string `db:"bar"`
}

func (ts *tstruct2) TableName() string {
	return "test_table2"
}

func expect(t *testing.T, expected any, recieved any) {
	if expected != recieved {
		t.Fatalf("expected: %v recieved: %v", expected, recieved)
	}
}

func TestNotFoundError(t *testing.T) {
	if !errors.Is(ErrNotFound, pgx.ErrNoRows) {
		t.Fatal("expect not found to be compatible with err no rows")
	}
}

func TestInsertAddsID(t *testing.T) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}

	if err := Exec(conn, `create table if not exists test_table2 (
		id serial primary key, 
		foo varchar(255) not null, 
		bar varchar(255) not null)`); err != nil {
		t.Fatal(err)
	}
	r := tstruct2{
		Foo: "hello",
		Bar: "world",
	}

	if err := Insert(conn, &r); err != nil {
		t.Fatal(err)
	}

	if r.ID == 0 {
		t.Fatal("expected id to be filled")
	}

	defer func() {
		if err := Exec(conn, "drop table test_table2"); err != nil {
			t.Log("failed to drop test_table2", err)
		}
		if err := conn.Close(ctx); err != nil {
			t.Log("failed to close connection", err)
		}
	}()
}

func TestUpdateQuery(t *testing.T) {
	st := tstruct2{
		ID:  3,
		Foo: "hello",
		Bar: "world",
	}

	sql, vals, err := updateQ(&st)
	if err != nil {
		t.Fatal(err)
	}

	expect(t, fmt.Sprintf("update %s set foo = $1, bar = $2 where id = $3", st.TableName()), sql)
	expect(t, 3, len(vals))
	expect(t, "hello", vals[0])
	expect(t, "world", vals[1])
	expect(t, int64(3), vals[2])
}

func TestInsertQuery(t *testing.T) {
	st := tstruct2{
		Foo: "hello",
		Bar: "world",
	}

	sql, vals, err := insertQ(&st)

	if err != nil {
		t.Fatal(err)
	}

	expected := fmt.Sprintf("insert into %s (foo, bar) values ($1, $2)", st.TableName())
	expect(t, expected, sql)
	expect(t, 2, len(vals))
	expect(t, vals[0], "hello")
	expect(t, vals[1], "world")
}

type tstruct struct {
	Name string `db:"name"`
	Foo  string `db:"foo,readonly"`
	Bar  bool   `db:"bar"`
}

func (ts tstruct) TableName() string {
	return "test_table_t"
}

func TestWritableValues(t *testing.T) {
	str := tstruct{
		Name: "hello",
		Foo:  "world",
		Bar:  true,
	}

	vals, err := writeableValues(&str)
	if err != nil {
		t.Fatal(err)
	}

	if len(vals) != 2 {
		t.Fatalf("expected length of 2 recieved: %d", len(vals))
	}

	if vals[0] != "hello" {
		t.Fatalf("expected hello recieved %s.", vals[0])
	}

	if vals[1] != true {
		t.Fatalf("expected true")
	}

}

func TestAnalyze(t *testing.T) {
	var tst TestStruct

	res, err := Analyze(&tst)
	if err != nil {
		t.Fatal(err)
	}

	info := res.Fields
	if err != nil {
		t.Fatal("expected addr of struct to pass")
	}

	if len(info) != 3 {
		t.Fatalf("expected 2 info fields got %d", len(info))
	}

	if info[0].Name != "Name" {
		t.Fatal("expected Name FieldName be Name")
	}

	if info[0].Column != "name" {
		t.Fatal("expected Name column be name")
	}

	if res.Table != "tablename" {
		t.Fatalf("expected tablename got %s", res.Table)
	}

}

func TestColumnCase(t *testing.T) {
	tt := [][]string{
		{"camelCaseABCdef", "camel_case_ab_cdef"},
		{"ID", "id"},
		{"UserID", "user_id"},
		{"Name", "name"},
		{"Error", "error"},
		{"VerifySSL", "verify_ssl"},
		{"VerifySSLExpiry", "verify_ssl_expiry"},
		{"Action", "action"},
		{"Address", "address"},
		{"Redirects", "redirects"},
		{"Timeout", "timeout"},
		{"UpAt", "up_at"},
		{"DownAt", "down_at"},
		{"CreatedAt", "created_at"},
		{"UpdatedAt", "updated_at"},
		{"camelCase", "camel_case"},
	}

	for _, v := range tt {
		result := snakecase(v[0])
		if result != v[1] {
			t.Fatalf("expected=%s recieved=%s", v[1], result)
		}
	}
}
