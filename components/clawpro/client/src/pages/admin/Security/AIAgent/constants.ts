import moment from '@/vendor/moment';

export const PRODUCT_NAME = "cwp";
export const PRODUCT_TITLE = "OpenClaw";

export const FORMAT_DATE_TIME = 'YYYY-MM-DD HH:mm:ss';

export const [AGENT, COMPONENT] = ["AI Agent", "AI-Component"];

// 近${interval}天
export const getLastDateRange = (interval: number): any => [
  moment()
    .subtract(interval - 1, 'd')
    .startOf('d'),
  moment().endOf('d'),
];

export const [ASSETS, CONTROL, LOGS, ALARMS, SKILLS] = [
  "ASSETS",
  "CONTROL",
  "LOGS",
  "ALARMS",
  "SKILLS",
];

export const [BASH_ALARM, MALICIOUS_ALARM]:any = ["0", "1"];
export const [BASH_POLICY, MALICIOUS_POLICY] = [
  "BASH_POLICY",
  "MALICIOUS_POLICY",
];

export const UNAUTH_CODE = [
  "OperationDenied",
  "UnauthorizedOperation",
  "AuthFailure.UnauthorizedOperation",
];

export const RISK_TYPE_MALICIOUS = "MALICIOUS_REQUEST";
export const RISK_TYPE_BASH = "BASH_EVENT";

export const CONSOLE_URL = 'https://console.cloud.tencent.com';
export const AUTHORIZE_ROUTE = `${CONSOLE_URL}/cwp/setting/authorize`;

export const GROUP_NAME = "云安全中心AI Agent内网管控安全组";
export const CSIP_AI_AGENT_NET_RULE = "CSIP_AI_AGENT_NET_RULE";

// 链接
const domainList = (window as any)?.tcesdk?.domain?.getDomainList?.() ?? {};
export const { protocol } = location;
export const buyDomain = `${protocol}//${domainList?.buy ?? (window as any).QCBUY_HOST}`;
export const buyUrl = `${buyDomain}/yunjing`;

export const checkMachineIsWindows = (item: any) =>
  item?.MachineOs?.toLowerCase?.()?.indexOf?.("window") > -1;

export const AGENT_NAME_ARR = [
  "OpenClaw",
  "ZeroClaw",
  "NanoClaw",
  "NanoBot",
  "PicoClaw",
  "ClawBot",
  "MoltBot",
];

export const FORMAT_BEGIN = "YYYY-MM-DD 00:00:00";
export const FORMAT_END = "YYYY-MM-DD 23:59:59";
export const FORMAT_NOW = "YYYY-MM-DD HH:mm:ss";
export const FORMAT_OFF = "1970-01-01 00:00:00";
export const FORMAT_DATE = "YYYY-MM-DD";
export const FORMAT_HOUR_BEGIN = "YYYY-MM-DD HH:00:00";
export const FORMAT_HOUR_END = "YYYY-MM-DD HH:59:59";
export const FORMAT_MONTH_BEGIN = "YYYY-MM-01 00:00:00";
export const FORMAT_MONTH_END = "YYYY-MM-31 23:59:59";
export const FORMAT_HOUR_MINUTE = "HH:mm";
export const FORMAT_TIME = "YYYYMMDDHHmmss";

const all = {
  value: "ALL",
  text: "全部服务器",
};

const cvm = {
  value: "CVM",
  text: "腾讯云服务器",
};

const eks = {
  value: "EKS-NATIVE",
  text: "腾讯容器服务原生节点",
};

const bm = {
  value: "BM",
  text: "腾讯黑石物理服务器1.0",
};
const mp = {
  ECM: {
    value: "ECM",
    text: "腾讯边缘计算机",
  },
  LH: {
    value: "LH",
    text: "腾讯轻量应用服务器",
  },
};
// 主机专区分类
export const machineTypes = {
  ALL: all,
  CVM: cvm,
  LH: mp.LH,
  ECM: mp.ECM,
  "EKS-NATIVE": eks,
  BM: bm,
};

export const ProtectLevelVersionMap:any = {
  BASIC_VERSION: "0",
  PRO_VERSION: "1",
  Flagship: "2",
  GENERAL_DISCOUNT: "3",
};

