package playlist

import (
	"log"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
)

func (pl *PlayList) GetItemsSnapshot() []*song.PreloadedSong {
	pl.ItemsLock.RLock()
	items := make([]*song.PreloadedSong, len(pl.Items))
	copy(items, pl.Items)
	pl.ItemsLock.RUnlock()

	return items
}

// Update must be in the watcher routine
func (pl *PlayList) Update(items []*song.PreloadedSong) {
	if pl.stopped {
		return
	}

	pl.ItemsLock.Lock()
	pl.Items = items
	pl.ItemsLock.Unlock()

	pl.notifyChange(ItemsChange)
	pl.CriticalUpdate()
}

func CreateFromQueueItem(item queue.QueueItem) *song.PreloadedSong {
	var newSong *song.PreloadedSong

	if temporaryItem != nil && item.MatchWithPreloaded(temporaryItem) {
		newSong = temporaryItem
		temporaryItem = nil
	} else {
		newSong = item.ToPreloaded()
	}
	if newSong == nil {
		newSong = song.CreateUnknownSong()
	}

	newSong.Adder = item.GetAdder()

	id := newSong.GetSongId()
	if id != "unknown" && id != "random_play" {
		persistence.GetLocalSongs().AddLocalSongIfNotExist(id, newSong.GetInfo().Title)
	}

	return newSong
}

func UpdateQueue(items []*song.PreloadedSong) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.Update(items)
}

// ClearAndSetQueue must be in the watcher routine
func ClearAndSetQueue(items []queue.QueueItem) {
	if currentPlaylist == nil {
		return
	}

	if len(currentPlaylist.Items) == 1 && len(items) == 1 {
		// better experience for random play
		currentPlaylist.Items[0].RemoveFromList()
		currentPlaylist.Update([]*song.PreloadedSong{CreateFromQueueItem(items[0])})
		return
	}

	if len(currentPlaylist.Items) > 0 {
		resetPlaylist(currentPlaylist.RoomName)
		log.Println("New playlist")
	}

	if len(items) > 0 {
		list := lo.Map(items, func(item queue.QueueItem, _ int) *song.PreloadedSong {
			return CreateFromQueueItem(item)
		})
		currentPlaylist.Update(list)
		currentPlaylist.Start()
		log.Println("Started playlist")
	}
}

// EnterNewRoom must be in the watcher routine
func EnterNewRoom(roomName string) {
	if currentPlaylist == nil {
		return
	}

	if len(currentPlaylist.Items) > 0 {
		resetPlaylist(roomName)
	} else {
		updateRoomName(roomName)
	}
}

// GetQueue must be in the watcher routine
func GetQueue() []*song.PreloadedSong {
	if currentPlaylist == nil {
		return nil
	}
	return currentPlaylist.Items
}
