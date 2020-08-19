// +build integration

package handlers

import (
	"context"
	"math/rand"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

type notFoundBlobStatter struct{}

func (bs *notFoundBlobStatter) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	return distribution.Descriptor{}, distribution.ErrBlobUnknown
}

type expectedBlobStatter struct {
	digest digest.Digest
}

func (bs *expectedBlobStatter) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	return distribution.Descriptor{Digest: bs.digest}, nil
}

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

func buildRandomBlob(t *testing.T, env *env) *models.Blob {
	t.Helper()

	bStore := datastore.NewBlobStore(env.db)

	b := &models.Blob{
		MediaType: "application/octect-stream",
		Digest:    randomDigest(t),
		Size:      rand.Int63n(10000),
	}
	err := bStore.Create(env.ctx, b)
	require.NoError(t, err)

	return b
}

func randomBlobDescriptor(t *testing.T) distribution.Descriptor {
	t.Helper()

	return distribution.Descriptor{
		MediaType: "application/octect-stream",
		Digest:    randomDigest(t),
		Size:      rand.Int63n(10000),
	}
}

func descriptorFromBlob(t *testing.T, b *models.Blob) distribution.Descriptor {
	t.Helper()

	return distribution.Descriptor{
		MediaType: b.MediaType,
		Digest:    b.Digest,
		Size:      b.Size,
	}
}

func linkBlob(t *testing.T, env *env, r *models.Repository, b *models.Blob) {
	t.Helper()

	rStore := datastore.NewRepositoryStore(env.db)
	err := rStore.LinkBlob(env.ctx, r, b)
	require.NoError(t, err)
}

