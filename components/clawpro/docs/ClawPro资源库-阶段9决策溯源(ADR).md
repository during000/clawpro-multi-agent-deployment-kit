# ClawPro 资源库 · 阶段 9 决策溯源（ADR）

> 本文是「资源 ↔ skill 连接」（阶段 9）的决策记录，供日后人或 AI 溯源背景，避免误改误删。
> 配套文档：`docs/ClawPro资源库-新资源接入SOP.md`（怎么加新资源）、`docs/ClawPro图标与资源库建设计划.md`（全量计划）、`docs/ClawPro图标与资源库背景目标.md`（背景目标）。
> 状态：已落地（2026-06-22）。本文所述数据均经真实脚本 / 产物 / git 历史核实。

---

## 1. 阶段 9 解决什么问题（一句话）

让设计治理 skill（`clawpro-portable-design-skill`）在生成「当前项目页面」时，**不再凭语义瞎猜图标路径**，而是从一张「已审计、磁盘验证、槽位对口」的候选表里选图——保证生成结果**不出离谱事故、稳定可复现**。

注意定位：阶段 9 提升的是**正确性 / 安全性 / 可复现性**，不是「自动让页面更好看」。审美仍取决于候选池质量与 AI 在池内的选择。

---

## 2. 产物与链路（很简单，别被"连接"二字吓到）

```
inventory（848 项资源台账，阶段 1~6 逐阶段审计）
        │
        ├── manual-overrides/component-resource-map.json（阶段 8：9 个槽位的风险约束）
        │
   build:skill-map（确定性派生，零猜测）
        │
        ▼
resource-skill-map.json（9 槽位 / 158 候选 / 14 红线）  ◀── skill 生成页面时只读这个
        │
   check:skill-map（守门员：任一不符即 fail-fast）
```

**三个产物**：
| 文件 / 命令 | 角色 |
|---|---|
| `client/src/design-assets/resource-skill-map.json` | 精简稳定的选图映射（skill 唯一读取项）|
| `scripts/build-resource-skill-map.mjs`（`npm run build:skill-map`）| 从 inventory 确定性重建映射，请勿手改产物 |
| `scripts/check-resource-skill-map.mjs`（`npm run check:skill-map`）| 10 项校验守门员，失败即退出 1 |

**当前真实快照**（`resource-skill-map.json` summary）：
- 9 槽位：`admin-sidebar`(22) / `card-left-icon`(67) / `number-card`(22) / `file-type`(21) / `run-status`(8) / `feature-card`(4) / `agent-avatar`(7) / `channel-icon`(6) / `brand-logo`(1)
- 候选合计 158，其中红线 14（agent-avatar 7 + channel-icon 6 + brand-logo 1）

> **修订补注（2026-06-23）**：`card-left-icon` 槽位由 64 增至 67（候选合计 155→158），新增 3 枚云开发「核心能力」卡片左侧渐变图标 `cloud-database` / `cloud-function` / `static-hosting`（`client/public/assets/admin-cloud-dev/`，own-svg / gradient-solid / 端别 admin，与既有 card-left-icon 一致）。已在 `manual-overrides/gradient-style.json`（gradient-solid）与 `classification.json`（confirmed）登记为事实源，经全链路 `scan→classify→detect-duplicates→apply-governance→build-canonical→build:skill-map→check:skill-map` 重生成校验通过（红线仍 14，非红线增量）。

---

## 3. 核心决策 A：以 inventory 为本项目真相，registry 降格为可移植样例

### 决策内容
- **本项目资源真相 = `inventory`**（`generated/resource-inventory.generated.json`，848 项，脚本扫描 + 逐阶段审计）。候选准入由 inventory 的审计字段确定性派生。
- **`icon-registry.example.json` 降格为「可移植身份样例」**：仅在跨仓移交时供宿主仓建立正式 registry 参照，**不**作为本项目候选准入闸门。

### 决策依据（均有硬证据）
1. **时间线（git 实证）**：`icon-registry.example.json` 诞生于 **2026-06-06**（随 skill 提交 `cfed8eab 添加 ClawPro 设计治理 Skill`，文件 `updatedAt` 自标 6-05）；而资源库建设计划 / 全量扫描整理 **2026-06-13 起**才启动，晚 7~8 天。即 registry **产生在整个资源库计划之前**。
2. **它自己的 notes 自述**："**初始**登记根目录 `icon/` 下的业务 SVG……"——关键词「初始」「只扫 `icon/` 一个目录」暴露其真实身份：skill 第一版的**兜底登记**，范围极窄，故仅 28 条。
3. **量级与交叠**：本项目 inventory 848 项真实资源，能在那 28 条样例里登记上的**仅 1 项**。若按旧口径「必须在 registry 里 `approved` 才能用」，等于把 847 项真实资源全挡在门外，候选库几乎为空——门槛卡错了对象。

