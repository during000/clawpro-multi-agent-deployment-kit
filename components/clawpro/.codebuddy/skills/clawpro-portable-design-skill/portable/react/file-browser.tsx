/**
 * Portable FileBrowser — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 FileBrowser 时的可移植兜底实现（多版本资产文件浏览器）。
 *  - 不依赖 shadcn / Tailwind；样式由 portable/css/file-browser.css 提供，
 *    类名体系与 portable/html-css/file-browser.html 完全一致（.fb__*）。
 *  - 三栏只读浏览：版本（14% / min 120）/ 文件树（22% / min 160）/ 内容（flex-1）；
 *    上传 / 拖拽 / 进度不在本组件职责内（见 upload-file-browser.md）。
 *  - 视觉规范（component-specs/file-browser.md §4）：
 *      容器 8px 圆角（radius-lg 例外）；三栏头部 48px；
 *      左栏版本项 #f4f4f5 选中、Latest 蓝边胶囊、changeLog Info Tooltip；
 *      中栏 28px 行、默认全展开、内置可查看后缀白名单；
 *      右栏 Preview/Source 切换、<pre> 等宽软换行 bg-subtle、空 / 未选文案。
 *  - getFileContent 不要发 fetch（每次切换都会调用）；异步内容请父组件预热缓存。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/file-browser.css";
 *
 * 用法：
 *   <PortableFileBrowser
 *     versions={[{ version: "v1.3.0", date: "2026-06-05", isLatest: true, changeLog: "…" }]}
 *     files={[{ name: "SKILL.md", size: "8.0 KB" }, { name: "src", children: [...] }]}
 *     getFileContent={(name) => contentMap[name]}
 *     showDownload
 *     onDownload={(v) => downloadZip(v)}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface VersionInfo {
  version: string;
  date: string;
  isLatest?: boolean;
  changeLog?: string;
}

export interface FileEntry {
  /** 文件 / 目录名 */
  name: string;
  /** 末尾弱色大小（如 "8.0 KB"），目录留空 */
  size?: string;
  /** 子节点（有则视为目录） */
  children?: FileEntry[];
}

export interface PortableFileBrowserProps {
  versions: VersionInfo[];
  files: FileEntry[];
  /** 按文件名取内容；返回 undefined 表示空文件 */
  getFileContent: (fileName: string) => string | undefined;
  height?: string;
  showDownload?: boolean;
  onDownload?: (version: string) => void;
  isDownloading?: boolean;
  defaultVersion?: string;
  defaultFile?: string;
  onVersionChange?: (version: string) => void;
  className?: string;
}

/* 内置可查看后缀白名单 */
const VIEWABLE_EXT = new Set([
  "md", "mdx", "xml", "json", "txt", "yaml", "yml", "toml", "ini", "cfg", "conf",
  "sh", "bat", "py", "js", "ts", "tsx", "jsx", "css", "html", "htm", "svg",
  "env", "gitignore", "dockerfile",
]);

function ext(name: string): string {
  const i = name.lastIndexOf(".");
  return i >= 0 ? name.slice(i + 1).toLowerCase() : "";
}

function isMarkdown(name: string): boolean {
  const e = ext(name);
  return e === "md" || e === "mdx";
}

/** 目录（有 children）可点击展开；文件按白名单判断；无后缀纯名称视为目录占位可点击 */
function isViewable(entry: FileEntry): boolean {
  if (entry.children) return true;
  const e = ext(entry.name);
  if (!e) return true;
  return VIEWABLE_EXT.has(e);
}

