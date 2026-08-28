/**
 * Onboarding Guide System - 新手引导体系
 *
 * 引导组件：
 * 1. GuideGlobalModal      - 影响面极大的更新：全局弹窗（强阻断）
 * 2. GuideModuleFloat      - 模块级更新：用户端非阻断浮窗
 * 3. GuideAdminNotify      - 模块级更新：管控端产品动态卡片（同端自动合并）
 * 4. GuideNavBubble        - 新功能预览：导航功能预览介绍气泡
 * 5. GuidePointBubble      - 功能级更新：点对点指引气泡
 * 6. GuideUpdateBar        - 全局更新：强提醒公告条
 * 7. GuideChangelogDrawer  - 更新记录：侧边抽屉
 * 8. GuideHighlightBubble  - 结构/UI 元素变化：高亮+步骤气泡（强阻断）
 * 9. GuideNewTag           - 轻量「新」标识（存续 ≤14 天）
 * 10. ProductUpdatesDrawer - 产品动态抽屉（高亮 3s 无底色自动消失）
 *
 * 共享基础设施：
 * + onboardingShared.ts    - 行为参数/埋点/持久化/气泡队列/文案校验/i18n/NewTag 时长
 * + onboardingHooks.ts     - useBubbleQueue / useFocusTrap / useExposure
 *
 * + OnboardingSimToggle    - 右下角模拟开关挂件
 * + OnboardingProvider     - 全局上下文
 */

// Types
export type {
  GuideComponentType,
  GuideEndpoint,
  SceneLayer,
  SceneCode,
  GuideStep,
  GuideAction,
  GuideFlow,
  GuideTrigger,
  GuideState,
  GuideActions,
} from "./types";

// Provider & Store
export { OnboardingProvider, useOnboarding } from "./OnboardingProvider";
export { useOnboardingStore } from "./useOnboardingStore";

// Components
export { GuideGlobalModal } from "./GuideGlobalModal";
export type { GlobalModalSlide, GlobalModalVariant, GlobalModalEndpoint } from "./GuideGlobalModal";

export { GuideOnboardingModal } from "./GuideOnboardingModal";
export type { OnboardingEndpoint, OnboardingModalVariant } from "./GuideOnboardingModal";

export { GuideModuleFloat } from "./GuideModuleFloat";
export type { ModuleFloatItem, ModuleFloatVariant } from "./GuideModuleFloat";

export { GuideAdminNotify, AdminNotifyCard, AdminNotifyStack, ADMIN_CARD_SKINS, buildProductUpdateNotices } from "./GuideAdminNotify";
export type { AdminNotifyItem, AdminNotifyVariant } from "./GuideAdminNotify";

export { GuideNewTag } from "./GuideNewTag";
export type { NewTagVariant } from "./GuideNewTag";

export { GuideNavBubble } from "./GuideNavBubble";

export { GuidePointBubble } from "./GuidePointBubble";
export type { PointBubbleVariant, PointBubbleContentVariant } from "./GuidePointBubble";

export { GuideUpdateBar } from "./GuideUpdateBar";

export { GuideChangelogDrawer } from "./GuideChangelogDrawer";
export type { ChangelogEntry, ChangelogVersion } from "./GuideChangelogDrawer";

export { ProductUpdatesDrawer } from "./ProductUpdatesDrawer";
export type { ProductUpdateItem, ProductUpdateType, ProductUpdateEndpoint } from "./ProductUpdatesDrawer";

export { GuideHighlightBubble } from "./GuideHighlightBubble";
export type { HighlightRegion } from "./GuideHighlightBubble";

// 共享基础设施（行为参数 / 埋点 / 持久化 / 气泡队列 / 文案校验 / i18n / New Tag 时长）
export {
  DEFAULT_BEHAVIOR,
  BEHAVIOR_PRESETS,
  resolveBehavior,
  trackOnboarding,
  buildPersistenceKey,
  isDismissed,
  markDismissed,
  markExposure,
  bubbleQueue,
  validateCopy,
  resolveI18n,
  isNewTagExpired,
  NEW_TAG_RECOMMENDED_DAYS,
  NEW_TAG_MAX_DAYS,
} from "./onboardingShared";
export type {
  BehaviorConfig,
  OnboardingAnalyticsEvent,
  OnboardingAnalyticsProps,
  I18nText,
} from "./onboardingShared";

// 共享 Hooks（气泡队列 / 焦点陷阱 / 曝光埋点）
export { useBubbleQueue, useFocusTrap, useExposure } from "./onboardingHooks";

// Widget
export { OnboardingSimToggle } from "./OnboardingSimToggle";

// Demo Panel (全局常驻)
export { OnboardingDemoPanel } from "./OnboardingDemoPanel";
