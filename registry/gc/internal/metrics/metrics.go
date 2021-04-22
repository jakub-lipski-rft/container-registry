package metrics

import (
	"strconv"
	"time"

	"github.com/docker/distribution/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	runDurationHist *prometheus.HistogramVec
	runCounter      *prometheus.CounterVec
	postponeCounter *prometheus.CounterVec
	timeSince       = time.Since // for test purposes only
)

const (
	subsystem   = "gc"
	workerLabel = "worker"
	errorLabel  = "error"
	noopLabel   = "noop"

	runDurationName = "run_duration_seconds"
	runDurationDesc = "A histogram of latencies for online GC worker runs."

	runTotalName = "runs_total"
	runTotalDesc = "A counter for online GC worker runs."

	postponeTotalName = "postpones_total"
	postponeTotalDesc = "A counter for online GC review postpones."
)

func init() {
	runDurationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      runDurationName,
			Help:      runDurationDesc,
			Buckets:   prometheus.DefBuckets,
		},
		[]string{workerLabel, noopLabel, errorLabel},
	)

	runCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      runTotalName,
			Help:      runTotalDesc,
		},
		[]string{workerLabel, noopLabel, errorLabel},
	)

	postponeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      postponeTotalName,
			Help:      postponeTotalDesc,
		},
		[]string{workerLabel},
	)

	prometheus.MustRegister(runDurationHist)
	prometheus.MustRegister(runCounter)
	prometheus.MustRegister(postponeCounter)
}

func WorkerRun(name string) func(noop bool, err error) {
	start := time.Now()
	return func(noop bool, err error) {
		failed := strconv.FormatBool(err != nil)
		np := strconv.FormatBool(noop)

		runCounter.WithLabelValues(name, np, failed).Inc()
		runDurationHist.WithLabelValues(name, np, failed).Observe(timeSince(start).Seconds())
	}
}

func ReviewPostpone(workerName string) {
	postponeCounter.WithLabelValues(workerName).Inc()
}
