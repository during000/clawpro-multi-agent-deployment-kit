/**
 * PreviewIndex
 * --------------------------------------------------------------
 * 全站页面预览索引，方便设计/PM 一键跳转所有页面。
 * 访问路径：/preview
 */
import { useMemo, useState } from "react";
import { Link } from "wouter";
import { Search, ExternalLink, Home } from "lucide-react";

type RouteItem = {
  path: string;
  name: string;
  desc?: string;
  /** 是否需要 a 标签整页跳转（如非 wouter 路由的静态资源） */
  external?: boolean;
};

type RouteGroup = {
  key: string;
  title: string;
  color: string; // tailwind 色阶前缀，例如 "blue"
  items: RouteItem[];
};

const GROUPS: RouteGroup[] = [
  {
    key: "components",
    title: "组件走查 / Components",
    color: "rose",
    items: [
      { path: "/preview/table", name: "Table 表格组件", desc: "标准 / 紧凑 / white 变体 / 固定列横滚 全场景走查" },
      { path: "/preview/empty-state", name: "空状态 EmptyState" },
      { path: "/preview/empty-state-spec-verify", name: "空状态 Portable Spec 验证", desc: "基于 portable fallback 的独立验证" },
      { path: "/preview/toast", name: "Toast 消息提示", desc: "success / error / info / warning 全类型" },
      { path: "/preview/avatar", name: "Avatar 头像", desc: "4 档尺寸 + fallback" },
      { path: "/preview/tree", name: "Tree 树结构", desc: "展开/收起 / 选中 / 禁用 / 多层嵌套" },
      { path: "/preview/breadcrumb", name: "Breadcrumb 面包屑", desc: "2~4 级层级导航" },
      { path: "/preview/transfer", name: "Transfer 穿梭框", desc: "instant 模式 / 搜索 / 分页 / 禁用项" },
      { path: "/preview/alert", name: "Alert 提示条", desc: "info / operation-info / warning / product-news + AdminNoticeAlert" },
      { path: "/preview/button", name: "Button 按钮", desc: "spec §3 8 个 variant + 5 处偏差可视化 + Portable fallback" },
      { path: "/preview/date-picker", name: "DatePicker 日期选择", desc: "spec §3/§5/§10 · 5 状态 + Admin/Tenant 圆角分流 + 硬编码偏差审计" },
      { path: "/preview/skill-map-ab", name: "skill-map 选图效果", desc: "阶段 9 · 按槽位从真实候选选图，零裂图/零 emoji/零红线违规自证" },
      { path: "/preview/date-time-picker", name: "DateTimePicker 日期时间选择", desc: "spec §13 · 时分/时分秒(showSeconds) + 步长 + 草稿态 + Admin/Tenant 圆角分流" },
    ],
  },
  {
    key: "landing",
    title: "Landing / 入口",
    color: "violet",
    items: [
      { path: "/", name: "Landing 页", desc: "ClawPro 官网首页（点击「登录」直接弹出 SSO 弹窗）" },
    ],
  },
  {
    key: "tenant",
    title: "租户端（Tenant）",
    color: "blue",
    items: [
      { path: "/my-openclaw", name: "我的 OpenClaw", desc: "Agent 列表" },
      { path: "/openclaw/1", name: "OpenClaw 详情", desc: "示例 id=1" },
      { path: "/model-quota", name: "模型额度" },
      { path: "/skill-square", name: "技能广场" },
      { path: "/help-docs", name: "帮助文档" },
    ],
  },
  {
    key: "admin-base",
    title: "管理端 · 基础配置",
    color: "emerald",
    items: [
      { path: "/admin/basic-info", name: "基础信息" },
      { path: "/admin/platform-policy", name: "平台策略" },
      { path: "/admin/members", name: "成员管理" },
      { path: "/admin/model-config", name: "模型配置" },
      { path: "/admin/channel-config", name: "渠道配置" },
      { path: "/admin/skill-config", name: "技能配置" },
      { path: "/admin/agent-template", name: "Agent 模板" },
      { path: "/admin/agent-tool-library", name: "Agent 工具库" },
      { path: "/admin/skill-detail/1", name: "技能详情", desc: "示例 id=1" },
    ],
  },
  {
    key: "admin-resource",
    title: "管理端 · 资源管理",
    color: "amber",
    items: [
      { path: "/admin/image-management", name: "镜像管理" },
      { path: "/admin/security-group", name: "安全组管理" },
      { path: "/admin/cloud-dev", name: "云研发管理" },
      { path: "/admin/memory-management", name: "记忆管理" },
      { path: "/admin/file-management", name: "文件管理" },
    ],
  },
  {
    key: "admin-monitor",
    title: "管理端 · 监控与运维",
    color: "rose",
    items: [
      { path: "/admin/openclaw-monitor", name: "OpenClaw 监控" },
      { path: "/admin/agent-migration", name: "Agent 迁移" },
      { path: "/admin/tokens-monitor", name: "Tokens 监控" },
      { path: "/admin/security-management", name: "安全管理" },
      { path: "/admin/session-management", name: "会话管理" },
      { path: "/admin/session/1", name: "会话详情", desc: "示例 id=1" },
      { path: "/admin/ops-observation", name: "运营观测" },
      { path: "/admin/audit-log", name: "审计日志" },
      { path: "/admin/api-docs", name: "API 文档" },
    ],
  },
];

