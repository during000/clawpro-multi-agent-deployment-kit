/**
 * ProductUpdatesDrawer - 产品动态右侧边栏
 *
 * 触发：管控端侧边栏底部「铃铛」按钮
 * 设计稿：Figma node 148-15206（clawpro 引导体系设计）
 *
 * 规范：全量 token 化（颜色/圆角/阴影走 --cp-* / --text-* / design-system 组件），
 *       无硬编码 hex、无原生 <h2>/<p> 拼接：
 * - 容器阴影：var(--shadow-overlay)
 * - 卡片：<SurfaceCard hover>（卡片三态：normal / hover 蓝灰描边+微抬）
 * - 文字：Typography 语义组件（PanelTitle / CardTitle / CompactText / MetaText / HelperText）
 * - 徽标：<Badge color>（功能上线=blue / 体验优化=green）；近期更新=胶囊 Badge
 * - 滚动条：.scrollbar-on-hover（默认隐藏，滚动/悬停时显现）
 */
import { useEffect, useMemo, useRef, useState } from "react";
import { Bell, ChevronRight, X } from "lucide-react";

import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { SurfaceCard } from "@/components/ui/Surface";
import {
  PanelTitle,
  CardTitle,
  CompactText,
  MetaText,
  HelperText,
} from "@/components/ui/Typography";

export type ProductUpdateEndpoint = "admin" | "tenant";
export type ProductUpdateType = "功能上线" | "体验优化";

export interface ProductUpdateItem {
  id: string;
  /** 更新类型徽标 */
  type: ProductUpdateType;
  /** 所属端 */
  endpoint: ProductUpdateEndpoint;
  /** 标题（含「管控端｜」「用户端｜」前缀） */
  title: string;
  /** 描述 */
  desc: string;
  /** 日期 */
  date: string;
  /** 是否近期更新 */
  recent?: boolean;
  /** 「前往体验」跳转链接，无则不展示 */
  actionHref?: string;
}

interface ProductUpdatesDrawerProps {
  open: boolean;
  onClose: () => void;
  /** 更新列表，不传则使用内置演示数据 */
  items?: ProductUpdateItem[];
  /** 需要高亮的动态 id 列表（如从产品动态卡片「查看详情」进入时，高亮卡片提到的几条） */
  highlightIds?: string[];
  /** 打开时是否默认开启「仅看近期更新」开关（如从 GuideAdminNotify「立即体验」进入时为 true） */
  defaultRecentOnly?: boolean;
}

/** 类型徽标 → Badge color */
const TYPE_BADGE_COLOR: Record<ProductUpdateType, "blue" | "green"> = {
  功能上线: "blue",
  体验优化: "green",
};

const TABS: { value: "all" | ProductUpdateEndpoint; label: string }[] = [
  { value: "all", label: "全部" },
  { value: "admin", label: "管控端" },
  { value: "tenant", label: "用户端" },
];

