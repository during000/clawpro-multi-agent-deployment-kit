# 🚨 常见错误排查手册 — 做完页面后对着这份表核对

> 产品做完一个页面后容易犯的 20 个错误。最后验收前对着这个清单逐项检查。

---

## 部分 1：颜色相关（5 个常见错）

### ❌ 错误 1：品牌蓝用错了

**症状**：看起来"太蓝"或"太深"，和 Figma 对不上

**原因**：
- 用的是其他蓝色（#007AFF / #0066FF / #4A90E2 等）
- 或者用了灰蓝混的色（#657A8C）

**修复**：
```tsx
// ❌ 错
<button style={{ backgroundColor: '#007AFF' }}>保存</button>

// ✅ 对
<button style={{ backgroundColor: 'var(--cp-brand-blue)' }}>保存</button>
```

**查验方法**：
1. 用取色器吸取按钮，应该是 **#1447E6**
2. 或者看 `references/foundation.md` 的"Admin 配色表"

---

### ❌ 错误 2：删除按钮用了蓝色

**症状**："删除"按钮和普通按钮一个颜色

**原因**：
- 删除按钮没有用 `variant="destructive"`
- 或者宿主仓不支持这个 variant，手写了蓝色

**修复**：
```tsx
// ❌ 错
<Button variant="claw-primary">删除</Button>

// ✅ 对
<Button variant="destructive">删除</Button>
```

