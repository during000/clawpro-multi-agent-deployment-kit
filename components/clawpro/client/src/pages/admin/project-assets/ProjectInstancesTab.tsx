/**
 * ProjectInstancesTab - 「项目资产管理」组织 / 项目实例（只读）
 * 复用管控端 Agent 列表页数据（MOCK_CLAWS）与展示口径：
 * - 实例名称 / ID（ins-xxx）、当前状态（ClawStatus，彩色文本）、用户ID（email）、
 *   归属（组织 / 项目全路径）、Agent 类型。
 * - 组织：聚合本组织 + 子组织的实例，「组织」列显示该实例归属组织的全链路名称；
 * - 项目：项目单层，「项目」列显示该实例归属的一个或多个项目（多个时「首项 +N」，
 *   hover 看全部，样式与「用户管理-全部视图」项目列一致）。
 * - 长文本列统一「max-width + truncate + hover Tooltip」；实例较多时底部分页。
 * 仅展示不可编辑。
 */
import { useEffect, useMemo, useState } from 'react';
import { StatusTag } from '@/components/ui/status-tag';
import { Empty, EmptyHeader, EmptyDescription } from '@/components/ui/empty';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import { Pagination } from '@/components/ui/pagination';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { UserGroup } from '../MemberManagement/types';
import { AGENT_TYPE_DISPLAY, STATUS_CONFIG, type Claw } from '../OpenClawMonitor';
import { getGroupPath, getOrgInstanceRows, getProjectInstanceRows } from './projectRelations';

const PAGE_SIZE = 10;

function agentTypeLabel(inst: Claw): string {
  if (inst.agentType === 'LocalAgent') {
    return inst.localProduct ? `本地 Agent · ${inst.localProduct}` : '本地 Agent';
  }
  return AGENT_TYPE_DISPLAY[inst.agentType] ?? inst.agentType;
}

interface ProjectInstancesTabProps {
  groupId: string;
  groups: UserGroup[];
  isProject: boolean;
}

export default function ProjectInstancesTab({ groupId, groups, isProject }: ProjectInstancesTabProps) {
  // 组织：聚合本组织 + 子组织；项目：项目单层。统一成带归属信息的行数据。
  const rows = useMemo(
    () =>
      isProject
        ? getProjectInstanceRows(groupId, groups).map((r) => ({
            inst: r as Claw,
            ownerPaths: r.projectIds.map((pid) => getGroupPath(pid, groups)),
          }))
        : getOrgInstanceRows(groupId, groups).map((r) => ({
            inst: r as Claw,
            ownerPaths: [getGroupPath(r.orgId, groups)],
          })),
    [groupId, groups, isProject],
  );

  const ownerLabel = isProject ? '项目' : '组织';

  // 切换节点 / 视图时回到第 1 页
  const [page, setPage] = useState(1);
  useEffect(() => {
    setPage(1);
  }, [groupId, isProject]);

  const pagedRows = useMemo(
    () => rows.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE),
    [rows, page],
  );

  if (rows.length === 0) {
    return (
      <Empty className="py-16">
        <EmptyHeader>
          <EmptyDescription>{isProject ? '该项目暂无 Agent 实例' : '该组织暂无 Agent 实例'}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div>
      <div className="rounded-[4px] border border-[var(--cp-border)] overflow-hidden bg-[var(--cp-surface)]">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead style={{ width: '240px' }}>实例名称 / ID</TableHead>
              <TableHead style={{ width: '110px' }}>当前状态</TableHead>
              <TableHead style={{ width: '200px' }}>用户 ID</TableHead>
              <TableHead style={{ minWidth: '160px' }}>{ownerLabel}</TableHead>
              <TableHead style={{ width: '140px' }}>Agent 类型</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pagedRows.map(({ inst, ownerPaths }) => {
              const meta = STATUS_CONFIG[inst.status];
              return (
                <TableRow key={inst.id}>
                  {/* 实例名称 / ID：定宽 + 双行截断，hover 看全称与完整 ID */}
                  <TableCell style={{ width: '240px' }}>
                    <Tooltip delayDuration={200}>
                      <TooltipTrigger asChild>
                        <div className="flex flex-col max-w-[220px] cursor-default">
                          <BodyMedium className="truncate">{inst.name}</BodyMedium>
                          <MetaText tone="secondary" className="truncate">{inst.instanceId}</MetaText>
                        </div>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-[360px]">
                        <div className="space-y-0.5">
                          <div className="break-all">{inst.name}</div>
                          <div className="break-all text-[var(--text-muted)]">{inst.instanceId}</div>
                        </div>
                      </TooltipContent>
                    </Tooltip>
                  </TableCell>
                  {/* 当前状态：复用 Agent 列表页 STATUS_CONFIG 的彩色文本 */}
                  <TableCell style={{ width: '110px' }}>
                    <StatusTag mode="text" variant={meta.tagVariant}>{meta.label}</StatusTag>
                  </TableCell>
                  {/* 用户 ID：定宽 + 截断，hover 看全称 */}
                  <TableCell style={{ width: '200px' }}>
                    <Tooltip delayDuration={200}>
                      <TooltipTrigger asChild>
                        <MetaText tone="secondary" className="block max-w-[180px] truncate cursor-default">
                          {inst.creator}
                        </MetaText>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-[360px] break-all">
                        {inst.creator}
                      </TooltipContent>
                    </Tooltip>
                  </TableCell>
                  {/* 组织 / 项目：首项 +N，hover 编号列表看全部（与用户管理-全部视图项目列一致） */}
                  <TableCell style={{ minWidth: '160px' }}>
                    {ownerPaths.length === 0 ? (
                      <span className="text-[var(--text-weak)]">—</span>
                    ) : (
                      <Tooltip delayDuration={200}>
                        <TooltipTrigger asChild>
                          <span className="inline-flex items-center gap-1 max-w-[220px] cursor-default">
                            <span className="truncate text-sm text-[var(--text-secondary)]">
                              {ownerPaths[0]}
                            </span>
                            {ownerPaths.length > 1 && (
                              <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                +{ownerPaths.length - 1}
                              </span>
                            )}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs max-w-[360px]">
                          <div className="space-y-1">
                            {ownerPaths.map((p, idx) => (
                              <div key={idx} className="break-all">
                                {ownerPaths.length > 1 && <span className="tabular-nums mr-1">{idx + 1}.</span>}
                                {p}
                              </div>
                            ))}
                          </div>
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableCell style={{ width: '140px' }}>
                    <MetaText tone="secondary">{agentTypeLabel(inst)}</MetaText>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
        {/* 底部翻页：翻页器自带「共 x 条」+ 页码，整体右对齐 */}
        <div className="flex justify-end px-4 py-2 border-t border-[var(--cp-border)]">
          <Pagination
            total={rows.length}
            current={page}
            pageSize={PAGE_SIZE}
            size="default"
            className="justify-end flex-nowrap"
            hideOnSinglePage
            onChange={(p) => { setPage(p); }}
          />
        </div>
      </div>
    </div>
  );
}
