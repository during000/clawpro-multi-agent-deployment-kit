/**
 * ScopeFilterDropdown - 筛选下拉面板（多选，不需要确认）
 *
 * 通用的应用范围多选筛选下拉面板组件。
 * 特性：
 *   - 搜索框（Input 组件 + Search 图标）
 *   - 全选/全不选切换
 *   - 组织展示：「全部用户」+ 「按组织」
 *   - Checkbox 组件选中态 + bg-[var(--bg-brand-selected)] 高亮
 *   - 底部已选计数 + 蓝色文字清除按钮
 *   - 选中即时生效，无需确认按钮
 *
 * 视觉规范：
 *   - 容器：rounded-[4px]，标准阴影，pt-2 px-2 pb-0
 *   - 选项行：h-8 px-3 rounded-[6px]，选中 bg-[var(--bg-brand-selected)]，未选中 hover:bg-[var(--bg-grey-hover)]
 *   - 子节点间距：space-y-0.5（2px）
 *   - Footer：border-t + h-9 垂直居中
 */
import { useState, useMemo } from "react";
import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { BodyText, MetaText, MetaMedium } from "@/components/ui/Typography";

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export interface ScopeFilterGroup {
  id: string;
  name: string;
}

export interface ScopeFilterDropdownProps {
  /** 所有组织选项 */
  groups: ScopeFilterGroup[];
  /** 当前选中的 key 集合（包含 'public' 代表全部用户，以及组织 id） */
  selectedKeys: Set<string>;
  /** 选中变化回调 */
  onChange: (keys: Set<string>) => void;
  /** 搜索框 placeholder（默认 "搜索..."） */
  searchPlaceholder?: string;
  /** "全部用户" 的 key（默认 'public'） */
  publicKey?: string;
  /** "全部应用范围" 文案（默认 "全部应用范围"） */
  allLabel?: string;
  /** "全部用户" 组织标题（默认 "全部用户"） */
  publicGroupLabel?: string;
  /** "按组织" 标题（默认 "按组织"） */
  groupSectionLabel?: string;
  /** 已选计数文案模板，{count} 会被替换（默认 "已选 {count} 项"） */
  selectedCountTemplate?: string;
  /** 面板宽度 class（默认 "w-56"） */
  widthClass?: string;
  /**
   * 隐藏「全部用户」分区（含段标题与选项行）。
   * 适用：publicKey 在外部已通过 groups 注入为伪组织（如"未分配组织"）的场景，
   * 避免出现冗余的段标题。默认 false。
   */
  hidePublicGroup?: boolean;
}

// ─── 组件实现 ────────────────────────────────────────────────────────────────

