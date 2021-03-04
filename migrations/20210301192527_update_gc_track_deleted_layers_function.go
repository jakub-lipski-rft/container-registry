package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301192527_update_gc_track_deleted_layers_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_track_deleted_layers ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_blob_review_queue (digest, review_after)
						VALUES (OLD.digest, gc_review_after('layer_delete'))
					ON CONFLICT (digest)
						DO UPDATE SET
							review_after = gc_review_after('layer_delete');
					RETURN NULL;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				// Restore previous version from 20210127141212_create_gc_track_deleted_layers_function.
				`CREATE OR REPLACE FUNCTION gc_track_deleted_layers ()
					RETURNS TRIGGER
					AS $$
				BEGIN
					INSERT INTO gc_blob_review_queue (digest)
						VALUES (OLD.digest)
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
