/**
 * onboardingHooks - 引导体系共享 React Hooks
 *
 *  - useBubbleQueue   气泡并发队列（最多 2 个），超出自动排队
 *  - useFocusTrap     强阻断组件焦点陷阱 + Esc 关闭 + body 滚动锁
 *  - useExposure      自动上报曝光埋点 + 记录曝光次数
 *
 * 详见 docs/引导组件规范汇总.md「七、规范优化建议」第 7、8 条。
 */
import { useEffect, useRef, useState, useCallback } from "react";
import {
  bubbleQueue,
  trackOnboarding,
  markExposure,
  type OnboardingAnalyticsProps,
} from "./onboardingShared";

// ─── 气泡并发队列 ──────────────────────────────────────────────

/**
 * 申请在全局气泡队列中占位。返回 canShow：
 *  - true  当前可展示（并发数 < 2 或已在展示中）
 *  - false 已进入排队，需等待其他气泡关闭后补位
 *
 * 组件应在 canShow=false 时不渲染气泡本体（仅占位），
 * 在 open 变 false 或卸载时自动 release。
 */
export function useBubbleQueue(id: string, open: boolean): boolean {
  const [canShow, setCanShow] = useState(false);

  useEffect(() => {
    if (!open) {
      bubbleQueue.release(id);
      setCanShow(false);
      return;
    }
    const unsub = bubbleQueue.subscribe(() => {
      setCanShow(bubbleQueue.isVisible(id));
    });
    setCanShow(bubbleQueue.request(id));
    return () => {
      unsub();
      bubbleQueue.release(id);
    };
  }, [id, open]);

  return canShow;
}

// ─── 强阻断焦点陷阱 ────────────────────────────────────────────

const FOCUSABLE =
  'a[href],button:not([disabled]),textarea,input,select,[tabindex]:not([tabindex="-1"])';

/**
 * 强阻断组件（GlobalModal / HighlightBubble）的无障碍辅助：
 *  - 打开时锁定 body 滚动
 *  - 将焦点移入容器，Tab 在容器内循环（focus trap）
 *  - Esc 触发 onClose（dismissible 时）
 *  - 关闭后焦点归还触发元素
 */
export function useFocusTrap(
  open: boolean,
  onClose: () => void,
  opts: { dismissible?: boolean } = {}
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const prevFocus = useRef<HTMLElement | null>(null);
  const { dismissible = true } = opts;

  useEffect(() => {
    if (!open) return;
    prevFocus.current = document.activeElement as HTMLElement;

    // 锁定 body 滚动
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    // 焦点移入
    const focusFirst = () => {
      const el = containerRef.current;
      if (!el) return;
      const focusables = el.querySelectorAll<HTMLElement>(FOCUSABLE);
      (focusables[0] ?? el).focus?.();
    };
    const raf = requestAnimationFrame(focusFirst);

    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && dismissible) {
        e.stopPropagation();
        onClose();
        return;
      }
      if (e.key !== "Tab") return;
      const el = containerRef.current;
      if (!el) return;
      const focusables = Array.from(el.querySelectorAll<HTMLElement>(FOCUSABLE));
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", handleKey, true);

    return () => {
      cancelAnimationFrame(raf);
      document.removeEventListener("keydown", handleKey, true);
      document.body.style.overflow = prevOverflow;
      prevFocus.current?.focus?.();
    };
  }, [open, onClose, dismissible]);

  return containerRef;
}

// ─── 曝光埋点 ──────────────────────────────────────────────────

/** open 变 true 时上报一次曝光埋点并累加曝光次数 */
export function useExposure(
  open: boolean,
  analytics: OnboardingAnalyticsProps,
  persistenceKey?: string
) {
  const reported = useRef(false);
  // analytics 对象每次渲染都是新引用，用 JSON 串做依赖稳定项
  const depKey = JSON.stringify(analytics);

  const report = useCallback(() => {
    trackOnboarding("onboarding_impression", analytics);
    if (persistenceKey) markExposure(persistenceKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [depKey, persistenceKey]);

  useEffect(() => {
    if (open && !reported.current) {
      reported.current = true;
      report();
    }
    if (!open) reported.current = false;
  }, [open, report]);
}
