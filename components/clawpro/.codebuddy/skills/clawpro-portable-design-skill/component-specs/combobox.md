# Combobox

> ⚠️ **重要变更（2026-06-11）**：本文档描述的「Combobox」**在 demo 仓里已不存在独立组件**——
> 「带搜索的对象选择器」已并入 `Select` 组件，作为它的 `searchable` 模式（即 `SearchableSelect`）暴露。
>
> - 真实导出：`import { SearchableSelect } from "@/components/ui/select"`
> - 没有 `Combobox` 组件、没有 `OpenClawCombobox`、没有 `ui/combobox.tsx` 文件。
> - 现存代码里 21 处调用全部是 `SearchableSelect`（如 `pages/admin/TokensMonitor.tsx`、`OpenClawMonitor.tsx`、`SkillLibrary/BatchDistributeDialog.tsx` 等）。
>
> 本文档保留作为 **alias / migration guide**，不要再当作独立组件 spec。新写代码请直接看：
> 1. **入口规范**：`component-specs/input-select.md`
> 2. **真实实现**：`client/src/components/ui/select.tsx` `SearchableSelect` 一节
> 3. **本文 §3 视觉标准 / §X 决策口径** 仍然适用（视觉口径没改，只是组件名变了）

---

## 1. Purpose（仍适用）

- 统一带搜索能力的对象选择器，避免页面里临时拼 `Button + Popover + Input + List` 后出现高度、边框、空态和选中态不一致。
- 这类组件在宿主仓通常已有选择逻辑，portable spec 的重点是把触发器、搜索区、浮层、选中态和清空语义收口。

## 2. Scope（仍适用）

- 适用端：Admin / Shared 优先；Tenant 的列表筛选、对象选择、弹窗表单可复用。
- 必用场景：候选项较多、需要搜索过滤、候选项是 Agent / 部门 / 分组 / 标签 / VPC / 业务对象。
- 不适用场景：少量静态枚举用 `Select`（不带 searchable）；日期时间用 DatePicker；纯文本输入用 Input；多选矩阵或跨页远程对象选择优先用 Dialog / Drawer。

## 3. Visual Standard（仍适用）

| Item | Default | Notes |
|---|---|---|
| Trigger Height | 36px / `h-9` | 与 Input / Select / DatePicker / Button 对齐 |
| Trigger Radius | Admin 4px；Tenant 筛选场景可胶囊，普通表单 4px | 跟随 `form-controls.md` 双轨口径 |
| Trigger Border | `var(--cp-border)`；hover / focus / open 切品牌蓝弱强调 | 与 Input / Select 同口径 |
| Trigger Text | 已选值 `var(--cp-text-title)`，placeholder `var(--cp-text-weak)` | 「全部」不要和真实已选值混淆 |
| Chevron | 16px `var(--cp-text-weak)` | open 时可旋转，不抢视觉 |
| Clear Action | 右侧 `X`，必须 `e.stopPropagation()` | 避免清空时同时打开面板 |
| Panel | 白底、4px、蓝灰描边、`var(--cp-shadow-overlay)` | 通过 Portal 逃逸 overflow |
| Panel Width | 默认跟随 trigger 宽度 | 树形面板可写死 280px，但要有业务原因 |
| Search 顶部 | Search icon + Input；`placeholder` 走 weak | 留 search border 与 trigger 视觉区隔 |
| Option | 32px 行高，文本 truncate；selected 用品牌蓝 `var(--cp-text-brand)` + `var(--cp-brand-tint)` | hover 走 `var(--bg-grey-normal)`；selected 业务态走 `var(--bg-grey-hover)` |
| Empty | 单行弱提示，不使用页面级 Empty 插画 | 见 `empty-state.md` Combobox/搜索下拉行 |

> Token 隔离规则（`claw / shared` vs `tenant-*` 不强制对齐 token）的口径与 `button.md §3.1` 对齐，本次合并到 SearchableSelect 后**没有变化**——同一个组件同时服务两端，差异通过外部 className / 业务策略表达。

## 4. Real Demo Repo Usage

