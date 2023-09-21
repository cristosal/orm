package pgxx

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type AnalisisTest struct {
	ID        int64      `db:"id"`
	Name      string     `db:"name"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func (at *AnalisisTest) TableName() string {
	return "test_table"
}

func TestIdentityIndexes(t *testing.T) {
	type A struct {
		Foo string `db:"foo"`
		ID  ID     `db:"id"`
	}

	type B struct {
		A
		Bar int  `db:"bar"`
		Baz bool `db:"baz,readonly"`
	}

	type C struct {
		B
		Qux string `db:"qux"`
	}
	v := C{B: B{A: A{}}}

	sch := MustAnalyze(&v)
	_, indexes, err := sch.Fields.Identity()
	if err != nil {
		t.Fatal(err)
	}

	expected := []int{0, 0, 1}
	if len(expected) != len(indexes) {
		t.Fatalf("expected indexes to be %d, got=%d", len(expected), len(indexes))
	}

	for i := range expected {
		if expected[i] != indexes[i] {
			t.Fatalf("expected index=%d got=%d", expected[i], indexes[i])
		}
	}

}

func TestFieldRecursionInEmbededStructs(t *testing.T) {
	type A struct {
		ID  ID     `db:"id"`
		Foo string `db:"foo"`
	}

	type B struct {
		A
		Bar int  `db:"bar"`
		Baz bool `db:"baz,readonly"`
	}

	type C struct {
		B
		Qux string `db:"qux"`
	}

	v := C{
		Qux: "test",
		B: B{
			A:   A{Foo: "foo"},
			Bar: 42,
			Baz: true,
		},
	}

	schema := MustAnalyze(&v)
	fields := schema.Fields.Writeable()

	tt := []string{"Foo", "Bar", "Qux"}
	for i := range tt {
		if fields[i].Name != tt[i] {
			t.Fatalf("expected=%s got=%s", tt[i], fields[i].Name)
		}
	}

	tt = []string{"id", "foo", "bar", "baz", "qux"}
	cols := schema.Fields.Columns()
	for i := range tt {
		if cols[i] != tt[i] {
			t.Fatalf("expected column=%s got=%s", tt[i], cols[i])
		}
	}

	f, _, err := schema.Fields.Identity()
	if err != nil {
		t.Fatalf("expected to have found identity field in embeded struct")
	}

	if f.Index != 0 {
		t.Fatalf("expected identity index to be first on struct")
	}

}

func TestEmbedInsert(t *testing.T) {
	type A struct {
		ID   ID
		Name string `db:"name"`
	}

	type TestEmbeds struct {
		A
		Age int `db:"age"`
	}

	conn := getConn(t)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `create temporary table test_embeds (
		id serial primary key,
		name varchar(255) not null,
		age int not null
	)`)

	if err != nil {
		t.Fatal(err)
	}
	r := TestEmbeds{A: A{Name: "Cristobal Salazar"}, Age: 29}

	if err := Insert(tx, &r); err != nil {
		t.Fatal(err)
	}

	row := tx.QueryRow(ctx, "select name, age from test_embeds")
	var (
		name string
		age  int
	)

	if err := row.Scan(&name, &age); err != nil {
		t.Fatal(err)
	}

	if name != r.Name {
		t.Fatal("expected names to match")
	}

	if age != r.Age {
		t.Fatal("expected age to match")
	}

}

func TestEmbededStructs(t *testing.T) {

	type A struct {
		Name string `db:"name"`
	}

	type B struct {
		A
		Age int `db:"age"`
	}

	ClearCache()

	sch, err := Analyze(B{})
	if err != nil {
		t.Fatal(err)
	}

	got := sch.Fields.Columns().List()
	expected := "age, name"

	if got != expected {
		t.Fatalf("expected=%s got=%s", expected, got)
	}

}

func TestScanableValues(t *testing.T) {
	flag.Parse()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.Exec(ctx, "create table if not exists test_table (id serial primary key, name varchar(255) not null, deleted_at timestamptz)")
	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.Exec(ctx, "insert into test_table (name, deleted_at) values ($1, $2)", "Hello World", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	at := AnalisisTest{}
	vals, err := ScanableValues(&at)
	if err != nil {
		t.Fatal(err)
	}

	if len(vals) != 3 {
		t.Fatalf("expected 3 got %d", len(vals))
	}

	res, err := Analyze(&at)
	if err != nil {
		t.Fatal(err)
	}

	row := conn.QueryRow(ctx, fmt.Sprintf("select %s from test_table", res.Fields.Columns().List()))
	if err = row.Scan(vals...); err != nil {
		t.Fatal(err)
	}

	if at.Name != "Hello World" {
		t.Fatal("expected struct field to be scanned")
	}

	if at.DeletedAt == nil {
		t.Fatal("expected deleted at to be scanned")
	}

	t.Cleanup(func() {
		_, err = conn.Exec(ctx, "drop table test_table")
		if err := conn.Close(ctx); err != nil {
			t.Log("failed to close connection")
		}
	})
}

func TestTimeValues(t *testing.T) {
	st := AnalisisTest{Name: "Test1"}
	vals, err := WriteableValues(&st)
	if err != nil {
		t.Fatal(err)
	}

	expect(t, vals[1], nil)

	now := time.Now()
	st.DeletedAt = &now

	vals, err = WriteableValues(&st)
	if err != nil {
		t.Fatal(err)
	}
	if vals[1] != &now {
		t.Fatalf("expected: %p recieved: %p", &now, vals[1])
	}

}
func getConn(t *testing.T) *pgx.Conn {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		conn.Close(ctx)
	})

	return conn
}
