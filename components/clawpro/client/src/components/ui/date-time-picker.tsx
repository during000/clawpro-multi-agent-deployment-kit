/**
 * DateTimePicker - 日期 + 时间选择器
 *
 * 开源底座：react-day-picker（与项目内 <DatePicker> / <Calendar> 同源，保留其
 * 全部日期能力：键盘操作、月份切换、min/max 禁用、本地时区解析等），右侧时分（秒）
 * 多列与底部「预览 + 确定」栏为按 OpenClaw 设计稿（Figma 截图 2026-06-16）映射实现。
 *
 * 设计令牌映射：
 *   - 选中态主色：#1447E6（与 DatePicker / Input focus 一致）
 *   - 触发框：白底 / 1px #E5E7EB 边 / focus & open 蓝边（同 DatePicker）
 *   - 确定按钮：黑底白字（Button variant="default"）
 *   - 弹层：Popover（8px 圆角 + 阴影，全局统一）
 *
 * 值格式：默认 YYYY-MM-DD HH:mm；开启 showSeconds 后为 YYYY-MM-DD HH:mm:ss
 */
import * as React from "react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export interface DateTimePickerProps {
  /** 值，默认格式 YYYY-MM-DD HH:mm；开启 showSeconds 后为 YYYY-MM-DD HH:mm:ss */
  value?: string;
  /** 变更回调，回传与 value 同格式的字符串 */
  onChange?: (value: string) => void;
  /** 占位文案 */
  placeholder?: string;
  /** 禁用态 */
  disabled?: boolean;
  /** 最小可选日期，格式 YYYY-MM-DD */
  min?: string;
  /** 最大可选日期，格式 YYYY-MM-DD */
  max?: string;
  /** 分钟步长，默认 1（生成 0,1,2...59） */
  minuteStep?: number;
  /** 是否显示「秒」列，默认 false。开启后值格式扩展为 YYYY-MM-DD HH:mm:ss */
  showSeconds?: boolean;
  /** 秒步长，默认 1（生成 0,1,2...59），仅 showSeconds 时生效 */
  secondStep?: number;
  /** 触发框额外类名 */
  className?: string;
  /**
   * 用户端形态：圆角变为 rounded-full（胶囊），与 tenant-* Button 系列对齐。
   * 仅 pages/tenant/** 业务页使用；管理端保持 rounded-[4px]。
   */
  tenant?: boolean;
}

/** 解析 YYYY-MM-DD HH:mm[:ss] 为 { date, hour, minute, second }（本地时区） */
function parseDateTimeString(str: string | undefined): {
  date?: Date;
  hour: number;
  minute: number;
  second: number;
} {
  if (!str) return { date: undefined, hour: 0, minute: 0, second: 0 };
  const [datePart, timePart = "00:00"] = str.trim().split(/\s+/);
  const [y, m, d] = datePart.split("-").map(Number);
  const [hh, mm, ss] = timePart.split(":").map(Number);
  const hour = Number.isFinite(hh) ? Math.min(23, Math.max(0, hh)) : 0;
  const minute = Number.isFinite(mm) ? Math.min(59, Math.max(0, mm)) : 0;
  const second = Number.isFinite(ss) ? Math.min(59, Math.max(0, ss)) : 0;
  if (!y || !m || !d) return { date: undefined, hour, minute, second };
  const date = new Date(y, m - 1, d);
  if (
    date.getFullYear() !== y ||
    date.getMonth() !== m - 1 ||
    date.getDate() !== d
  ) {
    return { date: undefined, hour, minute, second };
  }
  return { date, hour, minute, second };
}

/** 仅格式化日期为 YYYY-MM-DD */
function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

/** 组合为 YYYY-MM-DD HH:mm[:ss] */
function formatDateTime(
  date: Date,
  hour: number,
  minute: number,
  second: number,
  showSeconds: boolean
): string {
  const hh = String(hour).padStart(2, "0");
  const mm = String(minute).padStart(2, "0");
  const time = showSeconds
    ? `${hh}:${mm}:${String(second).padStart(2, "0")}`
    : `${hh}:${mm}`;
  return `${formatDate(date)} ${time}`;
}

interface TimeColumnProps {
  values: number[];
  active: number;
  onPick: (v: number) => void;
}

