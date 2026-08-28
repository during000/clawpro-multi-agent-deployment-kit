/**
 * PluginUpgradeContext - CLS 采集插件升级全局状态
 *
 * 设计目标：
 * - 升级是个长任务（可能 1-10 分钟），用户期间能切页面、切菜单、刷新
 * - 升级状态与浮动 UI 解耦，挂在全局根部，所有页面都能看到
 * - 支持 localStorage 持久化：用户刷新页面后，仍能恢复进度显示
 *
 * 后端对接说明：
 * - 当前为 mock：用 setInterval 模拟 progress 从 0 涨到 total
 * - 真实接入时，把 mock 部分替换为：轮询 / WebSocket / SSE 订阅后端任务进度
 */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

// ─── 类型 ─────────────────────────────────────────────────────────────
export type UpgradeStatus = "idle" | "running" | "succeeded" | "failed";

export interface UpgradeState {
  status: UpgradeStatus;
  /** 后端总共需要升级的机器数 */
  total: number;
  /** 已经升级完成的机器数 */
  progress: number;
  /** 任务 ID，用于后端对接（mock 阶段也保留以便持久化恢复） */
  taskId: string | null;
}

interface PluginUpgradeContextValue extends UpgradeState {
  /** 启动升级流程（mock 阶段会自动模拟进度推进） */
  start: () => void;
  /** 主动关闭升级浮窗（仅在 succeeded/failed 状态可调用） */
  dismiss: () => void;
}

// ─── 常量 ─────────────────────────────────────────────────────────────
export const LATEST_PLUGIN_VERSION = "v2";
const STORAGE_KEY = "clsPluginUpgradeState";

// Mock 配置：模拟 24 台机器，每台耗时 1.5s（共 36 秒）
const MOCK_TOTAL = 24;
const MOCK_TICK_MS = 1500;

// ─── Context ──────────────────────────────────────────────────────────
const PluginUpgradeContext = createContext<PluginUpgradeContextValue | null>(null);

const INITIAL_STATE: UpgradeState = {
  status: "idle",
  total: 0,
  progress: 0,
  taskId: null,
};

// ─── 持久化辅助 ───────────────────────────────────────────────────────
function loadPersistedState(): UpgradeState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return INITIAL_STATE;
    const parsed = JSON.parse(raw) as Partial<UpgradeState>;
    // 仅恢复 running 状态：succeeded/failed/idle 都不需要恢复浮窗
    if (parsed.status !== "running") return INITIAL_STATE;
    return {
      status: "running",
      total: parsed.total ?? 0,
      progress: parsed.progress ?? 0,
      taskId: parsed.taskId ?? null,
    };
  } catch {
    return INITIAL_STATE;
  }
}

function persistState(state: UpgradeState) {
  try {
    if (state.status === "idle") {
      localStorage.removeItem(STORAGE_KEY);
    } else {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    }
  } catch {
    /* ignore quota errors */
  }
}

// ─── Provider ─────────────────────────────────────────────────────────
export function PluginUpgradeProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<UpgradeState>(() => loadPersistedState());

  // mock 用的 interval 引用（真实接入时替换为 WS/poll handle）
  const tickerRef = useRef<number | null>(null);

  // 任意状态变化都同步到 localStorage
  useEffect(() => {
    persistState(state);
  }, [state]);

  // ─── Mock 进度推进 ─────────────────────────────────────────────────
  // 真实接入时：删除此 effect，改为在 start() 里订阅后端事件
  useEffect(() => {
    if (state.status !== "running") {
      if (tickerRef.current !== null) {
        window.clearInterval(tickerRef.current);
        tickerRef.current = null;
      }
      return;
    }

    // 已在跑就不重复起 ticker（页面刷新恢复时会进入这里）
    if (tickerRef.current !== null) return;

    tickerRef.current = window.setInterval(() => {
      setState((prev) => {
        if (prev.status !== "running") return prev;
        const next = prev.progress + 1;
        // 达到目标 → 升级成功
        if (next >= prev.total) {
          localStorage.setItem("clsPluginVersion", LATEST_PLUGIN_VERSION);
          localStorage.setItem("sessionPageVersion", "v2");
          return { ...prev, progress: prev.total, status: "succeeded" };
        }
        return { ...prev, progress: next };
      });
    }, MOCK_TICK_MS);

    return () => {
      if (tickerRef.current !== null) {
        window.clearInterval(tickerRef.current);
        tickerRef.current = null;
      }
    };
  }, [state.status]);

  // ─── 跨标签页同步 ──────────────────────────────────────────────────
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key !== STORAGE_KEY) return;
      setState(loadPersistedState());
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  // ─── 操作 ──────────────────────────────────────────────────────────
  const start = useCallback(() => {
    setState((prev) => {
      if (prev.status === "running") return prev; // 防重入
      return {
        status: "running",
        total: MOCK_TOTAL,
        progress: 0,
        taskId: `mock-${Date.now()}`,
      };
    });
  }, []);

  const dismiss = useCallback(() => {
    setState((prev) => {
      if (prev.status === "running") return prev; // 升级中不允许关闭
      return INITIAL_STATE;
    });
  }, []);

  return (
    <PluginUpgradeContext.Provider
      value={{ ...state, start, dismiss }}
    >
      {children}
    </PluginUpgradeContext.Provider>
  );
}

// ─── hook ─────────────────────────────────────────────────────────────
export function usePluginUpgrade() {
  const ctx = useContext(PluginUpgradeContext);
  if (!ctx) {
    throw new Error("usePluginUpgrade must be used within PluginUpgradeProvider");
  }
  return ctx;
}

// ─── 工具函数（供按钮判断红点） ──────────────────────────────────────
export function needsPluginUpgradeHint(
  clsEnabled: boolean,
  currentVersion: string,
): boolean {
  return clsEnabled && currentVersion !== LATEST_PLUGIN_VERSION;
}
