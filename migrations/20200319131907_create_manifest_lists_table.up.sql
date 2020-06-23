CREATE TABLE IF NOT EXISTS manifest_lists
(
    id             bigint                   NOT NULL GENERATED BY DEFAULT AS IDENTITY,
    schema_version integer                  NOT NULL,
    media_type     text,
    digest_hex     bytea                    NOT NULL,
    payload        bytea                    NOT NULL,
    created_at     timestamp with time zone NOT NULL DEFAULT now(),
    marked_at      timestamp with time zone,
    CONSTRAINT pk_manifest_lists PRIMARY KEY (id),
    CONSTRAINT uq_manifest_lists_digest_hex UNIQUE (digest_hex),
    CONSTRAINT ck_manifest_lists_media_type_length CHECK ((char_length(media_type) <= 255))
);