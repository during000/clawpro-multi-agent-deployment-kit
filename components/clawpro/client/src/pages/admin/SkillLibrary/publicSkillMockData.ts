/**
 * 公共技能库 & 技能初始包 Mock 数据
 */

export interface PublicSkill {
  id: string;
  slug: string;
  name: string;
  nameZh: string;
  description: string;
  descriptionZh: string;
  downloads: number;
  stars: number;
  version: string;
  category: string; // category slug
  tags: string[];
  files: PublicSkillFile[];
  versions: PublicSkillVersion[];
}

export interface PublicSkillFile {
  name: string;
  path: string;
  type: 'file' | 'folder';
  children?: PublicSkillFile[];
  content?: string;
}

export interface PublicSkillVersion {
  version: string;
  date: string;
  isLatest: boolean;
}

export interface FavoriteSkill {
  skillId: string;
  tags: string[]; // user-assigned tags
  addedAt: Date;
}

export interface SkillInitialPackage {
  id: string;
  name: string;
  scope: string; // 应用范围显示文本
  scopeType: 'public' | 'private'; // 应用范围类型：public=全部用户, private=按组织
  groupIds: string[]; // 当 scopeType='private' 时，关联的组织 ID 列表
  isActive: boolean; // 是否生效
  hasDraft: boolean; // 是否有未发布修改
  skills: PackageSkillItem[];
  createdAt: Date;
  updatedAt: Date;
}

export interface PackageSkillItem {
  skillId: string;
  skillName: string;
  skillNameZh?: string;
  source: 'public' | 'enterprise'; // 来源
  /** 当公共技能由公共技能包展开添加时，记录来源技能包 */
  sourcePackages?: Array<{ id: string; name: string }>;
  version: string;
  originalVersion?: string; // 刷新前的原始版本号（仅在版本被刷新后存在）
  addedAt: Date;
}

// ─── 公共技能 Mock 数据 ───────────────────────────────────────────────────────

export const PUBLIC_SKILL_CATEGORIES = [
  { id: 'all', name: '全部', slug: 'all' },
  { id: 'favorites', name: '我的收藏', slug: 'favorites' },
  { id: 'featured', name: '精选', slug: 'featured' },
  { id: 'ai', name: 'AI 智能', slug: 'ai' },
  { id: 'dev-tools', name: '开发工具', slug: 'dev-tools' },
  { id: 'efficiency', name: '效率提升', slug: 'efficiency' },
  { id: 'data-analysis', name: '数据分析', slug: 'data-analysis' },
  { id: 'content', name: '内容创作', slug: 'content' },
  { id: 'security', name: '安全合规', slug: 'security' },
  { id: 'communication', name: '通讯协作', slug: 'communication' },
];

