/**
 * demoFloatingPanel - 右下角多个 Demo 浮层
 * (BillingStatusToggle / AdminModeFloatingToggle /
 *  OnboardingDemoPanel / ClsStatusToggle) 之间的展开状态协调桥。
 *
 * 这些浮层的展开面板都固定浮在视口右下角（fixed right-4 bottom-4，
 * 层级最高）。为了避免打开一个面板时其他折叠 header 从旁边露出来，
 * 用这个 module-scope 单例广播「当前展开的浮层 id」，其余浮层收到
 * 后隐藏各自的折叠触发按钮，只有当前展开的浮层保留 header（同时又被
 * 展开面板本身覆盖）——效果等同于「任一面板展开时，那一列 header 整体
 * 消失，屏幕右下角只剩下这个展开面板」。
 *
 * 使用：
 *   const active = useActiveDemoPanel();       // 订阅当前展开 id
 *   setActiveDemoPanel("billing" | "member" | "onboarding" | "cls" | null);
 */
import { useSyncExternalStore } from "react";

export type DemoPanelId = "billing" | "member" | "onboarding" | "cls";

type Listener = () => void;

let activeId: DemoPanelId | null = null;
const listeners = new Set<Listener>();

function emit() {
  listeners.forEach(l => l());
}

export function setActiveDemoPanel(id: DemoPanelId | null) {
  if (activeId === id) return;
  activeId = id;
  emit();
}

export function getActiveDemoPanel(): DemoPanelId | null {
  return activeId;
}

function subscribe(cb: Listener): () => void {
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
  };
}

/** React hook：读取「当前展开的 demo 浮层 id」，会在其变化时触发重渲染 */
export function useActiveDemoPanel(): DemoPanelId | null {
  return useSyncExternalStore(
    subscribe,
    getActiveDemoPanel,
    () => null // SSR fallback
  );
}
