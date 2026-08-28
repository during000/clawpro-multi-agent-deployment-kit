/**
 * MultimodalConfirmDialog
 * 多模态切换二次确认（开 / 关共用）。
 *
 * 与「删除模型 AlertDialog」保持一致语言：
 *   - 用 AlertDialog 而不是 Dialog（按 SKILL §7.8 与删除一致）
 *   - 标题 PanelTitle，描述 BodyText tone="secondary"
 *   - 主按钮 variant="dialog-confirm"（开关属于"用户感知改变"动作，
 *     非纯破坏性，不上 destructive）
 */
import { Button } from "@/components/ui/button";
import {
  AlertDialog, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { PanelTitle, BodyText } from "@/components/ui/Typography";
import type { ModelRow } from "@/lib/modelConfigStore";

export interface MultimodalConfirmState {
  model: ModelRow;
  enable: boolean;
}

export interface MultimodalConfirmDialogProps {
  state: MultimodalConfirmState | null;
  onClose: () => void;
  onConfirm: (state: MultimodalConfirmState) => void;
}

export function MultimodalConfirmDialog({ state, onClose, onConfirm }: MultimodalConfirmDialogProps) {
  return (
    <AlertDialog open={!!state} onOpenChange={(open) => { if (!open) onClose(); }}>
      <AlertDialogContent className="sm:max-w-[420px]">
        <AlertDialogHeader>
          <AlertDialogTitle asChild>
            <PanelTitle>
              {state?.enable ? "开启多模态" : "关闭多模态"}
            </PanelTitle>
          </AlertDialogTitle>
        </AlertDialogHeader>
        <AlertDialogDescription asChild>
          <BodyText tone="secondary">
            {state?.enable
              ? `确认开启「${state.model.name}」的多模态属性么？开启后用户可在对话中上传图片等多模态内容。`
              : `确认关闭「${state?.model.name}」的多模态属性么？关闭后用户将无法在该模型下上传图片等多模态内容。`
            }
          </BodyText>
        </AlertDialogDescription>
        <AlertDialogFooter>
          <Button variant="claw-outline" size="claw-sm" onClick={onClose}>取消</Button>
          <Button
            variant="dialog-confirm"
            size="claw-sm"
            onClick={() => { if (state) onConfirm(state); }}
          >
            {state?.enable ? "确认开启" : "确认关闭"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
