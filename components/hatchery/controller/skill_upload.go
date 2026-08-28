package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

const skillUploadMaxSize = 100 << 20 // 100MB

// isSkillUploadTooLarge 判断上传文件是否超过 Skill 上传上限。
func isSkillUploadTooLarge(size int64) bool {
	return size > skillUploadMaxSize
}

// preparedSkillUpload 技能上传预处理结果（表单校验 + zip 规范化 + COS 路径）。
type preparedSkillUpload struct {
	Skill       model.Skill
	ZipData     []byte
	Files       []string // COS 路径列表
	CosZipKey   string
	CosDirKey   string
	SlugPrefix  string
	MaxExisting model.Skill // 同 slug 最高版本；ID==0 表示不存在
}

// skillUploadError 带 HTTP 状态码的上传预处理/存储错误。
type skillUploadError struct {
	Status int
	Err    error
}

func (e *skillUploadError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func newSkillUploadError(status int, err error) *skillUploadError {
	return &skillUploadError{Status: status, Err: err}
}

// prepareSkillUploadFromForm 解析 multipart 表单，完成版本校验、zip 规范化与元数据填充。
// ownerID 写入 _meta.json 的 ownerId；Skill.Status / UploaderID 由调用方按场景设置。
func prepareSkillUploadFromForm(r *http.Request, ownerID uint) (*preparedSkillUpload, *skillUploadError) {
	if err := r.ParseMultipartForm(skillUploadMaxSize); err != nil {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nRichError(err, i18n.MsgRequestBodyTooLargeWithError, err))
	}

	slug := r.FormValue("slug")
	name := r.FormValue("name")
	version := r.FormValue("version")
	if slug == "" || name == "" || version == "" {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgPluginSlugNameVerRequired))
	}
	if !isValidSlug(slug) {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgPluginInvalidSlug))
	}

	skill := model.Skill{
		Slug:        slug,
		Name:        name,
		Description: r.FormValue("description"),
		Version:     version,
		Changelog:   r.FormValue("changelog"),
	}
	if err := skill.ParseVersion(); err != nil {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nRichError(err, i18n.MsgBadRequestParamInvalid, "version"))
	}

	// 版本递增校验：新版本号必须大于该 slug 的最高现有版本（查所有状态）
	var maxExisting model.Skill
	if model.DB(r.Context()).Where("slug = ?", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&maxExisting).Error == nil {
		newScore := skill.VersionMajor*1000000 + skill.VersionMinor*1000 + skill.VersionPatch
		existingScore := maxExisting.VersionMajor*1000000 + maxExisting.VersionMinor*1000 + maxExisting.VersionPatch
		if newScore <= existingScore {
			return nil, newSkillUploadError(http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgSkillNewVersionMustBeGreater, version, maxExisting.Version))
		}
	}

	// 检查 (slug, version) 唯一（含软删除记录）
	var existing model.Skill
	err := model.DB(r.Context()).Unscoped().
		Where("slug = ? AND version = ?", slug, version).
		First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			model.DB(r.Context()).Unscoped().Delete(&model.Skill{}, existing.ID)
			model.DB(r.Context()).Where("skill_id = ?", existing.ID).Delete(&model.SkillCategoryMapping{})
		} else {
			return nil, newSkillUploadError(http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgSkillVersionExist, slug, version))
		}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgPluginFileFieldMissing))
	}
	defer file.Close()

	if isSkillUploadTooLarge(header.Size) {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgSkillUploadFileSizeTooLarge))
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, newSkillUploadError(http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgPluginReadUploadFail))
	}
	zipData := buf.Bytes()

	files, normalizedZipData, zipErr := validateSkillZip(zipData, slug)
	if zipErr != nil {
		return nil, newSkillUploadError(http.StatusBadRequest, zipErr)
	}
	zipData = normalizedZipData

	ownerIDStr := ""
	if ownerID > 0 {
		ownerIDStr = fmt.Sprintf("%d", ownerID)
	}
	zipData, err = injectMetaIntoZip(zipData, slug, map[string]string{
		"ownerId": ownerIDStr,
		"slug":    slug,
		"version": version,
	})
	if err != nil {
		return nil, newSkillUploadError(http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgSkillInjectMetaFail))
	}

	// 从注入后的最终 zip 中提取文件列表，确保与存储上传内容完全一致
	files = nil
	if finalZR, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData))); err == nil {
		for _, f := range finalZR.File {
			if !f.FileInfo().IsDir() {
				files = append(files, f.Name)
			}
		}
	}

	cosZipKey := fmt.Sprintf("%s/%s-%s.zip", slug, slug, version)
	cosDirKey := fmt.Sprintf("%s/%s-%s/", slug, slug, version)
	slugPrefix := slug + "/"
	for i, f := range files {
		files[i] = cosDirKey + strings.TrimPrefix(f, slugPrefix)
	}
	fileListJSON, _ := json.Marshal(files)
	if isFileListTooLarge(fileListJSON) {
		return nil, newSkillUploadError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgSkillFileListTooLarge))
	}

	skill.COSZipKey = cosZipKey
	skill.COSDirKey = cosDirKey
	skill.FileList = string(fileListJSON)
	skill.FileSize = header.Size

	return &preparedSkillUpload{
		Skill:       skill,
		ZipData:     zipData,
		Files:       files,
		CosZipKey:   cosZipKey,
		CosDirKey:   cosDirKey,
		SlugPrefix:  slugPrefix,
		MaxExisting: maxExisting,
	}, nil
}

