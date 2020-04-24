BEGIN;

ALTER TABLE manifests
    RENAME COLUMN digest_hex TO digest;

ALTER TABLE manifests
    RENAME CONSTRAINT uq_manifests_digest_hex TO uq_manifests_digest;

ALTER TABLE manifests
    ALTER COLUMN digest TYPE text USING concat('sha256', ':', encode(digest, 'hex'));

COMMIT;