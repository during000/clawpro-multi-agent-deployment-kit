import { postJSON, fetchJSON } from "@/services/request";
import type { 
  RuleSetResponse, 
  CreateRuleSetParams, 
  CreateRuleSetResponse,
  UpdateRuleSetRulesParams, 
  UpdateRuleSetRulesResponse, 
  ImportRulesFromSgParams, 
  ImportRulesFromSgResponse 
} from "@/types/api";

/**
 * 查询当前配置的安全组详情
 * GET /admin/config/security-group
 * 响应透传腾讯云 DescribeSecurityGroups 接口
 */
export const getSecurityGroup = () =>
  fetchJSON<Record<string, unknown>>("/admin/config/security-group", { silentError: true });

/**
 * 创建新安全组并自动绑定到站点配置
 * POST /admin/config/security-group (application/json)
 * 请求体透传给腾讯云 CreateSecurityGroup 接口
 */
export const createSecurityGroup = (params: any) =>
  postJSON<Record<string, unknown>>(
    "/admin/config/security-group",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );

/**
 * 【已迁移至新架构】查询当前安全组的规则列表
 * 新接口：GET /admin/config/security-group/ruleset
 * 使用 getRuleSet() 获取规则集及其投影的云端安全组信息
 * 
 * @deprecated 请使用 getRuleSet() 获取规则集数据
 */
export const getSecurityGroupPolicies = () =>
  fetchJSON<RuleSetResponse>("/admin/config/security-group/ruleset", { silentError: true });

/**
 * 【已迁移至新架构】创建安全组规则
 * 新接口：POST /admin/config/security-group/ruleset/rules
 * 使用 updateRuleSetRules() 更新规则集中的规则
 * 
 * @deprecated 请使用 updateRuleSetRules() 更新规则
 */
export const createSecurityGroupPolicies = (params: Record<string, unknown>) =>
  postJSON<UpdateRuleSetRulesResponse>(
    "/admin/config/security-group/ruleset/rules",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );

/**
 * 【已迁移至新架构】替换安全组规则
 * 新接口：POST /admin/config/security-group/ruleset/rules
 * 使用 updateRuleSetRules() 更新规则集中的规则
 * 
 * @deprecated 请使用 updateRuleSetRules() 更新规则
 */
export const replaceSecurityGroupPolicies = (params: Record<string, unknown>) =>
  postJSON<UpdateRuleSetRulesResponse>(
    "/admin/config/security-group/ruleset/rules",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );

/**
 * 【已迁移至新架构】删除安全组规则
 * 新接口：POST /admin/config/security-group/ruleset/rules
 * 使用 updateRuleSetRules() 更新规则集中的规则
 * 
 * @deprecated 请使用 updateRuleSetRules() 更新规则
 */
export const deleteSecurityGroupPolicies = (params: Record<string, unknown>) =>
  postJSON<UpdateRuleSetRulesResponse>(
    "/admin/config/security-group/ruleset/rules",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );

// ==================== 新接口包装函数（RuleSet 架构）====================

/**
 * 获取规则集（规则真值源）
 * GET /admin/config/security-group/ruleset
 * 返回当前 RuleSet、版本号、投影到的云端安全组列表
 */
export const getRuleSet = () =>
  fetchJSON<RuleSetResponse>("/admin/config/security-group/ruleset", { silentError: true });

/**
 * 创建规则集和第一个 ACTIVE 云端安全组
 * POST /admin/config/security-group/rulesets (application/json)
 * 用于初始化企业的规则管理基础设施（第一次创建时调用）
 */
export const createRuleSet = (params: CreateRuleSetParams) =>
  postJSON<CreateRuleSetResponse>(
    "/admin/config/security-group/rulesets",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );

/**
 * 更新规则集中的规则
 * POST /admin/config/security-group/ruleset/rules (application/json)
 * 更新会自动增加版本号并投影（fan-out）到所有 ACTIVE 云端安全组
 */
export const updateRuleSetRules = (params: UpdateRuleSetRulesParams) =>
  postJSON<UpdateRuleSetRulesResponse>(
    "/admin/config/security-group/ruleset/rules",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );

/**
 * 从外部安全组导入规则到规则集
 * POST /admin/config/security-group/ruleset/import-from-sg (application/json)
 * 用于从腾讯云安全组导入现有规则到规则集
 */
export const importRulesFromSg = (params: ImportRulesFromSgParams) =>
  postJSON<ImportRulesFromSgResponse>(
    "/admin/config/security-group/ruleset/import-from-sg",
    params as unknown as Record<string, unknown>,
    { silentError: true }
  );
