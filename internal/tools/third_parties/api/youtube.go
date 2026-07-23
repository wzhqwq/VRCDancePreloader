package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"google.golang.org/api/youtube/v3"
)

var youtubeVideoInfoCache = utils.NewWeakCache[*youtube.Video](10)

func GetYoutubeInfoFromApi(videoID, apiKey string, ctx context.Context) (*youtube.Video, error) {
	if info, ok := youtubeVideoInfoCache.Get(videoID); ok {
		return info, nil
	}

	apiCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	svc, err := youtube.NewService(apiCtx, requesting.WithYoutubeApiClient(apiKey))
	if err != nil {
		return nil, err
	}

	call := svc.Videos.List([]string{"snippet", "contentDetails"}).Id(videoID)
	resp, err := call.Do()
	if err != nil {
		if cause := context.Cause(apiCtx); errors.Is(err, context.Canceled) && cause != nil {
			return nil, cause
		}
		return nil, err
	}

	if len(resp.Items) == 0 {
		return nil, errors.New("video not found")
	}
	info := resp.Items[0]

	youtubeVideoInfoCache.Set(videoID, info)
	return info, nil
}

func GetYoutubeInfo(videoID, apiKey string, ctx context.Context) (*types.GeneralVideoInfo, error) {
	info, err := GetYoutubeInfoFromApi(videoID, apiKey, ctx)
	if err != nil {
		return nil, err
	}

	iso8601Duration := info.ContentDetails.Duration
	if !strings.Contains(iso8601Duration, "T") {
		return nil, errors.New("invalid duration format: " + iso8601Duration)
	}
	// P3Y6M4DT12H30M5S -> 12H30M5S -> 12h30m5s
	timeStr := strings.Split(iso8601Duration, "T")[1]
	// I think there's no video longer than 1 day
	duration, err := time.ParseDuration(strings.ToLower(timeStr))
	if err != nil {
		return nil, err
	}

	return &types.GeneralVideoInfo{
		Title:     info.Snippet.Title,
		Duration:  duration,
		GroupName: info.Snippet.ChannelTitle,
	}, nil
}
