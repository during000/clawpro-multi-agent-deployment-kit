# 设计走查 / 换皮 待办清单

> 维护：`feature/design-jingsujiang` 分支换皮过程中沉淀的"暂时绕过 / 后续治理"项。
>
> - 每条都有 **状态 / 影响面 / 临时方案 / 终极方案 / 触发上报的页面或文件**。
> - 完成后请把对应条目标记为 ✅ 并附 commit hash。
> - 与 `clawpro-portable-design-skill/references/conflict-log.md` 配合使用：
>   - 本文件记 **可执行的代办**（"以后要做"）
>   - conflict-log 记 **决策原因**（"为什么这么做"）

---

## P0 · 阻塞类规范缺失（立项治理）

> 这些是"组件库 / 规范本身缺失"，目前用临时方案绕过，会持续产生违规代码。

### TODO-001 · Alert 缺 `success` variant

- **状态**：⏳ 待治理（影响面：全栈）
- **触发**：`OpenClawDetailGuide.tsx` 迁移弹窗"已检测到上传的数据包"提示需要绿色成功 Alert，但 `Alert` 组件的 `variant` 只有 `default / info / operation-info / warning / product-news / destructive` 6 档，**没有 `success`**。
- **临时方案**：当前走 `variant="info"`（蓝色），语义略弱但合规。
- **终极方案**：在 `client/src/components/ui/alert.tsx` 增加 `success` variant：
  - `border-[var(--alert-success-border)]`
  - `bg-[var(--alert-success-bg)]`
  - `text-[var(--alert-success-foreground)]`
  - `[&>svg]:text-[var(--alert-success-icon)]`
  - 在 `index.css` 同步定义 4 个 `--alert-success-*` token（参考现有 `--alert-warning-*` 的写法）
- **影响面**：全用户端 / 管理端，所有"成功完成"语义提示都将受益。
- **优先级**：P1（建议下个 sprint 立项）。

### TODO-002 · cp-* 系列 token 在本仓未定义但被业务页大量引用

- **状态**：⏳ 待治理（影响面：全栈）
- **触发**：业务代码出现 `bg-[var(--cp-brand-blue-soft,#EFF6FF)]` `bg-[var(--cp-success-soft,oklch(...))]` 等带 fallback 的 var() 写法，但 `index.css` 里**没有真正定义** `--cp-*` 系列。这是 portable skill 的虚拟 token，本仓用的是 `--text-brand` `--text-success` 等真名。
- **临时方案**：保留 var() 的 fallback 值（带逗号的第二参数），使其在 token 未定义时仍能兜底显示。
- **终极方案**（二选一）：
  - **A. 在 `index.css` 显式映射** `--cp-brand-blue-soft: #EFF6FF;` `--cp-success-soft: oklch(...);` 等真 token。
  - **B. 全局批量替换** `--cp-brand-blue-soft` → 自定义本仓 token 名（如 `--surface-brand-soft`）。
- **影响面**：OpenClawDetailGuide / SkillSquare / FileSpace / ToolsMcpPanel 等多页。
- **优先级**：P2（不阻塞功能，但代码风格不一致）。

### TODO-003 · DialogBody 默认 `py-2` 在无 Footer 时下边距过小

- **状态**：⏳ 待治理（已多次手工绕过）
- **触发**：`SkillSquare` 详情弹窗 / `OpenClawDetailGuide` 迁移弹窗等"无操作按钮"的查看类弹窗，下边距只有 8px（DialogBody py-2 的下半），视觉局促。
- **临时方案**：每处手动 `<DialogBody className="pb-6">`。
- **终极方案**（二选一）：
  - **A. 给 `<DialogContent>` 加 `variant="readonly"` / `noFooter` prop**，自动补足下间距。
  - **B. 修改 `<DialogBody>` 默认 padding** 到 `py-3` / `pb-4`，让"无 Footer"也能视觉舒适。
- **影响面**：所有"查看类 / 无操作"弹窗。
- **优先级**：P2。

### TODO-004 · DIN 字体在 macOS 仅 Bold 字重，semibold 降级 PingFang

