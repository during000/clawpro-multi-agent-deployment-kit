package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

var rootRequiredTATScripts = map[string]struct{}{
	"cls_agent_installer.sh":    {},
	"cls_agent_uninstaller.sh":  {},
	"restore_post_reinstall.sh": {},
	// openclaw_recovery.sh 是独立本地 SQLite 自愈脚本（由 reinstallAndRestore 在 restore
	// 失败且信号 RESTORE_NEED_DB_REPAIR 命中时下发），需 root 才能修复归属于运行用户的
	// ~/.openclaw/state/openclaw.sqlite 并 chown 回去。
	"openclaw_recovery.sh": {},
	// secure_first_boot.sh 是自定义镜像首装的安全初始化总入口，当前承担
	// 强制轮换 .gateway.auth.token 的职责，未来可在脚本内部按步骤扩展更多
	// 加固动作（如 basePath/allowedOrigins 随机化、SSH host key 重生成等）。
	// 需要 root 才能改写归属于运行用户的 ~/.openclaw/openclaw.json 并 chown 回去，
	// 同时以 runuser/sudo 切换到 runtime_user 重启 user-scope 的 openclaw-gateway。
	"secure_first_boot.sh": {},
	// Hermes / ACE 系列脚本不再强制 root 执行，统一以 runtimeUser 身份下发。
}

// TAT 命令执行的节拍参数。
//
// 这些变量以 var 形式暴露，仅用于单元测试通过包级注入缩短时长，
// 生产代码不应修改。具体含义：
//   - runScriptPollInterval     : RunScript 主循环中两次 DescribeInvocationTasks 之间的轮询间隔。
//   - runScriptDeadlineBuffer   : 在脚本声明的 timeout 之外额外预留的缓冲时间，
//     用于 TAT 下发链路与最后一次状态回写的开销；默认 10s，与 TAT 推荐
//     的"轮询超过脚本超时 10s 即视为异常"保持一致。
//   - asyncFollowupInterval     : WaitResultTimeout 后台续查 goroutine 的轮询间隔。
//   - asyncFollowupTotalTimeout : 后台续查 goroutine 的总追踪时长上限。
var (
	runScriptPollInterval     = 2 * time.Second
	runScriptDeadlineBuffer   = 10 * time.Second
	asyncFollowupInterval     = 10 * time.Second
	asyncFollowupTotalTimeout = 1 * time.Minute
)

const tatRuntimePrelude = `
# TAT 在非登录 shell 下执行，HOME 可能缺失或不正确，先按当前用户显式纠正。
# 注意：避免使用 getent/id 等需要 fork 子进程的命令，在低内存/swap 紧张的机器上
# 容易触发 "fork: Resource temporarily unavailable" / "permission denied"。
# 改为直接读取 /etc/passwd（shell 内建 read），零 fork。
#
# final：部分脚本（见 rootRequiredTATScripts 白名单）出于权限需要以 root 执行，
# 但脚本读取的是实例运行用户（RuntimeUser）的家目录配置。RunScript 会把该用户
# 注入 OPENCLAW_RUNTIME_USER 环境变量，此处优先使用它作为"目标用户"来推导 HOME，
# 避免白名单脚本以 root 跑时 $HOME=/root 读不到 /home/<runtimeuser>/.xxx 配置。
_tat_user="${OPENCLAW_RUNTIME_USER:-${USER:-${LOGNAME:-}}}"
if [ -z "$_tat_user" ] && [ -r /proc/self/loginuid ]; then
  _tat_uid="$(cat /proc/self/status 2>/dev/null | awk '/^Uid:/{print $2; exit}')"
fi
if [ -z "$_tat_user" ]; then
  # 最后 fallback：唯一一次 id 调用
  _tat_user="$(id -un 2>/dev/null || true)"
fi
if [ -n "$_tat_user" ] && [ -r /etc/passwd ]; then
  _tat_home=""
  while IFS=: read -r _pw_name _pw_x _pw_uid _pw_gid _pw_gecos _pw_dir _pw_shell; do
    if [ "$_pw_name" = "$_tat_user" ]; then
      _tat_home="$_pw_dir"
      break
    fi
  done < /etc/passwd
  if [ -n "$_tat_home" ]; then
    export HOME="$_tat_home"
  elif [ "$_tat_user" = "root" ]; then
    export HOME="/root"
  else
    export HOME="/home/$_tat_user"
  fi
fi
unset _tat_user _tat_home _tat_uid _pw_name _pw_x _pw_uid _pw_gid _pw_gecos _pw_dir _pw_shell

export PATH="$HOME/.local/share/pnpm:$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
if [ -z "${XDG_RUNTIME_DIR:-}" ]; then
  # 仅在缺失时通过 /proc/self/status 读取 uid，避免再 fork 一次 id -u
  _tat_runtime_uid="$(awk '/^Uid:/{print $2; exit}' /proc/self/status 2>/dev/null || echo 0)"
  export XDG_RUNTIME_DIR="/run/user/${_tat_runtime_uid}"
  unset _tat_runtime_uid
fi
`

