/**
 * upgradePushStore - 「推送更新」生效状态本地存储
 *
 * 跨页面共享的数据：
 *   - 管控端「镜像配置 ImageManagement」：管理员推送、撤回、查看推送状态
 *   - 管控端「Agent 列表 OpenClawMonitor」：批量更新红点判断
 *   - 用户端「OpenClawDetail」：版本号旁的"建议升级"徽章
 *
 * 核心心智：
 *   1. 一个 Agent 类型只能有一条「正在提醒」推送
 *   2. 推送的目标版本 = 推送时该类型的"启用镜像"版本（与生效镜像绑定）
 *   3. 当管理员切换启用镜像时，旧推送自动失效（pruneOnVersionChange）
 *   4. 当用户的实例已经 ≥ 推送版本，用户端不展示提醒（消费方实时判断）
 *   5. 不设过期时间，靠"撤回 / 切换启用版本 / 实例已升级"自然消失
 */

const STORAGE_KEY = "admin_active_pushes_v1";
/** 用户主动撤回过的 Agent 类型集合（撤回后不再被默认 mock 回填） */
const CLEARED_KEY = "admin_active_pushes_cleared_v1";

export interface ActivePush {
  /** Agent 类型 ID（系统：OpenClaw / HermesAgent / LightClawACE 或 custom-xxx） */
  agentType: string;
  /** 推送时的目标版本（= 推送瞬间的启用镜像 agentVersion） */
  version: string;
  /** 推送时该 Agent 类型的展示名（避免后续展示需要再去 ImageManagement 反查） */
  agentTypeLabel: string;
  /** 推送的镜像名（人类可读） */
  imageName?: string;
  /** 镜像来源：腾讯云公共镜像 / 企业自维护 */
  imageSource?: "public" | "custom";
  /** 推送时间 yyyy-MM-dd HH:mm:ss */
  pushedAt: string;
  /** 推送人（mock：alice@acompany.com） */
  pushedBy: string;
  /** 推送时附带的提示文案（默认根据版本说明生成） */
  message?: string;
}

type PushMap = Record<string, ActivePush>;

/** 默认 mock 推送数据（演示用：管理员已推送 OpenClaw 升级提醒） */
const DEFAULT_PUSH_MAP: PushMap = {
  OpenClaw: {
    agentType: "OpenClaw",
    agentTypeLabel: "OpenClaw",
    version: "2026.4.23",
    imageName: "OpenClaw on Ubuntu 24.04",
    imageSource: "public",
    pushedAt: "2026-05-13 10:30:00",
    pushedBy: "alice@acompany.com",
    message: "管理员推荐升级到 v2026.4.23",
  },
};

// ── 内部 IO ─────────────────────────────────────────────
function readAll(): PushMap {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw) as PushMap;
  } catch {
    /* ignore */
  }
  return {};
}

function writeAll(map: PushMap): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
    // 同窗口需要 dispatch storage 事件，让其他组件订阅生效
    window.dispatchEvent(new Event("upgrade-push-changed"));
  } catch {
    /* ignore */
  }
}

/** 读取"用户已撤回"集合 */
function readClearedSet(): Set<string> {
  try {
    const raw = localStorage.getItem(CLEARED_KEY);
    if (raw) return new Set(JSON.parse(raw) as string[]);
  } catch {
    /* ignore */
  }
  return new Set();
}

/** 标记某 Agent 类型为"已撤回" */
function markCleared(agentType: string): void {
  const set = readClearedSet();
  set.add(agentType);
  try {
    localStorage.setItem(CLEARED_KEY, JSON.stringify(Array.from(set)));
  } catch {
    /* ignore */
  }
}

/** 清除"已撤回"标记（用户重新推送时调用） */
function unmarkCleared(agentType: string): void {
  const set = readClearedSet();
  if (set.has(agentType)) {
    set.delete(agentType);
    try {
      localStorage.setItem(CLEARED_KEY, JSON.stringify(Array.from(set)));
    } catch {
      /* ignore */
    }
  }
}

/** 判断某 Agent 类型是否被用户主动撤回过 */
function isCleared(agentType: string): boolean {
  return readClearedSet().has(agentType);
}

// ── 公开 API ────────────────────────────────────────────

/** 读取所有正在推送的记录 */
export function listActivePushes(): ActivePush[] {
  return Object.values(readAll());
}

/**
 * 查询某个 Agent 类型当前是否有推送
 *
 * 兜底逻辑：如果 localStorage 中没有该类型的推送，且默认 mock 中有 → 返回默认 mock
 * 这样即使用户的 localStorage 处于异常状态（空对象、被旧版本污染等），
 * 设计/演示场景下默认的"管理员推荐升级"徽章仍能正常展示。
 *
 * 用户主动撤回（clearActivePush）后会写入"已撤回"标记，此时不会再回退到默认 mock。
 */
export function getActivePush(agentType: string): ActivePush | null {
  const all = readAll();
  if (all[agentType]) return all[agentType];
  // 用户主动撤回过：尊重撤回结果，不再回退默认 mock
  if (isCleared(agentType)) return null;
  // 否则：回退到默认 mock（如果有）
  return DEFAULT_PUSH_MAP[agentType] ?? null;
}

/** 推送（覆盖该 Agent 类型的旧推送，同时清除"已撤回"标记） */
export function setActivePush(push: ActivePush): void {
  const all = readAll();
  all[push.agentType] = push;
  writeAll(all);
  // 重新推送，清除可能存在的"已撤回"标记
  unmarkCleared(push.agentType);
}

/** 撤回某个 Agent 类型的推送（同时记录"已撤回"标记，避免被默认 mock 回填） */
export function clearActivePush(agentType: string): void {
  const all = readAll();
  if (all[agentType]) {
    delete all[agentType];
    writeAll(all);
  }
  markCleared(agentType);
}

/**
 * 当某个 Agent 类型的"启用版本"变了，自动撤回与该类型有关的旧推送
 *
 * 调用时机：管理员在镜像配置页切换启用镜像、关闭"用户可见"开关时
 *
 * 逻辑：
 *   - 该类型的旧推送目标版本 ≠ 新启用版本 → 撤回
 *   - 没有启用版本（newEnabledVersion 为空）→ 也撤回
 */
export function pruneOnVersionChange(
  agentType: string,
  newEnabledVersion: string | null,
): void {
  const all = readAll();
  const exist = all[agentType];
  if (!exist) return;
  if (!newEnabledVersion || exist.version !== newEnabledVersion) {
    delete all[agentType];
    writeAll(all);
  }
}

/**
 * 用户端消费方判断：当前实例是否需要展示"管理员推荐升级"徽章
 *
 * 规则：
 *   - 该类型有正在推送
 *   - 且实例当前版本 < 推送版本（语义版本比较）
 */
export function shouldShowRecommendBadge(
  agentType: string,
  instanceVersion: string,
): { show: false } | { show: true; push: ActivePush } {
  const push = getActivePush(agentType);
  if (!push) return { show: false };
  if (compareVersion(instanceVersion, push.version) >= 0) return { show: false };
  return { show: true, push };
}

/**
 * 简易语义版本比较（兼容 X.Y.Z 与 YYYY.M.D 两种格式）
 *  - 返回值 < 0：a 旧
 *  - 返回值 = 0：相同
 *  - 返回值 > 0：a 新
 */
export function compareVersion(a: string, b: string): number {
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
