package controller

// 本地 Agent sync 的 Workspace 关系与项目资产对账。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func processSyncWorkspaces(ctx context.Context, inst *model.Instance, user *model.User,
	workspaces []syncWorkspace) error {
	if len(workspaces) == 0 {
		return nil
	}
	return model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 统一加锁顺序：先对实例行加 FOR UPDATE 排它锁，再写 local_agent_scope_bindings。
		// 否则与并发 report（report 事务先 UPDATE instances 后 upsert scope binding）
		// 以相反顺序加锁，形成 AB-BA 死锁（生产 MySQL 日志 LATEST DETECTED DEADLOCK 已出现）。
		// 顺带从锁定的最新行反序列化，避免基于事务外旧快照计算。
		var locked model.Instance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", inst.ID).First(&locked).Error; err != nil {
			return err
		}
		resources := deserializeLocalAgentResources(locked.LocalAgentResources)
		reportedPaths := make(map[string]bool)
		newWorkspaces := make([]model.WorkspaceResource, 0, len(workspaces))
		for _, ws := range workspaces {
			path := strings.TrimSpace(ws.Path)
			if path == "" || reportedPaths[path] {
				continue
			}
			reportedPaths[path] = true
			oldProjectID := uint(0)
			for _, oldWS := range resources.Workspaces {
				if oldWS.Path == path {
					oldProjectID = oldWS.ProjectID
					break
				}
			}
			projectID, err := resolveSyncWorkspaceProjectID(tx, user.ID, oldProjectID, ws.ProjectID)
			if err != nil {
				return fmt.Errorf("校验 workspace project_id (path=%s): %w", path, err)
			}
			wsRes := model.WorkspaceResource{Path: path, Name: ws.Name, IDEType: ws.IDEType, ProjectID: projectID}
			newWorkspaces = append(newWorkspaces, wsRes)
			if err := upsertWorkspaceScopeBinding(tx, inst.ID, wsRes, time.Now()); err != nil {
				return fmt.Errorf("写入 workspace project binding: %w", err)
			}
			if projectID > 0 {
				if _, err := requireProject(tx, projectID); err == nil {
					if _, err := diffProjectAndQueue(ctx, tx, inst.ID, projectID, path); err != nil {
						return err
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
		}
		resources.Workspaces = newWorkspaces
		jsonBytes, err := json.Marshal(resources)
		if err != nil {
			return fmt.Errorf("序列化 local_agent_resources: %w", err)
		}
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Update("local_agent_resources", string(jsonBytes)).Error; err != nil {
			return fmt.Errorf("更新 local_agent_resources: %w", err)
		}
		return nil
	})
}

// ---- 辅助函数 ------------------------------------------------------------

func resolveSyncWorkspaceProjectID(tx *gorm.DB, userID, oldProjectID uint, requested *uint) (uint, error) {
	if requested == nil {
		return oldProjectID, nil
	}
	projectID := *requested
	if projectID == 0 || projectID == oldProjectID {
		return projectID, nil
	}
	var count int64
	if err := tx.Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return projectID, nil
}

// deserializeLocalAgentResources 从 instances.LocalAgentResources 反序列化，nil 时返回空结构。
