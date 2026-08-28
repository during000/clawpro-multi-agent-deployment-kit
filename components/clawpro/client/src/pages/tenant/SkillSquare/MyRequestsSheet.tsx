/**
 * 用户端 · 我的申请 抽屉
 *
 * 展示当前员工提交的「发布 / 下架」历史，按 skill 聚合，一 skill 一张卡。
 *
 * 聚合策略：
 * - 用 skillName 作为分组 key（Mock 无稳定 skillId）
 * - 每组取时间最新的一条 MyRequest 作为该 skill 的"最新态"
 * - 卡片展示最新态；点击"申请记录"图标可查看该 skill 的全量事件
 *
 * 历史查看模式（historyMode）：
 * - 'inline'（方案 A · 默认）：点击"申请记录"图标，历史 timeline 在卡片下方内联展开
 * - 'sheet'（方案 B）：点击"申请记录"图标，从右侧滑出二级 Sheet 承载 timeline
 *
 * 最新态 → 卡片行为映射：
 * - pending_admin  待管理员审核 · 琥珀 · 卡片不可点 · 显示【撤回】
 * - published      已发布       · 绿   · 卡片可点（更新流程）· 显示【下架】
 * - rejected       已驳回       · 红   · 卡片可点（重新提交）· 展示驳回原因
 * - withdrawn      已撤回       · 灰   · 卡片可点（重新提交）
 */

import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { History, RefreshCw, Search, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet';
import {
  CardTitle,
  MetaText,
  CompactText,
} from '@/components/ui/Typography';
import {
  useMyRequests,
  useSkillRequestHistory,
  withdrawRequest,
  REQUEST_STATUS_LABEL,
  REQUEST_TYPE_LABEL,
  type MyRequest,
  type MyRequestStatus,
} from './myRequestsStore';

export type HistoryMode = 'inline' | 'sheet';

interface MyRequestsSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * 历史记录查看模式，默认 'inline'（方案 A：卡片内联展开）
   */
  historyMode?: HistoryMode;
  /**
   * 卡片点击回调（published / rejected / withdrawn 三态可触发）。
   */
  onCardClick?: (latest: MyRequest) => void;
  /**
   * 点击【下架】按钮时触发（仅 published 卡片有）。
   */
  onOffshelfClick?: (latest: MyRequest) => void;
}

/** 状态徽标配色（走 Tailwind color-name，未硬编码色值） */
const STATUS_TONE: Record<MyRequestStatus, string> = {
  pending_admin: 'bg-amber-50 text-amber-700 border-amber-100',
  published: 'bg-green-50 text-green-700 border-green-100',
  rejected: 'bg-red-50 text-red-600 border-red-100',
  withdrawn: 'bg-gray-100 text-gray-500 border-gray-200',
};

/** timeline 圆点配色（浅色，与徽标呼应） */
const STATUS_DOT: Record<MyRequestStatus, string> = {
  pending_admin: 'bg-amber-400',
  published: 'bg-green-500',
  rejected: 'bg-red-400',
  withdrawn: 'bg-gray-400',
};

function formatSubmittedAt(iso: string): string {
  const d = new Date(iso);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  return `${y}-${m}-${day} ${hh}:${mm}`;
}

/**
 * 按 skill 聚合：返回每个 skillName 的最新一条 MyRequest。
 * 排序规则：时间倒序（最新 skill 在最上面）。
 */
function aggregateBySkill(requests: MyRequest[]): MyRequest[] {
  const map = new Map<string, MyRequest>();
  for (const r of requests) {
    const existing = map.get(r.skillName);
    if (!existing || new Date(r.submittedAt) > new Date(existing.submittedAt)) {
      map.set(r.skillName, r);
    }
  }
  return Array.from(map.values()).sort(
    (a, b) => new Date(b.submittedAt).getTime() - new Date(a.submittedAt).getTime(),
  );
}

