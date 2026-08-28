package controller

import (
	"context"
	"log/slog"
	"strings"

	"hatchery/model"
)

// uniqueUsername 为 OneID 用户生成一个不冲突的本地用户名。
// 优先使用 preferred（OneID 中的姓名），冲突则依次追加 sub 后 4 位、完整 sub。
// excludeUserID > 0 时排除该用户本身（用于更新场景）。
// excludeSub 非空时排除同 one_id_sub 的记录（用于创建场景：FirstOrCreate 会复用同 sub 的软删除记录）。
func uniqueUsername(ctx context.Context, preferred, sub string, excludeUserID uint) string {
	if preferred == "" {
		preferred = "user-" + sub
	}

	candidates := []string{
		preferred,
	}
	if len(sub) > 4 {
		candidates = append(candidates, preferred+"_"+sub[len(sub)-4:])
	}
	candidates = append(candidates, preferred+"-"+sub)

	for _, name := range candidates {
		var count int64
		// Unscoped: 必须包含已软删除的用户，因为数据库唯一约束（idx_username_identifier）
		// 不区分 deleted_at，软删除的用户名仍占用唯一索引。
		q := model.DB(ctx).Unscoped().Model(&model.User{}).Where("username = ?", name)
		if excludeUserID > 0 {
			q = q.Where("id != ?", excludeUserID)
		}
		// 排除同 one_id_sub 的记录：创建时 FirstOrCreate 按 sub 匹配会复用该记录，
		// 所以它不构成真正的用户名冲突。
		if sub != "" {
			q = q.Where("(one_id_sub IS NULL OR one_id_sub != ?)", sub)
		}
		if err := q.Count(&count).Error; err != nil {
			slog.Warn("uniqueUsername: failed to check conflict, skipping candidate",
				"name", name, "err", err)
			continue
		}
		if count == 0 {
			return name
		}
	}
	// 理论上 preferred+"-"+sub 不会冲突（sub 是唯一的），但以防万一
	return preferred + "-" + sub
}

// isUsernameVariant 检查 currentUsername 是否是 oneidName 的一个去重变体。
// 即 currentUsername 等于 oneidName 本身，或以 oneidName+"_" / oneidName+"-" 开头。
// 用途：如果用户名已经是 OneID 姓名的变体（创建时因冲突追加了后缀），
// 就不再反复尝试改回原名，避免用户名来回变动。
func isUsernameVariant(currentUsername, oneidName string) bool {
	if currentUsername == oneidName {
		return true
	}
	if strings.HasPrefix(currentUsername, oneidName+"_") ||
		strings.HasPrefix(currentUsername, oneidName+"-") {
		return true
	}
	return false
}

// safeUpdateUsername 安全地将用户的 username 更新为 newName。
// 规则：
//  1. 如果当前用户名已经是 newName 的变体（如 "张三_8779"），跳过——不要来回改；
//  2. 如果 newName 与其他用户冲突，也跳过——不追加后缀，因为这是同步场景而非创建场景。
func safeUpdateUsername(ctx context.Context, user *model.User, newName string) {
	if newName == "" || newName == user.Username {
		return
	}
	// 如果当前用户名已经是 OneID 姓名的变体（创建时因冲突追加了后缀），
	// 不再尝试改回原名，避免用户名反复变动。
	if isUsernameVariant(user.Username, newName) {
		return
	}
	// 检查是否会与其他用户冲突（包含已软删除的用户，因为唯一约束不区分 deleted_at）
	var count int64
	if err := model.DB(ctx).Unscoped().Model(&model.User{}).Where("username = ? AND id != ?", newName, user.ID).Count(&count).Error; err != nil {
		slog.Warn("safeUpdateUsername: failed to check conflict, skipping",
			"user_id", user.ID, "desired", newName, "err", err)
		return
	}
	if count > 0 {
		slog.Warn("safeUpdateUsername: skipped due to conflict",
			"user_id", user.ID, "current", user.Username, "desired", newName)
		return
	}
	if err := model.DB(ctx).Model(user).Update("username", newName).Error; err != nil {
		slog.Warn("safeUpdateUsername: update failed",
			"user_id", user.ID, "current", user.Username, "desired", newName, "err", err)
	}
}
