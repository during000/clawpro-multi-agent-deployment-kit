# 管控端 Agent 详情抽屉 - 模型与通道编辑能力 PRD

> 文档版本：v1.0
> 关联分支：`feature/admin-openclaw-monitor`
> 关联页面：管控端「Agent 列表」（`/admin/openclaw-monitor`）→ Agent 详情抽屉
> 不修改文件：`client/src/App.tsx`、`client/src/components/AdminLayout.tsx`

---

## 1. 背景

在现状中，管控端「Agent 列表」页点击 `instanceId` 会从右侧滑出 593px 宽的 **Agent 详情抽屉**，抽屉内的「已应用模型」与「已接入通道」两个区块为**纯只读展示**。

随着企业管理员需要从管控视角统一调整 Agent 配置（例如某些 Agent 应该统一切换到新模型、批量补齐通道凭证等），需要在此抽屉中新增**对当前选中 Agent 实例**的「应用模型」与「已接入通道」**编辑能力**，且必须复用「模型配置」「通道配置」两个管理员页面已经维护好的统一数据，避免管理员在多个入口重复录入或维护配置漂移。

---

## 2. 目标与非目标

### 2.1 目标
- 管理员在「Agent 列表」抽屉内**直接**完成对某个 Agent 的"应用模型替换"与"已接入通道增/删/改"操作，无需跳页
- 模型选项**严格来自**管控端「模型配置」页中**当前对用户可见的模型**；通道选项**严格来自**管控端「通道配置」页中**当前对用户可见的通道**
- 管控端在两个配置页的任何变更（新增、删除、修改、开关可见性）**立刻反映**到 Agent 详情抽屉的下拉列表
- 编辑操作有**清晰反馈**（toast）和**危险操作二次确认**（AlertDialog）

### 2.2 非目标
- ❌ 不实现"多主备模型列表"（保持单一已应用模型语义）
- ❌ 不在管控端抽屉中重复进行模型详细参数（API Key / URL / 多模态开关等）的录入工作 —— 这些参数由「模型配置」页统一维护
- ❌ 不实现飞书 / 企微 / 微信的"快捷扫码授权"流程（管理员不替租户绑定个人账号凭证）
- ❌ 不做后端 API 联调，所有数据保留在 `localStorage` + 内存

---

## 3. 用户故事

| ID | 角色 | 我希望 | 以便于 |
|---|---|---|---|
| US-1 | 企业管理员 | 在 Agent 详情抽屉中切换某个 Agent 应用的模型 | 集中管理模型分配，无需让租户用户自己改 |
| US-2 | 企业管理员 | 切换模型时，下拉只展示我在「模型配置」页配好并设为用户可见的模型 | 不重复维护、不暴露未上架的模型 |
| US-3 | 企业管理员 | 在 Agent 详情抽屉中添加 / 移除 / 修改某个 Agent 的接入通道及其凭证 | 帮租户排查或预配置通道凭证 |
| US-4 | 企业管理员 | 添加通道时，下拉只展示我在「通道配置」页开启了"用户可见"的通道 | 与企业策略保持一致，避免误添加内部不允许的通道 |
| US-5 | 企业管理员 | 模型配置 / 通道配置页所做的变更，立即在抽屉的下拉里生效 | 不需要刷新页面或重新打开抽屉 |

---

## 4. 功能详述

### 4.1 数据共享：跨页面 store

#### 4.1.1 新增 `modelConfigStore`
路径：`client/src/lib/modelConfigStore.ts`

**职责**：作为管控端「模型配置」页与「Agent 详情抽屉」之间的**单一数据源**。

**类型定义**：
```ts
export interface ModelRow {
  id: string;
  name: string;            // 厂商名（"腾讯云 DeepSeek" / "自定义模型" / "OpenAI GPT-4o"）
  version: string;         // 版本名（"DeepSeek V3 0324"）
  modelUrl: string;
  visible: boolean;        // 用户可见开关（关键过滤字段）
  isDefault: boolean;
  dailyLimit: number;
  provider: string;        // 厂商标识；自定义模型为 "__custom__"
  versions: string[];
  isMultimodal?: boolean;
  visibilityScope: "all" | "groups";
  visibilityGroupIds: string[];
}

export const CUSTOM_PROVIDER_VALUE = "__custom__";
```