/** 单条历史事件行（timeline 中的一行） */
function HistoryTimelineRow({ event }: { event: MyRequest }) {
  return (
    <div className="flex items-start gap-2 py-1.5">
      <span
        className={`mt-1.5 h-2 w-2 flex-shrink-0 rounded-full ${STATUS_DOT[event.status]}`}
      />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <CompactText className="text-[var(--text-emphasis)]">
            {REQUEST_STATUS_LABEL[event.status]}
          </CompactText>
          <MetaText>· {REQUEST_TYPE_LABEL[event.type]}</MetaText>
          <MetaText>· {formatSubmittedAt(event.submittedAt)}</MetaText>
        </div>
        {event.status === 'rejected' && event.reason && (
          <MetaText className="mt-0.5 block">驳回原因：{event.reason}</MetaText>
        )}
        {event.type === 'offshelf' && event.offshelfReason && (
          <MetaText className="mt-0.5 block">下架原因：{event.offshelfReason}</MetaText>
        )}
      </div>
    </div>
  );
}

/** 内联历史列表（方案 A 用） */
function InlineHistoryList({ skillName }: { skillName: string }) {
  const history = useSkillRequestHistory(skillName);
  if (history.length === 0) return null;
  return (
    <div className="mt-2 rounded-[6px] border border-[var(--border)] bg-[var(--bg-page,#F7F8FA)] px-3 py-2">
      <MetaText className="block mb-1">申请记录（{history.length} 条）</MetaText>
      <div className="divide-y divide-[var(--border)]">
        {history.map((e) => (
          <HistoryTimelineRow key={e.id} event={e} />
        ))}
      </div>
    </div>
  );
}

