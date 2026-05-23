# Memory MCP 记忆体机制 — 推荐实现方案（供专家评审）

> **文档性质**：在 As-Is 与宪法基础上，**Phase-2b～2e 批准落地路径**（历史方案全文）。  
> **读者**：外部专家、实现 Agent。  
> **日期**：2026-05-23（v2：合并专家评审结论）  
> **状态**：**Phase-2b～2d 已完成**（2026-05-23）；**Phase-2e 暂缓**（2026-05-24，见 §13）  
> **进度总表**：`IMPLEMENTATION_PROGRESS.md`

**同目录文档**

| 文档 | 角色 |
|------|------|
| `DESIGN_INTENT.md` | 宪法（不可违背） |
| `CURRENT_IMPLEMENTATION_ARCHITECTURE.md` | 当前代码 As-Is |
| `ARCHITECTURE.md` | 模块地图与阶段 |
| `ARCHITECTURE_DRIFT.md` | 宪法 vs 代码差距 |
| `ACCEPTANCE_RULES.md` | 分阶段验收清单 |
| `README.md` | 阅读顺序索引 |

---

## 0. 执行摘要（Executive Summary）

| 维度 | 推荐 |
|------|------|
| **核心范式** | Store：**原子消解与实体对齐**（异步，可含 LLM）；Retrieve：**拓扑激活与自适应剪枝**（同步，默认无 LLM） |
| **图存储** | **不引入 Neo4j**；`facts.jsonl` + `edges.jsonl`（或 SQLite 单文件），每次 retrieve 构建内存邻接表 |
| **LLM 边界** | Store 异步 **1 次**结构化抽取（失败回退 rules）；Retrieve **默认 BM25×激活能级×weight**；LLM prune **默认关闭** |
| **实体对齐** | 硬规则规范化 + cosine≥0.92 自动合并；**禁止** Store 主链默认 LLM 对齐；模糊带 0.85–0.92 仅 2d 异步增量 |
| **Host 契约** | 不变：仅 `memory_store` / `memory_retrieve` 字符串；图/三元组/Ontology **不暴露** |
| **落地顺序** | 2b ✅ → 2c ✅ → 2d ✅ → **2e ⏸ 暂缓**（retrieve LLM prune，默认不实现） |

---

## 1. 不可违背的约束（与宪法对齐）

1. **对外工具**：仅 `memory_store`、`memory_retrieve`；参数/返回均为 string 语义 JSON。  
2. **存慢取快**：Store 同步 ACK + 异步 job；Retrieve 同步，**总预算建议 300ms**（可配置），超时 → 空/弱 hints，**不阻断 Host**。  
3. **执行 Agent 不直连记忆**：记忆仅经 Host 钩子（OnTurnRetrieve / OnTurnStore / DecideRoute 解析 route 块）。  
4. **图管理在 MCP 内**：禁止 Host 侧「建边/删节点」类 MCP 工具。  
5. **负反馈必备**：pitfall / failed 路径须进入事实库并抑制 Exec-Simple（`exec_simple_match=no`）。  
6. **可审计**：`episodes/` 永久保留；fact 逻辑删除/降权，不依赖物理抹除历史行。

---

## 2. 目标架构总览

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                        memory_store (同步 ACK)                           │
└───────────────────────────────────┬─────────────────────────────────────┘
                                    ▼
                    ┌───────────────────────────────┐
                    │   Store: 原子消解链 (异步)     │
                    │  filter → episode 落盘 → job   │
                    └───────────────┬───────────────┘
                                    ▼
        ┌───────────┐   ┌──────────────┐   ┌────────────┐   ┌────────────┐
        │ Template  │ → │ RulePreparse │ → │ LLM Extract│ → │ Validate   │
        │ Selector  │   │ (无 LLM)     │   │ (结构化)   │   │ L0/L1/L2   │
        └───────────┘   └──────────────┘   └─────┬──────┘   └─────┬──────┘
                                                  ▼                  ▼
                                        ┌─────────────────────────────┐
                                        │ Merge + Degenerate          │
                                        │ (硬规则 + 可选 LLM 对齐)    │
                                        └──────────────┬──────────────┘
                                                       ▼
                                        facts.jsonl + edges.jsonl
                                                       │
