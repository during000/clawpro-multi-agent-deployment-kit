/**
 * CommandTaskTab - 「命令下发 → 命令列表」Tab 内容
 *
 * 内容：命令模板沉淀（参考 TAT 命令管理）
 *   - 列表：命令 ID/名称（ID 可复制）/ 类型 / 内容预览 / 创建人 / 最近执行 / 总执行次数 / 操作（下发/编辑/删除）
 *   - 顶部："创建命令"按钮 + 搜索
 *
 * 注：执行记录已拆为「命令下发 → 执行记录」独立 Tab，本组件不再展示。
 */
import { useState, useMemo } from "react";
import { toast } from "sonner";
import {
  Plus, Search,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { SurfaceCard } from "@/components/ui/Surface";
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription, EmptyMedia } from "@/components/ui/empty";
import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from "@/components/ui/dialog";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell,
} from "@/components/ui/table";
import { StatusTag } from "@/components/ui/status-tag";
import {
  MOCK_COMMAND_TEMPLATES,
  type CommandTemplate,
} from "./mockData";
import CreateCommandDialog from "./components/CreateCommandDialog";
import DispatchCommandDialog from "./components/DispatchCommandDialog";
import CopyableId from "./components/CopyableId";

export default function CommandTaskTab() {
  const [search, setSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<CommandTemplate | undefined>(undefined);
  const [dispatchTarget, setDispatchTarget] = useState<CommandTemplate | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CommandTemplate | null>(null);
  const [tick, setTick] = useState(0); // 强制刷新 mock 列表

  const templates = useMemo(() => {
    const q = search.trim().toLowerCase();
    return MOCK_COMMAND_TEMPLATES.filter((t) => {
      if (!q) return true;
      return (
        t.id.toLowerCase().includes(q) ||
        t.name.toLowerCase().includes(q) ||
        t.content.toLowerCase().includes(q) ||
        (t.description ?? "").toLowerCase().includes(q) ||
        t.createdBy.toLowerCase().includes(q)
      );
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, tick]);

  return (
    <div className="space-y-4">
      {/* ─── 标题栏：标题 + 共 N 条 + 搜索 + 创建命令 ─────────────────────── */}
      <div className="flex items-center gap-3">
        <h2 className="font-semibold text-[#0A0A0A] text-base inline-flex items-center gap-2 shrink-0">
          命令库
          <span className="text-xs font-normal text-[#737373] tabular-nums">
            共 {templates.length} 条
          </span>
        </h2>
        <div className="ml-auto flex items-center gap-2">
          <div className="relative w-[260px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="搜索命令 ID、名称、内容、创建人"
              className="h-9 pl-9 bg-white"
            />
          </div>
          <Button
            variant="claw-primary"
            size="claw-sm"
            onClick={() => {
              setEditTarget(undefined);
              setCreateOpen(true);
            }}
          >
            <Plus className="w-4 h-4 mr-1" />
            创建命令
          </Button>
        </div>
      </div>

      {/* ─── 表格（与执行记录同款 SurfaceCard + elevated-white） ──────── */}
      {templates.length === 0 ? (
        <SurfaceCard className="overflow-hidden">
          <Empty className="border-0">
            <EmptyMedia />
            <EmptyHeader>
              {search ? (
                <EmptyDescription>没有匹配的命令</EmptyDescription>
              ) : (
                <>
                  <EmptyTitle>暂无命令</EmptyTitle>
                  <EmptyDescription>点击「创建命令」开始沉淀团队 SOP</EmptyDescription>
                </>
              )}
            </EmptyHeader>
          </Empty>
        </SurfaceCard>
      ) : (
        <SurfaceCard className="overflow-hidden">
        <Table variant="white">
          <TableHeader>
            <TableRow>
              <TableHead className="w-[22%]">命令 ID / 名称</TableHead>
              <TableHead className="w-[8%]">类型</TableHead>
              <TableHead>命令内容</TableHead>
              <TableHead className="w-[14%]">创建人</TableHead>
              <TableHead className="w-[14%]">最近执行</TableHead>
              <TableHead className="w-[14%]">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {templates.map((t) => (
              <TableRow key={t.id}>
                <TableCell>
                  <CopyableId id={t.id} />
                  <div className="font-medium text-[#0A0A0A] mt-0.5">{t.name}</div>
                  {t.description && (
                    <div className="text-[#A3A3A3] mt-0.5 truncate max-w-[260px]">
                      {t.description}
                    </div>
                  )}
                </TableCell>
                <TableCell>
                  <StatusTag mode="fill" variant="blue">{t.type}</StatusTag>
                </TableCell>
                <TableCell>
                  <TooltipProvider delayDuration={150}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <code className="font-mono text-[#737373] bg-gray-50 px-2 py-1 rounded block truncate max-w-[420px]">
                          {t.content.split("\n")[0]}
                          {t.content.includes("\n") && (
                            <span className="text-[#A3A3A3] ml-1">…</span>
                          )}
                        </code>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="max-w-[480px]">
                        <pre className="text-[11px] font-mono whitespace-pre-wrap break-all">
                          {t.content}
                        </pre>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </TableCell>
                <TableCell>
                  <div className="text-[#0A0A0A] truncate max-w-[120px]">{t.createdBy}</div>
                  <div className="text-[#A3A3A3] tabular-nums">{t.createdAt.slice(0, 10)}</div>
                </TableCell>
                <TableCell>
                  {t.lastRunAt ? (
                    <div className="text-[#0A0A0A] tabular-nums">{t.lastRunAt.slice(5, 16)}</div>
                  ) : (
                    <span className="text-[#A3A3A3]">从未执行</span>
                  )}
                  <div className="text-[#A3A3A3] tabular-nums">
                    共 {t.totalRuns} 次
                  </div>
                </TableCell>
                <TableActionCell actionsClassName="gap-3">
                  <Button variant="link" onClick={() => setDispatchTarget(t)}>
                    下发
                  </Button>
                  <Button
                    variant="link"
                    onClick={() => {
                      setEditTarget(t);
                      setCreateOpen(true);
                    }}
                  >
                    编辑
                  </Button>
                  <Button variant="link" onClick={() => setDeleteTarget(t)}>
                    删除
                  </Button>
                </TableActionCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        </SurfaceCard>
      )}

      {/* 子弹窗 */}
      <CreateCommandDialog
        open={createOpen}
        onOpenChange={(v) => {
          setCreateOpen(v);
          if (!v) setEditTarget(undefined);
        }}
        template={editTarget}
        onSaved={() => setTick((t) => t + 1)}
      />

      <DispatchCommandDialog
        open={!!dispatchTarget}
        onOpenChange={(v) => !v && setDispatchTarget(null)}
        command={dispatchTarget}
        onDispatched={() => setTick((t) => t + 1)}
      />

      {/* 删除确认 */}
      <Dialog open={!!deleteTarget} onOpenChange={(v) => !v && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle className="text-lg leading-none font-semibold">
              删除命令？
            </DialogTitle>
            <DialogDescription>
              即将删除「<span className="font-medium text-[#334155]">{deleteTarget?.name}</span>」，已存在的执行记录不会被删除。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (deleteTarget) {
                  const idx = MOCK_COMMAND_TEMPLATES.findIndex((x) => x.id === deleteTarget.id);
                  if (idx >= 0) MOCK_COMMAND_TEMPLATES.splice(idx, 1);
                  toast.success("命令已删除");
                  setDeleteTarget(null);
                  setTick((t) => t + 1);
                }
              }}
            >
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
