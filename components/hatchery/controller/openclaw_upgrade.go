package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// pendingUpload 记录一次未完成的分块上传任务信息，用于重试时断点续传。
type pendingUpload struct {
	ArchivePath string // CVM 上的压缩包路径
	ArchiveSize int64  // 压缩包大小（字节）
	FileKey     string // SMH 文件 key，用于重新初始化上传任务复用同一个 ConfirmKey
}

// pendingUploadStore 是 pendingUpload 信息的持久化访问层。
//
// 历史版本曾使用 sync.Map 做进程内缓存，在多副本（多 Pod / 多机器 + LB）部署下会因
// "写入副本 A、重试请求落到副本 B" 而失效，导致重试时退化为完整备份+上传，浪费大量耗时与
// CVM 磁盘空间。这里改为以 instances 表上的 pending_archive_path / pending_archive_size /
// pending_smh_file_key / pending_upload_at 四个字段持久化记录续传信息，多副本下也能正确续传。
//
// 该 store 仍然以 instanceId 作为外部键（与历史调用兼容），内部按 instanceId 反查实例 ID。
// 失败实例不存在 / DB 异常时只打日志、不阻塞主流程，避免续传记录的副作用影响主升级流程。
//
// 暴露的方法刻意保持 sync.Map 三件套的命名（Load/Store/Delete），让调用点和单测无需大改。
type pendingUploadStore struct{}

// pendingUploadCache 是全局单例，所有续传记录读写都通过它。
// 注意：尽管名字保留了 "cache" 字样以兼容既有调用方，实际持久化到 DB，跨副本/跨进程可见。
var pendingUploadCache = &pendingUploadStore{}

// pendingUploadCtx 是 store 在没有外部 ctx 时使用的兜底 context，
// 仅用于测试场景（生产代码均通过 *ContextAware 版本传入 ctx）。
var pendingUploadCtx = context.Background()

// Load 按 instanceId 读取续传记录。返回值兼容 sync.Map.Load 的 (value, ok)。
// value 为 *pendingUpload，ok 表示是否存在有效记录（pending_archive_path 与 pending_smh_file_key 均非空）。
func (s *pendingUploadStore) Load(instanceId string) (*pendingUpload, bool) {
	return s.LoadCtx(pendingUploadCtx, instanceId)
}

// Store 按 instanceId 写入续传记录。
func (s *pendingUploadStore) Store(instanceId string, p *pendingUpload) {
	s.StoreCtx(pendingUploadCtx, instanceId, p)
}

// Delete 按 instanceId 清除续传记录。
func (s *pendingUploadStore) Delete(instanceId string) {
	s.DeleteCtx(pendingUploadCtx, instanceId)
}

// LoadCtx 是 Load 的带 ctx 版本，生产路径优先使用，便于复用调用方的 trace/cancel 语义。
func (s *pendingUploadStore) LoadCtx(ctx context.Context, instanceId string) (*pendingUpload, bool) {
	if instanceId == "" {
		return nil, false
	}
	var inst model.Instance
	if err := model.DB(ctx).
		Select("id, instance_id, pending_archive_path, pending_archive_size, pending_smh_file_key").
		Where("instance_id = ?", instanceId).
		First(&inst).Error; err != nil {
		return nil, false
	}
	if inst.PendingArchivePath == "" || inst.PendingSMHFileKey == "" {
		return nil, false
	}
	return &pendingUpload{
		ArchivePath: inst.PendingArchivePath,
		ArchiveSize: inst.PendingArchiveSize,
		FileKey:     inst.PendingSMHFileKey,
	}, true
}

// StoreCtx 是 Store 的带 ctx 版本，生产路径优先使用。
// DB 写入失败仅打日志，不阻塞主流程：续传记录是优化项，丢失后最坏退化为完整备份+上传。
func (s *pendingUploadStore) StoreCtx(ctx context.Context, instanceId string, p *pendingUpload) {
	if instanceId == "" || p == nil {
		return
	}
	now := time.Now()
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id = ?", instanceId).
		Updates(map[string]interface{}{
			"pending_archive_path": p.ArchivePath,
			"pending_archive_size": p.ArchiveSize,
			"pending_smh_file_key": p.FileKey,
			"pending_upload_at":    &now,
		}).Error; err != nil {
		Logger(ctx).Warn("[pendingUploadStore] 写入续传记录失败（忽略，不影响主流程）",
			"instanceId", instanceId, "error", err)
	}
}

// DeleteCtx 是 Delete 的带 ctx 版本，生产路径优先使用。
// DB 写入失败仅打日志，不阻塞主流程：未及时清理的记录在下次 Load 时会被使用，
// 若届时 CVM 上备份包已被清理，performUpgradeResume 会自动降级为完整流程。
func (s *pendingUploadStore) DeleteCtx(ctx context.Context, instanceId string) {
	if instanceId == "" {
		return
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id = ?", instanceId).
		Updates(map[string]interface{}{
			"pending_archive_path": "",
			"pending_archive_size": 0,
			"pending_smh_file_key": "",
			"pending_upload_at":    nil,
		}).Error; err != nil {
		Logger(ctx).Warn("[pendingUploadStore] 清除续传记录失败（忽略，不影响主流程）",
			"instanceId", instanceId, "error", err)
	}
}

// buildSMHDownloadURLFn 将 SMH 下载 URL 构建抽象为可替换的函数钩子，
// 生产代码指向真实实现，测试时可注入 mock 以隔离 SMH 依赖。
var buildSMHDownloadURLFn = BuildCommonSMHDownloadURL

// waitForInstanceRunningFn 将等待实例 RUNNING 状态的逻辑抽象为可替换的函数钩子，
// 生产代码指向真实实现，测试时可注入 mock 以隔离 CVM 依赖。
var waitForInstanceRunningFn = waitForInstanceRunning

// resetInstanceFn 将 CVM ResetInstance 调用抽象为可替换的函数钩子，
// 生产代码指向真实实现，测试时可注入 mock 以隔离 CVM API 依赖。
var resetInstanceFn = func(client *cvm.Client, req *cvm.ResetInstanceRequest) (*cvm.ResetInstanceResponse, error) {
	return client.ResetInstance(req)
}

// waitForTATAgentOnlineFn 将等待 TAT Agent 上线的逻辑抽象为可替换的函数钩子，
// 生产代码指向真实实现，测试时可注入 mock 以隔离 TAT 依赖。
var waitForTATAgentOnlineFn = waitForTATAgentOnline

// reinstallSleepFn 将重装后的等待时间抽象为可替换的函数钩子，
// 生产代码等待 90 秒，测试时可注入 mock 以跳过等待。
var reinstallSleepFn = func() { time.Sleep(90 * time.Second) }

// restoreSleepFn 将数据恢复重试等待时间抽象为可替换的函数钩子，
// 生产代码等待 30 秒，测试时可注入 mock 以跳过等待。
var restoreSleepFn = func() { time.Sleep(30 * time.Second) }

// fetchVersionInfoFn 将版本信息拉取抽象为可替换的函数钩子，
// 生产代码调用真实实现，测试时可注入 mock 以隔离 TAT 依赖。
var fetchVersionInfoFn = func(ctx context.Context, inst model.Instance) error {
	// 显式判断 *hcommon.RichError，避免将 nil 的具体指针类型包装进 error 接口，
	// 导致调用方 err != nil 误判为失败。
	if rerr := FetchAndSaveVersionInfoSync(ctx, inst); rerr != nil {
		return rerr
	}
	return nil
}

// archiveExistsOnCVMFn 通过 TAT 检查 CVM 上指定路径的备份压缩包是否存在，
// 抽象为可替换函数钩子，便于单测注入 mock 隔离 TAT 依赖。
// 返回 (exists, err)：
//   - exists=true 表示文件存在；
//   - exists=false 且 err=nil 表示文件确认不存在；
//   - err!=nil 表示无法判定（如 TAT 调用失败），调用方按"无法判定"处理。
var archiveExistsOnCVMFn = func(ctx context.Context, instanceId string, archivePath string) (bool, error) {
	if instanceId == "" || archivePath == "" {
		return false, nil
	}
	// 用 echo 标记输出，避免 test -f 退出码非零导致 TAT 整体识别为命令失败：
	// 文件存在输出 EXIST，文件不存在输出 NOT_EXIST，脚本始终 exit 0。
	script := fmt.Sprintf(`if [ -f %q ]; then echo EXIST; else echo NOT_EXIST; fi`, archivePath)
	output, err := runInlineScriptFn(ctx, instanceId, script, 15)
	if err != nil {
		return false, err
	}
	return strings.Contains(output, "EXIST") && !strings.Contains(output, "NOT_EXIST"), nil
}

// smhUploadHooks 将 SMH 分块上传相关操作抽象为可替换的函数钩子，
// 生产代码指向真实实现，测试时可注入 mock 以隔离网络依赖。
var smhUploadHooks = struct {
	Prepare  func(ctx context.Context, instanceId string, archivePath string, archiveSize int64) (*SMHUploadCredential, error)
	GetParts func(ctx context.Context, confirmKey string) (map[int]bool, error)
	Renew    func(ctx context.Context, cred *SMHUploadCredential) error
	Confirm  func(ctx context.Context, confirmKey string) error
}{
	Prepare:  PrepareSMHCommonUpload,
	GetParts: GetSMHCommonUploadParts,
	Renew:    RenewSMHCommonUpload,
	Confirm:  ConfirmSMHCommonUpload,
}

// smhProbeInternalReachableFn 是 hatchery 侧 SMH 内网可达性探测函数指针，便于单测注入。
var smhProbeInternalReachableFn = ProbeSMHInternalReachable

// prepareSMHUploadWithFallback 封装“优先内网，不可达降级外网”的上传凭证获取流程：
//  1. 首先以 internal_domain=1 调用 SMH，拿到内网 PartURLTemplate；
//  2. 在 hatchery 侧对内网 Host 做一次 TCP 探测；
//  3. 探测失败则同参重新调用 SMH、取外网凭证，保证升级流程不被打断。
//
// 返回的 cred.UsedInternalDomain 忠实反映最终采用的是哪条链路，后续 Renew 会据此
// 保持一致。秒传命中时不需探测，直接返回。
func prepareSMHUploadWithFallback(ctx context.Context, instanceId string, archivePath string, archiveSize int64) (*SMHUploadCredential, error) {
	log := Logger(ctx)

	// 1. 首试内网
	internalCtx := WithSMHInternalDomain(ctx, true)
	cred, err := smhUploadHooks.Prepare(internalCtx, instanceId, archivePath, archiveSize)
	if err != nil {
		return nil, err
	}

	// 秒传命中：PartURLTemplate 为空，无需探测
	if cred.PartURLTemplate == "" {
		return cred, nil
	}

	// 2. 探测内网 Host 可达性
	if smhProbeInternalReachableFn(ctx, cred.PartURLTemplate) {
		log.Info("[SMH 上传] 内网域名可达，使用内网凭证上传",
			"instanceId", instanceId, "partURLTemplate", cred.PartURLTemplate)
		return cred, nil
	}

	// 3. 内网不可达→重新调用拿外网凭证
	log.Warn("[SMH 上传] 内网域名探测不可达，降级使用外网域名",
		"instanceId", instanceId, "partURLTemplate", cred.PartURLTemplate)
	externalCtx := WithSMHInternalDomain(ctx, false)
	credExt, errExt := smhUploadHooks.Prepare(externalCtx, instanceId, archivePath, archiveSize)
	if errExt != nil {
		log.Error("[SMH 上传] 重新获取外网凭证失败，回退使用内网凭证继续尝试",
			"instanceId", instanceId, "error", errExt)
		// 避免外网重试失败后升级整体中止：返回内网凭证允许 CVM 走原有路径继续尝试
		return cred, nil
	}
	log.Info("[SMH 上传] 已获取外网凭证用于上传",
		"instanceId", instanceId, "partURLTemplate", credExt.PartURLTemplate)
	return credExt, nil
}

// HandleUpgrade 主动触发实例镜像升级。
// POST /openclaw/upgrade
func HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	handleUpgrade(w, r, defaultStatusResolver)
}

func handleUpgrade(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	ctx := r.Context()
	log := Logger(ctx)
	log.Info("[Upgrade] 收到升级请求", "method", r.Method, "url", r.URL.String())
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		log.Warn("[Upgrade] 用户未登录，拒绝请求")
		return
	}

	if r.Method != http.MethodPost {
		log.Warn("[Upgrade] 非 POST 请求", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Error("[Upgrade] 获取实例失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.IsDoctorNode {
		log.Warn("[Upgrade] 拒绝龙虾医生节点", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}
	log.Info("[Upgrade] 目标实例", "instanceId", instance.InstanceId, "name", instance.Name)

	// 本地实例：不支持镜像升级（升级需 CVM 实例层面操作）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许升级
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		log.Warn("[Upgrade] 当前状态不允许升级",
			"instanceId", instance.InstanceId, "error", err)
		writeAgentGuardError(w, r, err)
		return
	}

	// 获取后台配置的默认（目标）镜像（根据实例的 agent_type）
	defaultImage, err := model.GetEnabledImageByType(ctx, instance.AgentType)
	if err != nil {
		log.Error("[Upgrade] 查询启用镜像失败",
			"instance_id", instance.ID, "agent_type", instance.AgentType, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
		return
	}
	if defaultImage == nil {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		log.Error("[Upgrade] 未找到该类型已启用的镜像",
			"instance_id", instance.ID, "agent_type", instance.AgentType)
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgUpgradeNoImageForType, typeName))
		return
	}
	log.Info("[Upgrade] 目标镜像", "imageId", defaultImage.ImageId, "imageName", defaultImage.ImageName)

	// 实例级前置入口（与批量升级共用同一份逻辑，新增检查请改 prepareInstanceForUpgrade）：
	//   - 拒绝官方镜像降级（OpenClaw 实例当前版本 > 官方镜像版本时）
	//   - openclaw.json 配置 providers key 合法性
	//   - 防重入（current_operation processing）
	//   - 该类型是否支持一键升级
	//   - 官方镜像 runtime_user / runtime_home 强制校正（写 DB，幂等）
	if outcome := prepareInstanceForUpgrade(ctx, instance, defaultImage, "[Upgrade]"); !outcome.OK {
		log.Warn("[Upgrade] 实例级前置检查未通过，拒绝升级",
			"instanceId", instance.InstanceId,
			"agent_type", instance.AgentType,
			"current_operation", instance.CurrentOperation,
			"current_version", instance.AgentVersion,
			"target_image", defaultImage.ImageId,
			"target_version", defaultImage.AgentVersion,
			"batch_status", outcome.BatchStatus,
			"error", outcome.Err,
		)
		writeError(w, r, outcome.HTTPCode, hcommon.EnsureRichErrorOrPanic(outcome.Err))
		return
	}

	// 升级启动入口（与批量升级共用同一份逻辑，新增启动期检查请改 startUpgradeForInstance）：
	//   - checkNeedsUpgrade 判定是否真的需要升级
	//   - setOperation 设置 OpUpgrade 操作锁
	//   - 启动异步 performUpgrade goroutine
	switch outcome := startUpgradeForInstance(ctx, instance, defaultImage, nil, "[Upgrade]"); {
	case outcome.AlreadyLatest:
		log.Info("[Upgrade] 实例已是最新版本，无需升级", "instanceId", instance.InstanceId)
		jsonOK(w, i18n.T(r.Context(), i18n.MsgUpgradeAlreadyLatest))
		return
	case outcome.Err != nil:
		writeError(w, r, outcome.HTTPCode, hcommon.EnsureRichErrorOrPanic(outcome.Err))
		return
	}

	log.Info("[Upgrade] 升级任务已提交，返回异步响应", "instanceId", instance.InstanceId)
	jsonOK(w, i18n.T(r.Context(), i18n.MsgUpgradeStarted))
}

