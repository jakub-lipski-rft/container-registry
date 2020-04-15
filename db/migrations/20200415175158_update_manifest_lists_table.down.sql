ALTER TABLE manifest_lists
    DROP CONSTRAINT IF EXISTS uq_manifest_lists_digest CASCADE,
    DROP COLUMN IF EXISTS digest CASCADE;