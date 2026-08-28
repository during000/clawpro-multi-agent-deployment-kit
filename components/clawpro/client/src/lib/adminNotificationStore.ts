/**
 * adminNotificationStore - 管控端额外的消息通知（运行时动态推送）
 *
 * 用法：
 *   - 业务侧调用 `pushAdminNotification(notif)` 推送一条新通知（自动 unshift 到最前）
 *   - `useAdminNotifications()` 在 React 组件内订阅最新列表
 *   - 数据存储在 localStorage("admin_notifications_extra")，跨标签页通过 storage 事件同步
 *   - 同一 dedupeKey 在 24h 内不会重复推送，避免用户反复点击编辑触发同一条校验失败时
 *     通知列表里堆叠出多条一样的消息
 *
 * 与 MOCK_NOTIFICATIONS 的关系：
 *   - 本 store 仅维护"运行时新增"的通知；MOCK_NOTIFICATIONS 留在 TenantLayout 内继续作为静态种子
 *   - TenantLayout 把 store 的列表合并到 MOCK 前面，再交给 NotificationPanel 展示
 */
import { useEffect, useState } from "react";
import type { Notification } from "@/components/topnav";

const STORAGE_KEY = "admin_notifications_extra";
const CHANGE_EVENT = "admin-notifications-change";
const DEDUPE_WINDOW_MS = 24 * 60 * 60 * 1000; // 24 小时

interface StoredNotification extends Notification {
  /** 用于去重的业务键；同 key 在 DEDUPE_WINDOW_MS 内不再重复推送 */
  dedupeKey?: string;
  /** 推送时间戳（毫秒）— 用于去重窗口判断 */
  pushedAt?: number;
}

function readFromStorage(): StoredNotification[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((x) => x && typeof x.id === "string" && typeof x.message === "string");
  } catch {
    return [];
  }
}

function writeToStorage(list: StoredNotification[]) {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
  window.dispatchEvent(new CustomEvent(CHANGE_EVENT));
}

function formatNow(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/**
 * 推送一条管控端通知
 * @returns 是否真正写入（被去重时返回 false）
 */
export function pushAdminNotification(
  input: Omit<Notification, "id" | "timestamp" | "read"> & {
    id?: string;
    dedupeKey?: string;
  },
): boolean {
  const now = Date.now();
  const current = readFromStorage();

  // 去重：同 dedupeKey 且在窗口内已存在，则不重复推送
  if (input.dedupeKey) {
    const existed = current.find(
      (n) => n.dedupeKey === input.dedupeKey && now - (n.pushedAt ?? 0) < DEDUPE_WINDOW_MS,
    );
    if (existed) return false;
  }

  const notif: StoredNotification = {
    id: input.id ?? `admin-${now}-${Math.random().toString(36).slice(2, 8)}`,
    message: input.message,
    category: input.category,
    actionHref: input.actionHref,
    actionLabel: input.actionLabel,
    timestamp: formatNow(),
    read: false,
    dedupeKey: input.dedupeKey,
    pushedAt: now,
  };

  writeToStorage([notif, ...current]);
  return true;
}

/** 清空所有运行时推送的通知 */
export function clearAdminNotifications() {
  writeToStorage([]);
}

/** React Hook：订阅最新列表，自动跨标签页 / 同标签页同步 */
export function useAdminNotifications(): Notification[] {
  const [list, setList] = useState<StoredNotification[]>(() => readFromStorage());

  useEffect(() => {
    const sync = () => setList(readFromStorage());
    const onStorage = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY) sync();
    };
    window.addEventListener("storage", onStorage);
    window.addEventListener(CHANGE_EVENT, sync);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(CHANGE_EVENT, sync);
    };
  }, []);

  // 对外只暴露标准 Notification 字段（隐藏 dedupeKey / pushedAt）
  return list.map(({ dedupeKey: _d, pushedAt: _p, ...rest }) => rest);
}
