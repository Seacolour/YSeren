# YSeren ✦

一个极轻量的"本地媒体文件局域网访问工具"（目标体验类似 nginx：**单个可执行文件**即可运行）。

## 功能

- **/stream/**：把本地目录挂载为可播放的 HTTP 路由（浏览器/手机可直接播放，支持 Range）
- **/api/videos**：前端用的 JSON 列表接口（递归扫描、搜索、分页）
- **视频 + 音频**：默认支持 mp4/mkv/webm/mov/avi + mp3/flac/wav/aac/ogg 等格式
- **zip（MVP）**：在目录树中识别 `.zip`，由用户手动点击"解压"
- **前端 UI（Svelte）**：手机适配的文件浏览/搜索/最近播放
- **单文件交付**：Go `embed` 把 `frontend/dist` 打进 exe

## 配置

默认读取配置文件：**当前目录**或 **exe 同目录**的 `yseren.yaml` / `yseren.yml`。

```bash
yseren.exe -config D:/path/to/yseren.yaml
```

示例 `yseren.yaml`：

```yaml
server:
  port: 1479
  log_level: info  # debug, info, warn, error

sources:
  - path: "D:/Videos"

# 可选：自定义支持的媒体格式
# media_extensions:
#   - .mp4
#   - .mp3
```

## 开发

```bash
# 后端
go run .

# 前端（需要另开终端）
cd frontend && npm install && npm run dev
```

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

访问 `http://localhost:1479/` 或局域网 IP。
