/**
 * HowItWorks - 工作原理三栏：左 CTA + 中 Models + 右 Channels/Skills
 * 包含三个交互卡片：大模型、通道、技能
 */
import { useLocation } from "wouter";

const FLOW_STEPS = [
  "用户通过 IM 工具发送消息",
  "消息通过通道传入 OpenClaw",
  "OpenClaw 调用大模型处理请求",
  "如需要，调用相关技能执行任务",
  "将结果返回给用户",
];

export default function HowItWorks() {
  const [, navigate] = useLocation();

  return (
    <div className="section-wrapper how-section">
      <span className="dots-deco dots-deco-left" />
      <div
        className="section-inner"
        style={{ display: "flex", borderLeft: "1px #E2E8F0 solid", borderRight: "1px #E2E8F0 solid" }}
      >
        {/* 左：CTA + 流程列表 */}
        <div className="how-left">
          <div className="how-label">概念介绍</div>
          <div className="how-title">OpenClaw 工作原理</div>
          <div className="how-list">
            {FLOW_STEPS.map((s, i) => (
              <div className="how-item" key={i}>
                <div className="how-dot" />
                <span>{s}</span>
              </div>
            ))}
          </div>
          <div className="how-cta" onClick={() => navigate("/my-openclaw")} role="button" tabIndex={0}>
            <span>开始使用</span>
            <img src="/landing-assets/68.svg" alt="" width={16} height={16} />
          </div>
        </div>

        {/* 中：01 大模型 */}
        <div className="how-col how-col-models">
          <div className="how-card models-card">
            <div className="how-card-header">
              <span className="how-card-num">01/</span>
              <span className="how-card-title">大模型驱动</span>
              <span className="how-card-en">Models</span>
            </div>
            <div className="how-card-desc">
              支持接入腾讯云 DeepSeek、混元、Coding Plan 等主流大模型，也支持自定义模型接入，灵活适配企业需求。
            </div>
            <div style={{ marginTop: 32 }}>
              <div
                style={{
                  fontSize: 14,
                  color: "rgba(0,0,0,0.70)",
                  letterSpacing: "0.21px",
                  marginBottom: 8,
                }}
              >
                主模型
              </div>
              <div className="model-select">
                <div>
                  <span className="name">腾讯云 DeepSeek</span>
                  <span className="version">DeepSeekV3 0324</span>
                </div>
                <img src="/landing-assets/69.svg" alt="" width={16} height={16} />
              </div>
            </div>
            <div className="backup-models">
              <div className="backup-models-label">备用模型（0）</div>
              <div className="backup-input">
                <div className="backup-input-inner">
                  <span className="typewriter">腾讯云 DeepSeek ｜ DeepSeek V3 0324</span>
                  <img src="/landing-assets/70.svg" alt="" width={16} height={16} />
                </div>
                <div className="backup-add">添加</div>
              </div>
              <div className="backup-popover">
                <div className="popover-row">
                  <img src="/landing-assets/71.svg" alt="" width={24} height={24} />
                  <span>腾讯云 DeepSeek（DeepSeek V3 0324）</span>
                </div>
                <div className="popover-row">
                  <img src="/landing-assets/72.svg" alt="" width={24} height={24} />
                  <span>腾讯云混元（混元 TurboS Latest）</span>
                </div>
                <div className="popover-row">
                  <img src="/landing-assets/151.svg" alt="" width={24} height={24} />
                  <span>自定义模型（Claude Opus 4.6）</span>
                </div>
                <div className="popover-row">
                  <img src="/landing-assets/151.svg" alt="" width={24} height={24} />
                  <span>自定义模型（Claude Opus 4.6）</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* 右：02 Channels + 03 Skills 上下排列 */}
        <div className="how-col how-col-wide">
          {/* 02 多通道 */}
          <div className="how-card channels-card">
            <div className="how-card-header">
              <span className="how-card-num">02/</span>
              <span className="how-card-title">多通道覆盖</span>
              <span className="how-card-en">Channels</span>
            </div>
            <div className="how-card-desc">用户与 Agent 交互的入口，支持微信、QQ、飞书等主流工具</div>
            <div className="channels-illust">
              <img
                className="channels-illust-static"
                src="/landing-assets/banner/channels-illust-static.png"
                alt="多通道覆盖"
              />
              <video
                className="channels-illust-video"
                src="/landing-assets/banner/channels-illust.mp4"
                muted
                loop
                playsInline
                preload="none"
                aria-hidden="true"
              />
            </div>
          </div>

          {/* 03 技能扩展 */}
          <div className="how-card skills-card">
            <div className="how-card-header">
              <span className="how-card-num">03/</span>
              <span className="how-card-title">技能扩展</span>
              <span className="how-card-en">Skills</span>
            </div>
            <div className="how-card-desc">为 Agent 添加搜索、绘图等扩展能力</div>

            <div className="install-queue">
              <div className="install-queue-header">
                <span className="install-queue-title">安装队列（2）</span>
                <div style={{ display: "flex", gap: 8 }}>
                  <img src="/landing-assets/77.svg" alt="" width={16} height={16} />
                  <img src="/landing-assets/78.svg" alt="" width={16} height={16} />
                </div>
              </div>
              <div className="install-queue-list">
                <InstallItem name="data-analysis 2.0.0" iconDefault="79" />
                <InstallItem name="obsidian 1.0.0" iconDefault="80" />
                <InstallItem name="video-transcribe 0.7.0" iconDefault="79" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <span className="dots-deco dots-deco-right" />
    </div>
  );
}

function InstallItem({ name, iconDefault }: { name: string; iconDefault: string }) {
  return (
    <div className="install-queue-item">
      <span className="install-queue-item-name">{name}</span>
      <div className="install-queue-status-wrap">
        <div className="install-queue-item-status status-installing default-state">
          <img src={`/landing-assets/${iconDefault}.svg`} alt="" width={12} height={12} />
          <span className="status-text">安装中</span>
        </div>
        <div className="install-queue-item-status status-success hover-state">
          <img src="/landing-assets/81.svg" alt="" width={12} height={12} />
          <span className="status-text">安装成功</span>
        </div>
      </div>
    </div>
  );
}
