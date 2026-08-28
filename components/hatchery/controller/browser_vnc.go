package controller

import (
	"bufio"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ========== AI 活跃状态管理（实例维度，纯内存，重启归零） ==========

// aiActiveState 记录单个实例的 AI 任务活跃状态。
// activeRequests 使用 atomic 原子操作，支持同一实例多个并发 LLM 请求。
// graceTimer 实现 Agent Loop 宽限期：当 LLM 请求返回 tool_calls 后，
// 在工具执行期间（不经过 Hatchery）保持 ai_active=true，直到下一次 LLM 请求到来或宽限期到期。
type aiActiveState struct {
	activeRequests int64        // 当前进行中的 LLM 请求数（原子操作）
	lastActiveAt   atomic.Value // 最后活跃时间（time.Time），用于超时兜底
	graceActive    int32        // 宽限期是否激活（0=否，1=是，原子操作）
	graceTimer     *time.Timer  // 宽限期定时器（受 graceMu 保护）
	graceMu        sync.Mutex   // 保护 graceTimer 的并发访问
}

var (
	aiActiveMap   sync.Map // key: uint (Instance.ID) → value: *aiActiveState
	takeoverMap   sync.Map // key: uint (Instance.ID) → value: bool
	installingMap sync.Map // key: uint (Instance.ID) → value: *struct{}（唯一 token），防止同一实例并发安装
)

// aiActiveTimeout 超时兜底时间：如果 lastActiveAt 超过此时间仍未更新，
// 强制视为非活跃，防止异常情况下永久锁定浏览器。
const aiActiveTimeout = 10 * time.Minute

// aiGracePeriod Agent Loop 宽限期：LLM 返回 tool_calls 后，等待下一次 LLM 请求的最大时间。
// 覆盖 Gateway 执行工具（如浏览器操作）的间隙，典型工具执行时间 2-30 秒。
const aiGracePeriod = 45 * time.Second

// getOrCreateState 获取或创建指定实例的 aiActiveState。
func getOrCreateState(instanceID uint) *aiActiveState {
	val, _ := aiActiveMap.LoadOrStore(instanceID, &aiActiveState{})
	return val.(*aiActiveState)
}

// MarkAIActive 标记指定实例有新的 LLM 请求开始（activeRequests 原子 +1）。
// 同时取消已有的宽限期定时器（新请求到来意味着 Agent Loop 仍在继续）。
// 在 HandleLLMProxy 中 quota 检查通过后调用。
func MarkAIActive(instanceID uint) {
	state := getOrCreateState(instanceID)
	newCount := atomic.AddInt64(&state.activeRequests, 1)
	state.lastActiveAt.Store(time.Now())

	// 新请求到来，取消宽限期（Agent Loop 仍在继续，无需宽限期兜底）
	wasGraceActive := atomic.LoadInt32(&state.graceActive) == 1
	cancelGraceTimer(state)
	slog.Info("[BrowserVNC] MarkAIActive: 新 LLM 请求开始",
		"instance_id", instanceID,
		"active_requests", newCount,
		"cancelled_grace", wasGraceActive)
}

// MarkAIInactive 标记指定实例的一个 LLM 请求结束（activeRequests 原子 -1）。
// 在 HandleLLMProxy 中通过 defer 调用，确保正常/异常/panic 均触发。
// 已废弃：请使用 MarkAIInactiveWithContext 以支持宽限期。
func MarkAIInactive(instanceID uint) {
	MarkAIInactiveWithContext(instanceID, false)
}

// MarkAIInactiveWithContext 标记指定实例的一个 LLM 请求结束，并根据 hasToolCalls 决定是否启动宽限期。
//   - hasToolCalls=true：LLM 返回了 tool_calls，预期后续还有 LLM 请求 → 启动宽限期
//   - hasToolCalls=false：LLM 返回纯文本，Agent Loop 即将结束 → 不启动宽限期，立即归零
func MarkAIInactiveWithContext(instanceID uint, hasToolCalls bool) {
	val, ok := aiActiveMap.Load(instanceID)
	if !ok {
		slog.Warn("[BrowserVNC] MarkAIInactive: 实例状态不存在",
			"instance_id", instanceID, "has_tool_calls", hasToolCalls)
		return
	}
	state := val.(*aiActiveState)
	newCount := atomic.AddInt64(&state.activeRequests, -1)

	// 负数保护：防止异常情况下（如 panic 后 defer 执行两次）activeRequests 变为负数
	if newCount < 0 {
		atomic.StoreInt64(&state.activeRequests, 0)
		newCount = 0
		slog.Warn("[BrowserVNC] activeRequests 变为负数，已修正为 0",
			"instance_id", instanceID)
	}

	slog.Info("[BrowserVNC] MarkAIInactive: LLM 请求结束",
		"instance_id", instanceID,
		"has_tool_calls", hasToolCalls,
		"active_requests_after", newCount,
		"grace_active", atomic.LoadInt32(&state.graceActive) == 1)

	// 仅当所有并发请求都结束时才考虑宽限期
	if newCount > 0 {
		slog.Info("[BrowserVNC] 仍有并发请求进行中，跳过宽限期处理",
			"instance_id", instanceID, "remaining", newCount)
		return
	}

	if hasToolCalls {
		// LLM 返回了 tool_calls → 启动宽限期，等待 Gateway 执行工具后发起下一次 LLM 请求
		startGraceTimer(instanceID, state)
		slog.Info("[BrowserVNC] LLM 返回 tool_calls，启动宽限期",
			"instance_id", instanceID, "grace_period", aiGracePeriod)
	} else {
		// LLM 返回纯文本 → Agent Loop 即将结束，立即取消宽限期
		cancelGraceTimer(state)
		slog.Info("[BrowserVNC] LLM 返回纯文本，Agent Loop 结束，ai_active 立即归零",
			"instance_id", instanceID)
	}
}

// startGraceTimer 启动宽限期定时器。到期后自动将 graceActive 设为 0。
func startGraceTimer(instanceID uint, state *aiActiveState) {
	state.graceMu.Lock()
	defer state.graceMu.Unlock()

	// 先取消已有的定时器
	if state.graceTimer != nil {
		state.graceTimer.Stop()
	}

	// 激活宽限期标志
	atomic.StoreInt32(&state.graceActive, 1)

	// 启动新定时器
	state.graceTimer = time.AfterFunc(aiGracePeriod, func() {
		atomic.StoreInt32(&state.graceActive, 0)
		slog.Info("[BrowserVNC] 宽限期到期，AI 状态归零", "instance_id", instanceID)
	})
}

// cancelGraceTimer 取消宽限期定时器并立即将 graceActive 设为 0。
func cancelGraceTimer(state *aiActiveState) {
	state.graceMu.Lock()
	defer state.graceMu.Unlock()

	if state.graceTimer != nil {
		state.graceTimer.Stop()
		state.graceTimer = nil
	}
	atomic.StoreInt32(&state.graceActive, 0)
}

// isAIActive 判断指定实例是否有进行中的 AI 操作。
// 三重信号源 OR 运算：
//  1. activeRequests > 0：有进行中的 LLM 请求
//  2. graceActive == 1：宽限期内（LLM 返回 tool_calls 后等待工具执行完成）
//  3. 超时兜底：lastActiveAt 超过 10 分钟自动视为非活跃
func isAIActive(instanceID uint) bool {
	val, ok := aiActiveMap.Load(instanceID)
	if !ok {
		return false
	}
	state := val.(*aiActiveState)

	// 信号源 1：有进行中的 LLM 请求
	count := atomic.LoadInt64(&state.activeRequests)
	if count > 0 {
		// 超时兜底：防止异常情况下 activeRequests 永远不归零
		if lastActive, ok := state.lastActiveAt.Load().(time.Time); ok {
			if time.Since(lastActive) > aiActiveTimeout {
				// 强制归零
				atomic.StoreInt64(&state.activeRequests, 0)
				cancelGraceTimer(state)
				slog.Warn("[BrowserVNC] AI 活跃状态超时兜底，强制归零", "instance_id", instanceID)
				return false
			}
		}
		return true
	}

	// 信号源 2：宽限期内（工具执行间隙）
	if atomic.LoadInt32(&state.graceActive) == 1 {
		return true
	}

	return false
}

// isTakeover 判断指定实例是否处于用户手动接管状态。
func isTakeover(instanceID uint) bool {
	val, ok := takeoverMap.Load(instanceID)
	if !ok {
		return false
	}
	return val.(bool)
}

// CleanupVNCState 清理指定实例的所有 VNC 相关内存状态。
// 在实例删除时调用，防止 sync.Map 中残留已删除实例的条目导致内存泄漏。
func CleanupVNCState(instanceID uint) {
	// 清理 AI 活跃状态（含宽限期定时器）
	if val, ok := aiActiveMap.LoadAndDelete(instanceID); ok {
		state := val.(*aiActiveState)
		cancelGraceTimer(state)
		slog.Info("[BrowserVNC] 已清理 AI 活跃状态", "instance_id", instanceID)
	}
	// 清理接管状态
	if _, ok := takeoverMap.LoadAndDelete(instanceID); ok {
		slog.Info("[BrowserVNC] 已清理接管状态", "instance_id", instanceID)
	}
	// 清理安装锁
	installingMap.Delete(instanceID)
	// 清理 VNC 连接计数器
	vncInstanceConns.Delete(instanceID)
}

// ========== VNC 固定端口常量 ==========

const (
	vncPort        = 5900 // Xvnc 固定端口，install_vnc.sh 硬编码
	websockifyPort = 6080 // websockify WebSocket 代理端口，供 noVNC 浏览器连接
)

// ========== Handler: 获取 VNC 连接信息 ==========

// HandleBrowserVNCAccess 获取指定实例的 VNC 连接地址。
// GET /openclaw/browser-vnc-access?id={instanceID}
// 返回 CVM 公网 IP + 固定端口 5900，以及安全组端口放通状态。
func HandleBrowserVNCAccess(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	// final：云端浏览器仅 OpenClaw 支持，ACE/Hermes 镜像无 noVNC/Chrome 栈。
	if err := checkInstanceSupportsBrowserVNC(r.Context(), instance); err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	siteConfig := model.GetSiteConfig(r.Context())
	// 按用户组解析 VNC 开关
	siteConfig.BrowserVNCEnable = usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyBrowserVNC, instance.GroupID, siteConfig.BrowserVNCEnable)
	browserVNCAccessCore(w, r, instance, siteConfig,
		func(instanceId string) (*cvmInstanceInfo, error) {
			client, rerr := NewCVMClient(r.Context())
			if rerr != nil {
				return nil, hcommon.I18nRichError(rerr, i18n.MsgCreateCVMClientFailed)
			}
			request := cvm.NewDescribeInstancesRequest()
			request.InstanceIds = common.StringPtrs([]string{instanceId})
			response, err := client.DescribeInstances(request)
			if err != nil {
				return nil, err
			}
			if response.Response == nil || len(response.Response.InstanceSet) == 0 {
				return nil, nil
			}
			inst := response.Response.InstanceSet[0]
			info := &cvmInstanceInfo{}
			if inst.InstanceState != nil {
				info.InstanceState = *inst.InstanceState
			}
			if len(inst.PublicIpAddresses) > 0 && inst.PublicIpAddresses[0] != nil {
				info.PublicIp = *inst.PublicIpAddresses[0]
			}
			return info, nil
		},
		// checkPortFn 传 nil：生产路径走 browserVNCAccessCore 的 default 分支，
		// 由 checkPortRuleOnInstanceSG 按 instance.SecurityGroupId 反查 RuleSet 做 drift-aware 检查。
		// checkPortFn 这个注入参数仅保留给单元测试 mock 用（不连 DB/云 API 时）。
		nil,
	)
}

