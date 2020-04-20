ALTER TABLE manifests
    DROP CONSTRAINT IF EXISTS uq_manifests_digest CASCADE,
    ADD COLUMN repository_id integer NOT NULL,
    ADD CONSTRAINT fk_manifests_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,
    ADD CONSTRAINT uq_manifests_repository_id_digest UNIQUE (repository_id, digest)
