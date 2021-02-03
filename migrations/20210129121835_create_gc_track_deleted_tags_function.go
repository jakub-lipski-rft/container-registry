package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210129121835_create_gc_track_deleted_tags_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_deleted_tags ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					IF EXISTS (
						SELECT
							1
						FROM
							manifests
						WHERE
							repository_id = OLD.repository_id
							AND id = OLD.manifest_id) THEN
						INSERT INTO gc_manifest_review_queue (repository_id, manifest_id)
							VALUES (OLD.repository_id, OLD.manifest_id)
						ON CONFLICT (repository_id, manifest_id)
							DO UPDATE SET
								review_after = now() + interval '1 day';
					END IF;
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				"DROP FUNCTION IF EXISTS gc_track_deleted_tags CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
