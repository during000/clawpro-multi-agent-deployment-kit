/**
 * 调用日志 Tab
 *
 * 展示 MCP 工具调用记录：时间、用户、工具名、参数摘要、结果、耗时
 * 第一期使用 mock 数据
 */

import { useState, useMemo } from 'react';
import { Search, CheckCircle2, XCircle, Clock } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { StatusTag } from '@/components/ui/status-tag';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import { MetaText, HelperText } from '@/components/ui/Typography';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';

interface MockCallLog {
  id: string;
  timestamp: string;
  userName: string;
  toolName: string;
  module: string;
  paramsSummary: string;
  result: 'success' | 'failed' | 'denied';
  durationMs: number;
}

const MOCK_LOGS: MockCallLog[] = [
  { id: '1', timestamp: '2026-07-14 14:32:18', userName: 'petzhou', toolName: 'list_skills', module: '企业技能库', paramsSummary: 'page=1, page_size=20', result: 'success', durationMs: 230 },
  { id: '2', timestamp: '2026-07-14 14:30:05', userName: 'petzhou', toolName: 'distribute_skill', module: '企业技能库', paramsSummary: 'slug=clawpro-requirement-writer, instance_ids=[9179]', result: 'success', durationMs: 1520 },
  { id: '3', timestamp: '2026-07-14 14:28:42', userName: 'petzhou', toolName: 'list_instances', module: 'Agent 实例', paramsSummary: 'page=1, page_size=10', result: 'success', durationMs: 180 },
  { id: '4', timestamp: '2026-07-14 14:25:33', userName: 'petzhou', toolName: 'get_org_usage', module: '用量监控', paramsSummary: 'start_date=2026-07-01, end_date=2026-07-14', result: 'success', durationMs: 890 },
  { id: '5', timestamp: '2026-07-14 14:20:11', userName: 'petzhou', toolName: 'delete_skill', module: '企业技能库', paramsSummary: 'slug=old-test-skill', result: 'denied', durationMs: 45 },
  { id: '6', timestamp: '2026-07-14 14:15:07', userName: 'petzhou', toolName: 'create_skill', module: '企业技能库', paramsSummary: 'slug=clawpro-mcp-test, version=1.0.0, zip=850B', result: 'success', durationMs: 3200 },
  { id: '7', timestamp: '2026-07-14 14:10:52', userName: 'petzhou', toolName: 'list_mcp_services', module: '企业 MCP 库', paramsSummary: '', result: 'success', durationMs: 150 },
  { id: '8', timestamp: '2026-07-14 14:05:38', userName: 'petzhou', toolName: 'get_usage_guide', module: '使用指南', paramsSummary: '', result: 'success', durationMs: 12 },
  { id: '9', timestamp: '2026-07-14 13:58:21', userName: 'petzhou', toolName: 'distribute_mcp_service', module: '企业 MCP 库', paramsSummary: 'service_id=computing-center-clawpro, instance_ids=[9179]', result: 'failed', durationMs: 5000 },
  { id: '10', timestamp: '2026-07-14 13:45:14', userName: 'petzhou', toolName: 'query_audit_log', module: '操作审计', paramsSummary: 'page=1, page_size=20', result: 'success', durationMs: 340 },
  { id: '11', timestamp: '2026-07-14 13:30:02', userName: 'petzhou', toolName: 'get_asset_detail', module: '资产管理', paramsSummary: 'target_type=group, target_id=g-100', result: 'success', durationMs: 210 },
  { id: '12', timestamp: '2026-07-14 13:20:47', userName: 'petzhou', toolName: 'list_users', module: '用户管理', paramsSummary: 'page=1, page_size=20', result: 'success', durationMs: 280 },
];

const RESULT_CONFIG = {
  success: { label: '成功', variant: 'green' as const, icon: CheckCircle2 },
  failed: { label: '失败', variant: 'red' as const, icon: XCircle },
  denied: { label: '拒绝', variant: 'orange' as const, icon: Clock },
};

