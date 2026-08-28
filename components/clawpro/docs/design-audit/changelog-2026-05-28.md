# 设计走查变更记录 · 2026-05-28（addietang）

> 本次会话由 CodeBuddy 完成，主题：Table 组件标准化 + 全局字号/字体/字色统一 + 全局自动固定列。

## 1. Table 组件原生 → 标准 Table 组件改造（5 文件，共 9 处）

- `client/src/pages/admin/DocManagement.tsx` — 文档列表 1 处
- `client/src/pages/admin/VersionManagement/CommandTaskTab.tsx` — 命令库 1 处（清理 DropdownMenu，操作平铺）
- `client/src/pages/admin/ServerManagement.tsx` — 镜像 / 入站 / 出站 共 3 处
- `client/src/pages/admin/ApiDocs.tsx` — 请求 Header / Token 类型 / 参数类型 共 3 处
- `client/src/pages/admin/VersionManagement/components/DispatchCommandDialog.tsx` — Step2 实例选择 1 处

替换收益：所有表格统一规范、TableActionCell 操作列、StatusTag 状态标签、Badge 状态徽章、density="compact" 紧凑表格。

## 2. 表格组件设计规范升级（v2026.05）

`client/src/components/ui/table.tsx` 顶部规范文档系统化重写为 8 节，
作为表格组件的"权威标准"，对应规则落点见 `client/src/index.css`。

### §1 密度
- default 行高 54px / compact 行高 40px。
- 两种密度只在行高 / 纵向 padding 上区分，字号 / 横向 padding / 颜色完全一致。

### §2 字号一致性（!important 全局强制 12px）
覆盖范围：
1. 表格单元格自身：`[data-slot="table" | "table-head" | "table-cell" | "table-action-cell" | "table-footer"]`
2. 表格内任意后代元素：`table[data-density] *`（按钮 / Input / Select / Switch / Checkbox / Label / Tooltip / code / pre / div / span / p / a ...）
3. 分页器：`[data-slot="pagination"]`（无论 size="default" 还是 "small"，字号一致仅高度不同）
4. 数量统计 / 摘要文字：SurfaceCard 内、与 table-container 同级的兄弟元素

唯一豁免：`[data-slot="badge"]`。

### §3 字色规范
- TableHead：default → #171717；compact → #737373
- TableCell / TableActionCell：默认强制 #0A0A0A（纯黑），不再按"主/次列"切灰。

### §4 字体（PingFang SC）
全局规则：
```css
*:not(svg):not(svg *) {
  font-family: 'PingFang SC', -apple-system, BlinkMacSystemFont, 'Helvetica Neue', sans-serif !important;
}
```
inline style fontFamily / Tailwind font-mono / font-din 全部强制覆盖为 PingFang SC。
仅 SVG `<text>` 等保留自身定义。

### §5 操作列 TableActionCell
所有操作按钮（含「删除」等危险操作）统一 `<Button variant="link">` 蓝色，
不再以红/黑区分语义，由文案 + 二次确认 Dialog 承载。
内置 flex wrapper：项间距固定 24px。

### §6 固定列（v2026.05 修订）
- `<Table>` 默认 `scrollX={undefined}` → 表格按容器宽度自适应，内容放得下时**不出现横滚条**（避免简单列表也强制横滚）。
- `<Table>` 默认 `autoFixedColumns={true}` → 列数较多触发横滚时，自动 sticky 首列（th/td:first-child）+ 操作列（TableActionCell）。
- 列数较多 / 内容长度不可控的表格请显式传 `scrollX={1500}` 或 `scrollX="max-content"` 启用横滚兜底。
- 业务侧已显式声明 `fixed="left"/"right"` 的列优先级更高，不被覆盖。
- 关闭自动固定列：`<Table autoFixedColumns={false}>`。

### §7 选中行 data-state
通过 `<TableRow data-state="selected">` 标记选中态，全行（含固定列）背景自动变色。

### §8 Pagination 搭配
推荐结构：
```tsx
<SurfaceCard>
  <Table>...</Table>
  <div className="px-4 py-3 border-t border-[#f0f0f0]">
    <Pagination total={...} showTotal={(t) => `共 ${t} 条记录`} ... />
  </div>
</SurfaceCard>
```

## 3. Pagination 组件改动

`client/src/components/ui/pagination.tsx`：
- `textSize` 由 `isSmall ? "text-xs" : "text-sm"` 改为统一 `"text-xs"`（12px）
- `<nav>` 加 `data-slot="pagination"` 标识
- size="default" / "small" 仅按钮高度（32px / 24px）不同，字号一致

## 4. ScopeEditPopover / ImageScopePopover Badge 风格统一

应用范围列的徽章统一为 `<Badge variant="outline">`（白底 #E5E5E5 描边 #0A0A0A 字）：
- `client/src/components/ScopeEditPopover.tsx` — 通用应用范围编辑组件（A公司 / +N / 未选组织）
- `client/src/pages/admin/ImageManagement.tsx` — ImageScopePopover renderScopeText（全部用户 / 组织名 / +N）
- 与 `AllUsersTag`（已是 outline）三处行为完全一致

