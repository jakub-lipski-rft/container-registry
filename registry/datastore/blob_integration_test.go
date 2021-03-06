// +build integration

package datastore_test

import (
	"testing"

	"github.com/opencontainers/go-digest"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadBlobFixtures(tb testing.TB) {
	testutil.ReloadFixtures(tb, suite.db, suite.basePath,
		testutil.NamespacesTable, testutil.RepositoriesTable, testutil.BlobsTable, testutil.RepositoryBlobsTable)
}

func unloadBlobFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(suite.db,
		testutil.NamespacesTable, testutil.RepositoriesTable, testutil.BlobsTable, testutil.RepositoryBlobsTable))
}

func TestBlobStore_ImplementsReaderAndWriter(t *testing.T) {
	require.Implements(t, (*datastore.BlobStore)(nil), datastore.NewBlobStore(suite.db))
}

func TestBlobStore_FindByDigest(t *testing.T) {
	reloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	b, err := s.FindByDigest(suite.ctx, "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21")
	require.NoError(t, err)

	// see testdata/fixtures/blobs.sql
	excepted := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
		Size:      108,
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", b.CreatedAt.Location()),
	}
	require.Equal(t, excepted, b)
}

func TestBlobStore_FindByDigest_NotFound(t *testing.T) {
	s := datastore.NewBlobStore(suite.db)
	b, err := s.FindByDigest(suite.ctx, "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e22")
	require.Nil(t, b)
	require.NoError(t, err)
}

func TestBlobStore_All(t *testing.T) {
	reloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	bb, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/blobs.sql
	local := bb[0].CreatedAt.Location()
	expected := models.Blobs{
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b",
			Size:      27091819,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28",
			Size:      107,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
			Size:      2802957,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
		{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Digest:    "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
			Size:      321,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:57:23.405516", local),
		},
		{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Digest:    "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
			Size:      123,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:56:26.573726", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
			Size:      108,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1",
			Size:      109,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:06:32.856423", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af",
			Size:      23882259,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa",
			Size:      203,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Digest:    "sha256:33f3ef3322b28ecfc368872e621ab715a04865471c47ca7426f3e93846157780",
			Size:      252,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:57:23.405516", local),
		},
	}

	require.Equal(t, expected, bb)
}

func TestBlobStore_All_NotFound(t *testing.T) {
	unloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	bb, err := s.FindAll(suite.ctx)
	require.Empty(t, bb)
	require.NoError(t, err)
}

func TestBlobStore_Count(t *testing.T) {
	reloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/blobs.sql
	require.Equal(t, 10, count)
}

func TestBlobStore_Create(t *testing.T) {
	unloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	b := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:1d9136cd62c9b60083de7763cfac547b1e571d10648393ade10325055a810556",
		Size:      203,
	}
	err := s.Create(suite.ctx, b)

	require.NoError(t, err)
	require.NotEmpty(t, b.CreatedAt)
}

func TestBlobStore_CreateOrFind(t *testing.T) {
	unloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	tmp := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:1d9136cd62c9b60083de7763cfac547b1e571d10648393ade10325055a810556",
		Size:      203,
	}

	// create non existing blob
	b := &models.Blob{
		MediaType: tmp.MediaType,
		Digest:    tmp.Digest,
		Size:      tmp.Size,
	}
	err := s.CreateOrFind(suite.ctx, b)
	require.NoError(t, err)
	require.Equal(t, tmp.MediaType, b.MediaType)
	require.Equal(t, tmp.Digest, b.Digest)
	require.Equal(t, tmp.Size, b.Size)
	require.NotEmpty(t, b.CreatedAt)

	// attempt to create existing blob
	l2 := &models.Blob{
		MediaType: tmp.MediaType,
		Digest:    tmp.Digest,
		Size:      tmp.Size,
	}
	err = s.CreateOrFind(suite.ctx, l2)
	require.NoError(t, err)
	require.Equal(t, b, l2)
}

func TestBlobStore_Create_NonUniqueDigestFails(t *testing.T) {
	reloadBlobFixtures(t)

	s := datastore.NewBlobStore(suite.db)
	b := &models.Blob{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28",
		Size:      108,
	}
	err := s.Create(suite.ctx, b)
	require.Error(t, err)
}

func TestBlobStore_Delete(t *testing.T) {
	reloadBlobFixtures(t)

	dgst := digest.Digest("sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9")
	s := datastore.NewBlobStore(suite.db)
	err := s.Delete(suite.ctx, dgst)
	require.NoError(t, err)

	b, err := s.FindByDigest(suite.ctx, dgst)
	require.Nil(t, b)
}

func TestBlobStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewBlobStore(suite.db)
	err := s.Delete(suite.ctx, "sha256:b9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9")
	require.EqualError(t, err, datastore.ErrNotFound.Error())
}
