package controller

import (
	"context"
	"net/http"

	"hatchery/model"
)

type projectVisibilityInfo struct {
	ProjectID   uint   `json:"project_id"`
	ProjectName string `json:"project_name"`
}

func validateVisibilityProjects(r *http.Request, projectIDs []uint) error {
	return model.ValidateProjectIDs(model.DB(r.Context()), projectIDs)
}

// buildProjectVisibilityData 以资源稳定 slug 批量返回项目应用范围。
func buildProjectVisibilityData(ctx context.Context, configType string, keys []string) (map[string][]projectVisibilityInfo, error) {
	result := make(map[string][]projectVisibilityInfo)
	if len(keys) == 0 {
		return result, nil
	}
	var bindings []model.ProjectConfigBinding
	if err := model.DB(ctx).Where("config_type = ? AND config_key IN ?", configType, keys).Find(&bindings).Error; err != nil {
		return nil, err
	}
	projectIDs := make([]uint, 0, len(bindings))
	for _, binding := range bindings {
		projectIDs = append(projectIDs, binding.ProjectID)
	}
	var projects []model.Project
	if len(projectIDs) > 0 {
		if err := model.DB(ctx).Where("id IN ?", uniqueUintIDs(projectIDs)).Find(&projects).Error; err != nil {
			return nil, err
		}
	}
	names := make(map[uint]string, len(projects))
	for _, project := range projects {
		names[project.ID] = project.Name
	}
	for _, binding := range bindings {
		if name, ok := names[binding.ProjectID]; ok {
			result[binding.ConfigKey] = append(result[binding.ConfigKey], projectVisibilityInfo{
				ProjectID: binding.ProjectID, ProjectName: name,
			})
		}
	}
	return result, nil
}
