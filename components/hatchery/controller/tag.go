package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"

	"hatchery/i18n"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencenttag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	"gorm.io/gorm"
)

// tagClient 抽象 Tag API 调用，便于测试 mock
type tagClient interface {
	DescribeTagKeys(request *tencenttag.DescribeTagKeysRequest) (*tencenttag.DescribeTagKeysResponse, error)
	DescribeTagValues(request *tencenttag.DescribeTagValuesRequest) (*tencenttag.DescribeTagValuesResponse, error)
}

// newTagClientFunc 创建 Tag 客户端的工厂函数，测试时可替换
var newTagClientFunc = func(ctx context.Context) (tagClient, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgGetCloudCredentialFailed)
	}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tag.tencentcloudapi.com"
	return tencenttag.NewClient(credential, "", cpf)
}

// tagPageSize 每次查询标签的分页大小
const tagPageSize uint64 = 1000

// HandleGetTagKeys 查询当前账号下所有标签键列表（自动分页获取全量）
// GET /api/tags/keys
func HandleGetTagKeys(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	client, err := newTagClientFunc(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateTagClientFailed))
		return
	}

	// 分页循环获取全量标签键
	keys := make([]string, 0)
	var offset uint64
	for {
		request := tencenttag.NewDescribeTagKeysRequest()
		request.Limit = common.Uint64Ptr(tagPageSize)
		request.Offset = common.Uint64Ptr(offset)
		request.Category = common.StringPtr("Custom") // 只查询自定义标签，排除系统标签

		response, err := client.DescribeTagKeys(request)
		if err != nil {
			slog.Error("[Tag] DescribeTagKeys 失败", "error", err, "offset", offset)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryTagKeysFailed))
			return
		}

		if response.Response != nil && response.Response.Tags != nil {
			for _, t := range response.Response.Tags {
				if t != nil {
					keys = append(keys, *t)
				}
			}
		}

		// 本页不足 pageSize 条，说明已获取全部
		pageCount := uint64(0)
		if response.Response != nil && response.Response.Tags != nil {
			pageCount = uint64(len(response.Response.Tags))
		}
		if pageCount < tagPageSize {
			break
		}
		offset += pageCount
	}

	jsonOK(w, map[string]interface{}{
		"keys": keys,
	})
}

// HandleGetTagValues 查询指定标签键的所有值列表（自动分页获取全量）
// GET /api/tags/values?key=xxx
func HandleGetTagValues(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	tagKey := r.URL.Query().Get("key")
	if tagKey == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgKeyParamRequired))
		return
	}

	client, err := newTagClientFunc(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateTagClientFailed))
		return
	}

	// 分页循环获取指定键的全量值
	values := make([]string, 0)
	var offset uint64
	for {
		request := tencenttag.NewDescribeTagValuesRequest()
		request.TagKeys = common.StringPtrs([]string{tagKey})
		request.Limit = common.Uint64Ptr(tagPageSize)
		request.Offset = common.Uint64Ptr(offset)
		request.Category = common.StringPtr("Custom") // 只查询自定义标签，排除系统标签

		response, err := client.DescribeTagValues(request)
		if err != nil {
			slog.Error("[Tag] DescribeTagValues 失败", "error", err, "key", tagKey, "offset", offset)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryTagValuesFailed))
			return
		}

		if response.Response != nil && response.Response.Tags != nil {
			for _, t := range response.Response.Tags {
				if t != nil && t.TagValue != nil {
					values = append(values, *t.TagValue)
				}
			}
		}

		pageCount := uint64(0)
		if response.Response != nil && response.Response.Tags != nil {
			pageCount = uint64(len(response.Response.Tags))
		}
		if pageCount < tagPageSize {
			break
		}
		offset += pageCount
	}

	jsonOK(w, map[string]interface{}{
		"key":    tagKey,
		"values": values,
	})
}

// DefaultTag 默认标签键值对
type DefaultTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ParseDefaultTags 解析 SiteConfig.DefaultTags JSON 字符串，格式错误返回空数组
func ParseDefaultTags(raw string) []DefaultTag {
	items := model.ParseTagItems(raw)
	if raw != "" && len(items) == 0 && strings.TrimSpace(raw) != "[]" {
		slog.Warn("[Tag] 默认标签配置格式错误，跳过", "raw", raw)
	}
	tags := make([]DefaultTag, 0, len(items))
	for _, item := range items {
		tags = append(tags, DefaultTag{Key: item.Key, Value: item.Value})
	}
	return tags
}

type adminTagPayload struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	VisibilityType string `json:"visibility_type"`
	GroupIDs       []uint `json:"group_ids"`
}

type adminTagsUpdatePayload struct {
	Tags []adminTagPayload `json:"tags"`
}

