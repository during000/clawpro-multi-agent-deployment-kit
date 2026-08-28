package task

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// CheckResult Agent 检测结果
type CheckResult int

const (
	CheckReady    CheckResult = iota // 确认就绪
	CheckNotReady                    // 确认未就绪
	CheckUnknown                     // 检测失败（API 超时等），跳过不阻塞
)

// AgentChecker 定义 Agent 检测器接口
type AgentChecker interface {
	Name() string
	Check(ctx context.Context, instanceIds []string) (map[string]CheckResult, error)
}

// ======== 已注册的 checker ========
var agentCheckers []AgentChecker
var agentCheckersMu sync.RWMutex

// RegisterAgentChecker 注册 Agent 检测器
func RegisterAgentChecker(c AgentChecker) {
	agentCheckersMu.Lock()
	defer agentCheckersMu.Unlock()
	agentCheckers = append(agentCheckers, c)
}

// GetAgentCheckers 获取已注册的检测器列表
func GetAgentCheckers() []AgentChecker {
	agentCheckersMu.RLock()
	defer agentCheckersMu.RUnlock()
	result := make([]AgentChecker, len(agentCheckers))
	copy(result, agentCheckers)
	return result
}

// ======== TAT Checker 实现 ========

// TATChecker TAT Agent 检测器
type TATChecker struct{}

func (c *TATChecker) Name() string {
	return "tat"
}

func (c *TATChecker) Check(ctx context.Context, instanceIds []string) (map[string]CheckResult, error) {
	if len(instanceIds) == 0 {
		return map[string]CheckResult{}, nil
	}

	// 复用已有的 TAT 客户端创建函数
	client, err := controller.NewTATClient(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateTATClientFailed)
	}

	// 调用 DescribeAutomationAgentStatus
	req := tat.NewDescribeAutomationAgentStatusRequest()
	req.InstanceIds = common.StringPtrs(instanceIds)

	resp, err := client.DescribeAutomationAgentStatus(req)
	if err != nil {
		// API 错误，返回 Unknown
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
			controller.Logger(ctx).Warn("[TATChecker] API 错误", "code", sdkErr.GetCode(), "message", sdkErr.Message)
		}
		return nil, err
	}

	// 解析结果
	result := make(map[string]CheckResult)
	if resp.Response != nil && resp.Response.AutomationAgentSet != nil {
		for _, agent := range resp.Response.AutomationAgentSet {
			instanceId := *agent.InstanceId
			status := controller.StrVal(agent.AgentStatus)
			// AgentStatus: Online = 就绪
			if status == "Online" {
				result[instanceId] = CheckReady
			} else {
				result[instanceId] = CheckNotReady
			}
		}
	}

	// 未返回的实例标记为 Unknown
	for _, id := range instanceIds {
		if _, ok := result[id]; !ok {
			result[id] = CheckUnknown
		}
	}

	return result, nil
}

func init() {
	// 注册 TAT Checker
	RegisterAgentChecker(&TATChecker{})

	RegisterTask(TaskDef{
		Name:         "agent-checker",
		Interval:     5 * time.Second,
		RunFunc:      checkAllAgents,
		NeedDistLock: false, // 内部自行 TryLock
		PerTenant:    true,
	})
}

