/**
 * 自定义 Agent 类型的本地 Mock 存储（localStorage）
 * 精简版：已剔除 Feature 脚本相关读写
 */
import type { CustomAgentType } from "./types";

const STORAGE_KEY_TYPES = "admin_custom_agent_types";
const STORAGE_KEY_VERSION = "admin_custom_agent_schema_version";
const CURRENT_SCHEMA_VERSION = "v4-no-scripts";

// 初次或版本升级时清理旧 mock，避免旧 schema（含 feature scripts）残留
(function migrateIfNeeded() {
  if (typeof window === "undefined") return;
  try {
    const v = localStorage.getItem(STORAGE_KEY_VERSION);
    if (v !== CURRENT_SCHEMA_VERSION) {
      localStorage.removeItem(STORAGE_KEY_TYPES);
      localStorage.removeItem("admin_custom_agent_scripts");
      localStorage.setItem(STORAGE_KEY_VERSION, CURRENT_SCHEMA_VERSION);
    }
  } catch {
    /* ignore */
  }
})();

// ── 默认 Mock 数据 ─────────────────────────────────────────
const DEFAULT_CUSTOM_TYPES: CustomAgentType[] = [
  // 示例 1：兼容 OpenClaw 的自定义类型
  {
    id: "custom-openclaw-gpu",
    displayName: "OpenClaw-GPU 定制",
    kernelBase: "OpenClaw",
    description: "基于 OpenClaw 内核，系统内置了 CUDA 11.8 和 GPU 驱动，管控功能与 OpenClaw 一致",
    createdAt: "2026-04-20 10:15:00",
    updatedAt: "2026-04-20 10:15:00",
    linkedInstanceCount: 6,
    createdBy: "alice@acompany.com",
  },
];

// ── 读写接口 ─────────────────────────────────────────────
export function loadCustomTypes(): CustomAgentType[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_TYPES);
    if (raw) return JSON.parse(raw);
  } catch {
    /* ignore */
  }
  saveCustomTypes(DEFAULT_CUSTOM_TYPES);
  return DEFAULT_CUSTOM_TYPES;
}

export function saveCustomTypes(types: CustomAgentType[]): void {
  localStorage.setItem(STORAGE_KEY_TYPES, JSON.stringify(types));
}

/** 按 id 更新单个 custom type（同时写回 localStorage） */
export function updateCustomType(
  id: string,
  patch: Partial<CustomAgentType>,
): CustomAgentType[] {
  const all = loadCustomTypes();
  const next = all.map((t) =>
    t.id === id ? { ...t, ...patch, updatedAt: nowStr() } : t,
  );
  saveCustomTypes(next);
  return next;
}

export function nowStr(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
