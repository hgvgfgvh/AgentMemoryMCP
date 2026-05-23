# 验收规则

可执行检查优先于 prompt 约定。宪法锚点见 `DESIGN_INTENT.md`；阶段交付见 `MEMORY_AGENT_IMPLEMENTATION_PLAN.md` §8、§12。

---

## 全局（所有阶段）

### AR-G1 工具面

- [ ] MCP `tools/list` **仅**包含 `memory_store`、`memory_retrieve`
- [ ] 无 Host 可见的建图/删节点/改图类工具
- [ ] 必填参数为 string 语义：`content` / `context`

### AR-G2 返回格式

- [ ] 工具结果为**单一** TextContent，正文为合法 JSON 字符串
- [ ] `memory_store` 含 `accepted`、`job_id`、`skipped`（string 值）
- [ ] `memory_retrieve` 含 `hints`、`skipped`（string 值）

### AR-G3 Host 集成

- [ ] 执行 Agent `capabilities.attach_to` **不出现** memory 工具名
- [ ] 仅 Host 钩子调用 MCP（OnTurnRetrieve / OnTurnStore / DecideRoute）

### AR-G4 验证命令

```powershell
cd C:\DATA\GODATA\AgentTestMemoryMCP
go test ./...
go build -o memory-mcp.exe ./cmd/memory-mcp
```

---

## Phase-1 / Phase-2a（当前须满足）

### AR-1 store 语义

- [ ] 非过滤内容：`accepted` 为 `"true"` 且 `job_id` 非空
- [ ] 寒暄：`skipped` 为 `"true"`，且不追加 episode
- [ ] 调用**立即返回**（不等待抽取完成）
- [ ] 异步 job 最终 `done` 或 `dead` 可观测

### AR-2 retrieve 语义

- [ ] **同步**返回（相对 Host 可接受延迟）
- [ ] 寒暄：`hints` 为空且 `skipped` 为 `"true"`
- [ ] 空库/失败：MCP 不崩溃；`hints` 可为空或占位
- [ ] hints 可含 `---memory-route---` JSON（AgentTest Exec-Simple）

### AR-3 factworld 数据

- [ ] `facts/facts.jsonl` 可追加、可按 `correlation_id` 替换
- [ ] `episodes/` 保留原始 episode
- [ ] `-engine factworld` 为默认生产引擎

### AR-4 控制台（可选启用）

- [ ] stdio 启动后默认 `http://127.0.0.1:8091/console/` 可访问（未设 `CONSOLE_DISABLE`）
- [ ] 控制台**只读**；不影响 store/retrieve 结果

---

## Phase-2b（图 + BFS + BM25 基准线）

### AR-2b-1 持久图

- [ ] `data/graph/edges.jsonl` 存在且与 store 写入一致
- [ ] retrieve 从 jsonl **加载内存邻接表**（非仅控制台推导）

### AR-2b-2 拓扑激活

- [ ] Weighted BFS：`maxDepth≤2`，`minEnergy` 可配置
- [ ] **出度惩罚**：hub 节点扩散后激活集合规模受控（单测：通用工具节点不激活 >N 事实）
- [ ] **防环**：`similar` 双向边不导致无限队列（单测）

### AR-2b-3 剪枝

- [ ] 默认 `MEMORY_MCP_RETRIEVE_PRUNE=bm25`
- [ ] `finalScore` 含 BM25、activatedEnergy、weight；pitfall 有惩罚项
- [ ] Top-K hints ≤ 配置上限（如 3）

### AR-2b-4 性能与 SLA

- [ ] retrieve P95 **<50ms**（无 LLM，本地 jsonl 规模基准数据集）
- [ ] `RETRIEVE_BUDGET_MS` 超时 → 降级 hints，**不**向 Host 抛致命错误

### AR-2b-5 pitfall

- [ ] `outcome=failed`（或等价）可写 pitfall 边/标记
- [ ] pitfall 命中时 `exec_simple_match=no`（经 memory-route）

### AR-2b-6 回归

- [ ] stdio 双工具契约不变；AgentTest Host **无需**改代码

---

## Phase-2c（Store LLM + 校验）

### AR-2c-1 抽取

- [ ] `source=agenttest-plan` 可走 template + preparse + **1 次** LLM extract
- [ ] LLM 超时/JSON 非法 → **回退** `ExtractFromEpisode` 单条 Summary
- [ ] 单 episode 可产生 **多条** atoms（审计在 `atoms.jsonl`）

### AR-2c-2 防幻觉

- [ ] **L0**：tools/artifacts ⊆ preparse
- [ ] **L1**：evidence 与 episode **Fuzzy** 重合 ≥85%（非裸 `Contains`）
- [ ] **L2**：仅冲突时触发；默认共存 + 降 confidence

### AR-2c-3 作业

- [ ] job dead 可重跑；metrics/日志可观测失败原因

---

## Phase-2d（退化 + 对齐）

### AR-2d-1 退化

- [ ] `supersedes` 边写入；被取代 fact `weight` 下降
- [ ] 久未 retrieve 访问衰减可配置

### AR-2d-2 对齐

- [ ] 硬规则 + **cosine≥0.92** 自动合并（单测：filesystem / file-system）
- [ ] **0.85–0.92** 仅异步队列 LLM；**Store 主链延迟不增加**（对比 2c 基线）

### AR-2d-3 correlation

- [ ] 同 `correlation_id` 不无限堆叠重复 Summary

---

## Phase-2e（可选 retrieve LLM）

### AR-2e-1

- [ ] `RETRIEVE_PRUNE=llm` 时单次 mini 剪枝，硬超时纳入总预算
- [ ] 超时/失败 **回退** bm25 结果
- [ ] 默认配置仍为 `bm25`

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-20 | Phase-1 验收项 |
| 2026-05-23 | 分层 AR-G / 2a / 2b～2e；对齐专家结论与实现方案 |
