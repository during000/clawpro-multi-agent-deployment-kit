/**
 * Empty 空状态组件总览
 *
 * 路由：/preview/empty-state
 *
 * 内容按「管控端 / 用户端」两端拆分：
 *   - 管控端：SurfaceCard 容器、Button 默认变体、py-12
 *   - 用户端：TenantCard 容器、tenant-primary/tenant-outline 变体、py-16、12px 圆角
 *
 * 来源规范：
 *   - SKILL-GLOBAL-COMPONENTS.md §24 / §24.1
 *   - SKILL.md §10.1（管控端三类空态分层）/ §8.7（弹窗空态）
 *   - SKILL-TENANT.md §3 / §5（用户端按钮 / 卡片差异）
 */
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
  EmptyMedia,
} from "@/components/ui/empty";
import { Button } from "@/components/ui/button";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { TenantCard } from "@/components/ui/Surface";
import {
  HelperText,
  SectionTitle,
  MetaText,
  MetaMedium,
} from "@/components/ui/Typography";
import { Plus, ChevronDown, Search, Bell } from "lucide-react";

/* -------------------------------------------------------------------- */
/* 局部小组件                                                            */
/* -------------------------------------------------------------------- */
function ShowcaseCard({
  title,
  desc,
  variantTag,
  children,
}: {
  title: string;
  desc?: string;
  variantTag?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="bg-white rounded-[6px] border border-[#e5e5e5] overflow-hidden">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <div className="flex items-baseline gap-2">
          <SectionTitle as="h3" className="text-sm font-medium">
            {title}
          </SectionTitle>
          {desc && (
            <MetaText as="span" tone="weak">
              {desc}
            </MetaText>
          )}
        </div>
        {variantTag && (
          <code className="text-[11px] font-mono text-[#737373] bg-white border border-[#e5e5e5] px-1.5 py-0.5 rounded">
            {variantTag}
          </code>
        )}
      </header>
      <div className="p-6 bg-white">{children}</div>
    </section>
  );
}

/* 默认插画统一调用 */
function EmptyIllustration() {
  return <EmptyMedia />;
}

/* 提示类插画：功能关闭 / 未开通 / 需管理员处理 */
function EmptyHintIllustration() {
  return <EmptyMedia variant="hint" />;
}

