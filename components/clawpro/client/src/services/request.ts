import axios from 'axios';
import type { AxiosRequestConfig, AxiosError } from 'axios';
// import type { HatcheryErrorResponse } from '@/types/api';
import { toast } from 'sonner';
// import { useAppStore } from '@/store';

export interface HatcheryErrorResponse {
  error: string;
  detail?: string;
  request_id?: string;
  instance_id?: string;
}

/**
 * 可选请求配置（供 service 层暴露给页面组件）
 * 目前只需 signal 用于取消请求，后续可按需扩展
 */
export interface RequestOptions {
  /** AbortController.signal —— 用于组件卸载时取消进行中的请求 */
  signal?: AbortSignal;
}

// ─── 页面级 AbortSignal 管理 ────────────────────────────────────────────────────
// 设计说明（opt-in 模式）：
// - 父组件通过 pushPageSignal / popPageSignal 管控 signal 生命周期
// - 接口定义处标记 needPageSignal: true，拦截器仅对标记的请求注入栈顶 signal
// - 未标记的请求（如公共接口 getSiteInfo）不受页面切换影响
// - 使用栈结构支持嵌套页面场景（实际通常只有一层）

const pageSignalStack: AbortSignal[] = [];

/**
 * 推入一个 signal（页面 mount 时调用）
 */
export function pushPageSignal(signal: AbortSignal): void {
  pageSignalStack.push(signal);
}

/**
 * 弹出栈顶 signal（页面 unmount 时调用）
 */
export function popPageSignal(): void {
  pageSignalStack.pop();
}

/**
 * 获取当前页面级 signal（栈顶），不存在时返回 undefined
 */
export function getCurrentPageSignal(): AbortSignal | undefined {
  return pageSignalStack.length > 0 ? pageSignalStack[pageSignalStack.length - 1] : undefined;
}

/**
 * 扩展 AxiosRequestConfig，支持 silentError / needPageSignal 选项
 * - silentError: true 时不自动弹出 toast.error
 * - needPageSignal: true 时拦截器自动注入栈顶页面级 AbortSignal
 */
declare module 'axios' {
  interface AxiosRequestConfig {
    /** 设为 true 则该请求报错时不自动弹出 toast */
    silentError?: boolean;
    /** 设为 true 表示该请求需要页面级 abort 管控（拦截器会自动注入栈顶 signal） */
    needPageSignal?: boolean;
  }
}

/**
 * Hatchery API 请求错误
 */
export class ApiError extends Error {
  status: number;
  /** 保留后端原始响应体，供调用方提取额外字段（如 captchaId） */
  responseData: unknown;
  /** 该请求是否使用了 silentError（即未弹出全局 toast） */
  silentError: boolean;
  constructor(message: string, status: number, responseData?: unknown, silentError?: boolean) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.responseData = responseData;
    this.silentError = silentError ?? false;
  }
}

/**
 * 请求被 AbortController 取消时的静默错误
 *
 * 设计说明：
 * - 底层封装函数（fetchJSON / postForm 等）统一捕获 axios cancel 错误并转为 CancelledError
 * - CancelledError 仍然是一个 rejection，调用方的 catch / finally 正常执行
 * - 但调用方无需手动判断 axios.isCancel —— 需要区分时只需 `if (err instanceof CancelledError)`
 * - 拦截器已跳过 cancel 的 toast 弹出，所以调用方 catch 里的通用错误处理自然兼容
 * - 相比 `new Promise(() => {})` 方案，不存在内存泄漏风险（Promise 链正常结束）
 */
export class CancelledError extends Error {
  constructor() {
    super('Request cancelled');
    this.name = 'CancelledError';
  }
}

/**
 * 包装 catch 回调，自动跳过 CancelledError（页面卸载导致的请求取消）
 *
 * @example
 * ```ts
 * // 之前
 * .catch((err) => {
 *   if (err instanceof CancelledError) return;
 *   setErrorMsg(err.message);
 * })
 *
 * // 之后
 * .catch(ignoreCancelled((err) => {
 *   setErrorMsg(err.message);
 * }))
 * ```
 */
export function ignoreCancelled(handler: (err: Error) => void) {
  return (err: unknown) => {
    if (err instanceof CancelledError) return;
    handler(err instanceof Error ? err : new Error(String(err)));
  };
}

