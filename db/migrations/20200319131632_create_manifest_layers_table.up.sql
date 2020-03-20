CREATE TABLE IF NOT EXISTS manifest_layers
(
    id          serial    NOT NULL,
    manifest_id integer   NOT NULL,
    layer_id    integer   NOT NULL,
    created_at  timestamp NOT NULL DEFAULT NOW(),
    marked_at   timestamp,
    deleted_at  timestamp,
    CONSTRAINT pk_manifest_layers PRIMARY KEY (id),
    CONSTRAINT fk_manifest_layers_manifest_id FOREIGN KEY (manifest_id)
        REFERENCES manifests (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_manifest_layers_layer_id FOREIGN KEY (layer_id)
        REFERENCES layers (id),
    CONSTRAINT uq_manifest_layers_manifest_id_layer_id UNIQUE (manifest_id, layer_id)
);