/**
 * TraceDrawer - Trace 详情侧边栏抽屉（共享组件）
 * 65% 宽度右侧抽屉，左半边 Span 树 + 右半边详情面板
 * 与 CLS Agent 可观测原型完全一致 + AI 分析入口
 */
import React, { useState, useEffect } from "react";
import { AlertTriangle, X, Search, ChevronDown, ChevronRight, Sparkles, Copy, Maximize2, Download } from "lucide-react";
import {
  type TraceData, type SpanData,
  fmtTokens, getSpansForTrace, getDagNodesForTrace,
} from "@/data/trace-data";

// 类型和常量
type SpanType = "TRACE" | "SPAN" | "GENERATION" | "RETRIEVER";
const TYPE_COLORS: Record<SpanType, string> = { TRACE: "bg-blue-500", SPAN: "bg-gray-400", GENERATION: "bg-amber-500", RETRIEVER: "bg-cyan-500" };
const TYPE_BADGE_COLORS: Record<SpanType, string> = { TRACE: "bg-blue-100 text-blue-700", SPAN: "bg-gray-100 text-gray-600", GENERATION: "bg-amber-100 text-amber-700", RETRIEVER: "bg-cyan-100 text-cyan-700" };

export interface TraceDrawerProps {
  trace: TraceData | null;
  onClose: () => void;
  onOpenSession?: (sessionId: string) => void;
}

