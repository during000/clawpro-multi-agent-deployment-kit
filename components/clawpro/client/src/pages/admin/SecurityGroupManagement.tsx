/**
 * SecurityGroupManagement - 管控端网络管理页
 * 采用 Tab 结构：私有网络和子网、安全组、公网、更多功能
 */
import { useState, useEffect, useRef, useLayoutEffect, useMemo, Fragment } from "react";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableHeader, TableBody, TableRow, TableExpandedRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import { SurfaceCard } from "@/components/ui/Surface";
import { Card } from "@/components/ui/card";
import { BodyMedium, BodyText, CardTitle, CompactText, HelperText, MetaMedium, MetaText, PanelTitle } from "@/components/ui/Typography";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Alert, AlertDescription, AlertOperationInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

import { toast } from "sonner";
import { useLocation } from "wouter";
import { useAdminDisabled } from "@/hooks/useAdminDisabled";
import { StatusTag } from "@/components/ui/status-tag";
import { Plus, Trash2, Pencil, Info, ExternalLink, Loader2, Check, ChevronDown, ChevronRight, ChevronLeft, Shield, Search, X, CircleAlert, GripVertical } from "lucide-react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
  type Modifier,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { ScopeSelect } from "@/components/ScopeSelect";
import { GroupSelect } from "@/components/GroupSelect";
// 与「模型配置 · 应用范围」对齐：共享 MemberManagement 的组织 mock，
//   - 列表行 badge 渲染完整组织路径
//   - 应用范围 Popover 使用同款树形多选（按 source 分桶 + 父子级联）
import type { UserGroup, GroupSource } from "./MemberManagement/types";
import { buildGroupTree, type GroupTreeNode } from "./MemberManagement/health";
import {
  MOCK_GROUPS as MOCK_ONEID_GROUPS_SHARED,
  MOCK_MANUAL_GROUPS as MOCK_MANUAL_GROUPS_SHARED,
} from "./MemberManagement/mock";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "@/components/ui/empty";

// ─────────────────────────────────────────────────────────────────────────────
// types / interfaces
// ─────────────────────────────────────────────────────────────────────────────

// ─── 子网 / VPC ──────────────────────────────────────────────────────────────

type SubnetEntity = {
  id: string;
  name: string;
  cidr: string;
  totalIp: number;
  remainingIp: number;
};

type VpcListItemType = "enterprise" | "group";

type VpcListItem = {
  id: string;
  vpcId: string;
  vpcName: string;
  cidr: string;
  type: VpcListItemType;
  // 应用范围：企业级（全局）固定为空数组；用户组类型支持多选，存用户组名数组
  associatedGroups: string[];
  subnetStrategy: "auto" | "specified";
  // 指定子网模式下，按可用区组织的子网 ID 列表；未分配该可用区则为空数组
  zoneSubnets: Record<string, string[]>;
  instanceCount: number;
};

// ─── 安全组 ──────────────────────────────────────────────────────────────────

type CloudSg = {
  sgId: string;
  cloudSgName: string;
  seq: number; // 1, 2, 3... 第 1 个是原生，≥2 为自动扩容
};

type SecurityGroup = {
  // [004] ClawPro 安全组对外不暴露 ID，name 作为企业内唯一标识 + 主键
  //   - 跨企业唯一性由 identifier + name 联合保证（identifier 由登录态隐式提供）
  //   - 前后端 API 全部以 name 作为参数（不再使用 sg.id）
  //   - 创建后 name 不可修改（避免重命名传播代价）
  name: string;
  remark: string;
  inboundCount: number;
  outboundCount: number;
  inboundRules: Rule[];
  outboundRules: Rule[];
  // [004] 该 ClawPro 安全组对应的云端安全组列表
  cloudSgs: CloudSg[];

  // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  // [004 · K12 · K13] v2.0 用户管理 & 应用范围 预留字段（本期不使用）
  //
  // 后端 security_groups 表在 004 建表时已预留以下字段，默认值保证本期
  // 行为不变。v2.0 激活"多条安全组 × 应用范围（组织）"能力时：
  //   - 只需在已有行上更新这几个字段 + 新增几行 SG 记录
  //   - 不需要 alter table、不需要改 API schema、不需要改前端数据契约
  //
  // 规则 K13：004 本期前端代码**不读写**这 3 个字段（如后端响应返回也忽略）
  // 未来 v2.0 前端代码可直接消费这些字段，本类型声明已为其预留位置
  //
  // scope_type:           "all" | "filtered"        DEFAULT "all"
  //   - all:      全企业生效（本期唯一取值）
  //   - filtered: 仅指定组织生效（v2.0 启用）
  // scope_group_ids:      string[]                  DEFAULT []
  //   - 仅 scope_type=filtered 时有意义
  // is_platform_default:  boolean                   DEFAULT true
  //   - v2.0 多条 SG 共存时，标记哪一条是"平台兜底"
  // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  // scope_type?: "all" | "filtered";    // v2.0 启用，本期不使用
  // scope_group_ids?: string[];          // v2.0 启用，本期不使用
  // is_platform_default?: boolean;       // v2.0 启用，本期不使用
};

// ─── 独立化升级状态 ────────────────────────────────────────────────────────────

// [004] 独立化升级状态（站点级，非 SG 级）
// 模拟后端 GET /admin/config/security-group 返回的 ClawPro namespace 字段
type MigrationStatus = {
  isLegacyMigrated: boolean;        // 本企业是否经历过独立化升级（存量=true，新建=false）
  ackedByCurrentAdmin: boolean;     // 当前管理员是否已关闭过蓝条
  legacySgId?: string;              // 原 sg-legacy（仅 isLegacyMigrated=true）
  agentCountAtMigration?: number;   // 迁移时的 Agent 数量快照（蓝条文案用）
};

// ─── 规则 ────────────────────────────────────────────────────────────────────

type Rule = {
  id: string;
  source: string;
  protocol: string;
  port: string;
  policy: string;
  remark: string;
  /** 是否由平台策略要求（删除时弹窗会差异化提示） */
  is_required?: boolean;
};

type RuleTableBodyProps = {
  rules: Rule[];
  type: "inbound" | "outbound";
  readonly?: boolean;
  onEdit?: (rule: Rule, type: "inbound" | "outbound") => void;
  onDelete?: (rule: Rule, type: "inbound" | "outbound") => void;
  /** 拖拽排序后回调，传入按新顺序排列的完整 rules 数组 */
  onReorder?: (reorderedRules: Rule[]) => void;
  paginate?: boolean;
};

type CommonRuleOptionKey = "allow-public" | "allow-ssh";

type CommonRuleOption = {
  key: CommonRuleOptionKey;
  label: string;
  description: string;
  defaultChecked: boolean;
  rules: {
    inbound: Array<Omit<Rule, "id">>;
    outbound: Array<Omit<Rule, "id">>;
  };
};

// ─── 树形多选辅助 ─────────────────────────────────────────────────────────────

type CheckState = "checked" | "unchecked" | "indeterminate";

// ─── Dialog Props ───────────────────────────────────────────────────────────

type CreateSecurityGroupDialogProps = {
  open: boolean;
  checkedOptions: CommonRuleOptionKey[];
  draft: {
    name: string;
    remark: string;
  };
  previewTab: "inbound" | "outbound";
  onOpenChange: (open: boolean) => void;
  onOptionToggle: (optionKey: CommonRuleOptionKey, checked: boolean) => void;
  onNameChange: (name: string) => void;
  onRemarkChange: (remark: string) => void;
  onPreviewTabChange: (tab: "inbound" | "outbound") => void;
  onCancel: () => void;
  onConfirm: () => void;
};

type SelectExistingSecurityGroupDialogProps = {
  open: boolean;
  searchKeyword: string;
  selectedSecurityGroup: SecurityGroup | null;
  previewTab: "outbound" | "inbound";
  candidateSecurityGroups: SecurityGroup[];
  candidatePage: number;
  candidateTotalPages: number;
  shouldShowPanelAccessAssist: boolean;
  isPanelAccessStaged: boolean;
  onOpenChange: (open: boolean) => void;
  onSearchChange: (value: string) => void;
  onClearSearch: () => void;
  onSelectSecurityGroup: (sg: SecurityGroup) => void;
  onCandidatePageChange: (page: number) => void;
  onTogglePanelAccessStaged: () => void;
  onPreviewTabChange: (tab: "outbound" | "inbound") => void;
  onCancel: () => void;
  onConfirm: () => void;
};

// ─────────────────────────────────────────────────────────────────────────────
// constants
// ─────────────────────────────────────────────────────────────────────────────

// ─── 可用区 ─────────────────────────────────────────────────────────────────

const AVAILABLE_ZONES = ["广州五区", "广州六区", "广州七区"];
const AUTO_ASSIGNED_VPC = { id: "vpc-jp7fjg13", name: "clawpro/default-vpc", cidr: "10.0.0.0/16" };
const NEW_GROUP_VPC_ID = "__new_group_vpc__";

// ─── 默认安全组身份 ───────────────────────────────────────────────────────────

const INITIAL_DEFAULT_SECURITY_GROUP_NAME = "ClawPro-Default";
const ENABLE_SECURITY_GROUP_EMPTY_STATE_DEMO = true;

// [004] ClawPro 默认安全组在云端的标准命名（K4 约定：clawpro-sg-{域名}-default-{序号}）
//       与 INITIAL_DEFAULT_SECURITY_GROUP_NAME 语义不同：
//         - INITIAL_DEFAULT_SECURITY_GROUP_NAME：ClawPro 安全组的对外 name（用户可见）
//         - DEFAULT_CLOUD_SECURITY_GROUP_NAME：对应到云端那一片云端安全组的 name（云端命名空间）
const DEFAULT_CLOUD_SECURITY_GROUP_NAME = "clawpro-sg-acme-default-01";

// ─── localStorage key ──────────────────────────────────────────────────────

// 默认安全组的本地快照 key：供平台策略页等其他管理员页面只读消费（单向同步）。
const DEFAULT_SECURITY_GROUP_SNAPSHOT_KEY = "admin_default_security_group_snapshot";
// [004] 独立化升级状态本地快照 key：模拟"管理员关闭蓝条"的持久化行为
const MIGRATION_ACK_STORAGE_KEY = "admin_sg_migration_ack";

// ─── 平台策略联动 ─────────────────────────────────────────────────────────────

const DEFAULT_PANEL_ACCESS_PORT = "443";

// 平台策略自动生成的所有必需规则定义（remark 用于匹配后端返回的已有规则）
const PLATFORM_REQUIRED_RULES: Array<Omit<Rule, "id"> & { is_required: true }> = [
  { source: "0.0.0.0/0", protocol: "TCP", port: "9833", policy: "允许", remark: "Agent 面板访问", is_required: true },
  { source: "0.0.0.0/0", protocol: "TCP", port: "6080", policy: "允许", remark: "云端浏览器访问", is_required: true },
];

// 判断一条规则是否匹配平台策略必需规则（按 source + protocol + port + policy 匹配）
function isPlatformRequiredRule(rule: Rule): boolean {
  return PLATFORM_REQUIRED_RULES.some(
    (pr) => rule.source === pr.source && rule.protocol === pr.protocol && rule.port === pr.port && rule.policy === pr.policy
  );
}

// 给规则列表中属于平台策略的规则标记 is_required
function markPlatformRequiredRules(rules: Rule[]): Rule[] {
  return rules.map((rule) =>
    !rule.is_required && isPlatformRequiredRule(rule) ? { ...rule, is_required: true } : rule
  );
}

// ─── 选择安全组弹窗 ───────────────────────────────────────────────────────────

const SECURITY_GROUP_DIALOG_PAGE_SIZE = 5;

// ─── ClawPro 安全组命名规范 ─────────────────────────────────────────────────

