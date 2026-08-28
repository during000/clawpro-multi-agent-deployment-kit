import type { ComponentType, ReactNode } from "react";
import TenantLayout from "@/components/TenantLayout";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { HelpIcon, BellIcon, SwitchAdminIcon } from "@/components/topnav/NavIcons";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { StatusBadge } from "@/components/agent/StatusBadge";
import {
  Activity,
  AlertCircle,
  ArrowLeft,
  ArrowRight,
  ArrowUpFromLine,
  Bot,
  BookOpen,
  Check,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Code,
  Copy,
  Download,
  Eye,
  EyeOff,
  FileText,
  Filter,
  HardDriveDownload,
  Info,
  LayoutGrid,
  List,
  Loader,
  MessageSquare,
  MessageSquarePlus,
  Mic,
  Monitor,
  MoreVertical,
  MousePointerClick,
  Plus,
  Puzzle,
  RefreshCw,
  Rocket,
  RotateCcw,
  Search,
  Send,
  Settings,
  ShieldCheck,
  Sparkles,
  Star,
  Terminal,
  Trash2,
  UserCog,
  UserMinus,
  X,
  XCircle,
  Zap,
} from "lucide-react";

type IconType = ComponentType<{ className?: string }>;

type UsageCategory = {
  title: string;
  scope: string;
  spec: string;
  modules: string[];
  icons: IconType[];
  note: string;
};

const usageCategories: UsageCategory[] = [
  {
    title: "全局导航与账号入口",
    scope: "TopNav / 用户菜单 / 通知抽屉",
    spec: "16×16 裸图标；按钮容器 px-2 py-[6px]，圆角 4px；用户头像 31×31 圆形首字母。",
    modules: ["components/topnav/NavIcons.tsx", "components/topnav/NavIconButton.tsx", "components/TenantLayout.tsx"],
    icons: [HelpIcon, BellIcon, SwitchAdminIcon, UserCog],
    note: "导航 icon 已有 Figma 导出自定义 SVG，视觉更贴稿；与 lucide 菜单图标并存。",
  },
  {
    title: "页面主操作与筛选",
    scope: "创建、返回、刷新、搜索、筛选、分页",
    spec: "主操作/搜索/分页基本为 16×16；表格小操作 12–14px；按钮容器多为 28/32/36px。",
    modules: ["pages/tenant/MyOpenClaw.tsx", "pages/tenant/SkillSquare.tsx", "pages/tenant/ModelQuota.tsx", "pages/tenant/HelpDocs.tsx"],
    icons: [Plus, Search, Filter, RefreshCw, ArrowLeft, ChevronLeft, ChevronRight],
    note: "尺寸总体稳定，但存在 w-3、w-3.5、w-4 混用，需要按层级沉淀规则。",
  },
  {
    title: "Agent 卡片操作",
    scope: "卡片右上角、下拉菜单、ID 复制、进入对话/终端",
    spec: "右上角容器 28×28，图标 16×16；ID 复制为 12×12；菜单图标 16×16 + mr-2。",
    modules: ["components/agent/AgentCard.tsx", "components/agent/ViewModeSegmented.tsx"],
    icons: [MoreVertical, RefreshCw, Settings, MessageSquare, Terminal, RotateCcw, HardDriveDownload, UserMinus, Trash2, Copy, Check],
    note: "操作 icon 密度高，当前依赖灰色 + hover 区分；危险操作仅文字/图标红色可再强化。",
  },
  {
    title: "数据统计与额度",
    scope: "模型额度统计卡、明细表、日期与信息提示",
    spec: "统计 icon 容器 32×32，内部 16×16 白色；分页 28×28；说明提示多为 16px 或 14px。",
    modules: ["pages/tenant/ModelQuota.tsx"],
    icons: [Activity, ArrowUpFromLine, Zap, Info, ChevronLeft, ChevronRight],
    note: "统计卡是最清晰的语义容器样式之一，可作为其它数据类 icon 的统一基准。",
  },
  {
    title: "帮助文档与内容分类",
    scope: "文档卡片、分类入口、文章返回",
    spec: "分类容器 40×40，内部 20×20 白色；卡片尾部箭头 16×16。",
    modules: ["pages/tenant/HelpDocs.tsx", "components/topnav/HelpPanel.tsx"],
    icons: [BookOpen, Star, Rocket, FileText, ChevronRight],
    note: "文档分类 icon 视觉完整，但与模型额度的 32px 容器不是同一规格。",
  },
  {
    title: "技能广场与安装分发",
    scope: "视图切换、技能卡、目录树、下载/安装/状态",
    spec: "搜索 16px；列表/网格切换 16px；技能卡操作 12–14px；空状态 48px。",
    modules: ["pages/tenant/SkillSquare.tsx"],
    icons: [LayoutGrid, List, Puzzle, Download, Plus, Code, Eye, ShieldCheck, CheckCircle2, XCircle, Loader],
    note: "技能卡同时使用 lucide、字母头像、状态圆点，信息密度最高，最值得优先统一。",
  },
  {
    title: "详情配置与安全输入",
    scope: "基础配置、模型/通道、技能弹窗、密钥可见性",
    spec: "表单内显隐 16×16；配置卡头部存在 24/32/40px 容器；加载态 14/32px。",
    modules: ["pages/tenant/OpenClawDetailGuide.tsx", "pages/tenant/ToolsMcpPanel.tsx", "pages/tenant/FileSpace.tsx"],
    icons: [Settings, Search, Plus, Eye, EyeOff, Copy, AlertCircle, Info, ArrowRight],
    note: "详情页 icon 使用面广，部分历史写法与新版容器/色阶存在差异。",
  },
  {
    title: "对话与云桌面",
    scope: "输入框、发送、语音、浏览器窗口、任务状态",
    spec: "输入/工具区 16px 为主；圆形发送按钮约 27×27；浏览器控制 16/20px。",
    modules: ["pages/tenant/ChatView.tsx"],
    icons: [Send, Mic, MessageSquarePlus, Monitor, MousePointerClick, Sparkles, X],
    note: "对话区是产品感最强的位置，发送按钮形态独立，应保留为专用规格。",
  },
];

