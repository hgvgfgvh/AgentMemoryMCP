# testdata

## `fixture_completed_todolist.json`

- **用途**：与 `internal/engine/fixture_completed_todolist.json` 保持同步的副本（`go:embed` 只能引用包内路径，编译嵌入以 `internal/engine/` 为准）。
- **形态**：对齐 AgentTest `plan/todolist.Document` 字段（非 MCP 协议强制 Schema）。
- **命中规则**：`context` 含样本 `user_requirement` 全文，或同时含 ≥2 个关键词（如 `WorkSpace`、`列出`、`目录`）。
- **来源说明**：仓库内无检入的 `WorkSpace/ToDoList/*.json` 时，用边界测试「Filesystem 列 WorkSpace 目录」场景手工固化此文件。

更新样本：直接编辑 JSON 后 `go test ./internal/engine/...`；或从 Host 复制已完成计划到本目录后改 `internal/engine/fixture.go` 的 embed 路径（开发期）。
