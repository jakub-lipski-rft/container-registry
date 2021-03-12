package azure

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/testsuites"
	"github.com/stretchr/testify/require"
	. "gopkg.in/check.v1"
)

const (
	envAccountName = "AZURE_STORAGE_ACCOUNT_NAME"
	envAccountKey  = "AZURE_STORAGE_ACCOUNT_KEY"
	envContainer   = "AZURE_STORAGE_CONTAINER"
	envRealm       = "AZURE_STORAGE_REALM"
)

var (
	accountName string
	accountKey  string
	container   string
	realm       string
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	config := []struct {
		env   string
		value *string
	}{
		{envAccountName, &accountName},
		{envAccountKey, &accountKey},
		{envContainer, &container},
		{envRealm, &realm},
	}

	missing := []string{}
	for _, v := range config {
		*v.value = os.Getenv(v.env)
		if *v.value == "" {
			missing = append(missing, v.env)
		}
	}

	root, err := ioutil.TempDir("", "driver-")
	if err != nil {
		panic(err)
	}
	defer os.Remove(root)

	azureDriverConstructor := func() (storagedriver.StorageDriver, error) {
		return New(accountName, accountKey, container, realm, root, false)
	}

	// Skip Azure storage driver tests if environment variable parameters are not provided
	skipCheck := func() string {
		if len(missing) > 0 {
			return fmt.Sprintf("Must set %s environment variables to run Azure tests", strings.Join(missing, ", "))
		}
		return ""
	}

	testsuites.RegisterSuite(azureDriverConstructor, skipCheck)
}

func TestPathToKey(t *testing.T) {
	var tests = []struct {
		name          string
		rootDirectory string
		providedPath  string
		expectedPath  string
		legacyPath    bool
	}{
		{
			name:          "legacy leading slash empty root directory",
			rootDirectory: "",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "/docker/registry/v2",
			legacyPath:    true,
		},
		{
			name:          "legacy leading slash single slash root directory",
			rootDirectory: "/",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "/docker/registry/v2",
			legacyPath:    true,
		},
		{
			name:          "empty root directory results in expected path",
			rootDirectory: "",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "docker/registry/v2",
		},
		{
			name:          "legacy empty root directory results in expected path",
			rootDirectory: "",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "/docker/registry/v2",
			legacyPath:    true,
		},
		{
			name:          "root directory no slashes prefixed to path with slash between root and path",
			rootDirectory: "opt",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "opt/docker/registry/v2",
		},
		{
			name:          "legacy root directory no slashes prefixed to path with slash between root and path",
			rootDirectory: "opt",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "/opt/docker/registry/v2",
			legacyPath:    true,
		},
		{
			name:          "root directory with slashes prefixed to path no leading slash",
			rootDirectory: "/opt/",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "opt/docker/registry/v2",
		},
		{
			name:          "dirty root directory prefixed to path cleanly",
			rootDirectory: "/opt////",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "opt/docker/registry/v2",
		},
		{
			name:          "nested custom root directory prefixed to path",
			rootDirectory: "a/b/c/d/",
			providedPath:  "/docker/registry/v2/",
			expectedPath:  "a/b/c/d/docker/registry/v2",
		},
		{
			name:          "legacy root directory results in expected root path",
			rootDirectory: "",
			providedPath:  "/",
			expectedPath:  "/",
			legacyPath:    true,
		},
		{
			name:          "root directory results in expected root path",
			rootDirectory: "",
			providedPath:  "/",
			expectedPath:  "",
		},
		{
			name:          "legacy root directory no slashes results in expected root path",
			rootDirectory: "opt",
			providedPath:  "/",
			expectedPath:  "/opt",
			legacyPath:    true,
		},
		{
			name:          "root directory no slashes results in expected root path",
			rootDirectory: "opt",
			providedPath:  "/",
			expectedPath:  "opt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDirectory := strings.Trim(tt.rootDirectory, "/")
			if rootDirectory != "" {
				rootDirectory += "/"
			}
			d := &driver{rootDirectory: rootDirectory, legacyPath: tt.legacyPath}
			require.Equal(t, tt.expectedPath, d.pathToKey(tt.providedPath))
		})
	}
}

func TestStatRootPath(t *testing.T) {
	var tests = []struct {
		name          string
		rootDirectory string
		legacyPath    bool
	}{
		{
			name:          "legacy empty root directory",
			rootDirectory: "",
			legacyPath:    true,
		},
		{
			name:          "empty root directory",
			rootDirectory: "",
		},
		{
			name:          "legacy slash root directory",
			rootDirectory: "/",
			legacyPath:    true,
		},
		{
			name:          "slash root directory",
			rootDirectory: "/",
		},
		{
			name:          "root directory no slashes",
			rootDirectory: "opt",
		},
		{
			name:          "legacy root directory no slashes",
			rootDirectory: "opt",
			legacyPath:    true,
		},
		{
			name:          "nested custom root directory",
			rootDirectory: "a/b/c/d/",
		},
		{
			name:          "legacy nested custom root directory",
			rootDirectory: "a/b/c/d/",
			legacyPath:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := New(accountName, accountKey, container, realm, tt.rootDirectory, tt.legacyPath)
			require.NoError(t, err)

			// Health checks stat "/" and expect either a not found error or a directory.
			fsInfo, err := d.Stat(context.Background(), "/")
			if !errors.As(err, &storagedriver.PathNotFoundError{}) {
				require.True(t, fsInfo.IsDir())
			}
		})
	}
}
