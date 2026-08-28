/*  */
/* eslint-disable  */

import React, { useEffect, useRef, useState } from "react";
import {
  Tag as TagIcon,
  Loader2,
  X,
} from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Button } from "@/components/ui/button";
import {
  PanelTitle,
  SectionTitle,
  BodyText,
  MetaText,
  CodeText,
} from "@/components/ui/Typography";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerClose,
  DrawerBody,
} from "@/components/ui/drawer";
import { DescribeMachines } from "@/pages/admin/Security/api";

import {
  PROTECTTYPE_VERSION_TYPES,
  ProtectLevelMap,
} from "../../constants";
import { setCookie } from "../../Common/cookieUtil";
import { getRequestParams } from "../../Common/CommonRiskHandleFunc";

import {
  getPolicyActionMap,
  RulesAttributeMap,
  GetHostTypeText,
  BASH_DETAIL_TORULE,
} from "./Constants";
import { getRuleLevelText } from "./BashPolicyList";

export function PolicyDetailDrawer({
  loading = undefined,
  from = undefined,
  selectItem,
  detailVisible,
  setDetailVisible,
  hasFlagship,
  setHandleType,
  setSettingVisible,
  handleDelPolicy,
  handleSwitchChange,
  aiAgentHostList,
}: any) {
  const PAGE_SIZE = 10;

  const [isEdit, setIsEdit] = useState(false);
  const [machineData, setMachineData] = useState<any[]>([]);
  const [machinePage, setMachinePage] = useState(1);
  const [machineSearch, setMachineSearch] = useState("");
  const [machineLoading, setMachineLoading] = useState(false);

  const requestIdRef = useRef(0);

  const fetchMachines = async (page: number, keywords?: string) => {
    if (!selectItem?.Quuids?.length) {
      setMachineData([]);
      return;
    }
    const rid = ++requestIdRef.current;
    setMachineLoading(true);
    try {
      const offset = (page - 1) * PAGE_SIZE;
      const query: any = {
        Offset: offset,
        Limit: PAGE_SIZE,
        MachineRegion: "all-regions",
        MachineType: "ALL",
      };
      if (keywords) {
        query.Keywords = keywords;
      }
      const params = getRequestParams(query, [
        { key: "Keywords", type: "string" },
        { key: "Quuid", type: "string" },
      ]);
      if (!params?.Filters) {
        params.Filters = [];
      }
      params?.Filters?.push?.({
        Name: "Quuid",
        Values: selectItem?.Quuids?.slice?.(offset, offset + PAGE_SIZE),
      });
      if (selectItem?.BashAction == 2) {
        params?.Filters?.push?.({
          Name: "Version",
          Values: ["Flagship"],
        });
      } else if (String(selectItem?.White) !== "1") {
        params?.Filters?.push?.({
          Name: "Version",
          Values: ["ProtectedMachines"],
        });
      }
      const resp: any = await DescribeMachines({
        ...(params || {}),
        Offset: 0,
        Limit: PAGE_SIZE,
      });
      if (rid === requestIdRef.current) {
        setMachineData(
          resp?.Machines?.map?.((d: any) => ({
            ...d,
            OpenClawName: aiAgentHostList?.find?.(
              (a: any) => a?.InstanceID === d?.MachineExtraInfo?.InstanceID
            )?.OpenClawName,
          })) || []
        );
      }
    } catch {
      // error handled by interceptor
    } finally {
      if (rid === requestIdRef.current) {
        setMachineLoading(false);
      }
    }
  };

  const refreshTable = () => {
    fetchMachines(machinePage, machineSearch);
  };

  useEffect(() => {
    if (detailVisible && isEdit) {
      refreshTable();
      setIsEdit(false);
    }
  }, [selectItem?.Quuids]);

  useEffect(() => {
    if (detailVisible) {
      setIsEdit(false);
      setMachinePage(1);
      setMachineSearch("");
      fetchMachines(1, "");
    }
  }, [detailVisible]);

  useEffect(() => {
    if (detailVisible && String(selectItem?.Scope) === "0") {
      fetchMachines(machinePage, machineSearch);
    }
  }, [machinePage]);

  return (
    <Drawer
      open={detailVisible}
      onOpenChange={open => {
        if (!open) setDetailVisible?.(false);
      }}
      direction="right"
    >
      <DrawerContent className="data-[vaul-drawer-direction=right]:w-[760px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
        <DrawerHeader className="flex flex-row items-start justify-between gap-4 p-4 bg-background text-left">
          <div className="flex-1 min-w-0 space-y-3">
            <div className="flex items-center gap-3 flex-wrap">
              <DrawerTitle asChild>
                <PanelTitle as="h2" className="truncate">
                  {`策略详情：${selectItem?.Name ?? ""}`}
                </PanelTitle>
              </DrawerTitle>
              {String(selectItem?.Enable) === "1" ? (
                <span className="badge-running">启用中</span>
              ) : (
                <span className="badge-shutdown">未启用</span>
              )}
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              {from === "eventDetail" ? (
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={() => {
                    setCookie(
                      BASH_DETAIL_TORULE,
                      `${selectItem?.Name},${selectItem?.Id}`,
                      0.1
                    );
                    // history.replace(createBashRoute('user_rule'));
                  }}
                >
                  {"编辑"}
                </Button>
              ) : (
                <>
                  {String(selectItem?.Category) === "0" &&
                  String(selectItem?.BashAction) !== "2" ? null : String(
                      selectItem?.BashAction
                    ) === "2" && !hasFlagship ? null : (
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="claw-primary" size="claw-sm" disabled={loading}>
                          {`${String(selectItem?.Enable) === "1" ? "关闭" : "开启"}策略`}
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent className="sm:max-w-[420px]">
                        <AlertDialogHeader>
                          <AlertDialogTitle>{`确定${String(selectItem?.Enable) === "1" ? "关闭" : "开启"}此策略？`}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {String(selectItem?.Enable) === "1"
                              ? "确认后，将关闭此策略，后续命中策略内容时，将不再执行相应动作，请谨慎操作。"
                              : "确认后，将开启此策略，后续命中策略内容时，将对应执行相应动作。"}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{"取消"}</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => handleSwitchChange(selectItem)}
                          >
                            {"确定"}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  )}
                  {String(selectItem?.BashAction) === "2" &&
                  !hasFlagship ? null : String(selectItem?.Category) === "0" ? (
                    String(selectItem?.BashAction) !== "2" ? null : (
                      <Button
                        variant="claw-outline"
                        size="claw-sm"
                        onClick={() => {
                          setIsEdit(true);
                          setHandleType?.("edit");
                          setSettingVisible?.(true);
                        }}
                      >
                        {"编辑"}
                      </Button>
                    )
                  ) : (
                    <Button
                      variant="claw-outline"
                      size="claw-sm"
                      onClick={() => {
                        setIsEdit(true);
                        setHandleType?.("edit");
                        setSettingVisible?.(true);
                      }}
                    >
                      {"编辑"}
                    </Button>
                  )}
                  {String(selectItem?.Category) === "1" && (
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="claw-outline" size="claw-sm">
                          {"删除"}
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent className="sm:max-w-[420px]">
                        <AlertDialogHeader>
                          <AlertDialogTitle>
                            {"确认删除此策略？"}
                          </AlertDialogTitle>
                          <AlertDialogDescription>
                            {
                              "确认后，策略将被删除，无法恢复，策略范围内的资产将不再生效，请谨慎操作。"
                            }
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{"取消"}</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => {
                              setDetailVisible?.(false);
                              handleDelPolicy(selectItem);
                            }}
                          >
                            {"确定"}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  )}
                </>
              )}
            </div>
          </div>
          <DrawerClose asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950 shrink-0"
              aria-label="关闭"
            >
              <X className="w-4 h-4" />
            </Button>
          </DrawerClose>
        </DrawerHeader>
        <DrawerBody>
          <div className="px-5 py-4 space-y-6">
            {/* 基本信息 */}
            <section>
              <SectionTitle as="h3" className="!text-sm !font-semibold mb-3">
                基本信息
              </SectionTitle>
              <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden divide-y divide-[#E5E5E5]">
                <div className="grid grid-cols-2 divide-x divide-[#E5E5E5]">
                  <div className="flex items-center gap-3 px-3 py-2.5">
                    <MetaText className="w-[88px] flex-shrink-0">策略名称</MetaText>
                    <BodyText as="span" className="flex-1 min-w-0 truncate" title={selectItem?.Name}>
                      {selectItem?.Name || '--'}
                    </BodyText>
                  </div>
                  <div className="flex items-center gap-3 px-3 py-2.5">
                    <MetaText className="w-[88px] flex-shrink-0">最近编辑时间</MetaText>
                    <BodyText as="span" className="flex-1 min-w-0">
                      {selectItem?.ModifyTime || '--'}
                    </BodyText>
                  </div>
                </div>
                <div className="flex items-start gap-3 px-3 py-2.5">
                  <MetaText className="w-[88px] flex-shrink-0">策略描述</MetaText>
                  <BodyText as="span" className="flex-1 min-w-0">
                    {selectItem?.Descript || '--'}
                  </BodyText>
                </div>
              </div>
            </section>

            {/* 拦截策略详情 */}
            <section>
              <SectionTitle as="h3" className="!text-sm !font-semibold mb-3">
                拦截策略详情
              </SectionTitle>
              <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden divide-y divide-[#E5E5E5]">
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <MetaText className="w-[88px] flex-shrink-0">黑/白名单</MetaText>
                  <span className={`text-sm ${
                    String(selectItem?.BashAction) === "1" ? "text-[#16A34A]" : "text-[#DC2626]"
                  }`}>
                    {String(selectItem?.BashAction) === "1" ? "白名单" : "黑名单"}
                  </span>
                </div>
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <MetaText className="w-[88px] flex-shrink-0">执行动作</MetaText>
                  {(() => {
                    const action = String(selectItem?.BashAction);
                    const color = action === '2' ? '#DC2626' // 拦截
                      : action === '0' ? '#D97706' // 告警
                        : action === '1' ? '#16A34A' // 放行
                          : '#525252';
                    return (
                      <span className="inline-flex items-center gap-1.5 text-sm" style={{ color }}>
                        <span className="w-1.5 h-1.5 rounded-full" style={{ background: color }} />
                        {getPolicyActionMap()?.[selectItem?.BashAction] || '--'}
                      </span>
                    );
                  })()}
                </div>
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <MetaText className="w-[88px] flex-shrink-0">威胁等级</MetaText>
                  <div>{getRuleLevelText(selectItem?.Level)}</div>
                </div>
                <div className="flex items-start gap-3 px-3 py-2.5">
                  <MetaText className="w-[88px] flex-shrink-0 pt-1.5">匹配内容</MetaText>
                  <div className="flex-1 min-w-0">
                    <div className="rounded-[4px] border border-[#E5E5E5] overflow-hidden">
                      <Table density="compact">
                        <TableHeader>
                          <TableRow>
                            <TableHead style={{ width: 100 }}>进程类型</TableHead>
                            {String(selectItem?.BashAction) !== "2" && (
                              <TableHead>进程文件路径</TableHead>
                            )}
                            <TableHead>进程命令行</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {(
                            Object.keys(selectItem?.Rules || {})
                              ?.filter?.(
                                key =>
                                  selectItem?.Rules?.[key]?.Cmdline ||
                                  selectItem?.Rules?.[key]?.Exe
                              )
                              ?.map?.(key => ({
                                type: key,
                                text: RulesAttributeMap?.[key],
                                path: selectItem?.Rules?.[key]?.Exe,
                                cmd: selectItem?.Rules?.[key]?.Cmdline,
                              })) || []
                          ).map((row: any) => (
                            <TableRow key={row.type}>
                              <TableCell style={{ width: 100 }}>{row.text}</TableCell>
                              {String(selectItem?.BashAction) !== "2" && (
                                <TableCell className="break-all">
                                  <CodeText tone="emphasis">{row.path}</CodeText>
                                </TableCell>
                              )}
                              <TableCell className="break-all">
                                <CodeText tone="emphasis">{row.cmd}</CodeText>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            {/* 生效 OpenClaw 范围 */}
            <section>
              <SectionTitle as="h3" className="!text-sm !font-semibold mb-3">
                {`生效OpenClaw范围${String(selectItem?.Scope) !== "0" ? "" : `（${selectItem?.Quuids?.length || 0}）`}`}
              </SectionTitle>
              {String(selectItem?.Scope) !== "0" ? (
                <div className="rounded-[4px] border border-[#E5E5E5] px-3 py-2.5 text-sm text-[#0A0A0A]">
                  {GetHostTypeText(selectItem?.Scope)}
                </div>
              ) : (
                <div className="relative rounded-[4px] border border-[#E5E5E5] overflow-hidden bg-white">
                  {machineLoading && (
                    <div className="absolute inset-0 bg-white/60 z-10 flex items-center justify-center">
                      <Loader2 className="w-6 h-6 animate-spin text-[#1447E6]" />
                    </div>
                  )}
                  <Table density="compact">
                    <TableHeader>
                      <TableRow>
                        <TableHead>Agent 名称 / ID</TableHead>
                        <TableHead>标签</TableHead>
                        <TableHead>版本状态</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {machineData.length === 0 && !machineLoading && (
                        <TableRow>
                          <TableCell colSpan={3}>
                            <Empty className="border-none py-6">
                              <EmptyHeader>
                                <EmptyMedia />
                                <EmptyTitle>暂无数据</EmptyTitle>
                              </EmptyHeader>
                            </Empty>
                          </TableCell>
                        </TableRow>
                      )}
                      {machineData.map((item: any) => (
                        <TableRow key={item?.Quuid}>
                          <TableCell className="align-top">
                            <div className="font-medium">{item?.OpenClawName || '-'}</div>
                            <CodeText tone="secondary" className="block mt-0.5">
                              {item?.MachineExtraInfo?.InstanceID || '-'}
                            </CodeText>
                          </TableCell>
                          <TableCell className="align-top">
                            {(item?.Tag?.length || 0) + (item?.CloudTags?.length || 0) > 0 ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-[4px] border border-[#E5E5E5] bg-white text-xs text-[#525252] cursor-pointer hover:border-[#1447E6]">
                                    <TagIcon className="w-3 h-3" />
                                    {(item?.Tag?.length || 0) + (item?.CloudTags?.length || 0) === 1 ? '标签' : '多个'}
                                    <span className="text-[#1447E6]">
                                      ({(item?.Tag ?? [])?.length + (item?.CloudTags ?? [])?.length})
                                    </span>
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent>
                                  <div className="max-h-[500px] overflow-y-auto">
                                    <p className="font-semibold">腾讯云标签</p>
                                    {item?.CloudTags?.map?.((data: any, index: number) => (
                                      <div key={index} className="max-w-[300px] break-all mt-0.5">
                                        {`${data?.TagKey}:${data?.TagValue}`}
                                      </div>
                                    ))}
                                    {!item?.CloudTags?.length && (
                                      <span className="text-[var(--text-weak)]">暂无腾讯云标签</span>
                                    )}
                                    <p className="font-semibold mt-2.5 mb-1">OpenClaw 标签</p>
                                    {item?.Tag?.map?.((data: any, index: number) => (
                                      <div key={index} className="max-w-[300px] break-all mt-0.5">
                                        {data?.Name}
                                      </div>
                                    ))}
                                    {!item?.Tag?.length && (
                                      <span className="text-[var(--text-weak)]">暂无 OpenClaw 标签</span>
                                    )}
                                  </div>
                                </TooltipContent>
                              </Tooltip>
                            ) : (
                              <span className="text-[var(--text-weak)]">暂无标签</span>
                            )}
                          </TableCell>
                          <TableCell className="align-top">
                            {ProtectLevelMap[
                              String(PROTECTTYPE_VERSION_TYPES[item?.ProtectType])
                            ] || '未安装'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>

                  {/* 表格页脚：数量统计左对齐，分页器右对齐 */}
                  <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]">
                    <span className="justify-self-start text-sm leading-[1.5] text-[#737373]">
                      共 {selectItem?.Quuids?.length || 0} 条记录
                    </span>
                    <Pagination
                      total={selectItem?.Quuids?.length || 0}
                      current={machinePage}
                      pageSize={PAGE_SIZE}
                      size="small"
                      className="justify-self-end justify-end flex-nowrap"
                      onChange={(p) => setMachinePage(p)}
                    />
                  </div>
                </div>
              )}
            </section>
          </div>
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}
