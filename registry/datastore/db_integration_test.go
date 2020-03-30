// +build integration

package datastore_test

import (
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/distribution/db/migrations"

	"github.com/stretchr/testify/require"

	"github.com/docker/distribution/registry/datastore"
)

func dsn(tb testing.TB) *datastore.DSN {
	tb.Helper()

	port, err := strconv.Atoi(os.Getenv("REGISTRY_DATABASE_PORT"))
	require.NoError(tb, err, "unexpected error parsing DSN port")

	return &datastore.DSN{
		Host:     os.Getenv("REGISTRY_DATABASE_HOST"),
		Port:     port,
		User:     os.Getenv("REGISTRY_DATABASE_USER"),
		Password: os.Getenv("REGISTRY_DATABASE_PASSWORD"),
		DBName:   os.Getenv("REGISTRY_DATABASE_DBNAME"),
		SSLMode:  os.Getenv("REGISTRY_DATABASE_SSLMODE"),
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dsnFactory func(tb testing.TB) *datastore.DSN
		wantErr    bool
	}{
		{
			name:       "success",
			dsnFactory: dsn,
			wantErr:    false,
		},
		{
			name: "error",
			dsnFactory: func(tb testing.TB) *datastore.DSN {
				dsn := dsn(tb)
				dsn.DBName = strconv.Itoa(rand.Intn(10))
				return dsn
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := datastore.Open(tt.dsnFactory(t))

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

func latestMigrationVersion(tb testing.TB) int {
	tb.Helper()

	all := migrations.AssetNames()
	sort.Strings(all)
	latest := all[len(all)-1]

	v, err := strconv.Atoi(strings.Split(latest, "_")[0])
	require.NoError(tb, err)

	return v
}

func TestDB_MigrateUp(t *testing.T) {
	db, err := datastore.Open(dsn(t))
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.MigrateUp())

	currentVersion, err := db.MigrateVersion()
	require.NoError(t, err)
	require.Equal(t, latestMigrationVersion(t), currentVersion)
}

func TestDB_MigrateDown(t *testing.T) {
	db, err := datastore.Open(dsn(t))
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.MigrateUp())
	require.NoError(t, db.MigrateDown())

	currentVersion, err := db.MigrateVersion()
	require.NoError(t, err)
	require.Equal(t, -1, currentVersion)
}
