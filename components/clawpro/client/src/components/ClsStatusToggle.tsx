/**
 * ClsStatusToggle - CLS 计费状态模拟浮层（仅 Demo/开发用）
 *
 * 悬浮在页面右下角，用 6 行组合表模拟 clawpro 控制台状态 × CLS 服务的笛卡尔积：
 *
 *   # | clawpro 控制台状态 | CLS 服务
 *   1 | 正常              | 正常
 * 2 | 正常 | 已停服
 *   3 | 到期前 15 天       | 正常
 *   4 | 到期前 15 天   | 已停服
 *   5 | 已停服      | 正常
 *   6 | 已停服        | 已停服
 *
 * 点击某一行 → 同时设置 ServiceStatusContext 的 phase + clsStatus，
 * 与「BillingStatusToggle」控制的更细的 7 档 phase 兼容（本组件只在 3 大类
 * 粒度上做快速切换：active / critical-d7 代表临期 / suspended-d1 代表停服）。
 *
 * 生产环境应移除此组件。
 */
import { useState, useEffect } from "react";
import { Database, ChevronUp, ChevronDown, Minimize2 } from "lucide-react";
import {
  useServiceStatus,
  type ServicePhase,
  type ClsStatus,
} from "../contexts/ServiceStatusContext";
import { useActiveDemoPanel, setActiveDemoPanel } from "./demoFloatingPanel";

/** clawpro 控制台状态的 3 大类粗粒度值（用于表格 6 行组合） */
type ClawproCoarse = "active" | "warning" | "suspended";

/** 从 7 档细分 phase 派生出 3 大类粗粒度值 */
function coarseFromPhase(phase: ServicePhase): ClawproCoarse {
  if (phase === "active") return "active";
  if (
    phase === "warning-d1" ||
    phase === "warning-d2-6" ||
    phase === "critical-d7"
  )
    return "warning";
  return "suspended";
}

/** 3 大类粗粒度值 → 该类的代表性 phase（用于表格点击时写回 ServiceStatusContext） */
const COARSE_TO_PHASE: Record<ClawproCoarse, ServicePhase> = {
  active: "active",
  // "到期前 15 天" 用现有临期档中最能代表 15 天场景的一档；如需更精细请去 BillingStatusToggle 里选
  warning: "warning-d2-6",
  suspended: "suspended-d1",
};

interface Combo {
  index: number;
  clawpro: ClawproCoarse;
  clawproLabel: string;
  cls: ClsStatus;
  clsLabel: string;
}

const COMBOS: Combo[] = [
  {
    index: 1,
    clawpro: "active",
    clawproLabel: "正常",
    cls: "active",
    clsLabel: "正常",
  },
  {
    index: 2,
    clawpro: "active",
    clawproLabel: "正常",
    cls: "suspended",
    clsLabel: "已停服",
  },
  {
    index: 3,
    clawpro: "warning",
    clawproLabel: "到期前 15 天",
    cls: "active",
    clsLabel: "正常",
  },
  {
    index: 4,
    clawpro: "warning",
    clawproLabel: "到期前 15 天",
    cls: "suspended",
    clsLabel: "已停服",
  },
  {
    index: 5,
    clawpro: "suspended",
    clawproLabel: "已停服",
    cls: "active",
    clsLabel: "正常",
  },
  {
    index: 6,
    clawpro: "suspended",
    clawproLabel: "已停服",
    cls: "suspended",
    clsLabel: "已停服",
  },
];

/** 状态圆点配色：与 BillingStatusToggle 保持一致的语义色 */
function dotClass(clawpro: ClawproCoarse | ClsStatus): string {
  if (clawpro === "active") return "bg-green-500";
  if (clawpro === "warning") return "bg-yellow-400";
  return "bg-red-500";
}