/** 单列时间滚动选择（小时 / 分钟） */
function TimeColumn({ values, active, onPick }: TimeColumnProps) {
  const containerRef = React.useRef<HTMLDivElement>(null);
  const activeRef = React.useRef<HTMLButtonElement>(null);

  React.useEffect(() => {
    // 选中项滚动到容器可视区域（仅在列内滚动，不影响整页）
    activeRef.current?.scrollIntoView({ block: "nearest" });
  }, [active]);

  return (
    <div
      ref={containerRef}
      className="flex w-14 flex-col gap-1 overflow-y-auto px-1 py-1 [scrollbar-width:thin]"
    >
      {values.map(v => {
        const isActive = v === active;
        return (
          <button
            key={v}
            ref={isActive ? activeRef : undefined}
            type="button"
            onClick={() => onPick(v)}
            className={cn(
              "flex h-8 shrink-0 items-center justify-center rounded-[4px] text-sm tabular-nums transition-colors cursor-pointer select-none",
              isActive
                ? "bg-[#1447E6] text-white font-medium"
                : "text-gray-950 hover:bg-[#eff4ff]"
            )}
          >
            {String(v).padStart(2, "0")}
          </button>
        );
      })}
    </div>
  );
}

function DateTimePicker({
  value,
  onChange,
  placeholder = "选择日期时间",
  disabled = false,
  min,
  max,
  minuteStep = 1,
  showSeconds = false,
  secondStep = 1,
  className,
  tenant = false,
}: DateTimePickerProps) {
  const [open, setOpen] = React.useState(false);

  // 草稿态：点「确定」前不提交
  const parsed = React.useMemo(() => parseDateTimeString(value), [value]);
  const [draftDate, setDraftDate] = React.useState<Date | undefined>(
    parsed.date
  );
  const [draftHour, setDraftHour] = React.useState<number>(parsed.hour);
  const [draftMinute, setDraftMinute] = React.useState<number>(parsed.minute);
  const [draftSecond, setDraftSecond] = React.useState<number>(parsed.second);

  // 打开时用当前值重置草稿
  React.useEffect(() => {
    if (open) {
      const p = parseDateTimeString(value);
      setDraftDate(p.date);
      setDraftHour(p.hour);
      setDraftMinute(p.minute);
      setDraftSecond(p.second);
    }
  }, [open, value]);

  const minDate = parseDateTimeString(min).date;
  const maxDate = parseDateTimeString(max).date;

  const disabledMatcher = React.useMemo(() => {
    const matchers: Array<{ before: Date } | { after: Date }> = [];
    if (minDate) matchers.push({ before: minDate });
    if (maxDate) matchers.push({ after: maxDate });
    return matchers.length > 0 ? matchers : undefined;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [min, max]);

  const hours = React.useMemo(
    () => Array.from({ length: 24 }, (_, i) => i),
    []
  );
  const minutes = React.useMemo(() => {
    const step = minuteStep > 0 ? Math.floor(minuteStep) : 1;
    const list: number[] = [];
    for (let i = 0; i < 60; i += step) list.push(i);
    return list;
  }, [minuteStep]);
  const seconds = React.useMemo(() => {
    const step = secondStep > 0 ? Math.floor(secondStep) : 1;
    const list: number[] = [];
    for (let i = 0; i < 60; i += step) list.push(i);
    return list;
  }, [secondStep]);

  const previewText = draftDate
    ? formatDateTime(draftDate, draftHour, draftMinute, draftSecond, showSeconds)
    : placeholder;

  const handleConfirm = () => {
    if (draftDate && onChange) {
      onChange(
        formatDateTime(draftDate, draftHour, draftMinute, draftSecond, showSeconds)
      );
    }
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild disabled={disabled}>
        <button
          type="button"
          disabled={disabled}
          data-tenant={tenant ? "true" : undefined}
          className={cn(
            "inline-flex items-center justify-between gap-2 h-9 px-3 text-sm border border-gray-200 bg-white transition-colors cursor-pointer select-none whitespace-nowrap min-w-[200px]",
            tenant ? "rounded-full" : "rounded-[4px]",
            "hover:border-blue-500",
            "focus:outline-none focus:border-blue-500",
            "focus-visible:outline-none focus-visible:border-blue-500",
            open && "border-blue-500",
            disabled &&
              "bg-[#FAFAFA] border-gray-200 text-gray-400 cursor-not-allowed hover:border-gray-200",
            className
          )}
        >
          <span className="inline-flex items-center gap-2 truncate">
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              className="shrink-0 text-gray-400"
            >
              <g clipPath="url(#clip0_dtp_cal)">
                <path
                  d="M13 1.75H11.75V1.5C11.75 1.30109 11.671 1.11032 11.5303 0.96967C11.3897 0.829018 11.1989 0.75 11 0.75C10.8011 0.75 10.6103 0.829018 10.4697 0.96967C10.329 1.11032 10.25 1.30109 10.25 1.5V1.75H5.75V1.5C5.75 1.30109 5.67098 1.11032 5.53033 0.96967C5.38968 0.829018 5.19891 0.75 5 0.75C4.80109 0.75 4.61032 0.829018 4.46967 0.96967C4.32902 1.11032 4.25 1.30109 4.25 1.5V1.75H3C2.66848 1.75 2.35054 1.8817 2.11612 2.11612C1.8817 2.35054 1.75 2.66848 1.75 3V13C1.75 13.3315 1.8817 13.6495 2.11612 13.8839C2.35054 14.1183 2.66848 14.25 3 14.25H13C13.3315 14.25 13.6495 14.1183 13.8839 13.8839C14.1183 13.6495 14.25 13.3315 14.25 13V3C14.25 2.66848 14.1183 2.35054 13.8839 2.11612C13.6495 1.8817 13.3315 1.75 13 1.75ZM4.25 3.25C4.25 3.44891 4.32902 3.63968 4.46967 3.78033C4.61032 3.92098 4.80109 4 5 4C5.19891 4 5.38968 3.92098 5.53033 3.78033C5.67098 3.63968 5.75 3.44891 5.75 3.25H10.25C10.25 3.44891 10.329 3.63968 10.4697 3.78033C10.6103 3.92098 10.8011 4 11 4C11.1989 4 11.3897 3.92098 11.5303 3.78033C11.671 3.63968 11.75 3.44891 11.75 3.25H12.75V4.75H3.25V3.25H4.25ZM3.25 12.75V6.25H12.75V12.75H3.25Z"
                  fill="currentColor"
                />
              </g>
              <defs>
                <clipPath id="clip0_dtp_cal">
                  <rect width="16" height="16" fill="white" />
                </clipPath>
              </defs>
            </svg>
            <span
              className={cn("truncate", value ? "text-gray-950" : "text-gray-400")}
            >
              {value || placeholder}
            </span>
          </span>
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0 overflow-hidden" align="start" sideOffset={4}>
        <div className="flex">
          {/* 左：日历（保留 react-day-picker 全部能力） */}
          <Calendar
            mode="single"
            selected={draftDate}
            onSelect={day => setDraftDate(day ?? undefined)}
            defaultMonth={draftDate}
            disabled={disabledMatcher}
            classNames={{
              today:
                "relative after:absolute after:bottom-1 after:left-1/2 after:-translate-x-1/2 after:w-1 after:h-1 after:rounded-full after:bg-[#1447E6]",
            }}
            className="[&_[data-selected-single=true]]:bg-[#1447E6] [&_[data-selected-single=true]]:text-white [&_[data-selected-single=true]]:hover:bg-[#1447E6] [&_[data-selected-single=true]]:hover:text-white [&_button:not([data-selected-single=true]):hover]:bg-[#eff4ff]"
          />

          {/* 右：时分（秒）列 */}
          <div className="flex border-l border-gray-200">
            <div className="max-h-[280px] overflow-hidden py-1">
              <TimeColumn
                values={hours}
                active={draftHour}
                onPick={setDraftHour}
              />
            </div>
            <div className="max-h-[280px] overflow-hidden py-1 border-l border-gray-100">
              <TimeColumn
                values={minutes}
                active={draftMinute}
                onPick={setDraftMinute}
              />
            </div>
            {showSeconds && (
              <div className="max-h-[280px] overflow-hidden py-1 border-l border-gray-100">
                <TimeColumn
                  values={seconds}
                  active={draftSecond}
                  onPick={setDraftSecond}
                />
              </div>
            )}
          </div>
        </div>

        {/* 底：预览 + 确定 */}
        <div className="flex items-center justify-between gap-3 border-t border-gray-200 px-4 py-2.5">
          <span
            className={cn(
              "text-sm tabular-nums truncate",
              draftDate ? "text-gray-500" : "text-gray-400"
            )}
          >
            {previewText}
          </span>
          <Button
            type="button"
            size="sm"
            onClick={handleConfirm}
            disabled={!draftDate}
          >
            确定
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export { DateTimePicker };
