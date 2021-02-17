package filesystem

import (
	"context"
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/testsuites"
	"github.com/stretchr/testify/require"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	root, err := ioutil.TempDir("", "driver-")
	if err != nil {
		panic(err)
	}
	defer os.Remove(root)

	driver, err := FromParameters(map[string]interface{}{
		"rootdirectory": root,
	})
	if err != nil {
		panic(err)
	}

	testsuites.RegisterSuite(func() (storagedriver.StorageDriver, error) {
		return driver, nil
	}, testsuites.NeverSkip)
}

func TestFromParametersImpl(t *testing.T) {
	tests := []struct {
		params   map[string]interface{} // technically the yaml can contain anything
		expected DriverParameters
		pass     bool
	}{
		// check we use default threads and root dirs
		{
			params: map[string]interface{}{},
			expected: DriverParameters{
				RootDirectory: defaultRootDirectory,
				MaxThreads:    defaultMaxThreads,
			},
			pass: true,
		},
		// Testing initiation with a string maxThreads which can't be parsed
		{
			params: map[string]interface{}{
				"maxthreads": "fail",
			},
			expected: DriverParameters{},
			pass:     false,
		},
		{
			params: map[string]interface{}{
				"maxthreads": "100",
			},
			expected: DriverParameters{
				RootDirectory: defaultRootDirectory,
				MaxThreads:    uint64(100),
			},
			pass: true,
		},
		{
			params: map[string]interface{}{
				"maxthreads": 100,
			},
			expected: DriverParameters{
				RootDirectory: defaultRootDirectory,
				MaxThreads:    uint64(100),
			},
			pass: true,
		},
		// check that we use minimum thread counts
		{
			params: map[string]interface{}{
				"maxthreads": 1,
			},
			expected: DriverParameters{
				RootDirectory: defaultRootDirectory,
				MaxThreads:    minThreads,
			},
			pass: true,
		},
	}

	for _, item := range tests {
		params, err := fromParametersImpl(item.params)

		if !item.pass {
			// We only need to assert that expected failures have an error
			require.Error(t, err)
			continue
		}

		require.NoError(t, err)

		// Note that we get a pointer to params back
		require.Equal(t, item.expected, *params)
	}
}

// TestDeleteFilesEmptyParentDir checks that DeleteFiles removes parent directories if empty.
func TestDeleteFilesEmptyParentDir(t *testing.T) {
	d, cleanup := newTempDirDriver(t)
	defer cleanup()

	parentDir := "/testdir"
	fp := path.Join(parentDir, "testfile")
	ctx := context.Background()

	err := d.PutContent(ctx, fp, []byte("contents"))
	require.NoError(t, err)

	_, err = d.DeleteFiles(ctx, []string{fp})
	require.NoError(t, err)

	// check deleted file
	_, err = d.Stat(ctx, fp)
	require.True(t, errors.As(err, &storagedriver.PathNotFoundError{}))

	// make sure the parent directory has been removed
	_, err = d.Stat(ctx, parentDir)
	require.True(t, errors.As(err, &storagedriver.PathNotFoundError{}))
}

// TestDeleteFilesNonEmptyParentDir checks that DeleteFiles does not remove parent directories if not empty.
func TestDeleteFilesNonEmptyParentDir(t *testing.T) {
	d, cleanup := newTempDirDriver(t)
	defer cleanup()

	parentDir := "/testdir"
	fp := path.Join(parentDir, "testfile")
	ctx := context.Background()

	err := d.PutContent(ctx, fp, []byte("contents"))
	require.NoError(t, err)

	// add another test file, this one is not going to be deleted
	err = d.PutContent(ctx, path.Join(parentDir, "testfile2"), []byte("contents"))
	require.NoError(t, err)

	_, err = d.DeleteFiles(ctx, []string{fp})
	require.NoError(t, err)

	// check deleted file
	_, err = d.Stat(ctx, fp)
	require.True(t, errors.As(err, &storagedriver.PathNotFoundError{}))

	// make sure the parent directory has not been removed
	_, err = d.Stat(ctx, parentDir)
	require.NoError(t, err)
}