- **状态**：⏳ 待治理（已记入 conflict-log）
- **触发**：`Typography.StatNumber` 用 `font-din` + `font-semibold(600)`，但系统 DIN Alternate 只有 `Bold`，浏览器找不到 600 → 降级 PingFang。
- **临时方案**：保留 `font-semibold`，接受降级。
- **终极方案**（3 选 1）：
  - **A. self-host TCloud Number 字体**，覆盖 `--font-din`。
  - **B. 改用 `font-bold(700)`**，统计数字会更粗，需要设计确认。
  - **C. 接受 PingFang 降级**，删除 `font-din` 类。
- **影响面**：ModelQuota 5 张统计卡 / 全站 StatNumber 用法。
- **优先级**：P1（视觉一致性强相关）。

---

## P1 · 局部违规但功能正常（待统一）

### TODO-005 · 字母头像装饰色（LETTER_COLORS）26 配色

- **状态**：✅ 已记 conflict-log（保留）
- **触发**：`SkillSquare.tsx` 字母头像 26 色 `getLetterColor()`，全是 hex（#E8F4FD #1A73E8 等）。
- **临时方案**：保留装饰色，token 体系不覆盖。
- **终极方案**：等设计统一发布"装饰色板"token 后再批量映射，**当前不动**。
- **影响面**：仅 SkillSquare。
- **优先级**：P3（不影响功能，等设计令牌发布）。

### TODO-006 · `ui/* 底座硬编码`（borders, hover 颜色等）

- **状态**：⏳ 待治理
- **触发**：
  - `ui/input.tsx` L71-72：`hover:border-[#355EF1]` `focus:border-[#355EF1]` 是硬编码（应是 `--text-brand`）
  - `ui/status-tag.tsx`：26 色彩 token 全 hex
  - `ui/dialog.tsx` L161：`text-[#7b818f]` 关闭按钮色
- **临时方案**：换皮不动底座，记录待治理。
- **终极方案**：底座单独排期 token 化。
- **影响面**：全站。
- **优先级**：P2。

### TODO-007 · OpenClawDetailGuide "安装新技能" 弹窗自定义结构

- **状态**：⏳ 待治理
- **触发**：`OpenClawDetailGuide.tsx` L649：`<DialogContent className="sm:max-w-[780px] p-0 overflow-hidden" showCloseButton={false}>` 整体绕过规范，自定义 Header（自定义关闭按钮）+ p-0 内边距。
- **临时方案**：保留现状（视觉效果是产品确认过的）。
- **终极方案**：等"复杂列表型弹窗"规范出台再统一。可能需要新增 `<DialogShell variant="custom">` 形态。
- **影响面**：仅 OpenClawDetailGuide。
- **优先级**：P3。

### TODO-008 · Empty 组件 / EmptyMedia 需统一兔子插画

- **状态**：✅ 已修复（保留作为参考）
- **触发**：之前 SkillSquare 改造时 EmptyMedia 自定义传入 SVG，违反 §24 规范。
- **修复**：删 children，用 `<EmptyMedia />` 默认兔子插画。
- **影响面**：全站。
- **完成 commit**：`cb88c294`（feat(tenant/skill-square)）。

---

## P2 · 视觉细节 / 局部 token 缺位

### TODO-012 · 迁移弹窗"重要提醒" Alert 语义复审

- **状态**：⏳ 待设计确认
- **触发**：`OpenClawDetailGuide.tsx` Agent 迁移弹窗 Step 2 的"重要提醒"使用 `Alert variant="destructive"`（红色）。
  - 文案语义是「操作前预警」（导入将覆盖配置，自动备份/回滚），不是"已失败"。
  - 按 skill 严格语义应该是 `variant="warning"`（橙色），但同弹窗上方"注意事项"已用 warning，连续两个橙框会视觉重复。
- **临时方案**：保留 `destructive` 红色做差异化。
- **终极方案**（3 选 1）：
  - **A.** 改 `warning`，接受两个橙框堆叠（语义最准，视觉略平）。
  - **B.** 上方"注意事项" 改 `info`，下方"重要提醒"改 `warning`（语义重新分配）。
  - **C.** 保留现状（已差异化最强）。
- **影响面**：仅迁移弹窗 1 处。
- **优先级**：P2（设计/产品复审）。

### TODO-009 · `bg-amber-50` 在 importing/verify 步骤底色

- **状态**：⏳ 待治理
- **触发**：`OpenClawDetailGuide.tsx` L2765 `bg-amber-50` 是 tailwind 颜色，无 token。
- **临时方案**：保留（与同区域 `cp-success-soft` 风格一致，等 cp-* 系列治理一并处理）。
- **终极方案**：随 TODO-002 一并解决。
- **影响面**：仅 1 处。

