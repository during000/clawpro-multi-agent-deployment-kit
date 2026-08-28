package controller

import (
	"bytes"
	"compress/gzip"
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
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// skillsChunkSize 分块传输时每块的原始字节数（base64 前）。
// 16KB 原始数据 → ~21.3KB base64，安全落在 TAT 24KB 输出上限内。
const skillsChunkSize = 16000

// runInlineScript 是 RunInlineScript 的可替换版本，供测试 mock 使用。
var runInlineScript = RunInlineScript

// listSkillsScriptRunner 是 listInstanceSkills 中 RunScript 的可替换版本，供测试 mock 使用。
var listSkillsScriptRunner = RunScript

// listInstanceSkills 通过 TAT 查询实例已安装的技能列表，返回原始 JSON 字符串。
// final：按 agent_type 分派（list_skills / list_skills_hermes / list_skills_ace），
// 避免 hermes/ace 实例跑 openclaw 专属脚本。
//
// 兼容 TAT 24KB 输出限制，三模式自动检测：
//   - 模式 1（分块）：脚本输出 {"mode":"file","path":"...","size":N,"chunks":N}，
//     Go 通过多次 TAT 调用逐块读取 → 合并 → 解压，容量无上限。
//   - 模式 2（压缩）：脚本输出 base64(gzip(JSON))，单次 TAT 调用，适合 ≤16KB 压缩数据。
//   - 模式 3（回退）：旧版脚本直接输出原始 JSON，透传。
func listInstanceSkills(ctx context.Context, instanceId string, runtimeUser string, agentType string) (string, error) {
	log := Logger(ctx)
	scriptName, rerr := ResolveScript(ctx, "list_skills", agentType)
	if rerr != nil {
		return "", hcommon.I18nRichError(rerr, i18n.MsgParseListSkillsScriptFailed, agentType)
	}
	output, err := listSkillsScriptRunner(ctx, instanceId, scriptName, 60, runtimeUser, nil, nil)
	if err != nil {
		log.Error("[skills] RunScript 查询技能列表失败",
			"instance", instanceId, "script", scriptName, "agent_type", agentType, "error", err)
		return output, err
	}

	trimmed := strings.TrimSpace(output)

	// 模式 1：分块传输 — 数据过大无法通过单次 TAT base64 传输
	var fileMeta struct {
		Mode   string `json:"mode"`
		Path   string `json:"path"`
		Size   int    `json:"size"`
		Chunks int    `json:"chunks"`
	}
	if json.Unmarshal([]byte(trimmed), &fileMeta) == nil && fileMeta.Mode == "file" && fileMeta.Path != "" {
		log.Info("[skills] 进入分块传输模式",
			"instance", instanceId, "file", fileMeta.Path,
			"size", fileMeta.Size, "chunks", fileMeta.Chunks)
		return readSkillsViaChunks(ctx, instanceId, fileMeta.Path, fileMeta.Size, fileMeta.Chunks)
	}

	// 模式 2：单次 base64(gzip(JSON)) 压缩传输（≤16KB gzip 数据，常规路径）
	if decoded, decErr := base64.StdEncoding.DecodeString(trimmed); decErr == nil {
		if reader, gzErr := gzip.NewReader(bytes.NewReader(decoded)); gzErr == nil {
			defer reader.Close()
			if decompressed, readErr := io.ReadAll(reader); readErr == nil {
				return string(decompressed), nil
			}
		}
	}

	// 模式 3：回退 — 旧版脚本直接输出原始 JSON，透传
	return output, nil
}

// readSkillsViaChunks 通过并发 TAT 调用逐块读取实例上的临时 gzip 文件，
// 合并后解压返回原始 JSON。每块 16KB 原始数据经 base64 编码后约 21.3KB，
// 安全落在 TAT 24KB 输出上限内。
//
// 关键点：
//  1. 使用 `dd skip=OFFSET count=SIZE iflag=skip_bytes,count_bytes` 精确读取，
//     不依赖 `tail | head | base64` 管道（后者在 head 提前退出触发 SIGPIPE 时
//     会概率性截断 tail 输出，导致 chunk 数据不足）。
//  2. 服务端脚本已通过 fileMeta.size 明确告知 gzip 文件精确字节数，
//     每个 chunk 的期望大小是可精确计算的常量，最后一块也不例外。
//  3. 拿到 chunk 后严格校验 len == expected，不等直接判定为 TAT stdout 传输截断，
//     此时才走重试（每次都是全新 TAT invocation，能拿到完整数据）。
//
// 最多 5 路并发：5 块约 2s（顺序 10s → 5x 提升），10 块约 4s（顺序 20s）。
func readSkillsViaChunks(ctx context.Context, instanceId string, filePath string, totalSize int, chunks int) (string, error) {
	const maxConcurrency = 5
	const maxRetries = 3
	const retryBackoff = 500 * time.Millisecond

	log := Logger(ctx)

	if totalSize <= 0 || chunks <= 0 {
		return "", hcommon.I18nRichError(
			fmt.Errorf("invalid chunk meta: size=%d chunks=%d", totalSize, chunks),
			i18n.MsgTATQueryResultFailed)
	}

	type chunkResult struct {
		data []byte
		err  error
	}

	results := make([]chunkResult, chunks)
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i := 0; i < chunks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			offset := idx * skillsChunkSize
			// 每个 chunk 的精确期望字节数（最后一块可能小于 skillsChunkSize）
			expected := skillsChunkSize
			if remain := totalSize - offset; remain < expected {
				expected = remain
			}

			// 使用 dd + iflag=skip_bytes,count_bytes 精确读取 [offset, offset+expected)
			// dd 是 seek-based 读取（read(2) 直到累计 count_bytes），不涉及管道 SIGPIPE，
			// 只要文件存在且长度足够就一定能读到精确字节数。
			// status=none 抑制 dd 的进度输出；base64 -w0 单行输出，避免换行影响 TAT stdout。
			script := fmt.Sprintf(
				"dd if=%s bs=%d count=%d skip=%d iflag=skip_bytes,count_bytes status=none 2>/dev/null | base64 -w0",
				filePath, expected, expected, offset)

			var chunkB64 string
			var decoded []byte
			var lastErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				if attempt > 0 {
					time.Sleep(retryBackoff * time.Duration(attempt))
				}
				chunkB64, lastErr = runInlineScript(ctx, instanceId, script, 30)
				if lastErr != nil {
					if !isTATRetryableError(lastErr) {
						break
					}
					continue
				}
				decoded, lastErr = base64.StdEncoding.DecodeString(strings.TrimSpace(chunkB64))
				if lastErr != nil {
					lastErr = fmt.Errorf("decode chunk %d: %w", idx+1, lastErr)
					break
				}
				// 精确校验：len(decoded) 必须严格等于预期字节数。
				// 不等意味着 TAT stdout 传输被截断（非最后一块的常见问题），重试拿全新数据。
				if len(decoded) != expected {
					log.Warn("[skills][chunks] 分块数据长度不符，重试",
						"instance", instanceId,
						"chunk", fmt.Sprintf("%d/%d", idx+1, chunks),
						"got", len(decoded), "expected", expected,
						"b64_len", len(chunkB64),
						"attempt", attempt+1)
					lastErr = fmt.Errorf("chunk data length mismatch: got %d bytes, expected %d", len(decoded), expected)
					continue
				}
				lastErr = nil
				break
			}
			if lastErr != nil {
				results[idx] = chunkResult{err: fmt.Errorf("chunk %d/%d: %w (retried %d times)", idx+1, chunks, lastErr, maxRetries)}
				return
			}
			results[idx] = chunkResult{data: decoded}
			log.Info("[skills][chunks] 分块读取成功",
				"instance", instanceId,
				"chunk", fmt.Sprintf("%d/%d", idx+1, chunks),
				"offset", offset, "expected", expected,
				"b64_len", len(chunkB64), "decoded_bytes", len(decoded))
		}(i)
	}

	wg.Wait()

	// 异步清理实例上的临时文件
	go func() {
		cleanupCtx, cancel := context.WithTimeout(hcommon.DetachContext(ctx), 15*time.Second)
		defer cancel()
		runInlineScript(cleanupCtx, instanceId, fmt.Sprintf("rm -f %s", filePath), 10)
	}()

	// 检查错误并按索引合并数据
	var buf bytes.Buffer
	buf.Grow(totalSize)
	for i, r := range results {
		if r.err != nil {
			log.Error("[skills][chunks] 分块读取失败，即将清理并返回",
				"instance", instanceId, "file", filePath,
				"chunk", fmt.Sprintf("%d/%d", i+1, chunks), "error", r.err)
			// best-effort 同步清理
			cleanupCtx, cancel := context.WithTimeout(hcommon.DetachContext(ctx), 10*time.Second)
			defer cancel()
			runInlineScript(cleanupCtx, instanceId, fmt.Sprintf("rm -f %s", filePath), 10)
			return "", hcommon.I18nRichError(r.err, i18n.MsgTATQueryResultFailed)
		}
		if r.data == nil {
			log.Error("[skills][chunks] 分块结果为空",
				"instance", instanceId, "file", filePath,
				"chunk", fmt.Sprintf("%d/%d", i+1, chunks))
			return "", hcommon.I18nRichError(
				fmt.Errorf("chunk %d/%d: 结果为空", i+1, chunks),
				i18n.MsgTATQueryResultFailed)
		}
		buf.Write(r.data)
	}

	// 双重保险：合并后的总字节数应严格等于服务端声明的 totalSize
	if buf.Len() != totalSize {
		log.Error("[skills][chunks] 合并后总长度与预期不符",
			"instance", instanceId, "file", filePath,
			"total_bytes", buf.Len(), "expected", totalSize, "chunks", chunks)
		return "", hcommon.I18nRichError(
			fmt.Errorf("assembled size mismatch: got %d, expected %d", buf.Len(), totalSize),
			i18n.MsgTATQueryResultFailed)
	}

	log.Info("[skills][chunks] 所有分块读取完成，开始解压合并",
		"instance", instanceId, "file", filePath, "chunks", chunks, "total_bytes", buf.Len())

	reader, err := gzip.NewReader(&buf)
	if err != nil {
		log.Error("[skills][chunks] 分块数据gzip解压失败",
			"instance", instanceId, "file", filePath, "chunks", chunks, "total_bytes", buf.Len(), "error", err)
		return "", hcommon.I18nRichError(
			fmt.Errorf("decompress chunked skills: %w", err),
			i18n.MsgTATQueryResultFailed)
	}
	defer reader.Close()
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		log.Error("[skills][chunks] 分块解压数据读取失败",
			"instance", instanceId, "file", filePath, "chunks", chunks, "error", err)
		return "", hcommon.I18nRichError(
			fmt.Errorf("read decompressed chunked skills: %w", err),
			i18n.MsgTATQueryResultFailed)
	}
	return string(decompressed), nil
}

