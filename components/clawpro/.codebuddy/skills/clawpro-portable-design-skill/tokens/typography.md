# Typography

## 1. 字体栈

```css
--font-sans: 'PingFang SC', -apple-system, BlinkMacSystemFont, 'Helvetica Neue', sans-serif;
--font-mono: 'Menlo', 'Consolas', 'Courier New', monospace;
--font-din: 'DIN Alternate', 'DIN', 'Helvetica Neue', sans-serif;
--font-en: 'Open Sans', 'Helvetica Neue', sans-serif;
```

## 2. 使用原则

- 中文 UI 默认使用 `PingFang SC`
- 数字优先 `font-din` 或等宽数字排版
- 代码、路径、Token、实例 ID 使用 `font-mono`
- 不在业务页面随意新增 `fontFamily`

## 3. 文字色语义 token

当前文字色以运行时 `client/src/index.css` 与 `Typography.tsx` 的 `--text-*` 蓝灰 / slate 语义 token 为准：

| Tone | Token | Value | Usage |
|---|---|---:|---|
| `emphasis` | `--text-emphasis` | `#020617` | 强强调、关键数字、按钮文字、强标题 |
| `primary` | `--text-title` | `#0F172A` | 页面标题、模块标题、卡片标题 |
| `body` | `--text-body` | `#1E293B` | 普通正文、表格主内容 |
| `secondary` | `--text-secondary` | `#334155` | 描述、补充说明、表格次要字段 |
| `muted` | `--text-muted` | `#64748B` | 时间、备注、辅助信息、表头 |
| `weak` / `helper` | `--text-weak` | `#94A3B8` | 占位、空态、极弱提示、HelperText |
| `brand` | `--text-brand` | `#1447E6` | 链接、选中态、品牌强调 |
| `danger` | `--text-danger` | `#DC2626` | 删除、错误、危险操作 |

## 4. 推荐语义层级

| Semantic | Usage | Default tone |
|---|---|---|
| Page Title | 页面标题 | `primary` |
| Section Title | 区块标题 | `primary` |
| Panel Title | 面板 / Dialog / Card 标题 | `primary` |
| Body Text | 普通正文 | `body` |
| Meta Text | 辅助说明、时间、描述 | `muted` |
| Stat Number | 统计数字 | `emphasis` |
| Code Text | 代码、Token、路径 | `secondary` |

## 5. 宿主仓迁移建议

- 如果宿主仓已有 Typography 组件，优先做语义映射，不强制复制当前项目 API。
- 如果宿主仓没有语义文字组件，至少保证标题 / 正文 / Meta / 数字 / 代码文字这 5 层不混乱。

