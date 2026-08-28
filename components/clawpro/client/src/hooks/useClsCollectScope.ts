/**
 * useClsCollectScope - CLS 采集范围（全局 Hook）
 *
 * 语义：
 *   - CLS 是全局服务，采集范围也是全局配置（跨运维观测 / 会话管理等页面共享）。
 *   - 存储在 localStorage("clsCollectGroupIds")，JSON 数组（组织 id 的最小覆盖集）。
 *   - 未设置 / 空数组 → 采集全部实例（默认）。
 *   - 与 globalClsEnabled 联动：关闭 CLS 时应清空采集范围（外部调用 reset 完成）。
 *
 * 同步：
 *   - 多标签 / 多页面通过 "storage" 事件同步。
 *   - 同标签页内多组件通过自定义 "cls-collect-scope-change" 事件同步。
 */
import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "clsCollectGroupIds";
const CHANGE_EVENT = "cls-collect-scope-change";

function readFromStorage(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((x) => typeof x === "string") : [];
  } catch {
    return [];
  }
}

function writeToStorage(value: string[]) {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
  // 同标签页内其它组件同步
  window.dispatchEvent(new CustomEvent<string[]>(CHANGE_EVENT, { detail: value }));
}

export function useClsCollectScope() {
  const [scope, setScopeState] = useState<string[]>(() => readFromStorage());

  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY) setScopeState(readFromStorage());
    };
    const onCustom = (e: Event) => {
      const detail = (e as CustomEvent<string[]>).detail;
      if (Array.isArray(detail)) setScopeState(detail);
      else setScopeState(readFromStorage());
    };
    window.addEventListener("storage", onStorage);
    window.addEventListener(CHANGE_EVENT, onCustom as EventListener);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(CHANGE_EVENT, onCustom as EventListener);
    };
  }, []);

  const setScope = useCallback((next: string[]) => {
    writeToStorage(next);
    setScopeState(next);
  }, []);

  const resetScope = useCallback(() => {
    writeToStorage([]);
    setScopeState([]);
  }, []);

  return { scope, setScope, resetScope } as const;
}
