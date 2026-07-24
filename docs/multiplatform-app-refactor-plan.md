# YSeren 多平台应用化重构计划

- 状态：Phase 0、Phase 1、Phase 2 Windows Desktop MVP、Phase 3 Android 均已完成；下一阶段为 Phase 4 Linux Desktop
- 适用平台：Windows、Linux、Android
- 播放端：现代浏览器
- 关联文档：[YSeren 项目推进计划](./task.md)

## 1. 文档目的

本文档用于指导 YSeren 从当前的“Go 单可执行文件 + YAML + Svelte Web UI”结构，逐步演进为同时支持以下产品入口的多平台工具：

- 面向熟悉配置和后台运行场景的 Headless 版本。
- 面向普通桌面用户的 Windows/Linux Desktop 版本。
- 面向移动媒体源场景的 Android 应用。
- 作为所有平台统一播放入口的浏览器 Web Player。

本次演进属于较大范围的架构调整，但不采用一次性推倒重写。每个阶段都必须保持现有 Headless 主链路可运行、可验证、可发布。

## 2. 已确认的产品边界

### 2.1 核心问题

YSeren 解决的问题是：

> 一台设备已经存在媒体文件，另一台设备无需上传、复制或重新整理，即可通过局域网直接浏览和播放。

成功标准应始终围绕“缩短从现有媒体文件到另一台设备开始播放的路径”，而不是扩展为完整媒体中心。

### 2.2 本阶段支持范围

- Windows 作为媒体源：
  - Headless 版本。
  - Desktop GUI 版本。
- Linux 作为媒体源：
  - Headless 版本优先。
  - Desktop GUI 在 Windows MVP 稳定后推进。
- Android 作为媒体源：
  - 通过 Storage Access Framework 选择目录。
  - 通过前台服务在局域网持续共享。
- 任意具备兼容浏览器的设备作为播放端。
- iPhone/iPad 可继续作为 Safari 播放端，但本阶段不实现 iOS 媒体源应用。

### 2.3 明确不做

- 不自建原生音视频播放器。
- 不承担转码、转封装和媒体编码兼容层。
- 不做媒体刮削、海报墙和元数据平台。
- 不做账号系统、云同步和远程存储。
- 不要求播放端安装 YSeren 应用。
- 不在本阶段实现 iOS 媒体源。
- 不因 GUI 版本而移除 YAML、命令行或 Headless 能力。

## 3. 总体架构

~~~mermaid
flowchart TD
    CORE["YSeren Go Core<br/>配置、媒体目录、索引、HTTP、Range、状态"] --> HEADLESS["YSeren Headless<br/>命令行 + YAML"]
    CORE --> DESKTOP["YSeren Desktop<br/>GUI + 托盘 + 本地控制"]
    HEADLESS --> CONTRACT["统一媒体服务契约"]
    DESKTOP --> CONTRACT
    ANDROID["YSeren Android<br/>Compose + SAF + 前台服务"] --> CONTRACT
    CONTRACT --> WEB["Svelte Web Player<br/>浏览、搜索、播放、进度、播放列表"]
    WEB --> BROWSER["Windows / Linux / Android / iOS 浏览器"]
~~~

架构必须保持三个职责边界：

1. 控制层
   - 选择共享目录。
   - 启动、停止和查看服务状态。
   - 展示、复制和打开局域网地址。
2. 服务层
   - 扫描允许共享的媒体文件。
   - 提供目录树、搜索和媒体流。
   - 正确处理 MIME、Range、错误和缓存。
3. 播放层
   - 由 Svelte Web Player 和浏览器承担。
   - 服务层不实现解码、渲染和转码。

## 4. 产品与发布形态

所有产品入口使用同一个版本号和同一个 Git tag，不维护功能分叉分支。

| 产品 | 面向用户 | 配置方式 | 主要产物 |
|------|----------|----------|----------|
| YSeren Headless | 高级用户、后台运行、服务器 | YAML、命令行参数 | 单可执行文件 + 示例 YAML |
| YSeren Desktop | 普通 Windows/Linux 用户 | GUI 自动管理，允许导入/导出 YAML | 安装包、Desktop Portable 包 |
| YSeren Android | Android 媒体源用户 | 原生应用内设置 | APK |
| YSeren Web Player | 所有播放设备 | 无需安装 | 嵌入各媒体源产物 |

