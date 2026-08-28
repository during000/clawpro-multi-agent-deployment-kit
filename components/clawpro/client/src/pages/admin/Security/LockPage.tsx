import React, { useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import SyncAssetBtn from "./AIAgent/Assets/SyncAssetBtn";

export default function LockPage({
  aiAgentHostList,
  getAllMachines,
  isTrialLoading,
  rencentScanTime,
  isGetAllMachinesLoading,
  setOpenTrialModalVisible,
}: any) {
  return (
    <div className="csip-AIAgent-lockPage" style={{ minHeight: "100vh" }}>
      <h2 style={{ fontSize: 36, fontWeight: 500 }}>
        AI Agent安全
      </h2>
      <div
        style={{
          margin: "15px 0 30px 0",
          fontSize: 16,
          color: "rgba(0,0,0,0.7)",
        }}
      >
        自动识别环境中运行 AI Agent
        的资产，围绕资产提供"风险检测、审计溯源、管控配置、Skills扫描能力"，帮助企业更安全地引入和治理
        AI Agent。
      </div>
      <div>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button
                style={{ marginRight: 10 }}
                disabled={
                  isTrialLoading ||
                  isGetAllMachinesLoading ||
                  !aiAgentHostList?.length
                }
                onClick={() => setOpenTrialModalVisible?.(true)}
              >
                {(isTrialLoading || isGetAllMachinesLoading) && (
                  <Loader2 className="animate-spin mr-1 h-4 w-4" />
                )}
                <span>申请试用</span>
              </Button>
            </span>
          </TooltipTrigger>
          {!aiAgentHostList?.length ? (
            <TooltipContent>
              当前暂无AI Agent资产，请点击同步资产，AI Agent资产同步后可申请试用
            </TooltipContent>
          ) : null}
        </Tooltip>
        {!aiAgentHostList?.length && !isGetAllMachinesLoading ? (
          <SyncAssetBtn
            refreshTable={getAllMachines}
            refreshFromLock={getAllMachines}
            rencentScanTime={rencentScanTime}
          />
        ) : null}
        <a
          href="https://cloud.tencent.com/document/product/664/129679"
          target="_blank"
          style={{ margin: "1px 0 0 10px", color: "#1447e6", fontSize: 14 }}
        >
          了解更多
        </a>
      </div>
      <h3 style={{ fontSize: 16, margin: "60px 0 20px", fontWeight: 500 }}>
        功能特性说明
      </h3>
      <div className="flex gap-4">
        <div className="flex-1">
          <div className="item-wrap">
            <div className="auto" />
            <div className="header">自动识别 AI Agent 资产</div>
            <div className="tips">
              基于OpenClaw特征与网络行为，自动识别环境中AI
              Agent资产，帮助您快速掌握 Agent 资产分布。
            </div>
          </div>
        </div>
        <div className="flex-1">
          <div className="item-wrap">
            <div className="control" />
            <div className="header">多层管控，降低安全暴露面</div>
            <div className="tips">
              支持OpenClaw、网络、身份层的安全管控，可通过内网安全组规则等方式，帮助您降低
              Agent 资产的潜在风险。
            </div>
          </div>
        </div>
        <div className="flex-1">
          <div className="item-wrap">
            <div className="logs" />
            <div className="header">全面审计，支持行为追溯</div>
            <div className="tips">
              提供围绕 Agent
              资产的审计日志与行为记录，支持从资产、事件、行为等维度，帮助追踪影响范围。
            </div>
          </div>
        </div>
        <div className="flex-1">
          <div className="item-wrap">
            <div className="skills" />
            <div className="header">Skills 扫描，发现潜在风险</div>
            <div className="tips">
              可对 Skills 进行扫描，帮助您识别 Agent
              使用过程中的潜在安全风险，降低高危技能带来的暴露面。
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
