/**
 * 统计每个 UI 组件在业务代码中的实际应用情况，并生成「组件 → 真实应用页面」映射。
 * 输出：
 *   1. 控制台打印 moduleCount / instanceCount 汇总
 *   2. 写入 client/src/pages/design-system/component-usage.generated.json
 *      格式：{ [componentId]: { moduleCount, instanceCount, pages: ApplicationPage[] } }
 *
 * 用法：node scripts/count-component-usage.mjs
 */
import { readFileSync, readdirSync, statSync, writeFileSync, mkdirSync } from "node:fs";
import { join, relative, dirname } from "node:path";

const ROOT = process.cwd();
const SRC = join(ROOT, "client/src");
const SELF = "pages/DesignSystemComponents.tsx";
const OUT = join(SRC, "pages/design-system/component-usage.generated.json");

const COMPONENTS = {
  typography: ["BodyText", "CardTitle", "SectionTitle", "PanelTitle", "MetaText", "StatNumber", "TenantPageTitle", "BodyMedium", "CodeText"],
  "surface-card": ["SurfaceCard"], "surface-inner": ["SurfaceInner"], "surface-config": ["SurfaceConfig"],
  "surface-overlay": ["SurfaceOverlay"], "tenant-card": ["TenantCard"], button: ["Button"], "button-group": ["ButtonGroup"],
  input: ["Input"], "input-group": ["InputGroup"], textarea: ["Textarea"], select: ["Select"], "date-picker": ["DatePicker"],
  checkbox: ["Checkbox"], field: ["Field"], "radio-group": ["RadioGroup"], "radio-card": ["RadioCard"], switch: ["Switch"],
  "filter-chip": ["FilterChip"], alert: ["Alert"], dialog: ["Dialog"], "alert-dialog": ["AlertDialog"], drawer: ["Drawer"],
  tooltip: ["Tooltip"], popover: ["Popover"], "info-popover": ["InfoPopover"], progress: ["Progress"], spinner: ["Spinner"],
  table: ["Table"], pagination: ["Pagination"], kbd: ["Kbd"], badge: ["Badge"], "status-tag": ["StatusTag"], empty: ["Empty"],
  stepper: ["Stepper"], segment: ["Segment"], "segmented-tabs": ["SegmentedTabs"], tabs: ["Tabs", "TabsList"],
  "tenant-section": ["TenantSection"], topnav: ["TopNav"], "admin-page-header": ["AdminPageHeader"], "admin-sidebar": ["AdminSidebar"],
  toast: ["toast("], avatar: ["Avatar"], tree: ["Collapsible", "Tree"], breadcrumb: ["Breadcrumb"], transfer: ["Transfer"],
};