type adminTagResponse struct {
	ID             uint   `json:"id"`
	Key            string `json:"key"`
	Value          string `json:"value"`
	VisibilityType string `json:"visibility_type"`
	GroupIDs       []uint `json:"group_ids"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// HandleAdminTags lists managed default tags.
// GET /admin/tags
func HandleAdminTags(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	if err := model.EnsureLegacyDefaultTagsMigrated(r.Context()); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMigrateDefaultTagsFailed).WithDetail(err.Error()))
		return
	}
	rows, groupMap, err := model.ListTags(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	items := make([]adminTagResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, newAdminTagResponse(row, groupMap[row.ID]))
	}
	jsonOK(w, map[string]interface{}{"tags": items})
}

// HandleCreateTag creates one managed tag key/value.
// POST /admin/tags/create
func HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	req, ok := decodeAdminTagPayload(w, r)
	if !ok {
		return
	}
	if err := validateTagGroups(r, req.VisibilityType, req.GroupIDs); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	row, err := model.CreateTag(r.Context(), req.Key, req.Value, normalizeVisibilityType(req.VisibilityType), req.GroupIDs)
	if err != nil {
		writeTagWriteError(w, r, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	jsonOK(w, map[string]interface{}{"tag": newAdminTagResponse(row, req.GroupIDs)})
}

// HandleUpdateTag updates one managed tag.
// POST /admin/tags/update?id=N
func HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	id, err := parseTagID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	req, ok := decodeAdminTagPayload(w, r)
	if !ok {
		return
	}
	if err := validateTagGroups(r, req.VisibilityType, req.GroupIDs); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	row, updErr := model.UpdateTag(r.Context(), id, req.Key, req.Value, normalizeVisibilityType(req.VisibilityType), req.GroupIDs)
	if updErr != nil {
		writeTagWriteError(w, r, hcommon.I18nRichError(updErr, i18n.MsgOperationFailed))
		return
	}
	jsonOK(w, map[string]interface{}{"tag": newAdminTagResponse(row, req.GroupIDs)})
}

// HandleReplaceAllTags replaces all managed tags.
// POST /admin/tags/replace-all
func HandleReplaceAllTags(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	var req adminTagsUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidRequestFormat))
		return
	}
	items := make([]model.TagWithScope, 0, len(req.Tags))
	for _, tagReq := range req.Tags {
		tagReq.VisibilityType = normalizeVisibilityType(tagReq.VisibilityType)
		if tagReq.VisibilityType == model.VisibilityAll {
			tagReq.GroupIDs = nil
		}
		if err := validateTagGroups(r, tagReq.VisibilityType, tagReq.GroupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		items = append(items, model.TagWithScope{
			Key:            tagReq.Key,
			Value:          tagReq.Value,
			VisibilityType: tagReq.VisibilityType,
			GroupIDs:       tagReq.GroupIDs,
		})
	}
	if err := model.ReplaceTags(r.Context(), items); err != nil {
		writeTagWriteError(w, r, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeleteTag deletes one managed tag.
// POST /admin/tags/delete?id=N
func HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	id, err := parseTagID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := model.DeleteTag(r.Context(), id); err != nil {
		writeTagWriteError(w, r, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

func decodeAdminTagPayload(w http.ResponseWriter, r *http.Request) (adminTagPayload, bool) {
	var req adminTagPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidRequestFormat))
		return req, false
	}
	req.VisibilityType = normalizeVisibilityType(req.VisibilityType)
	if req.VisibilityType == model.VisibilityAll {
		req.GroupIDs = nil
	}
	return req, true
}

func normalizeVisibilityType(v string) string {
	if v == "" {
		return model.VisibilityAll
	}
	return v
}

func newAdminTagResponse(row model.Tag, groupIDs []uint) adminTagResponse {
	if groupIDs == nil {
		groupIDs = []uint{}
	}
	return adminTagResponse{
		ID:             row.ID,
		Key:            row.TagKey,
		Value:          row.TagValue,
		VisibilityType: row.VisibilityType,
		GroupIDs:       groupIDs,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
}

func parseTagID(r *http.Request) (uint, error) {
	raw := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "id")
	}
	return uint(id), nil
}

func validateTagGroups(r *http.Request, visibilityType string, groupIDs []uint) error {
	if visibilityType != model.VisibilityGroup {
		return nil
	}
	groups, err := model.GetGroupsByIDs(r.Context(), groupIDs)
	if err != nil {
		return hcommon.I18nError(i18n.MsgQueryGroupFailed).WithDetail(err.Error())
	}
	exists := make(map[uint]struct{}, len(groups))
	for _, g := range groups {
		exists[g.ID] = struct{}{}
	}
	for _, id := range groupIDs {
		if _, ok := exists[id]; !ok {
			return hcommon.I18nError(i18n.MsgGroupNotFound).WithDetail(fmt.Sprintf("id=%d", id))
		}
	}
	return nil
}

func writeTagWriteError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgTagNotFound))
		return
	}
	writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
}