**API**：
| API | 说明 |
|---|---|
| `loadAdminModels()` | 读取所有模型；首次访问时返回默认数据 |
| `saveAdminModels(models)` | 写入并广播变更（同页面 + 跨标签页） |
| `onAdminModelsChange(cb)` | 订阅变更，返回 unsubscribe 函数 |
| `useAdminModelsState()` | **React Hook**，签名同 `useState<ModelRow[]>`，setter 自动持久化 + 广播 |
| `useAdminModels()` | **React Hook**，只读订阅模型列表 |

**广播机制**：
- 同标签页：`window.dispatchEvent(new CustomEvent("openclaw_admin_models_changed"))`
- 跨标签页：`localStorage` 的 `storage` 事件
- 与 `customChannelStore.ts` 同款模式，技术栈保持一致

**持久化键**：`localStorage["openclaw_admin_models"]`

#### 4.1.2 复用 `customChannelStore`
路径：`client/src/lib/customChannelStore.ts`（已存在，本期未改动）

复用的 API：
- `loadBuiltinChannelVisibility()` / `onBuiltinChannelVisibilityChange(cb)` — 6 个内置通道的可见性开关
- `loadVisibleCustomChannels()` / `onCustomChannelsChange(cb)` — 管控端定义的自定义通道（已自动过滤 `visible: true`）

### 4.2 「模型配置」页改造

路径：`client/src/pages/admin/ModelConfig.tsx`

**改造点**：
| 旧实现 | 新实现 |
|---|---|
| 本地定义 `interface ModelRow` | `import { type ModelRow } from "@/lib/modelConfigStore"` |
| 本地常量 `MOCK_MODELS` | 删除，由 store 默认数据 `DEFAULT_ADMIN_MODELS` 提供 |
| 本地常量 `CUSTOM_PROVIDER_VALUE` | `import { CUSTOM_PROVIDER_VALUE } from "@/lib/modelConfigStore"` |
| `useState<ModelRow[]>(MOCK_MODELS)` | `useAdminModelsState()` |

**效果**：页面所有功能（新增、删除、编辑、切换可见性、设默认、设每日上限、应用范围等）**行为完全不变**，但所有写操作自动 broadcast 给抽屉，且数据写入 `localStorage` 跨页面持久化。

### 4.3 Agent 详情抽屉：「已应用模型」

#### 4.3.1 数据结构

`ClawDetail` 中模型相关字段：
```ts
interface ClawDetail {
  appliedModelId: string;        // 关联 ModelRow.id（精确锁定）
  appliedModel: string;          // 厂商名冗余存储（如"腾讯云 DeepSeek"），用于回退展示
  appliedModelVersion: string;   // 版本/模型名冗余存储
  // ...（通道、技能字段不变）
}
```

`appliedModelId` 用于精确反查模型记录；`appliedModel` / `appliedModelVersion` 用于即使模型记录被删除时仍能展示原值（Q3 完全无感策略）。

#### 4.3.2 厂商组织逻辑

订阅 `useAdminModels()` → 过滤 `visible === true` → 按 `provider` 字段聚合为 `ProviderGroup[]`：

```ts
interface ProviderGroup {
  key: string;        // provider 值；自定义模型组固定为 "__custom__"
  label: string;      // 一级 Select 显示文本
  models: ModelRow[]; // 该厂商下所有可见模型记录
  isCustom: boolean;
}
```

排序规则：**按 `provider` 在可见模型中首次出现的顺序排列；自定义模型组始终放在最后**。

#### 4.3.3 编辑交互

**入口**：「已应用模型」标题右上角「编辑」按钮（`Pencil` 图标）