// isTATRetryableError 判断 TAT 调用错误是否应重试。
// 仅对瞬时网络错误（如 TLS handshake timeout、连接被拒绝、临时 DNS 失败等）返回 true。
func isTATRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// TLS/SSL 握手超时
	if strings.Contains(msg, "TLS handshake timeout") {
		return true
	}
	// 连接被拒绝/重置
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "connection reset") {
		return true
	}
	// 临时网络不可达
	if strings.Contains(msg, "no route to host") || strings.Contains(msg, "network is unreachable") {
		return true
	}
	// DNS 临时解析失败
	if strings.Contains(msg, "no such host") || strings.Contains(msg, "temporary failure") {
		return true
	}
	// HTTP timeout / EOF（TCP 连接中断）
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "EOF") {
		return true
	}
	// i/o timeout
	if strings.Contains(msg, "i/o timeout") {
		return true
	}
	// 通用网络错误
	if strings.Contains(msg, "broken pipe") {
		return true
	}
	// TAT API 限频（RequestLimitExceeded），稍后退避重试
	if strings.Contains(msg, "RequestLimitExceeded") {
		return true
	}
	return false
}

func HandleAddSkill(w http.ResponseWriter, r *http.Request) {
	handleAddSkill(w, r, defaultStatusResolver)
}

func handleAddSkill(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：不走 TAT/CVM，走 “创建 pending records”路线，由 reporter sync 取走。
	// 不走 checkInstanceSupportsSkill：本地 agent 类型（workbuddy / codebuddy）当前未注册
	// 到内置 agentTypesMap，该校验会一律拒绝。本地实例的 skill 能力由 reporter
	// ack 链路保证，不依赖该能力位；二期正式注册本地 agent 类型后可移除本说明。
	if instance.Source == model.InstanceSourceLocal {
		handleAddSkillLocal(w, r, user, instance)
		return
	}

	// 【关键防护】校验实例是否支持技能安装
	if err := checkInstanceSupportsSkill(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许安装技能
	if _, rerr := requireInstanceRunning(r.Context(), instance, resolver); rerr != nil {
		writeAgentGuardError(w, r, rerr)
		return
	}

	skillName := strings.TrimSpace(r.FormValue("skill_name"))
	agentID := strings.TrimSpace(r.FormValue("agent_id"))
	if agentID != "" {
		if model.GetAgentRuntimeType(r.Context(), instance.AgentType) != model.AgentTypeOpenClaw {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "agent_id", "仅支持 OpenClaw 实例"))
			return
		}
		if !isSafeOpenClawAgentID(agentID) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "agent_id"))
			return
		}
		agentID = strings.ToLower(agentID)
	}
	source := strings.ToLower(strings.TrimSpace(r.FormValue("source")))
	if source == "" {
		source = "public"
	}

	switch source {
	case "public":
		if applyErr := applyPublicSkill(r.Context(), instance, skillName, agentID); applyErr != nil {
			if applyErr.run {
				writeAddSkillRunError(w, r, user, instance, skillName, applyErr.err)
			} else {
				writeError(w, r, applyErr.status, hcommon.EnsureRichErrorOrPanic(applyErr.err))
			}
			return
		}

	case "enterprise":
		skill, status, err := resolveVisibleEnterpriseSkill(r.Context(), user.ID, skillName, r.FormValue("version"))
		if err != nil {
			writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if applyErr := applyEnterpriseSkill(r.Context(), instance, skill, agentID); applyErr != nil {
			if applyErr.run {
				writeAddSkillRunError(w, r, user, instance, skillName, applyErr.err)
			} else {
				writeError(w, r, applyErr.status, hcommon.EnsureRichErrorOrPanic(applyErr.err))
			}
			return
		}

	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "source", "仅支持 public 或 enterprise"))
		return
	}

	jsonOK(w, i18n.T(r.Context(), i18n.MsgSkillInstallSuccess))
}

type enterpriseSkillApplyError struct {
	status int
	err    error
	run    bool
}

var skillScriptRunner = RunScript

// applyPublicSkill performs the same one-shot public skill install used by the
// user add-skill endpoint. An empty source selects this path.
func applyPublicSkill(ctx context.Context, instance *model.Instance, skillName, agentID string) *enterpriseSkillApplyError {
	if skillName == "" {
		return &enterpriseSkillApplyError{
			status: http.StatusBadRequest,
			err:    hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skill_name"),
		}
	}
	params := map[string]string{"skill_name": skillName, "agent_id": agentID}
	runtimeType := model.GetAgentRuntimeType(ctx, instance.AgentType)
	if runtimeType == model.AgentTypeHermes || runtimeType == model.AgentTypeLightclawACE {
		params["skillhub_registry"] = model.GetSiteConfig(ctx).SkillHub
	}
	scriptName, err := ResolveScript(ctx, "add_skill", instance.AgentType)
	if err != nil {
		return &enterpriseSkillApplyError{
			status: http.StatusBadRequest,
			err:    hcommon.I18nRichError(err, i18n.MsgParseAddSkillScriptFailed),
		}
	}
	if _, err := skillScriptRunner(ctx, instance.InstanceId, scriptName, 120, instance.RuntimeUser, nil, params); err != nil {
		return &enterpriseSkillApplyError{
			status: http.StatusInternalServerError,
			err:    err,
			run:    true,
		}
	}
	return nil
}