### TODO-010 · SyntaxHighlighter customStyle 行号色 `#A3A3A3`

- **状态**：✅ 装饰色保留
- **触发**：`SkillSquare.tsx` L1368/L1400 `lineNumberStyle={{ color: '#A3A3A3' }}` 通过第三方组件 props 传入，无法 var()。
- **临时方案**：保留硬编码，归类为代码高亮装饰色。
- **终极方案**：第三方组件限制，无解。
- **影响面**：仅 SkillSquare 文件查看。

---

### TODO-012 · 胶囊单选（PillRadio）规范缺失，需抽公共组件

- **状态**：⏳ 待治理（多页有需求）
- **触发**：MyOpenClaw 创建弹窗"Agent 类型 / 角色身份"是 5/6 项的胶囊式 RadioGroupItem，每处需 13 行 className 长链（border + bg-white + token + 选中态多个 peer-data-[state=checked] 覆写），写一次错一次。
- **临时方案**：MyOpenClaw 内部定义 `PillRadioOption` 组件，仅本页可用。
- **终极方案**：抽到 `client/src/components/ui/pill-radio-option.tsx`（或并入 `radio-group.tsx`），新增 spec `component-specs/pill-radio.md`：
  - 默认：`h-6 rounded-full border-[var(--border)] bg-[var(--card)] text-[var(--text-emphasis)]`
  - 选中：`bg-[var(--text-emphasis)] text-white border-[var(--text-emphasis)]`
  - hover：`bg-[var(--accent)] border-[var(--border-control)]`
- **影响面**：MyOpenClaw 类型/角色（已用本地组件）、ToolsMcpPanel、其他可能的"标签筛选/类型选择器"页面。
- **优先级**：P1（下个 sprint 立项时直接抽出来）。

### TODO-013 · 页面级 keyframes 内联 `<style>` 应迁到 globals.css

- **状态**：⏳ 观察
- **触发**：MyOpenClaw 底部 `<style>{` @keyframes pulse `}</style>` 内联（HeroBanner 等抽出组件可能有同名定义重复），各页面散落。
- **临时方案**：保持局部内联，作为页面装饰动画。
- **终极方案**：把通用动画 keyframes 集中到 `client/src/index.css` 或 `client/src/styles/animations.css`，页面只用 `animation: ...` className/utility。
- **影响面**：MyOpenClaw + 任何含手写 @keyframes 的页面。
- **优先级**：P3。

---

## P3 · 优化建议（不阻塞）

### TODO-011 · ResetPassword 路由删除后兜底链接处理

- **状态**：⏳ 观察
- **触发**：`/reset-password` 路由已删除，但若用户从书签或外部链接访问，会触发 wouter 默认 404。
- **临时方案**：当前依赖 wouter 全局兜底页面（如有）。
- **终极方案**：在 App.tsx 加一条 `<Route path="/reset-password" component={NotFound} />` 或重定向到首页 + Toast 提示"此功能已合并到右上角菜单"。
- **影响面**：极小（旧链接持有者）。
- **优先级**：P3。

---

## 历史已完成（仅供回溯）

- ✅ ResetPassword 整页删除 → 复用 TenantLayout 弹窗（commit 待提交）
- ✅ SkillSquare 详情弹窗 size="lg" + DialogBody pb-6（commit `cb88c294`）
- ✅ Agent 迁移弹窗 size="lg" + DialogBody pb-6 + Alert 删硬覆盖
- ✅ TenantLayout 重置密码弹窗 token 化 + DialogBody 三段式
- ✅ MyOpenClaw / OpenClawDetailGuide / ModelQuota / FileSpace / ToolsMcpPanel / GroupChangeComponents 全 token 化（commit `67c215a2`）
- ✅ 新增 clawpro-page-refactor 7 步工作流 skill（commit `92980fae`）

- ✅ NotificationPanel 顶栏抽屉 token 化 + Empty 兔子插画 + Tabs→TenantSegment（含 segment.tsx active 边线 outline→border 全局升级，影响 8 处引用）
- ✅ MyOpenClaw 完整换皮：Empty 规范化 / 5 处 PillRadioOption 抽出 / SectionTitle / 复制按钮 ghost / Separator / 内联 style 清零 / 0 处硬编码颜色

---

**最近更新**：2026-06-10  
**维护人**：jingsujiang
