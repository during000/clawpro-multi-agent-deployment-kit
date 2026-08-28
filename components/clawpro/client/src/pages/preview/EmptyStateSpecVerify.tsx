/**
 * Empty State Spec Verify
 *
 * 路由：/preview/empty-state-spec-verify
 *
 * 目的：验证 component-specs/empty-state.md 翻译后的规范是否覆盖所有场景。
 * 本页面使用 portable fallback 方式实现（不依赖 demo 仓 Empty 组件），
 * 用于校验外部前端拿到 portable spec 后能否独立还原。
 *
 * 不影响原有 EmptyStatePreview.tsx。
 */
import React from "react";

/* ====================================================================
   CSS Variables 模拟（正式使用时由 tokens.css 提供）
   ==================================================================== */
const TOKEN_STYLE: React.CSSProperties = {
  // 这些变量模拟 portable/css/tokens.css 中的定义
  ["--cp-text-title" as string]: "#0F172A",
  ["--cp-text-weak" as string]: "#94A3B8",
  ["--cp-text-brand" as string]: "#1447E6",
  ["--cp-border" as string]: "#EAEEF4",
  ["--cp-bg-surface" as string]: "#FFFFFF",
};

/* ====================================================================
   Portable Fallback 组件（来自 spec §9）
   ==================================================================== */

