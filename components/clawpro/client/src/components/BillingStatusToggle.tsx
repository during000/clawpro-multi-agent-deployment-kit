/**
 * BillingStatusToggle - 计费状态模拟开关（仅开发/Demo 用）
 *
 * 悬浮在页面右下角，允许切换：
 *   - 管控台状态：7 档，按语义分 3 大类展示（正常 / 预警 / 停服后）
 *       • 正常：active
 *       • 预警：warning-d1(预警 D1) / warning-d2-6(预警 D2-6) / critical-d7(预警 D7)
 *       • 停服后：suspended-d1(D1、D8) / suspended-d2-12(D2-D7、D9-D12) / recycling-d13-15(D13-D15)
 *   - 席位欠费：3 档（正常 / 欠费（D1、D7）/ 欠费）
 *     • 欠费（D1、D7）：欠费缓冲期（D1 触发 ~ D7 到期），催充值弹窗 / 浮层 / 右下角通知卡 都在此段展示
 *     • 欠费：缓冲期结束进入正式欠费，仅保留 tooltip / 禁用态，弹窗不再骚扰
 *
 * 生产环境应移除此组件。
 */
import { useState, useEffect } from "react";
import { Settings2, ChevronUp, ChevronDown, Minimize2 } from "lucide-react";
import {
  useServiceStatus,
  type ServicePhase,
  type SeatArrearsPhase,
} from "../contexts/ServiceStatusContext";
import {
  useActiveDemoPanel,
  setActiveDemoPanel,
} from "./demoFloatingPanel";

/** 管控台状态 7 档定义 */
interface PhaseOption {
  key: ServicePhase;
  label: string;
  /** 状态圆点颜色 */
  dot: string;
}

/** 席位欠费 3 档定义 */
interface SeatArrearsOption {
  key: SeatArrearsPhase;
  label: string;
  dot: string;
}

/** 管控台状态分组：按语义分为「正常 / 预警 / 停服后」三大类，组间以分割线换行呈现 */
interface PhaseGroup {
  /** 分组标题（右侧小字标签） */
  title: string;
  /** 分组下方的一行说明，帮助研发/产品理解该阶段业务口径 */
  description: string;
  options: PhaseOption[];
}

const SEAT_ARREARS_OPTIONS: SeatArrearsOption[] = [
  { key: "active", label: "正常", dot: "bg-green-500" },
  { key: "arrears-buffer", label: "欠费（D1、D7）", dot: "bg-orange-500" },
  { key: "arrears", label: "欠费", dot: "bg-red-500" },
];

const PHASE_GROUPS: PhaseGroup[] = [
  {
    title: "正常",
    description: "服务在有效期内，全部功能可用，无任何提示。",
    options: [
      { key: "active", label: "正常", dot: "bg-green-500" },
    ],
  },
  {
    title: "预警",
    description: "到期前 7 天起（D1 / D2–6 / D7 逐步加强），功能仍全部可用。",
    options: [
      { key: "warning-d1", label: "预警 D1", dot: "bg-yellow-400" },
      { key: "warning-d2-6", label: "预警 D2-6", dot: "bg-yellow-400" },
      { key: "critical-d7", label: "预警 D7", dot: "bg-orange-500" },
    ],
  },
  {
    title: "停服后",
    description: "服务已停服，进入 15 天数据保留倒计时。",
    options: [
      { key: "suspended-d1", label: "停服后 D1、D8", dot: "bg-red-500" },
      { key: "suspended-d2-12", label: "停服后 D2-D7、D9-D12", dot: "bg-red-500" },
      { key: "recycling-d13-15", label: "停服后 D13-D15", dot: "bg-red-700" },
    ],
  },
];

/** 席位欠费一行的说明文案：与上述三大类互相独立，可叠加 */
const SEAT_ARREARS_DESCRIPTION =
  "服务未到期，但席位不足 / 欠费。可与上述任一阶段叠加（如临期 + 欠费）。";

