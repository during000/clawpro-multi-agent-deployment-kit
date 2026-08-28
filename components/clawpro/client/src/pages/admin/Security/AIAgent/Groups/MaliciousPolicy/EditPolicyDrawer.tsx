

/* eslint-disable  */


import React, { useEffect, useState } from 'react';
import { Base64 } from '@/vendor/js-base64';
import { toast } from 'sonner';
import { Loader2, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { PanelTitle, SectionTitle, MetaText } from '@/components/ui/Typography';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerClose,
  DrawerBody,
  DrawerFooter,
} from '@/components/ui/drawer';
import { ModifyRiskDnsPolicy, DescribeMachineGeneral } from '@/pages/admin/Security/api';

import { LICENSE_TYPES_MAP, hostVersionMap } from '../BashPolicy/Constants';
import { AUTHORIZE_ROUTE, checkMachineIsWindows } from '../../constants';
import MultiTypeSelectMachine from '../../Common/MultiTypeSelectMachine';
import { Transfer } from '@/components/ui/transfer';

export function EditPolicyDrawer({
  type = 'create',
  from = undefined,
  selectItem,
  visible = false,
  setVisible,
  refreshTable,
  initParams = undefined,
  hasFlagship,
  aiAgentHostList,
}: any) {
  const [loading, setLoading] = useState(false);
  const [isEnabled, setIsEnabled] = useState(false);
  const [policyName, setPolicyName] = useState('');
  const [policyDesc, setPolicyDesc] = useState('');
  const [whiteType, setWhiteType] = useState('0');
  const [domains, setDomains] = useState('');
  const [hostScope, setHostScope] = useState('1');
  const [selectMachine, setSelectMachine] = useState([] as any);
  const [policyAction, setPolicyAction] = useState('0');
  const [machineStat, setMachineStat] = useState({} as any);
  const [modalVisible, setModalVisible] = useState(false);
  const [isHandleOld, setIsHandleOld] = useState(true);
  const [modalResult, setModalResult] = useState('loading');
  const [fetchLoading, setFetchLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  const handleSubmit = async () => {
    const isSystemBlock = String(selectItem?.PolicyType) === '0' && String(selectItem?.PolicyAction) === '2';
    if (!policyName?.trim?.()) {
      toast.error('策略名称不能为空');
      return;
    }
    if (policyName?.trim?.()?.length > 20 && !isSystemBlock) {
      toast.error('策略名称不能超过20个字符');
      return;
    }
    if (policyDesc?.trim?.()?.length > 200 && !isSystemBlock) {
      toast.error('策略描述不能超过200个字符');
      return;
    }
    if (!domains?.trim?.()?.length) {
      toast.error('域名详情不能为空');
      return;
    }
    const arr =
      domains
        ?.split?.('\n')
        ?.map?.(item => item?.trim?.())
        ?.filter?.(item => item) ?? [];
    if (arr?.length <= 0 && !isSystemBlock) {
      toast.error('域名详情不能为空');
      return;
    }
    const domainReg = /^(?=^.{3,255}$)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+$/;
    if (arr?.some?.(d => !domainReg.test(d) && !(d?.indexOf?.('*.') === 0 && domainReg.test(d?.slice?.(2))))) {
      toast.error('请输入正确格式的域名或泛域名');
      return;
    }
    if (arr?.length > 100 && !isSystemBlock) {
      toast.error('域名最多只能输入100个');
      return;
    }
    if (hostScope === '1' && !selectMachine?.length) {
      toast.error('请选择OpenClaw');
      return;
    }
    setLoading(true);
    if (type === 'create') {
      setModalVisible(true);
      setModalResult('loading');
    }
    const params: any = {
      PolicyName: policyName?.trim?.(),
      PolicyType: type === 'create' ? 1 : selectItem?.PolicyType ?? 1,
      PolicyDesc: policyDesc?.trim?.(),
      PolicyAction: Number(policyAction),
      HostScope: hostScope == '0' ? (policyAction == '2' ? 2 : whiteType === '1' ? 3 : 1) : 0,
      Domains: isSystemBlock ? selectItem?.Domains || [] : arr?.map?.(str => Base64.encode(str)),
      IsEnabled: isEnabled ? 0 : 1,
      HostIds: hostScope == '1' ? selectMachine : [],
    };
    if (type === 'edit') {
      params.PolicyId = selectItem?.PolicyId;
    }
    if (policyAction == '1' && type === 'create') {
      params.IsDealOldEvent = isHandleOld ? 1 : 0;
    }
    if (from === 'alarmList' && initParams?.EventId) {
      params.EventId = initParams?.EventId;
    }
    const res: any = await ModifyRiskDnsPolicy({ Data: params });
    if (String(res?.Repeat) === '1') {
      setModalVisible(false);
      setLoading(false);
      toast.error('存在相同策略，无法创建');
      return;
    }
    if (res) {
      if (type === 'create') {
        setModalResult('success');
      } else {
        setVisible?.(false);
      }
      toast.success('操作成功');
      refreshTable?.();
    } else if (type === 'create') {
      setModalResult('fail');
      setErrMsg(res?.msg || res?.uiMsg || res?.message || '');
    }
    setLoading(false);
  };

  const getMachineTotal = async () => {
    const res: any = await DescribeMachineGeneral();
    setMachineStat(res || {});
  };

  useEffect(() => {
    if (visible) {
      setLoading(false);
      setIsHandleOld(type === 'create' ? true : String(selectItem?.IsDealOldEvent) === '1');
      setIsEnabled(type === 'create' ? true : String(selectItem?.IsEnabled) === '0');
      setPolicyName(type === 'create' ? '' : selectItem?.PolicyName || '');
      setPolicyDesc(type === 'create' ? '' : selectItem?.PolicyDesc || '');
      setWhiteType(
        type === 'create'
          ? initParams?.PolicyAction == 1
            ? '1'
            : '0'
          : String(selectItem?.PolicyAction) === '1'
            ? '1'
            : '0',
      );
      setHostScope('1');
      setSelectMachine(
        type === 'create'
          ? initParams?.PolicyAction == 2
            ? initParams?.HostIds
            : []
          : (selectItem?.HostIds || [])?.filter?.((id: any) => id),
      );
      setDomains(
        type === 'create'
          ? initParams?.PolicyAction
            ? initParams?.Domain
            : ''
          : selectItem?.Domains?.join?.('\n') || '',
      );
      setPolicyAction(
        type === 'create'
          ? initParams?.PolicyAction
            ? `${initParams?.PolicyAction}`
            : '0'
          : `${selectItem?.PolicyAction || '0'}`,
      );
      getMachineTotal();
    }
  }, [visible]);

  return (
    <>
      <Drawer open={visible} onOpenChange={open => { if (!open) setVisible?.(false); }} direction="right">
        <DrawerContent className="data-[vaul-drawer-direction=right]:w-[760px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
          <DrawerHeader className="flex flex-row items-center justify-between gap-4 p-4 bg-background text-left">
            <DrawerTitle asChild>
              <PanelTitle as="h2">{`${type === 'edit' ? '编辑' : '创建'}策略`}</PanelTitle>
            </DrawerTitle>
            <DrawerClose asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950"
                aria-label="关闭"
              >
                <X className="w-4 h-4" />
              </Button>
            </DrawerClose>
          </DrawerHeader>
          <DrawerBody>
            <div className="px-5 py-4 space-y-6">
              <Alert>
                <AlertDescription>
                  {'白名单策略支持全部 OpenClaw；告警策略支持专业版、旗舰版 OpenClaw；拦截策略仅支持 Linux 系统的旗舰版 OpenClaw，可点击\u00A0'}
                  <a className="text-[#1447E6] underline cursor-pointer" onClick={() => window.open(AUTHORIZE_ROUTE)}>升级版本</a>
                </AlertDescription>
              </Alert>

              {/* 基本信息 */}
              <section>
                <SectionTitle as="h3" className="!text-sm !font-semibold mb-3">基本信息</SectionTitle>
                <div className="space-y-4">
                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-2 text-sm text-[#525252] font-normal">
                      <span className="text-[#DC2626] mr-0.5">*</span>策略名称
                    </Label>
                    <Input
                      value={policyName}
                      onChange={e => setPolicyName(e.target.value)}
                      disabled={String(selectItem?.PolicyType) === '0' && String(selectItem?.PolicyAction) === '2'}
                      className="max-w-[600px]"
                      placeholder="请输入策略名称，限制 20 个字符以内"
                    />
                  </div>
                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-2 text-sm text-[#525252] font-normal">
                      策略描述
                    </Label>
                    <Input
                      value={policyDesc}
                      onChange={e => setPolicyDesc(e.target.value)}
                      disabled={String(selectItem?.PolicyType) === '0' && String(selectItem?.PolicyAction) === '2'}
                      className="max-w-[600px]"
                      placeholder="请输入策略描述，限制 200 个字符以内"
                    />
                  </div>
                  <div className="flex items-center gap-3">
                    <Label className="w-[88px] shrink-0 text-sm text-[#525252] font-normal">
                      <span className="text-[#DC2626] mr-0.5">*</span>开关
                    </Label>
                    <Switch checked={isEnabled} onCheckedChange={val => setIsEnabled(val)} />
                  </div>
                </div>
              </section>

              {/* 策略详情 */}
              <section>
                <SectionTitle as="h3" className="!text-sm !font-semibold mb-3">策略详情</SectionTitle>
                <div className="space-y-4">
                  <div className="flex items-center gap-3">
                    <Label className="w-[88px] shrink-0 text-sm text-[#525252] font-normal">
                      <span className="text-[#DC2626] mr-0.5">*</span>黑/白名单
                    </Label>
                    <RadioGroup
                      value={whiteType}
                      onValueChange={value => {
                        setWhiteType(value);
                        setPolicyAction(value === '0' ? '0' : '1');
                      }}
                      disabled={String(selectItem?.PolicyType) === '0' && String(selectItem?.PolicyAction) === '2'}
                      className="flex items-center gap-4"
                    >
                      <div className="flex items-center gap-2">
                        <RadioGroupItem value="0" id="blacklist" />
                        <Label htmlFor="blacklist" className="text-sm text-[#0A0A0A] font-normal cursor-pointer">黑名单</Label>
                      </div>
                      <div className="flex items-center gap-2">
                        <RadioGroupItem value="1" id="whitelist" />
                        <Label htmlFor="whitelist" className="text-sm text-[#0A0A0A] font-normal cursor-pointer">白名单</Label>
                      </div>
                    </RadioGroup>
                  </div>

                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-1.5 text-sm text-[#525252] font-normal">
                      <span className="text-[#DC2626] mr-0.5">*</span>执行动作
                    </Label>
                    <div className="flex-1 min-w-0">
                      {(() => {
                        const isLocked = String(selectItem?.PolicyType) === '0' && String(selectItem?.PolicyAction) === '2';
                        const items = [
                          { value: '0', label: '告警', disabled: whiteType === '1' || isLocked },
                          { value: '2', label: '拦截', disabled: whiteType === '1' || !hasFlagship || isLocked, tooltip: !hasFlagship ? '当前暂无旗舰版 OpenClaw，无法设置拦截策略，可点击升级版本' : null },
                          { value: '1', label: '放行', disabled: whiteType === '0' || isLocked },
                        ];
                        return (
                          <div className="inline-flex items-center gap-0.5 p-0.5 bg-[#F5F5F5] rounded-[4px]">
                            {items.map(it => {
                              const isActive = policyAction === it.value;
                              const btn = (
                                <button
                                  key={it.value}
                                  type="button"
                                  disabled={it.disabled}
                                  onClick={() => { if (!it.disabled) setPolicyAction(it.value); }}
                                  className={`px-3 py-1 text-xs rounded-[3px] transition-colors leading-5 ${
                                    isActive
                                      ? 'bg-white text-[#0A0A0A] font-medium'
                                      : it.disabled
                                        ? 'text-[#A3A3A3] cursor-not-allowed font-normal'
                                        : 'text-[#737373] hover:text-[#0A0A0A] font-normal'
                                  }`}
                                  style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                                >
                                  {it.label}
                                </button>
                              );
                              if (it.tooltip && it.disabled) {
                                return (
                                  <Tooltip key={it.value}>
                                    <TooltipTrigger asChild><span>{btn}</span></TooltipTrigger>
                                    <TooltipContent>{it.tooltip}</TooltipContent>
                                  </Tooltip>
                                );
                              }
                              return btn;
                            })}
                          </div>
                        );
                      })()}
                      <MetaText className="block mt-1.5">
                        {policyAction == '0' ? (
                          '当 OpenClaw 尝试对策略范围内的域名进行外联时，将产生告警记录。'
                        ) : policyAction == '2' ? (
                          <>
                            <span className="block">拦截规则只针对新启动进程发起的 IP / 域名 / 泛域名请求有效。</span>
                            <span className="block">当前仅支持 Linux 系统拦截，Windows 系统暂不支持。</span>
                          </>
                        ) : (
                          '当 OpenClaw 尝试对策略范围内的域名进行外联时，将不再产生告警或拦截行为。'
                        )}
                      </MetaText>
                    </div>
                  </div>

                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-2 text-sm text-[#525252] font-normal">
                      <span className="text-[#DC2626] mr-0.5">*</span>域名详情
                    </Label>
                    <div className="flex-1 min-w-0">
                      <Textarea
                        value={domains}
                        onChange={e => setDomains(e.target.value)}
                        disabled={String(selectItem?.PolicyType) === '0' && String(selectItem?.PolicyAction) === '2'}
                        className="max-w-[600px]"
                        style={{ resize: 'vertical' }}
                      />
                      <MetaText className="block mt-1.5">
                        请输入 IP / 域名 / 泛域名（如：www.12345.com、*.tencent.com 等，暂不支持 URL），多个内容以换行分隔
                      </MetaText>
                    </div>
                  </div>
                </div>
              </section>

              {/* 生效 OpenClaw 范围 */}
              <section>
                <div className="flex items-center justify-between mb-3">
                  <SectionTitle as="h3" className="!text-sm !font-semibold">
                    生效 OpenClaw 范围
                  </SectionTitle>
                  <MetaText>
                    {`已选择 ${
                      hostScope == '1'
                        ? selectMachine?.length || 0
                        : policyAction == '2'
                          ? machineStat?.FlagshipMachineCnt || 0
                          : whiteType === '1'
                            ? machineStat?.MachineCnt || 0
                            : (machineStat?.FlagshipMachineCnt || 0) + (machineStat?.SpecialtyMachineCnt || 0)
                    } 台`}
                  </MetaText>
                </div>
                {hostScope == '1' ? (
                  <Transfer<any>
                    dataSource={(aiAgentHostList ?? [])
                      .filter((h: any) => h?.Quuid)
                      .map((h: any) => ({ ...h, key: h?.Quuid }))}
                    rowKey="key"
                    targetKeys={selectMachine}
                    onChange={(nextKeys) => setSelectMachine(nextKeys)}
                    showSearch
                    searchPlaceholder={['搜索资产名称 / ID / IP', '搜索已选资产']}
                    pagination={{ pageSize: 8 }}
                    height={300}
                    titles={['全部 AI Agent 资产', '已选 AI Agent 资产']}
                    isItemDisabled={(h: any) => {
                      // 与原 isEnable 取反：
                      // - 拦截：仅旗舰版且非 Windows 可选
                      // - 白名单：全部可选
                      // - 告警/其他：旗舰版/PRO/通用折扣可选
                      if (policyAction == '2') {
                        return !(h?.ProtectType === 'Flagship' && !checkMachineIsWindows(h));
                      }
                      if (whiteType === '1') return false;
                      return !(
                        h?.ProtectType === 'Flagship' ||
                        h?.ProtectType === 'PRO_VERSION' ||
                        h?.ProtectType === 'GENERAL_DISCOUNT'
                      );
                    }}
                    renderDisabledTrigger={(h: any, defaultCheckbox) => {
                      const isWin = policyAction == '2' && checkMachineIsWindows(h);
                      const tip = isWin
                        ? '拦截策略仅支持 Linux 系统的旗舰版 OpenClaw'
                        : '基础版资产请升级到旗舰版以使用该能力';
                      return (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="inline-flex">{defaultCheckbox}</span>
                          </TooltipTrigger>
                          <TooltipContent>{tip}</TooltipContent>
                        </Tooltip>
                      );
                    }}
                    filterOption={(input, h: any) => {
                      const needle = input.toLowerCase();
                      return [h?.OpenClawName, h?.MachineName, h?.InstanceID, h?.MachineIp]
                        .filter((v) => typeof v === 'string')
                        .some((v) => v.toLowerCase().includes(needle));
                    }}
                    columns={[
                      {
                        key: 'name',
                        header: 'Agent 名称 / ID',
                        render: (h: any) => (
                          <div className="min-w-0">
                            <div className="truncate text-[var(--text-emphasis)]">
                              {h?.OpenClawName || h?.MachineName || '-'}
                            </div>
                            <MetaText className="block truncate">
                              {h?.InstanceID || '-'}
                            </MetaText>
                          </div>
                        ),
                      },
                      {
                        key: 'version',
                        header: '防护版本',
                        width: 100,
                        render: (h: any) => hostVersionMap[h?.ProtectType] ?? '-',
                      },
                      {
                        key: 'ip',
                        header: '内网IP',
                        width: 140,
                        render: (h: any) => h?.MachineIp || '-',
                      },
                    ]}
                  />
                ) : null}
              </section>

              {policyAction == '1' && type === 'create' && (
                <div className="flex items-center gap-2">
                  <Checkbox checked={isHandleOld} onCheckedChange={val => setIsHandleOld(!!val)} id="handle-old" />
                  <Label htmlFor="handle-old" className="text-sm text-[#0A0A0A] font-normal cursor-pointer">
                    对符合本策略规则的历史"待处理"告警，执行本策略规则的操作
                  </Label>
                </div>
              )}
            </div>
          </DrawerBody>
          <DrawerFooter className="flex flex-row items-center justify-end gap-3 p-4 bg-background border-t border-[#E5E5E5]">
            <Button variant="claw-outline" size="claw-sm" className="min-w-[80px]" onClick={() => setVisible?.(false)}>
              {'取消'}
            </Button>
            {policyName?.trim?.() && domains?.trim?.() ? (
              <Button variant="dialog-confirm" size="claw-sm" className="min-w-[80px]" disabled={loading || fetchLoading} onClick={handleSubmit}>
                {'保存'}
              </Button>
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <Button variant="dialog-confirm" size="claw-sm" className="min-w-[80px]" disabled>
                      {'保存'}
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent>{'未设置必填项'}</TooltipContent>
              </Tooltip>
            )}
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      <Dialog open={modalVisible} onOpenChange={open => { if (!open) { setModalVisible?.(false); setVisible?.(false); } }}>
        <DialogContent className="sm:max-w-[420px]" showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>
              {modalResult === 'loading'
                ? '策略创建中'
                : modalResult === 'success'
                  ? '策略创建成功'
                  : '策略创建失败'}
            </DialogTitle>
          </DialogHeader>
          <div className="text-sm text-[#525252] leading-relaxed">
            {modalResult === 'loading' && (
              <span className="inline-flex items-center gap-2 text-[#0A0A0A]">
                <Loader2 className="w-4 h-4 animate-spin" />
                正在创建策略，请耐心等待
              </span>
            )}
            {modalResult === 'success' && '已成功创建策略，策略已生效'}
            {modalResult !== 'loading' && modalResult !== 'success' &&
              (errMsg || '策略创建失败，请重新创建策略')}
          </div>
          <DialogFooter>
            {modalResult === 'loading' ? (
              <Button variant="claw-primary" size="claw-sm" disabled className="min-w-[90px]">
                <Loader2 className="w-4 h-4 animate-spin" />
              </Button>
            ) : modalResult === 'success' ? (
              from === 'alarmList' ? (
                <Button
                  variant="dialog-confirm"
                  size="claw-sm"
                  onClick={() => {
                    setModalVisible?.(false);
                    setVisible?.(false);
                  }}
                >
                  {'关闭'}
                </Button>
              ) : (
                <>
                  <Button
                    variant="claw-outline"
                    size="claw-sm"
                    onClick={() => {
                      setPolicyName('');
                      setPolicyDesc('');
                      setDomains('');
                      setSelectMachine([]);
                      setModalVisible?.(false);
                    }}
                  >
                    {'再创建一条'}
                  </Button>
                  <Button
                    variant="dialog-confirm"
                    size="claw-sm"
                    onClick={() => {
                      setModalVisible?.(false);
                      setVisible?.(false);
                    }}
                  >
                    {'返回列表'}
                  </Button>
                </>
              )
            ) : (
              <Button variant="dialog-confirm" size="claw-sm" onClick={() => setModalVisible?.(false)}>
                {'返回编辑策略'}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
