-- Database generated with pgModeler (PostgreSQL Database Modeler).
-- pgModeler  version: 0.9.2
-- PostgreSQL version: 12.0
-- Project Site: pgmodeler.io
-- Model Author: ---

-- Database creation must be done outside a multicommand file.
-- These commands were put in this file only as a convenience.
-- -- object: registry | type: DATABASE --
-- -- DROP DATABASE IF EXISTS registry;
-- CREATE DATABASE registry
-- 	ENCODING = 'UTF8'
-- 	LC_COLLATE = 'en_US.UTF-8'
-- 	LC_CTYPE = 'en_US.UTF-8'
-- 	TABLESPACE = pg_default
-- 	OWNER = postgres;
-- -- ddl-end --
-- 

-- object: public.repositories_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.repositories_id_seq CASCADE;
CREATE SEQUENCE public.repositories_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.repositories_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.repositories | type: TABLE --
-- DROP TABLE IF EXISTS public.repositories CASCADE;
CREATE TABLE public.repositories (
	id integer NOT NULL DEFAULT nextval('public.repositories_id_seq'::regclass),
	name text NOT NULL,
	path text NOT NULL,
	parent_id integer,
	created_at timestamp NOT NULL DEFAULT now(),
	deleted_at timestamp,
	CONSTRAINT pk_repositories PRIMARY KEY (id),
	CONSTRAINT uq_repositories_path UNIQUE (path)

);
-- ddl-end --
-- ALTER TABLE public.repositories OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_configurations_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.manifest_configurations_id_seq CASCADE;
CREATE SEQUENCE public.manifest_configurations_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.manifest_configurations_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_configurations | type: TABLE --
-- DROP TABLE IF EXISTS public.manifest_configurations CASCADE;
CREATE TABLE public.manifest_configurations (
	id integer NOT NULL DEFAULT nextval('public.manifest_configurations_id_seq'::regclass),
	media_type text NOT NULL,
	digest text NOT NULL,
	size bigint NOT NULL,
	payload json NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	deleted_at timestamp,
	CONSTRAINT pk_manifest_configs PRIMARY KEY (id),
	CONSTRAINT uq_manifest_configurations_digest UNIQUE (digest)

);
-- ddl-end --
-- ALTER TABLE public.manifest_configurations OWNER TO postgres;
-- ddl-end --

-- object: public.manifests_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.manifests_id_seq CASCADE;
CREATE SEQUENCE public.manifests_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.manifests_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.manifests | type: TABLE --
-- DROP TABLE IF EXISTS public.manifests CASCADE;
CREATE TABLE public.manifests (
	id integer NOT NULL DEFAULT nextval('public.manifests_id_seq'::regclass),
	schema_version integer NOT NULL,
	media_type text NOT NULL,
	digest text NOT NULL,
	configuration_id integer,
	payload json NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	marked_at timestamp,
	deleted_at timestamp,
	CONSTRAINT pk_manifests PRIMARY KEY (id),
	CONSTRAINT uq_manifests_digest UNIQUE (digest)

);
-- ddl-end --
-- ALTER TABLE public.manifests OWNER TO postgres;
-- ddl-end --

