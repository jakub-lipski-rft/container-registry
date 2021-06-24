// Package s3 provides a storagedriver.StorageDriver implementation to
// store blobs in Amazon S3 cloud storage.
//
// This package leverages the official aws client library for interfacing with
// S3.
//
// Because S3 is a key, value store the Stat call does not support last modification
// time for directories (directories are an abstraction for key, value stores)
//
// Keep in mind that S3 guarantees only read-after-write consistency for new
// objects, but no read-after-update or list-after-write consistency.
package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
    "os"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
    "github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	dcontext "github.com/docker/distribution/context"
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/base"
	"github.com/docker/distribution/registry/storage/driver/factory"
	"github.com/docker/distribution/version"
)

const driverName = "s3aws"

// minChunkSize defines the minimum multipart upload chunk size
// S3 API requires multipart upload chunks to be at least 5MB
const minChunkSize = 5 << 20

// maxChunkSize defines the maximum multipart upload chunk size allowed by S3.
const maxChunkSize = 5 << 30

const defaultChunkSize = 2 * minChunkSize

const (
	// defaultMultipartCopyChunkSize defines the default chunk size for all
	// but the last Upload Part - Copy operation of a multipart copy.
	// Empirically, 32 MB is optimal.
	defaultMultipartCopyChunkSize = 32 << 20

	// defaultMultipartCopyMaxConcurrency defines the default maximum number
	// of concurrent Upload Part - Copy operations for a multipart copy.
	defaultMultipartCopyMaxConcurrency = 100

	// defaultMultipartCopyThresholdSize defines the default object size
	// above which multipart copy will be used. (PUT Object - Copy is used
	// for objects at or below this size.)  Empirically, 32 MB is optimal.
	defaultMultipartCopyThresholdSize = 32 << 20
)

// listMax is the largest amount of objects you can request from S3 in a list call
const listMax = 1000

// deleteMax is the largest amount of objects you can request to be deleted in S3 using a DeleteObjects call. This is
// currently set to 1000 as per the S3 specification https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html
const deleteMax = 1000

// defaultMaxRequestsPerSecond defines the default maximum number of requests
// per second that can be made to the S3 API per driver instance. 350 is 10%
// of the requestsPerSecondUpperLimit based on the figures listed in
// https://docs.aws.amazon.com/AmazonS3/latest/dev/optimizing-performance.html
const defaultMaxRequestsPerSecond = 350

// defaultBurst is how many limiter tokens may be reserved at once. Currently,
// we only reserve one at a time via Limiter.Wait()
const defaultBurst = 1

// noStorageClass defines the value to be used if storage class is not supported by the S3 endpoint
const noStorageClass = "NONE"

// defaults related to exponential backoff
const (
	// defaultMaxRetries is how many times the driver will retry failed requests.
	defaultMaxRetries          = 5
	defaultInitialInterval     = backoff.DefaultInitialInterval
	defaultRandomizationFactor = backoff.DefaultRandomizationFactor
	defaultMultiplier          = backoff.DefaultMultiplier
	defaultMaxInterval         = backoff.DefaultMaxInterval
	defaultMaxElapsedTime      = backoff.DefaultMaxElapsedTime
)

// validRegions maps known s3 region identifiers to region descriptors
var validRegions = map[string]struct{}{}

// validObjectACLs contains known s3 object Acls
var validObjectACLs = map[string]struct{}{}

//DriverParameters A struct that encapsulates all of the driver parameters after all values have been set
type DriverParameters struct {
	AccessKey                   string
	SecretKey                   string
	Bucket                      string
	Region                      string
	RegionEndpoint              string
	Encrypt                     bool
	KeyID                       string
	Secure                      bool
	SkipVerify                  bool
	V4Auth                      bool
	ChunkSize                   int64
	MultipartCopyChunkSize      int64
	MultipartCopyMaxConcurrency int64
	MultipartCopyThresholdSize  int64
	RootDirectory               string
	StorageClass                string
	ObjectACL                   string
	SessionToken                string
	PathStyle                   bool
	MaxRequestsPerSecond        int64
	MaxRetries                  int64
	ParallelWalk                bool
	LogLevel                    aws.LogLevelType
}

func init() {
	partitions := endpoints.DefaultPartitions()
	for _, p := range partitions {
		for region := range p.Regions() {
			validRegions[region] = struct{}{}
		}
	}

	for _, objectACL := range []string{
		s3.ObjectCannedACLPrivate,
		s3.ObjectCannedACLPublicRead,
		s3.ObjectCannedACLPublicReadWrite,
		s3.ObjectCannedACLAuthenticatedRead,
		s3.ObjectCannedACLAwsExecRead,
		s3.ObjectCannedACLBucketOwnerRead,
		s3.ObjectCannedACLBucketOwnerFullControl,
	} {
		validObjectACLs[objectACL] = struct{}{}
	}

	// Register this as the default s3 driver in addition to s3aws
	factory.Register("s3", &s3DriverFactory{})
	factory.Register(driverName, &s3DriverFactory{})
}

// s3DriverFactory implements the factory.StorageDriverFactory interface
type s3DriverFactory struct{}

func (factory *s3DriverFactory) Create(parameters map[string]interface{}) (storagedriver.StorageDriver, error) {
	return FromParameters(parameters)
}

type driver struct {
	S3                          *s3wrapper
	Bucket                      string
	ChunkSize                   int64
	Encrypt                     bool
	KeyID                       string
	MultipartCopyChunkSize      int64
	MultipartCopyMaxConcurrency int64
	MultipartCopyThresholdSize  int64
	RootDirectory               string
	StorageClass                string
	ObjectACL                   string
	ParallelWalk                bool
}

type baseEmbed struct {
	base.Base
}

// Driver is a storagedriver.StorageDriver implementation backed by Amazon S3
// Objects are stored at absolute keys in the provided bucket.
type Driver struct {
	baseEmbed
}

func parseLogLevelParam(param interface{}) aws.LogLevelType {
	logLevel := aws.LogOff

	if param != nil {
		switch strings.ToLower(param.(string)) {
		case "logoff":
			log.Info("S3 logging level set to LogOff")
		case "logdebug":
			log.Info("S3 logging level set to LogDebug")
			logLevel = aws.LogDebug
		case "logdebugwithsigning":
			log.Info("S3 logging level set to LogDebugWithSigning")
			logLevel = aws.LogDebugWithSigning
		case "logdebugwithhttpbody":
			log.Info("S3 logging level set to LogDebugWithHTTPBody")
			logLevel = aws.LogDebugWithHTTPBody
		case "logdebugwithrequestretries":
			log.Info("S3 logging level set to LogDebugWithRequestRetries")
			logLevel = aws.LogDebugWithRequestRetries
		case "logdebugwithrequesterrors":
			log.Info("S3 logging level set to LogDebugWithRequestErrors")
			logLevel = aws.LogDebugWithRequestErrors
		case "logdebugwitheventstreambody":
			log.Info("S3 logging level set to LogDebugWithEventStreamBody")
			logLevel = aws.LogDebugWithEventStreamBody
		default:
			log.Infof("unknown loglevel %q, S3 logging level set to LogOff", param)
		}
	}

	return logLevel
}

