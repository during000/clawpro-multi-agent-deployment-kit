package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	sdk "hatchery/internal/tdaimemorysdk"
)

// --- 用户端接口 ---

type switchPlanRequest struct {
	// ID 是实例的 DB 主键，与 InstanceID 二选一即可。
	// 与项目其他双参数接口对齐：同时给定时 ID 优先。
	ID         uint   `json:"id,omitempty"`
	InstanceID string `json:"instance_id"`
	TargetPlan string `json:"target_plan"` // off / free / pro
}

// resolveMemoryInstance 统一解析"按 DB 主键 id 或 CVM instance_id"获取实例，
// 并完成所有权 + agent_type 校验。返回值 (实例, 建议的 HTTP status, error)：
//   - 参数缺失/非法 → 400
//   - 实例不存在     → 404
//   - 不归属当前用户 → 403
//   - agent_type 不支持 memory → 403
func resolveMemoryInstance(ctx context.Context, user *model.User, id uint, instanceID string) (*model.Instance, int, error) {
	if id == 0 && instanceID == "" {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIDRequired)
	}
	// 管理员视角不限所有者；普通用户先用自身 id 限制查询，
	// 若想给管理员"代查"能力也可以这里改用 0；当前保持原行为：先查全表再做 owner 校验，
	// 这样错误码层级和原实现一致（404 实例不存在 vs 403 无权访问）。
	inst, err := findInstanceByIDOrCVMID(ctx, 0, id, instanceID)
	if err != nil {
		if errors.Is(err, ErrInstanceNotFound) {
			if instanceID != "" {
				return nil, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceCVMNotFound, instanceID)
			}
			return nil, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceDBNotFound, id)
		}
		return nil, http.StatusBadRequest, err
	}
	if user.Role != "admin" && inst.UserID != user.ID {
		return nil, http.StatusForbidden, hcommon.I18nError(i18n.MsgNoAccessToInstance)
	}
	if err := checkInstanceSupportsMemory(ctx, inst); err != nil {
		return nil, http.StatusForbidden, err
	}
	return inst, http.StatusOK, nil
}

func resolveMemoryPlanTransition(raw string) (desiredPlan, jobType, switchStatus string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "off":
		return model.MemoryPlanOff, model.TdaiJobTypeSwitchToOff, model.MemorySwitchStatusSwitchingToOff, true
	case "free":
		return model.MemoryPlanFree, model.TdaiJobTypeSwitchToFree, model.MemorySwitchStatusSwitchingToFree, true
	case "pro":
		return model.MemoryPlanPro, model.TdaiJobTypeSwitchToPro, model.MemorySwitchStatusSwitchingToPro, true
	default:
		return "", "", "", false
	}
}

