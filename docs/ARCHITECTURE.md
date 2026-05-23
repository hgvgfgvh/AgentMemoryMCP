# 架构地图（与 DESIGN_INTENT 对齐）

> **宪法**：`DESIGN_INTENT.md`  
> **As-Is**：`CURRENT_IMPLEMENTATION_ARCHITECTURE.md`（Phase-2a factworld + 伴生控制台）  
> **To-Be 落地**：`MEMORY_AGENT_IMPLEMENTATION_PLAN.md`（2b～2e，**已批准实施**）  
> **漂移**：`ARCHITECTURE_DRIFT.md`

---

## 阶段总览

| 阶段 | 状态 | 要点 |
|------|------|------|
| Phase-1 stub | 已保留 | `-engine stub` / `test`；审计队列 |
| **Phase-2a** | **当前默认** | `FactWorldEngine` 规则抽取；关键词 retrieve；`facts.jsonl`；伴生 3D 控制台 |
| Phase-2b | **下一步** | `edges.jsonl`、Weighted BFS（出度惩罚+防环）、BM25 剪枝、pitfall |
| Phase-2c | 计划 | LLM 结构化抽取、L0/L1(fuzzy)、S5 失败回退 rules |
| Phase-2d | 计划 | supersede 退化、embedding≥0.92、模糊带异步 LLM 对齐 |
| Phase-2e | 计划 | 可选 retrieve LLM prune（默认仍 bm25） |

---

## 总览（当前 Phase-2a）

```text
Host（AgentTest 等）
  │ 钩子 OnTurnRetrieve / OnTurnStore / DecideRoute（不经执行 Agent tool_calls）
  ▼
MCP Client（stdio，默认）
  ▼
cmd/memory-mcp/main.go
  ├── memory_store   → engine.FactWorldEngine.Store
  │       filter → episode → job → [async] rules.ExtractFromEpisode → facts.jsonl
  └── memory_retrieve → engine.FactWorldEngine.Retrieve
          filter → facts.List → MatchScore → BuildHints + ---memory-route---
  └── console.Server（伴生 HTTP，只读，默认 :8091）
```

---

## 目标总览（Phase-2b+，见实现方案）

```text
Store (async):
  filter → episode → job → template → preparse → [LLM extract | rules fallback]
    → validate L0/L1(fuzzy)/L2 → merge → degenerate → facts.jsonl + edges.jsonl

Retrieve (sync, budget≤300ms):
  filter → loadGraph → seedAnchor(BM25) → weightedBFS → prune(bm25|llm?) → hints
```

---

## 模块职责（当前 + 规划）

| 路径 | 职责 | 阶段 |
|------|------|------|
| `cmd/memory-mcp` | MCP 入口；stdio/HTTP/console；注册工具 | 2a |
| `internal/engine` | `Engine`；`FactWorldEngine`、`StubEngine`、`TestEngine` | 2a |
| `internal/agent/rules.go` | 规则抽取 episode → Fact | 2a |
| `internal/facts` | `facts.jsonl` CRUD | 2a |
| `internal/filter` | store/retrieve 寒暄、空输入 | 2a |
| `internal/retrieve` | 打分、`BuildHints`、`memory-route` | 2a → 2b 演进 prune |
| `internal/response` | JSON 字符串响应 | 2a |
| `internal/textutil` | 分词/匹配 | 2a |
| `internal/console` | 3D 拓扑、搜索 API（只读） | 2a |
| `internal/graph/*` | 邻接表加载、Weighted BFS | **2b** |
| `internal/index/bm25.go` | BM25 索引 | **2b** |
| `internal/memoryagent/*` | Store 流水线、LLM extract、validate、fuzzy | **2c** |
| `internal/degenerate/*` | supersede、weight 衰减 | **2d** |
| `internal/llm/*` | OpenAI 兼容客户端、超时 | **2c** |

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
| **行为（2a）** | 全量读 facts → `MatchScore` → Top-K hints + `---memory-route---` |
| **行为（2b+）** | 加载图 → 锚定 → BFS → BM25 复合打分 → hints（预算内） |

---

## 持久化布局

### 当前（2a）

```text
data/
  facts/facts.jsonl
  episodes/YYYY-MM-DD/
  store_log/
  jobs/pending|done|dead/
```

### 目标（2b+）

```text
data/
  facts/facts.jsonl          # Summary Fact（Host hints 主索引）
  graph/edges.jsonl          # 持久边（BFS 依据）
  graph/nodes.jsonl          # 可选实体目录
  atoms/atoms.jsonl          # 原子三元组审计
  episodes/ ...
  jobs/ ...
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
| `MEMORY_MCP_RETRIEVE_PRUNE` | `bm25` | `bm25` \| `llm`（**2e**） |
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
