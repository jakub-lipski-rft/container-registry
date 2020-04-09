CREATE TABLE IF NOT EXISTS repository_manifests
(
    id            serial    NOT NULL,
    repository_id integer   NOT NULL,
    manifest_id   integer   NOT NULL,
    created_at    timestamp NOT NULL DEFAULT NOW(),
    deleted_at    timestamp,
    CONSTRAINT pk_repository_manifests PRIMARY KEY (id),
    CONSTRAINT uq_repository_manifests_repository_id_manifest_id UNIQUE (repository_id, manifest_id),
    CONSTRAINT fk_repository_manifests_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id) ON DELETE CASCADE,
    CONSTRAINT fk_repository_manifests_manifest_id FOREIGN KEY (manifest_id)
        REFERENCES manifests (id) ON DELETE CASCADE
);