建议的 Release 产物命名：

- YSeren-Headless-Windows-x64.zip
- YSeren-Headless-Windows-arm64.zip
- YSeren-Headless-Linux-x64.tar.gz
- YSeren-Headless-Linux-arm64.tar.gz
- YSeren-Desktop-Windows-x64-Setup.exe
- YSeren-Desktop-Windows-x64-Portable.zip
- YSeren-Desktop-Linux-x64.AppImage 或对应发行版包
- YSeren-Android.apk
- SHA256SUMS.txt

“Desktop”和“Headless”只表示交互入口不同，不表示功能质量和维护等级不同。

## 5. 建议的代码结构

目标结构可以在实施过程中逐步形成，不要求一次性移动所有文件。

~~~text
cmd/
  yseren/
    main.go                 Headless 入口
  yseren-desktop/
    main.go                 Desktop 入口

internal/
  config/                   YAML、默认值、校验和配置路径
  media/                    扩展名、媒体类型、目录扫描和索引
  server/                   路由、Handler、流媒体响应和前端静态资源
  runtime/                  Start、Stop、Restart、Status、URL 和事件
  version/                  版本解析与更新检查

frontend/                   Svelte Web Player
desktop/                    Desktop 控制界面和平台集成
android/                    Android 应用
docs/                       产品、协议和实施文档
~~~

Phase 1 的实际落地保留了仓库根目录 `main.go` 作为 Headless 入口，以继续兼容现有 `go build .`、构建脚本和 `-X main.Version=...`。核心实现已经迁入 `internal/`，嵌入式 Web Player 已成为可复用的 `frontend` Go 包；创建 Desktop 入口时再新增 `cmd/yseren-desktop`，无需为了目录形式提前改变现有 Headless 构建命令。

### 5.1 Go Core 的目标

Go Core 必须同时被 Headless 和 Desktop 直接调用。稳定架构中，Desktop 不应通过启动另一个 yseren.exe 子进程来获得核心能力。

核心运行时至少需要提供：

~~~go
type Runtime interface {
    Start(ctx context.Context, config Config) error
    Stop(ctx context.Context) error
    Restart(ctx context.Context, config Config) error
    Status() Status
    URLs() []string
}
~~~

实际接口可以在实施时调整，但必须满足：

- 可以在同一进程内启动和停止。
- 启动失败能返回可展示给普通用户的错误。
- Desktop 可以订阅状态和日志变化。
- Headless 可以复用同一套错误与日志。
- 测试可以使用独立端口和临时目录启动服务。
- 不再依赖全局 http.DefaultServeMux。
- 使用独立 http.Server 和 ServeMux，明确设置超时与关闭流程。

### 5.2 Android 的边界

Android 继续使用 Kotlin/Compose、Storage Access Framework 和前台服务，不要求强行运行 Go Core。

Android 与 Go Core 通过以下内容保持一致：

- API 契约。
- JSON 字段和时间单位。
- 媒体扩展名基线。
- MIME 判断规则。
- Range 行为。
- 错误语义。
- Svelte Web Player 构建产物。
- 共享的产品文案与视觉规范。

## 6. 配置与兼容策略

### 6.1 Headless 兼容要求

现有用户不应因为重构而修改原有启动方式。

必须保留：

- -config 指定配置文件。
- 当前目录和可执行文件目录下的 yseren.yaml / yseren.yml。
- 现有 server、sources、media_extensions、audio_extensions 字段。
- 现有默认端口和日志级别行为。
- 现有 Release 中的示例配置。

### 6.2 Desktop 配置位置

Desktop 默认不要求用户看到 YAML。

建议优先级：

1. 显式 -config 参数。
2. 可执行文件同目录的 yseren.yaml / yseren.yml，视为 Desktop Portable 模式。
3. 平台用户配置目录：
   - Windows：AppData 下的 YSeren 配置目录。
   - Linux：XDG_CONFIG_HOME/yseren，未设置时使用标准用户配置目录。
