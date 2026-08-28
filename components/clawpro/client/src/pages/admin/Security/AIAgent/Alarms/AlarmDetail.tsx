import React, { useState, useEffect, useCallback } from 'react';
import moment from '@/vendor/moment';
import { Copy, Info, X } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  PanelTitle,
  SectionTitle,
  BodyText,
  MetaText,
  MetaMedium,
  CodeText,
} from '@/components/ui/Typography';
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerClose,
  DrawerBody,
} from '@/components/ui/drawer';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { DescribeBashEventsInfoNew, DescribeMachines, DescribeRiskDnsEventInfo } from '@/pages/admin/Security/api';

import {
  BASH_ALARM,
  POLICY_TYPES,
  statusObjMapNew,
  MALICIOUS_STATUS_VAL_MAP,
  DATA_SOURCE_MAP,
  FORMAT_NOW,
} from '../constants';
import { parseJsonStr, parseBase64Str } from '../Common/CommonRiskHandleFunc';

import MaliciousOperate from './MaliciousOperate';
import BashOperate from './BashOperate';
import { getRuleLevelText } from './AlarmsList';

/* ---------- tiny copy helper ---------- */
const CopyBtn = ({ text }: { text?: string }) => {
  if (!text) return null;
  return (
    <Copy
      className="inline-block w-3 h-3 ml-1 text-[#A3A3A3] hover:text-[#1447E6] cursor-pointer align-middle"
      onClick={(e: React.MouseEvent) => {
        e.stopPropagation();
        navigator.clipboard.writeText(text);
      }}
    />
  );
};

/** 详情信息行：左 label / 右 value，统一字号、间距、列宽 */
const InfoRow = ({
  label,
  children,
  align = 'center',
}: {
  label: React.ReactNode;
  children: React.ReactNode;
  align?: 'start' | 'center';
}) => (
  <div className={`flex ${align === 'start' ? 'items-start' : 'items-center'} px-3 py-2.5 gap-3`}>
    <MetaText className="w-[112px] flex-shrink-0">{label}</MetaText>
    <div className="flex-1 min-w-0 text-sm text-[#0A0A0A] leading-[1.5]">{children}</div>
  </div>
);

export const getTitle = (item: any) => {
  const arr = item?.exe?.split?.('/');
  return `${arr?.[arr?.length - 1]}(${item?.pid})`;
};

export const getUserInfo = (account: string = '') => {
  const info = account?.split?.(':');
  return {
    user: info?.[0],
    group: info?.[1],
  };
};

export const renderBashDetailTags = (record: { Tags: any[] }) =>
  record?.Tags?.length ? (
    <div className="flex flex-wrap gap-1">
      {record?.Tags?.slice(0, 2)?.map((tag: any, i: number) => (
        <Badge key={i} variant="outline" className="max-w-[250px] truncate">
          {tag}
        </Badge>
      ))}
    </div>
  ) : (
    '--'
  );