// extractBackupDir 从备份脚本输出中提取本地压缩包路径。
// 备份脚本输出格式：BACKUP_DIR_PATH:/root/openclaw-state-<timestamp>.tgz
func extractBackupDir(output string) string {
	const prefix = "BACKUP_DIR_PATH:"
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

// extractArchiveSize 从备份脚本输出中提取压缩包字节数。
// 备份脚本输出格式：ARCHIVE_SIZE:<bytes>
func extractArchiveSize(output string) int64 {
	const prefix = "ARCHIVE_SIZE:"
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				return n
			}
		}
	}
	return 0
}

// ensureOfficialImageRuntimeUser 在一键升级 / 升级重试的前置阶段，确保"目标镜像为官方镜像"
// 时实例的 runtime_user / runtime_home 与镜像约定一致，防止重装后恢复脚本以不存在的用户
// （如 ubuntu）身份执行时直接失败。
//
// 适用范围（必须同时满足）：
//  1. AgentType == OpenClaw（含空字符串存量兼容）：本次升级能力仅对 OpenClaw 生效，
//     Hermes / LightclawACE 的官方镜像以业务账户（agentuser 或 ubuntu 等，随镜像版本变动）运行，
//     不能被改写为 root。
//  2. 目标镜像（即本次升级要刷入的 defaultImage）命中候选公共镜像列表
//     （hcommon.IsCandidateImage 为 true）。
//
// 约定：OpenClaw 官方镜像出厂用户固定为 root，HOME 目录固定为 /root。
//
// 参数：
//   - instance:       待升级实例（Go 内存对象，本函数会同步更新其 RuntimeUser/RuntimeHome）
//   - targetImageId:  本次升级的目标镜像 ID（即 defaultImage.ImageId），
//     用于判断重装后新系统是否为官方镜像。
//
// 行为：
//   - 非 OpenClaw 类型 / targetImageId 为空 / 目标非官方镜像：直接返回，不做变更；
//   - 目标为 OpenClaw 官方镜像：若 DB 里的 runtime_user/home 不是 root / /root，
//     则覆盖为 root / /root，同时刷新内存对象，保证后续 RunScript / UserData 渲染使用到的是最新值。
func ensureOfficialImageRuntimeUser(ctx context.Context, instance *model.Instance, targetImageId string) error {
	log := Logger(ctx)
	if instance == nil || targetImageId == "" {
		return nil
	}
	// 能力位门控：仅 NeedsRuntimeUserCorrection=true 的类型需校正（当前仅 OpenClaw 官方镜像出厂 root）
	meta := model.GetAgentTypeByCode(ctx, instance.AgentType)
	if meta == nil || !meta.NeedsRuntimeUserCorrection {
		log.Info("[ensureOfficialImageRuntimeUser] 当前类型不需要 runtime_user 校正，跳过",
			"instanceId", instance.InstanceId, "agentType", instance.AgentType, "targetImageId", targetImageId)
		return nil
	}
	if !hcommon.IsCandidateImage(targetImageId) {
		log.Info("[ensureOfficialImageRuntimeUser] 目标镜像非官方镜像，跳过 runtime_user 校正",
			"instanceId", instance.InstanceId, "targetImageId", targetImageId)
		return nil
	}

	const (
		expectedUser = "root"
		expectedHome = "/root"
	)
	if instance.RuntimeUser == expectedUser && instance.RuntimeHome == expectedHome {
		log.Info("[ensureOfficialImageRuntimeUser] 官方镜像 runtime_user/home 已符合预期",
			"instanceId", instance.InstanceId, "targetImageId", targetImageId)
		return nil
	}

	log.Warn("[ensureOfficialImageRuntimeUser] 目标为官方镜像但 runtime_user/home 与预期不一致，强制校正",
		"instanceId", instance.InstanceId, "targetImageId", targetImageId,
		"dbRuntimeUser", instance.RuntimeUser, "dbRuntimeHome", instance.RuntimeHome,
		"expectedRuntimeUser", expectedUser, "expectedRuntimeHome", expectedHome,
	)

	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]interface{}{
			"runtime_user": expectedUser,
			"runtime_home": expectedHome,
		}).Error; err != nil {
		log.Error("[ensureOfficialImageRuntimeUser] 更新 runtime_user/home 失败",
			"instanceId", instance.InstanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeFixOfficialImageRuntimeUserFailed)
	}

	// 同步内存对象，保证后续 RunScript / UserData 渲染读取到正确值
	instance.RuntimeUser = expectedUser
	instance.RuntimeHome = expectedHome
	return nil
}

// ensureOfficialImageRuntimeUserForUpgrade 是 ensureOfficialImageRuntimeUser 的入口侧薄包装，
// 供 HandleUpgrade / HandleUpgradeRetry 复用：失败时按入口 logPrefix 记录统一错误日志，
// 成功时透明返回。抽出此函数的目的是让"入口调用位点"的错误分支可通过单元测试直接覆盖，
// 而不必模拟整套 HTTP + CVM + TAT 依赖。
//
// 参数：
//   - logPrefix:     日志前缀（例如 "[Upgrade]" / "[UpgradeRetry]"），仅用于区分日志归属。
//   - targetImageId: 本次升级的目标镜像 ID。
//
// 返回：ensureOfficialImageRuntimeUser 的原始 error，调用方需据此写 HTTP 响应。
func ensureOfficialImageRuntimeUserForUpgrade(ctx context.Context, instance *model.Instance, targetImageId, logPrefix string) error {
	if err := ensureOfficialImageRuntimeUser(ctx, instance, targetImageId); err != nil {
		instanceID := ""
		if instance != nil {
			instanceID = instance.InstanceId
		}
		Logger(ctx).Error(fmt.Sprintf("%s 校正官方镜像 runtime_user 失败", logPrefix),
			"instanceId", instanceID, "error", err)
		return err
	}
	return nil
}

// rejectDowngradeOnOfficialImage 在 HandleUpgrade / HandleAdminBatchUpgrade /
// HandleUpgradeRetry 入口前置阶段执行，用于拒绝"把 OpenClaw 实例降级覆盖"的非法升级请求。
//
// 历史背景：函数原先仅拦截"官方公共镜像降级"，自定义镜像（管理员可在后台编辑 agent_version）
// 完全放行。结果出现了"管理员把启用中的自定义镜像版本号改小 / 切换到低版本自定义镜像 →
// 用户触发升级 → 实例从高版本被刷成低版本"的事故。本函数已扩展为：
// 无论目标是官方还是自定义镜像，只要满足版本下行条件就拦截，文案区分两类来源。
// rejectDowngradeOnOfficialImage 检查实例当前版本是否高于目标镜像版本，防止降级。
//
// 拦截条件（必须同时满足，任一不满足即放行）：
//  1. agent_type 启用了一键升级且版本格式兼容 CompareSemver；
//     CompareSemver 按 %d.%d.%d 解析三段数字，兼容 OpenClaw（2026.3.28）和 Hermes（0.17.0）两种格式；
//  2. 镜像声明了 agent_version（非空）；
//  3. 实例当前 agent_version 严格高于镜像版本（CompareSemver > 0）。
//
// 实例 agent_version 为空时放行，交给后续 checkNeedsUpgrade 自行决定。
// 文案区分官方/自定义镜像来源，便于定位问题。
func rejectDowngradeOnOfficialImage(ctx context.Context, instance *model.Instance, defaultImage *model.AIImage) error {
	if instance == nil || defaultImage == nil {
		return nil
	}
	// 条件 1：仅支持一键升级且版本格式兼容 CompareSemver 的类型参与拦截
	// （当前为 OpenClaw 和 Hermes，版本格式分别为 YYYY.M.D 和 semver，均按 %d.%d.%d 解析）
	if !model.AgentTypeSupportsUpgrade(ctx, instance.AgentType) {
		return nil
	}
	// 条件 2：镜像声明了 agent_version
	targetVersion := strings.TrimSpace(defaultImage.AgentVersion)
	if targetVersion == "" {
		return nil
	}
	currentVersion := strings.TrimSpace(instance.AgentVersion)
	if currentVersion == "" {
		// 当前版本未知，放行交给后续流程决策
		return nil
	}
	// 条件 3：实例当前版本严格高于目标镜像版本
	if model.CompareSemver(currentVersion, targetVersion) <= 0 {
		return nil
	}
	// 文案区分官方/自定义镜像来源
	if hcommon.IsCandidateImage(defaultImage.ImageId) {
		return hcommon.I18nError(i18n.MsgUpgradeRetryVersionDowngrade, currentVersion, targetVersion)
	}
	return hcommon.I18nError(i18n.MsgUpgradeVersionDowngradeCustomImage, currentVersion, targetVersion)
}

// checkNeedsUpgrade 检查单个实例是否需要升级镜像。
// 规则：
//  0. 实例当前状态不是 running → 不需要升级（仅运行中的实例支持升级）
//  1. 实例当前镜像 ID 与默认镜像 ID 不同 → 需要升级（镜像差异优先，不再做版本比对）
//  2. 镜像 ID 相同：
//     2a. 若实例 agent_version 与目标镜像 agent_version 一致 → 不需要升级
//     2b. 否则（版本不一致或无法判定） → 需要升级
//
// cvmInfoMap 为可选参数（最多传一个），如果传入则从中获取 CVM 信息，否则自动调用 batchFetchCVMInfoMap 查询。
// 批量场景下可预先查询好 cvmInfoMap 传入，避免逐个调用 CVM API。
//
// 返回 (instanceImageId, needUpgrade, error)
func checkNeedsUpgrade(ctx context.Context, instance *model.Instance, defaultImage *model.AIImage, cvmInfoMap ...map[string]*CVMInstanceInfo) (string, bool, error) {
	log := Logger(ctx)
	instanceId := instance.InstanceId
	if defaultImage == nil {
		return "", false, hcommon.I18nError(i18n.MsgUpgradeDefaultImageEmpty)
	}
	defaultImageId := defaultImage.ImageId
	log.Info("[checkNeedsUpgrade] 开始检查", "instanceId", instanceId, "defaultImageId", defaultImageId)

	// 获取 CVM 实例信息：优先使用传入的 cvmInfoMap，否则自行查询
	var infoMap map[string]*CVMInstanceInfo
	if len(cvmInfoMap) > 0 && cvmInfoMap[0] != nil {
		infoMap = cvmInfoMap[0]
	} else {
		infoMap = batchFetchCVMInfoMap(ctx, []string{instanceId})
	}

	info, ok := infoMap[instanceId]
	if !ok || info == nil {
		return "", false, hcommon.I18nError(i18n.MsgUpgradeCannotGetCVMInfo, instanceId)
	}

	// 规则 0：检查实例是否正常运行（只有 running 状态的实例才能升级）
	statusResp := ResolveInstanceStatus(ctx, instance, info, nil)
	if statusResp.Status != model.StatusRunning {
		log.Info("[checkNeedsUpgrade] 实例非运行状态，跳过升级",
			"instanceId", instanceId, "status", statusResp.Status, "label", statusResp.Label)
		return "", false, hcommon.I18nError(i18n.MsgUpgradeInstanceNotRunning)
	}

	instanceImageId := info.ImageId
	if instanceImageId == "" {
		return "", false, hcommon.I18nError(i18n.MsgUpgradeCVMImageIDEmpty, instanceId)
	}
	log.Info("[checkNeedsUpgrade] 实例当前镜像", "instanceId", instanceId, "instanceImageId", instanceImageId)

	// 规则 1：镜像 ID 不同，直接需要升级（版本比对是镜像一致时的进一步判断，这里不做）
	if instanceImageId != defaultImageId {
		log.Info("[checkNeedsUpgrade] 规则1命中：镜像 ID 不同，需要升级", "instanceImageId", instanceImageId, "defaultImageId", defaultImageId)
		return instanceImageId, true, nil
	}

	// 规则 2：镜像 ID 相同，使用 agent_version 进一步判断
	// 一致 → 无需升级；不一致或无法判定 → 仍标记为需要升级
	same, verr := isInstanceVersionSameAsImage(ctx, instance, defaultImage)
	if verr != nil {
		log.Warn("[checkNeedsUpgrade] 版本比对异常，按'需要升级'处理",
			"instanceId", instanceId, "error", verr)
		return instanceImageId, true, nil
	}
	if same {
		log.Info("[checkNeedsUpgrade] 规则2命中：镜像 ID 与 agent_version 均一致，无需升级",
			"instanceId", instanceId,
			"imageId", instanceImageId,
			"agentVersion", instance.AgentVersion)
		return instanceImageId, false, nil
	}
	log.Info("[checkNeedsUpgrade] 镜像 ID 一致但 agent_version 不一致，需要升级",
		"instanceId", instanceId,
		"imageId", instanceImageId,
		"currentVersion", instance.AgentVersion,
		"targetVersion", defaultImage.AgentVersion)
	return instanceImageId, true, nil
}

