/**
 * SessionDetail - 会话详情页面
 * 使用 langfuse_real_data + cls_real_data 真实数据
 * Trace 抽屉使用共享组件 TraceDrawer
 *
 * 版本路由：根据 localStorage.sessionPageVersion 决定渲染新版（v2）还是旧版（v1）。
 * 与会话管理列表页保持一致：用户未升级到新版时，会话详情也应显示旧版样式。
 *
 * 视觉（v2 换皮 0611）：
 *   ┌─ 顶部 KPI ─┐ 3 张 NumberCard（RequestsIcon / TotalTokensIcon / 计费图标）
 *   ┌─ Trace 卡片 ┐ SurfaceCard + SurfaceInner，头部全部走 --text-* / --border token
 *   ┌─ 显示切换 ─┐ SegmentGroup / SegmentOption（替换手写 Formatted/JSON 按钮）
 *   ┌─ 字段表格 ─┐ Table 组件 + Typography 语义色（brand/success/danger）
 *   ┌─ JSON 模式 ─┐ SurfaceInner + alert-info-bg 浅底，字体色全部 token
 */
import { useState, useCallback } from "react";
import { DollarSign } from "lucide-react";
import { useLocation } from "wouter";
import {
  type TraceData,
  fmtTokens,
  getSessionTraces as getSessionTracesFromData,
  getSessionById,
} from "@/data/trace-data";
import TraceDrawer from "@/components/TraceDrawer";
import SessionDetailLegacy from "./SessionDetailLegacy";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { BackButton } from "@/components/ui/back-button";
import { StatusTag } from "@/components/ui/status-tag";
import {
  NumberCard,
  RequestsIcon,
  TotalTokensIcon,
  GradientIcon,
} from "@/components/ui/number-card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";

// ─── 主组件 ───────────────────────────────────────────────────────────────────

interface SessionDetailProps {
  params?: { id: string };
}

export default function SessionDetail({ params }: SessionDetailProps) {
  // 旧版分流：与 SessionManagement 共用同一个版本 key
  const sessionPageVersion = typeof window !== "undefined"
    ? localStorage.getItem("sessionPageVersion")
    : null;
  if (sessionPageVersion !== "v2") {
    return <SessionDetailLegacy params={params} />;
  }

  return <SessionDetailV2 params={params} />;
}

/** 费用渐变图标（沿用 NumberCard.GradientIcon 渐变规范，复用 lucide DollarSign 的 path） */
function CostIcon() {
  return (
    <GradientIcon aria-label="总费用">
      {/* DollarSign 路径占位：维持 18×18 视觉重量，与其他渐变图标对齐 */}
      <path d="M9 1.5c.41 0 .75.34.75.75v1.13c1.71.27 3 1.55 3 3.12 0 .41-.34.75-.75.75s-.75-.34-.75-.75c0-.74-.78-1.5-1.88-1.69V8.4l.6.15c1.61.4 3.03 1.13 3.03 2.93 0 1.62-1.3 2.9-3.03 3.17v1.1c0 .41-.34.75-.75.75s-.75-.34-.75-.75v-1.1c-1.74-.27-3.03-1.55-3.03-3.17 0-.41.34-.75.75-.75s.75.34.75.75c0 .8.83 1.59 2.03 1.78V9.85l-.6-.15C6.79 9.3 5.37 8.57 5.37 6.77c0-1.55 1.29-2.84 3.03-3.12V2.25c0-.41.34-.75.75-.75Zm-.75 3.4c-1.07.2-1.88.95-1.88 1.87 0 .83.6 1.27 1.88 1.6V4.9Zm1.5 5.45v2.85c1.2-.19 2.03-.98 2.03-1.78 0-.86-.66-1.32-2.03-1.7Z" />
    </GradientIcon>
  );
}

