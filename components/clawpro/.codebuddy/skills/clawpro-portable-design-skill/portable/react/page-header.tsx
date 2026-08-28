/**
 * Portable PageHeader — ClawPro Portable Design Skill（Admin 优先）
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 PageHeader 时的可移植兜底实现。
 *  - 不依赖 shadcn / Tailwind；样式由 portable/css/page-header.css 提供。
 *  - 三段插槽：title / description / actions，外加 titleAccessory 槽。
 *  - 视觉规范（component-specs/page-header.md §3）：
 *      Title 24px(2xl) medium text-title；Description mt-1 14px text-secondary；
 *      Actions 右侧 gap-2；左信息右操作、支持换行。
 *      底部 spacing 仅 mb-6（默认）/ mb-8（spacing="loose"）两档。
 *  - 主操作必须在右侧 actions 槽；不要每页手写自由排版；description 与 actions
 *    不要挤一行；标题右侧 accessory 用 titleAccessory（自动对齐）。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/page-header.css";
 *
 * 用法：
 *   <PortablePageHeader
 *     title="帮助文档"
 *     description="此处配置的文档将展示在企业用户看到的帮助文档中。"
 *     actions={<PortableButton variant="claw-primary">添加文档</PortableButton>}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortablePageHeaderProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  /** 标题右侧辅助槽（badge / 状态 / 版本号等） */
  titleAccessory?: React.ReactNode;
  /** 右侧操作组 */
  actions?: React.ReactNode;
  /** 底部间距：normal=mb-6（默认）/ loose=mb-8 */
  spacing?: "normal" | "loose";
  className?: string;
  contentClassName?: string;
  titleClassName?: string;
  descriptionClassName?: string;
  actionsClassName?: string;
}

export function PortablePageHeader({
  title,
  description,
  titleAccessory,
  actions,
  spacing = "normal",
  className = "",
  contentClassName = "",
  titleClassName = "",
  descriptionClassName = "",
  actionsClassName = "",
}: PortablePageHeaderProps) {
  const root = [
    "cp-page-header",
    spacing === "loose" && "cp-page-header--loose",
    actions && "cp-page-header--has-actions",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div data-slot="page-header" className={root}>
      <div className={["cp-page-header__content", contentClassName].filter(Boolean).join(" ")}>
        <div className="cp-page-header__title-row">
          <h1 className={["cp-page-header__title", titleClassName].filter(Boolean).join(" ")}>
            {title}
          </h1>
          {titleAccessory}
        </div>
        {description != null && (
          <p
            className={["cp-page-header__desc", descriptionClassName]
              .filter(Boolean)
              .join(" ")}
          >
            {description}
          </p>
        )}
      </div>
      {actions != null && (
        <div
          className={["cp-page-header__actions", actionsClassName]
            .filter(Boolean)
            .join(" ")}
        >
          {actions}
        </div>
      )}
    </div>
  );
}