// isInstanceVersionSameAsImage 判断实例当前运行的 agent_version 是否与目标镜像一致。
// 仅适用于 OpenClaw 类型的一键升级/重试路径。
//
// 比对策略：
//  1. 若目标镜像未设置 agent_version（存量镜像） → 返回 (false, nil)，交给调用方走后续流程；
//  2. 若实例 DB 里的 agent_version 为空或明显过期 → 同步触发一次版本拉取刷新到最新；
//  3. 比对实例版本与镜像版本（去两端空白、大小写敏感）。
//
// 返回 (same, err)：
//   - same=true 表示版本一致（无需升级）；
//   - err!=nil 表示比对过程中出现底层错误（例如 TAT 拉取失败），调用方可选择忽略或告警。
func isInstanceVersionSameAsImage(ctx context.Context, instance *model.Instance, image *model.AIImage) (bool, error) {
	log := Logger(ctx)
	if instance == nil || image == nil {
		return false, nil
	}

	targetVersion := strings.TrimSpace(image.AgentVersion)
	if targetVersion == "" {
		// 镜像未声明版本（存量镜像），无从比较
		log.Info("[isInstanceVersionSameAsImage] 目标镜像未设置 agent_version，跳过版本比对",
			"instanceId", instance.InstanceId, "imageId", image.ImageId)
		return false, nil
	}

	// 若 DB 里的实例版本为空，尝试同步拉取一次
	if strings.TrimSpace(instance.AgentVersion) == "" {
		log.Info("[isInstanceVersionSameAsImage] 实例 agent_version 为空，触发同步拉取",
			"instanceId", instance.InstanceId)
		if err := FetchAndSaveVersionInfoSync(ctx, *instance); err != nil {
			log.Warn("[isInstanceVersionSameAsImage] 同步拉取版本失败，按'无法判定'处理",
				"instanceId", instance.InstanceId, "error", err)
			return false, err
		}
		// FetchAndSaveVersionInfoSync 内部直接写 DB，不会反向更新内存对象，这里重新 load 一次
		var refreshed model.Instance
		if err := model.DB(ctx).Select("id, instance_id, agent_version, version_fetched_at").
			First(&refreshed, instance.ID).Error; err != nil {
			log.Warn("[isInstanceVersionSameAsImage] 重新加载实例版本失败",
				"instanceId", instance.InstanceId, "error", err)
			return false, err
		}
		instance.AgentVersion = refreshed.AgentVersion
		instance.VersionFetchedAt = refreshed.VersionFetchedAt
	}

	currentVersion := strings.TrimSpace(instance.AgentVersion)
	if currentVersion == "" {
		log.Info("[isInstanceVersionSameAsImage] 实例 agent_version 仍为空，无法比对",
			"instanceId", instance.InstanceId)
		return false, nil
	}

	same := currentVersion == targetVersion
	log.Info("[isInstanceVersionSameAsImage] 版本比对结果",
		"instanceId", instance.InstanceId,
		"currentVersion", currentVersion,
		"targetVersion", targetVersion,
		"same", same)
	return same, nil
}

// waitForInstanceRunning 轮询等待 CVM 实例状态变为 RUNNING。
// 每 10 秒查询一次，超过 timeout 则返回错误。
func waitForInstanceRunning(ctx context.Context, client *cvm.Client, instanceId string, timeout time.Duration) error {
	log := Logger(ctx)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Second)

		req := cvm.NewDescribeInstancesStatusRequest()
		req.InstanceIds = common.StringPtrs([]string{instanceId})
		resp, err := client.DescribeInstancesStatus(req)
		if err != nil {
			log.Warn("查询实例状态失败，继续等待", "instanceId", instanceId, "error", err)
			continue
		}
		if resp.Response == nil || len(resp.Response.InstanceStatusSet) == 0 {
			log.Warn("查询实例状态返回空，继续等待", "instanceId", instanceId)
			continue
		}
		status := ""
		if resp.Response.InstanceStatusSet[0].InstanceState != nil {
			status = *resp.Response.InstanceStatusSet[0].InstanceState
		}
		log.Info("实例当前状态", "instanceId", instanceId, "status", status)
		if status == "RUNNING" {
			return nil
		}
	}
	return hcommon.I18nError(i18n.MsgUpgradeInstanceNotRecover, instanceId, timeout)
}

// waitForOpenclawReady 轮询等待 agent 安装完成并服务就绪。
// 注意：函数名保留历史名 waitForOpenclawReady，实际语义为通用 agent ready 检查，
// 按 agentType 分派 check_ready 脚本（openclaw→check_openclaw_ready.sh, hermes→check_hermes_ready.sh 等）。
// 通过执行 check_ready 脚本检查服务状态；agentType 为空时默认使用 openclaw。
// 每 30 秒检查一次，超过 timeout 则返回错误。
func waitForOpenclawReady(ctx context.Context, instanceId string, agentType string, timeout time.Duration) error {
	log := Logger(ctx)
	agentType = model.NormalizeAgentType(agentType)
	scriptName, err := ResolveScript(ctx, "check_ready", agentType)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUpgradeResolveCheckReadyFailed, agentType)
	}
	log.Info("[waitForAgentReady] 开始等待 agent 就绪", "instanceId", instanceId, "agentType", agentType, "timeout", timeout)

	runtimeUser := LookupRuntimeUser(ctx, instanceId)

	deadline := time.Now().Add(timeout)
	attempt := 0
	for time.Now().Before(deadline) {
		attempt++
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return hcommon.I18nRichError(ctx.Err(), i18n.MsgUpgradeWaitAgentReadyCanceled)
			case <-time.After(30 * time.Second):
			}
		}

		output, err := runScriptFn(ctx, instanceId, scriptName, 30, runtimeUser, nil, nil)
		if err != nil {
			log.Warn("[waitForAgentReady] 检查脚本执行失败，继续等待",
				"instanceId", instanceId, "attempt", attempt, "error", err)
			continue
		}

		// 从输出中找到最后一行 JSON（脚本前面有日志输出行）
		output = strings.TrimSpace(output)
		lines := strings.Split(output, "\n")
		jsonLine := ""
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "{") {
				jsonLine = line
				break
			}
		}

		if strings.Contains(jsonLine, `"ready": true`) || strings.Contains(jsonLine, `"ready":true`) {
			log.Info("[waitForAgentReady] agent 已就绪", "instanceId", instanceId, "attempt", attempt)
			return nil
		}

		// 提取 reason 用于日志
		reason := "unknown"
		if idx := strings.Index(jsonLine, `"reason": "`); idx >= 0 {
			rest := jsonLine[idx+len(`"reason": "`):]
			if end := strings.Index(rest, `"`); end >= 0 {
				reason = rest[:end]
			}
		} else if idx := strings.Index(jsonLine, `"reason":"`); idx >= 0 {
			rest := jsonLine[idx+len(`"reason":"`):]
			if end := strings.Index(rest, `"`); end >= 0 {
				reason = rest[:end]
			}
		}
		log.Info("[waitForAgentReady] agent 尚未就绪，继续等待",
			"instanceId", instanceId, "attempt", attempt, "reason", reason,
			"remaining", time.Until(deadline).Round(time.Second))
	}

	return hcommon.I18nError(i18n.MsgUpgradeOpenclawNotReady, instanceId, timeout)
}

// finalizeUpgradeResult 根据升级结果统一处理操作锁和通知。
// 成功：清空 current_operation 并写入成功通知；
// 失败：保留 current_operation=upgrade，仅把 state 置为 failed（便于前端识别"升级失败，可重试"），写入失败通知。
//
// 使用范围：所有走 performUpgrade 的升级路径，包括：
//   - OpenClaw 一键升级（HandleUpgrade）
//   - 升级重试（HandleUpgradeRetry，有备份时直接调用 reinstallAndRestore 后手动调用本函数）
//   - 管理员批量升级（HandleAdminBatchUpgrade → performUpgrade → defer 本函数）
//
// 上述入口均已限制 AgentType == OpenClaw，因此"失败保留 current_operation"的语义只会作用于 OpenClaw 实例，
// 不会影响其他 AgentType 或其他操作（reboot/reinstall/delete 等仍走 clearOperation 的原有行为）。
func finalizeUpgradeResult(ctx context.Context, instance *model.Instance, upgradeErr error) {
	log := Logger(ctx)
	if upgradeErr != nil {
		// 中止分支：未真正开始升级（如磁盘不足），清操作锁（不写 failed），发通知。
		if aborted, ok := isUpgradeAbortedErr(upgradeErr); ok {
			log.Warn("[performUpgrade] 升级中止（未真正开始，原实例仍可用），清除操作锁",
				"instanceId", instance.InstanceId,
				"reason", aborted.Reason,
				"user_msg", aborted.UserMsg)
			if err := clearOperation(model.DB(ctx), instance, model.OpStateSuccess); err != nil {
				log.Error("[performUpgrade] 中止分支清除操作锁失败",
					"instanceId", instance.InstanceId, "error", err)
			}
			// 发送通知：复用 UpgradeFailed 类型（前端红点），但标题/正文使用「未开始」文案
			go createErrorNotification(
				instance.UserID, instance.ID, instance.Name,
				model.NotifyTypeInstanceUpgradeFailed,
				i18n.T(hcommon.DetachContext(ctx), i18n.MsgUpgradeAbortedTitle),
				errors.New(aborted.UserMsg), hcommon.DetachContext(ctx),
			)
			return
		}
		log.Error("[performUpgrade] 升级失败，保留 current_operation=upgrade 并置 state=failed",
			"instanceId", instance.InstanceId, "error", upgradeErr)
		if err := markOperationFailed(model.DB(ctx), instance, upgradeErr.Error()); err != nil {
			log.Error("[performUpgrade] markOperationFailed 失败", "instanceId", instance.InstanceId, "error", err)
		}
		// 写入升级失败通知
		go createErrorNotification(
			instance.UserID, instance.ID, instance.Name,
			model.NotifyTypeInstanceUpgradeFailed,
			i18n.T(hcommon.DetachContext(ctx), i18n.MsgUpgradeFailedTitle),
			upgradeErr, hcommon.DetachContext(ctx),
		)
		return
	}
	log.Info("[performUpgrade] 升级成功，清除操作锁", "instanceId", instance.InstanceId)
	if err := clearOperation(model.DB(ctx), instance, model.OpStateSuccess); err != nil {
		log.Error("[performUpgrade] 清除操作锁失败", "instanceId", instance.InstanceId, "error", err)
	}

	// 升级成功后异步恢复 SMH 环境（init_smh_env.sh：安装 skill + 注入 token）。
	// CVM 此时已 RUNNING（reinstallAndRestore 已等待完成），syncSMHEnvWhenReady 内的
	// waitForCVMRunning 会在首轮即通过，不会额外等待。
	go syncSMHEnvWhenReadyFn(hcommon.DetachContext(ctx), *instance)

	// 写入升级成功通知
	go model.CreateSuccessNotification(
		hcommon.DetachContext(ctx),
		instance.UserID, instance.ID, instance.Name,
		model.NotifyTypeInstanceUpgradeSuccess,
		i18n.T(hcommon.DetachContext(ctx), i18n.MsgUpgradeSuccessTitle),
		i18n.T(hcommon.DetachContext(ctx), i18n.MsgUpgradeSuccessContent, instance.Name),
	)
}

// performUpgradeResume 断点续传路径：跳过重新备份，直接用缓存中的 archivePath/fileKey 续传上传。
// 适用于进程内上传失败后重试，CVM 上的备份包仍存在的场景。
// 若续传失败（如 CVM 上备份包已被清理），降级走完整流程（重新备份+上传）。
func performUpgradeResume(ctx context.Context, instance *model.Instance, defaultImageId string, pending *pendingUpload) (upgradeErr error) {
	log := Logger(ctx)
	log.Info("[performUpgradeResume] 尝试断点续传上传", "instanceId", instance.InstanceId,
		"archivePath", pending.ArchivePath, "fileKey", pending.FileKey)

	defer func() {
		finalizeUpgradeResult(ctx, instance, upgradeErr)
	}()

	// 前置：检查 CVM 上的备份压缩包是否仍存在。若文件已被清理（例如机器重启后被清空、
	// 用户手工删除等），任何后续的分块上传都必将失败。这里直接清除续传记录并降级走全量流程，
	// 避免下发一堆注定失败的分块上传脚本。
	exists, existsErr := archiveExistsOnCVMFn(ctx, instance.InstanceId, pending.ArchivePath)
	if existsErr != nil {
		log.Warn("[performUpgradeResume] 检查 CVM 备份包是否存在失败，按存在处理继续尝试续传",
			"instanceId", instance.InstanceId, "archivePath", pending.ArchivePath, "error", existsErr)
	} else if !exists {
		log.Warn("[performUpgradeResume] CVM 上备份压缩包已不存在，清除续传记录并降级走全量升级流程",
			"instanceId", instance.InstanceId, "archivePath", pending.ArchivePath)
		pendingUploadCache.DeleteCtx(ctx, instance.InstanceId)
		return performUpgrade(ctx, instance, defaultImageId, "")
	}

	// 重新调用 PrepareSMHCommonUpload：SMH 对同路径的未完成任务会返回同一个 ConfirmKey
	// 走“内网优先，不可达降级外网”的统一入口，保证与初次上传一致的选择逻辑
	cred, err := prepareSMHUploadWithFallback(ctx, instance.InstanceId, pending.ArchivePath, pending.ArchiveSize)
	if err != nil {
		log.Warn("[performUpgradeResume] 获取 SMH 上传凭证失败，降级走完整升级流程",
			"instanceId", instance.InstanceId, "error", err)
		return performUpgrade(ctx, instance, defaultImageId, "")
	}

	if cred.PartURLTemplate == "" {
		// 秒传命中（文件已完整上传）
		if cred.FileKey != "" {
			log.Info("[performUpgradeResume] SMH 秒传命中，直接进入重装恢复",
				"instanceId", instance.InstanceId, "fileKey", cred.FileKey)
			pendingUploadCache.DeleteCtx(ctx, instance.InstanceId)
			return reinstallAndRestore(ctx, instance, defaultImageId, cred.FileKey)
		}
		log.Warn("[performUpgradeResume] SMH 上传凭证异常，降级走完整升级流程", "instanceId", instance.InstanceId)
		return performUpgrade(ctx, instance, defaultImageId, "")
	}

	// 将续传记录更新为最新的 fileKey（理论上与 pending.FileKey 一致）
	pendingUploadCache.StoreCtx(ctx, instance.InstanceId, &pendingUpload{
		ArchivePath: pending.ArchivePath,
		ArchiveSize: pending.ArchiveSize,
		FileKey:     cred.FileKey,
	})

	// 查询已完成分块，续传剩余分块
	uploadedParts, err := smhUploadHooks.GetParts(ctx, cred.ConfirmKey)
	if err != nil {
		log.Warn("[performUpgradeResume] 查询已上传分块失败，降级为全量上传",
			"instanceId", instance.InstanceId, "error", err)
		uploadedParts = map[int]bool{}
	}
	log.Info("[performUpgradeResume] 断点续传分块状态",
		"instanceId", instance.InstanceId,
		"uploadedParts", len(uploadedParts),
		"totalParts", cred.TotalParts,
	)

	// 按图片流程：查完已上传分块后，立即主动 renew 一次，刷新 COS 上传凭证。
	// PrepareSMHCommonUpload 返回的是旧凭证（上次初始化时生成），可能已过期，
	// 必须先 renew 拿到全新的 COS 临时签名，再开始上传剩余分块。
	if renewErr := smhUploadHooks.Renew(ctx, cred); renewErr != nil {
		log.Error("[performUpgradeResume] 续期 SMH 上传凭证失败",
			"instanceId", instance.InstanceId, "error", renewErr)
		return hcommon.I18nRichError(renewErr, i18n.MsgUpgradeRenewSMHCredFailed)
	}
	log.Info("[performUpgradeResume] SMH 上传凭证已续期", "instanceId", instance.InstanceId)

	if err = smhUploadLoop(ctx, instance.InstanceId, cred, uploadedParts, pending.ArchivePath, instance.RuntimeUser); err != nil {
		return err
	}
	pendingUploadCache.DeleteCtx(ctx, instance.InstanceId)
	log.Info("[performUpgradeResume] 断点续传完成，进入重装恢复", "instanceId", instance.InstanceId, "fileKey", cred.FileKey)
	return reinstallAndRestore(ctx, instance, defaultImageId, cred.FileKey)
}

