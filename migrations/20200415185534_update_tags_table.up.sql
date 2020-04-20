ALTER TABLE tags
    ADD COLUMN manifest_list_id integer,
    ALTER COLUMN manifest_id DROP NOT NULL,
    ADD CONSTRAINT fk_tags_manifest_list_id FOREIGN KEY (manifest_list_id)
        REFERENCES manifest_lists (id) ON DELETE CASCADE,
    ADD CONSTRAINT chk_one_of_tags_manifest_id_or_manifest_list_id CHECK (
            (manifest_id IS NOT NULL AND manifest_list_id IS NULL) OR
            (manifest_id IS NULL AND manifest_list_id IS NOT NULL));