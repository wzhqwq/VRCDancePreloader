package internal_id

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var parsingLogger = utils.NewLogger("Parsing")

var numericIdRegex = regexp.MustCompile("[0-9]+")

func GetIdByUrl(url string) (string, bool) {
	if id, isYoutube := CheckYoutubeURL(url); isYoutube {
		return YtInternalPrefix + id, true
	}
	if id, isBiliBili := CheckBiliURL(url); isBiliBili {
		return BiliInternalPrefix + id, true
	}
	if id, isPyPy := CheckPyPyUrl(url); isPyPy {
		return PyPyInternalPrefix + strconv.Itoa(id), true
	}
	if id, isWanna := CheckWannaUrl(url); isWanna {
		return WannaInternalPrefix + strconv.Itoa(id), true
	}
	if id, isDuDu := CheckDuDuUrl(url); isDuDu {
		return DuDuInternalPrefix + strconv.Itoa(id), true
	}
	return "", false
}

func GetUrlById(internalId string) (string, bool) {
	if id, isYoutube := CheckIdIsYoutube(internalId); isYoutube {
		return GetStandardYoutubeURL(id), true
	}
	if id, isBiliBili := CheckIdIsBili(internalId); isBiliBili {
		return GetStandardBiliURL(id), true
	}
	if id, isPyPy := CheckIdIsPyPy(internalId); isPyPy {
		return GetPyPyVideoUrl(id), true
	}
	if id, isWanna := CheckIdIsWanna(internalId); isWanna {
		return GetWannaVideoUrl(id), true
	}
	if id, isDuDu := CheckIdIsDuDu(internalId); isDuDu {
		return GetDuDuVideoUrl(id), true
	}
	return "", false
}

func GetPlatformNameByInternalId(internalId string) string {
	if _, ok := CheckIdIsPyPy(internalId); ok {
		return "PyPyDance"
	}
	if _, ok := CheckIdIsWanna(internalId); ok {
		return "WannaDance"
	}
	if _, ok := CheckIdIsDuDu(internalId); ok {
		return "DuDuFitDance"
	}
	if _, ok := CheckIdIsBili(internalId); ok {
		return "BiliBili"
	}
	if _, ok := CheckIdIsYoutube(internalId); ok {
		return "YouTube"
	}
	return ""
}

const UrlBasedInternalPrefix = "url_"

func CheckIdIsUrlBased(id string) (int, bool) {
	cut, ok := strings.CutPrefix(id, UrlBasedInternalPrefix)
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
