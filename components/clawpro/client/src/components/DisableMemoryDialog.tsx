/**
 * DisableMemoryDialog - 禁用 Memory 确认弹窗
 * 
 * 功能：
 * - 显示禁用 Memory 的风险提示
 * - 强调数据保留但不被使用
 * - 橙色警告区块提示 Gateway 重启
 * - 红色确认按钮表示危险操作
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

interface DisableMemoryDialogProps {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function DisableMemoryDialog({
  open,
  onConfirm,
  onCancel,
}: DisableMemoryDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onCancel()}>
      <DialogContent className="sm:max-w-md rounded-[4px]">
        <DialogHeader>
          <DialogTitle className="text-lg font-bold text-gray-900">
            禁用 Memory 记忆功能
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Main Description */}
          <DialogDescription className="text-sm text-gray-600 leading-relaxed">
            禁用后将卸载 TDAI-Memory 记忆插件，已有记忆数据不会删除，但也不会被使用。卸载过程中将自动重启
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
              variant="destructive"
              onClick={onConfirm}
            >
              确认禁用
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
