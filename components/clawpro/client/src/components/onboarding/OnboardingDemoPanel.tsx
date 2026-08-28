/**
 * OnboardingDemoPanel - 全局常驻的新手引导体验控制浮窗
 * 
 * 挂载在 App 层级，叠加在所有真实页面之上。
 * 通过右下角浮窗面板，可以开启/关闭不同引导组件，
 * 在真实的管理端/用户端页面上体验引导效果。
 */
import { useState, useCallback, useRef, useEffect, useLayoutEffect } from "react";
import { useLocation } from "wouter";
import {
  Eye,
  Minimize2,
  ChevronUp,
  ChevronDown,
  Copy,
  Check,
} from "lucide-react";
import { GuideGlobalModal } from "./GuideGlobalModal";
import { GuideModuleFloat } from "./GuideModuleFloat";
import { GuideAdminNotify } from "./GuideAdminNotify";
import type { AdminNotifyItem } from "./GuideAdminNotify";
import { GuidePointBubble } from "./GuidePointBubble";
import { GuideNavBubble } from "./GuideNavBubble";
import { GuideUpdateBar } from "./GuideUpdateBar";
import { GuideChangelogDrawer } from "./GuideChangelogDrawer";
import { ProductUpdatesDrawer } from "./ProductUpdatesDrawer";
import { GuideHighlightBubble } from "./GuideHighlightBubble";
import { FilterChipGroup } from "@/components/ui/filter-chip";
import type { GlobalModalSlide, GlobalModalVariant, ModuleFloatItem, ModuleFloatVariant, ChangelogVersion, HighlightRegion, PointBubbleContentVariant } from "./index";
import type { GlobalModalEndpoint } from "./GuideGlobalModal";
import {
  useActiveDemoPanel,
  setActiveDemoPanel,
} from "../demoFloatingPanel";


// ─── 场景化演示数据 ─────────────────────────────────────────────

// 单条内容演示数据（用户端弹窗样式 —— 对应 OnboardingGuide）
const DEMO_SINGLE_SLIDE: GlobalModalSlide[] = [
  {
    titleLeft: "",
    titleRight: "更清晰美观",
    desc: "全新界面更美观，观感更舒适清爽，布局更清晰简约，快来体验新版设计吧！",
    videoSrc: "/landing-assets/onboarding/tenant-guide.mp4",
  },
];

// 多条内容演示数据 — 管控端样式（AdminOnboardingGuide：标题带"XX端 | 标题"竖线分割）
const DEMO_CAROUSEL_SLIDES_ADMIN: GlobalModalSlide[] = [
  {
    titleLeft: "管控端",
    titleRight: "更简洁高效",
    desc: "在保持原有操作习惯的基础上，界面更简约清爽，信息屏效比更高，颜值质感双提升！",
    videoSrc: "/landing-assets/onboarding/admin-guide.mp4",
  },
  {
    titleLeft: "用户端",
    titleRight: "更清晰美观",
    desc: "全新界面更美观，观感更舒适清爽，布局更清晰简约，快来体验新版设计吧！",
  },
];

// 多条内容演示数据 — 用户端样式（标题不展示"XX端"和竖线，更多 slides + mock 图片）
const DEMO_CAROUSEL_SLIDES_TENANT: GlobalModalSlide[] = [
  {
    titleRight: "更清晰美观",
    desc: "全新界面更美观，观感更舒适清爽，布局更清晰简约，快来体验新版设计吧！",
    videoSrc: "/landing-assets/onboarding/tenant-guide.mp4",
  },
  {
    titleRight: "对话视图上线",
    desc: "全新对话模式，让您与 Agent 的交互更加自然流畅，支持多轮对话和上下文记忆。",
  },
  {
    titleRight: "技能广场全新登场",
    desc: "在技能广场浏览和安装丰富的技能扩展，让您的 Agent 能力更强大、更智能。",
  },
  {
    titleRight: "个性化主题定制",
    desc: "支持自定义主题颜色与布局偏好，打造专属工作空间，提升日常使用愉悦感。",
  },
  {
    titleRight: "智能搜索增强",
    desc: "全新语义搜索能力，输入自然语言即可精准定位 Agent、技能和历史记录。",
  },
];

const DEMO_FLOAT_SINGLE: ModuleFloatItem[] = [
  {
    subtitle: "消息通知",
    title: "Agent 新版本上线",
    description: "您已完成基础配置，可以开始在浏览器中和Agent进行对话了，在浏览器中和Agent进行对话了。",
    image: "/assets/用户端浮窗示例图.png",
    actionText: "立即体验",
    actionHref: "/openclaw/agent-demo",
  },
];

const DEMO_FLOAT_MULTI: ModuleFloatItem[] = [
  {
    subtitle: "消息通知",
    title: "Agent 新版本上线",
    description: "您已完成基础配置，可以开始在浏览器中和Agent进行对话了，在浏览器中和Agent进行对话了。",
    image: "/assets/用户端浮窗示例图.png",
    actionText: "立即体验",
    actionHref: "/my-openclaw",
  },
  {
    subtitle: "功能更新",
    title: "技能广场全新改版",
    description: "技能广场已全面升级，支持按场景分类浏览、一键安装和版本管理，快来探索更多实用技能吧。",
    image: "/assets/onboarding-demo.png",
    actionText: "去看看",
    actionHref: "/admin/skill-config",
  },
  {
    subtitle: "系统公告",
    title: "成员分组权限优化",
    description: "新增多级分组继承能力，支持跨部门协作授权，管理员可按需灵活分配 Agent 访问权限。",
    actionText: "了解详情",
    actionHref: "/admin/members",
  },
];

// 产品动态非阻断弹窗演示数据 —— 按端 / 单条·多条拆分（4 种卡片来源）
// 经 buildProductUpdateNotices 去重聚合后，每端最多产出一张卡片，整体最多两张。
const NOTIFY_ADMIN_ITEMS: AdminNotifyItem[] = [
  {
    title: "管控端 | 新增 MCP 凭据托管功能",
    desc: "支持快捷配置 Hermes 模型、通道和 Skill，并提供角色设定和初始技能包等能力",
    btnText: "立即体验",
    endpoint: "admin",
    relatedIds: ["u-mcp"],
  },
  {
    title: "管控端 | Tokens 统计优化",
    desc: "Tokens 统计支持按时间维度设置，记忆管理功能同步上线。",
    btnText: "立即体验",
    endpoint: "admin",
    relatedIds: ["u-tokens"],
  },
  {
    title: "管控端 | 企业技能库支持版本一键升级",
    desc: "支持对已下发技能进行版本升级，可批量筛选落后实例并一键同步。",
    btnText: "立即体验",
    endpoint: "admin",
    relatedIds: ["4"],
  },
];

const NOTIFY_TENANT_ITEMS: AdminNotifyItem[] = [
  {
    title: "用户端 | Hermes 支持对话视图",
    desc: "用户端 Hermes 类型的 Agent 支持开启对话视图功能。",
    btnText: "立即体验",
    endpoint: "tenant",
    relatedIds: ["u-dialog"],
  },
  {
    title: "用户端 | 技能广场全新登场",
    desc: "在技能广场浏览和安装丰富的技能扩展，让 Agent 能力更强大。",
    btnText: "立即体验",
    endpoint: "tenant",
    relatedIds: ["u-square"],
  },
];

