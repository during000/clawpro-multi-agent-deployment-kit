/**
 * ScopeUserOrGroupPopover - 应用范围编辑（按分组 / 按用户 二选一）
 *
 * 设计参考：client/src/components/_internal/ScopeEditPopover.tsx（通道配置应用范围编辑）。
 *
 * 差异：
 *   - ScopeEditPopover 的 Segment 是「全部用户 / 按分组」二选一
 *   - 本组件的 Segment 是「按分组 / 按用户」二选一，互斥
 *   - 「按分组」详情完全复用 ScopeEditPopover 的树形 Checkbox 列表
 *   - 「按用户」新增用户列表勾选（带头像 + 邮箱 + 部门）
 *
 * 视觉骨架：Popover 触发器（徽章 + 铅笔）→ 弹出面板（Segment + 内容 + 底部确认）
 *
 * Token / 圆角 / 按钮 variant / 阴影完全对齐 ScopeEditPopover，确保跨页面视觉一致。
 */
import { useState, useMemo, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Popover, PopoverContent, PopoverTrigger, PopoverAnchor } from "@/components/ui/popover";
import { CompactText, MetaText } from "@/components/ui/Typography";
import { Pencil, X, ChevronRight, ChevronDown, UserCircle } from "lucide-react";

// ─── 类型定义 ────────────────────────────────────────────────
export interface ScopeUserGroup {
  id: string;
  name: string;
  parentId?: string | null;
}

export interface ScopeUserItem {
  id: string;
  name: string;
  email?: string;
  department?: string;
}

export type ScopeUserOrGroupMode = "groups" | "users";

export interface ScopeUserOrGroupPopoverProps {
  /** 当前模式（按分组 / 按用户） */
  mode: ScopeUserOrGroupMode;
  /** 当前选中的分组 id 列表 */
  selectedGroupIds: string[];
  /** 当前选中的用户 id 列表 */
  selectedUserIds: string[];
  /** 所有可选分组 */
  groups: ScopeUserGroup[];
  /** 所有可选用户 */
  users: ScopeUserItem[];
  /**
   * 保存回调（互斥语义：返回的两个数组中至多一个非空）
   * - mode="groups" → groupIds 可能非空，userIds 必为空
   * - mode="users"  → userIds 可能非空，groupIds 必为空
   */
  onConfirm: (mode: ScopeUserOrGroupMode, groupIds: string[], userIds: string[]) => void;
  /** 触发器自定义渲染（默认为铅笔图标按钮） */
  trigger?: React.ReactNode;
  /** Popover 对齐方式 */
  align?: "start" | "center" | "end";
  /** 是否显示范围徽章 */
  showBadges?: boolean;
  /** 最多展示几个 tag，超出折叠为 +N（默认 5） */
  maxVisibleBadges?: number;
  /** 空态文案（既未选用户也未选分组时） */
  emptyLabel?: string;
  /** 受控：外部 open 状态（不传则走内部 state，铅笔触发器自管理） */
  open?: boolean;
  /** 受控：open 变化回调（与 open 配合使用） */
  onOpenChange?: (open: boolean) => void;
  /** 是否隐藏内置铅笔触发器（业务通过外部按钮控制 open 时使用） */
  hideTrigger?: boolean;
}

// ─── 树结构工具（与 ScopeEditPopover 一致） ────────────────
interface TreeNode {
  id: string;
  name: string;
  children: TreeNode[];
}

