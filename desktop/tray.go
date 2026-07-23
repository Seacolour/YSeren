package main

import appruntime "yseren/internal/runtime"

type trayActions struct {
	Show          func()
	OpenBrowser   func()
	ToggleSharing func()
	Quit          func()
}

type trayController interface {
	Start(actions trayActions)
	Update(status appruntime.Status)
	Stop()
}