// 路由页面映射：文件 base 名 / 目录前缀 → { path, name, platform }
const PAGE_ROUTES = [
  // tenant
  { match: "pages/tenant/MyOpenClaw", path: "/my-openclaw", name: "我的 Agent", platform: "Tenant 用户端" },
  { match: "pages/tenant/OpenClawDetailGuide", path: "/openclaw/1", name: "OpenClaw 详情", platform: "Tenant 用户端" },
  { match: "pages/tenant/OpenClawDetail", path: "/openclaw/1", name: "OpenClaw 详情", platform: "Tenant 用户端" },
  { match: "pages/tenant/ModelQuota", path: "/model-quota", name: "模型额度", platform: "Tenant 用户端" },
  { match: "pages/tenant/SkillSquare", path: "/skill-square", name: "技能广场", platform: "Tenant 用户端" },
  { match: "pages/tenant/HelpDocs", path: "/help-docs", name: "帮助文档", platform: "Tenant 用户端" },
  { match: "pages/tenant/AgentChat", path: "/preview/agent-chat", name: "会话", platform: "Tenant 用户端" },
  { match: "pages/tenant/", path: "/my-openclaw", name: "我的 Agent", platform: "Tenant 用户端" },
  // admin 顶层页面
  { match: "pages/admin/BasicInfo", path: "/admin/basic-info", name: "基础信息", platform: "Admin 管控端" },
  { match: "pages/admin/standard/StandardBasicInfo", path: "/admin/basic-info", name: "基础信息", platform: "Admin 管控端" },
  { match: "pages/admin/PlatformPolicy", path: "/admin/platform-policy", name: "平台策略", platform: "Admin 管控端" },
  { match: "pages/admin/MemberManagement", path: "/admin/members", name: "成员管理", platform: "Admin 管控端" },
  { match: "pages/admin/ModelConfig", path: "/admin/model-config", name: "模型配置", platform: "Admin 管控端" },
  { match: "pages/admin/ChannelConfig", path: "/admin/channel-config", name: "通道配置", platform: "Admin 管控端" },
  { match: "pages/admin/SkillConfig", path: "/admin/skill-config", name: "技能配置", platform: "Admin 管控端" },
  { match: "pages/admin/ResourceManagement", path: "/admin/agent-template", name: "Agent 模板", platform: "Admin 管控端" },
  { match: "pages/admin/ImageManagement", path: "/admin/image-management", name: "镜像管理", platform: "Admin 管控端" },
  { match: "pages/admin/ServerManagement", path: "/admin/image-management", name: "镜像管理", platform: "Admin 管控端" },
  { match: "pages/admin/SecurityGroupManagement", path: "/admin/security-group", name: "安全组管理", platform: "Admin 管控端" },
  { match: "pages/admin/CloudDevManagement", path: "/admin/cloud-dev", name: "云开发管理", platform: "Admin 管控端" },
  { match: "pages/admin/OpenClawMonitor", path: "/admin/openclaw-monitor", name: "Agent 监控", platform: "Admin 管控端" },
  { match: "pages/admin/agentOps/", path: "/admin/agent-commands", name: "Agent 命令", platform: "Admin 管控端" },
  { match: "pages/admin/AgentMigration", path: "/admin/agent-migration", name: "Agent 迁移", platform: "Admin 管控端" },
  { match: "pages/admin/TokensMonitor", path: "/admin/tokens-monitor", name: "Tokens 监控", platform: "Admin 管控端" },
  { match: "pages/admin/SessionManagement", path: "/admin/session-management", name: "会话管理", platform: "Admin 管控端" },
  { match: "pages/admin/SessionDetail", path: "/admin/session-management", name: "会话管理", platform: "Admin 管控端" },
  { match: "pages/admin/OpsObservation", path: "/admin/ops-observation", name: "运维观测", platform: "Admin 管控端" },
  { match: "pages/admin/AuditLog", path: "/admin/audit-log", name: "操作记录", platform: "Admin 管控端" },
  { match: "pages/admin/MemoryManagement", path: "/admin/memory-management", name: "记忆管理", platform: "Admin 管控端" },
  { match: "pages/admin/FileManagement", path: "/admin/file-management", name: "文件管理", platform: "Admin 管控端" },
  { match: "pages/admin/SkillDetailPage", path: "/admin/skill-detail/1", name: "技能详情", platform: "Admin 管控端" },
  { match: "pages/admin/SkillLibrary/", path: "/admin/agent-tool-library", name: "Agent 工具库", platform: "Admin 管控端" },
  { match: "pages/admin/AgentToolLibrary", path: "/admin/agent-tool-library", name: "Agent 工具库", platform: "Admin 管控端" },
  { match: "pages/admin/ApiDocs", path: "/admin/api-docs", name: "API 文档", platform: "Admin 管控端" },
  { match: "pages/admin/Security/", path: "/admin/security-management", name: "安全管理", platform: "Admin 管控端" },
  { match: "pages/admin/SecurityManagement", path: "/admin/security-management", name: "安全管理", platform: "Admin 管控端" },
  { match: "pages/admin/SkillRolesTab", path: "/admin/skill-config", name: "技能配置", platform: "Admin 管控端" },
  { match: "pages/admin/DocManagement", path: "/admin/file-management", name: "文件管理", platform: "Admin 管控端" },
  { match: "pages/admin/AuthSourceImportDialog", path: "/admin/members", name: "成员管理", platform: "Admin 管控端" },
  { match: "pages/admin/VersionManagement/", path: "/admin/agent-tool-library", name: "Agent 工具库", platform: "Admin 管控端" },
  { match: "pages/admin/BatchUpdateNotice", path: "/admin/openclaw-monitor", name: "Agent 监控", platform: "Admin 管控端" },
  // 业务子组件归并到所属页面
  { match: "components/policy/", path: "/admin/platform-policy", name: "平台策略", platform: "Admin 管控端" },
  { match: "components/Scope", path: "/admin/platform-policy", name: "平台策略", platform: "Admin 管控端" },
  { match: "components/Memory", path: "/admin/memory-management", name: "记忆管理", platform: "Admin 管控端" },
  { match: "components/DisableMemory", path: "/admin/memory-management", name: "记忆管理", platform: "Admin 管控端" },
  { match: "components/EnableMemory", path: "/admin/memory-management", name: "记忆管理", platform: "Admin 管控端" },
  { match: "components/SecurityScanCard", path: "/admin/security-management", name: "安全管理", platform: "Admin 管控端" },
  { match: "components/admin/", path: "/admin/members", name: "成员管理", platform: "Admin 管控端" },
  { match: "components/agent/", path: "/my-openclaw", name: "我的 Agent", platform: "Tenant 用户端" },
  { match: "components/tenant/", path: "/my-openclaw", name: "我的 Agent", platform: "Tenant 用户端" },
  { match: "components/topnav/", path: "/my-openclaw", name: "我的 Agent", platform: "Tenant 用户端" },
  { match: "components/OpenClawCombobox", path: "/admin/security-group", name: "安全组管理", platform: "Admin 管控端" },
];

