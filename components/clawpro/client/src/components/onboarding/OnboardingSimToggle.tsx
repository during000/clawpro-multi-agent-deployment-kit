/**
 * OnboardingSimToggle - 右下角模拟开关挂件
 * 
 * 功能：
 * - 开启：模拟新手体验，重置所有引导进度并从头播放
 * - 关闭：模拟非新手状态，隐藏所有引导组件
 * - 带开发者面板：显示引导状态、已完成流程、localStorage 数据
 */
import { useState } from "react";
import { Play, Pause, RotateCcw, Settings, Eye, EyeOff, ChevronUp, ChevronDown } from "lucide-react";
import type { GuideState, GuideActions, GuideFlow } from "./types";

interface OnboardingSimToggleProps {
  state: GuideState;
  actions: GuideActions;
  flows?: GuideFlow[];
  /** 端类型（影响皮肤） */
  endpoint?: "admin" | "tenant";
}

export function OnboardingSimToggle({
  state,
  actions,
  flows = [],
  endpoint = "admin",
}: OnboardingSimToggleProps) {
  const [expanded, setExpanded] = useState(false);

  const themeClasses = endpoint === "tenant"
    ? "bg-gradient-to-r from-[#007AFF] to-[#5856D6]"
    : "bg-gray-900";

  return (
    <div className="fixed bottom-6 right-6 z-[99999]">
      {/* 展开面板 */}
      {expanded && (
        <div className="mb-3 w-[320px] bg-white rounded-xl border border-gray-200 shadow-2xl overflow-hidden animate-in slide-in-from-bottom-2 duration-200">
          {/* 面板头 */}
          <div className={`px-4 py-3 ${themeClasses}`}>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-white">新手引导模拟器</span>
              <span className="text-[10px] text-white/70 px-1.5 py-0.5 rounded bg-white/20">
                {endpoint === "tenant" ? "用户端" : "管理端"}
              </span>
            </div>
          </div>

          {/* 状态信息 */}
          <div className="p-4 space-y-3">
            {/* 模拟状态 */}
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-600">模拟状态</span>
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${state.isSimulating ? "bg-green-500 animate-pulse" : "bg-gray-300"}`} />
                <span className="text-xs font-medium text-gray-800">
                  {state.isSimulating ? "模拟新手中" : "关闭"}
                </span>
              </div>
            </div>

            {/* 当前流程 */}
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-600">当前流程</span>
              <span className="text-xs font-mono text-gray-800">
                {state.activeFlow || "无"}
              </span>
            </div>

            {/* 进度 */}
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-600">已完成</span>
              <span className="text-xs text-gray-800">
                {state.completedFlows.length}/{flows.length} 个流程
              </span>
            </div>

            {/* 步骤 */}
            {state.activeFlow && (
              <div className="flex items-center justify-between">
                <span className="text-xs text-gray-600">当前步骤</span>
                <span className="text-xs text-gray-800">
                  第 {state.activeStepIndex + 1} 步
                </span>
              </div>
            )}
          </div>

          {/* 操作按钮区 */}
          <div className="px-4 pb-4 flex flex-wrap gap-2">
            {!state.isSimulating ? (
              <button
                onClick={actions.startSimulation}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors"
              >
                <Play className="w-3 h-3" />
                开始模拟
              </button>
            ) : (
              <button
                onClick={actions.stopSimulation}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-red-500 hover:bg-red-600 rounded-md transition-colors"
              >
                <Pause className="w-3 h-3" />
                停止模拟
              </button>
            )}

            <button
              onClick={actions.resetAll}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
            >
              <RotateCcw className="w-3 h-3" />
              重置进度
            </button>

            {state.activeFlow && (
              <>
                <button
                  onClick={actions.nextStep}
                  className="px-3 py-1.5 text-xs font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
                >
                  下一步
                </button>
                <button
                  onClick={actions.skipFlow}
                  className="px-3 py-1.5 text-xs font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
                >
                  跳过流程
                </button>
              </>
            )}
          </div>

          {/* 流程列表 */}
          {flows.length > 0 && (
            <div className="border-t border-gray-100 px-4 py-3">
              <p className="text-[10px] text-gray-400 uppercase tracking-wider mb-2">可用流程</p>
              <div className="space-y-1 max-h-[120px] overflow-y-auto">
                {flows.map((flow) => {
                  const isCompleted = state.completedFlows.includes(flow.id);
                  const isActive = state.activeFlow === flow.id;
                  return (
                    <button
                      key={flow.id}
                      onClick={() => actions.startFlow(flow.id)}
                      disabled={isActive}
                      className={`w-full text-left px-2 py-1.5 rounded text-xs transition-colors ${
                        isActive
                          ? "bg-blue-50 text-blue-700 font-medium"
                          : isCompleted
                          ? "bg-green-50 text-green-700 line-through"
                          : "hover:bg-gray-50 text-gray-700"
                      }`}
                    >
                      <span className="flex items-center gap-2">
                        <span className={`w-1.5 h-1.5 rounded-full ${
                          isActive ? "bg-blue-500" : isCompleted ? "bg-green-500" : "bg-gray-300"
                        }`} />
                        {flow.name}
                        <span className="text-[10px] text-gray-400 ml-auto">{flow.endpoint}</span>
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}

      {/* 主按钮 */}
      <button
        onClick={() => setExpanded(!expanded)}
        className={`w-12 h-12 rounded-full shadow-lg flex items-center justify-center transition-all hover:scale-105 active:scale-95 ${themeClasses}`}
        title="新手引导模拟器"
      >
        {state.isSimulating ? (
          <Eye className="w-5 h-5 text-white" />
        ) : (
          <EyeOff className="w-5 h-5 text-white" />
        )}
        {/* 运行指示灯 */}
        {state.isSimulating && (
          <div className="absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full bg-green-400 border-2 border-white animate-pulse" />
        )}
      </button>
    </div>
  );
}
