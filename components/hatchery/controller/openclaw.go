package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// selectSGForNewInstanceFn 指向 SelectSGForNewInstance，专供测试 stub。
// 生产运行时不要修改。签名：(ctx, identifier, ruleSetID) → (sgID, usedBuffer, err)。
// 引入这个 hook 主要是为了在 HandleCreateInstance 里能触发 "selectedSG 为空字符串"
// 的防御分支（生产代码路径正常情况下不会返回空 SG）。
var selectSGForNewInstanceFn = SelectSGForNewInstance

// validateGlobalVpcAndSubnetsFn 指向 validateGlobalVpcAndSubnets，专供测试 stub。
// 生产代码走真实 VPC 云 API 校验；测试里让 handler 能绕过这步进入后续分支。
var validateGlobalVpcAndSubnetsFn = validateGlobalVpcAndSubnets

// validateCreateResourceConfigFn 指向 validateCreateResourceConfig，专供测试 stub。
// 生产代码调用云 API 预验证实例规格可售性；测试里替换为 nil 函数跳过预验证。
var validateCreateResourceConfigFn = validateCreateResourceConfig

// resolveResourceConfigFn resolves the independently managed resource policy.
// The hook remains replaceable so handler tests can exercise resolver failures.
var resolveResourceConfigFn = resolveEffectiveResourcePolicyConfig

func resolveEffectiveResourcePolicyConfig(ctx context.Context, groupID uint) (json.RawMessage, usergroup.Source, error) {
	resolved, err := model.ResolveEffectiveResourcePolicy(ctx, groupID)
	if err != nil {
		return nil, usergroup.Source{}, hcommon.I18nRichError(err, i18n.MsgDatabaseOperationFailed)
	}
	source := resourcePolicySource(resolved)
	return json.RawMessage(resolved.Policy.ConfigJSON), source, nil
}

// resourcePolicySource maps policy resolution metadata to its API source.
func resourcePolicySource(resolved *model.ResolvedResourcePolicy) usergroup.Source {
	source := usergroup.Source{Type: usergroup.SourceSiteDefault}
	if resolved.SourceGroupID == 0 {
		return source
	}
	source.Type = usergroup.SourceInherited
	if resolved.Depth == 0 {
		source.Type = usergroup.SourceLocal
	}
	source.GroupID = resolved.SourceGroupID
	return source
}

// errQuotaExceeded 实例配额超限哨兵。
// 在事务内部抛出，外层通过 errors.Is(txErr, errQuotaExceeded) 区分配额超限与其他错误。
var errQuotaExceeded = hcommon.I18nError(i18n.MsgInstanceQuotaReached)
var errGroupQuotaExceeded = hcommon.I18nError(i18n.MsgInstanceGroupQuotaReached)

// generateVpcName 根据独立站域名生成 VPC 名称。
// 从域名中提取前缀（如 x8swfkbg.tcaisite.com -> x8swfkbg）作为后缀。
// domain 为必填启动项
func generateVpcName(domain string) string {
	prefix := domain
	prefix = strings.TrimPrefix(prefix, "https://")
	prefix = strings.TrimPrefix(prefix, "http://")
	if idx := strings.Index(prefix, "."); idx > 0 {
		prefix = prefix[:idx]
	}
	slog.Info("使用独立站域名前缀作为 VPC 后缀", "domain", domain, "prefix", prefix)
	return "clawpro/default-vpc-" + prefix
}

// RequestUser 返回当前请求关联的用户，优先 Token 认证，其次 Session。
// 被封禁用户返回 (nil, error)；未登录/无效 Token 返回 (nil, nil)。
func RequestUser(r *http.Request) (*model.User, error) {
	user, err := getUserFromToken(r)
	if user != nil {
		return user, nil
	}
	if err != nil {
		// getUserFromToken 返回的 error 有三种：
		// 1. 封禁（BannedError）—— 透传，用户整体不可用
		// 2. Token 被禁用（TokenDisabledError）—— 仅影响外部 API 调用，
		//    但既然请求明确携带了 Bearer Token，应直接报错而非回退 session
		// 3. "无效的 API Token" —— 视为未登录，回退 session
		if errors.Is(err, model.BannedError{}) || errors.Is(err, model.TokenDisabledError{}) {
			return nil, err
		}
		return nil, nil
	}

	session := getSession(r)
	username, _ := session.Values["username"].(string)
	if username == "" {
		return nil, nil
	}

	// 多租户阶段二：校验 session 中的 identifier 与当前请求租户是否一致
	if !validateSessionIdentifier(session, r.Context()) {
		return nil, nil // 视为未登录
	}

	// 检查 OneID session 黑名单（仅 Gateway 模式下有效）
	// 条件与登出逻辑保持一致：只有同时配置了 GatewayURL 和 hcommon.TenantIDFromCtx(r.Context()) 才会有 OneID 登录/登出流程
	if GatewayURL != "" && hcommon.TenantIDFromCtx(r.Context()) != "" {
		sid, _ := session.Values["oneid_sid"].(string)
		sub, _ := session.Values["oneid_sub"].(string)
		loginAtUnix, _ := session.Values["login_at"].(int64)
		loginAt := time.Unix(loginAtUnix, 0)
		if model.IsSessionRevoked(r.Context(), sid, sub, loginAt) {
			return nil, model.RevokedError{}
		}
	}

	var u model.User
	if model.DB(r.Context()).Unscoped().Where("username = ?", username).First(&u).Error != nil {
		return nil, nil
	}
	// 检查用户是否被封禁（软删除）
	if u.DeletedAt.Valid {
		return nil, model.BannedError{}
	}
	return &u, nil
}

func getLoginUser(r *http.Request) (*model.User, error) {
	user, err := RequestUser(r)
	if err != nil {
		return nil, err
	}
	if user != nil && user.ID == 0 {
		return nil, nil
	}
	return user, nil
}

// InstanceWithProxy pairs an Instance with its AIModel display name.
type InstanceWithProxy struct {
	model.Instance
	AIModelDisplay string // "provider/model_id" or empty
}

// requireLogin checks login and writes error response if not authenticated.
// JSON requests get 401 JSON error; HTML requests get HX-Redirect to login page.
func requireLogin(w http.ResponseWriter, r *http.Request) *model.User {
	user, err := getLoginUser(r)
	if err != nil {
		if errors.Is(err, model.TokenDisabledError{}) {
			// Token 被禁用 —— 仅影响外部 API Token 调用，不影响 session 登录
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAPITokenDisabled))
			return nil
		}
		if errors.Is(err, model.RevokedError{}) {
			// OneID sid 已吊销：清 cookie，跳转首页触发重新登录
			session := getSession(r)
			session.Values["username"] = ""
			session.Options.MaxAge = -1
			session.Save(r, w)
			writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgSessionExpiredRelogin))
			return nil
		}
		// 用户被封禁
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAccountBanned))
		return nil
	}
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, ErrUnauthorized)
	}
	return user
}

