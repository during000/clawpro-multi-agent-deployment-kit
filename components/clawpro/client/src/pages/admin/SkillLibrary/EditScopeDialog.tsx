/**
 * 编辑应用范围 Popover 气泡
 * - 统一引用模型配置同款的 ScopeSelect（confirm 模式）
 * - 与 ModelConfig/ScopePopover 保持一致的交互与样式
 */
import { Edit2 } from 'lucide-react';
import { ScopeSelect, type ScopeType, type ScopeGroup } from '@/components/ScopeSelect';
import { type SkillScope, type Group } from './types';

interface EditScopePopoverProps {
  groups: Group[];
  /** 可选：项目列表，传入后「按组织」面板支持同时选组织和项目 */
  projects?: ScopeGroup[];
  currentScope: SkillScope;
  currentGroupIds: string[];
  onConfirm: (scope: SkillScope, groupIds: string[]) => void;
  /** 应用范围展示标签 */
  scopeLabels: string[];
  /** 是否是 public 范围 */
  isPublic: boolean;
}

export default function EditScopePopover({
  groups,
  projects,
  currentScope,
  currentGroupIds,
  onConfirm,
  scopeLabels,
}: EditScopePopoverProps) {
  const scope: ScopeType = currentScope === 'public' ? 'all' : 'groups';

  const scopeGroups = groups.map((g) => ({ id: g.id, name: g.name }));

  return (
    <ScopeSelect
      mode="confirm"
      scope={scope}
      selectedGroupIds={currentGroupIds}
      groups={scopeGroups}
      projects={projects}
      onConfirm={(newScope, newGroupIds) => {
        const skillScope: SkillScope = newScope === 'all' ? 'public' : 'private';
        onConfirm(skillScope, newGroupIds);
      }}
      showBadges
      scopeLabels={scopeLabels}
      maxVisibleBadges={1}
      trigger={
        <button
          onClick={(e) => e.stopPropagation()}
          className="p-0.5 text-[var(--text-weak)] hover:text-[var(--text-title)] rounded-[var(--radius-lg)] transition-colors flex-shrink-0"
          title="编辑应用范围"
        >
          <Edit2 className="w-3 h-3" />
        </button>
      }
      align="start"
    />
  );
}
