package controller

// 本地 Agent report/sync 并发死锁修复的专项单测。
//
// 背景：并发 report/sync 同一实例时，sync 事务先写 local_agent_scope_bindings
// 再更新 instances，与 report 事务（先 UPDATE instances 再 upsert scope binding）
// 加锁顺序相反，在 MySQL 下形成 AB-BA 死锁（生产 LATEST DETECTED DEADLOCK）。
//
// 修复：processSyncWorkspaces 事务开头对实例行加 FOR UPDATE 排它锁 + report
// handler 查询实例时同样 FOR UPDATE，统一 instances -> scope_bindings 加锁顺序。
//
// 说明：SQLite 不支持 SELECT ... FOR UPDATE（GORM 会忽略 clause.Locking），
// 单测无法复现 MySQL 行锁死锁，故本文件聚焦两点：
//  1. 修复引入的行为变化 —— sync 从"锁定的最新实例行"反序列化 resources，
//     不再基于事务外旧快照计算（旧实现会丢失并发 report 已写入的 project_id）。
//  2. 并发 report/sync 同一实例的逻辑正确性（配合 go test -race 验证无数据竞争）。

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"hatchery/model"
)

// TestProcessSyncWorkspaces_ReadsLatestInstanceSnapshot
// 覆盖修复后 processSyncWorkspaces 从"FOR UPDATE 锁定的最新实例行"反序列化
// local_agent_resources 的行为：
//   - DB 中实例已有并发 report 写入的 workspace（含 project_id）
//   - 传入的 inst 是事务外读到的旧快照（LocalAgentResources 为空）
//   - sync 上报同一 path 且不带 project_id（应保留旧值）
//
// 期望：最终 resources 中该 workspace 的 project_id 保留 DB 最新值。
// 旧实现基于传入旧快照计算，会丢失 project_id（降级为 0）。
func TestProcessSyncWorkspaces_ReadsLatestInstanceSnapshot(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "sync-latest-snapshot", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := model.Project{Name: "sync-latest-snapshot", SyncMode: "continuous"}
	if err := model.DB(ctx).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	// DB 最新值：模拟并发 report 已把 /latest 绑定到 project
	inst := model.Instance{
		Name: "sync-latest-snapshot", InstanceId: "sync-latest-001", UserID: user.ID,
		Source: model.InstanceSourceLocal,
		LocalAgentResources: &model.LocalAgentResources{
			Workspaces: []model.WorkspaceResource{{Path: "/latest", Name: "latest", IDEType: "workbuddy", ProjectID: project.ID}},
		},
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// 传入旧快照：模拟 sync handler 在事务外读到的过期数据（不含 /latest 的 project_id）
	staleInst := inst
	staleInst.LocalAgentResources = nil

	// sync 上报 /latest，不带 project_id（保持旧值）
	if err := processSyncWorkspaces(ctx, &staleInst, &user,
		[]syncWorkspace{{Path: "/latest", Name: "latest", IDEType: "workbuddy"}}); err != nil {
		t.Fatalf("process sync workspaces: %v", err)
	}

	var after model.Instance
	if err := model.DB(ctx).First(&after, inst.ID).Error; err != nil {
		t.Fatalf("query instance: %v", err)
	}
	res := deserializeLocalAgentResources(after.LocalAgentResources)
	if res == nil {
		t.Fatalf("local_agent_resources 不应为空")
	}
	found := false
	for _, ws := range res.Workspaces {
		if ws.Path == "/latest" {
			found = true
			if ws.ProjectID != project.ID {
				t.Errorf("/latest 应从 DB 最新行保留 project_id=%d，实际=%d（旧快照导致丢失）",
					project.ID, ws.ProjectID)
			}
		}
	}
	if !found {
		t.Fatalf("sync 上报的 workspace /latest 未写入 resources，实际=%+v", res.Workspaces)
	}
}

// TestLocalAgentReportSync_ConcurrentSameInstance
// 并发（交替 report + sync）打同一实例，验证：
//   - 所有请求均 200，不报死锁/内部错误
//   - 最终实例 local_agent_resources 与 scope_bindings 数据一致
//
// SQLite 单连接下事务天然串行，本测试主要用于配合 -race 验证并发无数据竞争，
// MySQL 下的真实行锁死锁复现由本地 MySQL 并发压测脚本完成。
func TestLocalAgentReportSync_ConcurrentSameInstance(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "concurrent-local-agent", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 预建实例，避免并发首报时 Create 竞争。
	// 注意：handler 会把 local_agent_id 派生为 instance_id（formatLocalInstanceID），
	// 实例的 InstanceId 必须用派生值，否则 report 走新建分支。
	const agentID = "abcdef0000000001"
	instID := formatLocalInstanceID("workbuddy", agentID)
	inst := model.Instance{Name: "concurrent", InstanceId: instID, UserID: user.ID, Source: model.InstanceSourceLocal}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// 串行预验证：单个 report 应能写入 workspaces（排除 body/字段解析问题）
	// 用独立 request，避免消费下面并发循环的 reqs body。
	probeReq := reportReq(t, "concurrent-local-agent", map[string]any{
		"agent_type":     "workbuddy",
		"local_agent_id": agentID,
		"host_name":      "host-probe",
		"workspaces": []map[string]any{
			{"path": "/ws-probe", "name": "ws-probe", "ide_type": "workbuddy"},
		},
	})
	rrProbe := httptest.NewRecorder()
	HandleLocalAgentReport(rrProbe, probeReq)
	if rrProbe.Code != http.StatusOK {
		t.Fatalf("串行 report 预验证失败: status=%d body=%s", rrProbe.Code, rrProbe.Body.String())
	}
	var probe model.Instance
	if err := model.DB(ctx).First(&probe, inst.ID).Error; err != nil {
		t.Fatalf("probe instance: %v", err)
	}
	if len(deserializeLocalAgentResources(probe.LocalAgentResources).Workspaces) == 0 {
		t.Fatalf("单个 report 未写入 workspaces, local_agent_resources=%v", probe.LocalAgentResources)
	}
	t.Logf("串行 report 预验证通过: workspaces=%v", probe.LocalAgentResources)

	const n = 20
	reqs := make([]*http.Request, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			reqs[i] = reportReq(t, "concurrent-local-agent", map[string]any{
				"agent_type":     "workbuddy",
				"local_agent_id": agentID,
				"host_name":      fmt.Sprintf("host-%d", i),
				"workspaces": []map[string]any{
					{"path": fmt.Sprintf("/ws-%d", i), "name": fmt.Sprintf("ws-%d", i), "ide_type": "workbuddy"},
				},
			})
		} else {
			reqs[i] = syncReq(t, "concurrent-local-agent", map[string]any{
				"agent_type":     "workbuddy",
				"local_agent_id": agentID,
				"status":         "running",
				"workspaces": []map[string]any{
					{"path": fmt.Sprintf("/ws-%d", i), "name": fmt.Sprintf("ws-%d", i), "ide_type": "workbuddy"},
				},
			})
		}
	}

	start := time.Now()
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(r *http.Request, idx int) {
			defer wg.Done()
			rr := httptest.NewRecorder()
			if idx%2 == 0 {
				HandleLocalAgentReport(rr, r)
			} else {
				HandleLocalAgentSync(rr, r)
			}
			if rr.Code != http.StatusOK {
				errCh <- fmt.Errorf("req %d: status=%d body=%s", idx, rr.Code, rr.Body.String())
			}
		}(reqs[i], i)
	}
	wg.Wait()
	close(errCh)
	elapsed := time.Since(start)

	hasErr := false
	for err := range errCh {
		hasErr = true
		t.Errorf("%v", err)
	}
	if hasErr {
		t.FailNow()
	}

	// 最终一致性：实例 resources 应包含并发上报的某 workspace；scope_bindings 可查
	var after model.Instance
	if err := model.DB(ctx).First(&after, inst.ID).Error; err != nil {
		t.Fatalf("query instance: %v", err)
	}
	res := deserializeLocalAgentResources(after.LocalAgentResources)
	if res == nil || len(res.Workspaces) == 0 {
		t.Fatalf("并发后 resources 不应为空")
	}
	var bindCount int64
	if err := model.DB(ctx).Model(&model.LocalAgentScopeBinding{}).
		Where("instance_id = ? AND scope = ?", inst.ID, model.LocalAgentScopeWorkspace).
		Count(&bindCount).Error; err != nil {
		t.Fatalf("count scope bindings: %v", err)
	}
	if bindCount == 0 {
		t.Fatalf("并发后 workspace scope_bindings 不应为空")
	}
	t.Logf("并发 %d 个 report/sync 请求完成，耗时 %v，workspaces=%d，bindings=%d",
		n, elapsed, len(res.Workspaces), bindCount)
}
