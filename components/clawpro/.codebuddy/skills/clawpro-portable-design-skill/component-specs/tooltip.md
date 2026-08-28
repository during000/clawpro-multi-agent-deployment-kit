# Tooltip

## 1. Purpose

- 统一 hover 提示浮层的视觉规范。
- 区分 Tooltip（短提示）和 Popover（长内容/可交互）。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：图标说明、截断文字完整展示、操作提示、禁用原因说明
- 不适用场景：需要用户交互的内容（用 Popover）、多行长说明（用 HoverCard 或 Popover）

## 3. Visual Standard

| Item | Value |
|---|---|
| 背景 | `#020617`（深黑） |
| 文字 | 白色 |
| 圆角 | `4px` |
| Padding | `px-3 py-1.5` |
| 字号 | `12px` (text-xs) / `leading-relaxed` |
| 最大宽度 | `240px`（超出换行） |
| 箭头 | 可选，指向触发元素 |
| 出现位置 | `top` 优先，空间不足时自动翻转 |

## 4. Portable Fallback

```tsx
function PortableTooltip({ children, content, side = 'top' }: any) {
  const [show, setShow] = React.useState(false);
  return (
    <span className="relative inline-flex" onMouseEnter={() => setShow(true)} onMouseLeave={() => setShow(false)}>
      {children}
      {show && (
        <span className={[
          "absolute z-50 max-w-[240px] rounded-[4px] bg-[#020617] px-3 py-1.5 text-xs text-white leading-relaxed whitespace-normal",
          side === 'top' ? "bottom-full left-1/2 -translate-x-1/2 mb-1.5" : "top-full left-1/2 -translate-x-1/2 mt-1.5",
        ].join(" ")}>
          {content}
        </span>
      )}
    </span>
  );
}
```

## 5. Do / Don't

**Do:**
- Tooltip 只放短文本说明（1~2 行）。
- 深黑底白字，高对比度。
- 禁用原因用 Tooltip 说明（如 Transfer 的 disabled 行）。

**Don't:**
- 不要在 Tooltip 里放可交互内容（用 Popover）。
- 不要用 `p-0` 重置 padding 后自定义间距。
- 不要放超长内容（超过 3 行改用 Popover）。
- 不要用浅色底（如白底灰字）。

## 6. QA Checklist

- [ ] 深黑底 + 白字
- [ ] 圆角 `4px`
- [ ] 内容不超过 2~3 行
- [ ] 无可交互内容（否则应改 Popover）
- [ ] fallback 使用 CSS variable

## 7. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related specs: `component-specs/popover-dropdown-menu.md`

## 8. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。Tooltip 5 项高频误用 → ClawPro 正确写法。

### 8.1 不要在 Tooltip 里放可交互内容

```tsx
// ❌ Tooltip 里放按钮，hover 离开就关闭，用户根本点不到
<Tooltip>
  <TooltipTrigger asChild><Info className="h-4 w-4" /></TooltipTrigger>
  <TooltipContent>
    <p>需要更多说明？</p>
    <Button size="sm" onClick={openDoc}>查看文档</Button>
  </TooltipContent>
</Tooltip>

// ❌ Tooltip 里塞链接 / Tabs / 表单
<TooltipContent>
  <a href="/docs/quota">配额说明</a>
</TooltipContent>

// ✅ 短说明 → Tooltip
<Tooltip>
  <TooltipTrigger asChild><Info className="h-4 w-4 text-[var(--cp-text-weak)]" /></TooltipTrigger>
  <TooltipContent>每日配额上限 1000 次</TooltipContent>
</Tooltip>

// ✅ 需要可交互 → HoverCard / Popover
<Popover>
  <PopoverTrigger asChild><button className="text-sm underline">配额说明</button></PopoverTrigger>
  <PopoverContent className="w-80">
    <p className="text-sm mb-2">每日配额上限 1000 次，超出后会进入排队队列。</p>
    <Button size="sm" variant="link" onClick={openDoc}>查看完整文档</Button>
  </PopoverContent>
</Popover>
```

### 8.2 浅色底反例：必须深黑底白字

```tsx
// ❌ 白底灰字，对比度不达标
<TooltipContent className="bg-white text-gray-600 border border-gray-200">
  说明文字
</TooltipContent>

// ❌ 用品牌蓝底，与选中态视觉混淆
<TooltipContent className="bg-[var(--cp-brand-blue)] text-white">
  说明文字
</TooltipContent>

// ✅ 深黑 #020617 底 + 白字（默认值，无需 className）
<TooltipContent>说明文字</TooltipContent>
```

### 8.3 不要 p-0 重置 padding 后自己拼

```tsx
// ❌ 重置 padding 自定义间距，破坏一致性
<TooltipContent className="p-0">
  <div className="px-2 py-1">说明文字</div>
</TooltipContent>

// ❌ 改成 px-4 py-3 让 tooltip 像小卡片
<TooltipContent className="px-4 py-3 text-sm">
  说明文字
</TooltipContent>

// ✅ 默认 px-3 py-1.5 + text-xs，不覆盖
<TooltipContent>说明文字</TooltipContent>
```

### 8.4 长文本 → HoverCard / Popover

```tsx
// ❌ Tooltip 塞 5 行说明，超过 240px 反复换行
<TooltipContent className="max-w-md">
  本配额规则用于控制单租户每日的最大 API 调用次数。当达到上限时，
  系统将拒绝后续请求并返回 429。配额会在 UTC+8 0 点自动重置，
  企业版用户可以联系商务调整阈值。详见配置中心 - 配额管理。
</TooltipContent>

// ✅ HoverCard 承载长说明（hover 也能阅读，鼠标可移入）
<HoverCard>
  <HoverCardTrigger asChild>
    <Info className="h-4 w-4 text-[var(--cp-text-weak)] cursor-help" />
  </HoverCardTrigger>
  <HoverCardContent className="w-80">
    <p className="text-sm leading-relaxed text-[var(--cp-text-body)]">
      本配额规则用于控制单租户每日的最大 API 调用次数。当达到上限时，
      系统将拒绝后续请求并返回 429。配额会在 UTC+8 0 点自动重置。
    </p>
  </HoverCardContent>
</HoverCard>
```

### 8.5 Disabled 元素的提示：用包裹 span 触发

```tsx
// ❌ 直接给 disabled Button 加 Tooltip，按钮 disabled 时不响应 pointer 事件，hover 不出
<Tooltip>
  <TooltipTrigger asChild>
    <Button disabled>解除绑定</Button>
  </TooltipTrigger>
  <TooltipContent>该用户为最后一位管理员，无法解除</TooltipContent>
</Tooltip>

// ✅ 外包一层 span 承担 hover 事件
<Tooltip>
  <TooltipTrigger asChild>
    <span tabIndex={0} className="inline-block">
      <Button disabled className="pointer-events-none">解除绑定</Button>
    </span>
  </TooltipTrigger>
  <TooltipContent>该用户为最后一位管理员，无法解除</TooltipContent>
</Tooltip>
```
