// +build integration

package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/datastore"
	dbtestutil "github.com/docker/distribution/registry/datastore/testutil"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/inmemory"
	"github.com/docker/distribution/testutil"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type env struct {
	ctx      context.Context
	driver   driver.StorageDriver
	db       *datastore.DB
	registry distribution.Namespace
	repo     distribution.Repository
	regOpts  []storage.RegistryOption
}

func (e *env) isDatabaseEnabled() bool {
	return os.Getenv("REGISTRY_DATABASE_ENABLED") == "true"
}

func (e *env) Shutdown(t *testing.T) {
	t.Helper()

	if !e.isDatabaseEnabled() {
		return
	}

	err := dbtestutil.TruncateAllTables(e.db)
	require.NoError(t, err)

	err = e.db.Close()
	require.NoError(t, err)
}

func initDatabase(t *testing.T, env *env) {
	t.Helper()

	if !env.isDatabaseEnabled() {
		return
	}

	db, err := dbtestutil.NewDB()
	require.NoError(t, err)

	env.db = db

	err = db.MigrateUp()
	require.NoError(t, err)

	env.regOpts = append(env.regOpts, storage.Database(db))
}

func newEnv(t *testing.T, repoName string) *env {
	t.Helper()

	env := &env{
		ctx:    context.Background(),
		driver: inmemory.New(),
	}

	initDatabase(t, env)

	reg, err := storage.NewRegistry(env.ctx, env.driver, env.regOpts...)
	require.NoError(t, err)

	env.registry = reg

	n, err := reference.WithName(repoName)
	require.NoError(t, err)

	repo, err := env.registry.Repository(env.ctx, n)
	require.NoError(t, err)

	env.repo = repo

	return env
}

func TestLayerUpload(t *testing.T) {
	env := newEnv(t, "layer/upload")
	defer env.Shutdown(t)

	testFilesystemLayerUpload(t, env)
	testDockerConfigurationPaylodUpload(t, env)
	testHelmConfigurationPaylodUpload(t, env)
	testMalformedPayloadUpload(t, env)
	testUnformattedPayloadUpload(t, env)
	testInvalidLayerUpload(t, env)
}

func testFilesystemLayerUpload(t *testing.T, env *env) {
	layer, dgst, err := testutil.CreateRandomTarFile()
	require.NoError(t, err)

	testLayerUpload(t, env, layer, dgst)
}

func testEmptyLayerUpload(t *testing.T, env *env) {
	testLayerUpload(t, env, bytes.NewReader([]byte{}), digest.FromBytes([]byte{}))
}
func testDockerConfigurationPaylodUpload(t *testing.T, env *env) {
	basePath, err := os.Getwd()
	require.NoError(t, err)

	path := filepath.Join(basePath, "testdata", "fixtures", "blobwriter", "docker_configuration.json")

	dockerPayload, err := ioutil.ReadFile(path)
	require.NoErrorf(t, err, "error reading fixture")

	testLayerUpload(t, env, bytes.NewReader(dockerPayload), digest.FromBytes(dockerPayload))
}

func testHelmConfigurationPaylodUpload(t *testing.T, env *env) {
	helmPayload := `{"name":"e-helm","version":"latest","description":"Sample Helm Chart","apiVersion":"v2","appVersion":"1.16.0","type":"application"}`

	testLayerUpload(t, env, strings.NewReader(helmPayload), digest.FromString(helmPayload))
}

func testMalformedPayloadUpload(t *testing.T, env *env) {
	malformedPayload := `{"invalid":"json",`
	testLayerUpload(t, env, strings.NewReader(malformedPayload), digest.FromString(malformedPayload))
}
func testUnformattedPayloadUpload(t *testing.T, env *env) {
	unformattedPayload := "unformatted string"
	testLayerUpload(t, env, strings.NewReader(unformattedPayload), digest.FromString(unformattedPayload))
}

func testLayerUpload(t *testing.T, env *env, layer io.ReadSeeker, dgst digest.Digest) {
	blobService := env.repo.Blobs(env.ctx)
	wr, err := blobService.Create(env.ctx)
	require.NoError(t, err)

	_, err = io.Copy(wr, layer)
	require.NoError(t, err)

	_, err = wr.Commit(env.ctx, distribution.Descriptor{Digest: dgst})
	require.NoError(t, err)

	if env.isDatabaseEnabled() {
		ls := datastore.NewLayerStore(env.db)

		layer, err := ls.FindByDigest(env.ctx, dgst)
		require.NoError(t, err)

		require.NotNil(t, layer)

		assert.Equal(t, layer.Size, wr.Size(), "db layer size and writer size should match")

		assert.Equal(t, layer.MediaType, "application/octet-stream", "db layer mediaType should be application/octet-stream")
	}

	desc, err := blobService.Stat(env.ctx, dgst)
	require.NoError(t, err)

	assert.Equal(t, desc.Size, wr.Size(), "blob size and writer size should match")

	assert.Equal(t, desc.MediaType, "application/octet-stream", "blob mediaType should be application/octet-stream")
}

func testInvalidLayerUpload(t *testing.T, env *env) {
	blobService := env.repo.Blobs(env.ctx)
	wr, err := blobService.Create(env.ctx)
	require.NoError(t, err)

	layer := strings.NewReader("test layer")
	dgst := digest.FromString("invalid digest")

	_, err = io.Copy(wr, layer)
	require.NoError(t, err)

	_, err = wr.Commit(env.ctx, distribution.Descriptor{Digest: dgst})
	if assert.Error(t, err) {
		assert.Equal(t, distribution.ErrBlobInvalidDigest{Digest: dgst, Reason: errors.New("content does not match digest")}, err)
	}

	if env.isDatabaseEnabled() {
		ls := datastore.NewLayerStore(env.db)

		layer, err := ls.FindByDigest(env.ctx, dgst)
		require.NoError(t, err)

		assert.Nil(t, layer, "layer should not present in database")
	}

	_, err = blobService.Stat(env.ctx, dgst)
	if assert.Error(t, err) {
		assert.Equal(t, distribution.ErrBlobUnknown, err)
	}
}
