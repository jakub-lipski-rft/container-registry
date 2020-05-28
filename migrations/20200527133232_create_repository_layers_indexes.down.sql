BEGIN;
DROP INDEX IF EXISTS ix_repository_layers_repository_id;
DROP INDEX IF EXISTS ix_repository_layers_layer_id;
COMMIT;