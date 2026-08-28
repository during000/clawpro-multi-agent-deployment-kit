import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { ShieldCheck, CircleAlert, Upload } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from '@/components/ui/dialog';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { BodyText, MetaText, MetaMedium } from '@/components/ui/Typography';
import { Skill, Category } from './types';

interface SkillReviewDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skill: Skill | null;
  categories: Category[];
  /**
   * 通过审核：
   * - reviewType='publish' → 把审核态改为 normal（正常发布）
   * - reviewType='offshelf' → 把审核态改为 offlined（下架）
   * 管控端不允许修改申请人提交时选中的标签分组
   */
  onApprove: (skillId: string, reviewType: 'publish' | 'offshelf') => void;
  /**
   * 驳回：
   * - reviewType='publish' → 从列表移除（员工再走一次流程更干净）
   * - reviewType='offshelf' → 回到 normal（保留 Skill 可用）
   */
  onReject: (skillId: string, reviewType: 'publish' | 'offshelf', reason: string) => void;
}

/**
 * 单一审核弹窗（对齐管控端 Demo #reviewOverlay）
 * - 顶部展示「申请类型 pill」：发布申请（蓝） / 下架申请（橙），避免管理员误判
 * - 只读详情：显示名称 / slug / 申请人 / 申请时间 / 描述 / 技能标签
 *   下架申请额外展示「下架原因」（员工填写）
 * - 安全审核状态条（当前 mock 数据固定"安全审核已通过"）
 * - 驳回原因 Textarea（驳回时必填）
 * - 底部：驳回 / 通过并发布 or 通过并下架（按类型自动切换）
 */
export default function SkillReviewDialog({
  open,
  onOpenChange,
  skill,
  categories,
  onApprove,
  onReject,
}: SkillReviewDialogProps) {
  const [rejectReason, setRejectReason] = useState<string>('');

  // 每次打开重置表单
  useEffect(() => {
    if (open) {
      setRejectReason('');
    }
  }, [open]);

  if (!skill) return null;

  const reviewType = skill.reviewType ?? 'publish';
  const isOffshelf = reviewType === 'offshelf';

  // 申请人选中的标签（只读展示）
  const submittedCategoryLabels = (skill.categories ?? [])
    .map((cid) => categories.find((c) => c.id === cid)?.name)
    .filter(Boolean)
    .join('、') || '-';

  const securityStatus = skill.securityInfo?.overallStatus ?? 'not_scanned';
  const isSecuritySafe = securityStatus === 'safe';

  const handleReject = () => {
    const reason = rejectReason.trim();
    if (!reason) {
      toast.error('请填写驳回原因');
      return;
    }
    onReject(skill.id, reviewType, reason);
    onOpenChange(false);
  };

  const handleApprove = () => {
    onApprove(skill.id, reviewType);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-[640px]"
        style={{ maxHeight: 'min(90vh, 820px)', display: 'flex', flexDirection: 'column' }}
      >
        <DialogHeader>
          <DialogTitle>审核技能申请</DialogTitle>
        </DialogHeader>

        <DialogBody className="flex-1 px-6 space-y-5">
          {/* 申请类型醒目提示条 —— 使用 clawpro Alert 组件：
              - 发布申请 → info（蓝底，Upload icon）
              - 下架申请 → warning（橙底，CircleAlert icon，需谨慎语义） */}
          {isOffshelf ? (
            <Alert variant="warning">
              <CircleAlert />
              <AlertTitle>下架申请</AlertTitle>
              <AlertDescription>
                员工发起下架，通过后该技能仍保留在企业技能库，但其他成员将无法搜索到；已安装用户不受影响。
              </AlertDescription>
            </Alert>
          ) : (
            <Alert variant="info">
              <Upload />
              <AlertTitle>发布申请</AlertTitle>
              <AlertDescription>
                员工提交的新技能，通过后将在企业技能库中对所有成员可见。
              </AlertDescription>
            </Alert>
          )}

          {/* 只读详情 —— 对齐 Demo：两列 grid，label 在上/值在下，每格底部分隔线 */}
          <div className="grid grid-cols-2 gap-x-6 gap-y-0">
            <div className="py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">显示名称</MetaText>
              <BodyText tone="primary">{skill.name}</BodyText>
            </div>
            <div className="py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">唯一标识</MetaText>
              <BodyText tone="primary" className="font-mono">{skill.slug}</BodyText>
            </div>
            <div className="py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">申请人</MetaText>
              <BodyText tone="primary">{skill.applicant ?? '-'}</BodyText>
            </div>
            <div className="py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">申请时间</MetaText>
              <BodyText tone="primary" className="tabular-nums">{skill.submittedAt ?? '-'}</BodyText>
            </div>
            <div className="py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">技能标签</MetaText>
              <BodyText tone="primary">{submittedCategoryLabels}</BodyText>
            </div>
            <div className="py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">版本</MetaText>
              <BodyText tone="primary" className="font-mono">{skill.version}</BodyText>
            </div>
            <div className="col-span-2 py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
              <MetaText tone="muted" className="block mb-1">描述</MetaText>
              <BodyText tone="primary" className="leading-relaxed whitespace-pre-wrap">
                {skill.description || '-'}
              </BodyText>
            </div>
            {isOffshelf && (
              <div className="col-span-2 py-2.5 border-b border-[var(--cp-border,#EAEEF4)]">
                <MetaText tone="muted" className="block mb-1">下架原因</MetaText>
                <BodyText tone="primary" className="leading-relaxed whitespace-pre-wrap">
                  {skill.offshelfReason || '-'}
                </BodyText>
              </div>
            )}
          </div>

          {/* 安全审核状态条 —— 对齐 Demo 的 accent pill（浅蓝底较宽） */}
          <div
            className={
              isSecuritySafe
                ? 'flex items-center gap-2 rounded-[4px] bg-green-50 px-3 py-2.5'
                : 'flex items-center gap-2 rounded-[4px] bg-[var(--bg-grey-normal,#FAFBFD)] px-3 py-2.5'
            }
          >
            <ShieldCheck
              className={isSecuritySafe ? 'w-4 h-4 text-green-600 shrink-0' : 'w-4 h-4 text-[var(--text-weak)] shrink-0'}
            />
            <BodyText tone={isSecuritySafe ? 'primary' : 'secondary'}>
              {isSecuritySafe ? '安全审核已通过' : '安全审核未完成'}
            </BodyText>
          </div>

          {/* 驳回原因 */}
          <div>
            <MetaMedium as="label" tone="secondary">驳回原因</MetaMedium>
            <Textarea
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder="驳回时必填"
              rows={3}
              className="resize-none min-h-[76px] mt-1.5"
            />
            <MetaText tone="muted" className="block mt-1.5">
              {isOffshelf
                ? '驳回后该技能将回到「正常」状态，员工可再次发起下架申请'
                : '驳回后该技能将从列表移除，员工需要重新发布申请'}
            </MetaText>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={handleReject}>
            驳回
          </Button>
          <Button variant="dialog-confirm" onClick={handleApprove}>
            {isOffshelf ? '通过并下架' : '通过并发布'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