/** 端类型 → 取单条 / 多条原始数据 */
function pickNotifyItems(
  endpoint: "admin" | "tenant",
  type: "single" | "multi"
): AdminNotifyItem[] {
  const pool = endpoint === "admin" ? NOTIFY_ADMIN_ITEMS : NOTIFY_TENANT_ITEMS;
  return type === "single" ? pool.slice(0, 1) : pool;
}

const DEMO_CHANGELOG: ChangelogVersion[] = [
  {
    version: "v3.2.0",
    date: "2026-06-09",
    entries: [
      { id: "1", title: "技能包管理合并", description: "公共技能库和企业技能库整合为技能管理中心", tag: "结构", date: "2026-06-09", href: "/admin/skill-config" },
      { id: "2", title: "成员分组支持多组", description: "一个用户可以属于多个分组，分组策略支持继承", tag: "逻辑", date: "2026-06-09", href: "/admin/members" },
      { id: "3", title: "新增「分发状态」列", description: "企业插件库列表新增分发状态字段", tag: "元素", date: "2026-06-09" },
      { id: "4", title: "安全模块上线", description: "新增 AI Agent 安全防护功能", tag: "系统", date: "2026-06-09", href: "/admin/security-management" },
      { id: "5", title: "用户端对话功能告知", description: "C 端新增对话模式，管理员需了解用户行为变化", tag: "跨端", date: "2026-06-09" },
    ],
  },
  {
    version: "v3.1.0",
    date: "2026-05-20",
    entries: [
      { id: "6", title: "「插件管理」更名", description: "「插件管理」更名为「企业插件库」，功能不变", tag: "元素", date: "2026-05-20" },
      { id: "7", title: "镜像管理新增版本推送", description: "支持一键推送最新镜像版本到所有实例", tag: "元素", date: "2026-05-20", href: "/admin/image-management" },
    ],
  },
];

// 高亮区域（根据当前页面动态生成）
//
// 优先使用 `selector` 让步骤指引气泡识别「当前真实页面」里的元素进行标注，
// 呼吸灯/矩形会按元素真实位置与尺寸贴合，而非凭空浮一个随机大小/位置的框。
// `top/left/width/height` 仅作为选择器未命中时的兜底坐标。
function getHighlightRegions(pathname: string): HighlightRegion[] {
  // 基础信息配置页（/admin/basic-info）：标注左侧前 3 个真实「步骤卡片」
  if (pathname.includes("/admin/basic-info")) {
    const stepCard = (n: number) =>
      `[data-guide="basic-steps"] > [data-surface="card"]:nth-child(${n})`;
    return [
      { id: "r1", selector: stepCard(1), title: "设置平台名称与品牌", description: "在这里配置展示在用户端的网站名称和 Logo。", bubblePlacement: "bottom", padding: 8 },
      { id: "r2", selector: stepCard(2), title: "配置用户默认配额", description: "设置新用户创建时自动应用的 Agent 数量上限和每日 Tokens 上限。", bubblePlacement: "bottom", padding: 8 },
      { id: "r3", selector: stepCard(3), title: "导入企业用户", description: "前往用户管理页添加企业用户，添加后即可使用平台。", bubblePlacement: "bottom", padding: 8 },
    ];
  }
  // 用户管理（/admin/members）：用 selector 贴合真实工具栏 / 视图切换，禁止写死坐标导致随机浮层
  if (pathname.includes("/admin/members")) {
    return [
      { id: "r1", selector: '[data-guide="member-toolbar"]', title: "新增批量操作工具栏", description: "支持批量导入、批量变更分组、批量分发配置等操作。", bubblePlacement: "bottom", padding: 8 },
      { id: "r2", selector: '[data-guide="member-view-segment"]', title: "分组筛选增强", description: "支持按多分组交集/并集筛选，快速定位目标成员。", bubblePlacement: "bottom", padding: 8 },
    ];
  }
  // 模型配置（/admin/model-config）：贴合页面唯一的模型列表表格（Table 渲染真实 <table>）
  if (pathname.includes("/admin/model-config")) {
    return [
      { id: "r1", selector: "table", title: "模型列表新增状态列", description: "实时显示每个模型的可用状态和调用量。", bubblePlacement: "bottom", padding: 8 },
    ];
  }
  // 技能管理（/admin/skill）：贴合 Tab 内容区第一个真实业务卡片
  if (pathname.includes("/admin/skill")) {
    return [
      { id: "r1", selector: '[data-surface="card"]', title: "技能管理整合", description: "原「公共技能库」和「企业技能库」已合并到这里统一管理。", bubblePlacement: "bottom", padding: 8 },
    ];
  }
  // 平台策略（/admin/platform-policy）：用页面内带 id 的配额卡片容器进行标注
  if (pathname.includes("/admin/platform-policy")) {
    return [
      { id: "r1", selector: "#plan4-claw", title: "配额设置", description: "管理单用户 Agent 数量上限、Tokens 上限等配额策略，支持按分组差异化配置。", bubblePlacement: "bottom", padding: 8 },
      { id: "r2", selector: "#plan4-configModel", title: "功能权限开关", description: "控制用户端功能开关，如是否允许自主配置模型等。", bubblePlacement: "bottom", padding: 8 },
    ];
  }
  // 用户端「我的 Agent」页：与管控端一致，用 selector 贴合真实页面元素进行标注，
  // 不再使用写死坐标凭空浮层。气泡方向由 GuideHighlightBubble 自动匹配并 clamp 进视口。
  if (pathname.includes("/my-openclaw") || pathname.includes("/openclaw")) {
    return [
      { id: "r1", selector: '[data-guide="tenant-agent-grid"] > *:first-child', title: "Agent 卡片", description: "每个 Agent 以卡片形式展示，可查看状态、快速进入对话，或对其进行管理操作。", bubblePlacement: "bottom", padding: 8 },
      { id: "r2", selector: '[data-guide="tenant-view-switch"]', title: "新增「管理 / 对话」视图切换", description: "可在卡片管理视图与沉浸式对话视图之间一键切换。", bubblePlacement: "bottom", padding: 8 },
      { id: "r3", selector: '[data-guide="tenant-create-agent"]', title: "快速创建 Agent", description: "点击「创建 Agent」，为你的 Agent 取一个名字即可开始。", bubblePlacement: "bottom", padding: 8 },
    ];
  }
  // 默认：按优先级尝试多种通用 selector 识别当前页面真实元素进行标注。
  // 不提供兜底坐标——selector 全部未命中时 GuideHighlightBubble 不渲染，避免随机浮层。
  return [
    {
      id: "r1",
      selector: '[data-surface="card"], section[id], [id^="plan4-"], table, .page-enter > :nth-child(2)',
      title: "页面更新区域",
      description: "此区域有功能变更，请查看具体变化。",
      bubblePlacement: "bottom",
      padding: 8,
    },
  ];
}

// ─── 引导组件类型定义 ─────────────────────────────────────────

interface GuideComponentOption {
  id: string;
  name: string;
  description: string;
  endpoint: "both" | "admin" | "tenant";
  sceneTags: string[];
  /** 对应版本更新感知场景清单中的详细使用场景 */
  sceneDetail: string;
}

