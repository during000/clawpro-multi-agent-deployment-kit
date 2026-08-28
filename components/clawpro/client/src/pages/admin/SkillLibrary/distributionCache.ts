/**
 * 下发状态缓存管理模块
 * 统一管理列表页和详情页的下发状态，数据持久化到 localStorage
 */
import { type AgentInstance, type DistributionStatus } from './types';

// ========== 缓存 Key ==========
const DISTRIBUTION_RECORDS_KEY = 'skillhub_distribution_records';

// ========== 类型定义 ==========
export interface CachedDistributionInstance {
  id: string;
  name: string;
  createdBy: string;
  distributionStatus: DistributionStatus;
  /** 成功下发时的插件版本，用于再次打开下发窗口判断是否需要更新 */
  distributedVersion?: string;
  failReason?: string;
}

/** 记录操作类型 */
export type RecordType = 'distribute' | 'delete';

export interface CachedDistributionRecord {
  id: string;
  skillId: string;
  timestamp: string; // ISO string，方便序列化
  totalCount: number;
  successCount: number;
  failedCount: number;
  inProgressCount: number;
  status: DistributionStatus | 'deleting';
  /** 记录类型：下发 or 删除，默认 distribute（兼容旧数据） */
  type?: RecordType;
  /** 操作人 */
  operator?: string;
  instances: CachedDistributionInstance[];
}

/** 每个 skill 的下发摘要（用于列表页展示） */
export interface SkillDistributionSummary {
  lastDistributionStatus: DistributionStatus | 'deleting';
  lastDistributionProgress: number;
  lastDistributionTime: string; // ISO string
  lastDistributionInstanceCount: number;
  lastDistributionSuccessCount: number;
  hasInProgress: boolean; // 是否有进行中的任务
  lastRecordType?: RecordType; // 最新记录类型：下发 or 卸载
}

// ========== 缓存读写 ==========

/** 获取所有下发记录 */
export function getAllDistributionRecords(): CachedDistributionRecord[] {
  try {
    const cached = localStorage.getItem(DISTRIBUTION_RECORDS_KEY);
    if (cached) return JSON.parse(cached);
  } catch (e) {
    console.warn('读取下发记录缓存失败:', e);
  }
  return [];
}

/** 获取某个 skill 的所有下发记录 */
export function getDistributionRecords(skillId: string): CachedDistributionRecord[] {
  return getAllDistributionRecords().filter(r => r.skillId === skillId);
}

/**
 * 基于当前插件 ID 合并历史下发记录，生成弹窗使用的实例状态。
 * 默认清空 mock 实例上的全局下发状态，避免不同插件之间状态串联。
 */
export function getInstancesWithPluginDistributionStatus(
  pluginId: string | undefined,
  pluginVersion: string | undefined,
  baseInstances: AgentInstance[]
): AgentInstance[] {
  const resetInstanceStatus = (instance: AgentInstance): AgentInstance => ({
    ...instance,
    distributionStatus: 'not_distributed',
    distributedVersion: undefined,
    failReason: undefined,
  });

  if (!pluginId) return baseInstances.map(resetInstanceStatus);

  const records = getDistributionRecords(pluginId)
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());

  const stateByInstanceId = new Map<string, Partial<CachedDistributionInstance>>();

  records.forEach(record => {
    const recordType = record.type || 'distribute';
    record.instances.forEach(instance => {
      const prev = stateByInstanceId.get(instance.id);

      if (recordType === 'delete') {
        if (record.status === 'deleting' || instance.distributionStatus === 'distributing') {
          stateByInstanceId.set(instance.id, {
            ...prev,
            id: instance.id,
            name: instance.name,
            createdBy: instance.createdBy,
            distributionStatus: 'distributing',
            distributedVersion: prev?.distributedVersion,
          });
          return;
        }

        if (instance.distributionStatus === 'success') {
          stateByInstanceId.set(instance.id, {
            id: instance.id,
            name: instance.name,
            createdBy: instance.createdBy,
            distributionStatus: 'not_distributed',
          });
          return;
        }

        if (instance.distributionStatus === 'failed') {
          stateByInstanceId.set(instance.id, {
            ...prev,
            id: instance.id,
            name: instance.name,
            createdBy: instance.createdBy,
            distributionStatus: prev?.distributionStatus === 'success' ? 'success' : 'failed',
            distributedVersion: prev?.distributedVersion,
            failReason: instance.failReason,
          });
        }
        return;
      }

      stateByInstanceId.set(instance.id, {
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy,
        distributionStatus: instance.distributionStatus,
        distributedVersion: instance.distributionStatus === 'success'
          ? (instance.distributedVersion || pluginVersion)
          : instance.distributedVersion,
        failReason: instance.failReason,
      });
    });
  });

  return baseInstances.map(instance => {
    const base = resetInstanceStatus(instance);
    const cached = stateByInstanceId.get(instance.id);
    if (!cached || !cached.distributionStatus || cached.distributionStatus === 'not_distributed') return base;

    return {
      ...base,
      distributionStatus: cached.distributionStatus,
      distributedVersion: cached.distributedVersion,
      failReason: cached.failReason,
    };
  });
}

