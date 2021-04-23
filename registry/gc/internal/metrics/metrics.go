package metrics

import (
	"strconv"
	"time"

	"github.com/docker/distribution/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	runDurationHist           *prometheus.HistogramVec
	runCounter                *prometheus.CounterVec
	deleteDurationHist        *prometheus.HistogramVec
	deleteCounter             *prometheus.CounterVec
	storageDeleteBytesCounter *prometheus.CounterVec
	postponeCounter           *prometheus.CounterVec

	timeSince = time.Since // for test purposes only
)

const (
	subsystem = "gc"

	workerLabel    = "worker"
	errorLabel     = "error"
	noopLabel      = "noop"
	artifactLabel  = "artifact"
	backendLabel   = "backend"
	mediaTypeLabel = "media_type"

	blobArtifact     = "blob"
	manifestArtifact = "manifest"
	storageBackend   = "storage"
	databaseBackend  = "database"

	runDurationName = "run_duration_seconds"
	runDurationDesc = "A histogram of latencies for online GC worker runs."
	runTotalName    = "runs_total"
	runTotalDesc    = "A counter for online GC worker runs."

	deleteTotalName    = "deletes_total"
	deleteTotalDesc    = "A counter of artifacts deleted during online GC."
	deleteDurationName = "delete_duration_seconds"
	deleteDurationDesc = "A histogram of latencies for artifact deletions during online GC."

	storageDeleteBytesTotalName = "storage_deleted_bytes_total"
	storageDeleteBytesTotalDesc = "A counter for bytes deleted from storage during online GC."

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

	deleteDurationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      deleteDurationName,
			Help:      deleteDurationDesc,
			Buckets:   prometheus.DefBuckets,
		},
		[]string{backendLabel, artifactLabel, errorLabel},
	)

	deleteCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      deleteTotalName,
			Help:      deleteTotalDesc,
		},
		[]string{backendLabel, artifactLabel},
	)

	storageDeleteBytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.NamespacePrefix,
			Subsystem: subsystem,
			Name:      storageDeleteBytesTotalName,
			Help:      storageDeleteBytesTotalDesc,
		},
		[]string{mediaTypeLabel},
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
	prometheus.MustRegister(deleteDurationHist)
	prometheus.MustRegister(deleteCounter)
	prometheus.MustRegister(postponeCounter)
	prometheus.MustRegister(storageDeleteBytesCounter)
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

func workerDelete(backend, artifact string) func(err error) {
	start := time.Now()
	return func(err error) {
		if err == nil {
			deleteCounter.WithLabelValues(backend, artifact).Inc()
		}
		failed := strconv.FormatBool(err != nil)
		deleteDurationHist.WithLabelValues(backend, artifact, failed).Observe(timeSince(start).Seconds())
	}
}

func blobDelete(backend string) func(err error) {
	return workerDelete(backend, blobArtifact)
}

func BlobDatabaseDelete() func(err error) {
	return blobDelete(databaseBackend)
}

func BlobStorageDelete() func(err error) {
	return blobDelete(storageBackend)
}

func ManifestDelete() func(err error) {
	return workerDelete(databaseBackend, manifestArtifact)
}

func StorageDeleteBytes(bytes int64, mediaType string) {
	storageDeleteBytesCounter.WithLabelValues(mediaType).Add(float64(bytes))
}

func ReviewPostpone(workerName string) {
	postponeCounter.WithLabelValues(workerName).Inc()
}
