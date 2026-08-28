import React from 'react';
import PageTabs from '@/pages/admin/agentOps/PageTabs';

import LogsIndex from './Logs/index';
import SecurityGroups from './Groups/index';
import { ASSETS, CONTROL, LOGS, ALARMS, SKILLS } from './constants';
import AgentAssetsList from './Assets/AgentAssetsList';
import AlarmsList from './Alarms/AlarmsList';
import SkillsList from './Skills';

const TAB_ITEMS = [
  { id: ASSETS, label: "AI Agent资产" },
  { id: CONTROL, label: "管控配置" },
  { id: LOGS, label: "审计日志" },
  { id: SKILLS, label: "恶意Skills" },
  { id: ALARMS, label: "威胁告警" },
] as const;

export default function ContentTables({
  tabRef,
  hasTrialNum,
  getInitAlarmCount,
  activeTab,
  setActiveTab,
  machineVersionCount,
  getAllMachines,
  aiAgentHostList,
  setAiAgentHostList,
  setRiskHostCount,
  storageGroupData,
  isUltimateVersion,
  hasFilterAlarm,
  setHasFilterAlarm,
  isHideLogTalkTab,
  rencentScanTime,
  isGetAllMachinesLoading,
  setOpenTrialModalVisible,
  openExposedDetailDrawer,
  openAssetDetail,
  showTrialBtn,
  setSelectedType,
  selectedAgentIds,
  setSelectedAgentIds,
  setOpenProtectModalVisible,
  // 仅设计走查/演示用
  mockSkillsList,
  mockSkillsTags,
  mockBashAlarms,
  mockMaliciousAlarms,
  mockLogs,
  mockBashPolicies,
  mockMaliciousPolicies,
  mockBashPolicyCount,
  mockMaliciousPolicyCount,
}: any) {
  return (
    <div className="AIAgent-contentTables" ref={tabRef}>
      {/* Tab 切换（AI Agent 资产 / 管控配置 / 审计日志 / 恶意Skills / 威胁告警）
        * 停服态豁免：切换 Tab 属于查看类操作（不产生变更），
        * 与其他视图切换同档，需保持 100% 不透明与正常交互。
        * PageTabs 内部未透传 disabled，"停服前已禁用则延续禁用"约束
        * 通过 LineTabs 自身的禁用能力依然生效（此处无）。 */}
      <div data-billing-exempt>
        <PageTabs
          tabs={TAB_ITEMS}
          active={activeTab}
          onChange={(value) => {
            setActiveTab?.(value);
            setHasFilterAlarm?.(false);
          }}
        />
      </div>

      <div className="mt-4">
        {activeTab === ASSETS && (
          <AgentAssetsList
            getInitAlarmCount={getInitAlarmCount}
            getAllMachines={getAllMachines}
            aiAgentHostList={aiAgentHostList}
            setAiAgentHostList={setAiAgentHostList}
            setRiskHostCount={setRiskHostCount}
            isGetAllMachinesLoading={isGetAllMachinesLoading}
            openAssetDetail={openAssetDetail}
            storageGroupData={storageGroupData}
            isUltimateVersion={isUltimateVersion}
            hasFilterAlarm={hasFilterAlarm}
            setHasFilterAlarm={setHasFilterAlarm}
            openExposedDetailDrawer={openExposedDetailDrawer}
            rencentScanTime={rencentScanTime}
            setOpenTrialModalVisible={setOpenTrialModalVisible}
            hasTrialNum={hasTrialNum}
            showTrialBtn={showTrialBtn}
            setSelectedType={setSelectedType}
            selectedAgentIds={selectedAgentIds}
            setSelectedAgentIds={setSelectedAgentIds}
            setOpenProtectModalVisible={setOpenProtectModalVisible}
          />
        )}
        {activeTab === CONTROL && (
          <SecurityGroups
            aiAgentHostList={aiAgentHostList}
            isGetAllMachinesLoading={isGetAllMachinesLoading}
            storageGroupData={storageGroupData}
            mockBashPolicies={mockBashPolicies}
            mockMaliciousPolicies={mockMaliciousPolicies}
            mockBashPolicyCount={mockBashPolicyCount}
            mockMaliciousPolicyCount={mockMaliciousPolicyCount}
          />
        )}
        {activeTab === LOGS && (
          <LogsIndex
            aiAgentHostList={aiAgentHostList}
            isGetAllMachinesLoading={isGetAllMachinesLoading}
            openAssetDetail={openAssetDetail}
            isHideLogTalkTab={isHideLogTalkTab}
            mockLogs={mockLogs}
          />
        )}
        {activeTab === SKILLS && (
          <SkillsList
            aiAgentHostList={aiAgentHostList}
            isGetAllMachinesLoading={isGetAllMachinesLoading}
            getAllMachines={getAllMachines}
            openAssetDetail={openAssetDetail}
            rencentScanTime={rencentScanTime}
            mockData={mockSkillsList}
            mockTags={mockSkillsTags}
          />
        )}
        {activeTab === ALARMS && (
          <AlarmsList
            machineVersionCount={machineVersionCount}
            aiAgentHostList={aiAgentHostList}
            openAssetDetail={openAssetDetail}
            mockBashAlarms={mockBashAlarms}
            mockMaliciousAlarms={mockMaliciousAlarms}
          />
        )}
      </div>
    </div>
  );
}
