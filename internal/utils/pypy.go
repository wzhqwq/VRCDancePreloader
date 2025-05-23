package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func GetPyPyVideoUrl(id int) string {
	return fmt.Sprintf("http://jd.pypy.moe/api/v1/videos/%d.mp4", id)
}
func GetPyPyThumbnailUrl(id int) string {
	return fmt.Sprintf("http://jd.pypy.moe/api/v1/thumbnails/%d.jpg", id)
}

func CheckPyPyUrl(url string) (int, bool) {
	// jd.pypy.moe/api/v1/videos/VIDEO_ID.mp4

	if len(url) < 11 {
		return 0, false
	}

	var id int
	_, err := fmt.Sscanf(url, "http://jd.pypy.moe/api/v1/videos/%d.mp4", &id)
	if err == nil {
		return id, true
	}

	return 0, false
}

func CheckPyPyThumbnailUrl(url string) bool {
	return strings.Contains(url, "jd.pypy.moe")
}

func CheckIdIsPyPy(id string) (int, bool) {
	if !strings.Contains(id, "pypy_") {
		return 0, false
	}

	num, err := strconv.Atoi(strings.Split(id, "pypy_")[1])
	if err != nil {
		return 0, false
	}

	return num, true
}
