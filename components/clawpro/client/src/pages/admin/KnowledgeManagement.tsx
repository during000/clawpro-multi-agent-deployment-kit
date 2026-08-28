/**
 * KnowledgeManagement - 管控端知识管理页（连接器）
 * Design: 「流动蓝图」Fluid Blueprint - Admin Side
 *
 * 结构（对齐「通道配置」ChannelConfig 的双视图模式）：
 *   - AdminPageHeader 标题「知识管理」
 *   - LineTabs 双视图：内置连接器 / 自定义连接器
 *   - 内置连接器：乐享知识库、腾讯文档企业版（卡片列表，仅 2 项，避免表格横向滚动）
 *       · 点「连接」→ 页面内切到授权视图（模拟第三方 OAuth 授权）→ 成功后卡片标记「已连接」
 *       · 已连接后卡片底部展开配置区：「应用范围」（ScopeSelect）与「用户可见」（Switch）横向并排
 *   - 自定义连接器：添加（名称 / 接入地址 / 凭证字段）+ 列表管理应用范围与可见性
 *
 * 说明：
 *   - 受「不修改 App.tsx」约束，授权页用「页面内视图切换」实现，不新增 wouter 路由。
 *   - 连接状态与配置为纯内存 mock（刷新重置），满足演示需要。
 *   - 「乐享 / 腾讯文档」采用官方远程品牌 logo（identity.tencent.com），加载失败回退首字色块占位。
 *     后续建议由设计将 logo 沉淀进 canonical-assets（见 references/conflict-log.md C-019）。
 */
import { Fragment, useState } from "react";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { toast } from "sonner";
import {
  Plus,
  Trash2,
  ChevronDown,
  ChevronRight,
  AlertCircle,
  ArrowLeft,
  ShieldCheck,
  CheckCircle2,
  Loader2,
} from "lucide-react";
import { ScopeSelect } from "@/components/ScopeSelect";
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription, EmptyMedia } from "@/components/ui/empty";
import type { UserGroup } from "./MemberManagement/types";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from "./MemberManagement/mock";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import {
  BodyMedium,
  BodyText,
  CardTitle,
  CodeText,
  CompactText,
  HelperText,
  MetaMedium,
  MetaText,
} from "@/components/ui/Typography";

// 合并所有组织（与通道配置 / 模型配置页保持一致）
const ALL_GROUPS: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];

// ─── 内置连接器 ─────────────────────────────────────────────────────────────────
// 品牌 logo 采用官方远程图（§2.8 图标为登记槽位；此处使用官方 URL，非 lucide 顶替），
// 加载失败时回退首字色块占位。后续建议沉淀进 canonical-assets，见 references/conflict-log.md C-019。

type ScopeState = { scope: "all" | "groups"; groupIds: string[] };

interface BuiltinConnector {
  id: string;
  name: string;
  desc: string;
  /** 官方品牌 logo（远程），加载失败时回退到首字色块占位 */
  logo: string;
  /** 占位方块底色（token）；logo 加载失败时的回退用色 */
  placeholderTone: string;
  /** 模拟授权时展示的授权范围说明 */
  authScopes: string[];
}

const BUILTIN_CONNECTORS: BuiltinConnector[] = [
  {
    id: "lexiang",
    name: "乐享知识库",
    desc: "搜索、创建和管理乐享知识库中的文档。支持导入 Markdown、按标签整理内容、追踪团队文档的更新动态。",
    logo: "https://identity.tencent.com/public/images/logo/lexiang.png",
    placeholderTone: "var(--text-brand)",
    authScopes: ["读取知识库目录与文档内容", "读取文档更新状态以保持同步", "获取知识库成员的可见范围"],
  },
  {
    id: "tencent-doc",
    name: "腾讯文档企业版",
    desc: "创建、编辑和协作腾讯文档。用自然语言管理在线表格、文档和幻灯片，轻松完成内容查询、数据整理和团队协同。",
    logo: "https://identity.tencent.com/public/images/logo/doc.png",
    placeholderTone: "var(--text-success)",
    authScopes: ["读取企业文档与表格内容", "读取文档协作与更新记录", "获取文档的组织可见范围"],
  },
];

// ─── 自定义连接器 ───────────────────────────────────────────────────────────────

interface CredentialField {
  id: string;
  key: string;
  label: string;
}

interface CustomConnector {
  id: string;
  name: string;
  endpoint: string;
  credentialFields: CredentialField[];
  visible: boolean;
}

