/**
 * ScopePopover
 * 模型行内「应用范围」编辑入口：
 *   - 显示当前 scope（"全部用户" / 第一个组织或项目名称 + "+N"）
 *   - 鼠标移上去显示完整组织/项目路径 Tooltip
 *   - 点笔图标打开 ScopeSelect 编辑面板，确认后回调 onSave
 *
 * 复用通用 ScopeSelect，组件本身只承担"展示 + 触发器"职责。
 */
import { useMemo } from "react";
import { Pencil } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MetaText } from "@/components/ui/Typography";
import { ScopeSelect, type ScopeType } from "@/components/ScopeSelect";
import type { ModelRow } from "@/lib/modelConfigStore";
import type { UserGroup } from "../MemberManagement/types";

/** 获取组织的完整路径（如 "全公司/技术部/前端组"） */
export function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const map = new Map(groups.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(groupId);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.join("/");
}

export interface ScopePopoverProps {
  model: ModelRow;
  groups: UserGroup[];
  projects: UserGroup[];
  onSave: (id: string, scope: ScopeType, groupIds: string[]) => void;
}

export function ScopePopover({ model, groups, projects, onSave }: ScopePopoverProps) {
  const selectedScopePaths = useMemo(() => {
    const scopeNodes = [...groups, ...projects];
    return (
      model.visibilityGroupIds
        .map((gid) => getGroupPath(gid, scopeNodes))
        .filter(Boolean)
    );
  }, [groups, projects, model.visibilityGroupIds]);

  const renderScopeText = () => {
    if (model.visibilityScope === "all" || selectedScopePaths.length === 0) {
      return <Badge variant="outline">全部用户</Badge>;
    }

    const firstName = selectedScopePaths[0];
    const rest = selectedScopePaths.length - 1;
    const tooltipText = selectedScopePaths.join("\n");

    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex max-w-full items-center gap-1 cursor-default">
            <Badge variant="secondary" className="max-w-[140px]">
              <span className="block truncate max-w-[124px]">{firstName}</span>
            </Badge>
            {rest > 0 && (
              <Badge variant="secondary">+{rest}</Badge>
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-[320px] whitespace-pre-line">
          <MetaText tone="inherit">{tooltipText}</MetaText>
        </TooltipContent>
      </Tooltip>
    );
  };

  return (
    <div className="inline-flex items-center gap-1.5 min-h-[20px] max-w-[220px]">
      {renderScopeText()}
      <ScopeSelect
        scope={model.visibilityScope}
        selectedGroupIds={model.visibilityGroupIds}
        groups={groups}
        projects={projects}
        showBadges={false}
        align="end"
        trigger={
          <button
            type="button"
            className="self-center text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
            title="编辑应用范围"
            onClick={(e) => e.stopPropagation()}
          >
            <Pencil className="w-3 h-3" />
          </button>
        }
        onConfirm={(scope, groupIds) => {
          onSave(model.id, scope, groupIds);
          toast.success("应用范围已更新");
        }}
      />
    </div>
  );
}
