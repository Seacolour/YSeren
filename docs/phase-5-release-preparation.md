# YSeren Phase 5 发布准备

- 状态：发布候选验证中
- 基线日期：2026-07-24
- 基线提交：`0b7a429`（Phase 4-a 完成）
- 目标：在不引入 Linux Desktop GUI 的前提下，把 Headless、Windows Desktop 和 Android 整合为可重复、可验证的同版本 Release

本文是 Phase 5 的执行与验收记录。当前尚未创建 tag 或公开正式版本。

## 1. 建议版本与发布范围

首个多入口整合版本建议使用 `v0.2.0`。这是在保持 Headless 兼容的同时新增 Windows Desktop、第二代 Android 媒体源和 Linux Headless 正式验证，符合 SemVer 的次版本升级语义。开始发布前仍需最终确认版本号。

首轮整合发布只承诺已经完成并具备验证条件的入口：

| 产品 | 平台 | 架构 | 首轮状态 |
| --- | --- | --- | --- |
| YSeren Desktop | Windows 10/11 | x64 | 纳入 |
| YSeren Headless | Windows | x64 | 纳入 |
| YSeren Headless | Linux | x64 | 纳入 |
| YSeren Headless | Linux | arm64 | 纳入，但明确标记为“交叉构建通过，真机运行未验证” |
| YSeren Android | Android 8.0+ | arm64/常规 APK ABI | 纳入 |
| YSeren Desktop | Linux | x64 | Phase 4-b，暂不纳入 |
| YSeren Headless | Windows | arm64 | 暂不纳入，缺少运行验证 |
| YSeren Headless | macOS | x64/arm64 | 暂不纳入，不属于当前 Windows、Android、Linux 支持范围 |
| YSeren iOS | iOS | - | 暂不纳入 |

推荐的 Release 文件名：

```text
YSeren-v0.2.0-Desktop-Windows-x64-Setup.exe
YSeren-v0.2.0-Desktop-Windows-x64-Portable.zip
YSeren-v0.2.0-Headless-Windows-x64.zip
YSeren-v0.2.0-Headless-Linux-x64.tar.gz
YSeren-v0.2.0-Headless-Linux-arm64.tar.gz
YSeren-v0.2.0-Android.apk
SHA256SUMS.txt
```

正式实现时由统一版本变量替换示例中的 `v0.2.0`，禁止在各构建任务中分别手写版本。

## 2. 当前可复用的发布基础

### 2.1 已完成能力

- Phase 4-a 基线提交 `0b7a429` 的远端 CI 已通过，包括前端构建、Go test/vet/build、Linux Headless smoke 和 Android debug APK。
- Headless 保持单可执行文件和旧 YAML 契约，Windows 与 Linux 共用同一 Go Core。
- Windows Desktop 已完成目录选择、共享启停、浏览器打开、托盘、开机启动、YAML 导入导出、当前用户安装器和 Portable 配置模式。
- Android 已完成 SAF 目录授权、前台服务、第二代控制界面、内置 Web Player、标准 HTTP 契约和真机跨设备播放。
- Linux Headless 已在 Ubuntu 24.04 WSL2 中完成原生构建、HTTP、Range、权限、符号链接、信号和干净归档冒烟。
- Release 工作流已有 Headless、Android、SHA256SUMS 和 GitHub Release 的基础实现。
- GitHub 仓库已配置 Android Release 所需的四个签名 secret 名称；发布前仍需通过一次升级安装验证确认签名连续性。

### 2.2 已存在的本地候选产物

这些文件只用于开发验证，不能直接当作正式 Release：

```text
desktop/build/bin/YSeren-Desktop-Windows-amd64-Setup.exe
desktop/build/bin/YSeren.exe
android/app/build/outputs/apk/debug/app-debug.apk
android/app/build/outputs/apk/release/app-release-unsigned.apk
```

正式产物必须由同一个 tag 对应的 GitHub Actions 工作流重新构建。

## 3. 进入 Release 前必须补齐的差距

### 3.1 统一版本契约

当前版本来源并不统一：

| 入口 | 当前行为 | Phase 5 要求 |
| --- | --- | --- |
| Headless | `main.Version` 默认 `dev`，Release 使用 ldflags；本地 `build.ps1` 会从非精确 tag 的 `git describe` 误取最近版本 | 只有精确 SemVer tag 才显示正式版本，其他构建统一显示 `dev` |
| Windows Desktop | Go 层 `Version` 默认 `dev`，`wails.json` 的 Windows 产品版本固定为 `0.0.0` | Go API、界面版本、EXE 元数据和安装器元数据全部来自同一个 tag |
| Android | `versionName = "0.1.2"`、`versionCode = 2` 写死在 Gradle | Release 通过 Gradle 属性注入 `versionName` 和递增的 `versionCode` |
| Release workflow | `workflow_dispatch` 可接受任意 `tag_name` | 只接受 `vMAJOR.MINOR.PATCH`，并确认 tag 指向当前构建提交 |

