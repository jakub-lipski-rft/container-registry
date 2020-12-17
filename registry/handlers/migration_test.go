package handlers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/docker/distribution/testutil"
	"github.com/stretchr/testify/require"
)

type handlerMock struct {
	validatorFn http.HandlerFunc
	response    string
}

func (h *handlerMock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if h.validatorFn != nil {
		h.validatorFn(rw, req)
	}

	rw.Write([]byte(h.response))
	rw.WriteHeader(http.StatusOK)
}

func TestMigrationWrapper_DoesNotWrapByDefault(t *testing.T) {
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}

	app := NewApp(context.Background(), config)
	c := &Context{App: app}

	expected := &handlerMock{}
	got := migrationWrapper(c, expected)

	// the input handler should be returned
	require.Equal(t, expected, got)
}

func TestMigrationWrapper_WrapsIfMigrationProxyEnabled(t *testing.T) {
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}
	config.Migration.Proxy.Enabled = true

	app := NewApp(context.Background(), config)
	ctx := &Context{App: app}

	got := migrationWrapper(ctx, &handlerMock{})

	// instead of the input handler we should receive an http.HandlerFunc
	require.IsType(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), got)
}

func TestProxyNewRepositories_ProxiesRequestsForNewRepos(t *testing.T) {
	// create fake target registry server
	targetHandler := &handlerMock{response: "hello from target registry"}
	targetServer := httptest.NewServer(targetHandler)
	defer targetServer.Close()

	// create test app
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}
	config.Migration.Proxy.Enabled = true
	config.Migration.Proxy.URL = targetServer.URL
	config.HTTP.Headers = http.Header{
		"foo": []string{"a"},
		"bar": []string{"b"},
	}
	app := NewApp(context.Background(), config)

	// target non-existing repository
	named, err := reference.WithName("test/repo")
	require.NoError(t, err)
	repo, err := app.registry.Repository(context.Background(), named)
	require.NoError(t, err)

	ctx := &Context{
		App:        app,
		Repository: repo,
		Context:    context.Background(),
	}

	// create test request and response
	req := httptest.NewRequest("GET", "http://old-registry.example.com/some/path", nil)
	reqBkp := req.Clone(context.Background())

	res := httptest.NewRecorder()
	res.Header().Add("Docker-Distribution-API-Version", "registry/2.0")
	for k, v := range config.HTTP.Headers {
		res.Header().Add(k, strings.Join(v, ","))
	}

	// validate request on the target registry side
	targetHandler.validatorFn = func(rw http.ResponseWriter, req *http.Request) {
		// validate that request Host is set to the target registry host
		u, err := url.Parse(targetServer.URL)
		require.NoError(t, err)
		require.Equal(t, u.Host, req.Host)

		// validate that X-Forwarded-* headers were added to the request
		remoteHost, _, err := net.SplitHostPort(reqBkp.RemoteAddr)
		require.NoError(t, err)
		require.Equal(t, remoteHost, req.Header.Get("X-Forwarded-For"))
		require.Equal(t, reqBkp.Header.Get("Host"), req.Header.Get("X-Forwarded-Host"))

		// validate that custom headers are removed from response writer
		require.Empty(t, res.Header().Get("Docker-Distribution-API-Version"))
		for k, _ := range config.HTTP.Headers {
			require.Empty(t, rw.Header().Get(k))
		}
	}

	// test handler
	h := migrationHandler{Context: ctx, fallback: &handlerMock{}}
	h.proxyNewRepositories(res, req)

	// validate that request is proxied to target registry
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, targetHandler.response, res.Body.String())
}

func TestProxyNewRepositories_DoesNotProxyRequestsForExistingRepos(t *testing.T) {
	// create fake target registry server
	targetHandler := &handlerMock{response: "hello from target registry"}
	targetServer := httptest.NewServer(targetHandler)
	defer targetServer.Close()

	// create test app
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}
	config.Migration.Proxy.Enabled = true
	config.Migration.Proxy.URL = targetServer.URL

	app := NewApp(context.Background(), config)

	// target existing repository
	named, err := reference.WithName("test/repo")
	require.NoError(t, err)
	repo, err := app.registry.Repository(context.Background(), named)
	require.NoError(t, err)

	// upload a blob to test repo (will create the repository path)
	ll, err := testutil.CreateRandomLayers(1)
	require.NoError(t, err)
	err = testutil.UploadBlobs(repo, ll)
	require.NoError(t, err)

	ctx := &Context{
		App:        app,
		Repository: repo,
		Context:    context.Background(),
	}

	// create test request and response
	req := httptest.NewRequest("GET", "http://old-registry.example.com/some/path", nil)
	res := httptest.NewRecorder()
	res.Header().Add("Docker-Distribution-API-Version", "registry/2.0")
	for k, v := range config.HTTP.Headers {
		res.Header().Add(k, strings.Join(v, ","))
	}

	// create fake proxy registry handler
	proxyHandler := &handlerMock{response: "hello from proxy registry"}

	// make sure it doesn't reach the target registry
	targetHandler.validatorFn = func(rw http.ResponseWriter, req *http.Request) {
		require.FailNow(t, "request reached target registry")
	}

	// test handler
	h := migrationHandler{Context: ctx, fallback: proxyHandler}
	h.proxyNewRepositories(res, req)

	// validate that request was not proxied to target registry but rather served by the old one
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, proxyHandler.response, res.Body.String())
}

