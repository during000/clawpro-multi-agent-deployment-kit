/**
 * Intro - 概念介绍段：单段大字，介绍 OpenClaw 是什么
 */
export default function Intro() {
  return (
    <div className="section-wrapper intro-section">
      <div className="intro-content">
        <div className="intro-deco-1">
          <img src="/landing-assets/61.svg" alt="" />
        </div>
        <div className="intro-title">
          OpenClaw 是一个开源的 <span className="highlight">AI Agent</span> 框架
          <br />
          专为企业和个人用户设计
          <br />
          让你能够快速创建、部署和管理专属的<span className="assistant-highlight">AI 智能助理</span>
        </div>
        <div style={{ position: "absolute", left: "845.50px", top: "114px" }}>
          <img src="/landing-assets/62.svg" alt="" />
        </div>
      </div>
    </div>
  );
}
