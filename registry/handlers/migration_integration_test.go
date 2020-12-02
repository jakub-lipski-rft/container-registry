// +build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

// This file is intended to test the HTTP API under a migration scenario, using the proxy feature. Fine grain tests of
// the handlers internal behaviour (e.g., Schema 1 support, content negotiation, etc.) are out of scope, as it's already
// heavily tested in api_integration_test.go. Similarly, fine grain tests of the migration wrapper internal behaviour
// (e.g., validating all request and response headers) are also out of scope. These tests can be found in
// migration_test.go. Here we're mainly concerned with ensuring that all HTTP handlers and methods are properly wrapped
// by the handlers.migrationHandler and that it doesn't affect the registry public API.

func withMigrationProxy(url string) configOpt {
	return func(config *configuration.Configuration) {
		config.Migration.Proxy.Enabled = true
		config.Migration.Proxy.URL = url
	}
}

func withHTTPHost(host string) configOpt {
	return func(config *configuration.Configuration) {
		config.HTTP.Host = host
	}
}

func TestMigrationBlobAPI_Get_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a blob in proxy registry
	args, blobURL := createRepoWithBlob(t, envProxy)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// fetch blob from proxy registry
	validateBlobGet(t, blobURL, args)
}

func TestMigrationBlobAPI_Get_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a blob in target registry
	args, blobURL := createRepoWithBlob(t, envTarget)
	// fetch blob through proxy registry
	u := overrideHostInURL(t, blobURL, envProxy.server.URL)
	validateBlobGet(t, u, args)
}

// We don't need to test this scenario for every single handler and method, cause it works the same for others (it's
// covered by the tests in migration_test.go, but keeping one here regardless).
func TestMigrationBlobAPI_Get_FromTargetRegistryFailsIfDown(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a blob in target registry
	_, blobURL := createRepoWithBlob(t, envTarget)
	// fetch blob through proxy registry
	envTarget.Shutdown()
	u := overrideHostInURL(t, blobURL, envProxy.server.URL)
	res, err := http.Get(u)
	require.NoError(t, err)
	require.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}

func TestMigrationBlobAPI_Head_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a blob in proxy registry
	args, blobURL := createRepoWithBlob(t, envProxy)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// head blob in proxy registry
	validateBlobHead(t, blobURL, args)
}

func TestMigrationBlobAPI_Head_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a blob in target registry
	args, blobURL := createRepoWithBlob(t, envTarget)
	validateBlobGet(t, blobURL, args)
	// head blob through proxy registry
	u := overrideHostInURL(t, blobURL, envProxy.server.URL)
	validateBlobHead(t, u, args)
}

func TestMigrationBlobAPI_Delete_InProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t, withDelete)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a blob in proxy registry
	_, blobURL := createRepoWithBlob(t, envProxy)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// delete blob in proxy registry
	validateBlobDelete(t, blobURL)
}

func TestMigrationBlobAPI_Delete_InTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t, withDelete)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a blob in target registry
	_, blobURL := createRepoWithBlob(t, envTarget)
	// delete blob through proxy registry
	u := overrideHostInURL(t, blobURL, envProxy.server.URL)
	validateBlobDelete(t, u)
}

func TestMigrationBlobUploadAPI_Get_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// start blob upload in proxy registry
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envProxy, args.imageName)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// check status of upload in proxy registry
	validateBlobUploadGet(t, uploadURL, uploadUUID)
}

func TestMigrationBlobUploadAPI_Get_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// start blob upload in target registry
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envTarget, args.imageName)
	// check status of upload through proxy registry
	u := overrideHostInURL(t, uploadURL, envProxy.server.URL)
	validateBlobUploadGet(t, u, uploadUUID)
}

func TestMigrationBlobUploadAPI_Head_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// start blob upload in proxy registry
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envProxy, args.imageName)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// check status of upload in proxy registry
	validateBlobUploadHead(t, uploadURL, uploadUUID)
}

func TestMigrationBlobUploadAPI_Head_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// start blob upload in target registry
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envTarget, args.imageName)
	// check status of upload through proxy registry
	u := overrideHostInURL(t, uploadURL, envProxy.server.URL)
	validateBlobUploadHead(t, u, uploadUUID)
}

func TestMigrationBlobUploadAPI_Post_InTargetRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t, withHTTPHost(envProxy.server.URL))
	defer envTarget.Shutdown()
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)

	// start blob upload in target registry through the proxy one
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envProxy, args.imageName)
	// check that Location is using the proxy registry's hostname
	validateURLHostMatch(t, uploadURL, envProxy.server.URL)
	// check status of upload in target registry
	tmp := overrideHostInURL(t, uploadURL, envTarget.server.URL)
	validateBlobUploadHead(t, tmp, uploadUUID)
	// check status of upload through proxy registry
	validateBlobUploadHead(t, uploadURL, uploadUUID)
}

