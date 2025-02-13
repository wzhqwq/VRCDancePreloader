package playlist

import (
	"log"
	"slices"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

func (pl *PlayList) RemoveItem(index int) {
	pl.Lock()
	defer pl.Unlock()

	if index >= len(pl.Items) {
		return
	}

	item := pl.Items[index]
	pl.Items = slices.Delete(pl.Items, index, index+1)

	item.RemoveFromList()
	log.Println("Removed item", item.GetInfo().Title)

	pl.notifyChange(ItemsChange)
	pl.CriticalUpdate()
}

func (pl *PlayList) InsertItem(item *song.PreloadedSong, beforeIndex int) {
	pl.Lock()
	defer pl.Unlock()

	if beforeIndex == -1 {
		pl.Items = append(pl.Items, item)
		log.Println("Appended item", item.GetInfo().Title)
	} else if beforeIndex < len(pl.Items) {
		pl.Items = slices.Insert(pl.Items, beforeIndex, item)
		log.Println("Inserted item", item.GetInfo().Title, "before", beforeIndex)
	} else {
		return
	}

	pl.notifyNewItem(item)
	pl.notifyChange(ItemsChange)
	pl.CriticalUpdate()
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
			newSong = song.CreateEmptySong()
		} else {
			// PyPyDance Song
			newSong = song.CreatePreloadedPyPySong(item.SongNum)
		}
	}

	newSong.Adder = item.PlayerName

	return newSong
}

func RemoveItem(index int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.RemoveItem(index)
}
func InsertItem(item types.QueueItem, beforeIndex int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.InsertItem(createFromQueueItem(item), beforeIndex)
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
	currentPlaylist.Start()
	log.Println("Restarted playlist")
}

func GetQueue() []*song.PreloadedSong {
	if currentPlaylist == nil {
		return nil
	}
	return currentPlaylist.Items
}