func TestProxyNewRepositories_FailIfTargetIsDown(t *testing.T) {
	// create fake target registry server
	targetHandler := &handlerMock{response: "hello from target registry"}
	targetServer := httptest.NewServer(targetHandler)
	defer targetServer.Close()

	// create test app
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}
	config.Migration.Proxy.Enabled = true
	config.Migration.Proxy.URL = targetServer.URL
	config.HTTP.Headers = http.Header{"foo": []string{"a"}}

	app := NewApp(context.Background(), config)

	// target non-existing repository
	named, err := reference.WithName("test/repo")
	require.NoError(t, err)
	repo, err := app.registry.Repository(context.Background(), named)
	require.NoError(t, err)

	ctx := &Context{
		App:        app,
		Repository: repo,
		Context:    context.Background(),
	}

	// create test request and response
	req := httptest.NewRequest("GET", "http://proxy-registry.example.com/v2/repo/tags/list", nil)
	res := httptest.NewRecorder()
	res.Header().Add("Docker-Distribution-API-Version", "registry/2.0")
	for k, v := range config.HTTP.Headers {
		res.Header().Add(k, strings.Join(v, ","))
	}

	// test handler
	targetServer.Close()

	h := migrationHandler{Context: ctx, fallback: &handlerMock{response: "hello from proxy registry"}}
	h.proxyNewRepositories(res, req)

	// validate that custom headers are not removed from response
	require.Equal(t, "registry/2.0", res.Header().Get("Docker-Distribution-API-Version"))
	for k, v := range config.HTTP.Headers {
		require.Equal(t, strings.Join(v, ","), res.Header().Get(k))
	}

	// validate that request failed to be proxied
	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	b, err := errcode.Errors{errcode.ErrorCodeUnavailable}.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, append(b, '\n'), res.Body.Bytes())
}

func TestProxyNewRepositories_ProxiesRequestsForNewReposThatMatchIncludeFilters(t *testing.T) {
	// create fake target registry server
	targetHandler := &handlerMock{response: "hello from target registry"}
	targetServer := httptest.NewServer(targetHandler)
	defer targetServer.Close()

	// create test app
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}
	config.Migration.Proxy.Enabled = true
	config.Migration.Proxy.URL = targetServer.URL
	config.Migration.Proxy.Include = []*configuration.Regexp{
		{Regexp: regexp.MustCompile("^a.*$")},
		{Regexp: regexp.MustCompile("^test/.*$")},
	}
	app := NewApp(context.Background(), config)

	// target non-existing repository
	named, err := reference.WithName("test/repo")
	require.NoError(t, err)
	repo, err := app.registry.Repository(context.Background(), named)
	require.NoError(t, err)

	ctx := &Context{
		App:        app,
		Repository: repo,
		Context:    context.Background(),
	}

	// create test request and response
	req := httptest.NewRequest("GET", "http://old-registry.example.com/some/path", nil)
	res := httptest.NewRecorder()

	// test handler
	h := migrationHandler{Context: ctx, fallback: &handlerMock{}}
	h.proxyNewRepositories(res, req)

	// validate that request is proxied to target registry
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, targetHandler.response, res.Body.String())
}

func TestProxyNewRepositories_DoesNotProxyRequestsForNewReposThatDoNotMatchIncludeFilters(t *testing.T) {
	// create fake target registry server
	targetHandler := &handlerMock{response: "hello from target registry"}
	targetServer := httptest.NewServer(targetHandler)
	defer targetServer.Close()

	// create test app
	config := &configuration.Configuration{
		Storage: configuration.Storage{"inmemory": configuration.Parameters{}},
	}
	config.Migration.Proxy.Enabled = true
	config.Migration.Proxy.URL = targetServer.URL
	config.Migration.Proxy.Include = []*configuration.Regexp{
		{Regexp: regexp.MustCompile("^a.*$")},
	}
	app := NewApp(context.Background(), config)

	// target non-existing repository
	named, err := reference.WithName("test/repo")
	require.NoError(t, err)
	repo, err := app.registry.Repository(context.Background(), named)
	require.NoError(t, err)

	ctx := &Context{
		App:        app,
		Repository: repo,
		Context:    context.Background(),
	}

	// create test request and response
	req := httptest.NewRequest("GET", "http://old-registry.example.com/some/path", nil)
	res := httptest.NewRecorder()

	// create fake proxy registry handler
	proxyHandler := &handlerMock{response: "hello from proxy registry"}

	// make sure it doesn't reach the target registry
	targetHandler.validatorFn = func(rw http.ResponseWriter, req *http.Request) {
		require.FailNow(t, "request reached target registry")
	}

	// test handler
	h := migrationHandler{Context: ctx, fallback: proxyHandler}
	h.proxyNewRepositories(res, req)

	// validate that request was not proxied to target registry but rather served by the old one
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, proxyHandler.response, res.Body.String())
}
