package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ─── 可替换函数变量（方便单元测试 mock） ──────────────────────────────────
var (
	// 内部使用（doctor.go 内部调用）
	doctorRunScriptFn              = RunScript
	doctorRunScriptAsyncFn         = RunScriptAsync
	doctorDescribeInvocationTaskFn = DescribeInvocationTask
	doctorNewCVMClientFn           = NewCVMClient
	doctorRequestSTSFn             = RequestInstanceScopedSTS
	doctorFetchCVMInfoFn           = fetchCVMInstanceInfo
	doctorLookupRuntimeUserFn      = LookupRuntimeUser
	doctorCheckGatewayReadyFn      = checkGatewayReady
	doctorInjectDefaultModelFn     = injectDefaultModel
	doctorDeleteSMHFileFn          = DeleteSMHCommonFile
	doctorUploadArchiveFn          = UploadArchiveToSMH
	doctorUploadArchiveKeyFn       = UploadArchiveToSMHWithKey
	doctorBuildSMHDownloadFn       = BuildCommonSMHDownloadURL
	doctorInstallComponentsFn      = installDoctorComponents
	doctorCheckAgentOnlineFn       = checkDoctorAgentOnline
	doctorResolveZoneFn            = resolveDoctorZone
	doctorDescribeSubnetZoneFn     = describeSubnetZoneViaVPC
	doctorApproveDeviceAsyncFn     = approveDeviceAsync
	doctorRestartTargetGatewayFn   = restartDoctorTargetGateway

	// 导出变量（供 task 包等外部调用方 mock）
	GetDoctorSessionMtimeFn = GetDoctorSessionMtime
	RefreshDoctorSTSFn      = RefreshDoctorSTS
	CleanupDoctorSessionFn  = CleanupDoctorSession
	DeleteSMHCommonFileFn   = DeleteSMHCommonFile
)

// ==================== 一键修复 ====================

// doctorMemoryContent 是龙虾医生节点的 IDENTITY.md 人设内容。
const doctorMemoryContent = `你是「龙虾医生」，一台由 ClawPro 平台临时创建的诊断专用 Agent，专门负责检测和修复用户的 Agent 运行问题。任务结束后你将自动下线，如果用户超过 10 分钟没有任何操作，你也将自动下线。

你是一位专业、沉稳的 Agent 运维专家。回复简洁直接，像一位靠谱的技术同事。任何修改操作必须先获得用户明确授权才能执行，执行前说清楚要做什么，执行后反馈结果。遇到无法解决的问题，坦诚说明并给出建议。

如果用户问你是什么，你可以说：我是 ClawPro 平台为您临时创建的一台诊断专用 Agent，专门帮您检测和修复当前 Agent 的运行问题，诊断结束后会自动下线，若超过 10 分钟无操作，也会自动下线。
`

// doctorMemoryB64 是 doctorMemoryContent 的 Base64 编码。
var doctorMemoryB64 = base64.StdEncoding.EncodeToString(
	[]byte(doctorMemoryContent))

// HandleDoctorQuickFix POST /openclaw/doctor/quick-fix
// 异步下发 openclaw doctor --fix --yes 命令，立即返回 invocation_id，
// 前端通过 GET /openclaw/doctor/quick-fix/status 轮询执行结果。
func HandleDoctorQuickFix(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()
	log := Logger(ctx)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}


	log.Info("[Doctor] 开始一键修复（异步下发）",
		"instance_id", instance.InstanceId,
		"user_id", user.ID)

	invocationId, err := doctorRunScriptAsyncFn(
		ctx,
		instance.InstanceId,
		"openclaw_doctor_fix.sh",
		300,
		instance.RuntimeUser,
		nil,
	)
	if err != nil {
		msg := hcommon.ErrorMessageWithCtx(r.Context(), err)
		detail := hcommon.ErrorDetailWithCtx(r.Context(), err)
		log.Error("[Doctor] 一键修复下发失败",
			"instance_id", instance.InstanceId,
			"error", msg,
			"detail", detail)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "fix_failed",
			"message": msg,
			"output":  detail,
		})
		return
	}

	log.Info("[Doctor] 一键修复命令已下发",
		"instance_id", instance.InstanceId,
		"invocation_id", invocationId)
	jsonOK(w, map[string]interface{}{
		"ok":            true,
		"invocation_id": invocationId,
	})
}

// HandleDoctorQuickFixStatus GET /openclaw/doctor/quick-fix/status
// 查询一键修复命令的执行状态，前端轮询使用。
func HandleDoctorQuickFixStatus(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()
	log := Logger(ctx)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 校验实例归属（确保用户只能查询自己实例的修复结果）
	_, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	invocationId := r.URL.Query().Get("invocation_id")
	if invocationId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorMissingInvocationID))
		return
	}

	result, err := doctorDescribeInvocationTaskFn(ctx, invocationId)
	if err != nil {
		msg := hcommon.ErrorMessageWithCtx(r.Context(), err)
		log.Error("[Doctor] 查询一键修复状态失败",
			"invocation_id", invocationId,
			"error", msg,
		)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "query_failed",
			"message": msg,
		})
		return
	}

	log.Debug("[Doctor] 一键修复状态查询",
		"invocation_id", invocationId,
		"status", result.Status,
		"finished", result.Finished)

	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"status":    result.Status,
		"output":    result.Output,
		"exit_code": result.ExitCode,
		"finished":  result.Finished,
	})
}

// ==================== 功能状态 ====================

// HandleDoctorFeature GET /openclaw/doctor/feature
func HandleDoctorFeature(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	config := model.GetSiteConfig(ctx)

	// 按实例分组策略解析龙虾医生开关（全局 DoctorEnabled 为兜底）
	doctorEnabled := usergroup.ResolvePolicyBoolForGroup(
		ctx, usergroup.PolicyKeyLobsterDoctor, instance.GroupID, config.DoctorEnabled)

	var authCount int64
	model.DB(ctx).Model(&model.DoctorAuthorization{}).
		Where("user_id = ? AND instance_id = ?",
			user.ID, instance.ID).
		Count(&authCount)

	jsonOK(w, map[string]interface{}{
		"ok":             true,
		"doctor_enabled": doctorEnabled,
		"authorized":     authCount > 0,
	})
}

// HandleDoctorAuthorize POST /openclaw/doctor/authorize
// 用户首次使用龙虾医生时确认授权，记录到独立表，后续不再弹窗。
func HandleDoctorAuthorize(
	w http.ResponseWriter, r *http.Request,
) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}


	// 幂等：已存在则跳过
	var existing model.DoctorAuthorization
	if model.DB(ctx).Where("user_id = ? AND instance_id = ?",
		user.ID, instance.ID).
		First(&existing).Error == nil {
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"message": i18n.T(r.Context(), i18n.MsgDoctorAlreadyAuthorized),
		})
		return
	}

	auth := model.DoctorAuthorization{
		UserID:     user.ID,
		InstanceID: instance.ID,
	}
	if err := model.DB(ctx).Create(&auth).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDoctorCreateAuthRecordFail))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"message": i18n.T(r.Context(), i18n.MsgDoctorAuthorizeSuccess),
	})
}

// ==================== 创建诊断会话 ====================