### 被本决策推翻的旧口径（阶段 9 据实修正）
原多处文档 / 脚本写「registry 是资产身份**唯一事实源**、skill-map **只引用**其 `status=approved` 资产、资产变更**先改 registry**」。这套口径与决策 A 直接冲突，已在以下位置修正并标注「口径修正（阶段 9）」：
- `docs/ClawPro图标与资源库建设计划.md`（§5.5 / §6.3 / §6.5 / 看板 / 阶段9基线段 / 前置约束第 3 条）
- `docs/ClawPro图标与资源库背景目标.md`（§六 / §七 7.1 / §八第 11 条）
- skill `references/assets-icons.md`（§1 / §5.5 / §9）
- `scripts/classify-resources.mjs`（注释措辞，逻辑零改动）

> **红线硬约束在本次修正中完全保留**：品牌 Logo / 渠道图标 / Agent 头像（14 项）仍禁改色、禁进普通图标槽位、跨仓由宿主注入。修正只动「谁是事实源」，不动安全红线。

---

## 4. 选图与兜底机制（为什么"匹配不准也不会离谱"）

skill 选图流程（`assets-icons.md` §5.5）：先判断命中哪个组件槽位 → 在该槽位 `candidates` 里按风格/尺寸/场景挑 → **未命中或无合适候选时回落**。

**三层兜底**：
1. **磁盘校验兜底（防裂图）**：`build` 与 `check` 脚本对每个 `file` 候选做 `fs.existsSync`，文件不存在直接 fail-fast，不生成产物。→ 158 候选**保证真实存在**。
2. **lucide 回退兜底（防瞎画）**：槽位没命中 / 无合适候选 → 回落 `lucide-react`（`allowLucideFallback=true` 的槽位）。
3. **红线保护兜底（防翻车）**：14 个红线候选锁死 `usageScope=host-injected`，禁改色、禁乱用。
4. **不可回退槽位的"兜给人"机制**：`card-left-icon` / `run-status` / `feature-card` / `number-card`（lucide 难等价 / 渐变家族）无合适候选时标 `needs-design-confirmation`，**让设计确认，而非 AI 自画 inline SVG**。

> **修订补注（2026-06-22，决策 B）**：`number-card` 槽位由 `allowLucideFallback=true` 翻转为 `false`（`recommendedResourceType` 去 lucide → `custom-svg`），并入上述第 4 点「不可回退、兜给人」组。
> - **背景**：早期标「可回退 lucide」是因 NumberCard 内置仅 4 枚 inline 渐变图标、怕 AI 选不到合适项才留兜底（即第 2 点的 lucide 回退兜底）。
> - **新事实**：阶段 9 确定性派生后，`number-card` 槽位候选已扩至 **22 枚渐变图标**（OpenClawMonitor 抽出 `monitor-*` 并复用 `instance-total` 后槽位充足），内置 4 枚仅为其中已固化常用项、非上限。候选已足，扁平单色 lucide 反而破坏渐变家族。
> - **动作**：据实撤回兜底、对齐 `card-left-icon`——无合适候选时标 `needs-design-confirmation` 交设计补绘，不回退 lucide、不手搓 inline svg。
> - **合法性**：属建设计划 §5 待定项允许的修订（「是否允许 lucide 回退」以审计真实产出为准）。事实源 `component-resource-map.json` 已翻转，下游 `resource-skill-map.json` 经 `build:skill-map` 重生成、`check:skill-map` 第 ⑨ 项防漂移校验通过（22 候选全 `allowLucideFallback=false` / `recommendedResourceType=custom-svg`）。

---

## 5. check:skill-map 守门的 10 件事（防漂移）

候选 id 必须在 inventory ② status=normal ③ 不在 governance.removedIds ④ slot 在阶段 8 白名单 ⑤ file 候选磁盘存在 ⑥ usageScope 合法 ⑦ 红线候选 = host-injected + canonicalKey 非空、非红线 = current-project-only ⑧ 红线类目只能落对应红线槽位 ⑨ allowLucideFallback / recommendedResourceType 与阶段 8 规格一致 ⑩ summary 计数一致。任一不符 → 退出 1。

---

## 6. 关键不变量（改动前必读）

- `resource-skill-map.json` 是**生成产物，禁止手改**；要改口径改 `build-resource-skill-map.mjs` 后重跑。
- skill **只读 `resource-skill-map.json`**，不读页面、不读 inventory 全量、不臆造候选。
- 候选只来自 inventory 已审计字段（`componentSlot` / `canonicalKey`），不靠语义猜 slot。
- 日常加图标**通常不需要改 skill**（见接入 SOP）；只有「新增槽位类型」这种结构性变化才动 skill + `component-resource-map.json`。
- 红线资源永不自动归并 / 删除 / 改色。

---

## 7. 决策参与与口径

- 事实源决策（A 方案）由用户拍板；本文所有数据点（848 / 28 / 1、9 槽位 158 候选 14 红线、git 时间线）均经脚本与 git 实证，非记忆推断。
