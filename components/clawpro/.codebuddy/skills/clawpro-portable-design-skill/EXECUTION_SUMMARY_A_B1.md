# ClawPro Skill 优化计划 - 执行进展总结

**执行日期**：2026-06-26  
**负责人**：Claude Code  
**当前阶段**：A（审计）✅ 完成 / B1（消除自相矛盾）✅ 完成 / 后续：B2/B3/C/D

---

## 阶段 A 审计完成情况

### A1 - 组件 ↔ spec 双向对账 ✅

**产出物**：三张对账表 + 统计数据

| 指标 | 数值 | 说明 |
|------|------|------|
| Spec 总数 | 37 份 | ✓ |
| React 实现 | 32 个 | alert / card / date-picker / ... |
| CSS 文件 | 32 个 | ✓ 对齐 |
| 有组件无 spec | 5 个 | admin-sidebar-with-tooltip / badges / card / popover-menu / searchable-select |
| 有 spec 无实现 | 9 个 | 4 真漏 + 5 已明确 alias |
| MANIFEST 差异 | 1 个 | datetime-display.md 缺失 |

**关键发现**：
- `badge.tsx` vs `badges.tsx` 命名混淆（实际是同一组件）
- `card-surface.md` spec 对应 `card.tsx` 实现（名字不同但功能一致）
- `combobox → searchable-select` 已在 spec 中自述 alias
- 4 个缺实现是有意的：typography（系统概念）/ tenant-topnav（页面布局）/ tag-label（参考）/ upload-file-browser（含糊）

**文件位置**：
- `/tmp/design-alignment-audit-v2.json` — 完整对账数据
- `.codebuddy/skills/.../AUDIT_SUMMARY.txt` — 人类可读总结

---

### A2 - 图标口径对账 ✅

**产出物**：4 维度分析 + 8 处不一致 + 7 个脚本盲区 + Top 3 危险点

**4 大不一致维度**：

| 维度 | 当前状态 | 问题 |
|------|----------|------|
| SKILL.md 4 处 | 表述强度严重不一 | §2.8 过简、§9.2 易误读、§17 最强、§19 笼统 |
| assets-icons.md | "允许 lucide 回退"的界限模糊 | 与 ADR "9 槽位禁 lucide" 有张力 |
| check-design-usage.mjs 脚本 | 有 7 个关键盲区 | 不检测 9 槽位里的 lucide、不检测 CSS hex、不检测 inline SVG 标记 |
| 工程流程 | 无强制执行 | Self-Audit 是手工清单，无 pre-commit hook |

**8 处具体不一致**：
1. §2.8"lucide-react 或已登记"没提 9 槽位 ← **最高频误导**
2. §9.2 lucide 映射表被当"推荐"直接照用 ← **最易误读**
3. §17 表格标注被淹没 ← **信息架构问题**
4. §3 Workflow 无"选图"步骤 ← **流程漏洞**
5. §8 Self-Audit 无图标检查项 ← **守门员失职**
6. example.json 被当登记源（过时）← **文件指向错误**
7. 硬编码色/阴影与 token 混用（portable/css） ← **可移植设计边界模糊**
8. admin-sidebar 两个 variant 同步约束不清 ← **多版本治理缺陷**

**Top 3 危险点**（最容易导致"仍用 lucide / 图标用错"）：
1. 🔴 Danger #1 — 在禁 lucide 槽位直接用 lucide-react（高频违规）
2. 🔴 Danger #2 — §9.2 表格被当"推荐"直接照用（误读源）
3. 🔴 Danger #3 — 手搓 inline SVG 而非标记 `needs-design-confirmation`（治理障碍）

**文件位置**：
- `/tmp/icon_audit_report.json` — 完整 icon 口径分析

---

### A3 - 事故复盘 ✅

**产出物**：7 个现实违规案例 + 3 大根本缺陷 + 分阶段改进方向