// HandleDoctorStart POST /openclaw/doctor/start
// 同步完成 CVM 创建并落库，后续等待就绪+激活由定时任务推进。
func HandleDoctorStart(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()
	log := Logger(ctx)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}


	log.Info("[Doctor] 收到创建诊断请求",
		"user_id", user.ID,
		"instance_id", instance.ID,
		"instance_cvm_id", instance.InstanceId,
		"agent_type", instance.AgentType)

	// 解析请求体（body 可选：空 body 表示 snapshot=false）
	var body struct {
		Snapshot bool `json:"snapshot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON).WithDetail(err.Error()))
		return
	}

	// 当前仅支持 OpenClaw 类型的实例
	if instance.AgentType != "" &&
		instance.AgentType != model.AgentTypeOpenClaw {
		log.Info("[Doctor] 不支持的实例类型",
			"agent_type", instance.AgentType)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "unsupported_agent_type",
			"message": i18n.T(r.Context(), i18n.MsgDoctorOnlyOpenClaw),
		})
		return
	}

	config := model.GetSiteConfig(ctx)

	// 按实例分组策略解析龙虾医生开关
	doctorEnabled := usergroup.ResolvePolicyBoolForGroup(
		ctx, usergroup.PolicyKeyLobsterDoctor, instance.GroupID, config.DoctorEnabled)
	if !doctorEnabled {
		log.Info("[Doctor] 功能未开启（分组策略）",
			"instance_group_id", instance.GroupID)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "doctor_disabled",
			"message": i18n.T(r.Context(), i18n.MsgDoctorFeatureDisabled),
		})
		return
	}

	// 后端兑底校验授权：避免绕过前端弹窗直接 POST /start
	// 授权表记录用户同意“龙虾医生会读实例上的对话历史与文件”，
	// 严格要求创建会话前必须已调过 /authorize
	var authCount int64
	if err := model.DB(ctx).Model(&model.DoctorAuthorization{}).
		Where("user_id = ? AND instance_id = ?", user.ID, instance.ID).
		Count(&authCount).Error; err != nil {
		log.Error("[Doctor] 查询授权记录失败",
			"user_id", user.ID,
			"instance_id", instance.ID,
			"error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDoctorQueryAuthRecordFailed).WithDetail(err.Error()))
		return
	}
	if authCount == 0 {
		log.Info("[Doctor] 用户未授权该实例，拒绝创建会话",
			"user_id", user.ID,
			"instance_id", instance.ID)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "not_authorized",
			"message": i18n.T(r.Context(), i18n.MsgDoctorNotAuthorized),
		})
		return
	}

	// 安全组：优先用目标实例的，兜底用全局
	securityGroupId := instance.SecurityGroupId
	if securityGroupId == "" {
		securityGroupId = config.SecurityGroupId
	}
	if securityGroupId == "" {
		log.Info("[Doctor] 安全组未配置")
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "security_group_not_set",
			"message": i18n.T(r.Context(), i18n.MsgDoctorSGNotConfigured),
		})
		return
	}

	// 事务 + 行锁：锁住 user 行，序列化同一用户的并发创建请求
	var session model.DoctorSession
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedUser model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&lockedUser, user.ID).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgLockUserRecordFailed)
		}

		var activeCount int64
		tx.Model(&model.DoctorSession{}).
			Where("user_id = ? AND status IN ?", user.ID,
				[]string{
					model.DoctorStatusCreating,
					model.DoctorStatusActive,
					model.DoctorStatusEnding,
				}).
			Count(&activeCount)
		if activeCount > 0 {
			return hcommon.I18nError(i18n.MsgDoctorActiveSessionExists)
		}

		session = model.DoctorSession{
			UserID:            user.ID,
			TargetInstanceID:  instance.ID,
			Status:            model.DoctorStatusCreating,
			SnapshotRequested: body.Snapshot,
		}
		if err := tx.Create(&session).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDoctorCreateSessionFailed)
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, hcommon.I18nError(i18n.MsgDoctorActiveSessionExists)) {
			log.Info("[Doctor] 已有进行中的会话")
			jsonOK(w, map[string]interface{}{
				"ok":      false,
				"error":   "active_session_exists",
				"message": i18n.T(r.Context(), i18n.MsgDoctorActiveSessionExists),
			})
		} else {
			var richErr *hcommon.RichError
			if errors.As(txErr, &richErr) {
				writeError(w, r, http.StatusInternalServerError, richErr)
			} else {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgOperationFailed))
			}
		}
		return
	}

	log.Info("[Doctor] 诊断会话已创建",
		"session_id", session.ID)

	// ---- 同步完成 CVM 创建，失败则标记 failed 并清理 ----

	// 1. 生成 ProxyToken
	log.Info("[Doctor] 生成 ProxyToken")
	proxyToken, err := model.GenerateProxyToken()
	if err != nil {
		log.Error("[Doctor] 生成 ProxyToken 失败", "error", err)
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgDoctorGenProxyTokenFailed),
		})
		return
	}

	// 2. 创建 instances 表记录
	log.Info("[Doctor] 创建实例记录")
	doctorInstance := model.Instance{
		Name:         fmt.Sprintf("龙虾医生-%d", session.ID),
		UserID:       user.ID,
		ProxyToken:   &proxyToken,
		AIModelID:    config.DefaultModelID,
		AgentType:    model.AgentTypeOpenClaw,
		IsDoctorNode: true,
		LastCVMState: "PENDING",
	}
	if err := model.DB(ctx).Create(&doctorInstance).Error; err != nil {
		log.Error("[Doctor] 创建实例记录失败", "error", err)
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgDoctorCreateInstanceFailed),
		})
		return
	}
	model.DB(ctx).Model(&session).Update("doctor_instance_id", doctorInstance.ID)

	// 后续失败时清理占位记录
	cvmCreated := false
	defer func() {
		if !cvmCreated {
			cleanupDoctorInstance(ctx, doctorInstance.ID)
		}
	}()

	// 3. 申请实例级 STS 临时密钥
	log.Info("[Doctor] 申请 STS 临时密钥",
		"target_cvm_id", instance.InstanceId)
	stsCred, err := doctorRequestSTSFn(ctx, instance.InstanceId)
	if err != nil {
		log.Error("[Doctor] 申请 STS 临时密钥失败", "error", err)
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgDoctorRequestSTSFailed),
		})
		return
	}
	model.DB(ctx).Model(&session).
		Update("sts_expired_at", time.Now().Unix()+7200)

	// 4. 获取启用镜像
	log.Info("[Doctor] 获取启用镜像")
	enabledImage, err := model.GetEnabledImageByType(
		ctx, model.AgentTypeOpenClaw)
	if err != nil {
		log.Error("[Doctor] 获取启用镜像失败", "error", err)
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgDoctorGetEnabledImageFailed),
		})
		return
	}

	// 5. CVM RunInstances（同步调用）
	log.Info("[Doctor] 创建 CVM 实例",
		"image_id", enabledImage.ImageId)
	client, err := doctorNewCVMClientFn(ctx)
	if err != nil {
		log.Error("[Doctor] 创建 CVM 客户端失败", "error", err)
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgCreateCVMClientFailed),
		})
		return
	}

	request, networkInfo, err := buildDoctorCVMRequest(
		ctx, &config, instance, proxyToken, stsCred,
		enabledImage.ImageId)
	if err != nil {
		log.Error("[Doctor] 构建 CVM 请求失败", "error", err)
		model.DB(ctx).Model(&session).
			Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "build_request_failed",
			"message": hcommon.ErrorMessageWithCtx(ctx, err),
		})
		return
	}
	response, cerr := CallSDKAPITyped(
		ctx, SDKComponentCVM, request, client.RunInstances)
	if cerr != nil {
		log.Error("[Doctor] CVM RunInstances 失败", "error", cerr)
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgDoctorCVMCreateFailed),
		})
		return
	}

	var cvmInstanceId string
	if response.Response != nil &&
		len(response.Response.InstanceIdSet) > 0 {
		cvmInstanceId = *response.Response.InstanceIdSet[0]
	}
	if cvmInstanceId == "" {
		log.Error("[Doctor] CVM 返回空 InstanceId")
		model.DB(ctx).Model(&session).Update("status", model.DoctorStatusFailed)
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "create_failed",
			"message": i18n.T(r.Context(), i18n.MsgDoctorCVMReturnEmptyID),
		})
		return
	}

	// CVM 创建成功，更新实例记录
	cvmCreated = true
	now := time.Now()
	model.DB(ctx).Model(&doctorInstance).
		Updates(map[string]interface{}{
			"instance_id":                  cvmInstanceId,
			"vpc_id":                       networkInfo.VpcId,
			"subnet_id":                    networkInfo.SubnetId,
			"security_group_id":            networkInfo.SecurityGroupId,
			"current_operation":            model.OpCreate,
			"current_operation_state":      model.OpStateProcessing,
			"current_operation_updated_at": &now,
			"last_known_status":            model.StatusCreating, // P3 操作即时写
			"status_synced_at":             now,                  // 防止后台覆盖
		})

	log.Info("[Doctor] CVM 创建成功，等待定时任务激活",
		"session_id", session.ID,
		"cvm_instance_id", cvmInstanceId,
		"doctor_instance_db_id", doctorInstance.ID)

	// 不再启动 goroutine，后续由定时任务推进
	jsonOK(w, map[string]interface{}{
		"ok": true,
	})
}

// doctorNetworkInfo 保存 buildDoctorCVMRequest 实际使用的网络参数，
// 用于在 CVM 创建成功后回写到 doctor instance 记录。
type doctorNetworkInfo struct {
	VpcId           string
	SubnetId        string
	SecurityGroupId string
}

// buildDoctorCVMRequest 构建龙虾医生节点的 CVM 请求。
// 网络配置直接复用目标实例已落库的 vpc_id/subnet_id/security_group_id，
// 不重新走分组解析链；兜底使用全局 site_config。
// 同时返回实际使用的网络参数，供调用方回写 DB。
func buildDoctorCVMRequest(
	ctx context.Context,
	config *model.SiteConfig,
	targetInstance *model.Instance,
	proxyToken string,
	stsCred *STSCredentials,
	imageId string,
) (*cvm.RunInstancesRequest, doctorNetworkInfo, error) {
	request := cvm.NewRunInstancesRequest()

	request.ImageId = common.StringPtr(imageId)
	request.InstanceCount = common.Int64Ptr(1)
	request.InstanceName = common.StringPtr("龙虾医生")
	request.InstanceChargeType = common.StringPtr(
		"POSTPAID_BY_HOUR")

	// 机型：复用 SiteConfig.CVMTemplate 中的 InstanceType，跟普通实例创建保持一致。
	// 解析失败 / 未设置 → 报错拒绝创建（与普通实例行为一致）。
	instanceType, err := parseCVMTemplateInstanceType(config.CVMTemplate)
	if err != nil {
		return nil, doctorNetworkInfo{},
			hcommon.I18nRichError(err, i18n.MsgCVMTemplateConfigError)
	}
	if instanceType == "" {
		return nil, doctorNetworkInfo{},
			hcommon.I18nError(i18n.MsgDoctorCVMTemplateNotConfigured)
	}
	request.InstanceType = common.StringPtr(instanceType)
	request.SystemDisk = &cvm.SystemDisk{
		DiskType: common.StringPtr("CLOUD_BSSD"),
		DiskSize: common.Int64Ptr(50),
	}

	// 网络：优先使用目标实例已落库的 VPC/子网/安全组，确保同 VPC 互通
	vpcId := targetInstance.VpcId
	subnetId := targetInstance.SubnetId
	securityGroupId := targetInstance.SecurityGroupId

	// 兜底：目标实例字段为空时使用全局配置
	if vpcId == "" {
		vpcId = config.VpcId
	}
	if subnetId == "" {
		// 全局 subnetMap 取第一个可用子网
		subnetMap := config.GetSubnetMap()
		for _, sids := range subnetMap {
			if len(sids) > 0 {
				subnetId = sids[0]
				break
			}
		}
	}
	if securityGroupId == "" {
		securityGroupId = config.SecurityGroupId
	}

	// 反查 subnet 所在可用区：先查 config，查不到调 VPC API 兑底。
	// 仍查不到返回 error（CVM API 要求 Placement.Zone 必填）。
	zone := doctorResolveZoneFn(ctx, subnetId, config)
	if zone == "" {
		return nil, doctorNetworkInfo{},
			hcommon.I18nError(i18n.MsgDoctorSubnetZoneNotFound, subnetId)
	}
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(zone),
	}
	if vpcId != "" && subnetId != "" {
		request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{
			VpcId:    common.StringPtr(vpcId),
			SubnetId: common.StringPtr(subnetId),
		}
	}

	// 安全组
	request.SecurityGroupIds = common.StringPtrs(
		[]string{securityGroupId})

	// 公网
	request.InternetAccessible = &cvm.InternetAccessible{
		InternetChargeType: common.StringPtr(
			"TRAFFIC_POSTPAID_BY_HOUR"),
		InternetMaxBandwidthOut: common.Int64Ptr(5),
		PublicIpAssigned:        common.BoolPtr(true),
	}

	// TAT
	request.EnhancedService = &cvm.EnhancedService{
		AutomationService: &cvm.RunAutomationServiceEnabled{
			Enabled: common.BoolPtr(true),
		},
	}

	// UserData：注入临时密钥和目标实例信息。
	// 调试日志：记录传入 UserData 的目标实例关键字段，防止“检测到
	// 环境变量 DOCTOR_TARGET_AGENT_TYPE 为空”这类问题时无法追查源头。
	log := Logger(ctx)
	log.Info("[Doctor] 构造 UserData 参数",
		"target_instance_pk", targetInstance.ID,
		"target_instance_id", targetInstance.InstanceId,
		"target_agent_type", targetInstance.AgentType,
		"target_agent_type_empty", targetInstance.AgentType == "",
		"tat_region", CVMRegion)

	userData := fmt.Sprintf(`#!/bin/bash
cat >> /etc/environment <<'EOF'
DOCTOR_TARGET_INSTANCE_ID="%s"
TAT_REGION="%s"
TEMP_SECRET_ID="%s"
TEMP_SECRET_KEY="%s"
TEMP_TOKEN="%s"
DOCTOR_TARGET_AGENT_TYPE="%s"
EOF
`, targetInstance.InstanceId, CVMRegion,
		stsCred.SecretId, stsCred.SecretKey, stsCred.Token,
		targetInstance.AgentType)

	request.UserData = common.StringPtr(
		base64.StdEncoding.EncodeToString([]byte(userData)))

	return request, doctorNetworkInfo{
		VpcId:           vpcId,
		SubnetId:        subnetId,
		SecurityGroupId: securityGroupId,
	}, nil
}

// lookupZoneBySubnetId 从全局 config 的 subnetMap 中反查 subnetId 对应的可用区。
// 纯函数，不调 VPC API；查不到返回空串，调用方可进一步走 VPC API 兑底。
func lookupZoneBySubnetId(ctx context.Context, subnetId string, config *model.SiteConfig) string {
	if subnetId == "" {
		return ""
	}
	// 先查主 subnetMap
	for zone, sids := range config.GetSubnetMap() {
		for _, sid := range sids {
			if sid == subnetId {
				return zone
			}
		}
	}
	// 再查默认 VPC 的 subnetMap
	for zone, sids := range config.GetDefaultSubnetMap() {
		for _, sid := range sids {
			if sid == subnetId {
				return zone
			}
		}
	}
	// 查 VpcConfig 表（分组 VPC 配置）
	var vpcConfigs []model.VpcConfig
	if model.DB(ctx).Find(&vpcConfigs).Error == nil {
		for _, vc := range vpcConfigs {
			subnetMap, _ := vc.GetSubnetMap()
			for zone, sids := range subnetMap {
				for _, sid := range sids {
					if sid == subnetId {
						return zone
					}
				}
			}
		}
	}
	return ""
}

// resolveDoctorZone 决定龙虾医生 CVM 的 Placement.Zone。
// 1. 先查 config 的 subnetMap（同步、零 API 开销）
// 2. 查不到时调 VPC DescribeSubnets API 反查（兑底历史/分组配置已变的子网）
// 3. 仍查不到 → 返回空串，调用方决定怎么处理
func resolveDoctorZone(ctx context.Context, subnetId string, config *model.SiteConfig) string {
	if subnetId == "" {
		return ""
	}
	if zone := lookupZoneBySubnetId(ctx, subnetId, config); zone != "" {
		return zone
	}
	// VPC API 兑底：subnet 不在任何 config subnetMap 中（可能历史/
	// 分组 VPC 配置变更），调 VPC API 实时查。
	return doctorDescribeSubnetZoneFn(ctx, subnetId)
}

// describeSubnetZoneViaVPC 调用 VPC DescribeSubnets API 反查 subnet 所在 zone。
// 抽出为独立函数变量，方便单测 mock。
func describeSubnetZoneViaVPC(ctx context.Context, subnetId string) string {
	vpcClient, err := GetVPCClient(ctx)
	if err != nil {
		slog.Warn("[Doctor] 创建 VPC 客户端失败，无法兑底反查 zone",
			"subnet_id", subnetId, "error", err)
		return ""
	}
	req := vpc.NewDescribeSubnetsRequest()
	req.SubnetIds = common.StringPtrs([]string{subnetId})
	resp, err := vpcClient.DescribeSubnets(req)
	if err != nil {
		slog.Warn("[Doctor] DescribeSubnets 失败，无法兑底反查 zone",
			"subnet_id", subnetId, "error", err)
		return ""
	}
	if resp.Response == nil {
		return ""
	}
	for _, s := range resp.Response.SubnetSet {
		if s.SubnetId != nil && *s.SubnetId == subnetId &&
			s.Zone != nil && *s.Zone != "" {
			slog.Info("[Doctor] 通过 VPC API 兑底查到 subnet zone",
				"subnet_id", subnetId, "zone", *s.Zone)
			return *s.Zone
		}
	}
	return ""
}

// parseCVMTemplateInstanceType 从 SiteConfig.CVMTemplate JSON 中提取 InstanceType 字段。

// 与普通实例创建路径（整个 request 反序列化）保持一致的错误语义：
//   - 模板为空：返回 ("", nil)，调用方可区分“没配”跟“配坏了”
//   - JSON 解析失败：返回 err
func parseCVMTemplateInstanceType(template string) (string, error) {
	if template == "" {
		return "", nil
	}
	var tpl struct {
		InstanceType string `json:"InstanceType"`
	}
	if err := json.Unmarshal([]byte(template), &tpl); err != nil {
		return "", err
	}
	return tpl.InstanceType, nil
}

// ActivateDoctorSession 检查单个 creating 状态的诊断会话，若节点已就绪则安装组件并激活。
// 返回 true 表示已激活或失败（不需再轮询），false 表示节点尚未就绪。
func ActivateDoctorSession(
	ctx context.Context,
	session *model.DoctorSession,
) bool {
	log := Logger(ctx).With("session_id", session.ID)
	log.Info("[Doctor] 开始激活会话", "session_status", session.Status)
	if session.DoctorInstanceID == nil {
		log.Error("[Doctor] 会话无 doctor_instance_id，跳过")
		return false
	}

	var doctorInst model.Instance
	if model.DB(ctx).First(
		&doctorInst, *session.DoctorInstanceID).Error != nil {
		log.Error("[Doctor] 龙虾医生实例记录不存在")
		model.DB(ctx).Model(session).
			Update("status", model.DoctorStatusFailed)
		return true
	}

	if doctorInst.InstanceId == "" {
		log.Error("[Doctor] CVM ID 为空，跳过")
		return false
	}

	// 主动查询 CVM 状态，更新 last_cvm_state（龙虾医生没有前端轮询触发 handleStatusSideEffects）
	if doctorInst.LastCVMState != "RUNNING" {
		cvmInfo, err := doctorFetchCVMInfoFn(
			ctx, doctorInst.InstanceId)
		if err != nil {
			log.Error("[Doctor] 查询 CVM 状态失败",
				"cvm_instance_id", doctorInst.InstanceId,
				"error", err)
			return false
		}

		// CVM 不存在或创建失败 → 标记 session 失败
		if cvmInfo == nil || cvmInfo.State == "" ||
			cvmInfo.State == "LAUNCH_FAILED" {
			state := "NOT_FOUND"
			if cvmInfo != nil {
				state = cvmInfo.State
			}
			log.Error("[Doctor] CVM 创建失败或不存在",
				"cvm_instance_id", doctorInst.InstanceId,
				"state", state)
			model.DB(ctx).Model(session).
				Update("status", model.DoctorStatusFailed)
			return true
		}

		log.Info("[Doctor] CVM 状态更新",
			"cvm_instance_id", doctorInst.InstanceId,
			"old_state", doctorInst.LastCVMState,
			"new_state", cvmInfo.State)
		model.DB(ctx).
			Model(&doctorInst).
			Update("last_cvm_state", cvmInfo.State)
		doctorInst.LastCVMState = cvmInfo.State

		if doctorInst.LastCVMState != "RUNNING" {
			log.Info("[Doctor] CVM 尚未 RUNNING",
				"cvm_instance_id", doctorInst.InstanceId,
				"state", doctorInst.LastCVMState)
			return false
		}
	}

	// 龙虾医生激活不允许依赖 instances.agent_ready 与 current_operation_state：
	// 这两个字段是 OpenClaw 通用调度字段，会被 AgentChecker、
	// handleStatusSideEffects 等周期 / 事件任务写入，如果被外部刷到
	// “就绪 / success”，会让激活流程跳过“组件安装 + 模型注入”。
	// 这里不读 / 不写这两个字段，每次都跳进组件安装 / 模型注入，脚本
	// 幂等保证重复执行无副作用。

	// CVM RUNNING 不代表 TAT Agent 已在线，主动检查一次。未上线时
	// 返回 false 等下一次定时重试，避免被 dispatchScript 内部的
	// "命令下发失败" 错误误判为 permanent failure 后直接 mark failed。
	if err := doctorCheckAgentOnlineFn(
		ctx, doctorInst.InstanceId); err != nil {
		log.Info("[Doctor] TAT Agent 尚未就绪，等下次重试",
			"cvm_instance_id", doctorInst.InstanceId,
			"error", err)
		return false
	}

	// 等待 OpenClaw Gateway 进程就绪（HTTP 层面）
	runtimeUser := doctorLookupRuntimeUserFn(ctx, doctorInst.InstanceId)
	if !doctorCheckGatewayReadyFn(ctx, doctorInst.InstanceId, runtimeUser) {
		return false
	}

	log.Info("[Doctor] 开始安装龙虾医生组件",
		"cvm_instance_id", doctorInst.InstanceId)

	// 安装龙虾医生组件（doctor-cli + Skill + IDENTITY.md）
	// 脚本幂等：重复执行无副作用（下载 → 解压 → 覆盖）
	if err := doctorInstallComponentsFn(
		ctx, doctorInst.InstanceId, runtimeUser); err != nil {
		log.Error("[Doctor] 安装龙虾医生组件失败",
			"error", err)
		model.DB(ctx).Model(session).
			Update("status", model.DoctorStatusFailed)
		return true
	}

	// 注入默认模型（同步执行）。
	// 进入此分支前 Gateway WS Health 已通过、TAT 必然 Online，agent_ready 检查
	// 第一次 polling 即返回（最多浪费一次 10s sleep），开销可控。
	// 同步执行的好处：注入失败立即标 failed，避免用户看到 "已激活但默认模型未生效"
	// 的中间态；极端情况（agent_ready 永不 ready）最多阻塞 10 分钟，由
	// doctor_activate task 入口的 10 分钟超时兜底。
	config := model.GetSiteConfig(ctx)
	if config.DefaultModelID > 0 {
		log.Info("[Doctor] 同步注入默认模型",
			"model_id", config.DefaultModelID)
		doctorInjectDefaultModelFn(
			ctx,
			*session.DoctorInstanceID, config.DefaultModelID)
	}

	// 自动批准首次连接产生的 pending device pairing 请求（保持异步）。
	// 否则前端 webchat connect 时会因 scope-upgrade 被 Gateway 1008 关闭，
	// 报 "pairing required: device is asking for more scopes than currently approved"。
	// 复用普通实例创建/重装的异步逻辑（含 TAT 上线检查 + agent_type guard）。
	go doctorApproveDeviceAsyncFn(
		hcommon.DetachContext(ctx),
		*session.DoctorInstanceID,
		doctorInst.InstanceId,
		runtimeUser,
	)

	// 如果请求了快照，先完成快照再标记 active（避免前端轮询到 active 时 has_snapshot 仍为 false）
	// 快照失败 → 标 failed，不要进入 active（避免用户看到 "连上了但没快照" 的中间态）
	if session.SnapshotRequested && !session.HasSnapshot {
		log.Info("[Doctor] 执行请求的快照")
		if err := createDoctorSnapshot(ctx, session); err != nil {
			log.Error("[Doctor] 快照创建失败，会话标为 failed",
				"error", err)
			model.DB(ctx).Model(session).
				Update("status", model.DoctorStatusFailed)
			return true // 不需再轮询，已是终态
		}
	}

	// final 校验：跳过中间步骤后，再做一个短轮询确认 Gateway 还是就绪状态。
	// 避免中间步骤（组件安装、inject、快照）期间 Gateway 重启 / 配置未生效造成
	// 短暂抖动。每 2 秒检查一次，最多 5 次；若仍未 ready，也不改状态，继续标 active。
	for attempt := 1; attempt <= 5; attempt++ {
		if doctorCheckGatewayReadyFn(
			ctx, doctorInst.InstanceId, runtimeUser) {
			break
		}
		if attempt < 5 {
			time.Sleep(2 * time.Second)
		}
	}

	activatedAt := time.Now()
	if err := model.DB(ctx).Model(session).Updates(map[string]interface{}{
		"status":       model.DoctorStatusActive,
		"activated_at": activatedAt,
	}).Error; err != nil {
		log.Error("[Doctor] 更新诊断会话激活状态失败",
			"error", err)
		return false
	}

	log.Info("[Doctor] 节点就绪，诊断会话已激活")

	return true
}

// checkDoctorAgentOnline 包装 NewTATClient + checkAgentOnline，方便单元测试 mock。
// 在 ActivateDoctorSession 中调用，用于区分 "TAT Agent 尚未上线"（transient）
// 与 "脚本本身执行失败"（permanent），避免前者被误判为 permanent failure
// 导致 session 被提前标记为 failed。
func checkDoctorAgentOnline(
	ctx context.Context, cvmInstanceId string,
) error {
	client, err := NewTATClient(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCreateTATClientFailed)
	}
	return checkAgentOnline(client, cvmInstanceId)
}

// checkGatewayReady 检查 OpenClaw Gateway 进程是否就绪（单次检查，不重试）。
// 由定时任务每分钟调用，未就绪时返回 false，下次轮询再检查。
func checkGatewayReady(
	ctx context.Context,
	cvmInstanceId string,
	runtimeUser string,
) bool {
	log := Logger(ctx)
	scriptName, rerr := ResolveScript(
		ctx, "check_ready", model.AgentTypeOpenClaw)
	if rerr != nil {
		log.Error("[Doctor] 无法解析 check_ready 脚本",
			"error", rerr)
		return false
	}

	output, err := doctorRunScriptFn(
		ctx, cvmInstanceId, scriptName, 30,
		runtimeUser, nil, nil)
	if err != nil {
		log.Info("[Doctor] Gateway 就绪检查脚本执行失败",
			"error", err)
		return false
	}

	var result struct {
		Ready  bool   `json:"ready"`
		Reason string `json:"reason,omitempty"`
	}
	if json.Unmarshal([]byte(output), &result) != nil {
		log.Info("[Doctor] Gateway 就绪检查输出解析失败",
			"output", output)
		return false
	}

	if !result.Ready {
		log.Info("[Doctor] Gateway 尚未就绪",
			"reason", result.Reason)
	}
	return result.Ready
}

func installDoctorComponents(
	ctx context.Context,
	cvmInstanceId string,
	runtimeUser string,
) error {
	log := Logger(ctx)
	// 1. 安装 doctor-cli（从公共 COS 桶下载）
	const doctorAssetBaseURL = "https://clawpro-feishu-1251783334.cos.ap-guangzhou.myqcloud.com/doctor-assets"
	cliURL := doctorAssetBaseURL + "/doctor-cli/doctor-cli-linux-amd64"
	if _, err := doctorRunScriptFn(
		ctx, cvmInstanceId, "install_doctor_cli.sh", 60,
		runtimeUser, nil,
		map[string]string{"download_url": cliURL},
	); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgDoctorInstallCLIFailed)
	}
	log.Info("[Doctor] doctor-cli 安装成功")

	// 2. 安装龙虾医生 Skill（从公共 COS 桶下载）
	skillURL := doctorAssetBaseURL + "/doctor-skill/dragon-doctor.zip"
	if _, err := doctorRunScriptFn(
		ctx, cvmInstanceId, "install_skill_from_smh.sh", 120,
		runtimeUser, nil,
		map[string]string{
			"download_url":  skillURL,
			"skill_slug":    "dragon-doctor",
			"skill_version": "1.0.0",
		},
	); err != nil {
		return hcommon.I18nRichError(
			err, i18n.MsgDoctorInstallSkillFailed)
	}
	log.Info("[Doctor] 龙虾医生 Skill 安装成功")

	// 3. 写入龙虾医生人设（IDENTITY.md）并删除 BOOTSTRAP.md 跳过引导
	if _, err := doctorRunScriptFn(
		ctx, cvmInstanceId, "write_doctor_identity.sh", 30,
		runtimeUser, nil,
		map[string]string{"content": doctorMemoryB64},
	); err != nil {
		// 人设写入失败不阻断整体流程，仅记录警告
		log.Error("[Doctor] 写入 IDENTITY.md 失败",
			"error", err)
	} else {
		log.Info("[Doctor] IDENTITY.md 写入成功，BOOTSTRAP.md 已删除")
	}

	return nil
}

// cleanupDoctorInstance 清理失败的龙虾医生实例记录。
func cleanupDoctorInstance(ctx context.Context, instanceID uint) {
	model.DB(ctx).Delete(&model.Instance{}, instanceID)
}

// ==================== 查询状态 ====================

// HandleDoctorStatus GET /openclaw/doctor/status?id=X
// 查询指定实例最新的诊断会话状态。
func HandleDoctorStatus(
	w http.ResponseWriter, r *http.Request,
) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instanceID, err := strconv.ParseUint(
		r.URL.Query().Get("id"), 10, 64)
	if err != nil || instanceID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var session model.DoctorSession
	result := model.DB(ctx).Where(
		"target_instance_id = ? AND user_id = ?",
		instanceID, user.ID).
		Order("id DESC").First(&session)
	if result.Error != nil {
		jsonOK(w, map[string]interface{}{
			"ok":                 true,
			"has_active_session": false,
		})
		return
	}

	data := map[string]interface{}{
		"ok": true,
		"has_active_session": session.Status != model.DoctorStatusEnded &&
			session.Status != model.DoctorStatusFailed,
		"status":       session.Status,
		"has_snapshot": session.HasSnapshot,
	}
	if session.DoctorInstanceID != nil {
		data["doctor_instance_db_id"] =
			*session.DoctorInstanceID
		// 返回 CVM 实例信息供前端 ClawPanel 使用
		var doctorInst model.Instance
		if model.DB(ctx).First(
			&doctorInst, *session.DoctorInstanceID).Error == nil {
			data["doctor_cvm_instance_id"] = doctorInst.InstanceId
			data["doctor_region"] = CVMRegion
		}
	}
	if session.Status == model.DoctorStatusFailed {
		data["error_message"] = "节点创建失败"
	}

	jsonOK(w, data)
}

// ==================== 快照 ====================

// createDoctorSnapshot 在目标实例上执行快照备份并上传到 SMH。
// 由激活定时任务在 snapshot_requested=true 时调用。
// 返回 error，调用方需检查。任何错误都会提前返回且不会设 has_snapshot=true。
func createDoctorSnapshot(
	ctx context.Context,
	session *model.DoctorSession,
) error {
	log := Logger(ctx).With("session_id", session.ID)
	var targetInstance model.Instance
	if err := model.DB(ctx).First(&targetInstance,
		session.TargetInstanceID).Error; err != nil {
		log.Error("[Doctor] 快照:目标实例不存在", "error", err)
		return hcommon.I18nRichError(err, i18n.MsgDoctorSnapshotTargetNotFound)
	}

	log.Info("[Doctor] 开始执行快照备份",
		"target_instance_cvm_id", targetInstance.InstanceId)
	backupOutput, err := doctorRunScriptFn(
		ctx,
		targetInstance.InstanceId,
		"backup_pre_reinstall.sh",
		600,
		targetInstance.RuntimeUser,
		nil, nil,
	)
	if err != nil {
		log.Error("[Doctor] 快照备份失败",
			"session_id", session.ID,
			"error", err)
		return hcommon.I18nRichError(err, i18n.MsgDoctorSnapshotBackupFailed)
	}

	// backup_pre_reinstall.sh 执行到这里说明已经成功停掉了目标实例的 Gateway
	// （脚本本身只停不拉，其设计前提是"备份后立即重装，重装恢复流程会重启 Gateway"）。
	// 龙虾医生快照场景不会重装目标实例，必须在此兜底重新拉起 Gateway，
	// 否则无论后续上传成功与否，目标实例的 Gateway 都会永久停在 stopped 状态。
	// 用 defer 保证兜底覆盖后续任意失败路径（提前 return 也会执行）。
	defer doctorRestartTargetGatewayFn(ctx, &targetInstance)

	archivePath := extractBackupDir(backupOutput)
	if archivePath == "" {
		log.Error("[Doctor] 无法获取备份压缩包路径")
		return hcommon.I18nError(i18n.MsgDoctorSnapshotArchivePathEmpty)
	}
	archiveSize := extractArchiveSize(backupOutput)

	log.Info("[Doctor] 开始上传快照到 SMH",
		"archive_path", archivePath,
		"archive_size", archiveSize)
	fileKey, rerr := doctorUploadArchiveFn(
		ctx,
		targetInstance.InstanceId,
		targetInstance.RuntimeUser,
		archivePath,
		archiveSize,
	)
	if rerr != nil {
		log.Error("[Doctor] 快照上传失败",
			"session_id", session.ID, "error", rerr)
		return hcommon.I18nRichError(rerr, i18n.MsgDoctorSnapshotUploadFailed)
	}

	model.DB(ctx).Model(&model.DoctorSession{}).
		Where("id = ?", session.ID).
		Updates(map[string]interface{}{
			"has_snapshot":      true,
			"snapshot_file_key": fileKey,
		})

	log.Info("[Doctor] 快照创建成功",
		"session_id", session.ID,
		"snapshot_file_key", fileKey)
	return nil
}

// restartDoctorTargetGateway 兜底重启被诊断目标实例的 Gateway。
// backup_pre_reinstall.sh 只停不拉（前提是调用方后续会重装实例），
// 龙虾医生快照场景不重装，必须显式拉起，否则目标实例 Gateway 会一直停着。
// 重启失败仅记录日志、不向上抛错：这是"最佳努力"兜底，不应影响快照本身的成功/失败结果，
// 避免因为重启这一步的抖动把一次已经成功的快照误判为失败。
func restartDoctorTargetGateway(ctx context.Context, targetInstance *model.Instance) {
	log := Logger(ctx).With(
		"target_instance_id", targetInstance.ID,
		"target_instance_cvm_id", targetInstance.InstanceId)
	log.Info("[Doctor] 快照后兜底重启目标实例 Gateway")
	if _, err := RunAgentScript(
		ctx, targetInstance, "restart_gateway", 60, nil, nil,
	); err != nil {
		log.Error("[Doctor] 快照后重启目标实例 Gateway 失败，需人工介入排查",
			"error", err)
		return
	}
	log.Info("[Doctor] 快照后目标实例 Gateway 已重启")
}

// ==================== 结束诊断 ====================

// HandleDoctorEnd POST /openclaw/doctor/end?id=X
// 异步结束诊断会话。接口立即标记 ending 状态并返回，
// 实际的回滚和资源清理由后台定时任务完成。
func HandleDoctorEnd(
	w http.ResponseWriter, r *http.Request,
) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ctx := r.Context()
	log := Logger(ctx)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}


	var body struct {
		Rollback bool `json:"rollback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON).WithDetail(err.Error()))
		return
	}

	log.Info("[Doctor] 收到结束诊断请求",
		"instance_id", instance.ID,
		"user_id", user.ID,
		"rollback", body.Rollback)

	var session model.DoctorSession
	if model.DB(ctx).
		Where("target_instance_id = ? AND user_id = ? AND status NOT IN ?",
			instance.ID, user.ID,
			[]string{model.DoctorStatusEnded, model.DoctorStatusFailed}).
		Order("id DESC").First(&session).Error != nil {
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "session_not_found",
			"message": i18n.T(r.Context(), i18n.MsgDoctorSessionNotFound),
		})
		return
	}
	if session.Status == model.DoctorStatusEnded ||
		session.Status == model.DoctorStatusFailed {
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"error":   "session_already_ended",
			"message": i18n.T(r.Context(), i18n.MsgDoctorSessionAlreadyEnded),
		})
		return
	}

	log.Info("[Doctor] 标记会话为 ending",
		"session_id", session.ID,
		"session_status", session.Status,
		"rollback_requested", body.Rollback)

	// 只做状态标记，立即返回
	model.DB(ctx).Model(&session).
		Updates(map[string]interface{}{
			"status":             model.DoctorStatusEnding,
			"rollback_requested": body.Rollback,
		})

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"status": model.DoctorStatusEnding,
	})
}

