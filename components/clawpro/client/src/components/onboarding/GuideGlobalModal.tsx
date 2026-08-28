/**
 * GuideGlobalModal - 全局弹窗（严格对齐 Figma 设计稿 node-id=4081-5304）
 *
 * 【优先级规则】当有多个一级弹窗同时出现时，GuideGlobalModal 优先展示。
 * z-index 为 9999，高于所有 Dialog / AlertDialog / Sheet 等一级弹窗（z-50），
 * 确保全局引导弹窗始终在最顶层，不会被业务弹窗遮挡。
 *
 * 变体：
 * - single: 单条内容 — 单视频铺满 + 标题图 + 底部标题/描述/按钮（对应用户端样式）
 * - carousel: 多条内容 — 多视频轮播 + 左右箭头 + 指示器圆点（对应管控端样式）
 *
 * 设计规范（来自 Figma）：
 * - 弹窗尺寸：680×512，圆角 8px
 * - 配图区域：1080×608(@2x)，实际渲染区域 540×304
 * - 主按钮：140×36，渐变背景 linear-gradient(90deg, #020617 70%, #1447E6 100%)
 *   - 管控端：圆角 4px
 *   - 用户端：圆角 60px（胶囊形）
 * - 次级按钮：140×36，白色背景 + #E5E5E5 描边，圆角与主按钮一致
 * - 标题：16px Medium #000，竖线分割（1px宽 12px高 #CCC，gap=10px）
 * - 副标题：12px Regular #737373，letter-spacing -0.0833em
 * - 指示器：激活态 18×4 #000，非激活态 4×4 #CACFDD，gap=4px，圆角 20px
 * - 大标题："全站视觉焕新升级" 24px Semibold，渐变填充 + text-shadow
 * - 底部内容区：flex-column gap=20px，居中对齐
 * - 左右切换箭头：24×24，圆形描边按钮，距弹窗中线 y=191
 * - 关闭按钮：右上角 24×24，圆角左下 20px
 */
import { useState, useRef, useEffect, useCallback } from "react";
import { useFocusTrap } from "./onboardingHooks";


export interface GlobalModalSlide {
  /** 标题左侧文字（carousel 模式用竖线分割显示；为空则不显示竖线，仅显示右侧标题） */
  titleLeft?: string;
  /** 标题右侧文字 */
  titleRight: string;
  /** 副标题描述 */
  desc: string;
  /** 视频地址（与 imageSrc 二选一，优先使用 videoSrc） */
  videoSrc?: string;
  /** 静态图片地址（当没有 videoSrc 时使用） */
  imageSrc?: string;
}

/** 弹窗变体 */
export type GlobalModalVariant = "single" | "carousel";

/** 端类型 */
export type GlobalModalEndpoint = "admin" | "tenant";

interface GuideGlobalModalProps {
  open: boolean;
  onClose: () => void;
  /** 变体 */
  variant: GlobalModalVariant;
  /** 幻灯片内容 */
  slides: GlobalModalSlide[];
  /** CTA 主按钮文案 */
  confirmText?: string;
  /** 次级按钮文案（传入则显示双按钮模式） */
  secondaryText?: string;
  /** CTA 点击回调 */
  onConfirm?: () => void;
  /** 次级按钮点击回调 */
  onSecondary?: () => void;
  /** 端类型 —— 决定按钮圆角风格（admin=4px 方角，tenant=60px 胶囊） */
  endpoint?: GlobalModalEndpoint;
}

