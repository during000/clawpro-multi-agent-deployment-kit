/**
 * 项目视图容器
 *
 * 「用户管理」页的独立 Tab，从组织视图中剥离出来，仅管理项目（source='project'）。
 * 项目为单层级结构（不支持子项目），与「项目资产管理」页共享同一个 groupStore / userStore，
 * 项目的新建 / 编辑 / 删除 / 成员增删双向同步。
 *
 * 布局：左右合为一个大卡片。
 *   - 左面板：标题「项目」+「新建」按钮 + 搜索框；下方单层级项目列表
 *   - 右面板：项目详情 / 成员表格（NodeContentPanel，isProject）
 */
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Plus, Search, Pencil, Trash2, MoreHorizontal, RefreshCw, Loader2 } from "lucide-react";
import { Alert, AlertDescription, AlertOperationInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { SurfaceCard } from "@/components/ui/Surface";
import { MoreActionsDropdown } from "@/components/ui/more-actions-dropdown";
import NodeContentPanel from "./NodeContentPanel";
import { GroupFormDialog, DeleteGroupDialog } from "./GroupDialog";
import type { UserGroup, UserOrg } from "./types";
import { getUsersOfGroupDeep } from "./health";
import { groupStore, removeGroupSubtree } from "./groupStore";
import { userStore, addUsersToGroup, removeUserFromGroup } from "./userStore";

type FormMode = "create" | "edit";

export default function ProjectView() {
  // 项目（source='project'）：与「项目资产管理」页共享 groupStore，双向同步
  const [projects, setProjects] = useState<UserGroup[]>(
    () => groupStore.getAll().filter((g) => g.source === "project"),
  );
  // 项目成员来自共享 userStore（与项目资产页一致，成员增删双向同步）
  const [projectUsers, setProjectUsers] = useState<UserOrg[]>(() => userStore.getAll());
  useEffect(
    () => groupStore.subscribe(() => setProjects(groupStore.getAll().filter((g) => g.source === "project"))),
    [],
  );
  useEffect(() => userStore.subscribe(() => setProjectUsers(userStore.getAll())), []);

  const [keyword, setKeyword] = useState("");
  const [selectedId, setSelectedId] = useState<string>(() => projects[0]?.id ?? "");
  const [refreshing, setRefreshing] = useState(false);

  // 刷新项目列表（从共享 groupStore 重新拉取，模拟同步延迟）
  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    setTimeout(() => {
      setProjects(groupStore.getAll().filter((g) => g.source === "project"));
      setRefreshing(false);
      toast.success("项目列表已刷新");
    }, 600);
  }, []);

  // 选中项回退保护
  useEffect(() => {
    if (!projects.find((p) => p.id === selectedId)) {
      setSelectedId(projects[0]?.id ?? "");
    }
  }, [projects, selectedId]);

  const filteredProjects = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return projects;
    return projects.filter((p) => p.name.toLowerCase().includes(kw));
  }, [projects, keyword]);

  const memberCount = useCallback(
    (projectId: string) => getUsersOfGroupDeep(projectId, projects, projectUsers).length,
    [projects, projectUsers],
  );

  // ─── 项目 CRUD（写入共享 groupStore；单层级，无子项目） ───────────────
  const [formOpen, setFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<FormMode>("create");
  const [formTarget, setFormTarget] = useState<{ id: string; name: string; parentId: string | null } | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);

  const openCreate = () => {
    setFormMode("create");
    setFormTarget(null);
    setFormOpen(true);
  };
  const openEdit = (id: string) => {
    const p = projects.find((x) => x.id === id);
    if (!p) return;
    setFormMode("edit");
    setFormTarget({ id: p.id, name: p.name, parentId: p.parentId });
    setFormOpen(true);
  };
  const openDelete = (id: string) => {
    const p = projects.find((x) => x.id === id);
    if (!p) return;
    setDeleteTarget({ id: p.id, name: p.name });
    setDeleteOpen(true);
  };

  const handleFormConfirm = (name: string) => {
    if (formMode === "edit" && formTarget) {
      groupStore.update(formTarget.id, (prev) => ({ ...prev, name }));
      toast.success("项目已更新");
    } else {
      const newProject: UserGroup = {
        id: `proj-${Date.now()}`,
        name,
        parentId: null,
        source: "project",
        readonly: false,
        createdAt: new Date().toISOString(),
      };
      groupStore.add(newProject);
      setSelectedId(newProject.id);
      toast.success("项目已创建");
    }
    setFormOpen(false);
  };

  const handleDeleteConfirm = (id: string) => {
    removeGroupSubtree(id);
    setDeleteOpen(false);
    setDeleteTarget(null);
    if (selectedId === id) {
      const remaining = projects.filter((p) => p.id !== id);
      setSelectedId(remaining[0]?.id ?? "");
    }
    toast.success("项目已删除，用户保留");
  };

  // ─── 成员操作 ─────────────────────────────────────────────
  const handleAddUsers = useCallback(
    (userIds: string[]) => {
      if (!selectedId) return;
      addUsersToGroup(userIds, selectedId);
      toast.success(`已添加 ${userIds.length} 名用户到项目`);
    },
    [selectedId],
  );
  const handleRemoveUser = useCallback(
    (userId: string) => {
      if (!selectedId) return;
      removeUserFromGroup(userId, selectedId);
      toast.success("已从项目中移除");
    },
    [selectedId],
  );

  const selectedProject = projects.find((p) => p.id === selectedId) ?? null;
  const selectedMembers = useMemo(
    () => (selectedProject ? getUsersOfGroupDeep(selectedProject.id, projects, projectUsers) : []),
    [selectedProject, projects, projectUsers],
  );
  const deleteMemberCount = useMemo(
    () => (deleteTarget ? getUsersOfGroupDeep(deleteTarget.id, projects, projectUsers).length : 0),
    [deleteTarget, projects, projectUsers],
  );

  return (
    <div className="space-y-3">
      {/* 常驻项目命名提醒 */}
      {projects.length > 0 && (
        <Alert variant="operation-info" className="items-center px-4 py-2.5">
          <AlertOperationInfoIcon className="self-center !translate-y-0" />
          <AlertDescription className="min-h-0 text-xs leading-[18px]">
            项目名称将在用户端展示，用户可查看自己所属的项目。请确保项目命名规范、清晰，避免使用内部代号或敏感信息。
          </AlertDescription>
        </Alert>
      )}

      <SurfaceCard
        className="flex overflow-hidden p-0"
        style={{ height: "calc(100vh - 220px)" }}
      >
        {/* 左侧：项目列表 */}
        <div className="w-[288px] shrink-0 border-r border-[var(--cp-border,#EAEEF4)] flex flex-col h-full">
          {/* 标题 + 新建 */}
          <div className="flex items-center gap-2 px-4 pt-4 pb-2">
            <h3 className="text-lg font-semibold text-[var(--text-title)] m-0">项目</h3>
            <Button
              variant="ghost"
              size="sm"
              className="gap-1 px-2.5 h-7 font-medium"
              onClick={openCreate}
            >
              <Plus className="w-3.5 h-3.5" />
              新建
            </Button>
          </div>

          {/* 搜索 + 刷新 */}
          <div className="px-3 pb-2">
            <div className="flex items-center gap-2">
              <div className="relative flex-1 min-w-0">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                <Input
                  type="text"
                  placeholder="搜索项目..."
                  className="h-8 pl-8 pr-3 text-xs"
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                />
              </div>
              <Button
                variant="claw-outline"
                size="icon-sm"
                onClick={handleRefresh}
                disabled={refreshing}
                title="刷新"
              >
                {refreshing ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <RefreshCw className="w-3.5 h-3.5" />
                )}
              </Button>
            </div>
          </div>

          {/* 列表 */}
          <div className="flex-1 overflow-y-auto pb-3">
            {filteredProjects.length > 0 ? (
              filteredProjects.map((p) => {
                const isActive = selectedId === p.id;
                return (
                  <div
                    key={p.id}
                    className={`group flex items-center gap-1.5 h-8 pr-3 pl-2 text-sm cursor-pointer rounded-[4px] mx-3 mb-0.5 transition-colors ${
                      isActive
                        ? "bg-[var(--bg-grey-hover)] text-[#09090b] font-medium"
                        : "text-[#09090b] hover:bg-[var(--bg-grey-hover)]"
                    }`}
                    onClick={() => setSelectedId(p.id)}
                  >
                    <span className="truncate" title={p.name}>
                      {p.name}
                    </span>
                    <span className={`text-[11px] tabular-nums shrink-0 ${isActive ? "text-[#71717a]" : "text-[#a1a1aa]"}`}>
                      ({memberCount(p.id)})
                    </span>
                    <span className="flex-1" />
                    <MoreActionsDropdown
                      trigger={
                        <button
                          type="button"
                          className={`w-5 h-5 flex items-center justify-center rounded transition-colors ${isActive ? "text-[#737373] hover:text-[#020617] hover:bg-[var(--bg-grey-hover)]" : "text-[#d4d4d4] hover:text-[#525252] hover:bg-[var(--bg-grey-hover)]"}`}
                          onClick={(e) => e.stopPropagation()}
                        >
                          <MoreHorizontal className="w-3 h-3" />
                        </button>
                      }
                      align="end"
                      stopPropagation
                      items={[
                        { label: "编辑项目", icon: Pencil, onClick: () => openEdit(p.id) },
                        { label: "删除项目", icon: Trash2, onClick: () => openDelete(p.id), variant: "destructive" },
                      ]}
                    />
                  </div>
                );
              })
            ) : (
              <div className="px-4 py-10 text-center text-xs text-[#A3A3A3]">
                {keyword.trim() ? "未找到匹配项目" : "暂无项目，可新建项目"}
              </div>
            )}
          </div>
        </div>

        {/* 右侧：项目详情 / 成员 */}
        <div className="flex-1 min-w-0 overflow-hidden flex flex-col">
          {selectedProject ? (
            <NodeContentPanel
              nodeId={selectedProject.id}
              nodeName={selectedProject.name}
              nodeSource="manual"
              nodeReadonly={false}
              groups={projects}
              nodePath={selectedProject.name}
              users={selectedMembers}
              hasOneid={false}
              hasDeptData={false}
              isManualMode={true}
              allUsers={projectUsers}
              onAddUsersToGroup={handleAddUsers}
              onRemoveFromGroup={handleRemoveUser}
              isProject={true}
            />
          ) : (
            <div className="flex items-center justify-center h-full text-[var(--text-weak)] text-sm">
              请选择项目
            </div>
          )}
        </div>
      </SurfaceCard>

      {/* 新建 / 编辑 项目弹窗（单层级：隐藏上级选择） */}
      <GroupFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        groups={projects}
        mode={formMode}
        target={formTarget}
        onConfirm={handleFormConfirm}
        term="项目"
        singleLevel
      />

      {/* 删除项目确认弹窗 */}
      <DeleteGroupDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        group={deleteTarget}
        memberCount={deleteMemberCount}
        groups={projects}
        onConfirm={handleDeleteConfirm}
        term="项目"
        checkAgentInstances={false}
      />
    </div>
  );
}
