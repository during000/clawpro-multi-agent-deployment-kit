/**
 * TokenValueEditor - Token 配额值编辑器
 *
 * 触发器是一个仿 Select 的下拉按钮，
 * 点击后弹出 Popover：顶部 SegmentGroup（无限制/自定义），
 * 选择自定义时显示 Input；底部右对齐取消/确认按钮。
 * 仅在点击「确认」时通过 onCommit 同步外部状态。
 */
import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { toast } from "sonner";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

export interface TokenValueEditorProps {
  mode: "custom" | "unlimited";
  valStr: string;
  onCommit: (nextMode: "custom" | "unlimited", nextValStr: string) => void;
  /** 可选：覆盖触发按钮的 className */
  triggerClassName?: string;
}

export function TokenValueEditor({ mode, valStr, onCommit, triggerClassName }: TokenValueEditorProps) {
  const [open, setOpen] = useState(false);
  const [draftMode, setDraftMode] = useState<"custom" | "unlimited">(mode);
  const [draftValStr, setDraftValStr] = useState<string>(valStr);

  // 每次打开时，用当前外部值初始化草稿
  const handleOpenChange = (v: boolean) => {
    if (v) {
      setDraftMode(mode);
      setDraftValStr(valStr);
    }
    setOpen(v);
  };

  const handleConfirm = () => {
    if (draftMode === "custom") {
      const n = parseInt(draftValStr, 10);
      if (isNaN(n) || n < 0) {
        toast.error("请输入有效数值");
        return;
      }
    }
    onCommit(draftMode, draftMode === "unlimited" ? "" : draftValStr);
    setOpen(false);
  };

  const triggerLabel =
    mode === "unlimited"
      ? "无限制"
      : valStr === ""
        ? ""
        : Number(valStr).toLocaleString();
  const isPlaceholder = mode === "custom" && valStr === "";

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          data-state={open ? "open" : "closed"}
          className={triggerClassName || "group relative w-32 h-9 px-3 pr-8 rounded-[4px] border border-[var(--border)] bg-white hover:border-blue-500 data-[state=open]:border-blue-500 transition-colors cursor-pointer flex items-center text-left text-sm"}
        >
          <span
            className={`truncate ${isPlaceholder ? "text-[var(--text-weak)]" : "text-[var(--text-title)]"} tabular-nums`}
          >
            {isPlaceholder ? "请输入" : triggerLabel}
          </span>
          <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 size-4 text-[var(--text-muted)] transition-transform duration-200 group-data-[state=open]:rotate-180 pointer-events-none" />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)]"
        align="start"
        sideOffset={4}
        style={{ width: 240 }}
      >
        <div className="p-2 space-y-2">
          <SegmentGroup className="w-full">
            <SegmentOption
              active={draftMode === "unlimited"}
              onClick={() => setDraftMode("unlimited")}
              className="flex-1"
            >
              无限制
            </SegmentOption>
            <SegmentOption
              active={draftMode === "custom"}
              onClick={() => setDraftMode("custom")}
              className="flex-1"
            >
              自定义
            </SegmentOption>
          </SegmentGroup>
          {draftMode === "unlimited" && (
            <p className="text-xs text-[var(--text-muted)] leading-relaxed">不限制数量上限</p>
          )}
          {draftMode === "custom" && (
            <Input
              type="number"
              autoFocus
              value={draftValStr}
              onChange={(e) => setDraftValStr(e.target.value)}
              className="h-9 text-xs bg-white"
              placeholder="请输入数量"
            />
          )}
        </div>
        <div className="flex items-center justify-end gap-2 mx-2 border-t border-[#EAEEF4] py-2">
          <Button size="sm" variant="outline" className="h-7 text-xs px-3" onClick={() => setOpen(false)}>取消</Button>
          <Button
            size="sm"
            variant="dialog-confirm"
            className="h-7 text-xs px-3"
            disabled={draftMode === "custom" && draftValStr.trim() === ""}
            onClick={handleConfirm}
          >
            确认
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export default TokenValueEditor;
