/**
 * Logs/index.tsx - 审计日志
 * 真实接口尚未对接（仍为占位），但当上层注入 mockLogs 时，渲染一份贴合最终视觉的 mock 表格用于设计走查。
 */
import React, { useMemo, useState } from 'react';
import { Search, RefreshCw, CircleAlert } from 'lucide-react';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from '@/components/ui/table';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyDescription,
} from '@/components/ui/empty';
import { SurfaceCard } from '@/components/ui/Surface';
import { Pagination } from '@/components/ui/pagination';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';

interface LogsIndexProps {
  aiAgentHostList?: any[];
  isGetAllMachinesLoading?: boolean;
  openAssetDetail?: (item: any) => void;
  isHideLogTalkTab?: boolean;
  from?: string;
  InstanceIds?: string[];
  /** 仅设计走查/演示用 */
  mockLogs?: any[];
}

const RISK_LEVEL_TEXT: Record<string, string> = {
  critical: '严重',
  high: '高危',
  medium: '中危',
  low: '低危',
};

/** 风险等级 → 无底色的彩色文字（规范 §16 StatusTag mode="text"） */
const renderRiskLevel = (level: string) => {
  const text = RISK_LEVEL_TEXT[level] || '低危';
  let color = '#16A34A'; // 低危 - 绿
  if (level === 'critical' || level === 'high') color = '#DC2626'; // 高/严重 - 红
  else if (level === 'medium') color = '#D97706'; // 中 - 橙
  return (
    <span className="text-sm font-medium" style={{ color }}>
      {text}
    </span>
  );
};

/** 处置结果 → 普通文字 */
const renderResult = (result: string) => {
  return (
    <span className="text-xs whitespace-nowrap text-[var(--text-title)]">
      {result || '-'}
    </span>
  );
};

