package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// errPluginHasRunningTask 插件有进行中的下发任务时返回此错误
var errPluginHasRunningTask = errors.New("该版本有进行中的下发任务，无法删除")

// pluginMeta 从 zip 中提取的插件元数据
type pluginMeta struct {
	PluginID     string // openclaw.plugin.json 中的 id
	PluginFormat string // "openclaw" | "bundle"
	Kind         string // "memory" | "context-engine" | ""
	ConfigSchema string // JSON string
	Providers    string // JSON array string
	Channels     string // JSON array string
}

// validatePluginZip 校验插件 zip 包并提取元数据。
// 以 openclaw.plugin.json 或 Bundle 目录为锚点确定插件根目录。
// 输出的 zip 内文件统一包裹在 {slug}/ 目录下。
func validatePluginZip(zipData []byte, slug string) ([]string, []byte, *pluginMeta, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgZipParseFail, err)
	}
	if len(r.File) == 0 {
		return nil, nil, nil, hcommon.I18nError(i18n.MsgZipEmpty)
	}

	const maxUncompressedSize = 200 << 20 // 200MB

	// 第一遍扫描：找到锚点文件，确定插件格式和根目录
	var manifestPath string
	var bundleDir string
	var anchorPrefix string

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.ReplaceAll(f.Name, "\\", "/")
		parts := strings.Split(name, "/")
		fileName := parts[len(parts)-1]

		// 查找 openclaw.plugin.json（原生格式）
		if fileName == "openclaw.plugin.json" && manifestPath == "" {
			manifestPath = name
		}
	}

	// 查找 Bundle 目录
	if manifestPath == "" {
		for _, f := range r.File {
			name := strings.ReplaceAll(f.Name, "\\", "/")
			for _, dir := range []string{".codex-plugin/", ".claude-plugin/", ".cursor-plugin/"} {
				if strings.Contains(name, dir) && bundleDir == "" {
					// 提取 bundle 目录的父路径
					idx := strings.Index(name, dir)
					bundleDir = dir[:len(dir)-1] // 去掉尾部 /
					if idx > 0 {
						anchorPrefix = name[:idx]
					}
					break
				}
			}
			if bundleDir != "" {
				break
			}
		}
	}

	if manifestPath == "" && bundleDir == "" {
		return nil, nil, nil, hcommon.I18nError(i18n.MsgPluginZipNoManifestOrBundle)
	}

	meta := &pluginMeta{}

	if manifestPath != "" {
		// 原生格式：解析 openclaw.plugin.json
		meta.PluginFormat = "openclaw"
		if idx := strings.LastIndex(manifestPath, "/"); idx >= 0 {
			anchorPrefix = manifestPath[:idx+1]
		}

		// 读取 manifest
		for _, f := range r.File {
			if strings.ReplaceAll(f.Name, "\\", "/") == manifestPath {
				rc, err := f.Open()
				if err != nil {
					return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgPluginZipReadManifestFail)
				}
				// 限制 manifest 文件最大 1MB，防止恶意构造的巨大 manifest
				const maxManifestSize = 1 << 20
				var buf bytes.Buffer
				if _, err := buf.ReadFrom(io.LimitReader(rc, maxManifestSize+1)); err != nil {
					rc.Close()
					return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgPluginZipReadManifestFail)
				}
				rc.Close()
				if buf.Len() > maxManifestSize {
					return nil, nil, nil, hcommon.I18nError(i18n.MsgPluginZipManifestTooLarge)
				}

				var manifest map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &manifest); err != nil {
					return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgPluginZipParseManifestFail)
				}

				// 提取 id（必填）
				if id, ok := manifest["id"].(string); ok && id != "" {
					meta.PluginID = id
				} else {
					return nil, nil, nil, hcommon.I18nError(i18n.MsgPluginZipManifestMissingID)
				}

				// 提取 kind
				if kind, ok := manifest["kind"].(string); ok {
					if kind != "" && kind != "memory" && kind != "context-engine" {
						return nil, nil, nil, hcommon.I18nError(i18n.MsgPluginZipManifestInvalidKind)
					}
					meta.Kind = kind
				}

				// 提取 configSchema
				if cs, ok := manifest["configSchema"]; ok {
					csJSON, _ := json.Marshal(cs)
					meta.ConfigSchema = string(csJSON)
				}

				// 提取 providers
				if p, ok := manifest["providers"]; ok {
					pJSON, _ := json.Marshal(p)
					meta.Providers = string(pJSON)
				}

				// 提取 channels
				if ch, ok := manifest["channels"]; ok {
					chJSON, _ := json.Marshal(ch)
					meta.Channels = string(chJSON)
				}

				break
			}
		}
	} else {
		// Bundle 格式
		meta.PluginFormat = "bundle"
		meta.PluginID = slug // Bundle 格式使用 slug 作为 plugin id
	}

	// 第二遍扫描：提取锚点目录下的文件，重新打包
	var totalSize uint64
	var files []string
	var badFiles []string
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if anchorPrefix != "" {
			if !strings.HasPrefix(name, anchorPrefix) {
				continue
			}
		}
		// 跳过 __MACOSX、.git 等
		if strings.Contains(name, "__MACOSX") || strings.Contains(name, ".git/") {
			continue
		}
		if strings.Contains(name, "..") {
			return nil, nil, nil, hcommon.I18nError(i18n.MsgZipInvalidPath, name)
		}
		if badChar := findBadFileNameChar(name); badChar != "" {
			relName := strings.TrimPrefix(name, anchorPrefix)
			badFiles = append(badFiles, fmt.Sprintf("%s（含字符 '%s'）", relName, badChar))
			continue
		}
		totalSize += f.UncompressedSize64
		if totalSize > maxUncompressedSize {
			return nil, nil, nil, hcommon.I18nError(i18n.MsgZipTooLarge)
		}

		flatName := slug + "/" + strings.TrimPrefix(name, anchorPrefix)

		rc, err := f.Open()
		if err != nil {
			return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgZipReadEntryFail, err)
		}
		// 使用 LimitReader 防止 ZIP 炸弹：不信任 header 中声明的大小，限制实际读取字节数
		limitedReader := io.LimitReader(rc, int64(maxUncompressedSize-totalSize)+1)
		var fbuf bytes.Buffer
		n, _ := fbuf.ReadFrom(limitedReader)
		rc.Close()
		if uint64(n) > maxUncompressedSize-totalSize+uint64(f.UncompressedSize64) {
			return nil, nil, nil, hcommon.I18nError(i18n.MsgPluginZipBombDetected)
		}

		newHeader := f.FileHeader
		newHeader.Name = flatName
		fw, err := writer.CreateHeader(&newHeader)
		if err != nil {
			return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgZipRepackFail, err)
		}
		fw.Write(fbuf.Bytes())

		files = append(files, flatName)
	}

	if err := writer.Close(); err != nil {
		return nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgZipFinishFail, err)
	}

	if len(badFiles) > 0 {
		list := strings.Join(badFiles, "、")
		if len(badFiles) > 5 {
			list = strings.Join(badFiles[:5], "、") + fmt.Sprintf(" 等共 %d 个文件", len(badFiles))
		}
		return nil, nil, nil, hcommon.I18nError(i18n.MsgZipBadFileName, list)
	}

	if len(files) == 0 {
		return nil, nil, nil, hcommon.I18nError(i18n.MsgZipNoValidFile)
	}

	return files, buf.Bytes(), meta, nil
}

// pluginUploadResult 插件 zip 上传到 SMH 的结果
type pluginUploadResult struct {
	COSZipKey    string
	COSDirKey    string
	FileList     string
	FileSize     int64
	PluginID     string
	PluginFormat string
	Kind         string
	ConfigSchema string
	Providers    string
	Channels     string
}

// uploadPluginZipToSMH 上传插件 zip 到 SMH，返回 COS keys、文件列表和元数据。
// 该函数独立于 DB 事务，上传成功后调用者负责持久化到 DB。
// 失败时会自动清理已上传的文件。
func uploadPluginZipToSMH(ctx context.Context, zipData []byte, slug, version string) (*pluginUploadResult, error) {
	const maxUploadSize = 200 << 20 // 200MB

	// 校验 zip 并提取元数据
	files, normalizedZipData, meta, err := validatePluginZip(zipData, slug)
	if err != nil {
		return nil, err
	}
	zipData = normalizedZipData

	cosZipKey := fmt.Sprintf("plugins/%s/%s-%s.zip", slug, slug, version)
	cosDirKey := fmt.Sprintf("plugins/%s/%s-%s/", slug, slug, version)

	slugPrefix := slug + "/"
	for i, f := range files {
		files[i] = cosDirKey + strings.TrimPrefix(f, slugPrefix)
	}
	fileListJSON, _ := json.Marshal(files)
	if isFileListTooLarge(fileListJSON) {
		slog.Warn("插件文件列表过长", "slug", slug, "version", version, "file_list_len", len(fileListJSON), "file_count", len(files))
		return nil, hcommon.I18nError(i18n.MsgPluginFileListTooLarge)
	}

	storageClient, storageErr := getStorageClient(ctx)
	if storageErr != nil {
		return nil, hcommon.I18nRichError(storageErr, i18n.MsgPluginSMHUnavailable, storageErr)
	}

	// 上传 zip
	if err := storageClient.Upload(cosZipKey, zipData, "application/zip"); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgPluginUploadZipFail, err)
	}

	// 解压后并发上传文件到 COS
	if zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData))); err == nil {
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
				continue
			}
			var fbuf bytes.Buffer
			fbuf.ReadFrom(io.LimitReader(rc, maxUploadSize))
			rc.Close()
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
						uploadErr = hcommon.I18nRichError(err, i18n.MsgSkillFileUploadSMHFail, it.name, err)
						failed.Store(true)
					})
				}
			}(item)
		}
		wg.Wait()

		if uploadErr != nil {
			// 清理已上传的文件
			go func() {
				_ = storageClient.Delete(cosZipKey, true)
				_ = storageClient.DeletePrefix(cosDirKey, true)
			}()
			return nil, uploadErr
		}
	}

	slog.Info("插件 zip 上传完成", "slug", slug, "version", version, "file_count", len(files))

	return &pluginUploadResult{
		COSZipKey:    cosZipKey,
		COSDirKey:    cosDirKey,
		FileList:     string(fileListJSON),
		FileSize:     int64(len(zipData)),
		PluginID:     meta.PluginID,
		PluginFormat: meta.PluginFormat,
		Kind:         meta.Kind,
		ConfigSchema: meta.ConfigSchema,
		Providers:    meta.Providers,
		Channels:     meta.Channels,
	}, nil
}

