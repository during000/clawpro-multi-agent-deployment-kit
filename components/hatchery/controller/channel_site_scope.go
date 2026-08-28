package controller

import (
	"context"

	"hatchery/model"
)

func currentSiteIsOverseas(ctx context.Context) bool {
	return IsOverseasFromCtx(ctx)
}

func filterChannelsByCurrentSiteScope(ctx context.Context, channels []model.AIChannel) []model.AIChannel {
	return model.FilterChannelsBySiteScope(channels, currentSiteIsOverseas(ctx))
}

func filterChannelIDsByCurrentSiteScope(ctx context.Context, channelIDs []string) []string {
	return model.FilterChannelIDsBySiteScope(channelIDs, currentSiteIsOverseas(ctx))
}

func channelInCurrentSiteScope(ctx context.Context, channelID string) bool {
	return model.ChannelInSiteScope(channelID, currentSiteIsOverseas(ctx))
}
