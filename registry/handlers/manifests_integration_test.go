// +build integration

package handlers

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	dbtestutil "github.com/docker/distribution/registry/datastore/testutil"
	"github.com/docker/libtrust"
)

type env struct {
	ctx context.Context
	db  *datastore.DB

	// isShutdown helps ensure that tests do not try to access the db after the
	// connection has been closed.
	isShutdown bool
}

func (e *env) isDatabaseEnabled() bool {
	return !e.isShutdown && os.Getenv("REGISTRY_DATABASE_ENABLED") == "true"
}

func (e *env) uploadLayerToDB(t *testing.T, desc distribution.Descriptor) {
	t.Helper()

	layerStore := datastore.NewLayerStore(e.db)

	dbLayer, err := layerStore.FindByDigest(e.ctx, desc.Digest)
	require.NoError(t, err)

	// Layer is already present.
	if dbLayer != nil {
		return
	}

	err = layerStore.Create(e.ctx, &models.Layer{
		MediaType: desc.MediaType,
		Digest:    desc.Digest,
		Size:      desc.Size,
	})
	require.NoError(t, err)
}

func (e *env) uploadSchema2ManifestToDB(t *testing.T, manifest schema2.Manifest, cfgPayload []byte, repoName string) digest.Digest {
	t.Helper()

	dManifest, err := schema2.FromStruct(manifest)
	require.NoError(t, err)

	path, err := reference.WithName(repoName)
	require.NoError(t, err)
	_, mPayload, err := dManifest.Payload()
	require.NoError(t, err)

	err = dbPutManifestSchema2(e.ctx, e.db, dManifest.Target().Digest, dManifest, mPayload, cfgPayload, path)
	require.NoError(t, err)

	return dManifest.Target().Digest
}

func (e *env) uploadSchema1ManifestToDB(t *testing.T, manifest schema1.Manifest, repoName string) digest.Digest {
	t.Helper()

	path, err := reference.WithName(repoName)
	require.NoError(t, err)

	pk, err := libtrust.GenerateECP256PrivateKey()
	require.NoError(t, err)
	sm, err := schema1.Sign(&manifest, pk)
	require.NoError(t, err)

	_, payload, err := sm.Payload()
	require.NoError(t, err)

	dgst := digest.FromBytes(payload)

	err = dbPutManifestSchema1(e.ctx, e.db, dgst, sm, payload, path)
	require.NoError(t, err)

	return dgst
}

func (e *env) shutdown(t *testing.T) {
	t.Helper()

	if !e.isDatabaseEnabled() {
		return
	}

	err := dbtestutil.TruncateAllTables(e.db)
	require.NoError(t, err)

	err = e.db.Close()
	require.NoError(t, err)

	e.isShutdown = true
}

func initDatabase(t *testing.T, env *env) {
	t.Helper()

	if !env.isDatabaseEnabled() {
		t.Skip("database connection is required for this test")
	}

	db, err := dbtestutil.NewDB()
	require.NoError(t, err)

	env.db = db

	err = db.MigrateUp()
	require.NoError(t, err)
}

func newEnv(t *testing.T) *env {
	t.Helper()

	env := &env{ctx: context.Background()}

	initDatabase(t, env)

	return env
}

func TestPutManifestSchema1DB(t *testing.T) {
	env1 := newEnv(t)
	defer env1.shutdown(t)

	testPutManifestSchmea1DB(t, env1)
	testPutManifestSchmea1DBIsIdempotent(t, env1)
	testPutManifestSchmea1DBMultipleRepositories(t, env1)
	testPutManifestSchmea1DBMultipleManifests(t, env1)
	testPutManifestSchmea1DBMissingLayer(t, env1)
}

func TestPutManifestSchema2DB(t *testing.T) {
	env1 := newEnv(t)
	defer env1.shutdown(t)

	testPutManifestSchmea2DB(t, env1)
	testPutManifestSchmea2DBIsIdempotent(t, env1)
	testPutManifestSchmea2DBMultipleRepositories(t, env1)
	testPutManifestSchmea2DBMultipleManifests(t, env1)
	testPutManifestSchmea2DBMissingLayer(t, env1)
}

