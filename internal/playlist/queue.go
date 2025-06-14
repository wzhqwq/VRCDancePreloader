package playlist

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"log"
	"slices"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

func (pl *PlayList) GetItemsSnapshot() []*song.PreloadedSong {
	pl.ItemsLock.RLock()
	items := make([]*song.PreloadedSong, len(pl.Items))
	copy(items, pl.Items)
	pl.ItemsLock.RUnlock()

	return items
}

// RemoveItem must be in the watcher routine
func (pl *PlayList) RemoveItem(index int) {
	if pl.stopped {
		return
	}
	if index >= len(pl.Items) {
		return
	}

	item := pl.Items[index]

	pl.ItemsLock.Lock()
	pl.Items = slices.Delete(pl.Items, index, index+1)
	pl.ItemsLock.Unlock()

	item.RemoveFromList()
	log.Println("Removed item", item.GetInfo().Title)

	pl.notifyChange(ItemsChange)
	pl.CriticalUpdate()
}

// InsertItem must be in the watcher routine
func (pl *PlayList) InsertItem(item *song.PreloadedSong, beforeIndex int) {
	if pl.stopped {
		return
	}

	if beforeIndex == -1 {
		pl.ItemsLock.Lock()
		pl.Items = append(pl.Items, item)
		pl.ItemsLock.Unlock()

		log.Println("Appended item", item.GetInfo().Title)
	} else if beforeIndex < len(pl.Items) {
		pl.ItemsLock.Lock()
		pl.Items = slices.Insert(pl.Items, beforeIndex, item)
		pl.ItemsLock.Unlock()

		log.Println("Inserted item", item.GetInfo().Title, "before", beforeIndex)
	} else {
		return
	}

	pl.notifyChange(ItemsChange)
	pl.CriticalUpdate()
}

// FromList must be in the watcher routine
func (pl *PlayList) FromList(items []*song.PreloadedSong) {
	if pl.stopped {
		return
	}
	if pl.Items == nil {
		return
	}

	pl.ItemsLock.Lock()
	pl.Items = items
	pl.ItemsLock.Unlock()

	pl.notifyChange(ItemsChange)
	pl.CriticalUpdate()
}

func createFromQueueItem(item types.QueueItem) *song.PreloadedSong {
	var newSong *song.PreloadedSong

	if temporaryItem != nil && temporaryItem.MatchWithQueueItem(&item) {
		newSong = temporaryItem
		temporaryItem = nil
	} else {
		if item.SongNum < 0 {
			// Custom Song
			newSong = song.CreatePreloadedCustomSong(item.VideoName, item.URL)
		} else if item.SongNum == 0 {
			newSong = song.CreateUnknownSong()
		} else {
			// PyPyDance Song
			newSong = song.CreatePreloadedPyPySong(item.SongNum)
		}
	}
	if newSong == nil {
		newSong = song.CreateUnknownSong()
	}

	newSong.Adder = item.PlayerName

	id := newSong.GetId()
	if id != "unknown" && id != "random_play" {
		persistence.GetLocalSongs().AddLocalSongIfNotExist(id, newSong.GetInfo().Title)
	}

	return newSong
}

// RemoveItem must be in the watcher routine
func RemoveItem(index int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.RemoveItem(index)
}

// InsertItem must be in the watcher routine
func InsertItem(item types.QueueItem, beforeIndex int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.InsertItem(createFromQueueItem(item), beforeIndex)
}

// ClearAndSetQueue must be in the watcher routine
func ClearAndSetQueue(items []types.QueueItem) {
	if currentPlaylist == nil {
		return
	}

	if len(currentPlaylist.Items) > 0 {
		lastRoomName := currentPlaylist.RoomName
		currentPlaylist.StopAll()

		currentPlaylist = newPlayList(currentPlaylist.maxPreload)
		currentPlaylist.RoomName = lastRoomName
		notifyNewList(currentPlaylist)

		log.Println("New playlist")
	}

	if len(items) > 0 {
		list := lo.Map(items, func(item types.QueueItem, _ int) *song.PreloadedSong {
			return createFromQueueItem(item)
		})
		currentPlaylist.FromList(list)
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
		currentPlaylist.StopAll()

		currentPlaylist = newPlayList(currentPlaylist.maxPreload)
		currentPlaylist.RoomName = roomName
		notifyNewList(currentPlaylist)
	} else {
		UpdateRoomName(roomName)
	}
}

// GetQueue must be in the watcher routine
func GetQueue() []*song.PreloadedSong {
	if currentPlaylist == nil {
		return nil
	}
	return currentPlaylist.Items
}
