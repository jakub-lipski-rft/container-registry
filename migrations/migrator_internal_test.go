package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrator_LatestVersion(t *testing.T) {
	m := NewMigrator(&sql.DB{})

	v, err := m.LatestVersion()
	require.NoError(t, err)

	require.Equal(t, v, allMigrations[len(allMigrations)-1].Id)
}

func TestMigrator_LatestVersion_NoMigrations(t *testing.T) {
	// backup known migrations
	bkp := make([]*Migration, len(allMigrations))
	for i, m := range allMigrations {
		// create shallow copy and capture its address
		v := *m
		bkp[i] = &v
	}

	// reset known migrations and defer restore from backup
	allMigrations = []*Migration{}
	defer func() { allMigrations = bkp }()

	// test
	m := NewMigrator(&sql.DB{})
	v, err := m.LatestVersion()
	require.NoError(t, err)
	require.Empty(t, v)
}
