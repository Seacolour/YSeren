package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	coreconfig "yseren/internal/config"
	appruntime "yseren/internal/runtime"
	coreversion "yseren/internal/version"
)

const desktopStateEvent = "desktop:state"

var errNoSources = errors.New("请先添加至少一个媒体目录")

type serviceController interface {
	Start(ctx context.Context, conf coreconfig.Config) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context, conf coreconfig.Config) error
	Status() appruntime.Status
	URLs() []string
	Done() <-chan struct{}
}

type AppOptions struct {
	Service          serviceController
	ConfigStore      *ConfigStore
	PreferencesStore *PreferencesStore
	Startup          startupManager
	Tray             trayController
	Version          string
}

type App struct {
	mu               sync.RWMutex
	operationMu      sync.Mutex
	ctx              context.Context
	service          serviceController
	configStore      *ConfigStore
	preferencesStore *PreferencesStore
	startupManager   startupManager
	tray             trayController
	config           coreconfig.Config
	preferences      Preferences
	loadError        string
	version          string
	quitting         bool
}

type SourceState struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

type DesktopState struct {
	Status      appruntime.Status `json:"status"`
	Sources     []SourceState     `json:"sources"`
	Port        int               `json:"port"`
	LogLevel    string            `json:"logLevel"`
	Preferences Preferences       `json:"preferences"`
	Config      ConfigLocation    `json:"config"`
	Version     string            `json:"version"`
	FirstRun    bool              `json:"firstRun"`
	CanStart    bool              `json:"canStart"`
	LoadError   string            `json:"loadError,omitempty"`
}

func NewApp(options AppOptions) *App {
	app := &App{
		service:          options.Service,
		configStore:      options.ConfigStore,
		preferencesStore: options.PreferencesStore,
		startupManager:   options.Startup,
		tray:             options.Tray,
		config:           coreconfig.DefaultConfig(),
		preferences:      DefaultPreferences(),
		version:          coreversion.Normalize(options.Version),
	}
	if app.service == nil {
		app.loadError = "Desktop 服务控制器未初始化"
	}
	if app.configStore == nil {
		app.loadError = joinMessage(app.loadError, "Desktop 配置存储未初始化")
	} else if conf, err := app.configStore.Load(); err != nil {
		app.loadError = joinMessage(app.loadError, err.Error())
	} else {
		app.config = conf
	}
	if app.preferencesStore != nil {
		if preferences, err := app.preferencesStore.Load(); err != nil {
			app.loadError = joinMessage(app.loadError, err.Error())
		} else {
			app.preferences = preferences
		}
	}
	if app.startupManager != nil {
		if enabled, err := app.startupManager.Enabled(); err != nil {
			app.loadError = joinMessage(app.loadError, err.Error())
		} else {
			app.preferences.LaunchAtStartup = enabled
		}
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	preferences := a.preferences
	canAutoStart := a.loadError == "" && len(a.config.Sources) > 0 && preferences.StartSharingOnLaunch
	a.mu.Unlock()

	if a.tray != nil {
		a.tray.Start(trayActions{
			Show: a.ShowWindow,
			OpenBrowser: func() {
				_ = a.OpenBrowser("")
			},
			ToggleSharing: func() {
				if a.service != nil && a.service.Status().State == appruntime.StateRunning {
					_, _ = a.StopSharing()
					return
				}
				_, _ = a.StartSharing()
			},
			Quit: a.Quit,
		})
	}
	a.emitState()
	if canAutoStart {
		go func() {
			_, _ = a.StartSharing()
		}()
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.RLock()
	minimize := a.preferences.MinimizeToTray && !a.quitting
	a.mu.RUnlock()
	if !minimize {
		return false
	}
	wailsruntime.WindowHide(ctx)
	return true
}

func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	if a.service != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = a.service.Stop(ctx)
		cancel()
	}
	if a.tray != nil {
		a.tray.Stop()
	}
}

