/**
 * CloudDevEnvDetailDialog - 云开发环境详情弹窗（共享组件）
 * 供 CloudDevManagement（管控端）和 AgentChat（用户端）共同使用
 *
 * 两端差异（通过 `mode` prop 控制）：
 *   | 能力                          | tenant（用户端） | admin（管控端） |
 *   |-------------------------------|------------------|-----------------|
 *   | 前往控制台                    | 不显示           | 显示            |
 *   | 列表项「查看详情」按钮         | 不显示           | 显示            |
 *   | 环境切换器下拉                | 仅用户关联环境   | 全量环境        |
 *   | 已删除环境提示横幅            | 显示             | 不会遇到        |
 *   | 绑定 / 换绑                   | 显示按钮         | 不显示          |
 *
 * 设计参考：腾讯云 lightclaw.cloud.tencent.com 云开发管理弹窗
 * 视觉规范：
 *   - 左侧 220px 白色侧边栏：品牌区 + 环境切换器 (+ tenant 端绑定按钮) + 导航
 *   - 列表项 4px 圆角 + var(--cp-border) 描边（admin 铁律）
 *   - StatusTag mode="fill"，Button variant="claw-outline" / "link"
 *   - 颜色统一走 --cp-* / --text-* token
 */
import { useState } from "react";
import { Dialog, DialogContent, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { StatusTag } from "@/components/ui/status-tag";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Database, Globe, ChevronDown,
  Home, LayoutGrid, FileText, FolderOpen,
  ExternalLink, Copy, Code2, AlertCircle, X, Info, Plus, RefreshCw,
  Files, HardDrive, ShieldCheck, Zap, Rocket,
  Cloud,
} from "lucide-react";

/* ── 共享类型 ─────────────────────────────────────────────── */
export type EnvStatus = "running" | "stopped" | "creating" | "error";

export interface Env {
  id: string;
  envId: string;
  name: string;
  status: EnvStatus;
  region: string;
  packageName: string;
  storageUsed: string;
  dbUsed: string;
  functionCount: number;
  staticHosting: boolean;
  createdAt: string;
  expireAt: string;
  lastDeployAt: string;
  appCount: number;
  appNames: string[];
  createdBy: string;
  allowedGroups: string[];
  allowedUsers: string[];
  dbType: "cloud" | "postgresql";
  overflowBilling: boolean;
  autoRenewal: boolean;
}

/* ── 共享常量 ─────────────────────────────────────────────── */
export const RM: Record<string, string> = { "ap-guangzhou": "广州", "ap-beijing": "北京", "ap-shanghai": "上海" };

/** 状态 → StatusTag variant 映射（对齐 OpenClawMonitor STATUS_CONFIG） */
export const SC: Record<EnvStatus, { label: string; tagVariant: "green" | "blue" | "red" | "gray" }> = {
  running:  { label: "运行中", tagVariant: "green" },
  stopped:  { label: "已停用", tagVariant: "gray" },
  creating: { label: "创建中", tagVariant: "blue" },
  error:    { label: "异常",   tagVariant: "red" },
};

/* ── 组件 Props ───────────────────────────────────────────── */
interface CloudDevEnvDetailDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 当前查看的环境；为 null 时弹窗不渲染 */
  env: Env | null;
  /**
   * 端别（默认 tenant，向后兼容 AgentChat 原调用）：
   *   - tenant：用户端，隐藏「前往控制台」「列表项查看详情」，提供「绑定/换绑」「已删除提示」「非绑定环境提示横幅」
   *   - admin： 管控端，显示「前往控制台」「列表项查看详情」，无绑定能力
   */
  mode?: "tenant" | "admin";
  /**
   * 切换器下拉的可选环境列表
   *   - tenant：仅传当前用户已关联的环境
   *   - admin： 传全量环境
   *   传 undefined 或长度 ≤ 1 时切换器为单环境只读态（不可下拉）
   */
  envOptions?: Env[];
  /** 切换器选中环境的回调 */
  onSelectEnv?: (env: Env) => void;
  /**
   * tenant 端：当前 Agent 已绑定的环境 id。
   * 用于区分"已绑定 / 查看中 / 未绑定"，并控制顶部「非当前绑定环境」横幅与切换器右侧角标。
   */
  boundEnvId?: string;
  /**
   * tenant 端：当前环境是否「已删除（Agent 仍引用）」。
   * true 时顶部显示红色警告横幅 + 引导换绑
   */
  isDeleted?: boolean;
  /** tenant 端：点击「换绑到当前查看环境」回调（参考站点用语） */
  onRebind?: () => void;
  /** tenant 端：点击「切回绑定环境」回调（参考站点用语） */
  onSwitchBackToBound?: () => void;
  /** tenant 端：切换器下拉顶部「+ 新建环境」回调（admin 端不显示） */
  onCreateEnv?: () => void;
  /** tenant 端：切换器下拉顶部「刷新」回调（admin 端不显示） */
  onRefreshEnvs?: () => void;
}

