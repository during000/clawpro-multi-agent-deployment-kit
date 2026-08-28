import { Skill, Category, OpenClawInstance, type SkillVersionRecord, Group, type SkillSecurityInfo } from './types';
import { MOCK_GROUPS_INIT } from '../MemberManagement';
import { MOCK_PROJECTS } from '../MemberManagement/mock';

// ========== 组织数据：直接使用【用户管理】-【组织】的数据 ==========
export const MOCK_GROUPS: Group[] = MOCK_GROUPS_INIT.map(g => ({
  id: g.id,
  name: g.name,
}));

// ========== 项目数据：来自【用户管理】-【项目】(source==='project') ==========
// 供工具库「应用范围」的「按组织」面板以「项目」分组小标题展示，可与组织同时勾选。
// 保留 parentId 以还原项目 / 子项目树形结构。
export const MOCK_PROJECT_GROUPS: Array<{ id: string; name: string; parentId?: string | null }> =
  MOCK_PROJECTS.map(p => ({
    id: p.id,
    name: p.name,
    parentId: p.parentId ?? null,
  }));

// 为 GitHub Skill 创建额外的文件内容
const githubSkillFiles: Record<string, string> = {
  'SKILL.md': `---
name: github
description: "Interact with GitHub using the \`gh\` CLI. Use \`gh issue\`, \`gh pr\`, \`gh run\`, and \`gh api\` for issues, PRs, CI runs, and advanced queries."
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

View logs for failed steps only:
\`\`\`bash
gh run view <run-id> --repo owner/repo --log-failed
\`\`\`

## API for Advanced Queries

The \`gh api\` command is useful for accessing data not available through other subcommands.`,
  'hha/ha.md': `## 我好
### niha
**默认有：**
通用办公  研发工具  系统运维   质量测试   需求设计    信息检索    项目管理    数据分析    安全合规
支持新增和删除。

|**序号**|**分类名称**|**描述（核心定位、覆盖范围）**|
|:-:|:-:|:-:|
|**1**|**通用办公**|文档总结、邮件润色、PPT 大纲、翻译助手|
|**2**|**研发工具**|代码 Review、接口调试、技术文档解析、架构建议|`,
};

export const DEFAULT_CATEGORIES: Category[] = [
  { id: '1', name: '通用办公', description: '文档总结、邮件润色、PPT 大纲、翻译助手' },
  { id: '2', name: '研发工具', description: '代码 Review、接口调试、技术文档解析、架构建议' },
  { id: '3', name: '系统运维', description: '资源巡检、环境部署、日志分析、告警诊断' },
  { id: '4', name: '质量测试', description: '用例生成、自动化脚本编写、Bug 辅助定位' },
  { id: '5', name: '需求设计', description: '需求评审、PRD 辅助写作、UI/UX 设计灵感' },
  { id: '6', name: '信息检索', description: '企业知识库查询、竞品实时监控、技术趋势搜索' },
  { id: '7', name: '项目管理', description: '进度汇总、周报自动生成、风险预警、任务拆解' },
  { id: '8', name: '数据分析', description: 'SQL 自动编写、报表解释、数据清洗逻辑' },
  { id: '9', name: '安全合规', description: '权限审计、代码漏洞扫描、合规性自查' },
  { id: '10', name: '其他', description: '其他分类' },
];

// ========== 安全检测 Mock 数据 ==========

const SECURITY_DIMENSIONS_SAFE: SkillSecurityInfo['engines'][0]['dimensions'] = [
  { name: '供应链风险', status: 'safe', detail: '未发现可疑的第三方依赖引入或供应链污染行为' },
  { name: '命令执行风险', status: 'safe', detail: '未检测到危险的系统命令调用或子进程执行操作' },
  { name: '网络请求与数据外传', status: 'safe', detail: '未发现未经授权的网络请求或敏感数据外传行为' },
  { name: '文件操作与敏感路径访问', status: 'safe', detail: '未检测到对敏感系统路径或凭证文件的异常访问' },
  { name: 'Prompt 注入风险', status: 'safe', detail: '未发现试图篡改 AI Agent 行为的 Prompt 注入指令' },
  { name: '远程脚本下载执行', status: 'safe', detail: '未检测到从远程服务器下载并执行脚本的行为' },
  { name: '可疑编码/混淆', status: 'safe', detail: '未发现可疑的代码编码混淆或加密逃逸技术' },
  { name: '其他安全风险', status: 'safe', detail: '未检测到其他类别的异常安全风险行为' },
];

const SECURITY_DIMENSIONS_SUSPICIOUS: SkillSecurityInfo['engines'][0]['dimensions'] = [
  { name: '供应链风险', status: 'safe', detail: '未发现可疑的第三方依赖引入或供应链污染行为' },
  { name: '命令执行风险', status: 'suspicious', detail: '检测到潜在的系统命令调用，存在一定风险' },
  { name: '网络请求与数据外传', status: 'safe', detail: '未发现未经授权的网络请求或敏感数据外传行为' },
  { name: '文件操作与敏感路径访问', status: 'safe', detail: '未检测到对敏感系统路径或凭证文件的异常访问' },
  { name: 'Prompt 注入风险', status: 'safe', detail: '未发现试图篡改 AI Agent 行为的 Prompt 注入指令' },
  { name: '远程脚本下载执行', status: 'safe', detail: '未检测到从远程服务器下载并执行脚本的行为' },
  { name: '可疑编码/混淆', status: 'suspicious', detail: '发现部分代码使用了 Base64 编码包裹，需人工确认' },
  { name: '其他安全风险', status: 'safe', detail: '未检测到其他类别的异常安全风险行为' },
];

