package metrics

import (
	"time"

	"github.com/docker/distribution/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	statementDurationHist *prometheus.HistogramVec
	timeSince             = time.Since // for test purposes only
)

const (
	subsystem              = "database"
	statementDurationName  = "statement_duration_seconds"
	statementDurationDesc  = "Histogram of database statement durations in seconds"
	statementDurationLabel = "statement"
)

func init() {
	statementDurationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      statementDurationName,
			Help:      statementDurationDesc,
			Buckets:   prometheus.DefBuckets,
		},
		[]string{statementDurationLabel},
	)

	prometheus.MustRegister(statementDurationHist)
}

func StatementDuration(statementName string) func() {
	start := time.Now()
	return func() {
		statementDurationHist.WithLabelValues(statementName).Observe(timeSince(start).Seconds())
	}
}
