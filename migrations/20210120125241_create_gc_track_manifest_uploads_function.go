package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210120125241_create_gc_track_manifest_uploads_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_manifest_uploads ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_manifest_review_queue (repository_id, manifest_id)
						VALUES (NEW.repository_id, NEW.id);
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				"DROP FUNCTION IF EXISTS gc_track_manifest_uploads CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
