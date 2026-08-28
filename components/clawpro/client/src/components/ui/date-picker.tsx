/**
 * DatePicker - Custom date picker using Popover + Calendar
 * Brand color: #1447E6 (consistent with Input component)
 */
import * as React from "react";
import { CalendarIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export interface DatePickerProps {
  /** Value in YYYY-MM-DD format */
  value?: string;
  /** Callback with YYYY-MM-DD string */
  onChange?: (value: string) => void;
  /** Placeholder text */
  placeholder?: string;
  /** Disabled state */
  disabled?: boolean;
  /** Min date in YYYY-MM-DD format */
  min?: string;
  /** Max date in YYYY-MM-DD format */
  max?: string;
  /** Additional className for the trigger button */
  className?: string;
  /**
   * 用户端形态：圆角变为 rounded-full（胶囊），与 tenant-* Button 系列对齐。
   * 仅 pages/tenant/** 业务页使用；管理端保持 rounded-[4px]。
   * 规范来源：SKILL-TENANT.md（2026-05-23 控件圆角对齐）
   */
  tenant?: boolean;
}

/** Parse YYYY-MM-DD string to Date (local timezone) */
function parseDateString(dateStr: string | undefined): Date | undefined {
  if (!dateStr) return undefined;
  const [y, m, d] = dateStr.split("-").map(Number);
  if (!y || !m || !d) return undefined;
  const date = new Date(y, m - 1, d);
  // Validate the date is real
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    return undefined;
  }
  return date;
}

/** Format Date to YYYY-MM-DD string */
function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function DatePicker({
  value,
  onChange,
  placeholder = "选择日期",
  disabled = false,
  min,
  max,
  className,
  tenant = false,
}: DatePickerProps) {
  const [open, setOpen] = React.useState(false);

  const selectedDate = parseDateString(value);
  const minDate = parseDateString(min);
  const maxDate = parseDateString(max);

  const handleSelect = (day: Date | undefined) => {
    if (day && onChange) {
      onChange(formatDate(day));
    }
    setOpen(false);
  };

  // Build disabled matcher for react-day-picker
  const disabledMatcher = React.useMemo(() => {
    const matchers: Array<{ before: Date } | { after: Date }> = [];
    if (minDate) {
      matchers.push({ before: minDate });
    }
    if (maxDate) {
      matchers.push({ after: maxDate });
    }
    return matchers.length > 0 ? matchers : undefined;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [min, max]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild disabled={disabled}>
        <button
          type="button"
          disabled={disabled}
          data-tenant={tenant ? "true" : undefined}
          className={cn(
            "inline-flex items-center justify-between gap-2 h-9 px-3 text-sm border border-gray-200 bg-white transition-colors cursor-pointer select-none whitespace-nowrap",
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
          <span
            className={cn(
              "truncate",
              value ? "text-gray-950" : "text-gray-400"
            )}
          >
            {value || placeholder}
          </span>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="shrink-0 text-gray-400"><g clipPath="url(#clip0_dp_cal)"><path d="M13 1.75H11.75V1.5C11.75 1.30109 11.671 1.11032 11.5303 0.96967C11.3897 0.829018 11.1989 0.75 11 0.75C10.8011 0.75 10.6103 0.829018 10.4697 0.96967C10.329 1.11032 10.25 1.30109 10.25 1.5V1.75H5.75V1.5C5.75 1.30109 5.67098 1.11032 5.53033 0.96967C5.38968 0.829018 5.19891 0.75 5 0.75C4.80109 0.75 4.61032 0.829018 4.46967 0.96967C4.32902 1.11032 4.25 1.30109 4.25 1.5V1.75H3C2.66848 1.75 2.35054 1.8817 2.11612 2.11612C1.8817 2.35054 1.75 2.66848 1.75 3V13C1.75 13.3315 1.8817 13.6495 2.11612 13.8839C2.35054 14.1183 2.66848 14.25 3 14.25H13C13.3315 14.25 13.6495 14.1183 13.8839 13.8839C14.1183 13.6495 14.25 13.3315 14.25 13V3C14.25 2.66848 14.1183 2.35054 13.8839 2.11612C13.6495 1.8817 13.3315 1.75 13 1.75ZM4.25 3.25C4.25 3.44891 4.32902 3.63968 4.46967 3.78033C4.61032 3.92098 4.80109 4 5 4C5.19891 4 5.38968 3.92098 5.53033 3.78033C5.67098 3.63968 5.75 3.44891 5.75 3.25H10.25C10.25 3.44891 10.329 3.63968 10.4697 3.78033C10.6103 3.92098 10.8011 4 11 4C11.1989 4 11.3897 3.92098 11.5303 3.78033C11.671 3.63968 11.75 3.44891 11.75 3.25H12.75V4.75H3.25V3.25H4.25ZM3.25 12.75V6.25H12.75V12.75H3.25Z" fill="currentColor"/></g><defs><clipPath id="clip0_dp_cal"><rect width="16" height="16" fill="white"/></clipPath></defs></svg>
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="w-auto p-0"
        align="start"
        sideOffset={4}
      >
        <Calendar
          mode="single"
          selected={selectedDate}
          onSelect={handleSelect}
          defaultMonth={selectedDate}
          disabled={disabledMatcher}
          classNames={{
            today:
              "relative after:absolute after:bottom-1 after:left-1/2 after:-translate-x-1/2 after:w-1 after:h-1 after:rounded-full after:bg-[#1447E6]",
          }}
          className="[&_[data-selected-single=true]]:bg-[#1447E6] [&_[data-selected-single=true]]:text-white [&_[data-selected-single=true]]:hover:bg-[#1447E6] [&_[data-selected-single=true]]:hover:text-white [&_button:not([data-selected-single=true]):hover]:bg-[#eff4ff]"
        />
      </PopoverContent>
    </Popover>
  );
}

export { DatePicker };
