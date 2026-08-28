/**
 * useDeferredSidebarCollapsed - 与 sidebar 宽度动画节奏对齐的"延迟收起"状态
 *
 * 背景：
 *   sidebar 的宽度切换是 `transition-[width] duration-300`（见ui/admin-sidebar.tsx）。
 *   sidebar 内 / 底部叠放的各种"告警条 / 迷你条"若直接跟随 useAdminSidebar().collapsed 判断挂载，
 *   会导致：collapsed → expanded 瞬间 React 立即 rerender、告警条先于 sidebar 宽度过渡完成就冒出来，
 *   造成「告警条比侧边栏先展开」的视觉倒错。
 *
 * 语义：
 *   - collapsed → expanded（false）：延迟 SIDEBAR_TRANSITION_MS 后再解锁，让 sidebar 宽度动画走完再显现
 *   - expanded → collapsed（true）：立即同步为 true，让内部条第一时间隐藏，避免溢出到 64px 收起态外
 *   - 卸载/快速来回切时通过 cleanup 清timer，避免竞态
 *
 * 使用方：所有挂在 sidebar 内或依赖 sidebar 宽度定位的"告警条"组件都应用此 hook 代替原始collapsed，
 *以保证同一次展开/收起动作中，所有告警条节奏一致（要么一起出、要么一起收）。
 */
import { useEffect, useState } from "react";
import { useAdminSidebar } from "@/components/ui/admin-sidebar";

/** 与 ui/admin-sidebar.tsx 的 `transition-[width] duration-300` 保持一致 */
export const SIDEBAR_TRANSITION_MS = 300;

export function useDeferredSidebarCollapsed(): boolean {
  const { collapsed } = useAdminSidebar();
  const [deferred, setDeferred] = useState(collapsed);

  useEffect(() => {
    if (collapsed) {
      // 收起：立即隐藏
      setDeferred(true);
      return;
    }
    // 展开：等 sidebar 宽度动画结束再让内部条挂载
    const t = window.setTimeout(() => setDeferred(false), SIDEBAR_TRANSITION_MS);
    return () => window.clearTimeout(t);
  }, [collapsed]);

  return deferred;
}
