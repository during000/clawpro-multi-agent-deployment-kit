/**
 * NavIcons - 顶部导航用的内联 SVG 图标集合
 *
 * 设计来源：Figma 公共组件/导航的图标素材（节点 297:3274/3275、363:5053、I297:3564;297:3447）
 *
 * - 所有 stroke / fill 均使用 currentColor，由父级 color 控制变色，便于 hover 联动
 * - 默认尺寸 16x16；可通过 size 覆盖
 */
import React from "react";

export interface IconProps extends React.SVGProps<SVGSVGElement> {
  size?: number | string;
}

/** 使用指南（书本/手册） — Figma 297:3274（viewBox 16×16，1:1 与设计稿一致） */
export function HelpIcon({ size = 16, ...rest }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...rest}
    >
      <path
        d="M2.00073 12.6667C2.00073 11.378 3.00809 10.3333 4.25073 10.3333H14.0007V15H4.00073C2.89616 15 2.00073 14.1046 2.00073 13V4.11111V3C2.00073 1.89543 2.89616 1 4.00073 1H14.0007V4.11111V15M14.0007 15H4.25073C3.00809 15 2.00073 13.9553 2.00073 12.6667M7.25073 4.11111H11.0007"
        stroke="currentColor"
        strokeOpacity="0.9"
        strokeWidth="1.5"
        strokeLinecap="square"
      />
    </svg>
  );
}

/** 消息通知（铃铛） — Figma 297:3275（viewBox 16×16，1:1 与设计稿一致） */
export function BellIcon({ size = 16, ...rest }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...rest}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M13.5997 6.93337C13.5997 4.06617 11.4449 1.70224 8.66634 1.37264V0H7.33301V1.37264C4.55447 1.70224 2.39967 4.06616 2.39967 6.93337V10.4329L1.33301 11.7662V13.6667H5.35364C5.51767 14.9821 6.6398 16 7.99967 16C9.35955 16 10.4817 14.9821 10.6457 13.6667H14.6663V11.7662L13.5997 10.4329V6.93337ZM9.291 13.6667H6.70835C6.85636 14.2418 7.3784 14.6667 7.99967 14.6667C8.62095 14.6667 9.14299 14.2418 9.291 13.6667ZM7.99967 2.66671C5.64326 2.66671 3.73301 4.57696 3.73301 6.93337V10.9006L2.66634 12.2339V12.3334H13.333V12.2339L12.2663 10.9006V6.93337C12.2663 4.57696 10.3561 2.66671 7.99967 2.66671Z"
        fill="currentColor"
      />
    </svg>
  );
}

/** 切换管控端（显示器 + 齿轮 + 切换箭头） — Figma 1077:34989 / 363:5053（更新版）
 *  设计语义：显示器底部留出"切换"箭头，右下角小齿轮表示"管理/控制"含义。
 *  viewBox 16×16；stroke 用 currentColor，沿用父级 hover 联动。 */
export function SwitchAdminIcon({ size = 16, ...rest }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...rest}
    >
      <path
        d="M3.66667 14H8M7 12H1V2H14.3333V6.33333M12.6666 9.00002V8.16669M12.6666 9.00002C11.5621 9.00002 10.6666 9.89542 10.6666 11C10.6666 12.1046 11.5621 13 12.6666 13M12.6666 9.00002C13.7712 9.00002 14.6666 9.89542 14.6666 11C14.6666 12.1046 13.7712 13 12.6666 13M12.6666 13V13.8334M10.9346 10L10.2129 9.58335M14.3987 12L15.1204 12.4167M14.3987 10L15.1204 9.58335M10.9346 12L10.2129 12.4167"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="square"
      />
    </svg>
  );
}

/** 用户菜单下拉箭头 — Figma I297:3564;297:3447 */
export function ChevronDownIcon({ size = 14, ...rest }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 14 14"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...rest}
    >
      <path
        d="M3.50003 5.24998L7.00002 8.74997L10.5 5.24998"
        stroke="currentColor"
        strokeWidth="1.11111"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
