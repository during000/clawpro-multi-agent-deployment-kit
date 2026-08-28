/**
 * MCPAddDialog - 新增 MCP 服务弹窗
 * 包含基本信息、连接方式切换、JSON 配置（固化外层结构，用户仅编辑服务器内容）、Markdown 编辑预览
 * 支持凭据托管：可直接输入 Token 或从凭据管理中选择已添加的凭据
 */
import { useState, useEffect, useCallback, useMemo } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from '@/components/ui/dialog';
import { Eye, Code, ChevronDown, ChevronRight, Globe, Terminal, AlignLeft, Sparkles, CircleAlert, X, Plus } from 'lucide-react';
import { SurfaceCard } from '@/components/ui/Surface';
import { StatusTag } from '@/components/ui/status-tag';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { cn } from '@/lib/utils';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import { Stepper } from '@/components/ui/stepper';
import { SearchableSelect } from '@/components/ui/select';
import { ScopeSelect } from '@/components/ScopeSelect';
import { MOCK_GROUPS, MOCK_PROJECT_GROUPS } from './mockData';

import { AnimatePresence, motion } from 'framer-motion';
import MDXRenderer from '@/components/MDXRenderer';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  PanelTitle,
  BodyMedium,
  BodyText,
  MetaText,
  MetaMedium,
  HelperText,
  CodeText,
} from '@/components/ui/Typography';
import {
  type MCPTransportType,
  type MCPConnectionCategory,
  type MCPService,
  type SkillScope,
  type ScopeLockConfig,
  MCP_CONNECTION_CATEGORY_MAP,
  MCP_REMOTE_PROTOCOL_MAP,
  MCP_TRANSPORT_MAP,
} from './types';
import { credentialStore } from '@/lib/credentialStore';

// ── 使用说明默认模板 ────────────────────────────────────
const DEFAULT_USAGE_DOC = `# 功能特点
此MCP具备的功能,比如：天气的MCP服务，支持天气的按小时查询、按天查询等功能

# 在 Openclaw 中使用
在 Openclaw 中添加mcp.json：

## 远程服务（Streamable HTTP / SSE）
\`\`\`json
{
    "mcp": {
        "servers": {
            "your-server-name": {
                "transport": "streamable-http",
                "url": "MCP服务的URL",
                "headers": {
                    "Authorization": "<your-token>"
                },
                "timeout": 60
            }
        }
    }
}
\`\`\`

## 本地命令（STDIO）
\`\`\`json
{
    "mcp": {
        "servers": {
            "your-server-name": {
                "transport": "stdio",
                "command": "python3",
                "args": ["/opt/mcp/your-server.py"],
                "env": {
                    "PYTHONUNBUFFERED": "1"
                },
                "cwd": "/path/to/your/workdir",
                "timeout": 60
            }
        }
    }
}
\`\`\`
`;

// ── 工具说明默认模板 ────────────────────────────────────
const DEFAULT_TOOL_DOC = `# 工具1：工具1的名称
功能：工具1具备的功能

---

参数：
* 参数1（必填）：参数1的详细内容
* 参数2（必填）：参数2的详细内容

| 参数 | 是否必填 | 内容 |
|------|-----|-----------|
| 参数1 | 必填 | 参数1的详细内容 |
| 参数2 | 必填 | 参数2的详细内容 |
`;

// ── 配置参考数据（可折叠查看的压缩表格） ──────────────────────────
interface ConfigRefItem {
  field: string;
  required: string;
  description: string;
}

const CONFIG_REFERENCE: Record<MCPTransportType, ConfigRefItem[]> = {
  sse: [
    { field: 'transport', required: '✅', description: '固定值 "sse"' },
    { field: 'url', required: '✅', description: '必须以 http 或 https 开头（常见以 /sse 结尾）' },
    { field: 'headers', required: '—', description: '如 MCP Server 要求 Token 认证，在此填写；否则可删除' },
    { field: 'security_zone', required: '—', description: '如 MCP 部署在 DevCloud，填写 "devnet"' },
    { field: 'timeout', required: '—', description: '超时时间，单位秒，默认 60' },
    { field: 'username', required: '—', description: '用户标识' },
  ],
  'streamable-http': [
    { field: 'transport', required: '✅', description: '固定值 "streamable-http"' },
    { field: 'url', required: '✅', description: '必须以 http 或 https 开头（常见以 /mcp 结尾）' },
    { field: 'headers', required: '—', description: '如 MCP Server 要求 Token 认证，在此填写；否则可删除' },
    { field: 'security_zone', required: '—', description: '如 MCP 部署在 DevCloud，填写 "devnet"' },
    { field: 'timeout', required: '—', description: '超时时间，单位秒，默认 60' },
    { field: 'username', required: '—', description: '用户标识' },
  ],
  stdio: [
    { field: 'transport', required: '✅', description: '固定值 "stdio"' },
    { field: 'command', required: '✅', description: '可执行文件路径（支持绝对/相对路径）' },
    { field: 'args', required: '—', description: '传给命令的参数数组，没有可留空 []' },
    { field: 'env', required: '—', description: '启动时的环境变量，没有可整段删除' },
    { field: 'cwd', required: '—', description: '子进程工作目录，默认继承 Agent 目录' },
    { field: 'timeout', required: '—', description: '超时时间，单位秒，默认 60' },
  ],
};

