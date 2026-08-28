---
name: feedback
description: FEC 反馈治理 Skill。当用户对任意 Skill 的产物提出修改、纠错、不满、补充需求或高度赞赏时，对该反馈做 FEC 2×2 象限判定，并把高价值反馈（象限 A/B）增量落盘为标准反馈单。支持两种调用方式：①交互式（AI 在对话中判断并同轮生成）②事件驱动自动化 A2（PostToolUse 记录每次 Skill 调用作门槛 + UserPromptSubmit 关键词预筛并落盘反馈轮 + 回合级 Stop 经"用过 Skill+有反馈轮"双门槛拉回 AI 按反馈轮精准收口，AI 判定后调 scripts/record_feedback.py 幂等落盘）。触发词：反馈、feedback、这个不对、应该改成、漏了、太啰嗦、记录一下。
metadata:
  short-description: Skill 反馈提炼与治理（FEC）
  version: 1.0.0
allowed-tools:
  - read_file
  - write_to_file
  - replace_in_file
  - search_content
  - execute_command
---

# Feedback SKILL（FEC 反馈提炼与治理）

> 一句话定位：把"用户对 Skill 产物的每一次修改/纠错意见"转化为"可沉淀、可提炼、可回注 Skill 的结构化反馈单"，让 Skill 具备自进化能力。

本 Skill 是原 `.codebuddy/rules/feedback-rule.md` + `.codebuddy/feedback/README.md` 中 FEC 治理逻辑的**单一权威来源**。判定标准与模板以本目录 `references/` 为准。

---

## 一、两种调用模式

本 Skill 必须同时服务两类调用者，二者共用同一套 FEC 判定与落盘规范：

### 模式 A：交互式（对话进行中，由主 AI 调用）
用户在对话里对某个 Skill 产物提出修改/纠错/新需求时，主 AI 在**同一轮**：
1. 修改产物内容；
2. 按本 Skill 做 FEC 判定；
3. 若判定为象限 A/B，生成/追加反馈单。

> 严禁"先改内容、回头再补反馈"。内容修改与反馈记录是并行动作。

### 模式 B：事件驱动自动化（三 hook 回合级精准收口 · 方案 A2）

> **技术分工**：FEC 判定（判象限、写 RCA、写优化项）需要"智力"，只能由 AI 做；确定性动作（记录 Skill 调用、关键词预筛、可靠落盘）由脚本做。二者用 CodeBuddy 官方 hook 事件（v1.16.0+）串成**无需用户主动操作**的闭环。
>
> **关键语义修正**：`Stop` 是**回合级**事件——「主 agent 每完成一次响应（每一轮回复结束）」触发，一个会话**可多次**；并非"整个会话结束仅一次"。真正会话级的 `SessionEnd` 又**不能把 AI 拉回判定**。因此本方案不是"攒到会话末尾统一收口"，而是 **A2：每轮回复结束、按「反馈轮」增量精准收口**。

**三个 hook（实时 + 兜底双保险）**：

1. **PostToolUse hook**（`hooks/skill_usage_recorder.py`，matcher=`use_skill`）：每次用完 Skill 后**确定性追加**一条调用记录到 `.sessions/<session_id>.jsonl`。纯脚本、零 LLM。作用是**收口门槛①的依据**（本会话是否用过 Skill）。
2. **UserPromptSubmit hook**（`hooks/feedback_gate.py`）：命中反馈关键词时，一方面注入 `additionalContext` **实时提醒**主 AI 当场走 FEC（实时线）；另一方面把这条命中的用户输入**落盘为「反馈轮」**到 `.sessions/<session_id>.turns.jsonl`（这是 A2「关键词预筛」环节——预筛放在天然持有 `prompt` 的这里做，Stop 只读结果）。
3. **Stop 编排 hook**（`hooks/clawpro_knowledge_deposit_stop.py`，复用 `hooks/feedback_stop_gate.py` 的 FEC 判定逻辑）：每轮回复结束触发，经**两道节流门槛**后把 FEC 收口要求与知识沉淀要求合并为一次 AI 拉回：
   - 门槛①「本会话用过 Skill」：`.sessions/<sid>.jsonl` 为空直接放行 → 与 Skill 无关的对话 100% 不介入、零 AI 往返；
   - 门槛②「有命中反馈的轮」：仅当反馈轮总数 > 已收口游标时才拉回 → 用过 Skill 但没给疑似反馈的轮（"谢谢/继续"）也不拉 AI。
   通过则 `continue:false + reason` 把 AI 拉回，对「命中反馈但未收口」的那些轮逐个做 FEC 判定与落盘。`stop_hook_active` 防死循环；`<sid>.cursor` 按**反馈轮数**记录已收口进度，保证跨停止周期幂等。

