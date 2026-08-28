/**
 * adminAgentStore - 管控端 Agent 列表本地缓存
 *
 * 背景：管控端 `OpenClawMonitor` 的表格数据源是 `MOCK_CLAWS` 静态数组 + 部门补全，没有后端接口。
 * 当管控端通过「+ 创建 Agent」弹窗创建出新 Agent 时，需要：
 *   1. 立即出现在管控端表格顶部（本 store 解决）
 *   2. 同步出现在目标 userId 的用户端「我的 Agent」页面（已由 openclawStore 解决）
 *
 * 本 store 只存「管控端新建的增量条目」，不存 MOCK_CLAWS 全量；
 * 读取时由调用方 `[...MOCK_CLAWS, ...loadAdminCreatedAgents()]` 合并，
 * 避免破坏既有 mock 数据，也避免缓存膨胀。
 */
const STORAGE_KEY = "admin_created_agents";

// 这里不能直接 import `Claw` 接口（它定义在 OpenClawMonitor.tsx 内部），
// 用一个结构等价的最小接口避免循环依赖；OpenClawMonitor 端再 `as Claw` 收口。
export interface AdminCreatedAgent {
  id: string;
  instanceId: string;
  name: string;
  /**
   * 用户ID：该 Agent 的归属用户 ID。
   * - 用户端自建 = 该用户自己
   * - 管控端代建 = 管理员在弹窗中为其分配 Agent 的目标用户 ID
   * 历史字段名沿用 `creator`（避免破坏既有 localStorage 数据），语义已迁移为"归属用户"。
   */
  creator: string;
  createTime: string;
  status:
    | "creating"
    | "createFail"
    | "running"
    | "loading"
    | "loadFail"
    | "shutdown"
    | "maintaining"
    | "pending"
    | "upgrading";
  version: string;
  agentType: "OpenClaw" | "Hermes" | "LightclawACE" | "LocalAgent";
  localProduct?: "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
  localConnectionStatus?: "connected" | "disconnected";
  pluginVersions: {
    wechat: string;
    dingtalk: string;
    feishu: string;
    wecom: string;
    qq: string;
  };
  /**
   * Agent 所属组织：管理员在创建 Agent 弹窗中为该 Agent 选定的分组。
   * 一旦创建即固定绑定，是 Agent 自身的属性，不随归属用户的组织变化而变。
   */
  groupId?: string;
  groupName?: string;
  department?: string;
  departmentId?: string;
  tags?: { key: string; value: string }[];
  billingMode?: "subscription" | "payg";
  runningMinutes?: number;
}

/** 读取「管控端创建的 Agent」缓存（不含 MOCK_CLAWS） */
export function loadAdminCreatedAgents(): AdminCreatedAgent[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as AdminCreatedAgent[]) : [];
  } catch {
    return [];
  }
}

/** 覆盖式保存 */
export function saveAdminCreatedAgents(list: AdminCreatedAgent[]): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
  } catch {
    // localStorage 满 / 隐私模式：静默失败，不阻塞用户操作
  }
}

/** 追加一条到队首（按创建时间倒序，最新的在最上面） */
export function appendAdminCreatedAgent(item: AdminCreatedAgent): void {
  const list = loadAdminCreatedAgents();
  saveAdminCreatedAgents([item, ...list]);
}

/** 按 id 删除一条 */
export function removeAdminCreatedAgent(id: string): void {
  const list = loadAdminCreatedAgents();
  saveAdminCreatedAgents(list.filter((c) => c.id !== id));
}

/** 清空缓存（调试用，目前未在 UI 暴露） */
export function clearAdminCreatedAgents(): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
}

// ============ 事件通知（同标签页内多个组件订阅） ============
type Listener = () => void;
const listeners: Set<Listener> = new Set();

export function onAdminCreatedAgentsChange(fn: Listener): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export function notifyAdminCreatedAgentsChange(): void {
  listeners.forEach((fn) => {
    try {
      fn();
    } catch {
      // 单个订阅者抛错不影响其他订阅者
    }
  });
}
