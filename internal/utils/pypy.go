package utils

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var pypyVideoURLRegex = regexp.MustCompile(`videos/(\d+)\.mp4|video\?id=(\d+)`)
var pypyVideoLegacyPathRegex = regexp.MustCompile(`videos/(\d+)\.mp4`)
var pypyVideoNewPath = "/video"

func GetPyPyVideoUrl(id int) string {
	return fmt.Sprintf("http://api.pypy.dance/video?id=%d", id)
}
func GetPyPyThumbnailUrl(id int) string {
	return fmt.Sprintf("https://api.pypy.dance/thumb?id=%d", id)
}

func CheckPyPyUrl(url string) (int, bool) {
	// http://api.pypy.dance/video?id=VIDEO_ID
	// http://jd.pypy.moe/api/v1/videos/VIDEO_ID.mp4
	if matches := pypyVideoURLRegex.FindStringSubmatch(url); len(matches) > 1 {
		id := matches[1]
		if id == "" && len(matches) > 2 {
			id = matches[2]
		}
		num, err := strconv.Atoi(id)
		if err != nil {
			log.Println("Invalid pypy video id:", id)
			return 0, false
		}
		return num, true
	}

	return 0, false
}

func CheckPyPyRequest(req *http.Request) (string, bool) {
	if matches := pypyVideoLegacyPathRegex.FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		return matches[1], true
	}
	if req.URL.Path == pypyVideoNewPath {
		id := req.URL.Query().Get("id")
		if !numericIdRegex.MatchString(id) {
			return "", false
		}
		return id, true
	}
	return "", false
}

func CheckPyPyResource(url string) bool {
	return strings.Contains(url, "api.pypy.dance") || strings.Contains(url, "jd.pypy.moe")
}

func CheckIdIsPyPy(id string) (int, bool) {
	if !strings.Contains(id, "pypy_") {
		return 0, false
	}

	num, err := strconv.Atoi(id[5:])
	if err != nil {
		return 0, false
	}

	return num, true
}
