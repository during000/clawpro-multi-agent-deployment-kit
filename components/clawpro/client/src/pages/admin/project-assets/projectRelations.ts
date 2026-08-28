/**
 * projectRelations - 「项目资产管理」项目成员 / 项目实例的取数逻辑
 * - 成员：复用用户管理的 MOCK_USERS，按 groupId 过滤（groupIds 包含该组织即为成员）
 * - 实例：直接复用管控端 Agent 列表页的 MOCK_CLAWS（真实形态：ins-xxx 实例ID、
 *   email 用户ID、ClawStatus 状态），保证与「Agent 列表」展示口径一致。
 *   归属关系按节点 id 做确定性映射（演示用，刷新稳定可复现）：
 *     · 组织：本组织 + 其子组织的实例聚合，每个实例归属唯一一个组织（本组织或某子组织）；
 *     · 项目：项目单层无层级，每个实例归属 1~N 个项目（一定含当前项目）。
 */
import type { UserGroup, UserOrg } from '../MemberManagement/types';
import { MOCK_CLAWS, type Claw } from '../OpenClawMonitor';

/** 该组织成员（groupIds 包含该组织的用户） */
export function getProjectMembers(groupId: string, users: UserOrg[]): UserOrg[] {
  return users.filter((u) => u.groupIds.includes(groupId));
}

/** 尚未加入该组织的候选用户（供添加成员弹窗） */
export function getCandidateUsers(groupId: string, users: UserOrg[]): UserOrg[] {
  return users.filter((u) => !u.groupIds.includes(groupId));
}

/**
 * 组织聚合成员（含本组织 + 所有子孙组织，按 userId 去重）。
 * 上级组织应统计其本级与全部下级组织的成员，故用此函数；
 * 项目单层无下级，请继续用 getProjectMembers（直接成员即可）。
 */
export function getOrgMembersDeep(
  rootId: string,
  groups: UserGroup[],
  users: UserOrg[],
): UserOrg[] {
  const subtreeIds = new Set(getOrgSubtreeIds(rootId, groups));
  const seen = new Set<string>();
  const result: UserOrg[] = [];
  for (const u of users) {
    if (seen.has(u.userId)) continue;
    if (u.groupIds.some((gid) => subtreeIds.has(gid))) {
      seen.add(u.userId);
      result.push(u);
    }
  }
  return result;
}

/** 组织 / 项目全链路名称，如「A公司 / 技术部 / 前端组」；项目单层则为项目名 */
export function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const map = new Map(groups.map((g) => [g.id, g]));
  const parts: string[] = [];
  let cur = map.get(groupId);
  while (cur) {
    parts.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return parts.length > 0 ? parts.join(' / ') : groupId;
}

/**
 * 某组织的子孙组织 id（含自身，BFS 顺序：自身在前）。
 * 仅统计组织（source !== 'project'），项目不参与组织聚合。
 */
export function getOrgSubtreeIds(rootId: string, groups: UserGroup[]): string[] {
  const orgs = groups.filter((g) => g.source !== 'project');
  const childrenMap = new Map<string, UserGroup[]>();
  for (const g of orgs) {
    const pid = g.parentId ?? '__root__';
    if (!childrenMap.has(pid)) childrenMap.set(pid, []);
    childrenMap.get(pid)!.push(g);
  }
  const result: string[] = [];
  const queue: string[] = [rootId];
  const seen = new Set<string>();
  while (queue.length > 0) {
    const id = queue.shift()!;
    if (seen.has(id)) continue;
    seen.add(id);
    result.push(id);
    for (const child of childrenMap.get(id) ?? []) {
      queue.push(child.id);
    }
  }
  return result;
}

function hashString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

/**
 * 按节点 id 确定性映射出一部分 Agent 实例（演示用）：
 * 同一 id 始终返回相同子集，不同节点返回不同子集。
 */
function pickInstancesForNode(nodeId: string): Claw[] {
  const all = MOCK_CLAWS;
  if (all.length === 0) return [];
  const h = hashString(nodeId);
  const count = Math.min(all.length, 3 + (h % 4)); // 3~6 个
  const start = h % all.length;
  const picked: Claw[] = [];
  const seen = new Set<string>();
  for (let i = 0; i < count; i++) {
    const inst = all[(start + i) % all.length];
    if (!seen.has(inst.id)) {
      seen.add(inst.id);
      picked.push(inst);
    }
  }
  return picked;
}

/** 组织实例行：附带该实例归属的组织 id（本组织或某子组织） */
export type OrgInstanceRow = Claw & { orgId: string };

/**
 * 组织实例（聚合本组织 + 子组织）：
 * 遍历子树（自身在前），每个节点取其确定性实例子集；
 * 同一实例只归属第一个（离根最近）命中它的组织，保证「每个实例组织唯一」。
 */
export function getOrgInstanceRows(rootId: string, groups: UserGroup[]): OrgInstanceRow[] {
  const subtree = getOrgSubtreeIds(rootId, groups);
  const rows: OrgInstanceRow[] = [];
  const seen = new Set<string>();
  for (const orgId of subtree) {
    for (const inst of pickInstancesForNode(orgId)) {
      if (seen.has(inst.id)) continue;
      seen.add(inst.id);
      rows.push({ ...inst, orgId });
    }
  }
  return rows;
}

/** 项目实例行：附带该实例归属的项目 id 列表（一定含当前项目） */
export type ProjectInstanceRow = Claw & { projectIds: string[] };

/**
 * 项目实例（项目单层，无聚合）：
 * 取当前项目的确定性实例子集；每个实例再确定性附加 0~2 个其它项目，
 * 保证「每个实例一定含当前项目，且可能属于多个项目」。
 */
export function getProjectInstanceRows(projectId: string, groups: UserGroup[]): ProjectInstanceRow[] {
  const otherProjects = groups
    .filter((g) => g.source === 'project' && g.id !== projectId)
    .map((g) => g.id);
  return pickInstancesForNode(projectId).map((inst) => {
    const projectIds = [projectId];
    if (otherProjects.length > 0) {
      const base = hashString(inst.id + projectId);
      const extra = base % 3; // 0~2 个额外项目
      for (let k = 0; k < extra; k++) {
        const pid = otherProjects[(base + k) % otherProjects.length];
        if (!projectIds.includes(pid)) projectIds.push(pid);
      }
    }
    return { ...inst, projectIds };
  });
}