func TestMigrationBlobUploadAPI_Patch_InProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// upload blob in chunks to proxy registry
	args := makeBlobArgs(t)
	blobURL := chunkedBlobUpload(t, args, envProxy)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// head blob in proxy registry
	validateBlobHead(t, blobURL, args)
}

func TestMigrationBlobUploadAPI_Patch_InTargetRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t, withHTTPHost(envProxy.server.URL))
	defer envTarget.Shutdown()
	withMigrationProxy(envTarget.server.URL)(envProxy.config)

	// upload blob in chunks to target registry through the proxy one
	args := makeBlobArgs(t)
	blobURL := chunkedBlobUpload(t, args, envProxy)
	// check that Location is using the proxy registry's hostname
	validateURLHostMatch(t, blobURL, envProxy.server.URL)
	// head blob from target registry
	tmp := overrideHostInURL(t, blobURL, envTarget.server.URL)
	validateBlobHead(t, tmp, args)
	// head blob through proxy registry
	validateBlobHead(t, blobURL, args)
}

func TestMigrationBlobUploadAPI_Put_InTargetRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t, withHTTPHost(envProxy.server.URL))
	defer envTarget.Shutdown()
	withMigrationProxy(envTarget.server.URL)(envProxy.config)

	// push blob to target registry through the proxy one
	args := makeBlobArgs(t)
	uploadURL, _ := startPushLayer(t, envProxy, args.imageName)
	blobURL := pushLayer(t, envProxy.builder, args.imageName, args.layerDigest, uploadURL, args.layerFile)
	// check that Location is using the proxy registry's hostname
	validateURLHostMatch(t, blobURL, envProxy.server.URL)
	// head blob in target registry
	tmp := overrideHostInURL(t, blobURL, envTarget.server.URL)
	validateBlobHead(t, tmp, args)
	// head blob through proxy registry
	validateBlobHead(t, blobURL, args)
}

func TestMigrationBlobUploadAPI_Delete_InProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// start blob upload in proxy registry
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envProxy, args.imageName)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	//delete upload in proxy registry
	validateBlobUploadDelete(t, uploadURL, uploadUUID)
}

func TestMigrationBlobUploadAPI_Delete_InTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// start blob upload in target registry
	args := makeBlobArgs(t)
	uploadURL, uploadUUID := startPushLayer(t, envTarget, args.imageName)
	// delete upload through proxy registry
	u := overrideHostInURL(t, uploadURL, envProxy.server.URL)
	validateBlobUploadDelete(t, u, uploadUUID)
}

func TestMigrationManifestAPI_Get_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a manifest in proxy registry
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envProxy, repo, putByDigest)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// fetch manifest from proxy registry
	validateSchema2ManifestGet(t, envProxy, repo, m)
}

func TestMigrationManifestAPI_Get_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a manifest in target registry
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envTarget, repo, putByDigest)
	// fetch manifest through proxy registry
	validateSchema2ManifestGet(t, envProxy, repo, m)
}

func TestMigrationManifestAPI_Head_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a manifest in proxy registry
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envProxy, repo, putByDigest)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// head manifest from proxy registry
	validateSchema2ManifestHead(t, envProxy, repo, m)
}

func TestMigrationManifestAPI_Head_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a manifest in target registry
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envTarget, repo, putByDigest)
	// head manifest through proxy registry
	validateSchema2ManifestHead(t, envProxy, repo, m)
}

func TestMigrationManifestAPI_Put_InProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a manifest in proxy registry
	repo := "test/repo"
	seedRandomSchema2Manifest(t, envProxy, repo, putByDigest)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// create another manifest in proxy registry
	m := seedRandomSchema2Manifest(t, envProxy, repo, putByDigest)
	// head manifest from proxy registry
	validateSchema2ManifestGet(t, envProxy, repo, m)
}

func TestMigrationManifestAPI_Put_InTargetRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	// note that we're telling the target registry to advertise itself with the proxy registry hostname for Location headers
	envTarget := newTestEnv(t, withHTTPHost(envProxy.server.URL))
	defer envTarget.Shutdown()

	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// create repo with a manifest in target registry through the proxy one
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envProxy, repo, putByDigest)
	// check that manifest exists in target registry
	validateSchema2ManifestGet(t, envTarget, repo, m)
}

func TestMigrationManifestAPI_Delete_InProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t, withDelete)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a manifest in proxy registry
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envProxy, repo, putByDigest)
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// delete manifest in proxy registry
	validateSchema2ManifestDelete(t, envProxy, repo, m)
}