func testPutManifestSchmea2DB(t *testing.T, env *env) {
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)

	repoPath := "manifestdb/happypath"
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
}

func testPutManifestSchmea2DBIsIdempotent(t *testing.T, env *env) {
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	repoPath := "manifestdb/idempotent"

	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)
	verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)

	manifestDigest = env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)
	verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
}

func testPutManifestSchmea2DBMultipleRepositories(t *testing.T, env *env) {
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)

	repoBasePath := "manifestdb/multirepo"

	for i := 0; i < 10; i++ {
		repoPath := fmt.Sprintf("%s%d", repoBasePath, i)
		manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

		verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
	}
}

func testPutManifestSchmea2DBMultipleManifests(t *testing.T, env *env) {
	repoPath := "manifestdb/multimanifest"

	for i := 0; i < 10; i++ {
		manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
		manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

		verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
	}
}

func testPutManifestSchmea2DBMissingLayer(t *testing.T, env *env) {
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)

	layerStore := datastore.NewLayerStore(env.db)

	// Remove layer from database before uploading manifest
	dbLayer, err := layerStore.FindByDigest(env.ctx, manifest.Layers[0].Digest)
	require.NoError(t, err)
	require.NotNil(t, dbLayer)

	layerStore.Delete(env.ctx, dbLayer.ID)
	require.NoError(t, err)

	// Try to put manifest with missing layer.
	dManifest, err := schema2.FromStruct(manifest)
	require.NoError(t, err)

	path, err := reference.WithName("manifestdb/missinglayer")
	require.NoError(t, err)
	_, mPayload, err := dManifest.Payload()
	require.NoError(t, err)

	err = dbPutManifestSchema2(env.ctx, env.db, dManifest.Target().Digest, dManifest, mPayload, cfgPayload, path)
	assert.Error(t, err)
}

func testPutManifestSchmea1DB(t *testing.T, env *env) {
	manifest := seedRandomSchema1Manifest(t, env)

	repoPath := "manifestdb/happypathschema1"
	manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

	verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
}

func testPutManifestSchmea1DBIsIdempotent(t *testing.T, env *env) {
	manifest := seedRandomSchema1Manifest(t, env)
	repoPath := "manifestdb/idempotentschema1"

	manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)
	verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)

	manifestDigest = env.uploadSchema1ManifestToDB(t, manifest, repoPath)
	verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
}

func testPutManifestSchmea1DBMultipleRepositories(t *testing.T, env *env) {
	manifest := seedRandomSchema1Manifest(t, env)

	repoBasePath := "manifestdb/multireposchema1"

	for i := 0; i < 10; i++ {
		repoPath := fmt.Sprintf("%s%d", repoBasePath, i)
		manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

		verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
	}
}

func testPutManifestSchmea1DBMultipleManifests(t *testing.T, env *env) {
	repoPath := "manifestdb/multimanifestschema1"

	for i := 0; i < 10; i++ {
		manifest := seedRandomSchema1Manifest(t, env)
		manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

		verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
	}
}

func testPutManifestSchmea1DBMissingLayer(t *testing.T, env *env) {
	manifest := seedRandomSchema1Manifest(t, env)

	layerStore := datastore.NewLayerStore(env.db)

	// Remove layer from database before uploading manifest
	dbLayer, err := layerStore.FindByDigest(env.ctx, manifest.FSLayers[0].BlobSum)
	require.NoError(t, err)
	require.NotNil(t, dbLayer)

	layerStore.Delete(env.ctx, dbLayer.ID)
	require.NoError(t, err)

	// Try to put manifest with missing layer.
	pk, err := libtrust.GenerateECP256PrivateKey()
	require.NoError(t, err)
	sm, err := schema1.Sign(&manifest, pk)
	require.NoError(t, err)

	_, payload, err := sm.Payload()
	require.NoError(t, err)

	dgst := digest.FromBytes(payload)
	path, err := reference.WithName("manifestdb/missinglayerschema1")

	err = dbPutManifestSchema1(env.ctx, env.db, dgst, sm, payload, path)
	assert.Error(t, err)
}