// getEffectiveRuntimeUser 返回实例的有效运行用户。
// 优先使用实例已探测到的 runtimeUser，为空时 fallback 到 root。
func getEffectiveRuntimeUser(runtimeUser string) string {
	if runtimeUser != "" {
		return runtimeUser
	}
	return "root"
}

func homeForUser(user string) string {
	if user == "root" {
		return "/root"
	}
	return fmt.Sprintf("/home/%s", user)
}

// safeRuntimeUserForEnv 校验 runtimeUser 是否可安全注入到 TAT shell 脚本中。
//
// 注入方式是 `export OPENCLAW_RUNTIME_USER=<value>`，若 value 中含 shell 元字符
// （空格/分号/反引号/$(/)等）会造成命令注入。此处严格白名单：
//   - 允许字符：[a-zA-Z0-9_-]
//   - 长度 1..32（Linux 用户名通常 ≤32）
//   - 不允许以 '-' 开头（避免与命令参数混淆）
//
// 不合法时返回空串，调用方应跳过注入（prelude 会自然走 $USER fallback）。
func safeRuntimeUserForEnv(user string) string {
	if user == "" {
		return ""
	}
	if len(user) > 32 {
		return ""
	}
	if user[0] == '-' {
		return ""
	}
	for i := 0; i < len(user); i++ {
		c := user[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return ""
	}
	return user
}

// LookupRuntimeUser 根据 CVM 实例 ID 查询数据库中的 RuntimeUser。
// 用于调用方只有 instanceId 而没有 instance 对象的场景。
// 查不到或为空时 fallback 到 "root"。
func LookupRuntimeUser(ctx context.Context, cvmInstanceId string) string {
	var inst model.Instance
	if err := model.DB(ctx).Select("runtime_user").Where("instance_id = ?", cvmInstanceId).First(&inst).Error; err != nil {
		slog.Warn("[TAT] LookupRuntimeUser 查询失败，fallback root", "instance_id", cvmInstanceId, "error", err)
		return "root"
	}
	return getEffectiveRuntimeUser(inst.RuntimeUser)
}

// ensureRuntimeUserMaxAttempts / ensureRuntimeUserRetryInterval 是 ensureRuntimeUser
// 的重试参数。生产默认 3 次重试 / 间隔 5s；单元测试可以在 setup 中缩短间隔以加速测试。
var (
	ensureRuntimeUserMaxAttempts   = 3
	ensureRuntimeUserRetryInterval = 5 * time.Second
)

// ensureRuntimeUser 确保实例的 RuntimeUser 已被探测并持久化，返回有效的运行用户。
// 与 detectAndSaveRuntimeUser（异步由 AgentChecker 触发）不同，此函数是同步的，
// 适合在技能/插件安装等必须知道正确用户的场景下调用。
//
// 逻辑：
//  1. 先查 DB，若 runtime_user 已有非空值则直接返回（AgentChecker 先跑完的情况）。
//  2. 若为空，带重试地同步执行探测脚本（最多 3 次，间隔 5s），覆盖 TAT Agent 启动延迟窗口。
//  3. 探测成功：用 WHERE runtime_user = ” 原子写入 DB，避免覆盖并发写入的结果。
//  4. 再次读取 DB 返回最终值（无论是自己写入还是被并发写入的）。
//  5. 所有重试均失败时：不写入 DB（避免写入可能错误的兜底值污染数据），返回临时 "root"。
//     DB 中 runtime_user 保持为空，后续调用（定时任务等）会再次触发探测。
func ensureRuntimeUser(ctx context.Context, instancePK uint, cvmInstanceId string, agentType string) string {
	// 1. 先查 DB
	var inst model.Instance
	if err := model.DB(ctx).Select("runtime_user").Where("id = ?", instancePK).First(&inst).Error; err != nil {
		slog.Warn("[ensureRuntimeUser] 查询实例失败", "instance_pk", instancePK, "error", err)
		return "root"
	}
	if inst.RuntimeUser != "" {
		return inst.RuntimeUser
	}

	// 2. DB 中为空，带重试地同步执行探测脚本
	scriptName, resolveErr := ResolveScript(ctx, "detect_install", agentType)
	if resolveErr != nil {
		slog.Warn("[ensureRuntimeUser] 未找到探测脚本", "instance_pk", instancePK, "agent_type", agentType, "error", resolveErr)
		return "root"
	}

	var output string
	var lastErr error
	for attempt := 1; attempt <= ensureRuntimeUserMaxAttempts; attempt++ {
		output, lastErr = RunScript(ctx, cvmInstanceId, scriptName, 30, "", nil, nil)
		if lastErr == nil {
			break
		}
		slog.Warn("[ensureRuntimeUser] 探测脚本执行失败，重试",
			"instance_pk", instancePK, "attempt", attempt, "max_attempts", ensureRuntimeUserMaxAttempts, "error", lastErr)
		if attempt < ensureRuntimeUserMaxAttempts {
			time.Sleep(ensureRuntimeUserRetryInterval)
			// 重试前再查一次 DB，可能 detectAndSaveRuntimeUser 已并发写入
			var check model.Instance
			if model.DB(ctx).Select("runtime_user").Where("id = ?", instancePK).First(&check).Error == nil && check.RuntimeUser != "" {
				return check.RuntimeUser
			}
		}
	}

	if lastErr != nil {
		// 所有重试均失败：不写入 DB，仅返回临时 "root" 供本次调用使用。
		// DB 中 runtime_user 保持为空，后续调用会再次尝试探测。
		slog.Warn("[ensureRuntimeUser] 探测全部失败，本次临时返回 root（不写入DB，等待后续重试）",
			"instance_pk", instancePK, "agent_type", agentType, "error", lastErr)
		return "root"
	}

	// 3. 解析探测结果
	var result struct {
		RuntimeUser string `json:"runtime_user"`
		RuntimeHome string `json:"runtime_home"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		slog.Warn("[ensureRuntimeUser] 解析探测结果失败", "instance_pk", instancePK, "output", output, "error", err)
		return "root"
	}
	if result.RuntimeUser == "" || result.RuntimeUser == "unknown" {
		slog.Warn("[ensureRuntimeUser] 探测脚本未返回有效用户", "instance_pk", instancePK, "agent_type", agentType)
		return "root"
	}

	// 4. 原子写入：仅当 runtime_user 仍为空时才更新，避免覆盖并发写入的结果
	model.DB(ctx).Model(&model.Instance{}).
		Where("id = ? AND runtime_user = ''", instancePK).
		Updates(map[string]interface{}{
			"runtime_user": result.RuntimeUser,
			"runtime_home": result.RuntimeHome,
		})

	// 5. 再读一次 DB 取最终值（可能是自己写的，也可能是并发写的）
	var final model.Instance
	if err := model.DB(ctx).Select("runtime_user").Where("id = ?", instancePK).First(&final).Error; err == nil && final.RuntimeUser != "" {
		return final.RuntimeUser
	}

	return result.RuntimeUser
}

// LookupAgentType 根据 CVM 实例 ID 查询数据库中的 AgentType。
// 用于调用方只有 cvm instance_id 而没有 Instance 对象，但需要按 agent_type 分派脚本时使用。
// 查不到时 fallback 到 "openclaw"（兼容存量数据）。
func LookupAgentType(ctx context.Context, cvmInstanceId string) string {
	var inst model.Instance
	if err := model.DB(ctx).Select("agent_type").Where("instance_id = ?", cvmInstanceId).First(&inst).Error; err != nil {
		slog.Warn("[TAT] LookupAgentType 查询失败，fallback openclaw", "instance_id", cvmInstanceId, "error", err)
		return model.AgentTypeOpenClaw
	}
	return model.NormalizeAgentType(inst.AgentType)
}

// getDefaultTATRunIdentity 返回 TAT 运行用户和工作目录。
// 适用于不涉及具体脚本文件、只需要以实例运行用户身份执行的场景（如 LightClaw 远程命令）。
func getDefaultTATRunIdentity(runtimeUser string) (runUser string, workdir string) {
	user := getEffectiveRuntimeUser(runtimeUser)
	return user, homeForUser(user)
}

// getTATRunIdentity 根据脚本名和 runtimeUser 决定 TAT 运行用户和工作目录。
// 如果脚本在 rootRequiredTATScripts 白名单中，强制以 root 执行。
func getTATRunIdentity(scriptName string, runtimeUser string) (runUser string, workdir string) {
	baseScript := filepath.Base(strings.TrimSpace(scriptName))
	if _, ok := rootRequiredTATScripts[baseScript]; ok {
		return "root", "/root"
	}

	user := getEffectiveRuntimeUser(runtimeUser)
	return user, homeForUser(user)
}

// NewTATClient 创建腾讯云 TAT SDK 客户端。
// 以 var 形式暴露，便于单元测试注入 mock（生产行为与原 func 完全一致，
// 与 NewCVMClient 保持同款测试支持模式）。
var NewTATClient = func(ctx context.Context) (*tat.Client, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return tat.NewClient(credential, CVMRegion, cpf)
}

func checkAgentOnline(client *tat.Client, instanceId string) error {
	req := tat.NewDescribeAutomationAgentStatusRequest()
	req.InstanceIds = common.StringPtrs([]string{instanceId})
	for i := 0; i < 15; i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		resp, err := client.DescribeAutomationAgentStatus(req)
		if err != nil {
			slog.Error("[TAT] 查询 Agent 状态失败", "instance", instanceId, "error", err)
			return hcommon.I18nRichError(err, i18n.MsgQueryTATAgentStatusFailed)
		}
		if resp.Response != nil && len(resp.Response.AutomationAgentSet) > 0 {
			agent := resp.Response.AutomationAgentSet[0]
			if agent.AgentStatus != nil && *agent.AgentStatus == "Online" {
				return nil
			}
			slog.Warn("[TAT] Agent 未就绪", "instance", instanceId, "status", *agent.AgentStatus, "attempt", i+1)
		} else {
			slog.Warn("[TAT] Agent 未就绪", "instance", instanceId, "detail", "无返回数据", "attempt", i+1)
		}
	}
	return hcommon.I18nError(i18n.MsgTATAgentNotReady, instanceId)
}

func logTATRunCommand(instanceID, scriptName, runUser, workdir string, timeout uint64, params map[string]string) {
	slog.Info("[TAT] RunCommand",
		"instance", instanceID,
		"script", scriptName,
		"runUser", runUser,
		"workdir", workdir,
		"timeout", timeout,
		"param_count", len(params),
	)
}

func scriptFailureDetail(scriptName, output string) string {
	if strings.Contains(scriptName, "set_channel") {
		return ""
	}
	return strings.TrimSpace(output)
}

// dispatchScript 是 RunScript 和 RunScriptAsync 的公共命令下发逻辑。
//
// 完整流程：加载脚本文件 → include 展开 → 注入 OPENCLAW_RUNTIME_USER →
// 拼接 tatRuntimePrelude → 创建 TAT 客户端 → 检查 Agent 在线 →
// 构建 RunCommandRequest（含可选参数）→ 调用 RunCommand → 提取 invocationId。
//
// 返回 invocationId 和 TAT client（供调用方后续轮询使用），不负责结果轮询。
func dispatchScript(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, params map[string]string) (invocationId string, client *tat.Client, err error) {
	scriptContent, loadErr := LoadScript(scriptName)
	if loadErr != nil {
		return "", nil, hcommon.I18nError(i18n.MsgTATLoadScriptFailed).WithI18nDetail(i18n.MsgTATScriptWithError, scriptName, loadErr)
	}

	// 展开 `# %INCLUDE% lib_*.sh` 指令（必须在 prelude 拼接之前，
	// 避免 prelude 内容被误当作 include 处理）。
	scriptContent, expandErr := ExpandIncludes(scriptContent, LoadScript)
	if expandErr != nil {
		return "", nil, hcommon.I18nError(i18n.MsgTATScriptIncludeExpandFailed).WithI18nDetail(i18n.MsgTATScriptWithError, scriptName, expandErr)
	}

	// 把实例的 RuntimeUser 注入为 OPENCLAW_RUNTIME_USER 环境变量，
	// 供 tatRuntimePrelude 推导正确的 $HOME。
	// 这在白名单脚本（以 root 身份执行）读取 /home/<runtimeuser>/ 下配置时尤其关键。
	// runtimeUser 在 safeRuntimeUserForEnv 中做了严格白名单过滤，避免 shell injection。
	if safeUser := safeRuntimeUserForEnv(runtimeUser); safeUser != "" {
		scriptContent = fmt.Sprintf("export OPENCLAW_RUNTIME_USER=%s\n", safeUser) + scriptContent
	}

	// TAT 运行在非登录 shell，统一注入用户级 CLI 与 user-bus 运行时环境，
	// 兼容 root/非 root 镜像下 openclaw/clawhub 命令解析。
	scriptContent = tatRuntimePrelude + "\n" + scriptContent

	client, tatErr := NewTATClient(ctx)
	if tatErr != nil {
		slog.Error("[TAT] 创建客户端失败", "script", scriptName, "instance", instanceId, "error", tatErr)
		return "", nil, hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithI18nDetail(i18n.MsgTATClientCreateFailed, tatErr)
	}

	if err := checkAgentOnline(client, instanceId); err != nil {
		return "", nil, hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithDetail(err.Error())
	}

	runReq := tat.NewRunCommandRequest()
	runUser, workdir := getTATRunIdentity(scriptName, runtimeUser)
	runReq.Content = common.StringPtr(base64.StdEncoding.EncodeToString([]byte(scriptContent)))
	runReq.InstanceIds = common.StringPtrs([]string{instanceId})
	runReq.CommandName = common.StringPtr(scriptName)
	runReq.CommandType = common.StringPtr("SHELL")
	runReq.Username = common.StringPtr(runUser)
	runReq.WorkingDirectory = common.StringPtr(workdir)
	runReq.Timeout = common.Uint64Ptr(timeout)
	runReq.SaveCommand = common.BoolPtr(false)

	if len(params) > 0 {
		runReq.EnableParameter = common.BoolPtr(true)
		paramsJSON, jsonErr := json.Marshal(params)
		if jsonErr != nil {
			return "", nil, hcommon.I18nError(i18n.MsgCommandFailed).WithI18nDetail(i18n.MsgSerializeScriptFailedWithDetail, jsonErr)
		}
		runReq.Parameters = common.StringPtr(string(paramsJSON))
	}

	logTATRunCommand(instanceId, scriptName, runUser, workdir, timeout, params)
	resp, runErr := client.RunCommand(runReq)
	if runErr != nil {
		slog.Error("[TAT] RunCommand 失败", "instance", instanceId, "script", scriptName, "error", runErr)
		return "", nil, hcommon.I18nError(i18n.MsgCommandFailed).WithDetail(runErr.Error())
	}

	if resp.Response == nil || resp.Response.InvocationId == nil {
		slog.Error("[TAT] RunCommand 返回无 InvocationId", "instance", instanceId, "script", scriptName)
		return "", nil, hcommon.I18nError(i18n.MsgCommandFailed).WithI18nDetail(i18n.MsgTATNoInvocationId)
	}

	invocationId = *resp.Response.InvocationId
	slog.Info("[TAT] 命令已下发", "script", scriptName, "instance", instanceId, "invocation", invocationId)
	return invocationId, client, nil
}

var (
	ErrTATCommandDispatchFailed = hcommon.I18nError(i18n.MsgTATCommandDispatchFailed) // 命令下发失败
	ErrTATCommandStartFailed    = hcommon.I18nError(i18n.MsgTATCommandStartFailed)    // 命令启动失败
)

// RunScript 通过腾讯云 TAT 在指定实例上执行脚本，返回脚本的标准输出。
// scriptName 为 scripts 目录下的脚本文件名，timeout 为命令执行超时秒数。
// runtimeUser 为实例探测到的运行用户（instance.RuntimeUser），为空时 fallback 到 root。
// onOutput 为可选的流式输出回调，每次轮询到新增输出时调用，传 nil 表示不需要流式输出。
// params 为可选的脚本参数，脚本中使用 {{key}} 占位符引用，传 nil 表示无参数。
func RunScript(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
	invocationId, client, err := dispatchScript(ctx, instanceId, scriptName, timeout, runtimeUser, params)
	if err != nil {
		return "", err
	}

	// Poll invocation result
	descReq := tat.NewDescribeInvocationTasksRequest()
	descReq.Filters = []*tat.Filter{
		{
			Name:   common.StringPtr("invocation-id"),
			Values: common.StringPtrs([]string{invocationId}),
		},
	}

	var outputSoFar string

	deadline := time.Now().Add(time.Duration(timeout)*time.Second + runScriptDeadlineBuffer)
	for time.Now().Before(deadline) {
		time.Sleep(runScriptPollInterval)
		descResp, err := client.DescribeInvocationTasks(descReq)
		if err != nil {
			slog.Error("[TAT] 查询执行结果失败", "script", scriptName, "invocation", invocationId, "error", err)
			return "", hcommon.I18nRichError(err, i18n.MsgTATQueryResultFailed)
		}
		if descResp.Response == nil || len(descResp.Response.InvocationTaskSet) == 0 {
			continue
		}
		task := descResp.Response.InvocationTaskSet[0]
		if task.TaskStatus == nil {
			continue
		}

		// 解码当前已有的输出，推送增量部分
		if task.TaskResult != nil && task.TaskResult.Output != nil {
			decoded, err := base64.StdEncoding.DecodeString(*task.TaskResult.Output)
			if err == nil {
				outputSoFar = string(decoded)
				if onOutput != nil {
					onOutput(outputSoFar)
				}
			}
		}

		status := *task.TaskStatus
		slog.Debug("[TAT] 轮询状态", "script", scriptName, "invocation", invocationId, "status", status)

		// EndTime 有值表示执行已结束
		if task.EndTime != nil && *task.EndTime != "" {
			// TAT 侧的错误信息：仅 DELIVER_FAILED / START_FAILED 等"脚本未开始执行"的状态
			// 才会填充，其他状态（FAILED / TIMEOUT）关键诊断在脚本 stdout 里。
			tatErrInfo := ""
			if task.ErrorInfo != nil {
				tatErrInfo = strings.TrimSpace(*task.ErrorInfo)
			}
			switch status {
			case "SUCCESS":
				output := strings.TrimSpace(outputSoFar)
				slog.Info("[TAT] 执行成功", "script", scriptName, "invocation", invocationId)
				return output, nil
			case "FAILED":
				errMsg := scriptFailureDetail(scriptName, outputSoFar)
				slog.Error("[TAT] 命令执行失败", "script", scriptName, "invocation", invocationId, "output", errMsg)
				return "", hcommon.I18nError(i18n.MsgCommandFailed).WithDetail(errMsg)
			case "TIMEOUT":
				errMsg := scriptFailureDetail(scriptName, outputSoFar)
				slog.Error("[TAT] 命令执行超时", "script", scriptName, "invocation", invocationId)
				return "", hcommon.I18nError(i18n.MsgTATInvocationTimeout).WithDetail(errMsg)
			case "DELIVER_FAILED":
				slog.Error("[TAT] 命令下发失败", "script", scriptName, "invocation", invocationId, "tat_error_info", tatErrInfo)
				return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithDetail(tatErrInfo)
			case "START_FAILED":
				slog.Error("[TAT] 命令启动失败", "script", scriptName, "invocation", invocationId, "tat_error_info", tatErrInfo)
				return "", hcommon.I18nError(i18n.MsgTATCommandStartFailed).WithDetail(tatErrInfo)
			default:
				slog.Warn("[TAT] 执行结束，未知状态", "script", scriptName, "invocation", invocationId, "status", status, "tat_error_info", tatErrInfo)
				return strings.TrimSpace(outputSoFar), nil
			}
		}
	}
	slog.Error("[TAT] 等待执行结果超时", "script", scriptName, "invocation", invocationId)
	go asyncFollowupInvocation(hcommon.DetachContext(ctx), invocationId, scriptName)
	return "", hcommon.I18nError(i18n.MsgTATWaitResultTimeout).WithDetail(fmt.Sprintf("invocation_id=%s, timeout=%s", invocationId, time.Duration(timeout)*time.Second+runScriptDeadlineBuffer))
}

// asyncFollowupInvocation 在主流程已超时返回后，后台继续轮询 TAT，
// 最多再追踪 1 分钟，把最终状态打印到日志便于排查。
// 日志统一带 "[TAT][followup]" 前缀，方便 grep。
//
// ctx 必须由调用方通过 hcommon.DetachContext 脱离 HTTP 请求上下文得到，
// 否则 handler return 后 ctx 被 cancel 会导致 SDK 调用立即失败。
func asyncFollowupInvocation(ctx context.Context, invocationId, scriptName string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[TAT][followup] 异步续查 panic", "invocation", invocationId, "panic", rec)
		}
	}()

	// 在 detached ctx 下重新创建 TAT client，确保多租户语境正确且不复用主流程 client。
	client, err := NewTATClient(ctx)
	if err != nil {
		slog.Warn("[TAT][followup] 创建 TAT client 失败，放弃续查",
			"invocation", invocationId, "script", scriptName, "err", err)
		return
	}

	descReq := tat.NewDescribeInvocationTasksRequest()
	descReq.Filters = []*tat.Filter{
		{Name: common.StringPtr("invocation-id"), Values: common.StringPtrs([]string{invocationId})},
	}

	deadline := time.Now().Add(asyncFollowupTotalTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(asyncFollowupInterval)
		descResp, err := client.DescribeInvocationTasks(descReq)
		if err != nil {
			slog.Warn("[TAT][followup] 查询失败，稍后重试", "invocation", invocationId, "err", err)
			continue
		}
		if descResp.Response == nil || len(descResp.Response.InvocationTaskSet) == 0 {
			continue
		}
		task := descResp.Response.InvocationTaskSet[0]
		if task.TaskStatus == nil || task.EndTime == nil || *task.EndTime == "" {
			continue
		}
		output := ""
		if task.TaskResult != nil && task.TaskResult.Output != nil {
			if decoded, decErr := base64.StdEncoding.DecodeString(*task.TaskResult.Output); decErr == nil {
				output = strings.TrimSpace(string(decoded))
			}
		}
		tatErrInfo := ""
		if task.ErrorInfo != nil {
			tatErrInfo = strings.TrimSpace(*task.ErrorInfo)
		}
		slog.Warn("[TAT][followup] 续查到最终状态",
			"script", scriptName, "invocation", invocationId,
			"final_status", *task.TaskStatus,
			"output", output,
			"tat_error_info", tatErrInfo)
		return
	}
	slog.Warn("[TAT][followup] 续查 1 分钟仍未结束，放弃追踪", "invocation", invocationId, "script", scriptName)
}

