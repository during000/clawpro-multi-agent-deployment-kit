import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
  Ban, CalendarClock, Filter, Pause, Play, RefreshCw, Search,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog, DialogBody, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { SurfaceCard } from "@/components/ui/Surface";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Table, TableActionCell, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import CopyableId from "./components/CopyableId";
import {
  MOCK_HISTORY,
  type HistoryRecord,
  type ScheduledFrequency,
  type ScheduledTaskStatus,
} from "./mockData";

type VisibleTaskStatus = "pending" | "waiting" | "running" | "paused" | "completed";
type StatusFilter = "all" | VisibleTaskStatus;

const VISIBLE_TASK_STATUSES: VisibleTaskStatus[] = ["pending", "waiting", "running", "paused", "completed"];

const STATUS_LABEL: Record<StatusFilter, string> = {
  all: "全部状态",
  pending: "未开始",
  waiting: "待执行",
  running: "执行中",
  completed: "已完成",
  paused: "已暂停",
};

const FREQUENCY_LABEL: Record<ScheduledFrequency, string> = {
  once: "仅执行一次",
  minutes: "每分钟",
  hourly: "每小时",
  daily: "每天",
  weekly: "每周",
  monthly: "每月",
};

function getFrequencyLabel(task: HistoryRecord["scheduledTask"]) {
  if (!task) return "—";
  if (task.frequency === "minutes") return `每 ${task.intervalMinutes ?? 5} 分钟`;
  if (task.frequency === "hourly") return `每 ${task.intervalHours ?? 1} 小时`;
  return FREQUENCY_LABEL[task.frequency];
}

function getTaskStatusLabel(status: ScheduledTaskStatus) {
  if ((VISIBLE_TASK_STATUSES as ScheduledTaskStatus[]).includes(status)) {
    return STATUS_LABEL[status as VisibleTaskStatus];
  }
  if (status === "canceled") return "已取消";
  return "执行失败";
}

function getTaskStatusVariant(status: ScheduledTaskStatus): "green" | "red" | "orange" | "blue" {
  if (status === "completed") return "green";
  if (status === "failed") return "red";
  if (status === "running" || status === "pending" || status === "waiting") return "blue";
  return "orange";
}

function getScheduleTimeLines(task: HistoryRecord["scheduledTask"]) {
  if (!task) {
    return { primary: "—", secondary: "—", tertiary: undefined as string | undefined };
  }

  const primary = task.nextExecuteAt
    ? `${task.nextExecuteAt === task.firstExecuteAt ? "首次" : "下次"}：${task.nextExecuteAt}`
    : `计划：${task.executeAt}`;

  return {
    primary,
    secondary: task.nextExecuteAt && task.nextExecuteAt !== task.firstExecuteAt
      ? `首次：${task.firstExecuteAt}`
      : undefined,
    tertiary: getFrequencyLabel(task),
  };
}

function getScheduledTaskBaseId(taskId: string) {
  return taskId.split("-RUN-")[0];
}

function getRecordTime(record: HistoryRecord) {
  return record.operatedAt.startsWith("—") ? (record.scheduledAt ?? "") : record.operatedAt;
}

function getExecutionRecords(record: HistoryRecord) {
  const baseId = getScheduledTaskBaseId(record.taskId);
  const runs = MOCK_HISTORY
    .filter((item) => (
      item.id !== record.id &&
      item.action === "command-execute" &&
      item.scheduledTask &&
      ["completed", "failed"].includes(item.scheduledTask.status) &&
      getScheduledTaskBaseId(item.taskId) === baseId &&
      item.taskId.includes("-RUN-")
    ))
    .sort((a, b) => getRecordTime(b).localeCompare(getRecordTime(a)));

  if (runs.length > 0) return runs;
  if (["running", "completed", "failed"].includes(record.scheduledTask?.status ?? "")) return [record];
  return [];
}

function getExecutionStatus(record: HistoryRecord): {
  label: "成功" | "失败" | "部分成功" | "执行中";
  variant: "green" | "red" | "orange" | "blue";
} {
  const running = Math.max(record.totalInstances - record.successCount - record.failedCount, 0);
  if (running > 0) return { label: "执行中", variant: "blue" };
  if (record.failedCount === 0) return { label: "成功", variant: "green" };
  if (record.successCount === 0) return { label: "失败", variant: "red" };
  return { label: "部分成功", variant: "orange" };
}

function canPause(record: HistoryRecord) {
  return record.scheduledTask
    ? ["pending", "waiting"].includes(record.scheduledTask.status) && record.scheduledTask.frequency !== "once"
    : false;
}

