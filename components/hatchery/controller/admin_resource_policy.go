package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

const resourcePolicyBodyLimit = 1 << 20

type resourcePolicyGroupItem struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
}

type resourcePolicyListItem struct {
	ID             uint                      `json:"id"`
	Name           string                    `json:"name"`
	IsDefault      bool                      `json:"is_default"`
	ResourceConfig *ResourceConfig           `json:"resource_config"`
	Groups         []resourcePolicyGroupItem `json:"groups"`
	CreatedAt      string                    `json:"created_at"`
	UpdatedAt      string                    `json:"updated_at"`
}

type resourcePolicyMutationRequest struct {
	ID             uint            `json:"id"`
	Name           string          `json:"name"`
	ResourceConfig json.RawMessage `json:"resource_config"`
	GroupIDs       []uint          `json:"group_ids"`
}

func decodeResourcePolicyRequest(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, resourcePolicyBodyLimit)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return decodeJSONObject(string(body), dst)
}

func normalizeResourcePolicyConfig(ctxReq *http.Request, raw json.RawMessage) (string, *ResourceConfig, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
		return "", nil, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "resource_config")
	}
	cfg, err := ParseResourceConfig(string(raw))
	if err != nil {
		return "", nil, hcommon.I18nRichError(err, i18n.MsgInvalidJSON)
	}
	NormalizeResourceConfig(cfg)
	if err := ValidateResourceConfig(ctxReq.Context(), cfg); err != nil {
		return "", nil, err
	}
	normalized, err := json.Marshal(cfg)
	if err != nil {
		return "", nil, hcommon.I18nRichError(err, i18n.MsgFailedToMarshalJSON)
	}
	return string(normalized), cfg, nil
}

func writeResourcePolicyModelError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrResourcePolicyNotFound):
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgResourcePolicyNotFound))
	case errors.Is(err, model.ErrResourcePolicyGroupNotFound):
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupNotFound))
	case errors.Is(err, model.ErrResourcePolicyNameConflict):
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgResourcePolicyNameConflict))
	case errors.Is(err, model.ErrResourcePolicyGroupOccupied):
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgResourcePolicyGroupOccupied))
	case errors.Is(err, model.ErrDefaultResourcePolicy):
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgDefaultResourcePolicyProtected))
	default:
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDatabaseOperationFailed))
	}
}

// HandleListResourcePolicies GET /admin/resource-policies.
func HandleListResourcePolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	page, pageSize := 1, 10
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "page"))
			return
		}
		page = parsed
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "page_size"))
			return
		}
		pageSize = parsed
	}

	policies, total, err := model.ListResourcePolicies(r.Context(), page, pageSize)
	if err != nil {
		writeResourcePolicyModelError(w, r, err)
		return
	}
	policyIDs := make([]uint, 0, len(policies))
	for _, policy := range policies {
		policyIDs = append(policyIDs, policy.ID)
	}
	groupsByPolicy, err := model.GetResourcePolicyGroups(r.Context(), policyIDs)
	if err != nil {
		writeResourcePolicyModelError(w, r, err)
		return
	}

	items := make([]resourcePolicyListItem, 0, len(policies))
	for _, policy := range policies {
		cfg, err := ParseResourceConfig(policy.ConfigJSON)
		if err != nil {
			writeResourcePolicyModelError(w, r, err)
			return
		}
		groups := groupsByPolicy[policy.ID]
		groupItems := make([]resourcePolicyGroupItem, 0, len(groups))
		for _, group := range groups {
			groupItems = append(groupItems, resourcePolicyGroupItem{
				ID:       group.ID,
				Name:     group.Name,
				FullPath: group.FullPath,
			})
		}
		items = append(items, resourcePolicyListItem{
			ID:             policy.ID,
			Name:           policy.DisplayName(r.Context()),
			IsDefault:      policy.IsDefault,
			ResourceConfig: cfg,
			Groups:         groupItems,
			CreatedAt:      policy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      policy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	jsonOK(w, map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// HandleCreateResourcePolicy POST /admin/resource-policies/create.
func HandleCreateResourcePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	var req resourcePolicyMutationRequest
	if err := decodeResourcePolicyRequest(w, r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNameCannotBeEmpty))
		return
	}
	if len([]rune(req.Name)) > 128 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "name"))
		return
	}
	if len(req.GroupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_ids"))
		return
	}
	configJSON, _, err := normalizeResourcePolicyConfig(r, req.ResourceConfig)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	policy, err := model.CreateResourcePolicy(r.Context(), req.Name, configJSON, req.GroupIDs)
	if err != nil {
		writeResourcePolicyModelError(w, r, err)
		return
	}
	jsonOK(w, map[string]interface{}{"id": policy.ID})
}

// HandleUpdateResourcePolicy POST /admin/resource-policies/update.
func HandleUpdateResourcePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	var req resourcePolicyMutationRequest
	if err := decodeResourcePolicyRequest(w, r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}
	configJSON, _, err := normalizeResourcePolicyConfig(r, req.ResourceConfig)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	existing, err := model.GetResourcePolicy(r.Context(), req.ID)
	if err != nil {
		writeResourcePolicyModelError(w, r, err)
		return
	}
	if existing.IsDefault && strings.TrimSpace(req.Name) == existing.DisplayName(r.Context()) {
		// The API exposes a localized default name. Treat an unchanged localized
		// display value as the canonical persisted name for round-trip updates.
		req.Name = existing.Name
	}
	if !existing.IsDefault {
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNameCannotBeEmpty))
			return
		}
		if len([]rune(req.Name)) > 128 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "name"))
			return
		}
		if len(req.GroupIDs) == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_ids"))
			return
		}
	}
	if _, err := model.UpdateResourcePolicy(r.Context(), req.ID, req.Name, configJSON, req.GroupIDs); err != nil {
		writeResourcePolicyModelError(w, r, err)
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeleteResourcePolicy POST /admin/resource-policies/delete.
func HandleDeleteResourcePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	var req struct {
		ID uint `json:"id"`
	}
	if err := decodeResourcePolicyRequest(w, r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}
	if err := model.DeleteResourcePolicy(r.Context(), req.ID); err != nil {
		writeResourcePolicyModelError(w, r, err)
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}
