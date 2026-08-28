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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { BodyText, BodyMedium, MetaText } from '@/components/ui/Typography';
import { Loader2 } from 'lucide-react';


interface ProCloseDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm?: () => void;
  ocCount: number;
  onGoToInstanceList?: () => void;
  /** 当前已配置的组织策略数量（不含预设策略），用于在影响列表中提示将一并清空 */
  groupPolicyCount?: number;
  /** 当前预设策略版本，若为 'pro' 则提示关闭后将自动回落为 Free 版 */
  presetVersion?: 'none' | 'free' | 'pro';
}

export const ProCloseDialog: React.FC<ProCloseDialogProps> = ({
  open,
  onOpenChange,
  onConfirm,
  ocCount,
  onGoToInstanceList,
  groupPolicyCount = 0,
  presetVersion,
}) => {
  const [confirmText, setConfirmText] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleClose = () => {
    setConfirmText('');
    onOpenChange(false);
  };

  const handleConfirm = async () => {
    setIsLoading(true);
    
    // 模拟 API 调用
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    setIsLoading(false);
    setConfirmText('');
    onOpenChange(false);
    onConfirm?.();
  };

  const handleGoToInstanceList = () => {
    handleClose();
    onGoToInstanceList?.();
  };

  const isConfirmValid = confirmText === '确认关闭';
  const hasActiveInstances = ocCount > 0;

  // 如果还有已开通的实例，显示拦截提示
  if (hasActiveInstances) {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="sm:max-w-[480px] rounded-[8px]">
          <DialogHeader>
            <DialogTitle>无法关闭服务</DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6">
            <div className="space-y-4">
              <Alert variant="warning">
                <AlertInfoIcon />
                <AlertDescription>请先关闭所有实例的 Memory Pro，再执行关闭服务操作。</AlertDescription>
              </Alert>
              <BodyText as="p" tone="secondary" className="leading-relaxed">
                当前还有 <BodyMedium as="span" tone="primary">{ocCount}</BodyMedium> 个实例开通了 Memory Pro 服务。您可以前往实例列表，使用「批量关闭」功能快速关闭多个实例的 Memory Pro。
              </BodyText>
            </div>
          </DialogBody>

          <DialogFooter className="gap-2 sm:gap-2">
            <Button variant="claw-outline" onClick={handleClose}>
              我知道了
            </Button>
            <Button variant="claw-primary" onClick={handleGoToInstanceList}>
              前往实例列表
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // 没有已开通的实例，允许关闭服务
  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[520px] rounded-[8px]">
        <DialogHeader>
          <DialogTitle>关闭 Memory Pro 服务</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6">
          <div className="space-y-4">
            <BodyText tone="muted">
              当前没有实例开通 Memory Pro 服务，可以安全关闭。
            </BodyText>

            {/* 关闭影响说明 */}
            <div className="space-y-2">
              <BodyMedium as="div" tone="primary">关闭后将产生以下影响：</BodyMedium>
              <ul className="space-y-1.5">
                <li className="flex items-start gap-2">
                  <span className="mt-1.5 w-1 h-1 rounded-full bg-[var(--text-muted)] flex-shrink-0" />
                  <BodyText tone="secondary">新创建的实例将<strong>无法开通 Memory Pro</strong></BodyText>
                </li>
                <li className="flex items-start gap-2">
                  <span className="mt-1.5 w-1 h-1 rounded-full bg-[var(--text-muted)] flex-shrink-0" />
                  <BodyText tone="secondary">随 Pro 默认启用的能力将一并失效</BodyText>
                </li>
                {((presetVersion && presetVersion !== 'none') || groupPolicyCount > 0) && (
                  <li className="flex items-start gap-2">
                    <span className="mt-1.5 w-1 h-1 rounded-full bg-[var(--text-muted)] flex-shrink-0" />
                    <BodyText tone="secondary">新建 Agent 默认记忆版本的<strong>预设策略将切换为「关闭」，已配置的组织策略将一并清空</strong>，需重新使用时请重新添加</BodyText>
                  </li>
                )}
                <li className="flex items-start gap-2">
                  <span className="mt-1.5 w-1 h-1 rounded-full bg-[var(--text-muted)] flex-shrink-0" />
                  <BodyText tone="secondary">如需重新使用 Memory Pro，需要重新开通服务</BodyText>
                </li>
              </ul>
            </div>

            {/* 二次确认 */}
            <div className="space-y-2 pt-2">
              <Label className="text-left block">
                请输入 <strong className="text-[var(--text-danger)]">确认关闭</strong> 以继续：
              </Label>
              <Input
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                placeholder='输入"确认关闭"'
                className={confirmText && !isConfirmValid ? 'border-red-300' : ''}
              />
              {confirmText && !isConfirmValid && (
                <MetaText as="p" tone="danger">请输入正确的确认文字</MetaText>
              )}
            </div>
          </div>
        </DialogBody>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="claw-outline" onClick={handleClose} disabled={isLoading}>
            取消
          </Button>
          <Button 
            variant="destructive"
            onClick={handleConfirm}
            disabled={!isConfirmValid || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                关闭中...
              </>
            ) : (
              '确认关闭服务'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
