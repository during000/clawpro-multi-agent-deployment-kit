/**
 * AgentChat - Figma「Lighthouse 产品需求 / page-首页-已连接」(节点 1003:22598) 1:1 还原
 * Design: 「流动蓝图」Fluid Blueprint v2
 *
 * ⚠️ 注意：本页面**故意不遵守** SKILL v2 §4「圆角 ≤ 4px」等约束，按 Figma 设计稿真实
 * 数值还原（应用卡 20px / Tab wrapper 16px / 输入卡 20px / 历史项 8px ……），
 * 用作设计还原稿。如要全站推行，需先升级 SKILL.md 的圆角规范。
 *
 * Figma 关键数值（已逐一比对）：
 *   - 整页底色 #F7F8FB；侧栏 / 主面板 #FFFFFF；分隔线 #E9ECF1
 *   - 应用卡 1200×768，圆角 20，shadow xs (0 1 4 rgba(0,0,0,0.05))
 *   - 侧栏 228 宽；分隔竖线 1px 高 894（用 right border 模拟）
 *   - Hermes / Beta 徽章：圆角 4，padding 0 5，高 16，字号 12/18 #757575
 *   - Segment Tab：wrapper 16 圆角 #F5F6F9 padding 4；激活 item 16 圆角 白底 +
 *     tea/shadow-xs；item padding 2 16；激活字 12/20 medium，未激活 12/20 regular
 *   - Agents 头像：40×40 圆，已选中：白底 + #0052D9 1px ring + tea/shadow-xs +
 *     conic 红色光晕 (255,79,79)；未选 #000 12% 透明圆底
 *   - 历史对话项：圆角 8，padding 8 12，文字 14/20 rgba(10,10,10,0.8)
 *   - 「前往小程序」按钮：196×36 圆角 8 white
 *   - 输入卡片：圆角 20，stroke #E9ECF1，shadow 0 4 12 rgba(0,0,0,0.04)
 *   - Deepseek pill：圆角 16，stroke #E6E9EF，padding 2 12
 *   - 指令库 pill：圆角 20，stroke #E9ECF1，padding 6 12
 *   - 快捷指令 chip：圆角 8，stroke #E9ECF1，padding 10 20，gap 16
 *   - Star 装饰渐变：linear-gradient(112deg, #E3453D 7%, #7D2621 70%)，10×11
 */
import { useState, useRef, useEffect, useMemo } from "react";
import { useLocation } from "wouter";
import {
  Pencil,
  ChevronDown,
  ChevronUp,
  Maximize2,
  Minimize2,
  MessageSquarePlus,
  Plus,
  ArrowUp,
  ArrowRight,
  Cloud,
  Check,
} from "lucide-react";
import CloudDevEnvDetailDialog, { type Env } from "@/components/admin/CloudDevEnvDetailDialog";
import CloudDevCreateEnvDialog, { type CreateEnvForm } from "@/components/admin/CloudDevCreateEnvDialog";
import { Popover, PopoverContent, PopoverAnchor } from "@/components/ui/popover";
import { toast } from "sonner";

/* ───────────── Mock 数据 ───────────── */

interface AgentItem {
  key: string;
  name: string;
  /** 角色虾头像图片路径（来自 /public/assets/avatars/） */
  avatar: string;
}

/**
 * Agent 头像映射（对照 Figma 节点 1387-19352）
 *
 * 角色分组（参考 Figma 设计：8 个角色分为 TOP（默认显示）+ BOTTOM（点"展开更多"显示））
 *
 * TOP（默认显示 5 个 + 第 6 格"展开/收起"按钮）：
 *   初始角色 / 程序员 / 设计师 / 理财助理 / 美食家
 * BOTTOM（点"展开更多"显示）：
 *   运营 / 程序员（云开发）
 *
 * 角色差异：
 * - 「程序员」(key=dev)：通用编程助手；带"云开发管理"入口按钮，浮层提示已绑定环境（mock 常量）。
 * - 「程序员（云开发）」(key=dev-cloud)：专门用于云开发场景；初始未绑定，
 *   进入后引导用户走完整的「创建 → 自动绑定 / 选择已有环境 → 换绑」闭环。
 */
const AGENT_GROUP_TOP: AgentItem[] = [
  { key: "default", name: "初始角色", avatar: "/assets/avatars/avatar-default.png" },
  { key: "dev", name: "程序员", avatar: "/assets/avatars/avatar-developer.png" },
  { key: "designer", name: "设计师", avatar: "/assets/avatars/avatar-designer.png" },
  { key: "finance", name: "理财助理", avatar: "/assets/avatars/avatar-analyst.png" },
  { key: "food", name: "美食家", avatar: "/assets/avatars/avatar-creator.png" },
];
const AGENT_GROUP_BOTTOM: AgentItem[] = [
  { key: "operator", name: "运营", avatar: "/assets/avatars/avatar-operator.png" },
  { key: "dev-cloud", name: "程序员（云开发）", avatar: "/assets/avatars/avatar-developer.png" },
];

/**
 * 实例切换 mock 数据
 *
 * 引擎 (engine):
 *   - OpenClaw: 完整对话视图 + 编排 + 工具调用，全功能
 *   - Hermes:   仅支持任务流（无对话视图）→ 在对话页 hover 时提示「Hermes 暂不支持对话视图」并 disabled
 *   - ACE:      执行引擎，有自身视图（这里也支持选中）
 *
 * 状态 (online):
 *   - true  → 实心绿点 #088F50
 *   - false → 空心灰圈 1px stroke #BFC4CC
 */
type InstanceEngine = "OpenClaw" | "Hermes" | "ACE";

interface InstanceItem {
  key: string;
  name: string;
  engine: InstanceEngine;
  online: boolean;
}

const INSTANCE_LIST: InstanceItem[] = [
  { key: "abc", name: "实例名称 AB...", engine: "Hermes", online: true },
  { key: "123", name: "实例名称 123", engine: "OpenClaw", online: true },
  { key: "liam-syd", name: "Liam悉尼", engine: "Hermes", online: false },
  { key: "liam-tyo", name: "Liam东京", engine: "ACE", online: false },
  { key: "liam-lon", name: "Liam伦敦", engine: "Hermes", online: false },
  { key: "ava-ny", name: "Ava纽约", engine: "OpenClaw", online: false },
];

interface HistoryItem {
  key: string;
  title: string;
  /** 已归档（置灰） */
  archived?: boolean;
}

const HISTORY_ITEMS: HistoryItem[] = [
  { key: "h1", title: "最近一小时的CPU使..." },
  { key: "h2", title: "查询可观测平台的告..." },
  { key: "h3", title: "绑定策略对象" },
  { key: "h4", title: "如何做小龙虾" },
  { key: "h5", title: "如何做麻辣小龙虾" },
  { key: "h6", title: "设计云产品 AI 助手", archived: true },
];

const QUICK_COMMANDS = [
  "帮我购买一个域名和一台服务器",
  "帮我开通 Lighthouse 实例防火墙",
  "帮我发送每日周报邮件",
  "帮我查询股票今日行情",
];

/* ───────────── 子组件 ───────────── */

