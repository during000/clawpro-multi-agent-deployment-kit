/**
 * canvasLayout —— 工作流可视化画布的 DAG 分层布局与连线计算
 *
 * 零依赖：按 dependsOn 计算拓扑分层（stage），同层纵向排布，层间横向推进（从左往右）；
 * 连线用贝塞尔路径。画布节点用绝对定位，坐标由本模块统一给出。
 */
import type { TenantPipelineTemplateNode } from '../tenantProjectStore';

export const NODE_W = 180;
export const NODE_H = 64;
export const GAP_X = 80; // 层间水平间距
export const GAP_Y = 28; // 同层垂直间距
export const PAD = 32; // 画布内边距

export interface CanvasNode {
  node: TenantPipelineTemplateNode;
  x: number;
  y: number;
  stage: number;
}

export interface CanvasEdge {
  from: string;
  to: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  edgeType?: 'forward' | 'loopback';
}

export interface CanvasLayout {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  width: number;
  height: number;
}

/** 按 dependsOn 计算每个节点的 stage（层号），有悬空依赖按 0 处理 */
function computeStages(
  nodes: TenantPipelineTemplateNode[],
): Record<string, number> {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const memo: Record<string, number> = {};
  const visiting = new Set<string>();
  const stageOf = (id: string): number => {
    if (memo[id] !== undefined) return memo[id];
    const n = byId.get(id);
    if (!n || n.dependsOn.length === 0 || visiting.has(id)) {
      memo[id] = 0;
      return 0;
    }
    visiting.add(id);
    const s =
      Math.max(
        ...n.dependsOn.map((d) => (byId.has(d) ? stageOf(d) : -1)),
      ) + 1;
    visiting.delete(id);
    memo[id] = Math.max(0, s);
    return memo[id];
  };
  nodes.forEach((n) => stageOf(n.id));
  return memo;
}

/** 计算画布布局：节点坐标 + 连线路径 + 画布尺寸（从左往右排布） */
export function layoutWorkflow(
  nodes: TenantPipelineTemplateNode[],
): CanvasLayout {
  const stages = computeStages(nodes);
  const byStage: Record<number, TenantPipelineTemplateNode[]> = {};
  nodes.forEach((n) => {
    const s = stages[n.id] ?? 0;
    (byStage[s] ??= []).push(n);
  });
  const stageKeys = Object.keys(byStage)
    .map(Number)
    .sort((a, b) => a - b);
  Object.values(byStage).forEach(stage => {
    // 完整研发链路是主路径，SOLO 小需求快捷分支固定放在下方。
    const isSolo = (node: TenantPipelineTemplateNode) =>
      node.id === 'SOLO' || node.title.startsWith('SOLO');
    stage.sort((a, b) => Number(isSolo(a)) - Number(isSolo(b)));
  });

  const pos = new Map<string, { x: number; y: number }>();
  const rowById = new Map<string, number>();
  let maxRows = 0;
  stageKeys.forEach((s, stageIdx) => {
    const list = byStage[s];
    const occupiedRows = new Set<number>();
    list.forEach((n, rowIdx) => {
      const dependencyRows = n.dependsOn
        .map(dependencyId => rowById.get(dependencyId))
        .filter((row): row is number => row !== undefined);
      // 分支创建时按当前层顺序分配泳道；后续节点继承上游泳道。
      // 多分支汇总继承最后声明的主链，避免主流程在汇总处改变泳道。
      let lane = list.length > 1
        ? rowIdx
        : dependencyRows.length > 0
          ? dependencyRows[dependencyRows.length - 1]
          : 0;
      while (occupiedRows.has(lane)) lane += 1;
      occupiedRows.add(lane);
      rowById.set(n.id, lane);
      maxRows = Math.max(maxRows, lane + 1);
      pos.set(n.id, {
        x: n.position?.x ?? PAD + stageIdx * (NODE_W + GAP_X),
        y: n.position?.y ?? PAD + lane * (NODE_H + GAP_Y),
      });
    });
  });

  const canvasNodes: CanvasNode[] = nodes.map((n) => {
    const p = pos.get(n.id)!;
    return { node: n, x: p.x, y: p.y, stage: stages[n.id] ?? 0 };
  });

  const edges: CanvasEdge[] = [];
  nodes.forEach((n) => {
    n.dependsOn.forEach((dep) => {
      const from = pos.get(dep);
      const to = pos.get(n.id);
      if (!from || !to) return;
      edges.push({
        from: dep,
        to: n.id,
        x1: from.x + NODE_W,
        y1: from.y + NODE_H / 2,
        x2: to.x,
        y2: to.y + NODE_H / 2,
      });
    });
  });

  // 回退边（循环）：循环节点右侧 → 结束节点右侧
  nodes.forEach((n) => {
    if (n.type === 'loop' && n.loopConfig?.endNodeId) {
      const from = pos.get(n.id);
      const to = pos.get(n.loopConfig.endNodeId);
      if (!from || !to) return;
      edges.push({
        from: n.id,
        to: n.loopConfig.endNodeId,
        x1: from.x + NODE_W,
        y1: from.y + NODE_H / 2,
        x2: to.x + NODE_W,
        y2: to.y + NODE_H / 2,
        edgeType: 'loopback',
      });
    }
  });

  const autoWidth =
    PAD * 2 +
    stageKeys.length * NODE_W +
    Math.max(0, stageKeys.length - 1) * GAP_X;
  const autoHeight =
    PAD * 2 + maxRows * NODE_H + Math.max(0, maxRows - 1) * GAP_Y;
  const manualWidth = Math.max(0, ...Array.from(pos.values()).map(p => p.x + NODE_W + PAD));
  const manualHeight = Math.max(0, ...Array.from(pos.values()).map(p => p.y + NODE_H + PAD));
  const width = Math.max(autoWidth, manualWidth);
  const height = Math.max(autoHeight, manualHeight);

  return { nodes: canvasNodes, edges, width: Math.max(width, 320), height: Math.max(height, 200) };
}

/** 生成两点间的水平贝塞尔连线 path（从左往右） */
export function edgePath(e: CanvasEdge): string {
  const cx = (e.x1 + e.x2) / 2;
  return `M ${e.x1} ${e.y1} C ${cx} ${e.y1}, ${cx} ${e.y2}, ${e.x2} ${e.y2}`;
}

/** 生成回退边的 U 形弯线路径（从节点右侧出发，向右弯曲回到目标节点右侧） */
export function loopEdgePath(e: CanvasEdge): string {
  const midX = Math.max(e.x1, e.x2) + 40;
  return `M ${e.x1} ${e.y1} C ${midX} ${e.y1}, ${midX} ${e.y2}, ${e.x2} ${e.y2}`;
}
