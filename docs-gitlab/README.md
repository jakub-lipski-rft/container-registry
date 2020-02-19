# Container Registry

## Guides

- [Local Integration Testing](storage-driver-integration-testing-guide.md)
- [Cleanup Invalid Link Files](cleanup-invalid-link-files.md)

## Differences From Upstream

### Configuration

#### S3 Storage Driver

##### Additional parameters

`pathstyle`

When set to `true`, the driver will use path style routes.
When not set, the driver will default to virtual path style routes, unless
`regionendpoint` is set. In which case, the driver will use path style routes.
When explicitly set to `false`, the driver will continue to default to virtual
host style routes, even when the `regionendpoint` parameter is set.

`parallelwalk`

When this feature flag is set to `true`, the driver will run certain operations,
most notably garbage collection, using multiple concurrent goroutines. This
feature will improve the performance of garbage collection, but will
increase the memory and CPU usage of this command as compared to the default,
particularly when the `--delete-untagged` (`-m`) option is specified.

`maxrequestspersecond`

This parameter determines the maximum number of requests that
the driver will make to the configured S3 bucket per second. Defaults to `350`
with a maximum value of `3500` which corresponds to the current rate limits of
S3: https://docs.aws.amazon.com/AmazonS3/latest/dev/optimizing-performance.html
`0` is a special value which disables rate limiting. This is not recommended
for use in production environments, as exceeding your request budget will result
in errors from the Amazon S3 service.

#### GCS Storage Driver

##### Additional parameters

`parallelwalk`

When this feature flag is set to `true`, the driver will run certain operations,
most notably garbage collection, using multiple concurrent goroutines. This
feature will improve the performance of garbage collection, but will
increase the memory and CPU usage of this command as compared to the default.

### Garbage Collection

#### Invalid Link Files

If a bad link file (e.g. 0B in size or invalid checksum) is found during the
*mark* stage, instead of stopping the garbage collector (standard behaviour)
it will log a warning message and ignore it, letting the process continue.
Blobs related with invalid link files will be automatically swept away in the
*sweep* stage if those blobs are not associated with another valid link file.

See [Cleanup Invalid Link Files](cleanup-invalid-link-files.md) for a guide on
how to detect and clean these files based on the garbage collector output log.

#### Debug Server

A pprof debug server can be used to collect profiling information on a
garbage collection run by providing an `address:port` to the command via
the `--debug-server` (`--s`) flag. Usage information for this server can be
found in the documentation for pprof: https://golang.org/pkg/net/http/pprof/

### API

#### Tag Delete

A new route, `DELETE /v2/<name>/tags/reference/<reference>`, was added to the
API, enabling the deletion of tags by name.

## Releases

Release planning is done by using the `Release Plan` issue template during the
planning phase of a new milestone.

The template will include a list of container registry issues which
are planned for the milestone that should be merged into and included in
the release.

Since multiple projects need to be updated to ensure a version of the Container
Registry is released, this issue should have a due date set to one week before
the milestone. This should allow enough time for the related merge requests to
go through, especially if feedback is received.

`release.sh`, located at the root of this repository, can be ran to aid in the
tagging a new release. It will automatically suggest the next release version,
and create the appropriate tag, and will prompt the user to make a changelog
entry if the chosen release version is not present in the changelog. Once the
changelog entry is made, the release tag will be pushed upstream.
