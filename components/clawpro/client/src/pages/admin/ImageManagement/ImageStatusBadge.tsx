/**
 * ImageStatusBadge - 镜像状态徽章（同步项目通用 StatusTag dot 样式）
 *
 * 状态映射：
 *   - available → 可用（绿）
 *   - creating  → 创建中（蓝）
 *   - failed / error → 异常（红）
 *
 * 主表行（AgentTypesTable）与切换镜像弹窗（SwitchImageDialog）共用。
 */
import { StatusTag } from "@/components/ui/status-tag";

const STATUS_MAP: Record<string, { text: string; variant: "green" | "blue" | "red" | "gray" }> = {
  available: { text: "可用", variant: "green" },
  creating: { text: "创建中", variant: "blue" },
  failed: { text: "异常", variant: "red" },
  error: { text: "异常", variant: "red" },
};

export function ImageStatusBadge({ status }: { status: string }) {
  const c = STATUS_MAP[status] ?? STATUS_MAP.available;
  return (
    <StatusTag mode="text" variant={c.variant}>
      {c.text}
    </StatusTag>
  );
}
