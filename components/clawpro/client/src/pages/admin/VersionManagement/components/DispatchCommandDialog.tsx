/**
 * DispatchCommandDialog - 命令下发执行弹窗（步骤式）
 *
 * 整体架构：单一弹窗内的 4 阶段状态机
 *
 *   ┌─ Step 1 ─┐    ┌─ Step 2 ─┐    ┌─ Step 3 ─┐
 *   │ 选命令   │ →  │ 选执行对象│ →  │ 执行策略 │
 *   └──────────┘    └──────────┘    └────┬─────┘
 *                                        │ 启用「灰度执行」？
 *                          ┌─────────────┴────────────┐
 *                          │ 是                       │ 否
 *                          ▼                          ▼
 *                    ┌──────────┐               ┌──────────┐
 *                    │ testing  │               │submitting│
 *                    └────┬─────┘               └──────────┘
 *                         │                          │
 *                         ▼                          ▼
 *                    ┌──────────┐               toast.success
 *                    │  review  │ → 终止/继续 → submitting
 *                    └──────────┘
 *
 * 两种打开方式：
 *   A. 先选命令 → 选实例：传 command，初始 step = 2
 *   B. 先选实例 → 选命令：传 presetInstanceIds、command=null，初始 step = 1
 */
import { useState, useMemo, useEffect } from "react";
import { toast } from "sonner";
import {
  FlaskConical, Server, Code2, Search,
  Loader2, CheckCircle2, XCircle, ArrowRight, X as XIcon,
  Info, CircleAlert, AlertCircle, ChevronDown, ChevronUp,
  CalendarClock, Clock3,
} from "lucide-react";
import {
  Dialog, DialogContent, DialogBody, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from "@/components/ui/dialog";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { DatePicker } from "@/components/ui/date-picker";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Stepper as UiStepper } from "@/components/ui/stepper";
import { RadioGroup } from "@/components/ui/radio-group";
import { RadioCard } from "@/components/ui/radio-card";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { StatusTag } from "@/components/ui/status-tag";
import { BodyMedium, CodeText, HelperText, MetaMedium, MetaText } from "@/components/ui/Typography";
import {
  Collapsible, CollapsibleContent,
} from "@/components/ui/collapsible";
import {
  MOCK_INSTANCES, AGENT_TYPE_LABEL, MOCK_HISTORY, MOCK_COMMAND_TEMPLATES, detectDangerousCommand,
  type CommandTemplate, type AgentTypeKey, type HistoryRecord, type ScheduledFrequency,
} from "../mockData";

type Phase = "prepare" | "testing" | "review" | "submitting";
type StepId = 1 | 2 | 3;
type ExecutionMode = "immediate" | "scheduled";

interface TestRunResult {
  status: "success" | "failed";
  stdout: string;
  stderr?: string;
  exitCode: number;
  durationMs: number;
}

interface Props {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  /** 命令模板。若为 null，弹窗会从 Step 1 开始让用户挑选 */
  command: CommandTemplate | null;
  /** 预选实例 ID 列表（来自 Agent 列表勾选） */
  presetInstanceIds?: string[];
  onDispatched?: (record: HistoryRecord) => void;
}

// 命令下发场景下展示的 Agent 类型筛选项（剔除自研 MyAgent，只保留 3 大类标准 Agent）
const AGENT_TYPES: AgentTypeKey[] = ["OpenClaw", "Hermes", "LightclawACE"];
const INSTANCE_PAGE_SIZE = 200;
const DEFAULT_MINUTE_INTERVAL = "5";
const DEFAULT_HOUR_INTERVAL = "1";

const STEP_DEFS: { id: StepId; label: string; desc: string }[] = [
  { id: 1, label: "选命令",     desc: "" },
  { id: 2, label: "选执行对象", desc: "" },
  { id: 3, label: "执行策略",   desc: "" },
];

const pad2 = (n: number) => String(n).padStart(2, "0");
const WEEKDAY_OPTIONS = [
  { value: "1", label: "周一" },
  { value: "2", label: "周二" },
  { value: "3", label: "周三" },
  { value: "4", label: "周四" },
  { value: "5", label: "周五" },
  { value: "6", label: "周六" },
  { value: "7", label: "周日" },
];
const MONTH_DAY_OPTIONS = Array.from({ length: 31 }, (_, i) => String(i + 1));

function formatScheduleName(date: Date) {
  return `定时命令任务-${date.getFullYear()}${pad2(date.getMonth() + 1)}${pad2(date.getDate())}-${pad2(date.getHours())}${pad2(date.getMinutes())}`;
}

