package registry

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/docker/distribution/configuration"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/labkit/monitoring"
)

// Tests to ensure nextProtos returns the correct protocols when:
// * config.HTTP.HTTP2.Disabled is not explicitly set => [h2 http/1.1]
// * config.HTTP.HTTP2.Disabled is explicitly set to false [h2 http/1.1]
// * config.HTTP.HTTP2.Disabled is explicitly set to true [http/1.1]
func TestNextProtos(t *testing.T) {
	config := &configuration.Configuration{}
	protos := nextProtos(config)
	if !reflect.DeepEqual(protos, []string{"h2", "http/1.1"}) {
		t.Fatalf("expected protos to equal [h2 http/1.1], got %s", protos)
	}
	config.HTTP.HTTP2.Disabled = false
	protos = nextProtos(config)
	if !reflect.DeepEqual(protos, []string{"h2", "http/1.1"}) {
		t.Fatalf("expected protos to equal [h2 http/1.1], got %s", protos)
	}
	config.HTTP.HTTP2.Disabled = true
	protos = nextProtos(config)
	if !reflect.DeepEqual(protos, []string{"http/1.1"}) {
		t.Fatalf("expected protos to equal [http/1.1], got %s", protos)
	}
}

func setupRegistry() (*Registry, error) {
	config := &configuration.Configuration{}
	configuration.ApplyDefaults(config)
	// probe free port where the server can listen
	ln, err := net.Listen("tcp", ":")
	if err != nil {
		return nil, err
	}
	defer ln.Close()
	config.HTTP.Addr = ln.Addr().String()
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	return NewRegistry(context.Background(), config)
}

func TestGracefulShutdown(t *testing.T) {
	var tests = []struct {
		name                string
		cleanServerShutdown bool
		httpDrainTimeout    time.Duration
	}{
		{
			name:                "http draintimeout greater than 0 runs server.Shutdown",
			cleanServerShutdown: true,
			httpDrainTimeout:    10 * time.Second,
		},
		{
			name:                "http draintimeout 0 or less does not run server.Shutdown",
			cleanServerShutdown: false,
			httpDrainTimeout:    0 * time.Second,
		},
	}

	for _, tt := range tests {
		registry, err := setupRegistry()
		if err != nil {
			t.Fatal(err)
		}

		registry.config.HTTP.DrainTimeout = tt.httpDrainTimeout

		// Register on shutdown fuction to detect if server.Shutdown() was ran.
		var cleanServerShutdown bool
		registry.server.RegisterOnShutdown(func() {
			cleanServerShutdown = true
		})

		// run registry server
		var errchan chan error
		go func() {
			errchan <- registry.ListenAndServe()
		}()
		select {
		case err = <-errchan:
			t.Fatalf("Error listening: %v", err)
		default:
		}

		// Wait for some unknown random time for server to start listening
		time.Sleep(3 * time.Second)

		// Send quit signal, this does not track to the signals that the registry
		// is actually configured to listen to since we're interacting with the
		// channel directly — any signal sent on this channel triggers the shutdown.
		quit <- syscall.SIGTERM
		time.Sleep(100 * time.Millisecond)

		if cleanServerShutdown != tt.cleanServerShutdown {
			t.Fatalf("expected clean shutdown to be %v, got %v", tt.cleanServerShutdown, cleanServerShutdown)
		}
	}
}

