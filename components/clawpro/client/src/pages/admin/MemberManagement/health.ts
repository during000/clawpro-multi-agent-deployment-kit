/**
 * 组织树 & 节点健康度（PRD v2.0）
 *
 * 组织健康度规则（§3.3.4.2）：
 *   节点 healthy ⇔ 对每个核心维度（模型 / 通道 / 安全组），
 *   以下之一成立：
 *     1) 该节点自身 scope 被某资源直接指向；
 *     2) 该节点任一祖先 scope 被某资源直接指向（继承）；
 *     3) 存在 scope.type === "all" 的该维度资源（全部用户档 + 平台默认）。
 *   以上任一未满足即 missing 对应项。
 *
 * 节点级独立判定：不因子孙状态影响自身（子节点红不让父节点红）。
 */
import type {
  InitHealth,
  NodeHealth,
  ResourceItem,
  Scope,
  UserGroup,
  UserOrg,
  ConfigCategory,
} from "./types";
import { MOCK_RESOURCES, getConfigEntries } from "./mock";

// ─── 组织树 ──────────────────────────────────────────────
export interface GroupTreeNode extends UserGroup {
  children: GroupTreeNode[];
  /** 从根到本节点的名称路径 */
  path: string;
  pathIds: string[];
  depth: number;
}

/** 扁平 parentId 列表 → 树（按同源分桶） */
export function buildGroupTree(groups: UserGroup[]): GroupTreeNode[] {
  const map = new Map<string, GroupTreeNode>();
  groups.forEach((g) =>
    map.set(g.id, {
      ...g,
      children: [],
      path: g.name,
      pathIds: [g.id],
      depth: 0,
    })
  );
  const roots: GroupTreeNode[] = [];
  groups.forEach((g) => {
    const node = map.get(g.id)!;
    if (g.parentId && map.has(g.parentId)) {
      const parent = map.get(g.parentId)!;
      parent.children.push(node);
      node.path = `${parent.path} / ${g.name}`;
      node.pathIds = [...parent.pathIds, g.id];
      node.depth = parent.depth + 1;
    } else {
      roots.push(node);
    }
  });
  return roots;
}

/**
 * 合并构树前的"组织扁平数据规范化"：
 *   - 若 `groups` 中包含 `source === "oneid-dept"`，把原本顶层的 `oneid-group` / `manual` 节点
 *     的 `parentId` 重映射到根 dept（默认 `dept-root`），让 A公司 成为唯一顶层。
 *   - 否则原样返回（保留引用，调用方零开销）。
 *
 * 用途：跨页面的"应用范围"选择器底层（如 ScopeEditPopover）只能消费扁平 { id, name, parentId }，
 *      无法直接用 buildUnifiedGroupTree；先用本函数规范化数据，再交给底层 buildTree 即可。
 */
export function normalizeGroupsForUnifiedTree<T extends { id: string; parentId?: string | null; source?: UserGroup["source"] }>(
  groups: T[],
  options?: { rootDeptId?: string }
): T[] {
  const rootDeptId = options?.rootDeptId ?? "dept-root";
  const hasDept = groups.some((g) => g.source === "oneid-dept");
  const hasRootDept = groups.some((g) => g.id === rootDeptId);
  if (!hasDept || !hasRootDept) return groups;

  return groups.map((g) => {
    if (
      (g.source === "oneid-group" || g.source === "manual") &&
      (g.parentId === null || g.parentId === undefined)
    ) {
      return { ...g, parentId: rootDeptId };
    }
    return g;
  });
}

/**
 * 合并构树（OneID 部门 + OneID 用户组 + 自建组织 一棵树）
 *
 * 自检测：
 *   - 若 `groups` 中存在 `source === "oneid-dept"`，视为"已同步部门"，把
 *     原本顶层的 `oneid-group` / `manual` 节点（parentId === null/undefined）
 *     重映射到根 dept（默认 `dept-root`）下，使 A公司 成为唯一顶层组织。
 *   - 否则退化为普通 `buildGroupTree`（行为不变，纯本地自建场景）。
 *
 * 关键：**只在构树时改写 parentId 的副本**，不会污染调用方传入的 `groups` 数组。
 *
 * 用途：跨页面的"应用范围 / 上级组织 / 目标组织"等选择器统一调用此函数，
 *      保证"同步数据源后 A公司 为唯一顶层"语义一致。
 */
