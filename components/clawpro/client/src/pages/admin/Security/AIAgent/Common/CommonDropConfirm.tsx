 
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import { ChevronDown } from 'lucide-react';

import { AUTHORIZE_ROUTE } from '../constants';

import { modifyEventsStatus } from './CommonRiskHandleFunc';

export default function CommonDropConfirm({
  item,
  type = undefined,
  addFunc,
  eventType,
  callback,
  isFlagship = undefined,
  diffTip = undefined,
  blockFunc = undefined,
  setIsShowModal,
  isWindowsMachine = undefined,
}:any) {
  const [handleType, setHandleType] = useState(type === 'crack0' ? 'upgrade' : 'mark');
  const [open, setOpen] = useState(false);
  return (
    <span>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="link"
            className="gap-0 px-0 h-auto"
            onClick={() => setOpen(true)}
          >
            <span>更多</span>
            <ChevronDown className="h-3 w-3" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" side="bottom" className="w-80 p-4">
          <div>
            <RadioGroup value={handleType} onValueChange={value => setHandleType(value)} className="flex flex-col gap-0">
              {type === 'crack0' && (
                <div>
                  <div className="flex items-center gap-2 py-1">
                    <RadioGroupItem value="upgrade" id={`upgrade-${item?.Id}`} />
                    <Label htmlFor={`upgrade-${item?.Id}`} className="cursor-pointer font-normal">
                      <span>升级专业版/旗舰版</span>
                      <span className="ml-1 inline-block rounded bg-[#DBEAFE] px-1.5 py-0.5 text-xs text-[#1447E6]">推荐</span>
                    </Label>
                  </div>
                  <div className="ml-6 text-xs text-muted-foreground pb-2">
                    升级专业版/旗舰版可开启密码破解自动阻断功能，建议升级并立即开启。
                  </div>
                </div>
              )}

              <div className="flex items-center gap-2 py-1">
                <RadioGroupItem value="mark" id={`mark-${item?.Id}`} />
                <Label htmlFor={`mark-${item?.Id}`} className="cursor-pointer font-normal">
                  <span>标记已处理</span>
                  {type !== 'crack0' && <span className="ml-1 inline-block rounded bg-[#DBEAFE] px-1.5 py-0.5 text-xs text-[#1447E6]">推荐</span>}
                </Label>
              </div>
              <div className="ml-6 text-xs text-muted-foreground pb-2">
                {diffTip
                  ? '若您已人工对该告警进行处理，可将告警标记为已处理。'
                  : '建议您参照告警详情中的"修复建议"，人工对该告警进行处理，处理后可将告警标记为已处理。'}
              </div>

              <div>
                <div className="flex items-center gap-2 py-1">
                  <RadioGroupItem value="trust" id={`trust-${item?.Id}`} />
                  <Label htmlFor={`trust-${item?.Id}`} className="cursor-pointer font-normal">
                    加入白名单
                  </Label>
                </div>
                <div className="ml-6 text-xs text-muted-foreground pb-2">
                  {type === 'bash'
                    ? '对告警的命令加入白名单操作后，当再次发生相同情况时将不再进行告警，同时当前告警状态将变更为"已加白"。'
                    : type === 'malicious'
                      ? '对当前告警的域名创建放行策略，当再次发生相同攻击时将不再进行告警，同时当前告警状态将变更为"已加白"。'
                      : type === 'memShell'
                        ? '对当前Java内存马告警创建白名单策略，当再次发生相同攻击时将不再进行告警，同时当前告警状态将变更为"已加白"。'
                        : '加入白名单操作后，当再次发生相同情况时将不再进行告警，请谨慎操作。'}
                </div>
              </div>

              {(type === 'bash' || type === 'malicious') && (
                <div>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="flex items-center gap-2 py-1">
                        <RadioGroupItem
                          value="createBlock"
                          id={`createBlock-${item?.Id}`}
                          disabled={!isFlagship || isWindowsMachine}
                        />
                        <Label htmlFor={`createBlock-${item?.Id}`} className={`cursor-pointer font-normal ${(!isFlagship || isWindowsMachine) ? 'opacity-50' : ''}`}>
                          <span>创建自定义拦截策略</span>
                          {isFlagship && (
                            <span className="ml-1 inline-block rounded bg-green-100 px-1.5 py-0.5 text-xs text-green-600">
                              NEW
                            </span>
                          )}
                        </Label>
                      </div>
                    </TooltipTrigger>
                    {isWindowsMachine && isFlagship ? (
                      <TooltipContent>Windows系统暂不支持</TooltipContent>
                    ) : null}
                  </Tooltip>
                  <div className="ml-6 text-xs text-muted-foreground pb-2">
                    {isFlagship ? (
                      type === 'bash' ? (
                        '对当前告警创建拦截策略，当再次发生相同攻击时将为您进行自动拦截。'
                      ) : (
                        '对当前告警的域名创建拦截策略，当再次发生相同攻击时将为您进行自动拦截。'
                      )
                    ) : (
                      <span>
                        {'自定义拦截策略仅针对旗舰版机器生效，可点击 '}
                        <Button
                          variant="link"
                          className="h-auto p-0"
                          onClick={() => {
                            window.open(AUTHORIZE_ROUTE);
                          }}
                        >
                          <span>升级版本</span>
                        </Button>
                      </span>
                    )}
                  </div>
                </div>
              )}

              <div className="flex items-center gap-2 py-1">
                <RadioGroupItem value="ignore" id={`ignore-${item?.Id}`} />
                <Label htmlFor={`ignore-${item?.Id}`} className="cursor-pointer font-normal">
                  忽略
                </Label>
              </div>
              <div className="ml-6 text-xs text-muted-foreground pb-2">
                仅将本次告警进行忽略，若再有相同情况发生依然会进行告警。
              </div>

              <div className="flex items-center gap-2 py-1">
                <RadioGroupItem value="del" id={`del-${item?.Id}`} />
                <Label htmlFor={`del-${item?.Id}`} className="cursor-pointer font-normal">
                  删除记录
                </Label>
              </div>
              <div className="ml-6 text-xs text-muted-foreground pb-2">
                删除该告警记录，控制台将不再显示，无法恢复记录，请慎重操作。
              </div>
            </RadioGroup>
          </div>

          <div className="flex gap-2 mt-3 pt-3 border-t">
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                setOpen(false);
                if (handleType === 'trust') {
                  addFunc?.();
                } else if (handleType === 'del' || handleType === 'ignore' || handleType === 'mark') {
                  if (type === 'memShell') {
                    // ModifyMemShellsStatus(item?.Id, batchStatusMap[handleType], callback);
                  } else if (type === 'loginlog' && handleType === 'del') {
                    setIsShowModal?.(true);
                  } else {
                    modifyEventsStatus(eventType, handleType, item?.Id, callback);
                  }
                } else if (handleType === 'upgrade') {
                  window.open(AUTHORIZE_ROUTE);
                } else if (handleType === 'createBlock') {
                  blockFunc?.();
                }
              }}
            >
              确认
            </Button>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setOpen(false)}>
              取消
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </span>
  );
}
