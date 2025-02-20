package playlist

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"sync"
)

type PlayList struct {
	sync.Mutex
	//utils.LoggingMutex

	Items []*song.PreloadedSong

	criticalUpdateCh chan struct{}
	maxPreload       int

	// event
	em *EventManager
}

var currentPlaylist *PlayList
var temporaryItem *song.PreloadedSong

func newPlayList(maxPreload int) *PlayList {
	return &PlayList{
		Items:            make([]*song.PreloadedSong, 0),
		criticalUpdateCh: make(chan struct{}, 1),
		maxPreload:       maxPreload,
		em:               NewEventManager(),
	}
}

func (pl *PlayList) Start() {
	go func() {
		pl.Preload()
		for {
			<-pl.criticalUpdateCh
			pl.Preload()
		}
	}()
}

func (pl *PlayList) StopAll() {
	pl.Lock()
	defer pl.Unlock()

	for _, item := range pl.Items {
		item.RemoveFromList()
	}
	pl.Items = make([]*song.PreloadedSong, 0)
	pl.notifyChange(ItemsChange)
}

func (pl *PlayList) SyncWithTime(url string, now float64) {
	var item *song.PreloadedSong
	if id, ok := utils.CheckPyPyUrl(url); ok {
		item = pl.findPyPySong(id)
	} else {
		item = pl.findCustomSong(url)
	}
	item.PlaySongStartFrom(now)
}

func MarkURLPlaying(url string, now float64) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.SyncWithTime(url, now)
}
