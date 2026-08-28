import { axiosInstance } from '@/services/request';

export type SecurityApiParams = any;

/**
 * 接口调用方式（读/写）
 * - query: 只读查询，走 /admin/cloud/query/{service}
 * - mutate: 变更操作，走 /admin/cloud/mutate/{service}
 */
export type SecurityApiMethod = 'query' | 'mutate';

export type RawSecurityApiResponse<T> =
    | T
    | {
        Response?: T;
    }
    | {
        data?: {
            Response?: T;
        };
    };

// type SecurityMockModule = {
//     default: unknown;
// };

// const securityMockModules = import.meta.glob('../mock/*.json', {
//     eager: true,
// }) as Record<string, SecurityMockModule>;

// const securityMockMap = Object.entries(securityMockModules).reduce<any>(
//     (acc, [path, mod]) => {
//         const fileName = path.split('/').pop()?.replace(/\.json$/, '');
//         if (fileName) {
//             acc[fileName] = mod.default;
//         }
//         return acc;
//     },
//     {}
// );

// function cloneSecurityMock<T>(value: T): T {
//     if (typeof value === 'undefined') {
//         return {} as T;
//     }

//     return JSON.parse(JSON.stringify(value)) as T;
// }

// export function isSecurityMockEnabled(): boolean {
//     // return import.meta.env.DEV && import.meta.env.USE_MOCK !== 'false';
//     return false;
// }

// export function getSecurityApiMock<T = any>(apiName: string): T {
//     return cloneSecurityMock(securityMockMap[apiName] as T);
// }

export function normalizeSecurityApiResponse<T>(
    response: RawSecurityApiResponse<T>
): T {
    return (
        (response as { data?: { Response?: T } })?.data?.Response
        || (response as { Response?: T })?.Response
        || (response as T)
    );
}

export function buildSecurityApiPath(service: string, method: SecurityApiMethod): string {
    return `/admin/cloud/${method}/${service}`;
}

/**
 * 通用腾讯云 API 透传调用
 *
 * 根据接口文档：
 * - POST /admin/cloud/query/{service}  只读查询
 * - POST /admin/cloud/mutate/{service} 变更操作
 * - Header X-TC-Action 指定要调用的 API 名称
 * - Body 为腾讯云 API 3.0 标准请求参数 JSON
 */
export async function callSecurityApi<T = any>(
    apiName: string,
    params: SecurityApiParams = {},
    serviceType: string,
    method: SecurityApiMethod = 'query',
): Promise<T> {
    // 开发环境无后端，静默返回空数据避免 404
    if (import.meta.env.DEV) {
        console.debug(`[Security Mock] ${method} ${apiName}`, params);
        return {} as T;
    }

    const url = buildSecurityApiPath(serviceType, method);
    const response = await axiosInstance.post<RawSecurityApiResponse<T>>(url, params, {
        headers: {
            'Content-Type': 'application/json',
            'X-TC-Action': apiName,
        },
        silentError: true,
    } as any);

    return normalizeSecurityApiResponse(response.data);
}

/**
 * 创建只读查询类 API（Describe / Get / Search 等）
 */
export function createSecurityApi<T = any>(apiName: string, serviceType: string) {
    return (params?: SecurityApiParams) =>
        callSecurityApi<T>(apiName, params ?? {}, serviceType, 'query');
}

/**
 * 创建变更操作类 API（Create / Modify / Delete / Set / Scan 等）
 */
export function createSecurityMutateApi<T = any>(apiName: string, serviceType: string) {
    return (params?: SecurityApiParams) =>
        callSecurityApi<T>(apiName, params ?? {}, serviceType, 'mutate');
}