**核实检查**：
- 删除按钮必须是 **红色 (#DC2626)**
- 或者是"白底红字"（删除确认之前是"幽灵"颜色）

---

### ❌ 错误 3：表格行背景没有分级

**症状**：表格行都是白色，看不出奇偶行

**原因**：
- 没有给奇数行/偶数行不同背景色
- 或者背景色选错了

**修复**：
```tsx
// ❌ 错
<tr style={{ backgroundColor: '#FFFFFF' }}>...</tr>

// ✅ 对
<tr style={{ backgroundColor: 'var(--bg-grey-normal)' }}>...</tr>  // 偶数行
<tr style={{ backgroundColor: 'var(--cp-surface)' }}>...</tr>      // 奇数行
```

**快速查验**：
- 表格列头：`--bg-grey-normal`（超浅灰 #F8FAFC）
- 表格行：奇偶轮换，或不轮换只有 hover 时亮
- Hover 时：`--bg-grey-hover`（浅灰 #F1F5F9）

---

### ❌ 错误 4：文字颜色散乱

**症状**：有的文字黑色、有的灰色、有的蓝色，没有统一

**原因**：
- 标题/正文/弱文字没有用对应 token
- 手写了灰色代码（如 #999 / #666 / #CCCCCC）

**修复**：
```tsx
// ❌ 错
<h2 style={{ color: '#0F172A' }}>标题</h2>
<p style={{ color: '#999999' }}>说明文字</p>
<a style={{ color: '#0066FF' }}>链接</a>

// ✅ 对
<h2 style={{ color: 'var(--text-title)' }}>标题</h2>
<p style={{ color: 'var(--text-weak)' }}>说明文字</p>
<a style={{ color: 'var(--cp-brand-blue)' }}>链接</a>
```

**快速查验**：
- 标题/重要：`--text-title` （接近黑 #0F172A）
- 正文：`--cp-text-title` （同上）
- 弱文字/说明：`--text-weak` 或 `--text-secondary`（灰 #64748B）
- 禁用/不可用：`--text-muted`（浅灰 #A0AEC0）
- 链接/品牌：`--cp-brand-blue`（蓝 #1447E6）

---

### ❌ 错误 5：描边颜色用错了

**症状**：表格边框或输入框边框看起来太深 / 太浅

**原因**：
- 用了主体颜色（#0F172A）或背景颜色
- 应该用专门的描边 token

**修复**：
```tsx
// ❌ 错
<div style={{ border: '1px solid #E0E0E0' }}>卡片</div>

// ✅ 对
<div style={{ border: '1px solid var(--cp-border)' }}>卡片</div>
```

**核实**：描边必须是 `--cp-border`（极浅灰 #E2E8F0）

---

## 部分 2：尺寸相关（5 个常见错）

### ❌ 错误 6：按钮高度不统一

**症状**：有的按钮 40px、有的 48px、有的 32px，各种尺寸混乱

**原因**：
- 没有统一按钮高度标准
- 或者混用了不同组件库的按钮

**修复**：
- **标准按钮**：36px（大多数场景）
- **小按钮**：32px（表格行内、工具条紧凑时）
- **不要用**：40px、48px、24px

**核实方法**：
1. 用浏览器开发者工具量一下高度
2. 所有正常操作按钮应该 36px
3. 表格行内的"编辑 / 删除"应该 32px

---

### ❌ 错误 7：圆角全乱套

**症状**：
- 有的按钮 8px 圆角、有的 12px、有的 24px（全圆角）
- 卡片里的子组件圆角和主卡片不一样

**原因**：Admin 端所有 UI 元素都应该 **4px** 圆角

**修复**：
```tsx
// ❌ 错
<button style={{ borderRadius: '8px' }}>按钮</button>
<div style={{ borderRadius: '12px' }}>卡片</div>
<div style={{ borderRadius: '50%' }}>头像</div>  // ✅ 头像除外

// ✅ 对（Admin）
<button style={{ borderRadius: '4px' }}>按钮</button>
<div style={{ borderRadius: '4px' }}>卡片</div>
<div style={{ borderRadius: '50%' }}>头像</div>  // 头像和 Switch 才用 50%
```

**核实清单**：
- [ ] 所有按钮都 4px（除非是头像 / Switch / 标签胶囊等特殊元素）
- [ ] 所有卡片都 4px
- [ ] 所有输入框都 4px
- [ ] 所有弹窗都 4px
- [ ] 只有头像、Switch、状态点、标签胶囊用 50%（full）

---

### ❌ 错误 8：表格行高不对

**症状**：表格行太紧 / 太松散，看起来不专业

**原因**：
- 表格行高没有统一标准
- 或者混用了不同的高度

**修复**：
- **表格数据行**：40px
- **表格列头**：可能 48px（看宿主仓规则）
- **不要用**：32px、36px、56px

**核实方法**：
1. 量表格行高，应该是 40px
2. 看 `component-specs/table.md` 对照自己的实现

---

### ❌ 错误 9：间距不规范

**症状**：元素间距各不相同，看起来散乱（6px、10px、13px、18px、22px 混用）

**原因**：没有遵循间距系统

**修复**：只用这些间距值：
- **8px** — 元素紧密相关（icon + text 之间）
- **12px** — 常见间距（按钮与按钮）
- **16px** — 卡片内部间距
- **24px** — 页面左右边距、大版块间距

```tsx
// ❌ 错
<div style={{ gap: '10px' }}><Button/><Button/></div>
<div style={{ margin: '15px' }}>内容</div>

// ✅ 对
<div style={{ gap: '12px' }}><Button/><Button/></div>
<div style={{ margin: '16px' }}>内容</div>
```

**核实清单**：
- [ ] 控件间距：12px
- [ ] 组件间距：16px
- [ ] 卡片内边距：16px
- [ ] 页面顶级内边距：24px
- [ ] 没有其他间距值

---

### ❌ 错误 10：输入框和按钮高度不匹配

**症状**：输入框和旁边的按钮一个高一个矮

**原因**：
- 输入框用了不同高度
- 或者按钮高度改了

**修复**：
- 输入框：36px 高（和标准按钮一致）
- 小输入框：32px 高（对应小按钮）

```tsx
// ❌ 错
<Input style={{ height: '40px' }} />
<Button style={{ height: '36px' }}>搜索</Button>

// ✅ 对
<Input style={{ height: '36px' }} />
<Button style={{ height: '36px' }}>搜索</Button>
```

---

## 部分 3：圆角相关（2 个常见错）

### ❌ 错误 11：客户端规则混到 Admin 里了

**症状**：Admin 页面有 12px 圆角 / 胶囊形按钮 / 模糊背景

**原因**：
- 复制了客户端（Tenant）的代码
- 或者设计稿来自客户端被混用了

**修复**：
1. 检查路径：是 `client/src/pages/admin/**` 还是 `client/src/pages/tenant/**`？
2. 如果是 Admin → 所有圆角必须 4px
3. 如果是 Tenant → 可以 12px（见 `references/tenant.md`）

```tsx
// ❌ 错（Admin 里不能这样）
<div style={{ borderRadius: '12px' }}>卡片</div>

// ✅ 对（Admin）
<div style={{ borderRadius: '4px' }}>卡片</div>

// ✅ 对（Tenant，不同规则）
<div style={{ borderRadius: '12px' }}>业务卡片</div>
```

---

### ❌ 错误 12：头像和状态点圆角错了

**症状**：头像看起来是方的、或状态点是方的

**原因**：头像和状态点应该 **100% 圆角（胶囊形）**

**修复**：
```tsx
// ❌ 错
<img style={{ borderRadius: '4px' }} />        // 头像不能 4px
<div style={{ borderRadius: '8px' }} />        // 状态点不能 8px

// ✅ 对
<img style={{ borderRadius: '50%' }} />        // 头像 100%
<div style={{ borderRadius: '50%' }} />        // 状态点 100%
```

---

## 部分 4：文字相关（3 个常见错）

### ❌ 错误 13：文字大小混乱

**症状**：标题 18px、数据 13px、按钮 15px，看起来不规范

**原因**：没有遵循 Typography 标准

**修复**：只用这些字号：
- **16px Semibold** — 页面大标题
- **14px Semibold** — 卡片标题 / 列表标题 / 按钮
- **14px Regular** — 正文 / 普通文字
- **12px Regular** — 表格数据 / 说明文字 / 辅助信息

```tsx
// ❌ 错
<h2 style={{ fontSize: '18px', fontWeight: '700' }}>标题</h2>
<p style={{ fontSize: '13px' }}>数据</p>
<button style={{ fontSize: '15px' }}>按钮</button>

// ✅ 对
<h2 style={{ fontSize: '16px', fontWeight: '600' }}>标题</h2>
<p style={{ fontSize: '12px' }}>数据</p>
<button style={{ fontSize: '14px' }}>按钮</button>
```

**核实清单**：
- [ ] 页面大标题：16px Semibold
- [ ] 卡片标题：14px Semibold
- [ ] 按钮文字：14px
- [ ] 表格数据：12px
- [ ] 说明文字：12px
- [ ] 没有 11px、13px、15px、18px、20px

---

### ❌ 错误 14：粗细不对

**症状**：文字看起来太细（light / 300 粗细）或太粗（bold / 800）

**原因**：只用了两个粗细，没有中间层

**修复**：只用这三个粗细：
- **600 (Semibold)** — 标题、重要信息
- **500 (Medium)** — 按钮文字、次级标题
- **400 (Regular)** — 正文、数据、说明

```tsx
// ❌ 错
<h2 style={{ fontWeight: '700' }}>标题</h2>
<button style={{ fontWeight: '400' }}>按钮</button>  // 按钮太细

// ✅ 对
<h2 style={{ fontWeight: '600' }}>标题</h2>
<button style={{ fontWeight: '500' }}>按钮</button>
```

---

### ❌ 错误 15：行高不对

**症状**：多行文字看起来挤（行高太小）或松散（行高太大）

**原因**：没有统一行高

**修复**：
- **单行标题**：1.2（16/12 = 1.33 左右）
- **多行正文**：1.5（最常用）
- **表格行**：1.4

```tsx
// ❌ 错
<h2 style={{ lineHeight: '1.0' }}>标题很长很长很长</h2>

// ✅ 对
<h2 style={{ lineHeight: '1.2' }}>标题很长很长很长</h2>
```

---

## 部分 5：组件相关（5 个常见错）

### ❌ 错误 16：KPI 卡自己拼了

**症状**：页面上有"图标 + 标题 + 大数字"的卡片，但用 SurfaceCard + 自定义 SVG + 内联数字拼的

**原因**：
- 不知道有 NumberCard 组件
- 或者宿主仓没有 NumberCard

**修复**：
1. 用 `NumberCard` 组件（见 `component-specs/number-card.md`）
2. 或者复制 `portable/react/number-card.tsx` 到宿主仓

```tsx
// ❌ 错
<SurfaceCard>
  <div style={{ display: 'flex' }}>
    <img src="icon.svg" />
    <div>
      <span>输入 Tokens</span>
      <strong style={{ fontSize: '24px' }}>1,234</strong>
    </div>
  </div>
</SurfaceCard>

// ✅ 对
<NumberCard 
  icon={<InputTokensIcon />} 
  label="输入 Tokens" 
  value="1,234" 
/>
```

---

### ❌ 错误 17：用了 Combobox（已废弃）

**症状**：组件名叫"Combobox"或"OpenClawCombobox"

**原因**：Combobox 已经并入 SearchableSelect，新代码不该还用旧名

**修复**：
1. 改用 `SearchableSelect`（或 `PortableSelect` 的 searchable 模式）
2. 查 `component-specs/combobox.md` 的迁移指引
3. 或复制 `portable/react/input-select.tsx` 里的 PortableSelect

```tsx
// ❌ 错（旧）
<Combobox options={...} />
<OpenClawCombobox options={...} />

// ✅ 对（新）
<SearchableSelect options={...} />
<PortableSelect searchable options={...} />
```

---

### ❌ 错误 18：弹窗用了 Dialog 而不是 AlertDialog

**症状**：删除、确认操作用的是普通 Dialog（两列按钮排列）

**原因**：
- 确认类对话框应该用 AlertDialog（左取消 / 右确认 / 右按钮红色）
- Dialog 是信息展示类

**修复**：
```tsx
// ❌ 错（删除用 Dialog）
<Dialog>
  <DialogHeader>确定删除？</DialogHeader>
  <DialogFooter>
    <Button>取消</Button>
    <Button variant="destructive">删除</Button>
  </DialogFooter>
</Dialog>

// ✅ 对（删除用 AlertDialog）
<AlertDialog>
  <AlertDialogHeader>确定删除？</AlertDialogHeader>
  <AlertDialogFooter>
    <AlertDialogCancel>取消</AlertDialogCancel>
    <AlertDialogAction variant="destructive">删除</AlertDialogAction>
  </AlertDialogFooter>
</AlertDialog>
```

见 `component-specs/dialog-drawer.md`

---

### ❌ 错误 19：空状态用了插画

**症状**：页面无数据时显示一个大插画（100×100 以上）

**原因**：
- Admin 端空状态是"文字为主 + 可选小图标"
- 大插画是客户端风格

**修复**：
```tsx
// ❌ 错（太装饰）
<div style={{ textAlign: 'center' }}>
  <img style={{ width: '200px' }} src="empty.svg" />
  <p>暂无数据</p>
</div>

// ✅ 对（Admin 风格）
<Empty>
  <EmptyHeader>
    <EmptyTitle>暂无数据</EmptyTitle>
    <EmptyDescription>还没有创建任何项目，<a>立即创建</a></EmptyDescription>
  </EmptyHeader>
</Empty>
```

见 `component-specs/empty-state.md`

---

### ❌ 错误 20：StatusTag 和 Badge 混用了

**症状**：
- 用 StatusTag 显示版本号 / 标签
- 用 Badge 显示运行状态 / 审核状态

**原因**：两个组件语义不同
- **StatusTag** —— 显示"状态"（运行中 / 已停止 / 失败）
- **Badge** —— 显示"标签"（v1.0 / 测试版 / 企业版）

**修复**：
```tsx
// ❌ 错
<Badge>运行中</Badge>       // 运行状态不该用 Badge
<StatusTag>v2.0</StatusTag> // 版本不该用 StatusTag

// ✅ 对
<StatusTag>运行中</StatusTag>
<Badge>v2.0</Badge>
```

见 `component-specs/status-tag.md` vs `component-specs/badge.md`

---

## 最终核对清单（提交前必做）

在把代码推上去前，对着这个清单逐项确认：

### 颜色 ✓
- [ ] 品牌蓝 = #1447E6（用 token `--cp-brand-blue`）
- [ ] 删除/危险 = #DC2626（用 `variant="destructive"`）
- [ ] 文字颜色用了 token（不是硬编码）
- [ ] 表格行有背景色分层

### 尺寸 ✓
- [ ] 按钮高度 = 36px（标准）或 32px（小）
- [ ] 输入框高度 = 36px（与按钮一致）
- [ ] 表格行高 = 40px
- [ ] 所有圆角 = 4px（Admin）或 12px（Tenant）
- [ ] 头像/Switch/点 = 100% 圆角
- [ ] 间距 ∈ {8, 12, 16, 24} px

### 文字 ✓
- [ ] 标题 = 16px Semibold / 14px Semibold
- [ ] 正文 = 14px Regular
- [ ] 数据 = 12px Regular
- [ ] 粗细 ∈ {400, 500, 600}
- [ ] 行高 1.2~1.5（根据内容）

### 组件 ✓
- [ ] KPI 卡用了 NumberCard
- [ ] 删除对话框用了 AlertDialog
- [ ] 空状态没有大插画
- [ ] StatusTag 用于"状态"，Badge 用于"标签"
- [ ] 搜索选择用 SearchableSelect（不用旧 Combobox）

### 整体 ✓
- [ ] 和 Figma 对标（如果有设计稿）
- [ ] 对照 `qa/admin-checklist.md` 逐项验收
- [ ] 截图给设计看一遍

---

**做完了？提交吧！🚀**
