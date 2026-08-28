/**
 * ClawPro 平台 MCP 的 localStorage 持久化 store
 *
 * 与项目其他模块（如 openclawStore、modelConfigStore）保持一致的范式：
 *   - 同步读写 localStorage
 *   - 通过 window 事件广播变更，组件用 useEffect 监听刷新
 *   - 解析失败时回退到默认 mock 数据
 */

import {
  CAPABILITIES,
  DEFAULT_AUTO_INSTALL_POLICY,
  DEFAULT_CAPABILITY_TOGGLES,
  INITIAL_DISTRIBUTED_AGENTS,
  INITIAL_USER_TOKENS,
} from './mockData';
import type {
  AutoInstallPolicy,
  CapabilityToggles,
  DistributedAgent,
  UserToken,
} from './types';

// ────────────────────────────────────────────────────────────
// Storage keys & version
// ────────────────────────────────────────────────────────────

const STORAGE_VERSION = '12';
const KEY_VERSION = 'clawpro_platform_mcp_version';

const KEY_TOGGLES = 'clawpro_platform_mcp_capability_toggles';
const KEY_TOKENS = 'clawpro_platform_mcp_user_tokens';
const KEY_DISTRIBUTED = 'clawpro_platform_mcp_distributed_agents';
const KEY_POLICY = 'clawpro_platform_mcp_auto_install_policy';

const EVENT_NAME = 'clawpro-platform-mcp-updated';

// ────────────────────────────────────────────────────────────
// 内部工具
// ────────────────────────────────────────────────────────────

function isBrowser(): boolean {
  return typeof window !== 'undefined' && typeof localStorage !== 'undefined';
}

function ensureVersion(): void {
  if (!isBrowser()) return;
  const v = localStorage.getItem(KEY_VERSION);
  if (v !== STORAGE_VERSION) {
    // 版本不一致，清空旧数据（mock 阶段简单粗暴；真实接入时改为迁移）
    localStorage.removeItem(KEY_TOGGLES);
    localStorage.removeItem(KEY_TOKENS);
    localStorage.removeItem(KEY_DISTRIBUTED);
    localStorage.removeItem(KEY_POLICY);
    localStorage.setItem(KEY_VERSION, STORAGE_VERSION);
  }
}

function safeParse<T>(raw: string | null, fallback: T): T {
  if (!raw) return fallback;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function emitChange(): void {
  if (!isBrowser()) return;
  window.dispatchEvent(new Event(EVENT_NAME));
}

// ────────────────────────────────────────────────────────────
// 公开 API：能力开关
// ────────────────────────────────────────────────────────────

export function loadCapabilityToggles(): CapabilityToggles {
  ensureVersion();
  if (!isBrowser()) return DEFAULT_CAPABILITY_TOGGLES;
  const stored = safeParse<CapabilityToggles | null>(
    localStorage.getItem(KEY_TOGGLES),
    null,
  );
  // 合并默认值：保证新加的能力也能出现
  const merged: CapabilityToggles = { ...DEFAULT_CAPABILITY_TOGGLES };
  if (stored) {
    for (const cap of CAPABILITIES) {
      if (stored[cap.id]) merged[cap.id] = stored[cap.id];
    }
  }
  return merged;
}

export function saveCapabilityToggles(toggles: CapabilityToggles): void {
  if (!isBrowser()) return;
  localStorage.setItem(KEY_TOGGLES, JSON.stringify(toggles));
  emitChange();
}

// ────────────────────────────────────────────────────────────
// 公开 API：Token
// ────────────────────────────────────────────────────────────

export function loadUserTokens(): UserToken[] {
  ensureVersion();
  if (!isBrowser()) return INITIAL_USER_TOKENS;
  const raw = localStorage.getItem(KEY_TOKENS);
  if (raw === null) return INITIAL_USER_TOKENS;
  return safeParse<UserToken[]>(raw, INITIAL_USER_TOKENS);
}

export function saveUserTokens(tokens: UserToken[]): void {
  if (!isBrowser()) return;
  localStorage.setItem(KEY_TOKENS, JSON.stringify(tokens));
  emitChange();
}

// ────────────────────────────────────────────────────────────
// 公开 API：已下发 Agent
// ────────────────────────────────────────────────────────────

export function loadDistributedAgents(): DistributedAgent[] {
  ensureVersion();
  if (!isBrowser()) return INITIAL_DISTRIBUTED_AGENTS;
  const raw = localStorage.getItem(KEY_DISTRIBUTED);
  if (raw === null) return INITIAL_DISTRIBUTED_AGENTS;
  return safeParse<DistributedAgent[]>(raw, INITIAL_DISTRIBUTED_AGENTS);
}

export function saveDistributedAgents(agents: DistributedAgent[]): void {
  if (!isBrowser()) return;
  localStorage.setItem(KEY_DISTRIBUTED, JSON.stringify(agents));
  emitChange();
}

// ────────────────────────────────────────────────────────────
// 公开 API：自动装载策略
// ────────────────────────────────────────────────────────────

export function loadAutoInstallPolicy(): AutoInstallPolicy {
  ensureVersion();
  if (!isBrowser()) return DEFAULT_AUTO_INSTALL_POLICY;
  const raw = localStorage.getItem(KEY_POLICY);
  if (raw === null) return DEFAULT_AUTO_INSTALL_POLICY;
  return safeParse<AutoInstallPolicy>(raw, DEFAULT_AUTO_INSTALL_POLICY);
}

export function saveAutoInstallPolicy(policy: AutoInstallPolicy): void {
  if (!isBrowser()) return;
  localStorage.setItem(KEY_POLICY, JSON.stringify(policy));
  emitChange();
}

// ────────────────────────────────────────────────────────────
// 公开 API：变更事件订阅
// ────────────────────────────────────────────────────────────

export function subscribeChange(handler: () => void): () => void {
  if (!isBrowser()) return () => {};
  window.addEventListener(EVENT_NAME, handler);
  return () => window.removeEventListener(EVENT_NAME, handler);
}
