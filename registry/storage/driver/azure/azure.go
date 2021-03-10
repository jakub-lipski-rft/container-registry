// Package azure provides a storagedriver.StorageDriver implementation to
// store blobs in Microsoft Azure Blob Storage Service.
package azure

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/base"
	"github.com/docker/distribution/registry/storage/driver/factory"

	azure "github.com/Azure/azure-sdk-for-go/storage"
)

const driverName = "azure"

const (
	paramAccountName = "accountname"
	paramAccountKey  = "accountkey"
	paramContainer   = "container"
	paramRealm       = "realm"
	maxChunkSize     = 4 * 1024 * 1024
)

type driver struct {
	client        azure.BlobStorageClient
	container     string
	rootDirectory string
}

type baseEmbed struct{ base.Base }

// Driver is a storagedriver.StorageDriver implementation backed by
// Microsoft Azure Blob Storage Service.
type Driver struct{ baseEmbed }

func init() {
	factory.Register(driverName, &azureDriverFactory{})
}

type azureDriverFactory struct{}

func (factory *azureDriverFactory) Create(parameters map[string]interface{}) (storagedriver.StorageDriver, error) {
	return FromParameters(parameters)
}

// FromParameters constructs a new Driver with a given parameters map.
func FromParameters(parameters map[string]interface{}) (*Driver, error) {
	accountName, ok := parameters[paramAccountName]
	if !ok || fmt.Sprint(accountName) == "" {
		return nil, fmt.Errorf("no %s parameter provided", paramAccountName)
	}

	accountKey, ok := parameters[paramAccountKey]
	if !ok || fmt.Sprint(accountKey) == "" {
		return nil, fmt.Errorf("no %s parameter provided", paramAccountKey)
	}

	container, ok := parameters[paramContainer]
	if !ok || fmt.Sprint(container) == "" {
		return nil, fmt.Errorf("no %s parameter provided", paramContainer)
	}

	realm, ok := parameters[paramRealm]
	if !ok || fmt.Sprint(realm) == "" {
		realm = azure.DefaultBaseURL
	}

	return New(fmt.Sprint(accountName), fmt.Sprint(accountKey), fmt.Sprint(container), fmt.Sprint(realm))
}

// New constructs a new Driver with the given Azure Storage Account credentials
func New(accountName, accountKey, container, realm string) (*Driver, error) {
	api, err := azure.NewClient(accountName, accountKey, realm, azure.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	blobClient := api.GetBlobService()

	// Create registry container
	containerRef := blobClient.GetContainerReference(container)
	if _, err = containerRef.CreateIfNotExists(nil); err != nil {
		return nil, err
	}

	d := &driver{
		client:    blobClient,
		container: container}
	return &Driver{baseEmbed: baseEmbed{Base: base.Base{StorageDriver: d}}}, nil
}

// Implement the storagedriver.StorageDriver interface.
func (d *driver) Name() string {
	return driverName
}

// GetContent retrieves the content stored at "path" as a []byte.
func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	blob, err := blobRef.Get(nil)
	if err != nil {
		if is404(err) {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}
		return nil, err
	}

	defer blob.Close()
	return ioutil.ReadAll(blob)
}

// PutContent stores the []byte content at a location designated by "path".
func (d *driver) PutContent(ctx context.Context, path string, contents []byte) error {
	// max size for block blobs uploaded via single "Put Blob" for version after "2016-05-31"
	// https://docs.microsoft.com/en-us/rest/api/storageservices/put-blob#remarks
	const limit = 256 * 1024 * 1024
	if len(contents) > limit {
		return fmt.Errorf("uploading %d bytes with PutContent is not supported; limit: %d bytes", len(contents), limit)
	}

	// Historically, blobs uploaded via PutContent used to be of type AppendBlob
	// (https://github.com/docker/distribution/pull/1438). We can't replace
	// these blobs atomically via a single "Put Blob" operation without
	// deleting them first. Once we detect they are BlockBlob type, we can
	// overwrite them with an atomically "Put Blob" operation.
	//
	// While we delete the blob and create a new one, there will be a small
	// window of inconsistency and if the Put Blob fails, we may end up with
	// losing the existing data while migrating it to BlockBlob type. However,
	// expectation is the clients pushing will be retrying when they get an error
	// response.
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	err := blobRef.GetProperties(nil)
	if err != nil && !is404(err) {
		return fmt.Errorf("failed to get blob properties: %v", err)
	}
	if err == nil && blobRef.Properties.BlobType != azure.BlobTypeBlock {
		if err := blobRef.Delete(nil); err != nil {
			return fmt.Errorf("failed to delete legacy blob (%s): %v", blobRef.Properties.BlobType, err)
		}
	}

	r := bytes.NewReader(contents)
	// reset properties to empty before doing overwrite
	blobRef.Properties = azure.BlobProperties{}
	return blobRef.CreateBlockBlobFromReader(r, nil)
}

