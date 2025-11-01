package cache

import (
	"context"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var ErrNotSupported = errors.New("video is not currently supported")

func NewEntry(id string) Entry {
	if num, ok := utils.CheckIdIsPyPy(id); ok {
		return newUrlBasedEntry(id, requesting.GetPyPyClient(), func(ctx context.Context) (string, error) {
			return utils.GetPyPyVideoUrl(num), nil
		})
	}
	if num, ok := utils.CheckIdIsWanna(id); ok {
		return newUrlBasedEntry(id, requesting.GetWannaClient(), func(ctx context.Context) (string, error) {
			return utils.GetWannaVideoUrl(num), nil
		})
	}
	if bvID, ok := utils.CheckIdIsBili(id); ok {
		return newUrlBasedEntry(id, requesting.GetBiliClient(), func(ctx context.Context) (string, error) {
			return third_party_api.GetBiliVideoUrl(bvID, ctx)
		})
	}
	if ytID, ok := utils.CheckIdIsYoutube(id); ok {
		return newUrlBasedEntry(id, requesting.GetYoutubeVideoClient(), func(ctx context.Context) (string, error) {
			return utils.GetStandardYoutubeURL(ytID), nil
		})
	}
	return nil
}
