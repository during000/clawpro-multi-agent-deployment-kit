/**
 * OnboardingGuidePreview - 新手引导体系演示页面
 * 路径: /preview/onboarding-guide
 *
 * 按「用户端 / 管控端」两个 Tab 分类组织，
 * 每个组件一张卡片：直接内嵌真实组件原型 + 样式约束 + 适用场景。
 *
 * 内嵌技术：用 InlineStage 容器把组件的 fixed 定位中和为 absolute，
 * 并按需缩放，保证与真实弹出效果样式 100% 一致。
 */
import { useState, useCallback, useEffect } from "react";
import {
  GuideGlobalModal,
  GuideModuleFloat,
  GuideNavBubble,
  GuidePointBubble,
  GuideHighlightBubble,
  AdminNotifyCard,
  AdminNotifyStack,
  GuideNewTag,
} from "@/components/onboarding";
import type { GlobalModalSlide, ModuleFloatItem, ChangelogVersion, HighlightRegion, AdminNotifyItem } from "@/components/onboarding";
import { AlertTriangle, ChevronRight, X, ExternalLink } from "lucide-react";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";

// ─── 演示数据 ──────────────────────────────────────────────────

const ADMIN_SLIDES: GlobalModalSlide[] = [
  { titleLeft: "管控端", titleRight: "更简洁高效", desc: "在保持原有操作习惯的基础上，界面更简约清爽，信息屏效比更高，颜值质感双提升！" },
  { titleLeft: "用户端", titleRight: "更清晰美观", desc: "全新界面更美观，观感更舒适清爽，布局更清晰简约，快来体验新版设计吧！" },
];

const ADMIN_SINGLE_SLIDES: GlobalModalSlide[] = [
  { titleLeft: "管控端", titleRight: "更简洁高效", desc: "在保持原有操作习惯的基础上，界面更简约清爽，信息屏效比更高，颜值质感双提升！" },
];

const TENANT_CAROUSEL_SLIDES: GlobalModalSlide[] = [
  { titleRight: "主标题核心标语", desc: "在保持原有操作习惯的基础上，界面更简约清爽，信息屏效比更高，颜值质感双提升！" },
  { titleRight: "对话视图上线", desc: "全新对话模式，让您与 Agent 的交互更加自然流畅，支持多轮对话和上下文记忆。" },
  { titleRight: "技能广场全新登场", desc: "在技能广场浏览和安装丰富的技能扩展，让您的 Agent 能力更强大、更智能。" },
];

const TENANT_SLIDES: GlobalModalSlide[] = [
  { titleRight: "主标题核心标语", desc: "在保持原有操作习惯的基础上，界面更简约清爽，信息屏效比更高，颜值质感双提升！" },
];

const DEMO_MODULE_ITEMS: ModuleFloatItem[] = [
  { title: "新增「按技能包分组」视图", description: "技能列表支持按技能包维度分组展示，可快速定位包内技能，提升管理效率。" },
  { title: "批量操作优化", description: "支持跨页勾选技能进行批量删除、分发、移动操作，不再受单页数量限制。" },
  { title: "搜索体验增强", description: "搜索框支持按名称、描述、标签等多维度模糊匹配，精准定位目标技能。" },
];

const DEMO_CHANGELOG: ChangelogVersion[] = [
  {
    version: "v3.2.0", date: "2026-06-09",
    entries: [
      { id: "1", title: "技能包管理合并", description: "公共技能库和企业技能库整合为技能管理中心", tag: "结构", date: "2026-06-09", href: "/admin/skill-config" },
      { id: "2", title: "成员分组支持多组", description: "一个用户可以属于多个分组，分组策略支持继承", tag: "逻辑", date: "2026-06-09" },
      { id: "3", title: "新增「分发状态」列", description: "企业插件库列表新增分发状态字段", tag: "元素", date: "2026-06-09" },
      { id: "4", title: "安全模块上线", description: "新增 AI Agent 安全防护功能", tag: "系统", date: "2026-06-09" },
    ],
  },
  {
    version: "v3.1.0", date: "2026-05-20",
    entries: [
      { id: "6", title: "「插件管理」更名", description: "「插件管理」更名为「企业插件库」，功能不变", tag: "元素", date: "2026-05-20" },
    ],
  },
];

const DEMO_HIGHLIGHT_REGIONS: HighlightRegion[] = [
  { id: "r1", top: 120, left: 80, width: 260, height: 50, title: "新增筛选栏", description: "这里新增了「按分发状态筛选」和「按技能包分组」两个筛选器。", bubblePlacement: "right" },
];

// ─── 组件元信息 ──────────────────────────────────────────────

interface ComponentInfo {
  id: string;
  code: string;
  title: string;
  desc: string;
  /** 推荐字数限制说明 */
  charLimit?: string;
  badge: "both" | "admin" | "tenant";
  constraints: { label: string; value: string }[];
  scenes: { tag: string; text: string }[];
  /** 预览舞台高度 */
  stageHeight: number;
}