/* ───────────── 图标 ───────────── */
const IconChevron = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);
const IconFile = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
    <polyline points="14 2 14 8 20 8" />
  </svg>
);
const IconFolder = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
  </svg>
);
const IconInfo = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <circle cx="12" cy="12" r="10" />
    <line x1="12" y1="16" x2="12" y2="12" />
    <line x1="12" y1="8" x2="12.01" y2="8" />
  </svg>
);
const IconDownload = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
    <polyline points="7 10 12 15 17 10" />
    <line x1="12" y1="15" x2="12" y2="3" />
  </svg>
);
const IconLoader = () => (
  <svg className="fb__spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <line x1="12" y1="2" x2="12" y2="6" />
    <line x1="12" y1="18" x2="12" y2="22" />
    <line x1="4.93" y1="4.93" x2="7.76" y2="7.76" />
    <line x1="16.24" y1="16.24" x2="19.07" y2="19.07" />
    <line x1="2" y1="12" x2="6" y2="12" />
    <line x1="18" y1="12" x2="22" y2="12" />
    <line x1="4.93" y1="19.07" x2="7.76" y2="16.24" />
    <line x1="16.24" y1="7.76" x2="19.07" y2="4.93" />
  </svg>
);
const IconEye = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
    <circle cx="12" cy="12" r="3" />
  </svg>
);
const IconCode = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <polyline points="16 18 22 12 16 6" />
    <polyline points="8 6 2 12 8 18" />
  </svg>
);

