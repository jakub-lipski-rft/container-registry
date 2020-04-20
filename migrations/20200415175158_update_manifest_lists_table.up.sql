ALTER TABLE manifest_lists
    ADD COLUMN digest text NOT NULL,
    ADD CONSTRAINT uq_manifest_lists_digest UNIQUE (digest);