package playlist

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"sync"
)

type PlayList struct {
	Items []*song.PreloadedSong

	RoomName string

	criticalUpdateCh chan struct{}
	maxPreload       int

	started bool
	stopped bool

	// event
	em *utils.EventManager[ChangeType]

	// lightweight locks
	ItemsLock sync.RWMutex
}

var currentPlaylist *PlayList
var temporaryItem *song.PreloadedSong

func newPlayList(maxPreload int) *PlayList {
	return &PlayList{
		Items:            make([]*song.PreloadedSong, 0),
		criticalUpdateCh: make(chan struct{}, 1),
		maxPreload:       maxPreload,
		em:               utils.NewEventManager[ChangeType](),
	}
}

func (pl *PlayList) Start() {
	if pl.started {
		return
	}
	pl.started = true

	go func() {
		pl.Preload()
		for {
			<-pl.criticalUpdateCh
			if pl.stopped {
				return
			}
			pl.Preload()
		}
	}()
}

func (pl *PlayList) StopAll() {
	if pl.stopped {
		return
	}
	pl.stopped = true

	items := pl.GetItemsSnapshot()
	for _, item := range items {
		item.RemoveFromList()
	}

	pl.notifyChange(Stopped)
	pl.CriticalUpdate()
}

func (pl *PlayList) SyncWithTime(url string, now float64) {
	if pl.stopped {
		return
	}

	var item *song.PreloadedSong
	if id, ok := utils.CheckPyPyUrl(url); ok {
		item = pl.FindPyPySong(id)
	} else {
		item = pl.FindCustomSong(url)
	}
	if item != nil {
		item.PlaySongStartFrom(now)
	}
}

func MarkURLPlaying(url string, now float64) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.SyncWithTime(url, now)
}

func UpdateRoomName(roomName string) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.RoomName = roomName
	currentPlaylist.notifyChange(RoomChange)
}

func GetCurrentPlaylist() *PlayList {
	return currentPlaylist
}
