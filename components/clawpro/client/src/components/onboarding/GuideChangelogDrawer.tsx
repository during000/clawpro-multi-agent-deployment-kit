/**
 * GuideChangelogDrawer - 更新记录：侧边抽屉
 * 对应场景：所有层级的汇总视图
 * 
 * 特点：
 * - 从右侧滑出的抽屉
 * - 按版本展示更新记录列表
 * - 每条记录含标签（结构/元素/逻辑/系统）
 * - 可从提示条的"查看详情"触发
 */
import { X, ExternalLink } from "lucide-react";

export interface ChangelogEntry {
  id: string;
  title: string;
  description: string;
  /** 变更层级标签 */
  tag: "结构" | "元素" | "逻辑" | "系统" | "跨端";
  /** 变更日期 */
  date: string;
  /** 关联页面路径 */
  href?: string;
}

export interface ChangelogVersion {
  version: string;
  date: string;
  entries: ChangelogEntry[];
}

interface GuideChangelogDrawerProps {
  open: boolean;
  onClose: () => void;
  /** 版本更新记录列表 */
  versions: ChangelogVersion[];
}

export function GuideChangelogDrawer({
  open,
  onClose,
  versions,
}: GuideChangelogDrawerProps) {
  if (!open) return null;

  const tagColors: Record<string, string> = {
    "结构": "bg-purple-50 text-purple-600 border-purple-200",
    "元素": "bg-blue-50 text-blue-600 border-blue-200",
    "逻辑": "bg-amber-50 text-amber-600 border-amber-200",
    "系统": "bg-red-50 text-red-600 border-red-200",
    "跨端": "bg-green-50 text-green-600 border-green-200",
  };

  return (
    <>
      {/* 遮罩 */}
      <div
        className="fixed inset-0 z-[9994] bg-black/30 backdrop-blur-[2px] animate-in fade-in duration-200"
        onClick={onClose}
      />

      {/* 抽屉 */}
      <div className="fixed top-0 right-0 bottom-0 z-[9995] w-full max-w-[420px] bg-white shadow-2xl animate-in slide-in-from-right duration-300">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <div>
            <h2 className="text-base font-semibold text-gray-900">更新记录</h2>
            <p className="text-xs text-gray-400 mt-0.5">
              每次版本更新的内容概览，点击可直达对应功能
            </p>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg flex items-center justify-center hover:bg-gray-100 transition-colors"
          >
            <X className="w-4 h-4 text-gray-500" />
          </button>
        </div>

        {/* Content - 滚动区 */}
        <div className="overflow-y-auto h-[calc(100%-72px)] px-6 py-4">
          {versions.map((ver) => (
            <div key={ver.version} className="mb-8 last:mb-0">
              {/* 版本标题 */}
              <div className="flex items-center gap-2 mb-3 sticky top-0 bg-white py-1">
                <span className="px-2 py-0.5 text-xs font-semibold text-gray-800 bg-gray-100 rounded-md">
                  {ver.version}
                </span>
                <span className="text-xs text-gray-400">{ver.date}</span>
              </div>

              {/* 条目列表 */}
              <div className="space-y-3 ml-1">
                {ver.entries.map((entry) => (
                  <div
                    key={entry.id}
                    className="group relative pl-4 before:absolute before:left-0 before:top-2 before:w-1.5 before:h-1.5 before:rounded-full before:bg-gray-300"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-0.5">
                          <span className={`inline-flex px-1.5 py-0.5 text-[10px] font-medium rounded border ${tagColors[entry.tag] || tagColors["元素"]}`}>
                            {entry.tag}
                          </span>
                          <h4 className="text-sm font-medium text-gray-800 truncate">{entry.title}</h4>
                        </div>
                        <p className="text-xs text-gray-500 leading-relaxed">{entry.description}</p>
                      </div>
                      {entry.href && (
                        <a
                          href={entry.href}
                          className="shrink-0 w-6 h-6 rounded flex items-center justify-center opacity-0 group-hover:opacity-100 hover:bg-gray-100 transition-all"
                          title="前往查看"
                        >
                          <ExternalLink className="w-3.5 h-3.5 text-gray-400" />
                        </a>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