function RequestCard({
  latest,
  historyMode,
  onCardClick,
  onOffshelfClick,
  onOpenHistorySheet,
}: {
  latest: MyRequest;
  historyMode: HistoryMode;
  onCardClick?: (latest: MyRequest) => void;
  onOffshelfClick?: (latest: MyRequest) => void;
  onOpenHistorySheet: (skillName: string) => void;
}) {
  const [inlineOpen, setInlineOpen] = useState(false);

  const canWithdraw = latest.status === 'pending_admin';
  const canOffshelf = latest.status === 'published' && latest.type === 'publish';
  const cardClickable =
    latest.status === 'published' ||
    latest.status === 'rejected' ||
    latest.status === 'withdrawn';

  const handleWithdraw = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (withdrawRequest(latest.id)) {
      toast.success('申请已撤回');
    }
  };

  const handleOffshelf = (e: React.MouseEvent) => {
    e.stopPropagation();
    onOffshelfClick?.(latest);
  };

  const handleHistoryClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (historyMode === 'inline') {
      setInlineOpen((v) => !v);
    } else {
      onOpenHistorySheet(latest.skillName);
    }
  };

  const handleCardClick = () => {
    if (cardClickable) {
      onCardClick?.(latest);
    }
  };

  return (
    <div
      role={cardClickable ? 'button' : undefined}
      tabIndex={cardClickable ? 0 : undefined}
      onClick={handleCardClick}
      onKeyDown={(e) => {
        if (cardClickable && (e.key === 'Enter' || e.key === ' ')) {
          e.preventDefault();
          handleCardClick();
        }
      }}
      className={[
        'flex flex-col gap-2 rounded-[var(--radius-lg)] border border-[var(--border)] bg-white p-4',
        cardClickable
          ? 'cursor-pointer transition-shadow hover:shadow-sm hover:border-[var(--tenant-primary,#1F6BFF)]'
          : '',
      ].join(' ')}
    >
      {/* 标题行：技能名（上） + slug（下） · 右侧类型 & 申请记录入口 */}
      <div className="flex items-start gap-2 min-w-0">
        <div className="flex-1 min-w-0">
          <CardTitle className="truncate" title={latest.skillName}>
            {latest.skillName}
          </CardTitle>
          {latest.skillSlug && (
            <div
              className="truncate mt-0.5 text-[12px] leading-4 text-[var(--text-muted)] font-mono"
              title={latest.skillSlug}
            >
              {latest.skillSlug}
            </div>
          )}
        </div>
        <div className="flex items-center gap-1.5 flex-shrink-0 pt-0.5">
          {cardClickable && (
            <>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onCardClick?.(latest);
                }}
                className="inline-flex items-center rounded px-1.5 py-0.5 text-[12px] text-[var(--text-muted)] hover:text-[var(--tenant-primary,#1F6BFF)] hover:bg-[var(--bg-page,#F7F8FA)]"
                title={
                  latest.status === 'published'
                    ? '更新技能内容'
                    : '重新提交此技能'
                }
              >
                <span>
                  {latest.status === 'published' ? '更新技能' : '重新提交'}
                </span>
              </button>
              <span className="text-[var(--text-muted)] text-[12px]">·</span>
            </>
          )}
          <button
            type="button"
            onClick={handleHistoryClick}
            className={[
              'inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[12px]',
              historyMode === 'inline' && inlineOpen
                ? 'text-[var(--tenant-primary,#1F6BFF)] bg-[var(--bg-page,#F7F8FA)]'
                : 'text-[var(--text-muted)] hover:text-[var(--tenant-primary,#1F6BFF)] hover:bg-[var(--bg-page,#F7F8FA)]',
            ].join(' ')}
            title={
              historyMode === 'inline'
                ? inlineOpen
                  ? '收起申请记录'
                  : '展开申请记录'
                : '查看申请记录'
            }
          >
            <History className="h-3.5 w-3.5" />
            <span>申请记录</span>
          </button>
        </div>
      </div>

      {/* 状态 + 时间 */}
      <div className="flex items-center gap-2 flex-wrap">
        <span
          className={`inline-flex items-center h-[22px] px-2 rounded border text-xs ${STATUS_TONE[latest.status]}`}
        >
          {REQUEST_STATUS_LABEL[latest.status]}
        </span>
        <MetaText>{formatSubmittedAt(latest.submittedAt)}</MetaText>
      </div>

      {/* 驳回原因（仅 rejected） */}
      {latest.status === 'rejected' && latest.reason && (
        <div className="rounded-[6px] border border-red-100 bg-red-50 px-3 py-2">
          <CompactText className="text-[var(--text-body)]">
            驳回原因：{latest.reason}
          </CompactText>
        </div>
      )}

      {/* 下架原因回显 */}
      {latest.type === 'offshelf' && latest.offshelfReason && (
        <CompactText className="text-[var(--text-muted)]">
          下架原因：{latest.offshelfReason}
        </CompactText>
      )}

      {/* 操作区：撤回 / 下架 */}
      {(canWithdraw || canOffshelf) && (
        <div className="flex justify-end pt-1 gap-2">
          {canOffshelf && (
            <Button variant="tenant-outline" size="claw-sm" onClick={handleOffshelf}>
              下架
            </Button>
          )}
          {canWithdraw && (
            <Button variant="tenant-outline" size="claw-sm" onClick={handleWithdraw}>
              撤回
            </Button>
          )}
        </div>
      )}

      {/* 方案 A：内联展开 */}
      {historyMode === 'inline' && inlineOpen && (
        <div onClick={(e) => e.stopPropagation()}>
          <InlineHistoryList skillName={latest.skillName} />
        </div>
      )}
    </div>
  );
}

