package playlist

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type PlayList struct {
	Items []*song.StatefulSong

	RoomName  string
	RoomBrand string

	shutdown atomic.Bool

	bulk  bool
	dirty bool

	// event
	em       *utils.EventManager[ChangeType]
	activeEm *utils.EventManager[*song.StatefulSong]

	// lightweight locks
	ItemsLock sync.RWMutex
}

var currentPlaylist *PlayList
var temporaryItem *song.StatefulSong

func newPlayList() *PlayList {
	return &PlayList{
		Items: make([]*song.StatefulSong, 0),

		em:       utils.NewEventManager[ChangeType](),
		activeEm: utils.NewEventManager[*song.StatefulSong](),
	}
}

func (pl *PlayList) StopAll() {
	if !pl.shutdown.CompareAndSwap(false, true) {
		return
	}

	items := pl.GetItemsSnapshot()
	for _, item := range items {
		item.Destroy()
	}

	pl.notifyChange(Stopped)
}

func (pl *PlayList) SearchByInternalId(id string) *song.StatefulSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.SongId() == id {
			return item
		}
	}
	return nil
}

func (pl *PlayList) SearchByUrl(url string) *song.StatefulSong {
	return pl.SearchByInternalId(raw_song.GetInternalIdByUrl(url))
}

func (pl *PlayList) SyncWithTime(url string, now time.Duration) bool {
	if pl.shutdown.Load() {
		return false
	}

	item := pl.SearchByUrl(url)
	if item != nil {
		item.StateMachine().PlaySongAndSync(now)
		return true
	}
	return false
}

func (pl *PlayList) MarkPlayingWithoutSync(url string) bool {
	if pl.shutdown.Load() {
		return false
	}

	item := pl.SearchByUrl(url)
	if item != nil {
		item.StateMachine().PlaySong()
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

func MarkURLPlayingWithoutSync(url string) bool {
	if currentPlaylist == nil {
		return false
	}
	return currentPlaylist.MarkPlayingWithoutSync(url)
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

	currentPlaylist = newPlayList()
	currentPlaylist.RoomName = roomName
	currentPlaylist.updateRoomBrand()
	notifyNewList(currentPlaylist)
}

func GetCurrentPlaylist() *PlayList {
	return currentPlaylist
}
