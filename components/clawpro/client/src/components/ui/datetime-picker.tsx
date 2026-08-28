/**
 * DateTimePicker - Popover + Calendar + 自定义时/分两列面板
 * 精度到分钟。受控组件。
 *
 * 布局方案：flex 两列 + position:absolute 时分面板。
 * 左侧 Calendar 决定容器自然高度；右侧用 position:relative 作为定位锚点，
 * 内部 position:absolute + inset-0 的面板精确填满该高度；
 * height:100% 在有明确高度的定位父级中可靠解析，overflow-y:auto 实现滚动。
 * 零 JS 高度同步，无时序问题。
 */
import * as React from "react";
import moment from "moment";
import { CalendarIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface DateTimePickerProps {
  /** ISO8601 字符串 或 null */
  value: string | null;
  onChange: (v: string | null) => void;
  placeholder?: string;
  /** 禁用早于该 Date 的选项(用于 startAt 不得早于现在) */
  minDate?: Date;
  disabled?: boolean;
  className?: string;
  /** 是否允许清空为 null（如终止时间「无终止」）。开启后面板底部左侧显示蓝色文字按钮 */
  clearable?: boolean;
  /** 清空按钮文案，默认「设为无终止」，仅 clearable 时生效 */
  clearLabel?: string;
}

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 60 }, (_, i) => i);

function pad(n: number) {
  return n < 10 ? `0${n}` : String(n);
}