// FromParameters constructs a new Driver with a given parameters map
// Required parameters:
// - accesskey
// - secretkey
// - region
// - bucket
// - encrypt
func FromParameters(parameters map[string]interface{}) (*Driver, error) {
	// Providing no values for these is valid in case the user is authenticating
	// with an IAM on an ec2 instance (in which case the instance credentials will
	// be summoned when GetAuth is called)
	accessKey := parameters["accesskey"]
	if accessKey == nil {
		accessKey = ""
	}
	secretKey := parameters["secretkey"]
	if secretKey == nil {
		secretKey = ""
	}

	regionEndpoint := parameters["regionendpoint"]
	if regionEndpoint == nil {
		regionEndpoint = ""
	}

	var result *multierror.Error
	regionName := parameters["region"]
	if regionName == nil || fmt.Sprint(regionName) == "" {
		err := errors.New("no region parameter provided")
		result = multierror.Append(result, err)
	}
	region := fmt.Sprint(regionName)
	// Don't check the region value if a custom endpoint is provided.
	if regionEndpoint == "" {
		if _, ok := validRegions[region]; !ok {
			err := fmt.Errorf("validating region provided: %v", region)

			result = multierror.Append(result, err)
		}
	}
	bucket := parameters["bucket"]
	if bucket == nil || fmt.Sprint(bucket) == "" {
		err := errors.New("no bucket parameter provided")
		result = multierror.Append(result, err)
	}
	encryptBool := false
	encrypt := parameters["encrypt"]
	switch encrypt := encrypt.(type) {
	case string:
		b, err := strconv.ParseBool(encrypt)
		if err != nil {
			err := errors.New("the encrypt parameter should be a boolean")
			result = multierror.Append(result, err)
		}
		encryptBool = b
	case bool:
		encryptBool = encrypt
	case nil:
		// do nothing
	default:
		err := errors.New("the encrypt parameter should be a boolean")
		result = multierror.Append(result, err)
	}
	secureBool := true
	secure := parameters["secure"]
	switch secure := secure.(type) {
	case string:
		b, err := strconv.ParseBool(secure)
		if err != nil {
			err := errors.New("the secure parameter should be a boolean")
			result = multierror.Append(result, err)
		}
		secureBool = b
	case bool:
		secureBool = secure
	case nil:
		// do nothing
	default:
		err := errors.New("the secure parameter should be a boolean")
		result = multierror.Append(result, err)
	}
	skipVerifyBool := false
	skipVerify := parameters["skipverify"]
	switch skipVerify := skipVerify.(type) {
	case string:
		b, err := strconv.ParseBool(skipVerify)
		if err != nil {
			err := errors.New("the skipVerify parameter should be a boolean")
			result = multierror.Append(result, err)
		}
		skipVerifyBool = b
	case bool:
		skipVerifyBool = skipVerify
	case nil:
		// do nothing
	default:
		err := errors.New("the skipVerify parameter should be a boolean")
		result = multierror.Append(result, err)
	}
	v4Bool := true
	v4auth := parameters["v4auth"]
	switch v4auth := v4auth.(type) {
	case string:
		b, err := strconv.ParseBool(v4auth)
		if err != nil {
			err := errors.New("the v4auth parameter should be a boolean")
			result = multierror.Append(result, err)
		}
		v4Bool = b
	case bool:
		v4Bool = v4auth
	case nil:
		// do nothing
	default:
		err := errors.New("the v4auth parameter should be a boolean")
		result = multierror.Append(result, err)
	}

	keyID := parameters["keyid"]
	if keyID == nil {
		keyID = ""
	}

	chunkSize, err := getParameterAsInt64(parameters, "chunksize", defaultChunkSize, minChunkSize, maxChunkSize)
	if err != nil {
		err := fmt.Errorf("converting chunksize to int64: %w", err)
		result = multierror.Append(result, err)
	}

	multipartCopyChunkSize, err := getParameterAsInt64(parameters, "multipartcopychunksize", defaultMultipartCopyChunkSize, minChunkSize, maxChunkSize)
	if err != nil {
		err := fmt.Errorf("converting multipartcopychunksize to valid int64: %w", err)
		result = multierror.Append(result, err)
	}

	multipartCopyMaxConcurrency, err := getParameterAsInt64(parameters, "multipartcopymaxconcurrency", defaultMultipartCopyMaxConcurrency, 1, math.MaxInt64)
	if err != nil {
		err := fmt.Errorf("converting multipartcopymaxconcurrency to valid int64: %w", err)
		result = multierror.Append(result, err)
	}

	multipartCopyThresholdSize, err := getParameterAsInt64(parameters, "multipartcopythresholdsize", defaultMultipartCopyThresholdSize, 0, maxChunkSize)
	if err != nil {
		err := fmt.Errorf("converting multipartcopythresholdsize to valid int64: %w", err)
		result = multierror.Append(result, err)
	}

	rootDirectory := parameters["rootdirectory"]
	if rootDirectory == nil {
		rootDirectory = ""
	}

	storageClass := s3.StorageClassStandard
	storageClassParam := parameters["storageclass"]
	if storageClassParam != nil {
		storageClassString, ok := storageClassParam.(string)
		if !ok {
			err := fmt.Errorf("the storageclass parameter must be one of %v, %v invalid",
				[]string{s3.StorageClassStandard, s3.StorageClassReducedRedundancy}, storageClassParam)
			result = multierror.Append(result, err)
		}
		// All valid storage class parameters are UPPERCASE, so be a bit more flexible here
		storageClassString = strings.ToUpper(storageClassString)
		if storageClassString != noStorageClass &&
			storageClassString != s3.StorageClassStandard &&
			storageClassString != s3.StorageClassReducedRedundancy {
			err := fmt.Errorf("the storageclass parameter must be one of %v, %v invalid",
				[]string{noStorageClass, s3.StorageClassStandard, s3.StorageClassReducedRedundancy}, storageClassParam)
			result = multierror.Append(result, err)
		}
		storageClass = storageClassString
	}

	objectACL := s3.ObjectCannedACLPrivate
	objectACLParam := parameters["objectacl"]
	if objectACLParam != nil {
		objectACLString, ok := objectACLParam.(string)
		if !ok {
			err := fmt.Errorf("object ACL parameter should be a string: %v", objectACLParam)
			result = multierror.Append(result, err)
		}

		if _, ok = validObjectACLs[objectACLString]; !ok {
			var objectACLkeys []string
			for key := range validObjectACLs {
				objectACLkeys = append(objectACLkeys, key)
			}
			err := fmt.Errorf("object ACL parameter should be one of %v: %v", objectACLkeys, objectACLParam)
			result = multierror.Append(result, err)
		}
		objectACL = objectACLString
	}

	pathStyleBool := false

	// If regionEndpoint is set, default to forcining pathstyle to preserve legacy behavior.
	if regionEndpoint != "" {
		pathStyleBool = true
	}

	pathStyle := parameters["pathstyle"]
	switch pathStyle := pathStyle.(type) {
	case string:
		b, err := strconv.ParseBool(pathStyle)
		if err != nil {
			err := errors.New("the pathstyle parameter should be a boolean")
			result = multierror.Append(result, err)
		}
		pathStyleBool = b
	case bool:
		pathStyleBool = pathStyle
	case nil:
		// do nothing
	default:
		err := errors.New("the pathstyle parameter should be a boolean")
		result = multierror.Append(result, err)
	}

	var parallelWalkBool bool

	parallelWalk := parameters["parallelwalk"]
	switch parallelWalk := parallelWalk.(type) {
	case string:
		b, err := strconv.ParseBool(parallelWalk)
		if err != nil {
			err := errors.New("the parallelwalk parameter should be a boolean")
			result = multierror.Append(result, err)
		}
		parallelWalkBool = b
	case bool:
		parallelWalkBool = parallelWalk
	case nil:
		// do nothing
	default:
		err := errors.New("the parallelwalk parameter should be a boolean")
		result = multierror.Append(result, err)
	}

	maxRequestsPerSecond, err := getParameterAsInt64(parameters, "maxrequestspersecond", defaultMaxRequestsPerSecond, 0, math.MaxInt64)
	if err != nil {
		err = fmt.Errorf("converting maxrequestspersecond to valid int64: %w", err)
		result = multierror.Append(result, err)
	}

	maxRetries, err := getParameterAsInt64(parameters, "maxretries", defaultMaxRetries, 0, math.MaxInt64)
	if err != nil {
		err := fmt.Errorf("converting maxrequestspersecond to valid int64: %w", err)
		result = multierror.Append(result, err)
	}

	// multierror return
	if err := result.ErrorOrNil(); err != nil {
		return nil, err
	}

	sessionToken := ""

	logLevel := parseLogLevelParam(parameters["loglevel"])

	params := DriverParameters{
		fmt.Sprint(accessKey),
		fmt.Sprint(secretKey),
		fmt.Sprint(bucket),
		region,
		fmt.Sprint(regionEndpoint),
		encryptBool,
		fmt.Sprint(keyID),
		secureBool,
		skipVerifyBool,
		v4Bool,
		chunkSize,
		multipartCopyChunkSize,
		multipartCopyMaxConcurrency,
		multipartCopyThresholdSize,
		fmt.Sprint(rootDirectory),
		storageClass,
		objectACL,
		fmt.Sprint(sessionToken),
		pathStyleBool,
		maxRequestsPerSecond,
		maxRetries,
		parallelWalkBool,
		logLevel,
	}

	return New(params)
}