func (a *App) GetState() DesktopState {
	a.mu.RLock()
	conf := a.config.Clone()
	preferences := a.preferences
	loadError := a.loadError
	version := a.version
	a.mu.RUnlock()

	state := DesktopState{
		Port:        conf.Server.Port,
		LogLevel:    conf.Server.LogLevel,
		Preferences: preferences,
		Version:     version,
		FirstRun:    len(conf.Sources) == 0,
		CanStart:    len(conf.Sources) > 0 && loadError == "",
		LoadError:   loadError,
	}
	if a.configStore != nil {
		state.Config = a.configStore.Location()
	}
	if a.service != nil {
		state.Status = a.service.Status()
	} else {
		state.Status = appruntime.Status{State: appruntime.StateFailed, LastError: "Desktop 服务控制器未初始化"}
	}
	state.Sources = make([]SourceState, 0, len(conf.Sources))
	for _, source := range conf.Sources {
		sourceState := SourceState{Name: source.Name, Path: source.Path, Available: true}
		if err := validateDirectory(source.Path); err != nil {
			sourceState.Available = false
			sourceState.Error = err.Error()
			state.CanStart = false
		}
		state.Sources = append(state.Sources, sourceState)
	}
	return state
}

func (a *App) ChooseDirectory() (string, error) {
	ctx, err := a.runtimeContext()
	if err != nil {
		return "", err
	}
	a.mu.RLock()
	defaultDirectory := ""
	if len(a.config.Sources) > 0 {
		defaultDirectory = a.config.Sources[len(a.config.Sources)-1].Path
	}
	a.mu.RUnlock()
	return wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:            "选择要共享的媒体目录",
		DefaultDirectory: defaultDirectory,
	})
}

func (a *App) AddSource(path string) (DesktopState, error) {
	absolutePath, err := prepareDirectory(path)
	if err != nil {
		return a.GetState(), err
	}

	a.mu.RLock()
	conf := a.config.Clone()
	wasEmpty := len(conf.Sources) == 0
	a.mu.RUnlock()
	for _, source := range conf.Sources {
		if samePath(source.Path, absolutePath) {
			return a.GetState(), fmt.Errorf("该目录已经在媒体源中: %s", absolutePath)
		}
	}
	conf.Sources = append(conf.Sources, coreconfig.Source{
		Name: uniqueSourceName(filepath.Base(absolutePath), conf.Sources),
		Path: absolutePath,
	})
	if err := a.applyConfig(conf); err != nil {
		return a.GetState(), err
	}
	if wasEmpty && a.service != nil && a.service.Status().State != appruntime.StateRunning {
		return a.StartSharing()
	}
	return a.GetState(), nil
}

func (a *App) UpdateSource(index int, name, path string) (DesktopState, error) {
	absolutePath, err := prepareDirectory(path)
	if err != nil {
		return a.GetState(), err
	}
	name = strings.TrimSpace(name)

	a.mu.RLock()
	conf := a.config.Clone()
	a.mu.RUnlock()
	if index < 0 || index >= len(conf.Sources) {
		return a.GetState(), fmt.Errorf("媒体源索引无效: %d", index)
	}
	for currentIndex, source := range conf.Sources {
		if currentIndex != index && samePath(source.Path, absolutePath) {
			return a.GetState(), fmt.Errorf("该目录已经在媒体源中: %s", absolutePath)
		}
	}
	conf.Sources[index] = coreconfig.Source{Name: name, Path: absolutePath}
	if err := a.applyConfig(conf); err != nil {
		return a.GetState(), err
	}
	return a.GetState(), nil
}

func (a *App) RemoveSource(index int) (DesktopState, error) {
	a.mu.RLock()
	conf := a.config.Clone()
	a.mu.RUnlock()
	if index < 0 || index >= len(conf.Sources) {
		return a.GetState(), fmt.Errorf("媒体源索引无效: %d", index)
	}
	conf.Sources = append(conf.Sources[:index], conf.Sources[index+1:]...)
	if err := a.applyConfig(conf); err != nil {
		return a.GetState(), err
	}
	return a.GetState(), nil
}

func (a *App) SetPort(port int) (DesktopState, error) {
	if port < 1 || port > 65535 {
		return a.GetState(), fmt.Errorf("端口必须在 1 到 65535 之间: %d", port)
	}
	a.mu.RLock()
	conf := a.config.Clone()
	a.mu.RUnlock()
	if conf.Server.Port == port {
		return a.GetState(), nil
	}
	conf.Server.Port = port
	if err := a.applyConfig(conf); err != nil {
		return a.GetState(), err
	}
	return a.GetState(), nil
}