// ── 连接方式对应的 server 内部 JSON 模板（不含 server key） ──────
// 用户只编辑 server 对象的内部字段，外层 { "mcp": { "servers": { "{name}": { ... } } } } 由系统固化
const SERVER_VALUE_TEMPLATES: Record<MCPTransportType, string> = {
  sse: [
    `"transport": "sse",`,
    `"url": "MCP服务的URL",`,
    `"headers": {`,
    `  "Authorization": "<your-token>"`,
    `},`,
    `"timeout": 60`,
  ].join('\n'),
  'streamable-http': [
    `"transport": "streamable-http",`,
    `"url": "MCP服务的URL",`,
    `"headers": {`,
    `  "Authorization": "<your-token>"`,
    `},`,
    `"timeout": 60`,
  ].join('\n'),
  stdio: [
    `"transport": "stdio",`,
    `"command": "python3",`,
    `"args": ["/opt/mcp/your-server.py"],`,
    `"env": {`,
    `  "PYTHONUNBUFFERED": "1"`,
    `},`,
    `"cwd": "/path/to/your/workdir",`,
    `"timeout": 60`,
  ].join('\n'),
};

/** 整理缩进：移除所有行的最小公共前导空白，清理尾部空行 */
function trimCommonIndent(text: string): string {
  const lines = text.replace(/\t/g, '    ').split('\n');
  // 过滤出非空行，计算最小缩进
  const nonEmptyLines = lines.filter(l => l.trim().length > 0);
  if (nonEmptyLines.length === 0) return text;
  const minIndent = Math.min(...nonEmptyLines.map(l => l.match(/^(\s*)/)?.[1].length ?? 0));
  if (minIndent === 0) return text;
  // 移除每行的公共缩进
  const trimmed = lines.map(l => (l.trim().length > 0 ? l.slice(minIndent) : '')).join('\n');
  // 移除尾部多余空行
  return trimmed.replace(/\n+$/, '');
}

/** 将用户编辑的 server 内部内容组装成完整 JSON 字符串 */
function assembleFullJson(serverName: string, serverValueContent: string): string {
  // 给用户输入的每一行加上 8 个空格的缩进（第四层在完整 JSON 中的位置，每层 2 空格）
  const indentedLines = serverValueContent
    .split('\n')
    .map(line => (line.trim() ? `        ${line}` : ''))
    .join('\n');
  const escapedName = JSON.stringify(serverName);
  return `{\n  "mcp": {\n    "servers": {\n      ${escapedName}: {\n${indentedLines}\n      }\n    }\n  }\n}`;
}

/** 从完整 JSON 中提取指定 server 的内部内容（不含 key 和外层花括号） */
function extractServerValue(fullJson: string, serverName: string): string {
  try {
    const parsed = JSON.parse(fullJson);
    const server = parsed?.mcp?.servers?.[serverName];
    if (!server || typeof server !== 'object') return '';
    const inner = JSON.stringify(server, null, 2);
    const lines = inner.split('\n');
    if (lines.length <= 2) return '';
    return lines.slice(1, -1).map(l => l.replace(/^  /, '')).join('\n');
  } catch {
    return '';
  }
}

/** name 格式校验：仅允许英文字母、数字、连字符，1-64 字符（参考 MCP 规范 SEP-986） */
const NAME_PATTERN = /^[a-zA-Z0-9][a-zA-Z0-9\-]{0,63}$/;

interface MCPAddDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (mcp: MCPService) => void;
  /** 已存在的 MCP 名称列表，用于名称去重校验 */
  existingNames?: string[];
  /** 当在「项目资产管理」内使用时，将应用范围锁定为指定组织（只读） */
  lockedScope?: ScopeLockConfig;
}

interface FormErrors {
  name?: string;
  displayName?: string;
  connectionCategory?: string;
  transport?: string;
  configJson?: string;
  token?: string;
}

