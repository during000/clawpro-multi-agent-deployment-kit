import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export type SwitchableAgentType = "openclaw" | "hermes";

const TYPE_LABEL: Record<SwitchableAgentType, string> = {
  openclaw: "OpenClaw",
  hermes: "Hermes",
};

export function AgentTypeSwitchDialog({
  open,
  onOpenChange,
  currentType,
  onConfirm,
  onViewBackups,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentType: SwitchableAgentType;
  onConfirm: (nextType: SwitchableAgentType) => void;
  onViewBackups: () => void;
}) {
  const targetType: SwitchableAgentType = currentType === "openclaw" ? "hermes" : "openclaw";
  const targetLabel = TYPE_LABEL[targetType];

  const handleConfirm = () => {
    onOpenChange(false);
    onConfirm(targetType);
  };

  const handleViewBackups = () => {
    onOpenChange(false);
    onViewBackups();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md" data-testid="agent-type-switch-dialog">
        <DialogHeader>
          <DialogTitle>切换为 {targetLabel}？</DialogTitle>
          <DialogDescription className="text-xs leading-5">
            当前实例将重装并更换为 {targetLabel}，原运行环境会被清除。
          </DialogDescription>
        </DialogHeader>

        <ol className="list-decimal space-y-1.5 pl-5 pr-2 text-xs leading-5 text-[var(--text-secondary)] marker:font-medium marker:text-[var(--text-title)]">
          <li>
            切换前将自动备份原 Agent 数据
            <button
              type="button"
              className="ml-2 text-[11px] font-medium leading-4 text-[var(--text-brand)] underline decoration-transparent underline-offset-4 outline-none transition-colors hover:decoration-current focus-visible:decoration-current focus-visible:outline-none"
              onClick={handleViewBackups}
              data-testid="view-agent-type-switch-backups"
            >
              查看数据备份
            </button>
          </li>
          <li>配置、模型、技能、记忆和角色设定将自动导入新 Agent</li>
          <li>历史会话仅随备份保留，临时文件、缓存和日志不会导入</li>
        </ol>

        <DialogFooter className="pt-3">
          <Button variant="tenant-outline" size="claw-sm" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            size="claw-sm"
            onClick={handleConfirm}
            data-testid="confirm-agent-type-switch"
          >
            确认切换
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