/** 红色渐变小星形（Figma fill_5X9YJJ：linear-gradient(112deg, #E3453D 7%, #7D2621 70%)，10×11） */
function StarBullet() {
  return (
    <svg
      aria-hidden
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="flex-shrink-0"
    >
      <path
        d="M6.55469 10.4499C6.49219 10.2077 6.36593 9.98662 6.18903 9.80972C6.01213 9.63282 5.79106 9.50656 5.54881 9.44406L1.25443 8.33669C1.18116 8.3159 1.11668 8.27177 1.07076 8.21101C1.02484 8.15024 1 8.07616 1 8C1 7.92384 1.02484 7.84976 1.07076 7.78899C1.11668 7.72823 1.18116 7.6841 1.25443 7.66331L5.54881 6.55524C5.79097 6.4928 6.01198 6.36664 6.18888 6.18987C6.36577 6.0131 6.49208 5.79218 6.55469 5.55006L7.66206 1.25567C7.68264 1.18211 7.72672 1.11731 7.78758 1.07115C7.84843 1.02499 7.92272 1 7.9991 1C8.07548 1 8.14976 1.02499 8.21062 1.07115C8.27147 1.11731 8.31556 1.18211 8.33614 1.25567L9.44281 5.55006C9.5053 5.7923 9.63157 6.01338 9.80847 6.19028C9.98537 6.36718 10.2064 6.49344 10.4487 6.55594L14.7431 7.66261C14.8169 7.68298 14.882 7.72701 14.9285 7.78796C14.9749 7.8489 15 7.92339 15 8C15 8.0766 14.9749 8.15109 14.9285 8.21204C14.882 8.27299 14.8169 8.31702 14.7431 8.33739L10.4487 9.44406C10.2064 9.50656 9.98537 9.63282 9.80847 9.80972C9.63157 9.98662 9.5053 10.2077 9.44281 10.4499L8.33544 14.7443C8.31486 14.8179 8.27077 14.8827 8.20992 14.9289C8.14907 14.975 8.07478 15 7.9984 15C7.92202 15 7.84773 14.975 7.78688 14.9289C7.72603 14.8827 7.68194 14.8179 7.66136 14.7443L6.55469 10.4499Z"
        fill="url(#paint0_linear_star_bullet)"
        fillOpacity="0.9"
      />
      <defs>
        <linearGradient id="paint0_linear_star_bullet" x1="8" y1="1" x2="8" y2="15" gradientUnits="userSpaceOnUse">
          <stop stopColor="#050505" />
          <stop offset="1" stopColor="#1447E6" />
        </linearGradient>
      </defs>
    </svg>
  );
}

/** Agent 头像（48×48 圆形，对齐「我的 Agent」页面尺寸；选中态：白底 + #0052D9 1px ring + tea/shadow-xs） */
function AgentAvatar({ item, selected }: { item: AgentItem; selected: boolean }) {
  return (
    <div className="relative flex h-12 w-12 items-center justify-center">
      {/* 圆形容器：选中=白底 + 1px 蓝边 + 阴影；未选=透明（直接显示 avatar 自带渐变背景） */}
      <span
        className="relative flex h-12 w-12 items-center justify-center rounded-full overflow-hidden"
        style={
          selected
            ? {
                background: "#FFFFFF",
                border: "1px solid #0052D9",
                boxShadow: "0 1px 4px 0 rgba(0,0,0,0.05)",
              }
            : undefined
        }
        // allow-shadow: Figma tea/shadow-xs (0 1 4 rgba(0,0,0,0.05))，仅本设计还原页使用
      >
        {/* 角色头像图片（来自 /public/assets/avatars/） */}
        <img
          src={item.avatar}
          alt={item.name}
          draggable={false}
          className="h-12 w-12 object-cover rounded-full pointer-events-none select-none"
        />
      </span>
    </div>
  );
}

/** 引擎徽章（OpenClaw / Hermes / ACE）：与 Hermes 徽章同款 4px 圆角 stroke */
function EngineBadge({ engine }: { engine: InstanceEngine }) {
  return (
    <span
      className="inline-flex items-center justify-center rounded-[4px] border flex-shrink-0"
      style={{
        height: 16,
        padding: "0 5px",
        borderColor: "#D6DBE3",
        fontSize: 12,
        lineHeight: "18px",
        color: "rgba(0,0,0,0.5)",
      }}
    >
      {engine}
    </span>
  );
}

/* ───────────── 主组件 ───────────── */

interface AgentChatProps {
  /** 嵌入「我的 Agent」内容区时为 true：去掉外层画布与尺寸标记，宽度自适应。 */
  embedded?: boolean;
}

