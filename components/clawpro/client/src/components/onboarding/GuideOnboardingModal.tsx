/**
 * GuideOnboardingModal - 新手引导默认弹窗
 *
 * 对应场景：首次登录 / 首次进入产品的新手引导弹窗。
 * 与 GuideGlobalModal（版本更新弹窗）不同：
 * - GuideGlobalModal：用于版本/视觉品牌升级公告，强调"变更"
 * - GuideOnboardingModal：用于首次访问的产品引导，强调"欢迎与上手"
 *
 * 端类型（endpoint）：
 *  - tenant: 用户端弹窗（蓝色渐变头部）
 *  - admin:  管控端弹窗（深蓝渐变头部，更稳重）
 *
 * 变体（variant）：
 *  - full:   完整版 —— 渐变头部 + 截图区 + 主标题/副标题 + 双 CTA（主按钮 + 跳过链接）
 *  - simple: 极简版 —— 渐变头部 + 截图区 + 单 CTA（用于轮播次屏或简化场景）
 */
import { X } from "lucide-react";

export type OnboardingEndpoint = "tenant" | "admin";
export type OnboardingModalVariant = "full" | "simple";

interface GuideOnboardingModalProps {
  open: boolean;
  onClose: () => void;
  /** 渐变头部主标题（例："全站视觉品牌升级"） */
  headline?: string;
  /** 端类型 —— 决定头部配色 */
  endpoint?: OnboardingEndpoint;
  /** 变体 —— 完整版 / 极简版 */
  variant?: OnboardingModalVariant;
  /** 底部主标题（完整版显示） */
  title?: string;
  /** 底部副标题/描述（完整版显示） */
  description?: string;
  /** 主 CTA 文案 */
  confirmText?: string;
  /** 次要文案（完整版的"跳过"链接） */
  skipText?: string;
  /** 截图区图片地址 */
  image?: string;
  /** 主 CTA 点击回调 */
  onConfirm?: () => void;
  /** 跳过链接点击回调 */
  onSkip?: () => void;
}

