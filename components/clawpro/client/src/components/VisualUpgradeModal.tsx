/**
 * VisualUpgradeModal - 全站视觉升级预告弹窗
 * Design: 「流动蓝图」Fluid Blueprint
 * - 进入管控端自动弹出（每次进入，除非点击「不再提醒」）
 * - 单张全宽图片轮播，16:8.5 宽屏比例
 * - 底部组织圆点：用户端（蓝色）3张 + 管控端（紫色）4张
 * - 图片左上角纯文字标签标识所属端
 * - 「不再提醒」写入 localStorage，「知道了」/× 仅本次关闭
 */
import { useState, useEffect, useRef, useCallback } from "react";
import { ChevronLeft, ChevronRight, X, Sparkles } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";

const DISMISS_KEY = "openclaw_visual_upgrade_dismissed_0610";

// 升级日期
const UPGRADE_DATE = new Date("2026-06-12T00:00:00+08:00");

function getDaysLeft(): number {
  const now = new Date();
  const diff = UPGRADE_DATE.getTime() - now.getTime();
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)));
}

// 轮播图片数据（占位，等真实截图后替换 src）
const SLIDES: { label: string; type: "user" | "admin"; src: string | null }[] = [
  { label: "用户端", type: "user", src: null },
  { label: "用户端", type: "user", src: null },
  { label: "用户端", type: "user", src: null },
  { label: "管控端", type: "admin", src: null },
  { label: "管控端", type: "admin", src: null },
  { label: "管控端", type: "admin", src: null },
  { label: "管控端", type: "admin", src: null },
];

const USER_COUNT = SLIDES.filter((s) => s.type === "user").length;
const ADMIN_COUNT = SLIDES.filter((s) => s.type === "admin").length;

