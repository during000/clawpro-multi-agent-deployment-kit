import { useState, useMemo, useCallback } from "react";
import type React from "react";
import { Eye, Code, Download, Loader, Info } from "lucide-react";
import { FileTree, type FileEntry } from "./tree";
import { BodyMedium, MetaText, BodyText } from "./Typography";
import { Badge } from "./badge";
import { TinyText } from "./Typography";
import { Tooltip, TooltipContent, TooltipTrigger } from "./tooltip";
import { MetaMedium } from "./Typography";

export interface VersionInfo {
  version: string;
  date: string;
  isLatest?: boolean;
  changeLog?: string;
}

export interface FileBrowserProps {
  versions: VersionInfo[];
  files: FileEntry[];
  getFileContent: (fileName: string) => string | undefined;
  height?: string;
  showDownload?: boolean;
  onDownload?: (version: string) => void;
  isDownloading?: boolean;
  defaultVersion?: string;
  defaultFile?: string;
  className?: string;
  onVersionChange?: (version: string) => void;
}

export function FileBrowser({
  versions,
  files,
  getFileContent,
  height = "47rem",
  showDownload = false,
  onDownload,
  isDownloading = false,
  defaultVersion,
  defaultFile,
  className,
  onVersionChange,
}: FileBrowserProps) {
  const [selectedVersion, setSelectedVersion] = useState<string>(() => {
    if (defaultVersion) return defaultVersion;
    return versions[0]?.version || "";
  });

  const [selectedFile, setSelectedFile] = useState<string | null>(() => {
    if (defaultFile) return defaultFile;
    const skillMd = files.find((f) => f.name.toLowerCase().endsWith("skill.md"));
    return skillMd?.name || files[0]?.name || null;
  });

  const [viewMode, setViewMode] = useState<"preview" | "source">(() => {
    if (!selectedFile) return "source";
    const lower = selectedFile.toLowerCase();
    return lower.endsWith(".md") || lower.endsWith(".mdx") ? "preview" : "source";
  });

  // 切换版本：同步内部 state + 通知父组件
  const handleVersionChange = useCallback((version: string) => {
    setSelectedVersion(version);
    if (onVersionChange) onVersionChange(version);
  }, [onVersionChange]);

  const handleSelectFile = useCallback((fileName: string | null) => {
    setSelectedFile(fileName);
    if (fileName) {
      const lower = fileName.toLowerCase();
      setViewMode(lower.endsWith(".md") || lower.endsWith(".mdx") ? "preview" : "source");
    }
  }, []);

  const selectedVersionInfo = useMemo(
    () => versions.find((v) => v.version === selectedVersion),
    [versions, selectedVersion]
  );

  const fileContent = useMemo(() => {
    if (!selectedFile) return "";
    return getFileContent(selectedFile) || "";
  }, [selectedFile, getFileContent]);

  return (
    <div
      className={`flex border border-[var(--border)] rounded-[var(--radius-lg)] overflow-hidden bg-white ${className || ""}`}
      style={{ height } as React.CSSProperties}
    >
      {/* Left Column: Version List */}
      <div className="w-[14%] min-w-[120px] border-r border-[var(--border)] flex flex-col">
        <div className="h-12 px-3 border-b border-[var(--border)] flex items-center">
          <BodyMedium as="p" tone="emphasis">版本</BodyMedium>
        </div>
        <div className="flex-1 overflow-y-auto">
          {versions.map((ver) => {
            const isSelected = selectedVersion === ver.version;
            return (
              <button
                key={ver.version}
                onClick={() => handleVersionChange(ver.version)}
                className={`w-full text-left px-3 py-2.5 border-b border-[#f4f4f5] transition-colors rounded-none ${
                  isSelected ? "bg-[#f4f4f5]" : "hover:bg-[#f4f4f5] cursor-pointer"
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <BodyMedium as="span" tone="emphasis" className="font-semibold">{ver.version}</BodyMedium>
                      {ver.isLatest && (
                        <span className="inline-flex h-[18px] items-center justify-center rounded-[2px] border border-[#1447E6] px-1 leading-none">
                          <TinyText as="span" tone="brand">New</TinyText>
                        </span>
                      )}
                    </div>
                    <MetaText as="p" tone="weak" className="mt-0.5">{ver.date}</MetaText>
                  </div>
                  {ver.changeLog && (
                    <Tooltip delayDuration={300}>
                      <TooltipTrigger asChild>
                        <span className="cursor-pointer mt-0.5 flex-shrink-0">
                          <Info className="w-3 h-3 text-[var(--text-weak)] hover:text-[var(--text-secondary)]" />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="right" className="max-w-[260px] p-3 bg-white border border-[var(--border)] shadow-lg">
                        <MetaMedium as="p" tone="primary" className="mb-1.5">更新说明</MetaMedium>
                        <MetaText as="p" tone="secondary" className="whitespace-pre-line leading-relaxed">{ver.changeLog}</MetaText>
                      </TooltipContent>
                    </Tooltip>
                  )}
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Middle Column: File Tree */}
      <div className="w-[22%] min-w-[160px] border-r border-[var(--border)] flex flex-col">
        <div className="h-12 px-3 border-b border-[var(--border)] flex items-center justify-between">
          <BodyMedium as="p" tone="emphasis">
            {selectedVersionInfo?.version || selectedVersion}
          </BodyMedium>
          {showDownload && onDownload && (
            <button
              onClick={() => onDownload(selectedVersion)}
              disabled={isDownloading}
              className="text-[var(--text-muted)] hover:text-[var(--text-title)] transition-colors"
              title="Download this version"
            >
              {isDownloading ? (
                <Loader className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Download className="w-3.5 h-3.5" />
              )}
            </button>
          )}
        </div>
        <div className="flex-1 overflow-y-auto px-3 py-2">
          <FileTree
            files={files}
            selectedFile={selectedFile}
            onSelectFile={handleSelectFile}
            isViewable={(name) => {
              const lower = name.toLowerCase();
              if (!lower.includes(".") && !lower.includes("/")) return true;
              const exts = [".md", ".mdx", ".xml", ".json", ".txt", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".sh", ".bat", ".py", ".js", ".ts", ".tsx", ".jsx", ".css", ".html", ".htm", ".svg", ".env", ".gitignore", ".dockerfile"];
              return exts.some((ext) => lower.endsWith(ext));
            }}
            defaultExpandAll={true}
          />
        </div>
      </div>

      {/* Right Column: File Content */}
      <div className="flex-1 flex flex-col bg-white">
        {selectedFile ? (
          <>
            <div className="h-12 px-3 border-b border-[var(--border)] flex items-center justify-between">
              <BodyMedium as="p" tone="emphasis">
                {selectedFile}
              </BodyMedium>
              <div className="flex items-center gap-0.5 bg-[var(--bg-grey-hover)]/60 rounded p-0.5">
                <button
                  onClick={() => setViewMode("preview")}
                  className={`flex items-center gap-1 px-2 py-1 rounded transition-colors ${
                    viewMode === "preview" ? "bg-white shadow-sm" : ""
                  }`}
                >
                  <Eye className="w-3 h-3" />
                  <MetaText
                    as="span"
                    tone={viewMode === "preview" ? "primary" : "muted"}
                    className={viewMode === "preview" ? "font-medium" : ""}
                  >
                    Preview
                  </MetaText>
                </button>
                <button
                  onClick={() => setViewMode("source")}
                  className={`flex items-center gap-1 px-2 py-1 rounded transition-colors ${
                    viewMode === "source" ? "bg-white shadow-sm" : ""
                  }`}
                >
                  <Code className="w-3 h-3" />
                  <MetaText
                    as="span"
                    tone={viewMode === "source" ? "primary" : "muted"}
                    className={viewMode === "source" ? "font-medium" : ""}
                  >
                    Source
                  </MetaText>
                </button>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto">
              {(() => {
                if (!fileContent) {
                  return (
                    <div className="flex items-center justify-center h-full">
                      <BodyText as="p" tone="weak">
                        No file content
                      </BodyText>
                    </div>
                  );
                }

                const lower = selectedFile.toLowerCase();
                const isMd = lower.endsWith(".md") || lower.endsWith(".mdx");

                if (viewMode === "source") {
                  return (
                    <pre className="text-xs text-[var(--text-body)] overflow-x-auto whitespace-pre-wrap break-words font-mono leading-5 bg-[var(--bg-subtle)] p-3 m-0">
                      {fileContent}
                    </pre>
                  );
                }

                if (isMd) {
                  return (
                    <div className="p-4">
                      <p className="text-sm text-[var(--text-muted)]">Markdown preview requires MDXRenderer</p>
                    </div>
                  );
                }

                return (
                  <pre className="text-xs text-[var(--text-body)] overflow-x-auto whitespace-pre-wrap break-words font-mono leading-5 bg-[var(--bg-subtle)] p-3 m-0">
                    {fileContent}
                  </pre>
                );
              })()}
            </div>
          </>
        ) : (
          <div className="flex items-center justify-center h-full">
            <BodyText as="p" tone="muted">
              Select a file to view content
            </BodyText>
          </div>
        )}
      </div>
    </div>
  );
}
