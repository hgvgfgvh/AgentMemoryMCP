# AgentTestMemoryMCP — 文档索引

本目录为记忆 MCP 的**单一文档源**（SSOT）。实现与 Agent 改码前请先读宪法，再对照 As-Is 与漂移表。

---

## 推荐阅读顺序

| 顺序 | 文档 | 谁读 | 内容 |
|------|------|------|------|
| 1 | [DESIGN_INTENT.md](./DESIGN_INTENT.md) | 所有人 | **宪法**：协议、钩子、双链路、BM25/Fuzzy/对齐边界 |
| 2 | [CURRENT_IMPLEMENTATION_ARCHITECTURE.md](./CURRENT_IMPLEMENTATION_ARCHITECTURE.md) | 评审 / 排障 | **As-Is**：factworld 2a、控制台、AgentTest 集成 |
| 3 | [ARCHITECTURE_DRIFT.md](./ARCHITECTURE_DRIFT.md) | 实现 Agent | 宪法 vs 代码；2b～2e 待办 |
| 4 | [MEMORY_AGENT_IMPLEMENTATION_PLAN.md](./MEMORY_AGENT_IMPLEMENTATION_PLAN.md) | 实现 Agent | **批准方案**：2b 优先、专家 Q4–Q6、检查清单 |
| 5 | [ARCHITECTURE.md](./ARCHITECTURE.md) | 实现 Agent | 模块职责、环境变量、阶段表 |
| 6 | [ACCEPTANCE_RULES.md](./ACCEPTANCE_RULES.md) | QA / CI | 可勾选验收项（2a 已满足 + 2b～2e） |

---

## 文档关系

```text
DESIGN_INTENT (宪法)
       │
       ├──► ARCHITECTURE_DRIFT (差距)
       │
       ├──► MEMORY_AGENT_IMPLEMENTATION_PLAN (To-Be，已批准)
       │
       ├──► ARCHITECTURE (模块地图)
       │
       ├──► CURRENT_IMPLEMENTATION (As-Is)
       │
       └──► ACCEPTANCE_RULES (验收)
```

---

## 阶段速查

| 阶段 | 状态 | 关键交付 |
|------|------|----------|
| 2a | **当前** | 规则抽取、`facts.jsonl`、关键词 retrieve、伴生 3D 控制台 |
| **2b** | **下一步** | `edges.jsonl`、BFS（出度+防环）、BM25 剪枝、pitfall、retrieve 预算 |
| 2c | 计划 | LLM 抽取、L0/L1 fuzzy、S5 回退 |
| 2d | 计划 | supersede、embedding≥0.92、异步模糊带对齐 |
| 2e | 计划 | 可选 retrieve LLM prune（默认仍 bm25） |

---

## 专家评审结论（摘要）

- **Retrieve**：默认 **BM25 × 激活能级 × weight**；LLM prune 仅少数场景且默认关闭。  
- **L1**：evidence 须 **Fuzzy**（~85%），禁止硬子串。  
- **Store 对齐**：硬规则 + **cosine≥0.92**；主链**禁止**同步 LLM 对齐。  
- **BFS**：须 **出度惩罚** 与 **visited 防环**。  
- **实施**：立即启动 **2b** 无 LLM 基准线。

详见 `MEMORY_AGENT_IMPLEMENTATION_PLAN.md` §10–§12。

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-23 | 初版索引；对齐 docs 全目录 v2 同步 |
