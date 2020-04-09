ALTER TABLE tags
    DROP CONSTRAINT IF EXISTS uq_tags_name_repository_id CASCADE,
    DROP CONSTRAINT IF EXISTS fk_tags_repository_id CASCADE,
    DROP COLUMN IF EXISTS repository_id CASCADE,
    ADD CONSTRAINT uq_tags_manifest_id_name UNIQUE (name, manifest_id);
