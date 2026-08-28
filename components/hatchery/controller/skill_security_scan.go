package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// ── 常量 & 错误 ────────────────────────────────────────────────────────────

const maxScanFileSize = 7 * 1024 * 1024 // 7MB，CreateSkillScan API 编码前上限
const scanTimeout = 30 * time.Minute    // 超时保护：SCANNING 超过此时间标记为 FAILED

var (
	ErrFileTooLargeForScan   = hcommon.I18nError(i18n.MsgFileTooLargeForScan)
	ErrScanAlreadyInProgress = hcommon.I18nError(i18n.MsgScanInProgress)
)

// ScanLimitError 表示 CSIP 返回 LimitExceeded（试用未开通/已到期/额度用完）
type ScanLimitError struct {
	Code    string // "LimitExceeded"
	Message string // CSIP 原始 Message
}

func (e *ScanLimitError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ── CSIP API 响应结构 ─────────────────────────────────────────────────────

// createSkillScanResp 对应 CreateSkillScan 的响应
type createSkillScanResp struct {
	Response struct {
		ContentHash   string `json:"ContentHash"`
		EngineVersion int    `json:"EngineVersion"`
		Status        string `json:"Status"`
		Message       string `json:"Message"`
		RequestId     string `json:"RequestId"`
		Error         *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

// describeSkillScanResultResp 对应 DescribeSkillScanResult 的响应
type describeSkillScanResultResp struct {
	Response struct {
		Status    string                 `json:"Status"` // SUCCESS, SCANNING, FAILED, NOT_FOUND
		Data      map[string]interface{} `json:"Data"`   // 完整结果 JSON，结构随 Status 变化
		RequestId string                 `json:"RequestId"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

// scanResultData 解析 DescribeSkillScanResult.Data 中的关键字段（SUCCESS 时）
type scanResultData struct {
	SkillName       string            `json:"SkillName"`
	ContentHash     string            `json:"ContentHash"`
	RiskLevel       string            `json:"RiskLevel"`
	PrimaryRuleID   string            `json:"PrimaryRuleID"`
	SecurityScore   int               `json:"SecurityScore"`
	EngineVersion   int               `json:"EngineVersion"`
	ReportURL       string            `json:"ReportURL"`
	RiskDescription string            `json:"RiskDescription"`
	Mitigation      string            `json:"Mitigation"`
	ScanItems       []scanItem        `json:"ScanItems"`
	CapabilityTags  []capabilityTag   `json:"CapabilityTags"`
	RuleCatalog     []ruleCatalogItem `json:"RuleCatalog"`
	ScannedAt       string            `json:"ScannedAt"`
	FailedAt        string            `json:"FailedAt"`
	Message         string            `json:"Message"`
}

type scanItem struct {
	ScanType string          `json:"ScanType"`
	RuleList []ruleViolation `json:"RuleList"`
}

type ruleViolation struct {
	RuleID      string `json:"RuleID"`
	Description string `json:"Description"`
}

type capabilityTag struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type ruleCatalogItem struct {
	RuleID   string `json:"RuleID"`
	RuleName string `json:"RuleName"`
}

// ── 通用 CSIP 调用 ────────────────────────────────────────────────────────

// callCSIPAction 通过 CommonClient 调用 CSIP API
// 使用与 doCloudProxy 相同的模式，因 SDK 可能尚未封装 SkillScan 方法
func callCSIPAction(ctx context.Context, action string, params map[string]interface{}) ([]byte, error) {
	cred, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "csip.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"

	client := common.NewCommonClient(cred, CVMRegion, cpf)

	request := tchttp.NewCommonRequest("csip", "2022-11-21", action)
	body, _ := json.Marshal(params)
	if err := request.SetActionParameters(string(body)); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCSIPParamError)
	}

	response := tchttp.NewCommonResponse()
	if err := client.Send(request, response); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCSIPAPICallFailedWithAction, action)
	}

	return response.GetBody(), nil
}

// parseCSIPError 解析 CSIP 响应中的 Error 字段
func parseCSIPError(code, message string) error {
	if code == "LimitExceeded" {
		return hcommon.I18nRichError(&ScanLimitError{Code: code, Message: message}, i18n.MsgCSIPAPIErrorWithCodeMessage, code, message)
	}
	return hcommon.I18nError(i18n.MsgCSIPAPIErrorWithCodeMessage, code, message)
}

// ── 核心业务逻辑 ──────────────────────────────────────────────────────────

// CreateSkillSecurityScan 为技能创建安全扫描任务
// 调用 CSIP CreateSkillScan API，成功后创建 DB 记录
func CreateSkillSecurityScan(ctx context.Context, zipData []byte, skillID uint, skillVersion, fileName string) (*model.SkillSecurityScan, error) {
	// 1. 文件大小校验
	if len(zipData) > maxScanFileSize {
		return nil, ErrFileTooLargeForScan
	}

	// 2. 调用 CSIP CreateSkillScan API
	respBody, rerr := callCSIPAction(ctx, "CreateSkillScan", map[string]interface{}{
		"FileBase64": base64.StdEncoding.EncodeToString(zipData),
		"FileName":   fileName,
	})
	if rerr != nil {
		return nil, rerr
	}

	// 3. 解析响应
	var resp createSkillScanResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCSIPFailedToParseResponse)
	}

	// 4. 错误处理
	if resp.Response.Error != nil {
		return nil, parseCSIPError(resp.Response.Error.Code, resp.Response.Error.Message)
	}

	contentHash := resp.Response.ContentHash
	engineVersion := resp.Response.EngineVersion

	slog.Info("[SkillScan] CreateSkillScan 成功",
		"skill_id", skillID, "content_hash", contentHash, "engine_version", engineVersion)

	// 5. 幂等检查：是否已存在相同 hash+engine 的记录
	existing, err := model.GetSkillSecurityScanByHash(ctx, contentHash, engineVersion)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCSIPQueryScanRecordFailed)
	}
	if existing != nil {
		// 同 hash+engine 记录已存在
		if existing.Status == "SCANNING" {
			// 仍在进行中，关联到当前 skill（可能是不同 skill 引用同一文件）
			return existing, nil
		}
		// SUCCESS 或 FAILED，复用结果
		return existing, nil
	}

	// 6. 创建 DB 记录
	scan := &model.SkillSecurityScan{
		SkillID:       skillID,
		SkillVersion:  skillVersion,
		ContentHash:   contentHash,
		EngineVersion: engineVersion,
		Status:        "SCANNING",
	}
	if err := model.DB(ctx).Create(scan).Error; err != nil {
		// 唯一索引冲突（并发场景），查询已有记录返回
		existingAgain, queryErr := model.GetSkillSecurityScanByHash(ctx, contentHash, engineVersion)
		if queryErr != nil {
			slog.Error("[SkillScan] 唯一索引冲突后查询失败", "error", queryErr)
		}
		if existingAgain != nil {
			return existingAgain, nil
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgCSIPFailedToCreateScanRecord)
	}

	slog.Info("[SkillScan] 扫描记录已创建", "scan_id", scan.ID, "skill_id", skillID)
	return scan, nil
}

// PollSkillSecurityScanResults 后台轮询所有 SCANNING 状态的扫描任务
func PollSkillSecurityScanResults(ctx context.Context) error {
	scans, err := model.GetPendingScanRecords(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCSIPQueryPendingScansFailed)
	}

	if len(scans) == 0 {
		return nil
	}

	slog.Debug("[SkillScan] 轮询扫描结果", "pending_count", len(scans))

	for i := range scans {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
		if err := pollSingleScan(ctxWithTimeout, &scans[i]); err != nil {
			slog.Error("[SkillScan] 轮询单个扫描失败", "scan_id", scans[i].ID, "error", err)
		}
		cancel()
	}

	return nil
}

// pollSingleScan 轮询单个扫描任务的结果
func pollSingleScan(ctx context.Context, scan *model.SkillSecurityScan) error {
	// 超时保护
	if time.Since(scan.CreatedAt) > scanTimeout {
		now := time.Now()
		if err := model.DB(ctx).Model(scan).Updates(map[string]interface{}{
			"status":          "FAILED",
			"failed_at":       now,
			"failure_message": "检测超时（超过 30 分钟未返回结果）",
		}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCSIPUpdateTimeoutStatusFailed)
		}
		slog.Warn("[SkillScan] 扫描超时，标记为 FAILED", "scan_id", scan.ID)
		return nil
	}

	// 调用 DescribeSkillScanResult
	respBody, err := callCSIPAction(ctx, "DescribeSkillScanResult", map[string]interface{}{
		"ContentHash":   scan.ContentHash,
		"EngineVersion": scan.EngineVersion,
	})
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCSIPDescribeResultCallFailed)
	}

	var resp describeSkillScanResultResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCSIPParseDescribeResultFailed)
	}

	if resp.Response.Error != nil {
		return hcommon.I18nError(i18n.MsgCSIPDescribeResultError,
			resp.Response.Error.Code, resp.Response.Error.Message)
	}

	switch resp.Response.Status {
	case "SCANNING":
		// 仍在进行中，不做处理
		return nil

	case "SUCCESS":
		return handleScanSuccess(ctx, scan, resp.Response.Data)

	case "FAILED":
		return handleScanFailed(ctx, scan, resp.Response.Data)

	case "NOT_FOUND":
		// 理论上不应出现（我们刚调过 CreateSkillScan），标记为 FAILED
		now := time.Now()
		return model.DB(ctx).Model(scan).Updates(map[string]interface{}{
			"status":          "FAILED",
			"failed_at":       now,
			"failure_message": "检测记录未找到 (NOT_FOUND)",
		}).Error

	default:
		slog.Warn("[SkillScan] 未知状态", "scan_id", scan.ID, "status", resp.Response.Status)
		return nil
	}
}

// handleScanSuccess 处理扫描成功的结果
func handleScanSuccess(ctx context.Context, scan *model.SkillSecurityScan, data map[string]interface{}) error {
	// 将 data map 转为结构化类型
	dataBytes, _ := json.Marshal(data)
	var result scanResultData
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCSIPParseScanDetailFailed)
	}

	// 解析扫描完成时间
	var scannedAt *time.Time
	if result.ScannedAt != "" {
		if t, err := time.Parse(time.RFC3339, result.ScannedAt); err == nil {
			scannedAt = &t
		}
	}

	// 更新扫描记录
	updates := map[string]interface{}{
		"status":           "SUCCESS",
		"risk_level":       result.RiskLevel,
		"primary_rule_id":  result.PrimaryRuleID,
		"security_score":   result.SecurityScore,
		"report_url":       result.ReportURL,
		"scanned_at":       scannedAt,
		"scan_result_data": dataBytes, // 保存完整 JSON
	}
	if err := model.DB(ctx).Model(scan).Updates(updates).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCSIPUpdateScanSuccessFailed)
	}

	// 保存违规项
	if err := saveScanViolations(ctx, scan.ID, result.ScanItems, result.RuleCatalog); err != nil {
		slog.Error("[SkillScan] 保存违规项失败", "scan_id", scan.ID, "error", err)
		// 不返回错误，主记录已更新成功
	}

	slog.Info("[SkillScan] 扫描完成",
		"scan_id", scan.ID, "risk_level", result.RiskLevel, "score", result.SecurityScore)
	return nil
}

// handleScanFailed 处理扫描失败的结果
func handleScanFailed(ctx context.Context, scan *model.SkillSecurityScan, data map[string]interface{}) error {
	var failedAt *time.Time
	var message string

	if data != nil {
		if fa, ok := data["FailedAt"].(string); ok && fa != "" {
			if t, err := time.Parse(time.RFC3339, fa); err == nil {
				failedAt = &t
			}
		}
		if msg, ok := data["Message"].(string); ok {
			message = msg
		}
	}
	if failedAt == nil {
		now := time.Now()
		failedAt = &now
	}
	if message == "" {
		message = "检测失败（CSIP 未返回具体原因）"
	}

	if err := model.DB(ctx).Model(scan).Updates(map[string]interface{}{
		"status":          "FAILED",
		"failed_at":       failedAt,
		"failure_message": message,
	}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCSIPUpdateScanFailedFailed)
	}

	slog.Warn("[SkillScan] 扫描失败", "scan_id", scan.ID, "message", message)
	return nil
}

// saveScanViolations 将 CSIP 返回的 ScanItems 展开为 SkillScanViolation 记录
func saveScanViolations(ctx context.Context, scanID uint, scanItems []scanItem, ruleCatalog []ruleCatalogItem) error {
	// 构建 ruleID → ruleName 映射
	ruleNameMap := make(map[string]string)
	for _, rc := range ruleCatalog {
		ruleNameMap[rc.RuleID] = rc.RuleName
	}

	for _, item := range scanItems {
		for _, rule := range item.RuleList {
			violation := &model.SkillScanViolation{
				SkillSecurityScanID: scanID,
				RuleID:              rule.RuleID,
				RuleName:            ruleNameMap[rule.RuleID],
				ScanType:            item.ScanType,
				Description:         rule.Description,
			}
			if err := model.DB(ctx).Create(violation).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgCSIPCreateViolationFailed, rule.RuleID)
			}
		}
	}
	return nil
}
