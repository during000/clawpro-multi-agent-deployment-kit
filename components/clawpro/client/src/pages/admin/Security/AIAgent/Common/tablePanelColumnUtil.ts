import moment from '@/vendor/moment';

import { GetLocalStorageItem, SetLocalStorageItem } from '@/pages/admin/Security/api';

import { FORMAT_BEGIN, FORMAT_END, FORMAT_NOW } from '../constants';

import { requestApi } from './requestApi';
import { parseJsonStr } from './CommonRiskHandleFunc';

export const rangeMap:any = {
  1: [moment(moment().format(FORMAT_BEGIN)), moment(moment().format(FORMAT_END))],
  2: [
    moment(
      moment()
        .subtract(1, 'd')
        .format(FORMAT_BEGIN),
    ),
    moment(
      moment()
        .subtract(1, 'd')
        .format(FORMAT_END),
    ),
  ],
  3: [
    moment(
      moment()
        .subtract(7, 'd')
        .format(FORMAT_NOW),
    ),
    moment(moment().format(FORMAT_NOW)),
  ],
  4: [
    moment(
      moment()
        .subtract(30, 'd')
        .format(FORMAT_NOW),
    ),
    moment(moment().format(FORMAT_NOW)),
  ],
  5: [],
};

export const rangeOptions = [
  { text: '今天', value: '1' },
  { text: '昨天', value: '2' },
  { text: '近7天', value: '3' },
  { text: '近30天', value: '4' },
  { text: '全部', value: '5' },
];

export const getRemoteStorage = async (Key: string, callback: { (val: string): void; (arg0: any): void; }) => {
  const res: any = await GetLocalStorageItem({ Key });
  callback?.(res?.Value || '');
};

export const setRemoteStorage = async (Key: string, val = '1', Expire = undefined) => {
  await SetLocalStorageItem({ Key, Value: val, Expire: Expire || 365 * 24 * 3600 });
};

// export const removeRemoteStorage = async Key => {
//   await RemoveLocalStorageItem({ Key });
// };

export const getMaxRemoteStorage = async (Key: string, callback: { (val: any): void; (val: any): void; (arg0: any): void; }) => {
  const res: any = await requestApi({
    cmd: 'GetLocalStorageItem',
    data: { Key, Scope: 'APPID' },
    regionId: 1,
    serviceType: 'csip',
  });
  callback?.(res?.Value || '');
};

export const setMaxRemoteStorage = async (Key: string, Value = '1', Expire:any = undefined) => {
  await requestApi({
    cmd: 'SetLocalStorageItem',
    data: { Key, Value, Expire: Expire || 365 * 24 * 3600, Scope: 'APPID' },
    regionId: 1,
    serviceType: 'csip',
  });
};

export const removeMaxRemoteStorage = async (Key: any) => {
  await requestApi({
    cmd: 'RemoveLocalStorageItem',
    data: { Key, Scope: 'APPID' },
    regionId: 1,
    serviceType: 'csip',
  });
};

/**
 * 存储tablepanel的表头数据
 * @param key localStorageKey
 * @param checkedList localStorageValue
 */
export const saveColKeys = async (Key: string, checkedList: string[], Expire = undefined) => {
  await SetLocalStorageItem({ Key, Value: JSON.stringify(checkedList), Expire: Expire || 365 * 24 * 3600 });
};

/**
 * 获取tablepanel的表头数据
 * @param storageKey localStorageKey
 */
export const fetchInitStorageShowColKeys = async (storageKey: string) => {
  // 迁移到远端存储。对于原来是用的客户，先进行一段时间的清理操作。
  localStorage.removeItem(storageKey);
  const storageColumn: any = await GetLocalStorageItem({ Key: storageKey });
  try {
    return parseJsonStr(storageColumn?.Value || null);
  } catch (error) {
    return [];
  }
};

// 自定义时间筛选
export const timeChangeHandler = ({ value, setRangeSegmentValue, setRangeValue, ref, startKey, endKey }:any) => {
  setRangeSegmentValue(null);
  setRangeValue(value);
  ref?.current?.onQueryDispatch?.({
    ...(startKey && endKey
      ? {
        [startKey]: value?.[0]?.format?.(FORMAT_NOW),
        [endKey]: value?.[1]?.format?.(FORMAT_NOW),
      }
      : {
        [startKey || endKey]: [value?.[0]?.format?.(FORMAT_NOW), value?.[1]?.format?.(FORMAT_NOW)],
      }),
    Offset: 0,
    Limit: 10,
  });
};

// 时间段筛选
export const timeSegChangeHandler = ({ value, setRangeSegmentValue, setRangeValue, ref, startKey, endKey }:any) => {
  setRangeSegmentValue(value);
  setRangeValue(rangeMap[value]);
  ref?.current?.onQueryDispatch?.({
    ...(startKey && endKey
      ? {
        [startKey]: rangeMap[value]?.[0]?.format?.(FORMAT_NOW),
        [endKey]: rangeMap[value]?.[1]?.format?.(FORMAT_NOW),
      }
      : {
        [startKey || endKey]: [
          rangeMap[value]?.[0]?.format?.(FORMAT_NOW),
          rangeMap[value]?.[1]?.format?.(FORMAT_NOW),
        ],
      }),
    Offset: 0,
    Limit: 10,
  });
};
