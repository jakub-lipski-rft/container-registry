package s3

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"

	"gopkg.in/check.v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/docker/distribution/context"
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/testsuites"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

var s3DriverConstructor func(rootDirectory, storageClass string) (*Driver, error)
var skipS3 func() string

func init() {
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")
	encrypt := os.Getenv("S3_ENCRYPT")
	keyID := os.Getenv("S3_KEY_ID")
	secure := os.Getenv("S3_SECURE")
	skipVerify := os.Getenv("S3_SKIP_VERIFY")
	v4Auth := os.Getenv("S3_V4_AUTH")
	region := os.Getenv("AWS_REGION")
	objectACL := os.Getenv("S3_OBJECT_ACL")
	root, err := ioutil.TempDir("", "driver-")
	regionEndpoint := os.Getenv("REGION_ENDPOINT")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")
	pathStyle := os.Getenv("AWS_PATH_STYLE")
	maxRequestsPerSecond := os.Getenv("S3_MAX_REQUESTS_PER_SEC")
	logLevel := os.Getenv("S3_LOG_LEVEL")

	if err != nil {
		panic(err)
	}
	defer os.Remove(root)

	s3DriverConstructor = func(rootDirectory, storageClass string) (*Driver, error) {
		encryptBool := false
		if encrypt != "" {
			encryptBool, err = strconv.ParseBool(encrypt)
			if err != nil {
				return nil, err
			}
		}

		secureBool := true
		if secure != "" {
			secureBool, err = strconv.ParseBool(secure)
			if err != nil {
				return nil, err
			}
		}

		skipVerifyBool := false
		if skipVerify != "" {
			skipVerifyBool, err = strconv.ParseBool(skipVerify)
			if err != nil {
				return nil, err
			}
		}

		v4Bool := true
		if v4Auth != "" {
			v4Bool, err = strconv.ParseBool(v4Auth)
			if err != nil {
				return nil, err
			}
		}

		pathStyleBool := false

		// If regionEndpoint is set, default to forcing pathstyle to preserve legacy behavior.
		if regionEndpoint != "" {
			pathStyleBool = true
		}

		if pathStyle != "" {
			pathStyleBool, err = strconv.ParseBool(pathStyle)
			if err != nil {
				return nil, err
			}
		}

		maxRequestsPerSecondInt64 := int64(defaultMaxRequestsPerSecond)

		if maxRequestsPerSecond != "" {
			if maxRequestsPerSecondInt64, err = strconv.ParseInt(maxRequestsPerSecond, 10, 64); err != nil {
				return nil, err
			}
		}

		parallelWalkBool := true

		logLevelType := parseLogLevelParam(logLevel)

		parameters := DriverParameters{
			accessKey,
			secretKey,
			bucket,
			region,
			regionEndpoint,
			encryptBool,
			keyID,
			secureBool,
			skipVerifyBool,
			v4Bool,
			minChunkSize,
			defaultMultipartCopyChunkSize,
			defaultMultipartCopyMaxConcurrency,
			defaultMultipartCopyThresholdSize,
			rootDirectory,
			storageClass,
			driverName + "-test",
			objectACL,
			sessionToken,
			pathStyleBool,
			maxRequestsPerSecondInt64,
			parallelWalkBool,
			logLevelType,
		}

		return New(parameters)
	}

	// Skip S3 storage driver tests if environment variable parameters are not provided
	skipS3 = func() string {
		if accessKey == "" || secretKey == "" || region == "" || bucket == "" || encrypt == "" {
			return "Must set AWS_ACCESS_KEY, AWS_SECRET_KEY, AWS_REGION, S3_BUCKET, and S3_ENCRYPT to run S3 tests"
		}
		return ""
	}

	testsuites.RegisterSuite(func() (storagedriver.StorageDriver, error) {
		return s3DriverConstructor(root, s3.StorageClassStandard)
	}, skipS3)
}

func TestFromParameters(t *testing.T) {
	// Minimal params needed to construct the driver.
	baseParams := map[string]interface{}{
		"region": "us-west-2",
		"bucket": "test",
		"v4auth": "true",
	}

	tests := []struct {
		params              map[string]interface{}
		wantedForcePathBool bool
	}{
		{
			map[string]interface{}{
				"pathstyle": "false",
			}, false,
		},
		{
			map[string]interface{}{
				"pathstyle": "true",
			}, true,
		},
		{
			map[string]interface{}{
				"regionendpoint": "test-endpoint",
				"pathstyle":      "false",
			}, false,
		},
		{
			map[string]interface{}{
				"regionendpoint": "test-endpoint",
				"pathstyle":      "true",
			}, true,
		},
		{
			map[string]interface{}{
				"regionendpoint": "",
			}, false,
		},
		{
			map[string]interface{}{
				"regionendpoint": "test-endpoint",
			}, true,
		},
	}

	for _, tt := range tests {

		// add baseParams to testing params
		for k, v := range baseParams {
			tt.params[k] = v
		}

		d, err := FromParameters(tt.params)
		if err != nil {
			t.Fatalf("unable to create a new S3 driver: %v", err)
		}

		pathStyle := d.baseEmbed.Base.StorageDriver.(*driver).S3.(*s3.S3).Client.Config.S3ForcePathStyle
		if pathStyle == nil {
			t.Fatal("expected pathStyle not to be nil")
		}

		if *pathStyle != tt.wantedForcePathBool {
			t.Fatalf("expected S3ForcePathStyle to be %v, got %v, with params %#v", tt.wantedForcePathBool, *pathStyle, tt.params)
		}
	}
}

