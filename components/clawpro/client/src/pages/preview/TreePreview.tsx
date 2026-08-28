/**
 * Tree Preview
 * 路由：/preview/tree
 * 展示树结构组件的展开/收起、选中、hover、禁用状态
 */
import { useState } from "react";
import { ChevronRight, ChevronDown, Folder, FolderOpen, FileText } from "lucide-react";
import { SectionTitle, MetaText } from "@/components/ui/Typography";

function DemoBlock({ title, desc, children }: { title: string; desc?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <SectionTitle as="h3" className="!text-sm">{title}</SectionTitle>
        {desc && <MetaText tone="weak">{desc}</MetaText>}
      </header>
      <div className="p-5">{children}</div>
    </section>
  );
}

type TreeNode = {
  id: string;
  label: string;
  count?: number;
  disabled?: boolean;
  children?: TreeNode[];
};

const MOCK_TREE: TreeNode[] = [
  {
    id: "1",
    label: "全部用户",
    count: 128,
    children: [
      { id: "1-1", label: "管理员组", count: 5 },
      { id: "1-2", label: "普通用户组", count: 98 },
      {
        id: "1-3",
        label: "开发团队",
        count: 25,
        children: [
          { id: "1-3-1", label: "前端组", count: 12 },
          { id: "1-3-2", label: "后端组", count: 8 },
          { id: "1-3-3", label: "AI 组", count: 5 },
        ],
      },
    ],
  },
  {
    id: "2",
    label: "已归档",
    count: 15,
    disabled: true,
    children: [
      { id: "2-1", label: "2025 年归档", count: 10, disabled: true },
      { id: "2-2", label: "2024 年归档", count: 5, disabled: true },
    ],
  },
];

function TreeItem({
  node,
  depth,
  activeId,
  onSelect,
}: {
  node: TreeNode;
  depth: number;
  activeId: string;
  onSelect: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const hasChildren = !!node.children?.length;
  const isActive = activeId === node.id;

  return (
    <>
      <div
        className={[
          "group flex items-center gap-1.5 h-8 pr-3 text-sm cursor-pointer rounded-[4px] transition-colors",
          node.disabled
            ? "text-[#a1a1aa] cursor-not-allowed opacity-60"
            : isActive
              ? "bg-[#f4f4f5] text-[#09090b] font-medium"
              : "text-[#09090b] hover:bg-[#f4f4f5]",
        ].join(" ")}
        style={{ paddingLeft: 8 + depth * 16 }}
        onClick={() => {
          if (node.disabled) return;
          onSelect(node.id);
          if (hasChildren) setExpanded(!expanded);
        }}
      >
        {hasChildren ? (
          <span className="w-4 h-4 flex items-center justify-center text-[#71717a] shrink-0">
            {expanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
          </span>
        ) : (
          <span className="w-4 h-4 shrink-0" />
        )}
        <span className="w-4 h-4 text-[#71717a] shrink-0">
          {hasChildren ? (expanded ? <FolderOpen className="w-3.5 h-3.5" /> : <Folder className="w-3.5 h-3.5" />) : <FileText className="w-3.5 h-3.5" />}
        </span>
        <span className="truncate">{node.label}</span>
        {node.count != null && (
          <span className="text-[11px] tabular-nums shrink-0 text-[#a1a1aa]">({node.count})</span>
        )}
      </div>
      {expanded && hasChildren && (
        <div>
          {node.children!.map((child) => (
            <TreeItem key={child.id} node={child} depth={depth + 1} activeId={activeId} onSelect={onSelect} />
          ))}
        </div>
      )}
    </>
  );
}

export default function TreePreview() {
  const [activeId, setActiveId] = useState("1-1");

  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8">
      <div className="max-w-3xl mx-auto space-y-8">
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">Tree 树结构</h1>
          <p className="text-sm text-[#64748B]">
            行高 32px · 图标色 #71717a · 缩进 16px/层 · 选中态 #f4f4f5
          </p>
        </header>

        <DemoBlock title="标准树" desc="展开/收起 + 选中 + 计数 + 多层嵌套">
          <div className="max-w-[320px] rounded-[4px] border border-[#e5e5e5] bg-white p-2">
            <div className="flex flex-col gap-0.5">
              {MOCK_TREE.map((node) => (
                <TreeItem key={node.id} node={node} depth={0} activeId={activeId} onSelect={setActiveId} />
              ))}
            </div>
          </div>
        </DemoBlock>

        <DemoBlock title="状态说明">
          <div className="grid grid-cols-2 gap-4 text-xs text-[#64748B]">
            <div className="space-y-1">
              <p className="font-medium text-[#0A0A0A]">交互状态</p>
              <p>• 默认：text-[#09090b] bg-transparent</p>
              <p>• Hover：bg-[#f4f4f5]</p>
              <p>• 选中：bg-[#f4f4f5] font-medium</p>
              <p>• 禁用：text-[#a1a1aa] opacity-60</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-[#0A0A0A]">视觉参数</p>
              <p>• 行高：32px (h-8)</p>
              <p>• 圆角：4px</p>
              <p>• 图标色：#71717a（统一）</p>
              <p>• 缩进：8 + depth × 16 px</p>
              <p>• 箭头/图标：3.5~4 (14~16px)</p>
            </div>
          </div>
        </DemoBlock>
      </div>
    </div>
  );
}
