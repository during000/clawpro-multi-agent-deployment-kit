import _ from '@/vendor/lodash';
import { Base64 } from '@/vendor/js-base64';
import { toast } from 'sonner';
import { ModifyRiskEventsStatus, DescribeRiskBatchStatus } from '@/pages/admin/Security/api';

import { UNAUTH_CODE } from '../constants';
import { SetStateAction } from 'react';
import { createSecurityApi } from '../../api/shared';

const OVER_TIME = 60 * 60 * 1000;

export const getItemKeyData = (item: { [x: string]: any; }, key: string) => {
  const key1 = key?.split?.(';')?.[0];
  const key2 = key?.split?.(';')?.[1];
  return key?.indexOf?.(';') > -1 ? `${item?.[key1]};${item?.[key2]}` : `${item[key]}`;
};

export function getSelectionRows(selectRows: any[], keys: any, allDevice: any[], uniKey = 'Id') {
  const recordKey = uniKey || 'Id';
  const ids = selectRows.map(item => getItemKeyData(item, recordKey));
  if (keys.length > ids.length) {
    // add
    const diff = _.difference(keys, ids);
    const addList = allDevice.filter(item => diff.includes(getItemKeyData(item, recordKey)));
    return selectRows.concat(addList);
  }
  // del
  return selectRows.filter(item => keys?.includes?.(getItemKeyData(item, recordKey)));
}

export const modifyEventsStatus = async (
  RiskType: string,
  Operate: string,
  Id: number[],
  callback: { (): void; (): void; (): void; (): void; (): void; (): void; (): void; (): void; (): void; (): void; (): void; },
  setBatchTimer = (a: any) => a,
  setIsBatchLoading = (a: any) => a,
  Ip = undefined,
  UpdateAll = undefined,
  KillProcess = undefined,
  ExcludeId = undefined,
  Filters = undefined,
) => {
  const operateMap:any = {
    mark: 0,
    ignore: 1,
    del: 2,
    del_13: 2,
    separate: 3,
    recover: 4,
    trust: 5,
    untrust: 6,
    kill: 7,
  };
  try {
    const params:any = {
      Ids: Array.isArray(Id) ? Id : [Number(Id)],
      Operate: operateMap[Operate],
      RiskType,
    };
    if (Ip) {
      (params as any).Ip = Array.isArray(Ip) ? Ip : [Ip];
      delete params.Ids;
    }
    if (UpdateAll) {
      (params as any).UpdateAll = true;
      delete params.Ids;
    }
    if (KillProcess) {
      (params as any).KillProcess = true;
    }
    if (ExcludeId) {
      (params as any).UpdateAll = true;
      (params as any).ExcludeId = Array.isArray(ExcludeId) ? ExcludeId : [ExcludeId];
      delete params.Ids;
    }
    if (Filters) {
      (params as any).Filters = Filters;
    }
    toast.loading('操作中...');
    const res:any = await ModifyRiskEventsStatus(params);
    if (res) {
      if (res?.IsSync == 1) {
        const start = Date.now();
        const data:any = await DescribeRiskBatchStatus({ RiskType });
        if (data) {
          if (data?.Status === 'Pending') {
            setIsBatchLoading?.(false);
            toast.dismiss();
            toast.success('操作成功');
            callback?.();
          } else {
            setIsBatchLoading?.(true);
            toast.loading('操作中...');
            const timer = window.setInterval(async () => {
              try {
                toast.loading('操作中...');
                const info:any = await DescribeRiskBatchStatus({ RiskType });
                if (info) {
                  if (info?.Status === 'Pending') {
                    window.clearInterval(timer);
                    setIsBatchLoading?.(false);
                    toast.dismiss();
                    callback?.();
                  } else if (Date.now() - start > OVER_TIME) {
                    window.clearInterval(timer);
                    setIsBatchLoading?.(false);
                    callback?.();
                    toast.dismiss();
                    toast.error('操作时间过长，已超时');
                  }
                } else {
                  window.clearInterval(timer);
                  setIsBatchLoading?.(false);
                  toast.dismiss();
                  callback?.();
                }
              } catch (err) {
                window.clearInterval(timer);
                setIsBatchLoading?.(false);
                toast.dismiss();
                callback?.();
              }
            }, 3000);
            setBatchTimer?.(timer);
          }
        } else {
          callback?.();
        }
      } else {
        toast.dismiss();
        toast.success('操作成功');
        callback?.();
      }
    }
  } catch (err) {
    setIsBatchLoading?.(false);
    toast.dismiss();
    toast.error('操作失败');
  }
};

