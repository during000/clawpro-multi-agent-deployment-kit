/**
 * GuideAdminNotify - 管控端非阻断通知卡片（产品动态）
 * 对应场景：结构层 1.2 页面重新排布 / 元素层 2.6 细节优化叠加（管控端）
 *
 * 设计稿：Figma node 4000-27807
 *
 * 【卡片数量与合并规则（强制）】
 * - 用户端卡片最多展示 1 张，管控端卡片最多展示 1 张，整体最多 2 张。
 * - 同端自动合并：如果多人上传了不同内容的同端卡片，必须自动合并为
 *   一张多条内容的卡片（卡片内逐条展示），禁止同端出现 2 张及以上独立卡片。
 *
 * 与用户端 GuideModuleFloat 的区别：
 * - 管控端使用浅色渐变皮肤卡片（紫 / 蓝 / 绿），宽 220px，圆角 8px
 * - 头部「产品动态」+ 关闭按钮，底部黑色 CTA 按钮
 * - 变体：
 *   - single    单条卡片（"立即体验"）
 *   - aggregate 聚合卡片（"管控端有 N 项新增"，按钮"查看详情"）
 *   - stacked   多条叠加（可逐个关闭，后面卡片过渡显现）
 *
 * 导出：
 * - GuideAdminNotify  固定右下角浮层 wrapper（供引导体系体验面板浮窗叠加在真实页面上）
 * - AdminNotifyCard   单张卡片（presentational，供预览页 / 画廊静态陈列）
 * - AdminNotifyStack  多条叠加（presentational，可配置对齐与重置 UI）
 * - ADMIN_CARD_SKINS  三套皮肤渐变色值
 */
import { useState } from "react";

/** 通知卡片变体 */
export type AdminNotifyVariant = "single" | "aggregate" | "stacked";

export interface AdminNotifyItem {
  /** 标题（含"XX端 | "前缀，≤16 字） */
  title: string;
  /** 描述（≤30 字） */
  desc: string;
  /** 按钮文案（≤6 字），默认"立即体验" */
  btnText?: string;
  /** 皮肤索引：0=紫 / 1=蓝 / 2=绿 */
  skinIndex?: number;
  /** 所属端：admin=管控端 / tenant=用户端（用于产品动态弹窗去重聚合逻辑） */
  endpoint?: "admin" | "tenant";
  /** 关联的产品动态条目 id（点击「查看详情/立即体验」后在产品动态抽屉里高亮这些条目） */
  relatedIds?: string[];
}

/** 去掉标题里的"XX端｜""XX端 | "前缀，仅保留正文部分 */
function stripEndpointPrefix(title: string): string {
  return title.replace(/^(管控端|用户端)\s*[｜|]\s*/, "");
}

/** 品牌蓝 */
const BRAND_BLUE = "#1447E6";

/**
 * 渲染卡片标题：多条聚合变体（「XX端有 N 项新增」）中的数字 N 用品牌蓝高亮。
 * 单条卡片标题不含「项新增」聚合特征，保持原色。
 */
function renderNotifyTitle(title: string): React.ReactNode {
  if (!/项新增/.test(title)) return title;
  return title.split(/(\d+)/).map((part, i) =>
    /^\d+$/.test(part) ? (
      <span key={i} style={{ color: BRAND_BLUE }}>
        {part}
      </span>
    ) : (
      part
    ),
  );
}

/** 根据标题前缀推断所属端 */
function inferEndpoint(title: string): "admin" | "tenant" | undefined {
  if (title.startsWith("管控端")) return "admin";
  if (title.startsWith("用户端")) return "tenant";
  return undefined;
}

/**
 * 产品动态非阻断弹窗逻辑：
 * - 卡片分 4 种：用户端单条 / 用户端多条 / 管控端单条 / 管控端多条
 * - 按端分组：每个端最多产出「一张」卡片（单条→原卡片；多条→聚合卡「XX端有 N 项新增」）
 * - 最多展示两张（管控端 + 用户端各一），严禁重复出现同端卡片
 * - 管控端在前、用户端在后
 */
