/**
 * ServiceStatusContext - 计费服务状态全局管理
 *
 * 管理两种停服场景：
 * 1. 管控台到期停服（预付费到期）：管控端只读，用户端不能新建 Agent
 *    - 停服后进入回收站保留 15 天，根据剩余天数区分文案紧迫度
 * 2. 席位欠费停服（后付费欠费）：用户端不能新建 Agent，管控端不受影响
 *    - 席位欠费分 2 档：
 *      • arrears-buffer  欠费 2h 内的缓冲期：仍在催充值窗口，展示弹窗 / 浮层 / 通知卡
 *      • arrears         2h 缓冲结束后进入正式欠费：仅保留常态禁用 & tooltip，弹窗类不再打扰
 *
 * ─── 状态阶段 (phase) ─────────────────────────────────────────────
 * 管控台服务生命周期切分为 7 档，便于 QA 与设计走查逐档验收：
 *
 *   active            正常服务期
 *   warning-d1        到期前 1 天预警
 *   warning-d2-6      到期前 2~6 天预警
 *   critical-d7       到期前 7 天临期（更强告警）
 *   suspended-d1      停服「关键日」档（D1 或 D8，代表值 daysSinceSuspended=1，最紧迫展示）
 *   suspended-d2-12   停服「中间日」档（D2-D7 或 D9-D12，代表值 daysSinceSuspended=7）
 *   recycling-d13-15  回收站末段：D13~D15，3 天永久删除倒计时
 *
 * 派生字段：
 *   - consoleStatus:      active / suspended（保留旧接口，兼容已有消费方）
 *   - daysUntilExpire:    距离到期天数（active 阶段可用；正常态为 null）
 *   - daysSinceSuspended: 已停服天数（suspended 阶段可用；正常态为 null）
 *   - recyclingDaysLeft:  回收站剩余天数 = 15 - daysSinceSuspended
 *   - isInRecycling:      是否已进入回收站阶段（recyclingDaysLeft ≤ 15）
 *   - isAdminDisabled:    管控端是否禁用（等价于 phase 起始于 "suspended"）
 */
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

// ─── 阶段定义 ────────────────────────────────────────────────────
export type ServiceStatus = "active" | "suspended";

/**
 * 席位欠费阶段（3 档）
 *   - active          : 未欠费
 *   - arrears-buffer  : 欠费触发后 2h 缓冲期（催充值弹窗 / 浮层活跃期）
 *   - arrears         : 缓冲期结束进入正式欠费（仅禁用 & tooltip 阻断，弹窗不再骚扰）
 */
export type SeatArrearsPhase = "active" | "arrears-buffer" | "arrears";

/**
 * CLS 服务状态（Demo/模拟用）
 *   - active    : CLS 服务正常
 *   - suspended : CLS 服务已停服（用户账户下 CLS 服务停止）
 *
 * 与 clawpro 控制台状态（phase）完全独立、可任意笛卡尔叠加，
 * 由 CLS 计费状态模拟浮层直接切换。
 */
export type ClsStatus = "active" | "suspended";

/** 管控台服务生命周期阶段（7 档） */
export type ServicePhase =
  | "active"
  | "warning-d1"
  | "warning-d2-6"
  | "critical-d7"
  | "suspended-d1"
  | "suspended-d2-12"
  | "recycling-d13-15";

/** 阶段元信息：区间中位（用于模拟天数） */
interface PhaseSpec {
  status: ServiceStatus;
  /** active 阶段：距离到期还有多少天 */
  daysUntilExpire: number | null;
  /** suspended 阶段：已停服多少天（1~15） */
  daysSinceSuspended: number | null;
}

const PHASE_MAP: Record<ServicePhase, PhaseSpec> = {
  active: { status: "active", daysUntilExpire: null, daysSinceSuspended: null },
  // 档位命名含义：warning-d1 = 到期前进入预警的第 1 天（离到期还剩 7 天）；critical-d7 = 到期倒数 D7（离到期只剩 1 天，最紧迫）
  "warning-d1": {
    status: "active",
    daysUntilExpire: 7,
    daysSinceSuspended: null,
  },
  "warning-d2-6": {
    status: "active",
    daysUntilExpire: 4,
    daysSinceSuspended: null,
  },
  "critical-d7": {
    status: "active",
    daysUntilExpire: 1,
    daysSinceSuspended: null,
  },
  // 停服 3 档（BillingStatusToggle 提供 UI）：
  //   suspended-d1     : D1 & D8 「关键日」共用档，代表值 daysSinceSuspended=1（recyclingDaysLeft=14）
  //   suspended-d2-12  : D2-D7 & D9-D12 中间档，代表值 daysSinceSuspended=7（recyclingDaysLeft=8）
  //   recycling-d13-15 : D13-D15 回收站末段，代表值 daysSinceSuspended=14（recyclingDaysLeft=1，红色最强告警）
  "suspended-d1": {
    status: "suspended",
    daysUntilExpire: null,
    daysSinceSuspended: 1,
  },
  "suspended-d2-12": {
    status: "suspended",
    daysUntilExpire: null,
    daysSinceSuspended: 7,
  },
  "recycling-d13-15": {
    status: "suspended",
    daysUntilExpire: null,
    daysSinceSuspended: 14,
  },
};

