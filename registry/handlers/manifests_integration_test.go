// +build integration

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
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
	pk  libtrust.PrivateKey

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

	sm, err := schema1.Sign(&manifest, e.pk)
	require.NoError(t, err)

	dgst := digest.FromBytes(sm.Canonical)

	err = dbPutManifestSchema1(e.ctx, e.db, dgst, sm, sm.Canonical, path)
	require.NoError(t, err)

	return dgst
}

func (e *env) uploadManifestListToDB(t *testing.T, manifestList manifestlist.ManifestList, repoName string) digest.Digest {
	t.Helper()

	path, err := reference.WithName(repoName)
	require.NoError(t, err)

	deserializedManifestList, err := manifestlist.FromDescriptors(manifestList.Manifests)
	require.NoError(t, err)

	_, payload, err := deserializedManifestList.Payload()
	require.NoError(t, err)

	dgst := digest.FromBytes(payload)

	err = dbPutManifestList(e.ctx, e.db, dgst, deserializedManifestList, payload, path)
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

	pk, err := libtrust.GenerateECP256PrivateKey()
	require.NoError(t, err)

	env := &env{ctx: context.Background(), pk: pk}

	initDatabase(t, env)

	return env
}

func TestTagManifest_Schema2(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagschema2"
	tagName := "tagschema2"

	// Upload schema2 manifest and tag it.
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	err := dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)
}

func TestTagManifest_Schema1(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagschema1"
	tagName := "tagschema1"

	// Upload schema1 manifest and tag it.
	manifest := seedRandomSchema1Manifest(t, env)
	manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

	err := dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)
}

func TestTagManifest_Idempotent(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagidempotent"
	tagName := "tagidempotent"

	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	// Retag the manifest with the same tag.
	err := dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)

	err = dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)
}

func TestTagManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagmanifestlist"
	tagName := "tagmanifestlist"

	// Upload and tag manifest list.
	manifestList := seedRandomManifestList(t, env)
	manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

	err := dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestListTag(t, env, manifestListDigest, tagName, repoPath)
}

func TestTagManifestList_Idempotent(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifestList := seedRandomManifestList(t, env)

	repoPath := "manifestdb/tagmanifestlistidempotent"
	tagName := "tagmanifestlist"

	manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

	// Tag manifest list twice.
	err := dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestListTag(t, env, manifestListDigest, tagName, repoPath)

	err = dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestListTag(t, env, manifestListDigest, tagName, repoPath)
}

func TestTagManifest_TagReplacesPreviousManifest(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	tagName := "tagschema2latest"
	repoPath := "manifestdb/tagschema2replace"

	// Upload and tag old manifest.
	oldManifest, oldCfgPayload := seedRandomSchema2Manifest(t, env)
	oldManifestDigest := env.uploadSchema2ManifestToDB(t, oldManifest, oldCfgPayload, repoPath)

	err := dbTagManifest(env.ctx, env.db, oldManifestDigest, tagName, repoPath)
	require.NoError(t, err)

	// Ensure tag is initially associated with correct manifest.
	verifyManifestTag(t, env, oldManifestDigest, tagName, repoPath)

	// Upload a new manifest and tag it with the same tag.
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	err = dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)
}

func TestTagManifest_TagReplacesPreviousManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagmanifestlist"
	tagName := "tagmanifestreplacemanifestlist"

	// Upload and tag an initial manifest list.
	manifestList := seedRandomManifestList(t, env)
	manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

	err := dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	// Ensure tag is initially associated with correct manifest list.
	verifyManifestListTag(t, env, manifestListDigest, tagName, repoPath)

	// Upload a schema2 manifest, retagging it with the same tag used for the manifest list.
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	err = dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	// Ensure tag is now associated with correct manifest.
	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)
}

func TestTagManifestList_TagReplacesPreviousManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagmanifestlist"
	tagName := "tagmanifestreplacemanifestlist"

	// Upload and tag an initial manifest list.
	oldManifestList := seedRandomManifestList(t, env)
	oldManifestListDigest := env.uploadManifestListToDB(t, oldManifestList, repoPath)

	err := dbTagManifestList(env.ctx, env.db, oldManifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	// Ensure that tag is initially associated with correct manifest list.
	verifyManifestListTag(t, env, oldManifestListDigest, tagName, repoPath)

	// Upload a new manifest list, retagging it with the same tag used
	// for the old manifest list.
	manifestList := seedRandomManifestList(t, env)
	manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

	err = dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	verifyManifestListTag(t, env, manifestListDigest, tagName, repoPath)
}