/** 方案 B：二级 Sheet · 展示某 skill 的全量申请事件 */
function SkillHistorySheet({
  skillName,
  open,
  onOpenChange,
}: {
  skillName: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const history = useSkillRequestHistory(skillName ?? '');
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-[min(380px,92%)] sm:max-w-[380px] p-0 flex flex-col gap-0"
      >
        <SheetHeader className="border-b border-[var(--border)] p-4 gap-1">
          <SheetTitle className="text-[16px] leading-6 text-[var(--text-emphasis)]">
            申请记录
          </SheetTitle>
          <SheetDescription>{skillName ?? ''}</SheetDescription>
        </SheetHeader>
        <div className="flex-1 min-h-0 overflow-y-auto p-4 bg-[var(--bg-page,#F7F8FA)]">
          {history.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <MetaText>暂无记录</MetaText>
            </div>
          ) : (
            <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-white px-4 py-2 divide-y divide-[var(--border)]">
              {history.map((e) => (
                <HistoryTimelineRow key={e.id} event={e} />
              ))}
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

export default function MyRequestsSheet({
  open,
  onOpenChange,
  historyMode = 'inline',
  onCardClick,
  onOffshelfClick,
}: MyRequestsSheetProps) {
  const requests = useMyRequests();
  const aggregated = aggregateBySkill(requests);

  const [historySheetOpen, setHistorySheetOpen] = useState(false);
  const [historySheetSkill, setHistorySheetSkill] = useState<string | null>(null);

  // 搜索关键词（模糊匹配 skillName + skillSlug，大小写不敏感）
  const [query, setQuery] = useState('');
  // 抽屉关闭时清空搜索，避免下次打开保留上次筛选状态
  useEffect(() => {
    if (!open) setQuery('');
  }, [open]);
  // 刷新按钮（Mock 阶段做视觉转动即可；数据源本就是 store 全量，无需真拉取）
  const [refreshing, setRefreshing] = useState(false);
  const handleRefresh = () => {
    if (refreshing) return;
    setRefreshing(true);
    // Mock 阶段：短暂转动一下以给出交互反馈；接入后端后替换为真实 fetch
    setTimeout(() => {
      setRefreshing(false);
      toast.success('已刷新申请进度');
    }, 500);
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return aggregated;
    return aggregated.filter((r) => {
      const name = r.skillName?.toLowerCase() ?? '';
      const slug = r.skillSlug?.toLowerCase() ?? '';
      return name.includes(q) || slug.includes(q);
    });
  }, [aggregated, query]);

  const handleOpenHistorySheet = (skillName: string) => {
    setHistorySheetSkill(skillName);
    setHistorySheetOpen(true);
  };

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent
          side="right"
          className="w-[min(430px,92%)] sm:max-w-[430px] p-0 flex flex-col gap-0"
        >
          <SheetHeader className="border-b border-[var(--border)] p-4 gap-1">
            <div className="flex items-center gap-2">
              <SheetTitle className="text-[16px] leading-6 text-[var(--text-emphasis)]">
                我的申请
              </SheetTitle>
              <button
                type="button"
                onClick={handleRefresh}
                disabled={refreshing}
                title="刷新申请进度"
                aria-label="刷新申请进度"
                className="flex-shrink-0 inline-flex items-center justify-center h-6 w-6 rounded text-[var(--text-muted)] hover:text-[var(--tenant-primary,#1F6BFF)] hover:bg-[var(--bg-page,#F7F8FA)] disabled:opacity-50"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${refreshing ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {/* 搜索框：模糊匹配技能名称 / slug */}
            <div className="relative mt-2">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-[var(--text-muted)] pointer-events-none" />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜索技能名称或 slug"
                className="h-8 pl-8 pr-8 text-[13px]"
              />
              {query && (
                <button
                  type="button"
                  onClick={() => setQuery('')}
                  title="清空搜索"
                  className="absolute right-1.5 top-1/2 -translate-y-1/2 inline-flex items-center justify-center h-5 w-5 rounded text-[var(--text-muted)] hover:text-[var(--text-emphasis)] hover:bg-[var(--bg-page,#F7F8FA)]"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
          </SheetHeader>

          <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-3 bg-[var(--bg-page,#F7F8FA)]">
            {filtered.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16">
                <MetaText>
                  {aggregated.length === 0
                    ? '暂无申请'
                    : `没有匹配"${query}"的申请`}
                </MetaText>
              </div>
            ) : (
              filtered.map((latest) => (
                <RequestCard
                  key={latest.skillName}
                  latest={latest}
                  historyMode={historyMode}
                  onCardClick={onCardClick}
                  onOffshelfClick={onOffshelfClick}
                  onOpenHistorySheet={handleOpenHistorySheet}
                />
              ))
            )}
          </div>
        </SheetContent>
      </Sheet>

      {/* 方案 B：二级历史 Sheet（仅 historyMode='sheet' 时会被触发） */}
      {historyMode === 'sheet' && (
        <SkillHistorySheet
          skillName={historySheetSkill}
          open={historySheetOpen}
          onOpenChange={setHistorySheetOpen}
        />
      )}
    </>
  );
}