type TabId = "basic" | "apps" | "database" | "functions" | "storage";

const TAB_TITLE: Record<TabId, string> = {
  basic: "基础信息",
  apps: "前端应用",
  database: "数据集合",
  functions: "云函数",
  storage: "文件存储",
};

export default function CloudDevEnvDetailDialog({
  open,
  onOpenChange,
  env,
  mode = "tenant",
  envOptions,
  onSelectEnv,
  boundEnvId,
  isDeleted = false,
  onRebind,
  onSwitchBackToBound,
  onCreateEnv,
  onRefreshEnvs,
}: CloudDevEnvDetailDialogProps) {
  const [tab, setTab] = useState<TabId>("basic");
  const [switcherOpen, setSwitcherOpen] = useState(false);

  // env=null 时，渲染"无可用环境"空态弹窗
  // 视觉骨架与正常态完全一致（960px / 左 aside + 右 section + 5 项 nav），
  // 右侧主区域用 EmptyState 占位，引导用户走"新建环境"路径
  if (!env) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-[960px] p-0 overflow-hidden" showCloseButton={false}>
          <div className="flex h-[70vh] min-h-[560px]">
            {/* ═══ 左侧导航栏（与正常态同骨架；切换器与 nav 均为只读 / 占位） ═══ */}
            <aside className="w-[220px] border-r border-[var(--cp-border)] bg-[var(--cp-surface)] flex flex-col flex-shrink-0">
              <div className="px-4 pt-5 pb-2.5">
                <DialogTitle className="text-sm font-semibold text-[var(--text-title)]">环境信息</DialogTitle>
                <DialogDescription className="sr-only">
                  当前 Agent 尚未绑定云开发环境，请新建环境后使用
                </DialogDescription>
              </div>

              {/* 切换器（与正常态外观一致）：
                  - trigger 显示 "请选择环境" 占位 + ChevronDown
                  - 下拉内容：顶部 "+ 新建环境" 按钮 + 列表区 "暂无可用环境" 文字占位
                  - 无 onRefreshEnvs 兜底处理；空态时不显示刷新（无意义） */}
              <div className="px-3 pb-3">
                <Popover open={switcherOpen} onOpenChange={setSwitcherOpen}>
                  <PopoverTrigger asChild>
                    <button
                      type="button"
                      className="w-full flex items-center gap-1.5 px-3 h-9 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-white hover:border-[var(--cp-border-control)] transition-colors"
                    >
                      <span className="text-sm text-[var(--text-weak)] truncate flex-1 text-left">请选择环境</span>
                      <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)] flex-shrink-0" />
                    </button>
                  </PopoverTrigger>
                  <PopoverContent align="start" className="w-[240px] p-0">
                    {/* 头部："+ 新建环境"（左）+ 刷新（右） */}
                    {(onCreateEnv || onRefreshEnvs) && (
                      <div className="flex items-center justify-between gap-2 px-3 py-2 border-b border-[var(--cp-border)]">
                        {onCreateEnv ? (
                          <button
                            type="button"
                            onClick={() => { onCreateEnv(); setSwitcherOpen(false); }}
                            className="inline-flex items-center gap-1 text-sm font-medium text-[var(--text-brand)] hover:opacity-80 transition-opacity"
                          >
                            <Plus className="w-3.5 h-3.5" />
                            新建环境
                          </button>
                        ) : <span />}
                        {onRefreshEnvs && (
                          <button
                            type="button"
                            onClick={() => onRefreshEnvs()}
                            className="w-6 h-6 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-title)] transition-colors"
                            aria-label="刷新环境列表"
                          >
                            <RefreshCw className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    )}

                    {/* 列表区：空态文字占位 */}
                    <div className="py-6 text-center text-sm text-[var(--text-weak)]">
                      暂无可用环境
                    </div>
                  </PopoverContent>
                </Popover>
              </div>

              {/* 导航项：全部禁用 */}
              <nav className="flex-1 px-3 pb-2 space-y-0.5 overflow-y-auto scrollbar-hide">
                {[
                  { id: "basic", icon: Home,        label: "基础信息" },
                  { id: "apps", icon: LayoutGrid,  label: "前端应用" },
                  { id: "database", icon: Database,    label: "数据库" },
                  { id: "functions", icon: Code2,       label: "云函数" },
                  { id: "storage", icon: FolderOpen,  label: "文件存储" },
                ].map((item) => (
                  <div
                    key={item.id}
                    className="w-full flex items-center gap-2.5 px-3 py-2 rounded-[var(--radius-lg)] text-sm text-[var(--text-weak)] cursor-not-allowed select-none"
                    aria-disabled="true"
                  >
                    <item.icon className="w-4 h-4 text-[var(--text-weak)]" />
                    {item.label}
                  </div>
                ))}
              </nav>
            </aside>

            {/* ═══ 右侧内容区 ═══════════════════════════════════════════════════ */}
            <section className="flex-1 flex flex-col min-w-0 bg-white">
              {/* 顶部标题 + 关闭按钮（沿用正常态结构） */}
              <header className="px-6 pt-5 pb-4 flex items-center justify-between flex-shrink-0">
                <h3 className="text-base font-semibold text-[var(--text-title)]">基础信息</h3>
                <button
                  type="button"
                  onClick={() => onOpenChange(false)}
                  className="w-7 h-7 rounded-[var(--radius-lg)] flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-normal)] transition-colors"
                  aria-label="关闭"
                >
                  <X className="w-4 h-4" />
                </button>
              </header>

              {/* 主内容区：EmptyState 居中（Cloud 图标 + 文案 + 主操作） */}
              <div className="flex-1 min-h-0 px-6 pb-6 flex items-center justify-center">
                <div className="flex flex-col items-center text-center max-w-[360px]">
                  <div
                    className="flex items-center justify-center mb-4"
                    style={{
                      width: 64,
                      height: 64,
                      borderRadius: "var(--radius-lg)",
                      background: "rgba(20, 71, 230, 0.08)",
                    }}
                  >
                    <Cloud className="w-8 h-8 text-[var(--cp-brand-blue)]" />
                  </div>
                  <p className="text-sm font-medium text-[var(--text-title)]">
                    当前 Agent 尚未绑定云开发环境
                  </p>
                  <p className="text-xs text-[var(--text-muted)] mt-1.5 leading-relaxed">
                    该角色专用于云开发场景，需要一个环境用于代码运行、部署与数据存储。请先新建环境后使用。
                  </p>
                  {onCreateEnv && (
                    <Button
                      variant="dialog-confirm"
                      size="sm"
                      onClick={onCreateEnv}
                      className="mt-5"
                    >
                      <Plus className="w-3.5 h-3.5 mr-1" />
                      新建环境
                    </Button>
                  )}
                </div>
              </div>
            </section>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  const isAdmin = mode === "admin";
  const isTenant = mode === "tenant";
  const hasMultipleEnvs = (envOptions?.length ?? 0) > 1;
  // tenant 端：当前查看的环境是否就是 Agent 已绑定的环境
  const isBoundEnv = isTenant && boundEnvId !== undefined && env.id === boundEnvId;
  // tenant 端：是否处于"非绑定环境的查看"状态（已绑定其它环境时，用于顶部横幅 + 切换器角标）
  const isViewingNonBound = isTenant && boundEnvId !== undefined && env.id !== boundEnvId && !isDeleted;
  // tenant 端：是否处于"完全未绑定"状态（Agent 还没绑定任何环境）
  // 显示与 isViewingNonBound 同样位置的引导横幅，但仅一个主操作"绑定到当前环境"
  const isUnbound = isTenant && boundEnvId === undefined && !isDeleted;

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { setTab("basic"); setSwitcherOpen(false); } onOpenChange(o); }}>
      <DialogContent className="sm:max-w-[960px] p-0 overflow-hidden" showCloseButton={false}>
        <div className="flex h-[70vh] min-h-[560px]">
          {/* ═══ 左侧导航栏 ═══════════════════════════════════════════════════ */}
          <aside className="w-[220px] border-r border-[var(--cp-border)] bg-[var(--cp-surface)] flex flex-col flex-shrink-0">
            {/* 标题区（DialogTitle，去图标） */}
            <div className="px-4 pt-5 pb-2.5">
              <DialogTitle className="text-sm font-semibold text-[var(--text-title)]">环境信息</DialogTitle>
              <DialogDescription className="sr-only">
                查看云开发环境的基础信息、前端应用、数据库、云函数与文件存储
              </DialogDescription>
            </div>

            {/* 环境切换器（多环境时下拉，单环境时只读）
                - tenant 端：右侧 badge 反映"已绑定 / 查看中 / 无标记"三态
                - 下拉头部："+ 新建环境"（左）+ 刷新（右）
                - 每个选项：状态圆点 + 名称 + 套餐 badge + 绑定/查看角标 */}
            <div className="px-3 pb-3">
              {hasMultipleEnvs ? (
                <Popover open={switcherOpen} onOpenChange={setSwitcherOpen}>
                  <PopoverTrigger asChild>
                    <button
                      type="button"
                      className="w-full flex items-center gap-1.5 px-3 h-9 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-white hover:border-[var(--cp-border-control)] transition-colors"
                    >
                      <span className="text-sm text-[var(--text-title)] truncate flex-1 text-left">{env.name}</span>
                      {isTenant && isBoundEnv && <StatusTag mode="fill" variant="blue">已绑定</StatusTag>}
                      {isTenant && isViewingNonBound && <StatusTag mode="fill" variant="orange">查看中</StatusTag>}
                      <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)] flex-shrink-0" />
                    </button>
                  </PopoverTrigger>
                  <PopoverContent align="start" className="w-[240px] p-0">
                    {/* tenant 端：头部 "+ 新建环境" + 刷新（admin 端不显示头部） */}
                    {isTenant && (onCreateEnv || onRefreshEnvs) && (
                      <div className="flex items-center justify-between gap-2 px-3 py-2 border-b border-[var(--cp-border)]">
                        {onCreateEnv ? (
                          <button
                            type="button"
                            onClick={() => { onCreateEnv(); setSwitcherOpen(false); }}
                            className="inline-flex items-center gap-1 text-sm font-medium text-[var(--text-brand)] hover:opacity-80 transition-opacity"
                          >
                            <Plus className="w-3.5 h-3.5" />
                            新建环境
                          </button>
                        ) : <span />}
                        {onRefreshEnvs && (
                          <button
                            type="button"
                            onClick={() => onRefreshEnvs()}
                            className="w-6 h-6 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-title)] transition-colors"
                            aria-label="刷新环境列表"
                          >
                            <RefreshCw className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    )}

                    <div className="max-h-72 overflow-y-auto scrollbar-hide py-1">
                      {(envOptions ?? []).map((opt) => {
                        const viewing = opt.id === env.id;
                        const bound = isTenant && opt.id === boundEnvId;
                        const isCreating = opt.status === "creating";
                        const statusDot =
                          opt.status === "running" ? "bg-[var(--text-success)]"
                          : opt.status === "creating" ? "bg-[var(--text-warning)]"
                          : opt.status === "error" ? "bg-[var(--text-danger)]"
                          : "bg-[var(--text-weak)]";
                        return (
                          <button
                            key={opt.id}
                            type="button"
                            onClick={() => {
                              onSelectEnv?.(opt);
                              setSwitcherOpen(false);
                            }}
                            className={`w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors text-left ${
                              viewing
                                ? "bg-[var(--bg-grey-normal)]"
                                : "hover:bg-[var(--bg-grey-normal)]"
                            }`}
                          >
                            <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${statusDot}`} />
                            <span className={`flex-1 truncate ${isCreating ? "text-[var(--text-weak)]" : "text-[var(--text-title)]"}`}>{opt.name}</span>
                            {/* 套餐 badge */}
                            {!isCreating && <StatusTag mode="fill" variant="blue">{opt.packageName}</StatusTag>}
                            {isCreating && <StatusTag mode="fill" variant="orange">初始化中</StatusTag>}
                            {/* tenant：当前查看 / 已绑定 角标 */}
                            {isTenant && bound && <StatusTag mode="fill" variant="blue">已绑定</StatusTag>}
                            {isTenant && viewing && !bound && <StatusTag mode="fill" variant="orange">查看中</StatusTag>}
                          </button>
                        );
                      })}
                    </div>
                  </PopoverContent>
                </Popover>
              ) : (
                <div className="w-full flex items-center gap-1.5 px-3 h-9 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-white">
                  <span className="text-sm text-[var(--text-title)] truncate flex-1">{env.name}</span>
                  {isTenant && isBoundEnv && <StatusTag mode="fill" variant="blue">已绑定</StatusTag>}
                </div>
              )}
            </div>

            {/* 导航 */}
            <nav className="flex-1 px-3 pb-2 space-y-0.5 overflow-y-auto scrollbar-hide">
              {[
                { id: "basic" as const,     icon: Home,        label: "基础信息" },
                { id: "apps" as const,      icon: LayoutGrid,  label: "前端应用" },
                { id: "database" as const,  icon: Database,    label: "数据库" },
                { id: "functions" as const, icon: Code2,       label: "云函数" },
                { id: "storage" as const,   icon: FolderOpen,  label: "文件存储" },
              ].map(item => {
                const active = tab === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setTab(item.id)}
                    className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-[var(--radius-lg)] text-sm transition-colors ${
                      active
                        ? "bg-[var(--cp-brand-blue)]/10 text-[var(--text-brand)] font-medium"
                        : "text-[var(--text-secondary)] hover:bg-[var(--bg-grey-normal)] hover:text-[var(--text-title)]"
                    }`}
                  >
                    <item.icon className={`w-4 h-4 ${active ? "text-[var(--text-brand)]" : "text-[var(--text-secondary)]"}`} />
                    {item.label}
                  </button>
                );
              })}
            </nav>
          </aside>

          {/* ═══ 右侧内容区 ═══════════════════════════════════════════════════ */}
          <section className="flex-1 flex flex-col min-w-0 bg-white">
            {/* 顶部标题 + 关闭按钮 */}
            <header className="px-6 pt-5 pb-4 flex items-center justify-between flex-shrink-0">
              <h3 className="text-base font-semibold text-[var(--text-title)]">
                {TAB_TITLE[tab]}
                {tab === "apps"      && <span className="ml-1 text-[var(--text-weak)] font-normal">({env.appCount})</span>}
                {tab === "database"  && <span className="ml-1 text-[var(--text-weak)] font-normal">(8)</span>}
                {tab === "functions" && <span className="ml-1 text-[var(--text-weak)] font-normal">({env.functionCount})</span>}
              </h3>
              <button
                type="button"
                onClick={() => onOpenChange(false)}
                className="w-5 h-5 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-title)] transition-colors"
                aria-label="关闭"
              >
                <X className="w-4 h-4" />
              </button>
            </header>

            <div className="flex-1 overflow-y-auto scrollbar-hide px-6 pb-6">
              {/* 仅 tenant 端：环境已删除提示横幅（红色，最强警告） */}
              {isTenant && isDeleted && (
                <div className="mb-5 flex items-start gap-3 rounded-[var(--radius-lg)] border border-[var(--text-danger)]/30 bg-[var(--text-danger)]/5 px-4 py-3">
                  <AlertCircle className="w-4 h-4 text-[var(--text-danger)] mt-0.5 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-[var(--text-title)]">该环境已被删除</p>
                    <p className="text-xs text-[var(--text-muted)] mt-0.5 leading-relaxed">
                      Agent 仍引用该环境，但环境已不可用。请换绑到其他可用环境，否则相关云开发能力将无法正常使用。
                    </p>
                  </div>
                  {onRebind && (
                    <Button variant="claw-outline" size="sm" onClick={onRebind} className="flex-shrink-0">换绑环境</Button>
                  )}
                </div>
              )}

              {/* 仅 tenant 端：非绑定环境信息横幅（中性灰，参考腾讯云原型） */}
              {isViewingNonBound && (
                <div className="mb-5 flex items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-3">
                  <Info className="w-4 h-4 text-[var(--text-muted)] flex-shrink-0" />
                  <p className="flex-1 min-w-0 text-sm text-[var(--text-secondary)]">
                    正在查看「<span className="font-medium text-[var(--text-title)]">{env.envId}</span>」，非当前绑定环境
                  </p>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {onSwitchBackToBound && (
                      <Button variant="claw-outline" size="sm" onClick={onSwitchBackToBound}>切回绑定环境</Button>
                    )}
                    {onRebind && (
                      <Button variant="dialog-confirm" size="sm" onClick={onRebind}>换绑到当前环境</Button>
                    )}
                  </div>
                </div>
              )}

              {/* 仅 tenant 端：完全未绑定（首次进入）的引导横幅 ——
                  视觉上比"非绑定环境查看"更突出（浅蓝品牌底 + 品牌色描边），与 Header"绑定云开发环境"按钮视觉呼应；
                  文案锚定当前查看环境，主操作"绑定到当前环境" */}
              {isUnbound && (
                <div className="mb-5 flex items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--cp-brand-blue)]/30 bg-[var(--cp-brand-blue)]/5 px-4 py-3">
                  <Info className="w-4 h-4 text-[var(--cp-brand-blue)] flex-shrink-0" />
                  <p className="flex-1 min-w-0 text-sm text-[var(--text-secondary)]">
                    正在查看「<span className="font-medium text-[var(--text-title)]">{env.envId}</span>」，尚未绑定到当前 Agent
                  </p>
                  {onRebind && (
                    <Button variant="dialog-confirm" size="sm" onClick={onRebind} className="flex-shrink-0">
                      绑定到当前环境
                    </Button>
                  )}
                </div>
              )}

              {/* ─── 基础信息 ─── */}
              {tab === "basic" && (
                <div className="space-y-5">
                  {/* 头部摘要（在卡片内）*/}
                  <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] p-4 space-y-3">
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="text-sm font-semibold text-[var(--text-title)] truncate">{env.name}</span>
                        <StatusTag mode="fill" variant="blue">{env.packageName}</StatusTag>
                        <StatusTag mode="fill" variant={SC[env.status].tagVariant}>{SC[env.status].label}</StatusTag>
                      </div>
                      {/* 仅 admin 端：前往控制台 */}
                      {isAdmin && (
                        <Button variant="claw-outline" size="sm" className="flex-shrink-0">
                          前往控制台
                          <ExternalLink className="w-3 h-3 ml-1" />
                        </Button>
                      )}
                    </div>

                    <div className="grid grid-cols-2 gap-x-6 gap-y-2.5">
                      <FieldRow label="地域" value={RM[env.region] || env.region} />
                      <FieldRow label="创建时间" value={env.createdAt} />
                      <FieldRow label="环境 ID">
                        <div className="flex items-center gap-1.5">
                          <span className="text-xs font-mono text-[var(--text-title)]">{env.envId}</span>
                          <button
                            type="button"
                            onClick={() => handleCopy(env.envId)}
                            className="text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
                            aria-label="复制环境 ID"
                          >
                            <Copy className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </FieldRow>
                      <FieldRow label="付费模式" value={env.expireAt === "-" ? "按量付费" : "包年包月"} />
                    </div>
                  </div>

                  {/* 统计卡片（与文件存储 tab 同款：单容器 + 4 列横排，lucide 图标 + 文字样式对齐） */}
                  <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] p-4">
                    <div className="grid grid-cols-4 gap-4">
                      {[
                        { icon: Rocket,   label: "应用",   value: String(env.appCount) },
                        { icon: Code2,    label: "云函数", value: String(env.functionCount) },
                        { icon: FileText, label: "文件",   value: env.staticHosting ? "1" : "0" },
                        { icon: Database, label: "集合",   value: "8" },
                      ].map(card => {
                        const Icon = card.icon;
                        return (
                          <div key={card.label} className="flex items-center gap-2.5 min-w-0">
                            <ListRowIcon><Icon className="w-4 h-4 text-[var(--text-muted)]" /></ListRowIcon>
                            <div className="min-w-0">
                              <p className="text-xs text-[var(--text-muted)] mb-0.5">{card.label}</p>
                              <p className="text-sm font-semibold tabular-nums truncate text-[var(--text-title)]">{card.value}</p>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                </div>
              )}

              {/* ─── 前端应用 ─── */}
              {tab === "apps" && (
                <div className="space-y-2.5">
                  {env.appNames.length > 0 ? env.appNames.map((name, i) => (
                    <ListRow
                      key={i}
                      icon={<ListRowIcon><Globe className="w-4 h-4 text-[var(--text-muted)]" /></ListRowIcon>}
                      title={
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-[var(--text-title)]">{name}</span>
                          <span className="px-1.5 py-0.5 text-xs text-[var(--text-muted)] bg-[var(--bg-grey-normal)] rounded-[var(--radius-lg)]">{["react", "vue", "next"][i % 3]}</span>
                        </div>
                      }
                      subtitle={`${env.createdAt} 创建`}
                      actions={
                        <>
                          <Button variant="link" className="text-xs">访问应用</Button>
                          {/* 仅 admin 端：查看详情 */}
                          {isAdmin && <Button variant="link" className="text-xs">查看详情</Button>}
                        </>
                      }
                    />
                  )) : <EmptyText>暂无前端应用</EmptyText>}
                </div>
              )}

              {/* ─── 数据库 ─── */}
              {tab === "database" && (() => {
                const collections: {
                  name: string; docs: number; size: string; indexes: number;
                  perm: string; permVar: "gray" | "blue";
                }[] = [
                  { name: "ai_bot_chat_history_5hobd2b",         docs: 8,  size: "102.86 KB", indexes: 2, perm: "仅创建者可读写", permVar: "gray" },
                  { name: "ai_bot_chat_history_5hobd2b-preview", docs: 0,  size: "0 B",       indexes: 1, perm: "仅创建者可读写", permVar: "gray" },
                  { name: "api_keys",                            docs: 5,  size: "1.53 KB",   indexes: 2, perm: "仅管理端可读写", permVar: "gray" },
                  { name: "contacts",                            docs: 31, size: "9.28 KB",   indexes: 2, perm: "所有用户可读，仅创建者可写", permVar: "blue" },
                  { name: "customers",                           docs: 26, size: "11.43 KB",  indexes: 3, perm: "所有用户可读，仅创建者可写", permVar: "blue" },
                  { name: "follow_ups",                          docs: 55, size: "19.37 KB",  indexes: 2, perm: "仅创建者可读写", permVar: "gray" },
                  { name: "health_plan_checkins",                docs: 0,  size: "0 B",       indexes: 1, perm: "仅创建者可读写", permVar: "gray" },
                  { name: "health_plan_tasks",                   docs: 4,  size: "1023 B",    indexes: 2, perm: "仅创建者可读写", permVar: "gray" },
                ];
                return (
                  <div className="space-y-2.5">
                    {/* 集合列表 */}
                    {collections.map((col, i) => (
                        <ListRow
                          key={i}
                          icon={<ListRowIcon><Database className="w-4 h-4 text-[var(--text-muted)]" /></ListRowIcon>}
                          title={
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="text-sm font-medium text-[var(--text-title)] truncate">{col.name}</span>
                              <StatusTag mode="fill" variant={col.permVar}>{col.perm}</StatusTag>
                            </div>
                          }
                          subtitle={<span className="tabular-nums">{col.docs} 文档 · {col.size} · {col.indexes} 索引</span>}
                          actions={isAdmin ? <Button variant="link" className="text-xs">管理数据</Button> : null}
                        />
                      ))}
                  </div>
                );
              })()}

              {/* ─── 云函数 ─── */}
              {tab === "functions" && (() => {
                const functions = [
                  { name: "crm-init-users",         runtime: "Nodejs18.15", status: "正常"     as const, type: "普通函数", created: "2026-04-14" },
                  { name: "crm-export",             runtime: "Nodejs18.15", status: "创建失败" as const, type: "普通函数", created: "2026-04-14" },
                  { name: "crm-aggregate",          runtime: "Nodejs18.15", status: "创建失败" as const, type: "普通函数", created: "2026-04-14" },
                  { name: "crm-seed-data",          runtime: "Nodejs18.15", status: "正常"     as const, type: "普通函数", created: "2026-04-10" },
                  { name: "crm-api",                runtime: "Nodejs18.15", status: "正常"     as const, type: "普通函数", created: "2026-04-09" },
                  { name: "crm-auth",               runtime: "Nodejs18.15", status: "正常"     as const, type: "普通函数", created: "2026-04-08" },
                  { name: "crm-skill-api",          runtime: "Nodejs18.15", status: "正常"     as const, type: "普通函数", created: "2026-04-08" },
                  { name: "lowcode-datasource-preview", runtime: "Nodejs12.16", status: "正常" as const, type: "普通函数", created: "2026-02-09" },
                ];
                return (
                  <div className="space-y-2.5">
                    {functions.map((fn, i) => {
                      const isError = fn.status === "创建失败";
                      return (
                        <ListRow
                          key={i}
                          icon={<ListRowIcon><Code2 className="w-4 h-4 text-[var(--text-muted)]" /></ListRowIcon>}
                          title={
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="text-sm font-medium text-[var(--text-title)]">{fn.name}</span>
                              <span className="px-1.5 py-0.5 text-xs text-[var(--text-muted)] bg-[var(--bg-grey-normal)] rounded-[var(--radius-lg)] tabular-nums">{fn.runtime}</span>
                              {isError ? (
                                <StatusTag mode="fill" variant="red">{fn.status}</StatusTag>
                              ) : (
                                <StatusTag mode="fill" variant="green">{fn.status}</StatusTag>
                              )}
                            </div>
                          }
                          subtitle={`${fn.type} · ${fn.created} 创建`}
                          actions={isAdmin ? <Button variant="link" className="text-xs">查看详情</Button> : null}
                        />
                      );
                    })}
                  </div>
                );
              })()}

              {/* ─── 文件存储 ─── */}
              {tab === "storage" && (() => {
                const storageFiles: { name: string; type: "folder" | "file"; size?: string; updatedAt?: string }[] = [
                  { name: "elon/", type: "folder" },
                  { name: "signature/", type: "folder" },
                  { name: "supplier-certifications/", type: "folder" },
                  { name: "test/", type: "folder" },
                  { name: "test2/", type: "folder" },
                  { name: "uploads/", type: "folder" },
                  { name: "weda-uploader/", type: "folder" },
                  { name: "weda/", type: "folder" },
                  { name: "tmp-1768899176561-lc_user_masked.json", type: "file", size: "548.99 KB", updatedAt: "2026-01-20" },
                  { name: "tmp-1768899254307-lc_user_masked.json", type: "file", size: "548.98 KB", updatedAt: "2026-01-20" },
                  { name: "tmp-1768899312445-lc_user_masked.json", type: "file", size: "549.12 KB", updatedAt: "2026-01-20" },
                  { name: "下载.html", type: "file", size: "30.53 KB", updatedAt: "2026-01-23" },
                ];
                return (
                <div className="space-y-5">
                  {/* 顶部 4 维度统计卡（与数据库/云函数共享的统计卡样式） */}
                  <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] p-4">
                    <div className="grid grid-cols-4 gap-4">
                      {[
                        { label: "对象数",  value: "32", icon: Files },
                        { label: "存储量",  value: "3.00 MB", icon: HardDrive },
                        { label: "权限",    value: "公有读", icon: ShieldCheck },
                        { label: "CDN 加速", value: "已启用", highlight: true, icon: Zap },
                      ].map(item => {
                        const Icon = item.icon;
                        return (
                          <div key={item.label} className="flex items-center gap-2.5 min-w-0">
                            <ListRowIcon><Icon className="w-4 h-4 text-[var(--text-muted)]" /></ListRowIcon>
                            <div className="min-w-0">
                              <p className="text-xs text-[var(--text-muted)] mb-0.5">{item.label}</p>
                              <p className={`text-sm font-semibold tabular-nums truncate ${item.highlight ? "text-[var(--text-success)]" : "text-[var(--text-title)]"}`}>{item.value}</p>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>

                  {/* 存储桶 + 文件列表卡 */}
                  <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden">
                    {/* 卡头：存储桶 ID + 存储管理 */}
                    <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-[var(--cp-border)]">
                      <div className="flex items-center gap-1.5 min-w-0">
                        <span className="text-sm text-[var(--text-brand)] truncate">{env.envId}-1308771514</span>
                        <button
                          type="button"
                          onClick={() => handleCopy(`${env.envId}-1308771514`)}
                          className="text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors flex-shrink-0"
                          aria-label="复制存储桶名"
                        >
                          <Copy className="w-3.5 h-3.5" />
                        </button>
                      </div>
                      {isAdmin && (
                        <Button variant="claw-outline" size="sm" className="flex-shrink-0">
                          存储管理
                          <ExternalLink className="w-3 h-3 ml-1" />
                        </Button>
                      )}
                    </div>

                    {/* 表头 */}
                    <div className="grid grid-cols-[1fr_140px_140px] gap-4 px-4 py-2.5 border-b border-[var(--cp-border)] bg-[var(--bg-grey-normal)]">
                      <span className="text-xs text-[var(--text-muted)]">文件名</span>
                      <span className="text-xs text-[var(--text-muted)]">大小</span>
                      <span className="text-xs text-[var(--text-muted)]">更新时间</span>
                    </div>

                    {/* 表体 */}
                    <div className="divide-y divide-[var(--cp-border)]">
                      {storageFiles.map(f => (
                        <div
                          key={f.name}
                          className="grid grid-cols-[1fr_140px_140px] gap-4 px-4 py-3 items-center hover:bg-[var(--bg-grey-normal)] transition-colors"
                        >
                          <div className="flex items-center gap-2.5 min-w-0">
                            {f.type === "folder" ? (
                              <FolderOpen className="w-4 h-4 text-[var(--text-brand)] flex-shrink-0" />
                            ) : (
                              <FileText className="w-4 h-4 text-[var(--text-weak)] flex-shrink-0" />
                            )}
                            <span
                              className={`text-sm truncate ${
                                f.type === "folder"
                                  ? "text-[var(--text-brand)] hover:underline cursor-pointer"
                                  : "text-[var(--text-title)]"
                              }`}
                            >
                              {f.name}
                            </span>
                          </div>
                          <span className="text-sm text-[var(--text-body)] tabular-nums">{f.size ?? "-"}</span>
                          <span className="text-sm text-[var(--text-muted)] tabular-nums">{f.updatedAt ?? "-"}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
                );
              })()}
            </div>
          </section>
        </div>
      </DialogContent>
    </Dialog>
  );
}

/* ── 子组件 ─────────────────────────────────────────────── */

/** 字段行：标签 + 值 */
function FieldRow({ label, value, children }: { label: string; value?: string; children?: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2 min-w-0">
      <span className="text-xs text-[var(--text-muted)] w-16 flex-shrink-0">{label}</span>
      {children ?? <span className="text-xs text-[var(--text-title)] truncate">{value}</span>}
    </div>
  );
}

/** 列表行图标容器（统一 9x9 brand-blue/10 浅蓝底） */
function ListRowIcon({ children }: { children: React.ReactNode }) {
  return (
    <div className="w-9 h-9 rounded-[8px] bg-[var(--bg-grey-hover)] flex items-center justify-center flex-shrink-0">
      {children}
    </div>
  );
}

/** 列表行（前端应用 / 数据集合 / 云函数 共用） */
function ListRow({ icon, title, subtitle, actions }: {
  icon: React.ReactNode;
  title: React.ReactNode;
  subtitle: React.ReactNode;
  actions: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between px-4 py-3 rounded-[var(--radius-lg)] border border-[var(--cp-border)] transition-colors">
      <div className="flex items-center gap-3 min-w-0">
        {icon}
        <div className="min-w-0">
          <div>{title}</div>
          <p className="text-xs text-[var(--text-muted)] mt-0.5">{subtitle}</p>
        </div>
      </div>
      {actions && <div className="flex items-center gap-2 flex-shrink-0 ml-3">{actions}</div>}
    </div>
  );
}

/** 空状态文字 */
function EmptyText({ children }: { children: React.ReactNode }) {
  return <div className="py-16 text-center text-sm text-[var(--text-weak)]">{children}</div>;
}
