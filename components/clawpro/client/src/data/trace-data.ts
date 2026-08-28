/**
 * trace-data.ts — 真实数据模块
 * 数据来源：
 *  - CLS 北京 Trace 主题 (region=ap-beijing, topic_id=9109c196-e0a2-45d6-b559-d3e78f46c619)
 *  - cls_real_data.js (完整的 CLS session/trace/sessionTraces 数据)
 *  - langfuse_real_data.js (Langfuse 数据)
 *  - TraceWaterfall.tsx MOCK_RAW_SPANS (CLS Trace 主题真实 span 数据)
 */

// ═══ 类型定义 ═══

export interface TraceData {
  id: string;
  name: string;
  timestamp: string;
  input: string;
  output: string;
  observations: number;
  latency: number;
  latencyStr: string;
  tags: string[];
  tokens: number;
  level: "DEFAULT" | "ERROR" | "WARNING";
  sessionId: string;
  metadata: {
    resourceAttributes?: Record<string, string>;
    scope?: { name?: string; version?: string; attributes?: Record<string, string> };
    [key: string]: unknown;
  };
  // 兼容旧字段
  traceId?: string;
}

export interface SessionData {
  id: string;
  createdAt: string;
  durationStr: string;
  durationSec: number;
  traces: number;
  cost: number;
  tags: string[];
  totalTokens: number;
  status: "normal" | "error" | "stuck";
  openClawName: string;
}

export interface SpanData {
  id: string;
  name: string;
  type: "TRACE" | "SPAN" | "GENERATION" | "RETRIEVER";
  latency: number;
  start: number;
  end: number;
  inTok: number;
  outTok: number;
  children: string[];
  level: number;
  statusLevel: "DEFAULT" | "ERROR" | "WARNING";
}

// ═══ CLS Trace 主题真实 Span 数据（来自 TraceWaterfall.tsx MOCK_RAW_SPANS）═══
// trace.id = 0f46b67fdc798cde1a4177225ffe, cron-新闻热点
// 9 条 span: AGENT → 3 个 STEP → 各含 LLM + TOOL

const CLS_TRACE_SPANS_0f46: SpanData[] = [
  { id: "2de476a7e3f1b9c2", name: "invoke_agent main", type: "TRACE", latency: 49.88, start: 0, end: 100, inTok: 71395, outTok: 130, children: ["5469f8c9a2b1e8f7", "b648a099d1c7e5f4", "40b19a56e3d2c8f1"], level: 0, statusLevel: "DEFAULT" },
  { id: "5469f8c9a2b1e8f7", name: "react step (round 1)", type: "SPAN", latency: 17.78, start: 0, end: 35.64, inTok: 0, outTok: 0, children: ["bdc2f15e4a7c3d91", "e2f437e8b5c9d102"], level: 1, statusLevel: "DEFAULT" },
  { id: "bdc2f15e4a7c3d91", name: "chat tc-code-latest", type: "GENERATION", latency: 0.028, start: 0.006, end: 0.062, inTok: 18777, outTok: 54, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "e2f437e8b5c9d102", name: "execute_tool browser", type: "SPAN", latency: 13.08, start: 0.066, end: 26.28, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "ERROR" },
  { id: "b648a099d1c7e5f4", name: "react step (round 2)", type: "SPAN", latency: 5.99, start: 35.65, end: 47.65, inTok: 0, outTok: 0, children: ["c2357419f8a2b6e3", "95463dc0a7b2c8d1"], level: 1, statusLevel: "DEFAULT" },
  { id: "c2357419f8a2b6e3", name: "chat tc-code-latest", type: "GENERATION", latency: 0.179, start: 35.66, end: 36.02, inTok: 21113, outTok: 68, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "95463dc0a7b2c8d1", name: "execute_tool browser", type: "SPAN", latency: 5.71, start: 36.03, end: 47.48, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "40b19a56e3d2c8f1", name: "react step (round 3)", type: "SPAN", latency: 0.38, start: 47.49, end: 48.25, inTok: 0, outTok: 0, children: ["c8cc3aed52f1b7a9"], level: 1, statusLevel: "DEFAULT" },
  { id: "c8cc3aed52f1b7a9", name: "chat tc-code-latest", type: "GENERATION", latency: 0.38, start: 47.50, end: 48.26, inTok: 23037, outTok: 556, children: [], level: 2, statusLevel: "DEFAULT" },
];

