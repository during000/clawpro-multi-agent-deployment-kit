/**
 * TreeSelect - 树形单选下拉组件（统一封装）
 *
 * 数据结构：TreeNode[]（{ id, name, children?, path? }）
 *
 * 通过 triggerVariant 区分两种触发器：
 *   - "button"：toolbar 按钮触发（承载原 TreeSelectFilter）
 *   - "filter-icon"：表头漏斗图标触发（承载原 TableHeaderTreeFilter）
 *
 * 设计：内部委托给 TreeSelectFilter / TableHeaderTreeFilter 两个旧实现，
 * 待业务全部迁移完成后，再把实现内联进本文件并删除旧文件。
 */
import * as React from "react";
import {
  TreeSelectFilter,
  type TreeNode as _TreeNode,
  type TreeSelectSection as _TreeSelectSection,
} from "@/components/_internal/TreeSelectFilter";
import { TableHeaderTreeFilter, type TreeFilterNode } from "@/components/_internal/TableHeaderTreeFilter";

// 向后兼容旧类型导出
export type { TreeFilterNode };
export type { TreeNode } from "@/components/_internal/TreeSelectFilter";
export type { TreeSelectSection } from "@/components/_internal/TreeSelectFilter";

// ─── 类型 ────────────────────────────────────────────────────────────────────

/** 通用树节点（兼容 TreeSelectFilter 的 path 扩展字段） */
export interface TreeSelectNode {
  id: string;
  name: string;
  children?: TreeSelectNode[];
  /** 节点路径（仅 button 变体用于底部面包屑） */
  path?: string;
}

export type TreeSelectTriggerVariant = "button" | "filter-icon";

/** 分区（部门 / 自定义分组等多分区单选场景） */
export interface TreeSelectSectionData {
  key: string;
  label: string;
  roots: TreeSelectNode[];
}

interface TreeSelectCommonProps {
  /** 树形数据（与 sections 二选一） */
  nodes: TreeSelectNode[];
  /** 当前选中节点 id（""=全部） */
  value: string;
  /**
   * 选中变化回调。
   * - button 变体：等价于旧 TreeSelectFilter 的 onChange
   * - filter-icon 变体：等价于旧 TableHeaderTreeFilter 的 onConfirm
   */
  onChange?: (value: string) => void;
  /** @deprecated filter-icon 变体下旧 prop，保留向后兼容；推荐用 onChange */
  onConfirm?: (value: string) => void;
  /** "全部"选项文案 */
  allLabel?: string;
  /** 搜索框 placeholder */
  searchPlaceholder?: string;
  /** 面板宽度（默认 280） */
  panelWidth?: number;
  /** 面板对齐方式 */
  align?: "start" | "center" | "end";
}

interface TreeSelectButtonProps extends TreeSelectCommonProps {
  triggerVariant?: "button";
  /** 触发器宽度（仅 button 变体） */
  triggerWidth?: number;
  /** 分区数据（多分区单选；优先级高于 nodes，仅 button 变体支持） */
  sections?: TreeSelectSectionData[];
  /** 是否显示搜索框（默认 true，仅 button 变体支持） */
  showSearch?: boolean;
}

interface TreeSelectFilterIconProps extends TreeSelectCommonProps {
  triggerVariant: "filter-icon";
  /** 列标题（仅 filter-icon 变体必填） */
  title: string;
  /**
   * 提交模式（仅 filter-icon 变体支持，默认 "confirm"）：
   * - "confirm"：底部带"取消/确认"按钮，点确认才生效
   * - "instant"：点选项立即生效并关闭面板（建议同时关闭搜索/footer）
   */
  commitMode?: "confirm" | "instant";
  /** 是否显示搜索框（仅 filter-icon 变体支持，默认 true） */
  showSearch?: boolean;
  /** 是否显示底部 footer（仅 filter-icon 变体支持，默认 true） */
  showFooter?: boolean;
}

export type TreeSelectProps = TreeSelectButtonProps | TreeSelectFilterIconProps;

// ─── 实现 ────────────────────────────────────────────────────────────────────

export function TreeSelect(props: TreeSelectProps) {
  const handler = props.onChange ?? props.onConfirm ?? (() => {});

  if (props.triggerVariant === "filter-icon") {
    const { nodes, value, title, allLabel, searchPlaceholder, panelWidth, align, commitMode, showSearch, showFooter } = props;
    return (
      <TableHeaderTreeFilter
        title={title}
        nodes={nodes as TreeFilterNode[]}
        value={value}
        onConfirm={handler}
        allLabel={allLabel}
        searchPlaceholder={searchPlaceholder}
        panelWidth={panelWidth}
        align={align}
        commitMode={commitMode}
        showSearch={showSearch}
        showFooter={showFooter}
      />
    );
  }

  // button 变体（默认）
  const { nodes, value, allLabel, searchPlaceholder, panelWidth, align, triggerWidth, sections, showSearch } = props;
  return (
    <TreeSelectFilter
      nodes={nodes as _TreeNode[]}
      sections={sections as _TreeSelectSection[] | undefined}
      value={value}
      onChange={handler}
      allLabel={allLabel}
      showSearch={showSearch}
      searchPlaceholder={searchPlaceholder}
      triggerWidth={triggerWidth}
      panelWidth={panelWidth}
      align={align}
    />
  );
}

export default TreeSelect;
