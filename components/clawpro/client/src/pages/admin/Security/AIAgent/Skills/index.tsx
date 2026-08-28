
import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { RefreshCw, Copy, Loader2, Info, Search, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
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
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  TableActionCell,
} from '@/components/ui/table';
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyDescription,
} from '@/components/ui/empty';
import { SurfaceCard } from '@/components/ui/Surface';
import { Pagination } from '@/components/ui/pagination';
import { DescribeMalWareList, DescribeSkillInfo } from '@/pages/admin/Security/api';

import {
  RISK_TYPE_SKILL,
  SKILL_LEVEL_DATA,
  SKILL_LEVEL_MAP,
  SKILL_STATUS_VAL_MAP,
  SKILL_BATCH_TITLE_MAP,
  SKILL_STATUS_DATA,
} from '../constants';
import { getSelectionRows, modifyEventsStatus, getBatchStatus } from '../Common/CommonRiskHandleFunc';
import SyncAssetBtn from '../Assets/SyncAssetBtn';
import BatchOperatorDialog from '../Alarms/BatchOperatorDialog';

import SkillDetailDrawer from './SkillDetailDrawer';
import { renderAgentItem } from '../Assets/AgentAssetsList';

/** 文本溢出省略 + tooltip */
const OverflowText = ({
  children,
  tooltip,
  className = '',
  onClick,
  copyable,
}: {
  children: React.ReactNode;
  tooltip?: React.ReactNode;
  className?: string;
  onClick?: () => void;
  copyable?: boolean;
}) => {
  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (typeof children === 'string') {
      navigator.clipboard.writeText(children);
    }
  };

  const content = (
    <span
      className={`block truncate max-w-full ${onClick ? 'text-[var(--text-title)] cursor-pointer underline underline-offset-4 decoration-gray-300 hover:decoration-gray-950' : 'text-[#525252]'} ${className}`}
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

export const getSkillLevelText = (level: string) => {
  // 规范 §16 StatusTag mode="text"：威胁等级使用无底色彩色文字
  const text = SKILL_LEVEL_MAP[level] || '无';
  let color = '#525252'; // 默认/无 - 灰
  if (level === '4' || level === '3') color = '#DC2626'; // 严重 / 高危 - 红
  else if (level === '2') color = '#D97706'; // 中危 - 橙
  else if (level === '1' || level === '0') color = '#16A34A'; // 低危 / 提示 - 绿
  return (
    <span className="text-sm font-medium" style={{ color }}>
      {text}
    </span>
  );
};

export const renderMalwareStatus = (status: number | string) => {
  // 规范 §8.2.2 状态徽章
  const statusStr = String(status);
  const statusObj = SKILL_STATUS_VAL_MAP[statusStr];
  const text = statusObj?.text || '未知';
  if (statusObj?.icon === 'loading') {
    return (
      <span className="badge-loading">
        <Loader2 className="w-3 h-3 animate-spin" />
        {text}
      </span>
    );
  }
  switch (statusObj?.theme) {
    case 'error':
      return <span className="badge-stopped">{text}</span>;
    case 'success':
      return <span className="badge-running">{text}</span>;
    case 'warning':
      return <span className="badge-pending">{text}</span>;
    default:
      return <span className="badge-shutdown">{text}</span>;
  }
};

export default function SkillsList({
  isGetAllMachinesLoading,
  aiAgentHostList,
  rencentScanTime,
  getAllMachines = undefined,
  InstanceId = undefined,
  getAllMalwareCount = undefined,
  openAssetDetail = undefined,
  // 仅设计走查/演示用：传入后跳过真实请求，直接展示 mock 数据
  mockData = undefined,
  mockTags = undefined,
}: any) {
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [selectedRows, setSelectedRows] = useState<any[]>([]);
  const [allData, setAllData] = useState<any[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [batchType, setBatchType] = useState<keyof typeof SKILL_BATCH_TITLE_MAP | ''>('');
  const [batchTimer, setBatchTimer] = useState(0);
  const [isBatchLoading, setIsBatchLoading] = useState(false);
  const [batchHandleModalVisible, setBatchHandleModalVisible] = useState(false);
  const [showBanner, setShowBanner] = useState(true);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailRecord, setDetailRecord] = useState<any>(null);
  const [confirmType, setConfirmType] = useState('');
  const [confirmItem, setConfirmItem] = useState<any>(null);
  const [killProcess, setKillProcess] = useState(true);
  const [curListTags, setCurListTags] = useState<Record<string, string[]>>({});
  const [isLoading, setIsLoading] = useState(false);

  // 分页与搜索状态
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState('4');
  const [levelFilter, setLevelFilter] = useState('');

  const [reloadCount, setReloadCount] = useState(0);

  const refreshTable = useCallback(() => {
    setPage(1);
    setReloadCount(c => c + 1);
    getAllMalwareCount?.();
  }, [getAllMalwareCount]);

  useEffect(() => {
    if (!isGetAllMachinesLoading) {
      refreshTable();
    }
  }, [isGetAllMachinesLoading]);

  useEffect(() => {
    getBatchStatus(RISK_TYPE_SKILL, setIsBatchLoading, refreshTable, setBatchTimer);
  }, []);

  useEffect(() => {
    return () => {
      window.clearInterval(batchTimer);
    };
  }, [batchTimer]);

  const clearSelected = useCallback(() => {
    setSelectedRows([]);
    setSelectedKeys([]);
  }, []);

  const fetchListTags = async (ids: number[] = []) => {
    try {
      const resp: any = await DescribeSkillInfo({ Ids: ids });
      const tags: Record<string, string[]> = {};
      resp?.SkillInfoList?.forEach((a: any) => {
        tags[a?.Id] = a?.Tags || [];
      });
      setCurListTags(tags);
    } catch (e) {
      // ignore
    }
  };

  // 数据请求
  useEffect(() => {
    // === Mock 分支：直接使用注入的 mock 数据，跳过真实请求 ===
    if (mockData !== undefined) {
      let list = mockData || [];
      if (statusFilter && statusFilter !== 'all') {
        list = list.filter((d: any) => String(d?.Status) === String(statusFilter));
      }
      if (levelFilter && levelFilter !== 'all') {
        list = list.filter((d: any) => String(d?.Level) === String(levelFilter));
      }
      if (searchText.trim()) {
        const kw = searchText.trim();
        list = list.filter((d: any) => (d?.VirusName || '').indexOf(kw) >= 0);
      }
      setAllData(list.slice((page - 1) * pageSize, page * pageSize));
      setTotalCount(list.length);
      setCurListTags(mockTags || {});
      setIsLoading(false);
      return;
    }

    if (isGetAllMachinesLoading || aiAgentHostList?.length === 0) {
      setAllData([]);
      setTotalCount(0);
      return;
    }

    const fetchData = async () => {
      setIsLoading(true);
      try {
        const filters: any[] = [{ Name: 'VirusType', Values: ['AgentSkill'] }];

        if (statusFilter && statusFilter !== 'all') {
          filters.push({ Name: 'Status', Values: [statusFilter] });
        }
        if (levelFilter && levelFilter !== 'all') {
          filters.push({ Name: 'Level', Values: [levelFilter] });
        }
        if (searchText.trim()) {
          filters.push({ Name: 'VirusName', Values: [searchText.trim()] });
        }

        if (InstanceId) {
          filters.push({ Name: 'InstanceID', Values: [InstanceId] });
        } else {
          filters.push({
            Name: 'InstanceID',
            Values: aiAgentHostList.map((a: any) => a?.InstanceID).filter(Boolean),
          });
        }

        const params = {
          Offset: (page - 1) * pageSize,
          Limit: pageSize,
          Filters: filters,
        };

        const resp: any = await DescribeMalWareList(params);
        const list = resp?.MalWareList || [];
        setAllData(list);
        setTotalCount(resp?.TotalCount ?? 0);
        if (list.length) {
          fetchListTags(list.map((a: any) => a?.Id));
        }
      } catch (error) {
        setAllData([]);
        setTotalCount(0);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [isGetAllMachinesLoading, aiAgentHostList, page, pageSize, statusFilter, levelFilter, searchText, InstanceId, reloadCount, mockData, mockTags]);

  // 批量轮询
  useEffect(() => {
    if (!isBatchLoading) return;
    const timer = setInterval(() => {
      refreshTable();
    }, 3000);
    return () => clearInterval(timer);
  }, [isBatchLoading]);

  const handleSeparate = (item: any) => {
    modifyEventsStatus(RISK_TYPE_SKILL, 'separate', item?.Id, () => {
      refreshTable();
      clearSelected();
    });
  };

  const handleRecover = (item: any) => {
    modifyEventsStatus(RISK_TYPE_SKILL, 'recover', item?.Id, () => {
      refreshTable();
      clearSelected();
    });
  };

  const openConfirm = (type: string, item: any) => {
    setConfirmType(type);
    setConfirmItem(item);
  };

  const handleConfirmOk = () => {
    if (!confirmItem || !confirmType) return;
    const callback = () => {
      refreshTable();
      clearSelected();
    };
    if (confirmType === 'separate') {
      handleSeparate(confirmItem);
    } else if (confirmType === 'recover') {
      handleRecover(confirmItem);
    } else {
      modifyEventsStatus(RISK_TYPE_SKILL, confirmType, confirmItem?.Id, callback);
    }
    setConfirmType('');
    setConfirmItem(null);
  };

  const confirmModalConfig: Record<string, { title: string; message: React.ReactNode }> = {
    separate: {
      title: '确认将此告警隔离？',
      message: (
        <div className="space-y-3">
          <p className="text-sm text-[#525252] leading-relaxed">
            隔离此病毒文件，让黑客无法再次启动它，便于您定位病毒文件位置，对其进行查杀。（注意：windows系统下，若该文件正在运行中，会导致隔离失败）
          </p>
          <div className="flex items-center gap-2">
            <Checkbox
              id="kill-process"
              checked={killProcess}
              onCheckedChange={(val) => setKillProcess(!!val)}
            />
            <label htmlFor="kill-process" className="text-sm text-[#525252] cursor-pointer">
              隔离并杀掉该文件相关进程，建议勾选。
            </label>
          </div>
        </div>
      ),
    },
    recover: {
      title: '确认恢复隔离？',
      message: <p className="text-sm text-[#525252] leading-relaxed">确认恢复隔离该文件？恢复后文件将可以正常访问。</p>,
    },
    trust: {
      title: '确认将此告警标记为已忽略？',
      message: (
        <p className="text-sm text-[#525252] leading-relaxed">
          确认后，此告警的处理状态将变更为已忽略，该资产当天若再命中该告警策略，则不告警，处置状态仍为&quot;已忽略&quot;；第二天若触发将重新告警（新增一条）
        </p>
      ),
    },
    mark: {
      title: '确认将此告警标记为已处置？',
      message: (
        <p className="text-sm text-[#525252] leading-relaxed">
          确认后，此告警的处理状态将变更为已处置，该资产当天若再命中该告警策略，则不告警，处置状态仍为&quot;已处置&quot;；第二天若触发将重新告警（新增一条）
        </p>
      ),
    },
    del: {
      title: '确认删除此告警？',
      message: (
        <p className="text-sm text-[#525252] leading-relaxed">
          删除该告警记录，控制台将不再显示，无法恢复记录，请慎重操作。
        </p>
      ),
    },
  };

  const renderConfirmModal = () => {
    const config = confirmModalConfig[confirmType];
    if (!config) return null;
    return (
      <Dialog open onOpenChange={() => setConfirmType('')}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>{config.title}</DialogTitle>
          </DialogHeader>
          <div className="py-2 text-sm text-[#525252] leading-relaxed">{config.message}</div>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setConfirmType('')}>
              取消
            </Button>
            <Button variant="dialog-confirm" size="claw-sm" onClick={handleConfirmOk}>
              确定
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  };

  const renderOperateColumn = (item: any) => {
    const status = String(item?.Status);

    if (status === '6') {
      return (
        <div className="flex items-center gap-2 whitespace-nowrap">
          <Button variant="link" className="px-0 h-auto text-sm shrink-0 no-underline hover:no-underline" onClick={() => openConfirm('recover', item)}>
            恢复隔离
          </Button>
          <Button variant="link" className="px-0 h-auto text-sm shrink-0 no-underline hover:no-underline" onClick={() => openConfirm('del', item)}>
            删除记录
          </Button>
        </div>
      );
    }

    if (status === '4') {
      return (
        <div className="flex items-center gap-2 whitespace-nowrap">
          <Button variant="link" className="px-0 h-auto text-sm shrink-0 no-underline hover:no-underline" onClick={() => openConfirm('separate', item)}>
            隔离文件
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="link" className="px-0 h-auto text-sm shrink-0 no-underline hover:no-underline">
                更多
                <ChevronDown className="w-3 h-3 ml-0" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => openConfirm('mark', item)}>
                标记处置
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => openConfirm('trust', item)}>
                标记忽略
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => openConfirm('del', item)}>
                删除记录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      );
    }

    const isProcessing = status === '10' || status === '11';
    return (
      <div className="whitespace-nowrap">
        <Button variant="link" className="px-0 h-auto text-sm no-underline hover:no-underline" disabled={isProcessing} onClick={() => openConfirm('del', item)}>
          删除记录
        </Button>
      </div>
    );
  };

  // 全选/反选
  const isAllSelected = allData.length > 0 && allData.every((item: any) => selectedKeys.includes(String(item?.Id)));
  const isSomeSelected = allData.some((item: any) => selectedKeys.includes(String(item?.Id)));

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      const newKeys = Array.from(new Set([...selectedKeys, ...allData.map((item: any) => String(item?.Id))]));
      const newRows = getSelectionRows(selectedRows, newKeys, allData);
      setSelectedKeys(newKeys);
      setSelectedRows(newRows);
    } else {
      const currentIds = allData.map((item: any) => String(item?.Id));
      const newKeys = selectedKeys.filter(k => !currentIds.includes(k));
      setSelectedKeys(newKeys);
      setSelectedRows(selectedRows.filter((r: any) => !currentIds.includes(String(r?.Id))));
    }
  };

  const handleSelectRow = (item: any, checked: boolean) => {
    const id = String(item?.Id);
    let newKeys: string[];
    if (checked) {
      newKeys = [...selectedKeys, id];
    } else {
      newKeys = selectedKeys.filter(k => k !== id);
    }
    const newRows = getSelectionRows(selectedRows, newKeys, allData);
    setSelectedKeys(newKeys);
    setSelectedRows(newRows);
  };

  // 列定义
  const columns = useMemo(() => [
    {
      key: 'checkbox',
      header: (
        <Checkbox
          checked={isAllSelected}
          // @ts-ignore
          indeterminate={isSomeSelected && !isAllSelected}
          onCheckedChange={handleSelectAll}
        />
      ),
      width: 40,
      fixed: 'left' as const,
      render: (item: any) => (
        <Checkbox
          checked={selectedKeys.includes(String(item?.Id))}
          onCheckedChange={(checked) => handleSelectRow(item, !!checked)}
        />
      ),
    },
    {
      key: 'Name',
      header: '告警名称',
      minWidth: 160,
      render: (item: any) => (
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className="block truncate text-[var(--text-title)] cursor-pointer underline underline-offset-4 decoration-gray-300 hover:decoration-gray-950 text-sm"
              onClick={() => {
                setDetailRecord(item);
                setDetailVisible(true);
              }}
            >
              {item?.VirusName || '-'}
            </span>
          </TooltipTrigger>
          <TooltipContent>{item?.VirusName || '-'}</TooltipContent>
        </Tooltip>
      ),
    },
    {
      key: 'AgentName',
      header: 'AI Agent/调用模型',
      width: 150,
      render: (item: any) =>
        renderAgentItem(
          aiAgentHostList?.find?.((a: any) => a?.InstanceID === item?.MachineExtraInfo?.InstanceID),
          InstanceId ? undefined : () => openAssetDetail?.(aiAgentHostList?.find?.((a: any) => a?.InstanceID === item?.MachineExtraInfo?.InstanceID)),
        ),
    },
    {
      key: 'Level',
      header: '威胁等级',
      minWidth: 80,
      render: (item: any) => getSkillLevelText(`${item?.Level}`),
    },
    {
      key: 'Tags',
      header: '告警特征',
      minWidth: 180,
      render: (item: any) => {
        const tags = curListTags?.[item?.Id];
        if (!tags || tags.length === 0) return <span className="text-[#A3A3A3]">-</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {tags.slice(0, 2).map((tag: string) => (
              <span key={tag} className="badge-shutdown">{tag}</span>
            ))}
            {tags.length > 2 && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="badge-shutdown cursor-help">+{tags.length - 2}</span>
                </TooltipTrigger>
                <TooltipContent>{tags.slice(2).join('、')}</TooltipContent>
              </Tooltip>
            )}
          </div>
        );
      },
    },
    {
      key: 'FilePath',
      header: 'Skills路径',
      minWidth: 200,
      render: (item: any) => (
        <OverflowText tooltip={item?.FilePath || '-'}>
          {item?.FilePath || '-'}
        </OverflowText>
      ),
    },
    // {
    //   key: 'InstanceID',
    //   header: '资产ID/名称',
    //   minWidth: 140,
    //   render: (item: any) => (
    //     <div>
    //       <OverflowText
    //         copyable
    //         tooltip={item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId}
    //       >
    //         {item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId || '-'}
    //       </OverflowText>
    //       <div>
    //         <OverflowText copyable tooltip={item?.Alias}>
    //           {item?.Alias || '-'}
    //         </OverflowText>
    //       </div>
    //     </div>
    //   ),
    // },
    {
      key: 'DetectTime',
      header: '首次/最近检测时间',
      minWidth: 160,
      render: (item: any) => (
        <div className="text-xs leading-[18px]">
          <div className="text-[#525252]">{item?.CreateTime || item?.FileCreateTime || '-'}</div>
          <div className="text-[#A3A3A3]">{item?.LatestScanTime || '-'}</div>
        </div>
      ),
    },
    {
      key: 'Status',
      header: '处理状态',
      minWidth: 90,
      render: (item: any) => renderMalwareStatus(item?.Status),
    },
    {
      key: 'Operate',
      header: '操作',
      minWidth: 180,
      fixed: 'right' as const,
      render: (item: any) => renderOperateColumn(item),
    },
  ], [allData, selectedKeys, selectedRows, curListTags, killProcess]);

  return (
    <div style={{ position: 'relative' }}>
      {/* 批量操作中提示 */}
      {isBatchLoading && (
        <div className="flex items-center gap-2 mb-3 px-3 py-2 bg-[#EFF6FF] text-[#1447E6] rounded-[4px] text-sm">
          <Loader2 className="w-4 h-4 animate-spin" />
          <span>正在进行批量操作中...请稍候...</span>
        </div>
      )}

      {/* 操作栏（主按钮统一右置） */}
      <div className="flex items-center justify-between gap-2 mb-4">
        {/* 左侧：筛选 + 搜索 + 刷新 */}
        <div className="flex items-center gap-2">
          {/* 状态筛选 */}
          <Select value={statusFilter} onValueChange={(val) => { setStatusFilter(val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="处理状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              {SKILL_STATUS_DATA.map((d: any) => (
                <SelectItem key={d.value} value={d.value}>{d.text}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 威胁等级筛选 */}
          <Select value={levelFilter} onValueChange={(val) => { setLevelFilter(val); setPage(1); }}>
            <SelectTrigger size="sm" className="w-[140px]">
              <SelectValue placeholder="威胁等级" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部等级</SelectItem>
              {SKILL_LEVEL_DATA.map((d: any) => (
                <SelectItem key={d.value} value={d.value}>{d.text}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* 搜索 */}
          <div className="relative max-w-xs">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#A3A3A3]" />
            <Input
              placeholder="搜索告警名称"
              value={searchText}
              onChange={(e) => {
                setSearchText(e.target.value);
                setPage(1);
              }}
              className="pl-8 h-8 w-[180px]"
            />
          </div>

          {/* 刷新 */}
          <button
            onClick={refreshTable}
            className="w-8 h-8 flex items-center justify-center rounded-[4px] border border-[#E5E5E5] bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
            title="刷新表格"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* 右侧：主按钮组（更多操作 / 标记已处理 / 同步资产） */}
        <div className="flex items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="claw-outline" size="claw-sm" disabled={!selectedKeys.length}>
                更多操作
                <ChevronDown className="w-3.5 h-3.5 ml-1" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                disabled={!selectedKeys.length}
                onClick={() => {
                  setBatchType('separate');
                  setBatchHandleModalVisible(true);
                }}
              >
                隔离文件
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!selectedKeys.length}
                onClick={() => {
                  setBatchType('markHandle');
                  setBatchHandleModalVisible(true);
                }}
              >
                标记处置
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!selectedKeys.length}
                onClick={() => {
                  setBatchType('markIgnore');
                  setBatchHandleModalVisible(true);
                }}
              >
                标记忽略
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!selectedKeys.length}
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
            variant="claw-outline"
            size="claw-sm"
            disabled={!selectedKeys.length}
            onClick={() => {
              setBatchType('mark');
              setBatchHandleModalVisible(true);
            }}
          >
            标记已处理
          </Button>
          {/* 同步资产按钮（主按钮，最右） */}
          {showBanner && !InstanceId && (
            <SyncAssetBtn
              size="sm"
              refreshTable={refreshTable}
              rencentScanTime={rencentScanTime}
            />
          )}
        </div>
      </div>

      {/* 已选提示 */}
      {selectedKeys.length > 0 && (
        <div className="mb-2 px-3 py-2 bg-[#EFF6FF] text-[#1447E6] rounded-[4px] text-sm flex items-center justify-between">
          <span>已选择 {selectedKeys.length} 项</span>
          <Button variant="link" className="h-auto p-0 text-sm" onClick={clearSelected}>
            清除选择
          </Button>
        </div>
      )}

      {/* 表格 / 空状态 */}
      {allData.length > 0 ? (
      <SurfaceCard className="relative overflow-hidden">
        {isLoading && (
          <div className="absolute inset-0 bg-white/60 z-10 flex items-center justify-center">
            <RefreshCw className="w-6 h-6 animate-spin text-[#1447E6]" />
          </div>
        )}

        <Table scrollX="max-content" variant="white" containerClassName="bg-white">
          <TableHeader>
            <TableRow>
              {columns.map((col: any) => {
                const isAction = col.key === 'Operate';
                const isCheckbox = col.key === 'checkbox';
                return (
                  <TableHead
                    key={col.key}
                    fixed={col.fixed}
                    fixedShadow={isCheckbox ? false : undefined}
                    style={col.width ? { width: col.width } : col.minWidth ? { minWidth: col.minWidth } : undefined}
                  >
                    {col.header}
                  </TableHead>
                );
              })}
            </TableRow>
          </TableHeader>
          <TableBody>
            {allData.map((item: any, idx: number) => (
              <TableRow key={item?.Id || idx}>
                {columns.map((col: any) => {
                  if (col.key === 'Operate') {
                    return (
                      <TableActionCell key={col.key} fixed="right" rawChildren>
                        {col.render(item)}
                      </TableActionCell>
                    );
                  }
                  if (col.key === 'checkbox') {
                    return (
                      <TableCell
                        key={col.key}
                        fixed="left"
                        fixedShadow={false}
                        style={{ width: col.width }}
                      >
                        {col.render(item)}
                      </TableCell>
                    );
                  }
                  return (
                    <TableCell
                      key={col.key}
                      style={col.width ? { width: col.width } : col.minWidth ? { minWidth: col.minWidth } : undefined}
                    >
                      {col.render(item)}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
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
            pageSize={pageSize}
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
              <EmptyDescription>暂无恶意 Skills 数据</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </SurfaceCard>
      )}

      {/* 批量操作弹窗 */}
      <BatchOperatorDialog
        visible={batchHandleModalVisible}
        title={(SKILL_BATCH_TITLE_MAP as Record<string, string>)[batchType]}
        okText="确定"
        onCancel={() => setBatchHandleModalVisible(false)}
        content={
          <div>
            {batchType !== 'del' && selectedRows?.filter((item: any) => String(item?.Status) !== '4')?.length > 0 && (
              <p className="mt-2 text-sm text-[#525252]">
                其中有{selectedRows?.filter((item: any) => String(item?.Status) !== '4')?.length}条数据将不能执行操作。只有数据为&quot;待处理&quot;状态，才能执行{(SKILL_BATCH_TITLE_MAP as Record<string, string>)[batchType]}操作
              </p>
            )}
            {((batchType !== 'del' && selectedRows?.filter((item: any) => String(item?.Status) === '4')?.length > 0) ||
              batchType === 'del') && (
                <p className="mt-2 text-sm text-[#525252]">
                  您确定要对选中的数据进行{(SKILL_BATCH_TITLE_MAP as Record<string, string>)[batchType]}操作吗？
                </p>
              )}
          </div>
        }
        disabled={batchType !== 'del' && selectedRows?.filter((item: any) => String(item?.Status) === '4')?.length === 0}
        onOk={() => {
          setBatchHandleModalVisible(false);
          const allIds = selectedKeys?.map?.((Id: string) => Number(Id));
          const ids = selectedRows?.filter((item: any) => String(item?.Status) === '4')?.map((item: any) => Number(item?.Id));
          modifyEventsStatus(
            RISK_TYPE_SKILL,
            batchType === 'markHandle' ? 'mark' : batchType === 'markIgnore' ? 'ignore' : batchType,
            batchType === 'del' ? allIds : ids,
            () => {
              refreshTable();
              clearSelected();
            },
            setBatchTimer,
            setIsBatchLoading,
          );
        }}
        renderItem={(item: any) => {
          const name = item?.VirusName || item?.FileName || '-';
          const path = item?.FilePath || '-';
          return `告警：${name} - 路径：${path}`;
        }}
        data={selectedRows}
      />

      {/* 详情抽屉 */}
      {detailVisible && (
        <SkillDetailDrawer
          visible={detailVisible}
          record={detailRecord}
          aiAgentInfo={aiAgentHostList?.find((item: any) => item?.InstanceID === detailRecord?.MachineExtraInfo?.InstanceID)}
          onClose={() => {
            setDetailVisible(false);
            setDetailRecord(null);
          }}
          onRefresh={refreshTable}
        />
      )}

      {/* 确认弹窗 */}
      {renderConfirmModal()}
    </div>
  );
}
