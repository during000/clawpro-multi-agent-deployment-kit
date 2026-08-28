/*  */


import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Base64 } from '@/vendor/js-base64';
import { toast } from 'sonner';
import { Info, RefreshCw, ChevronUp, ChevronDown, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { StatusTag } from '@/components/ui/status-tag';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem as SelectOption,
} from '@/components/ui/select';
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from '@/components/ui/alert-dialog';
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableCell,
  TableActionCell,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea } from '@/components/ui/scroll-area';
import { SurfaceCard } from '@/components/ui/Surface';
import { Pagination } from '@/components/ui/pagination';
import { ModifyRiskDnsPolicyStatus, DescribeRiskDnsPolicyList, DeleteRiskDnsPolicy } from '@/pages/admin/Security/api';

import { BLOCK_DEEP_ID, SYSTEM_STANDARD_ID, BLOCK_STANDARD_ID } from '../BashPolicy/Constants';
import { AUTHORIZE_ROUTE } from '../../constants';
import { getSelectionRows } from '../../Common/CommonRiskHandleFunc';

import { PolicyDetailDrawer } from './PolicyDetailDrawer';
import { EditPolicyDrawer } from './EditPolicyDrawer';
import {
  POLICY_TYPES,
  ALL_POLICY_TYPES_DATA,
  GetHostTypeText,
  getPolicyActionsData,
  getPolicyActionMap,
  POLICY_ACTION_THEME_MAP,
} from './CommonType';

export const renderSegOptionsNew = (isShowIcon: boolean, isSelectStandard: boolean, setShowCloseBubble = (d: any) => d) => [
  {
    text: (
      <span className="inner">
        <span>{'标准模式'}</span>
      </span>
    ),
    tooltip: isShowIcon ? '综合多个引擎检测结果，仅针对高置信度的风险进行自动防护，更适合日常安全运营使用。' : null,
    value: '0',
  },
  {
    text: (
      <span className="inner">
        <span>{'重保模式'}</span>
      </span>
    ),
    tooltip: isShowIcon ? '综合多个引擎检测结果，针对中、高置信度的风险进行自动防护。可能存在误拦截风险，适合重保防护，请谨慎启用。' : null,
    value: '1',
  },
];

/** 列定义 */
interface ColDef {
  key: string;
  header: string;
  width?: number | string;
  fixed?: 'left' | 'right';
  isAlwaysShow?: boolean;
  render?: (item: any, rowKey: string, index: number) => React.ReactNode;
}

