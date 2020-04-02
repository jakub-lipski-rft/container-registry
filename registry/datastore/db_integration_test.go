// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dsnFactory func() (*datastore.DSN, error)
		wantErr    bool
	}{
		{
			name:       "success",
			dsnFactory: testutil.NewDSN,
			wantErr:    false,
		},
		{
			name: "error",
			dsnFactory: func() (*datastore.DSN, error) {
				dsn, err := testutil.NewDSN()
				if err != nil {
					return nil, err
				}
				dsn.DBName = "nonexistent"
				return dsn, nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn, err := tt.dsnFactory()
			require.NoError(t, err)

			db, err := datastore.Open(dsn)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				defer db.Close()
				require.NoError(t, err)
				require.IsType(t, new(datastore.DB), db)
			}
		})
	}
}

func TestDB_MigrateUp(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.MigrateUp())

	currentVersion, err := db.MigrateVersion()
	require.NoError(t, err)
	require.Equal(t, testutil.LatestMigrationVersion(t), currentVersion)
}

func TestDB_MigrateDown(t *testing.T) {
	db, err := testutil.NewDB()
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.MigrateDown())
	defer db.MigrateUp()

	currentVersion, err := db.MigrateVersion()
	require.NoError(t, err)
	require.Equal(t, -1, currentVersion)
}