// applyEnterpriseSkill performs the same one-shot enterprise skill install
// used by the user add-skill endpoint. The download URL and request stay in
// memory and are never materialized as an installation task.
func applyEnterpriseSkill(ctx context.Context, instance *model.Instance, skill *model.Skill, agentID string) *enterpriseSkillApplyError {
	downloadURL, err := buildSMHDownloadURL(ctx, skill.COSZipKey, true)
	if err != nil {
		return &enterpriseSkillApplyError{
			status: http.StatusInternalServerError,
			err:    hcommon.I18nRichError(err, i18n.MsgSkillStoreGenDownloadURLFail),
		}
	}
	scriptName, err := ResolveScript(ctx, "install_skill_from_smh", instance.AgentType)
	if err != nil {
		return &enterpriseSkillApplyError{
			status: http.StatusBadRequest,
			err:    hcommon.I18nRichError(err, i18n.MsgParseAddSkillScriptFailed),
		}
	}
	params := map[string]string{
		"download_url":  downloadURL,
		"skill_slug":    skill.Slug,
		"skill_version": skill.Version,
		"agent_id":      agentID,
	}
	if _, err := skillScriptRunner(ctx, instance.InstanceId, scriptName, 120, instance.RuntimeUser, nil, params); err != nil {
		return &enterpriseSkillApplyError{
			status: http.StatusInternalServerError,
			err:    err,
			run:    true,
		}
	}
	return nil
}

// applyExtraSkillsAsync applies administrator-selected public or enterprise
// skills once after the Agent is ready. It intentionally creates no
// SkillInstallation rows and does not retry after a failure or process restart.
func applyExtraSkillsAsync(ctx context.Context, instanceID uint, skills []createSkillPreset) {
	instance, err := waitForCreatePresetInstance(ctx, instanceID)
	if err != nil {
		slog.Warn("[CreateSkillPreset] 等待实例就绪失败", "instance_id", instanceID)
		return
	}
	for index := range skills {
		var applyErr *enterpriseSkillApplyError
		if skills[index].Source == model.SkillSourceEnterprise {
			applyErr = applyEnterpriseSkill(ctx, instance, &skills[index].Enterprise, "")
		} else {
			applyErr = applyPublicSkill(ctx, instance, skills[index].Slug, "")
		}
		if applyErr != nil {
			slog.Warn("[CreateSkillPreset] 技能下发失败",
				"instance_id", instanceID,
				"source", skills[index].Source,
				"skill", skills[index].Slug,
			)
			continue
		}
		slog.Info("[CreateSkillPreset] 技能下发成功",
			"instance_id", instanceID,
			"source", skills[index].Source,
			"skill", skills[index].Slug,
		)
	}
}

// handleAddSkillLocal 本地实例的 add-skill 路径。
//
// 与 CVM 不同：不走 TAT/CVM，改为在 hatchery 侧创建 pending records，由 reporter sync 取走。
//   - 实例未接入（LastReportAt 超过阈值）→ 400
//   - 不查 skills 表，统一走 ClawHub 兜底：skill_id=0、slug=skillName、version=""，
//     由 reporter 拿到后自行去 ClawHub 拉对应 slug 的最新版本安装。
//   - 返回表示已下发，reporter 上线后拉取。
func handleAddSkillLocal(w http.ResponseWriter, r *http.Request, user *model.User, instance *model.Instance) {
	ctx := r.Context()

	// 1. 校验本地实例是否接入（阈值同 LocalInstanceOfflineThreshold）
	var info model.LocalInstanceInfo
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).First(&info).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceNotConnected))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if info.LastReportAt == nil || time.Since(*info.LastReportAt) > model.LocalInstanceOfflineThreshold {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceNotConnected))
		return
	}

	// 2. 解析参数（前端只传 skill_name，本地实例统一走 ClawHub 兜底）
	skillName := strings.TrimSpace(r.FormValue("skill_name"))
	if skillName == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skill_name"))
		return
	}

	// 3. 不查 skills 表，统一兜底：skill_id=0、slug=skillName、version=""。
	skillID := uint(0)
	skillSlug := skillName
	skillVersion := "" // ClawHub 默认拉最新

	// 4. 幂等保护：同一 (instance_id, slug) 如果已有 pending distribute，则复用。
	//   本地路径恒定 skill_id=0，按 task.slug 去重（JOIN tasks）。
	var existing model.SkillDistributionRecord
	dedupQuery := model.DB(ctx).
		Model(&model.SkillDistributionRecord{}).
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.status = ? AND skill_distribution_records.type = ?",
			instance.ID, "pending", "distribute").
		Where("skill_distribution_records.skill_id = 0 AND skill_distribution_tasks.slug = ?", skillSlug)
	if err := dedupQuery.First(&existing).Error; err == nil {
		jsonOK(w, map[string]any{
			"record_id":    existing.ID,
			"task_id":      existing.TaskID,
			"slug":         skillSlug,
			"version":      existing.Version,
			"status":       "pending",
			"deduplicated": true,
			"message":      i18n.T(ctx, i18n.MsgSkillInstallDispatched),
		})
		return
	}

	// 5. 创建 task + record（事务内原子）
	var (
		task   model.SkillDistributionTask
		record model.SkillDistributionRecord
	)
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		task = model.SkillDistributionTask{
			SkillID:    skillID,
			Version:    skillVersion,
			OperatorID: user.ID,
			Total:      1,
			Status:     "running",
			Type:       "distribute",
			Slug:       skillSlug,
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		record = model.SkillDistributionRecord{
			TaskID:      task.ID,
			SkillID:     skillID,
			InstanceID:  instance.ID,
			InstanceCID: instance.InstanceId,
			Version:     skillVersion,
			Status:      "pending",
			Type:        "distribute",
		}
		return tx.Create(&record).Error
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "[AddSkill][Local] 创建 pending record 失败",
			"user_id", user.ID, "instance_id", instance.ID, "slug", skillSlug, "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"record_id": record.ID,
		"task_id":   task.ID,
		"slug":      skillSlug,
		"version":   skillVersion,
		"status":    "pending",
		"message":   i18n.T(ctx, i18n.MsgSkillInstallDispatched),
	})
}

func isSafeOpenClawAgentID(agentID string) bool {
	if agentID == "" || len(agentID) > 64 {
		return false
	}
	for i, r := range agentID {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if i == 0 {
			if !isAlpha && !isDigit {
				return false
			}
			continue
		}
		if isAlpha || isDigit || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func resolveVisibleEnterpriseSkill(ctx context.Context, userID uint, skillName, version string) (*model.Skill, int, error) {
	skillName = strings.TrimSpace(skillName)
	version = strings.TrimSpace(version)
	if skillName == "" {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skill_name")
	}

	var skill model.Skill
	query := model.DB(ctx).Where("slug = ?", skillName)
	if version != "" {
		query = query.Where("version = ?", version)
	} else {
		query = query.Order("version_major DESC, version_minor DESC, version_patch DESC")
	}
	err := query.First(&skill).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist)
		}
		return nil, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillNotExist)
	}
	if skill.COSZipKey == "" {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "skill_name", "企业 Skill 缺少安装包")
	}
	visible, err := model.IsSkillVisibleToUser(ctx, &skill, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreVisibilityCheckFail)
	}
	if !visible {
		return nil, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist)
	}
	return &skill, 0, nil
}

