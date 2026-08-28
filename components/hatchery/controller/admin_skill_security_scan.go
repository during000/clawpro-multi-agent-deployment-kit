package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ── 响应结构 ──────────────────────────────────────────────────────────────

// scanStatusResp 列表页精简版安全状态
type scanStatusResp struct {
	ScanStatus    string     `json:"scan_status"`              // not_scanned/scanning/safe/suspicious/malicious
	RiskLevel     string     `json:"risk_level,omitempty"`     // benign/suspicious/malicious
	SecurityScore *int       `json:"security_score,omitempty"` // 0-100，指针避免 0 被 omitempty 吞掉
	ScannedAt     *time.Time `json:"scanned_at,omitempty"`
	ReportURL     string     `json:"report_url,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"` // 仅 scanning 时有值
}

// scanDetailResp 详情页完整版安全状态
type scanDetailResp struct {
	ScanStatus      string      `json:"scan_status"`
	ScanID          uint        `json:"scan_id,omitempty"`
	RiskLevel       string      `json:"risk_level,omitempty"`
	SecurityScore   *int        `json:"security_score,omitempty"`
	ScannedAt       *time.Time  `json:"scanned_at,omitempty"`
	ReportURL       string      `json:"report_url,omitempty"`
	CreatedAt       *time.Time  `json:"created_at,omitempty"`
	RiskDescription string      `json:"risk_description,omitempty"`
	Mitigation      string      `json:"mitigation,omitempty"`
	ScanItems       interface{} `json:"scan_items,omitempty"`
	CapabilityTags  interface{} `json:"capability_tags,omitempty"`
}

// buildScanStatusResp 从 DB scan 记录构建列表页精简响应
func buildScanStatusResp(scan *model.SkillSecurityScan) *scanStatusResp {
	if scan == nil {
		return &scanStatusResp{ScanStatus: "not_scanned"}
	}

	resp := &scanStatusResp{}

	switch scan.Status {
	case "SCANNING":
		resp.ScanStatus = "scanning"
		resp.CreatedAt = &scan.CreatedAt
	case "SUCCESS":
		resp.ScanStatus = mapRiskLevelToStatus(scan.RiskLevel)
		resp.RiskLevel = scan.RiskLevel
		score := scan.SecurityScore
		resp.SecurityScore = &score
		resp.ScannedAt = scan.ScannedAt
		resp.ReportURL = scan.ReportURL
	default: // FAILED 或其他
		resp.ScanStatus = "not_scanned"
	}

	return resp
}

// buildScanDetailResp 从 DB scan 记录构建详情页完整响应
func buildScanDetailResp(scan *model.SkillSecurityScan) *scanDetailResp {
	if scan == nil {
		return &scanDetailResp{ScanStatus: "not_scanned"}
	}

	resp := &scanDetailResp{
		ScanID: scan.ID,
	}

	switch scan.Status {
	case "SCANNING":
		resp.ScanStatus = "scanning"
		resp.CreatedAt = &scan.CreatedAt
	case "SUCCESS":
		resp.ScanStatus = mapRiskLevelToStatus(scan.RiskLevel)
		resp.RiskLevel = scan.RiskLevel
		score := scan.SecurityScore
		resp.SecurityScore = &score
		resp.ScannedAt = scan.ScannedAt
		resp.ReportURL = scan.ReportURL

		// 从 scan_result_data JSON 解析详细信息
		if scan.ScanResultData != nil && len(scan.ScanResultData) > 0 {
			var data map[string]interface{}
			if json.Unmarshal(scan.ScanResultData, &data) == nil {
				if rd, ok := data["RiskDescription"].(string); ok {
					resp.RiskDescription = rd
				}
				if mt, ok := data["Mitigation"].(string); ok {
					resp.Mitigation = mt
				}
				if si, ok := data["ScanItems"]; ok {
					resp.ScanItems = si
				}
				if ct, ok := data["CapabilityTags"]; ok {
					resp.CapabilityTags = ct
				}
			}
		}
	default:
		resp.ScanStatus = "not_scanned"
	}

	return resp
}

// mapRiskLevelToStatus 将 CSIP RiskLevel 映射为前端 scan_status
func mapRiskLevelToStatus(riskLevel string) string {
	switch riskLevel {
	case "benign":
		return "safe"
	case "suspicious":
		return "suspicious"
	case "malicious":
		return "malicious"
	default:
		return "safe" // 无风险等级时默认安全
	}
}

