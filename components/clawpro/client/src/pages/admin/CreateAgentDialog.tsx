/**
 * CreateAgentDialog —— 管控端「创建 Agent」三步弹窗
 *
 * 设计：严格遵循 ClawPro 管控端（Admin）设计规范。
 *   - 步骤指示：标准 <Stepper> 组件（替代手写 StepDot）
 *   - 字段成组：Step1/Step2 完全对齐用户端「我的 Agent」(MyOpenClaw) 创建弹窗的字段规格 —— 外层 <div className="space-y-5">（20px 字段间距）；label 直接用 <label> / <span> + "text-sm font-medium text-[var(--text-emphasis)]"（14px medium emphasis，保证全弹窗四个 label 视觉权重完全一致）；Agent 类型 / 角色身份 使用 Collapsible 单行水平结构（label 靠左 + 「已选 xxx ▽」靠右，无外框），与用户端一致；Step3 内层小卡片仍用 <Field className="gap-1.5"> 组合与 <MetaMedium as="label" tone="secondary">
 *   - 卡片层级：<SurfaceInner>（卡片内子面板），禁止 div 拼卡片
 *   - 已配置行：<Item variant="outline" size="sm">（统一行卡语义）
 *   - 空状态：<Empty> 组件家族（替代手拼 border-dashed 文本）
 *   - 用户搜索选择：<SearchableSelect>（combobox.md §5 强制规范，禁止手拼 Popover+Input+ListBox）
 *   - 危险确认：<AlertDialog>（删除模型 / 通道前二次确认）
 *   - 图标按钮：<Button variant="ghost" size="icon-sm">（替代手拼 button）
 *   - 提示横幅：<Alert> 组件家族
 *   - Typography 语义组件
 *   - 图标一律 lucide-react，无 emoji / 手写 SVG
 *
 * 步骤设计（3 步）：
 *   Step 1 · 归属
 *     - 选用户（管理员搜索 userId / 名称，命中后确认）
 *     - 【条件】仅当用户属于 ≥ 2 个分组时才出现「所属分组」
 *   Step 2 · 基础信息
 *     - Agent 名称 / Agent 类型 / 角色身份
 *   Step 3 · 配置（选填）
 *     - 模型 / 通道 / 技能，按平台策略与所选分组可见范围过滤
 */
import React, { useEffect, useMemo, useState } from "react";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
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
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SearchableSelect,
} from "@/components/ui/select";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { SurfaceInner } from "@/components/ui/Surface";
import { Separator } from "@/components/ui/separator";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import {
  BodyMedium,
  MetaMedium,
  MetaText,
  MiniBodyText,
  HelperText,
} from "@/components/ui/Typography";
import {
  ChevronDown,
  Plus,
  Trash2,
  Search,
  Eye,
  EyeOff,
  ArrowLeftRight,
  AlertCircle,
  CheckCircle2,
} from "lucide-react";
import { StatusTag } from "@/components/ui/status-tag";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Stepper } from "@/components/ui/stepper";
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Field,
  FieldGroup,
} from "@/components/ui/field";
import {
  Item,
  ItemContent,
  ItemActions,
} from "@/components/ui/item";
import { toast } from "sonner";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { MOCK_ROLES, AVAILABLE_SKILLS, type Role } from "@/lib/mockData";
import { useAdminModels, CUSTOM_PROVIDER_VALUE, type ModelRow } from "@/lib/modelConfigStore";
import { CHANNEL_OPTIONS, type ChannelConfig } from "@/lib/agentConfigConstants";
import {
  loadBuiltinChannelVisibility,
  loadVisibleCustomChannels,
  onBuiltinChannelVisibilityChange,
  onCustomChannelsChange,
  type CustomChannel as AdminCustomChannel,
} from "@/lib/customChannelStore";
import {
  useUsers,
  useUserById,
  useUserGroupItems,
  useGroupsMap,
  USER_GROUP_KIND_LABEL,
  type UserGroupItem,
} from "@/lib/useUsersAndGroups";
import { groupStore } from "./MemberManagement/groupStore";

// ─────────────────────────────────────────────────────────────────────────────
// 类型定义
// ─────────────────────────────────────────────────────────────────────────────

type AgentTypeValue = "openclaw" | "hermes" | "lightclawace";

const AGENT_TYPE_OPTIONS: { value: AgentTypeValue; label: string }[] = [
  { value: "openclaw", label: "OpenClaw" },
  { value: "hermes", label: "Hermes Agent" },
  { value: "lightclawace", label: "Lightclaw ACE" },
];

const STEPS = [
  { label: "归属" },
  { label: "基础信息" },
  { label: "配置（选填）" },
];

/** 已添加的一条模型（复用管控端 Agent 详情弹窗 AppliedModel 的语义，支持主/备选） */
interface DraftModel {
  /** 该条记录的唯一 id（用于本地增删/切主） */
  id: string;
  /** 对应「模型配置」页中的记录 id */
  modelConfigId: string;
  providerLabel: string;
  versionLabel: string;
  /** 是否主模型：仅在 agentType 支持主备时区分 */
  primary: boolean;
  /** 添加时间戳，用于备选组按后加先出（desc）排序 */
  addedAt: number;
}

/** 已添加的一条通道 */
interface DraftChannel {
  value: string;
  label: string;
  fieldValues: Record<string, string>;
}

interface ProviderGroup {
  key: string;
  label: string;
  models: ModelRow[];
  isCustom: boolean;
}

export interface CreateAgentResult {
  userId: string;
  groupId: string;
  groupName: string;
  name: string;
  agentType: AgentTypeValue;
  roleName: string;
  projectIds: string[];
  projectNames: string[];
  models: DraftModel[];
  channels: DraftChannel[];
  skills: string[];
}

interface CreateAgentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 确认创建回调（拿到草稿数据，由父组件落地为列表项 / 调接口） */
  onCreated?: (result: CreateAgentResult) => void;
}

// ─────────────────────────────────────────────────────────────────────────────
// 局部辅助组件
// ─────────────────────────────────────────────────────────────────────────────

/** 必填星号 —— 走 token，aria-label 保证可访问性 */
function RequiredMark() {
  return (
    <span aria-label="必填" className="text-[var(--text-danger)]">
      *
    </span>
  );
}

