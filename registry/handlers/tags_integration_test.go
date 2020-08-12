// +build integration

package handlers

import (
	"encoding/json"
	"testing"

	"github.com/docker/distribution/manifest/schema2"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/stretchr/testify/require"
)

func TestDeleteTagDB(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// Setup

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	// add a manifest
	mStore := datastore.NewManifestStore(env.db)
	m := &models.Manifest{
		SchemaVersion: 2,
		MediaType:     schema2.MediaTypeManifest,
		Digest:        "sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6",
		Payload:       json.RawMessage{},
	}
	err = mStore.Create(env.ctx, m)
	require.NoError(t, err)

	// associate manifest with repository
	err = rStore.AssociateManifest(env.ctx, r, m)
	require.NoError(t, err)

	// tag manifest
	tStore := datastore.NewTagStore(env.db)
	tag := &models.Tag{
		Name:         "latest",
		RepositoryID: r.ID,
		ManifestID:   m.ID,
	}
	err = tStore.Create(env.ctx, tag)
	require.NoError(t, err)

	// Test

	err = dbDeleteTag(env.ctx, env.db, r.Path, tag.Name, false)
	require.NoError(t, err)

	// the tag shouldn't be there
	tag, err = tStore.FindByID(env.ctx, tag.ID)
	require.NoError(t, err)
	require.Nil(t, tag)
}

func TestDeleteTagDB_RepositoryNotFound(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	err := dbDeleteTag(env.ctx, env.db, "foo", "bar", false)
	require.Error(t, err, "repository not found in database")

}

func TestDeleteTagDB_RepositoryNotFoundFallback(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	err := dbDeleteTag(env.ctx, env.db, "foo", "bar", true)
	require.NoError(t, err)

}

func TestDeleteTagDB_TagNotFound(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	err = dbDeleteTag(env.ctx, env.db, r.Path, "bar", false)
	require.Error(t, err, "repository not found in database")
}

func TestDeleteTagDB_TagNotFoundFallback(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	err = dbDeleteTag(env.ctx, env.db, r.Path, "bar", true)
	require.NoError(t, err)
}
