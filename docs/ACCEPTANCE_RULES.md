# 验收规则（Phase-1）

可执行检查优先于 prompt 约定。

## AR-1 工具面

- [ ] MCP `tools/list` 仅包含 `memory_store`、`memory_retrieve`（无图编辑类工具）
- [ ] 两工具必填参数均为 string 语义（`content` / `context`）

## AR-2 返回格式

- [ ] 工具结果为 **单一 TextContent**，正文为合法 JSON 字符串
- [ ] `memory_store` 含 `accepted`、`job_id`、`skipped`（值均为 string）
- [ ] `memory_retrieve` 含 `hints`、`skipped`（值均为 string）

## AR-3 store 语义

- [ ] 非过滤内容：`accepted` 为 `"true"` 且 `job_id` 非空
- [ ] 寒暄内容：`skipped` 为 `"true"`，且不追加 episode 文件
- [ ] 调用立即返回（不等待 Memory Agent；Phase-1 无 Agent）

## AR-4 retrieve 语义

- [ ] 同步返回（无额外 HTTP 往返以外的长阻塞）
- [ ] 寒暄：`hints` 为空且 `skipped` 为 `"true"`
- [ ] 失败或空库：不导致 MCP 进程崩溃；`hints` 可为占位说明

## AR-5 与 Host 集成约束（Phase-2 验收）

- [ ] AgentTest **不**将本 server 列入 `capabilities.attach_to`
- [ ] Host 钩子调用，执行 Agent 能力目录**不出现** memory 工具名

## 验证命令

```powershell
cd C:\DATA\GODATA\AgentTestMemoryMCP
go test ./...
go build -o memory-mcp.exe ./cmd/memory-mcp
```
