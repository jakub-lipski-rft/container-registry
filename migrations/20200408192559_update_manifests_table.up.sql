ALTER TABLE public.manifests
    DROP CONSTRAINT IF EXISTS fk_manifests_repository_id CASCADE,
    DROP CONSTRAINT IF EXISTS uq_manifests_repository_id_digest CASCADE,
    DROP COLUMN IF EXISTS repository_id CASCADE,
    ADD CONSTRAINT uq_manifests_digest UNIQUE (digest);