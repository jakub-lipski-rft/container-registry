package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301192713_update_gc_track_switched_tags_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_switched_tags ()
					RETURNS TRIGGER
					AS $$
				BEGIN
    				INSERT INTO gc_manifest_review_queue (repository_id, manifest_id, review_after)
						VALUES (OLD.repository_id, OLD.manifest_id, gc_review_after('tag_switch'))
					ON CONFLICT (repository_id, manifest_id)
						DO UPDATE SET
							review_after = gc_review_after('tag_switch');
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				// Restore previous version from 20210129124240_create_gc_track_switched_tags_function.
				`CREATE OR REPLACE FUNCTION gc_track_switched_tags ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_manifest_review_queue (repository_id, manifest_id)
						VALUES (OLD.repository_id, OLD.manifest_id)
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
