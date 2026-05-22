# 设计意图（人工维护 · 宪法层）

本仓库 **AgentTestMemoryMCP** 为独立的**长效事实记忆 MCP** 服务工程。本文档为**记忆系统 MCP** 及对接 Host 的顶层宪法；实现可以演进，但不得违背此处记录的取舍。

> Agent 同步 `ARCHITECTURE.md` 时须以本文件为准绳。当前实现若与本文冲突，记入 `ARCHITECTURE_DRIFT.md` 待人工裁定，不得用「现有代码」自动覆盖本文。

---

## 项目定位

| 项 | 约定 |
|----|------|
| **本 MCP 是什么** | 跨场景可挂载的**长效事实记忆**服务：维护独立「事实世界」，对外仅暴露**存入 / 取出**，内部由**记忆系统 Agent** 负责理解、关联、退化。 |
| **不是什么** | 不是会话滑动窗口、不是「上下文满了再落盘」的第一层短期记忆；不承担 Host 内 Plan/Behavior 的执行编排。 |
| **协议原则** | 对外原始载荷以 **字符串** 为主，**不规定** Host 侧结构体或 JSON Schema；各 Host 自行序列化后传入。 |

---

## 2026-05-20 — 记忆分层：第一层（Host）与第三层（本 MCP）

### 设计意图

- **第一层（短期 / 会话级）**：留在各 Host 工程内原有实现（如 ConversationBuffer、溢出 JSONL、按日 TTL 清理等）。职责是**本轮及近几轮**对话稳定、控制 token，可设置例如 **1 天** 到期整库清理短期文件。
- **第三层（本 MCP）**：实现 Host **长期迭代**所需的常驻事实记忆：跨会话、可关联、可退化；逻辑上视为独立**事实世界**。

两层**并行、不互相替代**；第三层不得为实现方便而改写 Host 第一层逻辑。

### 原因

- 短期记忆解决「当前会话可读、不爆上下文」；事实世界解决「上周的结论、失败模式、用户偏好」等跨轮需求。
- 混在一套滑动窗口里会导致长期事实被截断或重复写入，且难以做图关联与退化。

### 影响

- 本 MCP **不实现** Host 的 session buffer / 滑动窗口。
- Host 的 `retrieve` 可**读取**第一层会话摘要并作为 `context` 字符串的一部分传入，但第一层数据**不归本 MCP 持久化为主库**（除非 Host 明确把某段文本当作 `content` 存入，由记忆 Agent 自行判断是否值得落事实库）。

---

## 2026-05-20 — 记忆系统 Agent 与事实世界

### 设计意图

- 记忆在逻辑上可单独看作一个**事实世界（Fact World）**。
- **记忆系统 Agent**（内部实现，对外无感）专职：
  - 从外部存入的字符串中**抽取 / 归一化**事实点；
  - 维护事实点之间的**图结构**（节点、索引、关联边）；
  - 执行**退化**（重要性、久未访问、冲突合并、过期策略等）。
- 外部系统**只提交事实材料**，**不负责**建立或维护关系；关系管理 exclusively 在 MCP 内部。

### 原因

- 若由 Host 或执行 Agent 建图，会与各 Host 数据形态强耦合，且无法保证跨场景一致的质量。
- 单一记忆 Agent 便于统一 prompt、审计、退化策略与存储格式。

### 影响

- 禁止在 MCP 工具层暴露「建边 / 删节点 / 改图」等细粒度 API 给 Host。
- 内部可有多阶段流水线（解析 → 去重 → 连边 → 索引），但对外仍只有 **store / retrieve**。

---

## 2026-05-20 — 外部接口：仅存、取；字符串协议

### 设计意图

对外仅两个能力（MCP tools），且参数以 **string** 为原始输入/输出，便于 CLI、其它 Agent 框架、脚本等**开发式对接**：

#### `memory_store`（存入）

