/*  */
/* eslint-disable  */
 
 
import React, { useState, useEffect } from 'react';
import _ from '@/vendor/lodash';
import { Info, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea } from '@/components/ui/scroll-area';

import { DescribeImportMachineInfo, DescribeMachines } from '@/pages/admin/Security/api';

import { checkMachineIsWindows } from '../constants';

import { requestApi } from './requestApi';
import { renderCloudTags } from './CommonCloudTags';

const importTypeMap:any = {
  Ip: '内网IP',
  Name: 'Agent名称',
  Id: '实例ID',
};

export const renderStatusInfo = (record:any) => {
  if (!record?.Uuid || record?.MachineStatus === 'UNINSTALLED') {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="inline-block h-3.5 w-3.5 ml-1 text-muted-foreground" />
        </TooltipTrigger>
        <TooltipContent>未安装OpenClaw客户端</TooltipContent>
      </Tooltip>
    );
  }
  if (record?.InstanceState === 'TERMINATED_PRO_VERSION') {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="inline-block h-3.5 w-3.5 ml-1 text-muted-foreground" />
        </TooltipTrigger>
        <TooltipContent>OpenClaw已销毁</TooltipContent>
      </Tooltip>
    );
  }
  return null;
};

export const getRenderColumns = (showTagColumns:any, renderColumns:any, renderDisabledContent:any) => {
  const columns: any[] = [
    {
      header: 'Agent名称/实例ID',
      key: 'MachineName',
      render: (item: any) => (
        <div>
          <div className="machineName-btn-textOverflow" title={item?.MachineName || item?.InstanceName}>
            {item?.MachineName || item?.InstanceName || '未命名'}
          </div>
          <div>
            {item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId || '--'}
            {renderDisabledContent ? renderDisabledContent(item) : renderStatusInfo(item)}
          </div>
        </div>
      ),
    },
    {
      header: 'IP地址',
      key: 'MachineIp',
      width: 130,
      render: (item: any) => (
        <div>
          <div>
            <span className="newbuy-ip-label">公</span>
            <span className="newbuy-table-text">
              {item?.MachineExtraInfo?.WanIP || item?.MachineWanIp || item?.MachinePublicIp || '--'}
            </span>
          </div>
          <div>
            <span className="newbuy-ip-label">内</span>
            <span className="newbuy-table-text">
              {item?.MachineExtraInfo?.PrivateIP || item?.MachineIp || item?.MachinePrivateIp || '--'}
            </span>
          </div>
        </div>
      ),
    },
  ];
  if (showTagColumns) {
    columns.push({
      header: '标签',
      key: 'Tag',
      width: 120,
      render: (item: any) => renderCloudTags(item),
    });
  }
  return [...columns, ...renderColumns];
};

interface BatchImportMachineProps {
  ImportType: string;
  QuuidList?: string[];
  Uuids?: string[];
  selectableType?: any;
  renderColumns?: any[];
  showRightTagColumns?: boolean;
  renderDisabledContent?: (record?: any) => React.ReactNode;
  onChange: (keys: string[], rows: any) => void;
  selectedRows?: any[];
  openSwitch?: boolean;
  setFetchLoading?: any;
  isQrcodeSetting?: any;
  isProcessGuard?: any;
  realAgentHosts?: any;
}

export const getNewRows = (keys: any, selectRows: any[], recordKey: string, allDevice: { filter: (arg0: (item: any) => any) => never[]; }) => {
  let newRows = [];
  const ids = selectRows.map((item: { [x: string]: any; }) => item[recordKey]);
  const diff = _.difference(keys, ids);
  const addList = allDevice?.filter((item: { [x: string]: any; }) => diff.includes(item[recordKey])) ?? [];
  newRows = selectRows.concat(addList);
  return newRows;
};