**闭环链路**：
- 实时线：用户当场表达反馈（命中关键词）→ UserPromptSubmit 提醒 → AI 当轮改内容 + 判定 → A/B 调 `record_feedback.py` 落盘。
- 兜底线：命中的反馈轮落盘 → 每轮回复结束 Stop 经双门槛 → 有未收口反馈轮才拉 AI 逐条回顾，**防漏记**。
- 两线共用 `record_feedback.py`，靠 `session_id + offset` **幂等去重**，若实时线已落盘则收口时 AI 自行跳过。

> **游标基准（修正点）**：收口游标按**「反馈轮数」**推进，而非旧版的「skill 调用数」。旧版会在"产物刚生成、用户尚无反馈"的当轮就推进游标，导致用户在**后续轮**给出的真实反馈因无新增 skill 调用而不再触发拉回、被漏记。A2 以「命中反馈的用户轮」为基准，反馈出现在哪轮就在哪轮收口。

> **机制固有边界（诚实告知）**：① 能把 AI 拉回判定的只有回合级 `Stop`，会话级 `SessionEnd` 不能拉回——"攒到整个会话最后由 AI 一次性判定"在当前平台**做不到**，故取回合级精准收口；② Stop 收口依赖回复正常结束，进程被强杀/崩溃时 Stop 可能不触发 → 那次由 UserPromptSubmit 实时线部分弥补；③ 关键词预筛可能漏掉"平静、无关键词的反馈"——词表可放宽，且实时/兜底两线互补降低漏判。

**确定性脚本落盘**：象限 A/B 时调用 `scripts/record_feedback.py`，由脚本保证**幂等去重 / 追加不新建 / 自动建目录 / 按模板渲染**。象限 C/D 脚本自身会拒绝落盘。

`record_feedback.py` 关键入参（详见脚本 `--help`）：

| 参数 | 含义 |
|---|---|
| `--skill` | 本段交互中被使用的目标 Skill 目录名（落盘子目录） |
| `--scene-en` / `--scene-cn` | 文件名场景英文短名（kebab-case）/ 应用场景中文描述 |
| `--quadrant` | FEC 象限 A/B/C/D（C/D 脚本拒绝落盘） |
| `--quote` | 用户反馈原话（可多次传入，禁止粉饰） |
| `--rca-error` / `--rca-root` | Skill 犯了什么错 / 根因清单 key（可多次） |
| `--action` | 优化行动项（目的+手段，可多次；象限 A 必填） |
| `--session-id` / `--offset` | 幂等去重键（同一段交互重复调用不重复写） |

调用示例：
```bash
python3 .codebuddy/skills/feedback/scripts/record_feedback.py \
  --skill requirement-writer --scene-en lobster-doctor-prd --scene-cn '龙虾医生 PRD 生成' \
  --quadrant A --quote '这个不对，凭空捏造了 40% 宽度' \
  --rca-error '编造了无根据的 UI 尺寸' --rca-root missing-blank \
  --action '目的=严禁捏造前端 UI 细节；手段=无提及一律用 ⚠️待补充 占位留白' \
  --session-id "$SESSION_ID" --offset "$OFFSET"
```

**无有效反馈（象限 C/D 或无信号）时不产生任何文件、不产生任何 commit。**

---

## 二、触发信号识别

### 显性信号（关键词/结构）
- 否定/纠错："这个不对""不符合预期""有问题""应该改成""不需要这段"
- 缺漏/冗余："漏了…""少了…""太啰嗦""太简略""删掉…"
- 赞赏（优秀案例）："这个效果很好，记录一下"

### 元原则（最重要，禁止漏判平静反馈）
**不要靠"用户是否情绪不满"判断是否记录，要靠"这个问题是否系统性、会不会在后续同类生成中复发"。**

即使用户**平静地**提一个新需求，只要它暴露了 AI 默认做法中一个**会复发的系统性缺陷**（上一次默认做法被纠正），就属于象限 A，必须记录。每轮收到修改/新需求时先自问：

> "我上一次的默认做法，是不是有系统性问题、正在被用户纠正？"

---

## 三、FEC 2×2 象限判定（判定权威见 references）

完整判定矩阵与四象限处理动作见 → `references/fec-framework.md`

速查：

