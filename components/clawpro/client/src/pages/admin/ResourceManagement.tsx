/**
 * ResourceManagement - 资源管理页
 * 管控端「Agent 启动配置」分组下，管理企业内的 Agent 资源策略模板。
 *
 * 设计基线：clawpro-portable-design-skill / admin.md
 *   - 标题区：AdminPageHeader
 *   - 策略说明：使用 Alert operation-info 弱提示
 *   - 表格组件：使用 @/components/ui/table 实现标准化管理端表格
 */
import { useState, useMemo, useEffect } from "react";

import { Plus, Info } from "lucide-react";
import { GuidePointBubble } from "@/components/onboarding";
import { GroupSelect } from "@/components/GroupSelect";
import type { UserGroup } from "@/pages/admin/MemberManagement/types";
import { SectionTitle } from "@/components/ui/Typography";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { Input } from "@/components/ui/input";
import {
  Alert,
  AlertDescription,
  AlertOperationInfoIcon,
} from "@/components/ui/alert";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusTag } from "@/components/ui/status-tag";
import { SurfaceCard } from "@/components/ui/Surface";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Pagination } from "@/components/ui/pagination";
import { toast } from "sonner";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";

// ─── 资源策略数据接口 ───────────────────────────────────────────────────
interface ResourcePolicy {
  id: string;
  name: string;
  /** 应用范围：分组 ID 列表（preset 固定为 ["preset-all"]） */
  scopeIds: string[];
  /** 应用范围展示名（与 scopeIds 对应，用于表格展示） */
  scopeNames: string[];
  type: "preset" | "group";
  billingMode: "subscription" | "payg";
  instanceConfig: string;
  systemDiskType: string;
  systemDiskSize: number;
  assignPublicIp: boolean;
  bandwidthBillingMode: "monthly" | "traffic";
  bandwidthLimit: number;
}

// ─── Mock 分组数据（UserGroup 格式，与平台策略页保持一致） ───────────────
const MOCK_GROUPS: UserGroup[] = [
  { id: "mgrp-product",  name: "产品组",          parentId: null,         source: "manual", readonly: false },
  { id: "mgrp-rd",       name: "研发组",          parentId: null,         source: "manual", readonly: false },
  { id: "mgrp-rd-fe",    name: "研发-前端",        parentId: "mgrp-rd",    source: "manual", readonly: false },
  { id: "mgrp-rd-be",    name: "研发-后端",        parentId: "mgrp-rd",    source: "manual", readonly: false },
  { id: "mgrp-design",   name: "设计组",          parentId: null,         source: "manual", readonly: false },
  { id: "mgrp-ops",      name: "产品运营与市场推广团队", parentId: null,    source: "manual", readonly: false },
  { id: "mgrp-test",     name: "测试组",          parentId: null,         source: "manual", readonly: false },
  { id: "mgrp-finance",  name: "财务组",          parentId: null,         source: "manual", readonly: false },
  { id: "mgrp-security", name: "安全组",          parentId: null,         source: "manual", readonly: false },
];

/** 根据分组 ID 列表查询展示名 */
function resolveGroupNames(ids: string[]): string[] {
  return ids.map((id) => MOCK_GROUPS.find((g) => g.id === id)?.name ?? id);
}

const INSTANCE_CONFIGS = [
  { value: "Ai2.MEDIUM2（2核2GiB）", label: "2核2GiB", desc: "Ai2.MEDIUM2｜适合轻量任务、低成本测试场景" },
  { value: "Ai2.MEDIUM4（2核4GiB）", label: "2核4GiB", desc: "Ai2.MEDIUM4｜适合轻量任务、低并发使用场景" },
  { value: "Ai2.LARGE8（4核8GiB）", label: "4核8GiB", desc: "Ai2.LARGE8｜适合日常使用、通用 Agent 场景" },
  { value: "Ai2.2XLARGE16（8核16GiB）", label: "8核16GiB", desc: "Ai2.2XLARGE16｜适合高负载任务、复杂工具调用场景" },
];

const DISK_TYPES = ["通用型SSD云硬盘", "高性能云硬盘", "SSD 云硬盘"];

/** 获取指定计费模式的带宽上限（单位 Mbps） */
function getBandwidthMaxLimit(billingMode: "monthly" | "traffic"): number {
  return billingMode === "monthly" ? 20 : 200;
}