export const AGENT_STATUS_MAP_LIST = [
  {
    name: "在线",
    code: "ONLINE",
    theme: "success",
  },
  {
    name: "已离线",
    code: "OFFLINE",
    theme: "warning",
  },
  {
    name: "未安装",
    code: "UNINSTALLED",
    theme: "default",
  },
];

export const DATA_SOURCE_MAP: any = {
  0: "bash日志",
  1: "实时监控",
  2: "bash内存扫描",
};

const AGENT_STATUS_MAP: any = {};
AGENT_STATUS_MAP_LIST.forEach(item => {
  AGENT_STATUS_MAP[item.code] = item;
});

export { AGENT_STATUS_MAP };

export const AI_AGENT_MAP = {
  openclaw: "OpenClaw",
  other: "疑似Agent",
};

export const IDENTITY_MODE_MAP:any = {
  FINGER: "OpenClaw指纹",
  NETWORK: "网络请求",
  ASSET: "OpenClaw指纹",
};

export const TAG_TEXT_MAP = {
  CORE: "核心资产",
  NEW: "新",
};

export const META_DATA_MAP:any = {
  AK_TMP: "包含cam角色",
  USERDATA: "userdata中包含敏感信息",
  USER_DATA: "userdata中包含敏感信息",
};

export const META_DATA_DESC_MAP:any = {
  AK_TMP: "建议关注绑定的cam角色权限，避免密钥泄露风险",
  USERDATA: "建议排查userdata是否存在敏感信息，避免信息泄露",
  USER_DATA: "建议排查userdata是否存在敏感信息，避免信息泄露",
};

export const POLICY_TYPES:any = {
  0: "系统策略",
  1: "用户自定义",
};

export const BASH_LEVEL_DATA = [
  { text: "高危", value: "1" },
  { text: "中危", value: "2" },
  { text: "低危", value: "3" },
];

export const BASH_LEVEL_ALL = BASH_LEVEL_DATA.map(item => item.value);
export const BASH_LEVEL_MAP = BASH_LEVEL_DATA.reduce((pre: any, cur) => {
  pre[cur.value] = cur.text;
  return pre;
}, {});

export const BASH_STATUS_DATA = [
  {
    text: "待处理",
    value: "0",
    icon: "un_do",
    theme: "error",
  },
  {
    text: "已加白",
    value: "2",
    icon: "success",
    theme: "success",
  },
  {
    text: "已处理",
    value: "1",
    icon: "success",
    theme: "success",
  },
  {
    text: "已忽略",
    value: "3",
    icon: "dismiss",
    theme: "info",
  },
  {
    text: "已拦截",
    value: "5",
    icon: "success",
    theme: "primary",
  },
];

export const MALICIOUS_STATUS_DATA = [
  {
    text: "待处理",
    value: "0",
    icon: "un_do",
    theme: "error",
  },
  {
    text: "已加白",
    value: "2",
    icon: "success",
    theme: "success",
  },
  {
    text: "已处理",
    value: "4",
    icon: "success",
    theme: "success",
  },
  {
    text: "已忽略",
    value: "5",
    icon: "dismiss",
    theme: "info",
  },
  {
    text: "已拦截",
    value: "6",
    icon: "success",
    theme: "primary",
  },
];

export const BASH_STATUS_ALL = BASH_STATUS_DATA.map(item => item.value);

export const statusObjMapNew = BASH_STATUS_DATA.reduce((pre: any, cur) => {
  pre[cur.value] = {
    text: cur.text || "未知",
    icon: cur.icon || "help",
    theme: cur.theme || "info",
  };
  return pre;
}, {});

export const MALICIOUS_STATUS_VAL_MAP = MALICIOUS_STATUS_DATA.reduce(
  (pre: any, cur) => {
    pre[cur.value] = {
      text: cur.text || "未知",
      icon: cur.icon || "help",
      theme: cur.theme || "info",
    };
    return pre;
  },
  {}
);

export const batchTitleMap:any = {
  mark: "标记为已处理",
  ignore: "忽略",
  del: "删除记录",
};

export const SIDEBAR_TEXT = [
  {
    text: "网络管控",
    value: "net",
  },
  {
    text: "OpenClaw 管控",
    value: "host",
  },
  // {
  //   text: "身份管控",
  //   value: "user",
  // },
];

