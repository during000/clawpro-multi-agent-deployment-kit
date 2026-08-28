/**
 * StatusBadge - Agent 卡片状态标签（带动画版本）
 * 8 种状态：creating / createFail / running / shutdown / loading / loadFail / maintaining / pending
 *
 * 动画效果：
 *   - creating: 雷达圈从中心向外扩散
 *   - createFail: 感叹号摇摆
 *   - running: 水波填充动画
 *   - shutdown: 静态灰色图标
 *   - loading: 箭头自旋
 *   - loadFail: 感叹号摇摆变叉号
 *   - maintaining: 齿轮旋转
 *   - pending: 三点依次闪烁
 */
import type { CSSProperties, ReactElement } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export type OpenClawStatus =
  | "creating"
  | "createFail"
  | "running"
  | "shutdown"
  | "loading"
  | "loadFail"
  | "maintaining"
  | "pending"
  | "rollingBack";

interface StatusConfigItem {
  label: string;
  tooltipText?: string;
}

export const STATUS_VISUAL: Record<OpenClawStatus, StatusConfigItem> = {
  creating: { label: "创建中", tooltipText: "正在创建中，请稍候" },
  createFail: { label: "创建失败", tooltipText: "创建失败，可删除后重新创建" },
  running: { label: "运行中" },
  shutdown: { label: "已关机", tooltipText: "已关机，如需恢复请联系管理员" },
  loading: { label: "加载中", tooltipText: "加载中，请稍候" },
  loadFail: { label: "加载失败", tooltipText: "加载失败，可点击重试恢复" },
  maintaining: { label: "维护中", tooltipText: "维护中，请稍候" },
  pending: { label: "待处理", tooltipText: "已停用，请联系管理员处理" },
  rollingBack: { label: "回滚中", tooltipText: "数据回滚中，请稍候" },
};

/** 是否禁用交互（按钮、卡片点击）。从 STATUS_VISUAL 派生方便页面使用 */
export const STATUS_DISABLED: Record<OpenClawStatus, boolean> = {
  creating: true,
  createFail: true,
  running: false,
  shutdown: true,
  loading: true,
  loadFail: true,
  maintaining: true,
  pending: true,
  rollingBack: true,
};

/**
 * 头像是否灰显
 *
 * 0608 设计决议：8 种状态全部 `false` 是有意为之 —
 * 卡片头像始终保持彩色，禁用态仅靠状态标签 + 按钮 disabled 表达，避免双重视觉降级（headColor + opacity）
 * 让卡片显得过于冷清。如需恢复"创建失败/已关机/加载失败/待处理"灰头像，请先与设计 review 后修改。
 */
export const STATUS_GRAY_AVATAR: Record<OpenClawStatus, boolean> = {
  creating: false,
  createFail: false,
  running: false,
  shutdown: false,
  loading: false,
  loadFail: false,
  maintaining: false,
  pending: false,
  rollingBack: false,
};

/**
 * 卡片正文文字是否降级（40% 透明）
 *
 * 0609 设计决议：对齐 main 上"创建失败 / 已关机 / 加载失败 / 待处理"卡片整体偏灰透明的视觉，
 * 但仅作用于**正文文字**（标题、元信息、组织、创建时间），不影响：
 *   - 头像（保持彩色，见 STATUS_GRAY_AVATAR）
 *   - 状态徽章（异常态本身红/黄/灰更显眼，不需再降级）
 *   - 操作按钮（按钮的 disabled 态由 Button 组件 / variant 自行处理）
 *
 * 透明度统一 40%，与全站 disabled 态 token（`opacity-40` / 文字 40% 透明）一致。
 *
 * 进行中态（creating / loading / maintaining）保持 100% 不透明，以维持"请稍候"语义可读性。
 */
export const STATUS_DIMMED_TEXT: Record<OpenClawStatus, boolean> = {
  creating: false,
  createFail: true,
  running: false,
  shutdown: true,
  loading: false,
  loadFail: true,
  maintaining: false,
  pending: true,
  rollingBack: false,
};

/* -------------------------------------------------------------------------- */
/*                              状态图标组件                                    */
/* -------------------------------------------------------------------------- */

const SIZE = 16;

/** 创建中：雷达圈从三角形顶部依次扩散 */
const CreatingIcon = () => (
  <span className="status-creating-icon">
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" style={{ overflow: "visible" }}>
      <path d="M8 10L11 14H5L8 10Z" fill="#355EF1" />
      <circle className="status-ring1" cx="8" cy="7" r="0" stroke="#355EF1" fill="none" strokeWidth="2" />
      <circle className="status-ring2" cx="8" cy="7" r="0" stroke="#355EF1" fill="none" strokeWidth="2" />
      <circle className="status-ring3" cx="8" cy="7" r="0" stroke="#355EF1" fill="none" strokeWidth="2" />
    </svg>
  </span>
);

