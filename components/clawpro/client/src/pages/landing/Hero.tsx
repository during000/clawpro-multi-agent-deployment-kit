/**
 * Hero - 永辉版首屏
 * 渐变背景 + 标题区 + CTA + 浮动装饰 + 视觉复合层
 * 中央 Logo 与 Navbar、Footer 同步（来自 brandLogo store）
 *
 * 入场动画：所有 .yh-reveal 元素由父级 LandingPage 在挂载后统一触发，
 * 各元素通过 inline style 的 --yh-d 控制错峰延迟。
 */
import { useLocation } from "wouter";

export default function Hero() {
  const [, navigate] = useLocation();

  return (
    <section className="yh-hero">
      {/* ========== 内容区（垂直 flex 居中） ========== */}
      <div className="yh-hero-content">
        {/* Logo + Badge 容器（Badge 盖在 Logo 右上角） */}
        <div className="yh-hero-logo-wrapper yh-reveal" style={{ ["--yh-d" as never]: "0.15s" }}>
          <div
            className="yh-hero-badge"
          >
            <span>Enterprise</span>
          </div>
          <div className="yh-hero-icon-card">
            <img
              src="/landing-assets/yh-features/hero-logo.png"
              alt=""
              className="yh-hero-icon-card-logo"
            />
          </div>
        </div>

        {/* 大标题（距离大图标 36px：与设计稿 1920 画布 y:408→y:444 一致） */}
        <h1
          className="yh-hero-title yh-reveal"
          style={{ ["--yh-d" as never]: "0.40s" }}
        >
          ClawPro 智能体体验平台
        </h1>

        {/* 副标题区（两行 + 横线装饰，作为一个组） */}
        <div
          className="yh-hero-subtitle-group yh-reveal"
          style={{ ["--yh-d" as never]: "0.60s" }}
        >
          <p className="yh-hero-subtitle-line">快速创建属于你的24小时 AI 私人助理</p>
          <p className="yh-hero-subtitle-line">对话即可完成各种任务，随时随地提升效率</p>
          <img
            className="yh-hero-subtitle-underline"
            src="/landing-assets/yh-features/title-underline.png"
            alt=""
            aria-hidden="true"
          />
        </div>

        {/* CTA 按钮（距离副标题组 36px） */}
        <button
          className="yh-hero-cta yh-reveal"
          style={{ ["--yh-d" as never]: "0.80s" }}
          onClick={() => navigate("/my-openclaw")}
        >
          立即创建
        </button>

        {/* ========== 步骤条（无背景，虚线分隔） ========== */}
        <div
          className="yh-steps-bar yh-reveal"
          style={{ ["--yh-d" as never]: "1.00s" }}
        >
          <div className="yh-step-item">
            <span className="yh-step-num">01/</span>
            <span className="yh-step-label">创建 Agent</span>
          </div>

          <span className="yh-step-divider" />

          <div className="yh-step-item">
            <span className="yh-step-num">02/</span>
            <span className="yh-step-label">配置模型，在浏览器中对话</span>
          </div>

          <span className="yh-step-divider" />

          <div className="yh-step-item">
            <span className="yh-step-num">03/</span>
            <span className="yh-step-label">开启通道，在聊天软件中对话</span>
          </div>
        </div>
      </div>

      {/* ========== 底部滚动提示（距底 48px） ========== */}
      <div className="yh-scroll-hint">
        <div className="yh-scroll-mouse">
          <div className="yh-scroll-wheel" />
        </div>
        <span className="yh-scroll-text">向下滚动了解更多</span>
      </div>
    </section>
  );
}
