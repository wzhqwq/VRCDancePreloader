package third_party_api

import (
	"context"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/api"
)

func GetBiliVideoUrl(bvID string, ctx context.Context) (string, error) {
	playerInfo, err := api.GetPlayerInfo(bvID, ctx)
	if err != nil {
		return "", err
	}

	return playerInfo.Segments[0].URL, nil
}

func GetBiliVideoModTime(bvID string, ctx context.Context) (time.Time, error) {
	info, err := api.GetBvInfo(bvID, ctx)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(int64(info.Pages[0].CreatedTime), 0), nil
}

func GetBiliVideoTitle(bvID string) string {
	info, err := api.GetBvInfo(bvID, context.Background())
	if err != nil {
		logger.ErrorLn("Error while getting BiliBili video title:", err)
		return "BiliBili " + bvID
	}

	return info.Title
}

func GetBiliVideoThumbnail(bvID string) (string, error) {
	info, err := api.GetBvInfo(bvID, context.Background())
	if err != nil {
		return "", err
	}

	return info.Pic, nil
}

func GetBiliVideoDuration(bvID string) (int, error) {
	info, err := api.GetBvInfo(bvID, context.Background())
	if err != nil {
		return 0, err
	}

	return info.Pages[0].Duration, nil
}
