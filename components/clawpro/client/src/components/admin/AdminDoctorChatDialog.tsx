/**
 * AdminDoctorChatPanel - 管控端龙虾医生诊断内嵌面板
 *
 * 诊断状态从服务端（mock API）获取，不依赖 localStorage 存储状态。
 * 核心逻辑：
 * - 打开面板时查询服务端当前 Agent 是否有进行中的诊断
 * - 是自己发起的 → 展示完整对话，可交互/结束
 * - 是别人发起的 → 展示"诊断中"提示，不可操作
 * - 没有诊断 → 展示"开始诊断"按钮
 *
 * 管控端视觉适配：4px 圆角 + claw-primary/claw-outline 按钮 + --cp-* tokens。
 */
import { useState, useEffect, useRef, useCallback } from "react";
import {
  Dialog, DialogContent,
} from "@/components/ui/dialog";

import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { Plus, ArrowUp, Mic } from "lucide-react";
import {
  queryDiagnosisStatus,
  startDiagnosis,
  endDiagnosis,
  pollCreationStatus,
  updateDiagnosisMessages,
  touchSession,
  getCallerId,
  type DoctorMsg,
  type DiagPhase,
  type InstanceStatus,
  type DiagCheckItem,
  type RepairResult,
} from "@/lib/doctorDiagnosisApi";

// 类型从 doctorDiagnosisApi 导入，此处不再重复定义

// ─── 系统消息构造器 ──────────────────────────────────────────────────────────

function doctorMsg(text: string, persistent = false): DoctorMsg {
  return persistent
    ? { kind: "assistant", parts: [{ type: "text", text }] }
    : { kind: "assistant", parts: [{ type: "text", text }], transient: true };
}



// ─── 诊断选项（授权+快照）—— 保留 localStorage 持久化偏好 ─────────────────────

type DiagOptionKey = "authorize" | "snapshot";

const DIAG_OPTION_DEFAULTS: Record<DiagOptionKey, boolean> = {
  authorize: false,
  snapshot: true,
};

function loadDiagOption(agentId: string, key: DiagOptionKey): boolean {
  try {
    const raw = localStorage.getItem(`admin_doctor_diag_${key}_${agentId}`);
    if (raw === null) return DIAG_OPTION_DEFAULTS[key];
    return raw === "true";
  } catch {
    return DIAG_OPTION_DEFAULTS[key];
  }
}

function saveDiagOption(agentId: string, key: DiagOptionKey, enabled: boolean) {
  try {
    localStorage.setItem(`admin_doctor_diag_${key}_${agentId}`, enabled ? "true" : "false");
  } catch {
    // ignore
  }
}

function hasAskedAuth(agentId: string): boolean {
  try {
    return localStorage.getItem(`oc.admin.diag.authAsked.${agentId}`) === "true";
  } catch {
    return false;
  }
}

function markAskedAuth(agentId: string) {
  try {
    localStorage.setItem(`oc.admin.diag.authAsked.${agentId}`, "true");
  } catch {
    // ignore
  }
}

function markAuthorizedDiag(agentId: string) {
  try {
    localStorage.setItem(`oc.admin.diag.authorized.${agentId}`, "true");
  } catch {
    // ignore
  }
}

function useDiagnosisOptions(agentId: string) {
  const [authorize, setAuthorizeRaw] = useState<boolean>(() => loadDiagOption(agentId, "authorize"));
  const [snapshot, setSnapshotRaw] = useState<boolean>(() => loadDiagOption(agentId, "snapshot"));

  const setAuthorize = (v: boolean) => { setAuthorizeRaw(v); saveDiagOption(agentId, "authorize", v); };
  const setSnapshot = (v: boolean) => { setSnapshotRaw(v); saveDiagOption(agentId, "snapshot", v); };

  useEffect(() => {
    setAuthorizeRaw(loadDiagOption(agentId, "authorize"));
    setSnapshotRaw(loadDiagOption(agentId, "snapshot"));
  }, [agentId]);

  return { authorize, setAuthorize, snapshot, setSnapshot } as const;
}



// ─── 常量 ─────────────────────────────────────────────────────────────────────

const AUTO_END_MS = 10 * 60 * 1000;

const AUTO_FIRST_PROMPT =
  "请对当前 Agent 进行全面检测，覆盖所有可检测项目（包括但不限于网络、模型、通道、技能等运行状态），实时输出每项进度和结果，异常项附简短说明。完成后汇总问题列表，逐项给出原因和修复方案，等待我回复后再执行修复。";

const DOCTOR_COMMAND_LIST = [
  { command: "/new", label: "新建会话" },
  { command: "/compact", label: "压缩上下文" },
  { command: "/status", label: "查看状态" },
  { command: "/commands", label: "全部指令" },
];

// ─── 子组件：检测结果列表 ─────────────────────────────────────────────────────