const TENANT_COMPONENTS: ComponentInfo[] = [
  {
    id: "global-modal-tenant",
    code: "",
    title: "全局弹窗 GuideGlobalModal",
    desc: "影响面极大的更新 — 全屏阻断弹窗。变体：① 多条内容 variant=\"carousel\"（轮播+指示器） ② 单条内容 variant=\"single\"（双按钮模式）",
    charLimit: "推荐字数：主标题 ≤10 字，副标题 ≤20 字，按钮文案 ≤6 字",
    badge: "both",
    stageHeight: 0,
    constraints: [
      { label: "尺寸", value: "固定 680×512px，居中展示" },
      { label: "圆角", value: "8px" },
      { label: "遮罩", value: "bg-black/50 全屏遮罩" },
      { label: "变体", value: "用户端：single（单视频铺满 + 底部标题/按钮）" },
      { label: "配图", value: "配图范围 1080×608(@2x)，实际渲染 540×304" },
      { label: "大标题", value: "24px Semibold 渐变填充 + text-shadow，固定\"全站视觉焕新升级\"" },
      { label: "主标语", value: "16px Medium #000" },
      { label: "副标题", value: "12px Regular #737373，letter-spacing -0.0833em" },
      { label: "主按钮", value: "140×36px 渐变 linear-gradient(90deg, #020617 70%, #1447E6 100%)，圆角 60px（胶囊形）" },
      { label: "次按钮", value: "140×36px 白底 #E5E5E5 描边，圆角 60px（可选，双按钮模式）" },
      { label: "切换箭头", value: "24×24 圆形描边按钮，stroke rgba(0,0,0,0.29)" },
      { label: "白色遮罩", value: "固定遮罩 linear-gradient(180deg, transparent 3%, #fff 33%)，从 y=255 开始" },
      { label: "关闭按钮", value: "右上角 24×24，圆角左下 20px，rgba(255,255,255,0.4)" },
      { label: "动画", value: "视频 autoPlay loop muted，淡入 0.35s" },
      { label: "字数限制", value: "主标题 ≤10 字，副标题 ≤20 字，按钮文案 ≤6 字" },
      { label: "响应式", value: "固定 680×512px 居中，不随视口缩放；小屏(<768px)不展示" },
    ],
    scenes: [
      { tag: "结构层 1.1", text: "新增子页面 — 产品新增完整子页面" },
      { tag: "结构层 1.4", text: "页面整合/拆分 — 多个页面合并或拆分" },
      { tag: "系统层 4.1", text: "账号体系变更 — 登录方式/认证逻辑变化" },
      { tag: "系统层 4.3", text: "数据合规/隐私变更" },
    ],
  },
  {
    id: "module-float-tenant",
    code: "<GuideModuleFloat variant=\"single | multi\" />",
    title: "非阻断浮窗 GuideModuleFloat（用户端）",
    desc: "模块级更新 — 不阻断操作，右下角固定浮窗。变体：① 单条内容 variant=\"single\"（CTA 胶囊按钮） ② 多条内容 variant=\"multi\"（翻页导航）",
    charLimit: "推荐字数：标题 ≤12 字，描述 ≤32 字（含行动链接共两行）",
    badge: "both",
    stageHeight: 0,
    constraints: [
      { label: "位置", value: "fixed bottom-6 right-6（右下角）" },
      { label: "宽度", value: "360px" },
      { label: "圆角", value: "12px（xl）" },
      { label: "变体", value: "single（单CTA 胶囊按钮）/ multi（翻页导航）" },
      { label: "配图", value: "16:9 比例，rounded-xl" },
      { label: "CTA按钮", value: "Button tenant-primary 80×28px（单条模式）" },
      { label: "多条模式", value: "左侧\"n/N 跳过引导\"，右侧 ←→ 翻页，末页显示\"我知道了\"" },
      { label: "翻页按钮", value: "Button tenant-outline 38×28px，全圆角胶囊" },
      { label: "动画", value: "slide-in-from-bottom-4 duration-300" },
      { label: "层级", value: "z-[9990]" },
      { label: "字数限制", value: "标题 ≤12 字，描述 ≤32 字（含行动链接共两行）" },
      { label: "响应式", value: "固定宽 360px，fixed 定位右下角；小屏(<768px)宽度收缩为 calc(100vw - 48px)" },
    ],
    scenes: [
      { tag: "结构层 1.2", text: "页面重新排布 — 区域位置/顺序/分组变化" },
      { tag: "元素层 2.6", text: "细节优化叠加 — 多个微小优化汇总" },
      { tag: "跨端层 5.2", text: "管控端变化告知C端" },
    ],
  },
  {
    id: "point-bubble-tenant",
    code: "<GuidePointBubble contentVariant=\"text-only | text-button | text-image | push-notice\" variant=\"light\" />",
    title: "单UI提示气泡 GuidePointBubble（浅色）",
    desc: "直接在 UI 附近展示的点对点引导。变体：① 1.1 纯文本 contentVariant=\"text-only\" ② 1.2 纯文本+按钮 contentVariant=\"text-button\" ③ 1.3 纯文本+图片 contentVariant=\"text-image\" ④ 1.4 重点推送通知 contentVariant=\"push-notice\"",
    charLimit: "推荐字数：标题 ≤12 字，描述 ≤40 字，列表每项 ≤20 字（最多 3 条）",
    badge: "both",
    stageHeight: 560,
    constraints: [
      { label: "宽度", value: "气泡内容 260px（含 padding 约 284px）" },
      { label: "内边距", value: "文本容器 padding 12px，内部 gap 12px" },
      { label: "圆角", value: "气泡 4px；用户端按钮 tenant-outline/tenant-primary（全圆角胶囊）" },
      { label: "阴影", value: "0 8px 48px -12px rgba(0,0,0,.1), 0 1px 3px rgba(0,0,0,.05)" },
      { label: "标题", value: "14px / Medium / #000，可附蓝色标签" },
      { label: "描述", value: "12px(列表) / 14px(图片) Regular，rgba(0,0,0,.7)" },
      { label: "方向", value: "top / bottom / left / right 四向箭头" },
      { label: "标签", value: "bg rgba(20,71,230,.06) · 文字 #1447E6 · 圆角 2px · 11px" },
      { label: "有序列表", value: "序号+文本，最多 3 条" },
      { label: "配图", value: "520×292（16:9），容器 260×146" },
      { label: "1.2 按钮", value: "主 Button tenant-primary / 次 Button tenant-outline，全圆角胶囊" },
      { label: "1.4 推送", value: "渐变 #2C59E9→#5980FF，白标题 + rgba(255,255,255,.7) 描述" },
      { label: "脉冲热点", value: "12px 蓝色圆点 + ping 动画" },
      { label: "层级", value: "z-[9985]" },
      { label: "字数限制", value: "标题 ≤12 字，描述 ≤40 字，列表每项 ≤20 字（最多 3 条）" },
      { label: "响应式", value: "固定宽 260px，相对目标元素定位；不随视口缩放" },
    ],
    scenes: [
      { tag: "元素层 2.1", text: "新增按钮/操作入口" },
      { tag: "元素层 2.2", text: "新增表格列/字段" },
      { tag: "元素层 2.3", text: "新增筛选/排序选项" },
      { tag: "元素层 2.4", text: "名称/文案变更 — 标注改名元素" },
      { tag: "跨端层 5.2", text: "管控端变化告知 C 端" },
    ],
  },
  {
    id: "highlight-bubble-tenant",
    code: "<GuidePointBubble variant=\"dark\" contentVariant=\"text-only\" showSteps currentStep={n} totalSteps={N} showHotspot />",
    title: "步骤指引气泡 GuidePointBubble（用户端）",
    desc: "带呼吸灯的步骤性指引气泡 — 深色蓝底气泡 + 呼吸灯热点 + 步骤导航（n/N）。通过 variant=\"dark\" + showSteps 启用步骤模式，支持四个方向、有序列表、矩形标注。",
    charLimit: "推荐字数：标题 ≤12 字，描述 ≤40 字，步骤总数 ≤5 步",
    badge: "both",
    stageHeight: 280,
    constraints: [
      { label: "气泡宽", value: "278px（步骤模式 isStepGuide）" },
      { label: "背景", value: "纯蓝底 #2C59E9" },
      { label: "圆角", value: "var(--guide-bubble-radius)（4px）" },
      { label: "标题", value: "13px Medium 白色" },
      { label: "描述", value: "12px Regular 白色 90%" },
      { label: "呼吸灯", value: "圆形 14px 蓝点 + ping 动画；可切矩形标注（hotspotShape=\"rect\"）" },
      { label: "步骤导航", value: "\"n/N\" 计数 + 38×28 箭头按钮，末步显示\"我知道了\"文字按钮" },
      { label: "方向", value: "top / bottom / left / right 四向箭头" },
      { label: "变体", value: "深色指引（跨功能页面）/ 浅色指引（局部）" },
      { label: "端差异", value: "用户端 rounded-[24px] 胶囊 / 管控端 4px 方角（endpoint prop 切换）" },
      { label: "字数限制", value: "标题 ≤12 字，描述 ≤40 字，步骤总数 ≤5 步" },
      { label: "响应式", value: "气泡 278px 固定宽，相对目标元素定位；不随视口缩放" },
    ],
    scenes: [
      { tag: "结构层 1.2", text: "页面重新排布 — 标注新布局各区域" },
      { tag: "元素层 2.1", text: "新增按钮/操作入口 — 高亮新增交互元素" },
      { tag: "元素层 2.2", text: "新增表格列 — 高亮新列并解释含义" },
      { tag: "元素层 2.4", text: "名称/文案变更 — 标注改名元素" },
    ],
  },
];

