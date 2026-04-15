package third_party_api

import (
	"context"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func GetYoutubeTitle(videoID string) string {
	if YoutubeApiKey != "" {
		title, err := api.GetYoutubeTitleFromApi(videoID, YoutubeApiKey)
		if err != nil {
			logger.ErrorLn("Failed to get YouTube title by api:", videoID)
		} else {
			return title
		}
	}

	title, err := local_executables.GetVideoTitleWithYtDlp(utils.GetStandardYoutubeURL(videoID), context.Background())
	if err != nil {
		logger.ErrorLn("Failed to get YouTube title by yt-dlp:", videoID)
		return "YouTube " + videoID
	}

	return title
}

func GetYoutubeDuration(videoID string) (time.Duration, error) {
	if YoutubeApiKey != "" {
		duration, err := api.GetYoutubeDurationFromApi(videoID, YoutubeApiKey)
		if err != nil {
			logger.ErrorLn("Failed to get YouTube duration by api:", videoID)
		} else {
			return duration, nil
		}
	}

	return local_executables.GetDurationWithYtDlp(utils.GetStandardYoutubeURL(videoID), context.Background())
}
