# Upload / File Browser

> ⚠️ **职责边界（与 `file-browser.md` 去重）**：本 spec 只管**上传链路**——上传入口 / 拖拽区 / 上传进度 / 失败重试 / **已上传文件的列表展示**。
> **多版本资产包的「只读浏览」（版本侧栏 + 文件树 + 内容预览三栏）不在本 spec**，请见 `file-browser.md`（对应 `client/src/components/ui/file-browser.tsx`），二者职责正交、不要混用。
> 注：两份 spec 文件名相似易混（`upload` 写入 vs `file-browser` 只读），改名（`upload.md` / `asset-browser.md`）涉及 HTML/CSS demo 与 verify 脚本，已在 `references/conflict-log.md` 记录、留单独 PR 处理。

## 1. Purpose

- 统一**上传入口、拖拽区域、上传进度、失败/重试、上传后的文件列表与文件空态**。
- 避免上传区沿用默认 Upload 图标、过大虚线框、旧色值或无权限 / 失败态缺失。

## 2. Scope

- 适用端：Admin / Tenant / Shared。
- 必用场景：文件上传、图片上传、文档上传、上传进度、上传后的文件列表 / 选择。
- 不适用场景：**多版本资产包只读浏览 → `file-browser.md`**；纯展示/导航目录树 → `tree.md`；完整资源管理后台的信息架构（本 spec 只定义上传相关组件视觉和状态）。

## 3. Visual Standard

| Item | Default | Notes |
|---|---|---|
| Upload Dropzone | 白底、4px、dashed border | 使用 `--cp-border` |
| Upload Icon | 已登记资产或 lucide icon | 不用 emoji 和默认 Upload 大图标 |
| Text | 标题具体，描述说明限制 | 不只写“上传文件” |
| File Row | 36px-44px | 文件名 truncate，meta 弱化 |
| Progress | 4px-6px 品牌蓝 | 与 `loading-progress.md` 一致 |
| Error | danger 文本 + 重试 / 删除 | 不只 toast |
| Empty | 轻量 Empty | 页面级可用插画，紧凑容器不放大图 |
| Actions | 上传 / 删除 / 重试清楚分组 | 删除需确认或可撤销 |

## 4. Anatomy

```text
UploadArea
  Icon / Illustration
  Title
  Description
  Action

UploadedFileList（上传后的文件列表，非多版本只读浏览器）
  Toolbar optional
  FileRow（icon + name + progress + 取消/重试）
  Empty / Loading / Error
```

> 「多版本资产只读浏览」的三栏结构（版本 / 文件树 / 内容）见 `file-browser.md`，不在本 Anatomy 内。

## 5. States

- idle: 等待选择或拖拽文件。
- drag-over: 边框品牌蓝弱强调。
- uploading: 文件行显示进度。
- uploaded: 显示成功状态和文件 meta。
- failed: 显示失败原因、重试和删除。
- invalid-type: 文件类型不符合要求。
- too-large: 文件超出限制。
- empty: 无文件时说明原因和下一步。
- loading: 文件列表加载中。
- no-permission: 说明需要权限。

## 6. Demo Repo Usage

- 进度：`client/src/components/ui/progress.tsx`
- 空态：`client/src/components/ui/empty.tsx`
- 资产规则：`references/assets-icons.md`（图标登记：当前项目以 `client/src/design-assets/resource-skill-map.json` 为准，跨仓样例 `assets/icon-registry.example.json`）
- 典型页面：`client/src/pages/admin/FileManagement.tsx`
- 只读多版本浏览（非上传）：见 `file-browser.md` / `client/src/components/ui/file-browser.tsx`

## 7. Portable Fallback

### 7.1 If host repo already has upload / file components

- 保留宿主仓上传逻辑、安全校验和接口。
- 视觉对齐：4px、dashed border、品牌蓝 drag-over、文件行紧凑、进度条 token 化。
- 文件名、错误信息和服务端返回文本按纯文本渲染，不使用不可信 HTML。
- 文件类型、大小、权限校验必须由后端或业务层兜底，前端提示不能代替安全校验。

### 7.2 Minimal React fallback

```tsx
export function PortableUploadArea() {
  return (
    <div className="flex min-h-[160px] flex-col items-center justify-center gap-3 rounded-[4px] border border-dashed border-[var(--cp-border)] bg-[var(--cp-surface)] p-6 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-[4px] bg-[var(--cp-bg-subtle)] text-[var(--cp-text-weak)]">↑</div>
      <div>
        <div className="text-sm font-medium text-[var(--cp-text-title)]">上传文件</div>
        <div className="mt-1 text-xs text-[var(--cp-text-muted)]">支持 PDF / DOCX / PNG，单个文件不超过 20MB</div>
      </div>
      <button type="button" className="h-9 rounded-[4px] bg-[var(--cp-brand-black)] px-4 text-sm text-white">选择文件</button>
    </div>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-upload"><div class="cp-upload-icon">↑</div><strong>上传文件</strong><p>支持 PDF / DOCX / PNG，单个文件不超过 20MB</p><button>选择文件</button></div>
```

