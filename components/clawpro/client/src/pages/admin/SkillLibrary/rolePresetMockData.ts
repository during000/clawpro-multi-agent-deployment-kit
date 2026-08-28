/**
 * 角色设定 Mock 数据
 * 管理员可创建角色（如开发虾、写作虾等），每个角色包含从公共/企业技能库挑选的技能
 */

export interface RoleSkillItem {
  skillId: string;
  skillName: string;
  skillNameZh?: string;
  source: 'public' | 'enterprise';
  version: string;
}

export interface RolePreset {
  id: string;
  name: string;           // 角色名称，如"开发虾"
  emoji: string;          // 角色 emoji 图标
  description: string;    // 角色描述
  isEnabled: boolean;     // 是否启用（用户端可见）
  skills: RoleSkillItem[];
  createdAt: Date;
  updatedAt: Date;
}

// ─── Mock 数据 ────────────────────────────────────────────────────────────────

export const MOCK_ROLE_PRESETS: RolePreset[] = [
  {
    id: 'role-1',
    name: '开发虾',
    emoji: '💻',
    description: '专注于代码开发、代码审查、接口调试和技术文档解析，适合研发团队日常使用。',
    isEnabled: true,
    skills: [
      { skillId: 'skill-2', skillName: 'code-reviewer', skillNameZh: '代码审查工具', source: 'enterprise', version: '1.2.0' },
      { skillId: 'pub-2', skillName: 'github', skillNameZh: 'GitHub 工具', source: 'public', version: '2.1.0' },
      { skillId: 'pub-4', skillName: 'code-reviewer', skillNameZh: '代码审查助手', source: 'public', version: '1.0.3' },
      { skillId: 'pub-7', skillName: 'docker-ops', skillNameZh: 'Docker 运维', source: 'public', version: '1.1.0' },
    ],
    createdAt: new Date('2026-03-01'),
    updatedAt: new Date('2026-03-15'),
  },
  {
    id: 'role-2',
    name: '写作虾',
    emoji: '✍️',
    description: '擅长文档撰写、邮件润色、内容创作和 PPT 大纲生成，适合运营、市场和行政团队。',
    isEnabled: true,
    skills: [
      { skillId: 'skill-1', skillName: '文档总结助手', skillNameZh: '文档总结助手', source: 'enterprise', version: '1.0.0' },
      { skillId: 'pub-8', skillName: 'email-writer', skillNameZh: '邮件撰写助手', source: 'public', version: '1.2.1' },
      { skillId: 'pub-1', skillName: 'self-improving-agent', skillNameZh: '自我改进代理', source: 'public', version: '1.0.0' },
    ],
    createdAt: new Date('2026-03-05'),
    updatedAt: new Date('2026-03-20'),
  },
  {
    id: 'role-3',
    name: '数据虾',
    emoji: '📊',
    description: '专注于数据分析、SQL 查询、报表解读和数据清洗，适合数据团队和业务分析师。',
    isEnabled: true,
    skills: [
      { skillId: 'pub-5', skillName: 'data-analyst', skillNameZh: '数据分析专家', source: 'public', version: '2.0.0' },
      { skillId: 'pub-6', skillName: 'sql-expert', skillNameZh: 'SQL 专家', source: 'public', version: '1.3.0' },
      { skillId: 'pub-3', skillName: 'web-search-pro', skillNameZh: '网络搜索增强', source: 'public', version: '1.1.2' },
    ],
    createdAt: new Date('2026-03-08'),
    updatedAt: new Date('2026-03-22'),
  },
  {
    id: 'role-4',
    name: '财务虾',
    emoji: '💰',
    description: '协助财务报表分析、预算规划、数据核对和财务文档整理，适合财务和审计团队。',
    isEnabled: false,
    skills: [
      { skillId: 'pub-5', skillName: 'data-analyst', skillNameZh: '数据分析专家', source: 'public', version: '2.0.0' },
      { skillId: 'pub-6', skillName: 'sql-expert', skillNameZh: 'SQL 专家', source: 'public', version: '1.3.0' },
      { skillId: 'skill-1', skillName: '文档总结助手', skillNameZh: '文档总结助手', source: 'enterprise', version: '1.0.0' },
    ],
    createdAt: new Date('2026-03-10'),
    updatedAt: new Date('2026-03-10'),
  },
];

// ─── 可选技能列表（公共 + 企业合并，用于角色编辑时挑选） ─────────────────────

export interface SelectableSkill {
  skillId: string;
  skillName: string;
  skillNameZh: string;
  source: 'public' | 'enterprise';
  version: string;
  category: string;
}

export const SELECTABLE_SKILLS: SelectableSkill[] = [
  // 公共技能库
  { skillId: 'pub-1', skillName: 'self-improving-agent', skillNameZh: '自我改进代理', source: 'public', version: '1.0.0', category: '通用工具' },
  { skillId: 'pub-2', skillName: 'github', skillNameZh: 'GitHub 工具', source: 'public', version: '2.1.0', category: '开发工具' },
  { skillId: 'pub-3', skillName: 'web-search-pro', skillNameZh: '网络搜索增强', source: 'public', version: '1.1.2', category: '信息检索' },
  { skillId: 'pub-4', skillName: 'code-reviewer', skillNameZh: '代码审查助手', source: 'public', version: '1.0.3', category: '开发工具' },
  { skillId: 'pub-5', skillName: 'data-analyst', skillNameZh: '数据分析专家', source: 'public', version: '2.0.0', category: '数据分析' },
  { skillId: 'pub-6', skillName: 'sql-expert', skillNameZh: 'SQL 专家', source: 'public', version: '1.3.0', category: '数据分析' },
  { skillId: 'pub-7', skillName: 'docker-ops', skillNameZh: 'Docker 运维', source: 'public', version: '1.1.0', category: '系统运维' },
  { skillId: 'pub-8', skillName: 'email-writer', skillNameZh: '邮件撰写助手', source: 'public', version: '1.2.1', category: '通用办公' },
  // 企业技能库
  { skillId: 'skill-1', skillName: '文档总结助手', skillNameZh: '文档总结助手', source: 'enterprise', version: '1.0.0', category: '通用办公' },
  { skillId: 'skill-2', skillName: '代码审查工具', skillNameZh: '代码审查工具', source: 'enterprise', version: '1.2.0', category: '开发工具' },
  { skillId: 'skill-3', skillName: '日志分析器', skillNameZh: '日志分析器', source: 'enterprise', version: '2.0.1', category: '系统运维' },
  { skillId: 'skill-4', skillName: 'GitHub', skillNameZh: 'GitHub', source: 'enterprise', version: '1.0.0', category: '开发工具' },
];
