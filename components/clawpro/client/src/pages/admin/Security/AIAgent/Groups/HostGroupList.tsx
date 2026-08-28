/**
 * HostGroupList（OpenClaw 管控）
 *
 * 0603 调整：
 *   1) 去掉卡片头部（图标 + 标题 + 描述 + 启用状态徽标）；
 *   2) 进一步去掉外层 SurfaceCard 容器，让 Segment 与列表内容直接平铺，
 *      避免与下方表格再嵌一层"白底+边框+padding"，视觉更轻。
 *
 * 结构：
 *   <fragment>
 *     ├─ SegmentGroup（命令管控策略 / IP/DNS 管控策略）
 *     └─ 对应策略列表
 */

import React, { useState } from 'react';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { SectionTitle } from '@/components/ui/Typography';
import { BASH_POLICY, MALICIOUS_POLICY } from '../constants';

import { MaliciousPolicyList } from './MaliciousPolicy/MaliciousPolicyList';
import { BashPolicyList } from './BashPolicy/BashPolicyList';

export default function HostGroupList({
  bashPolicyCount,
  maliciousPolicyCount,
  getInitPolicyCount,
  aiAgentHostList,
  isFromDetail = false,
  // 仅设计走查/演示用
  mockBashPolicies,
  mockMaliciousPolicies,
}: any) {
  const [selectedType, setSelectedType] = useState(BASH_POLICY);

  // 资产详情页内嵌使用时（isFromDetail），不再包一层卡片头，避免双层卡片
  if (isFromDetail) {
    return (
      <div>
        <div className="mt-2.5 mb-5">
          {/* 停服态豁免：策略类型切换属于视图切换（查看类），保持可用 */}
          <SegmentGroup className="ml-2.5" data-billing-exempt>
            <SegmentOption
              className="!px-3"
              active={selectedType === BASH_POLICY}
              onClick={() => setSelectedType(BASH_POLICY)}
            >
              {`命令管控策略（${bashPolicyCount}）`}
            </SegmentOption>
            <SegmentOption
              className="!px-3"
              active={selectedType === MALICIOUS_POLICY}
              onClick={() => setSelectedType(MALICIOUS_POLICY)}
            >
              {`IP/DNS管控策略（${maliciousPolicyCount}）`}
            </SegmentOption>
          </SegmentGroup>
        </div>
        {selectedType === BASH_POLICY ? (
          <BashPolicyList
            isFromDetail={isFromDetail}
            hasFlagship={aiAgentHostList?.some?.((d: { ProtectType: string }) => d?.ProtectType === 'Flagship')}
            getInitPolicyCount={getInitPolicyCount}
            aiAgentHostList={aiAgentHostList}
            mockData={mockBashPolicies}
          />
        ) : (
          <MaliciousPolicyList
            hasFlagship={aiAgentHostList?.some?.((d: { ProtectType: string }) => d?.ProtectType === 'Flagship')}
            getInitPolicyCount={getInitPolicyCount}
            aiAgentHostList={aiAgentHostList}
            mockData={mockMaliciousPolicies}
          />
        )}
      </div>
    );
  }

  return (
    <>
      {/* 同行布局：左侧分区标题，右侧策略类型切换 Segment
          停服态豁免：命令管控策略 / IP/DNS 管控策略 属于「视图切换」查看类操作，
          不改变任何后端数据，停服时应保持可用。整组打data-billing-exempt，
          让 overlay 的灰化 CSS 与点击拦截同时放行；里面按钮若自身有 disabled，
          仍会由原生 disabled 生效（延续既有禁用）。 */}
      <div className="flex items-center justify-between gap-4 mb-4">
        <SectionTitle className="!text-base !font-semibold">OpenClaw 管控</SectionTitle>
        <SegmentGroup data-billing-exempt>
          <SegmentOption
            className="!px-3"
            active={selectedType === BASH_POLICY}
            onClick={() => setSelectedType(BASH_POLICY)}
          >
            {`命令管控策略（${bashPolicyCount}）`}
          </SegmentOption>
          <SegmentOption
            className="!px-3"
            active={selectedType === MALICIOUS_POLICY}
            onClick={() => setSelectedType(MALICIOUS_POLICY)}
          >
            {`IP/DNS管控策略（${maliciousPolicyCount}）`}
          </SegmentOption>
        </SegmentGroup>
      </div>

      <div>
        {selectedType === BASH_POLICY ? (
          <BashPolicyList
            isFromDetail={isFromDetail}
            hasFlagship={aiAgentHostList?.some?.((d: { ProtectType: string }) => d?.ProtectType === 'Flagship')}
            getInitPolicyCount={getInitPolicyCount}
            aiAgentHostList={aiAgentHostList}
            mockData={mockBashPolicies}
          />
        ) : (
          <MaliciousPolicyList
            hasFlagship={aiAgentHostList?.some?.((d: { ProtectType: string }) => d?.ProtectType === 'Flagship')}
            getInitPolicyCount={getInitPolicyCount}
            aiAgentHostList={aiAgentHostList}
            mockData={mockMaliciousPolicies}
          />
        )}
      </div>
    </>
  );
}