export default function ClsStatusToggle() {
  const { phase, clsStatus, setPhase, setClsStatus } = useServiceStatus();
  const [expanded, setExpanded] = useState(false);
  const activeDemoPanel = useActiveDemoPanel();
  const hideTrigger = activeDemoPanel !== null && activeDemoPanel !== "cls";

  // 与其它 Demo 浮层协调：任一浮层展开时，其余的折叠 header 隐藏
  useEffect(() => {
    if (expanded) setActiveDemoPanel("cls");
    else if (activeDemoPanel === "cls") setActiveDemoPanel(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded]);
  useEffect(() => {
    if (activeDemoPanel !== null && activeDemoPanel !== "cls" && expanded) {
      setExpanded(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeDemoPanel]);

  // 当前 phase 属于哪一类粗粒度值（用于高亮当前所处组合行）
  const currentCoarse = coarseFromPhase(phase);

  const handlePickCombo = (combo: Combo) => {
    setPhase(COARSE_TO_PHASE[combo.clawpro]);
    setClsStatus(combo.cls);
  };

  return (
    <div
      className="fixed right-4 z-[9999]"
      style={{
        // 四个 Demo 浮层在右下角一列纵向叠放：
        //   BillingStatusToggle    → bottom: 16px（默认 bottom-4）
        //   AdminModeFloatingToggle → bottom: 60px
        //   OnboardingDemoPanel    → bottom: 104px
        // ClsStatusToggle        → bottom: 148px（本组件，最上层）
        bottom: 148,
      }}
      data-billing-exempt
    >
      <div className="relative">
        {/* 展开面板：固定浮在视口右下角，层级最高 */}
        {expanded && (
          <div
            className="fixed right-4 bottom-4 z-[100000] w-[340px] bg-white rounded-xl border border-gray-200 overflow-hidden animate-in slide-in-from-bottom-2 duration-200"
            style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
            data-billing-exempt
          >
            {/* 面板 header */}
            <div
              className="px-4 py-3 bg-gray-900 select-none"
              data-billing-exempt
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-white">
                  CLS 计费状态模拟
                </span>
                <button
                  data-billing-exempt
                  onClick={e => {
                    e.stopPropagation();
                    setExpanded(false);
                  }}
                  className="w-6 h-6 rounded flex items-center justify-center hover:bg-white/20 transition-colors"
                  title="折叠面板"
                >
                  <Minimize2 className="w-3.5 h-3.5 text-white/70" />
                </button>
              </div>
              <p className="text-[11px] text-white/60 mt-1">
                clawpro 控制台 × CLS 服务的 6 种组合
              </p>
            </div>

            {/* 组合表：6 行，点击行切换 */}
            <div className="px-3 py-3" data-billing-exempt>
              <div className="rounded-md border border-gray-100 overflow-hidden">
                {/* 表头 */}
                <div className="grid grid-cols-[32px_1fr_1fr] items-center bg-gray-50 border-b border-gray-100">
                  <div className="text-[10px] font-semibold text-gray-500 text-center py-1.5">
                    #
                  </div>
                  <div className="text-[10px] font-semibold text-gray-500 py-1.5 pl-2">
                    clawpro 控制台状态
                  </div>
                  <div className="text-[10px] font-semibold text-gray-500 py-1.5 pl-2">
                    CLS 服务
                  </div>
                </div>

                {/* 6 行组合 */}
                {COMBOS.map(combo => {
                  const isActive =
                    currentCoarse === combo.clawpro && clsStatus === combo.cls;
                  return (
                    <button
                      key={combo.index}
                      data-billing-exempt
                      onClick={() => handlePickCombo(combo)}
                      className={`grid grid-cols-[32px_1fr_1fr] items-center w-full text-left border-b border-gray-100 last:border-b-0 transition-colors ${
                        isActive
                          ? "bg-gray-900 text-white"
                          : "bg-white text-gray-700 hover:bg-gray-50"
                      }`}
                      title={`切换到组合 ${combo.index}：clawpro=${combo.clawproLabel}，CLS=${combo.clsLabel}`}
                    >
                      <span
                        className={`text-[11px] font-mono text-center py-2 ${
                          isActive ? "text-white/80" : "text-gray-400"
                        }`}
                      >
                        {combo.index}
                      </span>
                      <span className="flex items-center gap-1.5 text-[11px] py-2 pl-2">
                        <span
                          className={`w-1.5 h-1.5 rounded-full shrink-0 ${dotClass(combo.clawpro)}`}
                        />
                        <span>{combo.clawproLabel}</span>
                      </span>
                      <span className="flex items-center gap-1.5 text-[11px] py-2 pl-2">
                        <span
                          className={`w-1.5 h-1.5 rounded-full shrink-0 ${dotClass(combo.cls)}`}
                        />
                        <span>{combo.clsLabel}</span>
                      </span>
                    </button>
                  );
                })}
              </div>

              {/* 附注：说明本面板的作用边界与驱动逻辑 */}
              <p className="text-[10px] text-gray-400 leading-4 mt-2">
                产研须知：本面板仅用于模拟 CLS 日志业务计费策略。面板依据 CLS
                的服务状态（2 种）与 ClawPro 控制台的计费服务状态（归纳为 3
                类），驱动 CLS 业务页面的相应变化；不涉及其他页面。
              </p>
            </div>
          </div>
        )}

        {/* 触发按钮 */}
        {!hideTrigger && (
          <button
            onClick={() => setExpanded(v => !v)}
            className="w-[180px] flex items-center gap-2 px-3 py-2 bg-white rounded-xl border border-gray-200 hover:bg-gray-50 transition-colors select-none"
            style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
            data-billing-exempt
            title="CLS 计费状态模拟（点击展开/折叠）"
          >
            <Database className="w-4 h-4 text-gray-500 shrink-0" />
            <span className="text-xs font-medium text-gray-700">
              CLS 计费状态模拟
            </span>
            {expanded ? (
              <ChevronDown className="w-3 h-3 text-gray-400 ml-auto shrink-0" />
            ) : (
              <ChevronUp className="w-3 h-3 text-gray-400 ml-auto shrink-0" />
            )}
          </button>
        )}
      </div>
    </div>
  );
}
