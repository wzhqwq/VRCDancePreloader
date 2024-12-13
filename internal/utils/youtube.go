package utils

import (
	"context"
	"fmt"
	"regexp"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var youtubeApiKey string

func SetYoutubeApiKey(key string) {
	youtubeApiKey = key
}

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

func GetYoutubeTitle(videoID string) string {
	if youtubeApiKey == "" {
		return fmt.Sprintf("Youtube %s", videoID)
	}
	svc, err := youtube.NewService(context.Background(), option.WithAPIKey(youtubeApiKey))
	if err != nil {
		return fmt.Sprintf("Youtube %s", videoID)
	}
	call := svc.Videos.List([]string{"snippet"}).Id(videoID)
	resp, err := call.Do()
	if err != nil {
		fmt.Println("Youtube API Error: ", err)
		return fmt.Sprintf("Youtube %s", videoID)
	}
	if len(resp.Items) == 0 {
		fmt.Println("No items")
		return fmt.Sprintf("Youtube %s", videoID)
	}
	return resp.Items[0].Snippet.Title
}
