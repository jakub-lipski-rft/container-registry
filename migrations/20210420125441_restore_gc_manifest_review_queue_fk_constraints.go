package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	// By changing the number of partitions from 1 to 64 we had to drop the existing partition before creating the new
	// ones. As a side effect, FKs for partitioned tables are wiped, so we need to restore them.
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210420125441_restore_gc_manifest_review_queue_fk_constraints",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_gc_manifest_review_queue_repo_id_and_manifest_id_manifests'
							AND table_name = 'gc_manifest_review_queue') THEN
					ALTER TABLE gc_manifest_review_queue
						ADD CONSTRAINT fk_gc_manifest_review_queue_repo_id_and_manifest_id_manifests FOREIGN KEY (repository_id, manifest_id) REFERENCES manifests (repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,
			},
			Down: []string{
				`ALTER TABLE gc_manifest_review_queue
    				DROP CONSTRAINT IF EXISTS fk_gc_manifest_review_queue_repo_id_and_manifest_id_manifests CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