function buildTree(groups: ScopeUserGroup[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  groups.forEach((g) => map.set(g.id, { id: g.id, name: g.name, children: [] }));
  const roots: TreeNode[] = [];
  groups.forEach((g) => {
    const node = map.get(g.id)!;
    if (g.parentId && map.has(g.parentId)) {
      map.get(g.parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

export function ScopeUserOrGroupPopover({
  mode,
  selectedGroupIds,
  selectedUserIds,
  groups,
  users,
  onConfirm,
  trigger,
  align = "start",
  showBadges = true,
  maxVisibleBadges = 5,
  emptyLabel = "未分配",
  open: controlledOpen,
  onOpenChange: controlledOnOpenChange,
  hideTrigger = false,
}: ScopeUserOrGroupPopoverProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const isControlled = controlledOpen !== undefined;
  const open = isControlled ? controlledOpen : internalOpen;
  const setOpen = (v: boolean) => {
    if (!isControlled) setInternalOpen(v);
    controlledOnOpenChange?.(v);
  };
  const [draftMode, setDraftMode] = useState<ScopeUserOrGroupMode>(mode);
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>([]);
  const [draftUserIds, setDraftUserIds] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [userSearchQuery, setUserSearchQuery] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const inputRef = useRef<HTMLInputElement>(null);

  // 树结构
  const tree = useMemo(() => buildTree(groups), [groups]);

  // 打开时同步状态
  const handleOpenChange = (v: boolean) => {
    if (v) {
      setDraftMode(mode);
      setDraftGroupIds([...selectedGroupIds]);
      setDraftUserIds([...selectedUserIds]);
      setSearchQuery("");
      setUserSearchQuery("");
      // 默认展开已选分组的祖先 + 根节点
      const expandSet = new Set<string>();
      const groupMap = new Map(groups.map((g) => [g.id, g]));
      selectedGroupIds.forEach((gid) => {
        let cur = groupMap.get(gid);
        while (cur && cur.parentId) {
          expandSet.add(cur.parentId);
          cur = groupMap.get(cur.parentId);
        }
      });
      tree.forEach((root) => expandSet.add(root.id));
      setExpanded(expandSet);
    }
    setOpen(v);
  };

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // 搜索过滤 - 分组
  const matchedGroupIds = useMemo(() => {
    if (!searchQuery.trim()) return null;
    const q = searchQuery.toLowerCase();
    return new Set(groups.filter((g) => g.name.toLowerCase().includes(q)).map((g) => g.id));
  }, [groups, searchQuery]);

  const isNodeVisible = (node: TreeNode): boolean => {
    if (!matchedGroupIds) return true;
    if (matchedGroupIds.has(node.id)) return true;
    return node.children.some(isNodeVisible);
  };

  // 搜索过滤 - 用户
  const filteredUsers = useMemo(() => {
    const q = userSearchQuery.trim().toLowerCase();
    if (!q) return users;
    return users.filter(
      (u) =>
        u.name.toLowerCase().includes(q) ||
        (u.email || "").toLowerCase().includes(q) ||
        (u.department || "").toLowerCase().includes(q)
    );
  }, [users, userSearchQuery]);

  // 切换分组选择
  const toggleGroup = (gid: string) => {
    setDraftGroupIds((prev) =>
      prev.includes(gid) ? prev.filter((id) => id !== gid) : [...prev, gid]
    );
  };

  // 切换用户选择
  const toggleUser = (uid: string) => {
    setDraftUserIds((prev) =>
      prev.includes(uid) ? prev.filter((id) => id !== uid) : [...prev, uid]
    );
  };

  // 清除选择
  const handleClearGroupSelection = () => {
    setDraftGroupIds([]);
    setSearchQuery("");
  };

  // 切换 Segment（互斥：清空对方草稿，避免提交时混合）
  const handleSwitchMode = (next: ScopeUserOrGroupMode) => {
    if (next === draftMode) return;
    setDraftMode(next);
    if (next === "groups") setDraftUserIds([]);
    else setDraftGroupIds([]);
  };

  // 确认按钮禁用条件：当前模式下未选任何项
  const isConfirmDisabled =
    (draftMode === "groups" && draftGroupIds.length === 0) ||
    (draftMode === "users" && draftUserIds.length === 0);

  const handleConfirm = () => {
    if (isConfirmDisabled) return;
    if (draftMode === "groups") {
      onConfirm("groups", draftGroupIds, []);
    } else {
      onConfirm("users", [], draftUserIds);
    }
    setOpen(false);
  };

  // 判断是否为纯平铺列表
  const isFlat = useMemo(() => tree.every((node) => node.children.length === 0), [tree]);

  // 渲染分组树节点
  const renderTreeNode = (node: TreeNode, depth: number): React.ReactNode => {
    if (!isNodeVisible(node)) return null;
    const checked = draftGroupIds.includes(node.id);
    const hasChildren = node.children.length > 0;
    const isExpanded = expanded.has(node.id);
    return (
      <div key={node.id} className="mt-0.5 first:mt-0">
        <div
          className={`w-full flex items-center gap-1.5 h-8 rounded-[6px] transition-colors cursor-pointer ${
            checked ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "hover:bg-[var(--bg-grey-hover)]"
          } ${isFlat ? "px-3" : ""}`}
          style={isFlat ? undefined : { paddingLeft: 12 + depth * 16 }}
          onClick={() => toggleGroup(node.id)}
        >
          {!isFlat && (
            hasChildren ? (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpand(node.id);
                }}
                className="w-4 h-4 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-secondary)] shrink-0"
              >
                {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
              </button>
            ) : (
              <span className="w-4 h-4 shrink-0" />
            )
          )}
          <Checkbox
            checked={checked}
            onCheckedChange={() => toggleGroup(node.id)}
            onClick={(e) => e.stopPropagation()}
          />
          <CompactText tone="emphasis" className="truncate">{node.name}</CompactText>
        </div>
        {hasChildren && isExpanded && node.children.map((c) => renderTreeNode(c, depth + 1))}
      </div>
    );
  };

  // 渲染范围徽章
  const renderBadges = () => {
    if (!showBadges) return null;

    const isUsers = mode === "users";
    // 以 id 为键建映射，避免在 map 内重复 find（O(n*m) -> O(1)）
    const groupNameMap = new Map(groups.map((g) => [g.id, g.name] as const));
    const userMap = new Map(users.map((u) => [u.id, u] as const));
    const labels = isUsers
      ? selectedUserIds.map((uid) => {
          const u = userMap.get(uid);
          return u ? (u.email ? `${u.name} (${u.email})` : u.name) : uid;
        })
      : selectedGroupIds.map((gid) => groupNameMap.get(gid) || gid);

    if (labels.length === 0) return <Badge variant="outline">{emptyLabel}</Badge>;

    const visibleCount = Math.max(1, maxVisibleBadges);
    const visibleLabels = labels.slice(0, visibleCount);
    const rest = labels.length - visibleLabels.length;

    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex items-center gap-1 cursor-default flex-wrap">
            {visibleLabels.map((label, idx) => (
              <Badge key={`${label}-${idx}`} variant="secondary" className={isUsers ? "max-w-[160px]" : "max-w-[140px]"}>
                {isUsers && <UserCircle className="w-3 h-3 mr-0.5 shrink-0" />}
                <span className="block truncate max-w-[124px]">{label}</span>
              </Badge>
            ))}
            {rest > 0 && <Badge variant="secondary">+{rest}</Badge>}
          </span>
        </TooltipTrigger>
        <TooltipContent side="top" className={isUsers ? "max-w-[320px] leading-relaxed" : "max-w-[280px] leading-relaxed"}>
          <MetaText tone="inherit">{labels.join("，")}</MetaText>
        </TooltipContent>
      </Tooltip>
    );
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverAnchor asChild>
        <div className="inline-flex items-center gap-1.5">
          {renderBadges()}
          {!hideTrigger && (
            <PopoverTrigger asChild>
              {trigger || (
                <button
                  onClick={(e) => e.stopPropagation()}
                  className="self-center text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
                  title="编辑应用范围"
                >
                  <Pencil className="w-3 h-3" />
                </button>
              )}
            </PopoverTrigger>
          )}
        </div>
      </PopoverAnchor>
      <PopoverContent
        className="w-80 p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)] flex flex-col"
        align={align}
        sideOffset={6}
        onClick={(e) => e.stopPropagation()}
      >
          {/* 内容区 */}
          <div className="p-2 space-y-2">
            {/* Segment 切换：按分组 / 按用户 */}
            <SegmentGroup className="w-full">
              <SegmentOption
                active={draftMode === "groups"}
                onClick={() => handleSwitchMode("groups")}
                className="flex-1"
              >
                按分组
              </SegmentOption>
              <SegmentOption
                active={draftMode === "users"}
                onClick={() => handleSwitchMode("users")}
                className="flex-1"
              >
                按用户
              </SegmentOption>
            </SegmentGroup>

            {/* 按分组内容（与 ScopeEditPopover 保持一致） */}
            {draftMode === "groups" && (
              <div className="space-y-2.5">
                {/* 输入框：已选标签 + 搜索输入 */}
                <div
                  className="flex flex-wrap items-center gap-1 px-2.5 py-1.5 min-h-[36px] max-h-[80px] overflow-y-auto border border-border rounded-[4px] bg-white focus-within:border-ring transition-colors cursor-text"
                  onClick={() => inputRef.current?.focus()}
                >
                  {draftGroupIds.map((gid) => {
                    const g = groups.find((gr) => gr.id === gid);
                    if (!g) return null;
                    return (
                      <span key={gid} className="inline-flex items-center gap-1 h-5 rounded-full bg-[#F5F5F5] px-2 text-xs text-[var(--text-title)] leading-none">
                        <span className="truncate max-w-[100px] leading-none">{g.name}</span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleGroup(gid);
                          }}
                          className="inline-flex items-center justify-center text-[var(--text-muted)] hover:text-[var(--text-emphasis)] shrink-0 h-3 w-3"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    );
                  })}
                  <input
                    ref={inputRef}
                    type="text"
                    placeholder={draftGroupIds.length === 0 ? "搜索分组…" : ""}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="flex-1 min-w-[60px] h-5 text-sm bg-transparent outline-none placeholder:text-[var(--text-weak)]"
                  />
                  {(draftGroupIds.length > 0 || searchQuery) && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleClearGroupSelection();
                      }}
                      className="text-[var(--text-weak)] hover:text-[var(--text-secondary)] shrink-0 ml-auto"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>

                {/* 分组树形列表 */}
                <div className="max-h-[220px] overflow-y-auto pb-2" onWheel={(e) => e.stopPropagation()}>
                  {tree.length === 0 ? (
                    <MetaText as="p" tone="weak" className="text-center py-4">暂无分组</MetaText>
                  ) : (
                    (() => {
                      const visibleNodes = tree.filter(isNodeVisible);
                      if (visibleNodes.length === 0) {
                        return <MetaText as="p" tone="weak" className="text-center py-4">无匹配分组</MetaText>;
                      }
                      return visibleNodes.map((node) => renderTreeNode(node, 0));
                    })()
                  )}
                </div>
              </div>
            )}

            {/* 按用户内容（用户列表勾选） */}
            {draftMode === "users" && (
              <div className="space-y-2.5">
                {/* 搜索框 */}
                <div className="px-2.5 py-1.5 min-h-[36px] border border-border rounded-[4px] bg-white focus-within:border-ring transition-colors">
                  <input
                    type="text"
                    placeholder="搜索用户姓名 / 邮箱 / 部门…"
                    value={userSearchQuery}
                    onChange={(e) => setUserSearchQuery(e.target.value)}
                    className="w-full h-5 text-sm bg-transparent outline-none placeholder:text-[var(--text-weak)]"
                  />
                </div>

                {/* 用户列表 */}
                <div className="max-h-[260px] overflow-y-auto pb-2" onWheel={(e) => e.stopPropagation()}>
                  {filteredUsers.length === 0 ? (
                    <MetaText as="p" tone="weak" className="text-center py-4">
                      {userSearchQuery ? "无匹配用户" : "暂无用户"}
                    </MetaText>
                  ) : (
                    filteredUsers.map((u) => {
                      const checked = draftUserIds.includes(u.id);
                      return (
                        <div
                          key={u.id}
                          className={`mt-0.5 first:mt-0 flex items-center gap-2 h-10 px-2 rounded-[6px] transition-colors cursor-pointer ${
                            checked ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "hover:bg-[var(--bg-grey-hover)]"
                          }`}
                          onClick={() => toggleUser(u.id)}
                        >
                          <Checkbox
                            checked={checked}
                            onCheckedChange={() => toggleUser(u.id)}
                            onClick={(e) => e.stopPropagation()}
                          />
                          <div className="flex-1 min-w-0 flex flex-col">
                            <CompactText tone="emphasis" className="truncate">{u.name}</CompactText>
                            {(u.email || u.department) && (
                              <MetaText tone="weak" className="truncate text-[11px] leading-tight">
                                {u.email}
                                {u.email && u.department ? " · " : ""}
                                {u.department}
                              </MetaText>
                            )}
                          </div>
                        </div>
                      );
                    })
                  )}
                </div>
              </div>
            )}
          </div>

          {/* 底部：已选计数 + 按钮 */}
          <div className="shrink-0 mx-2 border-t border-[#EAEEF4] py-2 flex items-center">
            <MetaText className="flex-1">
              {draftMode === "groups" && draftGroupIds.length > 0
                ? `已选 ${draftGroupIds.length} 个分组`
                : draftMode === "users" && draftUserIds.length > 0
                ? `已选 ${draftUserIds.length} 位用户`
                : ""}
            </MetaText>
            <div className="flex items-center gap-2">
              <Button
                variant="claw-outline"
                size="sm"
                className="text-xs h-7 px-2"
                onClick={() => setOpen(false)}
              >
                取消
              </Button>
              <Button
                variant="dialog-confirm"
                size="sm"
                className="text-xs h-7 px-3"
                disabled={isConfirmDisabled}
                onClick={handleConfirm}
              >
                确认
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
  );
}

export default ScopeUserOrGroupPopover;