// checkAllAgents 执行一轮 Agent 检测
func checkAllAgents(ctx context.Context) {
	// 每轮检测注入独立的 trace，便于日志追踪
	roundCtx := hcommon.WithTaskTrace(ctx, "agent_checker_round")
	// 整轮共用一个 logger：链路字段（request_id/trace_id/interface）由 roundCtx 注入，
	// 后续 5 处日志全部走 log.Xxx，避免重复调用 controller.Logger(roundCtx)。
	// 注：goroutine 内部因为换了 fetchCtx，链路字段不一样，那里仍单独构造 logger。
	log := controller.Logger(roundCtx)

	// 分布式锁，确保多点部署时同一时刻只有一个节点在执行检测任务
	lock, err := model.TryLock(roundCtx, "agents_status_checker")
	if err != nil {
		log.Debug("[AgentChecker] 拿不到分布式锁，跳过本次检测周期", "error", err)
		return
	}
	defer lock.Release()

	// 1. 查 DB：所有 agent_ready=0 的 CVM 实例
	// 不依赖 last_cvm_state，CVM 未 RUNNING 时 TAT Agent 自然不会 Online，checker 会自行过滤
	// 限制每次最多处理 100 个实例，避免内存问题
	// 额外排除操作刚下发不到 30 秒的实例，避免 CVM API 还没生效时误判 Agent 就绪
	// source='cvm' 守卫：本地 agent 实例不走 TAT 检测路径，不能拿它们的 instance_id
	// （如 local-codebuddy-xxx）去 DescribeAutomationAgentStatus，CVM API 会报 InvalidParameterValue。
	var instances []model.Instance
	if err := model.DB(roundCtx).Where(
		"agent_ready = 0 AND instance_id != '' AND source = ? AND "+
			"(current_operation_updated_at IS NULL OR current_operation_updated_at < ?)",
		model.InstanceSourceCVM,
		time.Now().Add(-30*time.Second),
	).Limit(100).
		Find(&instances).Error; err != nil {
		log.Warn("[AgentChecker] 查询实例失败", "error", err)
		return
	}

	if len(instances) == 0 {
		return
	}

	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		if inst.InstanceId != "" {
			ids = append(ids, inst.InstanceId)
		}
	}

	if len(ids) == 0 {
		return
	}

	// checker 批量查询使用 8s 超时（仅用于 TAT API 调用，不传给异步 goroutine）
	checkCtx, cancel := context.WithTimeout(roundCtx, 8*time.Second)
	defer cancel()

	// 2. 每个 checker 批量查一次
	checkers := GetAgentCheckers()
	checkerResults := make(map[string]map[string]CheckResult) // checkerName → instanceId → result

	for _, checker := range checkers {
		result, err := checker.Check(checkCtx, ids)
		if err != nil {
			log.Warn("[AgentChecker] 检测失败", "checker", checker.Name(), "error", err)
			continue // 整个 checker 失败，跳过
		}
		checkerResults[checker.Name()] = result
	}

	// 3. 合并结果：所有 checker 都 Ready才算就绪
	for _, inst := range instances {
		if inst.InstanceId == "" {
			continue
		}

		allReady := true
		for _, checker := range checkers {
			r, ok := checkerResults[checker.Name()]
			if !ok {
				allReady = false
				break
			}
			if result, exists := r[inst.InstanceId]; exists && result != CheckReady {
				allReady = false
				break
			}
		}

		if allReady {
			now := time.Now()
			updates := map[string]interface{}{
				"agent_ready": 1,
			}

			// 记录当前操作（在 updates 覆盖前保存，用于判断通知类型）
			prevOp := inst.CurrentOperation

			// 如果有 currentOp 且非 delete / upgrade，同步收敛
			// upgrade 的操作锁由 performUpgrade 的 finalizeUpgradeResult 负责清理，
			// 这里不能抢着置 success，否则前端会在升级还在跑 restore_post_reinstall 阶段
			// 就看到实例变成 running/OK，导致"升级中"状态丢失。
			if inst.CurrentOperation != "" &&
				inst.CurrentOperation != model.OpDelete &&
				inst.CurrentOperation != model.OpUpgrade {
				updates["current_operation"] = model.OpNone
				updates["current_operation_state"] = model.OpStateSuccess
				updates["current_operation_updated_at"] = &now
				updates["last_stable_state"] = "RUNNING"
			}

			if err := model.DB(roundCtx).Model(&inst).Updates(updates).Error; err != nil {
				log.Warn("[AgentChecker] 更新实例失败", "id", inst.ID, "error", err)
			} else {
				log.Info("[AgentChecker] Agent 就绪", "id", inst.ID, "checkers", len(checkers))
				// Agent 就绪后异步检测安装用户（仅在 RuntimeUser 为空时执行）
				// final：detect_openclaw_install.sh 是 openclaw 专属探测脚本，
				// 对 hermes/ace 实例跑它会基于 openclaw 路径启发式推断，结果可能
				// 误差（如 ACE 的 agentuser 恰好也有空的 .openclaw 目录）。
				// 这里按 agent_type 分派，避免跨类型污染。
				if inst.RuntimeUser == "" {
					go detectAndSaveRuntimeUser(hcommon.WithTaskTrace(hcommon.DetachContext(roundCtx), "detect_runtime_user"), inst.ID, inst.InstanceId, inst.AgentType)
				}
				// Agent 就绪后立即异步拉取版本信息，避免等待 24h VersionSync 定时任务
				// 使用 DetachContext 脱离 cancel 链，因为版本拉取可能需要 30s+
				go func(i model.Instance) {
					// DetachContext 保留 trace 字段且脱离 cancel 链；日志统一用该 detached ctx，
					// 与下游 FetchAndSaveVersionInfoSync 内部日志在同一 trace_id 下串联。
					fetchCtx := hcommon.DetachContext(roundCtx)
					defer func() {
						if r := recover(); r != nil {
							controller.Logger(fetchCtx).Error("[AgentChecker] 异步版本拉取 panic recovered（将由定时任务兜底）",
								"id", i.ID, "instance_id", i.InstanceId, "error", r)
						}
					}()
					if err := controller.FetchAndSaveVersionInfoSync(fetchCtx, i); err != nil {
						controller.Logger(fetchCtx).Warn("[AgentChecker] Agent 就绪后拉取版本失败（将由定时任务兜底）",
							"id", i.ID, "instance_id", i.InstanceId, "error", err)
					}
				}(inst)
				// 根据收敛前的 currentOp 写入成功通知
				switch prevOp {
				case model.OpCreate:
					notifyCtx := hcommon.DetachContext(roundCtx)
					go model.CreateSuccessNotification(
						notifyCtx, inst.UserID, inst.ID, inst.Name,
						model.NotifyTypeInstanceCreateSuccess,
						i18n.T(notifyCtx, i18n.MsgInstanceCreateSuccessTitle),
						i18n.T(notifyCtx, i18n.MsgInstanceCreateSuccessMessage, inst.Name),
					)
				case model.OpReinstall:
					notifyCtx := hcommon.DetachContext(roundCtx)
					go model.CreateSuccessNotification(
						notifyCtx, inst.UserID, inst.ID, inst.Name,
						model.NotifyTypeInstanceReinstallSuccess,
						i18n.T(notifyCtx, i18n.MsgInstanceReinstallSuccessTitle),
						i18n.T(notifyCtx, i18n.MsgInstanceReinstallSuccessMessage, inst.Name),
					)
				}

				// 自定义镜像安全闭环：用户拿已装好 openclaw 的机器做自定义镜像时，
				// ~/.openclaw/openclaw.json 中的 gateway.auth.token 会被一起打包，
				// 导致同源镜像装出的所有实例 token 完全一致。拿到任一份 openclaw.json
				// 即可访问所有同源实例的 gateway，存在严重安全风险。
				//
				// 此处仅对"新创建（OpCreate）+ openclaw 类型 + 自定义镜像"路径触发
				// 一次性强制轮换。reinstall/upgrade 是同一台实例的延续，token 保持不变。
				// hermes/lightclawace 走各自的 token 体系，不在此处理。
				//
				// 官方公共镜像（hcommon.IsCandidateImage=true）内置 first-boot-token.sh，
				// 首启时已自动随机化 gateway 的 token/port/basePath/allowedOrigins，
				// hatchery 再轮换会与之产生竞态（两侧都在 flock openclaw.json + systemctl
				// restart gateway），且会让官方脚本刚生成的 token 被覆盖一次，影响下游
				// API 网关侧 token 的一致性，故官方镜像直接跳过，由镜像内置脚本接管。
				if prevOp == model.OpCreate && inst.AgentType == model.AgentTypeOpenClaw {
					if hcommon.IsCandidateImage(inst.ImgId) {
						log.Info("[AgentChecker] 官方公共镜像，由镜像内置 first-boot-token.sh 负责首装初始化，跳过 hatchery 侧加固",
							"id", inst.ID, "instance_id", inst.InstanceId, "img_id", inst.ImgId)
					} else {
						log.Info("[AgentChecker] 自定义镜像，触发首装安全初始化",
							"id", inst.ID, "instance_id", inst.InstanceId, "img_id", inst.ImgId)
						go secureFirstBootAsyncFn(hcommon.WithTaskTrace(hcommon.DetachContext(roundCtx), "secure_first_boot"), inst)
					}
				}
			}
		}
	}
}

