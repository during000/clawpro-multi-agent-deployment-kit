import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Base64 } from '@/vendor/js-base64';
import { toast } from 'sonner';
import { Info, RefreshCw, ChevronUp, ChevronDown, CircleAlert, Search, Settings2, Bell } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { StatusTag } from '@/components/ui/status-tag';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { BodyText, BodyMedium, MetaText, CodeText } from '@/components/ui/Typography';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import {
  Popover,
  PopoverAnchor,
  PopoverContent,
} from '@/components/ui/popover';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import { TreeSelect } from '@/components/ui/tree-select';
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
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import { ScrollArea } from '@/components/ui/scroll-area';
import { SurfaceCard, SurfaceConfig } from '@/components/ui/Surface';
import { Pagination } from '@/components/ui/pagination';
import { ModifyBashPolicyStatus, DeleteBashPolicies, DescribeBashPolicies, ModifyBashPolicy } from '@/pages/admin/Security/api';

import { renderSegOptionsNew } from '../MaliciousPolicy/MaliciousPolicyList';
import { AUTHORIZE_ROUTE } from '../../constants';
import {
  fetchInitStorageShowColKeys,
  saveColKeys,
  getMaxRemoteStorage,
  setMaxRemoteStorage,
} from '../../Common/tablePanelColumnUtil';
import MultiTypeSelectMachine from '../../Common/MultiTypeSelectMachine';
import ExportCsv from '../../Common/ExportCsv';
import { Transfer } from '@/components/ui/transfer';
import { setCookie, getCookie } from '../../Common/cookieUtil';
import { getSelectionRows } from '../../Common/CommonRiskHandleFunc';

import { PolicyDetailDrawer } from './PolicyDetailDrawer';
import { EditPolicyDrawer } from './EditPolicyDrawer';
import {
  RulesAttributeMap,
  POLICY_TYPES,
  BASH_POLICY_LEVEL_ALL,
  ALL_POLICY_TYPES_DATA,
  GetHostTypeText,
  getPolicyActionMap,
  BASH_LEVEL_MAP,
  BLOCK_STANDARD_ID,
  BLOCK_DEEP_ID,
  SYSTEM_STANDARD_ID,
  BASH_DETAIL_TORULE,
  BASH_DETAIL_TOCREATE,
  getPolicyActionsData,
  POLICY_ACTION_THEME_MAP,
  hostVersionMap,
  LICENSE_TYPES_MAP,
  CSIP_AI_AGENT_BATCH_TIPS,
} from './Constants';

const PAGE = 'CSIP_BASH_POLICYLIST';

/** 每页条数可选列表 */
const PAGE_SIZE_OPTIONS = [10, 20, 30, 50, 100];

/** 列定义 */
interface ColDef {
  key: string;
  header: string;
  width?: number | string;
  fixed?: 'left' | 'right';
  stickyLeft?: number;
  stickyRight?: number;
  isAlwaysShow?: boolean;
  ellipsis?: boolean;
  columnEmptyValue?: string;
  render?: (item: any, rowKey: string, recordIndex: number) => React.ReactNode;
}

export const getRuleLevelText = (level: any) => {
  const text = BASH_LEVEL_MAP[level] || '无';
  // 规范：StatusTag mode="text" 纯彩色文字
  let variant: 'red' | 'orange' | 'green' | 'gray' = 'gray';
  if (String(level) === '1') variant = 'red';       // 高危
  else if (String(level) === '2') variant = 'orange'; // 中危
  else if (String(level) === '3') variant = 'green';  // 低危
  return <StatusTag variant={variant} mode="text">{text}</StatusTag>;
};