// HandleInstanceDeniedActions 批量查询 claw 实例对应 CVM 的禁用操作。
// 仅返回 DescribeInstanceVncUrl 相关的 DeniedAction。
//
// 路由: POST /openclaw/denied-actions
//
// 请求体 (JSON):
//
//	{"ids": [1, 2, 3]}
//
// 响应:
//
//	{"instances": [{"id": 1, "denied_actions": [{"action":"...","code":"...","message":"..."}]}, ...]}
func HandleInstanceDeniedActions(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		IDs         []uint   `json:"ids"`
		InstanceIDs []string `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	// 查询属于当前用户的实例
	var instances []model.Instance
	if len(req.IDs) > 0 {
		if err := model.DB(r.Context()).Where("id IN ? AND user_id = ?", req.IDs, user.ID).Find(&instances).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
	} else if len(req.InstanceIDs) > 0 {
		if err := model.DB(r.Context()).Where("instance_id IN ? AND user_id = ?", req.InstanceIDs, user.ID).Find(&instances).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
	} else {
		// 缺少参数时返回空结果
		jsonOK(w, map[string]interface{}{"instances": []instanceDeniedActions{}})
		return
	}

	// 提取关联 CVM 的实例（本地 agent 实例 source=local 的 instance_id 是 host CID，
	// 不是 CVM 格式的 ins-xxxxxxxx，不能传给 CVM API）
	var cvmInstanceIds []string
	for _, inst := range instances {
		if inst.InstanceId != "" && inst.Source != model.InstanceSourceLocal {
			cvmInstanceIds = append(cvmInstanceIds, inst.InstanceId)
		}
	}

	// 如果没有关联 CVM 的实例，直接返回空 denied_actions
	if len(cvmInstanceIds) == 0 {
		result := make([]instanceDeniedActions, len(instances))
		for i, inst := range instances {
			result[i] = instanceDeniedActions{ID: inst.ID, DeniedActions: []deniedAction{}}
		}
		jsonOK(w, map[string]interface{}{"instances": result})
		return
	}

	cvmDenied, err := describeInstancesDeniedActions(r.Context(), cvmInstanceIds, []string{"DescribeInstanceVncUrl"})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceDeniedOpsFailed))
		return
	}

	// 组装最终结果
	result := make([]instanceDeniedActions, 0, len(instances))
	for _, inst := range instances {
		item := instanceDeniedActions{ID: inst.ID}
		if inst.InstanceId != "" {
			if denied, ok := cvmDenied[inst.InstanceId]; ok {
				item.DeniedActions = denied
			}
		}
		if item.DeniedActions == nil {
			item.DeniedActions = []deniedAction{}
		}
		result = append(result, item)
	}

	jsonOK(w, map[string]interface{}{"instances": result})
}

func HandleInstanceList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	log.Info("[InstanceList] 查询用户实例列表", "user_id", user.ID, "username", user.Username)

	// 查询用户实例列表（自有 ∪ 待我接收的移交）
	var instances []model.Instance
	if err := model.DB(ctx).
		Where("(user_id = ? OR handover_target_user_id = ?) AND is_doctor_node = ?", user.ID, user.ID, false).
		Order("created_at desc").Find(&instances).Error; err != nil {
		log.Error("[InstanceList] 查询用户实例列表失败", "user_id", user.ID, "error", err)
	}
	log.Info("[InstanceList] 用户实例数量", "user_id", user.ID, "count", len(instances))

	// 兼容处理：空的 agent_type 默认为 openclaw
	for i := range instances {
		instances[i].AgentType = model.NormalizeAgentType(instances[i].AgentType)
	}

	// 批量获取 CVM 信息
	var instanceIds []string
	for i := range instances {
		if instances[i].InstanceId != "" {
			instanceIds = append(instanceIds, instances[i].InstanceId)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), instanceIds)

	// 检查是否有已销毁超过 1 天的实例需要清理
	var destroyedIDs []uint
	for i := range instances {
		cvmInfo := cvmInfoMap[instances[i].InstanceId]
		status := ResolveInstanceStatus(r.Context(), &instances[i], cvmInfo, nil)
		if status.Status == model.StatusDestroyed {
			destroyedIDs = append(destroyedIDs, instances[i].ID)
		}
	}
	if len(destroyedIDs) > 0 {
		log.Info("[InstanceList] 清理已销毁超过 1 天的实例", "user_id", user.ID, "count", len(destroyedIDs), "instance_ids", destroyedIDs)
		model.CleanupDestroyedInstances(r.Context(), destroyedIDs, 24*time.Hour)
	}

	// 解析分页参数：page 默认 1，page_size 默认 30，最大 100
	page, pageSize := parsePagination(r, 100, 30)

	// 构建基础查询条件（自有 ∪ 待我接收的移交）
	baseQuery := model.DB(ctx).Model(&model.Instance{}).
		Where("(user_id = ? OR handover_target_user_id = ?) AND is_doctor_node = ?", user.ID, user.ID, false)

	// 支持精准搜索：id 不为 0 时优先按主键 ID 匹配，否则尝试按 instance_id 匹配
	var detailSearchID uint
	if idStr := strings.TrimSpace(r.URL.Query().Get("id")); idStr != "" {
		if searchID, err := strconv.ParseUint(idStr, 10, 64); err == nil && searchID > 0 {
			baseQuery = baseQuery.Where("id = ?", searchID)
			detailSearchID = uint(searchID)
			log.Info("[InstanceList] 按主键 ID 精准搜索", "user_id", user.ID, "search_id", searchID)
		}
	} else if instanceID := strings.TrimSpace(r.URL.Query().Get("instance_id")); instanceID != "" {
		baseQuery = baseQuery.Where("instance_id = ?", instanceID)
		log.Info("[InstanceList] 按 instance_id 精准搜索", "user_id", user.ID, "instance_id", instanceID)
	} else {
		// 模糊搜索：keyword 命中 name / instance_id，按 rune 截断 50 个字符避免切坏多字节字符
		keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
		if runes := []rune(keyword); len(runes) > 50 {
			keyword = string(runes[:50])
		}
		if keyword != "" {
			like := "%" + keyword + "%"
			baseQuery = baseQuery.Where("name LIKE ? OR instance_id LIKE ?", like, like)
			log.Info("[InstanceList] 按 keyword 模糊搜索", "user_id", user.ID, "keyword", keyword)
		}
	}

	// 按 agent_type 过滤（多值 OR，逗号分隔）
	if rawAgentType := strings.TrimSpace(r.URL.Query().Get("agent_type")); rawAgentType != "" {
		var types []string
		seen := make(map[string]struct{})
		for _, t := range strings.Split(rawAgentType, ",") {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			types = append(types, t)
		}
		if len(types) > 0 {
			baseQuery = baseQuery.Where("agent_type IN ?", types)
			log.Info("[InstanceList] 按 agent_type 过滤", "user_id", user.ID, "agent_types", types)
		}
	}

	// 查询总数（清理后的）
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		log.Error("[InstanceList] 查询实例总数失败", "user_id", user.ID, "error", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var freshInstances []model.Instance
	if err := baseQuery.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&freshInstances).Error; err != nil {
		log.Error("[InstanceList] 重新查询实例列表失败", "user_id", user.ID, "error", err)
	}

	// 批量查询角色名称
	roleNameMap := batchFetchRoleNames(r.Context(), freshInstances)

	// 收集需要查询模型信息的 AIModelID（> 1 的才是预置模型）
	var modelIDs []uint
	for _, inst := range freshInstances {
		if inst.AIModelID > 1 {
			modelIDs = append(modelIDs, inst.AIModelID)
		}
	}

	// 批量查询模型信息
	modelInfoMap := make(map[uint]model.AIModel)
	if len(modelIDs) > 0 {
		var models []model.AIModel
		if err := model.DB(ctx).Where("id IN ?", modelIDs).Find(&models).Error; err != nil {
			log.Error("[InstanceList] 批量查询模型信息失败", "error", err, "model_ids", modelIDs)
		}
		for _, m := range models {
			modelInfoMap[m.ID] = m
		}
	}

	type instanceWithRole struct {
		model.Instance
		RoleName        string  `json:"role_name"`
		OsName          string  `json:"os_name"`
		LightClawUserID string  `json:"light_claw_user_id"`
		ModelProvider   string  `json:"model_provider,omitempty"`
		ModelName       string  `json:"model_name,omitempty"`
		GroupFullPath   string  `json:"group_full_path"`           // 创建时所属分组全路径（如"研发中心/后端组"），GroupID=0 时为空串
		IsBuiltin       bool    `json:"is_builtin"`                // 该实例的 agent_type 是否为内置类型
		CompatibleWith  string  `json:"compatible_with,omitempty"` // 自定义类型兼容的内置类型
		HostName        string  `json:"host_name,omitempty"`       // 仅 source=local
		OS              string  `json:"os,omitempty"`              // 仅 source=local
		LastReportAt    *string `json:"last_report_at,omitempty"`  // 仅 source=local，RFC3339
		// 二期新增：本地 agent 资源信息（仅精准查询 + source=local 时填充）
		LocalAgentResources *LocalAgentResourcesView `json:"local_agent_resources,omitempty"`
		ListRole            string                   `json:"list_role"` // owner / handover_incoming（待我接收的移交）
		UserName            string                   `json:"user_name"` // UserID 对应的用户名
		// 存量实例分组归属处理（stale-instances v1.0）— 与 /admin/instances 同构
		Flags                    []string `json:"flags"`
		HandoverTargetUserID     uint     `json:"handover_target_user_id"`
		HandoverRejectedByUserID uint     `json:"handover_rejected_by_user_id"`
		HandoverInitiatedAt      *string  `json:"handover_initiated_at"`
	}
	result := make([]instanceWithRole, len(freshInstances))
	// 🆕 批量查询 GroupID → full_path
	var groupIDs []uint
	for _, inst := range freshInstances {
		if inst.GroupID != 0 {
			groupIDs = append(groupIDs, inst.GroupID)
		}
	}
	groupPathMap := fetchGroupFullPathMap(r.Context(), groupIDs)

	// 🆕 批量预拉 LocalInstanceInfo（仅 source=local 的实例）
	var localInstancePKs []uint
	for _, inst := range freshInstances {
		if inst.Source == model.InstanceSourceLocal {
			localInstancePKs = append(localInstancePKs, inst.ID)
		}
	}
	localInfoMap := make(map[uint]model.LocalInstanceInfo, len(localInstancePKs))
	if len(localInstancePKs) > 0 {
		var infos []model.LocalInstanceInfo
		if err := model.DB(ctx).Where("instance_id IN ?", localInstancePKs).Find(&infos).Error; err != nil {
			log.Warn("[InstanceList] 批量查询 LocalInstanceInfo 失败", "error", err)
		}
		for _, info := range infos {
			localInfoMap[info.InstanceID] = info
		}
	}

	// 🆕 批量查 user_id → username（Unscoped 兼容被软删的历史 owner）
	userNameMap := map[uint]string{}
	{
		userIDSet := make(map[uint]struct{}, len(freshInstances))
		for _, inst := range freshInstances {
			if inst.UserID != 0 {
				userIDSet[inst.UserID] = struct{}{}
			}
		}
		if len(userIDSet) > 0 {
			ids := make([]uint, 0, len(userIDSet))
			for uid := range userIDSet {
				ids = append(ids, uid)
			}
			var rows []struct {
				ID       uint   `gorm:"column:id"`
				Username string `gorm:"column:username"`
			}
			_ = model.DB(ctx).Unscoped().Model(&model.User{}).
				Select("id, username").
				Where("id IN ?", ids).
				Find(&rows).Error
			for _, r := range rows {
				userNameMap[r.ID] = r.Username
			}
		}
	}

	for i, inst := range freshInstances {
		chargeType := instanceChargeTypeOrDefault(inst.InstanceChargeType)
		itemOsName := ""
		if cvmInfo := cvmInfoMap[inst.InstanceId]; cvmInfo != nil {
			if cvmInfo.InstanceChargeType != "" {
				chargeType = cvmInfo.InstanceChargeType
			}
			itemOsName = cvmInfo.OsName
		}
		inst.InstanceChargeType = chargeType
		listRole := "owner"
		if inst.UserID != user.ID && inst.HandoverTargetUserID == user.ID {
			listRole = "handover_incoming"
			// 对待接收方做最小白名单：屏蔽原 owner 私有配置
			inst.CustomModelConfig = ""
			inst.UserData = ""
		}
		item := instanceWithRole{
			Instance:                 inst,
			RoleName:                 roleNameMap[inst.RoleID],
			LightClawUserID:          lightClawUserID(r.Context(), user.ID),
			GroupFullPath:            groupPathMap[inst.GroupID],
			IsBuiltin:                model.IsBuiltinAgentType(inst.AgentType),
			CompatibleWith:           model.GetAgentRuntimeType(ctx, inst.AgentType),
			OsName:                   itemOsName,
			ListRole:                 listRole,
			UserName:                 userNameMap[inst.UserID],
			Flags:                    []string{}, // 默认空数组（非 null）；后续 enrich 填充
			HandoverTargetUserID:     inst.HandoverTargetUserID,
			HandoverRejectedByUserID: inst.HandoverRejectedByUserID,
			HandoverInitiatedAt:      formatNullableTime(inst.HandoverInitiatedAt),
		}
		// 内置类型不需要 compatible_with（它就是自身）
		if item.IsBuiltin {
			item.CompatibleWith = ""
		}
		// 只有 AIModelID > 1 才填充模型信息（0 和 1 是自定义模型，用 CustomModelConfig）
		if inst.AIModelID > 1 {
			if m, ok := modelInfoMap[inst.AIModelID]; ok {
				item.ModelProvider = m.Provider
				item.ModelName = m.ModelName
			}
		}
		// 本地实例补充托管信息
		if inst.Source == model.InstanceSourceLocal {
			if info, ok := localInfoMap[inst.ID]; ok {
				item.HostName = info.HostName
				item.OS = info.OS
				if info.LastReportAt != nil {
					s := info.LastReportAt.UTC().Format(time.RFC3339)
					item.LastReportAt = &s
				}
			}
			// 二期：精准查询时填充 local_agent_resources
			if detailSearchID != 0 && inst.ID == detailSearchID {
				item.LocalAgentResources = buildLocalAgentResourcesView(ctx, &inst, user.ID)
			}
		}
		result[i] = item
	}

	// 批量回填 stale-instances v1.0 的 flags 字段（instance_flags 是独立表，必须单独查）
	if len(result) > 0 {
		flagIDs := make([]uint, 0, len(result))
		for i := range result {
			flagIDs = append(flagIDs, result[i].ID)
		}
		flagsMap, err := model.GetInstanceFlagsBatch(r.Context(), flagIDs)
		if err != nil {
			slog.Warn("[InstanceList] enrich flags failed", "user_id", user.ID, "err", err)
		} else {
			for i := range result {
				if v, ok := flagsMap[result[i].ID]; ok && v != nil {
					result[i].Flags = v
				}
			}
		}
	}

	log.Info("[InstanceList] 返回 JSON 实例列表", "user_id", user.ID, "count", len(result), "page", page, "page_size", pageSize, "total", total)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	jsonOK(w, map[string]interface{}{
		"instances":   result,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

var openclawMultiAgentScriptRunner = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string) (string, error) {
	return RunScript(ctx, instanceID, scriptName, timeout, runtimeUser, nil, nil)
}

type multiAgentCheckResult struct {
	Count int `json:"count"`
}

func parseMultiAgentCheckResult(output string) (multiAgentCheckResult, error) {
	var result multiAgentCheckResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return result, hcommon.I18nError(i18n.MsgMultiAgentParseResultFailed).WithDetail(err.Error())
	}
	return result, nil
}

// HandleAgentCount 查询用户实例内部的 agent 数量。
// GET /openclaw/agent-count?id=123 或 ?instance_id=ins-xxx
func HandleAgentCount(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}
	if !model.AgentTypeSupportsMultiAgent(r.Context(), instance.AgentType) {
		writeAgentCountOK(w, 1)
		return
	}

	scriptName, rerr := ResolveScript(r.Context(), "check_multi_agent", instance.AgentType)
	if rerr != nil {
		writeAgentCountOK(w, 1)
		return
	}

	output, err := openclawMultiAgentScriptRunner(r.Context(), instance.InstanceId, scriptName, 30, instance.RuntimeUser)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMultiAgentQueryFailed))
		return
	}

	result, err := parseMultiAgentCheckResult(output)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	writeAgentCountOK(w, result.Count)
}

func writeAgentCountOK(w http.ResponseWriter, agentCount int) {
	if agentCount < 1 {
		agentCount = 1
	}
	jsonOK(w, map[string]interface{}{
		"ok":    true,
		"count": agentCount,
	})
}

// HandleCurrentImage 返回当前启用的镜像信息，前端据此判断是否展示"一键更新"按钮。
// 优先按实例的 agent_type 查询对应类型的启用镜像；若未传 agent_type 则回退到任意启用镜像。
// public 字段直接由 image_type 决定：PUBLIC_IMAGE → true。
func HandleCurrentImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	agentType := strings.TrimSpace(r.URL.Query().Get("agent_type"))
	log.Info("[CurrentImage] 查询当前启用镜像", "user_id", user.ID, "agent_type", agentType)

	type imageResp struct {
		model.AIImage
		Public bool `json:"public"`
	}

	if agentType != "" {
		img, err := model.GetEnabledImageByType(r.Context(), agentType)
		if err != nil {
			log.Error("[CurrentImage] 查询镜像失败", "user_id", user.ID, "agent_type", agentType, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
			return
		}
		if img == nil {
			log.Info("[CurrentImage] 未找到启用镜像", "user_id", user.ID, "agent_type", agentType)
			jsonOK(w, map[string]interface{}{"image": nil})
			return
		}
		log.Info("[CurrentImage] 返回启用镜像", "user_id", user.ID, "agent_type", agentType, "image_id", img.ImageId, "image_type", img.ImageType)
		jsonOK(w, map[string]interface{}{"image": imageResp{
			AIImage: *img,
			Public:  img.ImageType == "PUBLIC_IMAGE",
		}})
		return
	}

	// 兼容旧调用：未传 agent_type 时回退到任意启用镜像
	img := model.GetEnabledImage(r.Context())
	if img == nil {
		log.Info("[CurrentImage] 未找到任何启用镜像（无 agent_type）", "user_id", user.ID)
		jsonOK(w, map[string]interface{}{"image": nil})
		return
	}
	log.Info("[CurrentImage] 返回启用镜像（无 agent_type 回退）", "user_id", user.ID, "image_id", img.ImageId, "image_type", img.ImageType)
	jsonOK(w, map[string]interface{}{"image": imageResp{
		AIImage: *img,
		Public:  img.ImageType == "PUBLIC_IMAGE",
	}})
}

// HandleRenameInstance 修改实例名称（同步更新腾讯云 CVM InstanceName + 本地数据库）。
// 仅允许 RUNNING / STOPPED 状态的实例改名；先改 CVM，成功后再改本地 DB。
//
// 路由: POST /openclaw/rename
//
// 参数:
//   - id: 实例 ID
//   - name: 新名称（1~128 字符）
//
// 响应: {"ok": true}
func HandleRenameInstance(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" || len(name) > 128 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNameRequired))
		return
	}

	// 本地 agent 实例（source=local）的 instance_id 是 host CID，不是 CVM 格式，
	// 不能调 ModifyInstancesAttribute；直接改本地 DB 后返回。
	if instance.Source == model.InstanceSourceLocal {
		if err := model.DB(r.Context()).Model(instance).Update("name", name).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateLocalNameFailed))
			return
		}
		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	// 必须有关联的 CVM 实例（创建中/创建失败没有 InstanceId，不允许改名）
	if instance.InstanceId == "" {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgInstanceNotReadyForNameChange))
		return
	}

	// 状态校验：仅 RUNNING / STOPPED 允许改名
	cvmState, err := fetchCVMState(r.Context(), instance.InstanceId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceStateFailed))
		return
	}
	if cvmState != "RUNNING" && cvmState != "STOPPED" {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgCurrentStateCannotChangeName, cvmState))
		return
	}

	// 先修改腾讯云 CVM 实例名称
	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	modReq := cvm.NewModifyInstancesAttributeRequest()
	modReq.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	modReq.InstanceName = common.StringPtr(name)
	if _, err := client.ModifyInstancesAttribute(modReq); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModifyCVMNameFailed))
		return
	}

	// CVM 改名成功，再更新本地数据库
	if err := model.DB(r.Context()).Model(instance).Update("name", name).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateLocalNameFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	if r.Method != http.MethodPost {
		log.Warn("[DeleteInstance] 非法方法", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Warn("[DeleteInstance] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.IsDoctorNode {
		log.Warn("[DeleteInstance] 拒绝龙虾医生节点", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}

	// 本地实例：不走 /openclaw/delete 删除路径（无 CVM 实体可 Terminate，且应走
	// 下发 uninstall_teamai 任务 → reporter 执行 → ack 异步软删 的语义）。
	// 用户侧删除本地 Agent 请走 /local-agent/remove；管控侧走 /admin/local-agent/remove。
	if instance.Source == model.InstanceSourceLocal {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp))
		return
	}
	log.Info("[DeleteInstance] 收到删除请求", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "name", instance.Name)

	// 查询当前 CVM 状态，用于状态校验和分化删除判断
	cvmInfo, err := fetchCVMInstanceInfo(r.Context(), instance.InstanceId)
	if err != nil {
		log.Warn("[DeleteInstance] 查询 CVM 信息失败，按本地状态继续", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
	}
	currentStatus := ResolveInstanceStatus(r.Context(), instance, cvmInfo, nil)
	log.Info("[DeleteInstance] 当前实例状态", "user_id", user.ID, "instance_id", instance.ID, "status", currentStatus.Status)

	// 校验当前状态是否允许删除（loading/pending/creating 禁止删除）
	if err := canOperate(instance, model.OpDelete, currentStatus.Status); err != nil {
		log.Warn("[DeleteInstance] 当前状态不允许删除", "user_id", user.ID, "instance_id", instance.ID, "status", currentStatus.Status, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 分化删除：创建失败/已销毁/无 InstanceId → 不调用 CVM API，按“CVM 已不存在”策略尽力释放 Pro 记忆库
	if instance.InstanceId == "" || currentStatus.Status == model.StatusCreateFailed || currentStatus.Status == model.StatusDestroyed {
		log.Info("[DeleteInstance] 走本地清理流程（无 CVM 或已销毁）", "user_id", user.ID, "instance_id", instance.ID, "status", currentStatus.Status)
		if !ReleaseProMemSpaceForMissingInstance(r.Context(), instance.InstanceId) {
			log.Error("[DeleteInstance] Pro 记忆库释放失败，中止删除", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDeleteAbortProMemFailed))
			return
		}
		model.DB(r.Context()).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
		CleanupVNCState(instance.ID) // 清理 VNC 相关内存状态（aiActiveMap、takeoverMap 等）
		MarkInstanceUnbound(r.Context(), model.CurrentIdentifier(r.Context()), instance.SecurityGroupId)
		cleanupAgentProxyRoutes(r.Context(), "[DeleteInstance]", instance.InstanceId, instance.ID)
		model.DB(r.Context()).Delete(instance)
		MarkPersonalSpaceToBeDeleted(r.Context(), instance.ID)
		model.DeleteMemoryTDAIPluginRow(r.Context(), instance.InstanceId)
		log.Info("[DeleteInstance] 本地清理完成", "user_id", user.ID, "instance_id", instance.ID)

		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	// 正常实例：乐观锁写入 delete 操作标记
	if err := setOperation(model.DB(r.Context()), instance, model.OpDelete); err != nil {
		log.Warn("[DeleteInstance] 写入删除操作标记失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// Pro 记忆库的导出/释放由 instance_state 轮询感知到 CVM 销毁后异步处理，不再同步阻塞删除流程

	// 调用 TerminateInstances（进入 ISOLATED 状态）
	client, err := NewCVMClient(ctx)
	if err != nil {
		log.Error("[DeleteInstance] 创建 CVM 客户端失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	req := cvm.NewTerminateInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	if _, err := client.TerminateInstances(req); err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
			// CVM 已不存在：本路径不会走到 instance_state 副作用的释放分支（DB 行马上要被删，
			// 下一轮轮询扫不到），所以必须在这里主动释放远端 Pro 记忆库；
			// 释放失败 → 保留 instance 与 plugin 行，等待 instance_state 轮询副作用补偿。
			log.Warn("[DeleteInstance] CVM 已不存在，准备清理本地记录", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)
			if !ReleaseProMemSpaceForMissingInstance(r.Context(), instance.InstanceId) {
				log.Error("[DeleteInstance] CVM 已不存在但 Pro 记忆库释放失败，保留 DB 记录", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)
				clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDeleteRetainProMemFailed))
				return
			}
			model.DB(r.Context()).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
			cleanupAgentProxyRoutes(r.Context(), "[DeleteInstance]", instance.InstanceId, instance.ID)
			model.DB(r.Context()).Delete(instance)
			model.DeleteMemoryTDAIPluginRow(r.Context(), instance.InstanceId)
		} else {
			log.Error("[DeleteInstance] 调用 TerminateInstances 失败", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
			clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(cvmTerminateInstancesError(err, user.Role == "admin")))
			return
		}
	}
	clearAdjustmentFailure(r.Context(), instance.ID)

	MarkPersonalSpaceToBeDeleted(r.Context(), instance.ID)
	log.Info("[DeleteInstance] 已标记个人空间待删除", "user_id", user.ID, "instance_id", instance.ID)

	// 异步写入删除成功通知（必须在 DB 删除之前，用拷贝的 instance 信息）
	go model.CreateSuccessNotification(hcommon.DetachContext(r.Context()),
		instance.UserID, instance.ID, instance.Name,
		model.NotifyTypeInstanceDeleteSuccess, "实例删除成功",
		fmt.Sprintf("您的实例「%s」已成功删除。", instance.Name),
	)

	log.Info("[DeleteInstance] 删除流程成功触发，CVM 进入 ISOLATED", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)

	// 成功：返回（Purge 在 /status 副作用中异步完成）
	jsonOK(w, map[string]interface{}{"ok": true})
}

// validateGlobalVpcAndSubnets 校验管理员手动配置的全局 VPC 和子网在云端是否仍然存在。
// 返回 error 时包含友好的用户提示信息。subnetMap 为 zone -> []subnetId（每可用区可多子网）。
func validateGlobalVpcAndSubnets(ctx context.Context, vpcId string, subnetMap map[string][]string) error {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}
	return validateGlobalVpcAndSubnetsCore(vpcClient, vpcId, subnetMap)
}

// vpcValidatorClient 抽象 validateGlobalVpcAndSubnetsCore 需要的 VPC 能力，便于单测。
type vpcValidatorClient interface {
	DescribeVpcs(req *vpc.DescribeVpcsRequest) (*vpc.DescribeVpcsResponse, error)
	DescribeSubnets(req *vpc.DescribeSubnetsRequest) (*vpc.DescribeSubnetsResponse, error)
}

// validateGlobalVpcAndSubnetsCore 是 validateGlobalVpcAndSubnets 的可测试核心，
// 不依赖 newVpcClient 全局构造函数。
func validateGlobalVpcAndSubnetsCore(vpcClient vpcValidatorClient, vpcId string, subnetMap map[string][]string) error {
	// 1. 校验 VPC 是否存在
	descVpcReq := vpc.NewDescribeVpcsRequest()
	descVpcReq.VpcIds = common.StringPtrs([]string{vpcId})
	descVpcResp, err := vpcClient.DescribeVpcs(descVpcReq)
	if err != nil {
		return hcommon.I18nError(i18n.MsgQueryGlobalVPCFailed).WithDetail(err.Error())
	}
	if descVpcResp.Response == nil || len(descVpcResp.Response.VpcSet) == 0 {
		return hcommon.I18nError(i18n.MsgVPCNotExistContactAdmin, vpcId)
	}

	// 2. 收集所有子网 ID（去重），批量校验
	seen := make(map[string]bool)
	subnetIds := make([]string, 0)
	for _, sids := range subnetMap {
		for _, sid := range sids {
			if sid == "" || seen[sid] {
				continue
			}
			seen[sid] = true
			subnetIds = append(subnetIds, sid)
		}
	}
	if len(subnetIds) == 0 {
		return nil
	}

	descSubnetReq := vpc.NewDescribeSubnetsRequest()
	descSubnetReq.SubnetIds = common.StringPtrs(subnetIds)
	descSubnetResp, err := vpcClient.DescribeSubnets(descSubnetReq)
	if err != nil {
		return hcommon.I18nError(i18n.MsgQueryGlobalSubnetFailed).WithDetail(err.Error())
	}

	// 构建云端实际存在的子网集合
	cloudSubnetIds := make(map[string]bool)
	if descSubnetResp.Response != nil {
		for _, sn := range descSubnetResp.Response.SubnetSet {
			if sn.SubnetId != nil {
				cloudSubnetIds[*sn.SubnetId] = true
			}
		}
	}

	// 找出已不存在的子网
	var missing []string
	for _, sids := range subnetMap {
		for _, sid := range sids {
			if !cloudSubnetIds[sid] {
				missing = append(missing, sid)
			}
		}
	}
	if len(missing) > 0 {
		return hcommon.I18nError(i18n.MsgSubnetNotExistContactAdmin, vpcId, strings.Join(missing, "、"))
	}

	return nil
}

// ensureDefaultVpcAndSubnets 确保共享的默认 VPC (clawpro/default-vpc) 和所有 Regions 可用区的子网存在。
// 如果 VPC 或某些可用区的子网不存在，则自动创建。
// 返回 vpcId 和 zone->[]subnetId 映射（每个可用区支持多个子网）。
func ensureDefaultVpcAndSubnets(ctx context.Context, config *model.SiteConfig) (string, map[string][]string, error) {
	// 分布式锁：多实例间互斥，防止并发创建重复 VPC/子网
	lock, err := model.AcquireLock(ctx, "ensure-default-vpc", 30*time.Second)
	if err != nil {
		return "", nil, hcommon.I18nError(i18n.MsgAcquireDistributedLockFailed).WithDetail(err.Error())
	}
	defer lock.Release()

	ri, ok := Regions[CVMRegion]
	if !ok || len(ri.Zones) == 0 {
		return "", nil, hcommon.I18nError(i18n.MsgRegionNotConfiguredOrNoZone, CVMRegion)
	}
	zones := ri.Zones

	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return "", nil, hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}

	vpcId := config.DefaultVpcId
	subnetMap := config.GetDefaultSubnetMap()
	changed := false

	// ---- 1. 确保默认 VPC 存在 ----

	// 如果本地已记录 vpcId，先验证云端是否仍存在（可能被在腾讯云控制台手动删除）
	if vpcId != "" {
		checkReq := vpc.NewDescribeVpcsRequest()
		checkReq.VpcIds = common.StringPtrs([]string{vpcId})
		checkResp, err := vpcClient.DescribeVpcs(checkReq)
		if err != nil {
			// 查询失败（网络/限频等），无法确认 VPC 状态，直接报错而非盲目重建
			return "", nil, hcommon.I18nError(i18n.MsgVerifyDefaultVPCQueryFailed).WithDetail(err.Error())
		}
		if checkResp.Response == nil || len(checkResp.Response.VpcSet) == 0 {
			slog.Warn("本地记录的默认 VPC 在云端已不存在，清除本地记录并重建", "vpc_id", vpcId)
			vpcId = ""
			// VPC 已失效，清空本地子网映射（后续会重建 VPC 和子网并持久化）
			subnetMap = make(map[string][]string)
		}
	}

	if vpcId == "" {
		slog.Info("vpcId 为空, 创建默认VPC")

		// 尝试按名称查找已有的 VPC
		descReq := vpc.NewDescribeVpcsRequest()
		vpcName := generateVpcName(hcommon.DomainFromCtx(ctx))
		descReq.Filters = []*vpc.Filter{
			{
				Name:   common.StringPtr("vpc-name"),
				Values: common.StringPtrs([]string{vpcName}),
			},
		}
		descResp, err := vpcClient.DescribeVpcs(descReq)
		if err != nil {
			return "", nil, hcommon.I18nError(i18n.MsgQueryDefaultVPCFailed).WithDetail(err.Error())
		}
		if descResp.Response != nil && len(descResp.Response.VpcSet) > 0 {
			vpcId = *descResp.Response.VpcSet[0].VpcId
			slog.Info("找到已有默认 VPC", "vpc_id", vpcId, "vpc_name", vpcName)
		} else {
			// 创建默认 VPC
			createReq := vpc.NewCreateVpcRequest()
			createReq.VpcName = common.StringPtr(vpcName)
			createReq.CidrBlock = common.StringPtr("10.0.0.0/16")
			createResp, err := vpcClient.CreateVpc(createReq)
			if err != nil {
				return "", nil, hcommon.I18nError(i18n.MsgCreateDefaultVPCFailed).WithDetail(err.Error())
			}
			if createResp.Response == nil || createResp.Response.Vpc == nil || createResp.Response.Vpc.VpcId == nil {
				return "", nil, hcommon.I18nError(i18n.MsgCreateDefaultVPCDataError)
			}
			vpcId = *createResp.Response.Vpc.VpcId
			slog.Info("创建默认 VPC 成功", "vpc_id", vpcId, "vpc_name", vpcName)
		}

		config.DefaultVpcId = vpcId
		changed = true
	}

	// ---- 2. 确保每个可用区都有子网 ----

	// 查询 VPC 内已有的所有子网 CIDR，计算已占用的最大 slot 号，
	// 从 maxSlot+1 开始分配，避免与已有子网（包括不在 subnetMap 中的残留子网）CIDR 冲突。
	nextSlot := 0
	descSubnetsReq := vpc.NewDescribeSubnetsRequest()
	descSubnetsReq.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("vpc-id"),
			Values: common.StringPtrs([]string{vpcId}),
		},
	}
	descSubnetsResp, err := vpcClient.DescribeSubnets(descSubnetsReq)
	if err != nil {
		return "", nil, hcommon.I18nError(i18n.MsgQueryVPCSubnetsFailed).WithDetail(err.Error())
	}
	if descSubnetsResp.Response != nil {
		// 构建云端实际存在的 subnetId 集合
		cloudSubnetIds := make(map[string]bool)
		for _, sn := range descSubnetsResp.Response.SubnetSet {
			if sn.SubnetId != nil {
				cloudSubnetIds[*sn.SubnetId] = true
			}
		}

		// 构建 subnetMap 中已记录的 subnetId 集合，用于快速判重
		knownSubnetIds := make(map[string]bool)
		for _, sids := range subnetMap {
			for _, sid := range sids {
				knownSubnetIds[sid] = true
			}
		}

		// 反向清理：移除本地 subnetMap 中已不存在于云端的子网（如在腾讯云控制台手动删除的）
		for zone, sids := range subnetMap {
			var validSids []string
			for _, sid := range sids {
				if cloudSubnetIds[sid] {
					validSids = append(validSids, sid)
				} else {
					slog.Warn("本地子网在云端已不存在，从映射中移除", "subnet_id", sid, "zone", zone)
					changed = true
				}
			}
			if len(validSids) == 0 {
				delete(subnetMap, zone)
			} else {
				subnetMap[zone] = validSids
			}
		}

		for _, sn := range descSubnetsResp.Response.SubnetSet {
			if sn.CidrBlock == nil {
				continue
			}
			// 解析 10.0.{slot*32}.0/19 中的 slot 值
			var thirdOctet int
			if _, err := fmt.Sscanf(*sn.CidrBlock, "10.0.%d.0/19", &thirdOctet); err == nil {
				slot := thirdOctet / 32
				if slot+1 > nextSlot {
					nextSlot = slot + 1
				}
			}

			// 同步云端已有但本地 subnetMap 未记录的子网（如管理员在腾讯云控制台手工创建的）
			if sn.SubnetId != nil && sn.Zone != nil && !knownSubnetIds[*sn.SubnetId] {
				subnetMap[*sn.Zone] = append(subnetMap[*sn.Zone], *sn.SubnetId)
				knownSubnetIds[*sn.SubnetId] = true
				changed = true
				slog.Info("同步云端子网到本地映射", "subnet_id", *sn.SubnetId, "zone", *sn.Zone)
			}
		}
	}

	for _, zone := range zones {
		if sids, exists := subnetMap[zone]; exists && len(sids) > 0 {
			continue
		}

		// 为该可用区创建子网，CIDR: 10.0.{nextSlot*32}.0/19（每个子网 8190 个 IP）
		cidr := fmt.Sprintf("10.0.%d.0/19", nextSlot*32)
		subnetName := "clawpro/" + zone

		createSubnetReq := vpc.NewCreateSubnetRequest()
		createSubnetReq.VpcId = common.StringPtr(vpcId)
		createSubnetReq.SubnetName = common.StringPtr(subnetName)
		createSubnetReq.CidrBlock = common.StringPtr(cidr)
		createSubnetReq.Zone = common.StringPtr(zone)

		createSubnetResp, err := vpcClient.CreateSubnet(createSubnetReq)
		if err != nil {
			return "", nil, hcommon.I18nError(i18n.MsgCreateSubnetFailed, subnetName, zone, cidr).WithDetail(err.Error())
		}
		if createSubnetResp.Response == nil || createSubnetResp.Response.Subnet == nil || createSubnetResp.Response.Subnet.SubnetId == nil {
			return "", nil, hcommon.I18nError(i18n.MsgCreateSubnetDataError, subnetName)
		}

		subnetId := *createSubnetResp.Response.Subnet.SubnetId
		subnetMap[zone] = append(subnetMap[zone], subnetId)
		changed = true
		nextSlot++
		slog.Info("创建默认子网成功", "subnet_id", subnetId, "zone", zone, "cidr", cidr, "subnet_name", subnetName)
	}

	// ---- 3. 持久化 VPC ID 与子网映射 ----
	if changed {
		if err := config.SetDefaultSubnetMap(subnetMap); err != nil {
			return "", nil, hcommon.I18nError(i18n.MsgSerializeSubnetMapFailed).WithDetail(err.Error())
		}
		if err := model.DB(ctx).Model(config).Updates(map[string]interface{}{
			"default_vpc_id":     config.DefaultVpcId,
			"default_subnet_ids": config.DefaultSubnetIds,
		}).Error; err != nil {
			return "", nil, hcommon.I18nError(i18n.MsgPersistVPCConfigFailed).WithDetail(err.Error())
		}
	}

	return vpcId, subnetMap, nil
}

func parseCreateInstanceDiskType(raw string) (string, error) {
	diskType := strings.ToUpper(strings.TrimSpace(raw))
	if diskType == "" {
		return "", nil
	}
	if err := model.ValidateDiskType(diskType); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgWrongDiskType)
	}
	return diskType, nil
}

func applyCreateInstanceDiskType(request *cvm.RunInstancesRequest, diskType string) {
	if request == nil || diskType == "" {
		return
	}
	if request.SystemDisk == nil {
		request.SystemDisk = &cvm.SystemDisk{}
	}
	request.SystemDisk.DiskType = common.StringPtr(diskType)
}

func HandleCreateInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	if r.Method != http.MethodPost {
		log.Warn("[CreateInstance] 非法方法", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}
	result, ok := createInstance(w, r, user, createInstanceOptions{})
	if !ok {
		return
	}
	jsonOK(w, map[string]interface{}{
		"ok":          true,
		"redirect":    "/openclaw",
		"instance_id": result.InstanceID,
	})
}

func createInstance(w http.ResponseWriter, r *http.Request, user *model.User, options createInstanceOptions) (result createInstanceResult, ok bool) {
	ctx := r.Context()
	log := Logger(ctx)
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" || len(name) > 128 {
		log.Warn("[CreateInstance] 实例名称非法", "user_id", user.ID, "name_len", len(name))
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNameRequired))
		return
	}
	log.Info("[CreateInstance] 收到创建请求", "user_id", user.ID, "username", user.Username, "name", name)

	var (
		customTags []createInstanceTag
		err        error
	)
	if options.CustomTags != nil {
		customTags = *options.CustomTags
	} else {
		customTags, err = parseCreateInstanceTags(r.FormValue("tags"))
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
			return
		}
	}

	selectedDiskType, err := parseCreateInstanceDiskType(r.FormValue("disk_type"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 解析可选的 role_id 参数
	var roleID uint
	if roleIDStr := r.FormValue("role_id"); roleIDStr != "" {
		if rid, err := strconv.ParseUint(roleIDStr, 10, 64); err == nil && rid > 0 {
			// 校验角色存在且可见
			var role model.OpenClawRole
			if model.DB(r.Context()).Where("id = ? AND visible = ?", rid, true).First(&role).Error != nil {
				log.Warn("[CreateInstance] 所选角色不存在或不可用", "user_id", user.ID, "role_id", rid)
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRoleNotExistOrUnavailable))
				return
			}
			roleID = uint(rid)
		}
	}
	// 解析可选的 group_id 参数
	var groupID uint
	if groupIDStr := r.FormValue("group_id"); groupIDStr != "" {
		if gid, err := strconv.ParseUint(groupIDStr, 10, 64); err == nil && gid > 0 {
			// 校验分组存在性
			if err := usergroup.ValidateGroupIDs(r.Context(), []uint{uint(gid)}); err != nil {
				log.Warn("[CreateInstance] 所选分组不存在", "user_id", user.ID, "group_id", gid)
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupNotExist))
				return
			}
			groupID = uint(gid)
		}
	}

	// 校验角色对所选分组的可见性
	if roleID > 0 {
		if groupID > 0 {
			roleGroupIDs, _ := usergroup.GetAncestorIDs(r.Context(), groupID)
			if !usergroup.IsRoleVisibleToGroups(r.Context(), roleID, roleGroupIDs) {
				log.Warn("[CreateInstance] 所选分组不支持该角色", "user_id", user.ID, "group_id", groupID, "role_id", roleID)
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupNotSupportRole))
				return
			}
		} else {
			// 未分组用户只能使用 visibility_type='all' 的角色
			if !usergroup.IsRoleGloballyVisible(r.Context(), roleID) {
				log.Warn("[CreateInstance] 未分组用户不支持使用该角色", "user_id", user.ID, "role_id", roleID)
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRoleNotSupportUngroupedUser))
				return
			}
		}
	}

	// 解析 agent_type 参数
	agentType := model.NormalizeAgentType(strings.TrimSpace(r.FormValue("agent_type")))

	// 校验 agent_type（硬编码白名单校验）
	if err := checkAgentTypeValid(r.Context(), agentType); err != nil {
		slog.WarnContext(r.Context(), "[CreateInstance] 无效的智能体类型",
			"agent_type", agentType, "user", user.Username)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if !model.IsAgentTypeEnabled(r.Context(), agentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, agentType)
		slog.WarnContext(r.Context(), "[CreateInstance] 智能体类型已禁用",
			"agent_type", agentType, "user", user.Username)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgTypeCannotCreateContactAdmin, typeName))
		return
	}

	// 校验用户是否有权创建该镜像类型的实例（按 agent 绑定的分组可见性）
	allAgentTypeNames := make([]string, 0, len(model.GetAllAgentTypes(r.Context())))
	for _, t := range model.GetAllAgentTypes(r.Context()) {
		allAgentTypeNames = append(allAgentTypeNames, t.Code)
	}
	if groupID > 0 {
		imageGroupIDs, _ := usergroup.GetAncestorIDs(r.Context(), groupID)
		visibleTypes, err := usergroup.ResolveImageTypes(r.Context(), imageGroupIDs, allAgentTypeNames)
		if err != nil {
			slog.ErrorContext(r.Context(), "[CreateInstance] 解析镜像类型可见性失败",
				"user_id", user.ID, "group_id", groupID, "agent_type", agentType, "error", err)
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nError(i18n.MsgResolveVisibleImageTypesFailed).WithDetail(err.Error()))
			return
		}
		typeVisible := false
		for _, t := range visibleTypes {
			if t == agentType {
				typeVisible = true
				break
			}
		}
		if !typeVisible {
			typeName := model.GetAgentTypeDisplayName(ctx, agentType)
			slog.WarnContext(r.Context(), "[CreateInstance] 用户无权创建该镜像类型",
				"user_id", user.ID, "agent_type", agentType)
			writeError(w, r, http.StatusForbidden,
				hcommon.I18nError(i18n.MsgNoPermCreateInstanceType, typeName))
			return
		}
	} else {
		// 未分组用户：排除被分组限制的镜像类型（已在 group_config_bindings 中绑定的）
		restricted, err := model.GetRestrictedImageTypes(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "[CreateInstance] 查询受限镜像类型失败",
				"user_id", user.ID, "agent_type", agentType, "error", err)
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nError(i18n.MsgQueryRestrictedImageTypesFailed).WithDetail(err.Error()))
			return
		}
		restrictedSet := make(map[string]bool, len(restricted))
		for _, rt := range restricted {
			restrictedSet[rt] = true
		}
		if restrictedSet[agentType] {
			typeName := model.GetAgentTypeDisplayName(r.Context(), agentType)
			slog.WarnContext(r.Context(), "[CreateInstance] 未分组用户无权创建受限镜像类型",
				"user_id", user.ID, "agent_type", agentType)
			writeError(w, r, http.StatusForbidden,
				hcommon.I18nError(i18n.MsgNoPermCreateInstanceType, typeName))
			return
		}
	}

	// 【关键防护】非支持角色的类型，强制忽略 role_id
	if !model.AgentTypeSupportsRole(ctx, agentType) && roleID > 0 {
		slog.InfoContext(r.Context(), "[CreateInstance] 非角色支持类型，忽略 role_id",
			"agent_type", agentType, "role_id", roleID)
		roleID = 0
	}

	// 角色版本号：实例创建时同步写入 distributed_role_version，便于后续判断"已下发版本"。
	// 非 0 角色查不到 version 视为脏数据，仍然以空串记录，等管理员手动 distribute 时再更新。
	var roleVersion string
	if roleID > 0 {
		var role model.OpenClawRole
		if err := model.DB(r.Context()).Select("version").First(&role, roleID).Error; err == nil {
			roleVersion = role.Version
		} else {
			log.Warn("[CreateInstance] 加载角色版本号失败，distributed_role_version 留空", "role_id", roleID, "error", err)
		}
	}

	// 根据 agent_type 获取对应的启用镜像（提前校验，避免创建占位记录后再失败）
	enabledImage, err := model.GetEnabledImageByType(r.Context(), agentType)
	if err != nil {
		slog.ErrorContext(r.Context(), "[CreateInstance] 查询启用镜像失败",
			"agent_type", agentType, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
		return
	}
	if enabledImage == nil {
		typeName := model.GetAgentTypeDisplayName(ctx, agentType)
		slog.WarnContext(r.Context(), "[CreateInstance] 未找到该类型的启用镜像",
			"agent_type", agentType)
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgNoImageForTypeContactAdmin, typeName))
		return
	}

	// 【隐藏参数】仅供内部测试人员使用，不在外部 API 文档中暴露。
	// 仅当部署所在腾讯云账号 UIN 命中白名单时，才允许通过 image_id 指定低版本/特定镜像，
	// 用于升级流程等需要"先用低版本创建、再触发升级"的场景。
	// 安全约束：
	//   1. 部署 UIN 必须严格匹配白名单，否则参数被静默忽略（行为与未传参完全一致，不影响存量逻辑）；
	//   2. 指定的 image_id 必须存在于本部署的 ai_images 表中；
	//   3. 指定镜像的 agent_type 必须与请求的 agent_type 一致，避免跨类型穿越权限；
	//   4. 命中后会将 enabledImage 替换为指定镜像，agent_version 也随之变更。
	if rawOverrideImageID := strings.TrimSpace(r.FormValue("image_id")); rawOverrideImageID != "" {
		isInternal, err := IsInternalAccount(r.Context())
		if err != nil {
			// 判断异常时按"非内部账号"降级处理，行为与未传 image_id 完全一致，不影响存量逻辑。
			log.Warn("[CreateInstance] 内部账号判定失败，按非白名单部署降级",
				"user_id", user.ID, "error", err)
			isInternal = false
		}
		if !isInternal {
			// 非白名单部署：静默忽略该隐藏参数，等同于未传，保持向后兼容
			log.Info("[CreateInstance] 忽略 image_id 隐藏参数（非白名单部署）",
				"user_id", user.ID, "deployed_uin", hcommon.CVMUinFromCtx(r.Context()))
		} else {
			var overrideImg model.AIImage
			if err := model.DB(r.Context()).Where("image_id = ?", rawOverrideImageID).First(&overrideImg).Error; err != nil {
				log.Warn("[CreateInstance] 隐藏参数 image_id 指定的镜像不存在",
					"user_id", user.ID, "image_id", rawOverrideImageID, "error", err)
				writeError(w, r, http.StatusBadRequest,
					hcommon.I18nError(i18n.MsgImageNotFoundByID, rawOverrideImageID))
				return
			}
			// 类型必须一致，避免用 hermes 镜像创建 openclaw 类型实例（权限/能力矩阵会错乱）
			if model.NormalizeAgentType(overrideImg.AgentType) != agentType {
				log.Warn("[CreateInstance] 隐藏参数 image_id 与 agent_type 不匹配",
					"user_id", user.ID, "image_id", rawOverrideImageID,
					"image_agent_type", overrideImg.AgentType, "request_agent_type", agentType)
				writeError(w, r, http.StatusBadRequest,
					hcommon.I18nError(i18n.MsgImageAgentTypeMismatch))
				return
			}
			log.Warn("[CreateInstance] 命中内部测试隐藏参数 image_id，覆盖默认启用镜像",
				"user_id", user.ID, "username", user.Username,
				"original_image_id", enabledImage.ImageId, "override_image_id", overrideImg.ImageId,
				"override_agent_version", overrideImg.AgentVersion)
			enabledImage = &overrideImg
		}
	}

	// 从镜像获取 agent_version
	agentVersion := enabledImage.AgentVersion
	if options.Presets != nil && len(options.Presets.Models) > 1 {
		if fallbackErr := modelFallbackSupportError(ctx, agentType, agentVersion); fallbackErr != nil {
			writeError(w, r, http.StatusConflict, fallbackErr)
			return
		}
	}

	config := model.GetSiteConfig(r.Context())
	userData := strings.TrimSpace(r.FormValue("user_data"))
	if userData != "" {
		if !config.UserDataEnabled {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgUserDataDisabled))
			return
		}
		if err := validateUserData(userData); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 提前生成 ProxyToken，避免占位记录的 proxy_token 为空字符串导致唯一索引冲突
	proxyToken, err := model.GenerateProxyToken()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGenerateProxyTokenFailed))
		return
	}

	// 使用数据库事务 + 占位记录防止并发创建超出配额（支持多副本部署）
	var placeholderInstance model.Instance
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 锁住 user 行，序列化同一用户的并发创建请求（MySQL 生效，SQLite 静默忽略）
		var lockedUser model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedUser, user.ID).Error; err != nil {
			return hcommon.I18nError(i18n.MsgLockUserRecordFailed).WithDetail(err.Error())
		}

		// 实例配额检查：按 agent 选的 group_id 策略校验该组下的实例数
		// 注：排除 current_operation='delete' 的实例，删除中的实例应立即释放配额，
		// 避免"删除后马上创建"被误判为配额超限。
		if groupID > 0 {
			// 分组实例的 fallback 用全局默认值（site_config），而非用户个人配额
			siteConfig := model.GetSiteConfig(r.Context())
			groupInstanceQuota := usergroup.ResolvePolicyIntForGroup(r.Context(), usergroup.PolicyKeyInstanceQuota, groupID, siteConfig.DefaultInstanceQuota)
			var groupCount int64
			if err := tx.Model(&model.Instance{}).Where("user_id = ? AND group_id = ? AND is_doctor_node = ? AND current_operation != ? AND source != ?", user.ID, groupID, false, model.OpDelete, model.InstanceSourceLocal).Count(&groupCount).Error; err != nil {
				return hcommon.I18nError(i18n.MsgQueryInstanceCountFailed).WithDetail(err.Error())
			}
			if groupCount >= int64(groupInstanceQuota) {
				return errGroupQuotaExceeded
			}
		} else {
			// 原逻辑：全局配额
			var count int64
			if err := tx.Model(&model.Instance{}).Where("user_id = ? AND is_doctor_node = ? AND current_operation != ? AND source != ?", user.ID, false, model.OpDelete, model.InstanceSourceLocal).Count(&count).Error; err != nil {
				return hcommon.I18nError(i18n.MsgQueryInstanceCountFailed).WithDetail(err.Error())
			}
			if count >= int64(lockedUser.InstanceQuota) {
				return errQuotaExceeded
			}
		}
		// 创建占位记录，先占住配额
		placeholderInstance = model.Instance{
			Name:                   name,
			UserID:                 user.ID,
			ProxyToken:             &proxyToken,
			RoleID:                 roleID,
			DistributedRoleVersion: roleVersion,
			GroupID:                groupID,
			AgentType:              agentType,
			AgentVersion:           agentVersion,
			UserData:               userData,
		}
		if err := tx.Create(&placeholderInstance).Error; err != nil {
			return hcommon.I18nError(i18n.MsgCreatePlaceholderRecordFailed).WithDetail(err.Error())
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errQuotaExceeded) || errors.Is(txErr, errGroupQuotaExceeded) {
			log.Warn("[CreateInstance] 实例配额已达上限", "user_id", user.ID, "error", txErr)
			writeError(w, r, http.StatusForbidden, txErr.(*hcommon.RichError))
		} else {
			log.Error("[CreateInstance] 占位记录事务失败", "user_id", user.ID, "error", txErr)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgOperationFailed))
		}
		return
	}
	log.Info("[CreateInstance] 占位记录创建成功", "user_id", user.ID, "placeholder_id", placeholderInstance.ID, "agent_type", agentType)

	// 后续流程失败时自动清理占位记录和 SMH 空间
	success := false
	defer func() {
		if !success {
			log.Warn("[CreateInstance] 创建失败，清理占位记录", "user_id", user.ID, "placeholder_id", placeholderInstance.ID)
			// 用脱离取消信号的 context 删除：避免请求已取消（客户端断开/超时）时，
			// 清理操作本身也被 context canceled，导致占位记录残留成永久"创建中"孤儿。
			model.DB(hcommon.DetachContext(r.Context())).Delete(&placeholderInstance)
		}
	}()

	if config.CVMTemplate == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCVMConfigIncomplete))
		return
	}
	if CVMRegion == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCVMRegionNotConfigured))
		return
	}

	// Deserialize template JSON into RunInstancesRequest
	request := cvm.NewRunInstancesRequest()
	if err := json.Unmarshal([]byte(config.CVMTemplate), request); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCVMTemplateConfigError))
		return
	}

	request, err = applyResourceOverlay(r.Context(), resourceOverlayInput{
		BaseRequest: request,
		GroupID:     groupID,
		UserConfig:  r.FormValue("resource_config"),
		DiskType:    selectedDiskType,
		ImageSize:   enabledImage.ImageSize,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errResourceOverlayBadRequest) {
			status = http.StatusBadRequest
		}
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	request.ImageId = common.StringPtr(enabledImage.ImageId)

	// Determine zone, VPC, and subnet
	var zone, vpcId, subnetId string

	// 根据用户选择的分组解析 VPC 配置（分组策略 → 祖先策略 → 预设策略）
	resolvedVpcId, resolvedSubnetMap := usergroup.ResolveVpcConfig(r.Context(), groupID, config.VpcId, config.GetSubnetMap())

	// 过滤出真正有子网的 zone（key 存在但 slice 可能为空）
	zonesWithSubnet := make([]string, 0, len(resolvedSubnetMap))
	for z, sids := range resolvedSubnetMap {
		if len(sids) > 0 {
			zonesWithSubnet = append(zonesWithSubnet, z)
		}
	}
	if resolvedVpcId != "" && len(zonesWithSubnet) > 0 {
		log.Info("Using resolved VPC/subnet config",
			"group_id", groupID, "vpc_id", resolvedVpcId)
		// 校验 VPC 和子网在云端是否仍然存在
		if err := validateGlobalVpcAndSubnetsFn(r.Context(), resolvedVpcId, resolvedSubnetMap); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		// Pick a random zone from configured subnets
		zone = zonesWithSubnet[rand.Intn(len(zonesWithSubnet))]
		vpcId = resolvedVpcId
		candidates := resolvedSubnetMap[zone]
		if len(candidates) == 1 {
			// 单子网直接选中（避免额外云 API 调用）
			subnetId = candidates[0]
		} else {
			// 多子网按剩余 IP 加权挑选，跳过已满子网
			vpcClient, err := newVpcClient(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
				return
			}
			picked, err := pickSubnetByAvailableIP(vpcClient, candidates)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
			subnetId = picked
		}
	} else if resolvedVpcId != "" {
		// VPC configured but no subnets
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgVPCConfiguredWithoutSubnet))
		return
		// 暂不删除，后续可能需要支持用户级VPC，待产品规划
		// } else {
		// 	// Fallback: per-user VPC/subnet
		// 	var err error
		// 	zone, err = randomZone()
		// 	if err != nil {
		// 		writeError(w, r, http.StatusInternalServerError, err)
		// 		return
		// 	}

		// 	// Ensure user has a VPC
		// 	if err := ensureUserVpc(user, config); err != nil {
		// 		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError("创建用户 VPC 失败", err))
		// 		return
		// 	}

		// 	// Ensure user has a subnet for this zone
		// 	subnetId, err = ensureUserSubnet(user, config, zone)
		// 	if err != nil {
		// 		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError("创建用户子网失败", err))
		// 		return
		// 	}
		// 	vpcId = user.VpcId
	} else {
		log.Info("Using shared default VPC/subnet")
		// Fallback: 使用共享默认 VPC (clawpro/default-vpc) + 按可用区自动创建子网
		defaultVpcId, defaultSubnetMap, err := ensureDefaultVpcAndSubnets(r.Context(), &config)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateDefaultVPCFailed))
			return
		}

		log.Info("Default vpcId and subnet map", "vpc_id", defaultVpcId, "subnet map", defaultSubnetMap)

		if len(defaultSubnetMap) == 0 {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDefaultVPCNoUsableSubnet))
			return
		}

		// 从已有子网中随机选一个可用区，再从该可用区的子网列表中随机选一个子网
		zones := make([]string, 0, len(defaultSubnetMap))
		for z, sids := range defaultSubnetMap {
			if len(sids) > 0 {
				zones = append(zones, z)
			}
		}
		if len(zones) == 0 {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDefaultVPCNoUsableSubnet))
			return
		}
		zone = zones[rand.Intn(len(zones))]
		vpcId = defaultVpcId
		subnets := defaultSubnetMap[zone]
		subnetId = subnets[rand.Intn(len(subnets))]
	}

	if request.Placement == nil {
		request.Placement = &cvm.Placement{}
	}
	request.Placement.Zone = common.StringPtr(zone)
	request.InstanceName = common.StringPtr(name)
	request.InstanceCount = common.Int64Ptr(1)

	// 临时方式，待创建cam:PassRole权限上生产后，改为：CVM_QCSLinkedRoleInClawProAgent
	tmpRoleName := os.Getenv("AGENT_CAM_ROLE_NAME")
	if tmpRoleName != "" {
		log.Info("set agent cam role name", "role", tmpRoleName)
		request.CamRoleName = common.StringPtr(tmpRoleName)
	} else {
		request.CamRoleName = common.StringPtr("")
	}

	// Override VPC settings
	request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{
		VpcId:    common.StringPtr(vpcId),
		SubnetId: common.StringPtr(subnetId),
	}

	// 创建前预验证：确认最终实例规格在目标可用区可售；必须先于 SG 选择/扩容。
	chargeType := ""
	if request.InstanceChargeType != nil {
		chargeType = strings.TrimSpace(*request.InstanceChargeType)
	}
	instanceTypeName := ""
	if request.InstanceType != nil {
		instanceTypeName = strings.TrimSpace(*request.InstanceType)
	}
	if err := validateCreateResourceConfigFn(r.Context(), zone, chargeType, instanceTypeName); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 选安全组：从 RuleSet 对应的 SG 池挑一个 cvm_count 最低的 ACTIVE；必要时扩容，撞上限走 buffer
	rsForSelect, rsErr := GetDefaultRuleSet(r.Context())
	if rsErr != nil {
		log.Error("获取默认 RuleSet 失败", "error", rsErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgClawProSGNotConfigured))
		return
	}
	selectedSG, usedBuffer, sgErr := selectSGForNewInstanceFn(ctx, model.CurrentIdentifier(ctx), rsForSelect.ID)
	if sgErr != nil {
		if errors.Is(sgErr, ErrNoBaseConfigured) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgClawProSGNotConfigured))
		} else if errors.Is(sgErr, ErrPoolAtHardLimit) {
			log.Error("SG 池撞硬上限，无法创建实例", "error", sgErr)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSGCapacityFull))
		} else {
			log.Error("选安全组失败", "error", sgErr)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(sgErr, i18n.MsgOperationFailed))
		}
		return
	}
	if usedBuffer {
		log.Warn("[sg-pool] 实例绑定走到 buffer 区", "sg_id", selectedSG)
	}
	// 防御：SelectSGForNewInstance 正常路径不应返回空 SG，但万一上游逻辑 / 极端时序
	// 把 selectedSG 落成空串，会让 placeholder 落库即被记成 sg='' 孤儿（历史 bug 根因）。
	// 这里硬性拦下，让用户重试比留一台无主机器更可控。
	if selectedSG == "" {
		log.Error("[CreateInstance] selectedSG 为空，拒绝创建以避免孤儿实例",
			"user_id", user.ID, "placeholder_id", placeholderInstance.ID)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSGAllocationFailed))
		return
	}
	request.SecurityGroupIds = common.StringPtrs([]string{selectedSG})

	// 先把 SG 落到 placeholder：哪怕后续 RunInstances 失败 / Updates 失败，
	// SG Guardian 的 migrate / drainOrphan 也能按正确的 SG 把这条记录纳入管理。
	if err := model.DB(r.Context()).Model(&placeholderInstance).
		UpdateColumn("security_group_id", selectedSG).Error; err != nil {
		log.Error("[CreateInstance] 占位记录预写 SG 失败",
			"user_id", user.ID, "placeholder_id", placeholderInstance.ID, "sg_id", selectedSG, "error", err)
		// 不立即 return：SG Guardian 仍能兜底，让 CVM 创建继续走，避免误伤用户体验。
	}

	// Set UserData from init.sh template and optional user-provided UserData.
	var systemUserDataConfig *initUserDataConfig
	if config.SkillHub != "" {
		systemUserDataConfig = &initUserDataConfig{
			SkillHub:    config.SkillHub,
			RuntimeUser: "", // 新实例尚无 RuntimeUser，init.sh 将在运行时自行检测
			AgentType:   agentType,
		}
	}
	mergedUserData, err := buildUserData(ctx, systemUserDataConfig, userData)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if mergedUserData != "" {
		request.UserData = common.StringPtr(mergedUserData)
	}

	// 强制开启增强服务（自动化助手）
	request.EnhancedService = &cvm.EnhancedService{
		AutomationService: &cvm.RunAutomationServiceEnabled{
			Enabled: common.BoolPtr(true),
		},
	}

	// 注入默认标签（新 tags 表优先；迁移前 fallback 到 SiteConfig.DefaultTags）
	defaultTags, tgErr := model.ResolveTagsForGroup(r.Context(), groupID, config.DefaultTags)
	if tgErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(tgErr, i18n.MsgQueryDefaultTagsFailed))
		return
	}
	if tagCount := applyCreateInstanceTags(request, customTags, defaultTags); tagCount > 0 {
		log.Info("[CreateInstance] 注入实例标签",
			"count", tagCount,
			"custom_count", len(customTags),
			"default_count", len(defaultTags),
		)
	}

	// Build client
	client, err := NewCVMClient(ctx)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed))
		return
	}

	response, cvmErr := client.RunInstances(request)
	if cvmErr != nil {
		systemDiskSize := int64(0)
		if request.SystemDisk != nil && request.SystemDisk.DiskSize != nil {
			systemDiskSize = *request.SystemDisk.DiskSize
		}
		log.Error("[CreateInstance] CVM RunInstances 失败",
			"user_id", user.ID,
			"placeholder_id", placeholderInstance.ID,
			"name", name,
			"system_disk_size", systemDiskSize,
			"error", cvmErr,
		)
		writeError(w, r, cvmRunInstancesHTTPStatus(cvmErr, len(customTags) > 0), hcommon.EnsureRichErrorOrPanic(cvmRunInstancesError(cvmErr, user.Role == "admin")))
		go createErrorNotification(user.ID, 0, name, model.NotifyTypeInstanceCreateFailed, "实例创建失败", cvmErr, hcommon.DetachContext(r.Context()))
		return
	}

	var instanceId string
	if response.Response != nil && len(response.Response.InstanceIdSet) > 0 {
		instanceId = *response.Response.InstanceIdSet[0]
	}
	log.Info("[CreateInstance] CVM RunInstances 成功", "user_id", user.ID, "placeholder_id", placeholderInstance.ID, "cvm_id", instanceId, "agent_type", agentType)

	// CVM 已创建成功，后续 DB 写操作是"必须完成"的关键路径，不能因客户端断连/超时而中断。
	// 用 DetachContext 脱离请求取消信号，同时保留租户快照和链路追踪字段，叠加 30s 超时兜底。
	writeCtx, writeCancel := context.WithTimeout(hcommon.DetachContext(r.Context()), 60*time.Second)
	defer writeCancel()

	// 更新占位记录为正式实例（proxy_token 已在创建占位记录时生成），同时写入生命周期字段
	now := time.Now()
	instanceChargeType := cvmChargeTypePostpaidByHour
	if request.InstanceChargeType != nil && strings.TrimSpace(*request.InstanceChargeType) != "" {
		instanceChargeType = strings.TrimSpace(*request.InstanceChargeType)
	}
	createUpdates := map[string]interface{}{
		"instance_id":                  instanceId,
		"instance_charge_type":         instanceChargeType,
		"vpc_id":                       vpcId,
		"subnet_id":                    subnetId,
		"security_group_id":            selectedSG,
		"current_operation":            model.OpCreate,
		"current_operation_state":      model.OpStateProcessing,
		"current_operation_updated_at": &now,
		"last_cvm_state":               "PENDING",
		"last_known_status":            model.StatusCreating, // P3 操作即时写
		"status_synced_at":             now,                  // 防止后台覆盖
		"img_id":                       enabledImage.ImageId, // 创建时直写镜像缓存
	}
	// 创建时直写标签缓存，内容必须与本次 RunInstances 实际下发的标签一致。
	finalTagItems := createInstanceTagItemsForCache(customTags, defaultTags)
	if len(finalTagItems) > 0 {
		tagsJSON, err := json.Marshal(finalTagItems)
		if err != nil {
			log.Warn("[CreateInstance] 序列化实例标签失败", "error", err)
		} else {
			createUpdates["cvm_tags_json"] = string(tagsJSON)
		}
	}
	updateRes := model.DB(writeCtx).Model(&placeholderInstance).Updates(createUpdates)
	if updateRes.Error != nil {
		// CVM 已创建成功但 DB 写回失败：硬性补救一次只更新关键字段（instance_id / sg）
		// SG Guardian 后续可以用 instance_id 找到这台 CVM 并对账。
		log.Error("[CreateInstance] 更新占位记录失败，尝试只补关键字段",
			"placeholder_id", placeholderInstance.ID, "cvm_id", instanceId, "sg_id", selectedSG, "error", updateRes.Error)
		fallbackErr := model.DB(writeCtx).Model(&placeholderInstance).Updates(map[string]interface{}{
			"instance_id":       instanceId,
			"security_group_id": selectedSG,
		}).Error
		if fallbackErr != nil {
			// 兜底也失败：instance_id 未落库，defer 清理会删除占位记录。
			// CVM 将成为孤儿，SG Guardian 后续对账可发现并清理。
			log.Error("[CreateInstance] 兜底回写也失败，将清理占位记录",
				"placeholder_id", placeholderInstance.ID, "cvm_id", instanceId, "error", fallbackErr)
			return
		}
	}
	placeholderInstance.InstanceId = instanceId
	success = true

	// 维护 SG 池 cvm_count 缓存
	if err := model.IncrementSGCVMCount(writeCtx, selectedSG); err != nil {
		log.Warn("[sg-pool] 绑 SG 后 cvm_count +1 失败", "sg_id", selectedSG, "error", err)
	}

	// 补齐 memory_tdai plugin 行，后台轮询会自动按开关状态处理
	// final §2：仅对支持 Memory 的类型（OpenClaw）补行，避免 Hermes/ACE 后台轮询刷屏
	if model.AgentTypeSupportsMemory(ctx, agentType) {
		model.EnsureMemoryTDAIPluginRow(writeCtx, instanceId)
	}

	// 按默认记忆计划配置，增量实例自动提交切换任务（仅对支持记忆的 agent_type 生效）。
	// applyDefaultMemoryPlanForInstance 内部会：
	//   1) 校验 agent_type 是否支持记忆；
	//   2) 通过 resolveMemoryPlanTransition 将小写 off/free/pro 转成大写 plan 常量再写入 DB
	//      （避免 desired_plan 大小写不一致导致后续比较失效）。
	applyDefaultMemoryPlanForInstance(writeCtx, instanceId, config)

	// 异步检查实例绑定的安全组是否有出站规则，缺失则补全全放通规则。
	// 如果已有 default RuleSet（managed_sg_pool 接管），则跳过——Pool 模式下出站规则由 RuleSet 统一管理。
	if instanceId != "" {
		if _, rsErr := GetDefaultRuleSet(writeCtx); rsErr != nil {
			go ensureInstanceEgressRules(hcommon.DetachContext(ctx), instanceId)
		}
	}

	extraSkills := []createSkillPreset(nil)
	hasInitialModels := options.Presets != nil && len(options.Presets.Models) > 0

	// Admin-provided initial models replace the site default. Without them the
	// existing default-model behavior remains unchanged.
	if hasInitialModels {
		modelIDs := make([]uint, 0, len(options.Presets.Models))
		for _, aiModel := range options.Presets.Models {
			modelIDs = append(modelIDs, aiModel.ID)
		}
		trackGatewayRestartTask(placeholderInstance.ID)
		go func(ctx context.Context, pk uint, ids []uint) {
			defer recoverCreatePresetPanic("models", pk)
			defer untrackGatewayRestartTask(pk)
			applyInitialModels(ctx, pk, ids)
		}(hcommon.DetachContext(r.Context()), placeholderInstance.ID, modelIDs)
	} else if config.DefaultModelID > 0 && model.AgentTypeSupportsDefaultModelInjection(ctx, agentType) {
		var defaultModel model.AIModel
		if model.DB(writeCtx).Where("id = ? AND enabled = ?", config.DefaultModelID, true).First(&defaultModel).Error == nil {
			var existing model.InstanceModel
			txErr := model.DB(writeCtx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&placeholderInstance).Update("ai_model_id", defaultModel.ID).Error; err != nil {
					return err
				}
				return tx.Where("instance_id = ? AND ai_model_id = ? AND custom_model_id = ?",
					placeholderInstance.ID, defaultModel.ID, "").
					FirstOrCreate(&existing, model.InstanceModel{
						InstanceID: placeholderInstance.ID,
						AIModelID:  defaultModel.ID,
						Role:       model.ModelRolePrimary,
						SortOrder:  1,
					}).Error
			})
			if txErr != nil {
				slog.Error("[CreateInstance] 创建默认模型绑定记录失败，跳过默认模型注入",
					"instance_pk", placeholderInstance.ID, "model_id", defaultModel.ID, "error", txErr)
			} else {
				trackGatewayRestartTask(placeholderInstance.ID)
				go func(ctx context.Context, pk uint, modelID uint) {
					defer untrackGatewayRestartTask(pk)
					injectDefaultModel(ctx, pk, modelID)
				}(hcommon.DetachContext(r.Context()), placeholderInstance.ID, defaultModel.ID)
			}
		}
	}

	// Create-time channels are best-effort, exactly like a user invoking
	// set-channel after the Agent is ready. Credentials live only in this
	// detached goroutine and are never persisted or retried.
	if options.Presets != nil && len(options.Presets.Channels) > 0 {
		trackGatewayRestartTask(placeholderInstance.ID)
		externalBaseURL := requestExternalBaseURL(r)
		go func(ctx context.Context, baseURL string, pk uint, presets []manualChannelPreset) {
			defer recoverCreatePresetPanic("channels", pk)
			defer untrackGatewayRestartTask(pk)
			applyChannelPresetsAsync(ctx, baseURL, pk, presets)
		}(
			hcommon.DetachContext(r.Context()),
			externalBaseURL,
			placeholderInstance.ID,
			options.Presets.Channels,
		)
	}

	if options.Presets != nil {
		extraSkills = options.Presets.Skills
	}
	// Existing role/bundle skills keep their established installation records.
	createSkillInstallTasks(writeCtx, placeholderInstance.ID, roleID, groupID)
	trackGatewayRestartTask(placeholderInstance.ID)
	go func(ctx context.Context, pk uint, cvmID, at string) {
		defer untrackGatewayRestartTask(pk)
		installSkillsAsync(ctx, pk, cvmID, at, waitModeCreate)
	}(hcommon.DetachContext(r.Context()), placeholderInstance.ID, instanceId, agentType)

	// Admin-selected public/enterprise skills follow the same one-shot paths as
	// the user add-skill endpoint and are not persisted or retried.
	if len(extraSkills) > 0 {
		trackGatewayRestartTask(placeholderInstance.ID)
		go func(ctx context.Context, pk uint, skills []createSkillPreset) {
			defer recoverCreatePresetPanic("skills", pk)
			defer untrackGatewayRestartTask(pk)
			applyExtraSkillsAsync(ctx, pk, skills)
		}(
			hcommon.DetachContext(r.Context()),
			placeholderInstance.ID,
			extraSkills,
		)
	}

	// 创建插件安装任务并异步安装（角色插件 + 全局插件包，按用户可见性过滤去重）
	createPluginInstallTasks(writeCtx, placeholderInstance.ID, roleID)
	trackGatewayRestartTask(placeholderInstance.ID)
	go func(ctx context.Context, pk uint, cvmID string) {
		defer untrackGatewayRestartTask(pk)
		installPluginsAsync(ctx, pk, cvmID, waitModeCreate)
	}(hcommon.DetachContext(r.Context()), placeholderInstance.ID, instanceId)

	// 若有角色绑定，异步等待 RuntimeUser 就绪后立即下发 Soul；失败由周期任务兜底重试
	if roleID > 0 {
		trackGatewayRestartTask(placeholderInstance.ID)
		go func(ctx context.Context, pk uint, cvmID string) {
			defer untrackGatewayRestartTask(pk)
			setInstanceSoulWhenReady(ctx, pk, cvmID)
		}(hcommon.DetachContext(r.Context()), placeholderInstance.ID, instanceId)
	}

	// 若配置了自动创建个人空间，且实例类型支持网盘，异步创建空间并等 CVM 就绪后初始化环境
	// final §2：三端 SMH=true，但 init_smh_env 脚本目前仅 openclaw 版；
	// 为避免 Hermes/ACE 实例被自动 provision 后脚本执行失败，此处加脚本就绪校验。
	// 未来 Hermes/ACE 的 init_smh_env 脚本落盘后，只需在 scriptResolveTable 注册即可自动启用。
	smhAutoProvision := usergroup.ResolvePolicyBoolForGroup(ctx, usergroup.PolicyKeySMHAutoProvision, groupID, config.SMHAutoProvisionOnCreate)
	if smhAutoProvision && model.AgentTypeSupportsSMH(ctx, agentType) {
		if _, resolveErr := ResolveScript(ctx, "init_smh_env", agentType); resolveErr != nil {
			log.Warn("[SMH] 自动初始化脚本未就绪，跳过自动 provision",
				"agent_type", agentType, "instance_id", instanceId, "error", resolveErr)
		} else {
			trackGatewayRestartTask(placeholderInstance.ID)
			go func(ctx context.Context, inst model.Instance) {
				defer func() {
					untrackGatewayRestartTask(inst.ID)
					if r := recover(); r != nil {
						log.Error("[SMH] 初始化个人空间环境 panic", "instance_id", inst.InstanceId, "error", r)
					}
				}()
				_, err := CreatePersonalSpaceForInstance(ctx, &inst, user)
				if err != nil {
					log.Error("[SMH] 创建个人空间失败", "instance_id", inst.InstanceId, "error", err)
					return
				}
				syncSMHEnvWhenReady(ctx, inst)
			}(hcommon.DetachContext(r.Context()), placeholderInstance)
		}
	}

	// 新实例继承 CLS 开启状态——如果实例的分组命中 CLS 采集范围，标记为待安装
	if instanceId != "" {
		if err := inheritCLSScopeForNewInstance(writeCtx, groupID, instanceId); err != nil {
			log.Warn("[CLS Scope] 新实例继承 CLS 范围失败", "group_id", groupID, "instance_id", instanceId, "error", err)
		}
	}

	// 异步 approve device（等 TAT 就绪后执行）
	// final：approve_device.sh 调 `openclaw devices approve`，仅 OpenClaw 有此命令；
	// Hermes/ACE 跳过（与 lightclaw.go:149 同步路径一致）
	// 兼容 openclaw 的自定义类型走相同路径。
	if model.GetAgentRuntimeType(ctx, agentType) == model.AgentTypeOpenClaw {
		go approveDeviceAsync(hcommon.DetachContext(r.Context()), placeholderInstance.ID, instanceId, "")
	}

	log.Info("[CreateInstance] 创建流程完成", "user_id", user.ID, "instance_id", placeholderInstance.ID, "cvm_id", instanceId)
	return createInstanceResult{InstanceID: instanceId}, true
}

// NewCVMClient creates a CVM client from SiteConfig credentials.
//
// 以 var 形式暴露，便于单元测试注入 mock（测试中替换为返回
// (nil, error) 以覆盖"创建 CVM 客户端失败"错误分支）。
// 生产行为与原 func 完全一致。
var NewCVMClient = func(ctx context.Context) (*cvm.Client, error) {
	credential, rerr := getCredential(ctx)
	if rerr != nil {
		return nil, rerr
	}
	cpf := profile.NewClientProfile()
	client, err := cvm.NewClient(credential, CVMRegion, cpf)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed)
	}
	return client, nil
}

type initUserDataConfig struct {
	SkillHub    string
	RuntimeUser string
	AgentType   string
}

// CVM UserData 限制：base64 编码后不能超过 16KB（RunInstances API 文档）。
// 用户提交的 user_data 自身已是 base64；拆分后的系统 init 脚本每份约 1.3KB，
// 加上 multipart 头开销，12KB 的 base64 用户输入合并后通常仍留有富余。
const (
	maxCVMUserDataEncodedSize = 16 << 10
	maxUserDataInputSize      = 12 << 10
)

// renderUserData loads init.sh, renders it with config, and returns base64-encoded content.
func renderUserData(ctx context.Context, config initUserDataConfig) (string, error) {
	data, err := renderUserDataBytes(ctx, config)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func renderUserDataBytes(ctx context.Context, config initUserDataConfig) ([]byte, error) {
	scriptName, rerr := ResolveScript(ctx, "init", config.AgentType)
	if rerr != nil {
		// 自定义类型可能没有 init 脚本（无 compatible_with 或兼容目标无对应脚本），
		// 不阻断实例创建，返回空表示无系统 UserData。
		slog.Info("[renderUserData] 无可用 init 脚本，跳过系统 UserData",
			"agent_type", config.AgentType, "reason", rerr.Error())
		return nil, nil
	}
	scriptContent, loadErr := LoadScript(scriptName)
	if loadErr != nil {
		return nil, hcommon.I18nError(i18n.MsgLoadScriptFailed, scriptName).WithDetail(loadErr.Error())
	}
	tmpl, err := template.New(scriptName).Parse(scriptContent)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgParseTemplateFailed, scriptName).WithDetail(err.Error())
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, hcommon.I18nError(i18n.MsgRenderTemplateFailed, scriptName).WithDetail(err.Error())
	}
	return buf.Bytes(), nil
}

// validateUserData 校验 create 入参的 user_data：必须是合法 base64，且字符串长度不超过保守上限。
func validateUserData(userData string) error {
	if len(userData) > maxUserDataInputSize {
		return hcommon.I18nError(i18n.MsgUserDataExceedSize, maxUserDataInputSize>>10)
	}
	if _, err := base64.StdEncoding.DecodeString(userData); err != nil {
		return hcommon.I18nError(i18n.MsgUserDataInvalidBase64)
	}
	return nil
}

func buildUserData(ctx context.Context, systemConfig *initUserDataConfig, userData string) (string, error) {
	var systemData []byte
	if systemConfig != nil {
		var err error
		systemData, err = renderUserDataBytes(ctx, *systemConfig)
		if err != nil {
			return "", err
		}
	}
	var userDataBytes []byte
	if userData != "" {
		decoded, err := base64.StdEncoding.DecodeString(userData)
		if err != nil {
			return "", hcommon.I18nError(i18n.MsgUserDataInvalidBase64)
		}
		userDataBytes = decoded
	}

	// 判断用户 UserData 是否是 multipart 格式
	// cloud-init 的判断逻辑：忽略前导空白后以 "Content-Type: multipart/" 开头
	if len(userDataBytes) > 0 && isUserDataMultipart(userDataBytes) {
		// 用户已经是 multipart：在其第一个 boundary 前插入系统 part
		var merged []byte
		if len(systemData) > 0 {
			var err error
			merged, err = prependSystemPartToMultipart(systemData, userDataBytes)
			if err != nil {
				return "", hcommon.I18nError(i18n.MsgMergeUserDataFailed).WithDetail(err.Error())
			}
		} else {
			// 没有系统 UserData，用户 multipart 原样返回
			merged = userDataBytes
		}
		encoded := base64.StdEncoding.EncodeToString(merged)
		if len(encoded) > maxCVMUserDataEncodedSize {
			return "", hcommon.I18nError(i18n.MsgUserDataExceedCVMSize, maxCVMUserDataEncodedSize>>10)
		}
		return encoded, nil
	}

	// 非 multipart 路径：用 parts 列表组装
	parts := make([]userDataPart, 0, 2)
	if len(systemData) > 0 {
		parts = append(parts, buildSystemPart(systemData))
	}
	if len(userDataBytes) > 0 {
		parts = append(parts, buildSinglePart(
			detectUserDataContentType(userDataBytes),
			userDataBytes,
		))
	}

	var encoded string
	switch len(parts) {
	case 0:
		return "", nil
	case 1:
		// 单 part 直接传原始 body，不包 multipart：减小体积、与老行为一致。
		encoded = base64.StdEncoding.EncodeToString(parts[0].Body)
	default:
		encoded = base64.StdEncoding.EncodeToString(serializeMultipart(parts))
	}
	if len(encoded) > maxCVMUserDataEncodedSize {
		return "", hcommon.I18nError(i18n.MsgUserDataExceedCVMSize, maxCVMUserDataEncodedSize>>10)
	}
	return encoded, nil
}

// userDataPart 是要写进外层 multipart 的一个 part。
//
// Headers 是完整的 header 字节（不含起始 boundary 行和 header/body 之间的空行）。
// 对“我们自己生成的 part”（例如系统 init.sh、或者用户提交的单段非 multipart
// UserData 被我们代劳补头），Headers 由 buildSystemPart / buildSinglePart 构造。
// 对“从用户 multipart 中解析出来的 part”（PR2 引入），Headers 必须原样保留，
// 以避免破坏 charset / Content-Transfer-Encoding / Content-Disposition 等
// 描述 body 本身的语义。
type userDataPart struct {
	Headers []byte
	Body    []byte
}

const userDataBoundary = "==clawpro-userdata-boundary=="

// buildSystemPart 为系统 init.sh 构造 part。内容确定是 UTF-8 shell 脚本，
// 因此 charset / CTE 由我们声明。
func buildSystemPart(body []byte) userDataPart {
	return userDataPart{
		Headers: []byte("Content-Type: text/x-shellscript; charset=\"utf-8\"\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n"),
		Body: body,
	}
}

// buildSinglePart 为“用户提交的单段非 multipart UserData”构造 part。
// 原始请求里本来就没有 MIME 头，这些头是我们代劳补齐的，按文本 UTF-8 处理。
// 用户自带 multipart 的情况不走这里（PR2 解析后原样保留用户 part 的 Headers）。
func buildSinglePart(contentType string, body []byte) userDataPart {
	return userDataPart{
		Headers: []byte(fmt.Sprintf(
			"Content-Type: %s; charset=\"utf-8\"\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Transfer-Encoding: 8bit\r\n",
			contentType,
		)),
		Body: body,
	}
}

// serializeMultipart 按 RFC 2046 把一组 part 序列化成 multipart/mixed。
// boundary / 分隔行使用 CRLF；part body 按原字节写入，不改 body 内部换行。
func serializeMultipart(parts []userDataPart) []byte {
	var buf bytes.Buffer
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"")
	buf.WriteString(userDataBoundary)
	buf.WriteString("\"\r\nMIME-Version: 1.0\r\n\r\n")
	for _, p := range parts {
		writePart(&buf, userDataBoundary, p)
	}
	buf.WriteString("--")
	buf.WriteString(userDataBoundary)
	buf.WriteString("--\r\n")
	return buf.Bytes()
}

func writePart(buf *bytes.Buffer, boundary string, p userDataPart) {
	buf.WriteString("--")
	buf.WriteString(boundary)
	buf.WriteString("\r\n")
	buf.Write(p.Headers)
	if !bytes.HasSuffix(p.Headers, []byte("\r\n")) {
		buf.WriteString("\r\n")
	}
	// 空行隔开 header 和 body（RFC 要求 CRLF）
	buf.WriteString("\r\n")
	buf.Write(p.Body)
	// 保证 body 后至少一个换行，便于紧跟下一个 boundary 行
	if !bytes.HasSuffix(p.Body, []byte("\n")) {
		buf.WriteString("\r\n")
	}
}

func detectUserDataContentType(data []byte) string {
	trimmed := strings.TrimLeft(string(data), " \t\r\n")
	switch {
	case strings.HasPrefix(trimmed, "#cloud-config"):
		return "text/cloud-config"
	case strings.HasPrefix(trimmed, "#cloud-boothook"):
		return "text/cloud-boothook"
	case strings.HasPrefix(trimmed, "#include"):
		return "text/x-include-url"
	default:
		return "text/x-shellscript"
	}
}

// isUserDataMultipart 判断用户提交的 UserData 是否是 MIME multipart 格式。
//
// cloud-init 的判断逻辑：忽略前导空白后，如果以 "Content-Type: multipart/"
// 开头（大小写不敏感），就按 multipart/mixed 解析。
// 参考：https://docs.cloud-init.io/en/latest/explanation/format.html
func isUserDataMultipart(data []byte) bool {
	trimmed := strings.TrimLeft(string(data), " \t\r\n")
	return strings.HasPrefix(strings.ToLower(trimmed), "content-type: multipart/")
}

// extractBoundary 从 multipart UserData 明文中提取 boundary 参数。
//
// 输入是用户 UserData 解码后的原始字节（可能有前导空白）。
// 第一个有效行应是 "Content-Type: multipart/mixed; boundary=..."
func extractBoundary(data []byte) (string, error) {
	trimmed := strings.TrimLeft(string(data), " \t\r\n")
	// 取第一行
	firstLine := trimmed
	if idx := strings.IndexAny(firstLine, "\r\n"); idx > 0 {
		firstLine = firstLine[:idx]
	}

	// 用 mime.ParseMediaType 解析 Content-Type 头的值部分
	headerValue := strings.TrimSpace(strings.TrimPrefix(firstLine, "Content-Type:"))
	headerValue = strings.TrimSpace(strings.TrimPrefix(headerValue, "content-type:"))
	_, params, err := mime.ParseMediaType(headerValue)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgParseContentTypeFailed).WithDetail(err.Error())
	}
	boundary := params["boundary"]
	if boundary == "" {
		return "", hcommon.I18nError(i18n.MsgMultipartMissingBoundary)
	}
	return boundary, nil
}

// prependSystemPartToMultipart 在用户已有的 multipart UserData 中，
// 于第一个 boundary 之前插入系统 init.sh part。
//
// 实现方式：复用用户 multipart 的 boundary，在 MIME 头部之后、第一个
// "--boundary" 行之前，插入一个新的系统 part（同 boundary 分隔）。
// 用户原始内容从第一个 boundary 开始完全不动。
func prependSystemPartToMultipart(systemData, userData []byte) ([]byte, error) {
	boundary, err := extractBoundary(userData)
	if err != nil {
		return nil, err
	}

	boundaryMarker := "--" + boundary

	// 找到用户 multipart body 中第一个 boundary 的位置
	// MIME 格式：header 之后空行，然后第一个 "--boundary"
	firstBoundaryIdx := bytes.Index(userData, []byte(boundaryMarker))
	if firstBoundaryIdx < 0 {
		return nil, hcommon.I18nError(i18n.MsgUserDataMultipartBoundaryNotFound, boundary)
	}

	// 构造最终内容：
	// [用户原始 header + 空行（到第一个 boundary 之前的部分）]
	// + 系统 part（使用同一 boundary）
	// + [用户从第一个 boundary 到结尾]
	var buf bytes.Buffer

	// 写用户 header 部分（到第一个 boundary 之前）
	buf.Write(userData[:firstBoundaryIdx])

	// 写系统 init.sh part
	buf.WriteString(boundaryMarker)
	buf.WriteString("\r\n")
	buf.WriteString("Content-Type: text/x-shellscript; charset=\"utf-8\"\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")
	buf.Write(systemData)
	if !bytes.HasSuffix(systemData, []byte("\n")) {
		buf.WriteString("\r\n")
	}

	// 写用户原始 parts（从第一个 boundary 到末尾，完全不动）
	buf.Write(userData[firstBoundaryIdx:])

	return buf.Bytes(), nil
}

// ErrInstanceNotFound 表示实例在数据库中不存在，用于区分"参数格式错误（400）"和"资源不存在（404）"。
var ErrInstanceNotFound = hcommon.I18nError(i18n.MsgInstanceNotFound)

// instanceErrStatus 根据错误类型返回合适的 HTTP 状态码：
// 实例不存在 → 404，其他参数错误 → 400。
func instanceErrStatus(err error) int {
	if errors.Is(err, ErrInstanceNotFound) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

// findInstanceByIDOrCVMID 是所有"按主键 ID 或 CVM 实例 ID 双参数解析"的核心实现。
//
// 参数语义：
//   - id：DB 主键（uint），> 0 时优先使用，与 instanceID 同传时忽略后者；
//   - instanceID：CVM 实例 ID 字符串（如 "ins-xxx"）；
//   - userID：> 0 时附加 user_id 过滤（用户侧），== 0 时不附加（管理侧）。
//
// 错误语义：
//   - 两者均为空：返回参数错误（外层应 400）；
//   - 查询命中失败：返回 ErrInstanceNotFound（外层可借助 instanceErrStatus 映射 404）。
func findInstanceByIDOrCVMID(ctx context.Context, userID uint, id uint, instanceID string) (*model.Instance, error) {
	if id == 0 && instanceID == "" {
		return nil, hcommon.I18nError(i18n.MsgMissingIDOrInstanceID)
	}

	q := model.DB(ctx)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if id > 0 {
		q = q.Where("id = ?", id)
	} else {
		q = q.Where("instance_id = ?", instanceID)
	}

	var instance model.Instance
	if err := q.First(&instance).Error; err != nil {
		return nil, ErrInstanceNotFound
	}
	return &instance, nil
}

// extractInstanceIDOrCVMID 从 query / form 中读取 id（DB 主键）和 instance_id（CVM ID）。
// id 解析失败（非法数字或 0）时返回错误；二者皆缺由调用方/findInstanceByIDOrCVMID 判定。
func extractInstanceIDOrCVMID(r *http.Request) (id uint, instanceID string, err error) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	instanceID = r.URL.Query().Get("instance_id")
	if instanceID == "" {
		instanceID = r.FormValue("instance_id")
	}
	if idStr != "" {
		parsed, perr := strconv.ParseUint(idStr, 10, 64)
		if perr != nil || parsed == 0 {
			return 0, "", hcommon.I18nError(i18n.MsgInvalidID)
		}
		id = uint(parsed)
	}
	return id, instanceID, nil
}

// getInstanceByIDRaw 根据 id 查询实例。userID > 0 时限制所有者，userID == 0 时不限（管理员用）。
// 查询成功后自动将实例 ID 绑定到 ResponseWriter（通过 WrapInstanceId），writeError 可自动提取。
func getInstanceByIDRaw(w *http.ResponseWriter, r *http.Request, userID uint) (*model.Instance, error) {
	id, instanceID, err := extractInstanceIDOrCVMID(r)
	if err != nil {
		return nil, err
	}
	instance, err := findInstanceByIDOrCVMID(r.Context(), userID, id, instanceID)
	if err != nil {
		return nil, err
	}
	*w = WrapInstanceId(*w, instance.InstanceId)
	return instance, nil
}

// getInstanceByID reads "id" or "instance_id" from query or form, resolves to Instance with ownership check.
func getInstanceByID(w *http.ResponseWriter, r *http.Request, user *model.User) (*model.Instance, error) {
	return getInstanceByIDRaw(w, r, user.ID)
}

// findInstanceByIDOrCVMIDForUserOrHandover 与 findInstanceByIDOrCVMID 类似，但查询条件
// 扩展为「自有实例 ∪ 待我接收的移交实例」(user_id = ? OR handover_target_user_id = ?)，
// 仅供 /openclaw/status 等只读端点使用；写端点（reboot/stop/reinstall 等）仍走 getInstanceByID，
// 确保移交接收方在 accept 之前无法对实例执行写操作。
func findInstanceByIDOrCVMIDForUserOrHandover(ctx context.Context, userID uint, id uint, instanceID string) (*model.Instance, error) {
	if id == 0 && instanceID == "" {
		return nil, hcommon.I18nError(i18n.MsgMissingIDOrInstanceID)
	}

	q := model.DB(ctx).Where("user_id = ? OR handover_target_user_id = ?", userID, userID)
	if id > 0 {
		q = q.Where("id = ?", id)
	} else {
		q = q.Where("instance_id = ?", instanceID)
	}

	var instance model.Instance
	if err := q.First(&instance).Error; err != nil {
		return nil, ErrInstanceNotFound
	}
	return &instance, nil
}

// getInstanceByIDOrHandoverTarget 与 getInstanceByID 类似，但查询范围扩展为
// 「自有实例 ∪ 待我接收的移交实例」，用于 /openclaw/status 等只读端点。
func getInstanceByIDOrHandoverTarget(w *http.ResponseWriter, r *http.Request, user *model.User) (*model.Instance, error) {
	id, instanceID, err := extractInstanceIDOrCVMID(r)
	if err != nil {
		return nil, err
	}
	instance, err := findInstanceByIDOrCVMIDForUserOrHandover(r.Context(), user.ID, id, instanceID)
	if err != nil {
		return nil, err
	}
	*w = WrapInstanceId(*w, instance.InstanceId)
	return instance, nil
}

// findDeletedInstanceForStatus 查找已被软删除（deleted_at 非空）的实例，
// 仅用于 /openclaw/status 常规查询返回 ErrInstanceNotFound 后的兜底。
// 复用「自有 ∪ 待接收移交」过滤条件（与 findInstanceByIDOrCVMIDForUserOrHandover 一致），
// 仅额外加 .Unscoped() 与 deleted_at 判定，防 IDOR。
// 注意：单条带括号的 WHERE 是必要的——拆成多个 Where 会因 AND 优先级高于 OR 而放宽权限过滤。
func findDeletedInstanceForStatus(r *http.Request, user *model.User) *model.Instance {
	id, instanceID, err := extractInstanceIDOrCVMID(r)
	if err != nil || (id == 0 && instanceID == "") {
		return nil
	}
	q := model.DB(r.Context()).Unscoped().
		Where("(user_id = ? OR handover_target_user_id = ?) AND deleted_at IS NOT NULL", user.ID, user.ID)
	if id > 0 {
		q = q.Where("id = ?", id)
	} else {
		q = q.Where("instance_id = ?", instanceID)
	}
	var inst model.Instance
	if err := q.Order("id DESC").First(&inst).Error; err != nil {
		return nil
	}
	return &inst
}

var cvmStateMap = map[string]struct {
	Label     string
	Color     string // tailwind text color class
	Bg        string // tailwind bg color class
	Transient bool   // whether this is a transitional state that should auto-refresh
}{
	"PENDING":       {"创建中", "text-yellow-700", "bg-yellow-50", true},
	"LAUNCH_FAILED": {"创建失败", "text-red-700", "bg-red-50", false},
	"RUNNING":       {"运行中", "text-green-700", "bg-green-50", false},
	"STOPPED":       {"已关机", "text-gray-600", "bg-gray-100", false},
	"STARTING":      {"开机中", "text-blue-700", "bg-blue-50", true},
	"STOPPING":      {"关机中", "text-yellow-700", "bg-yellow-50", true},
	"REBOOTING":     {"重启中", "text-yellow-700", "bg-yellow-50", true},
	"REINSTALLING":  {"重装中", "text-yellow-700", "bg-yellow-50", true},
	"SHUTDOWN":      {"停止待销毁", "text-red-700", "bg-red-50", false},
	"TERMINATING":   {"销毁中", "text-red-700", "bg-red-50", true},
}

// fetchCVMState 查询指定 CVM 实例的运行状态，返回状态字符串（如 "RUNNING"）。
// 实例不存在时返回 "RELEASED"，CVM InstanceId 为空时返回 ""。
func fetchCVMState(ctx context.Context, instanceId string) (string, error) {
	if instanceId == "" {
		return "", nil
	}
	client, err := NewCVMClient(ctx)
	if err != nil {
		return "", err
	}
	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{instanceId})
	response, callErr := client.DescribeInstances(request)
	if callErr != nil {
		return "", hcommon.I18nRichError(callErr, i18n.MsgPluginUpgradeQueryFail)
	}
	if response.Response == nil || len(response.Response.InstanceSet) == 0 {
		return "RELEASED", nil
	}
	state := ""
	if response.Response.InstanceSet[0].InstanceState != nil {
		state = *response.Response.InstanceSet[0].InstanceState
	}
	return state, nil
}

func HandleInstanceStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 查询范围扩展为「自有实例 ∪ 待我接收的移交实例」，与 /openclaw/list 并集口径一致；
	// 写端点（reboot/stop 等）仍走 getInstanceByID，确保移交接收方在 accept 前无法执行写操作。
	instance, err := getInstanceByIDOrHandoverTarget(&w, r, user)
	if err != nil {
		// 实例可能已被软删除：用 Unscoped 兜底确认后强制返回终态 destroyed。
		// 注意：此处不能复用 ResolveInstanceStatus——其输入契约是「活实例 + CVM 实况」，
		// 对软删实例会命中 creating/load_failed/running 等分支，甚至 transient=true 导致前端无限轮询。
		if errors.Is(err, ErrInstanceNotFound) {
			if deleted := findDeletedInstanceForStatus(r, user); deleted != nil {
				w = WrapInstanceId(w, deleted.InstanceId)
				log.Info("[InstanceStatus] 实例已销毁", "user_id", user.ID,
					"instance_id", deleted.InstanceId, "id", deleted.ID)
				// 强制返回终态 destroyed：不能复用 ResolveInstanceStatus——其输入契约是
				// 「活实例 + CVM 实况」，对软删实例会命中 creating/load_failed/running 等分支，
				// 甚至 transient=true 导致前端无限轮询。actions 置空：记录已软删，delete 无意义。
				def := model.UserStatusMap[model.StatusDestroyed]
				chargeType := instanceChargeTypeOrDefault(deleted.InstanceChargeType)
				label, tooltip := def.Label, def.Tooltip
				if hcommon.DefaultLangFromCtx(ctx) != "zh" {
					label, tooltip = def.LabelEn, def.TooltipEn
				}
				jsonOK(w, InstanceStatusResponse{
					Status:             def.Status,
					Label:              label,
					Tooltip:            tooltip,
					Actions:            []string{},
					Transient:          def.Transient,
					InstanceChargeType: chargeType,
				})
				return
			}
		}
		log.Warn("[InstanceStatus] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.IsDoctorNode {
		log.Warn("[InstanceStatus] 拒绝龙虾医生节点", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	log.Info("[InstanceStatus] 查询实例状态", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "name", instance.Name)

	// 本地实例：不走 CVM 状态机，直接走 resolveLocalInstanceStatus。
	// 本地实例的 InstanceId 不是 CVM 格式（不是 ins-xxxxxxxx），所以不能调
	// fetchCVMInstanceInfo 会报「实例ID不合要求」。同时 handleStatusSideEffects 里
	// 写回的都是 CVM 状态缓存字段（LastCVMState / agent_ready 等），本地
	// 实例不需要。
	if instance.Source == model.InstanceSourceLocal {
		status := ResolveInstanceStatus(r.Context(), instance, nil, nil)
		log.Info("[InstanceStatus][Local] 返回本地实例状态",
			"user_id", user.ID, "instance_id", instance.ID, "status", status.Status)
		jsonOK(w, status)
		return
	}

	// 查询 CVM 实例信息
	cvmInfo, err := fetchCVMInstanceInfo(r.Context(), instance.InstanceId)
	if err != nil {
		log.Error("[InstanceStatus] 查询 CVM 信息失败", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 状态映射
	status := ResolveInstanceStatus(r.Context(), instance, cvmInfo, nil)
	log.Info("[InstanceStatus] 解析实例状态完成", "user_id", user.ID, "instance_id", instance.ID, "status", status.Status, "transient", status.Transient, "actions", status.Actions)

	// 终端 action 过滤（final §3.2 决策）：
	// 终端是三端均支持的基础能力，按 agent 绑定的分组策略决定是否下发，
	// 对所有 agentType 一视同仁（不再按类型差异化判断）。
	config := model.GetSiteConfig(r.Context())
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyAgentTerminal, instance.GroupID, config.TerminalEnabled) {
		filteredActions := make([]string, 0, len(status.Actions))
		for _, action := range status.Actions {
			if action != "terminal" {
				filteredActions = append(filteredActions, action)
			}
		}
		status.Actions = filteredActions
	}

	// 副作用处理（同步部分，Agent 检测由后台 goroutine 异步完成）
	handleStatusSideEffects(r.Context(), model.DB(r.Context()), instance, cvmInfo, status.Status)

	log.Info("[InstanceStatus] 返回 JSON 状态", "user_id", user.ID, "instance_id", instance.ID, "status", status.Status)
	jsonOK(w, status)
}

func HandleRebootInstance(w http.ResponseWriter, r *http.Request) {
	handleRebootInstance(w, r, defaultStatusResolver)
}

func handleRebootInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	ctx := r.Context()
	log := Logger(ctx)
	if r.Method != http.MethodPost {
		log.Warn("[RebootInstance] 非法方法", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Warn("[RebootInstance] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.IsDoctorNode {
		log.Warn("[RebootInstance] 拒绝龙虾医生节点", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}
	log.Info("[RebootInstance] 收到重启请求", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "name", instance.Name)

	// 本地实例：不支持重启（无 CVM 侧可控性）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许重启
	if _, err := requireActionAllowedForUser(r.Context(), instance, "reboot", resolver); err != nil {
		log.Warn("[RebootInstance] 当前状态不允许重启", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeAgentGuardError(w, r, err)
		return
	}

	if instance.InstanceId == "" {
		log.Warn("[RebootInstance] 实例无关联的 CVM", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 乐观锁：写操作标记（已包含并发检查），同时原子重置 Agent 状态
	if err := setOperationWithAgentReset(model.DB(r.Context()), instance, model.OpReboot); err != nil {
		log.Warn("[RebootInstance] 写入操作标记失败（并发冲突）", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	client, err := NewCVMClient(ctx)
	if err != nil {
		log.Error("[RebootInstance] 创建 CVM 客户端失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed))
		return
	}

	req := cvm.NewRebootInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	log.Info("[RebootInstance] 调用 CVM RebootInstances", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	if _, err := client.RebootInstances(req); err != nil {
		// 失败时清除操作标记
		log.Error("[RebootInstance] CVM 重启失败", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRebootInstanceFailed))
		return
	}
	clearAdjustmentFailure(r.Context(), instance.ID)
	log.Info("[RebootInstance] 重启请求已下发成功", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleRestartGatewayInstance POST /openclaw/restart-gateway.
func HandleRestartGatewayInstance(w http.ResponseWriter, r *http.Request) {
	handleRestartGatewayInstance(w, r, defaultStatusResolver)
}

func handleRestartGatewayInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	ctx := r.Context()
	log := Logger(ctx)
	if r.Method != http.MethodPost {
		log.Warn("[RestartGatewayInstance] 非法方法", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Warn("[RestartGatewayInstance] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	log.Info("[RestartGatewayInstance] 收到重启 Agent 请求", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "name", instance.Name)
	if status, err := adminRestartGatewayOne(ctx, instance, resolver); err != nil {
		log.Warn("[RestartGatewayInstance] 重启 Agent 失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	log.Info("[RestartGatewayInstance] 重启 Agent 成功", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleResetInstance POST /openclaw/reset —— 用户自助重装实例。
func HandleResetInstance(w http.ResponseWriter, r *http.Request) {
	// handleResetInstance(w, r, defaultStatusResolver)
	commonHandleResetInstance(w, r, defaultStatusResolver, reinstallUserOpts)
}

// Deprecated: 重构用户重装和管理员重装方法，使用统一的 commonHandleResetInstance
// FIXME: 2026-05-28 未便于快速回退，保留该函数，稳定后可以移除
func handleResetInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	ctx := r.Context()
	log := Logger(ctx)
	if r.Method != http.MethodPost {
		log.Warn("[ResetInstance] 非法方法", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Warn("[ResetInstance] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.IsDoctorNode {
		log.Warn("[ResetInstance] 拒绝龙虾医生节点", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}
	log.Info("[ResetInstance] 收到重装请求", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "name", instance.Name, "agent_type", instance.AgentType)

	// final §6 C3：仅支持重装的类型（OpenClaw）允许该操作，否则 403
	// Hermes/ACE 重装会走到 openclaw 脚本链路失败，必须在入口拦截
	if err := checkInstanceSupportsReinstall(r.Context(), instance); err != nil {
		log.Warn("[ResetInstance] 该类型不支持重装", "user_id", user.ID, "instance_id", instance.ID, "agent_type", instance.AgentType, "error", err)
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：不支持重装（无 CVM 侧镜像/系统盘可换）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许重装
	if _, err := requireActionAllowedForUser(r.Context(), instance, "reinstall", resolver); err != nil {
		log.Warn("[ResetInstance] 当前状态不允许重装", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeAgentGuardError(w, r, err)
		return
	}

	if instance.InstanceId == "" {
		log.Warn("[ResetInstance] 实例无关联的 CVM", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 乐观锁：写操作标记（已包含并发检查），同时原子重置 Agent 状态
	if err := setOperationWithAgentReset(model.DB(r.Context()), instance, model.OpReinstall); err != nil {
		log.Warn("[ResetInstance] 写入操作标记失败（并发冲突）", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 先查询启用镜像，确认有可用镜像再执行重置，避免清空数据后又无法继续
	enabledImage, err := model.GetEnabledImageByType(r.Context(), instance.AgentType)
	if err != nil {
		log.Error("[ResetInstance] 查询启用镜像失败", "user_id", user.ID, "instance_id", instance.ID, "agent_type", instance.AgentType, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
		return
	}
	if enabledImage == nil {
		log.Warn("[ResetInstance] 未找到该类型的启用镜像", "user_id", user.ID, "instance_id", instance.ID, "agent_type", instance.AgentType)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nError(i18n.MsgNoImageForType, typeName))
		return
	}

	// 三期新增：跨 agent_type 防御 —— 堵住 GetEnabledImageByType 回退到空类型镜像导致
	// Hermes/ACE 实例拿到老 OpenClaw 镜像去重装的错乱路径。
	if err := verifyReinstallImageMatches(r.Context(), instance, enabledImage); err != nil {
		log.Error("[ResetInstance] 重装镜像类型校验失败", "user_id", user.ID, "instance_id", instance.ID, "agent_type", instance.AgentType, "image_id", enabledImage.ImageId, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 重置版本信息（重装后需重新拉取，不清空 agent_type）
	if err := resetInstanceVersionInfo(r.Context(), instance); err != nil {
		log.Error("[ResetInstance] 重置版本信息失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgResetVersionFailed))
		return
	}

	client, err := NewCVMClient(ctx)
	if err != nil {
		log.Error("[ResetInstance] 创建 CVM 客户端失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed))
		return
	}

	req := cvm.NewResetInstanceRequest()
	req.InstanceId = common.StringPtr(instance.InstanceId)
	req.ImageId = common.StringPtr(enabledImage.ImageId)

	// Set UserData from init.sh template and persisted user-provided UserData.
	config := model.GetSiteConfig(r.Context())
	var systemUserDataConfig *initUserDataConfig
	if config.SkillHub != "" {
		systemUserDataConfig = &initUserDataConfig{
			SkillHub:    config.SkillHub,
			RuntimeUser: getEffectiveRuntimeUser(instance.RuntimeUser),
			AgentType:   instance.AgentType,
		}
	}
	mergedUserData, err := buildUserData(ctx, systemUserDataConfig, instance.UserData)
	if err != nil {
		log.Error("[ResetInstance] 渲染 UserData 失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if mergedUserData != "" {
		req.EnhancedService = &cvm.EnhancedService{
			AutomationService: &cvm.RunAutomationServiceEnabled{
				Enabled: common.BoolPtr(true),
			},
		}
		req.UserData = common.StringPtr(mergedUserData)
	}

	log.Info("[ResetInstance] 调用 CVM ResetInstance", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "image_id", enabledImage.ImageId)
	if _, err := client.ResetInstance(req); err != nil {
		log.Error("[ResetInstance] CVM 重装失败", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		richErr := hcommon.I18nError(i18n.MsgReinstallInstanceFailed).WithDetail(err.Error())
		writeError(w, r, http.StatusInternalServerError, richErr)
		go createErrorNotification(user.ID, instance.ID, instance.Name, model.NotifyTypeInstanceReinstallFailed, "实例重装失败", richErr, hcommon.DetachContext(r.Context()))
		return
	}
	clearAdjustmentFailure(r.Context(), instance.ID)
	log.Info("[ResetInstance] 重装请求已下发成功", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "image_id", enabledImage.ImageId)

	// 重装成功后直写镜像缓存（失败仅记录日志，不影响主流程）
	if err := model.DB(r.Context()).Model(instance).Update("img_id", enabledImage.ImageId).Error; err != nil {
		log.Warn("[ResetInstance] 直写 img_id 缓存失败", "instanceId", instance.InstanceId, "error", err)
	}

	// 重装后清空该实例的所有模型绑定，并将 instances.ai_model_id 置 0。
	// 两步操作放在同一事务中，避免部分失败导致 instance_models 与 instances 数据不一致。
	// 使用物理删除（Unscoped），防止软删除残留占用唯一索引导致重装后绑定模型报 Duplicate entry。
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := model.HardDeleteInstanceModels(tx, instance.ID); err != nil {
			return err
		}
		return tx.Model(instance).Update("ai_model_id", 0).Error
	}); err != nil {
		slog.Warn("[ResetInstance] 清空模型绑定失败（非阻塞）",
			"instance_pk", instance.ID, "error", err)
	}

	// 重置 plugin 状态，但保留 Pro 绑定信息
	resetMemoryPluginForReinstall(hcommon.DetachContext(r.Context()), instance.InstanceId)

	// 清理旧的技能安装记录，创建新任务并异步安装
	model.DB(r.Context()).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
	createSkillInstallTasks(r.Context(), instance.ID, instance.RoleID, instance.GroupID)
	// v7：重装场景无局部 agentType 变量，直接用 instance.AgentType
	go installSkillsAsync(hcommon.DetachContext(r.Context()), instance.ID, instance.InstanceId, instance.AgentType, waitModeReinstall)

	// 清理旧的插件安装记录，创建新任务并异步安装
	model.DB(r.Context()).Where("instance_id = ?", instance.ID).Delete(&model.PluginInstallation{})
	createPluginInstallTasks(r.Context(), instance.ID, instance.RoleID)
	go installPluginsAsync(hcommon.DetachContext(r.Context()), instance.ID, instance.InstanceId, waitModeReinstall)

	// 异步 approve device（等 TAT 就绪后执行）
	go approveDeviceAsync(hcommon.DetachContext(r.Context()), instance.ID, instance.InstanceId, instance.RuntimeUser)

	// 重装后恢复 SMH 个人空间环境（前置检查 + 重置 DB 状态 + 异步等待 CVM 就绪后触发安装）
	go syncSMHEnvWhenReadyFn(hcommon.DetachContext(r.Context()), *instance)

	jsonOK(w, map[string]interface{}{"ok": true})
}

func applyDefaultMemoryPlanForInstance(ctx context.Context, instanceID string, config model.SiteConfig) {
	if instanceID == "" {
		return
	}
	// 仅对支持记忆的 agent_type 生效
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).First(&inst).Error; err != nil {
		return
	}
	if !model.AgentTypeSupportsMemory(ctx, inst.AgentType) {
		return
	}
	model.EnsureMemoryTDAIPluginRow(ctx, instanceID)

	// 决定 planInput：优先匹配分组策略，不命中再用预设策略
	planInput := resolveMemoryPlanForGroup(ctx, inst.GroupID, config)

	desiredPlan, jobType, switchStatus, ok := resolveMemoryPlanTransition(planInput)
	if !ok || desiredPlan == model.MemoryPlanOff {
		return
	}

	bizKey := fmt.Sprintf("switch:%s", instanceID)
	if _, err := model.SubmitJob(ctx, jobType, bizKey, instanceID, "{}", "system:auto_default_plan", ""); err != nil {
		slog.Warn("[CreateInstance] 自动提交默认记忆计划任务失败（不阻塞创建）",
			"instance_id", instanceID, "default_plan", planInput, "error", err)
		return
	}

	model.DB(ctx).Model(&model.MemoryTDAIPlugin{}).
		Where("instance_id = ?", instanceID).
		Updates(map[string]any{
			"desired_plan":  desiredPlan,
			"switch_status": switchStatus,
		})
	slog.Info("[CreateInstance] 已提交默认记忆计划任务",
		"instance_id", instanceID, "default_plan", planInput)
}

// resolveMemoryPlanForGroup 根据实例所属分组匹配记忆分组策略，返回应使用的 plan（小写）。
// 匹配逻辑：查实例 group_id 的祖先链，看 memory_plan_group_policies 表中是否有命中。
// 命中则返回策略的 plan；不命中则返回预设策略。
func resolveMemoryPlanForGroup(ctx context.Context, groupID uint, config model.SiteConfig) string {
	defaultPlan := config.MemoryDefaultPlan
	if strings.TrimSpace(defaultPlan) == "" && config.MemoryTDAIEnable {
		defaultPlan = model.MemoryDefaultPlanFree
	}
	if defaultPlan == "" {
		defaultPlan = model.MemoryDefaultPlanOff
	}

	if groupID == 0 {
		return defaultPlan
	}

	// 查祖先链（含自身）
	ancestorIDs, err := model.ClosureAncestors(ctx, groupID, true)
	if err != nil || len(ancestorIDs) == 0 {
		return defaultPlan
	}

	// 查策略表
	var policy model.MemoryPlanGroupPolicy
	if err := model.DB(ctx).Where("group_id IN ?", ancestorIDs).First(&policy).Error; err == nil {
		return policy.Plan // 命中
	}

	return defaultPlan
}

func resetMemoryPluginForReinstall(ctx context.Context, instanceID string) {
	model.DB(ctx).Model(&model.MemoryTDAIPlugin{}).Where("instance_id = ?", instanceID).
		Updates(map[string]interface{}{
			"status":      model.MemoryTDAIPluginStatusNotInstalled,
			"retry_count": 0,
			"last_error":  "",
		})
	resubmitProSwitchAfterReinstall(ctx, instanceID)
}

// approveDeviceAsync 异步等待 TAT 就绪后执行 approve_device.sh 脚本，
// 自动审批 openclaw 设备的 pending 请求。
// 创建实例和重装实例时调用，因为新实例/重装后实例首次连接会产生 pending device request。
func approveDeviceAsync(ctx context.Context, instancePK uint, cvmInstanceId string, runtimeUser string) {
	logger := slog.With("task", "approveDeviceAsync", "instance_pk", instancePK, "cvm_instance_id", cvmInstanceId)

	// final：approve_device 仅 OpenClaw 支持。虽然现有上游入口（HandleCreate / HandleReinstall）
	// 本就只创建/重装 OpenClaw 实例，这里仍做一层防御式 guard，避免未来新入口绕过。
	agentType := LookupAgentType(ctx, cvmInstanceId)
	if !model.AgentTypeSupportsApprove(ctx, agentType) {
		logger.Info("实例类型不支持 approve_device，跳过", "agent_type", agentType)
		return
	}

	// 等待 TAT Agent 上线（每 10 秒检查一次，最多等 10 分钟）
	tatClient, err := NewTATClient(ctx)
	if err != nil {
		logger.Warn("创建 TAT 客户端失败，放弃 approve device", "error", err)
		return
	}
	agentOnline := false
	for attempt := 1; attempt <= 60; attempt++ {
		if err := checkAgentOnline(tatClient, cvmInstanceId); err == nil {
			agentOnline = true
			logger.Info("TAT Agent 已上线", "attempt", attempt)
			break
		}
		logger.Info("TAT Agent 尚未就绪", "attempt", attempt)
		time.Sleep(10 * time.Second)
	}
	if !agentOnline {
		logger.Warn("等待 TAT Agent 就绪超时，放弃 approve device")
		return
	}

	// final：TAT Agent 上线 ≠ openclaw-gateway 就绪。新建实例冷启动场景下，
	// TAT 通常先于 openclaw 完成安装/启动；若此处不等 openclaw 自身就绪就下发
	// approve_device.sh，脚本启动时 ~/.openclaw/devices/paired.json 尚未落盘，
	// step 0 prime list 即使触发握手也来不及，会在 step 1 直接报 "no paired.json" 失败。
	// 复用升级路径的 waitForOpenclawReady（最多 5 分钟），与 reinstallAndRestore 保持一致语义。
	if err := waitForOpenclawReady(ctx, cvmInstanceId, agentType, 5*time.Minute); err != nil {
		logger.Warn("等待 openclaw 就绪超时，放弃 approve device", "error", err)
		return
	}
	logger.Info("openclaw 已就绪，开始执行 approve_device.sh")

	// 执行 approve_device.sh（失败不影响主流程，仅记录日志）
	output, rerr := RunScript(ctx, cvmInstanceId, "approve_device.sh", 300, runtimeUser, nil, nil)
	if rerr != nil {
		logger.Warn("approve device 失败", "error", rerr)
		return
	}
	logger.Info("approve device 完成", "output", output)
}

func HandleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	// final：approve.sh 是 openclaw CLI 特有的"设备/CLI 授权回调"流程，
	// Hermes (harness) / ACE (lightclaw) 走自己的 OAuth / Server API，无需此接口。
	// 对非 OpenClaw 实例直接 403，避免执行不存在的 approve.sh 返回 500。
	if err := checkInstanceSupportsApprove(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	module := r.FormValue("module")
	if module == "" {
		module = "feishu"
	}
	code := r.FormValue("code")
	if code == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingCodeParam))
		return
	}

	output, err := RunScript(r.Context(), instance.InstanceId, "approve.sh", 60, instance.RuntimeUser, nil, map[string]string{
		"module": module,
		"code":   code,
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, map[string]string{"ok": "true", "output": output})
}

// runScriptForServiceStatusFn 是 HandleServiceStatus 调用 RunScript 的可替换包装，
// 方便单元测试 mock TAT 调用（无法在测试环境真实调用 TAT）。
var runScriptForServiceStatusFn = RunScript

func HandleServiceStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)

	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Warn("[ServiceStatus] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	log.Info("[ServiceStatus] 收到服务状态查询",
		"user_id", user.ID, "instance_id", instance.ID,
		"cvm_id", instance.InstanceId, "agent_type", instance.AgentType)

	// final：按 agent_type 分派 check_service 脚本
	// - openclaw：openclaw status --json
	// - hermes：harness gateway status + channel list
	// - ace：lightclaw status + 读 lightclaw.json
	scriptName, err := ResolveScript(ctx, "check_service", instance.AgentType)
	if err != nil {
		log.Warn("[ServiceStatus] 解析 check_service 脚本失败",
			"user_id", user.ID, "instance_id", instance.ID,
			"agent_type", instance.AgentType, "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgAgentTypeNotSupportCheck, instance.AgentType))
		return
	}
	log.Info("[ServiceStatus] 调用 check_service 脚本",
		"user_id", user.ID, "instance_id", instance.ID,
		"cvm_id", instance.InstanceId, "script", scriptName)
	output, err := runScriptForServiceStatusFn(r.Context(), instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, nil)
	if err != nil {
		log.Error("[ServiceStatus] 执行 check_service 脚本失败",
			"user_id", user.ID, "instance_id", instance.ID,
			"cvm_id", instance.InstanceId, "script", scriptName, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	log.Info("[ServiceStatus] 查询成功",
		"user_id", user.ID, "instance_id", instance.ID,
		"cvm_id", instance.InstanceId, "output_len", len(output))

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, output)
}

// HandleCheckAgentReady 通过 TAT 按 agent_type 分派检查脚本，判断 Agent 是否就绪。
// final §6 C16：原函数名 HandleCheckOpenclawPort 已泛化为"任意 agent ready 检查"。
//
// 按类型路由：
//   - openclaw     → check_openclaw_ready.sh
//   - hermes       → check_hermes_ready.sh
//   - lightclawace → check_ace_ready.sh
//
// 路由: GET /openclaw/check-openclaw-port（URL 保持历史名，前端兼容；语义已泛化）
//
// 响应: {"running": true} 或 {"running": false}（前端未处理非 2xx 响应，此接口统一以 2xx 返回）
func HandleCheckAgentReady(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 无兼容运行时类型（未声明 compatible_with 的自定义类型等）不定义 check_ready 业务语义——
	// 以 DB 里已确认的 agent_ready 作为运行判定：就绪即 running=true，否则 false。
	if model.GetAgentRuntimeType(r.Context(), instance.AgentType) == "" {
		jsonOK(w, map[string]any{"running": instance.AgentReady == 1})
		return
	}

	// v7：按 agent_type 分派 check_ready 脚本。
	// 特殊路径：与 RunScript 失败保持同一语义（返回 running:false JSON），不走 writeError，
	// 因为前端未处理非 2xx 响应。
	scriptName, resolveErr := ResolveScript(r.Context(), "check_ready", instance.AgentType)
	if resolveErr != nil {
		slog.Warn("[CheckAgentReady] resolveScript failed", "agent_type", instance.AgentType, "error", resolveErr)
		jsonOK(w, map[string]any{"running": false})
		return
	}

	output, err := RunScript(r.Context(), instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, nil)
	if err != nil {
		// 命令下发失败，返回 running: false
		jsonOK(w, map[string]any{"running": false})
		return
	}

	// check_openclaw_ready.sh 输出 {"ready": ...}，转换为前端期望的 {"running": ...}
	var result struct {
		Ready  bool   `json:"ready"`
		Reason string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		jsonOK(w, map[string]any{"running": false})
		return
	}
	resp := map[string]any{"running": result.Ready}
	if result.Reason != "" {
		resp["reason"] = result.Reason
	}
	jsonOK(w, resp)
}

// HandleInstanceTerminal 为普通用户获取其实例的终端授权访问 URL。
//
// 路由: POST /openclaw/terminal-url
//
// 流程:
//  1. 校验用户登录状态
//  2. 校验管理员是否开启了终端功能（TerminalEnabled）
//  3. 根据实例 ID 查询用户自己的实例，获取对应 CVM InstanceId
//  4. 调用 DescribeInstances 获取 Region、UserName、PlatformType
//  5. 调用 OrcaTerm GenerateAuthLoginUrl 获取终端登录 URL
//
// 请求参数:
//
//	id: 实例 ID（form 参数）
//
// 响应:
//
//	{ "login_url": "https://orcaterm.cloud.tencent.com/terminal?..." }
func HandleInstanceTerminal(w http.ResponseWriter, r *http.Request) {
	handleInstanceTerminal(w, r, defaultStatusResolver)
}

func handleInstanceTerminal(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 检查管理员是否开启终端功能
	// 注：终端开关只对 openclaw 类型生效，hermes 和 lightclaw-ace 始终允许进入终端进行配置
	config := model.GetSiteConfig(r.Context())

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 终端 guard（final §3.2 决策）：
	// 终端是三端均支持的基础能力，按 agent 绑定的分组策略决定。
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyAgentTerminal, instance.GroupID, config.TerminalEnabled) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgTerminalFeatureDisabled))
		return
	}

	// 本地实例：不支持进入远程终端（本地机器由用户直接操作）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许进入终端
	if _, err := requireActionAllowedForUser(r.Context(), instance, "terminal", resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 查询 CVM 实例信息，获取 Region、UserName、PlatformType
	cvmClient, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	descReq := cvm.NewDescribeInstancesRequest()
	descReq.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	var descResp *cvm.DescribeInstancesResponse
	derr := RetryCloudCall(r.Context(), func() error {
		var callErr error
		descResp, callErr = cvmClient.DescribeInstances(descReq)
		return callErr
	})
	if derr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(derr, i18n.MsgQueryCVMInstanceFailed))
		return
	}
	if descResp.Response == nil || len(descResp.Response.InstanceSet) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCVMInstanceNotFound))
		return
	}

	inst := descResp.Response.InstanceSet[0]

	// 使用默认的 CVMRegion
	instanceRegion := CVMRegion

	// 判断是否为 Windows 系统
	isWindows := inst.OsName != nil && strings.Contains(strings.ToLower(*inst.OsName), "windows")

	// UserName: 优先使用实例实际检测到的 RuntimeUser，
	// 未检测到时 fallback 到 root。Windows 实例使用 Administrator。
	userName := getEffectiveRuntimeUser(instance.RuntimeUser)
	if isWindows {
		userName = "Administrator"
	}

	slog.Info("[CVM] 用户终端 - 查询实例信息成功",
		"user_id", user.ID,
		"instance_id", instance.InstanceId,
		"region", instanceRegion,
		"user_name", userName,
		"os_name", StrVal(inst.OsName),
	)

	credential, err := getCredential(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGetCloudCredentialFailed))
		return
	}

	loginURL, urlErr := generateAuthLoginUrl(credential, instance.InstanceId, instanceRegion, userName)
	if urlErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGetTerminalLoginURLFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"login_url": loginURL,
	})
}

func HandleDescribeZones(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if user := requireLogin(w, r); user == nil {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBody))
		return
	}

	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	req := cvm.NewDescribeZonesRequest()
	if len(body) > 0 {
		if err := req.FromJsonString(string(body)); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
			return
		}
	}
	resp, err := client.DescribeZones(req)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryZonesFailed))
		return
	}

	jsonOK(w, resp)
}

// vpcPolicyClient 抽象 VPC 安全组的查询、创建与删除操作，便于单元测试 mock。
type vpcPolicyClient interface {
	DescribeSecurityGroupPolicies(req *vpc.DescribeSecurityGroupPoliciesRequest) (*vpc.DescribeSecurityGroupPoliciesResponse, error)
	CreateSecurityGroup(req *vpc.CreateSecurityGroupRequest) (*vpc.CreateSecurityGroupResponse, error)
	CreateSecurityGroupPolicies(req *vpc.CreateSecurityGroupPoliciesRequest) (*vpc.CreateSecurityGroupPoliciesResponse, error)
	DeleteSecurityGroup(req *vpc.DeleteSecurityGroupRequest) (*vpc.DeleteSecurityGroupResponse, error)
}

// cvmSGClient 抽象 CVM 安全组绑定/解绑操作，便于单元测试 mock。
type cvmSGClient interface {
	AssociateSecurityGroups(req *cvm.AssociateSecurityGroupsRequest) (*cvm.AssociateSecurityGroupsResponse, error)
	DisassociateSecurityGroups(req *cvm.DisassociateSecurityGroupsRequest) (*cvm.DisassociateSecurityGroupsResponse, error)
}

// ensureInstanceEgressRules 检查实例绑定的安全组是否有出站规则，若无则基于默认安全组复制一份新安全组
// （保留原入站规则，添加全放通出站规则），然后替换实例上的安全组绑定。
// 用于未配置系统安全组时，修复腾讯云默认安全组缺少出站规则的问题。
// 由于实例刚创建时可能处于 PENDING 状态，安全组信息尚未就绪，因此会先等待再重试查询。
func ensureInstanceEgressRules(ctx context.Context, instanceId string) {
	log := Logger(ctx)
	// 等待实例初始化完成，避免查询到空的安全组列表
	time.Sleep(15 * time.Second)

	var sgMap map[string][]string
	var err error

	// 重试机制：最多重试 10 次，每次间隔 10 秒
	for attempt := 1; attempt <= 10; attempt++ {
		sgMap, err = describeInstancesSecurityGroups(ctx, []string{instanceId})
		if err != nil {
			log.Warn("ensureInstanceEgressRules: 查询实例安全组失败",
				"instanceId", instanceId, "attempt", attempt, "err", err)
			if attempt < 10 {
				time.Sleep(10 * time.Second)
				continue
			}
			return
		}

		// 检查是否查询到了安全组信息
		sgIds, ok := sgMap[instanceId]
		if ok && len(sgIds) > 0 {
			break // 成功获取到安全组信息
		}

		log.Warn("ensureInstanceEgressRules: 实例安全组列表为空，可能尚未就绪",
			"instanceId", instanceId, "attempt", attempt)
		if attempt < 10 {
			time.Sleep(10 * time.Second)
		}
	}

	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		log.Warn("ensureInstanceEgressRules: 创建 VPC 客户端失败", "err", err)
		return
	}

	cvmClient, rerr := NewCVMClient(ctx)
	if rerr != nil {
		log.Warn("ensureInstanceEgressRules: 创建 CVM 客户端失败", "err", rerr)
		return
	}

	ensureEgressRulesCore(ctx, instanceId, sgMap, vpcClient, cvmClient)
}

// cleanPoliciesForCreate 清理从 DescribeSecurityGroupPolicies 返回的规则，使其可用于 CreateSecurityGroupPolicies：
//  1. 去除只读字段（PolicyIndex、Priority、ModifyTime）
//  2. 将所有 *string 类型字段中值为空字符串的指针置 nil（腾讯云 API 对空字符串会报 InvalidParameterValue.Malformed）
//  3. 将 AddressTemplate / ServiceTemplate 中空字符串子字段置 nil；若两个子字段均为 nil 则整个嵌套结构也置 nil
//  4. PolicyDescription 空字符串置 nil
func cleanPoliciesForCreate(policies []*vpc.SecurityGroupPolicy) []*vpc.SecurityGroupPolicy {
	// nilIfEmpty 将值为空字符串的 *string 指针置 nil
	nilIfEmpty := func(s *string) *string {
		if s != nil && *s == "" {
			return nil
		}
		return s
	}

	result := make([]*vpc.SecurityGroupPolicy, 0, len(policies))
	for _, p := range policies {
		cloned := *p
		// 去除只读字段
		cloned.PolicyIndex = nil
		cloned.Priority = nil
		cloned.ModifyTime = nil
		// 顶层 *string 字段：空字符串置 nil
		cloned.Protocol = nilIfEmpty(cloned.Protocol)
		cloned.Port = nilIfEmpty(cloned.Port)
		cloned.CidrBlock = nilIfEmpty(cloned.CidrBlock)
		cloned.Ipv6CidrBlock = nilIfEmpty(cloned.Ipv6CidrBlock)
		cloned.SecurityGroupId = nilIfEmpty(cloned.SecurityGroupId)
		cloned.PolicyDescription = nilIfEmpty(cloned.PolicyDescription)
		// AddressTemplate：子字段空字符串置 nil，全空则整体置 nil
		if cloned.AddressTemplate != nil {
			at := *cloned.AddressTemplate
			at.AddressId = nilIfEmpty(at.AddressId)
			at.AddressGroupId = nilIfEmpty(at.AddressGroupId)
			if at.AddressId == nil && at.AddressGroupId == nil {
				cloned.AddressTemplate = nil
			} else {
				cloned.AddressTemplate = &at
			}
		}
		// ServiceTemplate：子字段空字符串置 nil，全空则整体置 nil
		if cloned.ServiceTemplate != nil {
			st := *cloned.ServiceTemplate
			st.ServiceId = nilIfEmpty(st.ServiceId)
			st.ServiceGroupId = nilIfEmpty(st.ServiceGroupId)
			if st.ServiceId == nil && st.ServiceGroupId == nil {
				cloned.ServiceTemplate = nil
			} else {
				cloned.ServiceTemplate = &st
			}
		}
		result = append(result, &cloned)
	}
	return result
}

// ensureEgressRulesCore 是 ensureInstanceEgressRules 的可测试核心逻辑。
// 目标：实例只绑定一个安全组（基于默认安全组复制、带全放通出站规则的自动安全组），解绑其他所有安全组。
// 若 config 中已存有自动创建的安全组 ID（AutoCreatedSecurityGroupId），则直接复用，不再重复创建。
// 幂等检查：若实例已经只绑定了该自动安全组，则直接跳过。
func ensureEgressRulesCore(ctx context.Context, instanceId string, sgMap map[string][]string, vpcCli vpcPolicyClient, cvmCli cvmSGClient) {
	log := Logger(ctx)
	sgIds, ok := sgMap[instanceId]
	if !ok || len(sgIds) == 0 {
		log.Warn("ensureEgressRulesCore: 实例未绑定任何安全组", "instanceId", instanceId)
		return
	}

	config := model.GetSiteConfig(ctx)
	newSgId := ""
	if config.AutoCreatedSecurityGroupId != "" {
		// 已有自动创建的安全组，直接复用
		newSgId = config.AutoCreatedSecurityGroupId
		log.Info("ensureEgressRulesCore: 复用已有自动创建的安全组", "newSgId", newSgId)
		// 幂等检查：若实例已只绑定该自动安全组，直接跳过所有操作
		if len(sgIds) == 1 && sgIds[0] == newSgId {
			log.Info("ensureEgressRulesCore: 实例已只绑定自动安全组，跳过",
				"instanceId", instanceId, "newSgId", newSgId)
			return
		}
	} else {
		// 从实例绑定的安全组中找出 IsDefault==true 的默认安全组作为模板
		// 复制其入站规则，出站规则强制覆盖为全放通
		sourceSgId, err := findDefaultSgIdFromList(ctx, sgIds)
		if err != nil {
			log.Warn("ensureEgressRulesCore: 查询默认安全组失败", "err", err)
			return
		}
		if sourceSgId == "" {
			// 找不到默认安全组，降级取第一个
			sourceSgId = sgIds[0]
			log.Warn("ensureEgressRulesCore: 未找到默认安全组，降级使用第一个", "sgId", sourceSgId)
		} else {
			log.Info("ensureEgressRulesCore: 找到默认安全组作为模板", "sgId", sourceSgId)
		}
		var sourcePolicySet *vpc.SecurityGroupPolicySet
		policyReq := vpc.NewDescribeSecurityGroupPoliciesRequest()
		policyReq.SecurityGroupId = common.StringPtr(sourceSgId)
		policyResp, err := vpcCli.DescribeSecurityGroupPolicies(policyReq)
		if err != nil {
			log.Warn("ensureEgressRulesCore: 查询模板安全组规则失败", "sgId", sourceSgId, "err", err)
			return
		}
		if policyResp.Response != nil && policyResp.Response.SecurityGroupPolicySet != nil {
			sourcePolicySet = policyResp.Response.SecurityGroupPolicySet
		}

		// 创建空安全组，最多重试 3 次，保证创建成功后再写 DB
		createReq := vpc.NewCreateSecurityGroupRequest()
		createReq.GroupName = common.StringPtr("clawpro-default")
		createReq.GroupDescription = common.StringPtr(fmt.Sprintf("从 %s 复制并补全出站规则（自动生成）", sourceSgId))
		var createResp *vpc.CreateSecurityGroupResponse
		var createErr error
		for retry := 1; retry <= 3; retry++ {
			createResp, createErr = vpcCli.CreateSecurityGroup(createReq)
			if createErr == nil && createResp.Response != nil &&
				createResp.Response.SecurityGroup != nil &&
				createResp.Response.SecurityGroup.SecurityGroupId != nil {
				break
			}
			if createErr != nil {
				log.Warn("ensureEgressRulesCore: 创建新安全组失败", "sourceSgId", sourceSgId, "retry", retry, "err", createErr)
			} else {
				log.Warn("ensureEgressRulesCore: 创建新安全组返回数据异常", "sourceSgId", sourceSgId, "retry", retry)
				createErr = hcommon.I18nError(i18n.MsgCreateSecurityGroupDataError)
			}
			if retry < 3 {
				time.Sleep(2 * time.Second)
			}
		}
		if createErr != nil {
			log.Warn("ensureEgressRulesCore: 创建新安全组重试3次均失败，退出", "sourceSgId", sourceSgId, "err", createErr)
			return
		}
		newSgId = *createResp.Response.SecurityGroup.SecurityGroupId

		// 通过 CAS 将安全组 ID 写入 config，防止并发重复创建
		// 创建成功后才写 DB；写入失败则删除刚创建的安全组回滚
		result := model.DB(ctx).Model(&model.SiteConfig{}).
			Where("auto_created_security_group_id = ''").
			Update("auto_created_security_group_id", newSgId)
		if result.RowsAffected == 0 {
			// 并发场景：另一个 goroutine 已先写入，删除当前创建的无用空安全组，改用已有安全组
			config = model.GetSiteConfig(ctx)
			if config.AutoCreatedSecurityGroupId != "" {
				delReq := vpc.NewDeleteSecurityGroupRequest()
				delReq.SecurityGroupId = common.StringPtr(newSgId)
				if _, delErr := vpcCli.DeleteSecurityGroup(delReq); delErr != nil {
					log.Warn("ensureEgressRulesCore: 删除并发冲突产生的空安全组失败", "sgId", newSgId, "err", delErr)
				} else {
					log.Info("ensureEgressRulesCore: 已删除并发冲突产生的空安全组", "sgId", newSgId)
				}
				newSgId = config.AutoCreatedSecurityGroupId
				log.Info("ensureEgressRulesCore: 并发写入冲突，改用已有安全组", "newSgId", newSgId)
			} else {
				// DB 读取延迟，无法确定正确 ID，安全退出避免使用已删除的安全组
				log.Warn("ensureEgressRulesCore: CAS 冲突但读取不到已有安全组 ID，跳过", "instanceId", instanceId)
				return
			}
		} else {
			log.Info("ensureEgressRulesCore: 新安全组 ID 已存入 config", "newSgId", newSgId)

			// CAS 写入成功，为新安全组分两次添加规则（腾讯云 API 不支持 Ingress 和 Egress 同时传入）：
			// 第一次：添加全放通出站规则
			egressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
			egressReq.SecurityGroupId = common.StringPtr(newSgId)
			egressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
				Egress: []*vpc.SecurityGroupPolicy{
					{
						Protocol:          common.StringPtr("ALL"),
						Port:              common.StringPtr("ALL"),
						CidrBlock:         common.StringPtr("0.0.0.0/0"),
						Action:            common.StringPtr("ACCEPT"),
						PolicyDescription: common.StringPtr("全放通出站规则（自动补全）"),
					},
				},
			}
			var addPolicyErr error
			for retry := 1; retry <= 3; retry++ {
				if _, addPolicyErr = vpcCli.CreateSecurityGroupPolicies(egressReq); addPolicyErr == nil {
					break
				}
				log.Warn("ensureEgressRulesCore: 为新安全组添加出站规则失败",
					"newSgId", newSgId, "retry", retry, "err", addPolicyErr)
				if retry < 3 {
					time.Sleep(2 * time.Second)
				}
			}
			if addPolicyErr != nil {
				log.Warn("ensureEgressRulesCore: 为新安全组添加出站规则重试3次均失败，回滚并跳过绑定",
					"newSgId", newSgId, "err", addPolicyErr)
				// 回滚：仅当 DB 中仍是自己写入的值时才清空，防止并发场景下覆盖其他实例写入的值
				rollbackResult := model.DB(ctx).Model(&model.SiteConfig{}).
					Where("auto_created_security_group_id = ?", newSgId).
					Update("auto_created_security_group_id", "")
				if rollbackResult.RowsAffected == 1 {
					// 确认是自己写入的值被清空，才能安全删除安全组
					delReq := vpc.NewDeleteSecurityGroupRequest()
					delReq.SecurityGroupId = common.StringPtr(newSgId)
					if _, delErr := vpcCli.DeleteSecurityGroup(delReq); delErr != nil {
						log.Warn("ensureEgressRulesCore: 回滚删除空安全组失败", "sgId", newSgId, "err", delErr)
					} else {
						log.Info("ensureEgressRulesCore: 回滚已删除空安全组", "sgId", newSgId)
					}
				} else {
					log.Warn("ensureEgressRulesCore: 回滚时安全组已被其他实例使用，跳过删除", "sgId", newSgId)
				}
				return
			}
			// 第二次：从模板安全组复制入站规则（若有）
			if sourcePolicySet != nil && len(sourcePolicySet.Ingress) > 0 {
				ingressRules := cleanPoliciesForCreate(sourcePolicySet.Ingress)
				ingressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
				ingressReq.SecurityGroupId = common.StringPtr(newSgId)
				ingressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
					Ingress: ingressRules,
				}
				for retry := 1; retry <= 3; retry++ {
					if _, addPolicyErr = vpcCli.CreateSecurityGroupPolicies(ingressReq); addPolicyErr == nil {
						break
					}
					log.Warn("ensureEgressRulesCore: 为新安全组添加入站规则失败",
						"newSgId", newSgId, "retry", retry, "err", addPolicyErr)
					if retry < 3 {
						time.Sleep(2 * time.Second)
					}
				}
				if addPolicyErr != nil {
					log.Warn("ensureEgressRulesCore: 为新安全组添加入站规则重试3次均失败，回滚并跳过绑定",
						"newSgId", newSgId, "err", addPolicyErr)
					// 回滚：仅当 DB 中仍是自己写入的值时才清空，防止并发场景下覆盖其他实例写入的值
					rollbackResult := model.DB(ctx).Model(&model.SiteConfig{}).
						Where("auto_created_security_group_id = ?", newSgId).
						Update("auto_created_security_group_id", "")
					if rollbackResult.RowsAffected == 1 {
						// 确认是自己写入的值被清空，才能安全删除安全组
						delReq := vpc.NewDeleteSecurityGroupRequest()
						delReq.SecurityGroupId = common.StringPtr(newSgId)
						if _, delErr := vpcCli.DeleteSecurityGroup(delReq); delErr != nil {
							log.Warn("ensureEgressRulesCore: 回滚删除安全组失败", "sgId", newSgId, "err", delErr)
						} else {
							log.Info("ensureEgressRulesCore: 回滚已删除安全组", "sgId", newSgId)
						}
					} else {
						log.Warn("ensureEgressRulesCore: 回滚时安全组已被其他实例使用，跳过删除", "sgId", newSgId)
					}
					return
				}
			}
		}
	}

	if newSgId == "" {
		log.Warn("ensureEgressRulesCore: 新安全组 ID 为空，跳过绑定", "instanceId", instanceId)
		return
	}

	// 第二步：先绑定新安全组，确保实例始终有安全组保护
	assocReq := cvm.NewAssociateSecurityGroupsRequest()
	assocReq.SecurityGroupIds = common.StringPtrs([]string{newSgId})
	assocReq.InstanceIds = common.StringPtrs([]string{instanceId})
	if _, err := cvmCli.AssociateSecurityGroups(assocReq); err != nil {
		log.Warn("ensureEgressRulesCore: 绑定新安全组失败，跳过解绑旧安全组",
			"instanceId", instanceId, "newSgId", newSgId, "err", err)
		return
	}

	// 第三步：解绑所有旧安全组（排除刚绑定的新安全组）
	for _, sgId := range sgIds {
		if sgId == newSgId {
			continue
		}
		disReq := cvm.NewDisassociateSecurityGroupsRequest()
		disReq.SecurityGroupIds = common.StringPtrs([]string{sgId})
		disReq.InstanceIds = common.StringPtrs([]string{instanceId})
		if _, err := cvmCli.DisassociateSecurityGroups(disReq); err != nil {
			log.Warn("ensureEgressRulesCore: 解绑旧安全组失败（新安全组已绑定，实例仍受保护）",
				"instanceId", instanceId, "sgId", sgId, "err", err)
		} else {
			log.Info("ensureEgressRulesCore: 已解绑旧安全组",
				"instanceId", instanceId, "oldSgId", sgId, "newSgId", newSgId)
		}
	}
	log.Info("ensureEgressRulesCore: 安全组替换完成，实例现只绑定自动安全组",
		"instanceId", instanceId, "newSgId", newSgId)
}

// HandleRetryInstance POST /openclaw/retry - 加载失败重试
func HandleRetryInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	if r.Method != http.MethodPost {
		log.Warn("[RetryInstance] 非法方法", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Warn("[RetryInstance] 获取实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	log.Info("[RetryInstance] 收到重试请求", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "name", instance.Name, "current_operation", instance.CurrentOperation)

	// 前置检查：只有 load_failed 状态才能重试
	// 使用与 /openclaw/status 一致的口径：UserStatusMap[load_failed] 包含 retry
	cvmInfo, _ := fetchCVMInstanceInfo(r.Context(), instance.InstanceId)
	currentStatus := ResolveInstanceStatus(r.Context(), instance, cvmInfo, nil)
	if currentStatus.Status != model.StatusLoadFailed {
		log.Warn("[RetryInstance] 当前状态不允许重试", "user_id", user.ID, "instance_id", instance.ID, "status", currentStatus.Status)
		key, args := agentStatusRejectMessage(currentStatus)
		writeError(w, r, http.StatusConflict, hcommon.I18nError(key, args...))
		return
	}

	// 根据之前的 currentOperation 决定重试方式
	// create/reboot/"" → 重启；reinstall → 重装
	operation := instance.CurrentOperation
	switch operation {
	case model.OpCreate, model.OpNone:
		// create 超时重试：改用 reboot 操作，避免状态映射引擎继续走 create 分支
		// OpNone 和 "" 均为空字符串，此处统一处理
		operation = model.OpReboot
	case model.OpReinstall:
		// reinstall 超时重试：继续重装
	default:
		operation = model.OpReboot
	}
	log.Info("[RetryInstance] 决定重试操作", "user_id", user.ID, "instance_id", instance.ID, "origin_operation", instance.CurrentOperation, "retry_operation", operation)

	// 重试时允许覆盖操作标记
	if err := setOperationForRetry(model.DB(r.Context()), instance, operation); err != nil {
		log.Warn("[RetryInstance] 写入重试操作标记失败", "user_id", user.ID, "instance_id", instance.ID, "operation", operation, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 根据 operation 类型调用不同的 CVM API
	client, err := NewCVMClient(ctx)
	if err != nil {
		log.Error("[RetryInstance] 创建 CVM 客户端失败", "user_id", user.ID, "instance_id", instance.ID, "error", err)
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	switch operation {
	case model.OpReboot:
		req := cvm.NewRebootInstancesRequest()
		req.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
		log.Info("[RetryInstance] 调用 CVM RebootInstances", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId)
		if _, err := client.RebootInstances(req); err != nil {
			log.Error("[RetryInstance] CVM 重启失败", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
			clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRebootInstanceFailed))
			return
		}
	case model.OpReinstall:
		enabledImage := model.GetEnabledImage(r.Context())
		if enabledImage == nil {
			log.Warn("[RetryInstance] 未启用任何镜像，无法重装", "user_id", user.ID, "instance_id", instance.ID)
			clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgNoEnabledImage))
			return
		}
		req := cvm.NewResetInstanceRequest()
		req.InstanceId = common.StringPtr(instance.InstanceId)
		req.ImageId = common.StringPtr(enabledImage.ImageId)
		// 简化：重试时不重新渲染 UserData
		log.Info("[RetryInstance] 调用 CVM ResetInstance", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "image_id", enabledImage.ImageId)
		if _, err := client.ResetInstance(req); err != nil {
			log.Error("[RetryInstance] CVM 重装失败", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
			clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgReinstallInstanceFailed))
			return
		}
		// 重装后恢复 SMH 个人空间环境（前置检查 + 重置 DB 状态 + 异步等待 CVM 就绪后触发安装）
		go syncSMHEnvWhenReadyFn(hcommon.DetachContext(r.Context()), *instance)
	default:
		// 创建或其他操作的重试，简单重启
		req := cvm.NewRebootInstancesRequest()
		req.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
		log.Info("[RetryInstance] 调用 CVM RebootInstances（default 分支）", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "operation", operation)
		if _, err := client.RebootInstances(req); err != nil {
			log.Error("[RetryInstance] CVM 重启失败（default 分支）", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
			clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRebootInstanceFailed))
			return
		}
	}
	clearAdjustmentFailure(r.Context(), instance.ID)
	log.Info("[RetryInstance] 重试请求已下发成功", "user_id", user.ID, "instance_id", instance.ID, "cvm_id", instance.InstanceId, "operation", operation)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// cvmRunInstancesHTTPStatus maps tag validation failures to 400 only when the
// request supplied custom tags. Requests without custom tags retain the legacy
// 500 response semantics because any tag failure comes from managed defaults.
func cvmRunInstancesHTTPStatus(err error, customTagsProvided bool) int {
	if !customTagsProvided {
		return http.StatusInternalServerError
	}
	var sdkErr *sdkerrors.TencentCloudSDKError
	if !errors.As(err, &sdkErr) {
		return http.StatusInternalServerError
	}
	switch sdkErr.GetCode() {
	case "FailedOperation.IllegalTagKey",
		"FailedOperation.IllegalTagValue",
		"FailedOperation.TagKeyReserved",
		"InvalidParameterValue.DuplicateTags",
		"InvalidParameterValue.TagKeyNotFound",
		"InvalidParameterValue.TagQuotaLimitExceeded":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// cvmRunInstancesError 将 RunInstances SDK 错误转换为用户友好错误。
func cvmRunInstancesError(err error, isAdmin bool) error {
	var sdkErr *sdkerrors.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		switch sdkErr.GetCode() {
		case "LimitExceeded.CvmInstanceQuota",
			"LimitExceeded.InstanceQuota",
			"LimitExceeded.PrepayQuota",
			"LimitExceeded.PrepayUnderwriteQuota",
			"LimitExceeded.SpotQuota",
			"LimitExceeded.UserSpotQuota":
			msg := i18n.MsgCreateCapacityLimited
			if isAdmin {
				msg = i18n.MsgCreateCapacityLimitedAdmin
			}
			return hcommon.I18nError(msg).WithDetail(sdkErr.GetMessage())
		case "LimitExceeded.SecurityGroupInstanceCount",
			"LimitExceeded.CvmsVifsPerSecGroupLimitExceeded",
			"LimitExceeded.AssociateUSGLimitExceeded",
			"LimitExceeded.SingleUSGQuota":
			msg := i18n.MsgPlatformCapacityLimited
			if isAdmin {
				msg = i18n.MsgPlatformCapacityLimitedAdmin
			}
			return hcommon.I18nError(msg).WithDetail(sdkErr.GetMessage())
		}
	}
	return hcommon.I18nError(i18n.MsgCreateCVMInstanceFailed).WithDetail(err.Error())
}

// cvmTerminateInstancesError 将 TerminateInstances SDK 错误转换为用户友好错误。
func cvmTerminateInstancesError(err error, isAdmin bool) error {
	var sdkErr *sdkerrors.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		if sdkErr.GetCode() == "LimitExceeded.UserReturnQuota" {
			msg := i18n.MsgDeleteCapacityLimited
			if isAdmin {
				msg = i18n.MsgDeleteCapacityLimitedAdmin
			}
			return hcommon.I18nError(msg).WithDetail(sdkErr.GetMessage())
		}
	}
	return hcommon.I18nError(i18n.MsgTerminateCVMInstanceFailed).WithDetail(err.Error())
}

// batchFetchRoleNames 批量查询实例关联的角色名称，返回 roleID → roleName 映射。
// roleID 为 0 时映射为 "通用助手"。
func batchFetchRoleNames(ctx context.Context, instances []model.Instance) map[uint]string {
	result := map[uint]string{0: i18n.T(ctx, i18n.MsgGeneralAssistant)}

	// 收集所有非零 roleID
	roleIDSet := make(map[uint]bool)
	for _, inst := range instances {
		if inst.RoleID > 0 {
			roleIDSet[inst.RoleID] = true
		}
	}
	if len(roleIDSet) == 0 {
		return result
	}

	roleIDs := make([]uint, 0, len(roleIDSet))
	for id := range roleIDSet {
		roleIDs = append(roleIDs, id)
	}

	var roles []model.OpenClawRole
	model.DB(ctx).Where("id IN ?", roleIDs).Find(&roles)
	for _, role := range roles {
		result[role.ID] = role.Name
	}

	// 角色已被删除的情况：回退为 "通用助手"
	for id := range roleIDSet {
		if _, ok := result[id]; !ok {
			result[id] = i18n.T(ctx, i18n.MsgGeneralAssistant)
		}
	}

	return result
}