// ========== Handler: 查询 AI 任务状态 ==========

// HandleBrowserStatus 查询指定实例的 AI 任务状态和接管状态。
// GET /openclaw/browser-status?id={instanceID}
// 前端 3 秒轮询此接口，仅涉及一次轻量 DB 查询（功能开关检查）和内存读取，极轻量。
func HandleBrowserStatus(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// final：云端浏览器仅 OpenClaw 支持。
	// 此接口为 3 秒轮询入口，不能 403（否则前端 console 每 3 秒刷错误）。
	// 返回 200 + ai_active=false + takeover=false + unsupported=true，
	// 前端可据此隐藏"AI 操作中"遮罩层与「接管」按钮。
	// 本地 agent 实例同样不支持云端浏览器，走同一转发路径。
	if instance.Source == model.InstanceSourceLocal || !model.AgentTypeSupportsBrowserVNC(r.Context(), instance.AgentType) {
		jsonAPI(w)
		jsonOK(w, map[string]interface{}{
			"ok": true,
			"data": map[string]interface{}{
				"ai_active":   false,
				"takeover":    false,
				"unsupported": true,
				"agent_type":  instance.AgentType,
			},
		})
		return
	}

	siteConfig := model.GetSiteConfig(r.Context())
	vncEnabled := usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyBrowserVNC, instance.GroupID, siteConfig.BrowserVNCEnable)
	browserStatusCore(w, r, instance, vncEnabled)
}

// ========== Handler: 控制浏览器接管 ==========

// HandleBrowserTakeover 控制用户手动接管/释放浏览器。
// POST /openclaw/browser-takeover
// 参数: id={instanceID}, action=start|stop
func HandleBrowserTakeover(w http.ResponseWriter, r *http.Request) {
	handleBrowserTakeover(w, r, defaultStatusResolver)
}

func handleBrowserTakeover(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// final：云端浏览器仅 OpenClaw 支持，ACE/Hermes 无接管场景。
	if err := checkInstanceSupportsBrowserVNC(r.Context(), instance); err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：不支持远程接管浏览器（本地机器无 VNC 接受层）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许接管浏览器
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		jsonAPI(w)
		writeAgentGuardError(w, r, err)
		return
	}

	siteConfig := model.GetSiteConfig(r.Context())
	vncEnabled := usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyBrowserVNC, instance.GroupID, siteConfig.BrowserVNCEnable)
	browserTakeoverCore(w, r, instance, vncEnabled)
}

// ========== Handler: 检查 VNC 环境就绪状态 ==========