4. 无配置时进入首次运行向导。

Desktop 应允许：

- 导入现有 YAML。
- 导出当前核心配置为 YAML。
- 在 GUI 中修改来源目录、展示名称和端口。
- 在修改后明确提示是否需要重启服务。

窗口大小、托盘偏好、首次运行状态等 Desktop 专属设置，应保存在独立的 UI 偏好文件中，不污染核心 YAML 配置。

## 7. Desktop MVP

Desktop 是“控制外壳”，不是播放器。

### 7.1 信息架构

Windows/Linux Desktop 与 Android 尽量使用一致的三层结构：

1. 共享
   - 服务运行状态。
   - 当前共享目录摘要。
   - 本机访问地址。
   - 局域网地址。
   - 复制地址。
   - 打开浏览器。
   - 开始、停止和重启。
   - 可选二维码。
2. 媒体源
   - 添加目录。
   - 移除目录。
   - 修改显示名称。
   - 检查目录是否存在和可读。
   - 支持将文件夹拖入窗口。
3. 设置
   - 默认端口。
   - 开机启动。
   - 最小化到托盘。
   - 关闭窗口时的行为。
   - 日志查看和日志目录。
   - 高级设置中展示媒体扩展名等配置。

### 7.2 首次运行流程

目标流程：

> 双击应用 → 选择媒体目录 → 开始共享 → 显示局域网地址 → 打开浏览器

首次运行至少需要处理：

- 没有选择目录。
- 目录不存在或不可读。
- 端口被占用。
- Windows 防火墙未允许访问。
- 当前没有可用局域网 IPv4。
- 服务已经由另一个实例运行。

### 7.3 托盘和生命周期

Desktop MVP 需要：

- 托盘图标展示运行状态。
- 托盘菜单提供“打开 YSeren”“打开浏览器”“开始/停止共享”“退出”。
- 用户可选择关闭窗口时退出或最小化到托盘。
- 防止无意启动多个占用同一端口的实例。
- 应用退出时执行有超时的优雅停机。

### 7.4 Desktop 技术选择

Windows Phase 2 已确认采用 Wails v2.12.0 + Svelte 5。Desktop 作为独立嵌套 Go 模块，通过 `replace yseren => ..` 直接复用 Go Core，Wails、WebView 和托盘依赖不进入 Headless 模块。

本阶段的具体决定：

- Wails 固定为 v2.12.0，以兼容当前 Go 1.24 工具链；升级前必须重新确认最低 Go 版本。
- 托盘使用仍在维护的纯 Go `fyne.io/systray` v1.12.2。
- Windows 使用 Evergreen WebView2；安装器在缺失时调用 Wails 提供的 WebView2 bootstrapper。
- 安装包使用 NSIS，采用当前用户权限并安装到 `%LOCALAPPDATA%\Programs\YSeren`，不要求 UAC。
- 安装模式使用 `%APPDATA%\YSeren`，EXE 同目录存在 YAML 时切换为 Portable 模式。
- 浏览器继续承担媒体播放；Desktop 只负责本地控制和平台集成。
- Linux WebKitGTK、托盘和分发格式验证推迟到 Phase 4，不影响 Windows MVP 的框架决定。

## 8. Web Player 的长期定位

Svelte Web Player 是跨平台共享程度最高、最需要保持稳定的用户界面。

它继续负责：

- 目录浏览。
- 搜索。
- 音频和视频标签播放。
- URL 状态恢复。
- 播放进度。
- 浏览器原生媒体控制（包括浏览器支持的倍速菜单）。
- 播放列表。
- 更新提示。

它不负责：

- 自定义解码器。
- 转码。
- 承诺所有扫描格式均可在所有浏览器播放。
- 绕过浏览器和设备的编码限制。

后续应区分：

- “YSeren 可以共享的媒体格式”。
- “当前浏览器大概率可以直接播放的格式”。

播放失败时优先提供清晰说明、复制媒体地址或外部播放器入口，而不是立即引入原生播放器。

## 9. 跨平台 API 契约

### 9.1 建议的基础端点

最终基础契约建议包含：

