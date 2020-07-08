// +build integration

package migrations_test

import (
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
	plan, err := m.UpNPlan(len(all) - 1 - 2)
	require.NoError(t, err)

	allExceptLastTwoPlan := allPlan[:len(allPlan)-1-2]
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