/** 解析规格选项在表格中的显示文案 */
function tableInstanceConfigDisplay(config: string): string {
  if (config.includes("2核2GiB")) return "2核2GiB";
  if (config.includes("2核4GiB")) return "2核4GiB";
  if (config.includes("4核8GiB")) return "4核8GiB";
  if (config.includes("8核16GiB")) return "8核16GiB";
  return config;
}

/** 获取分组的完整路径（含祖先，用 " / " 分隔） */
function getScopePath(groupId: string, groups: UserGroup[]): string {
  const g = groups.find((x) => x.id === groupId);
  if (!g) return groupId;
  const parts: string[] = [g.name];
  let cur = g;
  while (cur.parentId) {
    const parent = groups.find((x) => x.id === cur.parentId);
    if (!parent) break;
    parts.unshift(parent.name);
    cur = parent;
  }
  return parts.join(" / ");
}

/**
 * 分组选择聚合（替换式，对齐 GroupSelect 共享聚合逻辑）：
 * 若某父分组下全部子分组已选中，且父分组自身未被其他策略占用，则子分组替换为父分组。
 * 父分组被占用（disabled）时不聚合，保留子分组独立选中；用于编辑回显与保存前的防御性归一化。
 */
function normalizeScopeIds(ids: string[], groups: UserGroup[], disabledIds: Set<string>): string[] {
  const result = new Set(ids);
  let changed = true;
  while (changed) {
    changed = false;
    for (const g of groups) {
      if (result.has(g.id)) continue;
      // 父分组自身被占用 → 不可作为聚合目标
      if (disabledIds.has(g.id)) continue;
      const children = groups.filter((c) => c.parentId === g.id);
      if (children.length === 0) continue;
      if (children.some((c) => disabledIds.has(c.id))) continue;
      if (children.every((c) => result.has(c.id))) {
        children.forEach((c) => result.delete(c.id));
        result.add(g.id);
        changed = true;
      }
    }
  }
  return Array.from(result);
}