const sizeSpecs = [
  { label: "12px", text: "表格/卡片弱操作", Icon: Info, className: "w-3 h-3", example: "SkillSquare：Info / Eye / Code" },
  { label: "14px", text: "行内操作与状态", Icon: Download, className: "w-3.5 h-3.5", example: "AgentCard ID 复制、技能安装" },
  { label: "16px", text: "导航/按钮/菜单主规格", Icon: RefreshCw, className: "w-4 h-4", example: "TopNav、创建、刷新、分页" },
  { label: "20px", text: "分类卡容器内部", Icon: BookOpen, className: "w-5 h-5", example: "HelpDocs 分类卡" },
  { label: "32px", text: "统计容器", Icon: Activity, className: "w-4 h-4", example: "ModelQuota StatCard" },
  { label: "48px", text: "头像/空状态", Icon: Bot, className: "w-12 h-12", example: "AgentAvatar、空状态" },
];

const auditStats = [
  { value: "10", label: "Tenant 页面", desc: "pages/tenant 下含 icon 页面" },
  { value: "4", label: "图标来源", desc: "lucide / 自定义 SVG / 头像图片 / 状态 SVG" },
  { value: "6", label: "常见尺寸", desc: "12 / 14 / 16 / 20 / 32 / 48" },
  { value: "8", label: "场景分类", desc: "从导航到对话区完整覆盖" },
];

function SectionTitle({ eyebrow, title, desc }: { eyebrow: string; title: string; desc?: string }) {
  return (
    <div className="mb-5">
      <p className="text-xs font-medium text-[#1447E6] mb-1">{eyebrow}</p>
      <h2 className="text-lg font-medium text-[#0A0A0A]">{title}</h2>
      {desc && <p className="text-sm text-[#737373] mt-1 leading-relaxed">{desc}</p>}
    </div>
  );
}

function IconChip({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className="flex flex-col items-center gap-2">
      <div className="w-9 h-9 rounded-[4px] bg-[#F5F5F5] border border-gray-200 flex items-center justify-center text-[#334155]">
        {children}
      </div>
      <span className="text-[11px] text-[#737373] leading-none">{label}</span>
    </div>
  );
}

function ModulePath({ children }: { children: ReactNode }) {
  return (
    <code className="inline-flex max-w-full rounded-[2px] bg-[#F5F5F5] px-1.5 py-0.5 text-[11px] text-[#334155] font-mono truncate">
      {children}
    </code>
  );
}

