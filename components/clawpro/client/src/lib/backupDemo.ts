/**
 * backupDemo - 数据备份需求演示数据注入工具
 *
 * 目的：
 *   为「数据备份与回滚」需求 demo 演示提供 3 个示例 Agent，分别对应
 *   none / creating / ready 三种备份点状态。让 PM 在 demo 环境上
 *   能分别看到这三种状态的实际展示（按钮置灰 / hover 提示 / 不渲染）。
 *
 * 设计要点：
 *   1. 状态独立存储于专用 map（`backup_demo_status_map`），避免污染 AgentItem 类型，
 *      联调真实接口时直接换成真实字段即可，helper 可整体删除；
 *   2. helper 幂等：localStorage 标记 `backup_demo_seeded_v1` 已存在时不再创建，
 *      重复调用零成本；
 *   3. 注入后会 notifyClawListChange()，让列表页自动刷新无需手动 reload。
 *
 * 联调 / 上线时：
 *   - 联调：用真实后端字段 `backup_point_status` 替换 `getDemoBackupStatus` 的读取；
 *   - 上线：移除 demo agent 注入调用（helper 文件可整段删除）。
 */

import { MOCK_OPENCLAW_LIST } from "./mockData";
import {
  loadClawList,
  saveClawList,
  notifyClawListChange,
  type AgentItem,
} from "./openclawStore";

/** 备份点状态枚举（与 OpenClawDetailGuide.tsx 内 type 保持一致） */
export type BackupPointStatus = "none" | "creating" | "ready";

interface DemoAgentSeed {
  id: string;
  instanceId: string;
  name: string;
  status: BackupPointStatus;
}

/** 已 seed 标记，避免重复创建 */
const SEED_FLAG = "backup_demo_seeded_v1";
/** demo 状态 map，agentId -> BackupPointStatus */
const STATUS_MAP_KEY = "backup_demo_status_map";

/** 3 个示例 Agent，分别对应三种备份点状态 */
const DEMO_AGENTS: DemoAgentSeed[] = [
  {
    id: "oc-backup-demo-none",
    instanceId: "ins-backup-demo-none",
    name: "[演示·无备份点] Agent",
    status: "none",
  },
  {
    id: "oc-backup-demo-creating",
    instanceId: "ins-backup-demo-creating",
    name: "[演示·备份生成中] Agent",
    status: "creating",
  },
  {
    id: "oc-backup-demo-ready",
    instanceId: "ins-backup-demo-ready",
    name: "[演示·有备份可回滚] Agent",
    status: "ready",
  },
];

/** 读取当前 localStorage 中的 demo 状态 map */
export function readBackupPointStatusMap(): Record<string, BackupPointStatus> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(STATUS_MAP_KEY);
    return raw ? (JSON.parse(raw) as Record<string, BackupPointStatus>) : {};
  } catch {
    return {};
  }
}

/**
 * 查询指定 agentId 的 mock 备份点状态。
 * - 不在 demo map 中（普通 Agent / 未更新或重装）→ 返回 fallback（默认 "ready"）
 * - 联调时改为读取真实后端字段 `backup_point_status`
 */
export function getDemoBackupStatus(
  agentId: string | undefined,
  fallback: BackupPointStatus = "ready",
): BackupPointStatus {
  if (!agentId) return fallback;
  const map = readBackupPointStatusMap();
  return map[agentId] ?? fallback;
}

/**
 * 注入 3 个示例 Agent + 状态 map（自适应幂等）。
 *
 * **每次调用都检查列表完整性**，缺了就补建，不依赖种子标记做早退。
 * 这样无论 merge 到哪个环境、换浏览器、清 localStorage、手动删了某个 demo agent，
 * 列表页/详情页 mount 时都会自动恢复。
 *
 * SEED_FLAG 保留作为 OTA 升级标记：helper 逻辑变更时改 key 即可强制重新种子。
 */
export function ensureBackupDemoAgents(): void {
  if (typeof window === "undefined") return;

  const list = loadClawList();
  const map = readBackupPointStatusMap();
  let added = false;
  let mapDirty = false;

  DEMO_AGENTS.forEach((seed, idx) => {
    // 1) 确保 status map 里有一份记录
    if (!map[seed.id]) {
      map[seed.id] = seed.status;
      mapDirty = true;
    }

    // 2) 确保 agent 在列表中（换浏览器 / 手动删 / 清 localStorage 后自动补）
    if (list.find((c) => c.id === seed.id)) return;

    const reference = MOCK_OPENCLAW_LIST[0] as unknown as AgentItem;
    const hour = String(9 + idx).padStart(2, "0");
    const seedItem: AgentItem = {
      ...reference,
      id: seed.id,
      instanceId: seed.instanceId,
      name: seed.name,
      status: "running",
      agentType: "openclaw",
      createdAt: `${new Date().toISOString().slice(0, 10)} ${hour}:00:00`,
      channels: [],
      skills: [],
    };
    list.unshift(seedItem);
    added = true;
  });

  if (added) {
    saveClawList(list);
    notifyClawListChange();
  }
  if (mapDirty) {
    localStorage.setItem(STATUS_MAP_KEY, JSON.stringify(map));
  }
  localStorage.setItem(SEED_FLAG, "1");
}
