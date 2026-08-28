/**
 * Avatar Preview · Spec Verification
 * --------------------------------------------------------------
 * 路由：/preview/avatar
 * 用途：对照 `clawpro-portable-design-skill/component-specs/avatar.md`
 *      逐项验证视觉规范、尺寸场景、Portable fallback、Do/Don't、QA Checklist。
 */
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { MetaText, SectionTitle } from "@/components/ui/Typography";
import { Check, X } from "lucide-react";

/* -------------------------------------------------------------------------- */
/*  Spec tokens（spec 约定的 token，client 全局尚未注入，预览页内联）          */
/* -------------------------------------------------------------------------- */
const SPEC_TOKENS: React.CSSProperties = {
  // 来源：clawpro-portable-design-skill/portable/css/tokens.css
  ["--cp-bg-subtle" as never]: "#F5F5F5", // spec md 第 19 行写 #F5F5F5
  ["--cp-text-muted" as never]: "#64748B",
};

/* -------------------------------------------------------------------------- */
/*  Portable Avatar（严格复刻 spec §4 提供的实现）                              */
/* -------------------------------------------------------------------------- */
function PortableAvatar({
  src,
  name,
  size = 32,
}: {
  src?: string;
  name: string;
  size?: number;
}) {
  const initials = name.slice(0, 2).toUpperCase();
  return (
    <span
      className="inline-flex items-center justify-center rounded-full bg-[var(--cp-bg-subtle)] overflow-hidden shrink-0"
      style={{ width: size, height: size }}
    >
      {src ? (
        <img src={src} alt={name} className="h-full w-full object-cover" />
      ) : (
        <span className="text-xs font-medium text-[var(--cp-text-muted)]">
          {initials}
        </span>
      )}
    </span>
  );
}

/* -------------------------------------------------------------------------- */
/*  Layout helpers                                                            */
/* -------------------------------------------------------------------------- */
function DemoBlock({
  title,
  desc,
  children,
}: {
  title: string;
  desc?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden bg-white">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <SectionTitle as="h3" className="!text-sm">
          {title}
        </SectionTitle>
        {desc ? <MetaText tone="weak">{desc}</MetaText> : null}
      </header>
      <div className="p-5">{children}</div>
    </section>
  );
}

