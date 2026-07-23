//go:build !windows

package main

import appruntime "yseren/internal/runtime"

type noopTray struct{}

func newTray() trayController             { return noopTray{} }
func (noopTray) Start(trayActions)        {}
func (noopTray) Update(appruntime.Status) {}
func (noopTray) Stop()                    {}
