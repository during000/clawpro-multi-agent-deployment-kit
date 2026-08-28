/**
 * DeleteModelDialog
 * 模型删除二次确认弹窗。
 *
 * 视觉规范说明：
 *   - 标题用 PanelTitle（强调文字 #0A0A0A）
 *   - 正文 BodyText tone="primary"，强调段 BodyText tone="danger"
 *   - 主按钮 variant="destructive"（红底白字）
 *   - 关闭由 AlertDialog 自带 onOpenChange 处理
 *
 * 注意：onConfirm 回调由调用方决定是否同步清理 default 模型 storage。
 */
import { Button } from "@/components/ui/button";
import {
  AlertDialog, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { PanelTitle, BodyText, BodyMedium } from "@/components/ui/Typography";
import type { ModelRow } from "@/lib/modelConfigStore";

export interface DeleteModelDialogProps {
  model: ModelRow | null;
  onClose: () => void;
  onConfirm: (model: ModelRow) => void;
}

export function DeleteModelDialog({ model, onClose, onConfirm }: DeleteModelDialogProps) {
  return (
    <AlertDialog open={!!model} onOpenChange={(open) => { if (!open) onClose(); }}>
      <AlertDialogContent className="sm:max-w-[420px]">
        <AlertDialogHeader>
          <AlertDialogTitle asChild>
            <PanelTitle>确认删除模型？</PanelTitle>
          </AlertDialogTitle>
        </AlertDialogHeader>
        <AlertDialogDescription asChild>
          <BodyText as="p" tone="primary">
            确定要删除模型 <BodyMedium tone="primary">{model?.name}</BodyMedium>（{model?.version}）吗？
            <BodyText as="span" tone="danger">
              {model?.isDefault && '该模型当前为默认模型，删除后将取消默认设置。'}删除后用户将无法使用该模型，此操作不可撤销。
            </BodyText>
          </BodyText>
        </AlertDialogDescription>

        <AlertDialogFooter>
          <Button variant="claw-outline" size="claw-sm" onClick={onClose}>
            取消
          </Button>
          <Button
            variant="destructive"
            size="claw-sm"
            onClick={() => { if (model) onConfirm(model); }}
          >
            确认删除
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
