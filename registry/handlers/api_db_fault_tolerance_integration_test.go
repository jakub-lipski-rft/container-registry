// +build integration
// +build toxiproxy

package handlers_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	toxiclient "github.com/Shopify/toxiproxy/client"
	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

// This file is intended to test the HTTP API fault tolerance under adverse network conditions related to the metadata
// database, using Shopify Toxiproxy as intermediary between the registry and its DB. Fine grain tests of the handlers
// internal behaviour (e.g., Schema 1 support, content negotiation, etc.) are out of scope. Here we're mainly concerned
// with ensuring that all HTTP handlers and methods are handling failure scenarios properly.

var (
	toxiClient    *toxiclient.Client
	toxiproxyHost string
)

func init() {
	toxiproxyHost = os.Getenv("TOXIPROXY_HOST")
	if toxiproxyHost == "" {
		panic("TOXIPROXY_HOST environment variable not set")
	}
	port := os.Getenv("TOXIPROXY_PORT")
	if port == "" {
		panic("TOXIPROXY_PORT environment variable not set")
	}
	toxiClient = toxiclient.NewClient(net.JoinHostPort(toxiproxyHost, port))
	if err := toxiClient.ResetState(); err != nil {
		panic(fmt.Errorf("failed to reset toxiproxy: %w", err))
	}
}

type dbProxy struct {
	proxy *toxiclient.Proxy
	t     *testing.T
}

