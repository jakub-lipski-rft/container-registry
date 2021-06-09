# GitLab Container Registry

## Guides

### Development

- [Development Environment Setup](development-environment-setup.md)
- [Local Integration Testing](storage-driver-integration-testing-guide.md)
- [Offline Garbage Collection Testing](garbage-collection-testing-guide.md)
- [Database Development Guidelines](database-dev-guidelines.md)
- [Database Migrations](database-migrations.md)

### Technical Documentation

- [Metadata Import](database-import-tool.md)
- [Push/pull Request Flow](push-pull-request-flow.md)
- [Authentication Request Flow](auth-request-flow.md)
- [Online Garbage Collection](db/online-garbage-collection.md)
- [HTTP API Queries](db/http-api-queries.md)
- [Migration Proxy Mode](migration-proxy.md)

### Troubleshooting

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

`maxretries`

The maximum number of times the driver will attempt to retry failed requests.
Set to `0` to disable retries entirely.

#### Azure Storage Driver

##### Additional parameters

`rootdirectory`

This parameter specifies the root directory in which all registry files are
stored. Defaults to the empty string (bucket root).

`trimlegacyrootprefix`

Orginally, the Azure driver would write to `//` as the root directory, also
appearing in some places as `/<no-name>/` within the Azure UI. This legacy
behavior must be preserved to support older deployments using this driver.
Set to `true` to build root paths without an extra leading slash.

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

#### Estimating Freed Storage

Garbage collection now estimates the amount of storage that will be freed.
It's possible to estimate freed storage without setting the registry to
read-only mode by doing a garbage collection dry run; however, this will affect
the accuracy of the estimate. Without the registry being read-only, blobs may be
re-referenced, which would lead to an overestimate. Blobs might be
dereferenced, leading to an underestimate.

#### Debug Server

A pprof debug server can be used to collect profiling information on a
garbage collection run by providing an `address:port` to the command via
the `--debug-server` (`--s`) flag. Usage information for this server can be
found in the documentation for pprof: https://golang.org/pkg/net/http/pprof/

### API

#### Tag Delete

A new route, `DELETE /v2/<name>/tags/reference/<reference>`, was added to the
API, enabling the deletion of tags by name.

#### Broken link files when fetching a manifest by tag

When fetching a manifest by tag, through `GET /v2/<name>/manifests/<tag>`, if
the manifest link file in 
`/docker/registry/v2/repositories/<name>/_manifests/tags/<tag>/current/link` is
empty or corrupted, instead of returning a `500 Internal Server Error` response
like the upstream implementation, it returns a `404 Not Found` response with a 
`MANIFEST_UNKNOWN` error in the body.

If for some reason a tag manifest link file is broken, in practice, it's as if
it didn't exist at all, thus the `404 Not Found`. Re-pushing the tag will fix
the broken link file.

#### Custom Headers on `GET /v2/`

Two new headers were added to the response of `GET /v2/` requests:

* `Gitlab-Container-Registry-Version`: The semantic version of the GitLab
Container Registry (e.g. `2.9.0-gitlab`). This is set during build time (in
`version.Version`).
* `Gitlab-Container-Registry-Features`: A comma separated list of supported
features/extensions that are not part of the Docker Distribution spec (e.g.
`tag_delete,...`). Its value (hardcoded in `version.ExtFeatures`) should be
updated whenever a custom feature is added/deprecated.

This is necessary to detect whether a registry is the GitLab Container Registry
and which extra features it supports.

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

## Contributing

Merge requests which make change that will impact users of this project should
have an accompanying [changelog](../CHANGELOG.md) entry in the same merge
request. These entries should be added under the `[Unreleased]` header. The
changelog follows the [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
specification.

### Golang Version Support

Starting from version 1.13, this project will support being built with the
latest three major [releases](https://golang.org/doc/devel/release.html) of the
Go Programming Language.

This support is ensured via the `.gitlab-ci.yml` file in the root of this
repository, if you modify this file to add additional jobs, please ensure that
those jobs are ran with all three versions.
