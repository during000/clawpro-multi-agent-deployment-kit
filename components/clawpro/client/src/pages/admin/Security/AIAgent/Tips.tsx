import React from "react";

export default function AIAgentTips({ showTipsPanel }: any) {
  if (!showTipsPanel) return null;

  return (
    <div className="rounded-[4px] border border-[#E5E5E5] bg-white p-4">
      <div className="text-sm text-[#525252]">
        帮助快速识别环境中运行 AI Agent /
        调用大模型的资产，将这些资产的风险告警、管控策略生效情况与审计记录集中呈现，让你在"可见—可控—可追溯"的闭环下，安全引入并持续使用
        Agent。
        <a
          href="https://cloud.tencent.com/document/product/664/129679"
          target="_blank"
          className="ml-1.5 text-sm text-[#1447E6] hover:underline"
        >
          说明文档
        </a>
      </div>
      <div className="max-w-[80%] p-3 px-4 bg-gray-50 rounded-[4px] mt-3">
        <ul className="list-disc list-inside space-y-2 text-sm text-[#525252]">
          <li>
            <strong>资产可见：</strong>
            自动识别 AI Agent 资产（运行 AI Agent
            或通过网络请求调用大模型的资产），生成统一资产清单，支持按 Agent
            类型/业务/资产组快速管理。
          </li>
          <li>
            <strong>风险可控：</strong>
            围绕 Agent
            资产聚合网络、OpenClaw层关键告警，支持按威胁等级/时间/资产/来源归因筛选与排序，快速锁定"最需优先处置"的风险点。
          </li>
        </ul>
      </div>
    </div>
  );
}
