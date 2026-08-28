package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

type errorStatusResolver struct{ err error }

func (e errorStatusResolver) ResolveStatus(context.Context, *model.Instance) (InstanceStatusResponse, error) {
	return InstanceStatusResponse{}, hcommon.I18nRichError(e.err, i18n.MsgOperationFailed)
}

func newPowerRequest(method, target, contentType, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	return req
}

func TestParsePowerTargetRequest(t *testing.T) {
	tooManyIDs := make([]uint, powerBatchMaxTargets+1)
	for i := range tooManyIDs {
		tooManyIDs[i] = uint(i + 1)
	}

	tests := []struct {
		name      string
		req       *http.Request
		wantBatch bool
		wantID    uint
		wantCVM   string
		wantIDs   []uint
		wantCVMs  []string
		wantErr   bool
	}{
		{
			name:      "json ids dedupe and drop zero",
			req:       newPowerRequest(http.MethodPost, "/x", "application/json", `{"ids":[1,0,2,1]}`),
			wantBatch: true,
			wantIDs:   []uint{1, 2},
		},
		{
			name:      "json instance_ids trim and dedupe",
			req:       newPowerRequest(http.MethodPost, "/x", "application/json", `{"instance_ids":[" ins-a ","","ins-a","ins-b"]}`),
			wantBatch: true,
			wantCVMs:  []string{"ins-a", "ins-b"},
		},
		{
			name:   "json id",
			req:    newPowerRequest(http.MethodPost, "/x", "application/json", `{"id":7}`),
			wantID: 7,
		},
		{
			name:    "json instance_id",
			req:     newPowerRequest(http.MethodPost, "/x", "application/json", `{"instance_id":" ins-json "}`),
			wantCVM: "ins-json",
		},
		{
			name:   "form id",
			req:    newPowerRequest(http.MethodPost, "/x", "application/x-www-form-urlencoded", "id=8"),
			wantID: 8,
		},
		{
			name:    "form instance_id",
			req:     newPowerRequest(http.MethodPost, "/x", "application/x-www-form-urlencoded", "instance_id=ins-form"),
			wantCVM: "ins-form",
		},
		{name: "missing target", req: newPowerRequest(http.MethodPost, "/x", "application/json", `{}`), wantErr: true},
		{name: "invalid form id", req: newPowerRequest(http.MethodPost, "/x", "application/x-www-form-urlencoded", "id=0"), wantErr: true},
		{name: "empty ids", req: newPowerRequest(http.MethodPost, "/x", "application/json", `{"ids":[0,0]}`), wantBatch: true, wantErr: true},
		{name: "empty instance_ids", req: newPowerRequest(http.MethodPost, "/x", "application/json", `{"instance_ids":["", "  "]}`), wantBatch: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := parsePowerTargetRequest(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.IsBatch() != tt.wantBatch || req.ID != tt.wantID || req.InstanceID != tt.wantCVM {
				t.Fatalf("got req=%+v batch=%v", req, req.IsBatch())
			}
			if !uintSlicesEqual(req.IDs, tt.wantIDs) || !stringSlicesEqual(req.InstanceIDs, tt.wantCVMs) {
				t.Fatalf("got IDs=%v CVMs=%v", req.IDs, req.InstanceIDs)
			}
		})
	}

	tooManyIDBody, _ := json.Marshal(map[string][]uint{"ids": tooManyIDs})
	if _, err := parsePowerTargetRequest(newPowerRequest(http.MethodPost, "/x", "application/json", string(tooManyIDBody))); err == nil {
		t.Fatal("expected too many ids error")
	}
	tooManyCVMs := make([]string, powerBatchMaxTargets+1)
	for i := range tooManyCVMs {
		tooManyCVMs[i] = "ins-x"
	}
	tooManyCVMBody, _ := json.Marshal(map[string][]string{"instance_ids": tooManyCVMs})
	if _, err := parsePowerTargetRequest(newPowerRequest(http.MethodPost, "/x", "application/json", string(tooManyCVMBody))); err == nil {
		t.Fatal("expected too many instance_ids error")
	}
}

