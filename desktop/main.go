package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	webfrontend "yseren/frontend"
	appruntime "yseren/internal/runtime"
)

// Version is injected by release builds with -X main.Version=...
var Version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	configPath := flag.String("config", "", "配置文件路径")
	background := flag.Bool("background", false, "在后台启动并隐藏主窗口")
	flag.Parse()

	store, err := NewConfigStore(ConfigStoreOptions{ExplicitPath: *configPath})
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法初始化 Desktop 配置:", err)
		os.Exit(1)
	}
	preferencesStore := NewPreferencesStore(store.PreferencesPath())
	service := appruntime.New(appruntime.Options{
		FrontendHandler: webfrontend.Handler(),
		Version:         Version,
		Logger:          slog.Default(),
	})
	app := NewApp(AppOptions{
		Service:          service,
		ConfigStore:      store,
		PreferencesStore: preferencesStore,
		Startup:          newStartupManager(),
		Tray:             newTray(),
		Version:          Version,
	})

	err = wails.Run(&options.App{
		Title:       "YSeren",
		Width:       1040,
		Height:      720,
		MinWidth:    880,
		MinHeight:   600,
		StartHidden: *background,
		AssetServer: &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{
			R: 246,
			G: 247,
			B: 251,
			A: 255,
		},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "io.github.seacolour.yseren.desktop",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				app.ShowWindow()
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.SystemDefault,
		},
		Bind: []interface{}{app},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "YSeren Desktop 启动失败:", err)
		os.Exit(1)
	}
}
