/**
 * ProjectAssetManagement - 「项目资产管理」主页面
 * 左侧：组织/项目树（数据来自跨页共享 groupStore，与用户管理组织视图双向同步）；
 * 右侧：该组织/项目的资产配置 + 成员 + 实例。
 * 从组织/项目视角配置其 Agent 资产合集（公共技能/企业技能/企业插件/企业MCP/企业规范）。
 *
 * 组织 / 项目的新建 / 编辑 / 添加子级 / 删除，复用「用户管理-组织视图」的
 * GroupFormDialog / DeleteGroupDialog（项目部分通过 term="项目" 切换文案）。
 */
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { Boxes } from 'lucide-react';
import { Empty, EmptyHeader, EmptyMedia, EmptyDescription } from '@/components/ui/empty';
import { AdminPageHeader } from '@/components/ui/admin-page-header';
import type { UserGroup, UserOrg } from '../MemberManagement/types';
import { groupStore, removeGroupSubtree } from '../MemberManagement/groupStore';
import { userStore, addUsersToGroup, removeUserFromGroup } from '../MemberManagement/userStore';
import { GroupFormDialog, DeleteGroupDialog } from '../MemberManagement/GroupDialog';
import { getUsersOfGroupDeep } from '../MemberManagement/health';
import { tenantProjectStore } from '../../tenant/tenantProjectStore';
import GroupTreePanel from './GroupTreePanel';
import AssetPanel from './AssetPanel';
import ProjectFormDialog, {
  type ProjectFormTarget,
  type ProjectFormValues,
} from './ProjectFormDialog';

type FormKind = 'org' | 'project';
type FormState = {
  mode: 'create' | 'edit' | 'addChild';
  kind: FormKind;
  target: ({ id: string; name: string; parentId: string | null } & Partial<ProjectFormValues>) | null;
};

