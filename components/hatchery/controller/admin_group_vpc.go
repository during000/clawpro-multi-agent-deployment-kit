package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// 云 API 校验函数变量（可在测试中替换）
var (
	checkVpcExists    = validateVpcExists
	checkSubnetsInVpc = validateSubnetsInVpc
)

// ──────────────────────────────────────────────
// GET /admin/group-vpc-configs
// ──────────────────────────────────────────────

type vpcConfigListItem struct {
	ID               uint                `json:"id"`
	VpcId            string              `json:"vpc_id"`
	SubnetIds        map[string][]string `json:"subnet_ids"`
	StrategyName     string              `json:"strategy_name"`
	VisibilityType   string              `json:"visibility_type"`
	VisibilityGroups []vpcVisibilityItem `json:"visibility_groups"`
	CreatedAt        string              `json:"created_at"`
	UpdatedAt        string              `json:"updated_at"`
}

type vpcVisibilityItem struct {
	GroupID       uint   `json:"group_id"`
	GroupName     string `json:"group_name"`
	GroupFullPath string `json:"group_full_path"`
}

// HandleListGroupVpcConfigs 列出所有 VPC 配置
func HandleListGroupVpcConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	var configs []model.VpcConfig
	if err := model.DB(r.Context()).Order("id ASC").Find(&configs).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgQueryVpcConfigFailed))
		return
	}

	// 收集所有需要查询的 config_id
	configIDs := make([]string, 0, len(configs))
	for _, cfg := range configs {
		configIDs = append(configIDs, strconv.FormatUint(uint64(cfg.ID), 10))
	}

	// 批量查询 group_config_bindings
	var bindings []model.GroupConfigBinding
	if len(configIDs) > 0 {
		model.DB(r.Context()).Where("config_type = ? AND config_key IN ?",
			model.ConfigTypeVPC, configIDs).Find(&bindings)
	}

	// 按 config_key 分组绑定
	bindingMap := make(map[uint][]model.GroupConfigBinding)
	groupIDs := make([]uint, 0)
	for _, b := range bindings {
		var configID uint
		fmt.Sscanf(b.ConfigKey, "%d", &configID)
		bindingMap[configID] = append(bindingMap[configID], b)
		groupIDs = append(groupIDs, b.GroupID)
	}

	// 批量查分组信息
	groupMap := make(map[uint]model.UserGroup)
	if len(groupIDs) > 0 {
		var groups []model.UserGroup
		model.DB(r.Context()).Where("id IN ?", groupIDs).Find(&groups)
		for _, g := range groups {
			groupMap[g.ID] = g
		}
	}

	// 组装响应
	result := make([]vpcConfigListItem, 0, len(configs))
	for _, cfg := range configs {
		subnetMap, err := cfg.GetSubnetMap()
		if err != nil {
			continue
		}
		item := vpcConfigListItem{
			ID:               cfg.ID,
			VpcId:            cfg.VpcId,
			SubnetIds:        subnetMap,
			StrategyName:     cfg.StrategyName,
			VisibilityType:   cfg.VisibilityType,
			VisibilityGroups: []vpcVisibilityItem{},
			CreatedAt:        cfg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        cfg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// 填充可见分组
		if cfg.VisibilityType == model.VisibilityGroup {
			for _, b := range bindingMap[cfg.ID] {
				g := groupMap[b.GroupID]
				item.VisibilityGroups = append(item.VisibilityGroups, vpcVisibilityItem{
					GroupID:       g.ID,
					GroupName:     g.Name,
					GroupFullPath: g.FullPath,
				})
			}
		}
		result = append(result, item)
	}

	jsonOK(w, map[string]interface{}{"data": result})
}

// ──────────────────────────────────────────────
// POST /admin/group-vpc-configs/create
// ──────────────────────────────────────────────

