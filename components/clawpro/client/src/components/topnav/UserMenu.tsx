/**
 * UserMenu - 顶部导航右侧的用户菜单
 *
 * 设计来源：Figma 「我的资料」（节点 297:3564 / 297:3460）
 * 视觉规范（严格对齐 Figma）：
 *   - 容器：row、gap 8.89px、padding 4.44px 8.89px
 *   - 头像：31x31 圆形、bg #8CBCF7、首字母为大写字母（PingFang 600、字号 14、color #000）
 *   - 用户名：14 / line-height 22 / #020617，单行居中
 *
 * 交互：鼠标移入 200ms 延迟触发下拉面板，移出自动关闭。
 *
 * 用法：
 *   <UserMenu username="jingsujiang">
 *     <DropdownMenuItem>重置密码</DropdownMenuItem>
 *   </UserMenu>
 */
import React, { useRef, useState, useCallback } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export interface UserMenuProps {
  username: string;
  /** 头像背景色（默认 Figma 浅蓝 #8CBCF7） */
  avatarBg?: string;
  /** 头像字色（默认 #000） */
  avatarColor?: string;
  /** 头像中显示的首字母（默认取 username 首位） */
  avatarLetter?: string;
  /** 下拉菜单内容（自定义 DropdownMenuItem 等） */
  children?: React.ReactNode;
  /** 自定义类名 */
  className?: string;
}

/** hover 延迟打开 (ms) */
const OPEN_DELAY = 200;
/** hover 延迟关闭 (ms) */
const CLOSE_DELAY = 300;

export default function UserMenu({
  username,
  avatarBg = "#8CBCF7",
  avatarColor = "#000000",
  avatarLetter,
  children,
  className = "",
}: UserMenuProps) {
  const letter = (avatarLetter ?? username.charAt(0) ?? "?").toUpperCase();
  const [open, setOpen] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const handleMouseEnter = useCallback(() => {
    clearTimer();
    if (!open) {
      timerRef.current = setTimeout(() => setOpen(true), OPEN_DELAY);
    }
  }, [clearTimer, open]);

  const handleMouseLeave = useCallback(() => {
    clearTimer();
    timerRef.current = setTimeout(() => setOpen(false), CLOSE_DELAY);
  }, [clearTimer]);

  // 仅在明确的用户操作（Escape / 点击菜单项）时关闭，
  // 不响应 Radix 因焦点变化等自动触发的 close，避免与 hover 逻辑冲突
  const handleOpenChange = useCallback(
    (val: boolean) => {
      if (val) {
        // Radix 请求打开 — 通常由点击触发，直接允许
        clearTimer();
        setOpen(true);
      }
      // val === false 时，不立即关闭；关闭完全交由 handleMouseLeave 控制
      // 这样可防止 Radix 的 onPointerDownOutside / blur 导致闪烁
    },
    [clearTimer],
  );

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange} modal={false}>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
          className={[
            "group inline-flex items-center gap-[9px] rounded-[4px]",
            "px-[9px] py-[4px] transition-colors flex-shrink-0 nav-user-btn",
            className,
          ].join(" ")}
        >
          {/* 头像 */}
          <span
            className="inline-flex items-center justify-center flex-shrink-0"
            style={{
              width: 31,
              height: 31,
              borderRadius: "50%",
              background: avatarBg,
              color: avatarColor,
              fontFamily: "PingFang SC, sans-serif",
              fontWeight: 600,
              fontSize: 14,
              lineHeight: 1,
            }}
          >
            {letter}
          </span>
          {/* 用户名（溢出省略） */}
          <span className="text-[14px] leading-[22px] text-[#020617] group-hover:text-[#1447e6] truncate nav-btn-label" style={{ maxWidth: 120 }}>
            {username}
          </span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={2}
        className="w-56"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onCloseAutoFocus={(e) => e.preventDefault()}
      >
        {children}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