// getParameterAsInt64 converts parameters[name] to an int64 value (using
// default if nil), verifies it is no smaller than min, and returns it.
func getParameterAsInt64(parameters map[string]interface{}, name string, defaultt int64, min int64, max int64) (int64, error) {
	rv := defaultt
	param := parameters[name]
	switch v := param.(type) {
	case string:
		vv, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("%s parameter must be an integer, %v invalid", name, param)
		}
		rv = vv
	case int64:
		rv = v
	case int, uint, int32, uint32, uint64:
		rv = reflect.ValueOf(v).Convert(reflect.TypeOf(rv)).Int()
	case nil:
		// do nothing
	default:
		return 0, fmt.Errorf("converting value for %s: %#v", name, param)
	}

	if rv < min || rv > max {
		return 0, fmt.Errorf("the %s %#v parameter should be a number between %d and %d (inclusive)", name, rv, min, max)
	}

	return rv, nil
}

// New constructs a new Driver with the given AWS credentials, region, encryption flag, and
// bucketName
func New(params DriverParameters) (*Driver, error) {
	if !params.V4Auth &&
		(params.RegionEndpoint == "" ||
			strings.Contains(params.RegionEndpoint, "s3.amazonaws.com")) {
		return nil, fmt.Errorf("on Amazon S3 this storage driver can only be used with v4 authentication")
	}

	awsConfig := aws.NewConfig().WithLogLevel(params.LogLevel)
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("creating a new session: %w", err)
	}
	creds := credentials.NewChainCredentials([]credentials.Provider{
		&credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     params.AccessKey,
				SecretAccessKey: params.SecretKey,
				SessionToken:    params.SessionToken,
			},
		},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{},
		&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(sess)},
        webIdentityProvider(sess),
	})

	if params.RegionEndpoint != "" {
		awsConfig.WithEndpoint(params.RegionEndpoint)
	}

	awsConfig.WithS3ForcePathStyle(params.PathStyle)
	awsConfig.WithCredentials(creds)
	awsConfig.WithRegion(params.Region)
	awsConfig.WithDisableSSL(!params.Secure)

	if params.SkipVerify {
		awsConfig.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		})
	}

	sess, err = session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("creating a new session with aws config: %w", err)
	}

	userAgentHandler := request.NamedHandler{
		Name: "user-agent",
		Fn:   request.MakeAddToUserAgentHandler("docker-distribution", version.Version, runtime.Version()),
	}
	sess.Handlers.Build.PushFrontNamed(userAgentHandler)

	s3obj := s3.New(sess)

	// enable S3 compatible signature v2 signing instead
	if !params.V4Auth {
		setv2Handlers(s3obj)
	}

	// TODO Currently multipart uploads have no timestamps, so this would be unwise
	// if you initiated a new s3driver while another one is running on the same bucket.
	// multis, _, err := bucket.ListMulti("", "")
	// if err != nil {
	// 	return nil, err
	// }

	// for _, multi := range multis {
	// 	err := multi.Abort()
	// 	//TODO appropriate to do this error checking?
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	w := newS3Wrapper(
		s3obj,
		withRateLimit(params.MaxRequestsPerSecond, defaultBurst),
		withExponentialBackoff(params.MaxRetries),
		withBackoffNotify(func(err error, t time.Duration) {
			log.WithFields(log.Fields{"error": err, "delay_s": t.Seconds()}).Info("S3: retrying after error")
		}),
	)

	d := &driver{
		S3:                          w,
		Bucket:                      params.Bucket,
		ChunkSize:                   params.ChunkSize,
		Encrypt:                     params.Encrypt,
		KeyID:                       params.KeyID,
		MultipartCopyChunkSize:      params.MultipartCopyChunkSize,
		MultipartCopyMaxConcurrency: params.MultipartCopyMaxConcurrency,
		MultipartCopyThresholdSize:  params.MultipartCopyThresholdSize,
		RootDirectory:               params.RootDirectory,
		StorageClass:                params.StorageClass,
		ObjectACL:                   params.ObjectACL,
		ParallelWalk:                params.ParallelWalk,
	}

	return &Driver{
		baseEmbed: baseEmbed{
			Base: base.Base{
				StorageDriver: d,
			},
		},
	}, nil
}