┌──────────────────────────────────────────────────────┴──────────────────┐
│                     memory_retrieve (同步, budget≤300ms)                     │
└───────────────────────────────────┬───────────────────────────────────────┘
                                    ▼
                    ┌───────────────────────────────┐
                    │ Retrieve: 拓扑激活链         │
                    └───────────────┬───────────────┘
                                    ▼
        ┌───────────┐   ┌──────────────┐   ┌────────────┐   ┌────────────┐
        │ filter    │ → │ Seed Anchor  │ → │ Weighted   │ → │ Prune      │
        │           │   │ (BM25/关键词)│   │ BFS 1~2hop │   │ BM25默认   │
        └───────────┘   └──────────────┘   └────────────┘   │ LLM 可选   │
                                                             └─────┬──────┘
                                                                   ▼
                                                        HintComposer + memory-route
```

---

## 3. 内部数据模型

### 3.1 双层事实表示（兼容现网）

为 **不打断** 现有 Host hints 与 `parseExperienceFromHints`，采用 **双轨并行**：

| 轨 | 用途 | 形态 |
|----|------|------|
| **Summary Fact** | Host hints、memory-route、人类可读 | 一条/episode 主摘要（类似现 `Fact.Text`） |
| **Atomic Triple** | 图扩散、实体对齐、退化 | 多条三元组 + 图节点/边 |

```go
// 内部结构（不暴露 MCP tool）
type AtomicTriple struct {
    ID           string
    EpisodeID    string
    SubjectID    string   // 规范化实体节点 ID
    Predicate    string   // 受控谓词表
    ObjectID     string
    SubjectType  string   // ENTITY|ACTION|STATE|CONCEPT
    ObjectType   string
    Evidence     string   // episode 原文摘录，≤300 字
    Confidence   float64
    IsPitfall    bool
}

type GraphEdge struct {
    From       string
    To         string
    Type       string   // uses|triggers|caused_by|depends_on|supersedes|pitfall|similar
    Weight     float64
    EpisodeID  string
}
```

### 3.2 内部 Ontology v1（`source=agenttest-plan`）

**节点类型（NodeType）**

| 类型 | 说明 | 示例 |
|------|------|------|
| `ENTITY` | 工具、路径、项目、环境 | `filesystem`, `WorkSpace`, `memory-mcp.exe` |
| `ACTION` | 可执行操作 | `list_directory`, `write_file`, `Exec-Simple` |
| `STATE` | 结果/错误/状态 | `completed`, `failed`, `Connection Timeout` |
| `CONCEPT` | 配置/模式/抽象 | `exec_simple`, `tier=2` |

**谓词/边类型（Predicate / EdgeType）** — 封闭集合，LLM JSON Schema 枚举

| 边类型 | 含义 |
|--------|------|
| `uses` | ACTION/ENTITY → ENTITY/CONCEPT |
| `triggers` | ACTION → STATE |
| `caused_by` | STATE → STATE/CONCEPT |
| `depends_on` | ENTITY → ENTITY |
| `produces` | ACTION → ENTITY（artifact 路径） |
| `supersedes` | 新事实 → 旧事实 |
| `pitfall` | 失败模式关联（retrieve 时惩罚） |
| `similar` | 任务相似（可由 tags 共现推导，规则备选） |

**模板文件**：`templates/agenttest-plan.yaml` 定义 section 映射、必填谓词、pitfall 触发条件（`outcome=failed` | `ProcessError` 非空 | plan `status=failed`）。

### 3.3 持久化布局（在现 `data/` 上扩展）

```text
data/
  facts/facts.jsonl          # Summary Fact（主检索索引，兼容现网）
  graph/nodes.jsonl          # 可选：实体节点目录
  graph/edges.jsonl          # 持久边（扩散依据）
  atoms/atoms.jsonl          # 原子三元组（审计+对齐）
  episodes/...
  jobs/...