/** 获取当前仍安装了插件的实例（卸载弹窗和插件摘要使用） */
export function getCurrentPluginInstalledInstances(
  pluginId: string | undefined,
  pluginVersion: string | undefined,
  baseInstances: AgentInstance[]
): AgentInstance[] {
  return getInstancesWithPluginDistributionStatus(pluginId, pluginVersion, baseInstances)
    .filter(instance => instance.distributionStatus === 'success');
}


/** 保存所有下发记录 */
function saveAllDistributionRecords(records: CachedDistributionRecord[]) {
  try {
    localStorage.setItem(DISTRIBUTION_RECORDS_KEY, JSON.stringify(records));
  } catch (e) {
    console.warn('保存下发记录缓存失败:', e);
  }
}

/** 添加一条下发记录 */
export function addDistributionRecord(record: CachedDistributionRecord) {
  const all = getAllDistributionRecords();
  all.unshift(record); // 新记录放前面
  saveAllDistributionRecords(all);
  // 触发 storage 事件，通知其他组件（同页面需要自定义事件）
  window.dispatchEvent(new CustomEvent('distribution-cache-updated'));
}

/** 更新一条下发记录 */
export function updateDistributionRecord(
  recordId: string,
  updater: (record: CachedDistributionRecord) => CachedDistributionRecord
) {
  const all = getAllDistributionRecords();
  const idx = all.findIndex(r => r.id === recordId);
  if (idx !== -1) {
    all[idx] = updater(all[idx]);
    saveAllDistributionRecords(all);
    window.dispatchEvent(new CustomEvent('distribution-cache-updated'));
  }
}

/** 获取某个 skill 的下发摘要（用于列表页） */
export function getSkillDistributionSummary(skillId: string): SkillDistributionSummary | null {
  const records = getDistributionRecords(skillId);
  if (records.length === 0) return null;

  // 最新一条记录
  const latest = records[0];
  const hasInProgress = records.some(r => r.status === 'distributing' || r.status === 'deleting');

  const progress = latest.totalCount > 0
    ? Math.round((latest.successCount / latest.totalCount) * 100)
    : 0;

  return {
    lastDistributionStatus: latest.status,
    lastDistributionProgress: progress,
    lastDistributionTime: latest.timestamp,
    lastDistributionInstanceCount: latest.totalCount,
    lastDistributionSuccessCount: latest.successCount,
    hasInProgress,
    lastRecordType: latest.type,
  };
}

/** 获取插件列表页展示用下发摘要：计数与企业技能库一致，使用最新操作记录的 success/total */
export function getPluginDistributionSummary(
  pluginId: string,
  pluginVersion: string | undefined,
  baseInstances: AgentInstance[]
): SkillDistributionSummary | null {
  const records = getDistributionRecords(pluginId);
  if (records.length === 0) return null;

  const inProgressRecord = records.find(r => r.status === 'distributing' || r.status === 'deleting');
  if (inProgressRecord) {
    const progress = inProgressRecord.totalCount > 0
      ? Math.round((inProgressRecord.successCount / inProgressRecord.totalCount) * 100)
      : 0;
    return {
      lastDistributionStatus: inProgressRecord.status,
      lastDistributionProgress: progress,
      lastDistributionTime: inProgressRecord.timestamp,
      lastDistributionInstanceCount: inProgressRecord.totalCount,
      lastDistributionSuccessCount: inProgressRecord.successCount,
      hasInProgress: true,
      lastRecordType: inProgressRecord.type || 'distribute',
    };
  }

  const latest = records[0];
  const latestType = latest.type || 'distribute';
  const installedInstances = getCurrentPluginInstalledInstances(pluginId, pluginVersion, baseInstances);
  if (installedInstances.length > 0 || (latestType === 'distribute' && latest.failedCount > 0)) {
    const progress = latest.totalCount > 0
      ? Math.round((latest.successCount / latest.totalCount) * 100)
      : 0;
    return {
      lastDistributionStatus: latest.status,
      lastDistributionProgress: progress,
      lastDistributionTime: latest.timestamp,
      lastDistributionInstanceCount: latest.totalCount,
      lastDistributionSuccessCount: latest.successCount,
      hasInProgress: false,
      lastRecordType: latestType,
    };
  }

  return null;
}