// terminateDoctorCVM 销毁龙虾医生 CVM 实例。
func terminateDoctorCVM(ctx context.Context, cvmInstanceId string) {
	log := slog.With("instance_id", cvmInstanceId)

	log.Info("[Doctor] 开始销毁 CVM")

	client, err := doctorNewCVMClientFn(ctx)
	if err != nil {
		log.Error("[Doctor] 创建 CVM 客户端失败",
			"error", err)
		return
	}
	req := cvm.NewTerminateInstancesRequest()
	req.InstanceIds = common.StringPtrs(
		[]string{cvmInstanceId})
	if _, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, client.TerminateInstances); err != nil {
		log.Error("[Doctor] 销毁 CVM 失败",
			"error", err)
	} else {
		log.Info("[Doctor] CVM 已销毁")
	}
}

// CleanupDoctorSession 公共的诊断会话清理逻辑：
// 上传 session 文件 → 删 SMH 快照 → 销毁 CVM → 删实例记录 → status=ended。
// HandleDoctorEnd 和 task/endDoctorSession 共用。
func CleanupDoctorSession(
	ctx context.Context,
	session *model.DoctorSession,
) {
	log := Logger(ctx).With("session_id", session.ID)
	log.Info("[Doctor] 开始清理诊断会话",
		"doctor_instance_id", session.DoctorInstanceID,
		"has_snapshot", session.HasSnapshot)

	// 上传 session 文件到 SMH（保留对话历史）
	if session.DoctorInstanceID != nil {
		var doctorInst model.Instance
		if model.DB(ctx).First(
			&doctorInst,
			*session.DoctorInstanceID).Error == nil &&
			doctorInst.InstanceId != "" {
			log.Info("[Doctor] 上传 session 文件",
				"cvm_instance_id", doctorInst.InstanceId)
			UploadDoctorSessions(
				ctx,
				doctorInst.InstanceId,
				session.UserID,
				session.TargetInstanceID)
		}
	}

	// 删除 SMH 快照备份
	if session.HasSnapshot && session.SnapshotFileKey != "" && !session.SnapshotDeleted {
		log.Info("[Doctor] 删除 SMH 快照备份",
			"file_key", session.SnapshotFileKey)
		if delErr := doctorDeleteSMHFileFn(
			ctx, session.SnapshotFileKey); delErr != nil {
			log.Error("[Doctor] 删除 SMH 备份失败",
				"error", delErr)
		} else {
			model.DB(ctx).Model(session).Update("snapshot_deleted", true)
		}
	}

	// 销毁龙虾医生节点
	if session.DoctorInstanceID != nil {
		var doctorInst model.Instance
		if model.DB(ctx).First(
			&doctorInst,
			*session.DoctorInstanceID).Error == nil {
			// 更新实例状态为 delete/processing
			now := time.Now()
			model.DB(ctx).Model(&model.Instance{}).Where("id = ?", doctorInst.ID).Updates(map[string]interface{}{
				"current_operation":            model.OpDelete,
				"current_operation_state":      model.OpStateProcessing,
				"current_operation_updated_at": &now,
				"last_cvm_state":               "TERMINATING",
			})

			if doctorInst.InstanceId != "" {
				log.Info("[Doctor] 销毁龙虾医生 CVM",
					"cvm_instance_id", doctorInst.InstanceId)
				terminateDoctorCVM(ctx, doctorInst.InstanceId)
			}
		}
		log.Info("[Doctor] 删除龙虾医生实例记录",
			"doctor_instance_db_id",
			*session.DoctorInstanceID)
		model.DB(ctx).Delete(
			&model.Instance{},
			*session.DoctorInstanceID)
	}

	model.DB(ctx).Model(session).
		Update("status", model.DoctorStatusEnded)
	log.Info("[Doctor] 诊断会话清理完成")
}