function canResume(record: HistoryRecord) {
  return record.scheduledTask?.status === "paused";
}

function canCancel(record: HistoryRecord) {
  return record.scheduledTask ? ["pending", "waiting", "running", "paused"].includes(record.scheduledTask.status) : false;
}

export default function ScheduledTaskManagementTab() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [detailRecord, setDetailRecord] = useState<HistoryRecord | null>(null);
  const [cancelTarget, setCancelTarget] = useState<HistoryRecord | null>(null);
  const [tick, setTick] = useState(0);

  const scheduledTasks = useMemo(() => {
    const q = search.trim().toLowerCase();
    return MOCK_HISTORY.filter((record) => {
      if (record.action !== "command-execute" || !record.scheduledTask) return false;
      if (record.taskId.includes("-RUN-")) return false;
      if (!(VISIBLE_TASK_STATUSES as ScheduledTaskStatus[]).includes(record.scheduledTask.status)) return false;
      if (statusFilter !== "all" && record.scheduledTask.status !== statusFilter) return false;
      if (!q) return true;
      return (
        record.taskId.toLowerCase().includes(q) ||
        record.assetName.toLowerCase().includes(q) ||
        record.operator.toLowerCase().includes(q) ||
        record.scheduledTask.taskName.toLowerCase().includes(q) ||
        (record.commandExtra?.commandName ?? "").toLowerCase().includes(q) ||
        (record.commandExtra?.commandContent ?? "").toLowerCase().includes(q)
      );
    });
  }, [search, statusFilter, tick]);

  const updateTaskStatus = (record: HistoryRecord, status: ScheduledTaskStatus) => {
    if (!record.scheduledTask) return;
    record.scheduledTask.status = status;

    if (status === "paused") {
      record.operatedAt = "—（已暂停）";
      toast.success(`定时任务「${record.scheduledTask.taskName}」已暂停`);
    }
    if (status === "pending") {
      record.operatedAt = "—（未开始）";
      toast.success(`定时任务「${record.scheduledTask.taskName}」已恢复`);
    }
    if (status === "waiting") {
      record.operatedAt = "—（待执行）";
      toast.success(`定时任务「${record.scheduledTask.taskName}」已恢复`);
    }
    if (status === "canceled") {
      record.operatedAt = "—（已取消）";
      record.scheduledTask.nextExecuteAt = undefined;
      toast.success(`定时任务「${record.scheduledTask.taskName}」已取消`);
    }
    setTick((value) => value + 1);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-semibold text-[#0A0A0A] text-base inline-flex items-center gap-2 shrink-0">
          定时任务管理
          <span className="text-xs font-normal text-[#737373] tabular-nums">
            共 {scheduledTasks.length} 条
          </span>
        </h2>

        <div className="flex items-center gap-3">
          <div className="relative w-[320px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
            <Input
              placeholder="搜索任务名、命令、定时任务 ID 或创建人"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              className="pl-9 h-9"
            />
          </div>

          <Button
            variant="claw-outline"
            size="claw-square"
            title="刷新列表"
            onClick={() => setTick((value) => value + 1)}
          >
            <RefreshCw />
          </Button>
        </div>
      </div>

      <SurfaceCard className="overflow-hidden">
        <Table variant="white">
          <TableHeader>
            <TableRow>
              <TableHead className="w-[16%]">定时任务 ID</TableHead>
              <TableHead className="w-[34%]">任务内容</TableHead>
              <TableHead className="w-[12%]">执行方</TableHead>
              <TableHead className="w-[20%]">执行计划</TableHead>
              <TableHead className="w-[10%]">
                <div className="inline-flex items-center gap-1.5">
                  任务状态
                  <Popover>
                    <PopoverTrigger asChild>
                      <button
                        type="button"
                        aria-label="筛选任务状态"
                        className={cn(
                          "inline-flex h-6 w-6 items-center justify-center rounded-[4px] border transition-colors",
                          statusFilter === "all"
                            ? "border-transparent text-[#737373] hover:border-[#D4D4D4] hover:text-[#0A0A0A]"
                            : "border-[#1447E6] bg-[#1447E6]/5 text-[#1447E6]",
                        )}
                        title={statusFilter === "all" ? "筛选任务状态" : `当前筛选：${STATUS_LABEL[statusFilter]}`}
                      >
                        <Filter className="h-3.5 w-3.5" />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent align="end" className="w-[180px] p-2">
                      <div className="space-y-1">
                        {(["all", ...VISIBLE_TASK_STATUSES] as StatusFilter[]).map((item) => (
                          <button
                            key={item}
                            type="button"
                            onClick={() => setStatusFilter(item)}
                            className={cn(
                              "flex h-8 w-full items-center justify-between rounded-[4px] px-2 text-sm text-[#0A0A0A] hover:bg-[#F5F7FA]",
                              statusFilter === item && "bg-[#1447E6]/5 text-[#1447E6]",
                            )}
                          >
                            <span>{STATUS_LABEL[item]}</span>
                            {statusFilter === item && <span className="h-1.5 w-1.5 rounded-full bg-[#1447E6]" />}
                          </button>
                        ))}
                      </div>
                    </PopoverContent>
                  </Popover>
                </div>
              </TableHead>
              <TableHead className="w-[14%]">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {scheduledTasks.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="px-6 py-16 text-center text-sm text-[var(--text-weak)]">
                  暂无符合条件的定时任务
                </TableCell>
              </TableRow>
            ) : (
              scheduledTasks.map((record) => (
                <ScheduledTaskRow
                  key={record.id}
                  record={record}
                  onDetail={() => setDetailRecord(record)}
                  onCancel={() => setCancelTarget(record)}
                  onStatusChange={(status) => updateTaskStatus(record, status)}
                />
              ))
            )}
          </TableBody>
        </Table>
      </SurfaceCard>

      <ScheduledTaskDetailDialog
        record={detailRecord}
        onClose={() => setDetailRecord(null)}
      />
      <AlertDialog open={!!cancelTarget} onOpenChange={(open) => !open && setCancelTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认取消定时任务？</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            {cancelTarget?.scheduledTask
              ? `取消后，定时任务「${cancelTarget.scheduledTask.taskName}」将不再自动触发，列表中也不再展示该任务。`
              : "取消后，该定时任务将不再自动触发。"}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>再想想</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (cancelTarget) updateTaskStatus(cancelTarget, "canceled");
                setCancelTarget(null);
              }}
            >
              确认取消
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function ScheduledTaskRow({
  record,
  onDetail,
  onCancel,
  onStatusChange,
}: {
  record: HistoryRecord;
  onDetail: () => void;
  onCancel: () => void;
  onStatusChange: (status: ScheduledTaskStatus) => void;
}) {
  const task = record.scheduledTask!;
  const scheduleTime = getScheduleTimeLines(task);

  return (
    <TableRow>
      <TableCell>
        <CopyableId id={record.taskId} dark />
      </TableCell>

      <TableCell>
        <div className="space-y-0.5">
          <div className="text-sm text-[#0A0A0A] inline-flex items-center gap-1.5 max-w-[280px]">
            <span className="truncate">{task.taskName}</span>
            <StatusTag mode="fill" variant="blue" icon={<CalendarClock className="w-3 h-3" />} className="shrink-0">
              定时
            </StatusTag>
          </div>
          <code className="text-xs font-mono text-[#737373] truncate block max-w-[280px]">
            {record.commandExtra?.commandContent.split("\n")[0] ?? record.commandExtra?.commandName ?? "—"}
          </code>
        </div>
      </TableCell>

      <TableCell>
        <span className="text-sm text-[#0A0A0A] truncate max-w-[160px] inline-block align-middle">
          {task.createdBy}
        </span>
      </TableCell>

      <TableCell>
        <div className="grid min-h-[58px] content-center gap-0.5">
          <div className="text-sm leading-5 text-[#0A0A0A] tabular-nums whitespace-nowrap">
            {scheduleTime.primary}
          </div>
          <div className={cn(
            "text-sm leading-5 text-[#0A0A0A] tabular-nums whitespace-nowrap",
            !scheduleTime.secondary && "invisible",
          )}>
            {scheduleTime.secondary}
          </div>
          <div className="text-xs leading-4 text-[#A3A3A3]">
            {scheduleTime.tertiary}
          </div>
        </div>
      </TableCell>

      <TableCell>
        <div className="space-y-1">
          <StatusTag mode="text" variant={getTaskStatusVariant(task.status)}>
            {getTaskStatusLabel(task.status)}
          </StatusTag>
        </div>
      </TableCell>

      <TableActionCell>
        {canPause(record) && (
          <Button variant="link" onClick={() => onStatusChange("paused")}>
            <Pause className="w-3.5 h-3.5 mr-1" />
            暂停
          </Button>
        )}
        {canResume(record) && (
          <Button
            variant="link"
            onClick={() => onStatusChange(task.frequency === "once" || task.nextExecuteAt === task.firstExecuteAt ? "pending" : "waiting")}
          >
            <Play className="w-3.5 h-3.5 mr-1" />
            恢复
          </Button>
        )}
        {canCancel(record) && (
          <Button variant="link" onClick={onCancel}>
            <Ban className="w-3.5 h-3.5 mr-1" />
            取消
          </Button>
        )}
        <Button variant="link" onClick={onDetail}>
          详情
        </Button>
      </TableActionCell>
    </TableRow>
  );
}

