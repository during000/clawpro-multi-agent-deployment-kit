# FileBrowser（多版本资产文件浏览器）

## 1. Purpose

- 统一「资产包内文件 + 版本」的浏览体验：版本切换、文件树、文件内容查看（Preview / Source 双模式）。
- 替代 `Skill / Plugin / MCP` 详情页里各自手写的"版本侧栏 + 文件树 + 内容面板"三栏布局。
- 与 `upload-file-browser.md` 的差异：本组件**只读浏览**，不负责上传 / 拖拽 / 进度，二者职责互不重叠。

## 2. Scope

- 适用端：Admin（Skill / Plugin / MCP 详情页）。Tenant 暂不使用。
- 必用场景：可版本化资产包的内容预览（Skill 包、Plugin 包、MCP 包、未来的 Agent 配置包）。
- 不适用场景：单文件预览（用普通 `<pre>` 或 MDX 渲染）；上传场景（用 `upload-file-browser.md`）；带编辑能力的文件管理（业务自定义）。

## 3. Source Files

| 文件 | 用途 |
|------|------|
| `client/src/components/ui/file-browser.tsx` | FileBrowser 核心组件（三栏布局） |
| `client/src/components/ui/tree.tsx` | FileTree 子组件（中间栏文件树） |
| `client/src/components/ui/Typography.tsx` | 内部使用的 BodyMedium / MetaText / TinyText / MetaMedium |
| `client/src/pages/admin/SkillLibrary/SkillDetail.tsx` | 真实业务用例（含下载） |
| `client/src/pages/admin/SkillLibrary/PluginDetail.tsx` | 真实业务用例（含 onVersionChange） |
| `client/src/pages/admin/SkillLibrary/MCPDetail.tsx` | 真实业务用例（含 changeLog tooltip） |

## 4. Visual Standard

### 4.1 整体框架

| Item | Value | Notes |
|------|-------|-------|
| 容器 | `border` + `rounded-[var(--radius-lg)]` | 8px 圆角，外描边 `var(--border)` |
| 背景 | `bg-white` | 默认白底，子栏自带分隔线 |
| 默认高度 | `47rem`（约 752px） | 可通过 `height` prop 覆盖 |
| 布局 | `flex` 三栏，左 14% / 中 22% / 右 flex-1 | 最小宽度：左 120px / 中 160px |
| 栏间分隔 | `border-r border-[var(--border)]` | 1px 实线 |
| 栏头部行高 | `h-12 px-3`（48px 高，左右 12px） | 三栏头部行高一致 |

### 4.2 左栏：版本列表

| Item | Value | Notes |
|------|-------|-------|
| 项内边距 | `px-3 py-2.5` | 12px / 10px |
| 项分隔线 | `border-b border-[#f4f4f5]` | 极淡灰，与外框区分 |
| 选中态背景 | `bg-[#f4f4f5]` | hover 同色 |
| 版本号字体 | `BodyMedium` + `tone="emphasis"` + `font-semibold` | 14px / 600 |
| 日期字体 | `MetaText` + `tone="weak"` + `mt-0.5` | 12px / weak |
| Latest 标签 | `h-[18px]` / `border #1447E6` / `rounded-[2px]` / `px-1` | 蓝边框 + 蓝字 `TinyText tone="brand"` |
| changeLog 入口 | 右侧 `Info` 图标 12×12 / `text-weak` → hover `text-secondary` | Tooltip 触发 |
| changeLog Tooltip | `max-w-[260px]` / `p-3` / `bg-white` / `border` / `shadow-lg` / `side="right"` | 标题 `MetaMedium`，内容 `MetaText` `whitespace-pre-line` |

### 4.3 中栏：文件树

| Item | Value | Notes |
|------|-------|-------|
| 头部 | 左：当前版本号（`BodyMedium emphasis`）；右：可选下载按钮 | 下载按钮 `text-muted` → hover `text-title` |
| 下载图标 | `Download` 14×14；加载中 `Loader animate-spin` 14×14 | 走 `showDownload + onDownload + isDownloading` 三件套 |
| 树容器 | `flex-1 overflow-y-auto px-3 py-2` | 默认全部展开 `defaultExpandAll={true}` |
| 可查看后缀 | 内置白名单（见下） | 不在白名单的文件不可点击查看 |

**内置可查看后缀（`isViewable`）**：

```
.md .mdx .xml .json .txt .yaml .yml .toml .ini .cfg .conf
.sh .bat .py .js .ts .tsx .jsx .css .html .htm .svg
.env .gitignore .dockerfile
（无扩展名/无路径分隔的纯名称视为目录占位，可点击）
```

> 业务侧若需扩充白名单，需扩展组件而非临时 hack。

### 4.4 右栏：文件内容

