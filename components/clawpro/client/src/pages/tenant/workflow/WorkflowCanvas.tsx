/**
 * WorkflowCanvas —— 工作流可视化 DAG 画布编辑器
 *
 * 特性：
 *  - SVG 连线 + 绝对定位节点 + 自动分层布局
 *  - 执行者二选一（成员的 agent / 项目公共 agent）
 *  - 支持并联/汇合（多依赖 / 多下游）
 *  - 全屏 + 缩放（50%~200%）
 *  - 数据流：默认上一节点产出即作为下一节点输入，无需手动声明
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Plus,
  Trash2,
  GitBranch,
  Workflow,
  Bot,
  Maximize2,
  Minimize2,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  FileJson,
  Repeat,
  GripVertical,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { MetaText } from '@/components/ui/Typography';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import type {
  TenantAgent,
  TenantPipelineTemplate,
  TenantPipelineTemplateNode,
  TenantProjectMember,
} from '../tenantProjectStore';
import { layoutWorkflow, edgePath, loopEdgePath, NODE_W, NODE_H } from './canvasLayout';
import { NodeConfigAssets } from './NodeConfigAssets';

let seqCounter = 0;
function nextNodeId(): string {
  seqCounter += 1;
  return `c${Date.now().toString(36)}${seqCounter}`;
}

type ExecutorKind = 'memberAgent' | 'projectAgent';

function encodeExecutor(kind: ExecutorKind, ref: string): string {
  return `${kind}::${ref}`;
}
function decodeExecutor(v: string): { kind: ExecutorKind; ref: string } | null {
  const [k, r] = v.split('::');
  if (!k || !r) return null;
  return { kind: k as ExecutorKind, ref: r };
}

export interface WorkflowCanvasProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  template: TenantPipelineTemplate | null;
  onSave: (
    template: TenantPipelineTemplate | null,
    data: {
      name: string;
      description: string;
      nodes: TenantPipelineTemplateNode[];
    },
  ) => void;
  members: TenantProjectMember[];
  agents: TenantAgent[];
  onAttachAgent?: () => void;
  /** 直接嵌入工作流 Tab 右侧，不使用弹窗外壳。 */
  embedded?: boolean;
}

