package third_party_api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"google.golang.org/api/youtube/v3"
)

var youtubeVideoInfoCache = utils.NewWeakCache[*youtube.Video](10)

func GetYoutubeInfoFromApi(videoID string) (*youtube.Video, error) {
	if info, ok := youtubeVideoInfoCache.Get(videoID); ok {
		logger.DebugLn("cache hit", videoID)
		return info, nil
	}

	if YoutubeApiKey == "" {
		return nil, errors.New("empty YouTube API key")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	svc, err := youtube.NewService(ctx, requesting.WithYoutubeApiClient(YoutubeApiKey))
	if err != nil {
		return nil, err
	}

	call := svc.Videos.List([]string{"snippet", "contentDetails"}).Id(videoID)
	resp, err := call.Do()
	if err != nil {
		return nil, err
	}

	if len(resp.Items) == 0 {
		return nil, errors.New("video not found")
	}
	info := resp.Items[0]

	youtubeVideoInfoCache.Set(videoID, info)
	return info, nil
}

func GetYoutubeTitleFromApi(videoID string) (string, error) {
	info, err := GetYoutubeInfoFromApi(videoID)
	if err != nil {
		return "", err
	}

	return info.Snippet.Title, nil
}

func GetYoutubeDurationFromApi(videoID string) (time.Duration, error) {
	info, err := GetYoutubeInfoFromApi(videoID)
	if err != nil {
		return 0, err
	}

	iso8601Duration := info.ContentDetails.Duration
	if !strings.Contains(iso8601Duration, "T") {
		return 0, errors.New("invalid duration format: " + iso8601Duration)
	}
	// P3Y6M4DT12H30M5S -> 12H30M5S -> 12h30m5s
	timeStr := strings.Split(iso8601Duration, "T")[1]
	// I think there's no video longer than 1 day
	return time.ParseDuration(strings.ToLower(timeStr))
}

func GetYoutubeTitle(videoID string) string {
	title, err := GetYoutubeTitleFromApi(videoID)
	if err != nil {
		logger.ErrorLn("Failed to get YouTube title by api:", videoID)
		return "YouTube " + videoID
	}
	return title
}

func GetYoutubeDuration(videoID string) (time.Duration, error) {
	return GetYoutubeDurationFromApi(videoID)
}
