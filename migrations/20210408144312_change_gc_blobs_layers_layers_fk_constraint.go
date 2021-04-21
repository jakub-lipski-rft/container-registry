package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210408144312_change_gc_blobs_layers_layers_fk_constraint",
			Up: []string{
				`ALTER TABLE gc_blobs_layers
					DROP CONSTRAINT IF EXISTS fk_gc_blobs_layers_repository_id_and_layer_id_layers`,

				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_gc_blobs_layers_digest_repository_id_and_layer_id_layers'
							AND table_name = 'gc_blobs_layers') THEN
					ALTER TABLE gc_blobs_layers
    					ADD CONSTRAINT fk_gc_blobs_layers_digest_repository_id_and_layer_id_layers FOREIGN KEY (digest, repository_id, layer_id) REFERENCES layers (digest, repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,

				`CREATE INDEX IF NOT EXISTS index_gc_blobs_layers_on_digest_and_repository_id_and_layer_id ON gc_blobs_layers USING btree (digest, repository_id, layer_id)`,
			},
			Down: []string{
				"DROP INDEX IF EXISTS index_gc_blobs_layers_on_digest_and_repository_id_and_layer_id CASCADE",

				`ALTER TABLE gc_blobs_layers
					DROP CONSTRAINT IF EXISTS fk_gc_blobs_layers_digest_repository_id_and_layer_id_layers CASCADE`,

				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_gc_blobs_layers_repository_id_and_layer_id_layers'
							AND table_name = 'gc_blobs_layers') THEN
					ALTER TABLE gc_blobs_layers
    					ADD CONSTRAINT fk_gc_blobs_layers_repository_id_and_layer_id_layers FOREIGN KEY (repository_id, layer_id) REFERENCES layers (repository_id, id) ON DELETE CASCADE;
				END IF;
				END;
				$$`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
