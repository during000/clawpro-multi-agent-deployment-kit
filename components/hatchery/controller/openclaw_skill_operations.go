package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

type userSkillDependencies struct {
	execution             skillExecutionDependencies
	logger                *slog.Logger
	prepareDistributeItem func(context.Context, skillTaskItem) (skillTaskItem, string, *hcommon.RichError)
	tryLock               func(context.Context, string) (*model.DistLock, error)
	versions              *publicSkillVersionCache
}

func newUserSkillDependencies() userSkillDependencies {
	return userSkillDependencies{
		execution:             defaultSkillExecutionDependencies(),
		logger:                slog.Default(),
		prepareDistributeItem: prepareDistributeSkillItem,
		tryLock:               model.TryLock,
		versions:              newPublicSkillVersionCache(fetchPublicSkillLatestVersion, time.Now),
	}
}

// NewUserSkillHandlers 创建共享同一版本缓存的用户技能列表、更新和卸载 Handler。
func NewUserSkillHandlers() (list, update, uninstall http.HandlerFunc) {
	deps := newUserSkillDependencies()
	return func(w http.ResponseWriter, r *http.Request) {
			handleSkillsList(w, r, deps)
		}, func(w http.ResponseWriter, r *http.Request) {
			handleUserUpdateSkill(w, r, defaultStatusResolver, deps)
		}, func(w http.ResponseWriter, r *http.Request) {
			handleUserUninstallSkill(w, r, defaultStatusResolver, deps)
		}
}

func isValidRuntimeSkillSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > 128 || strings.Contains(slug, "..") {
		return false
	}
	for i := range len(slug) {
		char := slug[i]
		if char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			i > 0 && (char == '-' || char == '_' || char == '.') {
			continue
		}
		return false
	}
	return true
}

