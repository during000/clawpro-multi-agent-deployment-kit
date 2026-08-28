/**
 * modelConfigStore.ts
 * 管控端「模型配置」共享状态存储
 * 与 customChannelStore 同款模式：localStorage 持久化 + window 事件广播。
 *
 * 用于在管控端"模型配置"页与"Agent 列表 / 抽屉"页之间共享同一份模型数据。
 */

import { useEffect, useState, useCallback } from "react";

/** 自定义模型在 ModelRow.provider 上的占位值 */
export const CUSTOM_PROVIDER_VALUE = "__custom__";

/**
 * "添加自定义模型" Dialog 的 JSON 输入模式默认占位文本。
 * 留在 store 而不是页面里，避免页面级硬编码字符串。
 */
export const CUSTOM_MODEL_DEFAULT_JSON = `{
  "provider": "provider_name",
  "base_url": "baseurl",
  "api": "API协议",
  "api_key": "your-api-key-here",
  "model": {
    "id": "model_id",
    "name": "model_name"
  }
}`;

/**
 * 默认模型 id 在 localStorage 里的 key（供用户端读取）。
 * 与本 store 的 STORAGE_KEY 不同：default-model-id 是用户端订阅 key，
 * 模型列表数据走 STORAGE_KEY = openclaw_admin_models。
 */
export const DEFAULT_MODEL_STORAGE_KEY = "adminDefaultModelId";

/**
 * 设置或清除默认模型 id 到 localStorage 并广播 storage 事件，
 * 让用户端 (Tenant) 订阅可以即时同步。
 *
 * - id 传 string => setItem + dispatch newValue=id
 * - id 传 null   => removeItem + dispatch newValue=null
 *
 * 抽出来的原因：原本在 ModelConfig.tsx 重复了 3 次 inline 的 setItem/removeItem
 * + dispatchEvent(new StorageEvent(...))，容易遗漏其中一处导致用户端不同步。
 */
export function setDefaultModelStorage(id: string | null): void {
  if (id === null) {
    localStorage.removeItem(DEFAULT_MODEL_STORAGE_KEY);
  } else {
    localStorage.setItem(DEFAULT_MODEL_STORAGE_KEY, id);
  }
  window.dispatchEvent(new StorageEvent("storage", {
    key: DEFAULT_MODEL_STORAGE_KEY,
    newValue: id,
    storageArea: localStorage,
  }));
}

export interface ModelRow {
  id: string;
  /** 厂商名（如"腾讯云 DeepSeek" / "自定义模型" / "OpenAI GPT-4o"） */
  name: string;
  /** 版本名（如"DeepSeek V3 0324"） */
  version: string;
  modelUrl: string;
  /** 模型 API Key（密文存储，编辑时脱敏展示） */
  apiKey?: string;
  visible: boolean;
  isDefault: boolean;
  dailyLimit: number;
  /**
   * API Key 由用户端自行配置：管理员添加模型时不填写密钥，交由用户端填写。
   * 该模式下不设置每日 Tokens 上限、也不支持连通性检测。
   */
  userProvidedKey?: boolean;
  /** 厂商标识，对应 AVAILABLE_MODELS.value；自定义模型为 __custom__ */
  provider: string;
  /** 该厂商可用的版本列表（非自定义模型才有意义） */
  versions: string[];
  isMultimodal?: boolean;
  /** 应用范围：全部用户 / 按组织或项目 */
  visibilityScope: "all" | "groups";
  /** 按组织/项目时选中的组织或项目 id（沿用既有字段名以兼容存量数据） */
  visibilityGroupIds: string[];
  /** 高级配置：最大输出长度 */
  maxTokens?: string;
  /** 高级配置：上下文长度（仅自定义模型） */
  contextWindow?: string;
  /** 高级配置：自定义请求头 */
  headers?: { key: string; value: string }[];
}

/**
 * 脱敏展示 API Key：保留开头 5 位 + 结尾 4 位，中间用 * 填充。
 * 长度不足 9 位时全部脱敏。
 */
export function maskApiKey(key: string | undefined): string {
  if (!key) return "";
  if (key.length <= 9) return "*".repeat(key.length);
  const prefix = key.slice(0, 5);
  const suffix = key.slice(-4);
  const maskedLen = Math.min(key.length - 9, 20);
  return `${prefix}${"*".repeat(maskedLen)}${suffix}`;
}

// ─── 默认数据 ────────────────────────────────────────────────────────────────
// 与原 ModelConfig.tsx 的 MOCK_MODELS 保持一致，确保首次进入页面体验不变。

