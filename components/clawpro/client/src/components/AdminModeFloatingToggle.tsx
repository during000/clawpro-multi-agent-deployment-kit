/**
 * AdminModeFloatingToggle - 管控端「成员管理模式」独立悬浮切换面板（仅开发/Demo 用）
 *
 * 需求：把「成员管理模式」从用户头像下拉里抽离，做成始终悬浮的独立面板，
 * 方便在任何管控端页面里随时切换（普通 / oneid专用 / 统一）。
 *
 * 挂载位置：AdminLayout 内部（须能访问 AdminModeContext）；tenant / landing 不显示。
 *
 * 位置：右下角，BillingStatusToggle（bottom-4 right-4）上方，纵向叠放不冲突。
 * 交互：可折叠（Header + 展开面板），首次展开态。
 *
 * 视觉：仿 BillingStatusToggle 的白底 + 圆角 + 阴影 + 折叠 header 结构，
 *      内容区直接复用 AdminModeToggle 展开态的分段选择器视觉。
 */
import { useState, useEffect } from "react";
import { Users, ChevronUp, ChevronDown, Minimize2 } from "lucide-react";
import { useAdminMode, type AdminMode } from "@/contexts/AdminModeContext";
import {
  useActiveDemoPanel,
  setActiveDemoPanel,
} from "./demoFloatingPanel";

const BRAND_GRADIENT = "linear-gradient(135deg, #007AFF, #5856D6)";

const LABEL: Record<AdminMode, string> = {
  custom: "普通",
  standard: "oneid专用",
  unified: "统一",
};

export default function AdminModeFloatingToggle() {
  const { mode, setMode, isCustom, isStandard, isUnified } = useAdminMode();
  const [expanded, setExpanded] = useState(false);
  const activeDemoPanel = useActiveDemoPanel();
  const hideTrigger = activeDemoPanel !== null && activeDemoPanel !== "member";

  // 与另外两个 Demo 浮层协调：任一浮层展开时，其余两个的折叠 header 隐藏；本组件展开时广播 "member"。
  useEffect(() => {
    if (expanded) setActiveDemoPanel("member");
    else if (activeDemoPanel === "member") setActiveDemoPanel(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded]);
  useEffect(() => {
    if (activeDemoPanel !== null && activeDemoPanel !== "member" && expanded) {
      setExpanded(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeDemoPanel]);

  return (
    <div
      className="fixed right-4 z-[9999]"
      style={{
        // 三个 Demo 浮层在右下角一列纵向叠放：
        //   BillingStatusToggle    → bottom: 16px（默认 bottom-4）
        //   AdminModeFloatingToggle → bottom: 60px（本组件）
        //   OnboardingDemoPanel    → bottom: 104px
        bottom: 60,
      }}
      data-billing-exempt
    >
      {/* 相对定位容器：仅承载折叠触发按钮；展开面板已改为 fixed 到视口右下角，脱离本容器。 */}
      <div className="relative">
        {/* 展开面板：固定浮在视口右下角，层级最高，打开时会覆盖三个折叠 header 那一列（预期） */}
        {expanded && (
          <div
            className="fixed right-4 bottom-4 z-[100000] w-[280px] bg-white rounded-xl border border-gray-200 overflow-hidden animate-in slide-in-from-bottom-2 duration-200"
            style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
            data-billing-exempt
          >
            {/* 面板 header：与"用户引导模拟"浮层对齐，右上角提供关闭按钮 */}
            <div className="px-4 py-3 bg-gray-900 select-none" data-billing-exempt>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-white">成员管理模式</span>
                <button
                  data-billing-exempt
                  onClick={(e) => {
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
                当前：{LABEL[mode]}
              </p>
            </div>
            <div className="px-3 py-3" data-billing-exempt>
              <div className="flex items-center bg-gray-100 rounded-[4px] p-0.5 gap-0.5">
            {/* 普通模式（custom） */}
            <button
              onClick={() => setMode("custom")}
              data-billing-exempt
              className={`
                flex-1 flex items-center justify-center gap-1 h-7 rounded-[4px] text-xs font-medium
                transition-all duration-200 select-none
                ${
                  isCustom
                    ? "bg-white text-violet-600 shadow-sm shadow-black/10 font-semibold"
                    : "text-gray-500 hover:text-gray-700"
                }
              `}
            >
              <span
                className={`w-1.5 h-1.5 rounded-full flex-shrink-0 transition-colors duration-200 ${
                  isCustom ? "bg-violet-500" : "bg-gray-400"
                }`}
              />
              普通
            </button>

            {/* OneID 专用模式（standard） */}
            <button
              onClick={() => setMode("standard")}
              data-billing-exempt
              className={`
                flex-1 flex items-center justify-center gap-1 h-7 rounded-[4px] text-xs font-medium
                transition-all duration-200 select-none
                ${
                  isStandard
                    ? "bg-white text-blue-600 shadow-sm shadow-black/10 font-semibold"
                    : "text-gray-500 hover:text-gray-700"
                }
              `}
            >
              <span
                className={`w-1.5 h-1.5 rounded-full flex-shrink-0 transition-colors duration-200 ${
                  isStandard ? "bg-blue-500" : "bg-gray-400"
                }`}
              />
              oneid专用
            </button>

            {/* 统一模式（unified） */}
            <button
              onClick={() => setMode("unified")}
              data-billing-exempt
              className={`
                flex-1 flex items-center justify-center gap-1 h-7 rounded-[4px] text-xs font-medium
                transition-all duration-200 select-none
                ${
                  isUnified
                    ? "bg-white shadow-sm shadow-black/10 font-semibold"
                    : "text-gray-500 hover:text-gray-700"
                }
              `}
              style={isUnified ? { color: "#5856D6" } : undefined}
            >
              <span
                className="w-1.5 h-1.5 rounded-full flex-shrink-0 transition-all duration-200"
                style={
                  isUnified
                    ? { background: BRAND_GRADIENT }
                    : { background: "#9ca3af" }
                }
              />
              统一
            </button>
              </div>
            </div>
          </div>
        )}

        {/* 触发按钮（始终 180px 折叠 header 大小；点击切换展开/折叠）
            当其它 Demo 浮层展开时，本触发按钮隐藏，避免视觉打架。 */}
        {!hideTrigger && (
          <button
            onClick={() => setExpanded((v) => !v)}
            className="w-[180px] flex items-center gap-2 px-3 py-2 bg-white rounded-xl border border-gray-200 hover:bg-gray-50 transition-colors select-none"
            style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
            data-billing-exempt
          >
            <Users className="w-4 h-4 text-gray-500 shrink-0" />
            <span className="text-xs font-medium text-gray-700">成员管理模式</span>
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
