package raw_song

import (
	"fmt"
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
	Name     string
	Url      string
	UniqueId string
}

func NewCustomSong(url string) CustomSong {
	if id, isYoutube := utils.CheckYoutubeURL(url); isYoutube {
		return CustomSong{
			Name:     "Youtube " + id,
			Url:      url,
			UniqueId: fmt.Sprintf("yt_%s", id),
		}
	}
	if id, isBiliBili := utils.CheckBiliURL(url); isBiliBili {
		return CustomSong{
			Name:     "BiliBili " + id,
			Url:      url,
			UniqueId: fmt.Sprintf("bili_%s", id),
		}
	}
	uniqueIdIncrement++
	return CustomSong{
		Name:     "Custom " + url,
		Url:      url,
		UniqueId: fmt.Sprintf("custom_%d", uniqueIdIncrement),
	}
}

func (cs *CustomSong) MatchUrl(url string) bool {
	if id, isYoutube := utils.CheckYoutubeURL(url); isYoutube {
		return cs.UniqueId == fmt.Sprintf("yt_%s", id)
	}
	if id, isBili := utils.CheckBiliURL(url); isBili {
		return cs.UniqueId == fmt.Sprintf("bili_%s", id)
	}
	return cs.Url == url
}

func (m *CustomSongs) Find(url string) (*CustomSong, bool) {
	m.Lock()
	defer m.Unlock()

	song, ok := m.songs[findCustomKey(url)]
	return song, ok
}

func (m *CustomSongs) FindOrCreate(url string) *CustomSong {
	m.Lock()
	defer m.Unlock()

	key := findCustomKey(url)
	if song, ok := m.songs[key]; ok {
		return song
	}
	song := NewCustomSong(url)
	m.songs[key] = &song
	return &song
}

func FindOrCreateCustomSong(url string) *CustomSong {
	if customSongs == nil {
		customSongs = &CustomSongs{
			songs: make(map[string]*CustomSong),
		}
	}
	return customSongs.FindOrCreate(url)
}

func findCustomKey(url string) string {
	if id, ok := utils.GetIdFromCustomUrl(url); ok {
		return id
	}
	return url
}
