/* eslint-disable  */
 
 
import React, { useState } from 'react';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';

const fixedLabelWidth = '6.5em';

interface MultiTypeSelectMachineProps {
  layout?: 'default' | 'fixed';
  selectTypeChange?: (value: any) => void;
  openSwitch?: boolean;
  selectComponent: React.ReactNode;
  isShowFilterActions?: any;
}

export default function MultiTypeSelectMachine({
  layout = 'default',
  selectTypeChange = () => {},
  selectComponent,
  openSwitch = true,
  isShowFilterActions = true,
}: MultiTypeSelectMachineProps) {
  const [selectType, setSelectType] = useState('1');
  const [importType, setImportType] = useState('Ip');

  return (
    <div>
      {isShowFilterActions ? (
        <div className="flex items-center gap-3 pt-5">
          <Label style={{ width: fixedLabelWidth, flexShrink: 0 }}>选择方式</Label>
          <Select
            value={selectType}
            disabled={!openSwitch}
            onValueChange={value => {
              setSelectType(value);
              selectTypeChange(value);
            }}
          >
            <SelectTrigger className="w-[200px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1">直接勾选</SelectItem>
              <SelectItem value="2">批量查询</SelectItem>
            </SelectContent>
          </Select>
          {selectType === '2' ? (
            <Select
              value={importType}
              disabled={!openSwitch}
              onValueChange={value => setImportType(value)}
            >
              <SelectTrigger className="w-[200px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Name">Agent名称</SelectItem>
                <SelectItem value="Id">实例ID</SelectItem>
                <SelectItem value="Ip">内网IP</SelectItem>
              </SelectContent>
            </Select>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
