/**
 * CredentialKeySelect
 * API Key 输入组件：通过单一 Select 选择"手动输入"或已添加的凭据。
 *
 * - 选择"手动输入 API Key"：展开文本输入框，用户自行填入
 * - 选择凭据名称：自动取凭据内第一个 Header 的值写入表单（Key 明文不在 UI 展示）
 */
import { useState, useMemo } from "react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MetaMedium, HelperText } from "@/components/ui/Typography";
import { MOCK_CREDENTIALS, type Credential } from "@/lib/credentialTypes";

const MANUAL_ENTRY_VALUE = "__manual__";
/** 「由用户端自行配置」：管理员不填写密钥，交由用户端填写。 */
export const USER_SIDE_VALUE = "__user_side__";

export interface CredentialKeySelectProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "password" | "text";
  label?: string;
  readOnly?: boolean;
  className?: string;
  /** 原始 API Key 值，仅在编辑场景传入。用于反推默认的 source 选中态（凭据 or 手动填写）。 */
  initialApiKey?: string;
  /** 是否提供「由用户端自行配置」选项。 */
  allowUserSide?: boolean;
  /** 编辑场景初始化为「由用户端自行配置」选中态。 */
  initialUserSide?: boolean;
  /** 选中态在「用户端自行配置」与其它选项之间切换时回调。 */
  onUserSideChange?: (userSide: boolean) => void;
}

function useEnabledCredentials(): Credential[] {
  return useMemo(() => MOCK_CREDENTIALS.filter((c) => c.enabled), []);
}

export function CredentialKeySelect({
  value,
  onChange,
  placeholder = "请输入 API Key",
  type = "password",
  label = "API Key",
  readOnly = false,
  className,
  initialApiKey,
  allowUserSide = false,
  initialUserSide = false,
  onUserSideChange,
}: CredentialKeySelectProps) {
  const credentials = useEnabledCredentials();

  // Select 选中值：凭据 ID / MANUAL_ENTRY_VALUE / USER_SIDE_VALUE
  // 新建时 value 为空 → 无默认项
  // 编辑时 value 可能为空但 initialApiKey 有值 → 用 initialApiKey 反推选中态
  // value 非空 → 直接根据 value 反推
  const [source, setSource] = useState<string>(() => {
    if (allowUserSide && initialUserSide) return USER_SIDE_VALUE;
    const effectiveValue = value || initialApiKey || "";
    if (!effectiveValue) return "";
    const matched = credentials.find(
      (c) => c.headers.length > 0 && c.headers[0].value === effectiveValue,
    );
    return matched ? matched.id : MANUAL_ENTRY_VALUE;
  });

  const isManual = source === MANUAL_ENTRY_VALUE;
  const isUserSide = source === USER_SIDE_VALUE;

  const handleSourceChange = (val: string) => {
    if (val === USER_SIDE_VALUE) {
      setSource(USER_SIDE_VALUE);
      onChange("");
      onUserSideChange?.(true);
      return;
    }
    if (val === MANUAL_ENTRY_VALUE) {
      setSource(MANUAL_ENTRY_VALUE);
      onChange("");
    } else {
      setSource(val);
      const cred = credentials.find((c) => c.id === val);
      if (cred && cred.headers.length > 0) {
        onChange(cred.headers[0].value);
      }
    }
    onUserSideChange?.(false);
  };

  return (
    <div className={className}>
      <MetaMedium as="label" tone="secondary" className="block mb-1.5">
        {label}
      </MetaMedium>

      <Select value={source} onValueChange={handleSourceChange}>
        <SelectTrigger>
          <SelectValue placeholder="请选择凭据" />
        </SelectTrigger>
        <SelectContent>
          {credentials.length === 0 ? (
            <div className="px-2 py-3 text-center">
              <HelperText>暂无可用的凭据</HelperText>
            </div>
          ) : (
            credentials.map((cred) => (
              <SelectItem key={cred.id} value={cred.id}>
                {cred.name}
              </SelectItem>
            ))
          )}
          <SelectItem value={MANUAL_ENTRY_VALUE}>
            手动填写
          </SelectItem>
          {allowUserSide && (
            <SelectItem value={USER_SIDE_VALUE}>
              暂不填写，由用户端自行配置
            </SelectItem>
          )}
        </SelectContent>
      </Select>

      {isManual && (
        <Input
          type={type}
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          readOnly={readOnly}
          className="mt-2"
        />
      )}

      {isUserSide && (
        <HelperText className="mt-2">
          管理员无需填写密钥，将由用户端自行配置。该模式下不支持设置每日 Tokens 上限与连通性检测。
        </HelperText>
      )}
    </div>
  );
}
