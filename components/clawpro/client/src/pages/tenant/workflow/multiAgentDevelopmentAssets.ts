import intakeRouterContent from "./assets/multi-agents-devflow/00-intake-router.md?raw";
import requirementAnalysisContent from "./assets/multi-agents-devflow/01-requirement-analysis.md?raw";
import architectureDesignContent from "./assets/multi-agents-devflow/02-architecture-design.md?raw";
import codeDevelopmentContent from "./assets/multi-agents-devflow/03-code-development.md?raw";
import codeReviewContent from "./assets/multi-agents-devflow/04-code-review.md?raw";
import e2eTestContent from "./assets/multi-agents-devflow/05-e2e-test.md?raw";
import knowledgeDistillationContent from "./assets/multi-agents-devflow/06-knowledge-distillation.md?raw";
import finalSummaryContent from "./assets/multi-agents-devflow/07-final-summary.md?raw";
import soloSmallChangeContent from "./assets/multi-agents-devflow/08-solo-small-change.md?raw";
import nodeHandoffContractContent from "./assets/multi-agents-devflow/contracts/node-handoff.schema.json?raw";
import workflowStateContractContent from "./assets/multi-agents-devflow/contracts/workflow-state.template.json?raw";

export interface WorkflowNodeConfigAsset {
  id: string;
  name: string;
  version: string;
  type: "rules" | "skill" | "contract";
  summary: string;
  source: string;
  content: string;
}

const SOURCE_REVISION =
  "dns-ai/multi-agents-devflow@8c2038836257314b0ece7b5ae484c41acb8ab9d5/.codebuddy";
export const MULTI_AGENT_DEVELOPMENT_ASSET_VERSION = "1.4";

function createAsset(
  id: string,
  name: string,
  summary: string,
  content: string,
  type: WorkflowNodeConfigAsset["type"] = "rules"
): WorkflowNodeConfigAsset {
  return {
    id,
    name,
    version: MULTI_AGENT_DEVELOPMENT_ASSET_VERSION,
    type,
    summary,
    source: SOURCE_REVISION,
    content,
  };
}

const assets = {
  nodeHandoffContract: createAsset(
    "node-handoff-contract",
    "节点输入输出与交接契约",
    "统一约束节点状态、结论、产物清单、决策、重试和下一节点字段。",
    nodeHandoffContractContent,
    "contract"
  ),
  workflowStateContract: createAsset(
    "workflow-state-contract",
    "工作流状态契约",
    "统一约束任务标识、工作区、分级证据、节点状态、决策和最终汇总。",
    workflowStateContractContent,
    "contract"
  ),
  intakeRouter: createAsset(
    "intake-router",
    "需求接入与分流",
    "定位真实工作区，初始化状态并按改动规模路由。",
    intakeRouterContent
  ),
  requirementAnalysis: createAsset(
    "requirement-analysis",
    "需求分析与澄清",
    "把原始需求和真实代码现状收敛为可设计、可验收的需求报告。",
    requirementAnalysisContent
  ),
  architectureDesign: createAsset(
    "architecture-design",
    "技术方案与执行计划",
    "基于真实代码生成方案，并拆成文件边界互斥的可执行任务。",
    architectureDesignContent
  ),
  codeDevelopment: createAsset(
    "code-development",
    "并行代码开发",
    "按执行计划和文件白名单完成真实代码修改与自检。",
    codeDevelopmentContent
  ),
  codeReview: createAsset(
    "code-review",
    "代码审查",
    "独立审查真实 diff，并给出可机器判定的 PASSED/FAILED。",
    codeReviewContent
  ),
  e2eTest: createAsset(
    "e2e-test",
    "E2E 测试",
    "基于真实代码和审查结论补齐并运行端到端验证。",
    e2eTestContent
  ),
  knowledgeDistillation: createAsset(
    "knowledge-distillation",
    "知识沉淀",
    "从真实上游产物提炼可跨任务复用的工程知识。",
    knowledgeDistillationContent
  ),
  finalSummary: createAsset(
    "final-summary",
    "最终汇总",
    "读取真实状态和产物，生成可验收、可交接的最终摘要。",
    finalSummaryContent
  ),
  soloSmallChange: createAsset(
    "solo-small-change",
    "小需求独立开发",
    "small 需求的轻量串行开发分支，超出边界立即升级。",
    soloSmallChangeContent
  ),
};

const sharedContracts = [
  assets.nodeHandoffContract,
  assets.workflowStateContract,
];

export const MULTI_AGENT_DEVELOPMENT_NODE_ASSETS: Record<
  string,
  WorkflowNodeConfigAsset[]
> = {
  "PHASE-0": [assets.intakeRouter, ...sharedContracts],
  SOLO: [assets.soloSmallChange, ...sharedContracts],
  "TASK-01": [assets.requirementAnalysis, ...sharedContracts],
  "TASK-02": [assets.architectureDesign, ...sharedContracts],
  "TASK-03": [assets.codeDevelopment, ...sharedContracts],
  "CODE-REVIEW": [assets.codeReview, ...sharedContracts],
  "TASK-04": [assets.e2eTest, ...sharedContracts],
  "TASK-05": [assets.knowledgeDistillation, ...sharedContracts],
  SUMMARY: [assets.finalSummary, ...sharedContracts],
};

export function cloneMultiAgentNodeAssets(
  nodeId: string
): WorkflowNodeConfigAsset[] {
  return (MULTI_AGENT_DEVELOPMENT_NODE_ASSETS[nodeId] ?? []).map(asset => ({
    ...asset,
  }));
}