func writeAddSkillRunError(w http.ResponseWriter, r *http.Request, user *model.User, instance *model.Instance, skillName string, err error) {
	// 将原生错误信息映射为友好的中文提示（RunScript 返回的都是 hcommon.RichError）
	errMsg := hcommon.ErrorDetailWithCtx(r.Context(), err)
	if errMsg == "" {
		errMsg = err.Error()
	}
	// 异步写入通知（构造包含 InstanceId 和 RequestId 的 hcommon.RichError）
	// 允许异步 goroutine 中使用请求者语言：在 detached ctx 上复制 Printer。
	notifyCtx := hcommon.DetachContext(r.Context())
	namedTitle := i18n.T(notifyCtx, i18n.MsgSkillNamedInstallFailed, skillName)
	rerr := hcommon.I18nRichError(err, i18n.MsgSkillNamedInstallFailed, skillName).
		WithBizRequestId(hcommon.GetRequestID(r.Context())).
		WithInstanceId(instance.InstanceId)
	go createErrorNotification(user.ID, instance.ID, instance.Name,
		model.NotifyTypeSkillInstallFailed, namedTitle,
		rerr, notifyCtx)
	switch {
	case strings.Contains(errMsg, "Skill not found"):
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillNotFoundCheckName).WithDetail(errMsg))
	case strings.Contains(errMsg, "Rate limit exceeded"):
		writeError(w, r, http.StatusTooManyRequests, hcommon.I18nError(i18n.MsgRateLimitExceeded).WithDetail(errMsg))
	default:
		// 已知业务错误都识别不上 → 诊断是否因安全组出站被拒
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(maybeWrapEgressBlocked(r.Context(), instance.InstanceId, err)))
	}
}

func handleSkillsList(w http.ResponseWriter, r *http.Request, deps userSkillDependencies) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	output, err := listInstanceSkills(r.Context(), instance.InstanceId, instance.RuntimeUser, instance.AgentType)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	items, err := enrichDistributedSkillVersions(r.Context(), user.ID, instance.ID, output, deps)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	}
	jsonOK(w, items)
}

// ── 技能安装管理接口 ───────────────────────────────────────────────

// HandleInstallSkills 查询技能安装状态
// GET /openclaw/install-skills?id=xxx
//
// CVM 实例：从 skill_installations 表查（保持原义）。
// local 实例：合并 local_instance_skills（success）+ skill_distribution_records.status='pending'/'failed'
//
//	合并规则（iwiki §5.B.3）：
//	  - records.status='pending'  → install_status='installing'（即使有旧版本 lis）
//	  - records.status='failed' 且 lis 无同 slug → install_status='failed'
//	  - records.status='failed' 但 lis 有同 slug → install_status='success'（旧版本仍装着）
//	  - 仅 lis → install_status='success'
func HandleInstallSkills(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// local 实例走独立路径
	if instance.Source == model.InstanceSourceLocal {
		writeLocalInstallSkillsResponse(w, r, instance)
		return
	}

	var skills []model.SkillInstallation
	if err := model.DB(r.Context()).Where("instance_id = ?", instance.ID).Find(&skills).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
		return
	}
	if skills == nil {
		skills = []model.SkillInstallation{}
	}

	jsonOK(w, map[string]interface{}{
		"instance_id": instance.ID,
		"skills":      skills,
		"total":       len(skills),
	})
}

// localInstallSkillItem 本地实例 install-skills 响应单项。
type localInstallSkillItem struct {
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	Version       string  `json:"version"`
	InstallStatus string  `json:"install_status"` // installing / success / failed
	ErrorMessage  string  `json:"error_message"`
	Source        string  `json:"source"`
	InstalledAt   *string `json:"installed_at"`
}

// writeLocalInstallSkillsResponse 本地实例版 install-skills 走 lis + records 合并。
func writeLocalInstallSkillsResponse(w http.ResponseWriter, r *http.Request, instance *model.Instance) {
	ctx := r.Context()

	// 1. 查 lis（success）
	var lisRows []model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).Find(&lisRows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
		return
	}

	// 2. 查 records.status ∈ {pending, failed} 的 distribute 记录，JOIN skills 拿 slug/name/skill_id
	type recordRow struct {
		Status    string
		Type      string
		Version   string
		Error     string
		CreatedAt time.Time
		Slug      string
		Name      string
		SkillID   uint
	}
	var recRows []recordRow
	if err := model.DB(ctx).
		Model(&model.SkillDistributionRecord{}).
		Select(`skill_distribution_records.status,
		        skill_distribution_records.type,
		        skill_distribution_records.version,
		        skill_distribution_records.error,
		        skill_distribution_records.created_at,
		        skills.slug,
		        skills.name,
		        skills.id AS skill_id`).
		Joins("JOIN skills ON skills.id = skill_distribution_records.skill_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.type = ? AND skill_distribution_records.status IN ?",
			instance.ID, "distribute", []string{"pending", "failed"}).
		Order("skill_distribution_records.id DESC").
		Scan(&recRows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
		return
	}

	// 3. 合并
	lisBySlug := make(map[string]model.LocalInstanceSkill, len(lisRows))
	for _, row := range lisRows {
		lisBySlug[row.Slug] = row
	}
	seenSlug := map[string]bool{}
	items := make([]localInstallSkillItem, 0, len(lisRows)+len(recRows))

	// 3.1 先取最新 record（同 slug 只取一条，ORDER BY id DESC）
	for _, rr := range recRows {
		if seenSlug[rr.Slug] {
			continue
		}
		seenSlug[rr.Slug] = true

		lis, hasLis := lisBySlug[rr.Slug]
		item := localInstallSkillItem{
			Slug:    rr.Slug,
			Name:    pickNonEmpty(lis.DisplayName, rr.Name),
			Version: pickNonEmpty(rr.Version, lis.Version),
			Source:  inferLocalSkillSourceFromRow(lis, rr.SkillID),
		}
		switch rr.Status {
		case "pending":
			item.InstallStatus = "installing"
			item.InstalledAt = formatLocalSkillTime(lis.InstalledAt)
		case "failed":
			if hasLis {
				item.InstallStatus = "success"
				item.ErrorMessage = ""
				item.Version = lis.Version
				item.InstalledAt = formatLocalSkillTime(lis.InstalledAt)
			} else {
				item.InstallStatus = "failed"
				item.ErrorMessage = rr.Error
			}
		}
		items = append(items, item)
	}

	// 3.2 补上 lis 中未被 record 覆盖的（纯 success）
	for _, lis := range lisRows {
		if seenSlug[lis.Slug] {
			continue
		}
		items = append(items, localInstallSkillItem{
			Slug:          lis.Slug,
			Name:          lis.DisplayName,
			Version:       lis.Version,
			InstallStatus: "success",
			Source:        defaultIfEmpty(lis.Source, model.LocalSkillSourceLocal),
			InstalledAt:   formatLocalSkillTime(lis.InstalledAt),
		})
	}

	jsonOK(w, map[string]any{
		"instance_id": instance.ID,
		"skills":      items,
		"total":       len(items),
	})
}

func pickNonEmpty(candidates ...string) string {
	for _, c := range candidates {
		if s := strings.TrimSpace(c); s != "" {
			return s
		}
	}
	return ""
}

func defaultIfEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// inferLocalSkillSourceFromRow 推断 pending/failed 状态下某条 record 对应
// 的 LocalInstanceSkill.Source。
//
//   - 优先用 lis.Source（ack 写入的权威值）；ack 路径已以 skill.ID==0 判断。
//   - lis 未入表（ack 未到）时则面向 record.SkillID 判断：
//   - SkillID == 0 → ClawHub 兜底 → public
//   - 其余 → 企业内部 → enterprise
func inferLocalSkillSourceFromRow(lis model.LocalInstanceSkill, recordSkillID uint) string {
	if strings.TrimSpace(lis.Source) != "" {
		return lis.Source
	}
	if recordSkillID == 0 {
		return model.LocalSkillSourcePublic
	}
	return model.LocalSkillSourceEnterprise
}

func formatLocalSkillTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// HandleLocalPendingSkillsRouter 在同一路由上按方法分流。
// GET    → HandleListLocalPendingSkills
// DELETE → HandleDeleteLocalPendingSkill
func HandleLocalPendingSkillsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		HandleListLocalPendingSkills(w, r)
	case http.MethodDelete:
		HandleDeleteLocalPendingSkill(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// HandleListLocalPendingSkills 查询本地实例上未处理的 skill 下发记录（安装中 / 安装失败）。
// GET /openclaw/local/pending-skills?id=<instance_id>
//
// 展示语义：对同一 slug，只展示"最近一次 success 之后"新发生的 pending/failed record。
//   - 从未成功过的 slug：展示全部 pending/failed（首装失败/安装中）
//   - 有成功过的 slug：只展示 last-success 之后的 pending/failed（成功前的全部历史失败忽略）
//   - 同 slug 内多条均展示、不去重——前端删一条数量即减 1，UX 自洽
func HandleListLocalPendingSkills(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, ierr := getInstanceByID(&w, r, user)
	if ierr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(ierr))
		return
	}
	if instance.Source != model.InstanceSourceLocal {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalOnlyEndpoint))
		return
	}

	type recordRow struct {
		RecordID  uint
		Status    string
		Type      string
		Version   string
		Error     string
		CreatedAt time.Time
		TaskID    uint
		Slug      string
		Name      string
	}

	// 语义：对同一 slug，只展示「最近一次 success 之后」新增的 pending/failed 记录。
	//   - 一条都没成功过的 slug：展示全部 pending/failed（首装失败/首装中都算未处理）
	//   - 有成功过的 slug：只展示 last-success 之后的 pending/failed
	//     （比如"装好后又重装失败" / "装好后手动重装中"）
	//
	// 不再按 slug 去重——同 slug 内多条 pending/failed 各占一行，DELETE 一条数量真的减 1。
	//
	// 实现：一条 SQL 查所有 distribute 的 pending / failed / success 记录 → 应用层 groupBy slug
	// → 找每个 slug 最近一条 success 的 id → 过滤保留 id > lastSuccessID 的 pending/failed。
	// 之所以把 success 也拉回来：DB 端做 correlated subquery / window function 兼容 SQLite 麻烦，
	// 数据量按实例 × slug 计一般在几百内，应用层筛更清晰。
	var rows []recordRow
	if err := model.DB(r.Context()).
		Model(&model.SkillDistributionRecord{}).
		Select(`skill_distribution_records.id AS record_id,
		        skill_distribution_records.status AS status,
		        skill_distribution_records.type AS type,
		        skill_distribution_records.version AS version,
		        skill_distribution_records.error AS error,
		        skill_distribution_records.created_at AS created_at,
		        skill_distribution_records.task_id AS task_id,
		        COALESCE(NULLIF(skills.slug, ''), skill_distribution_tasks.slug) AS slug,
		        COALESCE(NULLIF(skills.name, ''), skill_distribution_tasks.slug) AS name`).
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Joins("LEFT JOIN skills ON skills.id = skill_distribution_records.skill_id AND skill_distribution_records.skill_id <> 0").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.type = ? AND skill_distribution_records.status IN ?",
			instance.ID, "distribute", []string{"pending", "failed", "success"}).
		Order("skill_distribution_records.id DESC").
		Scan(&rows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
		return
	}

	// 第一趟：找出每个 slug 最近一条 success 的 record_id（按 id DESC 排，首个 success 即为 last）
	lastSuccessBySlug := map[string]uint{}
	for _, rr := range rows {
		if rr.Status != "success" {
			continue
		}
		if _, ok := lastSuccessBySlug[rr.Slug]; !ok {
			lastSuccessBySlug[rr.Slug] = rr.RecordID
		}
	}

	// 第二趟：只保留 pending/failed 且 record_id > lastSuccess（若从未成功过则全保留）
	items := make([]map[string]any, 0, len(rows))
	for _, rr := range rows {
		if rr.Status != "pending" && rr.Status != "failed" {
			continue
		}
		if lastID, ok := lastSuccessBySlug[rr.Slug]; ok && rr.RecordID <= lastID {
			continue
		}
		status := "installing"
		if rr.Status == "failed" {
			status = "failed"
		}
		items = append(items, map[string]any{
			"record_id":      rr.RecordID,
			"slug":           rr.Slug,
			"name":           rr.Name,
			"version":        rr.Version,
			"install_status": status,
			"error_message":  rr.Error,
			"task_id":        rr.TaskID,
			"created_at":     rr.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	jsonOK(w, map[string]any{
		"instance_id": instance.ID,
		"skills":      items,
		"total":       len(items),
	})
}

// HandleDeleteLocalPendingSkill 删除本地实例上「安装中 / 安装失败」的 skill 下发任务。
// HandleDeleteLocalPendingSkill 删除本地实例上「安装中 / 安装失败」的单条 skill 下发记录。
// DELETE /openclaw/local/pending-skills?id=<instance_id>&record_id=<record_id>
// 兼容 POST + ?_method=DELETE。
//
// 参数语义：
//   - id：本地实例主键（instances.id），也用于鉴权与防跨实例越权
//   - record_id：skill_distribution_records.id，直接从 GET 列表接口返回的 record_id 字段取
//
// 重设设计思路：旧版参数 skill_slug 存在两个问题：
//  1. slug 有两条来源（skills.slug / task.slug），需双源匹配，容易遗漏
//  2. 同一 slug 可能有多条历史 record，“删失败记录”变成“删 slug 全部历史”，语义模糊
//
// 直接按 record_id 删：一次 WHERE id=? AND instance_id=? 就搞定，语义精确。
func HandleDeleteLocalPendingSkill(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	override := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("_method")))
	switch r.Method {
	case http.MethodDelete:
	case http.MethodPost:
		if override != "DELETE" {
			writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
			return
		}
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, ierr := getInstanceByID(&w, r, user)
	if ierr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(ierr))
		return
	}
	if instance.Source != model.InstanceSourceLocal {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalOnlyEndpoint))
		return
	}

	recordIDStr := strings.TrimSpace(r.URL.Query().Get("record_id"))
	if recordIDStr == "" {
		recordIDStr = strings.TrimSpace(r.FormValue("record_id"))
	}
	if recordIDStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "record_id"))
		return
	}
	recordID, err := strconv.ParseUint(recordIDStr, 10, 64)
	if err != nil || recordID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "record_id"))
		return
	}

	// 按 record_id 精确删：
	//   - id 匹配目标 record
	//   - instance_id 作为二重守卫，防止拿到 A 实例的 record_id 去删 B 实例的记录
	//   - type/status 限定：仅允许删 pending/failed 的 distribute 记录，避免误删 success 或 uninstall
	res := model.DB(r.Context()).
		Where("id = ? AND instance_id = ? AND type = ? AND status IN ?",
			recordID, instance.ID, "distribute", []string{"pending", "failed"}).
		Delete(&model.SkillDistributionRecord{})
	if res.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(res.Error, i18n.MsgInternalError))
		return
	}

	// 幂等：命中 0 行返 200 + deleted=0（前端重试 / 并发删除时不报错）
	jsonOK(w, map[string]any{
		"ok":        true,
		"deleted":   res.RowsAffected,
		"record_id": recordID,
	})
}

