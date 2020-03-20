CREATE TABLE IF NOT EXISTS manifests
(
    id               serial    NOT NULL,
    repository_id    integer   NOT NULL,
    schema_version   integer   NOT NULL,
    media_type       text      NOT NULL,
    digest           text      NOT NULL,
    configuration_id integer   NOT NULL,
    payload          json      NOT NULL,
    created_at       timestamp NOT NULL DEFAULT NOW(),
    marked_at        timestamp,
    deleted_at       timestamp,
    CONSTRAINT pk_manifests PRIMARY KEY (id),
    CONSTRAINT fk_manifests_repository_id FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_manifests_configuration_id FOREIGN KEY (configuration_id)
        REFERENCES manifest_configurations (id)
        ON DELETE CASCADE,
    CONSTRAINT uq_manifests_repository_id_digest UNIQUE (repository_id, digest)
);