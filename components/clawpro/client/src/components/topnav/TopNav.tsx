/**
 * TopNav - 用户端顶部导航壳
 *
 * 设计来源：Figma 公共组件/导航（节点 358:2322 / 297:3719），0522 修改点（1141:11612）
 * 视觉规范（v2 / 0522）：
 *   - 容器：高 64px，padding 12px 28px
 *   - 背景：rgba(255,255,255,0.4) + backdrop-blur-md（毛玻璃半透，让下方"流动蓝图"渐变和卡片若隐若现）
 *   - 底边：1px solid #E2E8F0
 *   - 布局：CSS Grid 三栏（左 Logo / 中 Tabs / 右功能区）
 *
 * 与管控端 AdminLayout 顶栏的差异：
 *   - 管控端使用左侧导航 + 不透明顶栏；用户端使用半透明顶栏，依赖 backdrop-blur 与下方背景产生层次
 *   - 仅供用户端 (Tenant) 使用，管控端勿引用本组件
 *
 * 适配规则：
 *   - 三栏 Grid：1fr auto 1fr，中间栏 justify-self:center 天然居中
 *   - 左右栏内容固定不压缩，中间栏居中且不会与两侧重叠
 *   - < 1200px 时 min-width 锁死，出横向滚动条
 */
import React from "react";
import { Link } from "wouter";

export interface TopNavProps {
  /** 中央 Tab 列表（可选；不传则不渲染中央区） */
  center?: React.ReactNode;
  /** 右侧功能区（图标按钮 / 用户菜单等） */
  right?: React.ReactNode;
  /** 左侧 Logo 点击跳转，默认 "/" */
  logoHref?: string;
  /** 自定义类名 */
  className?: string;
}

/**
 * 右侧图标按钮之间的竖向分隔线（Figma Vector 5/6/7）
 * 颜色 #E2E8F0，高 13.33px，宽 1px。
 */
export function NavDivider() {
  return (
    <span
      aria-hidden
      className="inline-block h-[14px] w-px bg-[#E2E8F0] flex-shrink-0"
    />
  );
}

export default function TopNav({
  center,
  right,
  logoHref = "/",
  className = "",
}: TopNavProps) {
  return (
    <header
      className={`sticky top-0 z-50 h-[64px] backdrop-blur-md ${className}`}
      style={{
        background: "rgba(255, 255, 255, 0.8)",
        borderBottom: "1px solid #E2E8F0",
        minWidth: "1200px",
      }}
    >
      {/* 三栏 Grid：左 1fr / 中 auto / 右 1fr — 中栏天然页面正中
          [Figma 1077-33929] 顶栏左右 padding = 28px（px-7） */}
      <div
        className="h-full grid items-center px-7 min-w-[1200px]"
        style={{
          gridTemplateColumns: "1fr auto 1fr",
          gap: "24px",
        }}
      >
        {/* 左栏：Logo 靠左
            标题与 Landing 页 Navbar 保持一致：brand-logo.png 28×28 + “ClawPro 智能体体验平台”
            （PingFang SC / 18px / 500 / #0A0A0A，间距 11px） */}
        <div className="justify-self-start min-w-0">
          <Link href={logoHref}>
            <div
              className="flex items-center cursor-pointer transition-opacity hover:opacity-90 whitespace-nowrap"
              style={{ gap: 11 }}
            >
              <img
                src="/landing-assets/yh-features/brand-logo.png"
                alt="ClawPro"
                width={28}
                height={28}
                draggable={false}
                className="select-none"
                style={{ objectFit: "contain", flexShrink: 0, borderRadius: 4 }}
              />
              <span
                className="select-none"
                style={{
                  fontSize: "18px",
                  fontFamily: "'PingFang SC', sans-serif",
                  fontWeight: 500,
                  color: "#0A0A0A",
                  lineHeight: 1,
                }}
              >
                ClawPro 智能体体验平台
              </span>
            </div>
          </Link>
        </div>

        {/* 中栏：Segmented Tabs — auto 宽度，天然居中 */}
        <div className="whitespace-nowrap">
          {center}
        </div>

        {/* 右栏：功能图标 + 用户菜单靠右，min-w-0 防止撑开列宽 */}
        <div className="justify-self-end flex items-center gap-3 whitespace-nowrap min-w-0">
          {right}
        </div>
      </div>
    </header>
  );
}
