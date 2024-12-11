package playlist

import (
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type PlayList struct {
	sync.Mutex
	Items []*song.PreloadedSong

	criticalUpdateCh chan struct{}
	maxPreload       int

	// event
	em EventManager
}

var currentPlaylist *PlayList
var temporaryItem *song.PreloadedSong
var mutatingMutex = &sync.Mutex{}

func newPlayList(maxPreload int) *PlayList {
	return &PlayList{
		Items:            make([]*song.PreloadedSong, 0),
		criticalUpdateCh: make(chan struct{}, 1),
		maxPreload:       maxPreload,
	}
}

func Init(maxPreload int) {
	currentPlaylist = newPlayList(maxPreload)
	currentPlaylist.Start()
}

func (pl *PlayList) Start() {
	go func() {
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
	pl.Lock()
	defer pl.Unlock()

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
