import React, { useEffect, useState } from "react";
import { toast } from 'sonner';
// @tencent/i18n stub (private package not available in public registry)
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { HelperText } from "@/components/ui/Typography";
import { ChevronDown, ChevronUp, Info, PackageCheck, Settings, ShieldCheck } from "lucide-react";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import AdminLayout from "@/components/AdminLayout";
import {
  DescribeLicenseWhiteConfig,
  DescribeAIAgentAssetList,
  DescribeABTestConfig,
  GetOpenClawInstances,
  ScanAsset,
  DescribeOrderList,
  DescribeAIAgentAutoOpenConfig,
  ModifyAIAgentAutoOpenConfig,
  ModifyLicenseBinds,
  DescribeLicenseList,
  DescribeLicenseBindList,
  DescribeLicenseBindSchedule,
} from "@/pages/admin/Security/api";
import LockPage from "./LockPage";
import { CSIP_AI_AGENT_NET_RULE } from "./AIAgent/constants";
import { getMaxRemoteStorage } from "./AIAgent/Common/tablePanelColumnUtil";
import {
  executeInstallTasksWithDelay,
  parseJsonStr,
} from "./AIAgent/Common/CommonRiskHandleFunc";
import BindErrModal from "./AIAgent/Common/BindErrModal";
import AIAgent from "./AIAgent";
import TrialModal from "./TrialModal";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import "./aiAgent.css";

// @tencent/tcb-jssdk stub (private package not available in public registry)
const directToCheck = async (_params: unknown) => { console.warn('directToCheck: tcb-jssdk not available'); };
const setRenewFlagV2 = async (_params: unknown): Promise<void> => { console.warn('setRenewFlagV2: tcb-jssdk not available'); };