const ADMIN_COMPONENTS: ComponentInfo[] = [
  {
    id: "global-modal-admin",
    code: "",
    title: "全局弹窗 GuideGlobalModal",
    desc: "影响面极大的更新 — 全屏阻断弹窗。变体：① 多条内容 variant=\"carousel\"（轮播+竖线标题） ② 单条内容 variant=\"single\"（双按钮模式）",
    charLimit: "推荐字数：主标题 ≤10 字，副标题 ≤20 字，按钮文案 ≤6 字",
    badge: "both",
    stageHeight: 0,
    constraints: [
      { label: "尺寸", value: "固定 680×512px，居中展示" },
      { label: "圆角", value: "8px" },
      { label: "变体", value: "管控端：carousel（多视频轮播 + 左右箭头 + 指示器圆点）" },
      { label: "配图", value: "配图范围 1080×608(@2x)，实际渲染 540×304" },
      { label: "大标题", value: "24px Semibold 渐变填充 + text-shadow" },
      { label: "标题", value: "16px Medium #000，\"XX端 | 标题\" 竖线分割（1px宽 12px高 #CCC，gap=10px）" },
      { label: "副标题", value: "12px Regular #737373，letter-spacing -0.0833em" },
      { label: "主按钮", value: "140×36px 渐变 linear-gradient(90deg, #020617 70%, #1447E6 100%)，圆角 4px" },
      { label: "次按钮", value: "140×36px 白底 #E5E5E5 描边，圆角 4px（可选，双按钮模式）" },
      { label: "切换箭头", value: "24×24 圆形描边按钮 border-radius 40px，stroke rgba(0,0,0,0.29)" },
      { label: "指示器", value: "圆角胶囊点：active 18×4 #000，inactive 4×4 #CACFDD，gap=4px，圆角 20px" },
      { label: "白色遮罩", value: "固定遮罩 linear-gradient(180deg, transparent 3%, #fff 33%)，从 y=255 开始" },
      { label: "关闭按钮", value: "右上角 24×24，圆角左下 20px，rgba(255,255,255,0.4)" },
      { label: "过渡", value: "opacity 0.35s ease 淡入淡出切换" },
      { label: "字数限制", value: "主标题 ≤10 字，副标题 ≤20 字，按钮文案 ≤6 字" },
      { label: "响应式", value: "固定 680×512px 居中，不随视口缩放；小屏(<768px)不展示" },
    ],
    scenes: [
      { tag: "结构层 1.1", text: "新增子页面" },
      { tag: "结构层 1.4", text: "页面整合/拆分" },
      { tag: "系统层 4.2", text: "权限体系变更" },
      { tag: "逻辑层 3.3", text: "计费/配额变更" },
    ],
  },
  {
    id: "nav-bubble-admin",
    code: "<GuideNavBubble placement=\"right | bottom | left\" />",
    title: "导航预览气泡 GuideNavBubble",
    desc: "新功能入口预览 — 依附在导航项旁的介绍气泡。变体：通过 placement 控制方向（right / bottom / left）",
    charLimit: "推荐字数：标题 ≤15 字，描述 ≤50 字，按钮文案 ≤6 字",
    badge: "admin",
    stageHeight: 220,
    constraints: [
      { label: "宽度", value: "300px" },
      { label: "圆角", value: "12px（xl）" },
      { label: "箭头", value: "SVG 三角 8×16，指向导航入口" },
      { label: "方向", value: "right / bottom / left（相对目标）" },
      { label: "预览图", value: "140px 高，border-bottom 分割" },
      { label: "NEW 标签", value: "蓝色小标签 bg-blue-50 text-blue-600" },
      { label: "操作", value: "黑色小按钮 + 灰色\"稍后再说\"" },
      { label: "层级", value: "z-[9980]" },
      { label: "字数限制", value: "标题 ≤15 字，描述 ≤50 字，按钮文案 ≤6 字" },
      { label: "响应式", value: "固定宽 300px，相对导航项定位；导航栏折叠时隐藏" },
    ],
    scenes: [
      { tag: "结构层 1.5", text: "页面入口新增 — 侧边栏/导航出现新入口" },
      { tag: "结构层 1.3", text: "功能位置变动 — 功能从 A 搬到 B" },
    ],
  },
  {
    id: "module-float-admin",
    code: "<AdminNotifyCard />",
    title: "非阻断浮窗 AdminNotifyCard（管控端）",
    desc: "管控端通知卡片 — 浅色渐变主题，显示在左侧导航区域。变体：① 管控端单个卡片 ② 管控端聚合卡片 ③ 用户端单个卡片 ④ 用户端聚合卡片 ⑤ 多条叠加展示",
    charLimit: "推荐字数：标题 ≤16 字（含\"XX端 | \"前缀），描述 ≤30 字，按钮文案 ≤6 字",
    badge: "both",
    stageHeight: 0,
    constraints: [
      { label: "位置", value: "管控端左侧导航栏旁" },
      { label: "宽度", value: "220px" },
      { label: "内边距", value: "12px，内部 gap 8px" },
      { label: "圆角", value: "8px" },
      { label: "阴影", value: "0px 4px 12px rgba(0,0,0,0.1)" },
      { label: "皮肤1(紫)", value: "bg linear-gradient(179deg, #F2F5FF, rgba(252,252,254,0.93))，border #E3E8FA" },
      { label: "皮肤2(蓝)", value: "bg linear-gradient(179deg, #F2FBFF, rgba(252,252,254,0.93))，border #E3EDFA" },
      { label: "皮肤3(绿)", value: "bg linear-gradient(179deg, #F2FFF5, rgba(252,252,254,0.93))，border #E8F1E9" },
      { label: "副标题", value: "12px Regular rgba(0,0,0,0.5) \"产品动态\"" },
      { label: "标题", value: "14px Medium #000，\"管控端 | 功能名称\" 格式" },
      { label: "描述", value: "12px Regular rgba(0,0,0,0.5)" },
      { label: "按钮", value: "黑底白字 #000，高 28px，圆角 4px，全宽" },
      { label: "多条模式", value: "按时间顺序叠加展示，第二条自动缩小 0.9，关闭后展示另一条" },
      { label: "聚合卡片", value: "\"管控端有 N 项新增\" + \"查看详情\" 跳转产品动态抽屉" },
      { label: "字数限制", value: "标题 ≤16 字（含\"XX端 | \"前缀），描述 ≤30 字，按钮 ≤6 字" },
      { label: "响应式", value: "固定宽 220px，相对导航栏定位；导航栏折叠时跟随隐藏" },
    ],
    scenes: [
      { tag: "结构层 1.2", text: "页面重新排布" },
      { tag: "元素层 2.6", text: "细节优化叠加" },
      { tag: "跨端层 5.1", text: "C端变化告知管控端" },
    ],
  },
  {
    id: "point-bubble-admin",
    code: "<GuidePointBubble contentVariant=\"text-only | text-button | text-image | push-notice\" endpoint=\"admin\" />",
    title: "单UI提示气泡 GuidePointBubble（管控端）",
    desc: "内容与用户端一致，仅按钮圆角替换为管控端规范（4px 方角）。支持 text-only / text-button / text-image / push-notice 四种变体",
    charLimit: "推荐字数：标题 ≤12 字，描述 ≤40 字，列表每项 ≤20 字（最多 3 条）",
    badge: "both",
    stageHeight: 0,
    constraints: [
      { label: "宽度", value: "280px 固定" },
      { label: "四变体", value: "text-only / text-button / text-image / push-notice" },
      { label: "图片高", value: "120px object-cover（text-image）" },
      { label: "步骤", value: "支持 currentStep/totalSteps 步骤指示器" },
      { label: "按钮", value: "Button claw-outline / claw-primary，4px 方角（管控端规范）" },
      { label: "字数限制", value: "标题 ≤12 字，描述 ≤40 字，列表每项 ≤20 字（最多 3 条）" },
      { label: "响应式", value: "固定宽 280px，相对目标元素定位；不随视口缩放" },
    ],
    scenes: [
      { tag: "元素层 2.1", text: "新增按钮/操作入口" },
      { tag: "元素层 2.2", text: "新增表格列/字段" },
      { tag: "元素层 2.3", text: "新增筛选/排序选项" },
    ],
  },
  {
    id: "highlight-bubble-admin",
    code: "<GuidePointBubble variant=\"dark\" contentVariant=\"text-only\" showSteps currentStep={n} totalSteps={N} showHotspot endpoint=\"admin\" />",
    title: "步骤指引气泡 GuidePointBubble（管控端）",
    desc: "内容与用户端一致，按钮圆角替换为管控端规范（4px 方角）。深色蓝底气泡 + 呼吸灯热点 + 步骤导航，通过 endpoint=\"admin\" 切换方角。",
    charLimit: "推荐字数：标题 ≤12 字，描述 ≤40 字，步骤总数 ≤5 步",
    badge: "both",
    stageHeight: 280,
    constraints: [
      { label: "气泡宽", value: "278px（步骤模式 isStepGuide）" },
      { label: "背景", value: "纯蓝底 #2C59E9" },
      { label: "圆角", value: "var(--guide-bubble-radius)（4px）" },
      { label: "呼吸灯", value: "圆形 14px 蓝点 + ping 动画；可切矩形标注（hotspotShape=\"rect\"）" },
      { label: "导航按钮", value: "38×28 箭头按钮，endpoint=\"admin\" → 4px 方角" },
      { label: "最后一步", value: "右箭头替换为\"我知道了\"文字按钮" },
      { label: "字数限制", value: "标题 ≤12 字，描述 ≤40 字，步骤总数 ≤5 步" },
      { label: "响应式", value: "气泡 278px 固定宽，相对目标元素定位；不随视口缩放" },
    ],
    scenes: [
      { tag: "结构层 1.2", text: "页面重新排布" },
      { tag: "元素层 2.1~2.4", text: "新增/变更元素" },
    ],
  },
  {
    id: "update-bar-admin",
    code: "<GuideUpdateBar />",
    title: "强提醒公告条",
    desc: "不可关闭的顶部公告条 — 告警样式强提醒",
    badge: "admin",
    stageHeight: 120,
    constraints: [
      { label: "背景", value: "琥珀色 #FEF3C7" },
      { label: "边框", value: "border-b border-[#F59E0B]/30" },
      { label: "图标", value: "AlertTriangle 16px #D97706" },
      { label: "文字", value: "14px #92400E" },
      { label: "版本标签", value: "10px #92400E bg-[#FDE68A] rounded" },
      { label: "位置", value: "sticky top:64px（导航栏下方）z-49" },
      { label: "关闭", value: "不可关闭（强提醒）" },
      { label: "动画", value: "slide-in-from-top-1 duration-200" },
    ],
    scenes: [
      { tag: "元素层 2.5", text: "New Tag — 最轻量感知手段" },
      { tag: "跨端层 5.1", text: "C端变化告知管控端" },
      { tag: "跨端层 5.2", text: "管控端变化告知C端" },
      { tag: "时间维度 6.1", text: "预告期 — 版本上线倒计时" },
    ],
  },
  {
    id: "changelog-drawer-admin",
    code: "<GuideChangelogDrawer />",
    title: "更新记录抽屉",
    desc: "版本记录汇总 — 右侧滑出的侧边抽屉",
    badge: "admin",
    stageHeight: 380,
    constraints: [
      { label: "宽度", value: "max-w-[420px] 右侧全高" },
      { label: "遮罩", value: "bg-black/30 backdrop-blur-[2px]" },
      { label: "层级", value: "z-[9995]" },
      { label: "动画", value: "slide-in-from-right duration-300" },
      { label: "标签色", value: "结构(紫) / 元素(蓝) / 逻辑(琥珀) / 系统(红) / 跨端(绿)" },
      { label: "条目样式", value: "左侧圆点 + 标签 + 标题 + 描述" },
      { label: "版本粘性", value: "sticky top-0 版本号标题" },
      { label: "操作", value: "hover 时显示跳转图标" },
    ],
    scenes: [
      { tag: "所有层级", text: "按版本聚合完整更新记录" },
      { tag: "触发方式", text: "从提示条\"查看详情\"触发" },
      { tag: "主动查阅", text: "用户主动查阅历史版本变更" },
    ],
  },
];