func TestTagManifestList_TagReplacesPreviousManifest(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/tagmanifestlist"
	tagName := "tagmanifestlistlatest"

	// Upload a regular schema2 manifest.
	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	err := dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.NoError(t, err)

	// Ensure tag is now associated with correct manifest.
	verifyManifestTag(t, env, manifestDigest, tagName, repoPath)

	// Upload a manifest list, retagging it with the same tag used for the manifest.
	manifestList := seedRandomManifestList(t, env)
	manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

	err = dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.NoError(t, err)

	// Ensure that tag is associated with correct manifest list.
	verifyManifestListTag(t, env, manifestListDigest, tagName, repoPath)
}

func TestTagManifest_MissingManifest(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	tagName := "tagmanifestmissing"
	repoPath := "manifestdb/tagmanifestmissing"
	manifestDigest := digest.FromString("invalid digest")

	err := dbTagManifest(env.ctx, env.db, manifestDigest, tagName, repoPath)
	require.Error(t, err)

	// Ensure tag is not present in database.
	repoStore := datastore.NewRepositoryStore(env.db)
	dbRepo, err := repoStore.CreateOrFindByPath(env.ctx, repoPath)
	require.NoError(t, err)

	dbTag, err := repoStore.FindTagByName(env.ctx, dbRepo, tagName)
	require.NoError(t, err)
	require.Nil(t, dbTag)
}

func TestTagManifestList_MissingManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	tagName := "tagmanifestlistmissing"
	repoPath := "manifestdb/tagmanifestlistmissing"
	manifestListDigest := digest.FromString("invalid digest")

	err := dbTagManifestList(env.ctx, env.db, manifestListDigest, tagName, repoPath)
	require.Error(t, err)

	// Ensure tag is not present in database.
	repoStore := datastore.NewRepositoryStore(env.db)
	dbRepo, err := repoStore.CreateOrFindByPath(env.ctx, repoPath)
	require.NoError(t, err)

	dbTag, err := repoStore.FindTagByName(env.ctx, dbRepo, tagName)
	require.NoError(t, err)
	require.Nil(t, dbTag)
}

func TestPutManifestSchema2(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)

	repoPath := "manifestdb/happypath"
	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

	verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
}

func TestPutManifestSchema2_Idempotent(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
	repoPath := "manifestdb/idempotent"

	manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)
	verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)

	manifestDigest = env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)
	verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
}

func TestPutManifestSchema2_MultipleRepositories(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest, cfgPayload := seedRandomSchema2Manifest(t, env)

	repoBasePath := "manifestdb/multirepo"

	for i := 0; i < 10; i++ {
		repoPath := fmt.Sprintf("%s%d", repoBasePath, i)
		manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

		verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
	}
}

func TestPutManifestSchema2_MultipleManifests(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/multimanifest"

	for i := 0; i < 10; i++ {
		manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
		manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

		verifySchema2Manifest(t, env, manifestDigest, manifest, cfgPayload, repoPath)
	}
}

func TestPutManifestSchema2_MissingLayer(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

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

func TestPutManifestSchema1(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest := seedRandomSchema1Manifest(t, env)

	repoPath := "manifestdb/happypathschema1"
	manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

	verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
}

func TestPutManifestSchema1_Idempotent(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest := seedRandomSchema1Manifest(t, env)
	repoPath := "manifestdb/idempotentschema1"

	manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)
	verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)

	manifestDigest = env.uploadSchema1ManifestToDB(t, manifest, repoPath)
	verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
}

func TestPutManifestSchema1_MultipleRepositories(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest := seedRandomSchema1Manifest(t, env)

	repoBasePath := "manifestdb/multireposchema1"

	for i := 0; i < 10; i++ {
		repoPath := fmt.Sprintf("%s%d", repoBasePath, i)
		manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

		verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
	}
}

func TestPutManifestSchema1_MultipleManifests(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/multimanifestschema1"

	for i := 0; i < 10; i++ {
		manifest := seedRandomSchema1Manifest(t, env)
		manifestDigest := env.uploadSchema1ManifestToDB(t, manifest, repoPath)

		verifySchema1Manifest(t, env, manifestDigest, manifest, repoPath)
	}
}