// mapToScanStatusString 从 scan 记录返回简单的状态字符串（用户端列表）
func mapToScanStatusString(scan *model.SkillSecurityScan) string {
	if scan == nil {
		return "not_scanned"
	}
	switch scan.Status {
	case "SCANNING":
		return "scanning"
	case "SUCCESS":
		return mapRiskLevelToStatus(scan.RiskLevel)
	default:
		return "not_scanned"
	}
}

// ── HTTP Handlers ─────────────────────────────────────────────────────────

// HandleSkillScanTrigger 手动触发技能安全检测
// 路由: POST /admin/skills/scan-trigger
func HandleSkillScanTrigger(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		SkillID uint `json:"skill_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if req.SkillID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skill_id"))
		return
	}

	// 查询技能
	var skill model.Skill
	if err := model.DB(r.Context()).Where("id = ?", req.SkillID).First(&skill).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		}
		return
	}

	// 文件大小预检（无需下载即可判断）
	if skill.FileSize > int64(maxScanFileSize) {
		writeError(w, r, http.StatusBadRequest, ErrFileTooLargeForScan)
		return
	}

	// 检查是否已有进行中的扫描
	var existing model.SkillSecurityScan
	if model.DB(r.Context()).Where("skill_id = ? AND status = ?", req.SkillID, "SCANNING").
		First(&existing).Error == nil {
		writeError(w, r, http.StatusConflict, ErrScanAlreadyInProgress)
		return
	}

	// 下载 zip 文件
	cosZipKey := skill.COSZipKey
	if cosZipKey == "" {
		cosZipKey = fmt.Sprintf("%s/%s-%s.zip", skill.Slug, skill.Slug, skill.Version)
	}
	downloadURL, err := buildSMHDownloadURL(r.Context(), cosZipKey, false)
	if err != nil {
		slog.Error("[SkillScan] 生成下载 URL 失败", "skill_id", req.SkillID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGenerateDownloadLinkFailed))
		return
	}

	resp, err := SkillHTTPClient.Get(downloadURL)
	if err != nil {
		slog.Error("[SkillScan] 下载 zip 失败", "skill_id", req.SkillID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDownloadSkillFileFailed))
		return
	}
	zipData, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil || resp.StatusCode != http.StatusOK {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgReadSkillFileFailed, resp.StatusCode))
		return
	}

	// 触发扫描
	fileName := fmt.Sprintf("%s-%s.zip", skill.Slug, skill.Version)
	scan, err := CreateSkillSecurityScan(r.Context(), zipData, skill.ID, skill.Version, fileName)
	if err != nil {
		// LimitExceeded → 402
		rerr := hcommon.EnsureRichErrorOrPanic(err)
		var limitErr *ScanLimitError
		if errors.As(rerr, &limitErr) {
			writeError(w, r, http.StatusPaymentRequired, rerr.WithCustomData(map[string]any{
				"code": limitErr.Code,
			}))
			return
		}
		slog.Error("[SkillScan] 触发扫描失败", "skill_id", req.SkillID, "error", rerr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(rerr, i18n.MsgTriggerScanFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"scan_id": scan.ID,
		"status":  "scanning",
		"message": i18n.T(r.Context(), i18n.MsgSkillScanSubmitted),
	})
}

// HandleSkillScanConfigRouter 安全扫描默认配置路由分发
// 路由: GET/POST /admin/skills/scan-config
func HandleSkillScanConfigRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		HandleSkillScanConfig(w, r)
	case http.MethodPost:
		HandleSetSkillScanConfig(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// HandleSkillScanConfig 查询安全扫描默认配置
// 路由: GET /admin/skills/scan-config
func HandleSkillScanConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	config := model.GetSiteConfig(r.Context())
	jsonOK(w, map[string]interface{}{
		"skill_scan_default_enabled": config.SkillScanDefaultEnabled,
	})
}

// HandleSetSkillScanConfig 设置安全扫描默认配置
// 路由: POST /admin/skills/scan-config
func HandleSetSkillScanConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		SkillScanDefaultEnabled bool `json:"skill_scan_default_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}

	if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{
		"skill_scan_default_enabled": req.SkillScanDefaultEnabled,
	}); err != nil {
		slog.Error("[SkillScan] 更新扫描默认配置失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateConfigFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":                         true,
		"skill_scan_default_enabled": req.SkillScanDefaultEnabled,
	})
}
