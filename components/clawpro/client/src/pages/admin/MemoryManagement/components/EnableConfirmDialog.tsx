import React, { useState } from 'react';
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

interface EnableConfirmDialogProps {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * 开启 Memory Free 版确认弹窗
 * 完全按照 HTML 设计文件实现
 */
export const EnableConfirmDialog: React.FC<EnableConfirmDialogProps> = ({
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
    toast.success('已开启 Memory Free 版，正在为所有实例开启记忆插件');
    setIsChecked(false);
  };

  const handleCancel = () => {
    setIsChecked(false);
    onCancel();
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && handleCancel()}>
      <DialogContent className="max-w-[500px] rounded-[8px]">
        <DialogHeader className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-full bg-gradient-to-br from-[#007AFF] to-[#5856D6] flex items-center justify-center flex-shrink-0">
            <span className="text-xl">✓</span>
          </div>
          <div>
            <DialogTitle>开启 Memory Free 版</DialogTitle>
            <DialogDescription>
              确认后将在所有实例上安装并启用记忆功能
            </DialogDescription>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          {/* 蓝色提示框 - 开启后效果 */}
          <AdminNoticeAlert type="product-news" tagLabel="开启后效果" className="!h-auto !items-start py-3">
            <div className="flex flex-col gap-1">
              <CompactText tone="inherit" className="font-semibold">开启后效果</CompactText>
              <ul className="list-disc pl-4 space-y-0.5">
                <li><CompactText tone="inherit">新创建的 Agent 将<strong>默认安装并启用</strong> Memory Free 版记忆插件。</CompactText></li>
                <li><CompactText tone="inherit">所有现有 Agent 将会<strong>自动安装</strong>此插件，安装过程需要重启 Agent Gateway 服务。</CompactText></li>
              </ul>
            </div>
          </AdminNoticeAlert>

          {/* 黄色警告框 */}
          <AdminNoticeAlert type="pending-config" tagLabel="注意" className="!h-auto !items-start py-3">
            <CompactText tone="inherit" className="leading-relaxed">
              <strong>请注意：此操作涉及所有现有实例。</strong>安装过程中，实例的 Gateway 服务将重启，会导致<strong>服务短暂中断（约 1 分钟/实例）</strong>。建议避开业务高峰期进行操作。
            </CompactText>
          </AdminNoticeAlert>

          {/* 确认复选框 */}
          <div className="flex items-center gap-2">
            <Checkbox
              id="enableCheck"
              checked={isChecked}
              onCheckedChange={(checked) => setIsChecked(checked as boolean)}
            />
            <label htmlFor="enableCheck" className="cursor-pointer">
              <CompactText tone="secondary">我已了解上述说明，确认开启</CompactText>
            </label>
          </div>
        </div>

        <DialogFooter className="flex gap-3 justify-end">
          <Button variant="claw-outline" onClick={handleCancel}>
            取消
          </Button>
          <Button
            variant="claw-primary"
            onClick={handleConfirm}
            disabled={!isChecked}
          >
            确认开启
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