// HandleBrowserVNCCheck 检查指定实例的 VNC 云端浏览器环境是否已安装并正常运行。
// GET /openclaw/browser-vnc-check?id={instanceID}
// 通过 TAT 在 CVM 上执行 check_browser_vnc.sh 脚本，检查：
//   - 系统依赖包（tigervnc、openbox、xfce4、fcitx5 等）
//   - noVNC + websockify 安装状态
//   - Google Chrome 安装状态
//   - systemd unit 配置和服务进程状态
//   - 端口监听状态（5900/6080/9222）
func HandleBrowserVNCCheck(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// final：云端浏览器仅 OpenClaw 支持。Check 会触发 5-15s 的 TAT 脚本执行，
	// 对 ACE/Hermes 直接 403 避免无谓的 TAT 下发。
	if err := checkInstanceSupportsBrowserVNC(r.Context(), instance); err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	siteConfig := model.GetSiteConfig(r.Context())
	vncEnabled := usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyBrowserVNC, instance.GroupID, siteConfig.BrowserVNCEnable)
	browserVNCCheckCore(w, r, instance, vncEnabled,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			slog.Info("[BrowserVNC] 开始检查 VNC 环境", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
			output, err := agentScriptRunner(r.Context(), instanceId, script, timeout, runtimeUser, nil, nil)
			if err != nil {
				slog.Error("[BrowserVNC] 检查脚本执行失败", "instance_id", instance.ID, "error", err)
				return output, err
			}
			return output, nil
		},
		fetchCVMOsName,
	)
}

// fetchCVMOsName 通过 DescribeInstances API 获取指定 CVM 实例的操作系统名称。
// 返回示例："Ubuntu Server 24.04 LTS 64bit"、"Windows Server 2022 数据中心版 64位中文版"
// 查询失败或实例不存在时返回空字符串（不影响主流程）。
func fetchCVMOsName(ctx context.Context, instanceId string) string {
	if instanceId == "" {
		return ""
	}
	client, err := NewCVMClient(ctx)
	if err != nil {
		slog.Warn("[BrowserVNC] 获取 OsName 失败: 创建 CVM 客户端失败", "error", err)
		return ""
	}
	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{instanceId})
	response, callErr := client.DescribeInstances(request)
	if callErr != nil {
		slog.Warn("[BrowserVNC] 获取 OsName 失败: DescribeInstances 调用失败", "instance_id", instanceId, "error", callErr)
		return ""
	}
	if response.Response == nil || len(response.Response.InstanceSet) == 0 {
		return ""
	}
	return StrVal(response.Response.InstanceSet[0].OsName)
}

// ========== Handler: 安装 VNC 环境 ==========

// HandleBrowserVNCInstall 在指定实例上安装 VNC 云端浏览器环境。
// POST /openclaw/browser-vnc-install
// 参数: id={instanceID}
// 通过 TAT 在 CVM 上执行 install_browser_vnc.sh 脚本，安装：
//   - 系统依赖包：tigervnc-standalone-server、openbox、xfce4、dbus、fcitx5 等
//   - noVNC v1.5.0 + websockify v0.12.0
//   - Google Chrome (latest stable)
//   - systemd unit 配置和启动脚本
//   - 中文 locale (zh_CN.UTF-8) 及桌面集成
//
// 安装完成后自动启动所有服务（Xvnc:5900、websockify:6080、Chrome:9222）。
// 安装耗时约 1-2 分钟，前端应显示安装进度提示。
func HandleBrowserVNCInstall(w http.ResponseWriter, r *http.Request) {
	handleBrowserVNCInstall(w, r, defaultStatusResolver)
}

func handleBrowserVNCInstall(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// final：云端浏览器仅 OpenClaw 支持。Install 会触发 2-5 分钟的 TAT 脚本执行，
	// 且脚本依赖 openclaw 镜像的系统包与 runtime_user；对 ACE/Hermes 直接 403。
	if err := checkInstanceSupportsBrowserVNC(r.Context(), instance); err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：不支持安装 VNC（与接管浏览器同样需要 CVM 侧支撑）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许安装 VNC
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		jsonAPI(w)
		writeAgentGuardError(w, r, err)
		return
	}

	siteConfig := model.GetSiteConfig(r.Context())
	vncEnabled := usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyBrowserVNC, instance.GroupID, siteConfig.BrowserVNCEnable)
	browserVNCInstallCore(w, r, instance, vncEnabled,
		func(instanceId, script string, timeout uint64, runtimeUser string, unlockInstalling func()) (string, error) {
			slog.Info("[BrowserVNC] 开始安装 VNC 环境", "instance_id", instance.ID, "cvm_id", instance.InstanceId, "user_id", user.ID)
			// 使用独立 context，避免 CLB/Nginx 超时断开 HTTP 连接后 r.Context() 被取消，
			// 导致 TAT 轮询中止（安装脚本实测约 113s，tatCtx 设为 timeout+30 留足余量）。
			// 使用 DetachContext 而非 context.Background()，保留 TenantSnapshot 和链路追踪字段，
			// 避免多租户环境下 getCredential/model.GetSiteConfig 因缺失 Identifier 导致 DB 操作异常。
			tatCtx, tatCancel := context.WithTimeout(hcommon.DetachContext(r.Context()), time.Duration(timeout+30)*time.Second)
			defer tatCancel()

			// 当前端连接断开（r.Context() 取消）时，提前释放安装锁，
			// 允许用户重新发起安装请求，而不必等待 tatCtx 超时（最多 timeout+30s）。
			// unlockInstalling 由 browserVNCInstallCore 生成并传入，内部使用 CompareAndDelete，
			// 确保只删除本请求写入的那把锁，不会误删后续请求的锁。
			go func() {
				select {
				case <-r.Context().Done():
					// 前端连接断开，提前释放锁（TAT 仍在后台继续执行）
					unlockInstalling()
					slog.Info("[BrowserVNC] 前端连接断开，提前释放安装锁", "instance_id", instance.ID)
				case <-tatCtx.Done():
					// tatCtx 超时/完成，由 defer CompareAndDelete 负责清理，此处无需操作
				}
			}()

			output, err := agentScriptRunner(tatCtx, instanceId, script, timeout, runtimeUser, nil, nil)
			if err != nil {
				slog.Error("[BrowserVNC] 安装脚本执行失败", "instance_id", instance.ID, "error", err)
				return output, err
			}
			return output, nil
		},
	)
}

// parseBrowserVNCScriptOutput 从脚本输出中解析最后一个 JSON 对象。
// 脚本输出可能包含多行日志，JSON 结果可能是单行或多行格式化的（jq 输出）。
func parseBrowserVNCScriptOutput(output string) map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// 策略 1：从最后一行向前查找单行 JSON
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err == nil {
			return result
		}
	}

	// 策略 2：从最后一个 `}` 向前找到匹配的 `{`，提取多行 JSON 块
	// 适用于 jq 格式化输出的多行 JSON
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "}" {
			continue
		}
		// 找到结尾的 `}`，向前搜索匹配的 `{`
		depth := 0
		for j := i; j >= 0; j-- {
			trimmed := strings.TrimSpace(lines[j])
			depth += strings.Count(trimmed, "}")
			depth -= strings.Count(trimmed, "{")
			if depth == 0 {
				// 找到匹配的起始行，拼接整个 JSON 块
				jsonBlock := strings.Join(lines[j:i+1], "\n")
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(jsonBlock), &result); err == nil {
					return result
				}
				break
			}
		}
	}

	return nil
}

