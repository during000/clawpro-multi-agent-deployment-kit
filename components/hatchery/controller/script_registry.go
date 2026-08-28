package controller

import (
	"context"
	"regexp"
	"sync"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// inlineScriptRegistry 是一个线程安全的临时脚本注册表。
// key 为短脚本名称，value 为脚本内容。
// 用于在 Go 侧完成参数替换后，将脚本内容以短名称注册，
// 供 loadScript 查找，从而绕过 TAT 服务端的参数替换限制。
var (
	inlineScriptMu       sync.RWMutex
	inlineScriptRegistry = make(map[string]string)
)

// RegisterInlineScript 注册一个内联脚本，返回注册时使用的名称。
// 调用方负责在使用完毕后调用 UnregisterInlineScript 清理。
func RegisterInlineScript(name, content string) {
	inlineScriptMu.Lock()
	inlineScriptRegistry[name] = content
	inlineScriptMu.Unlock()
}

// UnregisterInlineScript 注销一个内联脚本。
func UnregisterInlineScript(name string) {
	inlineScriptMu.Lock()
	delete(inlineScriptRegistry, name)
	inlineScriptMu.Unlock()
}

// LookupInlineScript 查找内联脚本，未找到时返回 ("", false)。
func LookupInlineScript(name string) (string, bool) {
	inlineScriptMu.RLock()
	content, ok := inlineScriptRegistry[name]
	inlineScriptMu.RUnlock()
	return content, ok
}

// ========== v7 新增：脚本分派表 + ResolveScript + ExpandIncludes ==========

// scriptResolveTable 将逻辑 feature 名映射为特定 agentType 下的脚本文件名。
// 设计原则：
//   - 业务层只写 feature 名，不 hardcode 脚本文件名；
//   - agentType 不支持的 feature 返回 error（由 ResolveScript 语义保证 fail-closed）。
//
// 命名约定：openclaw 版为基线文件名；hermes/ace 版以 `_hermes` / `_ace` 后缀区分。
var scriptResolveTable = map[string]map[string]string{
	"set_model": {
		model.AgentTypeOpenClaw:     "set_model.sh",
		model.AgentTypeHermes:       "set_model_hermes.sh",
		model.AgentTypeLightclawACE: "set_model_ace.sh",
	},
	"add_skill": {
		model.AgentTypeOpenClaw:     "add_skill.sh",
		model.AgentTypeHermes:       "add_skill_hermes.sh",
		model.AgentTypeLightclawACE: "add_skill_ace.sh",
	},
	"batch_install_skills": {
		model.AgentTypeOpenClaw:     "batch_install_skills_from_smh.sh",
		model.AgentTypeHermes:       "batch_install_skills_from_smh_hermes.sh",
		model.AgentTypeLightclawACE: "batch_install_skills_from_smh_ace.sh",
	},
	"install_skill_from_smh": {
		model.AgentTypeOpenClaw:     "install_skill_from_smh.sh",
		model.AgentTypeHermes:       "install_skill_from_smh_hermes.sh",
		model.AgentTypeLightclawACE: "install_skill_from_smh_ace.sh",
	},
	"uninstall_skill": {
		model.AgentTypeOpenClaw:     "uninstall_skill.sh",
		model.AgentTypeHermes:       "uninstall_skill_hermes.sh",
		model.AgentTypeLightclawACE: "uninstall_skill_ace.sh",
	},
	"uninstall_plugin": {
		model.AgentTypeOpenClaw: "uninstall_plugin.sh",
	},
	"restart_gateway": {
		model.AgentTypeOpenClaw:     "restart_gateway.sh",
		model.AgentTypeHermes:       "restart_gateway_hermes.sh",
		model.AgentTypeLightclawACE: "restart_gateway_ace.sh",
	},
	"set_channel": {
		model.AgentTypeOpenClaw:     "set_channel.sh",
		model.AgentTypeHermes:       "set_channel_hermes.sh",
		model.AgentTypeLightclawACE: "set_channel_ace.sh",
	},
	"del_channel": {
		model.AgentTypeOpenClaw:     "del_channel.sh",
		model.AgentTypeHermes:       "del_channel_hermes.sh",
		model.AgentTypeLightclawACE: "del_channel_ace.sh",
	},
	"whatsapp_pairing": {
		model.AgentTypeOpenClaw: "set_channel_whatsapp.sh",
		// hermes/ace 不支持 WhatsApp 配对码模式
	},
	"del_whatsapp_channel": {
		model.AgentTypeOpenClaw: "del_channel_whatsapp.sh",
		// hermes/ace 不支持 WhatsApp 配对码模式
	},
	"list_channels": {
		model.AgentTypeOpenClaw:     "list_channels.sh",
		model.AgentTypeHermes:       "list_channels_hermes.sh",
		model.AgentTypeLightclawACE: "list_channels_ace.sh",
	},
	"check_ready": {
		model.AgentTypeOpenClaw:     "check_openclaw_ready.sh",
		model.AgentTypeHermes:       "check_hermes_ready.sh",
		model.AgentTypeLightclawACE: "check_ace_ready.sh",
	},
	"weixin_bot_creator": {
		model.AgentTypeOpenClaw:     "weixin_bot_creator.sh",
		model.AgentTypeHermes:       "weixin_bot_creator_hermes.sh",
		model.AgentTypeLightclawACE: "weixin_bot_creator_ace.sh",
	},
	"feishu_bot_creator": {
		model.AgentTypeOpenClaw:     "feishu_bot_creator.sh",
		model.AgentTypeHermes:       "feishu_bot_creator_hermes.sh",
		model.AgentTypeLightclawACE: "feishu_bot_creator_ace.sh", // final §8.2：薄 wrapper 调用 ACE Python 脚本
	},
	"whatsapp_bot_creator": {
		model.AgentTypeOpenClaw: "whatsapp_bot_creator.sh",
	},
	"qq_bot_creator": {
		model.AgentTypeOpenClaw: "qq_bot_creator.sh",
		// hermes/ace 不做 QQ 自动扫码（deprecated 路径），由 ResolveScript 自动返回 error → 400
	},
	// SMH skill 相关的3个脚本
	// 这里直接全都使用同一个脚本，在脚本里对不同的 agentType 进行区分
	"init_smh_env": {
		model.AgentTypeOpenClaw:     "init_smh_env.sh",
		model.AgentTypeHermes:       "init_smh_env.sh",
		model.AgentTypeLightclawACE: "init_smh_env.sh",
	},
	"remove_smh_env": {
		model.AgentTypeOpenClaw:     "remove_smh_env.sh",
		model.AgentTypeHermes:       "remove_smh_env.sh",
		model.AgentTypeLightclawACE: "remove_smh_env.sh",
	},
	"set_smh_token": {
		model.AgentTypeOpenClaw:     "set_smh_token.sh",
		model.AgentTypeHermes:       "set_smh_token.sh",
		model.AgentTypeLightclawACE: "set_smh_token.sh",
	},
	// final：版本同步脚本按 agent_type 分派。
	// - openclaw：硬编码类型+读 ~/.openclaw/* 目录
	// - hermes：通过 hermes --version + harness skills list 获取
	// - ace：通过 lightclaw --version + skills list 获取
	// 历史：原来 version_fetcher.go 硬编码 "get_version_info.sh"，
	// 导致 ACE/Hermes 实例跑 openclaw 脚本拿到空版本、错 agent_type。
	"get_version_info": {
		model.AgentTypeOpenClaw:     "get_version_info.sh",
		model.AgentTypeHermes:       "get_version_info_hermes.sh",
		model.AgentTypeLightclawACE: "get_version_info_ace.sh",
	},
	// final：list_skills 按类型分派
	// - openclaw：openclaw skills list --json
	// - hermes：harness skills list（兼容多种 JSON 输出模式）
	// - ace：解析 lightclaw skills list 表格文本（CLI 暂无 --json）
	"list_skills": {
		model.AgentTypeOpenClaw:     "list_skills.sh",
		model.AgentTypeHermes:       "list_skills_hermes.sh",
		model.AgentTypeLightclawACE: "list_skills_ace.sh",
	},
	// final：环境变量管理按类型分派
	// - openclaw：写 systemd user drop-in (openclaw-gateway.service.d)
	// - hermes：优先 harness env set，兜底 systemd user drop-in (hermes.service.d)
	// - ace：lightclaw env set / list / delete
	"set_env": {
		model.AgentTypeOpenClaw:     "set_env.sh",
		model.AgentTypeHermes:       "set_env_hermes.sh",
		model.AgentTypeLightclawACE: "set_env_ace.sh",
	},
	"get_env": {
		model.AgentTypeOpenClaw:     "get_env.sh",
		model.AgentTypeHermes:       "get_env_hermes.sh",
		model.AgentTypeLightclawACE: "get_env_ace.sh",
	},
	"check_multi_agent": {
		model.AgentTypeOpenClaw: "check_multi_agent.sh",
	},
	"extract_migration_models": {
		model.AgentTypeOpenClaw:     "extract_migration_models.sh",
		model.AgentTypeHermes:       "extract_migration_models_hermes.sh",
		model.AgentTypeLightclawACE: "extract_migration_models_ace.sh",
	},
	// final：check_service 按类型分派（实例状态诊断）
	// - openclaw：openclaw status --json
	// - hermes：harness gateway status + harness channel list
	// - ace：lightclaw status + 解析 lightclaw.json
	"check_service": {
		model.AgentTypeOpenClaw:     "check_service.sh",
		model.AgentTypeHermes:       "check_service_hermes.sh",
		model.AgentTypeLightclawACE: "check_service_ace.sh",
	},
	// final：安装探测脚本按类型分派。原 detect_openclaw_install.sh 硬编码检测
	// openclaw 目录，对 hermes/ace 实例跑它会返回 openclaw_bin="" 的误导性结果；
	// 三端各自检测自己的 CLI / config / 服务状态，顶层字段名保持一致以便 Go 侧
	// detectAndSaveRuntimeUser / HandleAdminDetectInstall 复用解析逻辑。
	"detect_install": {
		model.AgentTypeOpenClaw:     "detect_openclaw_install.sh",
		model.AgentTypeHermes:       "detect_hermes_install.sh",
		model.AgentTypeLightclawACE: "detect_ace_install.sh",
	},
	"set_soul": {
		model.AgentTypeOpenClaw:     "set_soul.sh",
		model.AgentTypeHermes:       "set_soul_hermes.sh",
		model.AgentTypeLightclawACE: "set_soul_ace.sh",
	},
	"remove_soul": {
		model.AgentTypeOpenClaw:     "remove_soul.sh",
		model.AgentTypeHermes:       "remove_soul_hermes.sh",
		model.AgentTypeLightclawACE: "remove_soul_ace.sh",
	},
	// 系统 cloud-init UserData：每个 agent_type 一份独立脚本，避免在一份脚本里
	// 用 case 分支同时覆盖所有类型（导致每次 UserData 里都要塞上所有分支文本，
	// 挤压 16KB 限制下的用户 user_data 可用空间）。
	"init": {
		model.AgentTypeOpenClaw:     "init.sh",
		model.AgentTypeHermes:       "init_hermes.sh",
		model.AgentTypeLightclawACE: "init_ace.sh",
	},
	// 一键升级链路按 agent_type 分派备份/恢复脚本：
	//   - openclaw：硬编码处理 ~/.openclaw 状态目录、plugin 镜像合并、installs.json 修复等
	//   - hermes：覆盖式恢复 ~/.hermes/，依赖 acli/harness gateway start/stop 控制服务
	//   - ace：暂未实现一键升级（SupportsUpgrade=false），不注册即可在 ResolveScript fail-closed
	// 注意：ACE 后续若实现升级，请在此处补 backup_pre_reinstall_ace.sh / restore_post_reinstall_ace.sh
	"backup_pre_reinstall": {
		model.AgentTypeOpenClaw: "backup_pre_reinstall.sh",
		model.AgentTypeHermes:   "backup_pre_reinstall_hermes.sh",
	},
	"restore_post_reinstall": {
		model.AgentTypeOpenClaw: "restore_post_reinstall.sh",
		model.AgentTypeHermes:   "restore_post_reinstall_hermes.sh",
	},
	// cleanup_upgrade_temp：升级临时文件清理，按 agent_type 分派（ace 暂未实现，不注册）。
	"cleanup_upgrade_temp": {
		model.AgentTypeOpenClaw: "cleanup_upgrade_temp.sh",
		model.AgentTypeHermes:   "cleanup_upgrade_temp_hermes.sh",
	},
}

// ResolveScript 根据 feature + agentType 查找真实脚本文件名。
//
// 语义：
//   - agentType == "" 时视为 openclaw（兼容存量数据）；
//   - feature 不存在 → 返回 "unknown feature" 错误；
//   - feature 存在但该 agentType 未配置 → 返回 "not supported" 错误（fail-closed）。
func ResolveScript(ctx context.Context, feature, agentType string) (string, error) {
	runtimeType := model.GetAgentRuntimeType(ctx, agentType)
	if runtimeType == "" {
		return "", hcommon.I18nError(i18n.MsgFeatureNotSupportedForAgentType, feature, agentType)
	}
	m, ok := scriptResolveTable[feature]
	if !ok {
		return "", hcommon.I18nError(i18n.MsgUnknownFeature, feature)
	}
	name, ok := m[runtimeType]
	if !ok {
		return "", hcommon.I18nError(i18n.MsgFeatureNotSupportedForAgentType, feature, agentType)
	}
	return name, nil
}

// ========== v8：feature + instance → TAT 执行 的统一封装 ==========
//
// 【背景】controller 里大量出现这个三段式模板：
//   scriptName, err := ResolveScript(feature, instance.AgentType)
//   if err != nil { writeError/日志... return }
//   if _, err := RunScript(instance.InstanceId, scriptName, timeout, instance.RuntimeUser, nil, params); err != nil {
//       writeError/日志... return
//   }
// 重复 30+ 处。RunAgentScript 把这段收敛为一次调用，调用方仅需关心"失败时
// 是返回 HTTP 错误 / 打警告日志 / 重试"等业务策略，而不用关心脚本解析细节。
//
// 【为什么不在入口 WriteError？】封装不感知 HTTP：
//   - 有些调用方是后台 goroutine（如 injectDefaultModel），根本没有 w/r；
//   - 有些调用方对 Resolve 失败的响应码语义不同（400 vs 500 vs "静默跳过"）；
//   - 让调用方拿到 sentinel error 自己决定，比封装里硬编码 HTTP 码更灵活。
//
// 【错误分类】通过 errors.Is 区分两类失败，调用方可精准映射：
//   - errors.Is(err, ErrScriptResolveFailed)：feature/agent_type 组合未注册脚本，
//     典型处理：HTTP 400 或 goroutine 中静默 return + warning 日志；
//   - errors.Is(err, ErrScriptRunFailed)：TAT 下发或脚本执行失败，
//     典型处理：HTTP 500 + 失败通知；RunScript 产生的原始 RichError 通过 Unwrap 保留。
//
// 【不改动】
//   - 原 RunScript / ResolveScript 行为零调整，保留其独立调用能力；
//   - version_fetcher.go 的 runScriptFn mock 接口不动（其 mock 粒度在 RunScript 层）；
//   - check_ready 等需要在 Resolve 失败时走特殊响应路径的调用不用此函数。

// ErrScriptResolveFailed 表示 ResolveScript 阶段失败（feature/agent_type 未注册）。
// 调用方通常映射为 HTTP 400 或"理论不可达"警告。
var ErrScriptResolveFailed = hcommon.I18nError(i18n.MsgScriptResolveFailed)

// ErrScriptRunFailed 表示 TAT 下发或脚本执行阶段失败。
// 调用方通常映射为 HTTP 500 + RichError Detail + 失败通知。
var ErrScriptRunFailed = hcommon.I18nError(i18n.MsgScriptRunFailed)

// agentScriptRunner 便于单元测试 mock 的函数变量。默认绑定真实 RunScript。
// 生产路径上不直接替换；测试路径 t.Cleanup 还原。
var agentScriptRunner = RunScript

// RunAgentScript 按 instance.AgentType 分派 feature 对应的脚本并下发执行。
//
// 参数：
//   - instance：目标实例（取 InstanceId / AgentType / RuntimeUser，调用方应保证非 nil）；
//   - feature：逻辑能力名（与 scriptResolveTable 一致，如 "set_model" / "set_channel"）；
//   - timeout：TAT 脚本执行超时秒数；
//   - onOutput：流式输出回调，传 nil 表示不需要流式；
//   - params：脚本 {{key}} 占位符参数，传 nil 表示无参数。
//
// 返回：
//   - output：脚本 stdout（与 RunScript 一致）；
//   - err：nil / ErrScriptResolveFailed 包装 / ErrScriptRunFailed 包装。
//     调用方用 errors.Is 区分；用 Unwrap 或 errors.As 取原始错误详情（含 RichError）。
func RunAgentScript(
	ctx context.Context,
	instance *model.Instance,
	feature string,
	timeout uint64,
	onOutput func(chunk string),
	params map[string]string,
) (string, error) {
	if instance == nil {
		return "", hcommon.I18nRichError(ErrScriptResolveFailed, i18n.MsgInstanceIsNil)
	}

	scriptName, rerr := ResolveScript(ctx, feature, instance.AgentType)
	if rerr != nil {
		return "", hcommon.I18nError(i18n.MsgScriptResolveFailedWrap, feature, instance.AgentType).
			WithCause(ErrScriptResolveFailed).WithCause(rerr)
	}

	output, err := agentScriptRunner(ctx, instance.InstanceId, scriptName, timeout, instance.RuntimeUser, onOutput, params)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgScriptRunFailedWrap, scriptName).
			WithCause(ErrScriptRunFailed).WithCause(err)
	}
	return output, nil
}

