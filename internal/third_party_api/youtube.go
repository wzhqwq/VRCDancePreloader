package third_party_api

import (
	"context"
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"google.golang.org/api/youtube/v3"
	"time"
)

func GetYoutubeTitle(videoID string) string {
	if youtubeApiKey == "" {
		return fmt.Sprintf("Youtube %s", videoID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	svc, err := youtube.NewService(ctx, requesting.WithYoutubeApiClient(youtubeApiKey))
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