```

**权重字段（Summary Fact 与 Node 共用逻辑）**

| 字段 | 初值依据 | 后续变化 |
|------|----------|----------|
| `weight` | success: 1.0；pitfall: 0.3；普通记录: 0.6 | 被 supersede ×0.2；久未 retrieve 衰减 |
| `confidence` | LLM/规则输出 | Check 失败则降 confidence |
| `last_active` | store/retrieve 时更新 | 退化公式输入 |
| `access_count` | retrieve 命中 +1 | 可选强化 |

---

## 4. Store 链：原子消解与实体对齐（细节）

### 4.1 流水线步骤

| 步 | 名称 | LLM | 输入 | 输出 |
|----|------|-----|------|------|
| S0 | filter | 否 | content | skip / continue |
| S1 | writeEpisode | 否 | 原始 content | episodes/*.md |
| S2 | enqueue | 否 | job meta | jobs/pending |
| S3 | selectTemplate | 否 | source, kind | template id |
| S4 | rulePreparse | 否 | content sections | tools[], artifacts[], outcome, user_req |
| S5 | llmExtract | **是（1 次）** | episode + preparse + template + topK 已有 summary | atoms[], summary_fact, proposed_edges, supersede_ids |
| S6 | validateL0 | 否 | atoms vs preparse | 剔除幻觉工具/路径 |
| S7 | validateL1 | 可选 LLM | 与高 weight 事实冲突 | A更新/B忽略/C共存 |
| S8 | mergeGraph | 硬规则为主 | 新 atoms/edges | 更新内存图 |
| S9 | degenerate | 规则 | supersede_ids, pitfall | 调 weight / 写 supersedes 边 |
| S10 | persist | 否 | | append/replace jsonl |

**失败策略**：S5 超时/JSON 非法 → **完整回退** 现 `ExtractFromEpisode`（单条 summary fact）。

### 4.2 LLM Extract 的 JSON Schema（核心）

```json
{
  "summary": {
    "text": "历史需求: ... 结果: completed 工具: ...",
    "outcome": "success|failed|unknown",
    "tools": ["filesystem__list_directory"],
    "artifacts": ["WorkSpace/foo.txt"],
    "tags": ["WorkSpace", "filesystem"]
  },
  "atoms": [
    {
      "subject_type": "ACTION",
      "subject": "list_directory",
      "predicate": "triggers",
      "object_type": "STATE",
      "object": "completed",
      "confidence": 0.9,
      "evidence": "门户回复摘录..."
    }
  ],
  "edges": [
    {"from": "node:filesystem", "to": "node:WorkSpace", "type": "uses", "weight": 0.8}
  ],
  "supersede_fact_ids": ["fact-xxx"],
  "pitfall": false
}
```

**Prompt 锚定原则（写入 template，非自由发挥）**

- 具独立生命周期的**名词** → ENTITY；**错误/结果** → STATE；**行为** → ACTION。  
- 必须形成 **subject–predicate–object**；predicate 仅允许 Ontology 枚举。  
- tools/artifacts **不得超出** RulePreparse 列表（L0 再滤一遍）。

### 4.3 实体对齐（嵌入/合并）

**第一层 — 硬规则（无 LLM）**

```text
nodeID = normalize(type + "|" + canonicalText)
canonicalText = lower(trim) + 去版本号可选规则（Go 1.26 → go）
若 nodeID 已存在 → 更新 last_active, access_count++
若 summary 同 correlation_id → ReplaceByCorrelation（保持现逻辑）
```

**第二层 — Embedding 硬合并（无 LLM，Store 主链必走）**

- 硬规则未命中后，计算新节点与已有节点 embedding 余弦相似度  
- **cosine ≥ 0.92**：视为同一实体，合并 nodeID，更新 `last_active`；不新建节点  
- 技术域（Tools/Path/Error）下硬规则 + 0.92 可覆盖约 70% 对齐需求（专家结论）

**第三层 — LLM 对齐（2d，异步增量，禁止阻塞 Store）**

- **仅**处理模糊带：**0.85 < cosine < 0.92** 且该节点 `access_count` 高于阈值  
- 独立低优先级队列异步执行 mini LLM：`是否同一实体？Y/N`  
- Y → `supersedes` 边 + 旧 fact `weight *= 0.2`  
- **绝不**在 S5–S10 主 Store 链上同步等待 LLM 对齐（专家：避免吞吐与成本崩塌）

### 4.4 冲突消解与退化（规则表）

| 场景 | 动作 |
|------|------|
| 新 success 覆盖同 intent 旧 success | `supersedes` 边；旧 weight×0.2 |
| 新 pitfall 与旧 success 同 tags | 保留两者；retrieve 时 pitfall 抑制 `exec_simple_match` |
| LLM 判 A（更新） | 执行 supersede |
| LLM 判 B（忽略） | 丢弃新 atoms 中冲突部分 |
| LLM 判 C（共存） | 均保留，降低新条 confidence |
| 无法判定 | **共存** + 降 confidence（保守） |

### 4.5 防幻觉 Check（三层）

| 层 | 机制 | 实现成本 |
|----|------|----------|
| **L0** | tools/artifacts/outcome 必须 ⊆ RulePreparse | 低，必做 |
| **L1** | 每条 atom 必带 evidence；与 episode **Fuzzy 锚定**（见 §4.5.1） | 低，必做 |
| **L2** | 与高 weight 事实冲突时 mini LLM 三选一 A/B/C | 中，仅冲突时触发 |

#### 4.5.1 L1 Fuzzy Evidence 锚定（专家 Q5：禁止纯 `strings.Contains`）

LLM 抽取的 evidence 常做大小写/标点/空格微调，硬子串误判率可达 30%+。

**推荐实现（无 LLM）**：

```text
norm(s) = 去空白/标点后的小写归一化（或 Token 集合）
对 evidence 在 episode 上计算:
  方案 A: 归一化后子串包含
  方案 B: Token 集合 Jaccard ≥ 0.85
  方案 C: 字符 bigram 重合度 ≥ 0.85