-- object: public.layers_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.layers_id_seq CASCADE;
CREATE SEQUENCE public.layers_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.layers_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.layers | type: TABLE --
-- DROP TABLE IF EXISTS public.layers CASCADE;
CREATE TABLE public.layers (
	id integer NOT NULL DEFAULT nextval('public.layers_id_seq'::regclass),
	media_type text NOT NULL,
	digest text NOT NULL,
	size bigint NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	marked_at timestamp,
	deleted_at timestamp,
	CONSTRAINT pk_layers PRIMARY KEY (id),
	CONSTRAINT uq_layers_digest UNIQUE (digest)

);
-- ddl-end --
-- ALTER TABLE public.layers OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_layers_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.manifest_layers_id_seq CASCADE;
CREATE SEQUENCE public.manifest_layers_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.manifest_layers_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_layers | type: TABLE --
-- DROP TABLE IF EXISTS public.manifest_layers CASCADE;
CREATE TABLE public.manifest_layers (
	id integer NOT NULL DEFAULT nextval('public.manifest_layers_id_seq'::regclass),
	manifest_id integer NOT NULL,
	layer_id integer NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	deleted_at timestamp,
	CONSTRAINT pk_manifest_layers PRIMARY KEY (id),
	CONSTRAINT uq_manifest_layers_manifest_id_layer_id UNIQUE (manifest_id,layer_id)

);
-- ddl-end --
-- ALTER TABLE public.manifest_layers OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_lists_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.manifest_lists_id_seq CASCADE;
CREATE SEQUENCE public.manifest_lists_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.manifest_lists_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_lists | type: TABLE --
-- DROP TABLE IF EXISTS public.manifest_lists CASCADE;
CREATE TABLE public.manifest_lists (
	id integer NOT NULL DEFAULT nextval('public.manifest_lists_id_seq'::regclass),
	schema_version integer NOT NULL,
	media_type text,
	digest text NOT NULL,
	payload json NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	marked_at timestamp,
	deleted_at timestamp,
	CONSTRAINT pk_manifest_lists PRIMARY KEY (id),
	CONSTRAINT uq_manifest_lists_digest UNIQUE (digest)

);
-- ddl-end --
-- ALTER TABLE public.manifest_lists OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_list_items_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.manifest_list_items_id_seq CASCADE;
CREATE SEQUENCE public.manifest_list_items_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.manifest_list_items_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.manifest_list_items | type: TABLE --
-- DROP TABLE IF EXISTS public.manifest_list_items CASCADE;
CREATE TABLE public.manifest_list_items (
	id integer NOT NULL DEFAULT nextval('public.manifest_list_items_id_seq'::regclass),
	manifest_list_id integer NOT NULL,
	manifest_id integer NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	deleted_at timestamp,
	CONSTRAINT pk_manifest_list_items PRIMARY KEY (id),
	CONSTRAINT uq_manifest_list_items_manifest_list_id_manifest_id UNIQUE (manifest_list_id,manifest_id)

);
-- ddl-end --
-- ALTER TABLE public.manifest_list_items OWNER TO postgres;
-- ddl-end --

-- object: public.tags_id_seq | type: SEQUENCE --
-- DROP SEQUENCE IF EXISTS public.tags_id_seq CASCADE;
CREATE SEQUENCE public.tags_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START WITH 1
	CACHE 1
	NO CYCLE
	OWNED BY NONE;
-- ddl-end --
-- ALTER SEQUENCE public.tags_id_seq OWNER TO postgres;
-- ddl-end --

-- object: public.tags | type: TABLE --
-- DROP TABLE IF EXISTS public.tags CASCADE;
CREATE TABLE public.tags (
	id integer NOT NULL DEFAULT nextval('public.tags_id_seq'::regclass),
	name text NOT NULL,
	repository_id integer NOT NULL,
	manifest_id integer NOT NULL,
	created_at timestamp NOT NULL DEFAULT now(),
	updated_at timestamp,
	deleted_at timestamp,
	CONSTRAINT pk_tags PRIMARY KEY (id),
	CONSTRAINT uq_tags_name_repository_id UNIQUE (name,repository_id)

);
-- ddl-end --
-- ALTER TABLE public.tags OWNER TO postgres;
-- ddl-end --

-- object: public.repository_manifests | type: TABLE --
-- DROP TABLE IF EXISTS public.repository_manifests CASCADE;
CREATE TABLE public.repository_manifests (
	id serial NOT NULL,
	repository_id integer NOT NULL,
	manifest_id integer NOT NULL,
	created_at timestamp NOT NULL DEFAULT NOW(),
	deleted_at timestamp,
	CONSTRAINT pk_repository_manifests PRIMARY KEY (id),
	CONSTRAINT uq_repository_manifests_repository_id_manifest_id UNIQUE (repository_id,manifest_id)

);
-- ddl-end --
-- ALTER TABLE public.repository_manifests OWNER TO postgres;
-- ddl-end --

-- object: public.repository_manifest_lists | type: TABLE --
-- DROP TABLE IF EXISTS public.repository_manifest_lists CASCADE;
CREATE TABLE public.repository_manifest_lists (
	id serial NOT NULL,
	repository_id integer NOT NULL,
	manifest_list_id integer NOT NULL,
	created_at timestamp NOT NULL DEFAULT NOW(),
	deleted_at timestamp,
	CONSTRAINT pk_repository_manifest_lists PRIMARY KEY (id),
	CONSTRAINT uq_repository_manifests_repository_id_manifest_list_id UNIQUE (repository_id,manifest_list_id)

);
-- ddl-end --
-- ALTER TABLE public.repository_manifest_lists OWNER TO postgres;
-- ddl-end --

