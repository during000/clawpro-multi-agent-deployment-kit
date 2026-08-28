# 用户管理 & 应用范围 PRD

> 版本：v2.0（重大重构）
> 日期：2026-04-27
> 负责：OpenClaw Enterprise 管控端
> 状态：决策确认完毕，进入实施

---

## 变更说明（v1.0 → v2.0）

v1.0 引入了独立的「部门视图」和 `Department` 表，本版本做出如下**结构性重构**，v1.0 相关设计作废：

| 方面 | v1.0 | v2.0 |
|---|---|---|
| 用户管理页 tab | 全部 / **部门** / 组织 | 全部 / 组织（**部门 tab 整体下掉**） |
| 数据表 | `Department` 与 `UserGroup` 两张独立表 | **只保留 `UserGroup`**（`Department` 合并进来） |
| 组织层级 | 单层 | **多层级树（无深度限制）** |
| 单用户归属 | 只能加入一个组织 | **可加入多个组织 / 组织架构 / 用户组** |
| 组织来源 | 普通/OneID 均为管理员自建 | 普通=管理员自建；OneID=**同步 OneID 的组织架构 + 用户组**（ClawPro 不再自建） |
| `Scope.filtered` 字段 | `groupIds[] + deptIds[]` | **只保留 `groupIds[]`**（组织架构以 UserGroup 记录形式存在） |
| OneID 同步策略 | 部门自动同步 | 用户组自动同步；**组织架构需管理员显式勾选同步**，已同步节点只读 |
| 配置 Tab | 组织/部门节点下附属 | 仅组织节点有（并列「成员列表」Tab） |
| 用户端 | 不做 | **规划本期后续迭代**；本期只做管控端 |

---

## 0. 背景与目标

### 0.1 现状问题

1. **无统一的组织架构视图**：用户数据散落在各资源配置页，缺少"按组织结构查看谁有什么"的能力。
2. **"应用范围"术语与交互不统一**：模型/技能已有 `ScopePopover`，通道/平台策略的"按组织"被灰掉，Agent 工具库无应用范围，镜像/VPC/安全组/EIP/记忆/网盘是"企业单例"模式。
3. **组织能力受限**：单层 + 单归属，无法表达真实企业的组织架构与跨团队协作。
4. **无法对接 OneID**：当前无法处理真实企业的组织架构、用户组、主部门/兼任等场景。
5. **无"最终生效配置"透视**：管理员无法快速判断某用户实际拥有哪些资源，及其覆盖来源。

### 0.2 本次目标

1. 将「组织」升级为**多层级树 + 多来源 + 多归属**的统一概念，支持普通模式与 OneID 模式共用同一套底层
2. OneID 模式下对接 OneID 的"组织架构"与"用户组"，组织架构需管理员显式勾选同步
3. 统一全站"应用范围"术语为 `全部用户 / 按组织`，抽出共享组件 `ScopePopover`
4. 唯一型资源（VPC/安全组/EIP/镜像/记忆/网盘/平台策略）从"企业单例"改造为"多条资源 + 应用范围 + 可选平台默认"
5. 用户管理页只保留 `全部 / 组织` 两个 tab；OneID 模式下新增「部门」列展示主部门+兼任
6. 组织视图右侧新增「配置总览」Tab，与「成员列表」并列；带初始化异常提醒
7. 组织左侧列表增加健康圆点与"仅看异常"筛选
8. 用户端组织选择（多组织时二选一）**规划写入 PRD**，**本期不做实现**

### 0.3 本期范围

- ✅ Admin 端（管控端）全部改造
- 📝 用户端（Tenant 端）**仅写规划**，不做实现
- ❌ 直接指定用户授权（废弃，强制走组织）

---

## 1. 核心概念与术语

### 1.1 术语对齐

| 术语 | 含义 | 备注 |
|---|---|---|
| **应用范围** | 一条资源对谁生效，由"全部用户"一档 或 "按组织 `groupIds[]`"构成 | 两档互斥 |
| **全部用户** | 所有用户命中 | |
| **按组织** | 命中指定组织及其**所有后代组织**的所有成员 | 多层级树，勾父即勾子树 |
| **平台默认** | 唯一型资源的全局兜底（可选），单选 | |
| **组织** | 统一抽象概念，见 §1.2 | v2.0 合并了 v1.0 的"组织"和"部门" |
| **主部门** | OneID 返回的 `primaryDeptId`，用户的主要归属；指向一个 `source='oneid-dept'` 的组织记录 | 仅 OneID 模式有 |
| **兼任** | 用户归属的非主部门的其他组织架构节点 | 仅 OneID 模式有 |
| **加法型资源** | 用户可同时拥有多条，多源命中取并集 | |
| **唯一型资源** | 用户一次只能用一条，多源命中按优先级取一条 | |

**应用范围结构**：

```
scope = { type: 'all' }
     或 { type: 'filtered', groupIds: [G1, G2, ...] }   // 至少 1 个
```

### 1.2 「组织」的统一抽象（v2.0 核心）

