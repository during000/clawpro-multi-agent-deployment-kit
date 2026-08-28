/**
 * 用户端「我的申请」Mock Store
 *
 * 用途：员工侧发布 Skill / 下架 Skill 提交后，进入本地待审核队列，
 * 不写入企业技能列表 MOCK_SKILLS，也不改 Skill 类型。
 *
 * 四种状态（对齐产品定义）：
 * - pending_admin  待管理员审核（可撤回）
 * - published      已发布（可再次点击触发"更新"或"下架"流程）
 * - rejected       已驳回（可再次点击重新提交）
 * - withdrawn      已撤回（可再次点击重新提交）
 *
 * 两种类型：
 * - publish   发布申请（小字：发布申请）
 * - offshelf  下架申请（小字：下架申请）
 *
 * 卡片可点击性：
 * - pending_admin  不可点击（只能通过"撤回"按钮操作）
 * - published / rejected / withdrawn  可点击（触发上层 onCardClick 回调）
 *
 * 设计说明：仅前端 Mock，无接口；用 module-scoped state + useSyncExternalStore 订阅。
 */

import { useSyncExternalStore } from 'react';

export type MyRequestType = 'publish' | 'offshelf';

export type MyRequestStatus =
  | 'pending_admin' // 待管理员审核
  | 'published'     // 已发布
  | 'rejected'      // 已驳回
  | 'withdrawn';    // 已撤回

export interface MyRequest {
  id: string;
  type: MyRequestType;
  /** 展示用技能名称 */
  skillName: string;
  /** 技能 slug（英文短标识，展示在标题下方，参考企业技能库列表） */
  skillSlug?: string;
  /** 关联的 skill id（下架申请必填；发布申请为新提交时可选） */
  skillId?: string;
  version?: string;
  status: MyRequestStatus;
  /** 驳回原因（仅 rejected 时有值） */
  reason?: string;
  /** 员工提交下架时填写的原因 */
  offshelfReason?: string;
  submittedAt: string; // ISO 字符串
}

// ── 初始 Mock 数据（覆盖四态；含同 skill 多次事件用于查看历史） ─────
const INITIAL_REQUESTS: MyRequest[] = [
  // registrar-invoice-reconcile · 最新态 pending_admin（含一条历史 withdrawn）
  {
    id: 'req-init-1',
    type: 'publish',
    skillName: '注册商发票对账',
    skillSlug: 'registrar-invoice-reconcile',
    status: 'pending_admin',
    submittedAt: '2026-07-24T14:38:00+08:00',
  },
  {
    id: 'req-init-1a',
    type: 'publish',
    skillName: '注册商发票对账',
    skillSlug: 'registrar-invoice-reconcile',
    status: 'withdrawn',
    submittedAt: '2026-07-22T09:15:00+08:00',
  },
  // 会议纪要助手 · 最新态 published（含一条 rejected 历史）
  {
    id: 'req-init-2',
    type: 'publish',
    skillName: '会议纪要助手',
    skillSlug: 'meeting-notes-assistant',
    status: 'published',
    submittedAt: '2026-07-20T11:20:00+08:00',
  },
  {
    id: 'req-init-2a',
    type: 'publish',
    skillName: '会议纪要助手',
    skillSlug: 'meeting-notes-assistant',
    status: 'rejected',
    reason: '权限声明缺失，请补充后再提交。',
    submittedAt: '2026-07-18T16:30:00+08:00',
  },
  // 日报生成 · 最新态 rejected
  {
    id: 'req-init-3',
    type: 'publish',
    skillName: '日报生成',
    skillSlug: 'daily-report-gen',
    status: 'rejected',
    reason: '检测到疑似明文密钥，请修改后重新上传。',
    submittedAt: '2026-07-23T10:12:00+08:00',
  },
  // 周报归档 · 最新态 withdrawn
  {
    id: 'req-init-4',
    type: 'publish',
    skillName: '周报归档',
    skillSlug: 'weekly-report-archive',
    status: 'withdrawn',
    submittedAt: '2026-07-22T18:05:00+08:00',
  },
];

