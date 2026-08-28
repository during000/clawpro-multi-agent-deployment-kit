# Colors

## 1. 当前推荐主色

| Token | Value | Usage |
|---|---|---|
| Brand Blue | `#1447E6` | 主色、链接、活跃态、品牌强调 |
| Brand Black | `#020617` | CTA 渐变起点、强调文字、主按钮深色 |
| Brand Blue Tint | `#EFF6FF` | 活跃菜单弱背景、弱强调底 |

## 2. 语义色

| Token | Value | Usage |
|---|---|---|
| Success | `#16A34A` | 成功、运行中 |
| Danger | `#DC2626` | 危险、错误、删除 |
| Warning | `#F59E0B` | 警告、待处理 |
| Status Info Bg | `#E8ECFE` | 信息型状态标签浅底 |
| Surface | `#FFFFFF` | 卡片、浮层、输入控件背景 |
| Page Bg | `#F7F9FC` | portable fallback 页面背景 |
| Bg Subtle | `#FAFBFD` | 表头、hover、弱背景 |
| Border | `#EAEEF4` | 蓝灰描边；卡片、表格、分割线、面板、Input / Select / DatePicker 默认描边 |
| Border Control | `#C8CFDA` | Checkbox / Radio 等可勾选控件默认描边 |
| Tenant Card Border | `#E2E8F0` | TenantCard normal 默认描边 |

## 3. 文本色

文本色使用运行时 `--text-*` 蓝灰 / slate 语义 token；portable fallback 可用同名 `--cp-text-*` 变量映射。

| Token | Value | Usage |
|---|---|---|
| `--text-emphasis` | `#020617` | 强强调、关键数字、按钮文字、强标题 |
| `--text-title` | `#0F172A` | 页面标题、模块标题、卡片标题 |
| `--text-body` | `#1E293B` | 普通正文、表格主内容 |
| `--text-secondary` | `#334155` | 描述、补充说明、表格次要字段 |
| `--text-muted` | `#64748B` | 时间、备注、辅助信息、表头 |
| `--text-weak` | `#94A3B8` | 占位、空态、弱提示、HelperText |
| `--text-brand` | `#1447E6` | 链接、选中态、品牌强调 |
| `--text-danger` | `#DC2626` | 删除、错误、危险操作 |

## 4. CTA 渐变

```css
linear-gradient(90deg, #020617 70%, #1447E6 100%)
```

## 5. 使用原则

- 优先使用这套 token，而不是继续引入旧品牌色方案。
- 只有 token 定义文件、基础 token 表和资产源文件允许出现必要 hex。
- 组件 spec、page recipe、portable fallback 示例优先引用 token / CSS variable，不直接散写业务色 hex。
- 宿主仓如有自己的色彩 token，优先建立映射层，不要页面里直接散写新旧色值。
- Tenant / Landing 如有更具体差异，以端级规范补充，但不应推翻品牌主色体系。

