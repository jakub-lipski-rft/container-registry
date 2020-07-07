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

	// add layer blob
	bStore := datastore.NewBlobStore(env.db)
	b := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		Size:      2802957,
	}
	err = bStore.Create(env.ctx, b)
	require.NoError(t, err)
	require.NotEmpty(t, r.ID)

	// link blob to repository
	err = rStore.LinkBlob(env.ctx, r, b)
	require.NoError(t, err)

	// make sure it's linked
	bb, err := rStore.Blobs(env.ctx, r)
	require.NoError(t, err)
	require.NotNil(t, bb)
	require.Contains(t, bb, b)

	// Test

	err = dbDeleteBlob(env.ctx, env.db, r.Path, b.Digest)
	require.NoError(t, err)

	// the layer blob should still be there
	b2, err := bStore.FindByID(env.ctx, b.ID)
	require.NoError(t, err)
	require.NotNil(t, b2)

	// but not the link for the repository
	bb2, err := rStore.Blobs(env.ctx, r)
	require.NoError(t, err)
	require.NotContains(t, bb2, b)
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
