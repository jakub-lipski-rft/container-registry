package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210503153313_create_gc_tmp_blobs_manifests_table",
			Up: []string{
				`CREATE TABLE IF NOT EXISTS gc_tmp_blobs_manifests (
					created_at timestamp WITH time zone NOT NULL DEFAULT now(),
					digest bytea NOT NULL,
					CONSTRAINT pk_gc_tmp_blobs_manifests PRIMARY KEY (digest)
				)`,
			},
			Down: []string{
				"DROP TABLE IF EXISTS gc_tmp_blobs_manifests CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
