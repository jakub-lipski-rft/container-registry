CREATE TABLE IF NOT EXISTS manifest_list_items
(
    id               serial    NOT NULL,
    manifest_list_id integer   NOT NULL,
    manifest_id      integer   NOT NULL,
    created_at       timestamp NOT NULL DEFAULT NOW(),
    deleted_at       timestamp,
    CONSTRAINT pk_manifest_list_items PRIMARY KEY (id),
    CONSTRAINT fk_manifest_list_items_manifest_list_id FOREIGN KEY (manifest_list_id)
        REFERENCES manifest_lists (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_manifest_list_items_manifest_id FOREIGN KEY (manifest_id)
        REFERENCES manifests (id),
    CONSTRAINT uq_manifest_list_items_manifest_list_id_manifest_id UNIQUE (manifest_list_id, manifest_id)
);