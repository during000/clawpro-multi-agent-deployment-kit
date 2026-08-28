/**
 * 公共技能库 Tab —— 外壳容器
 *
 * 职责：在「公共技能库」一级 Tab 下，提供二级 Tab 切换器：
 *  - 公共技能（PublicSkillTab）：单 Skill 列表
 *  - 公共技能包（PublicSkillPackageTab）：Skill 组合模板浏览
 *
 * 视觉规范：使用 SegmentGroup/SegmentOption（管控端 4px 方角标准），
 * 通过 grid grid-cols-2 w-full 让 Segment 等宽撑满，匹配设计稿"长条双栏"形态。
 */
import { useState } from 'react';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import PublicSkillTab from './PublicSkillTab';
import PublicSkillPackageTab from './PublicSkillPackageTab';

interface PublicSkillLibraryTabProps {
  packages: Array<{ id: string; name: string; isActive: boolean; scopeType?: 'all-users' | 'groups'; scopeLabel?: string; groupIds?: string[] }>;
  groups: Array<{ id: string; name: string; parentId?: string | null }>;
  onAddSkillToPackage: (skillId: string, packageId: string) => void;
}

type SubTabId = 'skill' | 'package';

const SUB_TABS: Array<{ id: SubTabId; label: string }> = [
  { id: 'skill', label: '公共技能' },
  { id: 'package', label: '公共技能包' },
];

export default function PublicSkillLibraryTab({
  packages,
  groups,
  onAddSkillToPackage,
}: PublicSkillLibraryTabProps) {
  const [subTab, setSubTab] = useState<SubTabId>('skill');

  return (
    <div className="space-y-4">
      {/* 二级 Tab：SegmentGroup 等宽双栏
          停服态下让"公共技能/公共技能包"二级 Tab 仍可点击：纯导航/查看类操作。
          通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局灰化与点击拦截。 */}
      <SegmentGroup
        aria-label="公共技能库 二级 Tab"
        className="!flex w-full"
        data-billing-exempt
      >
        {SUB_TABS.map((tab) => (
          <SegmentOption
            key={tab.id}
            active={subTab === tab.id}
            onClick={() => setSubTab(tab.id)}
            className="flex-1"
            data-billing-exempt
          >
            {tab.label}
          </SegmentOption>
        ))}
      </SegmentGroup>

      {/* Tab 内容 */}
      {subTab === 'skill' ? (
        <PublicSkillTab
          packages={packages}
          groups={groups}
          onAddSkillToPackage={onAddSkillToPackage}
        />
      ) : (
        <PublicSkillPackageTab
          packages={packages}
          groups={groups}
          onAddSkillToPackage={onAddSkillToPackage}
        />
      )}
    </div>
  );
}
