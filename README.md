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
# 同时可打开开发控制台：http://127.0.0.1:8090/console/
```

**记忆拓扑开发控制台（只读，与 stdio MCP 并行）：**

AgentTest 主进程仍用 stdio 挂载 MCP 时，可**另开终端**只读同一 `data` 目录：

```powershell
$env:MEMORY_MCP_DATA_DIR = "C:/DATA/GODATA/AgentTestMemoryMCP/data"
.\memory-mcp.exe -console 127.0.0.1:8091
# 浏览器打开 http://127.0.0.1:8091/console/
```

- **3D 力导向拓扑图**（three.js + 3d-force-graph）：Fact / Episode / Tag / Tool / Source 及关联边；支持旋转、缩放、平移、搜索聚焦
- 顶部**搜索栏**：按事实正文、ID、标签、tools、correlation 定位并高亮节点
- **vis-network 已内置**（`web/vendor/`，不依赖外网 CDN）

**若页面空白、看不到拓扑：**

1. 必须用 **`-console`**（仅 `go run` 默认 stdio **不会**开 8091 控制台）  
2. **重新 `go build`** 后再启动（旧 exe 无内置 vis / 无 console 路由）  
3. 指定数据目录：`-data C:/DATA/GODATA/AgentTestMemoryMCP/data` 或 `MEMORY_MCP_DATA_DIR`  
4. 启动日志应出现：`console data_dir=... facts=N`（N>0 才有图）  
5. 浏览器访问 **http://127.0.0.1:8091/console/**（注意 `/console/`）  
6. 自检：打开 http://127.0.0.1:8091/console/api/stats 应返回 `facts_count`  
7. 浏览器 **Ctrl+F5** 强刷缓存

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
