/**
 * 用户端 · 申请下架 Skill Dialog
 *
 * 员工端从技能广场卡片入口触发；仅对「我上传的」技能显示入口。
 * 复用 `@/components/ui/dialog`（tenant.md §7：Dialog 按组件 spec 分流，不套 12px TenantCard）。
 */

import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { MetaText } from '@/components/ui/Typography';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CircleAlert } from 'lucide-react';
import { addOffshelfRequest } from './myRequestsStore';

interface OffshelfSkillDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * 关联的 skill id。技能广场卡片入口必传；从「我的申请」里点已发布记录
   * 触发时，其对应 skill 可能不在 MOCK_SKILLS 中，此时可省略。
   */
  skillId?: string;
  skillName: string;
  version?: string;
}

export default function OffshelfSkillDialog({
  open,
  onOpenChange,
  skillId,
  skillName,
  version,
}: OffshelfSkillDialogProps) {
  const [reason, setReason] = useState('');

  // 关闭时清空输入
  useEffect(() => {
    if (!open) {
      setReason('');
    }
  }, [open]);

  const handleSubmit = () => {
    const trimmed = reason.trim();
    if (!trimmed) {
      toast.error('请先填写下架原因');
      return;
    }
    addOffshelfRequest({
      skillId,
      skillName,
      version,
      offshelfReason: trimmed,
    });
    toast.success('下架申请已提交');
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="sm"
        className="max-h-[min(90vh,560px)] flex flex-col"
        onPointerDownOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>申请下架 Skill</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6">
          <div className="space-y-4 py-1">
            <div>
              <MetaText tone="secondary">技能</MetaText>
              <div className="mt-1 text-[14px] leading-5 text-[var(--text-emphasis)] font-medium">
                {skillName}
                {version ? (
                  <span className="ml-2 text-[var(--text-muted)] font-normal">
                    v{version}
                  </span>
                ) : null}
              </div>
            </div>

            <Alert variant="warning">
              <CircleAlert />
              <AlertDescription>
                下架后将停止新安装；已经安装到 Agent 的副本仍可继续使用。
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="offshelf-reason">
                下架原因 <span className="text-red-500">*</span>
              </Label>
              <Textarea
                id="offshelf-reason"
                placeholder="请说明下架该技能的原因（如：存在质量问题、已被更好的技能替代等）"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={4}
              />
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button variant="dialog-confirm" onClick={handleSubmit}>
            提交下架申请
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