const KM_TABS = [
  {
    id: "builtin",
    label: "内置连接器",
    description:
      "接入乐享知识库、腾讯文档企业版等平台，完成授权后即可将其知识资源提供给 Agent 检索，无需额外开发。",
  },
  {
    id: "custom",
    label: "自定义连接器",
    description:
      "企业可接入自研知识源，配置连接器名称、接入地址与用户凭证字段。开启「用户可见」后连接器才会对用户展示。",
  },
];

const FIELD_PLACEHOLDERS = ["apiKey", "secretKey"];

// ─── 空白表单 ─────────────────────────────────────────────────────────────────────

type FormState = {
  name: string;
  endpoint: string;
  credentialFields: CredentialField[];
};

function emptyForm(): FormState {
  return { name: "", endpoint: "", credentialFields: [] };
}

// ─── 连接器图标 ────────────────────────────────────────────────────────────────
// 优先渲染官方品牌 logo（远程），加载失败时回退到「产品名首字 + token 色方块」中性占位；
// 容器统一 4px 圆角（--radius-lg）合规。见 references/conflict-log.md C-019。
function ConnectorLogo({ name, tone, logo }: { name: string; tone: string; logo?: string }) {
  const [failed, setFailed] = useState(false);

  if (logo && !failed) {
    return (
      <span className="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-[var(--radius-lg)]">
        <img
          src={logo}
          alt={name}
          className="h-8 w-8 object-contain"
          loading="lazy"
          onError={() => setFailed(true)}
        />
      </span>
    );
  }

  return (
    <span
      className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius-lg)] text-sm font-semibold"
      style={{ backgroundColor: `color-mix(in srgb, ${tone} 12%, transparent)`, color: tone }}
      aria-hidden="true"
    >
      {name.slice(0, 1)}
    </span>
  );
}

// ─── 主组件 ──────────────────────────────────────────────────────────────────────

