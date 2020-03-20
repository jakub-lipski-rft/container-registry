CREATE TABLE IF NOT EXISTS repositories
(
    id         serial    NOT NULL,
    name       text      NOT NULL,
    path       text      NOT NULL,
    parent_id  integer,
    created_at timestamp NOT NULL DEFAULT NOW(),
    deleted_at timestamp,
    CONSTRAINT pk_repositories PRIMARY KEY (id),
    CONSTRAINT fk_repositories_parent_id FOREIGN KEY (parent_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,
    CONSTRAINT uq_repositories_path UNIQUE (path)
);