func webIdentityProvider(sess client.ConfigProvider) credentials.Provider {
	svc := sts.New(sess)

	roleARN := os.Getenv("AWS_ROLE_ARN")
	tokenFilepath := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	roleSessionName := os.Getenv("AWS_ROLE_SESSION_NAME")
	return stscreds.NewWebIdentityRoleProvider(svc, roleARN, roleSessionName, tokenFilepath)
}

// Implement the storagedriver.StorageDriver interface

func (d *driver) Name() string {
	return driverName
}

// GetContent retrieves the content stored at "path" as a []byte.
func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	reader, err := d.Reader(ctx, path, 0)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

// PutContent stores the []byte content at a location designated by "path".
func (d *driver) PutContent(ctx context.Context, path string, contents []byte) error {
	_, err := d.S3.PutObjectWithContext(
		ctx,
		&s3.PutObjectInput{
			Bucket:               aws.String(d.Bucket),
			Key:                  aws.String(d.s3Path(path)),
			ContentType:          d.getContentType(),
			ACL:                  d.getACL(),
			ServerSideEncryption: d.getEncryptionMode(),
			SSEKMSKeyId:          d.getSSEKMSKeyID(),
			StorageClass:         d.getStorageClass(),
			Body:                 bytes.NewReader(contents),
		})
	return parseError(path, err)
}

// Reader retrieves an io.ReadCloser for the content stored at "path" with a
// given byte offset.
func (d *driver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	resp, err := d.S3.GetObjectWithContext(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(d.Bucket),
			Key:    aws.String(d.s3Path(path)),
			Range:  aws.String("bytes=" + strconv.FormatInt(offset, 10) + "-"),
		})

	if err != nil {
		if s3Err, ok := err.(awserr.Error); ok && s3Err.Code() == "InvalidRange" {
			return ioutil.NopCloser(bytes.NewReader(nil)), nil
		}

		return nil, parseError(path, err)
	}
	return resp.Body, nil
}

// Writer returns a FileWriter which will store the content written to it
// at the location designated by "path" after the call to Commit.
func (d *driver) Writer(ctx context.Context, path string, append bool) (storagedriver.FileWriter, error) {
	key := d.s3Path(path)
	if !append {
		// TODO (brianbland): cancel other uploads at this path

		resp, err := d.S3.CreateMultipartUploadWithContext(
			ctx,
			&s3.CreateMultipartUploadInput{
				Bucket:               aws.String(d.Bucket),
				Key:                  aws.String(key),
				ContentType:          d.getContentType(),
				ACL:                  d.getACL(),
				ServerSideEncryption: d.getEncryptionMode(),
				SSEKMSKeyId:          d.getSSEKMSKeyID(),
				StorageClass:         d.getStorageClass(),
			})
		if err != nil {
			return nil, err
		}
		return d.newWriter(key, *resp.UploadId, nil), nil
	}

	resp, err := d.S3.ListMultipartUploadsWithContext(
		ctx,
		&s3.ListMultipartUploadsInput{
			Bucket: aws.String(d.Bucket),
			Prefix: aws.String(key),
		})
	if err != nil {
		return nil, parseError(path, err)
	}

	for _, multi := range resp.Uploads {
		if key != *multi.Key {
			continue
		}

		resp, err := d.S3.ListPartsWithContext(
			ctx,
			&s3.ListPartsInput{
				Bucket:   aws.String(d.Bucket),
				Key:      aws.String(key),
				UploadId: multi.UploadId,
			})
		if err != nil {
			return nil, parseError(path, err)
		}
		var multiSize int64
		for _, part := range resp.Parts {
			multiSize += *part.Size
		}
		return d.newWriter(key, *multi.UploadId, resp.Parts), nil
	}
	return nil, storagedriver.PathNotFoundError{Path: path}
}

// Stat retrieves the FileInfo for the given path, including the current size
// in bytes and the creation time.
func (d *driver) Stat(ctx context.Context, path string) (storagedriver.FileInfo, error) {
	resp, err := d.S3.ListObjectsV2WithContext(
		ctx,
		&s3.ListObjectsV2Input{
			Bucket:  aws.String(d.Bucket),
			Prefix:  aws.String(d.s3Path(path)),
			MaxKeys: aws.Int64(1),
		})
	if err != nil {
		return nil, err
	}

	fi := storagedriver.FileInfoFields{
		Path: path,
	}

	if len(resp.Contents) == 1 {
		if *resp.Contents[0].Key != d.s3Path(path) {
			fi.IsDir = true
		} else {
			fi.IsDir = false
			fi.Size = *resp.Contents[0].Size
			fi.ModTime = *resp.Contents[0].LastModified
		}
	} else if len(resp.CommonPrefixes) == 1 {
		fi.IsDir = true
	} else {
		return nil, storagedriver.PathNotFoundError{Path: path}
	}

	return storagedriver.FileInfoInternal{FileInfoFields: fi}, nil
}

