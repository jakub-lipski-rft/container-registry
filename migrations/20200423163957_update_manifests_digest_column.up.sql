BEGIN;

ALTER TABLE manifests
    RENAME COLUMN digest TO digest_hex;

ALTER TABLE manifests
    RENAME CONSTRAINT uq_manifests_digest TO uq_manifests_digest_hex;

ALTER TABLE manifests
    ALTER COLUMN digest_hex TYPE bytea USING decode(split_part(digest_hex, ':', 2), 'hex');

COMMIT;