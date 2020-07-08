// +build integration

package migrations_test

import (
	"sort"
	"testing"

	"github.com/docker/distribution/migrations"
	"github.com/docker/distribution/registry/datastore/testutil"

	"github.com/stretchr/testify/require"
)

func TestMigrator_Version(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Up())

	latest, err := m.LatestVersion()
	require.NoError(t, err)

	current, err := m.Version()
	require.NoError(t, err)
	require.Equal(t, latest, current)
}

func TestMigrator_Version_NoMigrations(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Down())
	defer m.Up()

	v, err := m.Version()
	require.NoError(t, err)
	require.Empty(t, v)
}

func TestMigrator_Up(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Up())

	currentVersion, err := m.Version()
	require.NoError(t, err)

	v, err := m.LatestVersion()
	require.NoError(t, err)
	require.Equal(t, v, currentVersion)
}

func TestMigrator_UpN(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Down())
	defer m.Up()

	// apply all except the last two
	all := migrations.All()
	n := len(all) - 1 - 2
	nth := all[n-1]

	err = m.UpN(n)
	require.NoError(t, err)

	v, err := m.Version()
	require.NoError(t, err)
	require.Equal(t, nth.ID, v)

	// resume and apply the remaining
	err = m.UpN(0)
	require.NoError(t, err)

	v, err = m.Version()
	require.NoError(t, err)
	require.Equal(t, all[len(all)-1].ID, v)

	// make sure it's idempotent
	err = m.UpN(100)
	require.NoError(t, err)

	v2, err := m.Version()
	require.NoError(t, err)
	require.Equal(t, v, v2)
}

func TestMigrator_UpNPlan(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Down())
	defer m.Up()

	all := migrations.All()

	var allPlan []string
	for _, migration := range all {
		allPlan = append(allPlan, migration.ID)
	}

	// plan all except the last two
	n := len(allPlan) - 1 - 2
	allExceptLastTwoPlan := allPlan[:n]

	plan, err := m.UpNPlan(n)
	require.NoError(t, err)
	require.Equal(t, allExceptLastTwoPlan, plan)

	// apply two migrations and re-plan all (the first two shouldn't be part of the plan anymore)
	err = m.UpN(2)
	require.NoError(t, err)

	plan, err = m.UpNPlan(0)
	require.NoError(t, err)

	allExceptFirstTwoPlan := allPlan[2:]
	require.Equal(t, allExceptFirstTwoPlan, plan)

	// make sure it's idempotent
	plan, err = m.UpNPlan(100)
	require.NoError(t, err)
	require.Equal(t, allExceptFirstTwoPlan, plan)
}

func TestMigrator_Down(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Down())
	defer m.Up()

	currentVersion, err := m.Version()
	require.NoError(t, err)
	require.Empty(t, currentVersion)
}

func TestMigrator_DownN(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Up())
	defer m.Up()

	// rollback all except the first two
	all := migrations.All()
	n := len(all) - 1 - 2
	third := all[2]

	err = m.DownN(n)
	require.NoError(t, err)

	v, err := m.Version()
	require.NoError(t, err)
	require.Equal(t, third.ID, v)

	// resume and rollback the remaining two
	err = m.DownN(0)
	require.NoError(t, err)

	v, err = m.Version()
	require.NoError(t, err)
	require.Empty(t, v)

	// make sure it's idempotent
	err = m.DownN(100)
	require.NoError(t, err)

	v, err = m.Version()
	require.NoError(t, err)
	require.Empty(t, v)
}

func TestMigrator_DownNPlan(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	m := migrations.NewMigrator(db.DB)
	require.NoError(t, m.Up())

	all := migrations.All()

	var allPlan []string

	for _, migration := range all {
		allPlan = append(allPlan, migration.ID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(allPlan))) // down migrations are applied in reverse order

	// plan all except the last two
	n := len(allPlan) - 1 - 2
	allExceptLastTwoPlan := allPlan[:n]

	plan, err := m.DownNPlan(n)
	require.NoError(t, err)
	require.Equal(t, allExceptLastTwoPlan, plan)

	// apply two migrations and re-plan all (the first two shouldn't be part of the plan anymore)
	err = m.DownN(2)
	require.NoError(t, err)

	plan, err = m.DownNPlan(0)
	require.NoError(t, err)

	allExceptFirstTwoPlan := allPlan[2:]
	require.Equal(t, allExceptFirstTwoPlan, plan)

	// make sure it's idempotent
	plan, err = m.DownNPlan(100)
	require.NoError(t, err)
	require.Equal(t, allExceptFirstTwoPlan, plan)
}
