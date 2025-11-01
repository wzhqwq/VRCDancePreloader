package utils

import "fmt"

func GetIdFromCustomUrl(url string) (string, bool) {
	if id, isYoutube := CheckYoutubeURL(url); isYoutube {
		return id, true
	}
	if id, isBiliBili := CheckBiliURL(url); isBiliBili {
		return id, true
	}
	return "", false
}

func GetIdFromUrl(url string) (string, bool) {
	if id, isPyPy := CheckIdIsPyPy(url); isPyPy {
		return fmt.Sprintf("pypy_%d", id), true
	}
	if id, isWanna := CheckIdIsWanna(url); isWanna {
		return fmt.Sprintf("wanna_%d", id), true
	}
	if id, isYoutube := CheckYoutubeURL(url); isYoutube {
		return "yt_" + id, true
	}
	if id, isBiliBili := CheckBiliURL(url); isBiliBili {
		return "bili_" + id, true
	}
	return "", false
}