/**
 * 胶囊单选（与用户端「创建 Agent」同款交互，使用 token）
 * 胶囊圆角为 full 属于 §2.4 允许项（标签胶囊）
 */
function PillRadioOption({
  value,
  id,
  children,
}: {
  value: string;
  id: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center">
      <RadioGroupItem value={value} id={id} className="peer sr-only" />
      <Label
        htmlFor={id}
        className="flex items-center justify-center whitespace-nowrap h-6 rounded-full border border-[var(--border)] bg-[var(--card)] px-3 text-xs font-medium text-[var(--text-emphasis)] hover:bg-[var(--accent)] hover:border-[var(--border-control)] cursor-pointer peer-data-[state=checked]:bg-[var(--text-emphasis)] peer-data-[state=checked]:text-white peer-data-[state=checked]:border-[var(--text-emphasis)] peer-data-[state=checked]:hover:bg-[var(--text-emphasis)] peer-data-[state=checked]:hover:text-white transition-colors"
      >
        {children}
      </Label>
    </div>
  );
}

/** Step3 三段（模型 / 通道 / 技能）的段落标题 + 右侧操作 */
function SectionHeader({
  title,
  count,
  action,
}: {
  title: string;
  count: number;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <MetaText as="div">
        {title}（{count}）
      </MetaText>
      {action}
    </div>
  );
}

