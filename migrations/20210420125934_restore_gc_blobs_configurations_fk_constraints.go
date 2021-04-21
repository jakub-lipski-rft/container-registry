package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	// By changing the number of partitions from 1 to 64 we had to drop the existing partition before creating the new
	// ones. As a side effect, FKs for partitioned tables are wiped, so we need to restore them.
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210420125934_restore_gc_blobs_configurations_fk_constraints",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_gc_blobs_configurations_digest_blobs'
							AND table_name = 'gc_blobs_configurations') THEN
					ALTER TABLE gc_blobs_configurations
						ADD CONSTRAINT fk_gc_blobs_configurations_digest_blobs FOREIGN KEY (digest) REFERENCES blobs (digest) ON DELETE CASCADE;
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
							constraint_name = 'fk_gc_blobs_configurations_digest_repo_id_and_man_id_manifests'
							AND table_name = 'gc_blobs_configurations') THEN
					ALTER TABLE gc_blobs_configurations
    					ADD CONSTRAINT fk_gc_blobs_configurations_digest_repo_id_and_man_id_manifests FOREIGN KEY (digest, repository_id, manifest_id) REFERENCES manifests (configuration_blob_digest, repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,
			},
			Down: []string{
				`ALTER TABLE gc_blobs_configurations
					DROP CONSTRAINT IF EXISTS fk_gc_blobs_configurations_digest_repo_id_and_man_id_manifests CASCADE`,
				`ALTER TABLE gc_blobs_configurations
    				DROP CONSTRAINT IF EXISTS fk_gc_blobs_configurations_digest_blobs CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
