package usergroup

import (
	"context"
	"hatchery/model"
)

// ──────────────────────────────────────────────
// 祖先链查询
// ──────────────────────────────────────────────
// 封装 model.ClosureAncestors，提供分组配置 Resolver 需要的祖先链查询能力。

// GetAncestorIDs 获取单个组的祖先链（含自己，按深度从近到远排序）。
// ancestors[0] = 自己, ancestors[1] = 父, ancestors[2] = 祖父, ...
func GetAncestorIDs(ctx context.Context, groupID uint) ([]uint, error) {
	if groupID == 0 {
		return nil, nil
	}
	return model.ClosureAncestors(ctx, groupID, true)
}

// GetAllAncestorIDs 获取多个组的祖先链并集（去重）。
// 用于多组用户的加法型资源解析。
func GetAllAncestorIDs(ctx context.Context, groupIDs []uint) ([]uint, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}

	seen := make(map[uint]struct{})
	result := make([]uint, 0, len(groupIDs)*3)

	for _, gid := range groupIDs {
		ancestors, err := model.ClosureAncestors(ctx, gid, true)
		if err != nil {
			return nil, err
		}
		for _, aid := range ancestors {
			if _, ok := seen[aid]; !ok {
				seen[aid] = struct{}{}
				result = append(result, aid)
			}
		}
	}
	return result, nil
}

// GetUserAllGroupAndAncestorIDs 获取用户所有组 + 祖先的 ID 并集（加法型用）。
// 对于未分组用户返回 nil。
func GetUserAllGroupAndAncestorIDs(ctx context.Context, userID uint) ([]uint, error) {
	groupIDs, err := model.GetUserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return nil, nil
	}
	return GetAllAncestorIDs(ctx, groupIDs)
}