建议采用以下单一规则：

1. tag 必须匹配 `^v[0-9]+\.[0-9]+\.[0-9]+$`。
2. 工作流只解析一次，得到：
   - `RELEASE_TAG=v0.2.0`
   - `RELEASE_VERSION=0.2.0`
3. 非精确 tag 构建一律使用 `dev`，不再回退到最近的历史 tag。
4. Headless 使用 `-X main.Version=$RELEASE_VERSION`。
5. Desktop 同时把 `$RELEASE_VERSION` 注入 `main.Version` 和 Wails `productVersion`。
6. Android 通过 Gradle 属性注入版本。建议使用
   `major * 1,000,000 + minor * 1,000 + patch` 生成 `versionCode`；
   `v0.2.0` 对应 `2000`，高于现有的 `2`。
7. 自动化分别验证 Headless `/api/version`、Desktop EXE 产品版本和 Android APK `versionName/versionCode`。

### 3.2 Release 工作流

当前 `.github/workflows/release.yml` 尚缺 Windows Desktop 构建。Phase 5 应拆成以下职责：

1. `prepare`
   - 校验 tag 和提交关系。
   - 输出统一版本变量和 Android `versionCode`。
2. `headless`
   - 构建 Windows x64、Linux x64、Linux arm64。
   - 对 Linux x64 的干净归档执行现有 smoke 脚本。
3. `desktop-windows`
   - 在 `windows-latest` 安装 Go、Node.js、Wails `v2.12.0` 和 NSIS。
   - 先构建共享 Web Player，再构建 Desktop 自身前端。
   - 注入 Go 版本与 Windows 产品版本。
   - 生成 Setup EXE 和 Portable ZIP。
4. `android`
   - 运行单元测试和 Release lint。
   - 注入统一 `versionName/versionCode`。
   - 使用现有仓库 secrets 构建签名 APK。
   - 正式 tag 不允许静默降级为 unsigned APK。
5. `assemble`
   - 下载全部产物。
   - 校验文件名、版本和预期数量。
   - 生成覆盖所有发布文件的 `SHA256SUMS.txt`。
   - 创建 Draft Release，待手工验收后再发布。

`workflow_dispatch` 应默认只构建候选产物，不直接创建公开 Release；tag 流程也优先创建 Draft Release，避免未经手工检查就公开错误安装包。

### 3.3 CI 门禁

合并 Phase 5 前，CI 至少增加：

- Desktop `go test ./...` 和 `go vet ./...`。
- Desktop Svelte 生产构建。
- Android `:app:testDebugUnitTest`。
- Android `:app:lintVitalRelease`。
- 版本解析与 SemVer tag 拒绝测试。
- Release 脚本的无发布 dry-run 或产物命名测试。

Windows Desktop 的 NSIS 完整打包可以放在独立 Windows job，避免只在本地机器上可复现。

## 4. 分发体验材料

### 4.1 README 下载选择

README 需要在 Releases 之前先回答“我该下载哪一个”：

| 用户场景 | 推荐下载 |
| --- | --- |
| 普通 Windows 用户，希望选择文件夹后直接使用 | Desktop Windows x64 Setup |
| Windows 用户，不希望安装 | Desktop Windows x64 Portable |
| 熟悉 YAML 或需要脚本化运行 | Headless Windows x64 |
| Linux 服务器、NAS、WSL 或无桌面环境 | Headless Linux 对应架构 |
| 把 Android 手机或平板作为媒体源 | Android APK |

同时明确：

- 播放仍由浏览器承担，不需要在播放设备安装 YSeren。
- Linux Desktop GUI 尚未提供。
- YSeren 不转码，浏览器能否播放仍取决于媒体编码。
- 所有共享都应只在可信局域网中开启。

### 4.2 三条最短启动路径

正式文档至少提供以下三条不超过一分钟的路径：

1. Windows Desktop：安装 → 选择媒体目录 → 点击局域网地址或“在浏览器打开”。
2. Headless：解压 → 复制并编辑示例 YAML → 启动二进制 → 打开输出地址。
3. Android：安装 APK → 授权媒体目录 → 开始共享 → 在另一台设备打开局域网地址。

### 4.3 截图清单

Phase 5 需要重新截取干净、可公开的画面，避免包含个人目录、无关浏览器标签或过时 UI：

- Windows Desktop“共享”页：显示运行状态、一个演示媒体源和浏览器打开入口。
- Windows Desktop“媒体源”页：显示选择/管理目录能力。
- Android“共享”页：显示运行状态、局域网地址和一个演示媒体源。
- Web Player 文件浏览页：同时展示音频和视频条目。
- Web Player 播放页：使用已经移除自建倍速按钮的最新界面。

建议截图统一使用演示名称，例如 `Movies`、`Music`，并避免暴露真实用户名、个人文件名和不相关应用。

### 4.4 首次运行与排障

README 或独立文档必须覆盖：

