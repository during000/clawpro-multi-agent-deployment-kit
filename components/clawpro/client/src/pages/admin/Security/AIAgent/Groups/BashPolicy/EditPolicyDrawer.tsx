

/* eslint-disable  */


import React, { useEffect, useState } from 'react';
import { Base64 } from '@/vendor/js-base64';
import { toast } from 'sonner';
import { Info, AlertCircle, Loader2, Plus, Trash2, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import {
  PanelTitle,
  CardTitle,
  MetaText,
} from '@/components/ui/Typography';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
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
import { ModifyBashPolicy, DescribeMachineGeneral, CheckBashPolicyParams } from '@/pages/admin/Security/api';

import { AUTHORIZE_ROUTE } from '../../constants';
import { Transfer } from '@/components/ui/transfer';

import { PROCESS_TYPES, PROCESS_TYPES_MAP, hostVersionMap, heightMap } from './Constants';

export const MAX_TEXT_LEN = 40;

export function EditPolicyDrawer({
  type = 'create',
  from = undefined,
  selectItem,
  visible,
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
  const [whiteType, setWhiteType] = useState('0'); // 0:黑名单 1:白名单
  const [hostScope, setHostScope] = useState('1');
  const [selectMachine, setSelectMachine] = useState([] as any);
  const [policyAction, setPolicyAction] = useState('0'); // 0:告警 1:白名单 2:拦截
  const [machineStat, setMachineStat] = useState({} as any);
  const [modalVisible, setModalVisible] = useState(false);
  const [isHandleOld, setIsHandleOld] = useState(true);
  const [nameErrMsg, setNameErrMsg] = useState('');
  const [isNameCorrect, setIsNameCorrect] = useState(true);
  const [modalResult, setModalResult] = useState('loading');
  const [errMsg, setErrMsg] = useState('');
  const [policyLevel, setPolicyLevel] = useState('1');
  const [fetchLoading, setFetchLoading] = useState(false);
  const [processList, setProcessList] = useState([] as any);
  const [processHeights, setProcessHeights] = useState([30]);
  const [processCmdHeights, setProcessCmdHeights] = useState([30]);

  const checkBashRuleParams = async (name: string) => {
    if (type !== 'edit') {
      if (name?.trim?.()) {
        const params: any = {
          CheckField: 'Name',
          Name: name?.trim?.(),
          Rules: { Process: { Cmdline: '', Exe: '' }, PProcess: {}, AProcess: {} },
          ...(type === 'create' && initParams?.EventId ? { EventId: initParams?.EventId } : {}),
        };
        const res: any = await CheckBashPolicyParams(params);
        if (res) {
          setIsNameCorrect(res?.ErrCode === 0);
          setNameErrMsg(res?.ErrMsg || '');
          if (res?.ErrCode !== 0) {
            toast.error(res?.ErrMsg || '参数填写错误');
          }
        }
      }
      setLoading(false);
    }
  };

  const handleSubmit = async () => {
    if (!policyName?.trim?.()) {
      toast.error('策略名称不能为空');
      return;
    }
    if (!/^[0-9a-zA-Z\u4e00-\u9fa5]+$/.test(policyName?.trim?.())) {
      toast.error('策略名称格式不正确，仅支持英文、数字、中文');
      return;
    }
    if (policyName?.trim?.()?.length > 20) {
      toast.error('策略名称不能超过20个字符');
      return;
    }
    if (policyDesc?.trim?.()?.length > 200) {
      toast.error('策略描述不能超过200个字符');
      return;
    }
    if (processList?.some?.((item: any) => !item?.path?.trim?.() && !item?.cmd?.trim?.())) {
      toast.error('请至少填写进程文件路径或进程命令行其中一项');
      return;
    }
    if (hostScope === '1' && !selectMachine?.length) {
      toast.error('请选择OpenClaw');
      return;
    }
    setLoading(true);
    const rules: any = Object.keys(PROCESS_TYPES_MAP).reduce((pre: any, cur) => {
      const obj = processList?.filter?.((item: { processType: string; }) => item?.processType === cur)?.[0];
      if (obj?.path?.trim?.() || obj?.cmd?.trim?.()) {
        pre[cur] = {
          Exe: Base64.encode(obj?.path?.trim?.()),
          Cmdline: Base64.encode(obj?.cmd?.trim?.()),
        };
      }
      return pre;
    }, {});
    if (type !== 'edit') {
      const checkRes: any = await CheckBashPolicyParams({
        CheckField: `Name,${Object.keys(PROCESS_TYPES_MAP)
          .filter(key =>
            processList?.some?.((item: { processType: string; path: string; cmd: string; }) => item?.processType === key && (item?.path?.trim?.() || item?.cmd?.trim?.())),
          )
          .join(',')}`,
        Name: policyName?.trim?.(),
        Rules: rules,
        ...(type === 'create' && initParams?.EventId ? { EventId: initParams?.EventId } : {}),
      });
      if (checkRes?.ErrCode !== 0) {
        setLoading(false);
        toast.error(checkRes?.ErrMsg || '参数填写错误');
        return;
      }
    }
    if (type === 'create') {
      setModalVisible(true);
      setModalResult('loading');
    }
    if (policyAction === '2') {
      delete rules?.Process?.Exe;
    }
    const params: any = {
      Name: policyName?.trim?.(),
      Category: type === 'create' ? 1 : selectItem?.Category ?? 1,
      Descript: policyDesc?.trim?.(),
      White: Number(whiteType),
      BashAction: Number(policyAction),
      Scope: hostScope == '0' ? (policyAction == '2' ? 2 : whiteType === '1' ? 3 : 1) : 0,
      Enable: isEnabled ? 1 : 0,
      Level: whiteType == '1' ? 0 : Number(policyLevel),
      Rules: rules,
    };
    if (type === 'edit') {
      params.Id = selectItem?.Id;
    }
    if (hostScope == '1') {
      params.Quuids = selectMachine;
    }
    if (policyAction == '1' && type === 'create') {
      params.DealOldEvents = isHandleOld ? 1 : 0;
    }
    if (
      (from === 'alarmList' || from === 'detail') &&
      type === 'create' &&
      initParams?.EventId &&
      (initParams?.MachineType == 1 || initParams?.MachineType == 2) &&
      (hostScope == '0' || (hostScope == '1' && selectMachine?.includes?.(initParams?.Quuids?.[0])))
    ) {
      params.EventId = initParams?.EventId;
    }
    const res: any = await ModifyBashPolicy({ Policy: params });
    if (res) {
      if (type === 'create') {
        setModalResult('success');
      } else {
        setVisible?.(false);
      }
      toast.success('操作成功');
    } else if (type === 'create') {
      setModalResult('fail');
      setErrMsg(res?.msg || res?.uiMsg || res?.message || '');
    }
    refreshTable?.();
    setLoading(false);
  };

  const getMachineTotal = async () => {
    const res: any = await DescribeMachineGeneral();
    setMachineStat(res || {});
  };

  // 编辑状态下，规范匹配内容select数据
  const modifyProcessList =
    !selectItem?.Rules ||
      !Object.keys(selectItem?.Rules)?.length ||
      !Object.keys(selectItem?.Rules)?.filter?.(key => selectItem?.Rules?.[key]?.Cmdline || selectItem?.Rules?.[key]?.Exe)
        ?.length
      ? [{ processType: PROCESS_TYPES[0].value, path: '', cmd: '' }]
      : Object.keys(selectItem?.Rules)
        ?.filter?.(key => selectItem?.Rules?.[key]?.Cmdline || selectItem?.Rules?.[key]?.Exe)
        ?.map?.(key => ({
          processType: key,
          path: selectItem?.Rules?.[key]?.Exe || '',
          cmd: selectItem?.Rules?.[key]?.Cmdline || '',
        }));

  // 编辑时，监听 policyAction 和 whiteType 变化，自动重置 processList
  useEffect(() => {
    if (visible) {
      if (type === 'edit') {
        if (policyAction === String(selectItem?.BashAction) && whiteType === String(selectItem?.White)) {
          setProcessList(modifyProcessList);
        } else {
          setProcessList([{ processType: PROCESS_TYPES[0].value, path: '', cmd: '' }]);
        }
      }
      if (type === 'create') {
        setProcessList([
          {
            processType: PROCESS_TYPES[0].value,
            path: initParams ? initParams?.Path : '',
            cmd: initParams ? initParams?.Cmd : '',
          },
        ]);
      }
    }
  }, [policyAction, whiteType, visible]);

  useEffect(() => {
    if (type === 'create' && policyName?.trim?.()) {
      window.clearTimeout((window as any).checkBashPolicyParamsTimer);
      (window as any).checkBashPolicyParamsTimer = window.setTimeout(() => {
        checkBashRuleParams(policyName?.trim?.());
      }, 500);
    }
  }, [policyName]);

  // 数据初始化
  useEffect(() => {
    if (visible) {
      setLoading(false);
      setIsHandleOld(type === 'create' ? true : String(selectItem?.DealOldEvents) === '1');
      setIsEnabled(type === 'create' ? true : String(selectItem?.Enable) === '1');
      setPolicyName(type === 'create' ? '' : selectItem?.Name || '');
      setPolicyDesc(type === 'create' ? '' : selectItem?.Descript || '');
      setWhiteType(
        type === 'create'
          ? initParams?.PolicyAction == 1
            ? '1'
            : '0'
          : String(selectItem?.BashAction) === '1'
            ? '1'
            : '0',
      );
      setHostScope('1');
      setSelectMachine(
        type === 'create'
          ? initParams?.PolicyAction == 2
            ? initParams?.Quuids
            : []
          : (selectItem?.Quuids || [])?.filter?.((id: any) => id),
      );
      setPolicyLevel(
        type === 'create' ? (initParams?.PolicyLevel ? initParams?.PolicyLevel : '1') : selectItem?.Level || '1',
      );
      setPolicyAction(
        type === 'create'
          ? initParams?.PolicyAction
            ? `${initParams?.PolicyAction}`
            : '0'
          : `${selectItem?.BashAction || '0'}`,
      );
      setProcessList(
        type === 'create'
          ? initParams
            ? [
              {
                processType: PROCESS_TYPES[0].value,
                path: initParams?.Path || '',
                cmd: initParams?.Cmd || '',
              },
            ]
            : [{ processType: PROCESS_TYPES[0].value, path: '', cmd: '' }]
          : type === 'edit'
            ? !selectItem?.Rules ||
              !Object.keys(selectItem?.Rules)?.length ||
              !Object.keys(selectItem?.Rules)?.filter?.(
                key => selectItem?.Rules?.[key]?.Cmdline || selectItem?.Rules?.[key]?.Exe,
              )?.length
              ? [{ processType: PROCESS_TYPES[0].value, path: '', cmd: '' }]
              : Object.keys(selectItem?.Rules)
                ?.filter?.(key => selectItem?.Rules?.[key]?.Cmdline || selectItem?.Rules?.[key]?.Exe)
                ?.map?.(key => ({
                  processType: key,
                  path: selectItem?.Rules?.[key]?.Exe || '',
                  cmd: selectItem?.Rules?.[key]?.Cmdline || '',
                }))
            : [{ processType: PROCESS_TYPES[0].value, path: '', cmd: '' }],
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
                  <a className="text-[var(--text-brand)] underline cursor-pointer" onClick={() => window.open(AUTHORIZE_ROUTE)}>升级版本</a>
                </AlertDescription>
              </Alert>

              {/* 基本信息 */}
              <section>
                <CardTitle as="h3" className="mb-3">基本信息</CardTitle>
                <div className="space-y-4">
                  {/* 策略名称 */}
                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-2 text-sm text-[var(--text-secondary)] font-normal">
                      <span className="text-[var(--text-danger)] mr-0.5">*</span>策略名称
                    </Label>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-1.5">
                        <Input
                          maxLength={20}
                          value={policyName}
                          onChange={e => setPolicyName(e.target.value)}
                          className="max-w-[600px]"
                          placeholder="请输入策略名称，限制 20 个字符以内"
                        />
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Info className="w-3.5 h-3.5 text-[var(--text-weak)] hover:text-[var(--text-secondary)] cursor-pointer" />
                          </TooltipTrigger>
                          <TooltipContent>
                            {`支持英文、数字、中文，限制 20 个字符以内${type === 'edit' ? '' : '，不支持重名'}`}
                          </TooltipContent>
                        </Tooltip>
                      </div>
                      {!isNameCorrect && policyName?.trim?.() && (
                        <div className="mt-1.5 inline-flex items-center gap-1 text-xs text-[var(--text-danger)]">
                          <AlertCircle className="w-3.5 h-3.5" />
                          <span>{nameErrMsg || '策略名称不正确，请检查'}</span>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* 策略描述 */}
                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-2 text-sm text-[var(--text-secondary)] font-normal">
                      策略描述
                    </Label>
                    <div className="flex-1 min-w-0">
                      <Input
                        value={policyDesc}
                        onChange={e => setPolicyDesc(e.target.value)}
                        disabled={String(selectItem?.Category) === '0' && String(selectItem?.BashAction) === '2'}
                        className="max-w-[600px]"
                        placeholder="请输入策略描述，限制 200 个字符以内"
                      />
                    </div>
                  </div>

                  {/* 开关 */}
                  <div className="flex items-center gap-3">
                    <Label className="w-[88px] shrink-0 text-sm text-[var(--text-secondary)] font-normal">
                      <span className="text-[var(--text-danger)] mr-0.5">*</span>开关
                    </Label>
                    <Switch checked={isEnabled} onCheckedChange={val => setIsEnabled(val)} />
                  </div>
                </div>
              </section>

              {/* 策略详情 */}
              <section>
                <CardTitle as="h3" className="mb-3">策略详情</CardTitle>
                <div className="space-y-4">
                  {/* 黑/白名单 */}
                  <div className="flex items-center gap-3">
                    <Label className="w-[88px] shrink-0 text-sm text-[var(--text-secondary)] font-normal">
                      <span className="text-[var(--text-danger)] mr-0.5">*</span>黑/白名单
                    </Label>
                    <RadioGroup
                      value={whiteType}
                      onValueChange={value => {
                        setWhiteType(value);
                        setPolicyAction(value === '0' ? '0' : '1');
                      }}
                      disabled={String(selectItem?.Category) === '0' && String(selectItem?.BashAction) === '2'}
                      className="flex items-center gap-4"
                    >
                      <div className="flex items-center space-x-2">
                        <RadioGroupItem value="0" id="whiteType-0" />
                        <Label htmlFor="whiteType-0" className="text-sm text-[var(--text-title)] font-normal cursor-pointer">黑名单</Label>
                      </div>
                      <div className="flex items-center space-x-2">
                        <RadioGroupItem value="1" id="whiteType-1" />
                        <Label htmlFor="whiteType-1" className="text-sm text-[var(--text-title)] font-normal cursor-pointer">白名单</Label>
                      </div>
                    </RadioGroup>
                  </div>

                  {/* 执行动作 */}
                  {whiteType === '0' && (
                    <div className="flex items-start gap-3">
                      <Label className="w-[88px] shrink-0 pt-1.5 text-sm text-[var(--text-secondary)] font-normal">
                        <span className="text-[var(--text-danger)] mr-0.5">*</span>执行动作
                      </Label>
                      <div className="flex-1 min-w-0 flex flex-wrap items-center gap-3">
                        {(() => {
                          const isLocked = String(selectItem?.Category) === '0' && String(selectItem?.BashAction) === '2';
                          const actionItems = [
                            { value: '0', label: '告警', disabled: false },
                            { value: '2', label: '拦截', disabled: !hasFlagship },
                          ];
                          return (
                            <div className={`inline-flex items-center gap-0.5 p-0.5 bg-[var(--bg-grey-subtle)] rounded-[4px] ${isLocked ? 'pointer-events-none opacity-50' : ''}`}>
                              {actionItems.map(it => {
                                const isActive = policyAction === it.value;
                                const btn = (
                                  <button
                                    key={it.value}
                                    type="button"
                                    disabled={it.disabled}
                                    onClick={() => { if (!it.disabled) setPolicyAction(it.value); }}
                                    className={`px-3 py-1 text-xs rounded-[3px] transition-colors leading-5 ${
                                      isActive
                                        ? 'bg-white text-[var(--text-title)] font-medium'
                                        : it.disabled
                                          ? 'text-[var(--text-weak)] cursor-not-allowed font-normal'
                                          : 'text-[var(--text-muted)] hover:text-[var(--text-title)] font-normal'
                                    }`}
                                    style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                                  >
                                    {it.label}
                                  </button>
                                );
                                if (it.disabled) {
                                  return (
                                    <Tooltip key={it.value}>
                                      <TooltipTrigger asChild><span>{btn}</span></TooltipTrigger>
                                      <TooltipContent>
                                        <span>
                                          当前暂无旗舰版 OpenClaw，无法设置拦截策略，可
                                          <a
                                            onClick={() => window.open(AUTHORIZE_ROUTE)}
                                            className="text-white underline cursor-pointer"
                                          >点击升级版本</a>
                                        </span>
                                      </TooltipContent>
                                    </Tooltip>
                                  );
                                }
                                return btn;
                              })}
                            </div>
                          );
                        })()}
                        <MetaText>
                          {policyAction == '0'
                            ? '当发现 OpenClaw 存在威胁命令时，将产生告警。'
                            : policyAction == '1'
                              ? '当发现 OpenClaw 存在威胁命令时，将不再产生告警或拦截行为。'
                              : '当发现 OpenClaw 存在威胁命令时，将对威胁命令运行进行自动拦截，并产生拦截记录。'}
                        </MetaText>
                      </div>
                    </div>
                  )}

                  {/* 威胁等级 */}
                  {whiteType === '0' && (
                    <div className="flex items-center gap-3">
                      <Label className="w-[88px] shrink-0 text-sm text-[var(--text-secondary)] font-normal">
                        <span className="text-[var(--text-danger)] mr-0.5">*</span>威胁等级
                      </Label>
                      <div>
                        {(() => {
                          const isLocked = String(selectItem?.Category) === '0' && String(selectItem?.BashAction) === '2';
                          const levelItems = [
                            { value: '1', label: '高危' },
                            { value: '2', label: '中危' },
                            { value: '3', label: '低危' },
                          ];
                          return (
                            <div className={`inline-flex items-center gap-0.5 p-0.5 bg-[var(--bg-grey-subtle)] rounded-[4px] ${isLocked ? 'pointer-events-none opacity-50' : ''}`}>
                              {levelItems.map(it => {
                                const isActive = policyLevel === it.value;
                                return (
                                  <button
                                    key={it.value}
                                    type="button"
                                    onClick={() => setPolicyLevel(it.value)}
                                    className={`px-3 py-1 text-xs rounded-[3px] transition-colors leading-5 ${
                                      isActive
                                        ? 'bg-white text-[var(--text-title)] font-medium'
                                        : 'text-[var(--text-muted)] hover:text-[var(--text-title)] font-normal'
                                    }`}
                                    style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                                  >
                                    {it.label}
                                  </button>
                                );
                              })}
                            </div>
                          );
                        })()}
                      </div>
                    </div>
                  )}

                  <Alert>
                    <AlertDescription>
                      <ul className="list-disc pl-4 space-y-0.5">
                        <li>为提升告警精准度，已优化规则配置：</li>
                        <li>当设置父进程/进程路径规则时，系统将停止匹配 bash 历史日志，减少干扰确保告警准确有效。</li>
                        <li>【进程文件路径】进程文件所在的路径，如 curl，进程文件路径是 /usr/bin/curl</li>
                        <li>【进程命令行】启动进程的命令行，如 curl http://127.0.0.1:80</li>
                      </ul>
                    </AlertDescription>
                  </Alert>

                  {/* 匹配内容 */}
                  <div className="flex items-start gap-3">
                    <Label className="w-[88px] shrink-0 pt-1.5 text-sm text-[var(--text-secondary)] font-normal">
                      <span className="text-[var(--text-danger)] mr-0.5">*</span>匹配内容
                    </Label>
                    <div className="flex-1 min-w-0">
                      <div className="border border-[var(--border)] rounded-[4px] overflow-hidden">
                        <Table density="compact">
                          <TableHeader>
                            <TableRow>
                              <TableHead style={{ width: 120 }}>进程类型</TableHead>
                              {policyAction !== '2' && (
                                <TableHead>进程文件路径</TableHead>
                              )}
                              <TableHead>进程命令行</TableHead>
                              {!(whiteType === '0' && policyAction === '2') && (
                                <TableHead style={{ width: 80 }}>操作</TableHead>
                              )}
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {processList?.map?.((item: any, rowIndex: number) => (
                              <TableRow key={rowIndex}>
                                <TableCell className="align-top" style={{ width: 120 }}>
                                  <Select
                                    value={item?.processType}
                                    onValueChange={val => {
                                      const newList: any = [...processList];
                                      newList[rowIndex].processType = val;
                                      setProcessList(newList);
                                    }}
                                  >
                                    <SelectTrigger size="sm" className="w-full">
                                      <SelectValue placeholder="请选择进程类型" />
                                    </SelectTrigger>
                                    <SelectContent>
                                      {(whiteType === '0' && policyAction === '2'
                                        ? PROCESS_TYPES.slice(0, 1)
                                        : PROCESS_TYPES
                                      ).map(option => (
                                        <SelectItem
                                          key={option.value}
                                          value={option.value}
                                          disabled={
                                            whiteType === '0' && policyAction === '2'
                                              ? false
                                              : processList?.map?.((p: { processType: any; }) => p?.processType)?.indexOf?.(option?.value) > -1
                                          }
                                        >
                                          {option.text || option.value}
                                        </SelectItem>
                                      ))}
                                    </SelectContent>
                                  </Select>
                                </TableCell>
                                {policyAction !== '2' && (
                                  <TableCell className="align-top">
                                    <Textarea
                                      maxLength={1024}
                                      value={item?.path || ''}
                                      onChange={e => {
                                        const val = e.target.value;
                                        const newList: any = [...processList];
                                        newList[rowIndex].path = val;
                                        setProcessList(newList);
                                        const len = Math.ceil(val?.length / MAX_TEXT_LEN);
                                        setProcessHeights([
                                          ...processHeights?.slice?.(0, rowIndex),
                                          len > 12 ? heightMap[12] + (len - 12) * 16 : heightMap[len],
                                          ...processHeights?.slice?.(rowIndex + 1),
                                        ]);
                                      }}
                                      style={{ width: '100%', resize: 'vertical', height: processHeights[rowIndex] || 30 }}
                                      placeholder={`请输入${PROCESS_TYPES_MAP[item?.processType]}文件路径`}
                                    />
                                  </TableCell>
                                )}
                                <TableCell className="align-top">
                                  <Textarea
                                    maxLength={1024}
                                    value={item?.cmd || ''}
                                    onChange={e => {
                                      const val = e.target.value;
                                      const newList = [...processList];
                                      newList[rowIndex].cmd = val;
                                      setProcessList(newList);
                                      const len = Math.ceil(val?.length / MAX_TEXT_LEN);
                                      setProcessCmdHeights([
                                        ...processCmdHeights?.slice?.(0, rowIndex),
                                        len > 12 ? heightMap[12] + (len - 12) * 16 : heightMap[len],
                                        ...processCmdHeights?.slice?.(rowIndex + 1),
                                      ]);
                                    }}
                                    style={{ width: '100%', resize: 'vertical', height: processCmdHeights[rowIndex] || 30 }}
                                    placeholder={`请输入${PROCESS_TYPES_MAP[item?.processType]}命令行`}
                                  />
                                </TableCell>
                                {!(whiteType === '0' && policyAction === '2') && (
                                  <TableCell className="align-top" style={{ width: 80 }}>
                                    <div className="inline-flex items-center gap-2">
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <button
                                            type="button"
                                            className={`p-1 rounded transition-colors ${
                                              processList?.length >= 3
                                                ? 'opacity-40 cursor-not-allowed text-[var(--text-weak)]'
                                                : 'cursor-pointer text-[var(--text-brand)] hover:bg-[var(--bg-grey-subtle)]'
                                            }`}
                                            onClick={() => {
                                              if (processList?.length < 3) {
                                                const allProcessTypeList = PROCESS_TYPES.map(t => t?.value);
                                                const newType = allProcessTypeList.filter(
                                                  key => !processList?.map?.((p: { processType: any; }) => p?.processType)?.includes?.(key),
                                                )?.[0];
                                                setProcessList(
                                                  processList?.concat?.({ processType: newType, path: '', cmd: '' }),
                                                );
                                              }
                                            }}
                                          >
                                            <Plus className="w-4 h-4" />
                                          </button>
                                        </TooltipTrigger>
                                        {processList?.length >= 3 && (
                                          <TooltipContent>最多可添加三行</TooltipContent>
                                        )}
                                      </Tooltip>
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <button
                                            type="button"
                                            className={`p-1 rounded transition-colors ${
                                              processList?.length <= 1
                                                ? 'opacity-40 cursor-not-allowed text-[var(--text-weak)]'
                                                : 'cursor-pointer text-[var(--text-secondary)] hover:bg-[var(--bg-grey-subtle)] hover:text-[var(--text-danger)]'
                                            }`}
                                            onClick={() => {
                                              if (processList?.length > 1) {
                                                setProcessList(processList?.filter?.((_: any, i: number) => i !== rowIndex));
                                              }
                                            }}
                                          >
                                            <Trash2 className="w-4 h-4" />
                                          </button>
                                        </TooltipTrigger>
                                        {processList?.length <= 1 && (
                                          <TooltipContent>至少需添加一行</TooltipContent>
                                        )}
                                      </Tooltip>
                                    </div>
                                  </TableCell>
                                )}
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                      <MetaText className="block mt-1.5">
                        OpenClaw 无法识别 alias 命令，请输入最终执行命令的正则表达式
                      </MetaText>
                    </div>
                  </div>

                  {/* 告警处理 */}
                  {whiteType === '1' && type === 'create' && (
                    <div className="flex items-center gap-3">
                      <Label className="w-[88px] shrink-0 text-sm text-[var(--text-secondary)] font-normal">
                        <span className="text-[var(--text-danger)] mr-0.5">*</span>告警处理
                      </Label>
                      <div className="flex items-center space-x-2">
                        <Checkbox id="handleOld" checked={isHandleOld} onCheckedChange={value => setIsHandleOld(!!value)} />
                        <Label htmlFor="handleOld" className="text-sm text-[var(--text-title)] font-normal cursor-pointer">
                          对符合本规则的历史"待处理"告警执行加白操作
                        </Label>
                      </div>
                    </div>
                  )}
                </div>
              </section>

              {/* 生效 OpenClaw 范围 */}
              <section>
                <div className="flex items-center justify-between mb-3">
                  <CardTitle as="h3">
                    生效 OpenClaw 范围
                  </CardTitle>
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
                      // 与原 CvmSelectComponent 的 isEnable 取反：
                      // - 拦截：仅旗舰版可选
                      // - 白名单：全部可选
                      // - 告警/其他：旗舰版/PRO/通用折扣可选
                      if (policyAction == '2') return h?.ProtectType !== 'Flagship';
                      if (whiteType === '1') return false;
                      return !(
                        h?.ProtectType === 'Flagship' ||
                        h?.ProtectType === 'PRO_VERSION' ||
                        h?.ProtectType === 'GENERAL_DISCOUNT'
                      );
                    }}
                    renderDisabledTrigger={(_h: any, defaultCheckbox) => (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="inline-flex">{defaultCheckbox}</span>
                        </TooltipTrigger>
                        <TooltipContent>基础版资产请升级到旗舰版以使用该能力</TooltipContent>
                      </Tooltip>
                    )}
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
            </div>
          </DrawerBody>
          <DrawerFooter className="flex flex-row items-center justify-end gap-3 p-4 bg-background border-t border-[var(--border)]">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setVisible?.(false)} className="min-w-[80px]">
              {'取消'}
            </Button>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="dialog-confirm"
                    size="claw-sm"
                    disabled={!policyName?.trim?.() || loading || fetchLoading}
                    onClick={handleSubmit}
                    className="min-w-[80px]"
                  >
                    {'保存'}
                  </Button>
                </span>
              </TooltipTrigger>
              {!policyName?.trim?.() && (
                <TooltipContent>{'未设置必填项'}</TooltipContent>
              )}
            </Tooltip>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      <Dialog
        open={modalVisible}
        onOpenChange={open => {
          if (!open) {
            setModalVisible?.(false);
            setVisible?.(false);
          }
        }}
      >
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>
              {modalResult === 'loading'
                ? '策略创建中'
                : modalResult === 'success'
                  ? '策略创建成功'
                  : '策略创建失败'}
            </DialogTitle>
          </DialogHeader>
          <div className="text-sm text-[var(--text-secondary)] leading-relaxed">
            {modalResult === 'loading' && (
              <span className="inline-flex items-center gap-2 text-[var(--text-title)]">
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
              from === 'alarmList' || from === 'detail' ? (
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={() => {
                    setModalVisible?.(false);
                    setVisible?.(false);
                    // refreshTable?.();
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
                      setProcessList([{ processType: PROCESS_TYPES[0].value, path: '', cmd: '' }]);
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
