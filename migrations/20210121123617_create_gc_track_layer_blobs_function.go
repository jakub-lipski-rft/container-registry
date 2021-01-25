package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210121123617_create_gc_track_layer_blobs_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_layer_blobs ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_blobs_layers (repository_id, layer_id, digest)
						VALUES (NEW.repository_id, NEW.id, NEW.digest)
					ON CONFLICT (digest, layer_id)
						DO NOTHING;
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				"DROP FUNCTION IF EXISTS gc_track_layer_blobs CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
