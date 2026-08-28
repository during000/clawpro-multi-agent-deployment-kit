/**
 * standardsStore - 企业规范库共享 Store
 * 集中维护 AgentConfigAsset 类型与 MOCK 数据，供 StandardsLibraryTab 与「项目资产管理」共享读写。
 */
import { createLibraryStore } from './libraryStore';
import type { SkillScope } from './types';

export type AssetKind = 'entry' | 'rule' | 'hook';
export type TargetClient = 'claude_code' | 'codebuddy' | 'codex' | 'workbuddy';
export type DeliveryTaskStatus = 'pending' | 'running' | 'installed' | 'unsupported' | 'skipped' | 'failed';

export interface AgentConfigAsset {
  id: string;
  tenantId: string;
  name: string;
  slug: string;
  kind: AssetKind;
  targetClients: TargetClient[];
  contentMd: string;
  fileName?: string;
  hookCount?: number;
  description?: string;
  version: string;
  visibilityType: 'all' | 'group';
  scope: SkillScope;
  groupIds: string[];
  enabled: boolean;
  alwaysApply: boolean;
  pathGlobs: string[];
  checksum: string;
  createdBy: string;
  updatedAt: Date;
  lastTaskStatus: DeliveryTaskStatus;
}

export const ENTRY_CLIENTS: TargetClient[] = ['claude_code', 'codebuddy', 'codex'];
export const RULE_CLIENTS: TargetClient[] = ['claude_code', 'codebuddy', 'workbuddy'];

const STANDARDS_CACHE_KEY = 'standardshub_enterprise_assets_cache';
const STANDARDS_CACHE_VERSION_KEY = 'standardshub_enterprise_assets_cache_version';
const STANDARDS_CACHE_VERSION = '4';

export const STANDARDS_STORE_EVENT = 'standards-store-updated';

export const MOCK_ASSETS: AgentConfigAsset[] = [
  {
    id: 'asset-hook-001',
    tenantId: 'tenant-openclaw',
    name: 'update-awareness 发布门禁',
    slug: 'update-awareness-gate',
    kind: 'hook',
    targetClients: ['codebuddy'],
    contentMd: 'hooks:\n  - id: update-awareness-gate\n    description: "发布操作前执行 update-awareness 检查"\n    event: PreToolUse\n    tools: [codebuddy]\n    timeout: 15\n    command: |\n      ruby "${CODEBUDDY_PROJECT_DIR}/.codebuddy/skills/update-awareness/scripts/update-awareness-ask-hook.rb"\n',
    fileName: 'hooks.yaml',
    hookCount: 1,
    description: '发布操作前执行 update-awareness 检查，按用户范围下发到本地 Agent。',
    version: '1.0.0',
    visibilityType: 'all',
    scope: 'public',
    groupIds: [],
    enabled: true,
    alwaysApply: true,
    pathGlobs: [],
    checksum: 'sha256:4a61cf',
    createdBy: '平台管理员',
    updatedAt: new Date('2026-07-14'),
    lastTaskStatus: 'pending',
  },
  {
    id: 'asset-entry-001',
    tenantId: 'tenant-openclaw',
    name: '前端项目 CLAUDE.md',
    slug: 'frontend-project-entry',
    kind: 'entry',
    targetClients: ENTRY_CLIENTS,
    contentMd: '# 前端项目 CLAUDE.md\n- 优先遵循企业组件规范。\n- 修改后运行类型检查。',
    description: '前端项目全局工作准则',
    version: '1.3.0',
    visibilityType: 'group',
    scope: 'private',
    groupIds: ['grp-2'],
    enabled: true,
    alwaysApply: true,
    pathGlobs: [],
    checksum: 'sha256:7c81a2',
    createdBy: '平台管理员',
    updatedAt: new Date('2026-06-26'),
    lastTaskStatus: 'installed',
  },
  {
    id: 'asset-rule-001',
    tenantId: 'tenant-openclaw',
    name: '前端 React 规范',
    slug: 'frontend-react-rules',
    kind: 'rule',
    targetClients: RULE_CLIENTS,
    contentMd: '# 前端 React 规范\n- 使用函数组件。\n- 修改后运行类型检查。',
    description: '前端 React 开发规范',
    version: '1.4.0',
    visibilityType: 'group',
    scope: 'private',
    groupIds: ['grp-2'],
    enabled: true,
    alwaysApply: false,
    pathGlobs: ['src/**/*.{ts,tsx}'],
    checksum: 'sha256:21b9ef',
    createdBy: '前端工程委员会',
    updatedAt: new Date('2026-06-22'),
    lastTaskStatus: 'pending',
  },
  {
    id: 'asset-rule-002',
    tenantId: 'tenant-openclaw',
    name: 'Agent 安全合规基线',
    slug: 'agent-security-baseline',
    kind: 'rule',
    targetClients: RULE_CLIENTS,
    contentMd: '# Agent 安全合规基线\n- 不输出密钥。\n- 执行命令前说明影响范围。',
    description: 'Agent 安全合规基线',
    version: '1.1.0',
    visibilityType: 'all',
    scope: 'public',
    groupIds: [],
    enabled: true,
    alwaysApply: true,
    pathGlobs: [],
    checksum: 'sha256:92dc10',
    createdBy: '安全运营团队',
    updatedAt: new Date('2026-06-18'),
    lastTaskStatus: 'installed',
  },
  {
    id: 'asset-entry-002',
    tenantId: 'tenant-openclaw',
    name: '交付协作 CLAUDE.md',
    slug: 'delivery-collaboration-entry',
    kind: 'entry',
    targetClients: ENTRY_CLIENTS,
    contentMd: '# 交付协作 CLAUDE.md\n- 先澄清目标，再拆解任务。\n- 变更需说明风险与验证方式。',
    description: '交付协作全局工作准则',
    version: '1.0.0',
    visibilityType: 'all',
    scope: 'public',
    groupIds: [],
    enabled: true,
    alwaysApply: true,
    pathGlobs: [],
    checksum: 'sha256:3180ad',
    createdBy: '项目管理办公室',
    updatedAt: new Date('2026-06-12'),
    lastTaskStatus: 'pending',
  },
];

export const standardsStore = createLibraryStore<AgentConfigAsset>({
  cacheKey: STANDARDS_CACHE_KEY,
  versionKey: STANDARDS_CACHE_VERSION_KEY,
  version: STANDARDS_CACHE_VERSION,
  initialData: MOCK_ASSETS,
  getId: (a) => a.id,
  eventName: STANDARDS_STORE_EVENT,
  reviver: (a: any): AgentConfigAsset => {
    const asset = { ...a, updatedAt: new Date(a.updatedAt) } as AgentConfigAsset;
    if (asset.id === 'asset-hook-001' && asset.kind === 'hook' && asset.fileName === 'hooks.yaml') {
      const currentExample = MOCK_ASSETS[0];
      return {
        ...asset,
        contentMd: currentExample.contentMd,
        fileName: currentExample.fileName,
        description: currentExample.description,
        createdBy: currentExample.createdBy,
      };
    }
    return asset;
  },
});
