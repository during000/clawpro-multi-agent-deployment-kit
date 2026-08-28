/**
 * skillStore - 企业技能库共享 Store
 * 提升自 SkillListTab 内的 mock + localStorage 逻辑，供工具库与「项目资产管理」共享读写。
 */
import { createLibraryStore } from './libraryStore';
import { MOCK_SKILLS } from './mockData';
import type { Skill } from './types';

const SKILLS_CACHE_KEY = 'skillhub_enterprise_skills_cache';
const SKILLS_CACHE_VERSION_KEY = 'skillhub_enterprise_skills_cache_version';
const SKILLS_CACHE_VERSION = '11';

export const SKILL_STORE_EVENT = 'skill-store-updated';

export const skillStore = createLibraryStore<Skill>({
  cacheKey: SKILLS_CACHE_KEY,
  versionKey: SKILLS_CACHE_VERSION_KEY,
  version: SKILLS_CACHE_VERSION,
  initialData: MOCK_SKILLS,
  getId: (s) => s.id,
  eventName: SKILL_STORE_EVENT,
  reviver: (s: any): Skill => ({
    ...s,
    uploadTime: new Date(s.uploadTime),
    lastDistributionTime: s.lastDistributionTime ? new Date(s.lastDistributionTime) : undefined,
  }),
});
