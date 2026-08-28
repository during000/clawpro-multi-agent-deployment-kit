/**
 * AdminModeToggle - 管控端成员管理模式切换
 * 分段选择器（Segmented Control）风格，三段：
 *   普通（custom）- 紫色
 *   oneid专用（standard）- 蓝色
 *   统一（unified）- 品牌渐变蓝紫
 * 选中项白色卡片+阴影浮起，未选中项透明背景灰色文字
 * 默认激活普通模式（custom）
 */
import { useAdminMode, type AdminMode } from "@/contexts/AdminModeContext";

const BRAND_GRADIENT = "linear-gradient(135deg, #007AFF, #5856D6)";

const ORDER: AdminMode[] = ["custom", "standard", "unified"];

const DOT_COLOR: Record<AdminMode, string> = {
  custom: "bg-violet-500",
  standard: "bg-blue-500",
  unified: "", // 渐变靠 inline style
};

const TEXT_COLOR: Record<AdminMode, string> = {
  custom: "text-violet-600",
  standard: "text-blue-600",
  unified: "text-transparent bg-clip-text",
};

const LABEL: Record<AdminMode, string> = {
  custom: "普通",
  standard: "oneid专用",
  unified: "统一",
};

export default function AdminModeToggle({ collapsed }: { collapsed: boolean }) {
  const { mode, setMode, isCustom, isStandard, isUnified } = useAdminMode();

  /** 折叠态：点击循环切换到下一个模式 */
  const cycleMode = () => {
    const idx = ORDER.indexOf(mode);
    const next = ORDER[(idx + 1) % ORDER.length];
    setMode(next);
  };

  /** 折叠态指示点：unified 用渐变背景 */
  const collapsedDotStyle =
    mode === "unified" ? { background: BRAND_GRADIENT } : undefined;

  return (
    <div className={`px-3 pb-3 ${collapsed ? "flex justify-center" : ""}`}>
      {collapsed ? (
        /* 折叠状态：小圆点指示当前模式，点击循环 */
        <button
          onClick={cycleMode}
          title={`当前：${LABEL[mode]}模式，点击切换到下一个模式`}
          className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-gray-100 transition-colors"
        >
          <div
            className={`w-2.5 h-2.5 rounded-full transition-colors duration-300 ${DOT_COLOR[mode]}`}
            style={collapsedDotStyle}
          />
        </button>
      ) : (
        /* 展开状态：分段选择器（三段） */
        <div className="flex flex-col gap-1.5">
          <p className="text-[10px] text-gray-400 font-medium px-0.5">成员管理模式</p>

          {/* 分段选择器容器 */}
          <div className="flex items-center bg-gray-100 rounded-[4px] p-0.5 gap-0.5">
            {/* 普通模式（custom） */}
            <button
              onClick={() => setMode("custom")}
              className={`
                flex-1 flex items-center justify-center gap-1 h-7 rounded-[4px] text-xs font-medium
                transition-all duration-200 select-none
                ${isCustom
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
              className={`
                flex-1 flex items-center justify-center gap-1 h-7 rounded-[4px] text-xs font-medium
                transition-all duration-200 select-none
                ${isStandard
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
              className={`
                flex-1 flex items-center justify-center gap-1 h-7 rounded-[4px] text-xs font-medium
                transition-all duration-200 select-none
                ${isUnified
                  ? "bg-white shadow-sm shadow-black/10 font-semibold"
                  : "text-gray-500 hover:text-gray-700"
                }
              `}
              style={isUnified ? { color: "#5856D6" } : undefined}
            >
              <span
                className="w-1.5 h-1.5 rounded-full flex-shrink-0 transition-all duration-200"
                style={isUnified ? { background: BRAND_GRADIENT } : { background: "#9ca3af" }}
              />
              统一
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