export const PROTECTTYPE_VERSION_TYPES:any = {
  BASIC_VERSION: 0,
  PRO_VERSION: 1,
  Flagship: 2,
  GENERAL_DISCOUNT: 3,
};

export const ProtectLevelMap:any = {
  0: "基础版",
  1: "专业版",
  2: "旗舰版",
  3: "轻量版(Lighthouse)",
};

// 行为审计 - 请求类型
export const ACTION_LOG_REQUEST_TYPE_MAP = {
  network: "网络请求",
  command: "命令执行",
  file: "文件操作",
};

export const ACTION_LOG_REQUEST_TYPE_THEME = {
  network: "primary",
  command: "warning",
  file: "success",
};

export const ACTION_LOG_REQUEST_TYPE_OPTIONS = [
  { text: "网络请求", value: "network" },
  { text: "命令执行", value: "command" },
  { text: "文件操作", value: "file" },
];

// 行为审计 - 请求结果/命中情况
export const ACTION_LOG_RESULT_STATUS_OPTIONS = [
  { text: "命中安全组", value: "hit_security_group", theme: "primary" },
  { text: "已放行", value: "allowed", theme: "success" },
  { text: "已拦截", value: "blocked", theme: "error" },
];

// Skills检测 - 风险类型
export const RISK_TYPE_SKILL = "MALWARE";

// Skills检测 - 威胁等级
export const SKILL_LEVEL_DATA = [
  { text: "严重", value: "4" },
  { text: "高危", value: "3" },
  { text: "中危", value: "2" },
  { text: "低危", value: "1" },
  { text: "提示", value: "0" },
];

export const SKILL_LEVEL_MAP = SKILL_LEVEL_DATA.reduce(
  (pre, cur) => {
    pre[cur.value] = cur.text;
    return pre;
  },
  {} as Record<string, string>
);

export const SKILL_LEVEL_THEME_MAP: Record<
  string,
  { background: string; color: string; dot?: string }
> = {
  4: { background: "#FEF2F2", color: "#DC2626", dot: "#DC2626" },
  3: { background: "#FFF7ED", color: "#EA580C", dot: "#EA580C" },
  2: { background: "#FFFBEB", color: "#D97706", dot: "#D97706" },
  1: { background: "#FEF9C3", color: "#A16207", dot: "#A16207" },
  0: { background: "#EFF6FF", color: "#2563EB", dot: "#2563EB" },
};

// Skills检测 - 处理状态
export const SKILL_STATUS_DATA = [
  {
    text: "待处理",
    value: "4",
    icon: "un_do",
    theme: "error",
  },
  {
    text: "已处置",
    value: "14",
    icon: "newIcon",
    theme: "success",
  },
  {
    text: "已隔离",
    value: "6",
    icon: "newIcon",
    theme: "success",
  },
  {
    text: "已清理",
    value: "8",
    icon: "newIcon",
    theme: "success",
  },
  {
    text: "已忽略",
    value: "5",
    icon: "ignore",
    theme: "success",
  },
  {
    text: "正常",
    value: "9",
    icon: "success",
    theme: "success",
  },
  {
    text: "隔离中",
    value: "10",
    icon: "loading",
    theme: "warning",
  },
  {
    text: "恢复中",
    value: "11",
    icon: "loading",
    theme: "warning",
  },
  // {
  //   text: '已加白',
  //   value: '13',
  //   icon: 'success',
  //   theme: 'success',
  // },
];

export const SKILL_STATUS_VAL_MAP = SKILL_STATUS_DATA.reduce(
  (pre, cur) => {
    pre[cur.value] = {
      text: cur.text || "未知",
      icon: cur.icon || "help",
      theme: cur.theme || "info",
    };
    return pre;
  },
  {} as Record<string, { text: string; icon: string; theme: string }>
);

// Skills检测 - 批量操作标题
export const SKILL_BATCH_TITLE_MAP = {
  mark: "标记已处理",
  markHandle: "标记处置",
  markIgnore: "标记忽略",
  del: "删除记录",
  separate: "隔离文件",
  recover: "恢复隔离",
};

/**
 * 网络请求 dns_log
 * 命令执行 process_snapshot
 * 文件操作 file_log
 */
export enum AGENT_LOG_TYPE {
  DNS_LOG = "dns_log",
  PROCESS_SNAPSHOT = "process_snapshot",
  FILE_LOG = "file_log",
  Net_LOG = "net_log",
}

