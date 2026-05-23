# 架构漂移（设计意图 vs 当前实现）

**分类**：`aligned` | `reasonable evolution` | `technical debt` | `needs human decision` | `violates design`

**裁定原则**：以 `DESIGN_INTENT.md` 为准；已写入宪法的 Phase-2 目标，在 `MEMORY_AGENT_IMPLEMENTATION_PLAN.md` 中落地的，记为 **`approved evolution`**（待实现），非违宪。

---

## 当前阶段：Phase-2c（LLM 抽取 + L0/L1，代码已落地）

> 2026-05-23：已实现 `internal/memoryagent`（template、LLM extract、L0/L1 fuzzy、rules 回退）、`internal/atoms`、`internal/llm`；phase=`2c-factworld`。未配置 `MEMORY_MCP_LLM_API_BASE` 时 Store 自动 rules 回退。

## Phase-2b → 2c 已闭合项

| 设计意图 | 当前实现 | 分类 |
|----------|----------|------|
| LLM 结构化抽取（1 次） | `memoryagent.LLMExtract` | `aligned` |
| S5 失败 → rules Summary | `ProcessEpisode` fallback | `aligned` |
| L0 防幻觉 | `ValidateL0` | `aligned` |
| L1 Fuzzy evidence | `EvidenceAnchored` + `ValidateL1` | `aligned` |
| atoms.jsonl 审计 | `internal/atoms` | `aligned` |
| template agenttest-plan | embed YAML | `aligned` |
| L2 冲突 mini LLM | 未实现 | `technical debt` → 2c+ 可选 |

## Phase-2b（retrieve，已落地）

> `internal/graph`、`retrieve/pipeline`；`MEMORY_MCP_RETRIEVE_LEGACY=1` 可回退关键词检索。

## Phase-2a → 2b 已闭合项

| 设计意图 | 当前实现 | 分类 |
|----------|----------|------|
| 持久图 + BFS retrieve | `graph/edges.jsonl` + WeightedBFS | `aligned` |
| BM25 复合剪枝 | `retrieve.SearchWithGraph` | `aligned` |
| BFS 出度惩罚 + 防环 | `graph/bfs.go` + 单测 | `aligned` |
| Pitfall 抑制路由 | `IsPitfall` + `BuildHints` | `aligned` |
| retrieve 预算 | `MEMORY_MCP_RETRIEVE_BUDGET_MS` | `aligned` |

## 仍待 Phase-2c～2e

| 设计意图 | 状态 |
|----------|------|
| LLM 结构化抽取 | 待 2c |
| L1 Fuzzy evidence | 待 2c |
| embedding 对齐 / supersede 退化 | 待 2d |
| 可选 retrieve LLM prune | 待 2e |
| 控制台读持久 `edges.jsonl`（优先于推导） | 待增强 |

---

## 历史：Phase-2a（factworld 规则引擎）

| 设计意图（宪法） | 当前实现 | 分类 | 目标阶段 / 说明 |
|------------------|----------|------|-----------------|
| 仅存取两工具、string 协议 | 已实现 | `aligned` | — |
| Store 异步 ACK + job 队列 | goroutine + `jobs/*` | `aligned` | — |
| Retrieve 同步、失败不阻断 Host | 已实现（无硬超时） | `reasonable evolution` | **2b** 加 `RETRIEVE_BUDGET_MS` |
| 寒暄过滤 | `internal/filter` | `aligned` | — |
| Host 不挂载 memory 工具 | AgentTest 已满足 | `aligned` | — |
| 记忆 Agent 建图、连边 | 无持久图；控制台**推导**边 | `technical debt` | **2b** `edges.jsonl` + BFS |
| 拓扑激活 retrieve | 仅 `MatchScore` 关键词 | `technical debt` | **2b** BFS + BM25 |
| Retrieve 默认 BM25×能级×weight | 未实现 BM25/能级 | `technical debt` | **2b** |
| BFS 出度惩罚 + 防环 | 无 BFS | `technical debt` | **2b** |
| Retrieve 预算 ≤300ms | 无硬超时 | `technical debt` | **2b** |
| Pitfall 抑制 Exec-Simple | 未区分 pitfall 类型 | `technical debt` | **2b** |
| LLM 结构化抽取（Store 1 次） | `rules.go` 仅 | `technical debt` | **2c** |
| S5 失败 → 规则 Summary 回退 | 仅 rules 路径 | `aligned`（行为） | **2c** 显式双路径 |
| L1 evidence Fuzzy 锚定 | 无 atoms / 无 L1 | `technical debt` | **2c** |
| L0/L2 防幻觉 | 部分在 rules 内隐式 | `technical debt` | **2c** |
| 实体硬规则 + cosine≥0.92 | 无 embedding 对齐 | `technical debt` | **2d** |
| 模糊带异步 LLM 对齐 | 无 | `technical debt` | **2d** |
| supersede / weight 退化 | 仅 `ReplaceByCorrelation` | `technical debt` | **2d** |
| 可选 retrieve LLM prune | 无 | `technical debt` | **2e**（默认关闭） |
| 每 episode 多 fact / atoms | 通常 1 条 Summary | `technical debt` | **2c** |
| 模板 per `source` | 无 | `technical debt` | **2c** |
| 向量索引 | 无 | `needs human decision` | 2b 用 BM25；embedding 仅 2d 对齐 |
| 多租户 `source` 分库 | 同 `facts.jsonl` 字段 | `technical debt` | P2+ |
| 伴生开发控制台 | stdio 伴生 `:8091` 3D | `aligned` | 2b 控制台读 `edges.jsonl` |
| Stub/Test 引擎 | 保留 `-engine stub/test` | `aligned` | — |
| Neo4j 等外置图库 | 未引入 | `aligned` | 宪法禁止默认引入 |

---

## 专家评审已结案（不再记为 open drift）

| 议题 | 宪法 / 方案决议 | 实现状态 |
|------|-----------------|----------|
| Q4 Retrieve 默认 BM25 是否足够 | **是**；LLM prune 仅可选 | 待 2b |
| Q5 L1 须 Fuzzy（~85%） | **是**；禁止硬子串 | 待 2c |
| Q6 Store 禁止默认 LLM 对齐；0.92 硬合并 | **是** | 待 2d |
| BFS 出度惩罚 + visited 防环 | **必须** | 待 2b |
| 实施批准 | 优先 **2b** Baseline | 未开始 |

---

## 合理演进（已接受、非债务）

| 项 | 说明 |
|----|------|
| Phase-2a 规则引擎先行 | 无 LLM 即可对接 AgentTest Host 钩子，验证时序与 hints 格式 |
| `---memory-route---` 块 | Host `DecideRoute` 解析；属 Host+MCP 协同，非违宪 |
| 控制台边为推导可视化 | 2a 可接受；2b 起须与 `edges.jsonl` 一致 |
| 双次 retrieve（OnTurn + DecideRoute） | Host 侧合理；MCP 无状态，不合并 |

---

## 需人工裁定（暂无）

| 项 | 问题 |
|----|------|
| 双轨 Summary + Atomic 是否过度 | 2c 实现后按指标裁定；当前方案保留双轨 |
| Ontology 是否增加 `PLAN_STEP` | 2b 先用四类节点；不足再扩 |
| pitfall 边是否禁止向外扩散 | 默认允许低能扩散；可 A/B 调参 |

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-20 | Phase-1 stub 初始登记 |
| 2026-05-23 | 对齐 2a、按 2b～2e 重列漂移；专家 Q4–Q6 结案；伴生控制台 |