/** 回收站保留总时长（天） */
const RECYCLING_TOTAL_DAYS = 15;

// ─── State / API 类型 ────────────────────────────────────────────
export interface ServiceStatusState {
  /** 当前阶段（7 档细分） */
  phase: ServicePhase;
  /** 管控台服务状态：active=正常, suspended=停服(含回收站)。派生自 phase */
  consoleStatus: ServiceStatus;
  /** 席位欠费阶段（3 档：active / arrears-buffer / arrears） */
  seatArrearsPhase: SeatArrearsPhase;
  /** 席位是否欠费（派生：seatArrearsPhase !== "active"）——旧消费方兼容 */
  seatArrears: boolean;
  /** 是否处于席位欠费 2h 缓冲期（弹窗 / 浮层类催充值提示仅在此段展示） */
  isSeatArrearsBuffer: boolean;
  /** 是否处于正式欠费档（缓冲期结束后，仅保留左下角常态浮层 + 禁用态，弹窗不再打扰） */
  isSeatArrearsBlocked: boolean;
  /** 管控台到期时间（示例） */
  expireTime: string | null;
  /** 距离到期天数（active 阶段可用；否则 null） */
  daysUntilExpire: number | null;
  /** 已停服天数（suspended 阶段可用；否则 null） */
  daysSinceSuspended: number | null;
  /** 回收站剩余天数 = 15 - daysSinceSuspended（suspended 阶段可用；否则 null） */
  recyclingDaysLeft: number | null;
  /** 停服弹窗是否已被用户关闭（"下次不再提醒"） */
  suspendedModalDismissed: boolean;
  /** CLS 服务状态（active / suspended），与 phase 独立、可叠加 */
  clsStatus: ClsStatus;
  /** CLS 是否已停服（派生：clsStatus === "suspended"） */
  isClsSuspended: boolean;
}

interface ServiceStatusContextType extends ServiceStatusState {
  /** 管控端是否处于禁用态（stopped 段全禁用） */
  isAdminDisabled: boolean;
  /** 用户端"创建 Agent"是否禁用（到期停服 or 席位欠费） */
  isCreateAgentDisabled: boolean;
  /** 创建 Agent 禁用时的提示文案（欠费态含「联系管理员」高亮，故用 ReactNode） */
  createAgentDisabledTip: ReactNode;
  /** 是否处于回收站倒计时阶段（stopped 且 recyclingDaysLeft ≤ 15） */
  isInRecycling: boolean;
  /** 是否处于回收站末段（最后 3 天，D13-15） */
  isInRecyclingFinal: boolean;
  /** 是否处于到期前预警阶段（warning-d1 / d2-6 / critical-d7） */
  isNearingExpire: boolean;

  /** 设置阶段（新 API，推荐使用） */
  setPhase: (phase: ServicePhase) => void;

  /** 设置停服弹窗已关闭 */
  dismissSuspendedModal: () => void;

  /** 模拟：切换管控台状态（旧 API，仍保留兼容） */
  setConsoleStatus: (status: ServiceStatus) => void;
  /** 模拟：设置席位欠费阶段（新 API，推荐使用） */
  setSeatArrearsPhase: (phase: SeatArrearsPhase) => void;
  /** 模拟：切换席位欠费状态（旧 API，true→arrears-buffer / false→active） */
  setSeatArrears: (arrears: boolean) => void;
  /** 模拟：设置回收站剩余天数（旧 API，仍保留兼容） */
  setRecyclingDaysLeft: (days: number | null) => void;
  /** 模拟：设置 CLS 服务状态 */
  setClsStatus: (status: ClsStatus) => void;
}

const ServiceStatusContext = createContext<ServiceStatusContextType | null>(
  null
);

const DISMISS_KEY = "clawpro_suspended_modal_dismissed";