- GET /api/status
- GET /api/tree
- GET /api/tree?q=...
- GET /api/tree?refresh=1
- GET /api/version
- GET /stream/...
- 可选 GET /playlist.m3u

现有 /api/videos 暂时保留兼容，但当前 Web Player 已以 /api/tree 为主。待确认没有外部用户依赖后，再决定是否标记废弃。

### 9.2 JSON 字段

Go 与 Android 必须统一：

- type：dir 或 file。
- name。
- relPath。
- source。
- url。
- size：字节。
- modTime：Unix 秒。
- mediaType：video 或 audio。
- children。
- generatedAt：Unix 秒。

当前 Android 与 Go 在 modTime / lastModified 名称及秒/毫秒单位上存在差异，进入契约阶段时必须统一，并保留必要的兼容读取。

### 9.3 流媒体 URL

Phase 0 已在第一版契约中确定标准形式：

- /stream/{source}/{relativePath}

该形式支持桌面多来源，并已写入 `contracts/fixtures/tree-response.v1.json` 与 Go 回归测试。Android 可在过渡期兼容旧地址，但新生成地址应在 Phase 3 对齐该标准。

### 9.4 Range 行为

Windows、Linux、Android 必须满足相同的最小行为：

- 无 Range 的 GET 返回 200 和准确 Content-Length。
- 合法单段 Range 返回 206。
- 206 包含 Accept-Ranges、Content-Range 和准确 Content-Length。
- 支持 start-end、start- 和 suffix range。
- 非法或超范围请求返回 416，并包含 Content-Range: bytes */total。
- HEAD 不传输正文，但返回与 GET 一致的关键响应头。
- 客户端提前断开时正确关闭文件或 SAF 流。
- 正确返回 MIME，未知类型使用 application/octet-stream。

本阶段不要求多段 Range。

## 10. 安全与隐私底线

GUI 会降低共享门槛，因此在面向普通用户发布前，必须先收紧现有服务边界。

### 10.1 必须完成

- /stream/ 只允许访问已识别且位于配置来源内的媒体文件。
- 禁止原始目录列表。
- 禁止通过已知路径读取非媒体文件。
- 明确处理路径穿越、URL 解码、符号链接和 Windows junction。
- Desktop 控制能力只允许本机访问，或直接通过进程内调用实现。
- 不向局域网暴露修改配置、启停服务、选择目录等控制接口。
- UI 明确显示当前正在共享的目录。
- UI 提示仅应在可信局域网中使用。
- 不以管理员权限作为默认运行要求。

### 10.2 暂缓但保留扩展点

- 可选访问令牌。
- 设备配对。
- HTTPS。
- mDNS/Bonjour 发现。
- 网络接口白名单。

这些能力不进入首个 Desktop MVP，除非实际用户反馈证明已经影响基本安全性或可用性。

## 11. 多端视觉与交互统一

“多端统一”首先指产品语言一致，而不是强制共享同一套 UI 代码。

应统一：

- Logo 和品牌名称。
- 颜色、圆角、间距和状态色。
- “共享 / 媒体源 / 设置”信息架构。
- 运行中、已停止、启动失败等状态。
- 地址、复制、打开浏览器等操作文案。
- 基础设置和高级设置的分层。
- 空状态、错误状态和首次运行提示。

允许不同：

- Desktop 使用侧边导航，Android 使用底部导航。
- Desktop 使用系统目录选择器，Android 使用 SAF。
- Desktop 使用托盘，Android 使用前台通知。
- 平台权限和后台生命周期相关说明。

## 12. 分阶段实施计划

### Phase 0：基线与安全护栏

目标：在结构调整前固定现有行为，优先解决 GUI 扩大用户面后最危险的边界。

任务：

- 为配置、URL 编码、树过滤、缓存和媒体类型补充单元测试。
- 为 /api/tree 和流媒体 Range 增加 Handler 集成测试。
- 限制 /stream/ 只能读取允许的媒体文件。
- 禁止目录列表和非媒体文件读取。
- 记录当前 Go、前端、Android 构建基线。
- 建立第一版跨平台 JSON 和 Range 契约样例。

验收：

- Headless 启动方式不变。
- Go 测试、go vet、Go build、前端 build 通过。
- 非媒体文件即使路径已知也无法通过 /stream/ 获取。
- 合法媒体 Range 回归测试通过。

