 
/* eslint-disable  */
 
import React, { useEffect, useState, useRef } from 'react';
import _ from '@/vendor/lodash';
import { Download, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';

import { requestApi } from './requestApi';
import { handleIfUseBase64Download } from './downloadFile';
import { DescribeExportCsv } from './DescribeExportCsv';
import { changeDownloadUrlToHttps } from './changeDownloadUrlToHttps';

export type ExportCsvButtonType = 'primary' | 'weak' | 'pay' | 'text' | 'link' | 'icon';

export interface ExportCsvModel {
  DownloadUrl?: string;
}

type createContent = (loading: boolean) => React.ReactNode;

interface ExportCsvUtilProps {
  style?: React.CSSProperties;
  className?: string;
  title?: string;
  content?: string | createContent;
  requestApi?: string | any;
  reportTag?: string;
  stateProps?: any;
  params?: { [key: string]: any };
  onFinish?: (data: any) => boolean | void;
  onDownloadFinish?: () => void;
  interval?: number;
  bubbleContent?: React.ReactNode;
  placement?: any;
  beforeDownload?: () => void;
  serviceType?: any;
}

interface BaseData {
  DownloadUrl: string;
  TaskId?: string;
}

interface TaskData extends BaseData {
  Status?: string;
}

export const ExportCsv = (props: ExportCsvUtilProps) => {
  const {
    content = '',
    style = {},
    className = '',
    interval = 5000,
    title,
    onFinish,
    params,
    bubbleContent = null,
    requestApi: hpApi,
    stateProps,
    beforeDownload,
    serviceType,
    placement = 'top',
    onDownloadFinish,
  } = props;
  const [innerLoading, setInnerLoading] = useState(false);
  const [data, setData] = useState<BaseData>({} as BaseData);
  const [taskData, setTaskData] = useState<TaskData>({} as TaskData);
  const startTime = useRef<Date>(new Date());
  const timer:any = useRef<number>(null);

  const normalFetchData = typeof hpApi === 'string'
    ? async () => {
      const queryParams = {
        cmd: hpApi,
      } as {
        cmd: string;
        data?: any;
        serviceType?: any;
      };
      if (!_.isEmpty(params)) {
        queryParams.data = params;
      }
      if (!_.isEmpty(serviceType)) {
        queryParams.serviceType = serviceType;
      }
      const response = await requestApi(queryParams);
      const result = response ?? {};
      return result;
    }
    : hpApi;

  const fetchTaskData = async (currentTaskId: string) => {
    const result = await DescribeExportCsv(currentTaskId);
    if (!result || result?.Status === 'FINISHED' || result?.Status === 'ERROR') {
      clearInterval(timer.current);
      setInnerLoading(false);
      if (result?.DownloadUrl) {
        handleBeforeDownload(result);
      }
      if (result?.Status === 'ERROR') {
        toast.error('导出失败');
      }
      return;
    }
    setTaskData(result);
    setInnerLoading(false);
  };

  const fetchTask = async (currentTaskId: string) => {
    window.clearInterval(timer.current);
    timer.current = window.setInterval(() => {
      fetchTaskData(currentTaskId);
    }, interval);
  };

  const handleDownload = async (e: { currentTarget: { querySelector: (arg0: string) => any; }; }) => {
    const button = e?.currentTarget?.querySelector?.('button');
    if (typeof content !== 'string' && button?.disabled) {
      return;
    }
    beforeDownload?.();
    setInnerLoading(true);
    // toast.loading('导出数据中...');
    try {
      const result = await normalFetchData();
      setData(result);
      if (result?.code !== 0) {
        setInnerLoading(false);
      }
    } catch (err) {
      setInnerLoading(false);
    }
  };

  const handleBeforeDownload = async (currentData: any) => {
    window.clearInterval(timer.current);
    setInnerLoading(false);
    toast.dismiss();
    let result = null;
    const newDownloadUrl = changeDownloadUrlToHttps(currentData.DownloadUrl);
    currentData = {
      ...currentData,
      DownloadUrl: newDownloadUrl,
    };
    handleIfUseBase64Download(
      newDownloadUrl,
      currentData?.FileName,
      () => {
        result = onFinish?.(currentData);
        if (result === false) {
          return;
        }
        window.location.href = currentData.DownloadUrl;
      },
      () => {
        onDownloadFinish?.();
      },
    );
  };

  const handleDownloadFail = () => {
    setInnerLoading(false);
    window.clearInterval(timer.current);
    toast.dismiss();
    toast.error('导出文件失败');
  };

  useEffect(() => {
    if (data?.DownloadUrl) {
      handleBeforeDownload(data);
      return;
    }
    if (data?.TaskId) {
      startTime.current = new Date();
      fetchTask(data.TaskId);
    }
  }, [data]);

  useEffect(() => {
    if (stateProps?.data?.DownloadUrl) {
      handleBeforeDownload(stateProps.data);
      return;
    }
    if (stateProps?.data?.TaskId) {
      startTime.current = new Date();
      fetchTask(stateProps.data.TaskId);
    }
  }, [stateProps]);

  useEffect(() => {
    if (new Date().valueOf() - startTime?.current?.valueOf?.() >= 1000 * 60 * 5) {
      handleDownloadFail();
      return;
    }
    if (taskData?.Status === 'ERROR') {
      handleDownloadFail();
    }
  }, [taskData]);

  return (
    <div className={className} style={{ display: 'inline-block', padding: 4, ...style }}>
      <Tooltip>
        <TooltipTrigger asChild>
          {typeof content === 'string' ? (
            <span
              title={title}
              className="cursor-pointer inline-flex items-center"
              onClick={innerLoading ? () => {} : handleDownload}
            >
              {innerLoading
                ? <Loader2 className="h-4 w-4 animate-spin" />
                : <Download className="h-4 w-4" />}
            </span>
          ) : (
            <div onClick={innerLoading ? () => {} : handleDownload}>{content(innerLoading)}</div>
          )}
        </TooltipTrigger>
        {bubbleContent && <TooltipContent side={placement}>{bubbleContent}</TooltipContent>}
      </Tooltip>
    </div>
  );
};

export default ExportCsv;
