package playlist

import (
	"errors"
	"io"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

var currentPlaylist []*PlayItem
var temporaryItem *PlayItem
var mutatingMutex = &sync.Mutex{}

var maxPreload int

func Init(maxPreload_ int) {
	err := loadSongs()
	if err != nil {
		log.Println("Failed to load songs:", err)
		panic(err)
	}

	maxPreload = maxPreload_
	currentPlaylist = make([]*PlayItem, 0)

	go keepCriticalUpdate()
}

func Stop() {
	for _, item := range currentPlaylist {
		item.Dispose()
	}
	currentPlaylist = nil
	<-time.After(1 * time.Second)
}

func GetSongsResponse() []byte {
	if savedResponse == nil {
		<-songLoaded
	}
	return savedResponse
}

func RestartPlaylist(items []*PlayItem) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()
	if currentPlaylist == nil {
		return
	}

	for _, item := range currentPlaylist {
		item.Dispose()
		gui.RemovePlayItem(item)
	}
	currentPlaylist = items
	for index, item := range currentPlaylist {
		item.UpdateIndex(index)
		gui.AddPlayItem(item)
	}
	CriticalUpdate()
	log.Println("Restarted playlist")
}

func RemoveItem(id int) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()
	if currentPlaylist == nil {
		return
	}

	removed := false
	for i, item := range currentPlaylist {
		if item.ID == id {
			currentPlaylist = slices.Delete(currentPlaylist, i, i+1)
			log.Println("Removed item", item.Title)
			removed = true
			item.Dispose()
			gui.RemovePlayItem(item)
		}
		if removed {
			item.UpdateIndex(item.Index - 1)
		}
	}

	if removed {
		CriticalUpdate()
	}
}

func InsertItem(item *PlayItem, beforeId int) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()
	if currentPlaylist == nil {
		return
	}

	if beforeId == -1 {
		item.Index = len(currentPlaylist)
		currentPlaylist = append(currentPlaylist, item)
		log.Println("Appended item", item.Title)

		gui.AddPlayItem(item)
		CriticalUpdate()
	}
	beforeIndex := -1
	for i, item := range currentPlaylist {
		if item.ID == beforeId {
			beforeIndex = i
			item.Index = i
		}
		if beforeIndex != -1 {
			item.UpdateIndex(item.Index + 1)
		}
	}

	if beforeIndex != -1 {
		currentPlaylist = slices.Insert(currentPlaylist, beforeIndex, item)
		log.Println("Inserted item", item.Title, "before", beforeId)

		gui.AddPlayItem(item)
		CriticalUpdate()
	}
}

func createFromSongList(id int) *PlayItem {
	if song, ok := songMap[id]; ok {
		return NewPlayItem(
			song.Name,
			songGroups[song.Group],
			i18n.T("placeholder_unknown_adder"),
			"",
			song.ID,
			song.End,
			-1,
		)
	}
	return nil
}
func CreateFromQueueItem(item types.QueueItem) *PlayItem {
	if temporaryItem != nil && temporaryItem.ID == item.SongNum {
		temporaryItem.Adder = item.PlayerName
		temporaryItem.URL = item.URL
		item := temporaryItem
		temporaryItem = nil
		return item
	}

	return NewPlayItem(item.VideoName,
		item.Group,
		item.PlayerName,
		item.URL,
		item.SongNum,
		item.Length,
		-1,
	)
}

func FindSongToPlay(id int) *PlayItem {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()

	for index, item := range currentPlaylist {
		if item.ID == id {
			item.UpdateStatus(constants.Playing)
			for i := 0; i < index; i++ {
				if currentPlaylist[i].Status == constants.Playing {
					currentPlaylist[i].UpdateStatus(constants.Ended)
				}
			}
			return item
		}
	}
	if item := createFromSongList(id); item != nil {
		if temporaryItem != nil {
			temporaryItem.Dispose()
		}
		temporaryItem = item
		return item
	}
	return nil
}

func PlaySong(id int) (io.ReadSeekCloser, error) {
	item := FindSongToPlay(id)
	if item == nil {
		return nil, errors.New("Song not found")
	}

	reader, err := item.ToReader()
	if err != nil {
		return nil, err
	}
	return reader, nil
}
