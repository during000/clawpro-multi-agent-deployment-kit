/**
 * TablePreview
 * --------------------------------------------------------------
 * Table 组件「全场景走查」预览页，专门用来暴露表格相关 bug。
 * 访问路径：/preview/table
 *
 * 覆盖以下易出 bug 的场景：
 *   1. 标准表格（gray-header + default 密度）+ Pagination 搭配
 *   2. 紧凑表格（compact 密度）
 *   3. variant="white"（用于非白背景）
 *   4. 固定列 + 横向滚动（左固定复选框/名称列，右固定操作列）
 *      —— 重点验证：sticky 列在 hover / 选中态是否同步变色、
 *         左右阴影是否随滚动正确出现、表头与 body 边界是否对齐
 *   5. 选中行 data-state="selected"
 */
import { useState } from "react";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { Pagination } from "@/components/ui/pagination";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { StatusTag } from "@/components/ui/status-tag";
import { SurfaceCard } from "@/components/ui/Surface";

type Row = {
  id: string;
  name: string;
  type: string;
  status: "运行中" | "已停止" | "待处理";
  statusVariant: "green" | "red" | "gray";
  owner: string;
  region: string;
  cpu: string;
  mem: string;
  ip: string;
  createdAt: string;
};

const ROWS: Row[] = [
  { id: "ins-hermes01", name: "客服助手 Agent", type: "对话型", status: "运行中", statusVariant: "green", owner: "张三", region: "广州 / ap-guangzhou-3", cpu: "4 核", mem: "8 GB", ip: "10.0.12.31", createdAt: "2026-05-21 10:32" },
  { id: "ins-athena02", name: "数据分析 Agent", type: "工具型", status: "运行中", statusVariant: "green", owner: "李四", region: "上海 / ap-shanghai-2", cpu: "8 核", mem: "16 GB", ip: "10.0.12.45", createdAt: "2026-05-20 09:18" },
  { id: "ins-zeus03", name: "代码生成 Agent", type: "编码型", status: "已停止", statusVariant: "red", owner: "王五", region: "北京 / ap-beijing-1", cpu: "16 核", mem: "32 GB", ip: "10.0.13.02", createdAt: "2026-05-19 16:44" },
  { id: "ins-apollo04", name: "知识库问答 Agent", type: "RAG", status: "待处理", statusVariant: "gray", owner: "赵六", region: "深圳 / ap-shenzhen-1", cpu: "4 核", mem: "8 GB", ip: "10.0.13.77", createdAt: "2026-05-18 14:05" },
  { id: "ins-hera05", name: "营销文案 Agent", type: "生成型", status: "运行中", statusVariant: "green", owner: "孙七", region: "成都 / ap-chengdu-2", cpu: "8 核", mem: "16 GB", ip: "10.0.14.10", createdAt: "2026-05-17 11:50" },
];

function SectionTitle({ children, desc }: { children: React.ReactNode; desc?: string }) {
  return (
    <div className="mb-3">
      <h2 className="text-[15px] font-semibold text-slate-900">{children}</h2>
      {desc && <p className="mt-1 text-xs text-slate-500 leading-relaxed">{desc}</p>}
    </div>
  );
}

