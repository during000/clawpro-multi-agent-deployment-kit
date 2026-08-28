package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// createInstanceTag is the shared custom-tag shape accepted by both create APIs.
type createInstanceTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// parseCreateInstanceTags parses the JSON array carried in the user-facing
// application/x-www-form-urlencoded request. Tag value validation deliberately
// remains authoritative in CVM RunInstances so local rules cannot drift.
func parseCreateInstanceTags(raw string) ([]createInstanceTag, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var tags []createInstanceTag
	if err := decoder.Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode custom tags: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("custom tags must contain one JSON value")
		}
		return nil, fmt.Errorf("decode custom tags: %w", err)
	}
	return tags, nil
}

// mergeCreateInstanceTags combines request-scoped tags with managed defaults.
// A request-scoped tag wins when both sources use the same key. Duplicate keys
// within the request are preserved for RunInstances to validate.
func mergeCreateInstanceTags(customTags []createInstanceTag, defaultTags []model.TagItem) []*cvm.Tag {
	tags := make([]*cvm.Tag, 0, len(customTags)+len(defaultTags))
	customKeys := make(map[string]struct{}, len(customTags))
	for _, tag := range customTags {
		customKeys[tag.Key] = struct{}{}
		tags = append(tags, &cvm.Tag{
			Key:   common.StringPtr(tag.Key),
			Value: common.StringPtr(tag.Value),
		})
	}
	for _, tag := range defaultTags {
		if _, overridden := customKeys[tag.Key]; overridden {
			continue
		}
		tags = append(tags, &cvm.Tag{
			Key:   common.StringPtr(tag.Key),
			Value: common.StringPtr(tag.Value),
		})
	}
	return tags
}

func applyCreateInstanceTags(request *cvm.RunInstancesRequest, customTags []createInstanceTag, defaultTags []model.TagItem) int {
	tags := mergeCreateInstanceTags(customTags, defaultTags)
	if len(tags) == 0 {
		return 0
	}
	request.TagSpecification = []*cvm.TagSpecification{{
		ResourceType: common.StringPtr("instance"),
		Tags:         tags,
	}}
	return len(tags)
}

// createInstanceTagItemsForCache returns the exact tag set sent to
// RunInstances in the legacy CVM tag-cache shape.
func createInstanceTagItemsForCache(customTags []createInstanceTag, defaultTags []model.TagItem) []model.TagItem {
	merged := mergeCreateInstanceTags(customTags, defaultTags)
	items := make([]model.TagItem, 0, len(merged))
	for _, tag := range merged {
		if tag == nil || tag.Key == nil || tag.Value == nil {
			continue
		}
		items = append(items, model.TagItem{Key: *tag.Key, Value: *tag.Value})
	}
	return items
}
