ALTER TABLE tags
    DROP CONSTRAINT IF EXISTS chk_one_of_tags_manifest_id_or_manifest_list_id CASCADE,
    DROP CONSTRAINT IF EXISTS fk_tags_manifest_list_id CASCADE,
    ALTER COLUMN manifest_id SET NOT NULL,
    DROP COLUMN IF EXISTS manifest_list_id;