func TestPutManifestSchema1_MissingLayer(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifest := seedRandomSchema1Manifest(t, env)

	layerStore := datastore.NewLayerStore(env.db)

	// Remove layer from database before uploading manifest
	dbLayer, err := layerStore.FindByDigest(env.ctx, manifest.FSLayers[0].BlobSum)
	require.NoError(t, err)
	require.NotNil(t, dbLayer)

	layerStore.Delete(env.ctx, dbLayer.ID)
	require.NoError(t, err)

	// Try to put manifest with missing layer.
	sm, err := schema1.Sign(&manifest, env.pk)
	require.NoError(t, err)

	dgst := digest.FromBytes(sm.Canonical)
	path, err := reference.WithName("manifestdb/missinglayerschema1")

	err = dbPutManifestSchema1(env.ctx, env.db, dgst, sm, sm.Canonical, path)
	assert.Error(t, err)
}

func TestPutManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifestList := seedRandomManifestList(t, env)

	repoPath := "manifestdb/happypathmanifestlist"
	manifestDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

	verifyManifestList(t, env, manifestDigest, manifestList, repoPath)
}

func TestPutManifestList_Idempotent(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifestList := seedRandomManifestList(t, env)
	repoPath := "manifestdb/idempotentmanifestlist"

	manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)
	verifyManifestList(t, env, manifestListDigest, manifestList, repoPath)

	manifestListDigest = env.uploadManifestListToDB(t, manifestList, repoPath)
	verifyManifestList(t, env, manifestListDigest, manifestList, repoPath)
}

func TestPutManifestList_MultipleRepositories(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifestList := seedRandomManifestList(t, env)

	repoBasePath := "manifestdb/multirepomanifestlist"

	for i := 0; i < 10; i++ {
		repoPath := fmt.Sprintf("%s%d", repoBasePath, i)
		manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

		verifyManifestList(t, env, manifestListDigest, manifestList, repoPath)
	}
}

func TestPutManifestList_MultipleManifests(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	repoPath := "manifestdb/multimanifestlist"

	for i := 0; i < 10; i++ {
		manifestList := seedRandomManifestList(t, env)
		manifestListDigest := env.uploadManifestListToDB(t, manifestList, repoPath)

		verifyManifestList(t, env, manifestListDigest, manifestList, repoPath)
	}
}

