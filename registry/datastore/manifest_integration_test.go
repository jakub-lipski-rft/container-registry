// +build integration

package datastore_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/docker/distribution/manifest/manifestlist"

	"github.com/docker/distribution/registry/datastore"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadManifestFixtures(tb testing.TB) {
	testutil.ReloadFixtures(
		tb, suite.db, suite.basePath,
		// Manifest has a relationship with Repository, ManifestConfiguration and ManifestLayer (insert order matters)
		testutil.RepositoriesTable, testutil.ManifestConfigurationsTable, testutil.ManifestsTable,
		testutil.RepositoryManifestsTable, testutil.LayersTable, testutil.ManifestLayersTable,
	)
}

func unloadManifestFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(
		suite.db,
		// Manifest has a relationship with Repository, ManifestConfiguration and ManifestLayer (insert order matters)
		testutil.RepositoriesTable, testutil.ManifestConfigurationsTable, testutil.ManifestsTable,
		testutil.RepositoryManifestsTable, testutil.LayersTable, testutil.ManifestLayersTable,
	))
}

func TestManifestStore_FindByID(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	expected := &models.Manifest{
		ID:              1,
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
		ConfigurationID: sql.NullInt64{Int64: 1, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
		CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", m.CreatedAt.Location()),
	}
	require.Equal(t, expected, m)
}

func TestManifestStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, m)
	require.EqualError(t, err, "manifest not found")
}

func TestManifestStore_FindByDigest(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByDigest(suite.ctx, "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f")
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	excepted := &models.Manifest{
		ID:              2,
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
		ConfigurationID: sql.NullInt64{Int64: 2, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
		CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", m.CreatedAt.Location()),
	}
	require.Equal(t, excepted, m)
}

func TestManifestStore_FindByDigest_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	m, err := s.FindByDigest(suite.ctx, "sha256:78cc6esuite.db833591fb9d0ec5a0ac141571de42a6c3f23f042598810815b08417f2")
	require.Nil(t, m)
	require.EqualError(t, err, "manifest not found")
}

