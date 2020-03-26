// +build integration

package datastore_test

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/distribution/registry/datastore"
)

func newDSN(tb testing.TB) *datastore.DSN {
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
			dsnFactory: newDSN,
			wantErr:    false,
		},
		{
			name: "error",
			dsnFactory: func(tb testing.TB) *datastore.DSN {
				dsn := newDSN(tb)
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
				require.NoError(t, err)
				require.IsType(t, new(datastore.DB), db)
			}
		})
	}
}
