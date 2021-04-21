package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210408144217_add_layers_digest_and_repository_id_and_id_unique_constraint",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'unique_layers_digest_and_repository_id_and_id'
							AND table_name = 'layers') THEN
					ALTER TABLE layers
    					ADD CONSTRAINT unique_layers_digest_and_repository_id_and_id UNIQUE (digest, repository_id, id);
				END IF;
				END;
				$$`,
			},
			Down: []string{
				`ALTER TABLE layers
					DROP CONSTRAINT IF EXISTS unique_layers_digest_and_repository_id_and_id CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
