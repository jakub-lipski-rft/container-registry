package metrics

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/docker/distribution/metrics"
	"github.com/prometheus/client_golang/prometheus"
	testutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func mockTimeSince(d time.Duration) func() {
	bkp := timeSince
	timeSince = func(_ time.Time) time.Duration { return d }
	return func() { timeSince = bkp }
}

func TestWorkerRun(t *testing.T) {
	name := "foo"

	restore := mockTimeSince(10 * time.Millisecond)
	defer restore()

	report := WorkerRun(name)
	report(true, errors.New("foo"))
	report(true, errors.New("foo")) // to see the aggregated counter increase to 2
	report(true, nil)

	mockTimeSince(20 * time.Millisecond)
	report = WorkerRun(name)
	report(false, nil)
	report(false, errors.New("foo"))

	var expected bytes.Buffer
	expected.WriteString(`
# HELP registry_gc_run_duration_seconds A histogram of latencies for online GC worker runs.
# TYPE registry_gc_run_duration_seconds histogram
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.005"} 0
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.01"} 0
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.025"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.05"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.1"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.25"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="0.5"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="1"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="2.5"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="5"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="10"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="false",worker="foo",le="+Inf"} 1
registry_gc_run_duration_seconds_sum{error="false",noop="false",worker="foo"} 0.02
registry_gc_run_duration_seconds_count{error="false",noop="false",worker="foo"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.005"} 0
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.01"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.025"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.05"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.1"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.25"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="0.5"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="1"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="2.5"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="5"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="10"} 1
registry_gc_run_duration_seconds_bucket{error="false",noop="true",worker="foo",le="+Inf"} 1
registry_gc_run_duration_seconds_sum{error="false",noop="true",worker="foo"} 0.01
registry_gc_run_duration_seconds_count{error="false",noop="true",worker="foo"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.005"} 0
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.01"} 0
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.025"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.05"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.1"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.25"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="0.5"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="1"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="2.5"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="5"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="10"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="false",worker="foo",le="+Inf"} 1
registry_gc_run_duration_seconds_sum{error="true",noop="false",worker="foo"} 0.02
registry_gc_run_duration_seconds_count{error="true",noop="false",worker="foo"} 1
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.005"} 0
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.01"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.025"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.05"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.1"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.25"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="0.5"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="1"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="2.5"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="5"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="10"} 2
registry_gc_run_duration_seconds_bucket{error="true",noop="true",worker="foo",le="+Inf"} 2
registry_gc_run_duration_seconds_sum{error="true",noop="true",worker="foo"} 0.02
registry_gc_run_duration_seconds_count{error="true",noop="true",worker="foo"} 2
# HELP registry_gc_runs_total A counter for online GC worker runs.
# TYPE registry_gc_runs_total counter
registry_gc_runs_total{error="false",noop="false",worker="foo"} 1
registry_gc_runs_total{error="false",noop="true",worker="foo"} 1
registry_gc_runs_total{error="true",noop="false",worker="foo"} 1
registry_gc_runs_total{error="true",noop="true",worker="foo"} 2
`)
	durationFullName := fmt.Sprintf("%s_%s_%s", metrics.NamespacePrefix, subsystem, runDurationName)
	totalFullName := fmt.Sprintf("%s_%s_%s", metrics.NamespacePrefix, subsystem, runTotalName)

	err := testutil.GatherAndCompare(prometheus.DefaultGatherer, &expected, durationFullName, totalFullName)
	require.NoError(t, err)
}

func TestReviewPostpone(t *testing.T) {
	ReviewPostpone("foo")
	ReviewPostpone("foo")
	ReviewPostpone("bar")

	var expected bytes.Buffer
	expected.WriteString(`
# HELP registry_gc_postpones_total A counter for online GC review postpones.
# TYPE registry_gc_postpones_total counter
registry_gc_postpones_total{worker="bar"} 1
registry_gc_postpones_total{worker="foo"} 2
`)
	totalFullName := fmt.Sprintf("%s_%s_%s", metrics.NamespacePrefix, subsystem, postponeTotalName)

	err := testutil.GatherAndCompare(prometheus.DefaultGatherer, &expected, totalFullName)
	require.NoError(t, err)
}

func TestStorageDeleteBytes(t *testing.T) {
	StorageDeleteBytes(123, "foo")
	StorageDeleteBytes(321, "foo")
	StorageDeleteBytes(1, "bar")

	var expected bytes.Buffer
	expected.WriteString(`
# HELP registry_gc_storage_deleted_bytes_total A counter for bytes deleted from storage during online GC.
# TYPE registry_gc_storage_deleted_bytes_total counter
registry_gc_storage_deleted_bytes_total{media_type="bar"} 1
registry_gc_storage_deleted_bytes_total{media_type="foo"} 444
`)
	totalFullName := fmt.Sprintf("%s_%s_%s", metrics.NamespacePrefix, subsystem, storageDeleteBytesTotalName)

	err := testutil.GatherAndCompare(prometheus.DefaultGatherer, &expected, totalFullName)
	require.NoError(t, err)
}
