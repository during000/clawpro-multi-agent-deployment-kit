package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

const (
	maxAgentTaskPromptRunes = 32000
	maxAgentTaskPageSize    = 100
	maxAgentTaskResultBytes = 2 << 20
)

type createAgentTaskRequest struct {
	InstanceID     uint   `json:"instance_id"`
	ProjectID      uint   `json:"project_id"`
	WorkspacePath  string `json:"workspace_path"`
	Prompt         string `json:"prompt"`
	Executor       string `json:"executor,omitempty"`
	TargetAgentID  string `json:"target_agent_id,omitempty"`
	IMateProjectID string `json:"imate_project_id,omitempty"`
	DeliveryMode   string `json:"delivery_mode,omitempty"`
}

type agentTaskCommandPayload struct {
	AgentType      string `json:"agent_type"`
	ProjectID      uint   `json:"project_id"`
	WorkspacePath  string `json:"workspace_path"`
	Prompt         string `json:"prompt"`
	Executor       string `json:"executor,omitempty"`
	TargetAgentID  string `json:"target_agent_id,omitempty"`
	IMateProjectID string `json:"imate_project_id,omitempty"`
	DeliveryMode   string `json:"delivery_mode,omitempty"`
}

type agentTaskResponse struct {
	ID             uint       `json:"id"`
	InstanceID     uint       `json:"instance_id"`
	InstanceCID    string     `json:"instance_c_id"`
	ProjectID      uint       `json:"project_id"`
	WorkspacePath  string     `json:"workspace_path"`
	AgentType      string     `json:"agent_type"`
	Prompt         string     `json:"prompt"`
	Status         string     `json:"status"`
	Result         string     `json:"result,omitempty"`
	Error          string     `json:"error,omitempty"`
	SessionID      string     `json:"session_id,omitempty"`
	Executor       string     `json:"executor"`
	TargetAgentID  string     `json:"target_agent_id,omitempty"`
	IMateProjectID string     `json:"imate_project_id,omitempty"`
	DeliveryMode   string     `json:"delivery_mode"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

func newAgentTaskResponse(task model.LocalAgentTask) agentTaskResponse {
	response := agentTaskResponse{
		ID:            task.ID,
		InstanceID:    task.InstanceID,
		InstanceCID:   task.InstanceCID,
		ProjectID:     task.ProjectID,
		WorkspacePath: task.WorkspacePath,
		AgentType:     task.AgentType,
		Prompt:        task.Prompt,
		Status:        task.Status,
		Result:        task.Result,
		Error:         task.Error,
		SessionID:     task.SessionID,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		StartedAt:     task.StartedAt,
		FinishedAt:    task.FinishedAt,
	}
	var payload agentTaskCommandPayload
	if json.Unmarshal([]byte(task.Cmd), &payload) == nil {
		response.Executor = payload.Executor
		response.TargetAgentID = payload.TargetAgentID
		response.IMateProjectID = payload.IMateProjectID
		response.DeliveryMode = payload.DeliveryMode
	}
	if response.Executor == "" {
		response.Executor = "codebuddy"
	}
	if response.DeliveryMode == "" {
		response.DeliveryMode = "poll"
	}
	return response
}

// HandleAgentTaskCreate 创建一条由本地 TeamAI/Edge Runtime 执行的 Agent 任务。
// 工作区必须已由目标本地 Agent 上报并绑定到用户所属项目，避免服务端下发任意本地路径。
func HandleAgentTaskCreate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	var req createAgentTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	workspacePath := strings.TrimSpace(req.WorkspacePath)
	prompt := strings.TrimSpace(req.Prompt)
	executor := strings.ToLower(strings.TrimSpace(req.Executor))
	if executor == "" {
		executor = "codebuddy"
	}
	targetAgentID := strings.TrimSpace(req.TargetAgentID)
	imateProjectID := strings.TrimSpace(req.IMateProjectID)
	deliveryMode := strings.ToLower(strings.TrimSpace(req.DeliveryMode))
	if deliveryMode == "" {
		deliveryMode = "poll"
	}
	if executor != "codebuddy" && executor != "imate" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "executor"))
		return
	}
	if deliveryMode != "wss" && deliveryMode != "hook" && deliveryMode != "poll" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "delivery_mode"))
		return
	}
	if executor == "imate" && targetAgentID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "target_agent_id"))
		return
	}
	if executor == "imate" && imateProjectID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "imate_project_id"))
		return
	}
	if len(targetAgentID) > 191 || len(imateProjectID) > 191 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "imate target"))
		return
	}
	if req.InstanceID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_id"))
		return
	}
	if workspacePath == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "workspace_path"))
		return
	}
	if len([]rune(workspacePath)) > 512 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "workspace_path"))
		return
	}
	if prompt == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "prompt"))
		return
	}
	if len([]rune(prompt)) > maxAgentTaskPromptRunes {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "prompt"))
		return
	}

	db := model.DB(r.Context())
	var inst model.Instance
	if err := db.Where("id = ? AND user_id = ? AND source = ?", req.InstanceID, user.ID, model.InstanceSourceLocal).
		First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	var binding model.LocalAgentScopeBinding
	bindingQuery := db.Where("instance_id = ? AND scope = ? AND scope_key = ?",
		inst.ID, model.LocalAgentScopeWorkspace, workspacePath)
	if req.ProjectID > 0 {
		bindingQuery = bindingQuery.Where("project_id = ?", req.ProjectID)
	}
	if err := bindingQuery.First(&binding).Error; err != nil || binding.ProjectID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "workspace_path"))
		return
	}

	var memberCount int64
	if err := db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", binding.ProjectID, user.ID).
		Count(&memberCount).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if memberCount == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgProjectNotFound))
		return
	}

	payload, err := json.Marshal(agentTaskCommandPayload{
		AgentType:      inst.AgentType,
		ProjectID:      binding.ProjectID,
		WorkspacePath:  workspacePath,
		Prompt:         prompt,
		Executor:       executor,
		TargetAgentID:  targetAgentID,
		IMateProjectID: imateProjectID,
		DeliveryMode:   deliveryMode,
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	task := model.LocalAgentTask{
		Identifier:    model.CurrentIdentifier(r.Context()),
		InstanceID:    inst.ID,
		InstanceCID:   inst.InstanceId,
		Type:          model.LocalAgentTaskTypeExecuteAgent,
		Cmd:           string(payload),
		Status:        model.LocalAgentTaskStatusPending,
		OperatorID:    user.ID,
		ProjectID:     binding.ProjectID,
		WorkspacePath: workspacePath,
		AgentType:     inst.AgentType,
		Prompt:        prompt,
	}
	if err := db.Create(&task).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	wakeDelivered := 0
	if deliveryMode == "wss" {
		wakeDelivered = notifyLocalAgentTaskAvailable(task.InstanceID, task.ID)
	}
	jsonOK(w, map[string]any{
		"ok": true, "task": newAgentTaskResponse(task), "wake_delivered": wakeDelivered > 0,
	})
}

// HandleAgentTasks 返回当前用户创建的本地 Agent 任务，支持按 id/project_id/status 筛选。
func HandleAgentTasks(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	query := model.DB(r.Context()).Model(&model.LocalAgentTask{}).
		Where("operator_id = ? AND type = ?", user.ID, model.LocalAgentTaskTypeExecuteAgent)
	if rawID := strings.TrimSpace(r.URL.Query().Get("id")); rawID != "" {
		id, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil || id == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "id"))
			return
		}
		query = query.Where("id = ?", uint(id))
	}
	if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" {
		projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
		if err != nil || projectID == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "project_id"))
			return
		}
		query = query.Where("project_id = ?", uint(projectID))
	}
	if status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status"))); status != "" {
		if !isAgentTaskStatus(status) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "status"))
			return
		}
		query = query.Where("status = ?", status)
	}

	page, pageSize := parsePagination(r)
	if pageSize > maxAgentTaskPageSize {
		pageSize = maxAgentTaskPageSize
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	var tasks []model.LocalAgentTask
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	items := make([]agentTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, newAgentTaskResponse(task))
	}
	jsonOK(w, map[string]any{
		"ok": true, "tasks": items, "total": total, "page": page, "page_size": pageSize,
	})
}

func isAgentTaskStatus(status string) bool {
	switch status {
	case model.LocalAgentTaskStatusPending,
		model.LocalAgentTaskStatusRunning,
		model.LocalAgentTaskStatusSuccess,
		model.LocalAgentTaskStatusFailed,
		model.LocalAgentTaskStatusCancelled:
		return true
	default:
		return false
	}
}