/** 内置演示数据 */
const DEMO_UPDATES: ProductUpdateItem[] = [
  {
    id: "u-mcp",
    type: "功能上线",
    endpoint: "admin",
    title: "管控端｜新增 MCP 凭据托管功能",
    desc: "支持统一托管 MCP 凭据，配置后下发给实例自动注入，避免明文暴露与重复填写。",
    date: "2026-04-22",
    recent: true,
    actionHref: "/admin/skill-config",
  },
  {
    id: "u-tokens",
    type: "体验优化",
    endpoint: "admin",
    title: "管控端｜Tokens 统计优化",
    desc: "Tokens 统计支持按时间维度设置，记忆管理功能同步上线，用量分析更精细。",
    date: "2026-04-22",
    recent: true,
    actionHref: "/admin/tokens-monitor",
  },
  {
    id: "1",
    type: "功能上线",
    endpoint: "admin",
    title: "管控端｜Hermes 支持模型、通道和 Skill 一键配置",
    desc: "支持快捷配置 Hermes 模型、通道和 Skill，并提供角色设定和初始技能包等能力…",
    date: "2026-04-22",
    recent: true,
    actionHref: "/admin/skill-config",
  },
  {
    id: "2",
    type: "功能上线",
    endpoint: "tenant",
    title: "用户端｜ClawPro 支持新建 Hermes Agent 和 LightClaw ACE 实例",
    desc: "支持新建 Hermes Agent 和 LightClaw ACE 实例，可通过终端或 Web UI 面板配置",
    date: "2026-04-22",
    recent: true,
    actionHref: "/my-openclaw",
  },
  {
    id: "3",
    type: "体验优化",
    endpoint: "admin",
    title: "管控端｜OpenClaw 更名为 Agent",
    desc: "ClawPro 整体适配为通用 Agent 管控台，OpenClaw 相关描述更换为 Agent",
    date: "2026-04-22",
    recent: true,
  },
  {
    id: "4",
    type: "功能上线",
    endpoint: "admin",
    title: "管控端｜企业技能库支持技能版本一键升级功能",
    desc: "支持对已下发技能进行版本升级，可批量筛选该版本落后的实例并一键同步升级",
    date: "2026-04-22",
  },
  {
    id: "5",
    type: "功能上线",
    endpoint: "admin",
    title: "管控端｜初始技能包上线，搭配免费 50G 存储",
    desc: "管理员可在技能配置页面自由配置初始技能包并加入专有存储空间，OpenClaw 创建可极速下载预装技能。",
    date: "2026-04-22",
  },
  {
    id: "6",
    type: "功能上线",
    endpoint: "admin",
    title: "管控端｜ClawPro 新增法兰克福地域",
    desc: "ClawPro 法兰克福地域上线，支持欧洲区域就近部署（仅后端支持）",
    date: "2026-04-22",
  },
  {
    id: "7",
    type: "功能上线",
    endpoint: "tenant",
    title: "用户端｜Hermes 支持模型、通道和 Skill 一键配置",
    desc: "支持快捷配置 Hermes 模型、通道和 Skill，并提供角色设定和初始技能包等能力…",
    date: "2026-04-22",
    actionHref: "/my-openclaw",
  },
  {
    id: "8",
    type: "体验优化",
    endpoint: "tenant",
    title: "用户端｜OpenClaw 更名为 Agent",
    desc: "ClawPro 整体适配为通用 Agent 管控台，OpenClaw 相关描述更换为 Agent",
    date: "2026-04-22",
  },
  {
    id: "u-dialog",
    type: "功能上线",
    endpoint: "tenant",
    title: "用户端｜Hermes 支持对话视图",
    desc: "用户端 Hermes 类型的 Agent 支持开启对话视图功能，交互更自然流畅。",
    date: "2026-04-22",
    recent: true,
    actionHref: "/my-openclaw",
  },
  {
    id: "u-square",
    type: "功能上线",
    endpoint: "tenant",
    title: "用户端｜技能广场全新登场",
    desc: "在技能广场浏览和安装丰富的技能扩展，让 Agent 能力更强大、更智能。",
    date: "2026-04-22",
    recent: true,
    actionHref: "/skill-square",
  },
];

/** 是否存在「近期更新」卡片（供导航栏铃铛红点等外部入口复用，默认走内置演示数据） */
export function hasRecentProductUpdates(
  items: ProductUpdateItem[] = DEMO_UPDATES
): boolean {
  return items.some(item => item.recent);
}

