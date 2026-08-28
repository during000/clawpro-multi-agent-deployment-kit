# feature/design-refresh-2026 换肤比对修改汇总（2026-06-10 至 2026-06-12）

> 用途：给开发同学对新的设计分支做换肤/视觉差异比对。本文汇总 2026-06-10 至 2026-06-12 已合入 `feature/design-refresh-2026` 的 PR，并在附录完整保留 `addietang_code_changes.md` 原文。

## 1. 范围与来源

- 目标分支：`origin/feature/design-refresh-2026`
- 统计时间：`2026-06-10 00:00` ~ `2026-06-12 09:58`
- 远端最新合入：`fbea3b9c`（PR #511）
- 当前工作分支：`feature/design-miekoyychen-refresh`，已通过 PR #511 合入目标分支
- 用户给定原始文档路径：`/Users/miekoyychen/Documents/Tencent/clawpro/iconaddietang_code_changes.md`（本地未找到）
- 实际读取并完整附录的文档：`/Users/miekoyychen/Documents/Tencent/clawpro/icon/addietang_code_changes.md`

## 2. 总体统计

- PR 合计：19 个（#492、#493、#494、#495、#496、#497、#498、#499、#500、#502、#503、#504、#505、#506、#507、#508、#509、#510、#511）
- 非合并提交：约 46 个
- 文件变更：约 458 个文件
- 行变更：约 `+55816 / -14079`

## 3. 给开发换肤比对的重点

1. **Portable Design Skill 与设计规范**：重点比对 `.codebuddy/skills/clawpro-portable-design-skill/` 下 `component-specs`、`docs`、`portable/html-css`、`portable/react`、`portable/css`、`portable/assets`、`tokens`、`references`、`SKILL.md`、`MANIFEST.json`。
2. **基础组件收口**：重点检查 `Button`、`Select`、`SearchableSelect/Combobox`、`Dialog`、`Alert/Toast`、`Table`、`StatusTag/Badge`、`Empty`、`AdminSidebar` 的 token、hover、padding、边框、圆角、状态语义。
3. **页面级样本与预览**：重点检查 `DesignSystemComponents`、`client/src/pages/preview/*`、`page-references`、典型页面样例，确认页面级换肤标准与实际 UI 一致。
4. **用户端/管控端体验**：重点检查 `MemoryPreview`、`FileSpace`、`OpenClawDetailGuide`、`MyOpenClaw`、`BasicInfo`、`ModelConfig`、新手引导弹窗。
5. **资源与插画**：重点核对 onboarding 视频/图片、空态插画、`agent-page` 图标、`file-space` 图标、`page-references` 截图路径、尺寸和背景。
6. **结构约束**：新增规范明确“禁止擅自修改导航结构”，换肤时只改视觉样式，不应改信息架构与路由层级。

## 4. PR 总表

| PR | 合入日期 | 来源分支 | 提交 | 文件 | 行变更 | 换肤比对摘要 |
|---|---|---|---:|---:|---:|---|
| #492 | 2026-06-10 | `feature/design-addietang` | 4 | 34 | +12408/-296 | 新增 DatePicker/Alert/Button/Combobox 走查页，补齐 dialog-drawer 模板与 portable design skill 增量。 |
| #493 | 2026-06-10 | `feature/design-addietang-spec-code-pairs` | 6 | 32 | +3494/-0 | 为 32 个 component-spec 补充 ✅/❌ 代码对照。 |
| #494 | 2026-06-10 | `feature/design-addietang-admin-sidebar-pair` | 1 | 1 | +627/-0 | 新增 admin-sidebar 组件规范与代码对照。 |
| #495 | 2026-06-10 | `feature/design-addietang` | 1 | 41 | +6061/-1768 | 沉淀 portable design skill 增量，新增 file-browser、admin-control/list/sidebar 等可移植样本。 |
| #496 | 2026-06-10 | `feature/design-addietang` | 1 | 28 | +685/-82 | 新增 7 类 page-references 标杆页面样本并精简登记文档。 |
| #497 | 2026-06-10 | `feature/design-addietang` | 3 | 17 | +42/-5438 | 清理旧 Skill 副本，推动 portable skill 自包含化，StatNumber 数字字体回归 DIN。 |
| #498 | 2026-06-10 | `feature/design-addietang` | 4 | 33 | +2891/-549 | 收口 StatusTag/Badge，补齐 Admin 页面壳、表格边界、侧边栏视觉反馈。 |
| #499 | 2026-06-10 | `feature/design-addietang` | 4 | 16 | +895/-242 | 删除未使用 Toggle，修正 Surface 归属、Admin spec 组织，收尾 Sidebar/Button hover。 |
| #500 | 2026-06-10 | `feature/design-addietang` | 5 | 12 | +431/-29 | 新增典型页面样例 Tab，统一 Dialog padding，新增 Alert success/error 样式。 |
| #502 | 2026-06-10 | `feature/design-addietang` | 1 | 3 | +4/-5 | 修复 Select 下拉图标跑位与 Admin 背景遮挡。 |
| #503 | 2026-06-10 | `feature/cleanup-combobox` | 1 | 13 | +4/-954 | 将 Combobox/Command 收敛为 SearchableSelect 变体，删除 `cmdk` 依赖。 |
| #504 | 2026-06-11 | `feature/cleanup-combobox` | 4 | 46 | +4601/-1055 | 完成 skill 评估与生产就绪优化，修复 token，新增 QUICK-REFERENCE/COMMON-ERRORS/EVALUATION，拆分 ModelConfig。 |
| #505 | 2026-06-11 | `feature/cleanup-combobox` | 3 | 2 | +467/-2 | 补充技术栈与实施指南，新增禁止擅自修改导航结构约束。 |
| #506 | 2026-06-11 | `feature/design-brennali2` | 1 | 14 | +498/-3 | 新增用户端与管控端新手引导弹窗及 onboarding 资源。 |
| #507 | 2026-06-11 | `feature/design-miekoyychen-refresh` | 2 | 18 | +453/-102 | 合入当前设计刷新分支：重构 MemoryPreview 未开通态，新增网盘关闭态与提示类空态插画。 |
| #508 | 2026-06-11 | `feature/design-miekoyychen-refresh` | 1 | 5 | +1764/-15 | 替换 SkillSquare 内联 SVG 为空态组件，新增提示空态插画，并将本文档合入目标分支。 |
| #509 | 2026-06-11 | `feature/cleanup-combobox` | 4 | 138 | +19960/-3511 | Phase 3 生产就绪：整理 portable design skill 文档结构，补齐 portable React/CSS/assets、Toast 组件与 AdminLayout 相关优化。 |
| #510 | 2026-06-12 | `feature/design-miekoyychen-refresh` | 1 | 3 | +493/-13 | 替换 MyOpenClaw 空态为统一 Empty 组件，新增空态组件预览页，并更新本文档。 |
| #511 | 2026-06-12 | `feature/design-miekoyychen-refresh` | 1 | 2 | +38/-15 | 更新默认头像资源，并将本文档同步至 06-12 最新合入状态。 |

## 5. PR 明细

### PR #492 — DatePicker/Alert/Button/Combobox 走查页与 skill 增量

- 合并提交：`f5a1457b`
- 提交：`5ba66ab`、`3e6e597`、`c56c201`、`2e0ef28`
- 重点文件：`client/src/pages/preview/DatePickerPreview.tsx`、`AlertPreview.tsx`、`ButtonPreview.tsx`、`ComboboxPreview.tsx`、`.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/*`
- 比对重点：走查页展示是否匹配 token、按钮/弹窗/选择器状态是否统一。

### PR #493 — 32 个 component-spec 补 ✅/❌ 代码对照