// includeDirectiveRe 匹配 `# %INCLUDE% lib_xxx.sh` 行首指令。
// 仅匹配严格行首（^）的 `# %INCLUDE% ` 开头，避免误伤脚本中的普通注释。
var includeDirectiveRe = regexp.MustCompile(`(?m)^# %INCLUDE% (\S+)\s*$`)

// libNameRe 校验 lib 文件名白名单：仅允许 `lib_` 前缀 + `[a-z0-9_]` + `.sh` 后缀，
// 避免目录穿越（如 `../etc/passwd`）或异常文件名。
var libNameRe = regexp.MustCompile(`^lib_[a-z0-9_]+\.sh$`)

// ScriptLoader 定义脚本加载器签名。由 main 包通过 controller.LoadScript 注入，
// 解耦 controller 与 main 包的资源加载细节。
type ScriptLoader func(name string) (string, error)

// ExpandIncludes 递归解析脚本中的 `# %INCLUDE% lib_*.sh` 指令，
// 将 lib 文件内容原地内联替换。用于 TAT 远端投递场景：远端 CVM 只拿到一个自包含脚本。
//
// 防护：
//   - libName 必须匹配白名单正则（`lib_*.sh`）；
//   - 递归深度 > 4 报错，避免循环引用死递归；
//   - 倒序替换防止前序替换影响后序 match index。
func ExpandIncludes(content string, loader ScriptLoader) (string, error) {
	return expandIncludesDepth(content, loader, 0)
}

func expandIncludesDepth(content string, loader ScriptLoader, depth int) (string, error) {
	if depth > 4 {
		return "", hcommon.I18nError(i18n.MsgIncludeDepthExceeded)
	}
	matches := includeDirectiveRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}
	// 倒序替换：先替换靠后的 include，避免修改字符串长度后影响前面 match 的偏移量
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		libName := content[m[2]:m[3]]
		if !libNameRe.MatchString(libName) {
			return "", hcommon.I18nError(i18n.MsgInvalidIncludeName, libName)
		}
		libBody, loadErr := loader(libName)
		if loadErr != nil {
			return "", hcommon.I18nRichError(loadErr, i18n.MsgIncludeLoadFailed, libName)
		}
		libBody, richErr := expandIncludesDepth(libBody, loader, depth+1)
		if richErr != nil {
			return "", richErr
		}
		content = content[:m[0]] +
			"\n# ---- inlined from " + libName + " ----\n" +
			libBody +
			"\n# ---- end " + libName + " ----\n" +
			content[m[1]:]
	}
	return content, nil
}
