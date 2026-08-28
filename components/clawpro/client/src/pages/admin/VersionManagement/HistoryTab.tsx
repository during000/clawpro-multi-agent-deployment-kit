/**
 * HistoryTab - 运维记录（审计视角）
 *
 * 方案一：版本管理 + 命令下发统一在此审计
 *   - 「运维内容」列动态展示：
 *     · agent-upgrade  → "Agent 升级 X.X → Y.Y"
 *     · command-execute → "命令：清理临时日志"
 *   - 「类型」筛选：全部 / Agent 升级 / 命令执行
 *   - 详情弹窗按 action 类型分发：升级走原版字段，命令走 stdout/stderr/exitCode
 *
 * 通过 scope prop 控制范围：
 *   - "all"             → 全部记录（独立运维任务页用，已下线）
 *   - "agent-upgrade"   → 只看 Agent 升级（用于 Agent 版本页 → 更新记录 Tab）
 *   - "command-execute" → 只看命令执行（用于 命令下发页 → 执行记录 Tab）
 * scope 锁定时，类型筛选下拉会被隐藏。
 *
 * 样式对齐设计规范：rounded-2xl + 双层 boxShadow + h-9 控件
 */
import { useState, useMemo, useEffect } from "react";
import { useSearch } from "wouter";
import {
  Search, Loader2, RefreshCw,
  ArrowUpRight, Code2, FlaskConical, Filter, CalendarClock,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { SurfaceCard } from "@/components/ui/Surface";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Label } from "@/components/ui/label";
import { MetaMedium } from "@/components/ui/Typography";
import {
  MOCK_HISTORY,
  HISTORY_ACTION_LABEL,
  type HistoryRecord,
  type HistoryAction,
} from "./mockData";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import { StatusTag } from "@/components/ui/status-tag";
import { cn } from "@/lib/utils";
import CopyableId from "./components/CopyableId";

type ActionFilter = "all" | HistoryAction;
type OperatorFilter = "all" | "manual" | "auto";
type ResultFilter = "all" | "success" | "partial" | "failed";

const RESULT_FILTER_LABEL: Record<ResultFilter, string> = {
  all: "全部",
  success: "成功",
  partial: "部分成功",
  failed: "失败",
};

const SCHEDULED_TASK_STATUS_LABEL = {
  pending: "未开始",
  waiting: "待执行",
  running: "执行中",
  completed: "已完成",
  paused: "已暂停",
  canceled: "已取消",
  failed: "执行失败",
} as const;

function getScheduledFrequencyLabel(task: NonNullable<HistoryRecord["scheduledTask"]>) {
  if (task.frequency === "once") return "仅执行一次";
  if (task.frequency === "minutes") return `每 ${task.intervalMinutes ?? 5} 分钟`;
  if (task.frequency === "hourly") return `每 ${task.intervalHours ?? 1} 小时`;
  if (task.frequency === "daily") return "每天";
  if (task.frequency === "weekly") return "每周";
  return "每月";
}

interface Props {
  /** 记录范围：all=全部；agent-upgrade=仅 Agent 升级；command-execute=仅命令执行 */
  scope?: "all" | HistoryAction;
}

