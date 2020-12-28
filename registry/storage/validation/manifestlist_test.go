package validation_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/storage/validation"
	"github.com/docker/distribution/testutil"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

// return a manifest descriptor with a pre-pushed manifest placeholder.
func makeManifestDescriptor(t *testing.T, repo distribution.Repository) manifestlist.ManifestDescriptor {
	ctx := context.Background()

	manifestService, err := testutil.MakeManifestService(repo)
	require.NoError(t, err)

	m := makeSchema2ManifestTemplate(t, repo)

	dm, err := schema2.FromStruct(m)
	require.NoError(t, err)

	dgst, err := manifestService.Put(ctx, dm)
	require.NoError(t, err)

	return manifestlist.ManifestDescriptor{Descriptor: distribution.Descriptor{Digest: dgst, MediaType: schema2.MediaTypeManifest}}
}

func TestVerifyManifest_ManifestList(t *testing.T) {
	ctx := context.Background()

	registry := createRegistry(t)
	repo := makeRepository(t, registry, "test")

	manifestService, err := testutil.MakeManifestService(repo)
	require.NoError(t, err)

	descriptors := []manifestlist.ManifestDescriptor{
		makeManifestDescriptor(t, repo),
	}

	dml, err := manifestlist.FromDescriptors(descriptors)
	require.NoError(t, err)

	v := validation.NewManifestListValidator(manifestService, false)

	err = v.Validate(ctx, dml)
	require.NoError(t, err)
}

func TestVerifyManifest_ManifestList_MissingManifest(t *testing.T) {
	ctx := context.Background()

	registry := createRegistry(t)
	repo := makeRepository(t, registry, "test")

	manifestService, err := testutil.MakeManifestService(repo)
	require.NoError(t, err)

	descriptors := []manifestlist.ManifestDescriptor{
		makeManifestDescriptor(t, repo),
		{Descriptor: distribution.Descriptor{Digest: digest.FromString("fake-digest"), MediaType: schema2.MediaTypeManifest}},
	}

	dml, err := manifestlist.FromDescriptors(descriptors)
	require.NoError(t, err)

	v := validation.NewManifestListValidator(manifestService, false)

	err = v.Validate(ctx, dml)
	require.EqualError(t, err, fmt.Sprintf("errors verifying manifest: unknown blob %s on manifest", digest.FromString("fake-digest")))

	// Ensure that this error is not reported if SkipDependencyVerification is true
	v = validation.NewManifestListValidator(manifestService, true)

	err = v.Validate(ctx, dml)
	require.NoError(t, err)
}

func TestVerifyManifest_ManifestList_InvalidSchemaVersion(t *testing.T) {
	ctx := context.Background()

	registry := createRegistry(t)
	repo := makeRepository(t, registry, "test")

	manifestService, err := testutil.MakeManifestService(repo)
	require.NoError(t, err)

	descriptors := []manifestlist.ManifestDescriptor{}

	dml, err := manifestlist.FromDescriptors(descriptors)
	require.NoError(t, err)

	dml.ManifestList.Versioned.SchemaVersion = 9001

	v := validation.NewManifestListValidator(manifestService, false)

	err = v.Validate(ctx, dml)
	require.EqualError(t, err, fmt.Sprintf("unrecognized manifest list schema version %d", dml.ManifestList.Versioned.SchemaVersion))
}