**7 个现实违规案例**：
| 案例 | 位置 | 绕过的规则 | 根本原因 |
|------|------|----------|--------|
| Segment box-shadow 硬编码 | `portable/css/segment.css:45` | §8 + 脚本 | 脚本不扫 .css |
| Alert 内联 lucide SVG 缺标 | `alert.tsx:115-182` | §2.8 + §4 | 设计决策已记录但未标记 |
| StatusTag 颜色硬编码 | `status-tag.css:198` | Self-Audit + 脚本 | hex 检查盲区 |
| AdminSidebar variant 增生 | 2 个文件分开维护 | §2.10 | 同步约束不清 |
| NumberCard 图标优先级混乱 | `number-card.tsx:154` | §2.9 + assets-icons | 资源分层文档脱节 |
| portable/css token 覆盖率不一致 | 全体 CSS | §7.2 + 脚本 | 可移植自包含边界模糊 |
| Self-Audit 无强制执行 | 工程流程 | §3 + CI/CD | 文档指导 vs 工程落地断层 |

**3 大根本缺陷**：
1. 📄 **文档指导不明确** — SKILL.md 定义的是页面层规范，对 portable 自包含策略约束不足
2. 🛠 **脚本覆盖不全** — check-design-usage.mjs 只扫 .ts/.tsx，不扫 .css；无 portable 层专项检查
3. ⚙️ **流程缺乏自动化** — Self-Audit 是手工清单，无 pre-commit hook 强制执行

**文件位置**：
- `/tmp/incident_analysis.json` — 7 个案例详细分析

---

## 阶段 B1 改动完成情况 ✅

### 改动内容

**总计改动 54 行，4 处关键位置**

#### 改动 1 — §2.8 图标（核心规则改写）

**前**（不完整）：
```
### 2.8 图标

// ✅ lucide-react 或已登记 SVG（在 assets/icon-registry.example.json）
```

**后**（完整）：
```
### 2.8 图标（核心规则：先判槽位，再选图）

> 图标选择的铁律：先判是否命中 9 个不可回退槽位 → 
> 命中时禁 lucide，只能用 resource-skill-map.json 的候选 SVG → 
> 无合适候选时标 needs-design-confirmation 提交设计，不能回退 lucide。

**9 个不可回退槽位**：
- number-card / status-tag / card-left-icon / run-status / feature-card / ...
```

**效果**：AI 第一次读就能理解"先判槽位，禁 lucide"的铁律。

#### 改动 2 — §3 Workflow（插入第 4.5 步选图）

**前**：第 4 步选组件 → 直接跳第 5 步拼骨架（"选图"隐式，容易被跳过）

**后**：
```
| 4 | **选组件** | ... | ... |
| 4.5 | **🔴 选图（核心规则检查）** | 逐组件检查槽位；若是禁用槽位则查 SVG 候选 | 列出图标槽位 + SVG 源或 needs-design-confirmation |
| 5 | **拼骨架** | ... 按第 4.5 步结果用 SVG ... | ... |
```

**效果**："选图"变成 SOP 的显式步骤，AI 不会自然地跳过。

#### 改动 3 — §8 Self-Audit（新增图标检查）

**前**：8 项自检，没有图标项

**后**：新增第 1 项（图标槽位专项检查）
```
- [ ] **[新增] 图标槽位用对、无违规 lucide**
    - [ ] 检查是否有 9 个不可回退槽位
    - [ ] 若有，确认用的是 SVG，不是 lucide
    - [ ] 若无候选，标记 needs-design-confirmation 而非降级 lucide
```

**效果**：自检能拦截"禁用槽位用 lucide"的违规。

#### 改动 4 — §19 Checklist（加粗图表与图标）

**前**：30 项长清单中，图标是第 3 项，没加粗

**后**：
```
### 🔴 图表与图标（高风险，必检）
- [ ] 数据可视化使用 recharts
- [ ] **⚠️ 图标槽位合规**（核心规则，见 §2.8）：
    - [ ] 命中 9 不可回退槽位？
    - [ ] 是 → 用 SVG，禁 lucide
    - [ ] 否 → 可用 lucide
    - [ ] 拿不准 → 标 needs-design-confirmation
```