// HandleCreateGroupVpcConfig 创建 VPC 配置
func HandleCreateGroupVpcConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		VpcId        string `json:"vpc_id"`
		SubnetIds    string `json:"subnet_ids"`
		StrategyName string `json:"strategy_name"`
		GroupIDs     []uint `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	if err := validateVpcConfigRequest(r.Context(), req.VpcId, req.SubnetIds,
		req.StrategyName, req.GroupIDs, 0); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 解析 subnet_ids JSON
	var subnetMap map[string][]string
	if err := json.Unmarshal([]byte(req.SubnetIds), &subnetMap); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSubnetIdsParseFailed))
		return
	}

	// 校验 VPC 和子网
	if err := checkVpcExists(r.Context(), req.VpcId); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := checkSubnetsInVpc(r.Context(), req.VpcId, subnetMap); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 校验 group_ids 未绑定到其他 vpc_configs
	if err := validateGroupsNotBoundToVpc(r.Context(), req.GroupIDs, 0); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 事务：创建 vpc_configs + group_config_bindings
	tx := model.DB(r.Context()).Begin()
	visType := "all"
	if len(req.GroupIDs) > 0 {
		visType = "group"
	}

	vpcConfig := model.VpcConfig{
		VpcId:          req.VpcId,
		SubnetIds:      req.SubnetIds,
		StrategyName:   req.StrategyName,
		VisibilityType: visType,
	}
	if err := tx.Create(&vpcConfig).Error; err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgCreateVpcConfigFailed))
		return
	}

	// 创建 group_config_bindings
	if len(req.GroupIDs) > 0 {
		configKey := strconv.FormatUint(uint64(vpcConfig.ID), 10)
		if err := model.SetAdditiveBindings(tx, model.ConfigTypeVPC,
			configKey, req.GroupIDs); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgCreateGroupBindingFailed))
			return
		}
	}

	tx.Commit()
	jsonOK(w, map[string]interface{}{"id": vpcConfig.ID})
}

// ──────────────────────────────────────────────
// POST /admin/group-vpc-configs/update
// ──────────────────────────────────────────────

// HandleUpdateGroupVpcConfig 更新 VPC 配置
func HandleUpdateGroupVpcConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		ID           uint   `json:"id"`
		VpcId        string `json:"vpc_id"`
		SubnetIds    string `json:"subnet_ids"`
		StrategyName string `json:"strategy_name"`
		GroupIDs     []uint `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}

	// 查找现有配置
	var existing model.VpcConfig
	if model.DB(r.Context()).First(&existing, req.ID).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgVpcConfigNotFound))
		return
	}

	if err := validateVpcConfigRequest(r.Context(), req.VpcId, req.SubnetIds,
		req.StrategyName, req.GroupIDs, req.ID); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 解析 subnet_ids JSON
	var subnetMap map[string][]string
	if err := json.Unmarshal([]byte(req.SubnetIds), &subnetMap); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSubnetIdsParseFailed))
		return
	}

	// 校验 VPC 和子网
	if err := checkVpcExists(r.Context(), req.VpcId); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := checkSubnetsInVpc(r.Context(), req.VpcId, subnetMap); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 校验 group_ids 未绑定到其他 vpc_configs（排除自身）
	if err := validateGroupsNotBoundToVpc(r.Context(), req.GroupIDs, req.ID); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError, err.Error()))
		return
	}

	// 事务：更新 vpc_configs + 全量替换 group_config_bindings
	tx := model.DB(r.Context()).Begin()
	visType := "all"
	if len(req.GroupIDs) > 0 {
		visType = "group"
	}

	updates := map[string]interface{}{
		"vpc_id":          req.VpcId,
		"subnet_ids":      req.SubnetIds,
		"strategy_name":   req.StrategyName,
		"visibility_type": visType,
	}
	if err := tx.Model(&existing).Updates(updates).Error; err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgUpdateVpcConfigFailed))
		return
	}

	// 全量替换绑定
	configKey := strconv.FormatUint(uint64(req.ID), 10)
	if err := model.SetAdditiveBindings(tx, model.ConfigTypeVPC,
		configKey, req.GroupIDs); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgUpdateGroupBindingFailed))
		return
	}

	tx.Commit()
	jsonOK(w, map[string]interface{}{"ok": true})
}

// ──────────────────────────────────────────────
// POST /admin/group-vpc-configs/delete
// ──────────────────────────────────────────────

// HandleDeleteGroupVpcConfig 删除 VPC 配置
func HandleDeleteGroupVpcConfig(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}

	// 查找配置记录
	var vpcConfig model.VpcConfig
	if model.DB(r.Context()).First(&vpcConfig, req.ID).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgVpcConfigNotFound))
		return
	}

	// 事务：删除 vpc_configs + 清理 group_config_bindings
	tx := model.DB(r.Context()).Begin()
	if err := tx.Delete(&vpcConfig).Error; err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgDeleteVpcConfigFailed))
		return
	}

	// 清理绑定
	configKey := strconv.FormatUint(uint64(req.ID), 10)
	if err := tx.Where("config_type = ? AND config_key = ?",
		model.ConfigTypeVPC, configKey).Delete(&model.GroupConfigBinding{}).Error; err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgDeleteGroupBindingFailed))
		return
	}

	tx.Commit()
	jsonOK(w, map[string]interface{}{"ok": true})
}