const GUIDE_COMPONENTS: GuideComponentOption[] = [
  {
    id: "global-modal",
    name: "全局弹窗",
    description: "影响面极大的更新",
    endpoint: "both",
    sceneTags: ["结构层", "系统层"],
    sceneDetail: "适用场景：\n• 1.1 新增子页面 — 产品新增完整子页面，用户从未见过\n• 1.4 页面整合/拆分 — 多个页面合并或拆分\n• 4.1 账号体系变更 — 登录方式、认证逻辑变化\n• 4.3 数据合规/隐私变更 — 需用户明确知情或同意\n\n感知要点：全屏阻断，强制用户阅读并确认，适用于不可忽视的重大变更。",
  },
  {
    id: "module-float",
    name: "非阻断浮窗",
    description: "模块级更新",
    endpoint: "both",
    sceneTags: ["结构层", "元素层"],
    sceneDetail: "适用场景：\n• 1.2 页面重新排布 — 功能区域位置/顺序/分组变化，用户需重建空间映射\n• 2.6 细节优化叠加 — 多个微小优化叠加，单独不值得通知但汇总有感知价值\n\n感知要点：不阻断操作，可最小化，让用户在使用中自然了解变化。适合\"长得不一样了\"但功能没变的情况。",
  },
  {
    id: "point-bubble",
    name: "单UI提示气泡",
    description: "直接在UI附近展示的点对点引导",
    endpoint: "both",
    sceneTags: ["元素层"],
    sceneDetail: "分类体系：\n1/ 单UI提示（直接在UI附近展示）\n  • 1.1 纯文本类型 — 支持四个方向，可含列表\n  • 1.2 纯文本+按钮 — 支持副标题\n  • 1.3 纯文本+图片 — 带配图区域\n  • 1.4 重点推送通知 — 蓝色背景强调\n\n适用场景：2.1~2.3 新增按钮/表格列/筛选项",
  },
  {
    id: "highlight-bubble",
    name: "步骤指引气泡",
    description: "带呼吸灯或区域标注的步骤性指引",
    endpoint: "both",
    sceneTags: ["结构层", "元素层"],
    sceneDetail: "分类体系：\n2/ 步骤指引（带呼吸灯步骤性指引型弹窗）\n  • 深色指引 — 跨功能页面引导\n  • 浅色指引 — 页面内局部引导\n\n适用场景：\n• 1.2 页面重新排布 — 标注新布局各区域位置\n• 2.1 新增按钮/操作入口 — 高亮新增的可交互元素\n• 2.2 新增表格列 — 高亮新列并解释含义\n• 2.3 新增筛选/排序选项 — 高亮筛选栏变化\n• 2.4 名称/文案变更 — 标注\"改了名字\"的元素\n\n感知要点：Spotlight 遮罩高亮目标区域 + 步骤气泡解释。支持多区域串联步骤导航，引导用户逐一了解页面变化，建立新的认知映射。",
  },
];

// ─── 典型场景组合（对应版本更新感知场景清单） ──────────────────────

interface SceneCombination {
  id: string;
  name: string;
  layer: string;
  description: string;
  endpoint: "both" | "admin" | "tenant";
  /** 该场景触发的原子组件 id 组合 */
  components: string[];
  /** 适用场景说明 */
  sceneUsage?: string[];
  /** 描述中需要强调的片段（红色加粗） */
  descHighlight?: string;
}

const SCENE_COMBINATIONS: SceneCombination[] = [
  {
    id: "s-light",
    name: "最轻量提示",
    layer: "",
    description: "在UI附近直接展示提示气泡，打开页面后默认出现，有配图建议联系设计师审核样式",
    descHighlight: "有配图建议联系设计师审核样式",
    endpoint: "both",
    components: ["point-bubble"],
    sceneUsage: ["新增按钮/操作入口（次级）", "新增表格列/字段（次级）", "新增筛选/排序选项（次级）", "名称/文案变更", "页面入口新增（一级导航）", "页面入口禁用说明", "细节优化（入口处说明）"],
  },
  {
    id: "s-daily",
    name: "日常更新提示",
    layer: "",
    description: "日常功能更新使用，管控端展示产品动态卡片，用户端展示非阻断浮窗，通常点击后跳转对应页面衔接气泡展示，有配图建议联系设计师审核样式",
    descHighlight: "有配图建议联系设计师审核样式",
    endpoint: "both",
    components: ["module-float"],
    sceneUsage: ["新增子页面", "页面重新排布", "功能位置变动", "页面整合/拆分", "页面入口新增/下线", "新增按钮/操作入口（重要）", "新增表格列/字段（重要）", "新增筛选/排序选项（重要）", "细节优化（路径调整/问题修复）", "底层逻辑变更", "规则/策略变更", "计费/配额变更", "权限体系变更", "C端↔管控端跨端联动"],
  },
  {
    id: "s-heavy",
    name: "最重量提示",
    layer: "",
    description: "仅在系统性重大变更或需要用户明确知情同意情况下使用，新增请务必联系设计师审核样式",
    descHighlight: "请务必联系设计师审核样式",
    endpoint: "both",
    components: ["global-modal"],
    sceneUsage: ["账号体系变更", "数据合规/隐私政策变更", "权限体系变更（需确认）"],
  },
];

// ─── 当前激活组件/变体的标准引用名 ─────────────────────────────

function getActiveComponentRef(
  activeGuides: Record<string, boolean>,
  opts: {
    modalVariant: GlobalModalVariant;
    floatVariant: ModuleFloatVariant;
    bubbleContentVariant: PointBubbleContentVariant;
    bubbleSubVariant: string;
    highlightHotspot: "circle" | "rect";
    highlightWithList: boolean;
    isAdmin: boolean;
    notifyCardCount: 1 | 2;
    notifyAdminType: "single" | "multi";
    notifyTenantType: "single" | "multi";
    notifySingleEndpoint: "admin" | "tenant";
  }
): string[] {
  const parts: string[] = [];

  if (activeGuides["global-modal"]) {
    parts.push(`GuideGlobalModal / ${opts.modalVariant === "single" ? "单条内容" : "多条内容"}`);
  }
  if (activeGuides["module-float"]) {
    if (opts.isAdmin) {
      const cardLabel = opts.notifyCardCount === 1
        ? `${opts.notifySingleEndpoint === "admin" ? "管控端" : "用户端"}-${(opts.notifySingleEndpoint === "admin" ? opts.notifyAdminType : opts.notifyTenantType) === "single" ? "单条" : "多条"}`
        : `管控端${opts.notifyAdminType === "single" ? "单条" : "多条"}+用户端${opts.notifyTenantType === "single" ? "单条" : "多条"}`;
      parts.push(`GuideAdminNotify / ${cardLabel}`);
    } else {
      parts.push(`GuideModuleFloat / ${opts.floatVariant === "single" ? "单条内容" : "多条内容"}`);
    }
  }
  if (activeGuides["nav-bubble"]) {
    parts.push("GuideNavBubble");
  }
  if (activeGuides["point-bubble"]) {
    const subItems = POINT_BUBBLE_SUB_VARIANTS[opts.bubbleContentVariant];
    const subLabel = subItems.find((s) => s.id === opts.bubbleSubVariant)?.label || opts.bubbleSubVariant;
    parts.push(`GuidePointBubble / ${opts.bubbleContentVariant} / ${subLabel}`);
  }
  if (activeGuides["update-bar"]) {
    parts.push("GuideUpdateBar");
  }
  if (activeGuides["changelog-drawer"]) {
    parts.push("GuideChangelogDrawer");
  }
  if (activeGuides["highlight-bubble"]) {
    const hotLabel = opts.highlightHotspot === "circle" ? "呼吸灯" : "矩形标注";
    const listLabel = opts.highlightWithList ? "+有序列表" : "";
    parts.push(`GuideHighlightBubble / ${hotLabel}${listLabel}`);
  }

  return parts;
}

