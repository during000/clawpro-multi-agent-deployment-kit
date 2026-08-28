/**
 * agentStore - Agent 实例数据共享 store
 * 使用 localStorage 在 MyAgent 与 AgentDetail 之间同步数据（含 roleName 等动态字段）。
 */

import { MOCK_OPENCLAW_LIST, type AgentRoleSlot } from "./mockData";

const STORAGE_KEY = "openclaw_list";
// mock 种子版本：每次更新 MOCK_OPENCLAW_LIST 结构/数据（如新增实例分组字段）时递增此值，
// loadClawList 会据此自动用新 mock 覆盖旧的 localStorage 缓存，避免"改了 mock 但页面读旧缓存"。
const SEED_VERSION = "2026-08-13-group";
const SEED_VERSION_KEY = "openclaw_list_seed_version";

// [006] 分页验证用：URL 带 ?_pagingDemo=1 时，自动把 15 条 mock 扩到 80 条，方便验证分页控件
function maybeExpandForPagingDemo(list: AgentItem[]): AgentItem[] {
  if (typeof window === "undefined") return list;
  try {
    const url = new URL(window.location.href);
    if (url.searchParams.get("_pagingDemo") !== "1") return list;
  } catch {
    return list;
  }
  // 复制 5 份（15 * 5 = 75 条），给每条 id / instanceId / name 加后缀避免冲突
  const expanded: AgentItem[] = [];
  for (let copy = 1; copy <= 5; copy++) {
    list.forEach((item, idx) => {
      expanded.push({
        ...item,
        id: copy === 1 ? item.id : `${item.id}-copy${copy}`,
        instanceId: copy === 1 ? item.instanceId : `${item.instanceId}-c${copy}`,
        name: copy === 1 ? item.name : `${item.name} #${copy}`,
        // 时间依次往前递推，保证排序有变化
        createdAt: `2026-03-${String(20 - copy).padStart(2, "0")} ${String(10 + idx).padStart(2, "0")}:00:00`,
      });
    });
  }
  return expanded;
}

export interface AgentItem {
  id: string;
  instanceId: string;
  name: string;
  status: string;
  agentType?: "openclaw" | "hermes" | "lightclawace" | "localagent";
  localProduct?: "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
  localConnectionStatus?: "connected" | "disconnected";
  localResourceSyncStatus?: "syncing" | "synced";
  /** 本地 Agent 最近一次向 Hatchery 上报信息的时间 */
  lastReportedAt?: string;
  createdAt: string;
  model: string;
  modelVersion: string;
  channels: any[];
  skills: any[];
  standards?: any[];
  op?: string;
  roleName?: string;
  /**
   * 主角色位最后一次下发到该实例时的版本号（x.y 格式）。
   * 仅作为「无 roles 的存量单角色实例」的兼容存储路径；有 roles 时以 roles[i] 上的同名字段为准，
   * 本字段只作为主角色位的镜像同步维护。
   */
  distributedRoleVersion?: string;
  /**
   * 实例下的角色位全集（含主角色位，靠 AgentRoleSlot.isMain 标记）。
   * 用户端「我的 Agent」批量切换角色、管控端「角色设定」批量下发角色，均以本字段为唯一权威模型；
   * 缺省时退回 roleName/distributedRoleVersion 的单角色语义，存量实例零迁移。
   */
  roles?: AgentRoleSlot[];
  memoryStatus?: 'none' | 'free' | 'pro';
  groupId?: string;
  groupName?: string;
  /** 关联项目 ID 列表（实例会额外下发这些项目在「项目资产管理」中的资产/技能） */
  projectIds?: string[];
  /** 关联项目名称列表（与 projectIds 一一对应，用于展示） */
  projectNames?: string[];
  // ===== 计费相关字段（按量计费 + 关机不收费）=====
  /** 计费模式：subscription=包年包月（默认），payg=按量计费 */
  billingMode?: "subscription" | "payg";
  /** 小时单价（元/小时），payg 模式下生效 */
  hourlyRate?: number;
  /** 累计运行分钟数（仅运行态累加） */
  runningMinutes?: number;
  /** 最近一次进入运行态的时间戳（ISO 字符串，用于计时） */
  lastStartedAt?: string;
  /** 累计费用（元）= 运行小时 × 单价 */
  accumulatedCost?: number;
  /** 创建人邮箱（与 MyOpenClaw.OpenClawItem / mockData 对齐，用于共享权限判定） */
  creator?: string;
  /** 归属人邮箱。用户端自建场景下与 creator 相同；管控端代建时为目标用户邮箱 */
  owner?: string;
  /** 共享范围类型：private=仅自己，shared=共享到分组或个人 */
  shareScope?: "private" | "shared";
  shareGroupIds?: string[];
  shareGroupNames?: string[];
  shareUserIds?: string[];
  shareUserNames?: string[];
}

/** 从 localStorage 读取列表，首次使用 MOCK 数据初始化 */
export function loadClawList(): AgentItem[] {
  // [006] 分页验证模式：每次都以扩展后的 mock 为准（忽略 localStorage 缓存），避免旧数据干扰
  if (typeof window !== "undefined") {
    try {
      const url = new URL(window.location.href);
      if (url.searchParams.get("_pagingDemo") === "1") {
        const expanded = maybeExpandForPagingDemo(MOCK_OPENCLAW_LIST as unknown as AgentItem[]);
        saveClawList(expanded);
        return expanded;
      }
    } catch {
      // ignore
    }
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw) as AgentItem[];
  } catch {
    // ignore
  }
  // 首次：用 mock 数据初始化
  const initial = MOCK_OPENCLAW_LIST as unknown as AgentItem[];
  saveClawList(initial);
  return initial;
}

/** 保存列表到 localStorage */
export function saveClawList(list: AgentItem[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
}

/** 根据 id 查询单个 claw */
export function findClawById(id: string): AgentItem | undefined {
  const list = loadClawList();
  return list.find((c) => c.id === id);
}

// 事件通知机制：当列表变更时通知订阅者
type Listener = () => void;
const listeners: Set<Listener> = new Set();

export function onClawListChange(fn: Listener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

export function notifyClawListChange() {
  listeners.forEach((fn) => fn());
}