export function GuideGlobalModal({
  open,
  onClose,
  variant = "single",
  slides,
  confirmText = "立即体验",
  secondaryText,
  onConfirm,
  onSecondary,
  endpoint = "admin",
}: GuideGlobalModalProps) {
  const [current, setCurrent] = useState(0);

  const videoRefs = useRef<(HTMLVideoElement | null)[]>([]);

  // 重置 current 当 slides 变化或弹窗打开时
  useEffect(() => {
    if (open) setCurrent(0);
  }, [open, slides]);

  // 控制视频播放/暂停
  useEffect(() => {
    if (!open) return;
    videoRefs.current.forEach((v, i) => {
      if (!v) return;
      if (i === current) {
        v.play().catch(() => {});
      } else {
        v.pause();
        v.currentTime = 0;
      }
    });
  }, [current, open]);

  const handleClose = useCallback(() => {
    onClose();
  }, [onClose]);

  const handleConfirm = useCallback(() => {
    onConfirm?.();
    onClose();
  }, [onConfirm, onClose]);

  const handleSecondary = useCallback(() => {
    onSecondary?.();
    onClose();
  }, [onSecondary, onClose]);

  // 无障碍：焦点陷阱 + Esc 关闭 + body 滚动锁（强阻断组件必备）
  const trapRef = useFocusTrap(open, onClose, { dismissible: true });

  if (!open || slides.length === 0) return null;

  const isMulti = variant === "carousel" && slides.length > 1;
  const slide = slides[current];
  const btnRadius = endpoint === "tenant" ? 60 : 4;

  const switchTo = (next: number) => {
    if (next === current) return;
    setCurrent(next);
  };

  const goPrev = () => switchTo((current - 1 + slides.length) % slides.length);
  const goNext = () => switchTo((current + 1) % slides.length);

  return (
    <div
      ref={trapRef}
      className="fixed inset-0 z-[9999] flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-label={slide.titleRight || "版本更新"}
    >
      {/* 遮罩：半透明黑色蒙版，压暗背景页面 + 点击关闭 */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-[1px] animate-in fade-in duration-200"
        onClick={handleClose}
      />

      {/* 弹窗内容容器：固定 680×512，圆角 8px（对齐 Figma） */}
      <div
        style={{
          position: "relative",
          width: 680,
          height: 512,
          borderRadius: 8,
          overflow: "hidden",
          flexShrink: 0,
          zIndex: 1,
        }}
      >
        {/* 底部卡片背景图：铺满弹窗底层 */}
        <img
          src="/landing-assets/onboarding/card-bg.png"
          alt=""
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            width: 680,
            height: 512,
            objectFit: "cover",
            display: "block",
            zIndex: 0,
          }}
        />

        {/* 媒体层（视频 / 图片 / 渐变占位）：淡入淡出 */}
        {slides.map((s, i) =>
          s.videoSrc ? (
            <video
              key={i}
              ref={(el) => { videoRefs.current[i] = el; }}
              src={s.videoSrc}
              autoPlay={i === 0}
              loop
              muted
              playsInline
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: 680,
                height: 512,
                objectFit: "cover",
                display: "block",
                zIndex: i === current ? 1 : 0,
                opacity: i === current ? 1 : 0,
                transition: "opacity 0.35s ease",
              }}
            />
          ) : s.imageSrc ? (
            <img
              key={i}
              src={s.imageSrc}
              alt=""
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: 680,
                height: 310,
                objectFit: "cover",
                display: "block",
                zIndex: i === current ? 1 : 0,
                opacity: i === current ? 1 : 0,
                transition: "opacity 0.35s ease",
              }}
            />
          ) : (
            /* 无视频/图片时使用蓝色渐变占位背景 + 配图区域标注（对齐 Figma） */
            <div
              key={i}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: 680,
                height: 310,
                background: "linear-gradient(135deg, #2B4BC8 0%, #4A6FE8 25%, #7BA3F5 50%, #A8C4FA 75%, #D6E4FC 100%)",
                display: "block",
                zIndex: i === current ? 1 : 0,
                opacity: i === current ? 1 : 0,
                transition: "opacity 0.35s ease",
              }}
            >
              {/* 配图占位区域：从标题下方到蓝色渐变底部，左右留 70px 边距 */}
              <div
                style={{
                  position: "absolute",
                  top: 72,
                  left: 70,
                  right: 70,
                  bottom: 0,
                  borderRadius: "8px 8px 0 0",
                  background: "rgba(255,255,255,0.81)",
                  border: "1px solid #FFFFFF",
                  borderBottom: "none",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              >
                <span
                  style={{
                    color: "#6575A9",
                    fontSize: 16,
                    fontWeight: 400,
                    lineHeight: "28px",
                    fontFamily: '"PingFang SC", sans-serif',
                  }}
                >
                  1080 * 608
                </span>
              </div>
            </div>
          )
        )}

        {/* 固定白色遮罩：从底部渐变遮挡配图，确保底部文字可读 */}
        <div
          style={{
            position: "absolute",
            top: 255,
            left: 0,
            width: 680,
            height: 257,
            background: "linear-gradient(180deg, rgba(255,255,255,0) 3%, rgba(255,255,255,1) 33%)",
            zIndex: 2,
            pointerEvents: "none",
          }}
        />

        {/* 大标题："全站视觉焕新升级 ✦" — 24px Semibold，渐变填充 + text-shadow（双层实现） */}
        <div
          style={{
            position: "absolute",
            top: 30,
            left: "50%",
            transform: "translateX(-50%)",
            zIndex: 3,
            pointerEvents: "none",
            whiteSpace: "nowrap",
          }}
        >
          {/* 底层：text-shadow 投影（不透明白色文字，仅用于产生阴影） */}
          <span
            aria-hidden="true"
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              fontFamily: '"PingFang SC", sans-serif',
              fontSize: 24,
              fontWeight: 600,
              lineHeight: "28px",
              color: "transparent",
              textShadow: "0px 1px 4px rgba(8, 45, 181, 0.14)",
              display: "inline-flex",
              alignItems: "center",
              gap: 6,
            }}
          >
            全站视觉焕新升级 ✦
          </span>
          {/* 上层：渐变填充文字 */}
          <span
            style={{
              position: "relative",
              fontFamily: '"PingFang SC", sans-serif',
              fontSize: 24,
              fontWeight: 600,
              lineHeight: "28px",
              background: "linear-gradient(145deg, #FFFFFF 0%, #E4F7FF 33%, #FFFFFF 62%, #E4F7FF 100%)",
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
              backgroundClip: "text",
              display: "inline-flex",
              alignItems: "center",
              gap: 6,
            }}
          >
            全站视觉焕新升级 ✦
          </span>
        </div>

        {/* 左箭头（多条内容时显示） — 对齐 Figma 设计稿 SVG，距顶部 190px */}
        {isMulti && (
          <button
            onClick={(e) => { e.stopPropagation(); goPrev(); }}
            className="group absolute z-[3] left-[22px] w-6 h-6 p-0 border-none bg-transparent cursor-pointer outline-none"
            style={{ top: 190 }}
            aria-label="上一页"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="0.5" y="0.5" width="23" height="23" rx="11.5" className="stroke-black/[0.29] transition-all duration-200 group-hover:stroke-black/70" />
              <path fillRule="evenodd" clipRule="evenodd" d="M8.72363 12L12.9998 16.2762L13.9426 15.3334L10.6093 12L13.9426 8.66669L12.9998 7.72388L8.72363 12Z" className="fill-black/[0.29] transition-all duration-200 group-hover:fill-black/70" />
            </svg>
          </button>
        )}

        {/* 右箭头（多条内容时显示） — 对齐 Figma 设计稿 SVG，距顶部 190px */}
        {isMulti && (
          <button
            onClick={(e) => { e.stopPropagation(); goNext(); }}
            className="group absolute z-[3] right-[22px] w-6 h-6 p-0 border-none bg-transparent cursor-pointer outline-none"
            style={{ top: 190 }}
            aria-label="下一页"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="0.5" y="0.5" width="23" height="23" rx="11.5" className="stroke-black/[0.29] transition-all duration-200 group-hover:stroke-black/70" />
              <path fillRule="evenodd" clipRule="evenodd" d="M15.2764 12L11.0002 7.7238L10.0574 8.6666L13.3907 12L10.0574 15.3333L11.0002 16.2761L15.2764 12Z" className="fill-black/[0.29] transition-all duration-200 group-hover:fill-black/70" />
            </svg>
          </button>
        )}

        {/* 关闭按钮：右上角 24×24，左下圆角曲线背景（对齐 Figma 设计稿 SVG） */}
        <button
          onClick={handleClose}
          className="group absolute top-0 right-0 z-[3] w-6 h-6 p-0 border-none cursor-pointer outline-none flex items-center justify-center"
          aria-label="关闭引导"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M0 0H24V24H20C8.95431 24 0 15.0457 0 4V0Z" className="fill-white/40 transition-all duration-200 group-hover:fill-white/70" />
            <path d="M17.959 7.1709L15.1299 9.99902L17.959 12.8281L16.8281 13.959L13.999 11.1299L11.1709 13.959L10.04 12.8281L12.8682 9.99902L10.04 7.1709L11.1709 6.04004L13.999 8.86816L16.8281 6.04004L17.959 7.1709Z" className="fill-[#020617]/50 transition-all duration-200 group-hover:fill-[#020617]/80" />
          </svg>
        </button>

        {/* 底部内容区：指示器 + 文本 + 按钮，flex-column gap=20px 居中 */}
        <div
          style={{
            position: "absolute",
            bottom: 0,
            left: 0,
            right: 0,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            gap: 20,
            paddingBottom: 30,
            zIndex: 3,
          }}
        >
          {/* 指示器圆点（多条内容时显示）：激活 18×4，非激活 4×4 */}
          {isMulti && (
            <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
              {slides.map((_, i) => (
                <span
                  key={i}
                  onClick={() => switchTo(i)}
                  style={{
                    width: i === current ? 18 : 4,
                    height: 4,
                    borderRadius: 20,
                    background: i === current ? "#000" : "#CACFDD",
                    cursor: "pointer",
                    transition: "all 0.3s ease",
                  }}
                />
              ))}
            </div>
          )}

          {/* 标题+副标题 */}
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              gap: 8,
              opacity: 1,
              transition: "opacity 0.35s ease",
            }}
            key={current}
          >
            {/* 标题：carousel 模式 + titleLeft 有值时用竖线分割，否则只显示 titleRight */}
            {isMulti && slide.titleLeft ? (
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  gap: 10,
                }}
              >
                <span
                  style={{
                    color: "#000",
                    fontFamily: '"PingFang SC", sans-serif',
                    fontSize: 16,
                    fontWeight: 500,
                    lineHeight: "24px",
                  }}
                >
                  {slide.titleLeft}
                </span>
                <span
                  style={{
                    width: 1,
                    height: 12,
                    background: "#CCC",
                    flexShrink: 0,
                  }}
                />
                <span
                  style={{
                    color: "#000",
                    fontFamily: '"PingFang SC", sans-serif',
                    fontSize: 16,
                    fontWeight: 500,
                    lineHeight: "24px",
                  }}
                >
                  {slide.titleRight}
                </span>
              </div>
            ) : (
              <h3
                style={{
                  color: "#000",
                  textAlign: "center",
                  fontFamily: '"PingFang SC", sans-serif',
                  fontSize: 16,
                  fontWeight: 500,
                  lineHeight: "24px",
                  margin: 0,
                }}
              >
                {slide.titleRight}
              </h3>
            )}

            {/* 副标题 */}
            <p
              style={{
                color: "#737373",
                textAlign: "center",
                fontFamily: '"PingFang SC", sans-serif',
                fontSize: 12,
                fontWeight: 400,
                lineHeight: "20px",
                letterSpacing: "-0.0833em",
                margin: 0,
                maxWidth: 500,
              }}
            >
              {slide.desc}
            </p>
          </div>

          {/* 按钮区域：支持双按钮（次级 + 主按钮），gap=12px 水平排列 */}
          <div
            style={{
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              gap: 12,
              height: 36,
            }}
          >
            {/* 次级按钮（仅当 secondaryText 传入时显示） */}
            {secondaryText && (
              <button
                onClick={handleSecondary}
                style={{
                  width: 140,
                  height: 36,
                  borderRadius: btnRadius,
                  border: "1px solid #E5E5E5",
                  background: "#FFFFFF",
                  color: "#000",
                  fontSize: 14,
                  fontWeight: 500,
                  cursor: "pointer",
                  transition: "background 0.2s, border-color 0.2s",
                  padding: "8px 24px",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontFamily: '"PingFang SC", sans-serif',
                  letterSpacing: "-0.0714em",
                }}
                onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "#F5F5F5"; }}
                onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "#FFFFFF"; }}
              >
                {secondaryText}
              </button>
            )}

            {/* 主按钮：渐变背景 linear-gradient(90deg, #020617 70%, #1447E6 100%) */}
            <button
              onClick={handleConfirm}
              style={{
                width: 140,
                height: 36,
                borderRadius: btnRadius,
                border: "none",
                background: "linear-gradient(90deg, #020617 70%, #1447E6 100%)",
                color: "#fff",
                fontSize: 14,
                fontWeight: 500,
                cursor: "pointer",
                transition: "opacity 0.2s",
                padding: "10px",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontFamily: '"PingFang SC", sans-serif',
                letterSpacing: "-0.0714em",
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.opacity = "0.9"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.opacity = "1"; }}
            >
              {confirmText}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
