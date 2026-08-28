/**
 * AuthSourceImportDialog - 数据源导入弹窗
 * Design: 「流动蓝图」Fluid Blueprint - Admin Side
 * 四步骤流程：1. 选择数据源方式 → 2. 添加应用凭证 → 3. 设置字段映射 → 4. 完成
 */
import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogBody,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Stepper } from "@/components/ui/stepper";
import { RadioGroup } from "@/components/ui/radio-group";
import { RadioCard } from "@/components/ui/radio-card";
import { HelperText, MetaMedium } from "@/components/ui/Typography";
import { toast } from "sonner";
import {
  Info,
  Plus,
  Trash2,
  CheckCircle,
  Eye,
  EyeOff,
} from "lucide-react";

// ─── 数据源数据定义 ──────────────────────────────────────────────────────────
interface AuthSource {
  id: string;
  name: string;
  description: string;
  /** 数据源图标 URL */
  iconUrl: string;
  /** 配置数据源时的表单字段定义 */
  fields: AuthField[];
}

/** 已配置的数据源（供外部使用） */
export interface ConfiguredAuthSource {
  id: string;
  name: string;
  iconUrl: string;
  description: string;
  enabled: boolean;
  /** 保存用户填写的凭证表单值（如 AppID、AppSecret），编辑时回填 */
  formValues?: Record<string, string>;
}

interface AuthField {
  key: string;
  label: string;
  type: "text" | "secret" | "select" | "radio" | "baseDN";
  placeholder?: string;
  required?: boolean;
  helpText?: string;
  /** select / radio 的选项 */
  options?: { value: string; label: string }[];
  /** 默认值 */
  defaultValue?: string;
  /** radio 选中某个值后，显示一个子输入框 */
  subInput?: {
    /** 当 radio 选中此值时显示子输入框 */
    showWhen: string;
    key: string;
    placeholder: string;
    type: "text" | "secret";
  };
  /** baseDN 的添加按钮文案 */
  addButtonText?: string;
}

