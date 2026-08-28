/**
 * PopupQueueContext - 全局弹窗优先级队列
 *
 * 解决多个"进入页面自动弹出"的弹窗互相叠加的问题。
 * 各弹窗通过 usePopupSlot 注册自己的优先级，同一时刻只渲染优先级最高的那个，
 * 当它被关闭后，队列里下一个优先级最高的弹窗会自动放行。
 *
 * 约定：priority 数字越小越先弹出。
 * 当前业务顺序（见 POPUP_PRIORITY）：
 *   1. 全站视觉焕新升级（OnboardingGuide）
 *   2. 欠费 / 停服（ServiceSuspendedModal）
 */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

interface PopupQueueContextValue {
  register: (id: string, priority: number) => void;
  unregister: (id: string) => void;
  /** 当前应展示的弹窗 id（已注册中优先级最高者） */
  topId: string | null;
}

const PopupQueueContext = createContext<PopupQueueContextValue | null>(null);

/**
 * 弹窗优先级常量：数字越小越先弹出。
 * 同一时刻只展示其中优先级最高（数字最小）的弹窗，关闭后依次放行。
 */
export const POPUP_PRIORITY = {
  /** 全站视觉焕新升级（最先弹出） */
  ONBOARDING_GUIDE: 10,
  /** 欠费 / 停服提醒（最后弹出） */
  SERVICE_SUSPENDED: 20,
} as const;

export function PopupQueueProvider({ children }: { children: ReactNode }) {
  // key=弹窗 id，value=优先级
  const [registry, setRegistry] = useState<Record<string, number>>({});

  const register = useCallback((id: string, priority: number) => {
    setRegistry((prev) => {
      if (prev[id] === priority) return prev;
      return { ...prev, [id]: priority };
    });
  }, []);

  const unregister = useCallback((id: string) => {
    setRegistry((prev) => {
      if (!(id in prev)) return prev;
      const next = { ...prev };
      delete next[id];
      return next;
    });
  }, []);

  const topId = useMemo(() => {
    let top: string | null = null;
    let topPriority = Number.POSITIVE_INFINITY;
    for (const [id, priority] of Object.entries(registry)) {
      if (priority < topPriority) {
        topPriority = priority;
        top = id;
      }
    }
    return top;
  }, [registry]);

  const value = useMemo<PopupQueueContextValue>(
    () => ({ register, unregister, topId }),
    [register, unregister, topId],
  );

  return (
    <PopupQueueContext.Provider value={value}>
      {children}
    </PopupQueueContext.Provider>
  );
}

/**
 * 弹窗排队 Hook。
 *
 * @param id        弹窗唯一标识
 * @param priority  优先级，数字越小越先展示（见 POPUP_PRIORITY）
 * @param active    该弹窗自身条件是否已满足（即"想要展示"）
 * @returns canShow 是否轮到该弹窗展示——只有当它是当前所有 active 弹窗中
 *                  优先级最高者时才为 true。
 *
 * 用法：把组件原本的"是否显示"条件传入 active，再用返回值控制实际渲染：
 *   const canShow = usePopupSlot("xxx", POPUP_PRIORITY.XXX, wantShow);
 *   if (!canShow) return null;            // 或 <Dialog open={canShow} />
 *
 * 兼容性：若未包裹 PopupQueueProvider，则退化为直接返回 active（不参与排队）。
 */
export function usePopupSlot(
  id: string,
  priority: number,
  active: boolean,
): boolean {
  const ctx = useContext(PopupQueueContext);
  const register = ctx?.register;
  const unregister = ctx?.unregister;

  useEffect(() => {
    if (register && unregister && active) {
      register(id, priority);
      return () => unregister(id);
    }
    return undefined;
  }, [register, unregister, active, id, priority]);

  if (!ctx) return active;
  return active && ctx.topId === id;
}
