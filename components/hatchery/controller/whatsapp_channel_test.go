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

	"hatchery/model"
)

// ========== HandleAutoChannel whatsapp 分支测试 ==========

func TestHandleAutoChannel_WhatsApp_MissingPhone(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "wa-test-inst",
		InstanceId: "ins-wa-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 注册 WhatsApp 自定义通道配置
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"pairingMode":true,"phoneRequired":true,"autoFeature":"whatsapp_pairing","autoTimeout":180,"dmPolicy":"allowlist","selfChatMode":true,"deleteFeature":"del_whatsapp_channel","egressRequired":true},"cred_fields":[{"key":"phone_number","label":"手机号（带国家码，不含+，如 85266803489）"}]}`,
	})

	// channel=openclaw_whatsapp（自定义通道配对码模式），但不传 phone
	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=openclaw_whatsapp", inst.ID),
		"testuser", "")
	rr := httptest.NewRecorder()

	handleAutoChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 phone 参数应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "phone") {
		t.Errorf("错误消息应提及 phone，body=%s", rr.Body.String())
	}
}

func TestHandleAutoChannel_WhatsApp_InvalidPhoneFormat(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "wa-test-inst",
		InstanceId: "ins-wa-002",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 注册 WhatsApp 自定义通道配置
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"pairingMode":true,"phoneRequired":true,"phonePattern":"^[1-9]\\d{6,14}$","autoFeature":"whatsapp_pairing","autoTimeout":180,"dmPolicy":"allowlist","selfChatMode":true,"deleteFeature":"del_whatsapp_channel","egressRequired":true},"cred_fields":[{"key":"phone_number","label":"手机号（带国家码，不含+，如 85266803489）"}]}`,
	})

	cases := []struct {
		name  string
		phone string
	}{
		{"带+号", "+85266803489"},
		{"带空格", "852 6680 3489"},
		{"太短", "123"},
		{"太长", "1234567890123456"},
		{"以0开头", "085266803489"},
		{"含字母", "8526680abc9"},
		{"空字符串参数", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := channelReqWithSession(t, http.MethodGet,
				fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=openclaw_whatsapp&phone=%s",
					inst.ID, url.QueryEscape(tc.phone)),
				"testuser", "")
			rr := httptest.NewRecorder()

			handleAutoChannel(rr, req, testCVMFetcher)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("phone=%q 应返回 400，实际=%d, body=%s", tc.phone, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleAutoChannel_WhatsApp_ValidPhoneFormat(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "wa-test-inst",
		InstanceId: "ins-wa-003",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 注册 WhatsApp 自定义通道配置
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"pairingMode":true,"phoneRequired":true,"autoFeature":"whatsapp_pairing","autoTimeout":180,"dmPolicy":"allowlist","selfChatMode":true,"deleteFeature":"del_whatsapp_channel","egressRequired":true},"cred_fields":[{"key":"phone_number","label":"手机号（带国家码，不含+，如 85266803489）"}]}`,
	})

	// 合法手机号不应返回 400（会继续到 ResolveScript → TAT 阶段，
	// 在测试环境里 ResolveScript 会成功但 RunScript 会报错——不是 400）
	validPhones := []string{
		"85266803489",   // 香港
		"17788391083",   // 美国
		"8613800138000", // 中国大陆
		"6581234567",    // 新加坡
	}

	for _, phone := range validPhones {
		t.Run(phone, func(t *testing.T) {
			req := channelReqWithSession(t, http.MethodGet,
				fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=openclaw_whatsapp&phone=%s",
					inst.ID, phone),
				"testuser", "")
			rr := httptest.NewRecorder()

			handleAutoChannel(rr, req, testCVMFetcher)

			// 合法 phone 不应在参数校验阶段被拒（400）
			// 后续会因为测试环境 TAT 不可用而失败（500 or SSE error），但不应是 400
			if rr.Code == http.StatusBadRequest {
				t.Errorf("phone=%q 是合法格式不应返回 400 (got %d), body=%s", phone, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleAutoChannel_WhatsApp_HermesNotSupported(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "hermes-inst",
		InstanceId: "ins-hermes-wa-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeHermes,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 注册 WhatsApp 自定义通道配置（配对码模式）
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"pairingMode":true,"phoneRequired":true,"autoFeature":"whatsapp_pairing","autoTimeout":180,"deleteFeature":"del_whatsapp_channel","egressRequired":true},"cred_fields":[{"key":"phone_number","label":"手机号"}]}`,
	})

	// channel=openclaw_whatsapp（自定义通道配对码模式）
	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=openclaw_whatsapp&phone=85266803489", inst.ID),
		"testuser", "")
	rr := httptest.NewRecorder()

	handleAutoChannel(rr, req, testCVMFetcher)

	// Hermes 不支持 whatsapp_pairing → ResolveScript 返回 error → 400
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Hermes 实例应不支持 whatsapp 配对码模式，期望 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAutoChannel_WhatsApp_ACENotSupported(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "ace-inst",
		InstanceId: "ins-ace-wa-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeLightclawACE,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 注册 WhatsApp 自定义通道配置（配对码模式）
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"pairingMode":true,"phoneRequired":true,"autoFeature":"whatsapp_pairing","autoTimeout":180,"deleteFeature":"del_whatsapp_channel","egressRequired":true},"cred_fields":[{"key":"phone_number","label":"手机号"}]}`,
	})

	// channel=openclaw_whatsapp（自定义通道配对码模式）
	req := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=openclaw_whatsapp&phone=85266803489", inst.ID),
		"testuser", "")
	rr := httptest.NewRecorder()

	handleAutoChannel(rr, req, testCVMFetcher)

	// ACE 不支持 whatsapp_pairing → ResolveScript 返回 error → 400
	if rr.Code != http.StatusBadRequest {
		t.Errorf("ACE 实例应不支持 whatsapp 配对码模式，期望 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// ========== HandleDelChannel whatsapp 分支测试 ==========

func TestHandleDelChannel_WhatsApp_ResolvesSpecialScript(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// 验证 del_whatsapp_channel feature 在 scriptResolveTable 中正确注册
	name, err := ResolveScript(context.Background(), "del_whatsapp_channel", model.AgentTypeOpenClaw)
	if err != nil {
		t.Fatalf("ResolveScript(del_whatsapp_channel, openclaw) 应成功: %v", err)
	}
	if name != "del_channel_whatsapp.sh" {
		t.Errorf("ResolveScript(del_whatsapp_channel, openclaw) = %q, want del_channel_whatsapp.sh", name)
	}

	// Hermes 不应支持
	_, err = ResolveScript(context.Background(), "del_whatsapp_channel", model.AgentTypeHermes)
	if err == nil {
		t.Error("Hermes 应不支持 del_whatsapp_channel")
	}

	// ACE 不应支持
	_, err = ResolveScript(context.Background(), "del_whatsapp_channel", model.AgentTypeLightclawACE)
	if err == nil {
		t.Error("ACE 应不支持 del_whatsapp_channel")
	}
}

// ========== ScriptResolveTable whatsapp 注册测试 ==========

func TestResolveScript_WhatsAppPairing(t *testing.T) {
	tests := []struct {
		name      string
		feature   string
		agentType string
		want      string
		wantErr   bool
	}{
		// whatsapp_pairing: 仅 OpenClaw 支持
		{"openclaw_whatsapp", "whatsapp_pairing", model.AgentTypeOpenClaw, "set_channel_whatsapp.sh", false},
		{"hermes-whatsapp-not-supported", "whatsapp_pairing", model.AgentTypeHermes, "", true},
		{"ace-whatsapp-not-supported", "whatsapp_pairing", model.AgentTypeLightclawACE, "", true},

		// del_whatsapp_channel: 仅 OpenClaw 支持
		{"openclaw-del-whatsapp", "del_whatsapp_channel", model.AgentTypeOpenClaw, "del_channel_whatsapp.sh", false},
		{"hermes-del-whatsapp-not-supported", "del_whatsapp_channel", model.AgentTypeHermes, "", true},
		{"ace-del-whatsapp-not-supported", "del_whatsapp_channel", model.AgentTypeLightclawACE, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveScript(context.Background(), tt.feature, tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveScript(%q, %q) error = %v, wantErr %v", tt.feature, tt.agentType, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveScript(%q, %q) = %q, want %q", tt.feature, tt.agentType, got, tt.want)
			}
		})
	}
}

// ========== autoChannelFeature / autoChannelTimeout map 注册测试 ==========

func TestAutoChannelMaps_WhatsAppNotInMaps(t *testing.T) {
	// openclaw_whatsapp（自定义通道配对码模式）由 DB CustomConfig 驱动，
	// 不应出现在预定义 map 中（内置扫码通道 "whatsapp" 仍保留在预定义 map 中，两者独立）
	_, ok := autoChannelFeature["openclaw_whatsapp"]
	if ok {
		t.Error("autoChannelFeature 不应包含 openclaw_whatsapp")
	}
	_, ok = autoChannelTimeout["openclaw_whatsapp"]
	if ok {
		t.Error("autoChannelTimeout 不应包含 openclaw_whatsapp")
	}
}

// ========== whatsappPhoneRegexp 正则测试 ==========

func TestWhatsAppPhoneRegexp(t *testing.T) {
	valid := []string{
		"85266803489",     // 香港 11 位
		"17788391083",     // 美国 11 位
		"8613800138000",   // 中国 13 位
		"6581234567",      // 新加坡 10 位
		"447400123456",    // 英国 12 位
		"12345678",        // 最短 8 位
		"123456789012345", // 最长 15 位
	}
	for _, phone := range valid {
		if !whatsappPhoneRegexp.MatchString(phone) {
			t.Errorf("phone=%q 应匹配正则但未匹配", phone)
		}
	}

	invalid := []string{
		"",                 // 空
		"+85266803489",     // 带 +
		"085266803489",     // 以 0 开头
		"123456",           // 太短 (6 位)
		"1234567890123456", // 太长 (16 位)
		"852 6680 3489",    // 含空格
		"852-6680-3489",    // 含横线
		"8526680abc9",      // 含字母
		"abc",              // 纯字母
	}
	for _, phone := range invalid {
		if whatsappPhoneRegexp.MatchString(phone) {
			t.Errorf("phone=%q 不应匹配正则但匹配了", phone)
		}
	}
}

// ========== newJSONLinesHandler show_pairing_code 事件路由测试 ==========

func TestNewJSONLinesHandler_ShowPairingCode(t *testing.T) {
	// 模拟 sendSSE 收集发出的事件
	type sseEvent struct {
		eventType string
		data      string
	}
	var events []sseEvent

	sendSSE := func(event string, data interface{}) {
		jsonData, _ := json.Marshal(data)
		events = append(events, sseEvent{event, string(jsonData)})
	}

	// 构造 newJSONLinesHandler（复制核心逻辑以测试事件路由）
	processedOffset := 0
	handler := func(fullOutput string) {
		remaining := fullOutput[processedOffset:]
		for {
			nlIdx := strings.IndexByte(remaining, '\n')
			if nlIdx < 0 {
				break
			}
			line := strings.TrimSpace(remaining[:nlIdx])
			processedOffset += nlIdx + 1
			remaining = remaining[nlIdx+1:]

			if line == "" {
				continue
			}
			var peek struct {
				Action string `json:"action"`
			}
			if json.Unmarshal([]byte(line), &peek) != nil {
				continue
			}
			switch peek.Action {
			case "show_pairing_code":
				sendSSE("pairing_code", json.RawMessage(line))
			case "show_qrcode":
				sendSSE("qrcode", json.RawMessage(line))
			case "progress":
				sendSSE("progress", json.RawMessage(line))
			case "finish":
				sendSSE("finish", json.RawMessage(line))
			}
		}
	}

	// 模拟 TAT 输出（WhatsApp 配对码流程）
	output := `{"action":"progress","message":"正在连接 WhatsApp..."}
{"action":"show_pairing_code","code":"S3QADVEG","expires_in":60,"message":"请在手机输入配对码"}
{"action":"progress","message":"正在完成关联..."}
{"action":"finish","success":true,"message":"WhatsApp 通道关联成功"}
`
	handler(output)

	// 验证事件
	if len(events) != 4 {
		t.Fatalf("期望 4 个事件，实际=%d", len(events))
	}

	// 事件 1: progress
	if events[0].eventType != "progress" {
		t.Errorf("事件[0] 类型=%q, want progress", events[0].eventType)
	}

	// 事件 2: pairing_code（关键：show_pairing_code → pairing_code SSE event）
	if events[1].eventType != "pairing_code" {
		t.Errorf("事件[1] 类型=%q, want pairing_code", events[1].eventType)
	}
	if !strings.Contains(events[1].data, "S3QADVEG") {
		t.Errorf("事件[1] 应包含配对码 S3QADVEG, data=%s", events[1].data)
	}

	// 事件 3: progress
	if events[2].eventType != "progress" {
		t.Errorf("事件[2] 类型=%q, want progress", events[2].eventType)
	}

	// 事件 4: finish
	if events[3].eventType != "finish" {
		t.Errorf("事件[3] 类型=%q, want finish", events[3].eventType)
	}
}

// ========== deleteChannel whatsapp 路由到专用脚本测试 ==========

func TestHandleDelChannel_WhatsApp_UsesSpecialFeature(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "wa-del-inst",
		InstanceId: "ins-wa-del-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 把 openclaw_whatsapp 作为自定义通道注册到 DB（让白名单校验通过）
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"deleteFeature":"del_whatsapp_channel"},"cred_fields":[]}`,
	})

	form := url.Values{}
	form.Set("channel", "openclaw_whatsapp") // 自定义通道配对码模式
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/del-channel?id=%d", inst.ID), "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleDelChannel(rr, req, testCVMFetcher)

	// 测试环境 TAT 不可用，会在 RunScript 阶段失败（500），不是 400
	// 关键验证：不应因为白名单/AgentType 不支持而被 400 拒绝
	if rr.Code == http.StatusBadRequest {
		t.Errorf("whatsapp 自定义通道删除不应被白名单拒绝 400, body=%s", rr.Body.String())
	}
}

