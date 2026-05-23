# AgentTestMemoryMCP — 文档索引

本目录为记忆 MCP 的**单一文档源**（SSOT）。改码前请先读宪法，再对照 As-Is、进度表与漂移登记。

---

## 推荐阅读顺序

| 顺序 | 文档 | 谁读 | 内容 |
|------|------|------|------|
| 0 | **[IMPLEMENTATION_PROGRESS.md](./IMPLEMENTATION_PROGRESS.md)** | 所有人 | **进度与 2e 暂缓决策** |
| 1 | [DESIGN_INTENT.md](./DESIGN_INTENT.md) | 所有人 | **宪法**：协议、钩子、双链路 |
| 2 | [CURRENT_IMPLEMENTATION_ARCHITECTURE.md](./CURRENT_IMPLEMENTATION_ARCHITECTURE.md) | 评审 / 排障 | **As-Is**（2d） |
| 3 | [ARCHITECTURE_DRIFT.md](./ARCHITECTURE_DRIFT.md) | 维护者 | 已闭合 vs 待办 |
| 4 | [MEMORY_AGENT_IMPLEMENTATION_PLAN.md](./MEMORY_AGENT_IMPLEMENTATION_PLAN.md) | 实现参考 | 批准方案全文 + §13 2e 说明 |
| 5 | [ARCHITECTURE.md](./ARCHITECTURE.md) | 实现参考 | 模块地图、环境变量 |
| 6 | [ACCEPTANCE_RULES.md](./ACCEPTANCE_RULES.md) | QA / CI | 分阶段验收（2b～2d 已勾选） |

---

## 文档关系

```text
DESIGN_INTENT (宪法)
       │
       ├──► IMPLEMENTATION_PROGRESS (进度 / 2e 决策)
       │
       ├──► CURRENT_IMPLEMENTATION (As-Is 2d)
       │
       ├──► ARCHITECTURE_DRIFT (差距 / deferred)
       │
       ├──► MEMORY_AGENT_IMPLEMENTATION_PLAN (方案 + 检查清单)
       │
       └──► ACCEPTANCE_RULES (验收)
```

---

## 阶段速查（2026-05-24）

| 阶段 | 状态 | 关键交付 |
|------|------|----------|
| 2a factworld + 控制台 | ✅ | 规则抽取、`facts.jsonl`、伴生 3D |
| 2b | ✅ | `edges.jsonl`、BFS、BM25、pitfall |
| 2c | ✅ | LLM 抽取、L0/L1 Fuzzy、`atoms.jsonl` |
| **2d** | ✅ **当前** | supersede、硬合并、异步对齐、访问衰减 |
| **2e** | ⏸ **暂缓** | retrieve LLM prune（默认不实现） |

---

## 专家评审结论（仍有效）

- Retrieve **默认 BM25**；2e LLM prune **暂不实现**（见进度文档）。  
- Store：硬规则 + **cosine≥0.92**；模糊带 **异步** LLM。  
- L1：**Fuzzy** ~85%。BFS：**出度惩罚 + 防环**。

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-23 | 初版索引 |
| 2026-05-24 | 增加 IMPLEMENTATION_PROGRESS；2a～2d 完成；2e 暂缓 |