// 排除：重复副本（带 " 2"）、demo / 布局 / 入口文件
const EXCLUDE = [
  / 2\.(tsx|ts)$/,
  /^pages\/ComponentPreview\./, /^pages\/Home\./, /^pages\/NotFound\./, /^pages\/SsoLoginDemo\./,
  /^App\.tsx$/, /^components\/AdminLayout\./, /^components\/TenantLayout\./,
  /^components\/SsoLoginDialog\./, /^components\/ErrorBoundary\./,
];

function routeOf(rel) {
  for (const r of PAGE_ROUTES) {
    if (rel.startsWith(r.match)) return r;
  }
  return null;
}

function walk(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    const st = statSync(full);
    if (st.isDirectory()) out.push(...walk(full));
    else if (/\.(tsx|ts)$/.test(name)) out.push(full);
  }
  return out;
}

const allFiles = walk(SRC).filter((f) => {
  const rel = relative(SRC, f).replace(/\\/g, "/");
  if (rel === SELF || rel.startsWith("components/ui/") || rel.startsWith("pages/preview/")) return false;
  if (EXCLUDE.some((re) => re.test(rel))) return false;
  return true;
});
const fileContents = allFiles.map((f) => ({ rel: relative(SRC, f).replace(/\\/g, "/"), text: readFileSync(f, "utf8") }));

const out = {};
const summary = [];
const unmatchedAgg = {}; // rel -> 总命中次数（有组件使用但没匹配路由）
for (const [id, tags] of Object.entries(COMPONENTS)) {
  let fileCount = 0, instanceCount = 0;
  const pageAgg = {}; // path -> { name, platform, count }
  for (const { rel, text } of fileContents) {
    let count = 0;
    for (const tag of tags) {
      const isFn = tag.endsWith("(");
      const re = isFn ? new RegExp(tag.replace("(", "\\("), "g") : new RegExp(`<${tag}[\\s/>]`, "g");
      const m = text.match(re);
      if (m) count += m.length;
    }
    if (count > 0) {
      fileCount += 1;
      instanceCount += count;
      const route = routeOf(rel);
      if (route) {
        if (!pageAgg[route.path]) pageAgg[route.path] = { name: route.name, path: route.path, platform: route.platform, count: 0 };
        pageAgg[route.path].count += count;
      } else {
        unmatchedAgg[rel] = (unmatchedAgg[rel] || 0) + count;
      }
    }
  }
  const allPages = Object.values(pageAgg).sort((a, b) => b.count - a.count);
  const pages = allPages.slice(0, 5).map((p, i) => ({ name: p.name, path: p.path, platform: p.platform, priority: i === 0 ? "高" : i < 3 ? "中" : "补充", usage: `实际使用 ${p.count} 处` }));
  // moduleCount = 真实使用该组件的文件/模块数；pages = 聚合到路由的 top5 页面
  out[id] = { moduleCount: fileCount, pageCount: allPages.length, instanceCount, pages };
  summary.push({ id, fileCount, pageCount: allPages.length, instanceCount });
}

mkdirSync(dirname(OUT), { recursive: true });
writeFileSync(OUT, JSON.stringify(out, null, 2), "utf8");
summary.sort((a, b) => b.instanceCount - a.instanceCount);
console.log("=== 组件统计（pageCount=路由页面数, fileCount=文件数, instanceCount=使用次数）===");
console.log(JSON.stringify(summary, null, 2));

const unmatched = Object.entries(unmatchedAgg).sort((a, b) => b[1] - a[1]).slice(0, 40);
console.log("\n=== 未匹配到路由的文件（按命中次数 top 40，需补 PAGE_ROUTES）===");
for (const [rel, c] of unmatched) console.log(`${c}\t${rel}`);
console.log("\nwritten:", relative(ROOT, OUT));

