/**
 * TeamAssets —— 员工端「团队门户」
 *
 * 独立路由 /team-assets（兼容旧 /agent-hub、/agent-community），顶部一级导航入口。
 *
 * 团队日常干活的统一入口。Tab 结构：
 *  1. Agent 广场——对标内部智能体控制台：职能分类(研发/产品/运营/架构) + owner/浏览/评论/调用/复用/评分，
 *     点开卡片可在页面内直接对话预览(mock)，并提供 A2A/API/MCP 接入信息供被别的 agent 调用；
 *  2+. 嵌入工具——把内部已有的自研系统以 iframe 方式嵌进来（配置驱动，见 EMBED_TABS）。
 *     能嵌就嵌，被对方 CSP/X-Frame-Options 拦截或加载超时则降级为「在新窗口打开」卡片。
 *
 * 新增一个内部工具入口：只需往 EMBED_TABS 里加一条配置（label/icon/url/desc），
 * 并确保域名在 ALLOWED_EMBED_HOSTS 白名单内（安全：仅允许可信内网域名嵌入）。
 */
import { useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import {
  Search,
  Store,
  Star,
  MessageSquare,
  Eye,
  Repeat,
  Bookmark,
  ExternalLink,
  Info,
  Copy,
  Send,
  ChevronRight,
  Plus,
  Trash2,
  Globe,
  AlertTriangle,
  RefreshCw,
  Award,
  Compass,
  Gauge,
  type LucideIcon,
} from 'lucide-react';
import TenantLayout from '@/components/TenantLayout';
import { Button } from '@/components/ui/button';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { MetaText } from '@/components/ui/Typography';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Empty, EmptyHeader, EmptyDescription } from '@/components/ui/empty';
import {
  teamAssetsStore,
  useAssetsState,
  AGENT_CATEGORY_META,
  ME_OWNER,
  type AgentCategory,
  type MarketAgent,
  type CustomEmbed,
} from './teamAssetsStore';

// ═══════════════════════════════════════════════════════════
//  嵌入工具配置（配置驱动 —— 新增内部工具只需加一条）
// ═══════════════════════════════════════════════════════════

/**
 * 允许被iframe 嵌入的可信域名白名单（安全：防止任意 URL 注入 / 点击劫持）。
 * 仅公司内网可信域名。新增嵌入项时，其host 必须命中此列表（支持子域后缀匹配）。
 */
const ALLOWED_EMBED_HOSTS = [
  'orcaai.woa.com',
  'aiinforsearch.cvm.aiops.woa.com',
  'effective-token.pages.woa.com',
  '.woa.com', // 兜底：允许 woa.com 内网子域（如需收紧可移除）
];

function isAllowedEmbedUrl(raw: string): boolean {
  try {
    const u = new URL(raw);
    if (u.protocol !== 'https:') return false;
    const host = u.hostname.toLowerCase();
    return ALLOWED_EMBED_HOSTS.some((h) =>
      h.startsWith('.') ? host === h.slice(1) || host.endsWith(h) : host === h,
    );
  } catch {
    return false;
  }
}

interface EmbedTabConfig {
  id: string;
  label: string;
  icon: LucideIcon;
  url: string;
  desc: string; // 一句话说明，用于降级卡片
  auth?: boolean; // 是否需要 SSO 登录态（用于给用户提示）
  fallbackToLink?: boolean; // 已知无法 iframe 嵌入的站点，直接显示跳转卡片（如工蜂 Pages 类 SSO cookie SameSite=Lax 的站）
}

/** 首批嵌入的内部工具。新增只在此追加一条即可。 */
const EMBED_TABS: EmbedTabConfig[] = [
  {
    id: 'orca-achievement',
    label: '成就墙',
    icon: Award,
    url: 'https://orcaai.woa.com/achievement',
    desc: 'OrcaAI 团队成就墙，沉淀与展示团队里程碑与荣誉。',
    auth: true,
  },
  {
    id: 'ai-info-search',
    label: '信息检索',
    icon: Compass,
    url: 'https://aiinforsearch.cvm.aiops.woa.com',
    desc: 'CVM AIOps 智能信息检索，一站式检索内部资料与知识。',
    auth: true,
  },
  {
    id: 'effective-token',
    label: 'Token 效能',
    icon: Gauge,
    url: 'https://effective-token.pages.woa.com',
    desc: 'Token 效能分析，透视团队 Token 使用效率与优化空间。',
    auth: true,
    // 工蜂 Pages 服务，SSO cookie SameSite=Lax 导致 iframe 内不携带 → 302 循环 → 无法嵌入
    fallbackToLink: true,
  },
];

