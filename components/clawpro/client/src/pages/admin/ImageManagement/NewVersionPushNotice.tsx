/**
 * NewVersionPushNotice - 新版本提醒横幅按钮
 *
 * 在 Agent 类型页顶部展示，点击即打开「全部更新记录」侧边栏。
 * 显示条件：存在开启了「有新版本时提醒用户更新」开关的 Agent 类型。
 */
import { Bell, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { PushableAgentType } from "./PushUpgradeDialog";

interface NewVersionPushNoticeProps {
  /** 可提醒的 Agent 类型列表（由父组件计算传入） */
  pushable: PushableAgentType[];
  /** 「查看全部更新记录」入口 */
  onViewAllRecords?: () => void;
}

export default function NewVersionPushNotice({
  pushable,
  onViewAllRecords,
}: NewVersionPushNoticeProps) {
  if (pushable.length === 0) return null;

  return (
    <Button
      type="button"
      variant="claw-outline"
      size="claw"
      onClick={onViewAllRecords}
      className="shrink-0 gap-2 border-[var(--border)] text-[var(--text-brand-deep)] hover:text-[var(--text-brand-deep)] [&_svg]:text-[var(--text-brand-deep)]"
    >
      <Bell className="size-3.5" />
      <span>提醒用户更新</span>
      <ChevronRight className="size-3.5" />
    </Button>
  );
}