// detectRuntimeUserMaxAttempts / detectRuntimeUserRetryInterval 是 detectAndSaveRuntimeUser
// 的重试参数。生产默认 3 次重试 / 间隔 5s；单元测试可以在 setup 中缩短间隔以加速测试。
var (
	detectRuntimeUserMaxAttempts   = 3
	detectRuntimeUserRetryInterval = 5 * time.Second
)

// detectAndSaveRuntimeUser 通过 TAT 以 root 下发探测脚本，检测实际安装在哪个用户下，
// 并将结果写回数据库。该函数在 Agent 就绪后异步调用，不阻塞主检测循环。
//
// final：按 agent_type 做分派（走 scriptResolveTable["detect_install"]）
//   - openclaw   → detect_openclaw_install.sh（探测 ~/.openclaw / clawhub / pnpm 等）
//   - hermes     → detect_hermes_install.sh（探测 ~/.hermes / hermes / harness CLI）
//   - lightclawace → detect_ace_install.sh（探测 ~/.lightclaw / lightclaw CLI）
//
// 三端脚本顶层 runtime_user/runtime_home 契约一致，这里用同一份 JSON 解析。
// 探测失败时不写入兜底值（避免写入可能错误的硬编码值污染 DB），保持 runtime_user 为空，
// 后续 ensureRuntimeUser 调用（定时任务等）会再次尝试探测。
func detectAndSaveRuntimeUser(ctx context.Context, instancePK uint, instanceId string, agentType string) {
	// 走 controller.Logger(ctx) 复用 ctx 注入的 request_id/trace_id/interface 等字段，
	// 与 [AgentChecker] 这一轮的其他日志在同一 trace 下串联。
	log := controller.Logger(ctx).With(
		"instance_pk", instancePK,
		"instance_id", instanceId,
		"agent_type", agentType,
	)

	defer func() {
		if r := recover(); r != nil {
			log.Error("[RuntimeUser] panic recovered", "error", r)
		}
	}()

	// 解析当前 agent_type 对应的探测脚本。
	scriptName, resolveErr := controller.ResolveScript(ctx, "detect_install", agentType)
	if resolveErr != nil {
		log.Warn("[RuntimeUser] 未找到探测脚本，跳过", "error", resolveErr)
		return
	}

	log.Info("[RuntimeUser] 开始检测运行用户", "script", scriptName)

	// 带重试执行探测脚本（最多 3 次，间隔 5s），覆盖 TAT Agent 启动延迟或瞬断窗口。
	var output string
	var lastErr error
	for attempt := 1; attempt <= detectRuntimeUserMaxAttempts; attempt++ {
		output, lastErr = controller.RunScript(ctx, instanceId, scriptName, 30, "", nil, nil)
		if lastErr == nil {
			break
		}
		log.Warn("[RuntimeUser] 探测脚本执行失败，重试",
			"attempt", attempt, "max_attempts", detectRuntimeUserMaxAttempts, "error", lastErr)
		if attempt < detectRuntimeUserMaxAttempts {
			time.Sleep(detectRuntimeUserRetryInterval)
		}
	}

	if lastErr != nil {
		// 所有重试均失败：不写入 DB，保持 runtime_user 为空。
		// 后续 ensureRuntimeUser 调用（定时任务、技能安装等）会再次尝试探测。
		log.Error("[RuntimeUser] 探测全部失败，不写入兜底值（保持为空等待后续重试）", "error", lastErr)
		return
	}

	var result struct {
		RuntimeUser string `json:"runtime_user"`
		RuntimeHome string `json:"runtime_home"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		log.Warn("[RuntimeUser] 解析检测结果失败", "output", output, "error", err)
		return
	}

	if result.RuntimeUser == "" || result.RuntimeUser == "unknown" {
		log.Warn("[RuntimeUser] 探测脚本未返回有效 runtime_user")
		return
	}

	// 原子写入：仅当 runtime_user 仍为空时才更新，防止覆盖并发写入（ensureRuntimeUser）的正确值。
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ? AND runtime_user = ''", instancePK).
		Updates(map[string]interface{}{
			"runtime_user": result.RuntimeUser,
			"runtime_home": result.RuntimeHome,
		}).Error; err != nil {
		log.Warn("[RuntimeUser] 回写数据库失败", "error", err)
		return
	}

	log.Info("[RuntimeUser] 检测完成", "user", result.RuntimeUser, "home", result.RuntimeHome)
}

// secureFirstBootMaxAttempts / secureFirstBootRetryInterval 控制 RunScript
// 失败时的重试。生产默认 2 次 / 间隔 5s；单测可在 setup 中缩短。
var (
	secureFirstBootMaxAttempts   = 2
	secureFirstBootRetryInterval = 5 * time.Second
)

// secureFirstBootRunScriptFn 是 controller.RunScript 的可替换包装，便于单测 mock TAT 调用。
// 与 [tdai_handler_switch.go](task/tdai_handler_switch.go) 的 taskRunScriptFn / [recover_hermes_runtime_user.go](task/recover_hermes_runtime_user.go)
// 的 recoverRunScriptFn 同款设计，保持项目一致的 DI 风格。
var secureFirstBootRunScriptFn = controller.RunScript

// secureFirstBootAsyncFn 是 secureFirstBootAsync 的可替换变量。上层
// checkAllAgents 中的 `go secureFirstBootAsync(...)` 是 `go secureFirstBootAsyncFn(...)`，
// 这样单测能 mock 这个变量来验证「是否触发」而不用启动实际的 goroutine。
var secureFirstBootAsyncFn = secureFirstBootAsync

// secureFirstBootAsync 自定义镜像首装后的安全初始化总入口。当前职责是强制轮换
// ~/.openclaw/openclaw.json 里的 gateway.auth.token；脚本 secure_first_boot.sh
// 内部按步骤组织，未来可在脚本侧扩展更多加固动作（basePath/allowedOrigins 随机化、
// SSH host key 重生成等），Go 侧无需改动。
//
// 触发条件：OpCreate + AgentTypeOpenClaw + 非候选镜像（自定义镜像），仅触发一次。
// 触发方：checkAllAgents 在 agent_ready 由 0 翻 1 的瞬间异步 go 出来。
// 该函数所有失败仅落日志，不写 DB、不发通知、不影响业务主流程——本就是安全加固性质的
// 一次性动作，失败的最坏后果就是该实例继续沿用镜像里的旧 token，可由后续手动修复。
func secureFirstBootAsync(ctx context.Context, inst model.Instance) {
	// 走 controller.Logger(ctx) 复用 ctx 注入的 request_id/trace_id/interface 等字段，
	// 与 [AgentChecker] 这一轮的其他日志（detect runtime user / fetch version 等）串联。
	log := controller.Logger(ctx).With(
		"instance_pk", inst.ID,
		"instance_id", inst.InstanceId,
		"agent_type", inst.AgentType,
	)

	defer func() {
		if r := recover(); r != nil {
			log.Error("[SecureFirstBoot] panic recovered", "error", r)
		}
	}()

	log.Info("[SecureFirstBoot] 开始自定义镜像首装安全初始化")

	// 注意：此时 inst.RuntimeUser 大概率仍为空（detectAndSaveRuntimeUser 是与本函数并行 go 出去的）。
	// 这里不刻意等 detect 完成，原因有二：
	//   1. 脚本以 root 身份执行（见 controller.rootRequiredTATScripts），自带 fallback 探测；
	//   2. hatchery 会把 inst.RuntimeUser 注入为 OPENCLAW_RUNTIME_USER（即使是空字符串，
	//      脚本也会 fallback 扫 /home/* 和 /root）。
	var lastErr error
	for attempt := 1; attempt <= secureFirstBootMaxAttempts; attempt++ {
		_, lastErr = secureFirstBootRunScriptFn(ctx, inst.InstanceId, "secure_first_boot.sh", 60, inst.RuntimeUser, nil, nil)
		if lastErr == nil {
			log.Info("[SecureFirstBoot] 安全初始化成功", "attempt", attempt)
			return
		}
		log.Warn("[SecureFirstBoot] 脚本执行失败，将重试",
			"attempt", attempt, "max_attempts", secureFirstBootMaxAttempts, "error", lastErr)
		if attempt < secureFirstBootMaxAttempts {
			time.Sleep(secureFirstBootRetryInterval)
		}
	}
	log.Error("[SecureFirstBoot] 安全初始化失败（已用完所有重试，仅记日志，不影响业务）", "error", lastErr)
}
