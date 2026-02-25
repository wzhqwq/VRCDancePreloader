package cache

import (
	"context"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func NewEntry(id string) entry.Entry {
	if num, ok := utils.CheckIdIsPyPy(id); ok {
		return entry.NewUrlBasedEntry(id, requesting.GetClient(requesting.PyPyDance), func(ctx context.Context) (*entry.RemoteVideoInfo, error) {
			return &entry.RemoteVideoInfo{
				FinalUrl: utils.GetPyPyVideoUrl(num),
			}, nil
		})
	}
	if num, ok := utils.CheckIdIsWanna(id); ok {
		return entry.NewUrlBasedEntry(id, requesting.GetClient(requesting.WannaDance), func(ctx context.Context) (*entry.RemoteVideoInfo, error) {
			return &entry.RemoteVideoInfo{
				FinalUrl: utils.GetWannaVideoUrl(num),
			}, nil
		})
	}
	if num, ok := utils.CheckIdIsDuDu(id); ok {
		return entry.NewUrlBasedEntry(id, requesting.GetClient(requesting.DuDuFitDance), func(ctx context.Context) (*entry.RemoteVideoInfo, error) {
			if num == 0 {
				// it's an ending video without PublishedAt, but it must update every Tuesday
				// So we assume the LastModified is 21:00 (UTF+8) at last Tuesday (or today's 21:00 if it's Tuesday)
				daysToMinus := (int(time.Now().Weekday()) + 5) % 7
				lastTuesday := time.Now().AddDate(0, 0, -daysToMinus).Truncate(24 * time.Hour)
				return &entry.RemoteVideoInfo{
					FinalUrl:     utils.GetDuDuVideoUrl(num),
					LastModified: lastTuesday.Add(time.Hour * 13),
				}, nil
			}
			if song, ok := raw_song.FindDuDuSong(num); ok {
				return &entry.RemoteVideoInfo{
					FinalUrl:     utils.GetDuDuVideoUrl(num),
					LastModified: time.Unix(int64(song.PublishedAt), 0),
				}, nil
			}
			return &entry.RemoteVideoInfo{
				FinalUrl: utils.GetDuDuVideoUrl(num),
			}, nil
		})
	}
	if bvID, ok := utils.CheckIdIsBili(id); ok {
		return entry.NewUrlBasedEntry(id, requesting.GetClient(requesting.BiliBiliApi), func(ctx context.Context) (*entry.RemoteVideoInfo, error) {
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
	}
	if ytID, ok := utils.CheckIdIsYoutube(id); ok {
		return entry.NewUrlBasedEntry(id, requesting.GetClient(requesting.YouTubeVideo), func(ctx context.Context) (*entry.RemoteVideoInfo, error) {
			return &entry.RemoteVideoInfo{
				FinalUrl: utils.GetStandardYoutubeURL(ytID),
			}, nil
		})
	}
	return nil
}
