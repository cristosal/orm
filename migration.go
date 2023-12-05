package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/spf13/viper"
)

const (
	// defaultMigrationTable is the name given to the migration table if not overriden by SetMigrationTable
	defaultMigrationTable = "_migrations"

	// defaultSchemaName is the name given to the migration table schema if not overriden by SetSchemaName
	defaultSchemaName = "public"
)

var (
	ErrNoMigration = errors.New("no migration found")

	tableName  string = "_migrations"
	schemaName string = "public"
)

// Migration is a structured change to the database
type Migration struct {
	ID          int64
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	Up          string `mapstructure:"up"`
	Down        string `mapstructure:"down"`
	Position    int64
	MigratedAt  time.Time
}

// MigrationTable returns the fully qualified, schema prefixed table name
func MigrationTable() string {
	return schemaName + "." + tableName
}

// SetMigrationTable sets the default table where migrations will be stored and executed
func SetMigrationTable(table string) {
	if table != "" {
		tableName = table
	}
}

// SetSchema sets the schema for the migration table
func SetSchema(schema string) {
	if schema != "" {
		schemaName = schema
	}
}

// CreateMigrationTable creates the table and schema where migrations will be stored and executed.
// The name of the table can be set using the SetMigrationTable method.
// The name of the schema can be set using the SetSchema method.
func CreateMigrationTable(db DB) error {
	if schemaName == "" {
		schemaName = defaultSchemaName
	}

	if tableName == "" {
		tableName = defaultMigrationTable
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
		position SERIAL NOT NULL,
		migrated_at TIMESTAMPTZ
	);`, MigrationTable()))

	return err
}

// DropMigrationTable
func DropMigrationTable(db DB) error {
	return Exec(db, fmt.Sprintf("DROP TABLE %s", MigrationTable()))
}

// PushMigration adds a migration to the database and executes it
func PushMigration(db DB, migration *Migration) error {
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

	if name == migration.Name {
		// we have already pushed it
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// insert record of the migration
	sql = fmt.Sprintf("INSERT INTO %s (name, description, up, down) VALUES ($1, $2, $3, $4)", MigrationTable())
	if _, err := tx.Exec(sql, migration.Name, migration.Description, migration.Up, migration.Down); err != nil {
		return err
	}

	// execute up migration
	if _, err := tx.Exec(migration.Up); err != nil {
		return err
	}

	// set migration as executed
	sql = fmt.Sprintf("UPDATE %s SET migrated_at = NOW() WHERE name = $1", MigrationTable())
	if _, err := tx.Exec(sql, migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}

// PushMigrations pushes multiple migrations returning the first error encountered
func PushMigrations(db DB, migrations []Migration) error {
	for i := range migrations {
		if err := PushMigration(db, &migrations[i]); err != nil {
			return err
		}
	}

	return nil
}

// PushMigrationFile pushes a migration from a file
func PushMigrationFile(db DB, filepath string) error {
	v := viper.New()
	v.SetConfigFile(filepath)
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	var migration Migration

	if err := v.Unmarshal(&migration); err != nil {
		return err
	}

	return PushMigration(db, &migration)
}

// PushMigrationFileFS pushes a file with given name from the filesystem
func PushMigrationFileFS(db DB, filesystem fs.FS, filepath string) error {
	v := viper.New()

	f, err := filesystem.Open(path.Join(".", filepath))

	if err != nil {
		return err
	}

	defer f.Close()
	ext := path.Ext(filepath)
	v.SetConfigType(ext[1:])

	if err := v.ReadConfig(f); err != nil {
		return err
	}

	var migration Migration
	if err := v.Unmarshal(&migration); err != nil {
		return err
	}

	return PushMigration(db, &migration)
}

// PushMigrationDir pushes all migrations inside a directory
func PushMigrationDir(db DB, dirpath string) error {
	entries, err := os.ReadDir(dirpath)
	if err != nil {
		return err
	}

	for i := range entries {
		filepath := path.Join(dirpath, entries[i].Name())
		if err := PushMigrationFile(db, filepath); err != nil {
			return err
		}
	}

	return nil
}

// PushMigrationDirFS pushes migrations from a directory inside fs
func PushMigrationDirFS(db DB, filesystem fs.FS, dirpath string) error {
	// here is where we read
	entries, err := fs.ReadDir(filesystem, dirpath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		filename := path.Join(dirpath, entry.Name())

		if entry.IsDir() {
			if err := PushMigrationDirFS(db, filesystem, filename); err != nil {
				return err
			}
		} else {
			if err := PushMigrationFileFS(db, filesystem, filename); err != nil {
				return err
			}
		}
	}

	return nil
}

// PushMigrationFS pushes all migrations in a directory using fs.FS
func PushMigrationFS(db DB, filesystem fs.FS) error {
	return PushMigrationDirFS(db, filesystem, ".")
}

// PopMigration reverts the last migration
func PopMigration(db DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	stmt := fmt.Sprintf(`SELECT name, down FROM %s ORDER BY position DESC`, MigrationTable())
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

// PopAllMigrations reverts all migrations
func PopAllMigrations(db DB) (int, error) {
	var n int

	for {
		if err := PopMigration(db); err != nil {
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

// PopMigrationsUntil pops until a migration with given name is reached
func PopMigrationsUntil(db DB, name string) error {
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

		if err := PopMigration(db); err != nil {
			return err
		}
	}
}

// GetLatestMigration returns the latest migration executed
func GetLatestMigration(db DB) (*Migration, error) {
	sql := fmt.Sprintf(`SELECT id, name, description, up, down, position, migrated_at FROM %s ORDER BY position DESC`, MigrationTable())
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
		&mig.Position,
		&mig.MigratedAt); err != nil {
		return nil, err
	}

	return &mig, nil
}

// ListMigrations returns all the executed migrations
func ListMigrations(db DB) ([]Migration, error) {
	sql := fmt.Sprintf(`SELECT id, name, description, up, down, position, migrated_at FROM %s ORDER BY position ASC`, MigrationTable())
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
			&migration.Position,
			&migration.MigratedAt); err != nil {
			return migrations, err
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}
