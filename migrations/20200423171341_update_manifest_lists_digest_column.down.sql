BEGIN;

ALTER TABLE manifest_lists
    RENAME COLUMN digest_hex TO digest;

ALTER TABLE manifest_lists
    RENAME CONSTRAINT uq_manifest_lists_digest_hex TO uq_manifest_lists_digest;

ALTER TABLE manifest_lists
    ALTER COLUMN digest TYPE text USING concat('sha256', ':', encode(digest, 'hex'));

COMMIT;