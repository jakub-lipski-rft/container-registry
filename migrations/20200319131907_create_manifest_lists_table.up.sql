CREATE TABLE IF NOT EXISTS manifest_lists
(
    id             serial    NOT NULL,
    repository_id  integer   NOT NULL,
    schema_version integer   NOT NULL,
    media_type     text,
    payload        json      NOT NULL,
    created_at     timestamp NOT NULL DEFAULT NOW(),
    marked_at      timestamp,
    deleted_at     timestamp,
    CONSTRAINT pk_manifest_lists PRIMARY KEY (id),
    CONSTRAINT fk_manifests_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE
);