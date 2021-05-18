// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/stretchr/testify/require"
)

func TestGCLayerLinkStore_FindAll(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewGCLayerLinkStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// The table is auto populated by the `gc_track_layer_blobs_trigger` trigger as soon as `reloadManifestFixtures`
	// loads the `layers` fixtures. See testdata/fixtures/layers.sql
	expected := []*models.GCLayerLink{
		{
			ID:           8,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      8,
			Digest:       "sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b",
		},
		{
			ID:           11,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      11,
			Digest:       "sha256:a0696058fc76fe6f456289f5611efe5c3411814e686f59f28b2e2069ed9e7d28",
		},
		{
			ID:           12,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      12,
			Digest:       "sha256:68ced04f60ab5c7a5f1d0b0b4e7572c5a4c8cce44866513d30d9df1a15277d6b",
		},
		{
			ID:           1,
			NamespaceID:  1,
			RepositoryID: 3,
			LayerID:      1,
			Digest:       "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		},
		{
			ID:           3,
			NamespaceID:  2,
			RepositoryID: 6,
			LayerID:      3,
			Digest:       "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		},
		{
			ID:           5,
			NamespaceID:  1,
			RepositoryID: 3,
			LayerID:      5,
			Digest:       "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		},
		{
			ID:           2,
			NamespaceID:  1,
			RepositoryID: 3,
			LayerID:      2,
			Digest:       "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
		},
		{
			ID:           4,
			NamespaceID:  2,
			RepositoryID: 6,
			LayerID:      4,
			Digest:       "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
		},
		{
			ID:           6,
			NamespaceID:  1,
			RepositoryID: 3,
			LayerID:      6,
			Digest:       "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
		},
		{
			ID:           7,
			NamespaceID:  1,
			RepositoryID: 3,
			LayerID:      7,
			Digest:       "sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1",
		},
		{
			ID:           9,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      9,
			Digest:       "sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af",
		},
		{
			ID:           10,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      10,
			Digest:       "sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa",
		},
		{
			ID:           13,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      13,
			Digest:       "sha256:c4039fd85dccc8e267c98447f8f1b27a402dbb4259d86586f4097acb5e6634af",
		},
		{
			ID:           14,
			NamespaceID:  1,
			RepositoryID: 4,
			LayerID:      14,
			Digest:       "sha256:c16ce02d3d6132f7059bf7e9ff6205cbf43e86c538ef981c37598afd27d01efa",
		},
	}

	require.Equal(t, expected, rr)
}

func TestGCLayerLinkStore_FindAll_NotFound(t *testing.T) {
	unloadManifestFixtures(t)

	s := datastore.NewGCLayerLinkStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestGcLayerLinkStore_Count(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewGCLayerLinkStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_blobs_layers.sql
	require.Equal(t, 14, count)
}