/* ───────────── 文件树节点（递归） ───────────── */
function TreeNode({
  entry,
  depth,
  selectedName,
  onSelect,
}: {
  entry: FileEntry;
  depth: number;
  selectedName: string | null;
  onSelect: (entry: FileEntry) => void;
}) {
  const isDir = !!entry.children;
  const [open, setOpen] = React.useState(true); // 默认全展开
  const disabled = !isViewable(entry);
  const selected = selectedName === entry.name;

  return (
    <>
      <div
        className="fb__node"
        role="treeitem"
        aria-selected={selected || undefined}
        aria-expanded={isDir ? open : undefined}
        data-disabled={disabled || undefined}
        tabIndex={disabled ? -1 : 0}
        style={{ paddingLeft: 6 + depth * 16 }}
        onClick={() => {
          if (disabled) return;
          if (isDir) setOpen((v) => !v);
          else onSelect(entry);
        }}
        onKeyDown={(e) => {
          if (disabled) return;
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            if (isDir) setOpen((v) => !v);
            else onSelect(entry);
          }
        }}
      >
        <span className="fb__node-chevron" data-expanded={isDir && open ? "true" : undefined}>
          {isDir ? <IconChevron /> : null}
        </span>
        <span className="fb__node-icon">{isDir ? <IconFolder /> : <IconFile />}</span>
        <span className="fb__node-label">{entry.name}</span>
        {entry.size && <span className="fb__node-size">{entry.size}</span>}
      </div>
      {isDir && open && (
        <div role="group">
          {entry.children!.map((child) => (
            <TreeNode
              key={child.name}
              entry={child}
              depth={depth + 1}
              selectedName={selectedName}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </>
  );
}

/* 找到首个可预览文件：优先 SKILL.md，否则首个非目录文件 */
function findDefaultFile(files: FileEntry[]): FileEntry | null {
  let first: FileEntry | null = null;
  const walk = (list: FileEntry[]): FileEntry | null => {
    for (const f of list) {
      if (!f.children) {
        if (!first) first = f;
        if (f.name === "SKILL.md") return f;
      }
      if (f.children) {
        const hit = walk(f.children);
        if (hit) return hit;
      }
    }
    return null;
  };
  return walk(files) ?? first;
}

export function PortableFileBrowser({
  versions,
  files,
  getFileContent,
  height = "47rem",
  showDownload = false,
  onDownload,
  isDownloading = false,
  defaultVersion,
  defaultFile,
  onVersionChange,
  className = "",
}: PortableFileBrowserProps) {
  const [activeVersion, setActiveVersion] = React.useState(defaultVersion ?? versions[0]?.version);
  const initialFile = React.useMemo(() => {
    if (defaultFile) return defaultFile;
    return findDefaultFile(files)?.name ?? null;
  }, [defaultFile, files]);
  const [selectedFile, setSelectedFile] = React.useState<string | null>(initialFile);
  const [viewMode, setViewMode] = React.useState<"preview" | "source">(
    initialFile && isMarkdown(initialFile) ? "preview" : "source"
  );
  const [tipVersion, setTipVersion] = React.useState<string | null>(null);

  const selectVersion = (v: string) => {
    setActiveVersion(v);
    onVersionChange?.(v);
  };

  const selectFile = (entry: FileEntry) => {
    setSelectedFile(entry.name);
    setViewMode(isMarkdown(entry.name) ? "preview" : "source");
  };

  const content = selectedFile ? getFileContent(selectedFile) : undefined;
  const merged = ["fb", className].filter(Boolean).join(" ");

  return (
    <div className={merged} style={{ height }}>
      {/* ─── 左栏：版本列表 ─── */}
      <div className="fb__col-versions">
        <div className="fb__header">
          <span className="fb__title">版本</span>
        </div>
        <div className="fb__versions" role="listbox" aria-label="版本列表">
          {versions.map((v) => (
            <div
              key={v.version}
              className="fb__version-item"
              role="option"
              aria-selected={v.version === activeVersion}
              tabIndex={0}
              onClick={() => selectVersion(v.version)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  selectVersion(v.version);
                }
              }}
            >
              <div className="fb__version-row">
                <span className="fb__version-name">{v.version}</span>
                {v.isLatest && <span className="fb__latest">Latest</span>}
                {v.changeLog && (
                  <button
                    type="button"
                    className="fb__changelog-trigger"
                    aria-label="变更说明"
                    onClick={(e) => e.stopPropagation()}
                    onMouseEnter={() => setTipVersion(v.version)}
                    onMouseLeave={() => setTipVersion(null)}
                    onFocus={() => setTipVersion(v.version)}
                    onBlur={() => setTipVersion(null)}
                  >
                    <IconInfo />
                  </button>
                )}
              </div>
              <div className="fb__version-date">{v.date}</div>
              {v.changeLog && tipVersion === v.version && (
                <div className="fb__tooltip" role="tooltip">
                  <p className="fb__tooltip-title">{v.version} 变更说明</p>
                  <p className="fb__tooltip-body">{v.changeLog}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* ─── 中栏：文件树 ─── */}
      <div className="fb__col-tree">
        <div className="fb__header">
          <span className="fb__title">{activeVersion}</span>
          {showDownload && (
            <div className="fb__tree-actions">
              <button
                type="button"
                className="fb__icon-btn"
                aria-label="下载该版本"
                disabled={isDownloading}
                onClick={() => activeVersion && onDownload?.(activeVersion)}
              >
                {isDownloading ? <IconLoader /> : <IconDownload />}
              </button>
            </div>
          )}
        </div>
        <div className="fb__tree" role="tree" aria-label="文件树">
          {files.map((entry) => (
            <TreeNode
              key={entry.name}
              entry={entry}
              depth={0}
              selectedName={selectedFile}
              onSelect={selectFile}
            />
          ))}
        </div>
      </div>

      {/* ─── 右栏：文件内容 ─── */}
      <div className="fb__col-content">
        <div className="fb__header">
          {selectedFile ? (
            <span className="fb__filename">{selectedFile}</span>
          ) : (
            <span className="fb__filename fb__filename--empty">未选择文件</span>
          )}
          {selectedFile && (
            <div className="fb__view-switch" role="tablist" aria-label="查看模式">
              <button
                type="button"
                role="tab"
                className="fb__view-btn"
                aria-selected={viewMode === "preview"}
                onClick={() => setViewMode("preview")}
              >
                <IconEye />
                Preview
              </button>
              <button
                type="button"
                role="tab"
                className="fb__view-btn"
                aria-selected={viewMode === "source"}
                onClick={() => setViewMode("source")}
              >
                <IconCode />
                Source
              </button>
            </div>
          )}
        </div>
        <div className="fb__content">
          {!selectedFile ? (
            <div className="fb__empty">Select a file to view content</div>
          ) : content == null || content === "" ? (
            <div className="fb__empty">No file content</div>
          ) : viewMode === "preview" && isMarkdown(selectedFile) ? (
            <div className="fb__empty">Markdown preview requires MDXRenderer</div>
          ) : (
            <pre className="fb__source">{content}</pre>
          )}
        </div>
      </div>
    </div>
  );
}
