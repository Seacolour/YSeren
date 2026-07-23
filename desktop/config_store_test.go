package main

import (
	"os"
	"path/filepath"
	"testing"

	coreconfig "yseren/internal/config"
)

func TestConfigStoreUsesExplicitPathFirst(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	explicitPath := filepath.Join(root, "explicit.yaml")
	writeTestConfig(t, explicitPath, 2480)
	writeTestConfig(t, filepath.Join(root, "exe", "yseren.yaml"), 3480)
	writeTestConfig(t, filepath.Join(root, "user", "YSeren", "yseren.yaml"), 4480)

	store, err := NewConfigStore(ConfigStoreOptions{
		ExplicitPath:   explicitPath,
		ExecutablePath: filepath.Join(root, "exe", "YSeren.exe"),
		UserConfigDir:  filepath.Join(root, "user", "YSeren"),
	})
	if err != nil {
		t.Fatalf("NewConfigStore() error = %v", err)
	}
	conf, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if store.Location().Mode != ConfigModeExplicit || conf.Server.Port != 2480 {
		t.Fatalf("location = %#v, port = %d", store.Location(), conf.Server.Port)
	}
}

func TestConfigStoreDetectsPortableMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	portablePath := filepath.Join(root, "app", "yseren.yml")
	writeTestConfig(t, portablePath, 3480)
	store, err := NewConfigStore(ConfigStoreOptions{
		ExecutablePath: filepath.Join(root, "app", "YSeren.exe"),
		UserConfigDir:  filepath.Join(root, "user", "YSeren"),
	})
	if err != nil {
		t.Fatalf("NewConfigStore() error = %v", err)
	}
	conf, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if store.Location().Mode != ConfigModePortable || store.Location().Path != portablePath || conf.Server.Port != 3480 {
		t.Fatalf("location = %#v, port = %d", store.Location(), conf.Server.Port)
	}
	if got := store.PreferencesPath(); got != filepath.Join(root, "app", "yseren.desktop.json") {
		t.Fatalf("PreferencesPath() = %q", got)
	}
}

func TestConfigStoreCreatesUserConfigOnFirstSave(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	userDir := filepath.Join(root, "user", "YSeren")
	store, err := NewConfigStore(ConfigStoreOptions{
		ExecutablePath: filepath.Join(root, "app", "YSeren.exe"),
		UserConfigDir:  userDir,
	})
	if err != nil {
		t.Fatalf("NewConfigStore() error = %v", err)
	}
	conf, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if conf.Server.Port != coreconfig.DefaultPort || store.Location().Exists {
		t.Fatalf("default location = %#v, config = %#v", store.Location(), conf)
	}
	conf.Server.Port = 2480
	if err := store.Save(conf); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !store.Location().Exists {
		t.Fatal("location should exist after Save")
	}
	loaded, err := coreconfig.LoadConfig(filepath.Join(userDir, "yseren.yaml"))
	if err != nil || loaded.Server.Port != 2480 {
		t.Fatalf("saved config = %#v, error = %v", loaded, err)
	}
}

func TestExplicitConfigMustExist(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.yaml")
	store, err := NewConfigStore(ConfigStoreOptions{ExplicitPath: path, UserConfigDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewConfigStore() error = %v", err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want missing explicit config error")
	}
}

func writeTestConfig(t *testing.T, path string, port int) {
	t.Helper()
	conf := coreconfig.DefaultConfig()
	conf.Server.Port = port
	if err := coreconfig.SaveConfig(path, conf); err != nil {
		t.Fatalf("SaveConfig(%q) error = %v", path, err)
	}
}

func TestPreferencesStoreRoundTripAndReplace(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings", "desktop.json")
	store := NewPreferencesStore(path)
	defaults, err := store.Load()
	if err != nil || !defaults.MinimizeToTray || !defaults.StartSharingOnLaunch {
		t.Fatalf("default preferences = %#v, error = %v", defaults, err)
	}
	updated := Preferences{LaunchAtStartup: true}
	if err := store.Save(updated); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	updated.MinimizeToTray = true
	if err := store.Save(updated); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded != updated {
		t.Fatalf("loaded preferences = %#v, error = %v", loaded, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("preferences file stat error = %v", err)
	}
}
