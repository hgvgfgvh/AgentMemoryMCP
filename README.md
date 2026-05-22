# AgentTestMemoryMCP

跨 Host 可复用的**长效事实记忆** MCP Server（第三层记忆）。Phase-1 仅搭建**对外 MCP 架构**；内部记忆 Agent、图结构、向量索引等**未实现**（见 `docs/ARCHITECTURE_DRIFT.md`）。

## 工具（字符串协议）

| 工具 | 说明 |
|------|------|
| `memory_store` | 存入：`content`（必填），`source` / `kind` / `correlation_id`（可选）。返回 **JSON 字符串**（`accepted`、`job_id`、`skipped` 等均为 string）。 |
| `memory_retrieve` | 取出：`context`（必填），`query_hint`（可选）。返回 **JSON 字符串**（`hints`、`skipped` 等均为 string）。 |

宪法与设计意图：[`docs/DESIGN_INTENT.md`](docs/DESIGN_INTENT.md)

## 构建与运行

```powershell
cd C:\DATA\GODATA\AgentTestMemoryMCP
go mod tidy
go build -o memory-mcp.exe ./cmd/memory-mcp
```

**stdio（供 Cursor / AgentTest `plan_memory_hook` 挂载）：**

```powershell
# 默认 factworld 引擎（规则抽取 + JSONL 事实库 + 关键词检索）
.\memory-mcp.exe

# 测试引擎：内嵌已完成 TodoList 样本（CI/回归）
.\memory-mcp.exe -engine test

# Phase-1 stub
.\memory-mcp.exe -engine stub

# 环境变量 MEMORY_MCP_ENGINE=factworld|test|stub
```

**HTTP（调试）：**

```powershell
.\memory-mcp.exe -http 127.0.0.1:8090
```

环境变量：

- `MEMORY_MCP_DATA_DIR`：stub 队列目录（默认 `./data`）

## AgentTest 挂载示例（Host 侧，Phase-2）

```yaml
# config/app.yaml capabilities.mcp.servers 片段（勿 attach_to planAgent/behaviorAgent）
- name: memory
  enabled: true
  description: "长效事实记忆：Host 钩子 store/retrieve，非执行 Agent 工具"
  command: "C:\\DATA\\GODATA\\AgentTestMemoryMCP\\memory-mcp.exe"
  args: []
```

Host 通过进程内 Client 在 `RunRouterTurn` 钩子调用 `memory_store` / `memory_retrieve`，不由 Plan/Behavior 主动 tool_calls。

## 测试

```powershell
go test ./...
```
