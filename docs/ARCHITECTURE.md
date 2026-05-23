# 架构地图（与 DESIGN_INTENT 对齐）

> **宪法**：`DESIGN_INTENT.md`  
> **As-Is**：`CURRENT_IMPLEMENTATION_ARCHITECTURE.md`（**Phase-2d**）  
> **进度**：`IMPLEMENTATION_PROGRESS.md`（2b～2d ✅；2e ⏸）  
> **方案全文**：`MEMORY_AGENT_IMPLEMENTATION_PLAN.md`  
> **漂移**：`ARCHITECTURE_DRIFT.md`

---

## 阶段总览

| 阶段 | 状态 | 要点 |
|------|------|------|
| Phase-1 stub | 已保留 | `-engine stub` / `test` |
| Phase-2a | ✅ | 规则抽取、关键词 retrieve、伴生控制台 |
| Phase-2b | ✅ | `edges.jsonl`、BFS、BM25、pitfall、retrieve 预算 |
| Phase-2c | ✅ | LLM extract、L0/L1 fuzzy、atoms、`rules` 回退 |
| **Phase-2d** | ✅ **当前** | supersede、硬合并、异步对齐、访问衰减 |
| Phase-2e | ⏸ **暂缓** | retrieve LLM prune（默认不实现，保持 `bm25`） |

---

## 总览（当前 Phase-2d）

```text
Host（AgentTest 等）
  │ 钩子 OnTurnRetrieve / OnTurnStore / DecideRoute
  ▼
MCP Client（stdio）
  ▼
cmd/memory-mcp/main.go
  ├── memory_store → FactWorldEngine.Store
  │     filter → episode → job → [async]
  │       memoryagent: template → preparse → [LLM | rules] → L0/L1 → merge
  │       degenerate + entity/align → facts.jsonl + edges.jsonl + atoms.jsonl
  └── memory_retrieve → FactWorldEngine.Retrieve
        filter → loadGraph → BFS → BM25 prune (budget) → hints + memory-route
        (跳过 superseded；命中 touch last_active)
  └── console.Server（伴生 :8091，只读）
```

---

## 模块职责（Phase-2d）

| 路径 | 职责 |
|------|------|
| `cmd/memory-mcp` | MCP 入口；stdio/HTTP/console |
| `internal/engine/factworld.go` | Store/Retrieve 编排 |
| `internal/agent/rules.go` | 规则抽取（LLM 回退） |
| `internal/memoryagent/*` | Store 流水线、template、validate、fuzzy |
| `internal/llm/*` | OpenAI 兼容客户端 |
| `internal/atoms` | `atoms.jsonl` |
| `internal/facts` | `facts.jsonl`（含 superseded / last_active） |
| `internal/graph/*` | `edges.jsonl`、BFS |
| `internal/index/bm25.go` | BM25 |
| `internal/retrieve/*` | pipeline、hints、memory-route |
| `internal/degenerate/*` | supersede、衰减、touch |
| `internal/entity` + `internal/embedding` | 硬合并 |
| `internal/align` | 模糊带异步 LLM |
| `internal/filter` | 寒暄过滤 |
| `internal/console` | 3D 开发台（只读） |

---

## MCP 工具契约

### memory_store

| 项 | 说明 |
|----|------|
| **入参** | `content`（required string），`source`，`kind`，`correlation_id` |
| **出参** | JSON 字符串：`accepted`、`job_id`、`skipped`、`skip_reason`、`message`、`phase` |
| **行为** | 过滤 → 同步 ACK → 异步 job → 规则抽取写 `facts.jsonl`（2c 起可多 atoms + edges） |

### memory_retrieve

| 项 | 说明 |
|----|------|
| **入参** | `context`（required string），`query_hint` |
| **出参** | JSON 字符串：`hints`、`skipped`、`skip_reason`、`phase` |
| **行为（2d）** | 加载图 → BFS → BM25 剪枝 → hints（预算内）；`phase=2d-factworld` |

---

## 持久化布局

### 当前（2d）

```text
data/
  facts/facts.jsonl          # Summary Fact（含 superseded / last_active）
  graph/edges.jsonl          # 持久边（BFS）
  atoms/atoms.jsonl          # 原子三元组审计
  episodes/YYYY-MM-DD/
  jobs/pending|done|dead/
  store_log/
```

---

## 传输与运行模式

| 模式 | 启动 | MCP | 控制台 |
|------|------|-----|--------|
| **生产默认** | Host `mcp_command` | stdio | 伴生 `127.0.0.1:8091`（可禁用） |
| 仅控制台 | `-console addr` | 无 | HTTP |
| MCP+控制台 | `-http addr` | Streamable HTTP | `/console/` |
| 占位 | `-engine stub` / `test` | stdio | 可选 |

---

## 环境变量（MCP 进程）

| 变量 | 默认 | 说明 |
|------|------|------|
| `MEMORY_MCP_DATA_DIR` | `./data` | 数据根 |
| `MEMORY_MCP_ENGINE` | `factworld` | `factworld` \| `stub` \| `test` |
| `MEMORY_MCP_CONSOLE_LISTEN` | `127.0.0.1:8091` | stdio 伴生控制台 |
| `MEMORY_MCP_CONSOLE_DISABLE` | - | `1` 关闭控制台 |
| `MEMORY_MCP_RETRIEVE_BUDGET_MS` | `300` | retrieve 总预算（**2b 起强制**） |
| `MEMORY_MCP_RETRIEVE_PRUNE` | `bm25` | 仅 `bm25` 已实现；`llm` 属 **2e 暂缓** |
| `MEMORY_MCP_LLM_EXTRACT` | `1` | Store 是否 LLM（**2c**） |
| `MEMORY_MCP_LLM_API_BASE` / `MODEL` | - | OpenAI 兼容（**2c**） |

---

## Host 集成（AgentTest 参考）

| 钩子 | 行为 |
|------|------|
| OnTurnRetrieve | `memory_retrieve` → 注入 `【跨会话事实参考】` |
| OnTurnStore | 异步 `memory_store`，`correlation_id=turn_id` |
| DecideRoute | 第二次 retrieve → 解析 `exec_simple_match` / confidence |

执行 Agent **不**挂载本 MCP；路由阈值（0.75、tier≤2）留在 Host `config/app.yaml`。

---

## 明确不做

- Neo4j / 外置图库（除非单机性能不足再评估 SQLite 图）
- Host 可见图编辑 MCP 工具
- Retrieve 多轮 Agent / ReAct
- Store 主链同步 LLM 实体对齐
- Retrieve 热路径默认强模型

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-20 | Phase-1 stub 架构地图 |
| 2026-05-23 | 对齐 2a 实现、2b～2e 路线图、模块与环境变量、交叉引用 |
| 2026-05-24 | 当前阶段 2d；总览与模块表结案；2e 暂缓 |
