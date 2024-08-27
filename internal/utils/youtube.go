package utils

import (
	"fmt"
	"regexp"
)

func GetYoutubeMQThumbnailURL(videoID string) string {
	return fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID)
}
func GetYoutubeHQThumbnailURL(videoID string) string {
	return fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
}

func CheckYoutubeURL(url string) (string, bool) {
	// youtube.com/watch?v=VIDEO_ID
	// youtube.com/v/VIDEO_ID
	// youtu.be/VIDEO_ID

	if len(url) < 11 {
		return "", false
	}

	matched := regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtube\.com/v/|youtu\.be/)([a-zA-Z0-9_-]{11})`).FindStringSubmatch(url)
	if len(matched) > 1 {
		return matched[1], true
	}

	return "", false
}