// ========== 安全组端口放通辅助函数 ==========

// loadVNCWhitelistCIDRs 从 config/clawpro_required_sg_rules.json 中加载
// key 为 "allow_vnc_whitelist" 的规则组，提取所有入站规则的 CidrBlock 列表。
// 规则来源统一管理在 JSON 配置文件中，新增/删除 Hatchery 节点 IP 只需修改 JSON。
func loadVNCWhitelistCIDRs() []requiredSGRule {
	ruleSet := clawproRequiredRuleSet()
	for _, category := range ruleSet.Categories {
		for _, group := range category.RuleGroups {
			if group.Key == "allow_vnc_whitelist" {
				return group.Rules
			}
		}
	}
	return nil
}

// ensureBrowserVNCPortRule 开启云端浏览器时，确保安全组中 websockify 端口 6080
// 仅允许 config/clawpro_required_sg_rules.json 中 allow_vnc_whitelist 规则组定义的白名单 IP 访问。
// 生产入口：使用真实 VPC API 客户端。
func ensureBrowserVNCPortRule(ctx context.Context, securityGroupId string) error {
	return ensureBrowserVNCPortRuleCore(ctx, securityGroupId, loadVNCWhitelistCIDRs, nil)
}

// vncSGClient 封装 ensureBrowserVNCPortRuleCore 所需的 VPC 安全组操作。
// 生产环境传 nil（内部自动创建 VPC 客户端），测试通过注入 mock 实现覆盖。
type vncSGClient interface {
	DescribePolicies(sgId string) ([]*vpc.SecurityGroupPolicy, error)
	DeletePolicies(sgId string, policyIndexes []int64) error
	CreatePolicies(sgId string, policies []*vpc.SecurityGroupPolicy) error
}

// realVncSGClient 是 vncSGClient 的生产实现，封装腾讯云 VPC SDK。
type realVncSGClient struct{ client *vpc.Client }

func (c *realVncSGClient) DescribePolicies(sgId string) ([]*vpc.SecurityGroupPolicy, error) {
	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgId)
	resp, err := c.client.DescribeSecurityGroupPolicies(req)
	if err != nil {
		return nil, err
	}
	if resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
		return nil, nil
	}
	return resp.Response.SecurityGroupPolicySet.Ingress, nil
}

func (c *realVncSGClient) DeletePolicies(sgId string, policyIndexes []int64) error {
	var delPolicies []*vpc.SecurityGroupPolicy
	for _, idx := range policyIndexes {
		idxCopy := idx
		delPolicies = append(delPolicies, &vpc.SecurityGroupPolicy{PolicyIndex: &idxCopy})
	}
	req := vpc.NewDeleteSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgId)
	req.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{Ingress: delPolicies}
	_, err := c.client.DeleteSecurityGroupPolicies(req)
	return err
}

func (c *realVncSGClient) CreatePolicies(sgId string, policies []*vpc.SecurityGroupPolicy) error {
	req := vpc.NewCreateSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgId)
	req.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{Ingress: policies}
	_, err := c.client.CreateSecurityGroupPolicies(req)
	return err
}

// ensureBrowserVNCPortRuleCore 是 ensureBrowserVNCPortRule 的可测试内核。
// loadWhitelistFn 和 client 通过参数注入，方便单元测试 mock。
// client 传 nil 时自动创建真实 VPC 客户端。
//
// 执行逻辑：
//  1. 从 JSON 配置加载 VNC 白名单规则（allow_vnc_whitelist 规则组）；
//  2. 扫描入站规则，找出所有匹配 6080 端口的规则；
//  3. 如果发现 CidrBlock 为 0.0.0.0/0 的旧全网放通规则，按 PolicyIndex 逐条删除（存量迁移）；
//  4. 检查白名单 CIDR 是否全部已存在，缺失的批量添加。
func ensureBrowserVNCPortRuleCore(ctx context.Context,
	securityGroupId string,
	loadWhitelistFn func() []requiredSGRule,
	client vncSGClient,
) error {
	if securityGroupId == "" {
		slog.Warn("[BrowserVNC] 未配置安全组，跳过 VNC 端口放通")
		return hcommon.I18nError(i18n.MsgVNCSGNotConfigured)
	}

	// ① 从 JSON 配置加载白名单规则
	whitelistRules := loadWhitelistFn()
	if len(whitelistRules) == 0 {
		slog.Warn("[BrowserVNC] JSON 配置中未找到 allow_vnc_whitelist 规则组，跳过 VNC 端口放通")
		return hcommon.I18nError(i18n.MsgVNCWhitelistRequired)
	}

	// 构建白名单 CIDR 集合（用于后续比对）
	whitelistCIDRs := make(map[string]requiredSGRule, len(whitelistRules))
	for _, rule := range whitelistRules {
		if rule.CidrBlock != "" {
			whitelistCIDRs[rule.CidrBlock] = rule
		}
	}

	// 如果未注入 client，创建真实 VPC 客户端
	if client == nil {
		vpcCli, err := newVpcClient(ctx)
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
		}
		client = &realVncSGClient{client: vpcCli}
	}

	// ② 查询当前入站规则
	ingress, err := client.DescribePolicies(securityGroupId)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgQuerySGRulesFailed)
	}

	portStr := strconv.Itoa(websockifyPort)
	var legacyPolicyIndexes []int64    // 需要删除的旧 0.0.0.0/0 规则的 PolicyIndex
	existingCIDRs := map[string]bool{} // 已存在的白名单 CIDR

	for _, policy := range ingress {
		if policy.Action == nil || !strings.EqualFold(*policy.Action, "ACCEPT") {
			continue
		}
		if policy.Protocol == nil {
			continue
		}
		proto := strings.ToUpper(*policy.Protocol)
		if proto != "TCP" && proto != "ALL" {
			continue
		}
		// 检查端口是否精确匹配 6080
		if policy.Port == nil || *policy.Port != portStr {
			continue
		}

		cidr := ""
		if policy.CidrBlock != nil {
			cidr = *policy.CidrBlock
		}

		if cidr == "0.0.0.0/0" {
			// 旧的全网放通规则，标记待删除
			if policy.PolicyIndex != nil {
				legacyPolicyIndexes = append(legacyPolicyIndexes, *policy.PolicyIndex)
				slog.Info("[BrowserVNC] 发现旧的 0.0.0.0/0 全网放通规则，将删除",
					"policy_index", *policy.PolicyIndex, "port", portStr)
			}
		} else if cidr != "" {
			existingCIDRs[cidr] = true
		}
	}

	// ③ 删除旧的 0.0.0.0/0 规则（存量迁移）
	if len(legacyPolicyIndexes) > 0 {
		if err := client.DeletePolicies(securityGroupId, legacyPolicyIndexes); err != nil {
			slog.Error("[BrowserVNC] 删除旧 0.0.0.0/0 规则失败", "error", err)
			return hcommon.I18nRichError(err, i18n.MsgVNCDeleteOldSGRuleFailed)
		}
		slog.Info("[BrowserVNC] 已删除旧 0.0.0.0/0 全网放通规则",
			"security_group_id", securityGroupId, "deleted_count", len(legacyPolicyIndexes))
	}

	// ④ 计算缺失的白名单 CIDR，批量添加
	var missingPolicies []*vpc.SecurityGroupPolicy
	for cidr, rule := range whitelistCIDRs {
		if existingCIDRs[cidr] {
			continue // 已存在，跳过
		}
		missingPolicies = append(missingPolicies, &vpc.SecurityGroupPolicy{
			PolicyIndex:       common.Int64Ptr(0),
			CidrBlock:         common.StringPtr(cidr),
			Protocol:          common.StringPtr(rule.Protocol),
			Port:              common.StringPtr(rule.Port),
			Action:            common.StringPtr(rule.Action),
			PolicyDescription: common.StringPtr(rule.Description),
		})
	}

	if len(missingPolicies) > 0 {
		if err := client.CreatePolicies(securityGroupId, missingPolicies); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgVNCAddWhitelistSGRuleFailed)
		}
		slog.Info("[BrowserVNC] 白名单安全组规则添加成功",
			"security_group_id", securityGroupId, "added_count", len(missingPolicies))
	} else if len(legacyPolicyIndexes) == 0 {
		slog.Info("[BrowserVNC] VNC 端口白名单规则已完整，无需变更",
			"security_group_id", securityGroupId, "port", websockifyPort)
	}

	return nil
}