// HandleMemoryPlanSwitch POST /openclaw/memory/plan/switch
// 切换实例记忆计划，返回 task_id 供前端轮询。
func HandleMemoryPlanSwitch(w http.ResponseWriter, r *http.Request) {
	log := Logger(r.Context())
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyPostMethod))
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var req switchPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	inst, status, err := resolveMemoryInstance(r.Context(), user, req.ID, req.InstanceID)
	if err != nil {
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	// 回填 CVM ID：若调用方仅传了 DB 主键 id，把真实 CVM ID 写回 req.InstanceID，
	// 这样下文所有按 instance_id 操作 plugin/job 的逻辑无需改动。
	req.InstanceID = inst.InstanceId

	// 校验 target_plan
	desiredPlan, jobType, switchStatus, ok := resolveMemoryPlanTransition(req.TargetPlan)
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidTargetPlan, req.TargetPlan))
		return
	}

	// 确保 plugin 行存在
	model.EnsureMemoryTDAIPluginRow(r.Context(), req.InstanceID)

	// 前置校验：若有进行中的切换，拒绝新请求
	var plugin model.MemoryTDAIPlugin
	if err := model.DB(r.Context()).Where("instance_id = ?", req.InstanceID).First(&plugin).Error; err == nil {
		if plugin.SwitchStatus != "" {
			writeError(w, r, http.StatusConflict,
				hcommon.I18nError(i18n.MsgMemorySwitchInProgress, req.InstanceID, plugin.SwitchStatus))
			return
		}
		// PRO → FREE 不支持，需先切到 OFF
		if plugin.CurrentPlan == model.MemoryPlanPro && desiredPlan == model.MemoryPlanFree {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgProToFreeNotSupported))
			return
		}
	}

	operator := user.Username

	bizKey := fmt.Sprintf("switch:%s", req.InstanceID)
	log.Info("[MemoryPlanSwitch] 收到切换请求",
		"instance_id", req.InstanceID,
		"target_plan", req.TargetPlan,
		"operator", operator,
	)

	job, err := model.SubmitJob(r.Context(), jobType, bizKey, req.InstanceID, "{}", operator, "")
	if err != nil {
		log.Error("[MemoryPlanSwitch] 提交任务失败",
			"instance_id", req.InstanceID, "job_type", jobType, "operator", operator, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSubmitJobFailed))
		return
	}

	log.Info("[MemoryPlanSwitch] 任务已提交",
		"instance_id", req.InstanceID, "job_id", job.ID, "target_plan", req.TargetPlan, "operator", operator)

	// 更新 plugin 的 desired_plan 和 switch_status
	model.DB(r.Context()).Model(&model.MemoryTDAIPlugin{}).
		Where("instance_id = ?", req.InstanceID).
		Updates(map[string]any{
			"desired_plan":  desiredPlan,
			"switch_status": switchStatus,
			"last_task_id":  job.ID,
		})

	jsonOK(w, map[string]any{
		"task_id":    job.ID,
		"job_type":   job.JobType,
		"state":      job.State,
		"biz_key":    job.BizKey,
		"created_at": job.CreatedAt,
	})
}

// HandleMemoryConfig GET /openclaw/memory/config?instance_id=
// 查询实例记忆配置/状态。
func HandleMemoryConfig(w http.ResponseWriter, r *http.Request) {
	log := Logger(r.Context())
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 同时支持 ?id=<db_pk> 与 ?instance_id=<cvm_id>，与项目其他双参数接口对齐。
	id, instanceID, perr := extractInstanceIDOrCVMID(r)
	if perr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(perr))
		return
	}
	inst, status, err := resolveMemoryInstance(r.Context(), user, id, instanceID)
	if err != nil {
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	// 回填真实 CVM ID，下文按 instance_id 查 plugin 行的逻辑无需改动。
	instanceID = inst.InstanceId
	log.Debug("[MemoryConfig] 收到查询请求", "instance_id", instanceID)

	var plugin model.MemoryTDAIPlugin
	if err := model.DB(r.Context()).Where("instance_id = ?", instanceID).First(&plugin).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNoMemoryConfig, instanceID))
		return
	}

	// 查最近任务
	var lastJob *model.TdaiJob
	if plugin.LastTaskID > 0 {
		var j model.TdaiJob
		if err := model.DB(r.Context()).First(&j, plugin.LastTaskID).Error; err == nil {
			lastJob = &j
		}
	}

	resp := map[string]any{
		"instance_id":      plugin.InstanceID,
		"current_plan":     plugin.CurrentPlan,
		"desired_plan":     plugin.DesiredPlan,
		"switch_status":    plugin.SwitchStatus,
		"status":           plugin.Status,
		"last_error":       plugin.LastError,
		"last_task_id":     plugin.LastTaskID,
		"last_switched_at": plugin.LastSwitchedAt,
	}
	if lastJob != nil {
		resp["last_task"] = map[string]any{
			"id":           lastJob.ID,
			"job_type":     lastJob.JobType,
			"state":        lastJob.State,
			"current_step": lastJob.CurrentStep,
			"progress":     lastJob.Progress,
			"attempt":      lastJob.Attempt,
			"last_error":   lastJob.LastError,
		}
	}

	jsonOK(w, resp)
}

// HandleMemoryTaskDetail GET /openclaw/memory/task?task_id=
// 查询单个任务详情。
func HandleMemoryTaskDetail(w http.ResponseWriter, r *http.Request) {
	_ = Logger(r.Context()) // 暂无业务日志，预留 trace 能力
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	taskIDStr := r.URL.Query().Get("task_id")
	if taskIDStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTaskIDRequired))
		return
	}
	taskID, err := strconv.ParseUint(taskIDStr, 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTaskIDMustBeNumber))
		return
	}

	var job model.TdaiJob
	if err := model.DB(r.Context()).First(&job, taskID).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgTaskNotFound, taskID))
		return
	}
	var inst model.Instance
	if err := model.DB(r.Context()).Where("instance_id = ?", job.InstanceID).First(&inst).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceForTaskNotFound))
		return
	}
	if user.Role != "admin" && inst.UserID != user.ID {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgNoAccessToTask))
		return
	}
	if err := checkInstanceSupportsMemory(r.Context(), &inst); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, job)
}