通过任一即 L1 通过；否则丢弃该 atom 或降 confidence
```

**不采用**：为 L1 单独调用 LLM。

---

## 5. Retrieve 链：拓扑激活与剪枝（细节）

### 5.1 流水线步骤与预算

| 步 | 名称 | 目标耗时 | LLM |
|----|------|----------|-----|
| R0 | filter | <1ms | 否 |
| R1 | loadGraph | <10ms | 否 | 读 jsonl 建 `map[nodeID][]Edge` |
| R2 | seedAnchor | <15ms | 否 | BM25 + 现 MatchScore 融合 |
| R2b | early exit | 0 | 否 | 若无 seed 且 maxScore<0.35 → 空 hints |
| R3 | weightedBFS | <5ms | 否 | depth≤2, 见 §5.2 |
| R4 | prune | <20ms | **默认否** | 复合打分 Top-K |
| R4' | llmPrune | <250ms | 可选 | `RETRIEVE_PRUNE=llm` |
| R5 | composeHints | <5ms | 否 | 含 memory-route |

**总预算**：300ms；超时跳过 R4'，用 R4 结果。

### 5.2 Weighted BFS（推荐算法 + 专家护栏 A/B）

```text
输入: seedNodes[] 每项带 initialEnergy（来自 anchor 分数）
参数: maxDepth=2, stepAttenuation=0.75, minEnergy=0.25

对每个 seed 初始化 activated[id]=initialEnergy
BFS 按 depth 分层，维护 visited 或「路径已访问」防环（专家 B）:

  对边 u→v:
    若 v 已在当前路径 visited → 跳过（防 similar 双向环）
    outDegPenalty = ln(3 + OutDegree(u))     // 专家 A：超级节点出度惩罚
    类型 pitfall:   factor = 0.5
    类型 supersedes: 不向前扩散
    其他:           factor = stepAttenuation / outDegPenalty

    nextEnergy = current * edge.W * factor
    若 nextEnergy > activated[v]: 更新；若 v 未访问则入队