// ========== 别名导出（兼容 Skill 命名引用）==========
/** 别名：getCurrentPluginInstalledInstances 的 Skill 命名版本 */
export const getCurrentSkillInstalledInstances = getCurrentPluginInstalledInstances;
/** 别名：getInstancesWithPluginDistributionStatus 的 Skill 命名版本 */
export const getInstancesWithSkillDistributionStatus = getInstancesWithPluginDistributionStatus;

/** 检查某个 skill 是否有进行中的下发或删除任务 */
export function hasInProgressDistribution(skillId: string): boolean {
  const records = getDistributionRecords(skillId);
  return records.some(r => r.status === 'distributing' || r.status === 'deleting');
}

/** 创建一个新的下发记录 ID */
export function createDistributionRecordId(): string {
  return 'dist-' + Date.now() + '-' + Math.random().toString(36).slice(2, 6);
}

/**
 * 初始化预设下发记录（仅当 localStorage 为空时）。
 * 预设部分技能有不同下发状态，其余保持未下发以展示置灰按钮。
 */
export function initMockDistributionRecords() {
  const existing = getAllDistributionRecords();
  if (existing.length > 0) return; // 已有数据则跳过

  const now = new Date();

  const presetRecords: CachedDistributionRecord[] = [
    // skill-6 K8s 故障排查助手 — 下发成功
    {
      id: 'dist-preset-1',
      skillId: 'skill-6',
      timestamp: new Date(now.getTime() - 2 * 3600_000).toISOString(),
      totalCount: 3,
      successCount: 3,
      failedCount: 0,
      inProgressCount: 0,
      status: 'success',
      operator: 'yequanzheng',
      instances: [
        { id: 'inst-1', name: '产品助手', createdBy: 'admin', distributionStatus: 'success' },
        { id: 'inst-2', name: '研发助手', createdBy: 'admin', distributionStatus: 'success' },
        { id: 'inst-3', name: '运营助手', createdBy: 'admin', distributionStatus: 'success' },
      ],
    },
    // skill-5 SQL 查询优化助手 — 下发失败
    {
      id: 'dist-preset-2',
      skillId: 'skill-5',
      timestamp: new Date(now.getTime() - 5 * 3600_000).toISOString(),
      totalCount: 2,
      successCount: 0,
      failedCount: 2,
      inProgressCount: 0,
      status: 'failed',
      operator: 'yequanzheng',
      instances: [
        { id: 'inst-1', name: '产品助手', createdBy: 'admin', distributionStatus: 'failed', failReason: '实例不可用' },
        { id: 'inst-2', name: '研发助手', createdBy: 'admin', distributionStatus: 'failed', failReason: '网络超时' },
      ],
    },
    // skill-4 GitHub 集成 — 下发成功（部分）
    {
      id: 'dist-preset-3',
      skillId: 'skill-4',
      timestamp: new Date(now.getTime() - 8 * 3600_000).toISOString(),
      totalCount: 2,
      successCount: 1,
      failedCount: 1,
      inProgressCount: 0,
      status: 'failed',
      operator: 'yequanzheng',
      instances: [
        { id: 'inst-1', name: '产品助手', createdBy: 'admin', distributionStatus: 'success' },
        { id: 'inst-2', name: '研发助手', createdBy: 'admin', distributionStatus: 'failed', failReason: '版本冲突' },
      ],
    },
  ];

  // 逐条写入
  presetRecords.forEach(r => {
    const all = getAllDistributionRecords();
    all.unshift(r);
    try {
      localStorage.setItem(DISTRIBUTION_RECORDS_KEY, JSON.stringify(all));
    } catch (e) {
      console.warn('初始化预设下发记录失败:', e);
    }
  });
}