## 5. TokensMonitor 数字列改黑

`client/src/pages/admin/TokensMonitor.tsx` 5 个 tab（按实例 / 按用户 / 按模型 / 按部门 / 按组织 / 按会话）：
- 数字列移除 `text-sm text-[#737373]` → 默认（继承黑色 + 12px）
- "用户 ID"列加 `font-medium`
- 总 Tokens 列简化为 `font-medium`

## 6. CSS 全局规则新增（client/src/index.css）

### 表格字号一致性规则（4 段）
- 表格单元格自身 12px
- 表格内任意后代元素 12px
- 分页器 12px
- 数量统计 / 摘要文字 12px

### 表格自动固定列规则（v2026.05 新规）
- `table[data-auto-fixed="true"]` 自动 sticky `th:first-child` / `td:first-child` / `[data-slot="table-action-cell"]`
- 仅作用于未显式声明 `[data-fixed]` 的单元格
- 行 hover / data-state="selected" 状态自动同步固定列底色
- 横向滚动时自动显示 1px 边界分隔线

### 全站字体强制规则
- `*:not(svg):not(svg *)` 强制 PingFang SC

## 7. 修改文件清单（25 个）

详见 `changed-files-2026-05-28.txt`。

核心组件：
- `client/src/components/ui/table.tsx`（规范 + autoFixedColumns + scrollX 默认值）
- `client/src/components/ui/pagination.tsx`（统一 12px + data-slot="pagination"）
- `client/src/components/ScopeEditPopover.tsx`（Badge variant="outline"）
- `client/src/index.css`（全局字号 / 字体 / 自动固定列规则）

业务页面：21 个文件 — 主要为旧 `<table>` 替换 + 视觉细节统一。

## 8. NodeContentPanel 配置总览 Tab 标签替换

`client/src/pages/admin/MemberManagement/NodeContentPanel.tsx` 的"配置总览"二级 Tab（11 个子 tab：模型 / 通道 / 技能 / 工具 / 记忆 / 网盘 / 镜像 / 网络 / 日志 / 安全 / 策略）下，所有手写 inline 胶囊全部替换为标准组件：

| 原实现 | 新实现 | 影响范围 |
|---|---|---|
| `LocalAnomalyHint`（红色 inline 胶囊） | `<StatusTag mode="dot" variant="red">` | 11 个子 tab 通用 |
| `ConfigOutdatedHint`（黄色 inline 胶囊） | `<Badge variant="outline">` + amber tokens（与 Alert `variant="warning"` 对齐：#FFF7ED / #FED7AA / #FF6900） | 网络 tab |
| `PolicyEntryValue`「已开启/已关闭」 | `<StatusTag mode="dot" variant="green/gray">` | 策略 tab |
| `PublicNetworkDetail`「已分配/未分配」 | `<StatusTag mode="text" variant="green/gray">` | 网络 tab |
| 子网可用区灰色 chip（3 处 inline `bg-gray-100`） | `<Badge variant="secondary">` | 网络 tab（preset / 普通 / zonesAllDeleted） |

至此 NodeContentPanel.tsx 配置总览 Tab 不再有任何手写 chip / 胶囊样式，全部走标准组件。

## 9. SkillListTab 卡片视图重构（5/29 补做）

`client/src/pages/admin/SkillLibrary/SkillListTab.tsx` 卡片视图全部手写 chip / 按钮重写为标准组件，恢复设计走查丢失版本：

| 区块 | 原实现 | 新实现 |
|---|---|---|
| 版本号 | `inline-block px-2.5 py-0.5 bg-gray-100 ...rounded-full` 灰色胶囊 | 右上角 mono 文字 `v1.0.0`（无背景） |
| 分类标签 | 手写灰色 chip ×N + +N overflow chip | `<Badge variant="outline">` ×N + +N overflow Badge |
| 应用范围 | 「应用范围：」前缀 + 旧 EditScopePopover | 「应用范围」无冒号弱化文案 + EditScopePopover（自带 outline Badge） |
| 下发按钮 | `variant="outline" size="sm"` 矮按钮（h-7） | `variant="claw-primary" size="sm"` 主按钮（h-8） |
| 更新 / 更多 | `variant="outline"` h-7 | `variant="claw-outline"` h-8 |
| 删除项 | `text-red-600 focus:text-red-600` + 红色 icon | 默认色（与 TableActionCell 规范一致，语义由二次确认 Dialog 承载） |
| 卡片头部字重 | `font-semibold` | `font-medium`（收敛与表格一致） |
| 卡片 hover | `hover:bg-gray-50` | `hover:bg-[#FAFAFA] hover:border-[#D4D4D4]`（Tea 令牌） |
| 操作区分隔 | 无 | `border-t border-[#F5F5F5] pt-3` 分隔线 |

## 10. 本次会话提交记录

