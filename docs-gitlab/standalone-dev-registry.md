# Setting up a Standalone Development Registry

This guide describes a basic process for setting up a registry for development
or testing purposes.

These instructions are not intended to represent best practices or to produce
a production environment, but rather to provide the quickest possible path to
a running repository.

## From Source

To install the container registry from source, we need to fetch the source code,
make binaries and execute.

### Requirements

You will need to have Go installed on your machine. Please refer to the official
documentation for guidance: https://golang.org/doc/install.

### Building

- Make source path. The GitLab container registry is a fork from the upstream
Docker Distribution repository, so we need to maintain the same import path 
(`github.com/docker/distribution`) for the source code to compile:
    ```
    mkdir -p $GOPATH/src/github.com/docker
    ```

- Fetch source code into `distribution`:
    ```
    cd $GOPATH/src/github.com/docker
    git clone git@gitlab.com:gitlab-org/container-registry.git distribution
    cd distribution
    ```

- All source code dependencies are in `vendor/`, so we don't need to download
them. However, if you want to update or install any other dependency, you'll 
need the [`vndr`](https://github.com/LK4D4/vndr) tool. Install it with the 
following command (please refer to the tool documentation for usage help):
    ```
    ./script/setup/install-dev-tools
    ```
    
- Make binaries:
    ```
    make binaries
    ```
    This step should complete without any errors. If not, please double check
    the official Go install documentation and make sure you have all build 
    dependencies installed on your machine.
    
### Running

This command will start the Registry in the foreground.

```
./bin/registry serve cmd/registry/config-dev.yml
```

The configuration file in `cmd/registry/config-dev.yml` can be substituted with
the full path of any valid configuration file. With this configuration file,
the registry API can be accessed from `localhost:5000`.

## Docker

### Requirements

This guide will use docker to simplify the deployment process. Please ensure
that you are able to run the `docker` command in your environment.

### Building

This command will build the registry from the code in the git repository,
whether the changes are committed or not. After this command completes, we
will have local access to a docker image called `registry:dev`. You may choose
to use any name:tag combination and you may also build multiple images with
different versions of the registry for easier comparison of changes.

This command will use the `Dockerfile` in the root of this repository.

```bash
docker build -t registry:dev .
```

### Running

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

### Logs

Registry logs can be accessed with the following command:

```bash
docker logs -f dev-registry
```

## Verification

If everything is running correctly the following command should produce this
output:

```bash
curl localhost:5000/v2/_catalog
{"repositories":[]}
```