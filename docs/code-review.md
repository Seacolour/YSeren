# YSeren 代码审查报告

> 审查日期：2025-02-08
> 审查范围：全部后端 Go 代码 + 前端 Svelte 代码
> 设计定位：本地局域网流媒体代理工具——聚焦功能正确性与使用体验，弱化安全防御

---

## 问题总览

| # | 类别 | 文件 | 问题摘要 | 状态 |
|---|------|------|----------|------|
| 1 | 🔴 功能 Bug | `main.go` / `api_tree.go` / `api_videos.go` | `/stream/` 路由注册与 URL 生成的编码策略不一致，中文/特殊字符 source name 会 404 | ✅ 已修复 |
| 2 | 🔴 功能 Bug | `frontend/src/App.svelte` | 搜索时 `nav` 不更新，导致搜索结果无法正确显示 | ✅ 已修复 |
| 3 | 🟡 健壮性 | `api_tree.go` | `buildTreeForPath` 路径存在性检查在 WalkDir 之后，无意义 | ✅ 已修复 |
| 4 | 🟡 健壮性 | `config.go` | `Server.Port` 默认值为 0，未指定端口时行为不可预期 | ✅ 已修复 |
| 5 | 🟡 健壮性 | `logger.go` | `Logger` 初始化前调用会 nil panic | ✅ 已修复 |
| 6 | 🟡 健壮性 | `api_videos.go` | `WalkDir` 错误被静默吞掉，排障困难 | ✅ 已修复 |
| 7 | 🟢 代码质量 | `frontend/src/progress.svelte.js` | 混用 Svelte 5 runes 与 Svelte 4 store，订阅方式冗余 | ✅ 已修复 |
| 8 | 🟢 代码质量 | `Player.svelte` / `FileList.svelte` | `formatTime` 函数重复定义 | ✅ 已修复 |
| 9 | 🟢 待定 | `frontend/src/components/RecentGrid.svelte` | 组件已移除，排序功能已集成到列表 | ✅ 已完成 |

---

## 详细说明

### #1 � `/stream/` 路由编码策略不一致

- **文件**：`main.go:38` vs `api_tree.go:193` / `api_videos.go:172`
- **问题**：
  - `main.go` 注册路由时 source name **未做 URL 编码**：`fmt.Sprintf("/stream/%s/", source.Name)`
  - API 生成 streamURL 时使用了 `url.PathEscape(sourceName)`
  - 如果 source name 包含空格或中文（如 `我的动漫`），注册的路由与前端拿到的 URL **不匹配**，导致 404
- **影响**：中文 source name 时流媒体完全不可用
- **修复方案**：统一编码策略——两处都用相同的方式处理 source name

---

### #2 � 搜索时 `nav` 状态不更新

- **文件**：`frontend/src/App.svelte:77-80`
- **问题**：搜索时 `fetchTree` 不更新 `nav`，但搜索结果返回的是一棵新的裁剪树。此时 `nav` 仍指向旧树的节点引用，`currentDir` 的 `children` 与实际搜索结果不匹配，用户可能看到空列表或旧数据
- **影响**：搜索功能体验不佳，结果展示可能为空
- **修复方案**：搜索时将 `nav` 重置为 `[treeRoot]`，让用户从搜索结果的根节点开始浏览

---

### #3 🟡 `buildTreeForPath` 路径存在性检查顺序错误

- **文件**：`api_tree.go:150-217`
- **问题**：`filepath.WalkDir`（第 150 行）已执行完毕，**之后**第 212 行才检查 `srcPath` 是否存在——此时检查毫无意义
- **修复方案**：将路径存在性检查移到 `WalkDir` 之前，不存在时记录日志并返回空树

---

### #4 � `Server.Port` 默认值为 0

- **文件**：`config.go:19-27`
- **问题**：YAML 中未指定 `port` 时，`conf.Server.Port` 默认为 `0`，Go 会随机分配端口，用户无法预知访问地址
- **修复方案**：在 `LoadConfig` 中添加默认端口 `1479`

---

### #5 � `Logger` 初始化前调用会 nil panic

- **文件**：`logger.go:9`
- **问题**：`Logger` 全局变量初始值为 `nil`。`startErrorServer` 路径中没有初始化 Logger，后续如果在该路径中加日志调用就会 panic
- **修复方案**：`var Logger = slog.Default()`

---

### #6 � `WalkDir` 错误被静默吞掉

- **文件**：`api_videos.go:185-187`
- **问题**：`err != nil` 时既不记录日志也不返回错误，完全静默，排障困难
- **修复方案**：添加 `LogWarn` 日志记录

---

### #7 🟢 混用 Svelte 5 runes 与 Svelte 4 store

- **文件**：`frontend/src/progress.js`、`frontend/src/components/FileList.svelte:25-29`
- **问题**：项目全面使用 Svelte 5 runes，但 `progress.js` 使用 Svelte 4 的 `writable` store，在 `FileList.svelte` 中需要手动 `subscribe` 桥接，代码冗余
- **修复方案**：迁移为 Svelte 5 模块级 `$state` 响应式状态

---

### #8 🟢 `formatTime` 函数重复定义

- **文件**：`Player.svelte:91-98`、`FileList.svelte:45-52`
- **问题**：两个文件各自定义了完全相同的 `formatTime` 函数
- **修复方案**：提取到 `frontend/src/utils.js`

---

### #9 🟢 `RecentGrid` 组件待定

- **文件**：`frontend/src/components/RecentGrid.svelte`
- **问题**：组件已编写但未在 `App.svelte` 中使用。`progress.js` 已有播放进度记录能力，具备集成"最近播放"功能的基础
- **待定**：决定是否集成该功能，或暂时移除死代码

---

## 代码质量总评

| 维度 | 评价 |
|------|------|
| 整体架构 | ✅ 清晰简洁，Go embed + Svelte SPA 单文件交付，符合"利刃"定位 |
| 功能正确性 | ⚠️ URL 编码不一致是最可能导致实际 bug 的问题 |
| 错误处理 | ⚠️ 部分错误被静默忽略，不利于排障 |
| 缓存设计 | ✅ singleflight + TTL 缓存设计合理 |
| 前端代码 | ✅ Svelte 5 runes 使用正确，组件划分清晰；store 混用可优化 |

---

## 推荐修复顺序

1. **#1** URL 编码不一致（功能 Bug，影响中文 source name）
2. **#2** 搜索 nav 不更新（功能 Bug，影响搜索体验）
3. **#5** Logger 默认值（一行改动，防 panic）
4. **#4** 端口默认值（一行改动，改善首次使用体验）
5. **#3** 路径检查顺序（小改动，改善日志可读性）
6. **#6** WalkDir 错误日志（一行改动）
7. **#7** progress.js 迁移 Svelte 5（代码质量）
8. **#8** formatTime 提取（代码质量）
9. **#9** RecentGrid 去留（待讨论）