func TestEmptyRootList(t *testing.T) {
	if skipS3() != "" {
		t.Skip(skipS3())
	}

	validRoot, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(validRoot)

	rootedDriver, err := s3DriverConstructor(validRoot, s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating rooted driver: %v", err)
	}

	emptyRootDriver, err := s3DriverConstructor("", s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating empty root driver: %v", err)
	}

	slashRootDriver, err := s3DriverConstructor("/", s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating slash root driver: %v", err)
	}

	filename := "/test"
	contents := []byte("contents")
	ctx := context.Background()
	err = rootedDriver.PutContent(ctx, filename, contents)
	if err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}
	defer rootedDriver.Delete(ctx, filename)

	keys, _ := emptyRootDriver.List(ctx, "/")
	for _, path := range keys {
		if !storagedriver.PathRegexp.MatchString(path) {
			t.Fatalf("unexpected string in path: %q != %q", path, storagedriver.PathRegexp)
		}
	}

	keys, _ = slashRootDriver.List(ctx, "/")
	for _, path := range keys {
		if !storagedriver.PathRegexp.MatchString(path) {
			t.Fatalf("unexpected string in path: %q != %q", path, storagedriver.PathRegexp)
		}
	}
}

func TestStorageClass(t *testing.T) {
	if skipS3() != "" {
		t.Skip(skipS3())
	}

	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	standardDriver, err := s3DriverConstructor(rootDir, s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating driver with standard storage: %v", err)
	}

	rrDriver, err := s3DriverConstructor(rootDir, s3.StorageClassReducedRedundancy)
	if err != nil {
		t.Fatalf("unexpected error creating driver with reduced redundancy storage: %v", err)
	}

	if _, err = s3DriverConstructor(rootDir, noStorageClass); err != nil {
		t.Fatalf("unexpected error creating driver without storage class: %v", err)
	}

	standardFilename := "/test-standard"
	rrFilename := "/test-rr"
	contents := []byte("contents")
	ctx := context.Background()

	err = standardDriver.PutContent(ctx, standardFilename, contents)
	if err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}
	defer standardDriver.Delete(ctx, standardFilename)

	err = rrDriver.PutContent(ctx, rrFilename, contents)
	if err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}
	defer rrDriver.Delete(ctx, rrFilename)

	standardDriverUnwrapped := standardDriver.Base.StorageDriver.(*driver)
	resp, err := standardDriverUnwrapped.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(standardDriverUnwrapped.Bucket),
		Key:    aws.String(standardDriverUnwrapped.s3Path(standardFilename)),
	})
	if err != nil {
		t.Fatalf("unexpected error retrieving standard storage file: %v", err)
	}
	defer resp.Body.Close()
	// Amazon only populates this header value for non-standard storage classes
	if resp.StorageClass != nil {
		t.Fatalf("unexpected storage class for standard file: %v", resp.StorageClass)
	}

	rrDriverUnwrapped := rrDriver.Base.StorageDriver.(*driver)
	resp, err = rrDriverUnwrapped.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(rrDriverUnwrapped.Bucket),
		Key:    aws.String(rrDriverUnwrapped.s3Path(rrFilename)),
	})
	if err != nil {
		t.Fatalf("unexpected error retrieving reduced-redundancy storage file: %v", err)
	}
	defer resp.Body.Close()
	if resp.StorageClass == nil {
		t.Fatalf("unexpected storage class for reduced-redundancy file: %v", s3.StorageClassStandard)
	} else if *resp.StorageClass != s3.StorageClassReducedRedundancy {
		t.Fatalf("unexpected storage class for reduced-redundancy file: %v", *resp.StorageClass)
	}

}