export default function CallLogsTab() {
  const [searchQuery, setSearchQuery] = useState('');
  const [filter, setFilter] = useState<'all' | 'success' | 'failed' | 'denied'>('all');

  const filteredLogs = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return MOCK_LOGS.filter(log => {
      if (filter !== 'all' && log.result !== filter) return false;
      if (q && !log.toolName.toLowerCase().includes(q) && !log.userName.toLowerCase().includes(q) && !log.module.toLowerCase().includes(q)) {
        return false;
      }
      return true;
    });
  }, [searchQuery, filter]);

  const stats = useMemo(() => ({
    total: MOCK_LOGS.length,
    success: MOCK_LOGS.filter(l => l.result === 'success').length,
    failed: MOCK_LOGS.filter(l => l.result === 'failed').length,
    denied: MOCK_LOGS.filter(l => l.result === 'denied').length,
  }), []);

  return (
    <div className="space-y-4">
      {/* 统计卡片 */}
      <div className="grid grid-cols-4 gap-3">
        <div className="bg-white border border-gray-200 rounded p-3">
          <MetaText className="text-gray-500">总调用</MetaText>
          <div className="text-xl font-semibold text-gray-900 mt-1">{stats.total}</div>
        </div>
        <div className="bg-white border border-gray-200 rounded p-3">
          <MetaText className="text-gray-500">成功</MetaText>
          <div className="text-xl font-semibold text-green-600 mt-1">{stats.success}</div>
        </div>
        <div className="bg-white border border-gray-200 rounded p-3">
          <MetaText className="text-gray-500">失败</MetaText>
          <div className="text-xl font-semibold text-red-600 mt-1">{stats.failed}</div>
        </div>
        <div className="bg-white border border-gray-200 rounded p-3">
          <MetaText className="text-gray-500">拒绝</MetaText>
          <div className="text-xl font-semibold text-amber-600 mt-1">{stats.denied}</div>
        </div>
      </div>

      {/* 工具栏 */}
      <div className="flex items-center justify-between gap-4">
        <SegmentGroup>
          {[
            { id: 'all', label: '全部' },
            { id: 'success', label: '成功' },
            { id: 'failed', label: '失败' },
            { id: 'denied', label: '拒绝' },
          ].map(opt => (
            <SegmentOption
              key={opt.id}
              active={filter === opt.id}
              onClick={() => setFilter(opt.id as typeof filter)}
            >
              {opt.label}
            </SegmentOption>
          ))}
        </SegmentGroup>
        <div className="relative w-64">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            placeholder="搜索工具名、用户、模块..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* 日志表格 */}
      <div className="bg-white border border-gray-200 rounded overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead style={{ width: '15%' }}>时间</TableHead>
              <TableHead style={{ width: '8%' }}>用户</TableHead>
              <TableHead style={{ width: '18%' }}>工具</TableHead>
              <TableHead style={{ width: '14%' }}>模块</TableHead>
              <TableHead style={{ width: '25%' }}>参数</TableHead>
              <TableHead style={{ width: '8%' }}>结果</TableHead>
              <TableHead style={{ width: '7%' }}>耗时</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredLogs.map(log => {
              const cfg = RESULT_CONFIG[log.result];
              const Icon = cfg.icon;
              return (
                <TableRow key={log.id}>
                  <TableCell>
                    <MetaText className="text-gray-600">{log.timestamp}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText className="text-gray-900">{log.userName}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText className="font-mono text-gray-900">{log.toolName}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText className="text-gray-600">{log.module}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText className="text-gray-500 font-mono text-xs break-all">{log.paramsSummary || '—'}</MetaText>
                  </TableCell>
                  <TableCell>
                    <StatusTag variant={cfg.variant} mode="fill">
                      {cfg.label}
                    </StatusTag>
                  </TableCell>
                  <TableCell>
                    <MetaText className="text-gray-600">{log.durationMs}ms</MetaText>
                  </TableCell>
                </TableRow>
              );
            })}
            {filteredLogs.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="text-center text-gray-400 py-10">
                  无匹配日志
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <HelperText className="text-gray-400">
        共 {filteredLogs.length} 条记录 · 数据为演示数据，正式接入后将对接 ClawPro 审计日志
      </HelperText>
    </div>
  );
}
