package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{Migration: &migrate.Migration{
		Id: "20201019155128_create_manifest_references_table_partitions",
		Up: []string{
			`CREATE TABLE partitions.manifest_references_default PARTITION OF public.manifest_references
			FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
		},
		Down: []string{
			"DROP TABLE IF EXISTS partitions.manifest_references_default CASCADE",
		},
	}}

	allMigrations = append(allMigrations, m)
}
