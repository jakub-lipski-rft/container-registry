ALTER TABLE manifest_lists
    ADD COLUMN repository_id integer NOT NULL,
    ADD CONSTRAINT fk_manifests_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id) ON DELETE CASCADE;