export const replaceStrLink = (str: any) => {
  if (typeof str === 'string') {
    let temp = str;
    temp
      .match(/(?:http(s)?:\/\/)?[\w.-]+(?:\.[\w.-]+)+[\w\-._~:/?#[\]@!$&'*+,;=.]+/g)
      ?.forEach?.(
        d =>
          (temp = temp.replace(
            d,
            `<a
            href=${d}
            target="_blank"
            rel="noopener noreferrer"
            style="text-decoration:underline"
          >${d}</a>`,
          )),
      );
    return temp;
  }
  return str;
};

/* ---------- helpers for tree rendering ---------- */

/** A single row inside a process tree node: two‑column layout */
const TreeRow = ({
  left,
  leftCopy,
  right,
  rightCopy,
}: {
  left: string;
  leftCopy?: string;
  right: string;
  rightCopy?: string;
}) => (
  <div className="grid grid-cols-2 gap-2 pl-2.5 pt-2.5">
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="text-[#737373] truncate">{left}</div>
      </TooltipTrigger>
      <TooltipContent side="top" align="start" className="max-h-[500px] overflow-y-auto">
        <div>
          {left}
          <CopyBtn text={leftCopy} />
        </div>
      </TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="text-[#737373] truncate">{right}</div>
      </TooltipTrigger>
      <TooltipContent side="top" align="start" className="max-h-[500px] overflow-y-auto">
        <div className="whitespace-pre-wrap break-words">
          {right}
          <CopyBtn text={rightCopy} />
        </div>
      </TooltipContent>
    </Tooltip>
  </div>
);

/** Build child detail rows for a single process tree node */
const buildTreeChildren = (psTree: any[], index: number) => {
  const rows: React.ReactNode[] = [];
  const info = getUserInfo(psTree?.[index]?.account);

  if (info?.user || psTree?.[index]?.exe) {
    rows.push(
      <TreeRow
        key={`${index}-1`}
        left={`进程所属用户：${info?.user || '-'}`}
        leftCopy={info?.user}
        right={`进程文件路径：${psTree?.[index]?.exe || '-'}`}
        rightCopy={psTree?.[index]?.exe}
      />,
    );
  }

  if (info?.group || psTree?.[index]?.cmdline) {
    rows.push(
      <TreeRow
        key={`${index}-2`}
        left={`进程所属用户组：${info?.group || '-'}`}
        leftCopy={info?.group}
        right={`进程命令行：${psTree?.[index]?.cmdline || '-'}`}
        rightCopy={psTree?.[index]?.cmdline}
      />,
    );
  }

  if (psTree?.[index]?.ssh_service || psTree?.[index]?.ssh_source) {
    rows.push(
      <TreeRow
        key={`${index}-3`}
        left={`SSH服务：${psTree?.[index]?.ssh_service || '-'}`}
        leftCopy={psTree?.[index]?.ssh_service}
        right={`登录源：${psTree?.[index]?.ssh_source || '-'}`}
        rightCopy={psTree?.[index]?.ssh_source}
      />,
    );
  }

  if (psTree?.[index]?.start_time) {
    const timeStr = moment(psTree[index].start_time * 1000).format(FORMAT_NOW);
    rows.push(
      <Tooltip key={`${index}-4`}>
        <TooltipTrigger asChild>
          <div className="aialrm-tree-item pl-2.5 pt-2.5 text-[#737373] truncate">
            {`进程启动时间：${timeStr}`}
          </div>
        </TooltipTrigger>
        <TooltipContent side="top" align="start" className="max-h-[500px] overflow-y-auto">
          <div>
            {`进程启动时间：${timeStr}`}
            <CopyBtn text={timeStr} />
          </div>
        </TooltipContent>
      </Tooltip>,
    );
  }

  return rows;
};

/** A single tree node (process) with collapsible children */
const TreeNode = ({
  label,
  copyText,
  children,
  level,
  danger,
}: {
  label: string;
  copyText?: string;
  children: React.ReactNode;
  level: number;
  danger?: boolean;
}) => (
  <div
    className={`aialarm-detail-tree${level}`}
    style={{ paddingLeft: level === 0 ? 0 : level === 1 ? 18 : 36 }}
  >
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="flex items-center gap-1">
          {danger && <span className="w-2 h-2 rounded-full bg-red-500 flex-shrink-0" />}
          <strong style={danger ? { color: '#B42C3F' } : undefined}>
            {danger ? '风险进程：' : '进程：'}
            {label}
          </strong>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-h-[500px] overflow-y-auto">
        <div>
          {label}
          <CopyBtn text={copyText} />
        </div>
      </TooltipContent>
    </Tooltip>
    {children}
  </div>
);

export default function AlarmDetail({
  visible,
  onClose,
  selectedAlarmType,
  item,
  aiAgentHostList,
  refreshTable,
  clearSelected,
  hasFlagship,
}: any) {
  const TAB_KEYS = ['summary', 'detail', 'scope', 'suggestion'];

  const [activeTab, setActiveTab] = useState('summary');
  const [record, setRecord] = useState({} as any);
  const [hasPsTree, setHasPsTree] = useState(false);
  const [psTree, setPsTree] = useState(null as any);
  const [machineInfo, setMachineInfo] = useState({} as any);

  const agentItemData = aiAgentHostList?.find?.(
    (d: { InstanceID: any }) => d?.InstanceID === item?.MachineExtraInfo?.InstanceID,
  );

  const scrollToSection = (key: string) => {
    setActiveTab(key);
    document.getElementById(`csip-AIAgent-detail-${key}`)?.scrollIntoView?.({ behavior: 'smooth' });
  };

  const handleScroll = useCallback(() => {
    const body = document.querySelector('.csip-AIAgent-alarmDetail-body');
    if (!body) return;
    const bodyRect = body.getBoundingClientRect();
    let current = 'summary';
    for (const key of TAB_KEYS) {
      const el = document.getElementById(`csip-AIAgent-detail-${key}`);
      if (el) {
        const elRect = el.getBoundingClientRect();
        if (elRect.top - bodyRect.top <= 10) {
          current = key;
        }
      }
    }
    setActiveTab(current);
  }, []);

  const getDetail = async (item: {
    Id: any;
    MachineExtraInfo: { InstanceID: any };
    InstanceID: any;
    InstanceId: any;
  }) => {
    const res: any = await Promise.all([
      selectedAlarmType === BASH_ALARM
        ? DescribeBashEventsInfoNew({ Id: item.Id })
        : DescribeRiskDnsEventInfo({ Id: item.Id }),
      DescribeMachines({
        Offset: 0,
        Limit: 1,
        MachineRegion: 'all-regions',
        MachineType: 'ALL',
        Filters: [
          {
            Name: 'InstanceIds',
            Values: [item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId],
          },
        ],
      }),
    ]);
    const data = res?.[0]?.[selectedAlarmType === BASH_ALARM ? 'BashEventsInfo' : 'Info'] || {};
    const info = {
      ...data,
      SuggestScheme: data?.SuggestScheme ? replaceStrLink(data?.SuggestScheme) : '暂无',
      SuggestSolution: data?.SuggestSolution ? replaceStrLink(data?.SuggestSolution) : '暂无',
    };
    const parsedPsTree =
      selectedAlarmType === BASH_ALARM ? parseJsonStr((parseBase64Str(info?.PsTree || '') || null) as any) : null;
    setRecord(info);
    setPsTree(parsedPsTree);
    setHasPsTree(
      selectedAlarmType === BASH_ALARM &&
        info?.PsTree?.length &&
        Array.isArray(parsedPsTree) &&
        parsedPsTree?.length > 0,
    );
    setMachineInfo(res?.[1]?.Machines?.[0] || {});
  };

  useEffect(() => {
    if (!visible) return undefined;
    let bindBody: Element | null = null;
    const timer = setTimeout(() => {
      const body = document.querySelector('.csip-AIAgent-alarmDetail-body');
      if (!body) return;
      bindBody = body;
      body.addEventListener('scroll', handleScroll);
    }, 100);
    return () => {
      clearTimeout(timer);
      if (bindBody) {
        bindBody.removeEventListener('scroll', handleScroll);
      }
    };
  }, [visible, handleScroll]);

  useEffect(() => {
    if (item.Id && visible) {
      getDetail(item);
    }
  }, [item, visible]);

  return (
    <Drawer
      open={visible}
      onOpenChange={open => {
        if (!open) {
          onClose?.();
        }
      }}
      direction="right"
    >
      <DrawerContent className="data-[vaul-drawer-direction=right]:w-[1000px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
        {/* 隐藏标题（已通过自定义 header 展示），保留可访问性 */}
        <DrawerTitle className="sr-only">
          {`${agentItemData?.OpenClawName || ''}存在${selectedAlarmType === BASH_ALARM ? '高危命令' : '恶意请求'}`}
        </DrawerTitle>
        <DrawerBody className="p-0 flex flex-col h-full overflow-hidden">
        {/* ---- header（吸顶，不参与滚动） ---- */}
        <div className="csip-AIAgent-alarmDetail flex-shrink-0 px-6 py-4 bg-background border-b border-[#E5E5E5]">
          {/* 第一行：图标+标题+状态 左侧，更多操作 右侧 */}
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-start gap-2 min-w-0">
              <img
                src="https://test-1256299843.cos.ap-shanghai.myqcloud.com/FEConsoleImage/csip-AIAgent-detailDrawer-title.png"
                alt="alarm"
                className="w-8 h-8 flex-shrink-0 mt-0.5"
              />
              <div className="min-w-0">
                {/* 标题 + 状态 Badge */}
                <div className="flex items-center gap-2 flex-wrap">
                  <PanelTitle as="span">
                    {`${agentItemData?.OpenClawName||''}存在${selectedAlarmType === BASH_ALARM ? '高危命令' : '恶意请求'}`}
                  </PanelTitle>
                  {(() => {
                    const theme = selectedAlarmType === BASH_ALARM
                      ? statusObjMapNew[item?.Status]?.theme
                      : MALICIOUS_STATUS_VAL_MAP[item?.HandleStatus]?.theme;
                    const text = selectedAlarmType === BASH_ALARM
                      ? statusObjMapNew[item?.Status]?.text
                      : MALICIOUS_STATUS_VAL_MAP[item?.HandleStatus]?.text;
                    const cls = theme === 'danger' ? 'badge-stopped'
                      : theme === 'success' ? 'badge-running'
                        : theme === 'warning' ? 'badge-pending'
                          : 'badge-shutdown';
                    return <span className={cls}>{text}</span>;
                  })()}
                </div>
                {/* 类型 + ID 紧跟标题下方 */}
                <div className="flex items-center gap-3 mt-1">
                  <MetaText>
                    类型：
                    <span className="text-[#0A0A0A]">
                      {selectedAlarmType === BASH_ALARM ? '高危命令' : '恶意请求'}
                    </span>
                  </MetaText>
                  <span className="w-px h-3 bg-[#E5E5E5]" />
                  <MetaText>
                    ID：<CodeText tone="emphasis">{item?.Id}</CodeText>
                  </MetaText>
                </div>
              </div>
            </div>
            {/* 操作按钮 + 关闭按钮 - 右侧 */}
            <div className="flex items-start gap-1 flex-shrink-0">
              {selectedAlarmType === BASH_ALARM ? (
                <BashOperate
                  record={item}
                  refreshTable={refreshTable}
                  clearSelected={clearSelected}
                  hasFlagship={hasFlagship}
                  aiAgentHostList={aiAgentHostList}
                  hasNoDetail
                />
              ) : (
                <MaliciousOperate
                  record={item}
                  refreshTable={refreshTable}
                  clearSelected={clearSelected}
                  hasFlagship={hasFlagship}
                  aiAgentHostList={aiAgentHostList}
                  hasNoDetail
                />
              )}
              <DrawerClose asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950 ml-1"
                  aria-label="关闭"
                >
                  <X className="w-4 h-4" />
                </Button>
              </DrawerClose>
            </div>
          </div>

          {/* tabs nav */}
          <div className="mt-4 inline-flex items-center gap-1 p-1 bg-[#F5F5F5] rounded-[4px]">
            {[
              { id: 'summary', label: '安全摘要' },
              { id: 'detail', label: '告警详情' },
              { id: 'scope', label: '影响范围' },
              { id: 'suggestion', label: '处置建议' },
            ].map(tab => {
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => scrollToSection(tab.id)}
                  className={`px-3 py-1 text-sm rounded-[3px] transition-colors ${
                    isActive
                      ? 'bg-white text-[#0A0A0A] font-medium'
                      : 'text-[#737373] hover:text-[#0A0A0A] font-normal'
                  }`}
                  style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                >
                  {tab.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* ---- scrollable body（独立滚动，触发吸顶联动） ---- */}
        <div className="csip-AIAgent-alarmDetail-body flex-1 min-h-0 overflow-y-auto px-6 py-4">
          <div>
            {/* 安全摘要 */}
            <div className="mb-6">
              <SectionTitle as="h4" className="!text-sm !font-semibold mb-3" id="csip-AIAgent-detail-summary">
                安全摘要
              </SectionTitle>
              <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden">
                {/* 危害描述 */}
                <div className="bg-[#FAFAFA] px-3 py-2.5 leading-relaxed" style={{ whiteSpace: 'pre-wrap' }}>
                  <MetaMedium className="!text-[#D97706]">危害描述：</MetaMedium>
                  <BodyText as="span" className="ml-1">
                    {record?.ThreatDesc || record?.HarmDescribe || '暂无'}
                  </BodyText>
                </div>
                {/* info cards */}
                <div className={`grid ${selectedAlarmType === BASH_ALARM ? 'grid-cols-2' : 'grid-cols-1'} divide-x divide-[#E5E5E5] border-t border-[#E5E5E5]`}>
                  {selectedAlarmType === BASH_ALARM ? (
                    <div className="px-3 py-2.5">
                      <MetaText className="block mb-1.5">威胁等级</MetaText>
                      <div>{getRuleLevelText(item?.RuleLevel)}</div>
                    </div>
                  ) : null}
                  <div className="px-3 py-2.5">
                    <MetaText className="block mb-1.5">关联策略</MetaText>
                    <div className="flex items-center flex-wrap gap-1">
                      <BodyText
                        as="span"
                        title={item?.[selectedAlarmType === BASH_ALARM ? 'RuleName' : 'PolicyName']}
                        className="inline-block max-w-[260px] truncate align-middle"
                      >
                        {item?.[selectedAlarmType === BASH_ALARM ? 'RuleName' : 'PolicyName']}
                      </BodyText>

                      {item?.[selectedAlarmType === BASH_ALARM ? 'RuleCategory' : 'PolicyType'] === 0 ||
                      item?.[selectedAlarmType === BASH_ALARM ? 'RuleCategory' : 'PolicyType'] === 1 ? (
                        <span className="inline-flex items-center gap-1 align-middle">
                          <span className="badge-shutdown">
                            {POLICY_TYPES[
                              item?.[selectedAlarmType === BASH_ALARM ? 'RuleCategory' : 'PolicyType']
                            ] || '--'}
                          </span>
                          {String(
                            item?.[selectedAlarmType === BASH_ALARM ? 'RuleCategory' : 'PolicyType'],
                          ) === '0' && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Info className="w-3.5 h-3.5 text-[#A3A3A3] hover:text-[#525252] cursor-pointer" />
                              </TooltipTrigger>
                              <TooltipContent className="max-w-xs">
                                系统策略为腾讯OpenClaw运营专家与算法专家经过多模型沉淀的规则配置，适用于大部分的高危命令检测。
                              </TooltipContent>
                            </Tooltip>
                          )}
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* 告警详情 */}
            <div className="mb-6">
              <SectionTitle
                as="h4"
                className="!text-sm !font-semibold mb-3"
                id="csip-AIAgent-detail-detail"
                style={{ marginTop: 8 }}
              >
                告警详情
              </SectionTitle>
              <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden divide-y divide-[#E5E5E5]">
                <InfoRow label="首次请求时间">
                  {record?.[selectedAlarmType === BASH_ALARM ? 'CreateTime' : 'FirstTime'] || '-'}
                </InfoRow>
                <InfoRow label="最近请求时间">
                  {record?.[selectedAlarmType === BASH_ALARM ? 'ModifyTime' : 'LastTime'] || '-'}
                </InfoRow>
                {selectedAlarmType === BASH_ALARM ? (
                  <InfoRow label="数据来源">{DATA_SOURCE_MAP?.[record.DetectBy] ?? '未知'}</InfoRow>
                ) : (
                  <InfoRow label="恶意请求域名">
                    {record?.Domain || '--'}
                    <CopyBtn text={record?.Domain} />
                  </InfoRow>
                )}

                <InfoRow label="标签特征" align="start">
                  {selectedAlarmType === BASH_ALARM ? (
                    renderBashDetailTags(record)
                  ) : record.Tags?.length ? (
                    <div className="flex flex-wrap gap-1">
                      {record.Tags.map((tag: any, i: number) => (
                        <span key={i} className="badge-shutdown max-w-[250px] truncate">
                          {tag}
                        </span>
                      ))}
                    </div>
                  ) : (
                    '--'
                  )}
                </InfoRow>
                {selectedAlarmType === BASH_ALARM ? (
                  <>
                    <InfoRow label="登录用户">{record?.User || '-'}</InfoRow>
                    <InfoRow label="命令内容" align="start">
                      <div style={{ whiteSpace: 'pre-wrap', wordWrap: 'break-word' }}>
                        {record?.BashCmd || '-'}
                        <CopyBtn text={record?.BashCmd} />
                      </div>
                    </InfoRow>
                    <InfoRow label="PID">{record?.Pid || '-'}</InfoRow>
                    <InfoRow label="行为特征">{record?.Tags?.[2] || '--'}</InfoRow>
                  </>
                ) : (
                  <>
                    <InfoRow label="进程">
                      {record?.ProcessName || '--'}
                      <CopyBtn text={record?.ProcessName} />
                    </InfoRow>
                    <InfoRow label="命令行">
                      {record?.CmdLine || '--'}
                      <CopyBtn text={record?.CmdLine} />
                    </InfoRow>
                    <InfoRow label="MD5">
                      {record?.ProcessMd5
                        ? !String(record?.ProcessMd5)?.replace?.(/0/g, '')
                          ? '--'
                          : <CodeText tone="emphasis">{record?.ProcessMd5}</CodeText>
                        : '--'}
                    </InfoRow>
                    <InfoRow label="请求次数">{record?.AccessCount || '--'}</InfoRow>
                    <InfoRow label="PID">{record?.Pid || '--'}</InfoRow>
                  </>
                )}
              </div>
            </div>

            {/* 影响范围 */}
            <div className="mb-6">
              <SectionTitle as="h4" className="!text-sm !font-semibold mb-3" id="csip-AIAgent-detail-scope">
                影响范围
              </SectionTitle>
              <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden">
                <div className="divide-y divide-[#E5E5E5]">
                  <InfoRow label="AI Agent">
                    {agentItemData?.OpenClawName || '-'}
                  </InfoRow>
                  <InfoRow label={<span>关联告警（高危命令/<br />恶意请求）</span>} align="start">
                    <div className="inline-flex items-center gap-2">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className={agentItemData?.BashCount ? 'badge-stopped' : 'badge-shutdown'}>
                            {agentItemData?.BashCount || 0}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>高危命令关联告警</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className={agentItemData?.MaliciousCount ? 'badge-stopped' : 'badge-shutdown'}>
                            {agentItemData?.MaliciousCount || 0}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>恶意请求关联告警</TooltipContent>
                      </Tooltip>
                    </div>
                  </InfoRow>
                  <InfoRow label="恶意Skills">
                    <span className={agentItemData?.SkillsCount ? 'badge-stopped' : 'badge-shutdown'}>
                      {agentItemData?.SkillsCount || 0}
                    </span>
                  </InfoRow>
                </div>
              </div>
            </div>

            {/* 处置建议 */}
            <div className="mb-6">
              <SectionTitle
                as="h4"
                className="!text-sm !font-semibold mb-3"
                id="csip-AIAgent-detail-suggestion"
              >
                处置建议
              </SectionTitle>
              <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden divide-y divide-[#E5E5E5]">
                <InfoRow label="建议方案" align="start">
                  <div style={{ whiteSpace: 'pre-wrap', wordWrap: 'break-word' }}>
                    <span
                      dangerouslySetInnerHTML={{
                        __html:
                          record?.[
                            selectedAlarmType === BASH_ALARM ? 'SuggestScheme' : 'SuggestSolution'
                          ] || '',
                      }}
                    />
                  </div>
                </InfoRow>
                <InfoRow label="参考链接" align="start">
                  <div style={{ wordBreak: 'break-all' }}>
                    {selectedAlarmType === BASH_ALARM
                      ? record?.References?.length
                        ? record?.References?.map?.((r: any, i: number) => (
                            <a
                              key={i}
                              href={r}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-[#1447E6] hover:underline block"
                            >
                              {r}
                            </a>
                          ))
                        : '暂无'
                      : record?.ReferenceLink
                        ? record?.ReferenceLink
                        : '暂无'}
                  </div>
                </InfoRow>
              </div>
            </div>

            {/* 进程树 */}
            {!hasPsTree ? null : (
              <div className="mb-6">
                <SectionTitle as="h4" className="!text-sm !font-semibold mb-3">
                  进程树
                </SectionTitle>
                <div className="process-tree cwp-progress-tree space-y-2">
                  {psTree?.[2] && (
                    <TreeNode
                      label={getTitle(psTree[2])}
                      copyText={getTitle(psTree[2])}
                      level={2}
                    >
                      {buildTreeChildren(psTree, 2)}
                    </TreeNode>
                  )}
                  {psTree?.[1] && (
                    <TreeNode
                      label={getTitle(psTree[1])}
                      copyText={getTitle(psTree[1])}
                      level={psTree?.[2] ? 1 : 0}
                    >
                      {buildTreeChildren(psTree, 1)}
                    </TreeNode>
                  )}
                  {psTree?.[0] && (
                    <TreeNode
                      label={getTitle(psTree[0])}
                      copyText={getTitle(psTree[0])}
                      level={!psTree?.[2] ? (psTree?.[1] ? 1 : 0) : 0}
                      danger
                    >
                      {buildTreeChildren(psTree, 0)}
                    </TreeNode>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}