**组织**是一棵带来源标记的多层级树，用 `UserGroup` 表统一表达，通过 `source` 字段区分三种来源：

| source 值 | 来源 | 普通模式 | OneID 模式 | 可编辑 | 层级 |
|---|---|---|---|---|---|
| `manual` | 管理员在 ClawPro 自建 | ✅ 支持 | ❌ **禁止** | 完全可编辑 | 多层级 |
| `oneid-dept` | OneID 的组织架构 | ❌ 不存在 | ✅ 手动勾选同步 | **只读**（只能"取消同步") | 多层级 |
| `oneid-group` | OneID 的用户组 | ❌ 不存在 | ✅ 自动同步 | **只读** | 多层级 |

**关键规则**：

- 普通模式与 OneID 模式**共用同一套 `UserGroup` 表**，只是存在的 `source` 值不同
- 用户归属是**多对多**：一个用户可以同时属于多个组织（不论来源）
- 单用户可加入的组织数**无上限**
- 组织层级深度**无上限**
- **勾选父组织 = 勾选该组织子树下的全部成员**（scope 判定时）
- OneID 模式不支持 `manual` 来源；普通模式不存在 `oneid-*` 来源

### 1.3 命中判定规则（加法型通用）

给定资源 R 与用户 U：

```
IF R.scope.type === 'all':
    命中
ELSE IF R.scope.type === 'filtered':
    FOR g IN R.scope.groupIds:
        IF U 直接或间接（通过祖先→后代）属于 g 的子树:
            命中
    未命中
```

"U 属于 g 子树" = U 的 `groupIds[]` 中任一组织是 g 自身或 g 的后代。

### 1.4 命中来源标记（用于唯一型优先级与 UI 标识）

当 U 命中资源 R 时，同时记录**命中来源类型**，用于唯一型优先级与覆盖状态展示：

| 来源 | 触发条件 |
|---|---|
| `all` | `R.scope.type === 'all'` |
| `primary-dept` | 命中组织 g 是 OneID 组织架构(`source='oneid-dept'`) 且 g 位于 U 的**主部门祖先链**上 |
| `secondary-dept` | 命中组织 g 是 OneID 组织架构(`source='oneid-dept'`) 且 g 位于 U 的**兼任祖先链**上 |
| `group` | 其他所有情况（manual / oneid-group，或 oneid-dept 但不在 U 的部门链上） |

**唯一型优先级**：`group > primary-dept > secondary-dept > all/平台默认`

**命中多种来源时**：按上述优先级取最高的那种来源记录。

### 1.5 主部门无效判定

以下任一条件命中即 `⚠ 主部门缺失`：
1. 主部门 = 企业根节点
2. 主部门为空 / null
3. 主部门在已同步的 `UserGroup` 中找不到（对应节点被"取消同步"或 OneID 删除）

### 1.6 唯一型"最终生效"规则（普通/OneID 通用）

```
优先级 1  来源 = group           → 多条同层冲突由管理员决策（见 §1.7）
优先级 2  来源 = primary-dept    → 主部门祖先链从深到浅取首个命中
优先级 3  来源 = secondary-dept  → 兼任链，同深度多命中由管理员决策
优先级 4  来源 = all / 平台默认
```

**跨层不构成冲突**：高层任一命中则直接返回。

### 1.7 冲突决策机制

只有**同一层内多条不同资源命中**才构成真冲突：

| 冲突类型 | 触发 | 决策入口 |
|---|---|---|
| 多组织同层冲突 | U 属于多个组织，分别被不同资源命中 | (1) 编辑资源应用范围时弹决策 Dialog；(2) 用户行 `⚠ 组织冲突 [详情]` Popover |
| 兼任同深度冲突 | U 的兼任链中同深度多个节点被不同资源命中 | 同上 |

**决策默认值**：
- 编辑应用范围弹窗：默认"本次操作涉及的资源"胜出
- 用户行 Popover：默认"最新绑定的资源"胜出

**未决策兜底**：取 `ScopeBinding.createdAt` 最晚者胜出，行上仍标 `⚠`。

**决策生命周期**：
- 候选集退化到 ≤1 条 → 决策自动归档
- 候选集仍 ≥2 条但 winner 在新候选集内 → 决策继续有效
- winner 已不在新候选集内 → 决策作废，安静降级为"最新绑定兜底"

### 1.8 多组织归属与用户端选择（规划）

当用户归属 **≥ 2 个组织**（不论来源）时，管控端的资源计算依然按 §1.3 / §1.6 执行（多来源合并）。

**但用户端在创建 Agent 时需要知道"我这次用哪个组织的资源"**。为此设计：
- 用户端记录 `activeGroupId`（用户 profile 字段）
- 每次创建 Agent 时，按 `activeGroupId` 指向的那一条组织线去解析资源
- 触发用户端重新选择的条件见 §4（用户端规划）

> 管控端本期只做"提醒管理员：这些用户的组织名会在用户端暴露"的常驻 Alert；**用户端选择交互本期不实现**。

