package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210120125245_create_gc_track_manifest_uploads_trigger",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							pg_trigger
						WHERE
							tgname = 'gc_track_manifest_uploads_trigger') THEN
						CREATE TRIGGER gc_track_manifest_uploads_trigger
							AFTER INSERT ON manifests
							FOR EACH ROW
							EXECUTE PROCEDURE gc_track_manifest_uploads ();
					END IF;
				END
				$$`,
			},
			Down: []string{
				"DROP TRIGGER IF EXISTS gc_track_manifest_uploads_trigger ON blobs CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