export default function HistoryTab({ scope = "all" }: Props) {
  const [search, setSearch] = useState("");
  // scope 锁定时，actionFilter 跟随 scope；scope=all 时由用户控制
  const [actionFilter, setActionFilter] = useState<ActionFilter>(scope === "all" ? "all" : scope);
  const [operatorFilter, setOperatorFilter] = useState<OperatorFilter>("all");
  const [resultFilter, setResultFilter] = useState<ResultFilter>("all");
  const [detailRecord, setDetailRecord] = useState<HistoryRecord | null>(null);

  // scope 变化时重置 actionFilter
  useEffect(() => {
    setActionFilter(scope === "all" ? "all" : scope);
  }, [scope]);

  // 支持从 URL ?action=command-execute 直接过滤（仅 scope=all 时生效，否则 scope 优先）
  const searchStr = useSearch();
  useEffect(() => {
    if (scope !== "all") return;
    const params = new URLSearchParams(searchStr);
    const a = params.get("action");
    if (a === "agent-upgrade" || a === "command-execute") {
      setActionFilter(a);
    }
  }, [searchStr, scope]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return MOCK_HISTORY.filter((r) => {
      // scope 锁定的优先级最高
      if (scope !== "all" && r.action !== scope) return false;
      // 执行记录只展示真实执行结果；未开始、待执行、暂停、取消、执行中的定时任务由「定时任务」承接。
      if (
        scope === "command-execute" &&
        r.scheduledTask &&
        !["completed", "failed"].includes(r.scheduledTask.status)
      ) {
        return false;
      }
      if (actionFilter !== "all" && r.action !== actionFilter) return false;
      if (operatorFilter === "manual" && r.isAuto) return false;
      if (operatorFilter === "auto" && !r.isAuto) return false;
      if (resultFilter !== "all" && getRecordResultKey(r) !== resultFilter) return false;
      if (q) {
        const hit =
          r.assetName.toLowerCase().includes(q) ||
          r.operator.toLowerCase().includes(q) ||
          r.id.toLowerCase().includes(q) ||
          r.taskId.toLowerCase().includes(q) ||
          (r.commandExtra?.commandContent ?? "").toLowerCase().includes(q);
        if (!hit) return false;
      }
      return true;
    });
  }, [search, actionFilter, operatorFilter, resultFilter, scope]);

  return (
    <div className="space-y-4">
      {/* ─── 标题栏：标题 + 计数 在左，筛选 + 搜索 + 刷新 在右 ────────── */}
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-semibold text-[#0A0A0A] text-base inline-flex items-center gap-2 shrink-0">
          执行记录
          <span className="text-xs font-normal text-[#737373] tabular-nums">
            共 {filtered.length} 条
          </span>
        </h2>
        <div className="flex items-center gap-3">
          {/* scope 锁定时隐藏类型下拉，避免误导用户该页可以切类型 */}
          {scope === "all" && (
            <Select value={actionFilter} onValueChange={(v) => setActionFilter(v as ActionFilter)}>
              <SelectTrigger className="h-9 w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类型</SelectItem>
                <SelectItem value="agent-upgrade">Agent 更新</SelectItem>
                <SelectItem value="command-execute">命令执行</SelectItem>
              </SelectContent>
            </Select>
          )}

          <Select value={operatorFilter} onValueChange={(v) => setOperatorFilter(v as OperatorFilter)}>
            <SelectTrigger className="h-9 w-[140px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部执行方</SelectItem>
              <SelectItem value="manual">管理员手动</SelectItem>
              <SelectItem value="auto">平台自动更新</SelectItem>
            </SelectContent>
          </Select>

          {/* 搜索框（与 Agent 列表主页同款样式） */}
          <div className="relative w-[320px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
            <Input
              placeholder="搜索内容、操作人、记录 ID 或命令"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 h-9"
            />
          </div>

          <Button variant="claw-outline" size="claw-square" title="刷新列表">
            <RefreshCw />
          </Button>
        </div>
      </div>

      {/* ─── 表格（与 Agent 列表主页同款 SurfaceCard 包裹） ──────── */}
      <SurfaceCard className="overflow-hidden">
        <Table variant="white">
          <TableHeader>
            <TableRow>
              <TableHead className="w-[14%]">任务 ID</TableHead>
              {scope === "all" && (
                <TableHead className="w-[10%] min-w-[110px]">类型</TableHead>
              )}
              <TableHead className="w-[28%]">运维内容</TableHead>
              <TableHead className="w-[12%]">执行方</TableHead>
              <TableHead className="w-[14%]">执行时间</TableHead>
              <TableHead className="w-[12%]">
                <div className="inline-flex items-center gap-1.5">
                  结果
                  <Popover>
                    <PopoverTrigger asChild>
                      <button
                        type="button"
                        aria-label="筛选结果"
                        className={cn(
                          "inline-flex h-6 w-6 items-center justify-center rounded-[4px] border transition-colors",
                          resultFilter === "all"
                            ? "border-transparent text-[#737373] hover:border-[#D4D4D4] hover:text-[#0A0A0A]"
                            : "border-[#1447E6] bg-[#1447E6]/5 text-[#1447E6]",
                        )}
                        title={resultFilter === "all" ? "筛选结果" : `当前筛选：${RESULT_FILTER_LABEL[resultFilter]}`}
                      >
                        <Filter className="h-3.5 w-3.5" />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent align="end" className="w-[180px] p-2">
                      <div className="space-y-1">
                        {(["all", "success", "partial", "failed"] as ResultFilter[]).map((item) => (
                          <button
                            key={item}
                            type="button"
                            onClick={() => setResultFilter(item)}
                            className={cn(
                              "flex h-8 w-full items-center justify-between rounded-[4px] px-2 text-sm text-[#0A0A0A] hover:bg-[#F5F7FA]",
                              resultFilter === item && "bg-[#1447E6]/5 text-[#1447E6]",
                            )}
                          >
                            <span>{RESULT_FILTER_LABEL[item]}</span>
                            {resultFilter === item && <span className="h-1.5 w-1.5 rounded-full bg-[#1447E6]" />}
                          </button>
                        ))}
                      </div>
                    </PopoverContent>
                  </Popover>
                </div>
              </TableHead>
              <TableHead className="w-[16%]">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={scope === "all" ? 7 : 6} className="px-6 py-16 text-center text-sm text-[var(--text-weak)]">
                  暂无符合条件的执行记录
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((r) => (
                <HistoryRow
                  key={r.id}
                  record={r}
                  showType={scope === "all"}
                  onDetail={() => setDetailRecord(r)}
                />
              ))
            )}
          </TableBody>
        </Table>
      </SurfaceCard>

      {/* 详情弹窗 */}
      <HistoryDetailDialog
        record={detailRecord}
        onClose={() => setDetailRecord(null)}
        showActionTag={scope === "all"}
      />
    </div>
  );
}

