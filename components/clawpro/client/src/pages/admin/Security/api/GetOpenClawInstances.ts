import { fetchJSON, buildUrl } from '@/services/request';

/**
   * 获取所有用户的实例列表（含所属用户名），支持分页
   * GET /admin/instances?page=1&page_size=20
   */
export const GetOpenClawInstances = (params?: any) =>
    fetchJSON<any>(buildUrl('/admin/instances', params as Record<string, string | number | undefined>), { silentError: true });