// ensureSinglePortRule 检查并放通单个端口，返回 error 以便调用方感知失败。
func ensureSinglePortRule(ctx context.Context, securityGroupId string, port int, description string) error {
	return ensureSinglePortRuleCore(securityGroupId, port, description,
		func(sgId string, port int) (bool, error) { return checkGatewayUIPortRuleExists(ctx, sgId, port) }, addSecurityGroupPortRule)
}

// ensureSinglePortRuleCore 是 ensureSinglePortRule 的可测试内核。
// 将端口检查和规则创建通过函数参数注入，方便 mock。
func ensureSinglePortRuleCore(
	securityGroupId string,
	port int,
	description string,
	checkFn func(sgId string, port int) (bool, error),
	addFn func(sgId string, port int, desc string) error,
) error {
	portExists, err := checkFn(securityGroupId, port)
	if err != nil {
		slog.Error("[BrowserVNC] 检查端口规则失败", "port", port, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgVNCCheckPortRuleFailed)
	}
	if !portExists {
		if err := addFn(securityGroupId, port, description); err != nil {
			slog.Error("[BrowserVNC] 创建安全组规则失败", "port", port, "error", err)
			return hcommon.I18nRichError(err, i18n.MsgSGCreateSGRulesFailed)
		}
		slog.Info("[BrowserVNC] 端口规则已添加", "port", port, "security_group_id", securityGroupId)
	} else {
		slog.Info("[BrowserVNC] 端口规则已存在，无需添加", "port", port)
	}
	return nil
}

// ========== Handler: VNC WebSocket 反向代理 ==========

// vncProxyConnections 记录当前活跃的 VNC 代理连接数（用于监控和限流）
var vncProxyConnections int64

// maxVNCProxyPerInstance 每个实例最大并发 VNC 代理连接数
const maxVNCProxyPerInstance = 3

// vncInstanceConns 记录每个实例的活跃代理连接数
var vncInstanceConns sync.Map // key: uint (Instance.ID) → value: *int64

