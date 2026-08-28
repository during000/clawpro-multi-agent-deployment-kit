import * as React from "react";

import { SurfaceCard } from "@/components/ui/Surface";
import { Button } from "@/components/ui/button";
import {
  DataTable,
  type DataTableColumn,
  TableActionCell,
} from "@/components/ui/table";
import { StatusTag } from "@/components/ui/status-tag";

/**
 * /preview/data-table — DataTable MVP 预览页
 *
 * 用途：
 *   - 验证 DataTable 7 个状态机（rowKey / selection / pagination / loading / empty / variant / scrollX）
 *   - 后续 pilot 迁移真实列表页前的样板间
 *
 * 这是设计走查页，不接任何后端。所有数据 mock。
 */

interface MockPolicy {
  id: string;
  name: string;
  type: "命令管控" | "IP/DNS 管控";
  status: "已启用" | "已停用";
  hosts: number;
  updatedAt: string;
}

const ALL_DATA: MockPolicy[] = Array.from({ length: 47 }, (_, i) => ({
  id: `pol-${String(i + 1).padStart(3, "0")}`,
  name: `策略 ${i + 1}`,
  type: i % 3 === 0 ? "IP/DNS 管控" : "命令管控",
  status: i % 4 === 0 ? "已停用" : "已启用",
  hosts: ((i * 7) % 200) + 1,
  updatedAt: `2026-06-${String(((i * 3) % 28) + 1).padStart(2, "0")} 10:20`,
}));

export default function DataTablePreview() {
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(10);
  const [selected, setSelected] = React.useState<string[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [emptyMode, setEmptyMode] = React.useState(false);

  const dataSource = React.useMemo(() => {
    if (emptyMode) return [] as MockPolicy[];
    const start = (page - 1) * pageSize;
    return ALL_DATA.slice(start, start + pageSize);
  }, [emptyMode, page, pageSize]);

  const columns: DataTableColumn<MockPolicy>[] = [
    {
      key: "id",
      title: "策略 ID",
      dataIndex: "id",
      width: 140,
    },
    {
      key: "name",
      title: "策略名称",
      dataIndex: "name",
    },
    {
      key: "type",
      title: "类型",
      dataIndex: "type",
      width: 140,
    },
    {
      key: "status",
      title: "状态",
      width: 100,
      render: (_, row) => (
        <StatusTag variant={row.status === "已启用" ? "green" : "gray"}>
          {row.status}
        </StatusTag>
      ),
    },
    {
      key: "hosts",
      title: "关联主机",
      dataIndex: "hosts",
      align: "right",
      width: 120,
    },
    {
      key: "updatedAt",
      title: "更新时间",
      dataIndex: "updatedAt",
      width: 200,
    },
    {
      key: "actions",
      title: "操作",
      width: 160,
      render: (_, row) => (
        <TableActionCell rawChildren>
          <div className="flex items-center gap-6">
            <Button variant="link" onClick={() => alert(`查看 ${row.id}`)}>
              查看
            </Button>
            <Button variant="link" onClick={() => alert(`编辑 ${row.id}`)}>
              编辑
            </Button>
          </div>
        </TableActionCell>
      ),
    },
  ];

  return (
    <div className="min-h-screen bg-[var(--bg-page)] p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-base font-semibold">DataTable MVP Preview</h1>
        <div className="flex items-center gap-2 text-xs">
          <Button
            size="sm"
            variant={loading ? "default" : "outline"}
            onClick={() => {
              setLoading(true);
              setTimeout(() => setLoading(false), 1500);
            }}
          >
            模拟 loading
          </Button>
          <Button
            size="sm"
            variant={emptyMode ? "default" : "outline"}
            onClick={() => setEmptyMode((v) => !v)}
          >
            {emptyMode ? "退出空态" : "切到空态"}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setSelected([])}
            disabled={selected.length === 0}
          >
            清空选中（{selected.length}）
          </Button>
        </div>
      </div>

      <SurfaceCard className="overflow-hidden">
        <DataTable<MockPolicy>
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          rowSelection={{
            selectedKeys: selected,
            onChange: (keys) => setSelected(keys),
            getCheckboxProps: (row) => ({
              disabled: row.status === "已停用",
            }),
          }}
          pagination={{
            current: page,
            pageSize,
            total: emptyMode ? 0 : ALL_DATA.length,
            showTotal: (t, [s, e]) => `${s}-${e} 共 ${t} 条`,
            showSizeChanger: true,
            onChange: (p, s) => {
              setPage(p);
              setPageSize(s);
            },
          }}
          scrollX={1100}
        />
      </SurfaceCard>

      <div className="text-xs text-gray-500">
        当前选中 keys: {selected.join(", ") || "(空)"}
      </div>
    </div>
  );
}
