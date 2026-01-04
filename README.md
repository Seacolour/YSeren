# lv-link

一个极轻量的“本地视频文件局域网访问工具”（目标体验类似 nginx：**单个可执行文件**即可运行）。

## 功能

- **/stream/**：把本地目录挂载为可播放的 HTTP 路由（浏览器/手机可直接播放，支持 Range）
- **/api/videos**：前端用的 JSON 列表接口（递归扫描、搜索、分页）
- **前端 UI（Svelte）**：手机适配的海报墙/搜索/最近播放
- **单文件交付**：Go `embed` 把 `frontend/dist` 打进 exe

## 配置

编辑 `v-link.yaml`：

```yaml
server:
  port: 1479

sources:
  # 你可以只挂载一个根目录，程序会递归扫描所有子目录，并保留 relPath 层级
  - path: "D:/Videos"

  # 也可以显式命名（可选）
  # - name: "videos"
  #   path: "D:/Videos"
```

## 开发/运行

后端：

```bash
go run .
```

前端（开发模式，需要单独起 Vite dev server）：

```bash
cd frontend
npm install
npm run dev
```

## 构建（推荐：单 exe）

1) 构建前端到 `frontend/dist`

```bash
cd frontend
npm install
npm run build
```

2) 回到项目根目录，构建后端 exe

```bash
cd ..
go build -o lv-link.exe
```

运行后访问：

- `http://<你的电脑IP>:1479/`（手机 UI）
- `http://<你的电脑IP>:1479/api/videos`（JSON）
- `http://<你的电脑IP>:1479/stream/<source>/...`（直链播放）


