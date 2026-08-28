package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ErrPluginConflict 表示插件在插件包中已存在的冲突错误。
type ErrPluginConflict struct {
	Slug    string
	Version string
}

func (e *ErrPluginConflict) Error() string {
	return fmt.Sprintf("插件 %s-%s 已存在于该插件包中", e.Slug, e.Version)
}

// HandleCreatePluginBundle 创建插件包
func HandleCreatePluginBundle(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginBundleNameRequired))
		return
	}
	if len(name) > 100 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginBundleNameTooLong))
		return
	}

	var bundle model.PluginBundle
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var existing model.PluginBundle
		if tx.Where("name = ?", name).First(&existing).Error == nil {
			return fmt.Errorf("conflict")
		}

		bundle = model.PluginBundle{
			Name:        name,
			PluginCount: 0,
			Enabled:     false,
		}
		if err := tx.Create(&bundle).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCreatePluginBundleDBFailed)
		}

		commonClient, err := GetCommonStorageClient(r.Context())
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgGetCommonStorageClientFail)
		}
		dirKey := fmt.Sprintf("plugin-bundles/%s/.keep", name)
		if err := commonClient.Upload(dirKey, []byte{}, "application/octet-stream"); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSBCreateSMHDirFailed)
		}

		return nil
	})

	if err != nil {
		if rerr, ok := err.(*hcommon.RichError); ok {
			writeError(w, r, http.StatusInternalServerError, rerr)
		} else {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgPluginBundleConflict))
		}
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "id": bundle.ID})
}

// HandleAdminPluginBundles 查看插件包列表
func HandleAdminPluginBundles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	var total int64
	model.DB(r.Context()).Model(&model.PluginBundle{}).Count(&total)

	var bundles []model.PluginBundle
	model.DB(r.Context()).Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bundles)

	if bundles == nil {
		bundles = []model.PluginBundle{}
	}

	// 为 visibility_type='group' 的插件包附加绑定组信息
	type bundleWithGroups struct {
		model.PluginBundle
		VisibilityGroups []usergroup.VisibilityGroupRef `json:"visibility_groups,omitempty"`
	}

	// 批量获取绑定组
	groupBundleIDs := make([]uint, 0)
	for _, b := range bundles {
		if b.VisibilityType == usergroup.VisibilityGroup {
			groupBundleIDs = append(groupBundleIDs, b.ID)
		}
	}
	bindingMap := usergroup.GetVisibilityGroupRefs(r.Context(), model.ConfigTypePluginBundle, groupBundleIDs)

	enriched := make([]bundleWithGroups, 0, len(bundles))
	for _, b := range bundles {
		item := bundleWithGroups{PluginBundle: b}
		if b.VisibilityType == usergroup.VisibilityGroup {
			item.VisibilityGroups = bindingMap[b.ID]
		}
		enriched = append(enriched, item)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	jsonOK(w, map[string]interface{}{
		"plugin_bundles": enriched,
		"page":           page,
		"page_size":      pageSize,
		"total":          total,
		"total_pages":    totalPages,
	})
}

// HandleDeletePluginBundle 删除插件包
func HandleDeletePluginBundle(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var bundle model.PluginBundle
	if model.DB(r.Context()).First(&bundle, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginBundleNotFound))
		return
	}

	if bundle.Enabled {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgPluginBundleEnabledNeedsOff))
		return
	}

	// 删除 SMH 文件
	if commonClient, err := GetCommonStorageClient(r.Context()); err == nil {
		dirPrefix := fmt.Sprintf("plugin-bundles/%s/", bundle.Name)
		if err := commonClient.DeletePrefix(dirPrefix, true); err != nil {
			slog.Warn("清理插件包 SMH 文件失败", "prefix", dirPrefix, "error", err)
		}
	}

	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("plugin_bundle_id = ?", bundle.ID).Delete(&model.BundlePlugin{}).Error; err != nil {
			return err
		}
		return tx.Delete(&bundle).Error
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDeletePluginBundleFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleTogglePluginBundle 启用/禁用插件包
func HandleTogglePluginBundle(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var bundle model.PluginBundle
		if tx.First(&bundle, id).Error != nil {
			return fmt.Errorf("not_found")
		}

		if bundle.Enabled {
			return tx.Model(&bundle).Update("enabled", false).Error
		}

		var otherCount int64
		tx.Model(&model.PluginBundle{}).Where("enabled = ? AND id != ?", true, bundle.ID).Count(&otherCount)
		if otherCount > 0 {
			return fmt.Errorf("conflict")
		}
		return tx.Model(&bundle).Update("enabled", true).Error
	})

	if txErr != nil {
		switch txErr.Error() {
		case "not_found":
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginBundleNotFound))
		case "conflict":
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgOtherPluginBundleEnabled))
		default:
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgOperationFailed))
		}
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandlePluginBundleDetail 插件包详情
func HandlePluginBundleDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var bundle model.PluginBundle
	if model.DB(r.Context()).First(&bundle, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginBundleNotFound))
		return
	}

	var plugins []model.BundlePlugin
	model.DB(r.Context()).Where("plugin_bundle_id = ?", bundle.ID).Find(&plugins)
	if plugins == nil {
		plugins = []model.BundlePlugin{}
	}

	jsonOK(w, map[string]interface{}{
		"plugin_bundle": bundle,
		"plugins":       plugins,
	})
}