const SKILL_CONTENT_SELF_IMPROVING = `---
name: self-improving-agent
description: "记录错误、纠正、能力缺口与最佳实践，形成可复用的持续改进闭环。适用于：命令失败、用户纠正、外部 API 失败"
---

# Self-Improving Agent（自我改进）

把"踩坑"变成"资产"。
每次失败、纠正或新发现，都写入结构化记录，后续可检索、可复盘、可沉淀到长期规则。

---

## 1. 什么时候必须记录

出现以下情况时，立即记录：

1. **命令/操作失败**（非 0 退出、超时、异常输出）
2. **用户纠正你**（"不对""应该是…"）
3. **用户提出你当前不具备的能力**
4. **外部服务失败**（API 报错、限流、鉴权失败）
5. **你意识到知识已过时或理解错误**
6. **你发现了可复用的更优流程**

---

## 2. 记录到哪里

在工作区使用 \`.learnings/\` 目录：

- \`.learnings/LEARNINGS.md\`：纠正、认知缺口、最佳实践
- \`.learnings/ERRORS.md\`：失败与异常
- \`.learnings/FEATURE_REQUESTS.md\`：能力请求

若目录不存在，先创建：

\`\`\`bash
mkdir -p .learnings
\`\`\`

写入记录示例（超长命令，可横向滚动查看）：

\`\`\`bash
python3 scripts/record.py --type error --trigger "API timeout" --cause "External service rate limit exceeded after 3 retries" --solution "Added exponential backoff with jitter, max 5 retries" --prevention "Monitor rate limit headers and implement circuit breaker pattern" --file .learnings/ERRORS.md
\`\`\`

技能目录结构如下：

\`\`\`
self-improving-agent/
├── SKILL.md （工作流程 + 触发条件）
├── hooks/
│   └── post-run.sh （每次运行后自动记录）
├── scripts/
│   └── record.py （结构化写入工具）└── _meta.json （技能元数据）
\`\`\`

---

## 3. 记录格式规范

每条记录应包含以下要素（原点列表示例）：

- **触发事件**：描述触发记录的具体情况
- **根本原因**：分析问题的根本原因
- **解决方案**：记录最终采用的解决方案
- **预防措施**：下次如何避免同类问题
- **关联文件**：涉及的代码文件或配置路径

写入记录的推荐步骤（数字列表示例）：

1. 确认触发条件已满足（参考第 1 节）
2. 选择对应的目标文件（\`LEARNINGS.md\` / \`ERRORS.md\`）
3. 按格式填写触发事件和根本原因
4. 补充解决方案和预防措施
5. 保存文件，确认写入成功


---

## 5. 文件索引

各记录类型对应的目标文件：

| 记录类型 | 目标文件 |
|----------|----------|
| 命令/操作失败 | \`.learnings/ERRORS.md\` |
| 用户纠正与认知缺口 | \`.learnings/LEARNINGS.md\` |
| 能力请求与新功能 | \`.learnings/FEATURE_REQUESTS.md\` |
| 外部服务失败 | \`.learnings/ERRORS.md\` |
| 可复用的优化流程 | \`.learnings/LEARNINGS.md\` |
| 知识过时或理解错误 | \`.learnings/LEARNINGS.md\` |

---

## 4. 参考资料

- [OpenClaw 官方文档](https://docs.openclaw.com/skills/self-improving)
- [最佳实践指南](https://clawhub.openclaw.com/skills/self-improving-agent)

---

## 6. 任务列表（Task Lists）

**项目进度**

- [x] 项目初始化
- [x] 数据库设计
- [x] API 接口开发
- [ ] 前端页面开发
- [ ] 单元测试编写
- [ ] 集成测试
- [ ] 部署上线

**日常待办**

- [x] 晨会讨论
- [x] 代码审查
- [ ] 编写技术文档
- [ ] 修复 Bug #1234
- [ ] 性能优化

---

## 7. 引用（Blockquote）

单行引用：

> 把"踩坑"变成"资产"，每次失败都是下一次成功的基石。

多行引用：

> **最佳实践**
>
> 每次运行结束后，立即记录本次的关键发现。
> 不要等到"有空了再整理"——那一天往往不会来。

嵌套引用：

> 外层引用：这是一条重要的设计原则。
>
> > 内层引用：该原则最早由 Kent Beck 在《Extreme Programming Explained》中提出。
`;

const SKILL_CONTENT_GITHUB = `---
name: github
description: "Interact with GitHub using the gh CLI. Use gh issue, gh pr, gh run, and gh api for issues, PRs, CI runs, and advanced queries."
---

# GitHub Skill

Use the \`gh\` CLI to interact with GitHub. Always specify \`--repo owner/repo\` when not in a git directory, or use URLs directly.

## Pull Requests

Check CI status on a PR:
\`\`\`bash
gh pr checks 55 --repo owner/repo
\`\`\`

List recent workflow runs:
\`\`\`bash
gh run list --repo owner/repo --limit 10
\`\`\`

View a run and see which steps failed:
\`\`\`bash
gh run view <run-id> --repo owner/repo
\`\`\`

## API for Advanced Queries

The \`gh api\` command is useful for accessing data not available through other subcommands.
`;

