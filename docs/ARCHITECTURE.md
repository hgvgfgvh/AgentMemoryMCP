# 架构地图（Phase-1 · 与 DESIGN_INTENT 对齐）

> 当前为 **1-stub** 阶段：MCP 工具面完整；事实世界引擎为占位实现。

## 总览

```text
Host（AgentTest 等，Phase-2）
  │ 钩子 OnTurnRetrieve / OnTurnStore（不经过执行 Agent tool_calls）
  ▼
MCP Client（stdio 或 HTTP）
  ▼
cmd/memory-mcp/main.go
  ├── memory_store   → engine.Engine.Store
  └── memory_retrieve → engine.Engine.Retrieve
        ▼
internal/engine/StubEngine   ← Phase-2 替换为 FactWorldEngine + Memory Agent
internal/filter              ← 寒暄 / 空输入（零 LLM）
internal/response            ← 统一 JSON 字符串响应
data/episodes_stub.jsonl     ← Phase-1 仅审计队列，非事实图
```

## 模块职责

| 路径 | 职责 |
|------|------|
| `cmd/memory-mcp` | MCP Server 入口；注册工具；stdio / Streamable HTTP |
| `internal/engine` | `Engine` 接口；`StubEngine` 异步写 JSONL、同步 stub retrieve |
| `internal/filter` | store/retrieve 双侧轻量过滤 |
| `internal/response` | 工具返回 JSON 编码（字段 string 语义） |

## MCP 工具契约

### memory_store

- **入参**：`content`（required string），`source`，`kind`，`correlation_id`（optional string）
- **出参**：单段 **TextContent**，内容为 JSON 字符串，例如：
  `{"accepted":"true","job_id":"job-...","skipped":"false","message":"...","phase":"1-stub"}`
- **行为（Phase-1）**：过滤 → 同步 ACK → goroutine 追加 `episodes_stub.jsonl`（无 Memory Agent）

### memory_retrieve

- **入参**：`context`（required string），`query_hint`（optional string）
- **出参**：JSON 字符串，例如：
  `{"hints":"【跨会话事实参考】\n...","skipped":"false","phase":"1-stub"}`
- **行为（Phase-1）**：过滤 → 读 stub 队列计数 → 返回占位 `hints`（无图遍历、无 LLM）

## 传输

| 模式 | 启动 |
|------|------|
| stdio | 默认，`mcp.StdioTransport` |
| HTTP | `-http 127.0.0.1:8090`，`mcp.NewStreamableHTTPHandler` |

## Phase-2 规划（未实现）

- `FactWorldEngine`：Memory Agent 流水线（解析 → 去重 → 连边 → 索引）
- 图存储与退化策略
- retrieve 热路径：索引 + 轻量 rerank（禁止多轮 Agent）
- 租户 / `source` 分库

详见 `ARCHITECTURE_DRIFT.md`。