// HandleAdminPlugins 查询插件列表（分页，每个 slug 只返回最新版本）
// pluginVisGroupInfo 插件可见性分组信息
type pluginVisGroupInfo struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// buildPluginVisibilityData 批量构建插件列表的可见性分组数据。
// 参数：plugins - 待处理的插件列表
// 返回：map[插件ID] => []分组信息
func buildPluginVisibilityData(ctx context.Context, plugins []model.Plugin) map[uint][]pluginVisGroupInfo {
	result := make(map[uint][]pluginVisGroupInfo)

	// 筛出 visibility_type="group" 的插件 ID
	var groupPluginIDs []uint
	for _, p := range plugins {
		if p.VisibilityType == model.VisibilityGroup {
			groupPluginIDs = append(groupPluginIDs, p.ID)
		}
	}
	if len(groupPluginIDs) == 0 {
		return result
	}

	// 批量查出所有关联
	groupMap, rerr := model.GetPluginVisibilityGroupIDs(ctx, groupPluginIDs)
	if rerr != nil {
		slog.Error("[PluginVisibility] 批量查询插件分组关联失败", "error", rerr)
		return result
	}

	// 收集去重的 group_id
	groupIDSet := make(map[uint]bool)
	for _, gids := range groupMap {
		for _, gid := range gids {
			groupIDSet[gid] = true
		}
	}
	if len(groupIDSet) == 0 {
		return result
	}
	allGroupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		allGroupIDs = append(allGroupIDs, gid)
	}

	// 批量查分组名称
	groups, err := model.GetGroupsByIDs(ctx, allGroupIDs)
	if err != nil {
		slog.Error("[PluginVisibility] 批量查询分组名称失败", "error", err)
		return result
	}
	groupNameMap := make(map[uint]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	// 组装结果
	for pluginID, gids := range groupMap {
		for _, gid := range gids {
			name, ok := groupNameMap[gid]
			if !ok {
				slog.Warn("[PluginVisibility] 分组已不存在，跳过", "group_id", gid, "plugin_id", pluginID)
				continue
			}
			result[pluginID] = append(result[pluginID], pluginVisGroupInfo{GroupID: gid, GroupName: name})
		}
	}
	return result
}

// pluginSlugStats 插件 slug 级别的统计信息
type pluginSlugStats struct {
	InstalledCount  int64
	HasRunningTask  bool
	DistributeCount int
}

// batchPluginSlugStats 批量查询插件的 installed_count 和 has_running_task。
// 两项统计都按 slug 维度计算，因此合并查询减少重复。
// 参数：plugins - 待处理的插件列表
// 返回：map[插件ID] => 统计信息
func batchPluginSlugStats(ctx context.Context, plugins []model.Plugin) map[uint]pluginSlugStats {
	result := make(map[uint]pluginSlugStats)
	if len(plugins) == 0 {
		return result
	}

	// 按 slug 分组
	slugMap := make(map[string][]uint)
	for _, p := range plugins {
		slugMap[p.Slug] = append(slugMap[p.Slug], p.ID)
	}

	// 每个 slug 只查一次 allPluginIDs，同时查 installed_count 和 running_task_count
	for slug, pids := range slugMap {
		var allPluginIDs []uint
		model.DB(ctx).Model(&model.Plugin{}).Where("slug = ?", slug).
			Pluck("id", &allPluginIDs)

		// 查询 installed_count（success records 按 instance_id 去重）
		var installedCount int64
		model.DB(ctx).Model(&model.PluginDistributionRecord{}).
			Where("plugin_db_id IN ? AND status = ? AND type = ?",
				allPluginIDs, "success", "distribute").
			Distinct("instance_id").Count(&installedCount)

		// 查询 has_running_task
		var runningCount int64
		model.DB(ctx).Model(&model.PluginDistributionTask{}).
			Where("plugin_db_id IN ? AND status = ?", allPluginIDs, "running").
			Count(&runningCount)

		// 查询 distribute_count（同 slug 所有版本的最大值）
		var maxDistCount int
		model.DB(ctx).Model(&model.Plugin{}).Where("slug = ?", slug).
			Select("COALESCE(MAX(distribute_count), 0)").Scan(&maxDistCount)

		// 将结果写入所有同 slug 的 plugin ID
		stats := pluginSlugStats{
			InstalledCount:  installedCount,
			HasRunningTask:  runningCount > 0,
			DistributeCount: maxDistCount,
		}
		for _, pid := range pids {
			result[pid] = stats
		}
	}

	return result
}

// applyPluginVisibility 应用插件可见性设置。
// 参数：
//   - tx: GORM 事务对象
//   - pluginID: 待设置可见性的插件 ID
//   - slug: 插件 slug（用于 inheritFromPrev 时查找前版本）
//   - visType: 可见性类型（"all" 或 "group"）
//   - groupIDsCSV: 分组 ID 列表（逗号分隔字符串，仅当 visType="group" 时有效）
//   - inheritFromPrev: 是否从前版本继承（用于 HandleUpdatePlugin 且未传 visibility_type 时）
//
// 返回：error（失败时返回错误，否则返回 nil）
func applyPluginVisibility(tx *gorm.DB, pluginID uint, slug, visType, groupIDsCSV string, inheritFromPrev bool) error {
	if inheritFromPrev {
		// 继承旧版本的可见性设置
		return model.CopyPluginVisibility(tx, slug, pluginID)
	}

	if visType != "all" && visType != "group" {
		return hcommon.I18nError(i18n.MsgInvalidVisibilityForModel)
	}

	if visType == "group" {
		groupIDs, _ := parseUintCSV(groupIDsCSV)
		if len(groupIDs) == 0 {
			return hcommon.I18nError(i18n.MsgPluginVisibilityGroupIDsRequired)
		}
		return model.SetPluginVisibility(tx, pluginID, visType, groupIDs)
	}

	// visType == "all"
	if err := tx.Model(&model.Plugin{}).Where("id = ?", pluginID).Update("visibility_type", "all").Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginSetVisibilityFail)
	}
	return nil
}

func HandleAdminPlugins(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)

	db := model.DB(r.Context()).Model(&model.Plugin{}).Where("id IN (?)", model.LatestVersionPluginIDs(r.Context()))

	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if catIDs := r.URL.Query().Get("category_ids"); catIDs != "" {
		ids := strings.Split(catIDs, ",")
		var intIDs []int
		for _, s := range ids {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				intIDs = append(intIDs, id)
			}
		}
		if len(intIDs) > 0 {
			subQuery := model.DB(r.Context()).Model(&model.PluginCategoryMapping{}).
				Select("plugin_id").
				Where("category_id IN ?", intIDs)
			db = db.Where("id IN (?)", subQuery)
		}
	}

	var total int64
	db.Count(&total)

	var plugins []model.Plugin
	db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&plugins)

	// 批量查询分类关联 + 最近一次下发任务
	type lastTask struct {
		TaskID    uint      `json:"task_id"`
		Status    string    `json:"status"`
		Total     int       `json:"total"`
		Success   int       `json:"success"`
		Failed    int       `json:"failed"`
		Version   string    `json:"version"`
		Type      string    `json:"type"`
		CreatedAt time.Time `json:"created_at"`
	}
	type pluginResp struct {
		model.Plugin
		Categories       []map[string]interface{} `json:"categories"`
		LastTask         *lastTask                `json:"last_task"`
		VisibilityGroups []pluginVisGroupInfo     `json:"visibility_groups"`
		InstalledCount   int64                    `json:"installed_count"`
		HasRunningTask   bool                     `json:"has_running_task"`
	}

	taskMap := make(map[uint]*lastTask)
	if len(plugins) > 0 {
		// 按 slug 查询所有版本的 plugin_db_id，确保能找到旧版本的下发任务
		allPluginIDsBySlug := make(map[string][]uint)
		for _, p := range plugins {
			if _, ok := allPluginIDsBySlug[p.Slug]; !ok {
				var ids []uint
				model.DB(r.Context()).Model(&model.Plugin{}).Where("slug = ?", p.Slug).Pluck("id", &ids)
				allPluginIDsBySlug[p.Slug] = ids
			}
		}

		// 收集所有 plugin IDs
		var allPluginIDs []uint
		seen := make(map[uint]bool)
		for _, ids := range allPluginIDsBySlug {
			for _, id := range ids {
				if !seen[id] {
					allPluginIDs = append(allPluginIDs, id)
					seen[id] = true
				}
			}
		}

		var tasks []model.PluginDistributionTask
		subQuery := model.DB(r.Context()).Model(&model.PluginDistributionTask{}).
			Select("MAX(id)").
			Where("plugin_db_id IN ?", allPluginIDs).
			Group("plugin_db_id")
		model.DB(r.Context()).Where("id IN (?)", subQuery).Find(&tasks)

		if len(tasks) > 0 {
			taskIDs := make([]uint, len(tasks))
			for i, t := range tasks {
				taskIDs[i] = t.ID
			}
			type taskStatusCount struct {
				TaskID uint
				Status string
				Count  int
			}
			var counts []taskStatusCount
			if err := model.DB(r.Context()).Model(&model.PluginDistributionRecord{}).
				Select("task_id, status, COUNT(*) as count").
				Where("task_id IN ?", taskIDs).
				Group("task_id, status").
				Scan(&counts).Error; err != nil {
				slog.Error("查询插件下发记录聚合失败", "error", err)
			}

			type counters struct{ Success, Failed int }
			countMap := make(map[uint]*counters)
			for _, c := range counts {
				if countMap[c.TaskID] == nil {
					countMap[c.TaskID] = &counters{}
				}
				switch c.Status {
				case "success":
					countMap[c.TaskID].Success = c.Count
				case "failed":
					countMap[c.TaskID].Failed = c.Count
				}
			}

			for _, t := range tasks {
				lt := &lastTask{
					TaskID:    t.ID,
					Status:    t.Status,
					Total:     t.Total,
					Version:   t.Version,
					CreatedAt: t.CreatedAt,
					Type:      t.Type,
				}
				if c := countMap[t.ID]; c != nil {
					lt.Success = c.Success
					lt.Failed = c.Failed
				}
				// 将任务映射到当前最新版本的 plugin ID
				// 找到 t.PluginDBID 属于哪个 slug，然后映射到 plugins 列表中的对应项
				for _, p := range plugins {
					ids := allPluginIDsBySlug[p.Slug]
					for _, id := range ids {
						if id == t.PluginDBID {
							// 只保留最新的任务（ID 最大）
							if existing, ok := taskMap[p.ID]; !ok || t.ID > existing.TaskID {
								taskMap[p.ID] = lt
							}
							break
						}
					}
				}
			}
		}
	}

	// 批量查询可见性分组
	visibilityMap := buildPluginVisibilityData(r.Context(), plugins)

	// 批量查询 installed_count 和 has_running_task（合并查询消除重复）
	statsMap := batchPluginSlugStats(r.Context(), plugins)

	// 批量查询分类
	categoryMap := make(map[uint][]map[string]interface{})
	if len(plugins) > 0 {
		pluginIDs := make([]uint, len(plugins))
		for i, p := range plugins {
			pluginIDs[i] = p.ID
		}
		var allMappings []model.PluginCategoryMapping
		model.DB(r.Context()).Where("plugin_id IN ?", pluginIDs).Find(&allMappings)

		catIDSet := make(map[uint]struct{})
		for _, m := range allMappings {
			catIDSet[m.CategoryID] = struct{}{}
		}
		catIDs := make([]uint, 0, len(catIDSet))
		for id := range catIDSet {
			catIDs = append(catIDs, id)
		}

		catMap := make(map[uint]model.PluginCategory)
		if len(catIDs) > 0 {
			var cats []model.PluginCategory
			model.DB(r.Context()).Where("id IN ?", catIDs).Find(&cats)
			for _, c := range cats {
				catMap[c.ID] = c
			}
		}

		for _, m := range allMappings {
			if cat, ok := catMap[m.CategoryID]; ok {
				categoryMap[m.PluginID] = append(categoryMap[m.PluginID], map[string]interface{}{"id": cat.ID, "name": cat.Name})
			}
		}
	}

	var result []pluginResp
	for _, p := range plugins {
		pr := pluginResp{
			Plugin:           p,
			LastTask:         taskMap[p.ID],
			VisibilityGroups: visibilityMap[p.ID],
			InstalledCount:   statsMap[p.ID].InstalledCount,
			HasRunningTask:   statsMap[p.ID].HasRunningTask,
		}
		// 用 slug 维度的最大 distribute_count 覆盖（旧版本下发的计数可能未继承到新版本）
		if stats := statsMap[p.ID]; stats.DistributeCount > pr.Plugin.DistributeCount {
			pr.Plugin.DistributeCount = stats.DistributeCount
		}
		pr.Categories = categoryMap[p.ID]
		if pr.Categories == nil {
			pr.Categories = []map[string]interface{}{}
		}
		if pr.VisibilityGroups == nil {
			pr.VisibilityGroups = []pluginVisGroupInfo{}
		}
		result = append(result, pr)
	}

	jsonOK(w, map[string]interface{}{
		"plugins":   result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// HandleCreatePlugin 创建插件
func HandleCreatePlugin(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	const maxUploadSize = 200 << 20 // 200MB
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgRequestBodyTooLargeWithError, err))
		return
	}

	slug := r.FormValue("slug")
	name := r.FormValue("name")
	version := r.FormValue("version")
	slog.Info("开始创建插件", "slug", slug, "name", name, "version", version)

	if slug == "" || name == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginSlugNameVerRequired))
		return
	}
	if !isValidSlug(slug) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginInvalidSlug))
		return
	}

	plugin := model.Plugin{
		Slug:        slug,
		Name:        name,
		Description: r.FormValue("description"),
		Version:     version,
		NpmPackage:  r.FormValue("npm_package"),
		Changelog:   r.FormValue("changelog"),
	}
	if err := plugin.ParseVersion(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 检查唯一性
	var existing model.Plugin
	err := model.DB(r.Context()).Unscoped().
		Where("slug = ? AND version = ?", slug, version).
		First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			model.DB(r.Context()).Unscoped().Delete(&model.Plugin{}, existing.ID)
			model.DB(r.Context()).Where("plugin_id = ?", existing.ID).Delete(&model.PluginCategoryMapping{})
		} else {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginVersionExist, slug, version))
			return
		}
	}

	// 读取 zip 文件
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginFileFieldMissing))
		return
	}
	defer file.Close()

	if header.Size > maxUploadSize {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginFileSizeTooLarge))
		return
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(io.LimitReader(file, maxUploadSize)); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginReadUploadFail, err))
		return
	}
	zipData := buf.Bytes()

	// 上传到 SMH
	uploadResult, err := uploadPluginZipToSMH(r.Context(), zipData, slug, version)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 填充元数据
	plugin.PluginID = uploadResult.PluginID
	plugin.PluginFormat = uploadResult.PluginFormat
	plugin.Kind = uploadResult.Kind
	plugin.ConfigSchema = uploadResult.ConfigSchema
	plugin.Providers = uploadResult.Providers
	plugin.Channels = uploadResult.Channels
	plugin.COSZipKey = uploadResult.COSZipKey
	plugin.COSDirKey = uploadResult.COSDirKey
	plugin.FileList = uploadResult.FileList
	plugin.FileSize = uploadResult.FileSize

	// 写 DB（文件已上传）
	tx := model.DB(r.Context()).Begin()

	if err := tx.Create(&plugin).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginVersionExist, slug, version))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginCreateRecordFail, err))
		return
	}

	// 处理分类关联
	if catIDs := r.FormValue("category_ids"); catIDs != "" {
		for _, s := range strings.Split(catIDs, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
				tx.Create(&model.PluginCategoryMapping{PluginID: plugin.ID, CategoryID: uint(id)})
			}
		}
	}

	// 处理可见性
	visType := r.FormValue("visibility_type")
	if visType == "" {
		visType = "all"
	}
	if err := applyPluginVisibility(tx, plugin.ID, slug, visType, r.FormValue("group_ids"), false); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	tx.Commit()
	slog.Info("插件创建成功", "slug", slug, "version", version, "id", plugin.ID, "plugin_id", plugin.PluginID)

	jsonOK(w, map[string]interface{}{"ok": true, "id": plugin.ID, "slug": slug, "version": version, "plugin_id": plugin.PluginID})
}

