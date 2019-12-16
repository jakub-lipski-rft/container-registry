package testutil

import (
	"fmt"
	"io"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/libtrust"
	"github.com/opencontainers/go-digest"
)

type Image struct {
	manifest       distribution.Manifest
	ManifestDigest digest.Digest
	Layers         map[digest.Digest]io.ReadSeeker
}

// MakeManifestList constructs a manifest list out of a list of manifest digests
func MakeManifestList(blobstatter distribution.BlobStatter, manifestDigests []digest.Digest) (*manifestlist.DeserializedManifestList, error) {
	ctx := context.Background()

	var manifestDescriptors []manifestlist.ManifestDescriptor
	for _, manifestDigest := range manifestDigests {
		descriptor, err := blobstatter.Stat(ctx, manifestDigest)
		if err != nil {
			return nil, err
		}
		platformSpec := manifestlist.PlatformSpec{
			Architecture: "atari2600",
			OS:           "CP/M",
			Variant:      "ternary",
			Features:     []string{"VLIW", "superscalaroutoforderdevnull"},
		}
		manifestDescriptor := manifestlist.ManifestDescriptor{
			Descriptor: descriptor,
			Platform:   platformSpec,
		}
		manifestDescriptors = append(manifestDescriptors, manifestDescriptor)
	}

	return manifestlist.FromDescriptors(manifestDescriptors)
}

// MakeSchema1Manifest constructs a schema 1 manifest from a given list of digests and returns
// the digest of the manifest
func MakeSchema1Manifest(digests []digest.Digest) (distribution.Manifest, error) {
	manifest := schema1.Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 1,
		},
		Name: "who",
		Tag:  "cares",
	}

	for _, digest := range digests {
		manifest.FSLayers = append(manifest.FSLayers, schema1.FSLayer{BlobSum: digest})
		manifest.History = append(manifest.History, schema1.History{V1Compatibility: ""})
	}

	pk, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unexpected error generating private key: %v", err)
	}

	signedManifest, err := schema1.Sign(&manifest, pk)
	if err != nil {
		return nil, fmt.Errorf("error signing manifest: %v", err)
	}

	return signedManifest, nil
}

// MakeSchema2Manifest constructs a schema 2 manifest from a given list of digests and returns
// the digest of the manifest
func MakeSchema2Manifest(repository distribution.Repository, digests []digest.Digest) (distribution.Manifest, error) {
	ctx := context.Background()
	blobStore := repository.Blobs(ctx)
	builder := schema2.NewManifestBuilder(blobStore, schema2.MediaTypeImageConfig, []byte{})
	for _, digest := range digests {
		builder.AppendReference(distribution.Descriptor{Digest: digest})
	}

	manifest, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("unexpected error generating manifest: %v", err)
	}

	return manifest, nil
}

func UploadRandomSchema1Image(repository distribution.Repository) (Image, error) {
	randomLayers, err := CreateRandomLayers(2)
	if err != nil {
		return Image{}, err
	}

	digests := []digest.Digest{}
	for digest := range randomLayers {
		digests = append(digests, digest)
	}

	manifest, err := MakeSchema1Manifest(digests)
	if err != nil {
		return Image{}, err
	}

	manifestDigest, err := UploadImage(repository, Image{manifest: manifest, Layers: randomLayers})
	if err != nil {
		return Image{}, err
	}

	return Image{
		manifest:       manifest,
		ManifestDigest: manifestDigest,
		Layers:         randomLayers,
	}, nil
}

func UploadRandomSchema2Image(repository distribution.Repository) (Image, error) {
	randomLayers, err := CreateRandomLayers(2)
	if err != nil {
		return Image{}, err
	}

	digests := []digest.Digest{}
	for digest := range randomLayers {
		digests = append(digests, digest)
	}

	manifest, err := MakeSchema2Manifest(repository, digests)
	if err != nil {
		return Image{}, err
	}

	manifestDigest, err := UploadImage(repository, Image{manifest: manifest, Layers: randomLayers})
	if err != nil {
		return Image{}, err
	}

	return Image{
		manifest:       manifest,
		ManifestDigest: manifestDigest,
		Layers:         randomLayers,
	}, nil
}

func UploadImage(repository distribution.Repository, im Image) (digest.Digest, error) {
	// upload layers
	err := UploadBlobs(repository, im.Layers)
	if err != nil {
		return "", fmt.Errorf("layer upload failed: %v", err)
	}

	// upload manifest
	ctx := context.Background()

	manifestService, err := MakeManifestService(repository)
	if err != nil {
		return "", fmt.Errorf("failed to create manifest service: %v", err)
	}

	manifestDigest, err := manifestService.Put(ctx, im.manifest)
	if err != nil {
		return "", fmt.Errorf("manifest upload failed: %v", err)
	}

	return manifestDigest, nil
}

func MakeManifestService(repository distribution.Repository) (distribution.ManifestService, error) {
	ctx := context.Background()

	manifestService, err := repository.Manifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to construct manifest store: %v", err)
	}
	return manifestService, nil
}