| Item | Value | Notes |
|------|-------|-------|
| 头部左 | 文件名 `BodyMedium emphasis` | 14px / 600 |
| 头部右：模式切换器 | 容器 `bg-[var(--bg-grey-hover)]/60 rounded p-0.5` 内含 2 个按钮 | 宽 0.5 间隙 |
| 模式按钮 | `px-2 py-1 rounded` + 选中态 `bg-white shadow-sm` | 图标 12×12（`Eye` / `Code`） |
| 选中文字 | `MetaText` + `tone="primary"` + `font-medium` | 12px / 500 |
| 未选中文字 | `MetaText` + `tone="muted"` | 12px / 弱化 |
| 内容区 | `flex-1 overflow-y-auto` | 滚动条由内容自己出 |
| Source 渲染 | `<pre>` + `text-xs leading-5 font-mono` + `whitespace-pre-wrap break-words` + `bg-[var(--bg-subtle)] p-3` | 等宽字体 / 软换行 |
| Markdown Preview | 占位提示 `Markdown preview requires MDXRenderer` | 业务侧需自行注入 MDXRenderer |
| 空文件 | 居中 `BodyText tone="weak"` "No file content" | — |
| 未选文件 | 居中 `BodyText tone="muted"` "Select a file to view content" | — |

## 5. API

```ts
export interface VersionInfo {
  version: string;
  date: string;
  isLatest?: boolean;
  changeLog?: string;
}

export interface FileBrowserProps {
  versions: VersionInfo[];           // 必填，至少 1 个
  files: FileEntry[];                // 必填，FileTree 数据（来自 ./tree）
  getFileContent: (fileName: string) => string | undefined;  // 必填，按文件名取内容

  // 可选
  height?: string;                   // 默认 "47rem"
  showDownload?: boolean;            // 默认 false
  onDownload?: (version: string) => void;
  isDownloading?: boolean;
  defaultVersion?: string;           // 默认 versions[0].version
  defaultFile?: string;              // 默认优先 SKILL.md / 否则 files[0]
  className?: string;
  onVersionChange?: (version: string) => void;
}
```

## 6. Default Behaviors

| 触发 | 默认行为 |
|------|----------|
| 首次渲染 | 选中第一个版本 + 优先选中 `SKILL.md` 文件（fallback：第一个文件） |
| 选中 `.md` / `.mdx` 文件 | 自动切到 Preview 模式 |
| 选中其他文件 | 自动切到 Source 模式 |
| 切换版本 | 内部 state 同步 + 触发 `onVersionChange`（父组件可同步业务态如选中版本号） |
| 切换文件 | 内部 state 同步 + 自动按文件类型重置 viewMode |

## 7. Usage Guidelines

### ✅ 推荐做法

1. **多版本资产包详情页**直接用 FileBrowser，不要再手写"版本栏 + 文件树 + Tab 内容"组合。
2. **`getFileContent` 在父组件做"按文件名 → 内容"的映射**，FileBrowser 不缓存内容，父组件可控加载策略（同步 / 异步 / 兜底）。
3. **下载按钮**只在确实可下载（zip / tar 包）时开启 `showDownload`，否则保持隐藏。
4. **changeLog**：版本若有更新说明，填进 `VersionInfo.changeLog`，自动出现 Info Tooltip；不要在版本名后拼接说明文本。
5. **Markdown 渲染**：业务若需真实渲染 markdown，应包一层注入 MDXRenderer 的 wrapper（当前组件只输出占位提示）。

### ❌ 禁止做法

1. ❌ 手写 `flex border rounded` 三栏代替 FileBrowser。
2. ❌ 把版本切换逻辑放在 FileBrowser 外（用 Tabs 切版本），破坏组件内部联动。
3. ❌ 自行覆盖 `var(--border)` 描边色或 `rounded-[var(--radius-lg)]` 圆角。
4. ❌ 在 `getFileContent` 里发 fetch 请求（每次 viewMode/file 切换都会重新调用，性能问题）。需异步加载时父组件自行 useEffect 预热缓存。
5. ❌ 修改三栏宽度比例（14% / 22% / flex-1）；最小宽度 120px / 160px 已经是临界值，再小会破坏交互。

## 8. Coverage（页面覆盖）

| 页面 | 用法亮点 |
|------|----------|
| `admin/SkillLibrary/SkillDetail.tsx` | 含 `showDownload + onDownload + isDownloading` 完整下载链路 |
| `admin/SkillLibrary/PluginDetail.tsx` | 含 `defaultVersion + onVersionChange`，与父组件双向同步选中版本 |
| `admin/SkillLibrary/MCPDetail.tsx` | 含 `versionsForBrowser` 构造，含 changeLog tooltip |

迁移建议：未来若新增 Agent 配置包详情页 / 镜像版本预览页，**第一选择是 FileBrowser**。

### 8.1 HTML/CSS fallback

见 `portable/html-css/file-browser.html`（含 3 个 demo：默认 Preview / Source 模式 / 边界态）。跨仓无 React 时直接复制该 HTML + `portable/css/tokens.css` 即可还原视觉。

## 9. Related Components

| 组件 | 关系 |
|------|------|
| `tree.tsx`（FileTree） | FileBrowser 中间栏的实现底层，独立 spec 见 `tree.md` |
| `upload-file-browser.md` | 上传 / 拖拽场景；与本组件正交，不要混用 |
| `dialog-drawer.md` | 当文件浏览器作为 Drawer 内容时遵循 Drawer 容器规范 |
| `tabs.md` | 文件浏览器常作为详情页 page header 下方一个 LineTab 的内容；外层一级 Tab 沿用 LineTabs 规范 |
| `segment.md` | 文件浏览器内部"全部 / 我的"等卡片内切换沿用 Segment 规范 |
