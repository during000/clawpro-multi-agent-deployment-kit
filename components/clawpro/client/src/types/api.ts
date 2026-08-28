/**
 * @/types/api - 通用 API 类型声明
 * 为 Security 模块提供类型支持
 */

export interface RuleSetResponse {
  RuleSetId?: string;
  RuleSetName?: string;
  Rules?: any[];
  [key: string]: any;
}

export interface CreateRuleSetParams {
  RuleSetName: string;
  [key: string]: any;
}

export interface CreateRuleSetResponse {
  RuleSetId?: string;
  [key: string]: any;
}

export interface UpdateRuleSetRulesParams {
  RuleSetId: string;
  Rules: any[];
  [key: string]: any;
}

export interface UpdateRuleSetRulesResponse {
  [key: string]: any;
}

export interface ImportRulesFromSgParams {
  SecurityGroupId: string;
  [key: string]: any;
}

export interface ImportRulesFromSgResponse {
  [key: string]: any;
}

// ─── PlatformPolicy 依赖的类型（从 openclaw-enterprise-fronted 同步） ─────────

export interface ModelVisibilityGroup {
  group_id: number;
  group_name?: string;
}

export interface ModelInfo {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  Provider: string;
  ModelID: string;
  ModelName: string;
  URL: string;
  ModelType: string;
  ContextLen: number;
  QuotaDay: number;
  Enabled: boolean;
  InputTypes?: string[] | null;
  visibility_type?: 'all' | 'group';
  visibility_groups?: ModelVisibilityGroup[];
  VisibilityType?: 'all' | 'group';
  VisibilityGroups?: ModelVisibilityGroup[];
}

export type GroupPolicyConfigKey =
  | 'token_quota_day'
  | 'instance_quota'
  | 'global_token_quota_day'
  | 'token_quota_rules'
  | 'global_token_quota_rules'
  | 'user_config_model'
  | 'user_config_channel'
  | 'custom_model'
  | 'agent_terminal'
  | 'gateway_ui'
  | 'chat_view'
  | 'browser_vnc'
  | 'lobster_doctor'
  | 'model_quota'
  | 'smh_auto_provision'
  | 'smh_self_enable';

export interface SetGroupPolicyParams {
  group_id: number;
  config_key: GroupPolicyConfigKey;
  value_json: string;
}

export interface DeleteGroupPolicyParams {
  group_id: number;
  config_key: GroupPolicyConfigKey;
}

export interface GroupConfigQuery {
  config_type: 'policy' | 'channel' | 'plugin_bundle' | 'mcp' | 'image_type';
  config_key: string;
}

export interface GroupConfigResult {
  config_key: string;
  groups: Array<{
    group_id: number;
    value?: Record<string, unknown>;
  }>;
}

export interface QueryGroupConfigsResponse {
  results: GroupConfigResult[];
}

export interface UserGroupTreeNode {
  id: number;
  name: string;
  parent_id?: number;
  full_path?: string;
  source?: string;
  to_be_deleted?: boolean;
  children?: UserGroupTreeNode[];
}

export interface UserGroupsTreeResponse {
  org_tree: UserGroupTreeNode[];
  user_groups: UserGroupTreeNode[];
}