/** 安全 */
const MOCK_SECURITY_SAFE: SkillSecurityInfo = {
  overallStatus: 'safe',
  contentHash: '1fabf1a131f59232ee64a06c4b7042ce',
  engines: [
    { engineName: '腾讯云 AI Agent 安全', status: 'safe', reportUrl: 'https://tix.qq.com/search/skill?keyword=1fabf1a131f59232ee64a06c4b7042ce', score: 95, dimensions: SECURITY_DIMENSIONS_SAFE },
  ],
};

/** 可疑 */
const MOCK_SECURITY_SUSPICIOUS: SkillSecurityInfo = {
  overallStatus: 'suspicious',
  contentHash: 'a8b2c3d4e5f6789012345abcdef67890',
  engines: [
    { engineName: '腾讯云 AI Agent 安全', status: 'suspicious', reportUrl: 'https://tix.qq.com/search/skill?keyword=a8b2c3d4e5f6789012345abcdef67890', score: 55, dimensions: SECURITY_DIMENSIONS_SUSPICIOUS },
  ],
};

/** 检测中 */
const MOCK_SECURITY_SCANNING: SkillSecurityInfo = {
  overallStatus: 'scanning',
  engines: [],
};

/** 恶意 */
const MOCK_SECURITY_MALICIOUS: SkillSecurityInfo = {
  overallStatus: 'malicious',
  contentHash: 'deadbeef1234567890abcdef12345678',
  engines: [
    { engineName: '腾讯云 AI Agent 安全', status: 'malicious', reportUrl: 'https://tix.qq.com/search/skill?keyword=deadbeef1234567890abcdef12345678', score: 15, dimensions: [
      { name: '供应链风险', status: 'malicious', detail: '发现恶意第三方依赖注入，存在供应链污染' },
      { name: '命令执行风险', status: 'malicious', detail: '检测到危险的系统命令调用，执行 rm -rf 和反弹 shell' },
      { name: '网络请求与数据外传', status: 'malicious', detail: '发现向外部 C2 服务器发送敏感数据' },
      { name: '文件操作与敏感路径访问', status: 'safe', detail: '未检测到对敏感系统路径或凭证文件的异常访问' },
      { name: 'Prompt 注入风险', status: 'suspicious', detail: '发现可能篡改 AI Agent 行为的指令片段' },
      { name: '远程脚本下载执行', status: 'malicious', detail: '检测到从远程服务器下载并执行恶意脚本' },
      { name: '可疑编码/混淆', status: 'malicious', detail: '发现大量代码使用多层编码混淆，隐藏恶意逻辑' },
      { name: '其他安全风险', status: 'safe', detail: '未检测到其他类别的异常安全风险行为' },
    ]},
  ],
};

/** 未检测 */
const MOCK_SECURITY_NOT_SCANNED: SkillSecurityInfo = {
  overallStatus: 'not_scanned',
  engines: [],
};