function UsageCategoryCard({ category }: { category: UsageCategory }) {
  return (
    <SurfaceCard className="rounded-[4px] p-5 flex flex-col gap-4">
      <div>
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-base font-semibold text-[#0A0A0A]">{category.title}</h3>
            <p className="text-xs text-[#737373] mt-1">{category.scope}</p>
          </div>
          <span className="rounded-[2px] bg-[#EFF6FF] px-2 py-1 text-xs font-medium text-[#1447E6] whitespace-nowrap">
            {category.icons.length} icons
          </span>
        </div>
      </div>

      <div className="flex flex-wrap gap-3">
        {category.icons.slice(0, 8).map((Icon, index) => (
          <IconChip key={`${category.title}-${index}`} label={Icon.displayName ?? Icon.name ?? "Icon"}>
            <Icon className="w-4 h-4" />
          </IconChip>
        ))}
      </div>

      <SurfaceInner className="rounded-[4px] p-3 space-y-2">
        <p className="text-xs leading-relaxed text-[#334155]">{category.spec}</p>
        <div className="flex flex-wrap gap-1.5">
          {category.modules.map((module) => (
            <ModulePath key={module}>{module}</ModulePath>
          ))}
        </div>
      </SurfaceInner>

      <p className="text-xs leading-relaxed text-[#737373]">视觉判断：{category.note}</p>
    </SurfaceCard>
  );
}

function TopNavSlice() {
  return (
    <SurfaceCard className="rounded-[4px] p-4">
      <div className="flex items-center justify-between border-b border-gray-200 pb-3 mb-4">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-[4px] flex items-center justify-center text-white" style={{ background: "linear-gradient(135deg, #1447E6, #2563EB)" }}>
            <Bot className="w-4 h-4" />
          </div>
          <span className="text-sm font-medium text-[#0A0A0A]">用户端导航</span>
        </div>
        <div className="flex items-center gap-1 text-[#020617]">
          <span className="w-8 h-8 rounded-[4px] flex items-center justify-center hover:bg-[#F5F5F5]"><HelpIcon /></span>
          <span className="relative w-8 h-8 rounded-[4px] flex items-center justify-center hover:bg-[#F5F5F5]"><BellIcon /><span className="absolute right-1.5 top-1.5 w-1 h-1 rounded-full bg-[#E85C5C]" /></span>
          <span className="h-5 w-px bg-[#E5E5E5]" />
          <span className="h-[34px] rounded-[20px] px-3 bg-[rgba(219,221,228,0.32)] flex items-center gap-2 hover:bg-[#F5F5F5]"><SwitchAdminIcon /><span className="text-sm">管控端</span></span>
        </div>
      </div>
      <p className="text-xs text-[#737373]">来源：自定义 Figma SVG + lucide 菜单图标。核心规格为 16×16。</p>
    </SurfaceCard>
  );
}

function AgentCardSlice() {
  return (
    <SurfaceCard className="rounded-[4px] p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-4 min-w-0">
          <AgentAvatar roleName="设计师" agentName="设计助手" size={48} />
          <div className="min-w-0">
            <p className="text-sm font-medium text-[#0A0A0A]">设计师的工作助手</p>
            <div className="mt-1"><StatusBadge status="running" /></div>
          </div>
        </div>
        <div className="flex items-center gap-1 text-[#737373]">
          <button className="w-7 h-7 rounded-[4px] flex items-center justify-center hover:bg-[#F5F5F5]"><RefreshCw className="w-4 h-4" /></button>
          <button className="w-7 h-7 rounded-[4px] flex items-center justify-center hover:bg-[#F5F5F5]"><MoreVertical className="w-4 h-4" /></button>
        </div>
      </div>
      <div className="mt-5 flex items-center justify-between gap-3">
        <div className="text-xs text-[#737373] flex items-center gap-1.5">
          ID: ins-g83c6wvc <Copy className="w-3 h-3" />
        </div>
        <div className="flex items-center gap-2">
          <button className="h-8 px-3 rounded-[4px] border border-gray-200 text-xs text-[#334155] flex items-center gap-1.5"><Settings className="w-3.5 h-3.5" />详细配置</button>
          <button className="h-8 px-3 rounded-[4px] border border-gray-200 text-xs text-[#334155] flex items-center gap-1.5"><MessageSquare className="w-3.5 h-3.5" />对话</button>
        </div>
      </div>
      <p className="mt-4 text-xs text-[#737373]">来源：AgentAvatar 图片资产 + StatusBadge 内联状态 SVG + lucide 操作图标。</p>
    </SurfaceCard>
  );
}

function QuotaSlice() {
  const items = [
    { Icon: Activity, label: "请求数", value: "2,186", cls: "bg-gradient-to-br from-blue-500 to-blue-600" },
    { Icon: ArrowUpFromLine, label: "输入 Tokens", value: "318k", cls: "bg-gradient-to-br from-indigo-500 to-indigo-600" },
    { Icon: Zap, label: "总 Tokens", value: "512k", cls: "bg-gradient-to-br from-blue-600 to-purple-600" },
  ];
  return (
    <SurfaceCard className="rounded-[4px] p-4">
      <div className="grid grid-cols-3 gap-3">
        {items.map(({ Icon, label, value, cls }) => (
          <SurfaceInner key={label} className="rounded-[4px] p-3">
            <div className="flex items-center gap-2.5 mb-3">
              <div className={`w-8 h-8 rounded-[4px] flex items-center justify-center ${cls}`}>
                <Icon className="w-4 h-4 text-white" />
              </div>
              <span className="text-xs text-[#737373]">{label}</span>
            </div>
            <p className="font-din text-2xl font-bold tabular-nums text-[#020617]">{value}</p>
          </SurfaceInner>
        ))}
      </div>
      <p className="mt-4 text-xs text-[#737373]">来源：ModelQuota StatCard。当前 32×32 方形容器更紧凑。</p>
    </SurfaceCard>
  );
}

function DocsSlice() {
  const docs = [
    { Icon: BookOpen, label: "概念介绍", cls: "bg-gradient-to-br from-blue-500 to-blue-600" },
    { Icon: Star, label: "功能特色", cls: "bg-gradient-to-br from-purple-500 to-purple-600" },
    { Icon: Rocket, label: "部署指引", cls: "bg-gradient-to-br from-green-500 to-green-600" },
  ];
  return (
    <SurfaceCard className="rounded-[4px] p-4">
      <div className="grid grid-cols-3 gap-3">
        {docs.map(({ Icon, label, cls }) => (
          <SurfaceInner key={label} className="rounded-[4px] p-3 flex items-center gap-3">
            <div className={`w-10 h-10 rounded-[4px] flex items-center justify-center ${cls}`}>
              <Icon className="w-5 h-5 text-white" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-[#0A0A0A] truncate">{label}</p>
              <p className="text-xs text-[#737373]">40 / 20 规格</p>
            </div>
            <ChevronRight className="w-4 h-4 text-[#A3A3A3]" />
          </SurfaceInner>
        ))}
      </div>
      <p className="mt-4 text-xs text-[#737373]">来源：HelpDocs / HelpPanel。视觉更醒目，适合分类入口。</p>
    </SurfaceCard>
  );
}

function SkillSlice() {
  return (
    <SurfaceCard className="rounded-[4px] p-4">
      <div className="flex items-center gap-3 mb-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
          <div className="h-9 rounded-[4px] border border-gray-200 bg-white pl-9 pr-3 flex items-center text-xs text-[#A3A3A3]">搜索技能、插件或 MCP</div>
        </div>
        <button className="w-9 h-9 rounded-[4px] border border-gray-200 flex items-center justify-center text-[#1447E6]"><LayoutGrid className="w-4 h-4" /></button>
        <button className="w-9 h-9 rounded-[4px] border border-gray-200 flex items-center justify-center text-[#737373]"><List className="w-4 h-4" /></button>
      </div>
      <SurfaceInner className="rounded-[4px] p-4">
        <div className="flex items-start gap-3">
          <div className="w-12 h-12 rounded-[4px] flex items-center justify-center text-xl font-semibold bg-[#E8F4FD] text-[#1A73E8]">A</div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-sm font-medium text-[#0A0A0A]">agent-browser</p>
              <ShieldCheck className="w-3.5 h-3.5 text-green-500" />
            </div>
            <p className="text-xs text-[#737373] mt-1">字母头像 + 12/14px 小操作 icon。</p>
          </div>
          <button className="w-7 h-7 rounded-[4px] border border-gray-200 flex items-center justify-center text-[#1447E6]"><Plus className="w-3.5 h-3.5" /></button>
        </div>
      </SurfaceInner>
      <p className="mt-4 text-xs text-[#737373]">来源：SkillSquare。当前混合字母头像、lucide、状态图标，需重点看统一性。</p>
    </SurfaceCard>
  );
}

function ChatSlice() {
  return (
    <SurfaceCard className="rounded-[4px] p-4">
      <div className="flex items-center gap-3 mb-4">
        <div className="w-9 h-9 rounded-[4px] bg-[#EFF6FF] flex items-center justify-center text-[#1447E6]"><Monitor className="w-5 h-5" /></div>
        <div>
          <p className="text-sm font-medium text-[#0A0A0A]">对话 + 云桌面</p>
          <p className="text-xs text-[#737373]">ChatView 专用操作区</p>
        </div>
      </div>
      <div className="rounded-[4px] border border-gray-200 px-3 py-2 flex items-center gap-2">
        <Sparkles className="w-4 h-4 text-[#1447E6]" />
        <span className="flex-1 text-xs text-[#737373]">输入消息或让 Agent 操作浏览器...</span>
        <Mic className="w-4 h-4 text-[#A3A3A3]" />
        <button className="w-[27px] h-[27px] rounded-full flex items-center justify-center text-white" style={{ background: "linear-gradient(90deg, #020617 70%, #1447E6 100%)" }}>
          <Send className="w-3.5 h-3.5" />
        </button>
      </div>
      <div className="mt-3 flex items-center gap-2 text-xs text-[#737373]">
        <MousePointerClick className="w-4 h-4" />浏览器可操作
        <MessageSquarePlus className="w-4 h-4 ml-2" />新对话
      </div>
      <p className="mt-4 text-xs text-[#737373]">来源：ChatView。圆形发送按钮是独立产品形态。</p>
    </SurfaceCard>
  );
}

export default function TenantIconAudit() {
  return (
    <TenantLayout>
      <div className="min-w-[1200px] overflow-x-clip">
        <div className="max-w-[1920px] mx-auto flex items-stretch page-enter min-h-[calc(100vh-64px)]">
          <div aria-hidden className="shrink-0 w-20 self-stretch" />
          <div className="flex-1 min-w-0 px-[42px] py-8">
            <header className="mb-8">
              <div className="flex items-start justify-between gap-8">
                <div>
                  <p className="text-xs font-medium text-[#1447E6] mb-2">Icon Audit / Tenant</p>
                  <h1 className="text-2xl font-medium text-[#0A0A0A]">用户端 Icon 全景盘点</h1>
                  <p className="mt-3 max-w-3xl text-sm leading-relaxed text-[#334155]">
                    按使用场景、出现位置和规格把当前用户端 icon 拉出来做视觉看板。这里不改动业务逻辑，只用于观察整体一致性、密度与可优化方向。
                  </p>
                </div>
                <SurfaceInner className="rounded-[4px] px-4 py-3 shrink-0">
                  <p className="text-xs text-[#737373]">扫描范围</p>
                  <p className="text-sm font-medium text-[#0A0A0A] mt-1">Tenant 页面 + 用户端公共组件</p>
                  <p className="text-xs text-[#A3A3A3] mt-1">lucide-react / 自定义 SVG / 图片头像 / 状态 SVG</p>
                </SurfaceInner>
              </div>
            </header>

            <section className="grid grid-cols-4 gap-4 mb-10">
              {auditStats.map((item) => (
                <SurfaceCard key={item.label} className="rounded-[4px] p-5">
                  <p className="font-din text-2xl font-bold tabular-nums text-[#020617]">{item.value}</p>
                  <p className="text-sm font-medium text-[#0A0A0A] mt-2">{item.label}</p>
                  <p className="text-xs text-[#737373] mt-1">{item.desc}</p>
                </SurfaceCard>
              ))}
            </section>

            <section className="mb-10">
              <SectionTitle
                eyebrow="01 / Size Ladder"
                title="规格刻度尺"
                desc="当前用户端 icon 基本围绕 12、14、16、20、32、48 六档展开，其中 16px 是主规格。"
              />
              <SurfaceCard className="rounded-[4px] p-5">
                <div className="grid grid-cols-6 gap-4">
                  {sizeSpecs.map(({ label, text, Icon, className, example }) => (
                    <SurfaceInner key={label} className="rounded-[4px] p-4 flex flex-col items-center text-center">
                      <div className="h-14 flex items-center justify-center text-[#1447E6]">
                        {label === "32px" ? (
                          <div className="w-8 h-8 rounded-[4px] bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
                            <Icon className="w-4 h-4 text-white" />
                          </div>
                        ) : (
                          <Icon className={`${className} ${label === "48px" ? "text-[#E5E5E5]" : "text-[#334155]"}`} />
                        )}
                      </div>
                      <p className="text-sm font-medium text-[#0A0A0A]">{label}</p>
                      <p className="text-xs text-[#737373] mt-1">{text}</p>
                      <p className="text-[11px] text-[#A3A3A3] mt-2 leading-relaxed">{example}</p>
                    </SurfaceInner>
                  ))}
                </div>
              </SurfaceCard>
            </section>

            <section className="mb-10">
              <SectionTitle
                eyebrow="02 / Usage Categories"
                title="按使用场景分类"
                desc="每类展示代表图标、所在组件/模块、当前规格和视觉观察。"
              />
              <div className="grid grid-cols-2 gap-4">
                {usageCategories.map((category) => (
                  <UsageCategoryCard key={category.title} category={category} />
                ))}
              </div>
            </section>

            <section className="mb-10">
              <SectionTitle
                eyebrow="03 / Module Samples"
                title="代表模块切片"
                desc="把典型使用位置抽出来放在同一页，便于横向比较容器尺寸、颜色、线性粗细和信息密度。"
              />
              <div className="grid grid-cols-2 gap-4">
                <TopNavSlice />
                <AgentCardSlice />
                <QuotaSlice />
                <DocsSlice />
                <SkillSlice />
                <ChatSlice />
              </div>
            </section>

            <section>
              <SectionTitle eyebrow="04 / Visual Findings" title="初步视觉判断与优化候选" />
              <div className="grid grid-cols-3 gap-4">
                <SurfaceCard className="rounded-[4px] p-5 border-l-4 border-l-[#16A34A]">
                  <h3 className="text-sm font-semibold text-[#0A0A0A] mb-2">当前比较稳的部分</h3>
                  <ul className="space-y-2 text-xs leading-relaxed text-[#334155]">
                    <li>• 16px 已成为导航、按钮、菜单的主规格。</li>
                    <li>• Agent 卡片状态 SVG 语义明确，和业务状态绑定紧。</li>
                    <li>• 模型额度统计 icon 容器清晰，适合作为数据类基准。</li>
                  </ul>
                </SurfaceCard>
                <SurfaceCard className="rounded-[4px] p-5 border-l-4 border-l-[#F59E0B]">
                  <h3 className="text-sm font-semibold text-[#0A0A0A] mb-2">需要观察的混用</h3>
                  <ul className="space-y-2 text-xs leading-relaxed text-[#334155]">
                    <li>• lucide、自定义 SVG、图片头像、字母头像、状态 SVG 并存。</li>
                    <li>• 同类容器存在 32px、36px、40px、48px 多规格。</li>
                    <li>• 灰色使用含 muted、gray 与 hex token，观感可能不一致。</li>
                  </ul>
                </SurfaceCard>
                <SurfaceCard className="rounded-[4px] p-5 border-l-4 border-l-[#1447E6]">
                  <h3 className="text-sm font-semibold text-[#0A0A0A] mb-2">建议优先优化</h3>
                  <ul className="space-y-2 text-xs leading-relaxed text-[#334155]">
                    <li>• 先统一“功能入口容器”：数据 32px、分类 40px、头像 48px。</li>
                    <li>• 技能广场优先梳理小图标密度与状态色。</li>
                    <li>• 把 12/14/16px 操作 icon 的使用边界写入规范。</li>
                  </ul>
                </SurfaceCard>
              </div>
            </section>
          </div>
          <div aria-hidden className="shrink-0 w-20 self-stretch" />
        </div>
      </div>
    </TenantLayout>
  );
}
