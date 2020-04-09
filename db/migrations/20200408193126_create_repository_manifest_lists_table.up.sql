CREATE TABLE IF NOT EXISTS repository_manifest_lists
(
    id               serial    NOT NULL,
    repository_id    integer   NOT NULL,
    manifest_list_id integer   NOT NULL,
    created_at       timestamp NOT NULL DEFAULT NOW(),
    deleted_at       timestamp,
    CONSTRAINT pk_repository_manifest_lists PRIMARY KEY (id),
    CONSTRAINT uq_repository_manifests_repository_id_manifest_list_id UNIQUE (repository_id, manifest_list_id),
    CONSTRAINT fk_repository_manifest_lists_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id) ON DELETE CASCADE,
    CONSTRAINT fk_repository_manifest_lists_manifest_list_id FOREIGN KEY (manifest_list_id)
        REFERENCES manifest_lists (id) ON DELETE CASCADE
);