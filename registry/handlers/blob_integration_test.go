// +build integration

package handlers

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/stretchr/testify/require"
)

func TestDeleteBlobDB(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// Setup

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	// add layer
	lStore := datastore.NewLayerStore(env.db)
	l := &models.Layer{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		Size:      2802957,
	}
	err = lStore.Create(env.ctx, l)
	require.NoError(t, err)
	require.NotEmpty(t, r.ID)

	// link layer to repository
	err = rStore.LinkLayer(env.ctx, r, l)
	require.NoError(t, err)

	// make sure it's linked
	ll, err := rStore.Layers(env.ctx, r)
	require.NoError(t, err)
	require.NotNil(t, ll)
	require.Contains(t, ll, l)

	// Test

	err = dbDeleteBlob(env.ctx, env.db, r.Path, l.Digest)
	require.NoError(t, err)

	// the layer should still be there
	l2, err := lStore.FindByID(env.ctx, l.ID)
	require.NoError(t, err)
	require.NotNil(t, l2)

	// but not the link for the repository
	ll2, err := rStore.Layers(env.ctx, r)
	require.NoError(t, err)
	require.NotContains(t, ll2, l)
}

func TestDeleteBlobDB_RepositoryNotFound(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	err := dbDeleteBlob(env.ctx, env.db, "foo", "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9")
	require.Error(t, err)
}

func TestDeleteBlobDB_BlobNotFound(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	err = dbDeleteBlob(env.ctx, env.db, r.Path, "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9")
	require.Error(t, err)
}
