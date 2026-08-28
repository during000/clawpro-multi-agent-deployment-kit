/**
 * SessionDetailLegacy - 旧版（v1）会话详情页面
 *
 * 视觉对齐 v2 0611 换皮规范（管控端 Admin）：
 *   - 顶部 KPI：3 张 NumberCard（TotalTokensIcon / CostIcon / RequestsIcon），统一品牌蓝渐变图标
 *   - 容器：SurfaceCard / SurfaceInner（4px 圆角，--shadow-card）
 *   - 文字：全部走 --text-title / --text-body / --text-muted token，不再 text-gray-*
 *   - 边框：--border (#EAEEF4)，不再 border-gray-100
 *   - 图表：品牌蓝 var(--cp-brand-blue, #1447E6)，网格线 #EAEEF4，刻度文字 #64748B
 *   - 角色标签：StatusTag（blue=user / purple→蓝软 / gray=tool/assistant）
 *   - 表格：Table 组件家族（与 v2 完全一致）
 *
 * 版本路由保留：localStorage.sessionPageVersion !== "v2" 时走本组件。
 * 数据结构与旧版完全一致，仅做视觉换皮，不改交互链表格列与图表数据。
 */
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import { useLocation } from "wouter";
import {
  Tooltip as UITooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { SurfaceCard } from "@/components/ui/Surface";
import { BackButton } from "@/components/ui/back-button";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
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

// ─── Mock 数据 ────────────────────────────────────────────────────────────────

// Token 流量数据
const TOKEN_FLOW = [
  { round: 1, input: 17, output: 315 },
  { round: 2, input: 17, output: 100 },
  { round: 3, input: 17, output: 109 },
  { round: 4, input: 17, output: 185 },
  { round: 5, input: 18, output: 185 },
  { round: 6, input: 18, output: 188 },
  { round: 7, input: 18, output: 185 },
  { round: 8, input: 18, output: 185 },
  { round: 9, input: 18, output: 185 },
];

// 交互链数据
const INTERACTION_CHAIN = [
  {
    timestamp: "2026-03-04 13:32:00",
    role: "user",
    content: "你能干啥",
    model: "—",
    stopReason: "—",
    input: "—",
    output: "—",
    cacheRW: "—",
    tokens: "—",
    cost: "—",
    duration: "—",
  },
  {
    timestamp: "2026-03-04 13:32:13",
    role: "assistant",
    content: "你好！我是你的 AI 助手。我能帮你做很多事情，包括...",
    model: "deepseek-v3.2",
    stopReason: "stop",
    input: "17K",
    output: "315",
    cacheRW: "0/0",
    tokens: "17K",
    cost: "0.0024",
    duration: "13.6s",
  },
  {
    timestamp: "2026-03-04 13:32:45",
    role: "user",
    content: "你管理一下我在伊朗的局势",
    model: "—",
    stopReason: "—",
    input: "—",
    output: "—",
    cacheRW: "—",
    tokens: "—",
    cost: "—",
    duration: "—",
  },
  {
    timestamp: "2026-03-04 13:32:52",
    role: "assistant",
    content: "我是一个 AI 助手，无法直接管理现实中的政治局势。但我可以帮助你分析...",
    model: "deepseek-v3.2",
    stopReason: "toolUse",
    input: "17K",
    output: "100",
    cacheRW: "0/0",
    tokens: "17K",
    cost: "0.0024",
    duration: "6.8s",
  },
  {
    timestamp: "2026-03-04 13:32:59",
    role: "tool",
    content: '{"status": "error", "tool": "web_fetch", "error": "missing_brave_api_key", "message": "web_sear...',
    model: "—",
    stopReason: "—",
    input: "—",
    output: "—",
    cacheRW: "—",
    tokens: "—",
    cost: "—",
    duration: "—",
  },
  {
    timestamp: "2026-03-04 13:33:08",
    role: "assistant",
    content: "让我查证这个信息。我是一个 AI 助手，无法直接管理现实中的政治局势...",
    model: "deepseek-v3.2",
    stopReason: "toolUse",
    input: "18K",
    output: "185",
    cacheRW: "0/0",
    tokens: "18K",
    cost: "0.0025",
    duration: "7.5s",
  },
];

// ─── 私有：费用渐变图标（与 v2 同款，复用 GradientIcon + DollarSign 路径） ─────
function CostIcon() {
  return (
    <GradientIcon aria-label="成本总量">
      <path d="M9 1.5c.41 0 .75.34.75.75v1.13c1.71.27 3 1.55 3 3.12 0 .41-.34.75-.75.75s-.75-.34-.75-.75c0-.74-.78-1.5-1.88-1.69V8.4l.6.15c1.61.4 3.03 1.13 3.03 2.93 0 1.62-1.3 2.9-3.03 3.17v1.1c0 .41-.34.75-.75.75s-.75-.34-.75-.75v-1.1c-1.74-.27-3.03-1.55-3.03-3.17 0-.41.34-.75.75-.75s.75.34.75.75c0 .8.83 1.59 2.03 1.78V9.85l-.6-.15C6.79 9.3 5.37 8.57 5.37 6.77c0-1.55 1.29-2.84 3.03-3.12V2.25c0-.41.34-.75.75-.75Zm-.75 3.4c-1.07.2-1.88.95-1.88 1.87 0 .83.6 1.27 1.88 1.6V4.9Zm1.5 5.45v2.85c1.2-.19 2.03-.98 2.03-1.78 0-.86-.66-1.32-2.03-1.7Z" />
    </GradientIcon>
  );
}

// 角色 → StatusTag variant 映射（与平台用户角色 / 监控筛选保持一致的语义色）
function roleVariant(role: string): "blue" | "purple" | "gray" {
  if (role === "user") return "blue";
  if (role === "assistant") return "purple";
  return "gray";
}

// ─── 主组件 ───────────────────────────────────────────────────────────────────

interface SessionDetailLegacyProps {
  params?: { id: string };
}

export default function SessionDetailLegacy({ params }: SessionDetailLegacyProps) {
  const sessionId = params?.id || "fb766833";
  const [, setLocation] = useLocation();

  // Mock 会话信息
  const sessionInfo = {
    id: sessionId,
    name: "会话详情",
    channel: "Feishu Dm",
    model: "deepseek-v3.2",
    totalCost: "$0.2743",
    avgCostPerRound: "$0.0076",
    totalTokens: "1.95M",
    totalRounds: 63,
    lastActiveTime: "2026-03-04 21:06",
    openClawName: "Agent-A",
  };

  // 成本趋势按 token 数粗略推算（与原数据保持一致）
  const COST_TREND = TOKEN_FLOW.map((item, idx) => ({
    minute: idx + 1,
    cost: Number(((0.0024 * (item.input + item.output)) / 1000).toFixed(4)),
  }));

  return (
    <div className="page-enter space-y-8">
      {/* 返回按钮 — 直接跳转到会话管理列表，避免 history.back() 在直链/外链入口时失效 */}
      <div>
        <BackButton onClick={() => setLocation("/admin/session-management")}>
          返回会话管理
        </BackButton>
      </div>

      {/* 标题区 */}
      <AdminPageHeader
        title="会话详情"
        description={
          <>
            会话 ID: {sessionInfo.id} · Agent 名称: {sessionInfo.openClawName}
          </>
        }
      />

      {/* ══ 顶部指标卡 ══════════════════════════════════════════════════════════ */}
      <div className="grid grid-cols-3 gap-5">
        <NumberCard
          icon={<TotalTokensIcon />}
          label="TOKEN 总量"
          value={sessionInfo.totalTokens}
        />
        <NumberCard
          icon={<CostIcon />}
          label="成本总量"
          value={sessionInfo.totalCost}
        />
        <NumberCard
          icon={<RequestsIcon />}
          label="会话轮次"
          value={sessionInfo.totalRounds}
        />
      </div>

      {/* ══ 图表区 ══════════════════════════════════════════════════════════════ */}
      <div className="grid grid-cols-2 gap-5">
        {/* Token 流量 */}
        <SurfaceCard className="overflow-hidden p-0">
          <div className="flex items-center justify-between px-5 py-3.5 border-b border-[var(--border)]">
            <span className="text-sm font-semibold text-[var(--text-title)]">Token 流量</span>
            <div className="flex items-center gap-3 text-[11px] text-[var(--text-muted)]">
              <span className="inline-flex items-center gap-1.5">
                <span className="size-2 rounded-full bg-[var(--cp-brand-blue,#1447E6)]" />
                Input
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="size-2 rounded-full bg-[#7CA3F5]" />
                Output
              </span>
            </div>
          </div>
          <div className="px-4 pt-4 pb-2">
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={TOKEN_FLOW} margin={{ top: 8, right: 8, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis
                  dataKey="round"
                  tick={{ fontSize: 11, fill: "var(--text-muted)" }}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: "var(--text-muted)" }}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip
                  contentStyle={{
                    background: "#fff",
                    border: "1px solid var(--border)",
                    borderRadius: 4,
                    fontSize: 12,
                    boxShadow: "var(--shadow-popover, 0 4px 16px rgba(15,23,42,0.08))",
                  }}
                  cursor={{ fill: "rgba(20,71,230,0.04)" }}
                />
                <Bar dataKey="input" fill="var(--cp-brand-blue, #1447E6)" radius={[2, 2, 0, 0]} />
                <Bar dataKey="output" fill="#7CA3F5" radius={[2, 2, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </SurfaceCard>

        {/* 成本趋势 */}
        <SurfaceCard className="overflow-hidden p-0">
          <div className="flex items-center justify-between px-5 py-3.5 border-b border-[var(--border)]">
            <span className="text-sm font-semibold text-[var(--text-title)]">成本趋势</span>
            <span className="inline-flex items-center gap-1.5 text-[11px] text-[var(--text-muted)]">
              <span className="size-2 rounded-full bg-[var(--cp-brand-blue,#1447E6)]" />
              Cost (USD)
            </span>
          </div>
          <div className="px-4 pt-4 pb-2">
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={COST_TREND} margin={{ top: 8, right: 8, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis
                  dataKey="minute"
                  tick={{ fontSize: 11, fill: "var(--text-muted)" }}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: "var(--text-muted)" }}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip
                  contentStyle={{
                    background: "#fff",
                    border: "1px solid var(--border)",
                    borderRadius: 4,
                    fontSize: 12,
                    boxShadow: "var(--shadow-popover, 0 4px 16px rgba(15,23,42,0.08))",
                  }}
                  cursor={{ fill: "rgba(20,71,230,0.04)" }}
                />
                <Bar dataKey="cost" fill="var(--cp-brand-blue, #1447E6)" radius={[2, 2, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </SurfaceCard>
      </div>

      {/* ══ 交互链 ═════════════════════════════════════════════════════════════ */}
      <div className="space-y-4">
        <div className="text-sm font-semibold text-[var(--text-title)]">交互链</div>
        <SurfaceCard className="overflow-hidden p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>角色</TableHead>
                  <TableHead>内容</TableHead>
                  <TableHead>模型</TableHead>
                  <TableHead>停止原因</TableHead>
                  <TableHead className="text-right">INPUT</TableHead>
                  <TableHead className="text-right">OUTPUT</TableHead>
                  <TableHead className="text-right">CACHE R/W</TableHead>
                  <TableHead className="text-right">TOKENS</TableHead>
                  <TableHead className="text-right">成本</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {INTERACTION_CHAIN.map((item, idx) => (
                  <TableRow key={idx}>
                    <TableCell className="text-[var(--text-body)] tabular-nums whitespace-nowrap">
                      {item.timestamp}
                    </TableCell>
                    <TableCell>
                      <StatusTag mode="soft" variant={roleVariant(item.role)}>
                        {item.role}
                      </StatusTag>
                    </TableCell>
                    <TableCell className="max-w-xs">
                      <TooltipProvider>
                        <UITooltip>
                          <TooltipTrigger asChild>
                            <span className="block truncate cursor-help text-[var(--text-body)]">
                              {item.content}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs">
                            {item.content}
                          </TooltipContent>
                        </UITooltip>
                      </TooltipProvider>
                    </TableCell>
                    <TableCell className="text-[var(--text-body)] whitespace-nowrap">
                      {item.model}
                    </TableCell>
                    <TableCell className="text-[var(--text-body)] whitespace-nowrap">
                      {item.stopReason}
                    </TableCell>
                    <TableCell className="text-right text-[var(--text-body)] tabular-nums">
                      {item.input}
                    </TableCell>
                    <TableCell className="text-right text-[var(--text-body)] tabular-nums">
                      {item.output}
                    </TableCell>
                    <TableCell className="text-right text-[var(--text-body)] tabular-nums">
                      {item.cacheRW}
                    </TableCell>
                    <TableCell className="text-right text-[var(--text-body)] font-mono tabular-nums">
                      {(() => {
                        if (item.input === "—" || item.output === "—") return "—";
                        const inputStr = (item.input as string).replace("K", "");
                        const inputNum = parseInt(inputStr) * 1000;
                        const outputNum = parseInt(item.output as string);
                        return (inputNum + outputNum).toLocaleString();
                      })()}
                    </TableCell>
                    <TableCell className="text-right text-[var(--text-body)] font-mono tabular-nums">
                      {item.cost === "—" ? "—" : `$${item.cost}`}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