/* ① 标准表格 + 分页 ─────────────────────────────── */
function StandardTable() {
  return (
    <SurfaceCard className="overflow-hidden">
      <Table density="default">
        <TableHeader>
          <TableRow>
            <TableHead>实例名称</TableHead>
            <TableHead>类型</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>负责人</TableHead>
            <TableHead className="text-right">创建时间</TableHead>
            <TableHead className="w-[160px]">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {ROWS.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-medium">{r.name}</TableCell>
              <TableCell>{r.type}</TableCell>
              <TableCell>
                <StatusTag mode="dot" variant={r.statusVariant}>{r.status}</StatusTag>
              </TableCell>
              <TableCell>{r.owner}</TableCell>
              <TableCell className="text-right tabular-nums">{r.createdAt}</TableCell>
              <TableActionCell>
                <Button variant="link">查看</Button>
                <Button variant="link">删除</Button>
              </TableActionCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="px-4 py-3 border-t border-gray-200">
        <Pagination total={245} current={1} pageSize={10} showTotal={(t) => `共 ${t} 条记录`} showSizeChanger />
      </div>
    </SurfaceCard>
  );
}

/* ② 紧凑密度 ──────────────────────────────────── */
function CompactTable() {
  return (
    <SurfaceCard className="overflow-hidden">
      <Table density="compact">
        <TableHeader>
          <TableRow>
            <TableHead>实例名称</TableHead>
            <TableHead>类型</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>负责人</TableHead>
            <TableHead className="text-right">创建时间</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {ROWS.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-medium">{r.name}</TableCell>
              <TableCell>{r.type}</TableCell>
              <TableCell>
                <StatusTag mode="dot" variant={r.statusVariant}>{r.status}</StatusTag>
              </TableCell>
              <TableCell>{r.owner}</TableCell>
              <TableCell className="text-right tabular-nums">{r.createdAt}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </SurfaceCard>
  );
}

/* ③ variant="white"（非白背景上浮起） ──────────── */
function WhiteVariantTable() {
  return (
    <div className="rounded-xl bg-[linear-gradient(120deg,#1447E6,#355EF1)] p-6">
      <Table variant="white" density="default">
        <TableHeader>
          <TableRow>
            <TableHead>实例名称</TableHead>
            <TableHead>类型</TableHead>
            <TableHead>状态</TableHead>
            <TableHead className="text-right">CPU</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {ROWS.slice(0, 3).map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-medium">{r.name}</TableCell>
              <TableCell>{r.type}</TableCell>
              <TableCell>
                <StatusTag mode="dot" variant={r.statusVariant}>{r.status}</StatusTag>
              </TableCell>
              <TableCell className="text-right tabular-nums">{r.cpu}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

/* ④ 固定列 + 横向滚动（最易出 bug） ────────────── */
function FixedColumnsTable() {
  const [selected, setSelected] = useState<Record<string, boolean>>({
    "ins-athena02": true,
  });
  const allChecked = ROWS.every((r) => selected[r.id]);
  const toggleAll = () =>
    setSelected(allChecked ? {} : Object.fromEntries(ROWS.map((r) => [r.id, true])));
  const toggle = (id: string) =>
    setSelected((p) => ({ ...p, [id]: !p[id] }));

  return (
    <SurfaceCard className="overflow-hidden">
      <Table scrollX={1500} density="default">
        <TableHeader>
          <TableRow>
            <TableHead fixed="left" fixedShadow={false} className="w-[48px]">
              <Checkbox checked={allChecked} onCheckedChange={toggleAll} />
            </TableHead>
            <TableHead fixed="left" className="w-[200px]">实例名称</TableHead>
            <TableHead className="w-[160px]">实例 ID</TableHead>
            <TableHead>类型</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>负责人</TableHead>
            <TableHead className="w-[220px]">地域 / 可用区</TableHead>
            <TableHead>CPU</TableHead>
            <TableHead>内存</TableHead>
            <TableHead className="w-[160px]">内网 IP</TableHead>
            <TableHead className="w-[180px]">创建时间</TableHead>
            <TableHead fixed="right" className="w-[160px]">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {ROWS.map((r) => (
            <TableRow key={r.id} data-state={selected[r.id] ? "selected" : undefined}>
              <TableCell fixed="left" fixedShadow={false}>
                <Checkbox checked={!!selected[r.id]} onCheckedChange={() => toggle(r.id)} />
              </TableCell>
              <TableCell fixed="left" className="font-medium">{r.name}</TableCell>
              <TableCell>{r.id}</TableCell>
              <TableCell>{r.type}</TableCell>
              <TableCell>
                <StatusTag mode="dot" variant={r.statusVariant}>{r.status}</StatusTag>
              </TableCell>
              <TableCell>{r.owner}</TableCell>
              <TableCell>{r.region}</TableCell>
              <TableCell className="tabular-nums">{r.cpu}</TableCell>
              <TableCell className="tabular-nums">{r.mem}</TableCell>
              <TableCell className="tabular-nums">{r.ip}</TableCell>
              <TableCell className="tabular-nums">{r.createdAt}</TableCell>
              <TableActionCell fixed="right">
                <Button variant="link">编辑</Button>
                <Button variant="link">删除</Button>
              </TableActionCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="px-4 py-3 border-t border-gray-200">
        <Pagination total={245} current={1} pageSize={10} showTotal={(t) => `共 ${t} 条记录`} showSizeChanger />
      </div>
    </SurfaceCard>
  );
}

export default function TablePreview() {
  return (
    <div className="min-h-screen bg-[var(--bg-grey-normal,#f5f6f8)]">
      <header className="sticky top-0 z-20 bg-white/85 backdrop-blur border-b border-slate-200">
        <div className="max-w-[1200px] mx-auto px-6 py-4">
          <h1 className="text-lg font-semibold text-slate-900">Table 组件 · 全场景走查</h1>
          <p className="text-xs text-slate-500 mt-1">
            源：<code className="font-mono">client/src/components/ui/table.tsx</code>　
            重点关注「固定列 + 横向滚动」的 hover / 选中同步与左右阴影。
          </p>
        </div>
      </header>

      <main className="max-w-[1200px] mx-auto px-6 py-8 space-y-12">
        <section>
          <SectionTitle desc="gray-header + default 密度，SurfaceCard 容器内 + 底部 Pagination（默认尺寸）。">
            ① 标准表格 + 分页
          </SectionTitle>
          <StandardTable />
        </section>

        <section>
          <SectionTitle desc="compact 密度：行高 40px，仅密度变化，字号/边框/分割线保持一致。">
            ② 紧凑密度
          </SectionTitle>
          <CompactTable />
        </section>

        <section>
          <SectionTitle desc="variant=&quot;white&quot;：整体白色 + 圆角浮起，用于蓝色渐变 / 灰底等非白背景。">
            ③ variant=&quot;white&quot;
          </SectionTitle>
          <WhiteVariantTable />
        </section>

        <section>
          <SectionTitle desc="scrollX={1500} 触发横滚：左固定复选框+名称列、右固定操作列。请左右滚动并 hover / 勾选行，检查 sticky 列是否跟随变色、左右阴影是否正确出现、表头与 body 边界是否对齐。">
            ④ 固定列 + 横向滚动（重点排查）
          </SectionTitle>
          <FixedColumnsTable />
        </section>
      </main>
    </div>
  );
}