func verifySchema1Manifest(t *testing.T, env *env, dgst digest.Digest, manifest schema1.Manifest, repoPath string) {
	t.Helper()

	mStore := datastore.NewManifestStore(env.db)

	// Ensure presence of manifest.
	dbManifest, err := mStore.FindByDigest(env.ctx, dgst)
	require.NoError(t, err)
	assert.NotNil(t, dbManifest)

	// Ensure repository is associated with manifest.
	dbRepos, err := mStore.Repositories(env.ctx, dbManifest)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRepos)

	var foundRepo bool
	for _, repo := range dbRepos {
		if repo.Path == repoPath {
			foundRepo = true
			break
		}
	}
	assert.True(t, foundRepo)

	// Ensure presence of each layer.
	dbLayers, err := mStore.Layers(env.ctx, dbManifest)
	require.NoError(t, err)
	assert.NotEmpty(t, dbLayers)

	for _, fsLayer := range manifest.FSLayers {
		var foundLayer bool
		for _, layer := range dbLayers {
			if layer.Digest == fsLayer.BlobSum {
				foundLayer = true
				break
			}
		}
		assert.True(t, foundLayer)
	}
}

func verifySchema2Manifest(t *testing.T, env *env, dgst digest.Digest, manifest schema2.Manifest, cfgPayload []byte, repoPath string) {
	t.Helper()

	mStore := datastore.NewManifestStore(env.db)

	// Ensure pressence of manifest.
	dbManifest, err := mStore.FindByDigest(env.ctx, dgst)
	require.NoError(t, err)
	assert.NotNil(t, dbManifest)

	// Ensure repositry is associated with manifest.
	dbRepos, err := mStore.Repositories(env.ctx, dbManifest)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRepos)

	var foundRepo bool
	for _, repo := range dbRepos {
		if repo.Path == repoPath {
			foundRepo = true
			break
		}
	}
	assert.True(t, foundRepo)

	// Ensure manifest configuration is associated with manifest and has the
	// correct payload.
	dbMCfg, err := mStore.Config(env.ctx, dbManifest)
	require.NoError(t, err)
	assert.NotNil(t, dbMCfg)
	assert.EqualValues(t, cfgPayload, dbMCfg.Payload)

	// Ensure presence of each layer.
	dbLayers, err := mStore.Layers(env.ctx, dbManifest)
	require.NoError(t, err)
	assert.NotEmpty(t, dbLayers)

	for _, desc := range manifest.Layers {
		var foundLayer bool
		for _, layer := range dbLayers {
			if layer.Digest == desc.Digest &&
				layer.Size == desc.Size {
				foundLayer = true
				break
			}
		}
		assert.True(t, foundLayer)
	}
}

// seedRandomSchema1Manifest generates a random schema1 manifest and ensures
// that its layers are present in the database.
func seedRandomSchema1Manifest(t *testing.T, env *env) schema1.Manifest {
	t.Helper()

	manifest := schema1.Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 1,
			MediaType:     schema1.MediaTypeManifest,
		},
		FSLayers: make([]schema1.FSLayer, 4),
	}

	for i := range manifest.FSLayers {
		_, desc := generateRandomLayer()
		env.uploadLayerToDB(t, desc)
		manifest.FSLayers[i].BlobSum = desc.Digest
	}

	return manifest
}

// seedRandomSchema2Manifest generates a random schema2 manifest and ensures
// that its config payload blob and layers are present in the database.
func seedRandomSchema2Manifest(t *testing.T, env *env) (schema2.Manifest, []byte) {
	t.Helper()

	manifest := schema2.Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 2,
			MediaType:     schema2.MediaTypeManifest,
		},
		Layers: make([]distribution.Descriptor, 4),
	}

	cfgPayload, cfgDesc := generateRandomLayer()
	env.uploadLayerToDB(t, cfgDesc)
	manifest.Config = cfgDesc

	for i := range manifest.Layers {
		_, desc := generateRandomLayer()
		env.uploadLayerToDB(t, desc)
		manifest.Layers[i] = desc
	}

	return manifest, cfgPayload
}

// generateRandomLayer generates a random layer payload and distribution.Descriptor
func generateRandomLayer() ([]byte, distribution.Descriptor) {
	content := make([]byte, 16)
	rand.Read(content)

	return content, distribution.Descriptor{
		Size:      int64(len(content)),
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
	}
}
