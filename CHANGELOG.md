## [Unreleased]
### Removed
- configuration: Remove proxy configuration migration section
- registry: Remove ability to migrate to remote registry

## [v3.5.0-gitlab] - 2021-06-10
### Changed
- registry/datastore: Partitioning by top-level namespace

### Fixed
- registry/storage: Offline garbage collection no longer inappropriately removes untagged manifests referenced by a manifest list

### Added
- registry/storage: S3 Driver will now use Exponential backoff to retry failed requests

## [v3.4.1-gitlab] - 2021-05-11
### Fixed
- registry/storage: S3 driver now respects rate limits in all cases

### Changed
- registry/storage: Upgrade Amazon S3 SDK to v1.38.26
- registry/storage: Upgrade golang.org/x/time to v0.0.0-20210220033141-f8bda1e9f3ba
- registry: Upgrade github.com/opencontainers/go-digest to v1.0.0
- registry/storage: Upgrade Azure SDK to v54.1.0

## [v3.4.0-gitlab] - 2021-04-26
### Changed
- registry/datastore: Switch from 1 to 64 partitions per table

### Fixed
- registry: Log operating system quit signal as string

### Added
- registry/gc: Add Prometheus counter and histogram for online GC runs
- registry/gc: Add Prometheus counter and histogram for online GC deletions
- registry/gc: Add Prometheus counter for online GC deleted bytes
- registry/gc: Add Prometheus counter for online GC review postpones
- registry/gc: Add Prometheus histogram for sleep durations between online GC runs
- registry/gc: Add Prometheus gauge for the online GC review queues size

## [v3.3.0-gitlab] - 2021-04-09
### Added
- registry: Add Prometheus counter for database queries

### Changed
- registry/storage: Upgrade Azure SDK to v52.5.0

## [v3.2.1-gitlab] - 2021-03-17
### Fixed
- configuration: Don't require storage section for the database migrate CLI

## [v3.2.0-gitlab] - 2021-03-15
### Added
- configuration: Add `rootdirectory` option to the azure storage driver
- configuration: Add `trimlegacyrootprefix` option to the azure storage driver

## [v3.1.0-gitlab] - 2021-02-25
### Added
- configuration: Add `preparedstatements` option to toggle prepared statements for the metadata database
- configuration: Add `draintimeout` to database stanza to set optional connection close timeout on shutdown
- registry/api/v2: Disallow manifest delete if referenced by manifest lists (metadata database only).
- registry: Add CLI flag to facilitate programmatic state checks for database migrations
- registry: Add continuous online garbage collection

### Changed
- registry/datastore: Metadata database does not use prepared statements by default

## [v3.0.0-gitlab] - 2021-01-20
### Added
- registry: Experimental PostgreSQL metadata database (disabled by default)
- registry/storage/cache/redis: Add size and maxlifetime pool settings

### Changed
- registry/storage: Upgrade Swift client to v1.0.52

### Fixed
- registry/api: Fix tag delete response body

### Removed
- configuration: Drop support for TLS 1.0 and 1.1 and default to 1.2
- registry/storage/cache/redis: Remove maxidle and maxactive pool settings
- configuration: Drop support for logstash and combined log formats and default to json
- configuration: Drop support for log hooks
- configuration: Drop NewRelic reporting support
- configuration: Drop Bugsnag reporting support
- registry/api/v2: Drop support for schema 1 manifests and default to schema 2

## [v2.13.1-gitlab] - 2021-01-13
### Fixed
- registry: Fix HTTP request duration and byte size Prometheus metrics buckets

## [v2.13.0-gitlab] - 2020-12-15
### Added
- registry: Add support for a pprof monitoring server
- registry: Use GitLab LabKit for HTTP metrics collection
- registry: Expose build info through the Prometheus metrics

### Changed
- configuration: Improve error reporting when `storage.redirect` section is misconfigured
- registry/storage: Upgrade the GCS SDK to v1.12.0

### Fixed
- registry: Fix support for error reporting with Sentry

