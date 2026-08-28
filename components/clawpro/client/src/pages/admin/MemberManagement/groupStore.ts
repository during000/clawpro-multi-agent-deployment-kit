/**
 * groupStore - 组织 / 项目的跨页共享数据源
 *
 * 背景：「用户管理-组织视图」与「项目资产管理」原本各自读取静态 MOCK_GROUPS 的本地副本，
 * 两页互不联动。按需求，两页对「组织 + 项目」的新建/重命名/删除/加子组织都要双向同步，
 * 因此把组织树提升为跨页共享 Store：localStorage 持久化 + window CustomEvent 广播。
 *
 * 数据范围：完整的可编辑组织树（manual / oneid-group）+ 项目（source='project'）。
 * 说明：OneID 只读部门（oneid-dept）仍由组织视图按其同步演示逻辑本地叠加，不进入本 Store。
 */
import { createLibraryStore } from '../SkillLibrary/libraryStore';
import { MOCK_GROUPS, MOCK_PROJECTS } from './mock';
import type { UserGroup } from './types';

const GROUP_STORE_CACHE_KEY = 'clawpro_group_tree_cache';
const GROUP_STORE_VERSION_KEY = 'clawpro_group_tree_cache_version';
// v2：项目改为单层级（不再有子项目 parentId），提升版本以清除旧的层级缓存
const GROUP_STORE_VERSION = '2';

export const GROUP_STORE_EVENT = 'group-store-updated';

export const groupStore = createLibraryStore<UserGroup>({
  cacheKey: GROUP_STORE_CACHE_KEY,
  versionKey: GROUP_STORE_VERSION_KEY,
  version: GROUP_STORE_VERSION,
  initialData: [...MOCK_GROUPS, ...MOCK_PROJECTS],
  getId: (g) => g.id,
  eventName: GROUP_STORE_EVENT,
});

/** 收集某组织及其全部后代 id（用于级联删除） */
export function collectGroupSubtreeIds(groups: UserGroup[], rootId: string): Set<string> {
  const ids = new Set<string>([rootId]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const g of groups) {
      if (g.parentId && ids.has(g.parentId) && !ids.has(g.id)) {
        ids.add(g.id);
        changed = true;
      }
    }
  }
  return ids;
}

/** 删除组织及其全部子组织（级联） */
export function removeGroupSubtree(rootId: string): void {
  const all = groupStore.getAll();
  const toRemove = collectGroupSubtreeIds(all, rootId);
  groupStore.replaceAll(all.filter((g) => !toRemove.has(g.id)));
}
