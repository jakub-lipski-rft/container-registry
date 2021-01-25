package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210125102717_add_gc_manifest_review_queue_manifests_fk_constraint",
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
	}

	allMigrations = append(allMigrations, m)
}