const AUTH_SOURCES: AuthSource[] = [
  // ① 企业微信（排第一）
  {
    id: "wecom",
    name: "企业微信",
    description: "同步企业微信内的通讯录数据",
    iconUrl: "https://identity.tencent.com/public/images/wework-v2-logo.png",
    fields: [
      { key: "appId", label: "App ID", type: "text", placeholder: "请输入企业微信应用的 App ID", required: true },
      { key: "appSecret", label: "App Secret", type: "secret", placeholder: "请输入企业微信应用的 App Secret", required: true },
      { key: "corpId", label: "企业ID", type: "text", placeholder: "请输入企业微信的企业ID", required: true },
    ],
  },
  // ② 私有化企微
  {
    id: "private-wecom",
    name: "私有化企微",
    description: "同步私有化企微内的通讯录数据",
    iconUrl: "https://identity.tencent.com/public/images/wework-v2-logo.png",
    fields: [
      { key: "wecomUrl", label: "企微地址", type: "text", placeholder: "请输入私有化企微服务地址", required: true },
      { key: "appSecret", label: "Secret", type: "secret", placeholder: "请输入 Secret", required: true },
      { key: "corpId", label: "企业 ID", type: "text", placeholder: "请输入企业 ID", required: true },
    ],
  },
  // ③ 飞书
  {
    id: "feishu",
    name: "飞书",
    description: "同步飞书内的通讯录数据",
    iconUrl: "https://identity.tencent.com/public/images/lark-v2-logo.png",
    fields: [
      { key: "appId", label: "App ID", type: "text", placeholder: "请输入飞书应用的 App ID", required: true, helpText: "前往飞书开放平台 -> 应用管理，查看应用的 App ID" },
      { key: "appSecret", label: "App Secret", type: "secret", placeholder: "请输入飞书应用的 App Secret", required: true, helpText: "前往飞书开放平台 -> 应用管理，查看应用的 App Secret" },
    ],
  },
  // ④ 钉钉
  {
    id: "dingtalk",
    name: "钉钉",
    description: "同步钉钉内的通讯录数据",
    iconUrl: "https://identity.tencent.com/public/images/dd-v2-logo.png",
    fields: [
      { key: "clientId", label: "Client ID", type: "text", placeholder: "请输入钉钉应用的 Client ID", required: true },
      { key: "clientSecret", label: "Client Secret", type: "secret", placeholder: "请输入钉钉应用的 Client Secret", required: true },
    ],
  },
  // ⑤ Windows AD
  {
    id: "windows-ad",
    name: "Windows AD",
    description: "同步本地用户目录数据",
    iconUrl: "https://identity.tencent.com/public/images/ad-v2-logo.png",
    fields: [
      { key: "protocol", label: "协议类型", type: "select", required: true, defaultValue: "LDAP", options: [{ value: "LDAP", label: "LDAP" }, { value: "LDAPS", label: "LDAPS" }] },
      { key: "serverAddress", label: "AD 服务器地址", type: "text", placeholder: "请输入 AD 服务器 IP 地址", required: true },
      { key: "serverPort", label: "AD 服务器端口号", type: "text", placeholder: "389", required: true, defaultValue: "389" },
      { key: "adminAccount", label: "管理员账户", type: "text", placeholder: "请输入 AD 链接的用户名", required: true, helpText: "支持使用 UPN 格式 (admin@example.com)、域前置格式 (example\\admin)、DN 格式 (cn=admin, ou=技术部, dc=example, dc=com)" },
      { key: "adminPassword", label: "管理员密码", type: "secret", placeholder: "请输入 AD 链接的密码", required: true },
      { key: "baseDN", label: "BaseDN", type: "baseDN", placeholder: "如 \"OU=Users, DC=example, DC=com\"", required: true, addButtonText: "添加 BaseDN" },
    ],
  },
  // ⑥ OpenLDAP
  {
    id: "openldap",
    name: "OpenLDAP",
    description: "通过 LDAP 访问用户目录数据",
    iconUrl: "https://identity.tencent.com/public/images/openldap-v2-logo.png",
    fields: [
      { key: "protocol", label: "协议类型", type: "select", required: true, defaultValue: "LDAP", options: [{ value: "LDAP", label: "LDAP" }, { value: "LDAPS", label: "LDAPS" }] },
      { key: "serverAddress", label: "LDAP 服务器地址", type: "text", placeholder: "请输入 LDAP 服务器 IP 地址", required: true },
      { key: "serverPort", label: "LDAP 服务器端口号", type: "text", placeholder: "389", required: true, defaultValue: "389" },
      { key: "adminAccount", label: "管理员账户", type: "text", placeholder: "请输入 LDAP 链接的用户名", required: true, helpText: "支持使用 UPN 格式 (admin@example.com)、域前置格式 (example\\admin)、DN 格式 (cn=admin, ou=技术部, dc=example, dc=com)" },
      { key: "adminPassword", label: "管理员密码", type: "secret", placeholder: "请输入 LDAP 链接的密码", required: true },
      { key: "baseDN", label: "BaseDN", type: "baseDN", placeholder: "如 \"OU=Users, DC=example, DC=com\"", required: true, addButtonText: "添加 BaseDN" },
    ],
  },
  // ⑦ 微软 Entra ID
  {
    id: "entra-id",
    name: "微软 Entra ID",
    description: "同步微软 Entra ID 内的通讯录数据",
    iconUrl: "https://identity.tencent.com/public/images/aad-v2-logo.png",
    fields: [
      { key: "deployEnv", label: "部署环境", type: "radio", required: true, defaultValue: "global", options: [{ value: "global", label: "Global 全球版" }, { value: "21vianet", label: "21Vianet 世纪互联中国版" }] },
      { key: "clientId", label: "Application (client) ID", type: "text", placeholder: "请输入 Application (client) ID", required: true },
      { key: "tenantId", label: "Directory (tenant) ID", type: "text", placeholder: "请输入 Directory (tenant) ID", required: true },
      { key: "credentialType", label: "凭证配置", type: "radio", required: true, defaultValue: "client_secret", options: [{ value: "client_secret", label: "客户端密钥" }, { value: "self_cert", label: "自建证书" }, { value: "third_cert", label: "第三方证书" }], subInput: { showWhen: "client_secret", key: "credentialValue", placeholder: "请输入客户端密钥", type: "secret" } },
    ],
  },
];

// ─── 部门映射条目 ──────────────────────────────────────────────────────────
interface DeptMapping {
  id: string;
  sourceAttr: string;
  sourceLabel: string;
  platformAttr: string;
  platformLabel: string;
  fixed: boolean; // 固定行不可删除、禁用编辑
}

