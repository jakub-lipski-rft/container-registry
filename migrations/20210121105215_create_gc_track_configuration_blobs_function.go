package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210121105215_create_gc_track_configuration_blobs_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_configuration_blobs ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					IF NEW.configuration_blob_digest IS NOT NULL THEN
						INSERT INTO gc_blobs_configurations (repository_id, manifest_id, digest)
							VALUES (NEW.repository_id, NEW.id, NEW.configuration_blob_digest)
						ON CONFLICT (digest, manifest_id)
							DO NOTHING;
					END IF;
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				"DROP FUNCTION IF EXISTS gc_track_configuration_blobs CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