export default function ResourceManagement() {
  // ── 资源策略列表 ──
  const [policies, setPolicies] = useState<ResourcePolicy[]>([
    {
      id: "preset",
      name: "企业默认资源策略",
      scopeIds: ["preset-all"],
      scopeNames: ["全部用户"],
      type: "preset",
      billingMode: "subscription",
      instanceConfig: "Ai2.LARGE8（4核8GiB）",
      systemDiskType: "高性能云硬盘",
      systemDiskSize: 80,
      assignPublicIp: true,
      bandwidthBillingMode: "traffic",
      bandwidthLimit: 5,
    },
    {
      id: "rd-group",
      name: "研发组高配策略",
      scopeIds: ["mgrp-rd"],
      scopeNames: ["研发组"],
      type: "group",
      billingMode: "payg",
      instanceConfig: "Ai2.2XLARGE16（8核16GiB）",
      systemDiskType: "SSD 云硬盘",
      systemDiskSize: 100,
      assignPublicIp: false,
      bandwidthBillingMode: "traffic",
      bandwidthLimit: 0,
    },
    {
      id: "test-group",
      name: "测试组资源策略",
      scopeIds: ["mgrp-test"],
      scopeNames: ["测试组"],
      type: "group",
      billingMode: "subscription",
      instanceConfig: "Ai2.MEDIUM4（2核4GiB）",
      systemDiskType: "高性能云硬盘",
      systemDiskSize: 50,
      assignPublicIp: true,
      bandwidthBillingMode: "monthly",
      bandwidthLimit: 5,
    },
  ]);

  // ── 弹窗与表单状态 ──
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogMode, setDialogOpenMode] = useState<"add" | "edit">("add");
  const [selectedPolicy, setSelectedPolicy] = useState<ResourcePolicy | null>(null);

  // 表单字段
  const [formName, setFormName] = useState("");
  // displayScopeIds：聚合后的应用范围值，驱动 GroupSelect 展示、tag 和提交
  const [displayScopeIds, setDisplayScopeIds] = useState<string[]>([]);
  const [formBillingMode, setFormBillingMode] = useState<"subscription" | "payg">("subscription");
  const [formInstanceConfig, setFormInstanceConfig] = useState("Ai2.MEDIUM4（2核4GiB）");
  const [formDiskType, setFormDiskType] = useState("通用型SSD云硬盘");
  const [formDiskSize, setFormDiskSize] = useState<number | "">(50);
  
  // 系统盘容量取值范围：30GiB～2048GiB
  const diskRange = useMemo(() => ({
    minSize: 30,
    maxSize: 2048,
  }), []);
  const [formAssignPublicIp, setFormAssignPublicIp] = useState(true);
  const [formBandwidthBilling, setFormBandwidthBilling] = useState<"monthly" | "traffic">("monthly");
  const [formBandwidthLimit, setFormBandwidthLimit] = useState(5);

  // 分页
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 删除确认
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [policyToDelete, setPolicyToDelete] = useState<ResourcePolicy | null>(null);

  // 预设策略引导气泡
  const [showPresetHint, setShowPresetHint] = useState(true);
  const [presetHintPos, setPresetHintPos] = useState<{ left: number; top: number } | null>(null);

  useEffect(() => {
    if (!showPresetHint) return;
    const el = document.querySelector<HTMLElement>('[data-bubble-anchor="preset-policy"]');
    if (el) {
      const rect = el.getBoundingClientRect();
      setPresetHintPos({ left: rect.right + 8, top: rect.top + rect.height / 2 });
    }
  }, [showPresetHint]);


  // 已被其他策略占用的分组 ID（编辑时排除自身）
  const usedGroupIds = useMemo(() => {
    const editing = dialogMode === "edit" ? selectedPolicy?.id : null;
    return new Set(
      policies
        .filter((p) => p.id !== editing && p.type !== "preset")
        .flatMap((p) => p.scopeIds)
    );
  }, [policies, dialogMode, selectedPolicy]);

  // 分页切片
  const pagedPolicies = useMemo(() => {
    const start = (page - 1) * pageSize;
    return policies.slice(start, start + pageSize);
  }, [policies, page, pageSize]);

  // ── 打开新增弹窗 ──
  const handleOpenAdd = () => {
    setFormName("");
    setDisplayScopeIds([]);
    setFormBillingMode("subscription");
    setFormInstanceConfig("Ai2.MEDIUM4（2核4GiB）");
    setFormDiskType("通用型SSD云硬盘");
    setFormDiskSize(50);
    setFormAssignPublicIp(true);
    setFormBandwidthBilling("monthly");
    setFormBandwidthLimit(5);
    setDialogOpenMode("add");
    setSelectedPolicy(null);
    setDialogOpen(true);
  };

  // ── 打开编辑弹窗 ──
  const handleOpenEdit = (policy: ResourcePolicy) => {
    setSelectedPolicy(policy);
    setFormName(policy.name);
    const initIds = policy.type === "preset"
      ? []
      : normalizeScopeIds(policy.scopeIds, MOCK_GROUPS, usedGroupIds);
    setDisplayScopeIds(initIds);
    setFormBillingMode(policy.billingMode);
    setFormInstanceConfig(policy.instanceConfig);
    setFormDiskType(policy.systemDiskType);
    setFormDiskSize(policy.systemDiskSize);
    setFormAssignPublicIp(policy.assignPublicIp);
    setFormBandwidthBilling(policy.bandwidthBillingMode);
    setFormBandwidthLimit(policy.bandwidthLimit);
    setDialogOpenMode("edit");
    setDialogOpen(true);
  };

  // ── 保存表单 ──
  const handleSave = () => {
    // 验证系统盘容量
    if (formDiskSize === "" || formDiskSize < diskRange.minSize || formDiskSize > diskRange.maxSize) {
      toast.error(`系统盘容量必须在 ${diskRange.minSize}GiB～${diskRange.maxSize}GiB 之间`);
      return;
    }
    // 如果分配公网 IP，验证带宽上限
    if (formAssignPublicIp) {
      const maxLimit = getBandwidthMaxLimit(formBandwidthBilling);
      if (formBandwidthLimit > maxLimit) {
        if (formBandwidthBilling === "monthly") {
          toast.error("带宽上限不能超过 20Mbps");
        } else {
          toast.error("带宽上限不能超过 200Mbps");
        }
        return;
      }
      if (formBandwidthLimit < 1) {
        toast.error("带宽上限不能低于 1Mbps");
        return;
      }
    }
    if (dialogMode === "add") {
      if (!formName.trim()) {
        toast.error("请输入策略名称");
        return;
      }
      if (policies.some((p) => p.name === formName.trim())) {
        toast.error("策略名称已存在");
        return;
      }
      if (displayScopeIds.length === 0) {
        toast.error("请选择应用范围");
        return;
      }
      const newPolicy: ResourcePolicy = {
        id: `group-${Date.now()}`,
        name: formName.trim(),
        scopeIds: displayScopeIds,
        scopeNames: resolveGroupNames(displayScopeIds),
        type: "group",
        billingMode: formBillingMode,
        instanceConfig: formInstanceConfig,
        systemDiskType: formDiskType,
        systemDiskSize: formDiskSize,
        assignPublicIp: formAssignPublicIp,
        bandwidthBillingMode: formAssignPublicIp ? formBandwidthBilling : "traffic",
        bandwidthLimit: formAssignPublicIp ? formBandwidthLimit : 0,
      };
      setPolicies([...policies, newPolicy]);
      toast.success(`策略「${formName.trim()}」添加成功`);
    } else if (dialogMode === "edit" && selectedPolicy) {
      if (!formName.trim()) {
        toast.error("策略名称不能为空");
        return;
      }
      const isPreset = selectedPolicy.type === "preset";
      const updatedName = isPreset ? "企业默认资源策略" : formName.trim();
      if (
        updatedName !== selectedPolicy.name &&
        policies.some((p) => p.id !== selectedPolicy.id && p.name === updatedName)
      ) {
        toast.error("策略名称已存在");
        return;
      }
      if (!isPreset && displayScopeIds.length === 0) {
        toast.error("请选择应用范围");
        return;
      }
      setPolicies(
        policies.map((p) =>
          p.id === selectedPolicy.id
            ? {
                ...p,
                name: updatedName,
                scopeIds: isPreset ? p.scopeIds : displayScopeIds,
                scopeNames: isPreset ? p.scopeNames : resolveGroupNames(displayScopeIds),
                billingMode: formBillingMode,
                instanceConfig: formInstanceConfig,
                systemDiskType: formDiskType,
                systemDiskSize: formDiskSize,
                assignPublicIp: formAssignPublicIp,
                bandwidthBillingMode: formAssignPublicIp ? formBandwidthBilling : "traffic",
                bandwidthLimit: formAssignPublicIp ? formBandwidthLimit : 0,
              }
            : p
        )
      );
      toast.success(`策略「${updatedName}」更新成功`);
    }
    setDialogOpen(false);
  };

  // ── 触发删除确认 ──
  const handleOpenDelete = (policy: ResourcePolicy) => {
    setPolicyToDelete(policy);
    setDeleteConfirmOpen(true);
  };

  // ── 执行删除 ──
  const handleConfirmDelete = () => {
    if (policyToDelete) {
      setPolicies(policies.filter((p) => p.id !== policyToDelete.id));
      toast.success(`已删除「${policyToDelete.name}」策略`);
    }
    setDeleteConfirmOpen(false);
  };

  const dialogTitle =
    dialogMode === "add"
      ? "新增分组策略"
      : `编辑策略「${selectedPolicy?.name ?? ""}」`;

  const isPresetEdit = dialogMode === "edit" && selectedPolicy?.type === "preset";

  return (
    <div className="page-enter">
      <AdminPageHeader
        title="资源配置"
        description="配置新建 Agent 实例使用的实例规格、计费模式、系统盘和公网资源策略。"
      />

      <Alert variant="operation-info" className="mb-6">
        <AlertOperationInfoIcon />
        <AlertDescription>
          <ul className="list-disc space-y-1 pl-4">
            <li>用户创建 Agent 实例时，系统会根据所选分组匹配资源策略；本分组无策略时采用最近的上级分组策略，上级均无策略时使用<strong className="font-semibold">「企业默认资源策略」</strong>。</li>
            <li>资源配置仅影响后续新建 Agent 实例，不影响已创建实例。</li>
          </ul>
        </AlertDescription>
      </Alert>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <SectionTitle>资源配置策略</SectionTitle>
          </div>
          <Button variant="claw-primary" size="claw-sm" onClick={handleOpenAdd}>
            <Plus className="w-3.5 h-3.5" />
            新增分组策略
          </Button>
        </div>

        <SurfaceCard className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[180px]">策略名称</TableHead>
                <TableHead className="w-[110px]">计费模式</TableHead>
                <TableHead className="w-[140px]">实例规格</TableHead>
                <TableHead className="w-[140px]">系统盘类型</TableHead>
                <TableHead className="w-[140px]">系统盘容量</TableHead>
                <TableHead className="w-[80px]">公网 IP</TableHead>
                <TableHead className="w-[120px]">公网计费</TableHead>
                <TableHead className="w-[100px]">带宽上限</TableHead>
                <TableHead className="w-[220px]">应用范围</TableHead>
                <TableHead className="w-[120px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pagedPolicies.map((policy) => (
                <TableRow key={policy.id}>
                  <TableCell className="font-medium">
                    <span
                      className="inline-flex items-center text-xs text-[var(--cp-text-body)]"
                      data-bubble-anchor={policy.type === "preset" ? "preset-policy" : undefined}
                    >
                      {policy.name}
                    </span>
                  </TableCell>
                  <TableCell>
                    {policy.billingMode === "subscription" ? (
                      <Badge color="blue">包年包月</Badge>
                    ) : (
                      <Badge color="purple">按量计费</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-[var(--cp-text-body)] tabular-nums">
                    {tableInstanceConfigDisplay(policy.instanceConfig)}
                  </TableCell>
                  <TableCell className="text-xs text-[var(--cp-text-body)]">
                    {policy.systemDiskType}
                  </TableCell>
                  <TableCell className="text-xs text-[var(--cp-text-body)] tabular-nums">
                    {policy.systemDiskSize}GiB
                  </TableCell>
                  <TableCell className="text-xs text-[var(--cp-text-body)]">
                    {policy.assignPublicIp ? "分配" : "不分配"}
                  </TableCell>
                  <TableCell className="text-xs text-[var(--cp-text-body)]">
                    {policy.assignPublicIp
                      ? policy.bandwidthBillingMode === "monthly"
                        ? "按带宽计费"
                        : "按流量计费"
                      : <span className="text-[var(--cp-text-weak)]">—</span>}
                  </TableCell>
                  <TableCell className="text-xs text-[var(--cp-text-body)] tabular-nums">
                    {policy.assignPublicIp ? `${policy.bandwidthLimit}Mbps` : <span className="text-[var(--cp-text-weak)]">—</span>}
                  </TableCell>
                  <TableCell>
                    {policy.type === "preset" ? (
                      <div className="inline-flex items-center gap-1.5">
                        <Badge variant="outline">预设策略</Badge>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Info className="size-3.5 text-[var(--cp-text-weak)]" />
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            企业默认资源配置，适用于未分配组织用户，以及未匹配到用户组资源配置的场景。
                          </TooltipContent>
                        </Tooltip>
                      </div>
                    ) : policy.scopeNames.length === 0 ? (
                      <span className="text-[var(--cp-text-weak)]">—</span>
                    ) : policy.scopeNames.length === 1 ? (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Badge variant="secondary" className="max-w-[140px]">
                            <span className="block truncate max-w-[124px]">{policy.scopeNames[0]}</span>
                          </Badge>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-[320px] text-xs leading-relaxed">
                          {getScopePath(policy.scopeIds[0], MOCK_GROUPS)}
                        </TooltipContent>
                      </Tooltip>
                    ) : (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="inline-flex max-w-full items-center gap-1 cursor-default">
                            <Badge variant="secondary" className="max-w-[140px]">
                              <span className="block truncate max-w-[124px]">{policy.scopeNames[0]}</span>
                            </Badge>
                            <Badge variant="secondary">+{policy.scopeNames.length - 1}</Badge>
                          </span>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-[320px] text-xs leading-relaxed whitespace-pre-line">
                          {policy.scopeIds.map((id) => getScopePath(id, MOCK_GROUPS)).join("\n")}
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableActionCell>
                    <Button
                      variant="link"
                      onClick={() => handleOpenEdit(policy)}
                      className="text-xs text-[var(--cp-text-brand)]"
                    >
                      编辑
                    </Button>
                    {policy.type !== "preset" && (
                      <Button
                        variant="link"
                        onClick={() => handleOpenDelete(policy)}
                        className="text-xs"
                      >
                        删除
                      </Button>
                    )}
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="px-4 py-2 border-t border-[var(--cp-border)]">
            <Pagination
              total={policies.length}
              current={page}
              pageSize={pageSize}
              showTotal={(total) => `共 ${total} 条记录`}
              onChange={(newPage, newPageSize) => {
                setPage(newPage);
                setPageSize(newPageSize);
              }}
              className="w-full justify-between"
              hideOnSinglePage
            />
          </div>
        </SurfaceCard>
      </section>

      {/* ─── 新增/编辑弹窗 ──────────────────────────────────────────── */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent size="md" onPointerDownOutside={(e) => e.preventDefault()} onInteractOutside={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>{dialogTitle}</DialogTitle>
            <DialogDescription>
              {dialogMode === "add"
                ? "创建分组级别的资源配置模板，分配给指定分组后自动应用于该分组下新建的 Agent 实例。"
                : "修改后仅影响后续命中该策略新建的 Agent 实例，已创建实例不受影响。"}
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6 space-y-4 py-4 max-h-[70vh]">
            {/* 1. 策略名称（编辑企业默认资源策略时不展示） */}
            {!isPresetEdit && (
            <div className="space-y-2">
              <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                策略名称
              </label>
                <Input
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  placeholder="请输入策略名称"
                  className="h-9 rounded-[var(--radius-lg)] text-sm"
                />
            </div>
            )}

            {/* 2. 计费模式 */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                计费模式
              </label>
              <RadioGroup
                value={formBillingMode}
                onValueChange={(v) => setFormBillingMode(v as "subscription" | "payg")}
                className="flex items-center gap-6 h-9"
              >
                <label className="flex items-center gap-2 cursor-pointer">
                  <RadioGroupItem value="subscription" />
                  <span className="text-sm text-[var(--cp-text-body)]">包年包月</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <RadioGroupItem value="payg" />
                  <span className="text-sm text-[var(--cp-text-body)]">按量计费</span>
                </label>
              </RadioGroup>

            </div>

            {/* 4. 实例规格 */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                实例规格
              </label>
              <Select value={formInstanceConfig} onValueChange={setFormInstanceConfig}>
                <SelectTrigger className="w-full h-9 rounded-[var(--radius-lg)] text-sm [&_.select-desc]:hidden">
                  <SelectValue>
                    {formInstanceConfig.includes("2核2GiB")
                      ? "2核2GiB"
                      : formInstanceConfig.includes("2核4GiB")
                        ? "2核4GiB"
                        : formInstanceConfig.includes("4核8GiB")
                          ? "4核8GiB"
                          : formInstanceConfig.includes("8核16GiB")
                            ? "8核16GiB"
                            : formInstanceConfig}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {INSTANCE_CONFIGS.map((spec) => (
                    <SelectItem key={spec.value} value={spec.value} className="text-sm">
                      <span className="text-sm text-[var(--cp-text-emphasis)] font-normal">{spec.label}</span>
                      <span className="text-xs text-[var(--cp-text-weak)] font-normal ml-2 select-desc">{spec.desc}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* 5. 系统盘类型 + 系统盘容量 */}
            <div className="grid grid-cols-2 gap-4 items-start">
              <div className="space-y-2">
                <div className="flex items-center gap-1">
                  <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                    系统盘类型
                  </label>
                </div>
                <Select value={formDiskType} onValueChange={setFormDiskType}>
                  <SelectTrigger className="w-full h-9 rounded-[var(--radius-lg)] text-sm">
                    <SelectValue placeholder="选择系统盘类型" />
                  </SelectTrigger>
                  <SelectContent>
                    {DISK_TYPES.map((t) => (
                      <SelectItem key={t} value={t} className="text-sm">
                        {t}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-1">
                  <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                    系统盘容量
                  </label>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Info className="h-4 w-4 text-[var(--cp-text-weak)]" />
                    </TooltipTrigger>
                    <TooltipContent side="top">
                      系统盘容量为新建 Agent 实例的实际创建容量，若用户选择的镜像要求容量超过该值，将无法创建 Agent 实例。
                    </TooltipContent>
                  </Tooltip>
                </div>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="flex items-center gap-2 cursor-help h-9">
                      <Input
                        type="text"
                        inputMode="numeric"
                        value={formDiskSize === 0 ? "" : formDiskSize}
                        onKeyDown={(e) => {
                          const isCtrlOrMeta = e.ctrlKey || e.metaKey;
                          const allowedKeys = [
                            "Backspace", "Delete", "Tab", "Escape", "Enter",
                            "ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"
                          ];
                          if (!/[0-9]/.test(e.key) && !allowedKeys.includes(e.key) && !isCtrlOrMeta) {
                            e.preventDefault();
                          }
                        }}
                        onChange={(e) => {
                          let valStr = e.target.value.replace(/\D/g, ""); // 移除非数字字符
                          valStr = valStr.replace(/^0+/, ""); // 过滤前导 0
                          if (valStr === "") {
                            setFormDiskSize(""); // 设为空字符，支持临时退格清空
                            return;
                          }
                          const valNum = Number(valStr);
                          if (valNum > diskRange.maxSize) {
                            setFormDiskSize(diskRange.maxSize);
                          } else {
                            setFormDiskSize(valNum);
                          }
                        }}
                        onBlur={() => {
                          if (formDiskSize === "" || formDiskSize < diskRange.minSize) {
                            setFormDiskSize(diskRange.minSize);
                          }
                        }}
                        className="h-9 rounded-[var(--radius-lg)] text-sm w-24 text-center tabular-nums"
                        placeholder="容量"
                      />
                      <span className="text-sm text-[var(--cp-text-weak)]">GiB</span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    系统盘容量取值范围：{diskRange.minSize}GiB～{diskRange.maxSize}GiB
                  </TooltipContent>
                </Tooltip>
              </div>
            </div>

            {/* 6. 是否分配公网 IP */}
            <div className="space-y-2">
              <div className="flex items-center gap-1">
                <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                  是否分配公网 IP
                </label>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Info className="h-4 w-4 text-[var(--cp-text-weak)]" />
                  </TooltipTrigger>
                  <TooltipContent side="top" className="max-w-[320px] text-xs leading-relaxed">
                    云服务器需要外网访问能力的时候，需要为云服务器分配公网 IP，如果云服务器不分配公网 IP，则不支持外出流量，并且无法使用外网 IP 对外进行互相通信。
                  </TooltipContent>
                </Tooltip>
              </div>
              <RadioGroup
                value={formAssignPublicIp ? "true" : "false"}
                onValueChange={(v) => {
                  const assign = v === "true";
                  setFormAssignPublicIp(assign);
                  if (assign) {
                    setFormBandwidthBilling("monthly");
                    setFormBandwidthLimit(5);
                  }
                }}
                className="flex items-center gap-6 h-9"
              >
                <label className="flex items-center gap-2 cursor-pointer">
                  <RadioGroupItem value="true" />
                  <span className="text-sm text-[var(--cp-text-body)]">分配</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <RadioGroupItem value="false" />
                  <span className="text-sm text-[var(--cp-text-body)]">不分配</span>
                </label>
              </RadioGroup>
            </div>

            {/* 7. 公网计费模式 + 8. 带宽上限 */}
            {formAssignPublicIp && (
              <>
                <div className="space-y-2">
                  <div className="flex items-center gap-1">
                    <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                      公网计费模式
                    </label>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Info className="h-4 w-4 text-[var(--cp-text-weak)]" />
                      </TooltipTrigger>
                      <TooltipContent side="top" className="max-w-[360px] text-xs leading-relaxed">
                        <div className="space-y-2">
                          <p><strong className="font-semibold">包月带宽：</strong>包月的固定带宽是指定公网出方向的带宽的大小，选择单台服务器最大带宽値。按固定带宽值计费，费用与实际使用流量无关。适合流量消耗大、带宽利用率较高的业务场景。</p>
                          <p><strong className="font-semibold">按流量计费（推荐）：</strong>使用流量是指服务器使用过程中产生的流量大小，网络费用仅取决于云服务器的出流量。适合流量波动大、带宽利用率不高的业务场景，在 Agent 使用场景中较为推荐。</p>
                        </div>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                  <RadioGroup
                    value={formBandwidthBilling}
                    onValueChange={(v) => {
                      const newMode = v as "monthly" | "traffic";
                      setFormBandwidthBilling(newMode);
                      // 切换公网计费模式时，带宽上限也默认回到 5Mbps
                      setFormBandwidthLimit(5);
                    }}
                    className="flex items-center gap-6 h-9"
                  >
                    <label className="flex items-center gap-2 cursor-pointer">
                      <RadioGroupItem value="monthly" />
                      <span className="text-sm text-[var(--cp-text-body)]">包月带宽</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <RadioGroupItem value="traffic" />
                      <span className="text-sm text-[var(--cp-text-body)]">按流量计费</span>
                    </label>
                  </RadioGroup>
                </div>

                <div className="space-y-2">
                  <div className="flex items-center gap-1">
                    <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                      带宽上限
                    </label>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Info className="h-4 w-4 text-[var(--cp-text-weak)]" />
                      </TooltipTrigger>
                      <TooltipContent side="top" className="max-w-[320px] text-xs leading-relaxed">
                        单台云服务器可以运行到的最高带宽，超过这个带宽上限将默认丢包。不同的网络计费模式，支持的公网带宽上限有所不同。
                      </TooltipContent>
                    </Tooltip>
                  </div>
                  <div className="flex items-center gap-4">
                    {/* 左侧：输入框 + 单位 Mbps */}
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <div className="flex items-center gap-1.5 shrink-0 cursor-help">
                          <Input
                            type="number"
                            min={1}
                            max={getBandwidthMaxLimit(formBandwidthBilling)}
                            value={formBandwidthLimit || ""}
                            onChange={(e) => {
                              const raw = e.target.value;
                              if (raw === "") {
                                setFormBandwidthLimit(0);
                                return;
                              }
                              let v = Number(raw);
                              const maxLimit = getBandwidthMaxLimit(formBandwidthBilling);
                              if (v > maxLimit) v = maxLimit;
                              if (v < 1) v = 1;
                              setFormBandwidthLimit(v);
                            }}
                            className="h-9 rounded-[var(--radius-lg)] text-sm w-24 text-center tabular-nums"
                            placeholder="带宽"
                          />
                          <span className="text-sm text-[var(--cp-text-weak)]">Mbps</span>
                        </div>
                      </TooltipTrigger>
                      <TooltipContent side="top">
                        {formBandwidthBilling === "monthly"
                          ? "包月带宽模式最高支持 20Mbps"
                          : "按流量计费模式最高支持 200Mbps"}
                      </TooltipContent>
                    </Tooltip>

                    {/* 分割线 */}
                    <div className="h-4 w-[1px] bg-[var(--cp-border)] shrink-0" />

                    {/* 右侧：快捷值按钮（固定 1/2/5/10） */}
                    <div className="flex items-center gap-1.5">
                      {[1, 2, 5, 10].map((val) => {
                        const isActive = formBandwidthLimit === val;
                        return (
                          <button
                            key={val}
                            type="button"
                            onClick={() => setFormBandwidthLimit(val)}
                            className={`min-w-[48px] h-9 rounded-[var(--radius-lg)] border text-sm font-medium transition-all text-center flex items-center justify-center px-2 ${
                              isActive
                                ? "border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected-solid)] text-[var(--cp-brand-blue)]"
                                : "border-[var(--cp-border)] bg-[var(--cp-surface-primary)] text-[var(--cp-text-emphasis)] hover:border-[var(--cp-border)]"
                            }`}
                          >
                            {val}Mbps
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </div>
              </>
            )}

            {/* 应用范围（编辑企业默认资源策略时不展示） */}
            {!isPresetEdit && (
            <div className="space-y-2">
              <label className="text-sm font-medium text-[var(--cp-text-secondary)]">
                应用范围
              </label>
                <GroupSelect
                  groups={MOCK_GROUPS}
                  selectedIds={displayScopeIds}
                  onChange={(ids) => {
                    setDisplayScopeIds(normalizeScopeIds(ids, MOCK_GROUPS, usedGroupIds));
                  }}
                  disabledIds={Array.from(usedGroupIds)}
                  disabledTooltip="该分组已被其他策略占用"
                  sourceFilter={["manual"]}
                  placeholder="请选择分组"
                />
            </div>
            )}
          </DialogBody>
          <DialogFooter className="px-6 pt-2 pb-6">
            <Button variant="claw-outline" size="claw" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button variant="dialog-confirm" size="claw" onClick={handleSave}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── 删除确认弹窗 ──────────────────────────────────────────────── */}
      <AlertDialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
        <AlertDialogContent onPointerDownOutside={(e) => e.preventDefault()} onInteractOutside={(e) => e.preventDefault()}>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除该分组策略？</AlertDialogTitle>
            <AlertDialogDescription>
              删除后，该策略应用范围内用户后续新建 Agent 实例将不再使用此策略。已创建的 Agent 实例不受影响。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-[var(--cp-text-danger)] text-white hover:bg-[var(--cp-text-danger)]/90"
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {presetHintPos && (
        <div className="fixed z-[9990]" style={{ left: presetHintPos.left, top: presetHintPos.top, transform: "translateY(-50%)" }}>
          <GuidePointBubble
            open={showPresetHint}
            onClose={() => setShowPresetHint(false)}
            title="企业默认资源策略"
            description=""
            listItems={[
              "适用于未加入分组，或所属分组未配置资源策略的用户",
              "仅对新建 Agent 实例生效",
              "请确认实例规格、系统盘、公网及计费模式符合企业默认要求",
            ]}
            contentVariant="text-button"
            placement="right"
            showHotspot={false}
            actionText="我知道了"
            onAction={() => setShowPresetHint(false)}
            endpoint="admin"
          />
        </div>
      )}
    </div>
  );
}