func TestGracefulShutdown_HTTPDrainTimeout(t *testing.T) {
	registry, err := setupRegistry()
	if err != nil {
		t.Fatal(err)
	}

	// run registry server
	var errchan chan error
	go func() {
		errchan <- registry.ListenAndServe()
	}()
	select {
	case err = <-errchan:
		t.Fatalf("Error listening: %v", err)
	default:
	}

	// Wait for some unknown random time for server to start listening
	time.Sleep(3 * time.Second)

	// send incomplete request
	conn, err := net.Dial("tcp", registry.config.HTTP.Addr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(conn, "GET /v2/ ")

	// send stop signal
	quit <- os.Interrupt
	time.Sleep(100 * time.Millisecond)

	// try connecting again. it shouldn't
	_, err = net.Dial("tcp", registry.config.HTTP.Addr)
	if err == nil {
		t.Fatal("Managed to connect after stopping.")
	}

	// make sure earlier request is not disconnected and response can be received
	fmt.Fprintf(conn, "HTTP/1.1\r\nHost: 127.0.0.1\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "200 OK" {
		t.Error("response status is not 200 OK: ", resp.Status)
	}
	if body, err := ioutil.ReadAll(resp.Body); err != nil || string(body) != "{}" {
		t.Error("Body is not {}; ", string(body))
	}
}

func requireEnvNotSet(t *testing.T, names ...string) {
	t.Helper()

	for _, name := range names {
		_, ok := os.LookupEnv(name)
		require.False(t, ok)
	}
}

func requireEnvSet(t *testing.T, name, value string) {
	t.Helper()

	require.Equal(t, value, os.Getenv(name))
}

func TestConfigureStackDriver_Disabled(t *testing.T) {
	config := &configuration.Configuration{}

	requireEnvNotSet(t, "GITLAB_CONTINUOUS_PROFILING")
	require.NoError(t, configureStackdriver(config))
	requireEnvNotSet(t, "GITLAB_CONTINUOUS_PROFILING")
}

func TestConfigureStackDriver_Enabled(t *testing.T) {
	config := &configuration.Configuration{
		Profiling: configuration.Profiling{
			Stackdriver: configuration.StackdriverProfiler{
				Enabled: true,
			},
		},
	}

	requireEnvNotSet(t, "GITLAB_CONTINUOUS_PROFILING")
	require.NoError(t, configureStackdriver(config))
	requireEnvSet(t, "GITLAB_CONTINUOUS_PROFILING", "stackdriver")
	require.NoError(t, os.Unsetenv("GITLAB_CONTINUOUS_PROFILING"))
}

func TestConfigureStackDriver_WithParams(t *testing.T) {
	config := &configuration.Configuration{
		Profiling: configuration.Profiling{
			Stackdriver: configuration.StackdriverProfiler{
				Enabled:        true,
				Service:        "registry",
				ServiceVersion: "2.9.1",
				ProjectID:      "internal",
			},
		},
	}

	requireEnvNotSet(t, "GITLAB_CONTINUOUS_PROFILING")
	require.NoError(t, configureStackdriver(config))
	defer os.Unsetenv("GITLAB_CONTINUOUS_PROFILING")

	requireEnvSet(t, "GITLAB_CONTINUOUS_PROFILING", "stackdriver?project_id=internal&service=registry&service_version=2.9.1")

}

func TestConfigureStackDriver_WithKeyFile(t *testing.T) {
	config := &configuration.Configuration{
		Profiling: configuration.Profiling{
			Stackdriver: configuration.StackdriverProfiler{
				Enabled: true,
				KeyFile: "/path/to/credentials.json",
			},
		},
	}

	requireEnvNotSet(t, "GITLAB_CONTINUOUS_PROFILING")
	require.NoError(t, configureStackdriver(config))
	defer os.Unsetenv("GITLAB_CONTINUOUS_PROFILING")

	requireEnvSet(t, "GITLAB_CONTINUOUS_PROFILING", "stackdriver")
}

func TestConfigureStackDriver_DoesNotOverrideGitlabContinuousProfilingEnvVar(t *testing.T) {
	value := "stackdriver?project_id=foo&service=bar&service_version=1"
	require.NoError(t, os.Setenv("GITLAB_CONTINUOUS_PROFILING", value))

	config := &configuration.Configuration{
		Profiling: configuration.Profiling{
			Stackdriver: configuration.StackdriverProfiler{
				Enabled:        true,
				Service:        "registry",
				ServiceVersion: "2.9.1",
				ProjectID:      "internal",
			},
		},
	}

	require.NoError(t, configureStackdriver(config))
	defer os.Unsetenv("GITLAB_CONTINUOUS_PROFILING")

	requireEnvSet(t, "GITLAB_CONTINUOUS_PROFILING", value)
}

func freeLnAddr(t *testing.T) net.Addr {
	t.Helper()

	ln, err := net.Listen("tcp", ":")
	require.NoError(t, err)
	addr := ln.Addr()
	require.NoError(t, ln.Close())

	return addr
}
func assertMonitoringResponse(t *testing.T, addr, path string, expectedStatus int) {
	t.Helper()

	u := url.URL{Scheme: "http", Host: addr, Path: path}
	req, err := http.Get(u.String())
	require.NoError(t, err)
	defer req.Body.Close()
	require.Equal(t, expectedStatus, req.StatusCode)
}

func TestConfigureMonitoring_NoErrorWithNoOptions(t *testing.T) {
	config := &configuration.Configuration{}

	go func() {
		err := monitoring.Start(configureMonitoring(config)...)
		require.NoError(t, err)
	}()
}

func TestConfigureMonitoring_HealthHandler(t *testing.T) {
	addr := freeLnAddr(t).String()
	config := &configuration.Configuration{}
	config.HTTP.Debug.Addr = addr

	go func() {
		err := monitoring.Start(configureMonitoring(config)...)
		require.NoError(t, err)
	}()

	assertMonitoringResponse(t, addr, "/debug/health", http.StatusOK)
	assertMonitoringResponse(t, addr, "/debug/pprof", http.StatusNotFound)
	assertMonitoringResponse(t, addr, "/metrics", http.StatusNotFound)
}

func TestConfigureMonitoring_PprofHandler(t *testing.T) {
	addr := freeLnAddr(t).String()
	config := &configuration.Configuration{}
	config.HTTP.Debug.Addr = addr
	config.HTTP.Debug.Pprof.Enabled = true

	go func() {
		err := monitoring.Start(configureMonitoring(config)...)
		require.NoError(t, err)
	}()

	assertMonitoringResponse(t, addr, "/debug/health", http.StatusOK)
	assertMonitoringResponse(t, addr, "/debug/pprof", http.StatusOK)
	assertMonitoringResponse(t, addr, "/metrics", http.StatusNotFound)
}

func TestConfigureMonitoring_MetricsHandler(t *testing.T) {
	ln, err := net.Listen("tcp", ":")
	require.NoError(t, err)
	defer ln.Close()

	addr := ln.Addr().String()
	config := &configuration.Configuration{}
	config.HTTP.Debug.Addr = addr
	config.HTTP.Debug.Prometheus.Enabled = true
	config.HTTP.Debug.Prometheus.Path = "/metrics"

	go func() {
		opts := configureMonitoring(config)
		// Use local Prometheus registry for each test, otherwise different tests may attempt to register the same
		// metrics in the default Prometheus registry, causing a panic.
		opts = append(opts, monitoring.WithPrometheusRegisterer(prometheus.NewRegistry()))
		opts = append(opts, monitoring.WithListener(ln))
		err = monitoring.Start(opts...)
		require.NoError(t, err)
	}()

	assertMonitoringResponse(t, addr, "/debug/health", http.StatusOK)
	assertMonitoringResponse(t, addr, "/debug/pprof", http.StatusNotFound)
	assertMonitoringResponse(t, addr, "/metrics", http.StatusOK)
}

func TestConfigureMonitoring_All(t *testing.T) {
	ln, err := net.Listen("tcp", ":")
	require.NoError(t, err)
	defer ln.Close()

	addr := ln.Addr().String()
	config := &configuration.Configuration{}
	config.HTTP.Debug.Addr = addr
	config.HTTP.Debug.Pprof.Enabled = true
	config.HTTP.Debug.Prometheus.Enabled = true
	config.HTTP.Debug.Prometheus.Path = "/metrics"

	go func() {
		opts := configureMonitoring(config)
		// Use local Prometheus registry for each test, otherwise different tests may attempt to register the same
		// metrics in the default Prometheus registry, causing a panic.
		opts = append(opts, monitoring.WithPrometheusRegisterer(prometheus.NewRegistry()))
		opts = append(opts, monitoring.WithListener(ln))
		err := monitoring.Start(opts...)
		require.NoError(t, err)
	}()

	assertMonitoringResponse(t, addr, "/debug/health", http.StatusOK)
	assertMonitoringResponse(t, addr, "/debug/pprof", http.StatusOK)
	assertMonitoringResponse(t, addr, "/metrics", http.StatusOK)
}
