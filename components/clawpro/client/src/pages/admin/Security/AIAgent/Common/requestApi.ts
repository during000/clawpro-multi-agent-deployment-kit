import _ from '@/vendor/lodash';
import { toast } from 'sonner';

import { PRODUCT_TITLE, UNAUTH_CODE } from '../constants';
import { createSecurityApi, createSecurityMutateApi } from '../../api/shared';
import { parseJsonStr } from './CommonRiskHandleFunc';

type IServiceType =
  | 'yunjing'
  | 'ssa'
  | 'cwp'
  | 'account'
  | 'tcss'
  | 'cam'
  | 'tag'
  | 'vpc'
  | 'cbs'
  | 'lighthouse'
  | 'tat'
  | 'region'
  | 'cvm'
  | 'csip';

export const CWP_API_SERVICE_TYPE = 'cwp';
export const CWP_API_VERSION = '2018-02-28';
export const SOC_API_SERVICE_TYPE = 'ssa';
export const SOC_API_VERSION = '2018-06-08';

export const checkIfRequestIsSuccess = (response: { code: number; data: { Response: any; }; }) => response?.code === 0 && response;
export const checkIfRequestIsFail = (response: { code: number; data: { Response: any; }; }) => !response || response.code != 0 || !response;

interface IRequestApiType {
  regionId?: number;
  serviceType?: IServiceType;
  version?: string;
  cmd: string;
  data?: object;
  options?: object;
  showInnerTips?: boolean;
}

/**
 * 调用云api接口
 * data 入参统一处理
 */
export const requestApi = async ({
  regionId = 1,
  serviceType = CWP_API_SERVICE_TYPE,
  version = CWP_API_VERSION,
  cmd,
  data = {},
  options = {},
  showInnerTips = true,
}: IRequestApiType) => {
  try {
    const response = await (cmd?.indexOf?.('Describe') === 0 || cmd?.indexOf?.('Get') === 0||cmd?.indexOf?.('SearchLog') === 0 ? createSecurityApi(cmd, serviceType)(data) : createSecurityMutateApi(cmd, serviceType)(data))
      .then((d: any) => d)
      .catch((e: any) => e);
    return response;
  } catch (error: any) {
    if (showInnerTips) {
      const ErrorMessageInfo = error?.data?.message;
      const errMsg = error?.code === 'Unauthorized' || ErrorMessageInfo?.indexOf?.('not authorized') > -1
        ? '您没有权限进行此操作，请设置相关权限后再进行操作'
        : '';
      toast.warning(errMsg || ErrorMessageInfo);
    }
    return error;
  }
};

export function fixFilters(query: { Filters: { [x: string]: any; hasOwnProperty: (arg0: string) => any; }; }) {
  const tmpState = parseJsonStr(JSON.stringify(query));
  tmpState.Filters = [];

  for (const key in query.Filters) {
    // eslint-disable-next-line no-prototype-builtins
    if (query.Filters.hasOwnProperty(key) && !_.isEmpty(query.Filters[key])) {
      tmpState.Filters.push({ Name: key, Values: query.Filters[key] });
    }
  }

  if (tmpState.Filters.length === 0) {
    delete tmpState.Filters;
  }

  return tmpState;
}
