module github.com/docker/distribution

go 1.13

require (
	cloud.google.com/go/storage v1.2.1
	github.com/Azure/azure-sdk-for-go v16.2.1+incompatible
	github.com/Azure/go-autorest v10.8.1+incompatible // indirect
	github.com/FZambia/sentinel v1.1.0
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d
	github.com/aws/aws-sdk-go v1.27.0
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v0.4.1
	github.com/bugsnag/bugsnag-go v1.0.3-0.20141110184014-b1d153021fcd
	github.com/bugsnag/osext v0.0.0-20130617224835-0dd3f918b21b // indirect
	github.com/bugsnag/panicwrap v0.0.0-20151223152923-e2c28503fcd0 // indirect
	github.com/denverdino/aliyungo v0.0.0-20181224103910-6df11717a253
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916
	github.com/docker/libtrust v0.0.0-20150114040149-fa567046d9b1
	github.com/gomodule/redigo v1.8.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.9.0
	github.com/marstr/guid v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mitchellh/osext v0.0.0-20151018003038-5e2d6d41470f // indirect
	github.com/ncw/swift v1.0.40
	github.com/olekukonko/tablewriter v0.0.4
	github.com/opencontainers/go-digest v0.0.0-20170106003457-a6d0ee40d420
	github.com/opencontainers/image-spec v1.0.0
	github.com/prometheus/client_golang v1.3.0
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.6.1
	github.com/yvasiyarov/go-metrics v0.0.0-20140926110328-57bccd1ccd43 // indirect
	github.com/yvasiyarov/gorelic v0.0.7-0.20141212073537-a9bba5b9ab50
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20140908184405-b21fdbd4370f // indirect
	gitlab.com/gitlab-org/labkit v1.1.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	google.golang.org/api v0.15.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/yaml.v2 v2.2.5
)
