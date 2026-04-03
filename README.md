# YSeren ✦

一个极轻量的本地媒体文件局域网访问工具，目标体验接近 nginx：**单个可执行文件**即可运行。

## 功能

- **/stream/**：把本地目录挂载为可播放的 HTTP 路由（浏览器/手机可直接播放，支持 Range）
- **/api/videos**：前端用的 JSON 列表接口（递归扫描、搜索、分页）
- **视频 + 音频**：默认支持 mp4/mkv/webm/mov/avi + mp3/flac/wav/aac/ogg 等格式
- **zip（MVP）**：在目录树中识别 `.zip`，由用户手动点击"解压"
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

## 开发环境

```bash
# 后端
go run .

# 前端（需要另开终端）
cd frontend && npm install && npm run dev
```

## 仓库信息

- 后端：Go
- 前端：Svelte + Vite
- 本地配置模板：[`yseren.example.yaml`](./yseren.example.yaml)
- 贡献说明：[`CONTRIBUTING.md`](./CONTRIBUTING.md)

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

## License

Apache-2.0. See [`LICENSE`](./LICENSE).

访问 `http://localhost:1479/` 或局域网 IP。
