/**
 * GuidePointBubble - 单UI提示气泡
 * 对应场景：元素层 2.1 新增按钮 / 2.2 新增表格列 / 2.3 新增筛选
 * 
 * ══ 分类体系（对齐组件交互设计稿） ══
 * 
 * 1/ 单UI提示，直接在UI附近展示
 *   1.1 纯文本类型 — 仅标题+描述，支持四个方向（top/bottom/left/right）
 *       - 短标题：仅一行标题
 *       - 标题+描述：标题+多行描述文字
 *       - 带列表：标题+有序/无序列表描述
 *   1.2 纯文本 + 按钮 — 底部带操作按钮
 *       - 有副标题按钮：副标题 + 按钮
 *       - 有内容+按钮：标题 + 描述 + 按钮
 *   1.3 纯文本 + 图片 — 带配图区域 + 描述 + 按钮
 *   1.4 重点推送通知 — 蓝色背景，带图标+强调样式（如版本升级）
 * 
 * 颜色变体（variant）：
 * - light: 白底黑字（默认）
 * - dark: 蓝底白字（用于推送通知模式）
 * 
 * 注：步骤指引气泡（带呼吸灯+高亮遮罩）请使用 GuideHighlightBubble 组件
 */
import { ArrowRight, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";

/** 气泡颜色变体 */
export type PointBubbleVariant = "light" | "dark";

/** 内容变体 */
export type PointBubbleContentVariant = "text-only" | "text-button" | "text-image" | "push-notice";

interface GuidePointBubbleProps {
  open: boolean;
  onClose: () => void;
  /** 标题 */
  title: string;
  /** 描述 */
  description: string;
  /** 副标题（可选，用于 1.2 有副标题按钮模式） */
  subtitle?: string;
  /** 颜色变体 */
  variant?: PointBubbleVariant;
  /** 内容变体 */
  contentVariant?: PointBubbleContentVariant;
  /** 可选图片（text-image 模式） */
  image?: string;
  /** 当前步骤（从 1 开始） */
  currentStep?: number;
  /** 总步骤数 */
  totalSteps?: number;
  /** 是否显示步骤 */
  showSteps?: boolean;
  /** 下一步回调 */
  onNext?: () => void;
  /** 上一步回调 */
  onPrev?: () => void;
  /** 气泡方向（箭头指向目标元素的方向） */
  placement?: "top" | "bottom" | "left" | "right";
  /** 是否显示脉冲热点 */
  showHotspot?: boolean;
  /** 主操作按钮文案（text-button / push-notice 模式） */
  actionText?: string;
  /** 主操作按钮回调 */
  onAction?: () => void;
  /** 次要按钮文案（双按钮模式，如"了解更多"，位于主按钮左侧） */
  secondaryActionText?: string;
  /** 次要按钮回调 */
  onSecondaryAction?: () => void;
  /** 描述列表项（用于 1.1 带列表模式） */
  listItems?: string[];
  /** 图片下方标注文本（如日期"2026-04-20"） */
  imageCaption?: string;
  /** 推送通知中的配图（1.4 大图模式） */
  noticeImage?: string;
  /** 标题右侧蓝色标签（如"还有 7 天上线"），仅浅色普通气泡显示 */
  tag?: string;
  /** 热点形状：circle（默认圆形脉冲）或 rect（蓝色圆角矩形标注） */
  hotspotShape?: "circle" | "rect";
  /** 矩形热点尺寸（hotspotShape="rect" 时生效） */
  hotspotSize?: { width: number; height: number };
  /** 端类型 — 控制按钮圆角（admin=4px 方角，tenant=full 胶囊） */
  endpoint?: "admin" | "tenant";
  /** 样式覆盖 */
  style?: React.CSSProperties;
}

export function GuidePointBubble({
  open,
  onClose,
  title,
  description,
  subtitle,
  variant = "light",
  contentVariant = "text-button",
  image,
  currentStep,
  totalSteps,
  showSteps = true,
  onNext,
  onPrev,
  placement = "bottom",
  showHotspot = true,
  actionText,
  onAction,
  listItems,
  imageCaption,
  noticeImage,
  tag,
  secondaryActionText,
  onSecondaryAction,
  hotspotShape = "circle",
  hotspotSize = { width: 120, height: 36 },
  endpoint = "tenant",
  style,
}: GuidePointBubbleProps) {
  if (!open) return null;

  const isMultiStep = showSteps && totalSteps !== undefined && totalSteps > 1;
  const isLast = currentStep === totalSteps;
  const isPushNotice = contentVariant === "push-notice";
  // 步骤指引模式：dark 变体 + 多步骤 → 使用纯蓝底（#2C59E9），区别于 push-notice 渐变
  const isStepGuide = variant === "dark" && isMultiStep;

  // 颜色：全部走 index.css 中的 --guide-bubble-* 语义 token（对齐 Figma node 4096:9477）
  const isDark = variant === "dark" || isPushNotice;
  const bubbleBg = isDark ? "" : "bg-[var(--guide-bubble-bg)]";
  const bubbleBorder = isDark ? "border-transparent" : "border-[var(--guide-bubble-border)]";
  const titleColor = isDark ? "text-[var(--guide-bubble-push-title)]" : "text-[var(--guide-bubble-title)]";
  const descColor = isDark ? "text-[var(--guide-bubble-push-desc)]" : "text-[var(--guide-bubble-desc)]";
  const closeColor = isDark ? "text-white hover:bg-white/10" : "text-[var(--guide-bubble-close)] hover:bg-black/5";
  const stepColor = isDark ? "text-white/90" : "text-[var(--guide-bubble-desc)]";
  // 1.2 按钮：主按钮（深色推送时反白）
  const btnBg = isDark
    ? "bg-[var(--guide-bubble-btn-secondary-bg)] text-[var(--text-brand)] hover:bg-white/90"
    : "bg-[var(--guide-bubble-btn-primary-bg)] text-[var(--guide-bubble-btn-primary-text)] hover:opacity-90";
  // 按钮圆角：admin=4px 方角，tenant=full 胶囊
  const btnRadius = endpoint === "admin" ? "rounded-[4px]" : "rounded-full";
  const btnRadiusPx = endpoint === "admin" ? "rounded-[4px]" : "rounded-[24px]";
  // 步骤指引箭头使用纯蓝 #2C59E9，push-notice 也用同色
  const arrowFill = isDark ? "#2C59E9" : "var(--guide-bubble-bg)";
  const arrowStroke = isDark ? "#2C59E9" : "var(--guide-bubble-arrow-stroke)";

  // 箭头方向对应的定位
  // 用 calc(...-1px) 让箭头向气泡方向回移 1px，使三角底边压住气泡边缘，
  // 与气泡连成一个整体形状（消除接缝处的分隔线）。
  const arrowPositionClasses: Record<string, string> = {
    top: "bottom-0 left-1/2 -translate-x-1/2 translate-y-[calc(100%-2px)] rotate-180",
    bottom: "top-0 left-1/2 -translate-x-1/2 -translate-y-[calc(100%-2px)]",
    left: "right-0 top-1/2 -translate-y-1/2 translate-x-[11px] rotate-90",
    right: "left-0 top-1/2 -translate-y-1/2 -translate-x-[11px] -rotate-90",
  };

  // 脉冲热点位置
  // left/right 时热点与气泡水平对齐（flex-row + items-center），
  // 中心圆与三角箭头尖端保持 4px 间距：箭头伸出 11px + 4px 间距 - 7px 半径 = 8px margin
  const hotspotPositionClasses: Record<string, string> = {
    top: "order-last mx-auto mt-2",
    bottom: "order-first mx-auto mb-2",
    left: "order-last ml-2",
    right: "order-first mr-2",
  };

  // 容器 flex 方向
  const containerDirection = placement === "left" || placement === "right" ? "flex-row" : "flex-col";

  return (
    // data-billing-exempt: 新功能引导气泡属于查看/关闭类通知，
    // 停服态下应保持可用（可读正文、可点 × 关闭），豁免全局禁用层。
    <div
      className={`relative z-[9985] flex ${containerDirection} items-center`}
      style={style}
      data-billing-exempt
    >
      {/* 脉冲热点 */}
      {showHotspot && hotspotShape === "circle" && (
        <div className={`relative w-3.5 h-3.5 shrink-0 ${hotspotPositionClasses[placement]}`}>
          <div className="absolute inset-0 rounded-full bg-[var(--text-brand)] animate-ping opacity-40" />
          <div className="absolute inset-0.5 rounded-full bg-[var(--text-brand)]" />
        </div>
      )}
      {/* 矩形标注热点 */}
      {showHotspot && hotspotShape === "rect" && (
        <div
          className={`relative shrink-0 ${hotspotPositionClasses[placement]}`}
          style={{ width: hotspotSize.width, height: hotspotSize.height }}
        >
          <div
            className="absolute inset-0 animate-pulse opacity-60"
            style={{
              borderRadius: "6px",
              border: "1px solid #1447E6",
              boxShadow: "0 4px 4px 0 rgba(20, 71, 230, 0.12)",
            }}
          />
          <div
            className="absolute inset-0"
            style={{
              borderRadius: "6px",
              border: "1px solid #1447E6",
              boxShadow: "0 4px 4px 0 rgba(20, 71, 230, 0.12)",
            }}
          />
        </div>
      )}

      {/* 气泡 */}
      <div className="relative animate-in fade-in slide-in-from-top-1 duration-200">
        {/* 三角箭头：填充三角盖住气泡边框接缝；
            描边仅画两条斜边且端点内缩，任何旋转方向都不会出现多余线条 */}
        <div className={`absolute ${arrowPositionClasses[placement]}`}>
          <svg width="16" height="9" viewBox="0 0 16 9" fill="none" style={{ display: "block" }}>
            {/* 填充三角（含底边重叠区） */}
            <path d="M0 9L8 1L16 9Z" fill={arrowFill} />
            {/* 仅两条斜边，端点内缩 1px 避免与气泡边框重合产生可见线 */}
            <path d="M1.5 8L8 1.5L14.5 8" stroke={arrowStroke} strokeWidth="1" fill="none" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </div>

        {/* 气泡主体 */}
        <div
          className={`${isStepGuide ? "w-[278px]" : "w-[280px]"} rounded-[var(--guide-bubble-radius)] border overflow-hidden ${bubbleBg} ${bubbleBorder}`}
          style={{
            boxShadow: isDark ? "var(--guide-bubble-push-shadow)" : "var(--guide-bubble-shadow)",
            ...(isStepGuide
              ? { background: "#2C59E9" }
              : isPushNotice || isDark
              ? { background: "var(--guide-bubble-push-gradient)" }
              : {}),
          }}
        >
          
          {/* ─── 重点推送通知样式（1.4） ─── */}
          {isPushNotice && (
            <>
              <div className="p-3 pb-2">
                <div className="flex items-center justify-between gap-2 mb-2">
                  <div className="flex items-center gap-2">
                    <svg width="16" height="16" viewBox="0 0 23 23" fill="none" className="shrink-0">
                      <g filter="url(#filter0_d_4096_9585)">
                        <path fillRule="evenodd" clipRule="evenodd" d="M8.36212 13.0625C9.395 14.1123 9.51456 15.2407 8.36967 16.4042C7.66034 17.1247 6.5119 17.5576 4.93146 17.7429C4.84968 17.7527 4.76701 17.7522 4.68479 17.7424C4.12924 17.6749 3.72346 17.1811 3.75013 16.6171L3.7559 16.5433L3.77857 16.3567C3.97235 14.85 4.3919 13.7469 5.07323 13.0545C6.21812 11.8914 7.32879 12.0132 8.36212 13.0625ZM17.8245 2.03464L18.0032 2.08931L18.1778 2.14753C18.6135 2.30027 18.9546 2.64498 19.1027 3.08219C19.7992 5.11684 19.5778 7.17815 18.4552 9.22169C17.9925 10.0635 17.4005 10.8603 16.6801 11.6119L16.4805 11.8159L16.2916 12.0008L16.2863 12.0581C16.1552 13.2821 15.2267 14.9008 13.5161 16.9936L13.3676 17.1745L13.0943 17.5012C12.785 17.8683 12.209 17.7661 12.0285 17.3323L12.0072 17.2741L11.0507 14.2216L10.9641 14.1616C10.4267 13.7788 9.91476 13.3613 9.43164 12.9119L9.19386 12.6852L8.96097 12.4541C8.27309 11.7541 7.65395 10.9897 7.11209 10.1715L7.0592 10.0892L3.94454 9.08213C3.50632 8.93991 3.36498 8.39814 3.64321 8.0577L3.68009 8.01592L3.72009 7.97859C6.28187 5.74039 8.17209 4.63062 9.52586 4.72306L9.61297 4.73062L9.66364 4.73684L9.77786 4.62128C10.3921 4.01062 11.0352 3.49107 11.7076 3.06397L11.9325 2.92486L12.1392 2.80397C14.0156 1.73998 15.9205 1.47776 17.8245 2.03464ZM12.0826 6.33281C11.3034 7.11236 11.3061 8.37857 12.0888 9.16123C12.8714 9.94389 14.1377 9.94656 14.9172 9.16746C15.6963 8.38791 15.6937 7.1217 14.911 6.33904C14.1283 5.55638 12.8621 5.55371 12.0826 6.33281Z" fill="url(#paint0_linear_4096_9585)"/>
                      </g>
                      <defs>
                        <filter id="filter0_d_4096_9585" x="0" y="0" width="23" height="23" filterUnits="userSpaceOnUse" colorInterpolationFilters="sRGB">
                          <feFlood floodOpacity="0" result="BackgroundImageFix"/>
                          <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="hardAlpha"/>
                          <feOffset dy="1.75"/>
                          <feGaussianBlur stdDeviation="1.75"/>
                          <feColorMatrix type="matrix" values="0 0 0 0 0.286275 0 0 0 0 0.341176 0 0 0 0 0.439216 0 0 0 0.197513 0"/>
                          <feBlend mode="normal" in2="BackgroundImageFix" result="effect1_dropShadow_4096_9585"/>
                          <feBlend mode="normal" in="SourceGraphic" in2="effect1_dropShadow_4096_9585" result="shape"/>
                        </filter>
                        <linearGradient id="paint0_linear_4096_9585" x1="11.5" y1="1.75039" x2="11.5" y2="17.7501" gradientUnits="userSpaceOnUse">
                          <stop stopColor="white"/>
                          <stop offset="1" stopColor="white" stopOpacity="0.35"/>
                        </linearGradient>
                      </defs>
                    </svg>
                    <h4 className="text-sm font-semibold text-[var(--guide-bubble-push-title)]">{title}</h4>
                  </div>
                  <button
                    onClick={onClose}
                    className="shrink-0 w-5 h-5 rounded flex items-center justify-center transition-colors hover:bg-white/10"
                  >
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                      <path d="M13.96 7.1709L11.1309 9.99902L13.96 12.8281L12.8291 13.959L10 11.1299L7.17188 13.959L6.04102 12.8281L8.86914 9.99902L6.04102 7.1709L7.17188 6.04004L10 8.86816L12.8291 6.04004L13.96 7.1709Z" fill="white"/>
                    </svg>
                  </button>
                </div>
                <p className="text-xs leading-relaxed text-[var(--guide-bubble-push-desc)]">{description}</p>
              </div>
              {/* 推送通知配图区（1.4 大图模式，如版本升级） */}
              {noticeImage !== undefined && (
                <div className="mx-3 mb-3 rounded-[var(--guide-bubble-radius)] overflow-hidden">
                  {noticeImage ? (
                    <img src={noticeImage} alt={title} className="w-full h-[128px] object-cover object-top" />
                  ) : (
                    /* 无图片时渲染 Figma 设计稿占位 SVG */
                    <svg width="100%" height="128" viewBox="0 0 260 128" fill="none" preserveAspectRatio="xMidYMid slice">
                      <mask id="mask0_push_placeholder" style={{ maskType: "alpha" }} maskUnits="userSpaceOnUse" x="0" y="0" width="260" height="128">
                        <rect width="260" height="128" rx="4" fill="url(#paint0_push_placeholder)"/>
                      </mask>
                      <g mask="url(#mask0_push_placeholder)">
                        <path d="M0 0H260V128H0V0Z" fill="url(#paint1_push_placeholder)"/>
                        <path d="M274.5 28.5V145.5H47.5V28.5H274.5Z" fill="white" fillOpacity="0.85" stroke="white"/>
                        <path d="M107.5 64.5V110.5H17.5V64.5H107.5Z" fill="white" fillOpacity="0.98" stroke="white"/>
                      </g>
                      <defs>
                        <linearGradient id="paint0_push_placeholder" x1="130" y1="0" x2="251.5" y2="128.343" gradientUnits="userSpaceOnUse">
                          <stop stopColor="#ECEEF6"/>
                          <stop offset="1" stopColor="#8A8C90" stopOpacity="0.47"/>
                        </linearGradient>
                        <linearGradient id="paint1_push_placeholder" x1="6" y1="-30.6572" x2="130" y2="128" gradientUnits="userSpaceOnUse">
                          <stop stopColor="#6A8EFF"/>
                          <stop offset="1" stopColor="#6A8EFF" stopOpacity="0.33"/>
                        </linearGradient>
                      </defs>
                    </svg>
                  )}
                </div>
              )}
              {/* 推送通知底部按钮 — 仅当有 actionText 或 secondaryActionText 时显示 */}
              {(actionText || secondaryActionText) && (
                <div className="px-3 pb-3 flex justify-end items-center gap-2">
                  {secondaryActionText && (
                    <button
                      onClick={onSecondaryAction || onClose}
                      className={`h-[28px] px-4 text-[12px] font-medium ${btnRadiusPx} transition-colors inline-flex items-center border border-white/40 text-white hover:bg-white/10`}
                    >
                      {secondaryActionText}
                    </button>
                  )}
                  <button
                    onClick={onAction || onClose}
                    className={`h-[28px] px-4 text-[12px] font-medium ${btnRadiusPx} transition-colors inline-flex items-center bg-white text-[#020617] hover:bg-white/90`}
                  >
                    {actionText}
                  </button>
                </div>
              )}
            </>
          )}

          {/* ─── 普通气泡样式（1.1 / 1.2 / 1.3） ─── */}
          {!isPushNotice && (
            <>
              {/* 文字区 */}
              <div className="p-3 pb-2">
                <div className="flex items-start justify-between gap-2 mb-1.5">
                  <div className="flex-1 min-w-0">
                    {/* 副标题（1.2 有副标题按钮模式） */}
                    {subtitle && (
                      <span className={`text-[11px] block mb-1 ${isDark ? "text-[var(--guide-bubble-push-desc)]" : "text-[var(--text-muted)]"}`}>
                        {subtitle}
                      </span>
                    )}
                    <div className="flex items-center gap-2 flex-wrap">
                      <h4 className={`text-[13px] font-medium ${titleColor}`}>{title}</h4>
                      {tag && !isDark && (
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded-[var(--radius-sm)] text-[11px] font-medium leading-[18px] text-[var(--text-brand)] bg-[var(--bg-brand-selected)]">
                          {tag}
                        </span>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={onClose}
                    className={`shrink-0 w-5 h-5 rounded flex items-center justify-center transition-colors ${closeColor}`}
                  >
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <path d="M13.9595 7.17139L11.1304 9.99951L13.9595 12.8286L12.8286 13.9595L9.99951 11.1304L7.17139 13.9595L6.04053 12.8286L8.86865 9.99951L6.04053 7.17139L7.17139 6.04053L9.99951 8.86865L12.8286 6.04053L13.9595 7.17139Z" fill="currentColor"/>
                    </svg>
                  </button>
                </div>
                {/* 描述文字 */}
                {description && (
                  <p className={`text-xs leading-relaxed ${descColor}`}>{description}</p>
                )}
                {/* 有序列表（1.1 带有序文本模式，数字序号，对齐设计稿，≤3 条） */}
                {listItems && listItems.length > 0 && (
                  <ol className={`mt-2 space-y-1 text-xs leading-[20px] ${descColor}`}>
                    {listItems.map((item, i) => (
                      <li key={i} className="flex items-start gap-1.5">
                        <span className="shrink-0 tabular-nums">{i + 1}.</span>
                        <span className="flex-1">{item}</span>
                      </li>
                    ))}
                  </ol>
                )}
              </div>

              {/* 可选图片（text-image 模式，1.3） */}
              {contentVariant === "text-image" && (
                <div className="mx-3 mb-3 space-y-1.5">
                  <div className="rounded-[var(--guide-bubble-radius)] overflow-hidden border border-[var(--guide-bubble-border)]">
                    {image ? (
                      <img src={image} alt={title} className="w-full h-[146px] object-cover object-top" />
                    ) : (
                      /* 无图片时渲染设计稿占位：灰底 #ECEEF6 + 白色半透明内框 */
                      <div className="w-full h-[146px] bg-[#ECEEF6] flex items-center justify-center">
                        <div className="w-[74%] h-[68%] rounded-[var(--guide-bubble-radius)] bg-white/85 border border-white" />
                      </div>
                    )}
                  </div>
                  {imageCaption && (
                    <span className="block text-xs text-[var(--text-weak)] px-0.5">{imageCaption}</span>
                  )}
                </div>
              )}

              {/* 步骤操作（2/ 步骤指引模式 — 对齐 Figma 3999:54590 步骤指引组件集） */}
              {isMultiStep && (
                <div className="px-3 pb-3 flex items-center justify-between">
                  {/* 左侧：步骤计数 */}
                  <span className={`text-[12px] tabular-nums ${stepColor}`}>
                    {currentStep}/{totalSteps}
                  </span>
                  {/* 右侧：导航按钮 */}
                  <div className="flex items-center gap-2">
                    {/* 上一步箭头（第1步时 disabled） */}
                    {isDark ? (
                      <button
                        onClick={onPrev}
                        disabled={!currentStep || currentStep <= 1}
                        className={`w-[38px] h-[28px] ${btnRadiusPx} flex items-center justify-center border transition-colors ${
                          !currentStep || currentStep <= 1
                            ? "border-white/20 text-white/30 cursor-not-allowed"
                            : "border-white/40 text-white hover:bg-white/10"
                        }`}
                      >
                        <ArrowLeft className="w-4 h-4" />
                      </button>
                    ) : (
                      <Button
                        variant={endpoint === "admin" ? "claw-outline" : "tenant-outline"}
                        size="claw-sm"
                        onClick={onPrev}
                        disabled={!currentStep || currentStep <= 1}
                        className="w-[38px] h-[28px] !px-0"
                      >
                        <ArrowLeft className="w-4 h-4" />
                      </Button>
                    )}
                    {/* 最后一步：显示文字按钮，否则显示右箭头 */}
                    {isLast ? (
                      isDark ? (
                        <button
                          onClick={onAction || onClose}
                          className={`h-[28px] px-4 text-[12px] font-medium ${btnRadiusPx} transition-colors inline-flex items-center bg-white text-[#020617] hover:bg-white/90`}
                        >
                          {actionText || "我知道了"}
                        </button>
                      ) : (
                        <Button
                          variant={endpoint === "admin" ? "claw-outline" : "tenant-outline"}
                          size="claw-sm"
                          onClick={onAction || onClose}
                          className="h-[28px] px-4 text-[12px]"
                        >
                          {actionText || "我知道了"}
                        </Button>
                      )
                    ) : (
                      isDark ? (
                        <button
                          onClick={onNext}
                          className={`w-[38px] h-[28px] ${btnRadiusPx} flex items-center justify-center border transition-colors border-white/40 text-white hover:bg-white/10`}
                        >
                          <ArrowRight className="w-4 h-4" />
                        </button>
                      ) : (
                        <Button
                          variant={endpoint === "admin" ? "claw-outline" : "tenant-outline"}
                          size="claw-sm"
                          onClick={onNext}
                          className="w-[38px] h-[28px] !px-0"
                        >
                          <ArrowRight className="w-4 h-4" />
                        </Button>
                      )
                    )}
                  </div>
                </div>
              )}

              {/* 单/双按钮（text-button 模式，1.2）：次要按钮在左（白底描边），主按钮在右（黑底白字） */}
              {!isMultiStep && contentVariant === "text-button" && (
                <div className="px-3 pb-3 flex justify-end gap-2">
                  {secondaryActionText && (
                    isDark ? (
                      <button
                        onClick={onSecondaryAction || onClose}
                        className={`h-[28px] px-3 text-[11px] font-medium ${btnRadius} transition-colors inline-flex items-center bg-[var(--guide-bubble-btn-secondary-bg)] border border-[var(--guide-bubble-btn-secondary-border)] text-[var(--guide-bubble-btn-secondary-text)] hover:opacity-80`}
                      >
                        {secondaryActionText}
                      </button>
                    ) : (
                      <Button
                        variant={endpoint === "admin" ? "claw-outline" : "tenant-outline"}
                        size="claw-sm"
                        onClick={onSecondaryAction || onClose}
                        className="h-[28px] px-3 text-[11px]"
                      >
                        {secondaryActionText}
                      </Button>
                    )
                  )}
                  {isDark ? (
                    <button
                      onClick={onAction || onClose}
                      className={`h-[28px] px-3 text-[11px] font-medium ${btnRadius} transition-colors inline-flex items-center ${btnBg}`}
                    >
                      {actionText || "我知道了"}
                    </button>
                  ) : (
                    <Button
                      variant={endpoint === "admin" ? "claw-primary" : "tenant-primary"}
                      size="claw-sm"
                      onClick={onAction || onClose}
                      className="h-[28px] px-3 text-[11px]"
                    >
                      {actionText || "我知道了"}
                    </Button>
                  )}
                </div>
              )}

              {/* 纯文本模式（text-only，1.1）— 无额外按钮，只有关闭 */}
              {!isMultiStep && contentVariant === "text-only" && (
                <div className="pb-2" />
              )}

              {/* 文本+图片模式下的按钮（1.3）— 仅当显式传入 actionText 时才显示 */}
              {!isMultiStep && contentVariant === "text-image" && actionText && (
                <div className="px-3 pb-3 flex justify-end">
                  {isDark ? (
                    <button
                      onClick={onAction || onClose}
                      className={`h-[28px] px-3 text-[11px] font-medium ${btnRadius} transition-colors inline-flex items-center ${btnBg}`}
                    >
                      {actionText}
                    </button>
                  ) : (
                    <Button
                      variant={endpoint === "admin" ? "claw-primary" : "tenant-primary"}
                      size="claw-sm"
                      onClick={onAction || onClose}
                      className="h-[28px] px-3 text-[11px]"
                    >
                      {actionText}
                    </Button>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