// HandleMemoryLibraryDetail GET /openclaw/memory/library/detail
// 查询记忆库中的数据记录。
// Pro 版：从远端 VDB 读取（Agent Memory API）
// Free 版：通过 TAT 在 CVM 上执行 read-local-memory 脚本，读取本地 sqlite 数据
//
// Query 参数：
//   - instance_id（必填）
//   - type: persona / scene / memory / conversation（必填）
//   - sub_type: 原子记忆子类型过滤 fact / preference / event（可选，仅 type=memory 时有效）
//   - page / page_size: 分页（memory 和 conversation 需要）
//   - start_time / end_time: 时间过滤，ISO8601（可选，仅 type=conversation 时有效）
func HandleMemoryLibraryDetail(w http.ResponseWriter, r *http.Request) {
	log := Logger(r.Context())
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	q := r.URL.Query()
	recordType := q.Get("type")
	// 同时支持 ?id=<db_pk> 与 ?instance_id=<cvm_id>，与项目其他双参数接口对齐。
	id, instanceID, perr := extractInstanceIDOrCVMID(r)
	if perr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(perr))
		return
	}
	if id == 0 && instanceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIDParamRequired))
		return
	}
	if recordType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTypeParamRequired))
		return
	}

	// 校验实例归属 + agent_type 支持
	inst, status, err := resolveMemoryInstance(r.Context(), user, id, instanceID)
	if err != nil {
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	// 回填真实 CVM ID，下文按 instance_id 操作 plugin/脚本的逻辑无需改动。
	instanceID = inst.InstanceId

	// 校验 type 值
	validTypes := map[string]bool{"persona": true, "scene": true, "memory": true, "conversation": true}
	if !validTypes[recordType] {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidMemoryType, recordType))
		return
	}

	// 原子记忆子类型：拼接到 RecordType 中，如 memory/persona、memory/episodic、memory/instruction
	subType := q.Get("sub_type")
	sdkRecordType := recordType
	if recordType == "memory" && subType != "" {
		validSubTypes := map[string]bool{"persona": true, "episodic": true, "instruction": true}
		if !validSubTypes[subType] {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidMemorySubType, subType))
			return
		}
		sdkRecordType = recordType + "/" + subType
	}

	// 查询实例的 plugin 行
	var plugin model.MemoryTDAIPlugin
	if err := model.DB(r.Context()).Where("instance_id = ?", instanceID).First(&plugin).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNoMemoryConfig, instanceID))
		return
	}

	switch plugin.CurrentPlan {
	case model.MemoryPlanPro:
		log.Info("[MemoryLibraryDetail] Pro 版查询",
			"instance_id", instanceID, "type", sdkRecordType, "space_id", plugin.PoolID)
		handleLibraryDetailPro(w, r, q, instanceID, sdkRecordType, &plugin)
	case model.MemoryPlanFree:
		log.Info("[MemoryLibraryDetail] Free 版查询",
			"instance_id", instanceID, "type", recordType)
		handleLibraryDetailFree(w, r, q, instanceID, recordType)
	default:
		log.Debug("[MemoryLibraryDetail] 实例未开通记忆服务",
			"instance_id", instanceID, "current_plan", plugin.CurrentPlan)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMemoryServiceNotEnabled))
	}
}

