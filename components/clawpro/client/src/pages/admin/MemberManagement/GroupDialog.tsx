/**
 * 组织弹窗组件
 *   - 新建组织（GroupFormDialog）
 *   - 编辑组织（GroupFormDialog mode="edit"）
 *   - 添加子组织（GroupFormDialog mode="addChild"）
 *   - 删除组织确认（DeleteGroupDialog）
 */
import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  ChevronRight,
  ChevronDown,
  Loader2,
  RefreshCw,
  Search,
  X,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogBody,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { BodyMedium, HelperText, MetaText, MetaMedium } from "@/components/ui/Typography";
import { CircleAlert } from "lucide-react";
import type { UserGroup } from "./types";
import {
  buildUnifiedGroupTree,
  type GroupTreeNode,
  findGroupNode,
  getResourcesOfGroup,
} from "./health";
import { MOCK_USER_GROUP_AGENTS } from "./mock";

// ─── 下拉树形选择器（单选，用于选择上级组织） ────────────────────
function ParentDropdownSelector({
  groups,
  value,
  onChange,
  disabled,
  excludeIds,
  term = "组织",
}: {
  groups: UserGroup[];
  value: string | null;
  onChange: (id: string | null) => void;
  disabled?: boolean;
  /** 排除的组织 id 集合（编辑时排除自身及子孙） */
  excludeIds?: Set<string>;
  /** 术语：组织 / 项目，用于占位与空态文案 */
  term?: string;
}) {
  // 合并树构建：若数据中包含 oneid-dept，则把 A公司（dept-root）作为唯一顶层，
  // 并把原本顶层的 oneid-group / manual 节点挂到 A公司 下。
  // 否则退化为普通的"自定义组织"树。
  const tree = useMemo(() => buildUnifiedGroupTree(groups), [groups]);

  // 已同步部门时：oneid-dept 部门节点在选择器中**置灰不可选**（部门来自数据源，不可作为本地组织的父级）。
  // 但 A公司根节点（dept-root）**允许选**，因为它代表"挂在公司一级"。
  const hasDept = useMemo(
    () => groups.some((g) => g.source === "oneid-dept"),
    [groups]
  );
  const isNodeDisabled = (node: GroupTreeNode): boolean =>
    hasDept && node.source === "oneid-dept" && node.id !== "dept-root";

  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);

  const [expanded, setExpanded] = useState<Set<string>>(() => {
    const s = new Set<string>();
    // 合并模式（已同步部门）下：默认仅展开 A公司(dept-root) + 用户组(oneid-group) / 自定义组织(manual)，
    // 部门(同步来的 oneid-dept)子级默认收起；普通模式下全展开。
    const walk = (nodes: GroupTreeNode[]) => {
      nodes.forEach((n) => {
        if (
          !hasDept ||
          n.id === "dept-root" ||
          n.source === "oneid-group" ||
          n.source === "manual"
        ) {
          s.add(n.id);
        }
        walk(n.children);
      });
    };
    walk(tree);
    return s;
  });

  // 打开时聚焦搜索框并清空搜索
  useEffect(() => {
    if (dropdownOpen) {
      setSearchQuery("");
      setTimeout(() => searchInputRef.current?.focus(), 0);
    }
  }, [dropdownOpen]);

  const toggle = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  // 获取选中组织的完整路径名称（如 "研发组/研发-前端"）
  const selectedFullPath = useMemo(() => {
    if (!value) return null;
    const node = findGroupNode(tree, value);
    if (!node) return null;
    // node.path 格式 "A / B"，转为 "A/B"
    return node.path.replace(/\s*\/\s*/g, "/");
  }, [value, tree]);

  // 搜索过滤：收集匹配节点 id 及其所有祖先 id
  const matchedIds = useMemo(() => {
    if (!searchQuery.trim()) return null; // null 表示不过滤
    const q = searchQuery.trim().toLowerCase();
    const matched = new Set<string>();
    const walkCollect = (nodes: GroupTreeNode[]) => {
      for (const n of nodes) {
        if (excludeIds?.has(n.id)) continue;
        if (n.name.toLowerCase().includes(q)) {
          // 添加该节点及其所有祖先
          for (const pid of n.pathIds) matched.add(pid);
        }
        walkCollect(n.children);
      }
    };
    walkCollect(tree);
    return matched;
  }, [searchQuery, tree, excludeIds]);

  const renderNode = (node: GroupTreeNode): React.ReactNode => {
    if (excludeIds?.has(node.id)) return null;
    // 如果正在搜索且该节点不在匹配集中，隐藏
    if (matchedIds && !matchedIds.has(node.id)) return null;

    const isSelected = value === node.id;
    const isExp = expanded.has(node.id);
    const hasChildren = node.children.length > 0;
    // 搜索时强制展开所有匹配路径
    const shouldShow = matchedIds ? true : isExp;
    const isDisabled = isNodeDisabled(node);

    return (
      <div key={node.id}>
        <div
          className={`flex items-center gap-1.5 h-8 px-2 rounded-[6px] text-sm transition-colors ${
            isDisabled
              ? "text-[#A3A3A3] cursor-not-allowed"
              : isSelected
                ? "text-[#1447E6] font-medium bg-[#FAFAFA] cursor-pointer"
                : "text-[#0A0A0A] hover:bg-[var(--bg-grey-hover)] cursor-pointer"
          }`}
          style={{ paddingLeft: 8 + node.depth * 16 }}
          onClick={() => {
            if (isDisabled) return;
            onChange(isSelected ? null : node.id);
            if (!isSelected) setDropdownOpen(false);
          }}
        >
          {hasChildren ? (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                toggle(node.id);
              }}
              className="w-4 h-4 flex items-center justify-center text-[#A3A3A3] hover:text-[#0A0A0A] shrink-0"
            >
              {(shouldShow) ? (
                <ChevronDown className="w-3.5 h-3.5" />
              ) : (
                <ChevronRight className="w-3.5 h-3.5" />
              )}
            </button>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <span className="truncate flex-1">{node.name}</span>
        </div>
        {hasChildren && shouldShow && node.children.map(renderNode)}
      </div>
    );
  };

  return (
    <Popover open={dropdownOpen} onOpenChange={setDropdownOpen}>
      <PopoverTrigger asChild>
        {/*
          视觉对齐 SelectTrigger（rounded-[4px]、#E5E5E5 边、hover/open 蓝边、ChevronDown 旋转）。
          移除原先的 X 清除按钮——用户在面板内再次点击当前项即可取消选择，与 SelectItem 行为一致。
        */}
        <button
          type="button"
          disabled={disabled}
          aria-haspopup="listbox"
          aria-expanded={dropdownOpen}
          data-state={dropdownOpen ? "open" : "closed"}
          className={
            "flex w-full items-center justify-between gap-2 h-9 px-3 text-sm font-normal whitespace-nowrap " +
            "bg-white border border-[#e5e5e5] rounded-[4px] transition-colors outline-none " +
            "hover:border-[#1447E6] data-[state=open]:border-[#1447E6] " +
            "disabled:cursor-not-allowed disabled:bg-[#FAFAFA] disabled:text-[var(--text-weak)]"
          }
        >
          {selectedFullPath ? (
            <span className="truncate text-[#0A0A0A] flex-1 text-left">
              {selectedFullPath}
            </span>
          ) : (
            <span className="flex-1 text-left text-[#A3A3A3]">
              {hasDept ? `请选择上级${term}` : `选填，不选则为一级${term}`}
            </span>
          )}
          <ChevronDown className="size-4 shrink-0 text-[#737373] transition-transform duration-200 [[data-state=open]>&]:rotate-180" />
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        sideOffset={4}
        className={
          "w-[var(--radix-popover-trigger-width)] " +
          // 与 SelectContent 同款：4px 圆角 + 双层阴影；override popover 默认值
          // allow-radius: 上面注释里 8px 是 popover 默认圆角说明，已被本组件覆盖为 4px
          "rounded-[4px] p-0 border border-[#e5e5e5] " +
          "shadow-[var(--shadow-popover)] " +
          // 让面板被可视区高度约束 + 内部 flex 列布局，列表区可滚动
          "max-h-[var(--radix-popover-content-available-height)] flex flex-col overflow-hidden"
        }
      >
        {/* 搜索框 */}
        <div className="shrink-0 px-2 pt-2 pb-1.5 border-b border-[#e5e5e5]">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-[#A3A3A3] pointer-events-none" />
            <Input
              ref={searchInputRef}
              placeholder={`搜索${term}`}
              className="h-8 pl-7 pr-7 text-sm"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {searchQuery && (
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-[#A3A3A3] hover:text-[#0A0A0A] transition-colors"
                onClick={() => setSearchQuery("")}
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>
        </div>
        {/* 树形列表 — 内容溢出时此区域滚动，面板整体高度受可视区约束。
            onWheel 阻止冒泡，避免滚轮事件被外层 Dialog 拦截导致无法滚动；
            overscroll-contain 防止滚动穿透到底层页面。 */}
        <div
          className="flex-1 min-h-0 overflow-y-auto overscroll-contain p-1.5"
          onWheel={(e) => e.stopPropagation()}
        >
          {tree.length === 0 ? (
            <HelperText className="text-center py-3">
              暂无可选{term}
            </HelperText>
          ) : matchedIds && matchedIds.size === 0 ? (
            <HelperText className="text-center py-3">
              未找到匹配{term}
            </HelperText>
          ) : (
            tree.map(renderNode)
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

// ─── 新建 / 编辑 / 添加子组织弹窗 ──────────────────────────
export interface GroupFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groups: UserGroup[];
  /** "create" | "edit" | "addChild" */
  mode: "create" | "edit" | "addChild";
  /** 编辑/添加子组织时的参考组织 */
  target?: { id: string; name: string; parentId: string | null } | null;
  onConfirm: (name: string, parentId: string | null) => void;
  /** 术语：组织 / 项目，用于标题与表单文案，默认"组织" */
  term?: string;
  /** 单层级模式：隐藏"上级"选择器，始终作为顶层创建（用于项目，不支持多层级） */
  singleLevel?: boolean;
}

export function GroupFormDialog({
  open,
  onOpenChange,
  groups,
  mode,
  target,
  onConfirm,
  term = "组织",
  singleLevel = false,
}: GroupFormDialogProps) {
  const [name, setName] = useState("");
  const [parentId, setParentId] = useState<string | null>(null);
  const nameInputRef = useRef<HTMLInputElement>(null);

  // OneID 是否已同步部门（数据中包含 oneid-dept）—— 决定新建/添加子组织时的默认父级
  const hasDept = useMemo(
    () => groups.some((g) => g.source === "oneid-dept"),
    [groups]
  );

  // 初始化
  useEffect(() => {
    if (!open) return;
    // 单层级模式（项目）：始终顶层，无上级概念
    if (singleLevel) {
      setName(mode === "edit" && target ? target.name : "");
      setParentId(null);
      return;
    }
    if (mode === "edit" && target) {
      setName(target.name);
      setParentId(target.parentId);
    } else if (mode === "addChild" && target) {
      setName("");
      setParentId(target.id);
    } else {
      setName("");
      // OneID 已同步部门时，"新建"对话框默认父级为 A公司（dept-root）；否则保持为顶层组织
      setParentId(hasDept ? "dept-root" : null);
    }
  }, [open, mode, target, hasDept]);

  // 编辑时排除自身及子孙（不能把自己设为自己或子孙的子节点）
  const excludeIds = useMemo(() => {
    if (mode !== "edit" || !target) return undefined;
    const s = new Set<string>();
    const t = buildUnifiedGroupTree(groups);
    const walk = (nodes: GroupTreeNode[]) => {
      for (const n of nodes) {
        if (n.id === target.id) {
          const addAll = (node: GroupTreeNode) => {
            s.add(node.id);
            node.children.forEach(addAll);
          };
          addAll(n);
          return;
        }
        walk(n.children);
      }
    };
    walk(t);
    return s;
  }, [mode, target, groups]);

  const isDuplicate = groups.some(
    (g) =>
      g.name.trim() === name.trim() &&
      g.source === "manual" &&
      (mode !== "edit" || g.id !== target?.id)
  );

  const isValid = name.trim().length > 0 && !isDuplicate;

  const title =
    mode === "create"
      ? `新建${term}`
      : mode === "edit"
        ? `编辑${term}`
        : `添加子${term}`;

  const confirmText =
    mode === "create" ? "确认创建" : mode === "edit" ? "保存" : "确认创建";

  // addChild 模式下，上级组织选中且锁定
  const parentLocked = mode === "addChild";

  // 锁定时显示上级名称
  const lockedParentName = parentLocked && target
    ? groups.find((g) => g.id === target.id)?.name ?? target.name
    : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          <div className="space-y-4">
            {/* 上级组织（单层级模式隐藏：项目不支持多层级） */}
            {!singleLevel && (
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">
                上级{term}
              </MetaMedium>
              {parentLocked && lockedParentName ? (
                <Input
                  value={lockedParentName}
                  disabled
                  readOnly
                />
              ) : (
                <ParentDropdownSelector
                  groups={groups}
                  value={parentId}
                  onChange={setParentId}
                  excludeIds={excludeIds}
                  term={term}
                />
              )}
            </div>
            )}

            {/* 组织名称 */}
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">
                {term}名称<span className="text-[#DC2626] ml-0.5">*</span>
              </MetaMedium>
              <Input
                ref={nameInputRef}
                type="text"
                placeholder={`请输入${term}名称`}
                className={`h-9 text-sm ${
                  isDuplicate && name.trim()
                    ? "border-[#d42a1e] focus-visible:border-[#d42a1e]"
                    : ""
                }`}
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
              />
              {isDuplicate && name.trim() && (
                <MetaText as="p" tone="danger">{term}名称已存在</MetaText>
              )}
              <HelperText>
                {term}名称为唯一标识，不能与已有{term}重名，创建后支持修改
              </HelperText>
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            disabled={!isValid}
            onClick={() => {
              if (!isValid) return;
              onConfirm(name.trim(), parentId);
            }}
          >
            {confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 删除组织确认弹窗 ──────────────────────────────────────
export interface DeleteGroupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group: { id: string; name: string } | null;
  memberCount: number;
  groups: UserGroup[];
  onConfirm: (groupId: string) => void;
  /** 术语：组织 / 项目，用于标题与信息文案，默认"组织" */
  term?: string;
  /** 是否校验并展示"下级 Agent 实例"，默认 true；项目删除时置为 false（仅校验专属配置） */
  checkAgentInstances?: boolean;
}

/** 统计某组织下的 Agent 实例（仅该组织自身，不递归子组织） */
function getGroupAgentStats(groupId: string) {
  let instanceCount = 0;
  const userIds = new Set<string>();
  for (const [userId, groupMap] of Object.entries(MOCK_USER_GROUP_AGENTS)) {
    const instances = groupMap[groupId];
    if (instances && instances.length > 0) {
      instanceCount += instances.length;
      userIds.add(userId);
    }
  }
  return { instanceCount, userCount: userIds.size };
}

export function DeleteGroupDialog({
  open,
  onOpenChange,
  group,
  memberCount,
  groups,
  onConfirm,
  term = "组织",
  checkAgentInstances = true,
}: DeleteGroupDialogProps) {
  const [configRefreshing, setConfigRefreshing] = useState(false);
  const [agentRefreshing, setAgentRefreshing] = useState(false);

  // 获取关联的资源配置
  const relatedResources = useMemo(
    () => (group ? getResourcesOfGroup(group.id) : []),
    [group]
  );

  const hasRelatedConfigs = relatedResources.length > 0;

  // 按 kind 去重（只关心有哪些类别，不计数）
  const configKinds = useMemo(() => {
    const kinds = new Set<string>();
    relatedResources.forEach((r) => kinds.add(r.kind));
    return Array.from(kinds);
  }, [relatedResources]);

  // 统计该组织下的 Agent 实例
  const agentStats = useMemo(
    () => (group ? getGroupAgentStats(group.id) : { instanceCount: 0, userCount: 0 }),
    [group]
  );
  const hasAgentInstances = checkAgentInstances && agentStats.instanceCount > 0;

  // 是否可以删除：无配置且（不校验实例 或 无实例）
  const canDelete = !hasRelatedConfigs && !hasAgentInstances;

  const kindLabel: Record<string, string> = {
    model: "模型",
    channel: "通道",
    skill: "技能",
    agentTool: "Agent 工具",
    memory: "记忆",
    drive: "网盘",
    image: "镜像",
    network: "网络",
    securityGroup: "网络",
    vpc: "网络",
    cls: "CLS 日志服务",
    aiAgentSecurity: "AI Agent 安全",
    platformPolicy: "平台策略",
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent
        className="sm:max-w-md"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
      >
        <AlertDialogHeader>
          <AlertDialogTitle className="text-[#0A0A0A]">删除{term}</AlertDialogTitle>
        </AlertDialogHeader>

        <div className="flex-1 overflow-y-auto">
          <div className="space-y-4">
            {/* 状态提示（统一使用黄色 warning Alert） */}
            <Alert variant="warning">
              <CircleAlert />
              {canDelete ? (
                <AlertDescription>
                  该{term}{checkAgentInstances ? '无关联配置且无 Agent 实例' : '无关联专属配置'}，可安全删除。删除后{term}内用户不会被删除，仅解除{term}关联。
                </AlertDescription>
              ) : (
                <>
                  <AlertTitle>无法删除该{term}</AlertTitle>
                  <AlertDescription>
                    <ul className="list-disc pl-4 space-y-0.5">
                      {hasRelatedConfigs && (
                        <li>以上配置的应用范围包含该{term}，请先前往对应配置页面移除该{term}后再执行删除。</li>
                      )}
                      {hasAgentInstances && (
                        <li>该{term}下仍有 Agent 实例，请先删除实例后再执行删除。</li>
                      )}
                    </ul>
                  </AlertDescription>
                </>
              )}
            </Alert>

            {/* 组织信息卡片（合并为单一卡片，所有内容右对齐统一正文样式） */}
            <div className="rounded-[4px] border border-[#e5e5e5] bg-white divide-y divide-[#e5e5e5]">
              {/* 组织名称 */}
              <div className="px-4 py-3 flex items-center justify-between gap-4">
                <MetaMedium as="label" tone="secondary">{term}名称</MetaMedium>
                <BodyMedium className="text-right">{group?.name}</BodyMedium>
              </div>

              {/* 组织内用户数 */}
              <div className="px-4 py-3 flex items-center justify-between gap-4">
                <MetaMedium as="label" tone="secondary">{term}内用户数</MetaMedium>
                <BodyMedium className="text-right">{memberCount} 人</BodyMedium>
              </div>

              {/* 组织专属配置 */}
              <div className="px-4 py-3 flex items-center justify-between gap-4">
                <MetaMedium as="label" tone="secondary">{term}专属配置</MetaMedium>
                <div className="flex items-center gap-2 min-w-0">
                  <BodyMedium className="text-right truncate">
                    {hasRelatedConfigs
                      ? configKinds.map((kind) => kindLabel[kind] ?? kind).join('、')
                      : '无关联配置'}
                  </BodyMedium>
                  <button
                    className="text-[#737373] hover:text-[#0A0A0A] transition-colors shrink-0"
                    title="刷新"
                    onClick={() => {
                      setConfigRefreshing(true);
                      setTimeout(() => setConfigRefreshing(false), 1200);
                    }}
                  >
                    {configRefreshing ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="w-3.5 h-3.5" />
                    )}
                  </button>
                </div>
              </div>

              {/* Agent 实例数（项目删除不校验实例时隐藏） */}
              {checkAgentInstances && (
              <div className="px-4 py-3 flex items-center justify-between gap-4">
                <MetaMedium as="label" tone="secondary">{term}下 Agent 实例</MetaMedium>
                <div className="flex items-center gap-2 min-w-0">
                  <BodyMedium className="text-right truncate">
                    {hasAgentInstances ? `${agentStats.instanceCount} 个实例` : '无 Agent 实例'}
                  </BodyMedium>
                  <button
                    className="text-[#737373] hover:text-[#0A0A0A] transition-colors shrink-0"
                    title="刷新"
                    onClick={() => {
                      setAgentRefreshing(true);
                      setTimeout(() => setAgentRefreshing(false), 1200);
                    }}
                  >
                    {agentRefreshing ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="w-3.5 h-3.5" />
                    )}
                  </button>
                </div>
              </div>
              )}
            </div>
          </div>
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => onOpenChange(false)}>
            取消
          </AlertDialogCancel>
          {canDelete && (
            <AlertDialogAction
              variant="destructive"
              onClick={() => group && onConfirm(group.id)}
            >
              确认删除
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
