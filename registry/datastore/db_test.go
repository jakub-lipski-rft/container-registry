package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/stretchr/testify/require"
)

func TestDSN_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  datastore.DSN
		out  string
	}{
		{name: "empty", arg: datastore.DSN{}, out: ""},
		{
			name: "full",
			arg: datastore.DSN{
				Host:     "127.0.0.1",
				Port:     5432,
				User:     "registry",
				Password: "secret",
				DBName:   "registry_production",
				SSLMode:  "require",
			},
			out: "host=127.0.0.1 port=5432 user=registry password=secret dbname=registry_production sslmode=require",
		},
		{
			name: "with zero port",
			arg: datastore.DSN{
				Port: 0,
			},
			out: "",
		},
		{
			name: "with spaces",
			arg: datastore.DSN{
				Password: "jw8s 0F4",
			},
			out: `password=jw8s\ 0F4`,
		},
		{
			name: "with quotes",
			arg: datastore.DSN{
				Password: "jw8s'0F4",
			},
			out: `password=jw8s\'0F4`,
		},
		{
			name: "with other special characters",
			arg: datastore.DSN{
				Password: "jw8s%^@0F4",
			},
			out: "password=jw8s%^@0F4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.out, tt.arg.String())
		})
	}
}