完成记录（2026-07-22）：

- 已用统一的受限流媒体 Handler 替换原始目录文件服务器；仅允许 GET/HEAD、已配置来源内的普通媒体文件，并阻止目录列表、非媒体文件、路径穿越、符号链接或 junction 逃逸以及 Windows ADS 路径。
- 已为配置默认值与校验、路径安全、URL 编码、目录树过滤与刷新、缓存并发与过期、流媒体 MIME、HEAD 和 Range 补充回归测试。
- 已建立 `contracts/fixtures/tree-response.v1.json` 与 `contracts/fixtures/range-cases.v1.json`，确定第一版目录树字段、Unix 秒时间单位、标准流媒体 URL 与单段 Range 行为。
- `go test ./... -count=10`、`go test ./... -count=1`、`go vet ./...`、`go build ./...` 与前端 `npm run build` 均通过；Go 覆盖率基线为 42.9%。
- Android `:app:assembleDebug` 使用 JDK 17 构建通过，确认 Phase 0 的共享契约文件和 Go 侧改动未破坏现有 Android 工程构建。
- `go test -race` 未执行：当前本机 Go 环境设置为 `CGO_ENABLED=0`，race detector 不受支持；这属于尚未覆盖的环境验证，不计为通过。
- 本阶段未进行真机、局域网跨设备或浏览器手工播放验证；这些项目继续保留在发布前验证矩阵中。

### Phase 1：提取 Go Core

目标：Headless 不再把所有能力绑定在 package main 和全局 HTTP 状态中。

任务：

- 拆分 config、media、server、runtime。
- 使用独立 ServeMux 和 http.Server。
- 实现 Start、Stop、Restart、Status、URLs。
- 保留原有 CLI 参数和配置发现行为。
- 让现有 yseren 入口只负责参数、日志和运行时装配。
- 为优雅关闭、端口占用和重复启动增加测试。

验收：

- Headless 功能和现有配置兼容。
- Runtime 可以在测试进程中启动、停止并再次启动。
- 测试不依赖全局 DefaultServeMux。
- 没有引入 Desktop 框架依赖到 Headless 构建。

完成记录（2026-07-22）：

- 已提取 `internal/config`、`internal/media`、`internal/server`、`internal/runtime` 和 `internal/version`；根 `package main` 仅保留 CLI、日志/崩溃记录与构建版本变量。
- 嵌入式 Svelte Web Player 已迁入可复用的 `frontend` Go 包，Headless 仍保持单可执行文件交付，未来 Desktop 无需导入或启动 `package main`。
- 每个 HTTP Application 使用独立 `http.ServeMux`、媒体索引缓存和版本检查缓存；生产代码不再注册或依赖 `http.DefaultServeMux`。
- Runtime 已实现 `Start`、`Stop`、`Restart`、`Status`、`URLs` 与 `Done`，启动时同步绑定端口并返回错误；HTTP Server 明确设置读 Header、空闲和 Header 大小限制，同时为长时间媒体响应保留无 WriteTimeout 的流式行为。
- 已覆盖临时端口启动、停止后再次启动、运行中重启、重复启动、端口占用、活动请求优雅停止、并发停止等待、Runtime 间缓存隔离以及默认 ServeMux 隔离。
- 保留原有 `-config` 参数与“显式路径 → 当前目录 → exe 同目录”的配置发现顺序，并增加自动发现回归测试；仓库根目录 `go build .` 和 `-X main.Version=...` 注入方式保持兼容。
- 迁移嵌入式前端时修复了 SPA 未知路由被 `/index.html` 规范化为 301 的问题，现在直接返回嵌入入口；启动错误页同时对动态错误信息执行 HTML 转义。
- `go test ./... -count=10`、`go test ./... -cover`、`go vet ./...`、Windows Go build、Linux amd64 CGO-disabled Go build、前端 `npm run build` 和 Android JDK 17 debug build 均通过；Runtime 覆盖率为 60.9%，Server 覆盖率为 83.6%。
- Go 依赖图中没有 Wails、Fyne、WebView 或其他 Desktop 框架依赖，Headless 构建仍保持纯 Go Core 边界。
- `go test -race` 仍未执行：本机 `CGO_ENABLED=0` 且未安装 GCC；真机、局域网跨设备和浏览器手工播放也尚未在本阶段执行。