/**
 * 安全请求封装：统一处理 CancelledError，调用方无需感知取消逻辑
 *
 * - CancelledError（页面卸载导致的请求取消）→ 三个回调全部跳过
 * - 正常成功 → onSuccess → onFinally
 * - 业务错误 → onError → onFinally
 *
 * @param promise 一个返回 Promise 的函数（惰性求值，避免在传参时就触发请求）
 * @param callbacks 回调配置
 *
 * @example
 * ```ts
 * // 简单用法
 * openclawApi.safeUpgrade(clawId, {
 *   onSuccess: () => {
 *     toast.success("升级已开始");
 *     navigate("/my-openclaw");
 *   },
 *   onError: (err) => toast.error(err.message),
 *   onFinally: () => setIsUpdating(false),
 * });
 *
 * // 链式调用（第一步成功后发起第二步）
 * openclawApi.safeCheckGatewayAccess(id, {
 *   onSuccess: (checkRes) => {
 *     if (!checkRes.accessible) { handleFail(); return; }
 *     // 第二步
 *     openclawApi.safeSetGatewayUI(id, {
 *       onSuccess: (res) => setWebUIUrl(res.gatewayUI),
 *       onError: (err) => setErrorMsg(err.message),
 *     });
 *   },
 *   onError: (err) => setErrorMsg(err.message),
 * });
 * ```
 */
export function safeRequest<T>(
  promise: () => Promise<T>,
  callbacks: {
    onSuccess?: (data: T) => void;
    onError?: (err: Error) => void;
    onFinally?: () => void;
  } = {}
): void {
  const { onSuccess, onError, onFinally } = callbacks;
  let cancelled = false;

  promise()
    .then((data) => {
      onSuccess?.(data);
    })
    .catch((err: unknown) => {
      if (err instanceof CancelledError) {
        cancelled = true;
        return;
      }
      onError?.(err instanceof Error ? err : new Error(String(err)));
    })
    .finally(() => {
      // 页面已卸载（cancelled）→ 跳过 onFinally，避免无效 setState
      if (!cancelled) onFinally?.();
    });
}

/**
 * axios 实例，预配置 Hatchery 协议所需的默认选项：
 * - withCredentials: 携带 Cookie（hatchery-session）
 * - Accept: application/json（Content Negotiation，确保拿到 JSON 响应）
 */
const instance = axios.create({
  baseURL: '/api',
  withCredentials: true,
  headers: {
    Accept: 'application/json',
  },
});

/**
 * 请求拦截器：按需注入页面级 AbortSignal（opt-in 模式）
 * - 只有接口定义处标记了 needPageSignal: true 的请求才会自动注入栈顶 signal
 * - 如果请求已有 signal（手动传入），优先使用请求自带的
 * - 未标记的请求（如公共接口）不受影响
 */
instance.interceptors.request.use((config) => {
  if (config.needPageSignal && !config.signal) {
    const pageSignal = getCurrentPageSignal();
    if (pageSignal) {
      config.signal = pageSignal;
    }
  }
  return config;
});

/**
 * 响应拦截器：统一处理 Hatchery 格式的错误响应
 * 错误格式：{error: "错误信息"}
 * 默认自动弹出 toast.error(error.message)，除非请求配置了 silentError: true
 */
