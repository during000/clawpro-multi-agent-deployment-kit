# Datetime Display 规范（卡片/列表中"创建时间·最近活跃·任意 datetime"展示）

> 适用范围：所有在响应式卡片 / 列表底部行 / 信息条里展示 `createdAt`、`updatedAt`、
> `lastActiveAt`、"创建时间"、"最近活跃" 等 datetime 字符串的场景。
> 配套：网格断点规则 + 卡片 min-width + 三档自适应降级 + Tooltip。
>
> 反例触发关键词："`2026-04-04 14:00:...`"、`2026-04-0...`、半截 ellipsis、窄屏被裁、
> 卡片挤压、按钮溢出卡片、窄屏退化成单列。
>
> 本规范优先级高于各业务页内联 `truncate`；与 `tooltip.md` 配合使用。
> 首个落地参考实现：`client/src/components/agent/AgentCard.tsx`。

---

## 0. 设计原则（背后的"为什么"）

1. **信息密度优先**：时间是用户判断"哪条记录是哪个、哪个最新"的关键信息，不允许下掉。
2. **绝不出现省略号**：`2026-04-04 14:00:...` / `2026-04-0...` 这类半截裁切**禁止出现**。
   要么完整、要么干净降级到下一档。
3. **窄屏不挤压，用横向滚动**：视口 < 1200px 不再尝试单列垂直堆叠，由页面 `min-w-[1200px]`
   直接撑出横向滚动条。这样信息布局始终保持"网格化"心智模型，避免 1 列时卡片一字排开像列表。
4. **按钮不溢出**：底部行操作按钮在所有断点下都必须完整可见。卡片 `min-w-[360px]`
   是这个不变量的根保证。

---

## 1. 终态规则总览（一张表看完）

### 1.1 卡片网格断点

```tsx
<div className="grid gap-5 grid-cols-2 min-[1420px]:grid-cols-3">
```

| 视口宽度 | 列数 | 每列约宽 | 行为说明 |
|---|---|---|---|
| `< 1200px` | **保持 2 列**，触发**横向滚动** | — | 由页面 `min-w-[1200px]` 兜底，**禁止降级到 1 列** |
| `1200 ~ 1420px`（含 13 寸 MBP 1280） | **2 列** | ≈ 470px | 时间 + 双按钮舒展，不挤压 |
| `≥ 1420px` | **3 列** | ≥ 380px | 时间 + 双按钮舒展，不挤压 |

### 1.2 卡片本体

```tsx
<TenantCard className="group relative min-w-[360px] ..." />
```

| 属性 | 值 | 备注 |
|---|---|---|
| `min-width` | **`360px`** | 单卡硬约束，保证按钮组永不溢出卡片右边缘 |
| 内边距 | 由 `<TenantCard padding="default">` 控制：`p-5 + gap-6` | 不要业务内联覆写 |

### 1.3 时间字段：三档自适应降级（绝不出现省略号）

| 档 | 显示 | 触发条件 |
|---|---|---|
| 0 | `2026-04-04 14:00:00 创建` | 容器宽度 ≥ 完整文本宽度 |
| 1 | `2026-04-04 创建` | 档 0 放不下，但日期+创建能放下 |
| 2 | `2026-04-04`（裸日期） | 档 1 也放不下（极端情况，理论不会触发） |

不论展示哪档，**hover Tooltip 永远展示档 0 完整时间**。

---

## 2. 时间字段实现规则

### 2.1 必须做

1. **测量驱动降级**，禁止 CSS 硬截。
   - 容器内放 2 个隐藏 measure `<span>`（档 0 / 档 1 完整文本）：
     `position:absolute`、`visibility:hidden`、`pointer-events:none`、`whiteSpace:nowrap`。
   - 容器内字体上下文（`font-family/size/weight`）显式声明，measure 节点继承，
     保证测量与渲染完全一致。
   - `ResizeObserver` 监听容器尺寸，挑能放下的最长档。
