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
				Host:        "127.0.0.1",
				Port:        5432,
				User:        "registry",
				Password:    "secret",
				DBName:      "registry_production",
				SSLMode:     "require",
				SSLCert:     "/path/to/client.crt",
				SSLKey:      "/path/to/client.key",
				SSLRootCert: "/path/to/root.crt",
			},
			out: "host=127.0.0.1 port=5432 user=registry password=secret dbname=registry_production sslmode=require sslcert=/path/to/client.crt sslkey=/path/to/client.key sslrootcert=/path/to/root.crt",
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

func TestDSN_Address(t *testing.T) {
	tests := []struct {
		name string
		arg  datastore.DSN
		out  string
	}{
		{name: "empty", arg: datastore.DSN{}, out: ":0"},
		{name: "no port", arg: datastore.DSN{Host: "127.0.0.1"}, out: "127.0.0.1:0"},
		{name: "no host", arg: datastore.DSN{Port: 5432}, out: ":5432"},
		{name: "full", arg: datastore.DSN{Host: "127.0.0.1", Port: 5432}, out: "127.0.0.1:5432"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.out, tt.arg.Address())
		})
	}
}
