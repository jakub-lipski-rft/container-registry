module github.com/docker/distribution

go 1.14

require (
	cloud.google.com/go/storage v1.12.0
	github.com/Azure/azure-sdk-for-go v54.1.0+incompatible
	github.com/Azure/go-autorest v10.8.1+incompatible // indirect
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/aws/aws-sdk-go v1.38.39
	github.com/benbjohnson/clock v1.0.3
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/denverdino/aliyungo v0.0.0-20181224103910-6df11717a253
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916
	github.com/docker/libtrust v0.0.0-20150114040149-fa567046d9b1
	github.com/getsentry/sentry-go v0.7.0
	github.com/go-redis/redis/v8 v8.4.8
	github.com/golang/mock v1.5.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jackc/pgconn v1.8.1
	github.com/jackc/pgerrcode v0.0.0-20201024163028-a0d42d470451
	github.com/jackc/pgx/v4 v4.11.0
	github.com/jszwec/csvutil v1.5.0
	github.com/lib/pq v1.9.0 // indirect
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/ncw/swift v1.0.52
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.0
	github.com/prometheus/client_golang v1.3.0
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	gitlab.com/gitlab-org/labkit v1.3.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/api v0.32.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
