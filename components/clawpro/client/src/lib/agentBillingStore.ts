/**
 * agentBillingStore - 计费模式存储
 *
 * 计费模式分两层（符合"启动配置 vs 运行时"的资源管理概念）：
 *
 *   1) 启动模板默认计费模式（DEFAULT_LAUNCH_MODE_KEY）
 *      - 在「资源管理 → 启动配置」里设置
 *      - 作用：新建 Agent 时套用此模式作为初始值
 *      - 修改后不影响已启动的 Agent
 *
 *   2) 已启动 Agent 各自的计费模式（PER_AGENT_KEY）
 *      - 创建时从启动模板复制一份，之后随 Agent 自身存在
 *      - Agent 列表关机弹窗据此判断：按量计费的 Agent 关机时提供「关机模式」选择
 *
 * 两者解耦，避免"修改模板影响已启动 Agent"的耦合问题。
 */

export type AgentBillingMode = "subscription" | "payg";

const DEFAULT_LAUNCH_MODE_KEY = "agent_default_launch_billing_mode";
const PER_AGENT_KEY = "agent_billing_modes";

type Listener = () => void;
const listeners: Set<Listener> = new Set();

// ─── 启动模板默认计费模式 ───────────────────────────────────────────────

/** 读取启动模板默认计费模式（未设置时回落包年包月） */
export function getDefaultLaunchBillingMode(): AgentBillingMode {
  try {
    const raw = localStorage.getItem(DEFAULT_LAUNCH_MODE_KEY);
    if (raw === "payg" || raw === "subscription") return raw;
  } catch {
    // ignore
  }
  return "subscription";
}

/** 设置启动模板默认计费模式（仅影响后续新建 Agent，不影响已有 Agent） */
export function setDefaultLaunchBillingMode(mode: AgentBillingMode): void {
  localStorage.setItem(DEFAULT_LAUNCH_MODE_KEY, mode);
  listeners.forEach((fn) => fn());
}

// ─── 已启动 Agent 各自的计费模式 ────────────────────────────────────────

/** 读取所有已启动 Agent 的计费模式映射 */
export function loadAgentBillingModes(): Record<string, AgentBillingMode> {
  try {
    const raw = localStorage.getItem(PER_AGENT_KEY);
    if (raw) return JSON.parse(raw) as Record<string, AgentBillingMode>;
  } catch {
    // ignore
  }
  return {};
}

/** 读取单个 Agent 的计费模式（未设置返回 undefined） */
export function getAgentBillingMode(agentId: string): AgentBillingMode | undefined {
  return loadAgentBillingModes()[agentId];
}

/** 设置单个 Agent 的计费模式（已启动 Agent 的运行时调整） */
export function setAgentBillingMode(agentId: string, mode: AgentBillingMode): void {
  const all = loadAgentBillingModes();
  all[agentId] = mode;
  localStorage.setItem(PER_AGENT_KEY, JSON.stringify(all));
  listeners.forEach((fn) => fn());
}

// ─── 公共 ──────────────────────────────────────────────────────────────

/** 订阅计费模式变更 */
export function onAgentBillingModeChange(fn: Listener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/** 计费模式中文标签 */
export function billingModeLabel(mode: AgentBillingMode): string {
  return mode === "payg" ? "按量计费" : "包年包月";
}