function SessionDetailV2({ params }: SessionDetailProps) {
  const sessionId = params?.id || "c9cf70cc-57c4-409d-a712-a91681b94796";
  const [, setLocation] = useLocation();
  const [displayMode, setDisplayMode] = useState<"formatted" | "json">("formatted");
  const [drawerTrace, setDrawerTrace] = useState<TraceData | null>(null);

  const sessionTraces = getSessionTracesFromData(sessionId);
  const session = getSessionById(sessionId);
  const totalTokens =
    sessionTraces.reduce((s, t) => s + t.tokens, 0) || session?.totalTokens || 0;
  const totalCost = session?.cost || 0;

  const handleOpenDrawer = useCallback((trace: TraceData) => {
    setDrawerTrace(trace);
  }, []);

  const handleCloseDrawer = useCallback(() => {
    setDrawerTrace(null);
  }, []);

  return (
    <div className="page-enter space-y-8">
      {/* Trace 详情抽屉 */}
      {drawerTrace && (
        <TraceDrawer trace={drawerTrace} onClose={handleCloseDrawer} />
      )}

      {/* 返回按钮 — 直接跳转到会话管理列表，避免 history.back() 在直链/外链入口时失效 */}
      <div>
        <BackButton onClick={() => setLocation("/admin/session-management")}>
          返回会话管理
        </BackButton>
      </div>

      <AdminPageHeader
        title="会话详情"
        description={<>会话 ID: {sessionId}</>}
      />

      {/* ══ 顶部指标卡 ══════════════════════════════════════════════════════════ */}
      <div className="grid grid-cols-3 gap-5">
        <NumberCard
          icon={<RequestsIcon />}
          label="TRACE 数"
          value={sessionTraces.length}
        />
        <NumberCard
          icon={<TotalTokensIcon />}
          label="TOKEN 总量"
          value={fmtTokens(totalTokens)}
        />
        <NumberCard
          icon={<CostIcon />}
          label="总费用"
          value={totalCost > 0 ? `$${totalCost.toFixed(4)}` : "—"}
        />
      </div>

      {/* ══ 交互链 — Trace 卡片列表 ═══════════════════════════════════════════════ */}
      <div className="space-y-5">
        {sessionTraces.map((trace) => {
          const isError = trace.level === "ERROR";
          return (
            <SurfaceCard key={trace.id} className="overflow-hidden p-0">
              {/* 卡片头部 */}
              <div className="flex items-center justify-between gap-3 px-4 py-2.5 border-b border-[var(--border)] bg-[var(--bg-grey-normal)]">
                <div className="flex items-center gap-3 min-w-0">
                  <span className="text-xs font-mono text-[var(--text-muted)] truncate">
                    {trace.id}
                  </span>
                  {isError && (
                    <StatusTag mode="soft" variant="red">
                      异常
                    </StatusTag>
                  )}
                  <button
                    onClick={() => handleOpenDrawer(trace)}
                    className="text-[var(--text-brand)] text-xs font-medium cursor-pointer hover:underline inline-flex items-center gap-1 shrink-0"
                  >
                    查看 Trace 详情 →
                  </button>
                </div>
                <div className="flex items-center gap-4 shrink-0">
                  {trace.tokens > 0 && (
                    <span className="text-[11px] text-[var(--text-muted)]">
                      Token:{" "}
                      <span className="font-semibold text-[var(--text-title)] tabular-nums">
                        {fmtTokens(trace.tokens)}
                      </span>
                    </span>
                  )}
                  <span
                    className={`text-[11px] ${
                      isError
                        ? "text-[var(--text-danger)]"
                        : "text-[var(--text-muted)]"
                    }`}
                  >
                    延迟:{" "}
                    <span className="font-semibold tabular-nums">
                      {trace.latencyStr}
                    </span>
                  </span>
                  <SegmentGroup className="h-7">
                    <SegmentOption
                      active={displayMode === "formatted"}
                      onClick={() => setDisplayMode("formatted")}
                      className="h-[calc(100%-1px)] px-3 text-xs"
                    >
                      Formatted
                    </SegmentOption>
                    <SegmentOption
                      active={displayMode === "json"}
                      onClick={() => setDisplayMode("json")}
                      className="h-[calc(100%-1px)] px-3 text-xs"
                    >
                      JSON
                    </SegmentOption>
                  </SegmentGroup>
                </div>
              </div>

              {/* 卡片内容 */}
              {displayMode === "formatted" ? (
                <div className="px-4 py-4 space-y-5">
                  <div>
                    <div className="text-sm font-semibold mb-2 text-[var(--text-title)]">
                      输入
                    </div>
                    <SurfaceInner className="overflow-hidden">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead className="w-[120px]">字段</TableHead>
                            <TableHead>值</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          <TableRow>
                            <TableCell className="font-mono text-[var(--text-brand)]">
                              request
                            </TableCell>
                            <TableCell className="text-[var(--text-success)] whitespace-pre-wrap break-words">
                              "{trace.input}"
                            </TableCell>
                          </TableRow>
                        </TableBody>
                      </Table>
                    </SurfaceInner>
                  </div>
                  <div>
                    <div className="text-sm font-semibold mb-2 text-[var(--text-title)]">
                      输出
                    </div>
                    <SurfaceInner className="overflow-hidden">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead className="w-[120px]">字段</TableHead>
                            <TableHead>值</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          <TableRow>
                            <TableCell className="font-mono text-[var(--text-brand)]">
                              response
                            </TableCell>
                            <TableCell
                              className={`whitespace-pre-wrap break-words ${
                                isError
                                  ? "text-[var(--text-danger)]"
                                  : "text-[var(--text-success)]"
                              }`}
                            >
                              "{trace.output}"
                            </TableCell>
                          </TableRow>
                        </TableBody>
                      </Table>
                    </SurfaceInner>
                  </div>
                </div>
              ) : (
                <div className="px-4 py-4 space-y-5">
                  <div>
                    <div className="text-sm font-semibold mb-2 text-[var(--text-title)]">
                      输入
                    </div>
                    <div className="p-3 font-mono text-[12px] leading-relaxed overflow-x-auto rounded-[var(--radius)] bg-[var(--alert-info-bg)] text-[var(--text-brand)]">
                      {"{ "}
                      <span className="text-[var(--text-brand-deep)]">
                        1 Items ∨
                      </span>
                      <br />
                      {"  request: "}
                      <span className="text-[var(--text-success)]">
                        "{trace.input}"
                      </span>
                      <br />
                      {"}"}
                    </div>
                  </div>
                  <div>
                    <div className="text-sm font-semibold mb-2 text-[var(--text-title)]">
                      输出
                    </div>
                    <div
                      className={`p-3 font-mono text-[12px] leading-relaxed overflow-x-auto rounded-[var(--radius)] ${
                        isError
                          ? "bg-[var(--alert-danger-bg,#FEF2F2)] text-[var(--text-danger)]"
                          : "bg-[var(--alert-info-bg)] text-[var(--text-brand)]"
                      }`}
                    >
                      {"{ "}
                      <span
                        className={
                          isError
                            ? "text-[var(--text-danger)] opacity-70"
                            : "text-[var(--text-brand-deep)]"
                        }
                      >
                        1 Items ∨
                      </span>
                      <br />
                      {"  response: "}
                      <span
                        className={
                          isError
                            ? "text-[var(--text-danger)]"
                            : "text-[var(--text-success)]"
                        }
                      >
                        "
                        {trace.output.length > 120
                          ? trace.output.slice(0, 120) + "..."
                          : trace.output}
                        "
                      </span>
                      <br />
                      {"}"}
                    </div>
                  </div>
                </div>
              )}
            </SurfaceCard>
          );
        })}
      </div>
    </div>
  );
}