export function buildProductUpdateNotices(items: AdminNotifyItem[]): AdminNotifyItem[] {
  const groups: Record<"admin" | "tenant", AdminNotifyItem[]> = { admin: [], tenant: [] };
  items.forEach(item => {
    const ep = item.endpoint ?? inferEndpoint(item.title);
    if (ep) groups[ep].push(item);
  });

  const result: AdminNotifyItem[] = [];
  (["admin", "tenant"] as const).forEach(ep => {
    const group = groups[ep];
    if (group.length === 0) return;
    const label = ep === "admin" ? "管控端" : "用户端";
    const defaultSkin = ep === "admin" ? 0 : 1; // 管控端=#F2F5FF / 用户端=#F2FBFF
    if (group.length === 1) {
      result.push({
        ...group[0],
        endpoint: ep,
        skinIndex: group[0].skinIndex ?? defaultSkin,
        relatedIds: group[0].relatedIds ?? [],
      });
    } else {
      result.push({
        title: `${label}有 ${group.length} 项新增`,
        desc: group.map(g => stripEndpointPrefix(g.title)).join("、"),
        btnText: "查看详情",
        skinIndex: group[0].skinIndex ?? defaultSkin,
        endpoint: ep,
        relatedIds: group.flatMap(g => g.relatedIds ?? []),
      });
    }
  });

  return result.slice(0, 2);
}

/** 皮肤：管控端(#F2F5FF) / 用户端(#F2FBFF) / 绿三套浅色渐变（对齐 Figma node 4000-27807） */
export const ADMIN_CARD_SKINS = [
  // index 0 —— 管控端
  { bg: "linear-gradient(178deg, #F2F5FF -24.67%, rgba(252, 252, 254, 0.93) 98.81%), #FCFCFE", border: "#E3E8FA" },
  // index 1 —— 用户端
  { bg: "linear-gradient(178deg, #F2FBFF -24.67%, rgba(252, 252, 254, 0.93) 98.81%), #FCFCFE", border: "#E3EDFA" },
  { bg: "linear-gradient(179deg, #F2FFF5 0%, #FCFCFE 100%)", border: "#E8F1E9" },
];

/** 单张通知卡片（presentational） */
export function AdminNotifyCard({
  title,
  desc,
  btnText = "立即体验",
  skinIndex = 0,
  onClose,
  onAction,
}: AdminNotifyItem & { onClose?: () => void; onAction?: () => void }) {
  const skin = ADMIN_CARD_SKINS[skinIndex % ADMIN_CARD_SKINS.length];
  return (
    <div
      style={{
        width: 220,
        padding: 12,
        background: skin.bg,
        border: `1px solid ${skin.border}`,
        borderRadius: 8,
        boxShadow: "0px 4px 12px 0px rgba(0, 0, 0, 0.1)",
        display: "flex",
        flexDirection: "column",
        gap: 8,
      }}
    >
      {/* 头部：产品动态 + 关闭 */}
      <div className="flex justify-between items-center" style={{ height: 21 }}>
        <span style={{ fontSize: 12, color: "var(--admin-notify-subtitle)" }}>产品动态</span>
        <button
          onClick={onClose}
          className="flex items-center justify-center hover:opacity-70 transition-opacity"
          style={{ width: 20, height: 20, borderRadius: 4, background: "transparent" }}
          aria-label="关闭"
        >
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M13.9595 7.17139L11.1304 9.99951L13.9595 12.8286L12.8286 13.9595L9.99951 11.1304L7.17139 13.9595L6.04053 12.8286L8.86865 9.99951L6.04053 7.17139L7.17139 6.04053L9.99951 8.86865L12.8286 6.04053L13.9595 7.17139Z" fill="black" fillOpacity="0.7"/>
          </svg>
        </button>
      </div>
      {/* 内容区 */}
      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        {/* 标题+描述 */}
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <h4 className="font-medium" style={{ fontSize: 13, color: "var(--admin-notify-title)", lineHeight: "20px", margin: 0 }}>{renderNotifyTitle(title)}</h4>
          <p style={{ fontSize: 12, color: "var(--admin-notify-desc)", lineHeight: "18px", margin: 0 }}>{desc}</p>
        </div>
        {/* 按钮 */}
        <button
          onClick={onAction}
          className="w-full text-center font-medium transition-colors bg-[#000] hover:bg-[#333] text-white"
          style={{ fontSize: 12, height: 28, borderRadius: 4, border: "none", cursor: "pointer", padding: "4px 12px" }}
        >
          {btnText}
        </button>
      </div>
    </div>
  );
}

/**
 * 多条叠加（presentational）：关闭顶部卡片后，后面的卡片通过过渡动画显现。
 * - align="center" 水平居中（预览画廊），minHeight 250；align="end" 右对齐（右下角浮层），minHeight 160
 * - showReset：全部关闭后是否显示「已全部关闭 / 重置演示」（画廊用 true，浮层用 false）
 */