/** 创建失败：感叹号摇摆 */
const CreateFailIcon = () => (
  <span className="status-fail-icon">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <g className="status-excl">
        <path
          d="M4.93262 2.6875H11.0664L14.1338 8L11.0664 13.3125H4.93262L1.86523 8L4.93262 2.6875Z"
          stroke="#D42A1E"
          strokeWidth="1.5"
        />
        <path d="M8 5V8.5" stroke="#D42A1E" strokeWidth="1.5" strokeLinecap="round" />
        <path d="M8 9.5V11" stroke="#D42A1E" strokeWidth="1.5" strokeLinecap="round" />
      </g>
    </svg>
  </span>
);

/** 运行中：水波填充动画 */
const RunningIcon = () => (
  <span className="status-running-icon">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <clipPath id="status-rc">
          <circle cx="8" cy="8" r="3.5" />
        </clipPath>
      </defs>
      <circle cx="8" cy="8" r="5.25" stroke="#16A34A" strokeWidth="1.5" fill="none" />
      <g clipPath="url(#status-rc)">
        <path className="status-wave" d="M3 9 Q5.5 7.5 8 9 Q10.5 10.5 13 9 L13 15 L3 15 Z" fill="#16A34A" />
        <path className="status-wave" style={{ animationDelay: "-0.6s" }} d="M3 10 Q5.5 8.5 8 10 Q10.5 11.5 13 10 L13 15 L3 15 Z" fill="#16A34A" opacity="0.5" />
      </g>
    </svg>
  </span>
);

/** 已关机：静态灰色图标 */
const ShutdownIcon = () => (
  <span className="status-shutdown-icon">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M8.75 2V8H7.25V2H8.75Z" fill="#858D99" />
      <path
        d="M10 2.3418C12.3303 3.1655 14 5.38761 14 8C14 11.3137 11.3137 14 8 14C4.68629 14 2 11.3137 2 8C2 5.38761 3.66968 3.1655 6 2.3418V3.96777C4.51828 4.70413 3.5 6.23314 3.5 8C3.5 10.4853 5.51472 12.5 8 12.5C10.4853 12.5 12.5 10.4853 12.5 8C12.5 6.23314 11.4817 4.70413 10 3.96777V2.3418Z"
        fill="#858D99"
      />
    </svg>
  </span>
);

/** 加载中：箭头自旋 */
const LoadingIcon = () => (
  <span className="status-loading-icon">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M7.32836 3.83878C6.56716 3.83878 5.89552 4.01855 5.26866 4.33316C3.56716 5.00732 2.35821 6.71518 2.35821 8.69271C2.35821 9.72642 2.67164 10.6253 3.20896 11.3893C2.44776 10.4006 2 9.14215 2 7.79383C2 4.46799 4.68657 1.77136 8 1.77136C9.52239 1.77136 10.9104 2.35563 11.9403 3.25451L10.2836 4.91743C9.47761 4.24327 8.44776 3.83878 7.32836 3.83878ZM12.5224 3.83878C13.1493 4.60282 13.5522 5.63653 13.5522 6.71518C13.5522 7.47923 13.3731 8.15338 13.0597 8.7826C12.3881 10.4905 10.6866 11.7039 8.71642 11.7039C7.32836 11.7039 6.02985 11.0747 5.1791 10.1309L3.52239 11.7489C4.64179 13.0073 6.20895 13.7714 8 13.7714C11.3134 13.7714 14 11.0747 14 7.74889C14 6.26574 13.4179 4.91743 12.5224 3.83878Z"
        fill="#355EF1"
      />
    </svg>
  </span>
);