function ScheduledTaskDetailDialog({
  record,
  onClose,
}: {
  record: HistoryRecord | null;
  onClose: () => void;
}) {
  if (!record?.scheduledTask) return null;

  const task = record.scheduledTask;
  const executionRecords = getExecutionRecords(record);

  return (
    <Dialog open={!!record} onOpenChange={onClose}>
      <DialogContent
        size="xl"
        style={{ maxHeight: "min(85vh, 880px)", display: "flex", flexDirection: "column" }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span>{task.taskName}</span>
            <StatusTag mode="fill" variant={getTaskStatusVariant(task.status)}>
              {getTaskStatusLabel(task.status)}
            </StatusTag>
          </DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          <div className="space-y-4">
            <SurfaceCard className="grid grid-cols-4 gap-x-6 gap-y-4 p-4">
              <DetailField label="定时任务 ID" value={record.taskId} mono />
              <DetailField label="执行频率" value={getFrequencyLabel(task)} />
              <DetailField label="创建人" value={task.createdBy} />
              <DetailField label="创建时间" value={task.createdAt} />
              <DetailField label="首次执行时间" value={task.firstExecuteAt} />
              <DetailField label="下次执行时间" value={task.nextExecuteAt ?? "—"} />
              <DetailField label="影响 Agent" value={`${record.totalInstances} 个`} />
              <DetailField label="任务状态" value={getTaskStatusLabel(task.status)} />
            </SurfaceCard>

            {record.commandExtra && (
              <SurfaceCard className="p-4 space-y-3">
                <div className="grid grid-cols-4 gap-x-6 gap-y-4">
                  <DetailField label="命令类型" value={record.commandExtra.commandType} />
                  <DetailField label="执行用户" value={record.commandExtra.runAsUser} mono />
                  <DetailField label="执行路径" value={record.commandExtra.workingDir} mono />
                  <DetailField label="超时时间" value={`${record.commandExtra.timeoutSec} 秒`} />
                </div>
                <div>
                  <div className="text-xs text-[#737373] mb-1">命令内容</div>
                  <pre className="text-xs font-mono text-[#0A0A0A] bg-[#FAFAFA] rounded-[4px] p-3 max-h-[140px] overflow-auto whitespace-pre-wrap break-all border border-[#E5E5E5]">
                    {record.commandExtra.commandContent}
                  </pre>
                </div>
              </SurfaceCard>
            )}

            <div>
              <div className="text-sm font-medium text-[#0A0A0A] mb-2">
                每次任务执行结果（{executionRecords.length}）
              </div>
              <SurfaceCard className="overflow-hidden">
                <Table density="compact" autoFixedColumns={false}>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[28%]">执行记录 ID</TableHead>
                      <TableHead className="w-[20%]">触发时间</TableHead>
                      <TableHead className="w-[16%]">执行结果</TableHead>
                      <TableHead className="w-[12%]">执行对象</TableHead>
                      <TableHead>结果统计</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {executionRecords.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className="px-6 py-10 text-center text-sm text-[var(--text-weak)]">
                          暂无执行记录
                        </TableCell>
                      </TableRow>
                    ) : (
                      executionRecords.map((execution) => {
                        const status = getExecutionStatus(execution);
                        return (
                          <TableRow key={execution.id}>
                            <TableCell>
                              <CopyableId id={execution.taskId} dark />
                            </TableCell>
                            <TableCell className="text-sm text-[#0A0A0A] tabular-nums">
                              {execution.operatedAt}
                            </TableCell>
                            <TableCell>
                              <StatusTag mode="text" variant={status.variant}>
                                {status.label}
                              </StatusTag>
                            </TableCell>
                            <TableCell className="text-sm text-[#0A0A0A] tabular-nums">
                              {execution.totalInstances} 个
                            </TableCell>
                            <TableCell className="text-sm text-[#525252] tabular-nums">
                              成功 {execution.successCount} / 失败 {execution.failedCount}
                            </TableCell>
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              </SurfaceCard>
            </div>
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

function DetailField({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-[#737373] mb-1">{label}</div>
      <div className={cn("text-sm font-medium text-[#0A0A0A] truncate", mono && "font-mono tabular-nums")}>
        {value}
      </div>
    </div>
  );
}
