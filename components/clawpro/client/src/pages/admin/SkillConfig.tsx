/**
 * SkillConfig - 管控端技能配置页
 * Design: 「流动蓝图」Fluid Blueprint
 * 四个 Tab：初始技能包（即将开放）、技能安装来源（现有功能）、公共技能库（即将开放）、企业技能库（即将开放）
 */
import { useEffect, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Puzzle,
  Pencil,
  X,
  Check,
  PackagePlus,
  RefreshCw,
  Upload,
  HardDrive,
  Download,
  Globe,
  Zap,
} from "lucide-react";
import SkillInitialPackageTab from "./SkillLibrary/SkillInitialPackageTab";
import SkillRolesTab from "./SkillRolesTab";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { LineTabs } from "@/components/ui/line-tabs";
import {
  BodyMedium,
  CardTitle,
  CompactText,
  MetaText,
} from "@/components/ui/Typography";

// ── Tab 定义 ──────────────────────────────────────────────
type SkillConfigTabId = "preset" | "roles" | "source";

const TABS: ReadonlyArray<{
  id: SkillConfigTabId;
  label: string;
  description: string;
  dataGuide?: string;
  comingSoon?: boolean;
}> = [
  {
    id: "preset",
    label: "初始技能包",
    description: "配置每个实例自动预装的技能集合，支持从公共技能库和企业技能库中挑选。",
    // 稳定选择器：用于在停服态下精准定位该 Tab 入口并附加 data-billing-exempt 豁免
    dataGuide: "skill-config-preset-tab",
  },
  {
    id: "roles",
    label: "角色设定",
    description: "创建和管理角色预设，用户在创建实例时可选择管理员配置好的角色，快速获得对应技能组合。",
    // 稳定选择器：用于在停服态下精准定位该 Tab 入口并附加 data-billing-exempt 豁免
    dataGuide: "skill-config-roles-tab",
  },
  {
    id: "source",
    label: "技能安装来源",
    description: "控制用户在实例配置页中可以从哪些来源浏览和安装新技能。",
    // 稳定选择器：用于在停服态下精准定位该 Tab 入口并附加 data-billing-exempt 豁免
    dataGuide: "skill-config-source-tab",
  },
];

// ── 初始技能包 介绍卡片（2张）────────────────────────────
const PRESET_CARDS = [
  {
    id: "pick",
    title: "从多来源挑选技能",
    description:
      "从多个公共技能库以及企业私有技能库中自由挑选技能，组合成每个 Agent 开箱即用的初始技能集合",
    icon: PackagePlus,
    color: "#355EF1",
  },
  {
    id: "manage",
    title: "灵活管理技能增删",
    description:
      "随时对初始技能包进行技能的添加和移除，灵活调整每个 Agent 的预装技能组合，适应企业需求变化",
    icon: RefreshCw,
    color: "#34C759",
  },
];

// ── 公共技能库 介绍卡片（2张）────────────────────────────
const PUBLIC_CARDS = [
  {
    id: "browse",
    title: "多渠道公共技能市场",
    description:
      "从多个公共技能库中浏览和挑选技能，按需组合形成适合企业实际场景的公共技能库，避免团队自行搜索安装的重复劳动",
    icon: Globe,
    color: "#355EF1",
  },
  {
    id: "speed",
    title: "海量技能自由选配",
    description:
      "汇聚数万个开箱即用的 Skill，涉及文件处理、代码执行、数据分析等多个领域，按需挑选组合，打造专属于企业的定制化 Agent 数字助理",
    icon: Zap,
    color: "#FF9500",
  },
];

// ── 企业技能库 介绍卡片（3张）────────────────────────────
const LIBRARY_CARDS = [
  {
    id: "upload",
    title: "上传企业 Skill",
    description:
      "支持企业自定义 Skill 压缩包上传与版本控制，构建企业私有技能仓库，确保核心资产仅限内部调用",
    icon: Upload,
    color: "#355EF1",
  },
  {
    id: "bucket",
    title: "自有存储桶",
    description:
      "采用企业私有存储模式，一键授权创建腾讯云专属存储桶，数据物理隔离，支持内网高速互联",
    icon: HardDrive,
    color: "#AF52DE",
  },
  {
    id: "broadcast",
    title: "一键批量下发",
    description:
      "将企业技能库中的最新技能批量下发至所有云服务器，统一技能环境，分钟级配置同步，大幅降低运维成本",
    icon: Download,
    color: "#FF9500",
  },
];