export default function ProjectAssetManagement() {
  const [groups, setGroups] = useState<UserGroup[]>(() => groupStore.getAll());
  const [users, setUsers] = useState<UserOrg[]>(() => userStore.getAll());
  const [selectedId, setSelectedId] = useState<string | null>(() => {
    const all = groupStore.getAll();
    return all.find((g) => g.source !== 'project')?.id ?? all[0]?.id ?? null;
  });
  // Tab 状态提升到父层：AssetPanel 带 key 会随节点切换整体重建，
  // 若 tab 存在子组件内则每次切换都会被重置。放到父层即可跨节点保持。
  const [panelTab, setPanelTab] = useState<'assets' | 'instances'>('assets');

  const [formState, setFormState] = useState<FormState | null>(null);
  const [deleteState, setDeleteState] = useState<{ kind: FormKind; group: { id: string; name: string } } | null>(null);

  // 订阅共享 store：用户管理页对组织/项目/成员的变更实时反映到本页
  useEffect(() => groupStore.subscribe(() => setGroups(groupStore.getAll())), []);
  useEffect(() => userStore.subscribe(() => setUsers(userStore.getAll())), []);

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedId) ?? null,
    [groups, selectedId],
  );

  const orgGroups = useMemo(() => groups.filter((g) => g.source !== 'project'), [groups]);
  const projectGroups = useMemo(() => groups.filter((g) => g.source === 'project'), [groups]);

  const handleAddUsers = (userIds: string[]) => {
    if (!selectedId) return;
    addUsersToGroup(userIds, selectedId);
  };

  // 刷新左侧组织 / 项目树：重新从共享 store 拉取最新数据
  const handleRefresh = () => {
    setGroups(groupStore.getAll());
    setUsers(userStore.getAll());
    toast.success('已刷新');
  };

  const handleRemoveUser = (userId: string) => {
    if (!selectedId) return;
    removeUserFromGroup(userId, selectedId);
  };

  // ─── 组织 / 项目 CRUD 弹窗触发 ──────────────────────────
  const openCreateOrg = () => setFormState({ mode: 'create', kind: 'org', target: null });
  const openCreateProject = () => setFormState({ mode: 'create', kind: 'project', target: null });
  const openCreateChild = (parentId: string) => {
    const parent = groups.find((g) => g.id === parentId);
    setFormState({
      mode: 'addChild',
      kind: parent?.source === 'project' ? 'project' : 'org',
      target: parent
        ? { id: parent.id, name: parent.name, parentId: parent.parentId }
        : null,
    });
  };
  const openEdit = (targetId: string) => {
    const g = groups.find((x) => x.id === targetId);
    if (!g) return;
    const sharedProject = g.source === 'project' ? tenantProjectStore.getById(g.id) : undefined;
    setFormState({
      mode: 'edit',
      kind: g.source === 'project' ? 'project' : 'org',
      target: {
        id: g.id,
        name: g.name,
        parentId: g.parentId,
        description: sharedProject?.description ?? g.description ?? '',
        goal: sharedProject?.goal ?? g.goal ?? '',
        allowMemberEdit: sharedProject?.allowMemberEdit ?? g.allowMemberEdit ?? true,
      },
    });
  };
  const openDelete = (targetId: string) => {
    const g = groups.find((x) => x.id === targetId);
    if (!g) return;
    setDeleteState({
      kind: g.source === 'project' ? 'project' : 'org',
      group: { id: g.id, name: g.name },
    });
  };

  const handleOrgFormConfirm = (name: string, parentId: string | null) => {
    if (!formState) return;
    if (formState.mode === 'edit' && formState.target) {
      groupStore.update(formState.target.id, (prev) => ({ ...prev, name, parentId }));
      toast.success('组织已更新');
    } else {
      const newGroup: UserGroup = {
        id: `mgrp-${Date.now()}`,
        name,
        parentId: formState.mode === 'addChild' ? formState.target?.id ?? null : parentId,
        source: 'manual',
        readonly: false,
        createdAt: new Date().toISOString().slice(0, 10),
      };
      groupStore.add(newGroup);
      setSelectedId(newGroup.id);
      toast.success(`组织「${name}」已创建`);
    }
    setFormState(null);
  };

  const handleProjectFormConfirm = (values: ProjectFormValues) => {
    if (!formState || formState.kind !== 'project') return;

    if (formState.mode === 'edit' && formState.target) {
      const projectId = formState.target.id;
      groupStore.update(projectId, (prev) => ({
        ...prev,
        name: values.name,
        description: values.description,
        goal: values.goal,
        allowMemberEdit: values.allowMemberEdit,
      }));
      if (tenantProjectStore.getById(projectId)) {
        tenantProjectStore.updateProjectInfo(projectId, {
          name: values.name,
          description: values.description,
          goal: values.goal,
        });
        tenantProjectStore.toggleAllowMemberEdit(projectId, values.allowMemberEdit);
      }
      toast.success('项目已更新');
    } else {
      const projectId = tenantProjectStore.createProject(values);
      const sharedProject = tenantProjectStore.getById(projectId);
      groupStore.add({
        id: projectId,
        name: values.name,
        description: values.description,
        goal: values.goal,
        allowMemberEdit: values.allowMemberEdit,
        parentId: null,
        source: 'project',
        readonly: false,
        createdAt: new Date().toISOString().slice(0, 10),
      });
      if (sharedProject) {
        addUsersToGroup(sharedProject.members.map((member) => member.userId), projectId);
      }
      setSelectedId(projectId);
      toast.success(`项目「${values.name}」已创建`);
    }
    setFormState(null);
  };

  const handleDeleteConfirm = (groupId: string) => {
    const isProject = deleteState?.kind === 'project';
    const name = deleteState?.group.name ?? '';
    removeGroupSubtree(groupId);
    if (isProject && tenantProjectStore.getById(groupId)) {
      tenantProjectStore.deleteProject(groupId);
    }
    if (selectedId === groupId) setSelectedId(null);
    toast.success(`${isProject ? '项目' : '组织'}「${name}」已删除`);
    setDeleteState(null);
  };

  const deleteTerm = deleteState?.kind === 'project' ? '项目' : '组织';
  const deleteGroups = deleteState?.kind === 'project' ? projectGroups : orgGroups;
  const deleteMemberCount = useMemo(
    () => (deleteState ? getUsersOfGroupDeep(deleteState.group.id, deleteGroups, users).length : 0),
    [deleteState, deleteGroups, users],
  );

  return (
    <div className="page-enter flex flex-col h-full min-h-0">
      <AdminPageHeader
        title="资产管理"
        description="以组织 / 项目为单位设定一套 Agent 配置资产，让该范围内的所有实例统一批量下发、保持一致。"
      />
      <div className="flex flex-1 min-h-0 overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)]">
        {/* 左侧组织/项目树 */}
        <div className="w-[300px] shrink-0 border-r border-[var(--cp-border)] bg-[var(--cp-surface)]">
          <GroupTreePanel
            groups={groups}
            users={users}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onRefresh={handleRefresh}
            onCreateOrg={openCreateOrg}
            onCreateProject={openCreateProject}
            onCreateChild={openCreateChild}
            onRename={openEdit}
            onDelete={openDelete}
          />
        </div>

        {/* 右侧资产面板 */}
        <div className="flex-1 min-w-0">
          {selectedGroup ? (
            <AssetPanel
              key={selectedGroup.id}
              groupId={selectedGroup.id}
              groupName={selectedGroup.name}
              groups={groups}
              users={users}
              tab={panelTab}
              onTabChange={setPanelTab}
              onAddUsers={handleAddUsers}
              onRemoveUser={handleRemoveUser}
              onEditProject={selectedGroup.source === 'project' ? () => openEdit(selectedGroup.id) : undefined}
            />
          ) : (
            <div className="flex items-center justify-center h-full">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="default">
                    <Boxes className="w-6 h-6" />
                  </EmptyMedia>
                  <EmptyDescription>请从左侧选择一个组织或项目以配置其资产</EmptyDescription>
                </EmptyHeader>
              </Empty>
            </div>
          )}
        </div>
      </div>

      {/* 组织新建 / 编辑 / 添加子级弹窗 */}
      <GroupFormDialog
        open={!!formState && formState.kind === 'org'}
        onOpenChange={(o) => !o && setFormState(null)}
        groups={orgGroups}
        mode={formState?.mode ?? 'create'}
        target={formState?.target ?? null}
        onConfirm={handleOrgFormConfirm}
        term="组织"
      />

      {/* 项目新建 / 编辑字段与用户端保持一致 */}
      <ProjectFormDialog
        open={!!formState && formState.kind === 'project'}
        onOpenChange={(o) => !o && setFormState(null)}
        mode={formState?.mode === 'edit' ? 'edit' : 'create'}
        target={formState?.kind === 'project' ? formState.target as ProjectFormTarget | null : null}
        projects={projectGroups}
        onConfirm={handleProjectFormConfirm}
      />

      {/* 组织 / 项目 删除确认弹窗（复用用户管理组织视图组件） */}
      <DeleteGroupDialog
        open={!!deleteState}
        onOpenChange={(o) => !o && setDeleteState(null)}
        group={deleteState?.group ?? null}
        memberCount={deleteMemberCount}
        groups={deleteGroups}
        onConfirm={handleDeleteConfirm}
        term={deleteTerm}
        checkAgentInstances={deleteState?.kind !== 'project'}
      />
    </div>
  );
}
