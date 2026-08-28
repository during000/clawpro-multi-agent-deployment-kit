/**
 * MemoryCard - Memory 配置卡片组件
 * 用于 Agent 详细配置页面，支持启用/禁用 Memory 功能
 * 
 * 遵循 Agent Enterprise 设计规范：
 * - 卡片圆角：rounded-[4px] (16px)
 * - 统一阴影：通过 inline style 设置
 * - 图标容器：使用品牌渐变
 * - 状态颜色：使用语义色
 */

import { useState } from "react";
import { Brain } from "lucide-react";
import { Switch } from "@/components/ui/switch";
import { SurfaceCard } from "@/components/ui/Surface";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { toast } from "sonner";
import EnableMemoryDialog from "./EnableMemoryDialog";
import DisableMemoryDialog from "./DisableMemoryDialog";

interface MemoryCardProps {
  clawId?: string;
  clawName?: string;
  onNavigateToAdmin?: () => void;
}

export default function MemoryCard({
  clawId = "demo-claw",
  clawName = "Demo Agent",
  onNavigateToAdmin,
}: MemoryCardProps) {
  // ── State Management ──
  const [isEnabled, setIsEnabled] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [enableDialogOpen, setEnableDialogOpen] = useState(false);
  const [disableDialogOpen, setDisableDialogOpen] = useState(false);
  const [pendingToggleState, setPendingToggleState] = useState(false);

  // ── Toggle Handler ──
  const handleToggleChange = (checked: boolean) => {
    if (isLoading) return;

    setPendingToggleState(checked);

    if (checked) {
      setEnableDialogOpen(true);
    } else {
      setDisableDialogOpen(true);
    }
  };

  // ── Enable Handler ──
  const handleEnableConfirm = async () => {
    setEnableDialogOpen(false);
    setIsLoading(true);

    try {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      setIsEnabled(true);
      toast.success("Memory 功能已启用！");
    } catch (error) {
      toast.error("启用失败，请重试");
      setIsEnabled(false);
    } finally {
      setIsLoading(false);
    }
  };

  // ── Disable Handler ──
  const handleDisableConfirm = async () => {
    setDisableDialogOpen(false);
    setIsLoading(true);

    try {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      setIsEnabled(false);
      toast.success("Memory 功能已禁用");
    } catch (error) {
      toast.error("禁用失败，请重试");
      setIsEnabled(true);
    } finally {
      setIsLoading(false);
    }
  };

  // ── Cancel Handlers ──
  const handleEnableCancel = () => {
    setEnableDialogOpen(false);
  };

  const handleDisableCancel = () => {
    setDisableDialogOpen(false);
  };

  // ── Compute Status Colors - 符合设计规范的语义色 ──
  const getStatusColor = () => {
    if (isLoading) {
      return {
        indicator: "bg-yellow-500",
        text: "text-yellow-600",
        bg: "bg-yellow-50",
        border: "border-yellow-100",
      };
    }
    if (isEnabled) {
      return {
        indicator: "bg-green-500",
        text: "text-green-600",
        bg: "bg-green-50",
        border: "border-green-100",
      };
    }
    return {
      indicator: "bg-gray-400",
      text: "text-gray-600",
      bg: "bg-gray-50",
      border: "border-gray-200",
    };
  };

  const statusColor = getStatusColor();

  // ── Status Text ──
  const getStatusText = () => {
    if (isLoading) {
      return isEnabled ? "正在禁用..." : "正在启用...";
    }
    return isEnabled ? "已启用" : "未启用";
  };

  return (
    <>
      <SurfaceCard
        className="overflow-hidden flex flex-col rounded-[12px]"
        style={{ height: "476px" }}
      >
        {/* ── Header - 符合设计规范 ── */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-gray-50">
          <div className="flex items-center gap-2">
            <div 
              className="w-6 h-6 rounded-[4px] flex items-center justify-center"
              style={{ background: "linear-gradient(90deg, #020617 70%, #355EF1 100%)" }}
            >
              <Brain className="w-3.5 h-3.5 text-white" />
            </div>
            <h2 className="font-semibold text-gray-900">记忆 (TDAI-Memory)</h2>
          </div>
        </div>

        {/* ── Content ── */}
        <div className="p-6 space-y-4 flex-1 flex flex-col justify-between">
          {/* Version Badge */}
          <div>
            <div 
              className="inline-block px-3 py-1 rounded-full text-xs font-medium text-white mb-3"
              style={{ background: "linear-gradient(90deg, #020617 70%, #355EF1 100%)" }}
            >
              TDAI-Memory Free 版
            </div>

            {/* Description */}
            <p className="text-sm text-gray-600 leading-relaxed">
              腾讯云自研 Agent 记忆系统，让 Agent
              跨会话记住用户偏好、任务进度与历史决策，持续提供个性化服务。
            </p>
          </div>

          {/* Status Block - 使用设计规范的状态颜色 */}
          <div
            className={`rounded-[4px] border p-4 flex items-center justify-between ${statusColor.bg} ${statusColor.border}`}
          >
            <div className="flex items-center gap-3">
              <span className={`text-sm font-medium ${statusColor.text}`}>
                {getStatusText()}
              </span>
            </div>

            {/* Toggle Switch */}
            <Switch
              checked={isEnabled}
              onCheckedChange={handleToggleChange}
              disabled={isLoading}
              className="scale-125 origin-right"
            />
          </div>

          {/* 信息提示 - 符合设计规范的提示横幅 */}
          {!isEnabled && (
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertDescription>
                启用后，Agent 将自动记忆对话中的关键信息，为用户提供更个性化的服务体验。
              </AlertDescription>
            </Alert>
          )}
        </div>
      </SurfaceCard>

      {/* ── Dialogs ── */}
      <EnableMemoryDialog
        open={enableDialogOpen}
        onConfirm={handleEnableConfirm}
        onCancel={handleEnableCancel}
      />

      <DisableMemoryDialog
        open={disableDialogOpen}
        onConfirm={handleDisableConfirm}
        onCancel={handleDisableCancel}
      />
    </>
  );
}
