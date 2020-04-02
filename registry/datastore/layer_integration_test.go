// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadLayerFixtures(tb testing.TB) {
	testutil.ReloadFixtures(tb, suite.db, suite.basePath, testutil.LayersTable)
}

func unloadLayerFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(suite.db, testutil.LayersTable))
}

func TestLayerStore_FindByID(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	l, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
	excepted := &models.Layer{
		ID:        1,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		Size:      2802957,
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", l.CreatedAt.Location()),
	}
	require.Equal(t, excepted, l)
}

func TestLayerStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewLayerStore(suite.db)
	l, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, l)
	require.EqualError(t, err, "layer not found")
}

func TestLayerStore_FindByDigest(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	l, err := s.FindByDigest(suite.ctx, "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21")
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
	excepted := &models.Layer{
		ID:        2,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
		Size:      108,
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:05:35.338639", l.CreatedAt.Location()),
	}
	require.Equal(t, excepted, l)
}

func TestLayerStore_FindByDigest_NotFound(t *testing.T) {
	s := datastore.NewLayerStore(suite.db)
	l, err := s.FindByDigest(suite.ctx, "sha256:78cc6e833591fb9d0ec5a0ac141571de42a6c3f23f042598810815b08417f2")
	require.Nil(t, l)
	require.EqualError(t, err, "layer not found")
}

func TestLayerStore_All(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	ll, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
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
		{
			ID:        3,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1",
			Size:      109,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:06:32.856423", local),
		},
		{
			ID:        4,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b",
			Size:      27091819,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			ID:        5,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af",
			Size:      23882259,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			ID:        6,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa",
			Size:      203,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
		{
			ID:        7,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28",
			Size:      107,
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-04 20:08:00.405042", local),
		},
	}

	require.Equal(t, expected, ll)
}

func TestLayerStore_All_NotFound(t *testing.T) {
	unloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	ll, err := s.FindAll(suite.ctx)
	require.Empty(t, ll)
	require.NoError(t, err)
}

func TestLayerStore_Count(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/layers.sql
	require.Equal(t, 7, count)
}

func TestLayerStore_Create(t *testing.T) {
	unloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	l := &models.Layer{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:1d9136cd62c9b60083de7763cfac547b1e571d10648393ade10325055a810556",
		Size:      203,
	}
	err := s.Create(suite.ctx, l)

	require.NoError(t, err)
	require.NotEmpty(t, l.ID)
	require.NotEmpty(t, l.CreatedAt)
}

func TestLayerStore_Create_NonUniqueDigestFails(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	l := &models.Layer{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28",
		Size:      108,
	}
	err := s.Create(suite.ctx, l)
	require.Error(t, err)
}

func TestLayerStore_Update(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	update := &models.Layer{
		ID:        7,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Digest:    "sha256:16844840277191fb603cc971dd59c97f05e12b49ae33766d33ae1b1333ab4f2a",
		Size:      642,
	}
	err := s.Update(suite.ctx, update)
	require.NoError(t, err)

	l, err := s.FindByID(suite.ctx, update.ID)
	require.NoError(t, err)

	update.CreatedAt = l.CreatedAt
	require.Equal(t, update, l)
}

func TestLayerStore_Update_NotFound(t *testing.T) {
	s := datastore.NewLayerStore(suite.db)

	update := &models.Layer{
		ID:        100,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
	}
	err := s.Update(suite.ctx, update)
	require.EqualError(t, err, "layer not found")
}

func TestLayerStore_Mark(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)

	r := &models.Layer{ID: 3}
	err := s.Mark(suite.ctx, r)
	require.NoError(t, err)

	require.True(t, r.MarkedAt.Valid)
	require.NotEmpty(t, r.MarkedAt.Time)
}

func TestLayerStore_Mark_NotFound(t *testing.T) {
	s := datastore.NewLayerStore(suite.db)

	r := &models.Layer{ID: 100}
	err := s.Mark(suite.ctx, r)
	require.EqualError(t, err, "layer not found")
}

func TestLayerStore_SoftDelete(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)

	l := &models.Layer{ID: 1}
	err := s.SoftDelete(suite.ctx, l)
	require.NoError(t, err)

	l, err = s.FindByID(suite.ctx, l.ID)
	require.NoError(t, err)

	require.True(t, l.DeletedAt.Valid)
	require.NotEmpty(t, l.DeletedAt.Time)
}

func TestLayerStore_SoftDelete_NotFound(t *testing.T) {
	s := datastore.NewLayerStore(suite.db)

	l := &models.Layer{ID: 100}
	err := s.SoftDelete(suite.ctx, l)
	require.EqualError(t, err, "layer not found")
}

func TestLayerStore_Delete(t *testing.T) {
	reloadLayerFixtures(t)

	s := datastore.NewLayerStore(suite.db)
	err := s.Delete(suite.ctx, 1)
	require.NoError(t, err)

	_, err = s.FindByID(suite.ctx, 1)
	require.EqualError(t, err, "layer not found")
}

func TestLayerStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewLayerStore(suite.db)
	err := s.Delete(suite.ctx, 100)
	require.EqualError(t, err, "layer not found")
}
