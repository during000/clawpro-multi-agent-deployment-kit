import type {
  AdminUsersResponse,
  AdminUsersParams,
  CreateUserParams,
  UpdateUserParams,
  ResetPasswordParams,
  SiteConfigResponse,
  SiteConfigInfo,
  UpdateCvmConfigParams,
  UpdateTemplatePatch,
  ModelListResponse,
  CreateModelParams,
  UpdateModelParams,
  UpdateModelVisibilityParams,
  ConnectivityTestParams,
  ConnectivityTestResponse,
  UserGroupsResponse,
  CreateUserGroupParams,
  CreateUserGroupResponse,
  UpdateUserGroupParams,
  DeleteUserGroupParams,
  UserGroupMembersResponse,
  AddGroupMembersParams,
  RemoveGroupMembersParams,
  AssociatedModelsResponse,
  UserGroupTreeResponse,
  UngroupedMembersResponse,
  ConfigOverviewResponse,
  DeleteImpactResponse,
  CreateUserGroupV6Params,
  UpdateUserGroupV6Params,
  SetGroupPolicyParams,
  DeleteGroupPolicyParams,
  GroupConfigQuery,
  GroupConfigQueryResponse,
  AdminChannelsResponse,
  AddCustomChannelParams,
  AddCustomChannelResponse,
  HatcheryOkResponse,
  ImageListResponse,
  ImportImageParams,
  CloudImageListResponse,
  SetImageTypeVisibilityParams,
  AdminAgentTypesResponse,
  CreateAgentTypeParams,
  DeleteAgentTypeParams,
  SetAgentTypeEnabledParams,
  SetAgentTypeEnabledResponse,
  ImageHistoryResponse,
  ImageHistoryParams,
  SetImageUpdateNoticeParams,
  UpdateNoticesResponse,
  AdminInstancesResponse,
  AdminInstancesParams,
  AdminInstancesByUserGroupParams,
  AdminInstancesByUserGroupResponse,
  AuditLogsResponse,
  AuditLogsParams,
  CreateSecurityGroupParams,
  UpdateSecurityGroupParams,
  SecurityGroupListResponse,
  CloudPoliciesResponse,
  RequiredRulesResponse,
  CheckRulesResponse,
  RuleSetResponse,
  CreateRuleSetParams,
  CreateRuleSetResponse,
  UpdateRuleSetRulesParams,
  UpdateRuleSetRulesResponse,
  ImportRulesFromSgParams,
  ImportRulesFromSgResponse,
  QuotasResponse,
  SetQuotaParams,
  UsageDataParams,
  UsageDataResponse,
  UsageLogsParams,
  UsageLogsResponse,
  CloudVpcListResponse,
  CloudVpcListParams,
  CloudSubnetListResponse,
  GroupVpcConfigsResponse,
  CreateGroupVpcConfigParams,
  UpdateGroupVpcConfigParams,
  DeleteGroupVpcConfigParams,
  BatchCreateUserItem,
  BatchCreateUsersResponse,
  UserLimitResponse,
  UserVpcResponse,
  DepartmentsResponse,
  TerminalUrlResponse,
  DeniedActionsResponse,
  InstanceChannelsResponse,
  InstanceSkillsResponse,
  InstanceModelsResponse,
  AvailableModelsResponse,
  AvailableChannelsResponse,
  AdminAddModelResponse,
  AdminSetModelResponse,
  AdminSwitchPrimaryModelResponse,
  AdminDelModelResponse,
  AdminSetChannelResponse,
  AdminDelChannelResponse,
  // Enterprise Skill Library
  SkillCategoriesResponse,
  AdminSkillsResponse,
  AdminSkillsParams,
  AdminSkillDetailResponse,
  AdminSkillFilesResponse,
  AdminSkillTasksResponse,
  AdminSkillTasksParams,
  AdminSkillInstancesResponse,
  AdminSkillInstancesParams,
  AdminSkillCreateResponse,
  SkillReferencesResponse,
  AdminSkillDistributeParams,
  AdminSkillDistributeResponse,
  AdminSkillUninstallParams,
  AdminSkillUninstallResponse,
  SmhInfoResponse,
  SmhStatResponse,
  SmhPersonalSpacesParams,
  SmhPersonalSpacesResponse,
  SmhInstancesParams,
  SmhInstancesResponse,
  SmhInstanceSpaceParams,
  SmhInstanceSpaceResponse,
  // Roles
  AdminRolesResponse,
  CreateRoleParams,
  CreateRoleResponse,
  ToggleRoleVisibleResponse,
  AdminNoticesResponse,
  // Enterprise Plugin Library
  PluginCategoriesResponse,
  AdminPluginsResponse,
  AdminPluginsParams,
  AdminPluginDetailResponse,
  AdminPluginFilesResponse,
  AdminPluginTasksResponse,
  AdminPluginTasksParams,
  AdminPluginInstancesResponse,
  AdminPluginInstancesParams,
  AdminPluginCreateResponse,
  AdminPluginDistributeParams,
  AdminPluginDistributeResponse,
  AdminPluginUninstallParams,
  AdminPluginUninstallResponse,
  AdminPluginUpdateResponse,
  FavoritedPluginsResponse,
  BatchUpgradeResponse,
  // Memory Pro
  MemoryOverviewResponse,
  MemoryProActivateParams,
  MemoryProActivateResponse,
  MemoryProReleaseResponse,
  MemoryPlanSwitchParams,
  MemoryPlanSwitchResponse,
  MemoryInstancesParams,
  MemoryInstancesResponse,
  PluginUpgradeCandidatesResponse,
  PluginUpgradeExecuteResponse,
  MemoryDefaultPlanResponse,
  MemoryDefaultPlanUpdateParams,
  MemoryDefaultPlanUpdateResponse,
  MemoryTdaiConfigResponse,
  MemoryTdaiConfigUpdateResponse,
  // Memory 分组策略
  MemoryGroupPoliciesResponse,
  MemoryGroupPolicyCreateParams,
  MemoryGroupPolicyUpdateParams,
  MemoryGroupPolicyDeleteParams,
  // Enterprise MCP Library
  AdminMCPsParams,
  AdminMCPsResponse,
  AdminMCPCreateParams,
  AdminMCPCreateResponse,
  AdminMCPUpdateParams,
  AdminMCPUpdateResponse,
  AdminMCPMetaParams,
  AdminMCPMetaResponse,
  AdminMCPDetailResponse,
  AdminMCPDeleteResponse,
  AdminMCPVersionsResponse,
  AdminMCPDistributeParams,
  AdminMCPDistributeResponse,
  AdminMCPTasksParams,
  AdminMCPTasksResponse,
  AdminMCPInstancesParams,
  AdminMCPInstancesResponse,
  // Admin Agent Commands（命令下发）
  AgentCommandsListParams,
  AgentCommandsListResponse,
  AgentCommandItem,
  AgentCommandCreateParams,
  AgentCommandUpdateParams,
  AgentCommandDeleteParams,
  AgentCommandDispatchParams,
  AgentCommandDispatchResponse,
  AgentCommandContinueDispatchResponse,
  AgentCommandAbortDispatchResponse,
  AgentCommandTaskItem,
  AgentCommandTasksListParams,
  AgentCommandTasksListResponse,
  AgentCommandTaskDetailParams,
  AgentCommandTaskDetailResponse,


} from '@/types/api';
import { fetchJSON, postForm, postFormRepeated, postFormData, postJSON, putJSON, deleteJSON, buildUrl } from './request';
import { axiosInstance } from './request';

/**
 * 管理后台 API
 * 所有接口需要 admin 角色权限
 */