function CheckList({ items }: { items: DiagCheckItem[] }) {
  return (
    <div className="space-y-1 mt-1">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-1.5 text-xs text-gray-900">
          <span
            className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
              item.status === "ok" ? "bg-green-500" :
              item.status === "error" ? "bg-red-500" : "bg-orange-400"
            }`}
          />
          <span className="w-20 flex-shrink-0">{item.label}</span>
          <span className={
            item.status === "error" ? "text-red-600 font-medium" :
            item.status === "warn" ? "text-orange-600 font-medium" :
            ""
          }>
            {item.detail ?? (item.status === "ok" ? "正常" : "异常")}
          </span>
        </div>
      ))}
    </div>
  );
}

// ─── 子组件：修复结果汇总 ─────────────────────────────────────────────────────

function RepairSummary({ results }: { results: RepairResult[] }) {
  const succeed = results.filter((r) => r.ok);
  const failed = results.filter((r) => !r.ok);
  return (
    <div className="space-y-2 mt-1.5">
      {succeed.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-gray-900 mb-1">已成功修复 {succeed.length} 项：</p>
          <div className="space-y-1">
            {succeed.map((r, i) => (
              <div key={i} className="flex items-center gap-1.5 text-xs text-gray-900">
                <span className="w-1.5 h-1.5 rounded-full flex-shrink-0 bg-green-500" />
                <span>{r.label}</span>
                <span className="text-emerald-500">✓</span>
              </div>
            ))}
          </div>
        </div>
      )}
      {failed.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-gray-900 mb-1">{failed.length} 项修复失败：</p>
          <div className="space-y-1">
            {failed.map((r, i) => (
              <div key={i} className="flex items-start gap-1.5 text-xs text-gray-900">
                <span className="w-1.5 h-1.5 rounded-full flex-shrink-0 bg-red-500 mt-1" />
                <div className="min-w-0">
                  <p>{r.label}</p>
                  {r.reason && <p className="text-[10px] text-gray-500 mt-0.5">{r.reason}</p>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── 子组件：打字动画 ─────────────────────────────────────────────────────────

function TypingBubble() {
  return (
    <div className="flex gap-2 py-0.5">
      <div className="flex items-center gap-0.5">
        <span className="w-1 h-1 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: "0ms" }} />
        <span className="w-1 h-1 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: "150ms" }} />
        <span className="w-1 h-1 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: "300ms" }} />
      </div>
    </div>
  );
}

// ─── 子组件：诊断选项勾选行（与用户端 DiagOptionRow 一致） ──────────────────

type DiagOptionCardProps = {
  checked: boolean;
  onChange: (next: boolean) => void;
  title: string;
  description: React.ReactNode;
  /** 紧凑模式，用于小窗内弹窗 */
  compact?: boolean;
};

function DiagOptionRow({ checked, onChange, title, description, compact }: DiagOptionCardProps) {
  // 勾选框尺寸：compact 也保持 16px。原14px 在小padding +缩小字号背景下视觉太弱，
  // 加上未勾选时是 1.5px 灰边框，用户容易看不到"框"（图1 反馈：以为无法勾选）。
  // 统一 16px +4px 圆角（原 5px 近似圆形），保留"复选框"的心智识别。
  const boxSize = 16;
  const checkSize = 10;
  return (
    <label className={`flex items-start cursor-pointer select-none ${compact ? "gap-2 px-2 py-1.5" : "gap-2.5 px-3 py-2.5"}`}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="sr-only"
      />
      <span
        aria-hidden
        className="mt-0.5 flex-shrink-0 inline-flex items-center justify-center rounded-[4px] transition-all"
        style={{
          width: boxSize,
          height: boxSize,
          background: checked ? "var(--cp-brand-blue, #1447E6)" : "#FFFFFF",
          border: checked ? "1px solid var(--cp-brand-blue, #1447E6)" : "1.5px solid #C4C9D4",
          boxShadow: checked ? "none" : "inset 0 0 0 0.5px rgba(0,0,0,0.02)",
        }}
      >
        {checked && (
          <svg width={checkSize} height={checkSize} viewBox="0 0 12 12" fill="none">
            <path d="M2.5 6.2L4.8 8.5L9.5 3.5" stroke="#FFFFFF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </span>
      <div className="flex-1 min-w-0">
        {/* title/description 字号上限对齐外层 MetaText(text-xs=12px)：
            大模式 title 12px / description 11px；紧凑模式再降一档 11/10。
            原大模式用 text-sm(14px) 会超过外层"龙虾医生"标题字号，破坏视觉层级。 */}
        <p className={`text-gray-700 leading-snug ${compact ? "text-[11px]" : "text-[12px]"}`}>{title}</p>
        <p className={`text-gray-400 mt-0.5 leading-relaxed ${compact ? "text-[10px]" : "text-[11px]"}`}>{description}</p>
      </div>
    </label>
  );
}

// ─── Props ────────────────────────────────────────────────────────────────────

export interface AdminDoctorChatPanelProps {
  active: boolean;
  agentInstanceId: string;
  onEnd?: () => void;
}

// ─── 主组件 ──────────────────────────────────────────────────────────────────

export function AdminDoctorChatPanel({
  active,
  agentInstanceId,
  onEnd,
}: AdminDoctorChatPanelProps) {
  // ─── 当前管理员 ID（mock 从 localStorage 获取）────────────────────────────────
  const callerIdRef = useRef(getCallerId("admin"));

  // ─── 状态：实例与诊断阶段 ────────────────────────────────────────────────────
  const [instanceStatus, setInstanceStatus] = useState<InstanceStatus>("none");
  const [diagPhase, setDiagPhase] = useState<DiagPhase>("idle");
  const [doctorInstanceId, setDoctorInstanceId] = useState<string>("");
  const [snapshotCreated, setSnapshotCreated] = useState(false);

  // ─── 状态：是否被他人占用 ──────────────────────────────────────────────────
  const [occupiedByOther, setOccupiedByOther] = useState(false);
  const [occupiedInitiatorType, setOccupiedInitiatorType] = useState<"admin" | "user" | null>(null);

  // ─── 状态：消息 ──────────────────────────────────────────────────────────────
  const [messages, setMessages] = useState<DoctorMsg[]>([]);

  // ─── 状态：输入框与 UI ───────────────────────────────────────────────────────
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);
  const [showCommands, setShowCommands] = useState(false);
  const commandsRef = useRef<HTMLDivElement | null>(null);

  // ─── 状态：弹窗 ──────────────────────────────────────────────────────────────
  const [showStartModal, setShowStartModal] = useState(false);
  const [showEndModal, setShowEndModal] = useState(false);
  const [rollbackChecked, setRollbackChecked] = useState(false);
  // 诊断启动选项（与用户端一致：授权 + 快照两个独立选项）
  const diagOptions = useDiagnosisOptions(agentInstanceId);

  // ─── 全屏对话框 ──────────────────────────────────────────────────────────────
  const [expanded, setExpanded] = useState(false);

  // ─── 容器尺寸监听（自适应弹窗） ────────────────────────────────────────────
  const panelContainerRef = useRef<HTMLDivElement>(null);
  const fullscreenContainerRef = useRef<HTMLDivElement>(null);
  const [containerSize, setContainerSize] = useState<{ width: number; height: number }>({ width: 0, height: 0 });

  // 依赖 showStartModal/showEndModal：弹窗打开的一瞬间强制重挂 observer 并立即测量，
  // 修复"首次打开弹窗时 containerSize.width 还是 0 或首挂时的过小值，导致 getModalStyles
  // 走到 compact 分支，弹窗字号/宽度都偏小；只有用户点过一次放大/缩小按钮触发 expanded
  // 状态变更，effect 才重跑拿到真实宽度"的问题。
  useEffect(() => {
    const target = expanded ? fullscreenContainerRef.current : panelContainerRef.current;
    if (!target) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width, height } = entry.contentRect;
        if (width > 0 || height > 0) {
          setContainerSize({ width, height });
        }
      }
    });
    observer.observe(target);

    // 初始测量：立即量一次 + rAF 延后一帧再量一次，
    // 覆盖"首挂时抽屉/Dialog 还未布局完成，getBoundingClientRect 返回 0"的边缘场景。
    const measure = () => {
      const el = expanded ? fullscreenContainerRef.current : panelContainerRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      if (rect.width > 0 || rect.height > 0) {
        setContainerSize({ width: rect.width, height: rect.height });
      }
    };
    measure();
    const rafId = requestAnimationFrame(measure);

    return () => {
      cancelAnimationFrame(rafId);
      observer.disconnect();
    };
  }, [expanded, showStartModal, showEndModal]);

  // ─── 自动滚动，让龙虾医生区块 / 弹窗 完整进入抽屉视口 ─────────────────────
  // 背景：管控端右侧抽屉页较长，"龙虾医生"区块位置偏下。存在三个视觉截断场景：
  //   ① 点「开始诊断」后 active=true，本组件挂载并向下扩展出对话/交互面板；
  //      新增内容处在抽屉视口之下，看不见，必须手动下拉抽屉。
  //   ② 授权弹窗（renderStartModal）与结束确认弹窗（renderEndModal）以
  //      `absolute inset-0` 铺在 panelContainerRef 内，弹窗高度较大时，
  //      Panel 底部（含弹窗底部按钮）超出抽屉视口，仍需手动下拉才能看全。
  //   ③ 结束诊断后 active变 false → 组件 return null，父组件回到
  //      "tip + 开始诊断"静态视图；但抽屉滚动位置停留在下拉后的位置，没有
  //      回到"点击开始诊断前"的视觉起点，视觉上没有"回到最初状态"。
  //
  // 三合一策略：
  //   · 弹窗打开时（内嵌）→ 滚 modalRef 到视口中央（block:"center"），
  //     保证标题 + 选项 + 按钮全部同屏可见。
  //   · 仅 active（无弹窗）→ 滚区块 root 到视口顶（block:"start"），
  //     让"龙虾医生"标题 + 说明文案 + 新展开面板整体入视。
  //   · active 从 true→false（结束诊断）→ 同样滚区块 root 到视口顶，
  //     使静态"tip + 开始诊断按钮"回到用户点击前的视觉位置。
  //
  // 说明：
  //   · 仅内嵌模式生效（expanded=false），全屏 Dialog 自带居中。
  //   · rAF 延后一帧，等 DOM 挂载完成后再滚，避免拿到旧布局。
  //   · behavior:"smooth" 保证视觉过渡自然。
  const modalRef = useRef<HTMLDivElement | null>(null);
  const prevActiveRef = useRef(active);
  useEffect(() => {
    if (expanded) {
      prevActiveRef.current = active;
      return;
    }
    const wasActive = prevActiveRef.current;
    prevActiveRef.current = active;

    // active 从 true→false（结束诊断）时组件即将 return null，此帧尚未 unmount，
    // panelContainerRef 仍可用，向上 closest 依然能拿到"龙虾医生"区块根。
    const blockEl = panelContainerRef.current?.closest<HTMLElement>("[data-doctor-block]");

    const rafId = requestAnimationFrame(() => {
      if (active && (showStartModal || showEndModal) && modalRef.current) {
        // ① 弹窗打开时：把弹窗自身滚到视口中央
        modalRef.current.scrollIntoView({ behavior: "smooth", block: "center" });
        return;
      }
      if (active && !wasActive) {
        // ② 刚激活（点开始诊断）：滚区块顶到视口顶
        (blockEl ?? panelContainerRef.current)?.scrollIntoView({
          behavior: "smooth",
          block: "start",
        });
        return;
      }
      if (!active && wasActive) {
        // ③ 结束诊断：滚区块顶到视口顶，回到点击前的视觉起点
        blockEl?.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
    return () => cancelAnimationFrame(rafId);
  }, [active, expanded, showStartModal, showEndModal]);

  // 根据容器大小计算弹窗自适应样式

  // 根据容器大小计算弹窗自适应样式
  const getModalStyles = useCallback(() => {
    const { width } = containerSize;
    // ─── 自适应阈值调整 ─────────────────────────────────────────────────────
    // 管控端右侧抽屉宽度约 400~480px，原阈值 500 让它永远走compact 分支，
    // 弹窗被强制缩到 maxWidth=280+ 字号 0.85×，视觉上"很小、无法随外部框长大"。
    // 降到 360 让抽屉里的弹窗也能获得"大模式"的正常字号与更充分的宽度，
    // 只有极窄侧栏（手机竖屏级）才继续走紧凑模式。
    const isLarge = expanded || width > 360;
    // 弹窗宽度：追随容器宽度成比例放大 —— 内嵌大模式 88% 容器宽（上限 480），
    // 紧凑模式 90% 容器宽（上限 320），避免出现"容器宽 460而弹窗永远只有 280"的失衡。
    const maxW = Math.max(
      240,
      isLarge? Math.min(480, width * 0.88) : Math.min(320, width * 0.9),
    );
    // ─── 字号上限：与外层"龙虾医生"标题/说明文案对齐 ────────────────────────
    // 外层区块（OpenClawMonitor.tsx#6763/6780）里：
    //   · "龙虾医生"标题：<MetaText font-medium> = text-xs(12px) + medium
    //   · "AI 智能诊断…"说明：<MetaText tone=weak text-xs> = 12px normal
    // 弹窗是该区块内的二级信息，**任何文字都不应超过 12px**，否则出现
    // "弹窗标题比它所属区块标题还大"的层级错乱。
    // 因此这里改为「绝对像素上限」而非「按scale 缩放」：
    //   ·弹窗内标题（开始/结束诊断）        → 12px 与外层标题等大
    //   · 弹窗内正文（描述、DiagOptionRow）  → 12px 与说明文案等大
    //   · 弹窗内辅助/hint（次级说明、按钮）  → 11px 明确弱于正文
    // 紧凑模式再降一档到 11/11/10，保证任何容器下都 ≤ 12px。
    const scale = isLarge ? 1 : 0.92; // 保留供 padding/gap 使用
    return {
      maxWidth: maxW,
      titleSize: isLarge ? 12 : 11,
      bodySize: isLarge ? 12 : 11,
      hintSize: isLarge ? 11 : 10,
      btnSize: isLarge ? 12 : 11,
      btnHeight: isLarge ? 30 : 28,
      padding: isLarge ? 18 : 12,
      gap: isLarge ? 10 : 8,
      compact: !isLarge,
      isLarge,
      scale,
    };
  }, [containerSize, expanded]);

  // ─── 状态：自动结束计时器 ────────────────────────────────────────────────────
  const autoEndTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ─── 守卫 ────────────────────────────────────────────────────────────────────
  const didAutoSendRef = useRef(false);

  // ─── refs ────────────────────────────────────────────────────────────────────
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // ─── 滚动到底部 ──────────────────────────────────────────────────────────────
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping, scrollToBottom]);

  // ─── 同步消息到服务端（替代 localStorage 缓存）─────────────────────────────
  useEffect(() => {
    if (instanceStatus !== "active" || !agentInstanceId) return;
    if (messages.length === 0) return;
    updateDiagnosisMessages(agentInstanceId, messages, diagPhase);
  }, [messages, diagPhase, instanceStatus, agentInstanceId]);

  // ─── 初始化：active 时从服务端查询诊断状态 ─────────────────────────────────
  useEffect(() => {
    if (!active || !agentInstanceId) return;

    const checkStatus = async () => {
      const result = await queryDiagnosisStatus(agentInstanceId, callerIdRef.current);

      if (result.active && result.session) {
        if (result.isMine) {
          // 是自己发起的诊断 → 恢复会话
          setOccupiedByOther(false);
          setDoctorInstanceId(result.session.doctorInstanceId);
          setInstanceStatus(result.session.status);
          setDiagPhase(result.session.phase);
          didAutoSendRef.current = true;
          if (result.session.messages.length > 0) {
            setMessages(result.session.messages);
          }
          restartAutoEndTimer();
        } else {
          // 别人发起的诊断 → 展示"诊断中"提示
          setOccupiedByOther(true);
          setOccupiedInitiatorType(result.session.initiatorType);
          setInstanceStatus("none");
        }
      } else {
        // 没有进行中的诊断 → 弹出确认弹窗
        setOccupiedByOther(false);
        setInstanceStatus("none");
        handleStartDiagnosisClick();
      }
    };

    checkStatus();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, agentInstanceId]);

  // ─── 10 分钟无操作自动结束 ──────────────────────────────────────────────────
  const restartAutoEndTimer = useCallback(() => {
    if (autoEndTimerRef.current) clearTimeout(autoEndTimerRef.current);
    autoEndTimerRef.current = setTimeout(() => {
      setMessages((prev) => [
        ...prev,
        doctorMsg("您已超过 10 分钟未操作，本次诊断已自动结束。如需继续排查，请重新点击「开始诊断」。", true),
      ]);
      doEndSession(false);
    }, AUTO_END_MS);
  }, []);

  // 组件卸载清理计时器
  useEffect(() => {
    return () => {
      if (autoEndTimerRef.current) clearTimeout(autoEndTimerRef.current);
    };
  }, []);

  // ─── 结束会话（调用服务端接口）──────────────────────────────────────────────
  const doEndSession = (rollback: boolean) => {
    if (autoEndTimerRef.current) clearTimeout(autoEndTimerRef.current);

    // 调用服务端结束接口
    endDiagnosis({ agentInstanceId, doctorInstanceId, rollback });

    if (rollback) {
      toast.success("已回滚到诊断前配置快照");
    }
    toast.info("诊断已结束");

    // ─── 重置为「最初静态」───────────────────────────────────────────────
    // 结束后组件回到抽屉的"未启动"状态：只保留父级"龙虾医生"tip + 「开始诊断」按钮，
    // 面板本身（消息、输入框、弹窗、全屏态）全部清零。避免残留状态导致下一次
    // 「开始诊断」时对话历史/确认弹窗错位显示。
    setMessages([]);
    setInput("");
    setIsTyping(false);
    setIsStreaming(false);
    setShowCommands(false);
    setShowStartModal(false);
    setShowEndModal(false);
    setRollbackChecked(false);
    setExpanded(false);
    setDoctorInstanceId("");
    setSnapshotCreated(false);
    setOccupiedByOther(false);
    setOccupiedInitiatorType(null);
    setInstanceStatus("none");
    setDiagPhase("idle");
    didAutoSendRef.current = false;

    // ─── 视觉复位：滚回"龙虾医生"区块顶部 ────────────────────────────────
    // 结束诊断后本组件会立即因 active=false 而 return null，effect 无法再触发
    // 滚动。这里在 onEnd() 之前同步捕获区块 DOM 节点，然后 requestAnimationFrame
    // 里滚动——rAF 回调运行时组件虽已 unmount，但 blockEl（父组件的节点）依然
    // 存在于 DOM，滚动照常生效，让抽屉视口回到"点击开始诊断前"的视觉起点。
    const blockEl = panelContainerRef.current?.closest<HTMLElement>("[data-doctor-block]");
    if (blockEl) {
      requestAnimationFrame(() => {
        blockEl.scrollIntoView({ behavior: "smooth", block: "start" });
      });
    }

    // 通知父组件收起面板（父组件会setDoctorActive(false)）；
    // 组件顶层的 active guard 会让本组件立即返回 null，抽屉即回到初始态。
    onEnd?.();
  };

  // ─── 「开始诊断」入口：弹授权+快照确认弹窗（与用户端一致） ─────────────────
  const handleStartDiagnosisClick = () => {
    // 强制回到内嵌小窗：全屏应仅由右上角「扩大」按钮显式触发，
    // 避免上次残留的 expanded=true 导致 confirm modal 弹在全屏面板里。
    setExpanded(false);
    // 默认勾选策略：每次打开弹窗重置为安全态（都勾选）
    diagOptions.setAuthorize(true);
    diagOptions.setSnapshot(true);
    setShowStartModal(true);
  };

  // ─── 弹窗内点「确认开始」：调用服务端创建诊断实例 ──────────────────────────
  const handleStartConfirm = async () => {
    setShowStartModal(false);
    // 选项 1：配置快照（独立）
    setSnapshotCreated(diagOptions.snapshot);
    // 选项 2：授权（独立）—— 一次性授权语义
    if (diagOptions.authorize) {
      markAuthorizedDiag(agentInstanceId);
    }
    markAskedAuth(agentInstanceId);

    // 调用服务端创建接口
    const result = await startDiagnosis({
      agentInstanceId,
      initiatorId: callerIdRef.current,
      initiatorType: "admin",
      snapshot: diagOptions.snapshot,
    });

    if (!result.success) {
      if (result.reason === "conflict") {
        toast.error("该 Agent 当前正在诊断中，请稍后再试");
        setOccupiedByOther(true);
      }
      onEnd?.();
      return;
    }

    // 创建成功
    setDoctorInstanceId(result.doctorInstanceId!);
    setInstanceStatus("creating");
    setDiagPhase("idle");
    didAutoSendRef.current = false;
    setMessages([]);
    startCreatePolling();
  };

  // ─── 弹窗取消：通知外层诊断未开始 ──────────────────────────────────────────
  const handleStartCancel = () => {
    setShowStartModal(false);
    onEnd?.();
  };

  // ─── 轮询创建状态（等待实例就绪）─────────────────────────────────────────────
  const startCreatePolling = () => {
    const poll = async () => {
      const { ready } = await pollCreationStatus(agentInstanceId);
      if (ready) {
        setInstanceStatus("active");
        if (!didAutoSendRef.current) {
          setTimeout(() => autoSendFirstPrompt(), 400);
        } else {
          setMessages((prev) => [
            ...prev,
            doctorMsg("龙虾医生已准备就绪，您可以继续与我对话了。"),
          ]);
        }
        restartAutoEndTimer();
      } else {
        setTimeout(poll, 500);
      }
    };
    setTimeout(poll, 1500);
  };

  // ─── 自动注入第一条提示词 ──────────────────────────────────────────────────
  const autoSendFirstPrompt = () => {
    if (didAutoSendRef.current) return;
    didAutoSendRef.current = true;
    setMessages((prev) => [...prev, { kind: "user", text: AUTO_FIRST_PROMPT }]);
    setDiagPhase("diagnosing");
    setIsTyping(true);
    // MOCK 龙虾医生检测响应
    setTimeout(() => {
      setMessages((prev) => [
        ...prev,
        {
          kind: "assistant",
          parts: [
            { type: "text", text: "您好，我是龙虾医生 🦞 收到，开始全面检测，将逐项实时输出结果…" },
          ],
        },
      ]);
    }, 600);
    setTimeout(() => {
      setIsTyping(false);
      setMessages((prev) => [
        ...prev,
        {
          kind: "assistant",
          parts: [
            { type: "text", text: "全部检测项执行完成，共 8 项，发现 3 项异常：" },
            {
              type: "check_list",
              items: [
                { label: "网络连接", status: "ok" },
                { label: "模型接口", status: "ok" },
                { label: "IM 通道", status: "error", detail: "飞书 Token 已过期" },
                { label: "技能运行状态", status: "ok" },
                { label: "MCP 工具连接", status: "error", detail: "tavily-search 进程崩溃" },
                { label: "文件系统权限", status: "ok" },
                { label: "环境变量配置", status: "warn", detail: "OPENAI_API_KEY 即将过期" },
                { label: "Plugin 加载", status: "ok" },
              ],
            },
            { type: "text", text: "异常项原因与修复方案：\n1. IM 通道（飞书）—— Token 已过期；建议刷新通道授权 Token。\n2. MCP 工具（tavily-search）—— 进程已崩溃；建议重启技能进程并校验本地端口占用。\n3. 环境变量配置 —— OPENAI_API_KEY 即将过期；建议更新 API Key。\n\n如需继续，请直接回复「开始修复」或描述您希望优先处理的问题。" },
          ],
        },
      ]);
      setDiagPhase("summary_ready");
    }, 2400);
  };

  // ─── 进入修复阶段（改为纯对话触发） ──────────────────────────────────────
  const handleStartRepair = (userText = "请按问题列表依次修复每个问题，每完成一项告知具体执行结果。全部处理完毕后，汇总输出本次修复的成功项和失败项列表。") => {
    if (instanceStatus !== "active") {
      toast.error("当前诊断已结束，请重新开始诊断");
      return;
    }
    restartAutoEndTimer();
    setMessages((prev) => [
      ...prev,
      { kind: "user", text: userText },
    ]);
    setDiagPhase("repairing");
    setIsTyping(true);
    // MOCK 修复结果
    setTimeout(() => {
      setIsTyping(false);
      setMessages((prev) => [
        ...prev,
        {
          kind: "assistant",
          parts: [
            { type: "text", text: "修复已完成，以下是本次修复汇总：" },
            {
              type: "repair_summary",
              results: [
                { label: "飞书通道 Token 刷新", ok: true },
                { label: "tavily-search 进程重启", ok: true },
                { label: "OPENAI_API_KEY 更新", ok: false, reason: "需要管理员手动更新密钥，自动修复权限不足" },
              ],
            },
            { type: "text", text: "2 项已成功修复，1 项需要您手动处理。如有其他问题可以继续向我提问。" },
          ],
        },
      ]);
      setDiagPhase("done");
    }, 2000);
  };

  // ─── 发送消息 / AI 调用 ───────────────────────────────────────────────────
  const callAI = async (text: string) => {
    setIsTyping(true);
    setIsStreaming(true);
    // 插入 loading 占位
    setMessages((prev) => [
      ...prev,
      { kind: "assistant", parts: [{ type: "text", text: "…" }], loading: true },
    ]);
    const controller = new AbortController();
    abortControllerRef.current = controller;
    try {
      // MOCK: 实际应调用后端 POST /api/admin/doctor/chat
      await new Promise((resolve) => setTimeout(resolve, 1200));
      if (controller.signal.aborted) throw new DOMException("", "AbortError");
      const mockReply = "收到您的问题，让我来分析一下。根据当前 Agent 的运行状态来看，这个问题可能与配置项有关。建议您检查相关环境变量是否正确设置。如需进一步排查，请提供更多信息。";
      setMessages((prev) => {
        const updated = [...prev];
        const last = updated[updated.length - 1];
        if (last.kind === "assistant" && last.loading) {
          updated[updated.length - 1] = { kind: "assistant", parts: [{ type: "text", text: mockReply }] };
        }
        return updated;
      });
    } catch (err: unknown) {
      if (!(err instanceof DOMException && err.name === "AbortError")) {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[updated.length - 1];
          if (last && last.kind === "assistant" && last.loading) {
            updated[updated.length - 1] = {
              kind: "assistant",
              parts: [{ type: "text", text: "抱歉，我暂时无法回复，请稍后再试。" }],
            };
            return updated;
          }
          return [
            ...prev,
            { kind: "assistant", parts: [{ type: "text", text: "抱歉，我暂时无法回复，请稍后再试。" }] },
          ];
        });
      } else {
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last && last.kind === "assistant" && last.loading) {
            return prev.slice(0, -1);
          }
          return prev;
        });
      }
    } finally {
      setIsStreaming(false);
      setIsTyping(false);
      abortControllerRef.current = null;
    }
  };

  // ─── 指令处理 ──────────────────────────────────────────────────────────────
  const handleDoctorCommand = (cmd: string): boolean => {
    if (cmd === "/new") {
      setMessages([
        doctorMsg("好的，已为您新建会话，之前的对话上下文已清空，我们重新开始吧！"),
      ]);
      didAutoSendRef.current = false;
      setTimeout(() => autoSendFirstPrompt(), 200);
      return true;
    }
    if (cmd === "/compact") {
      setMessages((prev) => [
        ...prev,
        doctorMsg("好的，我已为您压缩了对话上下文，仅保留最近 10 条对话作为后续推理依据。"),
      ]);
      return true;
    }
    if (cmd === "/status") {
      const statusLabel = ({
        none: "未启动", creating: "创建中", active: "运行中",
        destroying: "销毁中", ended: "已结束",
      } as const)[instanceStatus];
      const phaseLabel = ({
        idle: "待诊断", diagnosing: "检测中", summary_ready: "等待确认修复",
        repairing: "修复中", done: "自由对话中",
      } as const)[diagPhase];
      setMessages((prev) => [
        ...prev,
        doctorMsg(`当前我的运行状态是「${statusLabel}」，诊断阶段处于「${phaseLabel}」。`),
      ]);
      return true;
    }
    if (cmd === "/commands") {
      const list = DOCTOR_COMMAND_LIST.map((x) => `${x.command} — ${x.label}`).join("\n");
      setMessages((prev) => [
        ...prev,
        doctorMsg(`这是您当前可以使用的指令清单：\n${list}`),
      ]);
      return true;
    }
    return false;
  };

  const handleSend = () => {
    const text = input.trim();
    if (!text || inputDisabled) return;
    setInput("");
    if (textareaRef.current) textareaRef.current.style.height = "auto";
    const firstWord = text.split(/\s+/)[0];
    const isCommand = DOCTOR_COMMAND_LIST.some((x) => x.command === firstWord);
    const isRepairIntent =
      diagPhase === "summary_ready" &&
      /(开始修复|执行修复|确认修复|继续修复|先修复|修复吧|请修复|修复)/.test(text);

    if (isRepairIntent) {
      handleStartRepair(text);
      return;
    }

    setMessages((prev) => [...prev, { kind: "user", text }]);
    restartAutoEndTimer();
    touchSession(agentInstanceId); // 通知服务端有用户操作
    if (isCommand && handleDoctorCommand(firstWord)) return;
    callAI(text);
  };

  const handleStopStreaming = () => {
    abortControllerRef.current?.abort();
    setIsStreaming(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    // "/" 触发指令列表
    if (e.key === "/" && input === "") {
      setShowCommands(true);
    }
  };

  // ─── 结束诊断按钮 ──────────────────────────────────────────────────────────
  const handleEndClick = () => {
    setShowEndModal(true);
    setRollbackChecked(false);
  };

  const handleEndConfirm = () => {
    setShowEndModal(false);
    const shouldRollback = rollbackChecked && snapshotCreated;
    setMessages((prev) => [
      ...prev,
      doctorMsg(
        shouldRollback
          ? "本次诊断已结束，配置已回滚到诊断前快照。龙虾医生将离场，感谢使用！"
          : "本次诊断已结束。龙虾医生将离场，感谢使用！",
        true
      ),
    ]);
    doEndSession(shouldRollback);
  };

  // ─── 计算输入框状态 ────────────────────────────────────────────────────────
  const inputDisabled =
    instanceStatus !== "active" ||
    isStreaming ||
    diagPhase === "diagnosing" ||
    diagPhase === "repairing";

  const inputPlaceholder =
    instanceStatus === "none" ? "龙虾医生即将就位…" :
    instanceStatus === "creating" ? "龙虾医生正在就位…" :
    instanceStatus === "ended" ? "当前诊断已结束" :
    instanceStatus === "destroying" ? "龙虾医生正在离场…" :
    diagPhase === "diagnosing" ? "龙虾医生检测中，请稍候…" :
    diagPhase === "repairing" ? "正在执行修复…" :
    "向龙虾医生提问，或描述您遇到的问题…";

  // ─── 渲染单条消息 ──────────────────────────────────────────────────────────
  const renderMsg = (msg: DoctorMsg, idx: number) => {
    if (msg.kind === "system") {
      return (
        <div key={idx} className="flex justify-center">
          <span className="px-2 py-0.5 rounded-full bg-gray-100 text-[10px] text-gray-400">{msg.text}</span>
        </div>
      );
    }
    if (msg.kind === "user") {
      return (
        <div key={idx} className="flex justify-end">
          <div className="max-w-[78%] px-3 py-1.5 rounded-[var(--radius-lg)] bg-gray-100 text-xs text-gray-900 leading-relaxed whitespace-pre-wrap">
            {msg.text}
          </div>
        </div>
      );
    }
    // assistant
    return (
      <div key={idx} className="flex justify-start">
        <div className="max-w-[90%] text-xs text-gray-900 leading-relaxed whitespace-pre-wrap">
          {msg.loading ? (
            <TypingBubble />
          ) : (
            msg.parts.map((part, pi) => {
              if (part.type === "text") return <p key={pi} className="mb-0.5 last:mb-0">{part.text}</p>;
              if (part.type === "check_list") return <CheckList key={pi} items={part.items} />;
              if (part.type === "repair_summary") return <RepairSummary key={pi} results={part.results} />;
              return null;
            })
          )}
        </div>
      </div>
    );
  };

  // ─── 自动调整 textarea 高度 ────────────────────────────────────────────────
  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    // 指令弹层
    if (e.target.value === "/") {
      setShowCommands(true);
    } else if (!e.target.value.startsWith("/")) {
      setShowCommands(false);
    }
    // 自动高度
    e.target.style.height = "auto";
    e.target.style.height = Math.min(e.target.scrollHeight, 80) + "px";
  };

  // ─── 对话面板内容（内嵌模式和全屏共用） ──────────────────────────────────────
  const chatContent = (isFullscreen: boolean) => {
    const msgPadding = isFullscreen ? "px-5 py-4" : "px-3 py-3";

    // 布局策略：
    // ① 全屏模式（Dialog 内）：容器 h-full +消息区 flex-1 overflow-y-auto → 双区固定，内部滚动
    // ② 内嵌模式（右侧抽屉内）：不再限制固定高度（原为 h-[320px]），
    //    让消息区随诊断内容自然向下延展，由抽屉本身承担滚动（单一滚动条）。
    //    这样"开始诊断"后交互面板会自动下拉、完整展示，避免嵌套滚动导致视觉截断。
    //    极端场景（消息数百条）时用 max-h-[70vh] 兜底，避免把抽屉页面顶到失控高度。
    const containerCls = isFullscreen
      ? "flex flex-col h-full overflow-hidden"
      : "flex flex-col";
    const messageAreaCls = isFullscreen
      ? `flex-1 overflow-y-auto ${msgPadding} space-y-2`
      : `${msgPadding} space-y-2 max-h-[70vh] overflow-y-auto`;

    return (
      <div className={containerCls}>
        {/* 消息区 —— 无边框 */}
        <div className={messageAreaCls}>
          {messages.map((msg, idx) => renderMsg(msg, idx))}
          {isTyping && !messages.some((m) => m.kind === "assistant" && m.loading) && (
            <div className="flex justify-start">
              <TypingBubble />
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* 输入区 —— 用户端风格：圆角卡片 + ActionBar */}
        <div className="flex-shrink-0" style={{ padding: isFullscreen ? "0 16px 16px" : "0 12px 12px" }}>
          <div className="relative">
            {/* 指令列表浮层 */}
            {showCommands && instanceStatus === "active" && (
              <div
                ref={commandsRef}
                className="absolute bottom-full mb-1 left-0 w-52 rounded-[12px] border border-[#E9ECF1] bg-white py-1 z-10"
                style={{ boxShadow: "0 4px 12px 0 rgba(0,0,0,0.08)" }}
              >
                {DOCTOR_COMMAND_LIST.map((item) => (
                  <button
                    key={item.command}
                    type="button"
                    onClick={() => {
                      setInput(item.command + " ");
                      setShowCommands(false);
                      textareaRef.current?.focus();
                    }}
                    className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-gray-50 transition-colors text-left"
                  >
                    <span className="text-xs font-mono text-gray-900">{item.command}</span>
                    <span className="text-[11px] text-gray-400">{item.label}</span>
                  </button>
                ))}
              </div>
            )}

            {/* 输入卡片：圆角 20、stroke #E9ECF1、shadow */}
            <div
              className="bg-white relative"
              style={{
                borderRadius: 20,
                border: "1px solid #E9ECF1",
                boxShadow: "0 4px 12px 0 rgba(0,0,0,0.04)",
              }}
            >
              {/* textarea 区域 */}
              <div style={{ padding: isFullscreen ? "14px 18px" : "12px 16px", minHeight: isFullscreen ? 64 : 48 }}>
                <textarea
                  ref={textareaRef}
                  value={input}
                  onChange={handleTextareaChange}
                  onKeyDown={handleKeyDown}
                  disabled={inputDisabled}
                  placeholder={inputPlaceholder}
                  className="w-full resize-none focus:outline-none bg-transparent disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{
                    fontSize: isFullscreen ? 14 : 13,
                    lineHeight: "22px",
                    color: "#0A0A0A",
                    minHeight: isFullscreen ? 36 : 24,
                    maxHeight: 80,
                  }}
                  rows={1}
                />
              </div>

              {/* ActionBar：+ | 指令库 | 麦克风 ... 发送 */}
              <div
                className="flex items-center justify-between"
                style={{ padding: isFullscreen ? "0 14px 12px" : "0 12px 10px" }}
              >
                <div className="flex items-center" style={{ gap: 6 }}>
                  {/* + 按钮 */}
                  <button
                    type="button"
                    aria-label="附件"
                    className="flex items-center justify-center text-[#737373] hover:text-[#0A0A0A] hover:bg-[#F5F6F9] active:scale-90 transition-all rounded-full"
                    style={{ width: 28, height: 28 }}
                  >
                    <Plus className="h-4 w-4" />
                  </button>
                  {/* 分割线 */}
                  <span className="w-px h-3.5 bg-[#E5E5E5]" />
                  {/* 指令库 pill */}
                  <button
                    type="button"
                    onClick={() => setShowCommands(!showCommands)}
                    className="inline-flex items-center hover:border-[#1447E6]/40 active:scale-[0.97] transition-all"
                    style={{
                      borderRadius: 16,
                      border: "1px solid #E9ECF1",
                      padding: "4px 10px",
                      gap: 3,
                    }}
                  >
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: "rgba(0,0,0,0.7)" }}>
                      <rect x="3" y="3" width="10" height="10" rx="2" />
                      <path d="M6 7h4M6 9h4" />
                    </svg>
                    <span style={{ fontSize: 12, lineHeight: "18px", color: "rgba(0,0,0,0.9)" }}>
                      指令库
                    </span>
                  </button>
                  {/* 麦克风 */}
                  <button
                    type="button"
                    aria-label="语音输入"
                    className="flex items-center justify-center text-[#737373] hover:text-[#0A0A0A] hover:bg-[#F5F6F9] active:scale-90 transition-all rounded-full"
                    style={{ width: 28, height: 28 }}
                  >
                    <Mic className="h-4 w-4" />
                  </button>
                </div>

                {/* 右侧：发送/暂停 */}
                <div className="flex items-center" style={{ gap: 6 }}>
                  {isStreaming ? (
                    <button
                      onClick={handleStopStreaming}
                      aria-label="暂停输出"
                      className="flex items-center justify-center text-white rounded-full transition-all hover:opacity-90"
                      style={{ width: 28, height: 28, background: "rgba(0,0,0,0.92)" }}
                    >
                      <svg width="10" height="10" viewBox="0 0 12 12" fill="white">
                        <rect x="2" y="1.5" width="3" height="9" rx="1" />
                        <rect x="7" y="1.5" width="3" height="9" rx="1" />
                      </svg>
                    </button>
                  ) : (
                    <button
                      onClick={handleSend}
                      disabled={inputDisabled || !input.trim()}
                      aria-label="发送"
                      className="flex items-center justify-center text-white rounded-full transition-all duration-150 disabled:opacity-30 disabled:cursor-not-allowed"
                      style={{
                        width: 28,
                        height: 28,
                        background: input.trim() && !inputDisabled ? "rgba(0,0,0,0.92)" : "#d1d5db",
                      }}
                    >
                      <ArrowUp className="h-4 w-4" strokeWidth={2.5} />
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  // ─── 监听外层放大按钮事件 ─────────────────────────────────────────────────
  useEffect(() => {
    const handler = () => setExpanded((prev) => !prev);
    window.addEventListener("admin-doctor-toggle-expand", handler);
    return () => window.removeEventListener("admin-doctor-toggle-expand", handler);
  }, []);

  if (!active) return null;

  // ─── 被他人占用时，展示"诊断中"提示 ─────────────────────────────────────────
  if (occupiedByOther) {
    const isUserInitiated = occupiedInitiatorType === "user";
    return (
      <div className="mt-3 p-4 rounded-[var(--radius-lg)] border border-[#E9ECF1] bg-[#FAFBFC]">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-orange-400 animate-pulse" />
          <span className="text-sm text-gray-700 font-medium">该 Agent 当前正在诊断中</span>
        </div>
        <p className="mt-2 text-xs text-gray-500 leading-relaxed">
          {isUserInitiated
            ? "当前有用户正在对该 Agent 进行诊断，请等待诊断完成后再操作。"
            : "当前有其他管理员正在对该 Agent 进行诊断，请等待诊断完成后再操作。"}
        </p>
      </div>
    );
  }

  // ─── 自适应弹窗渲染器 ─────────────────────────────────────────────────────
  const renderStartModal = () => {
    const ms = getModalStyles();
    return (
      <div className="absolute inset-0 z-50 flex items-center justify-center rounded-[var(--radius-lg)] overflow-hidden">
        {/* 遮罩层 */}
        <div className="absolute inset-0 bg-black/30 rounded-[var(--radius-lg)]" onClick={handleStartCancel} />
        {/* 弹窗内容 —— 尺寸根据容器动态计算 */}
        <div
          ref={modalRef}
          className="relative z-10 bg-white rounded-[8px] shadow-[0_4px_12px_rgba(0,0,0,0.12)] animate-in fade-in zoom-in-95 duration-150"
          style={{
            width: "calc(100% - 24px)",
            maxWidth: ms.maxWidth,
            padding: ms.padding,
            // 兜底：极窄容器/未来内容膨胀时，弹窗自身可上下滚动，
            // 保证标题和按钮永远可见，绝不再被父容器 overflow-hidden 裁掉。
            maxHeight: "calc(100% - 16px)",
            overflowY: "auto",
          }}
        >
          <h3
            className="font-semibold text-gray-900 leading-tight"
            style={{ fontSize: ms.titleSize }}
          >
            开始诊断
          </h3>
          <div style={{ marginTop: ms.gap, display: "flex", flexDirection: "column", gap: ms.gap }}>
            <p className="text-gray-600 leading-relaxed" style={{ fontSize: ms.bodySize }}>
              即将创建龙虾医生 Agent，使用平台配置的诊断模型对当前 Agent 进行全面检测和修复。
            </p>

            {/* 授权 + 配置快照 合并到同一个气泡框中 */}
            <div
              className="rounded-[4px]"
              style={{ border: "1px solid #EDEFF5", background: "#FFFFFF" }}
            >
              {!hasAskedAuth(agentInstanceId) && (
                <DiagOptionRow
                  checked={diagOptions.authorize}
                  onChange={diagOptions.setAuthorize}
                  title="同意使用龙虾医生功能"
                  description="龙虾医生将在当前 Agent 上创建临时诊断节点，诊断结束后自动销毁。"
                  compact={ms.compact}
                />
              )}
              {/*
                网关重启影响（2026-08-04 实测）：创建快照触发 gateway 冷替换，
                对外服务中断约 16.4 秒（78 条 connection refused）。
                详见 .codebuddy/notes/gateway-restart-impact-2026-08-04.md；
                本处与结束诊断的回滚项统一使用「短暂中断」的保守表述。
              */}
              <DiagOptionRow
                checked={diagOptions.snapshot}
                onChange={diagOptions.setSnapshot}
                title="为本次诊断创建配置快照"
                description={
                  <>
                    勾选后，结束诊断时可一键回滚至 Agent 开始诊断前的状态。
                    <span className="block mt-1 text-gray-400">
                      创建快照会触发网关重启，期间该 Agent 对外服务将短暂中断。
                    </span>
                  </>
                }
                compact={ms.compact}
              />
            </div>

            <p className="text-gray-400 leading-relaxed" style={{ fontSize: ms.hintSize }}>初始化约需 3-5 分钟，请稍作等待。</p>
          </div>
          {/*
            按钮组视觉对齐用户端「开始诊断」弹窗（tenant 域）：
            ① 位置：justify-end 右下对齐（与用户端 OpenClawDetail.tsx 6108 行同款布局）
            ② 顺序：取消（左） → 确认开始（右），确认放在鼠标最终落点位置，符合表单主流约定
            ③ 形状：!rounded-full 胶囊形——与用户端 tenant-primary/tenant-outline 视觉一致
               这里刻意用className 局部覆盖 claw-* 变体的 4px 圆角，因为「龙虾医生」
               弹窗 UI 是从用户端整体平移过来的（含文案/选项/交互），按钮形状跟随用户端
               才能保持"同一功能同一形态"的产品心智，属于该弹窗的定向例外，不影响
               管控端其它按钮的 4px 圆角规范。
          */}
          <div className="flex justify-end" style={{ gap: 8, marginTop: ms.gap }}>
            <Button
              variant="claw-outline"
              size="sm"
              className="!rounded-full px-4"
              style={{ fontSize: ms.btnSize, height: ms.btnHeight }}
              onClick={handleStartCancel}
            >
              取消
            </Button>
            {(() => {
              const needAuth = !hasAskedAuth(agentInstanceId);
              const confirmDisabled = needAuth && !diagOptions.authorize;
              return (
                <Button
                  variant="claw-primary"
                  size="sm"
                  className="!rounded-full px-4"
                  style={{ fontSize: ms.btnSize, height: ms.btnHeight }}
                  onClick={handleStartConfirm}
                  disabled={confirmDisabled}
                  title={confirmDisabled ? "请先勾选「同意使用龙虾医生功能」" : undefined}
                >
                  确认开始
                </Button>
              );
            })()}
          </div>
        </div>
      </div>
    );
  };

  const renderEndModal = () => {
    const ms = getModalStyles();
    const checkboxSize = ms.isLarge ? 16 : 14;
    const checkIconSize = ms.isLarge ? 10 : 8;
    return (
      <div className="absolute inset-0 z-50 flex items-center justify-center rounded-[var(--radius-lg)] overflow-hidden">
        {/* 遮罩层 */}
        <div className="absolute inset-0 bg-black/30 rounded-[var(--radius-lg)]" onClick={() => setShowEndModal(false)} />
        {/* 弹窗内容 */}
        <div
          ref={modalRef}
          className="relative z-10 bg-white rounded-[8px] shadow-[0_4px_12px_rgba(0,0,0,0.12)] animate-in fade-in zoom-in-95 duration-150"
          style={{
            width: "calc(100% - 24px)",
            maxWidth: ms.maxWidth,
            padding: ms.padding,
            // 兜底：与startModal 一致，弹窗自身在极窄容器时可滚动，保证按钮可见。
            maxHeight: "calc(100% - 16px)",
            overflowY: "auto",
          }}
        >
          <h3
            className="font-semibold text-gray-900 leading-tight"
            style={{ fontSize: ms.titleSize }}
          >
            结束诊断
          </h3>
          <p className="text-gray-500 leading-relaxed" style={{ fontSize: ms.bodySize, marginTop: ms.gap * 0.5 }}>
            即将结束本次诊断，临时龙虾医生节点将被销毁。
          </p>
          {snapshotCreated && (
            <div style={{ marginTop: ms.gap }}>
              <label className="flex items-start gap-2 cursor-pointer select-none rounded-[4px] px-2 py-2 border border-[var(--cp-border,var(--border))] bg-white transition-colors">
                <input
                  type="checkbox"
                  checked={rollbackChecked}
                  onChange={(e) => setRollbackChecked(e.target.checked)}
                  className="sr-only"
                />
                <span
                  aria-hidden
                  className="mt-0.5 flex-shrink-0 inline-flex items-center justify-center rounded-[3px] transition-all"
                  style={{
                    width: checkboxSize,
                    height: checkboxSize,
                    background: rollbackChecked ? "var(--cp-brand-blue, #1447E6)" : "#FFFFFF",
                    border: rollbackChecked ? "1px solid transparent" : "1.5px solid #D5D8E0",
                  }}
                >
                  {rollbackChecked && (
                    <svg width={checkIconSize} height={checkIconSize} viewBox="0 0 12 12" fill="none">
                      <path d="M2.5 6.2L4.8 8.5L9.5 3.5" stroke="#FFFFFF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  )}
                </span>
                <div className="flex-1 min-w-0">
                  {/*
                    网关重启影响（2026-08-04 实测）：回滚会触发双段冷替换，
                    中断约 28.9 秒 + 8.4 秒，较创建快照更严重。详见
                    .codebuddy/notes/gateway-restart-impact-2026-08-04.md。
                    为与创建快照项对称并降低认知负担，当前统一使用「短暂中断」；
                    若后续需要精确化，可升级为"可能多次中断，请稍候至操作完全完成"。
                  */}
                  <p className="text-gray-700 leading-snug" style={{ fontSize: ms.bodySize }}>回滚到诊断前快照</p>
                  <p className="text-gray-400 mt-0.5 leading-relaxed" style={{ fontSize: ms.hintSize }}>勾选后，将把 Agent 配置恢复到诊断开始前的状态。</p>
                  <span className="block mt-1 text-gray-400" style={{ fontSize: ms.hintSize }}>
                    回滚会触发网关重启，期间该 Agent 对外服务将短暂中断。
                  </span>
                </div>
              </label>
            </div>
          )}
          <div className="flex justify-end" style={{ gap: 6, marginTop: ms.gap }}>
            <Button
              variant="claw-outline"
              size="sm"
              className="px-2.5"
              style={{ fontSize: ms.btnSize, height: ms.btnHeight }}
              onClick={() => setShowEndModal(false)}
            >
              取消
            </Button>
            <Button
              variant="claw-primary"
              size="sm"
              className="px-2.5"
              style={{ fontSize: ms.btnSize, height: ms.btnHeight }}
              onClick={handleEndConfirm}
            >
              确认结束
            </Button>
          </div>
        </div>
      </div>
    );
  };

  //─── 渲染 guard：active=false 时不渲染任何面板内容 ─────────────────────────
  // 父组件（OpenClawMonitor "龙虾医生"区块）在 doctorActive=false 时会独立渲染
  // tip + 「开始诊断」按钮的初始静态视图；组件自身只负责 active=true 时的对话面板。
  // 没有这道 guard，结束诊断后 active 变 false 但内部 state（messages 等）仍会渲染，
  // 出现"结束后仍看到对话历史 + 弹出开始诊断确认"的错位现象。
  if (!active) return null;

  // ─── 弹窗显示时的容器最小高度 ─────────────────────────────────────────────
  // 字号统一压到≤12px（与外层"龙虾医生"标题字号对齐）后，弹窗真实内容高度：
  //   · 开始诊断（首次授权态，标题+描述+2项 DiagOptionRow+提示+按钮）≈ 280~320px
  //   · 开始诊断（已授权态，隐藏"同意"项）≈ 220~250px
  //   · 结束诊断（含回滚项）≈ 220~260px
  // 取 340 兜底：足以容纳最大态首次授权弹窗（含 8px 上下留白），同时不会因
  // 过大导致小内容弹窗周围出现明显空白。极端场景下弹窗自身还有 overflowY:auto。
  const modalOpenMinHeight = (showStartModal || showEndModal) ? 340 : undefined;

  return (
    <div ref={panelContainerRef} className="relative" style={{ minHeight: modalOpenMinHeight }}>
      {/* 结束诊断按钮 —— 与"开始诊断"同级同样式 */}
      {!expanded && instanceStatus === "active" && diagPhase !== "idle" && (
        <div className="mt-3">
          <Button
            variant="claw-primary"
            size="sm"
            className="scale-[0.7] origin-left"
            onClick={handleEndClick}
          >
            结束诊断
          </Button>
        </div>
      )}

      {/* ─── 内嵌面板（缩小模式） ────────────────────────────────────────────
          追加渲染 guard：仅当 instanceStatus 已进入 creating/active/... 等实质态时才渲染。
          之前的问题：用户点「开始诊断」后 `doctorActive` 变true，组件挂载并立即
          setShowStartModal(true)；此时 instanceStatus 仍是 "none"，chatContent
          会渲染"龙虾医生即将就位…"的灰色输入框占位——用户在弹窗背后看到这块
          "已经进入诊断"的错觉视觉；结束/取消后哪怕 active 已复位、若还有极短的
          状态过渡期也可能一闪。现在 none阶段完全不渲染面板本体，弹窗背后只保留
          父级的 tip +遮罩，视觉与图 2（用户期望的"最初静态"）一致。
      */}
      {!expanded && instanceStatus !== "none" && (
        <div className="mt-3">
          {chatContent(false)}
        </div>
      )}

      {/* ─── 全屏 Dialog（展开模式） ──────────────────────────────────────────── */}
      <Dialog open={expanded} onOpenChange={(o) => { if (!o) setExpanded(false); }}>
        <DialogContent className="sm:max-w-[800px] h-[680px] flex flex-col p-0 gap-0 rounded-[var(--radius-lg)] overflow-hidden [&>button]:hidden">
          {/* 用于 ResizeObserver 测量的全屏容器 ref */}
          <div ref={fullscreenContainerRef} className="relative flex flex-col h-full w-full">
            {/* 顶部：提示文字 + 操作按钮 + 缩小按钮 */}
            <div className="relative flex-shrink-0 px-5 pt-8 pb-3">
              {/* 缩小按钮 —— 右上角 */}
              <button
                onClick={() => setExpanded(false)}
                className="absolute top-3 right-3 z-10 w-7 h-7 rounded-[var(--radius-lg)] flex items-center justify-center hover:bg-gray-100 transition-colors"
                title="缩小"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-500">
                  <polyline points="4 14 10 14 10 20" /><polyline points="20 10 14 10 14 4" /><line x1="14" y1="10" x2="21" y2="3" /><line x1="3" y1="21" x2="10" y2="14" />
                </svg>
              </button>
              <div className="pr-8">
                <p className="text-xs text-gray-500 leading-relaxed">
                  AI 智能诊断，帮助您发现并修复 Agent 运行问题。若诊断开始前已勾选「创建配置快照」，结束诊断后可选择配置回滚。
                </p>
                {instanceStatus === "active" && diagPhase !== "idle" && (
                  <div className="mt-3">
                    <Button
                      variant="claw-primary"
                      size="sm"
                      className="scale-[0.7] origin-left"
                      onClick={handleEndClick}
                    >
                      结束诊断
                    </Button>
                  </div>
                )}
              </div>
            </div>
            {/* 对话主体 */}
            <div className="flex-1 min-h-0">
              {chatContent(true)}
            </div>

            {/* ─── 全屏模式下的弹窗（渲染在 Dialog 内部） ─────────────────────── */}
            {showStartModal && renderStartModal()}
            {showEndModal && renderEndModal()}
          </div>
        </DialogContent>
      </Dialog>

      {/* ─── 小窗模式下的弹窗（渲染在面板内部） ──────────────────────────────── */}
      {!expanded && showStartModal && renderStartModal()}
      {!expanded && showEndModal && renderEndModal()}
    </div>
  );
}