// handleLibraryDetailPro 从远端 VDB 读取记忆数据（Pro 版）。
func handleLibraryDetailPro(w http.ResponseWriter, r *http.Request, q url.Values, instanceID, recordType string, plugin *model.MemoryTDAIPlugin) {
	log := Logger(r.Context())
	if plugin.PoolID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgProMemoryNotAllocated))
		return
	}

	// 构建 SDK 请求
	sdkReq := &sdk.DescribeMemSpaceRecordRequest{
		SpaceId:    plugin.PoolID,
		RecordType: recordType,
	}

	// 分页参数（memory* / conversation 需要）
	if strings.HasPrefix(recordType, "memory") || recordType == "conversation" {
		page, _ := strconv.Atoi(q.Get("page"))
		pageSize, _ := strconv.Atoi(q.Get("page_size"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize
		sdkReq.Offset = &offset
		sdkReq.Limit = &pageSize
	}

	// 对话记忆时间过滤 + 按时间倒序
	if recordType == "conversation" {
		if startTime := q.Get("start_time"); startTime != "" {
			sdkReq.StartTime = startTime
		}
		if endTime := q.Get("end_time"); endTime != "" {
			sdkReq.EndTime = endTime
		}
		sdkReq.OrderDirection = "DESC"
	}

	// 调用 Agent Memory API
	client, err := NewMemorySDKClient(r.Context())
	if err != nil {
		log.Error("[MemoryLibraryDetail] Pro 初始化 SDK 失败",
			"instance_id", instanceID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInitMemorySDKFailed))
		return
	}

	resp, err := client.DescribeMemSpaceRecord(r.Context(), sdkReq)
	if err != nil {
		log.Warn("[MemoryLibraryDetail] Pro 查询记忆库记录失败",
			"instance_id", instanceID, "type", recordType, "space_id", plugin.PoolID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryMemoryRecordFailed))
		return
	}

	// 将 VDB 原始 Fields 数组格式转成扁平 key-value，方便前端读取
	flatDocs := flattenVDBDocuments(resp.Documents)

	// 按文档约定的字段名格式化
	formatted := formatProDocuments(flatDocs, recordType)

	result := map[string]any{
		"instance_id": instanceID,
		"space_id":    plugin.PoolID,
		"type":        recordType,
		"total_count": resp.TotalCount,
		"documents":   formatted,
	}

	// persona 类型：取第一条（单条数据）
	if recordType == "persona" || strings.HasPrefix(recordType, "persona") {
		if len(formatted) > 0 {
			result["document"] = formatted[0]
		}
		delete(result, "documents")
	}

	jsonOK(w, result)
}

// formatProDocuments 按记录类型将 VDB 扁平字段映射为 API 文档约定的格式。
func formatProDocuments(docs []map[string]any, recordType string) []map[string]any {
	result := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		out := make(map[string]any)
		out["id"] = doc["id"]

		switch {
		case recordType == "persona":
			// 文档约定：{ "id", "content" }
			out["content"] = doc["content"]

		case recordType == "scene":
			// VDB 存的是带 META 头的 markdown，解析出结构化字段
			// 文档约定：{ "id", "content", "fileName", "summary", "heat", "created", "updated", "body" }
			content, _ := doc["content"].(string)
			out["content"] = content
			out["fileName"] = doc["filename"]
			meta, body := parseSceneMeta(content)
			if meta != nil {
				out["summary"] = meta["summary"]
				out["heat"] = meta["heat"]
				out["created"] = meta["created"]
				out["updated"] = meta["updated"]
			}
			out["body"] = body

		case strings.HasPrefix(recordType, "memory"):
			// 文档约定：{ "id", "content", "type", "priority", "scene_name", "timestamp" }
			out["content"] = doc["text"]
			out["type"] = doc["type"]
			out["priority"] = doc["priority"]
			out["scene_name"] = doc["scene_name"]
			out["timestamp"] = doc["timestamp_str"]

		case recordType == "conversation":
			// 文档约定：{ "id", "role", "content", "sessionKey", "sessionId", "recordedAt", "timestamp" }
			out["role"] = doc["role"]
			out["content"] = doc["message_text"]
			out["sessionKey"] = doc["session_key"]
			out["sessionId"] = doc["session_id"]
			out["timestamp"] = doc["timestamp"]
			// recorded_at_ms → ISO8601
			if msStr, ok := doc["recorded_at_ms"].(string); ok {
				if ms, err := strconv.ParseInt(msStr, 10, 64); err == nil {
					out["recordedAt"] = time.Unix(ms/1000, (ms%1000)*int64(time.Millisecond)).UTC().Format(time.RFC3339)
				} else {
					out["recordedAt"] = msStr
				}
			}

		default:
			// 未知类型，透传所有字段
			out = doc
		}

		result = append(result, out)
	}
	return result
}

