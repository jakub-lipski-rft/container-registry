BEGIN;

ALTER TABLE manifest_configurations
    RENAME COLUMN digest_hex TO digest;

ALTER TABLE manifest_configurations
    RENAME CONSTRAINT uq_manifest_configurations_digest_hex TO uq_manifest_configurations_digest;

ALTER TABLE manifest_configurations
    ALTER COLUMN digest TYPE text USING concat('sha256', ':', encode(digest, 'hex'));

COMMIT;