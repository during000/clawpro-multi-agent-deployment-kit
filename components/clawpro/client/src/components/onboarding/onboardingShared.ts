/**
 * onboardingShared - 引导体系共享基础设施
 *
 * 统一收敛规范优化建议中的横切关注点，供所有引导组件复用：
 *  1. 行为参数（behavior）token 化         —— BehaviorConfig / DEFAULT_BEHAVIOR / resolveBehavior
 *  2. 埋点规范                             —— trackOnboarding / OnboardingAnalyticsEvent
 *  3. 持久化 key 规范 + 读写              —— buildPersistenceKey / isDismissed / markDismissed
 *  4. 全局气泡队列（最多 2 个并发）       —— bubbleQueue
 *  5. 文案长度软校验                      —— validateCopy
 *  6. 国际化 key 预留                     —— resolveI18n / I18nText
 *  7. New Tag 存续时间约束                —— isNewTagExpired / NEW_TAG_MAX_DAYS
 *
 * 详见 docs/引导组件规范汇总.md「七、规范优化建议」。
 */

// ─── 1. 行为参数 token 化 ──────────────────────────────────────

export interface BehaviorConfig {
  /** 是否可手动关闭 */
  dismissible: boolean;
  /** 是否仅展示一次 */
  showOnce: boolean;
  /** 最大曝光次数（0 = 不限制） */
  maxExposures: number;
  /** 关闭后冷却天数（0 = 永久不再展示） */
  cooldownDays: number;
  /** 生效起始时间（ISO，null = 立即） */
  startsAt: string | null;
  /** 失效时间（ISO，null = 永不过期） */
  expiresAt: string | null;
}

export const DEFAULT_BEHAVIOR: BehaviorConfig = {
  dismissible: true,
  showOnce: true,
  maxExposures: 1,
  cooldownDays: 0,
  startsAt: null,
  expiresAt: null,
};

/** 各组件类型的默认行为预设（业务侧可覆盖） */
export const BEHAVIOR_PRESETS: Record<string, Partial<BehaviorConfig>> = {
  // 强阻断弹窗：强制阅读，仅一次
  "global-modal": { dismissible: true, showOnce: true, maxExposures: 1 },
  // 非阻断浮窗：可关闭，展示一次
  "module-float": { dismissible: true, showOnce: true, maxExposures: 1 },
  // 单 UI 气泡：曝光 2 次后不再展示
  "point-bubble": { dismissible: true, showOnce: false, maxExposures: 2 },
  // 步骤指引：仅一次
  "highlight-bubble": { dismissible: true, showOnce: true, maxExposures: 1 },
  // 导航气泡：曝光 2 次或点击后移除
  "nav-bubble": { dismissible: true, showOnce: false, maxExposures: 2 },
  // 强提醒公告条：不可手动关闭（强提醒）
  "update-bar": { dismissible: false, showOnce: false, maxExposures: 0 },
  // New Tag：展示 14 天或首次点击后移除
  "new_tag": { dismissible: false, showOnce: false, maxExposures: 0, cooldownDays: 14 },
};

export function resolveBehavior(
  type: string,
  override?: Partial<BehaviorConfig>
): BehaviorConfig {
  return { ...DEFAULT_BEHAVIOR, ...(BEHAVIOR_PRESETS[type] ?? {}), ...(override ?? {}) };
}

// ─── 2. 埋点规范 ───────────────────────────────────────────────

export type OnboardingAnalyticsEvent =
  | "onboarding_impression"
  | "onboarding_click"
  | "onboarding_dismiss";

export interface OnboardingAnalyticsProps {
  /** 更新 id（与 persistenceKey 对应） */
  updateId?: string;
  /** 组件类型 id */
  component: string;
  /** 场景层级 */
  layer?: string;
  /** 场景编号 */
  scenario?: string;
  /** 端类型 */
  endpoint?: "admin" | "tenant";
  [k: string]: unknown;
}

/**
 * 统一埋点上报入口。优先复用产品已有埋点 SDK（window.__track），
 * 缺失时降级为 console.debug，保证不报错且可追踪。
 */
export function trackOnboarding(
  event: OnboardingAnalyticsEvent,
  props: OnboardingAnalyticsProps
): void {
  if (typeof window === "undefined") return;
  const w = window as unknown as { __track?: (e: string, p: Record<string, unknown>) => void };
  try {
    if (typeof w.__track === "function") {
      w.__track(event, props);
    } else if (import.meta.env?.DEV) {
      // eslint-disable-next-line no-console
      console.debug(`[onboarding] ${event}`, props);
    }
  } catch {
    /* 埋点失败不应影响 UI */
  }
}

// ─── 3. 持久化 key 规范 + 读写 ─────────────────────────────────

/** 统一持久化 key 格式：onboarding.{component}.{updateId}.dismissed */
export function buildPersistenceKey(component: string, updateId: string): string {
  return `onboarding.${component}.${updateId}.dismissed`;
}

interface DismissRecord {
  dismissedAt: string;
  exposures: number;
}

