import * as React from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { BodyMedium, MetaText } from "@/components/ui/Typography";
import { StatusTag } from "@/components/ui/status-tag";

export interface SkillSelectCardProps {
  /** 技能名称 */
  name: string;
  /** 版本号（不含 v 前缀） */
  version: string;
  /** 技能描述 */
  description?: string;
  /** 卡片状态：default 默认态 | selected 选中态 | disabled 禁用态 */
  state?: "default" | "selected" | "disabled";
  /** 禁用态右上角提示文案，默认"已添加" */
  disabledLabel?: string;
  /** 标题行右侧额外内容（如应用范围标签），显示在选中对勾/禁用标签之前 */
  extra?: React.ReactNode;
  /** 点击事件（disabled 状态下不触发） */
  onClick?: () => void;
  className?: string;
}

/**
 * 技能多选卡片组件
 *
 * 用于"从公共技能库添加"等弹窗中的技能多选场景。
 * 包含三种状态：默认态、选中态、禁用态。
 *
 * 色彩 Token 来源：
 * - 选中边框/对勾：blue-500（对齐 Select hover 规范）
 * - 选中背景：rgba(20,71,230,0.06)（对齐表格选中行）
 * - 禁用背景：#FAFAFA（对齐 Input disabled）
 */
function SkillSelectCard({
  name,
  version,
  description,
  state = "default",
  disabledLabel = "已添加",
  extra,
  onClick,
  className,
}: SkillSelectCardProps) {
  const isDisabled = state === "disabled";
  const isSelected = state === "selected";

  return (
    <div
      onClick={() => !isDisabled && onClick?.()}
      className={cn(
        "rounded-[4px] border p-3 transition-all",
        isDisabled && "border-gray-200 bg-[#FAFAFA] opacity-40 cursor-not-allowed",
        isSelected && "border-blue-500 bg-[rgba(20,71,230,0.06)] cursor-pointer",
        !isDisabled && !isSelected && "border-gray-200 bg-white hover:border-blue-500 hover:shadow-sm cursor-pointer",
        className,
      )}
    >
      {/* 标题行：技能名称 + 版本号 + extra + 选中对勾/禁用标签 */}
      <div className="mb-1.5 flex min-w-0 items-center gap-2">
        <span className="flex min-w-0 flex-1 items-center gap-2">
          <BodyMedium className="block min-w-0 flex-1 truncate font-mono">{name}</BodyMedium>
          <StatusTag mode="fill" variant="gray" className="shrink-0">v{version}</StatusTag>
        </span>
        {extra}
        {isSelected && (
          <span className="inline-flex items-center justify-center w-5 h-5 rounded-full bg-blue-500 shrink-0">
            <Check className="w-3 h-3 text-white" />
          </span>
        )}
        {isDisabled && (
          <StatusTag mode="fill" variant="gray" className="shrink-0">{disabledLabel}</StatusTag>
        )}
      </div>
      {/* 描述 */}
      {description && (
        <MetaText as="p" tone="secondary" className="line-clamp-2 break-words">{description}</MetaText>
      )}
    </div>
  );
}

export { SkillSelectCard };
