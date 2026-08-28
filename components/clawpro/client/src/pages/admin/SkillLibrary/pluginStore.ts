/**
 * pluginStore - 企业插件库共享 Store
 * MOCK 数据由此集中维护，工具库 Tab 与「项目资产管理」共享读写。
 */
import { createLibraryStore } from './libraryStore';
import type { Plugin } from './PluginUploadDialog';

const PLUGINS_CACHE_KEY = 'pluginhub_enterprise_plugins_cache';
const PLUGINS_CACHE_VERSION_KEY = 'pluginhub_enterprise_plugins_cache_version';
const PLUGINS_CACHE_VERSION = '3';

export const PLUGIN_STORE_EVENT = 'plugin-store-updated';

export const MOCK_PLUGINS: Plugin[] = [
  {
    id: 'plugin-001',
    slug: 'code-formatter',
    name: '代码格式化插件',
    description: '自动格式化多种编程语言的代码，支持 Python、JavaScript、TypeScript、Go、Rust、Java、C++ 等主流语言。内置 Prettier、Black、gofmt 等格式化引擎，支持自定义规则配置和团队级统一风格管理。',
    version: '1.2.0',
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-08-10'),
    versions: ['1.0.0', '1.1.0', '1.2.0'],
    files: [],
  },
  {
    id: 'plugin-002',
    slug: 'data-export',
    name: '数据导出插件',
    description: '支持将对话数据导出为 CSV、Excel 格式',
    version: '1.0.0',
    scope: 'private',
    groupIds: ['grp-1'],
    uploadTime: new Date('2025-09-05'),
    versions: ['1.0.0'],
    files: [],
  },
  {
    id: 'plugin-003',
    slug: 'intelligent-doc-analyzer',
    name: '智能文档分析插件',
    description: '基于大语言模型的智能文档分析工具，支持 PDF、Word、Excel、PPT 等多种格式的深度解析。可自动提取文档摘要、关键信息、表格数据，并生成结构化报告。内置 OCR 能力，可处理扫描件和图片中的文字识别。支持多语言文档处理，包括中文、英文、日文等。适用于合同审查、财报分析、技术文档解读等企业级场景。',
    version: '2.0.1',
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-10-15'),
    versions: ['1.0.0', '2.0.0', '2.0.1'],
    files: [],
  },
  {
    id: 'plugin-004',
    slug: 'multi-cloud-deployer',
    name: '多云部署编排插件',
    description: '企业级多云部署编排工具，支持同时管理腾讯云、AWS、Azure、GCP 等主流云平台的资源。提供可视化编排界面，支持 Terraform、Pulumi 模板导入，内置蓝绿部署、金丝雀发布、滚动更新等多种发布策略。集成 CI/CD 流水线，可自动触发构建、测试和部署。支持跨云负载均衡、自动扩缩容、成本优化建议。具备完善的审计日志和回滚机制，确保生产环境安全稳定。适用于混合云架构、多区域容灾、DevOps 自动化等企业级场景。',
    version: '3.1.0',
    scope: 'private',
    groupIds: ['grp-2', 'grp-4'],
    uploadTime: new Date('2025-11-20'),
    versions: ['1.0.0', '2.0.0', '3.0.0', '3.1.0'],
    files: [],
  },
];

export const pluginStore = createLibraryStore<Plugin>({
  cacheKey: PLUGINS_CACHE_KEY,
  versionKey: PLUGINS_CACHE_VERSION_KEY,
  version: PLUGINS_CACHE_VERSION,
  initialData: MOCK_PLUGINS,
  getId: (p) => p.id,
  eventName: PLUGIN_STORE_EVENT,
  reviver: (p: any): Plugin => ({ ...p, uploadTime: new Date(p.uploadTime) }),
});