// ─── Provider ────────────────────────────────────────────────────
export function ServiceStatusProvider({ children }: { children: ReactNode }) {
  // 默认展示正常状态；其他阶段可通过右下角计费状态模拟开关手动切换。
  const [phase, setPhaseRaw] = useState<ServicePhase>("active");
  const [seatArrearsPhase, setSeatArrearsPhaseRaw] =
    useState<SeatArrearsPhase>("active");
  const [clsStatus, setClsStatusRaw] = useState<ClsStatus>("active");
  const seatArrears = seatArrearsPhase !== "active";
  const isSeatArrearsBuffer = seatArrearsPhase === "arrears-buffer";
  const isSeatArrearsBlocked = seatArrearsPhase === "arrears";
  const [suspendedModalDismissed, setSuspendedModalDismissed] = useState(() => {
    return localStorage.getItem(DISMISS_KEY) === "true";
  });

  // 阶段派生
  const spec = PHASE_MAP[phase];
  const consoleStatus = spec.status;
  const daysUntilExpire = spec.daysUntilExpire;
  const daysSinceSuspended = spec.daysSinceSuspended;

  const recyclingDaysLeft = useMemo(() => {
    if (consoleStatus !== "suspended") return null;
    if (daysSinceSuspended == null) return null;
    return Math.max(0, RECYCLING_TOTAL_DAYS - daysSinceSuspended);
  }, [consoleStatus, daysSinceSuspended]);

  const isAdminDisabled = consoleStatus === "suspended";
  const isCreateAgentDisabled = isAdminDisabled || seatArrears;
  const isInRecycling =
    isAdminDisabled &&
    recyclingDaysLeft !== null &&
    recyclingDaysLeft <= RECYCLING_TOTAL_DAYS;
  const isInRecyclingFinal = phase === "recycling-d13-15";
  const isNearingExpire =
    phase === "warning-d1" ||
    phase === "warning-d2-6" ||
    phase === "critical-d7";

  // 欠费态：Figma 420_76520「创建额度已用完，请<联系管理员>为您充值后重试」，
  // 「联系管理员」为淡蓝色 #B5C7FF（brand-color-disabled）
  const createAgentDisabledTip: ReactNode = seatArrears ? (
    <>
      创建额度已用完，请
      <span style={{ color: "#B5C7FF" }}>联系管理员</span>
      为您充值后重试
    </>
  ) : isAdminDisabled ? (
    "管控台已到期，无法创建新 Agent，请联系管理员续费"
  ) : (
    ""
  );

  const setPhase = useCallback((next: ServicePhase) => {
    setPhaseRaw(next);
    // 阶段变化时清除"下次不再提醒"标记，便于反复测试
    localStorage.removeItem(DISMISS_KEY);
    setSuspendedModalDismissed(false);
  }, []);

  // 席位欠费阶段切换（新 API）
  const setSeatArrearsPhase = useCallback((next: SeatArrearsPhase) => {
    setSeatArrearsPhaseRaw(next);
  }, []);

  // CLS 服务状态切换（Demo/模拟用）
  const setClsStatus = useCallback((next: ClsStatus) => {
    setClsStatusRaw(next);
  }, []);

  // 席位欠费旧 API：true → 进入缓冲期（催充值可见）；false → 恢复正常
  const setSeatArrears = useCallback((arrears: boolean) => {
    setSeatArrearsPhaseRaw(arrears ? "arrears-buffer" : "active");
  }, []);

  // ─── 兼容旧 API ─────────────────────────────────────────────
  /** 旧 setConsoleStatus：映射为最简阶段（active ↔ suspended-d1） */
  const setConsoleStatus = useCallback(
    (status: ServiceStatus) => {
      setPhase(status === "active" ? "active" : "suspended-d1");
    },
    [setPhase]
  );

  /**
   * 旧 setRecyclingDaysLeft：根据剩余天数反推阶段
   *   剩余 14 天 → suspended-d1（已停服 1 天）
   *   剩余 8~13 天 → suspended-d2-12
   *   剩余 0~2 天 → recycling-d13-15
   */
  const setRecyclingDaysLeft = useCallback(
    (days: number | null) => {
      if (days == null) {
        // 回到"刚停服未进回收站"——保持当前 stopped 阶段但天数置空场景在新模型下等价于 D1
        setPhase("suspended-d1");
        return;
      }
      const usedDays = RECYCLING_TOTAL_DAYS - days;
      if (usedDays <= 1) setPhase("suspended-d1");
      else if (usedDays <= 12) setPhase("suspended-d2-12");
      else setPhase("recycling-d13-15");
    },
    [setPhase]
  );

  const dismissSuspendedModal = useCallback(() => {
    setSuspendedModalDismissed(true);
    localStorage.setItem(DISMISS_KEY, "true");
  }, []);

  return (
    <ServiceStatusContext.Provider
      value={{
        phase,
        consoleStatus,
        seatArrearsPhase,
        seatArrears,
        isSeatArrearsBuffer,
        isSeatArrearsBlocked,
        expireTime: consoleStatus !== "active" ? "2026-08-15" : null,
        daysUntilExpire,
        daysSinceSuspended,
        recyclingDaysLeft,
        suspendedModalDismissed,
        clsStatus,
        isClsSuspended: clsStatus === "suspended",
        isAdminDisabled,
        isCreateAgentDisabled,
        createAgentDisabledTip,
        isInRecycling,
        isInRecyclingFinal,
        isNearingExpire,
        setPhase,
        dismissSuspendedModal,
        setConsoleStatus,
        setSeatArrearsPhase,
        setSeatArrears,
        setRecyclingDaysLeft,
        setClsStatus,
      }}
    >
      {children}
    </ServiceStatusContext.Provider>
  );
}

export function useServiceStatus() {
  const ctx = useContext(ServiceStatusContext);
  if (!ctx) {
    throw new Error(
      "useServiceStatus must be used within ServiceStatusProvider"
    );
  }
  return ctx;
}
