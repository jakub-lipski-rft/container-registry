CREATE TABLE IF NOT EXISTS layers
(
    id         serial    NOT NULL,
    media_type text      NOT NULL,
    digest     text      NOT NULL,
    size       bigint    NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    marked_at  timestamp,
    deleted_at timestamp,
    CONSTRAINT pk_layers PRIMARY KEY (id),
    CONSTRAINT uq_layers_digest UNIQUE (digest)
);