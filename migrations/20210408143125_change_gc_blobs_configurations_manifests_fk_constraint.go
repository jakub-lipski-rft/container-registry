package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210408143125_change_gc_blobs_configurations_manifests_fk_constraint",
			Up: []string{
				`ALTER TABLE gc_blobs_configurations
					DROP CONSTRAINT IF EXISTS fk_gc_blobs_configurations_repo_id_and_manifest_id_manifests`,

				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_gc_blobs_configurations_digest_repo_id_and_man_id_manifests'
							AND table_name = 'gc_blobs_configurations') THEN
					ALTER TABLE gc_blobs_configurations
    					ADD CONSTRAINT fk_gc_blobs_configurations_digest_repo_id_and_man_id_manifests FOREIGN KEY (digest, repository_id, manifest_id) REFERENCES manifests (configuration_blob_digest, repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,

				`CREATE INDEX IF NOT EXISTS index_gc_blobs_configurations_on_digest_and_repo_id_and_man_id ON gc_blobs_configurations USING btree (digest, repository_id, manifest_id)`,
			},
			Down: []string{
				"DROP INDEX IF EXISTS index_gc_blobs_configurations_on_digest_and_repo_id_and_man_id CASCADE",

				`ALTER TABLE gc_blobs_configurations
					DROP CONSTRAINT IF EXISTS fk_gc_blobs_configurations_digest_repo_id_and_man_id_manifests CASCADE`,

				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_gc_blobs_configurations_repo_id_and_manifest_id_manifests'
							AND table_name = 'gc_blobs_configurations') THEN
					ALTER TABLE gc_blobs_configurations
    					ADD CONSTRAINT fk_gc_blobs_configurations_repo_id_and_manifest_id_manifests FOREIGN KEY (repository_id, manifest_id) REFERENCES manifests (repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
