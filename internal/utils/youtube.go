package utils

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var youTubePathRegex = regexp.MustCompile(`/([a-zA-Z0-9_-]{11})$`)

func GetStandardYoutubeURL(videoID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
}
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

func CheckYouTubeRequest(req *http.Request) (string, bool) {
	// for v=
	id := req.URL.Query().Get("v")
	if id != "" {
		return id, true
	}
	// for path

	if matched := youTubePathRegex.FindStringSubmatch(req.URL.Path); len(matched) > 1 {
		return matched[1], true
	}
	return "", false
}

func CheckYoutubeThumbnailURL(url string) bool {
	return strings.Contains(url, "i.ytimg.com")
}

func CheckIdIsYoutube(id string) (string, bool) {
	if !strings.Contains(id, "yt_") {
		return "", false
	}

	return id[3:], true
}
