package internal_id

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var wannaVideoURLRegex = regexp.MustCompile(`play\?id=(\d+)`)

const wannaVideoPath = "/Api/Songs/play"
const WannaInternalPrefix = "wanna_"

func GetWannaVideoUrl(id int) string {
	return fmt.Sprintf("http://api.udon.dance/Api/Songs/play?id=%d", id)
}

func GetWannaThumbnailUrl(id int) string {
	return fmt.Sprintf("https://aya.kiva.moe/images/%d.jpg", id)
}

func CheckWannaUrl(url string) (int, bool) {
	if matches := wannaVideoURLRegex.FindStringSubmatch(url); len(matches) > 1 {
		id := matches[1]
		num, err := strconv.Atoi(id)
		if err != nil {
			parsingLogger.ErrorLn("Invalid WannaDance video id:", id)
			return 0, false
		}
		return num, true
	}

	return 0, false
}

func GetWannaListUrl() string {
	return "https://api.udon.dance/Api/Songs/list"
}

func CheckWannaRequest(req *http.Request) (string, bool) {
	if req.URL.Path == wannaVideoPath {
		id := req.URL.Query().Get("id")
		if !numericIdRegex.MatchString(id) {
			return "", false
		}
		return WannaInternalPrefix + id, true
	}
	return "", false
}

func CheckIdIsWanna(id string) (int, bool) {
	cut, ok := strings.CutPrefix(id, WannaInternalPrefix)
	if !ok {
		return 0, false
	}

	if !numericIdRegex.MatchString(cut) {
		return 0, false
	}

	num, err := strconv.Atoi(cut)
	if err != nil {
		return 0, false
	}

	return num, true
}
