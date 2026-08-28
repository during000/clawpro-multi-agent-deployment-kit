/**
 * userStore - 用户归属（成员）跨页共享数据源
 *
 * 「用户管理-组织视图」与「项目资产管理」都能给组织/项目「添加/移除用户」，
 * 成员归属需要双向同步，因此把 UserOrg 列表提升为跨页共享 Store。
 * 仅覆盖 OneID/标准模式使用的 MOCK_USERS（与 groupStore 的组织 id 体系一致）。
 */
import { createLibraryStore } from '../SkillLibrary/libraryStore';
import { MOCK_USERS } from './mock';
import type { UserOrg } from './types';

const USER_STORE_CACHE_KEY = 'clawpro_user_org_cache';
const USER_STORE_VERSION_KEY = 'clawpro_user_org_cache_version';
const USER_STORE_VERSION = '2';

export const USER_STORE_EVENT = 'user-store-updated';

export const userStore = createLibraryStore<UserOrg>({
  cacheKey: USER_STORE_CACHE_KEY,
  versionKey: USER_STORE_VERSION_KEY,
  version: USER_STORE_VERSION,
  initialData: MOCK_USERS,
  getId: (u) => u.userId,
  eventName: USER_STORE_EVENT,
});

/** 把一组用户加入某组织/项目 */
export function addUsersToGroup(userIds: string[], groupId: string): void {
  const set = new Set(userIds);
  const next = userStore.getAll().map((u) =>
    set.has(u.userId) && !u.groupIds.includes(groupId)
      ? { ...u, groupIds: [...u.groupIds, groupId] }
      : u,
  );
  userStore.replaceAll(next);
}

/** 把某用户从组织/项目移除 */
export function removeUserFromGroup(userId: string, groupId: string): void {
  const next = userStore.getAll().map((u) =>
    u.userId === userId ? { ...u, groupIds: u.groupIds.filter((g) => g !== groupId) } : u,
  );
  userStore.replaceAll(next);
}
