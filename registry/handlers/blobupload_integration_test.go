// +build integration

package handlers

import (
	"math/rand"
	"testing"

	"github.com/docker/distribution"
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

func randomLayerDescriptor(t *testing.T) distribution.Descriptor {
	t.Helper()

	return distribution.Descriptor{
		MediaType: schema2.MediaTypeLayer,
		Digest:    randomDigest(t),
		Size:      rand.Int63n(10000),
	}
}

func descriptorFromLayer(t *testing.T, l *models.Layer) distribution.Descriptor {
	t.Helper()

	return distribution.Descriptor{
		MediaType: l.MediaType,
		Digest:    l.Digest,
		Size:      l.Size,
	}
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

func findLayer(t *testing.T, env *env, d digest.Digest) *models.Layer {
	t.Helper()

	rStore := datastore.NewLayerStore(env.db)
	l, err := rStore.FindByDigest(env.ctx, d)
	require.NoError(t, err)
	require.NotNil(t, l)

	return l
}

func TestDBMountBlob_NonExistentSourceRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	err := dbMountBlob(env.ctx, env.db, "from", "to", randomDigest(t))
	require.Error(t, err)
	require.Equal(t, err.Error(), "source repository not found in database")
}

func TestDBMountBlob_NonExistentBlob(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")

	err := dbMountBlob(env.ctx, env.db, fromRepo.Path, "to", randomDigest(t))
	require.Error(t, err)
	require.Equal(t, err.Error(), "blob not found in database")
}

func TestDBMountBlob_NonExistentBlobLinkInSourceRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")
	l := buildRandomLayer(t, env) // not linked in fromRepo

	err := dbMountBlob(env.ctx, env.db, fromRepo.Path, "to", l.Digest)
	require.Error(t, err)
	require.Equal(t, err.Error(), "blob not found in database")
}

func TestDBMountBlob_NonExistentDestinationRepo(t *testing.T) {
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

func TestDBMountBlob_AlreadyLinked(t *testing.T) {
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

func TestDBPutBlobUploadComplete_NonExistentRepoAndLayer(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	desc := randomLayerDescriptor(t)
	err := dbPutBlobUploadComplete(env.ctx, env.db, "foo", desc)
	require.NoError(t, err)

	// the layer should have been created
	l := findLayer(t, env, desc.Digest)
	// and so does the repository
	r := findRepository(t, env, "foo")
	// and the link between layer and repository
	require.True(t, isLayerLinked(t, env, r, l))
}

func TestDBPutBlobUploadComplete_NonExistentRepoAndExistentLayer(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	l := buildRandomLayer(t, env)

	desc := descriptorFromLayer(t, l)
	err := dbPutBlobUploadComplete(env.ctx, env.db, "foo", desc)
	require.NoError(t, err)

	// the repository should have been created
	r := findRepository(t, env, "foo")
	// and so does the link between layer and repository
	require.True(t, isLayerLinked(t, env, r, l))
}

func TestDBPutBlobUploadComplete_ExistentRepoAndNonExistentLayer(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	r := buildRepository(t, env, "foo")

	desc := randomLayerDescriptor(t)
	err := dbPutBlobUploadComplete(env.ctx, env.db, r.Path, desc)
	require.NoError(t, err)

	// the layer should have been created
	l := findLayer(t, env, desc.Digest)
	// and so does the link between layer and repository
	require.True(t, isLayerLinked(t, env, r, l))
}

func TestDBPutBlobUploadComplete_ExistentRepoAndLayerButNotLinked(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	r := buildRepository(t, env, "foo")
	l := buildRandomLayer(t, env)

	desc := descriptorFromLayer(t, l)
	err := dbPutBlobUploadComplete(env.ctx, env.db, r.Path, desc)
	require.NoError(t, err)

	// the link between layer and repository should have been created
	require.True(t, isLayerLinked(t, env, r, l))
}

func TestDBPutBlobUploadComplete_ExistentRepoAndLayerAlreadyLinked(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	r := buildRepository(t, env, "foo")
	l := buildRandomLayer(t, env)
	linkLayer(t, env, r, l)

	desc := descriptorFromLayer(t, l)
	err := dbPutBlobUploadComplete(env.ctx, env.db, r.Path, desc)
	require.NoError(t, err)

	// the link between layer and repository should remain
	require.True(t, isLayerLinked(t, env, r, l))
}