// HandleUpdatePlugin 编辑插件元信息
func HandleUpdatePlugin(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	const maxUploadSize = 200 << 20 // 200MB
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgRequestBodyTooLargeWithError, err))
		return
	}

	slug := r.FormValue("slug")
	version := r.FormValue("version")
	if slug == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginSlugVersionRequired))
		return
	}

	// 查找当前最新版本
	var currentLatest model.Plugin
	if err := model.DB(r.Context()).Where("slug = ?", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&currentLatest).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	// 解析新版本号
	var newPlugin model.Plugin
	newPlugin.Version = version
	if err := newPlugin.ParseVersion(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 判断是原地更新元信息（version <= 当前最新版本）还是版本升级（version > 当前版本）
	newVer := newPlugin.VersionMajor*1000000 + newPlugin.VersionMinor*1000 +
		newPlugin.VersionPatch
	currentVer := currentLatest.VersionMajor*1000000 +
		currentLatest.VersionMinor*1000 + currentLatest.VersionPatch

	if newVer <= currentVer {
		// 原地更新元信息（兼容 API 用户原有行为：指定 slug+version 更新元数据）
		var targetPlugin model.Plugin
		if err := model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).
			First(&targetPlugin).Error; err != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginVersionNotExist))
			return
		}
		handleUpdatePluginMetadata(w, r, &targetPlugin)
		return
	}

	// 以下为版本升级逻辑（newVer > currentVer）

	// 检查版本唯一性
	var existing model.Plugin
	err := model.DB(r.Context()).Unscoped().
		Where("slug = ? AND version = ?", slug, version).
		First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			model.DB(r.Context()).Unscoped().Delete(&model.Plugin{}, existing.ID)
			model.DB(r.Context()).Where("plugin_id = ?", existing.ID).
				Delete(&model.PluginCategoryMapping{})
		} else {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgPluginVersionExist, slug, version))
			return
		}
	}

	// 初始化新插件记录（继承大部分字段）
	newPlugin.Slug = slug
	newPlugin.PluginID = currentLatest.PluginID
	newPlugin.PluginFormat = currentLatest.PluginFormat
	newPlugin.Kind = currentLatest.Kind
	newPlugin.ConfigSchema = currentLatest.ConfigSchema
	newPlugin.Providers = currentLatest.Providers
	newPlugin.Channels = currentLatest.Channels
	newPlugin.Name = currentLatest.Name
	newPlugin.Description = currentLatest.Description
	newPlugin.NpmPackage = currentLatest.NpmPackage

	// 处理可选的元信息覆写
	if name := r.FormValue("name"); name != "" {
		newPlugin.Name = name
	}
	if desc := r.FormValue("description"); desc != "" {
		newPlugin.Description = desc
	}
	if npm := r.FormValue("npm_package"); npm != "" {
		newPlugin.NpmPackage = npm
	}
	newPlugin.Changelog = r.FormValue("changelog")

	// 处理 zip 文件（可选）
	file, header, err := r.FormFile("file")
	hasFile := err == nil
	if hasFile {
		defer file.Close()
		if header.Size > maxUploadSize {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginFileSizeTooLarge))
			return
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(io.LimitReader(file, maxUploadSize)); err != nil {
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgPluginReadUploadFail, err))
			return
		}
		zipData := buf.Bytes()

		// 上传到 SMH
		uploadResult, err := uploadPluginZipToSMH(r.Context(), zipData, slug, version)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}

		// 更新元数据
		newPlugin.PluginID = uploadResult.PluginID
		newPlugin.PluginFormat = uploadResult.PluginFormat
		newPlugin.Kind = uploadResult.Kind
		newPlugin.ConfigSchema = uploadResult.ConfigSchema
		newPlugin.Providers = uploadResult.Providers
		newPlugin.Channels = uploadResult.Channels
		newPlugin.COSZipKey = uploadResult.COSZipKey
		newPlugin.COSDirKey = uploadResult.COSDirKey
		newPlugin.FileList = uploadResult.FileList
		newPlugin.FileSize = uploadResult.FileSize
	} else {
		// 无文件时，继承当前版本的 COS 信息
		newPlugin.COSZipKey = currentLatest.COSZipKey
		newPlugin.COSDirKey = currentLatest.COSDirKey
		newPlugin.FileList = currentLatest.FileList
		newPlugin.FileSize = currentLatest.FileSize
	}

	// 事务：创建新版本 + 可见性处理
	// 继承 distribute_count：直接赋值到 newPlugin，随 Create 一起写入，避免额外 UPDATE
	newPlugin.DistributeCount = currentLatest.DistributeCount

	// 手动设置时间为 UTC，避免 GORM 使用本地时区（UTC+8）填充
	now := time.Now().UTC()
	newPlugin.CreatedAt = now
	newPlugin.UpdatedAt = now

	tx := model.DB(r.Context()).Begin()
	if err := tx.Create(&newPlugin).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgPluginVersionExist, slug, version))
			return
		}
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgPluginCreateRecordFail, err))
		return
	}

	// 处理可见性
	visType := r.FormValue("visibility_type")
	inheritFromPrev := visType == ""
	if inheritFromPrev {
		visType = "all" // 默认值，实际会被 applyPluginVisibility 忽略
	}
	if err := applyPluginVisibility(tx, newPlugin.ID, slug, visType, r.FormValue("group_ids"), inheritFromPrev); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 处理分类关联
	if catIDs := r.FormValue("category_ids"); catIDs != "" {
		for _, s := range strings.Split(catIDs, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
				tx.Create(&model.PluginCategoryMapping{
					PluginID:   newPlugin.ID,
					CategoryID: uint(id),
				})
			}
		}
	} else {
		// 未传分类，继承旧版本
		var oldMappings []model.PluginCategoryMapping
		tx.Where("plugin_id = ?", currentLatest.ID).Find(&oldMappings)
		for _, m := range oldMappings {
			tx.Create(&model.PluginCategoryMapping{
				PluginID:   newPlugin.ID,
				CategoryID: m.CategoryID,
			})
		}
	}

	tx.Commit()
	slog.Info("插件版本更新成功", "slug", slug, "old_version",
		currentLatest.Version, "new_version", version, "id", newPlugin.ID)

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"id":      newPlugin.ID,
		"slug":    slug,
		"version": version,
	})
}

