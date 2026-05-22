# 架构漂移（设计意图 vs 当前实现）

分类：`aligned` | `reasonable evolution` | `technical debt` | `needs human decision` | `violates design`

---

## Phase-1 已知差距（均为 planned stub）

| 设计意图 | 当前实现 | 分类 | 说明 |
|----------|----------|------|------|
| Memory Agent 抽取事实、建图 | `FactWorldEngine` 规则抽取 + `facts.jsonl` | `reasonable evolution` | Phase-2a；无 LLM/向量 |
| retrieve 由 Agent 裁切关联记忆 | 关键词打分 + `---memory-route---` | `reasonable evolution` | 同步路径无 LLM |
| StubEngine 审计队列 | 仍保留 `-engine stub` | `aligned` | — |
| TestEngine 内嵌 fixture | 仍保留 `-engine test` | `aligned` | 与 factworld 并存 |
| store 异步队列 + 死信 | goroutine 追加文件 | `reasonable evolution` | Phase-1 足够验证 Host 钩子时序 |
| 图结构 / 退化 / 向量索引 | 无 | `technical debt` | Phase-2 |
| 多租户 `source` 分库 | 仅写入 JSONL 字段 | `technical debt` | 未分文件隔离 |
| 字符串协议、仅存取两工具 | 已实现 | `aligned` | — |
| 寒暄过滤 | `internal/filter` | `aligned` | 规则表可扩展 |
| 返回 JSON 字符串 | `internal/response` | `aligned` | — |

---

## 修订

| 日期 | 说明 |
|------|------|
| 2026-05-20 | Phase-1 stub 初始登记 |