export const AGENT_LOG_TYPE_MAP = {
  [AGENT_LOG_TYPE.DNS_LOG]: {
    text: "域名解析",
    theme: "primary",
    value: AGENT_LOG_TYPE.DNS_LOG,
  },
  [AGENT_LOG_TYPE.Net_LOG]: {
    text: "IP连接",
    theme: "primary",
    value: AGENT_LOG_TYPE.Net_LOG,
  },
  [AGENT_LOG_TYPE.PROCESS_SNAPSHOT]: {
    text: "命令执行",
    theme: "warning",
    value: AGENT_LOG_TYPE.PROCESS_SNAPSHOT,
  },
  [AGENT_LOG_TYPE.FILE_LOG]: {
    text: "文件写操作",
    theme: "default",
    value: AGENT_LOG_TYPE.FILE_LOG,
  },
};

export const MIX_MACHINE_TYPES = [
  {
    className: "tencent",
    noDataClass: "tencent-nodata",
    name: "腾讯云",
    value: "0",
  },
  {
    className: "ali",
    noDataClass: "ali-nodata",
    name: "阿里云",
    value: "2",
    optionName: "阿里云服务器",
  },
  {
    className: "huawei",
    noDataClass: "huawei-nodata",
    name: "华为云",
    value: "3",
    optionName: "华为云服务器",
  },
  {
    className: "aws",
    noDataClass: "aws-nodata",
    name: "Amazon",
    value: "4",
    optionName: "Amazon服务器",
  },
  {
    className: "ms",
    noDataClass: "ms-nodata",
    name: "Microsoft",
    value: "5",
    optionName: "Microsoft服务器",
  },
  {
    className: "google",
    noDataClass: "google-nodata",
    name: "谷歌GCP",
    value: "6",
    optionName: "Google服务器",
  },
  {
    className: "baidu",
    noDataClass: "baidu-nodata",
    name: "百度云",
    value: "9",
    optionName: "百度云服务器",
  },
  {
    className: "volcano",
    noDataClass: "volcano-nodata",
    name: "火山云",
    value: "10",
    optionName: "火山云服务器",
  },
  {
    className: "idc",
    noDataClass: "idc-nodata",
    name: "其他云或IDC",
    value: "1",
    optionName: "其他云或IDC服务器",
  },
  {
    className: "do",
    noDataClass: "do-nodata",
    name: "DigitalOcean",
    value: "8",
    optionName: "DigitalOcean服务器",
  },
  {
    className: "oracle",
    noDataClass: "oracle-nodata",
    name: "Oracle Cloud",
    value: "7",
    optionName: "OracleCloud服务器",
  },
];

export const VUL_DEFEND_HOST_TYPES = [
  {
    text: "全部",
    value: "ProtectedMachines",
  },
  {
    text: "旗舰版",
    value: "Flagship",
  },
  {
    text: "专业版",
    value: "PRO_VERSION",
  },
];

export const VUL_DEFEND_HOST_TYPES_PRO = [
  {
    text: "全部",
    value: "PRO_VERSION",
  },
  {
    text: "专业版",
    value: "PRO_VERSION",
  },
];
export const VUL_DEFEND_HOST_TYPES_FLAGSHIP = [
  {
    text: "全部",
    value: "Flagship",
  },
  {
    text: "旗舰版",
    value: "Flagship",
  },
];

export const VUL_DEFEND_HOST_TYPES_ANTIEXTORT_PAYED = [
  {
    text: "全部",
    value: "ProtectedMachines",
  },
  {
    text: "旗舰版",
    value: "Flagship",
  },
  {
    text: "专业版",
    value: "PRO_VERSION",
  },
];

export const VUL_DEFEND_HOST_TYPES_ANTIEXTORT = [
  {
    text: "全部",
    value: "",
  },
  {
    text: "旗舰版",
    value: "Flagship",
  },
  {
    text: "专业版",
    value: "PRO_VERSION",
  },
  {
    text: "基础版",
    value: "BASIC_VERSION",
  },
];