func TestOverThousandBlobs(t *testing.T) {
	if skipS3() != "" {
		t.Skip(skipS3())
	}

	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	standardDriver, err := s3DriverConstructor(rootDir, s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating driver with standard storage: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 1005; i++ {
		filename := "/thousandfiletest/file" + strconv.Itoa(i)
		contents := []byte("contents")
		err = standardDriver.PutContent(ctx, filename, contents)
		if err != nil {
			t.Fatalf("unexpected error creating content: %v", err)
		}
	}

	// cant actually verify deletion because read-after-delete is inconsistent, but can ensure no errors
	err = standardDriver.Delete(ctx, "/thousandfiletest")
	if err != nil {
		t.Fatalf("unexpected error deleting thousand files: %v", err)
	}
}

func TestMoveWithMultipartCopy(t *testing.T) {
	if skipS3() != "" {
		t.Skip(skipS3())
	}

	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	d, err := s3DriverConstructor(rootDir, s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating driver: %v", err)
	}

	ctx := context.Background()
	sourcePath := "/source"
	destPath := "/dest"

	defer d.Delete(ctx, sourcePath)
	defer d.Delete(ctx, destPath)

	// An object larger than d's MultipartCopyThresholdSize will cause d.Move() to perform a multipart copy.
	multipartCopyThresholdSize := d.baseEmbed.Base.StorageDriver.(*driver).MultipartCopyThresholdSize
	contents := make([]byte, 2*multipartCopyThresholdSize)
	rand.Read(contents)

	err = d.PutContent(ctx, sourcePath, contents)
	if err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}

	err = d.Move(ctx, sourcePath, destPath)
	if err != nil {
		t.Fatalf("unexpected error moving file: %v", err)
	}

	received, err := d.GetContent(ctx, destPath)
	if err != nil {
		t.Fatalf("unexpected error getting content: %v", err)
	}
	if !bytes.Equal(contents, received) {
		t.Fatal("content differs")
	}

	_, err = d.GetContent(ctx, sourcePath)
	switch err.(type) {
	case storagedriver.PathNotFoundError:
	default:
		t.Fatalf("unexpected error getting content: %v", err)
	}
}

type mockDeleteObjectsError struct {
	s3iface.S3API
}

// DeleteObjects mocks a serialization error while processing a DeleteObjects response.
func (m *mockDeleteObjectsError) DeleteObjects(input *s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	return nil, awserr.New(request.ErrCodeSerialization, "failed reading response body", nil)
}

func testDeleteFilesError(t *testing.T, mock s3iface.S3API, numFiles int) (int, error) {
	if skipS3() != "" {
		t.Skip(skipS3())
	}

	rootDir, err := ioutil.TempDir("", "driver-")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %v", err)
	}
	defer os.Remove(rootDir)

	d, err := s3DriverConstructor(rootDir, s3.StorageClassStandard)
	if err != nil {
		t.Fatalf("unexpected error creating driver: %v", err)
	}

	// mock the underlying S3 client
	d.baseEmbed.Base.StorageDriver.(*driver).S3 = mock

	// simulate deleting numFiles files
	paths := make([]string, 0, numFiles)
	for i := 0; i < numFiles; i++ {
		paths = append(paths, string(rand.Int()))
	}

	return d.DeleteFiles(context.Background(), paths)
}

// TestDeleteFilesError checks that DeleteFiles handles network/service errors correctly.
func TestDeleteFilesError(t *testing.T) {
	// simulate deleting 2*deleteMax files (should spawn 2 goroutines)
	count, err := testDeleteFilesError(t, &mockDeleteObjectsError{}, 2*deleteMax)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if count != 0 {
		t.Errorf("expected the deleted files count to be 0, got %d", count)
	}

	errs, ok := err.(storagedriver.MultiError)
	if !ok {
		t.Errorf("expected error to be of type storagedriver.MultiError, got %T", err)
	}
	if len(errs) != 2 {
		t.Errorf("expected the number of spawned goroutines to be 2, got %d", len(errs))
	}

	expected := awserr.New(request.ErrCodeSerialization, "failed reading response body", nil).Error()
	for _, e := range errs {
		if e.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, e)
		}
	}
}

type mockDeleteObjectsPartialError struct {
	s3iface.S3API
}

// DeleteObjects mocks an S3 DeleteObjects partial error. Half of the objects are successfully deleted, and the other
// half fails due to an 'AccessDenied' error.
func (m *mockDeleteObjectsPartialError) DeleteObjects(input *s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	var deleted []*s3.DeletedObject
	var errored []*s3.Error
	errCode := "AccessDenied"
	errMsg := "Access Denied"

	for i, o := range input.Delete.Objects {
		if i%2 == 0 {
			// error
			errored = append(errored, &s3.Error{
				Key:     o.Key,
				Code:    &errCode,
				Message: &errMsg,
			})
		} else {
			// success
			deleted = append(deleted, &s3.DeletedObject{Key: o.Key})
		}
	}

	return &s3.DeleteObjectsOutput{Deleted: deleted, Errors: errored}, nil
}

// TestDeleteFilesPartialError checks that DeleteFiles handles partial deletion errors correctly.
func TestDeleteFilesPartialError(t *testing.T) {
	// simulate deleting 2*deleteMax files (should spawn 2 goroutines)
	n := 2 * deleteMax
	half := n / 2
	count, err := testDeleteFilesError(t, &mockDeleteObjectsPartialError{}, n)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if count != half {
		t.Errorf("expected the deleted files count to be %d, got %d", half, count)
	}

	errs, ok := err.(storagedriver.MultiError)
	if !ok {
		t.Errorf("expected error to be of type storagedriver.MultiError, got %T", err)
	}
	if len(errs) != half {
		t.Errorf("expected the number of errors to be %d, got %d", half, len(errs))
	}

	p := `failed to delete file '.*': 'Access Denied'`
	for _, e := range errs {
		matched, err := regexp.MatchString(p, e.Error())
		if err != nil {
			t.Errorf("unexpected error matching pattern: %v", err)
		}
		if !matched {
			t.Errorf("expected error %q to match %q", e, p)
		}
	}
}
