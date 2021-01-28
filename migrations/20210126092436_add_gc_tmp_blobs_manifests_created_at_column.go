package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210126092436_add_gc_tmp_blobs_manifests_created_at_column",
			Up: []string{
				`ALTER TABLE gc_tmp_blobs_manifests
					ADD COLUMN IF NOT EXISTS created_at timestamp WITH time zone NOT NULL DEFAULT now()`,
			},
			Down: []string{
				"ALTER TABLE gc_tmp_blobs_manifests DROP COLUMN IF EXISTS created_at CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
