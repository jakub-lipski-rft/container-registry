package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	// By changing the number of partitions from 1 to 64 we had to drop the existing partition before creating the new
	// ones. As a side effect, FKs for partitioned tables are wiped, so we need to restore them.
	m := &Migration{
		Migration: &migrate.Migration{
			Id: "20210420125123_restore_repository_blobs_fk_constraints",
			Up: []string{
				`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT
							1
						FROM
							information_schema.table_constraints
						WHERE
							constraint_name = 'fk_repository_blobs_blob_digest_blobs'
							AND table_name = 'repository_blobs') THEN
					ALTER TABLE repository_blobs
						ADD CONSTRAINT fk_repository_blobs_blob_digest_blobs FOREIGN KEY (blob_digest) REFERENCES blobs (digest) ON DELETE CASCADE;
				END IF;
				END;
				$$`,
			},
			Down: []string{
				`ALTER TABLE repository_blobs
    				DROP CONSTRAINT IF EXISTS fk_repository_blobs_blob_digest_blobs CASCADE`,
			},
		},
		PostDeployment: false,
	}

	allMigrations = append(allMigrations, m)
}
