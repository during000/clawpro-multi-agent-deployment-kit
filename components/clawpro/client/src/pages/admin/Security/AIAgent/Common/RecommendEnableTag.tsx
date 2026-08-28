/**
 * RecommendEnableTag —— 「未开启，推荐开启」状态徽标
 *
 * 0602 收口：原本 NetGroupList、HostGroupList 各手写一份
 *   `text-yellow-600 border-yellow-400 bg-yellow-50` 的 Badge，
 * 抽到这里统一，避免颜色三件套散落。
 *
 * - 已开启场景使用 tone="success"（如 "已开启 N 条策略"）
 * - 未开启场景使用 tone="warning"（如 "未开启，推荐开启"）
 *
 * 注：这两组色仍属"语义状态色"豁免，但收敛到一个组件后，未来如果接入
 * 全局 StatusTag 体系，只需要改这一个文件。
 */
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type Tone = "warning" | "success";

interface RecommendEnableTagProps {
  tone?: Tone;
  className?: string;
  children: React.ReactNode;
}

const TONE_CLASS: Record<Tone, string> = {
  warning: "text-yellow-600 border-yellow-400 bg-yellow-50",
  success: "text-green-600 border-green-300 bg-green-50",
};

export function RecommendEnableTag({
  tone = "warning",
  className,
  children,
}: RecommendEnableTagProps) {
  return (
    <Badge variant="outline" className={cn("shrink-0", TONE_CLASS[tone], className)}>
      {children}
    </Badge>
  );
}

export default RecommendEnableTag;
