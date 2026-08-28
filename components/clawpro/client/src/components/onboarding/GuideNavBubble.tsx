/**
 * GuideNavBubble - 新功能预览：导航功能预览介绍气泡
 * 对应场景：结构层 1.5 页面入口新增 / 元素层 2.1 新增按钮
 * 
 * 特点：
 * - 依附在 sidebar/导航项旁边
 * - 带箭头指向导航入口
 * - 包含简短描述 + 可选的截图预览
 * - 非阻断，点击空白可关闭
 */
import { X, ArrowRight } from "lucide-react";

interface GuideNavBubbleProps {
  open: boolean;
  onClose: () => void;
  /** 功能名称 */
  title: string;
  /** 功能描述 */
  description: string;
  /** 预览图片 */
  image?: string;
  /** 气泡位置（相对于目标元素） */
  placement?: "right" | "bottom" | "left";
  /** 跳转链接 */
  href?: string;
  /** "去看看"按钮文案 */
  actionText?: string;
  /** 目标元素定位（用于演示，实际使用 Portal 定位） */
  style?: React.CSSProperties;
}

export function GuideNavBubble({
  open,
  onClose,
  title,
  description,
  image,
  placement = "right",
  href,
  actionText = "去看看",
  style,
}: GuideNavBubbleProps) {
  if (!open) return null;

  const arrowClasses = {
    right: "left-0 top-5 -translate-x-full",
    bottom: "top-0 left-6 -translate-y-full rotate-90",
    left: "right-0 top-5 translate-x-full rotate-180",
  };

  return (
    <div className="relative z-[9980] animate-in fade-in slide-in-from-left-2 duration-200" style={style}>
      {/* 箭头 */}
      <div className={`absolute ${arrowClasses[placement]}`}>
        <svg width="8" height="16" viewBox="0 0 8 16" fill="none">
          <path d="M8 0L0 8L8 16" fill="white" />
          <path d="M8 0L0 8L8 16" stroke="#E5E7EB" strokeWidth="1" fill="none" />
        </svg>
      </div>

      {/* 气泡主体 */}
      <div className="w-[300px] bg-white rounded-xl border border-gray-200 shadow-xl overflow-hidden">
        {/* 预览图 */}
        {image && (
          <div className="border-b border-gray-100">
            <img src={image} alt={title} className="w-full h-[140px] object-cover" />
          </div>
        )}

        {/* 文字区 */}
        <div className="p-4">
          <div className="flex items-start justify-between gap-2">
            <div>
              <div className="flex items-center gap-2 mb-1.5">
                <span className="px-1.5 py-0.5 text-[10px] font-medium text-blue-600 bg-blue-50 rounded">NEW</span>
                <h4 className="text-sm font-medium text-gray-800">{title}</h4>
              </div>
              <p className="text-xs text-gray-500 leading-relaxed">{description}</p>
            </div>
            <button
              onClick={onClose}
              className="shrink-0 w-5 h-5 rounded flex items-center justify-center hover:bg-gray-100 transition-colors"
            >
              <X className="w-3.5 h-3.5 text-gray-400" />
            </button>
          </div>

          {/* 操作按钮 */}
          {href && (
            <div className="mt-3 flex items-center gap-2">
              <a
                href={href}
                className="h-[28px] inline-flex items-center gap-1 px-3 text-xs font-medium text-white bg-gray-900 hover:bg-gray-800 rounded-md transition-colors"
              >
                {actionText}
                <ArrowRight className="w-3 h-3" />
              </a>
              <button
                onClick={onClose}
                className="h-[28px] px-3 text-xs text-gray-500 hover:text-gray-700 transition-colors inline-flex items-center"
              >
                稍后再说
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
