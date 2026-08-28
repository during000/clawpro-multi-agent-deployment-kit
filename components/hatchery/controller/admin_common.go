package controller

import (
	"log/slog"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/model"
)

func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if user, err := getUserFromToken(r); user != nil {
		// admin-token（ID == 0）或 role 为 admin 的用户 API Token 均可访问管理接口
		if user.Role != "admin" {
			writeError(w, r, http.StatusForbidden, ErrAdminRequired)
			return false
		}
		return true
	} else if err != nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.EnsureRichErrorOrPanic(err))
		return false
	}
	session := getSession(r)
	username, _ := session.Values["username"].(string)
	if username == "" {
		writeError(w, r, http.StatusUnauthorized, ErrUnauthorized)
		return false
	}

	// 从数据库实时查询最新 role，而非信任 session 缓存。
	// 这样管理员在后台变更用户角色后，无需用户刷新页面即可立即生效。
	var dbUser model.User
	if err := model.DB(r.Context()).Select("role").Where("username = ?", username).First(&dbUser).Error; err != nil {
		writeError(w, r, http.StatusForbidden, ErrForbidden)
		return false
	}
	role := dbUser.Role

	// 顺便同步 session，减少后续查询
	if sessionRole, _ := session.Values["role"].(string); sessionRole != role {
		session.Values["role"] = role
		if err := session.Save(r, w); err != nil {
			slog.Warn("requireAdmin: failed to save session", "err", err)
		}
	}

	if role != "admin" {
		writeError(w, r, http.StatusForbidden, ErrAdminRequired)
		return false
	}
	return true
}
