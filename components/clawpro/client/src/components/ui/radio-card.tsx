/**
 * RadioCard - 单选卡片组件
 *
 * 规范：
 *   - 默认：1px 边框 #EAEEF4 白底
 *   - hover：边框 #1447E6/40
 *   - 选中：边框 #1447E6，背景 #1447E6/5
 *   - 圆角：4px
 *
 * 变体：
 *   - default（默认）：横向布局，左侧 RadioGroupItem 圆点 + 右侧标题 / 描述。
 *   - icon：横向布局，左侧图标（32×32）+ 右侧上标题下描述（不显示 radio 圆点，
 *           选中态完全靠边框 + 背景表达，配合 role="radio" 保持无障碍）。
 *
 * 用法（配合 RadioGroup 使用）：
 *   <RadioGroup value={v} onValueChange={setV}>
 *     <RadioCard id="opt-a" value="a" checked={v === "a"} title="选项 A" description="描述" />
 *     <RadioCard
 *       variant="icon"
 *       id="opt-b" value="b" checked={v === "b"}
 *       icon={<img src="..." alt="" className="w-6 h-6 object-contain" />}
 *       title="飞书" description="同步飞书内的通讯录数据"
 *     />
 *   </RadioGroup>
 */
import * as React from "react";
import { cn } from "@/lib/utils";
import { RadioGroupItem } from "./radio-group";
import { HelperText } from "@/components/ui/Typography";

type RadioCardVariant = "default" | "icon";

interface RadioCardProps {
  id: string;
  value: string;
  checked?: boolean;
  disabled?: boolean;
  title: React.ReactNode;
  description?: React.ReactNode;
  /** 选中态时额外叠加的卡片 className，用于定制特殊颜色（如 native 橙色） */
  checkedClassName?: string;
  /** RadioGroupItem 选中态覆盖 className（如 native 橙色 radio 点） */
  radioCheckedClassName?: string;
  /** 卡片底部附加内容（如 checkbox 确认项） */
  children?: React.ReactNode;
  /** 布局变体，默认 default */
  variant?: RadioCardVariant;
  /**
   * 标题字号档位（不影响描述/子内容）：
   *   - md（默认）：14px 正文级，适用于顶层、独立的选择场景；
   *   - sm：12px 数据级，适用于密集列表或处于次级层级（外层已有更高层级标题）的场景，
   *         避免选项标题字号超过上层主信息造成层级倒挂。
   */
  size?: "md" | "sm";
  /**
   * 标题字重（不影响描述/子内容）：
   *   - semibold（默认）：加粗标题，突出选项主信息；
   *   - normal：常规字重，用于标题需与同区正文（如 MetaText）保持一致视觉权重的场景。
   */
  titleWeight?: "semibold" | "normal";
  /**
   * icon 变体使用的图标节点（如 <img />、<svg />、Lucide 图标）。
   * default 变体会忽略此属性。
   */
  icon?: React.ReactNode;
  /**
   * 仅对 icon 变体生效：图标不再包裹 32×32 白底方形容器，直接裸露图标本身
   * （用于图标自带形状/背景的场景，如圆形头像 AgentAvatar），同时收窄卡片内边距。
   */
  bareIcon?: boolean;
  /**
   * 使用用户端业务卡标准圆角（--radius-card = 12px）替代默认 4px。
   */
  radiusCard?: boolean;
  /**
   * 全圆角（胶囊/药丸形，rounded-full），优先级高于 radiusCard。
   * 适用于带头像的角色类型选项卡等横向紧凑卡片。
   */
  pill?: boolean;
  /** 仅对 icon 变体生效：自定义最外层 className */
  className?: string;
}

function RadioCard({
  id,
  value,
  checked = false,
  disabled,
  title,
  description,
  checkedClassName,
  radioCheckedClassName,
  children,
  variant = "default",
  size = "md",
  titleWeight = "semibold",
  icon,
  bareIcon = false,
  radiusCard = false,
  pill = false,
  className,
}: RadioCardProps) {
  // ─── icon 变体：左图标 + 右（上标题 / 下描述），横向布局 ─────────────
  if (variant === "icon") {
    return (
      <label
        htmlFor={id}
        className={cn(
          "group relative flex items-center transition-colors outline-none border",
          pill ? "rounded-full" : radiusCard ? "rounded-[var(--radius-card)]" : "rounded-[4px]",
          // 裸图标（圆形头像等）：收窄内边距与图文间距，选项卡整体等比缩小
          bareIcon ? "gap-1.5 px-2 py-1.5" : "gap-3 px-3 py-3",
          "border-gray-200 bg-white",
          !checked && !disabled && "hover:border-blue-500/40 cursor-pointer",
          checked && (checkedClassName ?? "border-blue-500 bg-blue-500/5"),
          disabled && "cursor-not-allowed opacity-60",
          className,
        )}
      >
        {/* 视觉无 radio 圆点，但保留 RadioGroupItem 维持无障碍语义与表单值 */}
        <RadioGroupItem
          id={id}
          value={value}
          disabled={disabled}
          className="sr-only"
        />
        {/* 图标：bareIcon 时直接渲染图标本身（保留其自有形状，如圆形头像）；
            否则包裹 32×32 白底圆角 + 灰描边的方形容器 */}
        {icon &&
          (bareIcon ? (
            <span className="shrink-0 inline-flex items-center justify-center">
              {icon}
            </span>
          ) : (
            <div className="w-8 h-8 rounded-[4px] bg-white border border-gray-200 flex items-center justify-center overflow-hidden shrink-0">
              {icon}
            </div>
          ))}
        {/* 文本区：上标题 / 下描述 */}
        <div className="flex-1 min-w-0">
          <p
            className={cn(
              "text-gray-950 leading-snug truncate",
              titleWeight === "normal" ? "font-normal" : "font-semibold",
              size === "sm" ? "text-xs" : "text-sm",
            )}
          >
            {title}
          </p>
          {description && (
            <HelperText as="p" className="mt-0.5 leading-relaxed">
              {description}
            </HelperText>
          )}
          {children}
        </div>
      </label>
    );
  }

  // ─── default 变体：横向 radio + 标题 + 描述 ────────────────────────────
  return (
    <label
      htmlFor={id}
      className={cn(
        "flex items-start gap-2.5 rounded-[4px] border px-3 py-3 transition-colors",
        "border-gray-200 bg-white",
        !checked && !disabled && "hover:border-blue-500/40 cursor-pointer",
        checked && (checkedClassName ?? "border-blue-500 bg-blue-500/5"),
        disabled && "cursor-not-allowed opacity-60",
        className,
      )}
    >
      <RadioGroupItem
        id={id}
        value={value}
        disabled={disabled}
        className={cn(
          "mt-0.5 shrink-0",
          checked && radioCheckedClassName,
        )}
      />
      <div className="flex-1 min-w-0">
        <div
          className={cn(
            "font-medium text-gray-950 mb-0.5 leading-snug",
            size === "sm" ? "text-xs" : "text-sm",
          )}
        >
          {title}
        </div>
        {description && (
          <HelperText as="p" className="leading-relaxed">{description}</HelperText>
        )}
        {children}
      </div>
    </label>
  );
}

export { RadioCard };