// performUpgrade 执行完整的升级流程：备份 → 上传 SMH → 重装 → 恢复 → 清理。
// 返回 error 表示升级失败，nil 表示升级成功。
var performUpgrade = func(ctx context.Context, instance *model.Instance, defaultImageId string, currentImageId string) (upgradeErr error) {
	log := Logger(ctx)
	log.Info("[performUpgrade] 开始升级实例镜像", "instanceId", instance.InstanceId,
		"currentImage", currentImageId, "targetImage", defaultImageId)

	defer func() {
		finalizeUpgradeResult(ctx, instance, upgradeErr)
	}()

	// 同步确保 RuntimeUser 已被探测并写回 instance：
	// 升级链路下游（backup_pre_reinstall_*.sh / restore_post_reinstall_*.sh / 重装 UserData
	// 渲染等）都依赖 instance.RuntimeUser；若 AgentChecker 异步探测尚未跑完，DB 中的字段
	// 仍为空，会导致 getEffectiveRuntimeUser() 全程 fallback 到 "root"：
	//   - backup 阶段：runScriptFn 以 root 身份在 /root/.hermes 打包（应在 $HOME/.hermes，
	//     如 /home/agentuser/.hermes 或 /home/ubuntu/.hermes，取决于镜像版本）
	//   - reinstall UserData：按 root 渲染 init.sh，重装后用户体系与备份不匹配
	//   - restore 阶段：模板变量 {{runtime_user}} 被注入 "root"，恢复脚本把数据解压到
	//     /root/.hermes，agentuser 视角下 .hermes 仍是新镜像出厂目录，等同于恢复失败。
	// 这里同步走一次 ensureRuntimeUser，最坏返回 "root" 与现状一致，不引入额外风险。
	instance.RuntimeUser = ensureRuntimeUser(ctx, instance.ID, instance.InstanceId, instance.AgentType)
	log.Info("[performUpgrade] RuntimeUser 已确认", "instanceId", instance.InstanceId, "runtimeUser", instance.RuntimeUser)

	// 第一、二步：备份 + 上传 SMH
	fileKey, err := backupAndUploadToSMH(ctx, instance)
	if err != nil {
		return err
	}

	// 第三~五步：重装 + 等待就绪 + 数据恢复 + 清理 SMH 备份
	return reinstallAndRestore(ctx, instance, defaultImageId, fileKey)
}

// backupAndUploadToSMH 执行升级流程的第一、二步：
//  1. 在 CVM 上执行备份脚本生成压缩包
//  2. 将压缩包分块上传到 SMH common space
//
// 返回上传成功后的 SMH fileKey。调用方在重装完成后可通过 fileKey 生成下载 URL 做数据恢复。
//
// 备份脚本通过 ResolveScript("backup_pre_reinstall", agentType) 分派：
//   - openclaw → backup_pre_reinstall.sh（备份 ~/.openclaw/）
//   - hermes   → backup_pre_reinstall_hermes.sh（备份 ~/.hermes/）
//
// 两个脚本的 stdout 契约（BACKUP_DIR_PATH:/ARCHIVE_SIZE:）保持一致，extractBackupDir /
// extractArchiveSize 无需 agent-specific 处理。
func backupAndUploadToSMH(ctx context.Context, instance *model.Instance) (string, error) {
	log := Logger(ctx)

	// 打包前二次探测磁盘空间：通道错误放行，不足时中止（不写 failed，发通知）。
	if res, perr := precheckUpgradeDiskSpace(ctx, instance); perr != nil {
		log.Warn("[performUpgrade] 磁盘空间二次探测通道错误，放行让后续兜底",
			"instanceId", instance.InstanceId, "error", perr)
	} else if res != nil && !res.OK() {
		log.Warn("[performUpgrade] 磁盘空间不足，中止升级（原实例保持可用）",
			"instanceId", instance.InstanceId,
			"required_kb", res.RequiredKB,
			"home_avail_kb", res.HomeAvailKB,
			"reason", res.Reason)
		return "", buildAbortedByDiskInsufficient(ctx, res)
	}

	// 第一步：重装前数据备份。按 agent_type 分派备份脚本。
	backupScript, resolveErr := ResolveScript(ctx, "backup_pre_reinstall", instance.AgentType)
	if resolveErr != nil {
		log.Error("[performUpgrade] 解析备份脚本失败", "instanceId", instance.InstanceId, "agentType", instance.AgentType, "error", resolveErr)
		return "", hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportUpgradeWithDetail, instance.AgentType)
	}
	log.Info("[performUpgrade] 第一步：执行重装前数据备份", "instanceId", instance.InstanceId, "script", backupScript)
	backupOutput, err := runScriptFn(ctx, instance.InstanceId, backupScript, 600, instance.RuntimeUser, nil, nil)
	if err != nil {
		// 备份阶段检测到本地数据库损坏且无法无损修复 → 中止升级（而非普通失败可重试）：
		// 此时机器尚未重装、原始数据仍完整保留在原盘，绝不能继续做不可逆的重装。
		// 走 errUpgradeAborted 分支（clearOperation success + 中止通知，原实例保持可用），
		// 因为库不修好，重试也没用，需人工离线抢救。
		if backupDBUnrecoverable(backupOutput, err) {
			log.Warn("[performUpgrade] 备份阶段检测到本地数据库损坏且无法无损修复，中止升级（原实例保持可用）",
				"instanceId", instance.InstanceId)
			return "", buildAbortedByDBUnrecoverable(ctx)
		}
		log.Error("[performUpgrade] 数据备份失败", "instanceId", instance.InstanceId, "error", err)
		return "", hcommon.I18nRichError(err, i18n.MsgUpgradeBackupFailed)
	}
	log.Info("[performUpgrade] 数据备份完成", "instanceId", instance.InstanceId)

	// 第二步：备份完成后立即上传到 SMH（重装会清空磁盘，必须在重装前上传）
	log.Info("[performUpgrade] 第二步：上传备份包到 SMH", "instanceId", instance.InstanceId)
	archivePath := extractBackupDir(backupOutput)
	if archivePath == "" {
		log.Error("[performUpgrade] 无法从备份输出中提取压缩包路径", "instanceId", instance.InstanceId, "backupOutput", backupOutput)
		return "", hcommon.I18nError(i18n.MsgUpgradeBackupArchivePathMissing)
	}
	archiveSize := extractArchiveSize(backupOutput)
	log.Info("[performUpgrade] 备份包信息", "instanceId", instance.InstanceId, "archivePath", archivePath, "archiveSize", archiveSize)

	// “内网优先，不可达降级外网”获取 SMH 上传凭证，提升 CVM ↔ COS 的传输效率
	cred, rerr := prepareSMHUploadWithFallback(ctx, instance.InstanceId, archivePath, archiveSize)
	if rerr != nil {
		log.Error("[performUpgrade] 获取 SMH 上传凭证失败", "instanceId", instance.InstanceId, "error", rerr)
		return "", hcommon.I18nRichError(rerr, i18n.MsgSMHUploadCredFailed)
	}

	if cred.PartURLTemplate == "" {
		// 秒传成功场景：SMH 返回 200 表示相同内容已存在，直接用 cred.FileKey
		if cred.FileKey != "" {
			log.Info("[performUpgrade] SMH 秒传命中，跳过分块上传", "instanceId", instance.InstanceId, "fileKey", cred.FileKey)
			pendingUploadCache.DeleteCtx(ctx, instance.InstanceId)
			return cred.FileKey, nil
		}
		log.Error("[performUpgrade] SMH 上传凭证缺少分块上传 URL 模板", "instanceId", instance.InstanceId)
		return "", hcommon.I18nError(i18n.MsgUpgradeSMHCredMissingChunkURL)
	}

	// 将本次上传任务信息写入续传记录（DB 持久化，多副本/重启后仍可断点续传）
	pendingUploadCache.StoreCtx(ctx, instance.InstanceId, &pendingUpload{
		ArchivePath: archivePath,
		ArchiveSize: archiveSize,
		FileKey:     cred.FileKey,
	})

	log.Info("[performUpgrade] 通过 TAT 在 CVM 上分块上传备份包到 SMH",
		"instanceId", instance.InstanceId,
		"totalParts", cred.TotalParts,
		"partSize", cred.PartSize,
	)

	// 断点续传：查询 SMH 上已完成的分块，重试时跳过已上传的块
	uploadedParts, rerr := smhUploadHooks.GetParts(ctx, cred.ConfirmKey)
	if rerr != nil {
		// 查询失败不阻断流程，降级为全量上传
		log.Warn("[performUpgrade] 查询已上传分块失败，降级为全量上传", "instanceId", instance.InstanceId, "error", rerr)
		uploadedParts = map[int]bool{}
	}
	if len(uploadedParts) > 0 {
		log.Info("[performUpgrade] 断点续传：跳过已完成分块",
			"instanceId", instance.InstanceId,
			"uploadedParts", len(uploadedParts),
			"totalParts", cred.TotalParts,
		)
	}

	// 预先加载脚本模板，在 Go 侧做参数替换后注册为临时脚本
	// （TAT RunCommand 的 Parameters 参数替换对 SaveCommand=false 的临时命令不生效）
	if err := smhUploadLoop(ctx, instance.InstanceId, cred, uploadedParts, archivePath, instance.RuntimeUser); err != nil {
		return "", err
	}
	// 上传完成，清除续传记录
	pendingUploadCache.DeleteCtx(ctx, instance.InstanceId)
	log.Info("[performUpgrade] 备份包已上传到 SMH", "instanceId", instance.InstanceId, "fileKey", cred.FileKey)
	return cred.FileKey, nil
}

// smhUploadLoop 执行分块上传循环：跳过已完成分块、按需续期、逐块上传、最终 Confirm。
// 该函数是 backupAndUploadToSMH 和 performUpgradeResume 的共用核心，
// 也是单元测试的直接测试目标（通过 smhUploadHooks / runScriptFn 注入 mock）。
func smhUploadLoop(ctx context.Context, instanceId string, cred *SMHUploadCredential, uploadedParts map[int]bool, archivePath string, runtimeUser string) error {
	log := Logger(ctx)

	uploadScriptTemplate, err := LoadScript("upload_to_smh.sh")
	if err != nil {
		log.Error("[smhUploadLoop] 加载上传脚本失败", "instanceId", instanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeLoadUploadScriptFailed)
	}
	for partNum := 1; partNum <= cred.TotalParts; partNum++ {
		// 断点续传：跳过已完成的分块
		if uploadedParts[partNum] {
			log.Info("[smhUploadLoop] 跳过已上传分块", "instanceId", instanceId, "partNumber", partNum, "totalParts", cred.TotalParts)
			continue
		}
		// 主动续期：如果当前凭证剩余有效期不足 10 分钟，先调用 SMH RenewMultipartUpload
		// 刷新 PartURLTemplate / PartHeaders / Expiration，避免后续分块 PUT 因凭证过期失败。
		if cred.Expiration != nil && time.Until(*cred.Expiration) < 10*time.Minute {
			log.Info("[smhUploadLoop] SMH 上传凭证临近过期，主动续期",
				"instanceId", instanceId,
				"partNumber", partNum,
				"expiration", *cred.Expiration,
			)
			if renewErr := smhUploadHooks.Renew(ctx, cred); renewErr != nil {
				log.Error("[smhUploadLoop] SMH 上传凭证主动续期失败", "instanceId", instanceId, "partNumber", partNum, "error", renewErr)
				return hcommon.I18nRichError(renewErr, i18n.MsgUpgradeRenewSMHCredFailed)
			}
		}

		partURL := strings.ReplaceAll(cred.PartURLTemplate, "{partNumber}", strconv.Itoa(partNum))
		offset := int64(partNum-1) * cred.PartSize
		uploadURLB64 := base64.StdEncoding.EncodeToString([]byte(partURL))
		log.Debug("[smhUploadLoop] 分块上传参数", "instanceId", instanceId, "partNumber", partNum, "partURL", partURL, "offset", offset)

		// 将 SMH 返回的分块 PUT Header 序列化：key 直接赋值，value 单独 base64 编码
		// 避免任何 base64 多行解析和填充截断问题
		log.Info("[smhUploadLoop] SMH 分块 PUT Headers", "instanceId", instanceId, "partNumber", partNum, "headers", cred.PartHeaders)
		headerCount := len(cred.PartHeaders)
		headerKVLines := ""
		if headerCount > 0 {
			i := 0
			for k, v := range cred.PartHeaders {
				headerKVLines += fmt.Sprintf("HEADER_%d_KEY=%q\n", i, k)
				headerKVLines += fmt.Sprintf("HEADER_%d_VAL_B64=%q\n", i, base64.StdEncoding.EncodeToString([]byte(v)))
				i++
			}
		}

		// 在 Go 侧替换占位符，只替换赋値行，校验行保持原样（仍含 {{key}} 字面量）
		uploadScript := uploadScriptTemplate
		uploadScript = strings.ReplaceAll(uploadScript, `ARCHIVE_PATH="{{archivepath}}"`, fmt.Sprintf(`ARCHIVE_PATH="%s"`, archivePath))
		uploadScript = strings.ReplaceAll(uploadScript, `UPLOAD_URL_B64="{{uploadurlb64}}"`, fmt.Sprintf(`UPLOAD_URL_B64="%s"`, uploadURLB64))
		uploadScript = strings.ReplaceAll(uploadScript, `OFFSET="{{offset}}"`, fmt.Sprintf(`OFFSET="%s"`, strconv.FormatInt(offset, 10)))
		uploadScript = strings.ReplaceAll(uploadScript, `PART_SIZE="{{partsize}}"`, fmt.Sprintf(`PART_SIZE="%s"`, strconv.FormatInt(cred.PartSize, 10)))
		uploadScript = strings.ReplaceAll(uploadScript, `PART_NUMBER="{{partnumber}}"`, fmt.Sprintf(`PART_NUMBER="%s"`, strconv.Itoa(partNum)))
		uploadScript = strings.ReplaceAll(uploadScript, `TOTAL_PARTS="{{totalparts}}"`, fmt.Sprintf(`TOTAL_PARTS="%s"`, strconv.Itoa(cred.TotalParts)))
		uploadScript = strings.ReplaceAll(uploadScript, `HEADER_COUNT={{headercount}}`, fmt.Sprintf(`HEADER_COUNT=%d`, headerCount))
		uploadScript = strings.ReplaceAll(uploadScript, `{{headerkvlines}}`, headerKVLines)

		// 注册为临时脚本名称，RunScript 内部通过 loadScript → LookupInlineScript 取到内容
		tmpName := fmt.Sprintf("_upload_smh_part%d_%d.sh", partNum, time.Now().UnixNano())
		RegisterInlineScript(tmpName, uploadScript)
		log.Info("[smhUploadLoop] 上传分块", "instanceId", instanceId, "partNumber", partNum, "totalParts", cred.TotalParts)
		_, rerr := runScriptFn(ctx, instanceId, tmpName, 600, runtimeUser, nil, nil)
		UnregisterInlineScript(tmpName)
		if rerr != nil {
			log.Error("[smhUploadLoop] CVM 上传分块到 SMH 失败", "instanceId", instanceId, "partNumber", partNum, "error", rerr, "detail", hcommon.ErrorDetailWithCtx(ctx, rerr))
			return hcommon.I18nRichError(rerr, i18n.MsgUpgradeUploadChunkFailed, partNum, cred.TotalParts)
		}
	}
	log.Info("[smhUploadLoop] 所有分块上传完成，确认 SMH 上传", "instanceId", instanceId)

	if err := smhUploadHooks.Confirm(ctx, cred.ConfirmKey); err != nil {
		log.Error("[smhUploadLoop] 确认 SMH 上传失败", "instanceId", instanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeConfirmSMHUploadFailed)
	}
	return nil
}