// ─── 主 Tab ──────────────────────────────────────────────
type AssetTab = 'market' | string; // 'market' 或某个 embed tab id

// ─── 工具 ────────────────────────────────────────────────
function fmtNum(n: number): string {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}
function isMe(owner: string): boolean {
  return owner === ME_OWNER;
}

function CategoryBadge({ category }: { category: AgentCategory }) {
  const m = AGENT_CATEGORY_META[category];
  return (
    <span
      className="inline-flex items-center h-5 px-1.5 rounded-[4px] text-[11px] text-white"
      style={{ background: m.tint }}
    >
      {m.label}
    </span>
  );
}

function Stars({ rating }: { rating: number }) {
  return (
    <span className="inline-flex items-center gap-0.5 text-[#F59E0B]">
      <Star className="w-3.5 h-3.5 fill-[#F59E0B]" />
      <span className="text-xs font-medium tabular-nums">{rating.toFixed(1)}</span>
    </span>
  );
}

function CopyBlock({ label, code }: { label: string; code: string }) {
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      toast.success(`${label} 已复制`);
    } catch {
      toast.error('复制失败，请手动选择文本');
    }
  };
  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <span className="text-xs font-medium text-[var(--text-secondary)]">{label}</span>
        <button
          type="button"
          onClick={copy}
          className="inline-flex items-center gap-1 text-[11px] text-[var(--text-brand)] hover:opacity-80"
        >
          <Copy className="w-3 h-3" />复制
        </button>
      </div>
      <pre className="text-[11px] leading-relaxed text-[var(--text-body)] bg-[var(--cp-surface)] border border-[var(--cp-border)] rounded-[4px] p-2.5 whitespace-pre-wrap break-all m-0 max-h-[160px] overflow-y-auto">
        {code}
      </pre>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════
//  Tab 1：Agent 广场（对标内部智能体控制台 agenthub）
// ═══════════════════════════════════════════════════════════

function AgentMarketCard({ agent, onOpen }: { agent: MarketAgent; onOpen: (a: MarketAgent) => void }) {
  return (
    <button
      type="button"
      onClick={() => onOpen(agent)}
      className="flex flex-col text-left rounded-[8px] border border-[var(--cp-border)] bg-white p-4 hover:border-[var(--cp-brand-blue)] hover:shadow-[0_2px_12px_rgba(20,71,230,0.08)] transition-all"
    >
      {/* 头部 */}
      <div className="flex items-start gap-3">
        <span className="shrink-0 inline-flex items-center justify-center w-11 h-11 rounded-[8px] bg-[var(--color-gray-100)] text-2xl">
          {agent.avatar}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-semibold text-[var(--text-title)] truncate">{agent.name}</span>
            <CategoryBadge category={agent.category} />
          </div>
          <div className="mt-1 text-xs text-[var(--text-muted)] truncate">
            👤 {agent.owner}{isMe(agent.owner) && <span className="text-[var(--text-brand)]">（我）</span>} · {agent.team}
          </div>
        </div>
        <span
          onClick={(e) => { e.stopPropagation(); teamAssetsStore.toggleAgentCollect(agent.id); }}
          className={`shrink-0 transition-colors cursor-pointer ${agent.collected ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] hover:text-[var(--text-body)]'}`}
          title={agent.collected ? '取消收藏' : '收藏'}
        >
          <Bookmark className={`w-4 h-4 ${agent.collected ? 'fill-[var(--cp-brand-blue)]' : ''}`} />
        </span>
      </div>

      {/* 简介 */}
      <p className="text-sm text-[var(--text-body)] leading-relaxed mt-2.5 mb-0 line-clamp-2">{agent.tagline}</p>

      {/* 底部：浏览/评论/复用 + 评分 */}
      <div className="flex items-center gap-3 mt-3.5 pt-3 border-t border-[var(--cp-border)] text-xs text-[var(--text-muted)]">
        <span className="inline-flex items-center gap-1"><Eye className="w-3.5 h-3.5" />{fmtNum(agent.views)}</span>
        <span className="inline-flex items-center gap-1"><MessageSquare className="w-3.5 h-3.5" />{agent.comments}</span>
        <span className="inline-flex items-center gap-1"><Repeat className="w-3.5 h-3.5" />{agent.reuseCount}</span>
        <Stars rating={agent.rating} />
        {agent.url ? (
          <a
            href={agent.url}
            target="_blank"
            rel="noreferrer noopener"
            onClick={(e) => e.stopPropagation()}
            className="ml-auto inline-flex items-center gap-1 text-[var(--text-brand)] font-medium hover:opacity-80"
          >
            访问 <ExternalLink className="w-3.5 h-3.5" />
          </a>
        ) : (
          <span className="ml-auto inline-flex items-center gap-1 text-[var(--text-brand)] font-medium">
            进入 <ChevronRight className="w-3.5 h-3.5" />
          </span>
        )}
      </div>
    </button>
  );
}