-- object: fk_repositories_parent_id | type: CONSTRAINT --
-- ALTER TABLE public.repositories DROP CONSTRAINT IF EXISTS fk_repositories_parent_id CASCADE;
ALTER TABLE public.repositories ADD CONSTRAINT fk_repositories_parent_id FOREIGN KEY (parent_id)
REFERENCES public.repositories (id) MATCH SIMPLE
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_manifests_configuration_id | type: CONSTRAINT --
-- ALTER TABLE public.manifests DROP CONSTRAINT IF EXISTS fk_manifests_configuration_id CASCADE;
ALTER TABLE public.manifests ADD CONSTRAINT fk_manifests_configuration_id FOREIGN KEY (configuration_id)
REFERENCES public.manifest_configurations (id) MATCH SIMPLE
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_manifest_layers_manifest_id | type: CONSTRAINT --
-- ALTER TABLE public.manifest_layers DROP CONSTRAINT IF EXISTS fk_manifest_layers_manifest_id CASCADE;
ALTER TABLE public.manifest_layers ADD CONSTRAINT fk_manifest_layers_manifest_id FOREIGN KEY (manifest_id)
REFERENCES public.manifests (id) MATCH SIMPLE
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_manifest_layers_layer_id | type: CONSTRAINT --
-- ALTER TABLE public.manifest_layers DROP CONSTRAINT IF EXISTS fk_manifest_layers_layer_id CASCADE;
ALTER TABLE public.manifest_layers ADD CONSTRAINT fk_manifest_layers_layer_id FOREIGN KEY (layer_id)
REFERENCES public.layers (id) MATCH SIMPLE
ON DELETE NO ACTION ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_manifest_list_items_manifest_list_id | type: CONSTRAINT --
-- ALTER TABLE public.manifest_list_items DROP CONSTRAINT IF EXISTS fk_manifest_list_items_manifest_list_id CASCADE;
ALTER TABLE public.manifest_list_items ADD CONSTRAINT fk_manifest_list_items_manifest_list_id FOREIGN KEY (manifest_list_id)
REFERENCES public.manifest_lists (id) MATCH SIMPLE
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_manifest_list_items_manifest_id | type: CONSTRAINT --
-- ALTER TABLE public.manifest_list_items DROP CONSTRAINT IF EXISTS fk_manifest_list_items_manifest_id CASCADE;
ALTER TABLE public.manifest_list_items ADD CONSTRAINT fk_manifest_list_items_manifest_id FOREIGN KEY (manifest_id)
REFERENCES public.manifests (id) MATCH SIMPLE
ON DELETE NO ACTION ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_tags_repository_id | type: CONSTRAINT --
-- ALTER TABLE public.tags DROP CONSTRAINT IF EXISTS fk_tags_repository_id CASCADE;
ALTER TABLE public.tags ADD CONSTRAINT fk_tags_repository_id FOREIGN KEY (repository_id)
REFERENCES public.repositories (id) MATCH SIMPLE
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_tags_manifest_id | type: CONSTRAINT --
-- ALTER TABLE public.tags DROP CONSTRAINT IF EXISTS fk_tags_manifest_id CASCADE;
ALTER TABLE public.tags ADD CONSTRAINT fk_tags_manifest_id FOREIGN KEY (manifest_id)
REFERENCES public.manifests (id) MATCH FULL
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_repository_manifests_repository_id | type: CONSTRAINT --
-- ALTER TABLE public.repository_manifests DROP CONSTRAINT IF EXISTS fk_repository_manifests_repository_id CASCADE;
ALTER TABLE public.repository_manifests ADD CONSTRAINT fk_repository_manifests_repository_id FOREIGN KEY (repository_id)
REFERENCES public.repositories (id) MATCH FULL
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_repository_manifests_manifest_id | type: CONSTRAINT --
-- ALTER TABLE public.repository_manifests DROP CONSTRAINT IF EXISTS fk_repository_manifests_manifest_id CASCADE;
ALTER TABLE public.repository_manifests ADD CONSTRAINT fk_repository_manifests_manifest_id FOREIGN KEY (manifest_id)
REFERENCES public.manifests (id) MATCH FULL
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_repository_manifest_lists_repository_id | type: CONSTRAINT --
-- ALTER TABLE public.repository_manifest_lists DROP CONSTRAINT IF EXISTS fk_repository_manifest_lists_repository_id CASCADE;
ALTER TABLE public.repository_manifest_lists ADD CONSTRAINT fk_repository_manifest_lists_repository_id FOREIGN KEY (repository_id)
REFERENCES public.repositories (id) MATCH FULL
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --

-- object: fk_repository_manifest_lists_manifest_list_id | type: CONSTRAINT --
-- ALTER TABLE public.repository_manifest_lists DROP CONSTRAINT IF EXISTS fk_repository_manifest_lists_manifest_list_id CASCADE;
ALTER TABLE public.repository_manifest_lists ADD CONSTRAINT fk_repository_manifest_lists_manifest_list_id FOREIGN KEY (manifest_list_id)
REFERENCES public.manifest_lists (id) MATCH FULL
ON DELETE CASCADE ON UPDATE NO ACTION;
-- ddl-end --