// parseSceneMeta 解析场景记忆 markdown 中的 META 头。
// 格式：-----META-START-----\nkey: value\n...\n-----META-END-----\n\nbody...
func parseSceneMeta(content string) (meta map[string]string, body string) {
	const metaStart = "-----META-START-----"
	const metaEnd = "-----META-END-----"

	startIdx := strings.Index(content, metaStart)
	endIdx := strings.Index(content, metaEnd)
	if startIdx < 0 || endIdx < 0 || endIdx <= startIdx {
		return nil, content
	}

	metaBlock := content[startIdx+len(metaStart) : endIdx]
	body = strings.TrimSpace(content[endIdx+len(metaEnd):])

	meta = make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(metaBlock), "\n") {
		if idx := strings.Index(line, ": "); idx > 0 {
			meta[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+2:])
		}
	}
	return meta, body
}

// flattenVDBDocuments 将 VDB 原始文档格式：
//
//	{"Fields": [{"Name":"text","Type":"string","Value":"hello"}, ...], "Id": "xxx"}
//
// 转成扁平 key-value：
//
//	{"text": "hello", ..., "id": "xxx"}
func flattenVDBDocuments(docs []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		flat := make(map[string]any)

		// 提取 Id
		if id, ok := doc["Id"]; ok {
			flat["id"] = id
		}

		// 展开 Fields 数组
		if fields, ok := doc["Fields"].([]any); ok {
			for _, f := range fields {
				field, ok := f.(map[string]any)
				if !ok {
					continue
				}
				name, _ := field["Name"].(string)
				if name == "" {
					continue
				}
				flat[name] = field["Value"]
			}
		}

		result = append(result, flat)
	}
	return result
}

// ensureMemoryPluginFn、ensureHermesMemoryPluginFn、runScriptForMemoryPlanFn 和 runInlineScriptFn 是可替换的函数变量，方便单元测试 mock。
var ensureMemoryPluginFn = ensureMemoryPluginImpl
var ensureHermesMemoryPluginFn = ensureHermesMemoryPluginImpl
var runScriptForMemoryPlanFn = RunScript
var runInlineScriptFn = RunInlineScript

// EnsureMemoryPlugin 确保插件已安装并就绪，根据实例类型分发：
//   - Hermes：调用 ensure_memory_plugin_hermes.sh，仅做文件存在性校验（不自愈）
//   - OpenClaw：调用 ensure_memory_plugin.sh（npm 源），必要时自动安装/重装/升级
//
// 返回 nil 表示就绪；返回 error 表示安装/升级失败，应中止后续操作。
// 导出供 task 包的 handler 作为前置步骤调用。
func EnsureMemoryPlugin(ctx context.Context, instanceID string) error {
	return ensureMemoryPlugin(ctx, instanceID)
}

// SetEnsureMemoryPluginForTest 替换 ensureMemoryPluginFn 并返回还原函数。
// 供外部包（如 task）在测试中 mock EnsureMemoryPlugin 的底层实现。
func SetEnsureMemoryPluginForTest(fn func(context.Context, string) error) func() {
	orig := ensureMemoryPluginFn
	ensureMemoryPluginFn = fn
	return func() { ensureMemoryPluginFn = orig }
}

// ensureMemoryPlugin 根据实例类型分发：
//   - Hermes：调用 ensure_memory_plugin_hermes.sh，仅做文件存在性校验（不自愈）
//   - OpenClaw：调用 ensure_memory_plugin.sh，必要时自动安装/升级
//
// 返回 nil 表示就绪；返回 error 表示未就绪，应中止后续读取。
// 包内使用的别名（兼容本文件中已有的调用点）。
func ensureMemoryPlugin(ctx context.Context, instanceID string) error {
	if model.GetAgentRuntimeType(ctx, LookupAgentType(ctx, instanceID)) == model.AgentTypeHermes {
		return ensureHermesMemoryPluginFn(ctx, instanceID)
	}
	return ensureMemoryPluginFn(ctx, instanceID)
}

