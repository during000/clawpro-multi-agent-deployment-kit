/**
 * SecurityGroups（管控配置）
 *
 * 0602 重构：拍平了原本的 L2 Segment「网络管控 / OpenClaw管控」切换。
 *
 * 重构原因：
 *   原结构是 L1 主 Tabs → L2 Segment（网络 / OpenClaw） → L3 内层 Tabs（命令 / IP-DNS）
 *   三层切换叠加，且 L2、L3 都是"药丸分段"视觉，用户区分不出"我在第几层"。
 *   且业务上"网络管控"和"OpenClaw管控"是平级、可同时启用的两个能力域，
 *   Segment 暗示互斥反而误导用户。
 *
 * 现结构（两层即可，且各层视觉语言不撞）：
 *   L1 主 Tabs（下划线，PageTabs）
 *     └─ 管控配置
 *         ├─ 网络管控 SurfaceCard（NetGroupList，自带头部 + 状态徽标）
 *         └─ OpenClaw 管控 SurfaceCard（HostGroupList，内部唯一一层药丸 Tabs：命令 / IP-DNS）
 */

import React, { useState, useEffect } from "react";
import {
  DescribeBashPolicies,
  DescribeRiskDnsPolicyList,
} from "@/pages/admin/Security/api";
import { SectionTitle } from "@/components/ui/Typography";

import NetGroupList from "./NetGroupList";
import HostGroupList from "./HostGroupList";

const SecurityGroups = ({
  isGetAllMachinesLoading,
  aiAgentHostList,
  storageGroupData,
  // 仅设计走查/演示用
  mockBashPolicies,
  mockMaliciousPolicies,
  mockBashPolicyCount,
  mockMaliciousPolicyCount,
}: any) => {
  const [bashPolicyCount, setBashPolicyCount] = useState(0);
  const [maliciousPolicyCount, setMaliciousPolicyCount] = useState(0);
  const [isCVMEnable, setIsCVMEnable] = useState(false);

  const isMock = !!(mockBashPolicies || mockMaliciousPolicies);

  const getInitPolicyCount = async () => {
    if (isMock) return; // mock 模式不打真实接口
    const res: any = await Promise.all([
      DescribeBashPolicies({ Offset: 0, Limit: 1 }),
      DescribeRiskDnsPolicyList({ Offset: 0, Limit: 1 }),
    ]);
    setBashPolicyCount(Math.max((res?.[0]?.TotalCount || 0) - 1, 0));
    setMaliciousPolicyCount(Math.max((res?.[1]?.TotalCount || 0) - 1, 0));
  };

  useEffect(() => {
    getInitPolicyCount();
  }, [isMock]);

  // mock 模式下使用注入计数
  const effectiveBashCount = isMock ? (mockBashPolicyCount ?? 0) : bashPolicyCount;
  const effectiveMaliciousCount = isMock ? (mockMaliciousPolicyCount ?? 0) : maliciousPolicyCount;

  return (
    <div className="flex flex-col space-y-8">
      {/* 网络管控 */}
      <section>
        <SectionTitle className="!text-base !font-semibold mb-4">网络管控</SectionTitle>
        <NetGroupList
          aiAgentHostList={aiAgentHostList}
          isGetAllMachinesLoading={isGetAllMachinesLoading}
          storageGroupData={storageGroupData}
          isCVMEnable={isCVMEnable}
          setIsCVMEnable={setIsCVMEnable}
        />
      </section>

      {/* OpenClaw 管控（标题与 Segment 同行，由 HostGroupList 内部渲染） */}
      <section>
        <HostGroupList
          bashPolicyCount={effectiveBashCount}
          maliciousPolicyCount={effectiveMaliciousCount}
          getInitPolicyCount={getInitPolicyCount}
          aiAgentHostList={aiAgentHostList}
          mockBashPolicies={mockBashPolicies}
          mockMaliciousPolicies={mockMaliciousPolicies}
        />
      </section>
    </div>
  );
};

export default SecurityGroups;