// HandleRetryFailedSkills 重试安装失败的技能
// POST /openclaw/retry-failed-skills?id=xxx
func HandleRetryFailedSkills(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
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

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 查询失败的安装记录，重置为 None
	result := model.DB(r.Context()).Model(&model.SkillInstallation{}).
		Where("instance_id = ? AND install_status = ?", instance.ID, model.SkillInstallFailed).
		Updates(map[string]interface{}{
			"install_status": model.SkillInstallNone,
			"error_message":  "",
		})

	retryCount := result.RowsAffected
	if retryCount == 0 {
		jsonOK(w, map[string]interface{}{"ok": true, "retry_count": 0})
		return
	}

	// 异步重新安装（重试场景，CVM 已在运行中，无需等待）
	go installSkillsAsync(hcommon.DetachContext(r.Context()), instance.ID, instance.InstanceId, instance.AgentType, waitModeRetry)

	jsonOK(w, map[string]interface{}{"ok": true, "retry_count": retryCount})
}

// HandleCancelFailedSkills 取消安装失败的技能
// POST /openclaw/cancel-failed-skills?id=xxx
func HandleCancelFailedSkills(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
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

	result := model.DB(r.Context()).Model(&model.SkillInstallation{}).
		Where("instance_id = ? AND install_status = ?", instance.ID, model.SkillInstallFailed).
		Update("install_status", model.SkillInstallCancelled)

	jsonOK(w, map[string]interface{}{"ok": true, "cancel_count": result.RowsAffected})
}

// ── 安装核心逻辑 ───────────────────────────────────────────────────

// createSkillInstallTasks 为实例创建技能安装任务记录。
// 查询所有已启用且对该用户可见的技能包中的技能，快照写入 skill_installations 表。
// 同一 slug 的技能按版本号取最新版本（避免多包含同 slug 不同版本时的冲突）。
func createSkillInstallTasks(ctx context.Context, instanceID uint, roleID uint, groupID uint) {
	// 用于去重的 slug 集合（记录版本分数，高版本覆盖低版本）
	type seenEntry struct {
		Index        int
		VersionScore int
	}
	seen := make(map[string]seenEntry)

	// 统一的技能条目（slug + name + version + cosZipKey）
	type skillEntry struct {
		Name      string
		Slug      string
		Version   string
		CosZipKey string
	}
	var allSkills []skillEntry

	// 添加技能到列表（含版本去重逻辑）
	addSkill := func(name, slug, version, cosZipKey string) {
		newScore := model.VersionScore(version)
		if existing, ok := seen[slug]; ok {
			// 已有同 slug 的技能，比较版本号，高版本覆盖低版本
			if newScore > existing.VersionScore {
				allSkills[existing.Index] = skillEntry{
					Name: name, Slug: slug, Version: version, CosZipKey: cosZipKey,
				}
				seen[slug] = seenEntry{existing.Index, newScore}
			}
		} else {
			seen[slug] = seenEntry{len(allSkills), newScore}
			allSkills = append(allSkills, skillEntry{
				Name: name, Slug: slug, Version: version, CosZipKey: cosZipKey,
			})
		}
	}

	// ① 角色技能（优先级高，先加入）
	if roleID > 0 {
		var roleSkills []model.OpenClawRoleSkill
		model.DB(ctx).Where("open_claw_role_id = ?", roleID).Find(&roleSkills)
		for _, rs := range roleSkills {
			addSkill(rs.Name, rs.Slug, rs.Version, rs.CosZipKey)
		}
	}

	// ② 全局 + 分组技能包技能（按 agent 绑定的分组可见性过滤）
	var bundles []model.SkillBundle
	model.DB(ctx).Where("enabled = ?", true).Find(&bundles)
	if len(bundles) > 0 {
		// 查询分组（含祖先链，加法型 Union 语义）
		// groupID == 0 时 userGroupSet 为空，仅 visibility_type='all' 的技能包可见
		var bundleGroupIDs []uint
		if groupID > 0 {
			bundleGroupIDs, _ = usergroup.GetAncestorIDs(ctx, groupID)
		}
		userGroupSet := make(map[uint]bool)
		for _, gid := range bundleGroupIDs {
			userGroupSet[gid] = true
		}

		// 批量查询 group 类型技能包的分组关联
		var groupBundleIDs []uint
		for _, b := range bundles {
			if b.VisibilityType == usergroup.VisibilityGroup {
				groupBundleIDs = append(groupBundleIDs, b.ID)
			}
		}
		bundleGroupMap := make(map[uint][]uint)
		if len(groupBundleIDs) > 0 {
			bundleGroupMap, _ = model.GetSkillBundleVisibilityGroupIDs(ctx, groupBundleIDs)
		}

		// 过滤可见的技能包 ID
		var visibleBundleIDs []uint
		for _, b := range bundles {
			if b.VisibilityType == usergroup.VisibilityAll {
				// all 类型直接通过
				visibleBundleIDs = append(visibleBundleIDs, b.ID)
				continue
			}
			if b.VisibilityType == usergroup.VisibilityGroup {
				// group 类型：检查用户分组与技能包分组是否有交集
				for _, gid := range bundleGroupMap[b.ID] {
					if userGroupSet[gid] {
						visibleBundleIDs = append(visibleBundleIDs, b.ID)
						break
					}
				}
			}
		}

		if len(visibleBundleIDs) > 0 {
			var bundleSkills []model.BundleSkill
			model.DB(ctx).Where("skill_bundle_id IN ?", visibleBundleIDs).Find(&bundleSkills)
			for _, bs := range bundleSkills {
				addSkill(bs.Name, bs.Slug, bs.Version, bs.CosZipKey)
			}
		}
	}

	if len(allSkills) == 0 {
		return
	}

	for _, s := range allSkills {
		installation := model.SkillInstallation{
			InstanceID:    instanceID,
			Name:          s.Name,
			Slug:          s.Slug,
			Version:       s.Version,
			CosZipKey:     s.CosZipKey,
			InstallStatus: model.SkillInstallNone,
		}
		if err := model.DB(ctx).Create(&installation).Error; err != nil {
			slog.Error("[SkillInstall] 创建安装记录失败", "instance_id", instanceID, "slug", s.Slug, "error", err)
		}
	}

	slog.Info("[SkillInstall] 技能安装任务已创建", "instance_id", instanceID, "skill_count", len(allSkills), "role_id", roleID)
}

// CVM 等待模式：控制 installSkillsAsync 在安装技能前如何等待 CVM 就绪。
const (
	// waitModeCreate 创建实例：等待 CVM 变为 RUNNING + 30s TAT Agent 缓冲
	waitModeCreate = iota
	// waitModeReinstall 重装实例：先等非 RUNNING（确认重装开始）→ 再等 RUNNING → 30s 缓冲
	waitModeReinstall
	// waitModeRetry 重试失败技能：CVM 已在运行，跳过所有等待
	waitModeRetry
)