// HandleBrowserVNCProxy 将前端 WebSocket 连接代理到 CVM 上的 websockify（6080 端口）。
// GET /openclaw/vnc-ws-proxy?id={instanceID}
//
// 工作原理：
//  1. 验证用户登录态和实例归属
//  2. 获取 CVM 公网 IP
//  3. 将 HTTP 连接升级为原始 TCP（通过 Hijack）
//  4. 向 CVM:6080 发起 WebSocket 握手
//  5. 双向透传数据（浏览器 ↔ Hatchery ↔ CVM）
//
// 优势：
//   - 前端 iframe 通过同域 WSS 连接，无自签名证书信任问题
//   - 经过 Hatchery 认证，CVM 6080 端口无需公网暴露（可选）
//   - 对 noVNC 完全透明，无需修改 noVNC 代码
func HandleBrowserVNCProxy(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 获取实例并校验归属（提前获取，用于分组策略解析）
	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 校验云端浏览器功能是否开启（按 agent 绑定的分组策略）
	siteConfig := model.GetSiteConfig(r.Context())
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyBrowserVNC, instance.GroupID, siteConfig.BrowserVNCEnable) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgVNCFeatureNotEnabled))
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 每实例并发连接数限制
	countPtr := getOrCreateConnCount(instance.ID)
	current := atomic.AddInt64(countPtr, 1)
	if current > maxVNCProxyPerInstance {
		// 超限时立即释放计数，避免占用名额
		atomic.AddInt64(countPtr, -1)
		writeError(w, r, http.StatusTooManyRequests, hcommon.I18nError(i18n.MsgVNCConnectionLimitReached, maxVNCProxyPerInstance))
		return
	}
	// 确保后续任何 return 路径都能正确释放计数
	defer atomic.AddInt64(countPtr, -1)

	// 获取 CVM 公网 IP
	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	response, err := client.DescribeInstances(request)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
		return
	}
	if response.Response == nil || len(response.Response.InstanceSet) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCVMInstanceNotFound))
		return
	}
	inst := response.Response.InstanceSet[0]
	var publicIp string
	if len(inst.PublicIpAddresses) > 0 && inst.PublicIpAddresses[0] != nil {
		publicIp = *inst.PublicIpAddresses[0]
	}
	if publicIp == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInstanceNoPublicIP))
		return
	}

	// 检查 CVM 实例是否处于运行状态
	// inst.InstanceState == nil 表示 CVM API 返回异常数据，同样视为非 RUNNING 状态拒绝连接
	if inst.InstanceState == nil || *inst.InstanceState != "RUNNING" {
		state := "UNKNOWN"
		if inst.InstanceState != nil {
			state = *inst.InstanceState
		}
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgInstanceStateNotRunning, state))
		return
	}

	// 检查请求是否为 WebSocket 升级
	if !isWebSocketUpgrade(r) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgWebSocketRequired))
		return
	}

	// 记录审计日志（WebSocket 长连接在建立时记录一次）
	slog.Info("[BrowserVNC] WebSocket 代理连接建立",
		"instance_id", instance.ID,
		"cvm_id", instance.InstanceId,
		"user_id", user.ID,
		"user", user.Username,
		"target", fmt.Sprintf("%s:%d", publicIp, websockifyPort),
		"active_conns", current)

	// 全局连接计数
	totalConns := atomic.AddInt64(&vncProxyConnections, 1)
	defer func() {
		newTotal := atomic.AddInt64(&vncProxyConnections, -1)
		slog.Info("[BrowserVNC] WebSocket 代理连接关闭",
			"instance_id", instance.ID,
			"user_id", user.ID,
			"remaining_total", newTotal)
	}()
	slog.Info("[BrowserVNC] 当前活跃 VNC 代理连接总数", "total", totalConns)

	// ========== 阶段 1：Hatchery 自己完成与浏览器的 WebSocket 握手 ==========
	// 不转发 CVM 的 101 响应，而是 Hatchery 自己计算 Sec-WebSocket-Accept 并返回 101。
	// 这样可以完全避免 Nginx WebSocket 代理对 101 响应的干扰（头名称规范化、安全头注入等），
	// 确保浏览器收到的 Sec-WebSocket-Accept 与自己发送的 Sec-WebSocket-Key 完全匹配。

	// 从请求头中获取浏览器的 Sec-WebSocket-Key
	wsKey := r.Header.Get("Sec-WebSocket-Key")
	if wsKey == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingWebSocketKey))
		return
	}

	// 先连接 CVM websockify，确认后端可达后再向浏览器返回 101
	backendAddr := fmt.Sprintf("%s:%d", publicIp, websockifyPort)
	backendConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		backendAddr,
		&tls.Config{InsecureSkipVerify: true}, // CVM 使用自签名证书
	)
	if err != nil {
		slog.Error("[BrowserVNC] 连接 CVM websockify 失败", "addr", backendAddr, "error", err)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgVNCCVMConnectFailed))
		return
	}
	defer backendConn.Close() // 确保提前 return 时也能关闭，与 closeConns() 重复关闭是安全的

	// 设置 TCP keepalive 防止中间网络设备清除空闲连接
	if tc, ok := backendConn.NetConn().(interface{ SetKeepAlive(bool) error }); ok {
		tc.SetKeepAlive(true)
	}

	// 向 CVM websockify 发送 WebSocket 升级请求
	// 使用新的 Sec-WebSocket-Key（Hatchery 作为 CVM 的客户端）
	backendWSKey := generateBackendWSKey(instance.ID)
	backendConn.Write([]byte(buildBackendWSRequest(backendAddr, backendWSKey)))

	// 读取 CVM websockify 的 101 响应
	backendBuf := bufio.NewReader(backendConn)
	backendResp, callErr := http.ReadResponse(backendBuf, nil)
	if callErr != nil {
		slog.Error("[BrowserVNC] 读取 CVM websockify 响应失败", "error", callErr)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgVNCBackendResponseAbnormal))
		return
	}
	if backendResp.StatusCode != http.StatusSwitchingProtocols {
		slog.Warn("[BrowserVNC] CVM websockify 未返回 101",
			"status", backendResp.StatusCode)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgVNCBackendReturnedStatus, backendResp.StatusCode))
		return
	}

	slog.Info("[BrowserVNC] CVM websockify 握手成功", "addr", backendAddr)

	// ========== 阶段 2：Hijack 浏览器连接，返回 Hatchery 自己构建的 101 响应 ==========
	hj, ok := w.(http.Hijacker)
	if !ok {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgServerNoUpgradeSupport))
		return
	}
	clientConn, clientBuf, callErr := hj.Hijack()
	if callErr != nil {
		slog.Error("[BrowserVNC] Hijack 失败", "error", callErr)
		return
	}
	defer clientConn.Close() // 确保提前 return 时也能关闭，与 closeConns() 重复关闭是安全的

	// 计算 Sec-WebSocket-Accept（RFC 6455 Section 4.2.2）
	wsAccept := computeWebSocketAccept(wsKey)

	// 构建并发送 101 响应
	subProtocol := r.Header.Get("Sec-WebSocket-Protocol")
	clientConn.Write([]byte(buildClient101Response(wsAccept, subProtocol)))

	slog.Info("[BrowserVNC] 浏览器 WebSocket 握手完成",
		"instance_id", instance.ID,
		"ws_key", wsKey,
		"ws_accept", wsAccept)

	// ========== 阶段 3：双向透传 WebSocket 帧 ==========
	bidirectionalCopy(clientConn, clientBuf, backendConn, backendBuf)
}

// getOrCreateConnCount 获取或创建指定实例的连接计数器。
func getOrCreateConnCount(instanceID uint) *int64 {
	val, _ := vncInstanceConns.LoadOrStore(instanceID, new(int64))
	return val.(*int64)
}

// isWebSocketUpgrade 检查请求是否为 WebSocket 升级请求。
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// computeWebSocketAccept 根据 RFC 6455 Section 4.2.2 计算 Sec-WebSocket-Accept 值。
// SHA1(Sec-WebSocket-Key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11") 后 base64 编码。
func computeWebSocketAccept(wsKey string) string {
	h := sha1.New()
	h.Write([]byte(wsKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// buildBackendWSRequest 构建发送给 CVM websockify 的 WebSocket 升级请求。
func buildBackendWSRequest(backendAddr, backendWSKey string) string {
	var b strings.Builder
	b.WriteString("GET /?token=none HTTP/1.1\r\n")
	b.WriteString(fmt.Sprintf("Host: %s\r\n", backendAddr))
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	b.WriteString(fmt.Sprintf("Sec-WebSocket-Key: %s\r\n", backendWSKey))
	b.WriteString("Sec-WebSocket-Version: 13\r\n")
	b.WriteString("Sec-WebSocket-Protocol: binary\r\n")
	b.WriteString("\r\n")
	return b.String()
}

// buildClient101Response 构建返回给浏览器的 101 Switching Protocols 响应。
func buildClient101Response(wsAccept, subProtocol string) string {
	var b strings.Builder
	b.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	b.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n", wsAccept))
	if subProtocol != "" {
		b.WriteString(fmt.Sprintf("Sec-WebSocket-Protocol: %s\r\n", subProtocol))
	}
	b.WriteString("\r\n")
	return b.String()
}

// generateBackendWSKey 生成 Hatchery 作为 WebSocket 客户端连接 CVM 时使用的 Sec-WebSocket-Key。
func generateBackendWSKey(instanceID uint) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("hatchery-%d-%d", instanceID, time.Now().UnixNano())))
}

// bidirectionalCopy 在两个连接之间双向透传数据。
// 任一方向出错/关闭时，关闭两端连接以通知另一方向退出。
// 等待两个方向的 goroutine 都完成后返回。
func bidirectionalCopy(clientConn net.Conn, clientBuf *bufio.ReadWriter, backendConn net.Conn, backendBuf *bufio.Reader) {
	var wg sync.WaitGroup
	wg.Add(2)

	closeOnce := sync.Once{}
	closeConns := func() {
		closeOnce.Do(func() {
			clientConn.Close()
			backendConn.Close()
		})
	}

	// 客户端 → CVM
	go func() {
		defer wg.Done()
		if clientBuf.Reader.Buffered() > 0 {
			buffered := make([]byte, clientBuf.Reader.Buffered())
			n, _ := clientBuf.Read(buffered)
			if n > 0 {
				backendConn.Write(buffered[:n])
			}
		}
		_, err := io.Copy(backendConn, clientConn)
		if err != nil {
			slog.Debug("[BrowserVNC] 客户端→CVM 方向结束", "error", err)
		}
		closeConns()
	}()

	// CVM → 客户端
	go func() {
		defer wg.Done()
		if backendBuf.Buffered() > 0 {
			buffered := make([]byte, backendBuf.Buffered())
			n, _ := backendBuf.Read(buffered)
			if n > 0 {
				clientConn.Write(buffered[:n])
			}
		}
		_, err := io.Copy(clientConn, backendConn)
		if err != nil {
			slog.Debug("[BrowserVNC] CVM→客户端 方向结束", "error", err)
		}
		closeConns()
	}()

	wg.Wait()
}

