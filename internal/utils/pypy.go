package utils

import "fmt"

func GetPyPyVideoUrl(id int) string {
	return fmt.Sprintf("http://jd.pypy.moe/api/v1/videos/%d.mp4", id)
}
func GetPyPyThumbnailUrl(id int) string {
	return fmt.Sprintf("http://jd.pypy.moe/api/v1/thumbnails/%d.jpg", id)
}
