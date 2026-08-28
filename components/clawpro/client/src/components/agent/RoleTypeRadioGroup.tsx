/**
 * RoleTypeRadioGroup — 用户端「角色类型」单选选项卡组件
 *
 * 由 RoleManageSheet「切换为 / 角色类型」等场景抽取封装，统一「头像 + 名称 + 全圆角单选卡片」
 * 的网格视觉与交互，避免各处内联复刻（改一处不同步其他处）。
 *
 * 视觉规范（沿用 RadioCard icon + bareIcon + pill 变体）：
 *   - 卡片：白底、1px 灰边框、全圆角（rounded-full 胶囊形）
 *   - hover：边框 #1447E6/40
 *   - 选中：边框 #1447E6，背景 #1447E6/5
 *   - 内容：左侧 20px 圆形头像（AgentAvatar）+ 右侧角色名（12px 常规字重）
 *   - 布局：默认 4 列网格（grid-cols-4），间距 10px（gap-2.5）
 *
 * 用法：
 *   <RoleTypeRadioGroup
 *     value={value}
 *     onValueChange={setValue}
 *     options={[
 *       { value: "__general__", label: "通用助手" },
 *       { value: "role-1", label: "程序员" },
 *     ]}
 *   />
 *
 * 说明：
 *   - option.avatarRoleName 缺省时以 option.label 作为头像映射键（AgentAvatar 内部含别名归一）。
 *   - idPrefix 用于生成稳定唯一的 RadioGroupItem id，同页多组时请传入不同前缀。
 */
import { RadioGroup } from "@/components/ui/radio-group";
import { RadioCard } from "@/components/ui/radio-card";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { cn } from "@/lib/utils";

export interface RoleTypeOption {
  /** 选项值（RadioGroup 的 value，须唯一且非空） */
  value: string;
  /** 展示名称 */
  label: string;
  /** 头像映射键，缺省时回退到 label */
  avatarRoleName?: string;
  /** 是否禁用该项 */
  disabled?: boolean;
}

export interface RoleTypeRadioGroupProps {
  /** 选项列表 */
  options: RoleTypeOption[];
  /** 当前选中值 */
  value: string;
  /** 选中变化回调 */
  onValueChange: (value: string) => void;
  /** 网格列数，默认 4 */
  columns?: number;
  /** RadioGroupItem id 前缀（同页多组时传入不同前缀以保证唯一），默认 "role-type" */
  idPrefix?: string;
  /** 头像尺寸（px），默认 20 */
  avatarSize?: number;
  /** 外层网格附加 className */
  className?: string;
  /** 无障碍标签 */
  "aria-label"?: string;
}

/** 列数 → Tailwind grid-cols 映射（避免动态类名被 purge） */
const GRID_COLS: Record<number, string> = {
  1: "grid-cols-1",
  2: "grid-cols-2",
  3: "grid-cols-3",
  4: "grid-cols-4",
  5: "grid-cols-5",
  6: "grid-cols-6",
};

function RoleTypeRadioGroup({
  options,
  value,
  onValueChange,
  columns = 4,
  idPrefix = "role-type",
  avatarSize = 20,
  className,
  "aria-label": ariaLabel = "角色类型",
}: RoleTypeRadioGroupProps) {
  return (
    <RadioGroup
      value={value}
      onValueChange={onValueChange}
      aria-label={ariaLabel}
      className={cn("grid gap-2.5", GRID_COLS[columns] ?? "grid-cols-4", className)}
    >
      {options.map((opt) => (
        <RadioCard
          key={opt.value}
          variant="icon"
          size="sm"
          titleWeight="normal"
          bareIcon
          radiusCard
          pill
          id={`${idPrefix}-${opt.value}`}
          value={opt.value}
          checked={value === opt.value}
          disabled={opt.disabled}
          title={opt.label}
          icon={<AgentAvatar roleName={opt.avatarRoleName ?? opt.label} size={avatarSize} />}
        />
      ))}
    </RadioGroup>
  );
}

export { RoleTypeRadioGroup };
