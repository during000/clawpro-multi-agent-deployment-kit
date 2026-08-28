import React, { useRef } from 'react';
import { Check } from 'lucide-react';
import { PanelTitle, MetaText, MiniBodyText } from '@/components/ui/Typography';

interface ComparisonTableProps {
  /** Pro 服务是否已开通（开通后 Pro 卡可在外层另作徽标处理；本视图按设计稿不显示徽标） */
  isProActive?: boolean;
}

/**
 * 版本对比展示：Free 版 vs Pro 版（Figma node 134:879 还原）
 *
 * 布局：
 *   - 左列 Free 版：白底图标容器 + 标题 + 入门方案 outline tag + 3 条特性
 *   - 中间 1px 垂直分隔线
 *   - 右列 Pro 版：黑→深蓝渐变图标容器 + 标题 + 黑底「推荐」+ 企业级方案 outline tag
 *     - 上半：3 条特性（与 Free 对齐）
 *     - 右上 2×2 网格：4 个企业级特性卡片
 *
 * 特性条文案约束：
 *   - Free 版后两条「仅关键词检索」「小于 1w 条记忆数据」为弱化态（30% 灰）
 */
export const ComparisonTable: React.FC<ComparisonTableProps> = (
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _props,
) => {
  return (
    <div className="grid grid-cols-[170px_1px_1fr] gap-x-10 gap-y-0">
      {/* ─────────── 左列：Free 版 ─────────── */}
      <div className="flex flex-col">
        {/* 图标 — 自带渐变方块背景的 36×36 svg */}
        <img
          src="/assets/admin-memory-management/version-compare/icon-free.svg"
          alt="Free"
          className="w-9 h-9 shrink-0"
        />

        {/* 标题 + 标签 */}
        <div className="mt-[16px]">
          <PanelTitle as="div" className="text-[16px] leading-none">Free 版</PanelTitle>
          <div className="mt-[13px] inline-flex h-[18px] items-center rounded-full border border-black/40 px-2">
            <MetaText tone="primary" className="text-[10px] leading-none">入门方案</MetaText>
          </div>
        </div>

        {/* 特性列表 */}
        <ul className="mt-[16px] space-y-2">
          <FeatureRow label="本地文件持久化" />
          <FeatureRow label="仅关键词检索" muted />
          <FeatureRow label="小于 1w 条记忆数据" muted />
        </ul>
      </div>

      {/* ─────────── 垂直分隔线（高度 2/3，居中） ─────────── */}
      <div className="self-center w-px h-2/3 bg-[#E6E9EF]" />

      {/* ─────────── 右列：Pro 版（左：头部+特性 / 右：2x2 卡片，两子列顶端对齐） ─────────── */}
      <div className="grid grid-cols-[190px_minmax(0,1fr)] gap-x-10 items-start">
        {/* 左：图标 + 标题 + 标签 + 3 条特性 */}
        <div>
          <div
            className="w-9 h-9 rounded-[4px] flex items-center justify-center shrink-0"
            style={{ background: 'linear-gradient(133deg, #010618 0%, #05207E 100%)' }}
          >
            <img
              src="/assets/admin-memory-management/version-compare/pro-icon.svg"
              alt=""
              className="w-[18px] h-[18px]"
              aria-hidden="true"
            />
          </div>

          <div className="mt-[16px] flex items-end gap-2">
            <PanelTitle as="span" className="text-[16px] leading-none">Pro 版</PanelTitle>
          </div>

          <div className="mt-[13px] flex items-center gap-1">
            <span className="inline-flex h-[18px] items-center rounded-full bg-[#0A0A0A] px-2">
              <MetaText tone="inherit" className="text-[10px] leading-none text-white">推荐</MetaText>
            </span>
            <span className="inline-flex h-[18px] items-center rounded-full border border-black/40 px-2">
              <MetaText tone="primary" className="text-[10px] leading-none">企业级方案</MetaText>
            </span>
          </div>

          <ul className="mt-[16px] space-y-2">
            <FeatureRow label="腾讯云向量数据库（VDB）" />
            <FeatureRow label="语义 + 关键词双路检索" />
            <FeatureRow label="支持百万级记忆数据" />
          </ul>
        </div>

        {/* 右：企业级特性 2x2 卡片网格（顶端与 Pro 版标题对齐，每张卡片定最小宽确保文本不换行） */}
        <div
          className="grid gap-2 mt-[52px]"
          style={{ gridTemplateColumns: 'repeat(2, minmax(220px, 260px))' }}
        >
          <EnterpriseFeatureCard
            iconSrc="/assets/admin-memory-management/version-compare/feature-tenant.svg"
            label="租户权限隔离，访问更安全"
          />
          <EnterpriseFeatureCard
            iconSrc="/assets/admin-memory-management/version-compare/feature-encrypt.svg"
            label="全链路加密，保障数据安全"
          />
          <EnterpriseFeatureCard
            iconSrc="/assets/admin-memory-management/version-compare/feature-backup.svg"
            label="数据备份，可靠性更高"
          />
          <EnterpriseFeatureCard
            iconSrc="/assets/admin-memory-management/version-compare/feature-token.svg"
            label="短期记忆压缩，Token 节省 50%+"
          />
        </div>
      </div>
    </div>
  );
};

/* ─────────────── 子组件 ─────────────── */

/** 特性单行：勾 + 文案。muted=true 时文字置 30% 灰（Free 版未启用特性） */
const FeatureRow: React.FC<{ label: string; muted?: boolean }> = ({ label, muted = false }) => (
  <li className="flex items-center gap-1.5">
    <Check
      className={`w-3.5 h-3.5 shrink-0 ${muted ? 'text-black/30' : 'text-[var(--text-emphasis)]'}`}
      strokeWidth={1.5}
    />
    <MiniBodyText
      tone={muted ? 'inherit' : 'primary'}
      className={`leading-5 ${muted ? 'text-black/30' : ''}`}
    >
      {label}
    </MiniBodyText>
  </li>
);

/** 企业级特性卡片：图标 + 文案，9px 圆角 + 1px 浅描边
 *  附带 spotlight 鼠标跟随高光效果（参考 reactbits.dev/components/spotlight-card） */
const EnterpriseFeatureCard: React.FC<{ iconSrc: string; label: string }> = ({
  iconSrc,
  label,
}) => {
  const cardRef = useRef<HTMLDivElement>(null);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    const card = cardRef.current;
    if (!card) return;
    const rect = card.getBoundingClientRect();
    card.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`);
    card.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`);
  };

  return (
    <div
      ref={cardRef}
      onMouseMove={handleMouseMove}
      className="group relative overflow-hidden rounded-[9px] border border-[#EAEAEA] px-4 py-3 h-full transition-colors hover:border-[#D8E1FF]"
    >
      {/* spotlight 高光层 */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
        style={{
          background:
            'radial-gradient(circle 200px at var(--mouse-x, 50%) var(--mouse-y, 50%), rgba(213, 229, 255, 0.6), transparent 70%)',
        }}
      />
      {/* 内容 */}
      <div className="relative">
        <img src={iconSrc} alt="" aria-hidden="true" className="w-4 h-4" />
        <MiniBodyText as="div" tone="primary" className="mt-1 leading-5 whitespace-nowrap">{label}</MiniBodyText>
      </div>
    </div>
  );
};
