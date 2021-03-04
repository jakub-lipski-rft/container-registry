package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301192307_update_gc_track_manifest_uploads_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_manifest_uploads ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_manifest_review_queue (repository_id, manifest_id, review_after)
						VALUES (NEW.repository_id, NEW.id, gc_review_after('manifest_upload'));
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				// Restore previous version from 20210120125241_create_gc_track_manifest_uploads_function.
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
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
