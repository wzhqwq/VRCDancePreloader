package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func GetPyPyVideoUrl(id int) string {
	return fmt.Sprintf("https://api.pypy.dance/video?id=%d", id)
}
func GetPyPyThumbnailUrl(id int) string {
	return fmt.Sprintf("https://api.pypy.dance/thumb?id=%d", id)
}

func CheckPyPyUrl(url string) (int, bool) {
	// http://api.pypy.dance/video?id=VIDEO_ID

	if len(url) < 31 {
		return 0, false
	}

	var id int
	_, err := fmt.Sscanf(url, "http://api.pypy.dance/video?id=%d", &id)
	if err == nil {
		return id, true
	}

	return 0, false
}

func CheckPyPyResource(url string) bool {
	return strings.Contains(url, "api.pypy.dance")
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