// HandleDeletePlugin 删除指定 slug+version
func HandleDeletePlugin(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.FormValue("slug")
	version := r.FormValue("version")
	if slug == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginSlugVersionRequired))
		return
	}

	var plugin model.Plugin
	if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&plugin).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var runningCount int64
		if err := tx.Model(&model.PluginDistributionTask{}).
			Where("plugin_db_id = ? AND status = ?", plugin.ID, "running").
			Count(&runningCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillCheckTaskFail, err)
		}
		if runningCount > 0 {
			return errPluginHasRunningTask
		}
		tx.Where("plugin_id = ?", plugin.ID).Delete(&model.PluginCategoryMapping{})
		if err := tx.Delete(&plugin).Error; err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errPluginHasRunningTask) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginVersionInUse))
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgPluginDeleteFailed, txErr))
		}
		return
	}

	// 清理 COS 文件
	cosDirPrefix := fmt.Sprintf("plugins/%s/%s-%s/", slug, slug, version)
	if err := deleteCOSPrefix(r.Context(), cosDirPrefix); err != nil {
		slog.Warn("COS cleanup failed (dir)", "prefix", cosDirPrefix, "error", err)
	}
	cosZipKey := fmt.Sprintf("plugins/%s/%s-%s.zip", slug, slug, version)
	if client, err := getStorageClient(r.Context()); err == nil {
		if err := client.Delete(cosZipKey, true); err != nil {
			slog.Warn("COS cleanup failed (zip)", "key", cosZipKey, "error", err)
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminPluginDetail 查询插件详情
func HandleAdminPluginDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	version := r.URL.Query().Get("version")
	var plugin model.Plugin
	if version == "" || version == "latest" {
		if model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&plugin).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&plugin).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, slug, version))
			return
		}
	}

	var allVersions []model.Plugin
	model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&allVersions)
	var versions []string
	for _, v := range allVersions {
		versions = append(versions, v.Version)
	}

	var categories []map[string]interface{}
	var mappings []model.PluginCategoryMapping
	model.DB(r.Context()).Where("plugin_id = ?", plugin.ID).Find(&mappings)
	if len(mappings) > 0 {
		// 批量查询分类，避免 N+1 问题
		catIDs := make([]uint, len(mappings))
		for i, m := range mappings {
			catIDs[i] = m.CategoryID
		}
		var cats []model.PluginCategory
		model.DB(r.Context()).Where("id IN ?", catIDs).Find(&cats)
		for _, cat := range cats {
			categories = append(categories, map[string]interface{}{"id": cat.ID, "name": cat.Name})
		}
	}
	if categories == nil {
		categories = []map[string]interface{}{}
	}

	// 查询可见性分组
	visibilityMap := buildPluginVisibilityData(r.Context(), []model.Plugin{plugin})
	var visGroups []map[string]interface{}
	for _, vg := range visibilityMap[plugin.ID] {
		visGroups = append(visGroups, map[string]interface{}{
			"group_id":   vg.GroupID,
			"group_name": vg.GroupName,
		})
	}
	if visGroups == nil {
		visGroups = []map[string]interface{}{}
	}

	// 查询 installed_count 和 has_running_task
	statsMap := batchPluginSlugStats(r.Context(), []model.Plugin{plugin})
	stats := statsMap[plugin.ID]

	jsonOK(w, map[string]interface{}{
		"plugin": map[string]interface{}{
			"id":                plugin.ID,
			"slug":              plugin.Slug,
			"name":              plugin.Name,
			"version":           plugin.Version,
			"description":       plugin.Description,
			"plugin_id":         plugin.PluginID,
			"plugin_format":     plugin.PluginFormat,
			"kind":              plugin.Kind,
			"npm_package":       plugin.NpmPackage,
			"config_schema":     plugin.ConfigSchema,
			"providers":         plugin.Providers,
			"channels":          plugin.Channels,
			"categories":        categories,
			"file_size":         plugin.FileSize,
			"cos_zip_key":       plugin.COSZipKey,
			"cos_dir_key":       plugin.COSDirKey,
			"created_at":        plugin.CreatedAt,
			"updated_at":        plugin.UpdatedAt,
			"changelog":         plugin.Changelog,
			"visibility_type":   plugin.VisibilityType,
			"visibility_groups": visGroups,
			"distribute_count":  plugin.DistributeCount,
			"installed_count":   stats.InstalledCount,
			"has_running_task":  stats.HasRunningTask,
		},
		"versions": versions,
	})
}

