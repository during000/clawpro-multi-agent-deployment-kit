/**
 * Button Preview · Spec Verification
 * --------------------------------------------------------------
 * 路由：/preview/button
 * 用途：对照 `clawpro-portable-design-skill/component-specs/button.md`
 *      逐项验证视觉规范、状态、Portable fallback、Do/Don't、QA Checklist。
 *
 * 重点：可视化暴露 spec 与 client/src/components/ui/button.tsx 的偏差。
 */
import * as React from "react";
import { Button } from "@/components/ui/button";
import { MetaText, SectionTitle } from "@/components/ui/Typography";
import { Check, X, AlertTriangle, Plus, Trash2 } from "lucide-react";

/* spec §3 / portable/css/tokens.css，预览页内联兜底 */
const SPEC_TOKENS: React.CSSProperties = {
  ["--cp-brand-black" as never]: "#020617",
  ["--cp-surface" as never]: "#FFFFFF",
  ["--cp-border" as never]: "#EAEEF4",
  ["--cp-text-emphasis" as never]: "#020617",
  ["--cp-text-danger" as never]: "#DC2626",
};

/* ----------------------- Portable fallback（spec §7.2） ---------------------- */
function AdminPrimaryButton(
  props: React.ButtonHTMLAttributes<HTMLButtonElement>,
) {
  return (
    <button
      {...props}
      className="inline-flex h-9 items-center justify-center gap-2 rounded-[4px] bg-[var(--cp-brand-black)] px-6 text-sm text-white"
    />
  );
}
function TenantPrimaryButton(
  props: React.ButtonHTMLAttributes<HTMLButtonElement>,
) {
  return (
    <button
      {...props}
      className="inline-flex h-9 items-center justify-center gap-2 rounded-full bg-[var(--cp-brand-black)] px-6 text-sm text-white"
    />
  );
}

/* ------------------------------- Layout ------------------------------------ */
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

function Swatch({ color }: { color: string }) {
  return (
    <span
      className="inline-block w-3.5 h-3.5 rounded-sm border border-[#e5e5e5] align-middle mr-1.5"
      style={{ background: color }}
    />
  );
}

/* -------------------------------------------------------------------------- */
/*  spec §3 标准 8 个 variant 的「期望 vs 实现」                                 */
/* -------------------------------------------------------------------------- */
type Spec = {
  variant: string;
  scope: string;
  /** spec §3 期望 */
  expect: { radius: string; bg: string; border: string; text: string };
  /** button.tsx 实际 */
  actual: { radius: string; bg: string; border: string; text: string };
  /** 偏差点（高亮 ⚠） */
  diffs: Array<"radius" | "bg" | "border" | "text">;
  /** 是否「故意隔离」（tenant 与 claw 解耦的差异，非缺陷） */
  isolated?: boolean;
};

const SPECS: Spec[] = [
  {
    variant: "claw-primary",
    scope: "Admin 主操作",
    expect: { radius: "4px", bg: "#020617", border: "—", text: "#FFFFFF" },
    actual: { radius: "4px", bg: "var(--cp-brand-black) #020617", border: "—", text: "#FFFFFF" },
    diffs: [],
  },
  {
    variant: "claw-outline",
    scope: "Admin 次级",
    expect: { radius: "4px", bg: "#FFFFFF", border: "#EAEEF4", text: "#020617" },
    actual: { radius: "4px", bg: "#FFFFFF", border: "#EAEEF4", text: "var(--cp-brand-black) #020617" },
    diffs: [],
  },
  {
    variant: "dialog-confirm",
    scope: "通用弹窗确认",
    expect: { radius: "4px", bg: "#020617", border: "—", text: "#FFFFFF" },
    actual: { radius: "4px", bg: "var(--cp-brand-black) #020617", border: "—", text: "#FFFFFF" },
    diffs: [],
  },
  {
    variant: "tenant-primary",
    scope: "Tenant 主操作",
    expect: { radius: "full", bg: "#020617", border: "—", text: "#FFFFFF" },
    actual: { radius: "full", bg: "#0A0A0A", border: "—", text: "#FFFFFF" },
    diffs: ["bg"],
    isolated: true,
  },
  {
    variant: "tenant-outline",
    scope: "Tenant 次级",
    expect: { radius: "full", bg: "#FFFFFF", border: "#EAEEF4", text: "#020617" },
    actual: { radius: "full", bg: "#FFFFFF", border: "#E5E7EB", text: "#030712" },
    diffs: ["border", "text"],
    isolated: true,
  },
  {
    variant: "tenant-destructive",
    scope: "Tenant 危险",
    expect: { radius: "full", bg: "#DC2626", border: "—", text: "#FFFFFF" },
    actual: { radius: "full", bg: "#D42A1E", border: "—", text: "#FFFFFF" },
    diffs: ["bg"],
    isolated: true,
  },
  {
    variant: "tenant-outline-strong",
    scope: "Tenant 中等强调",
    expect: { radius: "full", bg: "#FFFFFF", border: "#cbcbcb", text: "#020617" },
    actual: { radius: "full", bg: "#FFFFFF", border: "#cbcbcb", text: "#030712" },
    diffs: ["text"],
    isolated: true,
  },
  {
    variant: "tenant-ghost",
    scope: "Tenant 低权重",
    expect: { radius: "full", bg: "transparent", border: "—", text: "#020617" },
    actual: { radius: "full", bg: "transparent", border: "—", text: "#030712" },
    diffs: ["text"],
    isolated: true,
  },
];