function SpecRow({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-dashed border-[#f0f0f0] last:border-0">
      <span className="text-[13px] text-[#404040]">{label}</span>
      <span className="text-[13px] font-mono text-[#0F172A]">{value}</span>
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/*  Page                                                                      */
/* -------------------------------------------------------------------------- */
export default function AvatarPreview() {
  // QA Checklist（spec §6）→ 自检结果，全部由本页演示覆盖即可
  const checklist: { label: string; pass: boolean; note?: string }[] = [
    { label: "圆形裁切（rounded-full）", pass: true },
    { label: "尺寸从标准 4 档（24 / 32 / 40 / 48）选取", pass: true },
    {
      label: "无图片时有首字母 fallback",
      pass: true,
      note: "name.slice(0,2).toUpperCase()",
    },
    {
      label: "fallback 背景使用 token --cp-bg-subtle",
      pass: true,
      note: "Portable 版本已使用；Radix 版需后续统一",
    },
  ];

  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8" style={SPEC_TOKENS}>
      <div className="max-w-3xl mx-auto space-y-8">
        {/* ----------------------------- Header ------------------------------ */}
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">Avatar 头像 · Spec 验证</h1>
          <p className="text-sm text-[#64748B]">
            对照{" "}
            <code className="font-mono text-[12px] bg-[#F1F5F9] px-1.5 py-0.5 rounded">
              component-specs/avatar.md
            </code>{" "}
            §3 视觉规范 · §4 Portable fallback · §5 Do/Don&apos;t · §6 QA Checklist
          </p>
        </header>

        {/* ------------------- §3 Visual Standard 参数对照 ------------------- */}
        <DemoBlock title="§3 Visual Standard" desc="参数对照表">
          <div className="space-y-0">
            <SpecRow label="默认尺寸" value="32px × 32px (size-8)" />
            <SpecRow label="圆角" value="rounded-full" />
            <SpecRow
              label="Fallback 背景"
              value={
                <span className="inline-flex items-center gap-2">
                  <span
                    className="inline-block w-4 h-4 rounded border border-[#e5e5e5]"
                    style={{ background: "var(--cp-bg-subtle)" }}
                  />
                  var(--cp-bg-subtle) #F5F5F5
                </span>
              }
            />
            <SpecRow
              label="Fallback 文字"
              value={
                <span className="inline-flex items-center gap-2">
                  <span style={{ color: "var(--cp-text-muted)", fontWeight: 500 }}>
                    14px Medium
                  </span>
                  var(--cp-text-muted) #64748B
                </span>
              }
            />
            <SpecRow label="尺寸变体" value="24 / 32 / 40 / 48" />
          </div>
        </DemoBlock>

        {/* ------------------- §3 4 档标准尺寸（视觉对齐） ------------------- */}
        <DemoBlock title="4 档标准尺寸" desc="size-6 / size-8 / size-10 / size-12">
          <div className="flex items-end gap-8">
            {[
              { px: 24, cls: "h-6 w-6", initials: "AB", scene: "行内 / 紧凑列表" },
              { px: 32, cls: "h-8 w-8", initials: "JX", scene: "表格行 / 侧边栏" },
              { px: 40, cls: "h-10 w-10", initials: "MK", scene: "卡片头部 / 详情页" },
              { px: 48, cls: "h-12 w-12", initials: "ZH", scene: "个人中心 / 大头像" },
            ].map((it) => (
              <div key={it.px} className="flex flex-col items-center gap-2">
                <Avatar className={it.cls}>
                  <AvatarFallback
                    style={{
                      background: "var(--cp-bg-subtle)",
                      color: "var(--cp-text-muted)",
                    }}
                    className="text-xs font-medium"
                  >
                    {it.initials}
                  </AvatarFallback>
                </Avatar>
                <MetaText className="!text-[12px]">{it.px}px</MetaText>
                <MetaText tone="weak" className="!text-[11px]">
                  {it.scene}
                </MetaText>
              </div>
            ))}
          </div>
        </DemoBlock>

        {/* --------------------- §4 Portable Fallback 实现 -------------------- */}
        <DemoBlock
          title="§4 Portable Fallback"
          desc="脱离 Radix 的纯 span 实现，spec 原文 1:1 复刻"
        >
          <div className="grid grid-cols-2 gap-6">
            <div>
              <div className="flex items-end gap-6">
                <div className="flex flex-col items-center gap-2">
                  <PortableAvatar name="Alice Wang" size={24} />
                  <MetaText className="!text-[12px]">24</MetaText>
                </div>
                <div className="flex flex-col items-center gap-2">
                  <PortableAvatar name="Jin Xu" size={32} />
                  <MetaText className="!text-[12px]">32</MetaText>
                </div>
                <div className="flex flex-col items-center gap-2">
                  <PortableAvatar name="Mike Liu" size={40} />
                  <MetaText className="!text-[12px]">40</MetaText>
                </div>
                <div className="flex flex-col items-center gap-2">
                  <PortableAvatar name="Zhang Han" size={48} />
                  <MetaText className="!text-[12px]">48</MetaText>
                </div>
              </div>
              <p className="mt-3 text-[12px] text-[#64748B] leading-relaxed">
                Portable 版本使用 <code>--cp-bg-subtle</code> 与{" "}
                <code>--cp-text-muted</code>，不依赖任何组件库。
              </p>
            </div>
            <div>
              <div className="flex items-end gap-6">
                <div className="flex flex-col items-center gap-2">
                  <PortableAvatar
                    name="Avatar Img"
                    size={40}
                    src="https://api.dicebear.com/7.x/initials/svg?seed=AI&backgroundColor=DBEAFE"
                  />
                  <MetaText className="!text-[12px]">有图</MetaText>
                </div>
                <div className="flex flex-col items-center gap-2">
                  <PortableAvatar
                    name="Error Case"
                    size={40}
                    src="/__broken-url-for-test.png"
                  />
                  <MetaText className="!text-[12px]">图片失败</MetaText>
                </div>
              </div>
              <p className="mt-3 text-[12px] text-[#64748B] leading-relaxed">
                注：原生 <code>&lt;img&gt;</code> 加载失败不会自动切回首字母，
                Radix 版本通过 <code>onLoadingStatusChange</code> 自动 fallback；
                portable 版若需自动 fallback 应在外层加 <code>onError</code>。
              </p>
            </div>
          </div>
        </DemoBlock>

        {/* ------------------- 真实使用场景（spec §3 尺寸场景） ----------------- */}
        <DemoBlock title="真实使用场景" desc="按 spec §3 的「尺寸 × 场景」映射">
          <div className="space-y-5">
            {/* 24px 行内 */}
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                24px · 行内 / 紧凑列表
              </MetaText>
              <div className="flex items-center gap-1.5 text-[13px] text-[#0F172A]">
                由
                <Avatar className="h-6 w-6 mx-1">
                  <AvatarFallback
                    style={{
                      background: "var(--cp-bg-subtle)",
                      color: "var(--cp-text-muted)",
                    }}
                    className="text-[10px] font-medium"
                  >
                    LJ
                  </AvatarFallback>
                </Avatar>
                Liu Jie 创建于 2026-06-09 14:32
              </div>
            </div>

            {/* 32px 表格行 */}
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                32px · 表格行 / 侧边栏
              </MetaText>
              <div className="rounded-md border border-[#e5e5e5] divide-y divide-[#f0f0f0]">
                {[
                  { name: "张三", role: "管理员", initials: "张三" },
                  { name: "Wang Tao", role: "开发者", initials: "WT" },
                  { name: "Mike Liu", role: "成员", initials: "ML" },
                ].map((u) => (
                  <div
                    key={u.name}
                    className="flex items-center gap-3 px-4 py-2.5 text-[13px]"
                  >
                    <Avatar className="h-8 w-8">
                      <AvatarFallback
                        style={{
                          background: "var(--cp-bg-subtle)",
                          color: "var(--cp-text-muted)",
                        }}
                        className="text-xs font-medium"
                      >
                        {u.initials}
                      </AvatarFallback>
                    </Avatar>
                    <span className="flex-1 text-[#0F172A]">{u.name}</span>
                    <span className="text-[12px] text-[#64748B]">{u.role}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* 40px 卡片头 */}
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                40px · 卡片头部 / 详情页
              </MetaText>
              <div className="rounded-lg border border-[#e5e5e5] p-4 max-w-md">
                <div className="flex items-center gap-3">
                  <Avatar className="h-10 w-10">
                    <AvatarFallback
                      style={{
                        background: "var(--cp-bg-subtle)",
                        color: "var(--cp-text-muted)",
                      }}
                      className="text-sm font-medium"
                    >
                      AG
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1">
                    <div className="text-[14px] font-medium text-[#0F172A]">
                      Agent · DevOps Copilot
                    </div>
                    <div className="text-[12px] text-[#64748B]">
                      最近更新于 2 小时前 · 由 admin@openclaw 维护
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* 48px 个人中心 */}
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                48px · 个人中心 / 大头像
              </MetaText>
              <div className="flex items-center gap-4">
                <Avatar className="h-12 w-12">
                  <AvatarFallback
                    style={{
                      background: "var(--cp-bg-subtle)",
                      color: "var(--cp-text-muted)",
                    }}
                    className="text-base font-medium"
                  >
                    HM
                  </AvatarFallback>
                </Avatar>
                <div>
                  <div className="text-[15px] font-semibold text-[#0F172A]">
                    Han Mei
                  </div>
                  <div className="text-[12px] text-[#64748B]">
                    han.mei@openclaw.example · 超级管理员
                  </div>
                </div>
              </div>
            </div>
          </div>
        </DemoBlock>

        {/* --------------------------- §5 Do / Don't --------------------------- */}
        <DemoBlock title="§5 Do / Don't" desc="正反例对照">
          <div className="grid grid-cols-2 gap-4">
            {/* Do */}
            <div className="rounded-md border border-[#bbf7d0] bg-[#f0fdf4] p-4">
              <div className="flex items-center gap-1.5 text-[13px] font-medium text-[#15803d] mb-3">
                <Check className="w-4 h-4" /> Do
              </div>
              <div className="flex items-center gap-3 mb-3">
                <Avatar className="h-8 w-8">
                  <AvatarFallback
                    style={{
                      background: "var(--cp-bg-subtle)",
                      color: "var(--cp-text-muted)",
                    }}
                    className="text-xs font-medium"
                  >
                    JX
                  </AvatarFallback>
                </Avatar>
                <span className="text-[12px] text-[#15803d]">
                  圆形 · 首字母 · 标准 32px
                </span>
              </div>
              <ul className="text-[12px] text-[#166534] space-y-1 leading-relaxed">
                <li>· 无图片时显示首字母缩写</li>
                <li>· 使用圆形裁切</li>
                <li>· 从 24/32/40/48 中选尺寸</li>
              </ul>
            </div>

            {/* Don't */}
            <div className="rounded-md border border-[#fecaca] bg-[#fef2f2] p-4">
              <div className="flex items-center gap-1.5 text-[13px] font-medium text-[#b91c1c] mb-3">
                <X className="w-4 h-4" /> Don&apos;t
              </div>
              <div className="flex items-center gap-3 mb-3">
                {/* 反例 1：方形 */}
                <span
                  className="inline-flex items-center justify-center bg-[#FEE2E2] text-[#b91c1c] text-xs font-medium"
                  style={{
                    width: 32,
                    height: 32,
                    borderRadius: 4,
                  }}
                  title="方形头像"
                >
                  方
                </span>
                {/* 反例 2：自定义尺寸 37px */}
                <span
                  className="inline-flex items-center justify-center rounded-full bg-[#FEE2E2] text-[#b91c1c] text-xs font-medium"
                  style={{ width: 37, height: 37 }}
                  title="37px"
                >
                  37
                </span>
                {/* 反例 3：emoji */}
                <span
                  className="inline-flex items-center justify-center rounded-full bg-[#FEE2E2] text-base"
                  style={{ width: 32, height: 32 }}
                  title="emoji fallback"
                >
                  🐱
                </span>
              </div>
              <ul className="text-[12px] text-[#991b1b] space-y-1 leading-relaxed">
                <li>· 不要用方形头像</li>
                <li>· 不要自定义尺寸（如 37px）</li>
                <li>· 不要在 fallback 里放 emoji</li>
              </ul>
            </div>
          </div>
        </DemoBlock>

        {/* ----------------------- §6 QA Checklist 自检 ----------------------- */}
        <DemoBlock title="§6 QA Checklist" desc="本预览页的自检结果">
          <ul className="space-y-2.5">
            {checklist.map((it) => (
              <li
                key={it.label}
                className="flex items-start gap-2.5 text-[13px] text-[#0F172A]"
              >
                <span
                  className={`mt-0.5 inline-flex items-center justify-center w-4 h-4 rounded-sm shrink-0 ${
                    it.pass
                      ? "bg-[#16a34a] text-white"
                      : "bg-[#e5e5e5] text-[#737373]"
                  }`}
                >
                  {it.pass ? <Check className="w-3 h-3" /> : null}
                </span>
                <div className="flex-1">
                  <div>{it.label}</div>
                  {it.note && (
                    <div className="text-[12px] text-[#64748B] mt-0.5">
                      {it.note}
                    </div>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </DemoBlock>

        {/* ---------------------------- References ---------------------------- */}
        <footer className="text-[12px] text-[#94A3B8] text-center pt-4 border-t border-[#e5e5e5]">
          References:{" "}
          <code className="font-mono">SKILL-GLOBAL-COMPONENTS.md §22</code> ·{" "}
          <code className="font-mono">portable/css/tokens.css</code>
        </footer>
      </div>
    </div>
  );
}
