/**
 * AgentCommandsPage - 「执行命令」独立页面
 *
 * 路由: /admin/agent-commands
 * 内含两个 Tab：
 *   - list    命令模板  ← 现 CommandTaskTab（命令模板沉淀，含创建/编辑/下发/删除）
 *   - scheduled 定时任务 ← 管理尚未沉淀为执行结果的定时任务
 *   - history 执行记录  ← HistoryTab，scope 锁定为 command-execute
 *
 * URL 同步：?tab=list | scheduled | history
 *
 * 注：此页不挂在侧边栏，仅作为「Agent 列表 → 命令下发」按钮的延展，
 *     以及 toast「查看执行记录」跳转目标。
 */
import { useEffect, useState } from "react";
import { useSearch, useLocation } from "wouter";
import CommandTaskTab from "../VersionManagement/CommandTaskTab";
import HistoryTab from "../VersionManagement/HistoryTab";
import ScheduledTaskManagementTab from "../VersionManagement/ScheduledTaskManagementTab";
import AgentOpsPageHeader from "./AgentOpsPageHeader";
import PageTabs from "./PageTabs";

type TabId = "list" | "scheduled" | "history";

const TABS = [
  {
    id: "list" as const,
    label: "命令模板",
    description: "沉淀团队的运维命令模板，便于复用与审计；点击「下发」可批量到 Agent 实例执行。",
  },
  {
    id: "history" as const,
    label: "执行记录",
    description: "仅展示已完成执行的命令结果，支持查看每台 Agent 的退出码、输出和耗时。",
  },
  {
    id: "scheduled" as const,
    label: "定时任务",
    description: "集中管理定时命令任务，支持暂停、恢复和取消任务。",
  },
];

export default function AgentCommandsPage() {
  const [activeTab, setActiveTab] = useState<TabId>("list");
  const [, navigate] = useLocation();
  const searchStr = useSearch();

  useEffect(() => {
    const t = new URLSearchParams(searchStr).get("tab") as TabId | null;
    if (t === "list" || t === "scheduled" || t === "history") {
      setActiveTab(t);
    } else if (!t) {
      setActiveTab("list");
    }
  }, [searchStr]);

  const handleTabChange = (id: TabId) => {
    setActiveTab(id);
    const params = new URLSearchParams(searchStr);
    if (id === "list") params.delete("tab");
    else params.set("tab", id);
    const qs = params.toString();
    navigate(qs ? `/admin/agent-commands?${qs}` : "/admin/agent-commands", { replace: true });
  };

  const current = TABS.find((t) => t.id === activeTab)!;

  return (
    <div className="page-enter">
      <AgentOpsPageHeader
        title="执行命令"
        description="在 Agent 实例上批量执行 Shell 命令，支持命令模板复用与执行审计。"
      />

      <PageTabs<TabId>
        tabs={TABS}
        active={activeTab}
        onChange={handleTabChange}
        description={current.description}
      />

      {activeTab === "list" && <CommandTaskTab />}
      {activeTab === "history" && <HistoryTab scope="command-execute" />}
      {activeTab === "scheduled" && <ScheduledTaskManagementTab />}
    </div>
  );
}
