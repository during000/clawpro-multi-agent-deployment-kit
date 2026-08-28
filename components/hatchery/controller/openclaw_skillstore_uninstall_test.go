package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

func seedSkillStoreUninstallUser(t *testing.T, username string) model.User {
	t.Helper()
	user := model.User{Username: username, Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user
}

func seedSkillStoreUninstallSkill(t *testing.T, slug string, visibility string) model.Skill {
	t.Helper()
	skill := model.Skill{
		Slug: slug, Name: "用户端卸载测试技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: visibility,
	}
	if err := model.DB(context.Background()).Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	return skill
}

func seedSkillStoreUninstallInstance(t *testing.T, user model.User, name, cid string) model.Instance {
	t.Helper()
	inst := model.Instance{
		Name: name, InstanceId: cid, UserID: user.ID,
		LastCVMState: "RUNNING", AgentReady: 1,
		AgentType: "openclaw", RuntimeUser: "root",
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return inst
}

func TestHandleSkillStoreUninstall_Unauthorized(t *testing.T) {
	setupSkillStoreTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/skillstore/uninstall", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleSkillStoreUninstall(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应返回 401，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSkillStoreUninstall_ForeignInstance(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := seedSkillStoreUninstallUser(t, "uninstall-owner")
	other := seedSkillStoreUninstallUser(t, "uninstall-other")
	skill := seedSkillStoreUninstallSkill(t, "store-uninstall-foreign", model.VisibilityAll)
	otherInst := seedSkillStoreUninstallInstance(t, other, "别人实例", "ins-store-foreign")

	req := skillStorePostReq(t, "/openclaw/skillstore/uninstall", user.Username,
		`{"slug":"`+skill.Slug+`","instance_ids":[`+uintStr(otherInst.ID)+`]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreUninstall(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("卸载他人实例应返回 403，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSkillStoreUninstall_InvisibleSkill(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := seedSkillStoreUninstallUser(t, "uninstall-invisible-user")
	skill := seedSkillStoreUninstallSkill(t, "store-uninstall-secret", model.VisibilityGroup)
	group := model.UserGroup{Name: "secret-group"}
	if err := model.DB(context.Background()).Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatalf("创建技能可见性关联失败: %v", err)
	}
	inst := seedSkillStoreUninstallInstance(t, user, "我的实例", "ins-store-invisible")

	req := skillStorePostReq(t, "/openclaw/skillstore/uninstall", user.Username,
		`{"slug":"`+skill.Slug+`","instance_ids":[`+uintStr(inst.ID)+`]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreUninstall(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("不可见技能应返回 404，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSkillStoreUninstall_BasicFlow(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := seedSkillStoreUninstallUser(t, "uninstall-basic-store-user")
	skill := seedSkillStoreUninstallSkill(t, "store-uninstall-basic", model.VisibilityAll)
	inst := seedSkillStoreUninstallInstance(t, user, "我的实例", "ins-store-uninstall-basic")

	req := skillStorePostReq(t, "/openclaw/skillstore/uninstall", user.Username,
		`{"slug":"`+skill.Slug+`","instance_ids":[`+uintStr(inst.ID)+`]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreUninstall(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if resp["task_id"] == nil {
		t.Fatalf("响应缺少 task_id: %v", resp)
	}
	waitSkillTaskAsync(t)

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).Where("skill_id = ?", skill.ID).First(&task).Error; err != nil {
		t.Fatalf("未创建卸载任务: %v", err)
	}
	if task.Type != model.TaskTypeUninstall {
		t.Fatalf("Task.Type=%q，期望 %q", task.Type, model.TaskTypeUninstall)
	}
	if task.OperatorID != user.ID {
		t.Fatalf("OperatorID=%d，期望当前用户 ID=%d", task.OperatorID, user.ID)
	}
	if task.Total != 1 {
		t.Fatalf("Task.Total=%d，期望 1", task.Total)
	}

	var record model.SkillDistributionRecord
	if err := model.DB(context.Background()).Where("task_id = ? AND instance_id = ?", task.ID, inst.ID).First(&record).Error; err != nil {
		t.Fatalf("未创建卸载记录: %v", err)
	}
	if record.Type != model.TaskTypeUninstall {
		t.Fatalf("Record.Type=%q，期望 %q", record.Type, model.TaskTypeUninstall)
	}
}
