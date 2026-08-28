import React, { useState } from 'react';
import { AlertTriangle } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { AdminNoticeAlert } from '@/components/ui/admin-notice-alert';
import { CompactText } from '@/components/ui/Typography';
import { toast } from 'sonner';

interface DisableConfirmDialogProps {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * 关闭 Memory Free 版确认弹窗
 * 使用 Dialog / AdminNoticeAlert / Button / Typography 组件，禁止手写样式
 */
export const DisableConfirmDialog: React.FC<DisableConfirmDialogProps> = ({
  open,
  onConfirm,
  onCancel,
}) => {
  const [isChecked, setIsChecked] = useState(false);

  const handleConfirm = () => {
    if (!isChecked) {
      toast.error('请勾选确认框');
      return;
    }
    onConfirm();
    toast.success('已关闭 Memory Free 版');
    setIsChecked(false);
  };

  const handleCancel = () => {
    setIsChecked(false);
    onCancel();
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && handleCancel()}>
      <DialogContent className="max-w-[500px] rounded-[8px]">
        <DialogHeader className="flex flex-row items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-[var(--surface-danger,#FEE2E2)] flex items-center justify-center flex-shrink-0">
            <AlertTriangle className="w-5 h-5 text-[var(--text-danger)]" />
          </div>
          <div className="flex-1">
            <DialogTitle>关闭 Memory Free 版</DialogTitle>
            <DialogDescription>
              此操作将立即禁用所有实例的记忆功能
            </DialogDescription>
          </div>
        </DialogHeader>

        <div className="space-y-3">
          {/* 关闭后效果 - 黄色提示 */}
          <AdminNoticeAlert
            type="pending-config"
            tagLabel="关闭后效果"
            className="!h-auto !items-start py-3"
            title="关闭后效果"
            description={
              <ul className="space-y-1 mt-1">
                <li>• 新创建的 Agent 将<strong>不再默认启用</strong>记忆功能。</li>
                <li>• 所有现有实例的记忆插件将被<strong>禁用</strong>（插件保留，但停止工作）。</li>
                <li>• 已产生的记忆数据不会删除，重新开启后可恢复使用。</li>
              </ul>
            }
          />

          {/* 重要提示 - 红色告警 */}
          <AdminNoticeAlert
            type="resource-alert"
            tagLabel="重要提示"
            className="!h-auto !items-start py-3"
            title="重要提示"
            description={
              <span>
                关闭后，所有现有实例将<strong>立即失去记忆能力</strong>，对话将回退到无记忆状态。请务必提前通知相关用户。
              </span>
            }
          />

          {/* 确认复选框 */}
          <div className="flex items-center gap-2">
            <Checkbox
              id="disableCheck"
              checked={isChecked}
              onCheckedChange={(checked) => setIsChecked(checked as boolean)}
            />
            <label htmlFor="disableCheck" className="cursor-pointer">
              <CompactText tone="secondary">我已了解上述说明，确认关闭</CompactText>
            </label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="claw-outline" size="claw-sm" onClick={handleCancel}>
            取消
          </Button>
          <Button
            variant="destructive"
            size="claw-sm"
            onClick={handleConfirm}
            disabled={!isChecked}
          >
            确认关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