| 象限 | 性质 | 动作 |
|---|---|---|
| **A** 系统性 Skill 逻辑缺陷 | 格式冗余/废话/技术细节泄漏/幻觉数值 | **100% 记录** → 转化为 SKILL.md 全局否定约束 |
| **B** 隐性知识/规则漏洞 | 业务模型/生命周期理解断层 | **100% 记录** → RCA 转化为知识库升级 |
| **C** 纯业务内容微调 | 本次任务的临时业务决策 | **不记录**，直接在产物里改 |
| **D** 垃圾噪声 | 拼写/情绪宣泄/无实质建议 | **不记录** |

---

## 四、落盘规范

### 4.1 路径与命名
```
.codebuddy/feedback/{active-skill-name}/{YYYY-MM-DD}-{scene-en-name}.md
```
- `{active-skill-name}`：本段交互中被使用的 Skill 目录名（如 `requirement-writer`、`clawpro-test-suite`）。
- 目录不存在则自动创建（`write_to_file` 会自动建父目录）。
- **合法特例（勿误判为 bug）**：当**被反馈的对象就是 feedback skill 自身**时，`{active-skill-name}` = `feedback`，故路径天然为 `.codebuddy/feedback/feedback/{日期}-{场景}.md`。这个"看似嵌套"的 `feedback/feedback/` 是**正确结果、非误操作**，严禁凭路径字面直觉当异常清理。

### 4.2 幂等与"追加不新建"（自动化下的硬要求）
- **去重键**：`session_id + processed_offset`。同一段交互不得被重复记录。
- **连续多轮同一场景**：**追加到同一个反馈文件**（补原话、补 RCA、补优化项），**不新建**文件。
- 判断"是否同一场景"：同一 `active_skill` + 同一功能主题（如都在改"龙虾医生"PRD）→ 视为同一场景，追加。

### 4.3 收口时机（不在用户连续改的过程中反复写）
检测到反馈信号先**攒着、标记待记录**，满足任一条件再增量落盘一次：
- 该 Skill 产物**连续 N 轮无新反馈信号**（用户不改了）；
- 用户**切换到新 Skill / 新任务**；
- **由 Stop 编排 hook（`hooks/clawpro_knowledge_deposit_stop.py`）回合级精准收口**：`Stop` 是"每轮回复结束"触发的回合级事件（非会话级 `SessionEnd`——后者不能把 AI 拉回判定）。编排 hook 复用 `feedback_stop_gate.py` 的"用过 Skill + 有命中反馈轮"双门槛和游标逻辑，把 FEC 收口与知识沉淀要求合并为一次 AI 拉回；FEC 仍只回顾**命中反馈但未收口的那些轮**，游标按**反馈轮数**推进。

### 4.4 反馈单模板
生成/追加内容严格遵循 → `references/feedback-template.md`
四段结构：①基本信息（含**未粉饰的原话**）②FEC 判定 ③RCA 根因 ④SKILL 优化行动项。

### 4.5 优化行动项的"目的优先"原则（元问题）
写优化行动项时，**先锁定用户要解决的真问题/目的，再谈实现手段，严禁把手段当目的**。
- 反例：把行动项写成"用 emoji 色点"（手段当目的）。
- 正例："实现优先级视觉分层，让人一眼区分层级（目的）；手段：纯 .md 可见约束下用 emoji 色点/加粗/缩进等"。

对**纯 .md 交付物**做视觉强调时：目的是让不同层级/类别一眼可辨，手段禁止用 `<span style>`/`<font color>` 等依赖 HTML 渲染的方式，改用纯文本源码下即可见的手段（emoji 色点 🔴🟠🟢、符号、加粗、缩进、分组标题）。

---

## 五、噪声拦截（省钱兼防污染）
- 象限 C → 直接在产物文本改掉，**不生成**任何反馈文件。
- 象限 D → **不生成**任何文件，交互式下礼貌回复即可；headless 下静默退出。

---

## 六、与旧规则/框架文件的关系
| 文件 | 新职责 |
|---|---|
| `skills/feedback/`（本 Skill） | **单一权威源**：触发、判定、落盘、模板全在此 |
| `rules/feedback-rule.md` | 收敛为**薄触发层**，仅声明"何时该调用本 Skill"，判定/模板不再重复 |
| `feedback/README.md` | FEC 框架总览/导读，指向本 Skill 为权威 |
| `feedback/{skill}/*.md` | 反馈产物（证据），由本 Skill 生成/追加 |

**流向**：反馈单（证据）→ 提炼通用规则 → 注入目标 Skill 的 SKILL.md → 后续生成自动遵循。

---

## 参考文件
- `references/fec-framework.md`：FEC 2×2 判定指南（判定权威）
- `references/feedback-template.md`：反馈单模板（生成权威）