function readRecord(key: string): DismissRecord | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as DismissRecord) : null;
  } catch {
    return null;
  }
}

function writeRecord(key: string, rec: DismissRecord): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(key, JSON.stringify(rec));
  } catch {
    /* 忽略写入失败 */
  }
}

/** 是否应隐藏（已关闭且在冷却期内，或超过最大曝光次数） */
export function isDismissed(key: string, behavior: BehaviorConfig): boolean {
  const rec = readRecord(key);
  if (!rec) return false;
  // 超过最大曝光次数
  if (behavior.maxExposures > 0 && rec.exposures >= behavior.maxExposures) return true;
  // 冷却期判断
  if (behavior.showOnce && behavior.cooldownDays === 0) return true;
  if (behavior.cooldownDays > 0) {
    const elapsed = (Date.now() - new Date(rec.dismissedAt).getTime()) / 86400000;
    return elapsed < behavior.cooldownDays;
  }
  return false;
}

/** 记录一次关闭 */
export function markDismissed(key: string): void {
  const rec = readRecord(key) ?? { dismissedAt: "", exposures: 0 };
  writeRecord(key, { dismissedAt: new Date().toISOString(), exposures: rec.exposures });
}

/** 记录一次曝光 */
export function markExposure(key: string): void {
  const rec = readRecord(key) ?? { dismissedAt: "", exposures: 0 };
  writeRecord(key, { ...rec, exposures: rec.exposures + 1 });
}

// ─── 4. 全局气泡队列（最多 2 个并发） ──────────────────────────

const MAX_CONCURRENT_BUBBLES = 2;

type QueueListener = (visibleIds: string[]) => void;

class BubbleQueue {
  private waiting: string[] = [];
  private visible: string[] = [];
  private listeners = new Set<QueueListener>();

  /** 请求展示一个气泡，返回是否可立即展示（false 则进入排队） */
  request(id: string): boolean {
    if (this.visible.includes(id)) return true;
    if (this.visible.length < MAX_CONCURRENT_BUBBLES) {
      this.visible.push(id);
      this.emit();
      return true;
    }
    if (!this.waiting.includes(id)) this.waiting.push(id);
    return false;
  }

  /** 释放一个气泡，自动从队列补位 */
  release(id: string): void {
    this.visible = this.visible.filter((x) => x !== id);
    this.waiting = this.waiting.filter((x) => x !== id);
    while (this.visible.length < MAX_CONCURRENT_BUBBLES && this.waiting.length > 0) {
      const next = this.waiting.shift()!;
      this.visible.push(next);
    }
    this.emit();
  }

  isVisible(id: string): boolean {
    return this.visible.includes(id);
  }

  subscribe(fn: QueueListener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(): void {
    const snapshot = [...this.visible];
    this.listeners.forEach((fn) => fn(snapshot));
  }
}

/** 全局唯一气泡队列实例 */
export const bubbleQueue = new BubbleQueue();

// ─── 5. 文案长度软校验 ─────────────────────────────────────────

const COPY_LIMITS = {
  title: 14,
  body: 60, // 1-2 句短句
  cta: 6,
};

/** 文案超长时 console.warn（仅开发环境），防止 UI 溢出 */
export function validateCopy(
  field: keyof typeof COPY_LIMITS,
  text?: string,
  context?: string
): void {
  if (!text || typeof window === "undefined" || !import.meta.env?.DEV) return;
  const limit = COPY_LIMITS[field];
  if (text.length > limit) {
    // eslint-disable-next-line no-console
    console.warn(
      `[onboarding] 文案超长：${context ?? ""} ${field}（${text.length}/${limit} 字）："${text}"`
    );
  }
}

// ─── 6. 国际化 key 预留 ────────────────────────────────────────

/** 支持直接传中文，或传 { key, default } 形式预留 i18n */
export type I18nText = string | { key: string; default: string };

/** 解析 I18nText：未来接入 i18n 时在此统一替换为 t(key) */
export function resolveI18n(text?: I18nText): string {
  if (text == null) return "";
  if (typeof text === "string") return text;
  // TODO: 接入 i18n 后改为 i18n.t(text.key, text.default)
  return text.default;
}

// ─── 7. New Tag 存续时间约束 ───────────────────────────────────

/** New Tag 建议存续 14 天，最长 30 天 */
export const NEW_TAG_RECOMMENDED_DAYS = 14;
export const NEW_TAG_MAX_DAYS = 30;

/**
 * 判断 New Tag 是否已过期。
 * @param startDate 上线日期（ISO）
 * @param maxDays   下线天数（默认 14，最长强制 clamp 到 30）
 */
export function isNewTagExpired(startDate?: string, maxDays = NEW_TAG_RECOMMENDED_DAYS): boolean {
  if (!startDate) return false;
  const days = Math.min(maxDays, NEW_TAG_MAX_DAYS);
  const elapsed = (Date.now() - new Date(startDate).getTime()) / 86400000;
  return elapsed > days;
}