// ─── 内嵌舞台：把 fixed 定位组件限制在容器内并按需缩放 ────────────

function InlineStage({
  height,
  scale = 1,
  children,
}: {
  height: number;
  scale?: number;
  children: React.ReactNode;
}) {
  return (
    <div
      className="relative w-full overflow-hidden rounded-lg
        [&_.fixed]:!absolute"
      style={{ height }}
    >
      <div
        className="absolute inset-0 flex items-center justify-center"
        style={scale !== 1 ? { transform: `scale(${scale})`, transformOrigin: "center center" } : undefined}
      >
        {children}
      </div>
    </div>
  );
}

/**
 * 更新记录抽屉的内嵌渲染（真实组件依赖 fixed + 右侧贴边，
 * 在卡片内用 InlineStage 限制并左移到可视范围）。
 * 这里复刻抽屉内容样式，与真实组件保持一致（同一套 className）。
 */
function InlineChangelogDrawer({ versions }: { versions: ChangelogVersion[] }) {
  const tagColors: Record<string, string> = {
    "结构": "bg-purple-50 text-purple-600 border-purple-200",
    "元素": "bg-blue-50 text-blue-600 border-blue-200",
    "逻辑": "bg-amber-50 text-amber-600 border-amber-200",
    "系统": "bg-red-50 text-red-600 border-red-200",
    "跨端": "bg-green-50 text-green-600 border-green-200",
  };
  return (
    <div className="w-[360px] h-full bg-white shadow-2xl rounded-lg overflow-hidden flex flex-col mx-auto">
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 shrink-0">
        <div>
          <h2 className="text-base font-semibold text-gray-900">更新记录</h2>
          <p className="text-xs text-gray-400 mt-0.5">每次版本更新的内容概览，点击可直达对应功能</p>
        </div>
        <button className="w-8 h-8 rounded-lg flex items-center justify-center hover:bg-gray-100 transition-colors">
          <X className="w-4 h-4 text-gray-500" />
        </button>
      </div>
      <div className="overflow-y-auto flex-1 px-6 py-4">
        {versions.map((ver) => (
          <div key={ver.version} className="mb-8 last:mb-0">
            <div className="flex items-center gap-2 mb-3 sticky top-0 bg-white py-1">
              <span className="px-2 py-0.5 text-xs font-semibold text-gray-800 bg-gray-100 rounded-md">{ver.version}</span>
              <span className="text-xs text-gray-400">{ver.date}</span>
            </div>
            <div className="space-y-3">
              {ver.entries.map((e) => (
                <div key={e.id} className="group flex gap-3">
                  <span className="shrink-0 mt-1.5 w-1.5 h-1.5 rounded-full bg-gray-300" />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-0.5">
                      <span className={`px-1.5 py-0.5 text-[10px] font-medium rounded border ${tagColors[e.tag]}`}>{e.tag}</span>
                      <h4 className="text-sm font-medium text-gray-800 truncate">{e.title}</h4>
                      {e.href && <ExternalLink className="w-3 h-3 text-gray-300 opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />}
                    </div>
                    <p className="text-xs text-gray-500 leading-relaxed">{e.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/** 强提醒公告条的内嵌渲染（真实组件用 Portal 到页面顶部，无法直接嵌入，复用同款 className） */
function InlineUpdateBar() {
  return (
    <div className="w-full max-w-[560px] bg-[#FEF3C7] border border-[#F59E0B]/30 rounded-lg px-4 py-2.5">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2.5 min-w-0">
          <AlertTriangle className="w-4 h-4 shrink-0 text-[#D97706]" />
          <span className="shrink-0 px-1.5 py-0.5 text-[10px] font-medium text-[#92400E] bg-[#FDE68A] rounded">v3.2</span>
          <span className="text-sm text-[#92400E]">系统将于 2026年7月1日 进行计费规则调整，请提前确认资源配额</span>
        </div>
        <button className="inline-flex items-center gap-0.5 text-xs font-medium text-[#D97706] hover:text-[#92400E] transition-colors shrink-0">
          查看详情
          <ChevronRight className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}

// ─── 各组件内嵌原型（全部使用真实组件） ─────────────────────────

function StagePointBubble({ dark }: { dark?: boolean }) {
  return (
    <GuidePointBubble
      open
      onClose={() => {}}
      title={dark ? "🎉 版本升级通知" : "新增批量操作"}
      description={dark ? "全新能力已上线，立即查看详情" : "选中多条记录后可批量执行操作"}
      variant={dark ? "dark" : "light"}
      contentVariant={dark ? "push-notice" : "text-button"}
      actionText={dark ? "立即查看" : "知道了"}
      showHotspot
      placement="bottom"
    />
  );
}

/**
 * 用户端单UI提示气泡 — 1.1 纯文本类型矩阵（全部真实 GuidePointBubble 组件）
 * 严格对齐 Figma node 4096:9477「用户端指引气泡 - 1.1 纯文本类型，支持四个方向」
 *
 * 结构：3 行（常规版 / 可配标签 / 支持有序文本）× 4 方向（上 / 下 / 左 / 右）
 */
const PB_TITLE = "标题文本介绍";
const PB_DESC = "快速管理已开通的企业应用，也可以在这里创建自建应用实现统一登录。";
const PB_LIST = [
  "管控端新增「Agent 类型管理」页面支持镜像更新推送",
  "管控端 Agent 列表工具栏新增镜像更新提醒铃铛",
  "管控端平台策略新增「允许员工自助更新 Agent 版本」开关",
];
const PB_PLACEMENTS: Array<"top" | "bottom" | "left" | "right"> = ["top", "bottom", "left", "right"];
const PB_PLACEMENT_LABEL: Record<string, string> = { top: "下", bottom: "上", left: "右", right: "左" };

function TenantPointBubbleRow({
  rowLabel,
  rowHint,
  render,
}: {
  rowLabel: string;
  rowHint?: string;
  render: (placement: "top" | "bottom" | "left" | "right") => React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-6">
      {/* 行标签 */}
      <div className="w-[120px] shrink-0 pt-6">
        <div className="text-[13px] text-[var(--text-secondary)] leading-relaxed">{rowLabel}</div>
        {rowHint && <div className="text-[11px] text-[var(--text-weak)] mt-1 leading-relaxed">{rowHint}</div>}
      </div>
      {/* 4 个方向：每列最小 290px，保证 280px 气泡不被挤压重叠；空间不足时自动换行 */}
      <div
        className="grid gap-x-8 gap-y-8 flex-1"
        style={{ gridTemplateColumns: "repeat(auto-fit, minmax(290px, 1fr))" }}
      >
        {PB_PLACEMENTS.map((p) => (
          <div key={p} className="flex flex-col items-center gap-3 min-w-0">
            {render(p)}
            <span className="text-[11px] text-[var(--text-weak)]">箭头朝{PB_PLACEMENT_LABEL[p]}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function TenantPointBubbleGallery({ endpoint = "tenant" }: { endpoint?: "admin" | "tenant" }) {
  return (
    <div className="w-full space-y-10">
      {/* ─── 1.1 纯文本类型 ─── */}
      <div>
        <h3 className="text-[20px] font-semibold text-[var(--text-title)] mb-2">1.1 纯文本类型</h3>
        <code className="text-[12px] font-mono text-gray-400 inline-block">{`<GuidePointBubble contentVariant="text-only" />`}</code>
      </div>

      {/* 常规版 */}
      <TenantPointBubbleRow
        rowLabel="常规版"
        render={(p) => (
          <GuidePointBubble
            open onClose={() => {}} title={PB_TITLE} description={PB_DESC}
            contentVariant="text-only" showHotspot={false} placement={p} endpoint={endpoint}
          />
        )}
      />

      {/* 可配标签 */}
      <TenantPointBubbleRow
        rowLabel="可配标签"
        rowHint="标签用组件样式"
        render={(p) => (
          <GuidePointBubble
            open onClose={() => {}} title={PB_TITLE} tag="还有 7 天上线" description={PB_DESC}
            contentVariant="text-only" showHotspot={false} placement={p} endpoint={endpoint}
          />
        )}
      />

      {/* 支持添加有序文本 */}
      <TenantPointBubbleRow
        rowLabel="支持添加有序文本"
        rowHint="有序列表不超过 3 条"
        render={(p) => (
          <GuidePointBubble
            open onClose={() => {}} title={PB_TITLE} description={PB_DESC}
            contentVariant="text-only" listItems={PB_LIST} showHotspot={false} placement={p} endpoint={endpoint}
          />
        )}
      />

      {/* ─── 1.2 纯文本 + 按钮 ─── */}
      <div className="pt-6 border-t border-[var(--border)]">
        <h3 className="text-[20px] font-semibold text-[var(--text-title)] mb-2">1.2 纯文本 + 按钮</h3>
        <code className="text-[12px] font-mono text-gray-400 inline-block mb-6">{`<GuidePointBubble contentVariant="text-button" />`}</code>
      </div>

      {/* 可配单/双按钮 */}
      <TenantPointBubbleRow
        rowLabel="可配单/双按钮"
        rowHint="用户端按钮为全圆角"
        render={(p) => (
          <GuidePointBubble
            open onClose={() => {}} title={PB_TITLE} description={PB_DESC}
            contentVariant="text-button"
            secondaryActionText={p === "top" || p === "left" ? "了解更多" : undefined}
            actionText="我知道了"
            showHotspot={false} placement={p} endpoint={endpoint}
          />
        )}
      />

      {/* 有序+按钮 */}
      <TenantPointBubbleRow
        rowLabel="有序+按钮"
        render={(p) => (
          <GuidePointBubble
            open onClose={() => {}} title={PB_TITLE} description={PB_DESC}
            contentVariant="text-button"
            listItems={PB_LIST}
            secondaryActionText={p === "top" || p === "left" ? "了解更多" : undefined}
            actionText="我知道了"
            showHotspot={false} placement={p} endpoint={endpoint}
          />
        )}
      />

      {/* ─── 1.3 纯文本 + 图片 ─── */}
      <div className="pt-6 border-t border-[var(--border)]">
        <h3 className="text-[20px] font-semibold text-[var(--text-title)] mb-2">1.3 纯文本 + 图片</h3>
        <code className="text-[12px] font-mono text-gray-400 inline-block mb-6">{`<GuidePointBubble contentVariant="text-image" />`}</code>
      </div>

      {/* 支持配图 */}
      <TenantPointBubbleRow
        rowLabel="支持配图"
        rowHint="图片尺寸为：520 × 292&#10;比例为 16:9"
        render={(p) => (
          <GuidePointBubble
            open onClose={() => {}} title="运维管理功能升级" description={PB_DESC}
            contentVariant="text-image"
            imageCaption="2026-04-20"
            showHotspot={false} placement={p} endpoint={endpoint}
          />
        )}
      />

      {/* ─── 1.4 重点推送通知 ─── */}
      <div className="pt-6 border-t border-[var(--border)]">
        <h3 className="text-[20px] font-semibold text-[var(--text-title)] mb-2">1.4 重点推送通知，如：版本升级</h3>
        <code className="text-[12px] font-mono text-gray-400 inline-block mb-6">{`<GuidePointBubble contentVariant="push-notice" variant="dark" />`}</code>
      </div>

      {/* 三个变体横向排列 */}
      <div className="flex items-start gap-8 flex-wrap">
        <div className="flex flex-col items-center gap-3">
          <span className="text-[13px] text-[var(--text-secondary)]">纯文本</span>
          <GuidePointBubble
            open onClose={() => {}} title="版本升级，重磅来袭！"
            description="当前版本为 V1.0升级版本后可一件接入微信"
            variant="dark" contentVariant="push-notice"
            showHotspot={false} placement="bottom" endpoint={endpoint}
          />
          <span className="text-[11px] text-[var(--text-weak)]">步骤指引</span>
        </div>
        <div className="flex flex-col items-center gap-3">
          <span className="text-[13px] text-[var(--text-secondary)]">纯文本+按钮</span>
          <GuidePointBubble
            open onClose={() => {}} title="版本升级，重磅来袭！"
            description="当前版本为 V1.0升级版本后可一件接入微信"
            variant="dark" contentVariant="push-notice"
            secondaryActionText="知道了"
            actionText="立即升级"
            showHotspot={false} placement="bottom" endpoint={endpoint}
          />
          <span className="text-[11px] text-[var(--text-weak)]">步骤指引</span>
        </div>
        <div className="flex flex-col items-center gap-3">
          <span className="text-[13px] text-[var(--text-secondary)]">纯文本+图片+按钮</span>
          <GuidePointBubble
            open onClose={() => {}} title="版本升级，重磅来袭！"
            description="当前版本为 V1.0升级版本后可一件接入微信"
            variant="dark" contentVariant="push-notice"
            noticeImage=""
            secondaryActionText="知道了"
            actionText="立即升级"
            showHotspot={false} placement="bottom" endpoint={endpoint}
          />
          <span className="text-[11px] text-[var(--text-weak)]">步骤指引</span>
        </div>
      </div>

      
    </div>
  );
}

const STEP_TITLES = [
  "第1步：点击对话图开启对话",
  "第2步：点击对话视图开启对话",
  "第3步：点击对话视图开启对话",
];
const STEP_DESC = "您已完成基础配置，可以开始在浏览器中和Agent进行对话了";

/** 可交互的步骤指引气泡（内置 state，点击箭头切换步骤） */
function InteractiveStepBubble({
  placement,
  listItems,
  actionText = "我知道了",
  hotspotShape,
  hotspotSize,
  endpoint = "tenant",
}: {
  placement: "top" | "bottom" | "left" | "right";
  listItems?: string[];
  actionText?: string;
  hotspotShape?: "circle" | "rect";
  hotspotSize?: { width: number; height: number };
  endpoint?: "admin" | "tenant";
}) {
  const [step, setStep] = useState(1);
  return (
    <GuidePointBubble
      open onClose={() => setStep(1)}
      title={STEP_TITLES[step - 1]}
      description={STEP_DESC}
      variant="dark" contentVariant="text-only"
      listItems={listItems}
      currentStep={step} totalSteps={3} showSteps
      actionText={actionText}
      showHotspot placement={placement}
      hotspotShape={hotspotShape}
      hotspotSize={hotspotSize}
      endpoint={endpoint}
      onNext={() => setStep((s) => Math.min(3, s + 1))}
      onPrev={() => setStep((s) => Math.max(1, s - 1))}
    />
  );
}

/** 步骤指引气泡画廊 — 对齐 Figma「2/ 步骤指引」node 4113-10615 */
function StepGuideBubbleGallery({ endpoint = "tenant" }: { endpoint?: "admin" | "tenant" }) {
  return (
    <div className="w-full space-y-10">
      <div>
        <h3 className="text-[20px] font-semibold text-[var(--text-title)] mb-2">2/ 步骤指引，带呼吸灯步骤性指引型弹窗</h3>
        <code className="text-[12px] font-mono text-gray-400 inline-block mb-2">{`<GuidePointBubble variant="dark" showSteps currentStep={n} totalSteps={N} endpoint="${endpoint}" />`}</code>
        <p className="text-[14px] text-[var(--text-secondary)] mb-6">纯文本 + 按钮 （跨功能页面 – {endpoint === "admin" ? "管控端 4px 方角" : "用户端全圆角"}）</p>
      </div>

      {/* 支持不同方向 */}
      <TenantPointBubbleRow
        rowLabel="支持不同方向"
        render={(p) => <InteractiveStepBubble placement={p} endpoint={endpoint} />}
      />

      {/* 支持有序列表 */}
      <TenantPointBubbleRow
        rowLabel="支持有序列表"
        rowHint="有序列表不超过3条"
        render={(p) => <InteractiveStepBubble placement={p} listItems={PB_LIST} endpoint={endpoint} />}
      />

      {/* 支持矩形标注指引 */}
      <TenantPointBubbleRow
        rowLabel="支持矩形标注指引"
        rowHint="呼吸灯替换为蓝色圆角矩形"
        render={(p) => <InteractiveStepBubble placement={p} hotspotShape="rect" hotspotSize={{ width: 60, height: 24 }} endpoint={endpoint} />}
      />
    </div>
  );
}

/** 全局弹窗变体展示：多条+单条 左右并排，通过 endpoint 区分端 */
/** 非阻断浮窗 — 用户端变体展示：single + multi 左右均分 */
function ModuleFloatTenantVariants() {
  return (
    <div className="w-full grid grid-cols-2 gap-0">
      <div className="flex justify-center">
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-gray-700">变体1: 单条内容</h4>
          <code className="text-[12px] font-mono text-gray-400 inline-block">{`<GuideModuleFloat variant="single" />`}</code>
          <div className="[&_.fixed]:!relative [&_.fixed]:!inset-auto [&_.fixed]:!z-auto">
            <GuideModuleFloat open onClose={() => {}} moduleName="技能库" description="技能库模块有以下更新：" items={DEMO_MODULE_ITEMS} variant="single" />
          </div>
        </div>
      </div>
      <div className="flex justify-center">
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-gray-700">变体2: 多条内容</h4>
          <code className="text-[12px] font-mono text-gray-400 inline-block">{`<GuideModuleFloat variant="multi" />`}</code>
          <div className="[&_.fixed]:!relative [&_.fixed]:!inset-auto [&_.fixed]:!z-auto">
            <GuideModuleFloat open onClose={() => {}} moduleName="技能库" description="技能库模块有以下更新：" items={DEMO_MODULE_ITEMS} variant="multi" />
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * 管控端通知卡片（AdminNotifyCard / AdminNotifyStack）已提取为正式全局组件，
 * 见 @/components/onboarding/GuideAdminNotify。本页直接引用，下面仅保留演示数据。
 */
const ADMIN_STACK_DEMO: AdminNotifyItem[] = [
  { title: "管控端 | 新增 MCP 凭据托管功能", desc: "支持快捷配置 Hermes 模型、通道和 Skill，并提供角色设定和初始技能包等能力", skinIndex: 0 },
  { title: "用户端 | Hermes支持对话视图", desc: "用户端 Hermes 类型的 Agent 支持开启对话视图功能。", skinIndex: 1 },
  { title: "管控端 | Tokens 统计优化", desc: "Tokens 统计支持按时间维度设置，记忆管理功能同步上线。", skinIndex: 2 },
];

/** 非阻断浮窗 — 管控端变体展示（严格对齐 Figma node 4000-27807） */
function ModuleFloatAdminVariants() {
  return (
    <div className="w-full space-y-10">
      <div className="w-full grid grid-cols-3 gap-0">
        {/* 第一列：变体1 + 变体2 */}
        <div className="flex justify-center">
          <div className="space-y-6">
            <div className="space-y-3">
              <h4 className="text-sm font-semibold text-gray-700">变体1: 管控端单个卡片</h4>
              <code className="text-[12px] font-mono text-gray-400 inline-block">{`<AdminNotifyCard type="single" />`}</code>
              <AdminNotifyCard title="管控端 | 新增 MCP 凭据托管功能" desc="支持快捷配置 Hermes 模型、通道和 Skill，并提供角色设定和初始技能包等能力" skinIndex={0} />
            </div>
            <div className="space-y-3">
              <h4 className="text-sm font-semibold text-gray-700">变体2: 管控端聚合卡片</h4>
              <code className="text-[12px] font-mono text-gray-400 inline-block">{`<AdminNotifyCard type="aggregate" />`}</code>
              <AdminNotifyCard title="管控端有 3 项新增" desc="新增 MCP 凭据托管功能、Tokens 统计支持时间维度设置、记忆分…" btnText="查看详情" skinIndex={0} />
            </div>
          </div>
        </div>

        {/* 第二列：变体3 + 变体4 */}
        <div className="flex justify-center">
          <div className="space-y-6">
            <div className="space-y-3">
              <h4 className="text-sm font-semibold text-gray-700">变体3: 用户端单个卡片</h4>
              <code className="text-[12px] font-mono text-gray-400 inline-block">{`<AdminNotifyCard type="single" endpoint="tenant" />`}</code>
              <AdminNotifyCard title="用户端 | Hermes支持对话视图" desc="用户端 Hermes 类型的 Agent 支持开启对话视图功能。" skinIndex={1} />
            </div>
            <div className="space-y-3">
              <h4 className="text-sm font-semibold text-gray-700">变体4: 用户端聚合卡片</h4>
              <code className="text-[12px] font-mono text-gray-400 inline-block">{`<AdminNotifyCard type="aggregate" endpoint="tenant" />`}</code>
              <AdminNotifyCard title="用户端有 3 项新增" desc="内置大模型支持多模态、记忆管理功能上线、模型支持设为默认…" btnText="查看详情" skinIndex={1} />
            </div>
          </div>
        </div>

        {/* 第三列：变体5 多条叠加（可交互） */}
        <div className="flex justify-center">
          <div className="space-y-3">
            <h4 className="text-sm font-semibold text-gray-700">变体5: 多条叠加展示</h4>
              <code className="text-[12px] font-mono text-gray-400 inline-block">{`<AdminNotifyStack cards={...} />`}</code>
            <AdminNotifyStack cards={ADMIN_STACK_DEMO} />
          </div>
        </div>
      </div>


    </div>
  );
}

/** 全局弹窗变体展示：多条+单条 左右并排，1:1 原尺寸 680×512，共用一个蒙版背景 */
function GlobalModalVariantsRow({ endpoint }: { endpoint: "tenant" | "admin" }) {
  const carouselSlides = endpoint === "tenant" ? TENANT_CAROUSEL_SLIDES : ADMIN_SLIDES;
  const singleSlides = endpoint === "tenant" ? TENANT_SLIDES : ADMIN_SINGLE_SLIDES;

  return (
    <div className="w-full space-y-8">
      {/* 隐藏弹窗自身的遮罩 */}
      <style>{`
        .gm-stage .absolute.inset-0[class*="bg-black"] { background: transparent !important; }
      `}</style>

      {/* 多条内容 */}
      <div>
        <div className="mb-3">
          <h4 className="text-sm font-semibold text-gray-700">多条内容</h4>
          <code className="text-[12px] font-mono text-gray-400 inline-block mt-1">{`<GuideGlobalModal variant="carousel" endpoint="${endpoint}" />`}</code>
        </div>
        <div
          className="relative rounded-xl overflow-hidden flex justify-center items-center"
          style={{ padding: 40, background: "#1a1a2e" }}
        >
          <div className="gm-stage relative shrink-0 [&_.fixed]:!absolute rounded-lg shadow-lg" style={{ width: 680, height: 512 }}>
            <GuideGlobalModal open onClose={() => {}} variant="carousel" slides={carouselSlides} confirmText="立即体验" endpoint={endpoint} />
          </div>
        </div>
      </div>

      {/* 单条内容 */}
      <div>
        <div className="mb-3">
          <h4 className="text-sm font-semibold text-gray-700">单条内容</h4>
          <code className="text-[12px] font-mono text-gray-400 inline-block mt-1">{`<GuideGlobalModal variant="single" endpoint="${endpoint}" />`}</code>
        </div>
        <div
          className="relative rounded-xl overflow-hidden flex justify-center items-center"
          style={{ padding: 40, background: "#1a1a2e" }}
        >
          <div className="gm-stage relative shrink-0 [&_.fixed]:!absolute rounded-lg shadow-lg" style={{ width: 680, height: 512 }}>
            <GuideGlobalModal open onClose={() => {}} variant="single" slides={singleSlides} confirmText="立即体验" secondaryText="次级按钮" endpoint={endpoint} />
          </div>
        </div>
      </div>
    </div>
  );
}

/** New Tag 演示：模拟管控端侧边栏导航中的 New 标签效果 */
function NewTagDemo() {
  const demoItems = [
    { icon: "/assets/admin-sidebar/skill-config.svg", label: "技能配置", hasNew: true },
    { icon: "/assets/admin-sidebar/agent-tool-library.svg", label: "Agent 工具库", hasNew: true },
    { icon: "/assets/admin-sidebar/ops-observation.svg", label: "运维观测", hasNew: true },
    { icon: "/assets/admin-sidebar/cloud-dev.svg", label: "云开发管理", hasNew: true },
    { icon: "/assets/admin-sidebar/ai-agent-security.svg", label: "AI Agent 安全", hasNew: true },
    { icon: "/assets/admin-sidebar/session-management.svg", label: "会话管理", hasNew: true },
  ];

  return (
    <div className="w-full flex justify-center">
      <div className="bg-white rounded-xl border border-[#EAEEF4] overflow-hidden" style={{ width: 240 }}>
        <div className="px-4 pt-4 pb-2">
          <p className="text-xs text-[#737373] mb-1 px-2">当前带 New Tag 的导航项</p>
        </div>
        <div className="px-4 pb-4 space-y-0.5">
          {demoItems.map((item) => (
            <div
              key={item.label}
              className="flex items-center gap-2 px-2 rounded-[4px] text-sm text-[#0A0A0A]"
              style={{ height: 34 }}
            >
              <img src={item.icon} alt="" className="size-4 shrink-0" />
              <span className="min-w-0 flex-1 truncate">{item.label}</span>
              {item.hasNew && <GuideNewTag variant="new" className="ml-auto" />}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function getStage(info: ComponentInfo) {
  switch (info.id) {
    case "global-modal-tenant":
      return <GlobalModalVariantsRow endpoint="tenant" />;
    case "global-modal-admin":
      return <GlobalModalVariantsRow endpoint="admin" />;
    case "new-tag-admin":
      return <NewTagDemo />;
    case "module-float-tenant":
      return <ModuleFloatTenantVariants />;
    case "module-float-admin":
      return <ModuleFloatAdminVariants />;
    case "nav-bubble-admin":
      return (
        <InlineStage height={info.stageHeight}>
          <GuideNavBubble open onClose={() => {}} title="运营观测" description="全新数据看板，实时掌握运营数据趋势与异常" href="/admin/ops-observation" actionText="去看看" placement="left" />
        </InlineStage>
      );
    case "point-bubble-tenant":
      return <div className="w-full py-2"><TenantPointBubbleGallery /></div>;
    case "point-bubble-admin":
      return <div className="w-full py-2"><TenantPointBubbleGallery endpoint="admin" /></div>;
    case "highlight-bubble-tenant":
      return <div className="w-full py-2"><StepGuideBubbleGallery /></div>;
    case "highlight-bubble-admin":
      return <div className="w-full py-2"><StepGuideBubbleGallery endpoint="admin" /></div>;
    case "update-bar-admin":
      return <div className="w-full flex items-center justify-center"><InlineUpdateBar /></div>;
    case "changelog-drawer-admin":
      return <InlineStage height={info.stageHeight}><InlineChangelogDrawer versions={DEMO_CHANGELOG} /></InlineStage>;
    default:
      return null;
  }
}

// ─── 组件分组（统一按组件类型聚合，卡片内 Tab 切换用户端/管控端） ─────────

interface ComponentGroup {
  /** 锚点 id */
  anchor: string;
  /** 导航名称 */
  navLabel: string;
  /** 用户端变体信息（可选） */
  tenant?: ComponentInfo;
  /** 管控端变体信息（可选） */
  admin?: ComponentInfo;
}


const COMPONENT_GROUPS: ComponentGroup[] = [
  {
    anchor: "global-modal",
    navLabel: "全局弹窗",
    tenant: TENANT_COMPONENTS.find((c) => c.id === "global-modal-tenant"),
    admin: ADMIN_COMPONENTS.find((c) => c.id === "global-modal-admin"),
  },
  {
    anchor: "module-float",
    navLabel: "非阻断浮窗",
    tenant: TENANT_COMPONENTS.find((c) => c.id === "module-float-tenant"),
    admin: ADMIN_COMPONENTS.find((c) => c.id === "module-float-admin"),
  },
  {
    anchor: "point-bubble",
    navLabel: "单UI提示气泡",
    tenant: TENANT_COMPONENTS.find((c) => c.id === "point-bubble-tenant"),
    admin: ADMIN_COMPONENTS.find((c) => c.id === "point-bubble-admin"),
  },
  {
    anchor: "highlight-bubble",
    navLabel: "步骤指引气泡",
    tenant: TENANT_COMPONENTS.find((c) => c.id === "highlight-bubble-tenant"),
    admin: ADMIN_COMPONENTS.find((c) => c.id === "highlight-bubble-admin"),
  },
  {
    anchor: "new-tag",
    navLabel: "New Tag",
    admin: {
      id: "new-tag-admin",
      code: "<GuideNewTag variant=\"new | coming-soon | custom\" />",
      title: "New Tag GuideNewTag",
      desc: "管控端侧边栏导航项的标签。变体：① variant=\"new\"（淡蓝底+品牌蓝文字 \"New\"） ② variant=\"coming-soon\"（淡橙底+橙字 \"即将开放\"） ③ variant=\"custom\"（灰底灰字+自定义文案）",
      badge: "admin",
      stageHeight: 0,
      constraints: [
        { label: "组件", value: "GuideNewTag（variant=\"new\"）" },
        { label: "高度", value: "18px，rounded-full 胶囊形" },
        { label: "内边距", value: "px-1.5（左右各 6px）" },
        { label: "字号", value: "11px，font-normal，tracking 0.015em" },
        { label: "文案", value: "固定 \"New\"（英文）" },
        { label: "背景", value: "color-mix(in srgb, #1447E6 10%, #FFFFFF) ≈ 淡蓝色" },
        { label: "文字色", value: "#1447E6（品牌蓝）" },
        { label: "位置", value: "ml-auto 推到菜单行最右侧" },
        { label: "折叠态", value: "侧边栏折叠时不显示" },
        { label: "TTL", value: "含过期机制，上线一段时间后自动隐藏" },
      ],
      scenes: [
        { tag: "元素层 2.5", text: "New Tag — 最轻量的感知手段" },
        { tag: "元素层 2.6", text: "细节优化叠加 — 多项小优化的汇总入口" },
      ],
    },
  },
];

// ─── 主组件 ──────────────────────────────────────────────────

export default function OnboardingGuidePreview() {
  const [activeAnchor, setActiveAnchor] = useState(COMPONENT_GROUPS[0].anchor);

  // 滚动监听：高亮当前可见的锚点
  const handleScroll = useCallback(() => {
    for (const group of COMPONENT_GROUPS) {
      const el = document.getElementById(group.anchor);
      if (el) {
        const rect = el.getBoundingClientRect();
        if (rect.top <= 120 && rect.bottom > 120) {
          setActiveAnchor(group.anchor);
          break;
        }
      }
    }
  }, []);

  useEffect(() => {
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, [handleScroll]);

  const scrollTo = (anchor: string) => {
    const el = document.getElementById(anchor);
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };

  return (
    <div className="min-h-screen bg-[#f8f9fc]" style={{ fontFamily: '-apple-system, BlinkMacSystemFont, "PingFang SC", "Segoe UI", sans-serif' }}>
      <header className="bg-[#0A0A0A] px-10 py-12 text-white">
        <h1 className="text-[28px] font-bold mb-2">引导组件规范 onboarding-guide</h1>
        <p className="text-sm opacity-70 max-w-[680px]">
          ClawPro 版本更新感知引导系统，包含 7 种原子组件。部分组件两端通用（卡片内 Tab 切换），部分仅限管控端。
          预览区直接内嵌真实组件，样式与实际弹出效果完全一致。
        </p>

        <p className="text-sm opacity-50 mt-4">交互：sarrygeng，视觉：brennali</p>
      </header>

      <div className="flex">
        {/* 左侧锚点导航 */}
        <nav className="sticky top-0 h-screen w-[200px] shrink-0 border-r border-gray-200 bg-white py-6 px-4 overflow-y-auto">
          <p className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 mb-3 px-2">组件导航</p>
          <ul className="space-y-0.5">
            {COMPONENT_GROUPS.map((g) => (
              <li key={g.anchor}>
                <button
                  onClick={() => scrollTo(g.anchor)}
                  className={`w-full text-left px-3 py-2 rounded-md text-[13px] transition-all ${
                    activeAnchor === g.anchor
                      ? "bg-[#EEF2FF] text-[#1447E6] font-medium"
                      : "text-gray-600 hover:bg-gray-50 hover:text-gray-900"
                  }`}
                >
                  {g.navLabel}
                </button>
              </li>
            ))}
          </ul>
        </nav>

        {/* 右侧内容区 */}
        <main className="flex-1 min-w-0 px-8 py-10 space-y-10">
          {COMPONENT_GROUPS.map((group) => (
            <section key={group.anchor} id={group.anchor} className="scroll-mt-4">
              <GroupCard group={group} />
            </section>
          ))}
        </main>
      </div>

    </div>
  );
}

// ─── 子组件：组件分组卡片（卡片内 Tab 切换用户端/管控端） ──────────

/** 场景示例占位图（模拟一个完整页面中该组件的使用场景） */
function SceneExamplePlaceholder({ componentName }: { componentName: string }) {
  return (
    <div className="px-8 py-10 flex items-center justify-center">
      <div className="w-full max-w-[900px] border border-dashed border-gray-300 rounded-xl bg-gray-50/50 p-10">
        <div className="flex gap-6">
          {/* 模拟侧边栏 */}
          <div className="w-[180px] shrink-0 bg-white rounded-lg border border-gray-200 p-3 space-y-2">
            <div className="h-2 w-20 bg-gray-200 rounded" />
            <div className="h-2 w-16 bg-gray-100 rounded" />
            <div className="h-2 w-24 bg-gray-100 rounded" />
            <div className="h-2 w-14 bg-gray-100 rounded" />
            <div className="h-2 w-20 bg-gray-100 rounded" />
          </div>
          {/* 模拟主内容区 */}
          <div className="flex-1 bg-white rounded-lg border border-gray-200 p-4 space-y-3">
            <div className="h-3 w-32 bg-gray-200 rounded" />
            <div className="h-2 w-full bg-gray-100 rounded" />
            <div className="h-2 w-3/4 bg-gray-100 rounded" />
            <div className="h-20 w-full bg-gray-50 rounded border border-gray-100 flex items-center justify-center">
              <span className="text-xs text-gray-400">{componentName} 在此处触发</span>
            </div>
            <div className="h-2 w-1/2 bg-gray-100 rounded" />
          </div>
        </div>
      </div>
    </div>
  );
}

function GroupCard({ group }: { group: ComponentGroup }) {
  const hasBoth = !!(group.tenant && group.admin);
  const defaultTab = group.tenant ? "tenant" : "admin";
  const [tab, setTab] = useState<"tenant" | "admin">(defaultTab);
  const [contentTab, setContentTab] = useState<"prototype" | "scene" | "style">("prototype");

  const info = tab === "tenant" ? group.tenant : group.admin;
  if (!info) return null;

  const badgeStyle = {
    both: "bg-[#DBEAFE] text-[#1E40AF]",
    admin: "bg-[#FEE2E2] text-[#991B1B]",
    tenant: "bg-[#D1FAE5] text-[#065F46]",
  };
  const badgeLabel = { both: "两端通用", admin: "仅管控端", tenant: "仅用户端" };

  const contentTabs = [
    { key: "prototype" as const, label: "原型与适用场景" },
    { key: "style" as const, label: "样式约束" },
  ];

  return (
    <div className="bg-white border border-gray-200 rounded-2xl overflow-hidden shadow-[0_1px_3px_rgba(0,0,0,0.04)]">
      {/* 卡片头部 */}
      <div className="px-8 pt-6 pb-0 flex items-start gap-4">
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-gray-900 mb-1">{info.title}</h3>
          <p className="text-[13px] text-gray-500">{info.desc}</p>
          {info.charLimit && (
            <p className="text-[13px] text-gray-500 mt-0.5">{info.charLimit}</p>
          )}
        </div>

        {/* 右上角：Tab 切换 或 Badge */}
        {hasBoth ? (
          <SegmentGroup className="shrink-0">
            <SegmentOption active={tab === "tenant"} onClick={() => setTab("tenant")}>
              用户端
            </SegmentOption>
            <SegmentOption active={tab === "admin"} onClick={() => setTab("admin")}>
              管控端
            </SegmentOption>
          </SegmentGroup>
        ) : (
          <span className={`inline-flex items-center px-2.5 py-1 text-[11px] font-semibold rounded-md mt-1 shrink-0 ${badgeStyle[info.badge]}`}>
            {badgeLabel[info.badge]}
          </span>
        )}
      </div>

      {/* 警示提示 — 仅全局弹窗卡片显示 */}
      {group.anchor === "global-modal" && (
        <div className="px-8 mt-4">
          <Alert variant="warning">
            <AlertInfoIcon className="h-4 w-4" />
            <AlertDescription>新增全局弹窗请务必联系设计同学审核弹窗样式</AlertDescription>
          </Alert>
        </div>
      )}

      {/* 内容 Tab 栏 */}
      <div className="px-8 flex border-b border-gray-200 mt-4">
        {contentTabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setContentTab(t.key)}
            className={`px-4 py-2.5 text-[13px] font-medium border-b-2 transition-all ${
              contentTab === t.key
                ? "text-[#1447E6] border-[#1447E6]"
                : "text-gray-500 border-transparent hover:text-gray-700"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab 内容区 */}
      {contentTab === "prototype" && (
        <>
          {/* 原型预览区 */}
          <div className="px-8 py-6 bg-gray-100 border-b border-gray-100 overflow-x-auto">
            {getStage(info)}
          </div>
          {/* 适用场景 */}
          <div className="px-8 py-6">
            <h4 className="text-[11px] font-semibold uppercase tracking-wider text-gray-400 mb-3">适用场景</h4>
            <ul className="space-y-1.5">
              {info.scenes.map((s, i) => (
                <li key={i} className="flex items-start gap-2 text-[13px] text-gray-700">
                  <span className="inline-flex px-1.5 py-0.5 text-[10px] font-medium rounded bg-[#F0F0FF] text-[#5B21B6] shrink-0 mt-px">{s.tag}</span>
                  <span>{s.text}</span>
                </li>
              ))}
            </ul>
          </div>
        </>
      )}

      {contentTab === "style" && (
        <div className="px-8 py-6">
          <ul className="space-y-0">
            {info.constraints.map((c, i) => (
              <li key={i} className="flex items-start gap-2 py-1.5 border-b border-gray-50 last:border-b-0 text-[13px] text-gray-700">
                <span className="text-[11px] font-medium text-gray-400 min-w-[56px] pt-px shrink-0">{c.label}</span>
                <span>{c.value}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
