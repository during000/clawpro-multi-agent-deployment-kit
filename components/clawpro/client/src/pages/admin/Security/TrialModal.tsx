import React, { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  DescribeOrderList,
  DescribeLogStorageConfig,
  CreateWhiteListOrder,
  ModifyLicenseBinds,
  ModifyLogStorageConfig,
  DescribeLicenseBindSchedule,
  ScanAsset,
  // ApplyTrial,
  // CreateScanTask,
} from "@/pages/admin/Security/api";

import BindErrModal from "./AIAgent/Common/BindErrModal";

function LogStatusTag({ enabled }: { enabled: boolean }) {
  return (
    <Badge variant="outline" className="gap-1.5">
      <span
        className={`inline-block h-1.5 w-1.5 rounded-full ${enabled ? "bg-green-500" : "bg-yellow-500"
          }`}
      />
      {enabled ? "已开启" : "未开启"}
    </Badge>
  );
}

export default function TrialModal({
  openTrialModalVisible,
  setOpenTrialModalVisible,
  setHasTrialNum,
  aiAgentHostList,
  getAllMachines,
  isTrialLoading,
  setIsTrialLoading,
  isGetAllMachinesLoading,
  trialDays,
  checkIfHasTrial,
}: any) {
  const [hasBuyLog, setHasBuyLog] = useState(false);
  const [logConfigData, setLogConfigData] = useState({} as any);
  const [bindErrVisible, setBindErrVisible] = useState(false);
  const [bindTaskId, setBindTaskId] = useState(0);
  const [initTotal, setInitTotal] = useState(0);
  const [initErrData, setInitErrData] = useState([]);

  const checkIfHasBuyLog = async () => {
    const res: any = await Promise.all([
      DescribeOrderList({
        Offset: 0,
        Limit: 1,
        Filters: [
          { Name: 'SourceType', Values: ['15'] },
          { Name: 'InquireKey', Values: ['sv_yunjing_vas_la'] },
        ],
      }),
      DescribeLogStorageConfig(),
    ]);
    setLogConfigData(res?.[1] || {});
    setHasBuyLog(res?.[0]?.TotalCount > 0);
  };

  const createTrial = async () => {
    setIsTrialLoading(true);
    const basicHostCount = aiAgentHostList?.filter?.((d: any) => d?.ProtectType === 'BASIC_VERSION')?.length;
    const LicenseNum = basicHostCount ? basicHostCount : aiAgentHostList?.length;
    const params = {
      LicenseType: 2,
      LicenseNum,
      Deadline: trialDays,
      SourceType: 15,
      RegionId: 1,
      RuleName: "csip_yunying",
      Path: `AIAgentTrial+${window.location.pathname}`,
      ExtraParam: JSON.stringify({ LicenseNum }),
    };
    if (LicenseNum > 0) {
      const res: any = await CreateWhiteListOrder(params);
      if (res?.Resource) {
        await ModifyLogStorageConfig({
          Type: Array.from(
            new Set([
              ...(logConfigData?.Type || []),
              "system_audit_log",
              "dns_log",
              "process_snapshot",
              "file_log",
            ])
          ),
          Period:
            typeof logConfigData?.Period === "number" &&
              logConfigData?.Period > 0
              ? logConfigData?.Period
              : 3640,
          IsModifyPeriod: false,
          Granularity:
            typeof logConfigData?.Granularity === "string" &&
              logConfigData?.Granularity
              ? logConfigData?.Granularity
              : "day",
          MsgLanguage:
            typeof logConfigData?.MsgLanguage === "string" &&
              logConfigData?.MsgLanguage
              ? logConfigData?.MsgLanguage
              : "zh",
        });
        const quuids = aiAgentHostList?.filter?.((d: any) => d?.ProtectType === 'BASIC_VERSION' && d?.Quuid)?.map?.((d: any) => d?.Quuid);
        const bindInfo = {
          ResourceId: res?.Resource?.ResourceId,
          LicenseType: 2,
          IsAll: false,
          QuuidList: quuids,
        };
        if (quuids?.length > 0) {
          const bindRes: any = await ModifyLicenseBinds(bindInfo);
          if (bindRes?.TaskId) {
            setBindTaskId?.(bindRes?.TaskId);
            // toast.loading("绑定授权中...");
            const start = Date.now();
            (window as any).CsipAiAgentBindTimer = window.setInterval(
              async () => {
                try {
                  const data: any = await DescribeLicenseBindSchedule({ TaskId: bindRes?.TaskId });
                  if (data?.Schedule >= 100) {
                    window.clearInterval((window as any).CsipAiAgentBindTimer);
                    const err: any = await DescribeLicenseBindSchedule({
                      TaskId: bindRes?.TaskId,
                      Offset: 0,
                      Limit: 10,
                      Filters: [{ Name: "Status", Values: ["2"] }],
                    });
                    if (err?.TotalCount > 0) {
                      setInitTotal?.(err?.TotalCount ?? 0);
                      setInitErrData?.(err?.List ?? []);
                      setBindErrVisible?.(true);
                    }
                    await Promise.all([
                      ScanAsset({ Quuids: [], AssetTypeIds: [17] }),
                      // CreateScanTask(),
                      getAllMachines(),
                    ]);
                    setOpenTrialModalVisible(false);
                    setHasTrialNum?.(false);
                    setIsTrialLoading(false);
                    checkIfHasTrial?.();
                  } else if (!data?.Schedule && data?.Schedule !== 0) {
                    getAllMachines();
                    setOpenTrialModalVisible(false);
                    setHasTrialNum?.(false);
                    setIsTrialLoading(false);
                    window.clearInterval((window as any).CsipAiAgentBindTimer);
                    checkIfHasTrial?.();
                  } else if (Date.now() - start > 60 * 60 * 1000) {
                    getAllMachines();
                    setOpenTrialModalVisible(false);
                    setHasTrialNum?.(false);
                    setIsTrialLoading(false);
                    window.clearInterval((window as any).CsipAiAgentBindTimer);
                    checkIfHasTrial?.();
                  }
                } catch (err) {
                  getAllMachines();
                  setOpenTrialModalVisible(false);
                  setHasTrialNum?.(false);
                  setIsTrialLoading(false);
                  window.clearInterval((window as any).CsipAiAgentBindTimer);
                  checkIfHasTrial?.();
                }
              },
              1000
            );
          } else {
            setOpenTrialModalVisible(false);
            setHasTrialNum?.(false);
            setIsTrialLoading(false);
            checkIfHasTrial?.();
          }
        } else {
          setOpenTrialModalVisible(false);
          setHasTrialNum?.(false);
          setIsTrialLoading(false);
          checkIfHasTrial?.();
        }
      } else {
        checkIfHasTrial?.();
      }
    } else {
      await Promise.all([
        ModifyLogStorageConfig({
          Type: Array.from(
            new Set([
              ...(logConfigData?.Type || []),
              "system_audit_log",
              "dns_log",
              "process_snapshot",
              "file_log",
            ])
          ),
          Period:
            typeof logConfigData?.Period === "number" &&
              logConfigData?.Period > 0
              ? logConfigData?.Period
              : 3640,
          IsModifyPeriod: false,
          Granularity:
            typeof logConfigData?.Granularity === "string" &&
              logConfigData?.Granularity
              ? logConfigData?.Granularity
              : "day",
          MsgLanguage:
            typeof logConfigData?.MsgLanguage === "string" &&
              logConfigData?.MsgLanguage
              ? logConfigData?.MsgLanguage
              : "zh",
        }),
        ScanAsset({ Quuids: [], AssetTypeIds: [17] }),
        // CreateScanTask()
      ]);
      checkIfHasTrial?.();
    }
    setOpenTrialModalVisible(false);
    setHasTrialNum?.(false);
    setIsTrialLoading(false);
  };

  useEffect(() => {
    if (openTrialModalVisible) {
      checkIfHasBuyLog();
    }
  }, [openTrialModalVisible]);

  return (
    <>
      <Dialog
        open={openTrialModalVisible}
        onOpenChange={setOpenTrialModalVisible}
      >
        <DialogContent className="csip-AIAgent-trial-modal sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>申请试用（AI Agent 安全）</DialogTitle>
          </DialogHeader>
          <div>
            <div style={{ fontSize: 13 }}>
              为保障功能正常使用，申请试用后将同步为您同步开启相关依赖能力，并开启必要的日志采集。
            </div>
            <hr style={{ margin: "20px 0 -5px 0" }} />
            <div
              className="maliciousRequest-editPolicy"
              style={{ margin: "25px 0 0", fontSize: 13 }}
            >
              <div className="mg-bt-16">
                <div className="label-txt">AI Agent机器</div>
                <div className="content">
                  {`${aiAgentHostList?.length || 0} 台${!aiAgentHostList?.filter?.(
                    (d: any) => d?.ProtectType === "BASIC_VERSION" )?.length
                    ? ''
                    : `（其中${aiAgentHostList?.filter?.(
                      (d: any) => d?.ProtectType === "BASIC_VERSION")?.length
                    }台资产未开通旗舰版，将赠送旗舰版，有效期14天）`
                    }`}
                </div>
              </div>
              <div>
                <div className="label-txt">开启日志采集</div>
                <div className="content">
                  {hasBuyLog ? (
                    <>
                      以下为此功能所需采集的日志类型，若部分日志服务未开通，将为您自动开启存储
                      <div
                        style={{
                          padding: "15px 18px",
                          marginTop: 12,
                          background: "#F7F8FB",
                        }}
                      >
                        <div style={{ marginBottom: 12, color: "rgba(0,0,0,0.5)" }}>
                          当前日志开启状态：
                        </div>
                        <div className="flex">
                          <div className="flex-1" style={{ paddingBottom: 0 }}>
                            <div className="flex">
                              <div className="w-[40%]">
                                <span style={{ color: "rgba(0,0,0,0.7)" }}>
                                  系统审计信息：
                                </span>
                              </div>
                              <div className="w-[60%]">
                                <LogStatusTag
                                  enabled={logConfigData?.Type?.includes?.(
                                    "system_audit_log"
                                  )}
                                />
                              </div>
                            </div>
                          </div>
                          <div className="flex-1" style={{ paddingBottom: 0 }}>
                            <div className="flex">
                              <div className="w-[40%]">
                                <span style={{ color: "rgba(0,0,0,0.7)" }}>
                                  DNS 日志：
                                </span>
                              </div>
                              <div className="w-[60%]">
                                <LogStatusTag
                                  enabled={logConfigData?.Type?.includes?.(
                                    "dns_log"
                                  )}
                                />
                              </div>
                            </div>
                          </div>
                        </div>
                        <div className="flex" style={{ marginTop: 8 }}>
                          <div className="flex-1">
                            <div className="flex">
                              <div className="w-[40%]">
                                <span style={{ color: "rgba(0,0,0,0.7)" }}>
                                  进程日志：
                                </span>
                              </div>
                              <div className="w-[60%]">
                                <LogStatusTag
                                  enabled={logConfigData?.Type?.includes?.(
                                    "process_snapshot"
                                  )}
                                />
                              </div>
                            </div>
                          </div>
                          <div className="flex-1">
                            <div className="flex">
                              <div className="w-[40%]">
                                <span style={{ color: "rgba(0,0,0,0.7)" }}>
                                  文件监控日志：
                                </span>
                              </div>
                              <div className="w-[60%]">
                                <LogStatusTag
                                  enabled={logConfigData?.Type?.includes?.(
                                    "file_log"
                                  )}
                                />
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </>
                  ) : (
                    <>
                      将自动赠送您 100GB 的日志分析量并为您开启下述日志采集，有效期14天
                      <div>（系统审计信息、DNS 日志、进程日志、文件监控日志）</div>
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              disabled={isTrialLoading || isGetAllMachinesLoading}
              onClick={createTrial}
            >
              {(isTrialLoading || isGetAllMachinesLoading) && <Loader2 className="animate-spin mr-1 h-4 w-4" />}
              开始体验
            </Button>
            <Button variant="outline" onClick={() => setOpenTrialModalVisible(false)}>
              取消
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {bindErrVisible && (
        <BindErrModal
          total={initTotal}
          initErrData={initErrData}
          bindTaskId={bindTaskId}
          visible={bindErrVisible}
          setVisible={setBindErrVisible}
          aiAgentHostList={aiAgentHostList}
        />
      )}
    </>
  );
}
