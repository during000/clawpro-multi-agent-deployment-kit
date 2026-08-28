/**
 * ChannelConfig - 管控端通道配置页
 * 支持内置通道（微信/QQ/企业微信/钉钉/飞书）可见性管理
 * 以及自定义通道的添加、可见性控制（不支持编辑，仅删除）
 */
import { Fragment, useEffect, useMemo, useState } from "react";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
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
import { Plus, Trash2, ChevronDown, ChevronRight, AlertCircle, X } from "lucide-react";
import { ScopeSelect } from "@/components/ScopeSelect";
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription, EmptyMedia } from "@/components/ui/empty";
import type { UserGroup } from "./MemberManagement/types";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from "./MemberManagement/mock";
import {
  type CustomChannel,
  type CredentialField,
  loadCustomChannels,
  saveCustomChannels,
  onCustomChannelsChange,
  loadBuiltinChannelVisibility,
  saveBuiltinChannelVisibility,
  onBuiltinChannelVisibilityChange,
} from "@/lib/customChannelStore";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { LineTabs } from "@/components/ui/line-tabs";
import { SegmentedTabs } from "@/components/ui/segmented-tabs";
import { canonicalAssets } from "@/design-assets/canonical-assets";
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

// 合并所有组织（与模型配置页保持一致）
const ALL_GROUPS: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];

// ─── 图标资源 ────────────────────────────────────────────────────────────────────
// 渠道图标接入 canonical 统一入口（建设计划 §9 阶段 6 / §5.3）：
// 修改 canonical-assets.ts 中 channels.* 的路径即可统一影响本页（一改多处生效）。
const CHANNEL_ICON_SRC: Record<string, string> = {
  wechat: canonicalAssets.channels.wechat,
  qq: canonicalAssets.channels.qq,
  wework: canonicalAssets.channels.wecom,
  "wework-app": canonicalAssets.channels.wecomApp,
  dingtalk: canonicalAssets.channels.dingtalk,
  feishu: canonicalAssets.channels.feishu,
};


// ─── 内置通道列表 ─────────────────────────────────────────────────────────────────

const BUILTIN_CHANNELS = [
  { id: "wechat", name: "微信" },
  { id: "qq", name: "QQ" },
  { id: "wework", name: "企业微信" },
  { id: "wework-app", name: "企业微信应用" },
  { id: "dingtalk", name: "钉钉" },
  { id: "feishu", name: "飞书" },
];

// ─── Tab 定义（与 SkillConfig 同款） ─────────────────────────────────────────────
const CHANNEL_TABS = [
  {
    id: "builtin",
    label: "内置通道",
    description:
      "通过微信、QQ 等机器人接入，可实现与对应渠道的智能机器人对话，满足全场景下的个人沟通与企业服务需求，覆盖不同团队多样化协作场景。",
  },
  {
    id: "custom",
    label: "自定义通道",
    description:
      "企业可配置自研 IM 通道信息，添加后用户可在 Agent 配置页选择对应通道并填写凭证。开启「用户可见」后通道才会对用户展示。目前自定义通道仅支持 WebSocket 长连接方式接入。",
  },
];

// 预设颜色列表
const ICON_COLORS = [
  "#6366F1", "#8B5CF6", "#EC4899", "#F59E0B",
  "#10B981", "#3B82F6", "#EF4444", "#14B8A6",
];

// 凭证字段 placeholder 循环列表
const FIELD_PLACEHOLDERS = ["accessKey", "secretKey"];

let colorIdx = 0;
function nextColor() {
  const c = ICON_COLORS[colorIdx % ICON_COLORS.length];
  colorIdx++;
  return c;
}

// ─── 空白表单 ─────────────────────────────────────────────────────────────────────

type FormState = {
  name: string;
  channelId: string;
  serverUrl: string;
  wsUrl: string;
  credentialFields: CredentialField[];
};

function emptyForm(): FormState {
  return {
    name: "",
    channelId: "",
    serverUrl: "",
    wsUrl: "",
    credentialFields: [],
  };
}

