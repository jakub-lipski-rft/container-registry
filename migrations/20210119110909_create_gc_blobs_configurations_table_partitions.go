package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210119110909_create_gc_blobs_configurations_table_partitions",
			Up: []string{
				`CREATE TABLE IF NOT EXISTS partitions.gc_blobs_configurations_default PARTITION OF public.gc_blobs_configurations
				FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
			},
			Down: []string{
				"DROP TABLE IF EXISTS partitions.gc_blobs_configurations_default CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
