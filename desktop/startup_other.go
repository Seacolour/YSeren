//go:build !windows

package main

import "errors"

type unsupportedStartupManager struct{}

func newStartupManager() startupManager { return unsupportedStartupManager{} }

func (unsupportedStartupManager) Enabled() (bool, error) { return false, nil }

func (unsupportedStartupManager) SetEnabled(bool) error {
	return errors.New("当前平台尚未实现开机启动")
}