---

## 2. 信息架构

### 2.1 菜单结构

**不新增菜单项**，沿用现有"用户管理"菜单。**页面内部两个 tab**：

```
用户管理
├─ 全部（列表）
└─ 组织（左树 + 右内容区）
```

普通模式和 OneID 模式共用同一页，差异体现在：
- 全部列表：OneID 模式下**新增「部门」列**
- 组织左树：OneID 模式下**新增「导入组织架构」按钮**、来源徽标
- 列展示：OneID 模式下组织列细组织织架构/用户组

### 2.2 两个 tab 的布局

| Tab | 左侧 | 右侧 |
|---|---|---|
| **全部** | — | 全宽列表（搜索 + 筛选 + 表格） |
| **组织** | 组织树（带圆点 + 筛选 + 导入按钮） | Tab 组（`成员列表 | 配置总览`） |

---

## 3. 页面详细设计（管控端）

### 3.1 用户管理页顶部工具栏

保持现有实现：
- 页面标题 "成员管理"
- 右上角操作按钮（普通模式："添加用户"；OneID 模式："手动同步"）
- tab 切换 `[全部 | 组织]`
- 模式开关（普通/OneID）保持 dev 用调试开关

**下掉**：v1.0 的"部门"tab 档。

---

### 3.2 全部视图：表格列调整

**沿用现有**：搜索、部门/组织筛选下拉、角色筛选、清除筛选按钮、表格选择、分页。

**列定义**：

| 模式 | 列 |
|---|---|
| **普通** | 姓名 / 邮箱 / **组织** / 加入时间 / 操作 |
| **OneID** | 姓名 / 邮箱 / **部门** / **组织** / 加入时间 / 操作 |

#### 3.2.1 「部门」列（仅 OneID 模式）

展示主部门 + 兼任：

- 主文本：**主部门完整路径**，如 `ClawPro企业 / 产品一部 / 前端组`
- 主部门无效时：显示根路径 + 红色小 `⚠`
- 有兼任：右侧 `+N` 胶囊
- Hover `+N` → `HoverCard`：

```
┌──────────────────────────────────────────┐
│ 1. ClawPro企业 / 产品一部 / 前端组 [主部门]│
│ 2. ClawPro企业 / 视频业务 / 海外一部       │
│ 3. ClawPro企业 / 技术部 / AI 组            │
└──────────────────────────────────────────┘
```

- `[主部门]` 小徽标：`text-[10px] bg-blue-50 text-blue-600 rounded px-1.5 py-0.5`

**数据源**：主部门来自 OneID，"部门"=`UserGroup.source='oneid-dept'`。

#### 3.2.2 「组织」列（两种模式都有）

展示该用户**所有归属的组织**（任何来源）：

**普通模式**：只会有 `source='manual'` 的组织，列展示示例：

```
产品组, AI 攻坚组
```

- 多个组织逗号分隔；超过 2 个显 `前两个 +N`
- 每个组织点击跳到组织视图对应节点（可选，本期可先不做跳转）

**OneID 模式**：可能混合 `oneid-group` 和 `oneid-dept`（以及主部门本身作为部门也算一条）。展示规则：

- **组织列仅展示"用户组 + 非主部门的兼任组织架构"**（主部门已在「部门」列展示，避免重复）
- 类型用**小图标**区分：
  - `Users` 图标（lucide）= 用户组（`oneid-group`）
  - `Building2` 图标（lucide）= 组织架构（`oneid-dept`）

示例：

```
[Users] AI 攻坚小组, [Building2] 海外一部, [Users] VIP 客户组 +2
```

**多组织提醒徽标**：如果该用户归属组织数（组织列中显示的条目数）`>= 2`，在组织列尾加一个橙色 `⚠` 小图标，HoverCard 说明"该用户在用户端需选择使用哪个组织，组织名会被暴露"（辅助说明；顶部 Alert 是主提醒，见 §3.3）。

---

### 3.3 组织视图整体结构

```
┌────────────────────────────────────────────────────────────────┐
│ [顶部 Alert：多组织提醒，常驻]  (仅当存在归属 ≥2 组织的用户)      │
├─────────────────┬──────────────────────────────────────────────┤
│                 │  [组织名] · N 人       [Tab: 成员 | 配置]     │
│  [左侧组织树]   │  ─────────────────────────────────────       │
│                 │                                                │
│                 │  (右侧 Tab 内容)                              │
└─────────────────┴──────────────────────────────────────────────┘
```

#### 3.3.1 顶部常驻 Alert（多组织暴露提醒）

**位置**：组织 tab 的**最顶部**（左树和右内容区之上，贯穿整个内容区宽度）。

**触发**：只要系统中存在**至少一位归属 ≥ 2 个组织**的用户就展示；否则隐藏。

**样式**：
- 容器：`rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 mb-4`
- 图标：lucide `AlertTriangle` w-4 h-4 `text-amber-600`
- 文案：