export default function MCPAddDialog({
  open,
  onOpenChange,
  onConfirm,
  existingNames = [],
  lockedScope,
}: MCPAddDialogProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  /** 应用范围 */
  const [scope, setScope] = useState<SkillScope>('public');
  const [groupIds, setGroupIds] = useState<string[]>([]);
  /** 连接类别：远程服务 / 本地命令 */
  const [connectionCategory, setConnectionCategory] = useState<MCPConnectionCategory | ''>('');
  /** 远程服务的协议子类型（仅在 connectionCategory === 'remote' 时使用） */
  const [remoteProtocol, setRemoteProtocol] = useState<'streamable-http' | 'sse'>('streamable-http');
  // 用户只编辑 server 对象的内部字段
  const [serverValueContent, setServerValueContent] = useState('');
  const [configRefExpanded, setConfigRefExpanded] = useState(false);
  const [usageDoc, setUsageDoc] = useState('');
  const [toolDoc, setToolDoc] = useState('');
  const [usageViewMode, setUsageViewMode] = useState<'edit' | 'preview'>('edit');
  const [toolViewMode, setToolViewMode] = useState<'edit' | 'preview'>('edit');
  const [errors, setErrors] = useState<FormErrors>({});
  const [tokenValidating, setTokenValidating] = useState(false);
  /** 凭据托管开关 */
  const [credentialHostingEnabled, setCredentialHostingEnabled] = useState(false);
  /** IP 白名单列表 */
  const [ipWhitelist, setIpWhitelist] = useState<string[]>(['']);
  /** 凭据输入方式：real = 填写真实凭据, placeholder = 保留占位符（用户端自填） */
  const [credentialInputMode, setCredentialInputMode] = useState<'real' | 'placeholder'>('real');
  /** 凭据下拉选中值：空 / 已添加凭据 ID / 特殊值 MANUAL_CREDENTIAL_VALUE 表示"手动填写" */
  const [credentialSelectValue, setCredentialSelectValue] = useState<string>('');
  /** 手动填写的 Authorization 凭据（仅在选中"手动填写"时使用） */
  const [credentialToken, setCredentialToken] = useState('');
  /** 凭据列表（从 store 获取，用于选择器） */
  const [credentialList, setCredentialList] = useState(() => credentialStore.getEnabled());

  // 订阅凭据 store 变更
  useEffect(() => {
    return credentialStore.subscribe(() => {
      setCredentialList(credentialStore.getEnabled());
    });
  }, []);

  /** 凭据下拉中"手动填写"选项的固定 value */
  const MANUAL_CREDENTIAL_VALUE = '__manual__';

  /** 凭据下拉选项：已启用凭据 + 末尾"手动填写" */
  const credentialOptions = useMemo(() => {
    const opts = credentialList.map(c => ({ value: c.id, label: c.name || c.id }));
    opts.push({ value: MANUAL_CREDENTIAL_VALUE, label: '手动填写' });
    return opts;
  }, [credentialList]);

  /** 当前实际的 transportType（由 connectionCategory + remoteProtocol 派生） */
  const effectiveTransportType: MCPTransportType | '' =
    connectionCategory === 'local'
      ? 'stdio'
      : connectionCategory === 'remote'
        ? remoteProtocol
        : '';

  /** 用于判断用户编辑区是否仍是模板（未改动时可自动替换） */
  const allTemplateValues = Object.values(SERVER_VALUE_TEMPLATES);

  // 重置表单
  const resetForm = useCallback(() => {
    setStep(1);
    setName('');
    setDisplayName('');
    setDescription('');
    setScope('public');
    setGroupIds([]);
    setConnectionCategory('');
    setRemoteProtocol('streamable-http');
    setServerValueContent('');
    setConfigRefExpanded(false);
    setUsageDoc(DEFAULT_USAGE_DOC);
    setToolDoc(DEFAULT_TOOL_DOC);
    setUsageViewMode('edit');
    setToolViewMode('edit');
    setErrors({});
    setCredentialHostingEnabled(false);
    setIpWhitelist(['']);
    setCredentialInputMode('real');
    setCredentialSelectValue('');
    setCredentialToken('');
  }, []);

  useEffect(() => {
    if (open) {
      setUsageDoc(DEFAULT_USAGE_DOC);
      setToolDoc(DEFAULT_TOOL_DOC);
      if (lockedScope) {
        setScope('private');
        setGroupIds([lockedScope.lockedGroupId]);
      }
    } else {
      resetForm();
    }
  }, [open, resetForm, lockedScope]);

  /** 根据凭据托管开关状态返回对应的远程服务模板（开启后去掉 Authorization 行） */
  const getRemoteTemplate = (protocol: 'streamable-http' | 'sse', hosting: boolean): string => {
    if (!hosting) return SERVER_VALUE_TEMPLATES[protocol];
    return [
      `"transportType": "${protocol}",`,
      `"url": "MCP服务的URL",`,
      `"timeout": 60`,
    ].join('\n');
  };

  // 切换连接类别
  const handleCategoryChange = (category: MCPConnectionCategory) => {
    setConnectionCategory(category);
    if (category === 'local') {
      // 本地命令 → 直接填充 stdio 模板
      if (!serverValueContent || allTemplateValues.includes(serverValueContent)) {
        setServerValueContent(SERVER_VALUE_TEMPLATES.stdio);
      }
    } else if (category === 'remote') {
      // 远程服务 → 填充当前选中的协议模板
      if (!serverValueContent || allTemplateValues.includes(serverValueContent)) {
        setServerValueContent(getRemoteTemplate(remoteProtocol, credentialHostingEnabled));
      }
    }
    setConfigRefExpanded(false);
  };

  // 切换远程协议子类型
  const handleRemoteProtocolChange = (protocol: 'streamable-http' | 'sse') => {
    setRemoteProtocol(protocol);
    if (!serverValueContent || allTemplateValues.includes(serverValueContent)) {
      setServerValueContent(getRemoteTemplate(protocol, credentialHostingEnabled));
    }
  };

  // 切换凭据托管开关
  const handleCredentialHostingChange = (enabled: boolean) => {
    setCredentialHostingEnabled(enabled);
    if (!enabled) {
      setCredentialSelectValue('');
      setCredentialToken('');
    }
    // 如果当前是远程服务且配置区是模板，同步更新模板（去掉或加上 Authorization）
    if (connectionCategory === 'remote' && (!serverValueContent || allTemplateValues.includes(serverValueContent))) {
      setServerValueContent(getRemoteTemplate(remoteProtocol, enabled));
    }
  };

  // ── 校验逻辑 ──────────────────────────────────────
  const validate = (): boolean => {
    const newErrors: FormErrors = {};

    // name 校验
    if (!name.trim()) {
      newErrors.name = '请输入服务标识';
    } else if (!NAME_PATTERN.test(name.trim())) {
      newErrors.name = '仅支持英文字母、数字、连字符，长度 1-64 个字符';
    } else if (existingNames.some(n => n === name.trim())) {
      newErrors.name = '该标识已存在，请使用其他名称';
    }

    // 连接类别校验
    if (!connectionCategory) {
      newErrors.connectionCategory = '请选择连接方式';
    }

    // JSON 校验
    const transport = effectiveTransportType;
    if (!serverValueContent.trim()) {
      newErrors.configJson = '请填写服务配置';
    } else if (name.trim() && NAME_PATTERN.test(name.trim())) {
      const fullJson = assembleFullJson(name.trim(), serverValueContent);
      try {
        const parsed = JSON.parse(fullJson);
        const server = parsed?.mcp?.servers?.[name.trim()];

        if (!server || typeof server !== 'object') {
          newErrors.configJson = '配置格式错误，请检查 JSON 语法';
        } else {
          // transport 匹配校验
          if (!newErrors.configJson && transport) {
            if (server.transport && server.transport !== transport) {
              newErrors.configJson = `transport 与连接方式不一致（期望 "${transport}"，实际 "${server.transport}"）`;
            }
          }
          // URL / command 校验
          if (!newErrors.configJson && transport) {
            if (transport === 'sse' || transport === 'streamable-http') {
              if (!server.url || (typeof server.url === 'string' && !/^https?:\/\//.test(server.url) && server.url !== 'MCP服务的URL')) {
                newErrors.configJson = 'URL 必须以 http 或 https 开头';
              }
            }
            if (transport === 'stdio') {
              if (!server.command || (typeof server.command === 'string' && !server.command.trim())) {
                newErrors.configJson = '请输入可执行命令';
              }
            }
          }
        }
      } catch {
        newErrors.configJson = 'JSON 格式错误，请检查';
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const validateStep1 = (): boolean => validate();

  /** Mock 校验 Token 有效性：模拟异步请求，以 "invalid" 开头的 token 判定为无效 */
  const mockValidateToken = async (token: string): Promise<boolean> => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve(!token.toLowerCase().startsWith('invalid'));
      }, 800);
    });
  };

  const handleNext = async () => {
    if (!validateStep1()) return;
    // 开启凭据托管 + 填写真实凭据 + 选中"手动填写"时，校验 Token 是否填写
    if (
      credentialHostingEnabled &&
      credentialInputMode === 'real' &&
      credentialSelectValue === MANUAL_CREDENTIAL_VALUE
    ) {
      if (!credentialToken.trim()) {
        setErrors(prev => ({ ...prev, token: '请输入 Authorization 凭据' }));
        return;
      }
      setTokenValidating(true);
      const isValid = await mockValidateToken(credentialToken.trim());
      setTokenValidating(false);
      if (!isValid) {
        setErrors(prev => ({ ...prev, token: '凭据无效，请检查后重新填写' }));
        return;
      }
    }
    setStep(2);
  };

  const handleBack = () => setStep(1);

  const handleSubmit = () => {
    const trimmedName = name.trim();
    const fullJson = assembleFullJson(trimmedName, serverValueContent);

    // 构建凭据映射
    let finalToken: string | undefined;
    if (credentialHostingEnabled && credentialInputMode === 'real') {
      if (credentialSelectValue === MANUAL_CREDENTIAL_VALUE && credentialToken.trim()) {
        // 手动填写：直接取用户输入的 Authorization
        finalToken = credentialToken.trim();
      } else if (credentialSelectValue && credentialSelectValue !== MANUAL_CREDENTIAL_VALUE) {
        // 选中已添加凭据：从其 headers 中取 Authorization 的 value
        const found = credentialList.find(c => c.id === credentialSelectValue);
        if (found) {
          const authHeader = found.headers.find(h => h.key.toLowerCase() === 'authorization');
          finalToken = authHeader?.value ?? found.headers[0]?.value;
        }
      }
    }
    // placeholder 模式下 finalToken 保持 undefined

    const newMCP: MCPService = {
      name: trimmedName,
      displayName: displayName.trim() || trimmedName,
      description: description.trim(),
      version: '1.0.0',
      versions: ['1.0.0'],
      transport: effectiveTransportType as MCPTransportType,
      configJson: fullJson,
      usageDoc: usageDoc.trim() || undefined,
      toolDoc: toolDoc.trim() || undefined,
      credentialHostingEnabled,
      ipWhitelist: credentialHostingEnabled ? ipWhitelist.filter(ip => ip.trim()) : undefined,
      token: finalToken,
      scope,
      groupIds: scope === 'public' ? [] : groupIds,
      createdAt: new Date(),
      updatedAt: new Date(),
    };

    onConfirm(newMCP);
    toast.success('MCP 服务创建成功');
    onOpenChange(false);
  };

  /** 显示用的 name，用于固化行展示 */
  const displayServerName = name.trim() && NAME_PATTERN.test(name.trim()) ? name.trim() : 'your-server-name';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-[720px] flex flex-col"
        style={{ maxHeight: 'min(90vh, 780px)' }}
        onPointerDownOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>新增 MCP 服务</DialogTitle>
        </DialogHeader>

        {/* ── 步骤指示器（左对齐，规范 Stepper 组件） ──────────────────────────── */}
        <Stepper
          current={step}
          steps={[
            { label: '基本信息' },
            { label: '文档说明' },
          ]}
        />

        <DialogBody className="flex-1 mt-4 px-6">
        {/* ── 第一步：基本信息 + 服务配置 ────────── */}
        <AnimatePresence mode="wait">
        {step === 1 && (
          <motion.div
            key="step-1"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
            className="space-y-4"
          >
            {/* 用户自填字段提示 — Alert 必须在内容区最上方 */}
            <Alert variant="warning">
              <CircleAlert />
              <AlertDescription>
                用户可在租户端自选配此 MCP，请注意敏感数据泄露风险。
              </AlertDescription>
            </Alert>

            <div className="space-y-4">
              <PanelTitle>基本信息</PanelTitle>

                {/* 服务标识 (name) — 唯一 key */}
                <div>
                  <Label htmlFor="mcp-name">
                    <MetaMedium as="label" tone="secondary">服务标识 <span className="text-red-500">*</span></MetaMedium>
                  </Label>
                  <HelperText as="span" className="mt-0.5 mb-1 block">唯一标识，对应 JSON 中的 server key，创建后不可修改</HelperText>
                  <Input
                    id="mcp-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., weather-mcp"
                    className="mt-1"
                  />
                  {errors.name ? (
                    <MetaText tone="danger" as="p" className="mt-1">{errors.name}</MetaText>
                  ) : (
                    <HelperText className="mt-1">仅支持英文字母、数字、连字符，长度 1-64 个字符</HelperText>
                  )}
                </div>

                {/* 名称 (displayName) */}
                <div>
                  <Label htmlFor="mcp-display-name">
                    <MetaMedium as="label" tone="secondary">名称</MetaMedium>
                  </Label>
                  <HelperText as="span" className="mt-0.5 mb-1 block">可选的显示名称，不填则默认与服务标识一致</HelperText>
                  <Input
                    id="mcp-display-name"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder="e.g., 天气 MCP 服务"
                    className="mt-1"
                  />
                </div>

                {/* 描述 */}
                <div>
                  <Label htmlFor="mcp-desc">
                    <MetaMedium as="label" tone="secondary">描述</MetaMedium>
                  </Label>
                  <Textarea
                    id="mcp-desc"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder="MCP 服务的简要说明"
                    className="mt-1 resize-none"
                    rows={2}
                  />
                </div>

                {/* 凭据托管已挪到服务配置下方 */}

                {/* 应用范围 */}
                <div>
                  <Label>
                    <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
                  </Label>
                  <div className="mt-1">
                    {lockedScope ? (
                      <div className="flex items-center h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-[var(--text-secondary)]">
                        <MetaMedium tone="secondary">{lockedScope.lockedGroupName}</MetaMedium>
                      </div>
                    ) : (
                      <ScopeSelect
                        scope={scope === 'public' ? 'all' : 'groups'}
                        selectedGroupIds={groupIds}
                        groups={MOCK_GROUPS}
                        projects={MOCK_PROJECT_GROUPS}
                        onConfirm={(s, ids) => {
                          if (s === 'all') {
                            setScope('public');
                            setGroupIds([]);
                          } else {
                            setScope('private');
                            setGroupIds(ids);
                          }
                        }}
                      />
                    )}
                  </div>
                </div>

                {/* 连接方式 — 两级 Radio */}
                <div>
                  <Label>
                    <MetaMedium as="label" tone="secondary">连接方式 <span className="text-red-500">*</span></MetaMedium>
                  </Label>
                  {/* 第一级：远程服务 / 本地命令 */}
                  <div className="flex gap-3 mt-2">
                    {(Object.keys(MCP_CONNECTION_CATEGORY_MAP) as MCPConnectionCategory[]).map((cat) => {
                      const isSelected = connectionCategory === cat;
                      const IconComp = cat === 'remote' ? Globe : Terminal;
                      return (
                        <button
                          key={cat}
                          type="button"
                          onClick={() => handleCategoryChange(cat)}
                          className={cn(
                            "flex-1 flex flex-col gap-1 rounded-[4px] border px-3 py-3 transition-colors text-left",
                            "border-gray-200 bg-white",
                            !isSelected && "hover:border-[#1447E6]/40 cursor-pointer",
                            isSelected && "border-[#1447E6] bg-[#1447E6]/5",
                          )}
                        >
                          <div className="flex items-center gap-2.5">
                            <IconComp className={cn("w-4 h-4 shrink-0", isSelected ? "text-[#1447E6]" : "text-[#A3A3A3]")} />
                            <BodyMedium className="leading-snug">{MCP_CONNECTION_CATEGORY_MAP[cat].label}</BodyMedium>
                          </div>
                          <HelperText className="leading-relaxed block">{MCP_CONNECTION_CATEGORY_MAP[cat].description}</HelperText>
                        </button>
                      );
                    })}
                  </div>
                  {errors.connectionCategory && (
                    <MetaText tone="danger" as="p" className="mt-1">{errors.connectionCategory}</MetaText>
                  )}

                  {/* 第二级：远程服务的协议子选项 */}
                  <AnimatePresence>
                  {connectionCategory === 'remote' && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: 'auto', opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={{ duration: 0.25, ease: 'easeOut' }}
                      className="overflow-hidden"
                    >
                    <div className="mt-3 ml-1">
                      <Label className="mb-1.5 block">
                        <MetaMedium as="label" tone="secondary">传输协议</MetaMedium>
                      </Label>
                      <RadioGroup
                        value={remoteProtocol}
                        onValueChange={(v) => handleRemoteProtocolChange(v as 'streamable-http' | 'sse')}
                        className="flex gap-3"
                      >
                        {(Object.keys(MCP_REMOTE_PROTOCOL_MAP) as ('streamable-http' | 'sse')[]).map((proto) => {
                          const isSelected = remoteProtocol === proto;
                          const info = MCP_REMOTE_PROTOCOL_MAP[proto];
                          return (
                            <label
                              key={proto}
                              htmlFor={`proto-${proto}`}
                              className={cn(
                                "flex items-center gap-2 px-3 py-2 rounded-[4px] border transition-colors cursor-pointer",
                                isSelected
                                  ? "border-[#1447E6] bg-[#1447E6]/5"
                                  : "border-gray-200 bg-white hover:border-[#1447E6]/40"
                              )}
                            >
                              <RadioGroupItem id={`proto-${proto}`} value={proto} />
                              <BodyMedium className={cn("", isSelected ? "" : "text-[var(--text-muted)]")}>{info.label}</BodyMedium>
                              {info.tag && (
                                <StatusTag
                                  variant={proto === 'streamable-http' ? 'green' : 'yellow'}
                                  mode="soft"
                                >
                                  {info.tag}
                                </StatusTag>
                              )}
                            </label>
                          );
                        })}
                      </RadioGroup>
                    </div>
                    </motion.div>
                  )}
                  </AnimatePresence>
                </div>
              </div>

              {/* ── 服务配置 (JSON) ────────────────────── */}
              <AnimatePresence>
              {effectiveTransportType && (
              <motion.div
                key={`config-section-${effectiveTransportType}`}
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ duration: 0.3, ease: 'easeOut' }}
                className="overflow-hidden"
              >
              <div className="space-y-2 border-t border-gray-200 pt-4">
                <PanelTitle>
                  服务配置 <span className="text-red-500">*</span>
                </PanelTitle>
                <HelperText className="mb-1">外层结构 <code className="px-1 py-0.5 bg-gray-100 rounded text-xs font-mono">mcp.servers.{displayServerName}</code> 已固定，仅需编辑服务器字段内容；可用 <code className="px-1 py-0.5 bg-gray-100 rounded text-xs font-mono">&lt;&gt;</code> 框住需用户填写的内容。</HelperText>

                {/* 可折叠的配置参考 — SurfaceCard + 压缩表格 */}
                {effectiveTransportType && (
                  <SurfaceCard className="overflow-hidden">
                    <button
                      type="button"
                      onClick={() => setConfigRefExpanded(!configRefExpanded)}
                      className="w-full flex items-center gap-2 px-3 py-2 bg-gray-50 hover:bg-gray-100 transition-colors"
                    >
                      {configRefExpanded ? (
                        <ChevronDown className="w-3.5 h-3.5" />
                      ) : (
                        <ChevronRight className="w-3.5 h-3.5" />
                      )}
                      <BodyMedium>查看「{MCP_TRANSPORT_MAP[effectiveTransportType].label}」配置参考</BodyMedium>
                    </button>
                    {configRefExpanded && (
                      <div className="border-t border-gray-200 bg-white max-h-[320px] overflow-y-auto">
                        <Table density="compact" containerClassName="border-0 rounded-none">
                          <TableHeader>
                            <TableRow>
                              <TableHead>字段</TableHead>
                              <TableHead>必填</TableHead>
                              <TableHead>说明</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {CONFIG_REFERENCE[effectiveTransportType].map((item) => (
                              <TableRow key={item.field}>
                                <TableCell>
                                  <CodeText>{item.field}</CodeText>
                                </TableCell>
                                <TableCell>{item.required}</TableCell>
                                <TableCell>{item.description}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                  </SurfaceCard>
                )}

                {/* 固化外层 + 可编辑 server 内部字段 的编辑器 */}
                <div className="border border-gray-200 rounded-[4px] overflow-hidden font-mono text-xs">
                  {/* 固定前缀行（不可编辑）— 4 层深度，2 空格缩进 */}
                  <div className="bg-gray-50 text-gray-400 px-3 py-1.5 border-b border-gray-200 select-none leading-relaxed text-xs whitespace-pre">
                    <div>{'{'}</div>
                    <div>{'  "mcp": {'}</div>
                    <div>{'    "servers": {'}</div>
                    <div>{'      '}<span className="text-gray-500">{`"${displayServerName}"`}</span>{': {'}</div>
                  </div>
                  {/* 可编辑区域（第四层内容） */}
                  <div className="relative">
                    {/* 整理缩进按钮 — 悬浮在编辑区右上角 */}
                    <TooltipProvider delayDuration={300}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button
                            type="button"
                            onClick={() => setServerValueContent(trimCommonIndent(serverValueContent))}
                            className="absolute top-1.5 right-2 z-10 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
                          >
                            <AlignLeft className="w-3 h-3" />
                            整理缩进
                          </button>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs">
                          移除多余的公共缩进空格
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                    <Textarea
                      value={serverValueContent}
                      onChange={(e) => setServerValueContent(e.target.value)}
                      placeholder={connectionCategory ? '已填入模板，可直接修改配置字段' : '请先选择连接方式'}
                      className="border-0 rounded-none font-mono text-xs min-h-[120px] focus-visible:ring-0 focus-visible:ring-offset-0 resize-y leading-relaxed"
                      rows={6}
                      style={{ paddingLeft: 'calc(0.75rem + 8ch)', fontSize: '12px' }}
                    />
                  </div>
                  {/* 固定后缀行（不可编辑） */}
                  <div className="bg-gray-50 text-gray-400 px-3 py-1.5 border-t border-gray-200 select-none leading-relaxed text-xs whitespace-pre">
                    <div>{'      }'}</div>
                    <div>{'    }'}</div>
                    <div>{'  }'}</div>
                    <div>{'}'}</div>
                  </div>
                </div>
                  {errors.configJson && (
                    <MetaText tone="danger" as="p" className="mt-1">{errors.configJson}</MetaText>
                  )}
                {/* 展示配置中检测到的需用户填写字段 */}
                {(() => {
                  const matches = serverValueContent.match(/<([^>]+)>/g);
                  // 从配置文本中提取包含占位符的 JSON key 名称
                  const extractFieldKeys = (content: string, placeholders: string[]): string[] => {
                    const keys: string[] = [];
                    placeholders.forEach(ph => {
                      // 匹配 "KeyName": "...<placeholder>..." 或 "KeyName": "<placeholder>" 模式
                      const keyMatch = content.match(new RegExp(`"([^"]+)"\\s*:\\s*"[^"]*${ph.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}[^"]*"`));
                      if (keyMatch) {
                        keys.push(keyMatch[1]);
                      }
                    });
                    return [...new Set(keys)];
                  };
                  const placeholders = matches ? [...new Set(matches)] : [];
                  const fieldKeys = placeholders.length > 0 ? extractFieldKeys(serverValueContent, placeholders) : [];
                  return (
                    <Alert variant="info" className="mt-2 mb-3">
                      <Sparkles />
                      <AlertDescription>
                        <span className="inline-flex items-center gap-1.5 flex-wrap">
                          <span>需填写字段：</span>
                          {fieldKeys.length > 0 ? (
                            fieldKeys.map((f, i) => (
                              <span key={f} className="inline-flex items-center">
                                {i > 0 && <span className="mx-0.5 text-[#94A3B8]">、</span>}
                                <span className="px-2 py-0.5 bg-[#DBEAFE] text-[#1447E6] rounded-full font-medium text-xs">{f}</span>
                              </span>
                            ))
                          ) : (
                            <span className="text-[#94A3B8]">无</span>
                          )}
                        </span>
                      </AlertDescription>
                    </Alert>
                  );
                })()}
              </div>
              </motion.div>
              )}
              </AnimatePresence>

              {/* ── 凭据托管（已挪到服务配置下方） ─────────────── */}
              <SurfaceCard className="space-y-4 p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-medium">
                      <MetaMedium as="label" tone="secondary">凭据托管</MetaMedium>
                    </Label>
                    <HelperText className="mt-0.5">开启后，平台将托管该 MCP 服务的访问凭据</HelperText>
                  </div>
                  <Switch
                    checked={credentialHostingEnabled}
                    onCheckedChange={handleCredentialHostingChange}
                  />
                </div>

                {credentialHostingEnabled && (
                  <div className="space-y-4 pt-1 border-t border-gray-100">
                    {/* 凭据配置区 */}
                    <div className="space-y-3">
                      <PanelTitle>凭据配置</PanelTitle>

                      {/* ── 顶层模式切换：填写真实凭据 / 保留占位符 ── */}
                      <div className="grid grid-cols-2 gap-3">
                        <button
                          type="button"
                          onClick={() => setCredentialInputMode('real')}
                          className={cn(
                            "flex flex-col gap-1 rounded-[4px] border px-3 py-3 text-left transition-colors",
                            credentialInputMode === 'real'
                              ? "border-[#1447E6] bg-[#1447E6]/5"
                              : "border-gray-200 bg-white hover:border-[#1447E6]/40"
                          )}
                        >
                          <span className={cn("text-sm font-medium", credentialInputMode === 'real' ? "text-[#1447E6]" : "text-gray-900")}>
                            填写真实凭据
                          </span>
                          <span className="text-xs text-gray-500">用户端直接使用，无需再填写</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setCredentialInputMode('placeholder')}
                          className={cn(
                            "flex flex-col gap-1 rounded-[4px] border px-3 py-3 text-left transition-colors",
                            credentialInputMode === 'placeholder'
                              ? "border-[#1447E6] bg-[#1447E6]/5"
                              : "border-gray-200 bg-white hover:border-[#1447E6]/40"
                          )}
                        >
                          <span className={cn("text-sm font-medium", credentialInputMode === 'placeholder' ? "text-[#1447E6]" : "text-gray-900")}>
                            保留占位符
                          </span>
                          <span className="text-xs text-gray-500">用户端自行填写凭据</span>
                        </button>
                      </div>

                      {/* ── 填写真实凭据 ── */}
                      {credentialInputMode === 'real' && (
                        <div className="space-y-3">
                          <div>
                            <Label className="text-sm block">
                              <MetaMedium as="label" tone="secondary">选择凭据</MetaMedium>
                            </Label>
                            <SearchableSelect
                              value={credentialSelectValue}
                              onChange={(val) => {
                                setCredentialSelectValue(val);
                                if (errors.token) setErrors(prev => ({ ...prev, token: undefined }));
                                // 切到已选凭据时清空手动输入
                                if (val !== MANUAL_CREDENTIAL_VALUE) {
                                  setCredentialToken('');
                                }
                              }}
                              options={credentialOptions}
                              placeholder="请选择凭据"
                              className="mt-1"
                            />
                            {errors.token ? (
                              <MetaText tone="danger" as="p" className="mt-1">{errors.token}</MetaText>
                            ) : (
                              <HelperText className="mt-1">从凭据管理中选择已保存的凭据，或选择"手动填写"自行输入</HelperText>
                            )}
                          </div>

                          {/* 选中"手动填写"时显示 Authorization 输入框 */}
                          {credentialSelectValue === MANUAL_CREDENTIAL_VALUE && (
                            <div>
                              <Label className="text-sm block">
                                <MetaMedium as="label" tone="secondary">Authorization</MetaMedium>
                              </Label>
                              <Input
                                value={credentialToken}
                                onChange={(e) => {
                                  setCredentialToken(e.target.value);
                                  if (errors.token) setErrors(prev => ({ ...prev, token: undefined }));
                                }}
                                placeholder="请输入真实凭据"
                                className={cn("mt-1", errors.token && "border-destructive focus:border-destructive")}
                              />
                              {errors.token ? (
                                <MetaText tone="danger" as="p" className="mt-1">{errors.token}</MetaText>
                              ) : (
                                <HelperText className="mt-1">请填入真实凭据，保存后用户端直接使用</HelperText>
                              )}
                            </div>
                          )}
                        </div>
                      )}

                      {/* ── 保留占位符 ── */}
                      {credentialInputMode === 'placeholder' && (
                        <div className="rounded-[4px] border border-dashed border-gray-300 bg-gray-50 p-3">
                          <div className="flex items-start gap-2">
                            <CircleAlert className="w-4 h-4 text-amber-500 mt-0.5 shrink-0" />
                            <div className="space-y-1">
                              <BodyMedium className="font-medium">用户端将自行填写凭据</BodyMedium>
                              <HelperText>
                                请在 <strong>服务配置</strong> 中使用
                                <code className="px-1 py-0.5 bg-white border border-gray-200 rounded text-xs font-mono mx-0.5">&lt;your-token&gt;</code>
                                形式的占位符标记需用户填写的凭据。保存后用户端使用时需要自行填入实际凭据。
                              </HelperText>
                              {(() => {
                                const matches = serverValueContent.match(/<([^>]+)>/g);
                                const count = matches ? matches.length : 0;
                                if (count === 0) {
                                  return (
                                    <MetaText tone="warning" as="p" className="mt-1">
                                      当前配置中未检测到占位符，建议在凭据字段（如 <code className="px-1 py-0.5 bg-white border border-gray-200 rounded text-xs font-mono">Authorization</code>）填写占位符
                                    </MetaText>
                                  );
                                }
                                return (
                                  <MetaText tone="success" as="p" className="mt-1">
                                    当前配置中已检测到 {count} 个占位符
                                  </MetaText>
                                );
                              })()}
                            </div>
                          </div>
                        </div>
                      )}
                    </div>

                    {/* IP 白名单区 */}
                    <div className="space-y-2 border-t border-gray-100 pt-3">
                      <Label className="text-sm block">
                        <MetaMedium as="label" tone="secondary">IP 白名单</MetaMedium>
                      </Label>
                      <HelperText>仅允许以下 IP 地址访问该 MCP 服务，支持单个 IP 或 CIDR 格式，不填写则所有 IP 均可访问</HelperText>
                      <div className="space-y-2">
                        {ipWhitelist.map((ip, idx) => (
                          <div key={idx} className="flex items-center gap-2">
                            <Input
                              value={ip}
                              onChange={(e) => {
                                const next = [...ipWhitelist];
                                next[idx] = e.target.value;
                                setIpWhitelist(next);
                              }}
                              placeholder="e.g., 192.168.1.100 或 10.0.0.0/8"
                              className="font-mono"
                            />
                            {ipWhitelist.length > 1 && (
                              <button
                                type="button"
                                onClick={() => setIpWhitelist(ipWhitelist.filter((_, i) => i !== idx))}
                                className="flex-shrink-0 w-7 h-7 flex items-center justify-center rounded-md text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors"
                              >
                                <X className="w-3.5 h-3.5" />
                              </button>
                            )}
                          </div>
                        ))}
                        <button
                          type="button"
                          onClick={() => setIpWhitelist([...ipWhitelist, ''])}
                          className="flex items-center gap-1.5 text-xs text-blue-600 hover:text-blue-700 transition-colors mt-1"
                        >
                          <Plus className="w-3.5 h-3.5" />
                          添加 IP
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </SurfaceCard>

          </motion.div>
        )}

        {/* ── 第二步：使用说明 + 工具说明 ────────── */}
        {step === 2 && (
          <motion.div
            key="step-2"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
            className="space-y-4"
          >
            <div className="space-y-4">
              {/* ── 使用说明 (Markdown) ────────────────── */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <PanelTitle>使用说明</PanelTitle>
                  <div className="flex items-center gap-0.5 bg-gray-200/60 rounded p-0.5">
                    <button
                      type="button"
                      onClick={() => setUsageViewMode('edit')}
                      className={`flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors ${
                        usageViewMode === 'edit'
                          ? 'bg-white text-gray-900 shadow-sm font-medium'
                          : 'text-gray-500 hover:text-gray-700'
                      }`}
                    >
                      <Code className="w-3 h-3" />
                      编辑
                    </button>
                    <button
                      type="button"
                      onClick={() => setUsageViewMode('preview')}
                      className={`flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors ${
                        usageViewMode === 'preview'
                          ? 'bg-white text-gray-900 shadow-sm font-medium'
                          : 'text-gray-500 hover:text-gray-700'
                      }`}
                    >
                      <Eye className="w-3 h-3" />
                      预览
                    </button>
                  </div>
                </div>
                <HelperText>Markdown 格式，说明如何使用该 MCP 服务</HelperText>
                {usageViewMode === 'edit' ? (
                  <Textarea
                    value={usageDoc}
                    onChange={(e) => setUsageDoc(e.target.value)}
                    placeholder="# 使用说明&#10;&#10;在此编写 Markdown 格式的使用说明..."
                    className="mt-1 font-mono text-xs max-h-[240px] overflow-y-auto"
                    rows={10}
                  />
                ) : (
                  <div className="border border-gray-200 rounded-[4px] p-4 max-h-[240px] overflow-y-auto bg-white">
                    {usageDoc.trim() ? (
                      <MDXRenderer content={usageDoc} />
                    ) : (
                      <div className="text-center py-6"><HelperText>暂无内容</HelperText></div>
                    )}
                  </div>
                )}
              </div>

              {/* ── 工具说明 (Markdown) ────────────────── */}
              <div className="space-y-2 border-t border-gray-200 pt-4">
                <div className="flex items-center justify-between">
                  <PanelTitle>工具说明</PanelTitle>
                  <div className="flex items-center gap-0.5 bg-gray-200/60 rounded p-0.5">
                    <button
                      type="button"
                      onClick={() => setToolViewMode('edit')}
                      className={`flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors ${
                        toolViewMode === 'edit'
                          ? 'bg-white text-gray-900 shadow-sm font-medium'
                          : 'text-gray-500 hover:text-gray-700'
                      }`}
                    >
                      <Code className="w-3 h-3" />
                      编辑
                    </button>
                    <button
                      type="button"
                      onClick={() => setToolViewMode('preview')}
                      className={`flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors ${
                        toolViewMode === 'preview'
                          ? 'bg-white text-gray-900 shadow-sm font-medium'
                          : 'text-gray-500 hover:text-gray-700'
                      }`}
                    >
                      <Eye className="w-3 h-3" />
                      预览
                    </button>
                  </div>
                </div>
                <HelperText>Markdown 格式，说明该 MCP 暴露的工具及参数</HelperText>
                {toolViewMode === 'edit' ? (
                  <Textarea
                    value={toolDoc}
                    onChange={(e) => setToolDoc(e.target.value)}
                    placeholder="# 工具列表&#10;&#10;在此编写 Markdown 格式的工具说明..."
                    className="mt-1 font-mono text-xs max-h-[240px] overflow-y-auto"
                    rows={10}
                  />
                ) : (
                  <div className="border border-gray-200 rounded-[4px] p-4 max-h-[240px] overflow-y-auto bg-white">
                    {toolDoc.trim() ? (
                      <MDXRenderer content={toolDoc} />
                    ) : (
                      <div className="text-center py-6"><HelperText>暂无内容</HelperText></div>
                    )}
                  </div>
                )}
              </div>
            </div>

          </motion.div>
        )}
        </AnimatePresence>
      </DialogBody>

      <DialogFooter className="items-center shrink-0 mt-0">
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          取消
        </Button>
        {step === 2 && (
          <Button variant="outline" onClick={handleBack}>
            上一步
          </Button>
        )}
        {step === 1 ? (
          <Button variant="dialog-confirm" onClick={handleNext} disabled={!name.trim() || !connectionCategory || !serverValueContent.trim()}>
            下一步
          </Button>
        ) : (
          <Button variant="dialog-confirm" onClick={handleSubmit}>
            创建 MCP
          </Button>
        )}
      </DialogFooter>
    </DialogContent>
    </Dialog>
  );
}