export default function WorkflowCanvas({
  open,
  onOpenChange,
  template,
  onSave,
  members,
  agents,
  onAttachAgent,
  embedded = false,
}: WorkflowCanvasProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [nodes, setNodes] = useState<TenantPipelineTemplateNode[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const [zoom, setZoom] = useState(1); // 0.5 ~ 2
  const [showImport, setShowImport] = useState(false);
  const [importJson, setImportJson] = useState('');
  const [dragging, setDragging] = useState(false);
  const dragStart = useRef({ x: 0, y: 0, scrollLeft: 0, scrollTop: 0 });
  const nodeDrag = useRef<{
    id: string;
    pointerId: number;
    startX: number;
    startY: number;
    originX: number;
    originY: number;
  } | null>(null);
  const canvasRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    if (template) {
      setName(template.name);
      setDescription(template.description);
      setNodes(
        template.nodes.map((n) => ({
          ...n,
          dependsOn: [...n.dependsOn],
        })),
      );
      setSelectedId(template.nodes[0]?.id ?? null);
    } else {
      const first: TenantPipelineTemplateNode = {
        id: nextNodeId(),
        title: '开始节点',
        agentRole: '',
        dependsOn: [],
      };
      setName('');
      setDescription('');
      setNodes([first]);
      setSelectedId(first.id);
    }
    setFullscreen(false);
    setZoom(1);
  }, [open, template]);

  const layout = useMemo(() => layoutWorkflow(nodes), [nodes]);
  const selected = nodes.find((n) => n.id === selectedId) ?? null;

  const memberAgents = useMemo(() => {
    const grouped: Record<string, TenantAgent[]> = {};
    agents
      .filter((a) => a.kind !== 'project')
      .forEach((a) => {
        (grouped[a.ownerId] ||= []).push(a);
      });
    return grouped;
  }, [agents]);
  const projectAgents = agents.filter((a) => a.kind === 'project');

  const addNode = (mode: 'root' | 'child' | 'loop') => {
    const id = nextNodeId();
    const dependsOn = mode === 'child' && selectedId ? [selectedId] : [];
    setNodes((prev) => [
      ...prev,
      {
        id,
        title: mode === 'loop' ? '循环节点' : mode === 'root' ? '新起点' : '新子节点',
        agentRole: '',
        dependsOn,
        ...(mode === 'loop' ? { type: 'loop' as const } : {}),
      },
    ]);
    setSelectedId(id);
  };

  const removeNode = (id: string) => {
    setNodes((prev) =>
      prev
        .filter((n) => n.id !== id)
        .map((n) => ({
          ...n,
          dependsOn: n.dependsOn.filter((d) => d !== id),
        })),
    );
    setSelectedId((cur) => (cur === id ? null : cur));
  };

  const patchNode = (
    id: string,
    patch: Partial<TenantPipelineTemplateNode>,
  ) => {
    setNodes((prev) => prev.map((n) => (n.id === id ? { ...n, ...patch } : n)));
  };

  const toggleDep = (id: string, dep: string) => {
    setNodes((prev) =>
      prev.map((n) => {
        if (n.id !== id) return n;
        const set = new Set(n.dependsOn);
        if (set.has(dep)) set.delete(dep);
        else set.add(dep);
        return { ...n, dependsOn: Array.from(set) };
      }),
    );
  };

  /** 计算某节点的所有下游节点 id（直接或间接依赖它的节点） */
  const downstreamNodeIds = (nodeId: string): Set<string> => {
    const ids = new Set<string>();
    const queue = nodes
      .filter((n) => n.dependsOn.includes(nodeId))
      .map((n) => n.id);
    while (queue.length > 0) {
      const cur = queue.shift()!;
      if (ids.has(cur)) continue;
      ids.add(cur);
      nodes
        .filter((n) => n.dependsOn.includes(cur))
        .forEach((n) => queue.push(n.id));
    }
    return ids;
  };

  const handleImport = () => {
    try {
      const data = JSON.parse(importJson);
      if (!data.name || !Array.isArray(data.nodes)) return;
      setName(data.name);
      setDescription(data.description ?? '');
      const nodeMap = new Map<string, string>();
      data.nodes.forEach((n: any, i: number) => {
        nodeMap.set(n.title, `imp-${Date.now()}-${i}`);
      });
      const importedNodes: TenantPipelineTemplateNode[] = data.nodes.map(
        (n: any) => {
          let executorKind: 'memberAgent' | 'projectAgent' | undefined;
          let executorRef: string | undefined;
          if (n.agentName) {
            const agent = agents.find((a) => a.name === n.agentName);
            if (agent) {
              executorKind =
                agent.kind === 'project' ? 'projectAgent' : 'memberAgent';
              executorRef = agent.id;
            }
          }
          return {
            id: nodeMap.get(n.title)!,
            title: n.title,
            agentRole: n.agentRole ?? '',
            dependsOn: (n.dependsOn ?? [])
              .map((dep: string) => nodeMap.get(dep))
              .filter(Boolean),
            promptTemplate: n.promptTemplate,
            position:
              typeof n.position?.x === 'number' && typeof n.position?.y === 'number'
                ? { x: n.position.x, y: n.position.y }
                : undefined,
            executorKind,
            executorRef,
            ...(n.type === 'loop' ? { type: 'loop' as const } : {}),
            ...(n.type === 'loop' && n.loopConfig?.endNodeId
              ? {
                  loopConfig: {
                    endNodeId:
                      nodeMap.get(n.loopConfig.endNodeId) ??
                      n.loopConfig.endNodeId,
                    maxCount: n.loopConfig.maxCount ?? 3,
                    exitCondition: n.loopConfig.exitCondition,
                  },
                }
              : {}),
          };
        },
      );
      setNodes(importedNodes);
      setSelectedId(importedNodes[0]?.id ?? null);
      setShowImport(false);
      setImportJson('');
    } catch {
      /* 解析失败 */
    }
  };

  const handleCanvasMouseDown = (e: React.MouseEvent) => {
    if (e.target !== e.currentTarget) return;
    if (!canvasRef.current) return;
    setDragging(true);
    dragStart.current = {
      x: e.clientX,
      y: e.clientY,
      scrollLeft: canvasRef.current.scrollLeft,
      scrollTop: canvasRef.current.scrollTop,
    };
  };

  const handleCanvasMouseMove = (e: React.MouseEvent) => {
    if (!dragging || !canvasRef.current) return;
    canvasRef.current.scrollLeft =
      dragStart.current.scrollLeft - (e.clientX - dragStart.current.x);
    canvasRef.current.scrollTop =
      dragStart.current.scrollTop - (e.clientY - dragStart.current.y);
  };

  const handleCanvasMouseUp = () => setDragging(false);

  const valid =
    name.trim() &&
    nodes.length > 0 &&
    nodes.every(
      (n) =>
        n.title.trim() &&
        (n.type === 'loop' || (n.promptTemplate ?? '').trim()) &&
        (n.type !== 'loop' || n.loopConfig?.endNodeId),
    );

  const handleSave = () => {
    onSave(template, {
      name: name.trim(),
      description: description.trim(),
    nodes: nodes.map((n) => ({
      ...n,
      id: n.id,
      title: n.title.trim() || '未命名',
      agentRole: '',
      executorKind: n.executorKind,
      executorRef: n.executorRef,
      dependsOn: [...n.dependsOn],
      promptTemplate: n.promptTemplate,
      position: n.position,
      type: n.type,
      loopConfig: n.loopConfig,
    })),
    });
    onOpenChange(false);
  };

  const executorLabel = (n: TenantPipelineTemplateNode): string => {
    if (!n.executorKind || !n.executorRef) return '未派发';
    const ag = agents.find((a) => a.id === n.executorRef);
    if (!ag) return n.executorKind === 'projectAgent' ? '项目 Agent' : 'Agent';
    return ag.name;
  };

  const executorIcon = (_n: TenantPipelineTemplateNode) => (
    <Bot className="w-3 h-3" />
  );

  const contentClass = fullscreen
    ? 'w-[96vw] max-w-none h-[92vh] flex flex-col p-0'
    : 'sm:max-w-[1000px] flex flex-col';
  const contentStyle = fullscreen
    ? undefined
    : ({ maxHeight: 'min(92vh, 860px)' } as React.CSSProperties);

  const editorContent = (
    <>
        {embedded ? (
          <div className="px-6 pt-4">
            <div className="flex items-center gap-2 text-base font-semibold text-[var(--text-title)]">
              <Workflow className="h-4 w-4 text-[var(--cp-brand-blue)]" />
              {template ? '编辑工作流' : '新建工作流'}
            </div>
            <p className="mt-1 text-sm text-[var(--text-secondary)]">
              拖拽节点可调整画布位置；使用“+”新增节点。<b>支持并联</b>：一节点可依赖多上游，也可分支到多下游。
            </p>
          </div>
        ) : (
        <DialogHeader className={fullscreen ? 'px-6 pt-4' : undefined}>
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <DialogTitle className="flex items-center gap-2">
                <Workflow className="w-4 h-4 text-[var(--cp-brand-blue)]" />
                {template ? '编辑工作流' : '新建工作流'}
              </DialogTitle>
              <DialogDescription>
                拖拽节点可调整画布位置；使用“+”新增节点。<b>支持并联</b>：一节点可依赖多上游，也可分支到多下游。
              </DialogDescription>
            </div>
            <button
              type="button"
              onClick={() => setFullscreen((v) => !v)}
              className="shrink-0 inline-flex items-center gap-1 h-8 px-2 mr-8 rounded-[4px] border border-[var(--cp-border)] text-sm text-[var(--text-body)] hover:border-[var(--cp-brand-blue)]"
              title={fullscreen ? '退出全屏' : '全屏'}
            >
              {fullscreen ? (
                <Minimize2 className="w-4 h-4" />
              ) : (
                <Maximize2 className="w-4 h-4" />
              )}
            </button>
          </div>
        </DialogHeader>
        )}

        <div className="px-6">
          <div className="max-w-md space-y-1.5">
            <Label>工作流名称</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：需求交付流水线 / 每日进度汇报"
            />
          </div>
        </div>

        {/* 画布工具条：缩放 + 数据流开关 */}
        <div className="flex items-center justify-between gap-2 px-6 pt-2">
          <div className="flex items-center gap-1">
            <button
              type="button"
              className="inline-flex items-center h-7 px-2 rounded-[4px] border border-[var(--cp-border)] text-xs text-[var(--text-body)] hover:border-[var(--cp-brand-blue)]"
              onClick={() => setZoom((z) => Math.max(0.5, +(z - 0.1).toFixed(2)))}
              title="缩小"
            >
              <ZoomOut className="w-3.5 h-3.5" />
            </button>
            <span className="text-xs text-[var(--text-body)] tabular-nums w-10 text-center">
              {Math.round(zoom * 100)}%
            </span>
            <button
              type="button"
              className="inline-flex items-center h-7 px-2 rounded-[4px] border border-[var(--cp-border)] text-xs text-[var(--text-body)] hover:border-[var(--cp-brand-blue)]"
              onClick={() => setZoom((z) => Math.min(2, +(z + 0.1).toFixed(2)))}
              title="放大"
            >
              <ZoomIn className="w-3.5 h-3.5" />
            </button>
            <button
              type="button"
              className="inline-flex items-center h-7 px-2 rounded-[4px] border border-[var(--cp-border)] text-xs text-[var(--text-body)] hover:border-[var(--cp-brand-blue)]"
              onClick={() => setZoom(1)}
              title="重置缩放"
            >
              <RotateCcw className="w-3.5 h-3.5" />
            </button>
          </div>
          <Button
            variant="tenant-outline"
            size="sm"
            onClick={() => setShowImport(true)}
          >
            <FileJson className="w-4 h-4" />
            导入 JSON
          </Button>
        </div>

        {showImport && (
          <div className="px-6 py-3 space-y-2 border-b border-[var(--cp-border)] bg-[var(--cp-surface)] max-h-[50vh] overflow-y-auto">
            <div className="flex items-center justify-between">
              <Label>粘贴工作流 JSON</Label>
              <button
                type="button"
                onClick={() => setShowImport(false)}
                className="text-xs text-[var(--text-weak)] hover:text-[var(--text-body)]"
              >
                关闭
              </button>
            </div>
            <Textarea
              value={importJson}
              onChange={(e) => setImportJson(e.target.value)}
              placeholder={'{\n  "name": "工作流名称",\n  "description": "描述",\n  "nodes": [\n    { "title": "节点A", "promptTemplate": "指令", "dependsOn": [] }\n  ]\n}'}
              className="min-h-[120px] max-h-[300px] overflow-y-auto font-mono text-xs"
            />
            <details>
              <summary className="cursor-pointer text-xs text-[var(--text-secondary)]">
                格式说明
              </summary>
              <div className="mt-1 text-xs text-[var(--text-weak)] leading-relaxed">
                name（必填）、description（可选）、nodes[]（必填）：每个节点含 title、promptTemplate、dependsOn（用标题引用）、agentName（可选）、agentRole（可选）、type（可选，loop=循环节点）、loopConfig（循环节点必填，含 endNodeId 用标题引用、maxCount、exitCondition）
              </div>
            </details>
            <div className="flex gap-2">
              <Button
                variant="tenant-primary"
                size="sm"
                disabled={!importJson.trim()}
                onClick={handleImport}
              >
                导入
              </Button>
              <Button
                variant="tenant-outline"
                size="sm"
                onClick={() => {
                  setShowImport(false);
                  setImportJson('');
                }}
              >
                取消
              </Button>
            </div>
          </div>
        )}

        <div
          className={`flex-1 min-h-0 gap-3 px-6 py-3 ${
            embedded
              ? 'grid grid-cols-1 grid-rows-[360px_auto] overflow-y-auto'
              : 'grid grid-cols-[1fr_320px]'
          }`}
        >
          {/* 画布区 */}
          <div
            ref={canvasRef}
            onMouseDown={handleCanvasMouseDown}
            onMouseMove={handleCanvasMouseMove}
            onMouseUp={handleCanvasMouseUp}
            onMouseLeave={handleCanvasMouseUp}
            className={`relative overflow-auto rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[linear-gradient(var(--cp-border)_1px,transparent_1px),linear-gradient(90deg,var(--cp-border)_1px,transparent_1px)] bg-[length:20px_20px] bg-[#FAFBFC] ${dragging ? 'cursor-grabbing' : 'cursor-grab'}`}
          >
            <div
              className="relative origin-top-left"
              style={{
                width: layout.width,
                height: layout.height,
                minHeight: 240,
                transform: `scale(${zoom})`,
                transformOrigin: '0 0',
              }}
              onDoubleClick={(e) => {
                if (e.target === e.currentTarget) addNode('root');
              }}
            >
              <svg
                className="absolute inset-0 pointer-events-none"
                width={layout.width}
                height={layout.height}
              >
                <defs>
                  <marker
                    id="wf-arrow"
                    markerWidth="8"
                    markerHeight="8"
                    refX="7"
                    refY="4"
                    orient="auto"
                  >
                    <path d="M0,0 L8,4 L0,8 Z" fill="var(--cp-brand-blue)" />
                  </marker>
                </defs>
                {/* 连线：正向依赖（蓝实线）+ 循环回退（橙虚线） */}
                {layout.edges.map((e) => {
                  const isLoop = e.edgeType === 'loopback';
                  return (
                    <path
                      key={`${e.from}-${e.to}-${e.edgeType ?? 'forward'}`}
                      d={isLoop ? loopEdgePath(e) : edgePath(e)}
                      fill="none"
                      stroke={isLoop ? '#f59e0b' : 'var(--cp-brand-blue)'}
                      strokeWidth={1.5}
                      strokeDasharray={isLoop ? '4 3' : undefined}
                      markerEnd={isLoop ? undefined : 'url(#wf-arrow)'}
                      opacity={isLoop ? 0.6 : 0.7}
                    />
                  );
                })}
              </svg>
              {layout.nodes.map((cn) => {
                const active = cn.node.id === selectedId;
                return (
                  <div
                    key={cn.node.id}
                    onClick={() => setSelectedId(cn.node.id)}
                    onPointerDown={(event) => {
                      event.stopPropagation();
                      event.currentTarget.setPointerCapture(event.pointerId);
                      nodeDrag.current = {
                        id: cn.node.id,
                        pointerId: event.pointerId,
                        startX: event.clientX,
                        startY: event.clientY,
                        originX: cn.x,
                        originY: cn.y,
                      };
                      setSelectedId(cn.node.id);
                    }}
                    onPointerMove={(event) => {
                      const drag = nodeDrag.current;
                      if (!drag || drag.id !== cn.node.id) return;
                      patchNode(cn.node.id, {
                        position: {
                          x: Math.max(
                            8,
                            drag.originX + (event.clientX - drag.startX) / zoom,
                          ),
                          y: Math.max(
                            8,
                            drag.originY + (event.clientY - drag.startY) / zoom,
                          ),
                        },
                      });
                    }}
                    onPointerUp={(event) => {
                      if (nodeDrag.current?.id !== cn.node.id) return;
                      nodeDrag.current = null;
                      event.currentTarget.releasePointerCapture(event.pointerId);
                    }}
                    className={`absolute flex touch-none flex-col justify-center rounded-[4px] border px-3 text-left transition-all cursor-move group ${
                      cn.node.type === 'loop'
                        ? active
                          ? 'border-[#f59e0b] bg-[#FFFBEB] shadow-[0_0_0_2px_#FEF3C7]'
                          : 'border-[#f59e0b] bg-[#FFFBEB] hover:shadow-[0_0_0_2px_#FEF3C7]'
                        : active
                          ? 'border-[var(--cp-brand-blue)] bg-white shadow-[0_0_0_2px_var(--bg-brand-selected)]'
                          : 'border-[var(--cp-border)] bg-white hover:border-[var(--cp-brand-blue)]'
                    }`}
                    style={{
                      left: cn.x,
                      top: cn.y,
                      width: NODE_W,
                      height: NODE_H,
                    }}
                  >
                    <span className="flex items-center gap-1 text-sm font-medium text-[var(--text-title)] truncate">
                      <GripVertical className="h-3.5 w-3.5 shrink-0 text-[var(--text-weak)]" />
                      <span className="truncate">{cn.node.title || '未命名'}</span>
                    </span>
                    <span className="mt-0.5 inline-flex items-center gap-1 text-[11px] text-[var(--text-weak)]">
                      {cn.node.type === 'loop' ? (
                        <>
                          <Repeat className="w-3 h-3 text-[#f59e0b]" />
                          循环节点
                        </>
                      ) : (
                        <>
                          {executorIcon(cn.node)}
                          {executorLabel(cn.node)}
                        </>
                      )}
                    </span>
                    {/* 悬停显示：快速添加子节点 */}
                    <button
                      type="button"
                      onPointerDown={(e) => e.stopPropagation()}
                      onClick={(e) => {
                        e.stopPropagation();
                        setSelectedId(cn.node.id);
                        addNode('child');
                      }}
                      className="absolute -right-2 -bottom-2 w-5 h-5 flex items-center justify-center rounded-full bg-[var(--cp-brand-blue)] text-white opacity-0 group-hover:opacity-100 transition-opacity shadow-sm hover:scale-110"
                      title="添加子节点"
                    >
                      <Plus className="w-3 h-3" />
                    </button>
                  </div>
                );
              })}
            </div>
          </div>

          {/* 右侧属性面板 */}
          <div className="flex flex-col gap-2 overflow-y-auto">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="tenant-primary" size="sm" className="w-full">
                  <Plus className="h-4 w-4" />
                  添加节点
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-48">
                <DropdownMenuItem onClick={() => addNode('child')} disabled={!selectedId}>
                  添加子节点
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => addNode('root')}>
                  添加并联起点
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => addNode('loop')}>
                  添加循环节点
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <MetaText tone="weak" className="px-1">
              拖拽节点调整位置 · 使用“+”新增节点
            </MetaText>
            {selected ? (
              <div className="space-y-3 rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-3">
                <div className="space-y-1.5">
                  <Label>节点名称</Label>
                  <Input
                    value={selected.title}
                    onChange={(e) =>
                      patchNode(selected.id, { title: e.target.value })
                    }
                  />
                </div>
                {selected.type === 'loop' && (
                  <div className="space-y-2 rounded-[4px] border border-[#f59e0b]/30 bg-[#FFFBEB] p-2.5">
                    <div className="space-y-1.5">
                      <Label>循环结束节点 <span className="text-[var(--text-danger)]">*</span></Label>
                      <Select
                        value={selected.loopConfig?.endNodeId ?? '__none__'}
                        onValueChange={(v) =>
                          patchNode(selected.id, {
                            loopConfig: v === '__none__' ? undefined : {
                              endNodeId: v,
                              maxCount: selected.loopConfig?.maxCount ?? 3,
                              exitCondition: selected.loopConfig?.exitCondition,
                            },
                          })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="选择结束节点" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__none__">不设置</SelectItem>
                          {Array.from(downstreamNodeIds(selected.id)).map((id) => {
                            const n = nodes.find((nn) => nn.id === id);
                            return n ? (
                              <SelectItem key={n.id} value={n.id}>
                                {n.title || '未命名'}
                              </SelectItem>
                            ) : null;
                          })}
                        </SelectContent>
                      </Select>
                    </div>
                    {selected.loopConfig?.endNodeId && (
                      <div className="grid grid-cols-2 gap-2">
                        <div className="space-y-1.5">
                          <Label>最大循环次数</Label>
                          <Input
                            type="number"
                            value={selected.loopConfig.maxCount ?? 3}
                            onChange={(e) =>
                              patchNode(selected.id, {
                                loopConfig: {
                                  endNodeId: selected.loopConfig!.endNodeId,
                                  maxCount: Number(e.target.value) || 3,
                                  exitCondition: selected.loopConfig?.exitCondition,
                                },
                              })
                            }
                            min={1}
                            max={10}
                          />
                        </div>
                        <div className="space-y-1.5">
                          <Label>退出条件说明</Label>
                          <Input
                            value={selected.loopConfig?.exitCondition ?? ''}
                            onChange={(e) =>
                              patchNode(selected.id, {
                                loopConfig: {
                                  endNodeId: selected.loopConfig!.endNodeId,
                                  maxCount: selected.loopConfig?.maxCount ?? 3,
                                  exitCondition: e.target.value || undefined,
                                },
                              })
                            }
                            placeholder="如：评审通过"
                          />
                        </div>
                      </div>
                    )}
                  </div>
                )}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <Label>派发给</Label>
                    {onAttachAgent && (
                      <button
                        type="button"
                        className="text-[11px] text-[var(--cp-brand-blue)] hover:underline"
                        onClick={onAttachAgent}
                      >
                        + 接入 agent
                      </button>
                    )}
                  </div>
                  <Select
                    value={
                      selected.executorKind && selected.executorRef
                        ? encodeExecutor(selected.executorKind, selected.executorRef)
                        : ''
                    }
                    onValueChange={(v) => {
                      const dec = decodeExecutor(v);
                      if (dec) {
                        patchNode(selected.id, {
                          executorKind: dec.kind,
                          executorRef: dec.ref,
                        });
                      }
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="选择执行者" />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.keys(memberAgents).length > 0 && (
                        <div className="px-2 py-1 text-[11px] text-[var(--text-weak)]">
                          成员的 agent
                        </div>
                      )}
                      {Object.entries(memberAgents).map(([ownerId, list]) => {
                        const owner =
                          members.find((m) => m.userId === ownerId)
                            ?.displayName ?? ownerId;
                        return list.map((a) => (
                          <SelectItem
                            key={`ma-${a.id}`}
                            value={encodeExecutor('memberAgent', a.id)}
                          >
                            <span className="inline-flex items-center gap-1.5">
                              <Bot className="w-3 h-3" />
                              {a.name}
                              <span className="text-[var(--text-weak)]">
                                · {owner}
                              </span>
                            </span>
                          </SelectItem>
                        ));
                      })}
                      {projectAgents.length > 0 && (
                        <div className="px-2 py-1 text-[11px] text-[var(--text-weak)]">
                          项目公共 agent
                        </div>
                      )}
                      {projectAgents.map((a) => (
                        <SelectItem
                          key={`pa-${a.id}`}
                          value={encodeExecutor('projectAgent', a.id)}
                        >
                          <span className="inline-flex items-center gap-1.5">
                            <Bot className="w-3 h-3 text-[var(--cp-brand-blue)]" />
                            {a.name}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {/* 指定上下游 */}
                <details className="group rounded-[4px] border border-[var(--cp-border)] bg-white px-2.5 py-2">
                  <summary className="flex cursor-pointer list-none items-center gap-1 text-xs text-[var(--text-secondary)]">
                    <GitBranch className="h-3.5 w-3.5" />
                    高级设置（上下游）
                  </summary>
                <div className="mt-2 space-y-2">
                  {/* 上游 */}
                  <div>
                    <MetaText tone="weak" className="mb-1 block">
                      上游（等待这些节点完成）
                    </MetaText>
                    <div className="flex flex-wrap gap-1.5">
                      {nodes.filter((n) => n.id !== selected.id).length === 0 ? (
                        <MetaText tone="weak">暂无其他节点</MetaText>
                      ) : (
                        nodes
                          .filter((n) => n.id !== selected.id)
                          .map((n) => {
                            const on = selected.dependsOn.includes(n.id);
                            return (
                              <button
                                key={n.id}
                                type="button"
                                onClick={() => toggleDep(selected.id, n.id)}
                                className={`h-6 px-2 rounded-[4px] border text-[11px] transition-colors ${
                                  on
                                    ? 'border-[var(--cp-brand-blue)] bg-[#F0F5FF] text-[var(--cp-brand-blue)]'
                                    : 'border-[var(--cp-border)] bg-white text-[var(--text-muted)] hover:border-[var(--text-weak)]'
                                }`}
                              >
                                {n.title || '未命名'}
                              </button>
                            );
                          })
                      )}
                    </div>
                  </div>
                  {/* 下游 */}
                  <div>
                    <MetaText tone="weak" className="mb-1 block">
                      下游（本节点完成后启动）
                    </MetaText>
                    <div className="flex flex-wrap gap-1.5">
                      {nodes.filter((n) => n.id !== selected.id).length === 0 ? (
                        <MetaText tone="weak">暂无其他节点</MetaText>
                      ) : (
                        nodes
                          .filter((n) => n.id !== selected.id)
                          .map((n) => {
                            const on = n.dependsOn.includes(selected.id);
                            return (
                              <button
                                key={n.id}
                                type="button"
                                onClick={() => toggleDep(n.id, selected.id)}
                                className={`h-6 px-2 rounded-[4px] border text-[11px] transition-colors ${
                                  on
                                    ? 'border-[var(--cp-brand-blue)] bg-[#F0F5FF] text-[var(--cp-brand-blue)]'
                                    : 'border-[var(--cp-border)] bg-white text-[var(--text-muted)] hover:border-[var(--text-weak)]'
                                }`}
                              >
                                {n.title || '未命名'}
                              </button>
                            );
                          })
                      )}
                    </div>
                  </div>
                </div>
                </details>

                {/* prompt 模板（必填：明确要 agent 干什么） */}
                <div className="space-y-1.5">
                  <Label className="flex items-center gap-1">
                    Prompt
                    <span className="text-[var(--text-danger)]">*</span>
                  </Label>
                  <textarea
                    value={selected.promptTemplate ?? ''}
                    onChange={(e) =>
                      patchNode(selected.id, {
                        promptTemplate: e.target.value,
                      })
                    }
                    placeholder="给 agent 的指令，例如：基于上一节点的产出，输出评审结论"
                    className={`w-full min-h-16 rounded-[4px] border bg-white px-2 py-1.5 text-xs text-[var(--text-body)] ${
                      (selected.promptTemplate ?? '').trim()
                        ? 'border-[var(--cp-border)]'
                        : 'border-[var(--text-danger)]'
                    }`}
                  />
                  {!(selected.promptTemplate ?? '').trim() && (
                    <MetaText tone="weak" className="text-[var(--text-danger)]">
                      请填写 Prompt，说明这个节点要 agent 做什么。
                    </MetaText>
                  )}
                </div>

                {(selected.configAssets?.length ?? 0) > 0 && (
                  <details className="group" open>
                    <summary className="flex cursor-pointer list-none items-center justify-between gap-2 text-xs font-medium text-[var(--text-secondary)] hover:text-[var(--text-body)] [&::-webkit-details-marker]:hidden">
                      <span>配置资产（{selected.configAssets?.length}）</span>
                      <span className="text-[11px] font-normal text-[var(--text-weak)] group-open:hidden">
                        展开
                      </span>
                      <span className="hidden text-[11px] font-normal text-[var(--text-weak)] group-open:inline">
                        收起
                      </span>
                    </summary>
                    <div className="mt-2">
                      <NodeConfigAssets assets={selected.configAssets ?? []} />
                    </div>
                  </details>
                )}

                {/* 数据流（自动）：上游产出即本节点输入，本节点产出即下游输入 */}
                <details className="group">
                  <summary className="cursor-pointer text-xs text-[var(--text-secondary)] hover:text-[var(--text-body)] select-none">
                    数据流（自动生成，点击展开）
                  </summary>
                  <div className="space-y-2 mt-2">
                    <div className="space-y-1.5">
                      <Label className="flex items-center gap-1">
                        <span className="text-[#22c55e]">←</span>
                        Input（自动）
                      </Label>
                      <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] px-2.5 py-2">
                        {selected.dependsOn.length === 0 ? (
                          <MetaText tone="weak">无上游，使用任务输入作为输入。</MetaText>
                        ) : (
                          <MetaText tone="secondary" className="block leading-relaxed">
                            自动接收上游产出：
                            {selected.dependsOn
                              .map(
                                (d) =>
                                  nodes.find((n) => n.id === d)?.title ?? '未命名',
                              )
                              .join('、')}
                          </MetaText>
                        )}
                      </div>
                    </div>
                    <div className="space-y-1.5">
                      <Label className="flex items-center gap-1">
                        Output（自动）
                        <span className="text-[#22c55e]">→</span>
                      </Label>
                      <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] px-2.5 py-2">
                        <MetaText tone="secondary" className="block leading-relaxed">
                          本节点跑完的产出，将自动作为下游节点的输入。
                        </MetaText>
                      </div>
                    </div>
                  </div>
                </details>

                <Button
                  variant="tenant-outline"
                  size="sm"
                  className="w-full text-[var(--text-danger)]"
                  disabled={nodes.length <= 1}
                  onClick={() => removeNode(selected.id)}
                >
                  <Trash2 className="w-4 h-4" />
                  删除此节点
                </Button>
              </div>
            ) : (
              <MetaText tone="weak" className="p-3">
                点击画布上的节点进行编辑
              </MetaText>
            )}
          </div>
        </div>

        <DialogFooter className={fullscreen || embedded ? 'px-6 pb-4' : undefined}>
          {!embedded && (
            <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
          )}
          <Button
            variant="tenant-primary"
            disabled={!valid}
            onClick={handleSave}
          >
            保存工作流
          </Button>
        </DialogFooter>
    </>
  );

  if (embedded) {
    return (
      <div className="flex h-full min-h-[620px] flex-col overflow-hidden rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white">
        {editorContent}
      </div>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={contentClass} style={contentStyle}>
        {editorContent}
      </DialogContent>
    </Dialog>
  );
}