// HandleAdminPluginFiles 查询插件文件列表
func HandleAdminPluginFiles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	var plugins []model.Plugin
	model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&plugins)
	if len(plugins) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	type versionFiles struct {
		Version   string   `json:"version"`
		Files     []string `json:"files"`
		Changelog string   `json:"changelog"`
		CreatedAt string   `json:"created_at"`
	}
	var result []versionFiles
	for _, p := range plugins {
		var files []string
		if p.FileList != "" {
			json.Unmarshal([]byte(p.FileList), &files)
		}
		if files == nil {
			files = []string{}
		}
		result = append(result, versionFiles{
			Version:   p.Version,
			Files:     files,
			Changelog: p.Changelog,
			CreatedAt: p.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	jsonOK(w, map[string]interface{}{
		"slug":     slug,
		"versions": result,
	})
}

// HandleAdminPluginTasks 查询下发任务列表
func HandleAdminPluginTasks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 校验 type 参数（可选）
	taskType := r.URL.Query().Get("type")
	if taskType != "" {
		// type 必填，只接受 distribute / uninstall
		if taskType != "distribute" && taskType != "uninstall" {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgPluginInvalidTypeParam))
			return
		}
	}

	var pluginIDs []uint
	model.DB(r.Context()).Model(&model.Plugin{}).Where("slug = ?", slug).Pluck("id", &pluginIDs)
	if len(pluginIDs) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	page, pageSize := parsePagination(r)
	var total int64
	query := model.DB(r.Context()).Model(&model.PluginDistributionTask{}).
		Where("plugin_db_id IN ?", pluginIDs)
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	query.Count(&total)

	var tasks []model.PluginDistributionTask
	query = model.DB(r.Context()).Where("plugin_db_id IN ?", pluginIDs)
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	type recordResp struct {
		InstanceID    uint   `json:"instance_id"`
		CVMInstanceID string `json:"cvm_instance_id"`
		InstanceName  string `json:"instance_name"`
		Username      string `json:"username"`
		Status        string `json:"status"`
		Error         string `json:"error"`
	}
	type taskResp struct {
		ID        uint         `json:"id"`
		CreatedAt interface{}  `json:"created_at"`
		Operator  string       `json:"operator"`
		Version   string       `json:"version"`
		Total     int          `json:"total"`
		Success   int          `json:"success"`
		Failed    int          `json:"failed"`
		Pending   int          `json:"pending"`
		Status    string       `json:"status"`
		Type      string       `json:"type"`
		Records   []recordResp `json:"records"`
	}

	var result []taskResp
	if len(tasks) > 0 {
		taskIDs := make([]uint, len(tasks))
		operatorIDs := make(map[uint]struct{})
		for i, t := range tasks {
			taskIDs[i] = t.ID
			if t.OperatorID > 0 {
				operatorIDs[t.OperatorID] = struct{}{}
			}
		}

		type taskStatusCount struct {
			TaskID uint
			Status string
			Count  int
		}
		var allCounts []taskStatusCount
		model.DB(r.Context()).Model(&model.PluginDistributionRecord{}).
			Select("task_id, status, COUNT(*) as count").
			Where("task_id IN ?", taskIDs).
			Group("task_id, status").
			Scan(&allCounts)
		type counters struct{ Success, Failed, Pending int }
		countMap := make(map[uint]*counters)
		for _, c := range allCounts {
			if countMap[c.TaskID] == nil {
				countMap[c.TaskID] = &counters{}
			}
			switch c.Status {
			case "success":
				countMap[c.TaskID].Success = c.Count
			case "failed":
				countMap[c.TaskID].Failed = c.Count
			case "pending":
				countMap[c.TaskID].Pending = c.Count
			}
		}

		opIDs := make([]uint, 0, len(operatorIDs))
		for id := range operatorIDs {
			opIDs = append(opIDs, id)
		}
		userMap := make(map[uint]string)
		if len(opIDs) > 0 {
			var users []model.User
			model.DB(r.Context()).Where("id IN ?", opIDs).Find(&users)
			for _, u := range users {
				userMap[u.ID] = u.Username
			}
		}

		var allRecords []model.PluginDistributionRecord
		model.DB(r.Context()).Where("task_id IN ?", taskIDs).Find(&allRecords)

		instIDSet := make(map[uint]struct{})
		for _, rec := range allRecords {
			instIDSet[rec.InstanceID] = struct{}{}
		}
		instIDs := make([]uint, 0, len(instIDSet))
		for id := range instIDSet {
			instIDs = append(instIDs, id)
		}
		type instDetail struct {
			ID     uint
			Name   string
			UserID uint
		}
		instMap := make(map[uint]instDetail)
		instUserIDs := make(map[uint]struct{})
		if len(instIDs) > 0 {
			var insts []instDetail
			model.DB(r.Context()).Model(&model.Instance{}).Select("id, name, user_id").Where("id IN ?", instIDs).Scan(&insts)
			for _, inst := range insts {
				instMap[inst.ID] = inst
				if inst.UserID > 0 {
					instUserIDs[inst.UserID] = struct{}{}
				}
			}
		}
		instUserIDList := make([]uint, 0, len(instUserIDs))
		for id := range instUserIDs {
			instUserIDList = append(instUserIDList, id)
		}
		instUserMap := make(map[uint]string)
		if len(instUserIDList) > 0 {
			var instUsers []model.User
			model.DB(r.Context()).Where("id IN ?", instUserIDList).Find(&instUsers)
			for _, u := range instUsers {
				instUserMap[u.ID] = u.Username
			}
		}

		recordsByTask := make(map[uint][]model.PluginDistributionRecord)
		for _, rec := range allRecords {
			recordsByTask[rec.TaskID] = append(recordsByTask[rec.TaskID], rec)
		}

		for _, t := range tasks {
			c := countMap[t.ID]
			tr := taskResp{
				ID:        t.ID,
				CreatedAt: t.CreatedAt,
				Version:   t.Version,
				Total:     t.Total,
				Status:    t.Status,
				Type:      t.Type,
				Operator:  userMap[t.OperatorID],
			}
			if c != nil {
				tr.Success = c.Success
				tr.Failed = c.Failed
				tr.Pending = c.Pending
			}
			for _, rec := range recordsByTask[t.ID] {
				rr := recordResp{
					InstanceID:    rec.InstanceID,
					CVMInstanceID: rec.InstanceCID,
					Status:        rec.Status,
					Error:         rec.Error,
				}
				if inst, ok := instMap[rec.InstanceID]; ok {
					rr.InstanceName = inst.Name
					rr.Username = instUserMap[inst.UserID]
				}
				tr.Records = append(tr.Records, rr)
			}
			if tr.Records == nil {
				tr.Records = []recordResp{}
			}
			result = append(result, tr)
		}
	}

	jsonOK(w, map[string]interface{}{
		"tasks":     result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// HandleAdminPluginInstances 查询实例安装情况
func HandleAdminPluginInstances(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	instanceType := r.URL.Query().Get("instance_type")
	slog.Debug("查询插件实例安装情况", "slug", slug, "status", statusFilter, "instance_type", instanceType)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 校验 type 参数（可选）
	taskType := r.URL.Query().Get("type")
	if taskType != "" {
		// type 必填，只接受 distribute / uninstall
		if taskType != "distribute" && taskType != "uninstall" {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgPluginInvalidTypeParam))
			return
		}
	}

	var pluginIDs []uint
	model.DB(r.Context()).Model(&model.Plugin{}).Where("slug = ?", slug).Pluck("id", &pluginIDs)
	if len(pluginIDs) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	// 获取最新版本，用于判断 outdated 状态
	var latestPlugin model.Plugin
	if err := model.DB(r.Context()).Where("slug = ?", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&latestPlugin).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalServerError))
		return
	}

	page, pageSize := parsePagination(r, 500)

	type instResp struct {
		InstanceID            uint       `json:"instance_id"              gorm:"column:instance_id"`
		CVMInstanceID         string     `json:"cvm_instance_id"          gorm:"column:cvm_instance_id"`
		InstanceName          string     `json:"instance_name"            gorm:"column:instance_name"`
		InstanceType          string     `json:"instance_type"            gorm:"column:instance_type"`
		UserID                uint       `json:"user_id"                  gorm:"column:user_id"`
		Source                string     `json:"-"                        gorm:"column:source"`
		Username              string     `json:"username"                 gorm:"column:username"`
		LastCVMState          string     `json:"last_cvm_state"           gorm:"column:last_cvm_state"`
		LastStableState       string     `json:"-"                        gorm:"column:last_stable_state"`
		CurrentOperation      string     `json:"-"                        gorm:"column:current_operation"`
		CurrentOperationState string     `json:"-"                        gorm:"column:current_operation_state"`
		AgentReady            int        `json:"-"                        gorm:"column:agent_ready"`
		CLSAgentStatus        int        `json:"-"                        gorm:"column:cls_agent_status"`
		CLSAgentStatusAt      *time.Time `json:"-"                        gorm:"column:cls_agent_status_at"`
		Status                string     `json:"status"                   gorm:"column:install_status"`
		Version               string     `json:"version"                  gorm:"column:version"`
		LatestVersion         string     `json:"latest_version"           gorm:"column:latest_version"`
	}

	baseQuery, err := model.BuildPluginInstanceQuery(r.Context(), pluginIDs, latestPlugin.Version)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginBuildQueryFail))
		return
	}

	// 按用户组筛选实例（辅助筛选，支持逗号分隔多个 group_id）
	// group_id=0 表示未分组用户的实例，可与正常 group_id 组合使用，如 group_id=0,1,3
	if groupIDStr := r.URL.Query().Get("group_id"); groupIDStr != "" {
		var groupIDs []int
		includeUngrouped := false
		for _, s := range strings.Split(groupIDStr, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				continue
			}
			if id == 0 {
				includeUngrouped = true
			} else if id > 0 {
				groupIDs = append(groupIDs, id)
			}
		}
		if includeUngrouped && len(groupIDs) > 0 {
			// 未分组 + 指定分组：OR 语义
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?) OR instances.user_id IN (?)", ungroupedSubQ, groupedSubQ)
		} else if includeUngrouped {
			// 仅未分组
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?)", ungroupedSubQ)
		} else if len(groupIDs) > 0 {
			// 仅指定分组（使用子查询避免 JOIN 产生重复行）
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id IN (?)", groupedSubQ)
		}
	}

	if search != "" {
		baseQuery = baseQuery.Where("instances.name LIKE ? OR instances.instance_id LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 按实例类型筛选（支持逗号分隔多类型，如 instance_type=openclaw,hermes）
	if instanceType != "" {
		types := strings.Split(instanceType, ",")
		trimmed := make([]string, 0, len(types))
		for _, t := range types {
			if s := strings.TrimSpace(t); s != "" && model.IsValidAgentType(r.Context(), s) {
				trimmed = append(trimmed, s)
			}
		}
		if len(trimmed) > 0 {
			baseQuery = baseQuery.Where("instances.agent_type IN ?", trimmed)
		}
	}

	// 安装状态筛选（SQL 层预过滤，减少全量数据量）
	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		// latestPlugin.Version 在 BuildPluginInstanceQuery 里已校验过，这里必不会报错
		caseClause, _ := model.PluginInstallStatusCase(latestPlugin.Version)
		baseQuery = baseQuery.Where("("+caseClause+") IN ?", statuses)
	}

	// ── 第一步：全量查询（不分页），用于批量计算实例语义状态后内存过滤 ──
	// 安全上限：避免实例数过多导致内存和 CVM API 压力
	const maxPluginInstanceQuery = 5000
	var allResults []instResp
	if err := baseQuery.Order("instances.created_at DESC").Limit(maxPluginInstanceQuery + 1).Scan(&allResults).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreQueryInstancesFail, err))
		return
	}
	truncated := len(allResults) > maxPluginInstanceQuery
	if truncated {
		allResults = allResults[:maxPluginInstanceQuery]
	}
	if allResults == nil {
		allResults = []instResp{}
	}

	// ── 第二步：批量查询 CVM 实时状态 ──
	var cvmIDs []string
	for _, r := range allResults {
		if r.CVMInstanceID != "" {
			cvmIDs = append(cvmIDs, r.CVMInstanceID)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	// ── 批量预查：消除循环内 N+1 ──
	siteConfig := model.GetSiteConfig(r.Context())
	preInstIDs := make([]uint, 0, len(allResults))
	localInstIDs := make([]uint, 0)
	for _, item := range allResults {
		preInstIDs = append(preInstIDs, item.InstanceID)
		if item.Source == model.InstanceSourceLocal {
			localInstIDs = append(localInstIDs, item.InstanceID)
		}
	}
	installingSkillMap := batchHasInstallingSkillInstallations(r.Context(), preInstIDs)
	localInfoMap := batchResolveLocalInstanceStatus(r.Context(), localInstIDs)
	batch := &InstanceStatusBatchLookup{SiteConfig: siteConfig, InstallingSkillMap: installingSkillMap, LocalInfoMap: localInfoMap}

	// ── 第三步：计算每个实例的语义状态，过滤出 running 的实例 ──
	type instWithStatus struct {
		instResp
		InstanceStatus      string
		InstanceStatusLabel string
		Transient           bool
	}
	var runningResults []instWithStatus
	for _, item := range allResults {
		tmpInst := model.Instance{
			LastCVMState:          item.LastCVMState,
			LastStableState:       item.LastStableState,
			CurrentOperation:      item.CurrentOperation,
			CurrentOperationState: item.CurrentOperationState,
			AgentReady:            item.AgentReady,
			CLSAgentStatus:        item.CLSAgentStatus,
			CLSAgentStatusAt:      item.CLSAgentStatusAt,
			InstanceId:            item.CVMInstanceID,
			Source:                item.Source,
		}
		tmpInst.ID = item.InstanceID
		cvmInfo := cvmInfoMap[item.CVMInstanceID]
		statusResp := ResolveInstanceStatus(r.Context(), &tmpInst, cvmInfo, batch)
		// 只保留 instance_status=running 的实例
		if statusResp.Status != model.StatusRunning {
			continue
		}
		runningResults = append(runningResults, instWithStatus{
			instResp:            item,
			InstanceStatus:      statusResp.Status,
			InstanceStatusLabel: statusResp.Label,
			Transient:           statusResp.Transient,
		})
	}

	// ── 第四步：内存分页 ──
	total := int64(len(runningResults))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(runningResults) {
		start = len(runningResults)
	}
	if end > len(runningResults) {
		end = len(runningResults)
	}
	pageResults := runningResults[start:end]

	// 批量加载用户所属分组
	userIDSet := make(map[uint]bool)
	for _, r := range pageResults {
		if r.UserID > 0 {
			userIDSet[r.UserID] = true
		}
	}
	userGroupMap := make(map[uint][]model.UserGroup)
	if len(userIDSet) > 0 {
		userIDs := make([]uint, 0, len(userIDSet))
		for uid := range userIDSet {
			userIDs = append(userIDs, uid)
		}
		if m, err := model.GetUserGroupsByUserIDs(r.Context(), userIDs); err == nil {
			userGroupMap = m
		} else {
			slog.Error("[PluginInstances] 批量查询用户分组失败", "error", err)
		}
	}

	type groupInfo struct {
		GroupID   uint   `json:"group_id"`
		GroupName string `json:"group_name"`
	}
	type instFinalResp struct {
		instResp
		UserGroups          []groupInfo `json:"user_groups"`
		InstanceStatus      string      `json:"instance_status"`
		InstanceStatusLabel string      `json:"instance_status_label"`
		Transient           bool        `json:"transient"`
	}
	finalResults := make([]instFinalResp, 0, len(pageResults))
	for _, r := range pageResults {
		item := instFinalResp{
			instResp:            r.instResp,
			InstanceStatus:      r.InstanceStatus,
			InstanceStatusLabel: r.InstanceStatusLabel,
			Transient:           r.Transient,
		}
		if groups, ok := userGroupMap[r.UserID]; ok {
			for _, g := range groups {
				item.UserGroups = append(item.UserGroups, groupInfo{GroupID: g.ID, GroupName: g.Name})
			}
		}
		if item.UserGroups == nil {
			item.UserGroups = []groupInfo{}
		}
		finalResults = append(finalResults, item)
	}

	resp := map[string]interface{}{
		"instances": finalResults,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}
	if truncated {
		resp["truncated"] = true
		resp["max_query_limit"] = maxPluginInstanceQuery
	}
	jsonOK(w, resp)
}

// handleUpdatePluginMetadata 原地更新插件元信息（兼容 API 用户原有行为）。
// 当 version <= 当前最新版本时调用，仅更新 name/description/npm_package/category_ids。
func handleUpdatePluginMetadata(w http.ResponseWriter, r *http.Request, plugin *model.Plugin) {
	updates := map[string]interface{}{}
	if name := r.FormValue("name"); name != "" {
		updates["name"] = name
	}
	if r.Form != nil {
		if _, exists := r.Form["description"]; exists {
			updates["description"] = r.FormValue("description")
		}
		if _, exists := r.Form["npm_package"]; exists {
			updates["npm_package"] = r.FormValue("npm_package")
		}
	}
	if len(updates) > 0 {
		model.DB(r.Context()).Model(plugin).Updates(updates)
	}

	// 更新分类关联
	if r.Form != nil {
		if _, exists := r.Form["category_ids"]; exists {
			model.DB(r.Context()).Where("plugin_id = ?", plugin.ID).Delete(&model.PluginCategoryMapping{})
			if catIDs := r.FormValue("category_ids"); catIDs != "" {
				for _, s := range strings.Split(catIDs, ",") {
					if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
						model.DB(r.Context()).Create(&model.PluginCategoryMapping{
							PluginID:   plugin.ID,
							CategoryID: uint(id),
						})
					}
				}
			}
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

const pluginDistributionBatchSize = 200

type pluginDistributionTarget struct {
	InstanceID  uint   `gorm:"column:instance_id"`
	InstanceCID string `gorm:"column:cvm_instance_id"`
}

func normalizePluginDistributionStatuses(statuses []string) ([]string, error) {
	return normalizeDistributionStatuses(
		statuses,
		[]string{"uninstalled", "installed", "outdated", "failed", "upgrade_failed", "uninstall_failed", "uninstall_failed_old"},
		[]string{"installing", "uninstalling"},
	)
}

func normalizePluginUninstallStatuses(statuses []string) ([]string, error) {
	return normalizeDistributionStatuses(
		statuses,
		[]string{"installed", "outdated", "upgrade_failed", "uninstall_failed", "uninstall_failed_old"},
		[]string{"installing", "uninstalling"},
	)
}

// HandleDistributePlugin 批量下发插件到实例
func HandleDistributePlugin(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	var req struct {
		Slug    string `json:"slug"`
		Version string `json:"version"`
		distributionSelection
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if err := req.distributionSelection.validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if req.SelectAll {
		if _, err := normalizePluginDistributionStatuses(req.Statuses); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "slug"))
		return
	}
	if !req.SelectAll && len(req.InstanceIDs) > 500 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginMaxInstances500))
		return
	}

	// 去重
	seen := make(map[uint]bool, len(req.InstanceIDs))
	var uniqueIDs []uint
	for _, id := range req.InstanceIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}
	req.InstanceIDs = uniqueIDs

	// 查找插件
	var plugin model.Plugin
	if req.Version == "" || req.Version == "latest" {
		if model.DB(r.Context()).Where("slug = ?", req.Slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&plugin).Error != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginNotFound))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ?", req.Slug, req.Version).First(&plugin).Error != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, req.Slug, req.Version))
			return
		}
	}

	// 获取分布式锁
	lockKey := fmt.Sprintf("plugin_distribute:%s:%s", req.Slug, plugin.Version)
	lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "plugin_distribute"), lockKey, 30*time.Minute)
	if lockErr != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgPluginVersionLocked))
		return
	}

	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}

	if req.SelectAll {
		task, total, err := createPluginSelectAllTask(r.Context(), plugin, model.TaskTypeDistribute, operatorID, req.distributionSelection)
		if err != nil {
			lock.Release()
			var richErr *hcommon.RichError
			if errors.As(err, &richErr) {
				writeError(w, r, http.StatusBadRequest, richErr)
			} else {
				slog.Error("[PluginSelectAll] 创建下发任务失败", "slug", plugin.Slug, "version", plugin.Version, "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail, err))
			}
			return
		}
		taskCtx := i18n.WithPrinter(hcommon.DetachContext(r.Context()), r.Context())
		wg := pluginDistributeWG
		if wg != nil {
			wg.Add(1)
		}
		go func() {
			if wg != nil {
				defer wg.Done()
			}
			defer lock.Release()
			defer recoverPluginSelectAllTaskPanic(taskCtx, task)
			runPluginSelectAllTask(taskCtx, plugin, task)
		}()
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"task_id": task.ID,
			"version": plugin.Version,
			"total":   total,
		})
		return
	}

	// 批量查询实例 CVM InstanceId、RuntimeUser 和 AgentType（避免循环查 DB）
	// 注意：必须在创建 task 之前完成过滤，确保 task.Total 与实际下发数一致
	type instInfo struct {
		ID               uint
		InstanceId       string
		RuntimeUser      string
		AgentType        string
		CurrentOperation string
	}
	var instInfos []instInfo
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Select("id, instance_id, runtime_user, agent_type, current_operation").
		Where("id IN ?", req.InstanceIDs).
		Scan(&instInfos).Error; err != nil {
		lock.Release()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgPluginQueryInstanceInfo, err))
		return
	}

	// 过滤不支持插件的实例类型（通过 AgentType.SupportsPlugin 配置判断）
	cidMap := make(map[uint]string, len(instInfos))
	ruMap := make(map[uint]string, len(instInfos))
	var validIDs []uint
	var skippedCount int
	for _, info := range instInfos {
		if model.IsResourceAdjustmentOperation(info.CurrentOperation) {
			skippedCount++
			continue
		}
		if !model.AgentTypeSupportsPlugin(r.Context(), info.AgentType) {
			skippedCount++
			continue
		}
		cidMap[info.ID] = info.InstanceId
		ruMap[info.ID] = info.RuntimeUser
		validIDs = append(validIDs, info.ID)
	}
	if skippedCount > 0 {
		slog.Info("插件下发跳过不支持插件的实例类型", "slug", req.Slug, "skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginNoValidInstall))
		return
	}
	req.InstanceIDs = validIDs

	// 在过滤后创建 task，确保 Total 与实际下发数一致
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    plugin.Version,
		OperatorID: operatorID,
		Total:      len(req.InstanceIDs),
		Status:     "running",
	}
	if err := model.DB(r.Context()).Create(&task).Error; err != nil {
		lock.Release()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginCreateTaskFail, err))
		return
	}

	records := make([]model.PluginDistributionRecord, 0, len(req.InstanceIDs))
	for _, instID := range req.InstanceIDs {
		records = append(records, model.PluginDistributionRecord{
			TaskID:      task.ID,
			PluginDBID:  plugin.ID,
			InstanceID:  instID,
			InstanceCID: cidMap[instID],
			Version:     plugin.Version,
			Status:      "pending",
		})
	}
	if err := model.DB(r.Context()).Create(&records).Error; err != nil {
		// 清理孤儿 task，避免永久停留在 running 状态
		if cleanErr := model.DB(r.Context()).Model(&task).Updates(map[string]interface{}{
			"status": "completed",
			"failed": task.Total,
		}).Error; cleanErr != nil {
			slog.Error("清理孤儿插件下发 task 失败", "task_id", task.ID, "error", cleanErr)
		}
		lock.Release()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail, err))
		return
	}

	// 生成下载 URL（在 goroutine 内部生成，避免 URL 过期）
	if pluginDistributeWG != nil {
		pluginDistributeWG.Add(1)
	}
	go func(ctx context.Context) {
		if pluginDistributeWG != nil {
			defer pluginDistributeWG.Done()
		}
		defer lock.Release()

		// 在 goroutine 内生成下载 URL，避免大批量下发时 URL 过期
		downloadURL, urlErr := buildSMHDownloadURL(ctx, plugin.COSZipKey, true)

		config := model.GetSiteConfig(ctx)
		maxConcurrency := config.SkillDistributeConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 100
		}
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0
		failedCount := 0

		for _, rec := range records {
			wg.Add(1)
			go func(record model.PluginDistributionRecord, runtimeUser string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// final 防御：插件安装仅 openclaw 支持（AgentType.SupportsPlugin=true only openclaw）
				// 批量下发入口应过滤 agent_type，但此处再做一道 runtime guard，避免跨类型污染
				recAgentType := LookupAgentType(ctx, record.InstanceCID)
				if !model.AgentTypeSupportsPlugin(ctx, recAgentType) {
					slog.Warn("跳过不支持插件的实例",
						"task_id", task.ID,
						"instance_id", record.InstanceCID,
						"agent_type", recAgentType,
						"slug", req.Slug)
					model.DB(ctx).Model(&record).Updates(map[string]interface{}{
						"status": "skipped",
						"error":  fmt.Sprintf("agent_type %s 不支持插件", recAgentType),
					})
					mu.Lock()
					failedCount++
					mu.Unlock()
					return
				}

				if urlErr != nil {
					model.DB(ctx).Model(&record).Updates(map[string]interface{}{
						"status": "failed",
						"error":  "SMH 下载 URL 生成失败: " + urlErr.Error(),
					})
					mu.Lock()
					failedCount++
					mu.Unlock()
					return
				}

				_, err := RunScript(ctx, record.InstanceCID, "install_plugin_from_smh.sh", 180, runtimeUser, nil, map[string]string{
					"download_url":   downloadURL,
					"plugin_slug":    req.Slug,
					"plugin_id":      plugin.PluginID,
					"plugin_version": plugin.Version,
					"plugin_kind":    plugin.Kind,
				})

				if err != nil {
					slog.Error("插件下发脚本执行失败", "task_id", task.ID, "instance_id", record.InstanceCID, "slug", req.Slug, "error", err)
					model.DB(ctx).Model(&record).Updates(map[string]interface{}{
						"status": "failed",
						"error":  err.Error(),
					})
					mu.Lock()
					failedCount++
					mu.Unlock()
				} else {
					model.DB(ctx).Model(&record).Update("status", "success")
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(rec, ruMap[rec.InstanceID])
		}

		wg.Wait()

		model.DB(ctx).Model(&task).Updates(map[string]interface{}{
			"status":  "completed",
			"success": successCount,
			"failed":  failedCount,
		})

		// 下发成功后递增 distribute_count（对齐 skill 侧 OnComplete 逻辑）
		if successCount > 0 {
			model.DB(ctx).Model(&model.Plugin{}).Where("id = ?", plugin.ID).
				UpdateColumn("distribute_count", gorm.Expr("distribute_count + ?", successCount))
		}

		slog.Info("插件下发任务完成", "task_id", task.ID, "slug", req.Slug, "version", plugin.Version, "success", successCount, "failed", failedCount)
	}(hcommon.DetachContext(r.Context()))

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": task.ID,
		"version": plugin.Version,
		"total":   len(req.InstanceIDs),
	})
}

