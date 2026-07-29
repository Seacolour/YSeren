# YSeren Desktop

YSeren Desktop 是 Windows 上的轻量控制外壳。它在同一进程内复用仓库根目录的 Go Core，负责选择媒体目录、启停共享、展示局域网地址以及托盘和开机启动等桌面集成；媒体浏览和播放仍由默认浏览器承担。

Desktop 与 Headless 是并列交付形态。根目录的 `go build .`、`-config` 和原有 YAML 使用方式不受 Desktop 模块影响。

## 当前支持范围

- Windows 10/11 x64。
- 首次运行选择目录并自动开始共享、打开浏览器。
- “共享 / 媒体源 / 设置”三页控制界面。
- 多媒体源添加、编辑、移除及目录可读性检查。
- 进程内启动、停止和配置变更重启。
- 地址复制、默认浏览器打开、YAML 导入和导出。
- 系统托盘、关闭到托盘、单实例唤回和优雅退出。
- 当前用户级开机启动。
- AppData 安装模式和 YAML 同目录 Portable 模式。

YSeren 不在 Desktop 内实现原生播放器、转码、刮削或云同步。

## 架构边界

`desktop` 是独立 Go 模块，通过下面的本地替换直接调用 Go Core：

```go
replace yseren => ..
```

Wails、WebView 和托盘依赖只存在于 Desktop 模块，不会进入 Headless 的依赖图。Desktop 不启动外置 `yseren.exe` 子进程。

当前 Windows 技术栈：

- Go 1.24。
- Wails v2.12.0。
- Svelte 5 + Vite 8。
- `fyne.io/systray` v1.12.2。
- Microsoft Edge WebView2 Evergreen Runtime。
- NSIS 3.x 安装包。

Wails 固定在 v2.12.0，是因为该版本兼容当前 Go 1.24 工具链；升级 Wails 前应先核对其最低 Go 版本。

## 配置位置

Desktop 按以下优先级选择核心 YAML：

1. `YSeren.exe -config <path>` 指定的文件。
2. `YSeren.exe` 同目录的 `yseren.yaml` 或 `yseren.yml`，视为 Portable 模式。
3. `%APPDATA%\YSeren\yseren.yaml`，视为普通安装模式。

窗口和托盘偏好与核心配置分离：

- Portable：EXE 同目录的 `yseren.desktop.json`。
- 普通安装：`%APPDATA%\YSeren\desktop.json`。

制作 Portable 包时，应把 `YSeren.exe` 和一份可写的 `yseren.yaml` 放在同一目录。若同目录不存在 YAML，应用会使用 AppData 模式。

## 本地开发

前置依赖：

- Go 1.24.x。
- Node.js 与 npm。
- WebView2 Runtime。
- Wails CLI v2.12.0。

安装固定版本的 Wails CLI：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
```

启动开发模式：

```powershell
cd desktop
wails dev
```

构建 Windows EXE：

```powershell
cd desktop
wails build -clean
```

输出文件为 `desktop/build/bin/YSeren.exe`。

非正式构建在应用内显示为“开发版本”。正式 Release 会从同一个
`vMAJOR.MINOR.PATCH` tag 注入 Go 运行版本、Windows EXE 产品版本和安装器版本。

## 构建用户级安装包

先安装 NSIS 3.x，并确保 `makensis.exe` 所在目录已经加入当前终端的 `PATH`，然后运行：

```powershell
cd desktop
wails build -nsis -clean
```

安装包输出为：

```text
desktop/build/bin/YSeren-Desktop-Windows-amd64-Setup.exe
```

安装器使用当前用户权限，不触发 UAC，默认安装到：

```text
%LOCALAPPDATA%\Programs\YSeren
```

卸载信息写入 HKCU，快捷方式也只为当前用户创建。卸载时会移除程序、快捷方式、卸载项和 YSeren 开机启动项，但保留 `%APPDATA%\YSeren` 中的核心配置与 Desktop 偏好。

## 验证命令

```powershell
# Headless/Core（仓库根目录）
go test ./... -count=1
go vet ./...
go build .

# Desktop
cd desktop
go test ./... -count=1
go vet ./...
npm --prefix frontend run build
wails build -nsis -clean
```

手工验收至少覆盖首次选目录、启动/停止、Range 媒体访问、端口变更、托盘、单实例、浏览器打开、用户级安装与卸载。

## 已知边界

- 当前 GUI 文案以中文为主。
- WebView2 缺失时由 NSIS 安装流程尝试补齐；离线环境应预先安装 Runtime。
- Windows 防火墙首次提示和真实局域网访问仍取决于本机网络配置。
- Linux Desktop 和 Android 的后续工作由重构计划中的后续 Phase 推进。
