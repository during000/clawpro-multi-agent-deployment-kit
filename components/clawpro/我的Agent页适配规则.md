# 「我的 Agent」页 - 卡片网格适配规则

> 文件：`client/src/pages/tenant/MyOpenClaw.tsx` + `client/src/components/agent/AgentCard.tsx`
> 核心原则：**不挤压、不半截裁切、横向滚动兜底**

---

## 1. 网格断点

```tsx
<div className="grid gap-5 grid-cols-2 min-[1420px]:grid-cols-3">
```

| 视口 | 列数 | 备注 |
|---|---|---|
| `< 1200px` | 2 列 + **横向滚动** | 由页面 `min-w-[1200px]` 兜底，禁止退化为 1 列 |
| `1200 ~ 1420px`（含 13 寸 MBP） | **2 列** | 每列 ≈ 470px |
| `≥ 1420px` | **3 列** | 每列 ≥ 380px |

---

## 2. 卡片本体

```tsx
<TenantCard className="min-w-[360px] ..." />
```

- **`min-width: 360px`** 硬约束 → 任何断点下「设置 + 对话」双按钮永不溢出卡片右边缘。

---

## 3. 时间字段：三档自适应降级（绝不出现省略号）

| 档 | 显示 | 触发 |
|---|---|---|
| 0 | `2026-04-04 14:00:00 创建` | 容器够宽 |
| 1 | `2026-04-04 创建` | 档 0 放不下 |
| 2 | `2026-04-04`（裸日期） | 档 1 也放不下（理论不会触发） |

不论展示哪档，**hover Tooltip 永远展示档 0 完整时间**。

### 3.1 实现要点

- ✅ **测量驱动**：容器内放 2 个隐藏 measure span（档 0 / 档 1 完整文本），`ResizeObserver` 监听容器宽度，挑能放下的最长档。
- ✅ **默认 tier=2**（最短）：`useLayoutEffect` 测完再切回更长档，避免挂载瞬间出现「半截裁切」。
- ✅ **Callback ref**（`ref={setContainer}`）：Radix `TooltipTrigger asChild` 会吞 `useRef`，必须用 callback ref。
- ✅ **Tooltip 必挂**：`delayDuration={200}`、`side="top"`、内容 = 完整 datetime + 业务后缀。
- ❌ **禁止** `<span className="truncate">` + CSS `text-overflow: ellipsis` 硬截。
- ❌ **禁止** 「仅时分」「仅月日」等中间档。
- ❌ **禁止** 写死像素阈值（如 `< 290px`）。

参考实现：`AgentCard.tsx` → `CreatedAtText` 组件。

### 3.2 代码骨架

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
             style={{ fontSize: 12, lineHeight: "20px", color: "var(--muted-foreground)" }}>
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
  return raw;
};
```

---

## 4. 复用范围

- 任何展示 `createdAt` / `updatedAt` / `lastActiveAt` 的卡片，**必须复用 `CreatedAtText` 同形组件**，不要再写裸 `truncate`。
- 推荐参数化签名：`<DatetimeText value={...} suffix="创建" dimmed={false} />`。
- 表格列场景由 data-table 列定义统一处理（列宽确定，不需要 measure），但 hover Tooltip 展示完整时间这一条仍然适用。

---

## 5. 验收清单

- [ ] 视口 1920 → 1500 → 1280 → 1100 拖动，时间永远是「完整 / `YYYY-MM-DD` 创建 / 裸日期」三态之一，**无省略号**。
- [ ] 任一态 hover 都有 Tooltip，内容为完整 datetime。
- [ ] 视口 < 1200px → 页面横向滚动（不是变 1 列）。
- [ ] 13 寸 MBP（1280）下卡片是 **2 列**（不是 3 列被挤压）。
- [ ] 任意断点下「设置 + 对话」双按钮完整可见、不溢出卡片。
- [ ] 异常态（shutdown / loadFail / createFail / pending）整段 `opacity-40`。
- [ ] 解析非常规 datetime 字符串时，UI 不出现空白；最差兜底为原字符串。

---

## 6. 已落地改动

| 文件 | 改动 |
|---|---|
| `client/src/pages/tenant/MyOpenClaw.tsx` | grid 改为 `grid-cols-2 min-[1420px]:grid-cols-3` |
| `client/src/components/agent/AgentCard.tsx` | TenantCard 加 `min-w-[360px]`；新增 `CreatedAtText` 三档降级 |
