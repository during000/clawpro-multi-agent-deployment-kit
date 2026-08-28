/**
 * Features - 永辉版特色功能区
 * 布局：上 2 张大卡 + 下 3 张小卡
 * 每张卡片插图：normal 显示静态图，hover 切换为视频播放
 */
import FeatureMedia from "./FeatureMedia";
import Footer from "./Footer";

const FEATURES = [
  {
    num: "[01]",
    tag: "永续在线",
    title: "云端部署，7×24 随时可用",
    desc: "Agent 运行在云端，无需安装任何软件，手机、电脑随时打开即用，出差途中也不中断",
    staticSrc: "/landing-assets/yh-features/1_static.png",
    videoSrc: "/landing-assets/yh-features/anim1.mp4",
  },
  {
    num: "[02]",
    tag: "开箱即用",
    title: "3 分钟上手，步骤简单到不像话",
    desc: "创建 Agent 后，只需配置好模型和对话通道，3 分钟内就能开始和 Agent 对话，全程不需要任何技术背景",
    staticSrc: "/landing-assets/yh-features/2_static.png",
    videoSrc: "/landing-assets/yh-features/anim2.mp4",
  },
  {
    num: "[03]",
    tag: "多端接入",
    title: "多种通道，随时随地发起对话",
    desc: "既可以直接在网页上对话，也可以接入企微、飞书、钉钉等聊天软件，在你最顺手的地方使用",
    staticSrc: "/landing-assets/yh-features/3_static.png",
    videoSrc: "/landing-assets/yh-features/anim3.mp4",
  },
  {
    num: "[04]",
    tag: "丰富技能",
    title: "一个 Agent，搞定多种任务",
    desc: "内置海量技能，从文档整理、数据分析到联网搜索，按需开启，让 Agent 真正接管你的重复劳动",
    staticSrc: "/landing-assets/yh-features/4_static.png",
    videoSrc: "/landing-assets/yh-features/anim4.mp4",
  },
  {
    num: "[05]",
    tag: "持续学习",
    title: "越用越懂你，持续积累记忆",
    desc: "Agent 能记住你的偏好、常用表达和使用习惯，用得越久，回答越贴合你的实际需求",
    staticSrc: "/landing-assets/yh-features/5_static.png",
    videoSrc: "/landing-assets/yh-features/anim5.mp4",
  },
];

export default function Features() {
  return (
    <section className="yh-features">
      {/* ========== Section Header ========== */}
      <div className="yh-section-header">
        <div className="yh-section-badge">
          Agent 基本概念与特色
        </div>
        <h2 className="yh-section-title">
          你的专属AI助手，随时待命
        </h2>
        <p className="yh-section-desc">
          无需技术背景，3 分钟创建属于你的智能伙伴
        </p>
      </div>

      {/* ========== 上排 2 大卡 ========== */}
      <div className="yh-features-row yh-features-row-top">
        {FEATURES.slice(0, 2).map((f, i) => (
          <article
            key={i}
            className="yh-feature-card yh-feature-card-lg"
          >
            <div className="yh-feature-card-header">
              <span className="yh-feature-num">{f.num}</span>
              <span className="yh-feature-tag">{f.tag}</span>
            </div>
            <div className="yh-feature-card-body">
              <h3 className="yh-feature-title">{f.title}</h3>
              <p className="yh-feature-desc">{f.desc}</p>
            </div>
            <FeatureMedia
              staticSrc={f.staticSrc}
              videoSrc={f.videoSrc}
              alt={f.tag}
              className="yh-feature-visual yh-feature-visual-lg"
            />
          </article>
        ))}
      </div>

      {/* ========== 下排 3 小卡 ========== */}
      <div className="yh-features-row yh-features-row-bottom">
        {FEATURES.slice(2).map((f, i) => (
          <article
            key={i + 2}
            className="yh-feature-card yh-feature-card-sm"
          >
            <div className="yh-feature-card-header">
              <span className="yh-feature-num">{f.num}</span>
              <span className="yh-feature-tag">{f.tag}</span>
            </div>
            <div className="yh-feature-card-body-sm">
              <h3 className="yh-feature-title-sm">{f.title}</h3>
              <p className="yh-feature-desc-sm">{f.desc}</p>
            </div>
            <FeatureMedia
              staticSrc={f.staticSrc}
              videoSrc={f.videoSrc}
              alt={f.tag}
              className="yh-feature-visual yh-feature-visual-sm"
            />
          </article>
        ))}
      </div>

      {/* Footer 内嵌在 Features 区底部 */}
      <Footer />
    </section>
  );
}