func (a *App) StartSharing() (DesktopState, error) {
	if a.service == nil {
		return a.GetState(), errors.New("Desktop 服务控制器未初始化")
	}
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	a.mu.RLock()
	conf := a.config.Clone()
	loadError := a.loadError
	a.mu.RUnlock()
	if loadError != "" {
		return a.GetState(), errors.New(loadError)
	}
	if len(conf.Sources) == 0 {
		return a.GetState(), errNoSources
	}
	for _, source := range conf.Sources {
		if err := validateDirectory(source.Path); err != nil {
			return a.GetState(), fmt.Errorf("媒体源 %q 不可用: %w", source.Name, err)
		}
	}
	if err := a.service.Start(context.Background(), conf); err != nil {
		a.emitState()
		return a.GetState(), friendlyRuntimeError(err, conf.Server.Port)
	}
	a.watchService(a.service.Done())
	a.emitState()
	return a.GetState(), nil
}

func (a *App) StopSharing() (DesktopState, error) {
	if a.service == nil {
		return a.GetState(), errors.New("Desktop 服务控制器未初始化")
	}
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := a.service.Stop(ctx)
	cancel()
	a.emitState()
	return a.GetState(), err
}

func (a *App) RestartSharing() (DesktopState, error) {
	if a.service == nil {
		return a.GetState(), errors.New("Desktop 服务控制器未初始化")
	}
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	a.mu.RLock()
	conf := a.config.Clone()
	a.mu.RUnlock()
	if len(conf.Sources) == 0 {
		return a.GetState(), errNoSources
	}
	for _, source := range conf.Sources {
		if err := validateDirectory(source.Path); err != nil {
			return a.GetState(), fmt.Errorf("媒体源 %q 不可用: %w", source.Name, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := a.service.Restart(ctx, conf)
	cancel()
	if err == nil {
		a.watchService(a.service.Done())
	} else {
		err = friendlyRuntimeError(err, conf.Server.Port)
	}
	a.emitState()
	return a.GetState(), err
}

func (a *App) OpenBrowser(address string) error {
	ctx, err := a.runtimeContext()
	if err != nil {
		return err
	}
	address, err = a.resolveAddress(address)
	if err != nil {
		return err
	}
	wailsruntime.BrowserOpenURL(ctx, address)
	return nil
}

func (a *App) CopyAddress(address string) error {
	ctx, err := a.runtimeContext()
	if err != nil {
		return err
	}
	address, err = a.resolveAddress(address)
	if err != nil {
		return err
	}
	return wailsruntime.ClipboardSetText(ctx, address)
}

func (a *App) ImportConfig() (DesktopState, error) {
	ctx, err := a.runtimeContext()
	if err != nil {
		return a.GetState(), err
	}
	path, err := wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{
		Title: "导入 YSeren YAML 配置",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "YAML 配置 (*.yaml;*.yml)",
			Pattern:     "*.yaml;*.yml",
		}},
	})
	if err != nil || path == "" {
		return a.GetState(), err
	}
	conf, err := coreconfig.LoadConfig(path)
	if err != nil {
		return a.GetState(), fmt.Errorf("导入配置失败: %w", err)
	}
	if err := a.applyConfig(conf.Clone()); err != nil {
		return a.GetState(), err
	}
	return a.GetState(), nil
}

func (a *App) ExportConfig() error {
	ctx, err := a.runtimeContext()
	if err != nil {
		return err
	}
	path, err := wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
		Title:           "导出 YSeren YAML 配置",
		DefaultFilename: "yseren.yaml",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "YAML 配置 (*.yaml)",
			Pattern:     "*.yaml",
		}},
	})
	if err != nil || path == "" {
		return err
	}
	a.mu.RLock()
	conf := a.config.Clone()
	a.mu.RUnlock()
	return coreconfig.SaveConfig(path, conf)
}

func (a *App) UpdatePreferences(minimizeToTray, startSharingOnLaunch bool) (DesktopState, error) {
	a.mu.Lock()
	preferences := a.preferences
	preferences.MinimizeToTray = minimizeToTray
	preferences.StartSharingOnLaunch = startSharingOnLaunch
	a.mu.Unlock()
	if a.preferencesStore != nil {
		if err := a.preferencesStore.Save(preferences); err != nil {
			return a.GetState(), err
		}
	}
	a.mu.Lock()
	a.preferences = preferences
	a.mu.Unlock()
	a.emitState()
	return a.GetState(), nil
}

func (a *App) SetLaunchAtStartup(enabled bool) (DesktopState, error) {
	if a.startupManager == nil {
		return a.GetState(), errors.New("开机启动管理器未初始化")
	}
	if err := a.startupManager.SetEnabled(enabled); err != nil {
		return a.GetState(), err
	}
	a.mu.Lock()
	a.preferences.LaunchAtStartup = enabled
	preferences := a.preferences
	a.mu.Unlock()
	if a.preferencesStore != nil {
		if err := a.preferencesStore.Save(preferences); err != nil {
			return a.GetState(), err
		}
	}
	a.emitState()
	return a.GetState(), nil
}