// runCompatScripts 在重装恢复完成后依次执行各版本兼容脚本。
// 所有脚本失败均不影响升级整体结果（仅记录 warning）。
// 后续新增兼容脚本请在此函数中追加调用。
//
// 当前包含：
//   - compat_installs_json.sh：修复 installs.json 中插件 installPath（4.x → 5.x 路径迁移）
//   - compat_plugins.sh：插件兼容修复（wecom 老名清理等）
func runCompatScripts(ctx context.Context, instance *model.Instance) {
	log := Logger(ctx)
	scripts := []struct {
		name    string
		timeout uint64
		desc    string
	}{
		{"compat_installs_json.sh", 300, "installs.json 路径修复"},
		{"compat_plugins.sh", 300, "插件兼容修复"},
	}
	for _, s := range scripts {
		log.Info("[performUpgrade] 执行版本兼容脚本", "instanceId", instance.InstanceId, "script", s.name, "desc", s.desc)
		if _, err := runScriptFn(ctx, instance.InstanceId, s.name, s.timeout, instance.RuntimeUser, nil, nil); err != nil {
			log.Warn("[performUpgrade] 版本兼容脚本执行失败（不影响升级结果）",
				"instanceId", instance.InstanceId, "script", s.name, "error", err)
		} else {
			log.Info("[performUpgrade] 版本兼容脚本执行完成", "instanceId", instance.InstanceId, "script", s.name)
		}
	}
}

// cleanupUpgradeTemp 升级完成后的收尾清理：
//   - 清理 HOME 及 /tmp 下的临时备份压缩包（防止占用机器存储）
//   - 仅保留 ~/.openclaw/upgrades/（或 ~/.hermes/upgrades/）下最近 3 个子目录，其余删除
//
// 拆分为独立脚本而非合入 restore_post_reinstall*.sh：
// 后者体积已接近 TAT 命令下发上限，继续追加逻辑会带来下发失败风险。
// 失败仅记录 warning，不影响升级整体结果。
//
// 按 agent_type 通过 ResolveScript 表驱动分派（与 backup/restore 一致），未注册类型跳过。
func cleanupUpgradeTemp(ctx context.Context, instance *model.Instance) {
	log := Logger(ctx)
	scriptName, resolveErr := ResolveScript(ctx, "cleanup_upgrade_temp", instance.AgentType)
	if resolveErr != nil {
		log.Warn("[performUpgrade] 当前 agent_type 未注册 cleanup_upgrade_temp，跳过清理",
			"instanceId", instance.InstanceId, "agentType", instance.AgentType, "error", resolveErr)
		return
	}
	log.Info("[performUpgrade] 执行升级临时文件清理脚本",
		"instanceId", instance.InstanceId, "script", scriptName)
	if _, err := runScriptFn(ctx, instance.InstanceId, scriptName, 120, instance.RuntimeUser, nil, nil); err != nil {
		log.Warn("[performUpgrade] 升级临时文件清理失败（不影响升级结果）",
			"instanceId", instance.InstanceId, "script", scriptName, "error", err)
	} else {
		log.Info("[performUpgrade] 升级临时文件清理完成",
			"instanceId", instance.InstanceId, "script", scriptName)
	}
}

// fixPluginNodeModules 升级数据恢复后，修复 extensions/*/node_modules 与 openclaw 软链。
//
// 背景：备份打包时排除了 extensions/*/node_modules，导致 restore_post_reinstall.sh
// 中"版本一致跳过重装"的插件以及非预设用户插件依赖缺失，且 ESM 原生
// import "openclaw/..." 需要 plugin_dir/node_modules/openclaw 软链。
//
// 该脚本以 runtimeUser 身份执行（不进 root 白名单），失败不阻断升级主流程，
// 仅记录 Warn 日志供事后排查。
func fixPluginNodeModules(ctx context.Context, instance *model.Instance) {
	log := Logger(ctx)
	pluginFixParams := map[string]string{}
	if _, fixErr := runScriptFn(ctx, instance.InstanceId, "restore_plugin_node_modules.sh", 600, instance.RuntimeUser, nil, pluginFixParams); fixErr != nil {
		log.Warn("[performUpgrade] 修复插件 node_modules 失败（不影响升级结果）",
			"instanceId", instance.InstanceId, "error", fixErr)
	} else {
		log.Info("[performUpgrade] 插件 node_modules 修复完成", "instanceId", instance.InstanceId)
	}
}

// approveDeviceAfterUpgrade 在升级数据恢复完成后异步执行设备审批脚本，
// 并紧接着同步 imageModel（5.7 自愈）。两步串行执行，避免与升级主流程其他
// gateway 重启动作并发互踩。
// 由 reinstallAndRestore 以 goroutine 方式调用，不阻塞升级主流程。
// 失败仅记录 warning，不影响升级结果。
func approveDeviceAfterUpgrade(ctx context.Context, instance *model.Instance) {
	log := Logger(ctx)

	// final：performUpgrade 第四步前置虽已 waitForOpenclawReady 过一次，但其后
	// restore_post_reinstall.sh 内部会 stop/start openclaw-gateway 来加载恢复后的数据，
	// 故 gateway 此刻可能仍未回到 active；若直接下发 approve_device.sh，脚本 step 0
	// 轮询触发 list 时握手仍可能无法落盘 paired.json，最终在 step 1 报 "no paired.json"。
	// 这里再 wait 一次，与新建/重装入口（approveDeviceAsync）保持一致语义。
	// 失败仅 Warn，不阻塞 imageModel 自愈步骤。
	log.Info("[performUpgrade] 设备审批前再次等待 openclaw 就绪", "instanceId", instance.InstanceId)
	if err := waitForOpenclawReady(ctx, instance.InstanceId, instance.AgentType, 5*time.Minute); err != nil {
		log.Warn("[performUpgrade] 等待 openclaw 就绪超时，跳过设备审批（不影响升级结果）",
			"instanceId", instance.InstanceId, "error", err)
		// 不 return：继续走 imageModel 自愈，让升级整体效果尽量收敛
	} else {
		log.Info("[performUpgrade] openclaw 已就绪，开始执行设备审批", "instanceId", instance.InstanceId)
		_, err := runScriptFn(ctx, instance.InstanceId, "approve_device.sh", 300, instance.RuntimeUser, nil, nil)
		if err != nil {
			log.Warn("[performUpgrade] 设备审批失败（不影响升级结果）", "instanceId", instance.InstanceId, "error", err)
		} else {
			log.Info("[performUpgrade] 设备审批完成", "instanceId", instance.InstanceId)
		}
	}

	// 5.7 imageModel 自愈：5.6 实例的 ~/.openclaw/openclaw.json 没有 imageModel 字段，
	// 必须在升级完成后主动下发一次，避免用户首次发图直接报 "No image model is configured"。
	// 必须在 approve_device.sh 之后串行执行，否则两个脚本会同时 systemctl restart openclaw-gateway，
	// 把 approve_device.sh 等待 gateway 就绪的 20s 窗口冲没（参见 v0.x.x 修复记录）。
	// 失败仅 Warn 不阻塞升级（用户后续模型变更会再次触发 buildImageModelRefs 兜底自愈）。
	if syncErr := syncInstanceModelsToCVM(ctx, instance, ""); syncErr != nil {
		log.Warn("[performUpgrade] imageModel 自愈失败（不影响升级结果，用户后续模型变更会再次触发）",
			"instanceId", instance.InstanceId, "error", syncErr)
	}
}