func isBlobLinked(t *testing.T, env *env, r *models.Repository, b *models.Blob) bool {
	t.Helper()

	rStore := datastore.NewRepositoryStore(env.db)
	bb, err := rStore.Blobs(env.ctx, r)
	require.NoError(t, err)

	for _, blob := range bb {
		if blob.Digest == b.Digest {
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

func findBlob(t *testing.T, env *env, d digest.Digest) *models.Blob {
	t.Helper()

	rStore := datastore.NewBlobStore(env.db)
	b, err := rStore.FindByDigest(env.ctx, d)
	require.NoError(t, err)
	require.NotNil(t, b)

	return b
}

func TestDBMountBlob_NonExistentSourceRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// Test for cases where only the source repo does not exist.
	buildRepository(t, env, "to")

	b := buildRandomBlob(t, env)

	err := dbMountBlob(env.ctx, env.db, &expectedBlobStatter{b.Digest}, "from", "to", b.Digest, false)
	require.Error(t, err)
	require.Equal(t, "source repository not found in database", err.Error())
}

func TestDBMountBlob_NonExistentSourceRepoFilesystemFallback(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// Test for cases where only the source repo does not exist.
	buildRepository(t, env, "to")

	b := buildRandomBlob(t, env)

	err := dbMountBlob(env.ctx, env.db, &expectedBlobStatter{b.Digest}, "from", "to", b.Digest, true)
	require.NoError(t, err)

	srcRepo := findRepository(t, env, "from")
	require.True(t, isBlobLinked(t, env, srcRepo, b))

	destRepo := findRepository(t, env, "to")
	require.True(t, isBlobLinked(t, env, destRepo, b))
}

func TestDBMountBlob_NonExistentBlob(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")

	err := dbMountBlob(env.ctx, env.db, &notFoundBlobStatter{}, fromRepo.Path, "to", randomDigest(t), false)
	require.Error(t, err)
	require.Equal(t, "blob not found in database", err.Error())
}

func TestDBMountBlob_NonExistentBlobFilesystemFallback(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")

	err := dbMountBlob(env.ctx, env.db, &notFoundBlobStatter{}, fromRepo.Path, "to", randomDigest(t), true)
	require.Error(t, err)
	require.Equal(t, "unknown blob", err.Error())
}

func TestDBMountBlob_NonExistentBlobLinkInSourceRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")
	b := buildRandomBlob(t, env) // not linked in fromRepo

	err := dbMountBlob(env.ctx, env.db, &notFoundBlobStatter{}, fromRepo.Path, "to", b.Digest, false)
	require.Error(t, err)
	require.Equal(t, "blob not found in database", err.Error())
}

func TestDBMountBlob_NonExistentBlobLinkInSourceRepoFilesystemFallback(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")
	b := buildRandomBlob(t, env) // not linked in fromRepo

	err := dbMountBlob(env.ctx, env.db, &notFoundBlobStatter{}, fromRepo.Path, "to", b.Digest, true)
	require.Error(t, err)
	require.Equal(t, "unknown blob", err.Error())
}

func TestDBMountBlob_NonExistentDestinationRepo(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	fromRepo := buildRepository(t, env, "from")
	b := buildRandomBlob(t, env)
	linkBlob(t, env, fromRepo, b)

	err := dbMountBlob(env.ctx, env.db, &expectedBlobStatter{digest: b.Digest}, fromRepo.Path, "to", b.Digest, false)
	require.NoError(t, err)

	destRepo := findRepository(t, env, "to")
	require.True(t, isBlobLinked(t, env, destRepo, b))
}

func TestDBMountBlob_AlreadyLinked(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	b := buildRandomBlob(t, env)

	fromRepo := buildRepository(t, env, "from")
	linkBlob(t, env, fromRepo, b)

	destRepo := buildRepository(t, env, "to")
	linkBlob(t, env, destRepo, b)

	err := dbMountBlob(env.ctx, env.db, &expectedBlobStatter{digest: b.Digest}, fromRepo.Path, destRepo.Path, b.Digest, false)
	require.NoError(t, err)

	require.True(t, isBlobLinked(t, env, destRepo, b))
}

func TestDBMountBlob_SourceBlobOnlyInFileSystemFilesystemFallback(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	b := &models.Blob{Digest: randomDigest(t)}

	fromRepo := buildRepository(t, env, "from")
	destRepo := buildRepository(t, env, "to")

	err := dbMountBlob(env.ctx, env.db, &expectedBlobStatter{digest: b.Digest}, fromRepo.Path, destRepo.Path, b.Digest, true)
	require.NoError(t, err)

	require.True(t, isBlobLinked(t, env, fromRepo, b))
	require.True(t, isBlobLinked(t, env, destRepo, b))
}

func TestDBPutBlobUploadComplete_NonExistentRepoAndBlob(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	desc := randomBlobDescriptor(t)
	err := dbPutBlobUploadComplete(env.ctx, env.db, "foo", desc)
	require.NoError(t, err)

	// the blob should have been created
	b := findBlob(t, env, desc.Digest)
	// and so does the repository
	r := findRepository(t, env, "foo")
	// and the link between blob and repository
	require.True(t, isBlobLinked(t, env, r, b))
}

func TestDBPutBlobUploadComplete_NonExistentRepoAndExistentBlob(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	b := buildRandomBlob(t, env)

	desc := descriptorFromBlob(t, b)
	err := dbPutBlobUploadComplete(env.ctx, env.db, "foo", desc)
	require.NoError(t, err)

	// the repository should have been created
	r := findRepository(t, env, "foo")
	// and so does the link between blob and repository
	require.True(t, isBlobLinked(t, env, r, b))
}

func TestDBPutBlobUploadComplete_ExistentRepoAndNonExistentBlob(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	r := buildRepository(t, env, "foo")

	desc := randomBlobDescriptor(t)
	err := dbPutBlobUploadComplete(env.ctx, env.db, r.Path, desc)
	require.NoError(t, err)

	// the blob should have been created
	b := findBlob(t, env, desc.Digest)
	// and so does the link between blob and repository
	require.True(t, isBlobLinked(t, env, r, b))
}

func TestDBPutBlobUploadComplete_ExistentRepoAndBlobButNotLinked(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	r := buildRepository(t, env, "foo")
	b := buildRandomBlob(t, env)

	desc := descriptorFromBlob(t, b)
	err := dbPutBlobUploadComplete(env.ctx, env.db, r.Path, desc)
	require.NoError(t, err)

	// the link between blob and repository should have been created
	require.True(t, isBlobLinked(t, env, r, b))
}

func TestDBPutBlobUploadComplete_ExistentRepoAndBlobAlreadyLinked(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	r := buildRepository(t, env, "foo")
	b := buildRandomBlob(t, env)
	linkBlob(t, env, r, b)

	desc := descriptorFromBlob(t, b)
	err := dbPutBlobUploadComplete(env.ctx, env.db, r.Path, desc)
	require.NoError(t, err)

	// the link between blob and repository should remain
	require.True(t, isBlobLinked(t, env, r, b))
}
