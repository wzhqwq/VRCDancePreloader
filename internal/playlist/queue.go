package playlist

import (
	"log"
	"slices"

	"github.com/samber/lo"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

func (pl *PlayList) RemoveItem(queueData *types.QueueItem) {
	pl.Lock()
	defer pl.Unlock()

	removedIndex := -1
	for i, item := range pl.Items {
		if item.MatchWithQueueItem(queueData) {
			removedIndex = i
			pl.Items = slices.Delete(pl.Items, i, i+1)
			item.RemoveFromList()
			log.Println("Removed item", item.GetInfo().Title)
			break
		}
	}
	if removedIndex != -1 {
		pl.notifyChange(ItemsChange)
		pl.CriticalUpdate()
	}
}

func (pl *PlayList) InsertItem(item *song.PreloadedSong, before *types.QueueItem) {
	pl.Lock()
	defer pl.Unlock()

	if before == nil {
		pl.Items = append(pl.Items, item)
		pl.notifyNewItem(item)
		pl.notifyChange(ItemsChange)
		log.Println("Appended item", item.GetInfo().Title)

		pl.CriticalUpdate()
		return
	}
	insertIndex := -1
	for i, item := range pl.Items {
		if item.MatchWithQueueItem(before) {
			insertIndex = i
			pl.Items = slices.Insert(pl.Items, insertIndex, item)
			pl.notifyNewItem(item)
			log.Println("Inserted item", item.GetInfo().Title, "before", before.VideoName)
			break
		}
	}
	if insertIndex != -1 {
		pl.notifyChange(ItemsChange)
		pl.CriticalUpdate()
	}
}

func (pl *PlayList) FromList(items []*song.PreloadedSong) {
	pl.Lock()
	defer pl.Unlock()
	if pl.Items == nil {
		return
	}

	pl.Items = items
	for _, item := range items {
		pl.notifyNewItem(item)
	}
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
			newSong = song.CreateRandomPlaySong()
		} else {
			// PyPyDance Song
			newSong = song.CreatePreloadedPyPySong(item.SongNum)
		}
	}

	newSong.Adder = item.PlayerName

	return newSong
}

func RemoveItem(queueData *types.QueueItem) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.RemoveItem(queueData)
}
func InsertItem(item types.QueueItem, before *types.QueueItem) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.InsertItem(createFromQueueItem(item), before)
}
func ClearAndSetQueue(items []types.QueueItem) {
	maxPreload := 2
	if currentPlaylist != nil {
		currentPlaylist.StopAll()
		maxPreload = currentPlaylist.maxPreload
	}

	currentPlaylist = newPlayList(maxPreload)
	notifyNewList(currentPlaylist)
	list := lo.Map(items, func(item types.QueueItem, _ int) *song.PreloadedSong {
		return createFromQueueItem(item)
	})
	currentPlaylist.FromList(list)
	log.Println("Restarted playlist")
}
