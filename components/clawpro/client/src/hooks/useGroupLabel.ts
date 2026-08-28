/**
 * Hook：返回分组相关的 UI 文案（设计项目简化版，直接返回默认「分组」文案）。
 */
export function useGroupLabel() {
  return {
    group: "分组",
    groups: "分组",
    create: "创建分组",
    search: "搜索分组",
    select: "选择分组",
    selectPlaceholder: "选择分组",
    empty: "暂无分组",
    manage: "管理分组",
    all: "全部分组",
  };
}