2. **默认 tier=2（最短）**，`useLayoutEffect` 测完后再切回更长档。
   这样组件挂载瞬间不会闪现"完整文本被半截裁切"。
3. **`ref` 用 callback ref**（`ref={setContainer}`）。Radix `TooltipTrigger asChild`
   通过 cloneElement 转发 ref，直接 `useRef` 可能被吞，必须用 callback ref 拿到真实 DOM。
4. **Tooltip 必挂**：`delayDuration={200}`、`side="top"`、`className="text-xs"`，
   内容恒等于完整 datetime + 业务后缀（"创建" / "活跃" 等）。
5. **样式 token**：`font-size:12px`、`line-height:20px`、`color: var(--muted-foreground)`，
   字体族 `PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif`。
   异常态（`shutdown / loadFail / createFail / pending`）整段 `opacity-40`。

### 2.2 严禁

- ❌ 直接 `<span className="truncate">{createdAt} 创建</span>`，靠 CSS `text-overflow: ellipsis` 截断。
- ❌ 引入"仅时分""仅月日"等中间档展示。三档：完整 / 仅日期+创建 / 裸日期，没有别的。
- ❌ 用固定断点（如 `xl:hidden`）藏时分秒——不同卡片宽度阈值不一样，必须靠测量。
- ❌ 把降级阈值写死成像素值（如 `< 290px`）——卡片 padding / 兄弟节点宽度变化都会让阈值失效。
- ❌ 网格在窄屏退化为 1 列。视口 < 1200px 必须由页面 `min-w-[1200px]` 触发横向滚动。

### 2.3 参考实现

文件：`client/src/components/agent/AgentCard.tsx` 的 `CreatedAtText` 组件。
对外签名：

```tsx
<CreatedAtText createdAt={claw.createdAt} dimmed={isDimmedText} />
```

骨架：

```tsx
const CreatedAtText = ({ createdAt, dimmed }: { createdAt: string; dimmed: boolean }) => {
  const [container, setContainer] = useState<HTMLDivElement | null>(null);
  const fullRef = useRef<HTMLSpanElement>(null);
  const dateRef = useRef<HTMLSpanElement>(null);
  const [tier, setTier] = useState<0 | 1 | 2>(2); // 默认最短

  const dateOnly = extractDateOnly(createdAt);
  const fullText = `${createdAt} 创建`;
  const dateText = `${dateOnly} 创建`;

  useLayoutEffect(() => {
    if (!container || !fullRef.current || !dateRef.current) return;
    const check = () => {
      const available = container.clientWidth;
      if (fullRef.current!.offsetWidth <= available + 1) setTier(0);
      else if (dateRef.current!.offsetWidth <= available + 1) setTier(1);
      else setTier(2);
    };
    check();
    const ro = new ResizeObserver(check);
    ro.observe(container);
    return () => ro.disconnect();
  }, [container, fullText, dateText]);

  const visible = tier === 0 ? fullText : tier === 1 ? dateText : dateOnly;

  return (
    <Tooltip delayDuration={200}>
      <TooltipTrigger asChild>
        <div ref={setContainer}
             className={`relative truncate min-w-0 flex-1${dimmed ? " opacity-40" : ""}`}
             style={{
               fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
               fontWeight: 400,
               fontSize: 12,
               lineHeight: "20px",
               color: "var(--muted-foreground)",
             }}>
          <span ref={fullRef} aria-hidden style={MEASURE_STYLE}>{fullText}</span>
          <span ref={dateRef} aria-hidden style={MEASURE_STYLE}>{dateText}</span>
          {visible}
        </div>
      </TooltipTrigger>
      <TooltipContent side="top" className="text-xs">{fullText}</TooltipContent>
    </Tooltip>
  );
};

const MEASURE_STYLE: React.CSSProperties = {
  position: "absolute",
  left: 0,
  top: 0,
  visibility: "hidden",
  pointerEvents: "none",
  whiteSpace: "nowrap",
};
```