// RunInlineScript 通过腾讯云 TAT 在指定实例上执行内联脚本内容，返回标准输出。
// 与 RunScript 不同的是，scriptContent 直接传入脚本内容而非从文件加载。
func RunInlineScript(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
	log := Logger(ctx)

	client, err := NewTATClient(ctx)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithI18nDetail(i18n.MsgTATClientCreateFailed, err)
	}

	if err := checkAgentOnline(client, instanceId); err != nil {
		return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithDetail(err.Error())
	}

	runReq := tat.NewRunCommandRequest()
	// TAT 运行在非登录 shell，统一注入用户级 CLI 与 user-bus 运行时环境，
	// 兼容 root/非 root 镜像下 $HOME 与 XDG 变量的正确性（与 RunScript 保持一致）。
	finalContent := tatRuntimePrelude + "\n" + scriptContent
	runReq.Content = common.StringPtr(base64.StdEncoding.EncodeToString([]byte(finalContent)))
	runReq.InstanceIds = common.StringPtrs([]string{instanceId})
	runReq.CommandName = common.StringPtr("inline_script")
	runReq.CommandType = common.StringPtr("SHELL")
	// 动态决定运行用户与工作目录：从 DB 查实例真实 RuntimeUser；
	// 为空（例如未探测完成）时 fallback 到 root。
	// 不能硬编码 root：新镜像的 openclaw 默认以非 root 账号运行，
	// 脚本里通常使用 $HOME 读取数据（如 ~/.openclaw/...），以 root 执行会读不到正确路径。
	runUser, workdir := getDefaultTATRunIdentity(LookupRuntimeUser(ctx, instanceId))
	runReq.Username = common.StringPtr(runUser)
	runReq.WorkingDirectory = common.StringPtr(workdir)
	runReq.Timeout = common.Uint64Ptr(timeout)
	runReq.SaveCommand = common.BoolPtr(false)

	slog.Info("[TAT] RunInlineScript", "instance", instanceId, "timeout", timeout, "run_user", runUser)
	resp, err := client.RunCommand(runReq)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(err.Error())
	}

	if resp.Response == nil || resp.Response.InvocationId == nil {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithI18nDetail(i18n.MsgTATNoInvocationId)
	}

	invocationId := *resp.Response.InvocationId

	descReq := tat.NewDescribeInvocationTasksRequest()
	descReq.Filters = []*tat.Filter{
		{
			Name:   common.StringPtr("invocation-id"),
			Values: common.StringPtrs([]string{invocationId}),
		},
	}

	var outputSoFar string
	deadline := time.Now().Add(time.Duration(timeout+10) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		descResp, err := client.DescribeInvocationTasks(descReq)
		if err != nil {
			log.Error("[TAT][inline] 查询执行结果失败", "instance", instanceId, "invocation", invocationId, "timeout", timeout, "error", err)
			return "", hcommon.I18nError(i18n.MsgTATQueryResultFailed).WithDetail(err.Error())
		}
		if descResp.Response == nil || len(descResp.Response.InvocationTaskSet) == 0 {
			continue
		}
		task := descResp.Response.InvocationTaskSet[0]
		if task.TaskStatus == nil {
			continue
		}
		if task.TaskResult != nil && task.TaskResult.Output != nil {
			decoded, decErr := base64.StdEncoding.DecodeString(*task.TaskResult.Output)
			if decErr == nil {
				outputSoFar = string(decoded)
			}
		}
		if task.EndTime != nil && *task.EndTime != "" {
			status := *task.TaskStatus
			switch status {
			case "SUCCESS":
				return strings.TrimSpace(outputSoFar), nil
			case "FAILED":
				log.Error("[TAT][inline] 命令执行失败", "instance", instanceId, "invocation", invocationId, "output", outputSoFar)
				return "", hcommon.I18nError(i18n.MsgCommandFailed).WithDetail(strings.TrimSpace(outputSoFar))
			case "TIMEOUT":
				log.Error("[TAT][inline] 命令执行超时", "instance", instanceId, "invocation", invocationId)
				return "", hcommon.I18nError(i18n.MsgTATInvocationTimeout).WithDetail(strings.TrimSpace(outputSoFar))
			case "DELIVER_FAILED":
				log.Error("[TAT][inline] 命令下发失败", "instance", instanceId, "invocation", invocationId)
				return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithDetail(strings.TrimSpace(outputSoFar))
			case "START_FAILED":
				log.Error("[TAT][inline] 命令启动失败", "instance", instanceId, "invocation", invocationId)
				return "", hcommon.I18nError(i18n.MsgTATCommandStartFailed).WithDetail(strings.TrimSpace(outputSoFar))
			default:
				return strings.TrimSpace(outputSoFar), nil
			}
		}
	}
	return "", hcommon.I18nError(i18n.MsgTATWaitResultTimeout)
}

