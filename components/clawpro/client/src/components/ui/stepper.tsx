/**
 * Stepper - 步骤条组件
 *
 * 用法：
 *   <Stepper
 *     current={2}
 *     steps={[
 *       { label: "选择数据源方式" },
 *       { label: "添加应用凭证" },
 *       { label: "设置字段映射" },
 *       { label: "完成" },
 *     ]}
 *   />
 *
 * 状态：
 *   - completed (序号 < current)：实心品牌蓝圆圈 + Check 图标 + 次级文字
 *   - active    (序号 === current)：实心品牌蓝圆圈 + 序号 + 主文字加粗
 *   - pending   (序号 > current)：浅灰圆圈 + 序号 + 极弱文字
 *
 * 间隔：步骤之间用 ChevronRight 分隔（与设计稿一致）
 */
import React from "react";
import { Check, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

export interface StepperItem {
  /** 步骤标题 */
  label: string;
}

export interface StepperProps {
  /** 当前激活步骤（1-based） */
  current: number;
  /** 步骤列表 */
  steps: StepperItem[];
  /** 自定义容器类名 */
  className?: string;
}

type Status = "completed" | "active" | "pending";

const getStatus = (index: number, current: number): Status => {
  const stepNum = index + 1;
  if (stepNum < current) return "completed";
  if (stepNum === current) return "active";
  return "pending";
};

export function Stepper({ current, steps, className }: StepperProps) {
  return (
    <div className={cn("flex items-center gap-2 flex-wrap", className)} role="list">
      {steps.map((step, idx) => {
        const status = getStatus(idx, current);
        return (
          <React.Fragment key={idx}>
            <div className="flex items-center gap-2" role="listitem" aria-current={status === "active" ? "step" : undefined}>
              {/* 步骤圆圈 */}
              <span
                className={cn(
                  "w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium tabular-nums shrink-0 transition-colors",
                  status === "completed" && "bg-blue-500 text-white",
                  status === "active" && "bg-blue-500 text-white",
                  status === "pending" && "bg-gray-100 text-gray-400",
                )}
              >
                {status === "completed" ? <Check className="w-3.5 h-3.5" /> : idx + 1}
              </span>
              {/* 步骤标题 */}
              <span
                className={cn(
                  "text-sm transition-colors",
                  status === "active" && "font-medium text-gray-950",
                  status === "completed" && "text-gray-500",
                  status === "pending" && "text-gray-400",
                )}
              >
                {step.label}
              </span>
            </div>
            {idx < steps.length - 1 && (
              <ChevronRight className="w-4 h-4 text-gray-400 shrink-0" aria-hidden />
            )}
          </React.Fragment>
        );
      })}
    </div>
  );
}

export default Stepper;