```
[⚠] 当前有 N 位用户归属了多个组织。这些用户在端侧创建 Agent 时，
     需要自行选择使用哪个组织，且组织名称会被展示给用户。
     [查看这些用户 →]
```

- 右端 `[查看这些用户 →]` 点击 → 切换到「全部」tab，自动过滤显示这些用户

**不可关闭**（常驻）。

#### 3.3.2 左侧组织树

宽度 `w-72`（288px）。

**头部工具栏**（紧凑）：
```
┌──────────────────────────────────────┐
│ [🔍 搜索组织...]                      │
│ [筛选: 全部 ▾]          [+ 新建组织]  │  ← 普通模式
│ [筛选: 全部 ▾]   [⬇ 导入组织架构]    │  ← OneID 模式
└──────────────────────────────────────┘
```

**筛选下拉**（下拉菜单）：
- 全部（默认）
- 仅显示正常
- 仅显示异常

**「+ 新建组织」按钮**（仅普通模式）：品牌渐变主按钮，弹出新建组织对话框（可选择 `parentId` 作为所属父组织）。

**「⬇ 导入组织架构」按钮**（仅 OneID 模式）：见 §3.5。

**树节点行**：

```
  ▾ [Building2] 产品一部        ● 245
    ▾ [Users] AI 攻坚小组      ● 18
      [Building2] 前端协作组    ○ 6
```

- 折叠图标：左侧 `ChevronRight`(折叠) / `ChevronDown`(展开)
- 来源图标（仅 OneID 模式显示，普通模式下 manual 不显示图标）：
  - `Building2` = 组织架构（oneid-dept）
  - `Users` = 用户组（oneid-group）
- 名称
- 右端：**健康圆点 + 成员数**（成员数 = 该组织及子组织去重后的成员总数）
- 选中态：品牌渐变左边框 + 浅蓝底
- 只读节点（oneid-dept / oneid-group）鼠标悬停显示"来源：OneID 组织架构"Tooltip

**节点操作菜单**（右键或 `MoreHorizontal`）：
- `manual` 节点：重命名 / 新增子组织 / 删除
- `oneid-dept` / `oneid-group` 节点：**仅"取消同步"**（oneid-dept）/ 无操作（oneid-group 自动同步）

**健康圆点**（节点级独立判断，§1.6 规则，不看子孙聚合）：
- ● 绿色 (`bg-emerald-500`)：该节点**自身作为 scope 目标**时，所能提供给成员的资源满足"三大核心"（≥1 模型 + ≥1 通道 + 1 安全组）
- ● 红色 (`bg-red-500`)：缺任一项
- 判定时**考虑祖先继承 + 全部用户档 + 平台默认**的兜底（§3.3.4.2 细化）

#### 3.3.3 右侧头部

```
[组织名]  [来源徽标]   · N 人
路径：全公司 / 产品一部 / AI 攻坚小组

[Tab: 成员列表 | 配置总览]
```

- 来源徽标（仅 OneID 模式）：
  - 组织架构：`bg-blue-50 text-blue-700 text-xs rounded-full px-2 py-0.5` 含 `Building2` 图标
  - 用户组：`bg-purple-50 text-purple-700 ...` 含 `Users` 图标
- 成员数=该组织**直属成员 + 所有后代组织成员**去重合计

#### 3.3.4 右侧「成员列表」Tab

表格列与§3.2.1 全部视图一致（OneID 模式多出"部门"列），额外新增：

- **「覆盖状态」列**（仅 OneID 模式，表达该用户相对当前组织节点的覆盖情况）
- **「操作」列**：`查看配置` → 抽屉

表格数据：当前组织**及所有后代组织**的去重成员列表。

**覆盖状态取值**（对当前选中节点 N 的视角）：

| 状态 | 触发 |
|---|---|
| `按本节点` | 用户的最终生效 = 当前节点配置给他的 |
| `⚠ 组织覆盖` | 组织比当前节点优先级更高 |
| `⚠ 组织冲突` | 多组织同层冲突 |
| `⚠ 兼任覆盖` | 兼任部门命中更优 |
| `⚠ 同深度冲突` | 兼任同深度冲突 |
| `⚠ 主部门缺失` | U 主部门无效 |

点击 `[详情]` → 就地 Popover 展示最终生效与冲突决策（同 v1.0 设计，沿用 §3.3.3 抽屉）。

**操作 → 查看配置抽屉**：宽 `w-[600px]`，分三段：
- 基本信息（主部门 / 兼任 / 组织列表）
- 加法型资源（模型/通道/技能/Agent 工具，列最终并集）
- 唯一型资源（VPC/安全组/EIP/镜像/记忆/网盘/平台策略，列最终生效+来源）

#### 3.3.4.2 节点健康度详细判定

对组织节点 N 的健康度判定：

```
IF N 作为 scope=filtered.groupIds=[N.id] 时，所能派发给 N 子树成员的资源集合中：
    存在至少 1 条模型、1 条通道、1 个安全组 能被 N 子树中所有成员最终生效命中
    ↓
    健康

ELSE 异常
```

