package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	coreconfig "yseren/internal/config"
	appruntime "yseren/internal/runtime"
)

type fakeService struct {
	mu         sync.Mutex
	status     appruntime.Status
	done       chan struct{}
	startErr   error
	starts     int
	stops      int
	restarts   int
	lastConfig coreconfig.Config
}

func newFakeService() *fakeService {
	return &fakeService{status: appruntime.Status{State: appruntime.StateStopped}}
}

func (f *fakeService) Start(_ context.Context, conf coreconfig.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	if f.startErr != nil {
		f.status = appruntime.Status{State: appruntime.StateFailed, LastError: f.startErr.Error()}
		return f.startErr
	}
	f.lastConfig = conf.Clone()
	f.done = make(chan struct{})
	f.status = appruntime.Status{State: appruntime.StateRunning, Port: conf.Server.Port, URLs: []string{"http://localhost:1479/"}}
	return nil
}

func (f *fakeService) Stop(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	if f.done != nil {
		close(f.done)
		f.done = nil
	}
	f.status = appruntime.Status{State: appruntime.StateStopped}
	return nil
}

func (f *fakeService) Restart(ctx context.Context, conf coreconfig.Config) error {
	f.mu.Lock()
	f.restarts++
	if f.done != nil {
		close(f.done)
		f.done = nil
	}
	f.status = appruntime.Status{State: appruntime.StateStopped}
	f.mu.Unlock()
	return f.Start(ctx, conf)
}

func (f *fakeService) Status() appruntime.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	status := f.status
	status.URLs = append([]string(nil), status.URLs...)
	return status
}

func (f *fakeService) URLs() []string { return f.Status().URLs }

func (f *fakeService) Done() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.done != nil {
		return f.done
	}
	done := make(chan struct{})
	close(done)
	return done
}

type fakeStartupManager struct{ enabled bool }

func (f *fakeStartupManager) Enabled() (bool, error) { return f.enabled, nil }
func (f *fakeStartupManager) SetEnabled(enabled bool) error {
	f.enabled = enabled
	return nil
}

func TestAddFirstSourceSavesAndStartsSharing(t *testing.T) {
	t.Parallel()

	app, service, store := newTestApp(t)
	mediaDir := filepath.Join(t.TempDir(), "Videos")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("create media dir: %v", err)
	}
	state, err := app.AddSource(mediaDir)
	if err != nil {
		t.Fatalf("AddSource() error = %v", err)
	}
	if state.Status.State != appruntime.StateRunning || len(state.Sources) != 1 || state.Sources[0].Name != "Videos" {
		t.Fatalf("state = %#v", state)
	}
	if service.starts != 1 || service.lastConfig.Sources[0].Path != mediaDir {
		t.Fatalf("service starts = %d, config = %#v", service.starts, service.lastConfig)
	}
	loaded, err := store.Load()
	if err != nil || len(loaded.Sources) != 1 {
		t.Fatalf("saved config = %#v, error = %v", loaded, err)
	}
}

func TestRunningConfigChangesRestartAndRemovingLastSourceStops(t *testing.T) {
	t.Parallel()

	app, service, _ := newTestApp(t)
	mediaDir := filepath.Join(t.TempDir(), "Media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("create media dir: %v", err)
	}
	if _, err := app.AddSource(mediaDir); err != nil {
		t.Fatalf("AddSource() error = %v", err)
	}
	if _, err := app.SetPort(2480); err != nil {
		t.Fatalf("SetPort() error = %v", err)
	}
	if service.restarts != 1 || service.lastConfig.Server.Port != 2480 {
		t.Fatalf("restarts = %d, config = %#v", service.restarts, service.lastConfig)
	}
	state, err := app.RemoveSource(0)
	if err != nil {
		t.Fatalf("RemoveSource() error = %v", err)
	}
	if state.Status.State != appruntime.StateStopped || service.stops != 1 || len(state.Sources) != 0 {
		t.Fatalf("state = %#v, stops = %d", state, service.stops)
	}
}

func TestAddSourceRejectsDuplicateAndMissingDirectory(t *testing.T) {
	t.Parallel()

	app, _, _ := newTestApp(t)
	mediaDir := filepath.Join(t.TempDir(), "Media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("create media dir: %v", err)
	}
	if _, err := app.AddSource(mediaDir); err != nil {
		t.Fatalf("AddSource() error = %v", err)
	}
	if _, err := app.AddSource(mediaDir); err == nil {
		t.Fatal("duplicate AddSource() error = nil")
	}
	if _, err := app.AddSource(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing AddSource() error = nil")
	}
}

func TestStartSharingReturnsFriendlyPortConflict(t *testing.T) {
	t.Parallel()

	app, service, _ := newTestApp(t)
	mediaDir := filepath.Join(t.TempDir(), "Media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("create media dir: %v", err)
	}
	service.startErr = errors.New("listen tcp :1479: bind: address already in use")
	_, err := app.AddSource(mediaDir)
	if err == nil || err.Error() != "端口 1479 已被其他程序占用，请在设置中更换端口" {
		t.Fatalf("AddSource() error = %v", err)
	}
}

func newTestApp(t *testing.T) (*App, *fakeService, *ConfigStore) {
	t.Helper()
	root := t.TempDir()
	store, err := NewConfigStore(ConfigStoreOptions{
		ExecutablePath: filepath.Join(root, "app", "YSeren.exe"),
		UserConfigDir:  filepath.Join(root, "config"),
	})
	if err != nil {
		t.Fatalf("NewConfigStore() error = %v", err)
	}
	service := newFakeService()
	app := NewApp(AppOptions{
		Service:          service,
		ConfigStore:      store,
		PreferencesStore: NewPreferencesStore(store.PreferencesPath()),
		Startup:          &fakeStartupManager{},
		Version:          "dev",
	})
	return app, service, store
}