### Phase 2：Windows Desktop MVP

目标：普通 Windows 用户无需编辑 YAML 即可完成共享。

任务：

- 完成 Desktop 框架技术验证并记录决定。
- 实现首次运行向导。
- 实现“共享 / 媒体源 / 设置”三页。
- 实现目录选择、启动停止、状态、地址、复制和打开浏览器。
- 实现托盘、单实例和退出策略。
- 实现 AppData 配置及 Desktop Portable 模式。
- 制作 Windows 安装包和 Portable 包。

验收：

- 新用户可在三步内从首次启动进入浏览器播放页面。
- Desktop 与 Headless 使用相同 Go Core。
- Desktop 不依赖外置 yseren.exe 子进程。
- 端口冲突、无目录和目录不可读有明确提示。
- 关闭、托盘和退出行为可预测。

完成记录（2026-07-23）：

- 新增独立 `desktop` Go 模块，Wails 依赖未进入根 Headless 模块；Desktop 在同一进程内调用 Go Runtime，不启动外置 `yseren.exe`。
- 实现首次运行流程以及“共享 / 媒体源 / 设置”三页，支持目录选择、媒体源增删改、目录可读性、启动停止、地址复制、默认浏览器打开、端口热重启和 YAML 导入导出。
- 实现显式配置、EXE 同目录 Portable、AppData 三层配置优先级，并将窗口、托盘和启动偏好保存在独立 JSON 文件中。
- 实现 Windows 托盘菜单、关闭到托盘、单实例唤回、当前用户开机启动和有超时的优雅退出。
- 使用 Wails v2.12.0、Svelte 5、WebView2 与 `fyne.io/systray`；Windows EXE 和 NSIS 用户级安装包均已成功生成。
- 已真实验证首次选目录后自动共享并打开浏览器、Runtime 启停、`/api/tree`、MP4 单段 Range、媒体源添加与改名、端口热重启、关闭到托盘、单实例唤回和完整退出。
- 用户已使用安装版从 `D:\Code\yseren\resource` 加载真实测试资源；Windows Chrome 通过局域网地址正常浏览 MP3、MP4，ZIP 压缩包按设计被自动忽略。
- 用户级安装已验证落在 `%LOCALAPPDATA%\Programs\YSeren\YSeren.exe`，卸载信息写入 HKCU，并为当前用户创建桌面和开始菜单快捷方式。
- 当前阶段只完成 Windows；Linux Desktop、Android 契约收敛和跨设备真机矩阵仍按后续 Phase 推进。

### Phase 3：Android 契约收敛

目标：Android 媒体源与 Go 服务对 Web Player 表现一致。

任务：

- 统一 /api/status 和 /api/tree 数据结构。
- 统一 modTime、size、mediaType 和 generatedAt。
- 对齐标准流媒体 URL。
- 修正非法 Range、HEAD、后缀 Range 和资源释放。
- 引入共享契约样例测试。
- 将 Android 控制台的信息架构和 Desktop 对齐。

验收：

- 同一 Web Player 无平台特判即可访问 Go 与 Android 服务。
- 契约样例在 Go 和 Android 中都通过。
- Android debug/release 构建通过。
- Android 真机前台服务和浏览器播放完成手工验证。

完成记录（2026-07-24）：

