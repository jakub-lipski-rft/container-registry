package migrations

import (
	migrate "github.com/rubenv/sql-migrate"
)

var allMigrations []*Migration

type Migration struct {
	*migrate.Migration
}

func All() []*Migration {
	return allMigrations
}