`extractDateOnly` 兼容多种 datetime 入参 + 兜底：

```ts
const extractDateOnly = (raw: string): string => {
  if (!raw) return raw;
  const m = raw.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})/);
  if (m) {
    const [, y, mo, d] = m;
    return `${y}-${mo.padStart(2, "0")}-${d.padStart(2, "0")}`;
  }
  const t = new Date(raw);
  if (!isNaN(t.getTime())) {
    return `${t.getFullYear()}-${String(t.getMonth() + 1).padStart(2, "0")}-${String(t.getDate()).padStart(2, "0")}`;
  }
  return raw; // 兜底
};
```

---

## 3. 复用与扩展

- **同类字段（`updatedAt` / `lastActiveAt` / "最近活跃"）必须复用**本规范。
  推荐组件签名参数化：
  ```tsx
  <DatetimeText value={createdAt} suffix="创建" dimmed={false} />
  ```
- **新增卡片**（Skill 卡 / Channel 卡 / 模型卡 / 实例卡 …）若展示 datetime，
  禁止再写裸 `truncate`，必须复用 `CreatedAtText` / `DatetimeText`。
- **表格列**：表格场景由 `data-table.md` 列定义统一处理（列宽确定，不需要 measure），
  但 hover Tooltip 展示完整时间这一条仍然适用。

---

## 4. AI 执行指令（机器读这一节）

> 触发条件：看到任一卡片 / 列表行底部展示 datetime 字段，且容器宽度受响应式影响。
>
> 三步法，不要省略：
> 1. **替换**裸 `<span>{datetime}</span>` 为 `CreatedAtText` / `DatetimeText` 同形组件；
>    没有就照 §2.3 创建一个，**不要内联实现**。
> 2. **测量**：容器内必须存在 2 个隐藏 measure 节点（档 0 / 档 1）+ ResizeObserver；
>    禁止仅靠 CSS 截断。
> 3. **Tooltip 必挂**：内容 = 完整 datetime + 业务后缀；样式遵循 §2.1。
>
> 网格断点决策：默认照搬 §1.1（`grid-cols-2 min-[1420px]:grid-cols-3`），
> **禁止降级到 1 列**，让页面 `min-w-[1200px]` 触发横向滚动。
>
> 触发关键词：`createdAt`、`updatedAt`、`lastActiveAt`、"创建时间"、"最近活跃"、
> "时间被省略号"、"窄屏裁掉时间"、"卡片挤压"、"按钮溢出卡片"、
> 底部行 `mt-auto + flex justify-between`。

---

## 5. 验收清单（Code Review / QA 用）

- [ ] 视口 1920 → 1500 → 1280 → 1100 拖动，时间永远是 "完整 / `YYYY-MM-DD` 创建 / 裸日期"
      三态之一，**没有任何省略号**。
- [ ] 任一态下 hover 都有 Tooltip，展示完整 datetime + 业务后缀。
- [ ] 视口 < 1200px 时，**页面横向滚动**（不是卡片变 1 列）。
- [ ] 13 寸 MBP（视口 1280）下，卡片是 **2 列**（不是 3 列被挤压）。
- [ ] 异常态（shutdown / loadFail / createFail / pending）整段 `opacity-40`，
      字号 / 颜色 token 不变。
- [ ] 任意断点下，"设置 + 对话"双按钮始终完整可见，不溢出卡片右边缘。
- [ ] 解析非常规 datetime 字符串时，UI 不出现空白；最差兜底为原字符串。

---

## 6. 改动落地位置（首个参考实现）

| 文件 | 改动 |
|---|---|
| `client/src/pages/tenant/MyOpenClaw.tsx` | grid 断点改为 `grid-cols-2 min-[1420px]:grid-cols-3` |
| `client/src/components/agent/AgentCard.tsx` | TenantCard 加 `min-w-[360px]`；`CreatedAtText` 三档降级实现 |