## [v2.12.0-gitlab] - 2020-11-23
### Deprecated
- configuration: Deprecate log hooks, to be removed by January 22nd, 2021
- configuration: Deprecate Bugsnag support, to be removed by January 22nd, 2021
- configuration: Deprecate NewRelic support, to be removed by January 22nd, 2021
- configuration: Deprecate logstash and combined log formats, to be removed by January 22nd, 2021
- registry/api: Deprecate Docker Schema v1 compatibility, to be removed by January 22nd, 2021
- configuration: Deprecate TLS 1.0 and TLS 1.1 support, to be removed by January 22nd, 2021

### Added
- registry: Add support for error reporting with Sentry
- registry/storage/cache/redis: Add Prometheus metrics for Redis cache store
- registry: Add TLS support for Redis
- registry: Add support for Redis Sentinel
- registry: Enable toggling redirects to storage backends on a per-repository basis

### Changed
- configuration: Cloudfront middleware `ipfilteredby` setting is now optional

### Fixed
- registry/storage: Swift path generation now generates multiple directories as intended
- registry/client/auth: OAuth token authentication now returns a `ErrNoToken` if a token is not found in the response
- registry/storage: Fix custom User-Agent header on S3 requests
- registry/api/v2: Text-charset selector removed from `application/json` content-type

## [v2.11.0-gitlab] - 2020-09-08
## Added
- registry: Add new configuration for changing the output for logs and the access logs format

## Changed
- registry: Use GitLab LabKit for correlation and logging
- registry: Normalize log messages

## [v2.10.0-gitlab] - 2020-08-05
## Added
- registry: Add support for continuous profiling with Google Stackdriver

## [v2.9.1-gitlab] - 2020-05-05
## Added
- registry/api/v2: Show version and supported extra features in custom headers

## Changed
- registry/handlers: Encapsulate the value of err.detail in logs in a JSON object

### Fixed
- registry/storage: Fix panic during uploads purge

## [v2.9.0-gitlab] - 2020-04-07
### Added
- notifications: Notification related Prometheus metrics
- registry: Make minimum TLS version user configurable
- registry/storage: Support BYOK for OSS storage driver

### Changed
- Upgrade to Go 1.13
- Switch to Go Modules for dependency management
- registry/handlers: Log authorized username in push/pull requests

### Fixed
- configuration: Fix pointer initialization in configuration parser
- registry/handlers: Process Accept header MIME types in case-insensitive way

## [v2.8.2-gitlab] - 2020-03-13
### Changed
- registry/storage: Improve performance of the garbage collector for GCS
- registry/storage: Gracefully handle missing tags folder during garbage collection
- registry/storage: Cache repository tags during the garbage collection mark phase
- registry/storage: Upgrade the GCS SDK to v1.2.1
- registry/storage: Provide an estimate of how much storage will be removed on garbage collection
- registry/storage: Make the S3 driver log level configurable
- registry/api/v2: Return not found error when getting a manifest by tag with a broken link

### Fixed
- registry/storage: Fix PathNotFoundError not being ignored in repository enumeration during garbage collection when WalkParallel is enabled

## v2.8.1-gitlab

- registry/storage: Improve consistency of garbage collection logs

## v2.8.0-gitlab

- registry/api/v2: Add tag delete route

## v2.7.8-gitlab

- registry/storage: Improve performance of the garbage collection algorithm for S3

## v2.7.7-gitlab

- registry/storage: Handle bad link files gracefully during garbage collection
- registry/storage: AWS SDK v1.26.3 update
- registry: Include build info on Prometheus metrics

## v2.7.6-gitlab

- CI: Add integration tests for the S3 driver
- registry/storage: Add compatibilty for S3v1 ListObjects key counts

## v2.7.5-gitlab

- registry/storage: Log a message if PutContent is called with 0 bytes

## v2.7.4-gitlab

- registry/storage: Fix Google Cloud Storage client authorization with non-default credentials
- registry/storage: Fix error handling of GCS Delete() call when object does not exist

## v2.7.3-gitlab

- registry/storage: Update to Google SDK v0.47.0 and latest storage driver (v1.1.1)

## v2.7.2-gitlab

- registry/storage: Use MD5 checksums in the registry's Google storage driver
