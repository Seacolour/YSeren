# lv-link 项目优化清单

## 后端优化

### Phase 1: 代码整理 (简单)
- [ ] **1.1 抽取 Source 查找逻辑**
  - 将 `api_tree.go`、`api_videos.go`、`api_zip.go` 中重复的 source 查找代码抽取到 `config.go`
  - 新增 `(c *Config) GetSourcePath(name string) (string, bool)` 方法

- [ ] **1.2 统一错误响应格式**
  - 新增 `writeError(w, code, msg)` 函数
  - 所有 API 返回统一的 JSON 错误格式 `{"error": "message"}`

- [ ] **1.3 抽取路径安全校验**
  - 将 ZipSlip 防护逻辑抽取为公共函数 `isPathSafe(basePath, targetPath) bool`

### Phase 2: 功能增强 (中等)
- [ ] **2.1 视频格式可配置化**
  - 在 `v-link.yaml` 中支持 `video_extensions` 配置项
  - 保持默认值兼容现有行为

- [ ] **2.2 缓存优化 (singleflight)**
  - 引入 `golang.org/x/sync/singleflight` 防止缓存击穿
  - 优化 `treeCache` 和 `videosCache` 的并发安全性

- [ ] **2.3 添加结构化日志**
  - 使用 Go 1.21+ 的 `log/slog` 替代 `fmt.Printf`
  - 支持日志级别配置

### Phase 3: 生产就绪 (高级)
- [ ] **3.1 Graceful Shutdown**
  - 使用 `context` + `signal.NotifyContext` 实现优雅关闭
  - 等待进行中的请求完成

- [ ] **3.2 HTTPS 支持 (可选)**
  - 配置文件支持 TLS 证书路径
  - 自动 HTTP -> HTTPS 重定向

---

## 前端优化

### Phase 4: 组件拆分
- [ ] **4.1 创建 Topbar 组件**
  - 提取搜索栏和品牌 logo

- [ ] **4.2 创建 Player 组件**
  - 提取视频播放器相关逻辑

- [ ] **4.3 创建 FileList 组件**
  - 提取文件列表（目录/zip/视频）

- [ ] **4.4 创建 Breadcrumb 组件**
  - 提取面包屑导航

- [ ] **4.5 创建 RecentGrid 组件**
  - 提取"最近播放"网格

### Phase 5: 样式优化
- [ ] **5.1 提取 CSS 变量**
  - 将重复颜色值抽取为 CSS 变量
  - 创建统一的设计 token

- [ ] **5.2 添加 Loading 骨架屏**
  - 优化加载状态体验

---

## 测试与文档

### Phase 6: 质量保障
- [ ] **6.1 添加后端单元测试**
  - 核心函数测试覆盖

- [ ] **6.2 更新 README**
  - 补充配置项说明
  - 添加架构图
