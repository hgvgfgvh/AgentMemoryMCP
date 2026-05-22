# AgentTestMemoryMCP

**记忆系统 MCP**（第三层事实世界）。改代码前必读 [`docs/DESIGN_INTENT.md`](docs/DESIGN_INTENT.md)。

## Phase-1（当前）

- 实现：`cmd/memory-mcp` + `internal/engine/StubEngine`
- 工具：`memory_store`、`memory_retrieve`（字符串 / JSON 字符串返回）
- **未实现**：Memory Agent、图结构、向量索引、退化策略

## 构建

```powershell
go build -o memory-mcp.exe ./cmd/memory-mcp
```

## 文档

| 文件 | 角色 |
|------|------|
| `docs/DESIGN_INTENT.md` | 宪法 |
| `docs/ARCHITECTURE.md` | 实现地图 |
| `docs/ACCEPTANCE_RULES.md` | 可验收规则 |
| `docs/ARCHITECTURE_DRIFT.md` | 已知差距 |

## 约束

- 不得新增 Host 可见的图编辑 API。
- Phase-1 勿在 AgentTest 仓库内改 Host 钩子（Phase-2 另项）。
