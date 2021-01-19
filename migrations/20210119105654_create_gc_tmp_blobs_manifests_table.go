package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210119105654_create_gc_tmp_blobs_manifests_table",
			Up: []string{
				`CREATE TABLE IF NOT EXISTS gc_tmp_blobs_manifests (
					digest bytea NOT NULL,
					CONSTRAINT pk_gc_tmp_blobs_manifests PRIMARY KEY (digest)
				)`,
			},
			Down: []string{
				"DROP TABLE IF EXISTS gc_tmp_blobs_manifests CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