// redetectAndPersistRuntimeUser 在重装 + TAT Agent 就绪之后调用，以 root 身份在新系统
// 上轮询执行 detect_install 探测脚本，得到「新镜像里 hermes/openclaw/ace 实际安装在哪个
// 用户下」这一 ground truth，并把结果回写 DB + 同步 instance 内存对象。
//
// 它取代了旧的 waitRuntimeUserReady（其语义是「等 DB 旧值预期的用户出现」）。旧语义在
// 镜像侧改变默认运行用户时（例如 Hermes v0.12.0 起从 agentuser 切到 ubuntu）会必然
// 卡死 5 分钟超时——因为 DB 里旧实例的 runtime_user 仍是 agentuser，新镜像里却根本没这
// 个用户，永远等不到。
//
// 新语义（与 task/recover_hermes_runtime_user.go 的 recoverOneInstance 一致）：
//   - 探测脚本是 ground truth：扫到啥用户就用啥
//   - 探测结果 != DB 旧值 → 以探测为准，UPDATE DB 并同步 instance.RuntimeUser/RuntimeHome
//   - 探测结果非空且非 unknown 即视为就绪，立即返回
//   - 仅当连续探测全部失败/返回空（达到外层 timeout）才报错
//
// 设计要点：
//   - runtimeUser 传 "" 调 RunScript → TAT prelude 内部 root fallback，
//     不依赖目标用户存在（重装初期 agentuser/ubuntu 可能尚未创建完成）；
//   - 复用 ResolveScript("detect_install", agentType) 分派到各运行时的探测脚本，
//     不新增脚本；
//   - 单次脚本超时 30s；外层 timeout 内每 15s 轮询一次；
//   - 写回 DB 使用 model.DB(ctx).Updates，遵循多租户回调链；
//   - ctx 取消立即返回 ctx.Err()。
//
// 调用方在该函数返回 nil 后可放心使用 instance.RuntimeUser 作为后续 restore 脚本、
// post-hook、UserData 的真值（即使升级前 DB 里写的是过时的 agentuser）。
func redetectAndPersistRuntimeUser(ctx context.Context, instance *model.Instance, timeout time.Duration) error {
	log := Logger(ctx)
	if instance == nil {
		return nil
	}

	scriptName, resolveErr := ResolveScript(ctx, "detect_install", instance.AgentType)
	if resolveErr != nil {
		// 未注册探测脚本的 agent_type（理论上不应出现，因为只有支持升级的类型才会到这里）：
		// 不阻塞升级，按既有行为继续。日志告警便于排查。
		log.Warn("[redetectAndPersistRuntimeUser] 当前 agent_type 未注册 detect_install，跳过重探",
			"instanceId", instance.InstanceId, "agentType", instance.AgentType, "error", resolveErr)
		return nil
	}

	deadline := time.Now().Add(timeout)
	const pollInterval = 5 * time.Second
	attempt := 0

	type detectResult struct {
		RuntimeUser string `json:"runtime_user"`
		RuntimeHome string `json:"runtime_home"`
	}

	for {
		attempt++
		// runtimeUser 传 ""：触发 RunScript 内部 fallback 到 root，
		// 探测脚本本身只做 /home/* 文件系统扫描，不依赖目标用户存在。
		output, err := runScriptFn(ctx, instance.InstanceId, scriptName, 30, "", nil, nil)
		if err == nil {
			var result detectResult
			if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); jsonErr == nil {
				if result.RuntimeUser != "" && result.RuntimeUser != "unknown" {
					// 探测成功 → 决定是否需要回写 DB + 同步内存
					if result.RuntimeUser == instance.RuntimeUser && result.RuntimeHome == instance.RuntimeHome {
						log.Info("[redetectAndPersistRuntimeUser] 探测结果与 DB 一致，无需更新",
							"instanceId", instance.InstanceId, "runtimeUser", result.RuntimeUser,
							"runtimeHome", result.RuntimeHome, "attempt", attempt)
						return nil
					}

					// 不一致：以探测为准，写回 DB + 同步内存对象
					log.Warn("[redetectAndPersistRuntimeUser] 探测结果与 DB 不一致，以新镜像探测为准回写 DB",
						"instanceId", instance.InstanceId,
						"oldRuntimeUser", instance.RuntimeUser, "newRuntimeUser", result.RuntimeUser,
						"oldRuntimeHome", instance.RuntimeHome, "newRuntimeHome", result.RuntimeHome,
						"attempt", attempt)

					updateResult := model.DB(ctx).Model(&model.Instance{}).
						Where("id = ?", instance.ID).
						Updates(map[string]interface{}{
							"runtime_user": result.RuntimeUser,
							"runtime_home": result.RuntimeHome,
						})
					if updateResult.Error != nil {
						log.Error("[redetectAndPersistRuntimeUser] 回写 DB 失败，中止升级以避免运行用户不一致",
							"instanceId", instance.InstanceId, "error", updateResult.Error)
						return hcommon.I18nRichError(updateResult.Error, i18n.MsgUpgradePersistRuntimeUserFailed, instance.AgentType)
					}
					if updateResult.RowsAffected != 1 {
						updateErr := fmt.Errorf("instance %d runtime user update affected %d rows", instance.ID, updateResult.RowsAffected)
						log.Error("[redetectAndPersistRuntimeUser] 回写 DB 未命中实例，中止升级以避免运行用户不一致",
							"instanceId", instance.InstanceId, "error", updateErr)
						return hcommon.I18nRichError(updateErr, i18n.MsgUpgradePersistRuntimeUserFailed, instance.AgentType)
					}
					instance.RuntimeUser = result.RuntimeUser
					instance.RuntimeHome = result.RuntimeHome
					return nil
				}
				log.Info("[redetectAndPersistRuntimeUser] 探测结果暂未就绪（runtime_user 为空或 unknown），继续等待",
					"instanceId", instance.InstanceId, "detectedUser", result.RuntimeUser,
					"attempt", attempt)
			} else {
				log.Warn("[redetectAndPersistRuntimeUser] 解析探测脚本输出失败，继续等待",
					"instanceId", instance.InstanceId, "attempt", attempt,
					"output", strings.TrimSpace(output), "error", jsonErr)
			}
		} else {
			log.Warn("[redetectAndPersistRuntimeUser] 探测脚本执行失败，继续等待",
				"instanceId", instance.InstanceId, "script", scriptName,
				"attempt", attempt, "error", err)
		}

		if time.Now().After(deadline) {
			log.Error("[redetectAndPersistRuntimeUser] 重探运行用户超时",
				"instanceId", instance.InstanceId, "agentType", instance.AgentType,
				"timeout", timeout, "attempts", attempt)
			return hcommon.I18nError(i18n.MsgUpgradeRedetectRuntimeUserFailed, instance.AgentType)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// isRetryableRestoreDispatchError 仅识别 TAT 命令尚未实际执行时的暂态失败。
// 恢复脚本一旦开始执行就可能修改实例文件，脚本错误与超时必须立即返回，禁止盲目重试。
func isRetryableRestoreDispatchError(err error) bool {
	return errors.Is(err, ErrTATCommandDispatchFailed) || errors.Is(err, ErrTATCommandStartFailed)
}

// restoreDBRepairSignal 是 openclaw doctor 报告本地 SQLite 损坏时的错误片段。
// restore_post_reinstall.sh 不做特殊处理，doctor 失败时 fail_exit（exit 1），
// 该错误片段出现在脚本 stdout（TAT 装入 RichError.Detail），Go 侧据此识别。
const restoreDBRepairSignal = "database disk image is malformed"

// needLocalDBRepair 检查 restore 脚本输出或错误详情是否包含本地 DB 损坏信号。
// restore_post_reinstall.sh 的 doctor 报 "database disk image is malformed" 时 fail_exit，
// stdout 落入 RichError.Detail（而非 runScriptFn 返回的 output），故两处都查。
func needLocalDBRepair(output string, err error) bool {
	if strings.Contains(output, restoreDBRepairSignal) {
		return true
	}
	if err == nil {
		return false
	}
	var re *hcommon.RichError
	if errors.As(err, &re) {
		return strings.Contains(re.Detail, restoreDBRepairSignal)
	}
	return false
}

// backupDBUnrecoverableSignal 是 backup_pre_reinstall.sh 在“备份阶段无损修复失败”时
// 输出的机器可识别标记。ensure_healthy_sqlite 修不回本地 SQLite 时会输出该标记并 exit 1，
// 标记随脚本 stdout 落入 runScriptFn 返回的 output 或 RichError.Detail。
const backupDBUnrecoverableSignal = "BACKUP_DB_UNRECOVERABLE"

// backupDBUnrecoverable 检查备份脚本输出或错误详情是否包含“DB 不可无损修复”信号。
// 命中时调用方应中止升级（原盘尚未重装、数据仍完整），而非走普通失败重试。
func backupDBUnrecoverable(output string, err error) bool {
	if strings.Contains(output, backupDBUnrecoverableSignal) {
		return true
	}
	if err == nil {
		return false
	}
	var re *hcommon.RichError
	if errors.As(err, &re) {
		return strings.Contains(re.Detail, backupDBUnrecoverableSignal)
	}
	return false
}

// handleDBMalformedRecovery 处理 restore_post_reinstall.sh 检测到 DB malformed 的自愈编排。
// 当 restoreErr 包含 malformed 信号时，下发 openclaw_recovery.sh（resume 模式）修库，
// 然后续跑 restore_post_reinstall.sh（resume_after_doctor 模式）完成步骤 3-7。
// 返回值：
//   - newRestoreOutput: 新的 restore 输出
//   - newRestoreErr: 新的 restore 错误（nil 表示自愈成功或未触发自愈且原 err 为 nil）
//   - dbRebuiltEmpty: recovery 是否走了兜底删库重建空库
func handleDBMalformedRecovery(ctx context.Context, instance *model.Instance, restoreScript string, restoreOutput string, restoreErr error) (string, error, bool) {
	log := Logger(ctx)

	if restoreErr == nil || !needLocalDBRepair(restoreOutput, restoreErr) {
		return restoreOutput, restoreErr, false
	}

	log.Warn("[performUpgrade] 检测到本地 DB malformed，下发 recovery（resume 模式：修库+doctor+重启 gateway）",
		"instanceId", instance.InstanceId)

	// 下发 openclaw_recovery.sh（resume=true）：修库 + doctor --fix --yes + 重启 gateway，
	// 不重跑 restore_post（避免重新解压覆盖修好的库→死循环）
	recoveryParams := map[string]string{
		"resume":       "true",
		"runtime_user": getEffectiveRuntimeUser(instance.RuntimeUser),
	}
	recoveryOutput, recErr := runScriptFn(ctx, instance.InstanceId, "openclaw_recovery.sh", 600,
		instance.RuntimeUser, nil, recoveryParams)
	if recErr != nil {
		log.Error("[performUpgrade] 独立修库脚本失败", "instanceId", instance.InstanceId, "error", recErr)
		return restoreOutput, hcommon.I18nRichError(recErr, i18n.MsgUpgradeRestoreFailed), false
	}

	// recovery 兜底删库重建空库时会输出该标记：历史数据（会话/agent 状态）已丢失，
	// 坏库副本已备份。升级仍算成功（空库可用），但需额外通知用户历史数据未能恢复。
	dbRebuiltEmpty := strings.Contains(recoveryOutput, "RECOVERY_DB_REBUILT_EMPTY")
	if dbRebuiltEmpty {
		log.Warn("[performUpgrade] recovery 兜底重建空库，历史数据未能恢复（坏库已备份）",
			"instanceId", instance.InstanceId)
	}
	log.Info("[performUpgrade] 修库 + doctor + 重启 gateway 完成", "instanceId", instance.InstanceId)

	// recovery 修好了 DB + 重启 gateway，但 restore_post 的步骤 3-7
	// （doctor 配置迁移/插件适配/skillhub/浏览器配置/收尾重启）尚未执行，需续跑。
	resumeParams := map[string]string{
		"runtime_user":        getEffectiveRuntimeUser(instance.RuntimeUser),
		"resume_after_doctor": "true",
	}
	if _, resumeErr := runScriptFn(ctx, instance.InstanceId, restoreScript, 1200,
		instance.RuntimeUser, nil, resumeParams); resumeErr != nil {
		log.Error("[performUpgrade] 续跑 restore 步骤 3-7 失败", "instanceId", instance.InstanceId, "error", resumeErr)
		return restoreOutput, hcommon.I18nRichError(resumeErr, i18n.MsgUpgradeRestoreFailed), dbRebuiltEmpty
	}
	log.Info("[performUpgrade] restore 续跑完成（步骤 3-7: doctor 配置迁移/插件适配/skillhub/浏览器/收尾）", "instanceId", instance.InstanceId)
	return "RESTORE_NEED_DB_REPAIR recovered by openclaw_recovery.sh --resume + restore_post_reinstall.sh --resume-after-doctor", nil, dbRebuiltEmpty
}

// reinstallAndRestore 执行升级流程的第三~五步：
//  3. 调用 ResetInstance 重装系统、等待 RUNNING、重试 TAT 就绪；
//  4. 等 openclaw 就绪后下发 restore_post_reinstall.sh 从 SMH 下载并恢复数据；
//  5. 恢复成功后删除 SMH 备份。
//
// fileKey 为 SMH common space 中备份文件的 key（形如 backups/{instanceId}/openclaw-state-<ts>.tgz）。
// 在升级重试场景中，如果 SMH 上已有历史备份，可直接传入该 fileKey 跳过备份+上传。
func reinstallAndRestore(ctx context.Context, instance *model.Instance, defaultImageId string, fileKey string) error {
	log := Logger(ctx)

	// dbRebuiltEmpty 标记 recovery 是否走了兜底"删库重建空库"路径：
	// 若为 true，升级仍算成功但历史数据已丢失，需在成功后额外发降级告警通知。
	dbRebuiltEmpty := false

	if fileKey == "" {
		return hcommon.I18nError(i18n.MsgUpgradeFileKeyEmpty)
	}

	// 兜底确保 RuntimeUser 已就绪：
	// performUpgrade 入口已经做过一次同步探测，但 reinstallAndRestore 还会被以下路径直接调用：
	//   - HandleUpgradeRetry 命中 SMH 历史备份时跳过 performUpgrade 直接进入此函数；
	//   - performUpgradeResume 秒传命中 / 续传完成两个分支也直接进入此函数。
	// 其中 retry 路径不一定经过 performUpgrade，重启 / 多副本场景下 instance 对象内存里的
	// RuntimeUser 也可能为空。这里再走一次 ensureRuntimeUser（DB 已有则秒返回，幂等）确保
	// 后续 UserData 渲染、restore_post_reinstall_*.sh 模板变量、ready 探测都拿到正确用户。
	instance.RuntimeUser = ensureRuntimeUser(ctx, instance.ID, instance.InstanceId, instance.AgentType)
	log.Info("[reinstallAndRestore] RuntimeUser 已确认", "instanceId", instance.InstanceId, "runtimeUser", instance.RuntimeUser)

	// 提前生成 SMH 内网下载 URL（重装后直接用）
	smhURL, err := buildSMHDownloadURLFn(ctx, fileKey, true)
	if err != nil {
		log.Error("[performUpgrade] 生成 SMH 下载 URL 失败", "instanceId", instance.InstanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeBuildSMHDownloadURLFailed)
	}
	log.Info("[performUpgrade] SMH 下载 URL 已生成", "instanceId", instance.InstanceId, "fileKey", fileKey, "internalDomain", true)

	// 第三步：重装系统
	log.Info("[performUpgrade] 第三步：开始重装系统", "instanceId", instance.InstanceId, "imageId", defaultImageId)

	reinstallClient, rerr := NewCVMClient(ctx)
	if rerr != nil {
		return hcommon.I18nRichError(rerr, i18n.MsgCreateCVMClientFailed)
	}
	resetReq := cvm.NewResetInstanceRequest()
	resetReq.InstanceId = common.StringPtr(instance.InstanceId)
	resetReq.ImageId = common.StringPtr(defaultImageId)
	resetReq.EnhancedService = &cvm.EnhancedService{
		AutomationService: &cvm.RunAutomationServiceEnabled{
			Enabled: common.BoolPtr(true),
		},
		SecurityService: &cvm.RunSecurityServiceEnabled{
			Enabled: common.BoolPtr(true),
		},
		MonitorService: &cvm.RunMonitorServiceEnabled{
			Enabled: common.BoolPtr(true),
		},
	}
	// 渲染 UserData（init.sh 模板 + 实例创建时保存的用户 UserData），确保重装后 TAT Agent 和用户脚本正常执行。
	config := model.GetSiteConfig(ctx)
	var systemUserDataConfig *initUserDataConfig
	if config.SkillHub != "" {
		systemUserDataConfig = &initUserDataConfig{
			SkillHub:    config.SkillHub,
			RuntimeUser: getEffectiveRuntimeUser(instance.RuntimeUser),
			AgentType:   instance.AgentType,
		}
	}
	mergedUserData, rerr := buildUserData(ctx, systemUserDataConfig, instance.UserData)
	if rerr != nil {
		return rerr
	}
	if mergedUserData != "" {
		resetReq.UserData = common.StringPtr(mergedUserData)
		log.Debug("[performUpgrade] 已设置 UserData", "instanceId", instance.InstanceId, "skillHub", config.SkillHub, "hasUserData", instance.UserData != "")
	}
	log.Info("[performUpgrade] 调用 ResetInstance", "instanceId", instance.InstanceId, "imageId", defaultImageId)
	if _, err := resetInstanceFn(reinstallClient, resetReq); err != nil {
		log.Error("[performUpgrade] ResetInstance 失败", "instanceId", instance.InstanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgReinstallInstanceFailed)
	}
	log.Info("[performUpgrade] ResetInstance 已下发，等待实例重装完成", "instanceId", instance.InstanceId)

	// 升级成功后直写镜像缓存（失败仅记录日志，不影响主流程）
	if err := model.DB(ctx).Model(instance).Update("img_id", defaultImageId).Error; err != nil {
		log.Warn("[performUpgrade] 直写 img_id 缓存失败", "instanceId", instance.InstanceId, "error", err)
	}

	// 等待实例重装完成（状态变为 RUNNING），最长等待 15 分钟
	if err := waitForInstanceRunningFn(ctx, reinstallClient, instance.InstanceId, 15*time.Minute); err != nil {
		log.Error("[performUpgrade] 等待实例 RUNNING 超时", "instanceId", instance.InstanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeWaitReinstallTimeout)
	}

	// 重置 CLS 状态（重装后需重新安装 Agent）
	model.DB(ctx).Model(instance).Updates(map[string]interface{}{"cls_agent_status": 0})

	// 重装后检查 TAT Agent 是否就绪，最多重装 3 次
	const maxReinstallAttempts = 3
	tatReady := false
	for attempt := 1; attempt <= maxReinstallAttempts; attempt++ {
		log.Info("[performUpgrade] 等待 TAT Agent 就绪", "instanceId", instance.InstanceId, "attempt", attempt)
		// 重装后 TAT Agent 需要一段时间才能完全就绪（公共镜像需通过 UserData 安装 TAT Agent，耗时更长）
		reinstallSleepFn()

		log.Info("[performUpgrade] 检查 TAT Agent 状态", "instanceId", instance.InstanceId, "attempt", attempt)
		if waitForTATAgentOnlineFn(ctx, instance.InstanceId, 3*time.Minute) {
			tatReady = true
			break
		}

		if attempt == maxReinstallAttempts {
			break
		}

		log.Warn("[performUpgrade] TAT Agent 未就绪，尝试重新重装实例", "instanceId", instance.InstanceId, "attempt", attempt)
		if _, err := resetInstanceFn(reinstallClient, resetReq); err != nil {
			log.Error("[performUpgrade] 重新重装实例失败", "instanceId", instance.InstanceId, "attempt", attempt, "error", err)
			return hcommon.I18nRichError(err, i18n.MsgUpgradeRetryReinstallFailed, attempt)
		}
		log.Info("[performUpgrade] 重新重装已下发，等待实例恢复 RUNNING", "instanceId", instance.InstanceId, "attempt", attempt)
		if err := waitForInstanceRunningFn(ctx, reinstallClient, instance.InstanceId, 15*time.Minute); err != nil {
			log.Error("[performUpgrade] 重新重装后等待 RUNNING 超时", "instanceId", instance.InstanceId, "attempt", attempt, "error", err)
			return hcommon.I18nRichError(err, i18n.MsgUpgradeRetryReinstallWaitTimeout, attempt)
		}
	}
	if !tatReady {
		log.Error("[performUpgrade] TAT Agent 经过 3 次重装仍未就绪，升级中止", "instanceId", instance.InstanceId)
		return hcommon.I18nError(i18n.MsgUpgradeAgentReadyAttemptsExhausted, maxReinstallAttempts)
	}
	log.Info("[performUpgrade] TAT Agent 已就绪", "instanceId", instance.InstanceId)

	// 前置：在新系统上重新探测 RuntimeUser 并写回 DB。
	//
	// 重装本身就是镜像变更事件——镜像侧可能在不同版本间改变默认运行用户（典型例子：
	// Hermes v0.12.0 起从 agentuser 切到 ubuntu）。把 DB 里旧实例的 runtime_user 当真
	// 来等"agentuser 出现"会必然超时。正确做法是：以 root 身份让探测脚本扫一遍新系统
	// 的 /home/*，扫到啥用啥，并同步回 DB + 内存对象。后续 restore 脚本、UserData
	// 模板变量、二次 ready 检查全部基于这个新值。
	//
	// 兜底语义：探测连续失败到 5min 超时才报错，给一个明确信号（detect_install 脚本
	// 本身坏了 / TAT Agent 异常），而不是耗在错误期望上。
	log.Info("[performUpgrade] 第四步（前置检查 0）：重新探测 RuntimeUser 并写回 DB",
		"instanceId", instance.InstanceId, "agentType", instance.AgentType,
		"oldRuntimeUser", instance.RuntimeUser)
	if err := redetectAndPersistRuntimeUser(ctx, instance, 5*time.Minute); err != nil {
		log.Error("[performUpgrade] 重探 RuntimeUser 失败，中止升级",
			"instanceId", instance.InstanceId, "error", err)
		return err
	}
	log.Info("[performUpgrade] RuntimeUser 已对齐新镜像",
		"instanceId", instance.InstanceId, "runtimeUser", instance.RuntimeUser,
		"runtimeHome", instance.RuntimeHome)

	// 第四步：重装后数据恢复
	// 在恢复数据前，必须等待 openclaw 安装完成并服务就绪，防止恢复的数据被 openclaw 初始化流程覆盖。
	// 检查两个条件：① ~/.openclaw/openclaw.json 存在；② openclaw gateway 端口正常响应。
	log.Info("[performUpgrade] 第四步（前置检查）：等待 agent 安装完成并就绪", "instanceId", instance.InstanceId)
	if err := waitForOpenclawReady(ctx, instance.InstanceId, instance.AgentType, 20*time.Minute); err != nil {
		log.Error("[performUpgrade] 等待 agent 就绪超时，中止数据恢复", "instanceId", instance.InstanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeWaitOpenclawReadyTimeout)
	}
	log.Info("[performUpgrade] openclaw 已就绪，开始执行数据恢复", "instanceId", instance.InstanceId)

	// 数据恢复脚本同样按 agent_type 分派：openclaw → restore_post_reinstall.sh，
	// hermes → restore_post_reinstall_hermes.sh。两者参数契约 {{url}}/{{runtime_user}} 一致。
	// 用独立变量名 resolveErr 与下方 runScriptFn 的 restoreErr 区分，便于日志与错误归属一目了然。
	restoreScript, resolveErr := ResolveScript(ctx, "restore_post_reinstall", instance.AgentType)
	if resolveErr != nil {
		log.Error("[performUpgrade] 解析恢复脚本失败", "instanceId", instance.InstanceId, "agentType", instance.AgentType, "error", resolveErr)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportUpgradeWithDetail, instance.AgentType)
	}
	log.Info("[performUpgrade] 第四步：执行重装后数据恢复", "instanceId", instance.InstanceId, "script", restoreScript)
	restoreParams := map[string]string{
		"url":          smhURL,
		"runtime_user": getEffectiveRuntimeUser(instance.RuntimeUser),
	}
	// 仅 TAT 命令派发/启动暂态失败可重试；脚本自身失败或超时必须立即返回，避免重复执行恢复副作用。
	const restoreMaxRetry = 5
	var restoreErr error
	var restoreOutput string
	for attempt := 1; attempt <= restoreMaxRetry; attempt++ {
		if attempt > 1 {
			log.Warn("[performUpgrade] 数据恢复命令下发失败，等待后重试",
				"instanceId", instance.InstanceId, "attempt", attempt, "error", restoreErr)
			restoreSleepFn()
		}
		// restore 脚本包含下载+解压+venv重建+启动gateway 等6步，慢盘/大备份包场景需充裕超时
		restoreOutput, restoreErr = runScriptFn(ctx, instance.InstanceId, restoreScript, 2400, instance.RuntimeUser, nil, restoreParams)
		if restoreErr == nil {
			break
		}
		log.Warn("[performUpgrade] 数据恢复尝试失败",
			"instanceId", instance.InstanceId, "attempt", attempt, "error", restoreErr,
			"output_len", len(restoreOutput))
		if !isRetryableRestoreDispatchError(restoreErr) {
			break
		}
	}

	// 本地 SQLite 自愈编排：restore_post_reinstall.sh 的 doctor 步骤检测到
	// "database disk image is malformed" 时 fail_exit（exit 1）并把该字符串输出到
	// stdout（TAT 装入 RichError.Detail），Go 侧 needLocalDBRepair 据此识别。
	// 此时解压已落地、库已就位但损坏，由 Go 下发独立修库脚本 openclaw_recovery.sh
	// （resume 模式：修库 + doctor --fix --yes + 重启 gateway），跳过 restore_post 的
	// 下载+解压（库已就位，重跑会覆盖修好的库→死循环），在 recovery 内完成恢复。
	restoreOutput, restoreErr, dbRebuiltEmpty = handleDBMalformedRecovery(ctx, instance, restoreScript, restoreOutput, restoreErr)

	if restoreErr != nil {
		log.Error("[performUpgrade] 数据恢复失败", "instanceId", instance.InstanceId, "error", restoreErr,
			"output_len", len(restoreOutput))
		return hcommon.I18nRichError(restoreErr, i18n.MsgUpgradeRestoreFailed)
	}
	log.Info("[performUpgrade] 数据恢复完成", "instanceId", instance.InstanceId,
		"output_len", len(restoreOutput))

	// 升级后置 hook：按 runtime type 从 upgradePostHookTable 分派，未注册类型 no-op。
	runtimeType := model.GetAgentRuntimeType(ctx, instance.AgentType)
	if hookFn, ok := upgradePostHookTable[runtimeType]; ok {
		if hookErr := hookFn(ctx, instance); hookErr != nil {
			return hookErr
		}
	} else {
		log.Info("[performUpgrade] 当前 runtime 无升级后置 hook，直接进入收尾",
			"instanceId", instance.InstanceId, "agentType", instance.AgentType)
	}

	// 重装完成后重置 agent_ready，确保前端显示 loading 而非 running，等待新 Agent 就绪
	if err := model.DB(ctx).Model(instance).Updates(map[string]interface{}{"agent_ready": 0}).Error; err != nil {
		log.Error("[performUpgrade] 重置 agent_ready 失败", "instanceId", instance.InstanceId, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgUpgradeResetAgentReadyFailed)
	}

	// 升级完成后主动拉取新版本信息（openclaw 已就绪，拉取成功会直接覆盖旧版本数据）
	log.Info("[performUpgrade] 主动拉取升级后的版本信息", "instanceId", instance.InstanceId)
	if err := fetchVersionInfoFn(ctx, *instance); err != nil {
		// 版本拉取失败不影响升级结果，后续 AgentChecker 就绪时会再次拉取
		log.Warn("[performUpgrade] 拉取版本信息失败（不影响升级结果）", "instanceId", instance.InstanceId, "error", err)
	}

	// 第八步：恢复成功后删除 SMH 上的备份目录（释放存储空间，连同历史残留一并清理）
	// 目录形如 backups/{instanceId}/，里面可能有多个历史备份文件（含上次失败/重试残留）
	backupDir := "backups/" + instance.InstanceId
	log.Info("[performUpgrade] 第八步：删除 SMH 备份目录", "instanceId", instance.InstanceId, "backupDir", backupDir)
	if err := DeleteSMHCommonDirectory(ctx, backupDir); err != nil {
		// 删除失败不影响升级结果，仅记录警告
		log.Warn("[performUpgrade] 删除 SMH 备份目录失败（不影响升级结果）", "instanceId", instance.InstanceId, "backupDir", backupDir, "error", err)
	} else {
		log.Info("[performUpgrade] SMH 备份目录已清空", "instanceId", instance.InstanceId, "backupDir", backupDir)
	}

	// recovery 兜底重建空库：历史数据未能恢复，额外发一条降级提示通知（升级本身仍算成功）。
	if dbRebuiltEmpty {
		go model.CreateSuccessNotification(
			hcommon.DetachContext(ctx),
			instance.UserID, instance.ID, instance.Name,
			model.NotifyTypeInstanceUpgradeSuccess,
			i18n.T(hcommon.DetachContext(ctx), i18n.MsgUpgradeSuccessDBRebuiltTitle),
			i18n.T(hcommon.DetachContext(ctx), i18n.MsgUpgradeSuccessDBRebuiltContent, instance.Name),
		)
	}

	log.Info("[performUpgrade] 升级全流程完成", "instanceId", instance.InstanceId)
	return nil
}

// HandleUpgradeRetry 升级失败后的重试入口。
// POST /openclaw/upgrade/retry?id={instanceID}
//
// 行为：
//  1. 校验当前实例处于"升级失败"状态（CurrentOperation=upgrade, CurrentOperationState=failed）
//  2. 查询 SMH common space 的 backups/{instanceId}/ 目录下是否存在历史备份文件：
//     - 存在：取最新备份，跳过备份+上传环节，直接执行重装 + 数据恢复（reinstallAndRestore）
//     - 不存在：走完整升级流程（performUpgrade：备份 → 上传 → 重装 → 恢复）
func HandleUpgradeRetry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	log.Info("[UpgradeRetry] 收到升级重试请求", "method", r.Method, "url", r.URL.String())
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		log.Warn("[UpgradeRetry] 用户未登录，拒绝请求")
		return
	}

	if r.Method != http.MethodPost {
		log.Warn("[UpgradeRetry] 非 POST 请求", "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		log.Error("[UpgradeRetry] 获取实例失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if instance.IsDoctorNode {
		log.Warn("[UpgradeRetry] 拒绝龙虾医生节点", "user_id", user.ID, "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}
	log.Info("[UpgradeRetry] 目标实例",
		"instanceId", instance.InstanceId,
		"name", instance.Name,
		"currentOperation", instance.CurrentOperation,
		"currentOperationState", instance.CurrentOperationState,
	)

	// 前置检查 1：仅支持 SupportsUpgrade=true 的类型（与 HandleUpgrade 保持一致）
	if err := checkInstanceSupportsUpgrade(ctx, instance); err != nil {
		log.Warn("[UpgradeRetry] 该类型不支持一键升级",
			"instanceId", instance.InstanceId, "agent_type", instance.AgentType)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 前置检查 2：必须处于"升级失败"状态才允许重试
	// 依赖 performUpgrade 失败路径使用 markOperationFailed 保留 current_operation=upgrade 的语义
	if instance.CurrentOperation != model.OpUpgrade || instance.CurrentOperationState != model.OpStateFailed {
		log.Warn("[UpgradeRetry] 当前状态不允许重试",
			"instanceId", instance.InstanceId,
			"currentOperation", instance.CurrentOperation,
			"currentOperationState", instance.CurrentOperationState,
		)
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgUpgradeRetryNotInFailedState,
				instance.CurrentOperation, instance.CurrentOperationState))
		return
	}

	// 获取后台配置的默认（目标）镜像
	defaultImage, err := model.GetEnabledImageByType(r.Context(), instance.AgentType)
	if err != nil {
		log.Error("[UpgradeRetry] 查询启用镜像失败",
			"instance_id", instance.ID, "agent_type", instance.AgentType, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
		return
	}
	if defaultImage == nil {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		log.Error("[UpgradeRetry] 未找到该类型已启用的镜像",
			"instance_id", instance.ID, "agent_type", instance.AgentType)
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgUpgradeNoImageForType, typeName))
		return
	}
	log.Info("[UpgradeRetry] 目标镜像", "imageId", defaultImage.ImageId, "imageName", defaultImage.ImageName)

	// 前置检查：拒绝降级（覆盖官方镜像 + 自定义启用镜像两类来源）。
	// 与 HandleUpgrade / HandleAdminBatchUpgrade 共享 rejectDowngradeOnOfficialImage 纯函数，
	// 防止灰度/自定义版本被低版本目标镜像"降级"覆盖；错误文案在被调函数内根据 IsCandidateImage
	// 区分官方/自定义两类，便于运维与用户定位问题来源。
	//
	// 注意：retry 入口未复用 prepareInstanceForUpgrade（其前置语义不同：必须 failed 状态、
	// 不做 providerKeys / 防重入），但本项是与"是否首次升级"无关的安全检查，必须在所有
	// 升级入口同步执行；放在 SMH 备份查询之前，避免对一个本来就要被拒绝的请求做远程查询。
	if err := rejectDowngradeOnOfficialImage(ctx, instance, defaultImage); err != nil {
		log.Warn("[UpgradeRetry] 实例版本高于目标镜像版本，拒绝降级",
			"instanceId", instance.InstanceId,
			"currentVersion", instance.AgentVersion,
			"targetImageId", defaultImage.ImageId,
			"targetVersion", defaultImage.AgentVersion,
			"isOfficialImage", hcommon.IsCandidateImage(defaultImage.ImageId),
			"error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 查询当前运行时在 SMH 上的历史备份（无时效限制），避免 Hermes/OpenClaw 备份互相误用。
	existingFileKey, hasBackup, err := FindLatestSMHCommonBackup(ctx, instance.InstanceId, instance.AgentType)
	if err != nil {
		// 查询 SMH 失败不阻断重试，降级走完整备份+上传流程
		log.Warn("[UpgradeRetry] 查询 SMH 备份失败，降级走完整升级流程",
			"instanceId", instance.InstanceId, "error", err)
		hasBackup = false
		existingFileKey = ""
	}
	log.Info("[UpgradeRetry] SMH 备份查询结果",
		"instanceId", instance.InstanceId, "hasBackup", hasBackup, "fileKey", existingFileKey)

	// 短路：无历史备份 + 当前版本已与目标镜像一致 → 视为"升级已完成"，不再下发任何任务。
	// 这一分支杜绝"无备份 + 版本一致却仍走完整重装"的场景下可能发生的长耗时 / 超时问题。
	if !hasBackup {
		if same, verr := isInstanceVersionSameAsImage(ctx, instance, defaultImage); verr != nil {
			log.Warn("[UpgradeRetry] 版本比对异常，按需要升级处理",
				"instanceId", instance.InstanceId, "error", verr)
		} else if same {
			log.Info("[UpgradeRetry] 无历史备份且版本已与目标镜像一致，视为升级已完成",
				"instanceId", instance.InstanceId,
				"agentVersion", instance.AgentVersion,
				"imageAgentVersion", defaultImage.AgentVersion)
			// 当前为升级失败态（CurrentOperation=upgrade, state=failed），将其清成 success，
			// 释放操作锁，前端可直接从"失败"切换为"正常"态。
			if cerr := clearOperation(model.DB(ctx), instance, model.OpStateSuccess); cerr != nil {
				log.Error("[UpgradeRetry] 版本一致但清除操作锁失败",
					"instanceId", instance.InstanceId, "error", cerr)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(cerr, i18n.MsgUpgradeClearFailedStateFailed))
				return
			}
			clearAdjustmentFailure(ctx, instance.ID)
			jsonOK(w, i18n.T(r.Context(), i18n.MsgUpgradeRetryAlreadyLatest))
			return
		}
	}

	// 前置：若本次升级的目标镜像是官方公共镜像，强制把 runtime_user/home 对齐为 root / /root，
	// 防止重装后 restore_post_reinstall.sh 在新官方镜像上以不存在的用户（如 ubuntu）身份执行。
	if err := ensureOfficialImageRuntimeUserForUpgrade(ctx, instance, defaultImage.ImageId, "[UpgradeRetry]"); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 加操作锁（允许覆盖 current_operation_state=failed）
	if err := setOperationForRetry(model.DB(ctx), instance, model.OpUpgrade); err != nil {
		log.Error("[UpgradeRetry] 设置重试操作锁失败", "instanceId", instance.InstanceId, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgUpgradeRetrySetOpLockFailed))
		return
	}

	// 异步执行升级：有备份走快速路径（跳过备份+上传），无备份走完整流程
	go func(ctx context.Context) {
		var retryErr error
		if hasBackup && existingFileKey != "" {
			log.Info("[UpgradeRetry] 命中 SMH 历史备份，跳过备份+上传，直接重装恢复",
				"instanceId", instance.InstanceId, "fileKey", existingFileKey)
			retryErr = reinstallAndRestore(ctx, instance, defaultImage.ImageId, existingFileKey)
			// reinstallAndRestore 不负责操作锁的释放，这里手动 finalize
			finalizeUpgradeResult(ctx, instance, retryErr)
		} else if pending, ok := pendingUploadCache.LoadCtx(ctx, instance.InstanceId); ok {
			// 命中续传记录（DB 持久化，多副本 / 重启后仍可恢复）：跳过重新备份，直接续传
			log.Info("[UpgradeRetry] 命中未完成上传记录，尝试断点续传",
				"instanceId", instance.InstanceId,
				"archivePath", pending.ArchivePath,
				"fileKey", pending.FileKey,
			)
			// performUpgradeResume 内部 defer 会调用 finalizeUpgradeResult
			retryErr = performUpgradeResume(ctx, instance, defaultImage.ImageId, pending)
		} else {
			log.Info("[UpgradeRetry] 未找到 SMH 历史备份，走完整升级流程", "instanceId", instance.InstanceId)
			// performUpgrade 内部 defer 会调用 finalizeUpgradeResult，无需重复处理
			retryErr = performUpgrade(ctx, instance, defaultImage.ImageId, "")
		}
		if retryErr != nil {
			log.Error("[UpgradeRetry] 异步重试升级失败", "instanceId", instance.InstanceId, "error", retryErr)
		}
	}(hcommon.DetachContext(ctx))
	clearAdjustmentFailure(ctx, instance.ID)

	log.Info("[UpgradeRetry] 升级重试任务已提交，返回异步响应", "instanceId", instance.InstanceId, "reuseBackup", hasBackup)
	if hasBackup {
		jsonOK(w, i18n.T(r.Context(), i18n.MsgUpgradeRetryStartedWithBackup))
	} else {
		jsonOK(w, i18n.T(r.Context(), i18n.MsgUpgradeRetryStarted))
	}
}

// checkOpenclawConfigFn 是 checkOpenclawConfigProviderKeys 调用 RunInlineScript 的可替换包装，
// 方便单元测试 mock TAT 调用。
var checkOpenclawConfigFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
	return RunInlineScript(ctx, instanceId, scriptContent, timeout)
}