instance.interceptors.response.use(
  (response) => response,
  (error: AxiosError<HatcheryErrorResponse>) => {
    // 请求被 AbortController 取消时，直接透传原始错误（不进后续 toast 逻辑）
    // 底层封装函数（fetchJSON / postForm 等）会统一将其转为 CancelledError
    // 调用方无需手动判断 axios.isCancel
    if (axios.isCancel(error)) {
      return Promise.reject(error);
    }

    // 重新断言类型：axios.isCancel 的类型守卫会过度收窄 AxiosError，需恢复原始类型
    const err = error as AxiosError<HatcheryErrorResponse>;
    const status = err.response?.status ?? 0;
    let errorMsg = `Request failed: ${status}`;

    // 提取错误详情用于复制
    let errorForCopy = '--';
    let detailForCopy = '--';
    let requestIdForCopy = '--';
    let instanceIdForCopy = "--";

    if (err.response?.data) {
      // data 实际可能是 object（HatcheryErrorResponse）或 string（nginx HTML 错误页等）
      const data = err.response.data as HatcheryErrorResponse | string;
      if (typeof data === 'object' && data.error) {
        errorMsg = data.error;
        errorForCopy = data.error;
        if (data.detail) {
          detailForCopy = data.detail;
        }
        if (data.request_id) {
          requestIdForCopy = data.request_id;
          errorMsg += ` (request_id: ${data.request_id})`;
        }
        if (data.instance_id) {
          instanceIdForCopy = data.instance_id;
        }
      } else if (typeof data === 'string') {
        // 如果返回的是 HTML（如 nginx 错误页），不直接展示原始 HTML
        if (data.trimStart().startsWith('<')) {
          errorMsg = `服务器错误 (${status})，请联系管理员`;
        } else {
          errorMsg = data;
        }
        errorForCopy = data;
      }
    }

    console.error(`API Error [${status}]: ${errorMsg}`);

    // 构造复制内容
    const copyContent = `error: ${errorForCopy}\nmessage: ${detailForCopy}\nrequestId: ${requestIdForCopy}\n实例 ID: ${instanceIdForCopy}`;

    // 兼容性复制函数
    const copyToClipboard = async (text: string): Promise<boolean> => {
      // 优先使用 Clipboard API
      if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
        try {
          await navigator.clipboard.writeText(text);
          return true;
        } catch {
          // 如果 Clipboard API 失败，降级到 execCommand
        }
      }
      // 降级方案：使用 execCommand
      try {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-9999px';
        textArea.style.top = '-9999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        const success = document.execCommand('copy');
        document.body.removeChild(textArea);
        return success;
      } catch {
        return false;
      }
    };

    // silentError: true 时跳过 toast，仅抛出 ApiError 供调用方自行处理
    const silentError = err.config?.silentError === true;

    // 只有当有 request_id、detail 或 instance_id 时才显示"复制详情"按钮
    const hasDetailInfo = requestIdForCopy !== "--" || detailForCopy !== "--" || instanceIdForCopy !== "--";

    if (!silentError) toast.error(errorMsg, {
      closeButton: true,
      ...(hasDetailInfo && {
        action: {
          label: '复制详情',
          onClick: () => {
            copyToClipboard(copyContent).then((success) => {
              if (success) {
                toast.success('已复制错误详情');
              } else {
                toast.error('复制失败');
              }
            });
          },
        },
      }),
    });

    throw new ApiError(errorMsg, status, err.response?.data, err.config?.silentError);
  }
);

/**
 * 通用 JSON 请求（GET / DELETE 等）
 */
export async function fetchJSON<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await instance.request<T>({ url, ...config });
  return res.data;
}

/**
 * POST application/x-www-form-urlencoded
 */
export async function postForm<T>(
  url: string,
  params: Record<string, string | number | undefined>,
  config?: AxiosRequestConfig
): Promise<T> {
  const body = new URLSearchParams();
  for (const [key, val] of Object.entries(params)) {
    if (val !== undefined && val !== null) {
      body.append(key, String(val));
    }
  }
  const res = await instance.post<T>(url, body.toString(), {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    ...config,
  });
  return res.data;
}

/**
 * POST application/x-www-form-urlencoded（支持重复 key，如 key[] / value[]）
 */
export async function postFormRepeated<T>(
  url: string,
  params: Array<[string, string]>,
  config?: AxiosRequestConfig
): Promise<T> {
  const body = new URLSearchParams();
  for (const [key, val] of params) {
    body.append(key, val);
  }
  const res = await instance.post<T>(url, body.toString(), {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    ...config,
  });
  return res.data;
}

/**
 * POST multipart/form-data（用于文件上传）
 * 注意：axios 会自动设置 Content-Type 和 boundary
 */
export async function postFormData<T>(
  url: string,
  formData: FormData
): Promise<T> {
  const res = await instance.post<T>(url, formData);
  return res.data;
}

/**
 * POST application/json
 */
export async function postJSON<T>(
  url: string,
  data: Record<string, unknown>,
  config?: AxiosRequestConfig
): Promise<T> {
  const res = await instance.post<T>(url, data, {
    headers: { 'Content-Type': 'application/json' },
    ...config,
  });
  return res.data;
}

/**
 * PUT application/json
 */
export async function putJSON<T>(
  url: string,
  data: Record<string, unknown>
): Promise<T> {
  const res = await instance.put<T>(url, data, {
    headers: { 'Content-Type': 'application/json' },
  });
  return res.data;
}

/**
 * DELETE application/json
 */
export async function deleteJSON<T>(
  url: string,
  data: Record<string, unknown>
): Promise<T> {
  const res = await instance.delete<T>(url, {
    data,
    headers: { 'Content-Type': 'application/json' },
  });
  return res.data;
}

/**
 * 构造带 query 参数的 URL
 */
