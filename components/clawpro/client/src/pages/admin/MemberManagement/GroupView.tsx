/**
 * 组织视图容器（v3.0）
 *
 * 布局：
 *   - 常驻 Alert：当存在任何多归属用户时显示（不可关闭）
 *   - 主体：左右合为一个大卡片，中间可拖拽分割线，支持收起/展开左侧面板
 *   - 左面板顶部：标题"组织" + "新建"按钮 + 收起按钮；下方搜索框 + 刷新按钮
 *   - 右面板：组织详情/成员表格
 */
import React, { useEffect, useMemo, useState, useCallback, useRef } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription, AlertOperationInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { SurfaceCard } from "@/components/ui/Surface";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import GroupList, { UNASSIGNED_GROUP_ID } from "./GroupList";
import NodeContentPanel from "./NodeContentPanel";
import { GroupFormDialog, DeleteGroupDialog } from "./GroupDialog";

import type { UserGroup, UserOrg, UserOverrideInfo, AnomalousGroup } from "./types";
import {
  getUsersOfGroupDeep,
  buildGroupTree,
  findGroupNode,
  getGroupInitHealth,
  hasNetworkOutdated,
} from "./health";
import { MOCK_GROUPS, MOCK_MANUAL_GROUPS, MOCK_USERS_MANUAL, MOCK_SYNC_RESULT, MOCK_USER_GROUP_AGENTS, getPrimaryDeptPath } from "./mock";
import { BodyText, MetaMedium } from "@/components/ui/Typography";
import AgentInstanceHandlingDialog, {
  type ParentMigration,
  type AgentInstance,
} from "./AgentInstanceHandlingDialog";
import ConfigDiffDialog, {
  buildMockInstanceCompare,
  type InstanceConfigCompare,
} from "./ConfigDiffDialog";

// OneID 全量组织架构 mock（同步时一次性全部拉取）
const ONEID_ALL_DEPT_NODES: Array<{
  id: string;
  name: string;
  parentId: string | null;
}> = [
  { id: "dept-root", name: "A公司", parentId: null },
  { id: "dept-tech", name: "技术部", parentId: "dept-root" },
  { id: "dept-fe", name: "前端组", parentId: "dept-tech" },
  { id: "dept-be", name: "后端组", parentId: "dept-tech" },
  { id: "dept-ai", name: "AI 组", parentId: "dept-tech" },
  { id: "dept-devops", name: "运维组", parentId: "dept-tech" },
  { id: "dept-qa", name: "测试组", parentId: "dept-tech" },
  { id: "dept-product", name: "产品部", parentId: "dept-root" },
  { id: "dept-pm", name: "产品策划", parentId: "dept-product" },
  { id: "dept-design", name: "设计组", parentId: "dept-product" },
  { id: "dept-operation", name: "运营组", parentId: "dept-product" },
  { id: "dept-operation-1", name: "运营一组", parentId: "dept-operation" },
  { id: "dept-operation-2", name: "运营二组", parentId: "dept-operation" },
  { id: "dept-hr", name: "人力资源", parentId: "dept-root" },
  { id: "dept-finance", name: "财务部", parentId: "dept-root" },
  { id: "dept-legal", name: "法务部", parentId: "dept-root" },
];

interface GroupViewProps {
  /** 是否开启 OneID 模式。OneID：使用 oneid-dept + oneid-group；普通：使用 manual */
  hasOneid: boolean;
  /**
   * OneID 中是否存在部门数据。仅在 `hasOneid=true` 时生效；为 false 时：
   * - 左树隐藏「组织架构/部门」整段
   * - 右侧成员表隐藏「部门」列与"部门："信息
   */
  hasDeptData?: boolean;
  users: UserOrg[];
  overrides: Record<string, UserOverrideInfo>;
  onResolveConflict: (userId: string, winnerResourceId: string) => void;
  /** 通知父组件弹出同步结果弹窗（传入组织异常数据） */
  onShowSyncResult?: (anomalousGroups: AnomalousGroup[]) => void;
  /** 通知父组件组织架构是否已同步为组织 */
  onDeptSyncedChange?: (synced: boolean) => void;
  /** 父组件下发的异常组织数据（手动同步按钮触发时传入） */
  externalAnomalousGroups?: AnomalousGroup[];
}

