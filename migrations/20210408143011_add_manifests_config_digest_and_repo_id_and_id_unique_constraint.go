package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210408143011_add_manifests_config_digest_and_repo_id_and_id_unique_constraint",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'unique_manifests_configuration_blob_digest_and_repo_id_and_id'
							AND table_name = 'manifests') THEN
					ALTER TABLE manifests
    					ADD CONSTRAINT unique_manifests_configuration_blob_digest_and_repo_id_and_id UNIQUE (configuration_blob_digest, repository_id, id);
				END IF;
				END;
				$$`,
			},
			Down: []string{
				`ALTER TABLE manifests
					DROP CONSTRAINT IF EXISTS unique_manifests_configuration_blob_digest_and_repo_id_and_id CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