export default function KnowledgeManagement() {
  // Tab 切换
  const [activeTab, setActiveTab] = useState("builtin");
  const currentTab = KM_TABS.find((t) => t.id === activeTab)!;

  // ── 授权视图（页面内切换） ──
  const [authConnector, setAuthConnector] = useState<BuiltinConnector | null>(null);
  const [authorizing, setAuthorizing] = useState(false);

  // ── 内置连接器状态（纯内存 mock） ──
  const [builtinConnected, setBuiltinConnected] = useState<Record<string, boolean>>({});
  const [builtinVisibility, setBuiltinVisibility] = useState<Record<string, boolean>>({});
  const [builtinScopes, setBuiltinScopes] = useState<Record<string, ScopeState>>({});

  // ── 自定义连接器状态（纯内存 mock） ──
  const [customConnectors, setCustomConnectors] = useState<CustomConnector[]>([]);
  const [customScopes, setCustomScopes] = useState<Record<string, ScopeState>>({});
  const [expandedCustomId, setExpandedCustomId] = useState<string | null>(null);

  // ── 弹窗 / 表单 ──
  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm());
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [disconnectId, setDisconnectId] = useState<string | null>(null);

  // ── 连接 / 授权流程 ──
  const startConnect = (c: BuiltinConnector) => {
    setAuthConnector(c);
    setAuthorizing(false);
  };

  const cancelAuth = () => {
    setAuthConnector(null);
    setAuthorizing(false);
  };

  const confirmAuth = () => {
    if (!authConnector) return;
    setAuthorizing(true);
    // 模拟第三方 OAuth 授权跳转 / 回调
    window.setTimeout(() => {
      const c = authConnector;
      setBuiltinConnected((prev) => ({ ...prev, [c.id]: true }));
      setBuiltinVisibility((prev) => ({ ...prev, [c.id]: true }));
      setBuiltinScopes((prev) => ({ ...prev, [c.id]: prev[c.id] ?? { scope: "all", groupIds: [] } }));
      setAuthorizing(false);
      setAuthConnector(null);
      toast.success(`「${c.name}」授权成功，已完成连接`);
    }, 900);
  };

  const disconnect = (id: string) => {
    const c = BUILTIN_CONNECTORS.find((x) => x.id === id);
    setBuiltinConnected((prev) => ({ ...prev, [id]: false }));
    setBuiltinVisibility((prev) => ({ ...prev, [id]: false }));
    setDisconnectId(null);
    toast.success(`「${c?.name}」已断开连接`);
  };

  // ── 自定义连接器操作 ──
  const openAddDialog = () => {
    setForm(emptyForm());
    setDialogOpen(true);
  };

  const handleSave = () => {
    if (!form.name.trim()) { toast.error("请填写连接器名称"); return; }
    if (!form.endpoint.trim()) { toast.error("请填写接入地址"); return; }
    for (const f of form.credentialFields) {
      if (!f.key.trim()) { toast.error("凭证字段 Key 不能为空"); return; }
      if (!f.label.trim()) { toast.error("凭证字段名称不能为空"); return; }
    }
    const newConnector: CustomConnector = {
      id: `custom_${Date.now()}`,
      name: form.name.trim(),
      endpoint: form.endpoint.trim(),
      credentialFields: form.credentialFields,
      visible: false,
    };
    setCustomConnectors((prev) => [...prev, newConnector]);
    toast.success(`「${newConnector.name}」已添加，默认不可见，开启「用户可见」后用户即可选择`);
    setDialogOpen(false);
  };

  const handleDelete = (id: string) => {
    setCustomConnectors((prev) => prev.filter((c) => c.id !== id));
    if (expandedCustomId === id) setExpandedCustomId(null);
    setDeleteConfirmId(null);
    toast.success("连接器已删除");
  };

  const toggleCustomVisible = (id: string, v: boolean) => {
    setCustomConnectors((prev) => prev.map((c) => (c.id === id ? { ...c, visible: v } : c)));
    const c = customConnectors.find((x) => x.id === id);
    toast.success(`「${c?.name}」已${v ? "开启用户可见" : "关闭用户可见"}`);
  };

  // ── 凭证字段操作 ──
  const addCredentialField = () => {
    setForm((f) => ({
      ...f,
      credentialFields: [...f.credentialFields, { id: `field_${Date.now()}`, key: "", label: "" }],
    }));
  };
  const removeCredentialField = (fieldId: string) => {
    setForm((f) => ({ ...f, credentialFields: f.credentialFields.filter((x) => x.id !== fieldId) }));
  };
  const updateCredentialFieldKey = (fieldId: string, key: string) => {
    setForm((f) => ({
      ...f,
      credentialFields: f.credentialFields.map((x) => (x.id === fieldId ? { ...x, key } : x)),
    }));
  };
  const updateCredentialFieldLabel = (fieldId: string, label: string) => {
    setForm((f) => ({
      ...f,
      credentialFields: f.credentialFields.map((x) => (x.id === fieldId ? { ...x, label } : x)),
    }));
  };

  // ── 授权视图（点连接后进入的「新页面」） ──
  if (authConnector) {
    return (
      <div className="page-enter">
        <Button variant="link-dark" onClick={cancelAuth} className="gap-1.5">
          <ArrowLeft className="w-4 h-4" />
          返回连接器列表
        </Button>

        <div className="mt-6 flex justify-center">
          <SurfaceCard className="w-full max-w-xl p-8">
            <div className="flex flex-col items-center text-center">
              <ConnectorLogo name={authConnector.name} tone={authConnector.placeholderTone} logo={authConnector.logo} />
              <CardTitle as="h2" className="mt-4">
                授权连接「{authConnector.name}」
              </CardTitle>
              <CompactText as="p" tone="muted" className="mt-2 leading-relaxed">
                即将跳转至 {authConnector.name} 完成授权。授权后系统将以只读方式访问以下内容，用于向 Agent 提供知识检索能力。
              </CompactText>
            </div>

            <SurfaceInner className="mt-6 px-4 py-4 space-y-2.5">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-[var(--text-brand)]" />
                <MetaMedium tone="secondary">本次授权范围</MetaMedium>
              </div>
              <ul className="space-y-2">
                {authConnector.authScopes.map((s) => (
                  <li key={s} className="flex items-start gap-2">
                    <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5 text-[var(--text-success)]" />
                    <MetaText tone="body">{s}</MetaText>
                  </li>
                ))}
              </ul>
            </SurfaceInner>

            <div className="mt-6 flex items-center justify-center gap-3">
              <Button variant="claw-outline" onClick={cancelAuth} disabled={authorizing}>
                取消
              </Button>
              <Button variant="claw-primary" onClick={confirmAuth} disabled={authorizing}>
                {authorizing ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    授权中...
                  </>
                ) : (
                  "授权并连接"
                )}
              </Button>
            </div>
          </SurfaceCard>
        </div>
      </div>
    );
  }

  // ── 列表视图 ──
  return (
    <div className="page-enter">
      <AdminPageHeader title="知识管理" />

      {/* Tab 切换器（与通道配置同款 LineTabs：黑色下划线）
        * 停服态豁免：切换「内置连接器 / 自定义连接器」属于查看类操作（不产生变更），
        * 与其他 Tab 视图切换同档，需保持 100% 不透明与正常交互。
        * 原生 <button> 未设置 disabled，"停服前已禁用则延续禁用"约束
        * 通过原生 disabled 属性依然生效（此处无）。 */}
      <div className="mb-1" data-billing-exempt>
        <div className="flex items-center gap-2 border-b border-[#dbe6ff]">
          {KM_TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`relative px-4 py-3 transition-colors whitespace-nowrap ${
                activeTab === tab.id ? "border-b-2 border-[#0A0A0A] -mb-px" : ""
              }`}
            >
              <BodyMedium
                tone={activeTab === tab.id ? "primary" : "muted"}
                className={activeTab !== tab.id ? "hover:text-[var(--text-title)] transition-colors" : ""}
              >
                {tab.label}
              </BodyMedium>
            </button>
          ))}
        </div>
      </div>

      {/* Tab 描述 */}
      <div className="flex items-center gap-3 mt-3 mb-6">
        <CompactText as="p" tone="muted" className="leading-relaxed">
          {currentTab.description}
        </CompactText>
      </div>

      {/* ── 内置连接器 Tab（卡片列表：消除横向滚动，介绍完整展示） ── */}
      {activeTab === "builtin" && (
        <div className="space-y-3">
          {BUILTIN_CONNECTORS.map((c) => {
            const connected = builtinConnected[c.id] || false;
            return (
              <SurfaceCard key={c.id} className="p-4">
                {/* 头部：logo + 名称/状态/介绍 + 连接操作 */}
                <div className="flex items-start gap-4">
                  <ConnectorLogo name={c.name} tone={c.placeholderTone} logo={c.logo} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <BodyMedium tone="primary">{c.name}</BodyMedium>
                      {connected ? (
                        <Badge color="green">已连接</Badge>
                      ) : (
                        <Badge variant="secondary">未连接</Badge>
                      )}
                    </div>
                    <CompactText as="p" tone="muted" className="mt-1 leading-relaxed">
                      {c.desc}
                    </CompactText>
                  </div>
                  <div className="shrink-0">
                    {connected ? (
                      <Button variant="claw-outline" size="sm" onClick={() => setDisconnectId(c.id)}>
                        断开连接
                      </Button>
                    ) : (
                      <Button variant="claw-primary" size="sm" onClick={() => startConnect(c)}>
                        连接
                      </Button>
                    )}
                  </div>
                </div>

                {/* 连接后展开配置区：应用范围 + 用户可见（横向并排，不滚动） */}
                {connected && (
                  <div className="mt-4 pt-4 border-t border-[var(--cp-border)] flex flex-wrap items-center gap-x-6 gap-y-3">
                    <div className="flex items-center gap-3">
                      <MetaMedium tone="secondary" className="shrink-0">应用范围</MetaMedium>
                      <ScopeSelect
                        scope={builtinScopes[c.id]?.scope || "all"}
                        selectedGroupIds={builtinScopes[c.id]?.groupIds || []}
                        groups={ALL_GROUPS}
                        maxVisibleBadges={5}
                        onConfirm={(scope, groupIds) => {
                          setBuiltinScopes((prev) => ({ ...prev, [c.id]: { scope, groupIds } }));
                        }}
                      />
                    </div>
                    <div className="flex items-center gap-3">
                      <MetaMedium tone="secondary" className="shrink-0">用户可见</MetaMedium>
                      <Switch
                        checked={builtinVisibility[c.id] || false}
                        onCheckedChange={(v) => {
                          setBuiltinVisibility((prev) => ({ ...prev, [c.id]: v }));
                          toast.success(`${c.name} 已${v ? "开启用户可见" : "关闭用户可见"}`);
                        }}
                      />
                    </div>
                  </div>
                )}
              </SurfaceCard>
            );
          })}
        </div>
      )}

      {/* ── 自定义连接器 Tab ── */}
      {activeTab === "custom" && (
        <div className="space-y-3">
          <div className="flex items-center justify-end">
            <Button size="sm" onClick={openAddDialog}>
              <Plus className="w-4 h-4" />
              添加自定义连接器
            </Button>
          </div>

          {customConnectors.length === 0 ? (
            <SurfaceCard className="overflow-hidden">
              <Empty className="border-0 py-12">
                <EmptyHeader>
                  <EmptyMedia />
                  <EmptyTitle>暂无自定义连接器</EmptyTitle>
                  <EmptyDescription>点击「添加自定义连接器」接入企业自研知识源</EmptyDescription>
                </EmptyHeader>
              </Empty>
            </SurfaceCard>
          ) : (
            <SurfaceCard className="overflow-hidden">
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead style={{ minWidth: 320 }}>连接器名称</TableHead>
                    <TableHead style={{ width: "100%", minWidth: 200 }}>应用范围</TableHead>
                    <TableHead style={{ width: 120, minWidth: 120 }}>用户可见</TableHead>
                    <TableHead style={{ width: 140, minWidth: 140, maxWidth: 140 }}>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {customConnectors.map((c) => (
                    <Fragment key={c.id}>
                      <TableRow
                        className="cursor-pointer"
                        onClick={() => setExpandedCustomId(expandedCustomId === c.id ? null : c.id)}
                      >
                        <TableCell>
                          <div className="flex items-center gap-2 min-w-0">
                            <span
                              className="shrink-0 w-4 h-4 inline-flex items-center justify-center text-[var(--text-muted)]"
                              aria-hidden="true"
                            >
                              {expandedCustomId === c.id ? (
                                <ChevronDown className="w-3.5 h-3.5" />
                              ) : (
                                <ChevronRight className="w-3.5 h-3.5" />
                              )}
                            </span>
                            <BodyMedium tone="primary" className="truncate">{c.name}</BodyMedium>
                          </div>
                        </TableCell>
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <ScopeSelect
                            scope={customScopes[c.id]?.scope || "all"}
                            selectedGroupIds={customScopes[c.id]?.groupIds || []}
                            groups={ALL_GROUPS}
                            maxVisibleBadges={5}
                            onConfirm={(scope, groupIds) => {
                              setCustomScopes((prev) => ({ ...prev, [c.id]: { scope, groupIds } }));
                            }}
                          />
                        </TableCell>
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <Switch checked={c.visible} onCheckedChange={(v) => toggleCustomVisible(c.id, v)} />
                        </TableCell>
                        <TableActionCell onClick={(e) => e.stopPropagation()}>
                          <Button variant="link" size="sm" onClick={() => setDeleteConfirmId(c.id)}>
                            删除
                          </Button>
                        </TableActionCell>
                      </TableRow>

                      {/* 展开详情行：横跨四列 */}
                      {expandedCustomId === c.id && (
                        <TableRow className="hover:bg-transparent">
                          <TableCell colSpan={4} className="bg-[#fafafa]/50">
                            <div className="space-y-3 py-1">
                              <SurfaceInner className="px-4 py-3 space-y-2">
                                <CardTitle as="h4">接入地址</CardTitle>
                                <CodeText as="span" tone="body" className="break-all">
                                  {c.endpoint || "—"}
                                </CodeText>
                              </SurfaceInner>
                              <SurfaceInner className="px-4 py-3 space-y-2">
                                <CardTitle as="h4">用户凭证字段</CardTitle>
                                {c.credentialFields.length === 0 ? (
                                  <MetaText as="p" tone="weak">无凭证字段</MetaText>
                                ) : (
                                  <div className="flex flex-wrap gap-2">
                                    {c.credentialFields.map((f, idx) => (
                                      <Badge key={f.id} variant="outline" className="gap-1">
                                        <MetaText tone="weak">{idx + 1}.</MetaText>
                                        <CodeText as="span" tone="muted">{f.key}</CodeText>
                                        <MetaText tone="weak">/</MetaText>
                                        <MetaText tone="secondary">{f.label}</MetaText>
                                      </Badge>
                                    ))}
                                  </div>
                                )}
                              </SurfaceInner>
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </Fragment>
                  ))}
                </TableBody>
              </Table>
            </SurfaceCard>
          )}
        </div>
      )}

      {/* ── 新增自定义连接器弹窗 ── */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[720px] flex flex-col max-h-[min(640px,90vh)]">
          <DialogHeader className="shrink-0">
            <DialogTitle>添加自定义连接器</DialogTitle>
            <HelperText>接入企业自研知识源，保存后可在列表中管理应用范围与可见性。</HelperText>
          </DialogHeader>

          <DialogBody className="px-6 space-y-4">
            <Alert variant="warning">
              <AlertCircle />
              <AlertDescription>
                自定义连接器需企业按对接规范提供可访问的知识检索接口，保存后开启「用户可见」方可对用户展示。
              </AlertDescription>
            </Alert>

            {/* 基础信息 */}
            <section className="space-y-3">
              <CardTitle as="h3">连接器基础信息</CardTitle>
              <div className="space-y-3">
                <div className="space-y-2">
                  <MetaMedium as="label" tone="secondary">
                    连接器名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    placeholder="展示给用户的连接器名字，如「内部 Wiki」"
                    value={form.name}
                    onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  />
                </div>
                <div className="space-y-2">
                  <MetaMedium as="label" tone="secondary">
                    接入地址<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    placeholder="知识检索接口地址，如 https://kb.example.com/api"
                    value={form.endpoint}
                    onChange={(e) => setForm((f) => ({ ...f, endpoint: e.target.value }))}
                    className="font-mono"
                  />
                  <HelperText>需为企业可访问的 HTTP(S) 知识检索接口地址</HelperText>
                </div>
              </div>
            </section>

            {/* 凭证字段 */}
            <section className="space-y-3">
              <div className="flex items-center justify-between">
                <CardTitle as="h3">用户凭证字段</CardTitle>
                <Button size="sm" variant="outline" onClick={addCredentialField}>
                  <Plus className="w-3.5 h-3.5" />
                  添加字段
                </Button>
              </div>

              <div className="space-y-2">
                {form.credentialFields.length === 0 ? (
                  <div className="rounded-[4px] border border-dashed border-[var(--cp-border)] px-4 py-3 text-center">
                    <HelperText>暂未添加凭证字段</HelperText>
                  </div>
                ) : (
                  <SurfaceInner className="p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <span className="w-5 shrink-0" />
                      <div className="flex gap-2 flex-1">
                        <div className="flex-1">
                          <MetaMedium as="p" tone="secondary">字段 Key</MetaMedium>
                          <HelperText>写入配置文件的字段名</HelperText>
                        </div>
                        <div className="flex-1">
                          <MetaMedium as="p" tone="secondary">字段名称</MetaMedium>
                          <HelperText>用户看到的字段名称</HelperText>
                        </div>
                      </div>
                      <span className="w-7 shrink-0" />
                    </div>
                    {form.credentialFields.map((field, idx) => (
                      <div key={field.id} className="flex items-center gap-2">
                        <MetaText tone="muted" className="w-5 text-right shrink-0">{idx + 1}.</MetaText>
                        <div className="flex gap-2 flex-1">
                          <Input
                            placeholder={`如 ${FIELD_PLACEHOLDERS[idx % FIELD_PLACEHOLDERS.length]}`}
                            value={field.key}
                            onChange={(e) => updateCredentialFieldKey(field.id, e.target.value)}
                            className="flex-1 font-mono"
                          />
                          <Input
                            placeholder={idx % 2 === 0 ? "如 知识库的 API Key" : "如 知识库的 Secret Key"}
                            value={field.label}
                            onChange={(e) => updateCredentialFieldLabel(field.id, e.target.value)}
                            className="flex-1"
                          />
                        </div>
                        <button
                          className="w-7 shrink-0 text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors flex items-center justify-center"
                          onClick={() => removeCredentialField(field.id)}
                          title="删除此字段"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    ))}
                  </SurfaceInner>
                )}
                <HelperText className="leading-5 pt-0.5">
                  凭证字段名称会展示在用户端，用户选择该连接器后会看到对应的输入框
                </HelperText>
              </div>
            </section>
          </DialogBody>

          <DialogFooter className="items-center">
            <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── 删除确认弹窗 ── */}
      <AlertDialog open={!!deleteConfirmId} onOpenChange={(open) => !open && setDeleteConfirmId(null)}>
        <AlertDialogContent className="sm:max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">删除连接器</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <BodyText as="p" tone="primary">
              删除后，该自定义连接器将从用户端列表中移除，已引用该连接器的 Agent 配置不受影响。
              <BodyText as="span" tone="danger">此操作不可撤销。</BodyText>
            </BodyText>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}>
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ── 断开连接确认弹窗 ── */}
      <AlertDialog open={!!disconnectId} onOpenChange={(open) => !open && setDisconnectId(null)}>
        <AlertDialogContent className="sm:max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">断开连接</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <BodyText as="p" tone="primary">
              断开后，该连接器的知识资源将不再向 Agent 提供检索，需要重新授权才能恢复连接。
            </BodyText>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => disconnectId && disconnect(disconnectId)}>
              确认断开
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