// vncProxyValidationResult 封装 VNC 代理前置校验的结果。
type vncProxyValidationResult struct {
	PublicIp string // CVM 公网 IP
	ErrMsg   string // 错误信息（空表示校验通过）
	ErrCode  int    // HTTP 错误码（0 表示校验通过）
}

// validateVNCProxyInstance 校验 VNC 代理请求的前置条件（不含认证和实例归属校验）。
// 检查：功能开关、实例 CVM 关联、并发连接数限制、CVM 状态、公网 IP、WebSocket 升级头。
// 返回校验结果，调用方根据 ErrCode 是否为 0 判断是否通过。
func validateVNCProxyInstance(
	r *http.Request,
	instance *model.Instance,
	browserVNCEnable bool,
	countPtr *int64,
	describeFn func(instanceId string) (*cvmInstanceInfo, error),
) vncProxyValidationResult {
	if !browserVNCEnable {
		return vncProxyValidationResult{ErrMsg: "云端浏览器功能未开启", ErrCode: http.StatusForbidden}
	}

	if instance.InstanceId == "" {
		return vncProxyValidationResult{ErrMsg: "该实例无关联的 CVM", ErrCode: http.StatusBadRequest}
	}

	// 每实例并发连接数限制
	current := atomic.LoadInt64(countPtr)
	if current >= maxVNCProxyPerInstance {
		return vncProxyValidationResult{
			ErrMsg:  fmt.Sprintf("该实例 VNC 连接数已达上限（%d）", maxVNCProxyPerInstance),
			ErrCode: http.StatusTooManyRequests,
		}
	}

	// 查询 CVM 实例信息
	info, err := describeFn(instance.InstanceId)
	if err != nil {
		return vncProxyValidationResult{ErrMsg: "查询实例失败", ErrCode: http.StatusInternalServerError}
	}
	if info == nil {
		return vncProxyValidationResult{ErrMsg: "未找到 CVM 实例", ErrCode: http.StatusNotFound}
	}
	if info.PublicIp == "" {
		return vncProxyValidationResult{ErrMsg: "实例无公网 IP", ErrCode: http.StatusInternalServerError}
	}
	if info.InstanceState != "RUNNING" {
		return vncProxyValidationResult{
			ErrMsg:  fmt.Sprintf("实例当前状态为 %s，云端浏览器仅在实例运行中时可用", info.InstanceState),
			ErrCode: http.StatusConflict,
		}
	}

	// 检查 WebSocket 升级头
	if !isWebSocketUpgrade(r) {
		return vncProxyValidationResult{ErrMsg: "需要 WebSocket 连接", ErrCode: http.StatusBadRequest}
	}

	// 检查 Sec-WebSocket-Key
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		return vncProxyValidationResult{ErrMsg: "缺少 Sec-WebSocket-Key 头", ErrCode: http.StatusBadRequest}
	}

	return vncProxyValidationResult{PublicIp: info.PublicIp}
}

// addSecurityGroupPortRule 为指定端口创建安全组入站规则（通用函数）。
// 注意：此函数仅供 ensureSinglePortRule 等通用逻辑使用。
// VNC 6080 端口的放通已改为通过 ensureBrowserVNCPortRule 从 JSON 白名单配置读取，
// 不再经过此函数。
func addSecurityGroupPortRule(securityGroupId string, port int, description string) error {
	slog.Warn("[BrowserVNC] addSecurityGroupPortRule 被调用，请确认调用方是否应改用白名单模式",
		"security_group_id", securityGroupId, "port", port)
	return hcommon.I18nError(i18n.MsgVNCAddPortRuleDeprecated)
}

// ========== 可测试内核函数（供单元测试直接调用，绕过 requireLogin/getInstanceByID） ==========

// cvmInstanceInfo 封装 CVM 实例查询结果中测试所需的字段。
type cvmInstanceInfo struct {
	InstanceState string
	PublicIp      string
}

// browserVNCAccessCore 是 HandleBrowserVNCAccess 的可测试内核。
//
// 可注入两个函数：
//   - describeFn：查 CVM 实例状态 + 公网 IP。测试可 mock 避免调云 API。
//   - checkPortFn：端口放通检查。**测试专用**，见下面 switch 注释。生产必须传 nil。
func browserVNCAccessCore(
	w http.ResponseWriter,
	r *http.Request,
	instance *model.Instance,
	siteConfig model.SiteConfig,
	describeFn func(instanceId string) (*cvmInstanceInfo, error),
	checkPortFn func(sgId string, port int) (bool, error),
) {
	jsonAPI(w)

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	if !siteConfig.BrowserVNCEnable {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgVNCContactAdminToEnable))
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	info, err := describeFn(instance.InstanceId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if info == nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCVMInstanceNotFound))
		return
	}

	if info.InstanceState != "RUNNING" {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgInstanceStateNotRunningRUN, info.InstanceState))
		return
	}

	if info.PublicIp == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInstanceNoPublicIPForVNC))
		return
	}

	accessible := false
	message := ""
	// ── 端口放通检查：三分支策略 ──────────────────────────────────────────────
	//
	// 生产路径（checkPortFn == nil）：
	//   走 default 分支，调用 checkPortRuleOnInstanceSG：
	//     1. 按 instance.SecurityGroupId 反查 managed_sg_pool → 定位所属 RuleSet
	//     2. 同步态（rule_version == ruleset.version）走 DB 快路径读规则
	//     3. 漂移态（rule_version 落后或 DriftAt 非空）调云 API 读真相
	//   这套逻辑 RuleSet 感知 + drift-aware，是新模型下唯一正确的检查方式。
	//
	// 测试路径（checkPortFn != nil）：
	//   走 case checkPortFn 分支。测试可注入一个 mock fn 避免连 DB / 云 API，
	//   此时检查的是 siteConfig.SecurityGroupId（为简化测试设定，不反映新模型语义）。
	//   ⚠️ 生产严禁传非 nil —— 会越过 RuleSet 层检查到错误的 SG（FROZEN 老 base 或空），
	//   导致存量用户误报"未放通"、新用户被"未配置安全组"卡死。
	//
	// 实例无 SG（case instance.SecurityGroupId == ""）：
	//   实例刚创建还没绑 SG，或异常状态。跳过检查，accessible=false。
	//
	// 必需规则由 admin 开 BrowserVNCEnable 开关时 RefreshAllRuleSetsForRequiredRules
	// 扇出（allow_vnc_whitelist 组），本函数只负责"读"不负责"写"。
	switch {
	case checkPortFn != nil:
		// 测试专用分支
		if siteConfig.SecurityGroupId == "" {
			message = i18n.T(r.Context(), i18n.MsgVNCSGNotConfiguredCheck)
		} else if ok, err := checkPortFn(siteConfig.SecurityGroupId, websockifyPort); err != nil {
			slog.Warn("检查 VNC 端口安全组规则失败", "ws_err", err)
			message = i18n.T(r.Context(), i18n.MsgVNCSGCheckFailed)
		} else if ok {
			accessible = true
		} else {
			message = i18n.T(r.Context(), i18n.MsgVNCSGNotOpened, websockifyPort)
		}
	case strings.TrimSpace(instance.SecurityGroupId) == "":
		// 实例无 SG，跳过检查
		message = i18n.T(r.Context(), i18n.MsgVNCInstanceWithoutSG)
	default:
		// 生产路径：RuleSet 感知 + drift-aware
		// anyCIDR=true：VNC 规则写入端用白名单 IP（/32），读取端不要求 0.0.0.0/0
		ok, d, err := checkPortRuleOnInstanceSG(r.Context(), instance, websockifyPort, "TCP", portRuleCheckOptions{anyCIDR: true})
		if errors.Is(err, ErrSGBootstrapNotDone) {
			// 全新租户：不报错，显示友好 message
			message = err.Error()
		} else if err != nil {
			slog.Warn("检查 VNC 端口放通失败",
				"instance_id", instance.InstanceId, "sg_id", instance.SecurityGroupId, "err", err)
			message = i18n.T(r.Context(), i18n.MsgVNCSGCheckFailed)
		} else if ok {
			accessible = true
		} else {
			if d {
				message = i18n.T(r.Context(), i18n.MsgVNCSGSyncing, websockifyPort)
			} else {
				message = i18n.T(r.Context(), i18n.MsgVNCSGNotOpenedAlt, websockifyPort)
			}
		}
	}

	data := map[string]interface{}{
		"host":       info.PublicIp,
		"port":       vncPort,
		"ws_port":    websockifyPort,
		"accessible": accessible,
	}
	if accessible {
		data["vnc_url"] = fmt.Sprintf("vnc://%s:%d", info.PublicIp, vncPort)
		data["novnc_url"] = fmt.Sprintf("https://%s:%d/vnc.html?autoconnect=true&resize=scale&reconnect=true&reconnect_delay=3000", info.PublicIp, websockifyPort)
		data["ws_proxy_path"] = fmt.Sprintf("/openclaw/vnc-ws-proxy?id=%d", instance.ID)
	} else {
		data["vnc_url"] = ""
		data["novnc_url"] = ""
		data["ws_proxy_path"] = ""
		data["message"] = message
	}

	jsonOK(w, map[string]interface{}{
		"ok":   true,
		"data": data,
	})
}