- `/api/tree` 已收敛为 `root -> android -> media`，Android 新生成的流地址统一为 `/stream/android/<relative-path>`，并保留旧 `/stream/<relative-path>` 地址兼容。
- `size`、`modTime`、`generatedAt`、`mediaType`、`source` 和 URL 字段已与共享契约对齐；时间统一使用 Unix 秒，`/api/status` 不再向局域网暴露 SAF `content://` URI，并新增 `/api/version`。
- 修复了完整 GET、HEAD、固定区间、开放区间、后缀 Range、非法 Range 416、方法限制与 SAF 输入流关闭；Android 单元测试直接复用 `contracts/fixtures` 中的目录树和 Range 契约样例。
- Android 控制界面已重做为“共享 / 媒体源 / 设置”三页 Compose 应用，采用固定品牌配色和底部导航；支持可读目录路径、媒体数量、选择/更换/移除/重新扫描目录、启停服务、本机浏览器入口、局域网地址复制和运行中端口热重启。
- 雷电 Android 9/API 28 模拟器识别了测试目录中的 3 个 MP4，并忽略 ZIP；真实 HTTP 验证覆盖 200、206、416、HEAD、旧地址和 405，端口热重启 `1479 -> 1480 -> 1479` 通过。
- Windows 浏览器通过 ADB 端口转发访问同一套 Web Player，无平台特判进入 Android 媒体树并实际播放 `bear.mp4`；播放进度前进、媒体错误为空，流请求为 206，浏览器控制台无错误和警告。
- `:app:testDebugUnitTest`、`:app:assembleDebug`、`:app:lintVitalRelease` 与 `:app:assembleRelease` 已通过；无签名变量时成功生成 unsigned Release APK。
- 雷电模拟器使用 NAT，模拟器显示的 `172.16.1.15:1479` 无法由 Windows 直接访问，因此模拟器浏览器结果只记录为“模拟器 + ADB 转发”。随后已在 Android 真机上从 `内部存储 / Movies` 识别 24 个媒体文件，并由同一 Wi-Fi 下的 Windows Chrome 直接访问 `http://192.168.50.171:1479/`，完成标准目录树浏览和媒体实际播放；Phase 3 真机跨设备验收通过。
- Web Player 播放页移除了与 Chrome 等浏览器原生媒体菜单重复的自建倍速按钮，播放速度继续交由浏览器原生控件处理。

### Phase 4：Linux Desktop

目标：在不削弱 Linux Headless 的前提下提供可安装 GUI。

任务：

- 验证目标发行版的 WebView、托盘和目录选择器。
- 明确最低支持发行版或运行库。
- 选择 AppImage、deb、rpm、tar.gz 中的首批分发格式。
- 适配 XDG 配置和日志目录。
- 验证桌面环境差异和无托盘环境的退化行为。

验收：

- Linux Headless 仍可作为无 GUI 依赖的单文件运行。
- 至少一个 Desktop 分发格式完成安装、启动、共享和卸载验证。
- GUI 缺少托盘或系统组件时有明确降级提示。

### Phase 5：发布整合与体验打磨

目标：形成可长期维护的多产品 Release。

任务：

- 统一版本注入，修复非 tag 构建误显示为最近正式版本的问题。
- 一个 tag 构建全部 Headless、Desktop 和 Android 产物。
- 生成 SHA256SUMS。
- 更新 README 下载选择说明。
- 编写 Headless、Desktop、Android 的最短启动路径。
- 增加截图、首次运行说明和防火墙排障。

验收：

- Release 中各产物版本一致。
- 用户能明确选择 Desktop、Headless 或 Android。
- 旧版 YAML 可直接用于新版 Headless，并可由 Desktop 导入。
- 发布前验证矩阵全部完成并留有记录。

## 13. 测试与验证矩阵

### 13.1 自动化

- Go：
  - 配置解析和校验。
  - 媒体扩展名与类型。
  - 路径安全。
  - URL 编码。
  - 树过滤和排序。
  - 缓存并发与过期。
  - API Handler。
  - Range 和 HEAD。
  - Runtime 生命周期。
- Svelte：
  - 生产构建。
  - 核心状态模块单元测试。
  - 目录、搜索和播放 URL 的浏览器 smoke test。
- Android：
  - Repository 单元测试。
  - JSON 契约测试。
  - Range 解析测试。
  - Service/Controller 生命周期测试。
- 跨平台：
  - 相同测试目录生成相同的标准化目录树。
  - 相同 Range 请求得到等价响应。

### 13.2 手工验证

至少覆盖：

- Windows 本机浏览器。
- Windows 到同一局域网 Android 浏览器。
- Linux Headless 到移动浏览器。
- Android 媒体源到 Windows 浏览器。
- 中文、空格、加号、百分号和 Unicode 文件名。
- 大文件拖动进度和多次跳转。
- 音频、视频和浏览器不支持的编码。
- 服务重启、端口占用、网络切换和客户端中断。
- Windows 防火墙首次提示。

