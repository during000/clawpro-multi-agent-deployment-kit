package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"gorm.io/gorm/logger"
)

func TestProfileLocalAgentReportChangedAssets(t *testing.T) {
	db := setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	user := model.User{Username: "report-profile-changed", Role: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	counter := &reportSQLCounter{Interface: logger.Default.LogMode(logger.Silent)}
	oldLogger := db.Config.Logger
	db.Config.Logger = counter
	t.Cleanup(func() { db.Config.Logger = oldLogger })
	body := profileChangedReportBody()
	callProfileReport(t, user.Username, body)
	counter.count.Store(0)
	setProfileVersions(body, "2.0.0")
	callProfileReport(t, user.Username, body)
	count := counter.count.Load()
	t.Logf("12 项全量变更的 report SQL=%d", count)
	if count > 20 {
		t.Fatalf("12 项全量变更 report SQL=%d，批量更新退化", count)
	}
}

func callProfileReport(t *testing.T, username string, body map[string]any) {
	t.Helper()
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, username, body))
	if rr.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func profileChangedReportBody() map[string]any {
	return map[string]any{
		"agent_type": "codebuddy", "local_agent_id": "0123456789abcdef",
		"user_level": map[string]any{
			"skills": []map[string]any{{"slug": "s1"}, {"slug": "s2"}, {"slug": "s3"}, {"slug": "s4"}, {"slug": "s5"}, {"slug": "s6"}},
			"rules":  []map[string]any{{"slug": "r1"}, {"slug": "r2"}, {"slug": "r3"}},
		},
		"workspaces": []map[string]any{{
			"path": "/workspace", "skills": []map[string]any{{"slug": "ws1"}}, "rules": []map[string]any{{"slug": "wr1"}, {"slug": "wr2"}},
		}},
	}
}

func setProfileVersions(body map[string]any, version string) {
	userLevel := body["user_level"].(map[string]any)
	for _, key := range []string{"skills", "rules"} {
		for _, item := range userLevel[key].([]map[string]any) {
			item["version"] = version
		}
	}
	workspace := body["workspaces"].([]map[string]any)[0]
	for _, key := range []string{"skills", "rules"} {
		for _, item := range workspace[key].([]map[string]any) {
			item["version"] = version
		}
	}
}