export function ScopeFilterDropdown({
  groups,
  selectedKeys,
  onChange,
  searchPlaceholder = "搜索...",
  publicKey = "public",
  allLabel = "全部应用范围",
  publicGroupLabel = "全部用户",
  groupSectionLabel = "按组织",
  selectedCountTemplate = "已选 {count} 项",
  widthClass = "w-56",
  hidePublicGroup = false,
}: ScopeFilterDropdownProps) {
  const [searchQuery, setSearchQuery] = useState("");

  // 所有可选 key（public + 所有组织 id）；hidePublicGroup 时不包含 publicKey
  const allKeys = useMemo(
    () => (hidePublicGroup ? groups.map((g) => g.id) : [publicKey, ...groups.map((g) => g.id)]),
    [publicKey, groups, hidePublicGroup]
  );

  const isAllSelected =
    allKeys.length > 0 && allKeys.every((k) => selectedKeys.has(k));

  // 搜索过滤
  const filteredGroups = useMemo(
    () =>
      groups.filter((g) =>
        g.name.toLowerCase().includes(searchQuery.toLowerCase())
      ),
    [groups, searchQuery]
  );

  const showPublic =
    !hidePublicGroup && (!searchQuery || publicGroupLabel.includes(searchQuery));
  const showGroupSection =
    !searchQuery ||
    groupSectionLabel.includes(searchQuery) ||
    filteredGroups.length > 0;

  // 切换单个 key
  const toggleKey = (key: string) => {
    const next = new Set(selectedKeys);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    onChange(next);
  };

  // 全选/全不选
  const toggleAll = () => {
    if (isAllSelected) {
      onChange(new Set());
    } else {
      onChange(new Set(allKeys));
    }
    setSearchQuery("");
  };

  // 清除
  const handleClear = () => {
    onChange(new Set());
    setSearchQuery("");
  };

  const selectedCount = selectedKeys.size;
  const countText = selectedCountTemplate.replace(
    "{count}",
    String(selectedCount)
  );

  return (
    <div
      className={`bg-white rounded-[4px] shadow-[var(--shadow-popover)] pt-2 px-2 pb-0 ${widthClass}`}
    >
      {/* 搜索框 */}
      <div className="mb-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 pointer-events-none" />
          <Input
            type="text"
            placeholder={searchPlaceholder}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8 pl-8 pr-2 text-sm"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      </div>

      {/* 全部应用范围 — 全选/全不选切换 */}
      {(!searchQuery || allLabel.includes(searchQuery)) && (
        <button
          type="button"
          onClick={toggleAll}
          className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${
            isAllSelected
              ? "bg-[var(--bg-brand-selected)]"
              : "hover:bg-[var(--bg-grey-hover)]"
          }`}
        >
          <Checkbox checked={isAllSelected} className="pointer-events-none" />
          <BodyText tone="secondary" className="truncate text-left">
            {allLabel}
          </BodyText>
        </button>
      )}

      {/* 全部用户 区域 */}
      {showPublic && (
        <>
          <div className="px-3 pt-2 pb-1 select-none">
            <MetaMedium tone="weak">{publicGroupLabel}</MetaMedium>
          </div>
          <button
            type="button"
            onClick={() => toggleKey(publicKey)}
            className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${
              selectedKeys.has(publicKey)
                ? "bg-[var(--bg-brand-selected)]"
                : "hover:bg-[var(--bg-grey-hover)]"
            }`}
          >
            <Checkbox
              checked={selectedKeys.has(publicKey)}
              className="pointer-events-none"
            />
            <BodyText tone="secondary" className="truncate text-left">
              {publicGroupLabel}
            </BodyText>
          </button>
        </>
      )}

      {/* 按组织 区域 */}
      {showGroupSection && (
        <>
          <div className="px-3 pt-2.5 pb-1 select-none">
            <MetaMedium tone="weak">{groupSectionLabel}</MetaMedium>
          </div>
          <div
            className="max-h-44 overflow-y-auto space-y-0.5 pb-2"
            onWheel={(e) => e.stopPropagation()}
          >
            {filteredGroups.map((group) => {
              const checked = selectedKeys.has(group.id);
              return (
                <button
                  key={group.id}
                  type="button"
                  onClick={() => toggleKey(group.id)}
                  className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${
                    checked
                      ? "bg-[var(--bg-brand-selected)]"
                      : "hover:bg-[var(--bg-grey-hover)]"
                  }`}
                >
                  <Checkbox checked={checked} className="pointer-events-none" />
                  <BodyText
                    tone="secondary"
                    className="truncate text-left"
                    title={group.name}
                  >
                    {group.name}
                  </BodyText>
                </button>
              );
            })}
            {filteredGroups.length === 0 && !showPublic && searchQuery && (
              <MetaText as="p" tone="weak" className="py-2 text-center">
                没有匹配的结果
              </MetaText>
            )}
          </div>
        </>
      )}

      {/* 底部：已选数量 + 清除 */}
      {selectedCount > 0 && (
        <div className="border-t border-[#EAEEF4] py-2 flex items-center justify-between">
          <MetaText>{countText}</MetaText>
          <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={handleClear}>
            清除
          </Button>
        </div>
      )}
    </div>
  );
}

export default ScopeFilterDropdown;