func createPluginSelectAllTask(
	ctx context.Context,
	plugin model.Plugin,
	action string,
	operatorID uint,
	selection distributionSelection,
) (model.PluginDistributionTask, int, error) {
	var statuses []string
	var err error
	if action == model.TaskTypeUninstall {
		statuses, err = normalizePluginUninstallStatuses(selection.Statuses)
	} else {
		statuses, err = normalizePluginDistributionStatuses(selection.Statuses)
	}
	if err != nil {
		return model.PluginDistributionTask{}, 0, err
	}

	var pluginIDs []uint
	if err := model.DB(ctx).Model(&model.Plugin{}).
		Where("slug = ?", plugin.Slug).
		Pluck("id", &pluginIDs).Error; err != nil {
		return model.PluginDistributionTask{}, 0, err
	}
	baseQuery, err := model.BuildPluginInstanceQuery(ctx, pluginIDs, plugin.Version)
	if err != nil {
		return model.PluginDistributionTask{}, 0, err
	}
	statusCase, err := model.PluginInstallStatusCase(plugin.Version)
	if err != nil {
		return model.PluginDistributionTask{}, 0, err
	}
	baseQuery = baseQuery.Where("("+statusCase+") IN ?", statuses)
	baseQuery = model.FilterInstancesByUserGroups(ctx, baseQuery, selection.GroupIDs)
	baseQuery = applyDistributionSearch(baseQuery, selection.Search)

	var task model.PluginDistributionTask
	var afterID uint
	total := 0
	for {
		var rows []pluginDistributionTarget
		if err := baseQuery.Session(&gorm.Session{}).
			Where("instances.id > ?", afterID).
			Order(clause.OrderByColumn{
				Column:  clause.Column{Table: "instances", Name: "id"},
				Reorder: true,
			}).
			Limit(pluginDistributionBatchSize).
			Scan(&rows).Error; err != nil {
			cleanupPluginSelectAllTask(hcommon.DetachContext(ctx), task.ID)
			return model.PluginDistributionTask{}, 0, err
		}
		if len(rows) == 0 {
			break
		}

		targets := make([]pluginDistributionTarget, 0, len(rows))
		seen := make(map[uint]struct{}, len(rows))
		for _, row := range rows {
			if row.InstanceID > afterID {
				afterID = row.InstanceID
			}
			if _, ok := seen[row.InstanceID]; ok {
				continue
			}
			seen[row.InstanceID] = struct{}{}
			targets = append(targets, row)
		}
		if len(targets) == 0 {
			continue
		}

		if task.ID == 0 {
			task = model.PluginDistributionTask{
				PluginDBID: plugin.ID,
				Version:    plugin.Version,
				OperatorID: operatorID,
				Status:     model.TaskStatusRunning,
				Type:       action,
			}
			if err := model.DB(ctx).Create(&task).Error; err != nil {
				cleanupPluginSelectAllTask(hcommon.DetachContext(ctx), task.ID)
				return model.PluginDistributionTask{}, 0, err
			}
		}

		records := make([]model.PluginDistributionRecord, 0, len(targets))
		for _, target := range targets {
			records = append(records, model.PluginDistributionRecord{
				TaskID:      task.ID,
				PluginDBID:  plugin.ID,
				InstanceID:  target.InstanceID,
				InstanceCID: target.InstanceCID,
				Version:     plugin.Version,
				Status:      model.RecordStatusPending,
				Type:        action,
			})
		}
		if err := model.DB(ctx).Create(&records).Error; err != nil {
			cleanupPluginSelectAllTask(hcommon.DetachContext(ctx), task.ID)
			return model.PluginDistributionTask{}, 0, err
		}
		total += len(targets)
	}
	if total == 0 {
		return model.PluginDistributionTask{}, 0, hcommon.I18nError(i18n.MsgPluginNoValidInstall)
	}
	task.Total = total
	if err := model.DB(ctx).Model(&task).Update("total", total).Error; err != nil {
		cleanupPluginSelectAllTask(hcommon.DetachContext(ctx), task.ID)
		return model.PluginDistributionTask{}, 0, err
	}
	return task, total, nil
}

