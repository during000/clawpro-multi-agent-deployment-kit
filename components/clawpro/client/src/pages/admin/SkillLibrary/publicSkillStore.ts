/**
 * publicSkillStore - 公共技能市场只读 Store
 * 公共技能不支持上传，仅供「项目资产管理」勾选加入组织资产。
 * 数据源为公共技能市场 PUBLIC_SKILLS（只读），此处提供统一查询接口。
 */
import { PUBLIC_SKILLS, type PublicSkill } from './publicSkillMockData';

export const PUBLIC_SKILL_STORE_EVENT = 'public-skill-store-updated';

export const publicSkillStore = {
  eventName: PUBLIC_SKILL_STORE_EVENT,
  getAll: (): PublicSkill[] => [...PUBLIC_SKILLS],
  getById: (id: string): PublicSkill | undefined => PUBLIC_SKILLS.find((s) => s.id === id),
  subscribe: (listener: () => void): (() => void) => {
    if (typeof window === 'undefined') return () => {};
    window.addEventListener(PUBLIC_SKILL_STORE_EVENT, listener);
    return () => window.removeEventListener(PUBLIC_SKILL_STORE_EVENT, listener);
  },
};
