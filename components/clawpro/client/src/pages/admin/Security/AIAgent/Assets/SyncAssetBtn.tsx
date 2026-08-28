import React, { useEffect, useRef, useState } from 'react';
import moment from '@/vendor/moment';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import { SyncAssetScan, ScanAsset } from '@/pages/admin/Security/api';

import { FORMAT_NOW } from '../constants';

export default function SyncAssetBtn({
  btnType = undefined,
  style = {},
  refreshTable,
  refreshFromLock = undefined,
  rencentScanTime,
  variant,
  size,
}: any) {
  const [syncLoading, setSyncLoading] = useState(false);
  const syncTimerRef = useRef<number | undefined>(undefined);
  const lockSyncTimerRef = useRef<number | undefined>(undefined);

  const getSyncData = async (isInit: any) => {
    const res: any = await SyncAssetScan({ Sync: false });
    if (res?.State === 'SYNCING') {
      setSyncLoading(true);
      syncTimerRef.current = window.setTimeout(() => {
        getSyncData(false);
      }, 3000);
    } else {
      if (!isInit) {
        refreshTable?.();
      }
      setSyncLoading(false);
    }
  };

  const startScan = async () => {
    setSyncLoading(true);
    const res: any = await Promise.all([
      ScanAsset({ Quuids: [], AssetTypeIds: [17] }),
      // CreateScanTask()
    ]);
    getSyncData(false);
    if (refreshFromLock) {
      lockSyncTimerRef.current = window.setInterval(() => refreshFromLock?.(), 30000);
    }
    if (!res?.[0]?.TaskId) {
      setSyncLoading(false);
    }
  };

  useEffect(() => {
    getSyncData(true);
  }, []);

  useEffect(() => {
    return () => {
      window.clearTimeout(syncTimerRef.current);
      window.clearInterval(lockSyncTimerRef.current);
    };
  }, []);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        {btnType === 'link' && syncLoading ? (
          <span className="inline-flex items-center">
            <Loader2 className="w-4 h-4 animate-spin" />
          </span>
        ) : (
          <Button
            variant={variant ?? (btnType === 'link' ? 'link' : 'default')}
            size={size}
            style={style || {}}
            disabled={syncLoading}
            onClick={startScan}
          >
            {syncLoading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            <span>{syncLoading ? '同步资产中...' : '同步资产'}</span>
          </Button>
        )}
      </TooltipTrigger>
      <TooltipContent side="bottom" className="max-w-xs">
        <div>
          <div className="mb-2">
            {syncLoading
              ? '同步资产中...'
              : `最近采集时间：${rencentScanTime ? moment(rencentScanTime).format(FORMAT_NOW) : '-'}`}
          </div>
          <h4 className="text-xs font-semibold mb-1">同步资产说明</h4>
          <p className="text-xs">
            <span className="font-medium">手动触发同步：</span>
            随时可触发
          </p>
          <p className="text-xs">
            <span className="font-medium">定时自动同步：</span>
            每8小时左右自动采集一次
          </p>
          <p className="text-xs">
            <span className="font-medium">同步超时时长：</span>
            1小时，超出则采集失败
          </p>
          <p className="text-xs text-muted-foreground">
            注：OpenClaw被销毁或清理，相关资产指纹数据将同步被清除。
          </p>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