func cleanupPluginSelectAllTask(ctx context.Context, taskID uint) {
	if taskID == 0 {
		return
	}
	if err := model.DB(ctx).Where("task_id = ?", taskID).Delete(&model.PluginDistributionRecord{}).Error; err != nil {
		slog.Error("[PluginSelectAll] 清理下发记录失败", "task_id", taskID, "error", err)
	}
	if err := model.DB(ctx).Delete(&model.PluginDistributionTask{}, taskID).Error; err != nil {
		slog.Error("[PluginSelectAll] 清理下发任务失败", "task_id", taskID, "error", err)
	}
}

func failPendingPluginDistributionRecords(ctx context.Context, taskID uint, cause error) (int, error) {
	result := model.DB(ctx).Model(&model.PluginDistributionRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.RecordStatusPending).
		Updates(map[string]interface{}{
			"status": model.RecordStatusFailed,
			"error":  hcommon.ErrorMessageWithCtx(ctx, cause),
		})
	return int(result.RowsAffected), result.Error
}

type pluginSelectAllJob struct {
	record   model.PluginDistributionRecord
	instance model.Instance
}

func recoverPluginSelectAllTaskPanic(ctx context.Context, task model.PluginDistributionTask) {
	recovered := recover()
	if recovered == nil {
		return
	}
	cause := fmt.Errorf("panic: %v", recovered)
	slog.Error("[PluginSelectAll] task panic", "task_id", task.ID, "panic", recovered, "stack", string(debug.Stack()))
	if _, err := failPendingPluginDistributionRecords(ctx, task.ID, cause); err != nil {
		slog.Error("[PluginSelectAll] 收敛 panic 任务失败", "task_id", task.ID, "error", err)
	}
	var success int64
	if err := model.DB(ctx).Model(&model.PluginDistributionRecord{}).
		Where("task_id = ? AND status = ?", task.ID, model.RecordStatusSuccess).
		Count(&success).Error; err != nil {
		slog.Error("[PluginSelectAll] 统计 panic 任务成功记录失败", "task_id", task.ID, "error", err)
	}
	failed := task.Total - int(success)
	if failed < 0 {
		failed = 0
	}
	if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
		"status":  model.TaskStatusCompleted,
		"success": int(success),
		"failed":  failed,
	}).Error; err != nil {
		slog.Error("[PluginSelectAll] 更新 panic 任务状态失败", "task_id", task.ID, "error", err)
	}
}

func runPluginSelectAllTask(
	ctx context.Context,
	plugin model.Plugin,
	task model.PluginDistributionTask,
) {
	var downloadURL string
	var downloadURLErr error
	var downloadURLOnce sync.Once
	resolveDownloadURL := func() (string, error) {
		downloadURLOnce.Do(func() {
			downloadURL, downloadURLErr = buildSMHDownloadURL(ctx, plugin.COSZipKey, true)
		})
		return downloadURL, downloadURLErr
	}
	var pluginIDs []uint
	if task.Type == model.TaskTypeUninstall {
		if err := model.DB(ctx).Model(&model.Plugin{}).
			Where("slug = ?", plugin.Slug).
			Pluck("id", &pluginIDs).Error; err != nil {
			slog.Error("[PluginSelectAll] 查询插件版本失败", "task_id", task.ID, "error", err)
		}
	}
	maxConcurrency := model.GetSiteConfig(ctx).SkillDistributeConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 100
	}
	successCount := 0
	failedCount := 0
	var afterID uint
	for {
		records, jobs, loadFailed, err := loadPluginSelectAllBatch(ctx, task.ID, afterID)
		if err != nil {
			slog.Error("[PluginSelectAll] 分批读取下发记录失败", "task_id", task.ID, "error", err)
			failed, convergeErr := failPendingPluginDistributionRecords(ctx, task.ID, err)
			if convergeErr != nil {
				slog.Error("[PluginSelectAll] 收敛未处理记录失败", "task_id", task.ID, "error", convergeErr)
			} else {
				failedCount += failed
			}
			break
		}
		if len(records) == 0 {
			break
		}
		afterID = records[len(records)-1].ID
		failedCount += loadFailed
		success, failed := executePluginSelectAllBatch(ctx, plugin, task.Type, pluginIDs, jobs, resolveDownloadURL, maxConcurrency)
		successCount += success
		failedCount += failed
	}

	if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
		"status":  model.TaskStatusCompleted,
		"success": successCount,
		"failed":  failedCount,
	}).Error; err != nil {
		slog.Error("[PluginSelectAll] 更新任务统计失败", "task_id", task.ID, "error", err)
	}
	if task.Type == model.TaskTypeDistribute && successCount > 0 {
		if err := model.DB(ctx).Model(&model.Plugin{}).Where("id = ?", plugin.ID).
			UpdateColumn("distribute_count", gorm.Expr("distribute_count + ?", successCount)).Error; err != nil {
			slog.Error("[PluginSelectAll] 更新下发计数失败", "plugin_id", plugin.ID, "error", err)
		}
	}
}

func loadPluginSelectAllBatch(
	ctx context.Context,
	taskID uint,
	afterID uint,
) ([]model.PluginDistributionRecord, []pluginSelectAllJob, int, error) {
	var records []model.PluginDistributionRecord
	if err := model.DB(ctx).
		Where("task_id = ? AND id > ? AND status = ?", taskID, afterID, model.RecordStatusPending).
		Order("id ASC").
		Limit(pluginDistributionBatchSize).
		Find(&records).Error; err != nil {
		return nil, nil, 0, err
	}
	if len(records) == 0 {
		return nil, nil, 0, nil
	}

	instanceIDs := make([]uint, 0, len(records))
	for _, record := range records {
		instanceIDs = append(instanceIDs, record.InstanceID)
	}
	var instances []model.Instance
	if err := model.DB(ctx).
		Select("id, instance_id, runtime_user, agent_type").
		Where("id IN ?", instanceIDs).
		Find(&instances).Error; err != nil {
		slog.Error("[PluginSelectAll] 批量加载实例失败", "task_id", taskID, "error", err)
		for i := range records {
			if updateErr := model.DB(ctx).Model(&records[i]).Updates(map[string]interface{}{
				"status": model.RecordStatusFailed,
				"error":  hcommon.ErrorMessageWithCtx(ctx, hcommon.I18nRichError(err, i18n.MsgPluginQueryInstanceInfo)),
			}).Error; updateErr != nil {
				slog.Error("[PluginSelectAll] 更新失败记录失败", "record_id", records[i].ID, "error", updateErr)
			}
		}
		return records, nil, len(records), nil
	}
	instanceMap := make(map[uint]model.Instance, len(instances))
	for _, instance := range instances {
		instanceMap[instance.ID] = instance
	}

	jobs := make([]pluginSelectAllJob, 0, len(records))
	failed := 0
	for i := range records {
		instance, ok := instanceMap[records[i].InstanceID]
		if !ok {
			if err := model.DB(ctx).Model(&records[i]).Updates(map[string]interface{}{
				"status": model.RecordStatusFailed,
				"error":  i18n.T(ctx, i18n.MsgInstanceNotFound),
			}).Error; err != nil {
				slog.Error("[PluginSelectAll] 更新缺失实例记录失败", "record_id", records[i].ID, "error", err)
			}
			failed++
			continue
		}
		jobs = append(jobs, pluginSelectAllJob{record: records[i], instance: instance})
	}
	return records, jobs, failed, nil
}

func resolvePluginSelectAllFailedStatus(
	ctx context.Context,
	plugin model.Plugin,
	taskType string,
	pluginIDs []uint,
	record model.PluginDistributionRecord,
) string {
	if taskType == model.TaskTypeUninstall {
		return model.ResolvePluginUninstallFailedStatus(ctx, record.InstanceID, pluginIDs, plugin.Version)
	}
	return model.RecordStatusFailed
}

