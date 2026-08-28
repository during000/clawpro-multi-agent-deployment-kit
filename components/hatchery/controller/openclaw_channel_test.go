package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initChannelTestDB 初始化内存 SQLite 数据库用于通道测试
func initChannelTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.SiteConfig{},
		&model.AIChannel{}, &model.GroupConfigBinding{}, &model.UserGroup{}, &model.GroupClosure{}, &model.AgentProxyRoute{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

func channelReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	session.Save(req, rr)

	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// v7：矩阵放开 Hermes Channel 支持后，原"hermes 配通道 403"断言失效。
// 重写为：hermes 配置白名单外的 channel（如 "qq"）应被 v7 白名单拦截为 400。
func TestHandleSetChannel_NonWhitelistedChannel_Hermes(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// 创建用户和 hermes 类型实例
	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "hermes-inst",
		InstanceId: "ins-hermes-001",
		UserID:     user.ID,
		AgentType:  "hermes",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	// "qq" 不在 Hermes 白名单 {qqbot}，应被 v7 白名单拦截
	// （feishu/weixin/ddingtalk/wecom 均已从白名单移除：feishu 为 2026-04-20 产品侧下线，
	// 其余因对应 harness CLI 不支持或产品侧决定不开放）
	form := url.Values{}
	form.Set("channel", "qq")
	form.Set("key", "appid")
	form.Set("value", "12345")
	req := channelReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("hermes 实例配置白名单外通道应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetChannel_MethodNotAllowed(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/channel", nil)
	rr := httptest.NewRecorder()

	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 请求应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleSetChannel_DomesticRejectsOverseasOnlyChannels(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	i18n.SetDefaultLang("zh")
	defer i18n.SetDefaultLang("zh")

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "openclaw-inst",
		InstanceId: "ins-openclaw-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack"})

	cases := []struct {
		channel string
		params  map[string]string
	}{
		{channel: "slack", params: map[string]string{"app_token": "xapp-test", "bot_token": "xoxb-test"}},
	}
	for _, tc := range cases {
		t.Run(tc.channel, func(t *testing.T) {
			form := url.Values{}
			form.Set("channel", tc.channel)
			for k, v := range tc.params {
				form.Add("key", k)
				form.Add("value", v)
			}
			req := channelReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "testuser", form.Encode())
			rr := httptest.NewRecorder()

			handleSetChannel(rr, req, testCVMFetcher)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("domestic site should reject %s like unsupported channel, got %d body=%s", tc.channel, rr.Code, rr.Body.String())
			}
			if strings.Contains(rr.Body.String(), "当前站点") {
				t.Fatalf("domestic site should not expose site-scope rejection, body=%s", rr.Body.String())
			}
		})
	}
}

func TestApplyChannelPresetsAsync_UsesManualChannelOperationInOrder(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "preset-channel-u", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	instance := &model.Instance{
		Name:        "preset-channel-inst",
		InstanceId:  "ins-preset-channel",
		UserID:      user.ID,
		AgentType:   model.AgentTypeOpenClaw,
		AgentReady:  1,
		RuntimeUser: "ubuntu",
	}
	if err := model.DB(context.Background()).Create(instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID:      "preset-custom",
		Name:           "Preset Custom",
		Custom:         true,
		VisibilityType: model.VisibilityAll,
		CustomConfig:   `{"server":{"endpoint":"https://channel.example.com"},"cred_fields":[{"key":"user_token","label":"User Token"}]}`,
	}).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	originalRunner := channelScriptRunner
	var tokens []string
	channelScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		tokens = append(tokens, params["user_token"])
		if params["channel"] != "preset-custom" || params["is_custom"] != "true" {
			t.Fatalf("unexpected channel params: %#v", params)
		}
		if !strings.Contains(params["channel_config"], "https://channel.example.com") {
			t.Fatalf("custom server config not merged: %s", params["channel_config"])
		}
		return "ok", nil
	}
	defer func() { channelScriptRunner = originalRunner }()

	applyChannelPresetsAsync(context.Background(), "https://hatchery.example.com", instance.ID, []manualChannelPreset{
		{Channel: "preset-custom", Config: map[string]string{"user_token": "token-1"}},
		{Channel: "preset-custom", Config: map[string]string{"user_token": "token-2"}},
	})

	if len(tokens) != 2 || tokens[0] != "token-1" || tokens[1] != "token-2" {
		t.Fatalf("channel application order = %#v, want [token-1 token-2]", tokens)
	}
}
