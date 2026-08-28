/**
 * CloudDevActivation - 云开发功能开通页
 * 未开通时展示，点击"立即开通"后进入管理列表页
 */
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { SectionTitle, TenantPageTitle, BodyText, MetaText } from "@/components/ui/Typography";
import { Zap, ArrowRight } from "lucide-react";
import DarkVeil from "@/components/ui/DarkVeil";

const FEATURE_ICON_BASE = "/assets/admin-cloud-dev";

const FEATURES = [
  {
    iconSrc: "/assets/admin-skill-packages/advanced-dev-skill-package.svg",
    title: "独立云开发环境",
    desc: "为每位成员分配独立的云开发运行环境，资源隔离互不干扰",
  },
  {
    iconSrc: `${FEATURE_ICON_BASE}/cloud-database.svg`,
    title: "云数据库",
    desc: "内置 NoSQL 文档数据库，支持集合管理、索引配置与数据导入导出",
  },
  {
    iconSrc: `${FEATURE_ICON_BASE}/cloud-function.svg`,
    title: "云函数",
    desc: "编写并部署服务端逻辑，支持 Node.js 多版本运行时，自动弹性扩缩容",
  },
  {
    iconSrc: `${FEATURE_ICON_BASE}/static-hosting.svg`,
    title: "静态网站托管",
    desc: "一键部署前端应用，自带 CDN 加速与 HTTPS 支持，访问即享极速体验",
  },
];

const BENEFIT_ICON_BASE = "/assets/admin-memory-management/version-compare";

const BENEFITS = [
  { iconSrc: `${BENEFIT_ICON_BASE}/feature-token.svg`, label: "管理员可创建、删除、管理云开发环境" },
  { iconSrc: `${BENEFIT_ICON_BASE}/feature-tenant.svg`, label: "灵活分配环境给指定成员使用" },
  { iconSrc: `${BENEFIT_ICON_BASE}/feature-backup.svg`, label: "统一监控环境运行状态与资源用量" },
  { iconSrc: `${BENEFIT_ICON_BASE}/feature-encrypt.svg`, label: "企业级安全管控与审计日志" },
];

interface CloudDevActivationProps {
  onActivated: () => void;
}

export default function CloudDevActivation({ onActivated }: CloudDevActivationProps) {
  const [loading, setLoading] = useState(false);

  const handleActivate = () => {
    setLoading(true);
    // 模拟开通请求
    setTimeout(() => {
      localStorage.setItem("cloudDevActivated", "true");
      setLoading(false);
      onActivated();
    }, 1200);
  };

  return (
    <div className="page-enter">
      <AdminPageHeader
        title="云开发管理"
        description="管理企业云开发环境的创建、分配与生命周期"
      />

      {/* 开通卡片 */}
      <SurfaceCard className="overflow-hidden">
        {/* Hero 区域 */}
        <div className="relative overflow-hidden px-[60px] py-12">
          {/* 底层：统一 #E0EBFE 基底，整体均匀，避免局部突兀 */}
          <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />
          {/* DarkVeil 动态背景：保持正常比例（不拉伸），轻微下移让飘带落到下方，顶部用蒙版淡出 → 露出的是蓝紫基底而非白 */}
          <DarkVeil
            speed={1.1}
            warpAmount={1.1}
            noiseIntensity={0.05}
            tintColor="#B2C3FF"
            className="pointer-events-none absolute inset-0 h-full w-full"
            style={{
              opacity: 1,
              transform: "translateY(72px)",
              maskImage:
                "linear-gradient(to bottom, transparent 0%, #000 22%)",
              WebkitMaskImage:
                "linear-gradient(to bottom, transparent 0%, #000 22%)",
            }}
          />
          {/* 柔化叠层：中部微弱提亮（降低强度避免突兀），底部收束到 #E0EBFE 与基底无缝衔接 */}
          <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" />

          <div className="relative z-10 grid items-end gap-4 md:grid-cols-2">
            {/* 左：图标 + 标题 + 描述 + 按钮（占左半列，与核心能力左列对齐；右内边距让与卡片间距为 48px） */}
            <div className="min-w-0 md:pr-8">
              <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-[8px] border border-white/60 bg-white/30 backdrop-blur-md">
                <img src="/assets/admin-sidebar/cloud-dev.svg" alt="cloud-dev" className="h-7 w-7" />
              </div>
              <TenantPageTitle className="mb-2">开通云开发能力</TenantPageTitle>
              <BodyText className="max-w-[580px] text-[var(--text-muted)]">
                为企业成员提供独立的云端开发环境，集成数据库、云函数与静态托管，让 Agent 拥有完整的后端服务能力，快速构建与部署应用
              </BodyText>
              <div className="mt-6 flex items-center">
                <Button
                  variant="claw-primary"
                  size="lg"
                  onClick={handleActivate}
                  disabled={loading}
                >
                  {loading ? (
                    <span className="flex items-center gap-2">
                      <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                      开通中...
                    </span>
                  ) : (
                    <>
                      立即开通
                      <ArrowRight className="ml-2 w-5 h-5" />
                    </>
                  )}
                </Button>
              </div>
            </div>

            {/* 右：开通后可获得 —— 4 张卡片，占右半列、底端与左列按钮对齐；卡片宽度自适应内容（≤240px） */}
            <div className="min-w-0">
              <div className="grid w-fit grid-cols-[repeat(2,minmax(0,240px))] gap-2.5">
                {BENEFITS.map(({ iconSrc, label }) => (
                  <div
                    key={label}
                    className="rounded-[9px] border border-white/60 bg-white/40 px-3.5 py-3 backdrop-blur-sm transition-colors hover:border-white/80"
                  >
                    <img src={iconSrc} alt="" aria-hidden="true" className="h-4 w-4" />
                    <p className="mt-1.5 text-xs leading-5 text-[var(--text-secondary)]">{label}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* 功能特性 */}
        <div className="px-[60px] pt-10 pb-10">
          <SectionTitle className="mb-6">核心能力</SectionTitle>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {FEATURES.map((f) => (
              <SurfaceInner
                key={f.title}
                className="flex items-center p-4 transition-all duration-200 hover:-translate-y-0.5 hover:border-[var(--cp-border-control)]"
              >
                <div className="flex items-center gap-4">
                  <img src={f.iconSrc} alt="" aria-hidden="true" className="h-9 w-9 flex-shrink-0" />
                  <div className="min-w-0">
                    <h4 className="mb-1 text-sm font-semibold text-[var(--text-title)]">{f.title}</h4>
                    <p className="text-xs leading-relaxed text-[var(--text-muted)]">{f.desc}</p>
                  </div>
                </div>
              </SurfaceInner>
            ))}
          </div>
        </div>

        {/* 底部说明 */}
        <div className="flex items-center justify-center gap-1.5 px-[60px] pb-10 text-xs text-[var(--text-weak)]">
          <Zap className="h-3.5 w-3.5" />
          <span>开通即用，无需额外配置基础设施</span>
        </div>
      </SurfaceCard>
    </div>
  );
}
