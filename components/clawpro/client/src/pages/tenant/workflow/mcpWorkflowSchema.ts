/**
 * mcpWorkflowSchema —— agent 经 MCP 回传工作流的受信结构与安全校验
 *
 * 背景（P3）：ClawPro 自身也是 AI-Native——允许 agent 通过 MCP 把「画好的工作流」以 JSON
 * 回传到项目，落库为流水线模板。由于 JSON 来自不受信来源，必须严格校验后再落库：
 *  - 仅接受白名单字段（name/description/nodes[]{id,title,agentRole,dependsOn[]}）；
 *  - 校验类型与长度、限制节点数上限、拒绝原型污染键；
 *  - 校验 dependsOn 引用合法（不悬空、不自环、无环）。
 * 校验失败返回可操作的错误原因，绝不落库脏数据。
 */
import type {
  TenantPipelineTemplateNode,
} from '../tenantProjectStore';

/** 受信的 MCP 工作流节点结构 */
export interface McpWorkflowNode {
  id: string;
  title: string;
  agentRole: string;
  dependsOn: string[];
}

/** 受信的 MCP 工作流载荷结构 */
export interface McpWorkflowPayload {
  name: string;
  description: string;
  nodes: McpWorkflowNode[];
}

export type ParseResult =
  | { ok: true; data: McpWorkflowPayload }
  | { ok: false; error: string };

/** 约束上限，防止超大/恶意输入 */
const MAX_NODES = 50;
const MAX_STR = 200;
const MAX_ID = 64;
/** 原型污染保护：禁止出现的危险键 */
const FORBIDDEN_KEYS = new Set(['__proto__', 'prototype', 'constructor']);

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

/** 安全字符串：非空、去空白、限长 */
function safeString(v: unknown): string | null {
  if (typeof v !== 'string') return null;
  const trimmed = v.trim();
  if (!trimmed || trimmed.length > MAX_STR) return null;
  return trimmed;
}

/** 校验 id：仅允许字母数字、连字符、下划线，限长 */
function safeId(v: unknown): string | null {
  if (typeof v !== 'string') return null;
  const trimmed = v.trim();
  if (!trimmed || trimmed.length > MAX_ID) return null;
  if (FORBIDDEN_KEYS.has(trimmed)) return null;
  if (!/^[A-Za-z0-9_-]+$/.test(trimmed)) return null;
  return trimmed;
}

/**
 * 解析并严格校验 MCP 回传的工作流 JSON 字符串或对象。
 * @param raw JSON 字符串或已解析对象
 */