export const getBatchStatus = async (RiskType: string, setIsBatchLoading: { (value: SetStateAction<boolean>): void; (value: SetStateAction<boolean>): void; (value: SetStateAction<boolean>): void; (arg0: boolean): void; }, callback: { (): void; (): void; (): void; (): void; }, setBatchTimer: { (value: SetStateAction<number>): void; (value: SetStateAction<number>): void; (value: SetStateAction<number>): void; (arg0: number): void; }) => {
  const data: any = await DescribeRiskBatchStatus({ RiskType });
  if (data?.Status === 'Handling') {
    setIsBatchLoading?.(true);
    const start = Date.now();
    const timer = window.setInterval(async () => {
      try {
        const info:any = await DescribeRiskBatchStatus({ RiskType });
        if (info) {
          if (info?.Status === 'Pending') {
            window.clearInterval(timer);
            setIsBatchLoading?.(false);
            callback?.();
          } else if (Date.now() - start > OVER_TIME) {
            window.clearInterval(timer);
            setIsBatchLoading?.(false);
            callback?.();
            toast.error('操作时间过长，已超时');
          }
        } else {
          window.clearInterval(timer);
          setIsBatchLoading?.(false);
          callback?.();
        }
      } catch (err) {
        window.clearInterval(timer);
        setIsBatchLoading?.(false);
        callback?.();
      }
    }, 2000);
    setBatchTimer?.(timer);
  } else {
    setIsBatchLoading?.(false);
  }
};

export const getRequestParams = (
  query: any,
  keysObj: { key: string; type: string }[],
  ExactMatch: boolean | null = false,
) => {
  const params = parseJsonStr(JSON.stringify(query));
  keysObj.forEach(item => {
    const key = item.key;
    if (params?.[key] === 'undefined') {
      delete params[key];
    } else if (params?.[key] !== undefined) {
      let filterObj: any = {};
      if (Array.isArray(params?.[key])) {
        filterObj = {
          Name: key,
          Values: [...params[key]],
        };
      } else {
        filterObj = {
          Name: key,
          Values: [item.type === 'number' ? Number(params[key]) : params[key]],
        };
      }

      if (ExactMatch !== null) {
        filterObj.ExactMatch = ExactMatch;
      }
      params.Filters.push(filterObj);
      delete params[key];
    }
  });

  return params;
};

export const sleep = (ms: number | undefined) => new Promise(resolve => setTimeout(resolve, ms));

export async function executeInstallTasksWithDelay(tasks: any[], ms: number, showErr = false) {
  if (!tasks?.length || !Array.isArray(tasks) || tasks?.some?.(d => !Array.isArray(d))) {
    return;
  }
  let result: any[] = [];
  for (const paramsArr of tasks) {
    await sleep(ms);
    const res = await Promise.all(
      paramsArr?.map?.((d: any) =>
        (!d
          ? null
          : createSecurityApi(d.cmd, d.serviceType)(d.data)
            .then((d: any) => d)
            .catch((e: any) => e)),
      ),
    );
    result = result.concat(res);
  }
  return result;
}

export function DyadicArr(arr: any[] = [], num: number) {
  const newarr = [];
  for (let i = 0; i < arr?.length; i += num) {
    newarr.push(arr.slice(i, i + num));
  }
  return newarr;
}

export const parseJsonStr = (str: string) => {
  try {
    const res = JSON.parse(str);
    return res;
  } catch (e) {
    console.log(e);
    return '';
  }
};

export const parseBase64Str = (str: string) => {
  try {
    const res = Base64.decode(str);
    return res;
  } catch (e) {
    console.log(e);
    return str || '';
  }
};

export async function executeTasksWithDelay(tasks: any, ms: number) {
  let result: any[] = [];
  for (const task of tasks) {
    await sleep(ms);
    const res = await Promise.all(task);
    result = result.concat(res);
  }
  return result;
}

export const requestAllPromises = async (
  total: number,
  cmd: (arg0: any) => any,
  params: (arg0: any, arg1: any) => any,
  cmdBackName = 'List',
  limit = 100,
  offset = undefined,
  maxRequestTime = 20,
  time = 1000,
) => {
  let list: any[] = [];
  if (total > limit) {
    const executeTimes = Math.ceil(total / limit) - 1;
    const allPromises = new Array(executeTimes).fill(1)
      .reduce((pre, cur, i) => {
        const index = Math.ceil((i + 1) / maxRequestTime) - 1;
        if (pre[index]) {
          if (pre[index]?.length < maxRequestTime) {
            pre[index] = pre[index].concat(cur);
          } else {
            pre[index + 1] = [cur];
          }
        } else {
          pre[index] = [cur];
        }
        return pre;
      }, []);
    const tasks = allPromises.map((p: any[], x: number) =>
      p?.map?.((d: any, i: number) =>
        cmd({
          ...(typeof params === 'function' ? params(x, i) : params),
          Limit: limit,
          Offset: typeof offset !== 'undefined' ? offset : x * limit * maxRequestTime + (i + 1) * limit,
        }),
      ),
    );
    await executeTasksWithDelay(tasks, time)
      .then(res => {
        list = list?.concat?.(res?.map?.(item => item?.[cmdBackName] || [])?.flat?.(2));
      })
      .catch(err => console.log(err));
  }
  return list;
};