| 参数 | 类型 | 说明 |
|------|------|------|
| `content` | string，**必填** | 本次待沉淀的原始文本（单条 episode 即可；可含多事实，由内部 Agent 拆分）。 |
| `source` | string，可选 | Host 标识，如 `agenttest-plan`、`novelagents`，供过滤与分库策略选用。 |
| `kind` | string，可选 | 粗分类，如 `episode`、`note`；不强制枚举。 |
| `correlation_id` | string，可选 | Host 侧关联 ID（如 turn_id、工单号）；用于去重与追溯，**非**结构化业务主键契约。 |

- **语义**：外部仅「投递材料」；**异步**处理 acceptable（先 ACK，再队列 + 记忆 Agent）。
- **性能**：存入**可以慢**；允许排队、批处理、失败重试与死信。

#### `memory_retrieve`（取出）

| 参数 | 类型 | 说明 |
|------|------|------|
| `context` | string，**必填** | Host 本轮**完整上下文**字符串：至少含用户本轮输入；应含 Host 第一层会话记忆摘要（若存在）。 |
| `query_hint` | string，可选 | 补充检索意图；**不能替代** `context`（单独 query 无法保证联想链完整）。 |

| 返回 | 类型 | 说明 |
|------|------|------|
| `hints` | string | 记忆系统 Agent 裁切后的**参考提示文本**，供 Host 注入 system/user 前缀；**不**默认返回整图或全量匹配节点。 |

- **语义**：在掌握 `context` 后，由记忆 Agent **自主决定**返回哪些事实（兼顾「只匹配关键词不够」与「全匹配过多」）。
- **性能**：取出**必须快**；热路径禁止完整 Memory Agent 多轮推理（可用索引 + 轻量 rerank；重型推理仅 store 异步路径）。

### 原因

- 固定结构体（如 TodoList DTO）会绑定某一 Host，阻碍「各种其他场景」复用。
- 检索仅给 `query_hint` 时，无法覆盖「记忆 A 联想记忆 B」类需求；全量返回匹配节点则上下文爆炸。

### 影响

- MCP InputSchema 中**不得**将 Host 专有字段（如 `plan_id`、`steps[]`）标为 required。
- Host 可将内部结构（如 TodoList JSON）**序列化为 string** 写入 `content`；是否在 `content` 首行写约定标签（如 `[source=agenttest-plan]`）由 Host 自选，MCP 仅作可选解析优化。

---

## 2026-05-20 — 集成方式：MCP 服务 + Host 钩子（非执行 Agent 主动调用）

### 设计意图

- 本能力以 **外部 MCP Server** 形式提供。
- **存入 / 取出** 不由 Plan/Behavior 等执行 Agent 通过 `tool_calls` 触发，而由 **Host 进程内钩子**在固定生命周期点调用 MCP（进程内 Client 或 stdio，实现细节见 `ARCHITECTURE.md`）。
- 本 MCP **不**挂载到 Host 执行 Agent 的 `attach_to` / 能力目录渐进披露列表，避免模型误调记忆工具。

### 钩子边界（规范）

| 钩子 | 时机 | 行为 |
|------|------|------|
| **OnTurnRetrieve** | 用户**新一轮**输入进入系统后、`BeginTurn` 之后、主 Agent `Process` 之前 | 同步 `memory_retrieve`；将 `hints` 注入本轮上下文（如独立块 `【跨会话事实参考】`）。Host 可做**零 LLM 轻过滤**（寒暄、空输入、重复输入）。 |
| **OnTurnStore** | **本轮需求处理结束**之后（主 Agent `Process` 返回，成功或失败） | 异步 `memory_store`；`content` 为本轮**完整需求日志**的字符串（见下节）。Host / MCP 双侧均可过滤寒暄。 |

**需求边界（保守默认）**：以 **用户每一轮输入** 对应 Host 内 **一次主 Process** 为一条 episode 的边界。不在本 MCP 内要求主模型智能判定「需求是否结束」。

- 同一轮内 Plan 调节、多步 Behavior 仍属**同一 episode**（Host 负责拼成一段 `content` 再 store 一次）。
- 寒暄等无沉淀价值轮次：Host 或 MCP 过滤后 **no-op store / 可选 no-op retrieve**。

### 原因

