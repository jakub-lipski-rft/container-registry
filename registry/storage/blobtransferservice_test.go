package storage_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/testutil"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestTransferBlob(t *testing.T) {
	source := newEnv(t, "src/image-a")
	sourceBlobs := source.repo.Blobs(source.ctx)

	target := newEnv(t, "dest/image-a")
	targetBlobs := target.repo.Blobs(target.ctx)

	// Upload layer to source registry.
	srcDesc := uploadRandomLayer(t, source)

	blobDataPath := blobDataPathFromDigest(srcDesc.Digest)

	// Confirm blob is not linked to the metadata or in common blob storage
	// on target registry.
	_, err := targetBlobs.Stat(target.ctx, srcDesc.Digest)
	require.EqualError(t, err, "unknown blob")
	_, err = target.driver.Stat(target.ctx, blobDataPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), blobDataPath)

	// Transfer the blob.
	bts, err := storage.NewBlobTransferService(source.driver, target.driver)
	require.NoError(t, err)

	err = bts.Transfer(source.ctx, srcDesc.Digest)
	require.NoError(t, err)

	// Ensure blob is still not linked to the metadata, but present in common
	// blob storage on target registry.
	_, err = targetBlobs.Stat(target.ctx, srcDesc.Digest)
	require.EqualError(t, err, "unknown blob")
	_, err = target.driver.Stat(target.ctx, blobDataPath)
	require.NoError(t, err)

	// Ensure source and target blobs are the same.
	sr, err := sourceBlobs.Open(source.ctx, srcDesc.Digest)
	require.NoError(t, err)

	sourceContent, err := ioutil.ReadAll(sr)
	require.NoError(t, err)

	tr, err := target.driver.Reader(target.ctx, blobDataPath, 0)
	require.NoError(t, err)

	targetContent, err := ioutil.ReadAll(tr)
	require.NoError(t, err)

	require.EqualValues(t, sourceContent, targetContent)
}

func TestTransferBlobExistsOnTarget(t *testing.T) {
	source := newEnv(t, "src/image-a")

	target := newEnv(t, "dest/image-a")
	targetBlobs := target.repo.Blobs(target.ctx)

	// Put layer directly into target registry's common blob storage.
	layer, dgst, err := testutil.CreateRandomTarFile()
	require.NoError(t, err)

	blobDataPath := blobDataPathFromDigest(dgst)

	var p []byte
	_, err = io.ReadFull(layer, p)
	require.NoError(t, err)

	err = target.driver.PutContent(target.ctx, blobDataPath, p)
	require.NoError(t, err)

	// Transfer blob.
	bts, err := storage.NewBlobTransferService(source.driver, target.driver)
	require.NoError(t, err)

	err = bts.Transfer(source.ctx, dgst)
	require.NoError(t, err)

	// Ensure blob is still not linked to the metadata, but present in common
	// blob storage on target registry.
	_, err = targetBlobs.Stat(target.ctx, dgst)
	require.EqualError(t, err, "unknown blob")
	_, err = target.driver.Stat(target.ctx, blobDataPath)
	require.NoError(t, err)
}

func blobDataPathFromDigest(dgst digest.Digest) string {
	return filepath.Join(
		"/docker/registry/v2/blobs/",
		dgst.Algorithm().String(),
		dgst.Hex()[0:2],
		dgst.Hex(),
		"data",
	)
}

type failDriver struct {
	driver.StorageDriver
}

func (d *failDriver) Writer(ctx context.Context, path string, append bool) (driver.FileWriter, error) {
	return nil, errors.New("failed to create writer")
}

func TestTransferBlobMissingBlob(t *testing.T) {
	source := newEnv(t, "src/image-a")
	target := newEnv(t, "dest/image-a")

	dgst := digest.FromString("fake-digest")

	// Transfer the blob, failing on stat before transfer begins.
	bts, err := storage.NewBlobTransferService(source.driver, target.driver)
	require.NoError(t, err)

	// Check that a ErrBlobTransferFailed for the correct blob was returned.
	err = bts.Transfer(source.ctx, dgst)
	e := &distribution.ErrBlobTransferFailed{}
	require.True(t, errors.As(err, e))
	require.Equal(t, dgst, e.Digest)

	// Ensure that did not attempt to clean up after an non partial transfer.
	require.False(t, e.Cleanup)
	require.Nil(t, e.CleanupErr)
	require.False(t, errors.As(e.Reason, &driver.PartialTransferError{}))
}

func TestTransferBlobPartialTransferError(t *testing.T) {
	source := newEnv(t, "src/image-a")
	target := newEnv(t, "dest/image-a")

	// Upload layer to source registry.
	srcDesc := uploadRandomLayer(t, source)

	// Transfer the blob, failing on write.
	bts, err := storage.NewBlobTransferService(source.driver, &failDriver{target.driver})
	require.NoError(t, err)

	// Check that a ErrBlobTransferFailed for the correct blob was returned.
	err = bts.Transfer(source.ctx, srcDesc.Digest)
	e := &distribution.ErrBlobTransferFailed{}
	require.True(t, errors.As(err, e))
	require.Equal(t, srcDesc.Digest, e.Digest)

	// Ensure that we attempted to clean up after a partial transfer.
	require.True(t, e.Cleanup)
	require.Nil(t, e.CleanupErr)
	require.True(t, errors.As(e.Reason, &driver.PartialTransferError{}))
}

func uploadLayer(t *testing.T, e *env, layer io.ReadSeeker, dgst digest.Digest) distribution.Descriptor {
	t.Helper()

	bs := e.repo.Blobs(e.ctx)

	wr, err := bs.Create(e.ctx)
	require.NoError(t, err)

	_, err = io.Copy(wr, layer)
	require.NoError(t, err)

	desc, err := wr.Commit(e.ctx, distribution.Descriptor{Digest: dgst})
	require.NoError(t, err)

	return desc
}

func uploadRandomLayer(t *testing.T, e *env) distribution.Descriptor {
	t.Helper()

	layer, dgst, err := testutil.CreateRandomTarFile()
	require.NoError(t, err)

	return uploadLayer(t, e, layer, dgst)
}
