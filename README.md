# YSeren ✦

一个极轻量的本地媒体文件局域网访问工具，目标体验接近 nginx：**单个可执行文件**即可运行。

## 功能

- **/stream/**：把本地目录挂载为可播放的 HTTP 路由（浏览器/手机可直接播放，支持 Range）
- **/api/videos**：前端用的 JSON 列表接口（递归扫描、搜索、分页）
- **视频 + 音频**：默认支持 mp4/mkv/webm/mov/avi + mp3/flac/wav/aac/ogg 等格式
- **前端 UI（Svelte）**：手机适配的文件浏览/搜索/最近播放
- **单文件交付**：Go `embed` 把 `frontend/dist` 打进 exe

## 快速开始

默认读取配置文件：**当前目录**或 **exe 同目录**的 `yseren.yaml` / `yseren.yml`。

仓库内提供了示例配置 [`yseren.example.yaml`](./yseren.example.yaml)。实际运行时请使用本地 `yseren.yaml` 或 `yseren.yml`，该文件默认不会提交到 git。

```bash
yseren.exe -config D:/path/to/yseren.yaml
```

示例配置：

```yaml
server:
  port: 1479
  log_level: info  # debug, info, warn, error

sources:
  - name: videos
    path: "D:/Videos"

# 可选：自定义支持的媒体格式（完整扫描名单）
# media_extensions:
#   - .mp4
#   - .mp3
#
# 可选：补充“应按音频播放器处理”的扩展名
# 不写 media_extensions 也可以，audio_extensions 会自动加入扫描名单
# audio_extensions:
#   - .opus
#   - .mka
```

如果你要新增自定义音频格式，优先写 `audio_extensions`。这样前端会继续用音频播放器，而不会误判成视频播放器。

注意：YSeren 只识别媒体文件，不识别也不处理压缩包。像 `.zip` 这类文件请先用系统文件管理器或其他工具解压，再让 YSeren 共享解压后的目录内容。

## 开发环境

```bash
# 后端
go run .

# 前端（需要另开终端）
cd frontend && npm install && npm run dev
```

## Android 方向

仓库现在开始包含一个 Android 侧的 MVP 脚手架，目标不是“手机播放端”，而是“手机作为媒体源，对局域网暴露 HTTP 流媒体”。

- Android 项目入口：[`android/README.md`](./android/README.md)
- 方案说明：[`docs/android-share-mvp.md`](./docs/android-share-mvp.md)
- 当前 Android MVP 端点：
  - `/`（APK 内置 Web UI）
  - `/api/status`
  - `/api/tree`
  - `/api/tree?path=...`
  - `/playlist.m3u`
  - `/stream/<relative-path>`

当前这部分仍处于原型阶段，核心路线是：
- 用 Android Storage Access Framework 选择目录
- 用 foreground service 持续共享
- 用轻量本地 HTTP 服务把目录内容扩散到局域网

## 仓库信息

- 后端：Go
- 前端：Svelte + Vite
- 本地配置模板：[`yseren.example.yaml`](./yseren.example.yaml)
- 贡献说明：[`CONTRIBUTING.md`](./CONTRIBUTING.md)
- 变更日志：[`CHANGELOG.md`](./CHANGELOG.md)
- 安全策略：[`SECURITY.md`](./SECURITY.md)

## 构建

```bash
# 一键构建（推荐，会自动先 build 前端再 go build）
.\build.ps1
# 或者双击 build.cmd

# 也支持自定义输出名
.\build.ps1 -Out yseren-custom.exe

# 如果你坚持手动构建，记得顺序必须是：
# 1) cd frontend && npm ci && npm run build && cd ..
# 2) go build -o yseren.exe
```

## 校验

```bash
go test ./...
go vet ./...

cd frontend && npm run build
```

## Releases

- 建议通过 GitHub Releases 分发正式版本。
- 推荐使用 `vX.Y.Z` 这样的 tag，例如 `v0.1.0`。
- 仓库内已提供 GitHub Actions 发布工作流：推送 `v*` tag 后会自动构建发布包并上传到 Release。
- Release 产物会包含二进制、示例配置、`README`、`LICENSE` 和 `SHA256SUMS.txt`。
- Android release workflow 会同时构建 APK；如仓库配置了 Android 签名 secrets，会上传签名 APK，否则上传 unsigned APK 作为测试/侧载产物。

## Packages

- 当前仓库以 GitHub Releases 为主要分发方式。
- 目前没有额外接入 GitHub Packages，因为这个项目更适合直接发布可执行文件。
- 如果后续需要更方便的安装体验，建议优先考虑 `winget`、`Scoop` 或容器镜像分发。

## License

Apache-2.0. See [`LICENSE`](./LICENSE).

访问 `http://localhost:1479/` 或局域网 IP。
