export const DETAIL_ID = {
  DEFAULT: 'list',
  LIST: 'list',
  USER_RULE: 'user_rule',
  SYS_RULE: 'sys_rule',
};

export const DATA_SOURCE_MAP = {
  0: 'bash内存',
  1: '实时监控',
};

export const BASH_STATUS_MAP = {
  0: '待处理',
  1: '已手动处理',
  2: '已加白名单',
};

export const BASH_RULE_CATEGORY_MAP = {
  0: '系统规则',
  1: '用户规则',
};

export const BASH_RULE_WHITE_MP = {
  0: '黑名单',
  1: '白名单',
};

export const BASH_DETAIL_TORULE = 'BASH_DETAIL_TORULE';
export const BASH_DETAIL_TOCREATE = 'BASH_DETAIL_TOCREATE';
export const CSIP_AI_AGENT_BATCH_TIPS = 'CSIP_AI_AGENT_BATCH_TIPS4';

export const SYSTEM_STANDARD_ID = 1000; // 系统规则(标准)
export const SYSTEM_VIP_ID = 1001; // 系统规则(重保)
export const BLOCK_STANDARD_ID = 2001; // 系统自动拦截规则(标准)
export const BLOCK_DEEP_ID = 2002; // 系统自动拦截规则(重保)

export const RulesAttributeMap:any = { Process: '进程', PProcess: '父进程', AProcess: '祖先进程' };

export const BASH_STATUS_OPTIONS = [
  {
    text: '全部状态',
    value: '',
  },
  {
    text: '待处理',
    value: '0',
    icon: 'un_do',
    theme: 'error',
  },
  {
    text: '已加白',
    value: '2',
    icon: 'success',
    theme: 'success',
  },
  {
    text: '已处理',
    value: '1',
    icon: 'success',
    theme: 'success',
  },
  {
    text: '已忽略',
    value: '3',
    icon: 'dismiss',
    theme: 'info',
  },
];

export const statusObjMap = BASH_STATUS_OPTIONS.reduce((pre:any, cur) => {
  pre[cur.value] = {
    text: cur.text || '未知',
    icon: cur.icon || 'help',
    theme: cur.theme || 'info',
  };
  return pre;
}, {});

export const POLICY_TYPES:any = {
  0: '系统策略',
  1: '用户自定义',
};

export const POLICY_TYPES_DATA = [
  {
    text: '全部命中策略类型',
    value: 'undefined',
  },
].concat(
  Object.keys(POLICY_TYPES).map(item => ({
    text: POLICY_TYPES[item],
    value: item,
  })),
);
export const ALL_POLICY_TYPES_DATA = [
  {
    text: '全部策略类型',
    value: '',
  },
].concat(
  Object.keys(POLICY_TYPES).map(item => ({
    text: POLICY_TYPES[item],
    value: item,
  })),
);

export const getPolicyActionMap = () => ({
  0: '告警',
  1: '放行',
  2: '拦截',
}) as any;

export const POLICY_ACTION_THEME_MAP:any = {
  0: 'warning',
  1: 'success',
  2: 'error',
};

export const getPolicyActionsData = () => {
  const policyActionMap:any = getPolicyActionMap();
  return [
    {
      text: '全部执行动作',
      value: 'undefined',
    },
  ].concat(
    Object.keys(policyActionMap).map(item => ({
      text: policyActionMap[item],
      value: item,
    })),
  );
};

export const GetHostTypeText = (hostScope: number = 1) => {
  const mapData:any = {
    3: '全部OpenClaw',
    2: '全部旗舰版OpenClaw',
    1: '全部专业版、旗舰版OpenClaw',
    0: '部分OpenClaw',
  };
  return mapData?.[hostScope];
};

export const BASH_LEVEL_DATA = [
  { text: '高危', value: '1' },
  { text: '中危', value: '2' },
  { text: '低危', value: '3' },
];

export const BASH_LEVEL_ALL = BASH_LEVEL_DATA.map(item => item.value);
export const BASH_LEVEL_MAP = BASH_LEVEL_DATA.reduce((pre:any, cur) => {
  pre[cur.value] = cur.text;
  return pre;
}, {});

export const BASH_POLICY_LEVEL_DATA = [
  { text: '高危', value: '1' },
  { text: '中危', value: '2' },
  { text: '低危', value: '3' },
  { text: '无', value: '0' },
];
export const BASH_POLICY_LEVEL_ALL = BASH_POLICY_LEVEL_DATA.map(item => item.value);
export const BASH_POLICY_LEVEL_MAP = BASH_POLICY_LEVEL_DATA.reduce((pre:any, cur) => {
  pre[cur.value] = cur.text;
  return pre;
}, {});

export const BASH_STATUS_DATA = [
  {
    text: '待处理',
    value: '0',
    icon: 'un_do',
    theme: 'error',
  },
  {
    text: '已加白',
    value: '2',
    icon: 'success',
    theme: 'success',
  },
  {
    text: '已处理',
    value: '1',
    icon: 'success',
    theme: 'success',
  },
  {
    text: '已忽略',
    value: '3',
    icon: 'dismiss',
    theme: 'info',
  },
  {
    text: '已拦截',
    value: '5',
    icon: 'success',
    theme: 'success',
  },
];

export const BASH_STATUS_ALL = BASH_STATUS_DATA.map(item => item.value);

export const statusObjMapNew = BASH_STATUS_DATA.reduce((pre:any, cur) => {
  pre[cur.value] = {
    text: cur.text || '未知',
    icon: cur.icon || 'help',
    theme: cur.theme || 'info',
  };
  return pre;
}, {});

export const PROCESS_TYPES = [
  { text: '进程', value: 'Process' },
  { text: '父进程', value: 'PProcess' },
  { text: '祖先进程', value: 'AProcess' },
];

export const PROCESS_TYPES_MAP = PROCESS_TYPES.reduce((pre:any, cur) => {
  pre[cur.value] = cur.text;
  return pre;
}, {});

export const LICENSE_TYPES_MAP:any = {
  0: '专业版',
  1: '专业版',
  2: '旗舰版',
  3: '轻量版(Lighthouse)',
};

export const hostVersionMap:any = {
  Flagship: '旗舰版',
  PRO_VERSION: '专业版',
  BASIC_VERSION: '基础版',
  GENERAL_DISCOUNT: '轻量版(Lighthouse)',
  '-': '未安装',
};

export const heightMap:any = {
  1: 30,
  2: 50,
  3: 68,
  4: 85,
  5: 105,
  6: 120,
  7: 140,
  8: 158,
  9: 175,
  10: 195,
  11: 212,
  12: 228,
};
