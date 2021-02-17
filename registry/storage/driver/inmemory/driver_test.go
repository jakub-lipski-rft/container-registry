package inmemory

import (
	"context"
	"math/rand"
	"testing"

	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/testsuites"
	"github.com/stretchr/testify/require"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

func init() {
	inmemoryDriverConstructor := func() (storagedriver.StorageDriver, error) {
		return New(), nil
	}
	testsuites.RegisterSuite(inmemoryDriverConstructor, testsuites.NeverSkip)
}

func TestTransferTo(t *testing.T) {
	d1 := New()
	d2 := New()

	b := make([]byte, rand.Intn(20))
	rand.Read(b)

	ctx := context.Background()
	path := "/test/path"

	err := d1.PutContent(ctx, path, b)
	require.NoError(t, err)

	err = d1.TransferTo(ctx, d2, path, path)
	require.NoError(t, err)

	c, err := d2.GetContent(ctx, path)
	require.NoError(t, err)

	require.EqualValues(t, b, c)
}
