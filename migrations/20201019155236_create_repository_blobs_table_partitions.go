package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20201019155236_create_repository_blobs_table_partitions",
			Up: []string{
				`CREATE TABLE partitions.repository_blobs_default PARTITION OF public.repository_blobs
				FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
			},
			Down: []string{
				"DROP TABLE IF EXISTS partitions.repository_blobs_default CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