export default function VisualUpgradeModal() {
  const [open, setOpen] = useState(false);
  const [current, setCurrent] = useState(0);
  const [paused, setPaused] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const daysLeft = getDaysLeft();
  const { isAdminDisabled } = useServiceStatus();

  useEffect(() => {
    // 停服状态下强制关闭视觉升级预告
    if (isAdminDisabled) {
      setOpen(false);
      return;
    }
    const dismissed = localStorage.getItem(DISMISS_KEY);
    if (!dismissed) setOpen(true);
  }, [isAdminDisabled]);

  const next = useCallback(() => {
    setCurrent((c) => (c + 1) % SLIDES.length);
  }, []);

  const prev = useCallback(() => {
    setCurrent((c) => (c - 1 + SLIDES.length) % SLIDES.length);
  }, []);

  // 自动播放
  useEffect(() => {
    if (!open || paused) return;
    timerRef.current = setInterval(next, 3500);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [open, paused, next]);

  const handleDismiss = () => {
    localStorage.setItem(DISMISS_KEY, "1");
    setOpen(false);
  };

  const handleClose = () => setOpen(false);

  // 停服时或未打开时不渲染
  if (isAdminDisabled || !open) return null;

  const slide = SLIDES[current];

  return (
    <div
      className="fixed inset-0 z-[9999] flex items-center justify-center"
      style={{ background: "rgba(0,0,0,0.45)", backdropFilter: "blur(2px)" }}
    >
      <div
        className="relative bg-white rounded-2xl overflow-hidden"
        style={{
          width: "min(656px, 92vw)",
          boxShadow: "0 24px 64px rgba(0,0,0,0.18), 0 4px 16px rgba(0,0,0,0.08)",
        }}
        onMouseEnter={() => setPaused(true)}
        onMouseLeave={() => setPaused(false)}
      >
        {/* ── 关闭按钮 ── */}
        <button
          onClick={handleClose}
          className="absolute top-4 right-4 z-10 w-7 h-7 flex items-center justify-center rounded-full text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors"
        >
          <X className="w-4 h-4" />
        </button>

        {/* ── 顶部信息区 ── */}
        <div
          className="px-6 pt-5 pb-5"
          style={{ background: "linear-gradient(135deg, #EBF4FF 0%, #F3F0FF 100%)" }}
        >
          {/* 标签 — 蓝色系 + Sparkles icon */}
          <span
            className="inline-flex items-center gap-1.5 text-xs font-semibold px-2.5 py-1 rounded-full mb-3"
            style={{ background: "rgba(0,122,255,0.1)", color: "#007AFF" }}
          >
            <Sparkles className="w-3 h-3" />
            视觉升级预告
          </span>

          {/* 标题行 */}
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-[18px] font-bold text-gray-900 leading-tight mb-1">
                全站视觉升级即将上线
              </h2>
              <p className="text-xs text-gray-500 leading-relaxed" style={{ display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
                我们将于 <span className="font-semibold text-gray-700">2026年6月12日</span> 对用户端与管控端进行视觉系统升级。升级后界面更美观、信息密度更高、屏效比大幅提升，在保持原有操作习惯的基础上，带来更流畅的使用体验。
              </p>
            </div>
            {daysLeft > 0 && (
              <div
                className="flex-shrink-0 flex items-center gap-1 px-3 py-1.5 rounded-full text-white text-xs font-semibold whitespace-nowrap"
                style={{ background: "#007AFF" }}
              >
                距升级还有 <span className="text-sm font-bold">{daysLeft}</span> 天
              </div>
            )}
            {daysLeft === 0 && (
              <div
                className="flex-shrink-0 flex items-center px-3 py-1.5 rounded-full text-white text-xs font-semibold whitespace-nowrap"
                style={{ background: "#34C759" }}
              >
                今日升级上线
              </div>
            )}
          </div>
        </div>

        {/* ── 轮播区 ── */}
        <div className="px-6 pt-4 pb-3">
          {/* 箭头 + 图片容器 整行 */}
          <div className="flex items-center gap-3">
            {/* 左箭头（图片外侧） */}
            <button
              onClick={prev}
              className="flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-full border border-gray-200 bg-white text-gray-500 hover:text-gray-900 hover:border-gray-300 hover:shadow-sm transition-all"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>

            {/* 图片容器，16:8.5 比例，带边框+阴影防白底融合 */}
            <div className="flex-1 relative" style={{ paddingBottom: "calc((100%) * 8.5 / 16)" }}>
              <div
                className="absolute inset-0 rounded-xl overflow-hidden"
                style={{
                  border: "1px solid rgba(0,0,0,0.08)",
                  boxShadow: "0 2px 12px rgba(0,0,0,0.08), inset 0 0 0 1px rgba(255,255,255,0.6)",
                }}
              >
                {/* 图片左上角标签 */}
                <div className="absolute top-3 left-3 z-10">
                  <span
                    className="text-xs font-semibold px-2.5 py-1 rounded-md"
                    style={
                      slide.type === "user"
                        ? { background: "rgba(0,122,255,0.88)", color: "#fff", backdropFilter: "blur(4px)" }
                        : { background: "rgba(88,86,214,0.88)", color: "#fff", backdropFilter: "blur(4px)" }
                    }
                  >
                    {slide.label}
                  </span>
                </div>

                {/* 图片 or 占位 */}
                {slide.src ? (
                  <img
                    src={slide.src}
                    alt={`${slide.label}新版预览`}
                    className="w-full h-full object-cover"
                    draggable={false}
                  />
                ) : (
                  <div
                    className="w-full h-full flex flex-col items-center justify-center gap-2"
                    style={{
                      background:
                        slide.type === "user"
                          ? "linear-gradient(135deg, #EBF4FF 0%, #D6EAFF 100%)"
                          : "linear-gradient(135deg, #EDE9FF 0%, #D8D3FF 100%)",
                    }}
                  >
                    <p
                      className="text-sm font-medium"
                      style={{ color: slide.type === "user" ? "#007AFF" : "#5856D6" }}
                    >
                      {slide.label}新版预览图片
                    </p>
                    <p className="text-xs text-gray-400">图片待上传</p>
                  </div>
                )}
              </div>
            </div>

            {/* 右箭头（图片外侧） */}
            <button
              onClick={next}
              className="flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-full border border-gray-200 bg-white text-gray-500 hover:text-gray-900 hover:border-gray-300 hover:shadow-sm transition-all"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>

          {/* ── 组织圆点指示器 ── */}
          <div className="flex items-center justify-center gap-4 mt-4">
            {/* 用户端组 */}
            <div className="flex flex-col items-center gap-1.5">
              <span className="text-[10px] text-gray-400 font-medium">用户端</span>
              <div className="flex items-center gap-1.5">
                {Array.from({ length: USER_COUNT }).map((_, i) => {
                  const idx = i;
                  const active = current === idx;
                  return (
                    <button
                      key={idx}
                      onClick={() => setCurrent(idx)}
                      className="rounded-full transition-all duration-200"
                      style={{
                        width: active ? 20 : 7,
                        height: 7,
                        background: active ? "#007AFF" : "rgba(0,122,255,0.25)",
                      }}
                    />
                  );
                })}
              </div>
            </div>

            {/* 分隔线 */}
            <div className="w-px h-6 bg-gray-200 self-end mb-0.5" />

            {/* 管控端组 */}
            <div className="flex flex-col items-center gap-1.5">
              <span className="text-[10px] text-gray-400 font-medium">管控端</span>
              <div className="flex items-center gap-1.5">
                {Array.from({ length: ADMIN_COUNT }).map((_, i) => {
                  const idx = USER_COUNT + i;
                  const active = current === idx;
                  return (
                    <button
                      key={idx}
                      onClick={() => setCurrent(idx)}
                      className="rounded-full transition-all duration-200"
                      style={{
                        width: active ? 20 : 7,
                        height: 7,
                        background: active ? "#5856D6" : "rgba(88,86,214,0.25)",
                      }}
                    />
                  );
                })}
              </div>
            </div>
          </div>
        </div>

        {/* ── 底部操作栏 ── */}
        <div
          className="flex items-center justify-end px-6 py-3"
          style={{ borderTop: "1px solid #F0F0F0" }}
        >
          <button
            onClick={handleClose}
            className="px-6 py-2 rounded-lg text-sm font-semibold text-white transition-opacity hover:opacity-90"
            style={{ background: "#007AFF" }}
          >
            知道了
          </button>
        </div>
      </div>
    </div>
  );
}
