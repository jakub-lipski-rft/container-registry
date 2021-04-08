package metrics

import (
	"time"

	"github.com/docker/distribution/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	queryDurationHist *prometheus.HistogramVec
	queryTotal        *prometheus.CounterVec
	timeSince         = time.Since // for test purposes only
)

const (
	subsystem      = "database"
	queryNameLabel = "name"

	queryDurationName = "query_duration_seconds"
	queryDurationDesc = "A histogram of latencies for database queries."

	queryTotalName = "queries_total"
	queryTotalDesc = "A counter for database queries."
)

func init() {
	queryDurationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      queryDurationName,
			Help:      queryDurationDesc,
			Buckets:   prometheus.DefBuckets,
		},
		[]string{queryNameLabel},
	)

	queryTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      queryTotalName,
			Help:      queryTotalDesc,
		},
		[]string{queryNameLabel},
	)

	prometheus.MustRegister(queryDurationHist)
	prometheus.MustRegister(queryTotal)
}

func InstrumentQuery(name string) func() {
	start := time.Now()
	return func() {
		queryTotal.WithLabelValues(name).Inc()
		queryDurationHist.WithLabelValues(name).Observe(timeSince(start).Seconds())
	}
}
