BEGIN;
CREATE INDEX IF NOT EXISTS ix_repository_layers_repository_id ON repository_layers (repository_id);
CREATE INDEX IF NOT EXISTS ix_repository_layers_layer_id ON repository_layers (layer_id);
COMMIT;