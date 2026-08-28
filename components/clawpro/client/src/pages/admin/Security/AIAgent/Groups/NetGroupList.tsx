import React, { useState, useEffect } from "react";
import { toast } from "sonner";
import { Pencil, Database } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { PanelTitle, BodyText, MetaText } from "@/components/ui/Typography";
import { Transfer } from "@/components/ui/transfer";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
// import { getSecurityGroup, getSecurityGroupPolicies } from "../../api";
import {
  EGRESS_RULE,
  CSIP_AI_AGENT_NET_RULE,
} from "../constants";
import { setMaxRemoteStorage } from "../Common/tablePanelColumnUtil";
import { requestApi } from "../Common/requestApi";
import { hostVersionMap } from "./BashPolicy/Constants";
import { RecommendEnableTag } from "../Common/RecommendEnableTag";

export default function NetGroupList({
  isGetAllMachinesLoading,
  aiAgentHostList,
  storageGroupData,
  isCVMEnable,
  setIsCVMEnable,
}: any) {
  const [isLoading, setIsLoading] = useState(false);
  const [realAgentHosts, setRealAgentHosts] = useState([] as any);
  const [hostScope, setHostScope] = useState("0");
  const [currentStep, setCurrentStep] = useState("0");
  const [selectedRegion, setSelectedRegion] = useState([] as any);
  const [fetchLoading, setFetchLoading] = useState(false);
  const [changeSecurityGroupModalVisible, setChangeSecurityGroupModalVisible] = useState(false);
  const [selectedMachines, setSelectedMachines] = useState([] as any);
  const [selectedQuuidList, setSelectedQuuidList] = useState([] as any);
  const [tempSelectedQuuidList, setTempSelectedQuuidList] = useState([] as any);
  const [tempSelectedMachines, setTempSelectedMachines] = useState([] as any);
  const [selectHostModalVisible, setSelectHostModalVisible] = useState(false);
  const [groupsDataMap, setGroupsDataMap] = useState({} as any); // {1:{SecurityGroupId:"sg-jzecnxed",SecurityGroupName:''}, 8:{...}}
  const [groupsDataMapInstanceIds, setGroupsDataMapInstanceIds] = useState({} as any); // {1:["ins-089cnfcd", "..."],8:[...]]}
  const [groupRuleListMap, setGroupRuleListMap] = useState({} as any); // {1:[{"PolicyIndex":0,"Port":"ALL","CidrBlock":"10.0.0.0/8","Action":"DROP","PolicyDescription":"阻止内网IP","Protocol":"ALL"}...],8:[...]}
  const [groupRuleListModalVisible, setGroupRuleListModalVisible] = useState(false);
  const [lhGroupsData, setLhGroupsData] = useState({} as any); // {'lhins-***':[Action:'',Port:'']} 已开启的
  const [selectedMachinesModalVisible, setSelectedMachinesModalVisible] = useState(false);

  const delCVMRules = async () => {
    setChangeSecurityGroupModalVisible(false);
    setIsLoading(true);
    const regionIds = Object.keys(groupsDataMap || {});
    if (!regionIds?.length) {
      setIsLoading(false);
      return;
    }
    const res: any = await Promise.all(
      regionIds
        .map(d => [
          requestApi({
            cmd: "DeleteSecurityGroupPolicies",
            data: {
              SecurityGroupId: groupsDataMap[d]?.SecurityGroupId,
              SecurityGroupPolicySet: { Egress: EGRESS_RULE },
            },
            regionId: Number(d),
            serviceType: "vpc",
          }),
          requestApi({
            cmd: "DisassociateSecurityGroups",
            data: {
              InstanceIds: groupsDataMapInstanceIds[d],
              SecurityGroupIds: [groupsDataMap?.[d]?.SecurityGroupId],
            },
            regionId: Number(d),
            serviceType: "cvm",
          }),
        ])
        .flat(2)
    );
    if (res?.some?.((d: { code: number }) => d?.code === 0)) {
      await Promise.all(
        regionIds.map(d =>
          requestApi({
            cmd: "DeleteSecurityGroup",
            data: {
              SecurityGroupId: groupsDataMap[d]?.SecurityGroupId,
            },
            regionId: Number(d),
            serviceType: "vpc",
          })
        )
      );
      setMaxRemoteStorage(
        CSIP_AI_AGENT_NET_RULE,
        JSON.stringify("{}"),
        3650 * 24 * 3600
      );
    }
    const ids = Array.from(
      new Set(realAgentHosts?.map?.((d: any) => d?.RegionInfo?.RegionId) || [])
    );
    getCVMSecurityGroups(ids);
    setIsLoading(false);
  };

  const getCVMSecurityGroups = async (regionIds: any[]) => {
    setIsLoading(true);
    if (!regionIds?.length) {
      setIsLoading(false);
      return;
    }
    console.log(6677, storageGroupData, regionIds);
    if (
      storageGroupData &&
      typeof storageGroupData === "object" &&
      Object.keys(storageGroupData)?.length > 0 &&
      Object.keys(storageGroupData)?.some?.(
        d => storageGroupData[d]?.SecurityGroupId
      )
    ) {
      const regionIdArr = regionIds.filter(
        (d: string | number) => storageGroupData[d]?.SecurityGroupId
      );
      if (regionIdArr?.length) {
        // 查询该安全组是否存在
        const res: any = await Promise.all(
          regionIdArr
            .map((d: string | number) => [
              requestApi({
                cmd: 'DescribeSecurityGroups',
                data: {
                  Filters: [{ Name: 'security-group-id', Values: [storageGroupData[d]?.SecurityGroupId] }],
                  Limit: '1',
                  Offset: '0',
                },
                regionId: Number(d),
                serviceType: 'vpc',
                showInnerTips: false,
              }),
              requestApi({
                cmd: "DescribeInstances",
                data: {
                  Filters: [
                    {
                      Name: "security-group-id",
                      Values: [storageGroupData[d]?.SecurityGroupId],
                    },
                  ],
                  Limit: 100,
                  Offset: 0,
                },
                regionId: Number(d),
                serviceType: "cvm",
                showInnerTips: false,
              }),
              // getSecurityGroupPolicies();
              requestApi({
                cmd: "DescribeSecurityGroupPolicies",
                data: {
                  SecurityGroupId: storageGroupData[d]?.SecurityGroupId,
                },
                regionId: Number(d),
                serviceType: "vpc",
                showInnerTips: false,
              }),
            ])
            .flat(2)
        );
        const realObj =
          regionIdArr.reduce(
            (pre: { [x: string]: any }, cur: string | number, i: number) => {
              if (
                res[i * 3]?.SecurityGroupSet?.[0]
                  ?.SecurityGroupId
              ) {
                pre[cur] = res[i * 3]?.SecurityGroupSet?.[0];
              }
              return pre;
            },
            {}
          ) || {};
        const isEnable =
          realObj &&
          typeof realObj === "object" &&
          Object.keys(realObj)?.length > 0;
        const insObj =
          regionIdArr?.reduce?.(
            (pre: { [x: string]: any }, cur: string | number, i: number) => {
              pre[cur] =
                res[i * 3 + 1]?.InstanceSet?.map?.(
                  (d: { InstanceId: any }) => d?.InstanceId
                ) || [];
              return pre;
            },
            {}
          ) || {};
        const rulesObj =
          regionIdArr?.reduce?.(
            (pre: { [x: string]: any }, cur: string | number, i: number) => {
              pre[cur] =
                res[i * 3 + 2]?.SecurityGroupPolicySet
                  ?.Egress || [];
              return pre;
            },
            {}
          ) || {};
        console.log(6688, res, realObj, insObj, rulesObj);
        setGroupsDataMap(realObj);
        setIsCVMEnable(isEnable);
        setGroupsDataMapInstanceIds(insObj);
        setGroupRuleListMap(rulesObj);
      }
    }
    setIsLoading(false);
  };

  const createGroupAndLinkHosts = async () => {
    setChangeSecurityGroupModalVisible(false);
    if (hostScope === "1" && !selectedMachines?.length) {
      return;
    }
    const regionIds = Array.from(
      new Set(selectedRegion.map((d: any) => d?.RegionId))
    );
    console.log(668899, hostScope, regionIds, selectedMachines);
    if (!regionIds?.length) {
      return;
    }
    try {
      const addRegionIds: any = regionIds?.filter?.(
        d => !Object.keys(groupsDataMap)?.includes?.(String(d))
      );
      console.log(31144, addRegionIds);
      // 创建安全组
      if (addRegionIds?.length) {
        const res: any = await Promise.all(
          addRegionIds.map((d: any) =>
            requestApi({
              cmd: "CreateSecurityGroupWithPolicies",
              data: {
                GroupDescription: "",
                GroupName: `云安全中心AI Agent内网管控安全组`,
                ProjectId: "0",
                SecurityGroupPolicySet: { Ingress: [], Egress: EGRESS_RULE },
                Tags: [],
              },
              regionId: Number(d),
              serviceType: "vpc",
            })
          )
        );
        console.log(30044, res);
        const resObj: any = {
          ...(groupsDataMap || {}),
          ...(res?.reduce?.(
            (pre: any, cur: any, i: any) => {
              if (
                typeof cur?.SecurityGroup === "object" &&
                Object.keys(cur?.SecurityGroup || {})?.length > 0
              ) {
                pre[addRegionIds[i]] = cur?.SecurityGroup || {};
              }
              return pre;
            },
            {}
          ) || {}),
        };
        console.log(30054, resObj);
        if (Object.keys(resObj)?.length) {
          setIsCVMEnable(true);
          await setMaxRemoteStorage(
            CSIP_AI_AGENT_NET_RULE,
            JSON.stringify(resObj),
            3650 * 24 * 3600
          );
        }
        setGroupsDataMap(resObj);
        setGroupRuleListMap(
          regionIds.reduce((pre: any, cur: any) => {
            pre[cur] = EGRESS_RULE;
            return pre;
          }, {})
        );
        // 绑定机器
        await Promise.all(
          regionIds.map(regionId =>
            requestApi({
              cmd: "AssociateSecurityGroups",
              data: {
                InstanceIds: (hostScope === "0"
                  ? realAgentHosts
                  : selectedMachines
                )
                  ?.filter?.(
                    (host: any) =>
                      host?.RegionInfo?.RegionId === Number(regionId)
                  )
                  ?.map?.(
                    (host: any) =>
                      host?.MachineExtraInfo?.InstanceID || host?.InstanceID
                  ),
                SecurityGroupIds: [resObj?.[regionId as any]?.SecurityGroupId],
              },
              regionId: Number(regionId),
              serviceType: "cvm",
            })
          )
        );
        const insObj =
          regionIds.reduce((pre: any, cur: any) => {
            pre[cur] = (hostScope === "0" ? realAgentHosts : selectedMachines)
              ?.filter?.((d: any) => d?.RegionInfo?.RegionId === Number(cur))
              ?.map?.(
                (d: any) => d?.MachineExtraInfo?.InstanceID || d?.InstanceID
              );
            return pre;
          }, {}) || {};
        setGroupsDataMapInstanceIds(insObj);
      } else {
        // 绑定或解绑
        const addHostsMap =
          Object.keys(groupsDataMapInstanceIds)?.reduce?.((pre: any, cur) => {
            const insIds = (
              hostScope === "0" ? realAgentHosts : selectedMachines
            )
              ?.filter?.((d: any) => d?.RegionInfo?.RegionId === Number(cur))
              ?.map?.(
                (d: any) => d?.MachineExtraInfo?.InstanceID || d?.InstanceID
              );
            const addInsIds = insIds?.filter?.(
              (d: any) => !groupsDataMapInstanceIds[cur]?.includes?.(d)
            );
            if (addInsIds?.length) {
              pre[cur] = addInsIds;
            }
            return pre;
          }, {}) || {}; // {1:['ins-*', ''], 8:[]}
        const delHostsMap =
          Object.keys(groupsDataMapInstanceIds)?.reduce?.((pre: any, cur) => {
            const insIds = (
              hostScope === "0" ? realAgentHosts : selectedMachines
            )
              ?.filter?.((d: any) => d?.RegionInfo?.RegionId === Number(cur))
              ?.map?.(
                (d: any) => d?.MachineExtraInfo?.InstanceID || d?.InstanceID
              );
            const delInsIds = groupsDataMapInstanceIds[cur]?.filter?.(
              (d: any) => !insIds?.includes?.(d)
            );
            if (delInsIds?.length) {
              pre[cur] = delInsIds;
            }
            return pre;
          }, {}) || {}; // {1:['ins-*', ''], 8:[]}
        console.log(300887, addHostsMap, delHostsMap);
        if (
          Object.keys(addHostsMap)?.length ||
          Object.keys(delHostsMap)?.length
        ) {
          await Promise.all(
            Object.keys(addHostsMap)
              .map(d =>
                requestApi({
                  cmd: "AssociateSecurityGroups",
                  data: {
                    InstanceIds: addHostsMap[d],
                    SecurityGroupIds: [groupsDataMap?.[d]?.SecurityGroupId],
                  },
                  regionId: Number(d),
                  serviceType: "cvm",
                })
              )
              ?.concat?.(
                Object.keys(delHostsMap).map(d =>
                  requestApi({
                    cmd: "DisassociateSecurityGroups",
                    data: {
                      InstanceIds: delHostsMap[d],
                      SecurityGroupIds: [groupsDataMap?.[d]?.SecurityGroupId],
                    },
                    regionId: Number(d),
                    serviceType: "cvm",
                  })
                )
              )
          );
          setGroupsDataMapInstanceIds(
            (hostScope === "0" ? realAgentHosts : selectedMachines)?.reduce?.(
              (pre: any, cur: any) => {
                const regionId = cur?.RegionInfo?.RegionId;
                pre[regionId] = Array.isArray(pre[regionId])
                  ? pre[regionId]?.concat?.(
                    cur?.MachineExtraInfo?.InstanceID || cur?.InstanceID
                  )
                  : [cur?.MachineExtraInfo?.InstanceID || cur?.InstanceID];
                return pre;
              },
              {}
            ) || {}
          );
        }
      }
      toast.success("操作成功");
    } catch (e) {
      console.log(e);
    }
  };

  useEffect(() => {
    if (changeSecurityGroupModalVisible) {
      setHostScope("0");
      setCurrentStep("0");
      setSelectedRegion(
        realAgentHosts?.map?.((d: { RegionInfo: any }) => d?.RegionInfo) || []
      );
    }
  }, [changeSecurityGroupModalVisible]);

  useEffect(() => {
    if (!isGetAllMachinesLoading) {
      const realCVMData =
        aiAgentHostList?.filter?.(
          (d: any) =>
            d?.ProtectType === "Flagship" && d?.MachineType === "CVM"
        ) || [];
      const regionIds = Array.from(
        new Set(
          realCVMData?.map?.(
            (d: { RegionInfo: { RegionId: any } }) => d?.RegionInfo?.RegionId
          ) || []
        )
      );
      setHostScope("0");
      setRealAgentHosts(realCVMData);
      setSelectedRegion(
        realCVMData?.map?.((d: { RegionInfo: any }) => d?.RegionInfo) || []
      );
      getCVMSecurityGroups(regionIds);
    }
  }, [isGetAllMachinesLoading]);

  return (
    <div className="csip-AIAgent-netGroup-content">
      <SurfaceCard className="p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 min-w-0">
            <img
              src="/icon/icon-aiagent-netgroup.svg"
              width={36}
              height={36}
              alt=""
              aria-hidden="true"
              className="shrink-0"
            />
            <div className="min-w-0">
              <PanelTitle>AI Agent内网管控安全组规则</PanelTitle>
              <BodyText tone="secondary" className="mt-1">
                开启后，将自动为 AI Agent旗舰版资产添加安全组规则：默认阻止内网访问，防止内网攻击面扩大。
              </BodyText>
            </div>
          </div>
          {isCVMEnable ? (
            <Pencil
              className="w-4 h-4 mt-1 shrink-0 cursor-pointer text-[#737373] hover:text-[#1447E6]"
              onClick={() => setChangeSecurityGroupModalVisible(true)}
            />
          ) : (
            <RecommendEnableTag>未开启，推荐开启</RecommendEnableTag>
          )}
        </div>

        <div className="border-t border-[#E5E5E5] my-4" />

        <div className="flex items-center text-sm">
          <div className="basis-[120px] shrink-0">
            <span className="text-[#737373]">安全组规则</span>
          </div>
          <div className="flex-1 text-[#525252]">
            {isCVMEnable ? (
              <>
                内网访问默认阻止（出站规则：
                <a
                  className="text-[#1447E6] cursor-pointer hover:underline"
                  onClick={() => setGroupRuleListModalVisible(true)}
                >
                  {(
                    Object.keys(groupRuleListMap)?.reduce?.((pre, cur) => {
                      pre = pre.concat?.(
                        groupRuleListMap[cur]?.map?.((d: any) => ({
                          ...d,
                          RegionId: Number(cur),
                          RegionName: aiAgentHostList?.find?.(
                            (d: { RegionInfo: { RegionId: number } }) =>
                              d?.RegionInfo?.RegionId === Number(cur)
                          )?.RegionName,
                        }))
                      );
                      return pre;
                    }, []) || []
                  )?.length || 0}
                </a>
                {"）"}
              </>
            ) : (
              "内网访问默认阻止"
            )}
          </div>
          <div className="basis-[120px] shrink-0">
            <span className="text-[#737373]">生效资产</span>
          </div>
          <div className="flex-1 text-[#525252]">
            {!isCVMEnable ? (
              "0台"
            ) : (
              <span
                className={
                  (
                    Object.values(groupsDataMapInstanceIds || {})?.flat?.(
                      2
                    ) || []
                  )?.length
                    ? "text-[#1447E6] cursor-pointer"
                    : "text-[#737373]"
                }
                onClick={() => setSelectedMachinesModalVisible(true)}
              >
                {Object.values(groupsDataMapInstanceIds || {})?.flat?.(2)
                  ?.length || 0}{" "}
                台
              </span>
            )}
          </div>
          {!isCVMEnable && (
            <Switch
              className="shrink-0 ml-4"
              checked={false}
              onCheckedChange={() => setChangeSecurityGroupModalVisible(true)}
            />
          )}
        </div>
      </SurfaceCard>

      <Dialog
        open={changeSecurityGroupModalVisible}
        onOpenChange={setChangeSecurityGroupModalVisible}
      >
        <DialogContent className="sm:max-w-[920px]">
          <DialogHeader>
            <DialogTitle>
              {isCVMEnable
                ? "编辑 AI Agent 内网管控安全组策略"
                : "确认开启 AI Agent 内网管控安全组策略？"}
            </DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="text-sm text-[#525252] leading-[1.5]">
              {`${isCVMEnable ? "确认" : "开启"}后，将自动为 AI Agent 旗舰版资产添加安全组规则：默认阻止内网访问，防止内网攻击面扩大。`}
            </div>

            {/* 步骤条 */}
            <div className="flex items-center gap-3 mt-5 mb-5">
              {[
                { key: "0", label: "选择防护资产" },
                { key: "1", label: "确认安全组策略" },
              ].map((step, idx) => {
                const active = currentStep === step.key;
                const done = Number(currentStep) > Number(step.key);
                return (
                  <React.Fragment key={step.key}>
                    <div className="flex items-center gap-2">
                      <span
                        className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium ${
                          active || done
                            ? "bg-[#1447E6] text-white"
                            : "bg-[#F5F5F5] text-[#A3A3A3]"
                        }`}
                      >
                        {idx + 1}
                      </span>
                      <span
                        className={`text-sm ${
                          active
                            ? "text-[#0A0A0A] font-medium"
                            : done
                              ? "text-[#525252]"
                              : "text-[#A3A3A3]"
                        }`}
                      >
                        {step.label}
                      </span>
                    </div>
                    {idx === 0 && (
                      <span className="flex-1 h-px bg-[#E5E5E5]" />
                    )}
                  </React.Fragment>
                );
              })}
            </div>

            {currentStep === "0" ? (
              <div className="space-y-4">
                {/* 选择资产 */}
                <div className="flex items-start gap-4">
                  <div className="w-[88px] shrink-0 pt-1.5 text-sm text-[#737373]">
                    选择资产
                  </div>
                  <div className="flex-1 min-w-0">
                    <RadioGroup
                      disabled={isLoading}
                      value={hostScope}
                      onValueChange={value => {
                        setHostScope(value);
                        if (value === "1") {
                          setTempSelectedQuuidList(selectedQuuidList);
                          setTempSelectedMachines(selectedMachines);
                          setSelectHostModalVisible(true);
                        } else {
                          setSelectedRegion(
                            realAgentHosts?.map?.(
                              (d: { RegionInfo: any }) => d?.RegionInfo
                            )
                          );
                        }
                      }}
                      className="flex flex-col gap-2"
                    >
                      <div className="flex items-center space-x-2">
                        <RadioGroupItem value="0" id="scope-all" />
                        <Label htmlFor="scope-all" className="text-sm text-[#0A0A0A] font-normal cursor-pointer">
                          全部 AI Agent 旗舰版资产
                        </Label>
                      </div>
                      <div
                        className="flex items-center space-x-2"
                        onClick={() => {
                          setTempSelectedQuuidList(selectedQuuidList);
                          setTempSelectedMachines(selectedMachines);
                          setSelectHostModalVisible(true);
                        }}
                      >
                        <RadioGroupItem value="1" id="scope-select" />
                        <Label htmlFor="scope-select" className="text-sm text-[#0A0A0A] font-normal cursor-pointer">
                          直接选择
                        </Label>
                      </div>
                    </RadioGroup>

                    {/* 已选资产提示条 */}
                    <div className="mt-3 px-3 py-2.5 border-l-2 border-[#1447E6] bg-[#EFF6FF] rounded-r-[4px]">
                      <span className="text-sm text-[#737373]">已选资产：</span>
                      <span
                        className={`text-sm ${
                          (hostScope === "0"
                            ? realAgentHosts
                            : selectedMachines
                          )?.length
                            ? "text-[#1447E6] cursor-pointer hover:underline"
                            : "text-[#A3A3A3]"
                        }`}
                        onClick={() => {
                          if (
                            (hostScope === "0"
                              ? realAgentHosts
                              : selectedMachines
                            )?.length
                          ) {
                            setSelectedMachinesModalVisible(true);
                          }
                        }}
                      >
                        {`${hostScope === "0" ? realAgentHosts?.length || 0 : selectedMachines?.length || 0} 个`}
                      </span>
                    </div>
                  </div>
                </div>

                {/* 安全组策略预览 */}
                <div className="rounded-[4px] bg-[#FAFAFA] p-4">
                  {hostScope === "1" && !selectedMachines?.length ? (
                    <div className="text-sm text-[#0A0A0A]">
                      您当前未选择任何资产，将移除您已有的安全组策略
                    </div>
                  ) : (
                    <div className="text-sm text-[#0A0A0A]">
                      将为您创建/编辑下述安全组策略：
                    </div>
                  )}
                  <div className="mt-3 space-y-3">
                    {Object.keys(groupsDataMap)?.length && Array.from(new Set(selectedRegion.map((d: any) => d?.RegionId)))?.every?.((d: any) =>
                      Object.keys(groupsDataMap)?.includes?.(String(d)),
                    ) ? null : (
                      <SurfaceInner className="p-3">
                        <div className="flex items-center gap-1.5 text-sm text-[#737373]">
                          <span className="w-1.5 h-1.5 rounded-full bg-[#1447E6]" />
                          创建安全组：
                        </div>
                        {Array.from(
                          new Set(
                            selectedRegion.map((d: any) => d?.RegionId)
                          )
                        )?.filter?.(d => !Object.keys(groupsDataMap)?.includes?.(String(d)))?.map?.((d, i) => (
                          <div key={i} className="mt-1.5 text-sm text-[#0A0A0A]">
                            云安全中心 AI Agent 内网管控安全组
                          </div>
                        ))}
                      </SurfaceInner>
                    )}
                    {Object.keys(groupsDataMap)?.some?.((d: any) =>
                      Array.from(
                        new Set(
                          selectedRegion.map(
                            (d: { RegionId: any }) => d?.RegionId
                          )
                        )
                      )?.includes?.(Number(d))
                    )
                      ? (Object.keys(groupsDataMap)
                        ?.filter?.((d: any) =>
                          Array.from(
                            new Set(
                              selectedRegion.map(
                                (d: { RegionId: any }) => d?.RegionId
                              )
                            )
                          )?.includes?.(Number(d))
                        )
                        ?.map?.((d: any) => (
                          <SurfaceInner key={d} className="p-3">
                            <div className="flex items-center gap-1.5 text-sm text-[#737373]">
                              <span className="w-1.5 h-1.5 rounded-full bg-[#1447E6]" />
                              编辑安全组（当前已有安全组）：
                            </div>
                            <div className="mt-1.5 text-sm text-[#0A0A0A]">
                              {groupsDataMap?.[d]?.SecurityGroupName}
                            </div>
                          </SurfaceInner>
                        )) as any)
                      : null}
                  </div>
                </div>
              </div>
            ) : (
              <div className="rounded-[4px] bg-[#FAFAFA] p-4">
                <div className="text-sm text-[#0A0A0A]">
                  {`将${hostScope === "1" && !selectedMachines?.length ? "移除" : "创建/编辑"}下述安全组：`}
                </div>
                <div className="mt-3 space-y-3">
                  {(hostScope === '1' && !selectedMachines?.length
                    ? Object.keys(groupsDataMap)
                    : Array.from(new Set(selectedRegion.map((d: any) => d?.RegionId)))
                  ).map((d, i) => (
                    <SurfaceInner key={i}>
                      <div className="grid grid-cols-2 divide-x divide-[#EAEEF4]">
                        <div className="px-4 py-3">
                          <div className="text-xs text-[#737373]">安全组名称</div>
                          <div className="mt-1 text-sm text-[#0A0A0A] font-medium">
                            云安全中心 AI Agent 内网管控安全组
                          </div>
                        </div>
                        <div className="px-4 py-3">
                          <div className="text-xs text-[#737373]">规则内容</div>
                          <div className="mt-1 text-sm text-[#0A0A0A]">
                            出站规则
                            <a
                              className="ml-1 text-[#1447E6] hover:underline cursor-pointer"
                              onClick={() => setGroupRuleListModalVisible(true)}
                            >
                              ({isCVMEnable
                                ? Object.values(groupRuleListMap)?.flat?.(2)?.length || 0
                                : EGRESS_RULE.length})
                            </a>
                          </div>
                        </div>
                      </div>
                    </SurfaceInner>
                  ))}
                </div>
              </div>
            )}
          </DialogBody>
          <DialogFooter>
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={() => setChangeSecurityGroupModalVisible(false)}
            >
              {"取消"}
            </Button>
            {currentStep === "0" ? (
              <Button
                variant="dialog-confirm"
                size="claw-sm"
                disabled={isGetAllMachinesLoading || isLoading}
                onClick={() => {
                  if (!realAgentHosts?.length) {
                    toast.error("暂无AI Agent旗舰版资产");
                    return;
                  }
                  if (!isCVMEnable && hostScope === '1' && !selectedMachines?.length) {
                    toast.error("请至少选择一台OpenClaw");
                    return;
                  }
                  setCurrentStep("1");
                }}
              >
                {"下一步"}
              </Button>
            ) : currentStep === "1" ? (
              <>
                <Button variant="claw-outline" size="claw-sm" onClick={() => setCurrentStep("0")}>
                  {"上一步"}
                </Button>
                <Button
                  variant="dialog-confirm"
                  size="claw-sm"
                  disabled={isGetAllMachinesLoading || isLoading}
                  onClick={() => {
                    if (hostScope === "1" && !selectedMachines?.length) {
                      delCVMRules();
                    } else {
                      createGroupAndLinkHosts();
                    }
                  }}
                >
                  {"保存"}
                </Button>
              </>
            ) : null}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={selectHostModalVisible}
        onOpenChange={setSelectHostModalVisible}
      >
        <DialogContent className="sm:max-w-[920px]">
          <DialogHeader>
            <DialogTitle>{"选择OpenClaw"}</DialogTitle>
          </DialogHeader>
          <div style={{ marginTop: -5 }}>
            <Transfer<any>
              dataSource={(aiAgentHostList ?? [])
                .filter(
                  (h: any) =>
                    h?.Quuid && h?.ProtectType === "Flagship",
                )
                .map((h: any) => ({ ...h, key: h?.Quuid }))}
              rowKey="key"
              targetKeys={tempSelectedQuuidList}
              onChange={(nextKeys) => {
                setTempSelectedQuuidList(nextKeys);
                setTempSelectedMachines(
                  (aiAgentHostList ?? []).filter((h: any) =>
                    nextKeys.includes(h?.Quuid),
                  ),
                );
              }}
              showSearch
              searchPlaceholder={["搜索资产名称 / ID / IP", "搜索已选资产"]}
              pagination={{ pageSize: 8 }}
              height={320}
              titles={["全部 AI Agent 资产", "已选 AI Agent 资产"]}
              filterOption={(input, h: any) => {
                const needle = input.toLowerCase();
                return [
                  h?.OpenClawName,
                  h?.MachineName,
                  h?.InstanceID,
                  h?.MachineIp,
                ]
                  .filter((v) => typeof v === "string")
                  .some((v) => v.toLowerCase().includes(needle));
              }}
              columns={[
                {
                  key: "name",
                  header: "Agent 名称 / ID",
                  render: (h: any) => (
                    <div className="min-w-0">
                      <div className="truncate text-[var(--text-emphasis)]">
                        {h?.OpenClawName || h?.MachineName || "-"}
                      </div>
                      <MetaText className="block truncate">
                        {h?.InstanceID || "-"}
                      </MetaText>
                    </div>
                  ),
                },
                {
                  key: "version",
                  header: "防护版本",
                  width: 100,
                  render: (h: any) => hostVersionMap[h?.ProtectType] ?? "-",
                },
                {
                  key: "ip",
                  header: "内网IP",
                  width: 140,
                  render: (h: any) => h?.MachineIp || "-",
                },
              ]}
            />
          </div>
          <DialogFooter>
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={() => setSelectHostModalVisible(false)}
            >
              <span>{"取消"}</span>
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              disabled={fetchLoading}
              onClick={() => {
                if (!isCVMEnable && !tempSelectedMachines?.length) {
                  toast.error("请至少选择一台OpenClaw");
                  return;
                }
                const hosts = tempSelectedMachines?.map?.(
                  (d: {
                    InstanceID: any;
                    MachineExtraInfo: { InstanceID: any };
                  }) =>
                    aiAgentHostList?.find?.(
                      (a: { InstanceID: any }) =>
                        a?.InstanceID ===
                        (d?.InstanceID || d?.MachineExtraInfo?.InstanceID)
                    )
                );
                setSelectedRegion(
                  hosts?.map?.((d: { RegionInfo: any }) => d?.RegionInfo)
                );
                setSelectedMachines(hosts);
                setSelectedQuuidList(
                  hosts
                    ?.filter?.((d: { Quuid: any }) => d?.Quuid)
                    ?.map?.((d: { Quuid: any }) => d?.Quuid)
                );
                setSelectHostModalVisible(false);
              }}
            >
              <span>{"保存"}</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={groupRuleListModalVisible}
        onOpenChange={setGroupRuleListModalVisible}
      >
        <DialogContent className="sm:max-w-[800px]">
          <DialogHeader>
            <DialogTitle>{`安全组规则-出站规则（${!isCVMEnable ? EGRESS_RULE.length : Object.values(groupRuleListMap)?.flat?.(2)?.length || 0}）`}</DialogTitle>
          </DialogHeader>
          <div style={{ marginTop: -5 }}>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>来源</TableHead>
                  <TableHead>协议端口</TableHead>
                  <TableHead>策略</TableHead>
                  <TableHead>备注</TableHead>
                  <TableHead>修改时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(!isCVMEnable
                  ? EGRESS_RULE
                  : ((Object.keys(groupRuleListMap)?.reduce?.((pre, cur) => {
                    pre = pre.concat?.(
                      groupRuleListMap[cur]?.map?.((d: any) => ({
                        ...d,
                        RegionId: Number(cur),
                        RegionName: aiAgentHostList?.find?.(
                          (d: { RegionInfo: { RegionId: number } }) =>
                            d?.RegionInfo?.RegionId === Number(cur)
                        )?.RegionName,
                      }))
                    );
                    return pre;
                  }, []) || []) as any)
                )?.map?.((item: any, index: number) => (
                  <TableRow key={index}>
                    <TableCell>{item?.CidrBlock || "--"}</TableCell>
                    <TableCell>{item?.Port || "--"}</TableCell>
                    <TableCell>
                      {item?.Action === "DROP" ? (
                        <Badge variant="destructive">{"拒绝"}</Badge>
                      ) : (
                        <Badge className="bg-green-500 text-white">
                          {"允许"}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>{item?.PolicyDescription || "--"}</TableCell>
                    <TableCell>{item?.ModifyTime || "--"}</TableCell>
                  </TableRow>
                ))}
                {!(
                  !isCVMEnable
                    ? EGRESS_RULE
                    : ((Object.keys(groupRuleListMap)?.reduce?.((pre, cur) => {
                      pre = pre.concat?.(groupRuleListMap[cur] || []);
                      return pre;
                    }, []) || []) as any)
                )?.length && (
                    <TableRow>
                      <TableCell
                        colSpan={5}
                        className="text-center text-[var(--text-weak)]"
                      >
                        {"暂无数据"}
                      </TableCell>
                    </TableRow>
                  )}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog
        open={selectedMachinesModalVisible}
        onOpenChange={setSelectedMachinesModalVisible}
      >
        <DialogContent className="sm:max-w-[800px]">
          <DialogHeader>
            <DialogTitle>{`已选资产（${(changeSecurityGroupModalVisible
              ? hostScope === "0"
                ? realAgentHosts || []
                : selectedMachines
              : !isCVMEnable
                ? realAgentHosts
                : Object.values(groupsDataMapInstanceIds || {})?.flat?.(2)
            )?.length || 0
              }台）`}</DialogTitle>
          </DialogHeader>
          <div style={{ marginTop: -5 }}>
            <Alert>
              <AlertDescription>
                {
                  "您正在设置动态选择方式，仅展示当前匹配资产，后续资产范围将基于所选内容范围变化而变化。"
                }
              </AlertDescription>
            </Alert>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>OpenClaw 名称</TableHead>
                  {/* <TableHead>IP地址</TableHead> */}
                  <TableHead>资产类型</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(changeSecurityGroupModalVisible
                  ? hostScope === "0"
                    ? realAgentHosts || []
                    : selectedMachines || []
                  : !isCVMEnable
                    ? realAgentHosts || []
                    : Object.values(groupsDataMapInstanceIds || {})
                      ?.flat?.(2)
                      ?.map?.(d =>
                        aiAgentHostList?.find?.(
                          (a: { InstanceID: unknown }) => a?.InstanceID === d
                        )
                      )
                )?.map?.((item: any, index: number) => (
                  <TableRow key={index}>
                    <TableCell>
                      <div>
                        {/* <span
                          className="text-sm"
                          title={
                            item?.InstanceID ||
                            item?.MachineExtraInfo?.InstanceID
                          }
                        >
                          {item?.InstanceID ||
                            item?.MachineExtraInfo?.InstanceID ||
                            "-"}
                        </span> */}
                        <div>
                          <span
                            className="text-sm"
                            title={item?.OpenClawName}
                          >
                            {item?.OpenClawName || "-"}
                          </span>
                        </div>
                      </div>
                    </TableCell>
                    {/* <TableCell>
                      <div className="text-[#0A0A0A]">
                        {"公："}
                        {item?.MachineExtraInfo?.WanIP ||
                          item?.MachineWanIp ||
                          "-"}
                      </div>
                      <div className="text-[#0A0A0A]">
                        {"内："}
                        {item?.MachineExtraInfo?.PrivateIP ||
                          item?.MachineIp ||
                          "-"}
                      </div>
                    </TableCell> */}
                    <TableCell>
                      <Database
                        className="w-4 h-4 inline-block"
                        style={{ margin: "-3px 3px 0 0" }}
                      />
                      {item?.MachineType === "LH"
                        ? "Lighthouse"
                        : item?.MachineType}
                    </TableCell>
                  </TableRow>
                ))}
                {!(
                  changeSecurityGroupModalVisible
                    ? hostScope === "0"
                      ? realAgentHosts || []
                      : selectedMachines || []
                    : !isCVMEnable
                      ? realAgentHosts || []
                      : Object.values(groupsDataMapInstanceIds || {})
                        ?.flat?.(2)
                        ?.map?.(d =>
                          aiAgentHostList?.find?.(
                            (a: { InstanceID: unknown }) =>
                              a?.InstanceID === d
                          )
                        )
                )?.length && (
                    <TableRow>
                      <TableCell
                        colSpan={3}
                        className="text-center text-[var(--text-weak)]"
                      >
                        {"暂无数据"}
                      </TableCell>
                    </TableRow>
                  )}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