// installSkillsAsync 异步安装技能到 CVM 实例。
// waitMode 控制等待策略：
//   - waitModeCreate:    等 CVM RUNNING + 30s TAT Agent 缓冲（新建实例）
//   - waitModeReinstall: 先等非 RUNNING → 再等 RUNNING + 30s 缓冲（重装实例）
//   - waitModeRetry:     跳过等待，直接安装（重试失败技能）
//
// v7 新增 agentType 形参：用于按 agent_type 分派 batch_install_skills 脚本。
// 设计选择（见 docs/multi-agent-compat-requirements.md 6.3.5）：方案 A 调用者透传，
// 避免函数内 DB 二次查询及其与调用者的时序不一致风险。
func installSkillsAsync(ctx context.Context, instanceID uint, cvmInstanceId, agentType string, waitMode int) {
	logger := slog.With("task", "installSkillsAsync",
		"instance_id", instanceID,
		"cvm_instance_id", cvmInstanceId,
		"agent_type", agentType)

	// 技能安装结束后触发角色同步状态聚合（无论成功/失败），及时收敛 record + instance.role_sync_status。
	// 若实例没有 updating record（非角色切换场景），refreshRoleRecord 内部会 no-op。
	defer refreshRoleRecord(ctx, instanceID)

	// 查询实例信息，供失败通知使用
	var inst model.Instance
	if err := model.DB(ctx).Select("user_id, name").First(&inst, instanceID).Error; err != nil {
		logger.Error("查询实例信息失败，无法发送通知", "error", err)
	}

	// 查询待安装的技能（status = None）
	var skills []model.SkillInstallation
	model.DB(ctx).Where("instance_id = ? AND install_status = ?", instanceID, model.SkillInstallNone).Find(&skills)
	if len(skills) == 0 {
		logger.Info("无待安装技能")
		return
	}

	// ── 阶段1：等待 CVM 就绪 ──

	if waitMode == waitModeReinstall {
		// 重装场景：分两步等待，确保等到的是重装后的 RUNNING，而不是重装前残留的 RUNNING。
		//   1a. 先等待 CVM 进入非 RUNNING 状态（确认重装已开始）
		//   1b. 再等待 CVM 重新变为 RUNNING（确认重装已完成）

		logger.Info("重装场景：等待 CVM 进入重装状态", "skill_count", len(skills))

		// ── 阶段1a：等待 CVM 变成非 RUNNING（最多等 2 分钟） ──
		reinstallStarted := false
		for attempt := 1; attempt <= 24; attempt++ {
			state, err := fetchCVMState(ctx, cvmInstanceId)
			if err != nil {
				logger.Warn("查询 CVM 状态失败", "phase", "wait_non_running", "attempt", attempt, "error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			if state != "RUNNING" {
				reinstallStarted = true
				logger.Info("CVM 已进入重装状态", "attempt", attempt, "state", state)
				break
			}
			logger.Info("CVM 仍在运行中，等待进入重装状态", "attempt", attempt, "state", state)
			time.Sleep(5 * time.Second)
		}

		if !reinstallStarted {
			// 如果 2 分钟内 CVM 一直保持 RUNNING，可能说明 ResetInstance 未生效或已经完成了。
			// 这种情况属于边界场景，记录警告后继续执行（按已就绪处理）。
			logger.Warn("等待 CVM 进入非 RUNNING 状态超时，CVM 可能已就绪或重装未生效，继续执行")
		}
	}

	if waitMode != waitModeRetry {
		// 创建/重装场景：等待 CVM 变为 RUNNING（最多等 10 分钟）
		logger.Info("等待 CVM 就绪", "skill_count", len(skills))
		cvmReady := false
		for attempt := 1; attempt <= 60; attempt++ {
			state, err := fetchCVMState(ctx, cvmInstanceId)
			if err != nil {
				logger.Warn("查询 CVM 状态失败", "phase", "wait_running", "attempt", attempt, "error", err)
				time.Sleep(10 * time.Second)
				continue
			}
			if state == "RUNNING" {
				cvmReady = true
				logger.Info("CVM 已就绪", "attempt", attempt)
				break
			}
			logger.Info("CVM 尚未就绪", "attempt", attempt, "state", state)
			time.Sleep(10 * time.Second)
		}

		if !cvmReady {
			logger.Error("等待 CVM 就绪超时，标记所有技能为失败")
			model.DB(ctx).Model(&model.SkillInstallation{}).
				Where("instance_id = ? AND install_status = ?", instanceID, model.SkillInstallNone).
				Updates(map[string]interface{}{
					"install_status": model.SkillInstallFailed,
					"error_message":  "等待 CVM 就绪超时",
				})
			if inst.UserID > 0 {
				notityCtx := hcommon.DetachContext(ctx)
				go createErrorNotification(inst.UserID, instanceID, inst.Name,
					model.NotifyTypeSkillInstallFailed, i18n.T(notityCtx, i18n.MsgSkillInstallFailedTitle),
					hcommon.I18nError(i18n.MsgWaitCVMTimeout), notityCtx)
			}
			return
		}

	}

	// ── 阶段2：构建 TAT 参数并执行 ──
	// 标记为 Installing
	model.DB(ctx).Model(&model.SkillInstallation{}).
		Where("instance_id = ? AND install_status = ?", instanceID, model.SkillInstallNone).
		Update("install_status", model.SkillInstalling)

	// 重新查询（包含可能被跳过的）
	model.DB(ctx).Where("instance_id = ? AND install_status = ?", instanceID, model.SkillInstalling).Find(&skills)

	// 构建 skills_list（TAB 分隔多行文本）
	var skillsListLines []string
	var validSkills []model.SkillInstallation

	for _, skill := range skills {
		if skill.CosZipKey == "" {
			// 快照中 cos_zip_key 为空（角色技能无此字段 / 创建时 SMH 尚未同步完成），
			// 实时从 bundle_skills 表按 slug 回查最新值
			var latestBundleSkill model.BundleSkill
			if err := model.DB(ctx).Where("slug = ? AND cos_zip_key != ''", skill.Slug).
				Order("updated_at DESC").First(&latestBundleSkill).Error; err == nil {
				skill.CosZipKey = latestBundleSkill.CosZipKey
				model.DB(ctx).Model(&skill).Update("cos_zip_key", latestBundleSkill.CosZipKey)
				logger.Info("从 bundle_skills 回查到 cos_zip_key", "slug", skill.Slug, "cos_zip_key", latestBundleSkill.CosZipKey)
			} else {
				logger.Warn("技能 cos_zip_key 为空且 bundle_skills 中也未找到，跳过安装", "slug", skill.Slug)
				model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
					"install_status": model.SkillInstallFailed,
					"error_message":  i18n.T(ctx, i18n.MsgSkillSMHSyncPending, skill.Slug),
				})
				continue
			}
		}

		downloadURL, err := BuildCommonSMHDownloadURL(ctx, skill.CosZipKey, true)
		if err != nil {
			logger.Error("生成下载 URL 失败", "slug", skill.Slug, "error", err)
			model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
				"install_status": model.SkillInstallFailed,
				"error_message":  "生成下载 URL 失败: " + err.Error(),
			})
			continue
		}

		skillsListLines = append(skillsListLines, fmt.Sprintf("%s\t%s\t%s", downloadURL, skill.Slug, skill.Version))
		validSkills = append(validSkills, skill)
	}

	if len(skillsListLines) == 0 {
		logger.Warn("没有有效的技能需要安装")
		return
	}

	skillsList := strings.Join(skillsListLines, "\n")
	// base64 编码，避免换行符/特殊字符在 TAT 模板替换时破坏 shell 脚本
	skillsListB64 := base64.StdEncoding.EncodeToString([]byte(skillsList))

	// 计算超时：每 10 个技能 180s，最高 1800s
	timeout := uint64((len(validSkills)/10 + 1) * 180)
	if timeout > 1800 {
		timeout = 1800
	}

	// 通过 TAT 执行 batch_install_skills_from_smh.sh，最多重试 5 次
	var output string
	var tatErr error

	// 确保 runtimeUser 已探测：直接同步调用 ensureRuntimeUser，
	// 避免与 AgentChecker 异步的 detectAndSaveRuntimeUser 产生时序竞态。
	runtimeUser := ensureRuntimeUser(ctx, instanceID, cvmInstanceId, agentType)

	// v7：按 agent_type 分派批量安装脚本。
	// 若 agentType 不被支持（理论上上游 guard 已拦截），这里作为最终 fail-closed：
	// 把当前 Installing 状态的记录置为 Failed，并通知用户。
	scriptName, resolveErr := ResolveScript(ctx, "batch_install_skills", agentType)
	if resolveErr != nil {
		logger.Error("解析 batch_install_skills 脚本失败", "error", resolveErr)
		model.DB(ctx).Model(&model.SkillInstallation{}).
			Where("instance_id = ? AND install_status = ?", instanceID, model.SkillInstalling).
			Updates(map[string]interface{}{
				"install_status": model.SkillInstallFailed,
				"error_message":  "不支持的 agent_type: " + agentType,
			})
		if inst.UserID > 0 {
			notifyCtx := hcommon.DetachContext(ctx)
			go createErrorNotification(inst.UserID, instanceID, inst.Name,
				model.NotifyTypeSkillInstallFailed, i18n.T(notifyCtx, i18n.MsgSkillInstallFailedTitle),
				hcommon.I18nError(i18n.MsgUnsupportedAgentType, agentType), notifyCtx)
		}
		return
	}

	for retry := 1; retry <= 5; retry++ {
		output, tatErr = RunScript(ctx, cvmInstanceId, scriptName, timeout, runtimeUser, nil, map[string]string{
			"skills_list": skillsListB64,
		})
		if tatErr == nil {
			break
		}
		logger.Warn("TAT 执行失败，重试", "retry", retry, "error", tatErr)
		time.Sleep(10 * time.Second)
	}

	if tatErr != nil {
		logger.Error("TAT 执行失败，所有重试已用尽", "error", tatErr)
		for _, skill := range validSkills {
			model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
				"install_status": model.SkillInstallFailed,
				"error_message":  "TAT 执行失败: " + tatErr.Error(),
			})
		}
		if inst.UserID > 0 {
			notifyCtx := hcommon.DetachContext(ctx)
			go createErrorNotification(inst.UserID, instanceID, inst.Name,
				model.NotifyTypeSkillInstallFailed, i18n.T(notifyCtx, i18n.MsgSkillInstallFailedTitle),
				hcommon.I18nRichError(tatErr, i18n.MsgSkillScriptExecuteFailed), notifyCtx)
		}
		return
	}

	// ── 阶段3：解析结果并更新状态 ──
	parseAndUpdateInstallResults(ctx, output, validSkills, inst.UserID, instanceID, inst.Name, logger)
}

