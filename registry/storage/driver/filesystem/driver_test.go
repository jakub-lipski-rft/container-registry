package filesystem

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/testsuites"
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
		params   map[string]interface{} // techincally the yaml can contain anything
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
			if err == nil {
				t.Fatalf("expected error configuring filesystem driver with invalid param: %+v", item.params)
			}
			continue
		}

		if err != nil {
			t.Fatalf("unexpected error creating filesystem driver: %s", err)
		}
		// Note that we get a pointer to params back
		if !reflect.DeepEqual(*params, item.expected) {
			t.Fatalf("unexpected params from filesystem driver. expected %+v, got %+v", item.expected, params)
		}
	}

}

// TestDeleteFilesEmptyParentDir checks that DeleteFiles removes parent directories if empty.
func TestDeleteFilesEmptyParentDir(t *testing.T) {
	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	d, err := FromParameters(map[string]interface{}{
		"rootdirectory": rootDir,
	})
	if err != nil {
		t.Fatalf("unexpected error creating driver: %v", err)
	}

	parentDir := "/testdir"
	fp := path.Join(parentDir, "testfile")
	ctx := context.Background()

	if err = d.PutContent(ctx, fp, []byte("contents")); err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}

	if _, err = d.DeleteFiles(ctx, []string{fp}); err != nil {
		t.Errorf("unexpected error deleting files: %v", err)
	}

	// check deleted file
	if _, err = d.Stat(ctx, fp); err == nil {
		t.Errorf("expected error reading deleted file, got nil")
	}
	if _, ok := err.(storagedriver.PathNotFoundError); !ok {
		t.Errorf("expected error to be of type storagedriver.PathNotFoundError, got %T", err)
	}

	// make sure the parent directory has been removed
	if _, err = d.Stat(ctx, parentDir); err == nil {
		t.Errorf("expected error reading parent directory, got nil")
	}
	if _, ok := err.(storagedriver.PathNotFoundError); !ok {
		t.Errorf("expected error to be of type storagedriver.PathNotFoundError, got %T", err)
	}
}

// TestDeleteFilesNonEmptyParentDir checks that DeleteFiles does not remove parent directories if not empty.
func TestDeleteFilesNonEmptyParentDir(t *testing.T) {
	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	d, err := FromParameters(map[string]interface{}{
		"rootdirectory": rootDir,
	})
	if err != nil {
		t.Fatalf("unexpected error creating driver: %v", err)
	}

	parentDir := "/testdir"
	fp := path.Join(parentDir, "testfile")
	ctx := context.Background()

	if err = d.PutContent(ctx, fp, []byte("contents")); err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}

	// add another test file, this one is not going to be deleted
	if err = d.PutContent(ctx, path.Join(parentDir, "testfile2"), []byte("contents")); err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}

	if _, err = d.DeleteFiles(ctx, []string{fp}); err != nil {
		t.Errorf("unexpected error deleting files: %v", err)
	}

	// check deleted file
	if _, err = d.Stat(ctx, fp); err == nil {
		t.Errorf("expected error reading deleted file, got nil")
	}
	if _, ok := err.(storagedriver.PathNotFoundError); !ok {
		t.Errorf("expected error to be of type storagedriver.PathNotFoundError, got %T", err)
	}

	// make sure the parent directory has not been removed
	if _, err = d.Stat(ctx, parentDir); err != nil {
		t.Errorf("unexpected error reading parent directory: %v", err)
	}
}

// TestDeleteFilesNonExistingParentDir checks that DeleteFiles is idempotent and doesn't return an error if a parent dir
// of a not found file doesn't exist as well.
func TestDeleteFilesNonExistingParentDir(t *testing.T) {
	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	d, err := FromParameters(map[string]interface{}{
		"rootdirectory": rootDir,
	})
	if err != nil {
		t.Fatalf("unexpected error creating driver: %v", err)
	}

	fp := path.Join("/non-existing-dir", "non-existing-file")
	count, err := d.DeleteFiles(context.Background(), []string{fp})
	if err != nil {
		t.Errorf("unexpected error deleting files: %v", err)
	}
	if count != 1 {
		t.Errorf("expected deleted count to be 1, got %d", count)
	}
}
