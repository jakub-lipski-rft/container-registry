package migrations

import (
	"database/sql"
	"strings"

	migrate "github.com/rubenv/sql-migrate"
)

const (
	migrationTableName = "schema_migrations"
	dialect            = "postgres"
)

func init() {
	migrate.SetTable(migrationTableName)
}

type migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *migrator {
	return &migrator{db: db}
}

func migrateSource() migrate.MigrationSource {
	return &migrate.MemoryMigrationSource{Migrations: allMigrations}
}

// versionFromID splits a migration ID in the form of `<version>_<name>` and returns version.
func versionFromID(ID string) string {
	return strings.Split(ID, "_")[0]
}

// Version returns the current applied migration version (if any).
func (m *migrator) Version() (string, error) {
	records, err := migrate.GetMigrationRecords(m.db, dialect)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", nil
	}

	id := records[len(records)-1].Id
	return versionFromID(id), nil
}

// LatestVersion identifies the version of the most recent migration in the repository (if any).
func (m *migrator) LatestVersion() (string, error) {
	all, err := migrateSource().FindMigrations()
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", nil
	}

	id := all[len(all)-1].Id
	return versionFromID(id), nil
}

func (m *migrator) migrate(direction migrate.MigrationDirection, limit int) error {
	_, err := migrate.ExecMax(m.db, dialect, migrateSource(), direction, limit)
	return err
}

// Up applies all non-applied up migrations.
func (m *migrator) Up() error {
	return m.migrate(migrate.Up, 0)
}

// Down applies all non-applied down migrations.
func (m *migrator) Down() error {
	return m.migrate(migrate.Down, 0)
}
