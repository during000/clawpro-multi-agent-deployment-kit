/**
 * SkillMapAbDemo — resource-skill-map 选图效果演示（融入真实管理端版）
 * ----------------------------------------------------------------------------
 * 目的：不再把槽位硬塞进一个「为展示而拼凑」的总览页，而是
 *   「在真实管理端外壳里，复刻 4 个真实存在的管理页」，让槽位资源自然落在
 *   它本就归属的页面/组件里，自证「这套候选拿去就能拼出真页面」。
 *
 * 承载方式：单一入口 /preview/skill-map-ab，复用真实 <AdminLayout>（真实侧栏 +
 *   真实 Logo），并通过 previewNavItem 在真实侧栏顶部注入一个高亮的「测试demo」
 *   菜单项（仅 demo 页传该 prop，真实管理页零影响）；顶部用 Tab 在 4 个管理页
 *   之间切换。
 *
 * 每个 Tab = 1 个「主角业务槽位」+ 若干「陪衬模块」（统计行 / 工具条 / 明细表），
 *   让页面像「正在运行的管理台」。陪衬数据为新编、彼此自洽（如 Tokens 明细表
 *   各行之和恰好等于上方指标卡总数）。
 *
 * 覆盖的 6 个 admin 槽位（3 个 tenant 槽位 agent-avatar / file-type / run-status
 *   在管理端无真实落点，按确认结论不纳入）：
 *   - brand-logo   → 真实 AdminLayout 内置 AdminSidebarLogo（host-injected，红线合规）
 *   - admin-sidebar→ 真实 ADMIN_NAV_GROUPS 的 /assets/admin-sidebar/*.svg
 *   - card-left-icon → 复刻「基础信息」页平台基础信息卡（/icon/*.svg，w-9 h-9）
 *   - number-card  → 复刻「Tokens 监控」页，复用 owning 组件 NumberCard
 *   - channel-icon → 复刻「通道配置」页内置通道表，走 canonicalAssets.channels
 *   - feature-card → 复刻「记忆管理」页版本对比表，直接复用真实 ComparisonTable
 *
 * 纪律：业务图标一律取自真实页面在用的路径 / canonical 入口 / owning 组件，
 *   不臆造路径、不臆造尺寸；webPath 引用统一走 SafeImg（onError 裂图探针）；
 *   KPI / 数字统计卡一律用 owning 组件 NumberCard（SKILL §2.9，禁止手搓自拼卡），
 *   其图标取真实页 OpenClawMonitor 主卡范式 = number-card 槽位的已登记渐变 SVG
 *   （18×18，#202020→#0080FF，如 已开通智能体网盘 / AI Agent资产），而非 lucide 扁平；
 *   仅在该槽位无对应业务图标时（如「调用配额」）回退 lucide，并套同款渐变描边与之同族；
 *   陪衬表格做成「纯文本 + StatusTag + lucide」零业务图标，不新增裂图面。
 */
import { useState } from "react";
import { Link } from "wouter";
import {
  ChevronRight,
  FileDown,
  Gauge,
  Home,
  ImageOff,
  Search,
} from "lucide-react";

import AdminLayout from "@/components/AdminLayout";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { SurfaceCard } from "@/components/ui/Surface";
import { StatusTag, type StatusTagColor } from "@/components/ui/status-tag";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import { BodyMedium, MetaText, PanelTitle } from "@/components/ui/Typography";

// SLOT: number-card → owning 组件 NumberCard + 内置渐变图标（与 Tokens 监控 1:1）
import {
  NumberCard,
  RequestsIcon,
  InputTokensIcon,
  OutputTokensIcon,
  TotalTokensIcon,
} from "@/components/ui/number-card";

// SLOT: feature-card → 直接复用记忆管理页真实组件（内含 EnterpriseFeatureCard 16px）
import { ComparisonTable } from "@/pages/admin/MemoryManagement/components/ComparisonTable";

// SLOT: channel-icon → 渠道图标统一入口（与通道配置页同源，一改多处生效）
import { canonicalAssets } from "@/design-assets/canonical-assets";

