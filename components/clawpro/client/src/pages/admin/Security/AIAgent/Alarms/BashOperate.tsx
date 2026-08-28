 
 
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
import { EditPolicyDrawer } from '../Groups/BashPolicy/EditPolicyDrawer';
import { RISK_TYPE_BASH } from '../constants';
import { modifyEventsStatus } from '../Common/CommonRiskHandleFunc';
import CommonDropConfirm from '../Common/CommonDropConfirm';

import OperateConfirm from './OperateConfirm';

export default function BashOperate({
  record,
  refreshTable,
  clearSelected,
  hasFlagship,
  aiAgentHostList,
  openDetail = undefined,
  hasNoDetail = false,
}: any) {
  const [policyVisible, setPolicyVisible] = useState(false);
  const [itemInfo, setItemInfo] = useState({} as any);
  const [fromType, setFromType] = useState('');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  const getOperatorContent = () => {
    const detail = (
      <Button variant="link" className="px-0 h-auto" onClick={openDetail}>
        {'详情'}
      </Button>
    );

    const del = (
      <OperateConfirm
        title={'删除提醒'}
        message={'是否确认删除该记录？删除该告警记录，控制台将不再显示，无法恢复记录，请慎重操作。'}
        btnText={'删除记录'}
        operateHandle={() =>
          modifyEventsStatus(RISK_TYPE_BASH, 'del', record?.Id, () => {
            refreshTable?.();
            clearSelected?.();
          })
        }
      />
    );

    return (
      <>
        <div className="inline-flex items-center gap-3 whitespace-nowrap">
          {hasNoDetail ? null : detail}
          {String(record?.Status) === '0' ? (
            <CommonDropConfirm
              item={record}
              type="bash"
              isFlagship={record?.MachineType == 2}
              addFunc={() => {
                setFromType('alarmList');
                setItemInfo({
                  PolicyLevel: record?.RuleLevel,
                  PolicyAction: 1,
                  PolicyReg: record?.RegexBashCmd || '',
                  EventId: record?.Id,
                  MachineType: record?.MachineType,
                  Quuids: record?.Quuid ? [record?.Quuid] : null,
                  Path: record?.RegexExe || '',
                  Cmd: record?.RegexBashCmd || '',
                });
                setPolicyVisible?.(true);
              }}
              blockFunc={() => {
                setFromType('alarmList');
                setItemInfo({
                  PolicyLevel: record?.RuleLevel,
                  PolicyAction: 2,
                  PolicyReg: record?.RegexBashCmd || '',
                  EventId: record?.Id,
                  MachineType: record?.MachineType,
                  Quuids: record?.Quuid ? [record?.Quuid] : null,
                  Path: record?.RegexExe || '',
                  Cmd: record?.RegexBashCmd || '',
                });
                setPolicyVisible?.(true);
              }}
              eventType={RISK_TYPE_BASH}
              callback={() => {
                refreshTable?.();
                clearSelected?.();
              }}
            />
          ) : String(record?.Status) === '5' ? (
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
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={() => {
                    setFromType('alarmList');
                    setItemInfo({
                      PolicyLevel: record?.RuleLevel,
                      PolicyAction: '1',
                      PolicyReg: record?.RegexBashCmd || '',
                      EventId: record?.Id,
                      MachineType: record?.MachineType,
                      Quuids: record?.Quuid ? [record?.Quuid] : null,
                      Path: record?.RegexExe || '',
                      Cmd: record?.RegexBashCmd || '',
                    });
                    setPolicyVisible?.(true);
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
          ) : (
            del
          )}
        </div>

        {/* 删除确认弹窗 - 替代原来嵌套在 Dropdown 里的 PopConfirm */}
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
                    modifyEventsStatus(RISK_TYPE_BASH, 'del', record?.Id, () => {
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
      </>
    );
  };

  return (
    <>
      {getOperatorContent()}

      {policyVisible && (
        <EditPolicyDrawer
          from={fromType}
          visible={policyVisible}
          setVisible={setPolicyVisible}
          type="create"
          initParams={itemInfo}
          selectItem={{}}
          refreshTable={refreshTable}
          hasFlagship={hasFlagship}
          aiAgentHostList={aiAgentHostList}
        />
      )}
    </>
  );
}
