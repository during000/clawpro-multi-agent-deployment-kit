/**
 * AdminLeftMiniBarState - 管控端左下角 sidebar 内叠放的迷你条状态协调
 *
 * sidebar 底部同时可能出现两条迷你条：
 *   • 销毁倒计时迷你条（WarningExpireFloat 折叠态）
 *   • 账户欠费迷你条（AdminArrearsFloatCard 折叠态）
 *
 * 叠放规则（自底向上）：账户欠费在下、销毁倒计时在上。
 * 但如果账户欠费迷你条当前没有显示（比如没欠费、或欠费大卡片还没折叠、或 sidebar 收起），
 * 销毁倒计时迷你条就应该"下移到贴 footer 位置"，避免出现空档。
 *
 * 除此之外，**大卡片**（欠费大卡片 / 销毁倒计时大卡片）本身也占据 sidebar 内左下角同一坑位，
 * 它们贴 sidebar footer 上方 12px 展示。若下方还有对方那条迷你条正在显示（44px），
 * 大卡片就应额外向上让出一个迷你条高度，避免视觉上遮挡下方迷你条。
 *
 * 这个模块提供两组极小的单例订阅：
 *   • 欠费迷你条 可见状态：由 AdminArrearsFloatCard 上报，WarningExpireFloat / AdminArrearsFloatCard 大卡片订阅
 *   • 到期/停服迷你条 可见状态：由 WarningExpireFloat 上报，AdminArrearsFloatCard 大卡片订阅
 */
import { useSyncExternalStore } from "react";

// ── 欠费迷你条 可见状态 ─────────────────────────────────────────────
let arrearsMiniVisible = false;
const arrearsListeners = new Set<() => void>();

function emitArrears() {
  arrearsListeners.forEach((l) => l());
}

/** 由 AdminArrearsFloatCard 调用：上报自己「迷你条形态可见」状态 */
export function setArrearsMiniVisible(visible: boolean) {
  if (arrearsMiniVisible === visible) return;
  arrearsMiniVisible = visible;
  emitArrears();
}

function subscribeArrears(cb: () => void): () => void {
  arrearsListeners.add(cb);
  return () => {
    arrearsListeners.delete(cb);
  };
}

function getArrearsSnapshot(): boolean {
  return arrearsMiniVisible;
}

/**
 * 订阅「账户欠费迷你条是否正在 sidebar 底部显示」
 * 用于 WarningExpireFloat 决定销毁倒计时迷你条的 bottom：
 *   true  → 叠在欠费迷你条之上（+44px）
 *   false → 下移到贴 sidebar footer 位置
 * 同时也可被 WarningExpireFloat 大卡片订阅，避免大卡片压住欠费迷你条。
 */
export function useArrearsMiniVisible(): boolean {
  return useSyncExternalStore(
    subscribeArrears,
    getArrearsSnapshot,
    () => false, // SSR fallback
  );
}

// ── 到期/停服迷你条 可见状态 ─────────────────────────────────────────
let expireMiniVisible = false;
const expireListeners = new Set<() => void>();

function emitExpire() {
  expireListeners.forEach((l) => l());
}

/** 由 WarningExpireFloat 调用：上报自己「销毁倒计时迷你条形态可见」状态 */
export function setExpireMiniVisible(visible: boolean) {
  if (expireMiniVisible === visible) return;
  expireMiniVisible = visible;
  emitExpire();
}

function subscribeExpire(cb: () => void): () => void {
  expireListeners.add(cb);
  return () => {
    expireListeners.delete(cb);
  };
}

function getExpireSnapshot(): boolean {
  return expireMiniVisible;
}

/**
 * 订阅「到期/停服迷你条是否正在 sidebar 底部显示」
 * 用于 AdminArrearsFloatCard 大卡片决定 bottom：
 *   true  → 抬高一个迷你条高度（+44px），避免大卡片压住下方到期迷你条
 *   false → 常规位置，贴 sidebar footer 上方 12px
 */
export function useExpireMiniVisible(): boolean {
  return useSyncExternalStore(
    subscribeExpire,
    getExpireSnapshot,
    () => false, // SSR fallback
  );
}