/* ----------------------------------------------------------------
 * 裂图探针：带 onError，任一 webPath 失效会立刻可见
 * ---------------------------------------------------------------- */
function SafeImg({
  src,
  alt,
  className,
}: {
  src: string;
  alt: string;
  className?: string;
}) {
  const [broken, setBroken] = useState(false);
  if (broken) {
    return (
      <span
        className={`inline-flex items-center justify-center bg-red-50 text-red-500 ${className ?? ""}`}
        title={`裂图：${src}`}
      >
        <ImageOff className="h-4 w-4" />
      </span>
    );
  }
  return (
    <img src={src} alt={alt} className={className} onError={() => setBroken(true)} />
  );
}

/* ----------------------------------------------------------------
 * KPI / 统计卡：一律走 owning 组件 NumberCard（SKILL §2.9 强制）。
 *   - 图标 = number-card 槽位已登记的渐变 SVG（18×18，#202020→#0080FF），
 *     与真实页 OpenClawMonitor 主 KPI 卡（monitor-*.svg / instance-*.svg）同范式。
 *   - 「调用配额」该槽位无对应业务图标 → lucide Gauge 兜底，但套同款渐变描边
 *     （QuotaGaugeIcon），与另两张渐变 SVG 同族，不再是孤立的扁平品牌色。
 * KpiExtra：把「二级文字 + 进度条」塞进 NumberCard 的 extra 槽（落数字右侧、视觉右下），
 *   而非 footer 通栏铺底。
 * ---------------------------------------------------------------- */
function QuotaGaugeIcon() {
  // lucide Gauge + 与 number-card 渐变 SVG 同款描边渐变（蓝→黑，右下→左上）
  return (
    <span className="relative inline-flex h-[18px] w-[18px]">
      <svg width="0" height="0" className="absolute" aria-hidden="true">
        <defs>
          <linearGradient id="kpi-quota-grad" x1="1" y1="1" x2="0" y2="0">
            <stop offset="0" stopColor="#0080FF" />
            <stop offset="1" stopColor="#202020" />
          </linearGradient>
        </defs>
      </svg>
      <Gauge className="h-[18px] w-[18px]" style={{ stroke: "url(#kpi-quota-grad)" }} />
    </span>
  );
}

