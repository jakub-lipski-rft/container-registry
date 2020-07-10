package datastore_test

import (
	"fmt"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"

	"github.com/docker/distribution/registry/datastore"
)

func TestAlgorithm_String(t *testing.T) {
	tests := []struct {
		name string
		have datastore.DigestAlgorithm
		want string
	}{
		{
			name: "unknown",
			have: datastore.Unknown,
			want: "0",
		},
		{
			name: "sha256",
			have: datastore.SHA256,
			want: "1",
		},
		{
			name: "sha512",
			have: datastore.SHA512,
			want: "2",
		},
		{
			name: "zero value",
			have: 0,
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.have.String())
		})
	}
}

func TestNewDigestAlgorithm(t *testing.T) {
	tests := []struct {
		name    string
		have    digest.Algorithm
		want    datastore.DigestAlgorithm
		wantErr bool
	}{
		{
			name: "sha256",
			have: digest.SHA256,
			want: datastore.SHA256,
		},
		{
			name: "sha512",
			have: digest.SHA512,
			want: datastore.SHA512,
		},
		{
			name:    "unknown",
			have:    digest.SHA384,
			wantErr: true,
		},
		{
			name:    "zero value",
			have:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := datastore.NewDigestAlgorithm(tt.have)

			if tt.wantErr {
				require.Zero(t, got)
				require.Error(t, err)
				require.EqualError(t, err, fmt.Sprintf("unknown digest algorithm %q", tt.have))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDigestAlgorithm_Parse(t *testing.T) {
	tests := []struct {
		name    string
		have    datastore.DigestAlgorithm
		want    digest.Algorithm
		wantErr bool
	}{
		{
			name: "sha256",
			have: datastore.SHA256,
			want: digest.SHA256,
		},
		{
			name: "sha512",
			have: datastore.SHA512,
			want: digest.SHA512,
		},
		{
			name:    "unknown",
			have:    datastore.Unknown,
			wantErr: true,
		},
		{
			name:    "zero value",
			have:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.have.Parse()

			if tt.wantErr {
				require.Zero(t, got)
				require.Error(t, err)
				require.EqualError(t, err, fmt.Sprintf("unknown digest algorithm %q", tt.have))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
