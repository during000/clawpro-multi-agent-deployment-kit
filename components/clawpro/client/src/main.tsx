import { createRoot } from "react-dom/client";
import App from "./App";
import { ServiceStatusProvider } from "./contexts/ServiceStatusContext";
import { PopupQueueProvider } from "./contexts/PopupQueueContext";
import BillingStatusToggle from "./components/BillingStatusToggle";
import ClsStatusToggle from "./components/ClsStatusToggle";
import "./index.css";

// —— Mock 会话重置：每次页面刷新/首次加载时，仅把「默认安全组快照」重置为
//    "未配置"空态，用于演示"从零配置安全组"的完整体验流程。
//    开关类状态（面板访问 / 云桌面）不在这里清除：用户即使中途删除了
//    默认安全组，已开启的开关也保持原状态，只在用户下一次主动切换开关时
//    再次做安全组校验 / 自动补规则（在 PlatformPolicy.tsx 中处理）。
if (typeof window !== "undefined") {
  try {
    localStorage.removeItem("admin_default_security_group_snapshot");
    localStorage.removeItem("openclaw_list");
    const distributionRecordsKey = "skillhub_distribution_records";
    const rawDistributionRecords = localStorage.getItem(distributionRecordsKey);
    if (rawDistributionRecords) {
      const nextDistributionRecords = JSON.parse(rawDistributionRecords).filter(
        (record: { skillId?: string }) => {
          const skillId = record?.skillId ?? "";
          return (
            !skillId.startsWith("public-skill:") &&
            !skillId.startsWith("public-package:")
          );
        }
      );
      localStorage.setItem(
        distributionRecordsKey,
        JSON.stringify(nextDistributionRecords)
      );
    }
  } catch {
    // 静默忽略（无痕模式、隐私限制等场景）
  }
}

createRoot(document.getElementById("root")!).render(
  <ServiceStatusProvider>
    <PopupQueueProvider>
      <App />
      <BillingStatusToggle />
      <ClsStatusToggle />
    </PopupQueueProvider>
  </ServiceStatusProvider>
);
