ALTER TABLE tags
    DROP CONSTRAINT IF EXISTS uq_tags_manifest_id_name CASCADE,
    ADD COLUMN repository_id integer NOT NULL,
    ADD CONSTRAINT uq_tags_name_repository_id UNIQUE (name,repository_id),
    ADD CONSTRAINT fk_tags_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id) MATCH SIMPLE
        ON DELETE CASCADE;