/**
 * AgentToolLibrary - 管控端 Agent 工具库页面
 * Design: 「流动蓝图」Fluid Blueprint
 * 五个 Tab：公共技能库、企业技能库、企业插件库、企业MCP库、企业规范库
 * 将原 SkillConfig 中的公共技能库和企业技能库迁移至此，并新增企业插件库、企业MCP库和企业规范库
 */
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { useLocation, useSearch } from "wouter";
import { ShieldCheck, ExternalLink, Check, ArrowRight, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { StatusTag } from "@/components/ui/status-tag";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog";
import EnterpriseSkillLibrary from "./EnterpriseSkillLibrary";
import PublicSkillLibraryTab from "./SkillLibrary/PublicSkillLibraryTab";
import PluginListTab from "./SkillLibrary/PluginListTab";
import MCPListTab from "./SkillLibrary/MCPListTab";
import StandardsLibraryTab from "./SkillLibrary/StandardsLibraryTab";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { LineTabs } from "@/components/ui/line-tabs";
import { BodyMedium, BodyText, MetaText, MetaMedium } from "@/components/ui/Typography";
import { useAdminMode } from "@/contexts/AdminModeContext";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from "./MemberManagement/mock";

const TABS = [
  {
    id: "public",
    label: "公共技能库",
    description: "浏览公共技能市场，收藏技能并加入初始技能包，形成适合企业实际场景的公共技能库。",
    // 稳定选择器：用于在停服态下精准定位该 Tab 入口并附加 data-billing-exempt 豁免
    dataGuide: "agent-tool-library-public-tab",
  },
  {
    id: "enterprise",
    label: "企业技能库",
    description: "Skill 一键入库、批量分发，打造安全稳定的企业级技能管理体系。",
    // 稳定选择器：用于在停服态下精准定位该 Tab 入口并附加 data-billing-exempt 豁免
    dataGuide: "agent-tool-library-enterprise-tab",
  },
  {
    id: "plugins",
    label: "企业插件库",
    description: "上传和管理企业自定义插件，按需下发到 Agent 云服务器，扩展 Agent 能力边界。",
  },
  {
    id: "mcp",
    label: "企业MCP库",
    description: "统一管理 MCP 服务配置，支持远程服务和本地命令两种连接方式，按需下发到智能体实例。",
  },
  {
    id: "standards",
    label: "企业规范库",
    description: "统一管理 Markdown 企业规范与 Hook 配置，按用户或用户组下发到 Agent。",
  },
  {
    id: "settings",
    label: "企业全局设定",
    description: "统一管理各项目的团队全局设定文档，作为 Agent 每次会话自动加载的团队基础上下文。",
  },
] as const;

type AgentToolLibraryTabId = (typeof TABS)[number]["id"];

const isAgentToolLibraryTabId = (value: string | null): value is AgentToolLibraryTabId =>
  TABS.some((tab) => tab.id === value);

export default function AgentToolLibrary() {
  const tabsContainerRef = useRef<HTMLDivElement>(null);
  const { hasOneid } = useAdminMode();
  const [, navigate] = useLocation();
  const searchStr = useSearch();
  const [activeTab, setActiveTab] = useState<AgentToolLibraryTabId>("public");
  const [packages, setPackages] = useState<Array<{ id: string; name: string; isActive: boolean; scopeType: 'all-users' | 'groups'; scopeLabel: string; groupIds?: string[] }>>([
    { id: 'pkg-1', name: '全员通用技能包', isActive: true, scopeType: 'all-users', scopeLabel: '全部组织' },
    { id: 'pkg-2', name: '高级开发技能包', isActive: false, scopeType: 'groups', scopeLabel: '指定用户组织' },
  ]);
  const agentGroups = useMemo(
    () => (hasOneid ? MOCK_ONEID_GROUPS : MOCK_MANUAL_GROUPS).map((group) => ({
      id: group.id,
      name: group.name,
      parentId: group.parentId,
    })),
    [hasOneid],
  );
  const scopedPackages = useMemo(
    () => packages.map((pkg) => {
      if (pkg.scopeType !== 'groups') return pkg;
      const groupIds = hasOneid ? ['dept-fe', 'dept-be'] : ['mgrp-rd-fe', 'mgrp-rd-be'];
      const scopeLabel = groupIds
        .map((groupId) => agentGroups.find((group) => group.id === groupId)?.name ?? groupId)
        .join('、');
      return { ...pkg, groupIds, scopeLabel };
    }),
    [agentGroups, hasOneid, packages],
  );
  const [packagesDraft, setPackagesDraft] = useState<Record<string, boolean>>({});

  // 安全检测服务状态（与 SkillListTab 共享 localStorage）
  const [securityServiceActive, setSecurityServiceActive] = useState<boolean>(() => {
    const saved = localStorage.getItem('skill_security_service_active');
    return saved === 'true';
  });
  const [securityApplyDialogOpen, setSecurityApplyDialogOpen] = useState(false);
  const [securitySuccessDialogOpen, setSecuritySuccessDialogOpen] = useState(false);
  const [securityServiceUsed] = useState(156); // mock 已用额度

  useEffect(() => {
    const tab = new URLSearchParams(searchStr).get("tab");
    setActiveTab(isAgentToolLibraryTabId(tab) ? tab : "public");
  }, [searchStr]);

  /*
   * 停服时仍允许「公共技能库」/「企业技能库」Tab 入口可点击：纯导航/查看类操作。
   * 在 LineTabs 外层加 data-billing-exempt 容器，
   * AdminDisabledOverlay CSS & 点击拦截对该容器内所有子元素失效。
   */
  // (no polling needed — the container-level exemption covers all children)

  const handleTabChange = (tabId: AgentToolLibraryTabId) => {
    setActiveTab(tabId);
    const params = new URLSearchParams(searchStr);
    if (tabId === "public") {
      params.delete("tab");
    } else {
      params.set("tab", tabId);
    }
    const queryString = params.toString();
    navigate(queryString ? `/admin/agent-tool-library?${queryString}` : "/admin/agent-tool-library", { replace: true });
  };

  const currentTab = TABS.find((t) => t.id === activeTab)!;

  return (
    <div className="page-enter w-full min-w-0">
      <AdminPageHeader title="Agent 工具库" />

      {/* Tab 切换器（统一 LineTabs，规范见 SKILL §11.5） */}
      <div ref={tabsContainerRef} data-billing-exempt>
        <LineTabs
          tabs={TABS}
          active={activeTab}
          onChange={handleTabChange}
        />
      </div>

      {/* Tab 描述（保留富内容：描述 + 安全检测 Alert，需自定义渲染） */}
      <div className="mt-3 mb-6 space-y-2">
        <BodyText as="p" tone="muted" className="leading-relaxed">{currentTab.description}</BodyText>
        {currentTab.id === 'enterprise' && (
          <Alert variant="default" className="items-center [&>svg]:translate-y-0">
            <AlertInfoIcon />
            <AlertDescription className="flex items-center justify-between gap-3 flex-wrap w-full">
              <span>由腾讯云 Agent Storage 提供服务，独享 50GB 免费空间</span>
              <div className="flex items-center gap-2">
              {!securityServiceActive ? (
                <Badge variant="outline" className="cursor-pointer gap-1.5 pr-1.5" onClick={() => setSecurityApplyDialogOpen(true)}>
                  <MetaText as="span" tone="body">恶意 Skills 扫描 API：未开通</MetaText>
                  <MetaMedium as="span" className="inline-flex items-center gap-0.5 !text-[var(--text-brand-deep)] hover:!text-[var(--text-brand-deeper,var(--text-brand-deep))]">
                    一键开通
                    <ArrowRight className="w-3 h-3" />
                  </MetaMedium>
                </Badge>
              ) : (
                <HoverCard openDelay={300}>
                <HoverCardTrigger asChild>
                  <span className="inline-flex">
                    <Badge color="green" className="cursor-pointer">
                      <MetaText as="span" tone="body">恶意 Skills 扫描 API：试用中</MetaText>
                    </Badge>
                  </span>
                </HoverCardTrigger>
                <HoverCardContent side="bottom" align="end" className="w-80 p-4">
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <BodyMedium as="span" tone="primary" className="flex items-center gap-1.5">
                          <ShieldCheck className="w-4 h-4 text-[var(--text-success)]" />
                          恶意 Skills 扫描 API
                        </BodyMedium>
                        <StatusTag variant="green" mode="soft">试用中</StatusTag>
                      </div>
                      <div className="space-y-2">
                        <div className="flex items-start gap-2">
                          <MetaText tone="muted" className="shrink-0 w-16">试用有效期</MetaText>
                          <MetaText>有效期至 2026年6月30日</MetaText>
                        </div>
                        <div className="flex items-start gap-2">
                          <MetaText tone="muted" className="shrink-0 w-16">已用额度</MetaText>
                          <MetaText>{securityServiceUsed}/1000次<MetaText tone="weak">（有效期到期后，剩余未使用的调用额度将清空）</MetaText></MetaText>
                        </div>
                        {/* 进度条 */}
                        <div className="w-full h-1.5 bg-[var(--bg-grey-hover)] rounded-full overflow-hidden">
                          <div
                            className="h-full rounded-full bg-[var(--cp-brand-blue)] transition-all"
                            style={{ width: `${(securityServiceUsed / 1000) * 100}%` }}
                          />
                        </div>
                      </div>
                      <div className="flex justify-end pt-1">
                        <a
                          href="https://cloud.tencent.com/document/api/664/131590"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-[var(--text-brand)] hover:text-[var(--text-brand)] flex items-center gap-1"
                        >
                          说明文档
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      </div>
                    </div>
                  </HoverCardContent>
                </HoverCard>
              )}
              {/* 调试用：状态切换按钮 */}
              <Button
                variant="claw-outline"
                size="icon-sm"
                onClick={() => {
                  const next = !securityServiceActive;
                  setSecurityServiceActive(next);
                  localStorage.setItem('skill_security_service_active', String(next));
                  toast.success(next ? '已模拟开通安全检测服务' : '已模拟取消安全检测服务');
                }}
                className="ml-1.5 size-7"
                title="切换开通状态（调试）"
              >
                <RefreshCw className="w-3 h-3" />
              </Button>
            </div>
            </AlertDescription>
          </Alert>
        )}
      </div>

      {/* 安全检测服务 — 申请开通弹窗 */}
      <Dialog open={securityApplyDialogOpen} onOpenChange={setSecurityApplyDialogOpen}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>申请免费试用（Skills 风险检测 API）</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-[5rem_1fr] gap-y-3">
              <BodyText as="span" tone="muted">试用有效期</BodyText>
              <BodyText as="span" tone="primary">有效期至 2026年6月30日</BodyText>
              <BodyText as="span" tone="muted">调用额度</BodyText>
              <BodyText as="span" tone="primary">1000次<MetaText tone="weak" className="ml-1">（有效期到期后，剩余未使用的调用额度将清空）</MetaText></BodyText>
              <BodyText as="span" tone="muted">操作指引</BodyText>
              <a
                href="https://cloud.tencent.com/document/api/664/131590"
                target="_blank"
                rel="noopener noreferrer"
                className="text-[var(--text-brand)] hover:opacity-80 flex items-center gap-1"
              >
                说明文档
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => setSecurityApplyDialogOpen(false)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                setSecurityServiceActive(true);
                localStorage.setItem('skill_security_service_active', 'true');
                setSecurityApplyDialogOpen(false);
                setSecuritySuccessDialogOpen(true);
              }}
            >
              立即领取
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 安全检测服务 — 开通成功弹窗 */}
      <Dialog open={securitySuccessDialogOpen} onOpenChange={setSecuritySuccessDialogOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>
              试用额度已开通
            </DialogTitle>
            <DialogDescription className="pt-2">
              1000次调用额度，有效期至 2026-06-30
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div>
              <BodyMedium as="p" tone="primary" className="mb-1">使用 API</BodyMedium>
              <BodyText as="p" tone="muted">
                您可以前往查看{' '}
                <a
                  href="https://cloud.tencent.com/document/api/664/131590"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[var(--text-brand)] hover:opacity-80 inline-flex items-center gap-0.5"
                >
                  说明文档
                  <ExternalLink className="w-3 h-3" />
                </a>
                ，基于说明文档调用并测试 API。
              </BodyText>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="dialog-confirm"
              onClick={() => setSecuritySuccessDialogOpen(false)}
            >
              我知道了
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Tab 内容 */}
      {activeTab === "public" && (
        <PublicSkillLibraryTab
          packages={scopedPackages}
          groups={agentGroups}
          onAddSkillToPackage={(skillId, packageId) => {
            setPackagesDraft(prev => ({ ...prev, [packageId]: true }));
          }}
        />
      )}

      {activeTab === "enterprise" && (
          <EnterpriseSkillLibrary securityServiceActive={securityServiceActive} />
      )}

      {activeTab === "plugins" && (
        <PluginListTab />
      )}

      {activeTab === "mcp" && (
        <MCPListTab />
      )}

      {activeTab === "standards" && (
        <StandardsLibraryTab />
      )}
    </div>
  );
}
