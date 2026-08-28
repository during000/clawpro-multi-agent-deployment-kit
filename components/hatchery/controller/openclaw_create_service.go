package controller

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"hatchery/model"
)

// createInstancePresets contains resources resolved and authorized before CVM creation.
// The ordinary user create path always leaves this nil.
type createInstancePresets struct {
	Models   []model.AIModel
	Channels []manualChannelPreset
	Skills   []createSkillPreset
}

type createSkillPreset struct {
	Source     string
	Slug       string
	Version    string
	Enterprise model.Skill
}

type createInstanceOptions struct {
	Presets    *createInstancePresets
	CustomTags *[]createInstanceTag
}

type createInstanceResult struct {
	InstanceID string
}

var (
	channelPresetPollInterval = 10 * time.Second
	channelPresetMaxWait      = 10 * time.Minute
)

// recoverCreatePresetPanic prevents a detached create-time preset task from
// terminating the process. It must be called directly with defer so recover
// executes on the panicking goroutine's stack.
func recoverCreatePresetPanic(task string, instanceID uint) {
	if r := recover(); r != nil {
		slog.Error("[CreateInstance] preset task panic",
			"task", task,
			"instance_id", instanceID,
			"error", r,
		)
	}
}

// waitForCreatePresetInstance waits for the newly created Agent to become
// usable. Preset credentials remain only in the goroutine closure; if the
// process exits or the wait fails, the best-effort operation is simply lost.
func waitForCreatePresetInstance(ctx context.Context, instanceID uint) (*model.Instance, error) {
	deadline := time.Now().Add(channelPresetMaxWait)
	for time.Now().Before(deadline) {
		var instance model.Instance
		if err := model.DB(ctx).First(&instance, instanceID).Error; err != nil {
			return nil, err
		}
		if instance.AgentReady == 1 {
			instance.RuntimeUser = ensureRuntimeUser(
				ctx, instance.ID, instance.InstanceId, instance.AgentType,
			)
			return &instance, nil
		}
		timer := time.NewTimer(channelPresetPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("agent ready timeout")
}

// applyChannelPresetsAsync applies create-time channels in request order,
// matching the direct user set-channel behavior. Nothing is persisted or
// retried; logs intentionally contain only the channel identifier.
func applyChannelPresetsAsync(ctx context.Context, externalBaseURL string, instanceID uint, presets []manualChannelPreset) {
	instance, err := waitForCreatePresetInstance(ctx, instanceID)
	if err != nil {
		slog.Warn("[ChannelPreset] 等待实例就绪失败", "instance_id", instanceID)
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, externalBaseURL, nil)
	if err != nil {
		slog.Warn("[ChannelPreset] 构造通道配置请求失败", "instance_id", instanceID)
		return
	}
	for _, preset := range presets {
		if _, err := applyManualChannelConfig(request, instance, preset); err != nil {
			slog.Warn("[ChannelPreset] 通道预设下发失败",
				"instance_id", instanceID,
				"channel", preset.Channel,
			)
			continue
		}
		slog.Info("[ChannelPreset] 通道预设下发成功",
			"instance_id", instanceID,
			"channel", preset.Channel,
		)
	}
}
