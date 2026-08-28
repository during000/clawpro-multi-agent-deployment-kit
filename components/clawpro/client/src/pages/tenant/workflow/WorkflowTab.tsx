/**
 * WorkflowTab —— 工作流独立维护页（3 Tab 之一）
 *
 * 内容：
 *  - 列出本项目所有工作流（TenantPipelineTemplate）
 *  - 新建/编辑（用画布 WorkflowCanvas）
 *  - 删除工作流
 *  - 接入 agent 入口：从我已注册的 agent 选一个 / 接入外部 agent（装CLI 后轮询捕获）
 *  - 展示"项目可用 agent 列表"
 *
 * 数据入口：project.pipelineTemplates / project.agents / project.members
 * 保存走：tenantProjectStore.createPipelineTemplate / updatePipelineTemplate / deletePipelineTemplate
 * agent 接入走：tenantProjectStore.attachMyAgent / startPendingAgentConnection / checkPendingAgent
 */
import { useState, useEffect, useRef } from 'react';
import { toast } from 'sonner';
import {
  Plus,
  Trash2,
  Workflow as WorkflowIcon,
  Bot,
  UserRound,
  RefreshCw,
  Loader2,
  CheckCircle2,
  Copy,
  Search,
  ChevronRight,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import { Empty, EmptyHeader, EmptyDescription } from '@/components/ui/empty';
import {
  tenantProjectStore,
  type TenantAgent,
  type TenantPipelineTemplate,
  type TenantPipelineTemplateNode,
  type TenantProject,
  type MyRegisteredAgent,
  type PendingAgentConnection,
} from '../tenantProjectStore';
import WorkflowCanvas from './WorkflowCanvas';

/** "接入 agent" 弹窗：①从我已注册的 agent 选一个 ②接入外部 agent（装CLI 后轮询捕获） */
export function AttachAgentDialog({
  open,
  onOpenChange,
  project,
  currentUser,
  onAttached,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  project: TenantProject;
  currentUser: string;
  onAttached?: (agentId: string) => void;
}) {
  const [mode, setMode] = useState<'existing' | 'external'>('existing');
  // 方式1：我的已有 agent
  const [myAgents, setMyAgents] = useState<MyRegisteredAgent[]>([]);
  const [selectedId, setSelectedId] = useState<string>('');
  // 方式2：外部接入（名称由外部 agent 注册时自动带入，无需在此输入）
  const [extPlatform, setExtPlatform] =
    useState<TenantAgent['platform']>('codebuddy');
  const [pending, setPending] = useState<PendingAgentConnection | null>(null);
  const [checking, setChecking] = useState(false);

  // 打开/切回时刷新我的 agent 列表、重置状态
  useEffect(() => {
    if (!open) return;
    setMyAgents(tenantProjectStore.getMyAgents(project.id, currentUser));
    setSelectedId('');
    setMode('existing');
    setExtPlatform('codebuddy');
    setPending(null);
    setChecking(false);
  }, [open, project.id, currentUser]);

  // 方式2：进入等待态后自动轮询捕获（每 2.5s），命中即落库
  useEffect(() => {
    if (!pending) return;
    const timer = window.setInterval(() => {
      const res = tenantProjectStore.checkPendingAgent(
        project.id,
        currentUser,
        pending,
      );
      if (res.ready && res.agentId) {
        window.clearInterval(timer);
        toast.success(`外部 agent「${pending.name}」已接入`);
        onAttached?.(res.agentId);
        onOpenChange(false);
      }
    }, 2500);
    return () => window.clearInterval(timer);
  }, [pending, project.id, currentUser, onAttached, onOpenChange]);

  // 方式1：确认接入选中的已有 agent
  const attachExisting = () => {
    if (!selectedId) {
      toast.error('请选择一个 agent');
      return;
    }
    const agentId = tenantProjectStore.attachMyAgent(
      project.id,
      currentUser,
      selectedId,
    );
    if (agentId) {
      const ag = myAgents.find((a) => a.id === selectedId);
      toast.success(`已接入「${ag?.name ?? 'agent'}」`);
      onAttached?.(agentId);
      onOpenChange(false);
    } else {
      toast.error('接入失败');
    }
  };

  // 方式2：生成接入码，进入等待态（名称待外部 agent 注册时回填）
  const startExternal = () => {
    setPending(
      tenantProjectStore.startPendingAgentConnection({
        name: '待接入 agent',
        platform: extPlatform,
      }),
    );
  };

  // 方式2：手动刷新（轮询之外的兜底 / 立即查一次）
  const manualRefresh = () => {
    if (!pending) return;
    setChecking(true);
    const res = tenantProjectStore.checkPendingAgent(
      project.id,
      currentUser,
      pending,
    );
    window.setTimeout(() => setChecking(false), 400);
    if (res.ready && res.agentId) {
      toast.success(`外部 agent「${pending.name}」已接入`);
      onAttached?.(res.agentId);
      onOpenChange(false);
    } else {
      toast.info('尚未检测到该 agent 完成注册，请稍候…');
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>接入 agent</DialogTitle>
          <DialogDescription>
            从你已注册的 agent 里选一个直接接入，或按流程接入一个外部 agent。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6 space-y-3">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setMode('existing')}
              className={`flex-1 rounded-[4px] border px-3 py-2 text-sm ${
                mode === 'existing'
                  ? 'border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)] text-[var(--text-brand)]'
                  : 'border-[var(--cp-border)] text-[var(--text-secondary)]'
              }`}
            >
              <UserRound className="w-3.5 h-3.5 inline mr-1" />
              从我的 agent 选
            </button>
            <button
              type="button"
              onClick={() => setMode('external')}
              className={`flex-1 rounded-[4px] border px-3 py-2 text-sm ${
                mode === 'external'
                  ? 'border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)] text-[var(--text-brand)]'
                  : 'border-[var(--cp-border)] text-[var(--text-secondary)]'
              }`}
            >
      <Bot className="w-3.5 h-3.5 inline mr-1" />
        接入外部 agent
      </button>
   </div>

          {/*方式1：从我已注册的 agent 里选 */}
          {mode === 'existing' && (
            <div className="space-y-2">
              <Label>我在 ClawPro 已注册的 agent</Label>
              {myAgents.length === 0 ? (
                <div className="rounded-[4px] border border-dashed border-[var(--cp-border)] bg-[var(--color-gray-100)] px-3 py-6 text-center">
                  <MetaText tone="weak">
                    没有可接入的已注册 agent（可能都已接入本项目）。
                    <br />
                    可切到「接入外部 agent」新接入一个。
                  </MetaText>
                </div>
              ) : (
                <div className="space-y-1.5">
                  {myAgents.map((a) => (
                    <button
                      key={a.id}
                      type="button"
                      onClick={() => setSelectedId(a.id)}
                      className={`flex w-full items-center gap-2 rounded-[4px] border px-3 py-2 text-left transition-colors ${
                        selectedId === a.id
                          ? 'border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)]'
                          : 'border-[var(--cp-border)] hover:border-[var(--cp-brand-blue)]'
                      }`}
                    >
                      <Bot className="w-4 h-4 text-[var(--cp-brand-blue)] shrink-0" />
                      <span className="flex-1 min-w-0">
                        <span className="block text-sm text-[var(--text-body)] truncate">
                          {a.name}
                        </span>
                        <span className="block text-xs text-[var(--text-weak)]">
                          {a.platform}
                        </span>
                      </span>
                      {selectedId === a.id && (
                        <CheckCircle2 className="w-4 h-4 text-[var(--cp-brand-blue)] shrink-0" />
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* 方式2：接入外部 agent */}
          {mode === 'external' && (
            <div className="space-y-3">
              {!pending ? (
                <>
                  <div className="rounded-[4px] border border-[#D6E4FF] bg-[#F8FBFF] px-3 py-2">
                    <MetaText tone="secondary" className="block leading-relaxed">
                      支持 CodeBuddy、ClawPro、Codex、iMate、WorkBuddy 等 Agent 平台。
                      点「生成接入码」后，在本地按提示安装 CLI/插件并用接入码注册，
                      系统会自动捕获并接入到当前项目。
                    </MetaText>
                  </div>
                </>
              ) : (
                <div className="space-y-3">
                  <div className="rounded-[4px] border border-[var(--cp-border)] bg-white px-3 py-2 flex items-center justify-between gap-2">
                    <div className="min-w-0">
                      <MetaText tone="weak">接入码</MetaText>
                      <code className="block text-sm font-mono text-[var(--text-body)] mt-0.5">
                        {pending.code}
                      </code>
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        navigator.clipboard?.writeText(pending.code);
                        toast.success('已复制接入码');
                      }}
                      className="shrink-0 inline-flex items-center gap-1 h-7 px-2 rounded-[4px] border border-[var(--cp-border)] text-xs text-[var(--text-body)] hover:border-[var(--cp-brand-blue)]"
                    >
                      <Copy className="w-3.5 h-3.5" />
                      复制
                    </button>
                  </div>
                  <div className="flex items-center gap-2 rounded-[4px] border border-[#D6E4FF] bg-[#F8FBFF] px-3 py-3">
                    <Loader2 className="w-4 h-4 text-[var(--cp-brand-blue)] animate-spin shrink-0" />
                    <MetaText tone="secondary" className="flex-1">
                      等待本地 agent 完成安装与注册…
                      检测到后将自动接入并带入其名称。
                    </MetaText>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="tenant-outline"
                      size="sm"
                      onClick={manualRefresh}
                      disabled={checking}
                    >
                      <RefreshCw
                        className={`w-4 h-4 ${checking ? 'animate-spin' : ''}`}
                      />
                      刷新
                    </Button>
                    <Button
                      variant="tenant-outline"
                      size="sm"
                      onClick={manualRefresh}
                    >
                      <CheckCircle2 className="w-4 h-4" />
                      我已完成安装
                    </Button>
                    <button
                      type="button"
                      onClick={() => setPending(null)}
                      className="ml-auto text-xs text-[var(--text-weak)] hover:text-[var(--text-body)]"
                    >
                      取消接入
                    </button>
                  </div>
                </div>
              )}
   </div>
)}
  </DialogBody>
  <DialogFooter>
     <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
 关闭
  </Button>
          {mode === 'existing' && (
            <Button
      variant="tenant-primary"
    onClick={attachExisting}
    disabled={!selectedId}
            >
   确认接入
        </Button>
   )}
      {mode === 'external' && !pending && (
        <Button variant="tenant-primary" onClick={startExternal}>
 生成接入码
            </Button>
          )}
     </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const FEATURED_BADGES: Record<string, string> = {
  'pl-ticket': '研发精选',
  'pl-unittest': '研发精选',
  'pl-standard-development-sop': '研发精选',
  'pl-multi-agent-development-sop': '研发精选',
  'pl-project-management-sop': '产品精选',
  'pl-product-frontend-demo-sop': '产品精选',
  'pl-pre-visit-brief': '销售精选',
  'pl-knowledge-base-inspection': '架构精选',
  'pl-weekly-report': '产品精选',
};

export default function WorkflowTab({
  project,
  currentUser,
  createSignal,
}: {
  project: TenantProject;
  currentUser: string;
  /** 外部触发新建工作流（Tab 栏"新建工作流"按钮通过它触发） */
  createSignal: number;
}) {
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(
    project.pipelineTemplates[0]?.id ?? null,
  );
  const [attachOpen, setAttachOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const lastCreateSignalRef = useRef(createSignal);

  const openCreate = () => {
    setSelectedTemplateId(null);
  };
  const openEdit = (t: TenantPipelineTemplate) => {
    setSelectedTemplateId(t.id);
  };

  // 外部信号驱动新建：只在信号"递增"时开，避免重挂时误触发
  useEffect(() => {
    if (createSignal > lastCreateSignalRef.current) openCreate();
    lastCreateSignalRef.current = createSignal;
  }, [createSignal]);

  const handleSave = (
    editing: TenantPipelineTemplate | null,
    data: {
      name: string;
      description: string;
      nodes: TenantPipelineTemplateNode[];
    },
  ) => {
    if (editing) {
      tenantProjectStore.updatePipelineTemplate(project.id, editing.id, data);
      setSelectedTemplateId(editing.id);
      toast.success('工作流已更新');
    } else {
      const templateId = tenantProjectStore.createPipelineTemplate(project.id, data);
      setSelectedTemplateId(templateId);
      toast.success('工作流已创建');
    }
  };
  const handleDelete = () => {
    if (!deleteId) return;
    tenantProjectStore.deletePipelineTemplate(project.id, deleteId);
    if (selectedTemplateId === deleteId) {
      setSelectedTemplateId(
        project.pipelineTemplates.find(template => template.id !== deleteId)?.id ?? null,
      );
    }
    setDeleteId(null);
    toast.success('工作流已删除');
  };

  const agents = project.agents;
  const selectedTemplate =
    project.pipelineTemplates.find(template => template.id === selectedTemplateId) ?? null;
  const q = search.trim().toLowerCase();
  const templates = q
    ? project.pipelineTemplates.filter(t => {
        const haystack = [
          t.name,
          t.description ?? '',
          ...t.nodes.map(n => n.title),
        ]
          .join(' ')
          .toLowerCase();
        return haystack.includes(q);
      })
    : project.pipelineTemplates;

  return (
    <div className="grid min-h-[680px] grid-cols-[280px_minmax(0,1fr)] overflow-hidden rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white">
      <aside className="flex min-h-0 flex-col border-r border-[var(--cp-border)] bg-[var(--cp-surface)]">
        <div className="border-b border-[var(--cp-border)] p-3">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-muted)]" />
            <Input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="搜索工作流"
              className="h-8 pl-8"
            />
          </div>
        </div>

        <div className="flex-1 space-y-1 overflow-y-auto p-2">
        {templates.length === 0 ? (
          <Empty className="py-10">
            <EmptyHeader>
              {search ? '无匹配的工作流' : '暂无工作流'}
            </EmptyHeader>
            <EmptyDescription>
              {search
                ? '试试调整搜索关键词。'
                : '新建一条工作流，把"某类工作怎么做"固化下来，供任务复用。'}
            </EmptyDescription>
          </Empty>
        ) : (
          templates.map((template) => {
            const badge = FEATURED_BADGES[template.id];
            const active = selectedTemplateId === template.id;
            return (
              <button
                key={template.id}
                type="button"
                onClick={() => openEdit(template)}
                className={`group relative w-full overflow-hidden rounded-[var(--radius-md)] border px-3 text-left transition-colors ${
                  badge ? 'pb-2.5 pt-6' : 'py-2.5'
                } ${
                  active
                    ? 'border-[var(--cp-brand-blue)] bg-white shadow-sm'
                    : 'border-transparent hover:border-[var(--cp-border)] hover:bg-white'
                }`}
              >
                <div className="flex items-start gap-2">
                  <WorkflowIcon className="mt-0.5 h-4 w-4 shrink-0 text-[var(--cp-brand-blue)]" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <BodyMedium className="truncate">{template.name}</BodyMedium>
                    </div>
                    <MetaText tone="weak" className="mt-1 block truncate">
                      {template.nodes.length} 节点 · {template.description || '暂无说明'}
                    </MetaText>
                  </div>
                  <ChevronRight className={`mt-0.5 h-4 w-4 shrink-0 ${active ? 'text-[var(--cp-brand-blue)]' : 'text-[var(--text-weak)]'}`} />
                </div>
                {badge && (
                  <span className="absolute left-0 top-0 rounded-br-[6px] bg-[#E8F1FF] px-2 py-0.5 text-[10px] font-medium leading-4 text-[var(--cp-brand-blue)]">
                    {badge}
                  </span>
                )}
              </button>
            );
          })
        )}
        </div>
      </aside>

      <section className="min-w-0 bg-white p-3">
        <WorkflowCanvas
          key={selectedTemplate?.id ?? 'new-workflow'}
          open
          embedded
          onOpenChange={() => undefined}
          template={selectedTemplate}
          onSave={handleSave}
          members={project.members}
          agents={agents}
          onAttachAgent={() => setAttachOpen(true)}
        />
      </section>

      {/* 保留原弹窗调用能力的单一嵌入编辑器 */}

      {/* 接入 agent 弹窗（画布内 "+接入agent" 链接触发） */}
      <AttachAgentDialog
        open={attachOpen}
        onOpenChange={setAttachOpen}
        project={project}
        currentUser={currentUser}
      />

      {/* 删除确认 */}
      <Dialog
        open={!!deleteId}
        onOpenChange={(o) => !o && setDeleteId(null)}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>删除工作流</DialogTitle>
            <DialogDescription>
              删除后使用该工作流已创建的任务不受影响，但无法再实例化新的任务。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              onClick={() => setDeleteId(null)}
            >
              取消
            </Button>
            <Button variant="tenant-primary" onClick={handleDelete}>
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
