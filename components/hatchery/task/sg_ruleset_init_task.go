package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// sgInitState per-tenant SG 初始化状态。
type sgInitState struct {
	completed int32  // 0=未完成, 1=已完成
	lastErr   string // 最后一次失败信息
	mu        sync.RWMutex
}

var sgInitStates sync.Map // key: identifier(string), value: *sgInitState

func getSGInitState(identifier string) *sgInitState {
	val, _ := sgInitStates.LoadOrStore(identifier, &sgInitState{})
	return val.(*sgInitState)
}

// IsSGInitCompleted 返回指定租户的 SG RuleSet 初始化是否已完成。
// identifier 为空时检查所有租户（阶段一单租户兼容）。
func IsSGInitCompleted() bool {
	// 阶段一：只有一个租户
	allDone := true
	sgInitStates.Range(func(_, v interface{}) bool {
		s := v.(*sgInitState)
		if loadInt32(&s.completed) != 1 {
			allDone = false
			return false
		}
		return true
	})
	return allDone
}

// GetSGInitError 返回最近一次 SG RuleSet 初始化失败的错误信息。
func GetSGInitError() string {
	var lastErr string
	sgInitStates.Range(func(_, v interface{}) bool {
		s := v.(*sgInitState)
		s.mu.RLock()
		e := s.lastErr
		s.mu.RUnlock()
		if e != "" {
			lastErr = e
			return false
		}
		return true
	})
	return lastErr
}

func init() {
	RegisterTask(TaskDef{
		Name:         "sg-ruleset-init",
		Interval:     30 * time.Second,
		InitialDelay: 60 * time.Second, // 避免启动初期与集成测试场景并发打云 API 导致限频
		RunFunc:      runSGRuleSetInit,
		NeedDistLock: false,
		PerTenant:    true,
	})
}

// runSGRuleSetInit 幂等执行一次 SG RuleSet 初始化。
// 已成功则直接 return；失败则记录错误、通知 admin，等下次 scheduler tick 重试。
func runSGRuleSetInit(ctx context.Context) {
	identifier := model.CurrentIdentifier(ctx)
	state := getSGInitState(identifier)

	if loadInt32(&state.completed) == 1 {
		return
	}
	if err := initSGRuleSet(ctx); err == nil {
		storeInt32(&state.completed, 1)
		state.mu.Lock()
		state.lastErr = ""
		state.mu.Unlock()
		slog.Info("[SGInit] init succeeded", "identifier", identifier)
		notifyAdminsOfSGInitRecovery(ctx)
	} else {
		state.mu.Lock()
		state.lastErr = err.Error()
		state.mu.Unlock()
		slog.Warn("[SGInit] init failed, will retry next tick", "identifier", identifier, "err", err)
		notifyAdminsOfSGInitFailure(ctx, err)
	}
}

// initSGRuleSet 幂等执行 SG RuleSet 初始化。
func initSGRuleSet(ctx context.Context) error {
	identifier := model.CurrentIdentifier(ctx)

	// 1. 快速幂等检查（无需拿锁）
	if _, err := model.GetDefaultRuleSet(ctx); err == nil {
		slog.Info("[SGInit] rule_set already exists, skip", "identifier", identifier)
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitPreCheck)
	}

	// 2. MySQL：取分布式锁串行化多 Pod
	lock, err := model.AcquireLock(ctx, "sg-bootstrap", 60*time.Second)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitAcquireLock)
	}
	defer lock.Release()

	// 3. double-check
	if _, err := model.GetDefaultRuleSet(ctx); err == nil {
		slog.Info("[SGInit] rule_set exists after lock, skip", "identifier", identifier)
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitPostLockCheck)
	}

	started := time.Now()
	slog.Info("[SGInit] begin ruleset init", "identifier", identifier)

	// 4. 读老 base SG 规则
	siteCfg := model.GetSiteConfig(ctx)
	oldBase := siteCfg.SecurityGroupId

	// 4.1 全新租户：不自动建，等管理员手动初始化
	if oldBase == "" {
		slog.Info("[SGInit] fresh tenant with no legacy SG, skip auto-init; waiting for manual /initialize",
			"identifier", identifier)
		return nil
	}

	var userRules []controller.Rule
	var oldBaseCVMCount int
	oldBaseForFrozen := oldBase
	{
		rules, err := controller.DescribeSGPoliciesWithRetry(ctx, oldBase)
		if err != nil {
			if controller.IsSGGoneError(err) {
				slog.Warn("[SGInit] old base SG no longer exists in cloud, proceeding with empty rules",
					"old_sg_id", oldBase, "err", err)
				userRules = nil
				oldBaseForFrozen = ""
			} else {
				return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitDescribeOldBase, oldBase)
			}
		} else {
			userRules = rules
		}

		if oldBaseForFrozen != "" {
			var cnt int64
			if err := model.DB(ctx).Model(&model.Instance{}).
				Where("security_group_id = ?", oldBase).
				Count(&cnt).Error; err != nil {
				slog.Warn("[SGInit] count instances on old base failed, defaulting to 0", "err", err)
			} else {
				oldBaseCVMCount = int(cnt)
			}
		}
		slog.Info("[SGInit] read old base rules",
			"old_sg_id", oldBase, "rule_count", len(userRules), "bound_cvm", oldBaseCVMCount,
			"old_base_gone", oldBaseForFrozen == "")
	}

	// 5. 创建 RuleSet + SG（forceSkipMerge=true: 保留老 base 原貌）
	rsID, newSGID, err := controller.CreateInitialRuleSetAndSG(ctx, model.DefaultRuleSetName, model.DefaultRuleSetRemark, userRules, oldBaseForFrozen, oldBaseCVMCount, true, false)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitFailed)
	}

	// 6. 审计
	model.LogAudit(ctx, started, 0, "system", "sg_ruleset_init", "rule_set",
		fmt.Sprintf("%d", rsID), "success preserved_user_rules_as_is=true")
	slog.Info("[SGInit] complete",
		"rule_set_id", rsID, "new_sg_id", newSGID, "old_base", oldBase,
		"frozen_cvm", oldBaseCVMCount,
		"preserved_user_rules_as_is", true,
		"elapsed_ms", time.Since(started).Milliseconds())

	// 7. 通知 admin
	notifyAdminsOfSGMigration(ctx, rsID, oldBase, newSGID)

	return nil
}