**编辑态布局**：两级 `Select`，自定义模型作为并列的一级项

| 控件 | 数据源 | 显示 |
|---|---|---|
| 一级 Select：模型厂商 | `providerGroups` 全部一级项 | 普通厂商：`name`（如「腾讯云 DeepSeek」）<br>自定义模型组：固定标签「自定义模型」 |
| 二级 Select：模型名称 | 选中厂商下 `models` 全集 | 普通模型：`version`（如「DeepSeek V3 0324」）<br>自定义模型组下：`name`（如「OpenAI GPT-4o」） |

**默认回填规则**（进入编辑态时）：
1. 用 `detail.appliedModelId` 精确匹配，找到对应的 `ProviderGroup` + `ModelRow` 回填
2. 若 id 找不到（被删 / 被关闭可见性）：回退到 `providerGroups[0]` 的 `models[0]`，**不弹任何 toast**（Q3 完全无感）

**切换一级时**：二级自动重置为该厂商的第一项

**保存按钮**：
- 触发：写入 `appliedModelId`、`appliedModel`（= 厂商 label）、`appliedModelVersion`（= 普通模型 version / 自定义模型 name）
- 反馈：`toast.success("模型已更新")`
- Disable 条件：`!modelDraftProvider || !modelDraftModelId`

**取消按钮**：直接退出编辑态，不修改 detail

**极端场景：当前没有任何可见模型**
- 编辑态显示琥珀色提示横幅（`AlertCircle` + 文案"当前「模型配置」页中没有对用户可见的模型，请前往该页面添加或开启模型可见性。"）
- 保存按钮 disable

#### 4.3.4 只读态展示

```
┌──────────────────────────────────────┐
│ 已应用模型                  [编辑]    │
├──────────────────────────────────────┤
│  腾讯云 DeepSeek                     │
│  DeepSeek V3 0324                    │
└──────────────────────────────────────┘
```

自定义模型的展示格式**与普通模型完全一致**（不加任何徽章）：
```
┌──────────────────────────────────────┐
│ 已应用模型                  [编辑]    │
├──────────────────────────────────────┤
│  自定义模型                          │
│  OpenAI GPT-4o                       │
└──────────────────────────────────────┘
```

#### 4.3.5 已删除字段（相比 v0）

本次按需求**删除**以下能力（已由「模型配置」页统一承担）：
- ❌ 自定义模型 JSON / 表单双 Tab 录入区
- ❌ 自定义模型 6 字段表单（provider/base_url/api/api_key/model_id/model_name）
- ❌ 多模态开关
- ❌「需自费」「多模态」橙色 / 蓝色徽章
- ❌「使用自定义模型需自行承担 Tokens 费用」琥珀色横幅

### 4.4 Agent 详情抽屉：「已接入通道」

#### 4.4.1 数据结构

```ts
interface ConnectedChannel {
  name: string;                            // 通道展示名（CHANNEL_OPTIONS.label）
  value: string;                           // 通道 value（用于反查 fields 定义）
  fieldValues: Record<string, string>;     // 凭证字段值（按 ChannelField.key 存储）
  bots: string[];
}
```

#### 4.4.2 通道选项过滤

订阅两个 store 计算 `availableChannelOptions`（添加通道下拉数据源）：

```ts
const availableChannelOptions = useMemo<ChannelConfig[]>(() => {
  // 内置通道：过滤可见性
  const builtins = CHANNEL_OPTIONS.filter((ch) => {
    const key = ch.builtinId ?? ch.value;
    return builtinChannelVisibility[key] !== false;
  });
  // 自定义通道：管控端定义的、当前可见的
  const customs = visibleCustomChannels.map((cc) => ({
    value: `admin_custom_${cc.id}`,
    label: cc.name,
    fields: cc.credentialFields.map(/* ... */),
    adminCustomMode: true,
    adminCustomId: cc.id,
    // ...
  }));
  return [...builtins, ...customs];
}, [builtinChannelVisibility, visibleCustomChannels]);
```

