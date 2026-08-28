/**
 * EnableMemoryDialog - 启用 Memory 确认弹窗
 * 
 * 功能：
 * - 显示启用 Memory 的风险提示
 * - 橙色警告区块提示 Gateway 重启
 * - 二次确认机制
 */

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { CircleAlert } from "lucide-react";

interface EnableMemoryDialogProps {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function EnableMemoryDialog({
  open,
  onConfirm,
  onCancel,
}: EnableMemoryDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onCancel()}>
      <DialogContent className="sm:max-w-md rounded-[4px]">
        <DialogHeader>
          <DialogTitle className="text-lg font-bold text-gray-900">
            启用 Memory 记忆功能
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Main Description */}
          <DialogDescription className="text-sm text-gray-600 leading-relaxed">
            启用 Memory 记忆功能需要安装 TDAI-Memory 记忆插件，安装过程中将自动重启
            Agent Gateway 服务。
          </DialogDescription>

          {/* Warning Block */}
          <div className="rounded-[4px] border border-amber-200 bg-amber-50 p-3 flex gap-3">
            <CircleAlert className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
            <div className="text-sm text-amber-900 leading-relaxed">
              <p className="font-medium mb-1">⚠️ 重启期间 Gateway 服务将短暂不可用</p>
              <p>
                （1 分钟以内），请确认当前没有重要对话进行中。
              </p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3 justify-end pt-2">
            <Button
              variant="claw-outline"
              onClick={onCancel}
            >
              取消
            </Button>
            <Button
              variant="claw-primary"
              onClick={onConfirm}
            >
              确认启用
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
