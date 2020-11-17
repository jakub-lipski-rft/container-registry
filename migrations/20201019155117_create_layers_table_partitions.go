package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20201019155117_create_layers_table_partitions",
			Up: []string{
				`CREATE TABLE partitions.layers_default PARTITION OF public.layers
				FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
			},
			Down: []string{
				"DROP TABLE IF EXISTS partitions.layers_default CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