export function buildUrl(
  base: string,
  params?: Record<string, string | number | undefined>
): string {
  if (!params) return base;
  const qs = new URLSearchParams();
  for (const [key, val] of Object.entries(params)) {
    if (val !== undefined && val !== null) {
      qs.append(key, String(val));
    }
  }
  const str = qs.toString();
  return str ? `${base}?${str}` : base;
}

/**
 * 创建 SSE (Server-Sent Events) 连接
 * SSE 使用原生 EventSource（axios 不支持 SSE）
 * 返回 EventSource 实例，调用方自行监听事件和关闭
 */
export function createSSE(url: string): EventSource {
  return new EventSource(`/api${url}`, { withCredentials: true });
}

// ─── 回调模式封装（safe* 系列） ─────────────────────────────────────────────────
// 设计说明：
// - 保留原有 fetchJSON / postForm / postJSON 等 Promise 模式函数不变
// - 新增 safeFetchJSON / safePostForm / safePostJSON 等回调模式封装
// - 内部自动处理 CancelledError：页面卸载导致的请求取消 → 三个回调全部跳过
// - service 层可直接透传 callbacks，页面调用时无需感知取消逻辑
//
// 使用场景对比：
// - Promise 模式：适合需要 await 顺序编排的场景（如 init 流程）
// - 回调模式：适合「发射后忘记」的操作型请求（如升级、删除、配置保存等）

/**
 * 回调模式的回调配置
 *
 * @example
 * ```ts
 * safeFetchJSON<UserInfo>("/user/info", { needPageSignal: true }, {
 *   onSuccess: (data) => setUser(data),
 *   onError: (err) => toast.error(err.message),
 *   onFinally: () => setLoading(false),
 * });
 * ```
 */
export interface SafeCallbacks<T> {
  /** 请求成功时调用 */
  onSuccess?: (data: T) => void;
  /** 请求失败时调用（CancelledError 不会触发） */
  onError?: (err: Error) => void;
  /** 请求结束后调用（CancelledError 不会触发） */
  onFinally?: () => void;
}

/**
 * 内部工具：将 Promise 接入回调模式，自动跳过 CancelledError
 */
function wrapWithCallbacks<T>(
  promise: Promise<T>,
  callbacks: SafeCallbacks<T>
): void {
  let cancelled = false;
  promise
    .then((data) => {
      callbacks.onSuccess?.(data);
    })
    .catch((err: unknown) => {
      if (err instanceof CancelledError || axios.isCancel(err)) {
        cancelled = true;
        return;
      }
      callbacks.onError?.(err instanceof Error ? err : new Error(String(err)));
    })
    .finally(() => {
      if (!cancelled) callbacks.onFinally?.();
    });
}

/**
 * 回调模式的 JSON 请求（GET / DELETE 等）
 * 签名与 fetchJSON 一致，额外接受 SafeCallbacks
 */
export function safeFetchJSON<T>(
  url: string,
  config?: AxiosRequestConfig,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(fetchJSON<T>(url, config), callbacks);
}

/**
 * 回调模式的 POST application/x-www-form-urlencoded
 */
export function safePostForm<T>(
  url: string,
  params: Record<string, string | number | undefined>,
  config?: AxiosRequestConfig,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(postForm<T>(url, params, config), callbacks);
}

/**
 * 回调模式的 POST application/x-www-form-urlencoded（支持重复 key）
 */
export function safePostFormRepeated<T>(
  url: string,
  params: Array<[string, string]>,
  config?: AxiosRequestConfig,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(postFormRepeated<T>(url, params, config), callbacks);
}

/**
 * 回调模式的 POST multipart/form-data（用于文件上传）
 */
export function safePostFormData<T>(
  url: string,
  formData: FormData,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(postFormData<T>(url, formData), callbacks);
}

/**
 * 回调模式的 POST application/json
 */
export function safePostJSON<T>(
  url: string,
  data: Record<string, unknown>,
  config?: AxiosRequestConfig,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(postJSON<T>(url, data, config), callbacks);
}

/**
 * 回调模式的 PUT application/json
 */
export function safePutJSON<T>(
  url: string,
  data: Record<string, unknown>,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(putJSON<T>(url, data), callbacks);
}

/**
 * 回调模式的 DELETE application/json
 */
export function safeDeleteJSON<T>(
  url: string,
  data: Record<string, unknown>,
  callbacks: SafeCallbacks<T> = {}
): void {
  wrapWithCallbacks(deleteJSON<T>(url, data), callbacks);
}

/**
 * 导出 axios 实例，供需要自定义请求的场景直接使用
 */
export { instance as axiosInstance };
