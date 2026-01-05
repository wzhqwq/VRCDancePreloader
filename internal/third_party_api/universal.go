package third_party_api

import (
	"strings"
	"time"

	"github.com/stephennancekivell/go-future/future"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Third Party API")

func GetThumbnailByInternalID(id string) future.Future[string] {
	if pypyId, isPypy := utils.CheckIdIsPyPy(id); isPypy {
		return future.Pure(utils.GetPyPyThumbnailUrl(pypyId))
	}
	if wannaId, isWanna := utils.CheckIdIsWanna(id); isWanna {
		song, ok := raw_song.FindWannaSong(wannaId)
		if ok {
			return future.Pure("group:" + song.Group)
		}
		return future.Pure("")
	}
	if duduId, isDuDu := utils.CheckIdIsDuDu(id); isDuDu {
		song, ok := raw_song.FindDuDuSong(duduId)
		if ok {
			return future.Pure("group:" + song.Group)
		}
		return future.Pure("")
	}
	if ytId, isYoutube := utils.CheckIdIsYoutube(id); isYoutube {
		if EnableYoutubeThumbnail {
			return future.Pure(utils.GetYoutubeMQThumbnailURL(ytId))
		}
	}
	if bvId, isBiliBili := utils.CheckIdIsBili(id); isBiliBili {
		return future.New(func() string {
			url, err := GetBiliVideoThumbnail(bvId)
			if err != nil {
				logger.ErrorLn("Error while getting BiliBili video thumbnail:", err)
				return ""
			}
			return url
		})
	}
	return future.Pure("")
}

func CompleteTitleByInternalID(id, title string) future.Future[string] {
	if _, isPypy := utils.CheckIdIsPyPy(id); isPypy {
		return future.Pure(title)
	}
	if _, isWanna := utils.CheckIdIsWanna(id); isWanna {
		return future.Pure(title)
	}
	if ytId, isYoutube := utils.CheckIdIsYoutube(id); isYoutube {
		if strings.Contains(title, ytId) && EnableYoutubeApi {
			return future.New(func() string {
				return GetYoutubeTitle(ytId)
			})
		}
		return future.Pure(title)
	}
	if bvId, isBiliBili := utils.CheckIdIsBili(id); isBiliBili {
		if strings.Contains(title, bvId) {
			return future.New(func() string {
				return GetBiliVideoTitle(bvId)
			})
		}
		return future.Pure(title)
	}
	return future.Pure(title)
}

func GetDurationByInternalID(id string) future.Future[time.Duration] {
	if ytId, isYoutube := utils.CheckIdIsYoutube(id); isYoutube {
		if EnableYoutubeApi {
			return future.New(func() time.Duration {
				d, err := GetYoutubeDuration(ytId)
				if err != nil {
					logger.ErrorLn("Error while getting YouTube video duration:", err)
					return 0
				}
				return d
			})
		}
		return future.Pure(time.Duration(0))
	}
	if bvId, isBiliBili := utils.CheckIdIsBili(id); isBiliBili {
		return future.New(func() time.Duration {
			d, err := GetBiliVideoDuration(bvId)
			if err != nil {
				logger.ErrorLn("Error while getting BiliBili video duration:", err)
				return 0
			}
			return time.Duration(d) * time.Second
		})
	}
	return future.Pure(time.Duration(0))
}