// ─── JSON 模式：前端只做 JSON 合法性校验 ────────────────────────────────────────────
//
// 用户在 JSON Tab 粘贴的结构可能因业务而异（channel_id / custom_config 等），
// 具体字段是否合法由后端校验。前端只保证「粘贴的是合法 JSON 对象」。
type JsonValidationResult =
  | { ok: true; data: Record<string, unknown> }
  | { ok: false; error: string; line?: number };

const CHANNEL_ID_PATTERN = /^[A-Za-z0-9_]+$/;

// JSON 输入模式的前端校验：仅校验「是否填了合法 JSON 对象」，不校验具体字段。
// 字段维度的校验（必填项、格式、契约）交由后端处理，避免不同用户结构差异被误拦。
function validateChannelJson(text: string): JsonValidationResult {
  if (!text.trim()) {
    return { ok: false, error: "JSON 内容不能为空" };
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch (e) {
    const msg = e instanceof Error ? e.message : "解析失败";
    // 尝试从 V8 错误信息中提取行/列以做位置提示
    const posMatch = /position\s+(\d+)/.exec(msg);
    let line: number | undefined;
    if (posMatch) {
      const pos = Number(posMatch[1]);
      line = text.slice(0, pos).split("\n").length;
    }
    return { ok: false, error: `JSON 解析失败：${msg}`, line };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { ok: false, error: "根节点必须是 JSON 对象" };
  }
  return { ok: true, data: parsed as Record<string, unknown> };
}

function formToChannelJson(form: FormState): string {
  return JSON.stringify(
    {
      name: form.name,
      channelId: form.channelId,
      serverUrl: form.serverUrl,
      wsUrl: form.wsUrl,
      credentialFields: form.credentialFields.map((f) => ({ key: f.key, label: f.label })),
    },
    null,
    2,
  );
}

// JSON 输入 Tab 初始展示的示例文本（仅前端占位提示，不参与校验，不代表最终提交契约）
const JSON_INPUT_TEMPLATE = `{
  "channel_id": "\${CUSTOM_CHANNEL_ID}",
  "name": "Test Custom Channel",
  "custom_config": {
    "server": {
      "api_url": "https://example.com/api",
      "token": ""
    },
    "cred_fields": [
      {"key": "token", "label": "API Token"}
    ]
  }
}`;

// ─── 主组件 ──────────────────────────────────────────────────────────────────────

export default function ChannelConfig() {
  // 内置通道可见性（从 localStorage 初始化，与租户端共享）
  const [builtinVisibility, setBuiltinVisibility] = useState<Record<string, boolean>>(
    () => loadBuiltinChannelVisibility()
  );

  // 自定义通道列表（从 localStorage 初始化）
  const [customChannels, setCustomChannels] = useState<CustomChannel[]>(() => loadCustomChannels());

  // 监听其他标签页的变更
  useEffect(() => {
    const unsub = onCustomChannelsChange(() => {
      setCustomChannels(loadCustomChannels());
    });
    return unsub;
  }, []);

  // 监听内置通道可见性的跨标签页变更
  useEffect(() => {
    const unsub = onBuiltinChannelVisibilityChange(() => {
      setBuiltinVisibility(loadBuiltinChannelVisibility());
    });
    return unsub;
  }, []);

  // 弹窗状态（仅用于新增）
  const [dialogOpen, setDialogOpen] = useState(false);

  // Tab 切换
  const [activeTab, setActiveTab] = useState("builtin");
  const currentTab = CHANNEL_TABS.find((t) => t.id === activeTab)!;

  // 表单状态
  const [form, setForm] = useState<FormState>(emptyForm());

  // ── 新增通道弹窗：输入模式（表单 / JSON） ──
  const [inputMode, setInputMode] = useState<"form" | "json">("form");
  // JSON 模式下文本框内容（受控，键入时实时更新）
  const [jsonText, setJsonText] = useState<string>("");
  // JSON 模式下的实时解析结果：成功时把 form 等同物存在这里，失败时记录错误
  const jsonValidation = useMemo(() => validateChannelJson(jsonText), [jsonText]);

  // 删除确认弹窗
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  // 展开/折叠通道详情（展示 IM 服务器地址 + 用户凭证字段）
  const [expandedCustomId, setExpandedCustomId] = useState<string | null>(null);

  // ── 通道应用范围状态（每个通道独立） ──
  // 内置通道：{ channelId: { scope, groupIds } }
  const [builtinScopes, setBuiltinScopes] = useState<Record<string, { scope: "all" | "groups"; groupIds: string[] }>>(
    () => {
      // 初始化：全部为"全部用户"
      const init: Record<string, { scope: "all" | "groups"; groupIds: string[] }> = {};
      BUILTIN_CHANNELS.forEach((ch) => { init[ch.id] = { scope: "all", groupIds: [] }; });
      return init;
    }
  );

  // 自定义通道：{ channelId: { scope, groupIds } }
  const [customScopes, setCustomScopes] = useState<Record<string, { scope: "all" | "groups"; groupIds: string[] }>>({});

  // ── 同步到 localStorage ──
  const updateChannels = (channels: CustomChannel[]) => {
    setCustomChannels(channels);
    saveCustomChannels(channels);
  };

  // ── 打开新增弹窗 ──
  const openAddDialog = () => {
    const fresh = emptyForm();
    setForm(fresh);
    setInputMode("form");
    // JSON 输入 Tab 初始展示统一使用示例模板（前端提示），与表单校验逻辑无关
    setJsonText(JSON_INPUT_TEMPLATE);
    setDialogOpen(true);
  };

  // 弹窗内切换输入模式：切到 JSON 时统一回填示例模板作为前端提示
  const switchInputMode = (next: "form" | "json") => {
    if (next === inputMode) return;
    if (next === "json") {
      setJsonText(JSON_INPUT_TEMPLATE);
    }
    setInputMode(next);
  };

  // ── 保存（仅新增） ──
  const handleSave = () => {
    let active: FormState;
    if (inputMode === "json") {
      // 前端只校验 JSON 本身是否合法，不校验字段；字段合法性交给后端。
      if (!jsonValidation.ok) {
        toast.error(jsonValidation.error);
        return;
      }
      const raw = jsonValidation.data as Record<string, unknown>;
      const pickStr = (v: unknown) => (typeof v === "string" ? v : "");
      // 宽松取值：兼容 channelId / channel_id 两种写法。其它字段（serverUrl/wsUrl 等）
      // 在业务契约里可能嵌在 custom_config 里、结构因人而异，前端不猜也不校验，交后端处理。
      const name = pickStr(raw.name);
      const channelId = pickStr(raw.channelId) || pickStr(raw.channel_id);

      // 前端只提醒"是否填了"，不做格式校验；提示文案与表单模式保持完全一致。
      // 特别地，channelId 若仍是示例占位符 ${CUSTOM_CHANNEL_ID}，视为未填。
      const nameTrimmed = name.trim();
      const channelIdTrimmed = channelId.trim();

      if (!nameTrimmed) { toast.error("请填写通道名称"); return; }
      if (!channelIdTrimmed || channelIdTrimmed === "${CUSTOM_CHANNEL_ID}") {
        toast.error("请填写 Channel ID");
        return;
      }

      active = {
        name: nameTrimmed,
        channelId: channelIdTrimmed,
        serverUrl: "",
        wsUrl: "",
        credentialFields: [],
      };
    } else {
      active = form;
      if (!active.name.trim()) { toast.error("请填写通道名称"); return; }
      if (!active.channelId.trim()) { toast.error("请填写 Channel ID"); return; }
      if (!CHANNEL_ID_PATTERN.test(active.channelId)) {
        toast.error("Channel ID 仅支持英文字母、数字和下划线");
        return;
      }
      if (!active.serverUrl.trim()) { toast.error("请填写 Server URL"); return; }
      if (!active.wsUrl.trim()) { toast.error("请填写 WebSocket URL"); return; }
      for (const f of active.credentialFields) {
        if (!f.key.trim()) { toast.error("凭证字段 Key 不能为空"); return; }
        if (!f.label.trim()) { toast.error("凭证字段名称不能为空"); return; }
      }
    }

    const newCh: CustomChannel = {
      id: `custom_${Date.now()}`,
      name: active.name,
      channelId: active.channelId,
      serverUrl: active.serverUrl,
      wsUrl: active.wsUrl,
      credentialFields: active.credentialFields,
      visible: false,
      color: nextColor(),
    };
    updateChannels([...customChannels, newCh]);
    toast.success(`「${active.name}」已添加，默认不可见，开启「用户可见」后用户即可选择`);
    setDialogOpen(false);
  };

  // ── 删除自定义通道 ──
  const handleDelete = (id: string) => {
    updateChannels(customChannels.filter(ch => ch.id !== id));
    setDeleteConfirmId(null);
    if (expandedCustomId === id) setExpandedCustomId(null);
    toast.success("通道已删除");
  };

  // ── 切换自定义通道可见性 ──
  const toggleCustomVisible = (id: string, v: boolean) => {
    const updated = customChannels.map(ch => ch.id === id ? { ...ch, visible: v } : ch);
    updateChannels(updated);
    const ch = customChannels.find(c => c.id === id);
    toast.success(`「${ch?.name}」已${v ? "开启用户可见" : "关闭用户可见"}`);
  };

  // ── 凭证字段操作 ──
  const addCredentialField = () => {
    setForm(f => ({
      ...f,
      credentialFields: [...f.credentialFields, { id: `field_${Date.now()}`, key: "", label: "" }],
    }));
  };

  const removeCredentialField = (fieldId: string) => {
    setForm(f => ({ ...f, credentialFields: f.credentialFields.filter(x => x.id !== fieldId) }));
  };

  const updateCredentialFieldKey = (fieldId: string, key: string) => {
    setForm(f => ({
      ...f,
      credentialFields: f.credentialFields.map(x => x.id === fieldId ? { ...x, key } : x),
    }));
  };

  const updateCredentialFieldLabel = (fieldId: string, label: string) => {
    setForm(f => ({
      ...f,
      credentialFields: f.credentialFields.map(x => x.id === fieldId ? { ...x, label } : x),
    }));
  };

  return (
    <div className="page-enter">
      <AdminPageHeader title="通道配置" />

      {/* Tab 切换器（与 Agent 工具库同款 LineTabs：黑色下划线）
        * 停服态豁免：切换「内置通道 / 自定义通道」属于查看类操作（不产生变更），
        * 与其他 Tab 视图切换同档，需保持 100% 不透明与正常交互。
        * 原生 <button> 未设置 disabled，"停服前已禁用则延续禁用"约束
        * 通过原生 disabled 属性依然生效（此处无）。 */}
      <div className="mb-1" data-billing-exempt>
        <div className="flex items-center gap-2 border-b border-[#dbe6ff]">
          {CHANNEL_TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`relative px-4 py-3 transition-colors whitespace-nowrap ${
                activeTab === tab.id
                  ? "border-b-2 border-[#0A0A0A] -mb-px"
                  : ""
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

      {/* Tab 描述（仅一行） */}
      <div className="flex items-center gap-3 mt-3 mb-6">
        <CompactText as="p" tone="muted" className="leading-relaxed">
          {currentTab.description}
          {activeTab === "custom" && (
            <a
              href="#"
              className="inline-flex items-center gap-1 ml-2 text-[var(--text-brand)] hover:text-[var(--text-brand-deep)] transition-colors"
              onClick={(e) => e.preventDefault()}
            >
              <MetaText tone="inherit" className="underline underline-offset-2">自定义通道配置指引</MetaText>
              <svg className="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            </a>
          )}
        </CompactText>
      </div>

      {/* ── 内置通道 Tab ── */}
      {activeTab === "builtin" && (
        <SurfaceCard className="overflow-hidden">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead style={{ minWidth: 280 }}>产品</TableHead>
                <TableHead style={{ width: "100%", minWidth: 200 }}>应用范围</TableHead>
                <TableHead style={{ width: 160, minWidth: 160 }}>用户可见</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {BUILTIN_CHANNELS.map((ch) => (
                <TableRow key={ch.id}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <span className="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-[8px] border border-[var(--border)]" aria-hidden="true">
                        <img src={CHANNEL_ICON_SRC[ch.id]} alt="" className="h-full w-full scale-[1.04] object-contain" />
                      </span>
                      <BodyMedium tone="primary">{ch.name}</BodyMedium>
                    </div>
                  </TableCell>
                  <TableCell>
                    <ScopeSelect
                      scope={builtinScopes[ch.id]?.scope || "all"}
                      selectedGroupIds={builtinScopes[ch.id]?.groupIds || []}
                      groups={ALL_GROUPS}
                      maxVisibleBadges={5}
                      onConfirm={(scope, groupIds) => {
                        setBuiltinScopes((prev) => ({ ...prev, [ch.id]: { scope, groupIds } }));
                      }}
                    />
                  </TableCell>
                  <TableCell>
                    <Switch
                      checked={builtinVisibility[ch.id] || false}
                      onCheckedChange={(v) => {
                        const updated = { ...builtinVisibility, [ch.id]: v };
                        setBuiltinVisibility(updated);
                        saveBuiltinChannelVisibility(updated);
                        toast.success(`${ch.name} 已${v ? "开启用户可见" : "关闭用户可见"}`);
                      }}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      )}

      {/* ── 自定义通道 Tab ── */}
      {activeTab === "custom" && (
        <div className="space-y-3">
          {/* 操作行（表格外）：添加通道 */}
          <div className="flex items-center justify-end">
            <Button
              size="sm"
              onClick={openAddDialog}
            >
              <Plus className="w-4 h-4" />
              添加自定义通道
            </Button>
          </div>

          {customChannels.length === 0 ? (
            <SurfaceCard className="overflow-hidden">
              <Empty className="border-0 py-12">
                <EmptyHeader>
                  <EmptyMedia />
                  <EmptyTitle>暂无自定义通道</EmptyTitle>
                  <EmptyDescription>点击「添加通道」配置企业自研 IM 通道</EmptyDescription>
                </EmptyHeader>
              </Empty>
            </SurfaceCard>
          ) : (
            <SurfaceCard className="overflow-hidden">
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead style={{ minWidth: 280 }}>通道名</TableHead>
                    <TableHead style={{ width: "100%", minWidth: 200 }}>应用范围</TableHead>
                    <TableHead style={{ width: 160, minWidth: 160 }}>用户可见</TableHead>
                    <TableHead style={{ width: 160, minWidth: 160, maxWidth: 160 }}>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {customChannels.map((ch) => (
                    <Fragment key={ch.id}>
                      <TableRow
                        className="cursor-pointer"
                        onClick={() => setExpandedCustomId(expandedCustomId === ch.id ? null : ch.id)}
                      >
                        <TableCell>
                          <div className="flex items-center gap-2 min-w-0">
                            <span
                              className="shrink-0 w-4 h-4 inline-flex items-center justify-center text-[var(--text-muted)]"
                              aria-hidden="true"
                            >
                              {expandedCustomId === ch.id
                                ? <ChevronDown className="w-3.5 h-3.5" />
                                : <ChevronRight className="w-3.5 h-3.5" />
                              }
                            </span>
                            <BodyMedium tone="primary" className="truncate">{ch.name}</BodyMedium>
                            <Badge variant="secondary" className="shrink-0 font-mono">
                              {ch.channelId}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <ScopeSelect
                            scope={customScopes[ch.id]?.scope || "all"}
                            selectedGroupIds={customScopes[ch.id]?.groupIds || []}
                            groups={ALL_GROUPS}
                            maxVisibleBadges={5}
                            onConfirm={(scope, groupIds) => {
                              setCustomScopes((prev) => ({ ...prev, [ch.id]: { scope, groupIds } }));
                            }}
                          />
                        </TableCell>
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <Switch
                            checked={ch.visible}
                            onCheckedChange={(v) => toggleCustomVisible(ch.id, v)}
                          />
                        </TableCell>
                        <TableActionCell onClick={(e) => e.stopPropagation()}>
                          <Button
                            variant="link"
                            size="sm"
                            onClick={() => setDeleteConfirmId(ch.id)}
                          >
                            删除
                          </Button>
                        </TableActionCell>
                      </TableRow>

                      {/* 展开详情行：横跨四列 */}
                      {expandedCustomId === ch.id && (
                        <TableRow className="hover:bg-transparent">
                          <TableCell colSpan={4} className="bg-[#fafafa]/50">
                            <div className="space-y-3 py-1">
                              {/* IM 服务器地址 */}
                              <SurfaceInner className="px-4 py-3 space-y-2">
                                <CardTitle as="h4">IM 服务器地址</CardTitle>
                                <div className="space-y-1.5">
                                  <div className="flex items-center gap-2">
                                    <MetaMedium tone="secondary" className="w-24 shrink-0">Server URL</MetaMedium>
                                    <CodeText as="span" tone="body" className="break-all">{ch.serverUrl || "—"}</CodeText>
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <MetaMedium tone="secondary" className="w-24 shrink-0">WebSocket URL</MetaMedium>
                                    <CodeText as="span" tone="body" className="break-all">{ch.wsUrl || "—"}</CodeText>
                                  </div>
                                </div>
                              </SurfaceInner>
                              {/* 用户凭证字段 */}
                              <SurfaceInner className="px-4 py-3 space-y-2">
                                <CardTitle as="h4">用户凭证字段</CardTitle>
                                {ch.credentialFields.length === 0 ? (
                                  <MetaText as="p" tone="weak">无凭证字段</MetaText>
                                ) : (
                                  <div className="flex flex-wrap gap-2">
                                    {ch.credentialFields.map((f, idx) => (
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

      {/* ── 新增自定义通道弹窗 ── */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[720px] flex flex-col h-[min(640px,90vh)]">
          <DialogHeader className="shrink-0">
            <DialogTitle>添加自定义通道</DialogTitle>
            <HelperText>
              配置企业自研 IM 通道信息，保存后可在通道列表中管理可见性。
            </HelperText>
          </DialogHeader>

          <DialogBody
            className={
              inputMode === "json"
                ? "px-6 flex flex-col gap-4 overflow-hidden"
                : "px-6 space-y-4"
            }
          >
            {/* 顶部提醒条 */}
            <Alert variant="warning">
              <AlertCircle />
              <AlertDescription>
                使用自定义通道前，企业需先开发与 Agent 适配的 IM 插件，并前往
                <MetaMedium as="strong" tone="inherit">镜像管理</MetaMedium>
                页面，导入内置该插件的自定义镜像并将其设为生效版本，方可正常使用。
              </AlertDescription>
            </Alert>

            {/* ── 输入模式切换（表单输入 / JSON 输入） ── */}
            <SegmentedTabs
              ariaLabel="新增通道输入模式"
              tabs={[
                { id: "form", label: "表单输入" },
                { id: "json", label: "JSON 输入" },
              ]}
              active={inputMode}
              onChange={(id) => switchInputMode(id as "form" | "json")}
              fullWidth
            />

            {/* ── JSON 输入分支 ── */}
            {inputMode === "json" && (
              <section className="flex-1 min-h-0 flex flex-col gap-2">
                <MetaMedium as="label" tone="secondary">
                  通道配置（JSON）<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Textarea
                  spellCheck={false}
                  placeholder={'{\n  "channel_id": "${CUSTOM_CHANNEL_ID}",\n  "name": "Test Custom Channel",\n  "custom_config": {\n    "server": {\n      "api_url": "https://example.com/api",\n      "token": ""\n    },\n    "cred_fields": [\n      {"key": "token", "label": "API Token"}\n    ]\n  }\n}'}
                  value={jsonText}
                  onChange={(e) => setJsonText(e.target.value)}
                  className="font-mono text-xs leading-5 flex-1 min-h-0 resize-none [field-sizing:fixed] overflow-auto"
                />
              </section>
            )}

            {/* ── 第一部分：通道基础信息 ── */}
            {inputMode === "form" && (
              <section className="space-y-3">
                <CardTitle as="h3">通道基础信息</CardTitle>
                <div className="space-y-3">
                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary">
                      通道名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      placeholder="展示给用户的通道名字，如「内部 IM」"
                      value={form.name}
                      onChange={(e) => setForm(f => ({ ...f, name: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary">
                      Channel ID<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      placeholder="写入 agent.json 的 key，需与插件名一致"
                      value={form.channelId}
                      onChange={(e) => setForm(f => ({ ...f, channelId: e.target.value }))}
                      className="font-mono"
                    />
                    <HelperText>仅支持英文字母、数字和下划线，需与对应插件名保持一致</HelperText>
                  </div>
                </div>
              </section>
            )}

            {/* ── 第二部分：IM 服务器地址 ── */}
            {inputMode === "form" && (
              <section className="space-y-3">
                <CardTitle as="h3">IM 服务器地址</CardTitle>
                <div className="space-y-3">
                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary">
                      Server URL<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      placeholder="自定义 IM 的 HTTP API 地址"
                      value={form.serverUrl}
                      onChange={(e) => setForm(f => ({ ...f, serverUrl: e.target.value }))}
                      className="font-mono"
                    />
                  </div>
                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary">
                      WebSocket URL<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      placeholder="自定义 IM 的长连接地址，可与 Server URL 相同"
                      value={form.wsUrl}
                      onChange={(e) => setForm(f => ({ ...f, wsUrl: e.target.value }))}
                      className="font-mono"
                    />
                    <HelperText>WebSocket 地址可与 Server URL 相同，系统会自动处理协议转换</HelperText>
                  </div>
                </div>
              </section>
            )}

            {/* ── 第三部分：用户凭证字段 ── */}
            {inputMode === "form" && (
              <section className="space-y-3">
                <div className="flex items-center justify-between">
                  <CardTitle as="h3">用户凭证字段</CardTitle>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={addCredentialField}
                  >
                    <Plus className="w-3.5 h-3.5" />
                    添加字段
                  </Button>
                </div>

                <div className="space-y-2">
                  {form.credentialFields.length === 0 ? (
                    <div className="rounded-[4px] border border-dashed border-gray-200 px-4 py-3 text-center">
                      <HelperText>暂未添加凭证字段</HelperText>
                    </div>
                  ) : (
                    <SurfaceInner className="p-3 space-y-2">
                      {/* 表头 */}
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
                      {/* 字段行 */}
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
                              placeholder={idx % 2 === 0 ? "如 机器人的AccessKey" : "如 机器人的SecretKey"}
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
                    用户凭证的字段名称会展示在用户端，用户选择该通道后会看到对应的输入框
                  </HelperText>
                </div>
              </section>
            )}
          </DialogBody>

          <DialogFooter className="items-center">
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button variant="dialog-confirm" onClick={handleSave}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── 删除确认弹窗（警示弹窗） ── */}
      <AlertDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
      >
        <AlertDialogContent className="sm:max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">删除通道</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <BodyText as="p" tone="primary">
              删除后，该自定义通道将从用户端通道列表中移除，已接入该通道的 Agent 配置不受影响。
              <BodyText as="span" tone="danger">此操作不可撤销。</BodyText>
            </BodyText>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
