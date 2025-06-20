package third_party_api

import (
	"github.com/stephennancekivell/go-future/future"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"strings"
)

func GetThumbnailByInternalID(id string) future.Future[string] {
	if pypyId, isPypy := utils.CheckIdIsPyPy(id); isPypy {
		return future.Pure(utils.GetPyPyThumbnailUrl(pypyId))
	}
	if _, isWanna := utils.CheckIdIsWanna(id); isWanna {
		return future.Pure("")
	}
	if ytId, isYoutube := utils.CheckIdIsYoutube(id); isYoutube {
		return future.Pure(utils.GetYoutubeMQThumbnailURL(ytId))
	}
	if bvId, isBiliBili := utils.CheckIdIsBili(id); isBiliBili {
		return future.New(func() string {
			url, err := GetBiliVideoThumbnail(requesting.GetBiliClient(), bvId)
			if err != nil {
				log.Println("error while getting bilibili video thumbnail:", err)
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
		if strings.Contains(title, ytId) {
			return future.New(func() string {
				return GetYoutubeTitle(ytId)
			})
		}
		return future.Pure(title)
	}
	if bvId, isBiliBili := utils.CheckIdIsBili(id); isBiliBili {
		if strings.Contains(title, bvId) {
			return future.New(func() string {
				t, err := GetBiliVideoTitle(requesting.GetBiliClient(), bvId)
				if err != nil {
					log.Println("error while getting bilibili video title:", err)
				}
				return t
			})
		}
		return future.Pure(title)
	}
	return future.Pure(title)
}
