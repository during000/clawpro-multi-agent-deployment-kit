/**
 * 下架 Skill 确认对话框
 * - 管控端管理员从「更多」→「下架」触发
 * - 下架 ≠ 删除：Skill 依然保留在企业技能库，仅屏蔽员工端搜索，已安装不受影响
 * - 主按钮使用 default variant（主色，非 destructive）以区分删除
 * - 点击遮罩 / ESC / 取消按钮 均视为取消（Radix AlertDialog 默认行为）
 *
 * 遵循项目警示弹窗规范：
 *  - 结构：AlertDialogHeader 仅放 Title，Description 与 Header/Footer 平级
 *  - 使用 warning Alert（橙底）承载后果说明
 */
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from '@/components/ui/alert-dialog';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { BodyText, BodyMedium } from '@/components/ui/Typography';
import { CircleAlert } from 'lucide-react';

interface OfflineSkillDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillName: string;
  onConfirm: () => void;
}

export default function OfflineSkillDialog({
  open,
  onOpenChange,
  skillName,
  onConfirm,
}: OfflineSkillDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="sm:max-w-[440px]">
        <AlertDialogHeader>
          <AlertDialogTitle>确认下架</AlertDialogTitle>
        </AlertDialogHeader>
        <AlertDialogDescription asChild>
          <div className="space-y-3">
            <BodyText as="p" tone="primary">
              确定要下架 Skill「<BodyMedium as="span" tone="primary">{skillName}</BodyMedium>」吗？
            </BodyText>
            <Alert variant="warning">
              <CircleAlert />
              <AlertDescription>
                下架后该技能仍保留在企业技能库，但其他成员将无法搜索到；已安装用户不受影响。
              </AlertDescription>
            </Alert>
          </div>
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction variant="default" onClick={onConfirm}>
            确认下架
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
