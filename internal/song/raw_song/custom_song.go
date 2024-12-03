package raw_song

import (
	"fmt"

	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

var customSongMap map[string]CustomSong
var uniqueIdIncrement int = 0

type CustomSong struct {
	Name         string
	Url          string
	UniqueId     string
	ThumbnailUrl string
}

func NewCustomSong(title, url string) *CustomSong {
	if id, isYoutube := utils.CheckYoutubeURL(url); isYoutube {
		return &CustomSong{
			Name:         title,
			Url:          url,
			UniqueId:     fmt.Sprintf("yt_%s", id),
			ThumbnailUrl: utils.GetYoutubeHQThumbnailURL(id),
		}
	}
	uniqueIdIncrement++
	return &CustomSong{
		Name:         title,
		Url:          url,
		UniqueId:     fmt.Sprintf("custom_%d", uniqueIdIncrement),
		ThumbnailUrl: "",
	}
}

func FindCustomSong(url string) (*CustomSong, bool) {
	song, ok := customSongMap[url]
	return &song, ok
}
