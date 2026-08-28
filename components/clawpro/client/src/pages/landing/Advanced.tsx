/**
 * Advanced - 进阶玩法：多模型切换示意 + 4 张卡片
 */
const ADVANCED_CARDS = [
  {
    icon: "/landing-assets/124.svg",
    title: "多模型切换",
    items: [
      "日常对话：使用轻量级模型，响应更快、成本更低",
      "复杂推理：切换到更强大的模型，获得更准确的答案",
      "代码生成：使用专门针对代码优化的模型",
    ],
  },
  {
    icon: "/landing-assets/125.svg",
    title: "研究助手",
    items: ["tavily-search：实时网络搜索", "summarize：内容摘要", "tencent-docs：文档处理"],
  },
  {
    icon: "/landing-assets/126.svg",
    title: "多通道策略",
    items: [
      "企业微信：适合工作场景，与企业内部系统集成",
      "飞书：适合有飞书办公套件的团队",
      "QQ：适合个人使用和非正式沟通",
    ],
  },
  {
    icon: "/landing-assets/127.svg",
    title: "最佳实践",
    items: [
      "为不同场景创建不同的 OpenClaw，避免一个助手承担过多职责",
      "定期检查 Tokens 消耗，合理规划使用量",
      "及时更新技能版本，获取最新功能和修复",
    ],
  },
];

export default function Advanced() {
  return (
    <div className="section-wrapper advanced-section">
      <img className="advanced-illust" src="/landing-assets/banner/multi-model.png" alt="" />
      <div className="section-inner">
        <div className="advanced-header">
          <div className="section-label">进阶玩法</div>
          <div className="section-title" style={{ marginTop: 8 }}>
            OpenClaw 多模型切换
          </div>
          <div className="section-desc" style={{ marginTop: 8 }}>
            你可以为同一个 OpenClaw 配置多个模型，根据不同的任务需求灵活切换
          </div>
        </div>
        <div className="advanced-cards">
          {ADVANCED_CARDS.map((c, i) => (
            <div className="advanced-card" key={i}>
              <img className="advanced-card-icon" src={c.icon} alt="" />
              <div className="advanced-card-title">{c.title}</div>
              <div className="advanced-card-desc">
                <ul>
                  {c.items.map((it, j) => (
                    <li key={j}>{it}</li>
                  ))}
                </ul>
              </div>
            </div>
          ))}
        </div>
      </div>
      <img
        src="/landing-assets/banner/dots.png"
        alt=""
        className="side-deco"
        style={{ right: 0, height: 670, objectFit: "cover", objectPosition: "left center", transform: "scaleX(-1)" }}
      />
    </div>
  );
}