// Reader retrieves an io.ReadCloser for the content stored at "path" with a
// given byte offset.
func (d *driver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	if ok, err := blobRef.Exists(); err != nil {
		return nil, err
	} else if !ok {
		return nil, storagedriver.PathNotFoundError{Path: path}
	}

	err := blobRef.GetProperties(nil)
	if err != nil {
		return nil, err
	}
	info := blobRef.Properties
	size := info.ContentLength
	if offset >= size {
		return ioutil.NopCloser(bytes.NewReader(nil)), nil
	}

	resp, err := blobRef.GetRange(&azure.GetBlobRangeOptions{
		Range: &azure.BlobRange{
			Start: uint64(offset),
			End:   0,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Writer returns a FileWriter which will store the content written to it
// at the location designated by "path" after the call to Commit.
func (d *driver) Writer(ctx context.Context, path string, append bool) (storagedriver.FileWriter, error) {
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	blobExists, err := blobRef.Exists()
	if err != nil {
		return nil, err
	}
	var size int64
	if blobExists {
		if append {
			err = blobRef.GetProperties(nil)
			if err != nil {
				return nil, err
			}
			blobProperties := blobRef.Properties
			size = blobProperties.ContentLength
		} else {
			err = blobRef.Delete(nil)
			if err != nil {
				return nil, err
			}
		}
	} else {
		if append {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}
		err = blobRef.PutAppendBlob(nil)
		if err != nil {
			return nil, err
		}
	}

	return d.newWriter(d.pathToKey(path), size), nil
}

// Stat retrieves the FileInfo for the given path, including the current size
// in bytes and the creation time.
func (d *driver) Stat(ctx context.Context, path string) (storagedriver.FileInfo, error) {
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	// Check if the path is a blob
	if ok, err := blobRef.Exists(); err != nil {
		return nil, err
	} else if ok {
		err = blobRef.GetProperties(nil)
		if err != nil {
			return nil, err
		}
		blobProperties := blobRef.Properties

		return storagedriver.FileInfoInternal{FileInfoFields: storagedriver.FileInfoFields{
			Path:    path,
			Size:    blobProperties.ContentLength,
			ModTime: time.Time(blobProperties.LastModified),
			IsDir:   false,
		}}, nil
	}

	// Check if path is a virtual container
	containerRef := d.client.GetContainerReference(d.container)
	blobs, err := containerRef.ListBlobs(azure.ListBlobsParameters{
		Prefix:     d.pathToDirKey(path),
		MaxResults: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(blobs.Blobs) > 0 {
		// path is a virtual container
		return storagedriver.FileInfoInternal{FileInfoFields: storagedriver.FileInfoFields{
			Path:  path,
			IsDir: true,
		}}, nil
	}

	// path is not a blob or virtual container
	return nil, storagedriver.PathNotFoundError{Path: path}
}

// List returns a list of objects that are direct descendants of the given path.
func (d *driver) List(ctx context.Context, path string) ([]string, error) {
	prefix := d.pathToDirKey(path)

	// Remove inital slash to list from the root of the container.
	if path == "/" {
		prefix = strings.TrimPrefix(prefix, "/")
	}

	list, err := d.list(prefix)
	if err != nil {
		return nil, err
	}
	if path != "/" && len(list) == 0 {
		return nil, storagedriver.PathNotFoundError{Path: path}
	}

	return list, nil
}

// Move moves an object stored at sourcePath to destPath, removing the original
// object.
func (d *driver) Move(ctx context.Context, sourcePath string, destPath string) error {
	srcBlobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(sourcePath))
	sourceBlobURL := srcBlobRef.GetURL()
	destBlobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(destPath))
	err := destBlobRef.Copy(sourceBlobURL, nil)
	if err != nil {
		if is404(err) {
			return storagedriver.PathNotFoundError{Path: sourcePath}
		}
		return err
	}

	return srcBlobRef.Delete(nil)
}

// Delete recursively deletes all objects stored at "path" and its subpaths.
func (d *driver) Delete(ctx context.Context, path string) error {
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	ok, err := blobRef.DeleteIfExists(nil)
	if err != nil {
		return err
	}
	if ok {
		return nil // was a blob and deleted, return
	}

	// Not a blob, see if path is a virtual container with blobs
	blobs, err := d.listBlobs(d.pathToDirKey(path))
	if err != nil {
		return err
	}

	for _, b := range blobs {
		blobRef = d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(b))
		if err = blobRef.Delete(nil); err != nil {
			return err
		}
	}

	if len(blobs) == 0 {
		return storagedriver.PathNotFoundError{Path: path}
	}
	return nil
}

// DeleteFiles deletes a set of files by iterating over their full path list and invoking Delete for each. Returns the
// number of successfully deleted files and any errors. This method is idempotent, no error is returned if a file does
// not exist.
func (d *driver) DeleteFiles(ctx context.Context, paths []string) (int, error) {
	count := 0
	for _, path := range paths {
		if err := d.Delete(ctx, d.pathToKey(path)); err != nil {
			if _, ok := err.(storagedriver.PathNotFoundError); !ok {
				return count, err
			}
		}
		count++
	}
	return count, nil
}

// URLFor returns a publicly accessible URL for the blob stored at given path
// for specified duration by making use of Azure Storage Shared Access Signatures (SAS).
// See https://msdn.microsoft.com/en-us/library/azure/ee395415.aspx for more info.
func (d *driver) URLFor(ctx context.Context, path string, options map[string]interface{}) (string, error) {
	expiresTime := time.Now().UTC().Add(20 * time.Minute) // default expiration
	expires, ok := options["expiry"]
	if ok {
		t, ok := expires.(time.Time)
		if ok {
			expiresTime = t
		}
	}
	blobRef := d.client.GetContainerReference(d.container).GetBlobReference(d.pathToKey(path))
	return blobRef.GetSASURI(azure.BlobSASOptions{
		BlobServiceSASPermissions: azure.BlobServiceSASPermissions{
			Read: true,
		},
		SASOptions: azure.SASOptions{
			Expiry: expiresTime,
		},
	})
}

// Walk traverses a filesystem defined within driver, starting
// from the given path, calling f on each file
func (d *driver) Walk(ctx context.Context, path string, f storagedriver.WalkFn) error {
	return storagedriver.WalkFallback(ctx, d, d.pathToDirKey(path), f)
}

// WalkParallel traverses a filesystem defined within driver in parallel, starting
// from the given path, calling f on each file.
func (d *driver) WalkParallel(ctx context.Context, path string, f storagedriver.WalkFn) error {
	// TODO: Verify that this driver can reliably handle parallel workloads before
	// using storagedriver.WalkFallbackParallel
	return d.Walk(ctx, d.pathToDirKey(path), f)
}

func (d *driver) TransferTo(ctx context.Context, destDriver storagedriver.StorageDriver, src, dest string) error {
	return storagedriver.ErrUnsupportedMethod{}
}

// list simulates a filesystem style list in which both files (blobs) and
// directories (virtual containers) are returned for a given prefix.
func (d *driver) list(prefix string) ([]string, error) {
	return d.listWithDelimter(prefix, "/")
}

// listBlobs lists all blobs whose names begin with the specified prefix.
func (d *driver) listBlobs(prefix string) ([]string, error) {
	return d.listWithDelimter(prefix, "")
}

func (d *driver) listWithDelimter(prefix, delimiter string) ([]string, error) {
	out := []string{}
	marker := ""
	containerRef := d.client.GetContainerReference(d.container)
	for {
		resp, err := containerRef.ListBlobs(azure.ListBlobsParameters{
			Marker:    marker,
			Prefix:    prefix,
			Delimiter: delimiter,
		})

		if err != nil {
			return out, err
		}

		for _, b := range resp.Blobs {
			out = append(out, d.keyToPath(b.Name))
		}

		for _, p := range resp.BlobPrefixes {
			out = append(out, d.keyToPath(p))
		}

		if (len(resp.Blobs) == 0 && len(resp.BlobPrefixes) == 0) ||
			resp.NextMarker == "" {
			break
		}
		marker = resp.NextMarker
	}
	return out, nil
}

func is404(err error) bool {
	statusCodeErr, ok := err.(azure.AzureStorageServiceError)
	return ok && statusCodeErr.StatusCode == http.StatusNotFound
}

type writer struct {
	driver    *driver
	path      string
	size      int64
	bw        *bufio.Writer
	closed    bool
	committed bool
	canceled  bool
}

func (d *driver) newWriter(path string, size int64) storagedriver.FileWriter {
	return &writer{
		driver: d,
		path:   path,
		size:   size,
		bw: bufio.NewWriterSize(&blockWriter{
			client:    d.client,
			container: d.container,
			path:      path,
		}, maxChunkSize),
	}
}

func (w *writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("already closed")
	} else if w.committed {
		return 0, fmt.Errorf("already committed")
	} else if w.canceled {
		return 0, fmt.Errorf("already canceled")
	}

	n, err := w.bw.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *writer) Size() int64 {
	return w.size
}

func (w *writer) Close() error {
	if w.closed {
		return fmt.Errorf("already closed")
	}
	w.closed = true
	return w.bw.Flush()
}

func (w *writer) Cancel() error {
	if w.closed {
		return fmt.Errorf("already closed")
	} else if w.committed {
		return fmt.Errorf("already committed")
	}
	w.canceled = true
	blobRef := w.driver.client.GetContainerReference(w.driver.container).GetBlobReference(w.path)
	return blobRef.Delete(nil)
}

func (w *writer) Commit() error {
	if w.closed {
		return fmt.Errorf("already closed")
	} else if w.committed {
		return fmt.Errorf("already committed")
	} else if w.canceled {
		return fmt.Errorf("already canceled")
	}
	w.committed = true
	return w.bw.Flush()
}

type blockWriter struct {
	client    azure.BlobStorageClient
	container string
	path      string
}

func (bw *blockWriter) Write(p []byte) (int, error) {
	n := 0
	blobRef := bw.client.GetContainerReference(bw.container).GetBlobReference(bw.path)
	for offset := 0; offset < len(p); offset += maxChunkSize {
		chunkSize := maxChunkSize
		if offset+chunkSize > len(p) {
			chunkSize = len(p) - offset
		}
		err := blobRef.AppendBlock(p[offset:offset+chunkSize], nil)
		if err != nil {
			return n, err
		}

		n += chunkSize
	}

	return n, nil
}

func (d *driver) pathToKey(path string) string {
	p := strings.TrimSpace(strings.TrimRight(d.rootDirectory+strings.TrimLeft(path, "/"), "/"))

	// The Azure driver as it was originally released did not strip the leading
	// slash from directories, resulting in a a directory structure containing
	// an extra leading slash compared to other object storage drivers. For
	// example: `/<no-name>/docker/registry/v2`. We need to preserve this behavior
	// by default to support historical deployments of the registry using azure.
	if d.rootDirectory == "" {
		return "/" + p
	}

	return p
}

func (d *driver) pathToDirKey(path string) string {
	return d.pathToKey(path) + "/"
}

func (d *driver) keyToPath(key string) string {
	return "/" + strings.Trim(strings.TrimPrefix(key, d.rootDirectory), "/")
}
