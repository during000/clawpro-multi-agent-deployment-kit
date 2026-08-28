/**
 * 凭据（Credential）共享类型定义与 Mock 数据
 * 供凭据管理页和模型配置页（关联凭据选择）共用
 */

/** 凭据 Header 键值对 */
export interface CredentialHeader {
  key: string;      // Header 名称，如 "Authorization", "X-API-Key"
  value: string;    // Header 值，如 "Bearer sk-prod-xxxx"
}

/** 凭据主体 */
export interface Credential {
  id: string;                    // 唯一标识
  name: string;                  // 凭据名称，如 "生产环境API密钥"
  enabled: boolean;              // 是否启用
  headers: CredentialHeader[];   // Header 注入参数
  queryParams: CredentialHeader[]; // Query 注入参数
  linkedMcpCount: number;        // 关联的 MCP 数量
  linkedModelCount: number;      // 关联的模型数量
  linkedApiCount: number;        // 关联的 API 数量
}

/** 关联服务 */
export interface LinkedService {
  id: string;
  name: string;
  credentialId: string | null;
}

/** 凭据管理 Mock 数据 */
export const MOCK_CREDENTIALS: Credential[] = [
  {
    id: "cred-001",
    name: "生产环境 API 密钥",
    enabled: true,
    headers: [
      { key: "Authorization", value: "Bearer sk-prod-a1b2c3d4e5f6g7h8i9j0" },
      { key: "X-API-Key", value: "prod-key-xxxxxxxxxxxxxxxx" },
    ],
    queryParams: [],
    linkedMcpCount: 2,
    linkedModelCount: 1,
    linkedApiCount: 1,
  },
  {
    id: "cred-002",
    name: "测试环境密钥",
    enabled: true,
    headers: [
      { key: "Authorization", value: "Bearer sk-test-k9l8m7n6o5p4q3r2s1t0" },
    ],
    queryParams: [],
    linkedMcpCount: 0,
    linkedModelCount: 2,
    linkedApiCount: 0,
  },
  {
    id: "cred-003",
    name: "第三方模型密钥",
    enabled: false,
    headers: [
      { key: "X-API-Key", value: "3rd-party-api-key-abcdef123456" },
      { key: "X-Secret", value: "secret-token-ghijkl789012" },
    ],
    queryParams: [],
    linkedMcpCount: 0,
    linkedModelCount: 0,
    linkedApiCount: 0,
  },
];