export const HIGH_CMD = [
  { text: '删除根目录', value: 'rm -rf /', cmd: 'rm -rf /$' },
  { text: '删除用户主目录', value: 'rm -rf ~/', cmd: 'rm -rf ~/$' },
  { text: '格式化磁盘', value: 'mkfs', cmd: 'mkfs' },
  { text: '原始磁盘写入', value: 'dd if=* of=/dev/sd*', cmd: 'dd if=.* of=/dev/sd.*' },
  {
    paramsText: '直接写入磁盘设备',
    text: '直接写入磁盘设备',
    value: (
      <>
        <span>&gt;</span> /dev/sd*
      </>
    ),
    cmd: '> /dev/sd.*',
  },
  {
    paramsText: '写入proc文件系统',
    text: '写入 /proc 文件系统',
    value: (
      <>
        echo * <span>&gt;</span> /proc/*
      </>
    ),
    cmd: 'echo .* > /proc/.*',
  },
  {
    paramsText: 'AI在任意目录启动http服务',
    text: 'AI在任意目录启动http服务',
    value: <span>python(3)?\s+-m\s+http\.server\s+</span>,
    cmd: 'python(3)?\\s+-m\\s+http\\.server\\s+',
  },
];

export function BashPolicyList({ hasFlagship, getInitPolicyCount, aiAgentHostList, isFromDetail, mockData }: any) {
  const isMock = !!(mockData && mockData.length);
  const [handleType, setHandleType] = useState('');
  const [selectItem, setSelectItem] = useState({} as any);
  const [confirmModal, setConfirmModal] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [settingVisible, setSettingVisible] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [selectedRows, setSelectedRows] = useState<any[]>([]);
  const [loadingIndex, setLoadingIndex] = useState(0);
  const [loadingList, setLoadingList] = useState<boolean[]>([]);
  const [tableshow, setTableshow] = useState(false);
  const [tableData, setTableData] = useState<any[]>([]);
  const [allData, setAllData] = useState<any[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [isLoading, setIsLoading] = useState(false);
  const [modalProtectMode, setModalProtectMode] = useState('0');
  const [autoBlockModalVisible, setAutoBlockModalVisible] = useState(false);
  const [modalAutoBlockSwitch, setModalAutoBlockSwitch] = useState(false);
  const [downloadParams, setDownloadParams] = useState({} as any);
  const [initColKeys, setInitColKeys] = useState<string[]>([]);
  const [changeModePopVisible, setChangeModePopVisible] = useState(false);
  const [standardBlockPolicy, setStandardBlockPolicy] = useState({} as any);
  const [importantBlockPolicy, setImportantBlockPolicy] = useState({} as any);
  const [closeAutoBlockModal, setCloseAutoBlockModal] = useState(false);
  const [showCloseBubble, setShowCloseBubble] = useState(true);
  const [batchAddPolicyModalVisible, setBatchAddPolicyModalVisible] = useState(false);
  const [selectMachine, setSelectMachine] = useState<string[]>([]);
  const [isShowBatchTips, setIsShowBatchTips] = useState(false);
  const [hasExpandBatchAddModal, setHasExpandBatchAddModal] = useState(false);
  const [columnSettingOpen, setColumnSettingOpen] = useState(false);

  // 搜索
  const [searchField, setSearchField] = useState<string>('Name');
  const [searchKey, setSearchKey] = useState('');

  // 筛选
  const [filterBashAction, setFilterBashAction] = useState<string>('');
  const [filterEnable, setFilterEnable] = useState<string>('');
  const [filterWhite, setFilterWhite] = useState<string>('');
  const [filterLevel, setFilterLevel] = useState<string[]>(BASH_POLICY_LEVEL_ALL);
  const [filterCategory, setFilterCategory] = useState<string>('');

  const requestIdRef = useRef(0);

  const clearSelected = useCallback(() => {
    setSelectedKeys([]);
    setSelectedRows([]);
  }, []);

  const getInitColKeys = async () => {
    const colKeys = await fetchInitStorageShowColKeys(PAGE);
    setInitColKeys(colKeys ?? []);
  };

  const batchCreatePolicy = async () => {
    setBatchAddPolicyModalVisible(false);
    const res: any = await Promise.all(
      HIGH_CMD.map(d =>
        ModifyBashPolicy({
          Policy: {
            Name: `拦截${d?.paramsText || d.text}命令`,
            Category: 1,
            Descript: `拦截命令${d.cmd}`,
            White: 0,
            BashAction: 2,
            Scope: 0,
            Enable: 1,
            Level: 1,
            Quuids: selectMachine?.filter?.(d => d)?.map?.(d => d),
            Rules: { Process: { Cmdline: Base64.encode(d?.cmd) } },
          } as any,
        }),
      ),
    );
    if (res) {
      setMaxRemoteStorage(CSIP_AI_AGENT_BATCH_TIPS, '1');
      setIsShowBatchTips(false);
      toast.success('操作成功');
    }
    refreshTable?.();
  };

  const getAutoBlockData = async () => {
    if (isMock) {
      // mock 模式：直接从 mockData 中读取系统拦截策略，并把 mockData 灌入表格
      const data = (mockData as any[]).filter(item => String(item?.BashAction) === '2' && String(item?.Category) === '0');
      setStandardBlockPolicy(data?.find?.((it: any) => it?.Id === BLOCK_STANDARD_ID) || {});
      setImportantBlockPolicy(data?.find?.((it: any) => it?.Id === BLOCK_DEEP_ID) || {});
      // 直接渲染 mock
      setAllData(mockData);
      setTotalCount(mockData.length);
      const offset = (page - 1) * pageSize;
      const currentData = mockData.slice(offset, offset + pageSize);
      setLoadingList(currentData.map(() => false));
      setTableData(currentData);
      setIsLoading(false);
      return;
    }
    const res: any = await DescribeBashPolicies({
      Offset: 0,
      Limit: 5,
      Filters: [{ Name: 'Category', Values: ['0'] }],
    });
    const data = res?.List?.filter?.((item: { BashAction: any; }) => String(item?.BashAction) === '2');
    setStandardBlockPolicy(data?.filter?.((item: { Id: number; }) => item?.Id === BLOCK_STANDARD_ID)?.[0] || {});
    setImportantBlockPolicy(data?.filter?.((item: { Id: number; }) => item?.Id === BLOCK_DEEP_ID)?.[0] || {});
    fetchTableData();
  };

  const handleSwitchChange = async (item: { Id: any; Enable: any; }, index = loadingIndex) => {
    setLoadingList(prev => [...prev.slice(0, index), true, ...prev.slice(index + 1)]);
    const res: any = await ModifyBashPolicyStatus({
      Id: item?.Id,
      Enable: String(item?.Enable) === '1' ? 0 : 1,
    });
    if (res) {
      toast.success('操作成功');
      clearSelected();
      if (tableData?.length === 1 && filterEnable && page > 1) {
        setPage(prev => Math.max(prev - 1, 1));
      } else {
        getAutoBlockData();
      }
    } else {
      setLoadingList(tableData?.map?.(() => false) ?? []);
    }
  };

  const handleDelPolicy = async (item: any = undefined) => {
    setDetailVisible(false);
    const handleSelectedKeys = selectedKeys.map(item => Number(item));
    const params = {
      Ids: item?.Id ? [Number(item?.Id)] : handleSelectedKeys,
    };
    const res: any = await DeleteBashPolicies(params);
    if (res) {
      toast.success('操作成功');
      setDetailVisible(false);
      if (params?.Ids?.length === tableData?.length && page > 1) {
        setPage(prev => Math.max(prev - 1, 1));
      } else {
        getAutoBlockData();
      }
      clearSelected();
    }
  };

  const handleAutoBlockChange = async (isOpen: boolean, mode: string) => {
    setAutoBlockModalVisible(false);
    setLoadingList(prev => [...prev.slice(0, loadingIndex), true, ...prev.slice(loadingIndex + 1)]);
    const firstCloseId = isOpen
      ? mode === '0'
        ? importantBlockPolicy?.Id
        : standardBlockPolicy?.Id
      : standardBlockPolicy?.Id;
    const res: any = await ModifyBashPolicyStatus({ Id: firstCloseId, Enable: 0 });
    const res1: any = await ModifyBashPolicyStatus({
      Id: firstCloseId === standardBlockPolicy?.Id ? importantBlockPolicy?.Id : standardBlockPolicy?.Id,
      Enable: isOpen ? 1 : 0,
    });
    if (res && res1) {
      toast.success('操作成功');
    }
    getAutoBlockData();
  };

  /** 数据请求 — 替代原 TablePanel 的 request */
  const fetchTableData = useCallback(async () => {
    // mock 模式：本地按筛选/搜索过滤
    if (isMock) {
      let list: any[] = [...(mockData as any[])];
      if (searchKey.trim() && searchField === 'Name') {
        list = list.filter(d => (d?.Name || '').includes(searchKey.trim()));
      }
      if (filterBashAction) list = list.filter(d => String(d?.BashAction) === filterBashAction);
      if (filterEnable) list = list.filter(d => String(d?.Enable) === filterEnable);
      if (filterWhite) list = list.filter(d => String(d?.White) === filterWhite);
      if (filterCategory) list = list.filter(d => String(d?.Category) === filterCategory);
      if (filterLevel?.length && filterLevel.join(',') !== BASH_POLICY_LEVEL_ALL.join(',')) {
        list = list.filter(d => filterLevel.includes(String(d?.Level)));
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

    setIsLoading(true);
    const reqId = ++requestIdRef.current;

    const filters: any[] = [];

    // 搜索
    if (searchKey.trim()) {
      if (searchField === 'Rule') {
        filters.push({ Name: 'Rule', Values: [Base64.encode(searchKey.trim())] });
      } else {
        filters.push({ Name: searchField, Values: [searchKey.trim()] });
      }
    }
    // 筛选
    if (filterBashAction) filters.push({ Name: 'BashAction', Values: [filterBashAction] });
    if (filterEnable) filters.push({ Name: 'Enable', Values: [filterEnable] });
    if (filterWhite) filters.push({ Name: 'White', Values: [filterWhite] });
    if (filterLevel?.length && filterLevel.join(',') !== BASH_POLICY_LEVEL_ALL.join(',')) {
      filters.push({ Name: 'Level', Values: filterLevel });
    }
    if (filterCategory) filters.push({ Name: 'Category', Values: [filterCategory] });

    const params: any = { Filters: filters };
    setDownloadParams({ Filters: filters });

    try {
      const res: any = await DescribeBashPolicies({ ...params, Limit: 100, Offset: 0 });
      if (reqId !== requestIdRef.current) return;

      let list = res?.List || [];
      if (res?.TotalCount > 100) {
        const num = Math.min(Math.ceil((res?.TotalCount - 100) / 100), 19);
        const resMore = await Promise.all(
          new Array(num)
            .fill(1)
            .map((_, index) => DescribeBashPolicies({ ...params, Offset: 100 * (index + 1), Limit: 100 })),
        );
        list = list.concat(resMore?.map?.((item: any) => item?.List ?? [])?.flat?.(3));
      }
      if (reqId !== requestIdRef.current) return;

      // 处理系统拦截策略排序
      const block = list?.filter?.((item: any) => String(item?.Category) === '0' && String(item?.BashAction) === '2');
      list = (!block?.length
        ? []
        : (filterEnable
          && ((filterEnable === '0'
            && [standardBlockPolicy, importantBlockPolicy]?.every?.(d => String(d?.Enable) === '0'))
            || (filterEnable === '1' && block?.length > 0)))
          || !filterEnable
          ? [importantBlockPolicy]
          : []
      )?.concat?.(list?.filter?.((item: any) => !(String(item?.Category) === '0' && String(item?.BashAction) === '2')));

      if (detailVisible && selectItem?.Id) {
        const found = list?.find?.((item: any) => item?.Id === selectItem?.Id);
        if (found) setSelectItem(found);
      }

      setAllData(list);
      setTotalCount(list?.length || 0);

      const offset = (page - 1) * pageSize;
      const currentData = list?.slice?.(offset, offset + pageSize) || [];
      setLoadingList(currentData.map(() => false));
      setTableData(currentData);
    } catch (err) {
      console.error(err);
    } finally {
      if (reqId === requestIdRef.current) {
        setIsLoading(false);
      }
    }
  }, [page, pageSize, searchKey, searchField, filterBashAction, filterEnable, filterWhite, filterLevel, filterCategory, standardBlockPolicy, importantBlockPolicy, detailVisible, selectItem?.Id, isMock, mockData]);

  const refreshTable = useCallback(() => {
    fetchTableData();
    getInitPolicyCount?.();
  }, [fetchTableData, getInitPolicyCount]);

  // 初始化
  useEffect(() => {
    getInitColKeys();
    getMaxRemoteStorage(CSIP_AI_AGENT_BATCH_TIPS, val => setIsShowBatchTips(val !== '1'));
    const createCookie = getCookie(BASH_DETAIL_TOCREATE);
    if (createCookie) {
      setCookie(BASH_DETAIL_TOCREATE, '', -1);
      setHandleType('create');
      setSelectItem({});
      setSettingVisible(true);
    }
    const cookieName = getCookie(BASH_DETAIL_TORULE);
    if (cookieName) {
      setSearchKey(cookieName?.split?.(',')?.[0] || '');
    }
  }, []);

  // 初次加载获取系统策略 + 列表
  useEffect(() => {
    getAutoBlockData();
  }, []);

  // 筛选/搜索/分页变化时重新加载
  useEffect(() => {
    fetchTableData();
  }, [fetchTableData]);

  // cookie 跳转到编辑
  useEffect(() => {
    const cookieName = getCookie(BASH_DETAIL_TORULE);
    if (cookieName && allData?.length) {
      const item = allData?.filter?.((item: any) => String(item?.Id) === cookieName?.split?.(',')?.[1]);
      if (item?.length === 1 && item?.[0]) {
        setSelectItem(item?.[0] ?? {});
        setHandleType('edit');
        setSettingVisible(true);
      }
      setCookie(BASH_DETAIL_TORULE, '', -1);
    }
  }, [allData]);

  useEffect(() => {
    if (autoBlockModalVisible) {
      const isOpen = String(standardBlockPolicy?.Enable) === '1' || String(importantBlockPolicy?.Enable) === '1';
      setModalAutoBlockSwitch(true);
      setModalProtectMode(isOpen ? String(importantBlockPolicy?.Enable) : '0');
    }
  }, [autoBlockModalVisible]);

  // 列定义
  const allColumns: ColDef[] = useMemo(() => [
    {
      key: 'Name',
      header: '策略名称',
      width: 240,
      // 与「组件表格」固定列规范一致：复选框 + 名称列同时左固定，
      // 由最右侧的名称列承载边界阴影（复选框列 fixedShadow={false}）。
      fixed: 'left',
      render: (item, _rowKey, recordIndex) =>
        String(item?.Category) === '0' ? (
          item?.Name
        ) : (
          <Button
            variant="link-dark"
            title={item?.Name}
            style={{
              maxWidth: '100%',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              display: 'inline-block',
              padding: 0
            }}
            onClick={() => {
              setSelectItem(item);
              setLoadingIndex(recordIndex);
              setDetailVisible(true);
            }}
          >
            {item?.Name}
          </Button>
        ),
    },
    {
      key: 'Category',
      header: '策略类型',
      width: 120,
      render: item =>
        item?.Category === 0 || item?.Category === 1 ? (
          <span className="inline-flex items-center text-[#0A0A0A]">
            {POLICY_TYPES[item?.Category] || '--'}
            {String(item?.Category) === '0' && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Info className="w-3.5 h-3.5 text-[#A3A3A3] hover:text-[#525252] ml-1 cursor-pointer" />
                </TooltipTrigger>
                <TooltipContent className="max-w-[320px]">
                  {'系统策略为腾讯OpenClaw运营专家与算法专家经过多模型沉淀的规则配置，适用于大部分的高危命令检测。'}
                </TooltipContent>
              </Tooltip>
            )}
          </span>
        ) : (
          '--'
        ),
    },
    {
      key: 'White',
      header: '黑/白名单',
      width: 110,
      render: item =>
        String(item?.White) === '1' ? (
          <span className="text-[#0A0A0A]">白名单</span>
        ) : (
          <span className="text-[#0A0A0A]">黑名单</span>
        ),
    },
    {
      key: 'Rule',
      header: '策略内容',
      width: 280,
      render: (item, _rowKey, recordIndex) => {
        const data = Object.keys(RulesAttributeMap).filter(
          key => item?.Rules?.[key]?.Exe || item?.Rules?.[key]?.Cmdline,
        );
        const content = item?.Rules?.[data?.[0]]?.Exe || item?.Rules?.[data?.[0]]?.Cmdline;
        return String(item?.Category) === '0' ? (
          '腾讯云恶意命令库'
        ) : data?.length > 0 ? (
          <span>
            {`${data?.map?.(key => RulesAttributeMap?.[key])?.join?.('、')}：${`${content?.slice?.(0, 20)}${content?.length > 20 ? '...' : ''}`}`}
            <span>
              {'等（'}
              <Button
                variant="link"
                style={{ margin: '-2px 2px 0', padding: 0, minWidth: 'auto', verticalAlign: 'middle' }}
                onClick={() => {
                  setSelectItem(item);
                  setLoadingIndex(recordIndex);
                  setDetailVisible(true);
                }}
              >
                {`${data?.length}个`}
              </Button>
              {'）'}
            </span>
          </span>
        ) : (
          '-'
        );
      },
    },
    {
      key: 'Level',
      header: '威胁等级',
      width: 100,
      render: item => (String(item?.Category) === '0' ? '-' : getRuleLevelText(item?.Level)),
    },
    {
      key: 'Scope',
      header: '生效OpenClaw',
      width: 130,
      render: (item, _rowKey, recordIndex) => (
        <div>
          {item?.Category === 0 ? (
            <span>{String(item?.Scope) !== '0' ? GetHostTypeText(item?.Scope) : item?.Quuids?.length || 0}</span>
          ) : (
            <Button
              variant="link"
              className="machineName-btn-textOverflow"
              title={String(item?.Scope) !== '0' ? GetHostTypeText(item?.Scope) : String(item?.Quuids?.length || 0)}
              onClick={() => {
                setSelectItem(item);
                setLoadingIndex(recordIndex);
                setDetailVisible(true);
              }}
            >
              <span>{String(item?.Scope) !== '0' ? GetHostTypeText(item?.Scope) : item?.Quuids?.length || 0}</span>
            </Button>
          )}
        </div>
      ),
    },
    {
      key: 'ModifyTime',
      header: '更新时间',
      width: 160,
    },
    {
      key: 'BashAction',
      header: '执行动作',
      width: 140,
      render: (item, _text, index) => {
        const actionText = getPolicyActionMap()?.[item?.BashAction] || '--';
        // 执行动作用 soft 模式标签
        let actionVariant: 'red' | 'orange' | 'green' | 'gray' = 'gray';
        if (String(item?.BashAction) === '2') actionVariant = 'red';       // 拦截
        else if (String(item?.BashAction) === '0') actionVariant = 'orange'; // 告警
        else if (String(item?.BashAction) === '1') actionVariant = 'green';  // 放行
        return (
        <div>
          <StatusTag variant={actionVariant} mode="soft">{actionText}</StatusTag>
          {String(item?.Category) === '0' && String(item?.BashAction) === '2' ? (
            (String(standardBlockPolicy?.Enable) === '0' && String(importantBlockPolicy?.Enable) === '0')
              || loadingList[index] ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="flex mt-1.5">
                    <div className="inline-flex items-center gap-0.5 p-0.5 bg-[#F5F5F5] rounded-[4px] pointer-events-none opacity-50 w-fit">
                      {renderSegOptionsNew(true, true, setShowCloseBubble).map(opt => {
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
                <TooltipContent className="max-w-[360px]">
                  {loadingList[index] ? null : !hasFlagship ? (
                    <span>
                      <span>{'高危命令自动拦截属于旗舰版功能，开启后将自动拦截检测出的系统高危命令，点击 '}</span>
                      <a style={{ color: '#1447E6' }} onClick={() => window.open(AUTHORIZE_ROUTE)}>
                        {'升级版本'}
                      </a>
                      <span>{'，一键开启拦截。'}</span>
                    </span>
                  ) : (
                    <span>
                      {'策略未开启，暂无法进行模式切换，可点击 '}
                      <a style={{ color: '#1447E6' }} onClick={() => setAutoBlockModalVisible(true)}>
                        {'开启策略'}
                      </a>
                    </span>
                  )}
                </TooltipContent>
              </Tooltip>
            ) : (
              <Popover open={changeModePopVisible} onOpenChange={setChangeModePopVisible}>
                <PopoverAnchor asChild>
                  <div className="flex mt-1.5">
                    <div className="inline-flex items-center gap-0.5 p-0.5 bg-[#F5F5F5] rounded-[4px] w-fit">
                      {renderSegOptionsNew(true, String(standardBlockPolicy?.Enable) === '1').map(opt => {
                        const isActive = opt.value === String(importantBlockPolicy?.Enable);
                        return (
                          <button
                            key={opt.value}
                            type="button"
                            onClick={() => {
                              if (opt.value !== String(importantBlockPolicy?.Enable)) {
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
                </PopoverAnchor>
                <PopoverContent className="w-80">
                  <div className="space-y-2">
                    <h4 className="font-medium">
                      {String(importantBlockPolicy?.Enable) === '1' ? (
                        '确认切换为标准模式？'
                      ) : (
                        <span>
                          <CircleAlert className="inline w-4 h-4 -mt-0.5 mr-1" />
                          {'确认切换为重保模式？'}
                        </span>
                      )}
                    </h4>
                    <div style={{ marginBottom: -10 }}>
                      {String(importantBlockPolicy?.Enable) === '1' ? (
                        '确认后，将切换为标准模式，综合多个引擎检测结果，仅针对高置信度的风险进行自动防护，更适合日常安全运营使用。'
                      ) : (
                        <span>
                          {'确认后，将切换为重保模式，综合多个引擎检测结果，针对高、中置信度的风险进行自动防护，'}
                          <span className="text-[#DC2626]">{'可能存在误拦截风险'}</span>
                          {'，适合重保防护，请谨慎启用。'}
                        </span>
                      )}
                    </div>
                    <div className="flex justify-end gap-2 pt-2">
                      <Button
                        variant="link"
                        onClick={() => {
                          setChangeModePopVisible(false);
                          setLoadingIndex(index);
                          handleAutoBlockChange?.(true, String(importantBlockPolicy?.Enable) === '1' ? '0' : '1');
                        }}
                      >
                        {'确定'}
                      </Button>
                      <Button variant="ghost" onClick={() => setChangeModePopVisible(false)}>
                        {'取消'}
                      </Button>
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
            )
          ) : null}
        </div>
        );
      },
    },
    {
      key: 'Enable',
      header: '开关',
      width: 80,
      render: (item, _rowKey, recordIndex) =>
        String(item?.Category) === '0'
          && String(item?.BashAction) !== '2'
          && String(item?.Id) === String(SYSTEM_STANDARD_ID) ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span><Switch disabled checked style={{ marginLeft: 5 }} /></span>
            </TooltipTrigger>
            <TooltipContent>{'系统默认告警策略默认生效，不支持关闭'}</TooltipContent>
          </Tooltip>
        ) : String(item?.BashAction) === '2' && String(item?.Category) === '0' ? (
          !hasFlagship ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Switch
                    disabled
                    style={{ marginLeft: 5 }}
                    checked={String(standardBlockPolicy?.Enable) === '1' || String(importantBlockPolicy?.Enable) === '1'}
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent className="max-w-[320px]">
                <span>
                  <span>{'高危命令自动拦截属于旗舰版功能，开启后将自动拦截检测出的系统高危命令，点击 '}</span>
                  <a style={{ color: '#1447E6' }} onClick={() => window.open(AUTHORIZE_ROUTE)}>
                    {'升级版本'}
                  </a>
                  <span>{'，一键开启拦截。'}</span>
                </span>
              </TooltipContent>
            </Tooltip>
          ) : (
            <Switch
              disabled={loadingList[recordIndex]}
              checked={String(standardBlockPolicy?.Enable) === '1' || String(importantBlockPolicy?.Enable) === '1'}
              style={{ marginLeft: 5 }}
              onCheckedChange={() => {
                if (String(standardBlockPolicy?.Enable) === '1' || String(importantBlockPolicy?.Enable) === '1') {
                  setCloseAutoBlockModal(true);
                } else {
                  setAutoBlockModalVisible(true);
                }
              }}
            />
          )
        ) : String(item?.BashAction) === '2' && !hasFlagship ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Switch disabled checked={String(item?.Enable) === '1'} style={{ marginLeft: 5 }} />
              </span>
            </TooltipTrigger>
            <TooltipContent className="max-w-[200px]">
              <span>
                <span>{'当前暂无旗舰版OpenClaw，无法设置拦截策略，可'}</span>
                <a style={{ color: '#1447E6' }} onClick={() => window.open(AUTHORIZE_ROUTE)}>
                  {'点击升级版本'}
                </a>
              </span>
            </TooltipContent>
          </Tooltip>
        ) : (
          <AlertDialog>
            <AlertDialogTrigger asChild><span>
              <Switch disabled={loadingList[recordIndex]} style={{ marginLeft: 5 }} checked={String(item?.Enable) === '1'} />
            </span></AlertDialogTrigger>
            <AlertDialogContent className="sm:max-w-[420px]">
              <AlertDialogHeader>
                <AlertDialogTitle>{`确定${String(item?.Enable) === '1' ? '关闭' : '开启'}此策略？`}</AlertDialogTitle>
                <AlertDialogDescription>
                  {String(item?.Enable) === '1'
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
        ),
    },
    {
      key: 'Action',
      header: '操作',
      width: 140,
      render: item => (
        <>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button
                  variant="link"
                  disabled={String(item?.Category) === '0' || (String(item?.BashAction) === '2' && !hasFlagship)}
                  onClick={() => {
                    setHandleType('edit');
                    setSelectItem(item);
                    setSettingVisible(true);
                  }}
                >
                  {'编辑'}
                </Button>
              </span>
            </TooltipTrigger>
            {(String(item?.Category) === '0' || (String(item?.BashAction) === '2' && !hasFlagship)) && (
              <TooltipContent className="max-w-[200px]">
                {String(item?.Category) === '0' ? (
                  '系统策略不支持编辑'
                ) : (
                  <span>
                    {'当前暂无旗舰版OpenClaw，无法设置拦截策略，可'}
                    <a onClick={() => window.open(AUTHORIZE_ROUTE)} style={{ textDecoration: 'underline' }}>
                      {'点击升级版本'}
                    </a>
                  </span>
                )}
              </TooltipContent>
            )}
          </Tooltip>
          {String(item?.Category) === '0' ? (
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
              <AlertDialogTrigger asChild><span>
                <Button variant="link">{'删除'}</Button>
              </span></AlertDialogTrigger>
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
    return allColumns.filter(col => col.isAlwaysShow || col.fixed === 'right' || initColKeys.includes(col.key));
  }, [allColumns, initColKeys]);

  // 行选择逻辑
  const selectableRows = useMemo(() => tableData.filter(d => String(d?.Category) === '1'), [tableData]);
  const allSelected = selectableRows.length > 0 && selectableRows.every(d => selectedKeys.includes(String(d?.Id)));
  const someSelected = !allSelected && selectableRows.some(d => selectedKeys.includes(String(d?.Id)));

  const toggleSelectAll = useCallback(() => {
    if (allSelected) {
      clearSelected();
    } else {
      const keys = selectableRows.map(d => String(d?.Id));
      const rows = getSelectionRows(selectedRows, keys, tableData, 'Id');
      setSelectedKeys(keys);
      setSelectedRows(rows);
    }
  }, [allSelected, selectableRows, selectedRows, tableData, clearSelected]);

  const toggleSelectRow = useCallback((item: any) => {
    const key = String(item?.Id);
    const isSelected = selectedKeys.includes(key);
    const newKeys = isSelected ? selectedKeys.filter(k => k !== key) : [...selectedKeys, key];
    const newRows = getSelectionRows(selectedRows, newKeys, tableData, 'Id');
    setSelectedKeys(newKeys);
    setSelectedRows(newRows);
  }, [selectedKeys, selectedRows, tableData]);

  return (
    <div className="w-full overflow-hidden">
      {isShowBatchTips && !isFromDetail && (
        <div
          className="mb-[18px] flex items-center justify-between gap-3 rounded-[4px] border border-[#F5C5C5] px-4 py-3 text-[var(--text-default)]"
          style={{
            background: "linear-gradient(90deg, #FFF 0%, #FFF1F0 100%)",
            boxShadow: "0 2px 0 0 rgba(187, 187, 187, 0.05)",
          }}
        >
          <div className="flex items-center gap-3 min-w-0">
            <span className="shrink-0 inline-flex items-center rounded-[4px] bg-[#E54545] px-2 py-[2px] text-xs font-medium text-white leading-tight">
              推荐开启策略
            </span>
            <MetaText as="span" tone="inherit" className="min-w-0 leading-relaxed">
              {'推荐您拦截 '}
              <span className="font-medium">{`${HIGH_CMD.length}条`}</span>
              {' 高风险命令，建议拦截'}
            </MetaText>
            <div className="flex items-center gap-2 min-w-0 flex-wrap">
              {HIGH_CMD.slice(0, 3).map((item, i) => (
                <span
                  key={i}
                  className="shrink-0 inline-flex items-center rounded-[4px] border border-[#FAD79A] bg-[#FFF7E6] px-2 py-[2px] text-xs text-[#AD6800] leading-tight"
                >
                  {item.value}
                </span>
              ))}
              {HIGH_CMD.length > 3 && (
                <span className="shrink-0 inline-flex items-center rounded-[4px] border border-[#FAD79A] bg-[#FFF7E6] px-2 py-[2px] text-xs text-[#AD6800] leading-tight">
                  {`+${HIGH_CMD.length - 3}条`}
                </span>
              )}
            </div>
          </div>
          <Button
            variant="link"
            className="shrink-0 !text-[var(--text-default)]"
            onClick={() => setBatchAddPolicyModalVisible(true)}
          >
            一键创建拦截策略 &gt;
          </Button>
        </div>
      )}
      {/* ===== 工具栏（主按钮统一右置） ===== */}
      <div className="flex items-center justify-between gap-2 flex-wrap mb-3">
        {/* 左侧：筛选 + 搜索 + 刷新
            停服态豁免：筛选下拉、关键字搜索、刷新按钮均属「查看类」操作，
            不改动后端数据，停服时保持可用。整组打 data-billing-exempt，
            让 overlay 的灰化 CSS 与点击拦截同时放行；若组件自身传入了 disabled，
            仍由原生 disabled 生效（延续既有禁用）。右侧「删除/创建策略」是写操作，
            不在此豁免范围内，继续沿用停服禁用。*/}
        <div className="flex items-center gap-2 flex-wrap" data-billing-exempt>
          <Select value={filterBashAction || 'undefined'} onValueChange={val => { setFilterBashAction(val === 'undefined' ? '' : val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="请选择执行动作" />
            </SelectTrigger>
            <SelectContent>
              {getPolicyActionsData().map((d: any) => (
                <SelectItem key={d.value} value={String(d.value)}>{d.text}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={filterEnable || '__all__'} onValueChange={val => { setFilterEnable(val === '__all__' ? '' : val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="请选择生效状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">全部生效状态</SelectItem>
              <SelectItem value="1">已生效</SelectItem>
              <SelectItem value="0">未生效</SelectItem>
            </SelectContent>
          </Select>
          <Select value={searchField} onValueChange={val => { setSearchField(val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[110px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="Name">策略名称</SelectItem>
              <SelectItem value="Rule">策略内容</SelectItem>
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
            onClick={() => getAutoBlockData()}
            className="w-8 h-8 flex items-center justify-center rounded-[4px] border border-[#EAEEF4] bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
            title="刷新表格"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
          </button>
        </div>
        {/* 右侧：主按钮组（删除 / 创建策略） */}
        <div className="flex items-center gap-2 flex-wrap">
          <Button
            variant="claw-outline"
            size="claw-sm"
            disabled={!selectedKeys?.length}
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
          {/* 列设置 */}
          {/* <DropdownMenu open={columnSettingOpen} onOpenChange={setColumnSettingOpen}>
            <DropdownMenuTrigger asChild>
              <button
                className="w-8 h-8 flex items-center justify-center rounded-[4px] border border-[#EAEEF4] bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
                title="自定义展示列"
              >
                <Settings2 className="w-3.5 h-3.5" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <div className="px-2 py-1.5 text-xs text-[#737373]">{`已勾选${initColKeys?.length || allColumns.filter(c => !c.isAlwaysShow && c.fixed !== 'right').length}个`}</div>
              <DropdownMenuSeparator />
              {allColumns.filter(c => !c.isAlwaysShow && c.fixed !== 'right').map(col => (
                <DropdownMenuItem
                  key={col.key}
                  className="text-xs"
                  onSelect={e => {
                    e.preventDefault();
                    const newKeys = initColKeys.includes(col.key)
                      ? initColKeys.filter(k => k !== col.key)
                      : [...initColKeys, col.key];
                    setInitColKeys(newKeys);
                    saveColKeys(PAGE, newKeys);
                  }}
                >
                  <Checkbox
                    checked={!initColKeys?.length || initColKeys.includes(col.key)}
                    className="mr-2"
                  />
                  {col.header}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
          <ExportCsv
            requestApi="ExportBashPolicies"
            params={downloadParams}
            onFinish={(data: any) => {
              window.location.href = data.DownloadUrl;
              return false;
            }}
          /> */}
        </div>
      </div>

      {/* ===== 表格 ===== */}
      <SurfaceCard className="relative overflow-hidden" style={{ maxWidth: '100%' }}>
        {isLoading && (
          <div className="absolute inset-0 bg-white/60 z-10 flex items-center justify-center">
            <RefreshCw className="w-6 h-6 animate-spin text-[#1447E6]" />
          </div>
        )}
        <Table scrollX={1400} variant="white" containerClassName="bg-white">
          <TableHeader>
            <TableRow>
              <TableHead fixed="left" fixedShadow={false} style={{ width: 56, minWidth: 56 }}>
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
                    style={{
                      ...(col.width ? { width: col.width, minWidth: col.width } : {}),
                      ...(col.stickyLeft != null ? { left: col.stickyLeft } : {}),
                    }}
                  >
                    {/* 列头筛选 - 黑/白名单：filter-icon + 即时单选（无搜索/无 footer） */}
                    {col.key === 'White' ? (
                      <TreeSelect
                        triggerVariant="filter-icon"
                        title={col.header as string}
                        commitMode="instant"
                        showSearch={false}
                        showFooter={false}
                        nodes={[
                          { id: '0', name: '黑名单' },
                          { id: '1', name: '白名单' },
                        ]}
                        value={filterWhite}
                        onChange={val => { setFilterWhite(val); setPage(1); }}
                        allLabel="全部"
                        panelWidth={140}
                      />
                    ) : col.key === 'Category' ? (
                      /* 列头筛选 - 策略类型：filter-icon + 即时单选（无搜索/无 footer） */
                      <TreeSelect
                        triggerVariant="filter-icon"
                        title={col.header as string}
                        commitMode="instant"
                        showSearch={false}
                        showFooter={false}
                        nodes={ALL_POLICY_TYPES_DATA
                          .filter((d: any) => d.value !== '')
                          .map((d: any) => ({ id: d.value, name: d.text }))}
                        value={filterCategory}
                        onChange={val => { setFilterCategory(val); setPage(1); }}
                        allLabel="全部策略类型"
                        panelWidth={180}
                      />
                    ) : (
                      col.header
                    )}
                  </TableHead>
                );
              })}
            </TableRow>
          </TableHeader>
          <TableBody>
            {tableData.length > 0 ? (
              tableData.map((item: any, idx: number) => {
                const rowId = String(item?.Id);
                const isSelected = selectedKeys.includes(rowId);
                const isSelectable = String(item?.Category) === '1';
                return (
                  <TableRow key={item?.Id || idx}>
                    <TableCell fixed="left" fixedShadow={false} style={{ width: 56, minWidth: 56 }}>
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
                          style={{
                            ...(col.stickyLeft != null ? { left: col.stickyLeft } : {}),
                            ...(col.fixed === 'left' && col.width ? { minWidth: col.width } : {}),
                          }}
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

      {settingVisible && (
        <EditPolicyDrawer
          visible={settingVisible}
          setVisible={setSettingVisible}
          type={handleType}
          selectItem={selectItem}
          refreshTable={getAutoBlockData}
          hasFlagship={hasFlagship}
          aiAgentHostList={aiAgentHostList}
        />
      )}

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

      <Dialog open={confirmModal} onOpenChange={setConfirmModal}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>
              {`确定${handleType === 'del' ? '删除' : String(selectItem?.Enable) === '1' ? '关闭' : '开启'}${selectedKeys?.length > 0 ? `选中的 ${selectedKeys?.length || 0}个 ` : '此'}策略吗？`}
            </DialogTitle>
          </DialogHeader>
          <div>
            <p className="text-[#525252] text-sm leading-relaxed">
              {handleType === 'del'
                ? '确认后，策略将被删除，无法恢复，策略范围内的资产将不再生效，请谨慎操作。'
                : String(selectItem?.Enable) === '1'
                  ? `确认后，策略将被关闭，生效范围内的OpenClaw将不再${selectItem?.BashAction == 2
                    ? '进行拦截'
                    : selectItem?.BashAction == 1
                      ? '放行'
                      : '进行告警'}。`
                  : `确认后，策略将生效，生效范围内的OpenClaw将${selectItem?.BashAction == 2
                    ? '开启拦截'
                    : selectItem?.BashAction == 1
                      ? '开启放行'
                      : '开启告警'}，请谨慎操作。`}
            </p>
            {selectedKeys?.length > 0 && !selectItem?.Id && selectedRows?.length > 0 && handleType !== 'del' && (
              <div className="mt-3">
                <p className="text-sm text-[#525252]">
                  您已选择
                  <span className="text-[#16A34A] mx-1">
                    {selectedRows?.length}个
                  </span>
                  策略，
                  <Button variant="link" onClick={() => setTableshow(!tableshow)} className="px-1 h-auto align-baseline">
                    查看详情
                    {tableshow ? <ChevronUp className="inline w-3 h-3" /> : <ChevronDown className="inline w-3 h-3" />}
                  </Button>
                </p>
                <div style={{ padding: tableshow ? '10px 0' : '' }}>
                  <div style={{ display: tableshow ? 'block' : 'none' }}>
                    <ScrollArea className="max-h-[360px]">
                      <Table>
                        <TableBody>
                          {(selectedRows || []).map((record: any, recordIndex: number) => (
                            <TableRow key={recordIndex}>
                              <TableCell className="w-[50px]">{recordIndex + 1}</TableCell>
                              <TableCell>{`ID:${record?.Id ?? '-'} - 策略名称:${record?.Name ?? '-'}`}</TableCell>
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
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={() => {
                setConfirmModal(false);
              }}
            >
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

      <Dialog open={closeAutoBlockModal} onOpenChange={setCloseAutoBlockModal}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>{'确认关闭自动拦截？'}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-[#525252] leading-relaxed">
            {'关闭该功能后，OpenClaw将不再自动拦截检测到的高危命令进程，可能造成被入侵风险，请谨慎操作。'}
          </p>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setCloseAutoBlockModal(false)}>
              {'取消'}
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                handleAutoBlockChange?.(false, '0');
                setCloseAutoBlockModal(false);
              }}
            >
              {'确定'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={autoBlockModalVisible} onOpenChange={setAutoBlockModalVisible}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>{'高危命令自动拦截'}</DialogTitle>
          </DialogHeader>
          <div>
            <div className="flex items-center gap-3">
              <span className="text-sm text-[#737373]">{'拦截开关：'}</span>
              <Switch
                checked={modalAutoBlockSwitch}
                onCheckedChange={val => setModalAutoBlockSwitch(val)}
              />
            </div>
            <SurfaceConfig className="p-4 my-4">
              <strong className="text-sm text-[#0A0A0A]">{'拦截原理说明：'}</strong>
              <span className="text-sm text-[#525252]">
                {'高危命令自动拦截采用杀命中规则的进程的方式，比如A创建/bin/bash -i进程（bash -i被加黑），这个时候创建的/bin/bash进程会被杀掉（或者创建失败），A进程不影响。'}
              </span>
              <div className="mt-2.5 px-1 pb-0.5 text-[#e3eaef] bg-black font-mono text-xs">
                root@VM-0-17-ubuntu:/home/ubuntu# ping 14.119.104.189 <br />
                Killed
              </div>
            </SurfaceConfig>
            <div className="flex items-start gap-3">
              <div className="w-[80px] flex-shrink-0 text-sm leading-[1.5] text-[#737373] pt-[6px]">
                {'防护模式：'}
              </div>
              <div className="flex-1 min-w-0">
                <Tooltip>
                  <TooltipTrigger asChild>
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
                  </TooltipTrigger>
                  {!modalAutoBlockSwitch && (
                    <TooltipContent>{'拦截开关未开启，暂无法进行模式切换'}</TooltipContent>
                  )}
                </Tooltip>
                {modalProtectMode === '0' ? (
                  <div className="mt-1.5 text-sm leading-[1.5] text-[#525252]">
                    {'仅针对高置信度的风险进行自动防护，更适合日常安全运营使用。'}
                    <Badge variant="secondary" className="ml-1 -mt-0.5">
                      {'推荐'}
                    </Badge>
                  </div>
                ) : (
                  <div className="mt-1.5 text-sm leading-[1.5] text-[#525252]">
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

      <Dialog open={batchAddPolicyModalVisible} onOpenChange={setBatchAddPolicyModalVisible}>
        <DialogContent className="sm:max-w-[920px] max-h-[min(820px,calc(100vh-80px))] flex flex-col">
          <DialogHeader>
            <DialogTitle>{'确认添加推荐策略？'}</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <BodyText tone="secondary" className="block mb-5">
              {'确认后，将一键为您添加下述拦截策略，智能拦截AI Agent场景下的恶意命令，保护您的AI Agent资产安全。'}
            </BodyText>

            {/* 开启策略 */}
            <div className="flex items-start gap-4 pb-5 border-b border-gray-100">
              <MetaText className="w-[120px] shrink-0 pt-0.5">开启策略</MetaText>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <BodyMedium>{`${HIGH_CMD.length}条`}</BodyMedium>
                  <Button
                    variant="link"
                    size="sm"
                    className="h-auto p-0"
                    onClick={() => setHasExpandBatchAddModal(!hasExpandBatchAddModal)}
                  >
                    {hasExpandBatchAddModal ? '收起' : '展开'}
                    {hasExpandBatchAddModal ? <ChevronUp className="inline w-3 h-3 ml-0.5" /> : <ChevronDown className="inline w-3 h-3 ml-0.5" />}
                  </Button>
                </div>
                {hasExpandBatchAddModal && (
                  <div className="mt-3 rounded-[8px] border border-gray-200 overflow-hidden">
                    <Table density="compact">
                      <TableHeader>
                        <TableRow>
                          <TableHead style={{ width: '32%' }}>{'策略名称'}</TableHead>
                          <TableHead style={{ width: '40%' }}>{'策略内容'}</TableHead>
                          <TableHead style={{ width: 100 }}>{'拦截动作'}</TableHead>
                          <TableHead style={{ width: 100 }}>{'威胁等级'}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {HIGH_CMD.map((item, i) => (
                          <TableRow key={i}>
                            <TableCell>{'拦截'}{item?.paramsText || item.text}{'命令'}</TableCell>
                            <TableCell>
                              <MetaText>{'进程：'}</MetaText>
                              <CodeText tone="danger">{item.cmd}</CodeText>
                            </TableCell>
                            <TableCell>
                              <StatusTag variant="red" mode="soft">拦截</StatusTag>
                            </TableCell>
                            <TableCell>{getRuleLevelText(1)}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </div>
            </div>

            {/* 选择AI Agent资产 */}
            <div className="flex items-start gap-4 pt-5">
              <MetaText className="w-[120px] shrink-0 pt-0.5">选择AI Agent资产</MetaText>
              <div className="flex-1 min-w-0">
                <Transfer<any>
                  dataSource={(aiAgentHostList ?? []).map((h: any) => ({
                    ...h,
                    key: h?.Quuid,
                  }))}
                  rowKey="key"
                  targetKeys={selectMachine}
                  onChange={(nextKeys) => setSelectMachine(nextKeys)}
                  showSearch
                  searchPlaceholder={['搜索资产名称 / ID / IP', '搜索已选资产']}
                  pagination={{ pageSize: 8 }}
                  height={300}
                  titles={['全部 AI Agent 资产', '已选 AI Agent 资产']}
                  isItemDisabled={(h: any) => h?.ProtectType !== 'Flagship'}
                  renderDisabledTrigger={(_h: any, defaultCheckbox) => (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex">{defaultCheckbox}</span>
                      </TooltipTrigger>
                      <TooltipContent>基础版资产请升级到旗舰版以使用该能力</TooltipContent>
                    </Tooltip>
                  )}
                  filterOption={(input, h: any) => {
                    const needle = input.toLowerCase();
                    return [h?.OpenClawName, h?.MachineName, h?.InstanceID, h?.MachineIp]
                      .filter((v) => typeof v === 'string')
                      .some((v) => v.toLowerCase().includes(needle));
                  }}
                  columns={[
                    {
                      key: 'name',
                      header: 'Agent 名称 / ID',
                      render: (h: any) => (
                        <div className="min-w-0">
                          <div className="truncate text-[var(--text-emphasis)]">
                            {h?.OpenClawName || h?.MachineName || '-'}
                          </div>
                          <MetaText className="block truncate">
                            {h?.InstanceID || '-'}
                          </MetaText>
                        </div>
                      ),
                    },
                    {
                      key: 'version',
                      header: '防护版本',
                      width: 100,
                      render: (h: any) => hostVersionMap[h?.ProtectType] ?? '-',
                    },
                    {
                      key: 'ip',
                      header: '内网IP',
                      width: 140,
                      render: (h: any) => h?.MachineIp || '-',
                    },
                  ]}
                />
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setBatchAddPolicyModalVisible(false)}>
              {'取消'}
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                if (!selectMachine?.length) {
                  toast.error('请至少选择一台OpenClaw');
                  return;
                }
                batchCreatePolicy();
              }}
            >
              {'添加策略'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