/* 模拟 Drawer 容器（不引入真实 Drawer，避免触发开关副作用） */
function MockDrawer({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-white border border-[#e5e5e5] rounded-[8px] overflow-hidden shadow-sm">
      <div className="flex items-center justify-between px-6 py-4 border-b border-[#f0f0f0]">
        <SectionTitle as="div" className="text-sm font-semibold">
          {title}
        </SectionTitle>
        <code className="text-[11px] font-mono text-[#a3a3a3]">Drawer</code>
      </div>
      <div className="bg-[#fafafa]">{children}</div>
    </div>
  );
}

/* 模拟下拉面板 */
function MockPanel({ title, width = 240, children }: { title: string; width?: number; children: React.ReactNode }) {
  return (
    <div className="flex flex-col items-start gap-2">
      <button
        type="button"
        className="inline-flex items-center gap-1.5 h-9 px-3 rounded-[4px] border border-[#d3d6db] bg-white text-sm text-[#020617]"
      >
        {title}
        <ChevronDown className="size-4 text-[#7b818f]" />
      </button>
      <div
        className="bg-white rounded-[4px] p-2"
        style={{
          width,
          boxShadow: "0px 0px 2px rgba(0,0,0,0.1), 0px 4px 16px rgba(0,0,0,0.12)",
        }}
      >
        {children}
      </div>
    </div>
  );
}

/* -------------------------------------------------------------------- */
/* 管控端规范内容                                                         */
/* -------------------------------------------------------------------- */
function AdminPanel() {
  return (
    <div>
      {/* 容器场景速查 */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">
        📌 容器场景速查（管控端）
      </h2>
      <div className="bg-white rounded-[6px] border border-[#e5e5e5] overflow-hidden mb-12">
        <table className="w-full text-sm">
          <thead className="bg-[#fafafa] border-b border-[#e5e5e5]">
            <tr className="text-left">
              <th className="px-5 py-3 font-medium text-[#0A0A0A] w-[18%]">容器类型</th>
              <th className="px-5 py-3 font-medium text-[#0A0A0A] w-[42%]">空态写法</th>
              <th className="px-5 py-3 font-medium text-[#0A0A0A]">关键约束</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#f0f0f0] text-[#171717]">
            {[
              { c: "页面 / 大区域", w: "Empty + 兔子插画 + 标题（+ 描述 + 操作）", rule: "border-0 + py-12" },
              { c: "卡片 (SurfaceCard)", w: "同上 (Empty 嵌入 SurfaceCard)", rule: "Empty 加 border-0" },
              { c: "表格（页面级 / 弹窗内 / 紧凑）", w: "td 内文字 — 双行：标题+描述 / 单行：text-[var(--text-weak)]", rule: "❌ 表格空态一律不用兔子插画" },
              { c: "Drawer 主内容区", w: "Empty + 兔子插画（同页面级）", rule: "py-16，第一层内容" },
              { c: "Drawer 嵌套子模块", w: "HelperText 单行/双行，无图标", rule: "层级 ≥ 2 降级" },
              { c: "Dialog / 弹窗内嵌区块", w: "HelperText × 2 + space-y-1", rule: "禁用插画" },
              { c: "Dropdown / Select 下拉", w: "HelperText 单行", rule: "面板内 px-3 py-6 居中" },
              { c: "Combobox / 搜索下拉", w: "HelperText + 可选 brand 链接", rule: "如「+ 新建」入口" },
              { c: "Popover / Tooltip", w: "HelperText 单行", rule: "12px #A3A3A3" },
              { c: "侧栏 / 树筛选无结果", w: "Empty + 兔子插画 + EmptyDescription", rule: "border-0 + 紧凑 padding" },
              { c: "字段值 / 行内 '暂无'", w: 'MetaText tone="weak"', rule: "不另起行不加图" },
            ].map((row) => (
              <tr key={row.c}>
                <td className="px-5 py-3 align-top">{row.c}</td>
                <td className="px-5 py-3 align-top">
                  <code className="font-mono text-xs bg-[#fafafa] px-1.5 py-0.5 rounded border border-[#f0f0f0] text-[#0A0A0A]">
                    {row.w}
                  </code>
                </td>
                <td className="px-5 py-3 align-top text-[#737373]">{row.rule}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 1. 通用空态 */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">
        1. 通用空态（页面 / 卡片 / 列表）
      </h2>
      <p className="text-sm text-[#737373] mb-4">
        统一兔子插画。<strong>双行</strong>：标题 14px 黑 + 描述 12px 灰；<strong>单行</strong>：直接 12px 灰描述。
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="基础（双行）" desc="标题 + 描述">
          <Empty className="border-0 py-12">
            <EmptyHeader>
              <EmptyIllustration />
              <EmptyTitle>暂无数据</EmptyTitle>
              <EmptyDescription>当前没有可显示的内容</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </ShowcaseCard>

        <ShowcaseCard title="带操作引导（双行 + 按钮）" desc="EmptyContent">
          <Empty className="border-0 py-12">
            <EmptyHeader>
              <EmptyIllustration />
              <EmptyTitle>还没有创建任何 Agent</EmptyTitle>
              <EmptyDescription>创建你的第一个 Agent，开始自动化工作流</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button>
                <Plus className="w-4 h-4" />
                新建 Agent
              </Button>
            </EmptyContent>
          </Empty>
        </ShowcaseCard>

        <ShowcaseCard title="搜索无结果（双行）" desc="筛选条件命中为空">
          <Empty className="border-0 py-12">
            <EmptyHeader>
              <EmptyIllustration />
              <EmptyTitle>没有匹配的结果</EmptyTitle>
              <EmptyDescription>尝试更换关键词或调整筛选条件</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </ShowcaseCard>

        <ShowcaseCard title="提示类（未开通 / 已关闭）" desc="功能关闭、需管理员处理" variantTag='EmptyMedia variant="hint"'>
          <Empty className="border-0 py-12">
            <EmptyHeader>
              <EmptyHintIllustration />
              <EmptyTitle>为 Agent 开启服务能力</EmptyTitle>
              <EmptyDescription>当前实例暂未开通，请联系管理员在管控端开启</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </ShowcaseCard>

        <ShowcaseCard title="单行：暂无记录" desc="文案极短只用描述">
          <Empty className="border-0 py-12">
            <EmptyHeader>
              <EmptyIllustration />
              <EmptyDescription>暂无记录</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </ShowcaseCard>

        <ShowcaseCard title="侧栏轻量空态" desc="窄区域 / 极简">
          <Empty className="border-0 px-4 py-10">
            <EmptyHeader>
              <EmptyIllustration />
              <EmptyDescription>暂无符合筛选条件的组织</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </ShowcaseCard>

        <ShowcaseCard title="表格空态（嵌入 td）" desc="页面级表格 — 不加插画 / 双行纯文字" variantTag="HelperText × 2">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell colSpan={2}>
                  <div className="text-center py-12 space-y-1">
                    <HelperText>暂无记录</HelperText>
                    <HelperText>尝试调整筛选条件，或新建一条记录</HelperText>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </ShowcaseCard>
      </div>

      {/* 2. Drawer */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">2. Drawer 抽屉空态</h2>
      <p className="text-sm text-[#737373] mb-4">
        Drawer 主内容区视为页面级（兔子插画）；嵌套子模块（如 Tab 下子列表）降级为 HelperText 文字。
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="Drawer 主内容区" desc="第一层内容为列表" variantTag="兔子插画">
          <MockDrawer title="模板列表">
            <Empty className="border-0 py-16">
              <EmptyHeader>
                <EmptyIllustration />
                <EmptyTitle>暂无模板</EmptyTitle>
                <EmptyDescription>从右上角「新建模板」开始创建</EmptyDescription>
              </EmptyHeader>
            </Empty>
          </MockDrawer>
        </ShowcaseCard>

        <ShowcaseCard title="Drawer 嵌套子模块" desc="Tab / 卡片内空态" variantTag="HelperText × 2">
          <MockDrawer title="资产详情">
            <div className="px-6 py-4 border-b border-[#f0f0f0]">
              <div className="flex gap-6 text-sm">
                <span className="text-[#0A0A0A] font-medium border-b-2 border-[#1447E6] pb-1.5">关联告警</span>
                <span className="text-[#737373]">行为记录</span>
                <span className="text-[#737373]">配置</span>
              </div>
            </div>
            <div className="px-6 py-12">
              <div className="text-center space-y-1">
                <HelperText>暂无关联告警</HelperText>
                <HelperText>该资产当前未触发任何告警规则</HelperText>
              </div>
            </div>
          </MockDrawer>
        </ShowcaseCard>
      </div>

      {/* 3. 浮层 */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">3. 下拉 / Popover / 弹窗内嵌空态</h2>
      <p className="text-sm text-[#737373] mb-4">
        浮层和弹窗内的空态：禁用插画，统一 <code className="font-mono text-xs">HelperText</code> 纯文字（12px <code className="font-mono text-xs">#A3A3A3</code>）。
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="Dropdown / Select 下拉" desc="面板内无可选项" variantTag="HelperText 单行">
          <MockPanel title="选择行业">
            <div className="text-center py-6">
              <HelperText>暂无可选项</HelperText>
            </div>
          </MockPanel>
        </ShowcaseCard>

        <ShowcaseCard title="Combobox 搜索下拉" desc="搜索无匹配 + 新建入口" variantTag="HelperText + 链接">
          <MockPanel title="搜索成员">
            <div className="relative mb-2">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[#a3a3a3]" />
              <input
                className="w-full h-8 pl-8 pr-3 rounded-[4px] border border-[#d3d6db] text-sm focus:outline-none focus:border-[#1447E6]"
                defaultValue="zhangsan"
              />
            </div>
            <div className="text-center py-4 space-y-2">
              <HelperText>没有匹配的结果</HelperText>
              <button type="button" className="text-xs text-[var(--text-brand)] hover:underline underline-offset-2">
                + 邀请「zhangsan」加入
              </button>
            </div>
          </MockPanel>
        </ShowcaseCard>

        <ShowcaseCard title="Popover 内空态" desc="如通知中心" variantTag="HelperText 单行">
          <div className="flex flex-col items-start gap-2">
            <button
              type="button"
              className="inline-flex items-center justify-center size-9 rounded-[6px] border border-[#e5e5e5] bg-white text-[#737373] hover:bg-[#f5f5f5]"
            >
              <Bell className="w-4 h-4" />
            </button>
            <div
              className="bg-white rounded-[4px] p-3"
              style={{ width: 280, boxShadow: "0px 0px 2px rgba(0,0,0,0.1), 0px 4px 16px rgba(0,0,0,0.12)" }}
            >
              <div className="border-b border-[#f0f0f0] pb-2 mb-2">
                <MetaMedium tone="secondary">通知</MetaMedium>
              </div>
              <div className="text-center py-6">
                <HelperText>暂无未读通知</HelperText>
              </div>
            </div>
          </div>
        </ShowcaseCard>

        <ShowcaseCard title="Dialog 内嵌区块空态" desc="弹窗里某个 section 为空" variantTag="HelperText × 2">
          <div className="bg-white border border-[#e5e5e5] rounded-[6px] p-5">
            <MetaMedium as="div" tone="secondary" className="mb-3">
              技能列表
            </MetaMedium>
            <div className="text-center py-12 space-y-1">
              <HelperText>该角色还没有技能</HelperText>
              <HelperText>可从公共技能库或企业技能库添加</HelperText>
            </div>
          </div>
        </ShowcaseCard>
      </div>

      {/* 4. 行内空态 */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">4. 行内空态</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="字段值为空" variantTag='MetaText tone="weak"'>
          <div className="space-y-3">
            <div className="flex gap-3 text-sm">
              <MetaMedium as="span" tone="secondary" className="w-24">腾讯云标签</MetaMedium>
              <MetaText as="span" tone="weak">暂无腾讯云标签</MetaText>
            </div>
            <div className="flex gap-3 text-sm">
              <MetaMedium as="span" tone="secondary" className="w-24">OpenClaw 标签</MetaMedium>
              <MetaText as="span" tone="weak">暂无 OpenClaw 标签</MetaText>
            </div>
          </div>
        </ShowcaseCard>

        <ShowcaseCard title="详情值缺省" desc="兜底 '—' 或 '暂无'" variantTag='MetaText tone="weak"'>
          <dl className="space-y-2 text-sm">
            <div className="flex gap-3">
              <dt className="w-24 text-[var(--text-secondary)]">建议方案</dt>
              <dd><MetaText as="span" tone="weak">暂无</MetaText></dd>
            </div>
            <div className="flex gap-3">
              <dt className="w-24 text-[var(--text-secondary)]">解决方案</dt>
              <dd><MetaText as="span" tone="weak">暂无</MetaText></dd>
            </div>
          </dl>
        </ShowcaseCard>
      </div>

      {/* 5. 已废弃 */}
      <h2 className="text-lg font-semibold text-[#dc2626] mb-4">5. 已废弃写法对照</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="❌ 大图标 + 硬编码灰字" variantTag="DEPRECATED">
          <div className="text-center py-16 border border-dashed border-red-200 rounded-[4px]">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-12 h-12 text-gray-200 mx-auto mb-4"
              viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"
            >
              <rect width="18" height="18" x="3" y="3" rx="2" />
              <path d="M12 8v8M8 12h8" />
            </svg>
            <p className="text-gray-400 mb-4">暂无数据描述</p>
            <Button variant="outline">操作按钮</Button>
          </div>
        </ShowcaseCard>

        <ShowcaseCard title="❌ 单行用大黑标题" variantTag="DEPRECATED">
          <div className="border border-dashed border-red-200 rounded-[4px] py-12">
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyIllustration />
                <EmptyTitle>暂无记录</EmptyTitle>
              </EmptyHeader>
            </Empty>
          </div>
          <p className="mt-3 text-xs text-[#dc2626]">
            ❌ 单行 14px 黑标题视觉过重，应直接用 12px <code className="font-mono">EmptyDescription</code>
          </p>
        </ShowcaseCard>
      </div>
    </div>
  );
}

/* -------------------------------------------------------------------- */
/* 用户端规范内容                                                         */
/* -------------------------------------------------------------------- */
function TenantPanel() {
  return (
    <div>
      {/* 关键差异表 */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">
        🔄 与管控端的关键差异
      </h2>
      <div className="bg-white rounded-[6px] border border-[#e5e5e5] overflow-hidden mb-12">
        <table className="w-full text-sm">
          <thead className="bg-[#fafafa] border-b border-[#e5e5e5]">
            <tr className="text-left">
              <th className="px-5 py-3 font-medium text-[#0A0A0A] w-1/3">维度</th>
              <th className="px-5 py-3 font-medium text-[#0A0A0A]">管控端</th>
              <th className="px-5 py-3 font-medium text-[#0A0A0A]">用户端</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#f0f0f0] text-[#171717]">
            {[
              ["主操作按钮", "Button (default)", "Button variant=tenant-primary（圆角胶囊）"],
              ["次操作按钮", "Button variant=outline", "Button variant=tenant-outline（圆角胶囊）"],
              ["卡片容器", "SurfaceCard（4px 圆角）", "TenantCard（12px 圆角，单层阴影）"],
              ["空态垂直 padding", "py-12", "py-16（更松，呼吸感更强）"],
              ["文案风格", "正式（如「暂无数据」）", "偏引导（如「这里还是空的」）"],
              ["插画 / Empty 组件", "完全一致", "完全一致"],
              ["浮层空态（弹窗/下拉）", "完全一致", "完全一致（HelperText 不区分两端）"],
            ].map(([dim, admin, tenant]) => (
              <tr key={dim as string}>
                <td className="px-5 py-3 align-top text-[#0A0A0A] font-medium">{dim}</td>
                <td className="px-5 py-3 align-top text-[#737373]">{admin}</td>
                <td className="px-5 py-3 align-top">
                  <span className="text-[#1447E6]">{tenant}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 1. 通用空态（用户端） */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">
        1. 通用空态（用户端）
      </h2>
      <p className="text-sm text-[#737373] mb-4">
        外层用 <code className="font-mono text-xs">TenantCard</code> 替代 <code className="font-mono text-xs">SurfaceCard</code>，按钮**统一使用** <code className="font-mono text-xs">tenant-outline</code>（线框胶囊，弱化引导）；多按钮场景**最多 2 个**，**禁止使用** <code className="font-mono text-xs">tenant-primary</code> 实心按钮。padding 用 <code className="font-mono text-xs">py-16</code>。
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard
          title="基础（双行）"
          desc="TenantCard + py-16"
          variantTag="TenantCard"
        >
          <TenantCard padding="none">
            <Empty className="border-0 py-16">
              <EmptyHeader>
                <EmptyIllustration />
                <EmptyTitle>这里还是空的</EmptyTitle>
                <EmptyDescription>添加你的第一个 Agent，开启自动化之旅</EmptyDescription>
              </EmptyHeader>
            </Empty>
          </TenantCard>
        </ShowcaseCard>

        <ShowcaseCard
          title="带操作引导（线框按钮）"
          desc="单按钮 — 主操作即引导"
          variantTag="tenant-outline"
        >
          <TenantCard padding="none">
            <Empty className="border-0 py-16">
              <EmptyHeader>
                <EmptyIllustration />
                <EmptyTitle>还没有 Agent</EmptyTitle>
                <EmptyDescription>来添加你的第一个 Agent 吧</EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button variant="tenant-outline">
                  <Plus className="w-4 h-4" />
                  添加 Agent
                </Button>
              </EmptyContent>
            </Empty>
          </TenantCard>
        </ShowcaseCard>

        <ShowcaseCard
          title="双按钮场景（最多 2 个）"
          desc="并列引导，全部线框，无主次差异"
          variantTag="tenant-outline × 2"
        >
          <TenantCard padding="none">
            <Empty className="border-0 py-16">
              <EmptyHeader>
                <EmptyIllustration />
                <EmptyTitle>暂无 MCP 配置</EmptyTitle>
                <EmptyDescription>从模板创建，或自己添加一个 MCP</EmptyDescription>
              </EmptyHeader>
              <EmptyContent className="flex-row gap-2">
                <Button variant="tenant-outline">
                  <Plus className="w-4 h-4" />
                  添加 MCP
                </Button>
                <Button variant="tenant-outline">
                  从模板创建
                </Button>
              </EmptyContent>
            </Empty>
          </TenantCard>
        </ShowcaseCard>

        <ShowcaseCard title="搜索无结果" desc="文案保持引导调性">
          <TenantCard padding="none">
            <Empty className="border-0 py-16">
              <EmptyHeader>
                <EmptyIllustration />
                <EmptyTitle>没找到匹配的内容</EmptyTitle>
                <EmptyDescription>试试换个关键词，或调整筛选条件</EmptyDescription>
              </EmptyHeader>
            </Empty>
          </TenantCard>
        </ShowcaseCard>

        <ShowcaseCard title="紧凑区域空态" desc="无外层卡片 / 无插画 / 双行纯文字" variantTag="HelperText × 2">
          <div className="text-center py-16 space-y-1">
            <HelperText>这里还没有内容</HelperText>
            <HelperText>来添加你的第一个项目吧</HelperText>
          </div>
        </ShowcaseCard>

        <ShowcaseCard title="表格空态（用户端）" desc="嵌入 td — 不加插画 / 双行纯文字" variantTag="HelperText × 2">
          <TenantCard padding="none">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell colSpan={2}>
                    <div className="text-center py-12 space-y-1">
                      <HelperText>暂无记录</HelperText>
                      <HelperText>尝试调整筛选条件，或新建一条记录</HelperText>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </TenantCard>
        </ShowcaseCard>
      </div>

      {/* 2. 浮层空态（同管控端） */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">
        2. 浮层 / 弹窗空态
      </h2>
      <p className="text-sm text-[#737373] mb-4">
        Dropdown / Popover / Dialog 内嵌空态<strong>与管控端完全一致</strong>，统一 <code className="font-mono text-xs">HelperText</code> 纯文字。
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="Dropdown 下拉" desc="单行 HelperText">
          <MockPanel title="选择技能">
            <div className="text-center py-6">
              <HelperText>暂无可选技能</HelperText>
            </div>
          </MockPanel>
        </ShowcaseCard>

        <ShowcaseCard title="Dialog 内嵌区块" desc="纯文字">
          <div className="bg-white border border-[#e5e5e5] rounded-[8px] p-5">
            <MetaMedium as="div" tone="secondary" className="mb-3">
              我的技能
            </MetaMedium>
            <div className="text-center py-12 space-y-1">
              <HelperText>还没有添加任何技能</HelperText>
              <HelperText>从技能广场添加你需要的技能</HelperText>
            </div>
          </div>
        </ShowcaseCard>
      </div>

      {/* 3. 行内空态（同管控端） */}
      <h2 className="text-lg font-semibold text-[#0A0A0A] mb-4">3. 行内空态</h2>
      <p className="text-sm text-[#737373] mb-4">
        与管控端完全一致：<code className="font-mono text-xs">MetaText tone="weak"</code>。
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="字段值为空">
          <div className="space-y-3">
            <div className="flex gap-3 text-sm">
              <MetaMedium as="span" tone="secondary" className="w-24">已使用模型</MetaMedium>
              <MetaText as="span" tone="weak">暂无</MetaText>
            </div>
            <div className="flex gap-3 text-sm">
              <MetaMedium as="span" tone="secondary" className="w-24">运行任务</MetaMedium>
              <MetaText as="span" tone="weak">暂无运行中的任务</MetaText>
            </div>
          </div>
        </ShowcaseCard>
      </div>

      {/* 4. 已废弃（用户端特有） */}
      <h2 className="text-lg font-semibold text-[#dc2626] mb-4">
        4. 用户端特有的废弃写法
      </h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-12">
        <ShowcaseCard title="❌ 用户端用 SurfaceCard" desc="应该用 TenantCard" variantTag="DEPRECATED">
          <div className="border border-dashed border-red-200 rounded-[4px]">
            <div className="bg-white rounded-[4px] border border-[#E5E5E5]">
              <Empty className="border-0 py-16">
                <EmptyHeader>
                  <EmptyIllustration />
                  <EmptyTitle>这里还是空的</EmptyTitle>
                </EmptyHeader>
              </Empty>
            </div>
          </div>
          <p className="mt-3 text-xs text-[#dc2626]">
            ❌ 用户端必须用 <code className="font-mono">TenantCard</code>（12px 圆角），<code className="font-mono">SurfaceCard</code> 仅限管控端
          </p>
        </ShowcaseCard>

        <ShowcaseCard title="❌ 用户端用 default 按钮" desc="应该用 tenant-primary" variantTag="DEPRECATED">
          <TenantCard padding="none">
            <div className="border border-dashed border-red-200 rounded-[12px]">
              <Empty className="border-0 py-16">
                <EmptyHeader>
                  <EmptyIllustration />
                  <EmptyTitle>还没有 Agent</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  {/* 故意用错的 default 按钮 */}
                  <Button>
                    <Plus className="w-4 h-4" />
                    添加 Agent
                  </Button>
                </EmptyContent>
              </Empty>
            </div>
          </TenantCard>
          <p className="mt-3 text-xs text-[#dc2626]">
            ❌ 用户端的按钮应该是<strong>圆角胶囊</strong>，请用 <code className="font-mono">variant="tenant-primary"</code>
          </p>
        </ShowcaseCard>
      </div>
    </div>
  );
}

/* -------------------------------------------------------------------- */
/* 主页面                                                                */
/* -------------------------------------------------------------------- */
export default function EmptyStatePreview() {
  return (
    <div className="page-enter min-h-screen bg-[#fafafa]">
      <div className="max-w-6xl mx-auto px-6 py-10">
        {/* 头部 */}
        <header className="mb-8">
          <h1 className="text-2xl font-semibold text-[#0A0A0A]">
            Empty 空状态组件总览
          </h1>
          <p className="mt-2 text-sm text-[#737373]">
            来源：SKILL-GLOBAL-COMPONENTS.md §24 · SKILL.md §10.1 / §8.7 · SKILL-TENANT.md §3 / §5
          </p>
          <p className="mt-1 text-xs text-[#a3a3a3]">
            按「容器类型」选答案，所有图标型空态统一使用兔子插画。
          </p>
        </header>

        {/* Tabs：管控端 / 用户端 */}
        <Tabs defaultValue="admin" className="w-full">
          <TabsList>
            <TabsTrigger value="admin">管控端规范（Admin）</TabsTrigger>
            <TabsTrigger value="tenant">用户端规范（Tenant）</TabsTrigger>
          </TabsList>

          <TabsContent value="admin" className="mt-6">
            <AdminPanel />
          </TabsContent>

          <TabsContent value="tenant" className="mt-6">
            <TenantPanel />
          </TabsContent>
        </Tabs>

        <footer className="border-t border-[#e5e5e5] pt-5 mt-12 text-xs text-[#a3a3a3]">
          本预览页位于 <code className="font-mono">/preview/empty-state</code>，
          所有示例直接调用 <code className="font-mono">@/components/ui/empty</code> 与 Typography 组件，
          视觉与业务页面完全一致。
        </footer>
      </div>
    </div>
  );
}
