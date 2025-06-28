package third_party_api

import (
	"context"
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"google.golang.org/api/youtube/v3"
	"log"
	"time"
)

func GetYoutubeTitleFromApi(videoID string) (string, error) {
	if YoutubeApiKey == "" {
		return "", errors.New("empty Youtube API key")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	svc, err := youtube.NewService(ctx, requesting.WithYoutubeApiClient(YoutubeApiKey))
	if err != nil {
		return "", err
	}

	call := svc.Videos.List([]string{"snippet"}).Id(videoID)
	resp, err := call.Do()
	if err != nil {
		return "", err
	}

	if len(resp.Items) == 0 {
		return "", errors.New("video not found")
	}

	return resp.Items[0].Snippet.Title, nil

}

func GetYoutubeTitle(videoID string) string {
	title, err := GetYoutubeTitleFromApi(videoID)
	if err != nil {
		log.Println("Failed to get youtube title by api: " + videoID)
		return "YouTube " + videoID
	}
	return title
}
