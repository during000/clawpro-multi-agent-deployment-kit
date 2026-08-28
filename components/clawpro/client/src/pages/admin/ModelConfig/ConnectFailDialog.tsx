/**
 * ConnectFailDialog
 * 「连通性检测失败」错误弹窗：把 API 返回的 JSON 报错原文以 CodeText 展示，
 * 用户点「我知道了」关闭。
 *
 * 触发于 ModelConfig 内 handleConnectTest，仅展示型，不修改数据。
 */
import { CircleAlert } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { PanelTitle, CodeText } from "@/components/ui/Typography";

export interface ConnectFailDialogProps {
  /** 接口返回的 JSON 文本；为 null 表示弹窗关闭 */
  result: string | null;
  onClose: () => void;
}

export function ConnectFailDialog({ result, onClose }: ConnectFailDialogProps) {
  return (
    <Dialog open={!!result} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle asChild>
            <PanelTitle tone="danger" className="flex items-center gap-2">
              <CircleAlert className="w-5 h-5" />
              模型连接失败
            </PanelTitle>
          </DialogTitle>
        </DialogHeader>
        <div className="py-2">
          <CodeText
            as="pre"
            className="block rounded-[var(--radius)] bg-[var(--bg-grey-normal)] border border-[var(--cp-border)] p-3 whitespace-pre-wrap break-all"
          >
            {result}
          </CodeText>
        </div>
        <DialogFooter>
          <Button variant="dialog-confirm" onClick={onClose}>
            我知道了
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