type runtimeSkillListItem struct {
	Slug            string  `json:"slug,omitempty"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Eligible        bool    `json:"eligible"`
	CanUninstall    bool    `json:"can_uninstall"`
	Version         *string `json:"version,omitempty"`
	LatestVersion   *string `json:"latest_version,omitempty"`
	UpdateAvailable *bool   `json:"update_available,omitempty"`
}

func enrichDistributedSkillVersions(ctx context.Context, userID, instanceID uint, output string, deps userSkillDependencies) ([]runtimeSkillListItem, error) {
	items := make([]runtimeSkillListItem, 0)
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		return nil, fmt.Errorf("decode instance skill list: %w", err)
	}

	slugs := make([]string, 0, len(items))
	for _, item := range items {
		if item.Slug != "" {
			slugs = append(slugs, item.Slug)
		}
	}
	states, err := model.ListDistributedSkillStates(ctx, instanceID, slugs)
	if err != nil {
		return nil, err
	}

	enterpriseSlugs := make([]string, 0, len(states))
	publicSlugs := make([]string, 0, len(states))
	for slug, state := range states {
		if !state.Installed {
			continue
		}
		switch state.Source {
		case model.SkillSourceEnterprise:
			enterpriseSlugs = append(enterpriseSlugs, slug)
		case model.SkillSourcePublic:
			publicSlugs = append(publicSlugs, slug)
		}
	}
	enterpriseLatest, err := listVisibleEnterpriseSkillLatest(ctx, userID, enterpriseSlugs)
	if err != nil {
		return nil, err
	}
	publicLatest := listPublicSkillLatest(ctx, publicSlugs, deps)

	for i := range items {
		state, distributed := states[items[i].Slug]
		if !distributed || !state.Installed {
			continue
		}

		latestVersion := ""
		switch state.Source {
		case model.SkillSourceEnterprise:
			if skill, ok := enterpriseLatest[state.Slug]; ok {
				latestVersion = skill.Version
			}
		case model.SkillSourcePublic:
			latestVersion = publicLatest[state.Slug]
		default:
			continue
		}

		version := state.Version
		currentVersion, currentErr := model.NormalizeSkillVersion(version)
		latestComparableVersion, latestErr := model.NormalizeSkillVersion(latestVersion)
		updateAvailable := currentErr == nil && latestErr == nil &&
			model.CompareSemver(currentVersion, latestComparableVersion) < 0
		items[i].Version = &version
		items[i].LatestVersion = &latestVersion
		items[i].UpdateAvailable = &updateAvailable
	}
	return items, nil
}

func listVisibleEnterpriseSkillLatest(ctx context.Context, userID uint, slugs []string) (map[string]model.Skill, error) {
	result := make(map[string]model.Skill, len(slugs))
	if len(slugs) == 0 {
		return result, nil
	}

	var skills []model.Skill
	if err := model.DB(ctx).
		Where("slug IN ? AND id IN (?)", slugs, model.LatestVersionSkillIDs(ctx)).
		Find(&skills).Error; err != nil {
		return nil, err
	}

	groupSkillIDs := make([]uint, 0, len(skills))
	for i := range skills {
		if skills[i].VisibilityType == model.VisibilityGroup {
			groupSkillIDs = append(groupSkillIDs, skills[i].ID)
		}
	}

	userGroupSet := make(map[uint]struct{})
	skillGroups := make(map[uint][]uint)
	if len(groupSkillIDs) > 0 {
		userGroupIDs, err := model.GetUserGroupIDs(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, groupID := range userGroupIDs {
			userGroupSet[groupID] = struct{}{}
		}
		if len(userGroupSet) > 0 {
			skillGroups, err = model.GetSkillVisibilityGroupIDs(ctx, groupSkillIDs)
			if err != nil {
				return nil, err
			}
		}
	}

	for i := range skills {
		visible := skills[i].VisibilityType != model.VisibilityGroup
		for _, groupID := range skillGroups[skills[i].ID] {
			if _, ok := userGroupSet[groupID]; ok {
				visible = true
				break
			}
		}
		if visible && skills[i].COSZipKey != "" {
			result[skills[i].Slug] = skills[i]
		}
	}
	return result, nil
}

func listPublicSkillLatest(ctx context.Context, slugs []string, deps userSkillDependencies) map[string]string {
	result := make(map[string]string, len(slugs))
	if len(slugs) == 0 {
		return result
	}

	const maxConcurrency = 8
	workerCount := min(len(slugs), maxConcurrency)
	jobs := make(chan string)
	const maxLoggedFailures = 10
	failedSlugs := make([]string, 0, min(len(slugs), maxLoggedFailures))
	failedCount := 0
	var firstErr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for slug := range jobs {
				version, err := deps.versions.Latest(ctx, slug, true)
				mu.Lock()
				if err != nil {
					failedCount++
					if len(failedSlugs) < maxLoggedFailures {
						failedSlugs = append(failedSlugs, slug)
					}
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					continue
				}
				result[slug] = version
				mu.Unlock()
			}
		}()
	}
	for _, slug := range slugs {
		jobs <- slug
	}
	close(jobs)
	wg.Wait()
	if failedCount > 0 {
		deps.logger.WarnContext(ctx, "查询 Public 技能最新版本失败",
			"failed_count", failedCount,
			"failed_slugs", failedSlugs,
			"err", firstErr,
		)
	}
	return result
}

func handleUserUpdateSkill(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver, deps userSkillDependencies) {
	user, instance, slug, ok := skillOperationRequest(w, r, resolver)
	if !ok {
		return
	}
	ctx := r.Context()
	state, found, err := getInstalledDistributedSkillState(ctx, instance.ID, slug)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	}
	if !found {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	item, latestVersion, status, richErr := prepareSkillUpdate(ctx, user.ID, state, deps.versions)
	if richErr != nil {
		writeError(w, r, status, richErr)
		return
	}

	installedSkillID := state.SkillID
	lock, err := deps.tryLock(ctx, skillTaskItemLockKey(item))
	if err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreVersionLocked))
		return
	}
	releaseLock := true
	defer func() {
		if releaseLock {
			lock.Release()
		}
	}()

	state, found, err = getInstalledDistributedSkillState(ctx, instance.ID, slug)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	}
	if !found {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}
	if state.Source != item.Source ||
		(state.Source == model.SkillSourceEnterprise && state.SkillID != installedSkillID) {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}
	if model.CompareSemver(state.Version, latestVersion) >= 0 {
		jsonOK(w, map[string]any{
			"slug": slug, "updated": false, "old_version": state.Version,
			"version": state.Version,
		})
		return
	}

	if item.Source == model.SkillSourcePublic {
		item, _, richErr = deps.prepareDistributeItem(ctx, item)
		if richErr != nil {
			writeError(w, r, http.StatusInternalServerError, richErr)
			return
		}
	}
	if err := failPreviousPendingSkillDistribute(ctx, item, []uint{instance.ID}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillUpdateFail, err))
		return
	}

	task, records, err := createSkillOperationTask(ctx, user.ID, instance, item, model.TaskTypeDistribute)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail, err))
		return
	}
	oldVersion := state.Version
	releaseLock = false
	cfg, executor := buildSkillDistributeExecution(
		ctx,
		item,
		task,
		records,
		lock,
		skillOperationInstanceInfo(instance),
		deps.execution,
	)
	if err := executeSkillTask(cfg, executor); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillUpdateFail, err))
		return
	}
	jsonOK(w, map[string]any{
		"slug": slug, "updated": true, "old_version": oldVersion,
		"version": item.Version,
	})
}

func handleUserUninstallSkill(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver, deps userSkillDependencies) {
	user, instance, slug, ok := skillOperationRequest(w, r, resolver)
	if !ok {
		return
	}
	ctx := r.Context()
	state, found, err := getInstalledDistributedSkillState(ctx, instance.ID, slug)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	}
	if !found {
		handleDirectSkillUninstall(w, r, instance, slug, deps)
		return
	}

	item := skillTaskItem{Source: state.Source, Slug: slug, Version: state.Version, SkillID: state.SkillID}
	installedSkillID := state.SkillID
	lockItem := item
	if state.Source == model.SkillSourceEnterprise {
		preparedItem, _, richErr := prepareUninstallSkillItem(ctx, item)
		if richErr != nil {
			writeError(w, r, http.StatusInternalServerError, richErr)
			return
		}
		lockItem = preparedItem
	}
	lock, err := deps.tryLock(ctx, skillTaskItemLockKey(lockItem))
	if err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}
	releaseLock := true
	defer func() {
		if releaseLock {
			lock.Release()
		}
	}()

	state, found, err = getInstalledDistributedSkillState(ctx, instance.ID, slug)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	}
	if !found {
		jsonOK(w, map[string]any{"slug": slug, "uninstalled": true, "version": state.Version})
		return
	}
	if state.Source != item.Source ||
		(state.Source == model.SkillSourceEnterprise && state.SkillID != installedSkillID) {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}
	item.Source = state.Source
	item.Version = state.Version
	item.SkillID = state.SkillID
	task, records, err := createSkillOperationTask(ctx, user.ID, instance, item, model.TaskTypeUninstall)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateUninstallTask, err))
		return
	}
	releaseLock = false
	cfg, executor := buildSkillUninstallExecution(
		ctx,
		item,
		task,
		records,
		lock,
		skillOperationInstanceInfo(instance),
		deps.execution,
	)
	if err := executeSkillTask(cfg, executor); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillDeleteFail, err))
		return
	}
	jsonOK(w, map[string]any{"slug": slug, "uninstalled": true, "version": state.Version})
}

func handleDirectSkillUninstall(w http.ResponseWriter, r *http.Request, instance *model.Instance, slug string, deps userSkillDependencies) {
	ctx := r.Context()
	lockItem := skillTaskItem{Source: instance.AgentType, Slug: slug}
	lock, err := deps.tryLock(ctx, skillTaskItemLockKey(lockItem))
	if err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}
	defer lock.Release()

	if _, distributed, err := getInstalledDistributedSkillState(ctx, instance.ID, slug); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	} else if distributed {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}

	scriptName, resolveErr := ResolveScript(ctx, "uninstall_skill", instance.AgentType)
	if resolveErr != nil {
		err := hcommon.I18nError(i18n.MsgUnsupportedAgentType, instance.AgentType)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillDeleteFail, err))
		return
	}
	if _, err := deps.execution.runScript(ctx, instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, map[string]string{
		"skill_slug": slug,
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillDeleteFail, err))
		return
	}
	jsonOK(w, map[string]any{"slug": slug, "uninstalled": true})
}

func skillOperationRequest(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) (*model.User, *model.Instance, string, bool) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return nil, nil, "", false
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return nil, nil, "", false
	}
	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return nil, nil, "", false
	}
	if rejectLocalOrWrite(w, r, instance) {
		return nil, nil, "", false
	}
	if richErr := checkInstanceSupportsSkill(r.Context(), instance); richErr != nil {
		writeError(w, r, http.StatusForbidden, richErr)
		return nil, nil, "", false
	}
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return nil, nil, "", false
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return nil, nil, "", false
	}
	if !isValidRuntimeSkillSlug(slug) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "slug"))
		return nil, nil, "", false
	}
	return user, instance, slug, true
}

func getInstalledDistributedSkillState(ctx context.Context, instanceID uint, slug string) (model.DistributedSkillState, bool, error) {
	states, err := model.ListDistributedSkillStates(ctx, instanceID, []string{slug})
	if err != nil {
		return model.DistributedSkillState{}, false, err
	}
	state, found := states[slug]
	supported := state.Source == model.SkillSourceEnterprise || state.Source == model.SkillSourcePublic
	return state, found && state.Installed && supported, nil
}

func prepareSkillUpdate(ctx context.Context, userID uint, state model.DistributedSkillState, versions *publicSkillVersionCache) (skillTaskItem, string, int, *hcommon.RichError) {
	item := skillTaskItem{Source: state.Source, Slug: state.Slug}
	switch state.Source {
	case model.SkillSourceEnterprise:
		skill, status, err := resolveVisibleEnterpriseSkill(ctx, userID, state.Slug, "")
		if err != nil {
			return item, "", status, hcommon.EnsureRichErrorOrPanic(err)
		}
		item.SkillID = skill.ID
		item.Version = skill.Version
		item.COSZipKey = skill.COSZipKey
		return item, skill.Version, 0, nil
	case model.SkillSourcePublic:
		latestVersion, err := versions.Latest(ctx, state.Slug, false)
		if err != nil {
			return item, "", http.StatusBadGateway, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed)
		}
		item.Version = latestVersion
		return item, latestVersion, 0, nil
	default:
		return item, "", http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist)
	}
}

func createSkillOperationTask(ctx context.Context, operatorID uint, instance *model.Instance, item skillTaskItem, action string) (model.SkillDistributionTask, []model.SkillDistributionRecord, error) {
	createdAt := time.Now()
	infoMap := skillOperationInstanceInfo(instance)
	return createSkillTaskAndRecords(ctx, item, action, operatorID, []uint{instance.ID}, infoMap, newSkillTaskBatchID(createdAt), createdAt)
}

func skillOperationInstanceInfo(instance *model.Instance) map[uint]skillInstanceInfo {
	return map[uint]skillInstanceInfo{
		instance.ID: {
			ID:          instance.ID,
			InstanceId:  instance.InstanceId,
			RuntimeUser: instance.RuntimeUser,
			AgentType:   instance.AgentType,
			Source:      instance.Source,
		},
	}
}
