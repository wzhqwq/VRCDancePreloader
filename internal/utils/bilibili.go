package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Thanks to https://github.com/gizmo-ds/bilibili-real-url and https://github.com/SocialSisterYi/bilibili-API-collect
// for the share of the usage of BiliBili api

func GetStandardBiliURL(bvID string) string {
	return fmt.Sprintf("https://www.bilibili.com/video/%s", bvID)
}

func GetBiliVideoInfoURL(bvID string) string {
	return fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?bvid=%s", bvID)
}

func GetBiliVideoPlayerURL(bvID string, cid int64) string {
	params := url.Values{
		"bvid":     []string{bvID},
		"cid":      []string{fmt.Sprintf("%d", cid)},
		"platform": []string{"html5"},
	}
	return fmt.Sprintf("https://api.bilibili.com/x/player/playurl?%s", params.Encode())
}

func CheckBiliURL(url string) (string, bool) {
	// www.bilibili.com/video/bvID
	// b23.tv/bvID
	// api.xin.moe/bvID

	if len(url) < 12 {
		return "", false
	}

	matched := regexp.MustCompile(`(?:bilibili\.com/video|b23.tv|api.xin.moe)/([a-zA-Z0-9]+)`).FindStringSubmatch(url)
	if len(matched) > 1 {
		return matched[1], true
	}

	return "", false
}

func CheckIdIsBili(id string) (string, bool) {
	if !strings.Contains(id, "bili_") {
		return "", false
	}

	return id[5:], true
}
