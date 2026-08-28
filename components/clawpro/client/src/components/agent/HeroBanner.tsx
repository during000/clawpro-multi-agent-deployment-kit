/**
 * HeroBanner - 我的 Agent 页面顶部欢迎条幅
 *
 * 严格 1:1 对齐 Figma 节点 1077:33962：
 *
 *   1077:33962 「页面引导语」
 *     ├ column / center / gap 8 / width 1680
 *     ├ 左右内边距统一由页面外层（MyOpenClaw `padding: 0 120px`）提供，本组件不再叠加
 *     ├ background: 透明（无底色）
 *     ├─ 标题: PingFang SC Medium 26/35.56 / letter -4.27% / fill #0A0A0A
 *     └─ Frame 2147227892 (row / align-center / gap 12 / hug × hug)
 *         ├─ 副文案: PingFang SC Regular 12/22.22 / letter 1.5% / #737373
 *         └─ 1077:34351 「查看步骤指引」按钮 - row / center / padding 2 8 / radius 3
 *             ├ fill: #E1F2FF
 *             ├ stroke 1px #C4D9F5
 *             └ 文字: PingFang SC Regular 12/20 #020617
 *
 * NOTE:
 *   1. HeroBanner 区域无任何底色/渐变背景，保持纯白。
 *   2. HeroBanner 与下方 QuickStartGuide 通过段间距（QuickStartGuide `mb-5`）分离，
 *      已不依赖底部分割线作为视觉接缝。
 *   3. 左右内边距由页面外层 `padding: 0 120px` 提供，本组件 padding 为 0。
 */
interface HeroBannerProps {
  /**
   * 当 QuickStartGuide 已关闭时传入此回调，HeroBanner 副文右侧会渲染
   * 「查看步骤指引」按钮，点击重新唤起 QuickStartGuide。
   * 不传则不渲染按钮（QuickStart 展开态）。
   */
  onShowQuickStart?: () => void;
}

export const HeroBanner = ({ onShowQuickStart }: HeroBannerProps) => {
  return (
    // 外层包裹严格固定 112px，与下方紧贴；防止内部副文+按钮换行时把容器撑高
    // data-guide：供步骤指引气泡（GuideHighlightBubble）按真实位置贴合标注，避免随机浮层
    <div className="relative" data-guide="tenant-hero" style={{ height: "112px" }}>
      {/* 1077:33962 - 页面引导语：高 112 / padding 0 / 无底色 / column / center / gap 8 */}
      <div
        style={{
          height: "112px",
          padding: 0,
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          alignItems: "flex-start",
          gap: "8px",
          overflow: "hidden",
        }}
      >
        {/* 主标题：纯黑 #0A0A0A */}
        <h1
          style={{
            fontFamily:
              "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
            fontWeight: 500,
            fontSize: "26px",
            lineHeight: "35.56px",
            letterSpacing: "-4.27%",
            margin: 0,
            color: "var(--foreground)",
          }}
        >
          快速创建你的专属 AI 助理 ✨
        </h1>

        {/*
          副文 + 「查看步骤指引」按钮 - Figma 1077:34354 row gap 12 align-center hug
          只有传入 onShowQuickStart 时才渲染按钮（即 QuickStart 已被关闭时）
          高度恒定锁在 26px 防止两态切换时标题跳动。
        */}
        <div
          className="flex items-center"
          style={{ gap: "12px", height: "26px" }}
        >
          {/* 副文案：muted-foreground */}
          <p
            style={{
              fontFamily:
                "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
              fontWeight: 400,
              fontSize: "12px",
              lineHeight: "22.22px",
              letterSpacing: "1.5%",
              color: "var(--muted-foreground)",
              margin: 0,
            }}
          >
            对话即可完成各种工作任务，多模型接入、多平台链接，随时随地提升工作效率
          </p>

          {/*
            1077:34351 「查看步骤指引」按钮
            row / center / padding 2px 8px / gap 10 / radius 3
            fill #E1F2FF / stroke 1px #C4D9F5
          */}
          {onShowQuickStart && (
            <button
              type="button"
              onClick={onShowQuickStart}
              className="inline-flex items-center justify-center transition-colors"
              style={{
                padding: "2px 8px",
                gap: "10px",
                borderRadius: "3px",
                background: "#E1F2FF",
                border: "1px solid #C4D9F5",
              }}
            >
              <span
                style={{
                  fontFamily:
                    "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
                  fontWeight: 400,
                  fontSize: "12px",
                  lineHeight: "20px",
                  color: "var(--text-emphasis)",
                  whiteSpace: "nowrap",
                }}
              >
                查看步骤指引
              </span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
