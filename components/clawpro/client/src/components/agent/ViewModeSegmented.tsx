/**
 * ViewModeSegmented - 管理视图 / 对话视图分段切换（用户端）
 *
 * 基于统一的 TenantSegment 胶囊组件实现（仅用户端可用），不再手写样式。
 * 尺寸由 TenantSegmentGroup size 决定（默认 default = 36px），
 * 颜色 / 阴影 / focus 全部走 token（--shadow-segment / --ring）。
 */
import { LayoutGrid, MessageSquare } from "lucide-react";
import { TenantSegmentGroup, TenantSegmentOption } from "@/components/ui/segment";

export type ViewMode = "card" | "chat";

interface ViewModeSegmentedProps {
  value: ViewMode;
  onChange: (mode: ViewMode) => void;
}

const ITEMS: { key: ViewMode; label: string; Icon: typeof LayoutGrid }[] = [
  { key: "card", label: "管理视图", Icon: LayoutGrid },
  { key: "chat", label: "对话视图", Icon: MessageSquare },
];

export const ViewModeSegmented = ({ value, onChange }: ViewModeSegmentedProps) => {
  return (
    <TenantSegmentGroup size="default" aria-label="视图切换">
      {ITEMS.map(({ key, label, Icon }) => (
        <TenantSegmentOption
          key={key}
          active={value === key}
          onClick={() => onChange(key)}
        >
          <Icon className="w-4 h-4" />
          {label}
        </TenantSegmentOption>
      ))}
    </TenantSegmentGroup>
  );
};