export const PUBLIC_SKILLS: PublicSkill[] = [
  {
    id: 'pub-1',
    slug: 'self-improving-agent',
    name: 'self-improving-agent',
    nameZh: '自我改进代理',
    description: 'Records errors, corrections, capability gaps and best practices to form a reusable continuous improvement loop.',
    descriptionZh: '记录错误、纠正、能力缺口与最佳实践，形成可复用的持续改进闭环。',
    downloads: 12300,
    stars: 4200,
    version: '1.0.0',
    category: 'dev-tools',
    tags: ['agent', 'learning', 'self-improvement'],
    versions: [
      { version: '1.0.0', date: '2026-03-22 22:56:11', isLatest: true },
    ],
    files: [
      {
        name: 'assets', path: 'assets', type: 'folder',
        children: [
          { name: 'logo.png', path: 'assets/logo.png', type: 'file', content: '（二进制图片文件）' },
        ]
      },
      {
        name: 'hooks', path: 'hooks', type: 'folder',
        children: [
          { name: 'post-run.sh', path: 'hooks/post-run.sh', type: 'file', content: '#!/bin/bash\n# Post-run hook\necho "Run completed"' },
        ]
      },
      {
        name: 'scripts', path: 'scripts', type: 'folder',
        children: [
          { name: 'record.py', path: 'scripts/record.py', type: 'file', content: '# Record learning script\nimport json\n\ndef record_learning(entry):\n    pass' },
        ]
      },
      { name: '_meta.json', path: '_meta.json', type: 'file', content: '{\n  "name": "self-improving-agent",\n  "version": "1.0.0"\n}' },
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: SKILL_CONTENT_SELF_IMPROVING },
    ],
  },
  {
    id: 'pub-2',
    slug: 'github',
    name: 'github',
    nameZh: 'GitHub 工具',
    description: 'Interact with GitHub using the gh CLI for issues, PRs, CI runs, and advanced queries.',
    descriptionZh: '使用 gh CLI 与 GitHub 交互，支持 Issue、PR、CI 运行和高级查询。',
    downloads: 9800,
    stars: 3100,
    version: '2.1.0',
    category: 'dev-tools',
    tags: ['github', 'cli', 'devops'],
    versions: [
      { version: '2.1.0', date: '2026-03-20 10:00:00', isLatest: true },
      { version: '2.0.0', date: '2026-02-15 08:30:00', isLatest: false },
      { version: '1.5.0', date: '2026-01-10 14:20:00', isLatest: false },
    ],
    files: [
      {
        name: 'hha', path: 'hha', type: 'folder',
        children: [
          { name: 'ha.md', path: 'hha/ha.md', type: 'file', content: '## 我好\n### niha\n**默认有：**\n通用办公  研发工具  系统运维   质量测试   需求设计    信息检索    项目管理    数据分析    安全合规' },
        ]
      },
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: SKILL_CONTENT_GITHUB },
    ],
  },
  {
    id: 'pub-3',
    slug: 'web-search-pro',
    name: 'web-search-pro',
    nameZh: '网络搜索增强',
    description: 'Enhanced web search with multi-source aggregation, result ranking, and content extraction.',
    descriptionZh: '增强型网络搜索，支持多源聚合、结果排序和内容提取。',
    downloads: 8500,
    stars: 2900,
    version: '3.2.1',
    category: 'general-office',
    tags: ['search', 'web', 'information-retrieval'],
    versions: [
      { version: '3.2.1', date: '2026-03-18 16:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# Web Search Pro\n\n增强型网络搜索技能，支持多源聚合。' },
    ],
  },
  {
    id: 'pub-4',
    slug: 'code-reviewer',
    name: 'code-reviewer',
    nameZh: '代码审查助手',
    description: 'Automated code review with best practices, security checks, and performance suggestions.',
    descriptionZh: '自动化代码审查，包含最佳实践、安全检查和性能建议。',
    downloads: 7600,
    stars: 2700,
    version: '1.4.0',
    category: 'dev-tools',
    tags: ['code-review', 'security', 'quality'],
    versions: [
      { version: '1.4.0', date: '2026-03-15 12:00:00', isLatest: true },
      { version: '1.3.0', date: '2026-02-20 09:00:00', isLatest: false },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# Code Reviewer\n\n自动化代码审查技能。' },
    ],
  },
  {
    id: 'pub-5',
    slug: 'data-analyst',
    name: 'data-analyst',
    nameZh: '数据分析专家',
    description: 'Comprehensive data analysis with visualization, statistical insights, and report generation.',
    descriptionZh: '全面的数据分析技能，支持可视化、统计洞察和报告生成。',
    downloads: 6900,
    stars: 2400,
    version: '2.0.0',
    category: 'data-analysis',
    tags: ['data', 'analysis', 'visualization'],
    versions: [
      { version: '2.0.0', date: '2026-03-10 11:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# Data Analyst\n\n数据分析专家技能。' },
    ],
  },
  {
    id: 'pub-6',
    slug: 'sql-expert',
    name: 'sql-expert',
    nameZh: 'SQL 专家',
    description: 'Advanced SQL query optimization, schema design, and database performance tuning.',
    descriptionZh: '高级 SQL 查询优化、模式设计和数据库性能调优。',
    downloads: 6200,
    stars: 2200,
    version: '1.8.0',
    category: 'data-analysis',
    tags: ['sql', 'database', 'optimization'],
    versions: [
      { version: '1.8.0', date: '2026-03-08 10:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# SQL Expert\n\nSQL 专家技能，支持查询优化和性能调优。' },
    ],
  },
  {
    id: 'pub-7',
    slug: 'docker-ops',
    name: 'docker-ops',
    nameZh: 'Docker 运维',
    description: 'Docker container management, image optimization, and deployment automation.',
    descriptionZh: 'Docker 容器管理、镜像优化和部署自动化。',
    downloads: 5800,
    stars: 2000,
    version: '1.2.0',
    category: 'ops',
    tags: ['docker', 'container', 'devops'],
    versions: [
      { version: '1.2.0', date: '2026-03-05 09:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# Docker Ops\n\nDocker 运维技能。' },
    ],
  },
  {
    id: 'pub-8',
    slug: 'email-writer',
    name: 'email-writer',
    nameZh: '邮件撰写助手',
    description: 'Professional email drafting with tone adjustment, template management, and multilingual support.',
    descriptionZh: '专业邮件撰写，支持语气调整、模板管理和多语言。',
    downloads: 5400,
    stars: 1900,
    version: '2.3.0',
    category: 'general-office',
    tags: ['email', 'writing', 'communication'],
    versions: [
      { version: '2.3.0', date: '2026-03-01 08:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# Email Writer\n\n邮件撰写助手技能。' },
    ],
  },
  {
    id: 'pub-9',
    slug: 'k8s-manager',
    name: 'k8s-manager',
    nameZh: 'Kubernetes 管理',
    description: 'Kubernetes cluster management, pod debugging, and resource optimization.',
    descriptionZh: 'Kubernetes 集群管理、Pod 调试和资源优化。',
    downloads: 4900,
    stars: 1700,
    version: '1.6.0',
    category: 'ops',
    tags: ['kubernetes', 'k8s', 'cloud'],
    versions: [
      { version: '1.6.0', date: '2026-02-25 15:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# K8s Manager\n\nKubernetes 管理技能。' },
    ],
  },
  {
    id: 'pub-10',
    slug: 'ppt-generator',
    name: 'ppt-generator',
    nameZh: 'PPT 生成助手',
    description: 'Automated PowerPoint generation from outlines with theme customization and chart integration.',
    descriptionZh: '从大纲自动生成 PPT，支持主题定制和图表集成。',
    downloads: 4500,
    stars: 1600,
    version: '1.1.0',
    category: 'general-office',
    tags: ['ppt', 'presentation', 'automation'],
    versions: [
      { version: '1.1.0', date: '2026-02-20 14:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# PPT Generator\n\nPPT 生成助手技能。' },
    ],
  },
  {
    id: 'pub-11',
    slug: 'security-scanner',
    name: 'security-scanner',
    nameZh: '安全扫描工具',
    description: 'Automated security vulnerability scanning for code, dependencies, and infrastructure.',
    descriptionZh: '自动化安全漏洞扫描，覆盖代码、依赖和基础设施。',
    downloads: 4200,
    stars: 1500,
    version: '2.0.1',
    category: 'security',
    tags: ['security', 'vulnerability', 'scanning'],
    versions: [
      { version: '2.0.1', date: '2026-02-18 11:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# Security Scanner\n\n安全扫描工具技能。' },
    ],
  },
  {
    id: 'pub-12',
    slug: 'api-tester',
    name: 'api-tester',
    nameZh: 'API 测试助手',
    description: 'Comprehensive API testing with request generation, response validation, and load testing.',
    descriptionZh: '全面的 API 测试，支持请求生成、响应验证和负载测试。',
    downloads: 3900,
    stars: 1400,
    version: '1.5.0',
    category: 'dev-tools',
    tags: ['api', 'testing', 'http'],
    versions: [
      { version: '1.5.0', date: '2026-02-15 10:00:00', isLatest: true },
    ],
    files: [
      { name: 'SKILL.md', path: 'SKILL.md', type: 'file', content: '# API Tester\n\nAPI 测试助手技能。' },
    ],
  },
  // ─── 开发工具分类补充数据（用于演示分页）─────────────────────────────────────
  {
    id: 'pub-dt-01', slug: 'git-helper', name: 'git-helper', nameZh: 'Git 操作助手',
    description: 'Streamline Git workflows with smart commit messages, branch management, and conflict resolution.',
    descriptionZh: '智能化 Git 工作流，支持提交信息生成、分支管理和冲突解决。',
    downloads: 3700, stars: 1300, version: '1.3.0', category: 'dev-tools', tags: ['git', 'vcs'],
    versions: [{ version: '1.3.0', date: '2026-02-10 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Git Helper' }],
  },
  {
    id: 'pub-dt-02', slug: 'npm-manager', name: 'npm-manager', nameZh: 'NPM 包管理',
    description: 'Manage Node.js dependencies, audit vulnerabilities, and optimize package.json.',
    descriptionZh: '管理 Node.js 依赖包，审计漏洞，优化 package.json 配置。',
    downloads: 3500, stars: 1250, version: '2.0.0', category: 'dev-tools', tags: ['npm', 'nodejs'],
    versions: [{ version: '2.0.0', date: '2026-02-08 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# NPM Manager' }],
  },
  {
    id: 'pub-dt-03', slug: 'regex-builder', name: 'regex-builder', nameZh: '正则表达式构建器',
    description: 'Build, test, and explain complex regular expressions with natural language.',
    descriptionZh: '用自然语言构建、测试和解释复杂正则表达式。',
    downloads: 3300, stars: 1200, version: '1.1.0', category: 'dev-tools', tags: ['regex', 'pattern'],
    versions: [{ version: '1.1.0', date: '2026-02-05 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Regex Builder' }],
  },
  {
    id: 'pub-dt-04', slug: 'ci-cd-helper', name: 'ci-cd-helper', nameZh: 'CI/CD 流水线助手',
    description: 'Design and troubleshoot CI/CD pipelines for GitHub Actions, GitLab CI, and Jenkins.',
    descriptionZh: '设计和排查 GitHub Actions、GitLab CI 和 Jenkins 流水线问题。',
    downloads: 3100, stars: 1150, version: '1.4.0', category: 'dev-tools', tags: ['ci', 'cd', 'devops'],
    versions: [{ version: '1.4.0', date: '2026-02-03 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# CI/CD Helper' }],
  },
  {
    id: 'pub-dt-05', slug: 'db-schema-designer', name: 'db-schema-designer', nameZh: '数据库模式设计',
    description: 'Design normalized database schemas, generate migrations, and visualize ER diagrams.',
    descriptionZh: '设计规范化数据库模式，生成迁移脚本，可视化 ER 图。',
    downloads: 2900, stars: 1100, version: '1.2.0', category: 'dev-tools', tags: ['database', 'schema'],
    versions: [{ version: '1.2.0', date: '2026-01-30 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# DB Schema Designer' }],
  },
  {
    id: 'pub-dt-06', slug: 'openapi-gen', name: 'openapi-gen', nameZh: 'OpenAPI 文档生成',
    description: 'Generate OpenAPI 3.0 specifications from code, comments, or natural language descriptions.',
    descriptionZh: '从代码、注释或自然语言描述生成 OpenAPI 3.0 规范文档。',
    downloads: 2750, stars: 1050, version: '2.1.0', category: 'dev-tools', tags: ['openapi', 'swagger'],
    versions: [{ version: '2.1.0', date: '2026-01-28 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# OpenAPI Gen' }],
  },
  {
    id: 'pub-dt-07', slug: 'test-writer', name: 'test-writer', nameZh: '单元测试生成',
    description: 'Automatically generate unit tests for JavaScript, Python, Go, and Java code.',
    descriptionZh: '自动为 JavaScript、Python、Go 和 Java 代码生成单元测试。',
    downloads: 2600, stars: 980, version: '1.7.0', category: 'dev-tools', tags: ['testing', 'unit-test'],
    versions: [{ version: '1.7.0', date: '2026-01-25 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Test Writer' }],
  },
  {
    id: 'pub-dt-08', slug: 'log-analyzer', name: 'log-analyzer', nameZh: '日志分析助手',
    description: 'Parse and analyze application logs to identify errors, performance bottlenecks, and anomalies.',
    descriptionZh: '解析和分析应用日志，识别错误、性能瓶颈和异常。',
    downloads: 2450, stars: 920, version: '1.0.0', category: 'dev-tools', tags: ['logs', 'monitoring'],
    versions: [{ version: '1.0.0', date: '2026-01-22 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Log Analyzer' }],
  },
  {
    id: 'pub-dt-09', slug: 'env-config', name: 'env-config', nameZh: '环境配置管理',
    description: 'Manage environment variables, .env files, and configuration across dev/staging/prod.',
    descriptionZh: '管理环境变量、.env 文件和跨开发/预发/生产环境的配置。',
    downloads: 2300, stars: 870, version: '1.5.0', category: 'dev-tools', tags: ['env', 'config'],
    versions: [{ version: '1.5.0', date: '2026-01-20 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Env Config' }],
  },
  {
    id: 'pub-dt-10', slug: 'code-formatter', name: 'code-formatter', nameZh: '代码格式化工具',
    description: 'Format and lint code across multiple languages with configurable style rules.',
    descriptionZh: '跨多种语言格式化和检查代码，支持可配置的风格规则。',
    downloads: 2200, stars: 830, version: '3.0.0', category: 'dev-tools', tags: ['formatter', 'linter'],
    versions: [{ version: '3.0.0', date: '2026-01-18 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Code Formatter' }],
  },
  {
    id: 'pub-dt-11', slug: 'dependency-updater', name: 'dependency-updater', nameZh: '依赖版本升级',
    description: 'Safely upgrade project dependencies with compatibility checks and changelog summaries.',
    descriptionZh: '安全升级项目依赖，包含兼容性检查和变更日志摘要。',
    downloads: 2100, stars: 790, version: '1.3.0', category: 'dev-tools', tags: ['dependencies', 'upgrade'],
    versions: [{ version: '1.3.0', date: '2026-01-15 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Dependency Updater' }],
  },
  {
    id: 'pub-dt-12', slug: 'mock-data-gen', name: 'mock-data-gen', nameZh: 'Mock 数据生成',
    description: 'Generate realistic mock data for APIs, databases, and UI prototyping.',
    descriptionZh: '为 API、数据库和 UI 原型生成真实感强的 Mock 数据。',
    downloads: 2000, stars: 750, version: '2.2.0', category: 'dev-tools', tags: ['mock', 'faker'],
    versions: [{ version: '2.2.0', date: '2026-01-12 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Mock Data Gen' }],
  },
  {
    id: 'pub-dt-13', slug: 'perf-profiler', name: 'perf-profiler', nameZh: '性能分析助手',
    description: 'Profile and optimize application performance with flame graphs and bottleneck detection.',
    descriptionZh: '通过火焰图和瓶颈检测分析并优化应用性能。',
    downloads: 1900, stars: 710, version: '1.1.0', category: 'dev-tools', tags: ['performance', 'profiling'],
    versions: [{ version: '1.1.0', date: '2026-01-10 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Perf Profiler' }],
  },
  {
    id: 'pub-dt-14', slug: 'graphql-helper', name: 'graphql-helper', nameZh: 'GraphQL 开发助手',
    description: 'Design GraphQL schemas, write queries/mutations, and debug resolver issues.',
    descriptionZh: '设计 GraphQL 模式，编写查询/变更，调试 resolver 问题。',
    downloads: 1800, stars: 670, version: '1.6.0', category: 'dev-tools', tags: ['graphql', 'api'],
    versions: [{ version: '1.6.0', date: '2026-01-08 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# GraphQL Helper' }],
  },
  {
    id: 'pub-dt-15', slug: 'terraform-helper', name: 'terraform-helper', nameZh: 'Terraform 基础设施',
    description: 'Write and review Terraform configurations for AWS, GCP, and Azure infrastructure.',
    descriptionZh: '编写和审查 AWS、GCP 和 Azure 基础设施的 Terraform 配置。',
    downloads: 1700, stars: 630, version: '2.0.0', category: 'dev-tools', tags: ['terraform', 'iac'],
    versions: [{ version: '2.0.0', date: '2026-01-05 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Terraform Helper' }],
  },
  {
    id: 'pub-dt-16', slug: 'proto-gen', name: 'proto-gen', nameZh: 'Protobuf 代码生成',
    description: 'Generate and maintain Protocol Buffer definitions and gRPC service stubs.',
    descriptionZh: '生成和维护 Protocol Buffer 定义及 gRPC 服务桩代码。',
    downloads: 1600, stars: 590, version: '1.2.0', category: 'dev-tools', tags: ['protobuf', 'grpc'],
    versions: [{ version: '1.2.0', date: '2026-01-03 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Proto Gen' }],
  },
  {
    id: 'pub-dt-17', slug: 'webhook-tester', name: 'webhook-tester', nameZh: 'Webhook 测试工具',
    description: 'Test, debug, and replay webhook events with request inspection and response simulation.',
    descriptionZh: '测试、调试和重放 Webhook 事件，支持请求检查和响应模拟。',
    downloads: 1500, stars: 550, version: '1.0.0', category: 'dev-tools', tags: ['webhook', 'http'],
    versions: [{ version: '1.0.0', date: '2025-12-28 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Webhook Tester' }],
  },
  {
    id: 'pub-dt-18', slug: 'cron-builder', name: 'cron-builder', nameZh: 'Cron 表达式构建',
    description: 'Build, validate, and explain cron expressions with natural language input.',
    descriptionZh: '用自然语言构建、验证和解释 Cron 表达式。',
    downloads: 1400, stars: 510, version: '1.1.0', category: 'dev-tools', tags: ['cron', 'scheduler'],
    versions: [{ version: '1.1.0', date: '2025-12-25 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Cron Builder' }],
  },
  {
    id: 'pub-dt-19', slug: 'jwt-debugger', name: 'jwt-debugger', nameZh: 'JWT 调试工具',
    description: 'Decode, verify, and generate JSON Web Tokens with claim inspection.',
    descriptionZh: '解码、验证和生成 JSON Web Token，支持 claim 检查。',
    downloads: 1300, stars: 470, version: '1.0.0', category: 'dev-tools', tags: ['jwt', 'auth'],
    versions: [{ version: '1.0.0', date: '2025-12-22 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# JWT Debugger' }],
  },
  {
    id: 'pub-dt-20', slug: 'markdown-renderer', name: 'markdown-renderer', nameZh: 'Markdown 渲染预览',
    description: 'Render and preview Markdown documents with extended syntax support.',
    descriptionZh: '渲染和预览 Markdown 文档，支持扩展语法。',
    downloads: 1200, stars: 430, version: '2.0.0', category: 'dev-tools', tags: ['markdown', 'preview'],
    versions: [{ version: '2.0.0', date: '2025-12-20 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Markdown Renderer' }],
  },
  {
    id: 'pub-dt-21', slug: 'color-palette', name: 'color-palette', nameZh: '色彩调色板生成',
    description: 'Generate accessible color palettes for UI design with contrast ratio checks.',
    descriptionZh: '为 UI 设计生成无障碍色彩调色板，包含对比度检查。',
    downloads: 1100, stars: 390, version: '1.3.0', category: 'dev-tools', tags: ['color', 'design'],
    versions: [{ version: '1.3.0', date: '2025-12-18 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Color Palette' }],
  },
  {
    id: 'pub-dt-22', slug: 'i18n-helper', name: 'i18n-helper', nameZh: '国际化翻译助手',
    description: 'Manage i18n translation files, detect missing keys, and auto-translate strings.',
    descriptionZh: '管理 i18n 翻译文件，检测缺失键值，自动翻译字符串。',
    downloads: 1000, stars: 350, version: '1.4.0', category: 'dev-tools', tags: ['i18n', 'localization'],
    versions: [{ version: '1.4.0', date: '2025-12-15 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# i18n Helper' }],
  },
  {
    id: 'pub-dt-23', slug: 'semver-manager', name: 'semver-manager', nameZh: '语义化版本管理',
    description: 'Manage semantic versioning, generate changelogs, and automate release notes.',
    descriptionZh: '管理语义化版本，生成变更日志，自动化发布说明。',
    downloads: 900, stars: 310, version: '1.0.0', category: 'dev-tools', tags: ['semver', 'release'],
    versions: [{ version: '1.0.0', date: '2025-12-12 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Semver Manager' }],
  },
  {
    id: 'pub-dt-24', slug: 'ssh-helper', name: 'ssh-helper', nameZh: 'SSH 连接助手',
    description: 'Manage SSH connections, keys, and tunnels with guided configuration.',
    descriptionZh: '管理 SSH 连接、密钥和隧道，提供引导式配置。',
    downloads: 800, stars: 270, version: '1.1.0', category: 'dev-tools', tags: ['ssh', 'remote'],
    versions: [{ version: '1.1.0', date: '2025-12-10 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# SSH Helper' }],
  },
  {
    id: 'pub-dt-25', slug: 'code-explainer', name: 'code-explainer', nameZh: '代码解释器',
    description: 'Explain complex code snippets in plain language with step-by-step breakdowns.',
    descriptionZh: '用通俗语言逐步解释复杂代码片段。',
    downloads: 700, stars: 230, version: '2.0.0', category: 'dev-tools', tags: ['explain', 'learning'],
    versions: [{ version: '2.0.0', date: '2025-12-08 10:00:00', isLatest: true }],
    files: [{ name: 'SKILL.md', path: 'SKILL.md', type: 'file' as const, content: '# Code Explainer' }],
  },
];

