 
 
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';
import { ChevronDown } from 'lucide-react';
import { EditPolicyDrawer } from '../Groups/MaliciousPolicy/EditPolicyDrawer';
import { RISK_TYPE_MALICIOUS } from '../constants';
import { modifyEventsStatus } from '../Common/CommonRiskHandleFunc';
import CommonDropConfirm from '../Common/CommonDropConfirm';

import OperateConfirm from './OperateConfirm';

export default function MaliciousOperate({
  record,
  refreshTable,
  clearSelected,
  hasFlagship,
  aiAgentHostList,
  openDetail = undefined,
  hasNoDetail = false,
}:any) {
  const [item, setItem] = useState({} as any);
  const [policyVisible, setPolicyVisible] = useState(false);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  return (
    <>
      <div className="inline-flex items-center gap-3 whitespace-nowrap">
        {hasNoDetail ? null : (
          <Button variant="link" className="px-0 h-auto" onClick={openDetail}>
            {'详情'}
          </Button>
        )}
        {String(record?.HandleStatus) === '6' ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="link"
                className="gap-0 px-0 h-auto"
              >
                <span>更多</span>
                <ChevronDown className="h-3 w-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className={`malware-table-dropdown`}>
              <DropdownMenuItem
                onClick={() => {
                  const params:any = {
                    type: 'add',
                    Id: record?.Id,
                    Domain: record?.Domain || '',
                    ProtectLevel: record?.ProtectLevel,
                    HostIds: record?.HostId ? [record?.HostId] : [],
                  };
                  setItem(params);
                  setPolicyVisible(true);
                }}
              >
                {'加入白名单'}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => {
                  setDeleteConfirmOpen(true);
                }}
              >
                {'删除记录'}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : String(record?.HandleStatus) === '0' ? (
          <CommonDropConfirm
            item={record}
            type="malicious"
            isFlagship={record?.ProtectLevel == 2}
            isWindowsMachine={String(record?.OsType) === '6'}
            addFunc={() => {
              const params:any = {
                type: 'add',
                Id: record?.Id,
                Domain: record?.Domain || '',
                ProtectLevel: record?.ProtectLevel,
                HostIds: record?.HostId ? [record?.HostId] : [],
              };
              setItem(params);
              setPolicyVisible(true);
            }}
            blockFunc={() => {
              const params:any = {
                type: 'block',
                Id: record?.Id,
                Domain: record?.Domain || '',
                ProtectLevel: record?.ProtectLevel,
                HostIds: record?.HostId ? [record?.HostId] : [],
              };
              setItem(params);
              setPolicyVisible(true);
            }}
            eventType={RISK_TYPE_MALICIOUS}
            callback={() => {
              refreshTable?.();
              clearSelected?.();
            }}
          />
        ) : (
          <OperateConfirm
            title={'删除提醒'}
            message={'是否确认删除该记录？删除该告警记录，控制台将不再显示，无法恢复记录，请慎重操作。'}
            btnType="link"
            btnText={'删除记录'}
            operateHandle={() =>
              modifyEventsStatus(RISK_TYPE_MALICIOUS, 'del', record?.Id, () => {
                refreshTable?.();
                clearSelected?.();
              })
            }
          />
        )}
      </div>

      {/* 删除确认弹窗 - 仅 HandleStatus===6 时使用 */}
      {String(record?.HandleStatus) === '6' && (
        <Popover open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
          <PopoverTrigger asChild>
            <span />
          </PopoverTrigger>
          <PopoverContent align="end" side="bottom" className="w-72 p-4">
            <div className="space-y-2">
              <div className="text-sm font-medium">{'删除提醒'}</div>
              <div className="text-sm text-muted-foreground">
                {'是否确认删除该记录？删除该告警记录，控制台将不再显示，无法恢复记录，请慎重操作。'}
              </div>
              <div className="flex gap-2 pt-2">
                <Button
                  variant="link"
                  size="sm"
                  onClick={() => {
                    setDeleteConfirmOpen(false);
                    modifyEventsStatus(RISK_TYPE_MALICIOUS, 'del', record?.Id, () => {
                      refreshTable?.();
                      clearSelected?.();
                    });
                  }}
                >
                  <span>{'确定'}</span>
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setDeleteConfirmOpen(false)}
                >
                  <span>{'取消'}</span>
                </Button>
              </div>
            </div>
          </PopoverContent>
        </Popover>
      )}

      <EditPolicyDrawer
        visible={policyVisible}
        setVisible={setPolicyVisible}
        type="create"
        from="alarmList"
        initParams={{
          EventId: item?.Id,
          PolicyAction: item?.type === 'add' ? 1 : 2,
          Domain: item?.Domain || '',
          HostIds: item?.HostIds,
          ProtectLevel: item?.ProtectLevel,
        }}
        selectItem={{}}
        refreshTable={refreshTable}
        hasFlagship={hasFlagship}
        aiAgentHostList={aiAgentHostList}
      />
    </>
  );
}
