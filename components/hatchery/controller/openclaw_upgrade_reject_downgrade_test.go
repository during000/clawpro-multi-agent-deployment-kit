package controller

// 本文件覆盖 /openclaw/upgrade 入口新增的"官方镜像降级拦截"前置检查。
// 同时包含：
//  1. rejectDowngradeOnOfficialImage 纯函数级单元测试（覆盖所有早返回分支 + 命中拒绝分支）
//  2. handleUpgrade 入口集成测试（验证命中后真实返回 400 + 不进入异步升级流程）

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"
)

// ─── rejectDowngradeOnOfficialImage 单元测试 ──────────────────────────────────
//
// 函数语义：覆盖官方镜像 + 自定义启用镜像两类来源，凡 OpenClaw 实例当前版本严格高于
// 目标镜像版本即拒绝；错误文案根据 IsCandidateImage 区分官方/自定义。
//
// 函数返回 nil 表示放行；返回非 nil error 表示拒绝。覆盖分支：
//   1. instance 为 nil → 放行
//   2. defaultImage 为 nil → 放行
//   3. 实例运行时类型非 OpenClaw → 放行
//   4. 镜像声明的 agent_version 为空 → 放行
//   5. 实例 agent_version 为空（无法判定）→ 放行
//   6. 实例版本 < 镜像版本（正向升级）→ 放行
//   7. 实例版本 == 镜像版本（不构成高于）→ 放行
//   8. 实例版本 > 官方镜像版本 → 命中拒绝（旧文案）
//   9. 实例版本 > 自定义镜像版本 → 命中拒绝（新文案）

// pickOfficialOpenClawCandidate 取一个官方候选 OpenClaw 镜像，作为测试目标镜像。
// 候选镜像列表硬编码于 common/image.go，OpenClaw 类型至少存在 1 个。
func pickOfficialOpenClawCandidate(t *testing.T) hcommon.CandidateImage {
	t.Helper()
	for _, c := range hcommon.CandidateImages {
		if c.AgentType == model.AgentTypeOpenClaw {
			return c
		}
	}
	t.Fatalf("候选镜像列表中未找到 OpenClaw 类型镜像")
	return hcommon.CandidateImage{}
}

func TestRejectDowngradeOnOfficialImage_NilInstance(t *testing.T) {
	if err := rejectDowngradeOnOfficialImage(context.Background(), nil, &model.AIImage{}); err != nil {
		t.Errorf("instance=nil 应放行，实际返回错误: %v", err)
	}
}

func TestRejectDowngradeOnOfficialImage_NilImage(t *testing.T) {
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2099.1.1"}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, nil); err != nil {
		t.Errorf("defaultImage=nil 应放行，实际返回错误: %v", err)
	}
}

// 自定义镜像（非候选）且实例版本严格高于镜像版本：扩展后的语义要求拒绝，
// 且使用专门的自定义镜像文案而非官方镜像文案。
func TestRejectDowngradeOnOfficialImage_CustomImageHigherRejected(t *testing.T) {
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2099.12.31"}
	img := &model.AIImage{ImageId: "img-private-xxxx", AgentVersion: "2026.1.1"}
	// 防御性确认：随机 ID 不在候选列表
	if hcommon.IsCandidateImage(img.ImageId) {
		t.Fatalf("用例前提失败：img-private-xxxx 不应是候选镜像")
	}
	err := rejectDowngradeOnOfficialImage(context.Background(), inst, img)
	if err == nil {
		t.Fatal("自定义镜像降级也应拒绝（覆盖管理员配置错误场景）")
	}
	msg := err.Error()
	// 文案应同时包含两个版本号
	if !strings.Contains(msg, "2099.12.31") || !strings.Contains(msg, "2026.1.1") {
		t.Errorf("错误文案应同时包含当前版本与目标版本，实际: %s", msg)
	}
	// 自定义镜像必须命中新文案，而不是官方镜像文案
	if strings.Contains(msg, "官方镜像") || strings.Contains(msg, "official") {
		t.Errorf("自定义镜像不应使用官方镜像文案，实际: %s", msg)
	}
	if !strings.Contains(msg, "自定义") && !strings.Contains(msg, "custom") {
		t.Errorf("自定义镜像应使用自定义镜像文案，实际: %s", msg)
	}
}