所有验证记录必须区分：

- 自动化测试通过。
- 编译通过。
- 模拟器验证。
- 真机验证。
- 尚未验证。

## 14. 迁移与提交原则

- 不做一次性大提交。
- 每次移动模块后立即保持 Headless 可构建。
- 先增加新入口，再删除旧入口。
- 先增加兼容读取，再改变输出格式。
- 生成文件和构建产物不混入功能提交。
- Go、Desktop、Android 和文档尽量保持可单独审查。
- 每个 Phase 完成后更新本文档状态和实际偏差。
- 未达到验收条件时，不将 Phase 标记为完成。

建议提交顺序：

1. 安全与基线测试。
2. Server/Runtime 抽取。
3. Headless 入口迁移。
4. Desktop 技术验证。
5. Windows Desktop MVP。
6. Android 契约收敛。
7. Linux Desktop。
8. Release 整合。

## 15. 主要风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| 重构破坏现有 Headless 用户 | 高 | 保持 CLI/YAML 契约并建立回归测试 |
| Desktop 框架污染 Headless 依赖 | 高 | 两个 cmd 入口，Core 不引用 GUI 包 |
| Go 与 Android API 漂移 | 高 | 共享契约样例和跨端测试 |
| 浏览器无法播放已扫描格式 | 中 | 区分共享与播放能力，提供失败提示/外部链接 |
| Linux WebView/托盘差异 | 中 | Headless 保底，先技术验证再承诺发行版 |
| GUI 让用户误共享敏感文件 | 高 | 只允许媒体文件、禁止目录列表、显示共享范围 |
| Desktop 与 Headless 配置互相覆盖 | 中 | 明确配置优先级，UI 偏好与核心 YAML 分离 |
| 多端同时推进导致范围失控 | 高 | 按 Phase 顺序，Windows MVP 后再收敛 Android/Linux |

## 16. 对 LocalSend 的借鉴边界

参考项目：[LocalSend](https://github.com/localsend/localsend)

值得借鉴：

- 一个共享核心服务于多个应用入口。
- 安装模式和 Portable 模式并存。
- Desktop 使用侧边导航，移动端使用底部导航。
- 托盘、单实例和开机启动。
- 默认界面克制，高级网络设置折叠。
- 多平台使用一致的产品语言。

不直接照搬：

- Flutter + Rust 技术栈。
- 发送/接收式文件复制流程。
- 要求两端安装应用。
- LocalSend 协议、设备身份和传输会话模型。
- 与 YSeren 主线无关的复杂设置。

YSeren 应保留自己的差异：

> LocalSend 让文件在设备间传输，YSeren 让媒体无需搬运即可被其他设备访问。

## 17. 实施前仍需确认的决策

以下事项在对应 Phase 开始前确认，不阻塞当前计划归档：

- Android Phase 3 保持单来源并固定使用 `android` 作为契约来源名；是否扩展多来源留待后续反馈决定。
- Windows Desktop 首版是否包含二维码。
- Linux 首批支持的发行版和打包格式。
- 首个 Desktop 版本是否定为 v0.2.0。
- 是否在 Desktop MVP 后加入 mDNS 发现。

## 18. Phase 2 完成后的第一批任务

Windows MVP 完成后不立即并行铺开所有平台，下一批工作按以下顺序推进：

1. 固化 Windows Desktop 的自动化和手工验收记录，处理首批用户反馈，但不在当前阶段构建正式 Release。
2. Phase 3 的契约、第二代控制界面、模拟器验证和 Android 真机到 Windows Chrome 的局域网播放均已完成。
3. 网络切换、客户端中断、Unicode 文件名和大文件多次跳转继续作为发布前增强矩阵，不阻塞 Phase 3 完成状态。
4. 下一步进入 Phase 4，单独验证 Linux WebKitGTK、托盘、XDG 配置与首批分发格式。
5. Phase 5 再统一版本号、Git tag、Headless/Desktop/Android 构建和 SHA256 清单。
