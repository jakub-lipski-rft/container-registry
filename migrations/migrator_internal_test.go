package migrations

import (
	"database/sql"
	"testing"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"
)

func TestMigrator_LatestVersion(t *testing.T) {
	m := NewMigrator(&sql.DB{})

	v, err := m.LatestVersion()
	require.NoError(t, err)

	id := versionFromID(allMigrations[len(allMigrations)-1].Id)
	require.Equal(t, v, id)
}

func TestMigrator_LatestVersion_NoMigrations(t *testing.T) {
	// backup known migrations
	bkp := make([]*migrate.Migration, len(allMigrations))
	for i, m := range allMigrations {
		// create shallow copy and capture its address
		v := *m
		bkp[i] = &v
	}

	// reset known migrations and defer restore from backup
	allMigrations = []*migrate.Migration{}
	defer func() { allMigrations = bkp }()

	// test
	m := NewMigrator(&sql.DB{})
	v, err := m.LatestVersion()
	require.NoError(t, err)
	require.Empty(t, v)
}
