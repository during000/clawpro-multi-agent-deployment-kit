import React, { useState } from 'react';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { SurfaceInner } from '@/components/ui/Surface';
import { AdminNoticeAlert } from '@/components/ui/admin-notice-alert';
import {
  CompactText,
  MiniBodyText,
  HelperText,
} from '@/components/ui/Typography';
import { Loader2, Lock } from 'lucide-react';

// 配置常量
const FIXED_MEMORY_SPACES = 500; // 固定配额：每个用户限额 500 个记忆空间

interface ProActivationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm?: (config: { 
    autoEnableForNewInstances: boolean;
  }) => void;
}

export const ProActivationDialog: React.FC<ProActivationDialogProps> = ({
  open,
  onOpenChange,
  onConfirm,
}) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // 增量配置：新创建的 Agent 是否默认开通 Pro
  const [autoEnableForNewInstances, setAutoEnableForNewInstances] = useState(true);

  const handleConfirm = async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      await new Promise((resolve, reject) => {
        setTimeout(() => {
          if (Math.random() > 0.05) {
            resolve(true);
          } else {
            reject(new Error('网络错误，请稍后重试'));
          }
        }, 800);
      });
      
      setIsLoading(false);
      onOpenChange(false);
      onConfirm?.({ autoEnableForNewInstances });
    } catch (err) {
      setIsLoading(false);
      setError(err instanceof Error ? err.message : '开通失败，请稍后重试');
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      setError(null);
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent size="md" className="rounded-[8px]">
        <DialogHeader>
          <DialogTitle>开通 Memory Pro</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6">
          <div className="space-y-4">
            {/* 错误提示 */}
            {error && (
              <AdminNoticeAlert type="resource-alert" tagLabel="错误" className="!h-auto !items-start py-3">
                <span>{error}</span>
              </AdminNoticeAlert>
            )}

            {/* 限免活动提示 */}
            <SurfaceInner className="bg-[#FAFAFA] px-5 py-4 space-y-3">
              <CompactText as="p" tone="primary" className="font-semibold">限时免费体验（至 2026.8.15）</CompactText>
              <MiniBodyText as="p" tone="muted" className="leading-relaxed">
                免费体验期内可使用全部 Pro 能力，体验结束前我们会提前通知定价；体验期结束后<span className="font-medium text-[var(--text-emphasis)]">不会自动扣费</span>，需在控制台主动确认转为付费后方可继续使用。
              </MiniBodyText>
              <div className="pt-3 border-t border-[#EAEEF4] space-y-1.5">
                <MiniBodyText as="p" tone="muted" className="leading-relaxed">
                  开通后将获得 <span className="font-semibold text-[var(--text-emphasis)]">{FIXED_MEMORY_SPACES}</span> 个记忆空间，每个记忆空间可绑定一个 Agent。
                </MiniBodyText>
                <MiniBodyText as="p" tone="muted" className="leading-relaxed">
                  开通服务需要 3–5 分钟准备资源，准备完成后即可使用。
                </MiniBodyText>
              </div>
            </SurfaceInner>

            {/* 配置项 */}
            <div className="space-y-4">
              {/* 记忆空间配额 */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <CompactText tone="primary">记忆空间配额</CompactText>
                  <Lock className="w-3.5 h-3.5 text-[var(--text-weak)]" />
                </div>
                <div className="flex items-center gap-2">
                  <CompactText tone="primary" className="font-semibold">{FIXED_MEMORY_SPACES} 个</CompactText>
                  <HelperText as="span">如需更多请联系商务</HelperText>
                </div>
              </div>

              {/* 默认开通 */}
              <div className="flex items-center justify-between">
                <CompactText tone="primary">默认开通</CompactText>
                <div className="flex items-center gap-2">
                  <Switch 
                    checked={autoEnableForNewInstances} 
                    onCheckedChange={setAutoEnableForNewInstances}
                  />
                  <HelperText as="span">新创建的 Agent 自动开通 Pro 版</HelperText>
                </div>
              </div>
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button 
            variant="claw-outline" 
            onClick={handleClose}
            disabled={isLoading}
            className="min-w-[80px]"
          >
            取消
          </Button>
          <Button 
            variant="claw-primary"
            onClick={handleConfirm}
            disabled={isLoading}
            className="min-w-[100px]"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                开通中...
              </>
            ) : (
              '确认开通'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
