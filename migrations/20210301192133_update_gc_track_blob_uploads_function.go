package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301192133_update_gc_track_blob_uploads_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_blob_uploads ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_blob_review_queue (digest, review_after)
						VALUES (NEW.digest, gc_review_after('blob_upload'))
					ON CONFLICT (digest)
						DO UPDATE SET
							review_after = gc_review_after('blob_upload');
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				// Restore previous version from 20210119113336_create_gc_track_blob_uploads_function.
				`CREATE OR REPLACE FUNCTION gc_track_blob_uploads ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_blob_review_queue (digest)
						VALUES (NEW.digest)
					ON CONFLICT (digest)
						DO UPDATE SET
							review_after = now() + interval '1 day';
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
