import { useState } from "react";

import { GuidePointBubble } from "@/components/onboarding/GuidePointBubble";
import { cn } from "@/lib/utils";

const SKILL_MANAGEMENT_UPDATE_ID = "update-awareness-skill-management-2026-07-14";
const SKILL_MANAGEMENT_NOTICE_KEY = "tenant.agentDetail.skillManagementUpdateNotice.dismissed.2026-07-14";

const isNoticeDismissed = () => {
  if (typeof window === "undefined") return false;

  try {
    return window.localStorage.getItem(SKILL_MANAGEMENT_NOTICE_KEY) === "true";
  } catch {
    return false;
  }
};

export function SkillManagementUpdateNotice({ className }: { className?: string }) {
  const [visible, setVisible] = useState(() => !isNoticeDismissed());

  const dismiss = () => {
    setVisible(false);

    if (typeof window === "undefined") return;

    try {
      window.localStorage.setItem(SKILL_MANAGEMENT_NOTICE_KEY, "true");
    } catch {
      // localStorage 不可用时仅关闭当前页面提示
    }
  };

  if (!visible) return null;

  return (
    <div
      className={cn("pointer-events-none absolute left-1/2 bottom-[calc(100%+4px)] -translate-x-1/2", className)}
      data-update-awareness-id={SKILL_MANAGEMENT_UPDATE_ID}
      data-update-anchor="skill-section-title"
    >
      <div className="pointer-events-auto">
        <GuidePointBubble
          open
          onClose={dismiss}
          title="技能管理已更新"
          description="可在已安装技能右侧更新新版，或点叉卸载不再使用的技能。"
          contentVariant="text-button"
          actionText="知道了"
          onAction={dismiss}
          placement="top"
          showHotspot
          showSteps={false}
          endpoint="tenant"
        />
      </div>
    </div>
  );
}
