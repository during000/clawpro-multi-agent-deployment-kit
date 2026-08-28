package controller

import (
	"encoding/json"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

const (
	cvmChargeTypePrepaid        = "PREPAID"
	cvmChargeTypePostpaidByHour = "POSTPAID_BY_HOUR"
)

func instanceChargeTypeFromCVMTemplate(cvmTemplate string) string {
	tplMap, err := cvmTemplateMap(cvmTemplate)
	if err != nil {
		return cvmChargeTypePrepaid
	}
	chargeType, _ := tplMap["InstanceChargeType"].(string)
	chargeType = strings.ToUpper(strings.TrimSpace(chargeType))
	if chargeType == "" {
		// 腾讯云 RunInstances 的 InstanceChargeType 缺省值为 POSTPAID_BY_HOUR。
		return cvmChargeTypePostpaidByHour
	}
	return chargeType
}

func instanceChargeTypeOrDefault(instanceChargeType string) string {
	chargeType := strings.TrimSpace(instanceChargeType)
	if chargeType == "" {
		return cvmChargeTypePrepaid
	}
	return chargeType
}

func applyInstanceChargeTypeToCVMTemplate(cvmTemplate, instanceChargeType string) (string, error) {
	chargeType := strings.ToUpper(strings.TrimSpace(instanceChargeType))
	tplMap, err := cvmTemplateMap(cvmTemplate)
	if err != nil {
		return "", err
	}
	switch chargeType {
	case cvmChargeTypePostpaidByHour:
		tplMap["InstanceChargeType"] = cvmChargeTypePostpaidByHour
	case cvmChargeTypePrepaid:
		tplMap["InstanceChargeType"] = cvmChargeTypePrepaid
	default:
		return "", hcommon.I18nError(i18n.MsgInstanceChargeTypeUnsupported)
	}
	b, marshalErr := json.Marshal(tplMap)
	if marshalErr != nil {
		return "", hcommon.I18nRichError(marshalErr, i18n.MsgSerializeCVMTemplateFailed)
	}
	return string(b), nil
}

func cvmTemplateMap(cvmTemplate string) (map[string]interface{}, error) {
	src := strings.TrimSpace(cvmTemplate)
	if src == "" {
		src = model.DefaultCVMTemplate
	}
	var tplMap map[string]interface{}
	if err := json.Unmarshal([]byte(src), &tplMap); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCVMTemplateFormatError)
	}
	if tplMap == nil {
		tplMap = map[string]interface{}{}
	}
	return tplMap, nil
}
