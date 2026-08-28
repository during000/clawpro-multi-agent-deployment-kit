/**
 * 用户 / 组织统一数据 hook（对齐"成员管理"页）
 *
 * 目标：
 *   - 让所有需要"员工库 + 分组"数据的页面（CreateAgentDialog / 未来的批量绑定弹窗等）
 *     和 MemberManagement 页共用**同一份**数据源；
 *   - 跟随 useAdminMode 自动切换 OneID / 普通模式下的用户与分组集合；
 *   - MemberManagement 侧任何编辑（加入/移出组织、改上级、同步等）
 *     完成 mutate 后调用 `notifyUsersChanged()`，所有 subscribers 立即刷新。
 *
 * 为什么不做成全局 store：
 *   现阶段是纯 mock，`MOCK_USERS` / `MOCK_GROUPS` 直接就是模块导出的顶层数组，
 *   MemberManagement 里已经在 mutate 这些数组（如 `MOCK_USERS[idx] = ...`）。
 *   引入 Zustand 反而要维护双写。我们做的是"最小侵入 + 事件总线"：
 *   保留原来的顶层数组，抽出一个 `notifyUsersChanged()` 让 mutate 方主动触发。
 *   将来接后端时，只要把 `readUsers` / `readGroups` 换成 API 请求即可，UI 层零改动。
 */
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  MOCK_USERS,
  MOCK_USERS_MANUAL,
  MOCK_GROUPS,
  MOCK_MANUAL_GROUPS,
  getPrimaryDeptPath,
} from "@/pages/admin/MemberManagement/mock";
import type { UserOrg, UserGroup } from "@/pages/admin/MemberManagement/types";
import { useAdminMode } from "@/contexts/AdminModeContext";

// ─── 事件总线 ────────────────────────────────────────────
type Listener = () => void;
const userListeners = new Set<Listener>();
const groupListeners = new Set<Listener>();

/** 通知所有订阅者：底层 MOCK_USERS / MOCK_USERS_MANUAL 已被 mutate。 */
export function notifyUsersChanged() {
  userListeners.forEach((l) => l());
}
/** 通知所有订阅者：底层 MOCK_GROUPS / MOCK_MANUAL_GROUPS 已被 mutate。 */
export function notifyGroupsChanged() {
  groupListeners.forEach((l) => l());
}

// ─── 分组语义 ────────────────────────────────────────────
/** 一条"用户所属分组"记录，附带语义类别，供 UI 分组显示。 */
export interface UserGroupItem {
  id: string;
  name: string;
  /** 完整路径（如 "A公司/技术部/后端组"），普通模式无路径概念时降级为 name */
  path: string;
  /** primary=OneID 主部门；secondary=OneID 兼任部门；oneidGroup=OneID 用户组；manual=普通模式自建组织 */
  kind: "primary" | "secondary" | "oneidGroup" | "manual";
  /** primary 但 OneID 侧已删除（primaryGroupValid=false）时置 true —— 用于红点/告警 */
  invalid?: boolean;
  /** 分组自身来源（透传给上层展示，无强绑关系） */
  source: UserGroup["source"];
}

// ─── 数据读取（现阶段直接读顶层数组） ─────────────────────
function readUsers(hasOneid: boolean): UserOrg[] {
  return hasOneid ? MOCK_USERS : MOCK_USERS_MANUAL;
}
function readGroups(hasOneid: boolean): UserGroup[] {
  return hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS;
}

// ─── Hooks ───────────────────────────────────────────────

/** 当前模式下的全量用户列表（会随 MemberManagement 侧编辑自动刷新）。 */
export function useUsers(): UserOrg[] {
  const { hasOneid } = useAdminMode();
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const l: Listener = () => setTick((n) => n + 1);
    userListeners.add(l);
    return () => {
      userListeners.delete(l);
    };
  }, []);
  return useMemo(() => readUsers(hasOneid).slice(), [hasOneid, tick]);
}

/** 当前模式下的全量组织列表。 */
export function useGroups(): UserGroup[] {
  const { hasOneid } = useAdminMode();
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const l: Listener = () => setTick((n) => n + 1);
    groupListeners.add(l);
    return () => {
      groupListeners.delete(l);
    };
  }, []);
  return useMemo(() => readGroups(hasOneid).slice(), [hasOneid, tick]);
}

/** id → 组织 的映射，UI 需要 name/path 查询时用。 */
export function useGroupsMap(): Map<string, UserGroup> {
  const groups = useGroups();
  return useMemo(() => new Map(groups.map((g) => [g.id, g])), [groups]);
}

/** 单个用户（不存在返回 null）。 */
export function useUserById(userId: string | null | undefined): UserOrg | null {
  const users = useUsers();
  return useMemo(() => {
    if (!userId) return null;
    return users.find((u) => u.userId === userId) ?? null;
  }, [users, userId]);
}

/**
 * 按 userId 模糊搜索（前 `limit` 条）。q 为空时返回空数组，避免一次性铺满下拉。
 */
export function useUserSearch(q: string, limit = 8): UserOrg[] {
  const users = useUsers();
  return useMemo(() => {
    const key = q.trim().toLowerCase();
    if (!key) return [];
    return users.filter((u) => u.userId.toLowerCase().includes(key)).slice(0, limit);
  }, [users, q, limit]);
}

/**
 * 给定用户，返回**语义化的所属分组列表**：
 *   - 主部门排最前，若 `primaryGroupValid=false` 打 `invalid` 标记；
 *   - 兼任 OneID 部门次之；
 *   - OneID 用户组次之；
 *   - 普通模式下自建组织统一归为 `manual`。
 *
 * 注意：只返回**当前用户在 `groupIds` 里实际持有**的分组，
 *       与 MemberManagement 用户列表口径一致。
 */
export function useUserGroupItems(user: UserOrg | null | undefined): UserGroupItem[] {
  const groupsMap = useGroupsMap();
  return useMemo(() => {
    if (!user) return [];
    const items: UserGroupItem[] = [];
    const seen = new Set<string>();

    // 1. 主部门
    if (user.primaryGroupId) {
      const g = groupsMap.get(user.primaryGroupId);
      // OneID 侧被删除时组织已不在 map 里；仍保留一条 invalid 记录方便管理员看到
      items.push({
        id: user.primaryGroupId,
        name: g?.name ?? user.primaryGroupId,
        path: g ? getPrimaryDeptPath(user.primaryGroupId, [...groupsMap.values()]) : user.primaryGroupId,
        kind: "primary",
        invalid: !user.primaryGroupValid || !g,
        source: g?.source ?? "oneid-dept",
      });
      seen.add(user.primaryGroupId);
    }

    // 2. 其它归属
    for (const gid of user.groupIds) {
      if (seen.has(gid)) continue;
      seen.add(gid);
      const g = groupsMap.get(gid);
      if (!g) continue;
      const kind: UserGroupItem["kind"] =
        g.source === "oneid-dept" ? "secondary"
        : g.source === "oneid-group" ? "oneidGroup"
        : "manual";
      items.push({
        id: gid,
        name: g.name,
        path: getPrimaryDeptPath(gid, [...groupsMap.values()]),
        kind,
        source: g.source,
      });
    }

    return items;
  }, [user, groupsMap]);
}

/** 语义类别 → 中文标签（供下拉分组标题） */
export const USER_GROUP_KIND_LABEL: Record<UserGroupItem["kind"], string> = {
  primary: "主部门",
  secondary: "兼任部门",
  oneidGroup: "用户组",
  manual: "所属组织",
};