export const NETWORK_OPTIONS = [
  {
    text: "vpc网络",
    value: "1",
  },
  { text: "基础网络", value: "2" },
  { text: "非腾讯云网络", value: "3" },
];
export const NETWORK_OPTIONS_MAP = NETWORK_OPTIONS.reduce((pre: any, cur) => {
  pre[cur.value] = cur.text;
  return pre;
}, {});

export const EXPOSED_TYPE_MAP:any = {
  EXPOSED: "暴露",
  UNEXPOSED: "未暴露",
  UNKNOWN: "未知",
};

export const renderLinkMap = (id: any, rid = undefined) => ({
  CVM: `${CONSOLE_URL}/cvm/instance/detail?rid=${rid}&id=${id}`,
  LH: `${CONSOLE_URL}/lighthouse/instance/detail?rid=${rid}&id=${id}`,
  ECM: `${CONSOLE_URL}/ecm/instance/detail/`,
}) as any;

export const EGRESS_RULE = [
  {
    CidrBlock: "10.0.0.0/8",
    Protocol: "ALL",
    Action: "DROP",
    Port: "ALL",
    PolicyDescription: "云安全中心AI Agent内网管控安全组阻止内网IP",
    PolicyIndex: 0,
  },
  {
    CidrBlock: "172.16.0.0/12",
    Protocol: "ALL",
    Action: "DROP",
    Port: "ALL",
    PolicyDescription: "云安全中心AI Agent内网管控安全组阻止内网IP",
    PolicyIndex: 0,
  },
  {
    CidrBlock: "192.168.0.0/16",
    Protocol: "ALL",
    Action: "DROP",
    Port: "ALL",
    PolicyDescription: "云安全中心AI Agent内网管控安全组阻止内网IP",
    PolicyIndex: 0,
  },
];

export const EGRESS_LH_RULE = [
  {
    Protocol: "TCP",
    Port: "ALL",
    Action: "DROP",
    FirewallRuleDescription: "阻止内网IP",
    CidrBlock: "10.0.0.0/8",
  },
  {
    CidrBlock: "172.16.0.0/12",
    Protocol: "TCP",
    Port: "ALL",
    Action: "DROP",
    FirewallRuleDescription: "阻止内网IP",
  },
  {
    CidrBlock: "192.168.0.0/16",
    Protocol: "TCP",
    Port: "ALL",
    Action: "DROP",
    FirewallRuleDescription: "阻止内网IP",
  },
];

export const checkIfHasEgress = (rules: any) =>
  Array.isArray(rules) && rules?.length >= 3
    ? rules.some(
        d =>
          d?.CidrBlock === "10.0.0.0/8" &&
          d.Action === "DROP" &&
          d.Protocol === "TCP" &&
          d.Port === "ALL"
      ) &&
      rules.some(
        d =>
          d?.CidrBlock === "172.16.0.0/12" &&
          d.Action === "DROP" &&
          d.Protocol === "TCP" &&
          d.Port === "ALL"
      ) &&
      rules.some(
        d =>
          d?.CidrBlock === "192.168.0.0/16" &&
          d.Action === "DROP" &&
          d.Protocol === "TCP" &&
          d.Port === "ALL"
      )
    : false;

export const checkIfHasCVMEgress = (rules: any) =>
  Array.isArray(rules) && rules?.length >= 3
    ? rules.some(
        d =>
          d?.CidrBlock === "10.0.0.0/8" &&
          d.Action === "DROP" &&
          d.Protocol === "ALL" &&
          d.Port === "ALL"
      ) &&
      rules.some(
        d =>
          d?.CidrBlock === "172.16.0.0/12" &&
          d.Action === "DROP" &&
          d.Protocol === "ALL" &&
          d.Port === "ALL"
      ) &&
      rules.some(
        d =>
          d?.CidrBlock === "192.168.0.0/16" &&
          d.Action === "DROP" &&
          d.Protocol === "ALL" &&
          d.Port === "ALL"
      )
    : false;

export const MODEL_ICON_MAP:any = {
  claude:
    "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-claude.svg",
  deepseek:
    "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-deepseek.svg",
  qwen: "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-qwen.svg",
  hunyuan:
    "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-hunyuan.svg",
  gemini:
    "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-gemini.svg",
  gpt: "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-gpt.svg",
  kimi: "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-kimi.svg",
  minimax:
    "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-minimax.svg",
  ["混元"]:
    "https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-model-hunyuan.svg",
};