/** 页面内直接对话预览（mock，不接真模型） */
function AgentChatDialog({ agent, onOpenChange }: { agent: MarketAgent | null; onOpenChange: (v: boolean) => void }) {
  const [tab, setTab] = useState<'chat' | 'integrate'>('chat');
  const [msgs, setMsgs] = useState<{ role: 'user' | 'assistant'; text: string }[]>([]);
  const [input, setInput] = useState('');

  // agent 变化时重置
  const agentId = agent?.id;
  useMemo(() => { setMsgs([]); setInput(''); setTab('chat'); }, [agentId]);

  if (!agent) return null;

  const ask = (text: string) => {
    const q = text.trim();
    if (!q) return;
    setMsgs((prev) => [
      ...prev,
      { role: 'user', text: q },
      {
        role: 'assistant',
        text: `（演示回复）「${agent.name}」已收到：${q}\n\n本预览为 mock，不消耗真实模型额度。真实调用请用右侧「接入」页里的 A2A / API / MCP 方式，由你自己的 agent 或系统发起。`,
      },
    ]);
    setInput('');
  };

  return (
    <Dialog open={!!agent} onOpenChange={onOpenChange}>
      <DialogContent size="lg" className="p-0 overflow-hidden">
        {/* 头部 */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-[var(--cp-border)]">
          <span className="shrink-0 inline-flex items-center justify-center w-10 h-10 rounded-[8px] bg-[var(--color-gray-100)] text-xl">
            {agent.avatar}
          </span>
          <div className="min-w-0 flex-1">
            <DialogHeader className="space-y-0">
              <DialogTitle className="flex items-center gap-2">
                {agent.name}
                <CategoryBadge category={agent.category} />
              </DialogTitle>
              <DialogDescription className="text-xs text-[var(--text-muted)] mt-0.5">
                👤 {agent.owner} · {agent.team} · 运行模型 {agent.model}
              </DialogDescription>
            </DialogHeader>
          </div>
          {agent.url && (
            <a
              href={agent.url}
              target="_blank"
              rel="noreferrer noopener"
              className="shrink-0 inline-flex items-center gap-1.5 h-8 px-3 rounded-[6px] bg-[var(--cp-brand-blue)] text-white text-sm font-medium hover:opacity-90"
              title={`访问 ${agent.name} 真实地址`}
            >
              <ExternalLink className="w-3.5 h-3.5" />
              访问 Agent
            </a>
          )}
        </div>

        {/* 子 Tab */}
        <div className="px-5 pt-3">
          <SegmentGroup>
            <SegmentOption active={tab === 'chat'} onClick={() => setTab('chat')}>页面内试用</SegmentOption>
            <SegmentOption active={tab === 'integrate'} onClick={() => setTab('integrate')}>接入（被调用）</SegmentOption>
          </SegmentGroup>
        </div>

        {tab === 'chat' ? (
          <div className="flex h-[440px]">
            {/* 左：常用问题 */}
            <div className="w-[200px] shrink-0 border-r border-[var(--cp-border)] p-3 overflow-y-auto">
              <div className="text-xs font-medium text-[var(--text-secondary)] mb-2">常用问题</div>
              <div className="space-y-1.5">
                {agent.presetQuestions.map((q) => (
                  <button
                    key={q}
                    type="button"
                    onClick={() => ask(q)}
                    className="block w-full text-left text-xs leading-relaxed text-[var(--text-body)] rounded-[4px] border border-[var(--cp-border)] px-2.5 py-2 hover:border-[var(--cp-brand-blue)] hover:text-[var(--text-brand)] transition-colors"
                  >
                    {q}
                  </button>
                ))}
              </div>
            </div>
            {/* 右：对话区 */}
            <div className="flex-1 min-w-0 flex flex-col">
              <div className="flex-1 overflow-y-auto p-4 space-y-3">
                {msgs.length === 0 ? (
                  <div className="h-full flex flex-col items-center justify-center text-center text-[var(--text-muted)]">
                    <span className="text-3xl mb-2">{agent.avatar}</span>
                    <p className="text-sm m-0">{agent.tagline}</p>
                    {agent.url ? (
                      <>
                        <p className="text-xs mt-1 mb-3">该 Agent 已在 knot / imate 上线，点击下方按钮在真实页面使用</p>
                        <a
                          href={agent.url}
                          target="_blank"
                          rel="noreferrer noopener"
                          className="inline-flex items-center gap-1.5 h-8 px-4 rounded-[6px] bg-[var(--cp-brand-blue)] text-white text-sm font-medium hover:opacity-90"
                        >
                          <ExternalLink className="w-3.5 h-3.5" />在 Agent 打开
                        </a>
                      </>
                    ) : (
                      <p className="text-xs mt-1 mb-0">从左侧常用问题开始，或直接输入</p>
                    )}
                  </div>
                ) : (
                  msgs.map((m, i) => (
                    <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                      <div
                        className={`max-w-[80%] rounded-[8px] px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap ${
                          m.role === 'user'
                            ? 'bg-[var(--cp-brand-blue)] text-white'
                            : 'bg-[var(--color-gray-100)] text-[var(--text-body)]'
                        }`}
                      >
                        {m.text}
                      </div>
                    </div>
                  ))
                )}
              </div>
              {/* 输入 */}
              <div className="border-t border-[var(--cp-border)] p-3">
                <div className="flex items-end gap-2">
                  <textarea
                    rows={1}
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); ask(input); }
                    }}
                    placeholder={`向「${agent.name}」提问…（演示，不消耗额度）`}
                    className="flex-1 resize-none h-9 max-h-24 px-3 py-1.5 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
                  />
                  <Button variant="tenant-primary" size="sm" onClick={() => ask(input)} disabled={!input.trim()}>
                    <Send className="w-4 h-4" />
                  </Button>
                </div>
                <div className="mt-1.5 text-[11px] text-[var(--text-weak)]">
                  底部模型：{agent.model}（演示态，仅前端 mock 回复）
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="h-[440px] overflow-y-auto p-5 space-y-4">
            <p className="text-sm text-[var(--text-body)] m-0">
              把「{agent.name}」集成到你自己的 agent 或系统里，任选一种接入方式：
            </p>
            {agent.url && (
              <div className="flex items-center justify-between rounded-[6px] border border-[#93C5FD] bg-[#F8FBFF] px-3.5 py-2.5">
                <div className="min-w-0">
                  <div className="text-sm font-medium text-[var(--text-title)]">Agent 访问地址</div>
                  <div className="text-[11px] text-[var(--text-muted)] truncate">{agent.url}</div>
                </div>
                <a
                  href={agent.url}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="shrink-0 inline-flex items-center gap-1.5 h-8 px-3 rounded-[6px] bg-[var(--cp-brand-blue)] text-white text-sm font-medium hover:opacity-90"
                >
                  <ExternalLink className="w-3.5 h-3.5" />在 Agent 打开
                </a>
              </div>
            )}
            <CopyBlock label="A2A endpoint" code={agent.endpoint.a2a} />
            <CopyBlock label="API 调用（curl）" code={agent.endpoint.apiCurl} />
            <CopyBlock label="MCP 配置片段" code={agent.endpoint.mcp} />
            <div className="flex items-center gap-2 rounded-[4px] bg-[#F8FBFF] border border-[#93C5FD] px-3 py-2 text-xs text-[var(--text-secondary)]">
              <Info className="w-3.5 h-3.5 shrink-0 text-[#1447E6]" />
              演示态：以上 endpoint 为示意，真实接入需在 ClawPro 申请调用凭证 <code className="mx-0.5">$CLAW_TOKEN</code>。
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function JoinDialog({
  open,
  onOpenChange,
  onAdd,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onAdd: (input: { name: string; url: string; category: AgentCategory; tagline: string; owner: string; team: string }) => void;
}) {
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [category, setCategory] = useState<AgentCategory>('dev');
  const [tagline, setTagline] = useState('');
  const [owner, setOwner] = useState('');
  const [team, setTeam] = useState('');

  useMemo(() => {
    if (open) { setName(''); setUrl(''); setCategory('dev'); setTagline(''); setOwner(''); setTeam(''); }
  }, [open]);

  const canSubmit = name.trim().length > 0 && tagline.trim().length > 0;

  const submit = () => {
    if (!canSubmit) return;
    onAdd({ name: name.trim(), url: url.trim(), category, tagline: tagline.trim(), owner: owner.trim(), team: team.trim() });
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>上架 Agent 到广场</DialogTitle>
          <DialogDescription className="text-sm text-[var(--text-muted)]">
            填写 Agent 基本信息，提交后即上架到广场。访问地址填 knot / imate 等机器人的对话链接即可。
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* 名称 */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
              Agent 名称 <span className="text-[#DC2626]">*</span>
            </label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={20}
              placeholder="如：代码评审助手"
              className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
            />
          </div>

          {/* 访问地址 */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">访问地址</label>
            <input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://knot.woa.com/bot/xxx 或 imate 机器人链接"
              className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
            />
            <p className="mt-1 text-[11px] text-[var(--text-muted)] m-0">
              选填，填 knot / imate 等机器人的对话链接，上架后可直接点击访问。
            </p>
          </div>

          {/* 职能分类 */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">职能分类</label>
            <SegmentGroup>
              {(['dev', 'product', 'ops', 'arch'] as AgentCategory[]).map((c) => (
                <SegmentOption key={c} active={category === c} onClick={() => setCategory(c)}>
                  {AGENT_CATEGORY_META[c].label}
                </SegmentOption>
              ))}
            </SegmentGroup>
          </div>

          {/* 一句话简介 */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
              一句话简介 <span className="text-[#DC2626]">*</span>
            </label>
            <input
              value={tagline}
              onChange={(e) => setTagline(e.target.value)}
              maxLength={50}
              placeholder="如：增量 MR 自动评审，产出可执行改进建议"
              className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
            />
          </div>

          {/* 负责人 + 团队 */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">负责人</label>
              <input
                value={owner}
                onChange={(e) => setOwner(e.target.value)}
                placeholder="选填，默认为自己"
                className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">归属团队</label>
              <input
                value={team}
                onChange={(e) => setTeam(e.target.value)}
                placeholder="选填"
                className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-1">
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant="tenant-primary" onClick={submit} disabled={!canSubmit}>
            <Plus className="w-4 h-4" />上架
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function AgentMarketTab({ agents }: { agents: MarketAgent[] }) {
  const [query, setQuery] = useState('');
  const [joinOpen, setJoinOpen] = useState(false);
  const [openAgent, setOpenAgent] = useState<MarketAgent | null>(null);

  const q = query.trim().toLowerCase();
  const list = useMemo(
    () =>
      agents.filter((a) => {
        if (!q) return true;
        return (
          a.name.toLowerCase().includes(q) ||
          a.tagline.toLowerCase().includes(q) ||
          a.team.toLowerCase().includes(q) ||
          a.owner.toLowerCase().includes(q)
        );
      }),
    [agents, q],
  );

  return (
    <div>
      {/* 搜索 + 上架按钮 */}
      <div className="flex items-center gap-3 mb-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
          <input
            className="w-full h-9 pl-9 pr-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索 agent 名称、用途、owner 或团队…"
          />
        </div>
        <Button variant="tenant-primary" size="sm" onClick={() => setJoinOpen(true)}>
          <Plus className="w-4 h-4" />上架 Agent
        </Button>
      </div>

      {/* 列表 */}
      {list.length === 0 ? (
        <Empty className="py-16">
          <EmptyHeader>
            <EmptyDescription>没有匹配的 agent</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          {list.map((a) => (
            <AgentMarketCard key={a.id} agent={a} onOpen={setOpenAgent} />
          ))}
        </div>
      )}

      <JoinDialog
        open={joinOpen}
        onOpenChange={setJoinOpen}
        onAdd={(input) => {
          teamAssetsStore.addCustomAgent(input);
          toast.success(`「${input.name}」已上架到广场`);
        }}
      />
      <AgentChatDialog agent={openAgent} onOpenChange={(v) => !v && setOpenAgent(null)} />
    </div>
  );
}

// ═══════════════════════════════════════════════════════════
//  添加嵌入弹窗
// ═══════════════════════════════════════════════════════════
function AddEmbedDialog({
  open,
  onOpenChange,
  onAdd,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onAdd: (input: { label: string; url: string; desc: string }) => void;
}) {
  const [label, setLabel] = useState('');
  const [url, setUrl] = useState('');
  const [desc, setDesc] = useState('');

  // 打开时重置
  useMemo(() => {
    if (open) { setLabel(''); setUrl(''); setDesc(''); }
  }, [open]);

  const trimmedUrl = url.trim();
  const urlValid = trimmedUrl.length > 0 && isAllowedEmbedUrl(trimmedUrl);
  const urlTouched = trimmedUrl.length > 0;
  const canSubmit = label.trim().length > 0 && urlValid;

  const submit = () => {
    if (!canSubmit) return;
    onAdd({ label: label.trim(), url: trimmedUrl, desc: desc.trim() });
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>添加嵌入工具</DialogTitle>
          <DialogDescription className="text-sm text-[var(--text-muted)]">
            把内部已有的自研系统以页面嵌入方式接进团队门户，填写必要信息即可完成。
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* 名称 */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
              Tab 名称 <span className="text-[#DC2626]">*</span>
            </label>
            <input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              maxLength={12}
              placeholder="如：成就墙、效能看板…"
              className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
            />
          </div>

          {/* URL */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
              嵌入地址 <span className="text-[#DC2626]">*</span>
            </label>
            <input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') submit(); }}
              placeholder="https://xxx.woa.com/…"
              className={`w-full h-9 px-3 rounded-[4px] border bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none ${
                urlTouched && !urlValid
                  ? 'border-[#DC2626] focus:border-[#DC2626]'
                  : 'border-[var(--cp-border)] focus:border-[var(--cp-brand-blue)]'
              }`}
            />
            {urlTouched && !urlValid ? (
              <p className="mt-1 text-[11px] text-[#DC2626] m-0">
                仅支持 https 的公司内网可信域名（*.woa.com），当前地址不被允许嵌入。
              </p>
            ) : (
              <p className="mt-1 text-[11px] text-[var(--text-muted)] m-0">
                仅允许 https 的内网可信域名（*.woa.com），出于安全会做白名单校验。
              </p>
            )}
          </div>

          {/* 说明 */}
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">一句话说明</label>
            <input
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') submit(); }}
              maxLength={40}
              placeholder="选填，用于工具条与加载失败时的说明"
              className="w-full h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm text-[var(--text-body)] placeholder:text-[var(--text-weak)] focus:outline-none focus:border-[var(--cp-brand-blue)]"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-1">
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant="tenant-primary" onClick={submit} disabled={!canSubmit}>
            <Plus className="w-4 h-4" />完成嵌入
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ═══════════════════════════════════════════════════════════
//  嵌入 Tab：iframe 嵌入内部工具 + 加载失败/白屏降级
// ═══════════════════════════════════════════════════════════
function EmbedTab({ config, onRemove }: { config: EmbedTabConfig; onRemove?: () => void }) {
  const allowed = isAllowedEmbedUrl(config.url);
  const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>('loading');
  const [reloadKey, setReloadKey] = useState(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const openNewTab = () => window.open(config.url, '_blank', 'noopener,noreferrer');

  const startTimeout = () => {
    if (timerRef.current) clearTimeout(timerRef.current);
    // 部分被X-Frame-Options 拦截的站点 iframe 不触发 onload，超时后给逃生入口
    timerRef.current = setTimeout(() => {
      setStatus((s) => (s === 'loading' ? 'error' : s));
    }, 12000);
  };

  const reload = () => {
    setStatus('loading');
    setReloadKey((k) => k + 1);
    startTimeout();
  };

  // 首次挂载 & 每次 reload 启动超时
  useMemo(() => {
    startTimeout();
    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reloadKey]);

  // 域名不在白名单：直接拒绝嵌入（安全兜底）
  if (!allowed) {
    return (
      <div className="rounded-[8px] border border-[#FDE68A] bg-[#FFFBEB] px-5 py-8 text-center">
        <AlertTriangle className="w-6 h-6 mx-auto text-[#B45309]" />
        <p className="text-sm text-[var(--text-body)] mt-2 mb-1">该地址不在可信嵌入白名单内，出于安全未嵌入</p>
        <p className="text-xs text-[var(--text-muted)] m-0 break-all">{config.url}</p>
        <Button variant="tenant-outline" size="sm" className="mt-3" onClick={openNewTab}>
          <ExternalLink className="w-4 h-4" />在新标签打开
        </Button>
      </div>
    );
  }

  // 已知无法 iframe 嵌入的站点（如工蜂 Pages SSO cookie SameSite=Lax）：直接显示跳转卡片
  if (config.fallbackToLink) {
    return (
      <div>
        {/* 工具条 */}
        <div className="flex items-center gap-3 rounded-[8px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-4 py-2.5 mb-3">
          <config.icon className="w-4 h-4 shrink-0 text-[var(--text-secondary)]" />
          <div className="min-w-0">
            <div className="text-sm font-medium text-[var(--text-title)] truncate">{config.label}</div>
            <div className="text-[11px] text-[var(--text-muted)] truncate">{config.desc}</div>
          </div>
          {onRemove && (
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => { if (window.confirm(`确定移除嵌入「${config.label}」吗？`)) onRemove(); }}
              title="移除该嵌入"
              className="ml-auto text-[#DC2626] hover:text-[#DC2626]"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </Button>
          )}
        </div>
        {/* 跳转卡片 */}
        <div className="rounded-[8px] border border-[var(--cp-border)] bg-white px-6 py-10 text-center">
          <config.icon className="w-10 h-10 mx-auto text-[var(--cp-brand-blue)]" />
          <p className="text-base font-medium text-[var(--text-title)] mt-3 mb-1">{config.label}</p>
          <p className="text-sm text-[var(--text-muted)] max-w-[460px] mx-auto leading-relaxed m-0 mb-1">
            {config.desc}
          </p>
          <p className="text-xs text-[var(--text-weak)] m-0 mb-4">
            该系统暂不支持页面内嵌入，点击下方按钮在新窗口打开使用。
          </p>
          <Button variant="tenant-primary" onClick={openNewTab}>
            <ExternalLink className="w-4 h-4" />在新窗口打开
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* 工具条 */}
      <div className="flex items-center gap-3 rounded-t-[8px] border border-b-0 border-[var(--cp-border)] bg-[var(--cp-surface)] px-4 py-2.5">
        <config.icon className="w-4 h-4 shrink-0 text-[var(--text-secondary)]" />
        <div className="min-w-0">
          <div className="text-sm font-medium text-[var(--text-title)] truncate">{config.label}</div>
          <div className="text-[11px] text-[var(--text-muted)] truncate">{config.desc}</div>
        </div>
        {config.auth && (
          <span className="hidden sm:inline-flex items-center gap-1 h-5 px-1.5 rounded-[4px] bg-[var(--color-gray-100)] text-[11px] text-[var(--text-secondary)] shrink-0">
            <Info className="w-3 h-3" />需 SSO 登录
          </span>
        )}
        <div className="ml-auto flex items-center gap-1.5 shrink-0">
          <Button variant="tenant-outline" size="sm" onClick={reload} title="重新加载">
            <RefreshCw className="w-3.5 h-3.5" />
          </Button>
          <Button variant="tenant-outline" size="sm" onClick={openNewTab}>
            <ExternalLink className="w-3.5 h-3.5" />新窗口打开
          </Button>
          {onRemove && (
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => {
                if (window.confirm(`确定移除嵌入「${config.label}」吗？`)) onRemove();
              }}
              title="移除该嵌入"
              className="text-[#DC2626] hover:text-[#DC2626]"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </Button>
          )}
        </div>
      </div>

      {/* iframe 容器 */}
      <div className="relative rounded-b-[8px] overflow-hidden border border-[var(--cp-border)] bg-white" style={{ height: 'calc(100vh - 320px)', minHeight: 520 }}>
        <iframe
          key={reloadKey}
          src={config.url}
          title={config.label}
          className="w-full h-full block"
          sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-downloads"
          referrerPolicy="no-referrer-when-downgrade"
          onLoad={() => {
            setStatus('loaded');
            if (timerRef.current) clearTimeout(timerRef.current);
          }}
        />

        {/* 加载中遮罩 */}
        {status === 'loading' && (
          <div className="absolute inset-0 flex flex-col items-center justify-center bg-white/80 pointer-events-none">
            <RefreshCw className="w-6 h-6 text-[var(--cp-brand-blue)] animate-spin" />
            <p className="text-sm text-[var(--text-muted)] mt-2 m-0">正在加载「{config.label}」…</p>
          </div>
        )}

        {/* 加载失败/白屏降级 */}
        {status === 'error' && (
          <div className="absolute inset-0 flex flex-col items-center justify-center bg-white px-6 text-center">
            <AlertTriangle className="w-7 h-7 text-[#B45309]" />
            <p className="text-sm font-medium text-[var(--text-title)] mt-2 mb-1">页面可能未允许被嵌入或需要登录</p>
            <p className="text-xs text-[var(--text-muted)] max-w-[420px] leading-relaxed m-0">
              {config.auth ? '若显示空白，通常是未登录公司 SSO 或该系统禁止被iframe 嵌套。' : '该系统可能禁止被 iframe 嵌套。'}
              可在新窗口打开使用。
            </p>
            <div className="flex items-center gap-2 mt-3">
              <Button variant="tenant-outline" size="sm" onClick={reload}>
                <RefreshCw className="w-4 h-4" />重试
              </Button>
              <Button variant="tenant-primary" size="sm" onClick={openNewTab}>
                <ExternalLink className="w-4 h-4" />在新窗口打开
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════
//  主组件
// ═══════════════════════════════════════════════════════════
export default function TeamAssets() {
  const state = useAssetsState();
  const [tab, setTab] = useState<AssetTab>('market');
  const [addOpen, setAddOpen] = useState(false);

  // 用户自定义嵌入项 → 统一成 EmbedTabConfig（用 Globe 图标标识自定义）
  const customConfigs: (EmbedTabConfig & { custom: true })[] = state.customEmbeds.map((e: CustomEmbed) => ({
    id: e.id,
    label: e.label,
    icon: Globe,
    url: e.url,
    desc: e.desc || '自定义嵌入工具',
    auth: true,
    custom: true,
  }));

  const activeBuiltin = EMBED_TABS.find((e) => e.id === tab);
  const activeCustom = customConfigs.find((e) => e.id === tab);

  const handleAdd = (input: { label: string; url: string; desc: string }) => {
    const id = teamAssetsStore.addCustomEmbed(input);
    setTab(id); // 添加后自动切到新 Tab
    toast.success(`已嵌入「${input.label}」`);
  };

  const handleRemove = (id: string, label: string) => {
    teamAssetsStore.removeCustomEmbed(id);
    if (tab === id) setTab('market'); // 删除当前 Tab 时回到 Agent 广场
    toast.success(`已移除「${label}」`);
  };

  return (
    <TenantLayout>
      <div className="min-w-[1200px]">
        <div className="max-w-[1920px] mx-auto page-enter">
          <div className="relative min-h-[calc(100vh-64px)] pl-[120px] pr-[120px] pt-5 pb-[75px]">
            {/* 标题 */}
            <div className="mb-4">
              <h1 className="text-xl font-semibold text-[var(--text-title)] mb-1 mt-0">团队门户</h1>
              <MetaText tone="secondary">好用的 Agent 与团队自研工具，统一收拢到这个入口——即开即用。</MetaText>
              {tab === 'market' && (
                <MetaText tone="weak" className="block mt-1">
                  好用的 Agent 都在这里，可页面内试用，也可被你的系统集成调用。
                </MetaText>
              )}
            </div>

            {/* 主 Tab：Agent 广场 + 配置驱动的嵌入工具 + 添加按钮 */}
            <div className="mb-4 flex items-center gap-2">
              <SegmentGroup>
                <SegmentOption className="gap-1.5" active={tab === 'market'} onClick={() => setTab('market')}>
                  <Store className="w-4 h-4" />
                  Agent 广场
                </SegmentOption>
                {EMBED_TABS.map((e) => (
                  <SegmentOption key={e.id} className="gap-1.5" active={tab === e.id} onClick={() => setTab(e.id)}>
                    <e.icon className="w-4 h-4" />
                    {e.label}
                  </SegmentOption>
                ))}
                {customConfigs.map((e) => (
                  <SegmentOption key={e.id} className="gap-1.5" active={tab === e.id} onClick={() => setTab(e.id)}>
                    <e.icon className="w-4 h-4" />
                    {e.label}
                  </SegmentOption>
                ))}
              </SegmentGroup>

              {/* 添加嵌入 */}
              <button
                type="button"
                onClick={() => setAddOpen(true)}
                title="添加嵌入工具"
                className="shrink-0 inline-flex items-center justify-center w-8 h-8 rounded-[6px] border border-dashed border-[var(--cp-border)] text-[var(--text-secondary)] hover:border-[var(--cp-brand-blue)] hover:text-[var(--text-brand)] transition-colors"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>

            {/* 内容 */}
            {tab === 'market' && <AgentMarketTab agents={state.agents} />}
            {activeBuiltin && <EmbedTab key={activeBuiltin.id} config={activeBuiltin} />}
            {activeCustom && (
              <EmbedTab
                key={activeCustom.id}
                config={activeCustom}
                onRemove={() => handleRemove(activeCustom.id, activeCustom.label)}
              />
            )}
          </div>
        </div>
      </div>

      <AddEmbedDialog open={addOpen} onOpenChange={setAddOpen} onAdd={handleAdd} />
    </TenantLayout>
  );
}
