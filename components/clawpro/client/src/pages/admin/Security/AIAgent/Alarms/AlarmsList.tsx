import React, { useEffect, useState, useRef, useMemo, useCallback } from 'react';
import { Base64 } from '@/vendor/js-base64';
import { RefreshCw, Search, Info, Loader2, ChevronDown, Copy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import { Checkbox } from '@/components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyDescription,
} from '@/components/ui/empty';
import { SurfaceCard } from '@/components/ui/Surface';
import { Pagination } from '@/components/ui/pagination';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableCell,
  TableActionCell,
  TableRow,
} from '@/components/ui/table';

import { DescribeBashEventsNew, DescribeRiskDnsEventList } from '@/pages/admin/Security/api';

import {
  BASH_ALARM,
  MALICIOUS_ALARM,
  BASH_LEVEL_MAP,
  POLICY_TYPES,
  statusObjMapNew,
  BASH_LEVEL_DATA,
  BASH_STATUS_DATA,
  MALICIOUS_STATUS_DATA,
  MALICIOUS_STATUS_VAL_MAP,
  batchTitleMap,
  RISK_TYPE_BASH,
  RISK_TYPE_MALICIOUS,
} from '../constants';
import { getSelectionRows, modifyEventsStatus, getBatchStatus } from '../Common/CommonRiskHandleFunc';
import { renderAgentItem } from '../Assets/AgentAssetsList';

import MaliciousOperate from './MaliciousOperate';
import BatchOperatorDialog from './BatchOperatorDialog';
import BashOperate from './BashOperate';
import AlarmDetail from './AlarmDetail';

/** 威胁等级渲染（规范 §16 StatusTag mode="text"：无底色彩色文字） */
export const getRuleLevelText = (level: any) => {
  const text = BASH_LEVEL_MAP[level] || '无';
  let color = '#16A34A'; // 低危/无 - 绿
  if (String(level) === '1') color = '#DC2626'; // 高危 - 红
  else if (String(level) === '2') color = '#D97706'; // 中危 - 橙
  else if (String(level) === '3') color = '#16A34A'; // 低危 - 绿
  else color = '#525252'; // 无 - 灰
  return <span className="text-sm font-medium" style={{ color }}>{text}</span>;
};

/** 策略类型渲染 */
export const renderPolicyType = (item: any) =>
(item?.RuleCategory === 0 || item?.RuleCategory === 1 ? (
  <Badge variant="secondary">
    {POLICY_TYPES[item?.RuleCategory] || '--'}
    {String(item?.RuleCategory) === '0' && (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="w-3.5 h-3.5 text-[#A3A3A3] cursor-help" style={{ verticalAlign: 'middle' }} />
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          系统策略为腾讯OpenClaw运营专家与算法专家经过多模型沉淀的规则配置，适用于大部分的高危命令检测。
        </TooltipContent>
      </Tooltip>
    )}
  </Badge>
) : (
  '--'
));