```tsx
import { SearchableSelect } from "@/components/ui/select";

<SearchableSelect
  options={[
    { value: "all", label: "全部 Agent" },
    { value: "ag-1", label: "Agent 1", searchText: "agent 1 主力" },
  ]}
  value={agentId}
  onChange={setAgentId}
  placeholder="请选择 Agent"
  searchable                       // 默认 true，可省略
  searchPlaceholder="搜索 Agent..."
  showCount                        // 默认 true，底部"共 N 条"
  align="start"
  clearable                         // 显示右侧 X 清除
/>
```

完整 props 见 `client/src/components/ui/select.tsx` 的 `SearchableSelectProps`：
`options / value / onChange / placeholder / searchable / searchPlaceholder /
showCount / countTemplate / triggerClassName / panelWidth / align / disabled / clearable`。

### 4.1 与基础 Select 的边界

| 场景 | 选 | 原因 |
|---|---|---|
| 选项 ≤ 8 条、不需要搜索 | `Select`（同文件，不带 `searchable`） | 浮层用 Radix，体验最轻 |
| 选项较多 / 用户预期能搜 | `SearchableSelect` | 内置搜索 + 计数 + truncate tooltip |
| 多选 + 选中即关闭 | `SearchableSelect` 不支持，改用 `MultiSearchableSelect`（同文件 §二） | 见 `select.tsx` |
| 跨页 / 远程加载 / 大量数据 | 不要用 SearchableSelect，走 Dialog / Drawer + 自定义列表 | 浮层不适合长列表 |
| 树形选择 | 不要用 SearchableSelect，走 `tree.md` 的 `TreeSelect` 派生 | |

## 5. Migration Rules

1. **不要再把 "Combobox" 当组件名写到代码里** —— 直接 `SearchableSelect`。
2. **旧 import 写法**（如果 git 历史里出现过）：
   ```ts
   // ❌ 已删除 / 从未存在
   import { Combobox } from "@/components/ui/combobox";
   import { OpenClawCombobox } from "@/components/OpenClawCombobox";

   // ✅ 唯一正确写法
   import { SearchableSelect } from "@/components/ui/select";
   ```
3. **PR review 反模式**（任一命中即退回）：
   - 在新代码里 `import` 任何带 `Combobox` 字样的标识符
   - 自拼 `<Popover><PopoverTrigger><Button role="combobox">…<Input>…<ListBox></PopoverContent>` 替代 SearchableSelect
   - 复用 `Select` 的 `<SelectTrigger>` 外面再包 search input

## 6. 文档侧待清理项（已知）

> 以下文件仍在用旧名「Combobox」字面量提及，**不再描述独立组件**，仅作为对 SearchableSelect 的口语化别称保留：
>
> - `SKILL.md`（关键词列 / §6 表 / §9 高风险组件表）
> - `references/components.md`（outline 借用例外）
> - `component-specs/input-select.md` §1 头部 mapping、§Scope
> - `component-specs/popover-dropdown-menu.md`（"长列表 + 搜索用 Combobox"提示）
> - `component-specs/search-filter-bar.md` / `transfer.md` / `empty-state.md`
> - `INDEX.md` / `README.md` / `HANDOFF.md` 的关键词列
>
> AI 在本轮 spec 收口中已把上述位置的 "Combobox" 替换为 "SearchableSelect（旧称 Combobox）" 或纯 SearchableSelect。如果你在 PR 里又看到旧名字漂回来，按上面 §5 处理。

## 7. Portable Fallback

> 已合并到 SearchableSelect，**不再单独提供 portable/react/combobox.tsx**。
>
> - 宿主仓如果需要"带搜索的对象选择器"的兜底实现，建议直接抄 `select.tsx` `SearchableSelect`（依赖 Radix Popover + Input + Lucide），自行裁剪到宿主仓。
> - 若宿主仓不能引 Radix，最小依赖路径是：
>   1. 用 `portable/react/input-select.tsx` 的 `PortableSelect` 处理"无搜索"场景
>   2. 在调用方再外挂一个轻量搜索框 + 自绘 dropdown（不在本 portable 包内）
> - 不再生成 `portable/react/combobox.tsx`，避免它和 `SearchableSelect` 长期双轨。
