package third_party_api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type BiliApiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`

	Data T `json:"data"`
}

type BvInfo struct {
	Bvid     string `json:"bvid"`
	Videos   int    `json:"videos"`
	Pic      string `json:"pic"`
	Title    string `json:"title"`
	State    int    `json:"state"`
	Duration int    `json:"duration"`
	Cid      int64  `json:"cid"`
	Pages    []struct {
		Cid        int64  `json:"cid"`
		Page       int    `json:"page"`
		Part       string `json:"part"`
		Duration   int    `json:"duration"`
		FirstFrame string `json:"first_frame"`
	} `json:"pages"`
}

func requestBiliApi[T any](url string, ctx context.Context) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := requesting.GetBiliClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("error getting video info: " + res.Status)
	}

	var resp BiliApiResponse[T]
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New("error getting video info: " + resp.Message)
	}

	return &resp.Data, nil
}

var biliVideoInfoCache = utils.NewWeakCache[*BvInfo](10)

func GetBvInfo(bvID string, ctx context.Context) (*BvInfo, error) {
	if info, ok := biliVideoInfoCache.Get(bvID); ok {
		return info, nil
	}

	info, err := requestBiliApi[BvInfo](utils.GetBiliVideoInfoURL(bvID), ctx)
	if err != nil {
		return nil, err
	}

	biliVideoInfoCache.Set(bvID, info)
	return info, nil
}

type BiliPlayerInfo struct {
	Message       string `json:"message"`
	TimeLength    int    `json:"timelength"`
	AcceptFormat  string `json:"accept_format"`
	AcceptQuality []int  `json:"accept_quality"`
	SeekParam     string `json:"seek_param"`
	SeekType      string `json:"seek_type"`
	Segments      []struct {
		Order  int    `json:"order"`
		Length int    `json:"length"`
		Size   int    `json:"size"`
		URL    string `json:"url"`
	} `json:"durl"`
}

func GetBiliVideoUrl(bvID string, ctx context.Context) (string, error) {
	info, err := GetBvInfo(bvID, ctx)
	if err != nil {
		return "", err
	}

	playerInfo, err := requestBiliApi[BiliPlayerInfo](utils.GetBiliVideoPlayerURL(bvID, info.Cid), ctx)
	if err != nil {
		return "", err
	}

	return playerInfo.Segments[0].URL, nil
}

func GetBiliVideoTitle(bvID string) string {
	info, err := GetBvInfo(bvID, context.Background())
	if err != nil {
		log.Println("error while getting bilibili video title:", err)
		return "BiliBili " + bvID
	}

	return info.Title
}

func GetBiliVideoThumbnail(bvID string) (string, error) {
	info, err := GetBvInfo(bvID, context.Background())
	if err != nil {
		return "", err
	}

	return info.Pic, nil
}

func GetBiliVideoDuration(bvID string) (int, error) {
	info, err := GetBvInfo(bvID, context.Background())
	if err != nil {
		return 0, err
	}

	return info.Duration, nil
}
