# Setting up a Standalone Development Registry

This guide describes a basic process for setting up a registry with the
changes that are local to the current git branch. This is a good way to do
simple ad-hoc testing for changes that you have made to a development branch.

These instructions are not intended to represent best practices or to produce
a production environment, but rather to provide the quickest possible path to
a running repository.

## Requirements

This guide will use docker to simplify the deployment process. Please ensure
that you are able to run the `docker` command in your environment.

## Building with Docker

This command will build the registry from the code in the git repository,
whether the changes are committed or not. After this command completes, we
will have local access to a docker image called `registry:dev`. You may choose
to use any name:tag combination and you may also build multiple images with
different versions of the registry for easier comparison of changes.

This command will use the `Dockerfile` in the root of this repository.

```bash
docker build -t registry:dev .
```

## Running with Docker

This command will start the Registry in a docker container.

The configuration file in `cmd/registry/config-dev.yml`  can be substituted with
the full path of any valid configuration file. With this configuration file,
the registry API can be accessed from `localhost:5000`.

The registry name, `dev-registry`, can be used to easily reference the container
in docker commands and is arbitrary.

This container is ran with host networking, this option facilitates an easier
and more general configuration, especially when using external services, such as
GCS or S3, but also removes the network isolation that a container typically
provides.

```bash
docker run -d --restart=always --network=host --name dev-registry -v `pwd`/cmd/registry/config-dev.yml:/etc/docker/registry/config.yml registry:dev
```

## Verification

If everything is running correctly the following command should produce this
output:

```bash
localhost:5000/v2/_catalog
{"repositories":[]}
```

## Logs

Registry logs can be accessed with the following command:

```bash
docker logs -f dev-registry
```
