/**
 * RecentHistorySection - 最近运维记录卡片（可复用）
 *
 * 用途：
 *   - 「Agent 版本管理」Tab 底部展示最近 5 条 agent-upgrade
 *   - 「命令下发」Tab 底部展示最近 5 条 command-execute
 *
 * 设计原则：与「运维任务」Tab 的列表风格保持一致（同结构，只是少了筛选 + 限 5 条）
 */
import { useMemo } from "react";
import {
  Sparkles, ExternalLink, Terminal, Server,
  CheckCircle2, XCircle, Loader2, ArrowUpRight, Code2, FlaskConical, User,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Empty, EmptyHeader, EmptyDescription, EmptyMedia } from "@/components/ui/empty";
import {
  MOCK_HISTORY,
  HISTORY_ACTION_LABEL,
  type HistoryAction,
  type HistoryRecord,
} from "../mockData";

interface Props {
  /** 过滤的动作类型；不传则显示全部 */
  action: HistoryAction;
  /** 最多展示几条，默认 5 */
  limit?: number;
  /** 卡片标题 */
  title?: string;
  /** 卡片描述 */
  description?: string;
  /** 卡片左侧图标颜色覆盖（默认蓝色） */
  iconColor?: string;
}

export default function RecentHistorySection({
  action,
  limit = 5,
  title,
  description,
  iconColor = "text-[#355EF1]",
}: Props) {
  const records = useMemo<HistoryRecord[]>(() => {
    return MOCK_HISTORY.filter((h) => h.action === action).slice(0, limit);
  }, [action, limit]);

  const goToHistoryTab = () => {
    // 按 action 分流到对应 Tab：升级类→Agent 版本/更新记录；命令类→命令下发/执行记录
    const target = action === "command-execute"
      ? "/admin/agent-commands?tab=history"
      : "/admin/agent-versions?tab=history";
    window.history.pushState(null, "", target);
    window.dispatchEvent(new PopStateEvent("popstate"));
  };

  // 默认 title / description（按 action 类型推断）
  const defaultTitle = action === "agent-upgrade" ? "最近更新记录" : "最近执行记录";
  const defaultDescription =
    action === "agent-upgrade"
      ? "展示最近 5 条 Agent 更新任务，完整记录请前往「运维任务」查看"
      : "展示最近 5 条命令执行情况，完整记录请前往「运维任务」查看";

  const TitleIcon = action === "agent-upgrade" ? Server : Terminal;

  return (
    <section
      className="bg-white rounded-2xl border border-gray-100 overflow-hidden"
      style={{ boxShadow: "0 1px 3px rgba(0,0,0,0.06), 0 4px 12px rgba(0,0,0,0.04)" }}
    >
      <div className="px-6 py-4 border-b border-gray-50 flex items-center gap-3">
        <TitleIcon className={`w-4 h-4 ${iconColor}`} />
        <div className="flex-1">
          <h2 className="font-semibold text-[#0A0A0A] text-base">{title ?? defaultTitle}</h2>
          <p className="text-xs text-[#737373] mt-0.5">{description ?? defaultDescription}</p>
        </div>
        <Button variant="outline" size="sm" className="h-8 text-xs" onClick={goToHistoryTab}>
          <ExternalLink className="w-3.5 h-3.5 mr-1" />
          查看完整记录
        </Button>
      </div>

      {records.length === 0 ? (
        <Empty className="border-0 py-12">
          <EmptyMedia />
          <EmptyHeader>
            <EmptyDescription>暂无记录</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[14%]">
                任务 ID
              </TableHead>
              <TableHead className="w-[28%]">
                {action === "agent-upgrade" ? "更新内容" : "命令"}
              </TableHead>
              <TableHead className="w-[14%]">
                执行人
              </TableHead>
              <TableHead className="w-[16%]">
                执行时间
              </TableHead>
              <TableHead>
                结果
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {records.map((r) => (
              <RecordRow key={r.id} record={r} />
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  );
}

function RecordRow({ record }: { record: HistoryRecord }) {
  const successRate = record.totalInstances > 0
    ? Math.round((record.successCount / record.totalInstances) * 100)
    : 0;

  return (
    <TableRow>
      <TableCell>
        <span className="text-xs font-mono text-[#737373] bg-gray-50 px-1.5 py-0.5 rounded select-all">
          {record.taskId}
        </span>
      </TableCell>

      {/* 内容（按类型动态展示） */}
      <TableCell>
        {record.action === "agent-upgrade" && (
          <div className="space-y-0.5">
            <div className="text-sm text-[#0A0A0A] truncate max-w-[280px] flex items-center gap-1.5">
              <ArrowUpRight className="w-3.5 h-3.5 text-[#355EF1] shrink-0" />
              {record.assetName}
            </div>
            {record.fromVersion && record.toVersion ? (
              <div className="text-xs font-mono text-[#737373] tabular-nums pl-5">
                {record.fromVersion} <span className="text-[#A3A3A3]">→</span>{" "}
                <span className="text-[#334155] font-semibold">{record.toVersion}</span>
              </div>
            ) : record.toVersion ? (
              <div className="text-xs font-mono text-[#737373] tabular-nums pl-5">
                → <span className="text-[#334155] font-semibold">{record.toVersion}</span>
              </div>
            ) : null}
          </div>
        )}
        {record.action === "command-execute" && (
          <div className="space-y-0.5">
            <div className="text-sm text-[#0A0A0A] truncate max-w-[280px] flex items-center gap-1.5">
              <Code2 className="w-3.5 h-3.5 text-purple-500 shrink-0" />
              {record.assetName}
              {record.commandExtra?.testInstanceId && (
                <span className="ml-0.5 inline-flex items-center text-[10px] font-medium px-1.5 py-0.5 rounded bg-amber-50 text-amber-700 align-middle shrink-0">
                  <FlaskConical className="w-2.5 h-2.5 mr-0.5" />
                  灰度机
                </span>
              )}
            </div>
            {record.commandExtra?.commandContent && (
              <code className="text-xs font-mono text-[#737373] truncate block max-w-[280px] pl-5">
                {record.commandExtra.commandContent.split("\n")[0]}
                {record.commandExtra.commandContent.includes("\n") && (
                  <span className="text-[#A3A3A3] ml-1">…</span>
                )}
              </code>
            )}
          </div>
        )}
      </TableCell>

      {/* 执行人 */}
      <TableCell>
        {record.isAuto ? (
          <div className="inline-flex items-center gap-1.5 text-sm">
            <Sparkles className="w-3.5 h-3.5 text-[#355EF1]" />
            <span className="text-[#355EF1] truncate max-w-[120px]">{record.operator}</span>
          </div>
        ) : (
          <div className="inline-flex items-center gap-1.5 text-sm">
            <User className="w-3.5 h-3.5 text-[#A3A3A3]" />
            <span className="text-[#334155] truncate max-w-[120px]">{record.operator}</span>
          </div>
        )}
      </TableCell>

      {/* 执行时间 */}
      <TableCell className="text-sm text-[#737373] tabular-nums whitespace-nowrap">
        {record.operatedAt}
      </TableCell>

      {/* 结果 */}
      <TableCell>
        <div className="flex items-center gap-2">
          {record.failedCount === 0 ? (
            <CheckCircle2 className="w-4 h-4 text-green-500" />
          ) : successRate > 0 ? (
            <Loader2 className="w-4 h-4 text-amber-500" />
          ) : (
            <XCircle className="w-4 h-4 text-red-500" />
          )}
          <span className="text-xs text-[#334155] font-mono tabular-nums">
            {record.successCount}/{record.totalInstances}
            {record.failedCount > 0 && (
              <span className="text-red-600 ml-1">· {record.failedCount} 失败</span>
            )}
          </span>
        </div>
      </TableCell>
    </TableRow>
  );
}

// 让 HistoryAction Label 不被 tree-shake 提示（仅类型用，不需要导出 label）
void HISTORY_ACTION_LABEL;
