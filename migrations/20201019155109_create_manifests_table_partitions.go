package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{Migration: &migrate.Migration{
		Id: "20201019155109_create_manifests_table_partitions",
		Up: []string{
			`CREATE TABLE partitions.manifests_default PARTITION OF public.manifests
			FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
		},
		Down: []string{
			"DROP TABLE IF EXISTS partitions.manifests_default CASCADE",
		},
	}}

	allMigrations = append(allMigrations, m)
}