export function AdminNotifyStack({
  cards,
  align = "center",
  showReset = true,
  onAllClosed,
  onAction,
}: {
  cards: AdminNotifyItem[];
  align?: "center" | "end" | "start";
  showReset?: boolean;
  onAllClosed?: () => void;
  /** 点击前置卡片的「查看详情/立即体验」按钮 */
  onAction?: (item: AdminNotifyItem) => void;
}) {
  const [dismissed, setDismissed] = useState<number[]>([]);
  const visibleCards = cards.filter((_, i) => !dismissed.includes(i));
  const allDismissed = visibleCards.length === 0;
  const isCenter = align === "center";
  const isStart = align === "start";
  const justifyClass =
    align === "center" ? "justify-center" : isStart ? "justify-start" : "justify-end";
  const horizClass = align === "center" ? "left-1/2" : isStart ? "left-0" : "right-0";

  const handleClose = (visibleIndex: number) => {
    const originalIndex = cards.indexOf(visibleCards[visibleIndex]);
    const next = [...dismissed, originalIndex];
    setDismissed(next);
    if (next.length >= cards.length) onAllClosed?.();
  };

  const handleReset = () => setDismissed([]);

  // 前置卡片（正常流，决定容器高度，其底边即对齐 wrapper 底部）
  const frontCard = visibleCards[0];
  // 背后叠层卡片：最多 1 张（整体最多两张卡片）；往上露出一截、不透明
  const peekCards = visibleCards.slice(1, 2);

  return (
    <div className={`relative flex ${justifyClass}`} style={{ width: 232 }}>
      {/* 背后叠层卡片 */}
      {[...peekCards].reverse().map((card, ri) => {
        const i = peekCards.length - ri; // 1, 2...
        const scale = 1 - i * 0.05;
        return (
          <div
            key={`${card.title}-${card.skinIndex}`}
            className={`absolute transition-all duration-300 ease-out ${horizClass}`}
            style={{
              top: -(i * 6),
              transform: isCenter
                ? `translateX(-50%) scale(${scale})`
                : `scale(${scale})`,
              transformOrigin: "top center",
              pointerEvents: "none",
              zIndex: 0,
            }}
          >
            <AdminNotifyCard
              title={card.title}
              desc={card.desc}
              btnText={card.btnText}
              skinIndex={card.skinIndex}
            />
          </div>
        );
      })}
      {/* 前置卡片 */}
      {frontCard && (
        <div className="relative" style={{ zIndex: 1 }}>
          <AdminNotifyCard
            title={frontCard.title}
            desc={frontCard.desc}
            btnText={frontCard.btnText}
            skinIndex={frontCard.skinIndex}
            onClose={() => handleClose(0)}
            onAction={() => onAction?.(frontCard)}
          />
        </div>
      )}
      {showReset && allDismissed && (
        <div className="flex flex-col items-center justify-center gap-3 py-10">
          <p className="text-sm text-gray-400">已全部关闭</p>
          <button
            onClick={handleReset}
            className="text-xs text-blue-500 hover:text-blue-700 transition-colors underline"
          >
            重置演示
          </button>
        </div>
      )}
    </div>
  );
}

interface GuideAdminNotifyProps {
  open: boolean;
  onClose: () => void;
  /** 变体：single 单条 / aggregate 聚合 / stacked 多条叠加 */
  variant?: AdminNotifyVariant;
  /** 内容项列表（single / aggregate 使用第一条；stacked 使用全部） */
  items: AdminNotifyItem[];
  /** 点击卡片「查看详情/立即体验」按钮回调（携带该卡片，含 relatedIds） */
  onAction?: (item: AdminNotifyItem) => void;
}

/** 固定右下角浮层 wrapper —— 供引导体系体验面板浮窗叠加在真实页面之上 */
export function GuideAdminNotify({
  open,
  onClose,
  variant = "single",
  items,
  onAction,
}: GuideAdminNotifyProps) {
  if (!open || items.length === 0) return null;

  return (
    <div
      className="fixed z-[9990] animate-in slide-in-from-bottom-4 duration-300"
      style={{
        // 左下角导航栏上层：在 240px 侧栏内水平居中（220px 卡片）
        left: "calc((var(--admin-sidebar-width) - 220px) / 2)",
        // 用户账号（72px footer）上方 12px
        bottom: "calc(var(--admin-sidebar-footer-height) + 12px)",
      }}
    >
      {variant === "stacked" && items.length > 1 ? (
        <AdminNotifyStack
          cards={buildProductUpdateNotices(items)}
          align="start"
          showReset={false}
          onAllClosed={onClose}
          onAction={onAction}
        />
      ) : (
        <AdminNotifyCard
          title={items[0].title}
          desc={items[0].desc}
          btnText={items[0].btnText}
          skinIndex={items[0].skinIndex}
          relatedIds={items[0].relatedIds}
          onClose={onClose}
          onAction={() => onAction?.(items[0])}
        />
      )}
    </div>
  );
}