// 自定义镜像版本 == 实例版本：不构成高于，应放行
func TestRejectDowngradeOnOfficialImage_CustomImageEqual(t *testing.T) {
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.1.1"}
	img := &model.AIImage{ImageId: "img-private-yyyy", AgentVersion: "2026.1.1"}
	if hcommon.IsCandidateImage(img.ImageId) {
		t.Fatalf("用例前提失败：img-private-yyyy 不应是候选镜像")
	}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("自定义镜像版本一致应放行，实际返回错误: %v", err)
	}
}

// 自定义镜像版本 > 实例版本：正向升级，应放行
func TestRejectDowngradeOnOfficialImage_CustomImageLower(t *testing.T) {
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2024.1.1"}
	img := &model.AIImage{ImageId: "img-private-zzzz", AgentVersion: "2026.1.1"}
	if hcommon.IsCandidateImage(img.ImageId) {
		t.Fatalf("用例前提失败：img-private-zzzz 不应是候选镜像")
	}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("自定义镜像版本高于实例版本应放行，实际返回错误: %v", err)
	}
}

// 非 OpenClaw 类型（如 hermes）应直接放行：本次保护仅覆盖 OpenClaw
func TestRejectDowngradeOnOfficialImage_NonOpenClawRuntimeType(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: model.AgentTypeHermes, AgentVersion: "9.9.9"}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: cand.AgentVersion}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("非 OpenClaw 运行时类型应放行，实际返回错误: %v", err)
	}
}

// 镜像 agent_version 为空（存量镜像）应放行
func TestRejectDowngradeOnOfficialImage_EmptyImageVersion(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2099.1.1"}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: "   "} // 空白
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("镜像 agent_version 为空应放行，实际返回错误: %v", err)
	}
}

// 实例 agent_version 为空（采集失败/存量）应放行交给后续流程
func TestRejectDowngradeOnOfficialImage_EmptyInstanceVersion(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: ""}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: cand.AgentVersion}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("实例 agent_version 为空应放行，实际返回错误: %v", err)
	}
}

// 实例版本低于官方镜像版本（正向升级）应放行
func TestRejectDowngradeOnOfficialImage_VersionLower(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2024.1.1"}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: cand.AgentVersion}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("实例版本低于镜像版本应放行，实际返回错误: %v", err)
	}
}

// 实例版本等于官方镜像版本应放行（不构成"高于"）
func TestRejectDowngradeOnOfficialImage_VersionEqual(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: cand.AgentVersion}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: cand.AgentVersion}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err != nil {
		t.Errorf("实例版本等于镜像版本应放行，实际返回错误: %v", err)
	}
}

// 实例版本严格高于官方镜像版本 → 命中拒绝（沿用官方镜像文案）
func TestRejectDowngradeOnOfficialImage_VersionHigherRejected(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, AgentVersion: "2099.12.31"}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: cand.AgentVersion}
	err := rejectDowngradeOnOfficialImage(context.Background(), inst, img)
	if err == nil {
		t.Fatal("实例版本高于官方镜像版本时必须返回拒绝错误")
	}
	msg := err.Error()
	// 错误文案应同时包含两个版本号，便于用户定位
	if !strings.Contains(msg, "2099.12.31") || !strings.Contains(msg, cand.AgentVersion) {
		t.Errorf("错误文案应同时包含当前版本与目标版本，实际: %s", msg)
	}
	// 官方镜像必须命中官方镜像文案
	if !strings.Contains(msg, "官方镜像") && !strings.Contains(msg, "official") {
		t.Errorf("官方镜像应使用官方镜像文案，实际: %s", msg)
	}
}

// AgentType 为空的存量实例应等价于 OpenClaw，命中相同的拒绝逻辑
func TestRejectDowngradeOnOfficialImage_EmptyAgentTypeRejected(t *testing.T) {
	cand := pickOfficialOpenClawCandidate(t)
	inst := &model.Instance{AgentType: "", AgentVersion: "2099.1.1"}
	img := &model.AIImage{ImageId: cand.ImageId, AgentVersion: cand.AgentVersion}
	if err := rejectDowngradeOnOfficialImage(context.Background(), inst, img); err == nil {
		t.Error("AgentType 为空的存量实例应被视为 OpenClaw，并按更高版本命中拒绝")
	}
}

