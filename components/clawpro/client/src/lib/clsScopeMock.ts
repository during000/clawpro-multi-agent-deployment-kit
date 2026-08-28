/**
 * clsScopeMock - 会话管理 / 运维观测 / Tokens 监控（按会话）
 * 三个页面共享的「组织 / 开启范围」mock 数据。
 *
 * 数据复刻自参考项目 (openclaw-enterprise) 中的：
 *   - lib/mockData.ts MOCK_DEPARTMENTS（部门树）
 *   - pages/admin/MemberManagement/mock.ts MOCK_MANUAL_GROUPS（自定义组织扁平表）
 *
 * 转换为 components/groupTreeShared 所需的 FilterSection / TreeNodeData 形态。
 *
 * 模式语义（与参考项目一致）：
 *   - OneID（标准模式，hasOneid=true）   → 展示「部门 + 自定义组织」两个分区
 *   - 普通（自定义模式，hasOneid=false） → 仅展示「自定义组织」分区
 */
import { useMemo } from "react";
import type { FilterSection, TreeNodeData } from "@/components/groupTreeShared";
import { useAdminMode } from "@/contexts/AdminModeContext";

// ─── 部门树（OneID 模式数据，复刻参考项目） ────────────────────
interface DepartmentNode {
  id: string;
  name: string;
  children?: DepartmentNode[];
}

const MOCK_DEPARTMENTS: DepartmentNode[] = [
  {
    id: "dept-root",
    name: "A公司",
    children: [
      {
        id: "dept-tech",
        name: "技术部",
        children: [
          { id: "dept-fe", name: "前端组" },
          { id: "dept-be", name: "后端组" },
          { id: "dept-ai", name: "AI 团队" },
        ],
      },
      {
        id: "dept-product",
        name: "产品部",
        children: [
          { id: "dept-pm", name: "产品经理组" },
          { id: "dept-design", name: "设计组" },
        ],
      },
      { id: "dept-ops", name: "运营部" },
      { id: "dept-hr", name: "人力资源部" },
    ],
  },
];

// ─── 自定义组织（扁平 + parentId，复刻参考项目 MOCK_MANUAL_GROUPS） ─
interface ManualGroup {
  id: string;
  name: string;
  parentId: string | null;
}

const MOCK_MANUAL_GROUPS: ManualGroup[] = [
  { id: "mgrp-product", name: "产品组", parentId: null },
  { id: "mgrp-rd", name: "研发组", parentId: null },
  { id: "mgrp-rd-fe", name: "研发-前端", parentId: "mgrp-rd" },
  { id: "mgrp-rd-be", name: "研发-后端", parentId: "mgrp-rd" },
  { id: "mgrp-design", name: "设计组", parentId: null },
  { id: "mgrp-ops", name: "产品运营与市场推广团队", parentId: null },
];

// ─── 转换辅助函数 ─────────────────────────────────────────────
function deptToTreeNode(d: DepartmentNode): TreeNodeData {
  return { id: d.id, name: d.name, children: d.children?.map(deptToTreeNode) };
}

function userGroupsToForest(groups: ManualGroup[]): TreeNodeData[] {
  const byId = new Map<string, TreeNodeData>();
  groups.forEach((g) => byId.set(g.id, { id: g.id, name: g.name, children: [] }));
  const roots: TreeNodeData[] = [];
  groups.forEach((g) => {
    const node = byId.get(g.id)!;
    if (g.parentId && byId.has(g.parentId)) {
      byId.get(g.parentId)!.children!.push(node);
    } else {
      roots.push(node);
    }
  });
  byId.forEach((n) => {
    if (n.children && n.children.length === 0) delete n.children;
  });
  return roots;
}

// ─── 分区常量（文件级，避免每次渲染重建） ─────────────────────
export const DEPT_SECTION: FilterSection = {
  key: "dept",
  label: "部门",
  roots: MOCK_DEPARTMENTS.map(deptToTreeNode),
};

export const CUSTOM_SECTION: FilterSection = {
  key: "custom",
  label: "自定义组织",
  roots: userGroupsToForest(MOCK_MANUAL_GROUPS),
};

/**
 * 根据当前管控端模式动态返回分区列表：
 *   - OneID（标准模式） → 单分区「组织架构」：A公司 为唯一顶层，自定义组织作为 A公司 二级。
 *   - 普通模式           → [自定义组织]
 *
 * OneID 模式下不再按"部门 / 自定义组织"双分区呈现，与"管控端 > 用户管理 > 同步数据源后
 * A公司 为唯一顶层组织"的全局语义保持一致。
 */
export function useFilterSections(): FilterSection[] {
  const { hasOneid } = useAdminMode();
  return useMemo<FilterSection[]>(() => {
    if (!hasOneid) return [CUSTOM_SECTION];

    // OneID 模式：把自定义组织作为 A公司 的子节点合并进部门树
    const merged: TreeNodeData[] = MOCK_DEPARTMENTS.map(deptToTreeNode);
    const customRoots = userGroupsToForest(MOCK_MANUAL_GROUPS);
    if (customRoots.length > 0 && merged.length > 0) {
      const company = merged[0]; // dept-root
      company.children = [...(company.children ?? []), ...customRoots];
    }
    return [{ key: "unified", label: "组织架构", roots: merged }];
  }, [hasOneid]);
}

