/**
 * Enterprise - 企业版核心优势 2x2 卡片 + 右侧大图
 */
const ENTERPRISE_CARDS = [
  {
    icon: "/landing-assets/114.svg",
    title: "企业级安全管控",
    desc: (
      <>
        完善的用户权限管理，Tokens 配额控制
        <br />
        操作审计日志，确保 AI 使用在可控范围内
      </>
    ),
  },
  {
    icon: "/landing-assets/115.svg",
    title: "集中化配置管理",
    desc: "管理员可统一配置可用模型、通道和帮助文档，用户无需关心底层配置，专注于使用 AI 提升工作效率。",
  },
  {
    icon: "/landing-assets/116.svg",
    title: "多用户协同",
    desc: "支持企业内多名用户各自创建和管理专属 OpenClaw，统一在企业账号体系下管理，互不干扰。",
  },
  {
    icon: "/landing-assets/117.svg",
    title: "实时监控与审计",
    desc: "全面的运营监控面板，实时掌握 OpenClaw 运行状态和 Tokens 消耗情况，操作记录全程可追溯。",
  },
];

export default function Enterprise() {
  return (
    <div className="section-wrapper enterprise-section">
      <span className="dots-deco dots-deco-left" />
      <div
        className="section-inner"
        style={{ borderLeft: "1px #E2E8F0 solid", borderRight: "1px #E2E8F0 solid", height: "100%" }}
      >
        <div className="enterprise-content">
          <div className="enterprise-left">
            <div className="enterprise-header">
              <div className="section-label">核心优势</div>
              <div className="section-title" style={{ marginTop: 8 }}>
                企业版核心优势
              </div>
              <div className="section-desc" style={{ marginTop: 8 }}>
                专为企业场景设计，提供完善的管控能力和极致的使用体验
              </div>
            </div>
            <div className="enterprise-grid">
              {ENTERPRISE_CARDS.map((c, i) => (
                <div className="enterprise-card" key={i}>
                  <img className="enterprise-card-icon" src={c.icon} alt="" />
                  <div className="enterprise-card-title">{c.title}</div>
                  <div className="enterprise-card-desc">{c.desc}</div>
                </div>
              ))}
            </div>
          </div>
          <div className="enterprise-right">
            <img
              className="enterprise-illust"
              src="/landing-assets/banner/enterprise-illust.png"
              alt="企业版核心优势"
            />
          </div>
        </div>
      </div>
      <span className="dots-deco dots-deco-right" />
    </div>
  );
}
