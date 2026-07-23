package playlist

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/watcher/queue"
)

func (pl *PlayList) GetItemsSnapshot() []*song.StatefulSong {
	pl.ItemsLock.RLock()
	items := make([]*song.StatefulSong, len(pl.Items))
	copy(items, pl.Items)
	pl.ItemsLock.RUnlock()

	return items
}

// Update must be in the watcher routine
func (pl *PlayList) Update(items []*song.StatefulSong) {
	if pl.shutdown.Load() {
		return
	}

	var changedActiveItem *song.StatefulSong

	pl.ItemsLock.Lock()
	if len(items) > 0 {
		if len(pl.Items) == 0 {
			changedActiveItem = items[0]
		} else if !pl.Items[0].Match(items[0]) {
			changedActiveItem = items[0]
		}
	}
	pl.Items = items
	pl.ItemsLock.Unlock()

	pl.notifyChange(ItemsChange)
	if changedActiveItem != nil {
		pl.notifyActiveSongChange(changedActiveItem)
	}
}

func CreateFromQueueItem(item queue.Item) *song.StatefulSong {
	var newSong *song.StatefulSong

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

	id := newSong.SongId()
	if id != "unknown" {
		persistence.GetLocalSongs().AddLocalSongIfNotExist(id, newSong.GetInfo().Title)
	}

	return newSong
}

func UpdateQueue(items []*song.StatefulSong) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.Update(items)
}

// ClearAndSetQueue must be in the watcher routine
func ClearAndSetQueue(items []queue.Item) {
	if currentPlaylist == nil {
		return
	}

	if len(currentPlaylist.Items) == 1 && len(items) == 1 {
		// better experience for random play
		currentPlaylist.Items[0].RemoveFromList()
		currentPlaylist.Update([]*song.StatefulSong{CreateFromQueueItem(items[0])})
		return
	}

	if len(currentPlaylist.Items) > 0 {
		resetPlaylist(currentPlaylist.RoomName)
		logger.InfoLn("New playlist")
	}

	if len(items) > 0 {
		list := lo.Map(items, func(item queue.Item, _ int) *song.StatefulSong {
			return CreateFromQueueItem(item)
		})
		currentPlaylist.Update(list)
		logger.InfoLn("Started playlist")
	}
}

// EnterNewRoom must be in the watcher routine
func EnterNewRoom(roomName string) {
	if currentPlaylist == nil {
		return
	}

	logger.InfoLn("Entering new room", roomName)

	if len(currentPlaylist.Items) > 0 {
		resetPlaylist(roomName)
	} else {
		updateRoomName(roomName)
	}
}

// GetQueue must be in the watcher routine
func GetQueue() []*song.StatefulSong {
	if currentPlaylist == nil {
		return nil
	}
	return currentPlaylist.Items
}