export const adminApi = {
  // ==================== 用户管理 ====================

  /**
   * 获取当前用户数及用户数上限
   * GET /admin/user-limit
   */
  getUserLimit: () =>
    fetchJSON<UserLimitResponse>('/admin/user-limit'),

  /**
   * 获取用户自动创建的关联 VPC 信息及是否有阻断删除的资源
   * GET /admin/user-vpc?id=xxx
   */
  getUserVpc: (userId: number) =>
    fetchJSON<UserVpcResponse>(buildUrl('/admin/user-vpc', { id: userId })),

  /**
   * 获取所有用户列表（含已软删除的），支持分页
   * GET /admin/users?page=1&page_size=20
   */
  getUsers: (params?: AdminUsersParams) =>
    fetchJSON<AdminUsersResponse>(buildUrl('/admin/users', params as Record<string, string | number | undefined>)),

  /**
   * 创建新用户
   * POST /admin/create (application/json)
   */
  createUser: (params: CreateUserParams) =>
    postJSON<HatcheryOkResponse>('/admin/create', {
      username: params.username,
      password: params.password,
      role: params.role,
      instance_quota: params.instance_quota,
      token_quota_day: params.token_quota_day,
      token_quota_rules: params.token_quota_rules,
      email: params.email,
      group_ids: params.group_ids,
    }),

  /**
   * 软删除用户（管理员不可删除，有实例存在的用户不可删除）
   * POST /admin/delete?id=xxx
   */
  deleteUser: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/delete', { id }), {}),

  /**
   * 永久删除用户（用户名下不能有实例）
   * POST /admin/hard-delete?id=xxx
   */
  hardDeleteUser: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/hard-delete', { id }), {}),

  /**
   * 恢复已软删除的用户
   * POST /admin/restore?id=xxx
   */
  restoreUser: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/restore', { id }), {}),

  /**
   * 重置用户密码
   * POST /admin/reset-password?id=xxx (x-www-form-urlencoded)
   * 初始管理员（ID=1）只能通过 Token 认证重置
   */
  resetUserPassword: (params: ResetPasswordParams) =>
    postForm<HatcheryOkResponse>(
      buildUrl('/admin/reset-password', { id: params.id }),
      {
        password: params.password,
        ...(params.email ? { email: params.email } : {}),
      }
    ),

  /**
   * 更新用户属性（用户名和密码除外）
   * POST /admin/update-user?id=xxx (application/json)
   */
  updateUser: (id: number, params: UpdateUserParams) =>
    postJSON<HatcheryOkResponse>(
      buildUrl('/admin/update-user', { id }),
      params as unknown as Record<string, unknown>
    ),

  /**
   * 批量创建用户
   * POST /admin/batch-create (application/json)
   * 请求体: BatchCreateUserItem[]
   * 响应: { results: [{ username, ok, error? }] }
   */
  batchCreateUsers: (users: BatchCreateUserItem[]) =>
    postJSON<BatchCreateUsersResponse>('/admin/batch-create', users as unknown as Record<string, unknown>),

  // ==================== API Token 管理 ====================

  /**
   * 禁用用户的 API Token
   * POST /admin/token/disable?id={userID}
   */
  disableUserToken: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/token/disable', { id }), {}),

  /**
   * 启用用户的 API Token
   * POST /admin/token/enable?id={userID}
   */
  enableUserToken: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/token/enable', { id }), {}),

  /**
   * 批量为所有用户生成并导出 API Token
   * POST /admin/export-tokens (application/json)
   * 已有 Token 的用户保留不变，没有 Token 的用户自动生成
   * 响应: [{ id, username, token }]
   */
  exportTokens: () =>
    postJSON<Array<{ id: number; username: string; token: string }>>('/admin/export-tokens', {}),

  // ==================== 站点配置 ====================

  /**
   * 获取站点配置
   * GET /admin/config
   */
  getConfig: async (): Promise<SiteConfigResponse> => {
    const res = await fetchJSON<SiteConfigResponse>('/admin/config');
    // 兼容老后端:可能把 *_rules 当字符串返回,这里统一解析为数组
    const cfg = res.config as SiteConfigInfo & {
      global_token_quota_rules?: unknown;
      default_token_quota_rules?: unknown;
    };
    if (typeof cfg.global_token_quota_rules === 'string') {
      try {
        cfg.global_token_quota_rules = JSON.parse(cfg.global_token_quota_rules);
      } catch {
        cfg.global_token_quota_rules = null;
      }
    }
    if (typeof cfg.default_token_quota_rules === 'string') {
      try {
        cfg.default_token_quota_rules = JSON.parse(cfg.default_token_quota_rules);
      } catch {
        cfg.default_token_quota_rules = null;
      }
    }
    return res;
  },

  /**
   * 更新站点配置（站点名称、Logo、全局配额、默认用户配额、终端开关、面板端口开关）
   * POST /admin/config (multipart/form-data)
   * 后面逐步废弃掉吧，多人开发合并导致问题太多了。请使用updateConfigByOpts
   */
  updateConfig: (params: {
    name?: string;
    logo?: File;
    global_token_quota_day?: number;
    global_token_quota_period?: 'day' | 'month';
    terminal_enabled?: boolean;
    gateway_ui_enable?: boolean;
    gateway_ui_addr_type?: string;
    default_instance_quota?: number;
    default_token_quota_day?: number;
    global_token_quota_rules?: string;       // JSON 数组字符串
    default_token_quota_rules?: string;      // JSON 数组字符串
    browser_vnc_enable?: boolean;
    chat_view_enabled?: boolean;
    doctor_enabled?: boolean;
    user_config_model_enabled?: boolean;
    user_config_channel_enabled?: boolean;
    model_quota_enabled?: boolean;
  }) => {
    const formData = new FormData();
    if (params.name) formData.append('name', params.name);
    if (params.logo) formData.append('logo', params.logo);
    if (params.global_token_quota_day !== undefined) formData.append('global_token_quota_day', String(params.global_token_quota_day));
    if (params.global_token_quota_period !== undefined) formData.append('global_token_quota_period', params.global_token_quota_period);
    if (params.terminal_enabled !== undefined) formData.append('terminal_enabled', String(params.terminal_enabled));
    if (params.gateway_ui_enable !== undefined) formData.append('gateway_ui_enable', String(params.gateway_ui_enable));
    if (params.gateway_ui_addr_type !== undefined) formData.append('gateway_ui_addr_type', params.gateway_ui_addr_type);
    if (params.default_instance_quota !== undefined) formData.append('default_instance_quota', String(params.default_instance_quota));
    if (params.default_token_quota_day !== undefined) formData.append('default_token_quota_day', String(params.default_token_quota_day));
    if (params.global_token_quota_rules !== undefined) formData.append('global_token_quota_rules', params.global_token_quota_rules);
    if (params.default_token_quota_rules !== undefined) formData.append('default_token_quota_rules', params.default_token_quota_rules);
    if (params.browser_vnc_enable !== undefined) formData.append('browser_vnc_enable', String(params.browser_vnc_enable));
    if (params.chat_view_enabled !== undefined) formData.append('chat_view_enabled', String(params.chat_view_enabled));
    if (params.doctor_enabled !== undefined) formData.append('doctor_enabled', String(params.doctor_enabled));
    if (params.user_config_model_enabled !== undefined) formData.append('user_config_model_enabled', String(params.user_config_model_enabled));
    if (params.user_config_channel_enabled !== undefined) formData.append('user_config_channel_enabled', String(params.user_config_channel_enabled));
    if (params.model_quota_enabled !== undefined) formData.append('model_quota_enabled', String(params.model_quota_enabled));
    return postFormData<HatcheryOkResponse>('/admin/config', formData);
  },
  /**
   * 更新站点配置（站点名称、Logo、全局配额、终端开关、面板端口开关）
   * POST /admin/config (multipart/form-data)
   */
  updateConfigByOpts: (opt: {
    name?: string, logo?: File, global_token_quota_day?: number, terminal_enabled?: boolean, gateway_ui_enable?: boolean;
    sso_im_types?: string[]
  }) => {
    const { name, logo, global_token_quota_day, terminal_enabled, gateway_ui_enable, sso_im_types } = opt;
    const formData = new FormData();
    if (name) formData.append('name', name);
    if (logo) formData.append('logo', logo);
    if (global_token_quota_day !== undefined) formData.append('global_token_quota_day', String(global_token_quota_day));
    if (terminal_enabled !== undefined) formData.append('terminal_enabled', String(terminal_enabled));
    if (gateway_ui_enable !== undefined) formData.append('gateway_ui_enable', String(gateway_ui_enable));
    if (sso_im_types !== undefined) formData.append('sso_im_types', JSON.stringify(sso_im_types));
    return postFormData<HatcheryOkResponse>('/admin/config', formData);
  },

  /**
   * 从 OneID 同步企业信息（名称和 Logo）
   * POST /admin/oneid-sync-enterprise
   */
  syncEnterprise: () =>
    postJSON<HatcheryOkResponse>('/admin/oneid-sync-enterprise', {}),

  /**
   * 手动触发全量同步 OneID 用户（身份、角色、部门、岗位）
   * POST /admin/oneid-sync-users
   *
   * @param syncDept true=强制本次进入部门落地（Step 1.8 + 2.6）；省略/false=仅当本地已有
   *                 oneid_dept 组才继续维护。
   *                 首次同步部门作为分组时应传 true，之后常规同步可省略。
   */
  syncOneIDUsers: (syncDept?: boolean) =>
    postJSON<{
      ok: boolean;
      message: string;
      profile_count?: number;
      dept_count?: number;
      user_count?: number;
      affected_users?: Array<{
        username: string;
        instance_count: number;
        action: 'disabled' | 'hard_deleted';
        vpc_id: string | null;
        vpc_has_resources?: boolean;
      }> | null;
      /** 落地后本地 user_groups 中 source=oneid_dept AND to_be_deleted=0 的数量（v6.13） */
      dept_group_count?: number;
      /** 本次新打 to_be_deleted=1 的分组明细（v6.13 类型升级为对象数组） */
      affected_dept_groups?: Array<{ group_id: number; full_path: string }> | null;
      /** 本次发生父节点切换的 oneid_dept 组 ID 列表（v6.13 新增） */
      change_parent_group_ids?: number[] | null;
      /** 本次被从旧组迁出的 (user_id, from_group_id) 事件列表（v6.13 新增） */
      move_group_user_ids?: Array<{ user_id: number; from_group_id: number }> | null;
      /** 本次未落地的部门明细 */
      landing_failures?: Array<{
        department_id: string;
        department_name: string;
        stage: string;
        err: string;
      }> | null;
    }>('/admin/oneid-sync-users', syncDept ? { sync_dept: true } : {}),

  /**
   * 查询 OneID 用户同步状态
   * GET /admin/oneid-sync-users/status
   */
  getSyncStatus: () =>
    fetchJSON<{ running: boolean; last_sync: string; profile_count: number; dept_count: number; oneid_user_count: number }>(
      '/admin/oneid-sync-users/status'
    ),

  /**
   * 获取 OneID 部门树
   * GET /admin/departments
   */
  getDepartments: () =>
    fetchJSON<DepartmentsResponse>('/admin/departments'),

  /**
   * 获取 OneID 免登链接（用于跳转 OneID 管理后台等页面）
   * GET /auth/oneid/jump?module=admin&route=users
   * 返回: { link: string, expires_in: number } 或 OneID 原始错误响应
   */
  getOneIDJumpLink: (module: string = 'admin', route: string = '') =>
    fetchJSON<{ link?: string; expires_in?: number; data?: { link: string; expires_in: number }; code?: number; msg?: string }>(
      buildUrl('/auth/oneid/jump', { module, route })
    ),

  /**
   * 更新 CVM 云服务器配置
   * POST /admin/config/cvm (x-www-form-urlencoded)
   */
  updateCvmConfig: (params: UpdateCvmConfigParams) =>
    postForm<HatcheryOkResponse>('/admin/config/cvm', params as Record<string, string | number | undefined>),

  /**
   * 通用模板修改（含公网管理等）
   * POST /admin/config/template (application/json)
   * 只传需要修改的字段片段（patch），未传字段保持不变
   */
  updateTemplate: (patch: UpdateTemplatePatch) =>
    postJSON<HatcheryOkResponse>('/admin/config/template', patch as unknown as Record<string, unknown>),

  // ==================== VPC & 子网 ====================

  /**
   * 获取当前 Region 下的 VPC 列表（支持分页和搜索）
   * GET /admin/vpc/cloud?offset=0&limit=20&vpc_name=xxx&vpc_id=xxx
   */
  getCloudVpcs: (params?: CloudVpcListParams) =>
    fetchJSON<CloudVpcListResponse>(buildUrl('/admin/vpc/cloud', params as Record<string, string | number | undefined>)),

  /**
   * 获取指定 VPC 和可用区下的子网列表
   * GET /admin/subnet/cloud?vpc_id=xxx&zone=yyy
   */
  getCloudSubnets: (vpcId: string, zone: string) =>
    fetchJSON<CloudSubnetListResponse>(buildUrl('/admin/subnet/cloud', { vpc_id: vpcId, zone })),

  // ==================== 分组 VPC 网络配置（v10 新增） ====================

  /**
   * 获取分组网络配置列表（含预设策略）
   * GET /admin/group-vpc-configs
   */
  getGroupVpcConfigs: () =>
    fetchJSON<GroupVpcConfigsResponse>('/admin/group-vpc-configs'),

  /**
   * 新增分组网络配置
   * POST /admin/group-vpc-configs/create (application/json)
   * v10：group_ids 支持多分组
   */
  createGroupVpcConfig: (params: CreateGroupVpcConfigParams) =>
    postJSON<HatcheryOkResponse>('/admin/group-vpc-configs/create', params as unknown as Record<string, unknown>),

  /**
   * 编辑分组网络配置
   * POST /admin/group-vpc-configs/update (application/json)
   */
  updateGroupVpcConfig: (params: UpdateGroupVpcConfigParams) =>
    postJSON<HatcheryOkResponse>('/admin/group-vpc-configs/update', params as unknown as Record<string, unknown>),

  /**
   * 删除分组网络配置
   * POST /admin/group-vpc-configs/delete (application/json)
   */
  deleteGroupVpcConfig: (params: DeleteGroupVpcConfigParams) =>
    postJSON<HatcheryOkResponse>('/admin/group-vpc-configs/delete', params as unknown as Record<string, unknown>),

  // ==================== 安全组配置 ====================

  /**
   * 查询当前配置的安全组详情
   * GET /admin/config/security-group
   * 响应透传腾讯云 DescribeSecurityGroups 接口
   */
  getSecurityGroup: () =>
    fetchJSON<Record<string, unknown>>('/admin/config/security-group'),

  /**
   * 列出当前账号/地域下的安全组（支持分页和关键字搜索）
   * GET /admin/config/security-group/list
   * 用于"选择已有安全组"弹窗
   */
  getSecurityGroupList: (params?: { offset?: number; limit?: number; keyword?: string; security_group_id?: string }) =>
    fetchJSON<SecurityGroupListResponse>(buildUrl('/admin/config/security-group/list', params as Record<string, string | number | undefined>)),

  /**
   * 查询指定安全组的规则列表（用于预览）
   * GET /admin/config/security-group/cloud-policies?security_group_id=xxx
   * 响应透传腾讯云 DescribeSecurityGroupPolicies
   */
  getCloudPolicies: (securityGroupId: string) =>
    fetchJSON<CloudPoliciesResponse>(
      buildUrl('/admin/config/security-group/cloud-policies', { security_group_id: securityGroupId })
    ),

  /**
   * 查询内部配置的 ClawPro 所需安全组规则列表（常用规则模板）
   * GET /admin/config/security-group/required-rules
   * 用于新建安全组弹窗中"快速添加常用规则"
   */
  getRequiredRules: () =>
    fetchJSON<RequiredRulesResponse>('/admin/config/security-group/required-rules'),

  /**
   * 检查指定安全组是否满足 ClawPro 所需规则
   * GET /admin/config/security-group/check-rules?security_group_id=xxx
   * 满足时返回空的 missing_rules 列表；不满足时返回缺少的规则列表
   */
  checkSecurityGroupRules: (securityGroupId: string) =>
    fetchJSON<CheckRulesResponse>(
      buildUrl('/admin/config/security-group/check-rules', { security_group_id: securityGroupId })
    ),

  /**
   * 创建新安全组并自动绑定到站点配置
   * POST /admin/config/security-group (application/json)
   * 请求体透传给腾讯云 CreateSecurityGroup 接口
   */
  createSecurityGroup: (params: CreateSecurityGroupParams) =>
    postJSON<Record<string, unknown>>(
      '/admin/config/security-group',
      params as unknown as Record<string, unknown>
    ),

  /**
   * 修改当前绑定的安全组属性
   * PUT /admin/config/security-group (application/json)
   * SecurityGroupId 由服务端自动填充
   */
  updateSecurityGroup: (params: UpdateSecurityGroupParams) =>
    putJSON<Record<string, unknown>>(
      '/admin/config/security-group',
      params as unknown as Record<string, unknown>
    ),

  /**
   * 查询当前安全组的规则列表
   * GET /admin/config/security-group/policies
   * 响应透传腾讯云 DescribeSecurityGroupPolicies
   */
  getSecurityGroupPolicies: () =>
    fetchJSON<Record<string, unknown>>('/admin/config/security-group/policies'),

  /**
   * 创建安全组规则
   * POST /admin/config/security-group/policies (application/json)
   * 请求体透传给腾讯云 CreateSecurityGroupPolicies
   */
  createSecurityGroupPolicies: (params: Record<string, unknown>) =>
    postJSON<Record<string, unknown>>('/admin/config/security-group/policies', params),

  /**
   * 替换安全组规则
   * PUT /admin/config/security-group/policies (application/json)
   * 请求体透传给腾讯云 ReplaceSecurityGroupPolicy
   */
  replaceSecurityGroupPolicies: (params: Record<string, unknown>) =>
    putJSON<Record<string, unknown>>('/admin/config/security-group/policies', params),

  /**
   * 删除安全组规则
   * DELETE /admin/config/security-group/policies (application/json)
   * 请求体透传给腾讯云 DeleteSecurityGroupPolicies
   */
  deleteSecurityGroupPolicies: (params: Record<string, unknown>) =>
    deleteJSON<Record<string, unknown>>('/admin/config/security-group/policies', params),

  // ==================== 规则集管理 (sg-ruleset-projection 架构) ====================

  /**
   * 获取规则集（规则真值源）
   * GET /admin/config/security-group/ruleset
   * 返回当前 RuleSet、版本号、投影到的云端安全组列表（仅 ACTIVE 成员）
   */
  getRuleSet: () =>
    fetchJSON<RuleSetResponse>('/admin/config/security-group/ruleset'),

  /**
   * 创建规则集和第一个 ACTIVE 云端安全组
   * POST /admin/config/security-group/rulesets (application/json)
   * 用于初始化企业的规则管理基础设施（第一次创建时调用）
   */
  createRuleSet: (params: CreateRuleSetParams) =>
    postJSON<CreateRuleSetResponse>('/admin/config/security-group/rulesets', params as unknown as Record<string, unknown>),

  /**
   * 更新规则集中的规则
   * POST /admin/config/security-group/ruleset/rules (application/json)
   * 更新会自动增加版本号并投影（fan-out）到所有 ACTIVE 云端安全组
   */
  updateRuleSetRules: (params: UpdateRuleSetRulesParams) =>
    postJSON<UpdateRuleSetRulesResponse>(
      '/admin/config/security-group/ruleset/rules',
      params as unknown as Record<string, unknown>
    ),

  /**
   * 从外部安全组导入规则到规则集
   * POST /admin/config/security-group/ruleset/import-from-sg (application/json)
   * 用于从腾讯云安全组导入现有规则到规则集
   */
  importRulesFromSg: (params: ImportRulesFromSgParams) =>
    postJSON<ImportRulesFromSgResponse>(
      '/admin/config/security-group/ruleset/import-from-sg',
      {
        name: params.name || 'ClawPro-Default',
        source_sg_id: params.security_group_id,
        ...(params.auto_fix_rules ? { auto_fix_rules: true } : {}),
      }
    ),

  // ==================== 模型配置 ====================

  /**
   * 获取所有模型配置
   * GET /admin/models
   * 注意：返回的 APIKey 已清空
   */
  getModels: () =>
    fetchJSON<ModelListResponse>('/admin/models'),

  /**
   * 创建新的 AI 模型配置
   * POST /admin/models/create (x-www-form-urlencoded)
   */
  createModel: (params: CreateModelParams) => {
    const { input_types, ...rest } = params;
    const entries: Array<[string, string]> = [];
    for (const [key, val] of Object.entries(rest)) {
      if (val !== undefined && val !== null) {
        entries.push([key, String(val)]);
      }
    }
    if (input_types && input_types.length > 0) {
      for (const t of input_types) {
        entries.push(['input_types', t]);
      }
    }
    return postFormRepeated<HatcheryOkResponse>('/admin/models/create', entries);
  },

  /**
   * 更新 AI 模型配置
   * POST /admin/models/update?id=xxx (x-www-form-urlencoded)
   * api_key 留空则不更新
   * input_types 以重复字段方式发送（input_types=text&input_types=image）
   */
  updateModel: (id: number, params: UpdateModelParams) => {
    const { input_types, ...rest } = params;
    const entries: Array<[string, string]> = [];
    for (const [key, val] of Object.entries(rest)) {
      if (val !== undefined && val !== null) {
        entries.push([key, String(val)]);
      }
    }
    if (input_types && input_types.length > 0) {
      for (const t of input_types) {
        entries.push(['input_types', t]);
      }
    }
    return postFormRepeated<HatcheryOkResponse>(
      buildUrl('/admin/models/update', { id }),
      entries,
    );
  },

  /**
   * 删除 AI 模型配置
   * POST /admin/models/delete?id=xxx
   * 系统内置记录不可删除
   */
  deleteModel: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/models/delete', { id }), {}),

  /**
   * 切换模型启用/禁用状态
   * POST /admin/models/toggle?id=xxx
   */
  toggleModel: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/models/toggle', { id }), {}),

  /**
   * 切换模型默认配置状态
   * POST /admin/models/toggle-default?id=xxx
   */
  toggleDefault: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/models/toggle-default', { id }), {}),

  /**
   * 更新模型可见范围
   * POST /admin/models/visibility?id=xxx (application/json)
   */
  updateModelVisibility: (id: number, params: UpdateModelVisibilityParams) =>
    postJSON<HatcheryOkResponse>(buildUrl('/admin/models/visibility', { id }), params as unknown as Record<string, unknown>),

  /**
   * 测试模型连通性
   * POST /admin/models/connectivity (application/json)
   * 支持两种调用方式：
   * 1. 按已保存模型 ID 探测：传入 id 参数
   * 2. 用临时凭证探测：传入 body {url, api_key, model_type}
   */
  testModelConnectivity: (params: ConnectivityTestParams, id?: number) =>
    postJSON<ConnectivityTestResponse>(
      id ? buildUrl('/admin/models/connectivity', { id }) : '/admin/models/connectivity',
      params as unknown as Record<string, unknown>,
      { silentError: true },
    ),

  // ==================== 用户分组 ====================

  /**
   * 获取所有用户组列表（分页，page_size 默认传 1000 全量拉取）
   * GET /admin/user-groups
   */
  getUserGroups: (params?: { page?: number; page_size?: number }) =>
    fetchJSON<UserGroupsResponse>(buildUrl('/admin/user-groups', params as Record<string, string | number | undefined>)),

  /**
   * 创建用户组
   * POST /admin/user-groups/create (application/json)
   */
  createUserGroup: (params: CreateUserGroupParams) =>
    postJSON<CreateUserGroupResponse>('/admin/user-groups/create', params as unknown as Record<string, unknown>),

  /**
   * 修改用户组信息
   * POST /admin/user-groups/update (application/json)
   */
  updateUserGroup: (params: UpdateUserGroupParams) =>
    postJSON<HatcheryOkResponse>('/admin/user-groups/update', params as unknown as Record<string, unknown>),

  /**
   * 删除用户组
   * POST /admin/user-groups/delete (application/json)
   */
  deleteUserGroup: (params: DeleteUserGroupParams) =>
    postJSON<HatcheryOkResponse>('/admin/user-groups/delete', params as unknown as Record<string, unknown>),

  /**
   * 查询组内成员列表
   * GET /admin/user-groups/members?id=xxx[&include_descendants=true&q=...]
   * v10：支持 include_descendants（沿闭包表展开子孙）和 q（用户名模糊查询）
   */
  getUserGroupMembers: (
    id: number,
    params?: { page?: number; page_size?: number; include_descendants?: boolean; q?: string },
  ) => {
    const normalized: Record<string, string | number | undefined> = { id };
    if (params?.page !== undefined) normalized.page = params.page;
    if (params?.page_size !== undefined) normalized.page_size = params.page_size;
    if (params?.include_descendants !== undefined)
      normalized.include_descendants = params.include_descendants ? 'true' : 'false';
    if (params?.q !== undefined && params.q !== '') normalized.q = params.q;
    return fetchJSON<UserGroupMembersResponse>(buildUrl('/admin/user-groups/members', normalized));
  },

  /**
   * 批量添加成员
   * POST /admin/user-groups/members/add (application/json)
   */
  addGroupMembers: (params: AddGroupMembersParams) =>
    postJSON<HatcheryOkResponse>('/admin/user-groups/members/add', params as unknown as Record<string, unknown>),

  /**
   * 批量移除成员
   * POST /admin/user-groups/members/remove (application/json)
   */
  removeGroupMembers: (params: RemoveGroupMembersParams) =>
    postJSON<HatcheryOkResponse>('/admin/user-groups/members/remove', params as unknown as Record<string, unknown>),

  /**
   * 查询指定用户组关联的模型列表
   * GET /admin/user-groups/associated-models?group_id=xxx
   */
  getGroupAssociatedModels: (groupId: number) =>
    fetchJSON<AssociatedModelsResponse>(buildUrl('/admin/user-groups/associated-models', { group_id: groupId })),

  /**
   * 获取分组树（v6 新增）
   * GET /admin/user-groups/tree
   */
  getUserGroupsTree: (params?: {
    q?: string;
    with_user_counts?: boolean;
    with_health?: boolean;
    sources?: string;
  }) => {
    // 将 boolean 转成字符串，buildUrl 只接受 string | number | undefined
    const normalized: Record<string, string | number | undefined> = {};
    if (params?.q !== undefined) normalized.q = params.q;
    if (params?.with_user_counts !== undefined) normalized.with_user_counts = params.with_user_counts ? "true" : "false";
    if (params?.with_health !== undefined) normalized.with_health = params.with_health ? "true" : "false";
    if (params?.sources !== undefined) normalized.sources = params.sources;
    return fetchJSON<UserGroupTreeResponse>(buildUrl('/admin/user-groups/tree', normalized));
  },

  /**
   * 获取未分组成员列表（v6 新增）
   * GET /admin/user-groups/ungrouped/members
   */
  getUngroupedMembers: (params?: { page?: number; page_size?: number; q?: string }) =>
    fetchJSON<UngroupedMembersResponse>(
      buildUrl('/admin/user-groups/ungrouped/members', params as Record<string, string | number | undefined>),
    ),

  /**
   * 获取配置总览（v6 新增）
   * GET /admin/user-groups/config-overview?group_ids=3,7,8&keys=model,channel
   */
  getGroupConfigOverview: (params: { group_ids: string; keys?: string }) =>
    fetchJSON<ConfigOverviewResponse>(
      buildUrl('/admin/user-groups/config-overview', params as Record<string, string | number | undefined>),
    ),

  /**
   * 获取删除影响报告（v6 新增）
   * GET /admin/user-groups/delete-impact?group_ids=3,4
   */
  getGroupDeleteImpact: (groupIds: string) =>
    fetchJSON<DeleteImpactResponse>(
      buildUrl('/admin/user-groups/delete-impact', { group_ids: groupIds }),
    ),

  /**
   * 创建用户组（v6：支持 parent_id）
   * POST /admin/user-groups/create
   */
  createUserGroupV6: (params: CreateUserGroupV6Params) =>
    postJSON<CreateUserGroupResponse>('/admin/user-groups/create', params as unknown as Record<string, unknown>),

  /**
   * 更新用户组（v6：支持 parent_id 三态换父）
   * POST /admin/user-groups/update
   */
  updateUserGroupV6: (params: UpdateUserGroupV6Params) =>
    postJSON<HatcheryOkResponse>('/admin/user-groups/update', params as unknown as Record<string, unknown>),

  /**
   * 设置分组级平台策略（v6 新增）
   * POST /admin/group-config/policy
   */
  setGroupConfigPolicy: (params: SetGroupPolicyParams) =>
    postJSON<HatcheryOkResponse>('/admin/group-config/policy', params as unknown as Record<string, unknown>),

  /**
   * 删除分组级平台策略（v6 新增）
   * POST /admin/group-config/policy/delete
   */
  deleteGroupConfigPolicy: (params: DeleteGroupPolicyParams) =>
    postJSON<HatcheryOkResponse>('/admin/group-config/policy/delete', params as unknown as Record<string, unknown>),

  /**
   * 查询某类/某 key 下所有已绑定的分组（v6 新增）
   * GET /admin/group-config/groups?queries=<url-encoded JSON>
   */
  queryGroupConfigs: (queries: GroupConfigQuery[]) =>
    fetchJSON<GroupConfigQueryResponse>(
      buildUrl('/admin/group-config/groups', { queries: JSON.stringify(queries) } as Record<string, string | number | undefined>),
    ),

  // ==================== 通道配置 ====================

  /**
   * 获取所有预定义通道及其启用状态
   * GET /admin/channels
   */
  getChannels: () =>
    fetchJSON<AdminChannelsResponse>('/admin/channels'),

  /**
   * 切换通道启用/禁用状态
   * POST /admin/channels/toggle?id=xxx
   */
  toggleChannel: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/channels/toggle', { id }), {}),

  /**
   * 更新通道应用范围（v6 新增）
   * POST /admin/channels/visibility
   * body: { channel_id, visibility_type, group_ids? }
   */
  updateChannelVisibility: (params: {
    channel_id: string | number;
    visibility_type: 'all' | 'group';
    group_ids?: number[];
  }) =>
    postJSON<HatcheryOkResponse>('/admin/channels/visibility', params as unknown as Record<string, unknown>),

  /**
   * 添加自定义通道
   * POST /admin/channels/add (application/json)
   */
  addCustomChannel: (params: AddCustomChannelParams) =>
    postJSON<AddCustomChannelResponse>('/admin/channels/add', params as unknown as Record<string, unknown>),

  /**
   * 删除自定义通道
   * POST /admin/channels/delete?id=xxx
   * 仅允许删除自定义通道（Custom=true）
   */
  deleteChannel: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/channels/delete', { id }), {}),

  // ==================== 镜像管理 ====================

  /**
   * 获取镜像管理列表
   * GET /admin/images
   */
  getImages: () =>
    fetchJSON<ImageListResponse>('/admin/images'),

  /**
   * 获取腾讯云私有镜像列表（排除已导入到本地的镜像）
   * GET /admin/images/cloud
   * 始终返回 JSON
   */
  getCloudImages: () =>
    fetchJSON<CloudImageListResponse>('/admin/images/cloud'),

  /**
   * 从腾讯云导入镜像信息
   * POST /admin/images/import (x-www-form-urlencoded)
   */
  importImage: (params: ImportImageParams) =>
    postForm<HatcheryOkResponse>('/admin/images/import', {
      image_id: params.image_id,
      image_name: params.image_name,
      agent_type: params.agent_type,
      agent_version: params.agent_version,
    }),

  /**
   * 删除镜像记录（仅删除本地数据库记录）
   * POST /admin/images/delete?id=xxx
   */
  deleteImage: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/images/delete', { id }), {}),

  /**
   * 更新镜像信息（Agent 类型和版本号）
   * POST /admin/images/update
   */
  updateImage: (params: { id: number; agent_type: string; agent_version: string }) =>
    postForm<HatcheryOkResponse>('/admin/images/update', {
      id: params.id,
      agent_type: params.agent_type,
      agent_version: params.agent_version,
    }),

  /**
   * 切换镜像启用状态（按 Agent 类型独立启用，每类型最多一个启用）
   * POST /admin/images/enable?id=xxx
   */
  enableImage: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/images/enable', { id }), {}),

  /**
   * 设置用户端首选 Agent 类型
   * POST /admin/images/set-default-type
   */
  setDefaultAgentType: (agentType: string) =>
    postForm<HatcheryOkResponse>('/admin/images/set-default-type', { agent_type: agentType }),

  /**
   * 设置镜像类型（agent_type）的应用范围（v10 新增）
   * POST /admin/images/type-visibility
   * body: { agent_type, visibility_type, group_ids? }
   */
  setImageTypeVisibility: (params: SetImageTypeVisibilityParams) =>
    postJSON<HatcheryOkResponse>('/admin/images/type-visibility', params as unknown as Record<string, unknown>),

  // ==================== Agent 类型管理 ====================

  /**
   * 获取所有智能体类型列表（含内置 + 自定义，含能力位和镜像统计）
   * GET /admin/agent-types
   */
  getAgentTypes: () =>
    fetchJSON<AdminAgentTypesResponse>('/admin/agent-types'),

  /**
   * 创建自定义智能体类型
   * POST /admin/agent-types/create
   * body: { name, compatible_with? }
   */
  createAgentType: (params: CreateAgentTypeParams) =>
    postForm<HatcheryOkResponse>('/admin/agent-types/create', {
      name: params.name,
      ...(params.compatible_with ? { compatible_with: params.compatible_with } : {}),
    }),

  /**
   * 删除自定义智能体类型
   * POST /admin/agent-types/delete
   * body: { name }
   */
  deleteAgentType: (params: DeleteAgentTypeParams) =>
    postForm<HatcheryOkResponse>('/admin/agent-types/delete', { name: params.name }),

  /**
   * 启用 / 禁用 / 切换 用户端可选的 Agent 类型（用户可见开关，类型维度）。
   * POST /admin/agent-types/enabled
   *
   * - `enabled` 与 `toggle` 二者必须且只能传一个；
   * - 当前用户端首选 (default_agent_type) 不允许被禁用，后端会返回 400。
   * - 注意：本接口只改变类型可见性，不影响该类型下哪个镜像被启用。
   *   选定/切换该类型的目标镜像版本仍走 `enableImage` (POST /admin/images/enable)。
   */
  setAgentTypeEnabled: (params: SetAgentTypeEnabledParams) => {
    const body: Record<string, string | number | undefined> = {
      agent_type: params.agent_type,
    };
    if (params.toggle !== undefined) {
      body.toggle = params.toggle ? 'true' : 'false';
    } else if (params.enabled !== undefined) {
      body.enabled = params.enabled ? 'true' : 'false';
    }
    return postForm<SetAgentTypeEnabledResponse>('/admin/agent-types/enabled', body);
  },

  // ==================== 镜像更新历史 & 通知 ====================

  /**
   * 获取官方镜像更新历史（全局数据，不按租户隔离）
   * GET /admin/images/history
   */
  getImageHistory: (params?: ImageHistoryParams) =>
    fetchJSON<ImageHistoryResponse>(buildUrl('/admin/images/history', params as unknown as Record<string, string | number | undefined>)),

  /**
   * 开启或关闭当前租户内某个官方镜像的更新提示
   * POST /admin/images/update-notice
   */
  setImageUpdateNotice: (params: SetImageUpdateNoticeParams) =>
    postForm<HatcheryOkResponse>('/admin/images/update-notice', {
      image_id: params.image_id,
      enabled: params.enabled ? '1' : '0',
    }),

  /**
   * 获取当前用户可见的官方镜像更新提示列表（用户端接口）
   * GET /openclaw/images/update-notices
   */
  getUpdateNotices: () =>
    fetchJSON<UpdateNoticesResponse>('/openclaw/images/update-notices'),

  // ==================== 实例监控 ====================

  /**
   * 获取所有用户的实例列表（含所属用户名），支持分页
   * GET /admin/instances?page=1&page_size=20
   */
  getInstances: (params?: AdminInstancesParams) =>
    fetchJSON<AdminInstancesResponse>(buildUrl('/admin/instances', params as Record<string, string | number | undefined>)),

  /**
   * 按 (user_id, group_id) 精确对或 group_ids 子树批量查询实例清单（不分页，一次返回全量）
   * POST /admin/instances/by-user-group
   * 用于"按分组 → 所属用户 → 其所有 Agent"的快速筛选（管控端存量实例处理弹窗）
   */
  getInstancesByUserGroup: (params: AdminInstancesByUserGroupParams) =>
    postJSON<AdminInstancesByUserGroupResponse>(
      '/admin/instances/by-user-group',
      params as unknown as Record<string, unknown>,
    ),

  /**
   * 管理员删除实例（可删除任意用户的实例）
   * POST /admin/instances/delete (x-www-form-urlencoded)
   */
  deleteInstance: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/instances/delete', { id }),

  /**
   * 管理员批量删除实例（v9：上限 100 个）
   * POST /admin/instances/delete (application/json)
   */
  batchDeleteInstances: (ids: number[]) =>
    postJSON<{
      ok: boolean;
      results: Array<{
        id: number;
        instance_id?: string;
        name: string;
        /** started / deleted / failed */
        status: string;
        message: string;
      }>;
    }>('/admin/instances/delete', { ids } as unknown as Record<string, unknown>),

  /**
   * 管理员开机实例
   * POST /admin/instances/start (x-www-form-urlencoded)
   */
  startInstance: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/instances/start', { id }),

  /**
   * 管理员关机实例
   * POST /admin/instances/stop (x-www-form-urlencoded)
   */
  stopInstance: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/instances/stop', { id }),

  /**
   * 管理员重启实例
   * POST /admin/instances/reboot (x-www-form-urlencoded)
   */
  rebootInstance: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/instances/reboot', { id }),

  /**
   * 管理员重装实例
   * POST /admin/instances/reset (x-www-form-urlencoded)
   */
  resetInstance: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/instances/reset', { id }),

  /**
   * 管理员获取任意实例的终端登录 URL
   * POST /admin/instances/terminal-url (x-www-form-urlencoded)
   */
  getTerminalUrl: (id: number) =>
    postForm<TerminalUrlResponse>('/admin/instances/terminal-url', { id }),

  /**
   * 管理员批量查询实例禁用操作
   * POST /admin/instances/denied-actions (application/json)
   *
   * 后端对单次请求的 ids 数量有上限（100）。此处在前端自动按 100 分片并行请求，
   * 合并后返回，调用方无需感知分片逻辑。
   */
  getDeniedActions: async (ids: number[]): Promise<DeniedActionsResponse> => {
    if (!ids || ids.length === 0) {
      return { instances: [] };
    }
    const CHUNK_SIZE = 100;
    // 去重，避免不必要的重复请求
    const uniqIds = Array.from(new Set(ids));
    if (uniqIds.length <= CHUNK_SIZE) {
      return postJSON<DeniedActionsResponse>(
        '/admin/instances/denied-actions',
        { ids: uniqIds } as unknown as Record<string, unknown>,
      );
    }
    const chunks: number[][] = [];
    for (let i = 0; i < uniqIds.length; i += CHUNK_SIZE) {
      chunks.push(uniqIds.slice(i, i + CHUNK_SIZE));
    }
    const results = await Promise.all(
      chunks.map((chunk) =>
        postJSON<DeniedActionsResponse>(
          '/admin/instances/denied-actions',
          { ids: chunk } as unknown as Record<string, unknown>,
        ),
      ),
    );
    const merged: DeniedActionsResponse['instances'] = [];
    for (const r of results) {
      if (r?.instances?.length) merged.push(...r.instances);
    }
    return { instances: merged };
  },

  /**
   * 管理员批量升级实例镜像
   * POST /admin/instances/batch-upgrade (application/json)
   * 最多同时升级 20 个实例，仅运行中的实例支持升级
   */
  batchUpgrade: (ids: number[]) =>
    postJSON<BatchUpgradeResponse>('/admin/instances/batch-upgrade', { ids } as unknown as Record<string, unknown>),

  /**
   * 获取实例上已配置的 IM 通道列表
   * GET /admin/instances/channels?id=xxx
   */
  getInstanceChannels: (id: number) =>
    fetchJSON<InstanceChannelsResponse>(buildUrl('/admin/instances/channels', { id })),

  /**
   * 获取实例上的技能列表
   * GET /admin/instances/skills?id=xxx
   */
  getInstanceSkills: (id: number) =>
    fetchJSON<InstanceSkillsResponse>(buildUrl('/admin/instances/skills', { id })),

  /**
   * 获取实例绑定的模型列表（主/备模型）
   * GET /admin/instances/models?id=xxx
   */
  getInstanceModels: (id: number) =>
    fetchJSON<InstanceModelsResponse>(buildUrl('/admin/instances/models', { id })),

  /**
   * 获取实例可配置的模型列表（已启用且对该实例分组可见）
   * GET /admin/instances/available-models?id=xxx
   */
  getAvailableModels: (id: number) =>
    fetchJSON<AvailableModelsResponse>(buildUrl('/admin/instances/available-models', { id })),

  /**
   * 获取实例可配置的通道列表（已启用、对该实例分组可见、agent_type支持）
   * GET /admin/instances/available-channels?id=xxx
   */
  getAvailableChannels: (id: number) =>
    fetchJSON<AvailableChannelsResponse>(buildUrl('/admin/instances/available-channels', { id })),

  /**
   * 管控端为实例添加模型（首个自动 primary，后续 fallback）
   * POST /admin/instances/add-model
   */
  addInstanceModel: (params: { id: number; ai_model_id: number }) =>
    postForm<AdminAddModelResponse>('/admin/instances/add-model', params),

  /**
   * 管控端设置/替换实例主模型
   * POST /admin/instances/set-model
   */
  setInstanceModel: (params: { id: number; ai_model_id: number }) =>
    postForm<AdminSetModelResponse>('/admin/instances/set-model', params),

  /**
   * 管控端切换主备模型
   * POST /admin/instances/switch-primary-model
   */
  switchInstancePrimaryModel: (params: { id: number; instance_model_id: number }) =>
    postForm<AdminSwitchPrimaryModelResponse>('/admin/instances/switch-primary-model', params),

  /**
   * 管控端删除模型绑定
   * POST /admin/instances/del-model
   */
  delInstanceModel: (params: { id: number; instance_model_id: number }) =>
    postForm<AdminDelModelResponse>('/admin/instances/del-model', params),

  /**
   * 管控端设置/编辑通道配置
   * POST /admin/instances/set-channel
   */
  setInstanceChannel: (params: { id: number; channel: string; key: string[]; value: string[] }) =>
    postFormRepeated<AdminSetChannelResponse>(
      '/admin/instances/set-channel',
      [
        ['id', String(params.id)],
        ['channel', params.channel],
        ...params.key.map((k) => ['key', k] as [string, string]),
        ...params.value.map((v) => ['value', v] as [string, string]),
      ]
    ),

  /**
   * 管控端删除已配置通道
   * POST /admin/instances/del-channel
   */
  delInstanceChannel: (params: { id: number; channel: string }) =>
    postForm<AdminDelChannelResponse>('/admin/instances/del-channel', params),

  // ==================== 审计日志 ====================

  /**
   * 获取审计日志（分页，支持按用户名筛选）
   * GET /admin/audit
   */
  getAuditLogs: (params?: AuditLogsParams) =>
    fetchJSON<AuditLogsResponse>(buildUrl('/admin/audit', params as Record<string, string | number | undefined>)),

  // ==================== 配额管理 ====================

  /**
   * 获取用户配额列表
   * GET /admin/quotas
   */
  getQuotas: () =>
    fetchJSON<QuotasResponse>('/admin/quotas'),

  /**
   * 设置用户每日 Token 配额
   * POST /admin/quotas/set (x-www-form-urlencoded)
   * token_quota_day=-1 表示不限
   */
  setQuota: (params: SetQuotaParams) =>
    postForm<HatcheryOkResponse>('/admin/quotas/set', {
      user_id: params.user_id,
      token_quota_day: params.token_quota_day,
    }),

  // ==================== 用量统计 ====================

  /**
   * 统一用量查询（支持灵活的聚合维度和筛选）
   * GET /admin/usage/data
   */
  getUsageData: (params?: UsageDataParams) =>
    fetchJSON<UsageDataResponse>(buildUrl('/admin/usage/data', params as Record<string, string | number | undefined>)),

  /**
   * 分页查询 LLM 使用明细记录
   * GET /admin/usage/logs
   */
  getUsageLogs: (params?: UsageLogsParams) =>
    fetchJSON<UsageLogsResponse>(buildUrl('/admin/usage/logs', params as Record<string, string | number | undefined>)),

  // ==================== 企业技能库 - 分类管理 ====================

  /**
   * 查询技能分类列表（分页）
   * GET /admin/skill-categories
   */
  getSkillCategories: (params?: { page?: number; page_size?: number }) =>
    fetchJSON<SkillCategoriesResponse>(buildUrl('/admin/skill-categories', params as Record<string, string | number | undefined>)),

  /**
   * 创建技能分类
   * POST /admin/skill-categories/create (x-www-form-urlencoded)
   */
  createSkillCategory: (name: string, description?: string) =>
    postForm<{ ok: true; id: number }>('/admin/skill-categories/create', {
      name,
      ...(description ? { description } : {}),
    }),

  /**
   * 更新技能分类
   * POST /admin/skill-categories/update (x-www-form-urlencoded)
   */
  updateSkillCategory: (id: number, name?: string, description?: string) =>
    postForm<HatcheryOkResponse>('/admin/skill-categories/update', {
      id,
      ...(name !== undefined ? { name } : {}),
      ...(description !== undefined ? { description } : {}),
    }),

  /**
   * 删除技能分类（有技能关联时禁止删除）
   * POST /admin/skill-categories/delete (x-www-form-urlencoded)
   */
  deleteSkillCategory: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/skill-categories/delete', { id }),

  // ==================== 企业技能库 - 技能 CRUD ====================

  /**
   * 查询技能列表（每个 slug 只返回最新版本）
   * GET /admin/skills
   */
  getSkills: (params?: AdminSkillsParams) =>
    fetchJSON<AdminSkillsResponse>(buildUrl('/admin/skills', params as Record<string, string | number | undefined>)),

  /**
   * 创建技能（上传 zip 压缩包）
   * POST /admin/skills/create (multipart/form-data)
   * 支持上传进度回调 onUploadProgress
   */
  createSkill: (
    formData: FormData,
    onUploadProgress?: (percent: number) => void
  ) =>
    axiosInstance.post<AdminSkillCreateResponse>('/admin/skills/create', formData, {
      onUploadProgress: onUploadProgress
        ? (e) => {
          const percent = e.total ? Math.round((e.loaded * 100) / e.total) : 0;
          onUploadProgress(percent);
        }
        : undefined,
    }).then((res) => res.data),

  /**
   * 更新技能元信息（名称、描述、分类），不涉及文件变更
   * POST /admin/skills/update (x-www-form-urlencoded)
   */
  updateSkill: (slug: string, version: string, params: { name?: string; description?: string; category_ids?: string }) =>
    postForm<HatcheryOkResponse>('/admin/skills/update', {
      slug,
      version,
      ...params,
    }),

  /**
   * 删除技能（不传version删除所有版本，可选cascade级联删除引用）
   * POST /admin/skills/delete (x-www-form-urlencoded)
   */
  deleteSkill: (slug: string, _version?: string, cascade?: boolean) =>
    postForm<HatcheryOkResponse>('/admin/skills/delete', {
      slug,
      ...(cascade ? { cascade: 'true' } : {}),
    }),

  /**
   * 查询技能引用关系（用于删除前影响评估，不传version查询所有版本）
   * GET /admin/skills/references?slug=xxx
   */
  getSkillReferences: (slug: string, _version?: string) =>
    fetchJSON<SkillReferencesResponse>(
      buildUrl('/admin/skills/references', { slug })
    ),

  /**
   * 查询技能详情（默认最新版本，可指定历史版本）
   * GET /admin/skills/detail?slug=xxx&version=xxx
   */
  getSkillDetail: (slug: string, version?: string) =>
    fetchJSON<AdminSkillDetailResponse>(
      buildUrl('/admin/skills/detail', { slug, ...(version ? { version } : {}) })
    ),

  /**
   * 查询技能文件列表（所有版本的文件目录结构）
   * GET /admin/skills/files?slug=xxx
   */
  getSkillFiles: (slug: string) =>
    fetchJSON<AdminSkillFilesResponse>(buildUrl('/admin/skills/files', { slug })),

  // ==================== 企业技能库 - 安全检测 ====================

  /**
   * 查询安全扫描默认配置
   * GET /admin/skills/scan-config
   */
  getSkillScanConfig: () =>
    fetchJSON<{ skill_scan_default_enabled: boolean }>('/admin/skills/scan-config'),

  /**
   * 设置安全扫描默认配置
   * POST /admin/skills/scan-config
   */
  setSkillScanConfig: (enabled: boolean) =>
    postJSON<{ ok: boolean; skill_scan_default_enabled: boolean }>('/admin/skills/scan-config', {
      skill_scan_default_enabled: enabled,
    }),

  /**
   * 手动触发安全检测
   * POST /admin/skills/scan-trigger
   */
  triggerSkillScan: (skillId: number) =>
    postJSON<{ ok: boolean; scan_id: number; status: string; message: string }>('/admin/skills/scan-trigger', {
      skill_id: skillId,
    }),

  // ==================== 企业技能库 - 下发与安装 ====================

  /**
   * 查询下发任务列表（按时间倒序）
   * GET /admin/skills/tasks?slug=xxx
   */
  getSkillTasks: (params: AdminSkillTasksParams) =>
    fetchJSON<AdminSkillTasksResponse>(buildUrl('/admin/skills/tasks', params as unknown as Record<string, string | number | undefined>)),

  /**
   * 查询实例安装情况（支持按状态筛选）
   * GET /admin/skills/instances?slug=xxx
   */
  getSkillInstances: (params: AdminSkillInstancesParams) =>
    fetchJSON<AdminSkillInstancesResponse>(buildUrl('/admin/skills/instances', params as unknown as Record<string, string | number | undefined>)),

  /**
   * 单独更新技能应用范围
   * POST /admin/skills/update (x-www-form-urlencoded)
   */
  updateSkillVisibility: (slug: string, version: string, visibilityType: 'all' | 'group', groupIds?: string) =>
    postForm<HatcheryOkResponse>('/admin/skills/update', {
      slug,
      version,
      visibility_type: visibilityType,
      ...(groupIds ? { group_ids: groupIds } : {}),
    }),

  /**
   * 批量下发技能到实例（异步，返回 task_id）
   * POST /admin/skills/distribute (application/json)
   */
  distributeSkill: (params: AdminSkillDistributeParams) =>
    postJSON<AdminSkillDistributeResponse>('/admin/skills/distribute', params as unknown as Record<string, unknown>),

  /**
   * 批量卸载技能（从实例上移除已安装的技能）
   * POST /admin/skills/uninstall (application/json)
   */
  uninstallSkill: (params: AdminSkillUninstallParams) =>
    postJSON<AdminSkillUninstallResponse>('/admin/skills/uninstall', params as unknown as Record<string, unknown>),

  /**
   * 查询 SMH 存储配置信息和 AccessToken
   * GET /admin/smhinfo
   */
  getSmhInfo: () =>
    fetchJSON<SmhInfoResponse>('/admin/smhinfo'),

  /**
   * 开启 SMH 个人空间功能
   * POST /admin/smh/enable-personal-space
   */
  enableSmhPersonalSpace: () =>
    postJSON<HatcheryOkResponse>('/admin/smh/enable-personal-space', {}),

  /**
   * 查询 SMH 存储的整体统计信息
   * GET /admin/smh/stat
   */
  getSmhStat: () =>
    fetchJSON<SmhStatResponse>('/admin/smh/stat'),

  /**
   * 查询实例个人空间列表（分页）
   * GET /admin/smh/personal-spaces
   */
  getSmhPersonalSpaces: (params?: SmhPersonalSpacesParams) =>
    fetchJSON<SmhPersonalSpacesResponse>(buildUrl('/admin/smh/personal-spaces', params as unknown as Record<string, string | number | undefined>)),

  /**
   * 设置创建实例时是否自动创建个人空间
   * POST /admin/smh/personal-space-auto-provision
   */
  setSmhAutoProvision: (enabled: boolean) =>
    postForm<HatcheryOkResponse>('/admin/smh/personal-space-auto-provision', {
      enabled: enabled ? '1' : '0',
    }),

  /**
   * 查询实例及个人空间绑定状态列表（分页）
   * GET /admin/smh/instances
   */
  getSmhInstances: (params?: SmhInstancesParams) =>
    fetchJSON<SmhInstancesResponse>(buildUrl('/admin/smh/instances', params as unknown as Record<string, string | number | undefined>)),

  /**
   * 批量开启或关闭实例的个人空间
   * POST /admin/smh/instance-space (application/json)
   */
  setSmhInstanceSpace: (params: SmhInstanceSpaceParams) =>
    postJSON<SmhInstanceSpaceResponse>('/admin/smh/instance-space', params as unknown as Record<string, unknown>),
  // ==================== 角色管理 ====================

  /**
   * 查询角色列表（含每个角色的技能列表）
   * GET /admin/roles
   */
  getRoles: (params?: import('@/types/api').AdminRolesParams) =>
    fetchJSON<AdminRolesResponse>(buildUrl('/admin/roles', params as Record<string, string | number | undefined>)),

  /**
   * 新增角色
   * POST /admin/roles/create (application/json)
   */
  createRole: (params: CreateRoleParams) =>
    postJSON<CreateRoleResponse>('/admin/roles/create', params as unknown as Record<string, unknown>),

  /**
   * 编辑角色（技能全量替换）
   * POST /admin/roles/update?id=xxx (application/json)
   */
  updateRole: (id: number, params: CreateRoleParams) =>
    postJSON<HatcheryOkResponse>(buildUrl('/admin/roles/update', { id }), params as unknown as Record<string, unknown>),

  /**
   * 删除角色（硬删除，级联删除关联技能）
   * POST /admin/roles/delete?id=xxx
   */
  deleteRole: (id: number) =>
    postForm<HatcheryOkResponse>(buildUrl('/admin/roles/delete', { id }), {}),

  /**
   * 切换角色可见性（取反）
   * POST /admin/roles/toggle-visible?id=xxx
   */
  toggleRoleVisible: (id: number) =>
    postForm<ToggleRoleVisibleResponse>(buildUrl('/admin/roles/toggle-visible', { id }), {}),

  /**
   * 批量更新角色排序
   * POST /admin/roles/reorder (application/json)
   * 请求体: { ids: number[] }
   */
  reorderRoles: (ids: number[]) =>
    postJSON<HatcheryOkResponse>('/admin/roles/reorder', { ids } as unknown as Record<string, unknown>),

  /**
   * 获取管控端通知栏数据（配置步骤完成状态 + 配额告警 + 产品动态）
   * GET /admin/notices?limit=&offset=
   * limit/offset 仅作用于 product_news，不传则返回全部
   */
  getNotices: (params?: { limit?: number; offset?: number }) => {
    const query = new URLSearchParams();
    if (params?.limit != null) query.set('limit', String(params.limit));
    if (params?.offset != null) query.set('offset', String(params.offset));
    const qs = query.toString();
    return fetchJSON<AdminNoticesResponse>(qs ? `/admin/notices?${qs}` : '/admin/notices');
  },

  // ==================== 企业插件库 - 分类管理 ====================

  /**
   * 查询插件分类列表（分页）
   * GET /admin/plugin-categories
   */
  getPluginCategories: (params?: { page?: number; page_size?: number }) =>
    fetchJSON<PluginCategoriesResponse>(buildUrl('/admin/plugin-categories', params as Record<string, string | number | undefined>)),

  /**
   * 创建插件分类
   * POST /admin/plugin-categories/create (x-www-form-urlencoded)
   */
  createPluginCategory: (name: string, description?: string) =>
    postForm<{ ok: true; id: number }>('/admin/plugin-categories/create', {
      name,
      ...(description ? { description } : {}),
    }),

  /**
   * 更新插件分类
   * POST /admin/plugin-categories/update (x-www-form-urlencoded)
   */
  updatePluginCategory: (id: number, name?: string, description?: string) =>
    postForm<HatcheryOkResponse>('/admin/plugin-categories/update', {
      id,
      ...(name !== undefined ? { name } : {}),
      ...(description !== undefined ? { description } : {}),
    }),

  /**
   * 删除插件分类（自动清理关联关系，不删除插件）
   * POST /admin/plugin-categories/delete (x-www-form-urlencoded)
   */
  deletePluginCategory: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/plugin-categories/delete', { id }),

  // ==================== 企业插件库 - 插件 CRUD ====================

  /**
   * 查询插件列表（每个 slug 只返回最新版本）
   * GET /admin/plugins
   */
  getPlugins: (params?: AdminPluginsParams) =>
    fetchJSON<AdminPluginsResponse>(buildUrl('/admin/plugins', params as Record<string, string | number | undefined>)),

  /**
   * 上传插件（multipart/form-data）
   * POST /admin/plugins/create
   * 支持上传进度回调 onUploadProgress
   */
  createPlugin: (
    formData: FormData,
    onUploadProgress?: (percent: number) => void
  ) =>
    axiosInstance.post<AdminPluginCreateResponse>('/admin/plugins/create', formData, {
      onUploadProgress: onUploadProgress
        ? (e) => {
          const percent = e.total ? Math.round((e.loaded * 100) / e.total) : 0;
          onUploadProgress(percent);
        }
        : undefined,
    }).then((res) => res.data),

  /**
   * 查询插件详情（默认最新版本，可指定历史版本）
   * GET /admin/plugins/detail?slug=xxx&version=xxx
   */
  getPluginDetail: (slug: string, version?: string) =>
    fetchJSON<AdminPluginDetailResponse>(
      buildUrl('/admin/plugins/detail', { slug, ...(version ? { version } : {}) })
    ),

  /**
   * 查询插件文件列表（所有版本的文件目录结构）
   * GET /admin/plugins/files?slug=xxx
   */
  getPluginFiles: (slug: string) =>
    fetchJSON<AdminPluginFilesResponse>(buildUrl('/admin/plugins/files', { slug })),

  /**
   * 更新插件版本（创建基于当前最新版本的新版本记录）
   * POST /admin/plugins/update (multipart/form-data)
   *
   * 必填：slug、version；
   * 可选：name、description、changelog、file、npm_package、category_ids、visibility_type、group_ids；
   * 未传的元信息字段自动从当前版本继承。
   */
  updatePluginVersion: (
    formData: FormData,
    onUploadProgress?: (percent: number) => void,
  ) =>
    axiosInstance.post<AdminPluginUpdateResponse>('/admin/plugins/update', formData, {
      onUploadProgress: onUploadProgress
        ? (e) => {
          const percent = e.total ? Math.round((e.loaded * 100) / e.total) : 0;
          onUploadProgress(percent);
        }
        : undefined,
    }).then((res) => res.data),

  /**
   * 仅更新插件元信息（名称/描述/分类），不创建新版本、不变更文件
   * POST /admin/plugins/update (x-www-form-urlencoded)
   *
   * 注意：后端 update 接口要求 version 必须高于当前最新版本，因此该方法仅在
   * 同时传新版本号时使用。如需保持版本号不变更新元信息，请使用 updatePluginVersion 走 FormData
   * 并传 file=空 同样会走继承逻辑。本方法保留作为旧调用兼容（同 slug、同 version 更新元信息已不可用）。
   */
  updatePlugin: (slug: string, version: string, params: { name?: string; description?: string; category_ids?: string }) =>
    postForm<HatcheryOkResponse>('/admin/plugins/update', {
      slug,
      version,
      ...params,
    }),

  /**
   * 更新插件应用范围（基于当前最新版本创建新版本记录）
   * POST /admin/plugins/update (multipart/form-data)
   * 注意：当前后端不支持仅修改应用范围而不升版本，调用方需自行判断/递增版本号。
   */
  updatePluginVisibility: (
    slug: string,
    version: string,
    visibilityType: 'all' | 'group',
    groupIds?: string,
  ) => {
    const formData = new FormData();
    formData.append('slug', slug);
    formData.append('version', version);
    formData.append('visibility_type', visibilityType);
    if (groupIds) formData.append('group_ids', groupIds);
    return axiosInstance
      .post<AdminPluginUpdateResponse>('/admin/plugins/update', formData)
      .then((res) => res.data);
  },

  /**
   * 删除指定插件版本（有进行中下发任务时禁止删除）
   * POST /admin/plugins/delete (x-www-form-urlencoded)
   */
  deletePlugin: (slug: string, version: string) =>
    postForm<HatcheryOkResponse>('/admin/plugins/delete', { slug, version }),

  // ==================== 企业插件库 - 下发与安装 ====================

  /**
   * 查询实例安装状态
   * GET /admin/plugins/instances?slug=xxx
   */
  getPluginInstances: (params: AdminPluginInstancesParams) =>
    fetchJSON<AdminPluginInstancesResponse>(buildUrl('/admin/plugins/instances', params as unknown as Record<string, string | number | undefined>)),

  /**
   * 批量下发插件到实例（异步，返回 task_id）
   * POST /admin/plugins/distribute (application/json)
   */
  distributePlugin: (params: AdminPluginDistributeParams) =>
    postJSON<AdminPluginDistributeResponse>('/admin/plugins/distribute', params as unknown as Record<string, unknown>),

  /**
   * 批量从实例中卸载插件（异步，返回 task_id）
   * POST /admin/plugins/uninstall (application/json)
   * 与下发操作互斥（同一插件同时只能有一个下发或卸载任务）
   */
  uninstallPlugin: (params: AdminPluginUninstallParams) =>
    postJSON<AdminPluginUninstallResponse>('/admin/plugins/uninstall', params as unknown as Record<string, unknown>),

  /**
   * 查询插件下发/卸载任务列表（按时间倒序）
   * GET /admin/plugins/tasks?slug=xxx&type=distribute|uninstall
   */
  getPluginTasks: (params: AdminPluginTasksParams) =>
    fetchJSON<AdminPluginTasksResponse>(buildUrl('/admin/plugins/tasks', params as unknown as Record<string, string | number | undefined>)),

  // ==================== 企业插件库 - 收藏管理 ====================

  /**
   * 收藏插件
   * POST /admin/plugins/favorite (x-www-form-urlencoded)
   */
  favoritePlugin: (params: {
    name: string;
    slug: string;
    plugin_id: string;
    version: string;
    description?: string;
    npm_package?: string;
  }) =>
    postForm<{ ok: boolean; id: number }>('/admin/plugins/favorite', params as Record<string, string | number | undefined>),

  /**
   * 取消收藏插件
   * POST /admin/plugins/unfavorite (x-www-form-urlencoded)
   */
  unfavoritePlugin: (id: number) =>
    postForm<HatcheryOkResponse>('/admin/plugins/unfavorite', { id }),

  /**
   * 查询已收藏插件列表
   * GET /admin/plugins/favorited
   */
  getFavoritedPlugins: (params?: { page?: number; page_size?: number }) =>
    fetchJSON<FavoritedPluginsResponse>(buildUrl('/admin/plugins/favorited', params as Record<string, string | number | undefined>)),

  // ==================== Tag 标签管理 ====================

  /** 查询腾讯云标签键列表 GET /api/tags/keys */
  getTagKeys: () =>
    fetchJSON<{ keys: string[] }>('/api/tags/keys'),

  /** 查询指定标签键的值列表 GET /api/tags/values?key=xxx */
  getTagValues: (key: string) =>
    fetchJSON<{ key: string; values: string[] }>(buildUrl('/api/tags/values', { key })),

  /** 获取默认标签配置（从 GET /admin/config 的 default_tags 字段读取） */
  getDefaultTags: async () => {
    const res = await fetchJSON<SiteConfigResponse>('/admin/config');
    const raw = res.config?.default_tags || '[]';
    let tags: Array<{ Key: string; Value: string }> = [];
    try {
      tags = typeof raw === 'string' ? JSON.parse(raw) : raw;
    } catch (e) {
      console.error('[Tag] default_tags JSON 解析失败:', e);
    }
    return { tags };
  },

  /** 更新默认标签配置（通过 POST /admin/config 的 default_tags 参数） */
  updateDefaultTags: (tags: Array<{ Key: string; Value: string }>) => {
    const formData = new FormData();
    formData.append('default_tags', JSON.stringify(tags));
    return postFormData<HatcheryOkResponse>('/admin/config', formData);
  },

  /** 查询本地管理的默认标签列表 GET /admin/tags */
  getAdminTags: () =>
    fetchJSON<{ tags: Array<{ id: number; key: string; value: string; visibility_type: string; group_ids: number[]; created_at: string; updated_at: string }> }>('/admin/tags'),

  /** 全量覆盖本地默认标签列表 POST /admin/tags/replace-all */
  replaceAllTags: (tags: Array<{ key: string; value: string; visibility_type: string; group_ids: number[] }>) =>
    postJSON<HatcheryOkResponse>('/admin/tags/replace-all', { tags }),

  // ==================== Memory Pro ====================

  /**
   * 获取 Memory Pro 服务概览（实例统计、Pro 容量、默认计划等）
   * GET /admin/memory/overview
   */
  getMemoryOverview: () =>
    fetchJSON<MemoryOverviewResponse>('/admin/memory/overview'),

  /**
   * 开通 Pro 服务（创建 VDB 实例）
   * POST /admin/memory/pro/activate
   */
  activateMemoryPro: (params: MemoryProActivateParams) =>
    postJSON<MemoryProActivateResponse>('/admin/memory/pro/activate', params as unknown as Record<string, unknown>),

  /**
   * 关闭 Pro 服务（前置条件：所有实例 Pro 记忆库已关闭）
   * POST /admin/memory/pro/release
   */
  releaseMemoryPro: () =>
    postJSON<MemoryProReleaseResponse>('/admin/memory/pro/release', {}),

  /**
   * 批量切换记忆计划（异步，返回 task_id）
   * POST /admin/memory/plan/switch
   */
  switchMemoryPlan: (params: MemoryPlanSwitchParams) =>
    postJSON<MemoryPlanSwitchResponse>('/admin/memory/plan/switch', params as unknown as Record<string, unknown>),

  /**
   * 获取实例列表（含记忆计划状态，支持分页、搜索和计划过滤）
   * GET /admin/memory/instances
   */
  getMemoryInstances: (params?: MemoryInstancesParams) =>
    fetchJSON<MemoryInstancesResponse>(buildUrl('/admin/memory/instances', params as Record<string, string | number | undefined>)),

  /**
   * 查询待升级的记忆插件实例列表
   * GET /admin/memory/plugin-upgrade/candidates
   */
  getPluginUpgradeCandidates: () =>
    fetchJSON<PluginUpgradeCandidatesResponse>('/admin/memory/plugin-upgrade/candidates'),

  /**
   * 触发批量升级记忆插件
   * POST /admin/memory/plugin-upgrade/execute
   */
  executePluginUpgrade: (params: { instance_ids: string[] }) =>
    postJSON<PluginUpgradeExecuteResponse>('/admin/memory/plugin-upgrade/execute', params as unknown as Record<string, unknown>),

  /**
   * 查询增量实例默认记忆计划
   * GET /admin/memory/default-plan
   */
  getMemoryDefaultPlan: () =>
    fetchJSON<MemoryDefaultPlanResponse>('/admin/memory/default-plan'),

  /**
   * 更新增量实例默认记忆计划
   * PUT /admin/memory/default-plan
   */
  updateMemoryDefaultPlan: (params: MemoryDefaultPlanUpdateParams) =>
    putJSON<MemoryDefaultPlanUpdateResponse>('/admin/memory/default-plan', params as unknown as Record<string, unknown>),

  // ==================== 旧版记忆接口（兼容） ====================

  /**
   * 查询全局记忆开关配置（旧版）
   * GET /admin/memory-tdai/config
   */
  getMemoryTdaiConfig: () =>
    fetchJSON<MemoryTdaiConfigResponse>('/admin/memory-tdai/config'),

  /**
   * 更新全局记忆开关配置（旧版）
   * PUT /admin/memory-tdai/config
   */
  updateMemoryTdaiConfig: (enabled: boolean) =>
    putJSON<MemoryTdaiConfigUpdateResponse>('/admin/memory-tdai/config', { memory_tdai_enable: enabled }),

  // ==================== Memory 分组策略 ====================

  /**
   * 查询分组策略列表
   * GET /admin/memory/group-policies
   */
  getMemoryGroupPolicies: () =>
    fetchJSON<MemoryGroupPoliciesResponse>('/admin/memory/group-policies'),

  /**
   * 创建分组策略
   * POST /admin/memory/group-policy
   */
  createMemoryGroupPolicy: (params: MemoryGroupPolicyCreateParams) =>
    postJSON<HatcheryOkResponse>('/admin/memory/group-policy', params as unknown as Record<string, unknown>),

  /**
   * 修改分组策略（全量替换）
   * PUT /admin/memory/group-policy
   */
  updateMemoryGroupPolicy: (params: MemoryGroupPolicyUpdateParams) =>
    putJSON<HatcheryOkResponse>('/admin/memory/group-policy', params as unknown as Record<string, unknown>),

  /**
   * 删除分组策略
   * POST /admin/memory/group-policy/delete
   */
  deleteMemoryGroupPolicy: (params: MemoryGroupPolicyDeleteParams) =>
    postJSON<HatcheryOkResponse>('/admin/memory/group-policy/delete', params as unknown as Record<string, unknown>),

  // ==================== 企业 MCP 库 ====================

  /**
   * 获取 MCP 列表
   * GET /admin/mcp
   */
  getMCPs: (params?: AdminMCPsParams) =>
    fetchJSON<AdminMCPsResponse>(buildUrl('/admin/mcp', params as Record<string, string | number | undefined>)),

  /**
   * 新增 MCP
   * POST /admin/mcp/create
   */
  createMCP: (params: AdminMCPCreateParams) =>
    postJSON<AdminMCPCreateResponse>('/admin/mcp/create', params as unknown as Record<string, unknown>),

  /**
   * 新增 MCP 版本
   * POST /admin/mcp/update
   */
  updateMCPVersion: (params: AdminMCPUpdateParams) =>
    postJSON<AdminMCPUpdateResponse>('/admin/mcp/update', params as unknown as Record<string, unknown>),

  /**
   * 修改 MCP 元数据
   * POST /admin/mcp/meta
   */
  updateMCPMeta: (params: AdminMCPMetaParams) =>
    postJSON<AdminMCPMetaResponse>('/admin/mcp/meta', params as unknown as Record<string, unknown>),

  /**
   * 获取 MCP 详情
   * GET /admin/mcp/detail
   */
  getMCPDetail: (serviceId: string, version?: string) =>
    fetchJSON<AdminMCPDetailResponse>(buildUrl('/admin/mcp/detail', {
      service_id: serviceId,
      ...(version ? { version } : {}),
    })),

  /**
   * 删除 MCP
   * POST /admin/mcp/delete
   */
  deleteMCP: (serviceId: string) =>
    postJSON<AdminMCPDeleteResponse>('/admin/mcp/delete', { service_id: serviceId }),

  /**
   * 获取 MCP 版本列表
   * GET /admin/mcp/versions
   */
  getMCPVersions: (serviceId: string) =>
    fetchJSON<AdminMCPVersionsResponse>(buildUrl('/admin/mcp/versions', { service_id: serviceId })),

  /**
   * 批量下发 MCP 到实例
   * POST /admin/mcp/distribute
   */
  distributeMCP: (params: AdminMCPDistributeParams) =>
    postJSON<AdminMCPDistributeResponse>('/admin/mcp/distribute', params as unknown as Record<string, unknown>),

  /**
   * 获取 MCP 下发任务列表
   * GET /admin/mcp/tasks
   */
  getMCPTasks: (params: AdminMCPTasksParams) =>
    fetchJSON<AdminMCPTasksResponse>(buildUrl('/admin/mcp/tasks', params as Record<string, string | number | undefined>)),

  /**
   * 获取 MCP 实例安装情况
   * GET /admin/mcp/instances
   */
  getMCPInstances: (params: AdminMCPInstancesParams) =>
    fetchJSON<AdminMCPInstancesResponse>(buildUrl('/admin/mcp/instances', params as Record<string, string | number | undefined>)),

  // ==================== Admin Agent Commands（命令下发） ====================

  /**
   * 查询命令模板列表（支持搜索 + 分页）
   * GET /admin/agent-commands
   *
   * 字段兼容：后端可能用 items / list / data / commands 任一作为列表字段名，
   * 此处统一归一为 items，便于上层组件无感知使用。
   */
  getAgentCommands: async (params?: AgentCommandsListParams): Promise<AgentCommandsListResponse> => {
    const raw = await fetchJSON<AgentCommandsListResponse & {
      list?: AgentCommandItem[];
      data?: AgentCommandItem[];
      commands?: AgentCommandItem[];
    }>(
      buildUrl('/admin/agent-commands', params as Record<string, string | number | undefined>),
    );
    const items = raw.items ?? raw.list ?? raw.data ?? raw.commands ?? [];
    if (!raw.items && (raw.total ?? 0) > 0 && items.length === 0) {
      // 兜底：后端返回了 total > 0，但常见字段名都没匹配上，打印原始响应便于排查
      // eslint-disable-next-line no-console
      console.warn('[getAgentCommands] 未匹配到列表字段，原始响应：', raw);
    }
    return {
      items,
      total: raw.total ?? items.length,
      page: raw.page ?? 1,
      page_size: raw.page_size ?? items.length,
    };
  },

  /**
   * 创建命令模板
   * POST /admin/agent-commands/create
   */
  createAgentCommand: (params: AgentCommandCreateParams) =>
    postJSON<AgentCommandItem>(
      '/admin/agent-commands/create',
      params as unknown as Record<string, unknown>,
    ),

  /**
   * 编辑命令模板（仅创建者本人或 super_admin）
   * POST /admin/agent-commands/update
   */
  updateAgentCommand: (params: AgentCommandUpdateParams) =>
    postJSON<AgentCommandItem>(
      '/admin/agent-commands/update',
      params as unknown as Record<string, unknown>,
    ),

  /**
   * 软删命令模板（仅创建者本人或 super_admin）
   * POST /admin/agent-commands/delete
   */
  deleteAgentCommand: (params: AgentCommandDeleteParams) =>
    postJSON<HatcheryOkResponse>(
      '/admin/agent-commands/delete',
      params as unknown as Record<string, unknown>,
    ),

  /**
   * 下发命令到一批 Agent 实例（异步派发，立即返回 dispatch_slug）
   * POST /admin/agent-commands/dispatch
   */
  dispatchAgentCommand: (params: AgentCommandDispatchParams) =>
    postJSON<AgentCommandDispatchResponse>(
      '/admin/agent-commands/dispatch',
      params as unknown as Record<string, unknown>,
    ),

  /**
   * 续跑：测试机终态后，触发剩余批次的下发
   * POST /admin/agent-commands/dispatch  body: { dispatch_slug }
   */
  continueAgentCommandDispatch: (dispatch_slug: string) =>
    postJSON<AgentCommandContinueDispatchResponse>(
      '/admin/agent-commands/dispatch',
      { dispatch_slug } as unknown as Record<string, unknown>,
    ),

  /**
   * 终止：测试机终态后，把剩余 pending 批次置为 cancelled
   * POST /admin/agent-commands/dispatch  body: { dispatch_slug, abort: true }
   */
  abortAgentCommandDispatch: (dispatch_slug: string) =>
    postJSON<AgentCommandAbortDispatchResponse>(
      '/admin/agent-commands/dispatch',
      { dispatch_slug, abort: true } as unknown as Record<string, unknown>,
    ),

  /**
   * 查询执行任务列表（支持搜索 + 分页 + 状态过滤）
   * GET /admin/agent-commands/tasks
   *
   * 后端响应包装字段为 tasks（按 dispatch_slug 聚合），此处归一为 items 便于上层使用。
   * 兼容 list / data 等其它命名以应对联调期变化。
   */
  getAgentCommandTasks: async (params?: AgentCommandTasksListParams): Promise<AgentCommandTasksListResponse> => {
    const raw = await fetchJSON<AgentCommandTasksListResponse & {
      tasks?: AgentCommandTaskItem[];
      list?: AgentCommandTaskItem[];
      data?: AgentCommandTaskItem[];
    }>(
      buildUrl('/admin/agent-commands/tasks', params as Record<string, string | number | undefined>),
    );
    const items = raw.tasks ?? raw.items ?? raw.list ?? raw.data ?? [];
    if (!raw.tasks && !raw.items && (raw.total ?? 0) > 0 && items.length === 0) {
      // eslint-disable-next-line no-console
      console.warn('[getAgentCommandTasks] 未匹配到列表字段，原始响应：', raw);
    }
    return {
      items,
      total: raw.total ?? items.length,
      page: raw.page ?? 1,
      page_size: raw.page_size ?? items.length,
    };
  },

  /**
   * 查询执行任务详情（含命令快照、测试机汇总、每台 Agent 的 stdout/stderr）
   * GET /admin/agent-commands/tasks/detail?dispatch_slug=task-xxxxxxxx
   */
  getAgentCommandTaskDetail: (params: AgentCommandTaskDetailParams) =>
    fetchJSON<AgentCommandTaskDetailResponse>(
      buildUrl('/admin/agent-commands/tasks/detail', {
        dispatch_slug: params.dispatch_slug,
        with_output: params.with_output === undefined ? undefined : (params.with_output ? 'true' : 'false'),
      }),
    ),
};
