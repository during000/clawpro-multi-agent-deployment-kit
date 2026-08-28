/**
 * ShareScopeTags - 共享范围已选标签（单行展示，超出 +N）
 *
 * 展示规则（与 AgentCard 卡片、ShareScopeTree 的 isGroupFullyCovered 完全一致）：
 * - 一个组只要被「整组覆盖」（显式勾选整组，或该组所有成员都被勾选）→ 归并为「组名」标签展示；
 * - 未被整组覆盖的成员 → 作为「个人」标签单独展示（并去重）。
 * - 单行溢出隐藏，超出部分用 "+N" 收起。
 *
 * 移除交互：
 * - 移除「组」标签 → 若该组在 groupIds 中则移除 groupId；同时清掉该组所有成员的 userId（兼容“成员全选”归并出的组）。
 * - 移除「个人」标签 → 移除对应 userId。
 */
import { useRef, useState } from "react";
import { Users, User } from "lucide-react";
import type { ShareGroupNode } from "./ShareScopeTree";

interface ShareScopeTagsProps {
  groups: ShareGroupNode[];
  groupIds: string[];
  userIds: string[];
  onRemoveGroup: (id: string) => void;
  onRemoveUser: (id: string) => void;
  /** 点击 "+N" 标签时触发（用于展开二级下拉，查看完整已选项） */
  onExpandClick?: () => void;
}

/** 单行最多展示的标签数量 */
const MAX_VISIBLE_TAGS = 3;

export function ShareScopeTags({ groups, groupIds, userIds, onRemoveGroup, onRemoveUser, onExpandClick }: ShareScopeTagsProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [overflowCount] = useState(0);

  // ===== 统一归并：整组 vs 个人（与卡片展示一致） =====
  // 1. 找出所有「整组覆盖」的组：显式勾选整组，或该组所有成员都被勾选
  const fullyCoveredGroups = groups.filter((g) => {
    if (groupIds.includes(g.id)) return true;
    return g.members.length > 0 && g.members.every((m) => userIds.includes(m.id));
  });
  // 2. 已被整组覆盖的成员 ID 集合（不再单独作为个人展示）
  const coveredMemberIds = new Set(
    fullyCoveredGroups.flatMap((g) => g.members.map((m) => m.id)),
  );

  // 组标签
  const groupTags = fullyCoveredGroups.map((g) => ({
    id: g.id,
    type: "group" as const,
    label: g.name,
  }));

  // 个人标签（剔除已归入整组的成员，去重）
  const userTags: { id: string; type: "user"; label: string }[] = [];
  const seen = new Set<string>();
  for (const uid of userIds) {
    if (coveredMemberIds.has(uid) || seen.has(uid)) continue;
    seen.add(uid);
    const user = groups.flatMap((g) => g.members).find((m) => m.id === uid);
    userTags.push({ id: uid, type: "user", label: user?.name ?? uid });
  }

  // 组标签排前面，个人标签在后
  const tags = [...groupTags, ...userTags];

  const total = tags.length;
  const visibleTags = overflowCount > 0 ? tags.slice(0, MAX_VISIBLE_TAGS - 1) : tags.slice(0, MAX_VISIBLE_TAGS);
  const remaining = total - visibleTags.length;

  return (
    <div ref={containerRef} className="flex items-center gap-1.5 min-h-[26px] overflow-hidden flex-nowrap">
      {visibleTags.map((tag) => (
        <span
          key={`${tag.type}-${tag.id}`}
          className="inline-flex items-center gap-1 h-6 rounded-full border border-[var(--border)] bg-[var(--muted)] px-2.5 text-xs font-medium text-[var(--text-emphasis)] flex-shrink-0"
        >
          {tag.type === "group" ? (
            <Users className="w-3 h-3 text-[var(--text-muted)]" />
          ) : (
            <User className="w-3 h-3 text-[var(--text-muted)]" />
          )}
          {tag.label}
          <button
            type="button"
            onClick={() =>
              tag.type === "group"
                ? onRemoveGroup(tag.id)
                : onRemoveUser(tag.id)
            }
            className="ml-0.5 text-[var(--text-muted)] hover:text-[var(--text-emphasis)] transition-colors"
          >
            ×
          </button>
        </span>
      ))}
      {remaining > 0 && (
        <button
          type="button"
          onClick={onExpandClick}
          className="inline-flex items-center justify-center h-6 rounded-full border border-dashed border-[var(--border)] bg-[var(--muted)]/60 px-2 text-xs text-[var(--text-muted)] font-medium flex-shrink-0 cursor-pointer hover:bg-[var(--muted)] hover:text-[var(--text-emphasis)] transition-colors focus:outline-none"
        >
          +{remaining}
        </button>
      )}
    </div>
  );
}
