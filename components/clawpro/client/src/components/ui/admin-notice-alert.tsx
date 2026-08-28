import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type AdminNoticeAlertType =
  | "product-news"
  | "pending-config"
  | "resource-alert"
  | "service-suspended";

interface AdminNoticeAlertProps {
  type: AdminNoticeAlertType;
  children?: ReactNode;
  controls?: ReactNode;
  className?: string;
  /** 覆盖默认 tag 标签文案（如不传使用 type 对应的默认文案） */
  tagLabel?: string;
  /** 设为 true 时隐藏左侧 tag 标签 */
  hideTag?: boolean;
  /** 快捷 title/description 模式 */
  title?: ReactNode;
  description?: ReactNode;
}

const TYPE_CONFIG: Record<
  AdminNoticeAlertType,
  {
    label: string;
    tagBackground: string;
    tagBorder: string;
    tagText: string;
    icon: "sparkle" | "alert";
  }
> = {
  "product-news": {
    label: "产品动态",
    tagBackground: "linear-gradient(139deg, #EFF3FF 18%, #F3F6FF 51%, #ECF1FF 100%)",
    tagBorder: "#C6D4FF",
    tagText: "#2547B1",
    icon: "sparkle",
  },
  "pending-config": {
    label: "待配置",
    tagBackground: "linear-gradient(139deg, #FFF2E6 18%, #FFF9F4 51%, #FFF2E6 100%)",
    tagBorder: "#F8DDC4",
    tagText: "#D76610",
    icon: "alert",
  },
  "resource-alert": {
    label: "资源告警",
    tagBackground: "linear-gradient(139deg, #FFF2E6 18%, #FFF9F4 51%, #FFF2E6 100%)",
    tagBorder: "#F8DDC4",
    tagText: "#D76610",
    icon: "alert",
  },
  "service-suspended": {
    label: "服务到期",
    tagBackground: "linear-gradient(139deg, #FFECEC 18%, #FFF4F4 51%, #FFECEC 100%)",
    tagBorder: "#F8C4C4",
    tagText: "#C8311C",
    icon: "alert",
  },
};

function AdminNoticeSparkleIcon() {
  return (
    <svg width="11" height="10" viewBox="0 0 11 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path
        d="M6.0625 3.67969L6.14062 3.85938L6.32031 3.9375L8.75098 5L6.32031 6.0625L6.14062 6.14062L6.0625 6.32031L5 8.75098L3.9375 6.32031L3.85938 6.14062L3.67969 6.0625L1.24805 5L3.67969 3.9375L3.85938 3.85938L3.9375 3.67969L5 1.24805L6.0625 3.67969ZM8.30566 7.78711L8.37988 7.93652L8.53027 8.01172L8.76758 8.12988L8.53027 8.24805L8.37988 8.32324L8.30566 8.47266L8.18652 8.71094L8.06836 8.47266L7.99414 8.32324L7.84375 8.24805L7.60547 8.12988L7.84375 8.01172L7.99414 7.93652L8.06836 7.78711L8.18652 7.54785L8.30566 7.78711Z"
        stroke="currentColor"
      />
    </svg>
  );
}

function AdminNoticeAlertIcon() {
  return (
    <svg width="11" height="11" viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="5.30357" cy="5.30357" r="4.80357" stroke="currentColor" />
      <path d="M4.71484 2.35718H5.89342V6.28575H4.71484V2.35718Z" fill="currentColor" />
      <path d="M4.71484 7.07146H5.89342V8.25003H4.71484V7.07146Z" fill="currentColor" />
    </svg>
  );
}

export function AdminNoticeAlert({ type, children, controls, className, tagLabel, hideTag, title, description }: AdminNoticeAlertProps) {
  const config = TYPE_CONFIG[type];

  // 渲染内容：优先使用 title/description 快捷模式，否则使用 children
  const renderContent = () => {
    if (title || description) {
      return (
        <div className="flex flex-col gap-0.5">
          {title && <span className="font-medium">{title}</span>}
          {description && <span className="text-gray-600">{description}</span>}
        </div>
      );
    }
    return children;
  };

  return (
    <div
      role="alert"
      className={cn(
        "flex h-10 w-full items-center gap-2.5 rounded-[4px] border border-white bg-white/75 px-3 text-xs leading-[18px] text-gray-950",
        className,
      )}
    >
      {!hideTag && (
        <div
          className="inline-flex h-[22px] shrink-0 items-center gap-[5px] rounded-[2px] border px-[6px] text-[11px] leading-[18px]"
          style={{
            background: config.tagBackground,
            borderColor: config.tagBorder,
            color: config.tagText,
          }}
        >
          <span
            className="inline-flex size-[11px] items-center justify-center"
            style={{
              color:
                config.icon === "alert"
                  ? type === "service-suspended"
                    ? "#C8311C"
                    : "#EE7A23"
                  : config.tagText,
            }}
          >
            {config.icon === "sparkle" ? <AdminNoticeSparkleIcon /> : <AdminNoticeAlertIcon />}
          </span>
          <span className="whitespace-nowrap">{tagLabel ?? config.label}</span>
        </div>
      )}
      <div className="flex min-w-0 flex-1 items-baseline gap-x-1">
        {renderContent()}
      </div>
      {controls ? <div className="shrink-0">{controls}</div> : null}
    </div>
  );
}