export function GuideOnboardingModal({
  open,
  onClose,
  headline = "全站视觉品牌升级",
  endpoint = "tenant",
  variant = "full",
  title = "更多视觉",
  description = "为你呈现全新的视觉体验，让产品更具品牌感。",
  confirmText = "立即体验",
  skipText = "暂不查看",
  image,
  onConfirm,
  onSkip,
}: GuideOnboardingModalProps) {
  if (!open) return null;

  const handleConfirm = () => {
    onConfirm?.();
    onClose();
  };
  const handleSkip = () => {
    onSkip?.();
    onClose();
  };

  // 端类型决定头部渐变配色
  const headerGradient =
    endpoint === "admin"
      ? "bg-gradient-to-br from-[#2B4DA8] to-[#5A4FE0]" // 管控端：更深的蓝
      : "bg-gradient-to-br from-[#4A90F7] to-[#6C63FF]"; // 用户端：明亮蓝

  // 极简版底部背景颜色与头部呼应
  const simpleBottomBg =
    endpoint === "admin"
      ? "bg-gradient-to-b from-[#5A4FE0]/8 to-white"
      : "bg-gradient-to-b from-[#6C63FF]/8 to-white";

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center">
      {/* 遮罩 */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* 弹窗 */}
      <div className="relative w-full max-w-[480px] mx-4 rounded-2xl overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-300 bg-white">
        {/* ─── 渐变头部区域 ─── */}
        <div className={`relative ${headerGradient} px-8 pt-7 pb-6`}>
          {/* 关闭按钮 */}
          <button
            onClick={onClose}
            className="absolute top-3 right-3 w-6 h-6 rounded-full bg-white/20 hover:bg-white/30 flex items-center justify-center transition-colors"
            aria-label="关闭"
          >
            <X className="w-3.5 h-3.5 text-white" />
          </button>

          {/* 主标题（渐变头部） */}
          <h2 className="text-xl font-semibold text-white text-center tracking-wide">
            {headline}
          </h2>
        </div>

        {/* ─── 截图区域 ─── */}
        <div
          className={`relative ${
            variant === "simple" ? simpleBottomBg : "bg-white"
          } px-6 pt-5 pb-0`}
        >
          <div className="relative rounded-lg overflow-hidden border border-gray-200 shadow-sm bg-white">
            {image ? (
              <img
                src={image}
                alt={title}
                className="w-full h-[200px] object-cover object-top"
              />
            ) : (
              /* 截图占位 —— 区分管控端/用户端的模拟界面 */
              <div className="w-full h-[200px] bg-gradient-to-br from-gray-50 to-gray-100 flex items-center justify-center">
                <div className="w-[92%] h-[88%] rounded border border-gray-200 bg-white shadow-sm flex flex-col">
                  {/* 模拟顶部导航 */}
                  <div className="h-6 border-b border-gray-100 flex items-center px-2 gap-1.5">
                    <div className="w-2 h-2 rounded-full bg-gray-200" />
                    <div className="w-12 h-1.5 rounded bg-gray-200" />
                    <div className="ml-auto w-8 h-1.5 rounded bg-gray-100" />
                  </div>
                  {/* 模拟内容 */}
                  <div className="flex-1 flex">
                    {endpoint === "admin" && (
                      <div className="w-[24%] border-r border-gray-100 p-1.5 space-y-1">
                        <div className="w-full h-1.5 rounded bg-gray-100" />
                        <div className="w-3/4 h-1.5 rounded bg-blue-100" />
                        <div className="w-full h-1.5 rounded bg-gray-100" />
                        <div className="w-2/3 h-1.5 rounded bg-gray-100" />
                        <div className="w-3/4 h-1.5 rounded bg-gray-100" />
                      </div>
                    )}
                    <div className="flex-1 p-2 space-y-1.5">
                      <div className="w-1/3 h-2 rounded bg-gray-200" />
                      <div className="w-full h-1.5 rounded bg-gray-100" />
                      <div className="w-full h-1.5 rounded bg-gray-100" />
                      <div className="w-3/4 h-1.5 rounded bg-gray-100" />
                      {endpoint === "tenant" && (
                        <div className="mt-1.5 grid grid-cols-3 gap-1.5">
                          <div className="h-8 rounded bg-gray-50 border border-gray-100" />
                          <div className="h-8 rounded bg-gray-50 border border-gray-100" />
                          <div className="h-8 rounded bg-gray-50 border border-gray-100" />
                        </div>
                      )}
                      {endpoint === "admin" && (
                        <div className="mt-1.5 space-y-1">
                          <div className="h-3 rounded bg-gray-50 border border-gray-100" />
                          <div className="h-3 rounded bg-gray-50 border border-gray-100" />
                          <div className="h-3 rounded bg-gray-50 border border-gray-100" />
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* ─── 底部内容区域 ─── */}
        <div className="bg-white px-8 pt-5 pb-6 text-center">
          {variant === "full" ? (
            <>
              {/* 主标题 */}
              <h3 className="text-base font-semibold text-gray-900 mb-1.5">
                {title}
              </h3>
              {/* 描述 */}
              <p className="text-sm text-gray-500 leading-relaxed mb-5 max-w-[380px] mx-auto">
                {description}
              </p>
              {/* 主 CTA */}
              <button
                onClick={handleConfirm}
                className="inline-flex items-center justify-center px-6 py-2.5 text-sm font-medium text-white bg-[#1a1a2e] hover:bg-[#16162a] rounded-full transition-colors shadow-sm"
              >
                {confirmText}
              </button>
              {/* 跳过链接 */}
              <div className="mt-3">
                <button
                  onClick={handleSkip}
                  className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
                >
                  {skipText}
                </button>
              </div>
            </>
          ) : (
            /* 极简版：仅主 CTA */
            <button
              onClick={handleConfirm}
              className="inline-flex items-center justify-center px-6 py-2.5 text-sm font-medium text-white bg-[#1a1a2e] hover:bg-[#16162a] rounded-full transition-colors shadow-sm"
            >
              {confirmText}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