function KpiExtra({ sub, percent }: { sub: string; percent: number }) {
  return (
    <div className="w-full">
      <MetaText as="p" className="mb-1.5 text-right leading-none">{sub}</MetaText>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-[#EEF2FF]">
        <div
          className="h-full rounded-full bg-[#1447E6]"
          style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
        />
      </div>
    </div>
  );
}

/* ================================================================
 * 页面① 基础信息 —— SLOT: card-left-icon（/icon/*.svg，w-9 h-9 rounded-[4px]）
 * 复刻 BasicInfo 右侧「平台基础信息 / API 文档」卡片真实用法
 * 陪衬：资源配额统计行 + 近期操作记录表
 * ================================================================ */
/* card-left-icon 槽位入口卡：均为已登记 36px 渐变磁贴（gradient-solid），
   webPath 真实存在；做成统一的「平台快捷入口」四宫格，替代原先
   「3 项挤一行 + 孤零零 API 文档 banner」的不齐布局。 */
const ENTRY_CARDS = [
  { icon: "/icon/api文档-icon.svg", title: "API 文档", desc: "查阅开放接口与调用示例" },
  { icon: "/icon/SkillHub 地址-icon.svg", title: "SkillHub 地址", desc: "浏览平台公共技能库" },
  { icon: "/assets/admin-security/audit-tracing.svg", title: "审计溯源", desc: "查看平台操作审计日志" },
  {
    icon: "/assets/admin-session-management/business-health-monitoring.svg",
    title: "业务健康监控",
    desc: "实时监控会话与业务健康",
  },
];

const INFO_LOGS: {
  time: string;
  operator: string;
  action: string;
  status: string;
  variant: StatusTagColor;
}[] = [
  { time: "2026-06-22 16:42", operator: "jingsujiang", action: "更新平台访问域名", status: "成功", variant: "green" },
  { time: "2026-06-22 14:08", operator: "liuwei", action: "新增关联腾讯云账号", status: "成功", variant: "green" },
  { time: "2026-06-21 19:25", operator: "system", action: "自动同步基础信息", status: "成功", variant: "green" },
  { time: "2026-06-21 10:11", operator: "chenhao", action: "修改 API 文档地址", status: "已回滚", variant: "orange" },
  { time: "2026-06-20 09:30", operator: "jingsujiang", action: "校验地域配置", status: "失败", variant: "red" },
];

function BasicInfoPage() {
  return (
    <>
      <AdminPageHeader
        title="基础信息配置"
        description="平台基础信息只读展示；左侧卡片图标取自 card-left-icon 槽位真实候选（/icon/*.svg）"
      />

      {/* 陪衬：资源配额统计行 —— 统一用 owning 组件 NumberCard（SKILL §2.9）。
         图标取 number-card 槽位已登记渐变 SVG（同 OpenClawMonitor 主卡范式）：
         存储→已开通智能体网盘、Agent→AI Agent资产；配额无对应业务图标，
         用 lucide Gauge 套同款渐变描边兜底。进度条入 extra 槽 → 落数字右下。 */}
      <div className="mb-5 grid grid-cols-1 gap-5 sm:grid-cols-3">
        <NumberCard
          icon={<QuotaGaugeIcon />}
          label="本月调用配额"
          value="57%"
          extra={<KpiExtra sub="11,460 / 20,000 次" percent={57} />}
        />
        <NumberCard
          icon={
            <SafeImg
              src="/icon/已开通智能体网盘.svg"
              alt=""
              className="h-[18px] w-[18px] shrink-0"
            />
          }
          label="存储用量"
          value="8.6 GB"
          extra={<KpiExtra sub="共 50 GB" percent={17} />}
        />
        <NumberCard
          icon={
            <SafeImg
              src="/icon/AI Agent资产.svg"
              alt=""
              className="h-[18px] w-[18px] shrink-0"
            />
          }
          label="在线 Agent"
          value="6 / 10"
          extra={<KpiExtra sub="4 个空闲" percent={60} />}
        />
      </div>

      {/* 纵向全宽分区：
         ① 平台快捷入口（主角 card-left-icon，4 张同构渐变磁贴入口卡，2×2 / 整行自适应）
         ② 近期操作记录（全宽表格，列自适应不溢出 → 无常显滚动条） */}
      <div className="flex flex-col gap-5">
        {/* ① 平台快捷入口（card-left-icon，/icon + /assets 已登记 36px 渐变磁贴，w-9 h-9 rounded-[4px]） */}
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-4">
          {ENTRY_CARDS.map((c) => (
            <SurfaceCard
              key={c.title}
              className="group flex cursor-pointer items-center gap-4 p-5 transition-colors"
            >
              <SafeImg
                src={c.icon}
                alt=""
                className="h-9 w-9 shrink-0 rounded-[4px]"
              />
              <div className="flex min-w-0 flex-col gap-0.5">
                <p className="truncate text-sm font-medium text-[var(--text-emphasis)] transition-colors group-hover:text-[var(--text-brand)]">
                  {c.title}
                </p>
                <MetaText as="p" className="truncate">{c.desc}</MetaText>
              </div>
              <ChevronRight className="ml-auto size-4 shrink-0 text-[var(--text-muted)] opacity-0 transition-opacity group-hover:opacity-100" />
            </SurfaceCard>
          ))}
        </div>

        {/* ② 近期操作记录（全宽表格） */}
        <div>
          <PanelTitle as="p" className="mb-3">近期操作记录</PanelTitle>
          <SurfaceCard className="overflow-hidden">
            <Table variant="white">
              <TableHeader>
                <TableRow>
                  <TableHead style={{ width: 180 }}>时间</TableHead>
                  <TableHead style={{ width: 160 }}>操作人</TableHead>
                  <TableHead style={{ width: "100%" }}>动作</TableHead>
                  <TableHead style={{ width: 100 }}>结果</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {INFO_LOGS.map((log) => (
                  <TableRow key={log.time}>
                    <TableCell>
                      <MetaText as="span" tone="secondary">{log.time}</MetaText>
                    </TableCell>
                    <TableCell>
                      <BodyMedium tone="primary">{log.operator}</BodyMedium>
                    </TableCell>
                    <TableCell>
                      <MetaText as="span" tone="secondary">{log.action}</MetaText>
                    </TableCell>
                    <TableCell>
                      <StatusTag mode="text" variant={log.variant}>
                        {log.status}
                      </StatusTag>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </SurfaceCard>
        </div>
      </div>
    </>
  );
}

/* ================================================================
 * 页面② Tokens 监控 —— SLOT: number-card（复用 owning 组件 NumberCard）
 * 陪衬：时间范围工具条 + 各模型用量明细表（各行之和 = 指标卡总数，自洽）
 * ================================================================ */
const STAT_CARDS = [
  { label: "总请求数", value: "1,841", Icon: RequestsIcon },
  { label: "输入 Tokens", value: "533,112", Icon: InputTokensIcon },
  { label: "输出 Tokens", value: "419,040", Icon: OutputTokensIcon },
  { label: "总 Tokens", value: "952,152", Icon: TotalTokensIcon },
];

const TOKEN_RANGES = [
  { id: "today", label: "今日" },
  { id: "7d", label: "近 7 天" },
  { id: "30d", label: "近 30 天" },
];

// 4 行明细：请求数合计 1,841 / 输入合计 533,112 / 输出合计 419,040（与上方卡片一致）
const MODEL_USAGE: {
  model: string;
  requests: string;
  input: string;
  output: string;
  status: string;
  variant: StatusTagColor;
}[] = [
  { model: "DeepSeek-V3", requests: "982", input: "301,440", output: "248,910", status: "正常", variant: "green" },
  { model: "Qwen2.5-Max", requests: "514", input: "168,220", output: "121,330", status: "正常", variant: "green" },
  { model: "文本向量-v2", requests: "287", input: "52,110", output: "0", status: "正常", variant: "green" },
  { model: "GLM-4-Air", requests: "58", input: "11,342", output: "48,800", status: "限流", variant: "orange" },
];

function TokensMonitorPage() {
  const [range, setRange] = useState("7d");
  return (
    <>
      <AdminPageHeader
        title="Tokens 监控"
        description="概览指标卡复用 owning 组件 NumberCard 及其内置渐变图标（number-card 槽位）"
      />

      {/* 陪衬：时间范围工具条 */}
      <div className="mb-5 flex items-center justify-between gap-3">
        <div className="inline-flex items-center gap-1 rounded-[6px] border border-[var(--border)] bg-white p-1">
          {TOKEN_RANGES.map((r) => (
            <button
              key={r.id}
              onClick={() => setRange(r.id)}
              className={`rounded-[4px] px-3 py-1 text-sm transition-colors ${
                range === r.id
                  ? "bg-[#EEF2FF] text-[#1447E6]"
                  : "text-[var(--text-muted)] hover:text-[var(--text-title)]"
              }`}
            >
              {r.label}
            </button>
          ))}
        </div>
        <button className="inline-flex items-center gap-1.5 rounded-[4px] border border-[var(--border)] bg-white px-3 py-1.5 text-sm text-[var(--text-body)] transition-colors hover:text-[var(--text-title)]">
          <FileDown className="size-4" /> 导出
        </button>
      </div>

      {/* 主角 number-card */}
      <div className="grid grid-cols-2 gap-5 lg:grid-cols-4">
        {STAT_CARDS.map(({ label, value, Icon }) => (
          <NumberCard key={label} icon={<Icon />} label={label} value={value} />
        ))}
      </div>

      {/* 陪衬：各模型用量明细表 */}
      <div className="mt-5">
        <PanelTitle as="p" className="mb-3">各模型用量明细</PanelTitle>
        <SurfaceCard className="overflow-hidden">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: 220, minWidth: 180 }}>模型</TableHead>
                <TableHead style={{ width: 140, minWidth: 120 }}>请求数</TableHead>
                <TableHead style={{ width: 160, minWidth: 140 }}>输入 Tokens</TableHead>
                <TableHead style={{ width: 160, minWidth: 140 }}>输出 Tokens</TableHead>
                <TableHead style={{ width: 120, minWidth: 100 }}>状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {MODEL_USAGE.map((m) => (
                <TableRow key={m.model}>
                  <TableCell>
                    <BodyMedium tone="primary">{m.model}</BodyMedium>
                  </TableCell>
                  <TableCell>
                    <MetaText as="span" tone="secondary">{m.requests}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText as="span" tone="secondary">{m.input}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText as="span" tone="secondary">{m.output}</MetaText>
                  </TableCell>
                  <TableCell>
                    <StatusTag mode="text" variant={m.variant}>{m.status}</StatusTag>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      </div>
    </>
  );
}

/* ================================================================
 * 页面③ 通道配置 —— SLOT: channel-icon（canonicalAssets.channels）
 * 复刻 ChannelConfig 内置通道表真实用法
 * 陪衬：搜索工具条 + 今日消息数 / 接入状态两列
 * ================================================================ */
const CHANNEL_ICON_SRC: Record<string, string> = {
  wechat: canonicalAssets.channels.wechat,
  qq: canonicalAssets.channels.qq,
  wework: canonicalAssets.channels.wecom,
  "wework-app": canonicalAssets.channels.wecomApp,
  dingtalk: canonicalAssets.channels.dingtalk,
  feishu: canonicalAssets.channels.feishu,
};

const BUILTIN_CHANNELS: {
  id: string;
  name: string;
  visible: boolean;
  today: string;
  connected: boolean;
}[] = [
  { id: "wechat", name: "微信", visible: true, today: "1,284", connected: true },
  { id: "qq", name: "QQ", visible: false, today: "—", connected: false },
  { id: "wework", name: "企业微信", visible: true, today: "892", connected: true },
  { id: "wework-app", name: "企业微信应用", visible: false, today: "—", connected: false },
  { id: "dingtalk", name: "钉钉", visible: true, today: "417", connected: true },
  { id: "feishu", name: "飞书", visible: true, today: "236", connected: true },
];

function ChannelConfigPage() {
  const [keyword, setKeyword] = useState("");
  const [visibility, setVisibility] = useState<Record<string, boolean>>(
    () => Object.fromEntries(BUILTIN_CHANNELS.map((c) => [c.id, c.visible])),
  );
  const rows = BUILTIN_CHANNELS.filter((c) => c.name.includes(keyword.trim()));
  const connectedCount = BUILTIN_CHANNELS.filter((c) => c.connected).length;

  return (
    <>
      <AdminPageHeader
        title="通道配置"
        description="内置通道图标走 canonicalAssets.channels 统一入口（channel-icon 槽位），与真实通道配置页同源"
      />

      {/* 陪衬：搜索工具条 */}
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="flex h-8 w-[260px] items-center gap-2 rounded-[4px] border border-[var(--border)] bg-white px-2.5">
          <Search className="size-4 shrink-0 text-[var(--text-muted)]" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索通道名称"
            className="min-w-0 flex-1 bg-transparent text-sm text-[var(--text-body)] outline-none placeholder:text-[var(--text-muted)]"
          />
        </div>
        <MetaText as="span" tone="secondary">
          共 {BUILTIN_CHANNELS.length} 个通道 · {connectedCount} 个已接入
        </MetaText>
      </div>

      {/* 主角 channel-icon */}
      <SurfaceCard className="overflow-hidden">
        <Table variant="white">
          <TableHeader>
            <TableRow>
              <TableHead style={{ minWidth: 240 }}>产品</TableHead>
              <TableHead style={{ width: "100%", minWidth: 180 }}>说明</TableHead>
              <TableHead style={{ width: 130, minWidth: 110 }}>今日消息数</TableHead>
              <TableHead style={{ width: 120, minWidth: 100 }}>接入状态</TableHead>
              <TableHead style={{ width: 120, minWidth: 100 }}>用户可见</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((ch) => (
              <TableRow key={ch.id}>
                <TableCell>
                  <div className="flex items-center gap-3">
                    <span
                      className="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-[8px] border border-[var(--border)]"
                      aria-hidden="true"
                    >
                      <SafeImg
                        src={CHANNEL_ICON_SRC[ch.id]}
                        alt=""
                        className="h-full w-full scale-[1.04] object-contain"
                      />
                    </span>
                    <BodyMedium tone="primary">{ch.name}</BodyMedium>
                  </div>
                </TableCell>
                <TableCell>
                  <MetaText as="span" tone="secondary">
                    通过 {ch.name} 机器人接入，支持智能对话
                  </MetaText>
                </TableCell>
                <TableCell>
                  <MetaText as="span" tone="secondary">{ch.today}</MetaText>
                </TableCell>
                <TableCell>
                  <StatusTag mode="text" variant={ch.connected ? "green" : "gray"}>
                    {ch.connected ? "已接入" : "未接入"}
                  </StatusTag>
                </TableCell>
                <TableCell>
                  <Switch
                    checked={visibility[ch.id] || false}
                    onCheckedChange={(v) =>
                      setVisibility((prev) => ({ ...prev, [ch.id]: v }))
                    }
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </SurfaceCard>
    </>
  );
}

/* ================================================================
 * 页面④ 记忆管理 —— SLOT: feature-card（直接复用真实 ComparisonTable，16px）
 * 陪衬：用量统计行 + 记忆条目表
 * ================================================================ */
const MEMORY_ITEMS: {
  content: string;
  type: string;
  variant: StatusTagColor;
  updated: string;
}[] = [
  { content: "用户偏好使用简体中文回复", type: "偏好", variant: "blue", updated: "2026-06-22" },
  { content: "平台部署在中国（上海）地域", type: "事实", variant: "teal", updated: "2026-06-21" },
  { content: "每周一上午汇报项目进展", type: "任务", variant: "violet", updated: "2026-06-20" },
  { content: "运维联系人：张工（值班）", type: "实体", variant: "amber", updated: "2026-06-18" },
];

function MemoryManagementPage() {
  return (
    <>
      <AdminPageHeader
        title="记忆管理"
        description="版本对比表直接复用记忆管理页真实组件 ComparisonTable，企业级特性卡图标即 feature-card 槽位（原生 16×16）"
      />

      {/* 陪衬：记忆实例统计行 —— 统一用 owning 组件 NumberCard（SKILL §2.9）。
         图标取自真实「记忆管理」页 /assets/admin-memory-management/*.svg
         （已登记为 number-card 槽位资源），18×18 入 NumberCard.icon；
         数据自洽：实例总数 = 未开启 + Free 版 + Pro 版 = 32 + 76 + 20 = 128。 */}
      <div className="mb-5 grid grid-cols-2 gap-5 lg:grid-cols-4">
        <NumberCard
          icon={
            <SafeImg
              src="/assets/admin-memory-management/instance-total.svg"
              alt=""
              className="h-[18px] w-[18px] shrink-0"
            />
          }
          label="实例总数"
          value="128"
        />
        <NumberCard
          icon={
            <SafeImg
              src="/assets/admin-memory-management/instance-disabled.svg"
              alt=""
              className="h-[18px] w-[18px] shrink-0"
            />
          }
          label="未开启"
          value="32"
        />
        <NumberCard
          icon={
            <SafeImg
              src="/assets/admin-memory-management/instance-free.svg"
              alt=""
              className="h-[18px] w-[18px] shrink-0"
            />
          }
          label="Free 版"
          value="76"
        />
        <NumberCard
          icon={
            <SafeImg
              src="/assets/admin-memory-management/instance-pro.svg"
              alt=""
              className="h-[18px] w-[18px] shrink-0"
            />
          }
          label="Pro 版"
          value="20"
        />
      </div>

      {/* 主角 feature-card */}
      <SurfaceCard className="p-6">
        <PanelTitle as="p" className="mb-6">Free 版 vs Pro 版</PanelTitle>
        <ComparisonTable />
      </SurfaceCard>

      {/* 陪衬：记忆条目表 */}
      <div className="mt-5">
        <PanelTitle as="p" className="mb-3">最近记忆条目</PanelTitle>
        <SurfaceCard className="overflow-hidden">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: "100%", minWidth: 240 }}>记忆内容</TableHead>
                <TableHead style={{ width: 120, minWidth: 100 }}>类型</TableHead>
                <TableHead style={{ width: 140, minWidth: 120 }}>更新时间</TableHead>
                <TableHead style={{ width: 100, minWidth: 90 }}>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {MEMORY_ITEMS.map((m) => (
                <TableRow key={m.content}>
                  <TableCell>
                    <BodyMedium tone="primary">{m.content}</BodyMedium>
                  </TableCell>
                  <TableCell>
                    <StatusTag mode="soft" variant={m.variant}>{m.type}</StatusTag>
                  </TableCell>
                  <TableCell>
                    <MetaText as="span" tone="secondary">{m.updated}</MetaText>
                  </TableCell>
                  <TableCell>
                    <button className="inline-flex items-center gap-0.5 text-sm text-[#1447E6] hover:underline">
                      查看 <ChevronRight className="size-3.5" />
                    </button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      </div>
    </>
  );
}

/* ================================================================
 * 顶部页面切换 Tab + 外壳
 * ================================================================ */
const TABS = [
  { id: "basic", label: "基础信息", slot: "card-left-icon", Page: BasicInfoPage },
  { id: "tokens", label: "Tokens 监控", slot: "number-card", Page: TokensMonitorPage },
  { id: "channel", label: "通道配置", slot: "channel-icon", Page: ChannelConfigPage },
  { id: "memory", label: "记忆管理", slot: "feature-card", Page: MemoryManagementPage },
] as const;

export default function SkillMapAbDemo() {
  const [active, setActive] = useState<(typeof TABS)[number]["id"]>("basic");
  const current = TABS.find((t) => t.id === active)!;
  const ActivePage = current.Page;

  return (
    <AdminLayout
      hideNoticeBar
      previewNavItem={{
        label: "测试demo",
        href: "/preview/skill-map-ab",
        iconSrc: "/assets/admin-sidebar/audit-log.svg",
      }}
    >
      <div className="page-enter">
        {/* demo 说明条 */}
        <div className="mb-4 flex items-center justify-between gap-4 rounded-[4px] border border-[#dbe6ff] bg-blue-50/60 px-4 py-2.5">
          <p className="text-xs text-[#1447E6]">
            选图效果演示 · 复用真实管理端外壳（侧栏 = admin-sidebar，Logo = brand-logo），
            下方 Tab 切换 4 个真实管理页，各自承载一个业务槽位。
          </p>
          <Link
            href="/preview"
            className="inline-flex shrink-0 items-center gap-1.5 text-xs text-gray-500 hover:text-gray-900"
          >
            <Home className="h-3.5 w-3.5" /> 预览索引
          </Link>
        </div>

        {/* 页面切换 Tab（黑色下划线，与 ChannelConfig LineTabs 同款） */}
        <div className="mb-6 flex items-center gap-2 border-b border-[#dbe6ff]">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActive(tab.id)}
              className={`relative px-4 py-3 whitespace-nowrap transition-colors ${
                active === tab.id ? "border-b-2 border-[#0A0A0A] -mb-px" : ""
              }`}
            >
              <BodyMedium
                tone={active === tab.id ? "primary" : "muted"}
                className={
                  active !== tab.id
                    ? "hover:text-[var(--text-title)] transition-colors"
                    : ""
                }
              >
                {tab.label}
              </BodyMedium>
              <span className="ml-2 align-middle">
                <StatusTag mode="fill" variant={active === tab.id ? "blue" : "gray"}>
                  {tab.slot}
                </StatusTag>
              </span>
            </button>
          ))}
        </div>

        {/* 当前页面 */}
        <ActivePage />
      </div>
    </AdminLayout>
  );
}