并维护一个 `channelLookup: Map<string, ChannelConfig>` 用于**已添加通道反查字段定义**（不受可见性变化影响，避免管控端关掉某通道导致已添加通道行失去字段元数据）。

#### 4.4.3 三种操作

##### 操作 1：添加通道

**入口**：「已接入通道（N）」标题右上角「+ 添加通道」按钮

**展开面板**：
- 通道下拉 `Select`：仅展示 `availableChannelOptions` 中**尚未被本 Agent 添加**的项目
- 根据所选通道的 `fields` 定义动态渲染输入框：
  - 普通字段（`secret: false`）：常规 `Input`
  - 加密字段（`secret: true`）：密码输入框 + `Eye/EyeOff` 切换明文
- 微信通道（`wechatMode: true`）：无字段，显示蓝色 `Info` 提示横幅「微信通道通过扫码授权接入，管控端仅创建占位记录，实际扫码绑定由租户在用户端完成。」

**校验**：
- 未选通道：「请选择要添加的通道」
- 必填字段为空：「请填写「字段名」」（精确定位）
- 重复添加：「「飞书」已添加，请勿重复」

**保存**：`toast.success("已添加通道「飞书」")`

##### 操作 2：编辑通道凭证（③b）

**入口**：点击已添加通道行头任意位置（`ChevronRight` 箭头展开 / 收起）

**展开后** —— 只读模式：
```
┌─────────────────────────────────────────┐
│ ›  飞书                            [×]   │  ← 行头（hover 显示删除按钮）
├─────────────────────────────────────────┤
│  飞书应用的App ID    cli_a1b2c3          │
│  飞书应用的App Secret  fsk••••••  [👁]   │
│                              [编辑凭证] │
└─────────────────────────────────────────┘
```

**点「编辑凭证」**进入编辑模式：所有字段变为 `Input`，加密字段提供 `Eye/EyeOff` 切换；底部「取消」「保存」按钮。

**密码遮罩规则**：`maskSecret(val)` = 前 3 字符 + `••••••`，空值显示 `—`

**保存**：校验必填后 `toast.success("「飞书」凭证已更新")`

**约束**：同时只展开一个通道，避免抽屉过长

##### 操作 3：移除通道

**入口**：行头 hover 时显示红色 `Trash2` 按钮

**确认**：弹出 `AlertDialog`（遵循 SKILL §10.4 危险操作规范），红色「确认移除」按钮

**确认文案**：
> 移除「飞书」后，该 Agent 将无法通过此通道收发消息。该操作不会删除通道下已有的凭证配置，可在用户端重新接入。

**反馈**：`toast.success("已移除通道「飞书」")`

#### 4.4.4 删除字段（相比 v0）

- ❌ 行头右侧「{fields.length} 项凭证」灰字提示

#### 4.4.5 跨页面同步示例

| 在管控端「通道配置」页操作 | 抽屉立即响应 |
|---|---|
| 打开「钉钉」的用户可见开关 | 添加通道下拉新增「钉钉」选项 |
| 关闭「飞书」的用户可见开关 | 添加通道下拉中「飞书」消失；但**已添加**的「飞书」行不受影响（仍可展开/编辑/删除） |
| 新增自定义通道「企业邮件」并设可见 | 添加通道下拉新增「企业邮件」选项 |
| 删除自定义通道 | 添加通道下拉同步移除 |

### 4.5 状态隔离

| 状态 | 隔离粒度 |
|---|---|
| `ClawDetail`（含已应用模型、已接入通道） | 按 `clawId` 隔离，存储在 `clawDetailMap: Record<string, ClawDetail>` |
| 编辑态（modelEditing / channelAdding / expandedChannel / channelEditDraft / visibleSecrets） | 切换实例时（`handleOpenDrawer`）全部重置 |
| 持久化 | 当前不持久化到 localStorage（Q4(a) 决议），刷新页面回到默认初始值 |