func TestPutManifestList_MissingManifest(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	manifestList := seedRandomManifestList(t, env)

	mStore := datastore.NewManifestStore(env.db)

	// Remove manifest from database before uploading manifest list
	dbManifest, err := mStore.FindByDigest(env.ctx, manifestList.Manifests[0].Descriptor.Digest)
	require.NoError(t, err)
	require.NotNil(t, dbManifest)

	err = mStore.Delete(env.ctx, dbManifest.ID)
	require.NoError(t, err)

	// Try to put manifest list with missing manifest.
	deserializedManifestList, err := manifestlist.FromDescriptors(manifestList.Manifests)
	require.NoError(t, err)

	_, payload, err := deserializedManifestList.Payload()
	require.NoError(t, err)

	path, err := reference.WithName("manifestdb/missinglayer")
	require.NoError(t, err)

	dgst := digest.FromBytes(payload)

	err = dbPutManifestList(env.ctx, env.db, dgst, deserializedManifestList, payload, path)
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
	require.NotEmpty(t, dbRepos)

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
	require.NotEmpty(t, dbLayers)

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

	// Ensure only the canonical representation (without signatures) is written to the database.
	sm, err := schema1.Sign(&manifest, env.pk)
	require.NoError(t, err)

	require.EqualValues(t, sm.Canonical, dbManifest.Payload)
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
	require.NotEmpty(t, dbRepos)

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
	require.NotEmpty(t, dbLayers)

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

func verifyManifestList(t *testing.T, env *env, dgst digest.Digest, manifestList manifestlist.ManifestList, repoPath string) {
	t.Helper()

	mListStore := datastore.NewManifestListStore(env.db)

	// Ensure presence of manifest list.
	dbManifestList, err := mListStore.FindByDigest(env.ctx, dgst)
	require.NoError(t, err)
	require.NotNil(t, dbManifestList)

	// Ensure manifests are associated with manifest list.
	dbManifests, err := mListStore.Manifests(env.ctx, dbManifestList)
	require.NoError(t, err)
	require.NotEmpty(t, dbManifests)

	for _, dbManifest := range dbManifests {
		var foundManifest bool
		for _, manifestDesc := range manifestList.Manifests {
			if manifestDesc.Digest == dbManifest.Digest {
				foundManifest = true
			}
		}
		assert.True(t, foundManifest)
	}

	// Ensure manifest list is associated with repository.
	dbRepos, err := mListStore.Repositories(env.ctx, dbManifestList)
	require.NoError(t, err)
	require.NotEmpty(t, dbRepos)

	var foundRepo bool
	for _, repo := range dbRepos {
		if repo.Path == repoPath {
			foundRepo = true
			break
		}
	}
	assert.True(t, foundRepo)
}

func verifyManifestTag(t *testing.T, env *env, dgst digest.Digest, tagName, repoPath string) {
	t.Helper()

	// Ensure tag is present in database.
	repoStore := datastore.NewRepositoryStore(env.db)

	dbRepo, err := repoStore.FindByPath(env.ctx, repoPath)
	require.NoError(t, err)
	require.NotNil(t, dbRepo)

	dbTag, err := repoStore.FindTagByName(env.ctx, dbRepo, tagName)
	require.NoError(t, err)
	require.NotNil(t, dbTag)

	// Ensure that tag is associated with correct manifest.
	tagstore := datastore.NewTagStore(env.db)
	mStore := datastore.NewManifestStore(env.db)

	tagDBManifest, err := tagstore.Manifest(env.ctx, dbTag)
	require.NoError(t, err)
	require.NotNil(t, tagDBManifest)

	dbManifest, err := mStore.FindByDigest(env.ctx, dgst)
	require.NoError(t, err)
	require.NotNil(t, dbManifest)

	require.Equal(t, dbManifest, tagDBManifest)
}

func verifyManifestListTag(t *testing.T, env *env, dgst digest.Digest, tagName, repoPath string) {
	t.Helper()

	// Ensure tag is present in database.
	repoStore := datastore.NewRepositoryStore(env.db)

	dbRepo, err := repoStore.FindByPath(env.ctx, repoPath)
	require.NoError(t, err)
	require.NotNil(t, dbRepo)

	dbTag, err := repoStore.FindTagByName(env.ctx, dbRepo, tagName)
	require.NoError(t, err)
	require.NotNil(t, dbTag)

	// Ensure that tag is associated with correct manifest list.
	tagstore := datastore.NewTagStore(env.db)
	mListStore := datastore.NewManifestListStore(env.db)

	tagDBManifestList, err := tagstore.ManifestList(env.ctx, dbTag)
	require.NoError(t, err)
	require.NotNil(t, tagDBManifestList)

	dbManifestList, err := mListStore.FindByDigest(env.ctx, dgst)
	require.NoError(t, err)
	require.NotNil(t, dbManifestList)

	require.Equal(t, dbManifestList, tagDBManifestList)
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

// seedRandomManifestList generates a random manifest list and ensures that
// it and it's manifests are present in the database.
func seedRandomManifestList(t *testing.T, env *env) manifestlist.ManifestList {
	manifestList := manifestlist.ManifestList{
		Versioned: manifest.Versioned{
			SchemaVersion: 2,
			MediaType:     manifestlist.MediaTypeManifestList,
		},
		Manifests: make([]manifestlist.ManifestDescriptor, 4),
	}

	for i := range manifestList.Manifests {
		manifest, cfgPayload := seedRandomSchema2Manifest(t, env)
		repoPath := "manifestdb/seed"

		manifestDigest := env.uploadSchema2ManifestToDB(t, manifest, cfgPayload, repoPath)

		manifestList.Manifests[i] = manifestlist.ManifestDescriptor{
			Descriptor: distribution.Descriptor{
				Digest: manifestDigest,
			},
			Platform: manifestlist.PlatformSpec{
				Architecture: "amd64",
				OS:           "linux",
			},
		}
	}

	return manifestList
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

func TestDeleteManifestDB_Manifest(t *testing.T) {
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
	tags := []*models.Tag{
		{
			Name:         "1.0.0",
			RepositoryID: r.ID,
			ManifestID:   sql.NullInt64{Int64: m.ID, Valid: true},
		},
		{
			Name:         "latest",
			RepositoryID: r.ID,
			ManifestID:   sql.NullInt64{Int64: m.ID, Valid: true},
		},
	}
	for _, tag := range tags {
		err = tStore.Create(env.ctx, tag)
		require.NoError(t, err)
	}

	// Test

	err = dbDeleteManifest(env.ctx, env.db, r.Path, m.Digest)
	require.NoError(t, err)

	// the manifest should still be there (deleting it is GC's responsibility, if no other repo uses it)
	m2, err := mStore.FindByID(env.ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, m, m2)

	// but the manifest and repository association should not
	mm, err := rStore.Manifests(env.ctx, r)
	require.NoError(t, err)
	require.Empty(t, mm)

	// neither the tags
	for _, tag := range tags {
		tag, err = tStore.FindByID(env.ctx, tag.ID)
		require.NoError(t, err)
		require.Nil(t, tag)
	}
}

func TestDeleteManifestDB_ManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// Setup

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	// add a manifest list
	mlStore := datastore.NewManifestListStore(env.db)
	ml := &models.ManifestList{
		SchemaVersion: 2,
		MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
		Digest:        "sha256:dc27c897a7e24710a2821878456d56f3965df7cc27398460aa6f21f8b385d2d0",
		Payload:       json.RawMessage(`{"schemaVersion":2}`),
	}
	err = mlStore.Create(env.ctx, ml)
	require.NoError(t, err)

	// associate manifest list with repository
	err = rStore.AssociateManifestList(env.ctx, r, ml)
	require.NoError(t, err)

	// tag manifest list
	tStore := datastore.NewTagStore(env.db)
	tags := []*models.Tag{
		{
			Name:           "1.0.0",
			RepositoryID:   r.ID,
			ManifestListID: sql.NullInt64{Int64: ml.ID, Valid: true},
		},
		{
			Name:           "latest",
			RepositoryID:   r.ID,
			ManifestListID: sql.NullInt64{Int64: ml.ID, Valid: true},
		},
	}
	for _, tag := range tags {
		err = tStore.Create(env.ctx, tag)
		require.NoError(t, err)
	}

	// Test

	err = dbDeleteManifest(env.ctx, env.db, r.Path, ml.Digest)
	require.NoError(t, err)

	// the manifest list should still be there (deleting it is GC's responsibility, if no other repo uses it)
	ml2, err := mlStore.FindByID(env.ctx, ml.ID)
	require.NoError(t, err)
	require.Equal(t, ml, ml2)

	// but the manifest list and repository association should not
	mm, err := rStore.ManifestLists(env.ctx, r)
	require.NoError(t, err)
	require.Empty(t, mm)

	// neither the tags
	for _, tag := range tags {
		tag, err = tStore.FindByID(env.ctx, tag.ID)
		require.NoError(t, err)
		require.Nil(t, tag)
	}
}

func TestDeleteManifestDB_DissociatedManifest(t *testing.T) {
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

	// Test

	err = dbDeleteManifest(env.ctx, env.db, r.Path, m.Digest)
	require.Error(t, err, "no manifest or manifest list found in database")

	// the manifest should still be there (it was not associated with the repository)
	m2, err := mStore.FindByID(env.ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, m, m2)
}

func TestDeleteManifestDB_DissociatedManifestList(t *testing.T) {
	env := newEnv(t)
	defer env.shutdown(t)

	// Setup

	// build test repository
	rStore := datastore.NewRepositoryStore(env.db)
	r, err := rStore.CreateByPath(env.ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, r)

	// add a manifest list
	mlStore := datastore.NewManifestListStore(env.db)
	ml := &models.ManifestList{
		SchemaVersion: 2,
		MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
		Digest:        "sha256:dc27c897a7e24710a2821878456d56f3965df7cc27398460aa6f21f8b385d2d0",
		Payload:       json.RawMessage(`{"schemaVersion":2}`),
	}
	err = mlStore.Create(env.ctx, ml)
	require.NoError(t, err)

	// Test

	err = dbDeleteManifest(env.ctx, env.db, r.Path, ml.Digest)
	require.Error(t, err, "no manifest or manifest list found in database")

	// the manifest list should still be there (it was not associated with the repository)
	ml2, err := mlStore.FindByID(env.ctx, ml.ID)
	require.NoError(t, err)
	require.Equal(t, ml, ml2)
}
