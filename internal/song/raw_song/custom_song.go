package raw_song

import (
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"strings"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type CustomSongs struct {
	sync.Mutex
	songs map[string]*CustomSong
}

var customSongs *CustomSongs
var uniqueIdIncrement = 0

type CustomSong struct {
	Name         string
	Url          string
	UniqueId     string
	ThumbnailUrl string
}

func NewCustomSong(title, url string) CustomSong {
	if id, isYoutube := utils.CheckYoutubeURL(url); isYoutube {
		if title == "" || strings.Contains(title, id) {
			title = third_party_api.GetYoutubeTitle(id)
		}
		return CustomSong{
			Name:         title,
			Url:          url,
			UniqueId:     fmt.Sprintf("yt_%s", id),
			ThumbnailUrl: utils.GetYoutubeHQThumbnailURL(id),
		}
	}
	if id, isBiliBili := utils.CheckBiliURL(url); isBiliBili {
		return CustomSong{
			Name:     title,
			Url:      url,
			UniqueId: fmt.Sprintf("bili_%s", id),
		}
	}
	uniqueIdIncrement++
	return CustomSong{
		Name:         title,
		Url:          url,
		UniqueId:     fmt.Sprintf("custom_%d", uniqueIdIncrement),
		ThumbnailUrl: "",
	}
}

func (cs *CustomSong) MatchUrl(url string) bool {
	if id, isYoutube := utils.CheckYoutubeURL(url); isYoutube {
		return cs.UniqueId == fmt.Sprintf("yt_%s", id)
	}
	return cs.Url == url
}

func (m *CustomSongs) Find(url string) (*CustomSong, bool) {
	m.Lock()
	defer m.Unlock()

	song, ok := m.songs[findCustomKey(url)]
	return song, ok
}

func (m *CustomSongs) FindOrCreate(title, url string) *CustomSong {
	m.Lock()
	defer m.Unlock()

	key := findCustomKey(title)
	if song, ok := m.songs[key]; ok {
		return song
	}
	song := NewCustomSong(title, url)
	m.songs[key] = &song
	return &song
}

func FindOrCreateCustomSong(title, url string) *CustomSong {
	if customSongs == nil {
		customSongs = &CustomSongs{
			songs: make(map[string]*CustomSong),
		}
	}
	return customSongs.FindOrCreate(title, url)
}

func findCustomKey(url string) string {
	if id, ok := utils.GetIdFromCustomUrl(url); ok {
		return id
	}
	return url
}