/** 加载失败：感叹号摇摆变叉号 */
const LoadFailIcon = () => (
  <span className="status-loadfail-icon">
    <svg width="16" height="16" viewBox="1 1 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M7.328 3.839C6.567 3.839 5.896 4.019 5.269 4.333C3.567 5.007 2.358 6.715 2.358 8.693C2.358 9.726 2.672 10.625 3.209 11.389C2.448 10.401 2 9.142 2 7.794C2 4.468 4.687 1.771 8 1.771C9.522 1.771 10.91 2.356 11.94 3.255L10.284 4.917C9.478 4.243 8.448 3.839 7.328 3.839ZM12.522 3.839C13.149 4.603 13.552 5.637 13.552 6.715C13.552 7.479 13.373 8.153 13.06 8.783C12.388 10.49 10.687 11.704 8.716 11.704C7.328 11.704 6.03 11.075 5.179 10.131L3.522 11.749C4.642 13.007 6.209 13.771 8 13.771C11.314 13.771 14 11.075 14 7.749C14 6.266 13.418 4.917 12.522 3.839Z"
        fill="#D42A1E"
      />
      <g className="status-lf-excl">
        <path d="M8 5.5V8" stroke="#D42A1E" strokeWidth="1.3" strokeLinecap="round" />
        <path d="M8 9.2V10.2" stroke="#D42A1E" strokeWidth="1.3" strokeLinecap="round" />
      </g>
      <g className="status-lf-cross">
        <path d="M6.5 6.5L9.5 9.5" stroke="#D42A1E" strokeWidth="1.3" strokeLinecap="round" />
        <path d="M9.5 6.5L6.5 9.5" stroke="#D42A1E" strokeWidth="1.3" strokeLinecap="round" />
      </g>
    </svg>
  </span>
);

/** 维护中：齿轮旋转 */
const MaintainingIcon = () => (
  <span className="status-maintain-icon">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M12.1323 3.63629C14.2311 5.60449 14.6656 8.85537 13.0189 11.3305C11.789 13.1793 9.71098 14.1236 7.64351 13.9944L7.1978 14.6644L5.94892 13.8335L6.87389 12.4432L7.73714 12.4979C9.29308 12.5951 10.8491 11.884 11.7701 10.4997C12.6336 9.20174 12.7336 7.61778 12.1766 6.27784L9.82794 9.8082L5.66502 7.03871L8.01367 3.50835C6.5626 3.5124 5.1403 4.21666 4.2768 5.51462C3.35587 6.8989 3.30123 8.60888 3.99213 10.0064L4.37613 10.7815L3.45117 12.1718L2.20229 11.341L2.648 10.671C1.72994 8.81406 1.798 6.53252 3.02792 4.68378C4.67459 2.2086 7.84089 1.35327 10.4672 2.52849L7.74474 6.62068L9.40991 7.72847L12.1323 3.63629Z"
        fill="#EB8C33"
      />
    </svg>
  </span>
);

/** 待处理：三点依次闪烁 */
const PendingIcon = () => (
  <span className="status-pending-icon">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="8" cy="8" r="5.25" stroke="#858D99" strokeWidth="1.5" fill="none" />
      <circle className="status-dot1" cx="5.8" cy="8" r="0.7" fill="#858D99" />
      <circle className="status-dot2" cx="8" cy="8" r="0.7" fill="#858D99" />
      <circle className="status-dot3" cx="10.2" cy="8" r="0.7" fill="#858D99" />
    </svg>
  </span>
);

const STATUS_ICONS: Record<OpenClawStatus, () => ReactElement> = {
  creating: CreatingIcon,
  createFail: CreateFailIcon,
  running: RunningIcon,
  shutdown: ShutdownIcon,
  loading: LoadingIcon,
  loadFail: LoadFailIcon,
  maintaining: MaintainingIcon,
  pending: PendingIcon,
  rollingBack: LoadingIcon,
};

/**
 * 单独导出图标，方便其他位置（如详情页、列表）按需复用同一套视觉。
 */
export const StatusIcon = ({ status }: { status: OpenClawStatus }) => {
  const Icon = STATUS_ICONS[status];
  if (!Icon) return null;
  return <Icon />;
};

/** @deprecated 仅为向后兼容保留，请使用 StatusIcon */
export const StatusDot = StatusIcon;

export const StatusBadge = ({ status }: { status: OpenClawStatus }) => {
  const cfg = STATUS_VISUAL[status] ?? { label: status || "未知" };

  const badge = (
    <span
      className="inline-flex items-center whitespace-nowrap flex-shrink-0"
      style={{
        gap: "4px",
        padding: "2px 0",
        height: "20px",
        borderRadius: "2px",
        background: "transparent",
        color: "#020617",
        fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
        fontWeight: 400,
        fontSize: "12px",
        lineHeight: "20px",
      }}
    >
      <StatusIcon status={status} />
      {cfg.label}
    </span>
  );

  if (cfg.tooltipText && status !== "running") {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{badge}</div>
        </TooltipTrigger>
        <TooltipContent side="bottom" className="text-xs">
          {cfg.tooltipText}
        </TooltipContent>
      </Tooltip>
    );
  }

  return badge;
};