**实际判定简化为**：对 N 子树中**每一位成员 U**，分别计算 U 的最终生效（§1.6 规则，带全部用户档 + 平台默认 + 祖先继承的全部条件），若所有 U 都满足"模型≥1 + 通道≥1 + 安全组=1"则正常；否则异常。

**这对应了确认点 5-2**：初始化检查**要**考虑该组织用户通过部门继承/全部用户档的满足情况，不是只看本节点是否直接被某条资源指向。

#### 3.3.5 右侧「配置总览」Tab（新）

展示**当前组织节点** 作为 `scope=filtered.groupIds=[节点id]` 目标时，直接指向它的所有资源；并**做初始化检查**。

**布局**：

```
┌──────────────────────────────────────────────────────┐
│ 初始化状态     ● 正常 / ● 异常                         │
│ (异常时) ⚠ 尚未完成初始化配置，请前往以下配置页补全： │
│   [补全模型 →] [补全通道 →] [补全安全组 →]           │
├──────────────────────────────────────────────────────┤
│ [模型]                                                │
│  · Claude 3.5 Sonnet   (直接配到本组织)   →模型管理   │
│  · GPT-4o              (配到祖先组织"产品部"继承得到) │
│                                                       │
│ [通道]                                                │
│  · 飞书                (直接配到本组织)    →通道管理  │
│                                                       │
│ [技能]                                                │
│  · 翻译, 摘要 +5                            →技能管理 │
│                                                       │
│ [Agent 工具库]                                        │
│  · （无）                                             │
├──────────────────────────────────────────────────────┤
│ [VPC]             VPC-AI (直接配到本组织) →VPC 管理   │
│ [安全组]          sg-main (继承自平台默认) →安全组    │
│ [公网 EIP]        (无，使用平台默认)                 │
│ [镜像]            coder-v2 (全部用户档)  →镜像管理    │
│ [记忆]            mem-default (平台默认) →记忆管理    │
│ [网盘]            (无)                                │
│ [平台策略]        7 项                     →平台策略  │
└──────────────────────────────────────────────────────┘
```

**初始化检查规则**（§5-2 确认）：
- 三大核心（模型 ≥ 1 / 通道 ≥ 1 / 安全组 = 1）对**该节点所有成员**的最终生效计算后全部满足 → **● 正常**
- 否则 → **● 异常**，列出缺失项及跳转按钮

**缺失项跳转**：
- 模型缺失 → `[补全模型 →]` 跳到模型管理页
- 通道缺失 → `[补全通道 →]` 跳到通道管理页
- 安全组缺失 → `[补全安全组 →]` 跳到安全组管理页
- 跳转时应带上 query 参数，目的页可预填 scope（`?scope-target-group={节点id}`）

**"直接配到本组织" vs "继承/全部用户档"**：
- 直接配：资源 `scope.type='filtered' && scope.groupIds` 含当前节点 id
- 继承：资源 scope 含当前节点的某个祖先组织 id
- 全部用户档：`scope.type='all'`
- 平台默认：`isPlatformDefault` 且未被 scope 命中
- 展示时右端灰色小文字说明来源

---

### 3.4 资源应用范围组件 `<ScopePopover>`

所有资源配置页统一用此组件。

#### 3.4.1 类型

```ts
type Scope =
  | { type: 'all' }
  | { type: 'filtered'; groupIds: string[] };     // length >= 1

interface ScopePopoverProps {
  scope: Scope;
  oneidEnabled: boolean;              // 仅用于分类显示，不影响底层数据
  resourceType: 'additive' | 'unique';
  onChange: (scope: Scope) => void;
  trigger: 'badge' | 'icon';
}
```

**v2.0 重大变化**：
- **移除 `deptIds` 字段**，组织架构以 `UserGroup.source='oneid-dept'` 存在，同样通过 `groupIds` 引用
- 不再有"按组织 vs 按部门"的二分，统一都是「按组织」
- OneID 模式下在 UI 上做**分类显示**（组织架构 / 用户组 / 自建组织），但写回的都是 `groupIds[]`

#### 3.4.2 UI

**顶部 Radio**：`全部用户` / `限定范围`

**限定范围时**（OneID 模式按来源分三区；普通模式只有一区）：

```
┌───────────────────────────────────────────┐
│ 应用范围                                   │
│                                            │
│ ( ) 全部用户                               │
│ (●) 限定范围（至少选一项）                 │
│                                            │
│  ── 组织架构 ──                            │  仅 OneID 模式显示
│   ☐ ClawPro企业                            │  (树状，支持折叠)
│     ☑ 产品一部                             │
│       ☐ AI 攻坚小组                        │
│                                            │
│  ── 用户组 ──                              │  仅 OneID 模式显示
│   ☐ VIP 客户组                             │
│                                            │
│  ── 自建组织 ──                            │  仅普通模式显示
│   ☑ 产品组                                 │
│   ☐ 研发组                                 │
│                                            │
│  ⓘ 勾选父组织 = 应用到该组织及其所有子组织 │
│                                            │
│                   [取消] [确定]            │
└───────────────────────────────────────────┘
```