func TestResolvePowerTargetsBatchHonorsOwnership(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()

	owner := &model.User{Username: "owner", Password: "x"}
	other := &model.User{Username: "other", Password: "x"}
	model.DB(context.Background()).Create(owner)
	model.DB(context.Background()).Create(other)
	owned := &model.Instance{Name: "owned", InstanceId: "ins-owned", UserID: owner.ID}
	foreign := &model.Instance{Name: "foreign", InstanceId: "ins-foreign", UserID: other.ID}
	model.DB(context.Background()).Create(owned)
	model.DB(context.Background()).Create(foreign)

	req := newPowerRequest(http.MethodPost, "/x", "", "")
	byID, err := resolvePowerTargetsByIDs(req, []uint{owned.ID, foreign.ID, 9999}, owner.ID)
	if err != nil || len(byID) != 3 || byID[0].Instance == nil || byID[1].Err == nil || byID[2].Err == nil {
		t.Fatalf("unexpected byID targets: %+v err=%v", byID, err)
	}

	byCVM, err := resolvePowerTargetsByCVMIDs(req, []string{"ins-owned", "ins-foreign", "ins-missing"}, owner.ID)
	if err != nil || len(byCVM) != 3 || byCVM[0].Instance == nil || byCVM[1].Err == nil || byCVM[2].Err == nil {
		t.Fatalf("unexpected byCVM targets: %+v err=%v", byCVM, err)
	}
}