export function MaliciousPolicyList({ hasFlagship, getInitPolicyCount, aiAgentHostList, mockData }: any) {
  const requestIdRef = useRef(0);
  const isMock = !!(mockData && mockData.length);

  const [isLoading, setIsLoading] = useState(false);
  const [isInit, setIsInit] = useState(true);
  const [handleType, setHandleType] = useState('');
  const [selectItem, setSelectItem] = useState({} as any);
  const [confirmModal, setConfirmModal] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [settingVisible, setSettingVisible] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState([] as any);
  const [selectedRows, setSelectedRows] = useState([] as any);
  const [loadingIndex, setLoadingIndex] = useState(0);
  const [loadingList, setLoadingList] = useState([] as any);
  const [tableshow, setTableshow] = useState(false);
  const [tableData, setTableData] = useState<any[]>([]);
  const [allData, setAllData] = useState<any[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalProtectMode, setModalProtectMode] = useState('0');
  const [autoBlockModalVisible, setAutoBlockModalVisible] = useState(false);
  const [modalAutoBlockSwitch, setModalAutoBlockSwitch] = useState(false);
  const [downloadParams, setDownloadParams] = useState({} as any);
  const [initColKeys, setInitColKeys] = useState<string[]>([]);
  const [changeModePopVisible, setChangeModePopVisible] = useState(false);
  const [standardBlockPolicy, setStandardBlockPolicy] = useState({} as any);
  const [importantBlockPolicy, setImportantBlockPolicy] = useState({} as any);
  const [closeAutoBlockModal, setCloseAutoBlockModal] = useState(false);
  const [columnSettingOpen, setColumnSettingOpen] = useState(false);

  // 筛选状态
  const [filterPolicyType, setFilterPolicyType] = useState('');
  const [filterPolicyAction, setFilterPolicyAction] = useState('');
  const [filterIsEnabled, setFilterIsEnabled] = useState('');
  const [searchField, setSearchField] = useState('PolicyName');
  const [searchKey, setSearchKey] = useState('');

  const refreshTable = useCallback(() => {
    // trigger refetch
    fetchTableData();
    getInitPolicyCount?.();
  }, []);

  const clearSelected = useCallback(() => {
    setSelectedKeys([]);
    setSelectedRows([]);
  }, []);

  const getInitColKeys = async () => {
    // 列设置工具已在历史 refactor 中移除，保留空实现避免运行时报错
    setInitColKeys([]);
  };

  const getAutoBlockData = async () => {
    if (isMock) {
      const data = (mockData as any[]).filter((item: any) => String(item?.PolicyAction) === '2' && String(item?.PolicyType) === '0');
      const standPolocy = data?.find?.((it: any) => it?.PolicyId === BLOCK_STANDARD_ID) || {};
      const importantPolicy = data?.find?.((it: any) => it?.PolicyId === BLOCK_DEEP_ID) || {};
      setStandardBlockPolicy(standPolocy);
      setImportantBlockPolicy(importantPolicy);
      // 直接灌 mock 表格
      setAllData(mockData);
      setTotalCount(mockData.length);
      const offset = (page - 1) * pageSize;
      const currentData = mockData.slice(offset, offset + pageSize);
      setLoadingList(currentData.map(() => false));
      setTableData(currentData);
      setIsInit(false);
      setIsLoading(false);
      return;
    }
    const res: any = await DescribeRiskDnsPolicyList({
      Offset: 0,
      Limit: 5,
      Filters: [{ Name: 'PolicyType', Values: ['0'] }],
    });
    const data = res?.List?.filter?.((item: any) => String(item?.PolicyAction) === '2');
    const standPolocy = data?.filter?.((item: any) => item?.PolicyId === BLOCK_STANDARD_ID)?.[0] || {};
    const importantPolicy = data?.filter?.((item: any) => item?.PolicyId === BLOCK_DEEP_ID)?.[0] || {};
    setStandardBlockPolicy(standPolocy);
    setImportantBlockPolicy(importantPolicy);
    fetchTableData(standPolocy, importantPolicy);
    setIsInit(false);
  };

  const handleSwitchChange = async (item: { PolicyId: any; IsEnabled: any; }, index = loadingIndex) => {
    setLoadingList((prev: boolean[]) => [...prev?.slice?.(0, index), true, ...prev?.slice?.(index + 1)]);
    const res: any = await ModifyRiskDnsPolicyStatus({
      PolicyId: item?.PolicyId,
      IsEnabled: String(item?.IsEnabled) === '0' ? 1 : 0,
    });
    if (res) {
      toast.success('操作成功');
      clearSelected();
      getAutoBlockData();
    }
  };

  const handleDelPolicy = async (item: any = undefined) => {
    setDetailVisible(false);
    const handleSelectedKeys = selectedKeys.map((k: string) => Number(k));
    const params = {
      PolicyIds: item?.PolicyId ? [Number(item?.PolicyId)] : handleSelectedKeys,
    };
    const res: any = await DeleteRiskDnsPolicy(params);
    if (res) {
      toast.success('操作成功');
      setDetailVisible(false);
      getAutoBlockData();
      clearSelected();
    }
  };

  const handleAutoBlockChange = async (isOpen: boolean, mode: string) => {
    setAutoBlockModalVisible(false);
    setLoadingList((prev: boolean[]) => [...prev?.slice?.(0, loadingIndex), true, ...prev?.slice?.(loadingIndex + 1)]);
    const firstCloseId = isOpen
      ? mode === '0'
        ? importantBlockPolicy?.PolicyId
        : standardBlockPolicy?.PolicyId
      : standardBlockPolicy?.PolicyId;
    const res: any = await ModifyRiskDnsPolicyStatus({
      PolicyId: firstCloseId,
      IsEnabled: 1,
    });
    const res1: any = await ModifyRiskDnsPolicyStatus({
      PolicyId:
        firstCloseId === standardBlockPolicy?.PolicyId ? importantBlockPolicy?.PolicyId : standardBlockPolicy?.PolicyId,
      IsEnabled: isOpen ? 0 : 1,
    });
    if (res && res1) {
      toast.success('操作成功');
    }
    getAutoBlockData();
  };

  /** 数据请求 — 替代原 TablePanel 的 request */
  const fetchTableData = useCallback(async (standPolocy = undefined, importantPolicy = undefined) => {
    // mock 模式：本地按筛选/搜索过滤
    if (isMock) {
      let list: any[] = [...(mockData as any[])];
      if (filterPolicyType) list = list.filter(d => String(d?.PolicyType) === filterPolicyType);
      if (filterPolicyAction) list = list.filter(d => String(d?.PolicyAction) === filterPolicyAction);
      if (filterIsEnabled && filterIsEnabled !== 'undefined') list = list.filter(d => String(d?.IsEnabled) === filterIsEnabled);
      if (searchKey) {
        const kw = searchKey.trim();
        if (searchField === 'PolicyName') {
          list = list.filter(d => (d?.PolicyName || '').includes(kw));
        } else if (searchField === 'Domain') {
          list = list.filter((d: any) =>
            (d?.Rules?.Domain || []).some((x: string) => (x || '').includes(kw))
            || (d?.Rules?.Ip || []).some((x: string) => (x || '').includes(kw)),
          );
        }
      }
      setAllData(list);
      setTotalCount(list.length);
      const offset = (page - 1) * pageSize;
      const currentData = list.slice(offset, offset + pageSize);
      setLoadingList(currentData.map(() => false));
      setTableData(currentData);
      setIsLoading(false);
      return;
    }

    const rid = ++requestIdRef.current;
    setIsLoading(true);
    try {
      const filters: any[] = [];
      if (filterPolicyType) filters.push({ Name: 'PolicyType', Values: [filterPolicyType] });
      if (filterPolicyAction) filters.push({ Name: 'PolicyAction', Values: [filterPolicyAction] });
      if (filterIsEnabled && filterIsEnabled !== 'undefined') filters.push({ Name: 'IsEnabled', Values: [filterIsEnabled] });
      if (searchKey) {
        const val = searchField === 'Domain' ? Base64.encode(searchKey) : searchKey;
        filters.push({ Name: searchField, Values: [val] });
      }

      const params: any = {
        Offset: 0,
        Limit: 100,
        Filters: filters,
        Order: 'DESC',
        By: 'UpdateTime',
      };
      setDownloadParams({ Filters: filters, Order: 'DESC', By: 'UpdateTime' });

      const res: any = await DescribeRiskDnsPolicyList(params);
      if (rid !== requestIdRef.current) return;

      let list = res?.List || [];
      if (res?.TotalCount > 100) {
        const num = Math.min(Math.ceil((res?.TotalCount - 100) / 100), 19);
        const resMore = await Promise.all(
          new Array(num)
            .fill(1)
            .map((d, index) => DescribeRiskDnsPolicyList({ ...params, Offset: 100 * (index + 1), Limit: 100 })),
        );
        if (rid !== requestIdRef.current) return;
        list = list?.concat?.(resMore?.map?.((item: any) => item?.List ?? [])?.flat?.(3));
      }

      // 拦截策略特殊处理
      const block = list?.filter?.((item: any) => String(item?.PolicyType) === '0' && String(item?.PolicyAction) === '2');
      list = (!block?.length
        ? []
        : (filterIsEnabled
          && ((String(filterIsEnabled) === '1'
            && [standPolocy || standardBlockPolicy, importantPolicy || importantBlockPolicy]?.every?.(d => String(d?.IsEnabled) === '1'))
            || (String(filterIsEnabled) === '0' && block?.length > 0)))
          || !filterIsEnabled
          || filterIsEnabled === 'undefined'
          ? [importantPolicy || importantBlockPolicy]
          : []
      )?.concat?.(list?.filter?.((item: any) => !(String(item?.PolicyType) === '0' && String(item?.PolicyAction) === '2')));

      if (detailVisible && selectItem?.PolicyId) {
        const found = list?.filter?.((item: { PolicyId: any; }) => item?.PolicyId === selectItem?.PolicyId)?.[0] || {};
        setSelectItem(found);
      }

      setAllData(list);
      setTotalCount(list?.length || 0);

      const offset = (page - 1) * pageSize;
      const currentData = list?.slice?.(offset, offset + pageSize) || [];
      setLoadingList(currentData.map(() => false));
      setTableData(currentData);
    } catch {
      // error handled
    } finally {
      if (rid === requestIdRef.current) {
        setIsLoading(false);
      }
    }
  }, [filterPolicyType, filterPolicyAction, filterIsEnabled, searchField, searchKey, page, pageSize, standardBlockPolicy, importantBlockPolicy, detailVisible, selectItem?.PolicyId, isMock, mockData]);

  // Re-slice when page/pageSize changes (from allData)
  useEffect(() => {
    if (!allData.length) return;
    const offset = (page - 1) * pageSize;
    const currentData = allData?.slice?.(offset, offset + pageSize) || [];
    setLoadingList(currentData.map(() => false));
    setTableData(currentData);
  }, [page, pageSize, allData]);

  useEffect(() => {
    if (autoBlockModalVisible) {
      const isOpen = String(standardBlockPolicy?.IsEnabled) === '0' || String(importantBlockPolicy?.IsEnabled) === '0';
      setModalAutoBlockSwitch(true);
      setModalProtectMode(isOpen ? String(standardBlockPolicy?.IsEnabled) : '0');
    }
  }, [autoBlockModalVisible]);

  useEffect(() => {
    getInitColKeys();
    getAutoBlockData();
  }, []);

  // Refetch when filters change
  useEffect(() => {
    if (!isInit) {
      setPage(1);
      fetchTableData();
    }
  }, [filterPolicyType, filterPolicyAction, filterIsEnabled, searchKey]);

  // 列定义
  const allColumns: ColDef[] = useMemo(() => [
    {
      key: 'PolicyName',
      header: '策略名称',
      width: 220,
      // 与「组件表格」固定列规范一致：复选框 + 名称列同时左固定，
      // 由最右侧的名称列承载边界阴影（复选框列 fixedShadow={false}）。
      fixed: 'left',
      isAlwaysShow: true,
      render: (item, rowKey, recordIndex) =>
      (String(item?.PolicyType) === '0' ? (
        item?.PolicyName
      ) : (
        <Button
          variant="link-dark"
          className="p-0 h-auto inline-block max-w-full overflow-hidden text-ellipsis whitespace-nowrap"
          title={item?.PolicyName}
          onClick={() => {
            setSelectItem(item);
            setLoadingIndex(recordIndex);
            setDetailVisible(true);
          }}
        >
          {item?.PolicyName}
        </Button>
      )),
    },
    {
      key: 'PolicyType',
      header: '策略类型',
      render: item =>
      (item?.PolicyType === 0 || item?.PolicyType === 1 ? (
        <span className="inline-flex items-center text-[#0A0A0A]">
          {POLICY_TYPES[item?.PolicyType] || '--'}
          {String(item?.PolicyType) === '0' && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Info className="w-3.5 h-3.5 text-[#A3A3A3] hover:text-[#525252] ml-1 cursor-pointer" />
              </TooltipTrigger>
              <TooltipContent className="max-w-[320px]">
                {'系统策略为腾讯OpenClaw运营专家与算法专家经过多模型沉淀的规则配置，适用于大部分的恶意请求检测。'}
              </TooltipContent>
            </Tooltip>
          )}
        </span>
      ) : (
        '--'
      )),
    },
    {
      key: 'CreateTime',
      header: '黑/白名单',
      width: 110,
      render: item =>
      (String(item?.PolicyAction) === '1' ? (
        <span className="text-[#0A0A0A]">白名单</span>
      ) : (
        <span className="text-[#0A0A0A]">黑名单</span>
      )),
    },
    {
      key: 'Domains',
      header: '域名详情',
      render: (item, rowKey, recordIndex) => {
        if (String(item?.PolicyType) === '0') {
          return '腾讯云恶意域名库';
        }
        const domainCount = item?.Domains?.length || 0;
        if (domainCount === 0) {
          return '--';
        }
        if (domainCount === 1) {
          return item?.Domains?.join?.('、') || '--';
        }
        return (
          <span>
            {'共'}
            <Button
              variant="link"
              className="p-0 h-auto align-top m-0"
              onClick={() => {
                setSelectItem(item);
                setLoadingIndex(recordIndex);
                setDetailVisible(true);
              }}
            >
              <span>{domainCount}</span>
            </Button>
            <span>{'个'}</span>
          </span>
        );
      },
    },
    {
      key: 'HostScope',
      header: '生效OpenClaw',
      render: (item, rowKey, recordIndex) => (
        <div className="maliciousRequest-policy-host">
          {String(item?.HostScope) !== '0' ? (
            GetHostTypeText(item?.HostScope)
          ) : (
            <span>
              {item?.HostIds?.length ? (
                <Button
                  variant="link"
                  className="p-0 h-auto"
                  onClick={() => {
                    setSelectItem(item);
                    setLoadingIndex(recordIndex);
                    setDetailVisible(true);
                  }}
                >
                  <span>{`${item?.HostIds?.length}`}</span>
                </Button>
              ) : (
                <span>0</span>
              )}
              <span>{'台'}</span>
            </span>
          )}
        </div>
      ),
    },
    {
      key: 'UpdateTime',
      header: '更新时间',
    },
    {
      key: 'PolicyAction',
      header: '执行动作',
      width: 250,
      render: (item, text, index) => {
        const theme = POLICY_ACTION_THEME_MAP[item?.PolicyAction];
        // 与「命令管控策略」表保持一致：执行动作统一用 StatusTag mode="soft"
        // 拦截 → red / 放行 → green / 告警 → orange
        const actionVariant: 'red' | 'orange' | 'green' | 'gray' =
          theme === 'error' ? 'red'
            : theme === 'success' ? 'green'
              : theme === 'warning' ? 'orange'
                : 'gray';
        return (
        <div>
          <StatusTag variant={actionVariant} mode="soft">{getPolicyActionMap()?.[item?.PolicyAction]}</StatusTag>
          {String(item?.PolicyType) === '0' && String(item?.PolicyAction) === '2' ? (
            (String(standardBlockPolicy?.IsEnabled) === '1' && String(importantBlockPolicy?.IsEnabled) === '1')
              || loadingList[index] ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="flex mt-1.5">
                    <div className="inline-flex items-center gap-0.5 p-0.5 bg-[#F5F5F5] rounded-[4px] pointer-events-none opacity-50 w-fit">
                      {renderSegOptionsNew(true, true).map(opt => {
                        const isActive = opt.value === '0';
                        return (
                          <span
                            key={opt.value}
                            className={`inline-flex items-center gap-0.5 px-2 py-0.5 text-xs rounded-[3px] transition-colors leading-5 ${
                              isActive
                                ? 'bg-white text-[#0A0A0A] font-medium'
                                : 'text-[#737373] font-normal'
                            }`}
                            style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                          >
                            {opt.text}
                            {opt.tooltip && (
                              <Info className={`w-3 h-3 ${isActive ? 'text-[var(--text-muted)]' : 'text-[var(--text-weak)]'}`} />
                            )}
                          </span>
                        );
                      })}
                    </div>
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  {loadingList[index] ? null : !hasFlagship ? (
                    <span>
                      <span>{'恶意请求自动拦截属于旗舰版功能，开启后将自动拦截检测出的系统恶意请求，点击 '}</span>
                      <a className="text-[#1447E6] cursor-pointer hover:underline" onClick={() => window.open(AUTHORIZE_ROUTE)}>
                        {'升级版本'}
                      </a>
                      <span>{'，一键开启拦截。'}</span>
                    </span>
                  ) : (
                    <span>
                      {'策略未开启，暂无法进行模式切换，可点击 '}
                      <a
                        className="text-[#1447E6] cursor-pointer hover:underline"
                        onClick={() => setAutoBlockModalVisible(true)}
                      >
                        {'开启策略'}
                      </a>
                    </span>
                  )}
                </TooltipContent>
              </Tooltip>
            ) : (
              <>
                <div className="flex mt-1.5">
                  <div className="inline-flex items-center gap-0.5 p-0.5 bg-[#F5F5F5] rounded-[4px] w-fit">
                    {renderSegOptionsNew(true, String(standardBlockPolicy?.IsEnabled) === '0').map(opt => {
                      const isActive = opt.value === String(standardBlockPolicy?.IsEnabled);
                      return (
                        <button
                          key={opt.value}
                          type="button"
                          onClick={() => {
                            if (opt.value !== String(standardBlockPolicy?.IsEnabled)) {
                              setLoadingIndex(index);
                              setChangeModePopVisible(true);
                            }
                          }}
                          className={`inline-flex items-center gap-0.5 px-2 py-0.5 text-xs rounded-[3px] transition-colors leading-5 ${
                            isActive
                              ? 'bg-white text-[#0A0A0A] font-medium'
                              : 'text-[#737373] hover:text-[#0A0A0A] font-normal'
                          }`}
                          style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                        >
                          {opt.text}
                          {opt.tooltip && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Info
                                  className={`w-3 h-3 ${isActive ? 'text-[var(--text-muted)]' : 'text-[var(--text-weak)]'} cursor-help`}
                                  onClick={e => e.stopPropagation()}
                                />
                              </TooltipTrigger>
                              <TooltipContent className="max-w-[320px]">
                                {opt.tooltip}
                              </TooltipContent>
                            </Tooltip>
                          )}
                        </button>
                      );
                    })}
                  </div>
                </div>
                <AlertDialog open={changeModePopVisible} onOpenChange={setChangeModePopVisible}>
                  <AlertDialogContent className="sm:max-w-[460px]">
                    <AlertDialogHeader>
                      <AlertDialogTitle>
                        {String(importantBlockPolicy?.IsEnabled) === '0'
                          ? '确认切换为标准模式？'
                          : '确认切换为重保模式？'}
                      </AlertDialogTitle>
                      <AlertDialogDescription>
                        {String(importantBlockPolicy?.IsEnabled) === '0'
                          ? '确认后，将切换为标准模式，综合多个引擎检测结果，仅针对高置信度的风险进行自动防护，更适合日常安全运营使用。'
                          : (
                            <span>
                              {'确认后，将切换为重保模式，综合多个引擎检测结果，针对高、中置信度的风险进行自动防护，'}
                              <span className="text-[#DC2626]">{'可能存在误拦截风险'}</span>
                              {'，适合重保防护，请谨慎启用。'}
                            </span>
                          )}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{'取消'}</AlertDialogCancel>
                      <AlertDialogAction onClick={() => {
                        handleAutoBlockChange?.(true, String(importantBlockPolicy?.IsEnabled) === '0' ? '0' : '1');
                      }}>
                        {'确定'}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </>
            )
          ) : null}
        </div>
        );
      },
    },
    {
      key: 'IsEnabled',
      header: '开关',
      width: 100,
      render: (item, rowKey, recordIndex) =>
      (String(item?.PolicyType) === '0'
        && String(item?.PolicyAction) !== '2'
        && String(item?.PolicyId) === String(SYSTEM_STANDARD_ID) ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Switch disabled checked className="ml-[5px]" />
            </span>
          </TooltipTrigger>
          <TooltipContent>{'系统默认告警策略默认生效，不支持关闭'}</TooltipContent>
        </Tooltip>
      ) : String(item?.PolicyAction) === '2' && String(item?.PolicyType) === '0' ? (
        !hasFlagship ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Switch
                  disabled
                  checked={
                    String(standardBlockPolicy?.IsEnabled) === '0' || String(importantBlockPolicy?.IsEnabled) === '0'
                  }
                  className="ml-[5px]"
                />
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <span>
                <span>{'恶意请求自动拦截属于旗舰版功能，开启后将自动拦截检测出的系统恶意请求，点击 '}</span>
                <a className="text-[#1447E6] cursor-pointer hover:underline" onClick={() => window.open(AUTHORIZE_ROUTE)}>
                  {'升级版本'}
                </a>
                <span>{'，一键开启拦截。'}</span>
              </span>
            </TooltipContent>
          </Tooltip>
        ) : (
          <Switch
            disabled={loadingList[recordIndex]}
            checked={String(standardBlockPolicy?.IsEnabled) === '0' || String(importantBlockPolicy?.IsEnabled) === '0'}
            className="ml-[5px]"
            onCheckedChange={() => {
              if (String(standardBlockPolicy?.IsEnabled) === '0' || String(importantBlockPolicy?.IsEnabled) === '0') {
                setCloseAutoBlockModal(true);
              } else {
                setAutoBlockModalVisible(true);
              }
            }}
          />
        )
      ) : String(item?.PolicyAction) === '2' && !hasFlagship ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Switch disabled checked={String(item?.IsEnabled) === '0'} className="ml-[5px]" />
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[200px]">
            <span>
              <span>{'当前暂无旗舰版OpenClaw，无法设置拦截策略，可'}</span>
              <a onClick={() => window.open(AUTHORIZE_ROUTE)} className="text-white underline cursor-pointer">
                {'点击升级版本'}
              </a>
            </span>
          </TooltipContent>
        </Tooltip>
      ) : (
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <span>
              <Switch
                disabled={loadingList[recordIndex]}
                checked={String(item?.IsEnabled) === '0'}
                className="ml-[5px]"
              />
            </span>
          </AlertDialogTrigger>
          <AlertDialogContent className="sm:max-w-[420px]">
            <AlertDialogHeader>
              <AlertDialogTitle>{`确定${String(item?.IsEnabled) === '0' ? '关闭' : '开启'}此策略？`}</AlertDialogTitle>
              <AlertDialogDescription>
                {String(item?.IsEnabled) === '0'
                  ? '确认后，将关闭此策略，后续命中策略内容时，将不再执行相应动作，请谨慎操作。'
                  : '确认后，将开启此策略，后续命中策略内容时，将对应执行相应动作。'}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{'取消'}</AlertDialogCancel>
              <AlertDialogAction onClick={() => handleSwitchChange(item, recordIndex)}>
                {'确定'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )),
    },
    {
      key: 'Action',
      header: '操作',
      width: 100,
      isAlwaysShow: true,
      render: item => (
        <>
          {String(item?.PolicyAction) === '2' && !hasFlagship ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button variant="link" disabled>
                    {'编辑'}
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent className="max-w-[200px]">
                <span>
                  {'当前暂无旗舰版OpenClaw，无法设置拦截策略，可'}
                  <a
                    onClick={() => window.open(AUTHORIZE_ROUTE)}
                    className="text-white underline cursor-pointer"
                  >
                    {'点击升级版本'}
                  </a>
                </span>
              </TooltipContent>
            </Tooltip>
          ) : String(item?.PolicyType) === '0' ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button variant="link" disabled>
                    {'编辑'}
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{'系统策略不支持编辑'}</TooltipContent>
            </Tooltip>
          ) : (
            <Button
              variant="link"
              onClick={() => {
                setHandleType('edit');
                setSelectItem(item);
                setSettingVisible(true);
              }}
            >
              {'编辑'}
            </Button>
          )}
          {String(item?.PolicyType) === '0' ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button variant="link" disabled>
                    {'删除'}
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{'系统策略不支持删除'}</TooltipContent>
            </Tooltip>
          ) : (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="link">{'删除'}</Button>
              </AlertDialogTrigger>
              <AlertDialogContent className="sm:max-w-[420px]">
                <AlertDialogHeader>
                  <AlertDialogTitle>{'确认删除此策略？'}</AlertDialogTitle>
                  <AlertDialogDescription>
                    {'确认后，策略将被删除，无法恢复，策略范围内的资产将不再生效，请谨慎操作。'}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{'取消'}</AlertDialogCancel>
                  <AlertDialogAction onClick={() => handleDelPolicy(item)}>
                    {'确定'}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
        </>
      ),
    },
  ], [standardBlockPolicy, importantBlockPolicy, hasFlagship, loadingList, changeModePopVisible]);

  // 根据列设置过滤显示列
  const visibleColumns = useMemo(() => {
    if (!initColKeys?.length) return allColumns;
    return allColumns.filter(col => col.isAlwaysShow || initColKeys.includes(col.key));
  }, [allColumns, initColKeys]);

  // 行选择逻辑
  const selectableRows = useMemo(() => tableData.filter(d => String(d?.PolicyType) === '1'), [tableData]);
  const allSelected = selectableRows.length > 0 && selectableRows.every(d => selectedKeys.includes(String(d?.PolicyId)));
  const someSelected = !allSelected && selectableRows.some(d => selectedKeys.includes(String(d?.PolicyId)));

  const toggleSelectAll = useCallback(() => {
    if (allSelected) {
      clearSelected();
    } else {
      const keys = selectableRows.map(d => String(d?.PolicyId));
      const rows = getSelectionRows(selectedRows, keys, tableData, 'PolicyId');
      setSelectedKeys(keys);
      setSelectedRows(rows);
    }
  }, [allSelected, selectableRows, selectedRows, tableData, clearSelected]);

  const toggleSelectRow = useCallback((item: any) => {
    const key = String(item?.PolicyId);
    const isSelected = selectedKeys.includes(key);
    const newKeys = isSelected ? selectedKeys.filter((k: string) => k !== key) : [...selectedKeys, key];
    const newRows = getSelectionRows(selectedRows, newKeys, tableData, 'PolicyId');
    setSelectedKeys(newKeys);
    setSelectedRows(newRows);
  }, [selectedKeys, selectedRows, tableData]);

  return (
    <div>
      {/* ===== 工具栏（主按钮统一右置） ===== */}
      <div className="flex items-center justify-between gap-2 flex-wrap mb-3">
        {/* 左侧：筛选 + 搜索 + 刷新
            停服态豁免：筛选下拉、关键字搜索、刷新按钮均属「查看类」操作，
            不改动后端数据，停服时保持可用。整组打 data-billing-exempt，
            overlay 的灰化 CSS 与点击拦截同时放行；组件自身若传入 disabled，
            仍由原生 disabled 生效（延续既有禁用）。右侧「删除/创建策略」是写操作，
            不在此豁免范围内，继续沿用停服禁用。*/}
        <div className="flex items-center gap-2 flex-wrap" data-billing-exempt>
          <Select value={filterPolicyType || 'undefined'} onValueChange={val => { setFilterPolicyType(val === 'undefined' ? '' : val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="请选择策略类型" />
            </SelectTrigger>
            <SelectContent>
              {ALL_POLICY_TYPES_DATA.map((d: any) => (
                <SelectOption key={d.value ?? '__all__'} value={d.value || '__all__'}>{d.text}</SelectOption>
              ))}
            </SelectContent>
          </Select>
          <Select value={filterPolicyAction || 'undefined'} onValueChange={val => { setFilterPolicyAction(val === 'undefined' ? '' : val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="请选择执行动作" />
            </SelectTrigger>
            <SelectContent>
              {getPolicyActionsData().map((d: any) => (
                <SelectOption key={d.value} value={String(d.value)}>{d.text}</SelectOption>
              ))}
            </SelectContent>
          </Select>
          <Select value={filterIsEnabled || '__all__'} onValueChange={val => { setFilterIsEnabled(val === '__all__' ? '' : val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="请选择生效状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectOption value="__all__">全部生效状态</SelectOption>
              <SelectOption value="0">已生效</SelectOption>
              <SelectOption value="1">未生效</SelectOption>
            </SelectContent>
          </Select>
          <Select value={searchField} onValueChange={val => { setSearchField(val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[110px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectOption value="PolicyName">策略名称</SelectOption>
              <SelectOption value="Domain">域名</SelectOption>
            </SelectContent>
          </Select>
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#A3A3A3]" />
            <Input
              placeholder="请输入关键字搜索"
              value={searchKey}
              onChange={e => { setSearchKey(e.target.value); setPage(1); }}
              className="pl-8 h-8 w-[180px] bg-white"
            />
          </div>
          <button
            type="button"
            onClick={() => getAutoBlockData()}
            className="w-8 h-8 flex items-center justify-center rounded-[4px] border border-[#E5E5E5] bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
            title="刷新表格"
            aria-label="刷新表格"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
          </button>
        </div>
        {/* 右侧：主按钮组（删除 / 创建策略） */}
        <div className="flex items-center gap-2 flex-wrap">
          <Button
            variant="claw-outline"
            size="claw-sm"
            disabled={selectedKeys?.length === 0}
            onClick={() => {
              setHandleType('del');
              setSelectItem({});
              setTableshow(false);
              setConfirmModal(true);
            }}
          >
            {'删除'}
          </Button>
          <Button
            variant="claw-primary"
            size="claw-sm"
            className="policy-create-btn"
            onClick={() => {
              setHandleType('create');
              setSelectItem({});
              setSettingVisible(true);
            }}
          >
            {'创建策略'}
          </Button>
        </div>
      </div>

      {/* ===== 表格 ===== */}
      <SurfaceCard className="relative overflow-hidden">
        {isLoading && (
          <div className="absolute inset-0 bg-white/60 z-10 flex items-center justify-center">
            <RefreshCw className="w-6 h-6 animate-spin text-[#1447E6]" />
          </div>
        )}
        <Table scrollX={1100} variant="white" containerClassName="bg-white">
          <TableHeader>
            <TableRow>
              <TableHead fixed="left" fixedShadow={false} className="w-14">
                <Checkbox
                  checked={allSelected ? true : someSelected ? 'indeterminate' : false}
                  onCheckedChange={toggleSelectAll}
                />
              </TableHead>
              {visibleColumns.map((col) => {
                const isAction = col.key === 'Action';
                return (
                  <TableHead
                    key={col.key}
                    fixed={isAction ? 'right' : col.fixed}
                    style={col.width ? { width: col.width, minWidth: col.fixed === 'left' ? col.width : undefined } : undefined}
                  >
                    {col.header}
                  </TableHead>
                );
              })}
            </TableRow>
          </TableHeader>
          <TableBody>
            {tableData.length > 0 ? (
              tableData.map((item: any, idx: number) => {
                const rowId = String(item?.PolicyId);
                const isSelected = selectedKeys.includes(rowId);
                const isSelectable = String(item?.PolicyType) === '1';
                return (
                  <TableRow key={item?.PolicyId || idx}>
                    <TableCell fixed="left" fixedShadow={false} className="w-14">
                      {isSelectable ? (
                        <Checkbox checked={isSelected} onCheckedChange={() => toggleSelectRow(item)} />
                      ) : (
                        <Checkbox disabled checked={false} />
                      )}
                    </TableCell>
                    {visibleColumns.map((col) => {
                      if (col.key === 'Action') {
                        return (
                          <TableActionCell key={col.key} fixed="right">
                            {col.render ? col.render(item, rowId, idx) : null}
                          </TableActionCell>
                        );
                      }
                      return (
                        <TableCell
                          key={col.key}
                          fixed={col.fixed}
                          style={col.fixed === 'left' && col.width ? { minWidth: col.width } : undefined}
                        >
                          {col.render ? col.render(item, rowId, idx) : (item?.[col.key] ?? '--')}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                );
              })
            ) : (
              <TableRow>
                <TableCell colSpan={visibleColumns.length + 1} className="text-center py-16 text-[var(--text-weak)]">
                  {'暂无数据'}
                </TableCell>
              </TableRow>
            )}
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

      <EditPolicyDrawer
        visible={settingVisible}
        setVisible={setSettingVisible}
        type={handleType}
        selectItem={selectItem}
        refreshTable={() => getAutoBlockData()}
        hasFlagship={hasFlagship}
        aiAgentHostList={aiAgentHostList}
      />

      <PolicyDetailDrawer
        loading={loadingList[loadingIndex]}
        selectItem={selectItem}
        detailVisible={detailVisible}
        setDetailVisible={setDetailVisible}
        hasFlagship={hasFlagship}
        setHandleType={setHandleType}
        handleDelPolicy={handleDelPolicy}
        handleSwitchChange={handleSwitchChange}
        setSettingVisible={setSettingVisible}
        aiAgentHostList={aiAgentHostList}
      />

      {/* 关闭自动拦截确认弹窗 */}
      <Dialog open={closeAutoBlockModal} onOpenChange={setCloseAutoBlockModal}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>{'确认关闭自动拦截？'}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-[#525252] leading-relaxed">
            {'关闭该功能后，OpenClaw将不再自动拦截检测到的恶意域名/IP访问，可能造成被入侵风险，请谨慎操作。'}
          </p>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setCloseAutoBlockModal(false)}>
              {'取消'}
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                handleAutoBlockChange(false, '0');
                setCloseAutoBlockModal(false);
              }}
            >
              {'确定'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 批量操作确认弹窗 */}
      <Dialog open={confirmModal} onOpenChange={setConfirmModal}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>
              {`确定${handleType === 'del' ? '删除' : String(selectItem?.IsEnabled) === '0' ? '关闭' : '开启'}${selectedKeys?.length > 0 ? `选中的 ${selectedKeys?.length || 0}个 ` : '此'}策略吗？`}
            </DialogTitle>
          </DialogHeader>
          <div>
            <p className="text-sm text-[#525252] leading-relaxed">
              {handleType === 'del'
                ? '确认后，策略将被删除，无法恢复，策略范围内的资产将不再生效，请谨慎操作。'
                : String(selectItem?.IsEnabled) === '0'
                  ? `确认后，策略将被关闭，生效范围内的OpenClaw将不再${selectItem?.PolicyAction == 2
                    ? '进行拦截'
                    : selectItem?.PolicyAction == 1
                      ? '放行'
                      : '进行告警'}。`
                  : `确认后，策略将生效，生效范围内的OpenClaw将${selectItem?.PolicyAction == 2
                    ? '开启拦截'
                    : selectItem?.PolicyAction == 1
                      ? '开启放行'
                      : '开启告警'}，请谨慎操作。`}
            </p>
            {selectedKeys?.length > 0 && !selectItem?.PolicyId && selectedRows?.length > 0 && handleType !== 'del' && (
              <div className="mt-3">
                <p className="text-sm text-[#525252]">
                  您已选择
                  <span className="text-[#16A34A] mx-1">
                    {selectedRows?.length}个
                  </span>
                  策略，
                  <Button variant="link" className="px-1 h-auto align-baseline" onClick={() => setTableshow(!tableshow)}>
                    查看详情
                    {tableshow ? <ChevronUp className="inline w-3 h-3" /> : <ChevronDown className="inline w-3 h-3" />}
                  </Button>
                </p>
                <div className={tableshow ? 'py-2.5' : ''}>
                  <div className={tableshow ? 'block' : 'hidden'}>
                    <ScrollArea className="max-h-[360px]">
                      <Table>
                        <TableBody>
                          {(selectedRows || []).map((record: any, recordIndex: number) => (
                            <TableRow key={recordIndex}>
                              <TableCell className="w-[50px]">{recordIndex + 1}</TableCell>
                              <TableCell>{record?.PolicyName ?? '-'}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </ScrollArea>
                  </div>
                </div>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setConfirmModal(false)}>
              {'取消'}
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                setConfirmModal(false);
                if (handleType === 'del') {
                  handleDelPolicy();
                } else {
                  handleSwitchChange(selectItem);
                }
              }}
            >
              {'确定'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 恶意请求自动拦截设置弹窗 */}
      <Dialog open={autoBlockModalVisible} onOpenChange={setAutoBlockModalVisible}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>{'恶意请求自动拦截'}</DialogTitle>
          </DialogHeader>
          <div>
            <div className="flex items-center gap-3">
              <span className="text-sm text-[#737373]">{'拦截开关：'}</span>
              <Switch
                checked={modalAutoBlockSwitch}
                onCheckedChange={val => setModalAutoBlockSwitch(val)}
              />
            </div>
            <div className="my-4 p-4 bg-[#FAFAFA] border border-[#E5E5E5] rounded-[4px]">
              <strong className="text-sm text-[#0A0A0A]">{'拦截原理说明：'}</strong>
              <span className="text-sm text-[#525252]">
                {'恶意请求是终止进程对规则域名/ip的访问，不会杀掉进程，会终止这个访问请求。'}
              </span>
              <div className="mt-2.5 px-[3px] pb-[2px] text-[#e3eaef] bg-black font-mono text-xs leading-5">
                root@VM-0-17-ubuntu:/home/ubuntu# ping 14.119.104.189 <br />
                ping: 14.119.104.189: Non-recoverable failure in name resolution
              </div>
            </div>
            <div className="maliciousRequest-editPolicy">
              <div>
                <div className="label-txt mg-tp-6 w-[70px]">
                  {'防护模式：'}
                </div>
                <div className="content">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span>
                        <div className={`inline-flex items-center gap-1 p-1 bg-[#F5F5F5] rounded-[4px] ${!modalAutoBlockSwitch ? 'pointer-events-none opacity-50' : ''}`}>
                          {renderSegOptionsNew(false, modalProtectMode === '0').map(opt => {
                            const isActive = opt.value === modalProtectMode;
                            return (
                              <button
                                key={opt.value}
                                type="button"
                                onClick={() => setModalProtectMode(opt.value)}
                                className={`px-3 py-1 text-sm rounded-[3px] transition-colors ${
                                  isActive
                                    ? 'bg-white text-[#0A0A0A] font-medium'
                                    : 'text-[#737373] hover:text-[#0A0A0A] font-normal'
                                }`}
                                style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                              >
                                {opt.text}
                              </button>
                            );
                          })}
                        </div>
                      </span>
                    </TooltipTrigger>
                    {!modalAutoBlockSwitch && (
                      <TooltipContent>{'拦截开关未开启，暂无法进行模式切换'}</TooltipContent>
                    )}
                  </Tooltip>
                  {modalProtectMode === '0' ? (
                    <div className="mt-1.5 text-sm text-[#525252]">
                      {'仅针对高置信度的风险进行自动防护，更适合日常安全运营使用。'}
                      <Badge variant="secondary" className="ml-2">
                        {'推荐'}
                      </Badge>
                    </div>
                  ) : (
                    <div className="text-sm text-[#525252] mt-1">
                      <div>{'综合多个引擎检测结果，针对中、高置信度的风险进行自动拦截。'}</div>
                      <div>
                        <span className="text-[#DC2626]">{'可能存在误拦截风险'}</span>
                        {'，适合重保防护，请谨慎启用。'}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setAutoBlockModalVisible(false)}>
              {'取消'}
            </Button>
            <Button variant="dialog-confirm" size="claw-sm" onClick={() => handleAutoBlockChange(modalAutoBlockSwitch, modalProtectMode)}>
              {'确定'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