- 宽度 `w-96`
- 分区标题：`text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2`
- 保存校验：限定范围下 `groupIds.length >= 1`，否则按钮置灰

#### 3.4.3 胶囊徽章

| scope | 文案 | 样式 |
|---|---|---|
| `all` | `全部用户` | 灰胶囊 |
| filtered，1 个 | `{组织名}` | 蓝色胶囊 |
| filtered，多个 | `{首个组织} +N` | 蓝色胶囊 |

HoverCard 按来源分类列出所有勾选项（OneID 模式）：

```
组织架构（1）
 · ClawPro企业 / 产品一部
用户组（1）
 · VIP 客户组
```

---

### 3.5 OneID「导入组织架构」弹窗（新）

**触发**：OneID 模式，组织视图左树工具栏 `[⬇ 导入组织架构]`。

**作用**：管理员选择要把 OneID 的哪些组织架构节点同步到本系统作为 `source='oneid-dept'` 的组织。用户组 `oneid-group` **不需要此弹窗（自动同步）**。

**弹窗结构**：

```
┌───────────────────────────────────────────────────────────┐
│  导入 OneID 组织架构                              [×]     │
│  选择需要同步到 ClawPro 的组织架构节点                     │
├───────────────────────────────────────────────────────────┤
│  [🔍 搜索]                              [全选] [全不选]   │
│                                                            │
│  ☑ ClawPro企业                                  [已同步]   │
│    ☑ 产品一部                                   [已同步]   │
│      ☐ 前端组                                              │
│      ☑ 后端组                                   [已同步]   │
│    ☐ 视频业务                                              │
│      ☐ 海外一部                                            │
│    ☐ 技术部                                                │
│                                                            │
│  共选中 4 项（含本次新增 2 项、取消同步 1 项）             │
│                                                            │
│                                   [取消] [确认同步]        │
└───────────────────────────────────────────────────────────┘
```

**交互**：
- 树状多选，勾选父节点**不自动勾选所有子节点**（需要管理员精确选择哪些层级同步）
- 右端 `[已同步]` 徽标表示该节点当前已是 `oneid-dept` 组织
- 取消勾选 `[已同步]` 节点 = 取消同步（该节点从 `UserGroup` 表删除；若有资源 scope 引用它需提示）
- 底部实时显示"本次新增 N 项 / 取消同步 M 项"
- 保存时校验：如果要取消同步的节点**正作为某资源的 scope 目标**，弹确认二级弹窗
- 宽度 `max-w-2xl`（例如 768px），高度 `max-h-[70vh]`，内部可滚

**取消同步的级联清理**：
- 从 `UserGroup` 表删除该记录
- 从 `UserGroupMembership` 删除所有相关行
- 从 `ScopeBinding` 删除所有 `targetId` = 该节点的绑定
- 任何资源 scope 因此变为 `groupIds=[]` 时自动降级为 `type='all'`（并记录 toast 提醒）

---

### 3.6 视觉规范沿用

- 页面根元素 `page-enter`
- Admin 背景 `#F0F2F8`，内容区 `p-8`
- 卡片：原生 `div` + `rounded-2xl` + inline boxShadow
- 徽章：`rounded-full px-2 py-0.5 text-xs`
- 主按钮：品牌渐变 + `btn-primary-glow`
- 健康圆点：正常 `w-2 h-2 rounded-full bg-emerald-500` / 异常 `bg-red-500`
- 警告图标：lucide `AlertTriangle`，amber 级覆盖/兼任 `text-amber-500`，red 级主部门缺失/同深度冲突 `text-red-500`

---

## 4. 用户端规划（Tenant 端，本期不实现）

> 以下为规划条文，进入后续迭代。

### 4.1 `activeGroupId` 持久化

用户 profile 增加字段 `activeGroupId: string | null`。

### 4.2 自动态

- 用户归属组织数 = 0：无影响，只按"全部用户"档获得资源
- 用户归属组织数 = 1：`activeGroupId` 自动设为该组织 id，**全程无感**
- 用户归属组织数 ≥ 2：需用户显式选择（见 §4.3）

### 4.3 触发用户选择

**首次登录**：
- 登录完成后，若归属组织 ≥ 2 且 `activeGroupId` 为空 → 弹出组织选择弹窗，必选一项才能进主界面

**已登录用户，组织发生变化**：

| 变化类型 | 是否弹 |
|---|---|
| 归属组织数 1 → 2（新增加入一个）| **弹** |
| 归属组织数 2 → 3（再加一个）| **弹** |
| 归属组织数 2 → 1（移除一个，且剩的那个是 activeGroupId）| 不弹，自动把 `activeGroupId` 锁定 |
| 归属组织数 2 → 1（移除一个，但剩的那个**不是** activeGroupId）| 不弹（因为只剩一个，自动锁定） |
| 归属组织数 ≥ 2，其中一个被移除，且被移除的就是 activeGroupId | **弹**（要求重选） |
| 归属组织数 ≥ 2，原 activeGroupId 仍在 | 不弹 |