func (p dbProxy) HostAndPort() (string, int) {
	p.t.Helper()

	_, portStr, err := net.SplitHostPort(p.proxy.Listen)
	require.NoError(p.t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(p.t, err)

	return toxiproxyHost, port
}

func (p dbProxy) Enable() {
	p.t.Helper()
	require.NoError(p.t, p.proxy.Enable())
}

func (p dbProxy) Disable() {
	p.t.Helper()
	require.NoError(p.t, p.proxy.Disable())
}

func (p dbProxy) Delete() {
	p.t.Helper()
	require.NoError(p.t, p.proxy.Delete())
}

func newDBProxy(t *testing.T) *dbProxy {
	t.Helper()

	dsn, err := testutil.NewDSNFromEnv()
	require.NoError(t, err)
	p, err := toxiClient.CreateProxy("db", "", dsn.Address())
	require.NoError(t, err)

	return &dbProxy{p, t}
}

func TestDBFaultTolerance_ConnectionRefused_Catalog(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	u, err := env.builder.BuildCatalogURL()
	require.NoError(t, err)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertGetResponse(t, u, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertGetResponse(t, u, http.StatusOK)
}

func TestDBFaultTolerance_ConnectionRefused_TagList(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	name, err := reference.WithName(repoName)
	require.NoError(t, err)
	u, err := env.builder.BuildTagsURL(name)
	require.NoError(t, err)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertGetResponse(t, u, http.StatusServiceUnavailable)
	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertGetResponse(t, u, http.StatusNotFound)
}

func TestDBFaultTolerance_ConnectionRefused_TagDelete(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withSchema1Compatibility, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	tagName := "latest"

	// query API with proxy disabled, should fail
	// create the repo, otherwise the request will halt on the filesystem search, which precedes the DB search
	createRepository(env, t, repoName, tagName)
	dbProxy.Disable()
	assertTagDeleteResponse(t, env, repoName, tagName, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	createRepository(env, t, repoName, tagName)
	assertTagDeleteResponse(t, env, repoName, tagName, http.StatusAccepted)
}

func TestDBFaultTolerance_ConnectionRefused_BlobGet(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	// we can use a non-existing repo and blob, as reads are executed against the DB first
	repoName := "foo"
	dgst := digest.FromString(repoName)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertBlobGetResponse(t, env, repoName, dgst, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertBlobGetResponse(t, env, repoName, dgst, http.StatusNotFound)
}

func TestDBFaultTolerance_ConnectionRefused_BlobHead(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	// we can use a non-existing repo and blob, as reads are executed against the DB first
	repoName := "foo"
	dgst := digest.FromString(repoName)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertBlobHeadResponse(t, env, repoName, dgst, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertBlobHeadResponse(t, env, repoName, dgst, http.StatusNotFound)
}

func TestDBFaultTolerance_ConnectionRefused_BlobDelete(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withDelete, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	// query API with proxy disabled, should fail
	// create the repo and blob, otherwise the request will halt on the filesystem search, which precedes the DB search
	args, _ := createRepoWithBlob(t, env)
	dbProxy.Disable()
	assertBlobDeleteResponse(t, env, args.imageName.String(), args.layerDigest, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	args, _ = createRepoWithBlob(t, env)
	assertBlobDeleteResponse(t, env, args.imageName.String(), args.layerDigest, http.StatusAccepted)
}

func TestDBFaultTolerance_ConnectionRefused_BlobPut(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	args := makeBlobArgs(t)
	assertBlobPutResponse(t, env, args.imageName.String(), args.layerDigest, args.layerFile, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	args = makeBlobArgs(t)
	assertBlobPutResponse(t, env, args.imageName.String(), args.layerDigest, args.layerFile, http.StatusCreated)
}

func TestDBFaultTolerance_ConnectionRefused_BlobPostMount(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	args, _ := createRepoWithBlob(t, env)
	destRepo := "foo"

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertBlobPostMountResponse(t, env, args.imageName.String(), destRepo, args.layerDigest, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertBlobPostMountResponse(t, env, args.imageName.String(), destRepo, args.layerDigest, http.StatusCreated)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestGetByDigest(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	m := seedRandomSchema2Manifest(t, env, repoName, putByDigest)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestGetByDigestResponse(t, env, repoName, m, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertManifestGetByDigestResponse(t, env, repoName, m, http.StatusOK)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestGetByTag(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "test/repo"
	tagName := "latest"
	seedRandomSchema2Manifest(t, env, repoName, putByTag(tagName))

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestGetByTagResponse(t, env, repoName, tagName, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertManifestGetByTagResponse(t, env, repoName, tagName, http.StatusOK)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestHeadByDigest(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	m := seedRandomSchema2Manifest(t, env, repoName, putByDigest)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestHeadByDigestResponse(t, env, repoName, m, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertManifestHeadByDigestResponse(t, env, repoName, m, http.StatusOK)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestHeadByTag(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "test/repo"
	tagName := "latest"
	seedRandomSchema2Manifest(t, env, repoName, putByTag(tagName))

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestHeadByTagResponse(t, env, repoName, tagName, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertManifestHeadByTagResponse(t, env, repoName, tagName, http.StatusOK)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestPutByDigest(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	m := seedRandomSchema2Manifest(t, env, repoName, putByDigest)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestPutByDigestResponse(t, env, repoName, m, m.MediaType, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertManifestPutByDigestResponse(t, env, repoName, m, m.MediaType, http.StatusCreated)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestPutByTag(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	tagName := "latest"
	m := seedRandomSchema2Manifest(t, env, repoName, putByTag(tagName))

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestPutByTagResponse(t, env, repoName, m, m.MediaType, tagName, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	assertManifestPutByTagResponse(t, env, repoName, m, m.MediaType, tagName, http.StatusCreated)
}

func TestDBFaultTolerance_ConnectionRefused_ManifestDelete(t *testing.T) {
	dbProxy := newDBProxy(t)
	defer dbProxy.Delete()

	env := newTestEnv(t, withDelete, withCustomDBHostAndPort(dbProxy.HostAndPort()))
	defer env.Shutdown()

	repoName := "foo"
	m := seedRandomSchema2Manifest(t, env, repoName, putByDigest)

	// query API with proxy disabled, should fail
	dbProxy.Disable()
	assertManifestDeleteResponse(t, env, repoName, m, http.StatusServiceUnavailable)

	// query API with proxy re-enabled, should succeed
	dbProxy.Enable()
	m = seedRandomSchema2Manifest(t, env, repoName, putByDigest)
	assertManifestDeleteResponse(t, env, repoName, m, http.StatusAccepted)
}

func assertGetResponse(t *testing.T, url string, expectedStatus int) {
	t.Helper()

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)
}

func assertHeadResponse(t *testing.T, url string, expectedStatus int) {
	t.Helper()

	resp, err := http.Head(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)
}

func assertPutResponse(t *testing.T, url string, body io.Reader, headers http.Header, expectedStatus int) {
	t.Helper()

	req, err := http.NewRequest("PUT", url, body)
	require.NoError(t, err)
	for k, vv := range headers {
		req.Header.Set(k, strings.Join(vv, ","))
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)
}

func assertPostResponse(t *testing.T, url string, body io.Reader, headers http.Header, expectedStatus int) {
	t.Helper()

	req, err := http.NewRequest("POST", url, body)
	require.NoError(t, err)
	for k, vv := range headers {
		req.Header.Set(k, strings.Join(vv, ","))
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)
}

func assertDeleteResponse(t *testing.T, url string, expectedStatus int) {
	t.Helper()

	resp, err := httpDelete(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)
}

func assertTagDeleteResponse(t *testing.T, env *testEnv, repoName, tagName string, expectedStatus int) {
	t.Helper()

	tmp, err := reference.WithName(repoName)
	require.NoError(t, err)
	named, err := reference.WithTag(tmp, tagName)
	require.NoError(t, err)
	u, err := env.builder.BuildTagURL(named)
	require.NoError(t, err)

	assertDeleteResponse(t, u, expectedStatus)
}

func assertBlobGetResponse(t *testing.T, env *testEnv, repoName string, dgst digest.Digest, expectedStatus int) {
	t.Helper()

	tmp, err := reference.WithName(repoName)
	require.NoError(t, err)
	ref, err := reference.WithDigest(tmp, dgst)
	require.NoError(t, err)
	u, err := env.builder.BuildBlobURL(ref)
	require.NoError(t, err)

	assertGetResponse(t, u, expectedStatus)
}

func assertBlobHeadResponse(t *testing.T, env *testEnv, repoName string, dgst digest.Digest, expectedStatus int) {
	t.Helper()

	tmp, err := reference.WithName(repoName)
	require.NoError(t, err)
	ref, err := reference.WithDigest(tmp, dgst)
	require.NoError(t, err)
	u, err := env.builder.BuildBlobURL(ref)
	require.NoError(t, err)

	assertHeadResponse(t, u, expectedStatus)
}

func assertBlobDeleteResponse(t *testing.T, env *testEnv, repoName string, dgst digest.Digest, expectedStatus int) {
	t.Helper()

	tmp, err := reference.WithName(repoName)
	require.NoError(t, err)
	ref, err := reference.WithDigest(tmp, dgst)
	require.NoError(t, err)
	u, err := env.builder.BuildBlobURL(ref)
	require.NoError(t, err)

	assertDeleteResponse(t, u, expectedStatus)
}

func assertBlobPutResponse(t *testing.T, env *testEnv, repoName string, dgst digest.Digest, body io.ReadSeeker, expectedStatus int) {
	t.Helper()

	name, err := reference.WithName(repoName)
	require.NoError(t, err)

	baseURL, _ := startPushLayer(t, env, name)
	u, err := url.Parse(baseURL)
	require.NoError(t, err)
	u.RawQuery = url.Values{
		"_state": u.Query()["_state"],
		"digest": []string{dgst.String()},
	}.Encode()

	assertPutResponse(t, u.String(), body, nil, expectedStatus)
}

func assertBlobPostMountResponse(t *testing.T, env *testEnv, srcRepoName, destRepoName string, dgst digest.Digest, expectedStatus int) {
	t.Helper()

	name, err := reference.WithName(destRepoName)
	require.NoError(t, err)
	u, err := env.builder.BuildBlobUploadURL(name, url.Values{
		"mount": []string{dgst.String()},
		"from":  []string{srcRepoName},
	})
	require.NoError(t, err)

	assertPostResponse(t, u, nil, nil, expectedStatus)
}

func assertManifestGetByDigestResponse(t *testing.T, env *testEnv, repoName string, m distribution.Manifest, expectedStatus int) {
	t.Helper()

	u := buildManifestDigestURL(t, env, repoName, m)
	assertGetResponse(t, u, expectedStatus)
}

func assertManifestGetByTagResponse(t *testing.T, env *testEnv, repoName, tagName string, expectedStatus int) {
	t.Helper()

	u := buildManifestTagURL(t, env, repoName, tagName)
	assertGetResponse(t, u, expectedStatus)
}

func assertManifestHeadByDigestResponse(t *testing.T, env *testEnv, repoName string, m distribution.Manifest, expectedStatus int) {
	t.Helper()

	u := buildManifestDigestURL(t, env, repoName, m)
	assertHeadResponse(t, u, expectedStatus)
}

func assertManifestHeadByTagResponse(t *testing.T, env *testEnv, repoName, tagName string, expectedStatus int) {
	t.Helper()

	u := buildManifestTagURL(t, env, repoName, tagName)
	assertHeadResponse(t, u, expectedStatus)
}

func assertManifestPutByDigestResponse(t *testing.T, env *testEnv, repoName string, m distribution.Manifest, mediaType string, expectedStatus int) {
	t.Helper()

	u := buildManifestDigestURL(t, env, repoName, m)
	_, body, err := m.Payload()
	require.NoError(t, err)

	assertPutResponse(t, u, bytes.NewReader(body), http.Header{"Content-Type": []string{mediaType}}, expectedStatus)
}

func assertManifestPutByTagResponse(t *testing.T, env *testEnv, repoName string, m distribution.Manifest, mediaType, tagName string, expectedStatus int) {
	t.Helper()

	u := buildManifestTagURL(t, env, repoName, tagName)
	_, body, err := m.Payload()
	require.NoError(t, err)

	assertPutResponse(t, u, bytes.NewReader(body), http.Header{"Content-Type": []string{mediaType}}, expectedStatus)
}

func assertManifestDeleteResponse(t *testing.T, env *testEnv, repoName string, m distribution.Manifest, expectedStatus int) {
	t.Helper()

	u := buildManifestDigestURL(t, env, repoName, m)
	assertDeleteResponse(t, u, expectedStatus)
}
