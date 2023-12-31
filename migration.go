package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"
)

const (
	// defaultMigrationTable is the name given to the migration table if not overriden by SetMigrationTable
	defaultMigrationTable = "_migrations"

	// defaultSchemaName is the name given to the migration table schema if not overriden by SetSchemaName
	defaultSchemaName = "public"
)

var (
	ErrNoMigration = errors.New("no migration found")

	ErrUnknownDriver = errors.New("unknown driver")

	migrationTablename string = "_migrations"
	schemaName         string = "public"
)

// Migration is a structured change to the database
type Migration struct {
	ID          int64
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	Up          string `mapstructure:"up"`
	Down        string `mapstructure:"down"`
	MigratedAt  time.Time
}

// MigrationTable returns the fully qualified, schema prefixed table name
func MigrationTable() string {
	switch driver {
	case "pgx", "postgres":
		return schemaName + "." + migrationTablename
	default:
		return migrationTablename
	}
}

// SetMigrationTable sets the default table where migrations will be stored and executed
func SetMigrationTable(table string) {
	if table != "" {
		migrationTablename = table
	}
}

// SetSchema sets the schema for the migration table
func SetSchema(schema string) {
	if schema != "" {
		schemaName = schema
	}
}

func createMigrationTableSqlite(db DB) error {
	if migrationTablename == "" {
		migrationTablename = defaultMigrationTable
	}

	_, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		up TEXT,
		down TEXT,
		migrated_at TEXT
	);`, migrationTablename))

	return err
}

func createMigrationTablePostgres(db DB) error {
	if schemaName == "" {
		schemaName = defaultSchemaName
	}

	if migrationTablename == "" {
		migrationTablename = defaultMigrationTable
	}

	_, err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		up TEXT,
		down TEXT,
		migrated_at TIMESTAMPTZ
	);`, MigrationTable()))

	return err
}

// CreateMigrationTable creates the table and schema where migrations will be stored and executed.
// The name of the table can be set using the SetMigrationTable method.
// The name of the schema can be set using the SetSchema method.
func CreateMigrationTable(db DB) error {
	switch driver {
	case "sqlite3":
		return createMigrationTableSqlite(db)
	case "postgres", "pgx":
		return createMigrationTablePostgres(db)
	default:
		return ErrUnknownDriver
	}
}

// DropMigrationTable
func DropMigrationTable(db DB) error {
	return Exec(db, fmt.Sprintf("DROP TABLE %s", MigrationTable()))
}

// AddMigration to the database and execute it.
// No error is returned if a migration with the same name has already been executed.
// Up and Down sql strings are parsed as go template strings before execution and saved to the database.
func AddMigration(db DB, migration *Migration) error {
	if migration.Name == "" {
		return errors.New("migration name is required")
	}

	if migration.Up == "" {
		return errors.New("up sql is required")
	}

	var (
		sql  = fmt.Sprintf("SELECT name FROM %s WHERE name = $1", MigrationTable())
		name string
		row  = db.QueryRow(sql, migration.Name)
	)

	row.Scan(&name)
	// return nil if migration has already been executed
	if name == migration.Name {
		return nil
	}

	up, err := parseMigrationTmpl(migration.Up)
	if err != nil {
		return err
	}

	down, err := parseMigrationTmpl(migration.Down)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// insert record of the migration
	sql = fmt.Sprintf("INSERT INTO %s (name, description, up, down) VALUES ($1, $2, $3, $4)", MigrationTable())
	if _, err := tx.Exec(sql, migration.Name, migration.Description, up, down); err != nil {
		return err
	}

	// execute up migration
	if _, err := tx.Exec(up); err != nil {
		return err
	}

	// set migration as executed
	sql = fmt.Sprintf("UPDATE %s SET migrated_at = NOW() WHERE name = $1", MigrationTable())
	if _, err := tx.Exec(sql, migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}

// AddMigrations pushes multiple migrations returning the first error encountered.
// See AddMigration for further info.
func AddMigrations(db DB, migrations []Migration) error {
	for i := range migrations {
		if err := AddMigration(db, &migrations[i]); err != nil {
			return err
		}
	}

	return nil
}

// RemoveMigration reverts the last migration.
// The down migration of the most recently added migration is executed, and the migration deleted from the table.
func RemoveMigration(db DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	stmt := fmt.Sprintf(`SELECT name, down FROM %s ORDER BY migrated_at DESC`, MigrationTable())
	row := tx.QueryRow(stmt)

	var (
		name string
		down string
	)

	if err := row.Scan(&name, &down); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoMigration
		}

		return err
	}

	if _, err := tx.Exec(down); err != nil {
		return err
	}

	stmt = fmt.Sprintf("DELETE FROM %s WHERE name = $1", MigrationTable())
	if _, err := tx.Exec(stmt, name); err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveAllMigrations reverts all migrations.
// See RemoveMigration for further info.
func RemoveAllMigrations(db DB) (int, error) {
	var n int

	for {
		if err := RemoveMigration(db); err != nil {
			if errors.Is(err, ErrNoMigration) {
				if n == 0 {
					return 0, ErrNoMigration
				}

				return n, nil
			}

			return n, err
		}
		n++
	}
}

// RemoveMigrationsUntil remvoes migrations until a migration with name is encountered.
func RemoveMigrationsUntil(db DB, name string) error {
	var (
		mig *Migration
		err error
	)

	for {
		mig, err = GetLatestMigration(db)

		if err != nil {
			return err
		}

		if mig.Name == name {
			return nil
		}

		if err := RemoveMigration(db); err != nil {
			return err
		}
	}
}

// GetLatestMigration returns the latest migration added
func GetLatestMigration(db DB) (*Migration, error) {
	sql := fmt.Sprintf(`SELECT id, name, description, up, down, migrated_at FROM %s ORDER BY migrated_at DESC`, MigrationTable())
	row := db.QueryRow(sql)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var mig Migration
	if err := row.Scan(
		&mig.ID,
		&mig.Name,
		&mig.Description,
		&mig.Up,
		&mig.Down,
		&mig.MigratedAt); err != nil {
		return nil, err
	}

	return &mig, nil
}

// ListMigrations returns all migrations
func ListMigrations(db DB) ([]Migration, error) {
	sql := fmt.Sprintf(`SELECT id, name, description, up, down, migrated_at FROM %s ORDER BY migrated_at ASC`, MigrationTable())
	rows, err := db.Query(sql)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	migrations := make([]Migration, 0)
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(
			&migration.ID,
			&migration.Name,
			&migration.Description,
			&migration.Up,
			&migration.Down,
			&migration.MigratedAt); err != nil {
			return migrations, err
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// parseMigrationTmpl parses the sql text as a template injecting .Schema and .MigrationTable variables
func parseMigrationTmpl(sql string) (string, error) {
	// parse up and down migrations
	t, err := template.New("").Parse(sql)
	if err != nil {
		return "", fmt.Errorf("error parsing migration sql: %w", err)
	}

	b := new(strings.Builder)

	if err := t.Execute(b, map[string]string{
		"Schema":         schemaName,
		"MigrationTable": migrationTablename,
	}); err != nil {
		return "", fmt.Errorf("error executing migration template: %w", err)
	}

	return b.String(), nil
}