export default function GroupView({
  hasOneid,
  hasDeptData = true,
  users,
  overrides,
  onResolveConflict,
  onShowSyncResult,
  onDeptSyncedChange,
  externalAnomalousGroups,
}: GroupViewProps) {
  // ─── 左侧面板：拖拽调宽 + 折叠 ─────────────────────────
  const [leftWidth, setLeftWidth] = useState(288);
  const [leftCollapsed, setLeftCollapsed] = useState(false);
  const isDragging = useRef(false);
  const startX = useRef(0);
  const startWidth = useRef(0);

  const handleMouseDown = (e: React.MouseEvent) => {
    isDragging.current = true;
    startX.current = e.clientX;
    startWidth.current = leftWidth;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    const handleMouseMove = (ev: MouseEvent) => {
      if (!isDragging.current) return;
      const delta = ev.clientX - startX.current;
      const newWidth = Math.min(Math.max(startWidth.current + delta, 200), 480);
      setLeftWidth(newWidth);
    };
    const handleMouseUp = () => {
      isDragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  // OneID 模式下，组织架构初始未同步，需管理员手动触发
  const [deptSynced, setDeptSynced] = useState(false);
  const [isSyncingDepts, setIsSyncingDepts] = useState(false);

  // OneID 模式下，用户组初始已加载
  const [ogSynced, setOgSynced] = useState(hasOneid);
  const [isRefreshingOg, setIsRefreshingOg] = useState(false);

  // ─── 同步异常组织 ────────────────────────────────────────
  /** 当前异常组织列表（配置未解绑的） */
  const [anomalousGroups, setAnomalousGroups] = useState<AnomalousGroup[]>([]);

  // 父组件通过手动同步按钮下发异常组织数据时，同步到内部状态（显示红点）
  useEffect(() => {
    if (externalAnomalousGroups && externalAnomalousGroups.length > 0) {
      setAnomalousGroups(externalAnomalousGroups);
    }
  }, [externalAnomalousGroups]);

  // 组织集合：OneID 模式初始加载自定义组织（用户组），组织架构需要同步后才加入
  const [groups, setGroups] = useState<UserGroup[]>(() => {
    if (hasOneid) {
      // 初始加载自定义组织（oneid-group），组织架构仍需手动同步
      return MOCK_GROUPS.filter((g) => g.source === "oneid-group");
    }
    return MOCK_MANUAL_GROUPS;
  });

  /** 直接异常组织 id 集合：异常组织自身 + 其子组织（不含父组织冒泡） */
  const directAnomalousGroupIds = useMemo(() => {
    if (anomalousGroups.length === 0) return new Set<string>();
    const ids = new Set<string>();

    anomalousGroups.forEach((ag) => {
      ids.add(ag.groupId);
      const addChildren = (parentId: string) => {
        groups.forEach((g) => {
          if (g.parentId === parentId) {
            ids.add(g.id);
            addChildren(g.id);
          }
        });
      };
      addChildren(ag.groupId);
    });

    return ids;
  }, [anomalousGroups, groups]);

  /** 计算需要显示红点的完整 id 集合：异常组织自身 + 其子组织 + 其父组织链 */
  const anomalousGroupIds = useMemo(() => {
    if (anomalousGroups.length === 0) return new Set<string>();
    const ids = new Set<string>(directAnomalousGroupIds);
    const groupMap = new Map(groups.map((g) => [g.id, g]));

    anomalousGroups.forEach((ag) => {
      // 所有父组织链（冒泡）
      let cur = groupMap.get(ag.groupId);
      while (cur && cur.parentId) {
        ids.add(cur.parentId);
        cur = groupMap.get(cur.parentId);
      }
    });

    return ids;
  }, [anomalousGroups, groups, directAnomalousGroupIds]);

  /** 异常组织详情 Map（groupId -> AnomalousGroup），供 Tooltip 动态文案使用 */
  const anomalousGroupDetails = useMemo(() => {
    const map = new Map<string, AnomalousGroup>();
    anomalousGroups.forEach((ag) => map.set(ag.groupId, ag));
    return map;
  }, [anomalousGroups]);

  /** 直接初始化未完成组织 id 集合（自身，不含父组织冒泡） */
  const directUninitializedGroupIds = useMemo(() => {
    const ids = new Set<string>();
    groups.forEach((g) => {
      const initHealth = getGroupInitHealth(g.id, groups);
      if (!initHealth.initialized) {
        ids.add(g.id);
      }
    });
    return ids;
  }, [groups]);

  /** 完整初始化未完成组织 id 集合（自身 + 父组织链冒泡） */
  const uninitializedGroupIds = useMemo(() => {
    if (directUninitializedGroupIds.size === 0) return new Set<string>();
    const ids = new Set<string>(directUninitializedGroupIds);
    const groupMap = new Map(groups.map((g) => [g.id, g]));

    directUninitializedGroupIds.forEach((gId) => {
      let cur = groupMap.get(gId);
      while (cur && cur.parentId) {
        ids.add(cur.parentId);
        cur = groupMap.get(cur.parentId);
      }
    });

    return ids;
  }, [directUninitializedGroupIds, groups]);

  /**
   * 网络配置待更新组织 id 集合（红色小圆点）
   *
   * 仅命中组织自身（不冒泡到父组织、不下发到子组织、不影响兄弟组织）。
   * 用于：左侧组织树该组织行的红点提示。
   */
  const networkOutdatedGroupIds = useMemo(() => {
    const ids = new Set<string>();
    groups.forEach((g) => {
      if (hasNetworkOutdated(g.id, groups)) {
        ids.add(g.id);
      }
    });
    return ids;
  }, [groups]);

  // OneID 切换时切换组织集合
  React.useEffect(() => {
    if (hasOneid) {
      // 加载自定义组织（oneid-group），组织架构需手动同步
      setGroups(MOCK_GROUPS.filter((g) => g.source === "oneid-group"));
      setDeptSynced(false);
      setOgSynced(true);
    } else {
      setGroups(MOCK_MANUAL_GROUPS);
      setDeptSynced(false);
      setOgSynced(false);
    }
  }, [hasOneid]);

  // 用户数据源
  const effectiveUsers = useMemo<UserOrg[]>(
    () => (hasOneid ? users : MOCK_USERS_MANUAL),
    [hasOneid, users]
  );

  const [selectedId, setSelectedId] = useState<string>(() => groups[0]?.id ?? "");

  // 选中项回退保护
  React.useEffect(() => {
    if (
      selectedId !== UNASSIGNED_GROUP_ID &&
      !groups.find((g) => g.id === selectedId)
    ) {
      setSelectedId(groups[0]?.id ?? "");
    }
  }, [groups, selectedId]);

  const selectedGroup = selectedId === UNASSIGNED_GROUP_ID
    ? null
    : groups.find((g) => g.id === selectedId);
  const tree = useMemo(() => buildGroupTree(groups), [groups]);
  const selectedNode = selectedGroup
    ? findGroupNode(tree, selectedGroup.id)
    : null;

  // 节点成员统计（含子孙聚合）
  const groupUsers = useMemo(() => {
    if (selectedId === UNASSIGNED_GROUP_ID) {
      // 未分配组织用户：不属于当前已加载组织的用户
      const loadedGroupIds = new Set(groups.map((g) => g.id));
      if (loadedGroupIds.size === 0) return effectiveUsers; // 没有组织 → 全部
      return effectiveUsers.filter(
        (u) => !u.groupIds.some((gid) => loadedGroupIds.has(gid))
      );
    }
    return selectedGroup
      ? getUsersOfGroupDeep(selectedGroup.id, groups, effectiveUsers)
      : [];
  }, [selectedId, selectedGroup, groups, effectiveUsers]);

  // 一键同步全部组织架构
  const handleSyncDepts = () => {
    setIsSyncingDepts(true);
    // 模拟同步延迟
    setTimeout(() => {
      const deptGroups: UserGroup[] = ONEID_ALL_DEPT_NODES.map((n) => ({
        id: n.id,
        name: n.name,
        parentId: n.parentId,
        source: "oneid-dept" as const,
        readonly: true,
        externalId: n.id,
        syncBatchId: "oneid-org",
        createdAt: new Date().toISOString(),
      }));
      setGroups((prev) => {
        // 移除旧的 oneid-dept，加入全量
        const withoutDept = prev.filter((g) => g.source !== "oneid-dept");
        // 将顶层 oneid-group（parentId === null）挂到根组织 dept-root 下，
        // 使其与"技术部"等顶层部门同级，统一以"A公司"为最上层
        const remappedOg = withoutDept.map((g) =>
          g.source === "oneid-group" && g.parentId === null
            ? { ...g, parentId: "dept-root" }
            : g
        );
        return [...remappedOg, ...deptGroups];
      });
      setDeptSynced(true);
      setIsSyncingDepts(false);
      onDeptSyncedChange?.(true);
      toast.success("已同步数据源");
    }, 1200);
  };

  // 加载用户组（mock：加载 oneid-group 类型的自建组织数据）
  const handleRefreshOg = () => {
    setIsRefreshingOg(true);
    setTimeout(() => {
      const ogGroupsRaw = MOCK_GROUPS.filter((g) => g.source === "oneid-group");
      setGroups((prev) => {
        const withoutOg = prev.filter((g) => g.source !== "oneid-group");
        // 若已同步部门，顶层 oneid-group 需挂到 dept-root 下，保持与"技术部"同级
        const deptSyncedNow = withoutOg.some((g) => g.source === "oneid-dept");
        const ogGroups = deptSyncedNow
          ? ogGroupsRaw.map((g) =>
              g.parentId === null ? { ...g, parentId: "dept-root" } : g
            )
          : ogGroupsRaw;
        return [...withoutOg, ...ogGroups];
      });
      setOgSynced(true);
      setIsRefreshingOg(false);
      toast.success(`已加载 ${ogGroupsRaw.length} 个用户组`);
    }, 800);
  };

  // 刷新同步（模拟：检测到腾讯统一身份管理平台删除了某个组织架构）
  const handleRefreshSync = useCallback(() => {
    if (!deptSynced) return;
    setIsSyncingDepts(true);
    setTimeout(() => {
      // 模拟从 OneID 获取最新组织架构（不包含已删除的运营组及其子组织）
      const deletedIds = new Set(["dept-operation", "dept-operation-1", "dept-operation-2"]);
      const latestNodes = ONEID_ALL_DEPT_NODES.filter(
        (n) => !deletedIds.has(n.id)
      );
      const deptGroups: UserGroup[] = latestNodes.map((n) => ({
        id: n.id,
        name: n.name,
        parentId: n.parentId,
        source: "oneid-dept" as const,
        readonly: true,
        externalId: n.id,
        syncBatchId: "oneid-org",
        createdAt: new Date().toISOString(),
      }));

      // 但这些异常组织仍保留在树中（因为有配置绑定，无法删除）
      const anomalousNodes: UserGroup[] = [
        {
          id: "dept-operation",
          name: "运营组",
          parentId: "dept-product",
          source: "oneid-dept",
          readonly: true,
          externalId: "dept-operation",
          syncBatchId: "oneid-org",
          createdAt: new Date().toISOString(),
        },
        {
          id: "dept-operation-1",
          name: "运营一组",
          parentId: "dept-operation",
          source: "oneid-dept",
          readonly: true,
          externalId: "dept-operation-1",
          syncBatchId: "oneid-org",
          createdAt: new Date().toISOString(),
        },
        {
          id: "dept-operation-2",
          name: "运营二组",
          parentId: "dept-operation",
          source: "oneid-dept",
          readonly: true,
          externalId: "dept-operation-2",
          syncBatchId: "oneid-org",
          createdAt: new Date().toISOString(),
        },
      ];

      setGroups((prev) => {
        const withoutDept = prev.filter((g) => g.source !== "oneid-dept");
        return [...withoutDept, ...deptGroups, ...anomalousNodes];
      });

      // 模拟：组织架构被删除后，用户从这些组织中被移除（groupIds 去掉运营相关 id）
      users.forEach((u, idx) => {
        if (u.groupIds.some((gid) => deletedIds.has(gid))) {
          users[idx] = {
            ...u,
            groupIds: u.groupIds.filter((gid) => !deletedIds.has(gid)),
          };
        }
      });

      // 设置异常组织
      setAnomalousGroups(MOCK_SYNC_RESULT.anomalousGroups);

      // 通知父组件弹出同步结果弹窗
      if (onShowSyncResult) {
        onShowSyncResult(MOCK_SYNC_RESULT.anomalousGroups);
      }

      setIsSyncingDepts(false);
    }, 1200);
  }, [deptSynced, onShowSyncResult, users]);

  // 模拟解绑配置：移除异常组织中的指定配置
  const handleUnbindConfig = useCallback((groupId: string, configName: string) => {
    setAnomalousGroups((prev) => {
      const updated = prev.map((ag) => {
        if (ag.groupId !== groupId) return ag;
        const newConfigs = ag.boundConfigs.filter((c) => c !== configName);
        return { ...ag, boundConfigs: newConfigs };
      }).filter((ag) => ag.boundConfigs.length > 0);
      return updated;
    });
    // 如果全部解绑，移除该组织
    setAnomalousGroups((prev) => {
      if (prev.length === 0) {
        // 从树中移除异常组织
        setGroups((g) => g.filter((grp) => grp.id !== groupId));
      }
      return prev;
    });
  }, []);

  // ─── 组织 CRUD（普通模式） ────────────────────────────────
  type FormMode = "create" | "edit" | "addChild";
  const [formOpen, setFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<FormMode>("create");
  const [formTarget, setFormTarget] = useState<{
    id: string;
    name: string;
    parentId: string | null;
  } | null>(null);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);

  // 编辑组织存量 Agent 实例处理弹窗（v3.0：按新组织路径聚合，随组织自动迁移）
  const [editGroupAgentDialog, setEditGroupAgentDialog] = useState<{
    open: boolean;
    parentMigration: ParentMigration;
    pendingAction: () => void;
  } | null>(null);
  const [configDiffDialog, setConfigDiffDialog] = useState<{ open: boolean; fromGroupName: string; toGroupName: string; instances: InstanceConfigCompare[] } | null>(null);

  // 新建组织
  const handleCreateGroup = () => {
    setFormMode("create");
    setFormTarget(null);
    setFormOpen(true);
  };

  // 添加子组织
  const handleAddChildGroup = (parentId: string) => {
    const parent = groups.find((g) => g.id === parentId);
    if (!parent) return;
    setFormMode("addChild");
    setFormTarget({ id: parent.id, name: parent.name, parentId: parent.parentId });
    setFormOpen(true);
  };

  // 编辑组织
  const handleEditGroup = (groupId: string) => {
    const g = groups.find((g) => g.id === groupId);
    if (!g) return;
    setFormMode("edit");
    setFormTarget({ id: g.id, name: g.name, parentId: g.parentId });
    setFormOpen(true);
  };

  // 确认新建/编辑/添加子组织
  const handleFormConfirm = (name: string, parentId: string | null) => {
    // OneID 模式已同步部门时，A公司（dept-root）是唯一顶层组织。
    // 顶部"新建组织"对话框默认 parentId=null，此处兜底改写为 "dept-root"，
    // 使新建顶层组织也归到 A公司 下，与"A公司唯一顶层"语义一致。
    if (
      (formMode === "create" || formMode === "addChild") &&
      hasOneid &&
      deptSynced &&
      parentId === null
    ) {
      parentId = "dept-root";
    }

    if (formMode === "edit" && formTarget) {
      const parentChanged = formTarget.parentId !== parentId;

      const doEdit = () => {
        setGroups((prev) =>
          prev.map((g) =>
            g.id === formTarget.id ? { ...g, name, parentId } : g
          )
        );
        toast.success("组织已更新");
      };

      if (parentChanged) {
        // 检测该组织下所有用户的 Agent 实例
        const groupId = formTarget.id;
        const groupUsers = effectiveUsers.filter((u) => u.groupIds.includes(groupId));
        // 新上级组织路径
        const newParentName = parentId
          ? getPrimaryDeptPath(parentId, groups) + " / " + formTarget.name
          : formTarget.name;
        // 变更说明用：旧/新上级组织路径
        const oldParentPath = formTarget.parentId
          ? getPrimaryDeptPath(formTarget.parentId, groups)
          : "无";
        const newParentOnlyPath = parentId ? getPrimaryDeptPath(parentId, groups) : "无";

        const migrationRows: ParentMigration["groups"][number]["rows"] = [];
        groupUsers.forEach((u) => {
          const userAgents = MOCK_USER_GROUP_AGENTS[u.userId];
          const instances = userAgents?.[groupId];
          if (instances && instances.length > 0) {
            instances.forEach((inst) => {
              migrationRows.push({ userId: u.userId, instanceId: inst.id, instanceName: inst.name });
            });
          }
        });

        if (migrationRows.length > 0) {
          setEditGroupAgentDialog({
            open: true,
            parentMigration: {
              note: `「${formTarget.name}」的上级组织从「${oldParentPath}」变更为「${newParentOnlyPath}」，该组织及其所有子组织的 Agent 实例将自动迁移到对应新路径。`,
              groups: [
                {
                  targetPath: newParentName,
                  fromGroupId: groupId,
                  toGroupId: groupId,
                  rows: migrationRows,
                },
              ],
            },
            pendingAction: doEdit,
          });
          return;
        }
      }

      doEdit();
    } else {
      // 新建 / 添加子组织
      // 来源判定：
      //   - 父节点是 oneid-dept（包括 A公司 根节点 dept-root）或 oneid-group → 新建为 oneid-group（OneID 自定义组织）
      //   - 父节点为空（顶层新建）或父是 manual → 新建为 manual
      const parentNode = parentId ? groups.find((g) => g.id === parentId) : null;
      const inheritOneidGroup =
        parentNode?.source === "oneid-dept" || parentNode?.source === "oneid-group";
      const newGroup: UserGroup = inheritOneidGroup
        ? {
            id: `og-manual-${Date.now()}`,
            name,
            parentId,
            source: "oneid-group",
            readonly: false,
            syncBatchId: "oneid-org",
            createdAt: new Date().toISOString(),
          }
        : {
            id: `manual-${Date.now()}`,
            name,
            parentId,
            source: "manual",
            readonly: false,
            createdAt: new Date().toISOString(),
          };
      setGroups((prev) => [...prev, newGroup]);
      setSelectedId(newGroup.id);
      toast.success("组织已创建");
    }
    setFormOpen(false);
  };

  // 删除组织
  const handleOpenDelete = (groupId: string) => {
    const g = groups.find((g) => g.id === groupId);
    if (!g) return;
    setDeleteTarget({ id: g.id, name: g.name });
    setDeleteOpen(true);
  };

  const handleDeleteConfirm = (groupId: string) => {
    // 删除该组织及其所有子孙
    const toDelete = new Set<string>();
    const addDescendants = (id: string) => {
      toDelete.add(id);
      groups.filter((g) => g.parentId === id).forEach((g) => addDescendants(g.id));
    };
    addDescendants(groupId);

    setGroups((prev) => prev.filter((g) => !toDelete.has(g.id)));
    setDeleteOpen(false);
    setDeleteTarget(null);
    if (toDelete.has(selectedId)) {
      const remaining = groups.filter((g) => !toDelete.has(g.id));
      setSelectedId(remaining.length > 0 ? remaining[0].id : UNASSIGNED_GROUP_ID);
    }
    toast.success("组织已删除，用户保留");
  };

  // 删除时的成员数统计
  const deleteMemberCount = useMemo(() => {
    if (!deleteTarget) return 0;
    return getUsersOfGroupDeep(deleteTarget.id, groups, effectiveUsers).length;
  }, [deleteTarget, groups, effectiveUsers]);

  // ─── 用户操作（普通模式） ────────────────────────────────
  const [, setUsersVersion] = useState(0);

  // 添加用户到组织
  const handleAddUsersToGroup = useCallback(
    (userIds: string[]) => {
      if (!selectedId || selectedId === UNASSIGNED_GROUP_ID) return;
      // 将 selectedId 加入这些用户的 groupIds
      const usersToUpdate = new Set(userIds);
      const updatedUsers = effectiveUsers.map((u) => {
        if (usersToUpdate.has(u.userId) && !u.groupIds.includes(selectedId)) {
          return { ...u, groupIds: [...u.groupIds, selectedId] };
        }
        return u;
      });
      // 因为 MOCK_USERS_MANUAL 是引用，直接更新对象属性来模拟后端操作
      updatedUsers.forEach((u, idx) => {
        if (usersToUpdate.has(u.userId)) {
          effectiveUsers[idx] = u;
        }
      });
      setUsersVersion((v) => v + 1);
      toast.success(`已添加 ${userIds.length} 名用户到组织`);
    },
    [selectedId, effectiveUsers]
  );

  // 从组织中移除用户
  const handleRemoveFromGroup = useCallback(
    (userId: string) => {
      if (!selectedId || selectedId === UNASSIGNED_GROUP_ID) return;
      const idx = effectiveUsers.findIndex((u) => u.userId === userId);
      if (idx >= 0) {
        effectiveUsers[idx] = {
          ...effectiveUsers[idx],
          groupIds: effectiveUsers[idx].groupIds.filter(
            (gid) => gid !== selectedId
          ),
        };
        setUsersVersion((v) => v + 1);
        toast.success("已从组织中移除");
      }
    },
    [selectedId, effectiveUsers]
  );

  const handleEditUserGroups = useCallback(
    (userId: string, newGroupIds: string[]) => {
      const idx = effectiveUsers.findIndex((u) => u.userId === userId);
      if (idx >= 0) {
        effectiveUsers[idx] = {
          ...effectiveUsers[idx],
          groupIds: newGroupIds,
        };
        setUsersVersion((v) => v + 1);
        toast.success("用户组织已更新");
      }
    },
    [effectiveUsers]
  );

  return (
    <div className="space-y-3">
      {/* 常驻组织命名提醒 */}
      {groups.length > 0 && (
        <Alert variant="operation-info" className="items-center px-4 py-2.5">
          <AlertOperationInfoIcon className="self-center !translate-y-0" />
          <AlertDescription className="min-h-0 text-xs leading-[18px]">
            组织名称将在用户端展示，用户可查看自己所属的组织。请确保组织命名规范、清晰，避免使用内部代号或敏感信息。
          </AlertDescription>
        </Alert>
      )}

      {/* 主体：合并为一个卡片，左右面板 + 可拖拽分割线 */}
      <SurfaceCard
        className="flex overflow-hidden p-0"
        style={{
          height: "calc(100vh - 220px)",
        }}
      >
        {/* 左侧面板 */}
        {!leftCollapsed && (
          <div
            className="shrink-0 relative"
            style={{ width: leftWidth }}
          >
            <div className="h-full overflow-hidden">
              <GroupList
                groups={groups}
                users={effectiveUsers}
                selectedId={selectedId}
                onSelect={setSelectedId}
                deptSynced={hasOneid && hasDeptData ? deptSynced : undefined}
                onSyncDepts={handleSyncDepts}
                isSyncingDepts={isSyncingDepts}
                hasOneid={hasOneid}
                isManualMode={true}
                onCreateGroup={handleCreateGroup}
                onAddChildGroup={handleAddChildGroup}
                onEditGroup={handleEditGroup}
                onDeleteGroup={handleOpenDelete}
                anomalousGroupIds={anomalousGroupIds}
                directAnomalousGroupIds={directAnomalousGroupIds}
                onRefreshSync={handleRefreshSync}
                uninitializedGroupIds={uninitializedGroupIds}
                directUninitializedGroupIds={directUninitializedGroupIds}
                networkOutdatedGroupIds={networkOutdatedGroupIds}
                anomalousGroupDetails={anomalousGroupDetails}
              />
            </div>
            {/* 收起按钮 —— 右边缘贴住分割线竖线 */}
            <button
              type="button"
              onClick={() => setLeftCollapsed(true)}
              className="absolute top-[18px] -right-2 w-6 h-7 flex items-center justify-center rounded-l-[4px] rounded-r-none bg-[var(--bg-grey-hover-subtle)] text-[var(--text-weak)] hover:bg-[var(--bg-grey-hover)] hover:text-[var(--text-muted)] transition-colors z-10"
              title="收起组织列表"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* 可拖拽分割线 */}
        {!leftCollapsed && (
          <div className="shrink-0 flex flex-col items-center relative w-4 z-20">
            {/* 中间竖线 + 拖拽手柄 */}
            <div
              className="flex-1 flex flex-col items-center justify-center cursor-col-resize group relative w-full"
              onMouseDown={handleMouseDown}
            >
              {/* 上段竖线 */}
              <div className="flex-1 w-px bg-[var(--cp-border,#EAEEF4)]" />
              {/* 拖拽手柄：圆角矩形 + 2×3 六点阵列 */}
              <div className="w-3 py-1.5 flex flex-col items-center justify-center gap-[2px] rounded-full bg-[var(--bg-grey-hover-subtle)] group-hover:bg-[var(--bg-grey-hover)] transition-colors">
                <div className="flex gap-[2px]">
                  <span className="w-[1.5px] h-[1.5px] rounded-full bg-[var(--text-weak)] transition-colors" />
                  <span className="w-[1.5px] h-[1.5px] rounded-full bg-[var(--text-weak)] transition-colors" />
                </div>
                <div className="flex gap-[2px]">
                  <span className="w-[1.5px] h-[1.5px] rounded-full bg-[var(--text-weak)] transition-colors" />
                  <span className="w-[1.5px] h-[1.5px] rounded-full bg-[var(--text-weak)] transition-colors" />
                </div>
                <div className="flex gap-[2px]">
                  <span className="w-[1.5px] h-[1.5px] rounded-full bg-[var(--text-weak)] transition-colors" />
                  <span className="w-[1.5px] h-[1.5px] rounded-full bg-[var(--text-weak)] transition-colors" />
                </div>
              </div>
              {/* 下段竖线 */}
              <div className="flex-1 w-px bg-[var(--cp-border,#EAEEF4)]" />
              {/* 扩大拖拽热区 */}
              <div className="absolute inset-y-0 -left-1.5 -right-1.5" />
            </div>
          </div>
        )}

        {/* 右侧面板 */}
        <div className="flex-1 min-w-0 overflow-hidden flex flex-col relative">
          {/* 折叠态：展开按钮 —— 贴在左侧边缘，圆角方形灰底 */}
          {leftCollapsed && (
            <div className="absolute left-0 top-3.5 z-10">
              <button
                type="button"
                onClick={() => setLeftCollapsed(false)}
                className="w-6 h-6 flex items-center justify-center rounded-r-[4px] bg-[var(--bg-grey-hover)] text-[var(--text-weak)] hover:bg-[var(--cp-border,#EAEEF4)] hover:text-[var(--text-muted)] transition-colors border border-l-0 border-[var(--cp-border,#EAEEF4)]"
                title="展开组织列表"
              >
                <ChevronRight className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
          {selectedId === UNASSIGNED_GROUP_ID ? (
            <NodeContentPanel
              nodeId={UNASSIGNED_GROUP_ID}
              nodeName="未分配组织"
              nodeSource="manual"
              nodeReadonly={false}
              groups={groups}
              nodePath="未分配组织"
              users={groupUsers}
              hasOneid={hasOneid}
              hasDeptData={hasDeptData}
              isManualMode={!hasOneid}
            />
          ) : selectedGroup ? (
            <NodeContentPanel
              nodeId={selectedGroup.id}
              nodeName={selectedGroup.name}
              nodeSource={selectedGroup.source as "oneid-dept" | "oneid-group" | "manual"}
              nodeReadonly={selectedGroup.readonly}
              groups={groups}
              nodePath={selectedNode?.path ?? selectedGroup.name}
              users={groupUsers}
              hasOneid={hasOneid}
              hasDeptData={hasDeptData}
              isManualMode={!selectedGroup.readonly}
              allUsers={effectiveUsers}
              onAddUsersToGroup={handleAddUsersToGroup}
              onRemoveFromGroup={handleRemoveFromGroup}
              onEditUserGroups={handleEditUserGroups}
              isAnomalous={anomalousGroups.some((ag) => ag.groupId === selectedGroup.id)}
              anomalousBoundConfigs={anomalousGroups.find((ag) => ag.groupId === selectedGroup.id)?.boundConfigs}
              isUninitialized={directUninitializedGroupIds.has(selectedGroup.id)}
            />
          ) : (
            <div className="flex items-center justify-center h-full text-[var(--text-weak)] text-sm">
              请选择组织
            </div>
          )}
        </div>
      </SurfaceCard>

      {/* 新建 / 编辑 / 添加子组织弹窗 */}
      <GroupFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        groups={groups}
        mode={formMode}
        target={formTarget}
        onConfirm={handleFormConfirm}
      />

      {/* 删除组织确认弹窗 */}
      <DeleteGroupDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        group={deleteTarget}
        memberCount={deleteMemberCount}
        groups={groups}
        onConfirm={handleDeleteConfirm}
      />

      {/* 编辑组织存量 Agent 实例处理弹窗（v3.0：随组织自动迁移，按新路径聚合） */}
      <AgentInstanceHandlingDialog
        open={!!editGroupAgentDialog?.open}
        onOpenChange={(open) => { if (!open) setEditGroupAgentDialog(null); }}
        scenario="editParent"
        parentMigration={editGroupAgentDialog?.parentMigration}
        onConfirm={() => {
          editGroupAgentDialog?.pendingAction();
          setEditGroupAgentDialog(null);
          toast.success("存量实例已迁移至新组织");
        }}
        onViewDiff={(fromGroupId, toGroupId, instances) => {
          const fromName = getPrimaryDeptPath(fromGroupId, groups);
          const toName = toGroupId ? getPrimaryDeptPath(toGroupId, groups) : "";
          setConfigDiffDialog({
            open: true,
            fromGroupName: fromName,
            toGroupName: toName,
            instances: buildMockInstanceCompare(
              (instances ?? []).map((i: AgentInstance) => ({ instanceName: i.name, instanceId: i.id }))
            ),
          });
        }}
      />
      <ConfigDiffDialog
        open={!!configDiffDialog?.open}
        onOpenChange={(open) => { if (!open) setConfigDiffDialog(null); }}
        newGroupName={configDiffDialog?.toGroupName ?? ""}
        instances={configDiffDialog?.instances ?? []}
      />

    </div>
  );
}