// ─── 主组件 ──────────────────────────────────────────────────

export function OnboardingDemoPanel() {
  const [location, navigate] = useLocation();
  const [expanded, setExpanded] = useState(false);
  // 与另外两个 Demo 浮层协调：任一浮层展开时，其余两个的折叠 header 隐藏
  const activeDemoPanel = useActiveDemoPanel();
  const hideTrigger = activeDemoPanel !== null && activeDemoPanel !== "onboarding";
  useEffect(() => {
    if (expanded) setActiveDemoPanel("onboarding");
    else if (activeDemoPanel === "onboarding") setActiveDemoPanel(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded]);
  useEffect(() => {
    if (activeDemoPanel !== null && activeDemoPanel !== "onboarding" && expanded) {
      setExpanded(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeDemoPanel]);
  const [activeTab, setActiveTab] = useState<"atoms" | "scenes">("atoms");
  const [highlightIndex, setHighlightIndex] = useState(0);
  const [modalVariant, setModalVariant] = useState<GlobalModalVariant>("single");
  // 步骤指引气泡（GuideHighlightBubble）：热点类型（呼吸灯 / 矩形标注）+ 是否附加有序列表
  const [highlightHotspot, setHighlightHotspot] = useState<"circle" | "rect">("circle");
  const [highlightWithList, setHighlightWithList] = useState(false);
  // 单UI提示气泡（GuidePointBubble）：内容变体 + 子样式，完全对齐预览页分类体系
  const [bubbleContentVariant, setBubbleContentVariant] = useState<PointBubbleContentVariant>("text-only");
  const [bubbleSubVariant, setBubbleSubVariant] = useState<string>("normal");
  const [floatVariant, setFloatVariant] = useState<ModuleFloatVariant>("single");
  // 管控端非阻断浮窗：支持选择卡片数量（1 / 2 张）与各端卡片类型（单条 / 多条）
  const [notifyCardCount, setNotifyCardCount] = useState<1 | 2>(1);
  const [notifySingleEndpoint, setNotifySingleEndpoint] = useState<"admin" | "tenant">("admin");
  const [notifyAdminType, setNotifyAdminType] = useState<"single" | "multi">("single");
  const [notifyTenantType, setNotifyTenantType] = useState<"single" | "multi">("single");
  // 产品动态抽屉：点击卡片「查看详情/立即体验」打开，并高亮卡片提到的动态
  const [productDrawerOpen, setProductDrawerOpen] = useState(false);
  const [highlightUpdateIds, setHighlightUpdateIds] = useState<string[]>([]);

  const handleNotifyAction = useCallback((item: AdminNotifyItem) => {
    setHighlightUpdateIds(item.relatedIds ?? []);
    setProductDrawerOpen(true);
  }, []);

  // 根据选择构造原始 items（交由 buildProductUpdateNotices 去重/聚合/截断为 ≤2 张）
  const notifyItems: AdminNotifyItem[] =
    notifyCardCount === 1
      ? pickNotifyItems(notifySingleEndpoint, notifySingleEndpoint === "admin" ? notifyAdminType : notifyTenantType)
      : [
          ...pickNotifyItems("admin", notifyAdminType),
          ...pickNotifyItems("tenant", notifyTenantType),
        ];


  // 「日常更新提示」场景：浮窗关闭后，若用户导航到 agent 详情页，自动展示 push-notice 气泡
  const [pendingFloatBubble, setPendingFloatBubble] = useState(false);

  // 各引导组件的开关状态
  const [activeGuides, setActiveGuides] = useState<Record<string, boolean>>({
    "global-modal": false,
    "module-float": false,
    "nav-bubble": false,
    "point-bubble": false,
    "update-bar": false,
    "changelog-drawer": false,
    "highlight-bubble": false,
  });

  const toggleGuide = useCallback((id: string) => {
    setActiveGuides((prev) => ({ ...prev, [id]: !prev[id] }));
    // 重置步骤
    if (id === "highlight-bubble") setHighlightIndex(0);
  }, []);

  const closeGuide = useCallback((id: string) => {
    setActiveGuides((prev) => ({ ...prev, [id]: false }));
  }, []);

  // 当用户从浮窗点击「立即体验」导航到 agent 详情页后，自动展示 push-notice 气泡
  useEffect(() => {
    if (pendingFloatBubble && location.includes("/openclaw/")) {
      // 延迟一帧让页面渲染完成
      const timer = setTimeout(() => {
        setBubbleContentVariant("push-notice");
        setBubbleSubVariant("text");
        setActiveGuides((prev) => ({ ...prev, "point-bubble": true }));
        setPendingFloatBubble(false);
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [pendingFloatBubble, location]);

  // 判断当前端
  const isAdmin = location.includes("/admin");
  const endpoint = isAdmin ? "admin" : "tenant";
  const highlightRegions = getHighlightRegions(location);

  // 固定在右下角：与 BillingStatusToggle(bottom-4) / AdminModeFloatingToggle(bottom:60px)
  // 纵向叠放，本面板放最上层（bottom:104px）。不再支持拖拽——三个 Demo 浮层现在同为一列。
  const containerRef = useRef<HTMLDivElement>(null);

  return (
    <>
      {/* ═══ 引导组件渲染层（叠加在真实页面上） ═══ */}

      {/* ① 全局弹窗（严格对齐 Figma 设计稿） */}
      <GuideGlobalModal
        open={activeGuides["global-modal"]}
        onClose={() => closeGuide("global-modal")}
        variant={modalVariant}
        slides={
          modalVariant === "single"
            ? DEMO_SINGLE_SLIDE
            : isAdmin
            ? DEMO_CAROUSEL_SLIDES_ADMIN
            : DEMO_CAROUSEL_SLIDES_TENANT
        }
        confirmText="立即体验"
        endpoint={endpoint as GlobalModalEndpoint}
      />

      {/* ② 非阻断浮窗：管控端用 AdminNotifyCard（产品动态卡片），用户端用 GuideModuleFloat */}
      {isAdmin ? (
        <GuideAdminNotify
          open={activeGuides["module-float"]}
          onClose={() => closeGuide("module-float")}
          variant="stacked"
          items={notifyItems}
          onAction={handleNotifyAction}
        />
      ) : (
        <GuideModuleFloat
          open={activeGuides["module-float"]}
          onClose={() => closeGuide("module-float")}
          onConfirm={() => {
            setPendingFloatBubble(true);
            navigate("/openclaw/agent-demo");
          }}
          subtitle="消息通知"
          title="Agent 新版本上线"
          items={floatVariant === "single" ? DEMO_FLOAT_SINGLE : DEMO_FLOAT_MULTI}
          variant={floatVariant}
        />
      )}

      {/* ③ 导航预览气泡 */}
      {activeGuides["nav-bubble"] && (
        <div className="fixed top-[200px] left-[70px] z-[9980]">
          <GuideNavBubble
            open={true}
            onClose={() => closeGuide("nav-bubble")}
            title="公共技能包"
            description="全新页面入口。在这里可以浏览和管理公共技能包，将技能批量组合后分发给用户。"
            href="/admin/skill-config"
            actionText="去看看"
          />
        </div>
      )}

      {/* ④ 单UI提示气泡（识别当前页面真实元素作为锚点，贴靠展示，跟随滚动，无呼吸灯） */}
      {activeGuides["point-bubble"] && (
        <PointBubblePortal
          onClose={() => closeGuide("point-bubble")}
          contentVariant={bubbleContentVariant}
          subVariant={bubbleSubVariant}
          endpoint={endpoint}
          pathname={location}
        />
      )}

      {/* ⑤ 强提醒公告条（Portal 插入导航下方，撑开页面） */}
      <GuideUpdateBar
        open={activeGuides["update-bar"]}
        message="您的账户存在欠费风险，当前服务可能受限。请尽快完成续费以恢复全部功能。"
        version="v3.2.0"
        onDetail={() => toggleGuide("changelog-drawer")}
        detailText="查看详情"
      />

      {/* ⑥ 更新记录抽屉 */}
      <GuideChangelogDrawer
        open={activeGuides["changelog-drawer"]}
        onClose={() => closeGuide("changelog-drawer")}
        versions={DEMO_CHANGELOG}
      />

      {/* ⑥' 产品动态抽屉：由产品动态卡片「查看详情/立即体验」触发，并高亮卡片提到的动态 */}
      <ProductUpdatesDrawer
        open={productDrawerOpen}
        onClose={() => setProductDrawerOpen(false)}
        highlightIds={highlightUpdateIds}
        defaultRecentOnly
      />

      {/* ⑦ 高亮+气泡 */}
      <GuideHighlightBubble
        open={activeGuides["highlight-bubble"]}
        onClose={() => closeGuide("highlight-bubble")}
        regions={highlightRegions}
        currentIndex={highlightIndex}
        onIndexChange={setHighlightIndex}
        endpoint={endpoint}
        hotspotShape={highlightHotspot}
        showList={highlightWithList}
      />

      {/* ═══ 控制浮窗 ═══ */}
      <div
        ref={containerRef}
        className="fixed right-4 z-[99999]"
        style={{ bottom: 104 }}
      >
        {/* 相对定位容器：仅承载折叠触发按钮；展开面板已改为 fixed 到视口右下角，脱离本容器。 */}
        <div className="relative">
          {/* 展开面板：固定浮在视口右下角，层级最高，打开时会覆盖三个折叠 header 那一列（预期） */}
          {expanded && (
            <div className="fixed right-4 bottom-4 z-[100000] w-[340px] bg-white rounded-xl border border-gray-200 shadow-2xl overflow-hidden animate-in slide-in-from-bottom-2 duration-200">
            {/* 面板头 */}
            <div
              className={`px-4 py-3 select-none ${isAdmin ? "bg-gray-900" : "bg-[#1a3a6b]"}`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-white">引导体系体验面板</span>
                  <span className="text-[10px] text-white/70 px-1.5 py-0.5 rounded bg-white/20">
                    {isAdmin ? "管理端" : "用户端"}
                  </span>
                </div>
                <button
                  onClick={(e) => { e.stopPropagation(); setExpanded(false); }}
                  onPointerDown={(e) => e.stopPropagation()}
                  className="w-6 h-6 rounded flex items-center justify-center hover:bg-white/20 transition-colors"
                  title="折叠面板"
                >
                  <Minimize2 className="w-3.5 h-3.5 text-white/70" />
                </button>
              </div>
              <p className="text-[11px] text-white/60 mt-1">
                仅供产研人员测试体验用
              </p>
              <p className="text-[10px] text-white/40 mt-0.5 font-mono">
                预览页：/preview/onboarding-guide
              </p>
            </div>

            {/* Tab 切换 */}
            <div className="flex border-b border-gray-200">
              <button
                onClick={() => setActiveTab("atoms")}
                className={`flex-1 py-2 text-xs font-medium text-center transition-colors ${
                  activeTab === "atoms"
                    ? "text-black border-b-2 border-black"
                    : "text-gray-500 hover:text-gray-700"
                }`}
              >
                原子组件预览 ({GUIDE_COMPONENTS.filter((c) => c.endpoint === "both" || c.endpoint === endpoint).length})
              </button>
              <button
                onClick={() => {
                  setActiveTab("scenes");
                  setActiveGuides((prev) => {
                    const next = { ...prev };
                    Object.keys(next).forEach((k) => { next[k] = false; });
                    return next;
                  });
                  setHighlightIndex(0);
                }}
                className={`flex-1 py-2 text-xs font-medium text-center transition-colors ${
                  activeTab === "scenes"
                    ? "text-black border-b-2 border-black"
                    : "text-gray-500 hover:text-gray-700"
                }`}
              >
                使用场景说明 ({SCENE_COMBINATIONS.filter((s) => s.endpoint === "both" || s.endpoint === endpoint).length})
              </button>
            </div>



            {/* ─── Tab 内容区 ─── */}
            {activeTab === "atoms" ? (
              /* 原子组件预览 */
              <div className="px-3 py-3 space-y-1 max-h-[320px] overflow-y-auto">
                {GUIDE_COMPONENTS.filter((comp) => comp.endpoint === "both" || comp.endpoint === endpoint).map((comp) => {
                  const isActive = activeGuides[comp.id];

                  return (
                    <div
                      key={comp.id}
                      className={`rounded-lg transition-all ${
                        isActive
                          ? "bg-gray-50 border border-gray-200"
                          : "hover:bg-gray-50 border border-transparent"
                      }`}
                    >
                      <div className="flex items-center gap-3 px-3 py-2.5">
                        <div className="flex-1 min-w-0">
                          <span className="text-xs font-medium text-gray-800">{comp.name}</span>
                          <span className="text-[11px] text-gray-400 block mt-0.5">{comp.description}</span>
                        </div>

                        <button
                          onClick={() => toggleGuide(comp.id)}
                          className={`w-8 h-[18px] rounded-full flex items-center px-0.5 shrink-0 transition-colors cursor-pointer ${
                            isActive ? "bg-blue-500" : "bg-gray-200"
                          }`}
                        >
                          <div className={`w-3.5 h-3.5 rounded-full bg-white shadow-sm transition-transform ${
                            isActive ? "translate-x-3.5" : "translate-x-0"
                          }`} />
                        </button>
                      </div>

                      {/* 全局弹窗变体选择器 */}
                      {comp.id === "global-modal" && isActive && (
                        <div className="flex items-center gap-1.5 px-3 pb-2.5 pt-2 border-t border-gray-200">
                          <FilterChipGroup
                            size="sm"
                            items={[
                              { id: "single", label: "单条内容" },
                              { id: "carousel", label: "多条内容" },
                            ]}
                            value={modalVariant}
                            onChange={(id) => setModalVariant(id as GlobalModalVariant)}
                          />
                        </div>
                      )}
                      {/* 非阻断浮窗变体 */}
                      {comp.id === "module-float" && isActive && (
                        <div className="px-3 pb-2.5 pt-2 border-t border-gray-200 space-y-1.5">
                          {isAdmin ? (
                            /* 管控端产品动态：卡片数量 + 各端卡片类型 */
                            <div className="space-y-1.5">
                              {/* 卡片数量 */}
                              <div className="flex items-center gap-1.5 flex-wrap">
                                <span className="text-[10px] text-gray-400 w-12 shrink-0">卡片数量</span>
                                <FilterChipGroup
                                  size="sm"
                                  items={[
                                    { id: "1", label: "1 张" },
                                    { id: "2", label: "2 张" },
                                  ]}
                                  value={String(notifyCardCount)}
                                  onChange={(id) => setNotifyCardCount(Number(id) as 1 | 2)}
                                />
                              </div>

                              {/* 单张时选择展示端 */}
                              {notifyCardCount === 1 && (
                                <div className="flex items-center gap-1.5 flex-wrap">
                                  <span className="text-[10px] text-gray-400 w-12 shrink-0">展示端</span>
                                  <FilterChipGroup
                                    size="sm"
                                    items={[
                                      { id: "admin", label: "管控端" },
                                      { id: "tenant", label: "用户端" },
                                    ]}
                                    value={notifySingleEndpoint}
                                    onChange={(id) => setNotifySingleEndpoint(id as "admin" | "tenant")}
                                  />
                                </div>
                              )}

                              {/* 管控端卡片类型 */}
                              {(notifyCardCount === 2 || notifySingleEndpoint === "admin") && (
                                <div className="flex items-center gap-1.5 flex-wrap">
                                  <span className="text-[10px] text-gray-400 w-12 shrink-0">管控端</span>
                                  <FilterChipGroup
                                    size="sm"
                                    items={[
                                      { id: "single", label: "单条" },
                                      { id: "multi", label: "多条" },
                                    ]}
                                    value={notifyAdminType}
                                    onChange={(id) => setNotifyAdminType(id as "single" | "multi")}
                                  />
                                </div>
                              )}

                              {/* 用户端卡片类型 */}
                              {(notifyCardCount === 2 || notifySingleEndpoint === "tenant") && (
                                <div className="flex items-center gap-1.5 flex-wrap">
                                  <span className="text-[10px] text-gray-400 w-12 shrink-0">用户端</span>
                                  <FilterChipGroup
                                    size="sm"
                                    items={[
                                      { id: "single", label: "单条" },
                                      { id: "multi", label: "多条" },
                                    ]}
                                    value={notifyTenantType}
                                    onChange={(id) => setNotifyTenantType(id as "single" | "multi")}
                                  />
                                </div>
                              )}
                            </div>
                          ) : (
                            /* 用户端：单条内容 / 多条内容 */
                            <FilterChipGroup
                              size="sm"
                              items={[
                                { id: "single", label: "单条内容" },
                                { id: "multi", label: "多条内容" },
                              ]}
                              value={floatVariant}
                              onChange={(id) => setFloatVariant(id as ModuleFloatVariant)}
                            />
                          )}
                        </div>
                      )}
                      {/* 单UI提示气泡变体 — 完全对齐预览页（/preview/onboarding-guide）分类体系 */}
                      {comp.id === "point-bubble" && isActive && (
                        <div className="px-3 pb-2.5 pt-2 border-t border-gray-200 space-y-1.5">
                          {/* 内容类型（1.1 ~ 1.4） */}
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <span className="text-[10px] text-gray-400 w-12 shrink-0">类型</span>
                            <FilterChipGroup
                              size="sm"
                              items={[
                                { id: "text-only", label: "1.1 纯文本" },
                                { id: "text-button", label: "1.2 文本+按钮" },
                                { id: "text-image", label: "1.3 文本+图片" },
                                { id: "push-notice", label: "1.4 推送通知" },
                              ]}
                              value={bubbleContentVariant}
                              onChange={(id) => {
                                const value = id as PointBubbleContentVariant;
                                setBubbleContentVariant(value);
                                setBubbleSubVariant(POINT_BUBBLE_SUB_VARIANTS[value][0].id);
                              }}
                            />
                          </div>
                          {/* 子样式（跟随预览页每个类型下的细分变体） */}
                          {POINT_BUBBLE_SUB_VARIANTS[bubbleContentVariant].length > 1 && (
                            <div className="flex items-center gap-1.5 flex-wrap">
                              <span className="text-[10px] text-gray-400 w-12 shrink-0">样式</span>
                              <FilterChipGroup
                                size="sm"
                                items={POINT_BUBBLE_SUB_VARIANTS[bubbleContentVariant]}
                                value={bubbleSubVariant}
                                onChange={(id) => setBubbleSubVariant(id)}
                              />
                            </div>
                          )}
                        </div>
                      )}

                      {/* 步骤指引气泡变体：热点类型（呼吸灯 / 矩形标注）+ 是否有序列表 */}
                      {comp.id === "highlight-bubble" && isActive && (
                        <div className="px-3 pb-2.5 pt-2 border-t border-gray-200 space-y-1.5">
                          {/* 热点类型 */}
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <span className="text-[10px] text-gray-400 w-12 shrink-0">热点</span>
                            <FilterChipGroup
                              size="sm"
                              items={[
                                { id: "circle", label: "呼吸灯" },
                                { id: "rect", label: "矩形标注" },
                              ]}
                              value={highlightHotspot}
                              onChange={(id) => setHighlightHotspot(id as "circle" | "rect")}
                            />
                          </div>
                          {/* 有序列表 */}
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <span className="text-[10px] text-gray-400 w-12 shrink-0">有序列表</span>
                            <FilterChipGroup
                              size="sm"
                              items={[
                                { id: "off", label: "不添加" },
                                { id: "on", label: "添加" },
                              ]}
                              value={highlightWithList ? "on" : "off"}
                              onChange={(id) => setHighlightWithList(id === "on")}
                            />
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            ) : (
              /* 典型场景预览 */
              <div className="px-3 py-3 space-y-2 max-h-[320px] overflow-y-auto">
                {SCENE_COMBINATIONS.filter((scene) => scene.endpoint === "both" || scene.endpoint === endpoint).map((scene) => {
                  const isApplicable = true;
                  // 判断该场景是否已激活（所有对应组件都为 true）
                  const isSceneActive = scene.components.every((c) => activeGuides[c]);

                  const toggleScene = () => {
                    if (!isApplicable) return;
                    const next = { ...activeGuides };
                    if (isSceneActive) {
                      // 关闭：将该场景的组件全部关闭
                      scene.components.forEach((c) => { next[c] = false; });
                    } else {
                      // 开启：将该场景的组件全部开启
                      scene.components.forEach((c) => { next[c] = true; });
                      setHighlightIndex(0);
                    }
                    setActiveGuides(next);
                  };

                  return (
                    <div
                      key={scene.id}
                      className={`rounded-lg border px-3 py-2.5 transition-all ${
                        !isApplicable
                          ? "opacity-40 border-gray-100"
                          : isSceneActive
                          ? "border-blue-200 bg-blue-50/40"
                          : "border-gray-200 hover:border-gray-300"
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-1.5">
                            <span className="text-xs font-medium text-gray-800">{scene.name}</span>
                          </div>
                          <p className="text-[11px] text-gray-400 mt-0.5">
                            {scene.descHighlight
                              ? <>{scene.description.split(scene.descHighlight)[0]}<span className="text-gray-800 font-semibold">{scene.descHighlight}</span>{scene.description.split(scene.descHighlight)[1] ?? ""}</>
                              : scene.description}
                          </p>
                        </div>

                        {/* 开关 */}
                        <button
                          onClick={toggleScene}
                          disabled={!isApplicable}
                          className={`w-8 h-[18px] rounded-full flex items-center px-0.5 shrink-0 transition-colors ${
                            !isApplicable ? "cursor-not-allowed" : "cursor-pointer"
                          } ${isSceneActive ? "bg-blue-500" : "bg-gray-200"}`}
                        >
                          <div className={`w-3.5 h-3.5 rounded-full bg-white shadow-sm transition-transform ${
                            isSceneActive ? "translate-x-3.5" : "translate-x-0"
                          }`} />
                        </button>
                      </div>
                      {scene.sceneUsage && scene.sceneUsage.length > 0 && (
                        <div className="mt-2 pt-2 border-t border-gray-100">
                          <span className="text-[10px] text-gray-800 font-medium block mb-1.5">适用场景：</span>
                          <div className="flex flex-wrap gap-1">
                            {scene.sceneUsage.map((tag, idx) => (
                              <span key={idx} className="text-[10px] text-gray-800 bg-gray-100 rounded px-1.5 py-0.5">{tag}</span>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {/* 底部：当前组件引用名（可折行 + 逐个复制） */}
            <ActiveComponentRefBar
              refs={getActiveComponentRef(activeGuides, {
                modalVariant,
                floatVariant,
                bubbleContentVariant,
                bubbleSubVariant,
                highlightHotspot,
                highlightWithList,
                isAdmin,
                notifyCardCount,
                notifyAdminType,
                notifyTenantType,
                notifySingleEndpoint,
              })}
            />
          </div>
        )}

          {/* 触发按钮（点击切换展开/折叠）
              当其它 Demo 浮层展开时，本触发按钮隐藏，避免视觉打架。 */}
          {!hideTrigger && (
            <button
              className="w-[180px] flex items-center gap-2 px-3 py-2 bg-white rounded-xl border border-gray-200 hover:bg-gray-50 transition-colors select-none"
              style={{ boxShadow: "0 4px 20px rgba(0,0,0,0.12)" }}
              title="用户引导模拟（点击展开/折叠）"
              onClick={() => setExpanded((v) => !v)}
            >
              <Eye className="w-4 h-4 text-gray-500 shrink-0" />
              <span className="text-xs font-medium text-gray-700">用户引导模拟</span>
              {expanded ? (
                <ChevronDown className="w-3 h-3 text-gray-400 ml-auto shrink-0" />
              ) : (
                <ChevronUp className="w-3 h-3 text-gray-400 ml-auto shrink-0" />
              )}
            </button>
          )}
        </div>
      </div>
    </>
  );
}

// ─── 底部组件引用名栏（支持折行 + 逐个复制） ──────────────────────

function ActiveComponentRefBar({ refs }: { refs: string[] }) {
  const [copiedIdx, setCopiedIdx] = useState<number | null>(null);

  const handleCopy = useCallback((text: string, idx: number) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedIdx(idx);
      setTimeout(() => setCopiedIdx((prev) => (prev === idx ? null : prev)), 1500);
    });
  }, []);

  return (
    <div className="px-4 py-3 border-t border-gray-100">
      <span className="text-[10px] text-gray-400 block mb-1.5">当前组件：</span>
      {refs.length === 0 ? (
        <span className="text-[10px] text-gray-400 italic">未激活</span>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {refs.map((ref, idx) => (
            <span
              key={idx}
              className="inline-flex items-center gap-1 bg-gray-100 rounded px-1.5 py-0.5 group"
            >
              <code className="text-[10px] font-mono text-gray-600 leading-tight">{ref}</code>
              <button
                onClick={() => handleCopy(ref, idx)}
                className="w-3.5 h-3.5 flex items-center justify-center rounded hover:bg-gray-200 transition-colors opacity-50 group-hover:opacity-100"
                title="复制组件名"
              >
                {copiedIdx === idx ? (
                  <Check className="w-2.5 h-2.5 text-green-500" />
                ) : (
                  <Copy className="w-2.5 h-2.5 text-gray-500" />
                )}
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── 单UI提示气泡演示数据（与预览页 /preview/onboarding-guide 保持完全一致） ──

/** 子样式定义：每个内容类型下的细分变体，分类与预览页一一对应 */
const POINT_BUBBLE_SUB_VARIANTS: Record<PointBubbleContentVariant, { id: string; label: string }[]> = {
  // 1.1 纯文本类型：常规 / 可配标签 / 支持有序文本
  "text-only": [
    { id: "normal", label: "常规" },
    { id: "tag", label: "带标签" },
    { id: "list", label: "有序列表" },
  ],
  // 1.2 纯文本 + 按钮：单按钮 / 双按钮 / 有序+按钮
  "text-button": [
    { id: "single", label: "单按钮" },
    { id: "double", label: "双按钮" },
    { id: "list", label: "有序+按钮" },
  ],
  // 1.3 纯文本 + 图片：支持配图
  "text-image": [
    { id: "image", label: "支持配图" },
  ],
  // 1.4 重点推送通知：纯文本 / 文本+按钮 / 图片+按钮
  "push-notice": [
    { id: "text", label: "纯文本" },
    { id: "button", label: "文本+按钮" },
    { id: "image", label: "图片+按钮" },
  ],
};

// 与预览页 OnboardingGuidePreview 中的演示常量保持一致
const PB_TITLE = "标题文本介绍";
const PB_DESC = "快速管理已开通的企业应用，也可以在这里创建自建应用实现统一登录。";
const PB_LIST = [
  "管控端新增「Agent 类型管理」页面支持镜像更新推送",
  "管控端 Agent 列表工具栏新增镜像更新提醒铃铛",
  "管控端平台策略新增「允许员工自助更新 Agent 版本」开关",
];

/** 气泡演示 props（仅内容相关字段，定位 / 端 / 热点由外层统一注入） */
interface BubbleDemoProps {
  title: string;
  description: string;
  tag?: string;
  listItems?: string[];
  imageCaption?: string;
  noticeImage?: string;
  actionText?: string;
  secondaryActionText?: string;
}

/** (内容类型, 子样式) → 与预览页逐字一致的气泡 props */
function buildBubbleDemo(
  contentVariant: PointBubbleContentVariant,
  subVariant: string
): BubbleDemoProps {
  switch (contentVariant) {
    // ─── 1.1 纯文本类型 ───
    case "text-only":
      if (subVariant === "tag") {
        return { title: PB_TITLE, description: PB_DESC, tag: "还有 7 天上线" };
      }
      if (subVariant === "list") {
        return { title: PB_TITLE, description: PB_DESC, listItems: PB_LIST };
      }
      return { title: PB_TITLE, description: PB_DESC }; // 常规版

    // ─── 1.2 纯文本 + 按钮 ───
    case "text-button":
      if (subVariant === "double") {
        return { title: PB_TITLE, description: PB_DESC, actionText: "我知道了", secondaryActionText: "了解更多" };
      }
      if (subVariant === "list") {
        return { title: PB_TITLE, description: PB_DESC, listItems: PB_LIST, actionText: "我知道了", secondaryActionText: "了解更多" };
      }
      return { title: PB_TITLE, description: PB_DESC, actionText: "我知道了" }; // 单按钮

    // ─── 1.3 纯文本 + 图片 ───
    case "text-image":
      return { title: "运维管理功能升级", description: PB_DESC, imageCaption: "2026-04-20" };

    // ─── 1.4 重点推送通知 ───
    case "push-notice": {
      const base = { title: "版本升级，重磅来袭！", description: "当前版本为 V1.0升级版本后可一件接入微信" };
      if (subVariant === "button") {
        return { ...base, secondaryActionText: "知道了", actionText: "立即升级" };
      }
      if (subVariant === "image") {
        return { ...base, noticeImage: "", secondaryActionText: "知道了", actionText: "立即升级" };
      }
      return base; // 纯文本
    }

    default:
      return { title: PB_TITLE, description: PB_DESC };
  }
}

// ─── 单UI提示气泡定位组件（页面元素识别 + 贴靠，无呼吸灯） ──────────────
//
// 核心：按当前页面路由解析一组「候选选择器」，按序识别真实存在的页面元素，
// 用 getBoundingClientRect 实时测量其位置/尺寸，把气泡贴靠到该元素旁，
// 并自动翻转方向 + clamp 进视口，确保不被截断、不凭空随机浮动。
// 命中不到任何元素时不渲染（避免随机浮层）。

type BubblePlacement = "top" | "bottom" | "left" | "right";

interface BubbleRect {
  top: number;
  left: number;
  width: number;
  height: number;
}

/**
 * 按当前页面返回单UI提示气泡的锚点候选选择器（按序匹配第一个命中的真实元素）
 * 与 getHighlightRegions 的页面识别口径保持一致。
 */
function getPointBubbleAnchors(pathname: string): {
  selectors: string[];
  placement: BubblePlacement;
} {
  // 管控端：mock 先标注左侧「Agent 类型」导航栏按钮（气泡朝右贴靠），
  // 命中不到（如侧边栏收起）时回退到第一个业务卡片。
  // 注意：需放在用户端判断之前，避免 /admin/openclaw-monitor 等含 "openclaw" 的管控端路由被误判。
  if (pathname.includes("/admin")) {
    return {
      selectors: [
        'a[href="/admin/agent-types"]',
        '[data-surface="card"]',
      ],
      placement: "right",
    };
  }
  // 用户端「我的 Agent」：优先贴「对话视图」视图切换，回退到创建入口 / hero
  if (pathname.includes("/my-openclaw") || pathname.includes("/openclaw")) {
    return {
      selectors: [
        '[role="tablist"][aria-label="视图切换"] [role="tab"]:nth-child(2)',
        '[data-guide="tenant-view-switch"]',
        '[data-guide="tenant-create-agent"]',
        '[data-guide="tenant-hero"]',
        '[data-surface="card"]',
      ],
      placement: "bottom",
    };
  }
  // 默认：识别当前页面主区域的第一个业务卡片
  return { selectors: ['[data-surface="card"]'], placement: "bottom" };
}

function PointBubblePortal({
  onClose,
  contentVariant = "text-only",
  subVariant = "normal",
  endpoint = "tenant",
  pathname = "",
}: {
  onClose: () => void;
  contentVariant?: PointBubbleContentVariant;
  subVariant?: string;
  endpoint?: "admin" | "tenant";
  pathname?: string;
}) {
  const { selectors, placement: preferred } = getPointBubbleAnchors(pathname);
  const selectorKey = selectors.join("|");

  // 目标元素实时位置（视口坐标系）
  const [rect, setRect] = useState<BubbleRect | null>(null);
  // 气泡真实尺寸 —— 用于自动翻转方向 + 视口 clamp
  const bubbleRef = useRef<HTMLDivElement>(null);
  const [bubbleSize, setBubbleSize] = useState<{ w: number; h: number }>({ w: 280, h: 0 });

  // 识别页面元素并持续测量（候选选择器按序命中），跟随滚动 / 布局变化
  useEffect(() => {
    const same = (a: BubbleRect | null, b: BubbleRect | null) =>
      !!a && !!b && a.top === b.top && a.left === b.left && a.width === b.width && a.height === b.height;

    let raf = 0;
    const measure = () => {
      let next: BubbleRect | null = null;
      for (const s of selectors) {
        const el = document.querySelector(s) as HTMLElement | null;
        if (el) {
          const r = el.getBoundingClientRect();
          if (r.width > 0 || r.height > 0) {
            next = { top: r.top, left: r.left, width: r.width, height: r.height };
            break;
          }
        }
      }
      setRect((prev) => (same(prev, next) ? prev : next));
      raf = requestAnimationFrame(measure);
    };
    measure();
    return () => cancelAnimationFrame(raf);
  }, [selectorKey]);

  // 首次把命中的目标滚动到可视区域中央
  useEffect(() => {
    for (const s of selectors) {
      const el = document.querySelector(s);
      if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "center" });
        break;
      }
    }
    // 仅在挂载时执行一次
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 测量气泡真实尺寸
  useLayoutEffect(() => {
    const el = bubbleRef.current;
    if (!el) return;
    const m = () => {
      const r = el.getBoundingClientRect();
      setBubbleSize((p) => (p.w === r.width && p.h === r.height ? p : { w: r.width, h: r.height }));
    };
    m();
    const ro = new ResizeObserver(m);
    ro.observe(el);
    return () => ro.disconnect();
  }, [rect, contentVariant, subVariant]);

  if (!rect) return null;

  // ── 自动匹配方向 + 视口 clamp ──────────────────────────────
  const GAP = 12; // 气泡与目标间距
  const MARGIN = 12; // 与视口边缘最小安全距离
  const vw = typeof window !== "undefined" ? window.innerWidth : 0;
  const vh = typeof window !== "undefined" ? window.innerHeight : 0;
  const bw = bubbleSize.w;
  const bh = bubbleSize.h;
  const cx = rect.left + rect.width / 2;
  const cy = rect.top + rect.height / 2;

  const space: Record<BubblePlacement, number> = {
    top: rect.top,
    bottom: vh - (rect.top + rect.height),
    left: rect.left,
    right: vw - (rect.left + rect.width),
  };
  const opposite: Record<BubblePlacement, BubblePlacement> = {
    top: "bottom",
    bottom: "top",
    left: "right",
    right: "left",
  };
  const need = (p: BubblePlacement) =>
    p === "top" || p === "bottom" ? bh + GAP + MARGIN : bw + GAP + MARGIN;

  let activePlacement: BubblePlacement = preferred;
  if (space[activePlacement] < need(activePlacement)) {
    if (space[opposite[activePlacement]] >= need(activePlacement)) {
      activePlacement = opposite[activePlacement];
    } else {
      activePlacement = (["bottom", "top", "right", "left"] as BubblePlacement[]).reduce(
        (best, cur) => (space[cur] > space[best] ? cur : best),
        activePlacement
      );
    }
  }

  let left: number;
  let top: number;
  switch (activePlacement) {
    case "top":
      left = cx - bw / 2;
      top = rect.top - GAP - bh;
      break;
    case "bottom":
      left = cx - bw / 2;
      top = rect.top + rect.height + GAP;
      break;
    case "left":
      left = rect.left - GAP - bw;
      top = cy - bh / 2;
      break;
    case "right":
    default:
      left = rect.left + rect.width + GAP;
      top = cy - bh / 2;
      break;
  }
  if (bw > 0) left = Math.max(MARGIN, Math.min(left, vw - bw - MARGIN));
  if (bh > 0) top = Math.max(MARGIN, Math.min(top, vh - bh - MARGIN));

  const demo = buildBubbleDemo(contentVariant, subVariant);

  return (
    <div className="fixed inset-0 z-[9985] pointer-events-none">
      <div
        ref={bubbleRef}
        className="absolute animate-in fade-in duration-200 pointer-events-auto"
        style={{ left, top, opacity: bh > 0 ? 1 : 0 }}
      >
        <GuidePointBubble
          open
          onClose={onClose}
          variant={contentVariant === "push-notice" ? "dark" : "light"}
          contentVariant={contentVariant}
          title={demo.title}
          description={demo.description}
          tag={demo.tag}
          listItems={demo.listItems}
          imageCaption={demo.imageCaption}
          noticeImage={demo.noticeImage}
          actionText={demo.actionText}
          secondaryActionText={demo.secondaryActionText}
          showSteps={false}
          placement={activePlacement}
          showHotspot={false}
          endpoint={endpoint}
        />
      </div>
    </div>
  );
}