// [004] ClawPro 安全组名称命名规范（与后端 + 平台策略页保持一致）
//   - 首字符必须是字母（大小写均可）
//   - 中间允许字母、数字、短横线
//   - 末字符必须是字母或数字（不允许以短横线结尾）
//   - 总长 3–32 个字符
const CLAWPRO_SG_NAME_REGEX = /^[A-Za-z][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$/;
// 命名规则（非法态红色错误、提交 toast 复用此简短版本）
const CLAWPRO_SG_NAME_RULE = "支持字母、数字、短横线，以字母开头，3–32 个字符";
// 命名补充说明（与规则分两行展示，避免长句换行不可控）
const CLAWPRO_SG_NAME_NOTE = "创建后不可修改，对应的云端安全组由 ClawPro 自动创建并命名。";

// ─────────────────────────────────────────────────────────────────────────────
// mock data / 初始化数据
// ─────────────────────────────────────────────────────────────────────────────

// ─── 迁移影响范围 Mock 数据 ────────────────────────────────────────────────────

const MIGRATION_INSTANCES = [
  { user: "alice@acompany.com", instance: "openclaw-alice-01", currentNet: "vpc-aa1 / subnet-aa1", targetNet: "vpc-jp7fjg13 / subnet-gz5-001", note: "可迁移" },
  { user: "alice@acompany.com", instance: "openclaw-alice-02", currentNet: "vpc-aa1 / subnet-aa1", targetNet: "vpc-jp7fjg13 / subnet-gz5-001", note: "暂不支持迁移：已绑定负载均衡" },
  { user: "bob@acompany.com", instance: "openclaw-bob-01", currentNet: "vpc-bb2 / subnet-bb2", targetNet: "vpc-jp7fjg13 / subnet-gz6-001", note: "可迁移" },
  { user: "carol@acompany.com", instance: "openclaw-carol-01", currentNet: "vpc-cc3 / subnet-cc3", targetNet: "vpc-jp7fjg13 / subnet-gz5-001", note: "暂不支持迁移：存在辅助网卡" },
  { user: "carol@acompany.com", instance: "openclaw-carol-02", currentNet: "vpc-cc3 / subnet-cc3", targetNet: "vpc-jp7fjg13 / subnet-gz6-001", note: "可迁移" },
  { user: "dave@acompany.com", instance: "openclaw-dave-01", currentNet: "vpc-dd4 / subnet-dd4", targetNet: "vpc-jp7fjg13 / subnet-gz7-001", note: "暂不支持迁移：存在辅助 IP 未释放" },
  { user: "eve@acompany.com", instance: "openclaw-eve-01", currentNet: "vpc-ee5 / subnet-ee5", targetNet: "vpc-jp7fjg13 / subnet-gz5-001", note: "暂不支持迁移：实例状态不允许迁移" },
  { user: "frank@acompany.com", instance: "openclaw-frank-01", currentNet: "vpc-ff6 / subnet-ff6", targetNet: "vpc-jp7fjg13 / subnet-gz6-001", note: "可迁移" },
];

const MIGRATION_FAILURE_SCENARIOS: Record<string, { step: string; reason: string }> = {
  "openclaw-bob-01": {
    step: "切换实例网络",
    reason: "请稍后重试；如问题持续存在，请联系管理员处理",
  },
  "openclaw-frank-01": {
    step: "重启实例",
    reason: "请检查实例是否处于可操作状态后重试",
  },
};

// ─── 默认安全组规则（入站/出站） ──────────────────────────────────────────────

const DEFAULT_INBOUND = [
  { id: "1", source: "0.0.0.0/0", protocol: "ICMP", port: "ALL", policy: "允许", remark: "放通 Ping 服务" },
  { id: "2", source: "::/0", protocol: "ICMPv6", port: "ALL", policy: "允许", remark: "放通 Ping 服务（IPv6）" },
  { id: "3", source: "0.0.0.0/0", protocol: "TCP", port: "22", policy: "允许", remark: "放通 Linux SSH 登录" },
  { id: "4", source: "::/0", protocol: "TCP", port: "22", policy: "允许", remark: "放通 Linux SSH 登录（IPv6）" },
  { id: "5", source: "0.0.0.0/0", protocol: "TCP", port: "3389", policy: "允许", remark: "放通 Windows 远程登录" },
  { id: "6", source: "::/0", protocol: "TCP", port: "3389", policy: "允许", remark: "放通 Windows 远程登录（IPv6）" },
  { id: "7", source: "10.0.0.0/8", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通内网（云私有网络）" },
  { id: "8", source: "172.16.0.0/12", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通内网（云私有网络）" },
  { id: "9", source: "192.168.0.0/16", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通内网（云私有网络）" },
  { id: "10", source: "0.0.0.0/0", protocol: "TCP", port: "80", policy: "允许", remark: "Web 服务 HTTP（Apache、Nginx 等）" },
  { id: "11", source: "0.0.0.0/0", protocol: "TCP", port: "443", policy: "允许", remark: "Web 服务 HTTPS（Apache、Nginx 等）" },
  { id: "12", source: "0.0.0.0/0", protocol: "TCP", port: "18789", policy: "允许", remark: "Agent 服务端口" },
  { id: "13", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
];

const DEFAULT_OUTBOUND = [
  { id: "1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通所有出站流量" },
  { id: "2", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
];

// ─── 子网 / VPC ──────────────────────────────────────────────────────────────

// 系统自动分配的子网（按可用区，模拟后端返回的实际生效资源）
const AUTO_ASSIGNED_SUBNETS: Record<string, SubnetEntity> = {
  "广州五区": { id: "subnet-gz5-001", name: "clawpro/ap-guangzhou-5", cidr: "10.0.1.0/24", totalIp: 254, remainingIp: 220 },
  "广州六区": { id: "subnet-gz6-001", name: "clawpro/ap-guangzhou-6", cidr: "10.0.2.0/24", totalIp: 254, remainingIp: 180 },
  "广州七区": { id: "subnet-gz7-001", name: "clawpro/ap-guangzhou-7", cidr: "10.0.3.0/24", totalIp: 254, remainingIp: 240 },
};

// Mock VPC 列表
const MOCK_VPCS = [
  { id: "vpc-jp7fjg13", name: "auto_test_vpc_2", cidr: "10.1.0.0/16" },
  { id: "vpc-9lyx5t8h", name: "CHC-带外", cidr: "192.168.0.0/16" },
  { id: "vpc-ri7mmw6n", name: "CHC-部署", cidr: "192.168.0.0/16" },
  { id: "vpc-ab3cd4ef", name: "企业内网", cidr: "172.16.0.0/12" },
];

// Mock 子网列表（按 VPC ID 过滤）
const MOCK_SUBNETS: Record<string, SubnetEntity[]> = {
  "vpc-jp7fjg13": [
    { id: "subnet-nvupa1uw", name: "clawpro/ap-nanjing-1", cidr: "10.0.0.0/19", totalIp: 1022, remainingIp: 854 },
    { id: "subnet-f7t69gji", name: "lb_auto_test_subnet", cidr: "10.1.0.0/24", totalIp: 254, remainingIp: 180 },
    { id: "subnet-h8u80hkj", name: "lb_auto_test_subnet_2", cidr: "10.1.1.0/24", totalIp: 254, remainingIp: 52 },
    { id: "subnet-p9q0r1st", name: "clawpro/ap-nanjing-3", cidr: "10.0.64.0/20", totalIp: 510, remainingIp: 500 },
  ],
  "vpc-9lyx5t8h": [
    { id: "subnet-gaclgbzu", name: "带外管理", cidr: "192.168.20.0/24", totalIp: 254, remainingIp: 200 },
  ],
  "vpc-ri7mmw6n": [
    // 广州五区：1 个正常子网
    { id: "subnet-mn3op5qr", name: "部署子网A", cidr: "192.168.1.0/24", totalIp: 254, remainingIp: 120 },
    // 广州六区：1 个正常子网（演示"部分异常、部分可用"混合场景）
    { id: "subnet-q47nb3hv", name: "部署子网B", cidr: "192.168.2.0/24", totalIp: 254, remainingIp: 80 },
    // 广州七区：1 个正常子网（演示"部分异常、部分可用"混合场景）
    { id: "subnet-x82ab1c", name: "部署子网C", cidr: "192.168.3.0/24", totalIp: 254, remainingIp: 60 },
    // [Mock] 异常态演示：以下子网均被管理员在腾讯云控制台直接删除，
    // 保留 id 但 name/cidr 置空，触发"配置异常"策略级提示与"异常子网"治理列表。
    // 广州六区：3 个异常子网
    { id: "subnet-st6uv7wx", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
    { id: "subnet-k91vax2m", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
    { id: "subnet-p3r8nq6t", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
    // 广州七区：4 个异常子网
    { id: "subnet-7dk29plx", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
    { id: "subnet-92mx8qa", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
    { id: "subnet-88avpl2", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
    { id: "subnet-x91ab2d", name: "", cidr: "", totalIp: 0, remainingIp: 0 },
  ],
  "vpc-ab3cd4ef": [
    { id: "subnet-yz9ab1cd", name: "企业内网子网", cidr: "172.16.1.0/24", totalIp: 254, remainingIp: 90 },
    { id: "subnet-cd1ef2gh", name: "企业内网-办公区", cidr: "172.16.2.0/24", totalIp: 254, remainingIp: 120 },
    { id: "subnet-ij3kl4mn", name: "企业内网-应用区", cidr: "172.16.3.0/24", totalIp: 254, remainingIp: 200 },
    { id: "subnet-op5qr6st", name: "企业内网-数据区", cidr: "172.16.4.0/24", totalIp: 254, remainingIp: 60 },
    { id: "subnet-uv7wx8yz", name: "企业内网-灾备区", cidr: "172.16.5.0/24", totalIp: 254, remainingIp: 240 },
    { id: "subnet-ab1cd2ef", name: "企业内网-测试区", cidr: "172.16.6.0/24", totalIp: 254, remainingIp: 180 },
    { id: "subnet-gh3ij4kl", name: "企业内网-生产区", cidr: "172.16.7.0/24", totalIp: 254, remainingIp: 50 },
    { id: "subnet-mn5op6qr", name: "企业内网-预发区", cidr: "172.16.8.0/24", totalIp: 254, remainingIp: 220 },
  ],
};

// ─── Mock 安全组数据 ──────────────────────────────────────────────────────────

// [004] 数据契约扩展：一个 ClawPro 安全组在腾讯云端实际对应的云安全组列表。
//   - 默认主安全组：若实例数超 2000 会自动追加分片（cloudSgs 长度 >1）
//   - 非默认主：长度恒为 1
//   - sgId: 腾讯云返回的 sg-xxxxxx
//   - cloudSgName: 腾讯云控制台中的 name
//       · 序号 01：与 ClawPro 安全组同名（管理员自定义或导入而来）
//       · 序号 ≥02：ClawPro 自动命名为 clawpro-sg-{域名}-{组织slug|default}-{序号}
//   - seq: 云端分片序号（01/02/03...），UI 不露出 base/shard 术语，只用序号
const MOCK_SECURITY_GROUPS: SecurityGroup[] = [
  {
    name: "ClawPro-Default",
    remark: "Agent 默认安全组",
    inboundCount: 13,
    outboundCount: 2,
    inboundRules: DEFAULT_INBOUND,
    outboundRules: DEFAULT_OUTBOUND,
    // [004] 默认主（存量迁移场景）：ClawPro 迁移时自动创建的第 1 片使用
    //       clawpro-sg-{域名}-default-{序号} 命名约定，实例超 2000 扩出第 2/3 片
    cloudSgs: [
      { sgId: "sg-cur00001", cloudSgName: DEFAULT_CLOUD_SECURITY_GROUP_NAME, seq: 1 },
      { sgId: "sg-eqei0gwz", cloudSgName: "clawpro-sg-acme-default-02", seq: 2 },
      { sgId: "sg-ca78qfkx", cloudSgName: "clawpro-sg-acme-default-03", seq: 3 },
    ],
  },
  {
    name: "Web-Server-SG",
    remark: "Web 服务器安全组，开放 80/443",
    inboundCount: 5,
    outboundCount: 2,
    inboundRules: [
      { id: "w1", source: "0.0.0.0/0", protocol: "TCP", port: "80", policy: "允许", remark: "HTTP" },
      { id: "w2", source: "0.0.0.0/0", protocol: "TCP", port: "443", policy: "允许", remark: "HTTPS" },
      { id: "w3", source: "10.0.0.0/8", protocol: "ALL", port: "ALL", policy: "允许", remark: "内网放通" },
      { id: "w4", source: "0.0.0.0/0", protocol: "TCP", port: "22", policy: "允许", remark: "SSH" },
      { id: "w5", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
    ],
    outboundRules: [
      { id: "wo1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通所有出站流量" },
      { id: "wo2", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
    ],
    cloudSgs: [{ sgId: "sg-web00002", cloudSgName: "Web-Server-SG", seq: 1 }],
  },
  {
    name: "Strict-Isolation-SG",
    remark: "严格隔离，仅允许必要端口",
    inboundCount: 3,
    outboundCount: 1,
    inboundRules: [
      { id: "s1", source: "10.0.0.0/8", protocol: "TCP", port: "22", policy: "允许", remark: "内网 SSH" },
      { id: "s2", source: "10.0.0.0/8", protocol: "TCP", port: "18789", policy: "允许", remark: "Agent 内网端口" },
      { id: "s3", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
    ],
    outboundRules: [
      { id: "so1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通所有出站流量" },
    ],
    cloudSgs: [{ sgId: "sg-strct003", cloudSgName: "Strict-Isolation-SG", seq: 1 }],
  },
  {
    name: "Dev-Test-SG",
    remark: "开发测试环境，开放所有端口",
    inboundCount: 2,
    outboundCount: 1,
    inboundRules: [
      { id: "d1", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "允许", remark: "全放通（仅测试用）" },
      { id: "d2", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
    ],
    outboundRules: [
      { id: "do1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通所有出站流量" },
    ],
    cloudSgs: [{ sgId: "sg-devtst04", cloudSgName: "Dev-Test-SG", seq: 1 }],
  },
];

const MOCK_SECURITY_GROUP_DIALOG_EXTRA_CANDIDATES: SecurityGroup[] = [
  {
    name: "Office-Standard-SG",
    remark: "办公网标准安全组，适合常规用户办公实例",
    inboundCount: 4,
    outboundCount: 2,
    inboundRules: [
      { id: "o1", source: "10.10.0.0/16", protocol: "TCP", port: "22", policy: "允许", remark: "办公网 SSH" },
      { id: "o2", source: "10.10.0.0/16", protocol: "TCP", port: "3389", policy: "允许", remark: "办公网远程桌面" },
      { id: "o3", source: "10.10.0.0/16", protocol: "TCP", port: "443", policy: "允许", remark: "办公系统 HTTPS" },
      { id: "o4", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "默认拒绝其余访问" },
    ],
    outboundRules: DEFAULT_OUTBOUND,
    cloudSgs: [{ sgId: "sg-offic005", cloudSgName: "Office-Standard-SG", seq: 1 }],
  },
  {
    name: "Data-Processing-SG",
    remark: "数据处理节点专用，保留必要服务访问",
    inboundCount: 3,
    outboundCount: 2,
    inboundRules: [
      { id: "dp1", source: "172.20.0.0/16", protocol: "TCP", port: "22", policy: "允许", remark: "运维 SSH" },
      { id: "dp2", source: "172.20.0.0/16", protocol: "TCP", port: "9090", policy: "允许", remark: "监控采集" },
      { id: "dp3", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "默认拒绝其余访问" },
    ],
    outboundRules: DEFAULT_OUTBOUND,
    cloudSgs: [{ sgId: "sg-datap006", cloudSgName: "Data-Processing-SG", seq: 1 }],
  },
  {
    name: "Bastion-Only-SG",
    remark: "仅允许堡垒机来源访问的安全组",
    inboundCount: 2,
    outboundCount: 1,
    inboundRules: [
      { id: "b1", source: "10.200.0.12/32", protocol: "TCP", port: "22", policy: "允许", remark: "堡垒机 SSH" },
      { id: "b2", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "默认拒绝其余访问" },
    ],
    outboundRules: [
      { id: "bo1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通所有出站流量" },
    ],
    cloudSgs: [{ sgId: "sg-bastn007", cloudSgName: "Bastion-Only-SG", seq: 1 }],
  },
  {
    name: "Application-Cluster-SG",
    remark: "应用集群通用安全组，放通服务编排端口",
    inboundCount: 4,
    outboundCount: 2,
    inboundRules: [
      { id: "a1", source: "10.30.0.0/16", protocol: "TCP", port: "8080", policy: "允许", remark: "应用服务入口" },
      { id: "a2", source: "10.30.0.0/16", protocol: "TCP", port: "8443", policy: "允许", remark: "应用服务 HTTPS" },
      { id: "a3", source: "10.30.0.0/16", protocol: "TCP", port: "22", policy: "允许", remark: "集群运维 SSH" },
      { id: "a4", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "默认拒绝其余访问" },
    ],
    outboundRules: DEFAULT_OUTBOUND,
    cloudSgs: [{ sgId: "sg-applc008", cloudSgName: "Application-Cluster-SG", seq: 1 }],
  },
  {
    name: "Audit-Readonly-SG",
    remark: "审计查看实例，限制仅查询与日志上报",
    inboundCount: 3,
    outboundCount: 2,
    inboundRules: [
      { id: "ar1", source: "10.40.8.0/24", protocol: "TCP", port: "22", policy: "允许", remark: "审计网 SSH" },
      { id: "ar2", source: "10.40.8.0/24", protocol: "TCP", port: "5601", policy: "允许", remark: "日志检索" },
      { id: "ar3", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "默认拒绝其余访问" },
    ],
    outboundRules: DEFAULT_OUTBOUND,
    cloudSgs: [{ sgId: "sg-audit009", cloudSgName: "Audit-Readonly-SG", seq: 1 }],
  },
  {
    name: "Trial-Sandbox-SG",
    remark: "试用沙箱环境，方便演示分页与搜索效果",
    inboundCount: 3,
    outboundCount: 1,
    inboundRules: [
      { id: "t1", source: "192.168.50.0/24", protocol: "TCP", port: "22", policy: "允许", remark: "试用环境 SSH" },
      { id: "t2", source: "192.168.50.0/24", protocol: "TCP", port: "443", policy: "允许", remark: "试用环境 HTTPS" },
      { id: "t3", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "默认拒绝其余访问" },
    ],
    outboundRules: [
      { id: "to1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通所有出站流量" },
    ],
    cloudSgs: [{ sgId: "sg-trial010", cloudSgName: "Trial-Sandbox-SG", seq: 1 }],
  },
];

const MOCK_SECURITY_GROUP_DIALOG_CANDIDATES = [
  ...MOCK_SECURITY_GROUPS,
  ...MOCK_SECURITY_GROUP_DIALOG_EXTRA_CANDIDATES,
];

// ─── 常用规则选项 ─────────────────────────────────────────────────────────────

const COMMON_RULE_OPTIONS: CommonRuleOption[] = [
  {
    key: "allow-public",
    label: "允许公网访问",
    description: "",
    defaultChecked: true,
    rules: {
      inbound: [],
      outbound: [
        {
          source: "0.0.0.0/0",
          protocol: "ALL",
          port: "ALL",
          policy: "允许",
          remark: "可访问公网",
        },
        {
          source: "::/0",
          protocol: "ALL",
          port: "ALL",
          policy: "允许",
          remark: "可访问公网",
        },
      ],
    },
  },
  {
    key: "allow-ssh",
    label: "允许 Linux SSH 登录",
    description: "",
    defaultChecked: false,
    rules: {
      inbound: [
        {
          source: "0.0.0.0/0",
          protocol: "TCP",
          port: "22",
          policy: "允许",
          remark: "放通Linux SSH登录",
        },
        {
          source: "::/0",
          protocol: "TCP",
          port: "22",
          policy: "允许",
          remark: "放通Linux SSH登录",
        },
      ],
      outbound: [],
    },
  },
];

// ─── 用户组 / 组织共享数据 ─────────────────────────────────────────────────────

// 与「模型配置」对齐：合并 OneID + 自建组织作为完整组织树数据源。
// 应用范围 Popover、列表 badge 渲染、新增/编辑用户组配置都以此为组织数据源。
const ALL_GROUPS_SHARED: UserGroup[] = [...MOCK_ONEID_GROUPS_SHARED, ...MOCK_MANUAL_GROUPS_SHARED];

// ─── VPC × 可用区 × 子网映射 ────────────────────────────────────────────────

const VPC_ZONE_SUBNETS: Record<string, Record<string, string[]>> = {
  "vpc-jp7fjg13": {
    "广州五区": ["subnet-nvupa1uw", "subnet-f7t69gji"],
    "广州六区": ["subnet-h8u80hkj"],
    "广州七区": ["subnet-p9q0r1st"],
  },
  "vpc-9lyx5t8h": {
    "广州五区": ["subnet-gaclgbzu"],
    "广州六区": [],
    "广州七区": [],
  },
  "vpc-ri7mmw6n": {
    "广州五区": ["subnet-mn3op5qr"],
    "广州六区": ["subnet-q47nb3hv", "subnet-st6uv7wx", "subnet-k91vax2m", "subnet-p3r8nq6t"],
    "广州七区": ["subnet-x82ab1c", "subnet-7dk29plx", "subnet-92mx8qa", "subnet-88avpl2", "subnet-x91ab2d"],
  },
  "vpc-ab3cd4ef": {
    "广州五区": ["subnet-yz9ab1cd", "subnet-cd1ef2gh", "subnet-ab1cd2ef", "subnet-gh3ij4kl", "subnet-mn5op6qr"],
    "广州六区": ["subnet-ij3kl4mn", "subnet-op5qr6st"],
    "广州七区": ["subnet-uv7wx8yz"],
  },
};

// ─── 初始 VPC 列表 ────────────────────────────────────────────────────────────

const INITIAL_VPC_LIST: VpcListItem[] = [
  {
    id: "vpc-row-001",
    vpcId: "vpc-jp7fjg13",
    vpcName: "企业默认网络",
    cidr: "10.0.0.0/16",
    type: "enterprise",
    associatedGroups: [],
    subnetStrategy: "auto",
    zoneSubnets: { "广州五区": [], "广州六区": [], "广州七区": [] },
    instanceCount: 36,
  },
  {
    id: "vpc-row-002",
    vpcId: "vpc-9lyx5t8h",
    vpcName: "研发组网络",
    cidr: "10.1.0.0/16",
    type: "group",
    associatedGroups: ["研发组"],
    subnetStrategy: "specified",
    zoneSubnets: { "广州五区": ["subnet-gaclgbzu"], "广州六区": [], "广州七区": [] },
    instanceCount: 12,
  },
  {
    id: "vpc-row-003",
    vpcId: "vpc-ri7mmw6n",
    // [Mock] 异常态演示：VPC 自身正常，但其下部分子网被云控制台删除，
    // 用于在列表展开行验证「⚠ 配置待更新」徽章 + 子网明细「另有 N 个已删除子网」收敛展示。
    vpcName: "产品组网络",
    cidr: "10.2.0.0/16",
    type: "group",
    associatedGroups: ["产品组"],
    subnetStrategy: "specified",
    zoneSubnets: {
      "广州五区": ["subnet-mn3op5qr"],
      "广州六区": ["subnet-q47nb3hv", "subnet-st6uv7wx", "subnet-k91vax2m", "subnet-p3r8nq6t"],
      "广州七区": ["subnet-x82ab1c", "subnet-7dk29plx", "subnet-92mx8qa", "subnet-88avpl2", "subnet-x91ab2d"],
    },
    instanceCount: 8,
  },
  {
    // [Mock] 异常态演示：VPC 自身在腾讯云控制台被直接删除，仅保留 vpcId；
    // 历史子网配置仍存于策略中，但实际资源已不存在，编辑弹窗应自动清空 VPC 与所有子网，
    // 用于验证"VPC 已删除"场景下的清理逻辑与橙色顶部提示。
    id: "vpc-row-004",
    vpcId: "vpc-x7ab29cd",
    vpcName: "",
    cidr: "",
    type: "group",
    associatedGroups: ["设计组"],
    subnetStrategy: "specified",
    zoneSubnets: {
      "广州五区": ["subnet-a91bc2d1"],
      "广州六区": ["subnet-b72kd8f2"],
      "广州七区": ["subnet-c81mn3q7"],
    },
    instanceCount: 0,
  },
];

// [Mock] 演示空态：URL 带 ?state=empty 时，列表初始仅保留预设策略（type === "enterprise"）一行，
// 用于展示"新建企业首次进入网络管理"的初始体验。仅作用于 vpcList，不联动其他 mock 状态。
function getInitialVpcList(): VpcListItem[] {
  if (typeof window === "undefined") return INITIAL_VPC_LIST;
  try {
    const params = new URLSearchParams(window.location.search);
    if (params.get("state") === "empty") {
      return INITIAL_VPC_LIST.filter((v) => v.type === "enterprise");
    }
  } catch {
    /* SSR / 非 URL 环境兜底 */
  }
  return INITIAL_VPC_LIST;
}

// ─── Tab 定义 ───────────────────────────────────────────────────────────────

const TABS = [
  {
    id: "vpc",
    label: "私有网络和子网",
    description: "配置 Agent 实例的私有网络和子网部署策略。",
  },
  {
    id: "security",
    label: "安全组",
    description: "配置 ClawPro 安全组的入站与出站规则，管控 Agent 云服务器的网络流量策略。",
  },
  {
    id: "public",
    label: "公网",
    description: "",
  },
  {
    id: "coming",
    label: "更多功能",
    description: "更多功能即将上线。",
  },
];

const MORE_FEATURE_CARDS = [
  {
    title: "模型加速服务",
    description: "为 Agent 调用海外模型或国内模型提供专属优化链路，实现跨境/跨网访问的低延迟、高稳定传输，显著提升大模型交互体验",
    iconSrc: "/assets/admin-network-features/model-acceleration.svg",
  },
  {
    title: "公网极速接入",
    description: "提供全球范围内广覆盖、大带宽、低延时的公网出口和高性能接入网关，保障 Agent 各场景下极速、灵活、稳定的网络接入体验",
    iconSrc: "/assets/admin-network-features/public-fast-access.svg",
  },
  {
    title: "企业网络环境互通",
    description: "为 Agent 平台与企业 IDC 之间提供大带宽、高速、安全的互通能力，保障云上云下协同",
    iconSrc: "/assets/admin-network-features/enterprise-network-interconnect.svg",
  },
];

// ─────────────────────────────────────────────────────────────────────────────
// 纯工具函数
// ─────────────────────────────────────────────────────────────────────────────

// ─── 迁移状态 ────────────────────────────────────────────────────────────────

// [004] Mock: 根据 URL query 切换不同场景，便于 vibecoding 演示
//   - 默认（无 query）：新建企业 · 无迁移（迁移条不显示）
//   - ?state=migrate：存量企业 · 强制未 ack（显示迁移条 + 完整迁移流程）
//   - ?state=unacked：[Legacy] 存量企业 · 强制未 ack（=migrate，向后兼容）
//   - ?state=acked：  [Legacy] 存量企业 · 强制已 ack（保留以演示已确认态）
//   - ?state=new：    [Legacy] 新建企业 · 无迁移（=默认，向后兼容）
function getInitialMigrationStatus(): MigrationStatus {
  if (typeof window === "undefined") {
    return { isLegacyMigrated: false, ackedByCurrentAdmin: false };
  }
  const params = new URLSearchParams(window.location.search);
  const state = params.get("state");

  // 无 state / state=new：新建企业，无迁移条
  if (state === null || state === "new") {
    return { isLegacyMigrated: false, ackedByCurrentAdmin: false };
  }

  // 仅 migrate / unacked / acked 走存量企业分支；其他未识别 state（如 empty）→ 同默认无迁移
  if (state !== "migrate" && state !== "unacked" && state !== "acked") {
    return { isLegacyMigrated: false, ackedByCurrentAdmin: false };
  }

  // ack 判定：
  //   - migrate / unacked → 强制未 ack（演示完整迁移流程）
  //   - acked             → 强制已 ack（演示已确认态）
  const ackedByCurrentAdmin = state === "acked";

  return {
    isLegacyMigrated: true,
    ackedByCurrentAdmin,
    // 原 sg-legacy：用贴近真实"管理员接手 ClawPro 之前自己的 sg"的命名
    legacySgId: "default-sg",
    agentCountAtMigration: 127,
  };
}

// ─── 安全组 id 与查找 ─────────────────────────────────────────────────────────

// [004] 生成符合云端格式的 sgId：sg- + 8 位随机字符（小写字母 + 数字）
//   云端真实 sg-id 格式：sg- 后接 8 位字符（如 sg-eqei0gwz）
//   ClawPro 安全组的本地 id 不强制此格式（仅 UI 不展示的内部唯一键）
function generateRandomSgId(): string {
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
  let suffix = "";
  for (let i = 0; i < 8; i++) {
    suffix += chars[Math.floor(Math.random() * chars.length)];
  }
  return `sg-${suffix}`;
}

function findSecurityGroupByName(securityGroupName: string) {
  return MOCK_SECURITY_GROUPS.find((sg) => sg.name === securityGroupName) ?? null;
}

function getInitialDefaultSecurityGroup(): SecurityGroup | null {
  // 优先读取本地快照：让「网络管理」与「平台策略」等页面共享同一份默认安全组状态，
  // 避免组件 re-mount 后（例如在侧边栏切换管理页）把之前的切换/补规则动作丢失。
  const snapshotRaw = typeof window !== "undefined"
    ? window.localStorage.getItem(DEFAULT_SECURITY_GROUP_SNAPSHOT_KEY)
    : null;
  if (snapshotRaw) {
    try {
      const snapshot = JSON.parse(snapshotRaw) as {
        name?: string;
        remark?: string;
        inboundRules?: Rule[];
        outboundRules?: Rule[];
        cloudSgs?: CloudSg[]; // [004] 持久化 ClawPro 安全组身份信息
      };
      if (snapshot && snapshot.name && Array.isArray(snapshot.inboundRules)) {
        const template = findSecurityGroupByName(snapshot.name);
        if (template) {
          return {
            ...template,
            name: snapshot.name ?? template.name,
            remark: snapshot.remark ?? template.remark,
            inboundRules: snapshot.inboundRules,
            inboundCount: snapshot.inboundRules.length,
            outboundRules: snapshot.outboundRules ?? template.outboundRules,
            outboundCount: (snapshot.outboundRules ?? template.outboundRules).length,
            // [004] cloudSgs 恒定不变：优先用快照中的（ClawPro 安全组身份），否则用 template 的
            cloudSgs: snapshot.cloudSgs ?? template.cloudSgs,
          };
        }
        // [004] 用户自定义 name 场景：用快照完整构造，cloudSgs 兜底单分片
        return {
          name: snapshot.name,
          remark: snapshot.remark ?? "",
          inboundCount: snapshot.inboundRules.length,
          outboundCount: (snapshot.outboundRules ?? []).length,
          inboundRules: snapshot.inboundRules,
          outboundRules: snapshot.outboundRules ?? [],
          cloudSgs: snapshot.cloudSgs ?? [
            { sgId: generateRandomSgId(), cloudSgName: snapshot.name, seq: 1 },
          ],
        };
      }
    } catch {
      // JSON 解析失败则 fallback 到 demo / 默认逻辑
    }
  }

  // [004] 存量企业（isLegacyMigrated=true）一定已经有 ClawPro 安全组（由迁移 Worker 自动创建），
  //       currentSg 不可能为 null，否则会出现"蓝色框说 ClawPro 已创建 / 主卡片说未配置"的自相矛盾 UI。
  //       故：存量企业场景下强制返回默认主 MOCK SG（跳过"未配置"演示开关）
  const migrationStatus = getInitialMigrationStatus();
  if (migrationStatus.isLegacyMigrated) {
    return findSecurityGroupByName(INITIAL_DEFAULT_SECURITY_GROUP_NAME);
  }

  // 仅将“未配置默认安全组”的演示开关收口在初始化阶段，避免主流程持续混入 demo 判断。
  // [004] 此分支只对新建企业（isLegacyMigrated=false）生效，模拟"首次配置前"的空态
  if (ENABLE_SECURITY_GROUP_EMPTY_STATE_DEMO) {
    return null;
  }

  return findSecurityGroupByName(INITIAL_DEFAULT_SECURITY_GROUP_NAME);
}

// 写入默认安全组快照：currentSg 为 null 时清除 key。
// [004] 快照需持久化 ClawPro 安全组的完整身份（含 cloudSgs / remark / outboundRules），
//       否则刷新后若快照 name 不在 MOCK 里，身份信息会丢失（B15 违约）。
function writeDefaultSecurityGroupSnapshot(
  currentSg: SecurityGroup | null,
  inboundRules: Rule[],
) {
  if (!currentSg) {
    localStorage.removeItem(DEFAULT_SECURITY_GROUP_SNAPSHOT_KEY);
    return;
  }
  const snapshot = {
    name: currentSg.name,
    remark: currentSg.remark,
    inboundRules,
    outboundRules: currentSg.outboundRules,
    cloudSgs: currentSg.cloudSgs,
  };
  localStorage.setItem(DEFAULT_SECURITY_GROUP_SNAPSHOT_KEY, JSON.stringify(snapshot));
}

// ─── 规则比较 / 构造 ──────────────────────────────────────────────────────────

function isSameRuleContent(rule: Omit<Rule, "id">, expectedRule: Omit<Rule, "id">) {
  return (
    rule.source === expectedRule.source &&
    rule.protocol === expectedRule.protocol &&
    rule.port === expectedRule.port &&
    rule.policy === expectedRule.policy &&
    rule.remark === expectedRule.remark
  );
}

function buildRulesFromOptions(checkedKeys: CommonRuleOptionKey[]) {
  const inbound: Array<Omit<Rule, "id">> = [];
  const outbound: Array<Omit<Rule, "id">> = [];

  COMMON_RULE_OPTIONS.filter((o) => checkedKeys.includes(o.key)).forEach((option) => {
    option.rules.inbound.forEach((r) => {
      if (!inbound.some((existing) => isSameRuleContent(existing, r))) {
        inbound.push(r);
      }
    });
    option.rules.outbound.forEach((r) => {
      if (!outbound.some((existing) => isSameRuleContent(existing, r))) {
        outbound.push(r);
      }
    });
  });

  const timestamp = Date.now();
  const toRules = (rules: Array<Omit<Rule, "id">>, direction: "inbound" | "outbound"): Rule[] =>
    rules.map((rule, index) => ({
      id: `rule-${direction}-${timestamp}-${index}`,
      ...rule,
    }));

  return {
    inbound: toRules(inbound, "inbound"),
    outbound: toRules(outbound, "outbound"),
  };
}

// ─── 平台策略联动（面板访问规则） ───────────────────────────────────────────

function buildPanelAccessRequiredRules(panelAccessPort: string): Array<Omit<Rule, "id">> {
  return [
    {
      source: "0.0.0.0/0",
      protocol: "TCP",
      port: panelAccessPort,
      policy: "允许",
      remark: "允许访问 Agent 面板",
    },
  ];
}

function doesRulePortCoverRequiredPort(rulePort: string, requiredPort: string) {
  if (rulePort === "ALL") {
    return true;
  }

  if (rulePort === requiredPort) {
    return true;
  }

  const requiredPortNumber = Number(requiredPort);
  if (Number.isNaN(requiredPortNumber)) {
    return false;
  }

  return rulePort.split(",").some((segment) => {
    const normalizedSegment = segment.trim();
    if (normalizedSegment === requiredPort) {
      return true;
    }

    const rangeMatch = normalizedSegment.match(/^(\d+)-(\d+)$/);
    if (!rangeMatch) {
      return false;
    }

    const rangeStart = Number(rangeMatch[1]);
    const rangeEnd = Number(rangeMatch[2]);
    return requiredPortNumber >= rangeStart && requiredPortNumber <= rangeEnd;
  });
}

function doesRuleCoverPanelAccessRequirement(rule: Rule, expectedRule: Omit<Rule, "id">) {
  return rule.policy === "允许"
    && rule.source === expectedRule.source
    && (rule.protocol === expectedRule.protocol || rule.protocol === "ALL")
    && doesRulePortCoverRequiredPort(rule.port, expectedRule.port);
}

function buildSecurityGroupWithPanelAccessRules(securityGroup: SecurityGroup, panelAccessPort: string): SecurityGroup {
  const requiredRules = buildPanelAccessRequiredRules(panelAccessPort);
  const missingRules = requiredRules.filter((expectedRule) =>
    !securityGroup.inboundRules.some((rule) => doesRuleCoverPanelAccessRequirement(rule, expectedRule))
  );

  // 给已存在的、覆盖了平台策略需求的规则标记 is_required
  const markedInboundRules = securityGroup.inboundRules.map((rule) => {
    const coversPlatformRequirement = requiredRules.some((expectedRule) =>
      doesRuleCoverPanelAccessRequirement(rule, expectedRule)
    );
    if (coversPlatformRequirement && !rule.is_required) {
      return { ...rule, is_required: true };
    }
    return rule;
  });

  if (missingRules.length === 0) {
    return {
      ...securityGroup,
      inboundRules: markedInboundRules,
    };
  }

  const timestamp = Date.now();
  const injectedRules: Rule[] = missingRules.map((rule, index) => ({
    id: `panel-access-${timestamp}-${index}`,
    ...rule,
    is_required: true,
  }));
  const inboundRules = [...injectedRules, ...markedInboundRules];

  return {
    ...securityGroup,
    inboundRules,
    inboundCount: inboundRules.length,
  };
}

// ─── ClawPro 安全组命名校验 ───────────────────────────────────────────────────

function isValidClawproSgName(name: string): boolean {
  return CLAWPRO_SG_NAME_REGEX.test(name);
}

// ─── 用户组织：树形多选工具函数（与 PlatformPolicy GroupTagSelector 同款语义） ────
//
// 关键差异（相对前版）：
//   - getCheckState：祖先被选中时，子节点视为 checked（"父覆盖子" 语义）
//   - 新增 isNodeOrAncestorSelected / hasSelectedDescendant / aggregateSelection，
//     供 GroupTagSelector 在勾选父节点后递归向上聚合（兄弟全选时合并到父级）
//   - 路径分隔符使用 " / "（前后带空格），与 PlatformPolicy 视觉一致

/** id → 完整组织路径（父/子，" / " 拼接） */
function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const g = groups.find((x) => x.id === groupId);
  if (!g) return groupId;
  const parts: string[] = [g.name];
  let current = g;
  while (current.parentId) {
    const parent = groups.find((x) => x.id === current.parentId);
    if (!parent) break;
    parts.unshift(parent.name);
    current = parent;
  }
  return parts.join(" / ");
}

/** name → 完整组织路径（专给列表行 GroupBadges 使用） */
function getGroupPathByName(name: string, groups: UserGroup[]): string {
  const g = groups.find((x) => x.name === name);
  if (!g) return name;
  return getGroupPath(g.id, groups);
}

/** 任一子孙被选中（不含自身） */
function hasSelectedDescendant(node: GroupTreeNode, selectedIds: Set<string>): boolean {
  for (const c of node.children) {
    if (selectedIds.has(c.id)) return true;
    if (hasSelectedDescendant(c, selectedIds)) return true;
  }
  return false;
}

/**
 * 三态：本身被选=checked；祖先被选=checked；有子孙被选=indeterminate；其他=unchecked
 * 与 PlatformPolicy 同款"父覆盖子"语义
 */
function getCheckState(
  node: GroupTreeNode,
  selectedIds: Set<string>,
  groupMap: Map<string, UserGroup>
): CheckState {
  if (selectedIds.has(node.id)) return "checked";
  let cur: UserGroup | undefined = groupMap.get(node.id);
  while (cur && cur.parentId) {
    if (selectedIds.has(cur.parentId)) return "checked";
    cur = groupMap.get(cur.parentId);
  }
  if (hasSelectedDescendant(node, selectedIds)) return "indeterminate";
  return "unchecked";
}

/** 获取子孙所有 id（含自身） */
function getDescendantIds(node: GroupTreeNode): string[] {
  const ids: string[] = [node.id];
  node.children.forEach((c) => ids.push(...getDescendantIds(c)));
  return ids;
}

/**
 * 递归向上聚合：若某父节点的所有直接可用（非 disabled）子节点都已被选中，
 * 则将这些子节点 id 全部移除，换成该父节点 id。继续向上直到无法再聚合。
 */
function aggregateSelection(
  selected: Set<string>,
  roots: GroupTreeNode[],
  disabledIds: Set<string>
): Set<string> {
  const result = new Set(selected);
  let changed = true;
  while (changed) {
    changed = false;
    const walk = (node: GroupTreeNode) => {
      if (node.children.length === 0) return;
      node.children.forEach(walk);
      if (result.has(node.id)) return;
      const hasDisabled = node.children.some((c) => disabledIds.has(c.id));
      if (hasDisabled) return;
      const allSelected = node.children.every((c) => result.has(c.id));
      if (!allSelected) return;
      node.children.forEach((c) => result.delete(c.id));
      result.add(node.id);
      changed = true;
    };
    roots.forEach(walk);
  }
  return result;
}

// ─── VPC × 子网计算 ──────────────────────────────────────────────────────────

function getSubnetsByVpcZone(vpcId: string, zone: string): SubnetEntity[] {
  const mapping = VPC_ZONE_SUBNETS[vpcId];
  if (!mapping) return [];
  const subnetIds = mapping[zone] ?? [];
  const allSubnets = MOCK_SUBNETS[vpcId] ?? [];
  // 仅返回腾讯云上仍存在的"健康"子网：name/cidr 为空视为已被外部删除，不进入可选/已选编辑区。
  return subnetIds
    .map((id) => allSubnets.find((s) => s.id === id))
    .filter((s): s is SubnetEntity => !!s && !!s.name && !!s.cidr);
}

// 根据 VPC 行的当前配置，计算每个可用区实际承接的子网列表
// - 指定子网模式：按用户配置
// - 自动分配模式：展示系统实际选择的默认子网（此处取 AUTO_ASSIGNED_SUBNETS，
//   若该 VPC 不在默认分配映射中，则回退取 VPC_ZONE_SUBNETS 的第一个子网作为代表）
function getEffectiveZoneSubnets(row: VpcListItem): Record<string, SubnetEntity[]> {
  const result: Record<string, SubnetEntity[]> = {};
  if (row.subnetStrategy === "specified") {
    for (const zone of AVAILABLE_ZONES) {
      const ids = row.zoneSubnets[zone] ?? [];
      const all = MOCK_SUBNETS[row.vpcId] ?? [];
      // 找不到对应子网实体时，仍返回一个 id-only 占位（name/cidr 置空），
      // 用于在 UI 中识别为"资源已被外部删除"的异常态，避免直接吞掉条目。
      result[zone] = ids.map(
        (id): SubnetEntity =>
          all.find((s) => s.id === id) ?? { id, name: "", cidr: "", totalIp: 0, remainingIp: 0 },
      );
    }
    return result;
  }
  // auto 策略
  for (const zone of AVAILABLE_ZONES) {
    // 企业级默认 VPC 使用 AUTO_ASSIGNED_SUBNETS
    if (row.vpcId === AUTO_ASSIGNED_VPC.id) {
      const s = AUTO_ASSIGNED_SUBNETS[zone];
      result[zone] = s ? [s] : [];
    } else {
      // 其他 VPC 的 auto 策略：取 VPC_ZONE_SUBNETS 在该可用区的第一个子网作为系统默认代表
      const candidates = getSubnetsByVpcZone(row.vpcId, zone);
      result[zone] = candidates.length > 0 ? [candidates[0]] : [];
    }
  }
  return result;
}

// ─── 异常判定 helper ────────────────────────────────────────────────────────
// 统一判定逻辑：仅基于"id 存在但 name/cidr 缺失"识别已被云控制台删除的资源，
// 不引入额外 status 字段，便于后续接入真实数据。

// 子网是否已被外部删除
function isSubnetResourceDeleted(s: SubnetEntity | undefined | null): boolean {
  if (!s) return false;
  return !!s.id && (!s.name || !s.cidr);
}

// VPC 是否已被外部删除（自动分配策略不参与判定）
function isVpcResourceDeleted(row: VpcListItem): boolean {
  if (row.subnetStrategy === "auto") return false;
  return !!row.vpcId && (!row.vpcName || !row.cidr);
}

// 某可用区下原配置的子网是否「全部已被删除」（即该可用区已无任何可用子网，
// 影响实例创建）。原配置为空（未分配）不算异常。
function isZoneAllSubnetsDeleted(list: SubnetEntity[]): boolean {
  if (list.length === 0) return false;
  return list.every(isSubnetResourceDeleted);
}

// 该 VPC 行是否需要展示「⚠ 配置待更新」：
// 触发条件收敛为 ── VPC 自身被删 OR 任一可用区下"原配置全部子网均被删除"。
// 部分删除但仍至少保留 1 个可用子网，不视为异常（不影响实例创建）。
function hasVpcRowAnomaly(row: VpcListItem): boolean {
  if (isVpcResourceDeleted(row)) return true;
  const zoneSubnets = getEffectiveZoneSubnets(row);
  for (const zone of AVAILABLE_ZONES) {
    if (isZoneAllSubnetsDeleted(zoneSubnets[zone] ?? [])) return true;
  }
  return false;
}

function CreateSecurityGroupDialog({
  open,
  checkedOptions,
  draft,
  previewTab,
  onOpenChange,
  onOptionToggle,
  onNameChange,
  onRemarkChange,
  onPreviewTabChange,
  onCancel,
  onConfirm,
}: CreateSecurityGroupDialogProps) {
  const { inbound: previewInbound, outbound: previewOutbound } = buildRulesFromOptions(checkedOptions);
  const previewRules = previewTab === "inbound" ? previewInbound : previewOutbound;
  const hasRiskyRule = [previewInbound, previewOutbound].some((rules) =>
    rules.some((rule) =>
      (rule.source === "0.0.0.0/0" || rule.source === "::/0") && rule.policy === "允许"
    )
  );
  // [004] name 格式校验：仅当用户输入了内容才显示校验错误（空值不报错，由 disabled 兜底）
  const trimmedName = draft.name.trim();
  const isNameValid = trimmedName.length === 0 || isValidClawproSgName(trimmedName);
  const isCreateDisabled = trimmedName.length === 0 || !isValidClawproSgName(trimmedName);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-2xl"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
      >
        <DialogHeader>
          <DialogTitle>自定义 ClawPro 安全组</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6 flex-1">
        <div className="space-y-4">
          <div className="space-y-2">
            <MetaMedium as="label" tone="secondary">ClawPro 安全组名称<MetaText tone="danger" className="ml-0.5">*</MetaText></MetaMedium>
            <Input
              placeholder="请输入 ClawPro 安全组名称"
              value={draft.name}
              onChange={(e) => onNameChange(e.target.value)}
              className={!isNameValid ? "border-[#DC2626] focus-visible:ring-red-200" : ""}
            />
            {/* [004] 格式校验提示：非法时红色错误（只讲规则），合法时灰色 hint（规则 + 特点） */}
            {!isNameValid ? (
              <MetaText as="p" tone="danger" className="leading-relaxed">
                名称格式不正确：{CLAWPRO_SG_NAME_RULE}。
              </MetaText>
            ) : (
              <div className="space-y-0.5">
                <MetaText as="p" className="leading-relaxed">
                  {CLAWPRO_SG_NAME_RULE}
                </MetaText>
                <MetaText as="p" className="leading-relaxed">
                  {CLAWPRO_SG_NAME_NOTE}
                </MetaText>
              </div>
            )}
          </div>

          <div className="space-y-2">
            <MetaMedium as="label" tone="secondary">备注</MetaMedium>
            <Input
              placeholder="简要描述此 ClawPro 安全组用途"
              value={draft.remark}
              onChange={(e) => onRemarkChange(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <MetaMedium as="label" tone="secondary">快速添加常用规则</MetaMedium>
            <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
              {COMMON_RULE_OPTIONS.map((option) => {
                const isChecked = checkedOptions.includes(option.key);
                return (
                  <label
                    key={option.key}
                    className="inline-flex items-center gap-2 cursor-pointer select-none"
                  >
                    <Checkbox
                      checked={isChecked}
                      onCheckedChange={(v) => {
                        const checked = v === true;
                        onOptionToggle(option.key, checked);
                        if (checked) {
                          if (option.key === "allow-public") {
                            onPreviewTabChange("outbound");
                          } else if (option.key === "allow-ssh") {
                            onPreviewTabChange("inbound");
                          }
                        }
                      }}
                    />
                    <BodyText as="span" tone="primary">
                      {option.label}
                    </BodyText>
                  </label>
                );
              })}
            </div>
          </div>

          <div className="space-y-2 mt-4">
            <div className="flex items-center gap-2">
              <MetaMedium as="label" tone="secondary">规则预览</MetaMedium>
            </div>

            {hasRiskyRule && (
              <Alert variant="warning" className="px-3 py-2.5">
                <CircleAlert />
                <AlertDescription>
                  当前规则中包含来源/目标为 0.0.0.0/0 或 ::/0 的允许规则，可能带来安全风险。建议创建 Agent 云服务器后及时收紧访问范围，仅保留必要的来源或目标。
                </AlertDescription>
              </Alert>
            )}

            {checkedOptions.length === 0 && (
              <Alert variant="warning" className="px-3 py-2.5">
                <CircleAlert />
                <AlertDescription>
                  无任何规则时 Agent 将无法正常使用，请在创建后手动配置规则。至少放通一条出站规则，否则所有出站流量将被拒绝。
                </AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <SegmentGroup>
                <SegmentOption
                  active={previewTab === "outbound"}
                  onClick={() => onPreviewTabChange("outbound")}
                >
                  出站规则
                  <MetaText tone="weak" className="ml-1">({previewOutbound.length})</MetaText>
                </SegmentOption>
                <SegmentOption
                  active={previewTab === "inbound"}
                  onClick={() => onPreviewTabChange("inbound")}
                >
                  入站规则
                  <MetaText tone="weak" className="ml-1">({previewInbound.length})</MetaText>
                </SegmentOption>
              </SegmentGroup>

              <SurfaceCard className="overflow-hidden">
                <div className="max-h-40 overflow-y-auto scrollbar-on-hover">
                  <Table density="compact" autoFixedColumns={false}>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{previewTab === "outbound" ? "目标" : "来源"}</TableHead>
                        <TableHead>协议</TableHead>
                        <TableHead>端口</TableHead>
                        <TableHead>策略</TableHead>
                        <TableHead>说明</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {previewRules.length > 0 ? (
                        previewRules.map((rule, index) => (
                          <TableRow
                            key={`preview-${previewTab}-${rule.source}-${rule.port}-${index}`}
                            className="animate-in fade-in"
                          >
                            <TableCell>{rule.source}</TableCell>
                            <TableCell>{rule.protocol}</TableCell>
                            <TableCell>{rule.port}</TableCell>
                            <TableCell>
                              {rule.policy === "允许" ? (
                                <StatusTag mode="fill" variant="green">
                                  {rule.policy}
                                </StatusTag>
                              ) : (
                                <StatusTag mode="fill" variant="gray">
                                  {rule.policy}
                                </StatusTag>
                              )}
                            </TableCell>
                            <TableCell><MetaText tone="muted">{rule.remark || "—"}</MetaText></TableCell>
                          </TableRow>
                        ))
                      ) : (
                        <TableRow>
                          <TableCell colSpan={5} className="text-center py-8">
                            <BodyText as="span" tone="weak">暂无规则</BodyText>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              </SurfaceCard>
            </div>
          </div>
        </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            onClick={onConfirm}
            disabled={isCreateDisabled}
          >
            创建
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── 应用范围（与平台策略 GroupTagSelector 同款） ─────────────────────────────
//
// VPC 用户组配置的「应用范围」选择器，移植自 PlatformPolicy.tsx 的 GroupTagSelector：
//   - 行内 trigger（标签输入框风格）+ 树形 Popover；勾选即生效，无确认按钮
//   - 数据契约：外部 value 仍是「组织 name 数组」（与 vpcList[].associatedGroups 一致），
//     不引入 id 字段；组件内部 name ↔ id 自行换算
//   - 仅展示两个桶：部门（oneid-dept） / 自定义组织（manual），屏蔽用户组桶（oneid-group）
//   - 业务约束：
//       a. 兜底/预设策略行（type === "enterprise"）通过列表行的紫色徽章独立呈现，
//          不进入选择器，因此本组件不再需要「全部用户」radio 切换
//       b. disabledIds：已被其他组织策略占用的组织 id，置灰展示 + Tooltip 提示
//       c. 当前编辑行已绑定的组织不会进入 disabledIds（编辑时可保留 / 可取消）

// 选择框内只展示部门和自定义组织（不含用户组）
const SOURCE_ORDER: GroupSource[] = ["oneid-dept", "manual"];
const SOURCE_LABELS: Record<GroupSource, string> = {
  "oneid-dept": "部门",
  "oneid-group": "用户组",
  "manual": "自定义组织",
  "project": "项目",
};

function GroupTagSelector({
  value,
  onChange,
  disabledIds,
  disabledTooltip,
}: {
  value: string[]; // 已选组织 name 数组
  onChange: (next: string[]) => void;
  disabledIds?: Set<string>;
  disabledTooltip?: string;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [hover, setHover] = useState(false);

  // 仅保留部门和自定义组织（不含用户组），与 PlatformPolicy 一致
  const visibleGroups = useMemo(
    () => ALL_GROUPS_SHARED.filter((g) => SOURCE_ORDER.includes(g.source)),
    []
  );
  const groupMap = useMemo(
    () => new Map(visibleGroups.map((g) => [g.id, g])),
    [visibleGroups]
  );

  // name ↔ id 映射（外部契约 name；内部操作 id）
  const nameToId = useMemo(() => {
    const m = new Map<string, string>();
    visibleGroups.forEach((g) => m.set(g.name, g.id));
    return m;
  }, [visibleGroups]);
  const valueIds = useMemo(() => {
    const ids: string[] = [];
    value.forEach((n) => {
      const id = nameToId.get(n);
      if (id) ids.push(id);
    });
    return ids;
  }, [value, nameToId]);

  const disabledSet = useMemo(() => disabledIds ?? new Set<string>(), [disabledIds]);

  // ── 三态切换（idle / editing / filled） ──
  // mode === "idle"     ：空态初始 → 显示"+ 添加组织策略"按钮
  // mode === "editing"  ：编辑中 → 输入框 trigger（点击展开 Popover）+ ❌ ✅
  //                       期间所有勾选 / 删除 tag 仅修改 draftIds，不触达 onChange
  // mode === "filled"   ：有值只读态 → "研发组 …共 N 个组织 ✏️"
  // 状态转移：
  //   idle    ─ 点 "+ 添加组织策略" ─→ editing(draft=[])
  //   editing ─ 点 ❌ ─────────────→ idle (valueIds 仍为 []) 或 filled (恢复原值)
  //   editing ─ 点 ✅ ─────────────→ filled (onChange 提交 draft)
  //   filled  ─ 点 ✏️ ─────────────→ editing(draft=valueIds)
  const [draftIds, setDraftIds] = useState<string[] | null>(null); // null = 非编辑态
  const isEditing = draftIds !== null;
  // editing 态下用 draft，否则用外部 value
  const effectiveIds = isEditing ? draftIds : valueIds;

  // 按 source 分桶 + 建树
  const groupsBySource = useMemo(() => {
    const buckets: Record<string, UserGroup[]> = { "oneid-dept": [], manual: [] };
    visibleGroups.forEach((g) => { if (buckets[g.source]) buckets[g.source].push(g); });
    return buckets;
  }, [visibleGroups]);
  const activeSources = useMemo(
    () => SOURCE_ORDER.filter((s) => (groupsBySource[s] || []).length > 0),
    [groupsBySource]
  );
  const treesMap = useMemo(() => {
    const map: Record<string, GroupTreeNode[]> = {};
    activeSources.forEach((s) => { map[s] = buildGroupTree(groupsBySource[s] || []); });
    return map;
  }, [activeSources, groupsBySource]);

  const allRoots = useMemo(
    () => activeSources.flatMap((s) => treesMap[s] || []),
    [activeSources, treesMap]
  );

  // 打开时：默认展开已选祖先 + 根节点
  const handleOpenChange = (v: boolean) => {
    if (v) {
      setSearch("");
      const expandSet = new Set<string>();
      effectiveIds.forEach((gid) => {
        let cur = groupMap.get(gid);
        while (cur && cur.parentId) {
          expandSet.add(cur.parentId);
          cur = groupMap.get(cur.parentId);
        }
      });
      activeSources.forEach((s) => {
        treesMap[s]?.forEach((root) => expandSet.add(root.id));
      });
      setExpanded(expandSet);
    }
    setOpen(v);
  };

  const toggleExpand = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  // 在所有可见树中查找节点
  const findTreeNode = (id: string): GroupTreeNode | null => {
    const walk = (nodes: GroupTreeNode[]): GroupTreeNode | null => {
      for (const n of nodes) {
        if (n.id === id) return n;
        const found = walk(n.children);
        if (found) return found;
      }
      return null;
    };
    return walk(allRoots);
  };

  // 写入：editing 时只改本地 draft；filled/idle 时直接 onChange
  const writeIds = (ids: Iterable<string>) => {
    const arr = Array.from(ids);
    if (isEditing) {
      setDraftIds(arr);
    } else {
      const names: string[] = [];
      arr.forEach((gid) => {
        const node = groupMap.get(gid);
        if (node) names.push(node.name);
      });
      onChange(names);
    }
  };

  // 进入编辑态：缓存当前值到 draft，并展开 Popover
  const enterEditing = () => {
    setDraftIds([...valueIds]);
    handleOpenChange(true);
  };

  // 取消编辑：丢弃 draft，关 Popover；valueIds 不变
  const cancelEditing = () => {
    setDraftIds(null);
    setOpen(false);
  };

  // 保存编辑：把 draft 提交到 onChange，关 Popover，回到 filled
  const saveEditing = () => {
    if (!draftIds || draftIds.length === 0) return;
    const names: string[] = [];
    draftIds.forEach((gid) => {
      const node = groupMap.get(gid);
      if (node) names.push(node.name);
    });
    onChange(names);
    setDraftIds(null);
    setOpen(false);
  };

  /**
   * 点击节点（与 PlatformPolicy GroupTagSelector 同款语义）：
   * - 本身已选 → 移除
   * - 祖先已选（自动 checked）→ 展开祖先为除当前路径外的兄弟子节点
   * - 半选（indeterminate）→ 清空所有子孙，加入自身
   * - 未选 → 加入自身
   * 最后执行递归向上聚合（兄弟全选时合并到父级）
   */
  const toggleNode = (node: GroupTreeNode) => {
    if (disabledSet.has(node.id)) return;
    const ids = new Set(effectiveIds);

    let ancestorSelectedId: string | null = null;
    let cur: UserGroup | undefined = groupMap.get(node.id);
    while (cur && cur.parentId) {
      if (ids.has(cur.parentId)) { ancestorSelectedId = cur.parentId; break; }
      cur = groupMap.get(cur.parentId);
    }

    if (ids.has(node.id)) {
      ids.delete(node.id);
    } else if (ancestorSelectedId) {
      // 展开祖先：移除祖先 → 沿祖先到 node 的路径，把每层"非下一跳"的子节点加入
      ids.delete(ancestorSelectedId);
      const pathNodes: UserGroup[] = [];
      let p: UserGroup | undefined = groupMap.get(node.id);
      while (p && p.id !== ancestorSelectedId) {
        pathNodes.push(p);
        p = p.parentId ? groupMap.get(p.parentId) : undefined;
      }
      pathNodes.reverse();
      let cursor = findTreeNode(ancestorSelectedId);
      for (let i = 0; i < pathNodes.length; i++) {
        const nextHopId = pathNodes[i].id;
        if (!cursor) break;
        cursor.children.forEach((c) => {
          if (c.id !== nextHopId && !disabledSet.has(c.id)) ids.add(c.id);
        });
        cursor = cursor.children.find((c) => c.id === nextHopId) || null;
      }
    } else {
      const state = hasSelectedDescendant(node, ids) ? "indeterminate" : "unchecked";
      if (state === "indeterminate") {
        getDescendantIds(node).forEach((d) => ids.delete(d));
      }
      ids.add(node.id);
    }

    const aggregated = aggregateSelection(ids, allRoots, disabledSet);
    writeIds(aggregated);
  };

  // 搜索过滤
  const matchedIds = useMemo(() => {
    if (!search.trim()) return null;
    const q = search.toLowerCase();
    return new Set(
      visibleGroups
        .filter((g) => g.name.toLowerCase().includes(q) || getGroupPath(g.id, visibleGroups).toLowerCase().includes(q))
        .map((g) => g.id)
    );
  }, [search, visibleGroups]);
  const isVisible = (node: GroupTreeNode): boolean => {
    if (!matchedIds) return true;
    if (matchedIds.has(node.id)) return true;
    return node.children.some(isVisible);
  };

  const renderNode = (node: GroupTreeNode, depth: number) => {
    if (!isVisible(node)) return null;
    const checkState = getCheckState(node, new Set(effectiveIds), groupMap);
    const isExpanded = expanded.has(node.id);
    const hasChildren = node.children.length > 0;
    const isDisabled = disabledSet.has(node.id);

    const nameSpan = <BodyText as="span" tone="secondary" className="truncate text-left">{node.name}</BodyText>;

    return (
      <div key={node.id}>
        <button
          type="button"
          onClick={() => !isDisabled && toggleNode(node)}
          disabled={isDisabled}
          className={`w-full flex items-center gap-2 h-8 px-3 rounded-[6px] transition-colors text-left ${isDisabled ? "opacity-40 cursor-not-allowed" : checkState === "checked" ? "bg-[var(--bg-brand-selected)]" : "hover:bg-[var(--bg-grey-hover)]"}`}
          style={{ paddingLeft: 8 + depth * 16 }}
        >
          {hasChildren ? (
            <span
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.id); }}
              className="w-4 h-4 flex items-center justify-center text-[var(--text-muted)] hover:text-[var(--text-secondary)] shrink-0 cursor-pointer"
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </span>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <Checkbox
            checked={checkState === "checked" ? true : checkState === "indeterminate" ? "indeterminate" : false}
            className="pointer-events-none"
          />
          {isDisabled && disabledTooltip ? (
            <Tooltip>
              <TooltipTrigger asChild>{nameSpan}</TooltipTrigger>
              <TooltipContent side="right" sideOffset={8} className="text-xs max-w-[260px] leading-relaxed">
                {disabledTooltip}
              </TooltipContent>
            </Tooltip>
          ) : (
            nameSpan
          )}
        </button>
        {hasChildren && isExpanded && node.children.map((c) => renderNode(c, depth + 1))}
      </div>
    );
  };

  return (
    <TooltipProvider>
      {isEditing ? (
        <div className="flex items-center gap-1.5 w-full">
          <div className="flex-1 min-w-0">
            <Popover
              open={open}
              onOpenChange={(v) => {
                handleOpenChange(v);
                // 编辑态下 Popover 关闭不退出编辑（用户可能只是收起列表整理 tag），
                // 退出由 ❌/✅ 显式触发
              }}
            >
              <PopoverTrigger asChild>
                <div
                  onMouseEnter={() => setHover(true)}
                  onMouseLeave={() => setHover(false)}
                  className="relative w-full min-h-7 px-2 py-1 rounded-[4px] border border-border bg-white hover:border-[#355EF1] transition-colors cursor-pointer flex items-center flex-wrap gap-1 pr-7"
                >
                  {effectiveIds.length === 0 ? (
                    <MetaText as="span" tone="weak" className="px-1">选择组织…</MetaText>
                  ) : (
                    effectiveIds.map((id) => {
                      const path = getGroupPath(id, ALL_GROUPS_SHARED);
                      return (
                        <span
                          key={id}
                          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-[#eff4ff] text-[var(--text-brand)] border border-blue-100 max-w-full"
                        >
                          <span className="truncate">{path}</span>
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              const next = new Set(effectiveIds);
                              next.delete(id);
                              writeIds(next);
                            }}
                            className="text-blue-400 hover:text-blue-700 shrink-0"
                          >
                            <X className="w-3 h-3" />
                          </button>
                        </span>
                      );
                    })
                  )}
                  {hover && effectiveIds.length > 0 && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        writeIds([]);
                      }}
                      className="absolute right-1.5 top-1/2 -translate-y-1/2 w-4 h-4 rounded-full bg-gray-300 hover:bg-gray-400 flex items-center justify-center shrink-0"
                      title="清空"
                    >
                      <X className="w-2.5 h-2.5 text-white" />
                    </button>
                  )}
                </div>
              </PopoverTrigger>
              <PopoverContent
                className="p-0 rounded-[4px] border border-[var(--border)] shadow-md"
                style={{ width: "var(--radix-popover-trigger-width)" }}
                align="start"
                sideOffset={4}
              >
                <div className="p-2.5 border-b border-[var(--border)]">
                  <div className="relative">
                    <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)] pointer-events-none" />
                    <Input
                      type="text"
                      placeholder="搜索组织"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      className="h-8 pl-8 pr-7 text-xs"
                    />
                    {search && (
                      <button onClick={() => setSearch("")} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#A3A3A3] hover:text-[#737373]">
                        <X className="w-3 h-3" />
                      </button>
                    )}
                  </div>
                </div>
                <div className="max-h-[280px] overflow-y-auto p-1.5">
                  {activeSources.length === 0 ? (
                    <p className="text-[11px] text-[var(--text-weak)] text-center py-4">暂无组织</p>
                  ) : (
                    activeSources.map((source) => {
                      const trees = treesMap[source] || [];
                      const hasVisibleTrees = trees.some(isVisible);
                      if (!hasVisibleTrees) return null;
                      return (
                        <div key={source} className="mb-1.5 last:mb-0">
                          <div className="px-2 pt-1.5 pb-1 text-[10px] font-medium text-[var(--text-weak)] uppercase tracking-wide">{SOURCE_LABELS[source]}</div>
                          {trees.map((root) => renderNode(root, 0))}
                        </div>
                      );
                    })
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </div>
          {/* ❌ 取消 / ✅ 保存（图标按钮：取消=描边淡灰，保存=实心蓝主操作，禁用态淡灰） */}
          <button
            type="button"
            onClick={cancelEditing}
            className="shrink-0 w-7 h-7 inline-flex items-center justify-center rounded-[4px] bg-white border border-border text-[var(--text-muted)] hover:bg-[#f5f5f5] hover:text-[var(--text-secondary)] transition-colors"
            title="取消"
          >
            <X className="w-3.5 h-3.5" />
          </button>
          <Button
            type="button"
            onClick={saveEditing}
            disabled={!draftIds || draftIds.length === 0}
            size="sm"
            className="shrink-0 w-7 h-7 p-0"
            title={!draftIds || draftIds.length === 0 ? "请至少选择一个组织" : "保存"}
          >
            <Check className="w-3.5 h-3.5" />
          </Button>
        </div>
      ) : valueIds.length === 0 ? (
        // idle 空态：显示"+ 添加组织策略"按钮
        <button
          type="button"
          onClick={enterEditing}
          className="inline-flex items-center gap-1 text-xs text-[var(--text-brand)] hover:text-blue-700 transition-colors"
        >
          <Plus className="w-3.5 h-3.5" />
          添加组织
        </button>
      ) : (
        // filled 有值只读态：首条徽章（截断）+ …共 N 个组织 + 铅笔；hover Tooltip 列出全部
        <div className="inline-flex items-center gap-1.5 min-h-7">
          {(() => {
            const firstPath = getGroupPath(valueIds[0], ALL_GROUPS_SHARED);
            const rest = valueIds.length - 1;
            const allPaths = valueIds.map((gid) => getGroupPath(gid, ALL_GROUPS_SHARED));
            const badge = (
              <span className="inline-flex items-center gap-1 cursor-default align-middle">
                <span className="inline-flex items-center max-w-[180px] px-2 py-0.5 rounded-full text-xs font-medium bg-[#eff4ff] text-[var(--text-brand)] border border-blue-100 whitespace-nowrap overflow-hidden">
                  <span className="truncate">{firstPath}</span>
                </span>
                {rest > 0 && (
                  <span className="text-[11px] text-[var(--text-muted)] whitespace-nowrap shrink-0">
                    …共 {valueIds.length} 个组织
                  </span>
                )}
              </span>
            );
            // 单组织路径过长 / 多组织都需要 Tooltip 显示完整内容
            return (
              <Tooltip>
                <TooltipTrigger asChild>{badge}</TooltipTrigger>
                <TooltipContent side="top" className="max-w-[360px] text-xs leading-relaxed">
                  <div className="space-y-0.5">
                    {allPaths.map((p, i) => <div key={i}>{p}</div>)}
                  </div>
                </TooltipContent>
              </Tooltip>
            );
          })()}
          <button
            type="button"
            onClick={enterEditing}
            className="text-[var(--text-weak)] hover:text-blue-500 transition-colors shrink-0"
            title="编辑应用范围"
          >
            <Pencil className="w-3 h-3" />
          </button>
        </div>
      )}
    </TooltipProvider>
  );
}

// ─── 列表行只读徽章（与 PlatformPolicy GroupBadges 同款：单行 + 折叠） ────────
//
// 输入是组织 name 数组（保持与 vpcList[].associatedGroups 一致）。
// 渲染：可见容器内逐个测量组织路径徽章宽度，放不下时截断并显示「…共 N 个组织」，
//       Tooltip 中以多行完整路径展示全部。
function GroupBadges({ groupNames }: { groupNames: string[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const tagRefs = useRef<(HTMLSpanElement | null)[]>([]);
  const moreRef = useRef<HTMLSpanElement>(null);
  const [visibleCount, setVisibleCount] = useState(groupNames.length);

  const paths = useMemo(
    () => groupNames.map((n) => getGroupPathByName(n, ALL_GROUPS_SHARED)),
    [groupNames]
  );

  useLayoutEffect(() => {
    if (groupNames.length === 0) return;
    const container = containerRef.current;
    if (!container) return;

    const computeVisible = () => {
      const available = container.clientWidth;
      if (available <= 0) return;

      const gap = 4; // gap-1
      let totalW = 0;
      let fitCount = 0;
      for (let i = 0; i < paths.length; i++) {
        const el = tagRefs.current[i];
        if (!el) break;
        const w = el.offsetWidth;
        const add = totalW === 0 ? w : w + gap;
        if (totalW + add <= available) {
          totalW += add;
          fitCount = i + 1;
        } else {
          break;
        }
      }

      if (fitCount === paths.length) {
        setVisibleCount(paths.length);
        return;
      }

      const moreEl = moreRef.current;
      if (!moreEl) {
        setVisibleCount(Math.max(1, fitCount));
        return;
      }
      for (let n = fitCount; n >= 1; n--) {
        let w = 0;
        for (let i = 0; i < n; i++) {
          const el = tagRefs.current[i];
          if (!el) continue;
          w += el.offsetWidth + (i === 0 ? 0 : gap);
        }
        moreEl.textContent = `…共 ${paths.length} 个组织`;
        const moreW = moreEl.offsetWidth;
        if (w + gap + moreW <= available) {
          setVisibleCount(n);
          return;
        }
      }
      setVisibleCount(1);
    };

    computeVisible();
    const observer = new ResizeObserver(computeVisible);
    observer.observe(container);
    return () => observer.disconnect();
  }, [paths, groupNames.length]);

  if (groupNames.length === 0) {
    return <MetaText as="span" tone="weak">—</MetaText>;
  }

  const omitted = paths.length - visibleCount;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div
          ref={containerRef}
          className="flex items-center gap-1 w-full max-w-[220px] overflow-hidden cursor-default"
        >
          {paths.slice(0, visibleCount).map((p, i) => (
            <StatusTag mode="fill"
              key={i}
              ref={(el) => { tagRefs.current[i] = el as HTMLSpanElement | null; }}
              variant="gray"
              className="shrink-0"
            >
              {p}
            </StatusTag>
          ))}
          {omitted > 0 && (
            <span className="inline-flex items-center px-1.5 py-0.5 text-[11px] text-[var(--text-muted)] whitespace-nowrap shrink-0">
              …共 {paths.length} 个组织
            </span>
          )}
          {/* 隐藏测量区 */}
          <div aria-hidden="true" className="absolute invisible pointer-events-none whitespace-nowrap" style={{ left: -99999, top: -99999 }}>
            {paths.map((p, i) => (
              <StatusTag mode="fill"
                key={`m-${i}`}
                ref={(el) => { tagRefs.current[i] = el as HTMLSpanElement | null; }}
                variant="gray"
              >
                {p}
              </StatusTag>
            ))}
            <span
              ref={moreRef}
              className="inline-flex items-center px-1.5 py-0.5 text-[11px] text-[var(--text-muted)] whitespace-nowrap"
            />
          </div>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-[360px] text-xs leading-relaxed">
        <div className="space-y-0.5">
          {paths.map((p, i) => <div key={i}>{p}</div>)}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

// ─── 子网胶囊行（列表展开"子网配置明细"，每个可用区一行） ──────────────────
//
// 视觉规则（信息分层 + 降噪）：
//   · 主信息：正常子网灰色胶囊（id | name | cidr）
//   · 次级信息：已删除子网统一收进 1 个浅灰胶囊
//     「已删除子网：subnet-a、subnet-b、…」；不带 icon、不带 warning 色
//   · 全删场景：单行展示「无可用子网」+ 已删除胶囊（紧凑表达"状态 + 原因"）
//   · 「⚠ 配置待更新」不在子网明细中重复，仅由列表策略行（VPC 主行）展示一次
function SubnetBadgesRow({ subnets }: { subnets: SubnetEntity[] }) {
  const allDeleted = isZoneAllSubnetsDeleted(subnets);
  const healthy = useMemo(() => subnets.filter((s) => !isSubnetResourceDeleted(s)), [subnets]);
  const deleted = useMemo(() => subnets.filter(isSubnetResourceDeleted), [subnets]);
  const containerRef = useRef<HTMLDivElement>(null);
  const tagRefs = useRef<(HTMLSpanElement | null)[]>([]);
  const moreRef = useRef<HTMLSpanElement>(null);
  const [visibleCount, setVisibleCount] = useState(healthy.length);

  useLayoutEffect(() => {
    if (healthy.length === 0) {
      setVisibleCount(0);
      return;
    }
    const container = containerRef.current;
    if (!container) return;

    const computeVisible = () => {
      const available = container.clientWidth;
      if (available <= 0) return;
      const gap = 6;
      let totalW = 0;
      let fitCount = 0;
      for (let i = 0; i < healthy.length; i++) {
        const el = tagRefs.current[i];
        if (!el) break;
        const w = el.offsetWidth;
        const add = totalW === 0 ? w : w + gap;
        if (totalW + add <= available) {
          totalW += add;
          fitCount = i + 1;
        } else {
          break;
        }
      }
      if (fitCount === healthy.length) {
        setVisibleCount(healthy.length);
        return;
      }
      const moreEl = moreRef.current;
      if (!moreEl) {
        setVisibleCount(Math.max(1, fitCount));
        return;
      }
      for (let n = fitCount; n >= 1; n--) {
        let w = 0;
        for (let i = 0; i < n; i++) {
          const el = tagRefs.current[i];
          if (!el) continue;
          w += el.offsetWidth + (i === 0 ? 0 : gap);
        }
        moreEl.textContent = `…共 ${healthy.length} 个可用子网`;
        const moreW = moreEl.offsetWidth;
        if (w + gap + moreW <= available) {
          setVisibleCount(n);
          return;
        }
      }
      setVisibleCount(1);
    };

    computeVisible();
    const observer = new ResizeObserver(computeVisible);
    observer.observe(container);
    return () => observer.disconnect();
  }, [healthy]);

  const omitted = healthy.length - visibleCount;

  // 已删除子网聚合行（全删 / 部分删 共用）
  // 结构：subnetId 在前（资源主体）、状态说明在后（轻量 warning 文案）
  // 颜色：全删（影响实例创建）→ 弱橙色 amber-600，与 VPC 行「配置待更新」呼应；
  //       部分删（仍有可用子网）→ 中性灰色，避免对仍可工作的可用区做无谓告警。
  const tone = allDeleted ? "text-amber-600" : "text-[var(--text-weak)]";
  const sepTone = allDeleted ? "text-amber-300" : "text-[var(--text-weak)]";
  const deletedPill = deleted.length > 0 && (
    <CompactText as="span" tone="inherit" className={`inline-flex items-center gap-2 ${tone} max-w-full break-all`}>
      <CompactText as="span" tone="inherit" className="min-w-0">
        {deleted.map((s, i) => (
          <Fragment key={s.id}>
            {i > 0 && <CompactText as="span" tone="inherit" className={sepTone}>、</CompactText>}
            <CompactText as="span" tone="inherit" className={tone}>{s.id}</CompactText>
          </Fragment>
        ))}
      </CompactText>
      <CompactText as="span" tone="inherit" className="shrink-0">已从腾讯云控制台被删除</CompactText>
    </CompactText>
  );

  // 全删：单行表达「状态 + 原因」
  if (allDeleted) {
    return (
      <div className="flex flex-wrap items-center gap-2 flex-1 min-w-0 cursor-default">
        <CompactText tone="weak" className="shrink-0">无可用子网</CompactText>
        {deletedPill}
      </div>
    );
  }

  // 部分删 / 全部正常
  return (
    <div className="flex flex-col gap-1 flex-1 min-w-0 cursor-default">
      {healthy.length > 0 && (
        <div ref={containerRef} className="flex items-center gap-1.5 min-w-0 overflow-hidden">
          {healthy.slice(0, visibleCount).map((s, i) => (
            <span
              key={s.id}
              ref={(el) => { tagRefs.current[i] = el; }}
              className="shrink-0"
            >
              <Badge variant="outline">
                {s.id} | {s.name} | {s.cidr}
              </Badge>
            </span>
          ))}
          {/* 折叠提示 */}
          {omitted > 0 && (
            <CompactText tone="weak" className="inline-flex items-center px-1.5 py-0.5 whitespace-nowrap shrink-0">
              …共 {healthy.length} 个可用子网
            </CompactText>
          )}
          {/* 隐藏测量区 */}
          <div aria-hidden="true" className="absolute invisible pointer-events-none whitespace-nowrap" style={{ left: -99999, top: -99999 }}>
            {healthy.map((s, i) => (
              <span
                key={`m-${s.id}`}
                ref={(el) => { tagRefs.current[i] = el; }}
                className="inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs whitespace-nowrap"
              >
                {s.id} | {s.name} | {s.cidr}
              </span>
            ))}
            <span ref={moreRef} className="inline-flex items-center px-1.5 py-0.5 text-xs text-[var(--text-weak)] whitespace-nowrap" />
          </div>
        </div>
      )}
      {deletedPill && <div className="min-w-0">{deletedPill}</div>}
    </div>
  );
}

export default function SecurityGroupManagement() {
  const { isAdminDisabled } = useAdminDisabled();
  const [activeTab, setActiveTab] = useState(() => {
    const params = new URLSearchParams(window.location.search);
    const tab = params.get("tab");
    if (tab && TABS.some((t) => t.id === tab)) return tab;
    return "vpc";
  });
  const initialDefaultSecurityGroup = getInitialDefaultSecurityGroup();

  // 安全组状态
  const [inboundRules, setInboundRules] = useState<Rule[]>(() => markPlatformRequiredRules(initialDefaultSecurityGroup?.inboundRules ?? []));
  const [outboundRules, setOutboundRules] = useState<Rule[]>(initialDefaultSecurityGroup?.outboundRules ?? []);
  const [securityTab, setSecurityTab] = useState<"outbound" | "inbound">("outbound");

  // [004] 独立化升级状态（存量/新建企业判断 + 蓝条 ack 状态）
  //   - migrationStatus.isLegacyMigrated：当前是否为存量企业（决定是否显示迁移黄条）
  //   - setMigrationStatus 由 handleAckMigration 调用以同步 localStorage 与会话内存模型
  const [migrationStatus, setMigrationStatus] = useState<MigrationStatus>(getInitialMigrationStatus);
  // 关闭蓝条（模拟后端 POST /admin/config/security-group/migration-ack）
  const handleAckMigration = () => {
    setMigrationStatus((prev) => ({ ...prev, ackedByCurrentAdmin: true }));
    if (typeof window !== "undefined") {
      window.localStorage.setItem(MIGRATION_ACK_STORAGE_KEY, "true");
    }
  };

  // 用户端访问 Agent 面板开关 - 持久化到 localStorage（潜伏能力，PR #169 暂未暴露入口）
  const [allowPanelAccess] = useState(() => {
    return localStorage.getItem("admin_allow_panel_access") === "true";
  });
  const [panelPort] = useState<string | null>(() => {
    return localStorage.getItem("admin_panel_port");
  });

  // ── VPC 列表管理状态 ──
  const [vpcList, setVpcList] = useState<VpcListItem[]>(getInitialVpcList);
  const [showEditVpcDialog, setShowEditVpcDialog] = useState<VpcListItem | null>(null);
  const [showDeleteVpcDialog, setShowDeleteVpcDialog] = useState<VpcListItem | null>(null);
  // 保存确认弹窗：把"已通过表单校验、待落库"的动作挂起，展示二次确认。
  // mode 决定弹窗文案分支（newGroup / editGlobal / editGroup）。
  const [pendingVpcSave, setPendingVpcSave] = useState<{
    mode: "newGroup" | "editGlobal" | "editGroup";
    execute: () => void;
  } | null>(null);
  const [editVpcDraft, setEditVpcDraft] = useState<{
    vpcId: string;
    subnetStrategy: "auto" | "specified";
    zoneSubnets: Record<string, string[]>;
    associatedGroups: string[];
  }>({ vpcId: "", subnetStrategy: "auto", zoneSubnets: {}, associatedGroups: [] });
  // 每个可用区下「添加子网」浮层开关
  const [zoneSubnetPickerOpen, setZoneSubnetPickerOpen] = useState<Record<string, boolean>>({});
  // 编辑弹窗中「VPC」选择器浮层开关
  const [editVpcPickerOpen, setEditVpcPickerOpen] = useState(false);
  const [vpcSearchQuery, setVpcSearchQuery] = useState("");
  const [subnetSearchQueries, setSubnetSearchQueries] = useState<Record<string, string>>({});
  // 进入编辑弹窗时若自动清理过失效资源，记录被删除的具体资源 ID 用于在 DialogHeader 顶部提示文案：
  // - vpcId   : 被删除的 VPC ID（仅 VPC 已删时填）
  // - subnetIds: 被删除的子网 ID 列表（VPC 仍可用时填）
  // - null    : 无清理动作
  const [editAutoCleaned, setEditAutoCleaned] = useState<
    null | { vpcId: string | null; subnetIds: string[] }
  >(null);
  // 列表中展开查看子网配置详情的 VPC id 集合
  const [expandedVpcIds, setExpandedVpcIds] = useState<Set<string>>(() => new Set());
  const toggleVpcExpanded = (id: string) => {
    setExpandedVpcIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // 迁移状态
  type MigrationBusinessState = "idle" | "running" | "success" | "failed";
  type MigrationViewState = "hidden" | "confirm" | "execution" | "result";
  type ConfirmMigrationTab = "migratable" | "blocked";
  type MigrationResultTab = "success" | "failed";
  type InstanceTaskStatus = "待迁移" | "迁移中" | "迁移成功" | "迁移失败";
  type MigrationInstance = typeof MIGRATION_INSTANCES[number];
  type InstanceTask = {
    user: string;
    instance: string;
    currentNet: string;
    targetNet: string;
    status: InstanceTaskStatus;
    failReason?: string;
  };

  const migrationDialogTitle = "Agent 实例网络迁移";
  const migrationFlowSteps = ["确认迁移范围", "执行迁移", "查看迁移结果"] as const;
  const confirmTableColumns = ["用户名", "Agent 实例", "当前网络", "目标网络", "说明"] as const;
  const executionTableColumns = ["用户名", "Agent 实例", "迁移任务状态"] as const;
  const resultTableColumns = ["用户名", "Agent 实例", "当前网络", "迁移任务状态", "说明"] as const;
  const confirmTableGridClass = "grid grid-cols-[1.3fr_1.4fr_1.7fr_1.7fr_1.8fr]";
  const executionTableGridClass = "grid grid-cols-[1.5fr_1.5fr_1fr]";
  const resultTableGridClass = "grid grid-cols-[1.3fr_1.4fr_1.7fr_1fr_2fr]";
  const migratableInstances = MIGRATION_INSTANCES.filter((item) => item.note === "可迁移");
  const blockedInstances = MIGRATION_INSTANCES.filter((item) => item.note !== "可迁移");
  const buildMigrationTasks = (items: MigrationInstance[] = migratableInstances): InstanceTask[] => (
    items.map((item) => ({
      user: item.user,
      instance: item.instance,
      currentNet: item.currentNet,
      targetNet: item.targetNet,
      status: "待迁移"
    }))
  );

  const [migrationBusinessState, setMigrationBusinessState] = useState<MigrationBusinessState>("idle");
  const [migrationViewState, setMigrationViewState] = useState<MigrationViewState>("hidden");
  const [confirmMigrationTab, setConfirmMigrationTab] = useState<ConfirmMigrationTab>("migratable");
  const [confirmTableExpanded, setConfirmTableExpanded] = useState(false);
  const [migrationResultTab, setMigrationResultTab] = useState<MigrationResultTab>("success");
  const [currentMigrationRunMode, setCurrentMigrationRunMode] = useState<"all" | "retryFailed" | null>(null);
  const [resolvedFailureInstances, setResolvedFailureInstances] = useState<string[]>([]);
  const [instanceTasks, setInstanceTasks] = useState<InstanceTask[]>(() => buildMigrationTasks());
  const [activeMigrationInstanceNames, setActiveMigrationInstanceNames] = useState<string[]>(() => migratableInstances.map((item) => item.instance));

  const activeConfirmInstances = confirmMigrationTab === "migratable" ? migratableInstances : blockedInstances;
  const migrationImpactSummary = {
    instanceCount: MIGRATION_INSTANCES.length,
    userCount: new Set(MIGRATION_INSTANCES.map((item) => item.user)).size
  };
  const activeMigrationInstanceNameSet = new Set(activeMigrationInstanceNames);
  const activeMigrationTasks = instanceTasks.filter((task) => activeMigrationInstanceNameSet.has(task.instance));
  const failedMigrationTasks = instanceTasks.filter((task) => task.status === "迁移失败");
  const currentRunSuccessfulMigrationTasks = activeMigrationTasks.filter((task) => task.status === "迁移成功");
  const currentRunFailedMigrationTasks = activeMigrationTasks.filter((task) => task.status === "迁移失败");
  const currentRunHasRunningTasks = activeMigrationTasks.some((task) => task.status === "待迁移" || task.status === "迁移中");
  const isCurrentRunCompleted = activeMigrationTasks.length > 0 && !currentRunHasRunningTasks;
  const isRetryFailedRun = currentMigrationRunMode === "retryFailed";
  const isRetryRunCompletedWithAllSuccess = isCurrentRunCompleted && isRetryFailedRun && currentRunFailedMigrationTasks.length === 0;
  const hasRunningTasks = instanceTasks.some((task) => task.status === "待迁移" || task.status === "迁移中");
  const hasFailedTasks = failedMigrationTasks.length > 0;
  // 仅"存量企业"才可能显示迁移黄条；迁移完成（success）后也隐藏
  const shouldShowMigrationBanner = migrationStatus.isLegacyMigrated && migrationBusinessState !== "success";
  const migrationBannerState: "idle" | "running" | "failed" = migrationBusinessState === "running"
    ? "running"
    : migrationBusinessState === "failed"
      ? "failed"
      : "idle";
  const isMigrationConfirmOpen = migrationViewState === "confirm";
  const isMigrationProgressOpen = migrationViewState === "execution" || migrationViewState === "result";
  const isViewingMigrationResult = migrationViewState === "result";
  const resultTabs = migrationBusinessState === "failed" ? (["success", "failed"] as const) : (["success"] as const);
  const activeResultTasks = migrationResultTab === "failed" ? currentRunFailedMigrationTasks : currentRunSuccessfulMigrationTasks;

  const closeMigrationDialog = () => {
    setMigrationViewState("hidden");
  };

  const openMigrationConfirmDialog = () => {
    setConfirmMigrationTab("migratable");
    setConfirmTableExpanded(false);
    setMigrationViewState("confirm");
  };

  const openMigrationProgressDialog = () => {
    setMigrationViewState("execution");
  };

  const openMigrationResultDialog = (tab: MigrationResultTab = migrationBusinessState === "failed" ? "failed" : "success") => {
    setMigrationResultTab(tab);
    setMigrationViewState("result");
  };

  const startMigrationRun = (mode: "all" | "retryFailed") => {
    const targetInstances = mode === "retryFailed"
      ? migratableInstances.filter((item) => failedMigrationTasks.some((task) => task.instance === item.instance))
      : migratableInstances;

    if (targetInstances.length === 0) {
      toast.error(mode === "retryFailed" ? "暂无可重试的失败实例" : "暂无可迁移实例");
      return;
    }

    const activeInstanceNames = targetInstances.map((item) => item.instance);
    const retryingInstanceNames = mode === "retryFailed" ? activeInstanceNames : [];
    const resolvedInstancesSnapshot = new Set(mode === "retryFailed" ? [...resolvedFailureInstances, ...retryingInstanceNames] : []);

    setActiveMigrationInstanceNames(activeInstanceNames);
    setCurrentMigrationRunMode(mode);
    setMigrationBusinessState("running");
    setMigrationViewState("execution");
    setMigrationResultTab("success");

    if (mode === "retryFailed") {
      setResolvedFailureInstances((prev) => Array.from(new Set([...prev, ...retryingInstanceNames])));
    } else {
      setResolvedFailureInstances([]);
    }

    setInstanceTasks((prev) => {
      if (mode === "retryFailed") {
        return prev.map((task) => (
          retryingInstanceNames.includes(task.instance)
            ? { ...task, status: "待迁移", failReason: undefined }
            : task
        ));
      }

      return buildMigrationTasks(targetInstances);
    });

    const updateTasks = (updater: (tasks: InstanceTask[]) => InstanceTask[]) => {
      setInstanceTasks((prev) => updater(prev));
    };

    const STEP_LABELS = ["确认网络配置", "切换实例网络", "重启实例"] as const;
    const runInstance = (currentInstance: MigrationInstance, stepIndex: number, delayOffset: number) => {
      const stepLabel = STEP_LABELS[stepIndex];
      const failureScenario = MIGRATION_FAILURE_SCENARIOS[currentInstance.instance];

      updateTasks((tasks) =>
        tasks.map((task) =>
          task.instance === currentInstance.instance
            ? { ...task, status: "迁移中", failReason: undefined }
            : task
        )
      );

      setTimeout(() => {
        if (failureScenario?.step === stepLabel && !resolvedInstancesSnapshot.has(currentInstance.instance)) {
          updateTasks((tasks) =>
            tasks.map((task) =>
              task.instance === currentInstance.instance
                ? { ...task, status: "迁移失败", failReason: failureScenario.reason }
                : task
            )
          );
          return;
        }

        if (stepIndex === STEP_LABELS.length - 1) {
          updateTasks((tasks) =>
            tasks.map((task) =>
              task.instance === currentInstance.instance
                ? { ...task, currentNet: currentInstance.targetNet, status: "迁移成功", failReason: undefined }
                : task
            )
          );
          return;
        }

        runInstance(currentInstance, stepIndex + 1, delayOffset);
      }, 700 + delayOffset);
    };

    targetInstances.forEach((currentInstance, index) => {
      const delayOffset = (index % 3) * 120;
      setTimeout(() => runInstance(currentInstance, 0, delayOffset), 300 + index * 220);
    });
  };

  const renderMigrationStepBar = (activeStep: number) => (
    <div className="px-1 py-1">
      <div className="flex items-center">
        {migrationFlowSteps.map((label, index) => {
          const isCompleted = activeStep > index;
          const isActive = activeStep === index;

          return (
            <div key={label} className="flex min-w-0 flex-1 items-center last:flex-none">
              <div className="flex min-w-0 items-center gap-3">
                <div
                  className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full border text-xs font-semibold transition-colors ${
                    isCompleted || isActive
                      ? "border-blue-600 bg-blue-600 text-white"
                      : "border-border bg-white text-[var(--text-weak)]"
                  }`}
                >
                  {isCompleted ? <Check className="h-3.5 w-3.5" /> : index + 1}
                </div>
                <span
                  className={`truncate text-sm font-medium transition-colors ${
                    isActive
                      ? "text-[var(--text-title)]"
                      : isCompleted
                        ? "text-[var(--text-brand)]"
                        : "text-[var(--text-weak)]"
                  }`}
                >
                  {label}
                </span>
              </div>
              {index < migrationFlowSteps.length - 1 && (
                <div
                  className={`mx-4 h-px flex-1 transition-colors ${
                    activeStep > index ? "bg-blue-600" : "bg-gray-200"
                  }`}
                />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );

  useEffect(() => {
    if (migrationBusinessState !== "running" || hasRunningTasks) return;

    const nextBusinessState: MigrationBusinessState = hasFailedTasks ? "failed" : "success";
    setMigrationBusinessState(nextBusinessState);
    setMigrationResultTab(nextBusinessState === "failed" ? "failed" : "success");
    setMigrationViewState((prev) => (prev === "hidden" ? "hidden" : "execution"));
  }, [migrationBusinessState, hasRunningTasks, hasFailedTasks]);

  const MigrationBanner = shouldShowMigrationBanner ? (
    <div className="mb-5 flex items-start gap-3 rounded-[4px] border border-amber-200 bg-amber-50 px-4 py-3">
      <CircleAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
      <div className="min-w-0 flex-1">
        <p className="text-xs leading-relaxed text-amber-800">
          {migrationBannerState === "running" ? (
            <>Agent 实例迁移任务正在执行中。单击「<button onClick={openMigrationProgressDialog} className="font-medium text-amber-700 underline underline-offset-2 transition-colors hover:text-amber-900">查看进度</button>」可查看实例迁移情况。</>
          ) : migrationBannerState === "failed" ? (
            <>仍有 Agent 实例未完成迁移，请根据失败原因处理后重试。单击「<button onClick={() => openMigrationResultDialog("failed")} className="font-medium text-amber-700 underline underline-offset-2 transition-colors hover:text-amber-900">查看详情</button>」可查看失败实例并继续迁移。</>
          ) : (
            <>检测到当前企业下仍有 Agent 实例运行在用户个人 VPC 中，为避免后续网络管理和实例治理持续分散，建议尽快发起迁移。单击「<button onClick={openMigrationConfirmDialog} className="font-medium text-amber-700 underline underline-offset-2 transition-colors hover:text-amber-900">查看详情</button>」可查看迁移范围并确认迁移。</>
          )}
        </p>
      </div>
    </div>
  ) : null;

  const MigrationConfirmDialog = (
    <Dialog open={isMigrationConfirmOpen} onOpenChange={(open) => !open && closeMigrationDialog()}>
      <DialogContent style={{ maxWidth: "min(90vw, 860px)", width: "min(90vw, 860px)" }}>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold text-[var(--text-title)]">{migrationDialogTitle}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-1">
          {renderMigrationStepBar(0)}

          <div className="flex items-start gap-2.5 rounded-[4px] border border-amber-100 bg-amber-50 px-3 py-2.5">
            <CircleAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
            <ul className="list-disc space-y-1 pl-4 text-xs leading-relaxed text-amber-700">
              <li>迁移过程中，Agent 实例会重启；迁移完成后，内网 IP 默认会发生变化，请提前告知相关用户。</li>
              <li>仅迁移“可迁移”列表的 Agent 实例，您可查看暂不支持迁移的 Agent 实例，按提示处理后在发起迁移。</li>
            </ul>
          </div>

          <div className="overflow-hidden rounded-[4px] border border-border bg-white">
            <div className="flex items-center justify-between gap-4 border-b border-border px-4" style={{ minHeight: "44px" }}>
              <div className="flex items-center">
                {(["migratable", "blocked"] as const).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => {
                      setConfirmMigrationTab(tab);
                      setConfirmTableExpanded(false);
                    }}
                    className={`relative px-4 py-2.5 text-sm font-medium transition-colors whitespace-nowrap ${
                      confirmMigrationTab === tab
                        ? "text-[var(--text-brand)] border-b-2 border-blue-600 -mb-px"
                        : "text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
                    }`}
                  >
                    {tab === "migratable" ? "可迁移" : "暂不支持迁移"}
                    <MetaText as="span" tone="weak" className="ml-1.5">
                      ({tab === "migratable" ? migratableInstances.length : blockedInstances.length})
                    </MetaText>
                  </button>
                ))}
              </div>
              <MetaText as="div" tone="weak" className="shrink-0">
                <span>
                  涉及 Agent 实例数：<MetaMedium as="span" tone="secondary" className="tabular-nums">{migrationImpactSummary.instanceCount}</MetaMedium>
                </span>
                <MetaText as="span" tone="weak" className="mx-2">｜</MetaText>
                <span>
                  涉及用户数：<MetaMedium as="span" tone="secondary" className="tabular-nums">{migrationImpactSummary.userCount}</MetaMedium>
                </span>
              </MetaText>
            </div>
            <div className={`${confirmTableGridClass} border-b border-border bg-[#fafafa]/50 px-3 py-2 text-xs font-medium text-[var(--text-muted)]`}>
              {confirmTableColumns.map((label) => (
                <span key={label} className="whitespace-nowrap">{label}</span>
              ))}
            </div>
            <div className="divide-y divide-gray-50 bg-white">
              {(confirmTableExpanded ? activeConfirmInstances : activeConfirmInstances.slice(0, 4)).map((item, index) => (
                <div key={`${item.instance}-${index}`} className={`${confirmTableGridClass} items-center px-3 py-2 text-xs text-[var(--text-secondary)] hover:bg-[#f5f5f5]/60`}>
                  <span className="truncate text-[var(--text-muted)]">{item.user}</span>
                  <span className="truncate font-medium text-[var(--text-title)]">{item.instance}</span>
                  <span className="truncate text-[var(--text-muted)]">{item.currentNet}</span>
                  <span className="truncate text-[var(--text-brand)]">{item.targetNet}</span>
                  <span className={item.note === "可迁移" ? "text-green-600" : "text-amber-700"}>{item.note}</span>
                </div>
              ))}
            </div>
            {activeConfirmInstances.length > 4 && (
              <button
                className="w-full border-t border-border bg-[#fafafa]/50 py-2 text-xs text-blue-500 transition-colors hover:bg-[#f5f5f5] hover:text-[var(--text-brand)]"
                onClick={() => setConfirmTableExpanded((value) => !value)}
              >
                {confirmTableExpanded ? "收起" : `展开查看剩余 ${activeConfirmInstances.length - 4} 条实例`}
              </button>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={closeMigrationDialog}>取消</Button>
          <Button variant="dialog-confirm" onClick={() => startMigrationRun("all")}>
            开始迁移
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  const MigrationProgressDialog = (
    <Dialog
      open={isMigrationProgressOpen}
      onOpenChange={(open) => {
        if (!open && isViewingMigrationResult) {
          closeMigrationDialog();
        }
      }}
    >
      <DialogContent
        className="sm:max-w-[920px]"
        showCloseButton={isViewingMigrationResult}
        style={{ maxWidth: "min(90vw, 900px)", width: "min(90vw, 900px)" }}
        onInteractOutside={(event) => {
          if (!isViewingMigrationResult) {
            event.preventDefault();
          }
        }}
        onEscapeKeyDown={(event) => {
          if (!isViewingMigrationResult) {
            event.preventDefault();
          }
        }}
      >
        <DialogHeader>
          <DialogTitle className="text-base font-semibold text-[var(--text-title)]">{migrationDialogTitle}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          {renderMigrationStepBar(isViewingMigrationResult ? 2 : 1)}

          {isViewingMigrationResult ? (
            <div className="overflow-hidden rounded-[4px] border border-border bg-white">
              <div className="flex items-center justify-between gap-4 border-b border-border px-4" style={{ minHeight: "44px" }}>
                <div className="flex items-center">
                  {resultTabs.map((tab) => (
                    <button
                      key={tab}
                      onClick={() => setMigrationResultTab(tab)}
                      className={`relative px-4 py-2.5 text-sm font-medium transition-colors whitespace-nowrap ${
                        migrationResultTab === tab
                          ? "text-[var(--text-brand)] border-b-2 border-blue-600 -mb-px"
                          : "text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
                      }`}
                    >
                      {tab === "success" ? "迁移成功" : "迁移失败"}
                      <MetaText as="span" tone="weak" className="ml-1.5">
                        ({tab === "success" ? currentRunSuccessfulMigrationTasks.length : currentRunFailedMigrationTasks.length})
                      </MetaText>
                    </button>
                  ))}
                </div>
              </div>
              <div className={`${resultTableGridClass} border-b border-border bg-[#fafafa]/50 px-3 py-2 text-xs font-medium text-[var(--text-muted)]`}>
                {resultTableColumns.map((label) => (
                  <span key={label}>{label}</span>
                ))}
              </div>
              <div className="divide-y divide-gray-50 max-h-56 overflow-y-auto bg-white">
                {activeResultTasks.length > 0 ? activeResultTasks.map((task, index) => (
                  <div key={`${migrationResultTab}-${task.instance}-${index}`} className={`${resultTableGridClass} items-center px-3 py-2 text-xs text-[var(--text-secondary)] hover:bg-[#f5f5f5]/60`}>
                    <span className="truncate text-[var(--text-muted)]">{task.user}</span>
                    <span className="truncate font-medium text-[var(--text-title)]">{task.instance}</span>
                    <span className="truncate text-[var(--text-muted)]">{task.currentNet}</span>
                    <span className={`inline-flex items-center gap-1 font-medium ${migrationResultTab === "success" ? "text-green-600" : "text-[var(--text-danger)]"}`}>
                      <span className={`h-1.5 w-1.5 rounded-full ${migrationResultTab === "success" ? "bg-green-500" : "bg-red-500"}`} />
                      {migrationResultTab === "success" ? "迁移成功" : "迁移失败"}
                    </span>
                    <span className={migrationResultTab === "success" ? "text-[var(--text-muted)] leading-relaxed" : "text-[var(--text-danger)] leading-relaxed"}>
                      {migrationResultTab === "success" ? "-" : task.failReason ?? "请稍后重试；如问题持续存在，请联系管理员处理"}
                    </span>
                  </div>
                )) : (
                  <BodyText as="div" tone="weak" className="px-3 py-10 text-center">
                    暂无{migrationResultTab === "success" ? "迁移成功" : "迁移失败"}实例
                  </BodyText>
                )}
              </div>
            </div>
          ) : (
            <>
              <div className="overflow-hidden rounded-[4px] border border-border bg-white">
                <div className={`${executionTableGridClass} border-b border-border bg-[#fafafa]/50 px-3 py-2 text-xs font-medium text-[var(--text-muted)]`}>
                  {executionTableColumns.map((label) => (
                    <span key={label}>{label}</span>
                  ))}
                </div>
                <div className="divide-y divide-gray-50 max-h-56 overflow-y-auto bg-white">
                  {activeMigrationTasks.map((task, index) => (
                    <div key={`${task.instance}-${index}`} className={`${executionTableGridClass} items-center px-3 py-2 text-xs text-[var(--text-secondary)] hover:bg-[#f5f5f5]/60`}>
                      <span className="truncate text-[var(--text-muted)]">{task.user}</span>
                      <span className="truncate font-medium text-[var(--text-title)]">{task.instance}</span>
                      <span className={`inline-flex items-center gap-1 font-medium ${
                        task.status === "迁移成功"
                          ? "text-green-600"
                          : task.status === "迁移中"
                            ? "text-[var(--text-brand)]"
                            : task.status === "迁移失败"
                              ? "text-[var(--text-danger)]"
                              : "text-[var(--text-weak)]"
                      }`}>
                        {task.status === "迁移中" ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <span className={`h-1.5 w-1.5 rounded-full ${
                            task.status === "迁移成功"
                              ? "bg-green-500"
                              : task.status === "迁移失败"
                                ? "bg-red-500"
                                : "bg-gray-300"
                          }`} />
                        )}
                        {task.status}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </div>
        <DialogFooter>
          {isViewingMigrationResult ? (
            migrationBusinessState === "failed" ? (
              <>
                <Button variant="outline" onClick={closeMigrationDialog}>稍后处理</Button>
                <Button variant="dialog-confirm" onClick={() => startMigrationRun("retryFailed")}>
                  重试失败实例
                </Button>
              </>
            ) : (
              <Button variant="dialog-confirm" onClick={closeMigrationDialog}>
                完成
              </Button>
            )
          ) : isCurrentRunCompleted ? (
            isRetryRunCompletedWithAllSuccess ? (
              <Button variant="dialog-confirm" onClick={closeMigrationDialog}>
                完成
              </Button>
            ) : currentRunFailedMigrationTasks.length > 0 ? (
              <>
                <Button variant="outline" onClick={closeMigrationDialog}>
                  最小化
                </Button>
                <Button variant="dialog-confirm" onClick={() => openMigrationResultDialog()}>
                  下一步：查看结果
                </Button>
              </>
            ) : (
              <Button variant="dialog-confirm" onClick={() => openMigrationResultDialog()}>
                下一步：查看结果
              </Button>
            )
          ) : (
            <Button variant="outline" onClick={closeMigrationDialog}>
              最小化
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  // 安全组切换状态
  const [currentSg, setCurrentSg] = useState<SecurityGroup | null>(initialDefaultSecurityGroup);

  const [isSgDialogOpen, setIsSgDialogOpen] = useState(false);
  // [PR #169] "导入规则"操作的二次确认弹窗开关（点击导入弹窗的"确定"后弹出）
  const [isConfirmSwitchDialogOpen, setIsConfirmSwitchDialogOpen] = useState(false);
  // 当前「导入已有安全组规则」弹窗的业务语义（与 main 状态机命名对齐）：
  //  - "replace"  已有默认安全组 → 仅复制规则到当前默认安全组，不新建、不切换
  //  - "create"   尚未配置默认安全组 → 复制规则并创建一个默认安全组（空态入口）
  const [sgDialogMode, setSgDialogMode] = useState<"create" | "replace">("replace");
  const [sgSearchKeyword, setSgSearchKeyword] = useState("");
  const [sgDialogCandidatePage, setSgDialogCandidatePage] = useState(1);
  const [sgDialogSelected, setSgDialogSelected] = useState<SecurityGroup | null>(null); // 弹窗内选中的候选
  const [sgDialogTab, setSgDialogTab] = useState<"outbound" | "inbound">("outbound");
  const [isSgDialogPanelAccessStaged, setIsSgDialogPanelAccessStaged] = useState(false);

  // 将「当前默认安全组 + 入方向规则」同步到本地快照，供平台策略页等管理员页面只读消费。
  useEffect(() => {
    writeDefaultSecurityGroupSnapshot(currentSg, inboundRules);
  }, [currentSg, inboundRules]);

  // 新建安全组状态
  const [showCreateSgDialog, setShowCreateSgDialog] = useState(false);
  const [createSgDraft, setCreateSgDraft] = useState({ name: "", remark: "" });

  const [createSgCheckedOptions, setCreateSgCheckedOptions] = useState<CommonRuleOptionKey[]>(
    COMMON_RULE_OPTIONS.filter((o) => o.defaultChecked).map((o) => o.key)
  );
  const [createSgPreviewTab, setCreateSgPreviewTab] = useState<"inbound" | "outbound">("outbound");

  // 路由导航（公网 → 资源配置）
  const [, navigate] = useLocation();

  // 编辑规则状态
  const [editingRule, setEditingRule] = useState<{ id: string; type: "inbound" | "outbound" } | null>(null);
  const [editDraft, setEditDraft] = useState<Partial<Rule>>({});
  const [showDeleteDialog, setShowDeleteDialog] = useState<{ id: string; type: "inbound" | "outbound"; is_required?: boolean } | null>(null);
  const [showAddDialog, setShowAddDialog] = useState<"inbound" | "outbound" | null>(null);
  const [addDraft, setAddDraft] = useState<Partial<Rule>>({});

  // [004] 导入规则到当前 ClawPro 安全组（B15：抄规则，ClawPro 安全组身份恒定）
  //
  // 【背景】
  //   ClawPro 安全组是一个本地对象（name / remark / cloudSgs 恒定，对外不暴露 ID）。
  //   "切换"和"选择已有"本质上都是"把另一个 sg 的规则抄到当前 ClawPro 安全组里"。
  //   sourceSg 仅作为"规则模板"，其 name/cloudSgs 等身份信息一律不继承。
  //
  // 【稳定态】currentSg 已存在
  //   ✅ 只覆盖 rules（inboundRules/outboundRules + 对应计数）
  //   ❌ 不改 currentSg.name / remark / cloudSgs（K2/K4/B15 联合约束）
  //
  // 【初始化】currentSg 为空（用户首次点"选择已有安全组"建立 ClawPro 安全组）
  //   ⚠️  不能直接 setCurrentSg(sourceSg)，否则 UI 会把腾讯云 sg 的名字
  //       显示成 ClawPro 安全组的身份，违反 004 解耦心智。
  //   ✅ 构造一个新的 ClawPro 安全组：
  //       - name: "ClawPro-Default"（ClawPro 默认名，不继承 sourceSg.name）
  //       - remark: "Agent 默认安全组"（固定业务语义，选择 I 策略）
  //       - cloudSgs: 单分片，cloudSgName 用 K4 约定格式 clawpro-sg-{域名}-default-01
  //       - rules: 抄 sourceSg 的规则
  //
  // 【Mock 简化 vs 真实后端】
  //   - 真实：ClawPro 后端在腾讯云建专属 sg + 抄规则 + Agent 换绑
  //   - Mock：只在前端状态里构造等效对象
  const applyCurrentSecurityGroup = (sourceSg: SecurityGroup) => {
    setCurrentSg((prev) => {
      // 初始化：构造新的 ClawPro 安全组，仅抄规则，身份走默认值
      if (!prev) {
        const cloudSgId = generateRandomSgId();
        return {
          name: INITIAL_DEFAULT_SECURITY_GROUP_NAME,
          remark: "Agent 默认安全组",
          inboundCount: sourceSg.inboundCount,
          outboundCount: sourceSg.outboundCount,
          inboundRules: sourceSg.inboundRules,
          outboundRules: sourceSg.outboundRules,
          // K4 约定：ClawPro 主动创建的云端 sg
          //   - sgId: 云端真实 sg-id 格式（sg-xxxxxxxx，8 位字符）
          //   - cloudSgName: 默认格式 clawpro-sg-{域名}-default-{序号}（不复用 sourceSg.name）
          cloudSgs: [
            { sgId: cloudSgId, cloudSgName: DEFAULT_CLOUD_SECURITY_GROUP_NAME, seq: 1 },
          ],
        };
      }
      // 稳定态：ClawPro 安全组身份恒定，仅抄规则
      return {
        ...prev,
        inboundRules: sourceSg.inboundRules,
        outboundRules: sourceSg.outboundRules,
        inboundCount: sourceSg.inboundCount,
        outboundCount: sourceSg.outboundCount,
      };
    });
    setInboundRules(sourceSg.inboundRules);
    setOutboundRules(sourceSg.outboundRules);
  };

  const resetCreateSecurityGroupState = () => {
    setCreateSgDraft({ name: "", remark: "" });
    setCreateSgCheckedOptions(COMMON_RULE_OPTIONS.filter((o) => o.defaultChecked).map((o) => o.key));
    setCreateSgPreviewTab("outbound");
  };

  const openCreateSecurityGroupDialog = () => {
    resetCreateSecurityGroupState();
    setShowCreateSgDialog(true);
  };

  const closeCreateSecurityGroupDialog = () => {
    setShowCreateSgDialog(false);
    resetCreateSecurityGroupState();
  };

  const handleCreateSecurityGroupDialogOpenChange = (open: boolean) => {
    if (open) {
      setShowCreateSgDialog(true);
      return;
    }
    closeCreateSecurityGroupDialog();
  };

  const handleCreateSgOptionToggle = (optionKey: CommonRuleOptionKey, checked: boolean) => {
    setCreateSgCheckedOptions((prev) => {
      if (checked) return [...prev, optionKey];
      return prev.filter((k) => k !== optionKey);
    });
  };

  const openSelectSecurityGroupDialog = (mode: "create" | "replace" = "replace") => {
    setSgDialogMode(mode);
    setSgSearchKeyword("");
    setSgDialogCandidatePage(1);
    setSgDialogSelected(null);
    setSgDialogTab("outbound");
    setIsSgDialogPanelAccessStaged(false);
    setIsSgDialogOpen(true);
  };

  const closeSelectSecurityGroupDialog = () => {
    setSgSearchKeyword("");
    setSgDialogCandidatePage(1);
    setSgDialogSelected(null);
    setIsSgDialogPanelAccessStaged(false);
    setIsSgDialogOpen(false);
  };

  // ── 可拖拽的规则行组件 ──
  function SortableRuleRow({
    rule,
    type,
    readonly,
    onEdit,
    onDelete,
  }: {
    rule: Rule;
    type: "inbound" | "outbound";
    readonly: boolean;
    onEdit?: (rule: Rule, type: "inbound" | "outbound") => void;
    onDelete?: (rule: Rule, type: "inbound" | "outbound") => void;
  }) {
    const {
      attributes,
      listeners,
      setNodeRef,
      transform,
      transition,
      isDragging,
    } = useSortable({ id: rule.id });

    const style: React.CSSProperties = {
      // 锁水平方向，只允许垂直拖拽
      transform: CSS.Translate.toString(transform ? { ...transform, x: 0 } : null),
      transition,
      zIndex: isDragging ? 10 : undefined,
      opacity: isDragging ? 0.5 : 1,
      position: isDragging ? "relative" : undefined,
    };

    return (
      <TableRow ref={setNodeRef} style={style} className={isDragging ? "bg-[var(--alert-info-bg)]/40" : ""}>
        <TableCell>{rule.source}</TableCell>
        <TableCell>{rule.protocol}</TableCell>
        <TableCell>{rule.port}</TableCell>
        <TableCell>
          {rule.policy === "允许" ? (
            <StatusTag mode="fill" variant="green">
              {rule.policy}
            </StatusTag>
          ) : (
            <StatusTag mode="fill" variant="gray">
              {rule.policy}
            </StatusTag>
          )}
        </TableCell>
        <TableCell>{rule.remark || "—"}</TableCell>
        {!readonly && (
          <TableActionCell>
            <Button variant="link" size="sm" onClick={() => onEdit?.(rule, type)}>编辑</Button>
            <Button variant="link" size="sm" onClick={() => onDelete?.(rule, type)}>删除</Button>
            {isAdminDisabled && activeTab === "security" ? (
              <span
                onClick={() => toast.info("管控台已到期，请续费后操作")}
                className="ml-1 inline-flex items-center justify-center size-6 rounded cursor-not-allowed"
                role="button"
                aria-label="拖拽排序"
                title="拖拽排序"
              >
                <button
                  type="button"
                  disabled
                  style={{ pointerEvents: "none", opacity: 0.4 }}
                  className="inline-flex items-center justify-center size-6 rounded text-[var(--text-muted)]"
                  aria-label="拖拽排序"
                  title="拖拽排序"
                >
                  <GripVertical className="w-4 h-4" />
                </button>
              </span>
            ) : (
              <button
                type="button"
                {...attributes}
                {...listeners}
                className="ml-1 inline-flex items-center justify-center size-6 rounded text-[var(--text-muted)] hover:text-[var(--text-secondary)] cursor-grab active:cursor-grabbing transition-colors"
                title="拖拽排序"
                aria-label="拖拽排序"
              >
                <GripVertical className="w-4 h-4" />
              </button>
            )}
          </TableActionCell>
        )}
      </TableRow>
    );
  }

  // ── 规则表格内容组件（不包含卡片外层）──
  function RuleTableBody({ rules, type, readonly = false, onEdit, onDelete, onReorder, paginate = false }: RuleTableBodyProps) {
    const [page, setPage] = useState(1);
    const PAGE_SIZE = 10;
    const containerRef = useRef<HTMLDivElement>(null);

    const sensors = useSensors(
      // 移动 5px 后才激活拖拽，避免误触发与点击「编辑/删除」按钮冲突
      useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
      useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
    );

    useEffect(() => {
      setPage(1);
    }, [type, rules.length]);

    const totalPages = Math.max(1, Math.ceil(rules.length / PAGE_SIZE));
    const currentPage = Math.min(page, totalPages);

    const displayRules = paginate ? rules.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE) : rules;

    // 自定义 modifier：锁定水平方向 + 限制垂直范围在容器边界内
    const restrictToRulesBoundary: Modifier = ({ transform, draggingNodeRect }) => {
      if (!containerRef.current || !draggingNodeRect) {
        return { ...transform, x: 0 };
      }
      const containerRect = containerRef.current.getBoundingClientRect();
      const minY = containerRect.top - draggingNodeRect.top;
      const maxY = containerRect.bottom - draggingNodeRect.bottom;
      return {
        ...transform,
        x: 0,
        y: Math.min(Math.max(transform.y, minY), maxY),
      };
    };

    const handleDragEnd = (event: DragEndEvent) => {
      const { active, over } = event;
      if (over && active.id !== over.id) {
        const oldIndex = rules.findIndex((r) => r.id === active.id);
        const newIndex = rules.findIndex((r) => r.id === over.id);
        if (oldIndex !== -1 && newIndex !== -1) {
          const reordered = arrayMove(rules, oldIndex, newIndex);
          onReorder?.(reordered);
          // TODO(后端接入): POST /admin/config/security-group/policies/reorder
          toast.success("规则顺序已更新");
        }
      }
    };

    // readonly 时不挂 DndContext，回退为普通展示行
    if (readonly) {
      return (
        <div className="flex flex-col w-full">
          <Table variant="white" density="default" autoFixedColumns={false}>
            <TableHeader>
              <TableRow>
                <TableHead>{type === "inbound" ? "来源" : "目标"}</TableHead>
                <TableHead>协议</TableHead>
                <TableHead>端口</TableHead>
                <TableHead>策略</TableHead>
                <TableHead>备注</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayRules.length > 0 ? (
                displayRules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell>{rule.source}</TableCell>
                    <TableCell>{rule.protocol}</TableCell>
                    <TableCell>{rule.port}</TableCell>
                    <TableCell>
                      {rule.policy === "允许" ? (
                        <StatusTag mode="fill" variant="green">{rule.policy}</StatusTag>
                      ) : (
                        <StatusTag mode="fill" variant="gray">{rule.policy}</StatusTag>
                      )}
                    </TableCell>
                    <TableCell>{rule.remark || "—"}</TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow className="hover:bg-transparent">
                  <TableCell colSpan={5} className="text-center py-8">
                    {type === "outbound" ? (
                      <MetaText as="p" tone="weak" className="leading-relaxed">出站规则为空时，所有出站流量将被拒绝，Agent 将无法正常使用</MetaText>
                    ) : (
                      <MetaText as="p" tone="weak">暂无入站规则</MetaText>
                    )}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      );
    }

    return (
      <div ref={containerRef} className="flex flex-col w-full">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          modifiers={[restrictToRulesBoundary]}
          onDragEnd={handleDragEnd}
        >
          <Table variant="white" density="default" autoFixedColumns={false}>
            <TableHeader>
              <TableRow>
                <TableHead>{type === "inbound" ? "来源" : "目标"}</TableHead>
                <TableHead>协议</TableHead>
                <TableHead>端口</TableHead>
                <TableHead>策略</TableHead>
                <TableHead>备注</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <SortableContext
              items={displayRules.map((r) => r.id)}
              strategy={verticalListSortingStrategy}
            >
              <TableBody>
                {displayRules.length > 0 ? (
                  displayRules.map((rule) => (
                    <SortableRuleRow
                      key={rule.id}
                      rule={rule}
                      type={type}
                      readonly={false}
                      onEdit={onEdit}
                      onDelete={onDelete}
                    />
                  ))
                ) : (
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={6} className="text-center py-8">
                      {type === "outbound" ? (
                        <MetaText as="p" tone="weak" className="leading-relaxed">出站规则为空时，所有出站流量将被拒绝，Agent 将无法正常使用</MetaText>
                      ) : (
                        <MetaText as="p" tone="weak">暂无入站规则</MetaText>
                      )}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </SortableContext>
          </Table>
        </DndContext>

        {paginate && (
          <div className="px-4 py-3 border-t border-gray-200 bg-white">
            <Pagination
              total={rules.length}
              current={currentPage}
              pageSize={10}
              showTotal={(total) => `共 ${total} 条规则`}
              className="w-full justify-between"
              onChange={(page) => setPage(page)}
            />
          </div>
        )}
      </div>
    );
  }

  const renderSelectExistingSecurityGroupDialog = ({
    open,
    searchKeyword,
    selectedSecurityGroup,
    previewTab,
    candidateSecurityGroups,
    candidatePage,
    candidateTotalPages,
    // 旧「一键添加 / 取消添加」交互已移除，相关 props 保留在接口定义以兼容调用方，但此处不再消费
    onOpenChange,
    onSearchChange,
    onClearSearch,
    onSelectSecurityGroup,
    onCandidatePageChange,
    onPreviewTabChange,
    onCancel,
    onConfirm,
  }: SelectExistingSecurityGroupDialogProps) => (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
    >
      <DialogContent
        className="sm:max-w-[704px] flex flex-col"
        style={{ maxHeight: "min(90vh, 820px)" }}
        onInteractOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>导入规则到 ClawPro 安全组</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          <div className="space-y-4">
            <Alert variant="info" className="px-4 py-3">
              <Info />
              <AlertDescription>
                {sgDialogMode === "create"
                  ? "以下为您云端已有安全组的规则，可作为规则模板导入。确认后，所选规则将复制到 ClawPro 安全组（默认名称为 ClawPro-Default），原云端安全组不受影响。"
                  : "以下为您云端已有安全组的规则，可作为规则模板导入。确认后，所选规则将复制到当前 ClawPro 安全组，原云端安全组不受影响。"}
              </AlertDescription>
            </Alert>

            {/* ── 规则模板 ── */}
            <div className="space-y-3">
              <BodyMedium as="div" tone="primary">选择规则模板</BodyMedium>
              <div className="rounded-[4px] border border-border overflow-hidden bg-white">
                <div className="relative border-b border-border p-3">
                  <Search className="absolute left-6 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)] z-10" />
                  <Input
                    type="text"
                    placeholder="搜索规则模板名称或 ID"
                    value={searchKeyword}
                    onChange={(e) => onSearchChange(e.target.value)}
                    className="pl-9 pr-9"
                  />
                  {searchKeyword && (
                    <button
                      onClick={onClearSearch}
                      className="absolute right-6 top-1/2 -translate-y-1/2 text-[var(--text-weak)] hover:text-[var(--text-muted)] z-10"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>

                {candidateSecurityGroups.length === 0 ? (
                  <div className="text-center py-10">
                    <HelperText>未找到匹配的规则模板</HelperText>
                  </div>
                ) : (
                  <RadioGroup
                    value={selectedSecurityGroup?.name ?? ""}
                    onValueChange={(name) => {
                      const sg = candidateSecurityGroups.find((s) => s.name === name);
                      if (sg) onSelectSecurityGroup(sg);
                    }}
                    className="gap-0 max-h-[280px] overflow-y-auto"
                    style={{ scrollbarGutter: "stable" }}
                  >
                    {candidateSecurityGroups.map((sg) => {
                      const isSelected = selectedSecurityGroup?.name === sg.name;
                      return (
                        <label
                          key={sg.name}
                          htmlFor={`tpl-${sg.name}`}
                          className={`w-full text-left px-4 py-2.5 border-b border-border last:border-b-0 cursor-pointer transition-colors flex items-start gap-3 ${
                            isSelected ? "bg-[rgba(20,71,230,0.06)]" : "hover:bg-[var(--bg-hover-subtle)]"
                          }`}
                        >
                          <RadioGroupItem
                            value={sg.name}
                            id={`tpl-${sg.name}`}
                            className="mt-0.5 shrink-0"
                          />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-1.5 min-w-0">
                              {isSelected ? (
                                <BodyMedium as="span" tone="primary" className="truncate">{sg.name}</BodyMedium>
                              ) : (
                                <BodyText as="span" tone="primary" className="truncate">{sg.name}</BodyText>
                              )}
                            </div>
                            <MetaText as="p" className="mt-0.5 truncate">{sg.remark || "—"}</MetaText>
                          </div>
                          <div className="mt-0.5 flex items-center gap-1.5 shrink-0 whitespace-nowrap">
                            <MetaText tone="weak">入站 {sg.inboundCount} 条</MetaText>
                            <span className="text-gray-200" aria-hidden>|</span>
                            <MetaText tone="weak">出站 {sg.outboundCount} 条</MetaText>
                          </div>
                        </label>
                      );
                    })}
                  </RadioGroup>
                )}
              </div>
            </div>

            {selectedSecurityGroup && (() => {
              const previewRulesList = previewTab === "outbound"
                ? selectedSecurityGroup.outboundRules
                : selectedSecurityGroup.inboundRules;
              return (
                <div className="space-y-2">
                  <BodyMedium as="div" tone="primary">规则预览</BodyMedium>
                  <div>
                    <SegmentGroup>
                      <SegmentOption
                        active={previewTab === "outbound"}
                        onClick={() => onPreviewTabChange("outbound")}
                      >
                        出站规则
                        <MetaText tone="weak" className="ml-1">({selectedSecurityGroup.outboundCount})</MetaText>
                      </SegmentOption>
                      <SegmentOption
                        active={previewTab === "inbound"}
                        onClick={() => onPreviewTabChange("inbound")}
                      >
                        入站规则
                        <MetaText tone="weak" className="ml-1">({selectedSecurityGroup.inboundCount})</MetaText>
                      </SegmentOption>
                    </SegmentGroup>
                  </div>

                  <SurfaceCard className="overflow-hidden">
                    <div className="max-h-64 overflow-y-auto scrollbar-on-hover">
                      <Table density="compact" autoFixedColumns={false}>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{previewTab === "outbound" ? "目标" : "来源"}</TableHead>
                            <TableHead>协议</TableHead>
                            <TableHead>端口</TableHead>
                            <TableHead>策略</TableHead>
                            <TableHead>备注</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {previewRulesList.length > 0 ? (
                            previewRulesList.map((rule) => (
                              <TableRow key={rule.id}>
                                <TableCell>{rule.source}</TableCell>
                                <TableCell>{rule.protocol}</TableCell>
                                <TableCell>{rule.port}</TableCell>
                                <TableCell>
                                  {rule.policy === "允许" ? (
                                    <StatusTag mode="fill" variant="green">
                                      {rule.policy}
                                    </StatusTag>
                                  ) : (
                                    <StatusTag mode="fill" variant="gray">
                                      {rule.policy}
                                    </StatusTag>
                                  )}
                                </TableCell>
                                <TableCell><MetaText tone="muted">{rule.remark || "—"}</MetaText></TableCell>
                              </TableRow>
                            ))
                          ) : (
                            <TableRow>
                              <TableCell colSpan={5} className="text-center py-8">
                                <MetaText as="span" tone="weak">
                                  {previewTab === "outbound"
                                    ? "出站规则为空时，所有出站流量将被拒绝，Agent 将无法正常使用"
                                    : "暂无入站规则"}
                                </MetaText>
                              </TableCell>
                            </TableRow>
                          )}
                        </TableBody>
                      </Table>
                    </div>
                  </SurfaceCard>
                </div>
              );
            })()}
          </div>
        </DialogBody>

        {/* 底部：操作按钮（固定） */}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={onCancel}
          >
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            disabled={!selectedSecurityGroup}
            onClick={onConfirm}
          >
            导入所选规则
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  const currentTab = TABS.find((t) => t.id === activeTab)!;
  // [004] U8：候选列表过滤掉 ClawPro 自建的云 sg（不能导入自己的规则到自己）
  //   判定条件（任一即过滤）：
  //     ① 当前 ClawPro 安全组本身
  //     ② cloudSgs.length > 1（参与 ClawPro 分片池，说明是 ClawPro 托管）
  //     ③ 任意 cloudSg 的 seq > 1（是扩容产生的分片）
  //     ④ cloudSgName 以 "clawpro-sg-" 开头（ClawPro 命名约定识别）
  const isClawproManagedSg = (sg: SecurityGroup) => {
    if (sg.cloudSgs.length > 1) return true;
    return sg.cloudSgs.some(
      (cs) => cs.seq > 1 || cs.cloudSgName.startsWith("clawpro-sg-")
    );
  };
  const selectableSecurityGroups = MOCK_SECURITY_GROUP_DIALOG_CANDIDATES.filter(
    (sg) =>
      sg.name !== currentSg?.name &&
      !isClawproManagedSg(sg) &&
      (sgSearchKeyword === "" ||
        sg.name.toLowerCase().includes(sgSearchKeyword.toLowerCase()) ||
        sg.name.toLowerCase().includes(sgSearchKeyword.toLowerCase()) ||
        (sg.remark ?? "").toLowerCase().includes(sgSearchKeyword.toLowerCase()))
  );
  const selectableSecurityGroupTotalPages = Math.max(1, Math.ceil(selectableSecurityGroups.length / SECURITY_GROUP_DIALOG_PAGE_SIZE));
  const selectableSecurityGroupCurrentPage = Math.min(sgDialogCandidatePage, selectableSecurityGroupTotalPages);
  const pagedSelectableSecurityGroups = selectableSecurityGroups.slice(
    (selectableSecurityGroupCurrentPage - 1) * SECURITY_GROUP_DIALOG_PAGE_SIZE,
    selectableSecurityGroupCurrentPage * SECURITY_GROUP_DIALOG_PAGE_SIZE
  );

  useEffect(() => {
    if (sgDialogCandidatePage > selectableSecurityGroupTotalPages) {
      setSgDialogCandidatePage(1);
    }
  }, [sgDialogCandidatePage, selectableSecurityGroupTotalPages]);

  const activePanelAccessPort = panelPort || DEFAULT_PANEL_ACCESS_PORT;
  const panelAccessRequiredRules = allowPanelAccess ? buildPanelAccessRequiredRules(activePanelAccessPort) : [];
  const sgDialogMissingPanelAccessRules = Boolean(
    allowPanelAccess
    && sgDialogSelected
    && panelAccessRequiredRules.some((expectedRule) =>
      !sgDialogSelected.inboundRules.some((rule) => doesRuleCoverPanelAccessRequirement(rule, expectedRule))
    )
  );
  const sgDialogPreviewSecurityGroup = sgDialogSelected && isSgDialogPanelAccessStaged
    ? buildSecurityGroupWithPanelAccessRules(sgDialogSelected, activePanelAccessPort)
    : sgDialogSelected;
  const shouldShowSgDialogPanelAccessAssist = Boolean(
    sgDialogSelected && allowPanelAccess && (sgDialogMissingPanelAccessRules || isSgDialogPanelAccessStaged)
  );

  return (
    <>
      <div className="page-enter">

        <AdminPageHeader title="网络管理" />

        {/* Tab 切换器 */}
        <div className="flex items-center gap-1 mb-1 border-b border-[#dbe6ff]">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              data-billing-exempt=""
              className={`relative flex items-center gap-1.5 px-4 py-3 transition-colors whitespace-nowrap ${
                activeTab === tab.id
                  ? "border-b-2 border-[#0A0A0A] -mb-px"
                  : ""
              }`}
            >
              <BodyMedium tone={activeTab === tab.id ? "primary" : "muted"}>
                {tab.label}
              </BodyMedium>
            </button>
          ))}
        </div>

        {/* Tab 描述 */}
        <BodyText tone="muted" className="mt-3 mb-6">{currentTab.description}</BodyText>

        {/* Tab 内容 */}
        {activeTab === "security" && (
        <div className="flex flex-col gap-4">
          {/* [004] 安全组 Tab 顶部双提示框（参照 VPC/子网 Tab 布局风格）
              统一规则：两个框都仅在已配置 ClawPro 安全组后才显示
              - 黄色（⚠️）：操作风险提示
              - 蓝色（ℹ️）：产品价值宣言
              布局：两 banner 同属"页面级告知区"，用内层 gap-3（12px）聚合为一组；
                   告知区与下方配置卡片之间保持外层 gap-6（24px）喘息空间。 */}
          {currentSg && (
            <Alert variant="operation-info">
              <AlertOperationInfoIcon />
              <AlertDescription>
                <p>
                  当前企业下所有 Agent 云服务器共用同一个 ClawPro 安全组，
                  <MetaMedium as="span" tone="danger" className="font-semibold">修改规则将对所有 Agent 立即统一生效，请谨慎操作。</MetaMedium>
                </p>
                <p>• 作用范围：此处规则变更仅作用于由 ClawPro 创建并托管的专属云端安全组，不会影响您原有的其他云端安全组及其资源。</p>
                <p>• 一致性保障：规则始终以 ClawPro 侧配置为准。所有变更会自动同步至云端；若云端规则被其他方式修改，系统会定时检查并自动恢复为 ClawPro 中的设定。</p>
              </AlertDescription>
            </Alert>
          )}


          {/* ===== 模块一：已绑定安全组 ===== */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <PanelTitle as="h3">ClawPro 安全组</PanelTitle>
              {currentSg && (
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={() => openSelectSecurityGroupDialog()}
                  className="shrink-0"
                >
                  导入规则
                </Button>
              )}
            </div>
            {currentSg ? (
              <Card className="overflow-hidden py-0 gap-0">
              <div className="px-5 pt-5 pb-5 flex flex-col">
                {/* Header：标题 + 副标题 */}
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <CardTitle as="h3" className="truncate">{currentSg.name}</CardTitle>
                      {currentSg.remark === "Agent 默认安全组" && (
                        <StatusTag mode="fill" variant="blue">Agent 默认安全组</StatusTag>
                      )}
                    </div>
                    {currentSg.remark && currentSg.remark !== "Agent 默认安全组" && (
                      <MetaText as="p" className="mt-1 line-clamp-2">
                        {currentSg.remark}
                      </MetaText>
                    )}
                  </div>
                </div>

                {/* 内部灰色区块：云端安全组列表 */}
                <div className="mt-4 rounded-[4px] bg-[var(--bg-grey-normal)] px-4 py-3 space-y-2">
                  <div className="flex items-center gap-3 flex-wrap">
                    <div className="flex items-center gap-1 shrink-0">
                      <MetaText tone="secondary">
                        对应 {currentSg.cloudSgs.length} 个云端安全组
                      </MetaText>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="inline-flex items-center text-[var(--text-weak)] hover:text-[var(--text-muted)] cursor-help">
                              <Info className="w-3.5 h-3.5" />
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-[320px] text-xs leading-relaxed">
                            在云端控制台中，「{currentSg.name}」对应以下由 ClawPro 自动生成的云端安全组：
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    {[...currentSg.cloudSgs]
                      .sort((a, b) => a.seq - b.seq)
                      .map((sg) => (
                        <div
                          key={sg.sgId}
                          className="inline-flex items-baseline gap-2 px-2.5 py-1 rounded-[4px] bg-white border border-[#e5e5e5] min-w-0 max-w-full"
                        >
                          <MetaText tone="body" className="shrink-0">
                            ID：{sg.sgId}
                          </MetaText>
                          <MetaText tone="secondary" className="shrink-0" aria-hidden>｜</MetaText>
                          <MetaText tone="body" className="truncate" title={sg.cloudSgName}>
                            名称：{sg.cloudSgName}
                          </MetaText>
                        </div>
                      ))}
                  </div>
                  {currentSg.cloudSgs.length > 1 && (
                    <div className="flex items-start gap-2 bg-white border border-[#e5e5e5] rounded-[4px] px-3 py-2.5 mt-2">
                      <Info className="w-3.5 h-3.5 text-[var(--text-muted)] mt-0.5 shrink-0" />
                      <MetaText tone="secondary" className="leading-relaxed">
                        当 Agent 数量超过单个云端安全组的承载上限时，ClawPro 会自动创建更多云端安全组来承载，所有安全组规则保持一致。
                      </MetaText>
                    </div>
                  )}
                </div>
              </div>
            </Card>
          ) : (
            <SurfaceCard className="overflow-hidden">
              <div className="px-6 py-6">
                {/* 未配置态：引导配置（对齐 Figma 812:4043） */}
                <Empty className="border-none py-5">
                  <EmptyHeader>
                    <EmptyMedia />
                    <EmptyTitle>暂未配置 ClawPro 安全组，请选择创建方式：</EmptyTitle>
                    <EmptyDescription className="whitespace-nowrap">设置完成后，当前企业下所有 Agent 所在云服务器将默认使用该 ClawPro 安全组</EmptyDescription>
                  </EmptyHeader>
                  {/* 操作按钮（空状态标准按钮） */}
                  <EmptyContent>
                    <div className="flex items-center gap-3">
                      <Button
                        variant="claw-outline"
                        size="claw"
                        onClick={openCreateSecurityGroupDialog}
                      >
                        自定义规则
                      </Button>
                      <Button
                        variant="claw-outline"
                        size="claw"
                        onClick={() => openSelectSecurityGroupDialog("create")}
                      >
                        导入已有规则
                      </Button>
                    </div>
                  </EmptyContent>
                </Empty>
              </div>
            </SurfaceCard>
            )}
          </div>

          {/* ===== 模块二：出站规则 / 入站规则 ===== */}
          <div className="space-y-3">
            {/* 规则 Tab + 添加按钮（独立于表格容器之外） */}
            <div className="flex items-center justify-between">
              <SegmentGroup>
                <SegmentOption active={securityTab === "outbound"} onClick={() => setSecurityTab("outbound")} data-billing-exempt="">
                  出站规则
                </SegmentOption>
                <SegmentOption active={securityTab === "inbound"} onClick={() => setSecurityTab("inbound")} data-billing-exempt="">
                  入站规则
                </SegmentOption>
              </SegmentGroup>
              {currentSg ? (
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={() => {
                    setAddDraft({ policy: "允许" });
                    setShowAddDialog(securityTab);
                  }}
                >
                  {securityTab === "outbound" ? "添加出站规则" : "添加入站规则"}
                </Button>
              ) : (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-block">
                      <Button
                        variant="claw-outline"
                        size="claw-sm"
                        disabled
                      >
                        {securityTab === "outbound" ? "添加出站规则" : "添加入站规则"}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>请先配置安全组后再添加规则</TooltipContent>
                </Tooltip>
              )}
            </div>

            {/* 规则表 */}
            <SurfaceCard className="overflow-hidden">
              {currentSg ? (
                <RuleTableBody
                  rules={securityTab === "outbound" ? outboundRules : inboundRules}
                  type={securityTab}
                  readonly={false}
                  paginate={true}
                  onEdit={(rule, type) => {
                    setEditingRule({ id: rule.id, type });
                    setEditDraft(rule);
                  }}
                  onDelete={(rule, type) => {
                    setShowDeleteDialog({ id: rule.id, type, is_required: rule.is_required });
                  }}
                  onReorder={(reorderedRules) => {
                    if (securityTab === "outbound") {
                      setOutboundRules(reorderedRules);
                    } else {
                      setInboundRules(reorderedRules);
                    }
                  }}
                />
              ) : (
                <div className="px-6 py-10 flex flex-col items-center justify-center">
                  {securityTab === "outbound" ? (
                    <MetaText as="p" tone="weak" className="leading-relaxed">出站规则为空时，所有出站流量将被拒绝，Agent 将无法正常使用</MetaText>
                  ) : (
                    <MetaText as="p" tone="weak" className="leading-relaxed">入站规则为空时，所有入站流量将被拒绝</MetaText>
                  )}
                </div>
              )}
            </SurfaceCard>
          </div>

        </div>
        )}

        {activeTab === "vpc" && (
        <div>
          {/* 私有网络与子网顶部迁移提示条（存量企业 Agent VPC 迁移入口） */}
          {MigrationBanner}

          {/* 顶部说明区 */}
          <Alert variant="operation-info" className="-mt-2 mb-5 w-full">
            <AlertOperationInfoIcon />
            <AlertDescription>
              <ul className="space-y-1.5">
                <li className="flex gap-1.5">
                  <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                  <MetaText as="span" tone="inherit"><MetaMedium as="span" tone="inherit">私有网络（VPC）：</MetaMedium>配置 Agent 实例新建时使用的 VPC。「预设策略」默认自动分配可用 VPC，您也可以指定企业已有 VPC。</MetaText>
                </li>
                <li className="flex gap-1.5">
                  <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                  <MetaText as="span" tone="inherit"><MetaMedium as="span" tone="inherit">子网：</MetaMedium>选择「自动分配」VPC 时，系统将按可用区自动分配子网；选择已有 VPC 时，可手动配置各可用区子网，或选择「不分配」跳过该可用区部署。</MetaText>
                </li>
                <li className="flex gap-1.5">
                  <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                  <MetaText as="span" tone="inherit"><MetaMedium as="span" tone="inherit">策略生效：</MetaMedium>新建 Agent 实例将优先使用所选用户组的组织策略；本组未配置时，使用最近的上级用户组策略，均未配置时使用「预设策略」。</MetaText>
                </li>
                <li className="flex gap-1.5">
                  <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                  <MetaText as="span" tone="inherit"><MetaMedium as="span" tone="inherit">变更影响：</MetaMedium>修改网络策略后，仅影响后续新建的 Agent 实例，已有 Agent 实例网络保持不变。</MetaText>
                </li>
              </ul>
            </AlertDescription>
          </Alert>

          {/* VPC 列表 */}
          <div className="flex items-center justify-between mb-3">
            <PanelTitle>私有网络与子网配置</PanelTitle>
            <Button variant="claw-primary" size="claw-sm" onClick={() => {
              const placeholder: VpcListItem = {
                id: NEW_GROUP_VPC_ID,
                vpcId: "",
                vpcName: "",
                cidr: "",
                type: "group",
                associatedGroups: [],
                subnetStrategy: "specified",
                zoneSubnets: AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => { acc[z] = []; return acc; }, {}),
                instanceCount: 0,
              };
              setEditVpcDraft({
                vpcId: "",
                subnetStrategy: "specified",
                zoneSubnets: AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => { acc[z] = []; return acc; }, {}),
                associatedGroups: [],
              });
              setEditAutoCleaned(null);
              setZoneSubnetPickerOpen({});
              setShowEditVpcDialog(placeholder);
            }}>
              <Plus className="w-3.5 h-3.5" />
              添加网络组织策略
            </Button>
          </div>
          <SurfaceCard className="overflow-hidden">
            {/* 表格 */}
            <Table variant="white">
              <TableHeader>
                <TableRow>
                  <TableHead style={{ minWidth: 280 }}>私有网络（VPC）</TableHead>
                  <TableHead style={{ width: 140, minWidth: 140 }}>子网数量</TableHead>
                  <TableHead style={{ minWidth: 240 }}>应用范围</TableHead>
                  <TableHead style={{ width: 140, minWidth: 140 }}>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {/* 显示顺序：预设策略（type === "enterprise"）固定排在表格首行（默认主线），
                    其余组织策略保持原有顺序（不影响 vpcList 状态本身的存储顺序） */}
                {[...vpcList]
                  .sort((a, b) => {
                    const aDefault = a.type === "enterprise" ? 1 : 0;
                    const bDefault = b.type === "enterprise" ? 1 : 0;
                    return bDefault - aDefault;
                  })
                  .map((row) => {
                  const effectiveZoneSubnets = getEffectiveZoneSubnets(row);
                  const assignedZones = AVAILABLE_ZONES.filter((z) => (effectiveZoneSubnets[z] ?? []).length > 0);
                  const totalSubnets = assignedZones.reduce((sum, z) => sum + effectiveZoneSubnets[z].length, 0);

                  return (
                    <Fragment key={row.id}>
                      {/* 主行 — 整行可点击展开/收起 */}
                      <TableRow className="cursor-pointer" onClick={() => toggleVpcExpanded(row.id)}>
                        {/* VPC：展开箭头 + 名称 + 轻类型标签（视觉弱化） + id·CIDR */}
                        <TableCell className="whitespace-normal">
                          <div className="flex items-start gap-2 min-w-0">
                            <span
                              className="mt-0.5 shrink-0 text-[var(--text-weak)] transition-colors"
                              aria-label={expandedVpcIds.has(row.id) ? "收起详情" : "展开详情"}
                            >
                              {expandedVpcIds.has(row.id) ? (
                                <ChevronDown className="w-3.5 h-3.5" />
                              ) : (
                                <ChevronRight className="w-3.5 h-3.5" />
                              )}
                            </span>
                            <div className="flex flex-col gap-0.5 min-w-0">
                              {(() => {
                                const vpcDeleted = isVpcResourceDeleted(row);
                                const hasAnomaly = hasVpcRowAnomaly(row);
                                // VPC 自身已删除时，云端不再返回 vpcName，主标题以 "—" 占位；
                                // 副信息行仍保留 vpcId（用于资源治理与排查）。
                                const displayName = vpcDeleted ? "" : row.vpcName;
                                return (
                                  <>
                                    <div className="flex items-center gap-1.5 min-w-0">
                                      <BodyMedium tone="primary" className="truncate">{displayName || "—"}</BodyMedium>
                                      {hasAnomaly && (
                                        <Tooltip>
                                          <TooltipTrigger asChild>
                                            <span>
                                              <StatusTag variant="red">配置待更新</StatusTag>
                                            </span>
                                          </TooltipTrigger>
                                          <TooltipContent side="top" className="max-w-[300px]">
                                            <MetaText tone="inherit" className="leading-relaxed">当前配置中的部分 VPC / 子网已在云端删除，请重新配置。</MetaText>
                                          </TooltipContent>
                                        </Tooltip>
                                      )}
                                    </div>
                                    {row.subnetStrategy === "auto" ? (
                                      // 自动分配 VPC：与子网保持一致的端到端语义，不展示具体 vpc-id/cidr
                                      <MetaText tone="weak">自动分配</MetaText>
                                    ) : vpcDeleted ? (
                                      // VPC 已被删除：保留 vpcId 作为治理信息，cidr 占位为短横
                                      <MetaText tone="weak" className="font-mono">{row.vpcId} · —</MetaText>
                                    ) : (
                                      <MetaText tone="weak" className="font-mono">{row.vpcId} · {row.cidr}</MetaText>
                                    )}
                                  </>
                                );
                              })()}
                            </div>
                          </div>
                        </TableCell>
                        {/* 子网配置：仅展示已配置子网总数 */}
                        <TableCell>
                          {totalSubnets === 0 ? (
                            <MetaText tone="weak" className="whitespace-nowrap">未配置</MetaText>
                          ) : (
                            <BodyText as="span" tone="secondary" className="whitespace-nowrap">
                              <BodyMedium as="span" tone="secondary" className="tabular-nums">{totalSubnets}</BodyMedium>
                              <BodyText as="span" tone="muted"> 个</BodyText>
                            </BodyText>
                          )}
                        </TableCell>
                        {/* 策略：胶囊式徽章（对齐 ModelConfig 风格）；预设策略额外带 tooltip 说明 */}
                        <TableCell>
                          {row.type === "enterprise" ? (
                            <span className="inline-flex items-center gap-1 align-middle">
                              <Badge variant="outline">预设策略</Badge>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="inline-flex items-center text-[var(--text-weak)] hover:text-[var(--text-muted)] cursor-default">
                                    <Info className="w-3 h-3" />
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="max-w-[280px]">
                                  <MetaText tone="inherit" className="leading-relaxed">企业默认网络配置，适用于未分配组织用户，以及未匹配到用户组网络配置的场景。</MetaText>
                                </TooltipContent>
                              </Tooltip>
                            </span>
                          ) : (
                            // 多选应用范围（与平台策略 GroupBadges 同款）：单行展示完整组织路径，
                            // 放不下时折叠成「…共 N 个组织」，Tooltip 列出全部路径
                            <GroupBadges groupNames={row.associatedGroups ?? []} />
                          )}
                        </TableCell>
                        {/* 操作：TableActionCell 内置 flex wrapper（gap-6），统一 link 蓝色按钮 */}
                        <TableActionCell onClick={(e) => e.stopPropagation()}>
                            <Button
                              variant="link"
                              size="sm"
                              onClick={() => {
                                const isSystemDefault =
                                  row.type === "enterprise" && row.vpcId === AUTO_ASSIGNED_VPC.id;
                                // 进入编辑前，先剔除原配置中已被云控制台删除的 VPC / 子网，
                                // 避免在表单中回显失效资源；同时记录失效资源 ID 用于顶部提示文案。
                                const vpcDeleted = isVpcResourceDeleted(row);
                                const allSubnetsForVpc = MOCK_SUBNETS[row.vpcId] ?? [];
                                const removedSubnetIds: string[] = [];
                                const sanitizedZoneSubnets = AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => {
                                  const original = row.zoneSubnets[z] ?? [];
                                  if (vpcDeleted) {
                                    // VPC 已删除：连带清空所有可用区子网（必须重新选择 VPC）
                                    acc[z] = [];
                                    return acc;
                                  }
                                  // VPC 仍可用：仅过滤掉已删除子网，保留正常子网回显
                                  const filtered = original.filter((id) => {
                                    const s = allSubnetsForVpc.find((x) => x.id === id);
                                    if (!s || isSubnetResourceDeleted(s)) {
                                      if (id) removedSubnetIds.push(id);
                                      return false;
                                    }
                                    return true;
                                  });
                                  acc[z] = filtered;
                                  return acc;
                                }, {});
                                setEditVpcDraft({
                                  // VPC 已删除：vpcId 置空，强制用户重新选择
                                  vpcId: vpcDeleted ? "" : row.vpcId,
                                  // 仅系统默认企业级 VPC 保留原策略（可能是 auto）；
                                  // 其他场景强制为 specified
                                  subnetStrategy: isSystemDefault ? row.subnetStrategy : "specified",
                                  zoneSubnets: sanitizedZoneSubnets,
                                  associatedGroups: [...(row.associatedGroups ?? [])],
                                });
                                if (vpcDeleted || removedSubnetIds.length > 0) {
                                  setEditAutoCleaned({
                                    vpcId: vpcDeleted ? row.vpcId : null,
                                    subnetIds: removedSubnetIds,
                                  });
                                } else {
                                  setEditAutoCleaned(null);
                                }
                                setZoneSubnetPickerOpen({});
                                setShowEditVpcDialog(row);
                              }}
                            >
                              编辑
                            </Button>
                            {row.type === "group" ? (
                              <Button
                                variant="link"
                                size="sm"
                                onClick={() => setShowDeleteVpcDialog(row)}
                              >
                                删除
                              </Button>
                            ) : (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="inline-flex">
                                    <Button
                                      variant="link"
                                      size="sm"
                                      disabled
                                    >
                                      删除
                                    </Button>
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent><MetaText tone="inherit">预设策略不可删除</MetaText></TooltipContent>
                              </Tooltip>
                            )}
                        </TableActionCell>
                      </TableRow>

                      {/* 次级详情行：仅展开时渲染，按可用区纵向展示 */}
                      {expandedVpcIds.has(row.id) && (
                        <TableExpandedRow onClick={(e) => e.stopPropagation()}>
                          <TableCell colSpan={4} className="px-5 pb-4 pt-4 whitespace-normal">
                            <div className="rounded-[4px] border border-gray-200 bg-white px-4 py-3">
                              <div className="flex flex-col gap-2">
                                {AVAILABLE_ZONES.map((zone) => {
                                  const subnets = effectiveZoneSubnets[zone] ?? [];
                                  const isAssigned = subnets.length > 0;
                                  const rowIsAutoAssigned = row.subnetStrategy === "auto";
                                  return (
                                    <div key={zone} className="flex items-start gap-3 min-w-0">
                                      <CompactText tone="muted" className="shrink-0 w-16 leading-[22px]">
                                        {zone}
                                      </CompactText>
                                      {rowIsAutoAssigned ? (
                                        <Badge variant="secondary">自动分配</Badge>
                                      ) : isAssigned ? (
                                        <SubnetBadgesRow subnets={subnets} />
                                      ) : (
                                        <CompactText tone="weak">未分配</CompactText>
                                      )}
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          </TableCell>
                        </TableExpandedRow>
                      )}
                    </Fragment>
                  );
                })}
              </TableBody>
            </Table>

            {/* 底部提示 */}
            <div className="px-4 py-3 border-t border-gray-200">
              <MetaText as="p" className="leading-relaxed">
                如现有私有网络/子网不符合要求，可以去腾讯云控制台{" "}
                <a
                  href="https://console.cloud.tencent.com/vpc/vpc"
                  target="_blank"
                  rel="noopener noreferrer"
                  data-billing-exempt=""
                  className="text-blue-500 hover:text-[var(--text-brand)] underline inline-flex items-center gap-0.5"
                >
                  <MetaText as="span" tone="inherit">新建私有网络</MetaText>
                  <ExternalLink className="w-3 h-3" />
                </a>
                {" "}或{" "}
                <a
                  href="https://console.cloud.tencent.com/vpc/subnet"
                  target="_blank"
                  rel="noopener noreferrer"
                  data-billing-exempt=""
                  className="text-blue-500 hover:text-[var(--text-brand)] underline inline-flex items-center gap-0.5"
                >
                  <MetaText as="span" tone="inherit">新建子网</MetaText>
                  <ExternalLink className="w-3 h-3" />
                </a>
                。
              </MetaText>
            </div>
          </SurfaceCard>

          {/* ─── 编辑 VPC 弹窗 ─── */}
          <Dialog open={!!showEditVpcDialog} onOpenChange={(open) => {
            if (!open) {
              setShowEditVpcDialog(null);
              // 重置草稿，避免上次选择的用户组 / VPC / 子网在下次新增时残留
              setEditVpcDraft({ vpcId: "", subnetStrategy: "auto", zoneSubnets: {}, associatedGroups: [] });
              setEditAutoCleaned(null);
              setZoneSubnetPickerOpen({});
            }
          }}>
            <DialogContent
              className="sm:max-w-[720px] flex flex-col"
              style={{ maxHeight: "min(90vh, 780px)" }}
              onInteractOutside={(e) => e.preventDefault()}
              onEscapeKeyDown={(e) => e.preventDefault()}
            >
              <DialogHeader>
                <DialogTitle>
                  {showEditVpcDialog?.type === "enterprise"
                    ? "编辑预设策略"
                    : showEditVpcDialog?.id === NEW_GROUP_VPC_ID
                      ? "添加网络组织策略"
                      : "编辑组织策略"}
                </DialogTitle>
              </DialogHeader>

              <DialogBody className="px-6 flex-1">
                <div className="space-y-4">
                  {/* Alert 提示统一放在内容区最上方 */}
                  {showEditVpcDialog?.type === "enterprise" ? (
                    <Alert variant="info">
                      <Info />
                      <AlertDescription>
                        <ul className="space-y-1">
                          <li className="flex gap-1.5">
                            <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                            <MetaText as="span" tone="inherit">预设策略作为企业默认网络配置，适用于未分配组织用户，以及未匹配到组织策略的场景。</MetaText>
                          </li>
                          <li className="flex gap-1.5">
                            <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                            <MetaText as="span" tone="inherit">修改生效后，仅影响后续新建的 Agent 实例，已有 Agent 实例网络保持不变。</MetaText>
                          </li>
                        </ul>
                      </AlertDescription>
                    </Alert>
                  ) : showEditVpcDialog?.id === NEW_GROUP_VPC_ID ? (
                    <Alert variant="info">
                      <Info />
                      <AlertDescription>
                        <ul className="space-y-1">
                          <li className="flex gap-1.5">
                            <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                            <MetaText as="span" tone="inherit">为用户组添加组织策略后，新建 Agent 实例时若选择该用户组，将优先使用此策略。</MetaText>
                          </li>
                          <li className="flex gap-1.5">
                            <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                            <MetaText as="span" tone="inherit">仅影响后续新建的 Agent 实例，已有 Agent 实例网络保持不变。</MetaText>
                          </li>
                        </ul>
                      </AlertDescription>
                    </Alert>
                  ) : (
                    <Alert variant="info">
                      <Info />
                      <AlertDescription>
                        <ul className="space-y-1">
                          <li className="flex gap-1.5">
                            <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                            <MetaText as="span" tone="inherit">可修改该组织策略的应用范围、VPC 和子网。</MetaText>
                          </li>
                          <li className="flex gap-1.5">
                            <MetaText as="span" tone="inherit" className="shrink-0">•</MetaText>
                            <MetaText as="span" tone="inherit">修改生效后，仅影响后续新建的 Agent 实例，已有 Agent 实例网络保持不变。</MetaText>
                          </li>
                        </ul>
                      </AlertDescription>
                    </Alert>
                  )}
                  {editAutoCleaned && showEditVpcDialog?.id !== NEW_GROUP_VPC_ID && (
                    <Alert variant="warning">
                      <CircleAlert />
                      <AlertDescription className="break-all">
                        <MetaText as="span" tone="inherit">
                          {(() => {
                            const segs: string[] = [];
                            if (editAutoCleaned.vpcId) {
                              segs.push(` VPC ${editAutoCleaned.vpcId}`);
                            }
                            if (editAutoCleaned.subnetIds.length > 0) {
                              segs.push(`子网 ${editAutoCleaned.subnetIds.join("、")}`);
                            }
                            return `检测到原配置中的${segs.join("、")} 已从腾讯云控制台被删除，已自动从本次编辑中移除。`;
                          })()}
                        </MetaText>
                      </AlertDescription>
                    </Alert>
                  )}

                  {(() => {
                    const isEnterprise = showEditVpcDialog?.type === "enterprise";
                    const isAutoAssigned =
                      isEnterprise &&
                      editVpcDraft.vpcId === AUTO_ASSIGNED_VPC.id &&
                      editVpcDraft.subnetStrategy === "auto";

                    return (
                      <>
                        {/* ── 应用范围（仅组织网络） ── */}
                        {showEditVpcDialog?.type === "group" && (
                          <div className="space-y-2">
                            <div className="flex items-center gap-1">
                              <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span className="inline-flex items-center text-[var(--text-weak)] hover:text-[var(--text-muted)] cursor-help">
                                      <Info className="w-3.5 h-3.5" />
                                    </span>
                                  </TooltipTrigger>
                                  <TooltipContent side="top" className="max-w-[280px] normal-case tracking-normal">
                                    <MetaText tone="inherit" className="leading-relaxed">新建 Agent 实例时若选择该用户组，将使用此组织策略。</MetaText>
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            </div>
                            {(() => {
                              const currentRowNames = new Set(showEditVpcDialog?.associatedGroups ?? []);
                              const occupiedNames = new Set<string>();
                              vpcList.forEach((r) => {
                                if (r.id === showEditVpcDialog?.id) return;
                                (r.associatedGroups ?? []).forEach((name) => occupiedNames.add(name));
                              });
                              currentRowNames.forEach((name) => occupiedNames.delete(name));
                              // 仅展示部门 / 自定义组织（屏蔽用户组桶 oneid-group）
                              const visibleGroups = ALL_GROUPS_SHARED.filter(
                                (g) => g.source === "oneid-dept" || g.source === "manual"
                              );
                              const nameToNode = new Map(visibleGroups.map((g) => [g.name, g]));
                              const idToName = new Map(visibleGroups.map((g) => [g.id, g.name]));
                              const disabledIdSet = new Set<string>();
                              occupiedNames.forEach((name) => {
                                const node = nameToNode.get(name);
                                if (!node) return;
                                disabledIdSet.add(node.id);
                              });
                              const isNewRow = showEditVpcDialog?.id === NEW_GROUP_VPC_ID;
                              const disabledTooltipText = isNewRow
                                ? "该用户组已配置策略，请编辑已有策略。"
                                : "该用户组已配置策略，请选择其他用户组。";
                              // name ↔ id 互转（外部 state 仍用 name 数组）
                              const selectedIds: string[] = [];
                              editVpcDraft.associatedGroups.forEach((name) => {
                                const node = nameToNode.get(name);
                                if (node) selectedIds.push(node.id);
                              });
                              return (
                                <GroupSelect
                                  groups={visibleGroups}
                                  selectedIds={selectedIds}
                                  disabledIds={Array.from(disabledIdSet)}
                                  disabledTooltip={disabledTooltipText}
                                  placeholder="请选择组织"
                                  onChange={(ids) => {
                                    const names: string[] = [];
                                    ids.forEach((id) => {
                                      const n = idToName.get(id);
                                      if (n) names.push(n);
                                    });
                                    setEditVpcDraft((prev) => ({ ...prev, associatedGroups: names }));
                                  }}
                                />
                              );
                            })()}
                          </div>
                        )}

                        {/* ── 私有网络（VPC） ── */}
                        <div className="space-y-2">
                          <MetaMedium as="label" tone="secondary">私有网络（VPC）</MetaMedium>
                          {(() => {
                            const triggerVpc = isAutoAssigned
                              ? AUTO_ASSIGNED_VPC
                              : MOCK_VPCS.find((v) => v.id === editVpcDraft.vpcId);
                            const totalCount = MOCK_VPCS.length + (isEnterprise ? 1 : 0);
                            return (
                              <Popover open={editVpcPickerOpen} onOpenChange={(v) => { setEditVpcPickerOpen(v); if (v) setVpcSearchQuery(""); }}>
                                <PopoverTrigger asChild>
                                  <button
                                    type="button"
                                    data-state={editVpcPickerOpen ? "open" : "closed"}
                                    className="w-full h-9 px-3 py-1 rounded-[4px] border border-gray-200 bg-white hover:border-blue-500 focus:border-blue-500 focus:outline-none data-[state=open]:border-blue-500 transition-colors flex items-center justify-between gap-2"
                                  >
                                    {triggerVpc ? (
                                      isAutoAssigned ? (
                                        <CompactText tone="emphasis">自动分配</CompactText>
                                      ) : (
                                        <span className="flex items-center gap-2 min-w-0">
                                          <CompactText tone="emphasis" className="shrink-0">{triggerVpc.id}</CompactText>
                                          <CompactText tone="weak" className="shrink-0">|</CompactText>
                                          <CompactText tone="emphasis" className="truncate">{triggerVpc.name}</CompactText>
                                          <CompactText tone="weak" className="shrink-0">|</CompactText>
                                          <CompactText tone="emphasis" className="shrink-0">{triggerVpc.cidr}</CompactText>
                                        </span>
                                      )
                                    ) : (
                                      <CompactText tone="weak">请选择 VPC</CompactText>
                                    )}
                                    <ChevronDown className={`w-4 h-4 text-[var(--text-weak)] shrink-0 transition-transform ${editVpcPickerOpen ? "rotate-180" : ""}`} />
                                  </button>
                                </PopoverTrigger>
                                <PopoverContent
                                  className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)]"
                                  style={{ width: "var(--radix-popover-trigger-width)" }}
                                  align="start"
                                  sideOffset={4}
                                >
                                  {/* 搜索框 */}
                                  <div className="p-2 pb-1">
                                    <div className="relative">
                                      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                                      <Input
                                        className="h-8 pl-8 text-sm"
                                        placeholder="搜索 VPC ID / 名称..."
                                        value={vpcSearchQuery}
                                        onChange={(e) => setVpcSearchQuery(e.target.value)}
                                      />
                                    </div>
                                  </div>
                                  {/* 选项列表 */}
                                  <div className="max-h-72 overflow-y-auto p-2 pt-1">
                                    {(() => {
                                      const q = vpcSearchQuery.toLowerCase().trim();
                                      const filteredVpcs = q ? MOCK_VPCS.filter((v) => v.id.toLowerCase().includes(q) || v.name.toLowerCase().includes(q) || v.cidr.includes(q)) : MOCK_VPCS;
                                      const showAuto = isEnterprise && (!q || "自动分配".includes(q));
                                      if (!showAuto && filteredVpcs.length === 0) {
                                        return <div className="py-3 text-center"><MetaText tone="weak">未找到匹配的 VPC</MetaText></div>;
                                      }
                                      return (
                                        <>
                                          {showAuto && (
                                            <div
                                              className={`relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm ${isAutoAssigned ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "text-[var(--text-title)] hover:bg-[#FAFAFA]"}`}
                                              onClick={() => {
                                                setEditVpcDraft((prev) => ({
                                                  ...prev,
                                                  vpcId: AUTO_ASSIGNED_VPC.id,
                                                  subnetStrategy: "auto",
                                                  zoneSubnets: AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => { acc[z] = []; return acc; }, {}),
                                                }));
                                                setEditVpcPickerOpen(false);
                                              }}
                                            >
                                              <span className="flex-1 truncate">自动分配</span>
                                              {isAutoAssigned && <Check className="w-4 h-4 text-blue-500 shrink-0" />}
                                            </div>
                                          )}
                                          {filteredVpcs.map((vpc) => {
                                            const selected = !isAutoAssigned && editVpcDraft.vpcId === vpc.id;
                                            return (
                                              <div
                                                key={vpc.id}
                                                className={`relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm ${selected ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "text-[var(--text-title)] hover:bg-[#FAFAFA]"}`}
                                                onClick={() => {
                                                  setEditVpcDraft((prev) => ({
                                                    ...prev,
                                                    vpcId: vpc.id,
                                                    subnetStrategy: "specified",
                                                    zoneSubnets: AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => { acc[z] = []; return acc; }, {}),
                                                  }));
                                                  setEditVpcPickerOpen(false);
                                                }}
                                              >
                                                <span className="flex-1 min-w-0 flex items-center gap-2 truncate">
                                                  <CompactText tone={selected ? "brand" : "emphasis"} className={`shrink-0 ${selected ? "font-medium" : ""}`}>{vpc.id}</CompactText>
                                                  <CompactText tone="weak" className="shrink-0">|</CompactText>
                                                  <CompactText tone={selected ? "brand" : "emphasis"} className={`truncate ${selected ? "font-medium" : ""}`}>{vpc.name}</CompactText>
                                                  <CompactText tone="weak" className="shrink-0">|</CompactText>
                                                  <CompactText tone={selected ? "brand" : "emphasis"} className={`shrink-0 ${selected ? "font-medium" : ""}`}>{vpc.cidr}</CompactText>
                                                </span>
                                                {selected && <Check className="w-4 h-4 text-blue-500 shrink-0" />}
                                              </div>
                                            );
                                          })}
                                        </>
                                      );
                                    })()}
                                  </div>
                                  {/* 底部计数 */}
                                  <div className="mx-2 border-t border-[#EAEEF4] px-1 h-9 flex items-center">
                                    <MetaText tone="weak">共 {totalCount} 条</MetaText>
                                  </div>
                                </PopoverContent>
                              </Popover>
                            );
                          })()}
                        </div>

                        {/* ── 可用区子网配置 ── */}
                        {(isAutoAssigned || editVpcDraft.vpcId || showEditVpcDialog?.id === NEW_GROUP_VPC_ID || !!editAutoCleaned?.vpcId) && (
                          <div className="space-y-2">
                            <div className="flex items-center justify-between">
                              <MetaMedium as="label" tone="secondary">可用区及子网配置</MetaMedium>
                              {!isAutoAssigned && (
                                <MetaText>
                                  已分配 {AVAILABLE_ZONES.filter((z) => (editVpcDraft.zoneSubnets[z] ?? []).length > 0).length} / {AVAILABLE_ZONES.length} 个可用区
                                </MetaText>
                              )}
                            </div>
                            <div className="space-y-3">
                              {AVAILABLE_ZONES.map((zone) => {
                                if (isAutoAssigned) {
                                  return (
                                    <div
                                      key={zone}
                                      className="rounded-[4px] border border-border bg-white"
                                    >
                                      <div className="flex items-center gap-3 px-4 py-3 min-w-0">
                                        <BodyMedium tone="primary" className="shrink-0">{zone}</BodyMedium>
                                        <MetaText>自动分配</MetaText>
                                      </div>
                                    </div>
                                  );
                                }

                                const selectedIds = editVpcDraft.zoneSubnets[zone] ?? [];
                                const zoneAllSubnets = getSubnetsByVpcZone(editVpcDraft.vpcId, zone);
                                const selectedSubnets = selectedIds
                                  .map((id) => zoneAllSubnets.find((s) => s.id === id))
                                  .filter((s): s is SubnetEntity => !!s);
                                const selectableSubnets = zoneAllSubnets.filter((s) => !selectedIds.includes(s.id));
                                const pickerKey = `${showEditVpcDialog?.id ?? ""}#${zone}`;
                                const isPickerOpen = !!zoneSubnetPickerOpen[pickerKey];
                                const isUnassigned = selectedIds.length === 0;

                                return (
                                  <div
                                    key={zone}
                                    className="rounded-[4px] border border-border bg-white"
                                  >
                                    {/* 可用区头部 */}
                                    <div className="flex items-center justify-between px-4 py-2.5 border-b border-[#f0f0f0]">
                                      <div className="flex items-center gap-2">
                                        <BodyMedium tone="primary">{zone}</BodyMedium>
                                        {isUnassigned ? (
                                          <MetaText>未分配</MetaText>
                                        ) : (
                                          <MetaText>{selectedSubnets.length} 个子网</MetaText>
                                        )}
                                      </div>
                                      <div className="flex items-center gap-1">
                                        {!isUnassigned && (
                                          <Button
                                            variant="claw-outline"
                                            size="claw-sm"
                                            onClick={() => setEditVpcDraft((prev) => ({
                                              ...prev,
                                              zoneSubnets: { ...prev.zoneSubnets, [zone]: [] },
                                            }))}
                                          >
                                            清空
                                          </Button>
                                        )}
                                      </div>
                                    </div>

                                    {/* 子网内容区 */}
                                    {!isUnassigned && (
                                      <div className="px-4 py-3 space-y-2">
                                        {/* 已选子网 chips — 白底灰框标准 tag */}
                                        <div className="flex flex-wrap gap-1.5">
                                          {selectedSubnets.map((subnet) => {
                                            const remainingLow = subnet.remainingIp / subnet.totalIp < 0.1;
                                            return (
                                              <div
                                                key={subnet.id}
                                                className="inline-flex items-center gap-2 pl-2.5 pr-1 py-1 rounded-[4px] bg-white border border-border"
                                              >
                                                <div className="flex items-center gap-1.5 min-w-0">
                                                  <MetaText tone="primary" className="font-medium truncate max-w-[140px]">{subnet.name}</MetaText>
                                                  <MetaText>{subnet.cidr}</MetaText>
                                                  <MetaText className={`tabular-nums ${remainingLow ? "text-amber-500" : ""}`}>
                                                    · 剩余 IP {subnet.remainingIp}/{subnet.totalIp}
                                                  </MetaText>
                                                </div>
                                                <button
                                                  type="button"
                                                  onClick={() => setEditVpcDraft((prev) => ({
                                                    ...prev,
                                                    zoneSubnets: {
                                                      ...prev.zoneSubnets,
                                                      [zone]: (prev.zoneSubnets[zone] ?? []).filter((id) => id !== subnet.id),
                                                    },
                                                  }))}
                                                  className="w-4 h-4 flex items-center justify-center rounded-sm text-[var(--text-muted)] hover:text-[var(--text-title)] hover:bg-[#f5f5f5] transition-colors"
                                                  title="移除此子网"
                                                >
                                                  <X className="w-3 h-3" />
                                                </button>
                                              </div>
                                            );
                                          })}
                                        </div>
                                      </div>
                                    )}

                                    {/* 添加子网 */}
                                    <div className={`px-4 ${isUnassigned ? "py-3" : "pb-3"}`}>
                                      <Popover
                                        open={isPickerOpen}
                                        onOpenChange={(o) => setZoneSubnetPickerOpen((prev) => ({ ...prev, [pickerKey]: o }))}
                                      >
                                        <PopoverTrigger asChild>
                                          <button
                                            type="button"
                                            disabled={selectableSubnets.length === 0}
                                            className={`h-8 w-full flex items-center justify-center gap-1.5 rounded-[4px] border border-dashed transition-colors ${
                                              selectableSubnets.length === 0
                                                ? "border-border cursor-not-allowed"
                                                : "border-[#d3d6db] hover:border-[#355EF1]"
                                            }`}
                                            title={!editVpcDraft.vpcId ? "请先选择 VPC" : selectableSubnets.length === 0 ? "该可用区下无可添加的子网" : "添加子网"}
                                          >
                                            <Plus className={`w-3.5 h-3.5 ${selectableSubnets.length === 0 ? "text-[var(--text-weak)]" : "text-[var(--text-secondary)]"}`} />
                                            <MetaText tone={selectableSubnets.length === 0 ? "weak" : "secondary"}>
                                              {!editVpcDraft.vpcId
                                                ? "请先选择 VPC"
                                                : selectableSubnets.length === 0
                                                  ? (zoneAllSubnets.length === 0 ? "该可用区暂无子网" : "已全部添加")
                                                  : "添加子网"}
                                            </MetaText>
                                          </button>
                                        </PopoverTrigger>
                                        <PopoverContent
                                          className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)] flex flex-col max-h-[320px]"
                                          style={{ width: "var(--radix-popover-trigger-width)" }}
                                          align="start"
                                          sideOffset={4}
                                        >
                                          {/* 搜索框 */}
                                          <div className="p-2 pb-1 shrink-0">
                                            <div className="relative">
                                              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                                              <Input
                                                className="h-8 pl-8 text-sm"
                                                placeholder="搜索子网 ID / 名称..."
                                                value={subnetSearchQueries[pickerKey] || ""}
                                                onChange={(e) => setSubnetSearchQueries((prev) => ({ ...prev, [pickerKey]: e.target.value }))}
                                              />
                                            </div>
                                          </div>
                                          {/* 选项列表 */}
                                          <div className="flex-1 min-h-0 overflow-y-auto overscroll-contain p-2 pt-1" onWheel={(e) => e.stopPropagation()}>
                                            {(() => {
                                              const sq = (subnetSearchQueries[pickerKey] || "").toLowerCase().trim();
                                              const filtered = sq ? selectableSubnets.filter((s) => s.id.toLowerCase().includes(sq) || s.name.toLowerCase().includes(sq) || s.cidr.includes(sq)) : selectableSubnets;
                                              if (filtered.length === 0) {
                                                return <div className="py-3 text-center"><MetaText tone="weak">未找到匹配的子网</MetaText></div>;
                                              }
                                              return filtered.map((subnet) => {
                                                const remainingLow = subnet.remainingIp / subnet.totalIp < 0.1;
                                                return (
                                                  <div
                                                    key={subnet.id}
                                                    className="relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm text-[var(--text-title)] hover:bg-[#FAFAFA]"
                                                    onClick={() => {
                                                      setEditVpcDraft((prev) => ({
                                                        ...prev,
                                                        zoneSubnets: {
                                                          ...prev.zoneSubnets,
                                                          [zone]: [...(prev.zoneSubnets[zone] ?? []), subnet.id],
                                                        },
                                                      }));
                                                      setZoneSubnetPickerOpen((prev) => ({ ...prev, [pickerKey]: false }));
                                                    }}
                                                  >
                                                    <span className="flex-1 min-w-0 flex items-center gap-2">
                                                      <CompactText tone="emphasis" className="shrink-0">{subnet.id}</CompactText>
                                                      <CompactText tone="weak" className="shrink-0">|</CompactText>
                                                      <CompactText tone="emphasis" className="truncate">{subnet.name}</CompactText>
                                                      <CompactText tone="weak" className="shrink-0">|</CompactText>
                                                      <CompactText tone="emphasis" className="shrink-0">{subnet.cidr}</CompactText>
                                                    </span>
                                                    <MetaText className={`shrink-0 tabular-nums ml-2 ${remainingLow ? "text-amber-500" : ""}`}>
                                                      剩余 IP {subnet.remainingIp}/{subnet.totalIp}
                                                    </MetaText>
                                                  </div>
                                                );
                                              });
                                            })()}
                                          </div>
                                          {/* 底部计数 */}
                                          <div className="shrink-0 mx-2 border-t border-[#EAEEF4] px-1 h-9 flex items-center">
                                            <MetaText tone="weak">共 {selectableSubnets.length} 条</MetaText>
                                          </div>
                                        </PopoverContent>
                                      </Popover>
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </>
                    );
                  })()}
                </div>
              </DialogBody>

              {/* 底部按钮 */}
              <DialogFooter>
                <Button variant="outline" onClick={() => setShowEditVpcDialog(null)}>取消</Button>
                <Button
                  variant="dialog-confirm"
                  onClick={() => {
                    if (!showEditVpcDialog) return;
                    const isEnterprise = showEditVpcDialog.type === "enterprise";
                    const isAutoAssigned =
                      isEnterprise &&
                      editVpcDraft.vpcId === AUTO_ASSIGNED_VPC.id &&
                      editVpcDraft.subnetStrategy === "auto";
                    // 校验：组织网络必须选择应用范围
                    if (!isEnterprise && editVpcDraft.associatedGroups.length === 0) {
                      toast.error("请选择应用范围");
                      return;
                    }
                    // 校验：非自动分配场景下必须已选 VPC
                    if (!isAutoAssigned && !editVpcDraft.vpcId) {
                      toast.error("请先选择私有网络（VPC）");
                      return;
                    }
                    // 校验：指定子网模式下至少有一个可用区分配了子网
                    if (editVpcDraft.subnetStrategy === "specified") {
                      const hasAny = AVAILABLE_ZONES.some((z) => (editVpcDraft.zoneSubnets[z] ?? []).length > 0);
                      if (!hasAny) {
                        toast.error("请至少为一个可用区配置子网");
                        return;
                      }
                    }
                    const vpc = MOCK_VPCS.find((v) => v.id === editVpcDraft.vpcId);
                    const isNewRow = showEditVpcDialog.id === NEW_GROUP_VPC_ID;
                    if (isNewRow) {
                      // 新增：校验所选用户组未被其他行占用
                      const selectedSet = new Set(editVpcDraft.associatedGroups);
                      const conflictGroup = vpcList
                        .flatMap((r) => r.associatedGroups ?? [])
                        .find((g) => selectedSet.has(g));
                      if (conflictGroup) {
                        toast.error(`用户组「${conflictGroup}」已绑定组织策略`);
                        return;
                      }
                      const displayName = editVpcDraft.associatedGroups[0] ?? "";
                      const newItem: VpcListItem = {
                        id: "vpc-row-" + Date.now(),
                        vpcId: editVpcDraft.vpcId,
                        vpcName: vpc?.name || `${displayName}网络`,
                        cidr: vpc?.cidr || "",
                        type: "group",
                        associatedGroups: [...editVpcDraft.associatedGroups],
                        subnetStrategy: editVpcDraft.subnetStrategy,
                        zoneSubnets: AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => {
                          acc[z] = [...(editVpcDraft.zoneSubnets[z] ?? [])];
                          return acc;
                        }, {}),
                        instanceCount: 0,
                      };
                      // 校验通过 → 挂起待二次确认
                      setPendingVpcSave({
                        mode: "newGroup",
                        execute: () => {
                          setVpcList((prev) => [...prev, newItem]);
                          setShowEditVpcDialog(null);
                        },
                      });
                      return;
                    }
                    // 编辑：构造新行 + 挂起待二次确认
                    const targetId = showEditVpcDialog.id;
                    setPendingVpcSave({
                      mode: isEnterprise ? "editGlobal" : "editGroup",
                      execute: () => {
                        setVpcList((prev) => prev.map((r) =>
                          r.id === targetId
                            ? {
                                ...r,
                                vpcId: editVpcDraft.vpcId,
                                // auto 模式（仅预设策略可选）锁定为"企业默认网络"，
                                // 避免回写具体 VPC 的 name/cidr 与"自动分配"语义矛盾
                                vpcName: editVpcDraft.subnetStrategy === "auto"
                                  ? "企业默认网络"
                                  : (vpc?.name || r.vpcName),
                                cidr: editVpcDraft.subnetStrategy === "auto"
                                  ? r.cidr
                                  : (vpc?.cidr || r.cidr),
                                subnetStrategy: editVpcDraft.subnetStrategy,
                                // 组织网络允许切换应用范围；企业级不变
                                associatedGroups: isEnterprise ? r.associatedGroups : [...editVpcDraft.associatedGroups],
                                zoneSubnets: editVpcDraft.subnetStrategy === "specified"
                                  ? AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => {
                                      acc[z] = [...(editVpcDraft.zoneSubnets[z] ?? [])];
                                      return acc;
                                    }, {})
                                  : AVAILABLE_ZONES.reduce<Record<string, string[]>>((acc, z) => { acc[z] = []; return acc; }, {}),
                              }
                            : r
                        ));
                        setShowEditVpcDialog(null);
                      },
                    });
                  }}
                >
                  保存
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* ─── 删除确认弹窗 - 警示弹窗规范 ─── */}
          <AlertDialog open={!!showDeleteVpcDialog} onOpenChange={(open) => !open && setShowDeleteVpcDialog(null)}>
            <AlertDialogContent className="sm:max-w-[420px]">
              <button
                type="button"
                aria-label="关闭"
                onClick={() => setShowDeleteVpcDialog(null)}
                className="absolute top-5 right-5 flex items-center justify-center size-5 rounded-sm text-[var(--text-muted)] transition-colors hover:text-[var(--text-title)] focus:outline-none"
              >
                <X className="size-5" />
                <span className="sr-only">关闭</span>
              </button>
              <AlertDialogHeader>
                <AlertDialogTitle asChild>
                  <PanelTitle>确认删除</PanelTitle>
                </AlertDialogTitle>
              </AlertDialogHeader>
              <AlertDialogDescription asChild>
                <BodyText as="p" tone="primary">
                  该操作仅影响后续新建的 Agent 实例，已有 Agent 实例网络保持不变。
                </BodyText>
              </AlertDialogDescription>
              <AlertDialogFooter>
                <Button variant="claw-outline" size="claw-sm" onClick={() => setShowDeleteVpcDialog(null)}>取消</Button>
                <Button
                  variant="destructive"
                  size="claw-sm"
                  onClick={() => {
                    if (!showDeleteVpcDialog) return;
                    setVpcList((prev) => prev.filter((r) => r.id !== showDeleteVpcDialog.id));
                    setShowDeleteVpcDialog(null);
                    toast.success("已删除该组织策略");
                  }}
                >
                  确认删除
                </Button>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          {/* ─── 保存确认弹窗（新增组织 / 编辑预设 / 编辑组织 三种模式共用） ─── */}
          <Dialog open={!!pendingVpcSave} onOpenChange={(open) => !open && setPendingVpcSave(null)}>
            <DialogContent
              className="sm:max-w-md"
              style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
            >
              {(() => {
                if (!pendingVpcSave) return null;
                const titleMap = {
                  newGroup: "确认添加",
                  editGlobal: "确认修改",
                  editGroup: "确认修改",
                } as const;
                const bodyMap = {
                  newGroup: "新增后，若用户在新建 Agent 实例时选择该用户组，则实例将使用所选 VPC 与子网。",
                  editGlobal: "修改后，未分配组织用户及未匹配到组织策略的用户，后续新建 Agent 实例时将使用新的预设策略。已有 Agent 实例不受影响。",
                  editGroup: "修改后，后续新建 Agent 实例时，将按新的应用范围、VPC 与子网配置生效。已有 Agent 实例不受影响。",
                } as const;
                const confirmTextMap = {
                  newGroup: "确认",
                  editGlobal: "确认",
                  editGroup: "确认",
                } as const;
                const successTextMap = {
                  newGroup: "已新增组织策略",
                  editGlobal: "已更新预设策略",
                  editGroup: "已更新组织策略",
                } as const;
                const mode = pendingVpcSave.mode;
                return (
                  <>
                    <DialogHeader>
                      <DialogTitle>{titleMap[mode]}</DialogTitle>
                    </DialogHeader>
                    <DialogBody className="px-6 flex-1">
                      <p className="text-sm text-[var(--text-title)] leading-relaxed">{bodyMap[mode]}</p>
                    </DialogBody>
                    <DialogFooter>
                      <Button variant="outline" onClick={() => setPendingVpcSave(null)}>取消</Button>
                      <Button
                        variant="dialog-confirm"
                        onClick={() => {
                          pendingVpcSave.execute();
                          setPendingVpcSave(null);
                          toast.success(successTextMap[mode]);
                        }}
                      >
                        {confirmTextMap[mode]}
                      </Button>
                    </DialogFooter>
                  </>
                );
              })()}
            </DialogContent>
          </Dialog>
        </div>
        )}

        {activeTab === "public" && (
        <div className="page-enter">
          <Empty className="border-0">
            <EmptyHeader>
              <EmptyMedia>
                <img src="/assets/admin-resource-management/empty-no-data.png" alt="" />
              </EmptyMedia>
              <EmptyTitle>公网配置已迁移至资源配置</EmptyTitle>
            <EmptyDescription className="whitespace-nowrap">
              公网 IP、公网计费模式和带宽上限已统一纳入资源配置策略管理，
              <span
                className="text-[var(--cp-brand-blue)] cursor-pointer hover:underline inline-flex items-center gap-0.5"
                onClick={() => navigate("/admin/agent-template")}
              >
                前往资源配置
                <ExternalLink className="w-3 h-3" />
              </span>
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
        </div>
        )}

        {activeTab === "coming" && (
          <div className="grid grid-cols-3 items-stretch gap-5">
            {MORE_FEATURE_CARDS.map((card) => (
              <div
                key={card.title}
                className="min-h-[102px] rounded-[4px] border border-border bg-white px-6 py-5"
              >
                <div className="flex items-start gap-[14px]">
                  <img src={card.iconSrc} alt="" aria-hidden="true" className="h-9 w-9 shrink-0" />
                  <div className="min-w-0 flex-1">
                    <CardTitle tone="emphasis">{card.title}</CardTitle>
                    <MetaText as="p" className="mt-1">
                      {card.description}
                    </MetaText>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

      </div>

      {/* ─── 新建安全组弹窗 ──────────────────────────────────────────────────────────────── */}
      <CreateSecurityGroupDialog
        open={showCreateSgDialog}
        checkedOptions={createSgCheckedOptions}
        draft={createSgDraft}
        previewTab={createSgPreviewTab}
        onOpenChange={handleCreateSecurityGroupDialogOpenChange}
        onOptionToggle={handleCreateSgOptionToggle}
        onNameChange={(name) => setCreateSgDraft((prev) => ({ ...prev, name }))}
        onRemarkChange={(remark) => setCreateSgDraft((prev) => ({ ...prev, remark }))}
        onPreviewTabChange={setCreateSgPreviewTab}
        onCancel={closeCreateSecurityGroupDialog}
        onConfirm={() => {
          const trimmedName = createSgDraft.name.trim();
          if (!trimmedName) {
            toast.error("请输入安全组名称");
            return;
          }
          if (!isValidClawproSgName(trimmedName)) {
            toast.error("安全组名称格式不合法：" + CLAWPRO_SG_NAME_RULE);
            return;
          }

          const { inbound: initialInbound, outbound: initialOutbound } = buildRulesFromOptions(createSgCheckedOptions);

          // [004 · BUG FIX 2026-05-05] 用户新建 ClawPro 安全组：
          //   - ClawPro 安全组 name/remark = 用户自定义（如 "mya"）
          //   - ClawPro 安全组对外没有 ID，name 即唯一标识（API 也以 name 为参数）
          //   - 云端 sg name 不复用用户起名，统一走 K4 默认格式 clawpro-sg-{域名}-default-{序号}
          //   - 云端 sgId 是新生成的 8 位格式（模拟云端 API 返回）
          //
          // 【陷阱】不能调用 applyCurrentSecurityGroup(newSg)：
          //   该函数的"初始化分支"（prev == null 时）会硬编码 name="ClawPro-Default" / remark="Agent 默认安全组"
          //   覆盖用户填的 name/remark。该分支专为"路径 B · 从云端导入规则"设计——
          //   云端 sg name 不可复用，所以走默认值。但路径 A 必须保留用户起名。
          //   修复方案：直接 setCurrentSg(newSg) 跳过初始化分支，并手动同步 inboundRules/outboundRules
          const cloudSgId = generateRandomSgId();
          const newSg: SecurityGroup = {
            name: trimmedName,
            remark: createSgDraft.remark,
            inboundCount: initialInbound.length,
            outboundCount: initialOutbound.length,
            inboundRules: initialInbound,
            outboundRules: initialOutbound,
            cloudSgs: [{ sgId: cloudSgId, cloudSgName: DEFAULT_CLOUD_SECURITY_GROUP_NAME, seq: 1 }],
          };
          MOCK_SECURITY_GROUPS.unshift(newSg);
          setCurrentSg(newSg);
          setInboundRules(initialInbound);
          setOutboundRules(initialOutbound);
          closeCreateSecurityGroupDialog();
          toast.success("创建成功，当前企业下所有 Agent 将使用该 ClawPro 安全组");
        }}
      />

      {/* ─── 切换安全组大弹窗（上下布局） ──────────────────────────────────────────────────────────────── */}
      {renderSelectExistingSecurityGroupDialog({
        open: isSgDialogOpen,
        searchKeyword: sgSearchKeyword,
        selectedSecurityGroup: sgDialogPreviewSecurityGroup,
        previewTab: sgDialogTab,
        candidateSecurityGroups: pagedSelectableSecurityGroups,
        candidatePage: selectableSecurityGroupCurrentPage,
        candidateTotalPages: selectableSecurityGroupTotalPages,
        shouldShowPanelAccessAssist: shouldShowSgDialogPanelAccessAssist,
        isPanelAccessStaged: isSgDialogPanelAccessStaged,
        onOpenChange: (open) => {
          if (!open) {
            closeSelectSecurityGroupDialog();
          }
        },
        onSearchChange: (value) => {
          setSgSearchKeyword(value);
          setSgDialogCandidatePage(1);
        },
        onClearSearch: () => {
          setSgSearchKeyword("");
          setSgDialogCandidatePage(1);
        },
        onSelectSecurityGroup: (sg) => {
          setSgDialogSelected(sg);
          setSgDialogTab("outbound");
          setSgSearchKeyword("");
          setSgDialogCandidatePage(1);
          setIsSgDialogPanelAccessStaged(false);
        },
        onCandidatePageChange: setSgDialogCandidatePage,
        onTogglePanelAccessStaged: () => {
          if (!isSgDialogPanelAccessStaged) {
            setSgDialogTab("inbound");
          }
          setIsSgDialogPanelAccessStaged((prev) => !prev);
        },
        onPreviewTabChange: setSgDialogTab,
        onCancel: closeSelectSecurityGroupDialog,
        onConfirm: () => {
          if (!sgDialogPreviewSecurityGroup) return;
          // 不直接执行变更，先弹二次确认弹窗（PR #169 精打磨）
          setIsConfirmSwitchDialogOpen(true);
        },
      })}

      {/* ─── 确认切换安全组二次确认弹窗（警示） ──────────────────────────────────────────────────── */}
      <AlertDialog open={isConfirmSwitchDialogOpen} onOpenChange={setIsConfirmSwitchDialogOpen}>
        <AlertDialogContent className="sm:max-w-[560px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">
              确认导入规则到当前 ClawPro 安全组
            </AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <Alert variant="warning" className="w-full px-3 py-3">
              <CircleAlert />
              <AlertDescription>
                <ul className="list-disc pl-4 space-y-1">
                  <li><BodyText as="span" tone="inherit">ClawPro 将把所选规则模板的规则<BodyMedium tone="inherit" className="font-semibold">复制</BodyMedium>到当前 ClawPro 安全组，当前企业下<BodyMedium tone="inherit" className="font-semibold">所有 Agent 所在云服务器</BodyMedium>将立即使用新规则。</BodyText></li>
                  <li><BodyText as="span" tone="inherit">所选规则模板在云端对应的原安全组<BodyMedium tone="inherit" className="font-semibold">不受影响</BodyMedium>，其关联的其他云端资源也不会被影响。</BodyText></li>
                </ul>
              </AlertDescription>
            </Alert>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setIsConfirmSwitchDialogOpen(false)}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (sgDialogPreviewSecurityGroup) {
                  applyCurrentSecurityGroup(sgDialogPreviewSecurityGroup);
                  // [004] 用户完成一次"导入规则"完整链路 = 自动 ack 独立化升级告知
                  //   - 触发点：二次确认"确认导入"按钮点击时（B15 完整链路终点）
                  //   - 行为：将迁移告知提示标记为已读，下次进入不再展示
                  //   - 仅触发 ack，不触发任何业务副作用
                  handleAckMigration();
                  toast.success("安全组规则已更新，当前企业下所有 Agent 将使用新规则");
                  setIsConfirmSwitchDialogOpen(false);
                  closeSelectSecurityGroupDialog();
                }
              }}
            >
              确认导入
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ─── 添加规则弹窗 ───────────────────────────────────────────────────────────────── */}
      <Dialog
        open={showAddDialog !== null}
        onOpenChange={(open) => {
          if (!open) {
            setShowAddDialog(null);
            setAddDraft({});
          }
        }}
      >
        <DialogContent
          className="sm:max-w-[420px]"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>
              {showAddDialog === "inbound" ? "添加入站规则" : "添加出站规则"}
            </DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
              {/* 来源 / 目标 */}
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">
                  {showAddDialog === "inbound" ? "来源" : "目标"}
                </MetaMedium>
                <Input
                  placeholder={showAddDialog === "inbound" ? "例如：0.0.0.0/0 或 10.0.0.0/8" : "例如：0.0.0.0/0"}
                  value={addDraft.source ?? ""}
                  onChange={(e) => setAddDraft((prev) => ({ ...prev, source: e.target.value }))}
                />
              </div>

              {/* 协议 + 端口：两列并排，协议下拉固定宽 */}
              <div className="flex gap-3 items-end">
                <div className="space-y-2 flex-none w-[120px]">
                  <MetaMedium as="label" tone="secondary">协议</MetaMedium>
                  <Select
                    value={addDraft.protocol ?? ""}
                    onValueChange={(v) => setAddDraft((prev) => ({ ...prev, protocol: v }))}
                  >
                    <SelectTrigger className="w-[120px]">
                      <SelectValue placeholder="请选择" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ALL">ALL</SelectItem>
                      <SelectItem value="TCP">TCP</SelectItem>
                      <SelectItem value="UDP">UDP</SelectItem>
                      <SelectItem value="ICMP">ICMP</SelectItem>
                      <SelectItem value="ICMPv6">ICMPv6</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2 flex-1">
                  <MetaMedium as="label" tone="secondary">端口</MetaMedium>
                  <Input
                    placeholder="例如 80 或 ALL"
                    value={addDraft.port ?? ""}
                    onChange={(e) => setAddDraft((prev) => ({ ...prev, port: e.target.value }))}
                  />
                </div>
              </div>

              {/* 策略：Radio Group（允许 / 拒绝），默认值"允许" */}
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">策略</MetaMedium>
                <RadioGroup
                  value={addDraft.policy ?? "允许"}
                  onValueChange={(v) => setAddDraft((prev) => ({ ...prev, policy: v }))}
                  className="flex items-center gap-6"
                >
                  <label htmlFor="add-policy-allow" className="flex items-center gap-2 cursor-pointer">
                    <RadioGroupItem value="允许" id="add-policy-allow" />
                    <BodyText as="span" tone="primary">允许</BodyText>
                  </label>
                  <label htmlFor="add-policy-deny" className="flex items-center gap-2 cursor-pointer">
                    <RadioGroupItem value="拒绝" id="add-policy-deny" />
                    <BodyText as="span" tone="primary">拒绝</BodyText>
                  </label>
                </RadioGroup>
              </div>

              {/* 备注（可选） */}
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">
                  备注
                  <MetaText tone="muted" className="font-normal ml-1">（可选）</MetaText>
                </MetaMedium>
                <Input
                  placeholder="简要描述此规则用途"
                  value={addDraft.remark ?? ""}
                  onChange={(e) => setAddDraft((prev) => ({ ...prev, remark: e.target.value }))}
                />
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowAddDialog(null);
                setAddDraft({});
              }}
            >
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              disabled={
                !addDraft.source?.trim()
                || !addDraft.protocol
                || !addDraft.port?.trim()
                || !addDraft.policy
              }
              onClick={() => {
                if (!addDraft.source?.trim()) {
                  toast.error(showAddDialog === "inbound" ? "请填写来源" : "请填写目标");
                  return;
                }
                if (!addDraft.protocol) {
                  toast.error("请选择协议");
                  return;
                }
                if (!addDraft.port?.trim()) {
                  toast.error("请填写端口");
                  return;
                }
                if (!addDraft.policy) {
                  toast.error("请选择策略");
                  return;
                }
                const newRule: Rule = {
                  id: String(Date.now()),
                  source: addDraft.source!,
                  protocol: addDraft.protocol!,
                  port: addDraft.port!,
                  policy: addDraft.policy!,
                  remark: addDraft.remark ?? "",
                };

                // 添加规则入口只会在已存在默认安全组时开启，这里直接追加到当前规则即可。
                if (showAddDialog === "inbound") {
                  setInboundRules((prev) => [...prev, newRule]);
                } else {
                  setOutboundRules((prev) => [...prev, newRule]);
                }
                setShowAddDialog(null);
                setAddDraft({});
                toast.success("规则添加成功");
              }}
            >
              添加规则
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── 编辑规则弹窗 ───────────────────────────────────────────────────────────────── */}
      <Dialog
        open={editingRule !== null}
        onOpenChange={(open) => {
          if (!open) {
            setEditingRule(null);
            setEditDraft({});
          }
        }}
      >
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>
              {editingRule?.type === "inbound" ? "编辑入站规则" : "编辑出站规则"}
            </DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
              {/* 来源 / 目标 */}
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">
                  {editingRule?.type === "inbound" ? "来源" : "目标"}
                </MetaMedium>
                <Input
                  placeholder={editingRule?.type === "inbound" ? "例如：0.0.0.0/0 或 10.0.0.0/8" : "例如：0.0.0.0/0"}
                  value={editDraft.source ?? ""}
                  onChange={(e) => setEditDraft((prev) => ({ ...prev, source: e.target.value }))}
                />
              </div>

              {/* 协议 + 端口：两列并排，协议下拉固定宽 */}
              <div className="flex gap-3 items-end">
                <div className="space-y-2 flex-none w-[120px]">
                  <MetaMedium as="label" tone="secondary">协议</MetaMedium>
                  <Select
                    value={editDraft.protocol ?? ""}
                    onValueChange={(v) => setEditDraft((prev) => ({ ...prev, protocol: v }))}
                  >
                    <SelectTrigger className="w-[120px]">
                      <SelectValue placeholder="请选择" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ALL">ALL</SelectItem>
                      <SelectItem value="TCP">TCP</SelectItem>
                      <SelectItem value="UDP">UDP</SelectItem>
                      <SelectItem value="ICMP">ICMP</SelectItem>
                      <SelectItem value="ICMPv6">ICMPv6</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2 flex-1">
                  <MetaMedium as="label" tone="secondary">端口</MetaMedium>
                  <Input
                    placeholder="例如 80 或 ALL"
                    value={editDraft.port ?? ""}
                    onChange={(e) => setEditDraft((prev) => ({ ...prev, port: e.target.value }))}
                  />
                </div>
              </div>

              {/* 策略：Radio Group（允许 / 拒绝），默认值"允许" */}
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">策略</MetaMedium>
                <RadioGroup
                  value={editDraft.policy ?? "允许"}
                  onValueChange={(v) => setEditDraft((prev) => ({ ...prev, policy: v }))}
                  className="flex items-center gap-6"
                >
                  <label htmlFor="edit-policy-allow" className="flex items-center gap-2 cursor-pointer">
                    <RadioGroupItem value="允许" id="edit-policy-allow" />
                    <BodyText as="span" tone="primary">允许</BodyText>
                  </label>
                  <label htmlFor="edit-policy-deny" className="flex items-center gap-2 cursor-pointer">
                    <RadioGroupItem value="拒绝" id="edit-policy-deny" />
                    <BodyText as="span" tone="primary">拒绝</BodyText>
                  </label>
                </RadioGroup>
              </div>

              {/* 备注（可选） */}
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">
                  备注
                  <MetaText tone="muted" className="font-normal ml-1">（可选）</MetaText>
                </MetaMedium>
                <Input
                  placeholder="简要描述此规则用途"
                  value={editDraft.remark ?? ""}
                  onChange={(e) => setEditDraft((prev) => ({ ...prev, remark: e.target.value }))}
                />
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setEditingRule(null);
                setEditDraft({});
              }}
            >
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              disabled={
                !editDraft.source?.trim()
                || !editDraft.protocol
                || !editDraft.port?.trim()
                || !editDraft.policy
              }
              onClick={() => {
                if (!editDraft.source?.trim()) {
                  toast.error(editingRule?.type === "inbound" ? "请填写来源" : "请填写目标");
                  return;
                }
                if (!editDraft.protocol) {
                  toast.error("请选择协议");
                  return;
                }
                if (!editDraft.port?.trim()) {
                  toast.error("请填写端口");
                  return;
                }
                if (!editDraft.policy) {
                  toast.error("请选择策略");
                  return;
                }
                const updatedRule = {
                  ...editDraft,
                  source: editDraft.source!,
                  protocol: editDraft.protocol!,
                  port: editDraft.port!,
                  policy: editDraft.policy!,
                  remark: editDraft.remark ?? "",
                } as Rule;
                if (editingRule?.type === "inbound") {
                  setInboundRules((prev) => prev.map((r) => (r.id === updatedRule.id ? updatedRule : r)));
                } else {
                  setOutboundRules((prev) => prev.map((r) => (r.id === updatedRule.id ? updatedRule : r)));
                }
                setEditingRule(null);
                setEditDraft({});
                toast.success("规则修改成功");
              }}
            >
              保存修改
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {MigrationConfirmDialog}
      {MigrationProgressDialog}
      {/* ─── 删除规则二次确认弹窗 - 警示弹窗 ─── */}
      <AlertDialog open={showDeleteDialog !== null} onOpenChange={(open) => !open && setShowDeleteDialog(null)}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <button
            type="button"
            aria-label="关闭"
            onClick={() => setShowDeleteDialog(null)}
            className="absolute top-5 right-5 flex items-center justify-center size-5 rounded-sm text-[var(--text-muted)] transition-colors hover:text-[var(--text-title)] focus:outline-none"
          >
            <X className="size-5" />
            <span className="sr-only">关闭</span>
          </button>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">确定删除该规则？</AlertDialogTitle>
            <AlertDialogDescription asChild>
              {showDeleteDialog?.is_required ? (
                <BodyText as="p" tone="primary">
                  <BodyText as="span" tone="danger">请注意：该规则由平台策略自动生成。</BodyText>
                  删除后，相关网络流量策略将立即失效，
                  <BodyText as="span" tone="danger">可能影响现有业务的网络访问</BodyText>
                  。请确认是否继续。
                </BodyText>
              ) : (
                <BodyText as="p" tone="primary">
                  删除后，相关网络流量策略将立即失效，
                  <BodyText as="span" tone="danger">可能影响现有业务的网络访问</BodyText>
                  。请确认是否继续。
                </BodyText>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setShowDeleteDialog(null)}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (!showDeleteDialog) return;
                const snapshot = showDeleteDialog;
                if (snapshot.type === "inbound") {
                  setInboundRules((prev) => prev.filter((r) => r.id !== snapshot.id));
                } else {
                  setOutboundRules((prev) => prev.filter((r) => r.id !== snapshot.id));
                }
                toast.success("规则已删除");
                setShowDeleteDialog(null);
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
