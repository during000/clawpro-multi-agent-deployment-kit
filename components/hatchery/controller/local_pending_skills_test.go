package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

func pendingSkillsRequest(t *testing.T, method, username string, instanceID, recordID uint) *http.Request {
	t.Helper()
	path := fmt.Sprintf("/openclaw/local/pending-skills?id=%d", instanceID)
	if recordID != 0 {
		path += fmt.Sprintf("&record_id=%d", recordID)
	}
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func TestHandleLocalPendingSkillsRouter_ListAndDelete(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, instance := seedLocalCommandsFixture(t)
	task := model.SkillDistributionTask{
		Slug: "pending-skill", Type: model.TaskTypeDistribute,
		Status: model.TaskStatusRunning, Total: 1, OperatorID: user.ID,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, InstanceID: instance.ID, InstanceCID: instance.InstanceId,
		Type: model.TaskTypeDistribute, Status: model.RecordStatusPending,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleLocalPendingSkillsRouter(rr, pendingSkillsRequest(t, http.MethodGet, user.Username, instance.ID, 0))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", rr.Code, rr.Body.String())
	}
	var listResp struct {
		Total  int `json:"total"`
		Skills []struct {
			RecordID uint `json:"record_id"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if listResp.Total != 1 || len(listResp.Skills) != 1 || listResp.Skills[0].RecordID != record.ID {
		t.Fatalf("unexpected GET response: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	HandleLocalPendingSkillsRouter(rr, pendingSkillsRequest(t, http.MethodDelete, user.Username, instance.ID, record.ID))
	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE status=%d body=%s", rr.Code, rr.Body.String())
	}
	var count int64
	if err := model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("id = ?", record.ID).Count(&count).Error; err != nil {
		t.Fatalf("count record: %v", err)
	}
	if count != 0 {
		t.Fatalf("record %d was not deleted", record.ID)
	}
}

func TestHandleLocalPendingSkillsRouter_MethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleLocalPendingSkillsRouter(rr, httptest.NewRequest(http.MethodPut, "/openclaw/local/pending-skills", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