注：`modelConfigStore` 和 `customChannelStore` 的数据是**持久化**的，但**每个 Agent 自己的 `ClawDetail`** 不持久化。

---

## 5. 技术实现

### 5.1 文件变更清单

#### 新增
| 路径 | 类型 | 说明 |
|---|---|---|
| `client/src/lib/modelConfigStore.ts` | 新增 | 模型数据共享 store（localStorage + 事件广播 + React hooks） |

#### 修改
| 路径 | 改动点 |
|---|---|
| `client/src/pages/admin/ModelConfig.tsx` | 删除本地 `ModelRow`/`MOCK_MODELS`/`CUSTOM_PROVIDER_VALUE`；`useState` → `useAdminModelsState()` |
| `client/src/pages/admin/OpenClawMonitor.tsx` | 抽屉模型卡重写为厂商→版本两级 Select；通道下拉数据源切换为可见性过滤；删除「N 项凭证」；订阅 customChannelStore；新增 providerGroups / availableChannelOptions / channelLookup 三个 memo |
| `client/src/lib/agentConfigConstants.ts` | 无变化（继续作为通道定义来源） |
| `client/src/pages/tenant/OpenClawDetail.tsx` | 无变化（继续读 `agentConfigConstants` + `customChannelStore`） |

#### 不修改
- `client/src/App.tsx`
- `client/src/components/AdminLayout.tsx`

### 5.2 关键 Hook / Memo

```ts
// 抽屉内
const adminModels = useAdminModels();
const visibleAdminModels = useMemo(() => adminModels.filter(m => m.visible), [adminModels]);
const providerGroups = useMemo<ProviderGroup[]>(() => { /* 按 provider 组织，自定义模型置末 */ }, [visibleAdminModels]);

const [builtinChannelVisibility, setBuiltinChannelVisibility] = useState(() => loadBuiltinChannelVisibility());
const [visibleCustomChannels, setVisibleCustomChannels] = useState(() => loadVisibleCustomChannels());
useEffect(() => onBuiltinChannelVisibilityChange(() => setBuiltinChannelVisibility(loadBuiltinChannelVisibility())), []);
useEffect(() => onCustomChannelsChange(() => setVisibleCustomChannels(loadVisibleCustomChannels())), []);

const availableChannelOptions = useMemo<ChannelConfig[]>(() => { /* 内置过滤 + 自定义合并 */ }, [builtinChannelVisibility, visibleCustomChannels]);
const channelLookup = useMemo<Map<string, ChannelConfig>>(() => { /* 已添加通道反查表 */ }, [availableChannelOptions]);
```

### 5.3 边界与回退

| 场景 | 行为 |
|---|---|
| 模型配置页全部模型 `visible: false` | 编辑态显示琥珀色提示横幅，保存按钮 disable |
| 已应用模型 id 在 store 中找不到 | 只读态按 `appliedModel`/`appliedModelVersion` 冗余字段展示；编辑态回退到首组首项，无 toast |
| 通道配置页关闭某内置通道可见性 | 添加通道下拉中该通道消失；已添加的该通道行**不受影响**（依靠 `channelLookup` 反查 `CHANNEL_OPTIONS` 全集） |
| 自定义通道在管控端被删除 | 添加通道下拉中该通道消失；已添加且属于该自定义通道的行 fields 反查为空（凭证字段无法展示标签，但 `fieldValues` 仍保留）。**这是已知边界，本期接受**。 |
| 同一标签页内多次切换 Agent | 所有编辑态、密码可见性、展开状态全部清零 |
| 跨标签页打开模型配置 + 抽屉 | 模型配置页操作通过 `storage` 事件同步到抽屉 |

---

## 6. UI 规范遵守清单

参照 `SKILL.md`（OpenClaw Enterprise Design System）：