| Commit | 说明 |
|---|---|
| `0488b68` | style(admin): table 组件规范升级 + 全局字号字体字色统一 + 自动固定列 |
| `b9baa03` | fix(ui/table): 还原 scrollX 默认值，避免简单列表表格强制横滚 |
| `c443480` | fix(admin/image): Agent 类型表「应用范围」列改用 Badge variant=outline |
| `dc9f06d` | docs(design-audit): 补齐本次会话 commit 记录与 PR #358 链接到 changelog |
| `52520a4` | style(admin/member): NodeContentPanel 配置总览 Tab 标签全部替换为标准组件 |
| `2f6adc6` | docs(design-audit): 补齐 NodeContentPanel 配置总览 Tab 标签替换记录到 changelog |
| `c14d3b6` | style(admin/skill-library): SkillListTab 卡片视图重构 — 标准化所有标签与按钮 |

## 11. PR 链接

https://github.com/tx-organization/openclaw-enterprise/pull/358

- base：`main`
- head：`feature/design-addietang`
- 状态：已创建，等待 review

## 12. 5/29 二次走查微调

### 12.1 SkillListTab 卡片视图「下发」按钮降级

`client/src/pages/admin/SkillLibrary/SkillListTab.tsx` ≈ L1015-L1026

| 项 | 改前 | 改后 |
|---|---|---|
| 「下发」按钮 variant | `claw-primary`（深色主按钮） | `claw-outline`（次级按钮） |

**原因**：卡片视图整列卡片每张都带一个深色主按钮，视觉过于抢眼且破坏卡片网格的克制基调。同行的「更新」「更多」均为 outline 次级按钮，「下发」改为同级 outline 后视觉一致，主操作语义由 icon（Send）+ 顺位（最左）承载。

### 12.2 ChannelConfig 自定义通道指引外链 hover 加深

`client/src/pages/admin/ChannelConfig.tsx` ≈ L366-L370

| 状态 | 改前 | 改后 |
|---|---|---|
| default | `text-[#355EF1]` | `text-[#355EF1]`（不变） |
| hover | `hover:text-[#355EF1]`（无变化，反馈不足） | `hover:text-[#0a226f]`（与 `Button variant="link"` 的 active 色对齐） |

**原因**：原 hover 与 default 同色 `#355EF1`，鼠标悬停时颜色不变化、反馈不足；新色 `#0a226f` 是项目 link 系统的标准 active 色，hover 时颜色加深，符合"hover 应有反馈但不抢戏"的链接交互规范。

### 12.3 本次提交

| Commit | 说明 |
|---|---|
| `f6baebf` | style(admin): SkillListTab 卡片下发按钮降级为次级 + ChannelConfig 外链 hover 加深 |
| `b4ebf05` | docs(design-audit): 补齐 5/29 二次走查微调记录到 changelog |


## 13. 5/29 三次走查微调（与 claude-internal 协作期间）

### 13.1 Table 默认表头反转：灰底 → 白底

`client/src/components/ui/table.tsx`

| 项 | 改前 | 改后 |
|---|---|---|
| `variant="default"` 表头背景 | `bg-gray-50`（#F9FAFB 灰底） | `bg-white`（白底） |
| `variant="elevated-white"` 表头 | 白底 | 白底（保持） |
| 新增 `variant="gray-header"` | — | `bg-gray-50` 灰底，用于需要表头与内容明显区分的场景 |
| 固定列背景（左/右） | `default`/`elevated-white` 各自处理 | `gray-header` 沿用 `FIXED_LEFT_CLS`/`FIXED_RIGHT_CLS`；其余 variant 统一白底固定列 |

**原因**：设计走查决定主表格默认走"白底极简"风格，与卡片化容器（`bg-white rounded-[4px] border`）天然融合，减少灰阶层级。需要灰底分隔的场景（如长列表、深嵌套面板）显式指定 `variant="gray-header"`。

**协作提示**：claude-internal 那边若原本写了灰底表头（默认 `variant`），现在会自动变白；如设计意图仍是灰底，需显式改为 `variant="gray-header"`。

### 13.2 NodeContentPanel 节点用户表底部字号统一 12px

`client/src/pages/admin/MemberManagement/NodeContentPanel.tsx` ≈ L790

| 项 | 改前 | 改后 |
|---|---|---|
| 「共 N 名用户」font-size | `text-sm`（14px） | `text-xs`（12px） |
| line-height | `leading-[1.5]`（21px） | `leading-[18px]` |
| color | `text-[#737373]` | `text-[#737373]`（不变） |

**原因**：与表格内字号 12px、分页器字号 12px 统一，遵守"表格生态全栈 12px"规范（见 § 1-3）。

### 13.3 本次提交

| Commit | 说明 |
|---|---|
| `1224f09` | style(ui/table+admin/member): 表头默认改白底 + 节点用户表底部字号 12px |

### 13.4 给 claude-internal 的同步指令

```bash
git fetch origin feature/design-addietang
git pull --rebase origin feature/design-addietang
# 如本地有改动冲突：
# git stash → git pull --rebase → git stash pop
```

重点核对：原本写 `<Table>` 期望灰底表头的场景，需显式补 `variant="gray-header"`。