输出: activated 中 energy ≥ minEnergy 的 fact/node 集合
```

**出度惩罚动机**：`completed`、`filesystem` 等 hub 节点若不惩罚，一次 BFS 可激活半图；惩罚后能量聚焦在**稀有、特异性**路径上。

**防环动机**：`similar` 等边可能形成 A↔B 环，须 `visited` 或按 depth 只访问一次（取 max energy 策略）。

**与「联想」的关系**：用户只说「编译 Go 项目」→ seed 锚到 `Go 1.26` → BFS 沿 `caused_by`/`pitfall` 拉到「代理未开导致 Timeout」—— **无需 retrieve 阶段 LLM 推理图结构**。

### 5.3 种子锚定（颗粒度控制）

```text
tokens = textutil.Tokenize(context + query_hint)
对每个 fact/node 计算:
  anchorScore = 0.6 * MatchScore(existing) + 0.4 * BM25(tokens, fact.text + tags)
取 Top-3 为 seeds；若 Top1 < 0.35 → early exit（防寒暄/无关联想）
```

### 5.4 剪枝：默认 BM25 复合打分（专家 Q4 已确认）

**结论（专家评审）**：在 AgentTest 这类**高频、高确定性、技术内网**场景下，**BM25 × 激活能级 × fact.weight** 的多维打分**工程上完全足够**，且比 1B–3B mini 模型更稳定、无时延抖动。拓扑扩散已在召回阶段注入结构关联，优于纯向量/关键词检索。

**默认（R4）— 复合打分（`MEMORY_MCP_RETRIEVE_PRUNE=bm25`）**

```text
finalScore = α * bm25(context, fact.text) + β * activatedEnergy + γ * fact.weight
            - δ * isPitfall
推荐初值: α=0.4, β=0.4, γ=0.2, δ=0.5
取 Top-3 facts → Markdown hints + memory-route
```

**必须启用 LLM Prune（R4'）的场景** — 仅当 `RETRIEVE_PRUNE=llm` 时：

| 场景 | 说明 |
|------|------|
| 候选过多且语义冲突 | BFS 拉出 **10+** 条，且存在强**时效/上下文**冲突（如同名配置在不同项目） |
| 用户意图极难用 BM25 表达 | context **极长且含糊**，或强反讽/暗喻（AgentTest 主场景**少见**） |

**可选（R4'）— 单次 LLM**

- 模型：Haiku / GPT-4o-mini / 本地 Qwen2.5-1.5B（`MEMORY_MCP_PRUNE_MODEL`）  
- 输入：用户 context + ≤15 条候选（每条 ≤120 字）  
- 输出：筛选后 3 条 + hints 正文  
- 硬超时：纳入 retrieve 总预算 300ms；超时 **回退 R4**  
- **禁止**多轮 tool；**禁止**访问全图  

**2e 阶段**再实现 R4'；**2b 验收仅要求 R4**。

### 5.5 memory-route 生成（与 Host 对齐）

```text
top = 最高分非 pitfall 的 success fact（若有）
若存在 pitfall 且与当前 intent 高重叠 → exec_simple_match=no, 附一行避坑说明
elif top.score >= 0.75 且 outcome in (success, completed) → exec_simple_match=yes
else → exec_simple_match=no
confidence = 归一化 finalScore（cap 1.0）
```

Host `DecideRoute` 仍用现有阈值（0.75）与 tier 护栏，**不迁移到 MCP 内**。

---

## 6. 与 Host（AgentTest）的接口（保持不变）

| 钩子 | 变更 |
|------|------|
| OnTurnRetrieve | 无协议变更；hints 质量提升 |
| OnTurnStore | 无协议变更；建议 Host 在 `ProcessError`/`failed` 时仍 store（已支持） |
| DecideRoute | 无变更；解析 `---memory-route---` |

**建议 Host 小改进（非必须）**：`retrieve` 的 context 拼接第一层 session 摘要（宪法已推荐），提升 anchor 质量。

**环境变量（MCP 进程，经 `mcp_env` 传入）**

| 变量 | 默认 | 说明 |
|------|------|------|
| `MEMORY_MCP_DATA_DIR` | `./data` | 已有 |
| `MEMORY_MCP_LLM_EXTRACT` | `1` | Store 是否用 LLM |
| `MEMORY_MCP_LLM_API_BASE` | - | OpenAI 兼容 |
| `MEMORY_MCP_LLM_MODEL` | - | 抽取用 |
| `MEMORY_MCP_RETRIEVE_BUDGET_MS` | `300` | retrieve 总超时 |
| `MEMORY_MCP_RETRIEVE_PRUNE` | `bm25` | `bm25` \| `llm` |
| `MEMORY_MCP_CONSOLE_*` | - | 已有 |

---

## 7. 代码包结构（推荐）

```text
AgentTestMemoryMCP/internal/
  memoryagent/
    pipeline.go       # Store job 编排
    template.go       # YAML 模板加载
    preparse.go       # RulePreparse（从 rules.go 抽离）
    llm_extract.go    # 结构化抽取 + 回退
    validate.go       # L0/L1(fuzzy)/L2
    fuzzy.go          # evidence↔episode 锚定（Jaccard/归一化子串）
    merge.go          # 硬规则合并
  graph/
    load.go           # jsonl → MemoryGraph
    bfs.go            # Weighted BFS
    persist.go        # 写 edges/atoms
  index/
    bm25.go           # 轻量 BM25（或小规模 embedding）
  retrieve/
    anchor.go         # seedAnchor
    prune.go          # bm25 / optional llm
    compose.go        # hints + route（从 hints.go 演进）
  degenerate/
    governor.go
  llm/
    client.go         # 超时、重试、JSON schema
  facts/              # 扩展字段，兼容旧行
  engine/
    factworld.go      # 编排 Store/Retrieve
