package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	coreconfig "yseren/internal/config"
)

type ConfigMode string

const (
	ConfigModeExplicit ConfigMode = "explicit"
	ConfigModePortable ConfigMode = "portable"
	ConfigModeUser     ConfigMode = "user"
)

type ConfigStoreOptions struct {
	ExplicitPath   string
	ExecutablePath string
	UserConfigDir  string
}

type ConfigStore struct {
	mu       sync.RWMutex
	location ConfigLocation
	userDir  string
}

type ConfigLocation struct {
	Path   string     `json:"path"`
	Mode   ConfigMode `json:"mode"`
	Exists bool       `json:"exists"`
}

func NewConfigStore(options ConfigStoreOptions) (*ConfigStore, error) {
	userDir := strings.TrimSpace(options.UserConfigDir)
	if userDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("无法确定用户配置目录: %w", err)
		}
		userDir = filepath.Join(base, "YSeren")
	}

	if explicit := strings.TrimSpace(options.ExplicitPath); explicit != "" {
		path, err := filepath.Abs(explicit)
		if err != nil {
			return nil, fmt.Errorf("解析显式配置路径失败: %w", err)
		}
		return &ConfigStore{
			location: ConfigLocation{Path: path, Mode: ConfigModeExplicit, Exists: regularFileExists(path)},
			userDir:  userDir,
		}, nil
	}

	executable := strings.TrimSpace(options.ExecutablePath)
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("无法确定 Desktop 可执行文件路径: %w", err)
		}
	}
	executableDir := filepath.Dir(executable)
	for _, name := range coreconfig.PreferredConfigNames {
		path := filepath.Join(executableDir, name)
		if regularFileExists(path) {
			return &ConfigStore{
				location: ConfigLocation{Path: path, Mode: ConfigModePortable, Exists: true},
				userDir:  userDir,
			}, nil
		}
	}

	path := filepath.Join(userDir, coreconfig.PreferredConfigNames[0])
	return &ConfigStore{
		location: ConfigLocation{Path: path, Mode: ConfigModeUser, Exists: regularFileExists(path)},
		userDir:  userDir,
	}, nil
}

func (s *ConfigStore) Load() (coreconfig.Config, error) {
	if s == nil {
		return coreconfig.Config{}, errors.New("Desktop 配置存储未初始化")
	}
	s.mu.RLock()
	location := s.location
	s.mu.RUnlock()
	if !location.Exists {
		if location.Mode == ConfigModeExplicit {
			return coreconfig.Config{}, fmt.Errorf("指定的配置文件不存在: %s", location.Path)
		}
		return coreconfig.DefaultConfig(), nil
	}
	conf, err := coreconfig.LoadConfig(location.Path)
	if err != nil {
		return coreconfig.Config{}, fmt.Errorf("加载配置 %s 失败: %w", location.Path, err)
	}
	return conf.Clone(), nil
}

func (s *ConfigStore) Save(conf coreconfig.Config) error {
	if s == nil {
		return errors.New("Desktop 配置存储未初始化")
	}
	s.mu.RLock()
	path := s.location.Path
	s.mu.RUnlock()
	if err := coreconfig.SaveConfig(path, conf); err != nil {
		return err
	}
	s.mu.Lock()
	s.location.Exists = true
	s.mu.Unlock()
	return nil
}

func (s *ConfigStore) Location() ConfigLocation {
	if s == nil {
		return ConfigLocation{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.location
}

func (s *ConfigStore) PreferencesPath() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	location := s.location
	userDir := s.userDir
	s.mu.RUnlock()
	if location.Mode == ConfigModePortable {
		return filepath.Join(filepath.Dir(location.Path), "yseren.desktop.json")
	}
	return filepath.Join(userDir, "desktop.json")
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