// browserStatusCore 是 HandleBrowserStatus 的可测试内核。
func browserStatusCore(w http.ResponseWriter, r *http.Request, instance *model.Instance, browserVNCEnable bool) {
	jsonAPI(w)

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	if !browserVNCEnable {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgVNCContactAdminToEnable))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"ai_active": isAIActive(instance.ID),
			"takeover":  isTakeover(instance.ID),
		},
	})
}

// browserTakeoverCore 是 HandleBrowserTakeover 的可测试内核。
func browserTakeoverCore(w http.ResponseWriter, r *http.Request, instance *model.Instance, browserVNCEnable bool) {
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	if !browserVNCEnable {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgVNCContactAdminToEnable))
		return
	}

	action := r.FormValue("action")
	switch action {
	case "start":
		takeoverMap.Store(instance.ID, true)
		slog.Info("[BrowserVNC] 用户开始接管浏览器", "instance_id", instance.ID)
	case "stop":
		takeoverMap.Delete(instance.ID)
		slog.Info("[BrowserVNC] 用户结束接管浏览器", "instance_id", instance.ID)
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidActionParameter))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"takeover": action == "start",
		},
	})
}

// browserVNCInstallCore 是 HandleBrowserVNCInstall 的可测试内核。
// runScriptFn 注入 TAT 脚本执行逻辑；unlockInstalling 由 core 生成并传入，
// 供 runScriptFn 在前端断开时提前释放安装锁（CompareAndDelete 保证不误删后续请求的锁）。
func browserVNCInstallCore(
	w http.ResponseWriter,
	r *http.Request,
	instance *model.Instance,
	browserVNCEnable bool,
	runScriptFn func(instanceId, script string, timeout uint64, runtimeUser string, unlockInstalling func()) (string, error),
) {
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	if !browserVNCEnable {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgVNCContactAdminToEnable))
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 并发安装保护：存储唯一 token（而非 bool），防止跨请求误删锁。
	// 竞态场景：请求A 断开提前释放锁 → 请求B 进入写入新 token → 请求A 的 defer 执行时
	// CompareAndDelete 发现 token 不匹配，不会误删请求B 的锁。
	lockToken := &struct{}{}
	if _, loaded := installingMap.LoadOrStore(instance.ID, lockToken); loaded {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgInstanceInstallingNoRepeat))
		return
	}
	defer installingMap.CompareAndDelete(instance.ID, lockToken)

	// 当前端连接断开（r.Context() 取消）时，提前释放安装锁，
	// 允许用户重新发起安装请求，而不必等待 TAT 脚本执行完毕。
	// lockToken 在此作用域内可见，CompareAndDelete 确保只删除本请求的锁。
	var unlockOnce sync.Once
	unlockInstalling := func() {
		unlockOnce.Do(func() { installingMap.CompareAndDelete(instance.ID, lockToken) })
	}

	// 安装脚本实测约 113s，300s（5分钟）有充足余量，同时避免 tatCtx 330s 超过 CLB 900s 太多
	output, err := runScriptFn(instance.InstanceId, "install_browser_vnc.sh", 300, instance.RuntimeUser, unlockInstalling)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgVNCInstallFailed))
		return
	}

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgParseInstallResultFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":   true,
		"data": result,
	})
}

// browserVNCCheckCore 是 HandleBrowserVNCCheck 的可测试内核。
func browserVNCCheckCore(
	w http.ResponseWriter,
	r *http.Request,
	instance *model.Instance,
	browserVNCEnable bool,
	runScriptFn func(instanceId, script string, timeout uint64, runtimeUser string) (string, error),
	fetchOsNameFn func(ctx context.Context, instanceId string) string,
) {
	jsonAPI(w)

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	if !browserVNCEnable {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgVNCContactAdminToEnable))
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 并行执行：TAT 脚本 + DescribeInstances（获取 OS 名称）
	// TAT 脚本执行 ~3-10s，DescribeInstances ~200-500ms，并行后总耗时取决于 TAT
	type osNameResult struct {
		name string
	}
	osNameCh := make(chan osNameResult, 1)
	if fetchOsNameFn != nil {
		go func() {
			osNameCh <- osNameResult{name: fetchOsNameFn(r.Context(), instance.InstanceId)}
		}()
	} else {
		osNameCh <- osNameResult{}
	}

	output, err := runScriptFn(instance.InstanceId, "check_browser_vnc.sh", 30, instance.RuntimeUser)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCheckVNCEnvFailed))
		return
	}

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgParseCheckResultFailed))
		return
	}

	// 获取并行查询的 OS 名称结果
	osResult := <-osNameCh
	if osResult.name != "" {
		result["os_name"] = osResult.name
	}

	jsonOK(w, map[string]interface{}{
		"ok":   true,
		"data": result,
	})
}
