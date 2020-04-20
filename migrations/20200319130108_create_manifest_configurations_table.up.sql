CREATE TABLE IF NOT EXISTS manifest_configurations
(
    id         serial    NOT NULL,
    media_type text      NOT NULL,
    digest     text      NOT NULL,
    size       bigint    NOT NULL,
    payload    json      NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    deleted_at timestamp,
    CONSTRAINT pk_manifest_configs PRIMARY KEY (id),
    CONSTRAINT uq_manifest_configurations_digest UNIQUE (digest)
);