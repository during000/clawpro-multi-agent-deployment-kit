/**
 * Tree 树结构组件
 * ─────────────────────────────────────────────────────────────────
 * 通用文件树 / 目录树组件，用于展示层级文件结构。
 *
 * 使用场景：
 *   - SkillDetail / PluginDetail / MCPDetail 的文件浏览面板
 *   - 任何需要展示层级目录+文件的场景
 *
 * 功能：
 *   - 自动按路径组织渲染目录结构
 *   - 点击目录行展开/折叠
 *   - 点击文件行触发选中回调
 *   - 支持自定义图标、禁用文件、选中高亮
 *   - 缩进跟随层级深度（每层 16px）
 *
 * 用法：
 *   <FileTree
 *     files={[{ name: "src/index.ts", size: 1024 }, { name: "README.md" }]}
 *     selectedFile="src/index.ts"
 *     onSelectFile={(name) => setSelected(name)}
 *     isViewable={(name) => /\.(ts|js|json|md)$/.test(name)}
 *   />
 */
import { useState, useEffect, useMemo } from "react";
import { ChevronDown, ChevronRight, Folder, FolderOpen, FileText } from "lucide-react";
import { cn } from "@/lib/utils";
import { BodyText } from "./Typography";

export interface FileEntry {
  name: string;
  size?: number;
  content?: string;
}

export interface FileTreeProps {
  /** 文件列表（扁平路径格式，如 "src/utils/index.ts"） */
  files: FileEntry[];
  /** 当前选中的文件路径 */
  selectedFile?: string | null;
  /** 选中文件回调 */
  onSelectFile?: (fileName: string | null) => void;
  /** 判断文件是否可查看/可点击（不可查看的文件禁用交互） */
  isViewable?: (fileName: string) => boolean;
  /** 默认展开所有目录（默认 true） */
  defaultExpandAll?: boolean;
  /** 额外 className */
  className?: string;
}

export function FileTree({
  files,
  selectedFile,
  onSelectFile,
  isViewable = () => true,
  defaultExpandAll = true,
  className,
}: FileTreeProps) {
  // 收集所有目录路径
  const allDirs = useMemo(() => {
    const dirs = new Set<string>();
    for (const file of files) {
      const parts = file.name.split("/");
      for (let i = 1; i < parts.length; i++) {
        dirs.add(parts.slice(0, i).join("/"));
      }
    }
    return dirs;
  }, [files]);

  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());

  // 默认展开所有目录
  useEffect(() => {
    if (defaultExpandAll) {
      setExpandedDirs(new Set(allDirs));
    }
  }, [allDirs, defaultExpandAll]);

  const toggleDir = (dirPath: string) => {
    setExpandedDirs((prev) => {
      const next = new Set(prev);
      if (next.has(dirPath)) {
        next.delete(dirPath);
      } else {
        next.add(dirPath);
      }
      return next;
    });
  };

  const handleFileClick = (fileName: string) => {
    if (!isViewable(fileName)) return;
    onSelectFile?.(selectedFile === fileName ? null : fileName);
  };

  // 排序文件
  const sorted = useMemo(
    () => [...files].sort((a, b) => a.name.localeCompare(b.name)),
    [files]
  );

  // 渲染树
  const renderedDirs = new Set<string>();
  const result: React.ReactNode[] = [];

  for (const file of sorted) {
    const parts = file.name.split("/");
    const isDir = file.name.endsWith("/");
    const isNested = parts.length > 1 && !isDir;
    const canView = !isDir && isViewable(file.name);

    // 渲染各层目录头
    if (isNested) {
      for (let i = 1; i < parts.length; i++) {
        const dirPath = parts.slice(0, i).join("/");
        if (!renderedDirs.has(dirPath)) {
          renderedDirs.add(dirPath);
          const depth = i - 1;
          const isExpanded = expandedDirs.has(dirPath);

          // 检查祖先是否展开
          let ancestorsExpanded = true;
          for (let j = 1; j < i; j++) {
            const ancestor = parts.slice(0, j).join("/");
            if (!expandedDirs.has(ancestor)) {
              ancestorsExpanded = false;
              break;
            }
          }
          if (!ancestorsExpanded) continue;

          result.push(
            <button
              key={`dir-${dirPath}`}
              type="button"
              onClick={() => toggleDir(dirPath)}
              className="w-full flex items-center gap-1.5 h-8 px-2 hover:bg-[#f4f4f5] rounded-[4px] transition-colors cursor-pointer"
              style={{ paddingLeft: `${8 + depth * 16}px` }}
            >
              {isExpanded ? (
                <ChevronDown className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
              ) : (
                <ChevronRight className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
              )}
              {isExpanded ? (
                <FolderOpen className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
              ) : (
                <Folder className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
              )}
              <BodyText as="span" tone="emphasis" className="truncate font-medium">
                {parts[i - 1]}
              </BodyText>
            </button>
          );
        }
      }

      // 检查父目录全部展开才渲染文件
      let allParentsExpanded = true;
      for (let i = 1; i < parts.length; i++) {
        const ancestor = parts.slice(0, i).join("/");
        if (!expandedDirs.has(ancestor)) {
          allParentsExpanded = false;
          break;
        }
      }
      if (!allParentsExpanded) continue;
    }

    // 跳过纯目录条目
    if (isDir) continue;

    const depth = parts.length - 1;
    const isSelected = selectedFile === file.name;
    result.push(
      <button
        key={file.name}
        type="button"
        onClick={() => handleFileClick(file.name)}
        disabled={!canView}
        className={cn(
          "w-full flex items-center gap-1.5 h-8 px-2 rounded-[4px] transition-colors",
          isSelected
            ? "bg-[#f4f4f5]"
            : canView
            ? "hover:bg-[#f4f4f5] cursor-pointer"
            : "cursor-not-allowed opacity-60"
        )}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
      >
        <FileText className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
        <BodyText
          as="span"
          tone={canView ? "emphasis" : "weak"}
          className={cn("truncate", isSelected && "font-medium")}
        >
          {parts[parts.length - 1]}
        </BodyText>
      </button>
    );
  }

  return <div className={cn("flex flex-col", className)}>{result}</div>;
}