export function buildUnifiedGroupTree(
  groups: UserGroup[],
  options?: { rootDeptId?: string }
): GroupTreeNode[] {
  const rootDeptId = options?.rootDeptId ?? "dept-root";
  const hasDept = groups.some((g) => g.source === "oneid-dept");
  const hasRootDept = groups.some((g) => g.id === rootDeptId);

  // 不满足合并条件 → 普通构树
  if (!hasDept || !hasRootDept) {
    return buildGroupTree(groups);
  }

  // 合并模式：把顶层 oneid-group / manual 节点的 parentId 改为 rootDeptId
  const normalized = normalizeGroupsForUnifiedTree(groups, { rootDeptId });

  const roots = buildGroupTree(normalized);

  // 排序根节点的直接子节点：oneid-dept → oneid-group → manual
  const sourceRank = (s: UserGroup["source"]) =>
    s === "oneid-dept" ? 0 : s === "oneid-group" ? 1 : 2;
  const sortChildren = (nodes: GroupTreeNode[]) => {
    nodes.forEach((n) => {
      n.children.sort((a, b) => sourceRank(a.source) - sourceRank(b.source));
      sortChildren(n.children);
    });
  };
  sortChildren(roots);

  return roots;
}

/** 在树中找节点 */
export function findGroupNode(
  roots: GroupTreeNode[],
  id: string
): GroupTreeNode | null {
  for (const r of roots) {
    if (r.id === id) return r;
    const hit = findGroupNode(r.children, id);
    if (hit) return hit;
  }
  return null;
}

/** 扁平化（前序遍历） */
export function flattenGroupTree(roots: GroupTreeNode[]): GroupTreeNode[] {
  const out: GroupTreeNode[] = [];
  const walk = (ns: GroupTreeNode[]) => {
    ns.forEach((n) => {
      out.push(n);
      if (n.children.length) walk(n.children);
    });
  };
  walk(roots);
  return out;
}

// ─── Scope 判定 ──────────────────────────────────────────
function scopeContainsGroup(scope: Scope, groupId: string): boolean {
  return scope.type === "filtered" && scope.groupIds.includes(groupId);
}

function scopeIsAll(scope: Scope): boolean {
  return scope.type === "all";
}

// ─── 获取节点的祖先 id 链（不含自身） ─────────────────────
function getAncestorIds(
  groupId: string,
  groups: UserGroup[]
): string[] {
  const map = new Map(groups.map((g) => [g.id, g]));
  const ids: string[] = [];
  let cur = map.get(groupId);
  while (cur && cur.parentId) {
    ids.push(cur.parentId);
    cur = map.get(cur.parentId);
  }
  return ids;
}

// ─── 节点级健康度（考虑继承 + 全部用户档 + 平台默认） ────
export function getGroupHealth(
  groupId: string,
  groups: UserGroup[],
  resources: ResourceItem[] = MOCK_RESOURCES
): NodeHealth {
  const ancestors = getAncestorIds(groupId, groups);
  const has = (kind: "model" | "channel" | "securityGroup") =>
    resources.some(
      (r) =>
        r.kind === kind &&
        (scopeIsAll(r.scope) ||
          scopeContainsGroup(r.scope, groupId) ||
          ancestors.some((a) => scopeContainsGroup(r.scope, a)))
    );

  const missing: NodeHealth["missing"] = [];
  if (!has("model")) missing.push("model");
  if (!has("channel")) missing.push("channel");
  if (!has("securityGroup")) missing.push("securityGroup");
  return { healthy: missing.length === 0, missing };
}

export const MISSING_LABEL: Record<
  NonNullable<NodeHealth["missing"][number]>,
  string
> = {
  model: "可见模型",
  channel: "可见通道",
  securityGroup: "安全组",
};

// ─── 用户查询 ────────────────────────────────────────────
/** 归属到某组织的直接用户 */
export function getUsersOfGroup(
  groupId: string,
  users: UserOrg[]
): UserOrg[] {
  return users.filter((u) => u.groupIds.includes(groupId));
}

/** 归属到某组织（含子孙）的用户（聚合，去重） */
export function getUsersOfGroupDeep(
  groupId: string,
  groups: UserGroup[],
  users: UserOrg[]
): UserOrg[] {
  const tree = buildGroupTree(groups);
  const node = findGroupNode(tree, groupId);
  if (!node) return [];
  const ids = new Set<string>();
  const walk = (n: GroupTreeNode) => {
    ids.add(n.id);
    n.children.forEach(walk);
  };
  walk(node);
  return users.filter((u) => u.groupIds.some((gid) => ids.has(gid)));
}

// ─── 节点下挂的资源（供「配置 Tab」使用） ─────────────────
export function getResourcesOfGroup(
  groupId: string,
  resources: ResourceItem[] = MOCK_RESOURCES
): ResourceItem[] {
  return resources.filter((r) => scopeContainsGroup(r.scope, groupId));
}