// List returns a list of the objects that are direct descendants of the given path.
func (d *driver) List(ctx context.Context, opath string) ([]string, error) {
	path := opath
	if path != "/" && path[len(path)-1] != '/' {
		path = path + "/"
	}

	// This is to cover for the cases when the rootDirectory of the driver is either "" or "/".
	// In those cases, there is no root prefix to replace and we must actually add a "/" to all
	// results in order to keep them as valid paths as recognized by storagedriver.PathRegexp
	prefix := ""
	if d.s3Path("") == "" {
		prefix = "/"
	}

	resp, err := d.S3.ListObjectsV2WithContext(
		ctx,
		&s3.ListObjectsV2Input{
			Bucket:    aws.String(d.Bucket),
			Prefix:    aws.String(d.s3Path(path)),
			Delimiter: aws.String("/"),
			MaxKeys:   aws.Int64(listMax),
		})
	if err != nil {
		return nil, parseError(opath, err)
	}

	files := []string{}
	directories := []string{}

	for {
		for _, key := range resp.Contents {
			files = append(files, strings.Replace(*key.Key, d.s3Path(""), prefix, 1))
		}

		for _, commonPrefix := range resp.CommonPrefixes {
			commonPrefix := *commonPrefix.Prefix
			directories = append(directories, strings.Replace(commonPrefix[0:len(commonPrefix)-1], d.s3Path(""), prefix, 1))
		}

		if *resp.IsTruncated {
			resp, err = d.S3.ListObjectsV2WithContext(
				ctx,
				&s3.ListObjectsV2Input{
					Bucket:            aws.String(d.Bucket),
					Prefix:            aws.String(d.s3Path(path)),
					Delimiter:         aws.String("/"),
					MaxKeys:           aws.Int64(listMax),
					ContinuationToken: resp.NextContinuationToken,
				})
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	if opath != "/" {
		if len(files) == 0 && len(directories) == 0 {
			// Treat empty response as missing directory, since we don't actually
			// have directories in s3.
			return nil, storagedriver.PathNotFoundError{Path: opath}
		}
	}

	return append(files, directories...), nil
}

// Move moves an object stored at sourcePath to destPath, removing the original
// object.
func (d *driver) Move(ctx context.Context, sourcePath string, destPath string) error {
	/* This is terrible, but aws doesn't have an actual move. */
	if err := d.copy(ctx, sourcePath, destPath); err != nil {
		return err
	}
	return d.Delete(ctx, sourcePath)
}

// copy copies an object stored at sourcePath to destPath.
func (d *driver) copy(ctx context.Context, sourcePath string, destPath string) error {
	// S3 can copy objects up to 5 GB in size with a single PUT Object - Copy
	// operation. For larger objects, the multipart upload API must be used.
	//
	// Empirically, multipart copy is fastest with 32 MB parts and is faster
	// than PUT Object - Copy for objects larger than 32 MB.

	fileInfo, err := d.Stat(ctx, sourcePath)
	if err != nil {
		return parseError(sourcePath, err)
	}

	if fileInfo.Size() <= d.MultipartCopyThresholdSize {
		_, err = d.S3.CopyObjectWithContext(
			ctx,
			&s3.CopyObjectInput{
				Bucket:               aws.String(d.Bucket),
				Key:                  aws.String(d.s3Path(destPath)),
				ContentType:          d.getContentType(),
				ACL:                  d.getACL(),
				ServerSideEncryption: d.getEncryptionMode(),
				SSEKMSKeyId:          d.getSSEKMSKeyID(),
				StorageClass:         d.getStorageClass(),
				CopySource:           aws.String(d.Bucket + "/" + d.s3Path(sourcePath)),
			})
		if err != nil {
			return parseError(sourcePath, err)
		}
		return nil
	}

	createResp, err := d.S3.CreateMultipartUploadWithContext(
		ctx,
		&s3.CreateMultipartUploadInput{
			Bucket:               aws.String(d.Bucket),
			Key:                  aws.String(d.s3Path(destPath)),
			ContentType:          d.getContentType(),
			ACL:                  d.getACL(),
			SSEKMSKeyId:          d.getSSEKMSKeyID(),
			ServerSideEncryption: d.getEncryptionMode(),
			StorageClass:         d.getStorageClass(),
		})
	if err != nil {
		return err
	}

	numParts := (fileInfo.Size() + d.MultipartCopyChunkSize - 1) / d.MultipartCopyChunkSize
	completedParts := make([]*s3.CompletedPart, numParts)
	errChan := make(chan error, numParts)

	// Reduce the client/server exposure to long lived connections regardless of
	// how many requests per second are allowed.
	limiter := make(chan struct{}, d.MultipartCopyMaxConcurrency)

	for i := range completedParts {
		i := int64(i)
		go func() {
			limiter <- struct{}{}

			firstByte := i * d.MultipartCopyChunkSize
			lastByte := firstByte + d.MultipartCopyChunkSize - 1
			if lastByte >= fileInfo.Size() {
				lastByte = fileInfo.Size() - 1
			}

			uploadResp, err := d.S3.UploadPartCopyWithContext(
				ctx,
				&s3.UploadPartCopyInput{
					Bucket:          aws.String(d.Bucket),
					CopySource:      aws.String(d.Bucket + "/" + d.s3Path(sourcePath)),
					Key:             aws.String(d.s3Path(destPath)),
					PartNumber:      aws.Int64(i + 1),
					UploadId:        createResp.UploadId,
					CopySourceRange: aws.String(fmt.Sprintf("bytes=%d-%d", firstByte, lastByte)),
				})
			if err == nil {
				completedParts[i] = &s3.CompletedPart{
					ETag:       uploadResp.CopyPartResult.ETag,
					PartNumber: aws.Int64(i + 1),
				}
			}
			errChan <- err
			<-limiter
		}()
	}

	for range completedParts {
		err := <-errChan
		if err != nil {
			return err
		}
	}

	_, err = d.S3.CompleteMultipartUploadWithContext(
		ctx,
		&s3.CompleteMultipartUploadInput{
			Bucket:          aws.String(d.Bucket),
			Key:             aws.String(d.s3Path(destPath)),
			UploadId:        createResp.UploadId,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: completedParts},
		})
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Delete recursively deletes all objects stored at "path" and its subpaths.
// We must be careful since S3 does not guarantee read after delete consistency
func (d *driver) Delete(ctx context.Context, path string) error {
	s3Objects := make([]*s3.ObjectIdentifier, 0, listMax)
	s3Path := d.s3Path(path)
	listObjectsV2Input := &s3.ListObjectsV2Input{
		Bucket: aws.String(d.Bucket),
		Prefix: aws.String(s3Path),
	}
ListLoop:
	for {
		// list all the objects
		resp, err := d.S3.ListObjectsV2WithContext(ctx, listObjectsV2Input)

		// resp.Contents can only be empty on the first call
		// if there were no more results to return after the first call, resp.IsTruncated would have been false
		// and the loop would be exited without recalling ListObjects
		if err != nil || len(resp.Contents) == 0 {
			return storagedriver.PathNotFoundError{Path: path}
		}

		for _, key := range resp.Contents {
			// Stop if we encounter a key that is not a subpath (so that deleting "/a" does not delete "/ab").
			if len(*key.Key) > len(s3Path) && (*key.Key)[len(s3Path)] != '/' {
				break ListLoop
			}
			s3Objects = append(s3Objects, &s3.ObjectIdentifier{
				Key: key.Key,
			})
		}

		// resp.Contents must have at least one element or we would have returned not found
		listObjectsV2Input.StartAfter = resp.Contents[len(resp.Contents)-1].Key

		// from the s3 api docs, IsTruncated "specifies whether (true) or not (false) all of the results were returned"
		// if everything has been returned, break
		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
	}

	// need to chunk objects into groups of deleteMax per s3 restrictions
	total := len(s3Objects)
	for i := 0; i < total; i += deleteMax {
		_, err := d.S3.DeleteObjectsWithContext(
			ctx,
			&s3.DeleteObjectsInput{
				Bucket: aws.String(d.Bucket),
				Delete: &s3.Delete{
					Objects: s3Objects[i:min(i+deleteMax, total)],
					Quiet:   aws.Bool(false),
				},
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteFiles deletes a set of files using the S3 bulk delete feature, with up to deleteMax files per request. If
// deleting more than deleteMax files, DeleteFiles will split files in deleteMax requests automatically. A separate
// goroutine is created for each request. Contrary to Delete, which is a generic method to delete any kind of object,
// DeleteFiles does not send a ListObjects request before DeleteObjects. Returns the number of successfully deleted
// files and any errors. This method is idempotent, no error is returned if a file does not exist.
func (d *driver) DeleteFiles(ctx context.Context, paths []string) (int, error) {
	s3Objects := make([]*s3.ObjectIdentifier, 0, len(paths))
	for i := range paths {
		p := d.s3Path(paths[i])
		s3Objects = append(s3Objects, &s3.ObjectIdentifier{Key: &p})
	}

	// collect errors from concurrent DeleteObjects requests
	var errs error
	errCh := make(chan error)
	errDone := make(chan struct{})
	go func() {
		for err := range errCh {
			errs = multierror.Append(errs, err)
		}
		errDone <- struct{}{}
	}()

	// count the number of successfully deleted files across concurrent DeleteObjects requests
	count := 0
	countCh := make(chan int)
	countDone := make(chan struct{})
	go func() {
		for n := range countCh {
			count += n
		}
		countDone <- struct{}{}
	}()

	// chunk files into batches of deleteMax (as per S3 restrictions), creating a goroutine per batch
	var wg sync.WaitGroup
	total := len(s3Objects)
	for i := 0; i < total; i += deleteMax {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			resp, err := d.S3.DeleteObjectsWithContext(
				ctx,
				&s3.DeleteObjectsInput{
					Bucket: aws.String(d.Bucket),
					Delete: &s3.Delete{
						Objects: s3Objects[i:min(i+deleteMax, total)],
						Quiet:   aws.Bool(false),
					},
				})
			if err != nil {
				errCh <- err
				return
			}

			// count successfully deleted files
			countCh <- len(resp.Deleted)

			// even if err is nil (200 OK response) it's not guaranteed that all files have been successfully deleted,
			// we need to check the []*s3.Error slice within the S3 response and make sure it's empty
			if len(resp.Errors) > 0 {
				// parse s3.Error errors and return a single storagedriver.MultiError
				var errs error
				for _, s3e := range resp.Errors {
					err := fmt.Errorf("deleting file '%s': '%s'", *s3e.Key, *s3e.Message)
					errs = multierror.Append(errs, err)
				}
				errCh <- errs
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	<-errDone
	close(countCh)
	<-countDone

	return count, errs
}

// URLFor returns a URL which may be used to retrieve the content stored at the given path.
// May return an UnsupportedMethodErr in certain StorageDriver implementations.
func (d *driver) URLFor(ctx context.Context, path string, options map[string]interface{}) (string, error) {
	methodString := "GET"
	method, ok := options["method"]
	if ok {
		methodString, ok = method.(string)
		if !ok || (methodString != "GET" && methodString != "HEAD") {
			return "", storagedriver.ErrUnsupportedMethod{}
		}
	}

	expiresIn := 20 * time.Minute
	expires, ok := options["expiry"]
	if ok {
		et, ok := expires.(time.Time)
		if ok {
			expiresIn = et.Sub(time.Now())
		}
	}

	var req *request.Request

	switch methodString {
	case "GET":
		req, _ = d.S3.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(d.Bucket),
			Key:    aws.String(d.s3Path(path)),
		})
	case "HEAD":
		req, _ = d.S3.HeadObjectRequest(&s3.HeadObjectInput{
			Bucket: aws.String(d.Bucket),
			Key:    aws.String(d.s3Path(path)),
		})
	default:
		panic("unreachable")
	}

	return req.Presign(expiresIn)
}

// Walk traverses a filesystem defined within driver, starting
// from the given path, calling f on each file
func (d *driver) Walk(ctx context.Context, from string, f storagedriver.WalkFn) error {
	path := from
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	prefix := ""
	if d.s3Path("") == "" {
		prefix = "/"
	}

	var objectCount int64
	if err := d.doWalk(ctx, &objectCount, d.s3Path(path), prefix, f); err != nil {
		return err
	}

	// S3 doesn't have the concept of empty directories, so it'll return path not found if there are no objects
	if objectCount == 0 {
		return storagedriver.PathNotFoundError{Path: from}
	}

	return nil
}

// WalkParallel traverses a filesystem defined within driver, starting
// from the given path, calling f on each file.
func (d *driver) WalkParallel(ctx context.Context, from string, f storagedriver.WalkFn) error {
	// If the ParallelWalk feature flag is not set, fall back to standard sequential walk.
	if !d.ParallelWalk {
		return d.Walk(ctx, from, f)
	}

	path := from
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	prefix := ""
	if d.s3Path("") == "" {
		prefix = "/"
	}

	var objectCount int64
	var retError error
	countChan := make(chan int64)
	countDone := make(chan struct{})
	errors := make(chan error)
	errDone := make(chan struct{})
	quit := make(chan struct{})

	// Consume object counts from each doWalkParallel call asynchronusly to avoid blocking.
	go func() {
		for i := range countChan {
			objectCount += i
		}
		countDone <- struct{}{}
	}()

	// If we encounter an error from any goroutine called from within doWalkParallel,
	// return early from any new goroutines and return that error.
	go func() {
		var closed bool
		// Consume all errors to prevent goroutines from blocking and to
		// report errors from goroutines that were already in progress.
		for err := range errors {
			// Signal goroutines to quit only once on the first error.
			if !closed {
				close(quit)
				closed = true
			}

			if err != nil {
				retError = multierror.Append(retError, err)
			}
		}
		errDone <- struct{}{}
	}()

	// doWalkParallel spawns and manages it's own goroutines, but it also calls
	// itself recursively. Passing in a WaitGroup allows us to wait for the
	// entire walk to complete without blocking on each doWalkParallel call.
	var wg sync.WaitGroup

	d.doWalkParallel(ctx, &wg, countChan, quit, errors, d.s3Path(path), prefix, f)

	wg.Wait()

	// Ensure that all object counts have been totaled before continuing.
	close(countChan)
	close(errors)
	<-countDone
	<-errDone

	// S3 doesn't have the concept of empty directories, so it'll return path not found if there are no objects
	if objectCount == 0 {
		return storagedriver.PathNotFoundError{Path: from}
	}

	return retError
}

func (d *driver) TransferTo(ctx context.Context, destDriver storagedriver.StorageDriver, src, dest string) error {
	return storagedriver.ErrUnsupportedMethod{}
}

type walkInfoContainer struct {
	storagedriver.FileInfoFields
	prefix *string
}

// Path provides the full path of the target of this file info.
func (wi walkInfoContainer) Path() string {
	return wi.FileInfoFields.Path
}

// Size returns current length in bytes of the file. The return value can
// be used to write to the end of the file at path. The value is
// meaningless if IsDir returns true.
func (wi walkInfoContainer) Size() int64 {
	return wi.FileInfoFields.Size
}

// ModTime returns the modification time for the file. For backends that
// don't have a modification time, the creation time should be returned.
func (wi walkInfoContainer) ModTime() time.Time {
	return wi.FileInfoFields.ModTime
}

// IsDir returns true if the path is a directory.
func (wi walkInfoContainer) IsDir() bool {
	return wi.FileInfoFields.IsDir
}

func (d *driver) doWalk(parentCtx context.Context, objectCount *int64, path, prefix string, f storagedriver.WalkFn) error {
	var retError error

	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.Bucket),
		Prefix:    aws.String(path),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(listMax),
	}

	ctx, done := dcontext.WithTrace(parentCtx)
	defer done("s3aws.ListObjectsV2Pages(%s)", path)

	listObjectErr := d.S3.ListObjectsV2PagesWithContext(ctx, listObjectsInput, func(objects *s3.ListObjectsV2Output, lastPage bool) bool {
		var count int64
		// KeyCount was introduced with version 2 of the GET Bucket operation in S3.
		// Some S3 implementations don't support V2 now, so we fall back to manual
		// calculation of the key count if required
		if objects.KeyCount != nil {
			count = *objects.KeyCount
		} else {
			count = int64(len(objects.Contents) + len(objects.CommonPrefixes))
		}

		*objectCount += count

		walkInfos := make([]walkInfoContainer, 0, count)

		for _, dir := range objects.CommonPrefixes {
			commonPrefix := *dir.Prefix
			walkInfos = append(walkInfos, walkInfoContainer{
				prefix: dir.Prefix,
				FileInfoFields: storagedriver.FileInfoFields{
					IsDir: true,
					Path:  strings.Replace(commonPrefix[:len(commonPrefix)-1], d.s3Path(""), prefix, 1),
				},
			})
		}

		for _, file := range objects.Contents {
			walkInfos = append(walkInfos, walkInfoContainer{
				FileInfoFields: storagedriver.FileInfoFields{
					IsDir:   false,
					Size:    *file.Size,
					ModTime: *file.LastModified,
					Path:    strings.Replace(*file.Key, d.s3Path(""), prefix, 1),
				},
			})
		}

		sort.SliceStable(walkInfos, func(i, j int) bool { return walkInfos[i].FileInfoFields.Path < walkInfos[j].FileInfoFields.Path })

		for _, walkInfo := range walkInfos {
			err := f(walkInfo)

			if err == storagedriver.ErrSkipDir {
				if walkInfo.IsDir() {
					continue
				} else {
					break
				}
			} else if err != nil {
				retError = err
				return false
			}

			if walkInfo.IsDir() {
				if err := d.doWalk(ctx, objectCount, *walkInfo.prefix, prefix, f); err != nil {
					retError = err
					return false
				}
			}
		}
		return true
	})

	if retError != nil {
		return retError
	}

	if listObjectErr != nil {
		return listObjectErr
	}

	return nil
}

func (d *driver) doWalkParallel(parentCtx context.Context, wg *sync.WaitGroup, countChan chan<- int64, quit <-chan struct{}, errors chan<- error, path, prefix string, f storagedriver.WalkFn) {
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.Bucket),
		Prefix:    aws.String(path),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(listMax),
	}

	ctx, done := dcontext.WithTrace(parentCtx)
	defer done("s3aws.ListObjectsV2Pages(%s)", path)

	listObjectErr := d.S3.ListObjectsV2PagesWithContext(ctx, listObjectsInput, func(objects *s3.ListObjectsV2Output, lastPage bool) bool {
		select {
		// The walk was canceled, return to stop requests for pages and prevent gorountines from leaking.
		case <-quit:
			return false
		default:

			var count int64
			// KeyCount was introduced with version 2 of the GET Bucket operation in S3.
			// Some S3 implementations don't support V2 now, so we fall back to manual
			// calculation of the key count if required
			if objects.KeyCount != nil {
				count = *objects.KeyCount
			} else {
				count = int64(len(objects.Contents) + len(objects.CommonPrefixes))
			}
			countChan <- count

			walkInfos := make([]walkInfoContainer, 0, count)

			for _, dir := range objects.CommonPrefixes {
				commonPrefix := *dir.Prefix
				walkInfos = append(walkInfos, walkInfoContainer{
					prefix: dir.Prefix,
					FileInfoFields: storagedriver.FileInfoFields{
						IsDir: true,
						Path:  strings.Replace(commonPrefix[:len(commonPrefix)-1], d.s3Path(""), prefix, 1),
					},
				})
			}

			for _, file := range objects.Contents {
				walkInfos = append(walkInfos, walkInfoContainer{
					FileInfoFields: storagedriver.FileInfoFields{
						IsDir:   false,
						Size:    *file.Size,
						ModTime: *file.LastModified,
						Path:    strings.Replace(*file.Key, d.s3Path(""), prefix, 1),
					},
				})
			}

			for _, walkInfo := range walkInfos {
				wg.Add(1)
				wInfo := walkInfo
				go func() {
					defer wg.Done()

					err := f(wInfo)

					if err == storagedriver.ErrSkipDir && wInfo.IsDir() {
						return
					}

					if err != nil {
						errors <- err
					}

					if wInfo.IsDir() {
						d.doWalkParallel(ctx, wg, countChan, quit, errors, *wInfo.prefix, prefix, f)
					}
				}()
			}
		}
		return true
	})

	if listObjectErr != nil {
		errors <- listObjectErr
	}
}

func (d *driver) s3Path(path string) string {
	return strings.TrimLeft(strings.TrimRight(d.RootDirectory, "/")+path, "/")
}

// S3BucketKey returns the s3 bucket key for the given storage driver path.
func (d *Driver) S3BucketKey(path string) string {
	return d.StorageDriver.(*driver).s3Path(path)
}

func parseError(path string, err error) error {
	if s3Err, ok := err.(awserr.Error); ok && s3Err.Code() == "NoSuchKey" {
		return storagedriver.PathNotFoundError{Path: path}
	}

	return err
}

func (d *driver) getEncryptionMode() *string {
	if !d.Encrypt {
		return nil
	}
	if d.KeyID == "" {
		return aws.String("AES256")
	}
	return aws.String("aws:kms")
}

func (d *driver) getSSEKMSKeyID() *string {
	if d.KeyID != "" {
		return aws.String(d.KeyID)
	}
	return nil
}

func (d *driver) getContentType() *string {
	return aws.String("application/octet-stream")
}

func (d *driver) getACL() *string {
	return aws.String(d.ObjectACL)
}

func (d *driver) getStorageClass() *string {
	if d.StorageClass == noStorageClass {
		return nil
	}
	return aws.String(d.StorageClass)
}

// writer attempts to upload parts to S3 in a buffered fashion where the last
// part is at least as large as the chunksize, so the multipart upload could be
// cleanly resumed in the future. This is violated if Close is called after less
// than a full chunk is written.
type writer struct {
	driver      *driver
	key         string
	uploadID    string
	parts       []*s3.Part
	size        int64
	readyPart   []byte
	pendingPart []byte
	closed      bool
	committed   bool
	canceled    bool
}

func (d *driver) newWriter(key, uploadID string, parts []*s3.Part) storagedriver.FileWriter {
	var size int64
	for _, part := range parts {
		size += *part.Size
	}
	return &writer{
		driver:   d,
		key:      key,
		uploadID: uploadID,
		parts:    parts,
		size:     size,
	}
}

type completedParts []*s3.CompletedPart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return *a[i].PartNumber < *a[j].PartNumber }

func (w *writer) Write(p []byte) (int, error) {
	ctx := context.Background()

	if w.closed {
		return 0, fmt.Errorf("already closed")
	} else if w.committed {
		return 0, fmt.Errorf("already committed")
	} else if w.canceled {
		return 0, fmt.Errorf("already canceled")
	}

	// If the last written part is smaller than minChunkSize, we need to make a
	// new multipart upload :sadface:
	if len(w.parts) > 0 && int(*w.parts[len(w.parts)-1].Size) < minChunkSize {
		var completedUploadedParts completedParts
		for _, part := range w.parts {
			completedUploadedParts = append(completedUploadedParts, &s3.CompletedPart{
				ETag:       part.ETag,
				PartNumber: part.PartNumber,
			})
		}

		sort.Sort(completedUploadedParts)

		_, err := w.driver.S3.CompleteMultipartUploadWithContext(
			ctx,
			&s3.CompleteMultipartUploadInput{
				Bucket:   aws.String(w.driver.Bucket),
				Key:      aws.String(w.key),
				UploadId: aws.String(w.uploadID),
				MultipartUpload: &s3.CompletedMultipartUpload{
					Parts: completedUploadedParts,
				},
			})
		if err != nil {
			w.driver.S3.AbortMultipartUploadWithContext(
				ctx,
				&s3.AbortMultipartUploadInput{
					Bucket:   aws.String(w.driver.Bucket),
					Key:      aws.String(w.key),
					UploadId: aws.String(w.uploadID),
				})
			return 0, err
		}

		resp, err := w.driver.S3.CreateMultipartUploadWithContext(
			ctx,
			&s3.CreateMultipartUploadInput{
				Bucket:               aws.String(w.driver.Bucket),
				Key:                  aws.String(w.key),
				ContentType:          w.driver.getContentType(),
				ACL:                  w.driver.getACL(),
				ServerSideEncryption: w.driver.getEncryptionMode(),
				StorageClass:         w.driver.getStorageClass(),
			})
		if err != nil {
			return 0, err
		}
		w.uploadID = *resp.UploadId

		// If the entire written file is smaller than minChunkSize, we need to make
		// a new part from scratch :double sad face:
		if w.size < minChunkSize {
			resp, err := w.driver.S3.GetObjectWithContext(
				ctx,
				&s3.GetObjectInput{
					Bucket: aws.String(w.driver.Bucket),
					Key:    aws.String(w.key),
				})
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()
			w.parts = nil
			w.readyPart, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return 0, err
			}
		} else {
			// Otherwise we can use the old file as the new first part
			copyPartResp, err := w.driver.S3.UploadPartCopyWithContext(
				ctx,
				&s3.UploadPartCopyInput{
					Bucket:     aws.String(w.driver.Bucket),
					CopySource: aws.String(w.driver.Bucket + "/" + w.key),
					Key:        aws.String(w.key),
					PartNumber: aws.Int64(1),
					UploadId:   resp.UploadId,
				})
			if err != nil {
				return 0, err
			}
			w.parts = []*s3.Part{
				{
					ETag:       copyPartResp.CopyPartResult.ETag,
					PartNumber: aws.Int64(1),
					Size:       aws.Int64(w.size),
				},
			}
		}
	}

	var n int

	for len(p) > 0 {
		// If no parts are ready to write, fill up the first part
		if neededBytes := int(w.driver.ChunkSize) - len(w.readyPart); neededBytes > 0 {
			if len(p) >= neededBytes {
				w.readyPart = append(w.readyPart, p[:neededBytes]...)
				n += neededBytes
				p = p[neededBytes:]
			} else {
				w.readyPart = append(w.readyPart, p...)
				n += len(p)
				p = nil
			}
		}

		if neededBytes := int(w.driver.ChunkSize) - len(w.pendingPart); neededBytes > 0 {
			if len(p) >= neededBytes {
				w.pendingPart = append(w.pendingPart, p[:neededBytes]...)
				n += neededBytes
				p = p[neededBytes:]
				err := w.flushPart()
				if err != nil {
					w.size += int64(n)
					return n, err
				}
			} else {
				w.pendingPart = append(w.pendingPart, p...)
				n += len(p)
				p = nil
			}
		}
	}
	w.size += int64(n)
	return n, nil
}

