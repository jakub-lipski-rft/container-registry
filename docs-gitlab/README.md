# Container Registry

## Guides

[Local Integration Testing](storage-driver-integration-testing-guide.md)

## Differences From Upstream

### Configuration

The S3 storage driver takes an additional parameter, `pathstyle`.
When set to `true`, the driver will use path style routes.
When not set, the driver will default to virtual path style routes, unless
`regionendpoint` is set. In which case, the driver will use path style routes.
When explicitly set to `false`, the driver will continue to default to virtual
host style routes, even when the `regionendpoint` parameter is set.