// providerKeyForbiddenChars 定义 models.providers key 中不允许出现的字符列表。
// 后续如需新增限制，只需在此处追加即可，检查逻辑无需改动。
var providerKeyForbiddenChars = []string{"/"}

// checkOpenclawConfigProviderKeys 通过 TAT 读取实例上的 /root/.openclaw/openclaw.json，
// 检查 models.providers 下的所有 key 是否包含 providerKeyForbiddenChars 中定义的非法字符。
//
// 背景：openclaw.json 中 models.providers 的 key 用于标识 provider，
// 格式应为不含非法字符的字符串（如 "hatchery-qwen3.6-plus"）。
// 若 key 中包含非法字符（如 "hatchery-qwen3.6-plus/qwen3.6-plus" 含 "/"），
// 则该配置不合法，升级前必须修复，否则升级后配置可能无法正常工作。
//
// 返回：
//   - nil：配置文件不存在、无法读取或 providers key 均合法，允许继续升级；
//   - error：发现含非法字符的 provider key，返回具体错误信息，调用方应拒绝升级。
func checkOpenclawConfigProviderKeys(ctx context.Context, instance *model.Instance) error {
	log := Logger(ctx)
	if instance == nil {
		return nil
	}

	// 仅 OpenClaw 实例存在 ~/.openclaw/openclaw.json，Hermes / ACE 走自身配置文件
	// （Hermes 是 ~/.hermes/config.yaml，结构完全不同），不需要也不能用本检查。
	if model.GetAgentRuntimeType(ctx, instance.AgentType) != model.AgentTypeOpenClaw {
		log.Info("[checkOpenclawConfig] 非 OpenClaw 运行时，跳过 providers key 检查",
			"instanceId", instance.InstanceId, "agentType", instance.AgentType)
		return nil
	}

	// 通过 TAT 读取配置文件内容；文件不存在时 cat 返回非零退出码，视为"无配置"，允许升级
	script := `cat /root/.openclaw/openclaw.json 2>/dev/null || true`
	output, err := checkOpenclawConfigFn(ctx, instance.InstanceId, script, 15)
	if err != nil {
		// TAT 执行失败（如 Agent 离线）时不阻断升级，仅记录警告
		log.Warn("[checkOpenclawConfig] 读取 openclaw.json 失败，跳过检查",
			"instanceId", instance.InstanceId, "error", err)
		return nil
	}

	output = strings.TrimSpace(output)
	if output == "" {
		// 文件不存在或为空，无需检查
		log.Info("[checkOpenclawConfig] openclaw.json 不存在或为空，跳过检查",
			"instanceId", instance.InstanceId)
		return nil
	}

	// 解析 JSON，只取 models.providers 的 key 列表
	var cfg struct {
		Models struct {
			Providers map[string]json.RawMessage `json:"providers"`
		} `json:"models"`
	}
	if err := json.Unmarshal([]byte(output), &cfg); err != nil {
		// JSON 解析失败时不阻断升级，仅记录警告
		log.Warn("[checkOpenclawConfig] 解析 openclaw.json 失败，跳过检查",
			"instanceId", instance.InstanceId, "error", err)
		return nil
	}

	// 检查每个 provider key 是否包含 providerKeyForbiddenChars 中定义的非法字符
	var invalidKeys []string
	for key := range cfg.Models.Providers {
		for _, ch := range providerKeyForbiddenChars {
			if strings.Contains(key, ch) {
				invalidKeys = append(invalidKeys, key)
				break
			}
		}
	}

	if len(invalidKeys) == 0 {
		log.Info("[checkOpenclawConfig] models.providers key 检查通过",
			"instanceId", instance.InstanceId, "providerCount", len(cfg.Models.Providers))
		return nil
	}

	log.Warn("[checkOpenclawConfig] 发现非法 provider key（含非法字符），拒绝升级",
		"instanceId", instance.InstanceId, "invalidKeys", invalidKeys,
		"forbiddenChars", providerKeyForbiddenChars)
	return hcommon.I18nError(i18n.MsgUpgradeInvalidModelProvider, strings.Join(providerKeyForbiddenChars, " "), strings.Join(invalidKeys, "、"))
}