// ─── 全局存在多归属用户的判定（用于常驻 Alert） ───────────
/** 只计算"去重后 >= 2"的归属组数。
 *  注意：OneID 模式下即便只在一个部门也算 1，不会触发；
 *  但兼任部门、加入多个用户组等场景会触发。 */
export function countMultiGroupUsers(users: UserOrg[]): number {
  return users.filter((u) => u.groupIds.length >= 2).length;
}

// ─── 初始化健康度检查 ─────────────────────────────────────
/**
 * 初始化检查：模型 / 通道 / 镜像 / 网络（VPC+安全组）
 * 根据配置总览条目判断：该组织的配置总览中有无对应维度的配置。
 * 只要某个维度完全没有条目，则属于初始化未完成。
 */
export function getGroupInitHealth(
  groupId: string,
  groups: UserGroup[]
): InitHealth {
  const entries = getConfigEntries(groupId, groups);

  const missing: InitHealth["missing"] = [];

  // 模型：有 category === "model" 的条目即满足
  const hasModel = entries.some((e) => e.category === "model");
  if (!hasModel) missing.push("model");

  // 通道：有 category === "channel" 的条目即满足
  const hasChannel = entries.some((e) => e.category === "channel");
  if (!hasChannel) missing.push("channel");

  // 镜像：有 category === "image" 的条目即满足
  const hasImage = entries.some((e) => e.category === "image");
  if (!hasImage) missing.push("image");

  // 网络（VPC+安全组）：category === "network" 中需同时包含 VPC 和安全组
  const networkEntries = entries.filter((e) => e.category === "network");
  const hasVpc = networkEntries.some((e) => e.subLabel === "私有网络与子网");
  const hasSg = networkEntries.some((e) => e.subLabel === "安全组");
  if (!hasVpc || !hasSg) missing.push("network");

  return { initialized: missing.length === 0, missing };
}

/** 初始化缺失项的分类映射（用于 UI 显示） */
export const INIT_MISSING_LABEL: Record<
  InitHealth["missing"][number],
  string
> = {
  model: "模型",
  channel: "通道",
  image: "镜像",
  network: "网络",
};

/** 初始化缺失项到 ConfigCategory 的映射 */
export const INIT_MISSING_TO_CATEGORY: Record<
  InitHealth["missing"][number],
  ConfigCategory
> = {
  model: "model",
  channel: "channel",
  image: "image",
  network: "network",
};

// ─── 网络配置「待更新」判定 ──────────────────────────────
/**
 * 判定某组织当前生效的网络配置中是否存在影响实例创建的「待更新」状况。
 *
 * 触发条件（任一命中即视为存在待更新）：
 *   1) VPC 整个被删除：VPC ID 存在，但 vpcName / vpcCidr 缺失；
 *   2) 某个可用区下所有子网均被删除：meta.zonesAllDeleted 非空。
 *
 * 不触发的场景（用户管理页静默处理，不展示已删除子网，也不提示）：
 *   - 部分子网删除但该可用区仍有可用子网。
 *
 * 范围与边界：
 *   - 仅判定该组织当前条目（不冒泡到父组织、不下发到子组织、不影响兄弟组织）。
 *   - 来源为「预设策略」（presetPolicy）的 VPC 由平台自动重建，永不视为待更新。
 *   - 与 SecurityGroupManagement 中网络明细页保持同源判定。
 */
export function hasNetworkOutdated(
  groupId: string,
  groups: UserGroup[]
): boolean {
  const entries = getConfigEntries(groupId, groups);
  for (const entry of entries) {
    if (entry.category !== "network") continue;
    if (entry.subLabel !== "私有网络与子网" || !entry.meta) continue;
    // 预设策略由平台自动重建，永不进入"待更新"
    if (entry.source.type === "presetPolicy") continue;
    const meta = entry.meta as Record<string, unknown>;
    // ① VPC 整个被删除
    const vpcId = meta.vpcId ? String(meta.vpcId) : "";
    const vpcName = meta.vpcName ? String(meta.vpcName) : "";
    const vpcCidr = meta.vpcCidr ? String(meta.vpcCidr) : "";
    if (vpcId && (!vpcName || !vpcCidr)) return true;
    // ② 某个可用区下所有子网均被删除
    const zonesAllDeleted = Array.isArray(meta.zonesAllDeleted)
      ? (meta.zonesAllDeleted as string[])
      : [];
    if (zonesAllDeleted.length > 0) return true;
  }
  return false;
}