export function parseMcpWorkflow(raw: unknown): ParseResult {
  // 1. 解析 JSON 字符串
  let obj: unknown = raw;
  if (typeof raw === 'string') {
    const text = raw.trim();
    if (!text) return { ok: false, error: '内容为空，请粘贴工作流 JSON' };
    try {
      obj = JSON.parse(text);
    } catch {
      return { ok: false, error: 'JSON 解析失败，请检查格式是否正确' };
    }
  }

  if (!isPlainObject(obj)) {
    return { ok: false, error: '顶层必须是一个 JSON 对象' };
  }

  // 2. name / description
  const name = safeString(obj.name);
  if (!name) {
    return { ok: false, error: 'name 字段缺失或非法（1-200 字符）' };
  }
  const description =
    obj.description === undefined ? '' : safeString(obj.description) ?? '';

  // 3. nodes 数组
  if (!Array.isArray(obj.nodes)) {
    return { ok: false, error: 'nodes 必须是数组' };
  }
  if (obj.nodes.length === 0) {
    return { ok: false, error: 'nodes 不能为空，至少一个节点' };
  }
  if (obj.nodes.length > MAX_NODES) {
    return { ok: false, error: `节点数超出上限（最多 ${MAX_NODES} 个）` };
  }

  // 4. 逐节点校验，收集合法 id
  const nodes: McpWorkflowNode[] = [];
  const idSet = new Set<string>();
  for (let i = 0; i < obj.nodes.length; i++) {
    const n = obj.nodes[i];
    if (!isPlainObject(n)) {
      return { ok: false, error: `nodes[${i}] 不是对象` };
    }
    const id = safeId(n.id);
    if (!id) {
      return {
        ok: false,
        error: `nodes[${i}].id 非法（仅字母/数字/-/_，≤64 字符）`,
      };
    }
    if (idSet.has(id)) {
      return { ok: false, error: `节点 id 重复：${id}` };
    }
    const title = safeString(n.title);
    if (!title) {
      return { ok: false, error: `nodes[${i}].title 缺失或非法（1-200 字符）` };
    }
    const agentRole = n.agentRole === undefined ? '成员' : safeString(n.agentRole);
    if (!agentRole) {
      return { ok: false, error: `nodes[${i}].agentRole 非法（1-200 字符）` };
    }
    // dependsOn
    let dependsOn: string[] = [];
    if (n.dependsOn !== undefined) {
      if (!Array.isArray(n.dependsOn)) {
        return { ok: false, error: `nodes[${i}].dependsOn 必须是数组` };
      }
      const deps: string[] = [];
      for (const d of n.dependsOn) {
        const depId = safeId(d);
        if (!depId) {
          return { ok: false, error: `nodes[${i}].dependsOn 含非法 id` };
        }
        if (depId === id) {
          return { ok: false, error: `节点 ${id} 不能依赖自身` };
        }
        if (!deps.includes(depId)) deps.push(depId);
      }
      dependsOn = deps;
    }
    idSet.add(id);
    nodes.push({ id, title, agentRole, dependsOn });
  }

  // 5. dependsOn 引用必须指向已存在节点（不悬空）
  for (const n of nodes) {
    for (const d of n.dependsOn) {
      if (!idSet.has(d)) {
        return { ok: false, error: `节点 ${n.id} 依赖了不存在的节点：${d}` };
      }
    }
  }

  // 6. 环检测（DFS）
  if (hasCycle(nodes)) {
    return { ok: false, error: '工作流存在循环依赖，请修正 dependsOn' };
  }

  return { ok: true, data: { name, description, nodes } };
}

/** DAG 环检测：任一节点经 dependsOn 能回到自身即为有环 */
function hasCycle(nodes: McpWorkflowNode[]): boolean {
  const map = new Map(nodes.map((n) => [n.id, n.dependsOn]));
  const state = new Map<string, 0 | 1 | 2>(); // 0=未访问 1=在栈中 2=完成
  const visit = (id: string): boolean => {
    const s = state.get(id) ?? 0;
    if (s === 1) return true; // 回到栈中节点 → 有环
    if (s === 2) return false;
    state.set(id, 1);
    for (const dep of map.get(id) ?? []) {
      if (visit(dep)) return true;
    }
    state.set(id, 2);
    return false;
  };
  for (const n of nodes) {
    if (visit(n.id)) return true;
  }
  return false;
}

/** 把受信的 MCP 节点转为项目流水线模板节点结构 */
export function toTemplateNodes(
  payload: McpWorkflowPayload,
): TenantPipelineTemplateNode[] {
  return payload.nodes.map((n) => ({
    id: n.id,
    title: n.title,
    dependsOn: [...n.dependsOn],
    agentRole: n.agentRole,
  }));
}

/** 示例 JSON（供画布「MCP 回传指引」展示，可复制） */
export const MCP_WORKFLOW_EXAMPLE = JSON.stringify(
  {
    name: '需求交付流水线',
    description: '由 agent 经 MCP 回传的标准需求交付流程',
    nodes: [
      { id: 'n1', title: '需求分析', agentRole: '产品', dependsOn: [] },
      { id: 'n2', title: '方案设计', agentRole: '后端', dependsOn: ['n1'] },
      { id: 'n3', title: '前端开发', agentRole: '前端', dependsOn: ['n2'] },
      { id: 'n4', title: '后端开发', agentRole: '后端', dependsOn: ['n2'] },
      { id: 'n5', title: '联调测试', agentRole: '测试', dependsOn: ['n3', 'n4'] },
    ],
  },
  null,
  2,
);