// ──────────────────────────────────────────────
// 辅助函数
// ──────────────────────────────────────────────

// validateVpcConfigRequest 校验 VPC 配置请求参数
func validateVpcConfigRequest(ctx context.Context, vpcId, subnetIds, strategyName string,
	groupIDs []uint, excludeID uint) error {
	if vpcId == "" {
		return hcommon.I18nError(i18n.MsgVpcIdCannotBeEmpty)
	}
	if subnetIds == "" {
		return hcommon.I18nError(i18n.MsgSubnetIdsCannotBeEmpty)
	}
	if len(groupIDs) == 0 {
		return hcommon.I18nError(i18n.MsgGroupRequiredForVpc)
	}
	if len(strategyName) > 20 {
		return hcommon.I18nError(i18n.MsgStrategyNameTooLong)
	}

	// 校验 subnet_ids JSON 格式
	var subnetMap map[string][]string
	if err := json.Unmarshal([]byte(subnetIds), &subnetMap); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSubnetIdsInvalidJSON)
	}

	// 校验 group_ids 存在性
	if len(groupIDs) > 0 {
		var count int64
		model.DB(ctx).Model(&model.UserGroup{}).Where("id IN ?", groupIDs).Count(&count)
		if count != int64(len(groupIDs)) {
			return hcommon.I18nError(i18n.MsgPartialUserGroupsNotFound)
		}
	}

	return nil
}

// validateGroupsNotBoundToVpc 校验 group_ids 未绑定到其他 vpc_configs
func validateGroupsNotBoundToVpc(ctx context.Context, groupIDs []uint, excludeID uint) error {
	if len(groupIDs) == 0 {
		return nil
	}

	// 查询这些分组的 vpc 绑定
	bindings, err := model.GetBindingsByGroups(ctx, groupIDs, model.ConfigTypeVPC)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgQueryGroupBindingFailed)
	}

	// 校验是否有其他 vpc_configs 占用
	for _, b := range bindings {
		var configID uint
		fmt.Sscanf(b.ConfigKey, "%d", &configID)
		if configID != excludeID {
			return hcommon.I18nError(i18n.MsgGroupAlreadyBoundVpc)
		}
	}

	return nil
}

// validateVpcExists 校验 VPC 在腾讯云是否存在
func validateVpcExists(ctx context.Context, vpcId string) error {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}
	req := vpc.NewDescribeVpcsRequest()
	req.VpcIds = common.StringPtrs([]string{vpcId})
	resp, err := vpcClient.DescribeVpcs(req)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgVerifyVpcFailed)
	}
	if resp.Response == nil || *resp.Response.TotalCount == 0 {
		return hcommon.I18nError(i18n.MsgVpcNotExist, vpcId)
	}
	return nil
}

// validateSubnetsInVpc 校验子网属于指定 VPC 且可用区匹配
func validateSubnetsInVpc(ctx context.Context, vpcId string, subnetMap map[string][]string) error {
	var allSubnetIds []string
	for _, sids := range subnetMap {
		allSubnetIds = append(allSubnetIds, sids...)
	}
	if len(allSubnetIds) == 0 {
		return hcommon.I18nError(i18n.MsgSubnetRequired)
	}

	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}

	descReq := vpc.NewDescribeSubnetsRequest()
	descReq.SubnetIds = common.StringPtrs(allSubnetIds)
	resp, err := vpcClient.DescribeSubnets(descReq)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgVerifySubnetFailed)
	}
	if resp.Response == nil {
		return hcommon.I18nError(i18n.MsgSubnetVerifyRespEmpty)
	}

	cloudSubnets := make(map[string]string)
	cloudSubnetZone := make(map[string]string)
	for _, s := range resp.Response.SubnetSet {
		if s.SubnetId != nil {
			sid := *s.SubnetId
			if s.VpcId != nil {
				cloudSubnets[sid] = *s.VpcId
			}
			if s.Zone != nil {
				cloudSubnetZone[sid] = *s.Zone
			}
		}
	}

	for zone, sids := range subnetMap {
		for _, sid := range sids {
			ownerVpc, exists := cloudSubnets[sid]
			if !exists {
				return hcommon.I18nError(i18n.MsgSubnetNotExist, sid)
			}
			if ownerVpc != vpcId {
				return hcommon.I18nError(i18n.MsgSubnetNotBelongVpc, sid, vpcId)
			}
			actualZone := cloudSubnetZone[sid]
			if actualZone != zone {
				return hcommon.I18nError(i18n.MsgSubnetZoneMismatch, sid, actualZone, zone)
			}
		}
	}
	return nil
}
