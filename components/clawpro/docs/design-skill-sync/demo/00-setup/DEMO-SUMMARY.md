# 演示总结 · 设计规范对齐机制端到端

> 本次彩排全部命令均真实执行、以退出码/grep 计数为证据；演示改动已全部还原，`git status` 干净（与开场一致）。

---

## (a) 演示总结文案（机制价值 + 本次证据）

### 一句话价值
代码是唯一真相源。设计规范/组件稳定后，**人改代码 → 贴一段 kickoff → AI 读 SOP → 自动沿链路对齐**（代码 → Tier1 沉淀 → Tier2 设计skill → Tier3 走查skill），产出「对不上清单」逐条确认后 reconcile，全程以**退出码 / git diff / grep 计数**为证据。

### 五段场景 · 本次真实证据

| 场景 | 演示点 | 本次实测证据 |
|---|---|---|
| **A** 改现有组件 | P1 双通道 + 三层对齐，退出码 1→0 | 差集命中 `StatusTagMode="banner"` + `--status-tag-banner-bg` + 顺带揪出历史漏声明 `role-user`（exit=1）；sync-tokens 244 token；补 spec §3.4 + refresh-fixtures(6/6)；复跑 exit=0 |
| **B** 新增组件 | onboarding 全流水线 + add -N 门槛 | 未 add-N「检查 0 个/exit0」vs add-N「CHANGED_NO_SPEC/exit1」；verify-manifest **FAILED(Errors:1) → PASSED(Component Specs 41)**；映射表 47 组件、rating-stars 入表 4 处；复跑 exit=0 |
| **C** 改语义规范 | 代码抓不到的规则由 AI 传播 | P1 差集对语义规范**无感**（检查 0 个/exit0）；一条规则联动 **3 文件跨三层**（Tier1 §0.6 + Tier2 card-surface §3.1 + Tier3 §0.B），`diff --stat` = 3 files / 4 insertions |
| **D** 页面模板 | Tier2 独有资产只校验不覆盖 | 基线 11 引用全 OK/exit0；挂错 → `MISS status-badge.md`/**exit=1** → 修回真实 spec → **exit=0** |
| **E** 彩蛋·全量 | 主动豁免、只扰增量 | 检查 **83 组件** / **历史豁免跳过 62** / 候选 **32**（不豁免约 94）/ exit=1；findings 仅作 backlog |

### 三条核心机制信条（可作结论页）
1. **P1 值/枚举一律以代码为准**——防住「文档声称 vs 代码实际」漂移（status-tag 事故防线）。
2. **脚本管确定性、AI 管语义传播**——差集/MANIFEST/存在性靠退出码；语义规则漂移靠 AI 沿 SOP 联动。
3. **只防增量、历史存量豁免**——62 项历史包袱默认挡在增量之外，全量体检才全景抖出。

---

## (b) 产物清单与命名建议

统一存 `docs/design-skill-sync/demo/<场景文件夹>/`，每场景一段录屏 + 若干截图：

```
00-setup/           README.md（总纲/kickoff/还原命令）、DEMO-SUMMARY.md（本文件）
                    A-baseline.png（git status clean）
A-modify-component/ SCRIPT.md、A-recording.mp4
                    A1-code-change.png / A0-trigger.png / A2-diff-exit1.png / A6-diff-exit0.png
B-new-component/    SCRIPT.md、B-recording.mp4
                    B1-addN-contrast.png / B0-trigger.png / B2-diff-exit1.png
                    B3-manifest-failed.png / B3-manifest-passed.png / B6-diff-exit0.png
C-spec-rule/        SCRIPT.md、C-recording.mp4
                    C1-tier1-rule.png / C0-trigger.png / C4-diff-stat.png
D-page-template/    SCRIPT.md、D-recording.mp4
                    D2-broken-ref.png / D0-trigger.png / D3-check-miss-exit1.png / D4-check-ok-exit0.png
E-full-audit/       SCRIPT.md、E-recording.mp4
                    E0-trigger.png / E1-full-exemption62.png / E1-full-findings32.png
```

命名规则：`<场景><步号>-<含义>-<关键态>.png`（关键态如 `exit1`/`exit0`/`failed`/`passed`），录屏 `<场景>-recording.mp4`。

---

## (c) 可发给团队的简短说明

> **【设计规范对齐机制 · 上手说明】**
> 我们把「设计规范/组件改动后，文档和两个 skill 对不上」这个老问题机制化了。**代码是唯一真相源**，用法只有一步：
>
> 1. 在代码里改组件/加规范（`client/src` 的 `.tsx` / `index.css`）；新组件记得 `git add -N`。
> 2. 在 CodeBuddy 贴这段 **kickoff**（增量）：
>    > `对齐 skill 规范。请阅读 docs/design-skill-sync/SOP.md 的 A 段，针对我刚才对 client/src 的改动，按 Stage0→Stage1→Stage2 执行：先跑 diff-code-vs-docs.mjs --since HEAD 出「对不上清单」，逐条确认后 reconcile（不覆盖 Tier2 独有内容），再重抽 fixtures，复跑至退出码 0。`
> 3. AI 会出「对不上清单」，你逐条拍板，它落盘到 Tier1/Tier2/Tier3，复跑到 `exit=0` 收口。
>
> 想体检整体对齐度时，换口令 **`全量体检 skill 对齐度`**（跑 `--full`，只出 backlog、不强制修）。
> 结论都以**退出码**为准（0 无差集 / 1 有候选 / 2 环境错）。细节见 `docs/design-skill-sync/SOP.md` 与 `REQUIREMENT-AND-PLAN.md`。
