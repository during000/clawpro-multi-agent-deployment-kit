/**
 * 共享凭据 Store
 * 基于 localStorage 的简单发布/订阅 store，供凭据管理页面和 MCP 新增弹窗共享凭据列表数据。
 */

export interface CredentialHeader {
  key: string;
  value: string;
}

export interface CredentialItem {
  id: string;
  name: string;
  enabled: boolean;
  headers: CredentialHeader[];
  queryParams: CredentialHeader[];
  linkedMcpCount: number;
  linkedModelCount: number;
  linkedApiCount: number;
}

const CREDENTIAL_STORE_KEY = 'openclaw_credential_store';
const CREDENTIAL_STORE_VERSION_KEY = 'openclaw_credential_store_version';
const STORE_VERSION = '2';

type Listener = () => void;
const listeners = new Set<Listener>();

function notify() {
  listeners.forEach((fn) => fn());
}

function makeInitialCredentials(): CredentialItem[] {
  return [
    {
      id: 'cred-001',
      name: '生产环境API密钥',
      enabled: true,
      headers: [
        { key: 'Authorization', value: 'Bearer sk-prod-xxxx' },
        { key: 'X-API-Key', value: 'prod-key-12345' },
      ],
      queryParams: [],
      linkedMcpCount: 2,
      linkedModelCount: 1,
      linkedApiCount: 3,
    },
    {
      id: 'cred-002',
      name: '测试环境令牌',
      enabled: true,
      headers: [{ key: 'Authorization', value: 'Bearer sk-test-yyyy' }],
      queryParams: [],
      linkedMcpCount: 0,
      linkedModelCount: 0,
      linkedApiCount: 1,
    },
    {
      id: 'cred-003',
      name: '第三方集成密钥',
      enabled: false,
      headers: [
        { key: 'X-Client-ID', value: 'client-abc' },
        { key: 'X-Client-Secret', value: 'secret-xyz' },
        { key: 'X-Region', value: 'ap-guangzhou' },
      ],
      queryParams: [],
      linkedMcpCount: 3,
      linkedModelCount: 2,
      linkedApiCount: 0,
    },
  ];
}

function normalizeKeyValues(value: unknown): CredentialHeader[] {
  if (!Array.isArray(value)) return [];

  return value.flatMap((entry) => {
    if (!entry || typeof entry !== 'object') return [];
    const item = entry as Record<string, unknown>;
    if (typeof item.key !== 'string' || typeof item.value !== 'string') return [];
    return [{ key: item.key, value: item.value }];
  });
}

function normalizeCredential(value: unknown): CredentialItem | null {
  if (!value || typeof value !== 'object') return null;
  const item = value as Record<string, unknown>;
  if (typeof item.id !== 'string' || typeof item.name !== 'string') return null;

  return {
    id: item.id,
    name: item.name,
    enabled: typeof item.enabled === 'boolean' ? item.enabled : true,
    headers: normalizeKeyValues(item.headers),
    queryParams: normalizeKeyValues(item.queryParams),
    linkedMcpCount: typeof item.linkedMcpCount === 'number' ? item.linkedMcpCount : 0,
    linkedModelCount: typeof item.linkedModelCount === 'number' ? item.linkedModelCount : 0,
    linkedApiCount: typeof item.linkedApiCount === 'number' ? item.linkedApiCount : 0,
  };
}

function loadFromStore(): CredentialItem[] {
  try {
    const raw = localStorage.getItem(CREDENTIAL_STORE_KEY);
    if (raw) {
      const parsed: unknown = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        const migrated = parsed
          .map(normalizeCredential)
          .filter((item): item is CredentialItem => item !== null);
        saveToStore(migrated);
        return migrated;
      }
    }
  } catch {
    // Ignore invalid or unavailable local storage and use initial data.
  }

  const initial = makeInitialCredentials();
  saveToStore(initial);
  return initial;
}

function saveToStore(items: CredentialItem[]) {
  try {
    localStorage.setItem(CREDENTIAL_STORE_KEY, JSON.stringify(items));
    localStorage.setItem(CREDENTIAL_STORE_VERSION_KEY, STORE_VERSION);
  } catch {
    // ignore
  }
}

let _cache: CredentialItem[] | null = null;

function getCache(): CredentialItem[] {
  if (!_cache) {
    _cache = loadFromStore();
  }
  return _cache;
}

function setCache(items: CredentialItem[]) {
  _cache = items;
  saveToStore(items);
  notify();
}

export const credentialStore = {
  /** 获取所有凭据列表 */
  getAll(): CredentialItem[] {
    return getCache();
  },

  /** 获取已启用的凭据列表（用于 MCP 新增弹窗中选择） */
  getEnabled(): CredentialItem[] {
    return getCache().filter((c) => c.enabled);
  },

  /** 替换整个凭据列表 */
  setAll(items: CredentialItem[]) {
    setCache(items);
  },

  /** 添加凭据 */
  add(item: CredentialItem) {
    setCache([...getCache(), item]);
  },

  /** 更新凭据 */
  update(id: string, updater: (prev: CredentialItem) => CredentialItem) {
    setCache(getCache().map((c) => (c.id === id ? updater(c) : c)));
  },

  /** 删除凭据 */
  remove(id: string) {
    setCache(getCache().filter((c) => c.id !== id));
  },

  /** 订阅变更 */
  subscribe(fn: Listener): () => void {
    listeners.add(fn);
    return () => listeners.delete(fn);
  },
};
