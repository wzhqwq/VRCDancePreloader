package cache

import (
	"context"
	"errors"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var ErrNotSupported = errors.New("video is not currently supported")

func NewEntry(id string) Entry {
	if num, ok := utils.CheckIdIsPyPy(id); ok {
		return newUrlBasedEntry(id, requesting.GetPyPyClient(), func(ctx context.Context) (*RemoteVideoInfo, error) {
			return &RemoteVideoInfo{
				FinalUrl: utils.GetPyPyVideoUrl(num),
			}, nil
		})
	}
	if num, ok := utils.CheckIdIsWanna(id); ok {
		return newUrlBasedEntry(id, requesting.GetWannaClient(), func(ctx context.Context) (*RemoteVideoInfo, error) {
			return &RemoteVideoInfo{
				FinalUrl: utils.GetWannaVideoUrl(num),
			}, nil
		})
	}
	if num, ok := utils.CheckIdIsDuDu(id); ok {
		return newUrlBasedEntry(id, requesting.GetDuDuClient(), func(ctx context.Context) (*RemoteVideoInfo, error) {
			if song, ok := raw_song.FindDuDuSong(num); ok {
				return &RemoteVideoInfo{
					FinalUrl:     utils.GetDuDuVideoUrl(num),
					LastModified: time.Unix(int64(song.PublishedAt), 0),
				}, nil
			}
			return &RemoteVideoInfo{
				FinalUrl: utils.GetDuDuVideoUrl(num),
			}, nil
		})
	}
	if bvID, ok := utils.CheckIdIsBili(id); ok {
		return newUrlBasedEntry(id, requesting.GetBiliClient(), func(ctx context.Context) (*RemoteVideoInfo, error) {
			mTime, err := third_party_api.GetBiliVideoModTime(bvID, ctx)
			if err != nil {
				return nil, err
			}

			url, err := third_party_api.GetBiliVideoUrl(bvID, ctx)
			if err != nil {
				return nil, err
			}

			return &RemoteVideoInfo{
				FinalUrl:     url,
				LastModified: mTime,
			}, nil
		})
	}
	if ytID, ok := utils.CheckIdIsYoutube(id); ok {
		return newUrlBasedEntry(id, requesting.GetYoutubeVideoClient(), func(ctx context.Context) (*RemoteVideoInfo, error) {
			return &RemoteVideoInfo{
				FinalUrl: utils.GetStandardYoutubeURL(ytID),
			}, nil
		})
	}
	return nil
}