export const getSelectedRowsAndKeys = (response: any) => {
  const result: any = response ?? { HostInfoList: [] };
  const newArrRows = result.HostInfoList?.map?.((item: { HostIp: any; AliasName: any; MachinePublicIp: any; MachineWanIp: any; TagList: any[]; }) => ({
    ...item,
    MachinePrivateIp: item.HostIp,
    MachineName: item.AliasName,
    MachinePublicIp: item?.MachinePublicIp ?? item?.MachineWanIp ?? '',
    MachineTag:
      item?.TagList?.map?.((subItem: any, subIndex: any) => ({
        Rid: subIndex,
        Name: subItem,
        TagId: subIndex,
      })) ?? [],
  }));
  const newArrKeys = result.HostInfoList?.map?.((item: { Quuid: any; }) => item.Quuid);
  return {
    Keys: newArrKeys,
    Rows: newArrRows,
  };
};

export default function BatchImportMachine({
  ImportType,
  QuuidList = undefined,
  Uuids = undefined,
  selectableType = undefined,
  renderColumns = [],
  showRightTagColumns = true,
  renderDisabledContent = undefined,
  onChange,
  openSwitch = true,
  selectedRows,
  setFetchLoading = undefined,
  isQrcodeSetting = undefined,
  isProcessGuard = undefined,
  realAgentHosts = undefined,
}: BatchImportMachineProps) {
  const [inputVal, setInputVal] = useState('');
  const [selectRows, setSelectRows] = useState([] as any);
  const [importVals, setImportVals] = useState([] as any);
  const [errorMessage, setErrorMessage] = useState('请至少选择一台OpenClaw');
  const [data, setData] = useState({} as any);

  const filterHosts = (list: any[]) =>
    (realAgentHosts?.length ? list?.filter?.((d: { InstanceID: any; }) => realAgentHosts.map((a: { InstanceID: any; }) => a.InstanceID)?.includes?.(d?.InstanceID)) : []);

  const getBatchMachines = async (params: any) => {
    const res: any = await DescribeImportMachineInfo(params);
    const lhData = res?.EffectiveMachineInfoList?.filter?.((d: { InstanceID: string | string[]; }) => d?.InstanceID?.indexOf?.('lh') === 0) || [];
    let winsData = [];
    if (isProcessGuard && res?.EffectiveMachineInfoList?.length > 0) {
      const resp: any = await DescribeMachines({
        MachineRegion: 'all-regions',
        MachineType: 'ALL',
        Offset: 0,
        Limit: 100,
        Filters: [{ Name: 'Quuid', Values: res?.EffectiveMachineInfoList?.map?.((d: { Quuid: any; }) => d?.Quuid)?.slice?.(0, 100) }],
      });
      const winQuuids = resp?.Machines?.filter?.((d: any) => checkMachineIsWindows(d))?.map?.((d: { Quuid: any; }) => d?.Quuid);
      winsData = res?.EffectiveMachineInfoList?.filter?.((d: { Quuid: any; }) => winQuuids?.includes?.(d?.Quuid));
    }
    setData(
      isProcessGuard
        ? { EffectiveMachineInfoList: filterHosts(winsData) }
        : isQrcodeSetting
          ? { EffectiveMachineInfoList: filterHosts(lhData) }
          : { EffectiveMachineInfoList: filterHosts(res?.EffectiveMachineInfoList || []) },
    );
  };

  const fetchSelectedRow = async () => {
    try {
      setFetchLoading?.(true);
      let result = { Rows: [], Keys: [] };
      if (typeof QuuidList?.[0] === 'string') {
        const response = await requestApi({
          cmd: 'DescribeHostInfo',
          data: { QuuidList },
        });
        result = getSelectedRowsAndKeys(response);
      } else if (typeof Uuids?.[0] === 'string') {
        const response = await requestApi({
          cmd: 'DescribeHostInfo',
          data: { Uuids },
        });
        result = getSelectedRowsAndKeys(response);
      } else {
        result = getSelectedRowsAndKeys({ HostInfoList: QuuidList });
      }
      setSelectRows(result.Rows);
      onChange?.(result.Keys, result.Rows);
      setErrorMessage('');
      setFetchLoading?.(false);
    } catch (error) {
      setSelectRows([]);
      onChange?.([], []);
      setFetchLoading?.(false);
    }
  };

  const fireChange = (keys: any) => {
    const recordKey = 'Quuid';
    const newRows:any = getNewRows(keys, selectRows, recordKey, data?.EffectiveMachineInfoList || []);
    setSelectRows(newRows);
    onChange(
      newRows.map((row: { Quuid: any; }) => row.Quuid),
      newRows,
    );
  };

  useEffect(() => {
    if (QuuidList?.length || Uuids?.length) {
      try {
        fetchSelectedRow();
      } catch (error) {
        setSelectRows([]);
      }
    }
    if (QuuidList?.length === 0) {
      setSelectRows([]);
    }
  }, [QuuidList, Uuids]);

  useEffect(() => {
    if (inputVal?.trim?.()) {
      setImportVals(
        inputVal
          ?.split?.('\n')
          ?.map?.(item => item.trim())
          ?.filter?.(item => item !== '') ?? [],
      );
    }
  }, [inputVal]);

  useEffect(() => {
    if (importVals?.length === 0) {
      return;
    }
    if (importVals?.length > 1000) {
      setErrorMessage('最多不能超过1000行');
      return;
    }
    setErrorMessage('');
    const params: any = {
      ImportType,
      MachineList: importVals,
    };
    if (selectableType) {
      params.Filters = [
        {
          Name: 'Version',
          Values: [selectableType],
        },
      ];
    }
    getBatchMachines(params);
  }, [importVals]);

  useEffect(() => {
    if (data) {
      const keys = data?.EffectiveMachineInfoList?.map((item: { Quuid: any; }) => item.Quuid) ?? [];
      const err = data?.InvalidMachineList?.toString();
      if (err) {
        setErrorMessage(`查询失败${importTypeMap[ImportType]}：${err}`);
      }
      fireChange(keys);
    }
  }, [data]);

  useEffect(() => {
    if (setSelectRows.length === 0) {
      setErrorMessage('请至少选择一台OpenClaw');
    }
  }, [setSelectRows]);

  useEffect(() => {
    if (selectedRows) {
      setSelectRows(selectedRows);
    }
  }, [selectedRows]);

  const columns = getRenderColumns(showRightTagColumns, renderColumns, renderDisabledContent);

  const cancelAllSelected = () => {
    setImportVals([]);
    setSelectRows([]);
    onChange([], []);
  };

  const removeRow = (quuid: string) => {
    const rows = selectRows.filter((item:any) => item.Quuid !== quuid);
    setSelectRows(rows);
    onChange(
      rows.map((row:any) => row.Quuid),
      rows,
    );
  };

  return (
    <div className="pt-5">
      <div className="grid grid-cols-2 gap-4">
        {/* Left panel - input */}
        <div className="border rounded-[4px]">
          <div className="border-b px-4 py-2 flex items-center justify-between">
            <span className="font-medium text-sm">{`请输入${importTypeMap[ImportType]}`}</span>
          </div>
          <div className="p-2">
            <span className="text-xs text-muted-foreground">每行输入1个，换行输入多个（最多不超过1000行）</span>
          </div>
          <div className="p-2 pt-0">
            <Textarea
              className="batch-import-input min-h-[340px] resize-none"
              value={inputVal}
              disabled={!openSwitch}
              onChange={e => setInputVal(e.target.value)}
            />
          </div>
        </div>

        {/* Right panel - selected machines */}
        <div className="border rounded-[4px]">
          <div className="border-b px-4 py-2 flex items-center justify-between">
            <span className="font-medium text-sm">{`已选择 ${selectRows.length} 台OpenClaw`}</span>
            <Button variant="link" size="sm" disabled={!openSwitch} onClick={() => cancelAllSelected()}>
              取消全部选择
            </Button>
          </div>
          <ScrollArea className="h-[340px]">
            <Table>
              <TableHeader>
                <TableRow>
                  {columns.map((col: any) => (
                    <TableHead key={col.key} style={col.width ? { width: col.width } : undefined}>
                      {col.header}
                    </TableHead>
                  ))}
                  <TableHead style={{ width: 40 }} />
                </TableRow>
              </TableHeader>
              <TableBody>
                {(selectRows || []).map((row: any) => (
                  <TableRow key={row.Quuid}>
                    {columns.map((col: any) => (
                      <TableCell key={col.key}>{col.render(row)}</TableCell>
                    ))}
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        disabled={!openSwitch}
                        onClick={() => removeRow(row.Quuid)}
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </div>
      </div>
      <p className="h-[18px] text-[#DC2626] mb-2.5">{errorMessage}</p>
    </div>
  );
}
