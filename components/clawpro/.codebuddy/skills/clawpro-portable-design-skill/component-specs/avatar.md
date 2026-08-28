# Avatar

## 1. Purpose

- 统一用户头像、Agent 头像的尺寸、圆角、fallback 规范。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：用户管理列表、Agent 卡片、会话头像、侧边栏底部用户区
- 不适用场景：Logo / 品牌图标（用 `<img>` + 设计资产）

## 3. Visual Standard

| Item | Value |
|---|---|
| 默认尺寸 | `32px × 32px` (size-8) |
| 圆角 | `rounded-full`（圆形） |
| Fallback 背景 | `var(--cp-bg-subtle)` (#F5F5F5) |
| Fallback 文字 | 14px / Medium / `var(--cp-text-muted)` |
| 常用尺寸变体 | `24px` (size-6) / `32px` (size-8) / `40px` (size-10) / `48px` (size-12) |

### 尺寸场景

| 尺寸 | 场景 |
|---|---|
| 24px | 行内/紧凑列表 |
| 32px | 表格行、侧边栏 |
| 40px | 卡片头部、详情页 |
| 48px | 个人中心、大头像 |

## 4. Portable Fallback

```tsx
function PortableAvatar({ src, name, size = 32 }: { src?: string; name: string; size?: number }) {
  const initials = name.slice(0, 2).toUpperCase();
  return (
    <span
      className="inline-flex items-center justify-center rounded-full bg-[var(--cp-bg-subtle)] overflow-hidden shrink-0"
      style={{ width: size, height: size }}
    >
      {src ? (
        <img src={src} alt={name} className="h-full w-full object-cover" />
      ) : (
        <span className="text-xs font-medium text-[var(--cp-text-muted)]">{initials}</span>
      )}
    </span>
  );
}
```

## 5. Do / Don't

**Do:**
- 无图片时显示首字母缩写。
- 使用圆形裁切。
- 尺寸从 4 种标准尺寸中选。

**Don't:**
- 不要用方形头像。
- 不要自定义尺寸（如 37px）。
- 不要在 fallback 里放 emoji。

## 6. QA Checklist

- [ ] 圆形裁切
- [ ] 尺寸从标准 4 档选取
- [ ] 无图片时有首字母 fallback
- [ ] fallback 背景使用 token

## 7. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`

## 代码对照（✅/❌）

### ❌ 错误：自由尺寸
```tsx
<Avatar className="w-9 h-9" />
<Avatar className="w-14 h-14" />
```
**为什么错**：尺寸不在 24/32/40/48 四档内，破坏视觉节奏。

### ✅ 正确：四档尺寸
```tsx
<Avatar size="sm" />   {/* 24px：表格行内 */}
<Avatar size="md" />   {/* 32px：列表/导航 */}
<Avatar size="lg" />   {/* 40px：卡片 */}
<Avatar size="xl" />   {/* 48px：详情页 */}
```

---

### ❌ 错误：方形圆角
```tsx
<Avatar className="rounded-md" src="..." />
```
**为什么错**：ClawPro Avatar 统一 `rounded-full`，方形会与 Logo/产品图标混淆。

### ✅ 正确：圆形
```tsx
<Avatar src={user.avatar}>
  <AvatarFallback>{user.name[0]}</AvatarFallback>
</Avatar>
```

---

### ❌ 错误：Fallback 渐变彩底
```tsx
<AvatarFallback className="bg-gradient-to-br from-blue-400 to-purple-500 text-white">
  {initials}
</AvatarFallback>
```
**为什么错**：彩色渐变与 ClawPro 克制风格冲突，且与 Tag 颜色语义混淆。

### ✅ 正确：弱色底 + 弱色文字
```tsx
<AvatarFallback className="bg-[var(--cp-bg-subtle)] text-[var(--cp-text-weak)] text-sm font-medium">
  {initials}
</AvatarFallback>
```

---

### ❌ 错误：Avatar 当按钮加 ring
```tsx
<Avatar
  className="cursor-pointer hover:ring-2 hover:ring-blue-500"
  onClick={openProfile}
/>
```
**为什么错**：Avatar 不应承担交互语义；hover ring 与 focus ring 互相干扰、a11y 不可达。

### ✅ 正确：button 包裹 Avatar
```tsx
<button
  type="button"
  onClick={openProfile}
  className="rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue)]"
  aria-label={`查看 ${user.name} 资料`}
>
  <Avatar size="md" src={user.avatar} />
</button>
```

---

### ❌ 错误：手写头像组负 margin
```tsx
<div className="flex">
  {users.map(u => <Avatar key={u.id} src={u.avatar} className="-ml-2 ring-2 ring-white" />)}
</div>
```
**为什么错**：每次重复实现 ring/overflow/+N 提示；超出数量也不会自动折叠。

### ✅ 正确：AvatarGroup
```tsx
<AvatarGroup max={3} size="md">
  {users.map(u => <Avatar key={u.id} src={u.avatar} alt={u.name} />)}
</AvatarGroup>
{/* 自动 -ml-2 / ring / +N 溢出 tooltip */}
```