func (a *App) ShowWindow() {
	ctx, err := a.runtimeContext()
	if err != nil {
		return
	}
	wailsruntime.WindowShow(ctx)
	wailsruntime.WindowUnminimise(ctx)
}

func (a *App) Quit() {
	ctx, err := a.runtimeContext()
	if err != nil {
		return
	}
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	wailsruntime.Quit(ctx)
}

func (a *App) applyConfig(conf coreconfig.Config) error {
	conf.ApplyDefaults()
	if err := conf.Validate(); err != nil {
		return err
	}
	if a.configStore == nil {
		return errors.New("Desktop 配置存储未初始化")
	}
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	if err := a.configStore.Save(conf); err != nil {
		return err
	}
	a.mu.Lock()
	a.config = conf.Clone()
	a.loadError = ""
	a.mu.Unlock()

	if a.service != nil && a.service.Status().State == appruntime.StateRunning {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var err error
		if len(conf.Sources) == 0 {
			err = a.service.Stop(ctx)
		} else {
			err = a.service.Restart(ctx, conf)
			if err == nil {
				a.watchService(a.service.Done())
			}
		}
		cancel()
		if err != nil {
			a.emitState()
			return friendlyRuntimeError(err, conf.Server.Port)
		}
	}
	a.emitState()
	return nil
}

func (a *App) runtimeContext() (context.Context, error) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return nil, errors.New("Desktop 窗口尚未准备完成")
	}
	return ctx, nil
}

func (a *App) resolveAddress(address string) (string, error) {
	if a.service == nil {
		return "", errors.New("Desktop 服务控制器未初始化")
	}
	urls := a.service.URLs()
	if len(urls) == 0 {
		return "", errors.New("共享服务尚未运行")
	}
	address = strings.TrimSpace(address)
	if address == "" {
		if len(urls) > 1 {
			return urls[1], nil
		}
		return urls[0], nil
	}
	for _, candidate := range urls {
		if address == candidate {
			return address, nil
		}
	}
	return "", errors.New("该地址不属于当前 YSeren 服务")
}

func (a *App) watchService(done <-chan struct{}) {
	if done == nil {
		return
	}
	go func() {
		<-done
		a.emitState()
	}()
}

func (a *App) emitState() {
	state := a.GetState()
	if a.tray != nil {
		a.tray.Update(state.Status)
	}
	ctx, err := a.runtimeContext()
	if err == nil {
		wailsruntime.EventsEmit(ctx, desktopStateEvent, state)
	}
}

func validateDirectory(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("目录路径不能为空")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("目录不存在")
		}
		return fmt.Errorf("无法访问目录: %w", err)
	}
	if !info.IsDir() {
		return errors.New("路径不是目录")
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("目录不可读: %w", err)
	}
	_, readErr := directory.Readdirnames(1)
	closeErr := directory.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return fmt.Errorf("目录不可读: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭目录失败: %w", closeErr)
	}
	return nil
}

func prepareDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("未选择目录")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("解析目录路径失败: %w", err)
	}
	absolutePath = filepath.Clean(absolutePath)
	if err := validateDirectory(absolutePath); err != nil {
		return "", err
	}
	return absolutePath, nil
}

func uniqueSourceName(base string, sources []coreconfig.Source) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "媒体"
	}
	base = strings.ReplaceAll(base, "/", "-")
	base = strings.ReplaceAll(base, "\\", "-")
	base = strings.ReplaceAll(base, "..", "-")
	base = strings.TrimSpace(base)
	if base == "" {
		base = "媒体"
	}
	used := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		used[strings.ToLower(source.Name)] = struct{}{}
	}
	if _, ok := used[strings.ToLower(base)]; !ok {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s %d", base, suffix)
		if _, ok := used[strings.ToLower(candidate)]; !ok {
			return candidate
		}
	}
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	return strings.EqualFold(left, right)
}

func friendlyRuntimeError(err error, port int) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(strings.ToLower(message), "address already in use") || strings.Contains(message, "通常每个套接字地址") {
		return fmt.Errorf("端口 %d 已被其他程序占用，请在设置中更换端口", port)
	}
	return err
}

func joinMessage(current, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return current + "；" + next
}
