 

export const [TAB_LIST, TAB_WHITE] = ['eventList', 'whiteList'];

export const MALICIOUS_STATUS_OPTIONS = [
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
    value: '4',
    icon: 'success',
    theme: 'success',
  },
  {
    text: '已忽略',
    value: '5',
    icon: 'dismiss',
    theme: 'info',
  },
];

export const MALICIOUS_STATUS_DATA = [
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
    value: '4',
    icon: 'success',
    theme: 'success',
  },
  {
    text: '已忽略',
    value: '5',
    icon: 'dismiss',
    theme: 'info',
  },
  {
    text: '已拦截',
    value: '6',
    icon: 'success',
    theme: 'success',
  },
];

export const MALICIOUS_STATUS_FILTERS = [
  {
    name: '待处理',
    key: '0',
  },
  {
    name: '已加白',
    key: '2',
  },
  {
    name: '已处理',
    key: '4',
  },
  {
    name: '已忽略',
    key: '5',
  },
];
export const ALL_STATUS = MALICIOUS_STATUS_DATA.map(item => item.value);

export const statusObjMap = MALICIOUS_STATUS_OPTIONS.reduce((pre:any, cur) => {
  pre[cur.value] = {
    text: cur.text || '未知',
    icon: cur.icon || 'help',
    theme: cur.theme || 'info',
  };
  return pre;
}, {});

export const statusObjMapNew = MALICIOUS_STATUS_DATA.reduce((pre:any, cur) => {
  pre[cur.value] = {
    text: cur.text || '未知',
    icon: cur.icon || 'help',
    theme: cur.theme || 'info',
  };
  return pre;
}, {});

export const POLICY_TYPES:any = {
  0: '系统策略',
  1: '用户自定义策略',
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
    value: 'undefined',
  },
].concat(
  Object.keys(POLICY_TYPES).map(item => ({
    text: POLICY_TYPES[item],
    value: item,
  })),
);

export const getPolicyActionMap:any = () => ({
  0: '告警',
  1: '放行',
  2: '拦截',
});

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
    1: '全部专业版、旗舰版OpenClaw',
    2: '全部旗舰版OpenClaw',
    0: '部分OpenClaw',
    3: '全部OpenClaw',
  };
  return mapData?.[hostScope];
};