export const DEFAULT_ADMIN_MODELS: ModelRow[] = [
  {
    id: "1", name: "腾讯云 DeepSeek", version: "DeepSeek V3 0324",
    modelUrl: "https://api.lkeap.cloud.tencent.com/v1", apiKey: "sk-ds-9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c", visible: true, isDefault: true, isMultimodal: false, dailyLimit: 500000,
    provider: "tencent-deepseek",
    versions: ["DeepSeek V3 0324", "DeepSeek R1", "DeepSeek V2.5"],
    visibilityScope: "all", visibilityGroupIds: [],
  },
  {
    id: "2", name: "腾讯云混元", version: "混元 TurboS Latest",
    modelUrl: "https://hunyuan.tencentcloudapi.com", apiKey: "sk-hy-a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", visible: true, isDefault: false, isMultimodal: false, dailyLimit: 200000,
    provider: "tencent-hunyuan",
    versions: ["混元 TurboS Latest", "混元 Pro", "混元 Standard"],
    visibilityScope: "groups",
    visibilityGroupIds: ["dept-tech", "dept-be", "dept-fe", "og-ai-core"],
  },
  {
    id: "3", name: "腾讯云 DeepSeek", version: "DeepSeek R1",
    modelUrl: "https://api.lkeap.cloud.tencent.com/v1", apiKey: "sk-ds-1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d", visible: true, isDefault: false, isMultimodal: false, dailyLimit: 100000,
    provider: "tencent-deepseek",
    versions: ["DeepSeek V3 0324", "DeepSeek R1", "DeepSeek V2.5"],
    visibilityScope: "all", visibilityGroupIds: [],
  },
  {
    id: "4", name: "OpenAI GPT-4o", version: "GPT-4o 2024-05-13",
    modelUrl: "https://api.openai.com/v1", apiKey: "sk-proj-abCdEfGhIjKlMnOpQrStUvWxYz0123456789", visible: true, isDefault: false, isMultimodal: true, dailyLimit: 300000,
    provider: CUSTOM_PROVIDER_VALUE,
    versions: [],
    visibilityScope: "groups",
    visibilityGroupIds: ["dept-ai", "dept-be", "og-ai-core"],
  },
  {
    id: "5", name: "Anthropic Claude", version: "Claude Sonnet 4",
    modelUrl: "https://api.anthropic.com/v1", apiKey: "sk-ant-api03-Xy9Zk2Lm3Np4Qr5St6Uv7Wx8Yz9Ab0Cd", visible: true, isDefault: false, isMultimodal: true, dailyLimit: 400000,
    provider: "anthropic-claude",
    versions: ["Claude Sonnet 4", "Claude Opus 4", "Claude Haiku 3.5"],
    visibilityScope: "all", visibilityGroupIds: [],
  },
];

// ─── 存储 & 广播 ─────────────────────────────────────────────────────────────

const STORAGE_KEY = "openclaw_admin_models";
const CHANGE_EVENT = "openclaw_admin_models_changed";

/** 读取所有模型；首次访问时用默认数据初始化 */
export function loadAdminModels(): ModelRow[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [...DEFAULT_ADMIN_MODELS];
    const parsed = JSON.parse(raw) as ModelRow[];
    if (!Array.isArray(parsed)) return [...DEFAULT_ADMIN_MODELS];
    return parsed;
  } catch {
    return [...DEFAULT_ADMIN_MODELS];
  }
}

/** 保存所有模型并广播变更 */
export function saveAdminModels(models: ModelRow[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(models));
  } catch {
    // 静默忽略（如 quota exceed），demo 场景下不应阻塞 UI
  }
  window.dispatchEvent(new CustomEvent(CHANGE_EVENT));
}

/** 监听模型变更（同页面 + 跨页面） */
export function onAdminModelsChange(callback: () => void): () => void {
  const handler = () => callback();
  window.addEventListener(CHANGE_EVENT, handler);
  // 跨标签页同步：localStorage 在其他标签页修改时只触发 storage 事件
  const storageHandler = (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) callback();
  };
  window.addEventListener("storage", storageHandler);
  return () => {
    window.removeEventListener(CHANGE_EVENT, handler);
    window.removeEventListener("storage", storageHandler);
  };
}

// ─── React Hook ──────────────────────────────────────────────────────────────

/**
 * Hook：作为 useState<ModelRow[]> 的替代品。
 * setter 自动 save + broadcast；同时订阅其他来源的变更并 sync 到本地 state。
 *
 * 用法：
 *   const [models, setModels] = useAdminModelsState();
 */
export function useAdminModelsState(): [ModelRow[], (next: ModelRow[] | ((prev: ModelRow[]) => ModelRow[])) => void] {
  const [models, setModelsState] = useState<ModelRow[]>(() => loadAdminModels());

  // 订阅外部变更（如另一个页面/组件改了 store）
  useEffect(() => {
    return onAdminModelsChange(() => {
      setModelsState(loadAdminModels());
    });
  }, []);

  const setModels = useCallback((next: ModelRow[] | ((prev: ModelRow[]) => ModelRow[])) => {
    setModelsState(prev => {
      const resolved = typeof next === "function" ? (next as (p: ModelRow[]) => ModelRow[])(prev) : next;
      saveAdminModels(resolved);
      return resolved;
    });
  }, []);

  return [models, setModels];
}

/**
 * Hook：只读订阅模型列表，自动响应变更。
 * 适合 Agent 抽屉这类只消费、不修改的场景。
 */
export function useAdminModels(): ModelRow[] {
  const [models, setModels] = useState<ModelRow[]>(() => loadAdminModels());
  useEffect(() => {
    return onAdminModelsChange(() => setModels(loadAdminModels()));
  }, []);
  return models;
}
