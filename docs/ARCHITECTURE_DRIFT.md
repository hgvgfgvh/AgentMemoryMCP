# 架构漂移（设计意图 vs 当前实现）

**分类**：`aligned` | `reasonable evolution` | `technical debt` | `deferred` | `violates design`

**裁定原则**：以 `DESIGN_INTENT.md` 为准。Phase-2b～2d 已落地项记为 **`aligned`**。

> **进度总表**：见 [IMPLEMENTATION_PROGRESS.md](./IMPLEMENTATION_PROGRESS.md)。

---

## 当前阶段：Phase-2d（生产默认）

| 项 | 说明 |
|----|------|
| **引擎** | `FactWorldEngine`，`-engine factworld` |
| **phase 标识** | `2d-factworld` |
| **Store** | LLM 抽取（可关）+ L0/L1 + rules 回退 + supersede/对齐 |
| **Retrieve** | 图加载 → BFS → BM25 剪枝；跳过 `superseded`；命中写回 `last_active` |

---

## Phase-2b～2d 已闭合（aligned）

| 能力 | 实现 |
|------|------|
| 持久图 + BFS + 出度惩罚 + 防环 | `internal/graph/*` |
| BM25 × 能级 × weight 剪枝 | `internal/retrieve/pipeline.go` |
| pitfall + Exec-Simple 抑制 | `hints.go`、`rules.go` |
| retrieve 预算 | `MEMORY_MCP_RETRIEVE_BUDGET_MS` |
| LLM 结构化抽取 + atoms | `internal/memoryagent/*`、`internal/atoms` |
| L0 / L1 Fuzzy | `validate.go`、`fuzzy.go` |
| supersede 退化 | `internal/degenerate` |
| 实体硬合并 ≥0.92 | `internal/entity`、`internal/embedding` |
| 模糊带异步 LLM 对齐 | `internal/align`（`MEMORY_MCP_ENTITY_ALIGN_ASYNC`） |
| 字符串双工具、Host 钩子 | 已满足 |
| 伴生控制台 `:8091` | `internal/console` |

---

## 暂缓 / 未做（deferred 或 backlog）

| 项 | 分类 | 说明 |
|----|------|------|
| **Phase-2e retrieve LLM prune** | **`deferred`** | **2026-05-24 决定暂不实现**；默认 BM25 足够，见 `IMPLEMENTATION_PROGRESS.md` |
| L2 Store 冲突 mini LLM | **`aligned`** | `conflict_detect.go` + `l2_conflict.go`；默认启用 |
| 控制台优先读 `edges.jsonl` | `technical debt` | 展示仍 facts 推导，retrieve 已用持久边 |
| 多租户 `source` 分库 | `technical debt` | 仅 JSON 字段 |
| Neo4j / 外置图库 | `aligned`（不做） | 宪法禁止默认引入 |
| Host 双次 retrieve 合并 | `needs human decision` | Host 侧优化 |

---

## 专家评审结案（已实现对应项）

| 议题 | 决议 | 状态 |
|------|------|------|
| Q4 默认 BM25 | 足够，LLM prune 非默认 | ✅ 2b；2e 暂缓 |
| Q5 L1 Fuzzy | 必须 | ✅ 2c |
| Q6 硬合并 0.92；Store 禁止同步 LLM 对齐 | 是 | ✅ 2d |
| BFS 出度 + 防环 | 必须 | ✅ 2b |

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-20 | Phase-1 stub 初始登记 |
| 2026-05-23 | 2a～2e 路线图登记 |
| 2026-05-24 | 2b～2d 标为 aligned；2e 标 deferred；精简历史债务表 |
