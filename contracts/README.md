# YSeren 跨平台契约样例

此目录存放与实现语言无关的契约样例，用于约束 Go、Android 以及未来其他媒体源实现。

这些文件不是运行时配置，也不会被打包为用户数据。它们是自动化测试和跨端评审的共同输入。

## 版本规则

- 文件名包含契约主版本，例如 tree-response.v1.json。
- 只增加可选字段时，可以继续使用当前主版本。
- 删除字段、改变字段类型、改变时间单位或改变 URL 语义时，必须新增主版本。
- 生产实现可以包含额外字段，但不得缺少当前版本要求的字段。

## 当前样例

### fixtures/tree-response.v1.json

目录树响应的标准样例：

- generatedAt 和 modTime 使用 Unix 秒。
- size 使用字节。
- type 只能是 dir 或 file。
- mediaType 只能是 video 或 audio。
- relPath 始终使用正斜杠。
- 当前标准 URL 样例沿用桌面端的 /stream/{source}/{relativePath} 形式。

Android 在契约收敛阶段需要将现有 lastModified 毫秒字段对齐为 modTime 秒字段，并在必要时保留过渡兼容。

### fixtures/range-cases.v1.json

单文件 HTTP Range 的最小兼容矩阵：

- 完整 GET。
- HEAD。
- 固定起止 Range。
- 开放结束 Range。
- 后缀 Range。
- 超出文件范围的 416。

本版本不要求 multipart/byteranges。

## 使用方式

- Go 测试直接读取 contracts/fixtures。
- Android 单元测试应读取同一份样例或由构建任务复制到测试资源目录。
- 修改契约样例时，Go 与 Android 的相关测试必须在同一变更中更新。