// batchInstallResult 表示 batch_install_skills_from_smh.sh 输出的 JSON 结构
type batchInstallResult struct {
	Results []struct {
		Slug    string `json:"slug"`
		Version string `json:"version"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"results"`
	Summary struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	} `json:"summary"`
}

// parseAndUpdateInstallResults 解析 TAT 输出并更新安装状态
func parseAndUpdateInstallResults(ctx context.Context, output string, skills []model.SkillInstallation, userID, instanceID uint, instanceName string, logger *slog.Logger) {
	// 从输出中找标记行
	lines := strings.Split(output, "\n")
	var jsonLine string

	// 方法1: 找标记行 "========== BATCH INSTALL RESULTS =========="，取下一行
	for i, line := range lines {
		if strings.Contains(line, "BATCH INSTALL RESULTS") && i+1 < len(lines) {
			jsonLine = strings.TrimSpace(lines[i+1])
			break
		}
	}

	// 方法2: 兜底——从末尾向前找 { 开头的合法 JSON 行
	if jsonLine == "" {
		for i := len(lines) - 1; i >= 0; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "{") {
				jsonLine = trimmed
				break
			}
		}
	}

	if jsonLine == "" {
		logger.Error("TAT 输出中未找到 JSON 结果，标记所有技能为失败")
		for _, skill := range skills {
			model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
				"install_status": model.SkillInstallFailed,
				"error_message":  "TAT 输出中未找到安装结果",
			})
		}
		if userID > 0 {
			notifyCtx := hcommon.DetachContext(ctx)
			go createErrorNotification(userID, instanceID, instanceName,
				model.NotifyTypeSkillInstallFailed, i18n.T(notifyCtx, i18n.MsgSkillInstallFailedTitle),
				hcommon.I18nError(i18n.MsgSkillScriptOutputAbnormal), notifyCtx)
		}
		return
	}

	var result batchInstallResult
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		logger.Error("解析 JSON 结果失败", "error", err, "json_line", jsonLine)
		for _, skill := range skills {
			model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
				"install_status": model.SkillInstallFailed,
				"error_message":  "解析安装结果 JSON 失败",
			})
		}
		rerr := hcommon.I18nRichError(err, i18n.MsgParseSkillInstallResultFailed).WithDetail(err.Error())
		if userID > 0 {
			notifyCtx := hcommon.DetachContext(ctx)
			go createErrorNotification(userID, instanceID, instanceName,
				model.NotifyTypeSkillInstallFailed, i18n.T(notifyCtx, i18n.MsgSkillInstallFailedTitle),
				rerr, notifyCtx)
		}
		return
	}

	// 构建结果映射（slug → result）
	resultMap := make(map[string]struct {
		Status  string
		Message string
	})
	for _, r := range result.Results {
		resultMap[r.Slug] = struct {
			Status  string
			Message string
		}{r.Status, r.Message}
	}

	// 更新每个技能的状态
	for _, skill := range skills {
		if r, ok := resultMap[skill.Slug]; ok {
			if r.Status == "success" {
				model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
					"install_status": model.SkillInstallSuccess,
					"error_message":  "",
				})
			} else {
				model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
					"install_status": model.SkillInstallFailed,
					"error_message":  r.Message,
				})
			}
		} else {
			// 兜底：未在结果中找到的技能标记为失败
			model.DB(ctx).Model(&skill).Updates(map[string]interface{}{
				"install_status": model.SkillInstallFailed,
				"error_message":  "安装结果中未找到该技能",
			})
		}
	}

	logger.Info("技能安装结果已更新",
		"total", result.Summary.Total,
		"success", result.Summary.Success,
		"failed", result.Summary.Failed)

	// 有失败技能时发送汇总通知
	if result.Summary.Failed > 0 && userID > 0 {
		var failedSlugs []string
		for _, r := range result.Results {
			if r.Status != "success" {
				failedSlugs = append(failedSlugs, r.Slug)
			}
		}
		slugsText := strings.Join(failedSlugs, ", ")
		if len(slugsText) > 200 {
			slugsText = slugsText[:200] + "..."
		}
		notifyCtx := hcommon.DetachContext(ctx)
		go createErrorNotification(userID, instanceID, instanceName,
			model.NotifyTypeSkillInstallFailed, i18n.T(notifyCtx, i18n.MsgPartialSkillsInstallFailed),
			hcommon.I18nError(i18n.MsgSkillsBatchFailed, result.Summary.Failed, slugsText), notifyCtx)
	}
}

// tryBuildSkillDownloadURL 尝试为指定 slug 构建 SMH 下载 URL，供 Hermes/ACE 的
// add_skill 脚本在公共源安装失败时作为 fallback 使用。
//
// 查找优先级：
//  1. bundle_skills 表（已启用的技能包中的技能，cos_zip_key 非空）
//  2. skills 表（企业上传的私有技能）
//
// 任一步骤失败或未找到时返回空字符串（不阻断主流程）。
func tryBuildSkillDownloadURL(ctx context.Context, slug string) string {
	// 优先从 bundle_skills 查找（技能包中的技能已同步到 common space）
	var bundleSkill model.BundleSkill
	if err := model.DB(ctx).Where("slug = ? AND cos_zip_key != ''", slug).
		Order("updated_at DESC").First(&bundleSkill).Error; err == nil {
		url, err := BuildCommonSMHDownloadURL(ctx, bundleSkill.CosZipKey, true)
		if err == nil {
			return url
		}
		slog.Warn("[tryBuildSkillDownloadURL] 生成 bundle_skill 下载 URL 失败", "slug", slug, "error", err)
	}

	// fallback: 从 skills 表查找企业技能（最新版本）
	var skill model.Skill
	if err := model.DB(ctx).Where("slug = ? AND cos_zip_key != ''", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&skill).Error; err == nil {
		url, err := buildSMHDownloadURL(ctx, skill.COSZipKey, true)
		if err == nil {
			return url
		}
		slog.Warn("[tryBuildSkillDownloadURL] 生成 skill 下载 URL 失败", "slug", slug, "error", err)
	}

	return ""
}
