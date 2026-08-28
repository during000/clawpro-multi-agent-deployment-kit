/**
 * 删除 Skill 确认对话框（F-04）
 * - 被技能包引用时，列出包名称 + 警告
 * - 无引用时，简洁确认
 * - 使用 AlertDialog 实现危险操作确认
 *
 * 遵循项目标准警示弹窗规范：
 *  - 标题使用黑色（#0A0A0A）
 *  - 正文普通文字使用 MetaText as="p" tone="primary"
 *  - 强调文字使用告警色 tone="danger"
 *  - 警示信息使用 Alert destructive 变体（带 CircleAlert 图标）
 *  - 主按钮使用 destructive variant（红底白字）
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
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { BodyText, BodyMedium } from '@/components/ui/Typography';
import { CircleAlert } from 'lucide-react';

interface DeleteSkillDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillName: string;
  /** 引用了该 Skill 的技能包名称列表 */
  referencedPackages?: string[];
  onConfirm: () => void;
}

export default function DeleteSkillDialog({
  open,
  onOpenChange,
  skillName,
  referencedPackages = [],
  onConfirm,
}: DeleteSkillDialogProps) {
  const hasReferences = referencedPackages.length > 0;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="sm:max-w-[420px]">
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
        </AlertDialogHeader>
        <AlertDialogDescription asChild>
          <div className="space-y-3">
            {hasReferences ? (
              <>
                <div className="space-y-1">
                  <BodyText as="p" tone="primary">
                    确定要删除 Skill「<BodyMedium as="span" tone="primary">{skillName}</BodyMedium>」吗？
                  </BodyText>
                  <BodyText as="p" tone="danger">此操作不可撤销。</BodyText>
                </div>
                <Alert variant="warning">
                  <CircleAlert />
                  <AlertTitle>该 Skill 被以下技能包引用</AlertTitle>
                  <AlertDescription>
                    <ul className="list-disc pl-4 space-y-0.5">
                      {referencedPackages.map((name) => (
                        <li key={name}><BodyText as="span" tone="primary">{name}</BodyText></li>
                      ))}
                    </ul>
                  </AlertDescription>
                </Alert>
                <BodyText as="p" tone="primary">
                  删除后将自动从上述技能包中移除该技能。
                </BodyText>
              </>
            ) : (
              <div className="space-y-1">
                <BodyText as="p" tone="primary">
                  确定要删除 Skill「<BodyMedium as="span" tone="primary">{skillName}</BodyMedium>」吗？
                </BodyText>
                <BodyText as="p" tone="danger">此操作不可撤销。</BodyText>
              </div>
            )}
          </div>
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>
            确认删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