// HandleUpdatePluginBundlePlugins 批量更新插件包内插件
func HandleUpdatePluginBundlePlugins(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	bundleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || bundleID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var bundle model.PluginBundle
	if model.DB(r.Context()).First(&bundle, bundleID).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginBundleNotFound))
		return
	}

	var req struct {
		Add []struct {
			ID     uint   `json:"id"`
			Source string `json:"source"` // enterprise
		} `json:"add"`
		Remove []uint `json:"remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	commonClient, err := GetCommonStorageClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGetCommonStorageClientFail))
		return
	}

	// 处理 remove
	removedCount := 0
	if len(req.Remove) > 0 {
		var toRemove []model.BundlePlugin
		model.DB(r.Context()).Where("id IN ? AND plugin_bundle_id = ?", req.Remove, bundle.ID).Find(&toRemove)
		for _, p := range toRemove {
			if p.CosZipKey != "" {
				_ = commonClient.Delete(p.CosZipKey, true)
			}
		}
		removedCount = len(toRemove)
	}

	// 处理 add
	type addedPlugin struct {
		Name        string
		Slug        string
		PluginID    string
		Version     string
		Source      string
		CosZipKey   string
		NpmPackage  string
		InstallMode string
		Kind        string
	}
	var addedPlugins []addedPlugin

	for _, item := range req.Add {
		if item.Source == "" {
			item.Source = "enterprise"
		}

		if item.Source == "enterprise" {
			var p model.Plugin
			if model.DB(r.Context()).First(&p, item.ID).Error != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnterprisePluginNotFound, item.ID))
				return
			}

			// 从 skillhub space 下载 zip
			downloadURL, err := buildSMHDownloadURL(r.Context(), p.COSZipKey, false)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGenEnterprisePluginURLFail))
				return
			}
			resp, err := SkillHTTPClient.Get(downloadURL)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDownloadEnterprisePluginFail))
				return
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDownloadEnterprisePluginCode, resp.StatusCode))
				return
			}
			// 限制最大读取 200MB，防止 OOM
			const maxDownloadSize = 200 << 20
			var zipBuf bytes.Buffer
			if _, err := zipBuf.ReadFrom(io.LimitReader(resp.Body, maxDownloadSize+1)); err != nil {
				resp.Body.Close()
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgReadEnterprisePluginFail))
				return
			}
			resp.Body.Close() // 读取完成后立即关闭，避免 for 循环内 defer 泄漏
			if zipBuf.Len() > maxDownloadSize {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnterprisePluginTooLarge))
				return
			}

			cosZipKey := fmt.Sprintf("plugin-bundles/%s/%s/%s-%s.zip", bundle.Name, p.Slug, p.Slug, p.Version)
			if err := commonClient.Upload(cosZipKey, zipBuf.Bytes(), "application/zip"); err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUploadPluginZipFail))
				return
			}

			addedPlugins = append(addedPlugins, addedPlugin{
				Name:        p.Name,
				Slug:        p.Slug,
				PluginID:    p.PluginID,
				Version:     p.Version,
				Source:      "enterprise",
				CosZipKey:   cosZipKey,
				NpmPackage:  p.NpmPackage,
				InstallMode: "smh",
				Kind:        p.Kind,
			})
		} else {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUnsupportedSkillSource, item.Source))
			return
		}
	}

	// 事务内批量更新 DB
	addedCount := 0
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if len(req.Remove) > 0 {
			if err := tx.Where("id IN ? AND plugin_bundle_id = ?", req.Remove, bundle.ID).Delete(&model.BundlePlugin{}).Error; err != nil {
				return err
			}
		}

		for _, ap := range addedPlugins {
			var dupCount int64
			tx.Model(&model.BundlePlugin{}).Where(
				"plugin_bundle_id = ? AND slug = ? AND version = ? AND source = ?",
				bundle.ID, ap.Slug, ap.Version, ap.Source,
			).Count(&dupCount)
			if dupCount > 0 {
				return &ErrPluginConflict{Slug: ap.Slug, Version: ap.Version}
			}

			bp := model.BundlePlugin{
				PluginBundleID: uint(bundleID),
				Name:           ap.Name,
				Slug:           ap.Slug,
				PluginID:       ap.PluginID,
				Version:        ap.Version,
				Source:         ap.Source,
				CosZipKey:      ap.CosZipKey,
				NpmPackage:     ap.NpmPackage,
				InstallMode:    ap.InstallMode,
				Kind:           ap.Kind,
			}
			if err := tx.Create(&bp).Error; err != nil {
				return err
			}
			addedCount++
		}

		var count int64
		tx.Model(&model.BundlePlugin{}).Where("plugin_bundle_id = ?", bundle.ID).Count(&count)
		return tx.Model(&bundle).Update("plugin_count", int(count)).Error
	})

	if txErr != nil {
		var conflictErr *ErrPluginConflict
		if errors.As(txErr, &conflictErr) {
			re := hcommon.I18nError(i18n.MsgPluginVersionConflictInBundle, conflictErr.Slug, conflictErr.Version)
			re.WithCause(txErr)
			writeError(w, r, http.StatusConflict, re)
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgOperationFailed))
		return
	}

	var finalCount int64
	model.DB(r.Context()).Model(&model.BundlePlugin{}).Where("plugin_bundle_id = ?", bundle.ID).Count(&finalCount)

	jsonOK(w, map[string]interface{}{
		"ok":           true,
		"plugin_count": int(finalCount),
		"added":        addedCount,
		"removed":      removedCount,
	})
}