| 规范点 | 实现 |
|---|---|
| 卡片用原生 `<div>` | `bg-white rounded-2xl border border-gray-200` ✅ |
| 主按钮品牌渐变 inline style | `linear-gradient(135deg, #007AFF, #5856D6)` ✅ |
| 图标只用 lucide-react | `Pencil` / `Plus` / `Trash2` / `ChevronRight` / `Eye` / `EyeOff` / `AlertCircle` / `Info` ✅ |
| 危险操作用 AlertDialog | 移除通道时使用，红色"确认移除"按钮 ✅ |
| Toast 走 sonner | 所有保存 / 错误反馈 ✅ |
| 中文 UI | 全部按钮、提示文案使用简体中文 ✅ |
| 间距系统 | `p-4 / p-6 / gap-2 / gap-3 / space-y-2 / space-y-3` ✅ |
| 字号体系 | `text-sm font-semibold` 主信息 / `text-xs text-gray-400` 副信息 / `text-xs font-medium text-gray-600` 标签 ✅ |
| 等宽字体 | 密码遮罩值用 `font-mono` ✅ |
| 信息 / 警告横幅 | 蓝色 `bg-blue-50 border-blue-100` / 琥珀 `bg-amber-50 border-amber-100` ✅ |

---

## 7. 测试用例

### 7.1 模型编辑用例

| ID | 操作 | 预期 |
|---|---|---|
| TC-M-01 | 点击实例 instanceId 打开抽屉 | 已应用模型显示「腾讯云 DeepSeek / DeepSeek V3 0324」 |
| TC-M-02 | 点击「编辑」 | 弹出两级 Select，一级回填「腾讯云 DeepSeek」，二级回填当前版本对应记录 |
| TC-M-03 | 一级下拉打开 | 仅看到「腾讯云 DeepSeek」「腾讯云混元」「自定义模型」三项（不应有 `visible: false` 的厂商） |
| TC-M-04 | 切到「自定义模型」 | 二级下拉显示「OpenAI GPT-4o」 |
| TC-M-05 | 保存 | 只读态显示「自定义模型 / OpenAI GPT-4o」，无任何徽章；toast: "模型已更新" |
| TC-M-06 | 切换到另一个 Agent | 编辑态被重置；该 Agent 应用模型独立显示 |
| TC-M-07 | 另开标签页改模型配置：把「腾讯云 DeepSeek V3 0324」改为 `visible: false` | 回到抽屉点编辑，一级仅剩「腾讯云混元」「自定义模型」 |
| TC-M-08 | 已应用模型对应记录被 visible 关闭 | 只读态仍按原值显示；编辑态自动回退首组首项；不弹任何提示 |
| TC-M-09 | 模型配置页把所有模型 visible 全部关闭 | 编辑态展示琥珀色横幅，保存按钮 disabled |
| TC-M-10 | 模型配置页新增一条厂商「Anthropic Claude」并设可见 | 抽屉编辑态的一级下拉新增「Anthropic Claude」 |

### 7.2 通道编辑用例

| ID | 操作 | 预期 |
|---|---|---|
| TC-C-01 | 打开抽屉 | 已接入通道行只显示通道名（无「N 项凭证」灰字） |
| TC-C-02 | 点「+ 添加通道」 | 下拉展示「企业微信 / QQ / 企业微信应用 / 飞书 / 微信」，**不含**钉钉（默认不可见） |
| TC-C-03 | 选「钉钉」（前提：管控端开启钉钉可见） | 钉钉出现在下拉；选中后字段录入区出现 clientId / clientSecret |
| TC-C-04 | 选「微信」 | 显示蓝色 Info 提示「通过扫码授权由租户在用户端完成」，无字段输入框 |
| TC-C-05 | 填好飞书 appId/appSecret，点「确认添加」 | 列表新增飞书行；toast: "已添加通道「飞书」" |
| TC-C-06 | 必填字段为空时点保存 | toast: "请填写「飞书应用的App ID」" |
| TC-C-07 | 点已添加飞书行头展开 | 显示只读凭证表，appSecret 默认 mask（前 3 字符 + ••••••） |
| TC-C-08 | 点密码字段右侧 Eye 图标 | 该字段切换明文显示 |
| TC-C-09 | 点「编辑凭证」修改 appSecret 并保存 | 只读态显示更新后的 mask；toast: "「飞书」凭证已更新" |
| TC-C-10 | 点行头红色删除按钮 | 弹 AlertDialog，红色"确认移除"按钮 |
| TC-C-11 | 确认移除 | 列表移除该行；toast: "已移除通道「飞书」" |
| TC-C-12 | 另开标签页关闭飞书的"用户可见" | 抽屉「+ 添加通道」下拉中飞书消失；**已添加**的飞书行不受影响 |
| TC-C-13 | 管控端新增自定义通道「企业邮件」并开启可见 | 抽屉添加通道下拉新增「企业邮件」 |
| TC-C-14 | 自定义通道凭证字段录入 | 按管控端定义的 `credentialFields` 渲染输入框（默认全部加密） |