// ── 介绍卡片组件 ──────────────────────────────────────────
function ComingSoonCards({
  cards,
}: {
  cards: { id: string; title: string; description: string; icon: React.ElementType; color: string }[];
}) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {cards.map((card) => {
        const Icon = card.icon;
        return (
          <Card key={card.id} className="py-5 gap-0 hover:border-[#1447E6] transition-colors cursor-pointer">
            <CardContent className="flex items-start gap-4">
              <div
                className="w-10 h-10 rounded-[4px] flex items-center justify-center flex-shrink-0"
                style={{ background: card.color }}
              >
                <Icon className="w-5 h-5 text-white" />
              </div>
              <div className="flex-1 min-w-0">
                <BodyMedium as="h3" className="block mb-1 font-semibold">{card.title}</BodyMedium>
                <MetaText as="p">{card.description}</MetaText>
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}

// ── 技能安装来源 Tab 内容（保持原有功能） ─────────────────
function SkillSourceTab() {
  const [skillhubUrl, setSkillhubUrl] = useState("https://clawhub.agent.com");
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(skillhubUrl);
  const [errorMessage, setErrorMessage] = useState("");

  const handleEdit = () => {
    setDraft(skillhubUrl);
    setEditing(true);
    setErrorMessage("");
  };

  const handleSave = () => {
    const trimmedUrl = draft.trim();
    if (!trimmedUrl) {
      setSkillhubUrl("");
      setEditing(false);
      setErrorMessage("");
      return;
    }
    try {
      new URL(trimmedUrl);
    } catch {
      setErrorMessage("请输入完整的 URL 地址（如：https://example.com）");
      return;
    }
    setSkillhubUrl(trimmedUrl);
    setEditing(false);
    setErrorMessage("");
  };

  const handleCancel = () => {
    setDraft(skillhubUrl);
    setEditing(false);
    setErrorMessage("");
  };

  const handleDraftChange = (value: string) => {
    setDraft(value);
    if (errorMessage) setErrorMessage("");
  };

  return (
    <Card className="overflow-hidden py-0 gap-0">
      {/* 卡片头部：图标 + 标题 + 描述 + 编辑按钮 */}
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start gap-3">
          <div className="shrink-0">
            <svg width="40" height="40" viewBox="0 0 36 36" fill="none" xmlns="http://www.w3.org/2000/svg">
              <g clipPath="url(#paint0_diamond_skillhub_clip)"><g transform="matrix(-0.151677 -0.153527 0.34384 -0.339697 48.6926 49.3816)"><rect x="0" y="0" width="327.238" height="55.7561" fill="url(#paint0_diamond_skillhub)" opacity="1" shapeRendering="crispEdges"/><rect x="0" y="0" width="327.238" height="55.7561" transform="scale(1 -1)" fill="url(#paint0_diamond_skillhub)" opacity="1" shapeRendering="crispEdges"/><rect x="0" y="0" width="327.238" height="55.7561" transform="scale(-1 1)" fill="url(#paint0_diamond_skillhub)" opacity="1" shapeRendering="crispEdges"/><rect x="0" y="0" width="327.238" height="55.7561" transform="scale(-1)" fill="url(#paint0_diamond_skillhub)" opacity="1" shapeRendering="crispEdges"/></g></g><rect width="36" height="36" rx="8" fill="url(#paint0_diamond_skillhub)" fillOpacity="0.01"/>
              <path d="M20.2602 12.9264C20.344 13.0103 20.4487 13.0703 20.5634 13.1003C20.6781 13.1302 20.7988 13.129 20.9129 13.0969C21.027 13.0647 21.1305 13.0026 21.2126 12.9171C21.2948 12.8316 21.3526 12.7257 21.3802 12.6104C21.4494 12.3223 21.5942 12.0579 21.7995 11.8444C22.0049 11.6309 22.2635 11.4759 22.5486 11.3955C22.8338 11.3152 23.1352 11.3123 23.4219 11.3871C23.7085 11.462 23.97 11.6119 24.1795 11.8214C24.389 12.0309 24.5388 12.2925 24.6135 12.5792C24.6883 12.8658 24.6853 13.1673 24.6048 13.4524C24.5243 13.7375 24.3693 13.996 24.1557 14.2013C23.9421 14.4066 23.6777 14.5513 23.3896 14.6204C23.2743 14.648 23.1684 14.7059 23.0829 14.788C22.9974 14.8701 22.9353 14.9736 22.9031 15.0877C22.8709 15.2018 22.8697 15.3225 22.8997 15.4372C22.9297 15.5519 22.9897 15.6566 23.0736 15.7404L24.1956 16.8617C24.345 17.0112 24.4636 17.1886 24.5445 17.3838C24.6253 17.5791 24.667 17.7884 24.667 17.9997C24.667 18.2111 24.6253 18.4204 24.5445 18.6156C24.4636 18.8109 24.345 18.9883 24.1956 19.1377L23.0736 20.2597C22.9898 20.3436 22.8851 20.4036 22.7704 20.4336C22.6557 20.4636 22.535 20.4624 22.4209 20.4302C22.3068 20.398 22.2033 20.3359 22.1212 20.2504C22.039 20.1649 21.9812 20.0591 21.9536 19.9437C21.8844 19.6557 21.7396 19.3913 21.5343 19.1777C21.3289 18.9642 21.0703 18.8093 20.7852 18.7289C20.5 18.6485 20.1986 18.6456 19.9119 18.7204C19.6253 18.7953 19.3638 18.9452 19.1543 19.1547C18.9448 19.3643 18.795 19.6258 18.7203 19.9125C18.6455 20.1992 18.6485 20.5006 18.729 20.7857C18.8095 21.0709 18.9645 21.3294 19.1781 21.5347C19.3917 21.74 19.6561 21.8846 19.9442 21.9537C20.0596 21.9813 20.1654 22.0392 20.2509 22.1213C20.3364 22.2035 20.3985 22.3069 20.4307 22.4211C20.4629 22.5352 20.4641 22.6558 20.4341 22.7705C20.4041 22.8853 20.3441 22.9899 20.2602 23.0737L19.1382 24.1951C18.9888 24.3445 18.8114 24.4631 18.6161 24.5439C18.4209 24.6248 18.2116 24.6665 18.0002 24.6665C17.7889 24.6665 17.5796 24.6248 17.3844 24.5439C17.1891 24.4631 17.0117 24.3445 16.8622 24.1951L15.7402 23.0731C15.6564 22.9892 15.5518 22.9292 15.437 22.8992C15.3223 22.8692 15.2017 22.8704 15.0876 22.9026C14.9734 22.9348 14.87 22.9969 14.7878 23.0824C14.7057 23.1679 14.6478 23.2737 14.6202 23.3891C14.551 23.6771 14.4063 23.9415 14.2009 24.1551C13.9956 24.3686 13.737 24.5235 13.4518 24.6039C13.1667 24.6843 12.8652 24.6872 12.5786 24.6124C12.2919 24.5375 12.0304 24.3876 11.821 24.1781C11.6115 23.9685 11.4617 23.707 11.3869 23.4203C11.3122 23.1336 11.3152 22.8322 11.3957 22.5471C11.4761 22.2619 11.6311 22.0034 11.8447 21.7981C12.0583 21.5928 12.3228 21.4482 12.6109 21.3791C12.7262 21.3515 12.8321 21.2936 12.9176 21.2115C13.0031 21.1293 13.0652 21.0259 13.0974 20.9117C13.1296 20.7976 13.1307 20.677 13.1008 20.5623C13.0708 20.4475 13.0108 20.3429 12.9269 20.2591L11.8049 19.1377C11.6555 18.9883 11.5369 18.8109 11.456 18.6156C11.3751 18.4204 11.3335 18.2111 11.3335 17.9997C11.3335 17.7884 11.3751 17.5791 11.456 17.3838C11.5369 17.1886 11.6555 17.0112 11.8049 16.8617L12.9269 15.7397C13.0107 15.6558 13.1154 15.5958 13.2301 15.5659C13.3448 15.5359 13.4655 15.5371 13.5796 15.5693C13.6937 15.6015 13.7972 15.6635 13.8793 15.749C13.9614 15.8345 14.0193 15.9404 14.0469 16.0557C14.1161 16.3438 14.2608 16.6082 14.4662 16.8217C14.6716 17.0353 14.9302 17.1902 15.2153 17.2706C15.5005 17.351 15.8019 17.3539 16.0885 17.279C16.3752 17.2042 16.6367 17.0543 16.8462 16.8447C17.0556 16.6352 17.2054 16.3736 17.2802 16.087C17.355 15.8003 17.3519 15.4989 17.2715 15.2137C17.191 14.9286 17.036 14.6701 16.8224 14.4648C16.6088 14.2595 16.3443 14.1148 16.0562 14.0457C15.9409 14.0181 15.835 13.9603 15.7495 13.8781C15.664 13.796 15.602 13.6925 15.5698 13.5784C15.5376 13.4643 15.5364 13.3436 15.5664 13.2289C15.5963 13.1142 15.6563 13.0095 15.7402 12.9257L16.8622 11.8044C17.0117 11.655 17.1891 11.5364 17.3844 11.4555C17.5796 11.3746 17.7889 11.333 18.0002 11.333C18.2116 11.333 18.4209 11.3746 18.6161 11.4555C18.8114 11.5364 18.9888 11.655 19.1382 11.8044L20.2602 12.9264Z" stroke="url(#paint1_linear_skillhub)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
              <defs>
                <clipPath id="paint0_diamond_skillhub_clip"><rect width="36" height="36" rx="8"/></clipPath>
                <linearGradient id="paint0_diamond_skillhub" x1="0" y1="0" x2="500" y2="500" gradientUnits="userSpaceOnUse"><stop stopColor="#1447E6" stopOpacity="0.01"/><stop offset="1" stopColor="white"/></linearGradient>
                <linearGradient id="paint1_linear_skillhub" x1="23.5689" y1="24.8185" x2="21.302" y2="19.2571" gradientUnits="userSpaceOnUse"><stop stopColor="#0080FF"/><stop offset="1" stopColor="#202020"/></linearGradient>
              </defs>
            </svg>
          </div>
          <div className="flex-1">
            <CardTitle as="h3">SkillHub 地址</CardTitle>
            <MetaText as="p" className="mt-1 leading-relaxed">
              填写企业自建或采购的 SkillHub 服务地址，用户的技能市场将从此地址加载可用技能列表。若留空，将默认使用 ClawHub 官方技能库。
            </MetaText>
          </div>
          {!editing && (
            <Button variant="claw-outline" size="claw-sm" className="shrink-0" onClick={handleEdit}>
              编辑
            </Button>
          )}
          {editing && (
            <div className="flex items-center gap-2 shrink-0">
              <Button variant="claw-outline" size="claw-sm" onClick={handleCancel}>取消</Button>
              <Button variant="dialog-confirm" size="claw-sm" onClick={handleSave}>保存</Button>
            </div>
          )}
        </div>
      </div>

      {/* 内容区 */}
      <div className="px-5 pb-4">
        <div className="rounded-[4px] bg-[#FAFBFD] overflow-hidden px-4 py-3">
          {editing ? (
            <div className="flex flex-col gap-2">
              <Input
                value={draft}
                onChange={(e) => handleDraftChange(e.target.value)}
                placeholder="https://clawhub.yourcompany.com"
                className={errorMessage ? "border-red-500" : ""}
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleSave();
                  if (e.key === "Escape") handleCancel();
                }}
              />
              {errorMessage && <MetaText as="p" tone="danger">{errorMessage}</MetaText>}
            </div>
          ) : (
            <CompactText as="div" tone="emphasis">
              {skillhubUrl || <CompactText tone="muted">未配置</CompactText>}
            </CompactText>
          )}
        </div>
      </div>
    </Card>
  );
}

// ── 主页面 ────────────────────────────────────────────────
export default function SkillConfig() {
  const [activeTab, setActiveTab] = useState<SkillConfigTabId>("preset");

  const currentTab = TABS.find((t) => t.id === activeTab)!;

  /*
   * 停服时仍允许「初始技能包」「角色设定」「技能安装来源」Tab 入口可点击：纯导航/查看类操作。
   * LineTabs 是通用 UI 组件、未提供 per-tab data-billing-exempt 注入点，
   * 这里通过 data-guide 定位对应的 button 元素并附加豁免标记，
   * 使 AdminDisabledOverlay 的全局 CSS 与点击拦截对其失效。
   */
  useEffect(() => {
    const presetEl = document.querySelector<HTMLElement>(
      '[data-guide="skill-config-preset-tab"]'
    );
    if (presetEl) {
      presetEl.setAttribute("data-billing-exempt", "");
    }
    const rolesEl = document.querySelector<HTMLElement>(
      '[data-guide="skill-config-roles-tab"]'
    );
    if (rolesEl) {
      rolesEl.setAttribute("data-billing-exempt", "");
    }
    const sourceEl = document.querySelector<HTMLElement>(
      '[data-guide="skill-config-source-tab"]'
    );
    if (sourceEl) {
      sourceEl.setAttribute("data-billing-exempt", "");
    }
  }, [activeTab]);

  return (
    <div className="page-enter">
      <AdminPageHeader title="技能配置" />

      {/* Tab 切换器 + 描述（统一使用 LineTabs，规范见 SKILL §11.5） */}
      <LineTabs
        tabs={TABS}
        active={activeTab}
        onChange={setActiveTab}
        description={currentTab.description}
      />

      {/* Tab 内容 */}
      {activeTab === "preset" && (
        <SkillInitialPackageTab onPackagesChange={() => {}} />
      )}
      {activeTab === "roles" && (
        <SkillRolesTab />
      )}
      {activeTab === "source" && <div className="max-w-[960px]"><SkillSourceTab /></div>}

    </div>
  );
}
