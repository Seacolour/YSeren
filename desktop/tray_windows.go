//go:build windows

package main

import (
	_ "embed"
	"sync"

	"fyne.io/systray"

	appruntime "yseren/internal/runtime"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

type windowsTray struct {
	mu         sync.Mutex
	actions    trayActions
	statusItem *systray.MenuItem
	toggleItem *systray.MenuItem
	started    bool
	stopOnce   sync.Once
}

func newTray() trayController {
	return &windowsTray{}
}

func (t *windowsTray) Start(actions trayActions) {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return
	}
	t.started = true
	t.actions = actions
	t.mu.Unlock()
	go systray.Run(t.ready, func() {})
}

func (t *windowsTray) ready() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("YSeren - 局域网媒体共享")
	systray.SetOnTapped(func() {
		if t.actions.Show != nil {
			t.actions.Show()
		}
	})

	openItem := systray.AddMenuItem("打开 YSeren", "显示主窗口")
	browserItem := systray.AddMenuItem("在浏览器中打开", "打开当前共享地址")
	systray.AddSeparator()
	t.statusItem = systray.AddMenuItem("共享已停止", "当前服务状态")
	t.statusItem.Disable()
	t.toggleItem = systray.AddMenuItem("开始共享", "启动或停止局域网共享")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "停止共享并退出 YSeren")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				if t.actions.Show != nil {
					t.actions.Show()
				}
			case <-browserItem.ClickedCh:
				if t.actions.OpenBrowser != nil {
					t.actions.OpenBrowser()
				}
			case <-t.toggleItem.ClickedCh:
				if t.actions.ToggleSharing != nil {
					t.actions.ToggleSharing()
				}
			case <-quitItem.ClickedCh:
				if t.actions.Quit != nil {
					t.actions.Quit()
				}
				return
			}
		}
	}()
}

func (t *windowsTray) Update(status appruntime.Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.statusItem == nil || t.toggleItem == nil {
		return
	}
	switch status.State {
	case appruntime.StateRunning:
		t.statusItem.SetTitle("共享运行中")
		t.toggleItem.SetTitle("停止共享")
		systray.SetTooltip("YSeren - 共享运行中")
	case appruntime.StateStarting:
		t.statusItem.SetTitle("正在启动共享…")
		t.toggleItem.SetTitle("停止共享")
	case appruntime.StateStopping:
		t.statusItem.SetTitle("正在停止共享…")
		t.toggleItem.SetTitle("开始共享")
	case appruntime.StateFailed:
		t.statusItem.SetTitle("共享启动失败")
		t.toggleItem.SetTitle("重试共享")
		systray.SetTooltip("YSeren - 共享启动失败")
	default:
		t.statusItem.SetTitle("共享已停止")
		t.toggleItem.SetTitle("开始共享")
		systray.SetTooltip("YSeren - 共享已停止")
	}
}

func (t *windowsTray) Stop() {
	t.stopOnce.Do(systray.Quit)
}
