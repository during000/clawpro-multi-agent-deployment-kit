// =============================================================================
// ClawPro 资源库 · canonical 资源统一入口（自动生成，请勿手改）
// -----------------------------------------------------------------------------
// 由 client/src/design-assets/scripts/build-canonical-assets.mjs 据阶段 2~5 真实
// 审计/治理产出生成（建设计划 §9 阶段 6 / §5.3）。改口径请改脚本后重跑。
//
// 用途与边界（务必遵守）：
//   - 仅供「当前项目页面层 / 页面级非组件代码」使用，提供「一改多处生效」能力：
//     修改此处某 key 的路径，所有 import 该 key 的页面处统一生效。
//   - 禁止在「共享组件源码」中 import 本文件（组件需保持可移植，用 props/lucide/宿主注入）。
//   - 禁止在「开发仓库 / 跨仓页面」中引用本文件或当前项目 /assets 路径。
//   - 入口仅收已确认 normal、业务专属、运行时可服务资源；不含 needs-review 资源。
//   - brand-fixed / avatar-like 资源禁止当普通 UI 图标改色（品牌/渠道/头像红线）。
//
// 生成时间：2026-07-20T04:15:15.417Z
// =============================================================================

export const canonicalAssets = {
  /** brands（brand-logo） */
  brands: {
    /** clawpro-logo */
    clawproLogo: "/assets/admin-sidebar/clawpro-logo.svg",
  },
  /** channels（channel-icon） */
  channels: {
    /** channel-wechat */
    wechat: "/assets/admin-channel-icons/channel-wechat.svg",
    /** channel-qq */
    qq: "/assets/admin-channel-icons/channel-qq.svg",
    /** channel-wecom，重复组 dup-001 */
    wecom: "/assets/admin-channel-icons/channel-wecom.svg",
    /** channel-wecom-app，重复组 dup-001 */
    wecomApp: "/assets/admin-channel-icons/channel-wecom-app.svg",
    /** channel-dingtalk */
    dingtalk: "/assets/admin-channel-icons/channel-dingtalk.svg",
    /** channel-feishu */
    feishu: "/assets/admin-channel-icons/channel-feishu.svg",
  },
  /** avatars（agent-avatar） */
  avatars: {
    /** avatar-default */
    default: "/assets/avatars/avatar-default.png",
    /** avatar-designer */
    designer: "/assets/avatars/avatar-designer.png",
    /** avatar-analyst */
    analyst: "/assets/avatars/avatar-analyst.png",
    /** avatar-creator */
    creator: "/assets/avatars/avatar-creator.png",
    /** avatar-developer */
    developer: "/assets/avatars/avatar-developer.png",
    /** avatar-pm */
    pm: "/assets/avatars/avatar-pm.png",
    /** avatar-operator */
    operator: "/assets/avatars/avatar-operator.png",
  },
} as const;

export type CanonicalAssetGroup = keyof typeof canonicalAssets;
