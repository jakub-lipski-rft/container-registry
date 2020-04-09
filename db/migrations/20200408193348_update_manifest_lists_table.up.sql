ALTER TABLE manifest_lists
    DROP CONSTRAINT IF EXISTS fk_manifests_repository_id CASCADE,
    DROP COLUMN IF EXISTS repository_id CASCADE;

