/**
 * versionModeStore - Agent 版本模式（自动更新 / 指定版本）本地存储
 *
 * 每个 Agent 类型可以选择：
 *   - auto：自动更新（默认），Agent 版本随平台发布自动升级
 *   - pinned：指定版本，管理员选择版本并锁定，不自动升级
 *
 * 存储结构：
 *   - versionMode_v1: Record<string, "auto" | "pinned">
 *   - pinnedVersion_v1: Record<string, string>  // 指定版本模式下锁定的版本号
 */
import { AGENT_VERSIONS } from "@/pages/admin/VersionManagement/mockData";
import { IMG_TO_VERSION_KEY } from "@/pages/admin/ImageManagement/deriveAgentTypeView";
import type { AgentTypeKey } from "@/pages/admin/VersionManagement/mockData";

const MODE_KEY = "admin_version_mode_v1";
const PINNED_VERSION_KEY = "admin_pinned_version_v1";

type VersionMode = "auto" | "pinned";

function readMap<T>(key: string): Record<string, T> {
  try {
    const raw = localStorage.getItem(key);
    if (raw) return JSON.parse(raw) as Record<string, T>;
  } catch { /* ignore */ }
  return {};
}

function writeMap<T>(key: string, map: Record<string, T>): void {
  try {
    localStorage.setItem(key, JSON.stringify(map));
  } catch { /* ignore */ }
}

// ── Mock 初始化：OpenClaw 默认使用旧版本（不在 top3 中），方便验证二次确认弹窗
const MOCK_INIT_KEY = "admin_version_mock_init_v2";
function ensureMockInit(): void {
  if (typeof window === "undefined") return;
  try {
    const ran = localStorage.getItem(MOCK_INIT_KEY);
    if (!ran) {
      // 强制重设 OpenClaw 为指定版本模式 + 旧版本
      const modeMap = readMap<VersionMode>(MODE_KEY);
      modeMap["OpenClaw"] = "pinned";
      writeMap(MODE_KEY, modeMap);
      const pinnedMap = readMap<string>(PINNED_VERSION_KEY);
      pinnedMap["OpenClaw"] = "2026.3.8";
      writeMap(PINNED_VERSION_KEY, pinnedMap);
      localStorage.setItem(MOCK_INIT_KEY, "1");
    }
  } catch { /* ignore */ }
}

// ── 公开 API ────────────────────────────────────────────

/** 获取某 Agent 类型的版本模式（默认 auto） */
export function getVersionMode(agentType: string): VersionMode {
  ensureMockInit();
  const map = readMap<VersionMode>(MODE_KEY);
  return map[agentType] ?? "auto";
}

/** 设置某 Agent 类型的版本模式 */
export function setVersionMode(agentType: string, mode: VersionMode): void {
  const map = readMap<VersionMode>(MODE_KEY);
  map[agentType] = mode;
  writeMap(MODE_KEY, map);
}

/** 获取指定版本模式下锁定的版本号 */
export function getPinnedVersion(agentType: string): string | null {
  const map = readMap<string>(PINNED_VERSION_KEY);
  return map[agentType] ?? null;
}

/** 设置指定版本模式下锁定的版本号 */
export function setPinnedVersion(agentType: string, version: string): void {
  const map = readMap<string>(PINNED_VERSION_KEY);
  map[agentType] = version;
  writeMap(PINNED_VERSION_KEY, map);
}

/**
 * 获取某 Agent 类型在当前模式下的"有效版本"
 * - auto 模式：返回最新平台版本
 * - pinned 模式：返回锁定的版本，若无则回退到最新版本
 */
export function getEffectiveVersion(agentType: string): string | null {
  const mode = getVersionMode(agentType);
  const versionKey = IMG_TO_VERSION_KEY[agentType] as AgentTypeKey | undefined;
  if (!versionKey) return null;

  const versions = AGENT_VERSIONS
    .filter((v) => v.agentType === versionKey)
    .sort((a, b) => b.releaseTime.localeCompare(a.releaseTime));

  if (versions.length === 0) return null;
  const latestVersion = versions[0].version;

  if (mode === "auto") return latestVersion;

  // pinned 模式
  const pinned = getPinnedVersion(agentType);
  return pinned ?? latestVersion;
}

/**
 * 获取某 Agent 类型的可选版本列表（历史版本，默认最近 5 个）
 * 当用户当前使用的版本不在最近 5 个中时，仍保留该版本在列表中
 */
export function getAvailableVersions(agentType: string, currentEffectiveVersion?: string | null): string[] {
  const versionKey = IMG_TO_VERSION_KEY[agentType] as AgentTypeKey | undefined;
  if (!versionKey) return [];

  const allVersions = AGENT_VERSIONS
    .filter((v) => v.agentType === versionKey)
    .sort((a, b) => b.releaseTime.localeCompare(a.releaseTime));

  // 取最近的 5 个版本
  const recent5 = allVersions.slice(0, 5).map((v) => v.version);

  // 如果当前有效版本不在最近 5 个中，也要包含进来
  if (currentEffectiveVersion && !recent5.includes(currentEffectiveVersion)) {
    return [...recent5, currentEffectiveVersion];
  }

  return recent5;
}

/**
 * 获取最新平台版本
 */
export function getLatestPlatformVersion(agentType: string): string | null {
  const versionKey = IMG_TO_VERSION_KEY[agentType] as AgentTypeKey | undefined;
  if (!versionKey) return null;

  const versions = AGENT_VERSIONS
    .filter((v) => v.agentType === versionKey)
    .sort((a, b) => b.releaseTime.localeCompare(a.releaseTime));

  return versions.length > 0 ? versions[0].version : null;
}

/**
 * 判断某 Agent 类型是否有新版本可用（平台最新版本 > 当前有效版本）
 */
export function hasNewerVersion(agentType: string, currentEffectiveVersion: string | null): boolean {
  const latest = getLatestPlatformVersion(agentType);
  if (!latest || !currentEffectiveVersion) return false;
  return compareVersionStr(latest, currentEffectiveVersion) > 0;
}

/**
 * 简单语义版本比较
 */
function compareVersionStr(a: string, b: string): number {
  const ax = (a ?? "").split(".").map((n) => parseInt(n, 10) || 0);
  const bx = (b ?? "").split(".").map((n) => parseInt(n, 10) || 0);
  const len = Math.max(ax.length, bx.length);
  for (let i = 0; i < len; i++) {
    const va = ax[i] ?? 0;
    const vb = bx[i] ?? 0;
    if (va !== vb) return va - vb;
  }
  return 0;
}
