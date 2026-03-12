package providers

import (
	"context"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/video_cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
)

func InitThirdPartyVideoProviders() {
	video_cache.SetBiliBiliUrlProvider(func(bvID string, ctx context.Context) (*entry.RemoteVideoInfo, error) {
		mTime, err := third_party_api.GetBiliVideoModTime(bvID, ctx)
		if err != nil {
			return nil, err
		}

		url, err := third_party_api.GetBiliVideoUrl(bvID, ctx)
		if err != nil {
			return nil, err
		}

		return &entry.RemoteVideoInfo{
			FinalUrl:     url,
			LastModified: mTime,
		}, nil
	})
	video_cache.SetYoutubeUrlProvider(func(youtubeID string, ctx context.Context) (*entry.RemoteVideoInfo, error) {
		url, err := local_executables.ResolveVideoUrlWithYtDlp(youtubeID, ctx)
		if err != nil {
			return nil, err
		}

		return &entry.RemoteVideoInfo{
			FinalUrl: url,
		}, nil
	})
}
