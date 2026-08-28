import * as React from "react";

import { cn } from "@/lib/utils";

/**
 * Typography（用户端文字语义组件）
 *
 * 默认颜色全部绑定到 `--text-*` 语义 token（TYPOGRAPHY_COLOR_MIGRATION_PLAN §2/§3.2）：
 * - --text-title     #0F172A 页面/模块标题（slate-900）
 * - --text-emphasis  #020617 强强调、关键数字（slate-950）
 * - --text-body      #1E293B 普通正文（slate-800）
 * - --text-secondary #334155 次级正文、描述、表格次要字段（slate-700）
 * - --text-muted     #64748B 时间、备注、辅助信息（slate-500）
 * - --text-weak      #94A3B8 占位、空状态、极弱提示、HelperText（slate-400）
 * - --text-brand     #1447E6 链接、活跃态、品牌强调
 * - --text-danger    #DC2626 删除、错误、危险操作
 *
 * 调整任意一档亮度，改 index.css 的 --text-* 即可，不必搜索全站。
 */
export const typographyColorTokens = {
  primary: "text-[var(--text-title)]",
  emphasis: "text-[var(--text-emphasis)]",
  body: "text-[var(--text-body)]",
  secondary: "text-[var(--text-secondary)]",
  muted: "text-[var(--text-muted)]",
  weak: "text-[var(--text-weak)]",
  helper: "text-[var(--text-weak)]",
  brand: "text-[var(--text-brand)]",
  danger: "text-[var(--text-danger)]",
  inherit: "text-inherit",
} as const;

export type TypographyColorToken = keyof typeof typographyColorTokens;

type TypographyProps<T extends React.ElementType> = {
  as?: T;
  /** 颜色 token；不传时使用该语义组件的默认 token */
  tone?: TypographyColorToken;
  className?: string;
} & Omit<React.ComponentPropsWithoutRef<T>, "as" | "className" | "color">;

type TypographyComponent<DefaultElement extends React.ElementType> = <
  T extends React.ElementType = DefaultElement,
>(
  props: TypographyProps<T> & {
    ref?: React.ComponentPropsWithRef<T>["ref"];
  },
) => React.ReactElement | null;

function createTypography<DefaultElement extends React.ElementType>(
  displayName: string,
  defaultAs: DefaultElement,
  baseClassName: string,
  defaultTone: TypographyColorToken,
): TypographyComponent<DefaultElement> {
  const Component = React.forwardRef<Element, TypographyProps<React.ElementType>>(
    ({ as, tone, className, ...props }, ref) => {
      const Comp = as ?? defaultAs;
      const resolvedTone = (tone ?? defaultTone) as TypographyColorToken;

      return React.createElement(Comp, {
        ...props,
        ref,
        className: cn(baseClassName, typographyColorTokens[resolvedTone], className),
      });
    },
  );

  Component.displayName = displayName;

  return Component as TypographyComponent<DefaultElement>;
}

export const TenantHeroTitle = createTypography(
  "TenantHeroTitle",
  "h1",
  "font-sans text-[26px] font-medium leading-[35.56px] tracking-[-0.0427em]",
  "primary",
);

export const TenantPageTitle = createTypography(
  "TenantPageTitle",
  "h1",
  "font-sans text-2xl font-medium leading-[1.4]",
  "primary",
);

export const TenantDocTitle = createTypography(
  "TenantDocTitle",
  "h1",
  "font-sans text-xl font-semibold leading-[1.4]",
  "primary",
);

export const SectionTitle = createTypography(
  "SectionTitle",
  "h2",
  "font-sans text-lg font-medium leading-[1.4]",
  "primary",
);

export const PanelTitle = createTypography(
  "PanelTitle",
  "h2",
  "font-sans text-base font-semibold leading-[1.4]",
  "primary",
);

export const CardTitle = createTypography(
  "CardTitle",
  "h3",
  "font-sans text-sm font-medium leading-[1.5]",
  "primary",
);

export const BodyText = createTypography(
  "BodyText",
  "p",
  "font-sans text-sm font-normal leading-[1.5]",
  "body",
);

export const BodyMedium = createTypography(
  "BodyMedium",
  "span",
  "font-sans text-sm font-medium leading-[1.5]",
  "emphasis",
);

export const CompactText = createTypography(
  "CompactText",
  "span",
  "font-sans text-[13px] font-normal leading-[1.5]",
  "secondary",
);

export const MiniBodyText = createTypography(
  "MiniBodyText",
  "span",
  "font-sans text-xs font-normal leading-[1.5]",
  "body",
);

export const MetaText = createTypography(
  "MetaText",
  "span",
  "font-sans text-xs font-normal leading-[1.5]",
  "muted",
);

/**
 * HelperText - 弹窗输入框/区块下方说明文本
 *
 * 用法：弹窗（Dialog / AlertDialog）内 Input 下方的提示语、Section 标题副说明、
 *      表头副说明、空态提示、虚线占位框文案等所有「最弱辅助说明」都用本组件。
 *
 * 视觉规范：text-xs / font-normal / 中性浅灰 #A3A3A3（neutral-400，无色彩倾向）。
 *           与 MetaText（默认 muted slate-500，含蓝灰倾向）不同：在弹窗内使用更柔和、
 *           更不抢眼的纯中性灰，避免与 Label（secondary）、Input 文本（body）抢视觉权重。
 *
 * 默认渲染为 <p>（语义化段落），需要内联时通过 `as="span"` 覆盖。
 *
 * 示例：
 *   <HelperText>仅支持英文字母、数字和下划线，需与对应插件名保持一致</HelperText>
 *   <HelperText>暂未添加凭证字段</HelperText>
 *   <HelperText as="span">写入配置文件的字段名</HelperText>
 */
export const HelperText = createTypography(
  "HelperText",
  "p",
  "font-sans text-xs font-normal leading-[1.5]",
  "muted",
);

export const MetaMedium = createTypography(
  "MetaMedium",
  "span",
  "font-sans text-xs font-medium leading-[1.5]",
  "muted",
);

export const SmallBodyText = createTypography(
  "SmallBodyText",
  "span",
  "font-sans text-xs font-normal leading-3 tracking-[0.18px]",
  "emphasis",
);

export const TinyText = createTypography(
  "TinyText",
  "span",
  "font-en text-[10px] font-semibold leading-none tracking-[0.02em]",
  "brand",
);

export const StatNumber = createTypography(
  "StatNumber",
  "span",
  "font-din text-2xl font-semibold leading-none tabular-nums",
  "emphasis",
);

export const InlineNumber = createTypography(
  "InlineNumber",
  "span",
  "font-din text-sm leading-[1.5] tabular-nums",
  "body",
);

export const CodeText = createTypography(
  "CodeText",
  "code",
  "font-mono text-xs leading-[1.5]",
  "secondary",
);

export const StepText = createTypography(
  "StepText",
  "span",
  "font-mono text-sm font-medium leading-none",
  "brand",
);

export const UrlText = createTypography(
  "UrlText",
  "span",
  "font-sans text-sm font-normal leading-[1.5] break-all text-[#020617]",
  "inherit",
);