### 7.3 一致性 / 回归用例

| ID | 操作 | 预期 |
|---|---|---|
| TC-R-01 | 「模型配置」页所有原功能（增删改、设默认、应用范围等） | 行为完全不变 |
| TC-R-02 | 用户端 Agent 详情页（`/openclaw/:id`） | 模型 / 通道相关原有逻辑完全不变（共享常量仍来自 `agentConfigConstants.ts`） |
| TC-R-03 | 抽屉中"已安装技能"区 | 保持只读，未涉及修改 |
| TC-R-04 | 刷新页面 | 模型配置页的修改保留（localStorage 持久化）；Agent 自身的 ClawDetail 回到默认值（设计如此） |
| TC-R-05 | Vite HMR / TS check / ESLint | 全绿 |

---

## 8. 已知边界与未来扩展

### 8.1 已知边界
1. **自定义通道删除后已添加行字段元数据丢失**：管控端从「通道配置」删除某自定义通道后，已被 Agent 添加的该自定义通道行无法反查到 `credentialFields`，凭证字段标签将无法展示（但 `fieldValues` 数据本身保留）。本期接受。
2. **ClawDetail 不持久化**：刷新页面后每个 Agent 的应用模型 / 接入通道回到默认状态。这是 Q4(a) 决议的 demo 简化方式。
3. **一级厂商显示文案策略**：同 provider 的多条模型记录的 `name` 字段被假设一致，UI 取第一条的 `name` 作为厂商组 label。若管理员把同一 provider 的两条记录改成不同 `name`（异常用法），UI 仅显示其中一个。

### 8.2 未来可扩展
1. ClawDetail 升级为持久化（接 `openclawStore.ts` 或后端 API）
2. 模型选择支持"按应用范围（visibilityGroupIds）二次过滤"——只显示当前 Agent 所属用户组织允许的模型
3. 通道凭证支持"凭证模板"——管理员预设一组凭证，添加通道时一键选择
4. 自定义模型 / 自定义通道删除时弹"已被 N 个 Agent 使用"二次确认

---

## 9. 验收标准

- ✅ 抽屉中可在不离开页面的前提下完成「应用模型替换」「通道增 / 删 / 改」
- ✅ 模型编辑两级 Select 数据 100% 来自「模型配置」页 `visible: true` 的记录
- ✅ 通道添加下拉 100% 来自「通道配置」页用户可见的内置 + 自定义通道
- ✅ 「N 项凭证」灰字已移除
- ✅ 任一管控端配置页修改后，抽屉下拉**实时同步**（同标签页 + 跨标签页）
- ✅ 已应用模型 / 已添加通道被关闭可见性后，**不破坏**抽屉只读态展示
- ✅ ESLint / `pnpm check` / Vite HMR 全绿，未引入新错误
- ✅ 未修改 `App.tsx` 和 `AdminLayout.tsx`
- ✅ UI 严格遵守 SKILL.md 设计规范

---

## 10. 文档变更记录

| 版本 | 日期 | 作者 | 说明 |
|---|---|---|---|
| v1.0 | 2026-05-11 | feature/admin-openclaw-monitor | 首次发布 |