export default function BillingStatusToggle() {
  const { phase, seatArrearsPhase, setPhase, setSeatArrearsPhase } = useServiceStatus();
  const [expanded, setExpanded] = useState(false);
  // 与另外两个 demo 浮层协调：任一浮层展开时，其余两个的折叠 header 隐藏，
  // 保证屏幕右下角只剩下当前展开面板，避免不同宽度的展开面板与其它 header 视觉打架。
  const activeDemoPanel = useActiveDemoPanel();
  const hideTrigger = activeDemoPanel !== null && activeDemoPanel !== "billing";

  // 展开状态与全局 active id 双向同步：本地关闭 → 广播 null；本地打开 → 广播 "billing"；
  // 被其它面板抢占（activeDemoPanel 变为其它 id）→ 本地强制折叠。
  useEffect(() => {
    if (expanded) setActiveDemoPanel("billing");
    else if (activeDemoPanel === "billing") setActiveDemoPanel(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded]);
  useEffect(() => {
    if (activeDemoPanel !== null && activeDemoPanel !== "billing" && expanded) {
      setExpanded(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeDemoPanel]);

  return (
    <div className="fixed bottom-4 right-4 z-[9999]" data-billing-exempt>
      {/* 相对定位容器：仅承载折叠触发按钮；展开面板已改为 fixed 到视口右下角，脱离本容器。 */}
      <div className="relative">
        {/* 展开面板：固定浮在视口右下角，与所有触发按钮列同一位置、层级最高，
            打开时会盖住三个折叠 header 那一列（预期）。 */}
        {expanded && (
          <div
            className="fixed right-4 bottom-4 z-[100000] w-[390px] bg-white rounded-xl border border-gray-200 overflow-hidden animate-in slide-in-from-bottom-2 duration-200"
            style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
            data-billing-exempt
          >
            {/* 面板 header：与"用户引导模拟"浮层对齐，右上角提供关闭按钮 */}
            <div className="px-4 py-3 bg-gray-900 select-none" data-billing-exempt>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-white">计费状态模拟</span>
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
                仅 Demo 模拟用，生产环境需移除
              </p>
            </div>
            <div className="px-3 py-3 space-y-3" data-billing-exempt>
          {/* 叠加与展示规则说明 —— 帮助阅读者理解多个通知组件叠加时的展示顺序与频次口径 */}
          <div>
            <label className="text-xs font-semibold text-gray-900 mb-1.5 block">叠加与展示规则</label>
            <div className="rounded-md border border-gray-100 bg-gray-50 p-2 space-y-1.5">
              {/* 每条规则用 flex 保证「序号」与「正文」左对齐；
                  正文用 flex-1 + min-w-0 + break-words 让文字随面板宽度自然换行截断，
                  避免面板被撑宽或文字溢出。 */}
              <div className="flex gap-1.5 text-[10px] leading-4 text-gray-600">
                <span className="text-gray-400 shrink-0">1.</span>
                <span className="flex-1 min-w-0 break-words">
                  <span className="text-gray-700 font-medium">场景叠加</span>：
                  同时命中多个场景时，优先展示「<span className="text-gray-700 font-medium">服务到期</span>」，
                  关闭后再展示「<span className="text-gray-700 font-medium">席位欠费</span>」。
                </span>
              </div>
              <div className="flex gap-1.5 text-[10px] leading-4 text-gray-600">
                <span className="text-gray-400 shrink-0">2.</span>
                <span className="flex-1 min-w-0 break-words">
                  <span className="text-gray-700 font-medium">组件优先级</span>：
                  <span className="text-gray-700 font-medium">弹窗</span>
                  &gt;
                  <span className="text-gray-700 font-medium">非阻断浮层</span>；
                  弹窗关闭后再展示浮层。
                </span>
              </div>
              <div className="flex gap-1.5 text-[10px] leading-4 text-gray-600">
                <span className="text-gray-400 shrink-0">3.</span>
                <span className="flex-1 min-w-0 break-words">
                  <span className="text-gray-700 font-medium">打扰频次</span>：
                  <span className="text-gray-700 font-medium">弹窗类组件每日仅弹一次</span>，
                  当日关闭后不再重复。
                </span>
              </div>
            </div>
          </div>

          {/* 管控台状态 —— 7 档，按「正常 / 预警 / 停服后」3 大类分组换行，组间加分割线 */}
          <div>
            <label className="text-xs font-semibold text-gray-900 mb-1.5 block">整体服务到期</label>
            <div className="divide-y divide-gray-100 rounded-md border border-gray-100">
              {PHASE_GROUPS.map((group) => {
                // 「停服后」有 3 个较长按钮（D1、D8 / D2-D7、D9-D12 / D13-D15），
                // 在 390px 面板宽下用默认 text-[11px]+px-2 会换行；只对该分组按钮做紧凑样式，
                // 保证三个按钮同一行显示。
                const compact = group.title === "停服后";
                const btnPadding = compact ? "px-1.5 py-1" : "px-2 py-1.5";
                const btnText = compact ? "text-[10px]" : "text-[11px]";
                const btnGap = compact ? "gap-0.5" : "gap-1";
                return (
                  <div key={group.title} className="p-1.5">
                    <div className="text-[10px] font-semibold text-gray-900 leading-4 px-1 pb-1">
                      {group.title}
                    </div>
                    {/* 分组说明：紧跟标题下方，帮助阅读者先看到业务口径，再选具体档位 */}
                    <p className="text-[10px] text-gray-400 leading-4 px-1 pb-1.5">
                      {group.description}
                    </p>
                    <div className={`flex flex-wrap ${btnGap}`}>
                      {group.options.map((opt) => {
                        const active = phase === opt.key;
                        return (
                          <button
                            key={opt.key}
                            data-billing-exempt
                            onClick={() => setPhase(opt.key)}
                            className={`flex items-center gap-1 ${btnPadding} rounded-md ${btnText} font-medium transition-all whitespace-nowrap ${
                              active
                                ? "bg-gray-900 text-white"
                                : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                            }`}
                            title={opt.label}
                          >
                            <span className={`w-1.5 h-1.5 rounded-full ${opt.dot}`} />
                            <span>{opt.label}</span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* 席位欠费 —— 3 档：正常 / 欠费（D1、D7）/ 欠费；去掉 truncate 完整显示 */}
          <div>
            <label className="text-xs font-semibold text-gray-900 mb-1.5 block">席位欠费</label>
            <div className="rounded-md border border-gray-100 p-1.5">
              {/* 席位欠费说明：与上方三大类可叠加。放在按钮组上方，与「整体服务到期」分组保持一致 */}
              <p className="text-[10px] text-gray-400 leading-4 px-1 pb-1.5">
                {SEAT_ARREARS_DESCRIPTION}
              </p>
              <div className="flex flex-wrap gap-1">
                {SEAT_ARREARS_OPTIONS.map((opt) => {
                  const active = seatArrearsPhase === opt.key;
                  return (
                    <button
                      key={opt.key}
                      data-billing-exempt
                      onClick={() => setSeatArrearsPhase(opt.key)}
                      className={`flex items-center gap-1 px-2 py-1.5 rounded-md text-[11px] font-medium transition-all whitespace-nowrap ${
                        active
                          ? "bg-gray-900 text-white"
                          : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                      }`}
                      title={opt.label}
                    >
                      <span className={`w-1.5 h-1.5 rounded-full ${opt.dot}`} />
                      <span>{opt.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>

            </div>
          </div>
        )}

        {/* 触发按钮（始终 180px 折叠 header 大小；点击切换展开/折叠）
            当其它 Demo 浮层展开时，本触发按钮隐藏，避免不同宽度的展开面板旁边露出多余按钮。 */}
        {!hideTrigger && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="w-[180px] flex items-center gap-2 px-3 py-2 bg-white rounded-xl border border-gray-200 hover:bg-gray-50 transition-colors select-none"
            style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
            data-billing-exempt
          >
            <Settings2 className="w-4 h-4 text-gray-500 shrink-0" />
            <span className="text-xs font-medium text-gray-700">计费状态模拟</span>
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
