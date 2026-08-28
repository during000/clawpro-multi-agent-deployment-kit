package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

func TestResolveMemoryPlanTransition(t *testing.T) {
	tests := []struct {
		in          string
		wantPlan    string
		wantJobType string
		wantSwitch  string
		wantOK      bool
	}{
		{"off", model.MemoryPlanOff, model.TdaiJobTypeSwitchToOff, model.MemorySwitchStatusSwitchingToOff, true},
		{"FREE", model.MemoryPlanFree, model.TdaiJobTypeSwitchToFree, model.MemorySwitchStatusSwitchingToFree, true},
		{" pro ", model.MemoryPlanPro, model.TdaiJobTypeSwitchToPro, model.MemorySwitchStatusSwitchingToPro, true},
		{"invalid", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			plan, jobType, switchStatus, ok := resolveMemoryPlanTransition(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if plan != tt.wantPlan || jobType != tt.wantJobType || switchStatus != tt.wantSwitch {
				t.Fatalf("got (%q,%q,%q), want (%q,%q,%q)", plan, jobType, switchStatus, tt.wantPlan, tt.wantJobType, tt.wantSwitch)
			}
		})
	}
}

func TestApplyDefaultMemoryPlanForInstance_Free(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-free-default", AgentType: model.AgentTypeOpenClaw})

	applyDefaultMemoryPlanForInstance(context.Background(), "ins-free-default", model.SiteConfig{MemoryDefaultPlan: "free"})

	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-free-default")
	if plugin == nil {
		t.Fatal("expected plugin row")
	}
	if plugin.DesiredPlan != model.MemoryPlanFree {
		t.Fatalf("desired_plan = %q, want FREE", plugin.DesiredPlan)
	}
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToFree {
		t.Fatalf("switch_status = %q, want SWITCHING_TO_FREE", plugin.SwitchStatus)
	}

	var job model.TdaiJob
	if err := model.DB(context.Background()).Where("instance_id = ?", "ins-free-default").First(&job).Error; err != nil {
		t.Fatalf("expected submitted job: %v", err)
	}
	if job.JobType != model.TdaiJobTypeSwitchToFree {
		t.Fatalf("job_type = %q, want SWITCH_TO_FREE", job.JobType)
	}
	if job.Operator != "system:auto_default_plan" {
		t.Fatalf("operator = %q, want system:auto_default_plan", job.Operator)
	}
}

func TestApplyDefaultMemoryPlanForInstance_FallbackLegacyBool(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-legacy-free", AgentType: model.AgentTypeOpenClaw})

	applyDefaultMemoryPlanForInstance(context.Background(), "ins-legacy-free", model.SiteConfig{MemoryTDAIEnable: true})

	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-legacy-free")
	if plugin == nil {
		t.Fatal("expected plugin row")
	}
	if plugin.DesiredPlan != model.MemoryPlanFree {
		t.Fatalf("desired_plan = %q, want FREE", plugin.DesiredPlan)
	}
}

func TestApplyDefaultMemoryPlanForInstance_OffOnlyEnsuresRow(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-off-default", AgentType: model.AgentTypeOpenClaw})

	applyDefaultMemoryPlanForInstance(context.Background(), "ins-off-default", model.SiteConfig{MemoryDefaultPlan: "off"})

	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-off-default")
	if plugin == nil {
		t.Fatal("expected plugin row")
	}
	if plugin.DesiredPlan != model.MemoryPlanOff {
		t.Fatalf("desired_plan = %q, want OFF", plugin.DesiredPlan)
	}
	if plugin.SwitchStatus != "" {
		t.Fatalf("switch_status = %q, want empty", plugin.SwitchStatus)
	}
	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Count(&count)
	if count != 0 {
		t.Fatalf("off default should not submit job, got %d", count)
	}
}

func TestResetMemoryPluginForReinstall_ResubmitsPro(t *testing.T) {
	setupMemoryProDB(t)
	if err := model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:      "ins-pro-reset",
		Status:          model.MemoryTDAIPluginStatusEnabled,
		RetryCount:      2,
		LastError:       "boom",
		CurrentPlan:     model.MemoryPlanPro,
		DesiredPlan:     model.MemoryPlanPro,
		SwitchStatus:    model.MemorySwitchStatusNone,
		PoolID:          "space-001",
		DatabaseName:    "db001",
		Endpoint:        "http://10.0.0.1:3306",
		EmbeddingModel:  "emb",
		VdbUsername:     "root",
		ApiKeySecretRef: "secret",
	}).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	resetMemoryPluginForReinstall(context.Background(), "ins-pro-reset")

	var plugin model.MemoryTDAIPlugin
	if err := model.DB(context.Background()).Where("instance_id = ?", "ins-pro-reset").First(&plugin).Error; err != nil {
		t.Fatalf("reload plugin: %v", err)
	}
	if plugin.Status != model.MemoryTDAIPluginStatusNotInstalled {
		t.Fatalf("status = %q, want NOT_INSTALLED", plugin.Status)
	}
	if plugin.RetryCount != 0 {
		t.Fatalf("retry_count = %d, want 0", plugin.RetryCount)
	}
	if plugin.LastError != "" {
		t.Fatalf("last_error = %q, want empty", plugin.LastError)
	}
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToPro {
		t.Fatalf("switch_status = %q, want SWITCHING_TO_PRO", plugin.SwitchStatus)
	}

	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 resubmitted job, got %d", count)
	}
}

func makeLoggedInJSONRequest(t *testing.T, method, path, body string, user model.User) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")

	seed := httptest.NewRecorder()
	sess, err := Store.Get(req, "hatchery-session")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	sess.Values["username"] = user.Username
	sess.Values["role"] = user.Role
	sess.Values["user_id"] = user.ID
	if err := sess.Save(req, seed); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, c := range seed.Result().Cookies() {
		req.AddCookie(c)
	}
	return req, httptest.NewRecorder()
}
