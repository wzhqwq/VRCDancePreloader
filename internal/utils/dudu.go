package utils

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var duDuVideoURLRegex = regexp.MustCompile(`videos/(\d+)`)
var duDuVideoPathRegex = regexp.MustCompile(`videos/(\d+)`)

func GetDuDuVideoUrl(id int) string {
	return fmt.Sprintf("https://api.dudufit.dance/api/v1/videos/%d?cdn=jpn", id)
}

func GetDuDuThumbnailUrl(id int) string {
	return fmt.Sprintf("https://api.dudufit.dance/thumbnails/%d.jpg", id)
}

func CheckDuDuUrl(url string) (int, bool) {
	if matches := duDuVideoURLRegex.FindStringSubmatch(url); len(matches) > 1 {
		id := matches[1]
		num, err := strconv.Atoi(id)
		if err != nil {
			log.Println("Invalid DuDuFitDance video id:", id)
			return 0, false
		}
		return num, true
	}

	return 0, false
}

func GetDuDuListUrl() string {
	return "https://api.dudufit.dance/api/v1/videos?cdn=jpn"
}

func CheckDuDuRequest(req *http.Request) (string, bool) {
	if matches := duDuVideoPathRegex.FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		return matches[1], true
	}
	return "", false
}

func CheckIdIsDuDu(id string) (int, bool) {
	if !strings.Contains(id, "dudu_") {
		return 0, false
	}

	num, err := strconv.Atoi(id[5:])
	if err != nil {
		return 0, false
	}

	return num, true
}