// TestDeleteFilesNonExistingParentDir checks that DeleteFiles is idempotent and doesn't return an error if a parent dir
// of a not found file doesn't exist as well.
func TestDeleteFilesNonExistingParentDir(t *testing.T) {
	d, cleanup := newTempDirDriver(t)
	defer cleanup()

	fp := path.Join("/non-existing-dir", "non-existing-file")
	count, err := d.DeleteFiles(context.Background(), []string{fp})
	if err != nil {
		t.Errorf("unexpected error deleting files: %v", err)
	}
	if count != 1 {
		t.Errorf("expected deleted count to be 1, got %d", count)
	}
}

func TestTransferTo(t *testing.T) {
	srcDriver, cleanup := newTempDirDriver(t)
	defer cleanup()

	destDriver, cleanup := newTempDirDriver(t)
	defer cleanup()

	b := make([]byte, 10)
	rand.Read(b)

	ctx := context.Background()
	path := "/happy/data/path"

	// Write content to source.
	err := srcDriver.PutContent(ctx, path, b)
	require.NoError(t, err)

	// Destination should not have already have content at the path.
	_, err = destDriver.Stat(ctx, path)
	require.True(t, errors.As(err, &storagedriver.PathNotFoundError{}))

	// Transfer to destination.
	err = srcDriver.TransferTo(ctx, destDriver, path, path)
	require.NoError(t, err)

	// Reading from destination should work.
	c, err := destDriver.GetContent(ctx, path)
	require.NoError(t, err)
	require.EqualValues(t, b, c)
}

func TestTransferToExistingDest(t *testing.T) {
	srcDriver, cleanup := newTempDirDriver(t)
	defer cleanup()

	destDriver, cleanup := newTempDirDriver(t)
	defer cleanup()

	b := make([]byte, 10)
	rand.Read(b)

	ctx := context.Background()
	path := "/existing/data/path"

	// Write content only at dest.
	err := destDriver.PutContent(ctx, path, b)
	require.NoError(t, err)

	// Transfer should stat for content on destintation side and return without error.
	err = srcDriver.TransferTo(ctx, destDriver, path, path)
	require.NoError(t, err)

	// Getting content from destination should match the original.
	c, err := destDriver.GetContent(ctx, path)
	require.NoError(t, err)
	require.EqualValues(t, b, c)

	// Source should not have been modified.
	_, err = srcDriver.Stat(ctx, path)
	require.True(t, errors.As(err, &storagedriver.PathNotFoundError{}))
}

func TestTransferToSameRootDir(t *testing.T) {
	rootDir, err := ioutil.TempDir("", "driver-")
	require.NoError(t, err)
	defer os.Remove(rootDir)

	srcDriver, err := FromParameters(map[string]interface{}{
		"rootdirectory": rootDir,
	})
	require.NoError(t, err)

	destDriver, err := FromParameters(map[string]interface{}{
		"rootdirectory": rootDir,
	})
	require.NoError(t, err)

	b := make([]byte, 10)
	rand.Read(b)

	ctx := context.Background()
	path := "/same/root/path"

	// Write content to source.
	err = srcDriver.PutContent(ctx, path, b)
	require.NoError(t, err)

	// Transfer to destination should exit early with error.
	err = srcDriver.TransferTo(ctx, destDriver, path, path)
	require.EqualError(t, err,
		"unable to begin transfer: srcDriver and destDriver must not have the same root directory")
}

func newTempDirDriver(t *testing.T) (*Driver, func()) {
	t.Helper()

	rootDir, err := ioutil.TempDir("", "driver-")
	require.NoError(t, err)

	d, err := FromParameters(map[string]interface{}{
		"rootdirectory": rootDir,
	})
	require.NoError(t, err)

	return d, func() { os.Remove(rootDir) }
}
