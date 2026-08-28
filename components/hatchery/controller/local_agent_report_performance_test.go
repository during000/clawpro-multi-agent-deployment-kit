package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/model"

	"gorm.io/gorm/logger"
)

type reportSQLCounter struct {
	logger.Interface
	count atomic.Int64
}

func (c *reportSQLCounter) LogMode(logger.LogLevel) logger.Interface { return c }

func (c *reportSQLCounter) Trace(context.Context, time.Time, func() (string, int64), error) {
	c.count.Add(1)
}

// TestHandleLocalAgentReport_BatchesReportedAssets 保护 report 的批量写入边界：
// 12 个上报资产不得退化为按资产逐条 INSERT/SELECT 的几十条 SQL。
func TestHandleLocalAgentReport_BatchesReportedAssets(t *testing.T) {
	db := setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	user := model.User{Username: "report-batch-assets", Role: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	counter := &reportSQLCounter{Interface: logger.Default.LogMode(logger.Silent)}
	oldLogger := db.Config.Logger
	db.Config.Logger = counter
	t.Cleanup(func() { db.Config.Logger = oldLogger })
	body := map[string]any{
		"agent_type": "codebuddy", "local_agent_id": "1234567890abcdef",
		"user_level": map[string]any{
			"skills": []map[string]any{{"slug": "s1"}, {"slug": "s2"}, {"slug": "s3"}, {"slug": "s4"}, {"slug": "s5"}, {"slug": "s6"}},
			"rules":  []map[string]any{{"slug": "r1"}, {"slug": "r2"}, {"slug": "r3"}},
		},
		"workspaces": []map[string]any{{
			"path": "/workspace", "name": "workspace", "ide_type": "codebuddy",
			"skills": []map[string]any{{"slug": "ws1"}},
			"rules":  []map[string]any{{"slug": "wr1"}, {"slug": "wr2"}},
		}},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, user.Username, body))
	if rr.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%s", rr.Code, rr.Body.String())
	}
	count := counter.count.Load()
	t.Logf("12 个资产的 report SQL=%d", count)
	if count > 24 {
		t.Fatalf("12 个资产的 report SQL=%d，批量写入退化", count)
	}
}

// TestHandleLocalAgentReport_BatchesWorkspaceSnapshots 防止 Workspace 数增加时，
// 对每个路径重复查询 skill/rule 快照和逐条写 scope binding。
func TestHandleLocalAgentReport_BatchesWorkspaceSnapshots(t *testing.T) {
	db := setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	user := model.User{Username: "report-batch-workspaces", Role: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	counter := &reportSQLCounter{Interface: logger.Default.LogMode(logger.Silent)}
	oldLogger := db.Config.Logger
	db.Config.Logger = counter
	t.Cleanup(func() { db.Config.Logger = oldLogger })
	body := map[string]any{
		"agent_type": "codebuddy", "local_agent_id": "fedcba0987654321",
		"workspaces": []map[string]any{
			{"path": "/workspace/a", "skills": []map[string]any{{"slug": "a-s1"}, {"slug": "a-s2"}}, "rules": []map[string]any{{"slug": "a-r1"}}},
			{"path": "/workspace/b", "skills": []map[string]any{{"slug": "b-s1"}, {"slug": "b-s2"}}, "rules": []map[string]any{{"slug": "b-r1"}}},
			{"path": "/workspace/c", "skills": []map[string]any{{"slug": "c-s1"}, {"slug": "c-s2"}}, "rules": []map[string]any{{"slug": "c-r1"}}},
		},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, user.Username, body))
	if rr.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%s", rr.Code, rr.Body.String())
	}
	count := counter.count.Load()
	t.Logf("3 个 workspace、9 个资产的 report SQL=%d", count)
	if count > 24 {
		t.Fatalf("3 个 workspace report SQL=%d，workspace 批处理退化", count)
	}
}
