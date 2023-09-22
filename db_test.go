package pgxx

import (
	"context"
	"errors"
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

func TestSelectMany(t *testing.T) {
	tx := getTx(t)
	ClearCache()
	// person
	type Person struct {
		ID   ID
		Name string
		Age  int
	}

	tx.Exec(ctx, `create temporary table person (
		id serial primary key,
		name varchar(255),
		age int not null
	)`)

	data := []Person{
		{Name: "Foo", Age: 12},
		{Name: "Bar", Age: 24},
		{Name: "Baz", Age: 32},
	}

	for i := range data {
		if err := Insert(tx, &data[i]); err != nil {
			t.Fatal(err)
		}
	}

	var persons []Person
	if err := SelectMany(tx, &persons, ""); err != nil {
		t.Fatal(err)
	}

	if len(persons) != len(data) {
		t.Fatalf("expected len=%d got=%d", len(data), len(persons))
	}

	for i := range persons {
		if persons[i].Name != data[i].Name {
			t.Fatal("expected names to match")
		}

		if persons[i].Age != data[i].Age {
			t.Fatal("expected ages to match")
		}
	}
}

func TestQueryOne(t *testing.T) {
	tx := getTx(t)
	ClearCache()
	tx.Exec(context.Background(), `create temporary table b (
		id serial primary key,
		name varchar(255) not null,
		age int not null
	)`)

	type A struct {
		ID   ID
		Name string
	}

	type B struct {
		A
		Age int
	}
	v := B{A: A{Name: "Test"}, Age: 12}
	if err := Insert(tx, &v); err != nil {
		t.Fatal(err)
	}

	var b B

	if err := One(tx, &b, "where name = $1", "Test"); err != nil {
		t.Fatal(err)
	}

	if b.ID != v.ID {
		t.Fatalf("expected id=%d got = %d", v.ID, b.ID)
	}

	if b.Name != v.Name {
		t.Fatalf("expected=%s got = %s", v.Name, b.Name)
	}

	if b.Age != v.Age {
		t.Fatalf("expected=%d got=%d", v.Age, b.Age)
	}

}

func TestInsertAddsIDInEmbededStructs(t *testing.T) {
	tx := getTx(t)
	ClearCache()

	tx.Exec(ctx, `create temporary table b (
		id serial primary key, 
		name varchar(255) not null,
		age int not null
	)`)

	type A struct {
		ID   ID
		Name string
	}

	type B struct {
		A
		Age int
	}

	v := B{A: A{Name: "Test"}, Age: 12}

	if err := Insert(tx, &v); err != nil {
		t.Fatal(err)
	}

	if v.ID == 0 {
		t.Fatal("expected id to be set")
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

func TestUpdate(t *testing.T) {
	tx := getTx(t)
	type A struct {
		ID   ID
		Name string
	}

	type B struct {
		A
		Age int
	}

	tx.Exec(ctx, `create temporary table b (
		id serial primary key,
		name varchar not null,
		age integer not null
	)`)

	b := &B{A: A{Name: "John Doe"}, Age: 12}
	if err := Insert(tx, b); err != nil {
		t.Fatal(err)
	}

	b.Age = 32
	b.Name = "Bob Smith"
	if err := Update(tx, b); err != nil {
		t.Fatal(err)
	}

	row := tx.QueryRow(ctx, "select name, age from b where id = $1", b.ID)
	var (
		age  int
		name string
	)

	if err := row.Scan(&name, &age); err != nil {
		t.Fatal(err)
	}

	if age != b.Age {
		t.Fatalf("expected age=%d got=%d", b.Age, age)
	}

	if name != b.Name {
		t.Fatalf("expected name=%s got=%s", b.Name, name)
	}

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

	vals, err := WriteableValues(&str)
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

	if len(info) != 4 {
		t.Fatalf("expected 4 info fields got %d", len(info))
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

func getTx(t *testing.T) Interface {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		tx.Rollback(ctx)
	})

	ClearCache()
	return tx
}