**交互**：
- 登录态内检测到变化 → 全局模态弹出，必选一项才能继续
- 弹窗展示所有归属组织的名称（带来源徽标）

### 4.4 组织选择弹窗

```
┌─────────────────────────────────────────────┐
│ 请选择要使用的组织                            │
│ 你被管理员加入了多个组织，需要指定本次使用哪个 │
├─────────────────────────────────────────────┤
│ (○) [Users] AI 攻坚小组                      │
│ (○) [Building2] 产品一部 / 前端组            │
│ (●) [Users] VIP 客户组                       │
│                                              │
│                  [确定]                      │
└─────────────────────────────────────────────┘
```

- 无取消按钮（必选）
- 选择后：写入 `activeGroupId`，继续主界面

### 4.5 资源解析规则

用户创建 Agent / 拉取可用资源时：
- `activeGroupId = null` → 按"未在任何组织"逻辑走，只拿全部用户档 + 平台默认
- `activeGroupId` 指向某组织 g → 把 g 视为"当前用户的唯一组织"做资源计算（其他组织忽略）

> 这意味着：**同一用户在不同 active 组织下看到的可用资源不同**。符合"用户自选组织"的直觉。

---

## 5. 数据模型

### 5.1 类型定义

```ts
// shared/schema.ts

// 统一的「组织」表（v2.0 合并 v1.0 的 Department 和 UserGroup）
interface UserGroup {
  id: string;
  name: string;
  description?: string;
  parentId: string | null;            // 多层级支持
  source: 'manual' | 'oneid-dept' | 'oneid-group';
  oneidNodeId?: string;               // OneID 节点 ID（oneid-* 时）
  readonly: boolean;                  // oneid-dept / oneid-group = true
  createdAt: Date;                    // 冲突"最新绑定兜底"用
}

// 归属关系（多对多）
interface UserGroupMembership {
  id: string;
  userId: string;
  groupId: string;
  joinedAt: Date;
}

// 用户 OneID 层信息（保持）
interface UserOrgProfile {
  userId: string;
  primaryGroupId: string | null;       // 主部门 → 指向 source='oneid-dept' 的 UserGroup
  primaryGroupValid: boolean;
  allSyncedDeptGroupIds: string[];     // 从 OneID 取的"此人归属的所有组织架构"交集（含主部门）
  allSyncedUserGroupIds: string[];     // 从 OneID 取的"此人归属的所有用户组"
  lastSyncAt: Date;
}

// 应用范围（嵌入各资源）
type Scope =
  | { type: 'all' }
  | { type: 'filtered'; groupIds: string[] };   // length >= 1

// 唯一型资源的平台默认
interface UniqueResource {
  scope: Scope;
  isPlatformDefault?: boolean;
}

// 应用范围绑定（每个 groupId 勾选一条，用于"最新绑定兜底"）
interface ScopeBinding {
  id: string;
  resourceKind: 'model' | 'channel' | 'skill' | 'agentTool'
             | 'vpc' | 'securityGroup' | 'eip' | 'image'
             | 'memory' | 'netDisk' | 'platformPolicy';
  resourceId: string;
  groupId: string;
  createdAt: Date;
  // 约束：(resourceKind, resourceId, groupId) 唯一
}

// 冲突决策记录（仅唯一型）
interface ConflictResolution {
  id: string;
  userId: string;
  resourceKind: 'vpc' | 'securityGroup' | 'eip' | 'image'
             | 'memory' | 'netDisk' | 'platformPolicy';
  agentType?: string;                  // 仅 image
  winnerResourceId: string;
  conflictLayer: 'group' | 'secondary-dept';
  candidateResourceIds: string[];
  resolvedAt: Date;
  resolvedBy: string;
  // 约束：(userId, resourceKind, agentType?) 唯一
}
```

### 5.2 废弃

- ❌ `Department` 表：整体移除，合并为 `UserGroup.source='oneid-dept'`
- ❌ `Scope.deptIds`：移除
- ❌ `ScopeBinding.targetType` / `targetId` 的二分：只保留 `groupId`

### 5.3 迁移策略

对于 v1.0 已有数据：

```
旧 Department 记录
  → 迁移为 UserGroup { source: 'oneid-dept', readonly: true, parentId, createdAt }

旧 Scope.deptIds = [D1, D2]
  → 合并进 Scope.groupIds = [...groupIds, ...deptIds]

旧 ScopeBinding {targetType:'dept', targetId:X}
  → ScopeBinding {groupId: X}  (X 此时是 UserGroup id)

旧 ConflictResolution.conflictLayer='secondaryDept'
  → 保留，枚举值改为 'secondary-dept'
```

---

## 6. 最终生效计算

### 6.1 加法型

