package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	coreconfig "yseren/internal/config"
)

func TestRuntimeStartsStopsAndRestarts(t *testing.T) {
	t.Parallel()

	runtime := newTestRuntime(Options{})
	conf := testConfig(t, 0)
	if err := runtime.Start(context.Background(), conf); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	assertRuntimeReachable(t, runtime)
	firstStartedAt := runtime.Status().StartedAt

	if err := runtime.Restart(context.Background(), conf); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	status := runtime.Status()
	if status.State != StateRunning || status.StartedAt.Before(firstStartedAt) {
		t.Fatalf("status after restart = %#v", status)
	}
	assertRuntimeReachable(t, runtime)

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if status := runtime.Status(); status.State != StateStopped || status.Port != 0 || len(status.URLs) != 0 {
		t.Fatalf("status after stop = %#v", status)
	}
	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}

	if err := runtime.Start(context.Background(), conf); err != nil {
		t.Fatalf("Start() after stop error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Stop(context.Background()) })
	assertRuntimeReachable(t, runtime)
}

func TestRuntimeRejectsDuplicateStart(t *testing.T) {
	t.Parallel()

	runtime := newTestRuntime(Options{})
	conf := testConfig(t, 0)
	if err := runtime.Start(context.Background(), conf); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Stop(context.Background()) })

	if err := runtime.Start(context.Background(), conf); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("duplicate Start() error = %v, want ErrAlreadyRunning", err)
	}
	if runtime.Status().State != StateRunning {
		t.Fatalf("duplicate start changed status to %#v", runtime.Status())
	}
}

func TestRuntimeReportsOccupiedPortSynchronously(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	runtime := newTestRuntime(Options{})
	err = runtime.Start(context.Background(), testConfig(t, port))
	if err == nil {
		t.Fatal("Start() error = nil, want occupied-port error")
	}
	status := runtime.Status()
	if status.State != StateFailed || status.LastError == "" {
		t.Fatalf("status after bind failure = %#v", status)
	}
}

func TestRuntimeGracefulStopWaitsForActiveRequest(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})
	factory := func(*coreconfig.Config) (http.Handler, error) {
		mux := http.NewServeMux()
		mux.HandleFunc("/block", func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			w.WriteHeader(http.StatusNoContent)
		})
		return mux, nil
	}
	runtime := newTestRuntime(Options{HandlerFactory: factory, ShutdownTimeout: 2 * time.Second})
	if err := runtime.Start(context.Background(), testConfig(t, 0)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get(runtime.URLs()[0] + "block")
		if err == nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	<-entered

	stopDone := make(chan error, 1)
	go func() { stopDone <- runtime.Stop(context.Background()) }()
	waitForState(t, runtime, StateStopping)
	select {
	case err := <-stopDone:
		t.Fatalf("Stop() returned before active request completed: %v", err)
	default:
	}

	close(release)
	if err := <-requestDone; err != nil {
		t.Fatalf("active request error = %v", err)
	}
	if err := <-stopDone; err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if runtime.Status().State != StateStopped {
		t.Fatalf("status after graceful stop = %#v", runtime.Status())
	}
}

func TestConcurrentStopsWaitForFullShutdown(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})
	factory := func(*coreconfig.Config) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			w.WriteHeader(http.StatusNoContent)
		}), nil
	}
	runtime := newTestRuntime(Options{HandlerFactory: factory, ShutdownTimeout: 2 * time.Second})
	if err := runtime.Start(context.Background(), testConfig(t, 0)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	requestDone := make(chan struct{})
	go func() {
		response, _ := http.Get(runtime.URLs()[0])
		if response != nil {
			_ = response.Body.Close()
		}
		close(requestDone)
	}()
	<-entered

	firstStop := make(chan error, 1)
	secondStop := make(chan error, 1)
	go func() { firstStop <- runtime.Stop(context.Background()) }()
	waitForState(t, runtime, StateStopping)
	go func() { secondStop <- runtime.Stop(context.Background()) }()
	select {
	case err := <-secondStop:
		t.Fatalf("concurrent Stop() returned early: %v", err)
	default:
	}

	close(release)
	<-requestDone
	if err := <-firstStop; err != nil {
		t.Fatalf("first Stop() error = %v", err)
	}
	if err := <-secondStop; err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
	if runtime.Status().State != StateStopped {
		t.Fatalf("status after concurrent stops = %#v", runtime.Status())
	}
}

func newTestRuntime(options Options) *Runtime {
	options.ListenAddress = "127.0.0.1"
	options.LANIPv4 = func() []string { return nil }
	return New(options)
}

func testConfig(t *testing.T, port int) coreconfig.Config {
	t.Helper()
	return coreconfig.Config{
		Server: coreconfig.ServerConfig{Port: port},
		Sources: []coreconfig.Source{{
			Name: "media",
			Path: filepath.Clean(t.TempDir()),
		}},
		MediaExtensions: append([]string(nil), coreconfig.DefaultMediaExtensions...),
	}
}

func assertRuntimeReachable(t *testing.T, runtime *Runtime) {
	t.Helper()
	status := runtime.Status()
	if status.State != StateRunning || status.Port == 0 || len(status.URLs) != 1 {
		t.Fatalf("running status = %#v", status)
	}
	response, err := http.Get(status.URLs[0] + "api/tree")
	if err != nil {
		t.Fatalf("GET /api/tree: %v", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/tree status = %d", response.StatusCode)
	}

	statusResponse, err := http.Get(status.URLs[0] + "api/status")
	if err != nil {
		t.Fatalf("GET /api/status: %v", err)
	}
	defer statusResponse.Body.Close()
	var body struct {
		State  string   `json:"state"`
		Source string   `json:"source"`
		Port   int      `json:"port"`
		URLs   []string `json:"urls"`
	}
	if err := json.NewDecoder(statusResponse.Body).Decode(&body); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}
	if statusResponse.StatusCode != http.StatusOK || body.State != string(StateRunning) || body.Source != "media" || body.Port != status.Port || len(body.URLs) != 1 {
		t.Fatalf("GET /api/status response = status %d body %#v", statusResponse.StatusCode, body)
	}
}

func waitForState(t *testing.T, runtime *Runtime, want State) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.Status().State == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("state = %s, want %s", runtime.Status().State, want)
}