// ==================== Session 文件管理 ====================

const doctorSessionsDir = "/root/.openclaw/agents/main/sessions"

// DoctorSessionsSMHKey 返回用于存储某用户+目标实例的 session 文件的 SMH key。
func DoctorSessionsSMHKey(
	userID, targetInstanceID uint,
) string {
	return fmt.Sprintf(
		"doctor-sessions/%d/%d/sessions.tar.gz",
		userID, targetInstanceID)
}

// UploadDoctorSessions 通过 TAT 在龙虾医生实例上打包 sessions 目录并上传到 SMH。
// 上传前先删除已有的存档，只保留最新一份。
// 使用 DoctorSessionsSMHKey 作为 SMH 文件路径，确保与 restoreDoctorSessions 下载路径一致。
func UploadDoctorSessions(
	ctx context.Context,
	cvmInstanceId string,
	userID, targetInstanceID uint,
) {
	log := Logger(ctx).With("instance_id", cvmInstanceId)
	fileKey := DoctorSessionsSMHKey(
		userID, targetInstanceID)

	// 先删除旧存档（忽略错误，可能不存在）
	_ = doctorDeleteSMHFileFn(ctx, fileKey)

	// 在龙虾医生实例上打包 sessions 目录
	packScript := fmt.Sprintf(
		`#!/bin/bash
set -e
SESSIONS_DIR="%s"
ARCHIVE="/tmp/doctor-sessions.tar.gz"
if [ ! -d "$SESSIONS_DIR" ] || \
   [ -z "$(ls -A "$SESSIONS_DIR"/*.jsonl 2>/dev/null)" ]; then
  echo "NO_SESSIONS"
  exit 0
fi
tar -czf "$ARCHIVE" -C "$(dirname "$SESSIONS_DIR")" \
  "$(basename "$SESSIONS_DIR")"
stat -c '%%s' "$ARCHIVE"
`, doctorSessionsDir)

	tmpName := fmt.Sprintf(
		"_doctor_pack_sessions_%d.sh",
		time.Now().UnixNano())
	RegisterInlineScript(tmpName, packScript)
	output, err := doctorRunScriptFn(
		ctx, cvmInstanceId, tmpName, 120, "root", nil, nil)
	UnregisterInlineScript(tmpName)

	if err != nil {
		log.Error("[Doctor] 打包 sessions 失败",
			"error", err)
		return
	}

	output = strings.TrimSpace(output)
	if output == "NO_SESSIONS" {
		log.Info("[Doctor] 无 session 文件，跳过上传")
		return
	}

	archiveSize, parseErr := strconv.ParseInt(output, 10, 64)
	if parseErr != nil || archiveSize <= 0 {
		log.Error("[Doctor] 解析存档大小失败",
			"output", output)
		return
	}

	// 使用 PrepareMigrationUpload 指定自定义 fileKey，
	// 确保上传路径与 restoreDoctorSessions 的下载路径一致
	_, uploadErr := doctorUploadArchiveKeyFn(
		ctx,
		cvmInstanceId, "root",
		"/tmp/doctor-sessions.tar.gz",
		archiveSize, fileKey)
	if uploadErr != nil {
		log.Error("[Doctor] 上传 sessions 到 SMH 失败",
			"error", uploadErr)
		return
	}

	log.Info("[Doctor] session 文件已上传到 SMH",
		"file_key", fileKey)
}

