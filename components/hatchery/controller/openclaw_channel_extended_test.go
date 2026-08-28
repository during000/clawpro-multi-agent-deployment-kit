package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ─── HandleSetChannel 未覆盖分支 ──────────────────────────────────────────

func TestHandleSetChannel_UnknownAgentTypeFails(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-set-unk",
		UserID: user.ID, AgentType: "totally_unknown",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Set("key", "appid")
	form.Set("value", "12345")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	// 未知 agent_type → checkInstanceSupportsChannel 拦截为 403
	if rr.Code != http.StatusForbidden {
		t.Errorf("未知 agent_type 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetChannel_MissingChannel(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-nochannel",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{} // 无 channel
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 channel 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSetChannel_MissingKV(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// 确保 AIChannel 表已迁移
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-nokv",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu") // openclaw 白名单内
	// 无 key / value
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 key/value 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetChannel_EmptyKVValue(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-emptykv",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Set("key", "appid")
	form.Set("value", "") // 空值
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 value 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleDelChannel ────────────────────────────────────────────────────

func TestHandleDelChannel_MethodNotAllowed(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/channel/del", nil)
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleDelChannel_UnsupportedChannelForHermes(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-hermes-del",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "wecom_app") // hermes 不支持企业微信应用
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel/del?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("hermes 删除白名单外通道应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelChannel_MissingChannel(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-delmiss",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel/del?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 channel 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleDelChannel_UnknownAgentType(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-del-unk",
		UserID: user.ID, AgentType: "future_type_xyz",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel/del?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("未知 agent_type 应返回 403（guard 拦截），实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleChannelsList（无 id 参数，返回所有 channel 列表）────────────────

func TestHandleChannelsList_NoID_ReturnsList(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{}); err != nil {
		t.Skipf("AIChannel migrate failed: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 插入内置 channel
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "feishu", Name: "Feishu"})
	// 插入 custom channel
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID: "custom_x", Name: "Custom X",
		Custom:       true,
		CustomConfig: `{"server":{"a":"b"},"cred_fields":[]}`,
	})

	req := channelReqWithSession(t, http.MethodGet, "/openclaw/channels", "u1", "")
	rr := httptest.NewRecorder()
	HandleChannelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, rr.Body.String())
	}
	if len(items) != 2 {
		t.Errorf("应返回 2 个 channel，实际=%d", len(items))
	}

	// 自定义 channel 的 AgentTypes 应是三端
	for _, it := range items {
		if cid, _ := it["ChannelID"].(string); cid == "custom_x" {
			ats, _ := it["AgentTypes"].([]interface{})
			if len(ats) != 3 {
				t.Errorf("自定义 channel 应返回 3 个 AgentTypes，实际=%d", len(ats))
			}
		}
	}
}

func TestHandleChannelsList_NoID_FiltersOverseasOnlyChannelsBySiteScope(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	i18n.SetDefaultLang("zh")
	defer i18n.SetDefaultLang("zh")
	if err := model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{}); err != nil {
		t.Skipf("AIChannel migrate failed: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "feishu", Name: "Feishu"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "msteams", Name: "Microsoft Teams"})

	req := channelReqWithSession(t, http.MethodGet, "/openclaw/channels", "u1", "")
	rr := httptest.NewRecorder()
	HandleChannelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, rr.Body.String())
	}
	if hasPascalChannel(items, "slack") {
		t.Fatalf("domestic site should hide slack, channels=%v", items)
	}
	if !hasPascalChannel(items, "msteams") {
		t.Fatalf("domestic site should show msteams (all-scope), channels=%v", items)
	}
	if !hasPascalChannel(items, "feishu") {
		t.Fatalf("domestic site should keep feishu, channels=%v", items)
	}

	i18n.SetDefaultLang("en")
	rr = httptest.NewRecorder()
	HandleChannelsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("海外站点应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	items = nil
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("解析海外响应失败: %v, body=%s", err, rr.Body.String())
	}
	for _, ch := range []string{"slack", "msteams"} {
		if !hasPascalChannel(items, ch) {
			t.Fatalf("overseas site should show %s, channels=%v", ch, items)
		}
	}
}

func hasPascalChannel(channels []map[string]interface{}, channelID string) bool {
	for _, ch := range channels {
		if got, _ := ch["ChannelID"].(string); got == channelID {
			return true
		}
	}
	return false
}

func TestHandleChannelsList_WithInstanceID_InstanceNotFound(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// id 存在但实例不存在
	req := channelReqWithSession(t, http.MethodGet, "/openclaw/channels?id=9999", "u1", "")
	rr := httptest.NewRecorder()
	HandleChannelsList(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleChannelsList_Unauthorized(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/channels", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleChannelsList(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

// ─── HandleAutoChannel ──────────────────────────────────────────────────

func TestHandleAutoChannel_UnsupportedChannelType(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-auto-unk",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=unknown_channel", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("不支持的 channel 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAutoChannel_HermesQQBotNotSupported(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-hermes-auto-qq",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	// hermes 不支持 qqbot 自动配置（ResolveScript 会 fail-closed）
	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=qqbot", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("hermes qqbot 自动配置应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAutoChannel_AceWeixinNotSupported(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-ace-auto-wx",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	// ace 不支持 ddingtalk 自动配置
	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=ddingtalk", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("ace ddingtalk 自动配置应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAutoChannel_DefaultChannelQqbot(t *testing.T) {
	// 不传 channel 参数时默认 qqbot（openclaw 支持）
	// 但实际会走到 RunScript → LoadScript（mock 失败），从而不 panic
	cleanup := initChannelTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: not found")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-auto-default",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d", inst.ID), // 无 channel 参数
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	// SSE 响应：预期输出 event: fail（因 RunScript 失败）
	body := rr.Body.String()
	if !strings.Contains(body, "fail") {
		t.Errorf("mock LoadScript 失败应触发 SSE fail，实际 body=%s", body)
	}
}

// ─── normalizeWecomShape（原 renameWecomAgentKey 扩展版） ────────────────────

// TestNormalizeWecomShape_WithAgent 保留原 renameWecomAgentKey 的 agent → wecom_app
// 改名行为；并验证新增的"缺 bot 自动补空对象"语义生效。
func TestNormalizeWecomShape_WithAgent(t *testing.T) {
	in := `{"wecom":{"agent":{"x":"ag_value"},"enabled":true},"feishu":{"enabled":false}}`
	out := normalizeWecomShape(in)

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("输出应为合法 JSON: %v", err)
	}
	wecom, ok := m["wecom"].(map[string]interface{})
	if !ok {
		t.Fatalf("wecom 字段应为对象，实际=%v", m["wecom"])
	}
	if _, exists := wecom["agent"]; exists {
		t.Errorf("agent 字段应被删除，实际仍存在: %v", wecom)
	}
	appVal, _ := wecom["wecom_app"].(map[string]interface{})
	if appVal["x"] != "ag_value" {
		t.Errorf("wecom_app 应从 agent 继承内容，实际=%v", wecom["wecom_app"])
	}
	// 新增：agent 改名为 wecom_app 后仍缺 bot，应被补为空对象
	botVal, hasBot := wecom["bot"].(map[string]interface{})
	if !hasBot {
		t.Errorf("缺 bot 时应自动补空对象，实际 wecom=%v", wecom)
	}
	if len(botVal) != 0 {
		t.Errorf("补齐的 bot 应为空对象，实际=%v", botVal)
	}
}

func TestNormalizeWecomShape_NoWecom(t *testing.T) {
	// 完全没有 wecom 字段（未配置实例）→ 保持不存在，不无中生有
	in := `{"feishu":{"enabled":true}}`
	out := normalizeWecomShape(in)
	if out != in {
		t.Errorf("无 wecom 字段应原样返回，实际=%s", out)
	}
}

// TestNormalizeWecomShape_BotOnly 覆盖核心需求场景：
// Hermes/ACE 实例只配置了 wecom bot（底层无 app 能力），接口应补齐 wecom_app={}，
// 使返回结构与 openclaw 对齐，前端免去按 agent_type 走不同渲染分支。
func TestNormalizeWecomShape_BotOnly(t *testing.T) {
	in := `{"wecom":{"bot":{"botId":"aibNIkwI","secret":"xxx"},"enabled":true}}`
	out := normalizeWecomShape(in)

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("输出应为合法 JSON: %v", err)
	}
	wecom := m["wecom"].(map[string]interface{})

	// bot 原样保留
	bot, ok := wecom["bot"].(map[string]interface{})
	if !ok {
		t.Fatalf("bot 字段应保留为对象，实际=%v", wecom["bot"])
	}
	if bot["botId"] != "aibNIkwI" {
		t.Errorf("bot 内容应原样保留，实际=%v", bot)
	}

	// wecom_app 应被补为空对象（核心断言）
	appVal, hasApp := wecom["wecom_app"].(map[string]interface{})
	if !hasApp {
		t.Fatalf("hermes/ace 缺 wecom_app 时应自动补空对象，实际 wecom=%v", wecom)
	}
	if len(appVal) != 0 {
		t.Errorf("补齐的 wecom_app 应为空对象 {}，实际=%v", appVal)
	}

	// enabled 原样保留
	if wecom["enabled"] != true {
		t.Errorf("enabled 应原样保留 true，实际=%v", wecom["enabled"])
	}
}

// TestNormalizeWecomShape_AppOnly 对称覆盖：某些场景只有 wecom_app 无 bot，
// 应补齐 bot={}。这条路径目前在线上极少命中（openclaw 一般 bot 先配置），但
// 作为"对称补齐"契约的兜底，避免未来新脚本输出形态时缺口不一致。
func TestNormalizeWecomShape_AppOnly(t *testing.T) {
	in := `{"wecom":{"wecom_app":{"agentId":"1000001"},"enabled":true}}`
	out := normalizeWecomShape(in)

	var m map[string]interface{}
	json.Unmarshal([]byte(out), &m)
	wecom := m["wecom"].(map[string]interface{})

	bot, hasBot := wecom["bot"].(map[string]interface{})
	if !hasBot {
		t.Fatalf("缺 bot 应补空对象，实际=%v", wecom)
	}
	if len(bot) != 0 {
		t.Errorf("补齐的 bot 应为空对象 {}，实际=%v", bot)
	}
	// wecom_app 内容原样
	app := wecom["wecom_app"].(map[string]interface{})
	if app["agentId"] != "1000001" {
		t.Errorf("wecom_app 内容应原样保留，实际=%v", app)
	}
}

// TestNormalizeWecomShape_BotAndAppAlreadyPresent 已经 shape 对齐的输入（两者都有）
// 不应做任何改动，返回等价 JSON。
func TestNormalizeWecomShape_BotAndAppAlreadyPresent(t *testing.T) {
	in := `{"wecom":{"bot":{"a":1},"wecom_app":{"b":2},"enabled":true}}`
	out := normalizeWecomShape(in)

	// 允许字段顺序差异，但内容必须等价
	var origMap, outMap map[string]interface{}
	json.Unmarshal([]byte(in), &origMap)
	json.Unmarshal([]byte(out), &outMap)

	origWecom := origMap["wecom"].(map[string]interface{})
	outWecom := outMap["wecom"].(map[string]interface{})
	if len(origWecom) != len(outWecom) {
		t.Errorf("字段数应一致，原=%d 新=%d", len(origWecom), len(outWecom))
	}
	for k, v := range origWecom {
		if fmt.Sprintf("%v", outWecom[k]) != fmt.Sprintf("%v", v) {
			t.Errorf("字段 %s 值不一致，原=%v 新=%v", k, v, outWecom[k])
		}
	}
}

// TestNormalizeWecomShape_EmptyWecomObject 覆盖边界：wecom 是空对象 `{}`
// —— 应补齐出 bot={}、wecom_app={} 两个空对象。
func TestNormalizeWecomShape_EmptyWecomObject(t *testing.T) {
	in := `{"wecom":{}}`
	out := normalizeWecomShape(in)

	var m map[string]interface{}
	json.Unmarshal([]byte(out), &m)
	wecom := m["wecom"].(map[string]interface{})

	if _, ok := wecom["bot"].(map[string]interface{}); !ok {
		t.Errorf("空 wecom 应补 bot={}，实际=%v", wecom)
	}
	if _, ok := wecom["wecom_app"].(map[string]interface{}); !ok {
		t.Errorf("空 wecom 应补 wecom_app={}，实际=%v", wecom)
	}
}

func TestNormalizeWecomShape_InvalidJSON(t *testing.T) {
	in := `not a json`
	out := normalizeWecomShape(in)
	if out != in {
		t.Errorf("非法 JSON 应原样返回，实际=%s", out)
	}
}

func TestNormalizeWecomShape_WecomNotObject(t *testing.T) {
	// wecom 字段不是对象（字符串/数字/null）→ 原样返回，不抛错
	in := `{"wecom":"not-an-object"}`
	out := normalizeWecomShape(in)
	if out != in {
		t.Errorf("wecom 非对象应原样返回，实际=%s", out)
	}
}

// ─── listInstanceChannels ────────────────────────────────────────────────

func TestListInstanceChannels_UnknownAgentType(t *testing.T) {
	// 未知 agent_type → ResolveScript 失败
	inst := &model.Instance{InstanceId: "ins-xxx", RuntimeUser: "agentuser", AgentType: "future_unknown_type"}
	_, err := listInstanceChannels(context.Background(), inst)
	if err == nil {
		t.Error("未知 agent_type 应返回错误")
	}
	if !strings.Contains(err.Error(), "list_channels") {
		t.Errorf("错误消息应提到 list_channels，实际=%v", err)
	}
}

func TestListInstanceChannels_Success(t *testing.T) {
	origRunner := listChannelsScriptRunner
	listChannelsScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"feishu":{"enabled":true},"wecom":{"agent":{"corpid":"xxx"}}}`, nil
	}
	defer func() { listChannelsScriptRunner = origRunner }()

	inst := &model.Instance{InstanceId: "ins-ok", RuntimeUser: "root", AgentType: "openclaw"}
	channels, err := listInstanceChannels(context.Background(), inst)
	if err != nil {
		t.Fatalf("不应报错，实际=%v", err)
	}
	// normalizeWecomShape 应将 agent 改名为 wecom_app
	wecom, ok := channels["wecom"].(map[string]interface{})
	if !ok {
		t.Fatal("wecom 应为 map")
	}
	if _, has := wecom["agent"]; has {
		t.Error("agent 字段应被 normalizeWecomShape 改名为 wecom_app")
	}
	if _, has := wecom["wecom_app"]; !has {
		t.Error("应有 wecom_app 字段")
	}
}

func TestListInstanceChannels_InvalidJSON(t *testing.T) {
	origRunner := listChannelsScriptRunner
	listChannelsScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "not json at all", nil
	}
	defer func() { listChannelsScriptRunner = origRunner }()

	inst := &model.Instance{InstanceId: "ins-bad", RuntimeUser: "root", AgentType: "openclaw"}
	_, err := listInstanceChannels(context.Background(), inst)
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
	if !strings.Contains(err.Error(), "解析通道列表失败") {
		t.Errorf("错误消息应包含'解析通道列表失败'，实际=%v", err)
	}
}

// ─── HandleSetChannel custom channel 分支 ──────────────────────────────

func TestHandleSetChannel_CustomChannelParseFails(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-custom-fail",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 插入一条自定义 channel，但 custom_config 是无效的 JSON
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "mybadcustom",
		Name:         "Bad Custom",
		Custom:       true,
		CustomConfig: "not-a-valid-json",
	})

	form := url.Values{}
	form.Set("channel", "mybadcustom")
	form.Set("key", "k")
	form.Set("value", "v")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	// ParseCustomConfig 失败 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("自定义配置解析失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetChannel_CustomChannelMergedConfig(t *testing.T) {
	// 触发源码 102-133 的 custom channel 成功路径（LoadScript mock 失败 → 500）
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script-load-fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-custom-ok",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 合法的自定义 channel 配置
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "mycustom",
		Name:         "My Custom",
		Custom:       true,
		CustomConfig: `{"server":{"endpoint":"https://example.com","api_key":"srv-key"},"cred_fields":[{"key":"user_token","label":"User Token"}]}`,
	})

	form := url.Values{}
	form.Set("channel", "mycustom")
	form.Set("key", "user_token")
	form.Set("value", "abc-user-value")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	// Custom 路径通过 → ResolveScript 成功 → RunScript → LoadScript 失败 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("LoadScript mock 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 等待异步 createErrorNotification goroutine
	// （数据库操作非常快，这里仅防御式等待）
	waitForGoroutines()
}

func TestHandleDelChannel_CustomChannel(t *testing.T) {
	// 覆盖 HandleDelChannel 中 custom channel 分支
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: del script fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-del-custom",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "mycustom_del",
		Name:         "My Custom Del",
		Custom:       true,
		CustomConfig: `{"server":{"a":"b"},"cred_fields":[]}`,
	})

	form := url.Values{}
	form.Set("channel", "mycustom_del")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel/del?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)

	// 自定义 channel 不受白名单约束 → ResolveScript 成功 → RunScript → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("del channel mock 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── normalizeWecomShape：ACE 真实 lightclaw.json 平铺格式端到端测试 ────────

// TestNormalizeWecomShape_ACEFlatFormat 使用 ACE 实例真实的 lightclaw.json wecom
// 平铺格式作为输入，验证 Go 层兜底归一化（模拟 list_channels_ace.sh jq 失败时
// 平铺格式直接透传到 Go 层的场景）能正确将 bot 级字段搬运到 .bot 子对象。
func TestNormalizeWecomShape_ACEFlatFormat(t *testing.T) {
	// 真实 lightclaw.json 中 wecom 的平铺格式（来自用户实际配置）
	in := `{"wecom":{"enabled":true,"botPrefix":"","botId":"aibcBbX0sHQ0sJkfJKeV-HDRMLV0MHYiDfh","secret":"Ab9eAZL1oq8hSUOf8RYSaoc6Holw7nRmcZP9Zup7u6T","name":"{{bot_name}}","websocketUrl":"","dmPolicy":"open","allowFrom":[],"groupPolicy":"open","groupAllowFrom":[],"groups":{},"sendThinkingMessage":true,"mediaDir":"~/.lightclaw/media","mediaLocalRoots":[]}}`
	out := normalizeWecomShape(in)

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("输出应为合法 JSON: %v", err)
	}
	wecom, ok := m["wecom"].(map[string]interface{})
	if !ok {
		t.Fatalf("wecom 字段应为对象，实际=%v", m["wecom"])
	}

	// ① bot 子对象应包含所有 bot 级字段
	bot, hasBotObj := wecom["bot"].(map[string]interface{})
	if !hasBotObj {
		t.Fatalf("应生成 bot 子对象，实际 wecom=%v", wecom)
	}
	expectedBotFields := map[string]interface{}{
		"botId":        "aibcBbX0sHQ0sJkfJKeV-HDRMLV0MHYiDfh",
		"secret":       "Ab9eAZL1oq8hSUOf8RYSaoc6Holw7nRmcZP9Zup7u6T",
		"name":         "{{bot_name}}",
		"websocketUrl": "",
		"dmPolicy":     "open",
	}
	for k, expected := range expectedBotFields {
		if fmt.Sprintf("%v", bot[k]) != fmt.Sprintf("%v", expected) {
			t.Errorf("bot.%s 应为 %v，实际=%v", k, expected, bot[k])
		}
	}
	// allowFrom 是数组，单独检查
	if _, hasAllowFrom := bot["allowFrom"]; !hasAllowFrom {
		t.Errorf("bot.allowFrom 应存在，实际 bot=%v", bot)
	}

	// ② wecom 根下不应残留 bot 级字段
	for _, k := range []string{"botId", "secret", "name", "websocketUrl", "dmPolicy", "allowFrom"} {
		if _, exists := wecom[k]; exists {
			t.Errorf("wecom 根下不应残留 bot 级字段 %s，实际 wecom=%v", k, wecom)
		}
	}

	// ③ 通道级字段应保留在 wecom 根下
	channelFields := []string{"enabled", "botPrefix", "groupPolicy", "groupAllowFrom",
		"groups", "sendThinkingMessage", "mediaDir", "mediaLocalRoots"}
	for _, k := range channelFields {
		if _, exists := wecom[k]; !exists {
			t.Errorf("通道级字段 %s 应保留在 wecom 根下，实际 wecom=%v", k, wecom)
		}
	}

	// ④ wecom_app 应被补为空对象
	appVal, hasApp := wecom["wecom_app"].(map[string]interface{})
	if !hasApp {
		t.Fatalf("ACE 缺 wecom_app 时应自动补空对象，实际 wecom=%v", wecom)
	}
	if len(appVal) != 0 {
		t.Errorf("补齐的 wecom_app 应为空对象 {}，实际=%v", appVal)
	}
}

// TestNormalizeWecomShape_ACEAfterScriptJQ 模拟 list_channels_ace.sh jq 归一化
// 成功后的输出（bot 已在子对象中），验证 Go 层不会重复搬运或破坏已归一化的结构。
func TestNormalizeWecomShape_ACEAfterScriptJQ(t *testing.T) {
	// 模拟 list_channels_ace.sh jq 归一化后的输出
	in := `{"wecom":{"enabled":true,"botPrefix":"","groupPolicy":"open","groupAllowFrom":[],"groups":{},"sendThinkingMessage":true,"mediaDir":"~/.lightclaw/media","mediaLocalRoots":[],"bot":{"botId":"aibcBbX0sHQ0sJkfJKeV-HDRMLV0MHYiDfh","secret":"Ab9eAZL1oq8hSUOf8RYSaoc6Holw7nRmcZP9Zup7u6T","name":"{{bot_name}}","websocketUrl":"","dmPolicy":"open","allowFrom":[]}}}`
	out := normalizeWecomShape(in)

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("输出应为合法 JSON: %v", err)
	}
	wecom := m["wecom"].(map[string]interface{})

	// bot 子对象内容应完整保留
	bot := wecom["bot"].(map[string]interface{})
	if bot["botId"] != "aibcBbX0sHQ0sJkfJKeV-HDRMLV0MHYiDfh" {
		t.Errorf("bot.botId 应保留原值，实际=%v", bot["botId"])
	}
	if bot["secret"] != "Ab9eAZL1oq8hSUOf8RYSaoc6Holw7nRmcZP9Zup7u6T" {
		t.Errorf("bot.secret 应保留原值，实际=%v", bot["secret"])
	}

	// wecom_app 应被补齐
	if _, hasApp := wecom["wecom_app"].(map[string]interface{}); !hasApp {
		t.Errorf("应补齐 wecom_app={}，实际 wecom=%v", wecom)
	}

	// 通道级字段应保留
	if wecom["enabled"] != true {
		t.Errorf("enabled 应保留 true，实际=%v", wecom["enabled"])
	}
	if wecom["sendThinkingMessage"] != true {
		t.Errorf("sendThinkingMessage 应保留 true，实际=%v", wecom["sendThinkingMessage"])
	}
}

// ─── HandleAutoChannel 前置 egress 诊断 ──────────────────────────────────────
//
// 设计要点（对照 controller/openclaw_channel.go L448-465）：
//   - 仅对 egressDiagnosticChannelWhitelist 中的 channel（openclaw-weixin / weixin / feishu）触发诊断
//   - blocked=true  → sendSSE event: fail，文案为 EgressBlockedMessage，跳过脚本
//   - blocked=false → 继续正常脚本路径（这里会走到 RunScript → LoadScript mock 失败）
//   - diagErr      → slog.Warn 跳过，继续正常脚本路径
//   - 白名单外 channel 完全跳过诊断，走正常脚本路径

// TestHandleAutoChannel_EgressBlocked_ShortCircuit 白名单 channel 诊断命中时，
// 响应体包含 event: fail 且 message = EgressBlockedMessage，且不应调用 LoadScript
// （前置诊断命中应在 RunScript 之前短路返回）。
func TestHandleAutoChannel_EgressBlocked_ShortCircuit(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// 替换 DiagnoseInstanceEgress 为返回 blocked=true
	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		return true, nil
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	// LoadScript mock：若被调用说明短路未生效，测试失败
	loadScriptCalled := false
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("mock: should not be called")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-egress-blocked",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	for _, channel := range []string{"feishu", "openclaw-weixin"} {
		t.Run(channel, func(t *testing.T) {
			loadScriptCalled = false
			req := channelReqWithSession(t, http.MethodGet,
				fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=%s", inst.ID, channel),
				"u1", "")
			rr := httptest.NewRecorder()
			handleAutoChannel(rr, req, testCVMFetcher)

			body := rr.Body.String()
			if !strings.Contains(body, "event: fail") {
				t.Errorf("应发送 event: fail，实际 body=%s", body)
			}
			if !strings.Contains(body, i18n.T(req.Context(), EgressBlockedMessage)) {
				t.Errorf("message 应为 EgressBlockedMessage=%q，实际 body=%s", EgressBlockedMessage, body)
			}
			if loadScriptCalled {
				t.Errorf("blocked 应短路返回，不应调用 LoadScript")
			}
		})
	}
}

// TestHandleAutoChannel_EgressNotBlocked_ProceedsToScript 诊断 blocked=false 时
// 不拦截，继续走 RunScript 正常路径（这里 LoadScript mock 失败 → sendSSE fail
// 带原始错误文案，不会是 EgressBlockedMessage）。
func TestHandleAutoChannel_EgressNotBlocked_ProceedsToScript(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		return false, nil
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	loadScriptCalled := false
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("mock: script-load-fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-egress-ok",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=feishu", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	if !loadScriptCalled {
		t.Errorf("blocked=false 应继续走 RunScript 路径，LoadScript 应被调用")
	}
	body := rr.Body.String()
	if strings.Contains(body, i18n.T(req.Context(), EgressBlockedMessage)) {
		t.Errorf("blocked=false 不应出现 EgressBlockedMessage，body=%s", body)
	}
}

// TestHandleAutoChannel_EgressDiagError_ProceedsToScript 诊断自身失败时（云 API
// 超时、凭证问题等）应视为"不确定"，不拦截，继续走脚本路径，避免诊断链路故障
// 反过来阻塞正常业务。
func TestHandleAutoChannel_EgressDiagError_ProceedsToScript(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		return false, hcommon.I18nError(i18n.MsgCVMAPITimeout)
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	loadScriptCalled := false
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("mock: script-load-fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-diag-err",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=feishu", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	if !loadScriptCalled {
		t.Errorf("诊断失败应继续走 RunScript 路径，LoadScript 应被调用")
	}
	if strings.Contains(rr.Body.String(), i18n.T(req.Context(), EgressBlockedMessage)) {
		t.Errorf("诊断失败不应出现 EgressBlockedMessage，body=%s", rr.Body.String())
	}
}

// TestHandleAutoChannel_NonWhitelistedChannel_SkipsDiagnostic 白名单外的
// channel（qqbot）即使其他条件都满足也不应触发诊断，避免对不依赖出站的通道误报。
func TestHandleAutoChannel_NonWhitelistedChannel_SkipsDiagnostic(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	diagCalled := false
	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		diagCalled = true
		return true, nil // 即便返回 blocked 也不应被调用
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: not found")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-qq-no-diag",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=qqbot", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	if diagCalled {
		t.Errorf("qqbot 非白名单，不应调用 DiagnoseInstanceEgress")
	}
}

// TestHandleAutoChannel_LarkChannel_BuildsGreetingParams 覆盖
// openclaw_channel.go 的 "lark" case 分支：构建包含英文动态欢迎词的
// params（与 feishu 的中文欢迎词对称），并通过 egress 诊断后进入 RunScript。
// lark 与 feishu 共用 feishu_bot_creator feature（autoChannelFeature["lark"]），
// 但 greeting 文案为英文，params 必须包含 greeting 键。
func TestHandleAutoChannel_LarkChannel_BuildsGreetingParams(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// lark 在 egress 诊断白名单中，mock 为未阻断以继续走 RunScript
	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		return false, nil
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	// mock LoadScript 失败：使 RunScript 走到 LoadScript 即失败，
	// 若 LoadScript 被调用，证明 lark case 已执行（构建 params 后继续到 RunScript）。
	loadScriptCalled := false
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("mock: script-load-fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-lark-greet",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=lark", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	// lark case（563-568）已执行 → 继续走 egress 诊断（未阻断）→ RunScript → LoadScript 失败
	if !loadScriptCalled {
		t.Errorf("lark case 应构建 params 后继续走 RunScript，LoadScript 应被调用")
	}
	body := rr.Body.String()
	if !strings.Contains(body, "event: fail") {
		t.Errorf("LoadScript mock 失败应触发 SSE fail，实际 body=%s", body)
	}
	// lark 未阻断，不应出现 EgressBlockedMessage
	if strings.Contains(body, i18n.T(req.Context(), EgressBlockedMessage)) {
		t.Errorf("lark 未阻断不应出现 EgressBlockedMessage，body=%s", body)
	}
}

func TestHandleAutoChannel_WhatsappChannel_SetsJSONLinesHandler(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// whatsapp 不在 egressDiagnosticChannelWhitelist 中，不应调用诊断
	diagCalled := false
	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		diagCalled = true
		return true, nil
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	// mock LoadScript 失败：若被调用说明 whatsapp case 已执行（设置 onOutput
	// 后继续到 RunScript），从而证明 L595-596 已被覆盖。
	loadScriptCalled := false
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("mock: script-load-fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-whatsapp-auto",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channel/auto?id=%d&channel=whatsapp", inst.ID),
		"u1", "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, testCVMFetcher)

	// whatsapp case（L595-596）已执行 → 跳过诊断 → RunScript → LoadScript 失败
	if !loadScriptCalled {
		t.Errorf("whatsapp case 应设置 onOutput 后继续走 RunScript，LoadScript 应被调用")
	}
	if diagCalled {
		t.Errorf("whatsapp 非白名单，不应调用 DiagnoseInstanceEgress")
	}
	body := rr.Body.String()
	if !strings.Contains(body, "event: fail") {
		t.Errorf("LoadScript mock 失败应触发 SSE fail，实际 body=%s", body)
	}
	if strings.Contains(body, i18n.T(req.Context(), EgressBlockedMessage)) {
		t.Errorf("whatsapp 未阻断不应出现 EgressBlockedMessage，body=%s", body)
	}
}

// ─── HandleSetChannel egress 诊断分支 ───────────────────────────────────────
//
// HandleSetChannel RunScript 失败后会对白名单 channel 调 maybeWrapEgressBlocked
// （controller/openclaw_channel.go L154-159）。这里用 DiagnoseInstanceEgress
// mock 验证端到端：响应是否包含 EgressBlockedMessage。

// TestHandleSetChannel_WhitelistedEgressBlocked_ReplacesError 白名单 channel
// RunScript 失败且诊断命中时，响应错误文案被替换为 EgressBlockedMessage。
func TestHandleSetChannel_WhitelistedEgressBlocked_ReplacesError(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		return true, nil
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: i/o timeout")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-set-blocked",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Set("key", "appid")
	form.Set("value", "12345")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), i18n.T(req.Context(), EgressBlockedMessage)) {
		t.Errorf("响应应包含 EgressBlockedMessage=%q，实际 body=%s", EgressBlockedMessage, rr.Body.String())
	}
	waitForGoroutines()
}

// TestHandleSetChannel_NonWhitelistedChannel_NoDiagnostic 白名单外 channel
// RunScript 失败时不应调用诊断，响应为 newRichError 包装的原错误。
func TestHandleSetChannel_NonWhitelistedChannel_NoDiagnostic(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	diagCalled := false
	origDiag := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (bool, error) {
		diagCalled = true
		return true, nil
	}
	defer func() { DiagnoseInstanceEgress = origDiag }()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: i/o timeout")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-set-wecom",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "wecom") // 白名单外
	form.Set("key", "bot_id")
	form.Set("value", "xxx")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if diagCalled {
		t.Errorf("wecom 非白名单，不应调用 DiagnoseInstanceEgress")
	}
	if strings.Contains(rr.Body.String(), i18n.T(req.Context(), EgressBlockedMessage)) {
		t.Errorf("不应出现 EgressBlockedMessage，body=%s", rr.Body.String())
	}
	waitForGoroutines()
}

// TestEgressDiagnosticChannelWhitelist_Content 保证白名单常量内容稳定：
// 只覆盖个人微信（两个别名）+ 飞书/Lark + WhatsApp，避免他人扩展时误加其他 channel。
func TestEgressDiagnosticChannelWhitelist_Content(t *testing.T) {
	want := map[string]bool{
		"openclaw-weixin":   true,
		"weixin":            true,
		"feishu":            true,
		"lark":              true,
		"openclaw_whatsapp": true,
	}
	if len(egressDiagnosticChannelWhitelist) != len(want) {
		t.Errorf("白名单条目数不匹配：期望=%d 实际=%d", len(want), len(egressDiagnosticChannelWhitelist))
	}
	for k := range want {
		if !egressDiagnosticChannelWhitelist[k] {
			t.Errorf("白名单缺失 %s", k)
		}
	}
	for _, k := range []string{"wecom", "wecom_app", "ddingtalk", "qqbot", ""} {
		if egressDiagnosticChannelWhitelist[k] {
			t.Errorf("%s 不应在 egress 诊断白名单中", k)
		}
	}
}

// ─── normalizeShowQrcode ───────────────────────────────────────────────────

func TestNormalizeShowQrcode_OpenClawFeishu(t *testing.T) {
	// OpenClaw 飞书：content 是 {"qrlogin":{"token":"<short_token>"}} → mode=qrlogin，content 不变，无 url 字段
	raw := json.RawMessage(`{"action":"show_qrcode","content":"{\"qrlogin\":{\"token\":\"abc123\"}}"}`)
	got := normalizeShowQrcode(raw)
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["mode"] != "qrlogin" {
		t.Errorf("mode: got=%q want=qrlogin", m["mode"])
	}
	if m["content"] != `{"qrlogin":{"token":"abc123"}}` {
		t.Errorf("content should be unchanged, got=%q", m["content"])
	}
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("mode=qrlogin should NOT have url field, got=%q", m["url"])
	}
}

func TestNormalizeShowQrcode_HermesFeishu(t *testing.T) {
	// Hermes 飞书：token 是 URL → mode=url，content 展开为裸 URL，无 url 字段（无 render_qr）
	raw := json.RawMessage(`{"action":"show_qrcode","content":"{\"qrlogin\":{\"token\":\"https://open.feishu.cn/page/launcher?user_code=XYZ\"}}"}`)
	got := normalizeShowQrcode(raw)
	m := got.(map[string]interface{})
	if m["mode"] != "url" {
		t.Errorf("mode: got=%q want=url", m["mode"])
	}
	if m["content"] != "https://open.feishu.cn/page/launcher?user_code=XYZ" {
		t.Errorf("content should be bare URL, got=%q", m["content"])
	}
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("Hermes feishu should NOT have url field (no render_qr), got=%q", m["url"])
	}
}

func TestNormalizeShowQrcode_FeishuVerificationURI(t *testing.T) {
	// OpenClaw 飞书 Device Code Flow：content 是 {"verification_uri":"https://..."} → mode=url，content 展开为裸 URL
	raw := json.RawMessage(`{"action":"show_qrcode","content":"{\"verification_uri\":\"https://open.feishu.cn/page/launcher?user_code=4FNE-MMRA\"}"}`)
	got := normalizeShowQrcode(raw)
	m := got.(map[string]interface{})
	if m["mode"] != "url" {
		t.Errorf("mode: got=%q want=url", m["mode"])
	}
	if m["content"] != "https://open.feishu.cn/page/launcher?user_code=4FNE-MMRA" {
		t.Errorf("content should be verification_uri URL, got=%q", m["content"])
	}
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("verification_uri feishu should NOT have url field, got=%q", m["url"])
	}
}

func TestNormalizeShowQrcode_WeixinURL(t *testing.T) {
	// Hermes/ACE 微信：content 是裸 URL，无 render_qr → mode=url，无 url 字段
	raw := json.RawMessage(`{"action":"show_qrcode","content":"https://liteapp.weixin.qq.com/q/abc"}`)
	got := normalizeShowQrcode(raw)
	m := got.(map[string]interface{})
	if m["mode"] != "url" {
		t.Errorf("mode: got=%q want=url", m["mode"])
	}
	if m["content"] != "https://liteapp.weixin.qq.com/q/abc" {
		t.Errorf("content should be unchanged URL, got=%q", m["content"])
	}
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("Hermes/ACE weixin should NOT have url field (no render_qr), got=%q", m["url"])
	}
}

func TestNormalizeShowQrcode_AsciiArt(t *testing.T) {
	// 纯字符画 → mode=ascii_art，无 url 字段
	raw := json.RawMessage(`{"action":"show_qrcode","content":"▄▀█▀▄"}`)
	got := normalizeShowQrcode(raw)
	m := got.(map[string]interface{})
	if m["mode"] != "ascii_art" {
		t.Errorf("mode: got=%q want=ascii_art", m["mode"])
	}
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("mode=ascii_art should NOT have url field, got=%q", m["url"])
	}
}

func TestNormalizeShowQrcode_RenderQR(t *testing.T) {
	// OpenClaw 微信：content 是裸 URL + render_qr=true → mode=ascii_art，content 转为字符画
	raw := json.RawMessage(`{"action":"show_qrcode","content":"https://liteapp.weixin.qq.com/q/abc","render_qr":true}`)
	got := normalizeShowQrcode(raw)
	m := got.(map[string]interface{})
	if m["mode"] != "ascii_art" {
		t.Errorf("mode: got=%q want=ascii_art", m["mode"])
	}
	// content 应该是字符画（包含 UTF8 半块字符），不再是裸 URL
	contentStr, _ := m["content"].(string)
	if contentStr == "https://liteapp.weixin.qq.com/q/abc" {
		t.Errorf("content should be converted to ascii art, but is still raw URL")
	}
	if len(contentStr) == 0 {
		t.Errorf("content should not be empty")
	}
	// 不应有冗余的 url 字段
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("render_qr should NOT output url field, got=%q", m["url"])
	}
}

func TestNormalizeShowQrcode_RenderQR_NoEffectOnNonURL(t *testing.T) {
	// render_qr=true 但 content 不是 URL（是字符画）→ 不做转换，mode=ascii_art
	raw := json.RawMessage(`{"action":"show_qrcode","content":"▄▀█▀▄","render_qr":true}`)
	got := normalizeShowQrcode(raw)
	m := got.(map[string]interface{})
	if m["mode"] != "ascii_art" {
		t.Errorf("mode: got=%q want=ascii_art", m["mode"])
	}
	if m["content"] != "▄▀█▀▄" {
		t.Errorf("content should be unchanged ascii art, got=%q", m["content"])
	}
	// content 不是 URL，mode 不是 url，render_qr 的转换逻辑不会触发
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("non-url content with render_qr should NOT have url field")
	}
}

func TestNormalizeShowQrcode_InvalidJSON(t *testing.T) {
	// 非法 JSON → 原样返回
	raw := json.RawMessage(`not json`)
	got := normalizeShowQrcode(raw)
	if _, ok := got.(json.RawMessage); !ok {
		t.Errorf("invalid JSON should return raw RawMessage, got %T", got)
	}
}

// ─── HandleSetChannel 分组可见性校验 ──────────────────────────────────────

func TestHandleSetChannel_GroupVisibility_ZeroGroupInstance_Forbidden(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-ch-grp0",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		GroupID: 0,
	}
	model.DB(context.Background()).Create(inst)

	// 创建一个 visibility_type=group 的通道
	enabled := true
	ch := &model.AIChannel{
		ChannelID: "feishu", Name: "飞书", Enabled: &enabled,
		VisibilityType: "group",
	}
	model.DB(context.Background()).Create(ch)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Add("key", "appid")
	form.Add("value", "12345")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("未分组实例应被拒绝使用分组通道，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetChannel_AllVisibility_ZeroGroupInstance_Allowed(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-ch-all0",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		GroupID: 0,
	}
	model.DB(context.Background()).Create(inst)

	// 创建一个 visibility_type=all 的通道
	enabled := true
	ch := &model.AIChannel{
		ChannelID: "feishu", Name: "飞书", Enabled: &enabled,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(ch)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Add("key", "appid")
	form.Add("value", "12345")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	// 不应返回 403（全局可见通道不受分组限制）
	if rr.Code == http.StatusForbidden {
		t.Errorf("全局可见通道不应被拒绝，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleChannelsList with id (wrapped response) ────────────────────

func TestHandleChannelsList_WithInstanceID_Success(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chlist",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		AgentReady: 1, RuntimeUser: "root",
	}
	model.DB(context.Background()).Create(inst)

	origRunner := listChannelsScriptRunner
	listChannelsScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"feishu":{"enabled":true}}`, nil
	}
	defer func() { listChannelsScriptRunner = origRunner }()

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channels?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleChannelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		AgentType                  string                 `json:"agent_type"`
		AgentTypeSupportedChannels []string               `json:"agent_type_supported_channels"`
		Channels                   map[string]interface{} `json:"channels"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.AgentType != model.AgentTypeOpenClaw {
		t.Errorf("agent_type 应为 openclaw，实际=%s", resp.AgentType)
	}
	if len(resp.AgentTypeSupportedChannels) == 0 {
		t.Error("agent_type_supported_channels 不应为空")
	}
	if _, ok := resp.Channels["feishu"]; !ok {
		t.Error("channels 应包含 feishu")
	}
}

func TestHandleChannelsList_WithInstanceID_KeepsExistingOverseasOnlyChannelsOnDomesticSite(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	i18n.SetDefaultLang("zh")
	defer i18n.SetDefaultLang("zh")
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chlist-site",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		AgentReady: 1, RuntimeUser: "root",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "msteams", Name: "Microsoft Teams"})

	origRunner := listChannelsScriptRunner
	listChannelsScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"feishu":{"enabled":true},"slack":{"enabled":true},"msteams":{"enabled":true}}`, nil
	}
	defer func() { listChannelsScriptRunner = origRunner }()

	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/channels?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleChannelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		AgentTypeSupportedChannels []string               `json:"agent_type_supported_channels"`
		Channels                   map[string]interface{} `json:"channels"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, ch := range []string{"slack", "msteams"} {
		if _, ok := resp.Channels[ch]; !ok {
			t.Fatalf("domestic site should keep existing %s config, channels=%v", ch, resp.Channels)
		}
	}
	if _, ok := resp.Channels["feishu"]; !ok {
		t.Fatalf("domestic site should keep configured feishu, channels=%v", resp.Channels)
	}
	for _, got := range resp.AgentTypeSupportedChannels {
		if got == "slack" {
			t.Fatalf("domestic site should hide slack from supported channels: %v", resp.AgentTypeSupportedChannels)
		}
	}
}

// ─── 补充覆盖：set-channel key/value 校验分支 ─────────────────────────────

func TestHandleSetChannel_OnlyKeyNoValue(t *testing.T) {
	// 覆盖 openclaw_channel.go:135-137 — len(values) == 0 分支
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-onlykey",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Set("key", "appid") // 只有 key，没有 value
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("只有 key 没有 value 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "value") {
		t.Errorf("错误信息应提及 value，实际=%s", rr.Body.String())
	}
}

func TestHandleSetChannel_KeyValueCountMismatch(t *testing.T) {
	// 覆盖 openclaw_channel.go:139 — len(keys) != len(values) 分支
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-mismatch",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Add("key", "appid")
	form.Add("key", "secret")
	form.Add("value", "val1") // 2 个 key，1 个 value
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("key/value 数量不匹配应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "不匹配") {
		t.Errorf("错误信息应提及不匹配，实际=%s", rr.Body.String())
	}
}

func TestHandleSetChannel_EmptyKeyInList(t *testing.T) {
	// 覆盖 openclaw_channel.go:144-146 — keys[i] == "" 分支
	cleanup := initChannelTestDB(t)
	defer cleanup()
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-emptykey",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Add("key", "") // 空 key
	form.Add("value", "val1")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 key 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "key") {
		t.Errorf("错误信息应提及 key，实际=%s", rr.Body.String())
	}
}

// ─── setChannel line 分支测试 ───────────────────────────────────────────

func TestSetChannel_MSTeams_Overseas(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)

	// Mock channelScriptRunner to avoid running actual scripts
	origRunner := channelScriptRunner
	channelScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		return "msteams configured", nil
	}
	defer func() { channelScriptRunner = origRunner }()

	user := &model.User{Username: "teamsuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "teams-inst", InstanceId: "ins-teams-set",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, RuntimeUser: "ubuntu",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "msteams", Name: "Microsoft Teams"})

	form := url.Values{}
	form.Set("channel", "msteams")
	form.Add("key", "app_id")
	form.Add("value", "test-app-id")
	form.Add("key", "app_secret")
	form.Add("value", "test-secret")

	// Inject overseas context so msteams passes site scope check
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "teamsuser", form.Encode())
	req = req.WithContext(hcommon.InjectTenant(req.Context(), hcommon.TenantSnapshot{DefaultLang: "en"}))

	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("msteams set channel code=%d body=%s", rr.Code, rr.Body.String())
	}

	// Verify proxy route info and teams_endpoint in response (lines 517, 520-523)
	body := rr.Body.String()
	if !strings.Contains(body, `"proxy_route_id"`) {
		t.Errorf("response should contain proxy_route_id: %s", body)
	}
	if !strings.Contains(body, `"proxy_endpoint"`) {
		t.Errorf("response should contain proxy_endpoint: %s", body)
	}
	if !strings.Contains(body, `"teams_endpoint"`) {
		t.Errorf("response should contain teams_endpoint: %s", body)
	}
}

func TestSetChannel_Line_Overseas(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)

	// Mock channelScriptRunner to avoid running actual scripts
	origRunner := channelScriptRunner
	channelScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		return "line configured", nil
	}
	defer func() { channelScriptRunner = origRunner }()

	user := &model.User{Username: "lineuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "line-inst", InstanceId: "ins-line-set",
		UserID: user.ID, AgentType: model.AgentTypeHermes, RuntimeUser: "ubuntu",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "line", Name: "LINE"})

	form := url.Values{}
	form.Set("channel", "line")
	form.Add("key", "channel_secret")
	form.Add("value", "test-secret")
	form.Add("key", "channel_access_token")
	form.Add("value", "test-token")

	// Inject overseas context so line passes site scope check
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "lineuser", form.Encode())
	req = req.WithContext(hcommon.InjectTenant(req.Context(), hcommon.TenantSnapshot{DefaultLang: "en"}))

	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("line set channel code=%d body=%s", rr.Code, rr.Body.String())
	}

	// Verify proxy route info in response (lines 517, 520-521)
	body := rr.Body.String()
	if !strings.Contains(body, `"proxy_route_id"`) {
		t.Errorf("response should contain proxy_route_id: %s", body)
	}
	if !strings.Contains(body, `"proxy_endpoint"`) {
		t.Errorf("response should contain proxy_endpoint: %s", body)
	}
}

func TestSetChannel_Line_EnsureProxyRouteError(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	// resolveInstanceAccessIPForAgentProxy returns error
	withAgentProxyHooks(t, "", nil)

	user := &model.User{Username: "lineerruser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "line-err-inst", InstanceId: "ins-line-err",
		UserID: user.ID, AgentType: model.AgentTypeHermes, RuntimeUser: "ubuntu",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "line", Name: "LINE"})

	form := url.Values{}
	form.Set("channel", "line")
	form.Add("key", "channel_secret")
	form.Add("value", "test-secret")

	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "lineerruser", form.Encode())
	req = req.WithContext(hcommon.InjectTenant(req.Context(), hcommon.TenantSnapshot{DefaultLang: "en"}))

	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	// ensureAgentProxyRoute should fail, returning 500 (line 478-480)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("line set channel with proxy error code=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── deleteChannel line 分支测试 ────────────────────────────────────────

func TestDeleteChannel_Line_DisablesProxyRoute(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)

	// Mock delChannelScriptRunner to avoid running actual scripts
	origRunner := delChannelScriptRunner
	delChannelScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		return "line deleted", nil
	}
	defer func() { delChannelScriptRunner = origRunner }()

	user := &model.User{Username: "dellineuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "line-del-inst", InstanceId: "ins-line-del",
		UserID: user.ID, AgentType: model.AgentTypeHermes, RuntimeUser: "ubuntu",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "line", Name: "LINE"})

	// Create an enabled proxy route for line
	route := model.AgentProxyRoute{
		RouteID:    "test-line-route",
		InstanceID: "ins-line-del",
		Kind:       model.AgentProxyRouteKindLine,
		TargetIP:   "10.1.1.1",
		TargetPort: 8646,
		TargetPath: "/line/webhook",
		Enabled:    true,
	}
	model.DB(context.Background()).Create(&route)

	form := url.Values{}
	form.Set("channel", "line")

	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel/del?id=%d", inst.ID), "dellineuser", form.Encode())
	req = req.WithContext(hcommon.InjectTenant(req.Context(), hcommon.TenantSnapshot{DefaultLang: "en"}))

	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("line delete channel code=%d body=%s", rr.Code, rr.Body.String())
	}

	// Verify the proxy route is disabled (lines 634-643)
	var updated model.AgentProxyRoute
	if err := model.DB(context.Background()).Where("route_id = ?", "test-line-route").First(&updated).Error; err != nil {
		t.Fatalf("query disabled route: %v", err)
	}
	if updated.Enabled {
		t.Fatal("line proxy route should be disabled after delete")
	}
}

func TestDeleteChannel_Line_DisableProxyRouteError(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)

	// Mock delChannelScriptRunner
	origRunner := delChannelScriptRunner
	delChannelScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		return "line deleted", nil
	}
	defer func() { delChannelScriptRunner = origRunner }()

	user := &model.User{Username: "dellineerr", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "line-del-err-inst", InstanceId: "ins-line-del-err",
		UserID: user.ID, AgentType: model.AgentTypeHermes, RuntimeUser: "ubuntu",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "line", Name: "LINE"})

	// Create a proxy route that will be disabled
	route := model.AgentProxyRoute{
		RouteID:    "test-line-err-route",
		InstanceID: "ins-line-del-err",
		Kind:       model.AgentProxyRouteKindLine,
		TargetIP:   "10.1.1.1",
		TargetPort: 8646,
		TargetPath: "/line/webhook",
		Enabled:    true,
	}
	model.DB(context.Background()).Create(&route)

	form := url.Values{}
	form.Set("channel", "line")

	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel/del?id=%d", inst.ID), "dellineerr", form.Encode())
	req = req.WithContext(hcommon.InjectTenant(req.Context(), hcommon.TenantSnapshot{DefaultLang: "en"}))

	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)

	// The route exists, so the disable should succeed, even if RefreshAllRuleSetsForRequiredRules fails
	if rr.Code != http.StatusOK {
		t.Fatalf("line delete channel code=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── applyManualChannelConfig line 分支测试 ─────────────────────────────

func TestApplyManualChannelConfig_Line(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)

	// Mock channelScriptRunner
	origRunner := channelScriptRunner
	channelScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		return "line preset applied", nil
	}
	defer func() { channelScriptRunner = origRunner }()

	user := &model.User{Username: "presetline", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "preset-line-inst", InstanceId: "ins-preset-line",
		UserID: user.ID, AgentType: model.AgentTypeHermes, RuntimeUser: "ubuntu",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "line", Name: "LINE"})

	// Inject overseas context
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{DefaultLang: "en"})
	req := httptest.NewRequest(http.MethodPost, "/test", nil).WithContext(ctx)

	preset := manualChannelPreset{
		Channel: "line",
		Config: map[string]string{
			"channel_secret":       "test-secret",
			"channel_access_token": "test-token",
		},
	}

	result, err := applyManualChannelConfig(req, inst, preset)
	if err != nil {
		t.Fatalf("applyManualChannelConfig for line failed: %v", err)
	}
	if result.ProxyRouteID == "" {
		t.Fatal("expected proxy route id to be set for line channel")
	}
	if result.ProxyEndpoint == "" {
		t.Fatal("expected proxy endpoint to be set for line channel")
	}
}
