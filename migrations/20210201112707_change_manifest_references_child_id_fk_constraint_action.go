package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210201112707_change_manifest_references_child_id_fk_constraint_action",
			Up: []string{
				`ALTER TABLE manifest_references
					DROP CONSTRAINT IF EXISTS fk_manifest_references_repository_id_and_child_id_manifests CASCADE,
					ADD CONSTRAINT fk_manifest_references_repository_id_and_child_id_manifests FOREIGN KEY (repository_id, child_id) REFERENCES manifests (repository_id, id) ON DELETE RESTRICT`,
			},
			Down: []string{
				`ALTER TABLE manifest_references
					DROP CONSTRAINT IF EXISTS fk_manifest_references_repository_id_and_child_id_manifests CASCADE,
					ADD CONSTRAINT fk_manifest_references_repository_id_and_child_id_manifests FOREIGN KEY (repository_id, child_id) REFERENCES manifests (repository_id, id) ON DELETE CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
