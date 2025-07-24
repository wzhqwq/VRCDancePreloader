package playlist

import (
	"context"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"
	"slices"
	"time"
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
	log.Println("Removed item", item.GetId())

	if pl.bulk {
		pl.dirty = true
	} else {
		pl.notifyChange(ItemsChange)
		pl.CriticalUpdate()
	}
}

// PullOutItem is the same as RemoveItem, except that it won't stop the lifecycle of selected item
func (pl *PlayList) PullOutItem(index int) {
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

	log.Println("Pulled out item", item.GetId())

	if pl.bulk {
		pl.dirty = true
	} else {
		pl.notifyChange(ItemsChange)
		pl.CriticalUpdate()
	}
}

// InsertItem must be in the watcher routine
func (pl *PlayList) InsertItem(item *song.PreloadedSong, beforeIndex int) {
	if pl.stopped {
		return
	}

	if beforeIndex == -1 || beforeIndex >= len(pl.Items) {
		pl.ItemsLock.Lock()
		pl.Items = append(pl.Items, item)
		pl.ItemsLock.Unlock()

		log.Println("Appended item", item.GetId())
	} else {
		pl.ItemsLock.Lock()
		beforeItem := pl.Items[beforeIndex]
		pl.Items = slices.Insert(pl.Items, beforeIndex, item)
		pl.ItemsLock.Unlock()

		log.Println("Inserted item", item.GetId(), "before", beforeItem.GetId())
	}

	if pl.bulk {
		pl.dirty = true
	} else {
		pl.notifyChange(ItemsChange)
		pl.CriticalUpdate()
	}
}

func (pl *PlayList) BulkUpdate(ctx context.Context) {
	if pl.bulk {
		return
	}
	pl.bulk = true
	pl.dirty = false
	go func() {
		select {
		case <-ctx.Done():
			if !pl.stopped && pl.dirty {
				pl.notifyChange(ItemsChange)
				pl.CriticalUpdate()
			}
		case <-time.After(time.Second * 1):
			// never reach here
		}
		pl.bulk = false
	}()
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

func createFromQueueItem(item queue.QueueItem) *song.PreloadedSong {
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
func InsertItem(item queue.QueueItem, beforeIndex int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.InsertItem(createFromQueueItem(item), beforeIndex)
}

func PullOutItem(index int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.PullOutItem(index)
}

// BulkUpdate freeze auto-update until cancel function is called or 300ms timeout is reached
func BulkUpdate(ctx context.Context) context.CancelFunc {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Millisecond*300)
	if currentPlaylist == nil {
		return cancel
	}
	currentPlaylist.BulkUpdate(timeoutCtx)
	return cancel
}

// ClearAndSetQueue must be in the watcher routine
func ClearAndSetQueue(items []queue.QueueItem) {
	if currentPlaylist == nil {
		return
	}

	if len(currentPlaylist.Items) == 1 && len(items) == 1 {
		// better experience for random play
		currentPlaylist.RemoveItem(0)
		currentPlaylist.InsertItem(createFromQueueItem(items[0]), -1)
		return
	}

	if len(currentPlaylist.Items) > 0 {
		resetPlaylist(currentPlaylist.RoomName)
		log.Println("New playlist")
	}

	if len(items) > 0 {
		list := lo.Map(items, func(item queue.QueueItem, _ int) *song.PreloadedSong {
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
