/**
 * doctorDiagnosisApi.ts — 龙虾医生诊断服务 Mock API 层
 *
 * 模拟服务端行为：诊断状态按 Agent 实例维度管理，所有端共享同一份数据。
 * 后续对接真实后端时，只需将本文件内的函数实现替换为 fetch/axios 调用即可。
 *
 * 设计原则：
 * - 诊断挂在 Agent 实例上（agentInstanceId），不是挂在用户/浏览器上
 * - 同一台 Agent 同一时间只能有一个诊断在进行
 * - 任何端/用户打开都能查询到诊断状态
 * - 只有发起者（initiatorId）能看到完整对话内容
 * - 其他人只能看到"诊断中"状态
 */

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export type DiagCheckItem = {
  label: string;
  status: "ok" | "error" | "warn";
  detail?: string;
};

export type RepairResult = {
  label: string;
  ok: boolean;
  reason?: string;
};

export type DoctorMessageContent =
  | { type: "text"; text: string }
  | { type: "check_list"; items: DiagCheckItem[] }
  | { type: "repair_summary"; results: RepairResult[] };

export type DoctorMsg =
  | { kind: "system"; text: string; transient?: boolean }
  | { kind: "assistant"; parts: DoctorMessageContent[]; loading?: boolean; transient?: boolean }
  | { kind: "user"; text: string };

export type DiagPhase =
  | "diagnosing"
  | "summary_ready"
  | "repairing"
  | "done"
  | "idle";

export type InstanceStatus = "none" | "creating" | "active" | "destroying" | "ended";

/** 发起端类型 */
export type InitiatorType = "admin" | "user";

/** 诊断会话记录（服务端模型） */
export interface DiagnosisSession {
  agentInstanceId: string;
  doctorInstanceId: string;
  initiatorId: string;       // 发起者唯一标识（管理员ID 或 用户ID）
  initiatorType: InitiatorType;
  status: InstanceStatus;
  phase: DiagPhase;
  messages: DoctorMsg[];
  createdAt: number;
  lastActiveAt: number;
}

/** 查询诊断状态的返回 */
export interface DiagnosisStatusResult {
  /** 是否有进行中的诊断 */
  active: boolean;
  /** 是否是当前调用者发起的 */
  isMine: boolean;
  /** 诊断会话信息（active=true 时存在） */
  session?: {
    doctorInstanceId: string;
    initiatorType: InitiatorType;
    status: InstanceStatus;
    phase: DiagPhase;
    messages: DoctorMsg[];
  };
}

/** 发起诊断的请求参数 */
export interface StartDiagnosisParams {
  agentInstanceId: string;
  initiatorId: string;
  initiatorType: InitiatorType;
  snapshot?: boolean;
}

/** 发起诊断的返回 */
export interface StartDiagnosisResult {
  success: boolean;
  doctorInstanceId?: string;
  /** 失败原因：conflict 表示已有诊断在进行 */
  reason?: "conflict";
}

/** 结束诊断的请求参数 */
export interface EndDiagnosisParams {
  agentInstanceId: string;
  doctorInstanceId: string;
  rollback?: boolean;
}

/** 结束诊断的返回 */
export interface EndDiagnosisResult {
  success: boolean;
}

// ─── Mock 数据存储（模拟服务端内存/数据库）────────────────────────────────────

/**
 * 使用 localStorage 作为 mock "服务端数据库"
 * Key: `__doctor_server_sessions__`
 * Value: Record<agentInstanceId, DiagnosisSession>
 *
 * 之所以仍然用 localStorage，是因为需要跨标签页共享（模拟多人同时访问）。
 * 但与之前不同的是：
 * - 前端组件不直接读写 localStorage
 * - 所有操作都通过本文件的 API 函数
 * - 后续替换为真实接口时，前端代码无需改动
 */
const MOCK_DB_KEY = "__doctor_server_sessions__";

function readMockDB(): Record<string, DiagnosisSession> {
  try {
    const raw = localStorage.getItem(MOCK_DB_KEY);
    if (!raw) return {};
    return JSON.parse(raw) as Record<string, DiagnosisSession>;
  } catch {
    return {};
  }
}

