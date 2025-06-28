package third_party_api

import (
	"encoding/json"
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"net/http"
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

func requestBiliApi[T any](client *http.Client, url string) (*T, error) {
	res, err := client.Get(url)
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

func GetBvInfo(client *http.Client, bvID string) (*BvInfo, error) {
	return requestBiliApi[BvInfo](client, utils.GetBiliVideoInfoURL(bvID))
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

func GetBiliVideoUrl(client *http.Client, bvID string) (string, error) {
	info, err := GetBvInfo(client, bvID)
	if err != nil {
		return "", err
	}

	playerInfo, err := requestBiliApi[BiliPlayerInfo](client, utils.GetBiliVideoPlayerURL(bvID, info.Cid))
	if err != nil {
		return "", err
	}

	return playerInfo.Segments[0].URL, nil
}

func GetBiliVideoTitle(client *http.Client, bvID string) (string, error) {
	info, err := GetBvInfo(client, bvID)
	if err != nil {
		return "", err
	}

	return info.Title, nil
}

func GetBiliVideoThumbnail(client *http.Client, bvID string) (string, error) {
	info, err := GetBvInfo(client, bvID)
	if err != nil {
		return "", err
	}

	return info.Pic, nil
}

func GetBiliVideoDuration(client *http.Client, bvID string) (int, error) {
	info, err := GetBvInfo(client, bvID)
	if err != nil {
		return 0, err
	}

	return info.Duration, nil
}
