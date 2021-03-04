package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210301185307_create_gc_review_after_function",
			Up: []string{
				`CREATE OR REPLACE FUNCTION gc_review_after (e text)
					RETURNS timestamp WITH time zone VOLATILE
					AS $$
				DECLARE
					result timestamp WITH time zone;
				BEGIN
					SELECT
						(now() + value) INTO result
					FROM
						gc_review_after_defaults
					WHERE
						event = e;
					IF result IS NULL THEN
						RETURN now() + interval '1 day';
					ELSE
						RETURN result;
					END IF;
				END;
				$$
				LANGUAGE plpgsql`,
			},
			Down: []string{
				"DROP FUNCTION IF EXISTS gc_review_after CASCADE",
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