func TestPreparePowerTargetsBatchResults(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()

	user := &model.User{Username: "power-user", Password: "x"}
	model.DB(context.Background()).Create(user)
	doctor := &model.Instance{Name: "doctor", InstanceId: "ins-doctor", UserID: user.ID, IsDoctorNode: true}
	noCVM := &model.Instance{Name: "no-cvm", UserID: user.ID}
	running := &model.Instance{Name: "running", InstanceId: "ins-running", UserID: user.ID}
	stopped := &model.Instance{Name: "stopped", InstanceId: "ins-stopped", UserID: user.ID}
	model.DB(context.Background()).Create(doctor)
	model.DB(context.Background()).Create(noCVM)
	model.DB(context.Background()).Create(running)
	model.DB(context.Background()).Create(stopped)

	req := newPowerRequest(http.MethodPost, "/x", "", "")
	batchResults := func(action string, targets []powerTarget, resolver instanceStatusResolver) ([]powerActionResult, []powerTarget) {
		results := make([]powerActionResult, 0, len(targets))
		readyTargets := make([]powerTarget, 0, len(targets))
		for _, target := range targets {
			if targetErr := validateAndLockPowerTarget(req, action, false, target, resolver); targetErr != nil {
				results = append(results, powerResultFromError(req, powerResultFromTarget(target), targetErr))
			} else {
				readyTargets = append(readyTargets, target)
			}
		}
		return results, readyTargets
	}

	results, readyTargets := batchResults(powerActionStart, []powerTarget{
		{InputID: 999, Err: ErrInstanceNotFound},
		{Instance: doctor},
		{Instance: noCVM},
		{Instance: running},
		{Instance: stopped},
	}, &mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"})
	if got := statuses(results); !stringSlicesEqual(got, []string{"failed", "failed", "failed", "skipped", "skipped"}) || len(readyTargets) != 0 {
		t.Fatalf("unexpected results: statuses=%v ready=%d results=%+v", got, len(readyTargets), results)
	}

	results, readyTargets = batchResults(powerActionStart, []powerTarget{{Instance: stopped}}, stoppedResolver)
	if len(results) != 0 || len(readyTargets) != 1 {
		t.Fatalf("expected one ready start target, results=%+v ready=%+v", results, readyTargets)
	}

	results, readyTargets = batchResults(powerActionStop, []powerTarget{{Instance: running}}, &mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"})
	if len(results) != 0 || len(readyTargets) != 1 {
		t.Fatalf("expected one ready stop target, results=%+v ready=%+v", results, readyTargets)
	}
}

func TestPreparePowerTargetsSingleErrorsAndMarkFailed(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()

	user := &model.User{Username: "single-power", Password: "x"}
	model.DB(context.Background()).Create(user)
	doctor := &model.Instance{Name: "doctor", InstanceId: "ins-doctor", UserID: user.ID, IsDoctorNode: true}
	noCVM := &model.Instance{Name: "no-cvm", UserID: user.ID}
	stopped := &model.Instance{Name: "stopped", InstanceId: "ins-stopped", UserID: user.ID}
	model.DB(context.Background()).Create(doctor)
	model.DB(context.Background()).Create(noCVM)
	model.DB(context.Background()).Create(stopped)

	req := newPowerRequest(http.MethodPost, "/x", "", "")
	cases := []struct {
		name   string
		target powerTarget
		want   int
		res    instanceStatusResolver
	}{
		{"missing", powerTarget{Err: ErrInstanceNotFound}, http.StatusNotFound, stoppedResolver},
		{"doctor", powerTarget{Instance: doctor}, http.StatusBadRequest, stoppedResolver},
		{"no cvm", powerTarget{Instance: noCVM}, http.StatusBadRequest, stoppedResolver},
		{"guard", powerTarget{Instance: stopped}, http.StatusConflict, &mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"}},
		{"infra", powerTarget{Instance: stopped}, http.StatusInternalServerError, errorStatusResolver{err: errors.New("cvm down")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			targetErr := validateAndLockPowerTarget(req, powerActionStart, false, tc.target, tc.res)
			if targetErr == nil || targetErr.HTTPStatus != tc.want {
				t.Fatalf("status=%d err=%v, want status %d", targetErr.HTTPStatus, targetErr.Err, tc.want)
			}
		})
	}

	target := powerTarget{Instance: stopped}
	if targetErr := validateAndLockPowerTarget(req, powerActionStart, false, target, stoppedResolver); targetErr != nil {
		t.Fatalf("expected ready target, got err=%v", targetErr)
	}
	results := powerResultsFromCloudError(req, []powerTarget{target}, errors.New("boom"))
	if len(results) != 1 || results[0].Status != "failed" || !strings.Contains(results[0].Message, "boom") {
		t.Fatalf("expected failed result: %+v", results)
	}
}

// TestValidateAndLockPowerTarget_PendingUserAction 验证：带 pending_user_action
// 标的实例，用户端 start 被 409 拒绝，管理端不受此限制。
func TestValidateAndLockPowerTarget_PendingUserAction(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).AutoMigrate(&model.InstanceFlag{}); err != nil {
		t.Fatalf("automigrate InstanceFlag: %v", err)
	}

	user := &model.User{Username: "pending-user", Password: "x"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{Name: "pending", InstanceId: "ins-pending", UserID: user.ID}
	model.DB(context.Background()).Create(inst)
	if err := model.AddInstanceFlag(context.Background(), inst.ID, model.InstanceFlagPendingUserAction, ""); err != nil {
		t.Fatalf("add flag: %v", err)
	}

	req := newPowerRequest(http.MethodPost, "/x", "", "")
	target := powerTarget{Instance: inst}

	// 用户端 start：拒绝
	if got := validateAndLockPowerTarget(req, powerActionStart, false, target, stoppedResolver); got == nil ||
		got.HTTPStatus != http.StatusConflict {
		t.Fatalf("user start should be blocked with 409, got %+v", got)
	}

	// 用户端 stop：允许（只拦 start）
	if got := validateAndLockPowerTarget(req, powerActionStop, false, target,
		&mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"}); got != nil {
		t.Fatalf("user stop should NOT be blocked, got %+v", got)
	}

	// 管理端 start：允许
	if got := validateAndLockPowerTarget(req, powerActionStart, true, target, stoppedResolver); got != nil {
		t.Fatalf("admin start should NOT be blocked, got %+v", got)
	}
}

// TestValidateAndLockPowerTarget_StaleGroup 验证：带 stale_group 标的实例，
// 用户端 start 被 409 拒绝，管理端不受此限制。
func TestValidateAndLockPowerTarget_StaleGroup(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).AutoMigrate(&model.InstanceFlag{}); err != nil {
		t.Fatalf("automigrate InstanceFlag: %v", err)
	}

	user := &model.User{Username: "stale-user", Password: "x"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{Name: "stale", InstanceId: "ins-stale", UserID: user.ID}
	model.DB(context.Background()).Create(inst)
	if err := model.AddInstanceFlag(context.Background(), inst.ID, model.InstanceFlagStaleGroup, ""); err != nil {
		t.Fatalf("add flag: %v", err)
	}

	req := newPowerRequest(http.MethodPost, "/x", "", "")
	target := powerTarget{Instance: inst}

	// 用户端 start：拒绝（stale_group 命中）
	if got := validateAndLockPowerTarget(req, powerActionStart, false, target, stoppedResolver); got == nil ||
		got.HTTPStatus != http.StatusConflict {
		t.Fatalf("user start should be blocked with 409 for stale_group, got %+v", got)
	}

	// 用户端 stop：允许（stale_group 只拦 start）
	if got := validateAndLockPowerTarget(req, powerActionStop, false, target,
		&mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"}); got != nil {
		t.Fatalf("user stop should NOT be blocked, got %+v", got)
	}

	// 管理端 start：允许（管理端不受 stale_group 限制）
	if got := validateAndLockPowerTarget(req, powerActionStart, true, target, stoppedResolver); got != nil {
		t.Fatalf("admin start should NOT be blocked by stale_group, got %+v", got)
	}
}

// TestClearStaleFlagsOnStartedTargets 验证 admin 开机成功后清 stale-instances 全部标：
// - status="started" 的实例：4 个 stale flag 全部被清除
// - status!="started" 的实例：全部 flag 保留
func TestClearStaleFlagsOnStartedTargets(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).AutoMigrate(&model.InstanceFlag{}); err != nil {
		t.Fatalf("automigrate InstanceFlag: %v", err)
	}

	ctx := context.Background()
	user := &model.User{Username: "u", Password: "x"}
	model.DB(ctx).Create(user)

	instStarted := &model.Instance{Name: "a", InstanceId: "ins-a", UserID: user.ID}
	instFailed := &model.Instance{Name: "b", InstanceId: "ins-b", UserID: user.ID}
	model.DB(ctx).Create(instStarted)
	model.DB(ctx).Create(instFailed)

	// 两个实例都打上全部 4 个 stale flag
	allFlags := []string{
		model.InstanceFlagStaleGroup,
		model.InstanceFlagPendingUserAction,
		model.InstanceFlagAllowMigrate,
		model.InstanceFlagAllowSameGroupHandover,
	}
	for _, id := range []uint{instStarted.ID, instFailed.ID} {
		for _, f := range allFlags {
			_ = model.AddInstanceFlag(ctx, id, f, "")
		}
	}

	targets := []powerTarget{{Instance: instStarted}, {Instance: instFailed}}
	results := []powerActionResult{
		{Status: "started"},
		{Status: "failed"},
	}
	clearStaleFlagsOnStartedTargets(ctx, targets, results)

	// A: 4 个 flag 全清
	for _, f := range allFlags {
		if ok, _ := model.HasInstanceFlag(ctx, instStarted.ID, f); ok {
			t.Errorf("started instance: flag %q should be cleared", f)
		}
	}

	// B: 4 个 flag 全保留
	for _, f := range allFlags {
		if ok, _ := model.HasInstanceFlag(ctx, instFailed.ID, f); !ok {
			t.Errorf("failed instance: flag %q should be preserved", f)
		}
	}
}

func TestClearAdjustmentFailuresOnSuccessfulPowerTargets(t *testing.T) {
	cleanup := initGuardTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db := model.DB(ctx)
	user := &model.User{Username: "adjustment-failure-power", Password: "x"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	started := &model.Instance{Name: "started", InstanceId: "ins-started", UserID: user.ID}
	failed := &model.Instance{Name: "failed", InstanceId: "ins-failed", UserID: user.ID}
	if err := db.Create(started).Error; err != nil {
		t.Fatalf("create started instance: %v", err)
	}
	if err := db.Create(failed).Error; err != nil {
		t.Fatalf("create failed instance: %v", err)
	}
	for _, instance := range []*model.Instance{started, failed} {
		adjustment := &model.InstanceAdjustment{
			InstanceID: instance.ID,
			Status:     adjustmentStatusFailed,
			Type:       adjustmentTypeInstanceType,
		}
		if err := db.Create(adjustment).Error; err != nil {
			t.Fatalf("create failed adjustment for %s: %v", instance.Name, err)
		}
	}

	clearAdjustmentFailuresOnSuccessfulPowerTargets(ctx,
		[]powerTarget{{Instance: started}, {Instance: failed}},
		[]powerActionResult{{Status: "started"}, {Status: "failed"}},
	)

	var remaining []model.InstanceAdjustment
	if err := db.Order("instance_id").Find(&remaining).Error; err != nil {
		t.Fatalf("query remaining adjustments: %v", err)
	}
	if len(remaining) != 1 || remaining[0].InstanceID != failed.ID {
		t.Fatalf("remaining adjustments=%+v, want only failed power target", remaining)
	}
}

func uintSlicesEqual(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func statuses(results []powerActionResult) []string {
	out := make([]string, len(results))
	for i := range results {
		out[i] = results[i].Status
	}
	return out
}