```css
.cp-upload { display: flex; min-height: 160px; flex-direction: column; align-items: center; justify-content: center; gap: 12px; border: 1px dashed var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 24px; text-align: center; }
.cp-upload-icon { display: flex; width: 40px; height: 40px; align-items: center; justify-content: center; border-radius: 4px; background: var(--cp-bg-subtle); color: var(--cp-text-weak); }
.cp-upload strong { font-size: 14px; color: var(--cp-text-title); }
.cp-upload p { margin: 0; font-size: 12px; color: var(--cp-text-muted); }
.cp-upload button { height: 36px; border: 0; border-radius: 4px; background: var(--cp-brand-black); padding: 0 16px; color: white; }
```

## 8. Migration Rules

- 旧写法：默认 Upload 大图标、粗虚线框、无上传限制说明、失败只 toast。
- 新口径：上传区必须说明对象、限制、下一步；文件行显示进度、错误、重试。
- 页面级空态可用登记插画；Dialog / Drawer 内上传空态不用大插画。
- 文件名过长必须 truncate，完整名可通过 tooltip / title 查看。
- 删除文件需确认或提供撤销路径。

## 9. Do / Don't

Do:

- 明确文件类型和大小限制。
- 上传中显示进度，失败给重试。
- 使用已登记图标或 lucide icon。

Don't:

- 不要用 emoji 或默认不受控上传图标。
- 不要把上传安全校验只放在前端。
- 不要在紧凑容器里放页面级大插画。

## 10. QA Checklist

- [ ] 上传区 4px、dashed border、token 色正确
- [ ] 文件限制说明清楚
- [ ] idle / drag-over / uploading / success / failed 状态完整
- [ ] 错误有原因和重试 / 删除入口
- [ ] 文件名过长可读且不撑破布局
- [ ] 删除有确认或撤销路径
- [ ] 不渲染不可信 HTML
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/progress.tsx`
- Demo page: `client/src/pages/admin/FileManagement.tsx`
- Related spec: `component-specs/file-browser.md`（多版本资产只读浏览，与本 spec 职责正交）
- Related spec: `component-specs/empty-state.md`
- Related spec: `component-specs/loading-progress.md`
- Related reference: `references/assets-icons.md`

## 代码对照（✅/❌）

### ❌ 错误：实线 border + 圆角 lg
```tsx
<div className="border border-gray-300 rounded-lg p-8 text-center">
  <p>点击或拖拽文件到此处</p>
</div>
```
**为什么错**：实线与表单/卡片混淆，无法传达"投放区"语义；圆角过大与 ClawPro 4px 节奏冲突。

### ✅ 正确：4px dashed 投放区
```tsx
<UploadDropzone>
  {/* 内部样式：
      border: 1px dashed var(--cp-border)
      border-radius: 4px
      background: var(--cp-bg-subtle) */}
</UploadDropzone>
```

---

### ❌ 错误：拖入态加深背景
```tsx
<div
  className={cn(
    'border-2 border-dashed',
    isDragOver && 'bg-gray-100'
  )}
>
```
**为什么错**：仅靠灰底变深变化太弱；dashed border 无变化，用户感知不到"可以放手"。

### ✅ 正确：drag-over 切品牌蓝边框
```tsx
<UploadDropzone
  /* idle      : border-[var(--cp-border)]            */
  /* drag-over : border-[var(--cp-brand-blue)]
                 bg-[var(--cp-brand-tint)]            */
/>
```

---

### ❌ 错误：用 emoji 表示文件类型
```tsx
<li>📄 report.pdf</li>
<li>🖼️ avatar.png</li>
<li>🗜️ data.zip</li>
```
**为什么错**：emoji 跨平台渲染不一致；与 Lucide 体系冲突；无法被 a11y 朗读。

### ✅ 正确：Lucide File* 图标
```tsx
<FileItem
  icon={<FileTextIcon size={16} className="text-[var(--cp-text-weak)]" />}
  name="report.pdf"
/>
{/* 按 mime/扩展名映射：FileText / FileImage / FileArchive / FileCode ... */}
```

---

### ❌ 错误：长文件名 wrap
```tsx
<div className="text-sm">{file.name}</div>
{/* 200 字符的文件名换行 5 行 */}
```
**为什么错**：列表行高暴涨；删除按钮被推飞；批量场景视觉崩溃。

### ✅ 正确：truncate + tooltip
```tsx
<Tooltip content={file.name}>
  <span className="truncate text-sm max-w-[280px]">
    {file.name}
  </span>
</Tooltip>
{/* 单行截断 + 悬浮看全名 */}
```

---

### ❌ 错误：上传进度走 toast
```tsx
toast.loading(`正在上传 ${file.name}...`);
{/* 然后又  */}
toast.success(`${file.name} 上传成功`);
```
**为什么错**：批量上传 10 个文件刷出 10 条 toast；进度信息无法回看；整体进度不可视。

### ✅ 正确：行内进度条
```tsx
<UploadFileList
  files={uploadingFiles}
  /* 每行：[icon] [name] [progress 0-100%] [取消/重试] */
/>
{/* 仅在全部完成 / 部分失败时弹一条总结 toast */}
```
