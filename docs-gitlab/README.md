# Container Registry

## Guides

- [Local Integration Testing](storage-driver-integration-testing-guide.md)
- [Cleanup Invalid Link Files](cleanup-invalid-link-files.md)

## Differences From Upstream

### Configuration

The S3 storage driver takes an additional parameter, `pathstyle`.
When set to `true`, the driver will use path style routes.
When not set, the driver will default to virtual path style routes, unless
`regionendpoint` is set. In which case, the driver will use path style routes.
When explicitly set to `false`, the driver will continue to default to virtual
host style routes, even when the `regionendpoint` parameter is set.

### Garbage Collection

#### Invalid Link Files

If a bad link file (e.g. 0B in size or invalid checksum) is found during the
*mark* stage, instead of stopping the garbage collector (standard behaviour)
it will log a warning message and ignore it, letting the process continue.
Blobs related with invalid link files will be automatically swept away in the
*sweep* stage if those blobs are not associated with another valid link file.

See [Cleanup Invalid Link Files](cleanup-invalid-link-files.md) for a guide on
how to detect and clean these files based on the garbage collector output log.