// ── module-scoped state ────────────────────────────────────
let requests: MyRequest[] = [...INITIAL_REQUESTS];
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

function subscribe(cb: () => void): () => void {
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
  };
}

function getSnapshot(): MyRequest[] {
  return requests;
}

// ── 状态展示映射（label） ────────────────────────────────
export const REQUEST_STATUS_LABEL: Record<MyRequestStatus, string> = {
  pending_admin: '待管理员审核',
  published: '已发布',
  rejected: '已驳回',
  withdrawn: '已撤回',
};

export const REQUEST_TYPE_LABEL: Record<MyRequestType, string> = {
  publish: '发布申请',
  offshelf: '下架申请',
};

// ── mutators ──────────────────────────────────────────────
function createId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

/**
 * 新增一条「发布申请」（员工点【发布 Skill】提交后调用）
 * 状态：pending_admin（Mock 侧不区分安全审核阶段，前端只呈现一个"待管理员审核"）
 */
export function addPublishRequest(input: {
  skillName: string;
  skillSlug?: string;
  skillId?: string;
  version?: string;
}): MyRequest {
  const req: MyRequest = {
    id: createId('req-pub'),
    type: 'publish',
    skillName: input.skillName,
    skillSlug: input.skillSlug,
    skillId: input.skillId,
    version: input.version,
    status: 'pending_admin',
    submittedAt: new Date().toISOString(),
  };
  requests = [req, ...requests];
  emit();
  return req;
}

/**
 * 新增一条「下架申请」
 * 语义：员工在"我的申请"里点已发布记录的下架按钮，新增一条 pending_admin
 * 的下架申请记录；原有的 published 发布记录保留不变。
 */
export function addOffshelfRequest(input: {
  skillId?: string;
  skillName: string;
  skillSlug?: string;
  version?: string;
  offshelfReason: string;
}): MyRequest {
  const req: MyRequest = {
    id: createId('req-off'),
    type: 'offshelf',
    skillName: input.skillName,
    skillSlug: input.skillSlug,
    skillId: input.skillId,
    version: input.version,
    status: 'pending_admin',
    offshelfReason: input.offshelfReason,
    submittedAt: new Date().toISOString(),
  };
  requests = [req, ...requests];
  emit();
  return req;
}

/**
 * 撤回一条申请（仅可撤回 pending_admin 的申请）
 * 语义：不删除记录，而是把状态置为 withdrawn 留痕，员工仍可在卡片点击重新提交。
 */
export function withdrawRequest(id: string): boolean {
  let changed = false;
  requests = requests.map((r) => {
    if (r.id === id && r.status === 'pending_admin') {
      changed = true;
      return { ...r, status: 'withdrawn' };
    }
    return r;
  });
  if (changed) {
    emit();
  }
  return changed;
}

/** 判断某 skill 是否已存在「进行中」的下架申请（用于卡片按钮置灰） */
export function hasPendingOffshelfRequest(skillId: string): boolean {
  return requests.some(
    (r) =>
      r.type === 'offshelf' &&
      r.skillId === skillId &&
      r.status === 'pending_admin',
  );
}

// ── hooks ─────────────────────────────────────────────────
export function useMyRequests(): MyRequest[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

/**
 * 汇总徽标数量：仅"待管理员审核"的申请计入（终态记录不影响徽标数字，
 * 与用户直觉一致——徽标反映"还需要我关注的事项数"）。
 */
export function useMyRequestCount(): number {
  const list = useMyRequests();
  return list.filter((r) => r.status === 'pending_admin').length;
}

/**
 * 按 skillName 查该 skill 的全量申请事件（时间倒序）
 * 用于「我的申请」卡片展开申请记录。
 */
export function useSkillRequestHistory(skillName: string): MyRequest[] {
  const list = useMyRequests();
  return list
    .filter((r) => r.skillName === skillName)
    .sort(
      (a, b) => new Date(b.submittedAt).getTime() - new Date(a.submittedAt).getTime(),
    );
}

/** 供测试或未来接入真实接口时清空 */
export function _resetForTest(next: MyRequest[] = []): void {
  requests = next;
  emit();
}