// InvocationTaskResult 表示 TAT 命令执行任务的查询结果。
//
// 用于 DescribeInvocationTask 返回给前端轮询，字段语义对前端 JSON 友好：
// Status 是任务状态字符串；Output 已 base64 解码；ExitCode 仅在结束后有效；
// Finished 表示是否到达终态（SUCCESS/FAILED/TIMEOUT/DELIVER_FAILED/START_FAILED）。
type InvocationTaskResult struct {
	Status   string `json:"status"`
	Output   string `json:"output"`
	ExitCode int64  `json:"exit_code"`
	Finished bool   `json:"finished"`
}

// RunScriptAsync 通过腾讯云 TAT 在指定实例上下发脚本，但不轮询结果，立即返回 invocationId。
// 调用方可通过 DescribeInvocationTask 查询执行状态和输出。
func RunScriptAsync(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, params map[string]string) (string, error) {
	invocationId, _, err := dispatchScript(ctx, instanceId, scriptName, timeout, runtimeUser, params)
	if err != nil {
		return "", err
	}
	return invocationId, nil
}

// DescribeInvocationTask 按 invocationId 查询一次 TAT 命令执行任务的状态和输出。
func DescribeInvocationTask(ctx context.Context, invocationId string) (*InvocationTaskResult, error) {
	client, err := NewTATClient(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgTATQueryResultFailed).WithI18nDetail(i18n.MsgTATClientCreateFailed, err)
	}

	descReq := tat.NewDescribeInvocationTasksRequest()
	descReq.Filters = []*tat.Filter{
		{
			Name:   common.StringPtr("invocation-id"),
			Values: common.StringPtrs([]string{invocationId}),
		},
	}

	descResp, err := client.DescribeInvocationTasks(descReq)
	if err != nil {
		slog.Error("[TAT] 查询执行结果失败", "invocation", invocationId, "error", err)
		return nil, hcommon.I18nError(i18n.MsgTATQueryResultFailed).WithDetail(err.Error())
	}

	if descResp.Response == nil || len(descResp.Response.InvocationTaskSet) == 0 {
		return &InvocationTaskResult{
			Status:   "PENDING",
			Finished: false,
		}, nil
	}

	task := descResp.Response.InvocationTaskSet[0]
	result := &InvocationTaskResult{
		Status:   "PENDING",
		Finished: false,
	}

	if task.TaskStatus != nil {
		result.Status = *task.TaskStatus
	}

	if task.TaskResult != nil {
		if task.TaskResult.Output != nil {
			decoded, decErr := base64.StdEncoding.DecodeString(*task.TaskResult.Output)
			if decErr == nil {
				result.Output = string(decoded)
			}
		}
		if task.TaskResult.ExitCode != nil {
			result.ExitCode = *task.TaskResult.ExitCode
		}
	}

	if task.EndTime != nil && *task.EndTime != "" {
		result.Finished = true
	}

	return result, nil
}
