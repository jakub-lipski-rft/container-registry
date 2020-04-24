BEGIN;

ALTER TABLE manifest_lists
    RENAME COLUMN digest TO digest_hex;

ALTER TABLE manifest_lists
    RENAME CONSTRAINT uq_manifest_lists_digest TO uq_manifest_lists_digest_hex;

ALTER TABLE manifest_lists
    ALTER COLUMN digest_hex TYPE bytea USING decode(split_part(digest_hex, ':', 2), 'hex');

COMMIT;