// ─── 技能初始包 Mock 数据 ─────────────────────────────────────────────────────

export const INITIAL_SKILL_PACKAGES_DEFAULT: SkillInitialPackage[] = [
  {
    id: 'pkg-1',
    name: '全员通用技能包',
    scope: '全部用户',
    scopeType: 'public',
    groupIds: [],
    isActive: true,
    hasDraft: false,
    createdAt: new Date('2026-01-01'),
    updatedAt: new Date('2026-03-01'),
    skills: [
      {
        skillId: 'pub-3',
        skillName: 'web-search-pro',
        skillNameZh: '网络搜索增强',
        source: 'public',
        version: '3.1.0',
        addedAt: new Date('2026-01-15'),
      },
      {
        skillId: 'pub-8',
        skillName: 'email-writer',
        skillNameZh: '邮件撰写助手',
        source: 'public',
        version: '2.1.0',
        addedAt: new Date('2026-01-20'),
      },
      {
        skillId: 'pub-10',
        skillName: 'ppt-generator',
        skillNameZh: 'PPT 生成助手',
        source: 'public',
        version: '1.1.0',
        addedAt: new Date('2026-02-01'),
      },
    ],
  },
  {
    id: 'pkg-2',
    name: '高级开发技能包',
    scope: '全部用户',
    scopeType: 'public',
    groupIds: [],
    isActive: false,
    hasDraft: true,
    createdAt: new Date('2026-02-01'),
    updatedAt: new Date('2026-03-20'),
    skills: [
      {
        skillId: 'pub-1',
        skillName: 'self-improving-agent',
        skillNameZh: '自我改进代理',
        source: 'public',
        version: '1.0.0',
        addedAt: new Date('2026-02-10'),
      },
      {
        skillId: 'pub-2',
        skillName: 'github',
        skillNameZh: 'GitHub 工具',
        source: 'public',
        version: '1.5.0',
        addedAt: new Date('2026-02-10'),
      },
      {
        skillId: 'pub-4',
        skillName: 'code-reviewer',
        skillNameZh: '代码审查助手',
        source: 'public',
        version: '1.3.0',
        addedAt: new Date('2026-02-15'),
      },
      {
        skillId: 'pub-7',
        skillName: 'docker-ops',
        skillNameZh: 'Docker 运维',
        source: 'public',
        version: '1.2.0',
        addedAt: new Date('2026-03-01'),
      },
      {
        skillId: 'skill-2',
        skillName: 'code-reviewer',
        skillNameZh: '代码审查工具',
        source: 'enterprise',
        version: '2.0.0',
        addedAt: new Date('2026-03-05'),
      },
    ],
  },
  {
    id: 'pkg-3',
    name: '运维团队技能包',
    scope: '运维组',
    scopeType: 'private',
    groupIds: ['grp-2'],
    isActive: true,
    hasDraft: false,
    createdAt: new Date('2026-02-15'),
    updatedAt: new Date('2026-03-15'),
    skills: [
      {
        skillId: 'pub-7',
        skillName: 'docker-ops',
        skillNameZh: 'Docker 运维',
        source: 'public',
        version: '1.0.0',
        addedAt: new Date('2026-02-20'),
      },
      {
        skillId: 'pub-9',
        skillName: 'k8s-manager',
        skillNameZh: 'Kubernetes 管理',
        source: 'public',
        version: '1.4.0',
        addedAt: new Date('2026-02-20'),
      },
    ],
  },
];
