/**
 * EditQuotaDialog
 * 模型行内「编辑每日配额」二次编辑弹窗。
 * 仅一个数字字段，没有复杂表单状态，从 ModelConfig.tsx 主文件抽出。
 */
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog, DialogContent, DialogBody, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { PanelTitle, MetaMedium } from "@/components/ui/Typography";
import type { ModelRow } from "@/lib/modelConfigStore";

export interface EditQuotaDialogProps {
  model: ModelRow | null;
  open: boolean;
  onClose: () => void;
  onSave: (id: string, limit: number) => void;
}

export function EditQuotaDialog({ model, open, onClose, onSave }: EditQuotaDialogProps) {
  const [limit, setLimit] = useState(model?.dailyLimit ?? 100000);

  // 每次打开时同步
  if (!model) return null;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent
        className="sm:max-w-md"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
      >
        <DialogHeader>
          <DialogTitle asChild>
            <PanelTitle>编辑配额 — {model.name}</PanelTitle>
          </DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6 flex-1">
          <div className="space-y-1.5">
            <MetaMedium as="label" tone="secondary" className="block">每日 Tokens 数量上限</MetaMedium>
            <Input
              type="number"
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
            />
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="claw-outline" size="claw-sm" onClick={onClose}>取消</Button>
          <Button
            variant="dialog-confirm"
            size="claw-sm"
            onClick={() => { onSave(model.id, limit); onClose(); }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
