package controller

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

type agentProxyRouteSpec struct {
	Kind       string
	TargetPort int
	TargetPath string
}

var agentProxyRouteSpecs = map[string]agentProxyRouteSpec{
	model.AgentProxyRouteKindTeams: {
		Kind:       model.AgentProxyRouteKindTeams,
		TargetPort: 3978,
		TargetPath: "/api/messages",
	},
	model.AgentProxyRouteKindLine: {
		Kind:       model.AgentProxyRouteKindLine,
		TargetPort: 8646,
		TargetPath: "/line/webhook",
	},
}

// agentProxyKindToChannel maps an agent proxy route kind to the corresponding
// channel ID used in predefinedChannels / channelInCurrentSiteScope / AgentTypeChannelAllowed.
var agentProxyKindToChannel = map[string]string{
	model.AgentProxyRouteKindTeams: "msteams",
	model.AgentProxyRouteKindLine:  "line",
}

var (
	resolveInstanceAccessIPForAgentProxy = resolveInstanceAccessIP
	refreshRuleSetsForAgentProxy         = RefreshAllRuleSetsForRequiredRules
)

func agentProxyRouteSpecForKind(kind string) (agentProxyRouteSpec, bool) {
	spec, ok := agentProxyRouteSpecs[kind]
	return spec, ok
}

func randomRouteID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func firstForwardedHeaderValue(value string) string {
	if i := strings.Index(value, ","); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

func normalizeExternalBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return strings.TrimRight(u.String(), "/")
}

func requestExternalBaseURL(r *http.Request) string {
	if base := normalizeExternalBaseURL(common.DomainFromCtx(r.Context())); base != "" {
		return base
	}

	scheme := firstForwardedHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := firstForwardedHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	u := url.URL{Scheme: scheme, Host: host}
	return strings.TrimRight(u.String(), "/")
}

func proxyEndpointForRoute(r *http.Request, route model.AgentProxyRoute) string {
	endpoint, err := url.JoinPath(requestExternalBaseURL(r), "api", "proxy", route.RouteID, route.TargetPath)
	if err != nil {
		return requestExternalBaseURL(r) + "/api/proxy/" + route.RouteID + route.TargetPath
	}
	return endpoint
}

func resolveInstanceAccessIP(r *http.Request, instanceID string) (string, error) {
	client, err := GetCVMClient(r.Context())
	if err != nil {
		return "", fmt.Errorf("创建CVM客户端失败: %w", err)
	}
	req := cvm.NewDescribeInstancesRequest()
	req.InstanceIds = tccommon.StringPtrs([]string{instanceID})
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return "", fmt.Errorf("查询实例失败: %w", err)
	}
	if resp.Response == nil || len(resp.Response.InstanceSet) == 0 {
		return "", errors.New("未找到实例")
	}
	inst := resp.Response.InstanceSet[0]
	if len(inst.PublicIpAddresses) > 0 && inst.PublicIpAddresses[0] != nil && *inst.PublicIpAddresses[0] != "" {
		return *inst.PublicIpAddresses[0], nil
	}
	if len(inst.PrivateIpAddresses) > 0 && inst.PrivateIpAddresses[0] != nil && *inst.PrivateIpAddresses[0] != "" {
		return *inst.PrivateIpAddresses[0], nil
	}
	return "", errors.New("实例无可用IP")
}

func ensureAgentProxyRoute(r *http.Request, instance *model.Instance, kind string) (*model.AgentProxyRoute, string, error) {
	if instance == nil || instance.InstanceId == "" {
		return nil, "", common.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "id", "实例无关联的 CVM")
	}
	spec, ok := agentProxyRouteSpecForKind(kind)
	if !ok {
		return nil, "", common.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "kind", kind)
	}
	ip, err := resolveInstanceAccessIPForAgentProxy(r, instance.InstanceId)
	if err != nil {
		return nil, "", common.I18nRichError(err, i18n.MsgQueryInstanceFailed)
	}

	var route model.AgentProxyRoute
	db := model.DB(r.Context())
	err = db.Where("instance_id = ? AND kind = ?", instance.InstanceId, spec.Kind).First(&route).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", common.I18nRichError(err, i18n.MsgDatabaseOperationFailed)
	}
	wasEnabled := false
	createdRoute := false
	if errors.Is(err, gorm.ErrRecordNotFound) {
		createdRoute = true
		routeID, genErr := randomRouteID()
		if genErr != nil {
			return nil, "", common.I18nRichError(genErr, i18n.MsgGenerateProxyRouteFailed)
		}
		route = model.AgentProxyRoute{
			RouteID:    routeID,
			InstanceID: instance.InstanceId,
			Kind:       spec.Kind,
			TargetPort: spec.TargetPort,
			TargetPath: spec.TargetPath,
			Enabled:    true,
		}
		// TargetIP is stored only as a cache for proxying; refresh it on every prepare/set-channel.
		route.TargetIP = ip
		if err := db.Create(&route).Error; err != nil {
			return nil, "", common.I18nRichError(err, i18n.MsgDatabaseOperationFailed)
		}
	} else {
		wasEnabled = route.Enabled
		updates := map[string]interface{}{
			"target_ip":   ip,
			"target_port": spec.TargetPort,
			"target_path": spec.TargetPath,
			"enabled":     true,
		}
		if err := db.Model(&route).Updates(updates).Error; err != nil {
			return nil, "", common.I18nRichError(err, i18n.MsgDatabaseOperationFailed)
		}
		route.TargetIP = ip
		route.TargetPort = spec.TargetPort
		route.TargetPath = spec.TargetPath
		route.Enabled = true
	}

	// 条件型 agent proxy 规则依赖 route enabled 状态。必须先启用 route，再刷新规则集，
	// 让 resolveConditionalRules 按运行时条件决定是否投影对应端口。
	if err := refreshRuleSetsForAgentProxy(r.Context()); err != nil {
		// 如果本次新建或从 disabled 切到 enabled，刷新失败时回滚 route enabled，避免暴露不可达 endpoint。
		if createdRoute || !wasEnabled {
			_ = db.Model(&route).Update("enabled", false).Error
		}
		return nil, "", common.I18nRichError(err, i18n.MsgApplyAgentProxySGRulesFailed)
	}
	return &route, proxyEndpointForRoute(r, route), nil
}