export function ProductUpdatesDrawer({
  open,
  onClose,
  items = DEMO_UPDATES,
  highlightIds,
  defaultRecentOnly = false,
}: ProductUpdatesDrawerProps) {
  const [activeTab, setActiveTab] = useState<"all" | ProductUpdateEndpoint>("all");
  const [recentOnly, setRecentOnly] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  /** 抽屉根容器引用，用于「点击抽屉外区域自动收起」 */
  const drawerRef = useRef<HTMLDivElement>(null);

  const highlightSet = useMemo(
    () => new Set(highlightIds ?? []),
    [highlightIds]
  );

  // 当前 Tab 下的列表（先按端过滤）
  const tabItems = useMemo(
    () =>
      items.filter(item => activeTab === "all" || item.endpoint === activeTab),
    [items, activeTab]
  );

  // 近期更新数量（全量，不跟随 tab 切换变化）
  const recentCount = useMemo(
    () => items.filter(item => item.recent).length,
    [items]
  );

  // 各 Tab 是否含「近期更新」卡片（用于 Tab 上的红点标识）
  const recentByTab = useMemo<Record<"all" | ProductUpdateEndpoint, boolean>>(
    () => ({
      all: items.some(item => item.recent),
      admin: items.some(item => item.endpoint === "admin" && item.recent),
      tenant: items.some(item => item.endpoint === "tenant" && item.recent),
    }),
    [items]
  );

  // 是否启用「仅近期更新」过滤
  const visibleItems = useMemo(
    () => (recentOnly ? tabItems.filter(item => item.recent) : tabItems),
    [tabItems, recentOnly]
  );

  // 打开抽屉时：按 defaultRecentOnly 初始化「仅看近期更新」开关
  // （如从 GuideAdminNotify「立即体验」进入时默认开启）。
  // 携带高亮 id 时：切回「全部」Tab，并把第一条高亮动态滚动到视口中央。
  useEffect(() => {
    if (!open) return;
    setRecentOnly(defaultRecentOnly);
    if (highlightSet.size === 0) return;
    setActiveTab("all");
    const firstId = (highlightIds ?? []).find(id => highlightSet.has(id));
    if (!firstId) return;
    const timer = window.setTimeout(() => {
      const el = scrollRef.current?.querySelector(
        `[data-update-id="${firstId}"]`
      );
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 120);
    return () => window.clearTimeout(timer);
  }, [open, highlightIds, highlightSet, defaultRecentOnly]);

  // 点击抽屉外区域自动收起 + 按下 ESC 关闭
  // ⚠️ 需排除「产品动态」铃铛触发器（aria-label="产品动态"），否则 mousedown 关闭后 click 会立刻重新打开
  // 使用 mousedown 而非 click：避免与抽屉内 onClick 顺序冲突（按下即判定）
  useEffect(() => {
    if (!open) return;
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      // 点击在抽屉内部 → 不关闭
      if (drawerRef.current?.contains(target)) return;
      // 点击在「产品动态」触发器或其内部 → 交给触发器自身处理，不在此关闭
      if (
        target instanceof Element &&
        target.closest('[aria-label="产品动态"]')
      ) {
        return;
      }
      onClose();
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, onClose]);

  // 滚动时显现滚动条，停止滚动 600ms 后隐藏（配合 .scrollbar-on-hover）
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    let timer: number | undefined;
    const handleScroll = () => {
      el.setAttribute("data-scrolling", "true");
      window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        el.removeAttribute("data-scrolling");
      }, 600);
    };
    el.addEventListener("scroll", handleScroll, { passive: true });
    return () => {
      el.removeEventListener("scroll", handleScroll);
      window.clearTimeout(timer);
    };
  }, [open]);

  if (!open) return null;

  return (
    <>
      {/* 抽屉：不加遮罩蒙版，仅以抽屉容器规范阴影 var(--shadow-drawer-left) 分隔背景
       *（与表格固定列同色 rgba(0,0,0,0.06)；同步 HelpPanel / NotificationPanel 等所有 SheetContent 右侧抽屉） */}
      <div
        ref={drawerRef}
        className="fixed top-0 right-0 bottom-0 z-[9995] flex w-full max-w-[400px] flex-col border-l border-[var(--cp-border)] bg-[var(--cp-surface)] shadow-[var(--shadow-drawer-left)] animate-in slide-in-from-right duration-300"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 pt-5">
          <PanelTitle>产品动态</PanelTitle>
          <button
            onClick={onClose}
            className="flex size-7 items-center justify-center rounded-[var(--radius-lg)] text-[var(--cp-text-muted)] transition-colors hover:bg-[var(--bg-grey-hover)] hover:text-[var(--cp-text-title)]"
            aria-label="关闭"
          >
            <X className="size-4" />
          </button>
        </div>

        {/* 近期更新行（卡片化；开启「仅看近期更新」时高亮品牌蓝底）
            ⚠️ 与 Header 之间显式留 16px 间距（mt-4），不要紧贴 */}
        <div
          className={`mx-6 mt-4 mb-3 flex items-center justify-between rounded-[var(--radius-lg)] border px-4 py-3 transition-colors ${
            recentOnly
              ? "border-[var(--cp-brand-blue)] bg-[var(--cp-brand-tint)]"
              : "border-[var(--cp-border)] bg-[var(--cp-surface)]"
          }`}
        >
          <div className="flex items-center gap-1.5">
            <Bell className="size-4 text-[var(--cp-text-muted)]" />
            <CompactText tone="secondary">
              近期共计 {recentCount} 条更新
            </CompactText>
          </div>
          <label className="flex cursor-pointer items-center gap-2">
            <CompactText tone="secondary">仅看近期更新</CompactText>
            <Switch
              checked={recentOnly}
              onCheckedChange={setRecentOnly}
              aria-label="仅看近期更新"
            />
          </label>
        </div>

        {/* Tabs：与用户管理页「全部/分组」完全一致的分段选择器（纯净 SegmentGroup）
            ⚠️ 抽屉/Sheet 浮层场景下，SegmentGroup 默认 token bg `var(--bg-segment-track)` = #DBDDE432
               为 20% alpha 半透明色，会因抽屉底层叠加导致视觉偏深；此处覆盖为白底等效不透明色
               #F8F8FA，确保和用户管理页（直接铺在白底上）视觉一致。 */}
        <div className="px-6 pb-4">
          <SegmentGroup className="bg-[#F8F8FA] border-[#F8F8FA]">
            {TABS.map(tab => (
              <SegmentOption
                key={tab.value}
                active={activeTab === tab.value}
                onClick={() => setActiveTab(tab.value)}
              >
                <span className="relative inline-flex items-center">
                  {tab.label}
                  {recentByTab[tab.value] && (
                    <span
                      aria-hidden="true"
                      className="absolute -right-2.5 top-0 size-1.5 rounded-full bg-[var(--text-danger)]"
                    />
                  )}
                </span>
              </SegmentOption>
            ))}
          </SegmentGroup>
        </div>

        {/* 列表（滚动区，默认隐藏滚动条）：左对齐到与标题/统计条/tab 一致的 px-6
            ⚠️ Tabs 与列表之间的 16px 间距由 Tabs 容器 pb-4 单独控制，列表顶部 pt-0 不再叠加 */}
        <div
          ref={scrollRef}
          className="scrollbar-on-hover flex-1 overflow-y-auto px-6 pt-0 pb-3"
        >
          {visibleItems.length === 0 ? (
            <div className="py-12 text-center">
              <HelperText>暂无产品动态</HelperText>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {visibleItems.map(item => (
                <UpdateCard
                  key={item.id}
                  item={item}
                  highlighted={highlightSet.has(item.id)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </>
  );
}

/** 单条更新卡片（hover 风格对齐企业技能库卡片视图：仅描边变品牌蓝 #355EF1，不加阴影、不微抬，
 *  以免抽屉滚动区裁切微抬位移/阴影；整卡为「前往体验」热区。
 *  ⚠️ 仅当 actionHref 存在（可点击跳转）时启用 cursor-pointer + hover 描边变色；
 *     纯展示卡片（无跳转）保持默认光标，不触发交互态视觉反馈。 */
function UpdateCard({
  item,
  highlighted = false,
}: {
  item: ProductUpdateItem;
  highlighted?: boolean;
}) {
  const interactive = Boolean(item.actionHref);
  const card = (
    <SurfaceCard
      data-update-id={item.id}
      data-state={highlighted ? "selected" : undefined}
      className={
        (highlighted
          ? "px-3 py-3 animate-product-update-highlight "
          : "px-3 py-3 ") +
        (interactive
          ? "cursor-pointer transition-colors hover:border-[#355EF1]"
          : "")
      }
    >
      {/* 类型徽标 */}
      <Badge color={TYPE_BADGE_COLOR[item.type]} className="font-medium">
        {item.type}
      </Badge>

      {/* 标题 */}
      <CardTitle className="mt-2">{item.title}</CardTitle>

      {/* 描述 */}
      <MetaText as="p" className="mt-1 line-clamp-2 leading-[18px]">
        {item.desc}
      </MetaText>

      {/* 底部：日期 / 近期更新（纯文字）+ 前往体验 */}
      <div className="mt-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <MetaText tone="weak">{item.date}</MetaText>
          {item.recent && (
            <span className="flex items-center gap-1">
              <span className="size-1.5 rounded-full bg-[var(--text-danger)]" />
              <MetaText tone="secondary">近期更新</MetaText>
            </span>
          )}
        </div>
        {item.actionHref && (
          <span className="flex items-center gap-0.5">
            <CompactText tone="brand">前往体验</CompactText>
            <ChevronRight
              aria-hidden="true"
              className="size-3.5 text-[var(--text-brand)]"
            />
          </span>
        )}
      </div>
    </SurfaceCard>
  );

  // 有跳转链接时整卡可点击
  return item.actionHref ? (
    <a href={item.actionHref} className="block">
      {card}
    </a>
  ) : (
    card
  );
}