/** 紧凑行内空态：Empty 的紧凑变体（不带插画，适合弹窗内行级空态） */
function EmptyHint({ title }: { title: string }) {
  return (
    <Empty className="border border-[var(--border)] p-4 md:p-4">
      <EmptyHeader>
        <EmptyTitle>{title}</EmptyTitle>
      </EmptyHeader>
    </Empty>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// 主组件
// ─────────────────────────────────────────────────────────────────────────────
export default function CreateAgentDialog({ open, onOpenChange, onCreated }: CreateAgentDialogProps) {
  const [step, setStep] = useState<1 | 2 | 3>(1);

  // ── Step1：归属（用户 + 分组） ──────────────────────────────────────────
  //   归属用户：ClawPro 规范 `combobox.md` §5 明确禁止「手拼 Popover + Input + ListBox」，
  //   一律走 <SearchableSelect>（`@/components/ui/select`）。
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);

  // ── Step2：基础信息 ─────────────────────────────────────────────────────
  const [name, setName] = useState("");
  const [agentType, setAgentType] = useState<AgentTypeValue>("openclaw");
  const [agentTypeOpen, setAgentTypeOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [roleOpen, setRoleOpen] = useState(false);

  // ── 项目（可多选，默认不选） ──────────────────────────────────────────────
  // 项目池来自共享 groupStore（「项目资产管理」创建的项目实时可见）；
  // 关联后，创建的实例会额外安装所选项目配置的特定技能、规范等。
  const [projectPool, setProjectPool] = useState<{ id: string; name: string }[]>(
    () => groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null).map((g) => ({ id: g.id, name: g.name })),
  );
  useEffect(
    () => groupStore.subscribe(() =>
      setProjectPool(
        groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null).map((g) => ({ id: g.id, name: g.name })),
      ),
    ),
    [],
  );
  const [selectedProjectIds, setSelectedProjectIds] = useState<string[]>([]);
  const [projectOpen, setProjectOpen] = useState(false);
  const toggleProject = (id: string) =>
    setSelectedProjectIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const visibleRoles = useMemo(() => MOCK_ROLES.filter((r) => r.visible), []);

  // 用户/组织统一从 useUsersAndGroups 拿，跟随 useAdminMode 切换（OneID/普通模式），
  // 并订阅 MemberManagement 的编辑事件（notifyUsersChanged）自动刷新。
  const groupsMap = useGroupsMap();
  const groupNameById = useMemo(() => {
    const m = new Map<string, string>();
    groupsMap.forEach((g, id) => m.set(id, g.name));
    return m;
  }, [groupsMap]);

  // 全量用户列表（供 SearchableSelect 自身完成过滤 / 搜索）
  const allUsers = useUsers();
  const userOptions = useMemo(
    () => allUsers.map((u) => ({ value: u.userId, label: u.userId })),
    [allUsers]
  );
  const selectedUser = useUserById(selectedUserId);

  // 当前所选用户所属的分组：**带语义类别**（主部门 / 兼任部门 / 用户组 / 组织）
  const selectedUserGroups: UserGroupItem[] = useUserGroupItems(selectedUser);

  // 有效的分组数（把 primary 但 invalid 的主部门排除在可选之外）
  const validGroupCount = selectedUserGroups.filter((g) => !g.invalid).length;
  // 是否需要出现「所属分组」字段：仅 ≥2 个可选分组时才让管理员选择
  const needGroupPick = validGroupCount >= 2;

  // 按 kind 分组，便于按主部门 / 兼任 / 用户组 / 组织 分段展示
  const groupedByKind = useMemo(() => {
    const buckets: Record<UserGroupItem["kind"], UserGroupItem[]> = {
      primary: [], secondary: [], oneidGroup: [], manual: [],
    };
    selectedUserGroups.forEach((it) => buckets[it.kind].push(it));
    return buckets;
  }, [selectedUserGroups]);

  // 选择 / 清空用户（SearchableSelect 传空串代表清除）
  const handleUserChange = (value: string) => {
    setSelectedUserId(value || null);
    if (!value) setSelectedGroupId(null);
  };

  // 用户切换后同步"所属分组"的默认值：
  //   - 只有 1 个有效分组 → 静默默认选中
  //   - ≥2 个有效分组 → 保持为空，等管理员主动挑
  //   - 0 个有效分组 → 保持为空（禁止提交，Alert 会给出提示）
  // 若已选的 selectedGroupId 因为用户组织变动或用户切换而不再有效，也在这里清掉。
  useEffect(() => {
    if (!selectedUser) {
      if (selectedGroupId !== null) setSelectedGroupId(null);
      return;
    }
    const validIds = selectedUserGroups.filter((g) => !g.invalid).map((g) => g.id);
    if (selectedGroupId && !validIds.includes(selectedGroupId)) {
      setSelectedGroupId(null);
      return;
    }
    if (!selectedGroupId && validIds.length === 1) {
      setSelectedGroupId(validIds[0]);
    }
    // 依赖：只在 selectedUser / selectedUserGroups 变化时才重新决策
    // selectedGroupId 变化不需要触发，否则会与用户主动选择反复覆盖
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedUser, selectedUserGroups]);

  // ── 平台策略（决定 Step3 显示哪些配置） ─────────────────────────────────
  const [allowConfigModel, setAllowConfigModel] = useState(() => {
    const v = localStorage.getItem("admin_allow_user_config_model");
    return v !== null ? v === "true" : true;
  });
  const [allowConfigChannel, setAllowConfigChannel] = useState(() => {
    const v = localStorage.getItem("admin_allow_user_config_channel");
    return v !== null ? v === "true" : true;
  });
  useEffect(() => {
    const handler = (e: StorageEvent) => {
      if (e.key === "admin_allow_user_config_model") {
        setAllowConfigModel(e.newValue !== null ? e.newValue === "true" : true);
      }
      if (e.key === "admin_allow_user_config_channel") {
        setAllowConfigChannel(e.newValue !== null ? e.newValue === "true" : true);
      }
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }, []);

  // ── 模型数据（按所选分组的可见范围过滤） ────────────────────────────────
  const adminModels = useAdminModels();
  const providerGroups = useMemo<ProviderGroup[]>(() => {
    const visible = adminModels.filter((m) => {
      if (!m.visible) return false;
      if (m.visibilityScope === "all") return true;
      return !!selectedGroupId && m.visibilityGroupIds.includes(selectedGroupId);
    });
    const orderedKeys: string[] = [];
    const buckets = new Map<string, ModelRow[]>();
    for (const m of visible) {
      if (!buckets.has(m.provider)) {
        buckets.set(m.provider, []);
        orderedKeys.push(m.provider);
      }
      buckets.get(m.provider)!.push(m);
    }
    const groups: ProviderGroup[] = [];
    let customGroup: ProviderGroup | null = null;
    for (const key of orderedKeys) {
      const models = buckets.get(key)!;
      if (key === CUSTOM_PROVIDER_VALUE) {
        customGroup = { key, label: "自定义模型", models, isCustom: true };
      } else {
        groups.push({ key, label: models[0].name, models, isCustom: false });
      }
    }
    if (customGroup) groups.push(customGroup);
    return groups;
  }, [adminModels, selectedGroupId]);

  // ── 通道数据 ────────────────────────────────────────────────────────────
  const [builtinChannelVisibility, setBuiltinChannelVisibility] = useState<Record<string, boolean>>(
    () => loadBuiltinChannelVisibility(),
  );
  const [visibleCustomChannels, setVisibleCustomChannels] = useState<AdminCustomChannel[]>(
    () => loadVisibleCustomChannels(),
  );
  useEffect(() => onBuiltinChannelVisibilityChange(() => setBuiltinChannelVisibility(loadBuiltinChannelVisibility())), []);
  useEffect(() => onCustomChannelsChange(() => setVisibleCustomChannels(loadVisibleCustomChannels())), []);

  const availableChannelOptions = useMemo<ChannelConfig[]>(() => {
    const builtins = CHANNEL_OPTIONS.filter((ch) => {
      const key = ch.builtinId ?? ch.value;
      return builtinChannelVisibility[key] !== false;
    });
    const customs: ChannelConfig[] = visibleCustomChannels.map((cc) => ({
      value: `admin_custom_${cc.id}`,
      label: cc.name,
      descText: `企业自定义通道（Channel ID: ${cc.channelId}）`,
      detailUrl: "#",
      adminCustomMode: true as const,
      adminCustomId: cc.id,
      fields: cc.credentialFields.map((f) => ({ key: f.key, label: f.label, secret: false })),
    }));
    return [...builtins, ...customs];
  }, [builtinChannelVisibility, visibleCustomChannels]);

  // ── Step3：模型草稿 ─────────────────────────────────────────────────────
  const [models, setModels] = useState<DraftModel[]>([]);
  const [modelAdding, setModelAdding] = useState(false);
  const [modelDraftProvider, setModelDraftProvider] = useState("");
  const [modelDraftModelId, setModelDraftModelId] = useState("");
  // 删除确认（受控 AlertDialog，避免每行各渲染一个）
  const [modelDeleteTarget, setModelDeleteTarget] = useState<DraftModel | null>(null);

  const startAddModel = () => {
    const g0 = providerGroups[0];
    setModelDraftProvider(g0?.key ?? "");
    setModelDraftModelId(g0?.models[0]?.id ?? "");
    setModelAdding(true);
  };
  const handleDraftProviderChange = (value: string) => {
    setModelDraftProvider(value);
    const g = providerGroups.find((x) => x.key === value);
    setModelDraftModelId(g?.models[0]?.id ?? "");
  };
  const confirmAddModel = () => {
    const group = providerGroups.find((g) => g.key === modelDraftProvider);
    const model = group?.models.find((m) => m.id === modelDraftModelId);
    if (!group || !model) {
      toast.error("请选择有效的模型厂商和模型名称");
      return;
    }
    if (models.some((m) => m.modelConfigId === model.id)) {
      toast.error("该模型已在列表中，请勿重复添加");
      return;
    }
    // 复用详情弹窗规则：仅 OpenClaw 支持主 + 备选；其它类型单模型无主备语义（默认 primary=true）
    const supportsPrimary = agentType === "openclaw";
    const hasPrimary = models.some((m) => m.primary);
    setModels((prev) => [
      ...prev,
      {
        id: `draft-model-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        modelConfigId: model.id,
        providerLabel: group.label,
        versionLabel: group.isCustom ? model.name : model.version,
        // OpenClaw：无主则设为主，否则设为备选；非 OpenClaw：始终 true（不展示标签）
        primary: supportsPrimary ? !hasPrimary : true,
        addedAt: Date.now(),
      },
    ]);
    setModelAdding(false);
  };
  const removeModel = (id: string) =>
    setModels((prev) => prev.filter((m) => m.id !== id));
  /** 将某条备选模型设为主：原主变为备选（互换 primary 标记） */
  const setPrimaryModel = (id: string) =>
    setModels((prev) => prev.map((m) => ({ ...m, primary: m.id === id })));

  // 切换 Agent 类型时降级：非 OpenClaw 类型不支持主备，只保留主模型（若无主则保留最早添加的一条并置为 primary）
  useEffect(() => {
    if (agentType === "openclaw") return;
    setModels((prev) => {
      if (prev.length <= 1) {
        return prev.map((m) => ({ ...m, primary: true }));
      }
      const primary = prev.find((m) => m.primary) ?? prev[0];
      return [{ ...primary, primary: true }];
    });
  }, [agentType]);

  // ── Step3：通道草稿 ─────────────────────────────────────────────────────
  const [channels, setChannels] = useState<DraftChannel[]>([]);
  const [channelAdding, setChannelAdding] = useState(false);
  const [channelDraft, setChannelDraft] = useState("");
  const [channelDraftFields, setChannelDraftFields] = useState<Record<string, string>>({});
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set());
  const [channelDeleteTarget, setChannelDeleteTarget] = useState<DraftChannel | null>(null);

  const toggleSecret = (key: string) =>
    setVisibleSecrets((prev) => {
      const next = new Set(prev);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });

  const startAddChannel = () => {
    setChannelDraft("");
    setChannelDraftFields({});
    setChannelAdding(true);
  };
  const handleChannelDraftChange = (value: string) => {
    setChannelDraft(value);
    setChannelDraftFields({});
  };
  const confirmAddChannel = () => {
    const cfg = availableChannelOptions.find((c) => c.value === channelDraft);
    if (!cfg) {
      toast.error("请选择要添加的通道");
      return;
    }
    if (channels.some((c) => c.value === cfg.value)) {
      toast.error("该通道已添加，请勿重复添加");
      return;
    }
    setChannels((prev) => [...prev, { value: cfg.value, label: cfg.label, fieldValues: { ...channelDraftFields } }]);
    setChannelAdding(false);
  };
  const removeChannel = (value: string) =>
    setChannels((prev) => prev.filter((c) => c.value !== value));

  // ── Step3：技能草稿（单弹窗：选择即安装） ──────────────────────────────
  const [skills, setSkills] = useState<string[]>([]);
  const [skillPickerOpen, setSkillPickerOpen] = useState(false);
  const [skillSearch, setSkillSearch] = useState("");
  const [skillChecked, setSkillChecked] = useState<string[]>([]);

  /**
   * 技能候选：不预先罗列全部技能，仅在用户输入搜索词后按模糊匹配返回结果。
   * - 空搜索：返回 []（UI 层展示"请输入关键词搜索"占位）
   * - 有搜索词：在未安装的技能里做子串模糊匹配
   */
  const skillCandidates = useMemo(() => {
    const q = skillSearch.trim().toLowerCase();
    if (!q) return [];
    return AVAILABLE_SKILLS.filter((s) => !skills.includes(s)).filter((s) =>
      s.toLowerCase().includes(q),
    );
  }, [skillSearch, skills]);

  const openSkillPicker = () => {
    setSkillSearch("");
    setSkillChecked([]);
    setSkillPickerOpen(true);
  };
  const toggleSkillCheck = (s: string) =>
    setSkillChecked((prev) => (prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]));
  /** 直接安装所选技能，不再有二次确认弹窗 */
  const confirmInstallSkills = () => {
    if (skillChecked.length === 0) {
      toast.error("请先选择要安装的技能");
      return;
    }
    const count = skillChecked.length;
    setSkills((prev) => [...prev, ...skillChecked]);
    setSkillPickerOpen(false);
    setSkillChecked([]);
    toast.success(`已添加 ${count} 个技能`);
  };
  // 暂未提供技能删除能力，故不再提供 removeSkill

  // ── 校验 ────────────────────────────────────────────────────────────────
  /**
   * Step1 归属校验：必须选中用户；分组是"有则用之"：
   *   - 用户有 ≥2 个分组：必须显式选一个分组
   *   - 用户有 1 个分组：默认选中，条件自动满足
   *   - 用户有 0 个分组：允许继续（Agent 归属到用户名下，分组留空）
   */
  const step1Valid =
    !!selectedUserId && (validGroupCount === 0 || !!selectedGroupId);
  /** Step2 基础信息校验：Agent 名称必填（类型 / 角色都有默认值） */
  const step2Valid = name.trim().length > 0;

  const resetAll = () => {
    setStep(1);
    setSelectedUserId(null);
    setSelectedGroupId(null);
    setName("");
    setAgentType("openclaw");
    setAgentTypeOpen(false);
    setSelectedRole(null);
    setRoleOpen(false);
    setSelectedProjectIds([]);
    setProjectOpen(false);
    setModels([]);
    setModelAdding(false);
    setModelDeleteTarget(null);
    setChannels([]);
    setChannelAdding(false);
    setChannelDeleteTarget(null);
    setSkills([]);
    setSkillPickerOpen(false);
    setSkillChecked([]);
    setSkillSearch("");
  };

  const handleClose = (next: boolean) => {
    onOpenChange(next);
    if (!next) resetAll();
  };

  const handleConfirmCreate = () => {
    if (!step1Valid || !selectedUserId) {
      setStep(1);
      toast.error("请先完善「归属」信息");
      return;
    }
    if (!step2Valid) {
      setStep(2);
      toast.error("请填写 Agent 名称");
      return;
    }
    // 允许分组为空：当用户没有任何分组时，Agent 仅归属到该用户，不绑定分组配置
    const result: CreateAgentResult = {
      userId: selectedUserId,
      groupId: selectedGroupId ?? "",
      groupName: selectedGroupId
        ? groupNameById.get(selectedGroupId) ?? selectedGroupId
        : "",
      name: name.trim(),
      agentType,
      roleName: selectedRole?.name ?? "通用助手",
      projectIds: selectedProjectIds,
      projectNames: projectPool.filter((p) => selectedProjectIds.includes(p.id)).map((p) => p.name),
      models,
      channels,
      skills,
    };
    onCreated?.(result);
    toast.success(`Agent「${result.name}」已创建`);
    handleClose(false);
  };

  // ───────────────────────────────────────────────────────────────────────
  // 渲染
  // ───────────────────────────────────────────────────────────────────────
  const channelDraftCfg = availableChannelOptions.find((c) => c.value === channelDraft);
  const channelDraftFieldsList = channelDraftCfg?.fields ?? [];
  const channelIsWechatLike = !!channelDraftCfg?.wechatMode;

  return (
    <>
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>创建 Agent</DialogTitle>
            <DialogDescription className="sr-only">
              新建的 Agent 将归属到您在此选择的用户名下，用户可以在其用户端使用与管理该 Agent。
            </DialogDescription>
          </DialogHeader>

          {/* 顶部提示条：带背景色的 info Alert，替代纯文字 description */}
          <Alert variant="info" className="mb-3">
            <AlertInfoIcon />
            <AlertDescription>
              新建的 Agent 将归属到您在此选择的用户名下，用户可以在其用户端使用与管理该 Agent。
            </AlertDescription>
          </Alert>

          {/* 步骤指示器：标准 Stepper 组件（替代手写 StepDot） */}
          <Stepper current={step} steps={STEPS} />

          <DialogBody className="px-6 max-h-[60vh]">
            {step === 1 && (
              <div className="space-y-5 pt-3 pb-1">
                {/* 选择用户 */}
                <div className="space-y-2">
                  <span className="flex items-center gap-1 text-sm font-medium text-[var(--text-emphasis)]">
                    归属用户 <RequiredMark />
                  </span>
                  <SearchableSelect
                    options={userOptions}
                    value={selectedUserId ?? ""}
                    onChange={handleUserChange}
                    placeholder="请选择用户"
                    searchPlaceholder="输入用户 ID 搜索..."
                    align="start"
                    clearable
                  />
                </div>

                {/* 所属分组：按语义分段展示（主部门 / 兼任部门 / 用户组 / 组织） */}
                {selectedUser && needGroupPick && (
                  <>
                    <Separator />
                    <div className="space-y-2">
                      <span className="flex items-center gap-1 text-sm font-medium text-[var(--text-emphasis)]">
                        所属分组 <RequiredMark />
                      </span>
                      <MetaText as="p" className="leading-relaxed">
                        该用户在多个分组中，不同分组可能使用不同的模型 / 通道 / 技能配置，请为本 Agent 选择要使用哪个分组的配置。
                      </MetaText>
                      <RadioGroup
                        value={selectedGroupId ?? ""}
                        onValueChange={setSelectedGroupId}
                        className="flex flex-col gap-3 pt-1"
                      >
                        {(["primary", "secondary", "oneidGroup", "manual"] as const).map((kind) => {
                          const list = groupedByKind[kind];
                          if (list.length === 0) return null;
                          return (
                            <div key={kind} className="flex flex-col gap-2">
                              <MetaText tone="weak" className="text-[11px]">
                                {USER_GROUP_KIND_LABEL[kind]}
                              </MetaText>
                              <div className="flex flex-wrap gap-2">
                                {list.map((g) => (
                                  g.invalid ? (
                                    <Tooltip key={g.id}>
                                      <TooltipTrigger asChild>
                                        <span>
                                          <StatusTag
                                            mode="soft"
                                            variant="gray"
                                            className="cursor-not-allowed opacity-60"
                                            icon={<AlertCircle className="w-3 h-3" />}
                                          >
                                            {g.name}（已失效）
                                          </StatusTag>
                                        </span>
                                      </TooltipTrigger>
                                      <TooltipContent side="top" className="text-xs">
                                        该主部门在 OneID 侧已被删除，不可用于新建 Agent
                                      </TooltipContent>
                                    </Tooltip>
                                  ) : (
                                    <PillRadioOption key={g.id} value={g.id} id={`grp-${g.id}`}>
                                      {g.name}
                                    </PillRadioOption>
                                  )
                                ))}
                              </div>
                            </div>
                          );
                        })}
                      </RadioGroup>
                    </div>
                  </>
                )}

              </div>
            )}

            {step === 2 && (
              <div className="space-y-5 pt-3 pb-1">
                {/* Agent 名称 */}
                <div className="space-y-2">
                  <label
                    htmlFor="create-agent-name"
                    className="flex items-center gap-1 text-sm font-medium text-[var(--text-emphasis)]"
                  >
                    Agent 名称 <RequiredMark />
                  </label>
                  <Input
                    id="create-agent-name"
                    placeholder="请输入 Agent 名称"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </div>

                {/* Agent 类型 —— 完全对齐用户端「我的 Agent」创建弹窗：单行 label + 已选 xxx ▽，无外框 */}
                <Collapsible open={agentTypeOpen} onOpenChange={setAgentTypeOpen}>
                  <CollapsibleTrigger className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none">
                    <span className="text-sm font-medium text-[var(--text-emphasis)]">Agent 类型</span>
                    <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                      <span className="text-xs text-[var(--text-weak)]">已选</span>
                      {AGENT_TYPE_OPTIONS.find((o) => o.value === agentType)?.label}
                      <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                    </span>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="overflow-hidden">
                    <div className="pt-2 pb-1">
                      <RadioGroup
                        value={agentType}
                        onValueChange={(value) => {
                          setAgentType(value as AgentTypeValue);
                          if (value !== "openclaw") setSelectedRole(null);
                        }}
                        className="flex flex-wrap gap-2"
                      >
                        {AGENT_TYPE_OPTIONS.map((o) => (
                          <PillRadioOption key={o.value} value={o.value} id={`agent-type-${o.value}`}>
                            {o.label}
                          </PillRadioOption>
                        ))}
                      </RadioGroup>
                    </div>
                  </CollapsibleContent>
                </Collapsible>

                {/* 角色身份 —— 同上，用户端同款单行结构 */}
                <Collapsible open={roleOpen} onOpenChange={setRoleOpen}>
                  <CollapsibleTrigger className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none">
                    <span className="text-sm font-medium text-[var(--text-emphasis)]">角色身份</span>
                    <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                      <span className="text-xs text-[var(--text-weak)]">已选</span>
                      {selectedRole?.name ?? "通用助手"}
                      <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                    </span>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="overflow-hidden">
                    <div className="pt-2 pb-1 flex flex-col gap-3">
                      <RadioGroup
                        value={selectedRole?.id ?? "__general__"}
                        onValueChange={(value) => {
                          if (value === "__general__") {
                            setSelectedRole(null);
                            return;
                          }
                          setSelectedRole(visibleRoles.find((r) => r.id === value) ?? null);
                        }}
                        className="flex flex-wrap gap-2"
                      >
                        <PillRadioOption value="__general__" id="role-general">通用助手</PillRadioOption>
                        {visibleRoles.map((role) => (
                          <PillRadioOption key={role.id} value={role.id} id={`role-${role.id}`}>
                            {role.name}
                          </PillRadioOption>
                        ))}
                      </RadioGroup>

                      {(() => {
                        const display = selectedRole
                          ? {
                              name: selectedRole.name,
                              skills: selectedRole.skills.map((s) => s.name).join("、"),
                              soul: selectedRole.soul,
                            }
                          : {
                              name: "通用助手",
                              skills: "web-search、file-reader、code-runner",
                              soul: "无固定行业偏好的通用 AI 伙伴，擅长日常问答、信息检索与轻量创作，按需切换专业度",
                            };
                        return (
                          <SurfaceInner className="overflow-hidden bg-[var(--muted)]">
                            <div className="p-4 flex flex-col gap-3">
                              <div className="flex items-center gap-2">
                                <AgentAvatar roleName={display.name} size={28} />
                                <BodyMedium>{display.name}角色介绍</BodyMedium>
                              </div>
                              <Separator />
                              <Field className="gap-1.5">
                                <MetaMedium>角色技能</MetaMedium>
                                <MetaText as="p" tone="secondary" className="leading-relaxed">
                                  {display.skills}
                                </MetaText>
                              </Field>
                              <Field className="gap-1.5">
                                <MetaMedium>角色风格</MetaMedium>
                                <MetaText as="p" tone="secondary" className="leading-relaxed">
                                  {display.soul}
                                </MetaText>
                              </Field>
                            </div>
                          </SurfaceInner>
                        );
                      })()}
                    </div>
                  </CollapsibleContent>
                </Collapsible>

                {/* 项目（可多选，默认不选） */}
                <Collapsible open={projectOpen} onOpenChange={setProjectOpen}>
                  <CollapsibleTrigger className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none">
                    <span className="text-sm font-medium text-[var(--text-emphasis)]">项目</span>
                    <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                      <span className="text-xs text-[var(--text-weak)]">已选</span>
                      {selectedProjectIds.length === 0 ? "不关联项目" : `${selectedProjectIds.length} 个项目`}
                      <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                    </span>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="overflow-hidden">
                    <div className="pl-0 pr-1.5 pt-2.5 pb-1 space-y-2">
                      <HelperText as="p">
                        可选，关联项目后，会额外安装所选项目配置的特定技能、规范等。
                      </HelperText>
                      {projectPool.length === 0 ? (
                        <MetaText as="p" tone="weak">暂无可关联的项目</MetaText>
                      ) : (
                        <div className="flex flex-wrap gap-2">
                          {projectPool.map((p) => {
                            const active = selectedProjectIds.includes(p.id);
                            return (
                              <button
                                key={p.id}
                                type="button"
                                onClick={() => toggleProject(p.id)}
                                className={`relative flex items-center justify-center whitespace-nowrap h-6 px-3 rounded-full border text-xs font-medium transition-colors cursor-pointer ${
                                  active
                                    ? "bg-[var(--text-emphasis)] text-white border-[var(--text-emphasis)]"
                                    : "border-[var(--border)] bg-[var(--card)] text-[var(--text-emphasis)] hover:bg-[var(--accent)] hover:border-[var(--border-control)]"
                                }`}
                              >
                                {p.name}
                                {active && (
                                  <CheckCircle2 className="absolute -top-1.5 -right-1.5 w-3.5 h-3.5 fill-white text-[var(--text-emphasis)]" />
                                )}
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  </CollapsibleContent>
                </Collapsible>
              </div>
            )}

            {step === 3 && (
              <FieldGroup className="gap-5 pt-3 pb-1">
                {/* 模型配置 —— 复用管控端 Agent 详情「已应用模型」的主/备选语义 */}
                {allowConfigModel && (() => {
                  const supportsPrimary = agentType === "openclaw";
                  const hasPrimary = models.some((m) => m.primary);
                  const primaryList = models.filter((m) => m.primary);
                  const backupList = [...models.filter((m) => !m.primary)].sort((a, b) => b.addedAt - a.addedAt);
                  // 非 OpenClaw 只允许单模型：已有模型后隐藏"添加"按钮
                  const canAddMore = supportsPrimary || models.length === 0;
                  const addButtonLabel = supportsPrimary
                    ? (hasPrimary ? "添加备选模型" : "添加主模型")
                    : "添加模型";

                  const renderModelRow = (m: DraftModel, isPrimary: boolean) => (
                    <Item key={m.id} variant="outline" size="sm">
                      <ItemContent>
                        <BodyMedium className="truncate leading-tight">{m.providerLabel}</BodyMedium>
                        {m.versionLabel && (
                          <MetaText tone="weak" className="truncate leading-tight">
                            {m.versionLabel}
                          </MetaText>
                        )}
                      </ItemContent>
                      {supportsPrimary && (
                        isPrimary
                          ? <StatusTag mode="fill" variant="green">主模型</StatusTag>
                          : <StatusTag mode="fill" variant="gray">备选模型</StatusTag>
                      )}
                      <ItemActions>
                        {supportsPrimary && !isPrimary && (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                aria-label="设为主模型"
                                onClick={() => setPrimaryModel(m.id)}
                              >
                                <ArrowLeftRight className="w-3.5 h-3.5" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="text-xs">设为主模型</TooltipContent>
                          </Tooltip>
                        )}
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="删除模型"
                              onClick={() => setModelDeleteTarget(m)}
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="text-xs">删除模型</TooltipContent>
                        </Tooltip>
                      </ItemActions>
                    </Item>
                  );

                  return (
                    <Field>
                      <SectionHeader
                        title="已配置模型"
                        count={models.length}
                        action={
                          !modelAdding && providerGroups.length > 0 && canAddMore ? (
                            <Button variant="link" size="sm" onClick={startAddModel}>
                              <Plus className="w-3 h-3" />
                              {addButtonLabel}
                            </Button>
                          ) : undefined
                        }
                      />

                      {models.length === 0 && !modelAdding && (
                        <EmptyHint
                          title={providerGroups.length === 0 ? "该用户暂无可见模型" : "暂未配置模型"}
                        />
                      )}

                      {/* 主模型组 */}
                      {primaryList.length > 0 && (
                        <div className="flex flex-col gap-2 mb-3">
                          {primaryList.map((m) => renderModelRow(m, true))}
                        </div>
                      )}

                      {/* 备选模型组 */}
                      {backupList.length > 0 && (
                        <div className="flex flex-col gap-2">
                          {backupList.map((m) => renderModelRow(m, false))}
                        </div>
                      )}

                      {modelAdding && (
                        <SurfaceInner className="mt-2 overflow-hidden">
                          <div className="border-b border-[var(--border)] px-3 py-2">
                            <MetaMedium>模型配置</MetaMedium>
                          </div>
                          <Field className="px-3 py-2 gap-1.5 border-b border-[var(--border)]">
                            {/* 保持与 Agent 详情一致：仅展示 label + Select，无右侧统计文案、无厂商项后的数量括号 */}
                            <MetaMedium as="label" tone="secondary">模型厂商</MetaMedium>
                            <Select value={modelDraftProvider} onValueChange={handleDraftProviderChange}>
                              <SelectTrigger className="h-8 w-full border-[var(--border)] bg-[var(--card)] text-xs">
                                <SelectValue placeholder={providerGroups.length === 0 ? "该分组暂无可用厂商" : "选择模型厂商"} />
                              </SelectTrigger>
                              <SelectContent>
                                {providerGroups.length === 0 ? (
                                  <div className="px-3 py-2 text-xs text-[var(--text-weak)]">该分组暂无可用厂商</div>
                                ) : (
                                  providerGroups.map((g) => (
                                    <SelectItem key={g.key} value={g.key}>{g.label}</SelectItem>
                                  ))
                                )}
                              </SelectContent>
                            </Select>
                          </Field>
                          <Field className="px-3 py-2 gap-1.5">
                            <MetaMedium as="label" tone="secondary">模型名称</MetaMedium>
                            <Select value={modelDraftModelId} onValueChange={setModelDraftModelId}>
                              <SelectTrigger className="h-8 w-full border-[var(--border)] bg-[var(--card)] text-xs">
                                <SelectValue placeholder={!modelDraftProvider ? "请先选择模型厂商" : "选择模型名称"} />
                              </SelectTrigger>
                              <SelectContent>
                                {(() => {
                                  const list = providerGroups.find((g) => g.key === modelDraftProvider)?.models ?? [];
                                  if (list.length === 0) {
                                    return <div className="px-3 py-2 text-xs text-[var(--text-weak)]">该厂商下暂无可用模型</div>;
                                  }
                                  return list.map((m) => {
                                    const isCustom = m.provider === CUSTOM_PROVIDER_VALUE;
                                    return (
                                      <SelectItem key={m.id} value={m.id}>{isCustom ? m.name : m.version}</SelectItem>
                                    );
                                  });
                                })()}
                              </SelectContent>
                            </Select>
                          </Field>
                          <div className="flex justify-end gap-2 border-t border-[var(--border)] px-3 py-2">
                            <Button size="sm" variant="claw-outline" onClick={() => setModelAdding(false)}>
                              取消
                            </Button>
                            <Button
                              size="sm"
                              variant="dialog-confirm"
                              onClick={confirmAddModel}
                              disabled={!modelDraftProvider || !modelDraftModelId}
                            >
                              确认添加
                            </Button>
                          </div>
                        </SurfaceInner>
                      )}
                    </Field>
                  );
                })()}

                {/* 通道配置 */}
                {allowConfigChannel && (
                  <Field>
                    <SectionHeader
                      title="已接入通道"
                      count={channels.length}
                      action={
                        !channelAdding ? (
                          <Button variant="link" size="sm" onClick={startAddChannel}>
                            <Plus className="w-3 h-3" />
                            添加通道
                          </Button>
                        ) : undefined
                      }
                    />

                    {channels.length === 0 && !channelAdding && (
                      <EmptyHint title="暂未接入通道" />
                    )}

                    {channels.length > 0 && (
                      <div className="flex flex-col gap-2">
                        {channels.map((c) => (
                          <Item key={c.value} variant="outline" size="sm">
                            <ItemContent>
                              <BodyMedium className="truncate">{c.label}</BodyMedium>
                            </ItemContent>
                            <ItemActions>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon-sm"
                                    aria-label="删除通道"
                                    onClick={() => setChannelDeleteTarget(c)}
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="text-xs">删除通道</TooltipContent>
                              </Tooltip>
                            </ItemActions>
                          </Item>
                        ))}
                      </div>
                    )}

                    {channelAdding && (
                      <SurfaceInner className="mt-2 overflow-hidden">
                        <div className="border-b border-[var(--border)] px-3 py-2">
                          <MetaMedium>通道配置</MetaMedium>
                        </div>
                        <Field className="px-3 py-2 gap-1.5 border-b border-[var(--border)]">
                          <MetaMedium as="label" tone="secondary">通道类型</MetaMedium>
                          <Select value={channelDraft} onValueChange={handleChannelDraftChange}>
                            <SelectTrigger className="h-8 w-full border-[var(--border)] bg-[var(--card)] text-xs">
                              <SelectValue placeholder="选择要添加的通道" />
                            </SelectTrigger>
                            <SelectContent>
                              {availableChannelOptions.filter((c) => !channels.some((x) => x.value === c.value)).length === 0 ? (
                                <MetaText as="div" tone="weak" className="px-3 py-6 text-center">所有通道均已添加</MetaText>
                              ) : (
                                availableChannelOptions
                                  .filter((c) => !channels.some((x) => x.value === c.value))
                                  .map((c) => <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>)
                              )}
                            </SelectContent>
                          </Select>
                        </Field>

                        {channelDraftCfg && channelIsWechatLike && (
                          <div className="px-3 py-2 border-b border-[var(--border)]">
                            <Alert variant="info">
                              <AlertInfoIcon />
                              <AlertDescription>
                                微信通道通过扫码授权接入，管控端仅创建占位记录，实际扫码绑定由用户在用户端完成。
                              </AlertDescription>
                            </Alert>
                          </div>
                        )}

                        {channelDraftCfg && !channelIsWechatLike && channelDraftFieldsList.length > 0 &&
                          channelDraftFieldsList.map((field) => {
                            const secretKey = `__draft__:${field.key}`;
                            const visible = visibleSecrets.has(secretKey);
                            return (
                              <Field key={field.key} className="px-3 py-2 gap-1.5 border-b border-[var(--border)]">
                                <MetaMedium as="label" tone="secondary">{field.label}</MetaMedium>
                                <div className="relative">
                                  <Input
                                    type={field.secret && !visible ? "password" : "text"}
                                    value={channelDraftFields[field.key] ?? ""}
                                    onChange={(e) => setChannelDraftFields((prev) => ({ ...prev, [field.key]: e.target.value }))}
                                    placeholder={field.label}
                                    className="h-8 border-[var(--border)] bg-[var(--card)] pr-9 text-xs"
                                  />
                                  {field.secret && (
                                    <Button
                                      variant="ghost"
                                      size="icon-sm"
                                      aria-label={visible ? "隐藏密码" : "显示密码"}
                                      className="absolute right-1 top-1/2 -translate-y-1/2"
                                      onClick={() => toggleSecret(secretKey)}
                                    >
                                      {visible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                                    </Button>
                                  )}
                                </div>
                              </Field>
                            );
                          })}
                        <div className="flex justify-end gap-2 border-t border-[var(--border)] px-3 py-2">
                          <Button size="sm" variant="claw-outline" onClick={() => setChannelAdding(false)}>
                            取消
                          </Button>
                          <Button size="sm" variant="dialog-confirm" onClick={confirmAddChannel} disabled={!channelDraft}>
                            确认添加
                          </Button>
                        </div>
                      </SurfaceInner>
                    )}
                  </Field>
                )}

                {/* 技能配置 */}
                <Field>
                  <SectionHeader
                    title="已安装技能"
                    count={skills.length}
                    action={
                      <Button variant="link" size="sm" onClick={openSkillPicker}>
                        <Plus className="w-3 h-3" />
                        添加技能
                      </Button>
                    }
                  />
                  {skills.length === 0 ? (
                    <EmptyHint title="暂未安装技能" />
                  ) : (
                    <div className="flex flex-col gap-2">
                      {skills.map((s) => (
                        // 暂未提供技能删除能力，已安装项仅做展示，不再提供行内删除按钮
                        <Item key={s} variant="outline" size="sm">
                          <ItemContent>
                            <MiniBodyText className="truncate">{s}</MiniBodyText>
                          </ItemContent>
                        </Item>
                      ))}
                    </div>
                  )}
                </Field>
              </FieldGroup>
            )}
          </DialogBody>

          <DialogFooter>
            {step === 1 && (
              <>
                <Button variant="claw-outline" onClick={() => handleClose(false)}>取消</Button>
                <Button variant="dialog-confirm" disabled={!step1Valid} onClick={() => setStep(2)}>
                  下一步
                </Button>
              </>
            )}
            {step === 2 && (
              <>
                <Button variant="claw-outline" onClick={() => setStep(1)}>上一步</Button>
                <Button variant="dialog-confirm" disabled={!step2Valid} onClick={() => setStep(3)}>
                  下一步
                </Button>
              </>
            )}
            {step === 3 && (
              <>
                <Button variant="claw-outline" onClick={() => setStep(2)}>上一步</Button>
                <Button variant="dialog-confirm" onClick={handleConfirmCreate}>确认创建</Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 技能 - 第一级弹窗：从技能库选择 ===== */}
      <Dialog open={skillPickerOpen} onOpenChange={setSkillPickerOpen}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>添加技能</DialogTitle>
            <DialogDescription className="sr-only">从技能库中选择要为该 Agent 安装的技能</DialogDescription>
          </DialogHeader>
          <FieldGroup className="py-1 gap-3">
            <Field>
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
                <Input
                  value={skillSearch}
                  onChange={(e) => setSkillSearch(e.target.value)}
                  placeholder="输入关键词模糊搜索技能"
                  className="pl-9"
                />
              </div>
            </Field>
            {/*
             * 不预先罗列所有技能，仅在用户输入关键词后按模糊匹配展示结果：
             *   1) 未输入        → EmptyHint 提示引导
             *   2) 有输入无结果  → SurfaceInner 内提示"未找到匹配"
             *   3) 有输入有结果  → 复选列表（可多选后一次性安装）
             */}
            {!skillSearch.trim() ? (
              <EmptyHint title="请输入关键词搜索要添加的技能" />
            ) : (
              <SurfaceInner className="overflow-hidden">
                {skillCandidates.length === 0 ? (
                  <div className="px-4 py-8 text-center">
                    <MetaText tone="weak">未找到匹配的技能</MetaText>
                  </div>
                ) : (
                  <div className="max-h-64 overflow-y-auto divide-y divide-[var(--border)]">
                    {skillCandidates.map((s) => (
                      <label
                        key={s}
                        className="flex cursor-pointer items-center gap-3 px-3 py-2.5 hover:bg-[var(--accent)] transition-colors"
                      >
                        <Checkbox checked={skillChecked.includes(s)} onCheckedChange={() => toggleSkillCheck(s)} />
                        <MiniBodyText className="min-w-0 flex-1 truncate">{s}</MiniBodyText>
                      </label>
                    ))}
                  </div>
                )}
              </SurfaceInner>
            )}
          </FieldGroup>
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => setSkillPickerOpen(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              disabled={skillChecked.length === 0}
              onClick={confirmInstallSkills}
            >
              确认安装{skillChecked.length > 0 ? `（${skillChecked.length}）` : ""}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 删除模型确认（AlertDialog 二次确认） ===== */}
      <AlertDialog
        open={!!modelDeleteTarget}
        onOpenChange={(o) => { if (!o) setModelDeleteTarget(null); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除该模型？</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            将从本 Agent 的已配置模型中移除「{modelDeleteTarget?.providerLabel}」，此操作不可撤销。
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (modelDeleteTarget) removeModel(modelDeleteTarget.id);
                setModelDeleteTarget(null);
              }}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ===== 删除通道确认（AlertDialog 二次确认） ===== */}
      <AlertDialog
        open={!!channelDeleteTarget}
        onOpenChange={(o) => { if (!o) setChannelDeleteTarget(null); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除该通道？</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            将从本 Agent 的已接入通道中移除「{channelDeleteTarget?.label}」，此操作不可撤销。
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (channelDeleteTarget) removeChannel(channelDeleteTarget.value);
                setChannelDeleteTarget(null);
              }}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