/** 时/分两列面板 —— 高度由外部 position:absolute 的 inset:0 决定 */
function TimeColumnPanel({
  hour,
  minute,
  onHourChange,
  onMinuteChange,
}: {
  hour: number;
  minute: number;
  onHourChange: (h: number) => void;
  onMinuteChange: (m: number) => void;
}) {
  const hourRef = React.useRef<HTMLDivElement>(null);
  const minRef = React.useRef<HTMLDivElement>(null);

  // hour 变化时只滚动小时列
  React.useEffect(() => {
    const raf = requestAnimationFrame(() => {
      const container = hourRef.current;
      if (!container || container.clientHeight === 0) return;
      const target = container.querySelector<HTMLElement>(`[data-val="${hour}"]`);
      if (target) {
        // 程序触发滚动时使用 smooth 行为
        container.style.scrollBehavior = 'smooth';
        container.scrollTop =
          target.offsetTop - container.clientHeight / 2 + target.clientHeight / 2;
        // 滚动完成后恢复为 auto（避免影响手动滚轮滚动）
        const resetBehavior = () => {
          container.style.scrollBehavior = 'auto';
          container.removeEventListener('scrollend', resetBehavior);
        };
        container.addEventListener('scrollend', resetBehavior);
      }
    });
    return () => cancelAnimationFrame(raf);
  }, [hour]);

  // minute 变化时只滚动分钟列
  React.useEffect(() => {
    const raf = requestAnimationFrame(() => {
      const container = minRef.current;
      if (!container || container.clientHeight === 0) return;
      const target = container.querySelector<HTMLElement>(`[data-val="${minute}"]`);
      if (target) {
        // 程序触发滚动时使用 smooth 行为
        container.style.scrollBehavior = 'smooth';
        container.scrollTop =
          target.offsetTop - container.clientHeight / 2 + target.clientHeight / 2;
        // 滚动完成后恢复为 auto（避免影响手动滚轮滚动）
        const resetBehavior = () => {
          container.style.scrollBehavior = 'auto';
          container.removeEventListener('scrollend', resetBehavior);
        };
        container.addEventListener('scrollend', resetBehavior);
      }
    });
    return () => cancelAnimationFrame(raf);
  }, [minute]);

  const itemCls = (active: boolean) =>
    cn(
      "size-8 flex items-center justify-center text-sm cursor-pointer rounded-md transition-colors select-none",
      active
        ? "bg-[var(--cp-brand-blue)] text-white font-medium"
        : "text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]",
    );

  const rowCls = "flex justify-center py-0.5";

  // 滚动列 inline style：隐藏滚动条但保留鼠标滚轮能力
  const scrollColStyle: React.CSSProperties = {
    flex: 1,
    overflowY: "auto",
    minHeight: 0,
    scrollbarWidth: "none",
    msOverflowStyle: "none",
    scrollBehavior: "auto", // 使用 auto 而非 smooth，避免 JS 滚动时卡顿
  };

  /**
   * 手动处理滚轮事件，绕过 react-remove-scroll 的拦截
   * 当 DateTimePicker 在 Dialog 内使用时，react-remove-scroll 会阻止滚动事件
   * 这里通过 onWheel 手动控制 scrollTop 来修复
   */
  const handleWheel = React.useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      // 只阻止默认行为，不 stopPropagation（让事件继续传播）
      e.preventDefault();

      const container = e.currentTarget;
      // 确保手动滚动时使用 auto 行为（避免和 smooth 冲突）
      container.style.scrollBehavior = 'auto';
      // 直接修改 scrollTop，使用 deltaY 作为滚动距离
      // 乘以 0.5 让滚动速度更接近原生体验
      container.scrollTop += e.deltaY * 0.5;
    },
    []
  );

  return (
    /* h-full 填满 position:absolute 父级的 inset:0 高度；flex 两列均分宽度 */
    <div style={{ display: "flex", height: "100%", overflow: "hidden" }}>
      <div ref={hourRef} style={scrollColStyle} className="py-1" onWheel={handleWheel}>
        {HOURS.map((h) => (
          <div key={h} data-val={h} className={rowCls} onClick={() => onHourChange(h)}>
            <div className={itemCls(h === hour)}>{pad(h)}</div>
          </div>
        ))}
      </div>
      <div ref={minRef} style={scrollColStyle} className="py-1" onWheel={handleWheel}>
        {MINUTES.map((m) => (
          <div key={m} data-val={m} className={rowCls} onClick={() => onMinuteChange(m)}>
            <div className={itemCls(m === minute)}>{pad(m)}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function DateTimePicker({
  value,
  onChange,
  placeholder = "选择日期与时间",
  minDate,
  disabled,
  className,
  clearable = false,
  clearLabel = "设为无终止时间",
}: DateTimePickerProps) {
  const [open, setOpen] = React.useState(false);

  const m = value ? moment(value) : null;
  const dateOnly = m ? m.toDate() : undefined;
  const hour = m ? m.hour() : 0;
  const minute = m ? m.minute() : 0;

  const handleDateChange = (d: Date | undefined) => {
    if (!d) {
      onChange(null);
      return;
    }
    const next = moment(d).hour(hour).minute(minute).second(0).toISOString();
    onChange(next);
  };

  const updateTime = (h: number, mn: number) => {
    const base = m ? m.clone() : moment();
    base.hour(h).minute(mn).second(0);
    onChange(base.toISOString());
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          disabled={disabled}
          className={cn(
            // px-3 与 SelectTrigger/Input 等输入框形态对齐（Button size=default 含 svg 时默认 px-4 偏大，用 ! 覆盖）
            "w-full min-w-0 justify-start text-left font-normal rounded-[var(--radius)] gap-1.5 overflow-hidden",
            "!px-3 has-[>svg]:!px-3",
            "border-[var(--border)] text-[var(--text-emphasis)]",
            // 作为输入框形态：hover/focus/open 仅高亮边框，不叠加灰色底色（与 Input 规范一致）
            "bg-white hover:bg-white focus-visible:bg-white data-[state=open]:bg-white",
            "hover:border-[var(--cp-brand-blue)] focus-visible:border-[var(--cp-brand-blue)]",
            "data-[state=open]:border-[var(--cp-brand-blue)]",
            // 仅"真正未填写"时用 placeholder 灰；clearable（如终止时间「无终止」）空值是有效语义，用正文色
            !value && !clearable && "text-[var(--text-muted)]",
            className,
          )}
        >
          <CalendarIcon className="h-4 w-4 shrink-0 text-gray-400" />
          <span className="min-w-0 flex-1 truncate">{m ? m.format("YYYY-MM-DD HH:mm") : placeholder}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        {/*
          布局结构：
          ┌─ flex (align-items:stretch 默认)
          │  ├─ <Calendar />           ← 自然高度，决定整行高度
          │  └─ div.relative.w-[110px] ← 被 flex 拉伸到与 Calendar 同高
          │     └─ div.absolute.inset-0.overflow-hidden  ← 精确填满父级尺寸
          │        └─ TimeColumnPanel(h-full, overflow-y-auto)  ← 滚动
        */}
        <div className="flex">
          <Calendar
            mode="single"
            selected={dateOnly}
            onSelect={handleDateChange}
            disabled={
              minDate
                ? (d) => d < moment(minDate).startOf("day").toDate()
                : undefined
            }
            autoFocus
            className={cn(
              "[&_[data-selected-single=true]]:!bg-[var(--cp-brand-blue)]",
              "[&_[data-selected-single=true]]:!text-white",
              "[&_[data-selected-single=true]]:!ring-0",
              "[&_[data-selected-single=true]]:!ring-offset-0",
              "[&_[data-selected-single=true]]:!shadow-none",
              "[&_button:not([data-selected-single=true]):hover]:bg-[var(--bg-grey-hover)]",
            )}
          />
          {/* relative 定位锚点：被 flex 拉伸到 Calendar 同高 */}
          <div className="relative border-l border-[var(--border)] w-[84px]">
            {/* absolute 填满父级：精确的 W×H，不参与 flex 高度计算 */}
            <div className="absolute inset-0 overflow-hidden">
              <TimeColumnPanel
                hour={hour}
                minute={minute}
                onHourChange={(h) => updateTime(h, minute)}
                onMinuteChange={(mn) => updateTime(hour, mn)}
              />
            </div>
          </div>
        </div>
        <div className="px-3 py-3 border-t border-[var(--border)] flex items-center justify-between gap-3">
          <span className="text-xs text-[var(--text-muted)] min-w-0 truncate">
            {m ? m.format("YYYY-MM-DD HH:mm") : placeholder}
          </span>
          <div className="flex items-center gap-3 shrink-0">
            {clearable && (
              <button
                type="button"
                className="text-xs text-[var(--cp-brand-blue)] hover:opacity-80 transition-opacity"
                onClick={() => {
                  onChange(null);
                  setOpen(false);
                }}
              >
                {clearLabel}
              </button>
            )}
            <Button
              size="sm"
              className="h-7 text-xs px-3 rounded-[var(--radius)]"
              onClick={() => setOpen(false)}
            >
              确定
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export default DateTimePicker;
