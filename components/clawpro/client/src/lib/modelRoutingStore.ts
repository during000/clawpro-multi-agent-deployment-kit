/**
 * modelRoutingStore.ts
 * 管控端「模型自动路由策略」共享状态存储
 *
 * 与 modelConfigStore 同款模式：localStorage 持久化 + window 事件广播。
 *
 * 概念：
 *   - 一条路由策略（RoutingStrategy）= 路由模式（balance / cost / quality）
 *                            + 备选模型 id 列表（candidateModelIds）
 *                            + 应用范围（ScopeType "all" | "groups"）
 *                            + 应用到的用户组 id 列表（selectedGroupIds）
 *
 * 约束（按需求 §3）：
 *   备选模型只能引用「管控端配置的非自定义模型」，即 ModelRow.provider !== "__custom__"。
 *   store 不强制校验，UI 层 RouteStrategyDialog 负责过滤。
 *
 * 用户端感知（按需求 §1）：
 *   用户端模型下拉里出现「智能路由（Auto）策略」分组/选项时，不需要读策略细节，
 *   仅需感知"存在哪些策略" + "当前用户命中哪一条"。本期仅生成可消费的
 *   `routingStrategies` 数组给上层读取，UI 渲染"由管理员配置，自动选择最优模型" 描述。
 */

import { useEffect, useState, useCallback } from "react";
import { CUSTOM_PROVIDER_VALUE } from "./modelConfigStore";

/** 路由模式：平衡 / 成本优先 / 效果优先 */
export type RouteMode = "balance" | "cost" | "quality";

export const ROUTE_MODE_OPTIONS: { value: RouteMode; label: string; description: string }[] = [
  { value: "balance", label: "平衡模式", description: "综合考虑成本与效果，自动选择性价比最高的模型" },
  { value: "cost", label: "成本优先", description: "优先选择成本最低的模型，兼顾可用性" },
  { value: "quality", label: "效果优先", description: "优先选择效果最佳的模型，不考虑成本" },
];

/** 与 ScopePopover 复用同一份 scope 语义 */
export type RouteScope = "all" | "groups";

export interface RoutingStrategy {
  id: string;
  /** 策略名称（用户自定义） */
  name: string;
  /** 路由模式 */
  mode: RouteMode;
  /** 备选模型 id 列表（仅允许非自定义模型） */
  candidateModelIds: string[];
  /** 应用范围：全部用户 / 按用户组 */
  scope: RouteScope;
  /** scope = groups 时选中的用户组 id */
  selectedGroupIds: string[];
  /** 备注（可选） */
  description?: string;
  /** 启用状态：关闭后用户端不再看到该策略 */
  enabled: boolean;
  /** 创建时间（毫秒） */
  createdAt: number;
}

// ─── 默认数据 ────────────────────────────────────────────────────────────────
// 提供 1 条 demo 策略，确保首次进入 Tab 时有数据可看，体验与现有 ModelConfig 一致。

export const DEFAULT_ROUTING_STRATEGIES: RoutingStrategy[] = [
  {
    id: "rs-default-1",
    name: "全公司默认智能路由",
    mode: "balance",
    candidateModelIds: [], // 默认不预选，避免 id 失同步；用户进入页面后自行选择
    scope: "all",
    selectedGroupIds: [],
    description: "系统默认创建的智能路由策略，可按需调整。",
    enabled: true,
    createdAt: Date.now(),
  },
];

// ─── 存储 & 广播 ─────────────────────────────────────────────────────────────

const STORAGE_KEY = "openclaw_admin_routing_strategies";
const CHANGE_EVENT = "openclaw_admin_routing_strategies_changed";

export function loadRoutingStrategies(): RoutingStrategy[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [...DEFAULT_ROUTING_STRATEGIES];
    const parsed = JSON.parse(raw) as RoutingStrategy[];
    if (!Array.isArray(parsed)) return [...DEFAULT_ROUTING_STRATEGIES];
    return parsed;
  } catch {
    return [...DEFAULT_ROUTING_STRATEGIES];
  }
}

export function saveRoutingStrategies(list: RoutingStrategy[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
  } catch {
    // 静默忽略（quota exceed 等），demo 场景不阻塞 UI
  }
  window.dispatchEvent(new CustomEvent(CHANGE_EVENT));
}

export function onRoutingStrategiesChange(callback: () => void): () => void {
  const handler = () => callback();
  window.addEventListener(CHANGE_EVENT, handler);
  // 跨标签页同步
  const storageHandler = (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) callback();
  };
  window.addEventListener("storage", storageHandler);
  return () => {
    window.removeEventListener(CHANGE_EVENT, handler);
    window.removeEventListener("storage", storageHandler);
  };
}

// ─── React Hook ──────────────────────────────────────────────────────────────

/**
 * useRoutingStrategiesState
 *   const [strategies, setStrategies] = useRoutingStrategiesState();
 * setter 自动 save + broadcast。
 */
export function useRoutingStrategiesState(): [
  RoutingStrategy[],
  (next: RoutingStrategy[] | ((prev: RoutingStrategy[]) => RoutingStrategy[])) => void
] {
  const [list, setListState] = useState<RoutingStrategy[]>(() => loadRoutingStrategies());

  useEffect(() => {
    return onRoutingStrategiesChange(() => setListState(loadRoutingStrategies()));
  }, []);

  const setList = useCallback(
    (next: RoutingStrategy[] | ((prev: RoutingStrategy[]) => RoutingStrategy[])) => {
      setListState((prev) => {
        const resolved = typeof next === "function" ? (next as (p: RoutingStrategy[]) => RoutingStrategy[])(prev) : next;
        saveRoutingStrategies(resolved);
        return resolved;
      });
    },
    []
  );

  return [list, setList];
}

/** 只读订阅（适合用户端下拉读取） */
export function useRoutingStrategies(): RoutingStrategy[] {
  const [list, setList] = useState<RoutingStrategy[]>(() => loadRoutingStrategies());
  useEffect(() => {
    return onRoutingStrategiesChange(() => setList(loadRoutingStrategies()));
  }, []);
  return list;
}

// ─── 工具函数 ────────────────────────────────────────────────────────────────

/** 判断一个 ModelRow 是否可作为路由备选模型：必须是管控端配置且非自定义 */
export function isEligibleCandidate(provider: string | undefined | null): boolean {
  return !!provider && provider !== CUSTOM_PROVIDER_VALUE;
}