// restoreDoctorSessions 和 restartDoctorGateway 已于 2026-05-17 下线：
// 创建龙虾医生时不再主动恢复历史 session，也不再重启 Gateway。
// 仅保留 UploadDoctorSessions，结束时继续将 session 备份到 SMH 供以后使用。

// DoctorMtimeResult 表示获取 session mtime 的结果。
type DoctorMtimeResult struct {
	Mtime   time.Time // 非零值表示成功获取到 mtime
	NoFiles bool      // true 表示远端无 .jsonl 文件（用户尚未开始对话）
	Err     error     // 非 nil 表示脚本执行或解析失败
}

// GetDoctorSessionMtime 通过 TAT 获取龙虾医生实例上
// sessions 目录下最新 .jsonl 文件的修改时间。
// 返回结果区分三种情况：有文件（Mtime 非零）、无文件（NoFiles=true）、获取失败（Err!=nil）。
func GetDoctorSessionMtime(
	ctx context.Context,
	cvmInstanceId string,
) DoctorMtimeResult {
	mtimeScript := fmt.Sprintf(
		`#!/bin/bash
LATEST=$(ls -t "%s"/*.jsonl 2>/dev/null | head -1)
if [ -z "$LATEST" ]; then
  echo "NO_FILES"
  exit 0
fi
stat -c '%%Y' "$LATEST"
`, doctorSessionsDir)

	tmpName := fmt.Sprintf(
		"_doctor_mtime_%d.sh",
		time.Now().UnixNano())
	RegisterInlineScript(tmpName, mtimeScript)
	output, err := doctorRunScriptFn(
		ctx, cvmInstanceId, tmpName, 30, "root", nil, nil)
	UnregisterInlineScript(tmpName)

	if err != nil {
		return DoctorMtimeResult{Err: err}
	}

	output = strings.TrimSpace(output)
	if output == "NO_FILES" {
		return DoctorMtimeResult{NoFiles: true}
	}

	epoch, parseErr := strconv.ParseInt(output, 10, 64)
	if parseErr != nil || epoch <= 0 {
		return DoctorMtimeResult{Err: hcommon.I18nError(i18n.MsgDoctorParseMtimeFailed, output)}
	}

	return DoctorMtimeResult{Mtime: time.Unix(epoch, 0)}
}

