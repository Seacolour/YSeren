package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Preferences struct {
	MinimizeToTray       bool `json:"minimizeToTray"`
	StartSharingOnLaunch bool `json:"startSharingOnLaunch"`
	LaunchAtStartup      bool `json:"launchAtStartup"`
}

func DefaultPreferences() Preferences {
	return Preferences{
		MinimizeToTray:       true,
		StartSharingOnLaunch: true,
	}
}

type PreferencesStore struct {
	path string
}

func NewPreferencesStore(path string) *PreferencesStore {
	return &PreferencesStore{path: path}
}

func (s *PreferencesStore) Load() (Preferences, error) {
	prefs := DefaultPreferences()
	if s == nil || s.path == "" {
		return prefs, errors.New("Desktop 偏好设置路径为空")
	}
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return prefs, nil
	}
	if err != nil {
		return prefs, fmt.Errorf("读取 Desktop 偏好设置失败: %w", err)
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return DefaultPreferences(), fmt.Errorf("解析 Desktop 偏好设置失败: %w", err)
	}
	return prefs, nil
}

func (s *PreferencesStore) Save(prefs Preferences) error {
	if s == nil || s.path == "" {
		return errors.New("Desktop 偏好设置路径为空")
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 Desktop 偏好设置失败: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("创建 Desktop 配置目录失败: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".desktop-*.tmp")
	if err != nil {
		return fmt.Errorf("创建 Desktop 偏好设置临时文件失败: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("写入 Desktop 偏好设置失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭 Desktop 偏好设置失败: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("替换 Desktop 偏好设置失败: %w", err)
	}
	removeTemporary = false
	return nil
}
