/**
 * ShareScopeTree - 共享范围二级树形选择器
 *
 * 交互方式：搜索框聚焦 / 点击时通过 Popover 弹出二级下拉列表（Portal 渲染到 body），
 * 避免被 Dialog 的 overflow: hidden 裁剪。
 *
 * 第一层：分组（Checkbox 勾选整组 → 全组成员共享）
 * 第二层：分组下的成员（Checkbox 勾选个人 → 仅该人可共享）
 * 支持跨组勾选多个个人，支持搜索过滤。
 */
import { useState } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import { Popover, PopoverAnchor, PopoverContent } from "@/components/ui/popover";
import { ChevronDown, Search, Users, User, AlertCircle } from "lucide-react";

export interface ShareGroupNode {
  id: string;
  name: string;
  members: { id: string; name: string; email: string }[];
}

interface ShareScopeTreeProps {
  groups: ShareGroupNode[];
  /** 已勾选的分组 ID 列表 */
  selectedGroupIds: string[];
  /** 已勾选的用户 ID 列表 */
  selectedUserIds: string[];
  onSelectedGroupIdsChange: (ids: string[]) => void;
  onSelectedUserIdsChange: (ids: string[]) => void;
  /** 错误提示文案，传入时在搜索框下方显示红色提示 */
  error?: string;
  /** 受控的下拉展开状态（不传则内部自管理）。用于外部（如点击 +N 标签）触发展开。 */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

export function ShareScopeTree({
  groups,
  selectedGroupIds,
  selectedUserIds,
  onSelectedGroupIdsChange,
  onSelectedUserIdsChange,
  error,
  open: openProp,
  onOpenChange,
}: ShareScopeTreeProps) {
  const [searchKeyword, setSearchKeyword] = useState("");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set());
  const [openInternal, setOpenInternal] = useState(false);
  // 受控优先：外部传了 open 就用外部的，否则用内部 state
  const open = openProp ?? openInternal;
  const setOpen = (next: boolean) => {
    if (onOpenChange) onOpenChange(next);
    else setOpenInternal(next);
  };

  const keyword = searchKeyword.trim().toLowerCase();

  // 过滤分组：名称匹配 或 成员名/邮箱匹配
  const filteredGroups = keyword
    ? groups.filter(
        (g) =>
          g.name.toLowerCase().includes(keyword) ||
          g.members.some(
            (m) =>
              m.name.toLowerCase().includes(keyword) ||
              m.email.toLowerCase().includes(keyword)
          )
      )
    : groups;

