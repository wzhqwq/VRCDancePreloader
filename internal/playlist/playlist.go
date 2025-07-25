package playlist

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"sync"
	"time"
)

type PlayList struct {
	Items []*song.PreloadedSong

	RoomName  string
	RoomBrand string

	criticalUpdateCh chan struct{}
	maxPreload       int

	started bool
	stopped bool

	bulk  bool
	dirty bool

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

	go pl.preloadLoop()
}

func (pl *PlayList) StopAll() {
	if pl.stopped {
		return
	}
	pl.stopped = true

	items := pl.GetItemsSnapshot()
	download.CancelDownload(
		lo.Map(items, func(item *song.PreloadedSong, _ int) string {
			return item.GetId()
		})...,
	)
	for _, item := range items {
		item.RemoveFromList()
	}

	pl.notifyChange(Stopped)
	pl.CriticalUpdate()
}

func (pl *PlayList) SyncWithTime(url string, now time.Duration) bool {
	if pl.stopped {
		return false
	}

	var item *song.PreloadedSong
	if id, ok := utils.CheckPyPyUrl(url); ok {
		item = pl.FindPyPySong(id)
	} else if id, ok = utils.CheckWannaUrl(url); ok {
		item = pl.FindWannaSong(id)
	} else {
		item = pl.FindCustomSong(url)
	}
	if item != nil {
		item.PlaySongStartFrom(now)
		return true
	}
	return false
}

func (pl *PlayList) updateRoomBrand() {
	if brand := utils.IdentifyRoomBrand(pl.RoomName); brand != "" {
		pl.RoomBrand = brand
	} else {
		pl.RoomBrand = i18n.T("placeholder_room_not_supported")
	}
}

func MarkURLPlaying(url string, now time.Duration) bool {
	if currentPlaylist == nil {
		return false
	}
	return currentPlaylist.SyncWithTime(url, now)
}

func updateRoomName(roomName string) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.RoomName = roomName
	currentPlaylist.updateRoomBrand()
	currentPlaylist.notifyChange(RoomChange)
}

func resetPlaylist(roomName string) {
	currentPlaylist.StopAll()

	currentPlaylist = newPlayList(currentPlaylist.maxPreload)
	currentPlaylist.RoomName = roomName
	currentPlaylist.updateRoomBrand()
	notifyNewList(currentPlaylist)
}

func GetCurrentPlaylist() *PlayList {
	return currentPlaylist
}
