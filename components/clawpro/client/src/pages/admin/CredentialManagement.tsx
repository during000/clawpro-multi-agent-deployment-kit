/**
 * CredentialManagement - 凭据管理页
 * 管控端「安全审计」分类下的独立页面。
 *
 * 功能：
 *   - 凭据列表（名称/ID、启用状态、关联MCP、关联模型）
 *   - 创建凭据（自定义名称、Header/Query密钥）
 *   - 编辑凭据（已关联时提示风险）
 *   - 关联MCP/模型（批量关联，自动解绑原凭据）
 *   - 删除凭据（已关联不可删）
 *   - 搜索（按名称/ID）
 *   - 分页
 *   - 空状态
 *
 * 设计基线：clawpro-portable-design-skill
 *   - 端别：管控端 Admin
 *   - 圆角：4px（--cp-radius-md）
 *   - 按钮：claw-primary / claw-outline / destructive / link
 *   - 卡片：SurfaceCard
 *   - 表格：Table + TableActionCell + Pagination
 *   - 空状态：Empty 组件家族
 *   - 色彩：全部 token 化（--cp-*）
 */
import { useState, useMemo, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { Pagination, DialogPagination } from "@/components/ui/pagination";
import { SurfaceCard } from "@/components/ui/Surface";
import { DataTable } from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { MoreActionsDropdown } from "@/components/ui/more-actions-dropdown";
import type { MoreActionItem } from "@/components/ui/more-actions-dropdown";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from "@/components/ui/empty";
import {
  BodyText,
  HelperText,
  MetaText,
} from "@/components/ui/Typography";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "sonner";
import {
  Plus,
  Search,
  Trash2,
  X,
  AlertTriangle,
  Box,
  Cpu,
  ChevronDown,
  Lock,
} from "lucide-react";
import { credentialStore, type CredentialHeader, type CredentialItem } from "@/lib/credentialStore";

// ════════════════════════════════════════════════════════════
// Types
// ════════════════════════════════════════════════════════════

// CredentialHeader and CredentialItem are imported from credentialStore
type Credential = CredentialItem;

interface LinkedService {
  id: string;
  name: string;
  credentialId: string | null;
}

type LinkTarget = "mcp" | "model";

// ════════════════════════════════════════════════════════════
// Mock Data
// ════════════════════════════════════════════════════════════

const PAGE_SIZE = 10;

const MOCK_MCPS: LinkedService[] = [
  { id: "mcp-001", name: "GitHub MCP Server", credentialId: "cred-001" },
  { id: "mcp-002", name: "Jira MCP Server", credentialId: "cred-001" },
  { id: "mcp-003", name: "Slack MCP Server", credentialId: "cred-003" },
  { id: "mcp-004", name: "Confluence MCP", credentialId: "cred-003" },
  { id: "mcp-005", name: "Notion MCP", credentialId: "cred-003" },
  { id: "mcp-006", name: "Linear MCP", credentialId: null },
  { id: "mcp-007", name: "Figma MCP", credentialId: null },
];

const MOCK_MODELS: LinkedService[] = [
  { id: "model-001", name: "GPT-4o", credentialId: "cred-001" },
  { id: "model-002", name: "Claude 3.5 Sonnet", credentialId: "cred-003" },
  { id: "model-003", name: "DeepSeek V3", credentialId: "cred-003" },
  { id: "model-004", name: "Gemini 2.0 Flash", credentialId: null },
  { id: "model-005", name: "Llama 3.1 70B", credentialId: null },
];

// ════════════════════════════════════════════════════════════
// Helpers
// ════════════════════════════════════════════════════════════

function normalizeEntries(entries: CredentialHeader[]): CredentialHeader[] {
  return entries
    .filter((entry) => entry.key.trim() && entry.value.trim())
    .map((entry) => ({ key: entry.key.trim(), value: entry.value }));
}

function validateEntries(
  headers: CredentialHeader[],
  queryParams: CredentialHeader[],
): string | null {
  const hasIncomplete = [...headers, ...queryParams].some((entry) => {
    const hasKey = Boolean(entry.key.trim());
    const hasValue = Boolean(entry.value.trim());
    return hasKey !== hasValue;
  });
  if (hasIncomplete) return "请补全已填写参数的名称和值";

  const normalizedHeaders = normalizeEntries(headers);
  const normalizedQueryParams = normalizeEntries(queryParams);
  if (normalizedHeaders.length === 0 && normalizedQueryParams.length === 0) {
    return "请至少配置一条有效的 Header 或 Query 参数";
  }

  const headerKeys = normalizedHeaders.map((entry) => entry.key.toLowerCase());
  if (new Set(headerKeys).size !== headerKeys.length) return "Header 名称不能重复";

  const queryKeys = normalizedQueryParams.map((entry) => entry.key);
  if (new Set(queryKeys).size !== queryKeys.length) return "Query 参数名称不能重复";

  return null;
}

interface KeyValueFieldsProps {
  entries: CredentialHeader[];
  onChange: (entries: CredentialHeader[]) => void;
  allowEmpty?: boolean;
}

function KeyValueFields({ entries, onChange, allowEmpty = false }: KeyValueFieldsProps) {
  function updateEntry(index: number, field: "key" | "value", value: string) {
    onChange(entries.map((entry, i) => (i === index ? { ...entry, [field]: value } : entry)));
  }

  function removeEntry(index: number) {
    if (!allowEmpty && entries.length <= 1) return;
    onChange(entries.filter((_, i) => i !== index));
  }

  return (
    <>
      {entries.map((entry, index) => (
        <div key={index} className="flex items-center gap-2">
          <Input
            placeholder="请输入名称，最长64字符"
            maxLength={64}
            value={entry.key}
            onChange={(event) => updateEntry(index, "key", event.target.value)}
            className="flex-1"
          />
          <Input
            placeholder="请输入值，最长256字符"
            maxLength={256}
            value={entry.value}
            onChange={(event) => updateEntry(index, "value", event.target.value)}
            className="flex-1"
          />
          {(allowEmpty || entries.length > 1) && (
            <Button
              variant="claw-outline"
              size="sm"
              className="size-8 p-0 text-[var(--cp-text-weak)] hover:text-[var(--cp-text-danger)]"
              onClick={() => removeEntry(index)}
              aria-label="删除参数"
            >
              <X className="size-4" />
            </Button>
          )}
        </div>
      ))}
    </>
  );
}

function getServiceList(target: LinkTarget): LinkedService[] {
  switch (target) {
    case "mcp":
      return MOCK_MCPS;
    case "model":
      return MOCK_MODELS;
  }
}

function getServiceLabel(target: LinkTarget): string {
  switch (target) {
    case "mcp":
      return "MCP";
    case "model":
      return "模型";
  }
}

function getServiceIcon(target: LinkTarget) {
  switch (target) {
    case "mcp":
      return Box;
    case "model":
      return Cpu;
  }
}

function getLinkedServices(target: LinkTarget, credentialId: string): LinkedService[] {
  return getServiceList(target).filter((service) => service.credentialId === credentialId);
}

function hasLinkedServices(credentialId: string): boolean {
  return (
    getLinkedServices("mcp", credentialId).length > 0 ||
    getLinkedServices("model", credentialId).length > 0
  );
}

function LinkedServiceCell({ services }: { services: LinkedService[] }) {
  if (services.length === 0) {
    return <span className="text-xs text-[var(--cp-text-muted)]">未关联</span>;
  }

  const title = services.map((service) => service.name).join("\n");

  return (
    <div className="flex max-w-[220px] flex-col gap-0.5" title={title}>
      <BodyText className="truncate text-xs">
        {services.map((service) => service.name).join("、")}
      </BodyText>
    </div>
  );
}

// ════════════════════════════════════════════════════════════
// Main Component
// ════════════════════════════════════════════════════════════

export default function CredentialManagement() {
  // --- State ---
  const [credentials, setCredentials] = useState<Credential[]>(() => credentialStore.getAll());
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);

  // 订阅 store 变更，同步外部更新
  useEffect(() => {
    return credentialStore.subscribe(() => {
      setCredentials(credentialStore.getAll());
    });
  }, []);

  // Create dialog
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createHeaders, setCreateHeaders] = useState<CredentialHeader[]>([{ key: "", value: "" }]);
  const [createQueryParams, setCreateQueryParams] = useState<CredentialHeader[]>([]);

  // Edit dialog
  const [editingCredential, setEditingCredential] = useState<Credential | null>(null);
  const [editName, setEditName] = useState("");
  const [editHeaders, setEditHeaders] = useState<CredentialHeader[]>([{ key: "", value: "" }]);
  const [editQueryParams, setEditQueryParams] = useState<CredentialHeader[]>([]);

  // Delete dialog
  const [deleteTarget, setDeleteTarget] = useState<Credential | null>(null);

  // Link dialog
  const [linkTarget, setLinkTarget] = useState<LinkTarget | null>(null);
  const [linkCredential, setLinkCredential] = useState<Credential | null>(null);
  const [linkSelectedIds, setLinkSelectedIds] = useState<string[]>([]);
  const [linkSearch, setLinkSearch] = useState("");
  const [linkPage, setLinkPage] = useState(1);
  const LINK_PAGE_SIZE = 20;
  // 二次确认弹窗：所选服务中当前已关联到「其他凭据」的列表
  const [linkConfirmServices, setLinkConfirmServices] = useState<LinkedService[] | null>(null);

  // --- Computed ---
  const filtered = useMemo(() => {
    if (!search.trim()) return credentials;
    const q = search.trim().toLowerCase();
    return credentials.filter(
      (c) => c.name.toLowerCase().includes(q) || c.id.toLowerCase().includes(q),
    );
  }, [credentials, search]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = useMemo(() => {
    const start = (page - 1) * PAGE_SIZE;
    return filtered.slice(start, start + PAGE_SIZE);
  }, [filtered, page]);

  const hasData = credentials.length > 0;
  const hasFilteredData = filtered.length > 0;

  // --- Handlers: Enable toggle ---
  function handleToggleEnabled(id: string, checked: boolean) {
    credentialStore.update(id, (c) => ({ ...c, enabled: checked }));
    toast.success(checked ? "凭据已启用" : "凭据已禁用");
  }

  // --- Handlers: Create ---
  function openCreateDialog() {
    setCreateName("");
    setCreateHeaders([{ key: "", value: "" }]);
    setCreateQueryParams([]);
    setShowCreateDialog(true);
  }

  function handleCreate() {
    const trimmedName = createName.trim();
    if (!trimmedName) {
      toast.error("请输入凭据名称");
      return;
    }
    const validationError = validateEntries(createHeaders, createQueryParams);
    if (validationError) {
      toast.error(validationError);
      return;
    }
    const newCred: Credential = {
      id: `cred-${Date.now()}`,
      name: trimmedName,
      enabled: true,
      headers: normalizeEntries(createHeaders),
      queryParams: normalizeEntries(createQueryParams),
      linkedMcpCount: 0,
      linkedModelCount: 0,
      linkedApiCount: 0,
    };
    credentialStore.add(newCred);
    setShowCreateDialog(false);
    toast.success("凭据创建成功");
  }

  // --- Handlers: Edit ---
  function openEditDialog(cred: Credential) {
    setEditingCredential(cred);
    setEditName(cred.name);
    setEditHeaders(
      cred.headers.length > 0 || cred.queryParams.length > 0
        ? [...cred.headers]
        : [{ key: "", value: "" }],
    );
    setEditQueryParams([...cred.queryParams]);
  }

  function handleSaveEdit() {
    if (!editingCredential) return;
    const trimmedName = editName.trim();
    if (!trimmedName) {
      toast.error("请输入凭据名称");
      return;
    }
    const validationError = validateEntries(editHeaders, editQueryParams);
    if (validationError) {
      toast.error(validationError);
      return;
    }
    credentialStore.update(editingCredential.id, (c) => ({
      ...c,
      name: trimmedName,
      headers: normalizeEntries(editHeaders),
      queryParams: normalizeEntries(editQueryParams),
    }));
    setEditingCredential(null);
    toast.success("凭据已更新");
  }

  // --- Handlers: Delete ---
  function openDeleteConfirm(cred: Credential) {
    setDeleteTarget(cred);
  }

  function handleDelete() {
    if (!deleteTarget) return;
    credentialStore.remove(deleteTarget.id);
    setDeleteTarget(null);
    toast.success("凭据已删除");
  }

  // --- Handlers: Link ---
  function openLinkDialog(target: LinkTarget, cred: Credential) {
    setLinkTarget(target);
    setLinkCredential(cred);
    setLinkSearch("");
    setLinkPage(1);
    const services = getServiceList(target);
    // 预选当前已关联此凭据的服务
    setLinkSelectedIds(
      services.filter((s) => s.credentialId === cred.id).map((s) => s.id),
    );
  }

  // 点击"确认关联"：若所选服务当前已关联到其他凭据，需二次确认；否则直接关联
  function handleClickConfirmLink() {
    if (!linkTarget || !linkCredential) return;
    const list = getServiceList(linkTarget);
    const conflicts = linkSelectedIds
      .map((id) => list.find((s) => s.id === id))
      .filter(
        (svc): svc is LinkedService =>
          !!svc && svc.credentialId !== null && svc.credentialId !== linkCredential!.id,
      );
    if (conflicts.length > 0) {
      setLinkConfirmServices(conflicts);
      return;
    }
    doConfirmLink();
  }

  function doConfirmLink() {
    if (!linkTarget || !linkCredential) return;
    const label = getServiceLabel(linkTarget);

    // 更新 mock 服务的凭据关联（自动解绑旧凭据）
    const list = getServiceList(linkTarget);
    // 仅处理当前选中但未关联此凭据的服务（新关联）
    const toLink = linkSelectedIds.filter((id) => {
      const svc = list.find((s) => s.id === id);
      return svc && svc.credentialId !== linkCredential.id;
    });
    // 解除此凭据之前关联但本次取消选中的服务
    list.forEach((s) => {
      if (s.credentialId === linkCredential.id && !linkSelectedIds.includes(s.id)) {
        s.credentialId = null;
      }
    });
    // 重新关联选中的服务（自动解绑其他凭据）
    toLink.forEach((serviceId) => {
      const svc = list.find((s) => s.id === serviceId);
      if (svc) {
        if (svc.credentialId && svc.credentialId !== linkCredential.id) {
          const oldCred = credentials.find((c) => c.id === svc.credentialId);
          if (oldCred) {
            updateCredentialLinkCount(svc.credentialId, linkTarget, -1);
          }
        }
        svc.credentialId = linkCredential.id;
      }
    });

    // 更新凭据的关联计数
    credentialStore.update(linkCredential.id, (c) => {
      const countKey = getCountKey(linkTarget);
      return { ...c, [countKey]: linkSelectedIds.length };
    });

    setLinkConfirmServices(null);
    setLinkTarget(null);
    setLinkCredential(null);
    toast.success(`已关联 ${linkSelectedIds.length} 个${label}`);
  }

  function updateCredentialLinkCount(credId: string, target: LinkTarget, delta: number) {
    const countKey = getCountKey(target);
    credentialStore.update(credId, (c) => ({
      ...c,
      [countKey]: Math.max(0, (c as any)[countKey] + delta),
    }));
  }

  function getCountKey(target: LinkTarget): "linkedMcpCount" | "linkedModelCount" {
    switch (target) {
      case "mcp":
        return "linkedMcpCount";
      case "model":
        return "linkedModelCount";
    }
  }

  // --- Handlers: More actions ---
  function getMoreActions(cred: Credential): MoreActionItem[] {
    const hasLinks = hasLinkedServices(cred.id);
    const items: MoreActionItem[] = [
      {
        label: "关联模型",
        icon: Cpu,
        onClick: () => openLinkDialog("model", cred),
      },
      {
        label: "关联MCP",
        icon: Box,
        onClick: () => openLinkDialog("mcp", cred),
      },
    ];

    items.push({
      label: "删除",
      icon: Trash2,
      variant: "destructive",
      disabled: hasLinks,
      disabledReason: hasLinks ? "仅支持删除未关联服务的凭据" : undefined,
      onClick: () => {
        if (!hasLinks) openDeleteConfirm(cred);
      },
      separatorBefore: true,
    });

    return items;
  }

  const isEditingLinked = editingCredential && hasLinkedServices(editingCredential.id);

  // --- Render ---
  return (
    <div className="page-enter">
      {/* 页面标题 */}
      <AdminPageHeader
        title="凭据管理"
        description="集中管理企业内的密钥、令牌等敏感凭据，避免真实密钥暴露给 Agent，降低凭据泄露风险。"
      />

      {/* 空状态：没有任何凭据 */}
      {!hasData ? (
        <Empty className="border-0 py-20">
          <EmptyMedia />
          <EmptyHeader>
            <EmptyTitle>暂无凭据</EmptyTitle>
            <EmptyDescription>
              还没有配置任何凭据，请创建一个凭据来管理密钥信息
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button variant="claw-primary" onClick={openCreateDialog}>
              <Plus className="size-4" />
              创建凭据
            </Button>
          </EmptyContent>
        </Empty>
      ) : (
        <>
          {/* 工具栏：搜索 + 创建按钮 */}
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
              <div className="relative min-w-48 max-w-xs flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-[var(--cp-text-weak)] pointer-events-none" />
                <Input
                  placeholder="搜索凭据名称/ID"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                  className="pl-9 bg-[var(--cp-surface)]"
                />
              </div>
            </div>
            <Button variant="claw-primary" onClick={openCreateDialog}>
              <Plus className="size-4" />
              创建凭据
            </Button>
          </div>

          {/* 搜索结果为空 */}
          {!hasFilteredData ? (
            <SurfaceCard>
              <Empty className="border-0 py-16">
                <EmptyMedia />
                <EmptyHeader>
                  <EmptyTitle>未找到匹配的凭据</EmptyTitle>
                  <EmptyDescription>
                    尝试使用其他关键词搜索
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
              <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[var(--cp-border)]">
                <span className="justify-self-start text-xs leading-[1.5] text-[var(--cp-text-muted)]">
                  共 {filtered.length} 条记录
                </span>
              </div>
            </SurfaceCard>
          ) : (
            <>
              {/* 凭据表格 */}
              <SurfaceCard>
                <Table variant="white">
                  <TableHeader>
                    <TableRow>
                      <TableHead>凭据名称</TableHead>
                      <TableHead>关联MCP</TableHead>
                      <TableHead>关联模型</TableHead>
                      <TableHead>启用状态</TableHead>
                      <TableHead>操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {paged.map((cred) => (
                      <TableRow key={cred.id}>
                        <TableCell>
                          <BodyText className="text-xs">{cred.name}</BodyText>
                        </TableCell>
                        <TableCell>
                          <LinkedServiceCell services={getLinkedServices("mcp", cred.id)} />
                        </TableCell>
                        <TableCell>
                          <LinkedServiceCell services={getLinkedServices("model", cred.id)} />
                        </TableCell>
                        <TableCell>
                          {cred.enabled && hasLinkedServices(cred.id) ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="inline-flex cursor-not-allowed">
                                  <Switch
                                    checked={cred.enabled}
                                    disabled
                                    className="pointer-events-none"
                                  />
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                仅支持禁用未关联服务的凭据
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <Switch
                              checked={cred.enabled}
                              onCheckedChange={(checked) =>
                                handleToggleEnabled(cred.id, checked)
                              }
                            />
                          )}
                        </TableCell>
                        <TableActionCell>
                          <Button
                            variant="link"
                            onClick={() => openEditDialog(cred)}
                          >
                            编辑
                          </Button>
                          <MoreActionsDropdown
                            items={getMoreActions(cred)}
                            triggerType="text"
                          />
                        </TableActionCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {/* 分页 */}
                <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[var(--cp-border)]">
                  <span className="justify-self-start text-xs leading-[1.5] text-[var(--cp-text-muted)]">
                    共 {filtered.length} 条记录
                  </span>
                  {totalPages > 1 && (
                    <nav className="justify-self-end">
                      <Pagination
                        total={filtered.length}
                        current={page}
                        pageSize={PAGE_SIZE}
                        onChange={(p) => setPage(p)}
                        hideOnSinglePage
                      />
                    </nav>
                  )}
                </div>
              </SurfaceCard>
            </>
          )}
        </>
      )}

      {/* ═══ 创建凭据弹窗 ═══ */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>创建凭据</DialogTitle>
          </DialogHeader>
          <DialogBody className="flex flex-col gap-4 px-6">
            {/* 模型密钥填写格式提示 */}
            <Alert variant="info">
              <AlertInfoIcon className="size-4" />
              <AlertDescription>
                <p className="font-medium">模型密钥填写规范（填写错误将导致无法正常调用模型）：</p>
                <p>
                  OpenAI 协议：密钥名称填 authorization，密钥值 = 前缀 Bearer + 空格 + API Key，如{" "}
                  <code className="rounded-[4px] border border-[var(--alert-info-border)] bg-[var(--cp-surface)] px-1 font-mono text-[12px]">
                    Bearer sk-xxxxxx
                  </code>
                </p>
                <p>Anthropic 协议：密钥名称填 x-api-key，密钥值直接填 API Key，不加前缀</p>
              </AlertDescription>
            </Alert>

            {/* 凭据名称 */}
            <div className="flex flex-col gap-2">
              <Label>
                凭据名称 <span className="text-[var(--cp-text-danger)]">*</span>
              </Label>
              <Input
                placeholder="请输入凭据名称"
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
              />
            </div>

            {/* Header 配置：默认方式 */}
            {createHeaders.length > 0 && (
              <div className="flex flex-col gap-3">
                <Label>Header</Label>
                <KeyValueFields
                  entries={createHeaders}
                  onChange={setCreateHeaders}
                  allowEmpty={createQueryParams.length > 0}
                />
              </div>
            )}

            {/* Query 配置：仅添加后展示 */}
            {createQueryParams.length > 0 && (
              <div className="flex flex-col gap-3">
                <Label>Query 参数</Label>
                <KeyValueFields
                  entries={createQueryParams}
                  onChange={setCreateQueryParams}
                  allowEmpty={createHeaders.length > 0}
                />

              </div>
            )}

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="claw-outline" size="sm" className="self-start">
                  <Plus className="size-3.5" />
                  添加密钥
                  <ChevronDown className="size-3.5" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-40">
                <DropdownMenuItem
                  onClick={() => setCreateHeaders([...createHeaders, { key: "", value: "" }])}
                >
                  添加为 Header 参数
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setCreateQueryParams([...createQueryParams, { key: "", value: "" }])
                  }
                >
                  添加为 Query 参数
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </DialogBody>
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => setShowCreateDialog(false)}>
              取消
            </Button>
            <Button variant="claw-primary" onClick={handleCreate}>
              创建
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ═══ 编辑凭据弹窗 ═══ */}
      <Dialog
        open={editingCredential !== null}
        onOpenChange={(open) => {
          if (!open) setEditingCredential(null);
        }}
      >
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>编辑凭据</DialogTitle>
          </DialogHeader>
          <DialogBody className="flex flex-col gap-4 px-6">
            {/* 密钥填写与修改提示（含已关联风险提示） */}
            <Alert variant="info">
              <AlertInfoIcon className="size-4" />
              <AlertDescription>
                {isEditingLinked && (
                  <p className="font-semibold text-[var(--alert-warning-icon)]">
                    当前密钥可能已被使用，若修改密钥会导致业务调用不成功，请谨慎操作
                  </p>
                )}
                <p className="font-medium">模型密钥填写规范（填写错误将导致无法正常调用模型）：</p>
                <p>
                  OpenAI 协议：密钥名称填 authorization，密钥值 = 前缀 Bearer + 空格 + API Key，如{" "}
                  <code className="rounded-[4px] border border-[var(--alert-info-border)] bg-[var(--cp-surface)] px-1 font-mono text-[12px]">
                    Bearer sk-xxxxxx
                  </code>
                </p>
                <p>Anthropic 协议：密钥名称填 x-api-key，密钥值直接填 API Key，不加前缀</p>
              </AlertDescription>
            </Alert>

            {/* 凭据名称 */}
            <div className="flex flex-col gap-2">
              <Label>
                凭据名称 <span className="text-[var(--cp-text-danger)]">*</span>
              </Label>
              <Input
                placeholder="请输入凭据名称"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
              />
            </div>

            {/* Header 配置：默认方式 */}
            {editHeaders.length > 0 && (
              <div className="flex flex-col gap-3">
                <Label>Header</Label>
                <KeyValueFields
                  entries={editHeaders}
                  onChange={setEditHeaders}
                  allowEmpty={editQueryParams.length > 0}
                />
              </div>
            )}

            {/* Query 配置：仅已有配置时展示 */}
            {editQueryParams.length > 0 && (
              <div className="flex flex-col gap-3">
                <Label>Query 参数</Label>
                <KeyValueFields
                  entries={editQueryParams}
                  onChange={setEditQueryParams}
                  allowEmpty={editHeaders.length > 0}
                />

              </div>
            )}

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="claw-outline" size="sm" className="self-start">
                  <Plus className="size-3.5" />
                  添加密钥
                  <ChevronDown className="size-3.5" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-40">
                <DropdownMenuItem
                  onClick={() => setEditHeaders([...editHeaders, { key: "", value: "" }])}
                >
                  添加为 Header 参数
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setEditQueryParams([...editQueryParams, { key: "", value: "" }])
                  }
                >
                  添加为 Query 参数
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </DialogBody>
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => setEditingCredential(null)}>
              取消
            </Button>
            <Button variant="claw-primary" onClick={handleSaveEdit}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ═══ 删除确认弹窗 ═══ */}
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除凭据</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除凭据「{deleteTarget?.name}」吗？此操作不可撤销。
              {deleteTarget && hasLinkedServices(deleteTarget.id) && (
                <span className="block mt-2 text-[var(--cp-text-danger)]">
                  该凭据已关联服务，请先解除所有关联后再删除。
                </span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={deleteTarget !== null && hasLinkedServices(deleteTarget.id)}
              >
                确认删除
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ═══ 关联MCP/模型弹窗 ═══ */}
      <Dialog
        open={linkTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setLinkTarget(null);
            setLinkCredential(null);
          }
        }}
      >
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              关联{linkTarget ? getServiceLabel(linkTarget) : ""}到凭据「{linkCredential?.name}」
            </DialogTitle>
          </DialogHeader>
          <DialogBody className="flex flex-col gap-3 px-6">
            <HelperText>
              搜索并选择要关联的{linkTarget ? getServiceLabel(linkTarget) : ""}，支持批量选择。
              若所选{linkTarget ? getServiceLabel(linkTarget) : ""}已关联其他凭据，关联后将自动解绑原凭据；
              当前已关联该凭据的{linkTarget ? getServiceLabel(linkTarget) : ""}不支持在此处取消关联。
            </HelperText>

            {/* 搜索框 */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-[var(--cp-text-weak)] pointer-events-none" />
              <Input
                placeholder={`搜索${linkTarget ? getServiceLabel(linkTarget) : ""}名称`}
                value={linkSearch}
                onChange={(e) => {
                  setLinkSearch(e.target.value);
                  setLinkPage(1);
                }}
                className="pl-9"
              />
            </div>

            {/* 已选中的服务（始终展示） */}
            {linkTarget && linkSelectedIds.length > 0 && (
              <div className="flex flex-wrap items-center gap-1.5">
                <MetaText>已选 {linkSelectedIds.length} 个：</MetaText>
                {linkSelectedIds.map((id) => {
                  const svc = getServiceList(linkTarget).find((s) => s.id === id);
                  if (!svc) return null;
                  // 当前已关联该凭据的服务不支持解绑，不展示移除按钮
                  const isBound = svc.credentialId === linkCredential?.id;
                  return (
                    <span
                      key={id}
                      className="inline-flex items-center gap-1 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-2 py-0.5 text-xs text-[var(--cp-text-title)]"
                    >
                      {svc.name}
                      {isBound ? (
                        <span
                          className="ml-0.5 inline-flex size-4 items-center justify-center text-[var(--cp-text-weak)]"
                          title="当前已关联该凭据，不支持解绑"
                        >
                          <Lock className="size-3" />
                        </span>
                      ) : (
                        <button
                          type="button"
                          className="ml-0.5 inline-flex size-4 items-center justify-center rounded-sm text-[var(--cp-text-weak)] hover:text-[var(--cp-text-danger)] transition-colors"
                          onClick={() =>
                            setLinkSelectedIds((prev) => prev.filter((x) => x !== id))
                          }
                        >
                          <X className="size-3" />
                        </button>
                      )}
                    </span>
                  );
                })}
              </div>
            )}

            {/* 列表（表格 + 分页） */}
            {linkTarget && (() => {
              const allServices = getServiceList(linkTarget);
              const filtered = linkSearch.trim()
                ? allServices.filter((svc) =>
                    svc.name.toLowerCase().includes(linkSearch.trim().toLowerCase()),
                  )
                : allServices;
              const totalCount = filtered.length;
              const totalPages = Math.max(1, Math.ceil(totalCount / LINK_PAGE_SIZE));
              const safePage = Math.min(linkPage, totalPages);
              const start = (safePage - 1) * LINK_PAGE_SIZE;
              const paged = filtered.slice(start, start + LINK_PAGE_SIZE);

              if (allServices.length === 0) {
                return (
                  <div className="flex items-center justify-center py-10 text-xs text-[var(--cp-text-muted)]">
                    暂无可关联的{linkTarget ? getServiceLabel(linkTarget) : ""}
                  </div>
                );
              }

              return (
                <>
                  <DataTable<LinkedService>
                    size="compact"
                    variant="gray-header"
                    rowKey="id"
                    dataSource={paged}
                    emptyText={`未找到匹配的${linkTarget ? getServiceLabel(linkTarget) : ""}`}
                    columns={[
                      {
                        key: "name",
                        title: `${linkTarget ? getServiceLabel(linkTarget) : ""}名称`,
                        render: (_, record) => (
                          <div className="flex flex-col gap-0.5">
                            <span className="font-medium">{record.name}</span>
                            {record.credentialId !== null &&
                              record.credentialId !== linkCredential?.id && (
                                <span className="text-[11px] text-[var(--cp-text-muted)]">
                                  已关联：{credentials.find((c) => c.id === record.credentialId)?.name ?? record.credentialId}
                                  （关联后将自动解绑）
                                </span>
                              )}
                          </div>
                        ),
                      },
                      {
                        key: "status",
                        title: "当前关联",
                        align: "right",
                        width: 140,
                        render: (_, record) => {
                          if (record.credentialId === null) {
                            return (
                              <span className="text-[var(--cp-text-muted)]">未关联</span>
                            );
                          }
                          if (record.credentialId === linkCredential?.id) {
                            return (
                              <span
                                className="inline-flex items-center gap-1 rounded-full bg-[#ECFDF5] px-2 py-0.5 text-[11px] font-medium text-[#047857]"
                                title="当前已绑定，不支持解绑"
                              >
                                <Lock className="size-2.5" />
                                当前凭据
                              </span>
                            );
                          }
                          const linkedName = credentials.find(
                            (c) => c.id === record.credentialId,
                          )?.name ?? record.credentialId;
                          return (
                            <span
                              className="inline-block max-w-[140px] truncate align-middle"
                              title={linkedName}
                            >
                              {linkedName}
                            </span>
                          );
                        },
                      },
                    ]}
                    rowSelection={{
                      selectedKeys: linkSelectedIds,
                      onChange: (keys) => setLinkSelectedIds(keys),
                      preserveSelectedKeys: true,
                      // 当前已关联该凭据的服务不支持解绑，禁用其复选框（保持选中）
                      getCheckboxProps: (record) => ({
                        disabled: record.credentialId === linkCredential?.id,
                      }),
                    }}
                    pagination={false}
                  />

                  <DialogPagination
                    total={totalCount}
                    currentPage={safePage}
                    totalPages={totalPages}
                    onPrevPage={() => setLinkPage(Math.max(1, safePage - 1))}
                    onNextPage={() => setLinkPage(Math.min(totalPages, safePage + 1))}
                    className="pt-2"
                  />
                </>
              );
            })()}
          </DialogBody>
          <DialogFooter>
            <Button
              variant="claw-outline"
              onClick={() => {
                setLinkTarget(null);
                setLinkCredential(null);
              }}
            >
              取消
            </Button>
            <Button variant="claw-primary" onClick={handleClickConfirmLink}>
              确认关联 ({linkSelectedIds.length})
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ═══ 关联二次确认弹窗（所选服务已关联其他凭据）═══ */}
      <AlertDialog
        open={linkConfirmServices !== null}
        onOpenChange={(open) => {
          if (!open) setLinkConfirmServices(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认重新关联</AlertDialogTitle>
            <AlertDialogDescription>
              以下 {linkConfirmServices?.length ?? 0} 个
              {linkTarget ? getServiceLabel(linkTarget) : ""}当前已关联其他凭据，
              关联到「{linkCredential?.name}」后将自动解绑原凭据，是否继续？
            </AlertDialogDescription>
          </AlertDialogHeader>
          {linkConfirmServices && linkConfirmServices.length > 0 && (
            <div className="max-h-48 overflow-y-auto rounded-[8px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 py-2">
              <ul className="flex flex-col gap-1.5">
                {linkConfirmServices.map((svc) => (
                  <li
                    key={svc.id}
                    className="flex items-center justify-between gap-2 text-xs"
                  >
                    <span className="truncate font-medium text-[var(--cp-text-title)]">
                      {svc.name}
                    </span>
                    <span className="shrink-0 text-[var(--cp-text-muted)]">
                      原凭据：
                      {credentials.find((c) => c.id === svc.credentialId)?.name ??
                        svc.credentialId}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
          <div className="flex items-start gap-2 rounded-[8px] border border-[var(--cp-warning-border,var(--cp-border))] bg-[var(--cp-warning-bg,var(--cp-surface))] px-3 py-2 text-xs leading-relaxed text-[var(--cp-text-muted)]">
            <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--cp-warning-text,var(--cp-text-muted))]" />
            <span>
              请确保「{linkCredential?.name}」凭据真实有效，否则正在应用相关
              {linkTarget ? getServiceLabel(linkTarget) : "模型/MCP"}
              的 Agent 将无法正常调用，从而影响业务运行。
            </span>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button variant="claw-primary" onClick={doConfirmLink}>
                确认关联
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