- 执行 Agent 主动调记忆会导致 prompt 污染、与 MCP 业务工具争抢步数、且难以保证「取快存慢」的 SLA。
- 每轮输入边界简单、可测，避免 Host 侧复杂的「需求完成」判定器。

### 影响

- 实现 Host SDK 时须提供：`BeforeProcess(ctx, userInput)` 与 `AfterProcess(ctx, episodeContent)` 两类钩子，而非要求业务 Agent 调用 MCP。

---

## 2026-05-20 — Host 侧 episode 内容约定（参考：AgentTest / Plan）

以下为 **推荐 Host 实践**，**非** MCP 协议强制字段。

- **范围**：仅本轮需求。例如 AgentTest 中：本轮 `turn_id` + 本轮 `todolist.Document` 终态 + 门户 `final` 回复；**不**包含其它 TodoList 文件、不**包含**多轮 session buffer 全文。
- **形态**：将上述材料序列化为 **单一 `content` 字符串**（Markdown 或 JSON 文本均可）。
- **示例首行标签（可选）**：`[source=agenttest-plan turn=<id> plan=<id>]`，便于 MCP 内过滤与去重。

AgentTest 对接要点（记入 drift / ARCHITECTURE 时可展开）：

- 钩子落点：`portal.RunRouterTurn`（Web / stdin / Community 统一入口）。
- `retrieve` 的 `context` 可拼接：用户本轮 input + `sessionmemory.PrepareUserContext` 产出（第一层）+ 可选上轮 outcome 一行摘要。

---

## 2026-05-20 — 过滤、退化与 SLA

### 设计意图

| 路径 | 过滤 | SLA |
|------|------|-----|
| Store | Host 规则（长度、寒暄表）+ MCP 内记忆 Agent / 规则二次取舍 | 异步；失败可重试、死信；不阻塞用户门户 |
| Retrieve | Host 轻过滤 + MCP 内快速索引 | 同步；超时则返回空 `hints`，**不得**阻断 Host 主流程 |

**退化**：事实世界须支持长期运行下的合并、降权、删除策略；具体算法由实现决定，但须可配置且可观测（日志 / 指标）。

### 影响

- `memory_retrieve` 失败降级为「无记忆提示」，不可抛致命错误给 Host。
- 短期 TTL 清理属于 Host 第一层；本 MCP 第三层使用**独立**退化策略，默认**非**「全局 1 天删库」（除非人工配置统一策略）。

---

## 2026-05-20 — 与执行 Agent、其它 MCP 的关系

### 设计意图

- 本 MCP 与 filesystem、sqlite、邮件等业务 MCP **正交**。
- 不提供 `list_tools` 给执行 Agent 做「查记忆」；记忆只通过 Host 钩子进入上下文。
- 多个 Host 可共用同一 MCP 实例时，须用 `source` + 可选租户配置隔离事实库（实现阶段在 `ARCHITECTURE.md` 定义）。

### 影响

- 文档与实现中不得将本服务注册为 Behavior/Plan 的 `capabilities.attach_to` 目标。

---

## 文档与护栏（ai-architecture-harness）

本仓库后续应补齐（可分批）：

```text
AGENTS.md                 # 运行时说明（tools 摘要、非宪法）
docs/DESIGN_INTENT.md     # 本文件
docs/ARCHITECTURE.md      # 实现地图（Agent 同步）
docs/ACCEPTANCE_RULES.md  # 可验收规则（取快、存异步、字符串协议等）
docs/GOLDEN_RULES.md      # 事故驱动的硬规则
docs/ARCHITECTURE_DRIFT.md
```

**不可违背的验收锚点（摘要）**

1. `memory_store` / `memory_retrieve` 的必填参数均为 **string** 类型语义。
2. Retrieve 同步路径有**超时上限**且失败不阻断 Host。
3. Store 默认**异步** ACK。
4. 无 Host 可见的图编辑 API。
5. 执行 Agent 能力目录中**不出现**本 MCP 工具名。

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-20 | 初版：第三层记忆 MCP、记忆 Agent、字符串协议、Host 钩子、轮次边界、AgentTest episode 参考实践。 |
