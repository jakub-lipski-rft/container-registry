BEGIN;

ALTER TABLE manifest_configurations
    RENAME COLUMN digest TO digest_hex;

ALTER TABLE manifest_configurations
    RENAME CONSTRAINT uq_manifest_configurations_digest TO uq_manifest_configurations_digest_hex;

ALTER TABLE manifest_configurations
    ALTER COLUMN digest_hex TYPE bytea USING decode(split_part(digest_hex, ':', 2), 'hex');

COMMIT;