// skill-0: 安全 ✅
// skill-1: 可疑 ⚠️
// skill-2: 恶意 ❌
// skill-3: 安全检测中 🔄（10s后随机完成）
// skill-4: 未检测
// skill-5: 安全 ✅
// skill-6: 未检测
// skill-7: 可疑 ⚠️
export const MOCK_SKILLS: Skill[] = [
  {
    id: 'skill-0',
    slug: 'knowledge-qa',
    name: '知识库问答',
    description: '基于企业知识库进行智能问答，快速检索内部文档、规范和流程信息。支持多轮对话、上下文理解，是全员通用的知识检索工具。',
    version: '2.0.0',
    categories: ['6', '1'],
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-03-22'),
    securityInfo: MOCK_SECURITY_SAFE,
    content: '# 知识库问答\n\n基于企业知识库进行智能问答的 Skill...',
    versions: ['2.0.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# 知识库问答\n\n基于企业知识库进行智能问答的 Skill...' },
      { name: 'README.md', size: 512, content: '# Knowledge QA\n\n## Features\n\n- 企业知识库智能检索\n- 多轮对话上下文理解\n- 支持文档、规范、流程' },
    ],
    versionHistory: [
      { version: '2.0.0', date: '2025-03-22', changeLog: '1、新增多轮对话支持\n2、优化检索精度', files: [
        { name: 'SKILL.md', size: 1024, content: '# 知识库问答\n\n基于企业知识库进行智能问答的 Skill...' },
      ]},
      { version: '1.0.0', date: '2025-02-10', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 900, content: '# 知识库问答 v1.0\n\n基础知识库问答功能。' },
      ]},
    ],
  },
  {
    id: 'skill-1',
    slug: 'doc-summarizer',
    name: '文档总结助手',
    description: '快速总结长文档，提取关键信息。支持多种文档格式，自动提取核心要点，生成简明扩订。适用于会议记录、研究报告、技术文档等场景。',
    version: '1.0.0',
    categories: ['1', '6'],
    scope: 'private',
    groupIds: ['grp-4'],
    uploadTime: new Date('2025-03-20'),
    securityInfo: MOCK_SECURITY_SUSPICIOUS,
    content: '# 文档总结助手\n\n这是一个用于快速总结长文档的 Skill...',
    versions: ['1.0.0', '0.9.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# 文档总结助手\n\n这是一个用于快速总结长文档的 Skill...' },
      { name: 'README.md', size: 512, content: '# README\n\n## 安装\n\n```bash\nnpm install doc-summarizer\n```\n\n## 使用说明\n\n支持多种文档格式，自动提取核心要点。' },
      { name: 'docs/guide.md', size: 768, content: '# 使用指南\n\n## 快速开始\n\n1. 上传文档\n2. 选择总结模式\n3. 获取结果\n\n## 高级配置\n\n支持自定义总结长度和风格。' },
    ],
    versionHistory: [
      { version: '1.0.0', date: '2025-03-20', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 1024, content: '# 文档总结助手\n\n这是一个用于快速总结长文档的 Skill...' },
        { name: 'README.md', size: 512, content: '# README\n\n## 安装\n\n```bash\nnpm install doc-summarizer\n```\n\n## 使用说明\n\n支持多种文档格式，自动提取核心要点。' },
        { name: 'docs/guide.md', size: 768, content: '# 使用指南\n\n## 快速开始\n\n1. 上传文档\n2. 选择总结模式\n3. 获取结果\n\n## 高级配置\n\n支持自定义总结长度和风格。' },
      ]},
      { version: '0.9.0', date: '2025-03-06', changeLog: '内测版本', files: [
        { name: 'SKILL.md', size: 980, content: '# 文档总结助手（内测版）\n\n这是一个用于快速总结长文档的 Skill 内测版本...' },
        { name: 'README.md', size: 400, content: '# README\n\n## 内测说明\n\n本版本为内测版，功能尚不完善。' },
      ]},
    ],
  },
  {
    id: 'skill-2',
    slug: 'code-reviewer',
    name: '代码审查工具',
    description: '自动审查代码质量和安全问题。支持 Python、JavaScript、Java 等主流语言，检测代码规范、安全漏洞、性能问题。提供详细的修改建议和最佳实践。',
    version: '2.1.0',
    categories: ['2'],
    scope: 'private',
    groupIds: ['grp-2'],
    uploadTime: new Date('2025-03-18'),
    securityInfo: MOCK_SECURITY_MALICIOUS,
    content: '# 代码审查工具\n\n这是一个用于代码审查的 Skill...',
    versions: ['2.1.0', '2.0.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# 代码审查工具\n\n这是一个用于代码审查的 Skill...' },
      { name: 'README.md', size: 512, content: '# Code Reviewer\n\n## Features\n\n- 支持 Python、JavaScript、Java 等主流语言\n- 检测代码规范和安全漏洞\n- 提供详细修改建议' },
      { name: 'config/rules.md', size: 768, content: '# 审查规则配置\n\n## 默认规则\n\n| 规则 | 说明 | 严重程度 |\n|------|------|----------|\n| no-eval | 禁止使用 eval | error |\n| no-console | 禁止 console.log | warning |' },
    ],
    versionHistory: [
      { version: '2.1.0', date: '2025-03-18', changeLog: '1、修改描述字段 from 自动审查代码 to 自动审查代码质量和安全问题\n2、更新SKILL文件', files: [
        { name: 'SKILL.md', size: 1024, content: '# 代码审查工具\n\n这是一个用于代码审查的 Skill...' },
        { name: 'README.md', size: 512, content: '# Code Reviewer\n\n## Features\n\n- 支持 Python、JavaScript、Java 等主流语言\n- 检测代码规范和安全漏洞\n- 提供详细修改建议' },
        { name: 'config/rules.md', size: 768, content: '# 审查规则配置\n\n## 默认规则\n\n| 规则 | 说明 | 严重程度 |\n|------|------|----------|\n| no-eval | 禁止使用 eval | error |\n| no-console | 禁止 console.log | warning |' },
      ]},
      { version: '2.0.0', date: '2025-03-04', changeLog: '1、新增 Java 语言支持\n2、更新SKILL文件', files: [
        { name: 'SKILL.md', size: 980, content: '# 代码审查工具 v2.0\n\n支持 Python、JavaScript、Java 三种语言的代码审查。' },
        { name: 'README.md', size: 480, content: '# Code Reviewer v2.0\n\n## What\'s New\n\n- 新增 Java 语言支持\n- 优化检测引擎' },
      ]},
      { version: '1.0.0', date: '2025-02-18', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 900, content: '# 代码审查工具 v1.0\n\n支持 Python、JavaScript 的代码审查。' },
        { name: 'README.md', size: 400, content: '# Code Reviewer v1.0\n\n## Features\n\n- 支持 Python、JavaScript\n- 基础代码规范检测' },
      ]},
    ],
  },
  {
    id: 'skill-3',
    slug: 'log-analyzer',
    name: '日志分析器',
    description: '分析系统日志，快速定位问题。支持应用日志、系统日志、数据库日志等多种日志类型。自动提取错误信息、分析异常模式、帮助定位根本原因。',
    version: '1.5.2',
    categories: ['3'],
    scope: 'private',
    groupIds: ['grp-1', 'grp-2'],
    uploadTime: new Date('2025-03-15'),
    securityInfo: MOCK_SECURITY_SCANNING,
    content: '# 日志分析器\n\n这是一个用于日志分析的 Skill...',
    versions: ['1.5.2', '1.5.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# 日志分析器\n\n这是一个用于日志分析的 Skill...' },
      { name: 'README.md', size: 512, content: '# Log Analyzer\n\n## 概述\n\n自动分析系统日志，快速定位问题根因。\n\n## 支持的日志类型\n\n- 应用日志\n- 系统日志\n- 数据库日志' },
      { name: 'examples/usage.md', size: 768, content: '# 使用示例\n\n## 基础用法\n\n```bash\nlog-analyzer --input /var/log/app.log\n```\n\n## 过滤特定错误\n\n```bash\nlog-analyzer --input /var/log/app.log --level error\n```' },
    ],
    versionHistory: [
      { version: '1.5.2', date: '2025-03-15', changeLog: '1、修复日志分析异常\n2、更新SKILL文件', files: [
        { name: 'SKILL.md', size: 1024, content: '# 日志分析器\n\n这是一个用于日志分析的 Skill...' },
        { name: 'README.md', size: 512, content: '# Log Analyzer\n\n## 概述\n\n自动分析系统日志，快速定位问题根因。\n\n## 支持的日志类型\n\n- 应用日志\n- 系统日志\n- 数据库日志' },
        { name: 'examples/usage.md', size: 768, content: '# 使用示例\n\n## 基础用法\n\n```bash\nlog-analyzer --input /var/log/app.log\n```\n\n## 过滤特定错误\n\n```bash\nlog-analyzer --input /var/log/app.log --level error\n```' },
      ]},
      { version: '1.5.0', date: '2025-03-01', changeLog: '1、新增数据库日志支持', files: [
        { name: 'SKILL.md', size: 980, content: '# 日志分析器 v1.5\n\n新增数据库日志支持。' },
        { name: 'README.md', size: 480, content: '# Log Analyzer v1.5\n\n## What\'s New\n\n- 新增数据库日志支持\n- 优化分析引擎' },
      ]},
      { version: '1.0.0', date: '2025-02-15', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 900, content: '# 日志分析器 v1.0\n\n基础日志分析功能。' },
        { name: 'README.md', size: 400, content: '# Log Analyzer v1.0\n\n## Features\n\n- 支持应用日志分析\n- 自动提取错误信息' },
      ]},
    ],
  },
  {
    id: 'skill-4',
    slug: 'github',
    name: 'GitHub',
    description: 'Interact with GitHub using the `gh` CLI. Use `gh issue`, `gh pr`, `gh run`, and `gh api` for issues, PRs, CI runs, and advanced queries.',
    version: '1.0.0',
    categories: ['2'],
    scope: 'private',
    groupIds: ['grp-4'],
    uploadTime: new Date('2025-03-20'),
    securityInfo: MOCK_SECURITY_NOT_SCANNED,
    content: `---
name: github
description: "Interact with GitHub using the \`gh\` CLI. Use \`gh issue\`, \`gh pr\`, \`gh run\`, and \`gh api\` for issues, PRs, CI runs, and advanced queries."
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

View logs for failed steps only:
\`\`\`bash
gh run view <run-id> --repo owner/repo --log-failed
\`\`\`

## API for Advanced Queries

The \`gh api\` command is useful for accessing data not available through other subcommands.`,
    versions: ['1.0.0', '0.9.0', '0.8.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: githubSkillFiles['SKILL.md'] },
      { name: 'hha/ha.md', size: 512, content: githubSkillFiles['hha/ha.md'] },
    ],
    versionHistory: [
      { version: '1.0.0', date: '2025-03-20', changeLog: '正式版发布', files: [
        { name: 'SKILL.md', size: 1024, content: githubSkillFiles['SKILL.md'] },
        { name: 'hha/ha.md', size: 512, content: githubSkillFiles['hha/ha.md'] },
      ]},
      { version: '0.9.0', date: '2025-03-06', changeLog: '1、新增 gh api 功能\n2、更新SKILL文件', files: [
        { name: 'SKILL.md', size: 900, content: '---\nname: github\ndescription: "Interact with GitHub using the gh CLI."\n---\n\n# GitHub Skill (v0.9)\n\nBeta version with gh api support.' },
      ]},
      { version: '0.8.0', date: '2025-02-20', changeLog: '内测版本', files: [
        { name: 'SKILL.md', size: 800, content: '---\nname: github\ndescription: "GitHub CLI integration."\n---\n\n# GitHub Skill (v0.8)\n\nAlpha version.' },
      ]},
    ],
  },
  {
    id: 'skill-5',
    slug: 'sql-optimizer',
    name: 'SQL 查询优化助手',
    description: '自动分析 SQL 查询语句，识别性能瓶颈并提供优化建议。支持 MySQL、PostgreSQL、TiDB 等主流数据库，提供索引建议、查询重写、执行计划分析等功能。',
    version: '1.2.0',
    categories: ['8', '2'],
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-03-25'),
    securityInfo: MOCK_SECURITY_SAFE,
    content: '# SQL 查询优化助手\n\n这是一个用于 SQL 查询优化的 Skill...',
    versions: ['1.2.0', '1.1.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# SQL 查询优化助手\n\n这是一个用于 SQL 查询优化的 Skill...' },
      { name: 'README.md', size: 512, content: '# SQL Optimizer\n\n## 功能\n\n- 自动分析 SQL 查询性能\n- 索引建议\n- 查询重写优化' },
    ],
    versionHistory: [
      { version: '1.2.0', date: '2025-03-25', changeLog: '1、新增 TiDB 支持\n2、优化索引建议算法\n3、新增执行计划可视化', files: [
        { name: 'SKILL.md', size: 1024, content: '# SQL 查询优化助手\n\n这是一个用于 SQL 查询优化的 Skill...' },
      ]},
      { version: '1.1.0', date: '2025-03-10', changeLog: '1、新增 PostgreSQL 支持\n2、优化建议准确率提升 30%', files: [
        { name: 'SKILL.md', size: 980, content: '# SQL 查询优化助手 v1.1\n\n新增 PostgreSQL 支持。' },
      ]},
      { version: '1.0.0', date: '2025-02-20', changeLog: '首次发布，支持 MySQL', files: [
        { name: 'SKILL.md', size: 900, content: '# SQL 查询优化助手 v1.0\n\n支持 MySQL 查询优化。' },
      ]},
    ],
  },
  {
    id: 'skill-6',
    slug: 'k8s-troubleshooter',
    name: 'K8s 故障排查助手',
    description: '面向 Kubernetes 集群的智能故障排查工具。支持 Pod 异常诊断、网络问题定位、资源配额分析、Events 日志解读，快速定位并修复集群问题。',
    version: '2.0.0',
    categories: ['3'],
    scope: 'private',
    groupIds: ['grp-2'],
    uploadTime: new Date('2025-04-01'),
    securityInfo: MOCK_SECURITY_NOT_SCANNED,
    content: '# K8s 故障排查助手\n\n这是一个用于 Kubernetes 故障排查的 Skill...',
    versions: ['2.0.0', '1.5.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# K8s 故障排查助手\n\n这是一个用于 Kubernetes 故障排查的 Skill...' },
      { name: 'README.md', size: 512, content: '# K8s Troubleshooter\n\n## 功能\n\n- Pod 异常诊断\n- 网络问题定位\n- 资源配额分析' },
    ],
    versionHistory: [
      { version: '2.0.0', date: '2025-04-01', changeLog: '1、全新诊断引擎，支持多集群\n2、新增网络拓扑可视化\n3、支持自定义告警规则', files: [
        { name: 'SKILL.md', size: 1024, content: '# K8s 故障排查助手\n\n这是一个用于 Kubernetes 故障排查的 Skill...' },
      ]},
      { version: '1.5.0', date: '2025-03-15', changeLog: '1、新增 Events 日志智能解读\n2、优化 Pod 异常诊断流程', files: [
        { name: 'SKILL.md', size: 980, content: '# K8s 故障排查助手 v1.5\n\n新增 Events 日志智能解读。' },
      ]},
      { version: '1.0.0', date: '2025-02-28', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 900, content: '# K8s 故障排查助手 v1.0\n\n基础 Kubernetes 故障排查。' },
      ]},
    ],
  },
  {
    id: 'skill-7',
    slug: 'meeting-summary',
    name: '会议纪要生成器',
    description: '基于会议录音或文字记录，自动生成结构化会议纪要。支持议题提取、待办事项追踪、决策记录，并可与企业微信/飞书集成推送。',
    version: '1.3.0',
    categories: ['1', '7'],
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-03-28'),
    securityInfo: MOCK_SECURITY_SUSPICIOUS,
    content: '# 会议纪要生成器\n\n这是一个用于会议纪要生成的 Skill...',
    versions: ['1.3.0', '1.2.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# 会议纪要生成器\n\n这是一个用于会议纪要生成的 Skill...' },
      { name: 'README.md', size: 512, content: '# Meeting Summary\n\n## 功能\n\n- 自动提取议题和决策\n- 生成待办事项\n- 支持企业微信推送' },
    ],
    versionHistory: [
      { version: '1.3.0', date: '2025-03-28', changeLog: '1、新增飞书集成\n2、优化议题提取准确率\n3、支持多语言会议记录', files: [
        { name: 'SKILL.md', size: 1024, content: '# 会议纪要生成器\n\n这是一个用于会议纪要生成的 Skill...' },
      ]},
      { version: '1.2.0', date: '2025-03-10', changeLog: '1、新增企业微信推送\n2、优化待办事项追踪', files: [
        { name: 'SKILL.md', size: 980, content: '# 会议纪要生成器 v1.2\n\n新增企业微信推送。' },
      ]},
      { version: '1.0.0', date: '2025-02-25', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 900, content: '# 会议纪要生成器 v1.0\n\n基础会议纪要生成。' },
      ]},
    ],
  },
  {
    id: 'skill-8',
    slug: 'api-doc-generator',
    name: 'API 文档生成器',
    description: '根据代码注释和接口定义自动生成 API 文档。支持 OpenAPI/Swagger 规范、多语言代码解析，可导出 Markdown、HTML 等格式，与 CI/CD 集成自动更新。',
    version: '1.0.0',
    categories: ['2', '1'],
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-04-08'),
    securityInfo: MOCK_SECURITY_SAFE,
    content: '# API 文档生成器\n\n根据代码注释和接口定义自动生成标准化 API 文档的 Skill...',
    versions: ['1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# API 文档生成器\n\n根据代码注释和接口定义自动生成标准化 API 文档的 Skill...' },
      { name: 'README.md', size: 512, content: '# API Doc Generator\n\n## Features\n\n- OpenAPI/Swagger 规范生成\n- 多语言代码解析\n- 导出 Markdown/HTML' },
      { name: '_meta.json', size: 256, content: '{\n  "name": "api-doc-generator",\n  "version": "1.0.0",\n  "description": "API 文档自动生成"\n}' },
    ],
    versionHistory: [
      { version: '1.0.0', date: '2025-04-08', changeLog: '首次发布，支持 OpenAPI 规范和多语言解析', files: [
        { name: 'SKILL.md', size: 1024, content: '# API 文档生成器\n\n根据代码注释和接口定义自动生成标准化 API 文档的 Skill...' },
        { name: 'README.md', size: 512, content: '# API Doc Generator\n\n## Features\n\n- OpenAPI/Swagger 规范生成' },
        { name: '_meta.json', size: 256, content: '{\n  "name": "api-doc-generator",\n  "version": "1.0.0"\n}' },
      ]},
    ],
  },
  {
    id: 'skill-9',
    slug: 'performance-monitor',
    name: '性能监控分析器',
    description: '实时监控应用性能指标，自动识别性能瓶颈和异常。支持 CPU/内存/IO 分析、慢查询检测、调用链追踪，并提供优化建议和告警通知。',
    version: '2.1.0',
    categories: ['3', '5'],
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-04-12'),
    securityInfo: MOCK_SECURITY_NOT_SCANNED,
    content: '# 性能监控分析器\n\n实时监控应用性能指标并自动识别瓶颈的 Skill...',
    versions: ['2.1.0', '2.0.0', '1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1200, content: '# 性能监控分析器\n\n实时监控应用性能指标并自动识别瓶颈的 Skill...' },
      { name: 'README.md', size: 600, content: '# Performance Monitor\n\n## Features\n\n- CPU/内存/IO 实时监控\n- 慢查询自动检测\n- 调用链可视化追踪' },
      { name: '_meta.json', size: 256, content: '{\n  "name": "performance-monitor",\n  "version": "2.1.0",\n  "description": "性能监控分析"\n}' },
      { name: 'dist/', size: 0, content: '' },
      { name: 'dist/index.js', size: 4096, content: '// Performance monitor entry point\nmodule.exports = { init() { /* ... */ } }' },
    ],
    versionHistory: [
      { version: '2.1.0', date: '2025-04-12', changeLog: '1、新增调用链追踪可视化\n2、优化告警灵敏度\n3、支持自定义监控指标', files: [
        { name: 'SKILL.md', size: 1200, content: '# 性能监控分析器\n\n实时监控应用性能指标并自动识别瓶颈的 Skill...' },
        { name: 'README.md', size: 600, content: '# Performance Monitor v2.1\n\n新增调用链追踪可视化。' },
        { name: '_meta.json', size: 256, content: '{\n  "name": "performance-monitor",\n  "version": "2.1.0"\n}' },
        { name: 'dist/', size: 0, content: '' },
        { name: 'dist/index.js', size: 4096, content: '// Performance monitor v2.1 entry point' },
      ]},
      { version: '2.0.0', date: '2025-03-20', changeLog: '1、全新架构重构\n2、新增慢查询自动检测', files: [
        { name: 'SKILL.md', size: 1100, content: '# 性能监控分析器 v2.0\n\n全新架构，新增慢查询检测。' },
      ]},
      { version: '1.0.0', date: '2025-02-15', changeLog: '首次发布', files: [
        { name: 'SKILL.md', size: 900, content: '# 性能监控分析器 v1.0\n\n基础性能监控功能。' },
      ]},
    ],
  },
  // skill-pending-1: 员工提交的待审核 Skill（对齐管控端 Demo：会议纪要助手 / meeting-summary-helper / 刘敏）
  {
    id: 'skill-pending-1',
    slug: 'meeting-summary-helper',
    name: '会议纪要助手',
    description: '自动整理会议录音转写文本，生成结构化会议纪要与待办项，支持导出到企业微信/邮件。',
    version: '1.0.0',
    categories: ['1'], // 通用办公
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2026-07-23T14:30:00'),
    securityInfo: MOCK_SECURITY_SAFE,
    content: '# 会议纪要助手\n\n自动整理会议录音转写文本，生成结构化会议纪要与待办项。',
    versions: ['1.0.0'],
    files: [
      { name: 'SKILL.md', size: 1024, content: '# 会议纪要助手\n\n自动整理会议录音转写文本，生成结构化会议纪要与待办项。' },
    ],
    versionHistory: [
      { version: '1.0.0', date: '2026-07-23', changeLog: '首次提交，等待管理员审核', files: [
        { name: 'SKILL.md', size: 1024, content: '# 会议纪要助手\n\n自动整理会议录音转写文本，生成结构化会议纪要与待办项。' },
      ]},
    ],
    reviewStatus: 'pending',
    reviewType: 'publish',
    applicant: '刘敏',
    submittedAt: '2026-07-23 14:30',
  },
  // skill-pending-2: 员工提交的「下架申请」，需要管理员审批
  {
    id: 'skill-pending-2',
    slug: 'daily-weekly-report-gen',
    name: '日报周报生成器',
    description: '一键汇总本人本周提交的 MR、TAPD 事项、日历会议，生成日报/周报草稿。',
    version: '2.1.0',
    categories: ['1'], // 通用办公
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2026-05-12T10:00:00'),
    securityInfo: MOCK_SECURITY_SAFE,
    content: '# 日报周报生成器\n\n一键汇总本人本周提交的 MR、TAPD 事项、日历会议。',
    versions: ['1.0.0', '2.0.0', '2.1.0'],
    files: [
      { name: 'SKILL.md', size: 1500, content: '# 日报周报生成器\n\n一键汇总本人本周提交的 MR、TAPD 事项、日历会议。' },
    ],
    versionHistory: [
      { version: '2.1.0', date: '2026-06-20', changeLog: '修复周末数据错位' },
      { version: '2.0.0', date: '2026-05-12', changeLog: '接入 TAPD OpenAPI' },
      { version: '1.0.0', date: '2026-04-01', changeLog: '首次发布' },
    ],
    reviewStatus: 'pending',
    reviewType: 'offshelf',
    applicant: '张伟',
    submittedAt: '2026-07-22 09:15',
    offshelfReason: '已被新版「智能周报」替代，且原版依赖的旧接口即将下线，避免误用。',
  },
  // skill-offlined-1: 已下架 Skill（管控端仍可见/可管理，员工端搜不到）
  {
    id: 'skill-offlined-1',
    slug: 'legacy-translator',
    name: '旧版翻译助手',
    description: '基于旧版翻译接口的多语言互译助手，已被新版「AI 翻译」替代。',
    version: '0.9.5',
    categories: ['2'], // 提示词工程
    scope: 'public',
    groupIds: [],
    uploadTime: new Date('2025-11-20T16:00:00'),
    securityInfo: MOCK_SECURITY_SAFE,
    content: '# 旧版翻译助手\n\n基于旧版翻译接口的多语言互译助手。',
    versions: ['0.9.5'],
    files: [
      { name: 'SKILL.md', size: 800, content: '# 旧版翻译助手\n\n已下架版本，仅供已安装用户继续使用。' },
    ],
    versionHistory: [
      { version: '0.9.5', date: '2025-11-20', changeLog: '最后一次维护版本' },
    ],
    reviewStatus: 'offlined',
  },
];