  const toggleExpand = (groupId: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) next.delete(groupId);
      else next.add(groupId);
      return next;
    });
  };

  const isGroupChecked = (groupId: string) => isGroupFullyCovered(groupId);

  const isGroupIndeterminate = (groupId: string) => {
    if (isGroupFullyCovered(groupId)) return false;
    const group = groups.find((g) => g.id === groupId);
    if (!group) return false;
    const checkedMembers = group.members.filter((m) => selectedUserIds.includes(m.id));
    return checkedMembers.length > 0 && checkedMembers.length < group.members.length;
  };

  const handleGroupCheck = (groupId: string, checked: boolean) => {
    if (checked) {
      onSelectedGroupIdsChange([...selectedGroupIds, groupId]);
      const group = groups.find((g) => g.id === groupId);
      if (group) {
        onSelectedUserIdsChange(
          selectedUserIds.filter((uid) => !group.members.some((m) => m.id === uid))
        );
      }
    } else {
      onSelectedGroupIdsChange(selectedGroupIds.filter((id) => id !== groupId));
    }
  };

  const isMemberChecked = (memberId: string) => {
    const parentGroup = groups.find((g) => g.members.some((m) => m.id === memberId));
    if (parentGroup && selectedGroupIds.includes(parentGroup.id)) return true;
    return selectedUserIds.includes(memberId);
  };

  const isMemberDisabled = (memberId: string) => {
    const parentGroup = groups.find((g) => g.members.some((m) => m.id === memberId));
    return !!parentGroup && selectedGroupIds.includes(parentGroup.id);
  };

  const handleMemberCheck = (memberId: string, checked: boolean) => {
    if (checked) {
      onSelectedUserIdsChange([...selectedUserIds, memberId]);
    } else {
      onSelectedUserIdsChange(selectedUserIds.filter((id) => id !== memberId));
    }
  };

  // ===== 全选逻辑 =====
  // 全选 = 选中所有成员（通过 selectedUserIds），而不是选中分组。
  // 这样用户全选后仍可以取消勾选某个成员。
  const allMemberIds = groups.flatMap((g) => g.members.map((m) => m.id));
  const isAllSelected =
    allMemberIds.length > 0 &&
    allMemberIds.every((id) => selectedUserIds.includes(id));
  const isPartialSelected =
    !isAllSelected &&
    (selectedGroupIds.length > 0 || selectedUserIds.length > 0);

  // 判断某个组是否"完全被覆盖"：要么分组被勾选，要么该组所有成员都被勾选
  const isGroupFullyCovered = (groupId: string) => {
    if (selectedGroupIds.includes(groupId)) return true;
    const group = groups.find((g) => g.id === groupId);
    if (!group) return false;
    return group.members.every((m) => selectedUserIds.includes(m.id));
  };

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      // 全选：把所有成员加到 selectedUserIds，不使用 selectedGroupIds
      onSelectedGroupIdsChange([]);
      onSelectedUserIdsChange(allMemberIds);
    } else {
      // 取消全选
      onSelectedGroupIdsChange([]);
      onSelectedUserIdsChange([]);
    }
  };

  const getVisibleMembers = (group: ShareGroupNode) => {
    if (!keyword) return group.members;
    return group.members.filter(
      (m) =>
        m.name.toLowerCase().includes(keyword) ||
        m.email.toLowerCase().includes(keyword)
    );
  };

  // 搜索框 placeholder：展示已选数量
  const totalSelected = selectedGroupIds.length + selectedUserIds.length;
  const placeholder = totalSelected > 0
    ? `已选 ${totalSelected} 项，搜索分组或成员...`
    : "点击选择共享范围...";

  return (
    <Popover open={open} onOpenChange={(o) => { setOpen(o); if (!o) setSearchKeyword(""); }}>
      {/* 锚点：搜索框 */}
      <PopoverAnchor asChild>
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-muted)] pointer-events-none" />
          <Input
            tenant
            type="text"
            placeholder={placeholder}
            value={searchKeyword}
            onChange={(e) => {
              setSearchKeyword(e.target.value);
              if (!open) setOpen(true);
            }}
            onFocus={() => setOpen(true)}
            onClick={() => setOpen(true)}
            className="h-8 w-full pl-8 pr-8 text-sm"
          />
          <ChevronDown
            className={`absolute right-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-muted)] pointer-events-none transition-transform duration-200 ${open ? "rotate-180" : ""}`}
          />
        </div>
      </PopoverAnchor>

      {/* 错误提示 */}
      {error && (
        <p className="mt-1 text-xs text-red-500 flex items-center gap-1">
          <AlertCircle className="w-3 h-3 flex-shrink-0" />
          {error}
        </p>
      )}

      {/* 二级下拉列表 —— 通过 Portal 渲染到 body，不被 Dialog 裁剪 */}
      <PopoverContent
        align="start"
        side="bottom"
        sideOffset={4}
        // 不自动把焦点抢走，保持搜索框可继续输入
        onOpenAutoFocus={(e) => e.preventDefault()}
        // Radix Dialog 启用 react-remove-scroll，会拦截 Dialog 外部（Portal）元素的滚轮，
        // 导致此下拉无法用鼠标滚轮/触控板滚动。这里手动驱动容器滚动并阻止冒泡。
        onWheel={(e) => {
          e.currentTarget.scrollTop += e.deltaY;
          e.stopPropagation();
        }}
        className="w-[var(--radix-popover-trigger-width)] p-0 max-h-[280px] overflow-y-auto overscroll-contain"
      >
        {/* 全选行 —— 仅在无搜索关键字时显示 */}
        {!keyword && filteredGroups.length > 0 && (
          <div
            className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--border)] bg-[var(--muted)]/30 sticky top-0 z-10"
            onClick={(e) => e.stopPropagation()}
          >
            <Checkbox
              checked={
                isAllSelected
                  ? true
                  : isPartialSelected
                  ? "indeterminate"
                  : false
              }
              onCheckedChange={(val) => handleSelectAll(!!val)}
            />
            <span className="text-sm font-medium text-[var(--text-emphasis)]">
              全选
            </span>
            <span className="text-xs text-[var(--text-muted)] ml-auto">
              {selectedGroupIds.length + selectedUserIds.length}/{allMemberIds.length}
            </span>
          </div>
        )}

        {filteredGroups.length === 0 ? (
          <div className="py-6 text-center text-xs text-[var(--text-muted)]">
            无匹配结果
          </div>
        ) : (
          filteredGroups.map((group) => {
            const expanded = expandedGroups.has(group.id) || !!keyword;
            const visibleMembers = getVisibleMembers(group);
            const checkedCount = group.members.filter(
              (m) => isMemberChecked(m.id)
            ).length;

            return (
              <Collapsible
                key={group.id}
                open={expanded}
                onOpenChange={() => toggleExpand(group.id)}
              >
                {/* 第一层：分组 */}
                <div className="flex items-center border-b border-[var(--border)] last:border-b-0">
                  <CollapsibleTrigger className="group flex items-center gap-2 flex-1 px-3 py-2.5 hover:bg-[var(--accent)] transition-colors cursor-pointer text-left">
                    <ChevronDown
                      className={`w-3.5 h-3.5 text-[var(--text-weak)] transition-transform duration-200 flex-shrink-0 ${
                        expanded ? "" : "-rotate-90"
                      }`}
                    />
                    <Users className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
                    <span className="text-sm text-[var(--text-emphasis)] truncate">
                      {group.name}
                    </span>
                    {!isGroupChecked(group.id) && checkedCount > 0 && (
                      <span className="text-xs text-[var(--text-muted)] ml-auto flex-shrink-0">
                        {checkedCount}/{group.members.length}
                      </span>
                    )}
                  </CollapsibleTrigger>
                  <div className="pr-3 flex-shrink-0" onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={
                        isGroupChecked(group.id)
                          ? true
                          : isGroupIndeterminate(group.id)
                          ? "indeterminate"
                          : false
                      }
                      onCheckedChange={(val) =>
                        handleGroupCheck(group.id, !!val)
                      }
                    />
                  </div>
                </div>

                {/* 第二层：成员 */}
                <CollapsibleContent>
                  <div className="bg-[var(--muted)]/50">
                    {visibleMembers.map((member) => (
                      <div
                        key={member.id}
                        className={`flex items-center gap-2 pl-10 pr-3 py-2 hover:bg-[var(--accent)] transition-colors ${
                          isMemberDisabled(member.id) ? "opacity-50" : ""
                        }`}
                      >
                        <Checkbox
                          checked={isMemberChecked(member.id)}
                          disabled={isMemberDisabled(member.id)}
                          onCheckedChange={(val) =>
                            handleMemberCheck(member.id, !!val)
                          }
                        />
                        <User className="w-3 h-3 text-[var(--text-muted)] flex-shrink-0" />
                        <span className="text-xs text-[var(--text-emphasis)] truncate">
                          {member.name}
                        </span>
                        <span className="text-xs text-[var(--text-muted)] ml-auto flex-shrink-0 truncate">
                          {member.email}
                        </span>
                      </div>
                    ))}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            );
          })
        )}
      </PopoverContent>
    </Popover>
  );
}