func (w *writer) Size() int64 {
	return w.size
}

func (w *writer) Close() error {
	if w.closed {
		return fmt.Errorf("already closed")
	}
	w.closed = true
	return w.flushPart()
}

func (w *writer) Cancel() error {
	if w.closed {
		return fmt.Errorf("already closed")
	} else if w.committed {
		return fmt.Errorf("already committed")
	}
	w.canceled = true
	_, err := w.driver.S3.AbortMultipartUploadWithContext(
		context.Background(),
		&s3.AbortMultipartUploadInput{
			Bucket:   aws.String(w.driver.Bucket),
			Key:      aws.String(w.key),
			UploadId: aws.String(w.uploadID),
		})
	return err
}

func (w *writer) Commit() error {
	ctx := context.Background()

	if w.closed {
		return fmt.Errorf("already closed")
	} else if w.committed {
		return fmt.Errorf("already committed")
	} else if w.canceled {
		return fmt.Errorf("already canceled")
	}
	err := w.flushPart()
	if err != nil {
		return err
	}
	w.committed = true

	var completedUploadedParts completedParts
	for _, part := range w.parts {
		completedUploadedParts = append(completedUploadedParts, &s3.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		})
	}

	sort.Sort(completedUploadedParts)

	_, err = w.driver.S3.CompleteMultipartUploadWithContext(
		ctx,
		&s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(w.driver.Bucket),
			Key:      aws.String(w.key),
			UploadId: aws.String(w.uploadID),
			MultipartUpload: &s3.CompletedMultipartUpload{
				Parts: completedUploadedParts,
			},
		})
	if err != nil {
		w.driver.S3.AbortMultipartUploadWithContext(
			ctx,
			&s3.AbortMultipartUploadInput{
				Bucket:   aws.String(w.driver.Bucket),
				Key:      aws.String(w.key),
				UploadId: aws.String(w.uploadID),
			})
		return err
	}
	return nil
}

// flushPart flushes buffers to write a part to S3.
// Only called by Write (with both buffers full) and Close/Commit (always)
func (w *writer) flushPart() error {
	if len(w.readyPart) == 0 && len(w.pendingPart) == 0 {
		// nothing to write
		return nil
	}
	if len(w.pendingPart) < int(w.driver.ChunkSize) {
		// closing with a small pending part
		// combine ready and pending to avoid writing a small part
		w.readyPart = append(w.readyPart, w.pendingPart...)
		w.pendingPart = nil
	}

	partNumber := aws.Int64(int64(len(w.parts) + 1))
	resp, err := w.driver.S3.UploadPartWithContext(
		context.Background(),
		&s3.UploadPartInput{
			Bucket:     aws.String(w.driver.Bucket),
			Key:        aws.String(w.key),
			PartNumber: partNumber,
			UploadId:   aws.String(w.uploadID),
			Body:       bytes.NewReader(w.readyPart),
		})
	if err != nil {
		return err
	}
	w.parts = append(w.parts, &s3.Part{
		ETag:       resp.ETag,
		PartNumber: partNumber,
		Size:       aws.Int64(int64(len(w.readyPart))),
	})
	w.readyPart = w.pendingPart
	w.pendingPart = nil
	return nil
}
