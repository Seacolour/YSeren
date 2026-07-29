//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	startupRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupValueName    = "YSeren"
)

type windowsStartupManager struct{}

func newStartupManager() startupManager {
	return windowsStartupManager{}
}

func (windowsStartupManager) Enabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE)
	if errorsIsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取开机启动设置失败: %w", err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue(startupValueName)
	if errorsIsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取开机启动命令失败: %w", err)
	}
	return strings.TrimSpace(value) != "", nil
}

func (windowsStartupManager) SetEnabled(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开开机启动注册表失败: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(startupValueName); err != nil && !errorsIsNotExist(err) {
			return fmt.Errorf("关闭开机启动失败: %w", err)
		}
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法确定 Desktop 可执行文件路径: %w", err)
	}
	command := `"` + strings.ReplaceAll(executable, `"`, "") + `" --background`
	if err := key.SetStringValue(startupValueName, command); err != nil {
		return fmt.Errorf("启用开机启动失败: %w", err)
	}
	return nil
}

func errorsIsNotExist(err error) bool {
	return err == registry.ErrNotExist
}
