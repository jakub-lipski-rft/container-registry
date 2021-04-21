package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	// By changing the number of partitions from 1 to 64 we had to drop the existing partition before creating the new
	// ones. As a side effect, FKs for partitioned tables are wiped, so we need to restore them.
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210420124634_restore_manifest_references_fk_constraints",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_manifest_references_repository_id_and_child_id_manifests'
							AND table_name = 'manifest_references') THEN
					ALTER TABLE manifest_references
						ADD CONSTRAINT fk_manifest_references_repository_id_and_child_id_manifests FOREIGN KEY (repository_id, child_id) REFERENCES manifests (repository_id, id) ON DELETE RESTRICT;
				END IF;
				END;
				$$`,
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_manifest_references_repository_id_and_parent_id_manifests'
							AND table_name = 'manifest_references') THEN
					ALTER TABLE manifest_references
						ADD CONSTRAINT fk_manifest_references_repository_id_and_parent_id_manifests FOREIGN KEY (repository_id, parent_id) REFERENCES manifests (repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,
			},
			Down: []string{
				`ALTER TABLE manifest_references
    				DROP CONSTRAINT IF EXISTS fk_manifest_references_repository_id_and_parent_id_manifests CASCADE`,
				`ALTER TABLE manifest_references
    				DROP CONSTRAINT IF EXISTS fk_manifest_references_repository_id_and_child_id_manifests CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
