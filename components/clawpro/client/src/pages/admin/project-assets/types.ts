/**
 * 「项目资产管理」核心类型定义
 *
 * 设计背景：从「组织」视角配置该组织的 Agent 资产合集（模型配置/公共技能/企业技能/企业插件/企业MCP/企业规范），
 * 不区分云端/本地 Agent。管理员为每个资产大类选择该组织的同步模式：
 *  - initial：仅作为该组织下所有 Agent 首次开启时的初始配置
 *  - autoSync：始终与 Agent 工具库保持同步（工具库新增/更新/删除该类资产时自动联动）
 */

/** 资产大类：模型配置 + Agent 工具库的 5 个配置项大类 */
export type AssetCategory =
  | 'modelConfig'
  | 'publicSkill'
  | 'enterpriseSkill'
  | 'enterprisePlugin'
  | 'enterpriseMcp'
  | 'enterpriseStandard';

/** 资产大类的展示信息 */
export const ASSET_CATEGORY_MAP: Record<AssetCategory, { label: string; description: string }> = {
  modelConfig: { label: '模型配置', description: '来自模型配置页，按应用范围选择可用模型' },
  publicSkill: { label: '公共技能', description: '来自公共技能市场，勾选后加入该组织资产' },
  enterpriseSkill: { label: '企业技能', description: '企业内部发布的技能' },
  enterprisePlugin: { label: '企业插件', description: '企业内部发布的插件' },
  enterpriseMcp: { label: '企业 MCP', description: '企业内部发布的 MCP 服务' },
  enterpriseStandard: { label: '企业规范', description: '企业内部发布的系统提示词/规范' },
};

/** 资产大类的顺序（用于 UI 渲染） */
export const ASSET_CATEGORY_ORDER: AssetCategory[] = [
  'modelConfig',
  'publicSkill',
  'enterpriseSkill',
  'enterprisePlugin',
  'enterpriseMcp',
  'enterpriseStandard',
];

/**
 * 同步模式（组织级整体开关，作用于该组织的全部项目资产）：
 * - initial（仅初始配置）：仅作为该组织下「新建 Agent」首次创建时的初始配置；
 *   之后修改项目资产不影响已创建的存量 Agent，只影响后续新建的 Agent。
 * - autoSync（始终同步）：该组织下所有 Agent（新增 + 存量）始终保持安装全部项目资产；
 *   项目资产更新/新增，保存后立即下发所有 Agent（删除不下发，即不从 Agent 卸载）。
 */
export type AssetSyncMode = 'initial' | 'autoSync';

/** 同步模式展示信息 */
export const ASSET_SYNC_MODE_MAP: Record<AssetSyncMode, { label: string; description: string }> = {
  initial: {
    label: '仅作为新增实例初始配置',
    description: '每次资产编辑保存后，资产仅作为此后新建实例的初始配置，已创建的实例保持原配置不变',
  },
  autoSync: {
    label: '所有实例始终同步更新',
    description: '每次资产编辑保存后，自动下发最新资产到该组织下的全部实例（含已创建和新创建的实例）',
  },
};

/** 单个资产项引用：指向对应资产 Store（skillStore/pluginStore/mcpStore/standardsStore/publicSkillStore）内的条目 */
export interface ProjectAssetItemRef {
  /** 指向资产 Store 内的唯一 id（Skill.id / Plugin.id / MCPService.name / AgentConfigAsset.id / public-skill 资源 id） */
  refId: string;
  /** 加入/上次确认更新时的版本快照，用于比对是否需要「版本更新提醒」 */
  versionAtBind: string;
  /** 加入该组织资产的时间 */
  addedAt: string;
}

/** 单个资产大类在某组织下的配置（同步模式已上提为组织级，此处仅保存已选资产） */
export interface ProjectAssetCategoryConfig {
  items: ProjectAssetItemRef[];
}

/** 某组织的完整项目资产配置 */
export interface ProjectAssetConfig {
  groupId: string;
  /** 组织级整体同步模式（作用于该组织全部 6 大类资产） */
  mode: AssetSyncMode;
  categories: Record<AssetCategory, ProjectAssetCategoryConfig>;
  /** 版本号，每次保存 +1 */
  version: number;
  updatedAt: string;
  updatedBy?: string;
}

/**
 * 更新记录的展示用 tag 类别（只有两种，用颜色区分）：
 * - manual：手动编辑（管理员自己编辑并保存）
 * - auto：自动同步（Agent 工具库有变更，自动同步到资产管理）
 */
export type ProjectAssetTagKind = 'manual' | 'auto';

/**
 * 一条更新记录内的「单个操作段落」：一行主文案 + 下方明细小字。
 * 一次手动保存可能同时做多个操作（新增 + 删除 + 改同步模式），
 * 因此一条记录可包含多个 section，每个 section 各占一行主文案。
 */
export interface ProjectAssetChangeSection {
  /** 主文案，如「新增 2 项资产」「删除 1 项资产」「同步模式修改为「所有实例始终同步更新」」 */
  title: string;
  /** 明细小字，如「企业技能：知识库问答」「企业规范：数据安全规范」；无明细时省略 */
  items?: string[];
}

/** 项目资产更新记录（版本历史） */
export interface ProjectAssetUpdateRecord {
  id: string;
  groupId: string;
  version: number;
  /** 展示用 tag：manual=手动编辑（蓝）/ auto=自动同步（灰） */
  tagKind: ProjectAssetTagKind;
  /** 变更操作段落列表：每个操作一行主文案 + 明细小字 */
  sections: ProjectAssetChangeSection[];
  operator?: string;
  createdAt: string;
}

/** 用于渲染"版本更新提醒"的漂移信息 */
export interface AssetVersionDrift {
  category: AssetCategory;
  refId: string;
  boundVersion: string;
  latestVersion: string;
}

/** 生成某组织的空初始配置（6大类皆为空，组织级 mode 默认 'initial'） */
export function createEmptyProjectAssetConfig(groupId: string): ProjectAssetConfig {
  const categories = ASSET_CATEGORY_ORDER.reduce((acc, category) => {
    acc[category] = { items: [] };
    return acc;
  }, {} as Record<AssetCategory, ProjectAssetCategoryConfig>);

  return {
    groupId,
    mode: 'initial',
    categories,
    version: 0,
    updatedAt: new Date().toISOString(),
  };
}