func TestMigrationManifestAPI_Delete_InTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t, withDelete)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a manifest in target registry through the proxy one
	repo := "test/repo"
	m := seedRandomSchema2Manifest(t, envTarget, repo, putByTag("latest"))
	// delete manifest in target registry through the proxy one
	validateSchema2ManifestDelete(t, envProxy, repo, m)
}

func TestMigrationTagsAPI_Get_FromProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a manifest and a tag in proxy registry
	repo := "test/repo"
	tag := "latest"
	seedRandomSchema2Manifest(t, envProxy, repo, putByTag(tag))
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// get tags from proxy registry
	validateTagsGet(t, repo, envProxy, []string{tag})
}

func TestMigrationTagsAPI_Get_FromTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a manifest and a tag in target registry
	repo := "test/repo"
	tag := "latest"
	seedRandomSchema2Manifest(t, envTarget, repo, putByTag(tag))
	// get tags through proxy registry
	validateTagsGet(t, repo, envProxy, []string{tag})
}

func TestMigrationTagAPI_Delete_InProxyRegistry(t *testing.T) {
	envProxy := newTestEnv(t)
	defer envProxy.Shutdown()
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()

	// create repo with a manifest and a tag in proxy registry
	repo := "test/repo"
	tag := "latest"
	seedRandomSchema2Manifest(t, envProxy, repo, putByTag(tag))
	// reconfigure proxy registry to proxy to target registry
	withMigrationProxy(envTarget.server.URL)(envProxy.config)
	// delete tag in proxy registry
	validateTagDelete(t, repo, envProxy, tag)
}

func TestMigrationTagAPI_Delete_InTargetRegistry(t *testing.T) {
	envTarget := newTestEnv(t)
	defer envTarget.Shutdown()
	envProxy := newTestEnv(t, withMigrationProxy(envTarget.server.URL))
	defer envProxy.Shutdown()

	// create repo with a manifest and a tag in target registry
	repo := "test/repo"
	tag := "latest"
	seedRandomSchema2Manifest(t, envTarget, repo, putByTag(tag))
	// delete tag through proxy registry
	validateTagDelete(t, repo, envProxy, tag)
}

func createRepoWithBlob(t *testing.T, env *testEnv) (blobArgs, string) {
	t.Helper()

	args := makeBlobArgs(t)
	uploadURLBase, _ := startPushLayer(t, env, args.imageName)
	blobURL := pushLayer(t, env.builder, args.imageName, args.layerDigest, uploadURLBase, args.layerFile)

	return args, blobURL
}

func overrideHostInURL(t *testing.T, url1, url2 string) string {
	t.Helper()

	u1, err := url.Parse(url1)
	require.NoError(t, err)
	u2, err := url.Parse(url2)
	require.NoError(t, err)
	u1.Host = u2.Host

	return u1.String()
}

func validateURLHostMatch(t *testing.T, url1, url2 string) {
	t.Helper()

	u1, err := url.Parse(url1)
	require.NoError(t, err)
	u2, err := url.Parse(url2)
	require.NoError(t, err)
	require.Equal(t, u1.Host, u2.Host)
}

