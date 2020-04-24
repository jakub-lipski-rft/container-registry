BEGIN;

ALTER TABLE layers
    RENAME COLUMN digest_hex TO digest;

ALTER TABLE layers
    RENAME CONSTRAINT uq_layers_digest_hex TO uq_layers_digest;

ALTER TABLE layers
    ALTER COLUMN digest TYPE text USING concat('sha256', ':', encode(digest, 'hex'));

COMMIT;