export default function TraceDrawer({ trace, onClose, onOpenSession }: TraceDrawerProps) {
  const [selectedSpanId, setSelectedSpanId] = useState("root");
  const [spanSearch, setSpanSearch] = useState("");
  const [allExpanded, setAllExpanded] = useState(true);
  const [viewMode, setViewMode] = useState<"tree" | "timeline">("tree");
  const [detailTab, setDetailTab] = useState(0); // 0=预览, 1=日志
  const [fmtMode, setFmtMode] = useState<"formatted" | "json">("formatted");
  const [aiAnalysisSpan, setAiAnalysisSpan] = useState<string | null>(null);
  const [aiLoading, setAiLoading] = useState<string | null>(null); // spanId or "overall"
  const [aiResults, setAiResults] = useState<Record<string, string>>({});
  const [expandedLogSpans, setExpandedLogSpans] = useState<Set<string>>(new Set());
  const [showOverallAi, setShowOverallAi] = useState(false);
  const [overallAiLoading, setOverallAiLoading] = useState(false);
  const [overallAiReady, setOverallAiReady] = useState(false);

  useEffect(() => {
    if (trace) {
      setSelectedSpanId("root");
      setSpanSearch("");
      setAllExpanded(true);
      setViewMode("tree");
      setDetailTab(0);
      setFmtMode("formatted");
      setAiAnalysisSpan(null);
      setAiLoading(null);
      setAiResults({});
      setExpandedLogSpans(new Set());
      setShowOverallAi(false);
      setOverallAiLoading(false);
      setOverallAiReady(false);
    }
  }, [trace]);

  // ESC 关闭
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose]);

  if (!trace) return null;

  const traceId = trace.id || "";
  const SPANS_DATA = getSpansForTrace(traceId);
  const DAG_NODES = getDagNodesForTrace(traceId);

  const sel = SPANS_DATA.find((s) => s.id === selectedSpanId) || SPANS_DATA[0];
  const hasError = SPANS_DATA.some((s) => s.statusLevel === "ERROR");

  const toggleLogSpan = (id: string) => {
    setExpandedLogSpans((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  // AI 分析 mock 结果
  const getAiAnalysis = (sp: SpanData) => {
    if (sp.statusLevel === "ERROR") {
      return `🔴 错误分析：该 ${sp.type} 环节 "${sp.name}" 发生异常。\n\n根因判断：LLM API 调用返回 rate_limit_exceeded 错误，触发超时重试后仍然失败。\n\n建议：\n1. 检查 API Key 的 QPM/TPM 配额是否已达上限\n2. 增加重试间隔和退避策略\n3. 考虑接入备用模型做 fallback`;
    }
    if (sp.latency > 10) {
      return `⚠️ 性能分析：该环节 "${sp.name}" 耗时 ${sp.latency}s，明显偏高。\n\n可能原因：\n1. 输入 Token 数 ${sp.inTok} 较大，生成 Token ${sp.outTok} 量大\n2. 模型推理在高并发时排队\n\n优化建议：启用 streaming 模式降低首 Token 延迟`;
    }
    return `✅ 该环节 "${sp.name}" 运行正常，耗时 ${sp.latency >= 1 ? sp.latency + "s" : Math.round(sp.latency * 1000) + "ms"}，无异常。`;
  };

  // 触发 Span AI 分析（带 loading）
  const triggerSpanAi = (sp: SpanData) => {
    if (aiAnalysisSpan === sp.id) { setAiAnalysisSpan(null); return; }
    setAiAnalysisSpan(sp.id);
    setSelectedSpanId(sp.id);
    if (aiResults[sp.id]) return; // 已有结果，直接展示
    setAiLoading(sp.id);
    setTimeout(() => {
      setAiResults(prev => ({ ...prev, [sp.id]: getAiAnalysis(sp) }));
      setAiLoading(null);
    }, 1500 + Math.random() * 1000);
  };

  // 触发整体 AI 分析（带 loading）
  const triggerOverallAi = () => {
    if (showOverallAi) { setShowOverallAi(false); return; }
    setShowOverallAi(true);
    if (overallAiReady) return; // 已有结果
    setOverallAiLoading(true);
    setTimeout(() => {
      setOverallAiLoading(false);
      setOverallAiReady(true);
    }, 2000 + Math.random() * 1000);
  };

  return (
    <>
      {/* Overlay */}
      <div
        className="fixed inset-0 bg-black/40 z-[100] transition-opacity duration-300"
        onClick={onClose}
      />
      {/* Panel */}
      <div
        className="fixed top-0 right-0 bottom-0 z-[101] bg-white flex flex-col overflow-hidden transition-transform duration-300"
        style={{ width: "65%", boxShadow: "-2px 0 20px rgba(0,0,0,0.12)" }}
      >
        {/* Header */}
        <div className="bg-white border-b border-gray-200 px-4 py-2.5 flex items-center justify-between flex-shrink-0">
          <div className="flex items-center gap-3">
            <nav className="flex items-center text-xs">
              <span className="text-blue-600 cursor-pointer hover:underline font-medium" onClick={onClose}>Traces</span>
              <span className="text-gray-400 mx-1.5">/</span>
              <span className="font-semibold">{trace.name}</span>
              <span className="text-gray-400 font-mono ml-1">({trace.id?.slice(0, 12) || trace.traceId})</span>
            </nav>
            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-violet-100 text-violet-700">Trace</span>
            {hasError && (
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-bold bg-red-100 text-red-700">
                <AlertTriangle className="w-3 h-3" /> ERROR
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-[11px] font-semibold transition-all ${
                showOverallAi
                  ? "bg-violet-100 text-violet-700 hover:bg-violet-200"
                  : "bg-gradient-to-r from-violet-500 to-blue-500 text-white hover:from-violet-600 hover:to-blue-600 shadow-sm"
              }`}
              onClick={() => triggerOverallAi()}
            >
              <Sparkles className="w-3 h-3" />
              {showOverallAi ? "收起分析" : "AI 分析"}
            </button>
            <button className="p-1 rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600" onClick={onClose}>
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* 整体 AI 分析面板 */}
        {showOverallAi && (
          <div className="px-4 py-3 bg-gradient-to-r from-violet-50 to-blue-50 border-b border-violet-200/60 flex-shrink-0">
            <div className="flex items-center gap-2 mb-2">
              <div className="w-5 h-5 rounded-md bg-gradient-to-r from-violet-500 to-blue-500 flex items-center justify-center">
                <Sparkles className="w-3 h-3 text-white" />
              </div>
              <span className="text-xs font-bold text-violet-800">Trace 整体分析</span>
              {overallAiLoading && <span className="text-[10px] text-violet-500 animate-pulse">分析中...</span>}
            </div>
            {overallAiLoading ? (
              <div className="space-y-2 py-1">
                {[85, 100, 70, 90, 60].map((w, i) => (
                  <div key={i} className="h-3 rounded-full bg-violet-200/40 animate-pulse" style={{ width: `${w}%`, animationDelay: `${i * 150}ms` }} />
                ))}
              </div>
            ) : (
              <div className="text-[11px] text-gray-700 leading-relaxed space-y-2">
                <div>
                  <span className="font-bold text-violet-800">概况：</span>
                  该 Trace「{trace.name}」共包含 {SPANS_DATA.length - 1} 个 Span，总耗时 {trace.latencyStr}，消耗 Token {fmtTokens(trace.tokens)}。
                  {hasError
                    ? ` 检测到 ${SPANS_DATA.filter(s => s.statusLevel === "ERROR").length} 个异常环节，需要重点关注。`
                    : " 所有环节运行正常。"
                  }
                </div>
                {hasError && (
                  <div>
                    <span className="font-bold text-red-700">异常诊断：</span>
                    {SPANS_DATA.filter(s => s.statusLevel === "ERROR").map(s =>
                      `「${s.name}」耗时 ${s.latency >= 1 ? s.latency + "s" : Math.round(s.latency * 1000) + "ms"}`
                    ).join("、")}
                    {" "}发生错误。建议：1) 检查工具调用的目标服务可用性；2) 确认 API Key 配额和网络连通性；3) 增加重试机制或 fallback 策略。
                  </div>
                )}
                {(() => {
                  const slowSpans = SPANS_DATA.filter(s => s.latency > 10 && s.statusLevel !== "ERROR");
                  if (slowSpans.length === 0) return null;
                  return (
                    <div>
                      <span className="font-bold text-amber-700">性能瓶颈：</span>
                      {slowSpans.map(s => `「${s.name}」${s.latency}s`).join("、")} 耗时较长，建议优化 Prompt 或启用 streaming 模式。
                    </div>
                  );
                })()}
                <div>
                  <span className="font-bold text-violet-800">Token 分布：</span>
                  输入 {SPANS_DATA.reduce((s, sp) => s + sp.inTok, 0)} Token，输出 {SPANS_DATA.reduce((s, sp) => s + sp.outTok, 0)} Token。
                  {SPANS_DATA.reduce((s, sp) => s + sp.inTok, 0) > 500 ? " 输入 Token 偏高，可考虑优化上下文裁剪策略。" : ""}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Body: Left span tree + Right detail */}
        <div className="flex flex-1 overflow-hidden">
          {/* Left: Span Tree */}
          <div className="w-1/2 border-r border-gray-200 flex flex-col overflow-hidden">
            {/* Toolbar */}
            <div className="px-3 py-2 bg-gray-50 border-b border-gray-200 flex items-center justify-between flex-shrink-0">
              <div className="flex items-center gap-2">
                <div className="relative">
                  <Search className="w-3 h-3 absolute left-2 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    type="text"
                    className="pl-7 pr-2 py-1 border border-gray-200 rounded text-[11px] w-[120px] outline-none focus:border-blue-500"
                    placeholder="搜索 Span..."
                    value={spanSearch}
                    onChange={(e) => setSpanSearch(e.target.value)}
                  />
                </div>
                <button
                  className="text-[10px] px-2 py-0.5 border border-gray-200 rounded text-gray-500 hover:text-blue-600 hover:border-blue-300"
                  onClick={() => setAllExpanded(!allExpanded)}
                >
                  {allExpanded ? "↑" : "↓"}
                </button>
              </div>
              <button
                className={`text-[10px] px-2 py-0.5 border rounded ${viewMode === "timeline" ? "border-blue-500 text-blue-600 bg-blue-50" : "border-gray-200 text-gray-500 hover:text-blue-600"}`}
                onClick={() => setViewMode(viewMode === "timeline" ? "tree" : "timeline")}
              >
                Timeline
              </button>
            </div>

            {/* Span rows */}
            <div className="flex-1 overflow-auto text-[11px]">
              {viewMode === "tree" ? (
                <>
                  {/* Time ruler */}
                  <div className="px-3 py-1 bg-gray-50 border-b border-gray-200 text-[9px] text-gray-400 flex justify-between">
                    {(() => {
                      const total = trace.latency || SPANS_DATA[0]?.latency || 1;
                      const steps = [0, 0.25, 0.5, 0.75, 1].map(p => {
                        const v = total * p;
                        return v >= 1 ? v.toFixed(1) + "s" : Math.round(v * 1000) + "ms";
                      });
                      return steps.map((s, i) => <span key={i}>{s}</span>);
                    })()}
                  </div>
                  {SPANS_DATA.map((sp) => {
                    const matchSearch = !spanSearch || sp.name.toLowerCase().includes(spanSearch.toLowerCase());
                    const isSelected = sp.id === selectedSpanId;
                    const hasChildren = sp.children.length > 0;
                    const hidden = !allExpanded && sp.level > 1;
                    const isSpanError = sp.statusLevel === "ERROR";
                    if (hidden) return null;
                    return (
                      <div key={sp.id}>
                        <div
                          className={`flex items-center cursor-pointer border-l-2 transition-all ${
                            isSpanError
                              ? isSelected ? "bg-red-50 border-l-red-600" : "bg-red-50/50 border-l-red-400 hover:bg-red-50"
                              : isSelected ? "bg-blue-50/80 border-l-blue-600" : "border-l-transparent hover:bg-blue-50/30"
                          } ${matchSearch ? "" : "opacity-40"}`}
                          style={{ padding: `7px 8px 7px ${10 + sp.level * 18}px` }}
                          onClick={() => setSelectedSpanId(sp.id)}
                        >
                          {hasChildren ? (
                            <span className="mr-1 text-gray-400 text-[10px]">{allExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}</span>
                          ) : <span className="w-3" />}
                          {sp.type !== "TRACE" && (
                            <span className={`mr-1.5 flex-shrink-0 inline-flex items-center justify-center w-3 h-3 rounded-sm ${
                              isSpanError ? "bg-red-100 text-red-700" : TYPE_BADGE_COLORS[sp.type as SpanType]
                            } text-[7px]`}>
                              {isSpanError ? "!" : sp.type === "GENERATION" ? "⚡" : sp.type === "RETRIEVER" ? "🔍" : "◻"}
                            </span>
                          )}
                          <span className={`font-medium flex-1 ${isSpanError ? "text-red-700" : ""}`}>
                            {sp.name}
                            {isSpanError && <span className="ml-1 px-1 py-0.5 rounded text-[9px] font-bold bg-red-100 text-red-600">ERROR</span>}
                            <span className={`font-normal ml-1 ${isSpanError ? "text-red-400" : "text-gray-400"}`}>{sp.latency >= 1 ? sp.latency + "s" : Math.round(sp.latency * 1000) + "ms"}</span>
                            {sp.inTok ? <span className="text-emerald-600 ml-1">{sp.inTok}→{sp.outTok}</span> : null}
                          </span>
                          {/* AI 诊断入口 — 仅异常 Span */}
                          {isSpanError && (
                            <button
                              className="mr-2 flex-shrink-0 inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[9px] font-medium transition-colors bg-red-100 text-red-700 hover:bg-red-200 ring-1 ring-red-300"
                              onClick={(e) => { e.stopPropagation(); triggerSpanAi(sp); }}
                              title="AI 诊断"
                            >
                              <Sparkles className="w-2.5 h-2.5" />
                              诊断错误
                            </button>
                          )}
                          <div className="w-[100px] h-3 bg-gray-100 rounded relative flex-shrink-0">
                            <div
                              className={`absolute h-full rounded ${isSpanError ? "bg-red-500" : TYPE_COLORS[sp.type as SpanType]}`}
                              style={{ left: `${sp.start}%`, width: `${Math.max(sp.end - sp.start, 0.5)}%`, top: "50%", transform: "translateY(-50%)" }}
                            />
                          </div>
                        </div>
                        {/* AI 分析展开面板 */}
                        {aiAnalysisSpan === sp.id && (
                          <div className={`mx-4 mb-2 mt-1 rounded-lg p-3 text-[11px] leading-relaxed whitespace-pre-wrap ${
                            isSpanError ? "bg-red-50 border border-red-200 text-red-800" : "bg-blue-50 border border-blue-200 text-blue-800"
                          }`}>
                            <div className="flex items-center gap-1.5 mb-2 font-semibold">
                              <Sparkles className={`w-3 h-3 ${aiLoading === sp.id ? "animate-spin" : ""}`} />
                              AI 智能分析
                              {aiLoading === sp.id && <span className="font-normal text-[10px] animate-pulse ml-1">分析中...</span>}
                            </div>
                            {aiLoading === sp.id ? (
                              <div className="space-y-1.5 py-1">
                                {[90, 75, 60, 85].map((w, i) => (
                                  <div key={i} className={`h-2.5 rounded-full ${isSpanError ? "bg-red-200/50" : "bg-blue-200/50"} animate-pulse`} style={{ width: `${w}%`, animationDelay: `${i * 120}ms` }} />
                                ))}
                              </div>
                            ) : (
                              aiResults[sp.id] || ""
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}

                  {/* DAG */}
                  <div className="border-t border-gray-200 p-4 bg-gray-50">
                    <div className="text-[10px] font-semibold text-gray-400 mb-3">流程图视图 (DAG)</div>
                    <div className="flex flex-col items-center gap-1">
                      {DAG_NODES.map((n, i) => {
                        const isEnd = n === "__start__" || n === "__end__";
                        const sp = SPANS_DATA.find((s) => s.name === n);
                        const isSelected = sp && sp.id === selectedSpanId;
                        const isSpanError = sp?.statusLevel === "ERROR";
                        const cls = isSpanError
                          ? "border-red-400 bg-red-50 text-red-700"
                          : isEnd
                            ? "border-emerald-400 bg-emerald-50 text-emerald-700"
                            : n === "retrieval"
                              ? "border-cyan-400 bg-cyan-50 text-cyan-700"
                              : sp?.type === "GENERATION"
                                ? "border-amber-400 bg-amber-50 text-amber-700"
                                : "border-gray-300 bg-white text-gray-700";
                        return (
                          <div key={n}>
                            <div
                              className={`inline-flex items-center gap-1 px-3 py-1.5 rounded text-[11px] font-medium border-[1.5px] cursor-pointer transition-all ${cls} ${isSelected ? "ring-2 ring-blue-600" : "hover:shadow-md"}`}
                              onClick={() => sp && setSelectedSpanId(sp.id)}
                            >
                              {isSpanError && <AlertTriangle className="w-3 h-3 text-red-600" />}
                              {n}{sp ? <span className={`ml-1 ${isSpanError ? "text-red-400" : "text-gray-400"}`}>{sp.latency >= 1 ? sp.latency + "s" : Math.round(sp.latency * 1000) + "ms"}</span> : null}
                            </div>
                            {i < DAG_NODES.length - 1 && (
                              <div className="flex justify-center"><div className={`w-px h-3 ${isSpanError && i === DAG_NODES.indexOf(n) ? "bg-red-300" : "bg-gray-300"}`} /></div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                </>
              ) : (
                /* Timeline view */
                <div className="p-4">
                  <div className="text-[10px] font-semibold text-gray-400 mb-3">Timeline 甘特图</div>
                  <div className="space-y-1">
                    {SPANS_DATA.filter((s) => s.id !== "root").map((sp) => {
                      const isSpanError = sp.statusLevel === "ERROR";
                      return (
                        <div key={sp.id} className={`flex items-center gap-2 text-[10px] py-1 ${isSpanError ? "bg-red-50/50" : ""}`}>
                          <div className={`w-[100px] truncate font-medium ${isSpanError ? "text-red-700" : ""}`}>
                            {isSpanError && <AlertTriangle className="w-3 h-3 text-red-500 inline mr-0.5" />}
                            {sp.name}
                          </div>
                          <div className="flex-1 h-5 bg-gray-100 rounded relative">
                            <div
                              className={`absolute h-full ${isSpanError ? "bg-red-500" : TYPE_COLORS[sp.type as SpanType]} rounded opacity-80 cursor-pointer hover:opacity-100`}
                              style={{ left: `${sp.start}%`, width: `${Math.max(sp.end - sp.start, 1)}%` }}
                              onClick={() => setSelectedSpanId(sp.id)}
                              title={`${sp.name}: ${sp.latency}s${isSpanError ? " [ERROR]" : ""}`}
                            />
                          </div>
                          <div className={`w-[40px] text-right ${isSpanError ? "text-red-500" : "text-gray-400"}`}>{sp.latency >= 1 ? sp.latency + "s" : Math.round(sp.latency * 1000) + "ms"}</div>
                          {isSpanError && (
                            <button
                              className="flex-shrink-0 inline-flex items-center gap-0.5 px-1 py-0.5 rounded text-[8px] font-medium bg-red-100 text-red-600 hover:bg-red-200"
                              onClick={() => triggerSpanAi(sp)}
                              title="AI 诊断"
                            >
                              <Sparkles className="w-2.5 h-2.5" />
                            </button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Right: Detail Panel */}
          <div className="w-1/2 flex flex-col overflow-hidden">
            {/* Info bar */}
            <div className="px-4 py-2 bg-white border-b border-gray-200 flex items-center gap-3 flex-wrap flex-shrink-0 text-[11px]">
              <span className="font-medium">延迟: <span className={`${trace.level === "ERROR" ? "text-red-600" : "text-blue-600"}`}>{trace.latencyStr}</span></span>
              <span className="text-gray-200">|</span>
              <span>Session: <span className="text-blue-600 cursor-pointer hover:underline" onClick={() => onOpenSession?.(trace.sessionId)}>{trace.sessionId}</span></span>
              <span className="text-gray-200">|</span>
              <span>Total: <strong>{fmtTokens(trace.tokens)}</strong> Token</span>
              <span className="text-gray-200">|</span>
              <span>{sel.inTok || 0} 输入 → {sel.outTok || 0} 输出</span>
            </div>

            {/* Tabs */}
            <div className="border-b border-gray-200 flex flex-shrink-0">
              {["预览", "日志"].map((label, i) => (
                <button
                  key={label}
                  className={`px-4 py-2.5 text-[11px] font-medium border-b-2 transition-colors ${detailTab === i ? "text-blue-600 border-blue-600" : "text-gray-400 border-transparent hover:text-blue-600"}`}
                  onClick={() => setDetailTab(i)}
                >
                  {label}
                </button>
              ))}
            </div>

            {/* Content */}
            <div className="flex-1 overflow-auto p-4">
              {/* Format toggle */}
              <div className="flex justify-end mb-3">
                <div className="flex gap-1">
                  {(["formatted", "json"] as const).map((mode) => (
                    <button
                      key={mode}
                      className={`text-[10px] px-2 py-0.5 border rounded transition-colors ${fmtMode === mode ? "border-blue-500 text-blue-600 bg-blue-50" : "border-gray-200 text-gray-500 hover:text-blue-600"}`}
                      onClick={() => setFmtMode(mode)}
                    >
                      {mode === "formatted" ? "Formatted" : "JSON"}
                    </button>
                  ))}
                </div>
              </div>

              {detailTab === 0 ? (
                /* ═══ 预览 Tab ═══ */
                fmtMode === "formatted" ? (
                  <div className="space-y-4">
                    {/* Tags */}
                    <div>
                      <div className="text-[11px] font-semibold text-gray-400 mb-2">Tags</div>
                      <div className="flex gap-1.5">
                        {trace.tags.map((tag) => (
                          <span key={tag} className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-purple-50 text-purple-600">{tag}</span>
                        ))}
                      </div>
                    </div>
                    {/* Input */}
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <div className="text-[11px] font-semibold text-gray-400">Input</div>
                        <div className="flex gap-1.5 text-gray-400">
                          <Maximize2 className="w-3 h-3 cursor-pointer hover:text-blue-600" />
                          <Copy className="w-3 h-3 cursor-pointer hover:text-blue-600" />
                        </div>
                      </div>
                      <table className="w-full border border-gray-200 rounded text-[11px]">
                        <thead><tr className="bg-gray-50 border-b border-gray-200"><th className="text-left px-3 py-2 font-semibold text-gray-500 uppercase text-[10px] tracking-wide" style={{ width: 100 }}>Path</th><th className="text-left px-3 py-2 font-semibold text-gray-500 uppercase text-[10px] tracking-wide">Value</th></tr></thead>
                        <tbody><tr><td className="px-3 py-2 font-mono text-blue-600 border-b border-gray-50">question</td><td className="px-3 py-2 text-green-700 border-b border-gray-50">"{sel.id === "root" ? trace.input : sel.name + " input data"}"</td></tr></tbody>
                      </table>
                    </div>
                    {/* Output */}
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <div className="text-[11px] font-semibold text-gray-400">Output</div>
                        <div className="flex gap-1.5 text-gray-400">
                          <Maximize2 className="w-3 h-3 cursor-pointer hover:text-blue-600" />
                          <Copy className="w-3 h-3 cursor-pointer hover:text-blue-600" />
                        </div>
                      </div>
                      <table className="w-full border border-gray-200 rounded text-[11px]">
                        <thead><tr className="bg-gray-50 border-b border-gray-200"><th className="text-left px-3 py-2 font-semibold text-gray-500 uppercase text-[10px] tracking-wide" style={{ width: 100 }}>Path</th><th className="text-left px-3 py-2 font-semibold text-gray-500 uppercase text-[10px] tracking-wide">Value</th></tr></thead>
                        <tbody><tr><td className="px-3 py-2 font-mono text-blue-600 border-b border-gray-50">answer</td><td className={`px-3 py-2 border-b border-gray-50 whitespace-pre-wrap ${sel.statusLevel === "ERROR" ? "text-red-600" : "text-green-700"}`}>"{sel.id === "root" ? trace.output : sel.name + " output data"}"</td></tr></tbody>
                      </table>
                    </div>
                    {/* Metadata */}
                    <div>
                      <div className="text-[11px] font-semibold text-gray-400 mb-2">Metadata</div>
                      <table className="w-full border border-gray-200 rounded text-[11px]">
                        <thead><tr className="bg-gray-50 border-b border-gray-200"><th className="text-left px-3 py-2 font-semibold text-gray-500 uppercase text-[10px] tracking-wide" style={{ width: 200 }}>Path</th><th className="text-left px-3 py-2 font-semibold text-gray-500 uppercase text-[10px] tracking-wide">Value</th></tr></thead>
                        <tbody>
                          <tr><td className="px-3 py-2 font-mono text-gray-500">▼ resourceAttributes</td><td className="px-3 py-2 text-gray-400">4 items</td></tr>
                          {Object.entries(trace.metadata.resourceAttributes ?? {}).map(([k, v]) => (
                            <tr key={k}><td className="px-3 py-2 font-mono text-blue-600 pl-6">{k}</td><td className="px-3 py-2 text-green-700">"{v}"</td></tr>
                          ))}
                          <tr><td className="px-3 py-2 font-mono text-gray-500">▼ scope</td><td className="px-3 py-2 text-gray-400">3 items</td></tr>
                          <tr><td className="px-3 py-2 font-mono text-blue-600 pl-6">name</td><td className="px-3 py-2 text-green-700">"{trace.metadata.scope?.name ?? ""}"</td></tr>
                          <tr><td className="px-3 py-2 font-mono text-blue-600 pl-6">version</td><td className="px-3 py-2 text-green-700">"{trace.metadata.scope?.version ?? ""}"</td></tr>
                          <tr><td className="px-3 py-2 font-mono text-gray-500 pl-6">▶ attributes</td><td className="px-3 py-2 text-gray-400">1 items</td></tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                ) : (
                  /* JSON 预览 */
                  <pre className="bg-gray-900 text-gray-300 rounded-lg p-3 text-[10px] overflow-x-auto leading-relaxed">
                    {JSON.stringify({
                      type: sel.type,
                      name: sel.name,
                      latency: sel.latency,
                      input_tokens: sel.inTok,
                      output_tokens: sel.outTok,
                      input: trace.input,
                      output: trace.output,
                      metadata: trace.metadata,
                    }, null, 2)}
                  </pre>
                )
              ) : (
                /* ═══ 日志 Tab ═══ */
                fmtMode === "json" ? (
                  <div className="bg-blue-50 rounded-lg p-4 font-mono text-[11px] text-blue-800 leading-relaxed overflow-x-auto">
                    {"{ "}<span className="text-blue-400">{SPANS_DATA.filter((s) => s.id !== "root").length} Items ∨</span><br />
                    {SPANS_DATA.filter((s) => s.id !== "root").map((sp, i) => (
                      <span key={sp.id}>
                        {"  "}{i}: {"{ "}<span className="text-blue-400">8 Items ∨</span><br />
                        {"    "}id: <span className="text-green-700">"{sp.id}"</span><br />
                        {"    "}type: <span className="text-green-700">"{sp.type}"</span><br />
                        {"    "}name: <span className="text-green-700">"{sp.name} ({sp.id})"</span><br />
                        {"    "}startTime: <span className="text-green-700">"2026-04-12 00:12:31 GMT+0800"</span><br />
                        {"    "}endTime: <span className="text-green-700">"2026-04-12 00:12:58 GMT+0800"</span><br />
                        {"    "}depth: <span className="text-amber-700">{sp.level}</span><br />
                        {"    "}input: {"{ "}<span className="text-blue-400">2 Items ∨</span><br />
                        {"      "}args: {"[ "}<span className="text-blue-400">2 Items ∨</span><br />
                        {"        "}0: <span className="text-green-700">"{trace.input || sp.name}"</span><br />
                        {"        "}1: <span className="text-green-700">"{trace.sessionId}"</span><br />
                        {"      "}]<br />
                        {"      "}kwargs: {"{ "}<span className="text-blue-400">0 Items ∨</span>{" }"}<br />
                        {"    }"}<br />
                        {"    "}metadata: {"{ "}<span className="text-blue-400">2 Items ∨</span><br />
                        {"      "}resourceAttributes: {"{ "}<span className="text-blue-400">4 Items ∨</span><br />
                        {"        "}telemetry.sdk.language: <span className="text-green-700">"python"</span><br />
                        {"        "}telemetry.sdk.name: <span className="text-green-700">"opentelemetry"</span><br />
                        {"        "}telemetry.sdk.version: <span className="text-green-700">"1.41.0"</span><br />
                        {"        "}service.name: <span className="text-green-700">"unknown_service"</span><br />
                        {"      }"}<br />
                        {"      "}scope: {"{ "}<span className="text-blue-400">3 Items ∨</span><br />
                        {"        "}name: <span className="text-green-700">"langfuse-sdk"</span><br />
                        {"        "}version: <span className="text-green-700">"3.7.0"</span><br />
                        {"        "}attributes: {"{ "}<span className="text-blue-400">1 Items ∨</span><br />
                        {"          "}public_key: <span className="text-green-700">"pk-lf-a20169c6"</span><br />
                        {"        }"}<br />
                        {"      }"}<br />
                        {"    }"}<br />
                        {"  }"}<br />
                      </span>
                    ))}
                    {"}"}
                  </div>
                ) : (
                  /* Formatted 日志 — Observation 表格 + 展开详情 */
                  <div>
                    <div className="flex items-center gap-2 mb-3">
                      <div className="relative flex-1">
                        <Search className="w-3 h-3 absolute left-2 top-1/2 -translate-y-1/2 text-gray-400" />
                        <input type="text" className="pl-7 pr-2 py-1 border border-gray-200 rounded text-[11px] w-[200px] outline-none focus:border-blue-500" placeholder="Search observations..." />
                      </div>
                    </div>
                    <table className="w-full border border-gray-200 rounded text-[11px]">
                      <thead>
                        <tr className="bg-gray-50 border-b border-gray-200">
                          <th className="text-left px-3 py-2 font-semibold text-gray-500 text-[10px] uppercase tracking-wide" style={{ width: "45%" }}>Observation</th>
                          <th className="text-left px-3 py-2 font-semibold text-gray-500 text-[10px] uppercase tracking-wide" style={{ width: "15%" }}>Depth</th>
                          <th className="text-left px-3 py-2 font-semibold text-gray-500 text-[10px] uppercase tracking-wide" style={{ width: "15%" }}>Start</th>
                          <th className="text-left px-3 py-2 font-semibold text-gray-500 text-[10px] uppercase tracking-wide" style={{ width: "25%" }}>Duration</th>
                        </tr>
                      </thead>
                      <tbody>
                        {SPANS_DATA.filter((s) => s.id !== "root").map((sp) => {
                          const isLogExpanded = expandedLogSpans.has(sp.id);
                          const isSpanError = sp.statusLevel === "ERROR";
                          return (
                            <React.Fragment key={sp.id}>
                              <tr
                                className={`cursor-pointer transition-colors ${
                                  isSpanError
                                    ? sp.id === selectedSpanId ? "bg-red-50 border-l-2 border-l-red-600" : "bg-red-50/40 hover:bg-red-50"
                                    : sp.id === selectedSpanId ? "bg-blue-50 border-l-2 border-l-blue-600" : "hover:bg-blue-50"
                                }`}
                                onClick={() => { setSelectedSpanId(sp.id); toggleLogSpan(sp.id); }}
                              >
                                <td className="py-2 px-3">
                                  <div className="flex items-center gap-1.5" style={{ paddingLeft: sp.level * 16 }}>
                                    <span className="text-gray-400 text-[10px]">{isLogExpanded ? "▼" : "▶"}</span>
                                    <span className={`inline-flex items-center px-1 rounded text-[9px] font-medium ${
                                      isSpanError ? "bg-red-100 text-red-700" : TYPE_BADGE_COLORS[sp.type as SpanType]
                                    }`}>
                                      {isSpanError ? "!" : sp.type === "GENERATION" ? "⚡" : sp.type === "RETRIEVER" ? "🔍" : "◻"}
                                    </span>
                                    <span className={`font-medium ${isSpanError ? "text-red-700" : ""}`}>{sp.name}</span>
                                    <span className="text-gray-400 font-mono text-[9px]">({sp.id})</span>
                                    {isSpanError && <span className="px-1 py-0.5 rounded text-[8px] font-bold bg-red-100 text-red-600">ERROR</span>}
                                    {sp.children.length > 0 && <span className="text-gray-400 text-[9px]">{sp.children.length} items</span>}
                                  </div>
                                </td>
                                <td className="py-2 px-3 text-gray-400">L{sp.level}</td>
                                <td className="py-2 px-3 text-gray-400">0:{String(Math.floor(sp.start * 0.27)).padStart(2, "0")}</td>
                                <td className={`py-2 px-3 font-medium ${isSpanError ? "text-red-600" : sp.latency > 10 ? "text-amber-600" : ""}`}>
                                  {sp.latency >= 1 ? sp.latency + "s" : Math.round(sp.latency * 1000) + "ms"}
                                </td>
                              </tr>
                              {isLogExpanded && (
                                <tr className={isSpanError ? "bg-red-50/30" : "bg-blue-50/50"}>
                                  <td colSpan={4} className="px-6 py-3">
                                    <div className="flex items-center justify-between mb-2">
                                      <span className="text-[10px] font-semibold text-gray-500">Span 详情</span>
                                      {isSpanError && (
                                        <button
                                          className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[9px] font-medium bg-red-100 text-red-700 hover:bg-red-200"
                                          onClick={(e) => { e.stopPropagation(); triggerSpanAi(sp); }}
                                        >
                                          <Sparkles className="w-2.5 h-2.5" />
                                          AI 诊断错误
                                        </button>
                                      )}
                                    </div>
                                    {aiAnalysisSpan === sp.id && (
                                      <div className={`mb-3 rounded-lg p-3 text-[11px] leading-relaxed whitespace-pre-wrap ${
                                        isSpanError ? "bg-red-50 border border-red-200 text-red-800" : "bg-blue-50 border border-blue-200 text-blue-800"
                                      }`}>
                                        <div className="flex items-center gap-1.5 mb-2 font-semibold">
                                          <Sparkles className={`w-3 h-3 ${aiLoading === sp.id ? "animate-spin" : ""}`} />
                                          AI 智能分析
                                          {aiLoading === sp.id && <span className="font-normal text-[10px] animate-pulse ml-1">分析中...</span>}
                                        </div>
                                        {aiLoading === sp.id ? (
                                          <div className="space-y-1.5 py-1">
                                            {[90, 75, 60, 85].map((w, i) => (
                                              <div key={i} className={`h-2.5 rounded-full ${isSpanError ? "bg-red-200/50" : "bg-blue-200/50"} animate-pulse`} style={{ width: `${w}%`, animationDelay: `${i * 120}ms` }} />
                                            ))}
                                          </div>
                                        ) : (
                                          aiResults[sp.id] || ""
                                        )}
                                      </div>
                                    )}
                                    <table className="w-full border border-gray-200 rounded text-[11px]">
                                      <thead><tr className="bg-gray-50 border-b border-gray-200"><th className="text-left px-3 py-2 font-semibold text-gray-500 text-[10px] uppercase" style={{ width: 120 }}>Path</th><th className="text-left px-3 py-2 font-semibold text-gray-500 text-[10px] uppercase">Value</th></tr></thead>
                                      <tbody>
                                        <tr><td className="px-3 py-2 font-mono text-blue-600">input</td><td className={`px-3 py-2 text-[10px] ${isSpanError ? "text-red-600" : "text-green-700"}`}>"{(trace.input || sp.name + " input").slice(0, 100)}{(trace.input || "").length > 100 ? "..." : ""}"</td></tr>
                                        <tr><td className="px-3 py-2 font-mono text-blue-600">output</td><td className={`px-3 py-2 text-[10px] ${isSpanError ? "text-red-600" : "text-green-700"}`}>"{(trace.output || "").slice(0, 100)}{(trace.output || "").length > 100 ? "..." : ""}"</td></tr>
                                        <tr><td className="px-3 py-2 font-mono text-blue-600">metadata</td><td className="px-3 py-2 text-[10px] text-green-700">"{JSON.stringify(trace.metadata || {}).slice(0, 120)}..."</td></tr>
                                      </tbody>
                                    </table>
                                  </td>
                                </tr>
                              )}
                            </React.Fragment>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
