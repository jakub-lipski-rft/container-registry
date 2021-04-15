# Iventoring Repositories in Object Storage

The inventory utility allows users to obtain a list of all repositories and a
total of their tags from object storage. This list will be written to `stdout`
in either a human-readable table or various machine-readable formats.

## The Inventory Command

This command can be accessed via the registry binary and takes the following form.

```bash
./registry inventory [flags] path/to/config.yml
```

### Options

#### Format

The `--format` flag determines how the inventory will be displayed. Options are
as follows:

##### Text (default)

`--format text` prints the inventory in a table. Additionally, this
output will include extra summary information for convenience.

##### Json

`--format json` prints the inventory in JSON.

##### CSV

`--format csv` prints the inventory CSV format with headers.

#### Tag Count

The `--tag-count` option will also sum the number of tags present in each
repository. This is set to `true` by default. Disabling this will increase the
speed and reduce the number of API requests to the storage backend. This option
should result in comparatively faster speeds as registry size increases.

## Prerequisites

### Configuration

The configuration passed to the inventory command should be based on the
configuration of the registry that you are inventoring. Particularly important
is that the `storage` section is configured the same way so that the inventory
command has access to the data used by the registry you wish to inventory.

The following sections discuss configuration options that are relevant to the
inventory process, directly or indirectly. These section assumes that you are
starting with a working and appropriate configuration for an existing registry.

#### Log Output

Since registries can grow to contain an indefinite amount of of repositories,
the inventory command logs its progress to provide visibility for users running
this command. If you wish to redirect the result of the inventory, which is
printed to `stdout`, without including the logging, you may wish to set logging
to output to stderr:

```yaml
log:
  output: stderr
```

#### ParallelWalk

For the GCS and S3 storage drivers, it's possible to greatly increase the speed
of the inventory by enabling the `ParallelWalk` feature. This will process
repositories in parallel, so this will increase the resource usage on the
machine run the inventory command and increase the rate at which API requests to
the storage backend are made.

```yaml
storage:
  gcs:
    bucket: bucketname
    keyfile: /path/to/keyfile
    parallelwalk: true
```
```yaml
storage:
  s3:
    accesskey: awsaccesskey
    secretkey: awssecretkey
    region: us-west-1
    regionendpoint: http://myobjects.local
    bucket: bucketname
    parallelwalk: true
```

## Inventory

Once you have have prepared a configuration file as described above, you are
ready to run the inventory command:

Navigate to the environment where your registry binary is located. You will need
to locate the registry binary the run the inventory command. For this example,
we will assume the registry binary is the in current working directory:

```bash
./registry inventory [flags] path/to/config.yml
```

### Running the Inventory on an Active Registry

It's possible to run the inventory command against a registry which is still in
use; however, please note that the inventory will skew a bit from the actual
state of the registry as new repositories and tags are written during the
inventory.
