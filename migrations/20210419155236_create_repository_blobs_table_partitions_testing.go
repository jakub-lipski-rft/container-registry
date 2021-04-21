// +build integration

package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210419155236_create_repository_blobs_table_partitions_testing",
			Up: []string{
				"DROP TABLE IF EXISTS partitions.repository_blobs_default CASCADE",
				"CREATE TABLE IF NOT EXISTS partitions.repository_blobs_p_0 PARTITION OF public.repository_blobs FOR VALUES WITH (MODULUS 4, REMAINDER 0)",
				"CREATE TABLE IF NOT EXISTS partitions.repository_blobs_p_1 PARTITION OF public.repository_blobs FOR VALUES WITH (MODULUS 4, REMAINDER 1)",
				"CREATE TABLE IF NOT EXISTS partitions.repository_blobs_p_2 PARTITION OF public.repository_blobs FOR VALUES WITH (MODULUS 4, REMAINDER 2)",
				"CREATE TABLE IF NOT EXISTS partitions.repository_blobs_p_3 PARTITION OF public.repository_blobs FOR VALUES WITH (MODULUS 4, REMAINDER 3)",
			},
			Down: []string{
				"DROP TABLE IF EXISTS partitions.repository_blobs_p_0 CASCADE",
				"DROP TABLE IF EXISTS partitions.repository_blobs_p_1 CASCADE",
				"DROP TABLE IF EXISTS partitions.repository_blobs_p_2 CASCADE",
				"DROP TABLE IF EXISTS partitions.repository_blobs_p_3 CASCADE",
				"CREATE TABLE IF NOT EXISTS partitions.repository_blobs_default PARTITION OF public.repository_blobs FOR VALUES WITH (MODULUS 1, REMAINDER 0)",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
