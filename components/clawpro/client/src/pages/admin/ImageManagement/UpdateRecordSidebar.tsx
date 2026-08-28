/**
 * UpdateRecordSidebar - 镜像配置页右侧栏（精简版）
 *
 * 内容（自上而下）：
 *   1. Agent 类型锚点导航
 *   2. 镜像新版本提醒条：当存在"启用版本 ≠ 实例版本"的 Agent 类型时展示一行
 *      · 文案：OpenClaw 新版本 v2026.4.23 上线，可以更新（动态）
 *      · 复用 useOutdatedTypes 与 Agent 列表保持判断口径一致
 *   3. 「推送更新」按钮（触发 PushUpgradeDialog）
 *   4. 正在推送中的列表 + 撤回（仅在有推送时展示）
 *   5. 「查看全部更新记录」链接（触发 Dialog 弹窗）
 */
import { useEffect, useState } from "react";
import { Star, Megaphone, ChevronRight, Sparkles, RotateCcw } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { MetaText } from "@/components/ui/Typography";
import { toast } from "sonner";

import { useOutdatedTypes } from "../BatchUpdateNotice";
import {
  buildFlatUpdateRecords,
  type FlatUpdateRecord,
} from "./publicImageRecords";
import {
  listActivePushes,
  clearActivePush,
  type ActivePush,
} from "@/lib/upgradePushStore";

interface Props {
  /** 已展示的 Agent 类型列表（用于锚点） */
  views: { agentType: string; isEnabled: boolean; version: string | null }[];
  /** 类型 ID → 展示名 */
  getTypeLabel: (agentType: string) => string;
  /** 当前用户端首选类型 */
  defaultAgentType: string;
  /** 触发推送弹窗 */
  onPush: () => void;
  /** 触发"查看全部更新记录"弹窗 */
  onViewAll: () => void;
}

// ─── 公共镜像更新记录条目（兼容历史名称导出，便于其他文件复用） ───
export type UpdateRecord = FlatUpdateRecord;

/** 扁平化所有公共镜像的更新记录（按时间倒序） */
export function buildUpdateRecords(): UpdateRecord[] {
  return buildFlatUpdateRecords();
}

