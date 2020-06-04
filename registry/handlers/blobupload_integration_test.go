// +build integration

package handlers

import (
	"math/rand"
	"testing"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func buildRepository(t *testing.T, env *env, path string) *models.Repository {
	t.Helper()

	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, path)
	require.NoError(t, err)
	require.NotNil(t, r)

	return r
}

func randomDigest(t *testing.T) digest.Digest {
	t.Helper()

	bytes := make([]byte, rand.Intn(10000))
	_, err := rand.Read(bytes)
	require.NoError(t, err)

	return digest.FromBytes(bytes)
}

func buildRandomLayer(t *testing.T, env *env) *models.Layer {
	t.Helper()

	lStore := datastore.NewLayerStore(env.db)

	l := &models.Layer{
		MediaType: schema2.MediaTypeLayer,
		Digest:    randomDigest(t),
		Size:      rand.Int63n(10000),
	}
	err := lStore.Create(env.ctx, l)
	require.NoError(t, err)

	return l
}

func linkLayer(t *testing.T, env *env, r *models.Repository, l *models.Layer) {
	t.Helper()

	rStore := datastore.NewRepositoryStore(env.db)
	err := rStore.LinkLayer(env.ctx, r, l)
	require.NoError(t, err)
}

func isLayerLinked(t *testing.T, env *env, r *models.Repository, l *models.Layer) bool {
	t.Helper()

	rStore := datastore.NewRepositoryStore(env.db)
	ll, err := rStore.Layers(env.ctx, r)
	require.NoError(t, err)

	for _, layer := range ll {
		if layer.Digest == l.Digest {
			return true
		}
	}

	return false
}

func findRepository(t *testing.T, env *env, path string) *models.Repository {
	t.Helper()

	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.FindByPath(env.ctx, path)
	require.NoError(t, err)
	require.NotNil(t, r)

	return r
}

func TestStartBlobUpload_dbMountBlob_NonExistentSourceRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	err := dbMountBlob(env.ctx, env.db, "from", "to", randomDigest(t))
	require.Error(t, err)
	require.Equal(t, err.Error(), "source repository not found in database")
}

func TestStartBlobUpload_dbMountBlob_NonExistentBlob(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")

	err := dbMountBlob(env.ctx, env.db, fromRepo.Path, "to", randomDigest(t))
	require.Error(t, err)
	require.Equal(t, err.Error(), "blob not found in database")
}

func TestStartBlobUpload_dbMountBlob_NonExistentBlobLinkInSourceRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")
	l := buildRandomLayer(t, env) // not linked in fromRepo

	err := dbMountBlob(env.ctx, env.db, fromRepo.Path, "to", l.Digest)
	require.Error(t, err)
	require.Equal(t, err.Error(), "blob not found in database")
}

func TestStartBlobUpload_dbMountBlob_NonExistentDestinationRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")
	l := buildRandomLayer(t, env)
	linkLayer(t, env, fromRepo, l)

	err := dbMountBlob(env.ctx, env.db, fromRepo.Path, "to", l.Digest)
	require.NoError(t, err)

	destRepo := findRepository(t, env, "to")
	require.True(t, isLayerLinked(t, env, destRepo, l))
}

func TestStartBlobUpload_dbMountBlob_AlreadyLinked(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	l := buildRandomLayer(t, env)

	fromRepo := buildRepository(t, env, "from")
	linkLayer(t, env, fromRepo, l)

	destRepo := buildRepository(t, env, "to")
	linkLayer(t, env, destRepo, l)

	err := dbMountBlob(env.ctx, env.db, fromRepo.Path, destRepo.Path, l.Digest)
	require.NoError(t, err)

	require.True(t, isLayerLinked(t, env, destRepo, l))
}