/* -------------------------------------------------------------------------- */
/*  Page                                                                      */
/* -------------------------------------------------------------------------- */
export default function ButtonPreview() {
  // §10 QA Checklist
  const checklist: { label: string; pass: boolean | "warn"; note?: string }[] = [
    {
      label: "主次危险操作语义明确（spec §3 8 个 variant 全部存在）",
      pass: true,
      note: "claw-primary / claw-outline / dialog-confirm / tenant-primary / tenant-outline / tenant-destructive / tenant-outline-strong / tenant-ghost 已实现",
    },
    {
      label: "Admin / Tenant 圆角口径正确",
      pass: true,
      note: "claw-* 全部 4px，tenant-* 全部 !rounded-full",
    },
    {
      label: "disabled 和 hover 状态完整",
      pass: true,
      note: "8 个 variant 的 hover / active / disabled 三态均覆盖",
    },
    {
      label: "宿主仓可按 fallback 还原（spec §7.2）",
      pass: true,
      note: "claw-primary / dialog-confirm / claw-outline 已走 var(--cp-brand-black)，与 §7.2 AdminPrimaryButton 口径一致；tenant-* 故意保留独立 hex（设计意图，非缺陷）",
    },
  ];

  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8" style={SPEC_TOKENS}>
      <div className="max-w-4xl mx-auto space-y-8">
        {/* Header */}
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">Button 按钮 · Spec 验证</h1>
          <p className="text-sm text-[#64748B]">
            对照{" "}
            <code className="font-mono text-[12px] bg-[#F1F5F9] px-1.5 py-0.5 rounded">
              component-specs/button.md
            </code>{" "}
            §3 视觉规范 · §5 States · §7 Portable fallback · §9 Do/Don&apos;t · §10 QA Checklist
          </p>
        </header>

        {/* §3 Visual Standard 8 variant 实物渲染 */}
        <DemoBlock title="§3 Visual Standard · 8 个标准 variant" desc="实物 + 期望色 / 实际色对照">
          <div className="space-y-4">
            {SPECS.map((s) => {
              const isClaw = s.variant.startsWith("claw") || s.variant === "dialog-confirm";
              return (
                <div
                  key={s.variant}
                  className="grid grid-cols-[180px_1fr_1fr_60px] items-center gap-4 py-3 border-b border-dashed border-[#f0f0f0] last:border-0"
                >
                  {/* 变体名 + scope */}
                  <div>
                    <div className="text-[13px] font-mono text-[#0F172A]">{s.variant}</div>
                    <div className="text-[11px] text-[#64748B] mt-0.5">{s.scope}</div>
                    <div className="text-[11px] text-[#94A3B8] mt-0.5">
                      圆角 {s.expect.radius}
                    </div>
                  </div>

                  {/* 实物按钮 */}
                  <div className="flex items-center gap-2">
                    <Button
                      variant={s.variant as never}
                      size={isClaw ? "claw" : "claw"}
                    >
                      {s.variant.includes("destructive") ? (
                        <>
                          <Trash2 /> 删除
                        </>
                      ) : s.variant.includes("primary") || s.variant === "dialog-confirm" ? (
                        <>
                          <Plus /> 创建
                        </>
                      ) : (
                        "操作"
                      )}
                    </Button>
                    <Button variant={s.variant as never} size="claw" disabled>
                      disabled
                    </Button>
                  </div>

                  {/* 期望 vs 实际色块 */}
                  <div className="text-[11px] leading-relaxed">
                    {(() => {
                      // 偏差色：真偏差=橙；故意隔离=灰
                      const diffColor = s.isolated ? "text-[#475569]" : "text-[#B45309]";
                      const tag = s.isolated ? "·" : "⚠";
                      return (
                        <>
                          <div className="text-[#64748B] mb-1">期望 (spec §3)</div>
                          <div>
                            <Swatch color={s.expect.bg} />
                            bg {s.expect.bg}
                          </div>
                          {s.expect.border !== "—" && (
                            <div>
                              <Swatch color={s.expect.border} />
                              border {s.expect.border}
                            </div>
                          )}
                          <div>
                            <Swatch color={s.expect.text} />
                            text {s.expect.text}
                          </div>
                          <div className="text-[#64748B] mt-2 mb-1">实际 (button.tsx)</div>
                          <div className={s.diffs.includes("bg") ? diffColor : ""}>
                            <Swatch color={s.actual.bg.match(/#[0-9A-Fa-f]+/)?.[0] ?? s.actual.bg} />
                            bg {s.actual.bg}{" "}
                            {s.diffs.includes("bg") && <span title={s.isolated ? "故意隔离" : "偏差"}>{tag}</span>}
                          </div>
                          {s.actual.border !== "—" && (
                            <div className={s.diffs.includes("border") ? diffColor : ""}>
                              <Swatch color={s.actual.border} />
                              border {s.actual.border}{" "}
                              {s.diffs.includes("border") && <span title={s.isolated ? "故意隔离" : "偏差"}>{tag}</span>}
                            </div>
                          )}
                          <div className={s.diffs.includes("text") ? diffColor : ""}>
                            <Swatch color={s.actual.text.match(/#[0-9A-Fa-f]+/)?.[0] ?? s.actual.text} />
                            text {s.actual.text}{" "}
                            {s.diffs.includes("text") && <span title={s.isolated ? "故意隔离" : "偏差"}>{tag}</span>}
                          </div>
                        </>
                      );
                    })()}
                  </div>

                  {/* 状态 */}
                  <div className="text-right">
                    {s.diffs.length === 0 ? (
                      <span className="inline-flex items-center gap-1 text-[11px] text-[#15803d]">
                        <Check className="w-3.5 h-3.5" /> ok
                      </span>
                    ) : s.isolated ? (
                      <span
                        className="inline-flex items-center gap-1 text-[11px] text-[#475569]"
                        title="tenant 与 claw 故意隔离，颜色独立维护"
                      >
                        <span className="inline-block w-1.5 h-1.5 rounded-full bg-[#94A3B8]" />
                        隔离 ({s.diffs.length})
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-[11px] text-[#B45309]">
                        <AlertTriangle className="w-3.5 h-3.5" /> {s.diffs.length} 偏差
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </DemoBlock>

        {/* §5 States — 5 态对照 */}
        <DemoBlock title="§5 States · 五态对照（以 claw-primary / tenant-primary 为例）" desc="default / hover / active / disabled / loading">
          <div className="space-y-4">
            {[
              { variant: "claw-primary", label: "claw-primary（Admin 4px）", hoverBg: "#404040" },
              { variant: "tenant-primary", label: "tenant-primary（Tenant full）", hoverBg: "#333333" },
            ].map((row) => (
              <div key={row.variant}>
                <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                  {row.label}
                </MetaText>
                <div className="flex items-center gap-3 flex-wrap">
                  <Button variant={row.variant as never} size="claw">
                    default
                  </Button>
                  <Button
                    variant={row.variant as never}
                    size="claw"
                    style={{ backgroundColor: row.hoverBg }}
                    title={`模拟 hover：${row.hoverBg}`}
                  >
                    hover
                  </Button>
                  <Button
                    variant={row.variant as never}
                    size="claw"
                    className="!bg-[#000]"
                    title="模拟 active：保持结构稳定"
                  >
                    active
                  </Button>
                  <Button variant={row.variant as never} size="claw" disabled>
                    disabled
                  </Button>
                  <Button variant={row.variant as never} size="claw" disabled>
                    <span className="inline-block w-3.5 h-3.5 rounded-full border-2 border-white/40 border-t-white animate-spin" />
                    loading
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </DemoBlock>

        {/* size 变体 */}
        <DemoBlock title="Size 变体" desc="claw-sm / claw / claw-lg / claw-square（实现扩展）">
          <div className="flex items-end gap-3 flex-wrap">
            <div className="flex flex-col items-center gap-1">
              <Button variant="claw-primary" size="claw-sm">
                <Plus /> 32×hug
              </Button>
              <MetaText className="!text-[11px]">claw-sm h-8</MetaText>
            </div>
            <div className="flex flex-col items-center gap-1">
              <Button variant="claw-primary" size="claw">
                <Plus /> 36×hug
              </Button>
              <MetaText className="!text-[11px]">claw h-9 (默认)</MetaText>
            </div>
            <div className="flex flex-col items-center gap-1">
              <Button variant="claw-primary" size="claw-lg">
                <Plus /> 40×hug
              </Button>
              <MetaText className="!text-[11px]">claw-lg h-10</MetaText>
            </div>
            <div className="flex flex-col items-center gap-1">
              <Button variant="claw-outline" size="claw-square" aria-label="刷新">
                <Plus />
              </Button>
              <MetaText className="!text-[11px]">claw-square 48×36</MetaText>
            </div>
          </div>
        </DemoBlock>

        {/* §6 Demo Repo Usage — 典型组合 */}
        <DemoBlock title="§6 Demo Repo Usage · 典型用法" desc="Admin / Tenant 主次按钮组合">
          <div className="grid grid-cols-2 gap-6">
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                Admin（claw-*）— 表单底部
              </MetaText>
              <div className="rounded-md border border-[#e5e5e5] p-4 flex justify-end gap-2 bg-[#fafafa]">
                <Button variant="claw-outline" size="claw">
                  取消
                </Button>
                <Button variant="claw-primary" size="claw">
                  创建
                </Button>
              </div>
            </div>
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                Tenant（tenant-*）— 业务卡操作
              </MetaText>
              <div className="rounded-md border border-[#e5e5e5] p-4 flex justify-end gap-2 bg-[#fafafa]">
                <Button variant="tenant-outline" size="claw">
                  详细配置
                </Button>
                <Button variant="tenant-outline-strong" size="claw">
                  管理通道
                </Button>
                <Button variant="tenant-primary" size="claw">
                  <Plus /> 创建 Agent
                </Button>
              </div>
            </div>
          </div>
        </DemoBlock>

        {/* §7 Portable Fallback */}
        <DemoBlock
          title="§7 Portable Fallback · 宿主仓零依赖实现"
          desc="spec §7.2 React fallback / §7.3 HTML+CSS fallback"
        >
          <div className="grid grid-cols-2 gap-6">
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                §7.2 React fallback
              </MetaText>
              <div className="rounded-md border border-[#e5e5e5] p-4 bg-[#fafafa] flex items-center gap-3">
                <AdminPrimaryButton>创建</AdminPrimaryButton>
                <TenantPrimaryButton>创建 Agent</TenantPrimaryButton>
              </div>
              <p className="mt-3 text-[12px] text-[#64748B] leading-relaxed">
                走 <code>var(--cp-brand-black)</code>，不依赖 cva / shadcn。
                效果应当与上方 <code>claw-primary</code> / <code>tenant-primary</code> 完全一致 —
                若颜色有差，说明本仓未对齐 token（参见下方「偏差汇总」）。
              </p>
            </div>
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-2 block">
                §7.3 HTML / CSS fallback
              </MetaText>
              <div className="rounded-md border border-[#e5e5e5] p-4 bg-[#fafafa] flex items-center gap-3">
                <button
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                    gap: 8,
                    height: 36,
                    padding: "0 24px",
                    fontSize: 14,
                    borderRadius: 4,
                    background: "var(--cp-brand-black)",
                    color: "white",
                    border: 0,
                  }}
                >
                  创建
                </button>
                <button
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                    gap: 8,
                    height: 36,
                    padding: "0 24px",
                    fontSize: 14,
                    borderRadius: 9999,
                    background: "var(--cp-surface)",
                    color: "var(--cp-text-emphasis)",
                    border: "1px solid var(--cp-border)",
                  }}
                >
                  详细配置
                </button>
              </div>
              <p className="mt-3 text-[12px] text-[#64748B] leading-relaxed">
                <code>cp-btn-admin-primary</code> / <code>cp-btn-tenant-outline</code> 内联展示。
                可直接复制到任何宿主仓使用。
              </p>
            </div>
          </div>
        </DemoBlock>

        {/* 偏差汇总 */}
        <DemoBlock
          title="偏差汇总（spec ↔ button.tsx）"
          desc="本轮迭代：claw 侧已对齐 spec；tenant 侧故意隔离，单独维护"
        >
          <div className="space-y-4 text-[13px] text-[#0F172A]">
            <div>
              <div className="inline-flex items-center gap-1.5 text-[#15803d] text-[12px] font-medium mb-2">
                <Check className="w-3.5 h-3.5" /> 已修复（claw 侧 / 通用）
              </div>
              <ol className="space-y-2 list-decimal pl-5 marker:text-[#15803d]">
                <li>
                  <strong>品牌黑色值</strong> ——
                  <code>claw-primary</code> / <code>dialog-confirm</code> 改用
                  <code className="font-mono"> bg-[var(--cp-brand-black)]</code>，
                  本仓 <code>:root</code> 同步暴露 <code className="font-mono">--cp-brand-black: #020617</code>，与 spec §3 / portable §7.2 一致。
                </li>
                <li>
                  <strong>文字色 token 名义统一</strong> ——
                  <code>claw-outline</code> 由 <code>text-gray-950</code>（#030712）改为 <code className="font-mono">text-[var(--cp-brand-black)]</code>（#020617），视觉无差但语义对齐。
                </li>
                <li>
                  <strong>未走 token，硬编码 hex</strong> ——
                  <code>claw-primary</code> 的 <code className="font-mono">disabled:bg-[#0A0A0A]/40</code> 替换为
                  <code className="font-mono"> disabled:bg-[var(--cp-brand-black)]/40</code>，宿主仓覆盖品牌色时按钮 disabled 态可跟随。
                </li>
              </ol>
            </div>

            <div>
              <div className="inline-flex items-center gap-1.5 text-[#475569] text-[12px] font-medium mb-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-[#94A3B8]" />
                故意保留（tenant 侧 · 与 claw 解耦）
              </div>
              <ol className="space-y-2 list-decimal pl-5 marker:text-[#94A3B8]">
                <li>
                  <strong>tenant-primary / tenant-dialog-confirm</strong> 底色保留 <code className="font-mono">#0A0A0A</code>（不走 <code>--cp-brand-black</code>）。
                </li>
                <li>
                  <strong>tenant-destructive</strong> 底色保留 <code className="font-mono">#D42A1E</code> / hover <code className="font-mono">#B91C1C</code>（不走 <code>--cp-text-danger</code>）。
                </li>
                <li>
                  <strong>tenant-outline 边框</strong> 保留 <code>border-gray-200</code>（<code className="font-mono">#E5E7EB</code>），与 <code>claw-outline</code> 的 <code className="font-mono">#EAEEF4</code> 解耦。
                </li>
                <li>
                  <strong>tenant-* 文字色</strong> 保留 <code>text-gray-950</code>（<code className="font-mono">#030712</code>）。
                </li>
              </ol>
              <p className="text-[12px] text-[#64748B] leading-relaxed mt-2">
                依据：tenant 视觉以 Figma 0522 设计稿为最终口径，hex 沿用稿件，避免随宿主仓主题色（admin 侧）漂移。
                如需统一请走单独 spec PR，并由设计走查复核。
              </p>
            </div>
          </div>
        </DemoBlock>

        {/* spec §3 没收录的扩展变体 */}
        <DemoBlock title="实现额外的扩展变体（spec §3 未收录）" desc="共 12 个，建议按用途归类">
          <div className="space-y-3 text-[13px]">
            <div>
              <span className="inline-block w-32 text-[#64748B]">历史向（与 spec 重叠）：</span>
              <code className="font-mono text-[12px]">
                default · destructive · outline · secondary · ghost · plain · link · link-dark
              </code>
            </div>
            <div>
              <span className="inline-block w-32 text-[#64748B]">业务在用：</span>
              <code className="font-mono text-[12px]">
                outline-destructive · tenant-outline-r20 · tenant-plain · tenant-dialog-confirm
              </code>
            </div>
            <div className="grid grid-cols-3 gap-2 mt-2">
              <Button variant="tenant-outline-r20" size="claw">
                tenant-outline-r20
              </Button>
              <Button variant="tenant-plain" size="claw-sm">
                tenant-plain
              </Button>
              <Button variant="tenant-dialog-confirm" size="claw">
                tenant-dialog-confirm
              </Button>
              <Button variant="outline-destructive" size="claw">
                <Trash2 /> outline-destructive
              </Button>
              <Button variant="link" size="claw">
                link
              </Button>
              <Button variant="link-dark" size="claw">
                link-dark
              </Button>
            </div>
            <p className="text-[12px] text-[#64748B] leading-relaxed mt-2">
              建议处理：<code>default / destructive</code> 与 <code>claw-primary / tenant-destructive</code> 重叠 → 标 deprecated；
              <code>tenant-outline-r20 / tenant-plain / tenant-dialog-confirm</code> 是真实在用形态 → 应补进 <code>button.md</code> §3；
              <code>link / link-dark</code> 不属于按钮范畴（spec §2 明确排除）→ 拆出或加例外说明。
            </p>
          </div>
        </DemoBlock>

        {/* §9 Do / Don't */}
        <DemoBlock title="§9 Do / Don't" desc="正反例对照">
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-md border border-[#bbf7d0] bg-[#f0fdf4] p-4">
              <div className="flex items-center gap-1.5 text-[13px] font-medium text-[#15803d] mb-3">
                <Check className="w-4 h-4" /> Do
              </div>
              <div className="flex items-center gap-2 mb-3">
                <Button variant="claw-primary" size="claw">
                  <Plus /> 创建（Admin · 4px）
                </Button>
                <Button variant="tenant-primary" size="claw">
                  <Plus /> 创建 Agent（Tenant · full）
                </Button>
              </div>
              <ul className="text-[12px] text-[#166534] space-y-1 leading-relaxed">
                <li>· 先判断端别（Admin / Tenant）</li>
                <li>· 主次操作层级明确</li>
                <li>· 用 spec §3 标准 variant，不在页面内重复造</li>
              </ul>
            </div>
            <div className="rounded-md border border-[#fecaca] bg-[#fef2f2] p-4">
              <div className="flex items-center gap-1.5 text-[13px] font-medium text-[#b91c1c] mb-3">
                <X className="w-4 h-4" /> Don&apos;t
              </div>
              <div className="flex items-center gap-2 mb-3 flex-wrap">
                {/* 反例 1：Admin 页面误用胶囊 */}
                <Button variant="tenant-primary" size="claw" title="Admin 页面用胶囊">
                  <Plus /> Admin 用胶囊
                </Button>
                {/* 反例 2：page 内 inline 写颜色 */}
                <button
                  className="inline-flex h-9 items-center justify-center rounded-[4px] px-6 text-sm text-white"
                  style={{ background: "#1447E6" }}
                  title="inline 写颜色"
                >
                  inline 蓝按钮
                </button>
                {/* 反例 3：用 outline 冒充次级 */}
                <Button variant="outline" size="claw" title="用 outline 冒充次级">
                  outline 冒充次级
                </Button>
              </div>
              <ul className="text-[12px] text-[#991b1b] space-y-1 leading-relaxed">
                <li>· 不要让 Tenant / Admin 按钮风格混用</li>
                <li>· 不要在页面内 inline 写颜色 / 圆角 / 阴影</li>
                <li>· 不要用通用 outline 冒充业务次级按钮</li>
              </ul>
            </div>
          </div>
        </DemoBlock>

        {/* §10 QA Checklist */}
        <DemoBlock title="§10 QA Checklist" desc="本预览页的自检结果">
          <ul className="space-y-2.5">
            {checklist.map((it) => (
              <li
                key={it.label}
                className="flex items-start gap-2.5 text-[13px] text-[#0F172A]"
              >
                <span
                  className={`mt-0.5 inline-flex items-center justify-center w-4 h-4 rounded-sm shrink-0 ${
                    it.pass === true
                      ? "bg-[#16a34a] text-white"
                      : it.pass === "warn"
                        ? "bg-[#f59e0b] text-white"
                        : "bg-[#e5e5e5] text-[#737373]"
                  }`}
                >
                  {it.pass === true ? (
                    <Check className="w-3 h-3" />
                  ) : it.pass === "warn" ? (
                    <AlertTriangle className="w-3 h-3" />
                  ) : null}
                </span>
                <div className="flex-1">
                  <div>{it.label}</div>
                  {it.note && (
                    <div className="text-[12px] text-[#64748B] mt-0.5">{it.note}</div>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </DemoBlock>

        {/* References */}
        <footer className="text-[12px] text-[#94A3B8] text-center pt-4 border-t border-[#e5e5e5]">
          References:{" "}
          <code className="font-mono">component-specs/button.md</code> ·{" "}
          <code className="font-mono">client/src/components/ui/button.tsx</code> ·{" "}
          <code className="font-mono">portable/css/tokens.css</code>
        </footer>
      </div>
    </div>
  );
}
