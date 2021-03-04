package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301192615_update_gc_track_deleted_manifest_lists_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_deleted_manifest_lists ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_manifest_review_queue (repository_id, manifest_id, review_after)
						VALUES (OLD.repository_id, OLD.child_id, gc_review_after('manifest_list_delete'))
					ON CONFLICT (repository_id, manifest_id)
						DO UPDATE SET
							review_after = gc_review_after('manifest_list_delete');
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				// Restore previous version from 20210127142751_create_gc_track_deleted_manifest_lists_function.
				`CREATE OR REPLACE FUNCTION gc_track_deleted_manifest_lists ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_manifest_review_queue (repository_id, manifest_id)
						VALUES (OLD.repository_id, OLD.child_id)
					ON CONFLICT (repository_id, manifest_id)
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