```

**Console**：改为读 `edges.jsonl` + `atoms.jsonl`，3D 图与线上一致。

---

## 8. 分阶段交付（PR 粒度）

| 阶段 | 状态 | 交付物 | 验收标准 |
|------|------|--------|----------|
| **2b** | ✅ 完成 | `edges.jsonl`、pitfall、Weighted BFS（**出度惩罚+防环**）、BM25 prune、图加载 | Baseline：边界测试通过；无 retrieve LLM |
| **2c** | ✅ 完成 | template、llm_extract、L0、**L1 fuzzy**、S5 失败回退 rules | `llm_store_smoke`；atoms.jsonl |
| **2d** | ✅ 完成 | supersede 退化、**embedding≥0.92 硬合并**、模糊带异步 LLM 对齐、access 衰减 | supersede 可观测；Store 不阻塞 |
| **2e** | ⏸ **暂缓** | 可选 R4' LLM prune | **不实施**；默认保持 `bm25`（§13） |

**每阶段不破坏**：stdio MCP 双工具、Host 无需改代码（2b～2d）——**已验证**。

**实施结论（2026-05-24）**：2b～2d 已按专家顺序落地；2e 经产品与实测评估**暂不排期**。

---

## 9. 明确不做（避免范围膨胀）

- Neo4j / 外部图数据库（除非单机性能不足，再评估 SQLite 图表）  
- Host 可见的图编辑 MCP 工具  
- Retrieve 多轮 Agent / ReAct  
- 用 SKILL 包替代 Memory 主闭环  
- 在 retrieve 热路径默认调用强模型  

---

## 10. 专家评审结论（已合并入正文）

### 10.1 开放问题结案状态

| ID | 问题 | 状态 | 决议摘要 |
|----|------|------|----------|
| Q4 | BM25 是否够替代 mini LLM | **已结案** | 默认足够；见 §5.4 |
| Q5 | L1 子串是否够 | **已结案** | 必须 Fuzzy，阈值约 85%；见 §4.5.1 |
| Q6 | 硬规则+0.92 与 Store LLM 对齐 | **已结案** | 0.92 合理；Store **禁止**默认 LLM 对齐；见 §4.3 |
| Q1–Q3 | 双轨/Ontology/BFS 参数 | **实施中裁定** | 2b 用现默认值；上线后用指标调参 |

### 10.2 专家肯定的方案要点（保持不动）

- **S5 失败 → 回退单条 Summary Fact**：LLM 故障仍保留线性 RAG 底座（冷启动降级）。  
- **supersedes + 访问衰减**：解决图谱只增不减。  
- **Host 契约不变**：拓扑封装在 MCP 内，Host 只消费 string hints。

### 10.3 专家补充的工程护栏（已写入 §5.2）

- **A. 出度惩罚**：`factor /= ln(3 + OutDegree(u))`，抑制 `completed`/`filesystem` 等 hub 炸图。  
- **B. BFS 防环**：`visited` 或路径去重，防止 `similar` 双向边死循环。

### 10.4 实施批准与结案

- **2026-05-23**：专家评审批准按 §8 排期；优先 2b Baseline，再 2c/2d。  
- **2026-05-24**：**2b～2d 已交付**；集成测试（`memory_boundary_test`、`memory_complex_test`、`llm_store_smoke`）通过。  
- **2e**：**暂缓**，理由见 §13 与 `IMPLEMENTATION_PROGRESS.md`。

---

## 11. 与专家对话的采纳/校准对照（完整）

| 专家观点 | 本方案态度 |
|----------|------------|
| 原子消解 + 拓扑激活双链路 | **完全采纳** |
| Store LLM 结构化三元组 | **采纳**（2c，带回退） |
| 内存邻接表、不引入 Neo4j | **采纳** |
| Retrieve 图 BFS 联想 | **采纳**（2b） |
| Retrieve 默认 BM25，LLM prune 可选 | **专家确认**；已写入 §5.4 |
| L1 必须 Fuzzy，禁止硬子串 | **专家确认**；已写入 §4.5.1 |
| cosine≥0.92 硬合并；Store 禁止默认 LLM 对齐 | **专家确认**；已写入 §4.3 |
| 模糊带 0.85–0.92 异步 LLM 对齐 | **采纳**（2d） |
| BFS 出度惩罚 + 防环 | **采纳**（2b） |
| 权重 + 访问衰减 / supersedes | **采纳** |
| 防幻觉 L2 三选一 A/B/C | **采纳**；默认共存保守 |

---

## 12. 分阶段实施检查清单（结案）

### 2b ✅

- [x] `graph/edges.jsonl` 持久化；Store job 写边  
- [x] `graph/load.go` + `graph/bfs.go`（出度惩罚 + visited）  
- [x] `index/bm25.go` + `retrieve/pipeline.go` 复合打分  
- [x] pitfall + retrieve 抑制 `exec_simple_match`  
- [x] 单元测试：hub / similar 环 / BM25  
- [ ] Console 读 `edges.jsonl`（**待办 P2**，retrieve 已用持久边）  
- [x] 漂移与 As-Is 文档已更新至 2d  

### 2c ✅

- [x] template + preparse + LLM extract + L0/L1 fuzzy  
- [x] S5 失败回退 rules；`atoms.jsonl`  

### 2d ✅

- [x] supersede + 访问衰减 + retrieve touch  
- [x] embedding≥0.92 硬合并；0.85–0.92 异步对齐  

### 2e ⏸ 暂缓

- [ ] R4' LLM prune — **不排期**（见 §13）

---

## 13. Phase-2e 暂缓决策（2026-05-24）

**决定**：**暂不实现** retrieve 热路径上的可选 LLM 剪枝（R4'）。

| 维度 | 说明 |
|------|------|
| **产品** | AgentTest 技术内网场景下，BM25×能级×weight 已满足联想与路由需求 |
| **专家** | Q4 已结案：默认 BM25 足够；LLM prune 仅少数高冲突/极含糊场景 |
| **工程** | 每轮可能双次 retrieve；加热路径 LLM 增加延迟与运维面，边际收益小 |
| **实测** | 2b～2d 闭环后边界/复杂/LLM store 测试均通过，无「必须 2e」信号 |

**保留**：环境变量 `MEMORY_MCP_RETRIEVE_PRUNE` 文档占位为 `bm25`（默认）；若未来实施须满足：纳入 `RETRIEVE_BUDGET_MS`、失败回退 bm25、禁止多轮 Agent。

**触发再评估**（需观测指标，非单点需求）：hints 多近似 fact 噪声干扰 Plan；多租户共库 BM25 失效；有数据证明 BM25 不足且非 Store/图问题。

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-05-23 | v1：初版推荐实现方案 |
| 2026-05-23 | v2：合并专家 Q4–Q6、出度惩罚、防环、L1 Fuzzy、对齐分层、实施批准与 2b 检查清单 |
| 2026-05-24 | v3：2b～2d 结案；§13 2e 暂缓；§8/§12 进度更新 |