const COLOR_MAP: Record<string, { dot: string; chip: string; ring: string }> = {
  violet: { dot: "bg-violet-500", chip: "bg-violet-50 text-violet-700 border-violet-200", ring: "hover:border-violet-400" },
  blue: { dot: "bg-blue-500", chip: "bg-blue-50 text-blue-700 border-blue-200", ring: "hover:border-blue-400" },
  emerald: { dot: "bg-emerald-500", chip: "bg-emerald-50 text-emerald-700 border-emerald-200", ring: "hover:border-emerald-400" },
  amber: { dot: "bg-amber-500", chip: "bg-amber-50 text-amber-700 border-amber-200", ring: "hover:border-amber-400" },
  rose: { dot: "bg-rose-500", chip: "bg-rose-50 text-rose-700 border-rose-200", ring: "hover:border-rose-400" },
};

export default function PreviewIndex() {
  const [keyword, setKeyword] = useState("");

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return GROUPS;
    return GROUPS
      .map((g) => ({
        ...g,
        items: g.items.filter(
          (it) =>
            it.path.toLowerCase().includes(kw) ||
            it.name.toLowerCase().includes(kw) ||
            (it.desc?.toLowerCase().includes(kw) ?? false),
        ),
      }))
      .filter((g) => g.items.length > 0);
  }, [keyword]);

  const total = useMemo(() => GROUPS.reduce((s, g) => s + g.items.length, 0), []);
  const matched = useMemo(() => filtered.reduce((s, g) => s + g.items.length, 0), [filtered]);

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
      {/* Header */}
      <header className="sticky top-0 z-10 bg-white/80 backdrop-blur border-b border-slate-200">
        <div className="max-w-6xl mx-auto px-6 py-5">
          <div className="flex items-center justify-between gap-4 flex-wrap">
            <div>
              <h1 className="text-xl font-semibold text-slate-900 flex items-center gap-2">
                <span className="inline-block w-2 h-2 rounded-full bg-blue-500" />
                OpenClaw 企业版 · 页面预览索引
              </h1>
              <p className="text-sm text-slate-500 mt-1">
                共 {total} 个页面，当前匹配 {matched} 个。点击卡片直接跳转。
              </p>
            </div>
            <a
              href="/"
              className="inline-flex items-center gap-1.5 text-sm text-slate-600 hover:text-blue-600 transition-colors"
            >
              <Home className="w-4 h-4" />
              回到 Landing 首页
            </a>
          </div>

          <div className="mt-4 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索页面名称、路径或描述，例如：会话、admin、monitor"
              className="w-full pl-9 pr-4 py-2.5 text-sm rounded-lg border border-slate-200 bg-white outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition"
            />
          </div>
        </div>
      </header>

      {/* Groups */}
      <main className="max-w-6xl mx-auto px-6 py-8">
        {filtered.length === 0 && (
          <div className="text-center py-20 text-slate-400 text-sm">没有匹配的页面，换个关键词试试。</div>
        )}

        {filtered.map((group) => {
          const c = COLOR_MAP[group.color] ?? COLOR_MAP.blue;
          return (
            <section key={group.key} className="mb-10">
              <div className="flex items-center gap-2 mb-4">
                <span className={`inline-block w-1.5 h-5 rounded-full ${c.dot}`} />
                <h2 className="text-base font-semibold text-slate-800">{group.title}</h2>
                <span className={`inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full border ${c.chip}`}>
                  {group.items.length}
                </span>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {group.items.map((it) => {
                  const card = (
                    <div
                      className={`group relative h-full rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-md hover:-translate-y-0.5 ${c.ring}`}
                    >
                      <div className="flex items-start justify-between gap-2 mb-2">
                        <span className="text-[15px] font-semibold text-slate-900 leading-snug">
                          {it.name}
                        </span>
                        <ExternalLink className="w-3.5 h-3.5 text-slate-300 group-hover:text-blue-500 mt-1 shrink-0" />
                      </div>
                      <code className="block text-xs text-slate-500 font-mono break-all mb-2">
                        {it.path}
                      </code>
                      {it.desc && (
                        <p className="text-xs text-slate-500 leading-relaxed">{it.desc}</p>
                      )}
                    </div>
                  );

                  // 静态资源用 a 标签，React 路由用 wouter Link
                  return it.external ? (
                    <a key={it.path} href={it.path} className="block h-full no-underline">
                      {card}
                    </a>
                  ) : (
                    <Link key={it.path} href={it.path} className="block h-full no-underline">
                      {card}
                    </Link>
                  );
                })}
              </div>
            </section>
          );
        })}

        <footer className="mt-16 pt-6 border-t border-slate-200 text-xs text-slate-400 text-center">
          这是仅用于预览的页面索引，访问路径 <code className="font-mono text-slate-500">/preview</code>。
        </footer>
      </main>
    </div>
  );
}