/** 执行记录结果：按这一轮命令在目标 Agent 上的执行结果聚合。 */
function getRecordResultKey(record: HistoryRecord): ResultFilter {
  const { successCount, failedCount } = record;
  if (failedCount === 0) return "success";
  if (successCount === 0) return "failed";
  return "partial";
}

function getOverallStatus(record: HistoryRecord): {
  label: string;
  variant: "green" | "red" | "orange" | "blue";
} {
  if (
    record.scheduledTask &&
    !["completed", "failed", "canceled"].includes(record.scheduledTask.status)
  ) {
    return { label: SCHEDULED_TASK_STATUS_LABEL[record.scheduledTask.status], variant: "blue" };
  }
  if (record.scheduledTask?.status === "canceled") return { label: "已取消", variant: "orange" };
  const { totalInstances, successCount, failedCount } = record;
  const running = totalInstances - successCount - failedCount;
  if (running > 0) return { label: "执行中", variant: "blue" };
  if (failedCount === 0) return { label: "成功", variant: "green" };
  if (successCount === 0) return { label: "失败", variant: "red" };
  return { label: "部分成功", variant: "orange" };
}

function HistoryRow({
  record,
  showType,
  onDetail,
}: {
  record: HistoryRecord;
  showType: boolean;
  onDetail: () => void;
}) {
  const successRate = record.totalInstances > 0
    ? Math.round((record.successCount / record.totalInstances) * 100)
    : 0;
  const overall = getOverallStatus(record);

  return (
    <TableRow>
      {/* 任务 ID（可复制） */}
      <TableCell>
        <CopyableId id={record.taskId} dark />
      </TableCell>

      {/* 类型（仅 scope=all 显示） */}
      {showType && (
        <TableCell>
          <StatusTag
            mode="fill"
            variant={record.action === "agent-upgrade" ? "blue" : "purple"}
            icon={record.action === "command-execute" ? <Code2 /> : <ArrowUpRight />}
          >
            {HISTORY_ACTION_LABEL[record.action]}
          </StatusTag>
        </TableCell>
      )}

      {/* 运维内容（按类型动态展示） */}
      <TableCell>
        {record.action === "agent-upgrade" && (
          <div className="space-y-0.5">
            <div className="text-sm text-[#0A0A0A] truncate max-w-[280px]">{record.assetName}</div>
            {record.fromVersion && record.toVersion ? (
              <div className="text-xs font-mono text-[#737373] tabular-nums">
                {record.fromVersion} <span className="text-[#A3A3A3]">→</span>{" "}
                <span className="text-[#334155] font-semibold">{record.toVersion}</span>
              </div>
            ) : record.toVersion ? (
              <div className="text-xs font-mono text-[#737373] tabular-nums">
                → <span className="text-[#334155] font-semibold">{record.toVersion}</span>
              </div>
            ) : null}
          </div>
        )}
        {record.action === "command-execute" && (
          <div className="space-y-0.5">
            <div className="text-sm text-[#0A0A0A] inline-flex items-center gap-1.5 max-w-[280px]">
              <span className="truncate">{record.assetName}</span>
              {record.scheduledTask && (
                <StatusTag
                  mode="fill"
                  variant="blue"
                  icon={<CalendarClock className="w-3 h-3" />}
                  className="shrink-0"
                >
                  定时
                </StatusTag>
              )}
              {record.commandExtra?.testInstanceId && (
                <StatusTag
                  mode="fill"
                  variant="orange"
                  icon={<FlaskConical />}
                  className="shrink-0"
                >
                  灰度执行
                </StatusTag>
              )}
            </div>
            {record.commandExtra?.commandContent && (
              <code className="text-xs font-mono text-[#737373] truncate block max-w-[280px]">
                {record.commandExtra.commandContent.split("\n")[0]}
                {record.commandExtra.commandContent.includes("\n") && (
                  <span className="text-[#A3A3A3] ml-1">…</span>
                )}
              </code>
            )}
            {record.scheduledTask && (
              <div className="text-xs text-[#737373] tabular-nums">
                定时 · {getScheduledFrequencyLabel(record.scheduledTask)}
                {record.scheduledTask.nextExecuteAt ? ` · 下次：${record.scheduledTask.nextExecuteAt}` : ""}
              </div>
            )}
          </div>
        )}
      </TableCell>

      {/* 执行方 */}
      <TableCell>
        {record.isAuto ? (
          <span className="text-sm text-[#1447E6]">{record.operator}</span>
        ) : (
          <span className="text-sm text-[#0A0A0A] truncate max-w-[160px] inline-block align-middle">
            {record.operator}
          </span>
        )}
      </TableCell>

      {/* 执行时间 */}
      <TableCell>
        <div className="text-sm text-[#0A0A0A] tabular-nums whitespace-nowrap">
          {record.operatedAt}
        </div>
        {record.scheduledAt && (
          <div className="text-xs text-[#A3A3A3] tabular-nums mt-0.5">
            计划：{record.scheduledAt}
          </div>
        )}
      </TableCell>

      {/* 结果（综合状态徽标 + 进度条 + 数字） */}
      <TableCell>
        <div className="space-y-1.5">
          <StatusTag mode="text" variant={overall.variant}>
            {overall.label}
          </StatusTag>
          <div className="flex items-center gap-2">
            <div className="flex-1 min-w-[60px] h-1.5 bg-gray-100 rounded-full overflow-hidden">
              <div
                className={cn(
                  "h-1.5 rounded-full transition-all",
                  // 与状态徽标同色相，避免"绿色徽标 + 黄色进度条"的视觉错配
                  overall.variant === "green" && "bg-[#008236]",
                  overall.variant === "orange" && "bg-[#F59E0B]",
                  overall.variant === "red" && "bg-[#DC2626]",
                  overall.variant === "blue" && "bg-[#1447E6]",
                )}
                style={{ width: `${successRate}%` }}
              />
            </div>
            <span className="text-xs text-gray-600 font-mono tabular-nums shrink-0 whitespace-nowrap">
              {record.successCount}/{record.totalInstances}
            </span>
          </div>
        </div>
      </TableCell>

      {/* 操作 */}
      <TableActionCell>
        <Button variant="link" onClick={onDetail}>
          详情
        </Button>
      </TableActionCell>
    </TableRow>
  );
}

