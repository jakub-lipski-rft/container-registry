package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{Migration: &migrate.Migration{
		Id: "20201019155144_create_tags_table_partitions",
		Up: []string{
			`CREATE TABLE partitions.tags_default PARTITION OF public.tags
			FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
		},
		Down: []string{
			"DROP TABLE IF EXISTS partitions.tags_default CASCADE",
		},
	}}

	allMigrations = append(allMigrations, m)
}
