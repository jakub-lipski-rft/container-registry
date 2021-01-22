package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210119113336_create_gc_track_blob_uploads_function",
			Up: []string{
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
			Down: []string{
				"DROP FUNCTION IF EXISTS gc_track_blob_uploads CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