// Langfuse pipeline trace span 模板（来自原型 spansData）
const LANGFUSE_PIPELINE_SPANS: SpanData[] = [
  { id: "root", name: "pipeline-full", type: "TRACE", latency: 27.45, start: 0, end: 100, inTok: 276, outTok: 699, children: ["chain"], level: 0, statusLevel: "DEFAULT" },
  { id: "chain", name: "pipeline-full", type: "SPAN", latency: 27.45, start: 0, end: 100, inTok: 276, outTok: 699, children: ["gen1", "ret", "rerank", "gen2", "post"], level: 1, statusLevel: "DEFAULT" },
  { id: "gen1", name: "query-rewrite", type: "GENERATION", latency: 1.62, start: 0, end: 6, inTok: 34, outTok: 10, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "ret", name: "retrieval", type: "RETRIEVER", latency: 0.005, start: 6, end: 6.5, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "rerank", name: "reranking", type: "SPAN", latency: 0.06, start: 7, end: 7.5, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "gen2", name: "llm-generation", type: "GENERATION", latency: 25.77, start: 7, end: 100, inTok: 242, outTok: 689, children: [], level: 2, statusLevel: "DEFAULT" },
  { id: "post", name: "post-processing", type: "SPAN", latency: 0.005, start: 99.5, end: 100, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
];

// ═══ Trace ID → Span 映射 ═══

const traceSpansMap: Record<string, SpanData[]> = {
  // ─── CLS Traces（北京 Trace 主题真实数据）───
  // e4ff73dcdc2c1e8d1c85256cdb47622d — 腾讯新闻热点 93.5s 成功（有browser ERROR再恢复）
  "e4ff73dcdc2c1e8d1c85256cdb47622d": CLS_TRACE_SPANS_0f46,
  // a492358fd67d6726e4072e600c19a6f1 — 腾讯新闻热点 54.5s 成功
  "a492358fd67d6726e4072e600c19a6f1": CLS_TRACE_SPANS_0f46.map(s => ({
    ...s,
    statusLevel: s.statusLevel === "ERROR" ? "DEFAULT" as const : s.statusLevel, // 这条无错误
    latency: s.id === "2de476a7e3f1b9c2" ? 54.55 : s.latency,
  })),
  // db319ec0ee9269774e846ff8198e57fe — 腾讯新闻热点 119.8s（多次工具失败）
  "db319ec0ee9269774e846ff8198e57fe": CLS_TRACE_SPANS_0f46.map(s => ({
    ...s,
    latency: s.id === "2de476a7e3f1b9c2" ? 119.83 : s.latency,
  })),
  // 84c04f8eb6cbe048ace317c1555c9dec — 腾讯新闻热点 94.1s（空 output）
  "84c04f8eb6cbe048ace317c1555c9dec": CLS_TRACE_SPANS_0f46.map(s => ({
    ...s,
    latency: s.id === "2de476a7e3f1b9c2" ? 94.05 : s.latency,
  })),
  // 38e6203d3b12f456863805c8d018e100 — 97.7s
  "38e6203d3b12f456863805c8d018e100": CLS_TRACE_SPANS_0f46.map(s => ({
    ...s,
    latency: s.id === "2de476a7e3f1b9c2" ? 97.66 : s.latency,
  })),
  // 87f13f28e5815e57f8a4f931b7fb28d9 — 101.2s
  "87f13f28e5815e57f8a4f931b7fb28d9": CLS_TRACE_SPANS_0f46.map(s => ({
    ...s,
    latency: s.id === "2de476a7e3f1b9c2" ? 101.19 : s.latency,
  })),
  // 短对话（只有 AGENT → LLM，无 TOOL 调用）
  "77b67546bc4c99239889093094108963": [
    { id: "root", name: "invoke_agent main", type: "TRACE", latency: 14.25, start: 0, end: 100, inTok: 26027, outTok: 0, children: ["step1"], level: 0, statusLevel: "DEFAULT" },
    { id: "step1", name: "react step", type: "SPAN", latency: 14.25, start: 0, end: 100, inTok: 0, outTok: 0, children: ["llm1"], level: 1, statusLevel: "DEFAULT" },
    { id: "llm1", name: "chat tc-code-latest", type: "GENERATION", latency: 14.25, start: 0, end: 100, inTok: 26027, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
  ],
  // ─── Langfuse Traces ───
  "8229efbc3f84f97855fa1c1ffc702d48": LANGFUSE_PIPELINE_SPANS,
  "8b3f999f1a95f7b7c04f2c35c3f54ac5": LANGFUSE_PIPELINE_SPANS.map(s => ({
    ...s,
    latency: s.id === "root" || s.id === "chain" ? 22.88 : s.id === "gen2" ? 21.21 : s.latency,
  })),
  "a382be4e9838bc7254f23944869f806c": LANGFUSE_PIPELINE_SPANS.map(s => ({
    ...s,
    latency: s.id === "root" || s.id === "chain" ? 1.67 : s.id === "gen1" ? 1.5 : s.id === "gen2" ? 0.12 : s.latency,
    inTok: Math.round(s.inTok * 0.5), outTok: Math.round(s.outTok * 0.2),
  })),
  "a7da65265270a3a95c6e112d451e964b": LANGFUSE_PIPELINE_SPANS.map(s => ({
    ...s,
    latency: s.id === "root" || s.id === "chain" ? 1.39 : s.id === "gen1" ? 1.22 : s.id === "gen2" ? 0.1 : s.latency,
    inTok: Math.round(s.inTok * 0.4), outTok: Math.round(s.outTok * 0.18),
  })),
  "baa577dc8640d82be6aab6e27082b669": [
    { id: "root", name: "rag-normal", type: "TRACE", latency: 3.81, start: 0, end: 100, inTok: 176, outTok: 89, children: ["chain"], level: 0, statusLevel: "DEFAULT" },
    { id: "chain", name: "rag-normal", type: "SPAN", latency: 3.81, start: 0, end: 100, inTok: 176, outTok: 89, children: ["ret", "gen1"], level: 1, statusLevel: "DEFAULT" },
    { id: "ret", name: "retrieval", type: "RETRIEVER", latency: 0.12, start: 0, end: 3.1, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
    { id: "gen1", name: "llm-generation", type: "GENERATION", latency: 3.69, start: 3.1, end: 100, inTok: 176, outTok: 89, children: [], level: 2, statusLevel: "DEFAULT" },
  ],
  "b0c545184cf29555be12ddbe6ad55a9b": [
    { id: "root", name: "rag-normal", type: "TRACE", latency: 3.88, start: 0, end: 100, inTok: 180, outTok: 89, children: ["chain"], level: 0, statusLevel: "DEFAULT" },
    { id: "chain", name: "rag-normal", type: "SPAN", latency: 3.88, start: 0, end: 100, inTok: 180, outTok: 89, children: ["ret", "gen1"], level: 1, statusLevel: "DEFAULT" },
    { id: "ret", name: "retrieval", type: "RETRIEVER", latency: 0.12, start: 0, end: 3.1, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
    { id: "gen1", name: "llm-generation", type: "GENERATION", latency: 3.76, start: 3.1, end: 100, inTok: 180, outTok: 89, children: [], level: 2, statusLevel: "DEFAULT" },
  ],
  "a134190885b4bbf7cebdca7b6d6c5c44": [
    { id: "root", name: "rag-qa", type: "TRACE", latency: 2.61, start: 0, end: 100, inTok: 136, outTok: 69, children: ["chain"], level: 0, statusLevel: "DEFAULT" },
    { id: "chain", name: "rag-qa", type: "SPAN", latency: 2.61, start: 0, end: 100, inTok: 136, outTok: 69, children: ["ret", "gen1"], level: 1, statusLevel: "DEFAULT" },
    { id: "ret", name: "retrieval", type: "RETRIEVER", latency: 0.12, start: 0, end: 4.6, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
    { id: "gen1", name: "llm-generation", type: "GENERATION", latency: 2.49, start: 4.6, end: 100, inTok: 136, outTok: 69, children: [], level: 2, statusLevel: "DEFAULT" },
  ],
  // Error traces
  "e694bc05e0f1e6d8d4f4b03aacc25193": [
    { id: "root", name: "agent-tool-failure", type: "TRACE", latency: 92.25, start: 0, end: 100, inTok: 1840, outTok: 618, children: ["chain"], level: 0, statusLevel: "ERROR" },
    { id: "chain", name: "agent-tool-failure", type: "SPAN", latency: 92.25, start: 0, end: 100, inTok: 1840, outTok: 618, children: ["gen1", "tool_err", "gen2", "gen3"], level: 1, statusLevel: "DEFAULT" },
    { id: "gen1", name: "planning", type: "GENERATION", latency: 2.31, start: 0, end: 2.5, inTok: 320, outTok: 150, children: [], level: 2, statusLevel: "DEFAULT" },
    { id: "tool_err", name: "tool-calculate", type: "SPAN", latency: 45.12, start: 2.5, end: 51.4, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "ERROR" },
    { id: "gen2", name: "error-recovery", type: "GENERATION", latency: 3.88, start: 51.4, end: 55.6, inTok: 760, outTok: 234, children: [], level: 2, statusLevel: "DEFAULT" },
    { id: "gen3", name: "final-response", type: "GENERATION", latency: 40.95, start: 55.6, end: 100, inTok: 760, outTok: 234, children: [], level: 2, statusLevel: "DEFAULT" },
  ],
  "6f901207ccb3d6bd50d98a53bbe05d77": [
    { id: "root", name: "rag-llm-error", type: "TRACE", latency: 0, start: 0, end: 100, inTok: 0, outTok: 0, children: ["chain"], level: 0, statusLevel: "ERROR" },
    { id: "chain", name: "rag-llm-error", type: "SPAN", latency: 0, start: 0, end: 100, inTok: 0, outTok: 0, children: ["ret", "llm_err"], level: 1, statusLevel: "DEFAULT" },
    { id: "ret", name: "retrieval", type: "RETRIEVER", latency: 0.003, start: 0, end: 5, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "DEFAULT" },
    { id: "llm_err", name: "llm-generation", type: "GENERATION", latency: 0, start: 5, end: 5.1, inTok: 0, outTok: 0, children: [], level: 2, statusLevel: "ERROR" },
  ],
};

/** 获取指定 Trace 的 Span 数据 */
export function getSpansForTrace(traceId: string): SpanData[] {
  return traceSpansMap[traceId] || CLS_TRACE_SPANS_0f46;
}

/** 获取指定 Trace 的 DAG 节点 */
export function getDagNodesForTrace(traceId: string): string[] {
  const spans = getSpansForTrace(traceId);
  const leafNames = spans.filter(s => s.level >= 2).map(s => s.name);
  return ["__start__", ...leafNames, "__end__"];
}

// 默认 span（fallback），用 CLS 真实数据
export const spansData: SpanData[] = CLS_TRACE_SPANS_0f46;

// ═══ Sessions 数据（完全来自 cls_real_data.js）═══

export const sessionsData: SessionData[] = [
  { id: "0d3cc2f8-7975-48fd-b2e1-26697326f832", createdAt: "2026-04-12 04:32:02", durationStr: "39s", durationSec: 40, traces: 22, cost: 0, tags: ["main"], totalTokens: 397410, status: "normal", openClawName: "cron-新闻热点" },
  { id: "cb1986ed-d868-4478-94af-c43d35fd27e1", createdAt: "2026-04-12 11:30:00", durationStr: "1m 34s", durationSec: 94, traces: 1, cost: 0, tags: ["main"], totalTokens: 37654, status: "normal", openClawName: "cron-新闻热点" },
  { id: "1e297787-c415-487c-b517-9048ef1e0611", createdAt: "2026-04-12 11:00:00", durationStr: "1m 59s", durationSec: 120, traces: 1, cost: 0, tags: ["main"], totalTokens: 206340, status: "stuck", openClawName: "cron-新闻热点" },
  { id: "c9cf70cc-57c4-409d-a712-a91681b94796", createdAt: "2026-04-12 10:31:39", durationStr: "1m 33s", durationSec: 94, traces: 1, cost: 0, tags: ["main"], totalTokens: 460186, status: "error", openClawName: "cron-新闻热点" },
  { id: "670ad354-c740-4ea7-ad4a-43a7f549bfd3", createdAt: "2026-04-12 09:30:00", durationStr: "1m 37s", durationSec: 98, traces: 1, cost: 0, tags: ["main"], totalTokens: 119224, status: "normal", openClawName: "cron-新闻热点" },
  { id: "dde18c43-8772-4652-96df-66131da659c3", createdAt: "2026-04-12 06:30:00", durationStr: "54s", durationSec: 55, traces: 1, cost: 0, tags: ["main"], totalTokens: 288684, status: "normal", openClawName: "cron-新闻热点" },
  { id: "3cd9e804-4212-419a-8420-c3c5c92eadc0", createdAt: "2026-04-12 07:00:00", durationStr: "1m 41s", durationSec: 101, traces: 1, cost: 0, tags: ["main"], totalTokens: 78206, status: "normal", openClawName: "cron-微博热搜" },
  // Langfuse sessions
  { id: "session-pipeline-001", createdAt: "2026-04-11 16:06:35", durationStr: "53s", durationSec: 53, traces: 4, cost: 0.0032, tags: ["full-chain", "pipeline"], totalTokens: 1950, status: "normal", openClawName: "feishu-客服助手" },
  { id: "session-error-002", createdAt: "2026-04-11 16:06:32", durationStr: "1m 32s", durationSec: 92, traces: 2, cost: 0.0008, tags: ["error", "tool-failure"], totalTokens: 520, status: "error", openClawName: "qq-日报生成" },
  { id: "session-prod-001", createdAt: "2026-04-11 16:06:26", durationStr: "7s", durationSec: 8, traces: 2, cost: 0.0018, tags: ["normal", "rag"], totalTokens: 1240, status: "normal", openClawName: "feishu-客服助手" },
  { id: "test-session-001", createdAt: "2026-04-11 15:26:50", durationStr: "33s", durationSec: 34, traces: 1, cost: 0.0048, tags: ["test", "rag"], totalTokens: 3200, status: "normal", openClawName: "qq-日报生成" },
  { id: "session-error-001", createdAt: "2026-04-11 16:06:26", durationStr: "0s", durationSec: 0, traces: 1, cost: 0, tags: ["error", "rag"], totalTokens: 0, status: "error", openClawName: "feishu-客服助手" },
];

// ═══ Traces 数据（CLS 真实 + Langfuse 真实）═══

export const tracesData: TraceData[] = [
  // ─── CLS Traces（完全来自 cls_real_data.js clsTracesData）───
  { id: "e4ff73dcdc2c1e8d1c85256cdb47622d", name: "invoke_agent main", timestamp: "2026-04-12 10:31:39", input: "[cron:b633122b 腾讯新闻热点查询] 请使用 browser 工具访问 https://news.qq.com/ ...", output: "让我重新尝试启动浏览器：\n太好了！成功获取到腾讯新闻热点数据 \u{1F4F0}\n\n\u{1F525} **腾讯新闻热点简报** \u{1F525}\n*2026年4月12日 周日 18:31*\n\n**1. \u{1F3D3} 跨越55载，再续\"乒乓情缘\"**\n**2. \u{1F42F} 打虎！丁业现被查**\n**3. \u{1F697} 中年人疯抢大车**\n**4. \u{1F4CA} 四组数据看国家高新区发展成色**", observations: 24, latency: 93.53, latencyStr: "93.5s", tags: ["step", "llm", "tool", "agent"], tokens: 460186, level: "ERROR", sessionId: "c9cf70cc-57c4-409d-a712-a91681b94796", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", "agent.tool_call_count": "7", "agent.message_count": "16", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  { id: "a492358fd67d6726e4072e600c19a6f1", name: "invoke_agent main", timestamp: "2026-04-12 06:30:00", input: "[cron:b633122b 腾讯新闻热点查询] 请使用 browser 工具访问 https://news.qq.com/ ...", output: "\u{1F4F0} **今日新闻热点简报** | 4月12日 周日\n\n**\u{1F525} 国际要闻**\n\u{1F4CC} **美伊谈判破裂** \u2014 伊朗表示因\"美方贪婪和野心\"未达成协议\n\u{1F4CC} **中东局势** \u2014 以总理宣称已\"摧毁\"伊朗核计划\n\n**\u{1F4BC} 财经科技**\n\u{1F4CC} **创业板改革** \u2014 增设第四套上市标准\n\u{1F4CC} **DeepSeek V4将发** \u2014 预计4月下旬发布", observations: 21, latency: 54.55, latencyStr: "54.5s", tags: ["step", "llm", "tool", "agent"], tokens: 288684, level: "DEFAULT", sessionId: "dde18c43-8772-4652-96df-66131da659c3", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  { id: "db319ec0ee9269774e846ff8198e57fe", name: "invoke_agent main", timestamp: "2026-04-12 11:00:00", input: "[cron:b633122b 腾讯新闻热点查询] 请使用 browser 工具访问 https://news.qq.com/ ...", output: "现在访问腾讯新闻网站获取热点新闻...\nbrowser 工具超时了，让我尝试用 web_fetch 直接获取页面内容：\n页面是动态加载的，web_fetch 无法获取完整内容。让我搜索腾讯新闻热点来获取信息：\nweb_search 也失败了。让我尝试访问新浪新闻作为替代：", observations: 18, latency: 119.83, latencyStr: "119.8s", tags: ["step", "llm", "tool", "agent"], tokens: 206340, level: "DEFAULT", sessionId: "1e297787-c415-487c-b517-9048ef1e0611", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  { id: "84c04f8eb6cbe048ace317c1555c9dec", name: "invoke_agent main", timestamp: "2026-04-12 11:30:00", input: "[cron:b633122b 腾讯新闻热点查询] 请使用 browser 工具访问 https://news.qq.com/ ...", output: "", observations: 5, latency: 94.05, latencyStr: "94.1s", tags: ["step", "llm", "tool", "agent"], tokens: 37654, level: "DEFAULT", sessionId: "cb1986ed-d868-4478-94af-c43d35fd27e1", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  { id: "38e6203d3b12f456863805c8d018e100", name: "invoke_agent main", timestamp: "2026-04-12 09:30:00", input: "[cron:b633122b 腾讯新闻热点查询] 请使用 browser 工具访问 https://news.qq.com/ ...", output: "", observations: 11, latency: 97.66, latencyStr: "97.7s", tags: ["step", "llm", "tool", "agent"], tokens: 119224, level: "DEFAULT", sessionId: "670ad354-c740-4ea7-ad4a-43a7f549bfd3", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  { id: "87f13f28e5815e57f8a4f931b7fb28d9", name: "invoke_agent main", timestamp: "2026-04-12 07:00:00", input: "[cron:b633122b 腾讯新闻热点查询] 请使用 browser 工具访问 https://news.qq.com/ ...", output: "浏览器工具当前不可用。让我尝试使用 web_fetch 获取腾讯新闻内容：", observations: 9, latency: 101.19, latencyStr: "101.2s", tags: ["step", "llm", "tool", "agent"], tokens: 78206, level: "DEFAULT", sessionId: "3cd9e804-4212-419a-8420-c3c5c92eadc0", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  { id: "77b67546bc4c99239889093094108963", name: "invoke_agent main", timestamp: "2026-04-12 11:32:01", input: "System: [2026-04-12 19:31:36 GMT+8] \u26A0\uFE0F \u{1F310} Browser failed\n\nRead HEARTBEAT.md if it exists ...", output: "你好，我无法给到相关内容。", observations: 3, latency: 14.25, latencyStr: "14.3s", tags: ["step", "llm", "agent"], tokens: 26027, level: "DEFAULT", sessionId: "0d3cc2f8-7975-48fd-b2e1-26697326f832", metadata: { "service.name": "openclaw-agent", "openclaw.version": "2026.4.2", "agent.name": "main", "agent.tool_call_count": "28", "agent.message_count": "456", host: "VM-0-7-ubuntu", region: "ap-beijing", resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "openclaw-agent" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6" } } } },
  // ─── Langfuse Traces（来自 langfuse_real_data.js tracesData）───
  { id: "8229efbc3f84f97855fa1c1ffc702d48", name: "pipeline-full", timestamp: "2026-04-11 16:12:31", input: "如何选择适合自己业务的 Embedding 模型？", output: "选择适合自己业务的 Embedding 模型，需要综合考虑以下几个关键因素：\n\n## 一、核心考量维度\n\n### 1. **任务类型匹配**\n- **语义搜索**：选择擅长理解语义相似度的模型\n- **文本分类/聚类**：需要模型能准确捕捉类别特征\n- **问答系统**：要求模型理解问题与答案的关联性\n- **多语言场景**：需要支持多语言的嵌入模型", observations: 6, latency: 27.45, latencyStr: "27.45s", tags: ["full-chain", "pipeline"], tokens: 803, level: "DEFAULT", sessionId: "session-pipeline-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "8b3f999f1a95f7b7c04f2c35c3f54ac5", name: "pipeline-full", timestamp: "2026-04-11 16:12:08", input: "RAG 系统在生产环境中有哪些常见的性能瓶颈？", output: "在 RAG 系统生产环境中，常见的性能瓶颈主要集中在以下几个方面：\n\n### 1. **检索阶段瓶颈**\n- **向量数据库查询延迟**：随着索引规模增大，相似性搜索可能变慢\n- **检索质量与速度的权衡**：提高召回率往往需要扩大搜索范围", observations: 6, latency: 22.88, latencyStr: "22.88s", tags: ["full-chain", "pipeline"], tokens: 684, level: "DEFAULT", sessionId: "session-pipeline-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "a382be4e9838bc7254f23944869f806c", name: "pipeline-full", timestamp: "2026-04-11 16:06:33", input: "如何选择适合自己业务的 Embedding 模型？", output: "None", observations: 2, latency: 1.67, latencyStr: "1.67s", tags: ["full-chain", "pipeline"], tokens: 133, level: "DEFAULT", sessionId: "session-pipeline-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "a7da65265270a3a95c6e112d451e964b", name: "pipeline-full", timestamp: "2026-04-11 16:06:31", input: "RAG 系统在生产环境中有哪些常见的性能瓶颈？", output: "None", observations: 2, latency: 1.39, latencyStr: "1.39s", tags: ["full-chain", "pipeline"], tokens: 119, level: "DEFAULT", sessionId: "session-pipeline-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "e694bc05e0f1e6d8d4f4b03aacc25193", name: "agent-tool-failure", timestamp: "2026-04-11 16:10:36", input: "", output: "None", observations: 4, latency: 92.25, latencyStr: "92.25s", tags: ["agent", "error", "tool-failure"], tokens: 2458, level: "ERROR", sessionId: "session-error-002", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "6f901207ccb3d6bd50d98a53bbe05d77", name: "rag-llm-error", timestamp: "2026-04-11 16:08:22", input: "如何优化 RAG 的召回率？", output: "None", observations: 3, latency: 0.0, latencyStr: "0.0s", tags: ["error", "rag", "rate-limit"], tokens: 0, level: "ERROR", sessionId: "session-error-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "baa577dc8640d82be6aab6e27082b669", name: "rag-normal", timestamp: "2026-04-11 16:08:02", input: "RAG 和微调有什么本质区别？", output: "根据提供的上下文，我无法回答这个问题。上下文主要介绍了 RAG 技术的基本原理、流程、优势以及 Prompt Engineering 的最佳实践，但没有涉及\u201c微调\u201d的相关信息。", observations: 3, latency: 3.81, latencyStr: "3.81s", tags: ["normal", "rag"], tokens: 265, level: "DEFAULT", sessionId: "session-prod-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "b0c545184cf29555be12ddbe6ad55a9b", name: "rag-normal", timestamp: "2026-04-11 16:08:06", input: "向量数据库的 ANN 算法原理是什么？", output: "关于向量数据库的 ANN（近似最近邻）算法原理，我无法基于当前提供的上下文回答这个问题。上下文主要介绍了 DeepSeek 模型的特点和 LLM 评估指标。", observations: 3, latency: 3.88, latencyStr: "3.88s", tags: ["normal", "rag"], tokens: 269, level: "DEFAULT", sessionId: "session-prod-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
  { id: "a134190885b4bbf7cebdca7b6d6c5c44", name: "rag-qa", timestamp: "2026-04-11 15:27:23", input: "RAG 技术的核心流程是什么？", output: "RAG 技术的核心流程是：先将用户问题转化为向量，在知识库中检索最相关的文档片段，再将检索结果作为上下文输入给 LLM，让模型基于真实知识生成回答。", observations: 3, latency: 2.61, latencyStr: "2.61s", tags: ["deepseek", "rag", "test"], tokens: 205, level: "DEFAULT", sessionId: "test-session-001", metadata: { resourceAttributes: { "telemetry.sdk.language": "python", "telemetry.sdk.name": "opentelemetry", "telemetry.sdk.version": "1.41.0", "service.name": "unknown_service" }, scope: { name: "langfuse-sdk", version: "3.7.0", attributes: { public_key: "pk-lf-a20169c6-10b2-4fe9-af02-56b6019e6c1a" } } } },
];

// ═══ Session → Trace ID 映射（完全来自 cls_real_data.js + langfuse_real_data.js）═══

export const sessionTracesMap: Record<string, string[]> = {
  // CLS sessions
  "c9cf70cc-57c4-409d-a712-a91681b94796": ["e4ff73dcdc2c1e8d1c85256cdb47622d"],
  "dde18c43-8772-4652-96df-66131da659c3": ["a492358fd67d6726e4072e600c19a6f1"],
  "1e297787-c415-487c-b517-9048ef1e0611": ["db319ec0ee9269774e846ff8198e57fe"],
  "cb1986ed-d868-4478-94af-c43d35fd27e1": ["84c04f8eb6cbe048ace317c1555c9dec"],
  "670ad354-c740-4ea7-ad4a-43a7f549bfd3": ["38e6203d3b12f456863805c8d018e100"],
  "3cd9e804-4212-419a-8420-c3c5c92eadc0": ["87f13f28e5815e57f8a4f931b7fb28d9"],
  "0d3cc2f8-7975-48fd-b2e1-26697326f832": ["77b67546bc4c99239889093094108963"],
  // Langfuse sessions
  "session-pipeline-001": ["a7da65265270a3a95c6e112d451e964b", "a382be4e9838bc7254f23944869f806c", "8b3f999f1a95f7b7c04f2c35c3f54ac5", "8229efbc3f84f97855fa1c1ffc702d48"],
  "session-error-002": ["e694bc05e0f1e6d8d4f4b03aacc25193"],
  "session-error-001": ["6f901207ccb3d6bd50d98a53bbe05d77"],
  "session-prod-001": ["baa577dc8640d82be6aab6e27082b669", "b0c545184cf29555be12ddbe6ad55a9b"],
  "test-session-001": ["a134190885b4bbf7cebdca7b6d6c5c44"],
};

// ═══ DAG 节点（默认，实际应使用 getDagNodesForTrace）═══

export const DAG_NODES = ["__start__", "chat tc-code-latest", "execute_tool browser", "__end__"];

// ═══ 辅助函数 ═══

export function fmtTokens(n: number): string {
  if (!n) return "0";
  if (n >= 1000000) return (n / 1000000).toFixed(1) + "M";
  if (n >= 1000) return (n / 1000).toFixed(1) + "K";
  return String(n);
}

export function getTraceById(id: string): TraceData | undefined {
  return tracesData.find((t) => t.id === id);
}

export function getSessionTraces(sessionId: string): TraceData[] {
  const ids = sessionTracesMap[sessionId] || [];
  return ids.map((id) => tracesData.find((t) => t.id === id)).filter(Boolean) as TraceData[];
}

export function getSessionById(sessionId: string): SessionData | undefined {
  return sessionsData.find((s) => s.id === sessionId);
}