/** 文本溢出省略 + tooltip + 可复制 */
const OverflowText = ({
  children,
  tooltip,
  className = '',
  onClick,
  copyable,
  style,
}: {
  children: React.ReactNode;
  tooltip?: React.ReactNode;
  className?: string;
  onClick?: () => void;
  copyable?: boolean;
  style?: React.CSSProperties;
}) => {
  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (typeof children === 'string') {
      navigator.clipboard.writeText(children);
    }
  };

  const content = (
    <span
      className={`block truncate max-w-full ${onClick ? 'text-[#1447E6] cursor-pointer hover:underline' : 'text-[#525252]'} ${className}`}
      style={style}
      onClick={onClick}
    >
      {children}
      {copyable && (
        <Copy
          className="inline-block w-3 h-3 ml-1 text-[#A3A3A3] hover:text-[#1447E6] cursor-pointer align-middle"
          onClick={handleCopy}
        />
      )}
    </span>
  );

  if (tooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{content}</TooltipTrigger>
        <TooltipContent>{tooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return content;
};

/** 状态主题 → Badge 渲染（绿=success/红=error/橙=warning/灰=info） */
const renderStatusBadge = (theme: string | undefined, text: string) => {
  if (theme === 'error') return <Badge color="red">{text}</Badge>;
  if (theme === 'success') return <Badge color="green">{text}</Badge>;
  if (theme === 'warning') {
    return <Badge className="border-transparent bg-amber-50 text-amber-700">{text}</Badge>;
  }
  return <Badge variant="secondary">{text}</Badge>;
};

/** 排序方向切换 */
type SortDir = 'asc' | 'desc' | '';
const nextSort = (cur: SortDir): SortDir => {
  if (cur === '') return 'desc';
  if (cur === 'desc') return 'asc';
  return '';
};

export default function AlarmsList({
  machineVersionCount,
  aiAgentHostList,
  openAssetDetail = undefined,
  InstanceId = undefined,
  alarmTabId = undefined,
  getAllInitAlarmCount = undefined,
  // 仅设计走查/演示用：传入后跳过真实请求，直接展示 mock 数据
  mockBashAlarms = undefined,
  mockMaliciousAlarms = undefined,
}: any) {
  const [showTable, setShowTable] = useState(true);
  const [selectedAlarmType, setSelectedAlarmType] = useState(BASH_ALARM);
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [selectedRows, setSelectedRows] = useState<any[]>([]);
  const [curTableData, setCurTableData] = useState<any[]>([]);
  const [batchType, setBatchType] = useState('');
  const [batchTimer, setBatchTimer] = useState(0);
  const [selectedItem, setSelectedItem] = useState({} as any);
  const [isBatchLoading, setIsBatchLoading] = useState(false);
  const [batchHandleModalVisible, setBatchHandleModalVisible] = useState(false);
  const [alarmDetailDrawerVisible, setAlarmDetailDrawerVisible] = useState(false);
  const [unHandleBashCount, setUnHandleBashCount] = useState(0);
  const [unHandleMaliciousCount, setUnHandleMaliciousCount] = useState(0);

  // 分页 & 搜索 & 排序 & 筛选
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalCount, setTotalCount] = useState(0);
  const [tableData, setTableData] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  // 搜索
  const [searchKey, setSearchKey] = useState('');
  const [searchField, setSearchField] = useState<string>(
    selectedAlarmType === BASH_ALARM ? 'MachineName' : 'MachineName',
  );

  // 排序
  const [sortBy, setSortBy] = useState(selectedAlarmType === BASH_ALARM ? 'CreateTime' : 'LastTime');
  const [sortOrder, setSortOrder] = useState<SortDir>('desc');

  // 筛选
  const [filterStatus, setFilterStatus] = useState<string[]>(['0']);
  const [filterLevel, setFilterLevel] = useState<string[]>([]);

  const requestIdRef = useRef(0);

  const getInitAlarmCount = async () => {
    if (!aiAgentHostList?.length && !InstanceId) {
      return;
    }
    const res: any = await Promise.all([
      DescribeBashEventsNew({
        Offset: 0,
        Limit: 1,
        Filters: [
          { Name: 'Status', Values: ['0'] },
          { Name: 'InstanceID', Values: InstanceId ? [InstanceId] : aiAgentHostList?.map?.((d: { InstanceID: any }) => d.InstanceID) },
        ],
      }),
      DescribeRiskDnsEventList({
        Offset: 0,
        Limit: 1,
        Filters: [
          { Name: 'HandleStatus', Values: ['0'] },
          { Name: 'InstanceID', Values: InstanceId ? [InstanceId] : aiAgentHostList?.map?.((d: { InstanceID: any }) => d.InstanceID) },
        ],
      }),
    ]);
    setUnHandleBashCount(res?.[0]?.TotalCount || 0);
    setUnHandleMaliciousCount(res?.[1]?.TotalCount || 0);
  };

  const clearSelected = useCallback(() => {
    setSelectedRows([]);
    setSelectedKeys([]);
  }, []);

  /** 数据请求 */
  const fetchData = useCallback(async () => {
    // === Mock 分支：直接使用注入的 mock 数据，跳过真实请求 ===
    const isMock = mockBashAlarms !== undefined || mockMaliciousAlarms !== undefined;
    if (isMock) {
      const isBashType = selectedAlarmType === BASH_ALARM;
      let list: any[] = (isBashType ? mockBashAlarms : mockMaliciousAlarms) || [];

      // 状态筛选
      if (filterStatus?.length) {
        const key = isBashType ? 'Status' : 'HandleStatus';
        list = list.filter((d: any) => filterStatus.includes(String(d?.[key])));
      }
      // 等级筛选（仅高危命令）
      if (isBashType && filterLevel?.length) {
        list = list.filter((d: any) => filterLevel.includes(String(d?.RuleLevel)));
      }
      // 搜索
      if (searchKey.trim()) {
        const kw = searchKey.trim();
        list = list.filter((d: any) => {
          if (searchField === 'MachineName') return (d?.MachineName || '').indexOf(kw) >= 0;
          if (searchField === 'InstanceID') return (d?.MachineExtraInfo?.InstanceID || d?.InstanceID || '').indexOf(kw) >= 0;
          if (searchField === 'RuleName') return (d?.RuleName || '').indexOf(kw) >= 0;
          if (searchField === 'Domain') return (d?.Domain || '').indexOf(kw) >= 0;
          return true;
        });
      }
      const enriched = list.map((d: any) => ({
        ...d,
        AgentName: aiAgentHostList?.find?.((a: any) => a?.InstanceID === d?.MachineExtraInfo?.InstanceID)?.AgentName || '',
      }));
      const sliced = enriched.slice((page - 1) * pageSize, page * pageSize);
      setCurTableData(sliced);
      setTableData(sliced);
      setTotalCount(enriched.length);
      setUnHandleBashCount((mockBashAlarms || []).filter((d: any) => String(d?.Status) === '0').length);
      setUnHandleMaliciousCount((mockMaliciousAlarms || []).filter((d: any) => String(d?.HandleStatus) === '0').length);
      setIsLoading(false);
      return;
    }

    if (!aiAgentHostList?.length) {
      setTableData([]);
      setTotalCount(0);
      return;
    }
    setIsLoading(true);
    const reqId = ++requestIdRef.current;

    const isBash = selectedAlarmType === BASH_ALARM;
    const params: any = {
      Offset: (page - 1) * pageSize,
      Limit: pageSize,
      Filters: [],
    };

    // 排序
    if (sortBy && sortOrder) {
      params.Order = sortOrder;
      params.By = sortBy;
    } else {
      params.Order = 'desc';
      params.By = isBash ? 'CreateTime' : 'LastTime';
    }

    // 状态筛选
    if (isBash) {
      const statusValues = filterStatus?.length ? filterStatus : BASH_STATUS_DATA.map(x => x.value);
      params.Filters.push({ Name: 'Status', Values: statusValues });
    } else {
      const statusValues = filterStatus?.length ? filterStatus : [];
      if (statusValues.length) {
        params.Filters.push({ Name: 'HandleStatus', Values: statusValues });
      }
    }

    // 等级筛选 (仅高危命令)
    if (isBash && filterLevel?.length) {
      params.Filters.push({ Name: 'RuleLevel', Values: filterLevel });
    }

    // 搜索
    if (searchKey.trim()) {
      if (searchField === 'MachineName') {
        params.Filters.push({ Name: 'MachineName', Values: [searchKey.trim()] });
      } else if (searchField === 'InstanceID') {
        params.Filters.push({ Name: 'InstanceID', Values: [searchKey.trim()] });
      } else if (searchField === 'RuleName') {
        params.Filters.push({ Name: 'RuleName', Values: [searchKey.trim()] });
      } else if (searchField === 'Domain') {
        params.Filters.push({ Name: 'Domain', Values: [Base64.encode(searchKey.trim())] });
      }
    }

    // InstanceId
    if (InstanceId) {
      params.Filters = params.Filters.filter((d: any) => d?.Name !== 'InstanceID').concat({
        Name: 'InstanceID',
        Values: [InstanceId],
      });
    } else {
      const allIds = aiAgentHostList.map((d: { InstanceID: any }) => d.InstanceID);
      const insId = params.Filters.find((d: any) => d?.Name === 'InstanceID');
      if (!insId) {
        params.Filters.push({ Name: 'InstanceID', Values: allIds });
      } else {
        const exists = insId.Values.filter((d: any) => allIds.includes(d));
        if (!exists.length) {
          setTableData([]);
          setTotalCount(0);
          setIsLoading(false);
          return;
        }
        params.Filters = params.Filters.filter((d: any) => d?.Name !== 'InstanceID').concat({
          Name: 'InstanceID',
          Values: exists,
        });
      }
    }

    try {
      const resp: any = await (isBash
        ? DescribeBashEventsNew(params)
        : DescribeRiskDnsEventList(params));

      if (reqId !== requestIdRef.current) return;

      const list = resp?.List?.map?.((d: any) => ({
        ...d,
        AgentName: aiAgentHostList?.find?.((a: { InstanceID: any }) => a.InstanceID === d?.MachineExtraInfo?.InstanceID)?.AgentName || '',
      })) || [];

      setCurTableData(list);
      setTableData(list);
      setTotalCount(resp?.TotalCount ?? 0);
    } catch (err) {
      console.error(err);
    } finally {
      if (reqId === requestIdRef.current) {
        setIsLoading(false);
      }
    }
  }, [selectedAlarmType, page, pageSize, sortBy, sortOrder, filterStatus, filterLevel, searchKey, searchField, aiAgentHostList, InstanceId, mockBashAlarms, mockMaliciousAlarms]);

  const refreshTable = useCallback(() => {
    fetchData();
    getInitAlarmCount?.();
    getAllInitAlarmCount?.();
  }, [fetchData, getInitAlarmCount, getAllInitAlarmCount]);

  // 加载数据
  useEffect(() => {
    if (showTable) {
      fetchData();
    }
  }, [fetchData, showTable]);

  useEffect(() => {
    getInitAlarmCount();
  }, [InstanceId]);

  useEffect(() => {
    if (alarmTabId === MALICIOUS_ALARM) {
      setSelectedAlarmType(MALICIOUS_ALARM);
    }
  }, [alarmTabId]);

  useEffect(() => {
    getBatchStatus(RISK_TYPE_BASH, setIsBatchLoading, refreshTable, setBatchTimer);
  }, []);

  useEffect(() => {
    return () => {
      window.clearInterval(batchTimer);
    };
  }, [batchTimer]);

  // 切换告警类型
  const handleSwitchAlarmType = (type: string) => {
    if (type === selectedAlarmType) return;
    setSelectedAlarmType(type);
    setPage(1);
    clearSelected();
    setBatchTimer(0);
    setIsBatchLoading(false);
    setFilterStatus(['0']);
    setFilterLevel([]);
    setSearchKey('');
    setSortBy(type === BASH_ALARM ? 'CreateTime' : 'LastTime');
    setSortOrder('desc');
    getBatchStatus(
      type === BASH_ALARM ? RISK_TYPE_BASH : RISK_TYPE_MALICIOUS,
      setIsBatchLoading,
      refreshTable,
      setBatchTimer,
    );
  };

  // 全选 / 取消全选
  const allSelected = tableData.length > 0 && tableData.every((d: any) => selectedKeys.includes(String(d?.Id)));
  const someSelected = tableData.some((d: any) => selectedKeys.includes(String(d?.Id))) && !allSelected;

  const toggleSelectAll = () => {
    if (allSelected) {
      const ids = tableData.map((d: any) => String(d?.Id));
      setSelectedKeys(prev => prev.filter(k => !ids.includes(k)));
      setSelectedRows(prev => prev.filter((r: any) => !ids.includes(String(r?.Id))));
    } else {
      const ids = tableData.map((d: any) => String(d?.Id));
      const newKeys = Array.from(new Set([...selectedKeys, ...ids]));
      setSelectedKeys(newKeys);
      const newRows: any = getSelectionRows(selectedRows, newKeys, curTableData);
      setSelectedRows(newRows);
    }
  };

  const toggleSelectRow = (item: any) => {
    const id = String(item?.Id);
    const isSelected = selectedKeys.includes(id);
    let newKeys: string[];
    if (isSelected) {
      newKeys = selectedKeys.filter(k => k !== id);
    } else {
      newKeys = [...selectedKeys, id];
    }
    setSelectedKeys(newKeys);
    const newRows: any = getSelectionRows(selectedRows, newKeys, curTableData);
    setSelectedRows(newRows);
  };

  /** 搜索字段选项 */
  const searchFields = useMemo(() => {
    const fields = [
      // { value: 'MachineName', label: '资产名称' },
      // ...(InstanceId ? [] : [{ value: 'InstanceID', label: '资产ID' }]),
    ];
    if (selectedAlarmType === BASH_ALARM) {
      fields.push({ value: 'RuleName', label: '命中策略名称' });
    } else {
      fields.push({ value: 'Domain', label: '恶意请求域名' });
    }
    return fields;
  }, [selectedAlarmType, InstanceId]);

  /** 状态筛选选项 */
  const statusOptions = selectedAlarmType === BASH_ALARM ? BASH_STATUS_DATA : MALICIOUS_STATUS_DATA;

  return (
    <div style={{ position: 'relative' }}>
      {isBatchLoading && (
        <div className="manage-batch-loading flex items-center gap-2 p-2 mb-2 bg-[#EFF6FF] border border-[#DBEAFE] rounded-[4px] text-sm text-[#1447E6]">
          <Loader2 className="w-4 h-4 animate-spin" />
          <span>正在进行批量操作中...请稍候...</span>
        </div>
      )}

      {/* 告警类型切换 + 操作栏（主按钮统一右置） */}
      {/* 左右 padding = 0，与下方 SurfaceCard 左右边沿对齐 */}
      <div className="flex items-center gap-2 mb-4 mt-[10px]" style={{ paddingTop: 10 }}>
        {/* 左侧：tab + 筛选 + 刷新 */}
        <SegmentGroup>
          <SegmentOption
            active={selectedAlarmType === BASH_ALARM}
            onClick={() => handleSwitchAlarmType(BASH_ALARM)}
          >
            高危命令（{unHandleBashCount}）
          </SegmentOption>
          <SegmentOption
            active={selectedAlarmType === MALICIOUS_ALARM}
            onClick={() => handleSwitchAlarmType(MALICIOUS_ALARM)}
          >
            恶意请求（{unHandleMaliciousCount}）
          </SegmentOption>
        </SegmentGroup>

        {/* 状态筛选 */}
        <Select
          value={filterStatus.length === 1 ? filterStatus[0] : '__all__'}
          onValueChange={val => {
            if (val === '__all__') {
              setFilterStatus([]);
            } else {
              setFilterStatus([val]);
            }
            setPage(1);
          }}
        >
          <SelectTrigger size="sm" className="w-[140px]">
            <SelectValue placeholder="处理状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部状态</SelectItem>
            {statusOptions.map(d => (
              <SelectItem key={d.value} value={d.value}>{d.text}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        {/* 等级筛选 (仅高危命令) */}
        {selectedAlarmType === BASH_ALARM && (
          <Select
            value={filterLevel.length === 1 ? filterLevel[0] : '__all__'}
            onValueChange={val => {
              if (val === '__all__') {
                setFilterLevel([]);
              } else {
                setFilterLevel([val]);
              }
              setPage(1);
            }}
          >
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="威胁等级" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">全部等级</SelectItem>
              {BASH_LEVEL_DATA.map(d => (
                <SelectItem key={d.value} value={d.value}>{d.text}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}

        <button
          onClick={refreshTable}
          className="w-8 h-8 flex items-center justify-center rounded-[4px] border border-[#E5E5E5] bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
          title="刷新表格"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
        </button>

        {/* 右侧：主按钮组（更多操作 / 标记已处理） */}
        <div className="flex items-center gap-2 ml-auto">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="claw-outline" size="claw-sm" disabled={!selectedKeys?.length}>
                更多操作
                <ChevronDown className="w-3.5 h-3.5 ml-1" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                disabled={!selectedKeys?.length}
                onClick={() => {
                  setBatchType('ignore');
                  setBatchHandleModalVisible(true);
                }}
              >
                忽略
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!selectedKeys?.length}
                onClick={() => {
                  setBatchType('del');
                  setBatchHandleModalVisible(true);
                }}
              >
                删除记录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <Button
            size="sm"
            disabled={!selectedKeys?.length}
            onClick={() => {
              setBatchType('mark');
              setBatchHandleModalVisible(true);
            }}
          >
            标记已处理
          </Button>
        </div>
      </div>

      {/* 表格 / 空状态 */}
      {tableData.length > 0 ? (
      <SurfaceCard className="relative overflow-hidden">
        {isLoading && (
          <div className="absolute inset-0 bg-white/60 z-10 flex items-center justify-center">
            <RefreshCw className="w-6 h-6 animate-spin text-[#1447E6]" />
          </div>
        )}

        <Table scrollX={1100} variant="white" containerClassName="bg-white">
          {selectedAlarmType === BASH_ALARM ? (
            /* 高危命令: 56px + 后 9 列百分比合计 100% */
            <colgroup>
              <col style={{ width: 56 }} />
              <col style={{ width: '12%' }} />
              <col style={{ width: '8%' }} />
              <col style={{ width: '10%' }} />
              <col style={{ width: '14%' }} />
              <col style={{ width: '14%' }} />
              <col style={{ width: '11%' }} />
              <col style={{ width: '11%' }} />
              <col style={{ width: '8%' }} />
              <col style={{ width: '12%' }} />
            </colgroup>
          ) : (
            /* 恶意请求: 56px + 后 8 列百分比合计 100% */
            <colgroup>
              <col style={{ width: 56 }} />
              <col style={{ width: '13%' }} />
              <col style={{ width: '12%' }} />
              <col style={{ width: '15%' }} />
              <col style={{ width: '15%' }} />
              <col style={{ width: '8%' }} />
              <col style={{ width: '14%' }} />
              <col style={{ width: '9%' }} />
              <col style={{ width: '14%' }} />
            </colgroup>
          )}
          <TableHeader>
            <TableRow>
              {/* 勾选列（固定左 1） */}
              <TableHead fixed="left" fixedShadow={false} style={{ width: 56, minWidth: 56 }}>
                <Checkbox
                  checked={allSelected ? true : someSelected ? 'indeterminate' : false}
                  onCheckedChange={toggleSelectAll}
                />
              </TableHead>
              {/* 告警名称（固定左 2，偏移 56px = 复选框列宽） */}
              <TableHead fixed="left" style={{ left: 56, width: 260, minWidth: 260 }}>告警名称</TableHead>
              {/* 威胁等级 (仅高危命令) */}
              {selectedAlarmType === BASH_ALARM && (
                <TableHead style={{ width: 100, minWidth: 100 }}>威胁等级</TableHead>
              )}
              {/* 命中策略 */}
              <TableHead style={{ width: 160, minWidth: 160 }}>命中策略</TableHead>
              {/* AI Agent/调用模型 */}
              <TableHead style={{ width: 200, minWidth: 200 }}>AI Agent/调用模型</TableHead>
              {/* 命令内容 或 恶意请求域名 + 请求次数 */}
              {selectedAlarmType === BASH_ALARM ? (
                <TableHead style={{ width: 220, minWidth: 220 }}>命令内容</TableHead>
              ) : (
                <>
                  <TableHead style={{ width: 200, minWidth: 200 }}>恶意请求域名</TableHead>
                  <TableHead
                    style={{ width: 100, minWidth: 100 }}
                    className="cursor-pointer select-none"
                    onClick={() => {
                      if (sortBy === 'AccessCount') {
                        const ns = nextSort(sortOrder);
                        setSortOrder(ns);
                        if (!ns) setSortBy('');
                      } else {
                        setSortBy('AccessCount');
                        setSortOrder('desc');
                      }
                      setPage(1);
                    }}
                  >
                    请求次数
                    {sortBy === 'AccessCount' && (
                      <span className="ml-1 text-[#1447E6]">{sortOrder === 'desc' ? '↓' : '↑'}</span>
                    )}
                  </TableHead>
                </>
              )}
              {/* 时间列 */}
              {selectedAlarmType === BASH_ALARM ? (
                <>
                  <TableHead
                    style={{ width: 160, minWidth: 160 }}
                    className="cursor-pointer select-none"
                    onClick={() => {
                      if (sortBy === 'CreateTime') {
                        const ns = nextSort(sortOrder);
                        setSortOrder(ns);
                        if (!ns) setSortBy('');
                      } else {
                        setSortBy('CreateTime');
                        setSortOrder('desc');
                      }
                      setPage(1);
                    }}
                  >
                    发生时间
                    {sortBy === 'CreateTime' && (
                      <span className="ml-1 text-[#1447E6]">{sortOrder === 'desc' ? '↓' : '↑'}</span>
                    )}
                  </TableHead>
                  <TableHead
                    style={{ width: 160, minWidth: 160 }}
                    className="cursor-pointer select-none"
                    onClick={() => {
                      if (sortBy === 'ModifyTime') {
                        const ns = nextSort(sortOrder);
                        setSortOrder(ns);
                        if (!ns) setSortBy('');
                      } else {
                        setSortBy('ModifyTime');
                        setSortOrder('desc');
                      }
                      setPage(1);
                    }}
                  >
                    处理时间
                    {sortBy === 'ModifyTime' && (
                      <span className="ml-1 text-[#1447E6]">{sortOrder === 'desc' ? '↓' : '↑'}</span>
                    )}
                  </TableHead>
                </>
              ) : (
                <TableHead
                  style={{ width: 160, minWidth: 160 }}
                  className="cursor-pointer select-none"
                  onClick={() => {
                    if (sortBy === 'LastTime') {
                      const ns = nextSort(sortOrder);
                      setSortOrder(ns);
                      if (!ns) setSortBy('');
                    } else {
                      setSortBy('LastTime');
                      setSortOrder('desc');
                    }
                    setPage(1);
                  }}
                >
                  最近请求时间
                  {sortBy === 'LastTime' && (
                    <span className="ml-1 text-[#1447E6]">{sortOrder === 'desc' ? '↓' : '↑'}</span>
                  )}
                </TableHead>
              )}
              {/* 处理状态 */}
              <TableHead style={{ width: 100, minWidth: 100 }}>处理状态</TableHead>
              {/* 操作 */}
              <TableHead fixed="right" style={{ width: 140, minWidth: 140 }}>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tableData.map((item: any, idx: number) => {
              const rowId = String(item?.Id);
              const isSelected = selectedKeys.includes(rowId);
              const statusMap = selectedAlarmType === BASH_ALARM ? statusObjMapNew : MALICIOUS_STATUS_VAL_MAP;
              const statusKey = selectedAlarmType === BASH_ALARM ? item?.Status : item?.HandleStatus;
              const statusInfo = statusMap[statusKey];

              return (
                <TableRow key={item?.Id || idx}>
                  {/* 勾选（固定左 1） */}
                  <TableCell fixed="left" fixedShadow={false}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => toggleSelectRow(item)}
                    />
                  </TableCell>
                  {/* 告警名称（固定左 2，偏移 56px） */}
                  <TableCell fixed="left" style={{ left: 56 }}>
                    {(() => {
                      const alarmName = `${aiAgentHostList?.find?.((d: any) => d?.InstanceID === item?.MachineExtraInfo?.InstanceID)?.OpenClawName || ''}存在${selectedAlarmType === BASH_ALARM ? '高危命令' : '恶意请求'}`;
                      return (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="link-dark"
                              className="px-0 h-auto text-left justify-start max-w-full"
                              onClick={() => {
                                setSelectedItem(item);
                                setAlarmDetailDrawerVisible(true);
                              }}
                            >
                              <span className="block truncate max-w-full">{alarmName}</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent className="max-w-sm break-all">{alarmName}</TooltipContent>
                        </Tooltip>
                      );
                    })()}
                  </TableCell>
                  {/* 威胁等级 */}
                  {selectedAlarmType === BASH_ALARM && (
                    <TableCell>{getRuleLevelText(item?.RuleLevel)}</TableCell>
                  )}
                  {/* 命中策略 */}
                  <TableCell className="text-[#525252] max-w-0 overflow-hidden text-ellipsis">
                    <span className="block truncate">
                      {(selectedAlarmType === BASH_ALARM ? item?.RuleName : item?.PolicyName) || '--'}
                    </span>
                  </TableCell>
                  {/* AI Agent */}
                  <TableCell className="max-w-0 overflow-hidden text-ellipsis">
                    {renderAgentItem(
                      aiAgentHostList?.find?.((a: { InstanceID: any }) => a.InstanceID === item?.MachineExtraInfo?.InstanceID),
                      InstanceId ? undefined : () => openAssetDetail?.(aiAgentHostList?.find?.((a: { InstanceID: any }) => a.InstanceID === item?.MachineExtraInfo?.InstanceID)),
                    )}
                  </TableCell>
                  {/* 命令内容 / 域名 + 请求次数 */}
                  {selectedAlarmType === BASH_ALARM ? (
                    <TableCell className="text-[#525252] max-w-0 overflow-hidden text-ellipsis">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="block truncate">{item?.BashCmd || '-'}</span>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-sm break-all">{item?.BashCmd}</TooltipContent>
                      </Tooltip>
                    </TableCell>
                  ) : (
                    <>
                      <TableCell className="text-[#525252] max-w-0 overflow-hidden text-ellipsis">
                        {item?.Domain?.length ? (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="block truncate">{item.Domain}</span>
                            </TooltipTrigger>
                            <TooltipContent className="max-w-sm break-all">{item.Url}</TooltipContent>
                          </Tooltip>
                        ) : '--'}
                      </TableCell>
                      <TableCell className="text-[#525252]">{item?.AccessCount ?? '--'}</TableCell>
                    </>
                  )}
                  {/* 时间 */}
                  {selectedAlarmType === BASH_ALARM ? (
                    <>
                      <TableCell className="text-[#737373]">{item?.CreateTime || '-'}</TableCell>
                      <TableCell className="text-[#737373]">{item?.ModifyTime || '-'}</TableCell>
                    </>
                  ) : (
                    <TableCell className="text-[#737373]">{item?.LastTime || '-'}</TableCell>
                  )}
                  {/* 处理状态 */}
                  <TableCell>
                    {renderStatusBadge(statusInfo?.theme, statusInfo?.text || '未知')}
                  </TableCell>
                  {/* 操作 */}
                  <TableActionCell fixed="right" rawChildren>
                    {
                      selectedAlarmType === BASH_ALARM ? (
                        <BashOperate
                          record={item}
                          refreshTable={refreshTable}
                          clearSelected={clearSelected}
                          aiAgentHostList={aiAgentHostList}
                          hasFlagship={machineVersionCount?.UltimateVersionNum > 0}
                          openDetail={() => {
                            setSelectedItem(item);
                            setAlarmDetailDrawerVisible(true);
                          }}
                        />
                      ) : (
                        <MaliciousOperate
                          record={item}
                          refreshTable={refreshTable}
                          clearSelected={clearSelected}
                          aiAgentHostList={aiAgentHostList}
                          hasFlagship={machineVersionCount?.UltimateVersionNum > 0}
                          openDetail={() => {
                            setSelectedItem(item);
                            setAlarmDetailDrawerVisible(true);
                          }}
                        />
                      )
                    }
                  </TableActionCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>

        {/* 表格页脚：数量统计左对齐，分页器右对齐 */}
        <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]">
          <span className="justify-self-start text-sm leading-[1.5] text-[#737373]">
            共 {totalCount} 条记录
          </span>
          <Pagination
            total={totalCount}
            current={page}
            pageSize={10}
            className="justify-self-end justify-end flex-nowrap"
            onChange={(p) => setPage(p)}
          />
        </div>
      </SurfaceCard>
      ) : (
        <SurfaceCard className="overflow-hidden">
          <Empty className="border-0 py-20">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyDescription>
                暂无威胁告警数据
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </SurfaceCard>
      )}

      {alarmDetailDrawerVisible && (
        <AlarmDetail
          visible={alarmDetailDrawerVisible}
          onClose={() => setAlarmDetailDrawerVisible(false)}
          selectedAlarmType={selectedAlarmType}
          item={selectedItem}
          aiAgentHostList={aiAgentHostList}
          refreshTable={refreshTable}
          clearSelected={clearSelected}
          hasFlagship={machineVersionCount?.UltimateVersionNum > 0}
        />
      )}

      <BatchOperatorDialog
        visible={batchHandleModalVisible}
        title={batchTitleMap[batchType]}
        okText={'确定'}
        onCancel={() => setBatchHandleModalVisible(false)}
        content={
          <div>
            {batchType != 'del'
              && selectedRows?.filter(
                item => String(item?.[selectedAlarmType === BASH_ALARM ? 'Status' : 'HandleStatus']) != '0',
              )?.length > 0 && (
                <p style={{ marginTop: 10 }}>
                  其中有{selectedRows?.filter(
                    item => String(item?.[selectedAlarmType === BASH_ALARM ? 'Status' : 'HandleStatus']) !== '0',
                  )?.length}条数据将不能执行操作。只有数据为"待处理"状态，才能执行{batchTitleMap[batchType]}操作
                </p>
              )}
            {((batchType != 'del'
              && selectedRows?.filter(
                item => String(item?.[selectedAlarmType === BASH_ALARM ? 'Status' : 'HandleStatus']) === '0',
              )?.length > 0)
              || batchType == 'del') && (
                <p style={{ marginTop: 10 }}>
                  您确定要对选中的数据进行{batchTitleMap[batchType]}操作吗？
                </p>
              )}
          </div>
        }
        disabled={
          batchType != 'del'
          && selectedRows?.filter(
            item => String(item?.[selectedAlarmType === BASH_ALARM ? 'Status' : 'HandleStatus']) === '0',
          )?.length == 0
        }
        onOk={() => {
          setBatchHandleModalVisible(false);
          const allIds = selectedKeys?.map?.(Id => Number(Id));
          const ids = selectedRows
            ?.filter(item => String(item?.[selectedAlarmType === BASH_ALARM ? 'Status' : 'HandleStatus']) === '0')
            ?.map((item: any) => Number(item?.Id));
          modifyEventsStatus(
            selectedAlarmType === BASH_ALARM ? RISK_TYPE_BASH : RISK_TYPE_MALICIOUS,
            batchType,
            batchType === 'del' ? allIds : ids,
            () => {
              refreshTable();
              clearSelected();
            },
            setBatchTimer,
            setIsBatchLoading,
          );
        }}
        renderItem={item => {
          const ip = (item?.MachineIp || item?.Hostip || item?.HostIp) ?? '-';
          return `OpenClaw：${ip} - 命中策略：${selectedAlarmType === BASH_ALARM ? item?.RuleName ?? '-' : item?.PolicyName ?? '-'}`;
        }}
        data={selectedRows}
      />
    </div>
  );
}