function writeMockDB(db: Record<string, DiagnosisSession>) {
  try {
    localStorage.setItem(MOCK_DB_KEY, JSON.stringify(db));
  } catch {
    // ignore
  }
}

// ─── Mock API 实现 ────────────────────────────────────────────────────────────

/**
 * 查询某台 Agent 当前的诊断状态
 *
 * @param agentInstanceId - Agent 实例 ID
 * @param callerId - 当前调用者 ID（管理员ID 或 用户ID）
 * @returns 诊断状态
 */
export async function queryDiagnosisStatus(
  agentInstanceId: string,
  callerId: string,
): Promise<DiagnosisStatusResult> {
  // 模拟网络延迟
  await delay(100);

  const db = readMockDB();
  const session = db[agentInstanceId];

  if (!session || session.status === "ended" || session.status === "none") {
    return { active: false, isMine: false };
  }

  const isMine = session.initiatorId === callerId;

  return {
    active: true,
    isMine,
    session: {
      doctorInstanceId: session.doctorInstanceId,
      initiatorType: session.initiatorType,
      status: session.status,
      phase: session.phase,
      // 只有发起者才能看到消息内容
      messages: isMine ? session.messages : [],
    },
  };
}

/**
 * 发起诊断
 *
 * @param params - 发起参数
 * @returns 创建结果
 */
export async function startDiagnosis(
  params: StartDiagnosisParams,
): Promise<StartDiagnosisResult> {
  await delay(200);

  const db = readMockDB();
  const existing = db[params.agentInstanceId];

  // 冲突检测：如果已有活跃/创建中的诊断，拒绝
  if (existing && (existing.status === "active" || existing.status === "creating")) {
    return { success: false, reason: "conflict" };
  }

  const doctorInstanceId = `doctor_${params.initiatorType}_${Date.now()}`;

  const session: DiagnosisSession = {
    agentInstanceId: params.agentInstanceId,
    doctorInstanceId,
    initiatorId: params.initiatorId,
    initiatorType: params.initiatorType,
    status: "creating",
    phase: "idle",
    messages: [],
    createdAt: Date.now(),
    lastActiveAt: Date.now(),
  };

  db[params.agentInstanceId] = session;
  writeMockDB(db);

  return { success: true, doctorInstanceId };
}

/**
 * 模拟实例创建完成（创建中 → 活跃）
 * 在真实场景中这个状态变更由后端推送/轮询发现
 */
export async function pollCreationStatus(
  agentInstanceId: string,
): Promise<{ ready: boolean }> {
  await delay(100);

  const db = readMockDB();
  const session = db[agentInstanceId];

  if (!session) return { ready: false };

  // Mock: 创建后 1.5s 变为 active（通过检查时间差）
  if (session.status === "creating" && Date.now() - session.createdAt >= 1500) {
    session.status = "active";
    session.lastActiveAt = Date.now();
    db[agentInstanceId] = session;
    writeMockDB(db);
    return { ready: true };
  }

  if (session.status === "active") {
    return { ready: true };
  }

  return { ready: false };
}

/**
 * 更新诊断消息（发起者推送消息到服务端）
 */
export async function updateDiagnosisMessages(
  agentInstanceId: string,
  messages: DoctorMsg[],
  phase?: DiagPhase,
): Promise<void> {
  const db = readMockDB();
  const session = db[agentInstanceId];
  if (!session) return;

  session.messages = messages;
  session.lastActiveAt = Date.now();
  if (phase !== undefined) {
    session.phase = phase;
  }
  db[agentInstanceId] = session;
  writeMockDB(db);
}

/**
 * 获取诊断消息（轮询）
 * 只有发起者才能获取完整消息
 */
export async function fetchDiagnosisMessages(
  agentInstanceId: string,
  callerId: string,
): Promise<{ messages: DoctorMsg[]; phase: DiagPhase } | null> {
  await delay(50);

  const db = readMockDB();
  const session = db[agentInstanceId];

  if (!session || session.initiatorId !== callerId) return null;

  return {
    messages: session.messages,
    phase: session.phase,
  };
}

/**
 * 结束诊断
 */
