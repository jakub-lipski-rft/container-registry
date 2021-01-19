package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210119105448_create_gc_manifest_review_queue_table",
			Up: []string{
				`CREATE TABLE IF NOT EXISTS gc_manifest_review_queue (
					repository_id bigint NOT NULL,
					manifest_id bigint NOT NULL,
					review_after timestamp with time zone NOT NULL DEFAULT now() + interval '1 day',
					review_count integer NOT NULL DEFAULT 0,
					CONSTRAINT pk_gc_manifest_review_queue PRIMARY KEY (repository_id, manifest_id)
				)`,
				"CREATE INDEX IF NOT EXISTS index_gc_manifest_review_queue_on_review_after ON gc_manifest_review_queue USING btree (review_after)",
			},
			Down: []string{
				"DROP INDEX IF EXISTS index_gc_manifest_review_queue_on_review_after CASCADE",
				"DROP TABLE IF EXISTS gc_manifest_review_queue CASCADE",
			},
		},
	}

	allMigrations = append(allMigrations, m)
}
