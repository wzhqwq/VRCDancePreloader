package utils

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var wannaVideoURLRegex = regexp.MustCompile(`play\?id=(\d+)`)
var wannaVideoPath = "/Api/Songs/play"

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
		return id, true
	}
	return "", false
}

func CheckIdIsWanna(id string) (int, bool) {
	if !strings.Contains(id, "wanna_") {
		return 0, false
	}

	num, err := strconv.Atoi(id[6:])
	if err != nil {
		return 0, false
	}

	return num, true
}