// ─── 主组件 ───────────────────────────────────────────────────
export default function UpdateRecordSidebar({
  views,
  getTypeLabel,
  defaultAgentType,
  onPush,
  onViewAll,
}: Props) {
  // 复用与 Agent 列表页相同的"是否有镜像更新"判断
  const outdatedTypes = useOutdatedTypes();
  const hasUpdate = outdatedTypes.length > 0;

  // 订阅活跃推送
  const [activePushes, setActivePushes] = useState<ActivePush[]>(() => listActivePushes());
  useEffect(() => {
    const refresh = () => setActivePushes(listActivePushes());
    window.addEventListener("upgrade-push-changed", refresh);
    window.addEventListener("storage", refresh);
    return () => {
      window.removeEventListener("upgrade-push-changed", refresh);
      window.removeEventListener("storage", refresh);
    };
  }, []);

  const handleRevoke = (push: ActivePush) => {
    clearActivePush(push.agentType);
    toast.success(`已撤回「${push.agentTypeLabel} v${push.version}」的推送提醒`);
  };

  return (
    <div className="space-y-5">
      {/* 1. Agent 类型导航 */}
      <div>
        <div className="text-xs font-semibold text-[var(--text-tertiary)] uppercase tracking-wide mb-2 px-3">
          Agent 类型
        </div>
        <nav className="flex flex-col gap-0.5">
          {views.map(({ agentType, isEnabled, version }) => {
            const isDef = defaultAgentType === agentType;
            return (
              <button
                key={agentType}
                onClick={() => {
                  const el = document.getElementById(`section-${agentType}`);
                  if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
                }}
                className="group flex items-center gap-1.5 px-3 py-2 rounded-[4px] text-left text-sm transition-colors text-[var(--text-secondary)] hover:bg-[var(--accent)]"
              >
                {isDef ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex items-center justify-center w-5 h-5 rounded-[4px] shrink-0 bg-[var(--cp-brand-blue)]">
                        <Star className="w-3 h-3 text-[var(--cp-brand-blue-foreground,#fff)]" />
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left" className="text-xs">
                      用户端首选 Agent 类型
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex items-center justify-center w-5 h-5 shrink-0">
                        <span
                          className={`w-1.5 h-1.5 rounded-full inline-block ${
                            isEnabled
                              ? "bg-[var(--alert-success-foreground,#16A34A)]"
                              : "bg-[var(--text-tertiary)]"
                          }`}
                        />
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left" className="text-xs">
                      {isEnabled ? `用户可见 v${version}` : "用户不可见"}
                    </TooltipContent>
                  </Tooltip>
                )}
                <span className="truncate">{getTypeLabel(agentType)}</span>
              </button>
            );
          })}
        </nav>
      </div>

      {/* 2. 镜像新版本提醒（仅在有更新时展示） */}
      {hasUpdate && (
        <div className="border-t border-border pt-4 px-3">
          <div className="rounded-[4px] border border-[var(--alert-warning-border,#FED7AA)] bg-[var(--alert-warning-bg)] px-3 py-2.5">
            <div className="flex items-start gap-1.5">
              <Sparkles className="w-3.5 h-3.5 text-[var(--alert-warning-foreground,#B45309)] mt-0.5 shrink-0" />
              <div className="flex-1 min-w-0">
                <div className="text-xs font-medium text-[var(--alert-warning-foreground,#B45309)] leading-snug">
                  有新版本上线
                </div>
                <ul className="mt-1 space-y-0.5">
                  {outdatedTypes.slice(0, 3).map((t) => (
                    <li
                      key={t.agentType}
                      className="text-[11px] text-[var(--alert-warning-foreground,#B45309)] leading-relaxed"
                    >
                      <span className="font-medium">{t.agentTypeLabel}</span>
                      <span className="ml-1 font-mono tabular-nums">
                        v{t.enabledVersion}
                      </span>
                    </li>
                  ))}
                </ul>
                {outdatedTypes.length > 3 && (
                  <div className="mt-1 text-[11px] text-[var(--alert-warning-foreground,#B45309)] opacity-80">
                    等 {outdatedTypes.length} 个 Agent 类型
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 3. 操作区：推送更新 + 正在推送列表 + 查看全部更新记录 */}
      <div className={hasUpdate ? "px-3" : "border-t border-border pt-4 px-3"}>
        <Button
          onClick={onPush}
          variant="claw-primary"
          size="sm"
          className="w-full h-8 gap-1.5 text-xs"
        >
          <Megaphone className="w-3.5 h-3.5" />
          推送更新
        </Button>

        {/* 正在推送中的列表（仅有推送时展示，每条带撤回） */}
        {activePushes.length > 0 && (
          <div className="mt-2 rounded-[4px] border border-border bg-[var(--accent)] px-2.5 py-2">
            <div className="flex items-center gap-1 text-[10px] font-semibold text-[var(--cp-brand-blue)] uppercase tracking-wide mb-1.5">
              <Megaphone className="w-2.5 h-2.5" />
              正在提醒用户更新
            </div>
            <ul className="space-y-1.5">
              {activePushes.map((p) => (
                <li
                  key={p.agentType}
                  className="flex items-center justify-between gap-2"
                >
                  <div className="min-w-0 flex-1">
                    <div className="text-[11px] font-medium text-[var(--text-title)] truncate">
                      {p.agentTypeLabel}
                    </div>
                    <MetaText as="div" className="text-[10px] font-mono tabular-nums truncate">
                      {formatVersion(p.version)} · {p.pushedAt.slice(0, 10)}
                    </MetaText>
                  </div>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => handleRevoke(p)}
                        variant="ghost"
                        size="sm"
                        className="h-6 px-1.5 gap-0.5 text-[10px] text-[var(--text-tertiary)] hover:text-[var(--text-danger)] hover:bg-[var(--alert-danger-bg)] shrink-0"
                      >
                        <RotateCcw className="w-2.5 h-2.5" />
                        撤回
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent side="left" className="text-xs max-w-[220px]">
                      撤回后用户端的"可更新"徽章将立即消失
                    </TooltipContent>
                  </Tooltip>
                </li>
              ))}
            </ul>
          </div>
        )}

        <Button
          onClick={onViewAll}
          variant="ghost"
          size="sm"
          className="w-full mt-2 h-8 justify-between px-3 text-xs text-[var(--text-secondary)]"
        >
          <span>查看全部更新记录</span>
          <ChevronRight className="w-3.5 h-3.5 text-[var(--text-tertiary)]" />
        </Button>
      </div>
    </div>
  );
}

// ─── 「查看全部」弹窗内的列表项导出，方便复用 ───────────────
export type { ActivePush };

/**
 * 统一展示版本号：
 *   - 已经带 v 前缀的（如 "v0.10.0"）保持不变
 *   - 不带 v 前缀的（如 "2026.4.23"）补一个 v
 */
export function formatVersion(version: string): string {
  if (!version) return "";
  return version.startsWith("v") || version.startsWith("V") ? version : `v${version}`;
}
