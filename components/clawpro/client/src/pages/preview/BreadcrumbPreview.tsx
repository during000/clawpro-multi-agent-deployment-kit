/**
 * Breadcrumb Preview · Spec Verification
 * --------------------------------------------------------------
 * 路由：/preview/breadcrumb
 * 用途：对照 `clawpro-portable-design-skill/component-specs/breadcrumb.md`
 *      逐项验证视觉规范、Portable fallback、Do/Don't、QA Checklist。
 */
import React from "react";
import { ChevronRight } from "lucide-react";
import { Check, X, AlertTriangle } from "lucide-react";

import { MetaText, SectionTitle } from "@/components/ui/Typography";

/* -------------------------------------------------------------------------- */
/*  Spec tokens（spec 引用，client 全局尚未注入，预览页内联兜底）             */
/* -------------------------------------------------------------------------- */
const SPEC_TOKENS: React.CSSProperties = {
  // 来源：clawpro-portable-design-skill/portable/css/tokens.css
  ["--cp-text-title" as never]: "#0F172A",
  ["--cp-text-muted" as never]: "#64748B",
  ["--cp-text-weak" as never]: "#94A3B8",
};

/* -------------------------------------------------------------------------- */
/*  Portable Breadcrumb（spec §4 1:1 复刻）                                    */
/* -------------------------------------------------------------------------- */
function PortableBreadcrumb({
  items,
}: {
  items: { label: string; href?: string }[];
}) {
  return (
    <nav className="flex items-center gap-1.5 text-sm">
      {items.map((item, i) => (
        <React.Fragment key={i}>
          {i > 0 && <span className="text-[var(--cp-text-weak)]">/</span>}
          {item.href ? (
            <a
              href={item.href}
              className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)] transition-colors"
            >
              {item.label}
            </a>
          ) : (
            <span className="font-medium text-[var(--cp-text-title)]">
              {item.label}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
  );
}

/* -------------------------------------------------------------------------- */
/*  Breadcrumb 变体：分隔符可选 "/" | ">" | <ChevronRight />                  */
/* -------------------------------------------------------------------------- */
type SeparatorKind = "slash" | "gt" | "chevron";

function Breadcrumb({
  items,
  separator = "chevron",
}: {
  items: { label: string; href?: string }[];
  separator?: SeparatorKind;
}) {
  const renderSep = () => {
    if (separator === "chevron") {
      return (
        <ChevronRight className="w-3.5 h-3.5 text-[var(--cp-text-weak)]" />
      );
    }
    return (
      <span className="text-[var(--cp-text-weak)]">
        {separator === "slash" ? "/" : ">"}
      </span>
    );
  };

  return (
    <nav className="flex items-center gap-1.5 text-sm" aria-label="Breadcrumb">
      {items.map((item, i) => (
        <React.Fragment key={i}>
          {i > 0 && renderSep()}
          {item.href ? (
            <a
              href={item.href}
              className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)] transition-colors"
            >
              {item.label}
            </a>
          ) : (
            <span
              aria-current="page"
              className="font-medium text-[var(--cp-text-title)]"
            >
              {item.label}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
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
      <div className="p-5 space-y-4">{children}</div>
    </section>
  );
}

function SpecRow({
  label,
  value,
  note,
}: {
  label: string;
  value: React.ReactNode;
  note?: React.ReactNode;
}) {
  return (
    <div className="py-2 border-b border-dashed border-[#f0f0f0] last:border-0">
      <div className="flex items-center justify-between gap-4">
        <span className="text-[13px] text-[#404040]">{label}</span>
        <span className="text-[13px] font-mono text-[#0F172A] text-right">
          {value}
        </span>
      </div>
      {note ? (
        <div className="mt-1 text-[12px] text-[#b45309] flex items-start gap-1">
          <AlertTriangle className="w-3 h-3 mt-[3px] shrink-0" />
          <span>{note}</span>
        </div>
      ) : null}
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/*  Page                                                                      */
/* -------------------------------------------------------------------------- */
export default function BreadcrumbPreview() {
  // QA Checklist（spec §6）
  const checklist: { label: string; pass: boolean; note?: string }[] = [
    {
      label: "当前页不可点击（无 href + font-medium + 深色）",
      pass: true,
      note: "使用 <span aria-current=\"page\">",
    },
    {
      label: "祖先页可点击且有 hover 态",
      pass: true,
      note: "灰色 → hover 切到 --cp-text-title",
    },
    { label: "分隔符使用弱色（--cp-text-weak）", pass: true },
    {
      label: "fallback 使用 var(--cp-*) CSS variable",
      pass: true,
      note: "Portable 与本页变体均经由 var(--cp-*) 渲染",
    },
  ];

  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8" style={SPEC_TOKENS}>
      <div className="max-w-3xl mx-auto space-y-8">
        {/* Header */}
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">
            Breadcrumb 面包屑 · Spec 验证
          </h1>
          <p className="text-sm text-[#64748B]">
            对照{" "}
            <code className="font-mono text-[12px] bg-[#F1F5F9] px-1.5 py-0.5 rounded">
              component-specs/breadcrumb.md
            </code>{" "}
            §3 视觉规范 · §4 Portable fallback · §5 Do/Don&apos;t · §6 QA
            Checklist
          </p>
        </header>

        {/* §3 Visual Standard */}
        <DemoBlock title="§3 Visual Standard" desc="参数对照表">
          <div>
            <SpecRow label="字号" value="14px (text-sm)" />
            <SpecRow
              label="当前页文字"
              value={
                <span className="inline-flex items-center gap-2">
                  <span className="font-medium text-[var(--cp-text-title)]">
                    示例
                  </span>
                  var(--cp-text-title) #0F172A
                </span>
              }
            />
            <SpecRow
              label="祖先页文字"
              value={
                <span className="inline-flex items-center gap-2">
                  <span className="text-[var(--cp-text-muted)]">示例</span>
                  var(--cp-text-muted) #64748B
                </span>
              }
              note={
                <>
                  spec md 表格中标注为 <code>#737373</code>，但{" "}
                  <code>portable/css/tokens.css</code> 实际值为{" "}
                  <code>#64748B</code>，本页采用 token 实际值。建议同步修正
                  spec 文档。
                </>
              }
            />
            <SpecRow
              label="祖先页 hover"
              value="→ var(--cp-text-title)"
            />
            <SpecRow
              label="分隔符"
              value={
                <span className="inline-flex items-center gap-2">
                  <span className="text-[var(--cp-text-weak)]">/</span>
                  /
                  <span className="text-[var(--cp-text-weak)]">&gt;</span>
                  · var(--cp-text-weak) #94A3B8
                </span>
              }
            />
            <SpecRow label="间距" value="gap-1.5 (6px)" />
          </div>
        </DemoBlock>

        {/* §4 Portable Fallback —— 1:1 复刻 */}
        <DemoBlock
          title="§4 Portable Fallback"
          desc="spec 提供的最小实现（仅 a/span，零依赖）"
        >
          <PortableBreadcrumb
            items={[
              { label: "首页", href: "/" },
              { label: "Agent 列表", href: "/agent" },
              { label: "DevOps Copilot" },
            ]}
          />
          <p className="text-[12px] text-[#64748B] leading-relaxed">
            完整使用 <code>var(--cp-text-muted)</code> /{" "}
            <code>var(--cp-text-title)</code> / <code>var(--cp-text-weak)</code>，
            分隔符为字符 <code>/</code>。
          </p>
        </DemoBlock>

        {/* 分隔符变体 */}
        <DemoBlock
          title="分隔符变体"
          desc='"/" / ">" / <ChevronRight /> 三种'
        >
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <span className="text-[12px] text-[#64748B] w-20 shrink-0">
                slash
              </span>
              <Breadcrumb
                separator="slash"
                items={[
                  { label: "首页", href: "/" },
                  { label: "用户管理", href: "/users" },
                  { label: "权限分配" },
                ]}
              />
            </div>
            <div className="flex items-center gap-3">
              <span className="text-[12px] text-[#64748B] w-20 shrink-0">
                gt
              </span>
              <Breadcrumb
                separator="gt"
                items={[
                  { label: "首页", href: "/" },
                  { label: "用户管理", href: "/users" },
                  { label: "权限分配" },
                ]}
              />
            </div>
            <div className="flex items-center gap-3">
              <span className="text-[12px] text-[#64748B] w-20 shrink-0">
                chevron
              </span>
              <Breadcrumb
                separator="chevron"
                items={[
                  { label: "首页", href: "/" },
                  { label: "用户管理", href: "/users" },
                  { label: "权限分配" },
                ]}
              />
            </div>
          </div>
        </DemoBlock>

        {/* 层级数变体 */}
        <DemoBlock title="层级数变体" desc="2 / 3 / 4 级">
          <div className="space-y-3">
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-1.5 block">
                2 级（详情页最浅层级）
              </MetaText>
              <Breadcrumb
                items={[
                  { label: "首页", href: "/" },
                  { label: "用户管理" },
                ]}
              />
            </div>
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-1.5 block">
                3 级（列表 → 详情）
              </MetaText>
              <Breadcrumb
                items={[
                  { label: "首页", href: "/" },
                  { label: "Agent 列表", href: "/agent" },
                  { label: "Alice 的技术助手" },
                ]}
              />
            </div>
            <div>
              <MetaText tone="weak" className="!text-[11px] !mb-1.5 block">
                4 级（深层嵌套）
              </MetaText>
              <Breadcrumb
                items={[
                  { label: "首页", href: "/" },
                  { label: "安全管理", href: "/security" },
                  { label: "AI Agent", href: "/security/aiagent" },
                  { label: "策略详情" },
                ]}
              />
            </div>
          </div>
        </DemoBlock>

        {/* 真实业务场景 */}
        <DemoBlock title="真实业务场景" desc="放在页面 PageHeader 之上的常见用法">
          <div className="rounded-md border border-[#e5e5e5] bg-white p-5">
            <Breadcrumb
              items={[
                { label: "管理端", href: "/admin" },
                { label: "会话管理", href: "/admin/session-management" },
                { label: "会话详情 #SES-2026-0609-0231" },
              ]}
            />
            <div className="mt-3 flex items-baseline justify-between">
              <h2 className="text-[18px] font-semibold text-[var(--cp-text-title)]">
                会话详情 #SES-2026-0609-0231
              </h2>
              <span className="text-[12px] text-[var(--cp-text-muted)]">
                创建于 2026-06-09 14:32 · 由 alice@openclaw 发起
              </span>
            </div>
          </div>
        </DemoBlock>

        {/* §5 Do / Don't */}
        <DemoBlock title="§5 Do / Don't" desc="正反例对照">
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-md border border-[#bbf7d0] bg-[#f0fdf4] p-4">
              <div className="flex items-center gap-1.5 text-[13px] font-medium text-[#15803d] mb-3">
                <Check className="w-4 h-4" /> Do
              </div>
              <Breadcrumb
                items={[
                  { label: "首页", href: "/" },
                  { label: "Agent 列表", href: "/agent" },
                  { label: "DevOps Copilot" },
                ]}
              />
              <ul className="mt-3 text-[12px] text-[#166534] space-y-1 leading-relaxed">
                <li>· 当前页不可点击 + font-medium + 深色</li>
                <li>· 祖先页灰色可点击，hover 变深</li>
                <li>· 无背景、无边框，纯文本</li>
              </ul>
            </div>

            <div className="rounded-md border border-[#fecaca] bg-[#fef2f2] p-4 space-y-3">
              <div className="flex items-center gap-1.5 text-[13px] font-medium text-[#b91c1c]">
                <X className="w-4 h-4" /> Don&apos;t
              </div>

              {/* 反例 1：加背景/边框 */}
              <div>
                <div className="text-[11px] text-[#991b1b] mb-1">
                  ✗ 加背景色 / 边框
                </div>
                <nav className="inline-flex items-center gap-1.5 text-sm rounded-full border border-[#fca5a5] bg-white px-3 py-1.5">
                  <a className="text-[#737373]" href="#">
                    首页
                  </a>
                  <span className="text-[#94A3B8]">/</span>
                  <span className="font-medium text-[#0F172A]">用户管理</span>
                </nav>
              </div>

              {/* 反例 2：仅一级 */}
              <div>
                <div className="text-[11px] text-[#991b1b] mb-1">
                  ✗ 仅一级也显示面包屑
                </div>
                <Breadcrumb items={[{ label: "首页" }]} />
              </div>

              {/* 反例 3：当前页可点击 */}
              <div>
                <div className="text-[11px] text-[#991b1b] mb-1">
                  ✗ 当前页可点击且非加粗
                </div>
                <nav className="flex items-center gap-1.5 text-sm">
                  <a className="text-[#737373]" href="#">
                    首页
                  </a>
                  <span className="text-[#94A3B8]">/</span>
                  <a className="text-[#737373] underline" href="#">
                    用户管理（当前页）
                  </a>
                </nav>
              </div>
            </div>
          </div>
        </DemoBlock>

        {/* §6 QA Checklist */}
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
                  {it.note ? (
                    <div className="text-[12px] text-[#64748B] mt-0.5">
                      {it.note}
                    </div>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        </DemoBlock>

        {/* References */}
        <footer className="text-[12px] text-[#94A3B8] text-center pt-4 border-t border-[#e5e5e5]">
          References:{" "}
          <code className="font-mono">component-specs/page-header.md</code> ·{" "}
          <code className="font-mono">component-specs/navigation-sidebar.md</code>
        </footer>
      </div>
    </div>
  );
}
