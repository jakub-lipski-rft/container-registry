CREATE TABLE IF NOT EXISTS tags
(
    id               bigint    NOT NULL GENERATED BY DEFAULT AS IDENTITY,
    name             text      NOT NULL,
    repository_id    bigint    NOT NULL,
    manifest_id      bigint,
    manifest_list_id bigint,
    created_at       timestamp with time zone NOT NULL DEFAULT now(),
    updated_at       timestamp with time zone,
    deleted_at       timestamp with time zone,
    CONSTRAINT pk_tags PRIMARY KEY (id),
    CONSTRAINT fk_tags_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id) ON DELETE CASCADE,
    CONSTRAINT fk_tags_manifest_id FOREIGN KEY (manifest_id)
        REFERENCES manifests (id) ON DELETE CASCADE,
    CONSTRAINT fk_tags_manifest_list_id FOREIGN KEY (manifest_list_id)
        REFERENCES manifest_lists (id) ON DELETE CASCADE,
    CONSTRAINT uq_tags_name_repository_id UNIQUE (name, repository_id),
    CONSTRAINT chk_tags_manifest_id_manifest_list_id CHECK (((manifest_id IS NOT NULL AND manifest_list_id IS NULL) OR
                                                             (manifest_id IS NULL AND manifest_list_id IS NOT NULL)))
);