export default function LogsIndex({ mockLogs }: LogsIndexProps) {
  const [searchText, setSearchText] = useState('');
  const [riskLevelFilter, setRiskLevelFilter] = useState<string>('all');
  const [actionFilter, setActionFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const [tab, setTab] = useState<'behavior' | 'talk'>('talk');
  const pageSize = 10;

  const hasMock = !!(mockLogs && mockLogs.length);

  const filtered = useMemo(() => {
    if (!hasMock) return [];
    let list = mockLogs as any[];
    if (riskLevelFilter !== 'all') {
      list = list.filter(d => d.RiskLevel === riskLevelFilter);
    }
    if (actionFilter !== 'all') {
      list = list.filter(d => d.Action === actionFilter);
    }
    if (searchText.trim()) {
      const kw = searchText.trim();
      list = list.filter(
        d =>
          (d.Prompt || '').indexOf(kw) >= 0 ||
          (d.AgentName || '').indexOf(kw) >= 0 ||
          (d.ToolName || '').indexOf(kw) >= 0,
      );
    }
    return list;
  }, [mockLogs, hasMock, riskLevelFilter, actionFilter, searchText]);

  // 未注入 mock：仍渲染完整页面骨架（横幅、Segment、操作栏、空表格），只在表格区显示空态
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);

  return (
    <div>
      {/* 提示横幅 */}
      <Alert variant="warning" className="mb-3">
        <CircleAlert />
        <AlertDescription>日志存储已满，新增审计日志将不再更新，请及时扩容。</AlertDescription>
      </Alert>
      <Alert variant="info" className="mb-3">
        <AlertInfoIcon />
        <AlertDescription>为减少信息干扰，系统默认过滤高频常规命令的执行日志。（ip neigh show）</AlertDescription>
      </Alert>

      {/* 行为审计 / 对话审计 tab + 操作栏（同一行）
          停服态豁免：审计日志 tab 切换、筛选下拉、关键字搜索、刷新按钮均属
          「查看类」操作，只改本地 state / 前端翻页，不发写请求，停服时应保持可用。
          分别给 SegmentGroup 与右侧筛选容器打 data-billing-exempt，
          overlay 灰化CSS 与点击拦截同时放行；组件自身若传入 disabled，
          仍由原生 disabled 生效（延续既有禁用）。*/}
      <div className="flex items-center justify-between gap-2 mb-4">
        <SegmentGroup aria-label="审计日志类型切换" data-billing-exempt>
          <SegmentOption
            active={tab === 'behavior'}
            onClick={() => setTab('behavior')}
          >
            行为审计
          </SegmentOption>
          <SegmentOption
            active={tab === 'talk'}
            onClick={() => setTab('talk')}
          >
            对话审计
          </SegmentOption>
        </SegmentGroup>

        {tab === 'talk' && (
          <div className="flex items-center gap-2" data-billing-exempt>
            <Select value={actionFilter} onValueChange={(v) => { setActionFilter(v); setPage(1); }}>
              <SelectTrigger size="sm" className="w-[140px]">
                <SelectValue placeholder="动作类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部动作</SelectItem>
                <SelectItem value="completion">大模型对话</SelectItem>
                <SelectItem value="tool_call">工具调用</SelectItem>
              </SelectContent>
            </Select>
            <Select value={riskLevelFilter} onValueChange={(v) => { setRiskLevelFilter(v); setPage(1); }}>
              <SelectTrigger size="sm" className="w-[140px]">
                <SelectValue placeholder="风险等级" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部等级</SelectItem>
                <SelectItem value="critical">严重</SelectItem>
                <SelectItem value="high">高危</SelectItem>
                <SelectItem value="medium">中危</SelectItem>
                <SelectItem value="low">低危</SelectItem>
              </SelectContent>
            </Select>
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#A3A3A3]" />
              <Input
                placeholder="搜索 Prompt / Agent / 工具"
                value={searchText}
                onChange={(e) => { setSearchText(e.target.value); setPage(1); }}
                className="pl-8 h-8 w-[220px]"
              />
            </div>
            <button
              onClick={() => setPage(1)}
              className="w-8 h-8 flex items-center justify-center rounded-[4px] border border-[#E5E5E5] bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
              title="刷新"
            >
              <RefreshCw className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </div>

      {tab === 'talk' ? (
        <>

      <SurfaceCard className="overflow-hidden">
      {paged.length > 0 ? (
        <>
      <Table scrollX={1200} variant="white" containerClassName="bg-white">
        <TableHeader>
          <TableRow>
            <TableHead className="w-[140px]">时间</TableHead>
            <TableHead className="w-[180px]">AI Agent / 模型</TableHead>
            <TableHead className="w-[110px]">动作</TableHead>
            <TableHead className="w-[110px]">工具</TableHead>
            <TableHead>Prompt / 请求</TableHead>
            <TableHead className="w-[140px]">风险</TableHead>
            <TableHead className="w-[90px]">等级</TableHead>
            <TableHead className="w-[110px]">处置结果</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {paged.map((item: any) => (
            <TableRow key={item.Id}>
              <TableCell className="whitespace-nowrap">{item.Time}</TableCell>
              <TableCell>
                <div className="min-w-0">
                  <div className="truncate text-[#0A0A0A]">{item.AgentName}</div>
                  <div className="truncate text-[#737373]">{item.Model}</div>
                </div>
              </TableCell>
              <TableCell>
                <span className="text-[#525252]">
                  {item.Action === 'tool_call' ? '工具调用' : '大模型对话'}
                </span>
              </TableCell>
              <TableCell>{item.ToolName}</TableCell>
              <TableCell>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="block truncate max-w-[420px] text-[#525252]">
                      {item.Prompt}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-md break-all">{item.Prompt}</TooltipContent>
                </Tooltip>
              </TableCell>
              <TableCell>{item.Risk}</TableCell>
              <TableCell>{renderRiskLevel(item.RiskLevel)}</TableCell>
              <TableCell>{renderResult(item.Result)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {/* 表格页脚：数量统计左对齐，分页器右对齐 */}
      <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]">
        <span className="justify-self-start text-sm leading-[1.5] text-[#737373]">
          共 {filtered.length} 条记录
        </span>
        <Pagination
          total={filtered.length}
          current={page}
          pageSize={pageSize}
          className="justify-self-end justify-end flex-nowrap"
          onChange={(p) => setPage(p)}
        />
      </div>
        </>
      ) : (
        <Empty className="border-0 py-20">
          <EmptyHeader>
            <EmptyMedia />
            <EmptyDescription>
              {hasMock ? '暂无符合条件的记录' : '暂无审计日志'}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      </SurfaceCard>
        </>
      ) : (
        <SurfaceCard className="overflow-hidden">
          <Empty className="border-0 py-20">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyDescription>
                行为审计日志即将开放
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </SurfaceCard>
      )}
    </div>
  );
}