export async function endDiagnosis(
  params: EndDiagnosisParams,
): Promise<EndDiagnosisResult> {
  await delay(200);

  const db = readMockDB();
  const session = db[params.agentInstanceId];

  if (!session) return { success: false };

  session.status = "ended";
  session.phase = "idle";
  db[params.agentInstanceId] = session;
  writeMockDB(db);

  // 1秒后从 mock DB 中彻底移除（模拟后端清理）
  setTimeout(() => {
    const db2 = readMockDB();
    delete db2[params.agentInstanceId];
    writeMockDB(db2);
  }, 1000);

  return { success: true };
}

/**
 * 刷新活跃时间（用户有操作时调用，延长自动结束计时）
 */
export async function touchSession(agentInstanceId: string): Promise<void> {
  const db = readMockDB();
  const session = db[agentInstanceId];
  if (!session || session.status !== "active") return;
  session.lastActiveAt = Date.now();
  db[agentInstanceId] = session;
  writeMockDB(db);
}

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// ─── 当前调用者 ID 管理（Mock）──────────────────────────────────────────────────
// 真实场景中来自登录态/token，这里用 localStorage 模拟

const CALLER_ID_KEY = "__doctor_caller_id__";

/**
 * 获取当前调用者 ID
 * 管控端和用户端会有不同的 ID
 */
export function getCallerId(type: InitiatorType): string {
  const key = `${CALLER_ID_KEY}_${type}`;
  let id = localStorage.getItem(key);
  if (!id) {
    id = `${type}_${Math.random().toString(36).slice(2, 10)}`;
    localStorage.setItem(key, id);
  }
  return id;
}

// ─── 调试工具：模拟其他管理员发起诊断（浏览器控制台调用）─────────────────────────
/**
 * 模拟另一个管理员对指定 Agent 发起诊断。
 *
 * 使用方法：在浏览器控制台输入：
 *   window.__simulateOtherAdminDiagnosis__("ins-k25f9zwg")
 *
 * 清除方法：
 *   window.__clearSimulatedDiagnosis__("ins-k25f9zwg")
 *
 * 调用后刷新页面即可看到"该 Agent 当前正在诊断中"的效果。
 */
function simulateOtherAdminDiagnosis(agentInstanceId: string) {
  const db = readMockDB();
  const existing = db[agentInstanceId];
  if (existing && existing.status === "active") {
    console.warn(`[Doctor Debug] Agent ${agentInstanceId} 已有诊断在进行中，先清除再模拟。`);
    delete db[agentInstanceId];
  }

  // 使用一个不同于当前管理员的 callerId
  const fakeAdminId = "admin_other_" + Math.random().toString(36).slice(2, 8);

  db[agentInstanceId] = {
    agentInstanceId,
    doctorInstanceId: "doc-sim-" + Date.now(),
    initiatorId: fakeAdminId,
    initiatorType: "admin",  // 关键：是另一个管理员
    status: "active",
    phase: "diagnosing",
    messages: [],
    createdAt: Date.now(),
    lastActiveAt: Date.now(),
  };

  writeMockDB(db);
  console.log(`[Doctor Debug] ✅ 已模拟"其他管理员"对 ${agentInstanceId} 发起诊断（initiatorId: ${fakeAdminId}）`);
  console.log(`[Doctor Debug] 刷新页面或重新选中该 Agent 即可看到效果。`);
  console.log(`[Doctor Debug] 清除命令: window.__clearSimulatedDiagnosis__("${agentInstanceId}")`);
}

function clearSimulatedDiagnosis(agentInstanceId: string) {
  const db = readMockDB();
  if (db[agentInstanceId]) {
    delete db[agentInstanceId];
    writeMockDB(db);
    console.log(`[Doctor Debug] ✅ 已清除 ${agentInstanceId} 的模拟诊断`);
  } else {
    console.log(`[Doctor Debug] ${agentInstanceId} 没有进行中的诊断`);
  }
}

// 挂载到 window 上，方便控制台调用
if (typeof window !== "undefined") {
  (window as any).__simulateOtherAdminDiagnosis__ = simulateOtherAdminDiagnosis;
  (window as any).__clearSimulatedDiagnosis__ = clearSimulatedDiagnosis;
}
