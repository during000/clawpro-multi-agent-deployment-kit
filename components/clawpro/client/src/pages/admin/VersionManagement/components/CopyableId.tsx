/**
 * CopyableId - 可点击复制的 ID 徽标
 *
 * 视觉参考 TAT 命令列表：
 *   - 等宽字体 + 浅灰底
 *   - 鼠标悬停时显示复制图标
 *   - 点击复制到剪贴板，sonner toast 提示
 *
 * 用法：<CopyableId id="cmd-q27hmxjd" />
 */
import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { toast } from "sonner";

interface Props {
  id: string;
  /** 是否使用蓝色文字（强调风格，匹配 TAT 截图）；默认 false 用灰色 */
  primary?: boolean;
  /** 是否使用黑色正文文字（用于该 ID 是当前列唯一主文字时） */
  dark?: boolean;
  /** 自定义额外的 className */
  className?: string;
}

export default function CopyableId({ id, primary = false, dark = false, className = "" }: Props) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(id);
      setCopied(true);
      toast.success("已复制", { description: id });
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error("复制失败，请手动复制");
    }
  };

  const textColor = primary
    ? "text-[#355EF1] hover:text-[#355EF1]"
    : dark
    ? "text-[#0A0A0A] hover:text-[#0A0A0A]"
    : "text-[#737373] hover:text-[#0A0A0A]";

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={`点击复制 ${id}`}
      className={`group inline-flex items-center gap-1 text-xs font-mono ${textColor} transition-colors ${className}`}
    >
      <span className="select-all">{id}</span>
      {copied ? (
        <Check className="w-3 h-3 text-green-500 shrink-0" />
      ) : (
        <Copy className="w-3 h-3 opacity-0 group-hover:opacity-60 transition-opacity shrink-0" />
      )}
    </button>
  );
}
