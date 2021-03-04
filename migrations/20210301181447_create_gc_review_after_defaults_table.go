package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301181447_create_gc_review_after_defaults_table",
			Up: []string{
				`CREATE TABLE IF NOT EXISTS gc_review_after_defaults (
					event text NOT NULL,
					value interval NOT NULL,
					CONSTRAINT pk_gc_review_after_defaults PRIMARY KEY (event),
					CONSTRAINT check_gc_review_after_defaults_event_length CHECK ((char_length(event) <= 255))
				)`,
			},
			Down: []string{
				"DROP TABLE IF EXISTS gc_review_after_defaults CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