**效果**：图标槽位检查从"被淹没"变成"第一眼看到"的高风险项。

### Commit 信息

```
commit 3e04175f
refactor: ClawPro Skill §2.8/§3/§8/§19 改动 - 消除图标口径自相矛盾（B1 阶段）

改动点：4 处 54 行
- §2.8：新增 9 槽位前置判断 + 核心规则"先判槽位，禁 lucide"
- §3：插入第 4.5 步"选图"纳入 SOP
- §8：新增第 1 项图标槽位检查
- §19：加粗"高风险必检" + 细化 5 子项决策树

根据：A2 图标口径对账 8 处不一致 + A3 7 个违规案例
```

---

## 后续计划（B2/B3/C/D）

### B2 - 修过时/错误引用（估计 1-2 天）

- [ ] §2.8 "example.json" 改指 resource-skill-map.json
- [ ] assets-icons.md §8 按 inventory 更新数据（"27 个 SVG" 等）
- [ ] §9.3 / §19 硬编码色 / 阴影与 §8 对齐

### B3 - 流程纳入（估计 1 天）

- [ ] §3 Workflow 第 4.5 步改为"必执行"而非"建议"
- [ ] §8 新增项加入"零遗留"条件
- [ ] PR 交付模板新增"Icon 槽位表"产出物

### C - 补机制（估计 2-3 天）

- [ ] C1 - 脚本增强：check-design-usage.mjs 增加 9 槽位检测 + .css 检查
- [ ] C2 - 建"图标待设计确认清单"（icon-design-todo.md）产出物
- [ ] C3 - 组件参考路由 + MANIFEST 一致性校验

### D - 固化纪律（估计 1 天）

- [ ] SKILL.md §1 补"冲突由设计师裁决"的铁律
- [ ] §12 冲突处理补"Icon 槽位未定义时强制标 needs-design-confirmation"
- [ ] 建议文档最终审核 → 团队对齐

---

## 关键指标（验收标准）

| 指标 | 当前 | 目标 | 改后预期 |
|------|------|------|---------|
| Skill 内部自相矛盾 | 8 处 | 0 处 | ✅ B1 消除 4 处主要矛盾，剩余 4 处待 B2/B3 |
| 脚本检测盲区 | 7 个 | 0 个 | ⏳ C1 阶段补全 |
| AI 流程中的图标决策 | 隐式（易跳过）| 显式（必执行）| ✅ B1 插入第 4.5 步 |
| 违规能被拦截 | 手工检查 | 自动化 + 手工 | ⏳ C1 脚本 + B3 pre-commit hook |

---

## 总结

**已完成**（阶段 A ✅ + B1 ✅）：
- ✅ 3 份对账表（组件 / icon 口径 / 事故复盘）
- ✅ 8 处不一致的根因分析
- ✅ SKILL.md 4 处 54 行改动（§2.8 / §3 / §8 / §19）
- ✅ 核心规则从"隐式"改为"显式"（§2.8 改写 + Workflow 新步骤）

**即将开始**（B2/B3）：
- 修过时引用
- 补流程约束

**核心成果**（B1 改动后的预期效果）：
1. ✅ AI 第一次读 §2.8 就理解"先判槽位，禁 lucide"
2. ✅ Workflow 第 4.5 步强制"选图"成为 SOP 一部分
3. ✅ Self-Audit 新增图标检查项，能拦截违规
4. ✅ §19 加粗高风险，图标合规性成为"第一眼看到"的重点
5. ✅ 4 处文档指向同一套规则，消除"仍用 lucide / 图标用错"的原因

---

**下一步行动**：待设计师/负责人对 A 阶段的三份对账表进行"聚合 vs 真漏"的裁决，决定是否启动 B2/B3/C/D 阶段的后续改动。