func ensureMemoryPluginImpl(ctx context.Context, instanceID string) error {
	_, err := runScriptForMemoryPlanFn(ctx, instanceID, "ensure_memory_plugin.sh", 300, LookupRuntimeUser(ctx, instanceID), nil, map[string]string{
		"plugin":      model.DefaultMemoryTDAIPluginName,
		"min_version": model.DefaultMemoryTDAIMinVersion,
		"dist_tag":    model.DefaultMemoryTDAIDistTag,
	})
	return err
}

// ensureHermesMemoryPluginImpl Hermes 专用：只校验 read-local-memory.mjs 文件存在，不自愈。
// Hermes 插件生命周期由后端 install_hermes_tdai_gateway.sh 闭环管控，此处发现缺失应直接报错。
func ensureHermesMemoryPluginImpl(ctx context.Context, instanceID string) error {
	_, err := runScriptForMemoryPlanFn(ctx, instanceID, "ensure_memory_plugin_hermes.sh", 30, LookupRuntimeUser(ctx, instanceID), nil, nil)
	return err
}

// buildReadLocalMemoryScript 生成在 CVM 上执行 read-local-memory 的内联 bash 脚本。
// 调用前应先通过 ensureMemoryPlugin 确保插件已安装且工具可用。
// pluginRoot 为插件根目录的绝对路径（由调用方通过 ResolveMemoryPluginRoot 探测得到，
// 或对 Hermes 类型直接使用固定路径）。
func buildReadLocalMemoryScript(pluginRoot, scriptArgs string) string {
	return fmt.Sprintf(`#!/bin/bash
export PATH="$HOME/.npm-global/bin:$PATH"
export NO_COLOR=1
cd "%s" || exit 1
_stderr_f="/tmp/_memory_read_stderr_$$"
node ./bin/read-local-memory.mjs %s 2>"$_stderr_f"
rc=$?
if [ $rc -ne 0 ]; then
  cat "$_stderr_f" >&1
fi
# 清理临时文件，校验前缀防止误删
case "$_stderr_f" in /tmp/_memory_read_stderr_*) rm -f "$_stderr_f" ;; esac
exit $rc
`, pluginRoot, scriptArgs)
}

