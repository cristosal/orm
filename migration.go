package orm

import (
	"database/sql"
	"errors"
	"fmt"
)

const (
	// DefaultMigrationTable is the name given to the migrations table if not overriden by SetMigrationTable
	DefaultMigrationTable = "_migrations"
)

var (
	ErrMigrationAlreadyExists = errors.New("migration already exists")
	migrationTable            = DefaultMigrationTable
)

// Migration represents a structured change to the database
type (
	Migration struct {
		ID   int    // id of migration for ordering
		Name string // name of migration must be unique
	}

	MigrationFunc func(tx *sql.Tx) error
)

// TableName used to store migrations
func (Migration) TableName() string {
	return migrationTable
}

// SetMigrationTable sets the table where migration history will be stored and executed
func SetMigrationTable(table string) {
	if table != "" {
		migrationTable = table
	}
}

// CreateMigrationTable creates the table where the migration history will be stored.
// The name of the table can be configured using the SetMigrationTable method.
func (o *ORM) CreateMigrationTable() error {
	return CreateMigrationTable(o.DB)
}

// CreateMigrationTable creates the table where the migration history will be stored.
// The name of the table can be configured using the SetMigrationTable method.
func CreateMigrationTable(db DB) error {
	sql := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY, 
			name VARCHAR(255) NOT NULL UNIQUE
		);`, migrationTable)

	_, err := db.Exec(sql)
	return err
}

// DropMigrationTable drops the migration history table
func (o *ORM) DropMigrationTable() error { return DropMigrationTable(o.DB) }

// DropMigrationTable drops the migration history table
func DropMigrationTable(db DB) error {
	return Exec(db, fmt.Sprintf("DROP TABLE %s", migrationTable))
}

// Migrate database and execute it.
func (o *ORM) Migrate(name string, fn MigrationFunc) error { return Migrate(o.DB, name, fn) }

// Migrate executes a migration
func Migrate(db DB, name string, fn MigrationFunc) error {
	m := &Migration{
		Name: name,
	}

	if m.Name == "" {
		return errors.New("migration name is required")
	}

	// check if we have executed a migration with this name already
	var found Migration
	_ = Get(db, &found, "WHERE name = $1", m.Name)

	if found.Name == m.Name {
		return ErrMigrationAlreadyExists
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// execute migration
	if err := fn(tx); err != nil {
		return err
	}

	if err := Add(tx, m); err != nil {
		return err
	}

	return tx.Commit()
}

// ListMigrations returns all migrations that have been executed
func (o *ORM) ListMigrations() ([]Migration, error) { return ListMigrations(o.DB) }

// ListMigrations returns all migrations
func ListMigrations(db DB) ([]Migration, error) {
	var migrations []Migration
	if err := List(db, &migrations, "ORDER BY id ASC"); err != nil {
		return nil, err
	}

	return migrations, nil
}
