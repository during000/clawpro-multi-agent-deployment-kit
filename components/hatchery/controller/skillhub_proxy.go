package controller

import (
	"log/slog"
	"net/http"
)

// WithSkillHubProxy 灰度代理装饰器。
// 根据 site_configs.skill_hub_enabled 决定走本地 DB 老流程还是 SkillHub API。
// 灰度命中时不依赖 SMH（SkillHub 自己管理技能存储）。
//
// 用法（main.go 路由注册）：
//
//	http.HandleFunc("/admin/skills",
//	    WithSkillHubProxy(controller.HandleAdminSkills, controller.HandleAdminSkillsViaSkillHub))
//
// 扩展新接口时只需：写 XxxViaSkillHub handler + 路由注册改一行。
func WithSkillHubProxy(localHandler, skillHubHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isSkillHubEnabled(r) {
			slog.InfoContext(r.Context(), "[SkillHubProxy] 灰度命中", "path", r.URL.Path)
			skillHubHandler(w, r)
			return
		}
		localHandler(w, r)
	}
}