// notifyAdminsOfSGMigration 存量迁移完成后，给本租户所有管理员发一条站内通知。
func notifyAdminsOfSGMigration(ctx context.Context, rsID uint, oldBase, newSGID string) {
	var admins []model.User
	if err := model.DB(ctx).Where("role = ?", "admin").Find(&admins).Error; err != nil {
		slog.Warn("[SGInit] query admins for migration notice failed (non-fatal)", "err", err)
		return
	}
	if len(admins) == 0 {
		slog.Warn("[SGInit] no admin user found, skip migration notice")
		return
	}

	title := i18n.T(ctx, i18n.MsgSGMigrationNoticeTitle)
	message := i18n.T(ctx, i18n.MsgSGMigrationNoticeMessage)
	okCount := 0
	for _, u := range admins {
		if err := model.CreateNotificationWithCategory(
			ctx, u.ID, 0, "",
			"sg_migration", model.NotifCategoryNotice,
			title, message, nil,
		); err != nil {
			slog.Warn("[SGInit] create migration notice failed", "admin_id", u.ID, "err", err)
			continue
		}
		okCount++
	}
	slog.Info("[SGInit] migration notices dispatched",
		"admin_count", len(admins), "success_count", okCount,
		"rule_set_id", rsID, "old_base", oldBase, "new_sg_id", newSGID)
}

// notifyAdminsOfSGInitFailure 通知所有 admin：SG RuleSet 初始化失败。
func notifyAdminsOfSGInitFailure(ctx context.Context, initErr error) {
	var unreadCount int64
	model.DB(ctx).Model(&model.Notification{}).
		Where("`type` = ? AND is_read = ?", "sg_bootstrap_failed", false).
		Count(&unreadCount)
	if unreadCount > 0 {
		slog.Info("[SGInit] skipping failure notification (unread notification exists)")
		return
	}

	var admins []model.User
	if err := model.DB(ctx).Where("role = ?", "admin").Find(&admins).Error; err != nil {
		slog.Warn("[SGInit] query admins for failure notice failed", "err", err)
		return
	}

	title := i18n.T(ctx, i18n.MsgSGInitFailureTitle)
	message := i18n.T(ctx, i18n.MsgSGInitFailureMessage, initErr.Error())
	for _, u := range admins {
		_ = model.CreateNotificationWithCategory(
			ctx, u.ID, 0, "",
			"sg_bootstrap_failed", model.NotifCategoryError,
			title, message, &model.NotifErrorDetail{
				Error:  "LimitExceeded.SecurityGroup",
				Detail: initErr.Error(),
			},
		)
	}
	slog.Info("[SGInit] failure notifications dispatched", "admin_count", len(admins))
}

// notifyAdminsOfSGInitRecovery 通知所有 admin：SG RuleSet 初始化已恢复。
func notifyAdminsOfSGInitRecovery(ctx context.Context) {
	var admins []model.User
	if err := model.DB(ctx).Where("role = ?", "admin").Find(&admins).Error; err != nil {
		slog.Warn("[SGInit] query admins for recovery notice failed", "err", err)
		return
	}

	title := i18n.T(ctx, i18n.MsgSGInitRecoveryTitle)
	message := i18n.T(ctx, i18n.MsgSGInitRecoveryMessage)
	for _, u := range admins {
		_ = model.CreateNotificationWithCategory(
			ctx, u.ID, 0, "",
			"sg_bootstrap_recovered", model.NotifCategorySuccess,
			title, message, nil,
		)
	}
	slog.Info("[SGInit] recovery notifications dispatched", "admin_count", len(admins))
}
