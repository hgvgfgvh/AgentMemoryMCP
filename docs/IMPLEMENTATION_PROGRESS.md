# 实现进度（Phase-2 路线图）

> **当前生产阶段**：**Phase-2d**（`phase=2d-factworld`）  
> **最后更新**：2026-05-24  
> **自动化验证**：`AgentTest` 仓库 `scripts/memory_boundary_test`（13/13）、`llm_store_smoke`、`memory_complex_test`（7/7）；`go test ./...` 通过。

---

## 总览

| 阶段 | 状态 | 完成日期 | 说明 |
|------|------|----------|------|
| Phase-1 stub | ✅ 完成 | 2026-05-20 | 双工具、异步 store；`-engine stub` 仍保留 |
| Phase-2a factworld | ✅ 完成 | 2026-05-23 | 规则抽取、`facts.jsonl`、关键词 retrieve、伴生 3D 控制台 |
| **Phase-2b** | ✅ 完成 | 2026-05-23 | 持久图、Weighted BFS、BM25 剪枝、pitfall、retrieve 预算 |
| **Phase-2c** | ✅ 完成 | 2026-05-23 | LLM 结构化抽取、L0/L1 Fuzzy、S5→rules 回退、`atoms.jsonl` |
| **Phase-2d** | ✅ 完成 | 2026-05-23 | supersede 退化、实体硬合并、模糊带异步对齐、访问衰减 |
| **Phase-2e** | ⏸ **暂缓** | — | 可选 retrieve LLM prune；见下文决策，**默认不实现** |

```text
[████████████████████] 2a → 2b → 2c → 2d   已完成
[                    ] 2e                 暂缓（配置占位即可）
```

---

## 各阶段交付对照

### Phase-2b（图检索基准线）✅

| 交付物 | 代码位置 |
|--------|----------|
| `graph/edges.jsonl` | `internal/graph/persist.go`、`derive.go` |
| Weighted BFS + 出度惩罚 + 防环 | `internal/graph/bfs.go` |
| BM25 复合剪枝 | `internal/index/bm25.go`、`internal/retrieve/pipeline.go` |
| pitfall + 路由抑制 | `internal/agent/rules.go`、`internal/retrieve/hints.go` |
| retrieve 预算 300ms | `internal/engine/env.go` |

### Phase-2c（Store LLM）✅

| 交付物 | 代码位置 |
|--------|----------|
| `templates/agenttest-plan.yaml` | `internal/memoryagent/templates/` |
| LLM 抽取（1 次） | `internal/memoryagent/llm_extract.go`、`internal/llm/` |
| L0 / L1 Fuzzy | `internal/memoryagent/validate.go`、`fuzzy.go` |
| S5 失败回退 rules | `internal/memoryagent/pipeline.go` |
| `atoms/atoms.jsonl` | `internal/atoms/repo.go` |

**运行要求**：`mcp_env` 配置 `MEMORY_MCP_LLM_API_BASE`（根路径，如 `https://api.deepseek.com`）、`MEMORY_MCP_LLM_API_KEY`、`MEMORY_MCP_LLM_MODEL`；`MEMORY_MCP_LLM_EXTRACT=1`。

### Store L2 语义冲突 ✅（2026-05-24）

| 交付物 | 代码位置 |
|--------|----------|
| 规则候选（tools/tags/artifact + outcome 对立） | `internal/memoryagent/conflict_detect.go` |
| mini LLM A/B/C + 默认 C | `internal/memoryagent/l2_conflict.go` |
| 接入 Store 异步链 | `pipeline.go` → `finalizeStoreOutput`；`factworld.processJob` |

### Phase-2d（退化与对齐）✅

| 交付物 | 代码位置 |
|--------|----------|
| supersede + weight×0.2 | `internal/degenerate/governor.go` |
| 久未访问衰减 | `internal/degenerate/governor.go` |
| Retrieve 命中 touch | `factworld.Retrieve` → `TouchRetrieve` |
| cosine≥0.92 硬合并 | `internal/entity/align.go`、`internal/embedding/bag.go` |
| 0.85–0.92 异步 LLM 对齐 | `internal/align/worker.go`（不阻塞 Store） |
| Fact 字段扩展 | `last_active`、`access_count`、`superseded` |

---

## Phase-2e：暂缓决策（2026-05-24）

### 结论

**暂不实现** retrieve 热路径上的可选 LLM 剪枝（R4'）。当前 **BM25 × 激活能级 × weight** 为唯一默认剪枝路径，与专家评审 Q4 及 AgentTest 全链路测试结论一致。

### 理由摘要

1. **能力闭环已闭合**：2b 拓扑召回 + 2c 结构化写入 + 2d 退化/对齐，已覆盖「联想、防幻觉、去重、取快存慢」。  
2. **专家已裁定**：AgentTest 类技术内网场景下，默认 BM25 **工程上足够**；LLM prune 仅适用于 10+ 强冲突候选或极含糊/反讽意图等少数场景。  
3. **成本不对称**：每轮可能两次 retrieve，加热路径 LLM 增加延迟抖动与运维面，边际收益小。  
4. **实测已绿**：边界测试、Web 冒烟、复杂三步流、LLM store + supersede 均通过，无「必须靠 2e 才能用」的信号。

### 若未来要做 2e 的触发条件（需同时观察一段时间）

- retrieve 常返回多条近似分 fact，hints 噪声明显干扰 Plan；  
- 多项目共库导致 BM25 难以再分，且业务要求极短 hints；  
- 有明确指标证明 BM25 路由/联想不足（而非 Store 质量或图问题）。

### 实现时的约束（预留给未来）

- 环境变量 `MEMORY_MCP_RETRIEVE_PRUNE=llm`，**默认 `bm25`**；  
- 硬超时纳入 `MEMORY_MCP_RETRIEVE_BUDGET_MS`，失败 **回退 bm25**；  
- 禁止多轮 Agent / 访问全图。

详见 `MEMORY_AGENT_IMPLEMENTATION_PLAN.md` §13。

---

## 后续可选增强（非 2e）

| 项 | 优先级 | 说明 |
|----|--------|------|
| 控制台读持久 `edges.jsonl` | P2 | 当前仍由 facts 推导展示，与 retrieve 用的边一致但未直读文件 |
| L2 冲突 mini LLM（Store） | ✅ | `memoryagent` L2：规则候选 + A/B/C；`MEMORY_MCP_L2_CONFLICT` |
| `source` 多租户分库 | P2+ | 同文件仅 JSON 字段 |
| Host 双次 retrieve 合并缓存 | 待议 | 设计讨论项，非 MCP 单独交付 |

---

## 验证命令

```powershell
# MemoryMCP 单元测试
cd C:\DATA\GODATA\AgentTestMemoryMCP
go test ./...
go build -o memory-mcp.exe ./cmd/memory-mcp

# Host 集成（主项目已启动时）
cd C:\DATA\GODATA\AgentTest
go run ./scripts/memory_boundary_test
go run ./scripts/llm_store_smoke
go run ./scripts/memory_complex_test
```

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-24 | 初版：2a～2d 完成登记；2e 暂缓决策与后续增强列表 |
