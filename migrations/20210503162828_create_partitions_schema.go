package migrations

import (
	migrate "github.com/rubenv/sql-migrate"
)

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210503162828_create_partitions_schema",
			Up: []string{
				"CREATE SCHEMA IF NOT EXISTS partitions",
			},
			Down: []string{
				"DROP SCHEMA IF EXISTS partitions CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
