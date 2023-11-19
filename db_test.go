package dbx_test

import (
	"database/sql"
	"testing"

	"github.com/cristosal/dbx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestOne(t *testing.T) {
	type TempTable struct{ V string }
	db := getDB(t)

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	defer tx.Rollback()

	if err := dbx.Exec(db, "create temporary table temp_table(v varchar(255) primary key)"); err != nil {
		t.Fatal(err)
	}

	if err := dbx.Exec(db, "insert into temp_table (v) values ($1)", "foo"); err != nil {
		t.Fatal(err)
	}

	var foo TempTable
	if err := dbx.One(db, &foo, ""); err != nil {
		t.Fatal(err)
	}

	if foo.V != "foo" {
		t.Fatalf("expected foo got = %s", foo.V)
	}

}

func TestTx(t *testing.T) {
	db := getDB(t)

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	defer tx.Rollback()

	err = dbx.Exec(tx, "create temporary table tmp(v varchar(255) primary key)")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExec(t *testing.T) {
	db := getDB(t)

	t.Cleanup(func() {
		dbx.Exec(db, "drop table test_table")
	})

	if err := dbx.Exec(db, "create table test_table (id serial primary key)"); err != nil {
		t.Fatal(err)
	}
}

func getDB(t *testing.T) *sql.DB {
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}

	return db
}
