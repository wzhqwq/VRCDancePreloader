package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
)

type DuDuFitDanceVideoAssetsResp struct {
	VideoID int `json:"video_id"`

	Assets []DuDuFitDanceVideoAsset `json:"assets"`
}

type DuDuFitDanceVideoAsset struct {
	AssetType   string  `json:"asset_type"`
	OriginalURL string  `json:"original_url"`
	Framerate   float64 `json:"framerate"`
	BitrateKbps int     `json:"bitrate_kbps"`
}

func GetDuDuAssetsInfoUrl(id int) string {
	return fmt.Sprintf("https://api.dudufit.dance/api/v1/assets/%d", id)
}

const DuDuFitDanceWebsite = "https://www.dudufit.dance/zh/videos/"

type DuDuOriginalVideoInfo struct {
	OriginalURL string
	Framerate   float64
	BitrateKbps int
}

func GetDuDuOriginalVideoInfo(id int, ctx context.Context) (DuDuOriginalVideoInfo, error) {
	client := requesting.GetClient(requesting.DuDuFitDance)

	req, err := client.NewGetRequest(GetDuDuAssetsInfoUrl(id), ctx)
	if err != nil {
		return DuDuOriginalVideoInfo{}, err
	}

	requesting.SetupHeader(req, DuDuFitDanceWebsite)

	res, err := client.Do(req)
	if err != nil {
		return DuDuOriginalVideoInfo{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return DuDuOriginalVideoInfo{}, fmt.Errorf("getting assets from DuDuFitDance: %s", res.Status)
	}

	var resp DuDuFitDanceVideoAssetsResp
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return DuDuOriginalVideoInfo{}, err
	}

	asset, ok := lo.Find(resp.Assets, func(item DuDuFitDanceVideoAsset) bool {
		return item.AssetType == "Original"
	})

	if !ok {
		return DuDuOriginalVideoInfo{}, errors.New("cannot find get original video info from DuDuFitDance")
	}

	return DuDuOriginalVideoInfo{asset.OriginalURL, asset.Framerate, asset.BitrateKbps}, nil
}