// handleLibraryDetailFree 通过 TAT 在 CVM 上执行 read-local-memory 脚本读取本地数据（Free 版）。
// API type → 脚本 level 映射：persona→L3, scene→L2, memory→L1, conversation→L0
func handleLibraryDetailFree(w http.ResponseWriter, r *http.Request, q url.Values, instanceID, recordType string) {
	log := Logger(r.Context())
	// type → level 映射
	levelMap := map[string]string{
		"persona":      "L3",
		"scene":        "L2",
		"memory":       "L1",
		"conversation": "L0",
	}
	level := levelMap[recordType]

	// 按 agent_type 选择数据目录：Hermes 的数据目录跟插件目录是分离的
	agentType := LookupAgentType(r.Context(), instanceID)
	dataDir := "$HOME/.openclaw/memory-tdai"
	if model.GetAgentRuntimeType(r.Context(), agentType) == model.AgentTypeHermes {
		dataDir = "$HOME/.memory-tencentdb/memory-tdai"
	}

	// 探测插件根目录（Hermes 使用固定路径，OpenClaw 通过 TAT 探测）
	var pluginRoot string
	if agentType == model.AgentTypeHermes {
		pluginRoot = "$HOME/.memory-tencentdb/tdai-memory-openclaw-plugin"
	} else {
		var err error
		pluginRoot, err = ResolveMemoryPluginRoot(r.Context(), instanceID)
		if err != nil {
			log.Warn("[MemoryLibraryDetail] 插件路径探测失败",
				"instance_id", instanceID, "error", err)
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgMemoryPluginPathFailed))
			return
		}
	}

	scriptArgs := fmt.Sprintf("-d %s -L %s --format json", dataDir, level)

	// 分页参数（L0 / L1 支持）
	if recordType == "memory" || recordType == "conversation" {
		page, _ := strconv.Atoi(q.Get("page"))
		pageSize, _ := strconv.Atoi(q.Get("page_size"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 10 {
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		scriptArgs += fmt.Sprintf(" -l %d --offset %d", pageSize, offset)
	}

	// 时间过滤（L0 conversation 支持 --since/--until）
	if recordType == "conversation" {
		if startTime := q.Get("start_time"); startTime != "" {
			scriptArgs += fmt.Sprintf(" --since %q", startTime)
		}
		if endTime := q.Get("end_time"); endTime != "" {
			scriptArgs += fmt.Sprintf(" --until %q", endTime)
		}
	}

	// L1 memory 的 sub_type 过滤
	if recordType == "memory" {
		if subType := q.Get("sub_type"); subType != "" {
			// 白名单校验，防止 shell 注入
			validSubTypes := map[string]bool{"fact": true, "preference": true, "event": true, "persona": true, "episodic": true, "instruction": true}
			if validSubTypes[subType] {
				scriptArgs += fmt.Sprintf(" -f 'type=%s'", subType)
			}
		}
	}

	// L2 scene 特殊处理：先拿列表，再逐个拿文件内容（后端聚合，前端无感）
	if recordType == "scene" {
		handleLibraryDetailFreeScene(w, r, instanceID)
		return
	}

	// 前置检查：确保插件已安装且 read-local-memory 工具可用（旧版插件缺少此工具，会自动升级）
	if err := ensureMemoryPlugin(r.Context(), instanceID); err != nil {
		log.Warn("[MemoryLibraryDetail] 插件就绪检查失败",
			"instance_id", instanceID, "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgMemoryPluginNotReady))
		return
	}

	// 通过 TAT 在 CVM 上执行脚本
	script := buildReadLocalMemoryScript(pluginRoot, scriptArgs)

	log.Info("[MemoryLibraryDetail] Free 执行 read-local-memory",
		"instance_id", instanceID, "type", recordType, "level", level, "script_args", scriptArgs)

	output, err := runInlineScriptFn(r.Context(), instanceID, script, 60)
	if err != nil {
		log.Warn("[MemoryLibraryDetail] Free 版本地读取失败",
			"instance_id", instanceID, "type", recordType, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgReadLocalMemoryFailed))
		return
	}

	log.Info("[MemoryLibraryDetail] Free 脚本执行成功",
		"instance_id", instanceID, "type", recordType, "output_len", len(output))

	// TAT Output 字段有 24KB 截断限制，检测是否被截断
	const tatOutputLimit = 24 * 1024
	if len(output) >= tatOutputLimit {
		log.Warn("[MemoryLibraryDetail] TAT 输出可能被截断（≥24KB），数据量过大",
			"instance_id", instanceID, "type", recordType, "output_len", len(output))
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgLocalMemoryDataTooLarge))
		return
	}

	// 脚本输出的是 JSON，直接解析后返回
	var scriptResult map[string]any
	if err := json.Unmarshal([]byte(output), &scriptResult); err != nil {
		log.Warn("[MemoryLibraryDetail] Free 版解析脚本输出失败",
			"instance_id", instanceID, "type", recordType, "output_len", len(output), "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgParseLocalMemoryFailed))
		return
	}

	// 统一响应格式
	result := map[string]any{
		"instance_id": instanceID,
		"type":        recordType,
		"source":      "local",
	}

	// 根据层级适配响应结构
	switch recordType {
	case "persona":
		// L3: { "level": "L3", "content": "..." }
		result["document"] = map[string]any{"content": scriptResult["content"]}
		result["total_count"] = 1
	case "scene":
		// 不会走到这里，scene 在前面已经单独处理并 return
	case "memory":
		// L1: { "level": "L1", "total": N, "offset": ..., "limit": ..., "data": [...] }
		result["total_count"] = scriptResult["total"]
		result["documents"] = scriptResult["data"]
	case "conversation":
		// L0: { "level": "L0", "total": N, "offset": ..., "limit": ..., "data": [...] }
		result["total_count"] = scriptResult["total"]
		result["documents"] = scriptResult["data"]
	}

	jsonOK(w, result)
}

// handleLibraryDetailFreeScene 场景记忆（L2）专用：后端聚合，先拿文件列表，再逐个拿文件内容。
// 对于单个文件 body 超过 TAT 24KB 限制的，填充提示文本。
func handleLibraryDetailFreeScene(w http.ResponseWriter, r *http.Request, instanceID string) {
	log := Logger(r.Context())

	// 按 agent_type 选择数据目录
	agentType := LookupAgentType(r.Context(), instanceID)
	dataDir := "$HOME/.openclaw/memory-tdai"
	if model.GetAgentRuntimeType(r.Context(), agentType) == model.AgentTypeHermes {
		dataDir = "$HOME/.memory-tencentdb/memory-tdai"
	}

	// 前置检查：确保插件已安装且 read-local-memory 工具可用
	if err := ensureMemoryPlugin(r.Context(), instanceID); err != nil {
		log.Warn("[FreeScene] 插件就绪检查失败", "instance_id", instanceID, "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgMemoryPluginNotReady))
		return
	}

	// 探测插件根目录
	var pluginRoot string
	if agentType == model.AgentTypeHermes {
		pluginRoot = "$HOME/.memory-tencentdb/tdai-memory-openclaw-plugin"
	} else {
		var err error
		pluginRoot, err = ResolveMemoryPluginRoot(r.Context(), instanceID)
		if err != nil {
			log.Warn("[FreeScene] 插件路径探测失败", "instance_id", instanceID, "error", err)
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgMemoryPluginPathFailed))
			return
		}
	}

	mkScript := func(args string) string {
		return buildReadLocalMemoryScript(pluginRoot, args)
	}

	// ── Step 1: 拿文件列表（只有元信息，很小） ──
	listArgs := fmt.Sprintf("-d %s -L L2 --format json", dataDir)
	listOutput, err := runInlineScriptFn(r.Context(), instanceID, mkScript(listArgs), 30)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgReadSceneListFailed))
		return
	}

	var listResult struct {
		Level string `json:"level"`
		Total int    `json:"total"`
		Data  []struct {
			FileName string `json:"fileName"`
			Summary  string `json:"summary"`
			Heat     int    `json:"heat"`
			Created  string `json:"created"`
			Updated  string `json:"updated"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(listOutput), &listResult); err != nil {
		log.Warn("[FreeScene] 解析列表失败", "instance_id", instanceID, "output_len", len(listOutput), "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgParseSceneListFailed))
		return
	}

	log.Info("[FreeScene] 获取到场景文件列表", "instance_id", instanceID, "count", listResult.Total)

	// ── Step 2: 逐个拿文件内容 ──
	const tatOutputLimit = 24 * 1024
	type sceneDoc struct {
		FileName string `json:"fileName"`
		Summary  string `json:"summary"`
		Heat     int    `json:"heat"`
		Created  string `json:"created"`
		Updated  string `json:"updated"`
		Body     string `json:"body"`
	}

	documents := make([]sceneDoc, 0, len(listResult.Data))
	for _, item := range listResult.Data {
		fileArgs := fmt.Sprintf("-d %s -L L2 --format json --file %q", dataDir, item.FileName)
		fileOutput, err := runInlineScriptFn(r.Context(), instanceID, mkScript(fileArgs), 30)

		doc := sceneDoc{
			FileName: item.FileName,
			Summary:  item.Summary,
			Heat:     item.Heat,
			Created:  item.Created,
			Updated:  item.Updated,
		}

		if err != nil {
			log.Warn("[FreeScene] 读取文件失败", "instance_id", instanceID, "file", item.FileName, "error", err)
			doc.Body = i18n.T(r.Context(), i18n.MsgMemoryPlanReadFailed)
		} else if len(fileOutput) >= tatOutputLimit {
			log.Warn("[FreeScene] 文件内容超过 24KB 限制", "instance_id", instanceID, "file", item.FileName, "size", len(fileOutput))
			doc.Body = i18n.T(r.Context(), i18n.MsgMemoryPlanContentTooLarge)
		} else {
			var fileResult struct {
				Body string `json:"body"`
			}
			if err := json.Unmarshal([]byte(fileOutput), &fileResult); err != nil {
				log.Warn("[FreeScene] 解析文件内容失败", "instance_id", instanceID, "file", item.FileName, "error", err)
				doc.Body = i18n.T(r.Context(), i18n.MsgMemoryPlanParseFailed)
			} else {
				doc.Body = fileResult.Body
			}
		}

		documents = append(documents, doc)
	}

	jsonOK(w, map[string]any{
		"instance_id": instanceID,
		"type":        "scene",
		"source":      "local",
		"total_count": listResult.Total,
		"documents":   documents,
	})
}