export default function ExposurePageContainer() {
  const [isLoading, setIsLoading] = useState(true);
  const [isTrialLoading, setIsTrialLoading] = useState(false);
  const [showTipsPanel, setShowTipsPanel] = useState(false);
  const [hasTrialNum, setHasTrialNum] = useState(false);
  const [hasTrialOrder, setHasTrialOrder] = useState(false);
  const [trialDays, setTrialDays] = useState(0);
  const [aiAgentHostList, setAiAgentHostList] = useState([]);
  const [storageGroupData, setStorageGroupData] = useState(null);
  const [isHideLogTalkTab, setIsHideLogTalkTab] = useState(false);
  const [rencentScanTime, setRencentScanTime] = useState("");
  const [isGetAllMachinesLoading, setIsGetAllMachinesLoading] = useState(false);
  const [openTrialModalVisible, setOpenTrialModalVisible] = useState(false);
  const [openProtectModalVisible, setOpenProtectModalVisible] = useState(false);
  const [openVersionConfigVisible, setOpenVersionConfigVisible] = useState(false);
  const [protectOpenType, setProtectOpenType] = useState<"new" | "quota">("new");
  const [protectPeriod, setProtectPeriod] = useState(1);
  const [selectedQuotaOrderId, setSelectedQuotaOrderId] = useState('');
  const [showTrialBtn, setShowTrialBtn] = useState(false);
  const [selectedType, setSelectedType] = useState('');
  const [selectedAgentIds, setSelectedAgentIds] = useState<string[]>([]);
  const [aiOrderList, setAiOrderList] = useState([] as any);
  const [aiOrderConfig, setAiOrderConfig] = useState({} as any);
  const [tempAiOrderConfig, setTempAiOrderConfig] = useState({} as any);
  const [orderAutoRenewMap, setOrderAutoRenewMap] = useState({} as any);
  const [tempOrderAutoRenewMap, setTempOrderAutoRenewMap] = useState({} as any);
  const [quotaDetailOrder, setQuotaDetailOrder] = useState(null as any);
  const [bindMachinesData, setBindMachinesData] = useState([] as any);
  const [bindMachinesTotal, setBindMachinesTotal] = useState(0);
  const [bindMachinesPage, setBindMachinesPage] = useState(1);
  const [bindMachinesLoading, setBindMachinesLoading] = useState(false);
  const [bindErrVisible, setBindErrVisible] = useState(false);
  const [bindTaskId, setBindTaskId] = useState(0);
  const [initTotal, setInitTotal] = useState(0);
  const [initErrData, setInitErrData] = useState([]);

  const bindMachinesPageSize = 5;

  const protectPeriodOptions = [
    { label: "1个月", value: 1 },
    { label: "3个月", value: 3 },
    { label: "6个月", value: 6 },
    { label: "12个月", value: 12 },
  ];

  const versionConfigItems = [
    { label: "自动续费", enabled: false, apiKey: 'AutoRenew', tips: '开启自动续费，账户余额足够时，服务到期后按月自动续费' },
    { label: "自动加购", enabled: false, apiKey: 'AutoRepurchaseSwitch', tips: '当您有新增基础版AI Agent主机，且当前已无可用授权时，将自动为您扩容并绑定新增主机。自动加购生效的前提是自动绑定和自动加购开关均已开启，否则实际不会进行自动加购。' },
    { label: "自动绑定", enabled: false, apiKey: 'AutoBindSwitch', tips: '当您有新增基础版AI Agent主机时，将为您自动绑定剩余可用授权。若同时存在多个剩余可用授权，将优先绑定有效时间长的授权。' },
    { label: "自动缩容", enabled: false, apiKey: 'AutoDowngradeSwitch', tips: <>{'开启后，订单将在主机销毁后24小时内完成自动缩容退费（每日10:30开始执行,不含当天的缩容任务）。所有交易记录前往'}
      <a style={{ margin: '0 4px', color:'#0052d9' }} href="https://console.cloud.tencent.com/expense/deal" target="_blank">
        {'订单中心'}
      </a>
    {'查看。'}</> },
  ];

  const getOrderId = (order: any) => order?.ResourceId;

  const getOrderTotal = (order: any) => order?.InquireNum;

  const getOrderUsed = (order: any) => order?.UsedNum;

  const getOrderRemain = (order: any) => Math.max((getOrderTotal(order) || 0) - (getOrderUsed(order) || 0), 0);

  const getOrderStartTime = (order: any) => order?.BeginTime;

  const getOrderEndTime = (order: any) => order?.EndTime;

  const getOrderAutoRenew = (order: any) => order?.AutoRenewFlag == 1;

  const protectAgentCount = selectedType === 'batch' ? Math.max(selectedAgentIds?.length || 0, 1) : 1;

  const getAiAgentConfig = async () => {
    try {
      const res: any = await Promise.all([
        DescribeAIAgentAutoOpenConfig(),
        DescribeOrderList({
          Offset: 0,
          Limit: 100,
          Filters: [
            { Name: "SourceType", Values: ["0"] },
            { Name: "InquireKey", Values: ["sv_yunjing_ai_agent_security"] },
          ],
        }),
      ]);
      console.log(3001, res);
      setAiOrderConfig(res?.[0] || {});
      setTempAiOrderConfig(res?.[0] || {});
      const list = res?.[1]?.List || [];
      setAiOrderList(list?.filter?.((d: any) => Date.parse(d?.EndTime?.replace?.(/-/g, '/') ?? 0) > Date.now()));
      const renewMap = list?.reduce?.((acc: any, order: any) => {
        acc[getOrderId(order)] = getOrderAutoRenew(order);
        return acc;
      }, {}) || {};
      setOrderAutoRenewMap(renewMap);
      setTempOrderAutoRenewMap(renewMap);
    } catch (err) {
      console.warn('[Security] getAiAgentConfig failed:', err);
    }
  };

  const getBindMachines = async (order: any, page = 1) => {
    if (!order?.ResourceId) {
      setBindMachinesData([]);
      setBindMachinesTotal(0);
      return;
    }
    setBindMachinesLoading(true);
    try {
      const resp: any = await DescribeLicenseList({
        Offset: 0,
        Limit: 1,
        Filters: [{ Name: 'ResourceId', Values: [order?.ResourceId] }],
      });
      const licenseId = resp?.List?.[0]?.LicenseId;
      if (!licenseId) {
        setBindMachinesData([]);
        setBindMachinesTotal(0);
        return;
      }
      const res: any = await DescribeLicenseBindList({
        Offset: (page - 1) * bindMachinesPageSize,
        Limit: bindMachinesPageSize,
        LicenseId: licenseId,
        LicenseType: 8,
        ResourceId: order?.ResourceId,
      });
      setBindMachinesData(res?.List || []);
      setBindMachinesTotal(res?.TotalCount || 0);
      setBindMachinesPage(page);
    } finally {
      setBindMachinesLoading(false);
    }
  };

  const bindOrder = async (orderId: any, agentIds: any) => {
    const quuids = aiAgentHostList?.filter?.((d: any) => agentIds?.includes?.(d?.InstanceID))?.map?.((d: any) => d?.Quuid);
    console.log(33022, aiAgentHostList, quuids);
    if (!quuids?.length) {
      return;
    }
    const bindRes: any = await ModifyLicenseBinds({
      ResourceId: orderId,
      LicenseType: 8,
      IsAll: false,
      QuuidList: quuids,
    });
    if (bindRes?.TaskId) {
      setBindTaskId(bindRes?.TaskId);
      const start = Date.now();
      (window as any).CsipAiAgentBindTimer = window.setInterval(async () => {
        try {
          const data: any = await DescribeLicenseBindSchedule({ TaskId: bindRes?.TaskId });
          if (data?.Schedule >= 100) {
            window.clearInterval((window as any).CsipAiAgentBindTimer);
            const err: any = await DescribeLicenseBindSchedule({
              TaskId: bindRes?.TaskId,
              Offset: 0,
              Limit: 10,
              Filters: [{ Name: 'Status', Values: ['2'] }],
            });
            if (err?.TotalCount > 0) {
              setInitTotal(err?.TotalCount ?? 0);
              setInitErrData(err?.List ?? []);
              setBindErrVisible(true);
            }
            await ScanAsset({ Quuids: [], AssetTypeIds: [17] });
            getAllMachines?.();
            getAiAgentConfig();
          } else if (!data?.Schedule && data?.Schedule !== 0) {
            getAllMachines?.();
            getAiAgentConfig();
            window.clearInterval((window as any).CsipAiAgentBindTimer);
          } else if (Date.now() - start > 60 * 60 * 1000) {
            getAllMachines?.();
            getAiAgentConfig();
            window.clearInterval((window as any).CsipAiAgentBindTimer);
          }
        } catch (err) {
          getAllMachines?.();
          getAiAgentConfig();
          window.clearInterval((window as any).CsipAiAgentBindTimer);
        }
      }, 2000);
    } else {
      getAllMachines?.();
      getAiAgentConfig();
      window.clearInterval((window as any).CsipAiAgentBindTimer);
    }
  };

  const handleChangeSwitch = async () => {
    setOpenVersionConfigVisible(false);
    const res: any = await ModifyAIAgentAutoOpenConfig({
      AutoBindSwitch: tempAiOrderConfig?.AutoBindSwitch,
      AutoRepurchaseSwitch: tempAiOrderConfig?.AutoRepurchaseSwitch,
      AutoDowngradeSwitch: tempAiOrderConfig?.AutoDowngradeSwitch,
      RepurchaseRenewSwitch: aiOrderConfig?.RepurchaseRenewSwitch || 0,
    });
    if (res && !res?.Error) {
      toast.success('操作成功');
    }
    const openOrders = aiOrderList?.filter?.((d: any) => d?.AutoRenewFlag != 1 && tempOrderAutoRenewMap?.[d?.ResourceId]);
    const closeOrders = aiOrderList?.filter?.((d: any) => d?.AutoRenewFlag == 1 && !tempOrderAutoRenewMap?.[d?.ResourceId]);
    if (openOrders?.length || closeOrders?.length) {
      try {
        const addParams: any = {
          regionId: 1,
          productCode: "p_yunjing",
          productType: "yunjing_pro",
          renewalFlag: '1',
          renewalList: openOrders?.map?.((d: any) => ({
            resourceId: d?.ResourceId,
            subProductCode: 'sp_yunjing_ai_security',
          })),
        };
        const closeParams: any = {
          regionId: 1,
          productCode: "p_yunjing",
          productType: "yunjing_pro",
          renewalFlag: '2',
          renewalList: closeOrders?.map?.((d: any) => ({
            resourceId: d?.ResourceId,
            subProductCode: 'sp_yunjing_ai_security',
          })),
        };
        await Promise.all(
          (openOrders?.length ? [setRenewFlagV2(addParams)] : [])
            .concat(closeOrders?.length ? [setRenewFlagV2(closeParams)] : []));
      } catch (err: any) {
        if (err?.code === 300042 || err?.msg?.indexOf?.('not authorized') > -1) {
          toast.error('您当前暂无续费操作权限，请添加对应权限后继续操作');
        } else {
          toast.error('操作失败');
        }
      }
    }
    getAiAgentConfig();
  };

  const getAllMachines = async () => {
    setIsGetAllMachinesLoading(true);
    try {
    const MAX_LIMIT = 100;
    const MAX_REQUEST = 18;
    const openclaws: any = await GetOpenClawInstances({
      page: 1,
      page_size: MAX_LIMIT,
    });
    console.log(11777, openclaws);
    let insData = openclaws?.instances || [];
    if (openclaws?.total > MAX_LIMIT) {
      const executeTimes = Math.ceil(openclaws?.total / MAX_LIMIT) - 1;
      const res: any = await Promise.all(
        new Array(executeTimes)
          .fill(1)
          .map((d: any, i: number) =>
            GetOpenClawInstances({ page: i + 2, page_size: MAX_LIMIT })
          )
      );
      insData = insData?.concat?.(
        res?.map?.((d: any) => d?.instances || [])?.flat?.(2)
      );
    }
    const insIds = insData?.map?.((d: any) => d?.InstanceId);
    if (insIds?.length) {
      const res: any = await DescribeAIAgentAssetList({
        Filter: {
          Offset: 0, Limit: MAX_LIMIT,
          // Filters: [{ Name: 'InstanceIDs', Values: insIds }]
        },
      });
      let list = res?.AssetList || [];
      const total = res?.TotalCount || list?.length || 0;
      if (total > 0) {
        window.clearInterval((window as any).CsipAIAgentLockSyncTimer);
      }
      if (total > MAX_LIMIT) {
        const params = {
          regionId: 1,
          serviceType: "csip",
          cmd: "DescribeAIAgentAssetList",
          data: {
            Filter: {
              Offset: 0,
              Limit: MAX_LIMIT,
            },
          },
        };
        const executeTimes = Math.ceil(total / MAX_LIMIT) - 1;
        const allNums: any[] = new Array(executeTimes)
          .fill(1)
          .reduce((pre, cur, i) => {
            const index = Math.ceil((i + 1) / MAX_REQUEST) - 1;
            if (pre[index]) {
              if (pre[index]?.length < MAX_REQUEST) {
                pre[index] = pre[index].concat(cur);
              } else {
                pre[index + 1] = [cur];
              }
            } else {
              pre[index] = [cur];
            }
            return pre;
          }, []);
        const paramsArr = allNums.map((p, x) =>
          p?.map?.((d: any, i: number) => ({
            ...params,
            data: {
              ...params.data,
              Filter: {
                Offset: x * MAX_LIMIT * MAX_REQUEST + (i + 1) * MAX_LIMIT,
                Limit: MAX_LIMIT,
              },
            },
          }))
        );
        await executeInstallTasksWithDelay(paramsArr, 500)
          .then(res => {
            list = list?.concat?.(
              res?.map?.(item => item?.AssetList || [])?.flat?.(2)
            );
          })
          .catch(err => console.log(err));
      }
      list = list?.filter?.((d: any) => insIds?.includes?.(d?.InstanceID));
      // 获取每台主机的详细信息
      if (list?.length) {
        let machines: any = [];
        const instanceIds = list.map((d: any) => d?.InstanceID);
        const insArr = instanceIds.reduce((pre: any, cur: any, i: number) => {
          const index = Math.ceil((i + 1) / MAX_LIMIT) - 1;
          if (pre[index]) {
            if (pre[index]?.length < MAX_LIMIT) {
              pre[index] = pre[index].concat(cur);
            } else {
              pre[index + 1] = [cur];
            }
          } else {
            pre[index] = [cur];
          }
          return pre;
        }, []);
        const allNums = insArr.reduce((pre: any, cur: any, i: number) => {
          const index = Math.ceil((i + 1) / MAX_REQUEST) - 1;
          if (pre[index]) {
            if (pre[index]?.length < MAX_REQUEST) {
              pre[index] = pre[index].concat([cur]);
            } else {
              pre[index + 1] = [cur];
            }
          } else {
            pre[index] = [cur];
          }
          return pre;
        }, []);
        const getParams = (instanceIds: any) => ({
          regionId: 1,
          serviceType: "cwp",
          cmd: "DescribeMachines",
          data: {
            Offset: 0,
            Limit: MAX_LIMIT,
            MachineRegion: "all-regions",
            MachineType: "CVM",
            Filters: [{ Name: "InstanceIds", Values: instanceIds }],
          },
        });
        const paramsArr = allNums.map((p: any[]) =>
          p?.map?.((d: any) => getParams(d))
        );
        await executeInstallTasksWithDelay(paramsArr, 500)
          .then(res => {
            machines = machines?.concat?.(
              res?.map?.(item => item?.Machines || [])?.flat?.(2)
            );
          })
          .catch(err => console.log(err));
        list = list?.map?.((d: any) => ({
          ...d,
          ...(machines?.find?.(
            (a: any) => a?.MachineExtraInfo?.InstanceID === d?.InstanceID
          ) || {}),
        }));
      }
      const latestTime = list?.length
        ? list?.sort?.(
            (a: any, b: any) =>
              Date.parse(b.IdentityTimeLast) - Date.parse(a.IdentityTimeLast)
          )?.[0]?.IdentityTimeLast || ""
        : "";
      console.log(56660, list, latestTime);
      setRencentScanTime(latestTime);
      setAiAgentHostList(
        list?.map?.((d: any) => ({
          ...d,
          OpenClawName:
            insData?.find?.((x: any) => x.InstanceId === d?.InstanceID)?.Name ||
            "",
        }))
      );
    }
    } catch (err) {
      console.warn('[Security] getAllMachines failed:', err);
    } finally {
      setIsGetAllMachinesLoading(false);
    }
  };

  const checkIfHasTrial = async () => {
    setIsLoading(true);
    try {
      const res: any = await Promise.all([
        DescribeABTestConfig(),
        DescribeLicenseWhiteConfig({ RuleName: "csip_yunying" }),
        ScanAsset({ Quuids: [], AssetTypeIds: [17] }),
        DescribeOrderList({
          Offset: 0,
          Limit: 1,
          Filters: [
            { Name: "SourceType", Values: ["15"] },
            { Name: "InquireKey", Values: ["sv_yunjing_ue_aams"] },
          ],
        }),
      ]);
      setIsHideLogTalkTab(
        res?.[0]?.Config?.find?.(
          (item: any) => item?.ProjectName === "csip_ai_security_hide_log"
        )?.Status
      );
      setShowTrialBtn(res?.[0]?.Config?.find?.((item: any) => item?.ProjectName === 'csip_ai_agent_trial')?.Status);
      setHasTrialNum(
        res?.[1]?.FlagShip?.Deadline > 0 &&
          res?.[1]?.FlagShip?.LicenseNum > 0 &&
          res?.[1]?.FlagShip?.IsApplyFor
      );
      setTrialDays(res?.[1]?.FlagShip?.Deadline || 0);
      setHasTrialOrder(res?.[3]?.TotalCount > 0);
    } catch (err) {
      console.warn('[Security] checkIfHasTrial failed:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const createOrder = async (num: any, time: any, autoRenewFlag: any) => {
    try {
      const params = JSON.stringify({
        goods: [{
          regionId: 1,
          zoneId: 100001,
          goodsNum: 1,
          projectId: 0,
          platform: 1,
          currency: 'CNY',
          goodsCategoryId: 2025040,
          payMode: 1,
          goodsDetail: {
            pid: 1058442,
            productCode: 'p_yunjing',
            subProductCode: 'sp_yunjing_ai_security',
            productInfo: [{ name: 'AI Agent安全', value: num }],
            autoRenewFlag,
            timeSpan: Number(time),
            timeUnit: 'm',
            sv_yunjing_ai_agent_security: String(num),
          }
        }],
      });
      console.log(666, params);
      await directToCheck({ itemDetails: params, target: '_blank' });
    } catch (error) {
      console.log(1133, error);
    }
  };

  useEffect(() => {
    if (quotaDetailOrder) {
      getBindMachines(quotaDetailOrder, 1);
    } else {
      setBindMachinesData([]);
      setBindMachinesTotal(0);
      setBindMachinesPage(1);
    }
  }, [quotaDetailOrder]);

  useEffect(() => {
    const availableOrder = aiOrderList?.find?.((order: any) => getOrderRemain(order) >= protectAgentCount);
    const currentOrder = aiOrderList?.find?.((order: any) => getOrderId(order) === selectedQuotaOrderId);
    if (availableOrder && (!selectedQuotaOrderId || getOrderRemain(currentOrder) < protectAgentCount)) {
      setSelectedQuotaOrderId(getOrderId(availableOrder));
    }
    if (!availableOrder) {
      setSelectedQuotaOrderId('');
    }
  }, [aiOrderList, selectedQuotaOrderId, protectAgentCount]);

  useEffect(() => {
    checkIfHasTrial();
    getAllMachines();
    getAiAgentConfig();
    getMaxRemoteStorage(CSIP_AI_AGENT_NET_RULE, (val: any) =>
      setStorageGroupData(!val ? null : parseJsonStr(val || null))
    );
  }, []);

  useEffect(() => {
    return () => {
      window.clearTimeout((window as any).CsipAIAgentSyncTimer);
      window.clearInterval((window as any).CsipAIAgentLockSyncTimer);
    };
  }, []);

  return (
    <>
      <AdminLayout>
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Spinner className="mr-2" />
            <span className="text-sm text-gray-500">数据加载中...</span>
          </div>
        ) : showTrialBtn && hasTrialNum && !hasTrialOrder ? (
          <LockPage
            aiAgentHostList={aiAgentHostList}
            getAllMachines={getAllMachines}
            isTrialLoading={isTrialLoading}
            rencentScanTime={rencentScanTime}
            isGetAllMachinesLoading={isGetAllMachinesLoading}
            setOpenTrialModalVisible={setOpenTrialModalVisible}
          />
        ) : (
          <div className="assetcenter-container csip-ai-agent-wrap">
            <AdminPageHeader
              title="AI Agent安全"
              description={
                <span>
                  帮助您持续监测AI Agent资产的风险告警、管控策略生效情况与审计记录，让你在"可见—可控—可追溯"的闭环下，安全引入并持续使用AI Agent。（已支持 OpenClaw，其他 Agent 类型敬请期待）
                  <button
                    className="inline-flex items-center gap-0.5 ml-1.5 text-[var(--text-brand)] hover:opacity-80 cursor-pointer align-baseline"
                    onClick={() => setShowTipsPanel(!showTipsPanel)}
                  >
                    功能使用说明
                    {showTipsPanel ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                  </button>
                </span>
              }
              actions={
                <>
                  {aiOrderList?.length ? (
                    <Button
                      variant="outline"
                      size="claw-sm"
                      onClick={() => {
                        setTempAiOrderConfig(aiOrderConfig);
                        setTempOrderAutoRenewMap(orderAutoRenewMap);
                        setOpenVersionConfigVisible(true);
                      }}
                    >
                      <Settings />
                      防护版本配置
                    </Button>
                  ) : null}
                </>
              }
            />
            {showTrialBtn ? hasTrialNum && aiAgentHostList?.filter?.((d: any) => d?.ProtectType === "BASIC_VERSION")?.length > 0 ? (
              <Alert variant="info" className="mb-6">
                <AlertInfoIcon />
                <AlertDescription>
                  监测到您当前有
                  {aiAgentHostList?.filter?.((d: any) => d?.ProtectType === "BASIC_VERSION")?.length}
                  个AI Agent资产处于未防护状态，可点击
                  <span
                    style={{
                      margin: "0 0 0 5px",
                      fontWeight: 500,
                      cursor: "pointer",
                    }}
                    onClick={() => setOpenTrialModalVisible(true)}
                  >
                    领取试用
                  </span>
                </AlertDescription>
              </Alert>
            ) : null : aiAgentHostList?.filter?.((d: any) => d?.ProtectType === "BASIC_VERSION")?.length > 0 ? (
              <Alert variant="info" className="mb-6">
                <AlertInfoIcon />
                <AlertDescription>
                  监测到您当前有
                  {aiAgentHostList?.filter?.((d: any) => d?.ProtectType === "BASIC_VERSION")?.length}
                  个AI Agent资产处于未防护状态，请开启防护，解锁完整能力
                  <span
                    style={{
                      margin: "0 0 0 5px",
                      fontWeight: 500,
                      cursor: "pointer",
                    }}
                    onClick={() => {
                      setSelectedAgentIds(aiAgentHostList?.filter?.((d: any) => d?.ProtectType === "BASIC_VERSION")?.map?.((d: any) => d?.InstanceID));
                      setSelectedType?.('batch');
                      setOpenProtectModalVisible(true);
                    }}
                  >
                    开启防护
                  </span>
                </AlertDescription>
              </Alert>
            ) : null}
            <div>
              <AIAgent
                hasTrialNum={hasTrialNum}
                showTipsPanel={showTipsPanel}
                setShowTipsPanel={setShowTipsPanel}
                getAllMachines={getAllMachines}
                aiAgentHostList={aiAgentHostList}
                setAiAgentHostList={setAiAgentHostList}
                isGetAllMachinesLoading={isGetAllMachinesLoading}
                storageGroupData={storageGroupData}
                isHideLogTalkTab={isHideLogTalkTab}
                rencentScanTime={rencentScanTime}
                setOpenTrialModalVisible={setOpenTrialModalVisible}
                showTrialBtn={showTrialBtn}
                setSelectedType={setSelectedType}
                selectedAgentIds={selectedAgentIds}
                setSelectedAgentIds={setSelectedAgentIds}
                setOpenProtectModalVisible={setOpenProtectModalVisible}
              />
            </div>
          </div>
        )}
      </AdminLayout>

      <TrialModal
        openTrialModalVisible={openTrialModalVisible}
        setOpenTrialModalVisible={setOpenTrialModalVisible}
        setHasTrialNum={setHasTrialNum}
        aiAgentHostList={aiAgentHostList}
        getAllMachines={getAllMachines}
        isTrialLoading={isTrialLoading}
        setIsTrialLoading={setIsTrialLoading}
        isGetAllMachinesLoading={isGetAllMachinesLoading}
        trialDays={trialDays}
        checkIfHasTrial={checkIfHasTrial}
      />

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

      <Dialog open={openVersionConfigVisible} onOpenChange={setOpenVersionConfigVisible}>
        <DialogContent className="sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>防护版本配置</DialogTitle>
          </DialogHeader>

          <div className="max-h-[70vh] overflow-y-auto">
            <p className="mb-4 text-sm leading-5 text-[#737373]">
              配置防护版本的自动化策略，开启后系统将自动执行相关操作。
            </p>

            <div className="mb-6 grid grid-cols-2 gap-3">
              {versionConfigItems.map(item => (
                <div key={item.label} className="flex items-center justify-between rounded-[4px] border border-[#E5E5E5] bg-white px-4 py-3">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span
                        className="text-sm font-medium text-[#0A0A0A]"
                        style={{
                          textDecoration: 'underline dashed rgb(0 0 0 / 30%)',
                          textUnderlineOffset: 5,
                          cursor: 'pointer'
                        }}
                      >{item.label}</span>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-[320px]">
                      {item?.tips}
                    </TooltipContent>
                  </Tooltip>
                  <div className="flex items-center gap-3 text-xs">
                    {item?.apiKey === 'AutoRenew' ? Object.keys(tempOrderAutoRenewMap)?.every?.(d => tempOrderAutoRenewMap?.[d]) ?
                      <span className="text-[#1447E6]">• 已开启</span> :
                        <span className="text-[#A3A3A3]">• 未开启</span> :
                          tempAiOrderConfig?.[item?.apiKey] == 1 ? <span className="text-[#1447E6]">• 已开启</span> :
                            <span className="text-[#A3A3A3]">• 未开启</span>}
                    {item?.apiKey === 'AutoRenew' ?
                    Object.keys(tempOrderAutoRenewMap)?.every?.(d => tempOrderAutoRenewMap?.[d]) ?
                    <button
                      type="button"
                      className="text-[#1447E6] hover:underline"
                      onClick={() => {
                        setTempOrderAutoRenewMap(Object.keys(tempOrderAutoRenewMap)?.reduce?.((pre: any, cur: any) => {
                          pre[cur] = false;
                          return pre;
                        }, {}) || {});
                      }}
                    >
                      关闭
                    </button> :
                      <button
                        type="button"
                        className="text-[#1447E6] hover:underline"
                        onClick={() => {
                          setTempOrderAutoRenewMap(Object.keys(tempOrderAutoRenewMap)?.reduce?.((pre: any, cur: any) => {
                            pre[cur] = true;
                            return pre;
                          }, {}) || {});
                        }}
                      >
                        开启
                      </button>
                    : tempAiOrderConfig?.[item?.apiKey] == 1 ?
                    <button
                      type="button"
                      className="text-[#1447E6] hover:underline"
                      onClick={() => {
                        setTempAiOrderConfig({ ...tempAiOrderConfig, [item?.apiKey]: 0 });
                      }}
                    >
                      关闭
                    </button> :
                    <button
                      type="button"
                      className="text-[#1447E6] hover:underline"
                      onClick={() => {
                        setTempAiOrderConfig({ ...tempAiOrderConfig, [item?.apiKey]: 1 });
                      }}
                    >
                      开启
                    </button>}
                  </div>
                </div>
              ))}
            </div>

            <div className="mb-3 text-sm font-medium text-[#0A0A0A]">
              配额信息
            </div>

            <div className="space-y-3">
              {aiOrderList?.length ? (
                aiOrderList.map((order: any, index: number) => {
                  const orderId = getOrderId(order);
                  const total = getOrderTotal(order);
                  const used = getOrderUsed(order);
                  const usagePercent = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
                  const isNearFull = total > 0 && usagePercent >= 90;
                  return (
                    <div key={orderId} className="rounded-[4px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3">
                      <div className="mb-3 flex items-center justify-between gap-3">
                        <div className="text-sm font-medium text-[#0A0A0A]">
                          AI防护版配额 #{index + 1}
                        </div>
                        <div className="flex items-center gap-3">
                          <Switch
                            checked={tempOrderAutoRenewMap[orderId]}
                            onCheckedChange={(value: boolean) => {
                              setTempOrderAutoRenewMap((prev: any) => ({ ...prev, [orderId]: Boolean(value) }));
                            }}
                          />
                          <span className="text-xs text-[#737373]">自动续费</span>
                          <button
                            type="button"
                            className="text-xs text-[#1447E6] hover:underline"
                            onClick={() => {
                              getBindMachines(order);
                              setQuotaDetailOrder(order);
                            }}
                          >
                            查看详情
                          </button>
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-6 text-xs">
                        <div>
                          <div className="mb-1 text-[#737373]">配额数量</div>
                          <div className="mb-1 font-medium text-[#0A0A0A]">
                            <span className="text-[#1447E6]">{used}</span>
                            {' / '}
                            {total || '-'} 已使用
                          </div>
                          <div className="h-1.5 overflow-hidden rounded-full bg-[#F5F5F5]">
                            <div
                              className={`h-full rounded-full ${isNearFull ? 'bg-[#D97706]' : 'bg-[#1447E6]'}`}
                              style={{ width: `${usagePercent}%` }}
                            />
                          </div>
                        </div>
                        <div>
                          <div className="mb-1 text-[#737373]">防护有效期</div>
                          <div className="font-medium text-[#0A0A0A]">
                            {getOrderStartTime(order)} ~ {getOrderEndTime(order)}
                          </div>
                        </div>
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="text-center py-10">
                  <HelperText>暂无防护配额</HelperText>
                </div>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setOpenVersionConfigVisible(false)}>
              取消
            </Button>
            <Button variant="dialog-confirm" size="claw-sm" onClick={handleChangeSwitch}>
              保存配置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!quotaDetailOrder} onOpenChange={(open) => !open && setQuotaDetailOrder(null)}>
        <DialogContent className="sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>配额详情</DialogTitle>
          </DialogHeader>

          {quotaDetailOrder && (
            <div>
              <div className="mb-6 grid grid-cols-2 gap-x-1 gap-y-4">
                <div>
                  <div className="mb-1 text-xs text-[#737373]">配额数量</div>
                  <div className="text-sm font-medium text-[#0A0A0A]">
                    {getOrderUsed(quotaDetailOrder)} / {getOrderTotal(quotaDetailOrder)} 已使用
                  </div>
                </div>
                <div>
                  <div className="mb-1 text-xs text-[#737373]">防护有效期</div>
                  <div className="text-sm font-medium text-[#0A0A0A]">
                    {getOrderStartTime(quotaDetailOrder)} ~ {getOrderEndTime(quotaDetailOrder)}
                  </div>
                </div>
                <div>
                  <div className="mb-1 text-xs text-[#737373]">自动续费</div>
                  <div className="text-sm font-medium">
                    {getOrderAutoRenew(quotaDetailOrder) ? (
                      <span className="badge-running">已开启</span>
                    ) : (
                      <span className="badge-shutdown">未开启</span>
                    )}
                  </div>
                </div>
                <div>
                  <div className="mb-1 text-xs text-[#737373]">配额类型</div>
                  <div className="text-sm font-medium text-[#0A0A0A]">AI防护版</div>
                </div>
              </div>

              <div className="mb-3 text-sm font-medium text-[#0A0A0A]">
                已消耗配额的 AI Agent（{bindMachinesTotal} 个）
              </div>

              <div className="overflow-hidden rounded-[4px] border border-[#E5E5E5] bg-white">
                {bindMachinesLoading ? (
                  <div className="flex items-center justify-center px-6 py-8 text-sm text-[#A3A3A3]">
                    <Spinner className="mr-2" />
                    加载中...
                  </div>
                ) : bindMachinesData.length ? (
                  bindMachinesData.map((agent: any) => (
                    <div
                      key={agent?.MachineExtraInfo?.InstanceID}
                      className="grid grid-cols-2 items-center border-b border-[#F5F5F5] px-4 py-3 last:border-b-0"
                    >
                      <span className="text-sm font-medium text-[#0A0A0A]">
                        {agent?.MachineName || '-'}
                      </span>
                      <span className="text-right text-xs text-[#737373]">
                        ID: {agent?.MachineExtraInfo?.InstanceID || '-'}
                      </span>
                    </div>
                  ))
                ) : (
                  <div className="text-center py-10">
                    <HelperText>暂无已绑定 Agent</HelperText>
                  </div>
                )}
              </div>

              {bindMachinesTotal > bindMachinesPageSize && (
                <div className="mt-4 flex items-center justify-end gap-3 text-sm text-[#737373]">
                  <span>
                    共 {bindMachinesTotal} 条，第 {bindMachinesPage} / {Math.ceil(bindMachinesTotal / bindMachinesPageSize)} 页
                  </span>
                  <Button
                    variant="claw-outline"
                    size="claw-sm"
                    disabled={bindMachinesPage <= 1 || bindMachinesLoading}
                    onClick={() => getBindMachines(quotaDetailOrder, bindMachinesPage - 1)}
                  >
                    上一页
                  </Button>
                  <Button
                    variant="claw-outline"
                    size="claw-sm"
                    disabled={bindMachinesPage >= Math.ceil(bindMachinesTotal / bindMachinesPageSize) || bindMachinesLoading}
                    onClick={() => getBindMachines(quotaDetailOrder, bindMachinesPage + 1)}
                  >
                    下一页
                  </Button>
                </div>
              )}
            </div>
          )}

          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setQuotaDetailOrder(null)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={openProtectModalVisible} onOpenChange={setOpenProtectModalVisible}>
        <DialogContent className="sm:max-w-[640px]">
          <DialogHeader>
            <DialogTitle>
              {selectedType === 'batch' ? `批量开启AI防护（${selectedAgentIds?.length}个Agent）` : '开启AI防护'}
            </DialogTitle>
          </DialogHeader>

          <div>
            {/* 类型切换：新购 / 使用现有配额 — 规范 LineTabs */}
            <div className="flex items-center gap-1 border-b border-[#f0f0f0] mb-4">
              {[
                { value: 'new', label: '新购' },
                { value: 'quota', label: '使用现有配额' },
              ].map(it => {
                const active = protectOpenType === it.value;
                return (
                  <button
                    key={it.value}
                    type="button"
                    onClick={() => setProtectOpenType(it.value as 'new' | 'quota')}
                    className={`relative px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap ${
                      active
                        ? 'text-[#0A0A0A] border-b-2 border-[#0A0A0A] -mb-px'
                        : 'text-[#737373] hover:text-[#0A0A0A]'
                    }`}
                  >
                    {it.label}
                  </button>
                );
              })}
            </div>

            {protectOpenType === "new" ? (
              <>
                <div className="mb-4">
                  <div className="mb-3 text-sm font-medium text-[#0A0A0A]">
                    <span className="mr-1 text-[#DC2626]">*</span>
                    计费周期
                  </div>
                  {/* 计费周期 — 规范 Segmented Control */}
                  <div className="inline-flex items-center gap-1 p-1 bg-[#F5F5F5] rounded-[4px]">
                    {protectPeriodOptions.map(item => {
                      const isActive = protectPeriod === item.value;
                      return (
                        <button
                          key={item.value}
                          type="button"
                          onClick={() => setProtectPeriod(item.value)}
                          className={`px-4 py-1 text-sm rounded-[3px] transition-colors min-w-[64px] ${
                            isActive
                              ? 'bg-white text-[#0A0A0A] font-medium'
                              : 'text-[#737373] hover:text-[#0A0A0A] font-normal'
                          }`}
                          style={isActive ? { boxShadow: 'var(--shadow-segment)' } : undefined}
                        >
                          {item.label}
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* 提示行 */}
                <div className="mb-4 flex items-start gap-1.5 text-sm leading-5 text-[#737373]">
                  <Info className="w-4 h-4 text-[#737373] mt-0.5 shrink-0" />
                  <span>开启防护后，将立即生效并开始计费。防护到期前可在「防护版本配置」中设置自动续费。</span>
                </div>

                {/* 预估费用卡片 — 规范浅灰中性卡片 */}
                <div className="rounded-[4px] border border-[#E5E5E5] bg-[#FAFAFA] px-5 py-4">
                  <div className="mb-2 text-sm font-medium text-[#737373]">预估费用</div>
                  <div className="mb-1 flex items-end gap-2">
                    <span className="text-2xl font-semibold leading-none text-[#1447E6]">
                      ¥{16 * Number(selectedType === 'batch' ? selectedAgentIds?.length : 1)}
                    </span>
                    <span className="pb-1 text-sm text-[#737373]">/月</span>
                  </div>
                  <div className="text-xs text-[#737373]">
                    {selectedType === 'batch'
                      ? `${selectedAgentIds?.length}个Agent x ¥16/月，含全量AI安全检测能力`
                      : '单Agent防护费用，含全量AI安全检测能力'}
                  </div>
                </div>
              </>
            ) : !aiOrderList?.length ? (
              <div className="text-center py-10 space-y-1">
                <HelperText>暂无可用配额</HelperText>
                <HelperText>当前暂无可用于 AI Agent 安全的防护配额，请选择新购后开启防护。</HelperText>
              </div>
            ) : (
              <div style={{ maxHeight: 500, overflowY: 'auto' }}>
                <div className="mb-3 text-sm text-[#525252]">
                  {selectedType === 'batch'
                    ? `本次已选择 ${protectAgentCount} 个Agent，请选择一个剩余配额充足的现有配额开启防护：`
                    : '选择一个现有配额来为此Agent开启防护：'}
                </div>
                <div className="space-y-3">
                  {aiOrderList.map((order: any, index: number) => {
                    const orderId = getOrderId(order);
                    const total = getOrderTotal(order) || 0;
                    const used = getOrderUsed(order) || 0;
                    const remain = getOrderRemain(order);
                    const disabled = remain < protectAgentCount;
                    const isUsedUp = remain <= 0;
                    const selected = selectedQuotaOrderId === orderId;

                    return (
                      <button
                        key={orderId}
                        type="button"
                        disabled={disabled}
                        className={`flex w-full items-center gap-3 rounded-[4px] border px-4 py-4 text-left transition-all ${
                          disabled
                            ? 'cursor-not-allowed border-[#E5E5E5] bg-[#FAFAFA] text-[#A3A3A3]'
                            : selected
                            ? 'border-[#1447E6] bg-white'
                            : 'border-[#E5E5E5] bg-white hover:border-[#C9D5FC]'
                        }`}
                        style={selected && !disabled ? { boxShadow: '0px 1px 3px 0px rgba(0,0,0,0.05)' } : undefined}
                        onClick={() => {
                          if (!disabled) {
                            setSelectedQuotaOrderId(orderId);
                          }
                        }}
                      >
                        <span
                          className={`flex h-4 w-4 shrink-0 items-center justify-center rounded-full border ${
                            selected && !disabled ? 'border-[#1447E6]' : 'border-[#A3A3A3]'
                          }`}
                        >
                          {selected && !disabled && <span className="h-2 w-2 rounded-full bg-[#1447E6]" />}
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="mb-1.5 flex items-center gap-2">
                            <span className={`text-sm font-medium ${disabled ? 'text-[#A3A3A3]' : 'text-[#0A0A0A]'}`}>
                              AI防护版配额 #{index + 1}
                            </span>
                            <span
                              className={`text-xs cursor-pointer ${disabled ? 'text-[#A3A3A3]' : 'text-[#1447E6] hover:underline'}`}
                              role="button"
                              tabIndex={0}
                              onClick={(e) => {
                                e.stopPropagation();
                                setQuotaDetailOrder(order);
                              }}
                            >
                              查看详情
                            </span>
                            {disabled ? (
                              <span className="badge-stopped">
                                {isUsedUp ? '已用完' : `配额不足，剩余 ${remain} 个`}
                              </span>
                            ) : (
                              <span className="badge-loading">剩余 {remain} 个</span>
                            )}
                          </div>
                          <div className={`flex flex-wrap gap-x-6 gap-y-1 text-xs ${disabled ? 'text-[#A3A3A3]' : 'text-[#737373]'}`}>
                            <span>配额：{used}/{total} 已使用</span>
                            {selectedType === 'batch' && <span>本次需要：{protectAgentCount} 个</span>}
                            <span>有效期：{getOrderStartTime(order)} ~ {getOrderEndTime(order)}</span>
                          </div>
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setOpenProtectModalVisible(false)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              disabled={protectOpenType === 'quota' && !selectedQuotaOrderId}
              onClick={() => {
                if (protectOpenType === "new") {
                  createOrder(selectedType === 'batch' ? selectedAgentIds?.length : 1, protectPeriod, 0);
                } else if (selectedQuotaOrderId) {
                  bindOrder(selectedQuotaOrderId, selectedAgentIds);
                }
                setOpenProtectModalVisible(false);
              }}
            >
              {protectOpenType === 'quota' ? '确认使用配额' : '确认开启'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