func executePluginSelectAllBatch(
	ctx context.Context,
	plugin model.Plugin,
	taskType string,
	pluginIDs []uint,
	jobs []pluginSelectAllJob,
	resolveDownloadURL func() (string, error),
	maxConcurrency int,
) (int, int) {
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	defer wg.Wait()
	var mu sync.Mutex
	successCount := 0
	failedCount := 0
	for _, current := range jobs {
		sem <- struct{}{}
		wg.Add(1)
		go func(current pluginSelectAllJob) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				cause := fmt.Errorf("panic: %v", recovered)
				slog.Error("[PluginSelectAll] record panic", "record_id", current.record.ID, "panic", recovered, "stack", string(debug.Stack()))
				failedStatus := resolvePluginSelectAllFailedStatus(ctx, plugin, taskType, pluginIDs, current.record)
				if err := model.DB(ctx).Model(&current.record).Updates(map[string]interface{}{
					"status": failedStatus,
					"error":  hcommon.ErrorMessageWithCtx(ctx, cause),
				}).Error; err != nil {
					slog.Error("[PluginSelectAll] 收敛 panic 记录失败", "record_id", current.record.ID, "error", err)
				}
				mu.Lock()
				failedCount++
				mu.Unlock()
			}()

			var runErr error
			switch {
			case !model.AgentTypeSupportsPlugin(ctx, current.instance.AgentType):
				runErr = hcommon.I18nError(i18n.MsgUnsupportedAgentType, current.instance.AgentType)
			case taskType == model.TaskTypeUninstall:
				var scriptName string
				scriptName, runErr = ResolveScript(ctx, "uninstall_plugin", current.instance.AgentType)
				if runErr == nil {
					_, runErr = RunScript(ctx, current.record.InstanceCID, scriptName, 60, current.instance.RuntimeUser, nil, map[string]string{
						"plugin_slug": plugin.Slug,
						"plugin_id":   plugin.PluginID,
						"plugin_kind": plugin.Kind,
					})
				}
			default:
				var downloadURL string
				downloadURL, runErr = resolveDownloadURL()
				if runErr == nil {
					_, runErr = RunScript(ctx, current.record.InstanceCID, "install_plugin_from_smh.sh", 180, current.instance.RuntimeUser, nil, map[string]string{
						"download_url":   downloadURL,
						"plugin_slug":    plugin.Slug,
						"plugin_id":      plugin.PluginID,
						"plugin_version": plugin.Version,
						"plugin_kind":    plugin.Kind,
					})
				}
			}

			if runErr != nil {
				failedStatus := resolvePluginSelectAllFailedStatus(ctx, plugin, taskType, pluginIDs, current.record)
				if err := model.DB(ctx).Model(&current.record).Updates(map[string]interface{}{
					"status": failedStatus,
					"error":  hcommon.ErrorMessageWithCtx(ctx, runErr),
				}).Error; err != nil {
					slog.Error("[PluginSelectAll] 更新失败记录失败", "record_id", current.record.ID, "error", err)
				}
				mu.Lock()
				failedCount++
				mu.Unlock()
				return
			}
			if err := model.DB(ctx).Model(&current.record).Update("status", model.RecordStatusSuccess).Error; err != nil {
				slog.Error("[PluginSelectAll] 更新成功记录失败", "record_id", current.record.ID, "error", err)
			}
			mu.Lock()
			successCount++
			mu.Unlock()
		}(current)
	}
	wg.Wait()
	return successCount, failedCount
}

// ── 收藏/取消收藏 ──────────────────────────────────────────────────

// HandleFavoritePlugin 收藏公共插件
func HandleFavoritePlugin(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		PluginID    string `json:"plugin_id"`
		Version     string `json:"version"`
		Description string `json:"description"`
		NpmPackage  string `json:"npm_package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	if req.Name == "" || req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNameSlugCannotBeEmpty))
		return
	}

	// 检查是否已收藏过同一个 slug 的插件，避免重复收藏
	var existingCount int64
	model.DB(r.Context()).Model(&model.PublicPlugin{}).Where("slug = ?", req.Slug).Count(&existingCount)
	if existingCount > 0 {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgPluginAlreadyFavorited))
		return
	}

	plugin := model.PublicPlugin{
		Name:        req.Name,
		Slug:        req.Slug,
		PluginID:    req.PluginID,
		Version:     req.Version,
		Description: req.Description,
		NpmPackage:  req.NpmPackage,
	}
	if err := model.DB(r.Context()).Create(&plugin).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginFavoriteFail))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "plugin_id": plugin.ID})
}

// HandleUnfavoritePlugin 取消收藏公共插件
func HandleUnfavoritePlugin(w http.ResponseWriter, r *http.Request) {
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

	var plugin model.PublicPlugin
	if model.DB(r.Context()).First(&plugin, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	model.DB(r.Context()).Delete(&plugin)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminFavoritedPlugins 获取已收藏插件列表
func HandleAdminFavoritedPlugins(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	var total int64
	model.DB(r.Context()).Model(&model.PublicPlugin{}).Count(&total)

	var plugins []model.PublicPlugin
	model.DB(r.Context()).Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&plugins)

	if plugins == nil {
		plugins = []model.PublicPlugin{}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	jsonOK(w, map[string]interface{}{
		"plugins":     plugins,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// HandleUninstallPlugin 批量卸载插件（从实例上移除）
// POST /admin/plugins/uninstall
func HandleUninstallPlugin(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	var req struct {
		Slug string `json:"slug"`
		distributionSelection
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if err := req.distributionSelection.validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if req.SelectAll {
		if _, err := normalizePluginUninstallStatuses(req.Statuses); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	slog.Info("开始批量卸载插件", "slug", req.Slug, "instance_count",
		len(req.InstanceIDs))
	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 去重 instance_ids
	seen := make(map[uint]bool, len(req.InstanceIDs))
	var uniqueIDs []uint
	for _, id := range req.InstanceIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}
	req.InstanceIDs = uniqueIDs

	// 查找插件（取最新版本）
	var plugin model.Plugin
	if model.DB(r.Context()).Where("slug = ?", req.Slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&plugin).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginNotFound))
		return
	}

	// 查同 slug 所有版本的 IDs（用于历史记录查询）
	var allPluginIDs []uint
	model.DB(r.Context()).Model(&model.Plugin{}).Where("slug = ?", req.Slug).
		Pluck("id", &allPluginIDs)

	// 获取分布式锁（与下发共用同一把锁，确保互斥）
	lockKey := fmt.Sprintf("plugin_dist:%d", plugin.ID)
	lock, lockErr := model.AcquireLock(
		hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "plugin_uninstall"),
		lockKey, 30*time.Minute,
	)
	if lockErr != nil {
		slog.Warn("插件卸载获取锁失败", "slug", req.Slug, "error", lockErr)
		writeError(w, r, http.StatusConflict,
			hcommon.I18nError(i18n.MsgPluginVersionLocked))
		return
	}
	if req.SelectAll {
		var operatorID uint
		if user, err := RequestUser(r); user != nil && err == nil {
			operatorID = user.ID
		}
		task, total, err := createPluginSelectAllTask(
			r.Context(),
			plugin,
			model.TaskTypeUninstall,
			operatorID,
			req.distributionSelection,
		)
		if err != nil {
			lock.Release()
			var richErr *hcommon.RichError
			if errors.As(err, &richErr) {
				writeError(w, r, http.StatusBadRequest, richErr)
			} else {
				slog.Error("[PluginSelectAll] 创建卸载任务失败", "slug", plugin.Slug, "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginCreateUninstallTaskFail))
			}
			return
		}
		taskCtx := i18n.WithPrinter(hcommon.DetachContext(r.Context()), r.Context())
		wg := pluginDistributeWG
		if wg != nil {
			wg.Add(1)
		}
		go func() {
			if wg != nil {
				defer wg.Done()
			}
			defer lock.Release()
			defer recoverPluginSelectAllTaskPanic(taskCtx, task)
			runPluginSelectAllTask(taskCtx, plugin, task)
		}()
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"task_id": task.ID,
			"message": "已开始卸载流程",
			"total":   total,
		})
		return
	}

	// 批量查询所有实例的 CVM InstanceId、RuntimeUser 和 AgentType
	type instInfo struct {
		ID               uint
		InstanceId       string
		RuntimeUser      string
		AgentType        string
		CurrentOperation string
	}
	var instInfos []instInfo
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Select("id, instance_id, runtime_user, agent_type, current_operation").
		Where("id IN ?", req.InstanceIDs).
		Scan(&instInfos).Error; err != nil {
		lock.Release()
		slog.Error("[UninstallPlugin] 查询实例信息失败", "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nError(i18n.MsgSkillStoreQueryInstanceFail))
		return
	}

	// 过滤不支持插件的实例类型
	cidMap := make(map[uint]string, len(instInfos))
	ruMap := make(map[uint]string, len(instInfos))
	var validIDs []uint
	var skippedCount int
	for _, info := range instInfos {
		if model.IsResourceAdjustmentOperation(info.CurrentOperation) {
			skippedCount++
			continue
		}
		if !model.AgentTypeSupportsPlugin(r.Context(), info.AgentType) {
			skippedCount++
			continue
		}
		cidMap[info.ID] = info.InstanceId
		ruMap[info.ID] = info.RuntimeUser
		validIDs = append(validIDs, info.ID)
	}
	if skippedCount > 0 {
		slog.Info("插件卸载跳过不支持插件的实例类型", "slug", req.Slug,
			"skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgPluginNoValidInstall))
		return
	}
	req.InstanceIDs = validIDs

	// 获取操作人
	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}

	// 创建卸载任务
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    plugin.Version,
		OperatorID: operatorID,
		Total:      len(req.InstanceIDs),
		Status:     "running",
		Type:       "uninstall",
	}
	if err := model.DB(r.Context()).Create(&task).Error; err != nil {
		lock.Release()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgPluginCreateUninstallTaskFail))
		return
	}

	// 批量构造并插入卸载记录
	records := make([]model.PluginDistributionRecord, 0, len(req.InstanceIDs))
	for _, instID := range req.InstanceIDs {
		records = append(records, model.PluginDistributionRecord{
			TaskID:      task.ID,
			PluginDBID:  plugin.ID,
			InstanceID:  instID,
			InstanceCID: cidMap[instID],
			Version:     plugin.Version,
			Status:      "pending",
			Type:        "uninstall",
		})
	}
	if err := model.DB(r.Context()).Create(&records).Error; err != nil {
		lock.Release()
		slog.Error("创建卸载记录失败", "task_id", task.ID, "slug", req.Slug,
			"error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgPluginCreateUninstallRecordFail))
		return
	}

	// 异步并发执行卸载
	executePluginTaskAsync(PluginTaskConfig{
		Ctx:     hcommon.DetachContext(r.Context()),
		Task:    task,
		Records: records,
		Lock:    lock,
		Slug:    req.Slug,
		OnFailed: func(ctx context.Context, record model.PluginDistributionRecord) string {
			// 通过 ResolveScript 按 agent_type 分派脚本，不支持时跳过
			agentType := LookupAgentType(ctx, record.InstanceCID)
			if _, err := ResolveScript(ctx, "uninstall_plugin", agentType); err != nil {
				slog.Warn("跳过不支持插件卸载的实例",
					"task_id", task.ID,
					"instance_id", record.InstanceCID,
					"agent_type", agentType,
					"slug", req.Slug)
				return "skipped"
			}
			return model.ResolvePluginUninstallFailedStatus(
				ctx, record.InstanceID, allPluginIDs, plugin.Version,
			)
		},
	}, func(ctx context.Context, record model.PluginDistributionRecord) error {
		// 通过 ResolveScript 按 agent_type 分派脚本
		agentType := LookupAgentType(ctx, record.InstanceCID)
		scriptName, resolveErr := ResolveScript(ctx, "uninstall_plugin", agentType)
		if resolveErr != nil {
			return hcommon.I18nError(i18n.MsgPluginAgentTypeNotSupportUninstall, agentType)
		}
		_, err := RunScript(ctx, record.InstanceCID, scriptName,
			60, ruMap[record.InstanceID], nil, map[string]string{
				"plugin_slug": req.Slug,
				"plugin_id":   plugin.PluginID,
				"plugin_kind": plugin.Kind,
			})
		return err
	})

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": task.ID,
		"message": "已开始卸载流程",
	})
}