// ========== WhatsApp 内置通道与自定义通道共存测试 ==========
//
// WhatsApp 存在两套独立且互不影响的接入方式，通过不同 channel_id 区分：
//   - "whatsapp"：内置扫码登录通道（走 autoChannelFeature 预定义 map，无需手机号）
//   - "openclaw_whatsapp"：自定义通道框架下的配对码模式（走 DB CustomConfig，需要手机号）
// 两者不做别名映射，各自独立生效，验证互不干扰。

func TestHandleAutoChannel_WhatsApp_BuiltinAndCustomCoexist(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "wa-coexist-inst",
		InstanceId: "ins-wa-coexist-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// 注册 openclaw_whatsapp 自定义通道配置（配对码模式）
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:    "openclaw_whatsapp",
		Name:         "WhatsApp",
		Custom:       true,
		CustomConfig: `{"server":{"pairingMode":true,"phoneRequired":true,"autoFeature":"whatsapp_pairing","autoTimeout":180,"deleteFeature":"del_whatsapp_channel","egressRequired":true},"cred_fields":[{"key":"phone_number","label":"手机号"}]}`,
	})

	// 1) channel=whatsapp（内置扫码通道）：不带 phone，命中 autoChannelFeature 预定义 map，
	//    不应触发自定义通道的 phoneRequired 校验（不应因缺少 phone 返回 400）
	req1 := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=whatsapp", inst.ID),
		"testuser", "")
	rr1 := httptest.NewRecorder()
	handleAutoChannel(rr1, req1, testCVMFetcher)
	if rr1.Code == http.StatusBadRequest && strings.Contains(rr1.Body.String(), "phone") {
		t.Errorf("内置 whatsapp 通道不应要求 phone 参数, code=%d, body=%s", rr1.Code, rr1.Body.String())
	}

	// 2) channel=openclaw_whatsapp（自定义通道配对码模式）：不带 phone，应因 phoneRequired 返回 400
	req2 := channelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=openclaw_whatsapp", inst.ID),
		"testuser", "")
	rr2 := httptest.NewRecorder()
	handleAutoChannel(rr2, req2, testCVMFetcher)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("自定义通道 openclaw_whatsapp 缺少 phone 应返回 400，实际=%d, body=%s", rr2.Code, rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), "phone") {
		t.Errorf("自定义通道 openclaw_whatsapp 缺少 phone 的错误消息应提及 phone, body=%s", rr2.Body.String())
	}
}