func checkAgentProxyKindAllowed(r *http.Request, kind string, instance *model.Instance) error {
	channel, ok := agentProxyKindToChannel[kind]
	if !ok {
		// Unknown kind — if it's not in our mapping, it was already rejected by spec lookup.
		return nil
	}
	if !channelInCurrentSiteScope(r.Context(), channel) {
		return common.I18nError(i18n.MsgChannelNotExist)
	}
	if !model.AgentTypeChannelAllowed(r.Context(), instance.AgentType, channel) {
		return common.I18nError(i18n.MsgAgentTypeNotSupportChannel, instance.AgentType, channel)
	}
	return nil
}

// HandleProxyPrepare creates or refreshes an opaque public proxy endpoint for a user-owned instance.
// POST /openclaw/proxy/prepare?id=<instance_id>&kind=teams
func HandleProxyPrepare(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	if kind == "" {
		kind = model.AgentProxyRouteKindTeams
	}
	if _, ok := agentProxyRouteSpecForKind(kind); !ok {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "kind", kind))
		return
	}
	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}
	if err := checkInstanceSupportsChannel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, common.EnsureRichErrorOrPanic(err))
		return
	}
	if err := checkAgentProxyKindAllowed(r, kind, instance); err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}
	route, endpoint, err := ensureAgentProxyRoute(r, instance, kind)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, common.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true, "kind": kind, "route_id": route.RouteID, "endpoint": endpoint})
}

// HandleAdminProxyPrepare creates or refreshes a proxy endpoint for any instance.
// POST /admin/instances/proxy/prepare?id=<instance_id>&kind=teams
func HandleAdminProxyPrepare(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	if kind == "" {
		kind = model.AgentProxyRouteKindTeams
	}
	if _, ok := agentProxyRouteSpecForKind(kind); !ok {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "kind", kind))
		return
	}
	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}
	if err := checkInstanceSupportsChannel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, common.EnsureRichErrorOrPanic(err))
		return
	}
	if err := checkAgentProxyKindAllowed(r, kind, instance); err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}
	route, endpoint, err := ensureAgentProxyRoute(r, instance, kind)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, common.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true, "kind": kind, "route_id": route.RouteID, "endpoint": endpoint})
}

func parseProxyPath(path string) (routeID, rest string, ok bool) {
	for _, prefix := range []string{"/proxy/", "/api/proxy/"} {
		if strings.HasPrefix(path, prefix) {
			remaining := strings.TrimPrefix(path, prefix)
			parts := strings.SplitN(remaining, "/", 2)
			if len(parts) != 2 || parts[0] == "" {
				return "", "", false
			}
			return parts[0], "/" + parts[1], true
		}
	}
	return "", "", false
}

// HandleAgentProxy is a generic reverse proxy entry. Currently only Teams webhook routes are created.
// External URL behind nginx: /api/proxy/{routeID}/api/messages
// Backend route after nginx prefix stripping: /proxy/{routeID}/api/messages
func HandleAgentProxy(w http.ResponseWriter, r *http.Request) {
	routeID, rest, ok := parseProxyPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var route model.AgentProxyRoute
	if err := model.DB(r.Context()).Where("route_id = ?", routeID).First(&route).Error; err != nil {
		// RouteID 是 secret，不暴露是否存在。
		http.NotFound(w, r)
		return
	}
	if !route.Enabled || route.TargetPort <= 0 || route.TargetPath == "" || route.TargetIP == "" {
		http.NotFound(w, r)
		return
	}
	if rest != route.TargetPath {
		http.NotFound(w, r)
		return
	}
	if route.Kind == model.AgentProxyRouteKindTeams && r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if (route.Kind == model.AgentProxyRouteKindLine) &&
		(r.Method != http.MethodPost && r.Method != http.MethodGet) { // LINE 需要支持 GET 请求
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	}

	target := &url.URL{Scheme: "http", Host: route.TargetIP + ":" + strconv.Itoa(route.TargetPort)}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = route.TargetPath
		req.URL.RawPath = ""
		req.URL.RawQuery = r.URL.RawQuery
		req.Host = target.Host
		// Preserve Bot Framework Authorization header and content headers by default.
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", r.Header.Get("X-Forwarded-Proto"))
		if req.Header.Get("X-Forwarded-Proto") == "" {
			if r.TLS != nil {
				req.Header.Set("X-Forwarded-Proto", "https")
			} else {
				req.Header.Set("X-Forwarded-Proto", "http")
			}
		}
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		slog.Warn("[agent-proxy] upstream error", "route_id", route.RouteID, "kind", route.Kind, "instance_id", route.InstanceID, "err", err)
		http.Error(rw, "bad gateway", http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}