function formatDateTime(value: string) {
  if (!value) return "";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

function getDateInputValue(value: string) {
  return value.includes("T") ? value.split("T")[0] : "";
}

function getTimeInputValue(value: string) {
  return value.includes("T") ? value.split("T")[1]?.slice(0, 5) ?? "" : "";
}

function getTodayInputValue() {
  const now = new Date();
  return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
}

function getCurrentWeekdayValue() {
  const day = new Date().getDay();
  return String(day === 0 ? 7 : day);
}

function toDateTimeInputValue(date: Date) {
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}T${pad2(date.getHours())}:${pad2(date.getMinutes())}`;
}

function applyClock(date: Date, clock: string) {
  const [h, m] = clock.split(":").map((v) => Number(v));
  const next = new Date(date);
  next.setHours(Number.isFinite(h) ? h : 0, Number.isFinite(m) ? m : 0, 0, 0);
  return next;
}

function getLastDayOfMonth(year: number, monthIndex: number) {
  return new Date(year, monthIndex + 1, 0).getDate();
}

function getNextDailySchedule(clock: string) {
  if (!clock) return "";
  const now = new Date();
  const next = applyClock(now, clock);
  if (next.getTime() <= now.getTime()) next.setDate(next.getDate() + 1);
  return toDateTimeInputValue(next);
}

function getNextWeeklySchedule(weekdayValue: string, clock: string) {
  if (!weekdayValue || !clock) return "";
  const now = new Date();
  const targetWeekday = Number(weekdayValue);
  const todayWeekday = now.getDay() === 0 ? 7 : now.getDay();
  let daysToAdd = (targetWeekday - todayWeekday + 7) % 7;
  const next = applyClock(now, clock);
  next.setDate(now.getDate() + daysToAdd);
  if (next.getTime() <= now.getTime()) {
    daysToAdd += 7;
    next.setDate(now.getDate() + daysToAdd);
  }
  return toDateTimeInputValue(next);
}

function getNextMonthlySchedule(dayValue: string, clock: string) {
  if (!dayValue || !clock) return "";
  const now = new Date();
  const targetDay = Number(dayValue);
  const build = (year: number, monthIndex: number) => {
    const day = Math.min(targetDay, getLastDayOfMonth(year, monthIndex));
    return applyClock(new Date(year, monthIndex, day), clock);
  };
  let next = build(now.getFullYear(), now.getMonth());
  if (next.getTime() <= now.getTime()) {
    next = build(now.getFullYear(), now.getMonth() + 1);
  }
  return toDateTimeInputValue(next);
}

function getNextMinuteSchedule(intervalValue: string) {
  const interval = Number(intervalValue);
  if (!Number.isInteger(interval) || interval < 1 || interval > 59) return "";
  const next = new Date();
  next.setSeconds(0, 0);
  next.setMinutes(next.getMinutes() + interval);
  return toDateTimeInputValue(next);
}

function getNextHourlySchedule(intervalValue: string) {
  const interval = Number(intervalValue);
  if (!Number.isInteger(interval) || interval < 1 || interval > 23) return "";
  const next = new Date();
  next.setSeconds(0, 0);
  next.setHours(next.getHours() + interval);
  return toDateTimeInputValue(next);
}

export default function DispatchCommandDialog({
  open, onOpenChange, command, presetInstanceIds, onDispatched,
}: Props) {
  // ── 命令选择 ─────────────────────────────────────────────
  const [pickedCommand, setPickedCommand] = useState<CommandTemplate | null>(command);
  const [commandSearch, setCommandSearch] = useState("");
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  // 命令模板列表是否已收起（选完命令后自动收起，仍可手动展开重选）
  const [templateListCollapsed, setTemplateListCollapsed] = useState<boolean>(!!command);

  // ── 实例选择 ─────────────────────────────────────────────
  const [agentTypeFilter, setAgentTypeFilter] = useState<AgentTypeKey | "all">("all");
  const [instanceSearch, setInstanceSearch] = useState("");
  const [instancePage, setInstancePage] = useState(1);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  // ── 灰度执行 ─────────────────────────────────────────────
  const [useCanary, setUseCanary] = useState(true);
  const [canaryInstanceId, setCanaryInstanceId] = useState<string | null>(null);

  // ── 执行方式 / 定时执行 ──────────────────────────────────
  const [executionMode, setExecutionMode] = useState<ExecutionMode>("immediate");
  const [scheduleTime, setScheduleTime] = useState("");
  const [scheduleFrequency, setScheduleFrequency] = useState<ScheduledFrequency>("once");
  const [scheduleMinuteInterval, setScheduleMinuteInterval] = useState(DEFAULT_MINUTE_INTERVAL);
  const [scheduleHourInterval, setScheduleHourInterval] = useState(DEFAULT_HOUR_INTERVAL);
  const [scheduleWeekday, setScheduleWeekday] = useState(getCurrentWeekdayValue());
  const [scheduleMonthDay, setScheduleMonthDay] = useState("1");
  const [scheduleTaskName, setScheduleTaskName] = useState("");
  const [scheduleRemark, setScheduleRemark] = useState("");
  const [taskNameTouched, setTaskNameTouched] = useState(false);

  // ── 步骤 + 阶段状态机 ────────────────────────────────────
  // currentStep 仅在 phase=prepare 时有意义
  const [currentStep, setCurrentStep] = useState<StepId>(1);
  const [phase, setPhase] = useState<Phase>("prepare");
  const [testResult, setTestResult] = useState<TestRunResult | null>(null);
  const [skipInstanceStep, setSkipInstanceStep] = useState(false);

  // 弹窗每次打开重置
  useEffect(() => {
    if (open) {
      setPickedCommand(command);
      // 入口已带 command（路径 A）→ 列表默认收起；否则展开等待用户选择
      setTemplateListCollapsed(!!command);
      setCommandSearch("");
      setAgentTypeFilter("all");
      setInstanceSearch("");
      setInstancePage(1);
      // 预选实例由 Agent 列表入口保证已过滤为运行中实例；这里不再用本页 mock 数据二次过滤，
      // 避免两个页面 mock 数据源不完全一致时把已选 Agent 丢掉。
      const presetSet = new Set<string>();
      if (presetInstanceIds && presetInstanceIds.length > 0) {
        presetInstanceIds.forEach((id) => presetSet.add(id));
      }
      setSelected(presetSet);
      setSkipInstanceStep(presetSet.size > 0);
      setUseCanary(true);
      setCanaryInstanceId(null);
      setExecutionMode("immediate");
      setScheduleTime("");
      setScheduleFrequency("once");
      setScheduleMinuteInterval(DEFAULT_MINUTE_INTERVAL);
      setScheduleHourInterval(DEFAULT_HOUR_INTERVAL);
      setScheduleWeekday(getCurrentWeekdayValue());
      setScheduleMonthDay("1");
      setScheduleTaskName(formatScheduleName(new Date()));
      setScheduleRemark("");
      setTaskNameTouched(false);
      setPhase("prepare");
      setTestResult(null);
      // 入口决定初始 step：
      // - 从命令库点「下发」：已有命令，未预选 Agent，则直接选执行对象
      // - 从 Agent 列表勾选后点「命令下发」：已预选 Agent，则只需选命令，随后跳过执行对象
      // - 命令和 Agent 都已有：直接进入执行策略
      setCurrentStep(command ? (presetSet.size > 0 ? 3 : 2) : 1);
      // 初始化参数值
      if (command?.useParams && command.params) {
        const init: Record<string, string> = {};
        command.params.forEach((p) => {
          init[p.key] = p.defaultValue ?? "";
        });
        setParamValues(init);
      } else {
        setParamValues({});
      }
    }
  }, [open, command, presetInstanceIds]);

  // 命令切换时同步参数值
  useEffect(() => {
    if (pickedCommand?.useParams && pickedCommand.params) {
      const init: Record<string, string> = {};
      pickedCommand.params.forEach((p) => {
        init[p.key] = p.defaultValue ?? "";
      });
      setParamValues(init);
    } else {
      setParamValues({});
    }
  }, [pickedCommand]);

  useEffect(() => {
    if (!open || executionMode !== "scheduled" || taskNameTouched) return;
    const d = scheduleTime ? new Date(scheduleTime) : new Date();
    if (!Number.isNaN(d.getTime())) setScheduleTaskName(formatScheduleName(d));
  }, [executionMode, open, scheduleTime, taskNameTouched]);

  // ── 命令搜索 + 分页 ───────────────────────────────────────
  const filteredCommands = useMemo(() => {
    const q = commandSearch.trim().toLowerCase();
    if (!q) return MOCK_COMMAND_TEMPLATES;
    return MOCK_COMMAND_TEMPLATES.filter((t) => (
      t.id.toLowerCase().includes(q) ||
      t.name.toLowerCase().includes(q) ||
      t.content.toLowerCase().includes(q) ||
      (t.description ?? "").toLowerCase().includes(q)
    ));
  }, [commandSearch]);

  // ── 候选实例（过滤后的全集） ─────────────────────────────
  // 命令下发场景下排除 MyAgent（自研 Agent），仅面向 OpenClaw / Hermes / LightclawACE
  const filteredInstances = useMemo(() => {
    return MOCK_INSTANCES.filter((i) => {
      if (i.status !== "running") return false;
      if (i.agentType === "MyAgent") return false;
      if (agentTypeFilter !== "all" && i.agentType !== agentTypeFilter) return false;
      if (instanceSearch.trim()) {
        const q = instanceSearch.trim().toLowerCase();
        return (
          i.name.toLowerCase().includes(q) ||
          i.instanceId.toLowerCase().includes(q) ||
          i.owner.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [agentTypeFilter, instanceSearch]);

  // 分页
  const totalInstancePages = Math.max(1, Math.ceil(filteredInstances.length / INSTANCE_PAGE_SIZE));
  const pagedInstances = useMemo(() => {
    const start = (instancePage - 1) * INSTANCE_PAGE_SIZE;
    return filteredInstances.slice(start, start + INSTANCE_PAGE_SIZE);
  }, [filteredInstances, instancePage]);

  // 搜索/筛选变化时回到第 1 页
  useEffect(() => {
    setInstancePage(1);
  }, [agentTypeFilter, instanceSearch]);

  const toggleInstance = (id: string) => {
    setSelected((prev) => {
      const n = new Set(prev);
      if (n.has(id)) {
        n.delete(id);
        if (canaryInstanceId === id) setCanaryInstanceId(null);
      } else n.add(id);
      return n;
    });
  };

  // 全局全选：勾选/取消所有【匹配当前筛选条件】的实例（不仅是当前页）
  const toggleAllInstances = (checked: boolean) => {
    const allFilteredIds = filteredInstances.map((i) => i.instanceId);
    if (checked) {
      setSelected(new Set([...Array.from(selected), ...allFilteredIds]));
    } else {
      const filteredSet = new Set(allFilteredIds);
      setSelected(new Set(Array.from(selected).filter((id) => !filteredSet.has(id))));
      // 取消勾选时若灰度机被取消了，清掉
      if (canaryInstanceId && filteredSet.has(canaryInstanceId)) {
        setCanaryInstanceId(null);
      }
    }
  };

  // 全选状态：基于「全部筛选结果」而非当前页
  const allInstancesChecked =
    filteredInstances.length > 0 &&
    filteredInstances.every((i) => selected.has(i.instanceId));
  const partialInstancesChecked =
    filteredInstances.some((i) => selected.has(i.instanceId)) && !allInstancesChecked;

  // ── 参数完整性 + 渲染后命令内容 ───────────────────────────
  const danger = pickedCommand ? detectDangerousCommand(pickedCommand.content) : { dangerous: false, reasons: [] };

  const missingParamKeys = useMemo(() => {
    if (!pickedCommand?.useParams || !pickedCommand.params) return [];
    return pickedCommand.params
      .map((p) => p.key)
      .filter((k) => !(paramValues[k] ?? "").trim());
  }, [pickedCommand, paramValues]);

  const renderedContent = useMemo(() => {
    if (!pickedCommand) return "";
    if (!pickedCommand.useParams || !pickedCommand.params?.length) return pickedCommand.content;
    return pickedCommand.content.replace(/\{\{\s*([a-zA-Z_][\w]*)\s*\}\}/g, (_m, k: string) => {
      return paramValues[k] ?? `{{${k}}}`;
    });
  }, [pickedCommand, paramValues]);

  // ── 各 Step 是否完成（用于 stepper 状态 + 下一步按钮 disable） ──
  const step1Done = !!pickedCommand && missingParamKeys.length === 0;
  const step2Done = selected.size > 0;
  const scheduleDate = scheduleTime ? new Date(scheduleTime) : null;
  const scheduleTimeValid = !!scheduleDate && scheduleDate.getTime() > Date.now();
  const scheduleTaskNameValid = scheduleTaskName.trim().length > 0;
  const minuteIntervalNumber = Number(scheduleMinuteInterval);
  const hourIntervalNumber = Number(scheduleHourInterval);
  const scheduleMinuteIntervalValid = scheduleFrequency !== "minutes" || (
    Number.isInteger(minuteIntervalNumber) &&
    minuteIntervalNumber >= 1 &&
    minuteIntervalNumber <= 59
  );
  const scheduleHourIntervalValid = scheduleFrequency !== "hourly" || (
    Number.isInteger(hourIntervalNumber) &&
    hourIntervalNumber >= 1 &&
    hourIntervalNumber <= 23
  );
  const scheduleValid = scheduleTimeValid && scheduleTaskNameValid && scheduleMinuteIntervalValid && scheduleHourIntervalValid;
  const canUseCanary = executionMode === "immediate" && selected.size > 1;
  const step3Done = executionMode === "scheduled"
    ? scheduleValid
    : (!canUseCanary || !useCanary || (canaryInstanceId !== null && selected.has(canaryInstanceId)));

  const canSubmit = step1Done && step2Done && step3Done;
  const visibleStepDefs = skipInstanceStep
    ? STEP_DEFS.filter((s) => s.id !== 2)
    : STEP_DEFS;
  const stepperCurrent = skipInstanceStep && currentStep === 3 ? 2 : currentStep;

  // ── 写入历史记录 ───────────────────────────────────────
  const writeHistoryRecord = (testInfo?: TestRunResult & { instanceId: string }) => {
    if (!pickedCommand) return null;
    const selectedIds = Array.from(selected);
    const now = new Date().toLocaleString("zh-CN", { hour12: false });

    const id = `h-${Date.now().toString(36)}`;
    const record: HistoryRecord = {
      id,
      taskId: `TASK-${new Date().toISOString().slice(0, 10).replace(/-/g, "")}-${Math.floor(Math.random() * 9000 + 1000)}`,
      action: "command-execute",
      assetName: pickedCommand.name,
      operator: "admin@acompany.com",
      isAuto: false,
      operatedAt: now,
      totalInstances: selectedIds.length,
      successCount: selectedIds.length,
      failedCount: 0,
      commandExtra: {
        commandId: pickedCommand.id,
        commandName: pickedCommand.name,
        commandType: "SHELL",
        commandContent: renderedContent,
        commandContentTemplate: pickedCommand.useParams ? pickedCommand.content : undefined,
        paramValues: pickedCommand.useParams && Object.keys(paramValues).length > 0
          ? { ...paramValues }
          : undefined,
        workingDir: pickedCommand.workingDir,
        runAsUser: pickedCommand.runAsUser,
        timeoutSec: pickedCommand.timeoutSec,
        testInstanceId: testInfo?.instanceId,
        testStatus: testInfo?.status,
        testMessage: testInfo
          ? testInfo.status === "success"
            ? `灰度执行成功（exit=${testInfo.exitCode}，耗时 ${testInfo.durationMs}ms）`
            : `灰度执行失败：${testInfo.stderr ?? "未知错误"}`
          : undefined,
      },
      perInstanceResult: selectedIds.map((iid) => {
        const inst = MOCK_INSTANCES.find((x) => x.instanceId === iid);
        if (testInfo && iid === testInfo.instanceId) {
          return {
            instanceId: iid,
            instanceName: inst?.name ?? iid,
            status: "success" as const,
            stdout: testInfo.stdout,
            exitCode: testInfo.exitCode,
            durationMs: testInfo.durationMs,
          };
        }
        return {
          instanceId: iid,
          instanceName: inst?.name ?? iid,
          status: "success" as const,
          stdout: "ok",
          exitCode: 0,
          durationMs: Math.floor(800 + Math.random() * 2000),
        };
      }),
    };
    MOCK_HISTORY.unshift(record);
    return record;
  };

  const writeScheduledHistoryRecord = () => {
    if (!pickedCommand || !scheduleTimeValid) return null;
    const selectedIds = Array.from(selected);
    const createdAt = new Date().toLocaleString("zh-CN", { hour12: false });
    const firstExecuteAt = formatDateTime(scheduleTime);
    const id = `h-scheduled-${Date.now().toString(36)}`;
    const record: HistoryRecord = {
      id,
      taskId: `TASK-${new Date().toISOString().slice(0, 10).replace(/-/g, "")}-${Math.floor(Math.random() * 9000 + 1000)}`,
      action: "command-execute",
      assetName: scheduleTaskName.trim(),
      operator: "admin@acompany.com",
      isAuto: false,
      scheduledAt: firstExecuteAt,
      operatedAt: "—（未开始）",
      scheduledTask: {
        taskName: scheduleTaskName.trim(),
        remark: scheduleRemark.trim() || undefined,
        executeAt: firstExecuteAt,
        firstExecuteAt,
        frequency: scheduleFrequency,
        intervalMinutes: scheduleFrequency === "minutes" ? Number(scheduleMinuteInterval) : undefined,
        intervalHours: scheduleFrequency === "hourly" ? Number(scheduleHourInterval) : undefined,
        status: "pending",
        nextExecuteAt: firstExecuteAt,
        createdBy: "admin@acompany.com",
        createdAt,
      },
      totalInstances: selectedIds.length,
      successCount: 0,
      failedCount: 0,
      commandExtra: {
        commandId: pickedCommand.id,
        commandName: pickedCommand.name,
        commandType: "SHELL",
        commandContent: renderedContent,
        commandContentTemplate: pickedCommand.useParams ? pickedCommand.content : undefined,
        paramValues: pickedCommand.useParams && Object.keys(paramValues).length > 0
          ? { ...paramValues }
          : undefined,
        workingDir: pickedCommand.workingDir,
        runAsUser: pickedCommand.runAsUser,
        timeoutSec: pickedCommand.timeoutSec,
      },
    };
    MOCK_HISTORY.unshift(record);
    return record;
  };

  // ── 灰度执行（mock） ────────────────────────────────────
  const runCanary = () => {
    if (!canaryInstanceId || !pickedCommand) return;
    setPhase("testing");
    const delay = 1200 + Math.random() * 600;
    setTimeout(() => {
      const success = true;
      setTestResult({
        status: success ? "success" : "failed",
        stdout: success
          ? `[mock] 命令执行成功\n命令：${renderedContent.split("\n")[0]}\n输出已记录，共 ${Math.floor(Math.random() * 8 + 1)} 行`
          : "",
        stderr: success ? undefined : "exit code 1: permission denied",
        exitCode: success ? 0 : 1,
        durationMs: Math.floor(delay),
      });
      setPhase("review");
    }, delay);
  };

  const proceedAfterCanary = () => {
    if (!testResult || !canaryInstanceId) return;
    setPhase("submitting");
    const record = writeHistoryRecord({
      ...testResult,
      instanceId: canaryInstanceId,
    });
    const remaining = selected.size - 1;
    toast.success(`命令已下发到 ${selected.size} 台 Agent`, {
      description: remaining > 0
        ? `灰度验证通过，继续下发到剩余 ${remaining} 台`
        : "仅 1 台实例，已完成",
      action: {
        label: "查看执行记录",
        onClick: () => {
          window.history.pushState(null, "", "/admin/agent-commands?tab=history");
          window.dispatchEvent(new PopStateEvent("popstate"));
        },
      },
    });
    if (record) onDispatched?.(record);
    onOpenChange(false);
  };

  const abortAfterCanary = () => {
    toast.info("已终止下发", {
      description: "灰度执行结果已记录，剩余实例未执行",
    });
    onOpenChange(false);
  };

  const handleSubmit = () => {
    if (!canSubmit) return;
    if (executionMode === "scheduled") {
      setPhase("submitting");
      const record = writeScheduledHistoryRecord();
      toast.success(`定时任务「${scheduleTaskName.trim()}」创建成功`, {
        description: `首次执行时间：${formatDateTime(scheduleTime)}`,
        action: {
          label: "查看定时任务",
          onClick: () => {
            window.history.pushState(null, "", "/admin/agent-commands?tab=scheduled");
            window.dispatchEvent(new PopStateEvent("popstate"));
          },
        },
      });
      if (record) onDispatched?.(record);
      onOpenChange(false);
      return;
    }
    if (canUseCanary && useCanary) {
      runCanary();
    } else {
      setPhase("submitting");
      const record = writeHistoryRecord();
      toast.success(`命令已下发到 ${selected.size} 台 Agent`, {
        action: {
          label: "查看执行记录",
          onClick: () => {
            window.history.pushState(null, "", "/admin/agent-commands?tab=history");
            window.dispatchEvent(new PopStateEvent("popstate"));
          },
        },
      });
      if (record) onDispatched?.(record);
      onOpenChange(false);
    }
  };

  // ── 步骤导航 ─────────────────────────────────────────────
  const nextStep = () => {
    if (currentStep === 1 && step1Done) setCurrentStep(skipInstanceStep ? 3 : 2);
    else if (currentStep === 2 && step2Done) setCurrentStep(3);
  };

  const prevStep = () => {
    if (currentStep === 3 && skipInstanceStep) setCurrentStep(1);
    else if (currentStep > 1) setCurrentStep((currentStep - 1) as StepId);
  };

  useEffect(() => {
    if (selected.size <= 1) {
      setCanaryInstanceId(null);
    }
  }, [selected.size]);

  // ── 阻止 testing/submitting 阶段被关闭 ────────────────────
  const handleOpenChange = (v: boolean) => {
    if (!v && (phase === "testing" || phase === "submitting")) return;
    onOpenChange(v);
  };

  if (!open) return null;

  const canaryInstanceName = canaryInstanceId
    ? MOCK_INSTANCES.find((x) => x.instanceId === canaryInstanceId)?.name ?? canaryInstanceId
    : "";

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="flex max-h-[min(90vh,780px)] flex-col sm:max-w-[920px]">
        <DialogHeader>
          <DialogTitle>
            {phase === "testing" && "灰度执行中"}
            {phase === "review" && "灰度执行结果"}
            {(phase === "prepare" || phase === "submitting") && (
              pickedCommand ? `下发命令：${pickedCommand.name}` : "命令下发"
            )}
          </DialogTitle>
          <DialogDescription>
            {phase === "testing" && "正在 1 台实例上跑命令，预计 1~2 秒返回结果。"}
            {phase === "review" && "请确认输出无异常后，再下发到剩余实例。"}
            {phase === "prepare" && STEP_DEFS[currentStep - 1].desc}
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">

        {/* ── prepare 阶段：Stepper + 当前步骤内容 ───────────── */}
        {phase === "prepare" && (
          <div className="space-y-4">
            <UiStepper
              current={stepperCurrent}
              steps={visibleStepDefs.map((s) => ({ label: s.label }))}
            />

            {currentStep === 1 && (
              <Step1PickCommand
                pickedCommand={pickedCommand}
                onPick={(c) => {
                  setPickedCommand(c);
                  // 选中命令模板后，自动收起模板列表
                  if (c) setTemplateListCollapsed(true);
                  else setTemplateListCollapsed(false);
                }}
                commandSearch={commandSearch}
                onSearchChange={setCommandSearch}
                filteredCommands={filteredCommands}
                paramValues={paramValues}
                onParamChange={(k, v) => setParamValues((prev) => ({ ...prev, [k]: v }))}
                missingParamKeys={missingParamKeys}
                renderedContent={renderedContent}
                danger={danger}
                templateListCollapsed={templateListCollapsed}
                onToggleTemplateList={() => setTemplateListCollapsed((v) => !v)}
              />
            )}

            {currentStep === 2 && (
              <Step2PickInstances
                agentTypeFilter={agentTypeFilter}
                onAgentTypeChange={setAgentTypeFilter}
                instanceSearch={instanceSearch}
                onInstanceSearchChange={setInstanceSearch}
                pagedInstances={pagedInstances}
                totalFiltered={filteredInstances.length}
                page={instancePage}
                totalPages={totalInstancePages}
                onPageChange={setInstancePage}
                selected={selected}
                onToggle={toggleInstance}
                onToggleAll={toggleAllInstances}
                allChecked={allInstancesChecked}
                partialChecked={partialInstancesChecked}
              />
            )}

            {currentStep === 3 && pickedCommand && (
              <Step3Policy
                pickedCommand={pickedCommand}
                selectedCount={selected.size}
                executionMode={executionMode}
                onExecutionModeChange={setExecutionMode}
                useCanary={useCanary}
                onUseCanaryChange={setUseCanary}
                canaryInstanceId={canaryInstanceId}
                onCanaryInstanceChange={setCanaryInstanceId}
                selectedInstanceIds={Array.from(selected)}
                scheduleTime={scheduleTime}
                onScheduleTimeChange={setScheduleTime}
                scheduleTimeValid={scheduleTimeValid}
                scheduleFrequency={scheduleFrequency}
                onScheduleFrequencyChange={setScheduleFrequency}
                scheduleMinuteInterval={scheduleMinuteInterval}
                onScheduleMinuteIntervalChange={setScheduleMinuteInterval}
                scheduleMinuteIntervalValid={scheduleMinuteIntervalValid}
                scheduleHourInterval={scheduleHourInterval}
                onScheduleHourIntervalChange={setScheduleHourInterval}
                scheduleHourIntervalValid={scheduleHourIntervalValid}
                scheduleWeekday={scheduleWeekday}
                onScheduleWeekdayChange={setScheduleWeekday}
                scheduleMonthDay={scheduleMonthDay}
                onScheduleMonthDayChange={setScheduleMonthDay}
                scheduleTaskName={scheduleTaskName}
                onScheduleTaskNameChange={(v) => {
                  setTaskNameTouched(true);
                  setScheduleTaskName(v);
                }}
                scheduleTaskNameValid={scheduleTaskNameValid}
                scheduleRemark={scheduleRemark}
                onScheduleRemarkChange={setScheduleRemark}
              />
            )}
          </div>
        )}

        {/* ── testing 阶段 ─────────────────────────────────── */}
        {phase === "testing" && (
          <div className="py-12 flex flex-col items-center text-center space-y-3">
            <div className="w-14 h-14 rounded-full bg-[var(--accent)] flex items-center justify-center">
              <Loader2 className="w-7 h-7 text-[var(--text-warning)] animate-spin" />
            </div>
            <div className="space-y-2">
              <BodyMedium>
                正在 <MetaMedium as="span" tone="inherit" className="text-[var(--text-warning)]">{canaryInstanceName}</MetaMedium> 上执行
              </BodyMedium>
              <MetaText tone="secondary">
                超时 {pickedCommand?.timeoutSec ?? 60} 秒，请勿关闭弹窗
              </MetaText>
            </div>
          </div>
        )}

        {/* ── review 阶段 ─────────────────────────────────── */}
        {phase === "review" && testResult && (
          <div className="space-y-4">
            <Alert variant="info">
              <Info />
              <AlertDescription>
                {testResult.status === "success"
                  ? `请确认输出无异常。点击「继续下发」会向剩余 ${selected.size - 1} 台实例发送同样的命令。`
                  : "灰度执行失败，建议检查命令后重新提交；剩余实例不会被执行。"}
              </AlertDescription>
            </Alert>

            <Alert variant={testResult.status === "success" ? "operation-info" : "error"}>
              {testResult.status === "success" ? <CheckCircle2 /> : <XCircle />}
              <AlertTitle>
                灰度机 <CodeText as="span" tone="inherit">{canaryInstanceName}</CodeText> {testResult.status === "success" ? "执行成功" : "执行失败"}
              </AlertTitle>
              <AlertDescription>
                <span>退出码：<CodeText as="span" tone="inherit" className="tabular-nums">{testResult.exitCode}</CodeText></span>
                <span className="mx-2">·</span>
                <span>耗时：<CodeText as="span" tone="inherit" className="tabular-nums">{testResult.durationMs}ms</CodeText></span>
              </AlertDescription>
            </Alert>

            {testResult.stdout && (
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" className="block">执行结果 (stdout)</MetaMedium>
                <pre className="text-xs font-mono text-[var(--text-title)] bg-[var(--accent)] border border-[var(--border)] rounded-[4px] p-3 max-h-[160px] overflow-auto whitespace-pre-wrap break-all">
                  {testResult.stdout}
                </pre>
              </div>
            )}

            {testResult.stderr && (
              <div className="space-y-2">
                <MetaMedium as="label" tone="danger" className="block">错误输出 (stderr)</MetaMedium>
                <pre className="text-xs font-mono text-[var(--text-danger)] bg-[var(--accent)] border border-[var(--text-danger)]/30 rounded-[4px] p-3 max-h-[160px] overflow-auto whitespace-pre-wrap break-all">
                  {testResult.stderr}
                </pre>
              </div>
            )}
          </div>
        )}
        </DialogBody>

        {/* ── Footer：根据阶段渲染不同按钮（testing/submitting 阶段无 footer） ── */}
        {phase === "prepare" && (
          <DialogFooter>
            <div className="flex-1 text-xs text-[var(--text-secondary)] self-center">
              {currentStep === 3 && executionMode === "immediate" && useCanary && canaryInstanceId && selected.size > 1 && (
                <>先在 <span className="font-medium text-[var(--text-title)]">{canaryInstanceName}</span> 灰度，确认后再下发到剩余 {selected.size - 1} 台</>
              )}
              {currentStep === 3 && executionMode === "immediate" && (!useCanary || selected.size === 1) && selected.size > 0 && (
                <>将一次性下发到 <span className="font-medium text-[var(--text-title)] tabular-nums">{selected.size}</span> 台实例</>
              )}
              {currentStep === 3 && executionMode === "scheduled" && scheduleTime && (
                <>将创建定时任务，首次执行时间：<span className="font-medium text-[var(--text-title)]">{formatDateTime(scheduleTime)}</span></>
              )}
            </div>
            <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            {currentStep > 1 && (
              <Button variant="claw-outline" onClick={prevStep}>
                上一步
              </Button>
            )}
            {currentStep < 3 ? (
              <Button
                variant="dialog-confirm"
                onClick={nextStep}
                disabled={
                  (currentStep === 1 && !step1Done) ||
                  (currentStep === 2 && !step2Done)
                }
              >
                下一步
              </Button>
            ) : (
              <Button
                variant="dialog-confirm"
                onClick={handleSubmit}
                disabled={!canSubmit}
              >
                {executionMode === "scheduled" ? "创建定时任务" : canUseCanary && useCanary ? "在灰度机上执行" : "立即下发"}
              </Button>
            )}
          </DialogFooter>
        )}

        {phase === "review" && testResult && (
          <DialogFooter>
            <Button variant="claw-outline" onClick={abortAfterCanary}>
              <XIcon className="w-3.5 h-3.5 mr-1" />
              终止下发
            </Button>
            {testResult.status === "success" && selected.size > 1 && (
              <Button
                variant="dialog-confirm"
                onClick={proceedAfterCanary}
              >
                继续下发剩余 {selected.size - 1} 台
                <ArrowRight className="w-3.5 h-3.5 ml-1" />
              </Button>
            )}
            {testResult.status === "success" && selected.size === 1 && (
              <Button
                variant="dialog-confirm"
                onClick={proceedAfterCanary}
              >
                完成
              </Button>
            )}
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ────────────────────────────────────────────────────────────
// Step 1 - 选命令 + 填参数
// ────────────────────────────────────────────────────────────
function Step1PickCommand(props: {
  pickedCommand: CommandTemplate | null;
  onPick: (c: CommandTemplate | null) => void;
  commandSearch: string;
  onSearchChange: (v: string) => void;
  filteredCommands: CommandTemplate[];
  paramValues: Record<string, string>;
  onParamChange: (k: string, v: string) => void;
  missingParamKeys: string[];
  renderedContent: string;
  danger: { dangerous: boolean; reasons: string[] };
  templateListCollapsed: boolean;
  onToggleTemplateList: () => void;
}) {
  const {
    pickedCommand, onPick, commandSearch, onSearchChange,
    filteredCommands,
    paramValues, onParamChange, missingParamKeys, renderedContent, danger,
    templateListCollapsed, onToggleTemplateList,
  } = props;

  return (
    <>
      {/* 命令选择列表（始终显示，但选中后默认收起） */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <MetaMedium as="label" tone="secondary" className="flex items-center gap-1">
            选择命令模板 <MetaMedium as="span" tone="danger">*</MetaMedium>
            {pickedCommand && (
              <MetaText as="span" tone="muted" className="ml-2 font-normal">
                · 已选「<MetaMedium as="span" tone="primary">{pickedCommand.name}</MetaMedium>」
              </MetaText>
            )}
          </MetaMedium>
          <div className="flex items-center gap-3">
            {pickedCommand && (
              <button
                type="button"
                onClick={onToggleTemplateList}
                className="text-xs text-[var(--text-brand)] hover:underline inline-flex items-center gap-0.5"
              >
                {templateListCollapsed ? (
                  <>
                    重新选择
                    <ChevronDown className="w-3 h-3" />
                  </>
                ) : (
                  <>
                    收起
                    <ChevronUp className="w-3 h-3" />
                  </>
                )}
              </button>
            )}
          </div>
        </div>

        <Collapsible open={!templateListCollapsed}>
          <CollapsibleContent
            className="overflow-hidden duration-150 ease-out
              data-[state=closed]:animate-accordion-up
              data-[state=open]:animate-accordion-down"
          >
            <div className="space-y-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
                <Input
                  value={commandSearch}
                  onChange={(e) => onSearchChange(e.target.value)}
                  placeholder="搜索命令名称、ID、内容、备注"
                  className="h-9 pl-9"
                />
              </div>
              {/* 固定高度溢出滚动区 */}
              <div
                className="max-h-[320px] overflow-y-auto pr-1
                  [&::-webkit-scrollbar]:w-[6px]
                  [&::-webkit-scrollbar-thumb]:rounded-full
                  [&::-webkit-scrollbar-thumb]:bg-transparent
                  [&::-webkit-scrollbar-track]:bg-transparent
                  hover:[&::-webkit-scrollbar-thumb]:bg-[var(--text-weak)]"
              >
                {filteredCommands.length === 0 ? (
                  <div className="py-10 text-center">
                    <HelperText>
                      {commandSearch ? "没有匹配的命令" : "暂无命令模板"}
                    </HelperText>
                    <MetaText tone="weak" className="mt-1">
                      请先到「执行命令」页创建命令
                    </MetaText>
                  </div>
                ) : (
                  <RadioGroup
                    value={pickedCommand?.id ?? ""}
                    onValueChange={(v) => {
                      const t = filteredCommands.find((c) => c.id === v);
                      if (t) onPick(t);
                    }}
                    className="gap-2"
                  >
                    {filteredCommands.map((t) => {
                      const isChecked = pickedCommand?.id === t.id;
                      return (
                        <RadioCard
                          key={t.id}
                          id={`cmd-tpl-${t.id}`}
                          value={t.id}
                          checked={isChecked}
                        title={
                          <div className="flex items-center gap-2 flex-wrap">
                            <Code2 className="w-3.5 h-3.5 text-[var(--decoration-purple)] shrink-0" />
                            <BodyMedium as="span" className="truncate">{t.name}</BodyMedium>
                            <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
                            <MetaText as="span" tone="muted">{t.id}</MetaText>
                            {t.useParams && t.params && t.params.length > 0 && (
                              <>
                                <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
                                <StatusTag
                                  variant="gray"
                                  mode="fill"
                                  className="h-[18px] px-1.5 text-[10px]"
                                >
                                  需要填写 {t.params.length} 个参数
                                </StatusTag>
                              </>
                            )}
                            </div>
                          }
                          description={
                            <code className="text-xs font-mono text-[var(--text-muted)] truncate block">
                              {t.content.split("\n")[0]}
                              {t.content.includes("\n") && <span className="text-[var(--text-weak)] ml-1">…</span>}
                            </code>
                          }
                        />
                      );
                    })}
                  </RadioGroup>
                )}
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>
      </div>

      {/* 已选命令卡片（位于命令列表 与 命令参数 之间） */}
      <div
        className={
          pickedCommand
            ? "rounded-[4px] border border-[var(--text-brand)] bg-[var(--accent)] p-3 space-y-2"
            : "rounded-[4px] border border-dashed border-[var(--border)] bg-[var(--accent)] px-3 py-2.5"
        }
      >
        {pickedCommand ? (
          <>
            <div className="flex items-center gap-2 text-xs text-[var(--text-muted)] flex-wrap">
              <Code2 className="w-3.5 h-3.5 text-[var(--decoration-purple)] shrink-0" />
              <BodyMedium as="span">{pickedCommand.name}</BodyMedium>
              <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
              <CodeText as="span" tone="muted">{pickedCommand.id}</CodeText>
              <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
              <MetaText as="span" tone="muted">类型：<MetaMedium as="span" tone="secondary">SHELL</MetaMedium></MetaText>
              <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
              <MetaText as="span" tone="muted">执行用户：<CodeText as="span" tone="secondary">{pickedCommand.runAsUser}</CodeText></MetaText>
              <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
              <MetaText as="span" tone="muted">路径：<CodeText as="span" tone="secondary">{pickedCommand.workingDir}</CodeText></MetaText>
              <span aria-hidden className="w-px h-3 bg-[var(--border)] shrink-0" />
              <MetaText as="span" tone="muted">超时：<MetaMedium as="span" tone="secondary" className="tabular-nums">{pickedCommand.timeoutSec}</MetaMedium> 秒</MetaText>
            </div>
            <pre className="text-xs font-mono text-[var(--text-secondary)] bg-[var(--popover)] rounded p-2 max-h-[100px] overflow-auto whitespace-pre-wrap break-all border border-[var(--border)]">
              {pickedCommand.content}
            </pre>
          </>
        ) : (
          <div className="flex items-center gap-1.5">
            <Code2 className="w-3.5 h-3.5 shrink-0 text-[var(--text-weak)]" />
            <HelperText as="span">尚未选择命令模板，请在上方列表中选择</HelperText>
          </div>
        )}
      </div>

      {/* 危险命令告警 */}
      {pickedCommand && danger.dangerous && (
        <Alert variant="error">
          <CircleAlert />
          <AlertTitle>检测到高危命令</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4 space-y-0.5">
              {danger.reasons.map((r, i) => (
                <li key={i}>{r}</li>
              ))}
            </ul>
            <div className="mt-1 font-medium">强烈建议在执行策略中开启「灰度执行」先验证。</div>
          </AlertDescription>
        </Alert>
      )}

      {/* 命令参数填值 */}
      {pickedCommand?.useParams && pickedCommand.params && pickedCommand.params.length > 0 && (
        <div className="space-y-3">
          {/* 小标题：输入命令参数 */}
          <MetaMedium as="label" tone="secondary" className="flex items-center gap-1">
            输入命令参数 <span className="text-[var(--text-danger)]">*</span>
          </MetaMedium>

          <SurfaceCard className="flex flex-col gap-3 py-4">
            <div className="px-4">
              <div className="flex items-center justify-between gap-2">
                <MetaMedium as="div" tone="secondary" className="inline-flex items-center gap-1.5 flex-wrap">
                  <Code2 className="w-3.5 h-3.5 text-[var(--decoration-purple)] shrink-0" />
                  命令参数
                  <MetaText as="span" tone="weak">
                    （命令内容中 <CodeText as="span" tone="weak">{"{{key}}"}</CodeText> 占位符的实际值）
                  </MetaText>
                </MetaMedium>
                {missingParamKeys.length === 0 && (
                  <span className="text-[11px] text-[var(--text-success)] inline-flex items-center gap-0.5 shrink-0">
                    <CheckCircle2 className="w-3 h-3" />
                    参数已就绪
                  </span>
                )}
              </div>
            </div>

            <div className="px-4 space-y-4">
              <Table
                density="compact"
                autoFixedColumns={false}
                containerClassName="border border-[var(--border)] rounded-[4px] overflow-hidden bg-[var(--popover)]"
              >
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[42%]">参数名</TableHead>
                    <TableHead>参数值</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pickedCommand.params.map((p) => {
                    const missing = !(paramValues[p.key] ?? "").trim();
                    return (
                      <TableRow key={p.key}>
                        <TableCell>
                          <span className="font-medium text-[var(--text-title)]">{p.key}</span>
                          {p.description && (
                            <>
                              <span className="mx-2 text-[var(--text-weak)]">｜</span>
                              <span className="text-[var(--text-muted)]">{p.description}</span>
                            </>
                          )}
                        </TableCell>
                        <TableCell>
                          <Input
                            id={`param-input-${p.key}`}
                            value={paramValues[p.key] ?? ""}
                            onChange={(e) => onParamChange(p.key, e.target.value)}
                            placeholder={p.defaultValue ? `默认：${p.defaultValue}` : "请输入参数值"}
                            className={`h-7 ${missing ? "border-[var(--text-danger)]/40" : ""}`}
                          />
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>

              {missingParamKeys.length > 0 && (
                <div className="text-xs text-[var(--text-danger)] inline-flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" />
                  以下参数未填值：
                  {missingParamKeys.map((k, i) => (
                    <span key={k} className="font-mono">
                      {i > 0 && "、"}
                      {k}
                    </span>
                  ))}
                </div>
              )}

              {missingParamKeys.length === 0 && (
                <div className="space-y-2">
                  <MetaMedium as="label" tone="muted">
                    替换后的命令内容
                  </MetaMedium>
                  <pre className="text-xs font-mono text-[var(--text-secondary)] bg-[var(--accent)] rounded p-2.5 max-h-[120px] overflow-auto whitespace-pre-wrap break-all border border-[var(--border)]">
                    {renderedContent}
                  </pre>
                </div>
              )}
            </div>
          </SurfaceCard>
        </div>
      )}
    </>
  );
}

// ────────────────────────────────────────────────────────────
// Step 2 - 选执行对象
// ────────────────────────────────────────────────────────────
function Step2PickInstances(props: {
  agentTypeFilter: AgentTypeKey | "all";
  onAgentTypeChange: (v: AgentTypeKey | "all") => void;
  instanceSearch: string;
  onInstanceSearchChange: (v: string) => void;
  pagedInstances: typeof MOCK_INSTANCES;
  totalFiltered: number;
  page: number;
  totalPages: number;
  onPageChange: (p: number) => void;
  selected: Set<string>;
  onToggle: (id: string) => void;
  onToggleAll: (checked: boolean) => void;
  allChecked: boolean;
  partialChecked: boolean;
}) {
  const {
    agentTypeFilter, onAgentTypeChange, instanceSearch, onInstanceSearchChange,
    pagedInstances, totalFiltered, page, totalPages, onPageChange,
    selected, onToggle, onToggleAll, allChecked, partialChecked,
  } = props;

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <MetaMedium as="label" tone="secondary" className="flex items-center gap-1">
          选择目标实例 <span className="text-[var(--text-danger)]">*</span>
          {selected.size > 0 && (
            <span className="ml-1 text-xs text-[var(--text-muted)] tabular-nums font-normal">
              · 已选 {selected.size} 台
            </span>
          )}
        </MetaMedium>
        <MetaText as="span" tone="weak">
          共 {totalFiltered} 台运行中实例
        </MetaText>
      </div>

      <div className="flex items-center gap-2 mb-2">
        <Select value={agentTypeFilter} onValueChange={(v) => onAgentTypeChange(v as AgentTypeKey | "all")}>
          <SelectTrigger className="h-9 w-[160px] text-sm">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部 Agent 类型</SelectItem>
            {AGENT_TYPES.map((t) => (
              <SelectItem key={t} value={t}>
                {AGENT_TYPE_LABEL[t]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          value={instanceSearch}
          onChange={(e) => onInstanceSearchChange(e.target.value)}
          placeholder="搜索实例名 / ID / 创建人"
          className="h-9 flex-1"
        />
      </div>

      <Table
        density="compact"
        autoFixedColumns={false}
        containerClassName="border border-[var(--border)] rounded-[4px] overflow-hidden"
      >
        <TableHeader>
          <TableRow>
            <TableHead className="w-[1%]">
              <Checkbox
                checked={allChecked ? true : partialChecked ? "indeterminate" : false}
                onCheckedChange={(v) => onToggleAll(!!v)}
                className="size-4"
              />
            </TableHead>
            <TableHead>
              实例
              {(allChecked || partialChecked) && (
                <span className="ml-1 text-[var(--text-weak)] font-normal">
                  （表头勾选 = 全选所有匹配筛选的实例）
                </span>
              )}
            </TableHead>
            <TableHead className="w-[16%]">类型</TableHead>
            <TableHead className="w-[16%]">版本</TableHead>
            <TableHead className="w-[20%]">创建人</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {pagedInstances.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="text-center py-10 text-[var(--text-weak)]">
                没有符合条件的实例
              </TableCell>
            </TableRow>
          ) : (
            pagedInstances.map((i) => {
              const checked = selected.has(i.instanceId);
              return (
                <TableRow
                  key={i.instanceId}
                  data-state={checked ? "selected" : undefined}
                  onClick={() => onToggle(i.instanceId)}
                  className="cursor-pointer"
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={checked}
                      onCheckedChange={() => onToggle(i.instanceId)}
                      className="size-4"
                    />
                  </TableCell>
                  <TableCell>
                    <span>{i.name}</span>
                    <span aria-hidden className="inline-block w-px h-3 bg-[var(--border)] align-middle mx-2" />
                    <span className="text-[var(--text-muted)]">{i.instanceId}</span>
                  </TableCell>
                  <TableCell>
                    {AGENT_TYPE_LABEL[i.agentType]}
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {i.agentVersion}
                  </TableCell>
                  <TableCell className="truncate max-w-[140px]">
                    {i.owner}
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {/* 分页器（仅 >1 页时显示） */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-3 text-xs">
          <span className="text-[var(--text-muted)] tabular-nums">
            第 {(page - 1) * INSTANCE_PAGE_SIZE + 1} - {Math.min(page * INSTANCE_PAGE_SIZE, totalFiltered)} 条 / 共 {totalFiltered} 条
          </span>
          <div className="inline-flex items-center gap-1">
            <button
              type="button"
              onClick={() => onPageChange(Math.max(1, page - 1))}
              disabled={page <= 1}
              className="px-2 h-7 rounded border border-[var(--border)] bg-[var(--popover)] text-[var(--text-secondary)] disabled:text-[var(--text-weak)] disabled:cursor-not-allowed hover:border-[var(--text-brand)] hover:text-[var(--text-brand)]"
            >
              ‹
            </button>
            <span className="text-[var(--text-muted)] tabular-nums px-2">
              {page} / {totalPages}
            </span>
            <button
              type="button"
              onClick={() => onPageChange(Math.min(totalPages, page + 1))}
              disabled={page >= totalPages}
              className="px-2 h-7 rounded border border-[var(--border)] bg-[var(--popover)] text-[var(--text-secondary)] disabled:text-[var(--text-weak)] disabled:cursor-not-allowed hover:border-[var(--text-brand)] hover:text-[var(--text-brand)]"
            >
              ›
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ────────────────────────────────────────────────────────────
// Step 3 - 执行策略
// ────────────────────────────────────────────────────────────
function Step3Policy(props: {
  pickedCommand: CommandTemplate;
  selectedCount: number;
  executionMode: ExecutionMode;
  onExecutionModeChange: (v: ExecutionMode) => void;
  useCanary: boolean;
  onUseCanaryChange: (v: boolean) => void;
  canaryInstanceId: string | null;
  onCanaryInstanceChange: (id: string) => void;
  selectedInstanceIds: string[];
  scheduleTime: string;
  onScheduleTimeChange: (v: string) => void;
  scheduleTimeValid: boolean;
  scheduleFrequency: ScheduledFrequency;
  onScheduleFrequencyChange: (v: ScheduledFrequency) => void;
  scheduleMinuteInterval: string;
  onScheduleMinuteIntervalChange: (v: string) => void;
  scheduleMinuteIntervalValid: boolean;
  scheduleHourInterval: string;
  onScheduleHourIntervalChange: (v: string) => void;
  scheduleHourIntervalValid: boolean;
  scheduleWeekday: string;
  onScheduleWeekdayChange: (v: string) => void;
  scheduleMonthDay: string;
  onScheduleMonthDayChange: (v: string) => void;
  scheduleTaskName: string;
  onScheduleTaskNameChange: (v: string) => void;
  scheduleTaskNameValid: boolean;
  scheduleRemark: string;
  onScheduleRemarkChange: (v: string) => void;
}) {
  const {
    pickedCommand, selectedCount,
    executionMode, onExecutionModeChange,
    useCanary, onUseCanaryChange,
    canaryInstanceId, onCanaryInstanceChange,
    selectedInstanceIds,
    scheduleTime, onScheduleTimeChange, scheduleTimeValid,
    scheduleFrequency, onScheduleFrequencyChange,
    scheduleMinuteInterval, onScheduleMinuteIntervalChange, scheduleMinuteIntervalValid,
    scheduleHourInterval, onScheduleHourIntervalChange, scheduleHourIntervalValid,
    scheduleWeekday, onScheduleWeekdayChange,
    scheduleMonthDay, onScheduleMonthDayChange,
    scheduleTaskName, onScheduleTaskNameChange, scheduleTaskNameValid,
    scheduleRemark, onScheduleRemarkChange,
  } = props;
  const showScheduleTimeError = executionMode === "scheduled" && !!scheduleTime && !scheduleTimeValid;
  const showTaskNameError = executionMode === "scheduled" && !scheduleTaskNameValid;
  const scheduleDateValue = getDateInputValue(scheduleTime);
  const scheduleClockValue = getTimeInputValue(scheduleTime);
  const firstExecuteClockValue = scheduleTime ? getTimeInputValue(scheduleTime) : "";
  const updateRecurringSchedule = (
    frequency: ScheduledFrequency,
    clock = scheduleClockValue,
    weekday = scheduleWeekday,
    monthDay = scheduleMonthDay,
    minuteInterval = scheduleMinuteInterval,
    hourInterval = scheduleHourInterval,
  ) => {
    if (frequency === "minutes") {
      onScheduleTimeChange(getNextMinuteSchedule(minuteInterval));
      return;
    }
    if (frequency === "hourly") {
      onScheduleTimeChange(getNextHourlySchedule(hourInterval));
      return;
    }
    if (frequency === "daily") {
      onScheduleTimeChange(getNextDailySchedule(clock));
      return;
    }
    if (frequency === "weekly") {
      onScheduleTimeChange(getNextWeeklySchedule(weekday, clock));
      return;
    }
    if (frequency === "monthly") {
      onScheduleTimeChange(getNextMonthlySchedule(monthDay, clock));
    }
  };
  const updateScheduleFrequency = (frequency: ScheduledFrequency) => {
    onScheduleFrequencyChange(frequency);
    if (frequency === "once") {
      onScheduleTimeChange("");
      return;
    }
    updateRecurringSchedule(frequency, scheduleClockValue || "09:00");
  };
  const updateScheduleDate = (date: string) => {
    if (!date) {
      onScheduleTimeChange("");
      return;
    }
    onScheduleTimeChange(`${date}T${scheduleClockValue || firstExecuteClockValue || "09:00"}`);
  };
  const updateScheduleClock = (clock: string) => {
    if (!clock) {
      onScheduleTimeChange(scheduleFrequency === "once" && scheduleDateValue ? `${scheduleDateValue}T00:00` : "");
      return;
    }
    if (scheduleFrequency === "minutes" || scheduleFrequency === "hourly") {
      onScheduleTimeChange(`${scheduleDateValue || getTodayInputValue()}T${clock}`);
      return;
    }
    if (scheduleFrequency !== "once") {
      updateRecurringSchedule(scheduleFrequency, clock);
      return;
    }
    onScheduleTimeChange(`${scheduleDateValue || getTodayInputValue()}T${clock}`);
  };
  const updateScheduleWeekday = (weekday: string) => {
    onScheduleWeekdayChange(weekday);
    updateRecurringSchedule("weekly", scheduleClockValue || "09:00", weekday);
  };
  const updateScheduleMonthDay = (day: string) => {
    onScheduleMonthDayChange(day);
    updateRecurringSchedule("monthly", scheduleClockValue || "09:00", scheduleWeekday, day);
  };
  const updateScheduleMinuteInterval = (interval: string) => {
    onScheduleMinuteIntervalChange(interval);
    if (!scheduleTime) {
      updateRecurringSchedule("minutes", scheduleClockValue || "09:00", scheduleWeekday, scheduleMonthDay, interval);
    }
  };
  const updateScheduleHourInterval = (interval: string) => {
    onScheduleHourIntervalChange(interval);
    if (!scheduleTime) {
      updateRecurringSchedule("hourly", scheduleClockValue || "09:00", scheduleWeekday, scheduleMonthDay, scheduleMinuteInterval, interval);
    }
  };
  const scheduleSummary = (() => {
    if (!scheduleTime || !scheduleTimeValid) return "";
    if (scheduleFrequency === "minutes") return `每 ${scheduleMinuteInterval} 分钟执行，首次执行时间：${formatDateTime(scheduleTime)}`;
    if (scheduleFrequency === "hourly") return `每 ${scheduleHourInterval} 小时执行，首次执行时间：${formatDateTime(scheduleTime)}`;
    if (scheduleFrequency === "daily") return `每天 ${scheduleClockValue} 执行，首次执行时间：${formatDateTime(scheduleTime)}`;
    if (scheduleFrequency === "weekly") {
      const weekdayLabel = WEEKDAY_OPTIONS.find((item) => item.value === scheduleWeekday)?.label ?? "周一";
      return `每${weekdayLabel} ${scheduleClockValue} 执行，首次执行时间：${formatDateTime(scheduleTime)}`;
    }
    if (scheduleFrequency === "monthly") return `每月 ${scheduleMonthDay} 日 ${scheduleClockValue} 执行，首次执行时间：${formatDateTime(scheduleTime)}`;
    return `首次执行时间：${formatDateTime(scheduleTime)}`;
  })();

  return (
    <>
      <div className="space-y-3">
        <MetaMedium as="label" tone="secondary" className="flex items-center gap-1">
          执行方式 <span className="text-[var(--text-danger)]">*</span>
        </MetaMedium>
        <RadioGroup
          value={executionMode}
          onValueChange={(v) => onExecutionModeChange(v as ExecutionMode)}
          className="grid-cols-2 gap-3"
        >
          <RadioCard
            id="command-execution-immediate"
            value="immediate"
            checked={executionMode === "immediate"}
            title={
              <span className="inline-flex items-center gap-1.5">
                <ArrowRight className="w-3.5 h-3.5 text-[var(--text-brand)]" />
                立即执行
              </span>
            }
            description="保持现有命令下发流程，提交后立即执行并展示执行结果。"
          />
          <RadioCard
            id="command-execution-scheduled"
            value="scheduled"
            checked={executionMode === "scheduled"}
            title={
              <span className="inline-flex items-center gap-1.5">
                <CalendarClock className="w-3.5 h-3.5 text-[var(--text-brand)]" />
                定时执行
              </span>
            }
            description="提交后仅创建定时任务，到达计划时间后自动触发命令。"
          />
        </RadioGroup>
      </div>

      {/* 不勾选灰度时的警示提示 — 放在内容区最上方 */}
      {executionMode === "immediate" && selectedCount > 1 && !useCanary && (
        <Alert variant="warning">
          <AlertCircle />
          <AlertDescription>
            不选择灰度执行，将一次性下发到全部 {selectedCount} 台实例，如命令有误可能同时影响所有实例，请谨慎操作。
          </AlertDescription>
        </Alert>
      )}

      {/* 任务摘要标题 */}
      <MetaMedium as="label" tone="secondary">
        即将对 <span className="font-bold text-[var(--text-brand)] tabular-nums">{selectedCount} 台</span> 实例 执行 <span className="font-bold text-[var(--text-brand)]">{pickedCommand.name}</span> 命令
      </MetaMedium>

      {executionMode === "scheduled" && (
        <SurfaceCard className="p-4 space-y-4 border-[var(--text-brand)] bg-[var(--accent)]">
          <div className="flex items-start gap-2">
            <CalendarClock className="w-4 h-4 text-[var(--text-brand)] mt-0.5 shrink-0" />
            <div className="min-w-0">
              <div className="text-sm font-medium text-[var(--text-title)]">定时任务配置</div>
              <HelperText className="mt-1">
                按执行频率配置触发规则，任务创建后不会立即下发命令。
              </HelperText>
            </div>
          </div>

          <div className="space-y-5">
            <div className="grid grid-cols-[180px_minmax(0,1fr)_132px] gap-3 items-start">
            <div className="space-y-1.5">
              <Label className="text-xs text-[var(--text-secondary)]">
                执行频率 <span className="text-[var(--text-danger)]">*</span>
              </Label>
              <Select
                value={scheduleFrequency}
                onValueChange={(v) => updateScheduleFrequency(v as ScheduledFrequency)}
              >
                <SelectTrigger className="h-9 bg-[var(--popover)]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="once">仅执行一次</SelectItem>
                  <SelectItem value="minutes">按分钟</SelectItem>
                  <SelectItem value="hourly">按小时</SelectItem>
                  <SelectItem value="daily">每天</SelectItem>
                  <SelectItem value="weekly">每周</SelectItem>
                  <SelectItem value="monthly">每月</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[var(--text-secondary)]">
                {scheduleFrequency === "once" ? "执行日期" : "周期规则"} <span className="text-[var(--text-danger)]">*</span>
              </Label>

              {scheduleFrequency === "once" && (
                <DatePicker
                  value={scheduleDateValue}
                  onChange={updateScheduleDate}
                  placeholder="选择日期"
                  className={`w-full bg-[var(--popover)] ${showScheduleTimeError ? "border-[var(--text-danger)]/60" : ""}`}
                />
              )}

              {scheduleFrequency === "daily" && (
                <div className="h-9 rounded-[4px] border border-[var(--border)] bg-[var(--popover)] px-3 text-sm text-[var(--text-title)] flex items-center">
                  每天
                </div>
              )}

              {scheduleFrequency === "minutes" && (
                <div className="grid grid-cols-[minmax(96px,120px)_minmax(0,1fr)] gap-2">
                  <div className="relative">
                    <Input
                      type="number"
                      min={1}
                      max={59}
                      value={scheduleMinuteInterval}
                      onChange={(e) => updateScheduleMinuteInterval(e.target.value)}
                      className={`h-9 pr-12 bg-[var(--popover)] ${!scheduleMinuteIntervalValid ? "border-[var(--text-danger)]/60" : ""}`}
                    />
                    <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-[var(--text-muted)]">
                      分钟
                    </span>
                  </div>
                  <DatePicker
                    value={scheduleDateValue}
                    onChange={updateScheduleDate}
                    placeholder="开始日期"
                    className={`w-full bg-[var(--popover)] ${showScheduleTimeError ? "border-[var(--text-danger)]/60" : ""}`}
                  />
                </div>
              )}

              {scheduleFrequency === "hourly" && (
                <div className="grid grid-cols-[minmax(96px,120px)_minmax(0,1fr)] gap-2">
                  <div className="relative">
                    <Input
                      type="number"
                      min={1}
                      max={23}
                      value={scheduleHourInterval}
                      onChange={(e) => updateScheduleHourInterval(e.target.value)}
                      className={`h-9 pr-12 bg-[var(--popover)] ${!scheduleHourIntervalValid ? "border-[var(--text-danger)]/60" : ""}`}
                    />
                    <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-[var(--text-muted)]">
                      小时
                    </span>
                  </div>
                  <DatePicker
                    value={scheduleDateValue}
                    onChange={updateScheduleDate}
                    placeholder="开始日期"
                    className={`w-full bg-[var(--popover)] ${showScheduleTimeError ? "border-[var(--text-danger)]/60" : ""}`}
                  />
                </div>
              )}

              {scheduleFrequency === "weekly" && (
                <Select value={scheduleWeekday} onValueChange={updateScheduleWeekday}>
                  <SelectTrigger className="h-9 bg-[var(--popover)]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {WEEKDAY_OPTIONS.map((item) => (
                      <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}

              {scheduleFrequency === "monthly" && (
                <Select value={scheduleMonthDay} onValueChange={updateScheduleMonthDay}>
                  <SelectTrigger className="h-9 bg-[var(--popover)]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {MONTH_DAY_OPTIONS.map((day) => (
                      <SelectItem key={day} value={day}>{day} 日</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[var(--text-secondary)]">
                {scheduleFrequency === "minutes" || scheduleFrequency === "hourly" ? "首次时间" : "执行时间"} <span className="text-[var(--text-danger)]">*</span>
              </Label>
              <div className="relative">
                <Clock3 className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-weak)]" />
                <Input
                  id={`scheduled-execute-clock-${scheduleFrequency}`}
                  type="time"
                  value={scheduleFrequency === "minutes" || scheduleFrequency === "hourly" ? firstExecuteClockValue : scheduleClockValue}
                  onChange={(e) => updateScheduleClock(e.target.value)}
                  className={`h-9 pl-9 bg-[var(--popover)] ${showScheduleTimeError ? "border-[var(--text-danger)]/60" : ""}`}
                />
              </div>
            </div>

            <div className="col-span-3 -mt-1 min-h-5">
              {scheduleSummary && (
                <HelperText>{scheduleSummary}</HelperText>
              )}
              {showScheduleTimeError && (
                <div className="text-xs text-[var(--text-danger)] inline-flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" />
                  执行时间需晚于当前时间
                </div>
              )}
              {scheduleFrequency === "minutes" && !scheduleMinuteIntervalValid && (
                <div className="text-xs text-[var(--text-danger)] inline-flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" />
                  请输入 1-59 的分钟间隔
                </div>
              )}
              {scheduleFrequency === "hourly" && !scheduleHourIntervalValid && (
                <div className="text-xs text-[var(--text-danger)] inline-flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" />
                  请输入 1-23 的小时间隔
                </div>
              )}
              {!scheduleTime && (
                <HelperText>
                  {scheduleFrequency === "once"
                    ? "请选择具体日期和时间。"
                    : scheduleFrequency === "minutes" || scheduleFrequency === "hourly"
                      ? "请选择开始日期和首次时间。"
                      : "请选择执行时间。"}
                </HelperText>
              )}
              {scheduleFrequency === "monthly" && Number(scheduleMonthDay) > 28 && (
                <HelperText>当月没有该日期时，将按当月最后一天执行。</HelperText>
              )}
            </div>
            </div>

            <div className="grid grid-cols-2 gap-x-4 gap-y-5">
              <div className="space-y-1.5">
                <Label htmlFor="scheduled-task-name" className="text-xs text-[var(--text-secondary)]">
                  任务名称 <span className="text-[var(--text-danger)]">*</span>
                </Label>
                <Input
                  id="scheduled-task-name"
                  value={scheduleTaskName}
                  onChange={(e) => onScheduleTaskNameChange(e.target.value)}
                  placeholder="定时命令任务-YYYYMMDD-HHMM"
                  className={showTaskNameError ? "border-[var(--text-danger)]/60" : ""}
                />
                {showTaskNameError ? (
                  <div className="text-xs text-[var(--text-danger)] inline-flex items-center gap-1">
                    <AlertCircle className="w-3 h-3" />
                    任务名称不能为空
                  </div>
                ) : (
                  <HelperText>系统自动生成，可手动编辑；任务 ID 用于唯一识别。</HelperText>
                )}
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="scheduled-task-remark" className="text-xs text-[var(--text-secondary)]">
                  任务备注
                </Label>
                <Textarea
                  id="scheduled-task-remark"
                  value={scheduleRemark}
                  onChange={(e) => onScheduleRemarkChange(e.target.value)}
                  placeholder="可填写执行窗口、变更单或风险说明"
                  className="h-9 min-h-9 resize-none"
                />
              </div>
            </div>
          </div>

          <Alert variant="info">
            <Info />
            <AlertDescription>
              定时任务会保存目标 Agent、命令内容、参数、执行时间、频率、创建人以及当前命令模板的超时配置。
            </AlertDescription>
          </Alert>
        </SurfaceCard>
      )}

      {/* 灰度执行 — 多选卡片样式 */}
      {executionMode === "immediate" && selectedCount > 1 && (
        <SurfaceCard
        className={`flex flex-col gap-3 py-4 transition-colors ${
          useCanary
            ? "border-[var(--text-brand)] bg-[var(--accent)]"
            : "border-[var(--border)] bg-[var(--popover)]"
        }`}
      >
        <div className="px-4 space-y-3">
          <label className="flex items-start gap-2 cursor-pointer">
            <Checkbox
              checked={useCanary}
              onCheckedChange={(v) => onUseCanaryChange(v === true)}
              className="mt-0.5"
            />
            <div className="flex-1">
              <div className="text-sm text-[var(--text-title)] font-medium">
                灰度执行（先跑 1 台，推荐）
              </div>
              <HelperText className="mt-1 leading-relaxed">
                先在 1 台实例上跑命令，看输出无异常后再下发到剩余 {selectedCount > 1 ? selectedCount - 1 : 0} 台；如果灰度机失败会自动中止，<span className="text-[var(--text-warning)] font-medium">不会影响其他实例</span>。
              </HelperText>
            </div>
          </label>

          {useCanary && (
            <div className="ml-6 space-y-3">
              {/* 流程示意图 — 弧形连接的分支流程图 */}
              <SurfaceInner className="p-3">
                <div className="flex items-center gap-2">
                  {/* 源节点 */}
                  <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded bg-[var(--accent)] border border-[var(--border)] shrink-0">
                    <FlaskConical className="w-3 h-3 text-[var(--text-warning)]" />
                    <span className="text-[var(--text-warning)] font-medium text-xs">1 台灰度机</span>
                  </div>

                  {/* 分支连接线 */}
                  <div aria-hidden className="relative h-14 w-11 shrink-0 text-[var(--text-weak)]">
                    <span className="absolute left-0 top-[27px] h-px w-11 origin-left -rotate-[22deg] rounded-full bg-current" />
                    <span className="absolute left-0 top-[27px] h-px w-11 origin-left rotate-[22deg] rounded-full bg-current" />
                  </div>

                  {/* 分支节点 */}
                  <div className="flex flex-col gap-2 text-xs flex-1">
                    <div className="flex items-center gap-2 h-6">
                      <span className="text-[var(--text-success)] font-medium shrink-0">成功</span>
                      <span className="text-[var(--text-title)] shrink-0">下发到</span>
                      <div className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-[var(--accent)] border border-[var(--border)] w-fit">
                        <Server className="w-3 h-3 text-[var(--text-brand)]" />
                        <span className="text-[var(--text-brand)] font-medium">
                          剩余 {selectedCount > 1 ? selectedCount - 1 : 0} 台
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 h-6">
                      <span className="text-[var(--text-danger)] font-medium shrink-0">失败</span>
                      <span className="text-[var(--text-title)]">
                        自动中止，剩余实例不会执行
                      </span>
                    </div>
                  </div>
                </div>
              </SurfaceInner>

              {/* 灰度机选择 */}
              <div>
                <MetaMedium as="label" tone="secondary" className="block mb-1.5">
                  挑选 1 台作为灰度机
                  <span className="text-[var(--text-weak)] ml-1">（建议挑非生产环境的实例）</span>
                </MetaMedium>
                <Select
                  value={canaryInstanceId ?? ""}
                  onValueChange={onCanaryInstanceChange}
                >
                  <SelectTrigger className="h-9 w-full max-w-[420px] text-sm bg-[var(--popover)]">
                    <SelectValue placeholder={selectedCount === 0 ? "请先选择执行对象" : "从已选实例中选 1 台"} />
                  </SelectTrigger>
                  <SelectContent>
                    {selectedInstanceIds.map((iid) => {
                      const inst = MOCK_INSTANCES.find((x) => x.instanceId === iid);
                      return (
                        <SelectItem key={iid} value={iid}>
                          <span>{inst?.name ?? iid}</span>
                          <span aria-hidden className="inline-block w-px h-3 bg-[var(--border)] align-middle mx-2" />
                          <span className="text-xs text-[var(--text-muted)]">{iid}</span>
                        </SelectItem>
                      );
                    })}
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}

          {!useCanary && (
            <HelperText className="ml-6">
              已关闭灰度执行。
            </HelperText>
          )}
        </div>
        </SurfaceCard>
      )}
    </>
  );
}