/** 页面级 / 卡片级空态 */
function PortableEmpty({
  title,
  description,
  action,
}: {
  title?: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-12 text-center">
      <img
        src="/assets/admin-sidebar/empty-aiagent.png"
        alt=""
        className="h-20 w-[100px] object-contain"
      />
      <div className="flex max-w-sm flex-col items-center gap-1">
        {title && (
          <p className="text-sm font-medium" style={{ color: "var(--cp-text-title)" }}>
            {title}
          </p>
        )}
        <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>
          {description}
        </p>
      </div>
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

/** 表格级空态（不用插画） */
function PortableTableEmpty({ line1, line2 }: { line1: string; line2?: string }) {
  return (
    <div className="text-center py-12 space-y-1">
      <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>
        {line1}
      </p>
      {line2 && (
        <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>
          {line2}
        </p>
      )}
    </div>
  );
}

/** 浮层 / 弹窗 / Dropdown 空态 */
function PortableOverlayEmpty({ text, link }: { text: string; link?: { label: string } }) {
  return (
    <div className="text-center py-6 space-y-2">
      <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>
        {text}
      </p>
      {link && (
        <button
          className="text-xs hover:underline"
          style={{ color: "var(--cp-text-brand)" }}
        >
          {link.label}
        </button>
      )}
    </div>
  );
}

/* ====================================================================
   展示容器
   ==================================================================== */
function DemoBlock({
  title,
  scene,
  children,
}: {
  title: string;
  scene: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <h3 className="text-sm font-medium text-[#0A0A0A]">{title}</h3>
        <span className="text-xs text-[#94A3B8]">{scene}</span>
      </header>
      <div className="p-5">{children}</div>
    </section>
  );
}

/* ====================================================================
   主页面
   ==================================================================== */
export default function EmptyStateSpecVerify() {
  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8" style={TOKEN_STYLE}>
      <div className="max-w-5xl mx-auto space-y-8">
        {/* 页面标题 */}
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">
            Empty State · Portable Spec Verify
          </h1>
          <p className="text-sm text-[#64748B]">
            基于 <code className="font-mono text-xs bg-[#F1F5F9] px-1 py-0.5 rounded">component-specs/empty-state.md</code> 的
            Portable Fallback 独立验证。不依赖 demo 仓 Empty 组件。
          </p>
        </header>

        {/* ============ 1. 页面级：双行 + 操作引导 ============ */}
        <DemoBlock title="1. 页面级空态（双行 + 操作按钮）" scene="§5 页面/大区域">
          <PortableEmpty
            title="还没有创建任何 Agent"
            description="创建你的第一个 Agent，开始自动化工作流"
            action={
              <button className="h-9 px-6 rounded-[4px] bg-[#0A0A0A] text-white text-sm font-medium">
                + 新建 Agent
              </button>
            }
          />
        </DemoBlock>

        {/* ============ 2. 页面级：单行 ============ */}
        <DemoBlock title="2. 页面级空态（单行 — 禁用粗黑标题）" scene="§6 单行规则">
          <PortableEmpty description="暂无记录" />
        </DemoBlock>

        {/* ============ 3. 卡片内空态 ============ */}
        <DemoBlock title="3. 卡片内空态" scene="§5 卡片容器">
          <div
            className="rounded-xl border overflow-hidden p-0"
            style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}
          >
            <PortableEmpty
              title="暂无技能配置"
              description="从公共技能库或企业技能库添加技能"
              action={
                <button className="h-8 px-4 rounded-[4px] border text-sm" style={{ borderColor: "var(--cp-border)", color: "#020617" }}>
                  添加技能
                </button>
              }
            />
          </div>
        </DemoBlock>

        {/* ============ 4. 表格级空态 ============ */}
        <DemoBlock title="4. 表格空态（双行纯文字，不用插画）" scene="§5 表格">
          <div className="rounded-[4px] border overflow-hidden" style={{ borderColor: "var(--cp-border)" }}>
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-[#FAFBFD] border-b" style={{ borderColor: "#f0f0f0" }}>
                  <th className="h-10 px-4 text-left text-xs font-medium text-[#737373]">名称</th>
                  <th className="h-10 px-4 text-left text-xs font-medium text-[#737373]">状态</th>
                  <th className="h-10 px-4 text-left text-xs font-medium text-[#737373]">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td colSpan={3}>
                    <PortableTableEmpty
                      line1="暂无记录"
                      line2="尝试调整筛选条件，或新建一条记录"
                    />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </DemoBlock>

        {/* ============ 5. 表格级单行 ============ */}
        <DemoBlock title="5. 表格空态（单行）" scene="§5 表格 + §6 单行">
          <div className="rounded-[4px] border overflow-hidden" style={{ borderColor: "var(--cp-border)" }}>
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-[#FAFBFD] border-b" style={{ borderColor: "#f0f0f0" }}>
                  <th className="h-10 px-4 text-left text-xs font-medium text-[#737373]">实例 ID</th>
                  <th className="h-10 px-4 text-left text-xs font-medium text-[#737373]">区域</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td colSpan={2}>
                    <PortableTableEmpty line1="没有符合条件的实例" />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </DemoBlock>

        {/* ============ 6. Dialog 内嵌区块空态 ============ */}
        <DemoBlock title="6. Dialog / 弹窗内嵌区块空态" scene="§5 Dialog">
          <div
            className="rounded-[12px] border p-6 max-w-md mx-auto"
            style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}
          >
            <h4 className="text-base font-semibold mb-4" style={{ color: "var(--cp-text-title)" }}>
              选择技能
            </h4>
            <div className="rounded-[4px] border border-dashed p-0" style={{ borderColor: "var(--cp-border)" }}>
              <PortableTableEmpty
                line1="该角色还没有技能"
                line2="可从公共技能库或企业技能库添加"
              />
            </div>
          </div>
        </DemoBlock>

        {/* ============ 7. Dropdown / Select 空态 ============ */}
        <DemoBlock title="7. Dropdown / Select 空面板" scene="§5 Dropdown/Select">
          <div className="max-w-[240px] mx-auto">
            <div
              className="rounded-[4px] border shadow-lg"
              style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}
            >
              <PortableOverlayEmpty text="暂无可选项" />
            </div>
          </div>
        </DemoBlock>

        {/* ============ 8. Combobox 搜索无匹配 ============ */}
        <DemoBlock title="8. Combobox 搜索无匹配" scene="§5 Combobox">
          <div className="max-w-[300px] mx-auto">
            <div
              className="rounded-[4px] border shadow-lg p-2"
              style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}
            >
              {/* 模拟搜索框 */}
              <div className="flex items-center gap-2 px-3 py-2 border-b" style={{ borderColor: "#f0f0f0" }}>
                <svg className="w-4 h-4 text-[#94A3B8]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <circle cx="11" cy="11" r="8" strokeWidth={2} />
                  <path d="m21 21-4.3-4.3" strokeWidth={2} strokeLinecap="round" />
                </svg>
                <span className="text-sm text-[#94A3B8]">搜索用户...</span>
              </div>
              <PortableOverlayEmpty text="没有匹配的结果" link={{ label: '+ 邀请「张三」加入' }} />
            </div>
          </div>
        </DemoBlock>

        {/* ============ 9. Popover 空态 ============ */}
        <DemoBlock title="9. Popover / 通知面板空态" scene="§5 Popover">
          <div className="max-w-[280px] mx-auto">
            <div
              className="rounded-[8px] border shadow-lg"
              style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}
            >
              <div className="px-4 py-3 border-b" style={{ borderColor: "#f0f0f0" }}>
                <span className="text-sm font-medium" style={{ color: "var(--cp-text-title)" }}>
                  通知
                </span>
              </div>
              <PortableOverlayEmpty text="暂无未读通知" />
            </div>
          </div>
        </DemoBlock>

        {/* ============ 10. Drawer 嵌套子模块降级 ============ */}
        <DemoBlock title="10. Drawer 嵌套子模块空态（降级纯文字）" scene="§5 Drawer 嵌套">
          <div
            className="rounded-[4px] border max-w-md mx-auto"
            style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}
          >
            {/* 模拟 Drawer 内的 Tab 子区块 */}
            <div className="px-4 py-3 border-b" style={{ borderColor: "#f0f0f0" }}>
              <span className="text-xs font-medium text-[#737373]">关联告警（0）</span>
            </div>
            <div className="text-center py-12 space-y-1">
              <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>暂无关联告警</p>
              <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>该资产当前未触发任何告警规则</p>
            </div>
          </div>
        </DemoBlock>

        {/* ============ 11. Tenant 端空态（线框按钮） ============ */}
        <DemoBlock title="11. Tenant 端空态（线框胶囊按钮，禁用实心）" scene="§3.1 Tenant 覆写">
          <div className="rounded-xl border overflow-hidden" style={{ borderColor: "var(--cp-border)", background: "var(--cp-bg-surface)" }}>
            <div className="flex flex-col items-center justify-center gap-4 py-16 text-center">
              <img
                src="/assets/admin-sidebar/empty-aiagent.png"
                alt=""
                className="h-20 w-[100px] object-contain"
              />
              <div className="flex max-w-sm flex-col items-center gap-1">
                <p className="text-sm font-medium" style={{ color: "var(--cp-text-title)" }}>
                  暂无可用 Agent
                </p>
                <p className="text-xs" style={{ color: "var(--cp-text-weak)" }}>
                  联系管理员创建 Agent 或申请访问权限
                </p>
              </div>
              <div className="flex gap-3 mt-2">
                {/* Tenant 空态按钮：全部线框，无主次差异 */}
                <button className="h-9 px-5 rounded-full border text-sm font-medium" style={{ borderColor: "var(--cp-border)", color: "#020617" }}>
                  联系管理员
                </button>
                <button className="h-9 px-5 rounded-full border text-sm font-medium" style={{ borderColor: "var(--cp-border)", color: "#020617" }}>
                  查看帮助
                </button>
              </div>
            </div>
          </div>
        </DemoBlock>

        {/* 底部说明 */}
        <footer className="text-center py-6">
          <p className="text-xs text-[#94A3B8]">
            本页面仅使用 portable fallback 实现，不依赖 demo 仓 Empty 组件 ·{" "}
            <code className="font-mono bg-[#F1F5F9] px-1 rounded">/preview/empty-state-spec-verify</code>
          </p>
        </footer>
      </div>
    </div>
  );
}