export const MOCK_OPENCLAW_INSTANCES: OpenClawInstance[] = [
  { id: 'local-3', name: 'CodeBuddy-研发本地环境', createdBy: 'developer', status: 'running', createdAt: '2026-04-09T10:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2'], agentType: 'LocalAgent', localProduct: 'CodeBuddy', agentVersion: 'codebuddy-1.8.0' },
  { id: 'local-2', name: 'WorkBuddy-运营笔记本', createdBy: 'olivia', status: 'running', createdAt: '2026-04-09T10:00:00Z', distributionStatus: 'success', distributedVersion: '2.0.0', groupIds: ['grp-1'], agentType: 'LocalAgent', localProduct: 'WorkBuddy', agentVersion: 'workbuddy-2.3.1' },
  { id: 'local-1', name: 'WorkBuddy-离线调试机', createdBy: 'developer', status: 'running', createdAt: '2026-04-09T09:30:00Z', distributionStatus: 'failed', groupIds: ['grp-2'], failReason: 'WorkBuddy 暂未在线', agentType: 'LocalAgent', localProduct: 'WorkBuddy', agentVersion: 'workbuddy-2.3.1' },
  { id: 'oc-7', name: 'OpenClaw-预发布环境', createdBy: 'admin', status: 'running', createdAt: '2026-04-05T11:00:00Z', distributionStatus: 'success', distributedVersion: '0.8.0', groupIds: ['grp-1', 'grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-6', name: 'OpenClaw-回归测试', createdBy: 'dev-team', status: 'running', createdAt: '2026-04-01T09:30:00Z', distributionStatus: 'success', distributedVersion: '1.0.0', groupIds: ['grp-2', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.10' },
  { id: 'oc-5', name: 'OpenClaw-灾备中心', createdBy: 'admin', status: 'running', createdAt: '2026-03-28T10:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-4', name: 'OpenClaw-备用实例', createdBy: 'ops', status: 'running', createdAt: '2026-03-20T14:30:00Z', distributionStatus: 'failed', groupIds: ['grp-2'], failReason: '命令下发失败', agentType: 'OpenClaw', agentVersion: '2026.3.15' },
  { id: 'oc-3', name: 'OpenClaw-开发环境', createdBy: 'developer', status: 'stopped', createdAt: '2026-03-15T09:00:00Z', distributionStatus: 'success', distributedVersion: '1.0.0', groupIds: ['grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.8' },
  { id: 'oc-2', name: 'OpenClaw-测试环境', createdBy: 'dev-team', status: 'running', createdAt: '2026-03-10T16:45:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1', 'grp-2'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-1', name: 'OpenClaw-生产环境', createdBy: 'admin', status: 'running', createdAt: '2026-02-01T08:00:00Z', distributionStatus: 'success', distributedVersion: '0.9.0', groupIds: ['grp-1', 'grp-2', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.3.20' },
  // 额外 mock 数据用于测试分页和跨页全选
  { id: 'oc-8', name: 'OpenClaw-华南节点A', createdBy: 'ops', status: 'running', createdAt: '2026-04-06T08:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-9', name: 'OpenClaw-华南节点B', createdBy: 'ops', status: 'running', createdAt: '2026-04-06T08:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-10', name: 'OpenClaw-华东节点A', createdBy: 'admin', status: 'running', createdAt: '2026-04-06T09:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-11', name: 'OpenClaw-华东节点B', createdBy: 'admin', status: 'running', createdAt: '2026-04-06T09:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-12', name: 'OpenClaw-华北节点A', createdBy: 'dev-team', status: 'running', createdAt: '2026-04-06T10:00:00Z', distributionStatus: 'failed', groupIds: ['grp-3'], failReason: '网络超时', agentType: 'OpenClaw', agentVersion: '2026.4.10' },
  { id: 'oc-13', name: 'OpenClaw-华北节点B', createdBy: 'dev-team', status: 'running', createdAt: '2026-04-06T10:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.10' },
  { id: 'oc-14', name: 'OpenClaw-西南节点', createdBy: 'ops', status: 'running', createdAt: '2026-04-06T11:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1', 'grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-15', name: 'OpenClaw-西北节点', createdBy: 'ops', status: 'running', createdAt: '2026-04-06T11:30:00Z', distributionStatus: 'failed', groupIds: ['grp-1'], failReason: '权限不足', agentType: 'OpenClaw', agentVersion: '2026.3.15' },
  { id: 'oc-16', name: 'OpenClaw-港澳节点A', createdBy: 'admin', status: 'running', createdAt: '2026-04-07T08:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-17', name: 'OpenClaw-港澳节点B', createdBy: 'admin', status: 'running', createdAt: '2026-04-07T08:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-18', name: 'OpenClaw-东南亚节点', createdBy: 'ops', status: 'running', createdAt: '2026-04-07T09:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-19', name: 'OpenClaw-欧洲节点', createdBy: 'ops', status: 'running', createdAt: '2026-04-07T09:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.10' },
  { id: 'oc-20', name: 'OpenClaw-北美节点A', createdBy: 'admin', status: 'running', createdAt: '2026-04-07T10:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-21', name: 'OpenClaw-北美节点B', createdBy: 'admin', status: 'running', createdAt: '2026-04-07T10:30:00Z', distributionStatus: 'failed', groupIds: ['grp-2'], failReason: '配置冲突', agentType: 'OpenClaw', agentVersion: '2026.3.15' },
  { id: 'oc-22', name: 'OpenClaw-日韩节点', createdBy: 'dev-team', status: 'running', createdAt: '2026-04-07T11:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-23', name: 'OpenClaw-压测环境A', createdBy: 'dev-team', status: 'running', createdAt: '2026-04-08T08:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2'], agentType: 'OpenClaw', agentVersion: '2026.4.10' },
  { id: 'oc-24', name: 'OpenClaw-压测环境B', createdBy: 'developer', status: 'running', createdAt: '2026-04-08T08:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-25', name: 'OpenClaw-沙箱环境', createdBy: 'developer', status: 'running', createdAt: '2026-04-08T09:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1', 'grp-2'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-26', name: 'OpenClaw-安全审计', createdBy: 'admin', status: 'running', createdAt: '2026-04-08T09:30:00Z', distributionStatus: 'failed', groupIds: ['grp-1', 'grp-3'], failReason: '证书过期', agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-27', name: 'OpenClaw-监控中心', createdBy: 'ops', status: 'running', createdAt: '2026-04-08T10:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1'], agentType: 'OpenClaw', agentVersion: '2026.4.10' },
  { id: 'oc-28', name: 'OpenClaw-日志分析', createdBy: 'ops', status: 'running', createdAt: '2026-04-08T10:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-3'], agentType: 'OpenClaw', agentVersion: '2026.3.28' },
  { id: 'oc-29', name: 'OpenClaw-数据中台', createdBy: 'admin', status: 'running', createdAt: '2026-04-08T11:00:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-1', 'grp-2', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.1' },
  { id: 'oc-30', name: 'OpenClaw-AI训练集群', createdBy: 'dev-team', status: 'running', createdAt: '2026-04-08T11:30:00Z', distributionStatus: 'not_distributed', groupIds: ['grp-2', 'grp-3'], agentType: 'OpenClaw', agentVersion: '2026.4.10' },
];

export const MOCK_CLOUD_OPENCLAW_INSTANCES: OpenClawInstance[] = MOCK_OPENCLAW_INSTANCES.filter(
  (instance) => instance.agentType !== 'LocalAgent',
);