export default function AgentChat({ embedded = false }: AgentChatProps) {
  const [, setLocation] = useLocation();
  const [activeAgentKey, setActiveAgentKey] = useState<string>("dev");
  const [inputText, setInputText] = useState<string>("");
  const [activeHistoryKey, setActiveHistoryKey] = useState<string>("h2");

  /* ── 云开发环境详情弹窗 ──
   * 数据由两个角色共用：
   *   - dev（程序员，旧）：boundCloudDevEnv 永远是 cloudEnvs[0]（常量绑定），仅展示用
   *   - dev-cloud（程序员（云开发），新）：boundEnvIdCloud 初始 null，演示完整的
   *     「未绑定 → 创建/选择已有 → 自动绑定 → 换绑」业务闭环；新建环境会加入 cloudEnvs 列表 */
  const INITIAL_CLOUD_ENVS: Env[] = useMemo(() => [
    { id: "1", envId: "openclaw-prod-8g2k1a", name: "生产环境", status: "running", region: "ap-guangzhou", packageName: "标准版", storageUsed: "2.3 GB / 10 GB", dbUsed: "156 MB / 2 GB", functionCount: 12, staticHosting: true, createdAt: "2025-12-01", expireAt: "2026-12-01", lastDeployAt: "2026-03-28 14:32:00", appCount: 3, appNames: ["库存管理系统", "CRM 后台", "数据看板"], createdBy: "alice@acompany.com", allowedGroups: ["dept-tech"], allowedUsers: ["alice@acompany.com", "carol@acompany.com"], dbType: "postgresql", overflowBilling: false, autoRenewal: true },
    { id: "3", envId: "openclaw-dev-9h4n3c", name: "开发环境", status: "running", region: "ap-beijing", packageName: "标准版", storageUsed: "0.8 GB / 10 GB", dbUsed: "42 MB / 2 GB", functionCount: 15, staticHosting: false, createdAt: "2026-02-01", expireAt: "2027-02-01", lastDeployAt: "2026-03-29 16:45:00", appCount: 5, appNames: ["模型训练平台", "数据标注工具", "推理服务", "监控面板", "文档站"], createdBy: "alice@acompany.com", allowedGroups: ["dept-fe", "dept-ai"], allowedUsers: ["alice@acompany.com", "eve@acompany.com", "iris@acompany.com"], dbType: "cloud", overflowBilling: false, autoRenewal: true },
    { id: "6", envId: "openclaw-crm-3l7r6f", name: "CRM 系统环境", status: "error", region: "ap-guangzhou", packageName: "标准版", storageUsed: "4.7 GB / 50 GB", dbUsed: "1.2 GB / 5 GB", functionCount: 28, staticHosting: true, createdAt: "2025-11-10", expireAt: "2026-05-10", lastDeployAt: "2026-03-25 08:30:00", appCount: 1, appNames: ["运营数据中心"], createdBy: "david@acompany.com", allowedGroups: ["dept-product", "dept-operation"], allowedUsers: ["david@acompany.com", "henry@acompany.com"], dbType: "postgresql", overflowBilling: false, autoRenewal: true },
    { id: "8", envId: "openclaw-mini-4n9t8h", name: "小程序环境", status: "running", region: "ap-guangzhou", packageName: "标准版", storageUsed: "1.8 GB / 10 GB", dbUsed: "210 MB / 2 GB", functionCount: 9, staticHosting: false, createdAt: "2026-02-10", expireAt: "2027-02-10", lastDeployAt: "2026-03-29 10:10:00", appCount: 1, appNames: ["组件文档站"], createdBy: "eve@acompany.com", allowedGroups: ["dept-fe", "dept-design"], allowedUsers: ["eve@acompany.com", "iris@acompany.com"], dbType: "cloud", overflowBilling: false, autoRenewal: true },
  ], []);
  const [cloudEnvs, setCloudEnvs] = useState<Env[]>(INITIAL_CLOUD_ENVS);
  const [cloudDevEnv, setCloudDevEnv] = useState<Env | null>(null);
  const [cloudDevDialogOpen, setCloudDevDialogOpen] = useState(false);
  /** dev-cloud 角色专属：当前 Agent 已绑定的环境 ID（null = 未绑定，初始演示空态） */
  const [boundEnvIdCloud, setBoundEnvIdCloud] = useState<string | null>(null);
  /** 是否为「程序员（云开发）」新角色——区分完整闭环 vs. 旧 dev 角色的浮层提示 */
  const isCloudDevRole = activeAgentKey === "dev-cloud";
  /** 当前 Agent 已绑定的云开发环境：
   *  - dev：常量绑定 cloudEnvs[0]（旧逻辑保持不变）
   *  - dev-cloud：由 boundEnvIdCloud 查询，可能为 null */
  const boundCloudDevEnv = useMemo(() => {
    if (isCloudDevRole) {
      return cloudEnvs.find((e) => e.id === boundEnvIdCloud) ?? null;
    }
    return cloudEnvs[0] ?? null;
  }, [isCloudDevRole, boundEnvIdCloud, cloudEnvs]);
  /** 云开发管理按钮旁的"已绑定环境"引导浮层：进入页面短暂展示一次，hover 按钮时再次展示 */
  const [cloudDevHintOpen, setCloudDevHintOpen] = useState(false);
  /** 新建云开发环境弹窗（与管控端共用 CloudDevCreateEnvDialog） */
  const [createEnvOpen, setCreateEnvOpen] = useState(false);
  /** Mock：环境总数，用于演示配额校验（用户端通常有"每用户最多 N 个环境"限制） */
  const TENANT_ENV_QUOTA_MAX = 5;
  useEffect(() => {
    // dev：仅在已绑定时浮出"已绑定 xxx"
    // dev-cloud：无论是否已绑定都浮出（未绑定时是引导用户去绑定）
    if (activeAgentKey === "dev") {
      if (!boundCloudDevEnv) return;
    } else if (activeAgentKey !== "dev-cloud") {
      return;
    }
    setCloudDevHintOpen(true);
    const t = window.setTimeout(() => setCloudDevHintOpen(false), 3500);
    return () => window.clearTimeout(t);
  // 仅在切换角色时短暂展示一次
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeAgentKey]);
  const sessionTitle =
    HISTORY_ITEMS.find((h) => h.key === activeHistoryKey)?.title ??
    "查询可观测平台的数据";

  /* 对话流 messages：每条 { id, role, content }
     默认空数组 → 显示 welcome 态；用户发送消息后开始累计 */
  type ChatMessage = { id: string; role: "user" | "assistant"; content: string };
  const [messages, setMessages] = useState<ChatMessage[]>([]);

  /* Header：会话标题切换 popover */
  const [titlePopoverOpen, setTitlePopoverOpen] = useState<boolean>(false);
  const titleTriggerRef = useRef<HTMLDivElement | null>(null);

  /* Header：全屏切换 */
  const [isFullscreen, setIsFullscreen] = useState<boolean>(false);

  /* Header：新建会话二次确认 */
  const [showNewChatConfirm, setShowNewChatConfirm] = useState<boolean>(false);

  /* Header：编辑角色 inline 输入 */
  const [roleEditOpen, setRoleEditOpen] = useState<boolean>(false);
  const [roleDraft, setRoleDraft] = useState<string>("");
  const roleEditRef = useRef<HTMLDivElement | null>(null);
  /* 用户对每个 agent 重命名后的覆盖名（key → name） */
  const [agentNameOverrides, setAgentNameOverrides] = useState<
    Record<string, string>
  >({});
  /* Agents 列表是否展开（默认收起，仅显示 TOP 三个） */
  const [agentsExpanded, setAgentsExpanded] = useState<boolean>(false);

  /* 实例切换：当前实例 + popover 开关 */
  const [activeInstanceKey, setActiveInstanceKey] = useState<string>("abc");
  const [instancePopoverOpen, setInstancePopoverOpen] = useState<boolean>(false);
  const instanceTriggerRef = useRef<HTMLDivElement | null>(null);

  const activeInstance =
    INSTANCE_LIST.find((i) => i.key === activeInstanceKey) ?? INSTANCE_LIST[0];

  /* 点击外部关闭 popover（实例切换 + 会话标题 + 编辑角色） */
  useEffect(() => {
    if (
      !instancePopoverOpen &&
      !titlePopoverOpen &&
      !roleEditOpen
    )
      return;
    const handleClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        instancePopoverOpen &&
        instanceTriggerRef.current &&
        !instanceTriggerRef.current.contains(target)
      ) {
        setInstancePopoverOpen(false);
      }
      if (
        titlePopoverOpen &&
        titleTriggerRef.current &&
        !titleTriggerRef.current.contains(target)
      ) {
        setTitlePopoverOpen(false);
      }
      if (
        roleEditOpen &&
        roleEditRef.current &&
        !roleEditRef.current.contains(target)
      ) {
        setRoleEditOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [
    instancePopoverOpen,
    titlePopoverOpen,
    roleEditOpen,
  ]);

  const activeAgentRaw =
    [...AGENT_GROUP_TOP, ...AGENT_GROUP_BOTTOM].find((a) => a.key === activeAgentKey) ??
    AGENT_GROUP_TOP[1];
  const activeAgent = {
    ...activeAgentRaw,
    name: agentNameOverrides[activeAgentRaw.key] ?? activeAgentRaw.name,
  };

  const handleSend = () => {
    const text = inputText.trim();
    if (!text) return;
    const userMsg: ChatMessage = {
      id: `u-${Date.now()}`,
      role: "user",
      content: text,
    };
    setMessages((prev) => [...prev, userMsg]);
    setInputText("");
    /* 模拟 AI 回复：500ms 后追加一条 assistant 消息 */
    setTimeout(() => {
      const replyMsg: ChatMessage = {
        id: `a-${Date.now()}`,
        role: "assistant",
        content: `收到你的请求：「${text}」。已记录到当前会话，正在为你处理。`,
      };
      setMessages((prev) => [...prev, replyMsg]);
    }, 500);
  };

  /* 对话流自动滚动到底 */
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages.length]);

  /* ───────── 应用卡片本体 ───────── */
  const appCard = (
    <div
      className="relative flex overflow-hidden bg-white border border-[#E9ECF1]"
      style={{
        width: isFullscreen ? "100%" : embedded ? "100%" : 1200,
        height: isFullscreen ? "100%" : embedded ? 720 : 768,
        borderRadius: 20, // Figma 设计稿真实值
        boxShadow: isFullscreen
          ? "0 8px 32px 0 rgba(20,71,230,0.10), 0 2px 8px 0 rgba(0,0,0,0.06)"
          : embedded
            ? "0 1px 4px 0 rgba(0,0,0,0.05)"
            : "0px 24px 60px -12px rgba(20,71,230,0.18), 0px 8px 24px -4px rgba(0,0,0,0.08)",
      }}
      // allow-shadow: Figma tea/shadow-xs（嵌入态）/ 独立预览页设计稿外阴影 / 全屏态适度投影
    >
      {/* ───────── 左侧 Sidebar 228px ───────── */}
      <aside
        className="w-[228px] flex-shrink-0 flex flex-col bg-white"
        style={{ borderRight: "1px solid #E5E7EB" }}
      >
        {/* 实例信息：可点击切换；hover 显示 chevron；点击展开 popover */}
        <div
          ref={instanceTriggerRef}
          className="relative flex flex-col gap-2 px-4 pt-4 pb-3"
        >
          <button
            type="button"
            onClick={() => setInstancePopoverOpen((v) => !v)}
            className="group flex items-center gap-2 w-full text-left"
            style={{ outline: "none" }}
          >
            <span
              className="text-[#0A0A0A] truncate flex-1"
              style={{ fontSize: 16, lineHeight: "24px", fontWeight: 600 }}
            >
              {activeInstance.name}
            </span>
            {/* 引擎徽章 */}
            <EngineBadge engine={activeInstance.engine} />
            {/* 状态点：在线 #088F50；hover 时被 chevron 取代 */}
            <span
              aria-hidden
              className="h-4 w-4 rounded-full flex items-center justify-center flex-shrink-0 group-hover:hidden"
            >
              {activeInstance.online ? (
                <span className="h-2 w-2 rounded-full" style={{ background: "#088F50" }} />
              ) : (
                <span
                  className="h-2 w-2 rounded-full"
                  style={{ border: "1px solid #BFC4CC", background: "transparent" }}
                />
              )}
            </span>
            {/* hover 时出现的 chevron */}
            <ChevronDown
              className="hidden group-hover:block flex-shrink-0"
              size={16}
              style={{
                color: "rgba(0,0,0,0.6)",
                transform: instancePopoverOpen ? "rotate(180deg)" : undefined,
                transition: "transform 0.15s ease",
              }}
            />
          </button>

          {/* Popover：实例列表 */}
          {instancePopoverOpen && (
            <div
              className="absolute left-2 right-2 z-30 bg-white"
              style={{
                top: "calc(100% - 4px)",
                borderRadius: 12,
                border: "1px solid #E9ECF1",
                boxShadow: "0 8px 24px 0 rgba(0,0,0,0.08), 0 1px 4px 0 rgba(0,0,0,0.05)",
                padding: 8,
              }}
              // allow-shadow: Figma 实例切换 popover 浮层阴影
            >
              {INSTANCE_LIST.map((ins) => {
                const isActive = ins.key === activeInstanceKey;
                // Hermes 引擎不支持对话视图：在对话页禁用并显示 tooltip
                const disabled = ins.engine === "Hermes";
                return (
                  <div key={ins.key} className="relative group/item">
                    <button
                      type="button"
                      onClick={() => {
                        if (disabled) return;
                        setActiveInstanceKey(ins.key);
                        setInstancePopoverOpen(false);
                      }}
                      disabled={disabled}
                      className="w-full flex items-center gap-2 transition-colors hover:bg-[#F5F6F9] disabled:cursor-not-allowed"
                      style={{
                        padding: "8px 12px",
                        borderRadius: 8,
                        opacity: disabled ? 0.5 : 1,
                      }}
                    >
                      <span
                        className="truncate flex-1 text-left"
                        style={{
                          fontSize: 14,
                          lineHeight: "22px",
                          color: isActive ? "rgba(10,10,10,0.92)" : "rgba(10,10,10,0.7)",
                          fontWeight: isActive ? 500 : 400,
                        }}
                      >
                        {ins.name}
                      </span>
                      <EngineBadge engine={ins.engine} />
                      <span
                        aria-hidden
                        className="h-4 w-4 rounded-full flex items-center justify-center flex-shrink-0"
                      >
                        {ins.online ? (
                          <span className="h-2 w-2 rounded-full" style={{ background: "#088F50" }} />
                        ) : (
                          <span
                            className="h-2 w-2 rounded-full"
                            style={{ border: "1px solid #BFC4CC", background: "transparent" }}
                          />
                        )}
                      </span>
                    </button>

                    {/* Tooltip：仅 disabled 项 hover 显示 */}
                    {disabled && (
                      <div
                        className="hidden group-hover/item:block absolute z-40 pointer-events-none"
                        style={{
                          left: "100%",
                          top: 0,
                          marginLeft: 8,
                          padding: "6px 10px",
                          background: "#FFFFFF",
                          border: "1px solid #E9ECF1",
                          borderRadius: 8,
                          boxShadow: "0 4px 12px 0 rgba(0,0,0,0.08)",
                          fontSize: 12,
                          lineHeight: "18px",
                          color: "rgba(10,10,10,0.85)",
                          whiteSpace: "nowrap",
                        }}
                        // allow-shadow: tooltip 浮层阴影
                      >
                        {ins.engine} 暂不支持对话视图
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* 主滚动区 */}
        <div
          className="flex-1 overflow-y-auto"
          style={{
            scrollbarWidth: "thin",
            scrollbarColor: "#E5E5E5 transparent",
            padding: "0 16px",
          }}
        >
          {/* Agents 标题区 */}
          <div className="flex items-center" style={{ padding: "9px 12px" }}>
            <span style={{ fontSize: 12, lineHeight: "20px", color: "#ADADAD", flex: 1 }}>
              Agents
            </span>
            <button
              className="h-[14px] w-[14px] flex items-center justify-center text-[#737373] hover:text-[#1447E6] active:scale-90 transition-all"
              aria-label="新增 Agent"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          </div>

          {/* Agents 头像组：3 列网格；默认仅显示 TOP；点击"展开更多"显示 BOTTOM */}
          <div className="grid grid-cols-3 gap-y-2.5">
            {(agentsExpanded
              ? [...AGENT_GROUP_TOP, ...AGENT_GROUP_BOTTOM]
              : AGENT_GROUP_TOP
            ).map((a) => (
              <button
                key={a.key}
                onClick={() => setActiveAgentKey(a.key)}
                className="flex flex-col items-center gap-2 py-2.5 active:scale-95 transition-transform"
              >
                <AgentAvatar item={a} selected={a.key === activeAgentKey} />
                <span
                  style={{
                    fontSize: 12,
                    lineHeight: "16px",
                    letterSpacing: "-0.0125em",
                    color: "rgba(10,10,10,0.7)",
                  }}
                >
                  {a.name}
                </span>
              </button>
            ))}
            {/* 展开 / 收起 切换按钮 */}
            <button
              onClick={() => setAgentsExpanded((v) => !v)}
              className="flex flex-col items-center gap-2 py-2.5 group/more active:scale-95 transition-transform"
              aria-label={agentsExpanded ? "收起" : "展开更多"}
              title={agentsExpanded ? "收起" : "展开更多"}
            >
              <span className="flex h-12 w-12 items-center justify-center rounded-full bg-[rgba(0,0,0,0.04)] group-hover/more:bg-[rgba(0,0,0,0.08)] text-[#737373] group-hover/more:text-[#1447E6] transition-colors">
                {agentsExpanded ? (
                  <ChevronUp className="h-4 w-4" />
                ) : (
                  <ChevronDown className="h-4 w-4" />
                )}
              </span>
              <span
                style={{
                  fontSize: 12,
                  lineHeight: "16px",
                  letterSpacing: "-0.0125em",
                  color: "rgba(10,10,10,0.7)",
                }}
              >
                {agentsExpanded ? "收起" : "展开更多"}
              </span>
            </button>
          </div>

          {/* 历史对话标题 */}
          <div className="flex items-center mt-1" style={{ padding: "9px 12px" }}>
            <span style={{ fontSize: 12, lineHeight: "20px", color: "#ADADAD" }}>历史对话</span>
          </div>

          {/* 历史对话列表：每项圆角 8、padding 8 12、文字 14/20 rgba(10,10,10,0.8) */}
          <div className="flex flex-col">
            {HISTORY_ITEMS.map((h) => {
              const isActive = h.key === activeHistoryKey;
              return (
                <button
                  key={h.key}
                  onClick={() => setActiveHistoryKey(h.key)}
                  className="text-left transition-colors hover:bg-[#F5F6F9]"
                  style={{
                    borderRadius: 8,
                    padding: "8px 12px",
                    background: isActive ? "#F5F6F9" : undefined,
                  }}
                >
                  <span
                    className="block truncate"
                    style={{
                      fontSize: 14,
                      lineHeight: "20px",
                      color: h.archived
                        ? "#A3A3A3"
                        : isActive
                          ? "rgba(10,10,10,0.95)"
                          : "rgba(10,10,10,0.8)",
                      fontWeight: isActive ? 500 : 400,
                    }}
                  >
                    {h.title}
                  </span>
                </button>
              );
            })}
          </div>

          {/* 底部留白渐变（Figma fill_7Z35UL，伪实现：直接给一段 padding） */}
          <div style={{ height: 24 }} />
        </div>

        {/* Sidebar 底部：详细配置 */}
        <div className="flex-shrink-0">
          <div className="px-3 pt-2 pb-4">
            <button
              className="group/config w-full flex items-center justify-between bg-white hover:bg-[#F5F6F9] active:scale-[0.98] transition-all"
              style={{
                height: 36,
                borderRadius: 8,
                padding: "0 12px",
              }}
              onClick={() => setLocation(`/openclaw/${activeInstanceKey}`)}
            >
              <span
                style={{
                  fontSize: 14,
                  lineHeight: "20px",
                  fontWeight: 400,
                  color: "#000",
                }}
              >
                详细配置
              </span>
              <ArrowRight className="h-4 w-4 text-[#737373] group-hover/config:text-[#1447E6] group-hover/config:translate-x-0.5 transition-all" />
            </button>
          </div>
        </div>
      </aside>

      {/* ───────── 主对话面板 ───────── */}
      <main className="flex-1 min-w-0 flex flex-col bg-white">
        {/* Header：高 60，padding 20 16 */}
        <header
          className="flex items-center justify-between flex-shrink-0"
          style={{
            height: 60,
            padding: "0 16px 0 20px",
          }}
        >
          <div className="flex items-center gap-2.5 min-w-0">
            {/* 角色名 + 编辑：点击 ✏ 后原地变 input 编辑（Enter 保存 / Esc 取消） */}
            <div ref={roleEditRef} className="relative flex items-center gap-1 flex-shrink-0">
              {roleEditOpen ? (
                <input
                  autoFocus
                  value={roleDraft}
                  onChange={(e) => setRoleDraft(e.target.value)}
                  onBlur={() => {
                    const next = roleDraft.trim();
                    if (next) {
                      setAgentNameOverrides((m) => ({
                        ...m,
                        [activeAgent.key]: next,
                      }));
                    }
                    setRoleEditOpen(false);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      const next = roleDraft.trim();
                      if (next) {
                        setAgentNameOverrides((m) => ({
                          ...m,
                          [activeAgent.key]: next,
                        }));
                      }
                      setRoleEditOpen(false);
                    }
                    if (e.key === "Escape") {
                      setRoleDraft(activeAgent.name);
                      setRoleEditOpen(false);
                    }
                  }}
                  className="outline-none focus:border-[#1447E6]"
                  style={{
                    height: 24,
                    width: Math.max(64, roleDraft.length * 14 + 24),
                    borderRadius: 6,
                    border: "1px solid #1447E6",
                    padding: "0 6px",
                    fontSize: 14,
                    lineHeight: "20px",
                    color: "#0A0A0A",
                    background: "#FFFFFF",
                  }}
                />
              ) : (
                <>
                  <span
                    style={{
                      fontSize: 14,
                      lineHeight: "20px",
                      color: "#0A0A0A",
                    }}
                  >
                    {activeAgent.name}
                  </span>
                  <button
                    onClick={() => {
                      setRoleDraft(activeAgent.name);
                      setRoleEditOpen(true);
                    }}
                    className="flex items-center justify-center text-[#A3A3A3] hover:text-[#1447E6] active:scale-90 transition-all"
                    style={{ width: 16, height: 16 }}
                    aria-label="编辑角色"
                    title="编辑角色"
                  >
                    <Pencil className="h-3 w-3" />
                  </button>
                </>
              )}
            </div>
            {/* 分隔竖线 */}
            <span aria-hidden className="h-3 w-px" style={{ background: "#DEE1E8" }} />
            {/* 会话标题：可点击展开会话切换列表 */}
            <div ref={titleTriggerRef} className="relative min-w-0">
              <button
                onClick={() => setTitlePopoverOpen((v) => !v)}
                className="group/sess flex items-center gap-1 min-w-0 hover:bg-[#F5F6F9] active:scale-[0.98] transition-all"
                style={{ padding: "4px 8px", borderRadius: 6, marginLeft: -4 }}
                title="切换会话"
              >
                <span
                  className="truncate"
                  style={{
                    fontSize: 14,
                    lineHeight: "20px",
                    color: "rgba(10,10,10,0.8)",
                  }}
                >
                  {sessionTitle}
                </span>
                <ChevronDown
                  className="h-4 w-4 flex-shrink-0 group-hover/sess:text-[rgba(0,0,0,0.7)] transition-all"
                  style={{
                    color: "rgba(0,0,0,0.4)",
                    transform: titlePopoverOpen ? "rotate(180deg)" : undefined,
                  }}
                />
              </button>

              {/* 会话切换 popover */}
              {titlePopoverOpen && (
                <div
                  className="absolute z-30 bg-white"
                  style={{
                    top: "calc(100% + 6px)",
                    left: -4,
                    width: 280,
                    borderRadius: 12,
                    border: "1px solid #E9ECF1",
                    boxShadow:
                      "0 8px 24px 0 rgba(0,0,0,0.08), 0 1px 4px 0 rgba(0,0,0,0.05)",
                    padding: 4,
                  }}
                  // allow-shadow: 会话切换浮层
                >
                  <div
                    style={{
                      padding: "8px 12px 4px",
                      fontSize: 12,
                      lineHeight: "18px",
                      color: "rgba(10,10,10,0.45)",
                    }}
                  >
                    最近会话
                  </div>
                  <div className="flex flex-col">
                    {HISTORY_ITEMS.map((h) => {
                      const isActive = h.key === activeHistoryKey;
                      return (
                        <button
                          key={h.key}
                          onClick={() => {
                            setActiveHistoryKey(h.key);
                            setTitlePopoverOpen(false);
                          }}
                          className="text-left transition-colors hover:bg-[#F5F6F9]"
                          style={{
                            borderRadius: 8,
                            padding: "8px 12px",
                            background: isActive ? "#F5F6F9" : undefined,
                          }}
                        >
                          <span
                            className="block truncate"
                            style={{
                              fontSize: 13,
                              lineHeight: "20px",
                              color: h.archived
                                ? "#A3A3A3"
                                : isActive
                                  ? "rgba(10,10,10,0.95)"
                                  : "rgba(10,10,10,0.8)",
                              fontWeight: isActive ? 500 : 400,
                            }}
                          >
                            {h.title}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          </div>
          {/* 右侧操作区：切换角色 + 新建会话 + 全屏 */}
          <div className="flex items-center gap-3 flex-shrink-0">
            {/* 云开发管理：dev / dev-cloud 角色可见
                - dev（旧）：单态——背景灰色按钮"云开发管理" + 浮层显示已绑定 envId
                - dev-cloud（新）：双态——已绑定时与 dev 相同；未绑定时品牌色虚框 + "绑定云开发环境" +
                  浮层提示"尚未绑定云开发环境，请先创建或绑定" */}
            {(activeAgentKey === "dev" || activeAgentKey === "dev-cloud") && (
              <Popover open={cloudDevHintOpen} onOpenChange={setCloudDevHintOpen}>
                <PopoverAnchor asChild>
                  <button
                    type="button"
                    onClick={() => {
                      // dev-cloud 未绑定：传 env=null，弹窗显示"无可用环境"空态，引导新建
                      // 其他场景：打开详情查看当前已绑定环境
                      if (isCloudDevRole && !boundCloudDevEnv) {
                        setCloudDevEnv(null);
                      } else {
                        setCloudDevEnv(boundCloudDevEnv ?? cloudEnvs[0] ?? null);
                      }
                      setCloudDevDialogOpen(true);
                    }}
                    onMouseEnter={() => setCloudDevHintOpen(true)}
                    onMouseLeave={() => setCloudDevHintOpen(false)}
                    className="inline-flex items-center gap-2 active:scale-[0.97] transition-all"
                    style={{
                      borderRadius: 16,
                      background: boundCloudDevEnv ? "#F2F4F8" : "rgba(20, 71, 230, 0.08)",
                      padding: "5px 16px",
                    }}
                  >
                    <Cloud className="h-4 w-4 text-[#1447E6]" />
                    <span
                      style={{
                        fontSize: 14,
                        lineHeight: "20px",
                        color: boundCloudDevEnv ? "rgba(10,10,10,0.8)" : "#1447E6",
                        fontWeight: boundCloudDevEnv ? 400 : 500,
                      }}
                    >
                      {boundCloudDevEnv ? "云开发管理" : "绑定云开发环境"}
                    </span>
                  </button>
                </PopoverAnchor>
                <PopoverContent
                  align="end"
                  side="bottom"
                  sideOffset={8}
                  // 不抢焦点 + 不响应外部点击关闭，open 完全由 hover/timeout 控制
                  onOpenAutoFocus={(e) => e.preventDefault()}
                  onInteractOutside={(e) => e.preventDefault()}
                  className="w-auto p-0 border-[var(--cp-border)] shadow-md"
                >
                  <div className="flex items-center gap-2 px-3 py-2 text-sm">
                    {boundCloudDevEnv ? (
                      <>
                        <Check className="w-4 h-4 text-[var(--text-success)] flex-shrink-0" />
                        <span className="text-[var(--text-secondary)]">
                          已绑定云开发环境：
                          <span className="text-[var(--text-brand)] font-mono">{boundCloudDevEnv.envId}</span>
                        </span>
                      </>
                    ) : (
                      <span className="text-[var(--text-secondary)]">
                        尚未绑定云开发环境，请先创建或绑定
                      </span>
                    )}
                  </div>
                </PopoverContent>
              </Popover>
            )}
            {/* 切换角色按钮：Figma 圆角16 bg #F2F4F8 */}
            <button
              type="button"
              className="inline-flex items-center gap-2 hover:bg-[#E8EBF0] active:scale-[0.97] transition-all"
              style={{
                borderRadius: 16,
                background: "#F2F4F8",
                padding: "5px 16px",
              }}
            >
              <img
                src={activeAgent.avatar}
                alt=""
                className="h-4 w-4 rounded-full object-cover pointer-events-none select-none"
              />
              <span
                style={{
                  fontSize: 14,
                  lineHeight: "20px",
                  color: "rgba(10,10,10,0.8)",
                }}
              >
                切换角色
              </span>
            </button>

            {/* 新建会话 */}
            <button
              onClick={() => setShowNewChatConfirm(true)}
              aria-label="新建会话"
              title="新建会话"
              className="flex items-center justify-center text-[#737373] hover:text-[#0A0A0A] hover:bg-[#F5F6F9] active:scale-90 transition-all rounded-full"
              style={{ width: 32, height: 32 }}
            >
              <MessageSquarePlus className="h-5 w-5" />
            </button>

            {/* 全屏切换 */}
            <button
              onClick={() => setIsFullscreen((v) => !v)}
              aria-label={isFullscreen ? "退出全屏" : "全屏"}
              title={isFullscreen ? "退出全屏" : "全屏"}
              className="flex items-center justify-center text-[#737373] hover:text-[#0A0A0A] hover:bg-[#F5F6F9] active:scale-90 transition-all rounded-full"
              style={{ width: 32, height: 32 }}
            >
              {isFullscreen ? (
                <Minimize2 className="h-5 w-5" />
              ) : (
                <Maximize2 className="h-5 w-5" />
              )}
            </button>
          </div>
        </header>

        {/* 内容区：有消息 → 对话流；无消息 → welcome 占位 */}
        {messages.length === 0 ? (
          <div className="flex-1 overflow-y-auto flex flex-col items-center justify-center px-8">
            {/* 居中的内容块：内部元素左对齐 */}
            <div className="flex flex-col items-start">
              {/* 角色头像装饰：56×56 角色虾头像（与左侧选中头像同款，省去外圈红光晕） */}
              <img
                src={activeAgent.avatar}
                alt={activeAgent.name}
                draggable={false}
                className="rounded-full mb-4 object-cover pointer-events-none select-none"
                style={{ width: 56, height: 56 }}
              />

              {/* 欢迎语：20/32 #000 regular（左对齐） */}
              <h2
                className="text-left mb-4"
                style={{
                  fontSize: 20,
                  lineHeight: "32px",
                  color: "#000",
                  fontWeight: 400,
                }}
              >
                你好，我是{activeAgent.name}，今天我们来做些什么呢？
              </h2>

              {/* 快捷指令 chip 组：左对齐堆叠，chip 宽度自适应内容 */}
              <div className="flex flex-col items-start" style={{ gap: 16 }}>
                {QUICK_COMMANDS.map((command) => (
                  <button
                    key={command}
                    onClick={() => setInputText(command)}
                    className="inline-flex items-center transition-all duration-150 text-left bg-white hover:border-[#1447E6]/30 hover:shadow-[0_2px_8px_0_rgba(20,71,230,0.08)] active:scale-[0.98]"
                    style={{
                      borderRadius: 8,
                      border: "1px solid #E9ECF1",
                      padding: "9px 16px",
                      gap: 8,
                    }}
                    // allow-shadow: hover 微凸阴影
                  >
                    <StarBullet />
                    <span
                      style={{
                        fontSize: 14,
                        lineHeight: "22px",
                        color: "#000",
                      }}
                    >
                      {command}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto px-8 py-6">
            <div className="mx-auto flex flex-col" style={{ maxWidth: 720, gap: 20 }}>
              {messages.map((m) =>
                m.role === "user" ? (
                  /* 用户消息：右对齐，灰色气泡 */
                  <div key={m.id} className="flex justify-end">
                    <div
                      style={{
                        maxWidth: "78%",
                        padding: "10px 14px",
                        borderRadius: 12,
                        background: "#F5F6F9",
                        fontSize: 14,
                        lineHeight: "22px",
                        color: "#0A0A0A",
                        whiteSpace: "pre-wrap",
                        wordBreak: "break-word",
                      }}
                    >
                      {m.content}
                    </div>
                  </div>
                ) : (
                  /* AI 消息：左对齐，前置 24×24 角色头像 */
                  <div key={m.id} className="flex items-start gap-2.5">
                    <img
                      src={activeAgent.avatar}
                      alt={activeAgent.name}
                      draggable={false}
                      className="rounded-full object-cover flex-shrink-0 pointer-events-none select-none"
                      style={{ width: 24, height: 24, marginTop: 2 }}
                    />
                    <div
                      style={{
                        maxWidth: "calc(100% - 36px)",
                        fontSize: 14,
                        lineHeight: "22px",
                        color: "#0A0A0A",
                        whiteSpace: "pre-wrap",
                        wordBreak: "break-word",
                      }}
                    >
                      {m.content}
                    </div>
                  </div>
                ),
              )}
              <div ref={messagesEndRef} />
            </div>
          </div>
        )}

        {/* 输入区：圆角 20、stroke #E9ECF1、shadow 0 4 12 rgba(0,0,0,0.04) */}
        <div className="flex-shrink-0" style={{ padding: "0 16px 16px" }}>
          <div
            className="bg-white relative"
            style={{
              borderRadius: 20,
              border: "1px solid #E9ECF1",
              boxShadow: "0 4px 12px 0 rgba(0,0,0,0.04)",
            }}
            // allow-shadow: Figma effect_AO2YEQ
          >
            {/* 输入区：padding 16 20 高 80 */}
            <div style={{ padding: "16px 20px", height: 80 }}>
              <textarea
                value={inputText}
                onChange={(e) => setInputText(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    handleSend();
                  }
                }}
                placeholder="发送消息开始对话"
                className="w-full h-full resize-none focus:outline-none bg-transparent"
                style={{
                  fontSize: 14,
                  lineHeight: "24px",
                  color: "#0A0A0A",
                }}
              />
            </div>

            {/* ActionBar：padding 0 16 16，space-between */}
            <div
              className="flex items-center justify-between"
              style={{ padding: "0 16px 16px" }}
            >
              <div className="flex items-center" style={{ gap: 8 }}>
                {/* 添加按钮（+） */}
                <button
                  type="button"
                  aria-label="附件"
                  className="flex items-center justify-center text-[#737373] hover:text-[#0A0A0A] hover:bg-[#F5F6F9] active:scale-90 transition-all rounded-full"
                  style={{ width: 32, height: 32 }}
                >
                  <Plus className="h-5 w-5" />
                </button>
                {/* 分割线 */}
                <span className="w-px h-4 bg-[#E5E5E5]" />
                {/* Deepseek pill：圆角 16、stroke #E6E9EF、padding 2 12 */}
                <button
                  type="button"
                  className="inline-flex items-center bg-white hover:border-[#1447E6]/40 active:scale-[0.97] transition-all"
                  style={{
                    borderRadius: 16,
                    border: "1px solid #E6E9EF",
                    padding: "2px 12px",
                    gap: 4,
                  }}
                >
                  <span
                    aria-hidden
                    className="flex h-3.5 w-3.5 flex-shrink-0 items-center justify-center rounded-full"
                    style={{ background: "#3970FB" }}
                  >
                    <svg width="8" height="8" viewBox="0 0 8 8" fill="white">
                      <path d="M4 0L5.2 2.8L8 4L5.2 5.2L4 8L2.8 5.2L0 4L2.8 2.8L4 0Z" />
                    </svg>
                  </span>
                  <span
                    style={{
                      fontSize: 12,
                      lineHeight: "20px",
                      color: "rgba(0,0,0,0.9)",
                    }}
                  >
                    Deepseek R1
                  </span>
                  <ChevronDown className="h-4 w-4" style={{ color: "rgba(0,0,0,0.4)" }} />
                </button>
                {/* 指令库 pill：圆角 20、stroke #E9ECF1、padding 6 12 */}
                <button
                  type="button"
                  className="inline-flex items-center hover:border-[#1447E6]/40 active:scale-[0.97] transition-all"
                  style={{
                    borderRadius: 20,
                    border: "1px solid #E9ECF1",
                    padding: "6px 12px",
                    gap: 4,
                  }}
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: "rgba(0,0,0,0.7)" }}>
                    <rect x="3" y="3" width="10" height="10" rx="2" />
                    <path d="M6 7h4M6 9h4" />
                  </svg>
                  <span
                    style={{
                      fontSize: 12,
                      lineHeight: "20px",
                      color: "rgba(0,0,0,0.9)",
                    }}
                  >
                    指令库
                  </span>
                </button>
                {/* 云桌面 pill：样式同指令库 */}
                <button
                  type="button"
                  className="inline-flex items-center hover:border-[#1447E6]/40 active:scale-[0.97] transition-all"
                  style={{
                    borderRadius: 20,
                    border: "1px solid #E9ECF1",
                    padding: "6px 12px",
                    gap: 4,
                  }}
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: "rgba(0,0,0,0.7)" }}>
                    <rect x="2" y="3" width="12" height="8" rx="1.5" />
                    <path d="M5 14h6M8 11v3" />
                  </svg>
                  <span
                    style={{
                      fontSize: 12,
                      lineHeight: "20px",
                      color: "rgba(0,0,0,0.9)",
                    }}
                  >
                    云桌面
                  </span>
                </button>
              </div>

              {/* 右侧操作：发送 */}
              <div className="flex items-center" style={{ gap: 8 }}>
                <button
                  onClick={handleSend}
                  disabled={!inputText.trim()}
                  aria-label="发送"
                  className="flex items-center justify-center text-white rounded-full transition-all duration-150 disabled:opacity-30 enabled:hover:bg-black enabled:active:scale-90"
                  style={{
                    width: 32,
                    height: 32,
                    background: "rgba(0,0,0,0.92)",
                  }}
                >
                  <ArrowUp className="h-5 w-5" strokeWidth={2.5} />
                </button>
              </div>
            </div>
          </div>

          {/* 免责声明：11/24 rgba(0,0,0,0.3) */}
          <p
            className="text-center"
            style={{
              fontSize: 11,
              lineHeight: "24px",
              color: "rgba(0,0,0,0.3)",
              marginTop: 8,
            }}
          >
            通过LightClaw你可以直接和 openclaw对话，内容由当前服务器配置的 AI 模型提供，请注意鉴别
          </p>
        </div>
      </main>

      {/* 新建会话二次确认 */}
      {showNewChatConfirm && (
        <div
          className="absolute inset-0 z-50 flex items-center justify-center"
          style={{ background: "rgba(0,0,0,0.32)" }}
          onClick={() => setShowNewChatConfirm(false)}
        >
          <div
            className="bg-white"
            onClick={(e) => e.stopPropagation()}
            style={{
              width: 360,
              borderRadius: 16,
              padding: 24,
              boxShadow:
                "0 16px 48px 0 rgba(0,0,0,0.16), 0 2px 8px 0 rgba(0,0,0,0.08)",
            }}
            // allow-shadow: 二次确认 modal
          >
            <div
              style={{
                fontSize: 16,
                lineHeight: "24px",
                fontWeight: 600,
                color: "#0A0A0A",
                marginBottom: 8,
              }}
            >
              确认新建会话？
            </div>
            <div
              style={{
                fontSize: 13,
                lineHeight: "20px",
                color: "rgba(10,10,10,0.6)",
                marginBottom: 20,
              }}
            >
              新建会话后，当前会话记录会被清空。
            </div>
            <div className="flex items-center justify-end gap-2">
              <button
                onClick={() => setShowNewChatConfirm(false)}
                className="hover:bg-[#F5F6F9] active:scale-95 transition-all"
                style={{
                  height: 32,
                  borderRadius: 8,
                  border: "1px solid #E9ECF1",
                  padding: "0 16px",
                  fontSize: 13,
                  color: "rgba(10,10,10,0.8)",
                }}
              >
                取消
              </button>
              <button
                onClick={() => {
                  setMessages([]);
                  setInputText("");
                  setShowNewChatConfirm(false);
                }}
                className="text-white hover:bg-black active:scale-95 transition-all"
                style={{
                  height: 32,
                  borderRadius: 8,
                  padding: "0 16px",
                  fontSize: 13,
                  fontWeight: 500,
                  background: "rgba(10,10,10,0.92)",
                }}
              >
                确认新建
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );

  /* ───────── 云开发环境详情弹窗 ─────────
     mode="tenant"：envOptions 为当前用户关联的环境，不显示「前往控制台」「列表项查看详情」；
     - boundEnvId 标识当前 Agent 已绑定的环境（用于切换器角标 + 非绑定环境信息横幅）
     - 切换查看到非绑定环境时，内容区顶部出现"切回绑定环境 / 换绑到当前环境"横幅
     - 当环境已被删除（status=error 且名称含"已删除"，mock 判定）时，顶部出现红色警告横幅 */
  const cloudDevDialog = (
    <CloudDevEnvDetailDialog
      open={cloudDevDialogOpen}
      onOpenChange={setCloudDevDialogOpen}
      env={cloudDevEnv}
      mode="tenant"
      envOptions={cloudEnvs}
      onSelectEnv={(e) => setCloudDevEnv(e)}
      boundEnvId={boundCloudDevEnv?.id}
      isDeleted={cloudDevEnv?.status === "error" && cloudDevEnv?.name?.includes("已删除")}
      onRebind={() => {
        // dev-cloud（新程序员角色）：真实换绑——切换 boundEnvIdCloud 到当前查看的环境
        // dev（旧程序员角色）：保持 stub（不支持换绑，仅查看）
        if (isCloudDevRole) {
          if (!cloudDevEnv) return;
          const prevName = boundCloudDevEnv?.name;
          setBoundEnvIdCloud(cloudDevEnv.id);
          toast.success(
            prevName
              ? `已从「${prevName}」换绑到「${cloudDevEnv.name}」`
              : `已绑定到环境「${cloudDevEnv.name}」`,
          );
        } else {
          // eslint-disable-next-line no-console
          console.log("[CloudDevEnvDetailDialog] tenant/dev：换绑请求（演示态，未真实接入）", cloudDevEnv?.id);
        }
      }}
      onSwitchBackToBound={() => {
        // 切回 Agent 当前绑定的环境
        if (boundCloudDevEnv) setCloudDevEnv(boundCloudDevEnv);
      }}
      onCreateEnv={() => {
        // 配额校验：超过用户端配额上限则阻止打开（参考 PRD：tenant 端做配额校验）
        if (cloudEnvs.length >= TENANT_ENV_QUOTA_MAX) {
          toast.error(`已达到云开发环境配额上限（${TENANT_ENV_QUOTA_MAX} 个）`, {
            description: "请先删除不再使用的环境或联系管理员调整配额。",
          });
          return;
        }
        // 通过校验：关闭详情弹窗（避免两层 Dialog 叠加视觉混乱），打开新建弹窗
        setCloudDevDialogOpen(false);
        setCreateEnvOpen(true);
      }}
      onRefreshEnvs={() => {
        // Mock 刷新（真实场景应重新拉取用户关联环境列表）
        toast.success("环境列表已刷新");
      }}
    />
  );

  /* ───────── 新建云开发环境弹窗（与管控端共用 CloudDevCreateEnvDialog） ─────────
     tenant 端规则：
       - dev-cloud（新程序员）：创建后加入 cloudEnvs + 自动绑定 + 同步到弹窗当前查看
       - dev（旧程序员）：保留原 stub（仅 toast 提示，不会真实加入列表） */
  const PKG_NAMES: Record<string, string> = { personal: "个人版", standard: "标准版", enterprise: "企业版" };
  const createEnvDialog = (
    <CloudDevCreateEnvDialog
      open={createEnvOpen}
      onOpenChange={setCreateEnvOpen}
      onConfirm={(form: CreateEnvForm) => {
        if (isCloudDevRole) {
          const id = String(Date.now());
          const eid = `openclaw-${form.name.replace(/\s+/g, "").toLowerCase().slice(0, 6)}-${Math.random().toString(36).slice(2, 8)}`;
          const today = new Date();
          const expire = new Date(today);
          expire.setFullYear(expire.getFullYear() + 1);
          const newEnv: Env = {
            id,
            envId: eid,
            name: form.name,
            status: "creating",
            region: form.region,
            packageName: PKG_NAMES[form.pkg] ?? form.pkg,
            storageUsed: "0 GB / 10 GB",
            dbUsed: "0 MB / 2 GB",
            functionCount: 0,
            staticHosting: false,
            createdAt: today.toISOString().slice(0, 10),
            expireAt: expire.toISOString().slice(0, 10),
            lastDeployAt: "-",
            appCount: 0,
            appNames: [],
            createdBy: "current-user@acompany.com",
            allowedGroups: [],
            allowedUsers: ["current-user@acompany.com"],
            dbType: form.dbType,
            overflowBilling: form.overflowBilling,
            autoRenewal: form.autoRenewal,
          };
          // 1) 加入环境列表
          setCloudEnvs((prev) => [...prev, newEnv]);
          // 2) 自动绑定到当前 Agent
          setBoundEnvIdCloud(id);
          // 3) 同步把详情弹窗的当前查看也切到新环境
          setCloudDevEnv(newEnv);
          // 4) 关闭新建弹窗 + 提示
          setCreateEnvOpen(false);
          toast.success(`环境「${form.name}」已创建并自动绑定`, {
            description: `环境 ID：${eid} · ${PKG_NAMES[form.pkg] ?? form.pkg}`,
          });
        } else {
          // 旧 dev 角色：保留演示态
          // eslint-disable-next-line no-console
          console.log("[CloudDevCreateEnvDialog] tenant/dev：确认创建环境（演示态，未加入列表）", form);
          setCreateEnvOpen(false);
          toast.success(`云开发环境「${form.name}」已开始创建`);
        }
      }}
    />
  );

  /* ───────── 全屏 wrapper：固定在导航下方（top:64）+ 纯色背景 + padding ───────── */
  const fullscreenWrapper = (
    <div
      className="fixed left-0 right-0 bottom-0 z-[200] flex items-center justify-center"
      style={{
        top: 64,
        background: "#F7F8FB",
        padding: 32,
      }}
    >
      {appCard}
    </div>
  );

  if (embedded) {
    return <>{isFullscreen ? fullscreenWrapper : appCard}{cloudDevDialog}{createEnvDialog}</>;
  }

  /* ───────── 独立预览模式：Figma 整页底色 #F7F8FB + 居中卡片 + 尺寸徽章 ───────── */
  if (isFullscreen) {
    return <>{fullscreenWrapper}{cloudDevDialog}{createEnvDialog}</>;
  }

  return (
    <>
      <div
        className="page-enter min-h-screen w-full flex items-center justify-center p-8 relative"
        style={{ background: "#F7F8FB" }}
      >
        {appCard}
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2">
          <span
            className="inline-flex items-center px-2 h-5 rounded-[4px] text-white"
            style={{
              background: "#1447E6",
              fontSize: 10,
              fontWeight: 500,
              letterSpacing: 0.5,
            }}
          >
            1200 × 768
          </span>
        </div>
      </div>
      {cloudDevDialog}
      {createEnvDialog}
    </>
  );
}
