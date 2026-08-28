/**
 * GroupSingleFilter - 分组单选筛选器
 *
 * 设计参考：
 *   - 外观复刻 Agent 列表页的 InstanceDepartmentFilter（120px 宽 Popover）
 *   - 支持部门 + 自定义分组两分区（沿用 FilterSection 数据结构）
 *   - 单选交互：点击节点选中；点击"全部分组"清空
 *   - 底部显示面包屑路径；选中子节点时高亮
 *
 * 实现：已收敛到统一的 TreeSelectFilter（支持 sections 分区 + 单选 + 确认），
 *   本组件仅做「FilterSection → TreeSelectSection」的数据适配，保持对外 API 不变。
 */
import { useMemo } from "react";
import { TreeSelectFilter, type TreeSelectSection } from "@/components/_internal/TreeSelectFilter";
import { type FilterSection, type TreeNodeData, findNode, collectDescendantIds } from "./groupTreeShared";

export interface GroupSingleFilterProps {
  /** 分区数据（部门 + 自定义分组） */
  sections: FilterSection[];
  /** 当前选中的分组 id；空串表示"全部分组" */
  value: string;
  /** 变更回调 */
  onChange: (value: string) => void;
  /** 触发器宽度（默认 120，对齐 Agent 列表） */
  triggerWidth?: number;
  /** 未选时的 placeholder */
  placeholder?: string;
}

export function GroupSingleFilter({
  sections,
  value,
  onChange,
  triggerWidth = 120,
  placeholder = "全部分组",
}: GroupSingleFilterProps) {
  // FilterSection 与 TreeSelectSection 结构兼容（key / label / roots），直接透传
  const treeSections = useMemo<TreeSelectSection[]>(
    () =>
      sections.map((s) => ({
        key: s.key,
        label: s.label,
        roots: s.roots,
      })),
    [sections],
  );

  return (
    <TreeSelectFilter
      sections={treeSections}
      value={value}
      onChange={onChange}
      allLabel={placeholder}
      showSearch={false}
      triggerWidth={triggerWidth}
    />
  );
}

// ─── 帮助函数：把单选 id 解析成"视作命中"的 id 集合（含子孙） ───
/**
 * 将单选 id 转成"包含自身 + 所有子孙"的 id 集合，供下游筛选。
 * 空字符串返回空集，下游判空即视作"全部"。
 */
export function getSingleGroupFilterIds(sections: FilterSection[], value: string): Set<string> {
  if (!value) return new Set();
  const node: TreeNodeData | undefined = findNode(sections, value);
  if (!node) return new Set();
  return new Set(collectDescendantIds(node));
}