// createSkillCategoryMappings 根据逗号分隔的 category_ids 创建分类关联。
func createSkillCategoryMappings(tx *gorm.DB, skillID uint, catIDsCSV string) {
	if catIDsCSV == "" {
		return
	}
	for _, s := range strings.Split(catIDsCSV, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
			_ = tx.Create(&model.SkillCategoryMapping{SkillID: skillID, CategoryID: uint(id)}).Error
		}
	}
}

// uploadSkillPackageToStorage 上传 zip 及解压文件到存储；失败时异步清理已上传内容。
func uploadSkillPackageToStorage(ctx context.Context, zipData []byte, cosZipKey, cosDirKey, slugPrefix string) *skillUploadError {
	storageClient, storageErr := getStorageClient(ctx)
	if storageErr != nil {
		return newSkillUploadError(http.StatusInternalServerError,
			hcommon.I18nRichError(storageErr, i18n.MsgPluginSMHUnavailable))
	}

	if err := storageClient.Upload(cosZipKey, zipData, "application/zip"); err != nil {
		return newSkillUploadError(http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgSkillUploadZipFail))
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		cleanupSkillPackage(ctx, cosZipKey, cosDirKey)
		return newSkillUploadError(http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgZipParseFail, err))
	}

	type fileItem struct {
		name    string
		fileKey string
		data    []byte
	}
	var items []fileItem
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			cleanupSkillPackage(ctx, cosZipKey, cosDirKey)
			return newSkillUploadError(http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgZipReadEntryFail, err))
		}
		var fbuf bytes.Buffer
		_, readErr := fbuf.ReadFrom(rc)
		rc.Close()
		if readErr != nil {
			cleanupSkillPackage(ctx, cosZipKey, cosDirKey)
			return newSkillUploadError(http.StatusInternalServerError,
				hcommon.I18nRichError(readErr, i18n.MsgZipReadEntryFail, readErr))
		}
		items = append(items, fileItem{
			name:    f.Name,
			fileKey: cosDirKey + strings.TrimPrefix(f.Name, slugPrefix),
			data:    fbuf.Bytes(),
		})
	}

	const maxConcurrency = 8
	sem := make(chan struct{}, maxConcurrency)
	var uploadErr error
	var errOnce sync.Once
	var failed atomic.Bool
	var wg sync.WaitGroup

	for _, item := range items {
		wg.Add(1)
		go func(it fileItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if failed.Load() {
				return
			}
			if err := storageClient.Upload(it.fileKey, it.data, "application/octet-stream"); err != nil {
				errOnce.Do(func() {
					slog.Error("解压文件上传失败", "file_key", it.fileKey, "error", err)
					uploadErr = hcommon.I18nRichError(err, i18n.MsgSkillFileUploadSMHFail, it.name, err)
					failed.Store(true)
				})
			}
		}(item)
	}
	wg.Wait()

	if uploadErr != nil {
		cleanupSkillPackage(ctx, cosZipKey, cosDirKey)
		return newSkillUploadError(http.StatusInternalServerError, uploadErr)
	}
	return nil
}

// cleanupSkillPackage 异步清理已上传到存储的技能包，避免上传中断留下孤儿文件。
// 存储客户端内部持有 ctx（查 token 走 DB、调 SMH API 也用它），handler 返回后
// r.Context() 会被 cancel，因此必须在 goroutine 内用 DetachContext 重新获取客户端。
func cleanupSkillPackage(ctx context.Context, cosZipKey, cosDirKey string) {
	go func(ctx context.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("清理技能包 panic", "key", cosZipKey, "recover", rec)
			}
		}()
		client, err := getStorageClient(ctx)
		if err != nil {
			slog.Error("清理技能包失败：获取存储客户端失败", "key", cosZipKey, "error", err)
			return
		}
		if err := client.Delete(cosZipKey, true); err != nil {
			slog.Error("清理 COS zip 失败", "key", cosZipKey, "error", err)
		}
		if err := client.DeletePrefix(cosDirKey, true); err != nil {
			slog.Error("清理 COS 目录失败", "prefix", cosDirKey, "error", err)
		}
	}(hcommon.DetachContext(ctx))
}

// maybeSubmitSkillSecurityScan 根据 submit_scan 参数决定是否异步触发安全检测。
func maybeSubmitSkillSecurityScan(r *http.Request, zipData []byte, skillID uint, version, zipFileName string) (submitted bool, skipReason string) {
	if r.FormValue("submit_scan") != "true" {
		return false, ""
	}
	if len(zipData) > maxScanFileSize {
		return false, "技能文件超过安全检测大小限制（7MB），跳过检测"
	}
	go func(ctx context.Context, data []byte, id uint, ver, name string) {
		if _, err := CreateSkillSecurityScan(ctx, data, id, ver, name); err != nil {
			slog.Warn("[SkillScan] 触发安全扫描失败", "skill_id", id, "error", err)
		}
	}(hcommon.DetachContext(r.Context()), zipData, skillID, version, zipFileName)
	return true, ""
}

// writeSkillUploadError 将 skillUploadError 写为 HTTP 错误响应。
func writeSkillUploadError(w http.ResponseWriter, r *http.Request, upErr *skillUploadError) {
	writeError(w, r, upErr.Status, hcommon.EnsureRichErrorOrPanic(upErr.Err))
}