// ────────────────────────────────────────────────────────────────
// 详情弹窗（按 action 类型分发渲染）— 项目规范的普通弹窗
// ────────────────────────────────────────────────────────────────
export function HistoryDetailDialog({
  record,
  onClose,
  showActionTag = false,
}: {
  record: HistoryRecord | null;
  onClose: () => void;
  /** 仅当列表 scope=all（混合多种类型）时显示标题右侧的类型 tag */
  showActionTag?: boolean;
}) {
  if (!record) return null;
  const actionVariant = record.action === "agent-upgrade" ? "blue" : "purple";
  const overall = getOverallStatus(record);
  return (
    <Dialog open={!!record} onOpenChange={onClose}>
      <DialogContent
        size="xl"
        style={{ maxHeight: "min(85vh, 880px)", display: "flex", flexDirection: "column" }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 flex-wrap">
            <span>{record.assetName}</span>
            <StatusTag mode="fill" variant={overall.variant} className="shrink-0">
              {overall.label}
            </StatusTag>
            {showActionTag && (
              <StatusTag mode="fill" variant={actionVariant} className="shrink-0">
                {HISTORY_ACTION_LABEL[record.action]}
              </StatusTag>
            )}
          </DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          <div className="space-y-4">
            {/* 执行信息（小标题在卡片上方） */}
            <div>
              <div className="text-sm font-medium text-[#0A0A0A] mb-2">执行信息</div>
              <SurfaceCard className="grid grid-cols-4 gap-x-6 gap-y-4 p-4">
                <Field label="任务 ID" value={record.taskId} mono />
                <Field label="记录 ID" value={record.id} mono />
                <Field label="执行方" value={record.isAuto ? `${record.operator}（自动）` : record.operator} />
                <Field label="执行时间" value={record.operatedAt} />
                <Field
                  label="执行结果"
                  value={overall.label}
                  highlight={overall.variant === "green" ? "success" : overall.variant === "red" ? "error" : undefined}
                />
                {record.scheduledAt && <Field label="计划执行时间" value={record.scheduledAt} />}
                {record.scheduledTask && (
                  <>
                    <Field label="任务名称" value={record.scheduledTask.taskName} />
                    <Field label="执行频率" value={getScheduledFrequencyLabel(record.scheduledTask)} />
                    {record.scheduledTask.nextExecuteAt && <Field label="下次执行时间" value={record.scheduledTask.nextExecuteAt} />}
                    <Field label="创建人" value={record.scheduledTask.createdBy} />
                  </>
                )}
                <Field label="影响 Agent" value={`${record.totalInstances} 个`} />
                <Field label="成功 Agent" value={`${record.successCount}`} highlight="success" />
                <Field
                  label="失败 Agent"
                  value={record.failedCount > 0 ? `${record.failedCount}` : "0"}
                  highlight={record.failedCount > 0 ? "error" : undefined}
                />
                {record.action === "agent-upgrade" && record.fromVersion && (
                  <Field label="原版本" value={record.fromVersion} mono />
                )}
                {record.action === "agent-upgrade" && record.toVersion && (
                  <Field label="目标版本" value={record.toVersion} mono />
                )}
              </SurfaceCard>
              {record.scheduledTask?.remark && (
                <Alert variant="info" className="mt-3">
                  <AlertDescription>
                    任务备注：{record.scheduledTask.remark}
                  </AlertDescription>
                </Alert>
              )}
            </div>

            {/* 命令执行专属：命令详情（小标题在卡片上方） */}
            {record.action === "command-execute" && record.commandExtra && (
              <div>
                <div className="text-sm font-medium text-[#0A0A0A] mb-2">命令详情</div>
                <SurfaceCard className="p-4 space-y-4">
                  <div className="grid grid-cols-4 gap-x-6 gap-y-4">
                    <Field label="命令类型" value={record.commandExtra.commandType} />
                    <Field label="执行用户" value={record.commandExtra.runAsUser} mono />
                    <Field label="执行路径" value={record.commandExtra.workingDir} mono />
                    <Field label="超时时间" value={`${record.commandExtra.timeoutSec} 秒`} />
                  </div>
                  <div>
                    <MetaMedium as="label" tone="secondary" className="block mb-1">命令内容</MetaMedium>
                    <pre className="text-xs font-mono text-[#0A0A0A] bg-[#FAFAFA] rounded-[4px] p-3 max-h-[180px] overflow-auto whitespace-pre-wrap break-all border border-[#E5E5E5]">
                      {record.commandExtra.commandContent}
                    </pre>
                  </div>

                  {/* 命令参数值（仅本次任务用到了参数化命令时显示） */}
                  {record.commandExtra.paramValues && Object.keys(record.commandExtra.paramValues).length > 0 && (
                    <div>
                      <MetaMedium as="label" tone="secondary" className="block mb-1">命令参数</MetaMedium>
                      <SurfaceCard className="overflow-hidden">
                        <Table density="compact" autoFixedColumns={false}>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-[36%]">参数名</TableHead>
                              <TableHead>参数值</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {Object.entries(record.commandExtra.paramValues).map(([k, v]) => (
                              <TableRow key={k}>
                                <TableCell className="font-mono text-[#0A0A0A] break-all">{k}</TableCell>
                                <TableCell className="font-mono text-[#525252] break-all">{v}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </SurfaceCard>
                      {record.commandExtra.commandContentTemplate && (
                        <details className="text-xs mt-2">
                          <summary className="text-[#737373] cursor-pointer hover:text-[#1447E6]">
                            查看命令模板原始内容（含 {"{{key}}"}）
                          </summary>
                          <pre className="text-xs font-mono text-[#737373] bg-[#FAFAFA] rounded-[4px] p-2 mt-1.5 max-h-[120px] overflow-auto whitespace-pre-wrap break-all border border-[#E5E5E5]">
                            {record.commandExtra.commandContentTemplate}
                          </pre>
                        </details>
                      )}
                    </div>
                  )}

                  {/* 灰度执行结果 — 规范 Alert（一行展示） */}
                  {record.commandExtra.testInstanceId && (
                    <Alert
                      variant={record.commandExtra.testStatus === "success" ? "warning" : "error"}
                    >
                      <FlaskConical />
                      <AlertDescription>
                        <span className="font-semibold">
                          灰度机验证{record.commandExtra.testStatus === "success" ? "通过" : "失败"}
                        </span>
                        {record.commandExtra.testMessage ? `：${record.commandExtra.testMessage}` : ""}
                      </AlertDescription>
                    </Alert>
                  )}
                </SurfaceCard>
              </div>
            )}

            {/* 每台 Agent 的结果 */}
            {record.perInstanceResult && record.perInstanceResult.length > 0 && (
              <div>
                <div className="text-sm font-medium text-[#0A0A0A] mb-2">
                  每个 Agent 执行结果（{record.perInstanceResult.length}）
                </div>
                <SurfaceCard className="overflow-hidden">
                  <Table density="compact" autoFixedColumns={false}>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Agent</TableHead>
                        <TableHead className="w-[16%]">状态</TableHead>
                        {record.action === "command-execute" ? (
                          <>
                            <TableHead className="w-[10%]">退出码</TableHead>
                            <TableHead className="w-[10%]">耗时</TableHead>
                            <TableHead>输出</TableHead>
                          </>
                        ) : (
                          <TableHead>备注</TableHead>
                        )}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {record.perInstanceResult.map((r) => (
                        <TableRow key={r.instanceId}>
                          <TableCell>
                            <div className="text-sm text-[#0A0A0A]">{r.instanceName}</div>
                            <div className="text-xs text-[#A3A3A3] font-mono mt-0.5">{r.instanceId}</div>
                          </TableCell>
                          <TableCell>
                            {r.status === "success" && (
                              <StatusTag mode="text" variant="green">成功</StatusTag>
                            )}
                            {r.status === "failed" && (
                              <StatusTag mode="text" variant="red">失败</StatusTag>
                            )}
                            {r.status === "running" && (
                              <StatusTag mode="text" variant="blue" icon={<Loader2 className="animate-spin" />}>
                                进行中
                              </StatusTag>
                            )}
                          </TableCell>
                          {record.action === "command-execute" ? (
                            <>
                              <TableCell className="text-sm text-[#0A0A0A] font-mono tabular-nums">
                                {r.exitCode ?? "—"}
                              </TableCell>
                              <TableCell className="text-sm text-[#0A0A0A] tabular-nums">
                                {r.durationMs ? `${(r.durationMs / 1000).toFixed(2)}s` : "—"}
                              </TableCell>
                              <TableCell className="text-sm text-[#525252]">
                                {r.stderr ? (
                                  <code className="text-xs text-red-600 font-mono break-all">{r.stderr}</code>
                                ) : r.stdout ? (
                                  <code className="text-xs text-[#525252] font-mono break-all">{r.stdout}</code>
                                ) : (
                                  "—"
                                )}
                              </TableCell>
                            </>
                          ) : (
                            <TableCell className="text-sm text-[#525252]">{r.message || "—"}</TableCell>
                          )}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </SurfaceCard>
              </div>
            )}
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="dialog-confirm" onClick={onClose}>
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Field({
  label, value, mono, highlight,
}: {
  label: string;
  value: string;
  mono?: boolean;
  highlight?: "success" | "error";
}) {
  const valueColor =
    highlight === "success" ? "text-green-600"
      : highlight === "error" ? "text-red-600"
      : "text-[#0A0A0A]";
  return (
    <div className="min-w-0">
      <MetaMedium as="label" tone="secondary" className="block mb-1">{label}</MetaMedium>
      <div className={`text-sm font-medium truncate ${valueColor} ${mono ? "font-mono tabular-nums" : ""}`}>
        {value}
      </div>
    </div>
  );
}
