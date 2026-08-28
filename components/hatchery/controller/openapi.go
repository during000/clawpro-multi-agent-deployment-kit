package controller

import (
	"net/http"
	"strings"
)

const openAPIHeader = "X-Hatchery-OpenAPI"

// WithOpenAPI 标记一个 handler 为开放 API，允许用户 Bearer Token 鉴权。
// 同时，当请求携带了 Bearer Token 时，自动将 Accept 设为 application/json。
// handler 执行完毕后会清理注入的内部标记，避免泄漏到后续链路。
func WithOpenAPI(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 注入内部标记（外部已被中间件清除，不可伪造）
		r.Header.Set(openAPIHeader, "1")
		defer r.Header.Del(openAPIHeader) // handler 完成后清理内部标记

		// 2. 当请求携带 Bearer Token 时，默认 Accept 为 JSON
		//    （API 客户端通常不会设置 Accept，避免返回 HTML）
		if hasBearerToken(r) && !strings.Contains(r.Header.Get("Accept"), "application/json") {
			r.Header.Set("Accept", "application/json")
		}

		handler(w, r)
	}
}

// isOpenAPIRequest 判断当前请求是否来自 WithOpenAPI 标记的路由
func isOpenAPIRequest(r *http.Request) bool {
	return r.Header.Get(openAPIHeader) == "1"
}

// hasBearerToken 判断请求是否携带了 Bearer Token
func hasBearerToken(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
}

// isAdminTokenRequest 判断请求是否使用的是启动参数设置的 AdminToken（超级管理令牌）。
// 与普通用户/管理员的 API Token 区分：AdminToken 拥有最高权限，可修改敏感配置。
func isAdminTokenRequest(r *http.Request) bool {
	if AdminToken == "" {
		return false
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	return token == AdminToken
}