```ts
function resolveAdditive(U: User): Resource[] {
  const userGroupTree = expandGroupsToSelfAndAncestors(U.groupIds);  
  // 注意：命中条件是"U 的某归属组织是 R.scope.groupIds 中某项的后代或自身"
  // 所以 U 端要把自己的 groupIds 展开到祖先，再与 R 的 groupIds 做交集
  
  return resources.filter(r => {
    if (r.scope.type === 'all') return true;
    return r.scope.groupIds.some(gid => 
      userGroupTree.includes(gid)
    );
  });
}

// "U 的某 groupId 或其祖先 ∈ R 的 scope.groupIds"的等价判定：
// 把 U 的每个 groupId 沿 parentId 向上展开到根，合并得到 userAncestorSet
function expandGroupsToSelfAndAncestors(userGroupIds: string[]): string[] {
  const result = new Set<string>();
  for (const gid of userGroupIds) {
    let cur: string | null = gid;
    while (cur) {
      result.add(cur);
      cur = groupMap.get(cur)?.parentId ?? null;
    }
  }
  return [...result];
}
```

### 6.2 唯一型

按优先级 `group > primary-dept > secondary-dept > all` 逐层取首个命中；同层多条不同资源进入冲突决策。

来源判定（§1.4）：
- 命中组织 g 的 `source === 'oneid-dept'` 且 g 在 U 主部门祖先链上 → `primary-dept`
- 命中组织 g 的 `source === 'oneid-dept'` 且 g 在 U 兼任祖先链上 → `secondary-dept`
- 其他（manual / oneid-group）→ `group`

---

## 7. 资源配置页改造清单

（与 v1.0 基本一致，只是 ScopePopover 字段变更）

| 资源 | 改造要点 |
|---|---|
| 模型 | 抽出 `ScopePopover` 共享组件；字段 `groupIds` |
| 通道 | 放开"按组织"；替换为共享 Popover |
| 技能 | 语义对齐 `public/private` → `all/filtered`；共享 Popover |
| Agent 工具库 | 新增 scope 字段 + Popover |
| 平台策略 | 每条策略项加 scope；放开按组织 |
| 镜像 | 每个 agentType 下多镜像 + scope + 平台默认 |
| VPC/安全组/EIP | 单例 → 多条 + scope + 平台默认 |
| 记忆/网盘 | 同上 |

---

## 8. 实施顺序

### Phase 1（本期）：管控端核心改造
1. 数据层：`UserGroup` 升级 + `Department` 移除 + 迁移脚本
2. 用户管理页：下掉部门 tab，全部/组织列调整
3. 组织视图：左树多层级 + 圆点 + 筛选；右 Tab（成员列表 / 配置总览）
4. OneID 导入组织架构弹窗
5. 多组织暴露常驻 Alert

### Phase 2（后续）：资源配置页改造
6. 共享 `ScopePopover` 组件落地
7. 加法型资源改造（模型/通道/技能/Agent 工具）
8. 唯一型资源改造（平台策略/镜像/VPC/安全组/EIP/记忆/网盘）
9. 平台默认 / 无平台默认红色横幅

### Phase 3（规划）：用户端
10. `activeGroupId` 字段与首次登录选择弹窗
11. 组织变化自动触发重选
12. 用户端资源解析按 `activeGroupId` 走

---

## 9. 决策记录

### v2.0 新增决策

| 编号 | 决策 | 备注 |
|---|---|---|
| V1 | 下掉部门 tab | 用户管理页只保留 `全部 / 组织` 两个 tab |
| V2 | 合并 Department → UserGroup | 通过 `source` 区分来源 |
| V3 | OneID 模式不支持 manual 组织 | 全部去 OneID 创建 |
| V4 | 用户组自动同步，组织架构手动勾选 | 分两种同步策略 |
| V5 | 同步的 OneID 节点只读 | 只允许"取消同步" |
| V6 | `Scope.deptIds` 移除 | 统一 `groupIds` |
| V7 | 多组织暴露提醒常驻 Alert | 方案 A，放在组织 tab 顶部 |
| V8 | 初始化检查考虑继承/全部用户档 | 不只看本节点直接指向 |
| V9 | 筛选用下拉菜单 | 全部/仅正常/仅异常 |
| V10 | 健康度父节点不聚合 | 每个组织节点独立判定 |
| V11 | 用户端本期不做 | 仅写规划 |

### 沿用 v1.0 决策

A1/A2/A3/B1/C1-C4/D1/D3/D4/E1/E2/F1/F2/F4/G1/G2/G5/H1/I2/I3/I4/I5/I6/I7/I8/I9 保持不变。

### 废弃 v1.0 决策

- D2 "viewMode 切换三档" → 替代为 V1（只保留两档）
- F3 "parentId 扁平"→ 继承到 V2 的 UserGroup 上（parentId 仍扁平，但表变了）
- I1 "组织+部门可同时勾" → 废弃（不再有部门 scope）
- G4 "镜像保留 agentType 组织" → 保持
- G6 "全部用户 ≠ 平台默认" → 保持

---

文档结束。