// ─── 成员映射条目 ──────────────────────────────────────────────────────────
interface MemberMapping {
  id: string;
  sourceAttr: string;
  sourceLabel: string;
  platformAttr: string;
  platformLabel: string;
  fixed: boolean;
  hasTooltip?: boolean;
  tooltipText?: string;
}

// 部门映射可选属性
const DEPT_SOURCE_ATTRS = [
  { value: "dept.name", label: "部门名称 dept.name" },
  { value: "dept.id", label: "部门ID dept.id" },
  { value: "dept.parent_id", label: "上级部门ID dept.parent_id" },
];

const DEPT_PLATFORM_ATTRS = [
  { value: "dept.name", label: "部门名称 dept.name" },
  { value: "dept.id", label: "部门ID dept.id" },
  { value: "dept.parent_id", label: "上级部门ID dept.parent_id" },
];

// 成员映射可选属性
const MEMBER_SOURCE_ATTRS = [
  { value: "user.user_id", label: "租户内用户唯一标识 user.user_id" },
  { value: "user.name", label: "姓名 user.name" },
  { value: "user.mobile", label: "手机号 user.mobile" },
  { value: "user.email", label: "邮箱 user.email" },
  { value: "user.en_name", label: "用户英文名 user.en_name" },
  { value: "user.nickname", label: "别名 user.nickname" },
  { value: "user.gender", label: "性别 user.gender" },
  { value: "user.city", label: "工作城市 user.city" },
  { value: "user.join_time", label: "入职时间 user.join_time" },
  { value: "user.work_station", label: "工位 user.work_station" },
  { value: "user.employee_no", label: "工号 user.employee_no" },
  { value: "user.employee_type", label: "用户类型 user.employee_type" },
  { value: "user.job_title", label: "职务 user.job_title" },
  { value: "user.avatar_origin", label: "用户头像信息 user.avatar_origin" },
];

const MEMBER_PLATFORM_ATTRS = [
  { value: "user.username", label: "登录用户名 user.username" },
  { value: "user.name", label: "姓名 user.name" },
  { value: "user.mobile", label: "登录手机号 user.mobile" },
  { value: "user.email", label: "联系邮箱 user.email" },
  { value: "user.joinTime", label: "入职时间 user.joinTime" },
  { value: "user.employeeNo", label: "工号 user.employeeNo" },
  { value: "user.position", label: "职位 user.position" },
];

// 默认部门映射
const getDefaultDeptMappings = (): DeptMapping[] => [
  { id: "dept-1", sourceAttr: "dept.name", sourceLabel: "部门名称 dept.name", platformAttr: "dept.name", platformLabel: "部门名称 dept.name", fixed: true },
];

// 默认成员映射
const getDefaultMemberMappings = (): MemberMapping[] => [
  { id: "member-1", sourceAttr: "user.user_id", sourceLabel: "租户内用户唯一标识 user.user_id", platformAttr: "user.username", platformLabel: "登录用户名 user.username", fixed: true },
  { id: "member-2", sourceAttr: "user.name", sourceLabel: "姓名 user.name", platformAttr: "user.name", platformLabel: "姓名 user.name", fixed: true },
  { id: "member-3", sourceAttr: "user.email", sourceLabel: "邮箱 user.email", platformAttr: "user.email", platformLabel: "联系邮箱 user.email", fixed: false },
  { id: "member-4", sourceAttr: "user.mobile", sourceLabel: "手机号 user.mobile", platformAttr: "user.mobile", platformLabel: "登录手机号 user.mobile", fixed: false, hasTooltip: true, tooltipText: "手机号将作为登录凭证之一" },
];

// ─── 主组件 ───────────────────────────────────────────────────────────────────
interface AuthSourceImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete?: (source: ConfiguredAuthSource) => void;
  /** 初始步骤（用于编辑/更换数据源时跳到指定步骤） */
  initialStep?: 1 | 2;
  /** 初始选中的数据源 ID（编辑时预选） */
  initialSourceId?: string | null;
  /** 编辑时回填的表单值（AppID、AppSecret 等） */
  initialFormValues?: Record<string, string> | null;
}