- Windows Defender Firewall 首次弹窗应允许“专用网络”，不建议允许公共网络。
- 两台设备必须位于可互访的同一局域网；访客 Wi-Fi、AP 隔离和模拟器 NAT 可能阻断访问。
- 端口被占用时如何更换默认 `1479`。
- Windows SmartScreen 对未代码签名安装包的提示。
- Android 前台通知、后台限制和电池优化对持续共享的影响。
- Linux 防火墙开放 TCP 端口的最小示例。
- 校验 `SHA256SUMS.txt` 的 Windows PowerShell 与 Linux 命令。

## 5. 发布前验证记录模板

每次正式发布复制下表并填写实际证据，不能只记录“构建成功”：

| 编号 | 产物/链路 | 验证内容 | 结果 | 证据 |
| --- | --- | --- | --- | --- |
| V01 | Windows Desktop Setup | 全新安装、首次选目录、共享、浏览器播放、卸载 | 待验证 | |
| V02 | Windows Desktop Upgrade | 从上一正式版升级，配置和偏好保留 | 待验证 | |
| V03 | Windows Desktop Portable | 解压运行、同目录 YAML、无安装依赖 | 待验证 | |
| V04 | Windows Headless | 旧版 YAML 直接启动、Range 播放、优雅退出 | 待验证 | |
| V05 | Linux x64 Headless | 干净解压、smoke、移动设备浏览器访问 | 待验证 | |
| V06 | Linux arm64 Headless | ELF 架构与静态链接；真机状态明确标注 | 待验证 | |
| V07 | Android APK | 使用 Release 签名安装并共享媒体 | 待验证 | |
| V08 | Android Upgrade | 从 `v0.1.2` 原包覆盖安装，目录授权和设置保留 | 待验证 | |
| V09 | 跨设备播放 | Windows → Android 浏览器、Android → Windows 浏览器 | 待验证 | |
| V10 | 版本一致性 | API、Desktop 元数据、APK 和文件名均为同一版本 | 待验证 | |
| V11 | 完整性 | 下载全部资产并通过 SHA256 校验 | 待验证 | |
| V12 | 文档冷启动 | 未参与开发的用户按 README 完成首次共享 | 待验证 | |

Linux arm64 在获得真实设备前可以通过构建门禁，但 Release 说明必须保留“运行未验证”，不能写成已完成真机验收。

## 6. 已知发布决策

Phase 5 实现时采用以下默认方向；如有新条件再调整：

- 候选版本：`v0.2.0`。
- GitHub Release：先生成 Draft，再手工公开。
- Android：正式 Release 必须签名，不上传 unsigned APK 冒充正式包。
- Windows：当前没有代码签名证书；首轮可发布未签名安装包，但必须在下载说明中解释 SmartScreen，并提供 SHA256。
- Linux Desktop GUI：不阻塞首轮整合发布。
- macOS、iOS、Windows arm64：不在首轮支持矩阵中。
- Linux arm64：允许提供交叉构建产物，但必须披露没有真机运行证据。

## 7. 推荐实施顺序

1. 完成统一版本解析、注入和自动化测试。
2. 扩充 CI，并让 Desktop 在 Windows runner 上可重复打包。
3. 重构 Release 工作流，完成全部产物和 SHA256 的 dry-run。
4. 更新 README、CHANGELOG、最短启动路径和排障说明。
5. 重拍并加入公开截图。
6. 使用候选产物执行完整验证矩阵。
7. 确认 `v0.2.0`，创建 tag 和 Draft Release。
8. 核对 Draft 资产、版本、签名、校验和与说明后手工发布。

第一批代码工作应只处理第 1 步“统一版本”，避免同时修改三端版本、CI、打包和文档而难以定位问题。

## 8. v0.2.0 发布候选验证进度

截至 2026-07-29，本地已完成：

- Web Player 和 Desktop 前端均使用 Vite `8.1.5`，`npm audit` 为 0 个已知漏洞，生产构建通过。
- Go 全量测试、vet、显式 `0.2.0` 构建和非 tag `dev` 构建通过；两个实际 Windows Headless 二进制的 `/api/version` 均返回预期值。
- Android 在 JDK 17 下通过单元测试、Release lint 和 unsigned Release 构建，产物为 `versionName=0.2.0`、`versionCode=2000`。
- Desktop Go 测试、vet、Svelte 构建和 Wails EXE 构建通过；EXE 的固定 File/Product Version 均为 `0.2.0.0`。
- Ubuntu 24.04 WSL2 中的 Go 测试、vet、Linux x64 版本感知 smoke 通过；Linux arm64 为静态 aarch64 ELF。
- GitHub Actions workflow 已通过 actionlint；README 本地链接、截图路径和 `git diff --check` 通过。

远端 Windows NSIS 打包、签名 Android APK、完整候选资产与 SHA256 仍由本次提交后的 GitHub Actions 验证。当前机器没有 NSIS，因此本地只验证 Wails EXE；正式安装包不得绕过远端 Windows job。
