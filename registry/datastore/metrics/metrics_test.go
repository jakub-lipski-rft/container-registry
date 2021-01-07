package metrics

import (
	"bytes"
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

func TestStatementDuration(t *testing.T) {
	statementName := "foo_find_by_id"

	restore := mockTimeSince(10 * time.Millisecond)
	defer restore()
	StatementDuration(statementName)()

	mockTimeSince(20 * time.Millisecond)
	StatementDuration(statementName)()

	var expected bytes.Buffer
	expected.WriteString(`
# HELP registry_database_statement_duration_seconds Histogram of database statement durations in seconds
# TYPE registry_database_statement_duration_seconds histogram
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.005"} 0
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.01"} 1
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.025"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.05"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.1"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.25"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="0.5"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="1"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="2.5"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="5"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="10"} 2
registry_database_statement_duration_seconds_bucket{statement="foo_find_by_id",le="+Inf"} 2
registry_database_statement_duration_seconds_sum{statement="foo_find_by_id"} 0.03
registry_database_statement_duration_seconds_count{statement="foo_find_by_id"} 2
`)
	fullName := fmt.Sprintf("%s_%s_%s", metrics.NamespacePrefix, subsystem, statementDurationName)

	err := testutil.GatherAndCompare(prometheus.DefaultGatherer, &expected, fullName)
	require.NoError(t, err)
}