// ─── handleUpgrade 入口集成测试 ────────────────────────────────────────────
//
// 复用 setupUpgradeExtraEnv + loggedInReq 工具，验证拒绝路径在真实 handler 中：
//  - 返回 400 BadRequest
//  - 响应体 error 字段包含两个版本号
//  - 实例 CurrentOperation 不被设置（说明未进入 setOperation/异步升级流程）

// TestHandleUpgrade_RejectDowngrade_OfficialImageAndOpenclawHigher 命中拒绝路径。
func TestHandleUpgrade_RejectDowngrade_OfficialImageAndOpenclawHigher(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-reject-downgrade")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.AgentVersion = "2099.12.31" // 远高于任何官方版本
	})

	// 选用一个官方候选 OpenClaw 镜像，并把镜像版本设为"较低"
	cand := pickOfficialOpenClawCandidate(t)
	img := &model.AIImage{
		ImageId:      cand.ImageId,
		Enabled:      true,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: cand.AgentVersion,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("创建启用镜像失败: %v", err)
	}

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id="+itoaUint(inst.ID), "upgrade-reject-downgrade", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("命中降级拒绝应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "2099.12.31") || !strings.Contains(errMsg, cand.AgentVersion) {
		t.Errorf("响应应同时包含当前版本与目标版本，实际错误: %s", errMsg)
	}

	// 关键副作用校验：拒绝阶段早于 setOperation，不应留下进行中的升级锁
	var fresh model.Instance
	if err := model.DB(context.Background()).First(&fresh, inst.ID).Error; err != nil {
		t.Fatalf("重新加载实例失败: %v", err)
	}
	if fresh.CurrentOperation == model.OpUpgrade && fresh.CurrentOperationState == model.OpStateProcessing {
		t.Errorf("拒绝阶段不应设置升级操作锁，实际=%s/%s",
			fresh.CurrentOperation, fresh.CurrentOperationState)
	}
}

// TestHandleUpgrade_RejectDowngrade_NonOfficialImageAlsoRejected 验证：
// 自定义镜像（用户自定义 imageId）在实例版本高于镜像版本时也应被拒绝。
// 覆盖“管理员把启用中自定义镜像版本号改小”这类事故场景。
func TestHandleUpgrade_RejectDowngrade_NonOfficialImageAlsoRejected(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-reject-private")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.AgentVersion = "2099.12.31"
	})

	// 使用一个非候选镜像 ID
	privateImageId := "img-private-test-001"
	if hcommon.IsCandidateImage(privateImageId) {
		t.Fatalf("用例前提失败：%s 不应是候选镜像", privateImageId)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      privateImageId,
		Enabled:      true,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "2026.1.1",
	}).Error; err != nil {
		t.Fatalf("创建启用镜像失败: %v", err)
	}

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id="+itoaUint(inst.ID), "upgrade-reject-private", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)

	// 自定义镜像也应命中前置拒绝 → 400
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("自定义镜像降级应命中前置拒绝返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	// 应同时包含两个版本号
	if !strings.Contains(errMsg, "2099.12.31") || !strings.Contains(errMsg, "2026.1.1") {
		t.Errorf("响应应同时包含当前版本与目标版本，实际错误: %s", errMsg)
	}
	// 自定义镜像必须走自定义镜像文案分支，不应命中官方镜像文案
	if strings.Contains(errMsg, "官方镜像") {
		t.Errorf("自定义镜像不应使用官方镜像文案，实际: %s", errMsg)
	}
	if !strings.Contains(errMsg, "自定义") {
		t.Errorf("自定义镜像应使用自定义镜像文案，实际: %s", errMsg)
	}

	// 关键副作用校验：拒绝阶段早于 setOperation，不应留下进行中的升级锁
	var fresh model.Instance
	if err := model.DB(context.Background()).First(&fresh, inst.ID).Error; err != nil {
		t.Fatalf("重新加载实例失败: %v", err)
	}
	if fresh.CurrentOperation == model.OpUpgrade && fresh.CurrentOperationState == model.OpStateProcessing {
		t.Errorf("拒绝阶段不应设置升级操作锁，实际=%s/%s",
			fresh.CurrentOperation, fresh.CurrentOperationState)
	}
}