export default function AuthSourceImportDialog({
  open,
  onOpenChange,
  onComplete,
  initialStep,
  initialSourceId,
  initialFormValues,
}: AuthSourceImportDialogProps) {
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1);
  const [selectedSource, setSelectedSource] = useState<string | null>(null);
  const [formValues, setFormValues] = useState<Record<string, string>>({});
  const [deptMappings, setDeptMappings] = useState<DeptMapping[]>(getDefaultDeptMappings());
  const [memberMappings, setMemberMappings] = useState<MemberMapping[]>(getDefaultMemberMappings());
  // 控制 secret 类型字段（如 AppSecret）的明文/遮盖显示
  const [secretVisible, setSecretVisible] = useState<Record<string, boolean>>({});

  // 重置所有状态
  const resetState = () => {
    setStep(1);
    setSelectedSource(null);
    setFormValues({});
    setDeptMappings(getDefaultDeptMappings());
    setMemberMappings(getDefaultMemberMappings());
    setSecretVisible({});
  };

  // 打开弹窗时：根据 initialStep / initialSourceId 初始化
  useEffect(() => {
    if (open) {
      const targetStep = initialStep || 1;
      const targetSourceId = initialSourceId || null;
      if (targetSourceId && targetStep >= 2) {
        // 编辑模式：预选数据源并跳到指定步骤
        const source = AUTH_SOURCES.find((s) => s.id === targetSourceId);
        if (source) {
          setSelectedSource(targetSourceId);
          // 如果有传入的已保存表单值，使用它回填；否则初始化为空
          if (initialFormValues) {
            setFormValues({ ...initialFormValues });
          } else {
            setFormValues(buildInitialFormValues(source.fields));
          }
          // 编辑模式下，secret 字段默认隐藏
          setSecretVisible({});
          setDeptMappings(getDefaultDeptMappings());
          setMemberMappings(getDefaultMemberMappings());
          setStep(targetStep);
        } else {
          setStep(1);
        }
      } else {
        setStep(targetStep);
      }
    }
  }, [open, initialStep, initialSourceId, initialFormValues]);

  // 关闭弹窗时重置
  useEffect(() => {
    if (!open) {
      setTimeout(resetState, 300);
    }
  }, [open]);

  const currentSource = AUTH_SOURCES.find((s) => s.id === selectedSource);

  /** 根据 fields 定义生成初始 formValues（含 defaultValue 和 baseDN 数组） */
  const buildInitialFormValues = (fields: AuthField[]): Record<string, string> => {
    const initValues: Record<string, string> = {};
    fields.forEach((f) => {
      if (f.type === "baseDN") {
        // baseDN 用 JSON 数组存储，默认一条空字符串
        initValues[f.key] = JSON.stringify([""]);
      } else {
        initValues[f.key] = f.defaultValue || "";
      }
      // 如果 radio 字段有 subInput，也初始化子输入的 key
      if (f.type === "radio" && f.subInput) {
        initValues[f.subInput.key] = "";
      }
    });
    return initValues;
  };

  // 选择数据源（仅切换选中态，不再自动进入下一步）
  const handleSelectSource = (sourceId: string) => {
    setSelectedSource(sourceId);
  };

  // 步骤1 → 步骤2：下一步
  const handleNextFromStep1 = () => {
    if (!selectedSource) {
      toast.error("请先选择一个数据源");
      return;
    }
    const source = AUTH_SOURCES.find((s) => s.id === selectedSource);
    if (source) {
      setFormValues(buildInitialFormValues(source.fields));
    }
    setDeptMappings(getDefaultDeptMappings());
    setMemberMappings(getDefaultMemberMappings());
    setStep(2);
  };

  // 返回上一步
  const handleBack = () => {
    if (step === 2) {
      setStep(1);
      setSelectedSource(null);
    } else if (step === 3) {
      setStep(2);
    }
  };

  // 步骤2 → 步骤3
  const handleNextToMapping = () => {
    if (!currentSource) return;
    // 校验必填字段
    for (const field of currentSource.fields) {
      if (!field.required) continue;
      if (field.type === "baseDN") {
        // baseDN：至少有一条非空值
        try {
          const arr = JSON.parse(formValues[field.key] || "[]") as string[];
          if (!arr.some((v) => v.trim())) {
            toast.error(`请填写 ${field.label}`);
            return;
          }
        } catch {
          toast.error(`请填写 ${field.label}`);
          return;
        }
      } else if (!formValues[field.key]?.trim()) {
        toast.error(`请填写 ${field.label}`);
        return;
      }
      // radio 的 subInput 校验
      if (field.type === "radio" && field.subInput) {
        const currentRadioValue = formValues[field.key];
        if (currentRadioValue === field.subInput.showWhen && !formValues[field.subInput.key]?.trim()) {
          toast.error(`请填写${field.label}的相关信息`);
          return;
        }
      }
    }
    setStep(3);
  };

  // 提交配置（步骤3 → 步骤4）
  const handleSubmit = () => {
    if (!currentSource) return;
    // 模拟提交
    setStep(4);
    toast.success("数据源配置成功");
    // 通知外部（携带 formValues 以便编辑时回填）
    onComplete?.({
      id: currentSource.id,
      name: currentSource.name,
      iconUrl: currentSource.iconUrl,
      description: `通过${currentSource.name}数据源进行数据同步和登录`,
      enabled: true,
      formValues: { ...formValues },
    });
  };

  // 添加成员映射
  const addMemberMapping = () => {
    setMemberMappings([
      ...memberMappings,
      { id: String(Date.now()), sourceAttr: "", sourceLabel: "", platformAttr: "", platformLabel: "", fixed: false },
    ]);
  };

  // 删除成员映射（仅非固定行）
  const removeMemberMapping = (id: string) => {
    setMemberMappings(memberMappings.filter((m) => m.id !== id));
  };

  // 更新成员映射
  const updateMemberMapping = (id: string, field: "sourceAttr" | "platformAttr", value: string) => {
    const sourceOptions = MEMBER_SOURCE_ATTRS;
    const platformOptions = MEMBER_PLATFORM_ATTRS;
    setMemberMappings(memberMappings.map((m) => {
      if (m.id !== id) return m;
      if (field === "sourceAttr") {
        const found = sourceOptions.find((o) => o.value === value);
        return { ...m, sourceAttr: value, sourceLabel: found?.label || value };
      } else {
        const found = platformOptions.find((o) => o.value === value);
        return { ...m, platformAttr: value, platformLabel: found?.label || value };
      }
    }));
  };

  // ─── 步骤条渲染 ────────────────────────────────────────────────────────────
  const renderStepBar = () => (
    <Stepper
      className="mb-6"
      current={step}
      steps={[
        { label: "选择数据源方式" },
        { label: "添加应用凭证" },
        { label: "设置字段映射" },
        { label: "完成" },
      ]}
    />
  );

  // ─── 步骤1：选择数据源 ────────────────────────────────────────────────────
  const renderStep1 = () => (
    <RadioGroup
      value={selectedSource ?? ""}
      onValueChange={handleSelectSource}
      aria-label="选择数据源"
      className="grid grid-cols-2 gap-4"
    >
      {AUTH_SOURCES.map((source) => {
        const isSelected = selectedSource === source.id;
        return (
          <RadioCard
            key={source.id}
            variant="icon"
            id={`auth-source-${source.id}`}
            value={source.id}
            checked={isSelected}
            title={source.name}
            description={source.description}
            icon={
              <img
                src={source.iconUrl}
                alt=""
                className="w-6 h-6 object-contain"
              />
            }
          />
        );
      })}
    </RadioGroup>
  );

  // ─── 步骤2：添加应用凭证 ──────────────────────────────────────────────────
  const renderStep2 = () => {
    if (!currentSource) return null;

    /** 渲染 baseDN 类型字段（支持多条 + 添加按钮） */
    const renderBaseDN = (field: AuthField) => {
      let entries: string[] = [""];
      try {
        entries = JSON.parse(formValues[field.key] || "[\"\"]") as string[];
      } catch {
        entries = [""];
      }

      const updateEntry = (index: number, value: string) => {
        const newEntries = [...entries];
        newEntries[index] = value;
        setFormValues({ ...formValues, [field.key]: JSON.stringify(newEntries) });
      };

      const addEntry = () => {
        const newEntries = [...entries, ""];
        setFormValues({ ...formValues, [field.key]: JSON.stringify(newEntries) });
      };

      const removeEntry = (index: number) => {
        if (entries.length <= 1) return;
        const newEntries = entries.filter((_, i) => i !== index);
        setFormValues({ ...formValues, [field.key]: JSON.stringify(newEntries) });
      };

      return (
        <div className="space-y-2">
          {entries.map((entry, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <Input
                value={entry}
                onChange={(e) => updateEntry(idx, e.target.value)}
                placeholder={field.placeholder}
                className="bg-white flex-1"
              />
              {entries.length > 1 && (
                <button
                  type="button"
                  className="w-8 h-8 flex items-center justify-center rounded-[4px] text-[#A3A3A3] hover:text-red-500 hover:bg-red-50 transition-colors shrink-0"
                  onClick={() => removeEntry(idx)}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          ))}
          <button
            type="button"
            className="flex items-center gap-1.5 text-sm text-[#355EF1] hover:text-[#355EF1] font-medium transition-colors"
            onClick={addEntry}
          >
            <Plus className="w-3.5 h-3.5" />
            {field.addButtonText || "添加 BaseDN"}
          </button>
        </div>
      );
    };

    /** 渲染 radio 类型字段 */
    const renderRadio = (field: AuthField) => {
      const currentValue = formValues[field.key] || field.defaultValue || "";
      return (
        <div className="space-y-3">
          <div className="flex items-center gap-6">
            {field.options?.map((opt) => (
              <label key={opt.value} className="flex items-center gap-2 cursor-pointer">
                <div
                  className={`w-4 h-4 rounded-full border-2 flex items-center justify-center transition-colors ${
                    currentValue === opt.value
                      ? "border-blue-500"
                      : "border-gray-300"
                  }`}
                  onClick={() =>
                    setFormValues({ ...formValues, [field.key]: opt.value })
                  }
                >
                  {currentValue === opt.value && (
                    <div className="w-2 h-2 rounded-full bg-blue-500" />
                  )}
                </div>
                <span
                  className={`text-sm ${
                    currentValue === opt.value ? "text-[#0A0A0A] font-medium" : "text-[#737373]"
                  }`}
                  onClick={() =>
                    setFormValues({ ...formValues, [field.key]: opt.value })
                  }
                >
                  {opt.label}
                </span>
              </label>
            ))}
          </div>
          {/* subInput：当 radio 选中特定值时显示子输入框 */}
          {field.subInput && currentValue === field.subInput.showWhen && (() => {
            const sub = field.subInput!;
            const isSubSecret = sub.type === "secret";
            const isSubVisible = secretVisible[sub.key] ?? false;
            return (
              <div className="relative">
                <Input
                  type={isSubSecret && !isSubVisible ? "password" : "text"}
                  value={formValues[sub.key] || ""}
                  onChange={(e) =>
                    setFormValues({ ...formValues, [sub.key]: e.target.value })
                  }
                  placeholder={sub.placeholder}
                  className={`bg-white ${isSubSecret ? "pr-10" : ""}`}
                />
                {isSubSecret && (
                  <button
                    type="button"
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[#A3A3A3] hover:text-[#737373] transition-colors"
                    onClick={() =>
                      setSecretVisible({ ...secretVisible, [sub.key]: !isSubVisible })
                    }
                    tabIndex={-1}
                  >
                    {isSubVisible ? (
                      <Eye className="w-4 h-4" />
                    ) : (
                      <EyeOff className="w-4 h-4" />
                    )}
                  </button>
                )}
              </div>
            );
          })()}
        </div>
      );
    };

    return (
      <div className="space-y-4 px-1 -mx-1 pb-2">
        {currentSource.fields.map((field) => {
          const isSecret = field.type === "secret";
          const isVisible = secretVisible[field.key] ?? false;
          return (
            <div key={field.key} className="space-y-2">
              <MetaMedium as="label" tone="secondary" className="flex items-center gap-1">
                {field.label}
                {field.required && <span className="text-red-500">*</span>}
              </MetaMedium>

              {/* ── text / secret 类型 ── */}
              {(field.type === "text" || field.type === "secret") && (
                <div className="relative">
                  <Input
                    type={isSecret && !isVisible ? "password" : "text"}
                    value={formValues[field.key] || ""}
                    onChange={(e) =>
                      setFormValues({ ...formValues, [field.key]: e.target.value })
                    }
                    placeholder={field.placeholder}
                    className={`bg-white ${isSecret ? "pr-10" : ""}`}
                  />
                  {isSecret && (
                    <button
                      type="button"
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-[#A3A3A3] hover:text-[#737373] transition-colors"
                      onClick={() =>
                        setSecretVisible({ ...secretVisible, [field.key]: !isVisible })
                      }
                      tabIndex={-1}
                    >
                      {isVisible ? (
                        <Eye className="w-4 h-4" />
                      ) : (
                        <EyeOff className="w-4 h-4" />
                      )}
                    </button>
                  )}
                </div>
              )}

              {/* ── select 下拉选择 ── */}
              {field.type === "select" && (
                <Select
                  value={formValues[field.key] || field.defaultValue || ""}
                  onValueChange={(v) =>
                    setFormValues({ ...formValues, [field.key]: v })
                  }
                >
                  <SelectTrigger className="bg-white">
                    <SelectValue placeholder="请选择" />
                  </SelectTrigger>
                  <SelectContent>
                    {field.options?.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}

              {/* ── radio 单选 ── */}
              {field.type === "radio" && renderRadio(field)}

              {/* ── baseDN 可增删多条 ── */}
              {field.type === "baseDN" && renderBaseDN(field)}

              {/* helpText */}
              {field.helpText && (
                <HelperText className="leading-relaxed">
                  {field.helpText}
                </HelperText>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  // ─── 步骤3：设置字段映射 ──────────────────────────────────────────────────
  const renderStep3 = () => {
    if (!currentSource) return null;
    return (
      <div className="space-y-4 pr-1 pb-2">
        {/* 部门映射 */}
        <div className="space-y-2">
          <MetaMedium as="label" tone="secondary" className="flex items-center gap-1.5">
            部门映射
            <span className="text-red-500">*</span>
          </MetaMedium>
          <div className="bg-white border border-[#E5E5E5] rounded-[6px] p-4 space-y-2">
            {/* 表头 */}
            <div className="grid grid-cols-[1fr_40px_1fr_36px] gap-2 items-center">
              <span className="text-xs font-semibold text-[#737373]">{currentSource.name}部门</span>
              <span />
              <span className="text-xs font-semibold text-[#737373]">ClawPro 部门</span>
              <span />
            </div>
            {deptMappings.map((mapping) => (
              <div
                key={mapping.id}
                className="grid grid-cols-[1fr_40px_1fr_36px] gap-2 items-center"
              >
                <Select value={mapping.sourceAttr} disabled={mapping.fixed}>
                  <SelectTrigger className={`h-9 text-sm min-w-0 w-full ${mapping.fixed ? "bg-gray-100 text-[#737373] cursor-not-allowed" : "bg-white"}`}>
                    <SelectValue placeholder="选择属性">{mapping.sourceLabel}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {DEPT_SOURCE_ATTRS.map((attr) => (
                      <SelectItem key={attr.value} value={attr.value}>
                        {attr.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span className="text-[#A3A3A3] text-center text-sm flex items-center justify-center">→</span>
                <Select value={mapping.platformAttr} disabled={mapping.fixed}>
                  <SelectTrigger className={`h-9 text-sm min-w-0 w-full ${mapping.fixed ? "bg-gray-100 text-[#737373] cursor-not-allowed" : "bg-white"}`}>
                    <SelectValue placeholder="选择属性">{mapping.platformLabel}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {DEPT_PLATFORM_ATTRS.map((attr) => (
                      <SelectItem key={attr.value} value={attr.value}>
                        {attr.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {/* 占位列，与成员映射的删除按钮列对齐 */}
                <span className="w-8 h-8" />
              </div>
            ))}
          </div>
        </div>

        {/* 成员映射 */}
        <div className="space-y-2">
          <MetaMedium as="label" tone="secondary" className="flex items-center gap-1.5">
            成员映射
            <span className="text-red-500">*</span>
          </MetaMedium>
          <div className="bg-white border border-[#E5E5E5] rounded-[6px] p-4 space-y-2">
            {/* 表头 */}
            <div className="grid grid-cols-[1fr_40px_1fr_36px] gap-2 items-center">
              <span className="text-xs font-semibold text-[#737373]">{currentSource.name}成员</span>
              <span />
              <span className="text-xs font-semibold text-[#737373]">ClawPro 成员</span>
              <span />
            </div>
            {memberMappings.map((mapping) => (
              <div
                key={mapping.id}
                className="grid grid-cols-[1fr_40px_1fr_36px] gap-2 items-center"
              >
                <Select
                  value={mapping.sourceAttr}
                  onValueChange={(v) => updateMemberMapping(mapping.id, "sourceAttr", v)}
                >
                  <SelectTrigger className="h-9 text-sm min-w-0 w-full bg-white">
                    <SelectValue placeholder="选择属性">{mapping.sourceLabel}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {MEMBER_SOURCE_ATTRS.map((attr) => (
                      <SelectItem key={attr.value} value={attr.value}>
                        {attr.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span className="text-[#A3A3A3] text-center text-sm flex items-center justify-center">→</span>
                <Select
                  value={mapping.platformAttr}
                  onValueChange={(v) => !mapping.fixed && updateMemberMapping(mapping.id, "platformAttr", v)}
                  disabled={mapping.fixed}
                >
                  <SelectTrigger className={`h-9 text-sm min-w-0 w-full ${mapping.fixed ? "bg-gray-100 text-[#737373] cursor-not-allowed" : "bg-white"}`}>
                    <SelectValue placeholder="选择属性">
                      <span className="flex items-center gap-1">
                        {mapping.platformLabel}
                        {mapping.hasTooltip && (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="cursor-default inline-flex">
                                <Info className="w-3.5 h-3.5 text-[#A3A3A3]" />
                              </span>
                            </TooltipTrigger>
                            <TooltipContent>{mapping.tooltipText}</TooltipContent>
                          </Tooltip>
                        )}
                      </span>
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {MEMBER_PLATFORM_ATTRS.map((attr) => (
                      <SelectItem key={attr.value} value={attr.value}>
                        {attr.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {!mapping.fixed ? (
                  <button
                    className="w-8 h-8 flex items-center justify-center rounded-[4px] text-[#A3A3A3] hover:text-red-500 hover:bg-red-50 transition-colors"
                    onClick={() => removeMemberMapping(mapping.id)}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                ) : (
                  <span className="w-8 h-8" />
                )}
              </div>
            ))}
            <button
              className="flex items-center gap-1.5 text-sm text-[#355EF1] hover:text-[#355EF1] font-medium transition-colors mt-1"
              onClick={addMemberMapping}
            >
              <Plus className="w-3.5 h-3.5" />
              添加映射
            </button>
          </div>
        </div>
      </div>
    );
  };

  // ─── 步骤4：完成 ──────────────────────────────────────────────────────────
  const renderStep4 = () => (
    <div className="flex flex-col items-center justify-center py-10">
      <div
        className="w-16 h-16 rounded-full flex items-center justify-center mb-5"
      >
        <CheckCircle className="w-8 h-8 text-white" />
      </div>
      <h3 className="text-lg font-semibold text-[#0A0A0A] mb-2">数据源配置成功</h3>
      <p className="text-sm text-[#737373] text-center max-w-xs leading-relaxed">
        {currentSource?.name} 数据源已成功配置，系统将自动开始同步通讯录数据，预计需要几分钟时间。
      </p>
    </div>
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onOpenChange(false);
      }}
    >
      <DialogContent
        className="sm:max-w-[720px]"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
      >
        <DialogHeader>
          <DialogTitle className="text-[#0A0A0A]">数据源导入</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          {/* 步骤条 */}
          {renderStepBar()}

          {/* 步骤内容 */}
          {step === 1 && renderStep1()}
          {step === 2 && renderStep2()}
          {step === 3 && renderStep3()}
          {step === 4 && renderStep4()}
        </DialogBody>

        {/* 底部按钮 */}
        <DialogFooter>
          {step === 1 && (
            <>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button
                variant="dialog-confirm"
                onClick={handleNextFromStep1}
                disabled={!selectedSource}
              >
                下一步
              </Button>
            </>
          )}
          {step === 2 && (
            <>
              <Button variant="outline" onClick={handleBack}>
                上一步
              </Button>
              <Button
                variant="dialog-confirm"
                onClick={handleNextToMapping}
              >
                下一步
              </Button>
            </>
          )}
          {step === 3 && (
            <>
              <Button variant="outline" onClick={handleBack}>
                上一步
              </Button>
              <Button
                variant="dialog-confirm"
                onClick={handleSubmit}
              >
                确定
              </Button>
            </>
          )}
          {step === 4 && (
            <Button
              variant="dialog-confirm"
              onClick={() => onOpenChange(false)}
            >
              完成
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