func TestManifestStore_FindAll(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	mm, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	local := mm[0].CreatedAt.Location()
	expected := models.Manifests{
		{
			ID:              1,
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			ConfigurationID: sql.NullInt64{Int64: 1, Valid: true},
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:              2,
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			ConfigurationID: sql.NullInt64{Int64: 2, Valid: true},
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:              3,
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6",
			ConfigurationID: sql.NullInt64{Int64: 3, Valid: true},
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":6775,"digest":"sha256:33f3ef3322b28ecfc368872e621ab715a04865471c47ca7426f3e93846157780"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":27091819,"digest":"sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":23882259,"digest":"sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":203,"digest":"sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":107,"digest":"sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:            4,
			SchemaVersion: 1,
			MediaType:     "application/vnd.docker.distribution.manifest.v1+json",
			Digest:        "sha256:ea1650093606d9e76dfc78b986d57daea6108af2d5a9114a98d7198548bfdfc7",
			Payload:       json.RawMessage(`{"schemaVersion":1,"name":"gitlab-org/gitlab-test/frontend","tag":"0.0.1","architecture":"amd64","fsLayers":[{"blobSum":"sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b"},{"blobSum":"sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af"},{"blobSum":"sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa"}],"history":[{"v1Compatibility":"{\"architecture\":\"amd64\",\"config\":{\"Hostname\":\"\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"/bin/sh\"],\"ArgsEscaped\":true,\"Image\":\"sha256:74df73bb19fbfc7fb5ab9a8234b3d98ee2fb92df5b824496679802685205ab8c\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":null,\"Labels\":null},\"container\":\"fb71ddde5f6411a82eb056a9190f0cc1c80d7f77a8509ee90a2054428edb0024\",\"container_config\":{\"Hostname\":\"fb71ddde5f64\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"/bin/sh\",\"-c\",\"#(nop) \",\"CMD [\\\"/bin/sh\\\"]\"],\"ArgsEscaped\":true,\"Image\":\"sha256:74df73bb19fbfc7fb5ab9a8234b3d98ee2fb92df5b824496679802685205ab8c\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":null,\"Labels\":{}},\"created\":\"2020-03-23T21:19:34.196162891Z\",\"docker_version\":\"18.09.7\",\"id\":\"13787be01505ffa9179a780b616b953330baedfca1667797057aa3af67e8b39d\",\"os\":\"linux\",\"parent\":\"c6875a916c6940e6590b05b29f484059b82e19ca0eed100e2e805aebd98614b8\",\"throwaway\":true}"},{"v1Compatibility":"{\"id\":\"c6875a916c6940e6590b05b29f484059b82e19ca0eed100e2e805aebd98614b8\",\"created\":\"2020-03-23T21:19:34.027725872Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop) ADD file:0c4555f363c2672e350001f1293e689875a3760afe7b3f9146886afe67121cba in / \"]}}"}],"signatures":[{"header":{"jwk":{"crv":"P-256","kid":"SVNG:A2VR:TQJG:H626:HBKH:6WBU:GFBH:3YNI:425G:MDXK:ULXZ:CENN","kty":"EC","x":"daLesX_y73FSCFCaBuCR8offV_m7XEohHZJ9z-6WvOM","y":"pLEDUlQMDiEQqheWYVC55BPIB0m8BIhI-fxQBCH_wA0"},"alg":"ES256"},"signature":"mqA4qF-St1HTNsjHzhgnHBeN38ptKJOi4wSeH4xc_FCEPv0OchAUJC6v2gYTP4TwostmX-AB1_z3jo9G_ZuX5w","protected":"eyJmb3JtYXRMZW5ndGgiOjIxNTQsImZvcm1hdFRhaWwiOiJDbjAiLCJ0aW1lIjoiMjAyMC0wNC0xNVQwODoxMzowNVoifQ"}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-15 09:47:26.461413", local),
		},
	}
	require.Equal(t, expected, mm)
}

func TestManifestStore_FindAll_NotFound(t *testing.T) {
	unloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	mm, err := s.FindAll(suite.ctx)
	require.Empty(t, mm)
	require.NoError(t, err)
}

func TestManifestStore_Count(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/manifests.sql
	require.Equal(t, 4, count)
}

func TestManifestStore_Layers(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	ll, err := s.Layers(suite.ctx, &models.Manifest{ID: 1})
	require.NoError(t, err)

	// see testdata/fixtures/manifest_layers.sql
	local := ll[0].CreatedAt.Location()
	expected := models.Layers{
		{
			ID:        1,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
			Size:      2802957,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
		{
			ID:        2,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
			Size:      108,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
	}
	require.Equal(t, expected, ll)
}

func TestManifestStore_Lists(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	ll, err := s.Lists(suite.ctx, &models.Manifest{ID: 1})
	require.NoError(t, err)

	// see testdata/fixtures/manifest_layers.sql
	local := ll[0].CreatedAt.Location()
	expected := models.ManifestLists{
		{
			ID:            1,
			SchemaVersion: 2,
			MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
			Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", local),
		},
		{
			ID:            2,
			SchemaVersion: 2,
			MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
			Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":42212,"digest":"sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6","platform":{"architecture":"amd64","os":"linux"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:04.470711", local),
		},
	}
	require.Equal(t, expected, ll)
}

func TestManifestStore_Repositories(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	rr, err := s.Repositories(suite.ctx, &models.Manifest{ID: 1})
	require.NoError(t, err)

	// see testdata/fixtures/repository_manifests.sql
	local := rr[0].CreatedAt.Location()
	expected := models.Repositories{
		{
			ID:        3,
			Name:      "backend",
			Path:      "gitlab-org/gitlab-test/backend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:42:12.566212", local),
		},
	}
	require.Equal(t, expected, rr)
}

func TestManifestStore_Create(t *testing.T) {
	unloadManifestFixtures(t)
	reloadRepositoryFixtures(t)
	reloadManifestConfigurationFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:46b163863b462eadc1b17dca382ccbfb08a853cffc79e2049607f95455cc44fa",
		ConfigurationID: sql.NullInt64{Int64: 1, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}
	err := s.Create(suite.ctx, m)

	require.NoError(t, err)
	require.NotEmpty(t, m.ID)
	require.NotEmpty(t, m.CreatedAt)
}

func TestManifestStore_Create_NonUniqueDigestFails(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6",
		ConfigurationID: sql.NullInt64{Int64: 3, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}
	err := s.Create(suite.ctx, m)
	require.Error(t, err)
}

func TestManifestStore_Update(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	update := &models.Manifest{
		ID:              3,
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:2a878989cffc014c2ffbb8da930b28b00be1ba2dd2910e05996e238f42344a37",
		ConfigurationID: sql.NullInt64{Int64: 3, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}
	err := s.Update(suite.ctx, update)
	require.NoError(t, err)

	m, err := s.FindByID(suite.ctx, update.ID)
	require.NoError(t, err)

	update.CreatedAt = m.CreatedAt
	require.Equal(t, update, m)
}

func TestManifestStore_Update_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	update := &models.Manifest{
		ID:              100,
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:2a878989cffc014c2ffbb8da930b28b00be1ba2dd2910e05996e238f42344a37",
		ConfigurationID: sql.NullInt64{Int64: 3, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}

	err := s.Update(suite.ctx, update)
	require.EqualError(t, err, "manifest not found")
}

func TestManifestStore_Mark(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	m := &models.Manifest{ID: 3}
	err := s.Mark(suite.ctx, m)
	require.NoError(t, err)

	require.True(t, m.MarkedAt.Valid)
	require.NotEmpty(t, m.MarkedAt.Time)
}

func TestManifestStore_Mark_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	m := &models.Manifest{ID: 100}
	err := s.Mark(suite.ctx, m)
	require.EqualError(t, err, "manifest not found")
}

func TestManifestStore_AssociateLayer(t *testing.T) {
	reloadManifestFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.ManifestLayersTable))

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{ID: 1}
	l := &models.Layer{ID: 3}

	err := s.AssociateLayer(suite.ctx, m, l)
	require.NoError(t, err)

	ll, err := s.Layers(suite.ctx, m)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_layers.sql
	local := ll[0].CreatedAt.Location()
	expected := models.Layers{
		{
			ID:        3,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1",
			Size:      109,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:06:32.856423", local),
		},
	}
	require.Equal(t, expected, ll)
}

func TestManifestStore_AssociateLayer_AlreadyAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	// see testdata/fixtures/manifest_layers.sql
	m := &models.Manifest{ID: 1}
	l := &models.Layer{ID: 1}
	err := s.AssociateLayer(suite.ctx, m, l)
	require.NoError(t, err)
}

func TestManifestStore_DissociateLayer(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{ID: 1}
	l := &models.Layer{ID: 1}

	err := s.DissociateLayer(suite.ctx, m, l)
	require.NoError(t, err)

	ll, err := s.Layers(suite.ctx, m)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_layers.sql
	local := ll[0].CreatedAt.Location()
	unexpected := models.Layers{
		{
			ID:        1,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
			Size:      2802957,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
	}
	require.NotContains(t, ll, unexpected)
}

func TestManifestStore_DissociateLayer_NotAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	m := &models.Manifest{ID: 1}
	l := &models.Layer{ID: 5}

	err := s.DissociateLayer(suite.ctx, m, l)
	require.NoError(t, err)
}

func TestManifestStore_SoftDelete(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)

	m := &models.Manifest{ID: 3}
	err := s.SoftDelete(suite.ctx, m)
	require.NoError(t, err)

	m, err = s.FindByID(suite.ctx, m.ID)
	require.NoError(t, err)

	require.True(t, m.DeletedAt.Valid)
	require.NotEmpty(t, m.DeletedAt.Time)
}

func TestManifestStore_SoftDelete_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)

	m := &models.Manifest{ID: 100}
	err := s.SoftDelete(suite.ctx, m)
	require.EqualError(t, err, "manifest not found")
}

func TestManifestStore_Delete(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewManifestStore(suite.db)
	err := s.Delete(suite.ctx, 3)
	require.NoError(t, err)

	_, err = s.FindByID(suite.ctx, 3)
	require.EqualError(t, err, "manifest not found")
}

func TestManifestStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewManifestStore(suite.db)
	err := s.Delete(suite.ctx, 5)
	require.EqualError(t, err, "manifest not found")
}
