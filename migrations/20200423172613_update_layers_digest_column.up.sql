BEGIN;

ALTER TABLE layers
    RENAME COLUMN digest TO digest_hex;

ALTER TABLE layers
    RENAME CONSTRAINT uq_layers_digest TO uq_layers_digest_hex;

ALTER TABLE layers
    ALTER COLUMN digest_hex TYPE bytea USING decode(split_part(digest_hex, ':', 2), 'hex');

COMMIT;