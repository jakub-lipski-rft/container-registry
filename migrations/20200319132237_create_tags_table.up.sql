CREATE TABLE IF NOT EXISTS tags
(
    id          serial    NOT NULL,
    name        text      NOT NULL,
    manifest_id integer   NOT NULL,
    created_at  timestamp NOT NULL DEFAULT NOW(),
    updated_at  timestamp,
    deleted_at  timestamp,
    CONSTRAINT pk_tags PRIMARY KEY (id),
    CONSTRAINT fk_tags_manifest_id FOREIGN KEY (manifest_id)
        REFERENCES manifests (id)
        ON DELETE CASCADE,
    CONSTRAINT uq_tags_manifest_id_name UNIQUE (name, manifest_id)
);