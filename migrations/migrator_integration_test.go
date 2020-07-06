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
