package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{Migration: &migrate.Migration{
		Id: "20201019155021_create_blobs_table_partitions",
		Up: []string{
			`CREATE TABLE partitions.blobs_default PARTITION OF public.blobs
			FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
		},
		Down: []string{
			"DROP TABLE IF EXISTS partitions.blobs_default CASCADE",
		},
	}}

	allMigrations = append(allMigrations, m)
}