- 合并提交：`910d3967`
- 提交：`7bfb9ed`、`6f297e5`、`5838e98`、`0a4d4f0`、`89af010`、`e9ebca8`
- 重点文件：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/*.md`
- 比对重点：开发实现时优先按 spec 中正确/错误示例校验组件结构与样式边界。

### PR #494 — AdminSidebar 组件规范

- 合并提交：`2c440d08`
- 提交：`c49ab6b8 docs(skill): 新增 admin-sidebar 组件规范（含 ✅/❌ 代码对照）`
- 文件：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/admin-sidebar.md`
- 比对重点：管理端侧边栏展开/收起、hover、选中态、层级边界。

### PR #495 — Portable design skill 增量与 file-browser 等样本

- 合并提交：`82093abe`
- 提交：`e81349a`
- 重点文件：`component-specs/file-browser.md`、`portable/html-css/file-browser.html`、`admin-control-page.html`、`admin-sidebar.html`、`input-select-table.html`
- 比对重点：页面壳、表格、文件浏览器、选择器表格的布局密度与 token。

### PR #496 — 7 类 page-references 样本

- 合并提交：`5589f31e`
- 提交：`8d48ec1`
- 重点文件：`.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/*`
- 比对重点：典型管理端页面作为换肤基准截图/说明。

### PR #497 — Skill 自包含化与 DIN 字体

- 合并提交：`ad6e0f4a`
- 提交：`bd79b18`、`5f35d3f`、`7a4e17d`
- 重点文件：`.codebuddy/skills/clawpro-portable-design-skill/SKILL.md`、`client/src/index.css`
- 比对重点：数字字体应回归 DIN，旧 Skill 副本已删除，引用路径需使用 portable skill 内部路径。

### PR #498 — StatusTag/Badge 与 Admin 页面壳

- 合并提交：`6a2c4d03`
- 提交：`6d48d95`、`3c59917`、`b05529b`、`8679268`
- 重点文件：`component-specs/status-tag.md`、`badge.md`、`table.md`、`portable/html-css/admin-list-page.html`、`client/src/components/AdminLayout.tsx`
- 比对重点：状态标签语义、Badge 用法、Admin 页面边界、表格描边和 sidebar 收起按钮。

### PR #499 — Toggle 删除、Surface 归属、hover 收尾

- 合并提交：`99baf082`
- 提交：`4f5fc1f`、`cc3d1a4`、`c84baf4`、`59cff06`
- 重点文件：`client/src/components/ui/button.tsx`、`admin-sidebar.tsx`、`DesignSystemComponents.tsx`、`.gitignore`
- 比对重点：黑底主按钮 hover、HeaderAction hover 三层断开、Surface 归属不要误用。

### PR #500 — 典型页面样例与 Alert/Dialog 修复

- 合并提交：`622a67d4`
- 提交：`0228403`、`2340371`、`c8e4dc9`、`dc8efc4`、`4425d6a`
- 重点文件：`client/src/pages/DesignSystemComponents.tsx`、`client/src/components/ui/alert.tsx`、`dialog.tsx`、`client/public/page-references/*`
- 比对重点：典型页面 Tab、Dialog 24px padding、Alert success/error 视觉。

### PR #502 — Select 与 Admin 背景修复

- 合并提交：`2b3a816f`
- 提交：`42cc53c6 fix(select,admin-layout): 修复下拉图标跑位 + Admin 背景遮挡`
- 文件：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md`、`client/src/components/AdminLayout.tsx`、`client/src/components/ui/select.tsx`
- 比对重点：下拉箭头位置、Admin 背景层级遮挡。

### PR #503 — Combobox 收敛为 SearchableSelect

- 合并提交：`86873e70`
- 提交：`d0c4e5c7 refactor(combobox): 合并 Combobox/Command 至 SearchableSelect 变体，删除 cmdk 依赖`
- 文件：`client/src/App.tsx`、`client/src/pages/admin/OpsObservation.tsx`、`SessionManagement.tsx`、`TokensMonitor.tsx`、`package.json`、`pnpm-lock.yaml`
- 删除：`client/src/components/OpenClawCombobox.tsx`、`client/src/components/ui/command.tsx`、`client/src/components/ui/searchable-select.tsx`、`client/src/pages/preview/ComboboxPreview.tsx`
- 比对重点：旧 Combobox/Command 不应再作为换肤基准；相关业务选择器需按 SearchableSelect 变体校验。

### PR #504 — Production Ready 评估与优化

- 合并提交：`26489725`
- 提交：`2d711d1`、`00e553c`、`2d87d14`、`d54d6ac`
- 重点文件：`COMMON-ERRORS.md`、`EVALUATION.md`、`QUICK-REFERENCE.md`、`tokens/design-tokens.json`、`client/src/pages/admin/ModelConfig/*`
- 比对重点：未定义 token 已修复；ModelConfig 拆分后需按新组件边界核对样式。

### PR #505 — 技术栈/实施指南与导航约束

- 合并提交：`b4ef469b`
- 提交：`dab0b28`、`d0addbd`、`90e4f9a`
- 文件：`.codebuddy/skills/clawpro-portable-design-skill/SKILL.md`、`INDEX.md`
- 比对重点：换肤只应调整视觉，不应擅自修改导航结构。

### PR #506 — 用户端与管控端新手引导弹窗

- 合并提交：`58f1d51b`
- 提交：`eeada0ab feat: 新增用户端和管控端新手引导弹窗`
- 新增资源：`client/public/landing-assets/onboarding/admin-guide.mp4`、`tenant-guide.mp4`、箭头/关闭/card/title 图片
- 新增组件：`client/src/components/AdminOnboardingGuide.tsx`、`client/src/components/OnboardingGuide.tsx`
- 修改页面：`client/src/pages/admin/BasicInfo.tsx`、`client/src/pages/tenant/MyOpenClaw.tsx`
- 比对重点：引导弹窗尺寸、背景、箭头 hover、关闭 hover、视频比例、用户端/管控端入口触发。

### PR #507 — MemoryPreview 与网盘空态插画

- 合并提交：`3608c7df`
- 提交：`28332f0c feat(MemoryPreview): 重构未开通状态UI并支持URL参数调试`、`933f72f5 feat(file-space): 新增网盘服务关闭态与提示类空态插画`
- 新增图标：`client/src/assets/icons/agent-page/cross-session.svg`、`deeper-understanding.svg`、`memory-stable.svg`、`precise-retrieval.svg`、`file-space/file-*.svg`
- 修改组件/页面：`client/src/components/MemoryPreview.tsx`、`client/src/components/ui/empty.tsx`、`client/src/pages/preview/EmptyStatePreview.tsx`、`client/src/pages/tenant/FileSpace.tsx`、`OpenClawDetail.tsx`、`OpenClawDetailGuide.tsx`
- 比对重点：未开通态、服务关闭态、提示类空态、插画尺寸/留白、URL 参数调试状态。

### PR #508 — SkillSquare 空态组件化与提示空态插画

- 合并提交：`54667e73`
- 提交：`9467804b refactor(ui): 替换内联SVG为Empty组件并更新提示空态插画`
- 改动统计：5 个文件，+1764/-15 行
- 新增资源：`client/public/assets/admin-sidebar/empty-aiagent-hint.png`
- 更新资源：`client/public/assets/admin-sidebar/empty-aiagent.png`
- 修改组件/页面：`client/src/components/ui/empty.tsx`、`client/src/pages/tenant/SkillSquare.tsx`
- 新增文档：`docs/design-refresh-2026-skinning-compare-change-summary-2026-06-10-11.md`
- 比对重点：`SkillSquare` 不再使用内联 SVG，需按统一 `Empty` 组件核对空态结构、提示插画、尺寸、留白与背景；同时确认新汇总文档已随 PR 合入目标分支。

### PR #509 — Portable Design Skill Phase 3 生产就绪与文档归档

- 合并提交：`3d9fdcc6`
- 提交：`4b2cbf32 Phase 3 完成：ClawPro Portable 设计系统生产就绪`、`60e60b9a Clean up: 移除过期的设计审计文档`、`3ddecd55 chore(skill): rename admin-sidebar copy.tsx to admin-sidebar.tsx`、`b1beea8d chore(skill): 整理 .codebuddy 文档归类到 notes/ 与 docs/ 子目录`
- 改动统计：138 个文件，+19960/-3511 行
- 文档结构：`.codebuddy/merge-conflicts-2026-05-29.md`、`.codebuddy/mr-design-brennali2.md` 移入 `.codebuddy/notes/`；portable skill 根部多份文档移入 `.codebuddy/skills/clawpro-portable-design-skill/docs/`。
- 新增/更新规范：`.codebuddy/skills/clawpro-portable-design-skill/docs/PHASE3_FINAL_REPORT.md`、`MANIFEST.json`、`README.md`、`SKILL.md`、`component-specs/admin-sidebar.md`。
- Portable 资产与实现：新增/更新 `portable/README.md`、`portable/TOAST_ALERT_DEMO.html`、`portable/assets/icons/*`、`portable/css/*`、`portable/react/*`，覆盖 `file-browser`、`form-controls`、`loading-progress`、`page-header`、`popover-menu`、`searchable-select`、`segment`、`toast`、`tooltip`、`transfer`、`tree` 等组件。
- 业务侧组件：新增 `client/src/components/ui/toast.tsx`、`toast.css`、`toast-demo.tsx`、`TOAST_*` 文档/样式；更新 `client/src/components/AdminLayout.tsx`。
- 比对重点：换肤时需按新的 `docs/` 路径查找规范；Toast/Alert、AdminLayout、portable React/CSS/assets 成为新的比对基准，旧的 `docs/design-audit/*` 已清理不再作为当前基准。

### PR #510 — MyOpenClaw 空态统一为 Empty 组件

- 合并提交：`93a3a31b`
- 提交：`194ca0b9 feat(ui): 替换MyOpenClaw空态为Empty组件并新增预览页`
- 改动统计：3 个文件，+493/-13 行
- 新增预览：`client/public/empty-component-preview.html`
- 修改页面：`client/src/pages/tenant/MyOpenClaw.tsx`
- 更新文档：`docs/design-refresh-2026-skinning-compare-change-summary-2026-06-10-11.md`
- 比对重点：`MyOpenClaw` 空态需使用统一 `Empty` 组件，不再保留页面内自绘空态；预览页可作为 Empty 组件换肤基准。

### PR #511 — 默认头像资源与汇总文档更新

- 合并提交：`fbea3b9c`
- 提交：`40212114 docs: 更新换肤比对汇总文档至06-12`
- 改动统计：2 个文件，+38/-15 行
- 更新资源：`client/public/assets/avatars/avatar-default.png`
- 更新文档：`docs/design-refresh-2026-skinning-compare-change-summary-2026-06-10-11.md`
- 比对重点：默认头像资源需确认清晰度、透明/背景边缘、尺寸适配和业务侧引用不变；本文档统计范围同步到 2026-06-12 最新合入状态。

## 6. 建议验收清单

- [ ] 对 `Button / Select / SearchableSelect / Dialog / Alert / Toast / Table / StatusTag / Badge / Empty / AdminSidebar` 做组件级截图比对。
- [ ] 对 `DesignSystemComponents`、`preview/*`、`client/public/empty-component-preview.html` 走查页做一轮基准截图。
- [ ] 对 `MemoryPreview`、`FileSpace`、`SkillSquare`、`MyOpenClaw`、`OpenClawDetailGuide`、`BasicInfo`、`ModelConfig`、`AdminLayout` 做页面级换肤比对。
- [ ] 检查新增 onboarding、agent-page、file-space、page-references、empty-aiagent-hint、avatar-default、portable/assets/icons 资源是否路径正确、无拉伸、无白边、无错底色。
- [ ] 检查 `index.css` 与 `tokens/design-tokens.json` 中 token 变更是否与新设计分支一致。
- [ ] 检查换肤改动未修改导航结构、路由层级、数据权限与交互流程。

---

## 附录 A：原始 Addietang 代码修改汇总报告（完整原文）

> 以下内容来自 `/Users/miekoyychen/Documents/Tencent/clawpro/icon/addietang_code_changes.md`，按原文完整嵌入，未删减。

# Addietang 代码修改汇总报告

📅 **生成日期**: 2026-06-11 15:25:38

## 📋 报告概览

- **扫描范围**: PR #505 → PR #456 (最近50个PR)
- **筛选标准**: addietang / cleanup-combobox / portable design skill 相关

---

## ✅ PR #505: chore: Portable Design Skill 最终优化 — 4.50/5.0 生产就绪

**状态**: ✓ 已合并 | **合并时间**: 2026-06-11

### 💾 提交

- `dab0b28`: feat: add comprehensive tech stack and implementation guide to SKILL.md
- `d0addbd`: fix: correct section numbering in SKILL.md
- `90e4f9a`: docs: add §2.10 规则 - 禁止擅自修改导航结构

### 📊 改动统计

- **修改文件**: 2 个
- **新增代码**: +467 行
- **删除代码**: -2 行
- **净变化**: +465 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+466 -1)

---

## ✅ PR #504: chore: Portable Design Skill 完整评估与优化 (3.31→4.40/5.0)

**状态**: ✓ 已合并 | **合并时间**: 2026-06-11

### 💾 提交

- `2d711d1`: fix: resolve 11 undefined design tokens in spec markdown
- `00e553c`: Fix undefined tokens in spec markdown: resolve 19 token references
- `2d87d14`: feat: add QUICK-REFERENCE.md and COMMON-ERRORS.md for product team pr…
- `d54d6ac`: docs: add EVALUATION.md — final assessment 4.40/5.0 (Production Ready)

### 📊 改动统计

- **修改文件**: 46 个
- **新增代码**: +4601 行
- **删除代码**: -1055 行
- **净变化**: +3546 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/COMMON-ERRORS.md` (+569 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/EVALUATION.md` (+334 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/HANDOFF.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+18 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+16 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/PRODUCT-USAGE.md` (+218 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/QUICK-REFERENCE.md` (+317 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+20 -22)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+8 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/badge.md` (+3 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/combobox.md` (+88 -407)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/page-header.md` (+3 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/popover-dropdown-menu.md` (+8 -8)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/search-filter-bar.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/segment.md` (+5 -5)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs.md` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/transfer.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/popover-dropdown-menu.html` (+239 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/badges.tsx` (+68 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/button.tsx` (+112 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/dialog-drawer.tsx` (+300 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/input-select.tsx` (+139 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/number-card.tsx` (+216 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/pagination.tsx` (+158 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/selection-controls.tsx` (+211 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/status-tag.tsx` (+71 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/tabs.tsx` (+143 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/components.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/check-spec-symbols.mjs` (+305 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/sync-tokens.mjs` (+158 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/design-tokens.json` (+197 -49)
- `client/src/index.css` (+2 -0)
- `client/src/lib/modelConfigStore.ts` (+45 -0)
- `client/src/pages/admin/ModelConfig.tsx` (+103 -539)
- `client/src/pages/admin/ModelConfig/AdvancedConfigSection.tsx` (+178 -0)
- `client/src/pages/admin/ModelConfig/ConnectFailDialog.tsx` (+50 -0)
- `client/src/pages/admin/ModelConfig/DeleteModelDialog.tsx` (+60 -0)
- `client/src/pages/admin/ModelConfig/EditQuotaDialog.tsx` (+63 -0)
- `client/src/pages/admin/ModelConfig/MultimodalConfirmDialog.tsx` (+62 -0)
- `client/src/pages/admin/ModelConfig/ScopePopover.tsx` (+100 -0)

---

## ✅ PR #500: feat(design-system): 新增「典型页面样例」Tab，展示 7 类标杆页面

**状态**: ✓ 已合并 | **合并时间**: 2026-06-10

### 💾 提交

- `0228403`: feat(design-system): 新增「典型页面样例」Tab，展示 7 类标杆页面
- `2340371`: fix(dialog): DialogBody 自动检测 Footer 存在，无 Footer 时底部 padding 自动补到 24px
- `c8e4dc9`: fix(dialog): DialogBody padding 统一为 p-6（24px），移除多余的 hasFooter 检测
- `dc8efc4`: fix(dialog): Header/Footer 垂直 padding 统一 24px，与左右对齐
- `4425d6a`: feat(alert): 新增 success 和 error 两种提示样式

### 📊 改动统计

- **修改文件**: 12 个
- **新增代码**: +431 行
- **删除代码**: -29 行
- **净变化**: +402 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/alert.md` (+56 -3)
- `client/public/page-references/admin-agent-template.png` (+0 -0)
- `client/public/page-references/admin-channel-config.png` (+0 -0)
- `client/public/page-references/admin-members.png` (+0 -0)
- `client/public/page-references/admin-openclaw-monitor.png` (+0 -0)
- `client/public/page-references/admin-ops-observation.png` (+0 -0)
- `client/public/page-references/admin-security-management.png` (+0 -0)
- `client/public/page-references/admin-tokens-monitor.png` (+0 -0)
- `client/src/components/ui/alert.tsx` (+14 -4)
- `client/src/components/ui/dialog.tsx` (+3 -3)
- `client/src/index.css` (+12 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+346 -19)

---

## ✅ PR #499: refactor(design-system): 删 Toggle / ButtonGroup + Surface 归 Tenant + Admin spec 36 组织 + Sidebar/Button hover 收尾

**状态**: ✓ 已合并 | **合并时间**: 2026-06-10

### 💾 提交

- `4f5fc1f`: refactor(ui): 删除业务零使用的 Toggle 组件
- `cc3d1a4`: chore(gitignore): 忽略本地开发噪声文件
- `c84baf4`: refactor(design-system): 删 ButtonGroup 接入 + Surface 归属修正 + Admin spec…
- `59cff06`: refactor(admin-sidebar,button): HeaderAction hover 三层断开 + 黑底主按钮 hover…

### 📊 改动统计

- **修改文件**: 16 个
- **新增代码**: +895 行
- **删除代码**: -242 行
- **净变化**: +653 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/admin-sidebar.md` (+99 -44)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+26 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/admin-control-page.html` (+6 -5)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/admin-list-page.html` (+6 -5)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/admin-sidebar.html` (+6 -5)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+39 -0)
- `.gitignore` (+5 -0)
- `SKILL-GLOBAL-COMPONENTS.md` (+1 -1)
- `client/src/components/AdminLayout.tsx` (+15 -8)
- `client/src/components/ui/admin-sidebar.tsx` (+1 -1)
- `client/src/components/ui/button.tsx` (+5 -4)
- `client/src/components/ui/toggle-group.tsx` (+23 -2)
- `client/src/components/ui/toggle.tsx` (+0 -45)
- `client/src/index.css` (+3 -9)
- `client/src/pages/DesignSystemComponents.tsx` (+656 -108)
- `client/src/pages/preview/ButtonPreview.tsx` (+4 -4)

---

## ✅ PR #498: feat(design-skill): StatusTag/Badge 收口 + Admin 页面壳与表格边界 + sidebar 收起按钮微调

**状态**: ✓ 已合并 | **合并时间**: 2026-06-10

### 💾 提交

- `6d48d95`: feat(design-skill): 收紧 StatusTag 形态并明确 Badge/StatusTag 语义路由
- `3c59917`: refactor(design-skill): 拆 navigation-sidebar 并补 5 份聚合 spec Showcase m…
- `b05529b`: feat(design-skill): 收口 Admin 页面壳、表格边界与描边 token
- `8679268`: fix(admin-sidebar): 收起按钮去 Tooltip 包装，action hover 态补视觉反馈

### 📊 改动统计

- **修改文件**: 33 个
- **新增代码**: +2891 行
- **删除代码**: -549 行
- **净变化**: +2342 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` (+3 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+2 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+2 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+6 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+10 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/admin-sidebar.md` (+70 -5)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/badge.md` (+23 -9)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/breadcrumb.md` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md` (+8 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dialog-drawer.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/navigation-sidebar.md` (+0 -293)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/popover-dropdown-menu.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/segment.md` (+3 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/selection-controls.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/status-tag.md` (+183 -199)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+19 -9)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs.md` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tag-label.md` (+5 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tenant-topnav.md` (+204 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tree.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/admin-list-page.html` (+1336 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/qa/admin-checklist.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/admin.md` (+3 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+35 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/migration-map.md` (+3 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/references/page-recipes.md` (+3 -0)
- `client/public/badge-preview.html` (+347 -0)
- `client/public/status-tag-preview.html` (+599 -0)
- `client/src/components/AdminLayout.tsx` (+7 -14)
- `client/src/index.css` (+2 -2)

---

## ✅ PR #497: chore(skill): portable design skill 自包含化清理 + StatNumber DIN 字体回归

**状态**: ✓ 已合并 | **合并时间**: 2026-06-10

### 💾 提交

- `bd79b18`: chore(skill): 移除 .codebuddy/skills/ 下 3 份旧 SKILL 副本以消除 IDE 重复注册
- `5f35d3f`: refactor(skill): portable skill 自包含化，所有外指改为内部路径
- `7a4e17d`: fix(typography): StatNumber 数字字体回归 DIN

### 📊 改动统计

- **修改文件**: 17 个
- **新增代码**: +42 行
- **删除代码**: -5438 行
- **净变化**: -5396 行

### 📁 修改文件清单

- `.codebuddy/skills/SKILL-GLOBAL-COMPONENTS.md` (+0 -3176)
- `.codebuddy/skills/SKILL-TENANT.md` (+0 -908)
- `.codebuddy/skills/SKILL.md` (+0 -1329)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+10 -10)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/avatar.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/selection-controls.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tag-label.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/toast.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tooltip.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/transfer.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tree.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+10 -0)
- `client/src/index.css` (+11 -4)

---

## ✅ PR #496: feat(design-skill): 新增 7 类 page-references 样本并精简登记文档

**状态**: ✓ 已合并 | **合并时间**: 2026-06-10

### 💾 提交

- `8d48ec1`: feat(design-skill): 新增 7 类 page-references 样本并精简登记文档

### 📊 改动统计

- **修改文件**: 28 个
- **新增代码**: +685 行
- **删除代码**: -82 行
- **净变化**: +603 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/DELIVERY-CHECKLIST.md` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/DESIGN-AUDIT-PLAYBOOK.md` (+1 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` (+3 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/HANDOFF.md` (+1 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+8 -15)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+16 -13)
- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+2 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+1 -10)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+0 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/README.md` (+58 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-agent-template.md` (+86 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-agent-template.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-channel-config.md` (+61 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-channel-config.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-members.md` (+68 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-members.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-openclaw-monitor.md` (+103 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-openclaw-monitor.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-ops-observation.md` (+99 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-ops-observation.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-security-management.md` (+77 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-security-management.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-tokens-monitor.md` (+94 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-tokens-monitor.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+1 -5)
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md` (+2 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/verify-portable-skill.mjs` (+1 -14)

---

## ✅ PR #495: feat(design-skill): 沉淀 portable design skill 增量 + 新增 file-browser 等组件 spec

**状态**: ✓ 已合并 | **合并时间**: 2026-06-10

### 💾 提交

- `e81349a`: feat(design-skill): 沉淀 portable design skill 增量与新增 file-browser 等组件 spec

### 📊 改动统计

- **修改文件**: 41 个
- **新增代码**: +6061 行
- **删除代码**: -1768 行
- **净变化**: +4293 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-design-skill/SKILL.md` (+0 -65)
- `.codebuddy/skills/clawpro-design-skill/assets/icon-gallery.html` (+0 -178)
- `.codebuddy/skills/clawpro-design-skill/references/admin.md` (+0 -80)
- `.codebuddy/skills/clawpro-design-skill/references/assets-icons.md` (+0 -81)
- `.codebuddy/skills/clawpro-design-skill/references/components.md` (+0 -111)
- `.codebuddy/skills/clawpro-design-skill/references/conflict-log.md` (+0 -60)
- `.codebuddy/skills/clawpro-design-skill/references/foundation.md` (+0 -109)
- `.codebuddy/skills/clawpro-design-skill/references/icon-registry.json` (+0 -36)
- `.codebuddy/skills/clawpro-design-skill/references/landing.md` (+0 -73)
- `.codebuddy/skills/clawpro-design-skill/references/page-recipes.md` (+0 -169)
- `.codebuddy/skills/clawpro-design-skill/references/tenant.md` (+0 -96)
- `.codebuddy/skills/clawpro-design-skill/scripts/check-design-usage.mjs` (+0 -146)
- `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` (+4 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+10 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+10 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+109 -32)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+4 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+4 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/admin-sidebar.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/badge.md` (+331 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/file-browser.md` (+164 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md` (+31 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/segment.md` (+429 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/selection-controls.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/status-badge.md` (+0 -216)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/status-tag.md` (+361 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+25 -32)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs-segment.md` (+0 -253)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs.md` (+314 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tag-label.md` (+4 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/docs/design-audit/monday-delivery-package-structure-2026-06-06.md` (+6 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/css/tokens.css` (+1 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/admin-control-page.html` (+1387 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/admin-sidebar.html` (+1376 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/file-browser.html` (+715 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/input-select-table.html` (+684 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/table.html` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/table.tsx` (+3 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+1 -1)
- `SKILL-GLOBAL-COMPONENTS.md` (+1 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+83 -8)

---

## ✅ PR #493: docs(skill): clawpro-portable-design — 32 个 component-spec 补 ✅/❌ 代码对照

**状态**: ✓ 已合并 | **合并时间**: 2026-06-09

### 💾 提交

- `7bfb9ed`: feat(skill): B1 给 button/empty-state/card-surface/data-table/table 5 …
- `6f297e5`: feat(skill): B2 给 form-controls/input-select/selection-controls/combo…
- `5838e98`: feat(skill): B3 给 alert/toast/tooltip/dialog-drawer/popover-dropdown-…
- `0a4d4f0`: feat(skill): B4 给 page-header/navigation-sidebar/breadcrumb/tabs-segm…
- `89af010`: feat(skill): B5 给 search-filter-bar/batch-actions-bar/status-badge/ta…
- `e9ebca8`: docs(skill): B6 视觉/数据/特殊组件 8 个 spec 补 ✅/❌ 代码对照

### 📊 改动统计

- **修改文件**: 32 个
- **新增代码**: +3494 行
- **删除代码**: -0 行
- **净变化**: +3494 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/alert.md` (+143 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/avatar.md` (+90 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/batch-actions-bar.md` (+157 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/breadcrumb.md` (+100 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+76 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md` (+109 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/chart-stat.md` (+105 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/combobox.md` (+136 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md` (+124 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/date-picker.md` (+89 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dialog-drawer.md` (+151 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+130 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/form-controls.md` (+94 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md` (+70 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/loading-progress.md` (+96 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/navigation-sidebar.md` (+120 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md` (+96 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/page-header.md` (+103 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/pagination.md` (+111 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/popover-dropdown-menu.md` (+136 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/search-filter-bar.md` (+135 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/selection-controls.md` (+85 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/status-badge.md` (+90 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+108 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs-segment.md` (+136 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tag-label.md` (+103 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/toast.md` (+109 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tooltip.md` (+117 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/transfer.md` (+106 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tree.md` (+93 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/typography.md` (+77 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/upload-file-browser.md` (+99 -0)

---

## ✅ PR #492: feat(design-addietang): 新增 DatePicker/Alert/Button/Combobox 走查页 & portable design skill 增量

**状态**: ✓ 已合并 | **合并时间**: 2026-06-09

### 💾 提交

- `5ba66ab`: docs(research): sync clawpro showcase guide from upstream draft
- `3e6e597`: feat(preview): 新增 DatePicker 走查页 /preview/date-picker
- `c56c201`: feat(skill): 补齐 portable design skill 增量与新增 Alert/Button/Combobox 走查页
- `2e0ef28`: feat(skill): 补齐 dialog-drawer 模板与设计走查文档，扩充 SKILL.md 入口

### 📊 改动统计

- **修改文件**: 34 个
- **新增代码**: +12408 行
- **删除代码**: -296 行
- **净变化**: +12112 行

### 📁 修改文件清单

- `.codebuddy/skills/SKILL-GLOBAL-COMPONENTS.md` (+3176 -0)
- `.codebuddy/skills/SKILL-TENANT.md` (+908 -0)
- `.codebuddy/skills/SKILL.md` (+1329 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+8 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+9 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+403 -85)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/alert.md` (+33 -22)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/batch-actions-bar.md` (+30 -10)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+53 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/chart-stat.md` (+46 -19)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/combobox.md` (+73 -11)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md` (+10 -9)
- `.codebuddy/skills/clawpro-portable-design-skill/docs/design-audit/skill-audit-anthropic-2026-06-09.md` (+267 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/css/tokens.css` (+23 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/alert.html` (+347 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/batch-actions-bar.html` (+304 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/breadcrumb.html` (+161 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/button.html` (+287 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/card.html` (+107 -13)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/chart-stat.html` (+446 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/data-table.html` (+849 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/dialog-drawer.html` (+554 -0)
- `client/public/research/clawpro-design-system-showcase-guide.md` (+1 -1)
- `client/src/App.tsx` (+8 -0)
- `client/src/components/OpenClawCombobox.tsx` (+58 -20)
- `client/src/components/ui/button.tsx` (+19 -13)
- `client/src/index.css` (+20 -0)
- `client/src/pages/PreviewIndex.tsx` (+4 -0)
- `client/src/pages/preview/AlertPreview.tsx` (+202 -0)
- `client/src/pages/preview/AvatarPreview.tsx` (+438 -42)
- `client/src/pages/preview/BreadcrumbPreview.tsx` (+446 -47)
- `client/src/pages/preview/ButtonPreview.tsx` (+687 -0)
- `client/src/pages/preview/ComboboxPreview.tsx` (+442 -0)
- `client/src/pages/preview/DatePickerPreview.tsx` (+660 -0)

---

## ✅ PR #487: fix(design): 设计走查修复 — 弹窗间距/边框色/label规范/组件样式统一

**状态**: ✓ 已合并 | **合并时间**: 2026-06-09

### 💾 提交

- `66f9488`: fix(design): 设计走查修复 — 弹窗间距/边框色/label规范/组件样式统一

### 📊 改动统计

- **修改文件**: 8 个
- **新增代码**: +157 行
- **删除代码**: -463 行
- **净变化**: -306 行

### 📁 修改文件清单

- `SKILL.md` (+10 -0)
- `client/src/components/ui/select.tsx` (+13 -2)
- `client/src/pages/admin/MemberManagement.tsx` (+9 -212)
- `client/src/pages/admin/ModelConfig.tsx` (+1 -1)
- `client/src/pages/admin/SkillLibrary/BatchDeleteDialog.tsx` (+103 -225)
- `client/src/pages/admin/SkillLibrary/BatchDistributeDialog.tsx` (+1 -1)
- `client/src/pages/admin/SkillLibrary/MCPAddDialog.tsx` (+17 -17)
- `client/src/pages/admin/SkillLibrary/MCPDetail.tsx` (+3 -5)

---

## ⏳ PR #485: Feature/design refresh 2026

**状态**: OPEN

### 💾 提交

- `5f24fdd`: Merge pull request #399 from tx-organization/feature/design-addietang
- `d256e03`: style(file-management): 描边色全量统一到 token #EAEEF4
- `6cc6099`: style: 文字色迁移至语义token + 表头颜色#505050 + FilterChip统一
- `eda1558`: design(admin): 视觉走查刷新与 AIAgent 安全模块重构
- `05a2386`: Merge pull request #404 from tx-organization/feature/design-jingsujiang
- `e8f619a`: Merge pull request #403 from tx-organization/feature/design-addietang
- `1b30733`: feat(design-brennali2): Landing 页定制 + 客户端 UI 优化
- `cf6677a`: Merge pull request #405 from tx-organization/feature/design-brennali2
- ... 还有 92 个提交

### 📊 改动统计

- **修改文件**: 100 个
- **新增代码**: +11032 行
- **删除代码**: -0 行
- **净变化**: +11032 行

### 📁 修改文件清单

- `.claude/backup-to-pages.sh` (+28 -0)
- `.claude/sync-to-mirror.sh` (+53 -0)
- `.codebuddy/figma/1060_12547/figma.html` (+1 -0)
- `.codebuddy/figma/1060_12549/figma.html` (+1 -0)
- `.codebuddy/figma/1060_12622/figma.html` (+1 -0)
- `.codebuddy/figma/1063_5600/figma.html` (+1 -0)
- `.codebuddy/figma/1067_6346/figma.html` (+1 -0)
- `.codebuddy/figma/1067_6554/figma.html` (+1 -0)
- `.codebuddy/figma/1069_6234/figma.html` (+1 -0)
- `.codebuddy/figma/1069_6286/figma.html` (+1 -0)
- `.codebuddy/figma/1078_6631/figma.html` (+1 -0)
- `.codebuddy/figma/1129_8896/figma.html` (+1 -0)
- `.codebuddy/figma/1132_8933/figma.html` (+1 -0)
- `.codebuddy/figma/1140_9957/figma.html` (+1 -0)
- `.codebuddy/figma/1140_9965/figma.html` (+1 -0)
- `.codebuddy/figma/1300_6711/figma.html` (+1 -0)
- `.codebuddy/figma/1300_6712/figma.html` (+1 -0)
- `.codebuddy/figma/1300_6713/figma.html` (+1 -0)
- `.codebuddy/figma/1300_6724/figma.html` (+1 -0)
- `.codebuddy/figma/1300_6725/figma.html` (+1 -0)
- `.codebuddy/figma/1300_6731/figma.html` (+1 -0)
- `.codebuddy/figma/134_783/figma.html` (+1 -0)
- `.codebuddy/figma/134_879/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44081/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44089/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44097/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44134/figma.html` (+1 -0)
- `.codebuddy/figma/3234_9794/figma.html` (+1 -0)
- `.codebuddy/figma/3251_13913/figma.html` (+1 -0)
- `.codebuddy/figma/3251_13937/figma.html` (+1 -0)
- `.codebuddy/figma/3369_11844/figma.html` (+1 -0)
- `.codebuddy/figma/747_183/figma.html` (+1 -0)
- `.codebuddy/figma/786_3666/figma.html` (+1 -0)
- `.codebuddy/figma/786_3680/figma.html` (+1 -0)
- `.codebuddy/figma/786_3697/figma.html` (+1 -0)
- `.codebuddy/figma/788_1117/figma.html` (+1 -0)
- `.codebuddy/figma/788_1121/figma.html` (+1 -0)
- `.codebuddy/figma/788_1125/figma.html` (+1 -0)
- `.codebuddy/figma/795_4842/figma.html` (+1 -0)
- `.codebuddy/figma/795_4954/figma.html` (+1 -0)
- `.codebuddy/figma/848_4199/figma.html` (+1 -0)
- `.codebuddy/figma/917_5511/figma.html` (+1 -0)
- `.codebuddy/figma/917_5795/figma.html` (+1 -0)
- `.codebuddy/figma/953_4221/figma.html` (+1 -0)
- `.codebuddy/figma/981_919/figma.html` (+1 -0)
- `.codebuddy/integration/eop.json` (+7 -0)
- `.codebuddy/merge-conflicts-2026-05-29.md` (+146 -0)
- `.codebuddy/mr-design-brennali2.md` (+52 -0)
- `.codebuddy/rules/anydev/rules/anydev.mdc` (+21 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/COMMON-ERRORS.md` (+569 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/DELIVERY-CHECKLIST.md` (+37 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/DESIGN-AUDIT-PLAYBOOK.md` (+268 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` (+172 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/EVALUATION.md` (+334 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/HANDOFF.md` (+114 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+158 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+152 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/PRODUCT-USAGE.md` (+218 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/QUICK-REFERENCE.md` (+317 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+139 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+983 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+160 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+133 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json` (+36 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/README.md` (+58 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-agent-template.md` (+86 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-agent-template.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-channel-config.md` (+61 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-channel-config.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-members.md` (+68 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-members.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-openclaw-monitor.md` (+103 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-openclaw-monitor.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-ops-observation.md` (+99 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-ops-observation.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-security-management.md` (+77 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-security-management.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-tokens-monitor.md` (+94 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-tokens-monitor.png` (+0 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/admin-sidebar.md` (+749 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/alert.md` (+410 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/avatar.md` (+163 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/badge.md` (+345 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/batch-actions-bar.md` (+386 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/breadcrumb.md` (+167 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+288 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md` (+245 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/chart-stat.md` (+266 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/combobox.md` (+119 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md` (+426 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/date-picker.md` (+313 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dialog-drawer.md` (+299 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+367 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/file-browser.md` (+164 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/form-controls.md` (+209 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md` (+232 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/loading-progress.md` (+227 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md` (+398 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/page-header.md` (+219 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/pagination.md` (+254 -0)

---

## ✅ PR #484: feat(design): 设计走查批量更新 — 用户端 & 管控端

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `2a1af37`: feat(design): 设计走查批量更新 — 用户端 & 管控端

### 📊 改动统计

- **修改文件**: 59 个
- **新增代码**: +9843 行
- **删除代码**: -302 行
- **净变化**: +9541 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+2 -0)
- `.npmrc` (+1 -0)
- `PRD_BILLING_PAY_AS_YOU_GO.md` (+223 -0)
- `client/src/App.tsx` (+10 -2)
- `client/src/components/AdminDisabledOverlay.tsx` (+114 -0)
- `client/src/components/BillingStatusToggle.tsx` (+130 -0)
- `client/src/components/CollectScopePopover.tsx` (+389 -0)
- `client/src/components/GroupMultiFilter.tsx` (+233 -0)
- `client/src/components/GroupSingleFilter.tsx` (+302 -0)
- `client/src/components/MDXRenderer.tsx` (+2 -0)
- `client/src/components/OpenClawCombobox.tsx` (+152 -0)
- `client/src/components/Pagination.tsx` (+129 -0)
- `client/src/components/PluginUpgradeFloating.tsx` (+117 -0)
- `client/src/components/QueryLimitExceededError.tsx` (+48 -0)
- `client/src/components/ServiceSuspendedModal.tsx` (+105 -0)
- `client/src/components/TokenTimeDimensionEditor.tsx` (+480 -0)
- `client/src/components/TraceDrawer.tsx` (+688 -0)
- `client/src/components/VisualUpgradeModal.tsx` (+294 -0)
- `client/src/components/groupTreeShared.tsx` (+287 -0)
- `client/src/components/topnav/HelpPanel.tsx` (+1 -1)
- `client/src/components/ui/button.tsx` (+13 -0)
- `client/src/contexts/PluginUpgradeContext.tsx` (+194 -0)
- `client/src/contexts/ServiceStatusContext.tsx` (+102 -0)
- `client/src/data/trace-data.ts` (+276 -0)
- `client/src/hooks/useAdminDisabled.ts` (+25 -0)
- `client/src/hooks/useClsCollectScope.ts` (+69 -0)
- `client/src/lib/agentBillingStore.ts` (+82 -0)
- `client/src/lib/clsScopeMock.ts` (+118 -0)
- `client/src/lib/mockData.ts` (+1 -1)
- `client/src/lib/openclawStore.ts` (+11 -0)
- `client/src/lib/upgradePushStore.ts` (+1 -1)
- `client/src/main.tsx` (+8 -1)
- `client/src/pages/LandingPage.tsx` (+293 -0)
- `client/src/pages/admin/AgentTemplate.tsx` (+1 -1)
- `client/src/pages/admin/AuthSourceImportDialog.tsx` (+1 -1)
- `client/src/pages/admin/BatchUpdateNotice.tsx` (+3 -3)
- `client/src/pages/admin/CloudDevManagement.tsx` (+1 -1)
- `client/src/pages/admin/ImageManagement.tsx` (+1 -1)
- `client/src/pages/admin/ImageManagement/PushUpgradeDialog.tsx` (+4 -4)
- `client/src/pages/admin/ImageManagement/PushUpgradePopover.tsx` (+4 -4)
- `client/src/pages/admin/ImageManagement/UpdateRecordSidebar.tsx` (+244 -0)
- `client/src/pages/admin/ImageManagement/UpdateRecordsDrawer.tsx` (+7 -7)
- `client/src/pages/admin/KnowledgeManagement.tsx` (+34 -0)
- `client/src/pages/admin/MemberManagement/AgentInstanceHandlingDialog.tsx` (+700 -0)
- `client/src/pages/admin/MemberManagement/ConfigDiffDialog.tsx` (+111 -0)
- `client/src/pages/admin/MemberManagement/mock.ts` (+16 -11)
- `client/src/pages/admin/MemoryManagement/components/ProServiceCard.tsx` (+742 -0)
- `client/src/pages/admin/ModelConfig.tsx` (+239 -6)
- `client/src/pages/admin/PlatformPolicy.tsx` (+6 -6)
- `client/src/pages/admin/SessionDetailLegacy.tsx` (+329 -0)
- `client/src/pages/admin/SkillRolesTab.tsx` (+1 -1)
- `client/src/pages/landing/Enterprise.tsx` (+1 -1)
- `client/src/pages/tenant/GroupChangeComponents.tsx` (+362 -0)
- `client/src/pages/tenant/HelpDocs.tsx` (+1 -1)
- `client/src/pages/tenant/MyOpenClaw.tsx` (+42 -1)
- `client/src/pages/tenant/OpenClawDetail.tsx` (+126 -2)
- `client/src/pages/tenant/OpenClawDetailGuide.tsx` (+774 -242)
- `client/src/pages/tenant/ResetPassword.tsx` (+3 -3)
- `docs/agent-instance-handling-ui.html` (+1190 -0)

---

## ✅ PR #483: feat(design-system): 扩充设计系统组件规范文档与预览页面

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `c0e26cb`: feat(design-system): 扩充设计系统组件规范文档与预览页面

### 📊 改动统计

- **修改文件**: 32 个
- **新增代码**: +7139 行
- **删除代码**: -490 行
- **净变化**: +6649 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+8 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+27 -17)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+17 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/alert.md` (+203 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/avatar.md` (+73 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/breadcrumb.md` (+67 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+182 -72)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/input-select.md` (+159 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/selection-controls.md` (+191 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+267 -99)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tag-label.md` (+88 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/toast.md` (+152 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tooltip.md` (+72 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/transfer.md` (+239 -137)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tree.md` (+157 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/typography.md` (+134 -0)
- `client/src/App.tsx` (+12 -0)
- `client/src/components/ui/input-group.tsx` (+6 -3)
- `client/src/components/ui/sonner.tsx` (+25 -8)
- `client/src/components/ui/tabs.tsx` (+1 -1)
- `client/src/index.css` (+3 -6)
- `client/src/pages/DesignSystemComponents.tsx` (+1779 -147)
- `client/src/pages/PreviewIndex.tsx` (+6 -0)
- `client/src/pages/design-system/component-usage.generated.json` (+1337 -0)
- `client/src/pages/preview/AvatarPreview.tsx` (+81 -0)
- `client/src/pages/preview/BreadcrumbPreview.tsx` (+88 -0)
- `client/src/pages/preview/EmptyStateSpecVerify.tsx` (+344 -0)
- `client/src/pages/preview/ToastPreview.tsx` (+72 -0)
- `client/src/pages/preview/TransferPreview.tsx` (+134 -0)
- `client/src/pages/preview/TreePreview.tsx` (+166 -0)
- `scripts/count-component-usage.mjs` (+168 -0)
- `table-spec-preview.html` (+881 -0)

---

## ✅ PR #482: fix(design): 统一图表hover背景色、子网标题对齐、策略卡片底色token、GroupSelect确认按钮

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `b3e34b3`: fix(design): 统一图表hover背景色、子网标题对齐、策略卡片底色token、GroupSelect确认按钮

### 📊 改动统计

- **修改文件**: 5 个
- **新增代码**: +13 行
- **删除代码**: -15 行
- **净变化**: -2 行

### 📁 修改文件清单

- `client/src/components/GroupSelect.tsx` (+3 -0)
- `client/src/pages/admin/MemoryManagement/components/DefaultMemoryVersion.tsx` (+1 -1)
- `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx` (+4 -9)
- `client/src/pages/admin/OpsObservation.tsx` (+2 -2)
- `client/src/pages/admin/SecurityGroupManagement.tsx` (+3 -3)

---

## ✅ PR #481: fix(design): 统一图表hover背景色、子网标题对齐、策略卡片底色token

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `f5765cf`: fix(design): 统一图表hover背景色、子网标题对齐、策略卡片底色token

### 📊 改动统计

- **修改文件**: 4 个
- **新增代码**: +10 行
- **删除代码**: -15 行
- **净变化**: -5 行

### 📁 修改文件清单

- `client/src/pages/admin/MemoryManagement/components/DefaultMemoryVersion.tsx` (+1 -1)
- `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx` (+4 -9)
- `client/src/pages/admin/OpsObservation.tsx` (+2 -2)
- `client/src/pages/admin/SecurityGroupManagement.tsx` (+3 -3)

---

## ✅ PR #480: feat(design-sarry): 弹窗规范修复、筛选面板重构、网络管理展开行边框优化

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `8f79b90`: feat(design-sarry): 弹窗规范修复、筛选面板重构、网络管理展开行边框优化

### 📊 改动统计

- **修改文件**: 83 个
- **新增代码**: +7342 行
- **删除代码**: -3921 行
- **净变化**: +3421 行

### 📁 修改文件清单

- `SKILL-GLOBAL-COMPONENTS.md` (+394 -11)
- `SKILL.md` (+128 -61)
- `client/src/App.tsx` (+2 -0)
- `client/src/components/AdminLayout.tsx` (+1 -1)
- `client/src/components/GroupSelect.tsx` (+497 -0)
- `client/src/components/OpenClawCombobox.tsx` (+0 -152)
- `client/src/components/ScopeSelect.tsx` (+146 -0)
- `client/src/components/SecurityScanCard.tsx` (+3 -3)
- `client/src/components/_internal/README.md` (+31 -0)
- `client/src/components/_internal/ScopeEditPopover.tsx` (+37 -30)
- `client/src/components/_internal/ScopeFilterDropdown.tsx` (+253 -0)
- `client/src/components/_internal/TableHeaderFilter.tsx` (+191 -0)
- `client/src/components/_internal/TableHeaderTreeFilter.tsx` (+309 -0)
- `client/src/components/_internal/TreeSelectFilter.tsx` (+272 -0)
- `client/src/components/admin/ColumnFilters.tsx` (+0 -331)
- `client/src/components/policy/TokenValueEditor.tsx` (+4 -4)
- `client/src/components/ui/Typography.tsx` (+1 -1)
- `client/src/components/ui/alert-dialog.tsx` (+20 -3)
- `client/src/components/ui/dialog.tsx` (+3 -3)
- `client/src/components/ui/dropdown-menu.tsx` (+4 -4)
- `client/src/components/ui/filter-trigger.tsx` (+138 -0)
- `client/src/components/ui/radio-card.tsx` (+4 -3)
- `client/src/components/ui/searchable-select.tsx` (+6 -0)
- `client/src/components/ui/select-panel.tsx` (+228 -0)
- `client/src/components/ui/select.tsx` (+331 -14)
- `client/src/components/ui/status-tag.tsx` (+1 -1)
- `client/src/components/ui/textarea.tsx` (+3 -3)
- `client/src/components/ui/tree-select.tsx` (+121 -0)
- `client/src/index.css` (+3 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+1 -1)
- `client/src/pages/admin/AgentToolLibrary.tsx` (+1 -3)
- `client/src/pages/admin/AuthSourceImportDialog.tsx` (+11 -11)
- `client/src/pages/admin/ChannelConfig.tsx` (+55 -40)
- `client/src/pages/admin/ComponentPreview.tsx` (+477 -0)
- `client/src/pages/admin/FileManagement.tsx` (+157 -238)
- `client/src/pages/admin/ImageManagement.tsx` (+16 -26)
- `client/src/pages/admin/ImageManagement/AgentTypesTable.tsx` (+4 -4)
- `client/src/pages/admin/ImageManagement/PublicImageHistoryDialog.tsx` (+1 -1)
- `client/src/pages/admin/MemberManagement.tsx` (+168 -311)
- `client/src/pages/admin/MemberManagement/GroupDialog.tsx` (+11 -20)
- `client/src/pages/admin/MemberManagement/GroupView.tsx` (+6 -5)
- `client/src/pages/admin/MemberManagement/NodeContentPanel.tsx` (+11 -11)
- `client/src/pages/admin/MemberManagement/SyncResultDialog.tsx` (+12 -11)
- `client/src/pages/admin/MemoryManagement/components/DefaultMemoryVersion.tsx` (+164 -189)
- `client/src/pages/admin/MemoryManagement/components/GroupColumn.tsx` (+4 -200)
- `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx` (+63 -98)
- `client/src/pages/admin/MemoryManagement/components/ProActivationDialog.tsx` (+1 -1)
- `client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx` (+11 -9)
- `client/src/pages/admin/ModelConfig.tsx` (+18 -28)
- `client/src/pages/admin/OpenClawMonitor.tsx` (+157 -335)
- `client/src/pages/admin/OpsObservation.tsx` (+89 -85)
- `client/src/pages/admin/PlatformPolicy.tsx` (+20 -447)
- `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/BashPolicyList.tsx` (+36 -26)
- `client/src/pages/admin/SecurityGroupManagement.tsx` (+216 -233)
- `client/src/pages/admin/ServerManagement.tsx` (+8 -7)
- `client/src/pages/admin/SessionManagement.tsx` (+74 -69)
- `client/src/pages/admin/SkillLibrary/AddCategoryDialog.tsx` (+6 -6)
- `client/src/pages/admin/SkillLibrary/BatchDeleteDialog.tsx` (+3 -3)
- `client/src/pages/admin/SkillLibrary/BatchDistributeDialog.tsx` (+156 -348)
- `client/src/pages/admin/SkillLibrary/CategoryManagementDialog.tsx` (+9 -20)
- `client/src/pages/admin/SkillLibrary/DeleteSkillDialog.tsx` (+32 -43)
- `client/src/pages/admin/SkillLibrary/EditCategoryDialog.tsx` (+1 -1)
- `client/src/pages/admin/SkillLibrary/EditScopeDialog.tsx` (+3 -3)
- `client/src/pages/admin/SkillLibrary/EnableCOSDialog.tsx` (+1 -1)
- `client/src/pages/admin/SkillLibrary/MCPAddDialog.tsx` (+1 -1)
- `client/src/pages/admin/SkillLibrary/MCPListTab.tsx` (+8 -8)
- `client/src/pages/admin/SkillLibrary/PluginListTab.tsx` (+8 -8)
- `client/src/pages/admin/SkillLibrary/PluginUpdateDialog.tsx` (+3 -3)
- `client/src/pages/admin/SkillLibrary/PluginUploadDialog.tsx` (+2 -2)
- `client/src/pages/admin/SkillLibrary/SkillDetail.tsx` (+5 -14)
- `client/src/pages/admin/SkillLibrary/SkillInitialPackageTab.tsx` (+67 -73)
- `client/src/pages/admin/SkillLibrary/SkillListTab.tsx` (+16 -142)
- `client/src/pages/admin/SkillLibrary/SkillUpdateDialog.tsx` (+3 -3)
- `client/src/pages/admin/SkillLibrary/SkillUploadDialog.tsx` (+8 -8)
- `client/src/pages/admin/SkillRolesTab.tsx` (+74 -79)
- `client/src/pages/admin/TokensMonitor.tsx` (+94 -87)
- `client/src/pages/admin/VersionManagement/HistoryTab.tsx` (+4 -3)
- `client/src/pages/admin/VersionManagement/components/CreateCommandDialog.tsx` (+15 -15)
- `client/src/pages/admin/VersionManagement/components/DispatchCommandDialog.tsx` (+16 -16)
- `docs/admin-dropdown-inventory.html` (+575 -0)
- `docs/dialog-audit-report.html` (+453 -0)
- `docs/dialog-fix-summary.html` (+209 -0)
- `docs/dropdown-panels-audit.html` (+647 -0)

---

## ✅ PR #477: docs(portable-skill): 补 NumberCard 可移植 spec（媛媛交付包）

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `ab3e7eb`: feat(ui): add NumberCard component & migrate Tokens overview cards
- `e3e3305`: docs(portable-skill): add NumberCard component spec for Monday handoff

### 📊 改动统计

- **修改文件**: 7 个
- **新增代码**: +647 行
- **删除代码**: -45 行
- **净变化**: +602 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+1 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+4 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md` (+271 -0)
- `SKILL-GLOBAL-COMPONENTS.md` (+87 -0)
- `client/src/components/ui/number-card.tsx` (+196 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+76 -0)
- `client/src/pages/admin/TokensMonitor.tsx` (+12 -44)

---

## ✅ PR #476: fix(ui): DesignSystemComponents 缺失 NumberCard import 导致预览白屏

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `ab3e7eb`: feat(ui): add NumberCard component & migrate Tokens overview cards

### 📊 改动统计

- **修改文件**: 4 个
- **新增代码**: +371 行
- **删除代码**: -44 行
- **净变化**: +327 行

### 📁 修改文件清单

- `SKILL-GLOBAL-COMPONENTS.md` (+87 -0)
- `client/src/components/ui/number-card.tsx` (+196 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+76 -0)
- `client/src/pages/admin/TokensMonitor.tsx` (+12 -44)

---

## ✅ PR #473: feat(design): 全局组件规范化 & 表格组件重构 & 多页面样式统一

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `fb53477`: feat(design): 全局组件规范化 & 表格组件重构 & 多页面样式统一

### 📊 改动统计

- **修改文件**: 100 个
- **新增代码**: +3532 行
- **删除代码**: -2423 行
- **净变化**: +1109 行

### 📁 修改文件清单

- `.codebuddy/figma/3226_44081/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44089/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44097/figma.html` (+1 -0)
- `.codebuddy/figma/3226_44134/figma.html` (+1 -0)
- `.codebuddy/figma/3369_11844/figma.html` (+1 -0)
- `.codebuddy/merge-conflicts-2026-05-29.md` (+146 -0)
- `.playwright-cli/page-2026-05-19T11-31-50-108Z.yml` (+0 -198)
- `.playwright-cli/page-2026-05-19T11-31-59-406Z.yml` (+0 -303)
- `.playwright-cli/page-2026-05-19T11-32-15-609Z.yml` (+0 -310)
- `.playwright-cli/page-2026-05-19T11-32-27-443Z.png` (+0 -0)
- `.playwright-cli/page-2026-06-02T09-04-11-005Z.yml` (+0 -563)
- `SKILL-GLOBAL-COMPONENTS.md` (+40 -26)
- `SKILL.md` (+1 -1)
- `assets/CodeBuddyAssets/3226_44081/1.svg` (+16 -0)
- `assets/CodeBuddyAssets/3226_44081/10.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/11.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/12.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/13.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/14.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/15.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/16.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/17.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/18.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/19.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/2.svg` (+4 -0)
- `assets/CodeBuddyAssets/3226_44081/20.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/21.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/22.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/23.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/24.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/25.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/26.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/3.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/4.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/5.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/6.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/7.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44081/8.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44081/9.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44089/1.svg` (+4 -0)
- `assets/CodeBuddyAssets/3226_44097/1.svg` (+3 -0)
- `assets/CodeBuddyAssets/3226_44097/10.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/11.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/12.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/13.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/14.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/15.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/16.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/17.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/18.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/2.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/3.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/4.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/5.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/6.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/7.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/8.svg` (+9 -0)
- `assets/CodeBuddyAssets/3226_44097/9.svg` (+9 -0)
- `assets/CodeBuddyAssets/3369_11844/1.svg` (+16 -0)
- `assets/CodeBuddyAssets/3369_11844/2.svg` (+4 -0)
- `assets/CodeBuddyAssets/3369_11844/3.svg` (+3 -0)
- `client/public/fonts/DINNextLTPro-Bold 2.OTF` (+0 -0)
- `client/public/fonts/DINNextLTPro-Regular 2.otf` (+0 -0)
- `client/public/icon/AI Agent资产 2.svg` (+9 -0)
- `client/public/icon/威胁告警 2.svg` (+9 -0)
- `client/public/icon/存在风险 2.svg` (+19 -0)
- `client/public/research/design-confirm-card-preview.html` (+158 -0)
- `client/src/App.tsx` (+2 -0)
- `client/src/components/Pagination.tsx` (+0 -164)
- `client/src/components/ScopePopover.tsx` (+465 -0)
- `client/src/components/policy/PolicyEditCard 2.tsx` (+384 -0)
- `client/src/components/policy/QuotaPolicyCard 2.tsx` (+364 -0)
- `client/src/components/policy/TokenValueEditor 2.tsx` (+127 -0)
- `client/src/components/policy/index 2.ts` (+27 -0)
- `client/src/components/policy/types 2.ts` (+46 -0)
- `client/src/components/ui/admin-notice-alert.tsx` (+39 -17)
- `client/src/components/ui/empty.tsx` (+3 -3)
- `client/src/components/ui/pagination.tsx` (+30 -17)
- `client/src/components/ui/table.tsx` (+111 -86)
- `client/src/index.css` (+73 -115)
- `client/src/pages/ComponentPreview.tsx` (+534 -3)
- `client/src/pages/PreviewIndex.tsx` (+9 -0)
- `client/src/pages/admin/BasicInfo.tsx` (+6 -3)
- `client/src/pages/admin/MemberManagement.tsx` (+86 -92)
- `client/src/pages/admin/MemoryManagement/MemoryManagement.tsx` (+2 -2)
- `client/src/pages/admin/MemoryManagement/components/DefaultMemoryVersion.tsx` (+1 -1)
- `client/src/pages/admin/MemoryManagement/components/DisableConfirmDialog.tsx` (+3 -1)
- `client/src/pages/admin/MemoryManagement/components/EnableConfirmDialog.tsx` (+3 -3)
- `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx` (+158 -209)
- `client/src/pages/admin/MemoryManagement/components/OneClickUpgradeDialog.tsx` (+2 -3)
- `client/src/pages/admin/MemoryManagement/components/ProActivationDialog.tsx` (+2 -2)
- `client/src/pages/admin/MemoryManagement/components/ProCloseDialog.tsx` (+56 -51)
- `client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx` (+84 -92)
- `client/src/pages/admin/OpenClawMonitor.tsx` (+57 -58)
- `client/src/pages/admin/Security/AIAgent/Alarms/AlarmsList.tsx` (+1 -1)
- `client/src/pages/admin/Security/AIAgent/Alarms/BatchOperatorDialog.tsx` (+10 -9)
- `client/src/pages/admin/Security/AIAgent/Assets/AgentAssetsList.tsx` (+1 -1)
- `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/BashPolicyList.tsx` (+5 -2)
- `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/EditPolicyDrawer.tsx` (+23 -22)
- `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/PolicyDetailDrawer.tsx` (+54 -65)

---

## ✅ PR #470: Design updates from feature/design-miekoyychen

**状态**: ✓ 已合并 | **合并时间**: 2026-06-08

### 💾 提交

- `c6a2aac`: feat(skills):添加设计审查与前端工艺技能配置
- `c5f2ad7`: docs(skill): 添加设计审查流程文档

### 📊 改动统计

- **修改文件**: 95 个
- **新增代码**: +39970 行
- **删除代码**: -45 行
- **净变化**: +39925 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-design-skill/SKILL.md` (+3 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/DELIVERY-CHECKLIST.md` (+3 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/DESIGN-AUDIT-PLAYBOOK.md` (+270 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/HANDOFF.md` (+3 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+2 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+3 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+4 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+5 -0)
- `.codebuddy/skills/impeccable/SKILL.md` (+182 -0)
- `.codebuddy/skills/impeccable/agents/impeccable_asset_producer.toml` (+92 -0)
- `.codebuddy/skills/impeccable/agents/impeccable_manual_edit_applier.toml` (+95 -0)
- `.codebuddy/skills/impeccable/agents/openai.yaml` (+4 -0)
- `.codebuddy/skills/impeccable/reference/adapt.md` (+311 -0)
- `.codebuddy/skills/impeccable/reference/animate.md` (+201 -0)
- `.codebuddy/skills/impeccable/reference/audit.md` (+133 -0)
- `.codebuddy/skills/impeccable/reference/bolder.md` (+113 -0)
- `.codebuddy/skills/impeccable/reference/brand.md` (+108 -0)
- `.codebuddy/skills/impeccable/reference/clarify.md` (+288 -0)
- `.codebuddy/skills/impeccable/reference/codex.md` (+105 -0)
- `.codebuddy/skills/impeccable/reference/colorize.md` (+257 -0)
- `.codebuddy/skills/impeccable/reference/craft.md` (+123 -0)
- `.codebuddy/skills/impeccable/reference/critique.md` (+790 -0)
- `.codebuddy/skills/impeccable/reference/delight.md` (+302 -0)
- `.codebuddy/skills/impeccable/reference/distill.md` (+111 -0)
- `.codebuddy/skills/impeccable/reference/document.md` (+429 -0)
- `.codebuddy/skills/impeccable/reference/extract.md` (+69 -0)
- `.codebuddy/skills/impeccable/reference/harden.md` (+347 -0)
- `.codebuddy/skills/impeccable/reference/init.md` (+172 -0)
- `.codebuddy/skills/impeccable/reference/interaction-design.md` (+189 -0)
- `.codebuddy/skills/impeccable/reference/layout.md` (+161 -0)
- `.codebuddy/skills/impeccable/reference/live.md` (+699 -0)
- `.codebuddy/skills/impeccable/reference/onboard.md` (+234 -0)
- `.codebuddy/skills/impeccable/reference/optimize.md` (+258 -0)
- `.codebuddy/skills/impeccable/reference/overdrive.md` (+130 -0)
- `.codebuddy/skills/impeccable/reference/polish.md` (+241 -0)
- `.codebuddy/skills/impeccable/reference/product.md` (+60 -0)
- `.codebuddy/skills/impeccable/reference/quieter.md` (+99 -0)
- `.codebuddy/skills/impeccable/reference/shape.md` (+165 -0)
- `.codebuddy/skills/impeccable/reference/typeset.md` (+279 -0)
- `.codebuddy/skills/impeccable/scripts/cleanup-deprecated.mjs` (+284 -0)
- `.codebuddy/skills/impeccable/scripts/command-metadata.json` (+94 -0)
- `.codebuddy/skills/impeccable/scripts/context-signals.mjs` (+225 -0)
- `.codebuddy/skills/impeccable/scripts/context.mjs` (+266 -0)
- `.codebuddy/skills/impeccable/scripts/critique-storage.mjs` (+242 -0)
- `.codebuddy/skills/impeccable/scripts/design-parser.mjs` (+835 -0)
- `.codebuddy/skills/impeccable/scripts/detect-csp.mjs` (+198 -0)
- `.codebuddy/skills/impeccable/scripts/detect.mjs` (+21 -0)
- `.codebuddy/skills/impeccable/scripts/detector/browser/injected/index.mjs` (+1725 -0)
- `.codebuddy/skills/impeccable/scripts/detector/cli/main.mjs` (+244 -0)
- `.codebuddy/skills/impeccable/scripts/detector/detect-antipatterns-browser.js` (+4543 -0)
- `.codebuddy/skills/impeccable/scripts/detector/detect-antipatterns.mjs` (+43 -0)
- `.codebuddy/skills/impeccable/scripts/detector/engines/browser/detect-url.mjs` (+252 -0)
- `.codebuddy/skills/impeccable/scripts/detector/engines/regex/detect-text.mjs` (+535 -0)
- `.codebuddy/skills/impeccable/scripts/detector/engines/static-html/css-cascade.mjs` (+986 -0)
- `.codebuddy/skills/impeccable/scripts/detector/engines/static-html/detect-html.mjs` (+208 -0)
- `.codebuddy/skills/impeccable/scripts/detector/engines/visual/screenshot-contrast.mjs` (+189 -0)
- `.codebuddy/skills/impeccable/scripts/detector/findings.mjs` (+12 -0)
- `.codebuddy/skills/impeccable/scripts/detector/node/file-system.mjs` (+198 -0)
- `.codebuddy/skills/impeccable/scripts/detector/profile/profiler.mjs` (+166 -0)
- `.codebuddy/skills/impeccable/scripts/detector/registry/antipatterns.mjs` (+419 -0)
- `.codebuddy/skills/impeccable/scripts/detector/rules/checks.mjs` (+2316 -0)
- `.codebuddy/skills/impeccable/scripts/detector/shared/color.mjs` (+124 -0)
- `.codebuddy/skills/impeccable/scripts/detector/shared/constants.mjs` (+101 -0)
- `.codebuddy/skills/impeccable/scripts/detector/shared/page.mjs` (+7 -0)
- `.codebuddy/skills/impeccable/scripts/impeccable-paths.mjs` (+126 -0)
- `.codebuddy/skills/impeccable/scripts/is-generated.mjs` (+69 -0)
- `.codebuddy/skills/impeccable/scripts/live-accept.mjs` (+689 -0)
- `.codebuddy/skills/impeccable/scripts/live-browser-session.js` (+123 -0)
- `.codebuddy/skills/impeccable/scripts/live-browser.js` (+8830 -0)
- `.codebuddy/skills/impeccable/scripts/live-commit-manual-edits.mjs` (+1241 -0)
- `.codebuddy/skills/impeccable/scripts/live-complete.mjs` (+75 -0)
- `.codebuddy/skills/impeccable/scripts/live-completion.mjs` (+18 -0)
- `.codebuddy/skills/impeccable/scripts/live-copy-edit-agent.mjs` (+683 -0)
- `.codebuddy/skills/impeccable/scripts/live-discard-manual-edits.mjs` (+51 -0)
- `.codebuddy/skills/impeccable/scripts/live-event-validation.mjs` (+136 -0)
- `.codebuddy/skills/impeccable/scripts/live-inject.mjs` (+459 -0)
- `.codebuddy/skills/impeccable/scripts/live-insert-ui.mjs` (+458 -0)
- `.codebuddy/skills/impeccable/scripts/live-insert.mjs` (+232 -0)
- `.codebuddy/skills/impeccable/scripts/live-manual-edit-evidence.mjs` (+363 -0)
- `.codebuddy/skills/impeccable/scripts/live-manual-edits-buffer.mjs` (+152 -0)
- `.codebuddy/skills/impeccable/scripts/live-poll.mjs` (+378 -0)
- `.codebuddy/skills/impeccable/scripts/live-resume.mjs` (+94 -0)
- `.codebuddy/skills/impeccable/scripts/live-server.mjs` (+2190 -0)
- `.codebuddy/skills/impeccable/scripts/live-session-store.mjs` (+271 -0)
- `.codebuddy/skills/impeccable/scripts/live-status.mjs` (+61 -0)
- `.codebuddy/skills/impeccable/scripts/live-wrap.mjs` (+842 -0)
- `.codebuddy/skills/impeccable/scripts/live.mjs` (+246 -0)
- `.codebuddy/skills/impeccable/scripts/modern-screenshot.umd.js` (+14 -0)
- `.codebuddy/skills/impeccable/scripts/palette.mjs` (+633 -0)
- `.codebuddy/skills/impeccable/scripts/pin.mjs` (+214 -0)
- `PRODUCT.md` (+37 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+0 -35)
- `docs/design-audit/README.md` (+8 -5)
- `docs/design-audit/new-chat-resume-template-2026-06-06.md` (+2 -0)

---

## ✅ PR #466: feat(design-skill): 新增开发使用说明及多组件 spec

**状态**: ✓ 已合并 | **合并时间**: 2026-06-06

### 💾 提交

- `10ee173`: feat(design-skill): 新增开发使用说明及多组件 spec

### 📊 改动统计

- **修改文件**: 22 个
- **新增代码**: +1614 行
- **删除代码**: -49 行
- **净变化**: +1565 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/DELIVERY-CHECKLIST.md` (+5 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` (+169 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/HANDOFF.md` (+8 -6)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+7 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+33 -7)
- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+7 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+7 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+24 -8)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+20 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/batch-actions-bar.md` (+209 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/chart-stat.md` (+134 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/combobox.md` (+240 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/loading-progress.md` (+131 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/navigation-sidebar.md` (+173 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/popover-dropdown-menu.md` (+164 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/upload-file-browser.md` (+145 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+9 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/references/migration-map.md` (+7 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/verify-portable-skill.mjs` (+101 -6)
- `docs/design-audit/README.md` (+5 -4)
- `docs/design-audit/new-chat-resume-template-2026-06-06.md` (+14 -10)

---

## ✅ PR #465: Feature/design miekoyychen

**状态**: ✓ 已合并 | **合并时间**: 2026-06-06

### 💾 提交

- `40d89a5`: docs(skill): 收口规范措辞并补充交付与组件文档
- `1b89a3f`: refactor(design-spec): 回写确认结果并替换硬编码色值为 token
- `6539f34`: docs(design-audit): 完善新对话续接模板
- `48cd158`: docs: 更新协作入口路由与周一协作计划

### 📊 改动统计

- **修改文件**: 57 个
- **新增代码**: +4039 行
- **删除代码**: -249 行
- **净变化**: +3790 行

### 📁 修改文件清单

- `.codebuddy/integration/eop.json` (+7 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/DELIVERY-CHECKLIST.md` (+33 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/HANDOFF.md` (+111 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/INDEX.md` (+109 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json` (+75 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+32 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+3 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` (+118 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+16 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+11 -11)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md` (+14 -13)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/date-picker.md` (+224 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dialog-drawer.md` (+146 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+4 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/form-controls.md` (+115 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/page-header.md` (+3 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/pagination.md` (+143 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/search-filter-bar.md` (+174 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/status-badge.md` (+126 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+10 -10)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs-segment.md` (+117 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/css/tokens.css` (+23 -6)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/card.html` (+6 -7)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/date-picker.html` (+50 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/empty-state.html` (+5 -6)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/search-filter-bar.html` (+43 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/table.html` (+5 -6)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/card.tsx` (+2 -2)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/date-picker.tsx` (+63 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/empty-state.tsx` (+3 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/search-filter-bar.tsx` (+34 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/table.tsx` (+8 -9)
- `.codebuddy/skills/clawpro-portable-design-skill/qa/admin-checklist.md` (+2 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/references/admin.md` (+11 -9)
- `.codebuddy/skills/clawpro-portable-design-skill/references/assets-icons.md` (+1 -1)
- `.codebuddy/skills/clawpro-portable-design-skill/references/components.md` (+11 -12)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+47 -16)
- `.codebuddy/skills/clawpro-portable-design-skill/references/foundation.md` (+24 -18)
- `.codebuddy/skills/clawpro-portable-design-skill/references/landing.md` (+5 -4)
- `.codebuddy/skills/clawpro-portable-design-skill/references/migration-map.md` (+4 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/references/page-recipes.md` (+7 -3)
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md` (+24 -20)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/package-portable-skill.sh` (+22 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/verify-portable-skill.mjs` (+33 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/colors.md` (+54 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/design-tokens.json` (+19 -8)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/radius-shadow.md` (+31 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/spacing.md` (+24 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/typography.md` (+50 -0)
- `docs/design-audit/README.md` (+97 -61)
- `docs/design-audit/design-confirmation-decisions-2026-06-06.md` (+70 -0)
- `docs/design-audit/design-governance-and-delivery-plan-2026-06-05.md` (+2 -2)
- `docs/design-audit/monday-collaboration-plan-2026-06-06.md` (+274 -0)
- `docs/design-audit/monday-delivery-collab-summary-2026-06-06.md` (+19 -2)
- `docs/design-audit/new-chat-resume-template-2026-06-06.md` (+150 -0)
- `docs/design-audit/p0-conflict-confirmation-page-2026-06-06.html` (+912 -0)
- `docs/design-audit/tenant-landing-design-confirmations-2026-06-06.md` (+313 -0)

---

## ✅ PR #463: Design updates

**状态**: ✓ 已合并 | **合并时间**: 2026-06-06

### 💾 提交

- `a86297b`: feat(design): 添加 ClawPro 可移植设计交付包

### 📊 改动统计

- **修改文件**: 34 个
- **新增代码**: +3028 行
- **删除代码**: -0 行
- **净变化**: +3028 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-portable-design-skill/README.md` (+103 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` (+112 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/STRUCTURE.md` (+91 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json` (+36 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/button.md` (+135 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md` (+133 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md` (+127 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/page-header.md` (+116 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md` (+180 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/css/tokens.css` (+13 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/card.html` (+30 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/empty-state.html` (+51 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/html-css/table.html` (+56 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/card.tsx` (+14 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/empty-state.tsx` (+19 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/portable/react/table.tsx` (+35 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/qa/admin-checklist.md` (+36 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/qa/component-review-checklist.md` (+11 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/qa/landing-checklist.md` (+9 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/qa/tenant-checklist.md` (+34 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/admin.md` (+80 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/assets-icons.md` (+81 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/components.md` (+111 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` (+60 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/foundation.md` (+109 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/landing.md` (+73 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/migration-map.md` (+59 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/page-recipes.md` (+169 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md` (+96 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/scripts/check-design-usage.mjs` (+147 -0)
- `.codebuddy/skills/clawpro-portable-design-skill/tokens/design-tokens.json` (+41 -0)
- `docs/design-audit/monday-delivery-collab-summary-2026-06-06.md` (+150 -0)
- `docs/design-audit/monday-delivery-package-structure-2026-06-06.md` (+352 -0)
- `docs/design-audit/monday-delivery-work-split-2026-06-06.md` (+159 -0)

---

## ✅ PR #462: feat(design-skill): 添加 ClawPro 设计治理 Skill

**状态**: ✓ 已合并 | **合并时间**: 2026-06-05

### 💾 提交

- `bce1a3b`: feat(ui): 优化管理侧边栏与策略页样式并统一设计变量
- `8c196ab`: style: 将字体和框架导入移至文件顶部
- `b3042b2`: feat(ui): 添加前往用户端箭头悬浮动效
- `f5d71a2`: fix(button): 使用CSS变量统一悬停背景色
- `651807f`: refactor(radio-group): 对齐 Checkbox 现状的品牌色 + 圆圈白底
- `f4ca9be`: docs(skill): 重写 §8 Checkbox + §21 RadioGroup 规范
- `4f0d96a`: style(admin): 镜像管理、技能包、安全组页面样式优化
- `1704c47`: feat(design-system): 新增多个组件展示与角色头像映射
- ... 还有 19 个提交

### 📊 改动统计

- **修改文件**: 22 个
- **新增代码**: +3476 行
- **删除代码**: -1500 行
- **净变化**: +1976 行

### 📁 修改文件清单

- `.codebuddy/skills/clawpro-design-skill/SKILL.md` (+63 -0)
- `.codebuddy/skills/clawpro-design-skill/assets/icon-gallery.html` (+178 -0)
- `.codebuddy/skills/clawpro-design-skill/references/admin.md` (+80 -0)
- `.codebuddy/skills/clawpro-design-skill/references/assets-icons.md` (+81 -0)
- `.codebuddy/skills/clawpro-design-skill/references/components.md` (+111 -0)
- `.codebuddy/skills/clawpro-design-skill/references/conflict-log.md` (+60 -0)
- `.codebuddy/skills/clawpro-design-skill/references/foundation.md` (+109 -0)
- `.codebuddy/skills/clawpro-design-skill/references/icon-registry.json` (+36 -0)
- `.codebuddy/skills/clawpro-design-skill/references/landing.md` (+73 -0)
- `.codebuddy/skills/clawpro-design-skill/references/page-recipes.md` (+169 -0)
- `.codebuddy/skills/clawpro-design-skill/references/tenant.md` (+96 -0)
- `.codebuddy/skills/clawpro-design-skill/scripts/check-design-usage.mjs` (+146 -0)
- `.codebuddy/skills/clawpro-skill/SKILL.md` (+0 -1488)
- `client/src/components/AdminNoticeBar.tsx` (+1 -1)
- `client/src/index.css` (+64 -3)
- `client/src/pages/DesignSystemComponents.tsx` (+35 -0)
- `client/src/pages/tenant/OpenClawDetail.tsx` (+8 -8)
- `docs/ClawPro设计规范沉淀与AI约束治理方案.md` (+475 -0)
- `docs/design-audit/design-audit-round3-materials-and-template-2026-06-05.md` (+305 -0)
- `docs/design-audit/design-component-compliance-audit-2026-06-05.md` (+653 -0)
- `docs/design-audit/design-governance-and-delivery-plan-2026-06-05.md` (+280 -0)
- `docs/design-audit/design-spec-conflict-audit-2026-06-05.md` (+453 -0)

---

## ✅ PR #461: design: 设计走查批次更新（全局组件 + AIAgent/SkillLibrary/MemberManagement/Version…

**状态**: ✓ 已合并 | **合并时间**: 2026-06-05

### 💾 提交

- `dcc85ab`: design: 设计走查批次更新（全局组件 + AIAgent/SkillLibrary/MemberManagement/Version…

### 📊 改动统计

- **修改文件**: 71 个
- **新增代码**: +2562 行
- **删除代码**: -1348 行
- **净变化**: +1214 行

### 📁 修改文件清单

- `SKILL-GLOBAL-COMPONENTS.md` (+228 -12)
- `SKILL-TENANT.md` (+122 -0)
- `SKILL.md` (+109 -6)
- `client/src/App.tsx` (+2 -0)
- `client/src/components/MemoryPreview.tsx` (+71 -78)
- `client/src/components/Pagination.tsx` (+36 -1)
- `client/src/components/topnav/NotificationPanel.tsx` (+22 -69)
- `client/src/components/ui/back-button.tsx` (+3 -3)
- `client/src/components/ui/empty.tsx` (+33 -3)
- `client/src/components/ui/line-tabs.tsx` (+104 -0)
- `client/src/pages/DesignSystemComponents.tsx` (+1 -2)
- `client/src/pages/admin/AgentMigration.tsx` (+1 -1)
- `client/src/pages/admin/AgentToolLibrary.tsx` (+8 -25)
- `client/src/pages/admin/ApiDocs.tsx` (+1 -1)
- `client/src/pages/admin/AuditLog.tsx` (+6 -1)
- `client/src/pages/admin/BasicInfo.tsx` (+37 -7)
- `client/src/pages/admin/ChannelConfig.tsx` (+16 -42)
- `client/src/pages/admin/FileManagement.tsx` (+8 -9)
- `client/src/pages/admin/ImageManagement/UpdateRecordsDrawer.tsx` (+7 -10)
- `client/src/pages/admin/MemberManagement.tsx` (+105 -106)
- `client/src/pages/admin/MemberManagement/GroupDialog.tsx` (+5 -5)
- `client/src/pages/admin/MemberManagement/GroupList.tsx` (+9 -8)
- `client/src/pages/admin/MemberManagement/GroupView.tsx` (+16 -16)
- `client/src/pages/admin/MemberManagement/MemberDrawer.tsx` (+19 -40)
- `client/src/pages/admin/MemberManagement/NodeContentPanel.tsx` (+78 -83)
- `client/src/pages/admin/MemberManagement/OverrideCell.tsx` (+44 -49)
- `client/src/pages/admin/MemberManagement/SyncResultDialog.tsx` (+32 -48)
- `client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx` (+1 -1)
- `client/src/pages/admin/ModelConfig.tsx` (+3 -2)
- `client/src/pages/admin/PlatformPolicy.tsx` (+2 -2)
- `client/src/pages/admin/Security/AIAgent/Alarms/AlarmsList.tsx` (+4 -10)
- `client/src/pages/admin/Security/AIAgent/Assets/AgentAssetsList.tsx` (+3 -9)
- `client/src/pages/admin/Security/AIAgent/Assets/AssetDetail.tsx` (+1 -1)
- `client/src/pages/admin/Security/AIAgent/Common/CvmSelectComponent.tsx` (+2 -2)
- `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/BashPolicyList.tsx` (+1 -1)
- `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/PolicyDetailDrawer.tsx` (+4 -4)
- `client/src/pages/admin/Security/AIAgent/Groups/MaliciousPolicy/MaliciousPolicyList.tsx` (+1 -1)
- `client/src/pages/admin/Security/AIAgent/Groups/MaliciousPolicy/PolicyDetailDrawer.tsx` (+4 -4)
- `client/src/pages/admin/Security/AIAgent/Groups/NetGroupList.tsx` (+2 -2)
- `client/src/pages/admin/Security/AIAgent/Logs/index.tsx` (+7 -19)
- `client/src/pages/admin/Security/AIAgent/Skills/index.tsx` (+3 -11)
- `client/src/pages/admin/Security/index.tsx` (+9 -6)
- `client/src/pages/admin/SecurityGroupManagement.tsx` (+3 -4)
- `client/src/pages/admin/SkillConfig.tsx` (+17 -36)
- `client/src/pages/admin/SkillLibrary/AddToPackageDialog.tsx` (+4 -5)
- `client/src/pages/admin/SkillLibrary/MCPAddDialog.tsx` (+2 -2)
- `client/src/pages/admin/SkillLibrary/MCPDetail.tsx` (+0 -6)
- `client/src/pages/admin/SkillLibrary/PluginDetail.tsx` (+0 -12)
- `client/src/pages/admin/SkillLibrary/PublicSkillLibraryTab.tsx` (+11 -8)
- `client/src/pages/admin/SkillLibrary/PublicSkillPackageTab.tsx` (+13 -9)
- `client/src/pages/admin/SkillLibrary/PublicSkillTab.tsx` (+12 -9)
- `client/src/pages/admin/SkillLibrary/SkillDetail.tsx` (+0 -6)
- `client/src/pages/admin/SkillLibrary/SkillInitialPackageTab.tsx` (+31 -29)
- `client/src/pages/admin/SkillRolesTab.tsx` (+7 -11)
- `client/src/pages/admin/VersionManagement/CommandTaskTab.tsx` (+16 -6)
- `client/src/pages/admin/VersionManagement/HistoryTab.tsx` (+1 -1)
- `client/src/pages/admin/VersionManagement/components/DispatchCommandDialog.tsx` (+1 -1)
- `client/src/pages/admin/VersionManagement/components/RecentHistorySection.tsx` (+7 -4)
- `client/src/pages/admin/agentOps/AgentOpsPageHeader.tsx` (+1 -1)
- `client/src/pages/admin/agentOps/PageTabs.tsx` (+7 -47)
- `client/src/pages/admin/standard/StandardBasicInfo.tsx` (+177 -82)
- `client/src/pages/preview/EmptyStatePreview.tsx` (+714 -0)
- `client/src/pages/tenant/ChatView.tsx` (+3 -4)
- `client/src/pages/tenant/FileSpace.tsx` (+96 -82)
- `client/src/pages/tenant/HelpDocs.tsx` (+1 -1)
- `client/src/pages/tenant/ModelQuota.tsx` (+6 -6)
- `client/src/pages/tenant/MyOpenClaw.tsx` (+7 -7)
- `client/src/pages/tenant/OpenClawDetail.tsx` (+88 -70)
- `client/src/pages/tenant/OpenClawDetailGuide.tsx` (+79 -92)
- `client/src/pages/tenant/SkillSquare.tsx` (+53 -78)
- `client/src/pages/tenant/ToolsMcpPanel.tsx` (+35 -29)

---

## 📊 总体汇总

### PR 统计表

| # | 标题 | 状态 | 文件 | 改动 |
|---|------|------|------|------|
| #505 | chore: Portable Design Skill 最终优化 — 4.50... | MERGED | 2 | +467/-2 |
| #504 | chore: Portable Design Skill 完整评估与优化 (3.... | MERGED | 46 | +4601/-1055 |
| #500 | feat(design-system): 新增「典型页面样例」Tab，展示 7 ... | MERGED | 12 | +431/-29 |
| #499 | refactor(design-system): 删 Toggle / Butt... | MERGED | 16 | +895/-242 |
| #498 | feat(design-skill): StatusTag/Badge 收口 +... | MERGED | 33 | +2891/-549 |
| #497 | chore(skill): portable design skill 自包含化... | MERGED | 17 | +42/-5438 |
| #496 | feat(design-skill): 新增 7 类 page-referenc... | MERGED | 28 | +685/-82 |
| #495 | feat(design-skill): 沉淀 portable design s... | MERGED | 41 | +6061/-1768 |
| #493 | docs(skill): clawpro-portable-design — 3... | MERGED | 32 | +3494/-0 |
| #492 | feat(design-addietang): 新增 DatePicker/Al... | MERGED | 34 | +12408/-296 |
| #487 | fix(design): 设计走查修复 — 弹窗间距/边框色/label规范/组... | MERGED | 8 | +157/-463 |
| #485 | Feature/design refresh 2026... | OPEN | 100 | +11032/-0 |
| #484 | feat(design): 设计走查批量更新 — 用户端 & 管控端... | MERGED | 59 | +9843/-302 |
| #483 | feat(design-system): 扩充设计系统组件规范文档与预览页面... | MERGED | 32 | +7139/-490 |
| #482 | fix(design): 统一图表hover背景色、子网标题对齐、策略卡片底色t... | MERGED | 5 | +13/-15 |
| #481 | fix(design): 统一图表hover背景色、子网标题对齐、策略卡片底色t... | MERGED | 4 | +10/-15 |
| #480 | feat(design-sarry): 弹窗规范修复、筛选面板重构、网络管理展开... | MERGED | 83 | +7342/-3921 |
| #477 | docs(portable-skill): 补 NumberCard 可移植 s... | MERGED | 7 | +647/-45 |
| #476 | fix(ui): DesignSystemComponents 缺失 Numbe... | MERGED | 4 | +371/-44 |
| #473 | feat(design): 全局组件规范化 & 表格组件重构 & 多页面样式统一... | MERGED | 100 | +3532/-2423 |
| #470 | Design updates from feature/design-mieko... | MERGED | 95 | +39970/-45 |
| #466 | feat(design-skill): 新增开发使用说明及多组件 spec... | MERGED | 22 | +1614/-49 |
| #465 | Feature/design miekoyychen... | MERGED | 57 | +4039/-249 |
| #463 | Design updates... | MERGED | 34 | +3028/-0 |
| #462 | feat(design-skill): 添加 ClawPro 设计治理 Skil... | MERGED | 22 | +3476/-1500 |
| #461 | design: 设计走查批次更新（全局组件 + AIAgent/SkillLib... | MERGED | 71 | +2562/-1348 |

**合计**: 26 个PR | 964 文件 | +126750 -20370 行