// ─── HandleUpgradeRetry 入口集成测试 ─────────────────────────────────────────
//
// 此前 reject downgrade 仅在 HandleUpgrade / HandleAdminBatchUpgrade 中生效，
// HandleUpgradeRetry 完全没有这道闸门：实例只要进入"升级失败"态，就能通过 retry
// 入口被官方镜像降级覆盖，构成安全旁路。本组用例覆盖修复后的：
//   1. retry 入口在"实例版本严格高于官方镜像版本"时返回 400 并不进入异步流程；
//   2. 非官方镜像（用户自定义 imageId）即使版本高也不会命中 reject，保持向后兼容。

// TestHandleUpgradeRetry_RejectDowngrade_OfficialImageAndOpenclawHigher 命中拒绝路径。
// 必须在前置检查阶段拦下，不允许进入后续 SMH 备份查询 / 操作锁设置。
func TestHandleUpgradeRetry_RejectDowngrade_OfficialImageAndOpenclawHigher(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-reject-downgrade")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		// retry 入口要求 CurrentOperation=upgrade & state=failed
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateFailed
		i.AgentVersion = "2099.12.31" // 远高于任何官方版本
	})

	// 选用一个官方候选 OpenClaw 镜像，并把镜像版本设为"较低"
	cand := pickOfficialOpenClawCandidate(t)
	img := &model.AIImage{
		ImageId:      cand.ImageId,
		Enabled:      true,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: cand.AgentVersion,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("创建启用镜像失败: %v", err)
	}


	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id="+itoaUint(inst.ID), "retry-reject-downgrade", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("retry 入口命中降级拒绝应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "2099.12.31") || !strings.Contains(errMsg, cand.AgentVersion) {
		t.Errorf("响应应同时包含当前版本与目标版本，实际错误: %s", errMsg)
	}

	// 关键副作用校验：拒绝阶段早于 setOperationForRetry，不应把 failed 锁覆盖成 processing
	var fresh model.Instance
	if err := model.DB(context.Background()).First(&fresh, inst.ID).Error; err != nil {
		t.Fatalf("重新加载实例失败: %v", err)
	}
	if fresh.CurrentOperationState == model.OpStateProcessing {
		t.Errorf("retry 拒绝阶段不应设置 processing 锁，实际 state=%s", fresh.CurrentOperationState)
	}
}

// TestHandleUpgradeRetry_RejectDowngrade_NonOfficialImageAlsoRejected 验证：
// 自定义镜像在 retry 入口上同样要被前置降级拒绝拦下，
// 且使用专门的自定义镜像文案，不走官方镜像文案分支。
func TestHandleUpgradeRetry_RejectDowngrade_NonOfficialImageAlsoRejected(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-reject-private")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateFailed
		i.AgentVersion = "2099.12.31"
	})

	privateImageId := "img-private-retry-001"
	if hcommon.IsCandidateImage(privateImageId) {
		t.Fatalf("用例前提失败：%s 不应是候选镜像", privateImageId)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      privateImageId,
		Enabled:      true,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "2026.1.1",
	}).Error; err != nil {
		t.Fatalf("创建启用镜像失败: %v", err)
	}


	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id="+itoaUint(inst.ID), "retry-reject-private", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)

	// retry 入口也应命中前置拒绝 → 400
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("自定义镜像 retry 降级应命中前置拒绝返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	// 自定义镜像必须命中自定义镜像文案，不走官方镜像文案分支
	if strings.Contains(body, "官方镜像") {
		t.Errorf("自定义镜像不应使用官方镜像文案，实际: %s", body)
	}
	if !strings.Contains(body, "自定义") {
		t.Errorf("自定义镜像应使用自定义镜像文案，实际: %s", body)
	}
	// 应同时包含两个版本号
	if !strings.Contains(body, "2099.12.31") || !strings.Contains(body, "2026.1.1") {
		t.Errorf("响应应同时包含当前版本与目标版本，实际: %s", body)
	}

	// 关键副作用校验：拒绝阶段早于 setOperationForRetry，不应把 failed 锁覆盖成 processing
	var fresh model.Instance
	if err := model.DB(context.Background()).First(&fresh, inst.ID).Error; err != nil {
		t.Fatalf("重新加载实例失败: %v", err)
	}
	if fresh.CurrentOperationState == model.OpStateProcessing {
		t.Errorf("retry 拒绝阶段不应设置 processing 锁，实际 state=%s", fresh.CurrentOperationState)
	}
}