// upgradePostHookFunc 统一升级后置 hook 签名，返回 error 用于标记升级失败（nil 不阻断）。
type upgradePostHookFunc func(ctx context.Context, instance *model.Instance) error

// upgradePostHookTable 按 runtime type 注册后置 hook，新增 runtime 只需注册一行。
var upgradePostHookTable = map[string]upgradePostHookFunc{
	model.AgentTypeOpenClaw: runOpenClawUpgradePostHooks,
	model.AgentTypeHermes:   runHermesUpgradePostHooks,
}

// runOpenClawUpgradePostHooks 执行 OpenClaw 升级后 5 项私有补丁：
// 1. sync_gateway_port.sh  2. fixPluginNodeModules  3. runCompatScripts
// 4. cleanupUpgradeTemp    5. approveDeviceAfterUpgrade
// 所有失败均 warn 不阻断，始终返回 nil。
func runOpenClawUpgradePostHooks(ctx context.Context, instance *model.Instance) error {
	log := Logger(ctx)
	// 第四步（补丁 A）：同步 gateway 端口
	// restore_post_reinstall.sh 恢复了 openclaw.json（含用户自定义 gateway.port），
	// 但 stage_pre_restore 中 openclaw gateway install 生成的 service 文件使用默认端口，
	// 两者可能不一致，导致 WS 连接不可用。此处单独下发小脚本完成端口同步并重启 gateway。
	syncPortParams := map[string]string{
		"runtime_user": getEffectiveRuntimeUser(instance.RuntimeUser),
	}
	if _, syncPortErr := runScriptFn(ctx, instance.InstanceId, "sync_gateway_port.sh", 60, instance.RuntimeUser, nil, syncPortParams); syncPortErr != nil {
		// 端口同步失败不阻断升级流程，仅记录警告
		log.Warn("[performUpgrade] gateway 端口同步脚本执行失败（不影响升级结果）", "instanceId", instance.InstanceId, "error", syncPortErr)
	} else {
		log.Info("[performUpgrade] gateway 端口同步完成", "instanceId", instance.InstanceId)
	}

	// 第四步（补丁 B）：修复插件 node_modules 与 openclaw 软链
	fixPluginNodeModules(ctx, instance)

	// 第五步（后置）：版本兼容脚本
	runCompatScripts(ctx, instance)

	// 第六步（收尾清理）：清理 /tmp 残留压缩包 + ~/.openclaw/upgrades/ 旧快照（仅保留最近 3 个）
	// 拆为独立脚本而非合入 restore_post_reinstall.sh：后者体积已接近 TAT 命令下发上限，
	// 继续追加逻辑可能导致下发失败，因此用 RunScript 单独下发执行。
	cleanupUpgradeTemp(ctx, instance)

	// 第七步（后置）：同步执行设备审批 + imageModel 自愈（失败均 warn 不阻断）。
	approveDeviceAfterUpgradeFn(ctx, instance)
	return nil
}

// approveDeviceAfterUpgradeFn 供测试替换的间接调用点。
var approveDeviceAfterUpgradeFn = approveDeviceAfterUpgrade

// runHermesUpgradePostHooks 执行 Hermes 升级后置：二次 ready 兜底 → 通道兼容 → 清理临时文件。
// 所有失败均 warn 不阻断（与 OpenClaw 语义对齐），agent_ready 由 AgentChecker 兜底。
func runHermesUpgradePostHooks(ctx context.Context, instance *model.Instance) error {
	log := Logger(ctx)
	if readyErr := waitForOpenclawReady(ctx, instance.InstanceId, instance.AgentType, 5*time.Minute); readyErr != nil {
		log.Warn("[performUpgrade] 数据恢复后 agent 二次就绪探测失败（不阻断升级，由 AgentChecker 兜底）",
			"instanceId", instance.InstanceId, "runtimeType", model.AgentTypeHermes, "error", readyErr)
	} else {
		log.Info("[performUpgrade] 数据恢复后 agent 已就绪", "instanceId", instance.InstanceId, "runtimeType", model.AgentTypeHermes)
	}

	// 通道依赖兼容修复：恢复 dingtalk/msteams 等可选 pip 包（参照 OpenClaw compat_plugins.sh 模式）
	// 失败不阻断升级流程（warn 级），与 OpenClaw runCompatScripts 错误处理语义保持一致
	log.Info("[performUpgrade] 执行 Hermes 通道依赖兼容脚本", "instanceId", instance.InstanceId)
	compatOutput, compatErr := runScriptFn(ctx, instance.InstanceId, "compat_channels_hermes.sh", 300, instance.RuntimeUser, nil, nil)
	if compatErr != nil {
		log.Warn("[performUpgrade] Hermes 通道兼容脚本执行失败（不影响升级结果）",
			"instanceId", instance.InstanceId, "error", compatErr, "output", compatOutput)
	} else {
		log.Info("[performUpgrade] Hermes 通道兼容脚本执行完成",
			"instanceId", instance.InstanceId, "output", compatOutput)
	}

	// 收尾清理：清理 /tmp 残留 + ~/.hermes/upgrades/ 保留最近 3 份。
	// cleanupUpgradeTemp 内部按 agent runtime 分派到 cleanup_upgrade_temp_hermes.sh。
	// 失败仅 warn，不阻塞升级主流程。
	cleanupUpgradeTemp(ctx, instance)

	return nil
}