func validateSchema2ManifestDelete(t *testing.T, envProxy *testEnv, repo string, m *schema2.DeserializedManifest) {
	t.Helper()

	manifestURL := buildManifestDigestURL(t, envProxy, repo, m)
	resp, err := httpDelete(manifestURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.Equal(t, http.NoBody, resp.Body)
}

func validateTagsGet(t *testing.T, repo string, env *testEnv, tags []string) {
	t.Helper()

	imageName, err := reference.WithName(repo)
	require.NoError(t, err)
	tagsURL, err := env.builder.BuildTagsURL(imageName)
	require.NoError(t, err)

	resp, err := http.Get(tagsURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body tagsAPIResponse
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&body)
	require.NoError(t, err)
	require.Equal(t, tagsAPIResponse{Name: repo, Tags: tags}, body)
}

func validateTagDelete(t *testing.T, repo string, env *testEnv, tag string) {
	t.Helper()

	imageName, err := reference.WithName(repo)
	require.NoError(t, err)
	ref, err := reference.WithTag(imageName, tag)
	require.NoError(t, err)
	tagURL, err := env.builder.BuildTagURL(ref)
	require.NoError(t, err)

	resp, err := httpDelete(tagURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.Equal(t, http.NoBody, resp.Body)
}

func validateSchema2ManifestGet(t *testing.T, envProxy *testEnv, repo string, m *schema2.DeserializedManifest) {
	t.Helper()

	manifestURL := buildManifestDigestURL(t, envProxy, repo, m)
	_, payload, err := m.Payload()
	require.NoError(t, err)
	dgst := digest.FromBytes(payload)

	req, err := http.NewRequest("GET", manifestURL, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", schema2.MediaTypeManifest)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	validateManifestResponseHeaders(t, resp, dgst)
	var fetchedManifest *schema2.DeserializedManifest
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&fetchedManifest)
	require.NoError(t, err)
	require.EqualValues(t, m, fetchedManifest)
}

func validateSchema2ManifestHead(t *testing.T, envProxy *testEnv, repo string, m *schema2.DeserializedManifest) {
	t.Helper()

	manifestURL := buildManifestDigestURL(t, envProxy, repo, m)
	_, payload, err := m.Payload()
	require.NoError(t, err)
	dgst := digest.FromBytes(payload)

	req, err := http.NewRequest("HEAD", manifestURL, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", schema2.MediaTypeManifest)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	validateManifestResponseHeaders(t, resp, dgst)
}

func validateManifestResponseHeaders(t *testing.T, resp *http.Response, dgst digest.Digest) {
	t.Helper()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, dgst.String(), resp.Header.Get("Docker-Content-Digest"))
	require.Equal(t, fmt.Sprintf(`"%s"`, dgst), resp.Header.Get("ETag"))
}

func validateBlobGet(t *testing.T, blobURL string, args blobArgs) {
	t.Helper()

	res, err := http.Get(blobURL)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.StatusCode)
	validateBlobResponseHeaders(t, args, res)
	v := args.layerDigest.Verifier()
	_, err = io.Copy(v, res.Body)
	require.NoError(t, err)
	require.True(t, v.Verified())
}

func validateBlobHead(t *testing.T, blobURL string, args blobArgs) {
	t.Helper()

	res, err := http.Head(blobURL)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.StatusCode)
	validateBlobResponseHeaders(t, args, res)
	require.Equal(t, http.NoBody, res.Body)
}

func validateBlobResponseHeaders(t *testing.T, args blobArgs, res *http.Response) {
	t.Helper()

	_, err := args.layerFile.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(args.layerFile)
	require.NoError(t, err)

	require.Equal(t, res.Header.Get("Content-Length"), strconv.Itoa(buf.Len()))
	require.Equal(t, res.Header.Get("Content-Type"), "application/octet-stream")
	require.Equal(t, res.Header.Get("Docker-Content-Digest"), args.layerDigest.String())
	require.Equal(t, res.Header.Get("ETag"), fmt.Sprintf(`"%s"`, args.layerDigest))
	require.Equal(t, res.Header.Get("Cache-Control"), "max-age=31536000")
}

func validateBlobDelete(t *testing.T, location string) {
	t.Helper()

	res, err := httpDelete(location)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, res.StatusCode)
	require.Equal(t, http.NoBody, res.Body)

	res, err = http.Head(location)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.Equal(t, http.NoBody, res.Body)
}

func validateBlobUploadHeaders(t *testing.T, uploadUUID string, res *http.Response) {
	t.Helper()

	require.Equal(t, http.StatusNoContent, res.StatusCode)
	checkHeaders(t, res, http.Header{
		"Location":           []string{"*"},
		"Range":              []string{"0-0"},
		"Docker-Upload-UUID": []string{uploadUUID},
	})
	require.Equal(t, http.NoBody, res.Body)
}

func validateBlobUploadGet(t *testing.T, uploadURL string, uploadUUID string) {
	t.Helper()

	res, err := http.Get(uploadURL)
	require.NoError(t, err)
	validateBlobUploadHeaders(t, uploadUUID, res)
}

func validateBlobUploadHead(t *testing.T, uploadURL string, uploadUUID string) {
	t.Helper()

	res, err := http.Get(uploadURL)
	require.NoError(t, err)
	validateBlobUploadHeaders(t, uploadUUID, res)
}

func validateBlobUploadDelete(t *testing.T, uploadURL string, uploadUUID string) {
	t.Helper()

	resp, err := httpDelete(uploadURL)
	require.NoError(t, err)

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	checkHeaders(t, resp, http.Header{
		"Docker-Upload-UUID": []string{uploadUUID},
	})
	require.Equal(t, http.NoBody, resp.Body)
}

func chunkedBlobUpload(t *testing.T, args blobArgs, envProxy *testEnv) string {
	t.Helper()

	layerLength, err := args.layerFile.Seek(0, io.SeekEnd)
	require.NoError(t, err)
	_, err = args.layerFile.Seek(0, io.SeekStart)
	require.NoError(t, err)
	uploadURL, _ := startPushLayer(t, envProxy, args.imageName)
	uploadURL, dgst := pushChunk(t, envProxy.builder, args.imageName, uploadURL, args.layerFile, layerLength)

	return finishUpload(t, envProxy.builder, args.imageName, uploadURL, dgst)
}
