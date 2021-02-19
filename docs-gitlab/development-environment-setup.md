# Development Environment Setup

This guide describes the process of setting up a registry for development
purposes. These instructions are not intended to represent best practices for
production environments.

## From Source

To install the container registry from source, we need to clone the source code,
make binaries and execute the `registry` binary.

### Requirements

You will need to have Go installed on your machine. You must use one of the
officially supported Go versions. Please refer to the Go [release
policy](https://golang.org/doc/devel/release.html#policy) and [install
documentation](https://golang.org/doc/install) for guidance.

### Building

> These instructions assume you are using a Go version with
[modules](https://golang.org/ref/mod) support enabled.

1. Clone repository:
    ```
    git clone git@gitlab.com:gitlab-org/container-registry.git
    cd container-registry
    ```
1. Make binaries:
    ```
    make binaries
    ```
    This step should complete without any errors. If not, please double check
    the official Go install documentation and make sure you have all build
    dependencies installed on your machine.

### Running

This command will start the registry in the foreground, listening at
`localhost:5000`:

```
./bin/registry serve config/example.yml
```

The configuration file [`config/example.yml`](../config/example.yml) is a sample
configuration file with the minimum required settings, plus some recommended
ones for a better development experience. Please see the [configuration
documentation](../docs/configuration.md) for more details and additional
settings.

## Docker

### Requirements

This guide will use Docker to simplify the deployment process. Please ensure
that you can run the `docker` command in your environment.

### Building

This command will build the registry from the code in the git repository,
whether the changes are committed or not. After this command completes, we
will have local access to a docker image called `registry:dev`. You may choose
to use any name:tag combination, and you may also build multiple images with
different versions of the registry for easier comparison of changes.

This command will use the `Dockerfile` in the root of this repository.

```bash
docker build -t registry:dev .
```

### Running

This command will start the registry in a docker container, with the API
listening at `localhost:5000`.

The configuration file [`config/example.yml`](../config/example.yml) is a sample
configuration file with the minimum required settings, plus some recommended
ones for a better development experience. Please see the [configuration
documentation](../docs/configuration.md) for more details and additional
settings.

The registry name, `dev-registry`, can be used to easily reference the container
in docker commands and is arbitrary.

This container is ran with host networking. This option facilitates an easier
and more general configuration, especially when using external services, such as
GCS or S3, but also removes the network isolation that a container typically
provides.

```bash
docker run -d \
    --restart=always \ 
    --network=host \
    --name dev-registry \
    -v `pwd`/config/example.yml:/etc/docker/registry/config.yml \
    registry:dev
```

### Logs

The registry logs can be accessed with the following command:

```bash
docker logs -f dev-registry
```

## Insecure Registries

For development purposes, you will likely use an unencrypted HTTP connection
(the default when using the provided sample configuration file) or self-signed
certificates.

In this case, you must instruct Docker to treat your registry as insecure.
Otherwise, you will not be able to push/pull images. Please follow the
instructions at https://docs.docker.com/registry/insecure/ to configure your
Docker daemon.

## Verification

If everything is running correctly, the following command should produce this
output:

```bash
curl localhost:5000/v2/_catalog
{"repositories":[]}
```

You can now try to build and push/pull images to/from your development registry,
for example:

```shell docker pull alpine:latest docker tag alpine:latest
localhost:5000/alpine:latest docker push localhost:5000/alpine:latest docker
pull localhost:5000/alpine:latest ```