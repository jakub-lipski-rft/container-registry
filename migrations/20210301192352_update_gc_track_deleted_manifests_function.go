package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301192352_update_gc_track_deleted_manifests_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_deleted_manifests ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					IF OLD.configuration_blob_digest IS NOT NULL THEN
						INSERT INTO gc_blob_review_queue (digest, review_after)
							VALUES (OLD.configuration_blob_digest, gc_review_after('manifest_delete'))
						ON CONFLICT (digest)
							DO UPDATE SET
								review_after = gc_review_after('manifest_delete');
					END IF;
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				// Restore previous version from 20210127115646_create_gc_track_deleted_manifests_function.
				`CREATE OR REPLACE FUNCTION gc_track_deleted_manifests ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					IF OLD.configuration_blob_digest IS NOT NULL THEN
						INSERT INTO gc_blob_review_queue (digest)
							VALUES (OLD.configuration_blob_digest)
						ON CONFLICT (digest)
							DO UPDATE SET
								review_after = now() + interval '1 day';
					END IF;
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
