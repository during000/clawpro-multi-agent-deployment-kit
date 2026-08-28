package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// envKeyPattern 校验环境变量名：标准 bash 变量名规则。
var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// maxEnvKeys 单次请求最大环境变量数量。
const maxEnvKeys = 50

// setEnvRequest 是 POST /openclaw/set-env 的请求体。
type setEnvRequest struct {
	ID         uint                   `json:"id"`
	InstanceID string                 `json:"instance_id"`
	Env        map[string]interface{} `json:"env"`
}

// HandleSetEnv 为实例批量设置/删除环境变量（通过 TAT 远程执行 set_env.sh）。
// POST /openclaw/set-env
func HandleSetEnv(w http.ResponseWriter, r *http.Request) {
	handleSetEnv(w, r, defaultStatusResolver)
}

func handleSetEnv(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var req setEnvRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 先校验 env 参数，再查 DB，避免无效请求浪费查询
	if len(req.Env) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnvRequired))
		return
	}
	if len(req.Env) > maxEnvKeys {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnvCountLimit, maxEnvKeys))
		return
	}

	// 校验 key 格式，value 只能是 string 或 null
	for key, val := range req.Env {
		if !envKeyPattern.MatchString(key) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidEnvName, key))
			return
		}
		if val != nil {
			if _, ok := val.(string); !ok {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidEnvValue, key))
				return
			}
		}
	}

	instance, err := getInstanceForEnv(r.Context(), &w, user, req.ID, req.InstanceID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：环境变量需通过 TAT 下发到 CVM，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许设置环境变量
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	envJSON, err := json.Marshal(req.Env)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMarshalParamsFailed))
		return
	}

	params := map[string]string{"env_json": string(envJSON)}
	// final：按 agent_type 分派 set_env 脚本
	// - openclaw：写 systemd user drop-in (openclaw-gateway.service.d)
	// - hermes：优先 harness env set，兜底 systemd drop-in
	// - ace：lightclaw env set
	scriptName, rerr := ResolveScript(r.Context(), "set_env", instance.AgentType)
	if rerr != nil {
		slog.Warn("解析 set_env 脚本失败", "agent_type", instance.AgentType, "error", err)
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nRichError(rerr, i18n.MsgAgentTypeNotSupportSetEnv, instance.AgentType))
		return
	}
	if _, err := RunScript(r.Context(), instance.InstanceId, scriptName, 120, instance.RuntimeUser, nil, params); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSetEnvFailed))
		return
	}

	slog.Info("[Env] 环境变量设置成功", "instance_id", instance.InstanceId, "user", user.Username, "keys", len(req.Env), "agent_type", instance.AgentType)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleGetEnv 查看实例当前环境变量（通过 TAT 远程执行 get_env.sh）。
// GET /openclaw/env?id=123 或 ?instance_id=ins-xxx
func HandleGetEnv(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	idStr := r.URL.Query().Get("id")
	instanceID := r.URL.Query().Get("instance_id")

	var id uint
	if idStr != "" {
		parsed, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || parsed == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidID))
			return
		}
		id = uint(parsed)
	}

	instance, err := getInstanceForEnv(r.Context(), &w, user, id, instanceID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	// final：按 agent_type 分派 get_env 脚本
	scriptName, rerr := ResolveScript(r.Context(), "get_env", instance.AgentType)
	if rerr != nil {
		slog.Warn("解析 get_env 脚本失败", "agent_type", instance.AgentType, "error", rerr)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentTypeNotSupportGetEnv, instance.AgentType))
		return
	}
	output, err := RunScript(r.Context(), instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, nil)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGetEnvFailed))
		return
	}

	// 解析脚本输出的 JSON
	var env map[string]string
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		slog.Error("[Env] 解析 get_env 输出失败", "instance_id", instance.InstanceId, "agent_type", instance.AgentType, "output", output, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgParseEnvFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "env": env})
}

// getInstanceForEnv 根据数据库 ID 或 CVM 实例 ID 查询实例，附加所有权校验。
// id 优先于 instanceID；两者均为空时返回错误。
func getInstanceForEnv(ctx context.Context, w *http.ResponseWriter, user *model.User, id uint, instanceID string) (*model.Instance, error) {
	instance, err := findInstanceByIDOrCVMID(ctx, user.ID, id, instanceID)
	if err != nil {
		// 兼容历史错误文案：保持 "实例不存在"，与原实现一致。
		if errors.Is(err, ErrInstanceNotFound) {
			return nil, hcommon.I18nError(i18n.MsgInstanceNotFound)
		}
		return nil, err
	}
	if instance.InstanceId == "" {
		return nil, hcommon.I18nError(i18n.MsgInstanceNoCVM)
	}

	*w = WrapInstanceId(*w, instance.InstanceId)
	return instance, nil
}
