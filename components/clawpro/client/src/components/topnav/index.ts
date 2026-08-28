/**
 * 顶部导航组件包 - 对照 Figma「公共组件/导航」（节点 358:2322 / 297:3719）实现
 *
 * 推荐用法：
 *
 *   import {
 *     TopNav, NavDivider,
 *     CenterTabs,
 *     NavIconButton, HelpIcon, BellIcon, SwitchAdminIcon,
 *     NotificationPanel,
 *     UserMenu,
 *   } from "@/components/topnav";
 *
 *   <TopNav
 *     center={
 *       <CenterTabs items={...} activeValue={location} onChange={...} />
 *     }
 *     right={
 *       <>
 *         <NavIconButton icon={<HelpIcon />} title="使用指南" />
 *         <NavDivider />
 *         <NotificationPanel notifications={...} />
 *         <NavDivider />
 *         <NavIconButton icon={<SwitchAdminIcon />} label="切换管控端" />
 *         <NavDivider />
 *         <UserMenu username="jingsujiang">{...}</UserMenu>
 *       </>
 *     }
 *   />
 */
export { default as TopNav, NavDivider } from "./TopNav";
export type { TopNavProps } from "./TopNav";

export { default as CenterTabs } from "./CenterTabs";
export type { CenterTabItem, CenterTabsProps } from "./CenterTabs";

export { default as NavIconButton } from "./NavIconButton";
export type { NavIconButtonProps } from "./NavIconButton";

export {
  HelpIcon,
  BellIcon,
  SwitchAdminIcon,
  ChevronDownIcon,
} from "./NavIcons";
export type { IconProps } from "./NavIcons";

export { default as NotificationPanel } from "./NotificationPanel";
export type {
  Notification,
  NotificationCategory,
  NotificationPanelProps,
} from "./NotificationPanel";

export { default as HelpPanel } from "./HelpPanel";

export { default as UserMenu } from "./UserMenu";
export type { UserMenuProps } from "./UserMenu";