// ==================== STS 临时密钥刷新 ====================

// RefreshDoctorSTS 检查所有 active 状态的龙虾医生会话，
// 如果 STS 临时密钥剩余有效期 ≤ 5 分钟，重新申请并通过 TAT 更新到龙虾医生实例。
func RefreshDoctorSTS(ctx context.Context) {
	const refreshThreshold = 6 * 60 // 6 分钟

	var sessions []model.DoctorSession
	model.DB(ctx).Where("status = ? AND sts_expired_at > 0",
		model.DoctorStatusActive).
		Find(&sessions)

	slog.Debug("[DoctorSTS] 开始检查临时密钥",
		"active_sessions", len(sessions))

	now := time.Now().Unix()
	for _, s := range sessions {
		if s.STSExpiredAt-now > refreshThreshold {
			continue // 还没到刷新时间
		}

		log := slog.With("session_id", s.ID,
			"target_instance_id", s.TargetInstanceID)

		// 查目标实例 CVM InstanceId
		var targetInst model.Instance
		if model.DB(ctx).First(&targetInst,
			s.TargetInstanceID).Error != nil {
			log.Warn("[DoctorSTS] 目标实例不存在，跳过")
			continue
		}

		// 查龙虾医生实例 CVM InstanceId
		if s.DoctorInstanceID == nil {
			continue
		}
		var doctorInst model.Instance
		if model.DB(ctx).First(
			&doctorInst,
			*s.DoctorInstanceID).Error != nil {
			log.Warn("[DoctorSTS] 龙虾医生实例不存在，跳过")
			continue
		}
		if doctorInst.InstanceId == "" {
			continue
		}

		// 重新申请 STS
		log.Debug("[DoctorSTS] 密钥即将过期，开始刷新",
			"remaining_seconds", s.STSExpiredAt-now)
		newCred, err := doctorRequestSTSFn(
			ctx, targetInst.InstanceId)
		if err != nil {
			log.Error("[DoctorSTS] 申请 STS 失败",
				"error", err)
			continue
		}

		// 通过 TAT 更新龙虾医生实例的环境变量
		updateScript := fmt.Sprintf(`#!/bin/bash
set -e
sed -i \
  -e 's|^TEMP_SECRET_ID=.*|TEMP_SECRET_ID="%s"|' \
  -e 's|^TEMP_SECRET_KEY=.*|TEMP_SECRET_KEY="%s"|' \
  -e 's|^TEMP_TOKEN=.*|TEMP_TOKEN="%s"|' \
  /etc/environment
echo "STS_REFRESHED"
`, newCred.SecretId, newCred.SecretKey, newCred.Token)

		tmpName := fmt.Sprintf(
			"_doctor_refresh_sts_%d.sh",
			time.Now().UnixNano())
		RegisterInlineScript(tmpName, updateScript)
		_, runErr := doctorRunScriptFn(
			ctx, doctorInst.InstanceId, tmpName, 30,
			"root", nil, nil)
		UnregisterInlineScript(tmpName)

		if runErr != nil {
			log.Error("[DoctorSTS] TAT 更新环境变量失败",
				"error", runErr)
			continue
		}

		// STS 刷新是技术字段维护，不应改变代表业务变更的 updated_at。
		newExpiry := time.Now().Unix() + 7200
		if dbErr := model.DB(ctx).Model(&model.DoctorSession{}).
			Where("id = ?", s.ID).
			UpdateColumn("sts_expired_at", newExpiry).Error; dbErr != nil {
			log.Error("[DoctorSTS] 更新临时密钥过期时间失败",
				"error", dbErr)
			continue
		}

		log.Info("[DoctorSTS] 临时密钥已刷新",
			"new_expired_at", newExpiry)
	}
}
